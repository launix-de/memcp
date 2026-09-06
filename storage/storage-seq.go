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
	var d52 scm.JITValueDesc
	_ = d52
	var d53 scm.JITValueDesc
	_ = d53
	var d54 scm.JITValueDesc
	_ = d54
	var d109 scm.JITValueDesc
	_ = d109
	var d110 scm.JITValueDesc
	_ = d110
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
	var d140 scm.JITValueDesc
	_ = d140
	var d225 scm.JITValueDesc
	_ = d225
	var d226 scm.JITValueDesc
	_ = d226
	var d227 scm.JITValueDesc
	_ = d227
	var d228 scm.JITValueDesc
	_ = d228
	var d230 scm.JITValueDesc
	_ = d230
	var d231 scm.JITValueDesc
	_ = d231
	var d232 scm.JITValueDesc
	_ = d232
	var d233 scm.JITValueDesc
	_ = d233
	var d234 scm.JITValueDesc
	_ = d234
	var d235 scm.JITValueDesc
	_ = d235
	var d236 scm.JITValueDesc
	_ = d236
	var d237 scm.JITValueDesc
	_ = d237
	var d239 scm.JITValueDesc
	_ = d239
	var d241 scm.JITValueDesc
	_ = d241
	var d242 scm.JITValueDesc
	_ = d242
	var d243 scm.JITValueDesc
	_ = d243
	var d244 scm.JITValueDesc
	_ = d244
	var d245 scm.JITValueDesc
	_ = d245
	var d248 scm.JITValueDesc
	_ = d248
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
	var d356 scm.JITValueDesc
	_ = d356
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
	var d362 scm.JITValueDesc
	_ = d362
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
	var d524 scm.JITValueDesc
	_ = d524
	var d525 scm.JITValueDesc
	_ = d525
	var d526 scm.JITValueDesc
	_ = d526
	var d528 scm.JITValueDesc
	_ = d528
	var d529 scm.JITValueDesc
	_ = d529
	var d530 scm.JITValueDesc
	_ = d530
	var d531 scm.JITValueDesc
	_ = d531
	var d532 scm.JITValueDesc
	_ = d532
	var d533 scm.JITValueDesc
	_ = d533
	var d534 scm.JITValueDesc
	_ = d534
	var d536 scm.JITValueDesc
	_ = d536
	var d538 scm.JITValueDesc
	_ = d538
	var d539 scm.JITValueDesc
	_ = d539
	var d540 scm.JITValueDesc
	_ = d540
	var d541 scm.JITValueDesc
	_ = d541
	var d544 scm.JITValueDesc
	_ = d544
	var d696 scm.JITValueDesc
	_ = d696
	var d697 scm.JITValueDesc
	_ = d697
	var d698 scm.JITValueDesc
	_ = d698
	var d699 scm.JITValueDesc
	_ = d699
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
			r4 := ctx.AllocReg()
			ctx.EmitMovRegMem32(r4, fieldAddr)
			d12 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r4}
			ctx.BindReg(r4, &d12)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).seqCount))
			r5 := ctx.AllocReg()
			ctx.EmitMovRegMemL(r5, thisptr.Reg, off)
			d12 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r5}
			ctx.BindReg(r5, &d12)
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
			r6 := ctx.AllocReg()
			ctx.EmitMovRegReg(r6, d22.Reg)
			ctx.EmitShlRegImm8(r6, 32)
			ctx.EmitShrRegImm8(r6, 32)
			d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r6}
			ctx.BindReg(r6, &d23)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d24 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			r7 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r7, fieldAddr)
			d24 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r7}
			ctx.BindReg(r7, &d24)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			r8 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r8, thisptr.Reg, off)
			d24 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r8}
			ctx.BindReg(r8, &d24)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d24)
		ctx.EnsureDesc(&d24)
		var d25 scm.JITValueDesc
		if d24.Loc == scm.LocImm {
			d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d24.Imm.Int()))))}
		} else {
			r9 := ctx.AllocReg()
			ctx.EmitMovRegReg(r9, d24.Reg)
			ctx.EmitShlRegImm8(r9, 56)
			ctx.EmitShrRegImm8(r9, 56)
			d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
			ctx.BindReg(r9, &d25)
		}
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
			r10 := ctx.AllocRegExcept(d23.Reg, d25.Reg)
			ctx.EmitMovRegReg(r10, d23.Reg)
			ctx.EmitImulInt64(r10, d25.Reg)
			d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
			ctx.BindReg(r10, &d26)
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
			r11 := ctx.AllocRegExcept(d26.Reg)
			ctx.EmitMovRegReg(r11, d26.Reg)
			ctx.EmitShrRegImm8(r11, 6)
			d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
			ctx.BindReg(r11, &d27)
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
			r12 := ctx.AllocRegExcept(d26.Reg)
			ctx.EmitMovRegReg(r12, d26.Reg)
			ctx.EmitAndRegImm32(r12, 63)
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
			ctx.BindReg(r12, &d28)
		}
		if d28.Loc == scm.LocReg && d26.Loc == scm.LocReg && d28.Reg == d26.Reg {
			ctx.TransferReg(d26.Reg)
			d26.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d26)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d29 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
			r13 := ctx.AllocReg()
			r14 := ctx.AllocRegExcept(r13)
			r15 := ctx.AllocRegExcept(r13, r14)
			ctx.EmitMovRegMem64(r13, fieldAddr)
			ctx.EmitMovRegMem64(r14, fieldAddr+8)
			ctx.EmitMovRegMem64(r15, fieldAddr+16)
			d29 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r13, Reg2: r14, Reg3: r15}
			ctx.BindReg(r13, &d29)
			ctx.BindReg(r14, &d29)
			ctx.BindReg(r15, &d29)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
			r16 := ctx.AllocReg()
			r17 := ctx.AllocRegExcept(r16)
			r18 := ctx.AllocRegExcept(r16, r17)
			ctx.EmitMovRegMem(r16, thisptr.Reg, off)
			ctx.EmitMovRegMem(r17, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r18, thisptr.Reg, off+16)
			d29 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r16, Reg2: r17, Reg3: r18}
			ctx.BindReg(r16, &d29)
			ctx.BindReg(r17, &d29)
			ctx.BindReg(r18, &d29)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d27)
		ctx.ReclaimUntrackedRegs()
		d31 = ctx.EmitSliceElementAddress(&d29, &d27, 8)
		ctx.EnsureDesc(&d31)
		ctx.EmitMovRegMem(d31.Reg, d31.Reg, 0)
		d30 = d31
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d30)
		ctx.EnsureDesc(&d28)
		var d32 scm.JITValueDesc
		if d30.Loc == scm.LocImm && d28.Loc == scm.LocImm {
			d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d30.Imm.Int()) << uint64(d28.Imm.Int())))}
		} else if d28.Loc == scm.LocImm {
			r19 := ctx.AllocRegExcept(d30.Reg)
			ctx.EmitMovRegReg(r19, d30.Reg)
			ctx.EmitShlRegImm8(r19, uint8(d28.Imm.Int()))
			d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d32)
		} else {
			{
				shiftSrc := d30.Reg
				r20 := ctx.AllocRegExcept(d30.Reg)
				ctx.EmitMovRegReg(r20, d30.Reg)
				shiftSrc = r20
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
		ctx.FreeDesc(&d33)
		ctx.ReclaimUntrackedRegs()
		d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d28)
		ctx.EnsureDescsTogether(&d36, &d28)
		var d37 scm.JITValueDesc
		if d36.Loc == scm.LocImm && d28.Loc == scm.LocImm {
			d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d36.Imm.Int() - d28.Imm.Int())}
		} else if d28.Loc == scm.LocImm && d28.Imm.Int() == 0 {
			r21 := ctx.AllocRegExcept(d36.Reg)
			ctx.EmitMovRegReg(r21, d36.Reg)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			ctx.BindReg(r21, &d37)
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
			r22 := ctx.AllocRegExcept(d36.Reg, d28.Reg)
			ctx.EmitMovRegReg(r22, d36.Reg)
			ctx.EmitSubInt64(r22, d28.Reg)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
			ctx.BindReg(r22, &d37)
		}
		if d37.Loc == scm.LocReg && d36.Loc == scm.LocReg && d37.Reg == d36.Reg {
			ctx.TransferReg(d36.Reg)
			d36.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d28)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d34)
		ctx.EnsureDesc(&d37)
		var d38 scm.JITValueDesc
		if d34.Loc == scm.LocImm && d37.Loc == scm.LocImm {
			d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d34.Imm.Int()) >> uint64(d37.Imm.Int())))}
		} else if d37.Loc == scm.LocImm {
			r23 := ctx.AllocRegExcept(d34.Reg)
			ctx.EmitMovRegReg(r23, d34.Reg)
			ctx.EmitShrRegImm8(r23, uint8(d37.Imm.Int()))
			d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
			ctx.BindReg(r23, &d38)
		} else {
			{
				shiftSrc := d34.Reg
				r24 := ctx.AllocRegExcept(d34.Reg)
				ctx.EmitMovRegReg(r24, d34.Reg)
				shiftSrc = r24
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
			r25 := ctx.AllocRegExcept(d32.Reg)
			ctx.EmitMovRegReg(r25, d32.Reg)
			d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
			ctx.BindReg(r25, &d39)
		} else if d32.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d38.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d32.Imm.Int()))
			ctx.EmitOrInt64(scratch, d38.Reg)
			d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d39)
		} else if d38.Loc == scm.LocImm {
			r26 := ctx.AllocRegExcept(d32.Reg)
			ctx.EmitMovRegReg(r26, d32.Reg)
			if d38.Imm.Int() >= -2147483648 && d38.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r26, int32(d38.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d38.Imm.Int()))
				ctx.EmitOrInt64(r26, scm.RegR11)
			}
			d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
			ctx.BindReg(r26, &d39)
		} else {
			r27 := ctx.AllocRegExcept(d32.Reg, d38.Reg)
			ctx.EmitMovRegReg(r27, d32.Reg)
			ctx.EmitOrInt64(r27, d38.Reg)
			d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
			ctx.BindReg(r27, &d39)
		}
		if d39.Loc == scm.LocReg && d32.Loc == scm.LocReg && d39.Reg == d32.Reg {
			ctx.TransferReg(d32.Reg)
			d32.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d32)
		ctx.FreeDesc(&d38)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d24)
		ctx.EnsureDesc(&d24)
		var d40 scm.JITValueDesc
		if d24.Loc == scm.LocImm {
			d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d24.Imm.Int()))))}
		} else {
			r28 := ctx.AllocReg()
			ctx.EmitMovRegReg(r28, d24.Reg)
			ctx.EmitShlRegImm8(r28, 56)
			ctx.EmitShrRegImm8(r28, 56)
			d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
			ctx.BindReg(r28, &d40)
		}
		ctx.ReclaimUntrackedRegs()
		d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d40)
		ctx.EnsureDescsTogether(&d41, &d40)
		var d42 scm.JITValueDesc
		if d41.Loc == scm.LocImm && d40.Loc == scm.LocImm {
			d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d41.Imm.Int() - d40.Imm.Int())}
		} else if d40.Loc == scm.LocImm && d40.Imm.Int() == 0 {
			r29 := ctx.AllocRegExcept(d41.Reg)
			ctx.EmitMovRegReg(r29, d41.Reg)
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
			ctx.BindReg(r29, &d42)
		} else if d41.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d40.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d41.Imm.Int()))
			ctx.EmitSubInt64(scratch, d40.Reg)
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d42)
		} else if d40.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d41.Reg)
			ctx.EmitMovRegReg(scratch, d41.Reg)
			if d40.Imm.Int() >= -2147483648 && d40.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d40.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d42)
		} else {
			r30 := ctx.AllocRegExcept(d41.Reg, d40.Reg)
			ctx.EmitMovRegReg(r30, d41.Reg)
			ctx.EmitSubInt64(r30, d40.Reg)
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
			ctx.BindReg(r30, &d42)
		}
		if d42.Loc == scm.LocReg && d41.Loc == scm.LocReg && d42.Reg == d41.Reg {
			ctx.TransferReg(d41.Reg)
			d41.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d40)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d39)
		ctx.EnsureDesc(&d42)
		var d43 scm.JITValueDesc
		if d39.Loc == scm.LocImm && d42.Loc == scm.LocImm {
			d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d39.Imm.Int()) >> uint64(d42.Imm.Int())))}
		} else if d42.Loc == scm.LocImm {
			r31 := ctx.AllocRegExcept(d39.Reg)
			ctx.EmitMovRegReg(r31, d39.Reg)
			ctx.EmitShrRegImm8(r31, uint8(d42.Imm.Int()))
			d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
			ctx.BindReg(r31, &d43)
		} else {
			{
				shiftSrc := d39.Reg
				r32 := ctx.AllocRegExcept(d39.Reg)
				ctx.EmitMovRegReg(r32, d39.Reg)
				shiftSrc = r32
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d42.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d42.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d42.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d43)
			}
		}
		if d43.Loc == scm.LocReg && d39.Loc == scm.LocReg && d43.Reg == d39.Reg {
			ctx.TransferReg(d39.Reg)
			d39.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d39)
		ctx.FreeDesc(&d42)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d43)
		ctx.EnsureDesc(&d43)
		ctx.EnsureDesc(&d43)
		var d44 scm.JITValueDesc
		if d43.Loc == scm.LocImm {
			d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d43.Imm.Int()))))}
		} else {
			r33 := ctx.AllocReg()
			ctx.EmitMovRegReg(r33, d43.Reg)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
			ctx.BindReg(r33, &d44)
		}
		ctx.FreeDesc(&d43)
		var d45 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
			r34 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r34, fieldAddr)
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r34}
			ctx.BindReg(r34, &d45)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
			r35 := ctx.AllocReg()
			ctx.EmitMovRegMem(r35, thisptr.Reg, off)
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r35}
			ctx.BindReg(r35, &d45)
		}
		ctx.EnsureDesc(&d44)
		ctx.EnsureDesc(&d45)
		ctx.EnsureDescsTogether(&d44, &d45)
		var d46 scm.JITValueDesc
		if d44.Loc == scm.LocImm && d45.Loc == scm.LocImm {
			d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() + d45.Imm.Int())}
		} else if d45.Loc == scm.LocImm && d45.Imm.Int() == 0 {
			r36 := ctx.AllocRegExcept(d44.Reg)
			ctx.EmitMovRegReg(r36, d44.Reg)
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
			ctx.BindReg(r36, &d46)
		} else if d44.Loc == scm.LocImm && d44.Imm.Int() == 0 {
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d45.Reg}
			ctx.BindReg(d45.Reg, &d46)
		} else if d44.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d45.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d44.Imm.Int()))
			ctx.EmitAddInt64(scratch, d45.Reg)
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d46)
		} else if d45.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d44.Reg)
			ctx.EmitMovRegReg(scratch, d44.Reg)
			if d45.Imm.Int() >= -2147483648 && d45.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d45.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d45.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d46)
		} else {
			r37 := ctx.AllocRegExcept(d44.Reg, d45.Reg)
			ctx.EmitMovRegReg(r37, d44.Reg)
			ctx.EmitAddInt64(r37, d45.Reg)
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
			ctx.BindReg(r37, &d46)
		}
		if d46.Loc == scm.LocReg && d44.Loc == scm.LocReg && d46.Reg == d44.Reg {
			ctx.TransferReg(d44.Reg)
			d44.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d44)
		ctx.EnsureDesc(&d46)
		ctx.EnsureDesc(&d46)
		var d47 scm.JITValueDesc
		if d46.Loc == scm.LocImm {
			d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d46.Imm.Int()))))}
		} else {
			r38 := ctx.AllocReg()
			ctx.EmitMovRegReg(r38, d46.Reg)
			ctx.EmitShlRegImm8(r38, 32)
			ctx.EmitShrRegImm8(r38, 32)
			d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
			ctx.BindReg(r38, &d47)
		}
		ctx.FreeDesc(&d46)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d47)
		ctx.EnsureDescsTogether(&idxInt, &d47)
		var d48 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm && d47.Loc == scm.LocImm {
			d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d47.Imm.Int()))}
		} else if d47.Loc == scm.LocImm {
			r39 := ctx.AllocRegExcept(idxInt.Reg)
			if d47.Imm.Int() >= -2147483648 && d47.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(idxInt.Reg, int32(d47.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d47.Imm.Int()))
				ctx.EmitCmpInt64(idxInt.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r39, scm.CondUnsignedBelow)
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r39}
			ctx.BindReg(r39, &d48)
		} else if idxInt.Loc == scm.LocImm {
			r40 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d47.Reg)
			ctx.EmitSetcc(r40, scm.CondUnsignedBelow)
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r40}
			ctx.BindReg(r40, &d48)
		} else {
			r41 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitCmpInt64(idxInt.Reg, d47.Reg)
			ctx.EmitSetcc(r41, scm.CondUnsignedBelow)
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r41}
			ctx.BindReg(r41, &d48)
		}
		ctx.FreeDesc(&d47)
		d49 = d48
		ctx.EnsureDesc(&d49)
		if d49.Loc != scm.LocImm && d49.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d49.Loc == scm.LocImm {
			if d49.Imm.Bool() {
				if ps.General {
				}
				ps50 := scm.PhiState{General: ps.General}
				ps50.OverlayValues = make([]scm.JITValueDesc, 50)
				ps50.OverlayValues[1] = d1
				ps50.OverlayValues[2] = d2
				ps50.OverlayValues[3] = d3
				ps50.OverlayValues[4] = d4
				ps50.OverlayValues[5] = d5
				ps50.OverlayValues[6] = d6
				ps50.OverlayValues[7] = d7
				ps50.OverlayValues[8] = d8
				ps50.OverlayValues[9] = d9
				ps50.OverlayValues[10] = d10
				ps50.OverlayValues[11] = d11
				ps50.OverlayValues[12] = d12
				ps50.OverlayValues[13] = d13
				ps50.OverlayValues[14] = d14
				ps50.OverlayValues[15] = d15
				ps50.OverlayValues[17] = d17
				ps50.OverlayValues[18] = d18
				ps50.OverlayValues[19] = d19
				ps50.OverlayValues[20] = d20
				ps50.OverlayValues[21] = d21
				ps50.OverlayValues[22] = d22
				ps50.OverlayValues[23] = d23
				ps50.OverlayValues[24] = d24
				ps50.OverlayValues[25] = d25
				ps50.OverlayValues[26] = d26
				ps50.OverlayValues[27] = d27
				ps50.OverlayValues[28] = d28
				ps50.OverlayValues[29] = d29
				ps50.OverlayValues[30] = d30
				ps50.OverlayValues[31] = d31
				ps50.OverlayValues[32] = d32
				ps50.OverlayValues[33] = d33
				ps50.OverlayValues[34] = d34
				ps50.OverlayValues[35] = d35
				ps50.OverlayValues[36] = d36
				ps50.OverlayValues[37] = d37
				ps50.OverlayValues[38] = d38
				ps50.OverlayValues[39] = d39
				ps50.OverlayValues[40] = d40
				ps50.OverlayValues[41] = d41
				ps50.OverlayValues[42] = d42
				ps50.OverlayValues[43] = d43
				ps50.OverlayValues[44] = d44
				ps50.OverlayValues[45] = d45
				ps50.OverlayValues[46] = d46
				ps50.OverlayValues[47] = d47
				ps50.OverlayValues[48] = d48
				ps50.OverlayValues[49] = d49
				return bbs[3].RenderPS(ps50)
			}
			if ps.General {
			}
			ps51 := scm.PhiState{General: ps.General}
			ps51.OverlayValues = make([]scm.JITValueDesc, 50)
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
			return bbs[5].RenderPS(ps51)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d52 := ps.PhiValues[0]
				ctx.EnsureDesc(&d52)
				ctx.EmitStoreToStack(d52, int32(bbs[1].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d53 := ps.PhiValues[1]
				ctx.EnsureDesc(&d53)
				ctx.EmitStoreToStack(d53, int32(bbs[1].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d54 := ps.PhiValues[2]
				ctx.EnsureDesc(&d54)
				ctx.EmitStoreToStack(d54, int32(bbs[1].PhiBase)+int32(32))
			}
			ps.General = true
			return bbs[1].RenderPS(ps)
		}
		lbl16 := ctx.ReserveLabel()
		lbl17 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d49.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl16)
		ctx.EmitJmp(lbl17)
		ctx.MarkLabel(lbl16)
		ctx.EmitJmp(lbl4)
		ctx.MarkLabel(lbl17)
		ctx.EmitJmp(lbl6)
		ps55 := scm.PhiState{General: true}
		ps55.OverlayValues = make([]scm.JITValueDesc, 55)
		ps55.OverlayValues[1] = d1
		ps55.OverlayValues[2] = d2
		ps55.OverlayValues[3] = d3
		ps55.OverlayValues[4] = d4
		ps55.OverlayValues[5] = d5
		ps55.OverlayValues[6] = d6
		ps55.OverlayValues[7] = d7
		ps55.OverlayValues[8] = d8
		ps55.OverlayValues[9] = d9
		ps55.OverlayValues[10] = d10
		ps55.OverlayValues[11] = d11
		ps55.OverlayValues[12] = d12
		ps55.OverlayValues[13] = d13
		ps55.OverlayValues[14] = d14
		ps55.OverlayValues[15] = d15
		ps55.OverlayValues[17] = d17
		ps55.OverlayValues[18] = d18
		ps55.OverlayValues[19] = d19
		ps55.OverlayValues[20] = d20
		ps55.OverlayValues[21] = d21
		ps55.OverlayValues[22] = d22
		ps55.OverlayValues[23] = d23
		ps55.OverlayValues[24] = d24
		ps55.OverlayValues[25] = d25
		ps55.OverlayValues[26] = d26
		ps55.OverlayValues[27] = d27
		ps55.OverlayValues[28] = d28
		ps55.OverlayValues[29] = d29
		ps55.OverlayValues[30] = d30
		ps55.OverlayValues[31] = d31
		ps55.OverlayValues[32] = d32
		ps55.OverlayValues[33] = d33
		ps55.OverlayValues[34] = d34
		ps55.OverlayValues[35] = d35
		ps55.OverlayValues[36] = d36
		ps55.OverlayValues[37] = d37
		ps55.OverlayValues[38] = d38
		ps55.OverlayValues[39] = d39
		ps55.OverlayValues[40] = d40
		ps55.OverlayValues[41] = d41
		ps55.OverlayValues[42] = d42
		ps55.OverlayValues[43] = d43
		ps55.OverlayValues[44] = d44
		ps55.OverlayValues[45] = d45
		ps55.OverlayValues[46] = d46
		ps55.OverlayValues[47] = d47
		ps55.OverlayValues[48] = d48
		ps55.OverlayValues[49] = d49
		ps55.OverlayValues[52] = d52
		ps55.OverlayValues[53] = d53
		ps55.OverlayValues[54] = d54
		ps56 := scm.PhiState{General: true}
		ps56.OverlayValues = make([]scm.JITValueDesc, 55)
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
		ps56.OverlayValues[52] = d52
		ps56.OverlayValues[53] = d53
		ps56.OverlayValues[54] = d54
		snap57 := d1
		snap58 := d2
		snap59 := d3
		snap60 := d4
		snap61 := d5
		snap62 := d6
		snap63 := d7
		snap64 := d8
		snap65 := d9
		snap66 := d10
		snap67 := d11
		snap68 := d12
		snap69 := d13
		snap70 := d14
		snap71 := d15
		snap72 := d17
		snap73 := d18
		snap74 := d19
		snap75 := d20
		snap76 := d21
		snap77 := d22
		snap78 := d23
		snap79 := d24
		snap80 := d25
		snap81 := d26
		snap82 := d27
		snap83 := d28
		snap84 := d29
		snap85 := d30
		snap86 := d31
		snap87 := d32
		snap88 := d33
		snap89 := d34
		snap90 := d35
		snap91 := d36
		snap92 := d37
		snap93 := d38
		snap94 := d39
		snap95 := d40
		snap96 := d41
		snap97 := d42
		snap98 := d43
		snap99 := d44
		snap100 := d45
		snap101 := d46
		snap102 := d47
		snap103 := d48
		snap104 := d49
		snap105 := d52
		snap106 := d53
		snap107 := d54
		alloc108 := ctx.SnapshotAllocState()
		if !bbs[5].Rendered {
			bbs[5].RenderPS(ps56)
		}
		ctx.RestoreAllocState(alloc108)
		d1 = snap57
		d2 = snap58
		d3 = snap59
		d4 = snap60
		d5 = snap61
		d6 = snap62
		d7 = snap63
		d8 = snap64
		d9 = snap65
		d10 = snap66
		d11 = snap67
		d12 = snap68
		d13 = snap69
		d14 = snap70
		d15 = snap71
		d17 = snap72
		d18 = snap73
		d19 = snap74
		d20 = snap75
		d21 = snap76
		d22 = snap77
		d23 = snap78
		d24 = snap79
		d25 = snap80
		d26 = snap81
		d27 = snap82
		d28 = snap83
		d29 = snap84
		d30 = snap85
		d31 = snap86
		d32 = snap87
		d33 = snap88
		d34 = snap89
		d35 = snap90
		d36 = snap91
		d37 = snap92
		d38 = snap93
		d39 = snap94
		d40 = snap95
		d41 = snap96
		d42 = snap97
		d43 = snap98
		d44 = snap99
		d45 = snap100
		d46 = snap101
		d47 = snap102
		d48 = snap103
		d49 = snap104
		d52 = snap105
		d53 = snap106
		d54 = snap107
		if !bbs[3].Rendered {
			return bbs[3].RenderPS(ps55)
		}
		return result
		ctx.FreeDesc(&d48)
		return result
	}
	bbs[2].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d109 := ps.PhiValues[0]
				ctx.EnsureDesc(&d109)
				ctx.EmitStoreToStack(d109, int32(bbs[2].PhiBase)+int32(0))
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
			d52 = ps.OverlayValues[52]
		}
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != scm.LocNone {
			d109 = ps.OverlayValues[109]
		}
		if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
			d4 = ps.PhiValues[0]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d4)
		ctx.EnsureDesc(&d4)
		ctx.EnsureDesc(&d4)
		var d110 scm.JITValueDesc
		if d4.Loc == scm.LocImm {
			d110 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(d4.Imm.Int()))))}
		} else {
			r42 := ctx.AllocReg()
			ctx.EmitMovRegReg(r42, d4.Reg)
			ctx.EmitShlRegImm8(r42, 32)
			ctx.EmitShrRegImm8(r42, 32)
			d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
			ctx.BindReg(r42, &d110)
		}
		ctx.EnsureDesc(&d110)
		if thisptr.Loc == scm.LocImm {
			baseReg := ctx.AllocReg()
			if d110.Loc == scm.LocReg {
				ctx.FreeReg(baseReg)
				baseReg = ctx.AllocRegExcept(d110.Reg)
			}
			ctx.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int())+unsafe.Offsetof((*StorageSeq)(nil).lastValue)))
			if d110.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d110.Imm.Int()))
				ctx.EmitStoreRegMem(scm.RegR11, baseReg, 0)
			} else {
				ctx.EmitStoreRegMem(d110.Reg, baseReg, 0)
			}
			ctx.FreeReg(baseReg)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
			if d110.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d110.Imm.Int()))
				ctx.EmitStoreRegMem(scm.RegR11, thisptr.Reg, off)
			} else {
				ctx.EmitStoreRegMem(d110.Reg, thisptr.Reg, off)
			}
		}
		ctx.FreeDesc(&d110)
		ctx.EnsureDesc(&d4)
		d111 = d4
		_ = d111
		ctx.StabilizeDescForControlFlow(&d111)
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
		ctx.EnsureDesc(&d111)
		ctx.EnsureDesc(&d111)
		var d112 scm.JITValueDesc
		if d111.Loc == scm.LocImm {
			d112 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d111.Imm.Int()))))}
		} else {
			r43 := ctx.AllocReg()
			ctx.EmitMovRegReg(r43, d111.Reg)
			ctx.EmitShlRegImm8(r43, 32)
			ctx.EmitShrRegImm8(r43, 32)
			d112 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
			ctx.BindReg(r43, &d112)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d113 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
			r44 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r44, fieldAddr)
			d113 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r44}
			ctx.BindReg(r44, &d113)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
			r45 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r45, thisptr.Reg, off)
			d113 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r45}
			ctx.BindReg(r45, &d113)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d113)
		ctx.EnsureDesc(&d113)
		var d114 scm.JITValueDesc
		if d113.Loc == scm.LocImm {
			d114 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d113.Imm.Int()))))}
		} else {
			r46 := ctx.AllocReg()
			ctx.EmitMovRegReg(r46, d113.Reg)
			ctx.EmitShlRegImm8(r46, 56)
			ctx.EmitShrRegImm8(r46, 56)
			d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
			ctx.BindReg(r46, &d114)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d112)
		ctx.EnsureDesc(&d114)
		ctx.EnsureDescsTogether(&d112, &d114)
		var d115 scm.JITValueDesc
		if d112.Loc == scm.LocImm && d114.Loc == scm.LocImm {
			d115 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d112.Imm.Int() * d114.Imm.Int())}
		} else if d112.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d114.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d112.Imm.Int()))
			ctx.EmitImulInt64(scratch, d114.Reg)
			d115 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d115)
		} else if d114.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d112.Reg)
			ctx.EmitMovRegReg(scratch, d112.Reg)
			if d114.Imm.Int() >= -2147483648 && d114.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d114.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d114.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d115 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d115)
		} else {
			r47 := ctx.AllocRegExcept(d112.Reg, d114.Reg)
			ctx.EmitMovRegReg(r47, d112.Reg)
			ctx.EmitImulInt64(r47, d114.Reg)
			d115 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
			ctx.BindReg(r47, &d115)
		}
		if d115.Loc == scm.LocReg && d112.Loc == scm.LocReg && d115.Reg == d112.Reg {
			ctx.TransferReg(d112.Reg)
			d112.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d112)
		ctx.FreeDesc(&d114)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d115)
		var d116 scm.JITValueDesc
		if d115.Loc == scm.LocImm {
			d116 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d115.Imm.Int() / 64)}
		} else {
			r48 := ctx.AllocRegExcept(d115.Reg)
			ctx.EmitMovRegReg(r48, d115.Reg)
			ctx.EmitShrRegImm8(r48, 6)
			d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
			ctx.BindReg(r48, &d116)
		}
		if d116.Loc == scm.LocReg && d115.Loc == scm.LocReg && d116.Reg == d115.Reg {
			ctx.TransferReg(d115.Reg)
			d115.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d115)
		var d117 scm.JITValueDesc
		if d115.Loc == scm.LocImm {
			d117 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d115.Imm.Int() % 64)}
		} else {
			r49 := ctx.AllocRegExcept(d115.Reg)
			ctx.EmitMovRegReg(r49, d115.Reg)
			ctx.EmitAndRegImm32(r49, 63)
			d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
			ctx.BindReg(r49, &d117)
		}
		if d117.Loc == scm.LocReg && d115.Loc == scm.LocReg && d117.Reg == d115.Reg {
			ctx.TransferReg(d115.Reg)
			d115.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d115)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d118 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 0
			r50 := ctx.AllocReg()
			r51 := ctx.AllocRegExcept(r50)
			r52 := ctx.AllocRegExcept(r50, r51)
			ctx.EmitMovRegMem64(r50, fieldAddr)
			ctx.EmitMovRegMem64(r51, fieldAddr+8)
			ctx.EmitMovRegMem64(r52, fieldAddr+16)
			d118 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r50, Reg2: r51, Reg3: r52}
			ctx.BindReg(r50, &d118)
			ctx.BindReg(r51, &d118)
			ctx.BindReg(r52, &d118)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 0)
			r53 := ctx.AllocReg()
			r54 := ctx.AllocRegExcept(r53)
			r55 := ctx.AllocRegExcept(r53, r54)
			ctx.EmitMovRegMem(r53, thisptr.Reg, off)
			ctx.EmitMovRegMem(r54, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r55, thisptr.Reg, off+16)
			d118 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r53, Reg2: r54, Reg3: r55}
			ctx.BindReg(r53, &d118)
			ctx.BindReg(r54, &d118)
			ctx.BindReg(r55, &d118)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d116)
		ctx.ReclaimUntrackedRegs()
		d120 = ctx.EmitSliceElementAddress(&d118, &d116, 8)
		ctx.EnsureDesc(&d120)
		ctx.EmitMovRegMem(d120.Reg, d120.Reg, 0)
		d119 = d120
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d119)
		ctx.EnsureDesc(&d117)
		var d121 scm.JITValueDesc
		if d119.Loc == scm.LocImm && d117.Loc == scm.LocImm {
			d121 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d119.Imm.Int()) << uint64(d117.Imm.Int())))}
		} else if d117.Loc == scm.LocImm {
			r56 := ctx.AllocRegExcept(d119.Reg)
			ctx.EmitMovRegReg(r56, d119.Reg)
			ctx.EmitShlRegImm8(r56, uint8(d117.Imm.Int()))
			d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r56}
			ctx.BindReg(r56, &d121)
		} else {
			{
				shiftSrc := d119.Reg
				r57 := ctx.AllocRegExcept(d119.Reg)
				ctx.EmitMovRegReg(r57, d119.Reg)
				shiftSrc = r57
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d117.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d117.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d117.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d121)
			}
		}
		if d121.Loc == scm.LocReg && d119.Loc == scm.LocReg && d121.Reg == d119.Reg {
			ctx.TransferReg(d119.Reg)
			d119.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d119)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d116)
		ctx.EnsureDesc(&d116)
		var d122 scm.JITValueDesc
		if d116.Loc == scm.LocImm {
			d122 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d116.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d116.Reg)
			ctx.EmitMovRegReg(scratch, d116.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d122)
		}
		if d122.Loc == scm.LocReg && d116.Loc == scm.LocReg && d122.Reg == d116.Reg {
			ctx.TransferReg(d116.Reg)
			d116.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d116)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d122)
		ctx.ReclaimUntrackedRegs()
		d124 = ctx.EmitSliceElementAddress(&d118, &d122, 8)
		ctx.EnsureDesc(&d124)
		ctx.EmitMovRegMem(d124.Reg, d124.Reg, 0)
		d123 = d124
		ctx.FreeDesc(&d122)
		ctx.ReclaimUntrackedRegs()
		d125 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d117)
		ctx.EnsureDescsTogether(&d125, &d117)
		var d126 scm.JITValueDesc
		if d125.Loc == scm.LocImm && d117.Loc == scm.LocImm {
			d126 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d125.Imm.Int() - d117.Imm.Int())}
		} else if d117.Loc == scm.LocImm && d117.Imm.Int() == 0 {
			r58 := ctx.AllocRegExcept(d125.Reg)
			ctx.EmitMovRegReg(r58, d125.Reg)
			d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r58}
			ctx.BindReg(r58, &d126)
		} else if d125.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d117.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d125.Imm.Int()))
			ctx.EmitSubInt64(scratch, d117.Reg)
			d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d126)
		} else if d117.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d125.Reg)
			ctx.EmitMovRegReg(scratch, d125.Reg)
			if d117.Imm.Int() >= -2147483648 && d117.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d117.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d117.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d126)
		} else {
			r59 := ctx.AllocRegExcept(d125.Reg, d117.Reg)
			ctx.EmitMovRegReg(r59, d125.Reg)
			ctx.EmitSubInt64(r59, d117.Reg)
			d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r59}
			ctx.BindReg(r59, &d126)
		}
		if d126.Loc == scm.LocReg && d125.Loc == scm.LocReg && d126.Reg == d125.Reg {
			ctx.TransferReg(d125.Reg)
			d125.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d117)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d123)
		ctx.EnsureDesc(&d126)
		var d127 scm.JITValueDesc
		if d123.Loc == scm.LocImm && d126.Loc == scm.LocImm {
			d127 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d123.Imm.Int()) >> uint64(d126.Imm.Int())))}
		} else if d126.Loc == scm.LocImm {
			r60 := ctx.AllocRegExcept(d123.Reg)
			ctx.EmitMovRegReg(r60, d123.Reg)
			ctx.EmitShrRegImm8(r60, uint8(d126.Imm.Int()))
			d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r60}
			ctx.BindReg(r60, &d127)
		} else {
			{
				shiftSrc := d123.Reg
				r61 := ctx.AllocRegExcept(d123.Reg)
				ctx.EmitMovRegReg(r61, d123.Reg)
				shiftSrc = r61
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d126.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d126.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d126.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d127)
			}
		}
		if d127.Loc == scm.LocReg && d123.Loc == scm.LocReg && d127.Reg == d123.Reg {
			ctx.TransferReg(d123.Reg)
			d123.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d123)
		ctx.FreeDesc(&d126)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d121)
		ctx.EnsureDesc(&d127)
		var d128 scm.JITValueDesc
		if d121.Loc == scm.LocImm && d127.Loc == scm.LocImm {
			d128 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d121.Imm.Int() | d127.Imm.Int())}
		} else if d121.Loc == scm.LocImm && d121.Imm.Int() == 0 {
			d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d127.Reg}
			ctx.BindReg(d127.Reg, &d128)
		} else if d127.Loc == scm.LocImm && d127.Imm.Int() == 0 {
			r62 := ctx.AllocRegExcept(d121.Reg)
			ctx.EmitMovRegReg(r62, d121.Reg)
			d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r62}
			ctx.BindReg(r62, &d128)
		} else if d121.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d127.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d121.Imm.Int()))
			ctx.EmitOrInt64(scratch, d127.Reg)
			d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d128)
		} else if d127.Loc == scm.LocImm {
			r63 := ctx.AllocRegExcept(d121.Reg)
			ctx.EmitMovRegReg(r63, d121.Reg)
			if d127.Imm.Int() >= -2147483648 && d127.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r63, int32(d127.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d127.Imm.Int()))
				ctx.EmitOrInt64(r63, scm.RegR11)
			}
			d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r63}
			ctx.BindReg(r63, &d128)
		} else {
			r64 := ctx.AllocRegExcept(d121.Reg, d127.Reg)
			ctx.EmitMovRegReg(r64, d121.Reg)
			ctx.EmitOrInt64(r64, d127.Reg)
			d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r64}
			ctx.BindReg(r64, &d128)
		}
		if d128.Loc == scm.LocReg && d121.Loc == scm.LocReg && d128.Reg == d121.Reg {
			ctx.TransferReg(d121.Reg)
			d121.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d121)
		ctx.FreeDesc(&d127)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d113)
		ctx.EnsureDesc(&d113)
		var d129 scm.JITValueDesc
		if d113.Loc == scm.LocImm {
			d129 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d113.Imm.Int()))))}
		} else {
			r65 := ctx.AllocReg()
			ctx.EmitMovRegReg(r65, d113.Reg)
			ctx.EmitShlRegImm8(r65, 56)
			ctx.EmitShrRegImm8(r65, 56)
			d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
			ctx.BindReg(r65, &d129)
		}
		ctx.ReclaimUntrackedRegs()
		d130 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d129)
		ctx.EnsureDescsTogether(&d130, &d129)
		var d131 scm.JITValueDesc
		if d130.Loc == scm.LocImm && d129.Loc == scm.LocImm {
			d131 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d130.Imm.Int() - d129.Imm.Int())}
		} else if d129.Loc == scm.LocImm && d129.Imm.Int() == 0 {
			r66 := ctx.AllocRegExcept(d130.Reg)
			ctx.EmitMovRegReg(r66, d130.Reg)
			d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
			ctx.BindReg(r66, &d131)
		} else if d130.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d129.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d130.Imm.Int()))
			ctx.EmitSubInt64(scratch, d129.Reg)
			d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d131)
		} else if d129.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d130.Reg)
			ctx.EmitMovRegReg(scratch, d130.Reg)
			if d129.Imm.Int() >= -2147483648 && d129.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d129.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d129.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d131)
		} else {
			r67 := ctx.AllocRegExcept(d130.Reg, d129.Reg)
			ctx.EmitMovRegReg(r67, d130.Reg)
			ctx.EmitSubInt64(r67, d129.Reg)
			d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r67}
			ctx.BindReg(r67, &d131)
		}
		if d131.Loc == scm.LocReg && d130.Loc == scm.LocReg && d131.Reg == d130.Reg {
			ctx.TransferReg(d130.Reg)
			d130.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d129)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d128)
		ctx.EnsureDesc(&d131)
		var d132 scm.JITValueDesc
		if d128.Loc == scm.LocImm && d131.Loc == scm.LocImm {
			d132 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d128.Imm.Int()) >> uint64(d131.Imm.Int())))}
		} else if d131.Loc == scm.LocImm {
			r68 := ctx.AllocRegExcept(d128.Reg)
			ctx.EmitMovRegReg(r68, d128.Reg)
			ctx.EmitShrRegImm8(r68, uint8(d131.Imm.Int()))
			d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r68}
			ctx.BindReg(r68, &d132)
		} else {
			{
				shiftSrc := d128.Reg
				r69 := ctx.AllocRegExcept(d128.Reg)
				ctx.EmitMovRegReg(r69, d128.Reg)
				shiftSrc = r69
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d131.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d131.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d131.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d132)
			}
		}
		if d132.Loc == scm.LocReg && d128.Loc == scm.LocReg && d132.Reg == d128.Reg {
			ctx.TransferReg(d128.Reg)
			d128.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d128)
		ctx.FreeDesc(&d131)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d132)
		ctx.EnsureDesc(&d132)
		ctx.EnsureDesc(&d132)
		var d133 scm.JITValueDesc
		if d132.Loc == scm.LocImm {
			d133 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d132.Imm.Int()))))}
		} else {
			r70 := ctx.AllocReg()
			ctx.EmitMovRegReg(r70, d132.Reg)
			d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r70}
			ctx.BindReg(r70, &d133)
		}
		ctx.FreeDesc(&d132)
		var d134 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 32
			r71 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r71, fieldAddr)
			d134 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r71}
			ctx.BindReg(r71, &d134)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 32)
			r72 := ctx.AllocReg()
			ctx.EmitMovRegMem(r72, thisptr.Reg, off)
			d134 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r72}
			ctx.BindReg(r72, &d134)
		}
		ctx.EnsureDesc(&d133)
		ctx.EnsureDesc(&d134)
		ctx.EnsureDescsTogether(&d133, &d134)
		var d135 scm.JITValueDesc
		if d133.Loc == scm.LocImm && d134.Loc == scm.LocImm {
			d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d133.Imm.Int() + d134.Imm.Int())}
		} else if d134.Loc == scm.LocImm && d134.Imm.Int() == 0 {
			r73 := ctx.AllocRegExcept(d133.Reg)
			ctx.EmitMovRegReg(r73, d133.Reg)
			d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
			ctx.BindReg(r73, &d135)
		} else if d133.Loc == scm.LocImm && d133.Imm.Int() == 0 {
			d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d134.Reg}
			ctx.BindReg(d134.Reg, &d135)
		} else if d133.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d134.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d133.Imm.Int()))
			ctx.EmitAddInt64(scratch, d134.Reg)
			d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d135)
		} else if d134.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d133.Reg)
			ctx.EmitMovRegReg(scratch, d133.Reg)
			if d134.Imm.Int() >= -2147483648 && d134.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d134.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d134.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d135)
		} else {
			r74 := ctx.AllocRegExcept(d133.Reg, d134.Reg)
			ctx.EmitMovRegReg(r74, d133.Reg)
			ctx.EmitAddInt64(r74, d134.Reg)
			d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r74}
			ctx.BindReg(r74, &d135)
		}
		if d135.Loc == scm.LocReg && d133.Loc == scm.LocReg && d135.Reg == d133.Reg {
			ctx.TransferReg(d133.Reg)
			d133.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d135)
		ctx.FreeDesc(&d133)
		var d136 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 56
			r75 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r75, fieldAddr)
			d136 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r75}
			ctx.BindReg(r75, &d136)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 56)
			r76 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r76, thisptr.Reg, off)
			d136 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r76}
			ctx.BindReg(r76, &d136)
		}
		d137 = d136
		ctx.EnsureDesc(&d137)
		if d137.Loc != scm.LocImm && d137.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d137.Loc == scm.LocImm {
			if d137.Imm.Bool() {
				if ps.General {
				}
				ps138 := scm.PhiState{General: ps.General}
				ps138.OverlayValues = make([]scm.JITValueDesc, 138)
				ps138.OverlayValues[1] = d1
				ps138.OverlayValues[2] = d2
				ps138.OverlayValues[3] = d3
				ps138.OverlayValues[4] = d4
				ps138.OverlayValues[5] = d5
				ps138.OverlayValues[6] = d6
				ps138.OverlayValues[7] = d7
				ps138.OverlayValues[8] = d8
				ps138.OverlayValues[9] = d9
				ps138.OverlayValues[10] = d10
				ps138.OverlayValues[11] = d11
				ps138.OverlayValues[12] = d12
				ps138.OverlayValues[13] = d13
				ps138.OverlayValues[14] = d14
				ps138.OverlayValues[15] = d15
				ps138.OverlayValues[17] = d17
				ps138.OverlayValues[18] = d18
				ps138.OverlayValues[19] = d19
				ps138.OverlayValues[20] = d20
				ps138.OverlayValues[21] = d21
				ps138.OverlayValues[22] = d22
				ps138.OverlayValues[23] = d23
				ps138.OverlayValues[24] = d24
				ps138.OverlayValues[25] = d25
				ps138.OverlayValues[26] = d26
				ps138.OverlayValues[27] = d27
				ps138.OverlayValues[28] = d28
				ps138.OverlayValues[29] = d29
				ps138.OverlayValues[30] = d30
				ps138.OverlayValues[31] = d31
				ps138.OverlayValues[32] = d32
				ps138.OverlayValues[33] = d33
				ps138.OverlayValues[34] = d34
				ps138.OverlayValues[35] = d35
				ps138.OverlayValues[36] = d36
				ps138.OverlayValues[37] = d37
				ps138.OverlayValues[38] = d38
				ps138.OverlayValues[39] = d39
				ps138.OverlayValues[40] = d40
				ps138.OverlayValues[41] = d41
				ps138.OverlayValues[42] = d42
				ps138.OverlayValues[43] = d43
				ps138.OverlayValues[44] = d44
				ps138.OverlayValues[45] = d45
				ps138.OverlayValues[46] = d46
				ps138.OverlayValues[47] = d47
				ps138.OverlayValues[48] = d48
				ps138.OverlayValues[49] = d49
				ps138.OverlayValues[52] = d52
				ps138.OverlayValues[53] = d53
				ps138.OverlayValues[54] = d54
				ps138.OverlayValues[109] = d109
				ps138.OverlayValues[110] = d110
				ps138.OverlayValues[111] = d111
				ps138.OverlayValues[112] = d112
				ps138.OverlayValues[113] = d113
				ps138.OverlayValues[114] = d114
				ps138.OverlayValues[115] = d115
				ps138.OverlayValues[116] = d116
				ps138.OverlayValues[117] = d117
				ps138.OverlayValues[118] = d118
				ps138.OverlayValues[119] = d119
				ps138.OverlayValues[120] = d120
				ps138.OverlayValues[121] = d121
				ps138.OverlayValues[122] = d122
				ps138.OverlayValues[123] = d123
				ps138.OverlayValues[124] = d124
				ps138.OverlayValues[125] = d125
				ps138.OverlayValues[126] = d126
				ps138.OverlayValues[127] = d127
				ps138.OverlayValues[128] = d128
				ps138.OverlayValues[129] = d129
				ps138.OverlayValues[130] = d130
				ps138.OverlayValues[131] = d131
				ps138.OverlayValues[132] = d132
				ps138.OverlayValues[133] = d133
				ps138.OverlayValues[134] = d134
				ps138.OverlayValues[135] = d135
				ps138.OverlayValues[136] = d136
				ps138.OverlayValues[137] = d137
				return bbs[13].RenderPS(ps138)
			}
			if ps.General {
			}
			ps139 := scm.PhiState{General: ps.General}
			ps139.OverlayValues = make([]scm.JITValueDesc, 138)
			ps139.OverlayValues[1] = d1
			ps139.OverlayValues[2] = d2
			ps139.OverlayValues[3] = d3
			ps139.OverlayValues[4] = d4
			ps139.OverlayValues[5] = d5
			ps139.OverlayValues[6] = d6
			ps139.OverlayValues[7] = d7
			ps139.OverlayValues[8] = d8
			ps139.OverlayValues[9] = d9
			ps139.OverlayValues[10] = d10
			ps139.OverlayValues[11] = d11
			ps139.OverlayValues[12] = d12
			ps139.OverlayValues[13] = d13
			ps139.OverlayValues[14] = d14
			ps139.OverlayValues[15] = d15
			ps139.OverlayValues[17] = d17
			ps139.OverlayValues[18] = d18
			ps139.OverlayValues[19] = d19
			ps139.OverlayValues[20] = d20
			ps139.OverlayValues[21] = d21
			ps139.OverlayValues[22] = d22
			ps139.OverlayValues[23] = d23
			ps139.OverlayValues[24] = d24
			ps139.OverlayValues[25] = d25
			ps139.OverlayValues[26] = d26
			ps139.OverlayValues[27] = d27
			ps139.OverlayValues[28] = d28
			ps139.OverlayValues[29] = d29
			ps139.OverlayValues[30] = d30
			ps139.OverlayValues[31] = d31
			ps139.OverlayValues[32] = d32
			ps139.OverlayValues[33] = d33
			ps139.OverlayValues[34] = d34
			ps139.OverlayValues[35] = d35
			ps139.OverlayValues[36] = d36
			ps139.OverlayValues[37] = d37
			ps139.OverlayValues[38] = d38
			ps139.OverlayValues[39] = d39
			ps139.OverlayValues[40] = d40
			ps139.OverlayValues[41] = d41
			ps139.OverlayValues[42] = d42
			ps139.OverlayValues[43] = d43
			ps139.OverlayValues[44] = d44
			ps139.OverlayValues[45] = d45
			ps139.OverlayValues[46] = d46
			ps139.OverlayValues[47] = d47
			ps139.OverlayValues[48] = d48
			ps139.OverlayValues[49] = d49
			ps139.OverlayValues[52] = d52
			ps139.OverlayValues[53] = d53
			ps139.OverlayValues[54] = d54
			ps139.OverlayValues[109] = d109
			ps139.OverlayValues[110] = d110
			ps139.OverlayValues[111] = d111
			ps139.OverlayValues[112] = d112
			ps139.OverlayValues[113] = d113
			ps139.OverlayValues[114] = d114
			ps139.OverlayValues[115] = d115
			ps139.OverlayValues[116] = d116
			ps139.OverlayValues[117] = d117
			ps139.OverlayValues[118] = d118
			ps139.OverlayValues[119] = d119
			ps139.OverlayValues[120] = d120
			ps139.OverlayValues[121] = d121
			ps139.OverlayValues[122] = d122
			ps139.OverlayValues[123] = d123
			ps139.OverlayValues[124] = d124
			ps139.OverlayValues[125] = d125
			ps139.OverlayValues[126] = d126
			ps139.OverlayValues[127] = d127
			ps139.OverlayValues[128] = d128
			ps139.OverlayValues[129] = d129
			ps139.OverlayValues[130] = d130
			ps139.OverlayValues[131] = d131
			ps139.OverlayValues[132] = d132
			ps139.OverlayValues[133] = d133
			ps139.OverlayValues[134] = d134
			ps139.OverlayValues[135] = d135
			ps139.OverlayValues[136] = d136
			ps139.OverlayValues[137] = d137
			return bbs[12].RenderPS(ps139)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d140 := ps.PhiValues[0]
				ctx.EnsureDesc(&d140)
				ctx.EmitStoreToStack(d140, int32(bbs[2].PhiBase)+int32(0))
			}
			ps.General = true
			return bbs[2].RenderPS(ps)
		}
		lbl19 := ctx.ReserveLabel()
		lbl20 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d137.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl19)
		ctx.EmitJmp(lbl20)
		ctx.MarkLabel(lbl19)
		ctx.EmitJmp(lbl14)
		ctx.MarkLabel(lbl20)
		ctx.EmitJmp(lbl13)
		ps141 := scm.PhiState{General: true}
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
		ps141.OverlayValues[52] = d52
		ps141.OverlayValues[53] = d53
		ps141.OverlayValues[54] = d54
		ps141.OverlayValues[109] = d109
		ps141.OverlayValues[110] = d110
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
		ps141.OverlayValues[140] = d140
		ps142 := scm.PhiState{General: true}
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
		ps142.OverlayValues[52] = d52
		ps142.OverlayValues[53] = d53
		ps142.OverlayValues[54] = d54
		ps142.OverlayValues[109] = d109
		ps142.OverlayValues[110] = d110
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
		ps142.OverlayValues[140] = d140
		snap143 := d1
		snap144 := d2
		snap145 := d3
		snap146 := d4
		snap147 := d5
		snap148 := d6
		snap149 := d7
		snap150 := d8
		snap151 := d9
		snap152 := d10
		snap153 := d11
		snap154 := d12
		snap155 := d13
		snap156 := d14
		snap157 := d15
		snap158 := d17
		snap159 := d18
		snap160 := d19
		snap161 := d20
		snap162 := d21
		snap163 := d22
		snap164 := d23
		snap165 := d24
		snap166 := d25
		snap167 := d26
		snap168 := d27
		snap169 := d28
		snap170 := d29
		snap171 := d30
		snap172 := d31
		snap173 := d32
		snap174 := d33
		snap175 := d34
		snap176 := d35
		snap177 := d36
		snap178 := d37
		snap179 := d38
		snap180 := d39
		snap181 := d40
		snap182 := d41
		snap183 := d42
		snap184 := d43
		snap185 := d44
		snap186 := d45
		snap187 := d46
		snap188 := d47
		snap189 := d48
		snap190 := d49
		snap191 := d52
		snap192 := d53
		snap193 := d54
		snap194 := d109
		snap195 := d110
		snap196 := d111
		snap197 := d112
		snap198 := d113
		snap199 := d114
		snap200 := d115
		snap201 := d116
		snap202 := d117
		snap203 := d118
		snap204 := d119
		snap205 := d120
		snap206 := d121
		snap207 := d122
		snap208 := d123
		snap209 := d124
		snap210 := d125
		snap211 := d126
		snap212 := d127
		snap213 := d128
		snap214 := d129
		snap215 := d130
		snap216 := d131
		snap217 := d132
		snap218 := d133
		snap219 := d134
		snap220 := d135
		snap221 := d136
		snap222 := d137
		snap223 := d140
		alloc224 := ctx.SnapshotAllocState()
		if !bbs[12].Rendered {
			bbs[12].RenderPS(ps142)
		}
		ctx.RestoreAllocState(alloc224)
		d1 = snap143
		d2 = snap144
		d3 = snap145
		d4 = snap146
		d5 = snap147
		d6 = snap148
		d7 = snap149
		d8 = snap150
		d9 = snap151
		d10 = snap152
		d11 = snap153
		d12 = snap154
		d13 = snap155
		d14 = snap156
		d15 = snap157
		d17 = snap158
		d18 = snap159
		d19 = snap160
		d20 = snap161
		d21 = snap162
		d22 = snap163
		d23 = snap164
		d24 = snap165
		d25 = snap166
		d26 = snap167
		d27 = snap168
		d28 = snap169
		d29 = snap170
		d30 = snap171
		d31 = snap172
		d32 = snap173
		d33 = snap174
		d34 = snap175
		d35 = snap176
		d36 = snap177
		d37 = snap178
		d38 = snap179
		d39 = snap180
		d40 = snap181
		d41 = snap182
		d42 = snap183
		d43 = snap184
		d44 = snap185
		d45 = snap186
		d46 = snap187
		d47 = snap188
		d48 = snap189
		d49 = snap190
		d52 = snap191
		d53 = snap192
		d54 = snap193
		d109 = snap194
		d110 = snap195
		d111 = snap196
		d112 = snap197
		d113 = snap198
		d114 = snap199
		d115 = snap200
		d116 = snap201
		d117 = snap202
		d118 = snap203
		d119 = snap204
		d120 = snap205
		d121 = snap206
		d122 = snap207
		d123 = snap208
		d124 = snap209
		d125 = snap210
		d126 = snap211
		d127 = snap212
		d128 = snap213
		d129 = snap214
		d130 = snap215
		d131 = snap216
		d132 = snap217
		d133 = snap218
		d134 = snap219
		d135 = snap220
		d136 = snap221
		d137 = snap222
		d140 = snap223
		if !bbs[13].Rendered {
			return bbs[13].RenderPS(ps141)
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
			d52 = ps.OverlayValues[52]
		}
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != scm.LocNone {
			d109 = ps.OverlayValues[109]
		}
		if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != scm.LocNone {
			d110 = ps.OverlayValues[110]
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
		if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != scm.LocNone {
			d140 = ps.OverlayValues[140]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		ctx.EnsureDesc(&d1)
		var d225 scm.JITValueDesc
		if d1.Loc == scm.LocImm {
			d225 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegReg(scratch, d1.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d225 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d225)
		}
		if d225.Loc == scm.LocImm {
			d225 = scm.JITValueDesc{Loc: scm.LocImm, Type: d225.Type, Imm: scm.NewInt(int64(uint64(d225.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d225.Reg, 32)
			ctx.EmitShrRegImm8(d225.Reg, 32)
		}
		if d225.Loc == scm.LocReg && d1.Loc == scm.LocReg && d225.Reg == d1.Reg {
			ctx.TransferReg(d1.Reg)
			d1.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d225)
		ctx.EmitStoreToStack(d225, int32(bbs[4].PhiBase)+int32(32))
		ctx.StabilizeDescForControlFlow(&d225)
		ctx.EnsureDesc(&d1)
		ctx.EnsureDesc(&d1)
		var d226 scm.JITValueDesc
		if d1.Loc == scm.LocImm {
			d226 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegReg(scratch, d1.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d226)
		}
		if d226.Loc == scm.LocImm {
			d226 = scm.JITValueDesc{Loc: scm.LocImm, Type: d226.Type, Imm: scm.NewInt(int64(uint64(d226.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d226.Reg, 32)
			ctx.EmitShrRegImm8(d226.Reg, 32)
		}
		if d226.Loc == scm.LocReg && d1.Loc == scm.LocReg && d226.Reg == d1.Reg {
			ctx.TransferReg(d1.Reg)
			d1.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d226)
		ctx.EmitStoreToStack(d226, int32(bbs[4].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d226)
		if ps.General {
			ctx.SyncDesc(&d2)
			if d2.Loc == scm.LocReg {
				ctx.ProtectReg(d2.Reg)
			} else if d2.Loc == scm.LocRegPair {
				ctx.ProtectReg(d2.Reg)
				ctx.ProtectReg(d2.Reg2)
			}
			d227 = d2
			if d227.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d227)
			d228 = d227
			if d228.Loc == scm.LocImm {
				d228 = scm.JITValueDesc{Loc: scm.LocImm, Type: d228.Type, Imm: scm.NewInt(int64(uint64(d228.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d228.Reg, 32)
				ctx.EmitShrRegImm8(d228.Reg, 32)
			}
			ctx.EmitStoreToStack(d228, int32(bbs[4].PhiBase)+int32(16))
			if d2.Loc == scm.LocReg {
				ctx.UnprotectReg(d2.Reg)
			} else if d2.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d2.Reg)
				ctx.UnprotectReg(d2.Reg2)
			}
		}
		ps229 := scm.PhiState{General: ps.General}
		ps229.OverlayValues = make([]scm.JITValueDesc, 229)
		ps229.OverlayValues[1] = d1
		ps229.OverlayValues[2] = d2
		ps229.OverlayValues[3] = d3
		ps229.OverlayValues[4] = d4
		ps229.OverlayValues[5] = d5
		ps229.OverlayValues[6] = d6
		ps229.OverlayValues[7] = d7
		ps229.OverlayValues[8] = d8
		ps229.OverlayValues[9] = d9
		ps229.OverlayValues[10] = d10
		ps229.OverlayValues[11] = d11
		ps229.OverlayValues[12] = d12
		ps229.OverlayValues[13] = d13
		ps229.OverlayValues[14] = d14
		ps229.OverlayValues[15] = d15
		ps229.OverlayValues[17] = d17
		ps229.OverlayValues[18] = d18
		ps229.OverlayValues[19] = d19
		ps229.OverlayValues[20] = d20
		ps229.OverlayValues[21] = d21
		ps229.OverlayValues[22] = d22
		ps229.OverlayValues[23] = d23
		ps229.OverlayValues[24] = d24
		ps229.OverlayValues[25] = d25
		ps229.OverlayValues[26] = d26
		ps229.OverlayValues[27] = d27
		ps229.OverlayValues[28] = d28
		ps229.OverlayValues[29] = d29
		ps229.OverlayValues[30] = d30
		ps229.OverlayValues[31] = d31
		ps229.OverlayValues[32] = d32
		ps229.OverlayValues[33] = d33
		ps229.OverlayValues[34] = d34
		ps229.OverlayValues[35] = d35
		ps229.OverlayValues[36] = d36
		ps229.OverlayValues[37] = d37
		ps229.OverlayValues[38] = d38
		ps229.OverlayValues[39] = d39
		ps229.OverlayValues[40] = d40
		ps229.OverlayValues[41] = d41
		ps229.OverlayValues[42] = d42
		ps229.OverlayValues[43] = d43
		ps229.OverlayValues[44] = d44
		ps229.OverlayValues[45] = d45
		ps229.OverlayValues[46] = d46
		ps229.OverlayValues[47] = d47
		ps229.OverlayValues[48] = d48
		ps229.OverlayValues[49] = d49
		ps229.OverlayValues[52] = d52
		ps229.OverlayValues[53] = d53
		ps229.OverlayValues[54] = d54
		ps229.OverlayValues[109] = d109
		ps229.OverlayValues[110] = d110
		ps229.OverlayValues[111] = d111
		ps229.OverlayValues[112] = d112
		ps229.OverlayValues[113] = d113
		ps229.OverlayValues[114] = d114
		ps229.OverlayValues[115] = d115
		ps229.OverlayValues[116] = d116
		ps229.OverlayValues[117] = d117
		ps229.OverlayValues[118] = d118
		ps229.OverlayValues[119] = d119
		ps229.OverlayValues[120] = d120
		ps229.OverlayValues[121] = d121
		ps229.OverlayValues[122] = d122
		ps229.OverlayValues[123] = d123
		ps229.OverlayValues[124] = d124
		ps229.OverlayValues[125] = d125
		ps229.OverlayValues[126] = d126
		ps229.OverlayValues[127] = d127
		ps229.OverlayValues[128] = d128
		ps229.OverlayValues[129] = d129
		ps229.OverlayValues[130] = d130
		ps229.OverlayValues[131] = d131
		ps229.OverlayValues[132] = d132
		ps229.OverlayValues[133] = d133
		ps229.OverlayValues[134] = d134
		ps229.OverlayValues[135] = d135
		ps229.OverlayValues[136] = d136
		ps229.OverlayValues[137] = d137
		ps229.OverlayValues[140] = d140
		ps229.OverlayValues[225] = d225
		ps229.OverlayValues[226] = d226
		ps229.OverlayValues[227] = d227
		ps229.OverlayValues[228] = d228
		ps229.PhiValues = make([]scm.JITValueDesc, 3)
		d230 = d2
		ps229.PhiValues[1] = d230
		if ps229.General && bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
			return result
		}
		return bbs[4].RenderPS(ps229)
		return result
	}
	bbs[4].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d231 := ps.PhiValues[0]
				ctx.EnsureDesc(&d231)
				ctx.EmitStoreToStack(d231, int32(bbs[4].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d232 := ps.PhiValues[1]
				ctx.EnsureDesc(&d232)
				ctx.EmitStoreToStack(d232, int32(bbs[4].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d233 := ps.PhiValues[2]
				ctx.EnsureDesc(&d233)
				ctx.EmitStoreToStack(d233, int32(bbs[4].PhiBase)+int32(32))
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
			d52 = ps.OverlayValues[52]
		}
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != scm.LocNone {
			d109 = ps.OverlayValues[109]
		}
		if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != scm.LocNone {
			d110 = ps.OverlayValues[110]
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
		if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != scm.LocNone {
			d140 = ps.OverlayValues[140]
		}
		if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != scm.LocNone {
			d225 = ps.OverlayValues[225]
		}
		if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != scm.LocNone {
			d226 = ps.OverlayValues[226]
		}
		if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
			d227 = ps.OverlayValues[227]
		}
		if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
			d228 = ps.OverlayValues[228]
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
		var d234 scm.JITValueDesc
		if d6.Loc == scm.LocImm && d7.Loc == scm.LocImm {
			d234 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d6.Imm.Int()) == uint64(d7.Imm.Int()))}
		} else if d7.Loc == scm.LocImm {
			r77 := ctx.AllocRegExcept(d6.Reg)
			if d7.Imm.Int() >= -2147483648 && d7.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d6.Reg, int32(d7.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d7.Imm.Int()))
				ctx.EmitCmpInt64(d6.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r77, scm.CondEqual)
			d234 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r77}
			ctx.BindReg(r77, &d234)
		} else if d6.Loc == scm.LocImm {
			r78 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d6.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d7.Reg)
			ctx.EmitSetcc(r78, scm.CondEqual)
			d234 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r78}
			ctx.BindReg(r78, &d234)
		} else {
			r79 := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitCmpInt64(d6.Reg, d7.Reg)
			ctx.EmitSetcc(r79, scm.CondEqual)
			d234 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r79}
			ctx.BindReg(r79, &d234)
		}
		d235 = d234
		ctx.EnsureDesc(&d235)
		if d235.Loc != scm.LocImm && d235.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d235.Loc == scm.LocImm {
			if d235.Imm.Bool() {
				if ps.General {
					ctx.SyncDesc(&d6)
					if d6.Loc == scm.LocReg {
						ctx.ProtectReg(d6.Reg)
					} else if d6.Loc == scm.LocRegPair {
						ctx.ProtectReg(d6.Reg)
						ctx.ProtectReg(d6.Reg2)
					}
					d236 = d6
					if d236.Loc == scm.LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d236)
					d237 = d236
					if d237.Loc == scm.LocImm {
						d237 = scm.JITValueDesc{Loc: scm.LocImm, Type: d237.Type, Imm: scm.NewInt(int64(uint64(d237.Imm.Int()) & 0xffffffff))}
					} else {
						ctx.EmitShlRegImm8(d237.Reg, 32)
						ctx.EmitShrRegImm8(d237.Reg, 32)
					}
					ctx.EmitStoreToStack(d237, int32(bbs[2].PhiBase)+int32(0))
					if d6.Loc == scm.LocReg {
						ctx.UnprotectReg(d6.Reg)
					} else if d6.Loc == scm.LocRegPair {
						ctx.UnprotectReg(d6.Reg)
						ctx.UnprotectReg(d6.Reg2)
					}
				}
				ps238 := scm.PhiState{General: ps.General}
				ps238.OverlayValues = make([]scm.JITValueDesc, 238)
				ps238.OverlayValues[1] = d1
				ps238.OverlayValues[2] = d2
				ps238.OverlayValues[3] = d3
				ps238.OverlayValues[4] = d4
				ps238.OverlayValues[5] = d5
				ps238.OverlayValues[6] = d6
				ps238.OverlayValues[7] = d7
				ps238.OverlayValues[8] = d8
				ps238.OverlayValues[9] = d9
				ps238.OverlayValues[10] = d10
				ps238.OverlayValues[11] = d11
				ps238.OverlayValues[12] = d12
				ps238.OverlayValues[13] = d13
				ps238.OverlayValues[14] = d14
				ps238.OverlayValues[15] = d15
				ps238.OverlayValues[17] = d17
				ps238.OverlayValues[18] = d18
				ps238.OverlayValues[19] = d19
				ps238.OverlayValues[20] = d20
				ps238.OverlayValues[21] = d21
				ps238.OverlayValues[22] = d22
				ps238.OverlayValues[23] = d23
				ps238.OverlayValues[24] = d24
				ps238.OverlayValues[25] = d25
				ps238.OverlayValues[26] = d26
				ps238.OverlayValues[27] = d27
				ps238.OverlayValues[28] = d28
				ps238.OverlayValues[29] = d29
				ps238.OverlayValues[30] = d30
				ps238.OverlayValues[31] = d31
				ps238.OverlayValues[32] = d32
				ps238.OverlayValues[33] = d33
				ps238.OverlayValues[34] = d34
				ps238.OverlayValues[35] = d35
				ps238.OverlayValues[36] = d36
				ps238.OverlayValues[37] = d37
				ps238.OverlayValues[38] = d38
				ps238.OverlayValues[39] = d39
				ps238.OverlayValues[40] = d40
				ps238.OverlayValues[41] = d41
				ps238.OverlayValues[42] = d42
				ps238.OverlayValues[43] = d43
				ps238.OverlayValues[44] = d44
				ps238.OverlayValues[45] = d45
				ps238.OverlayValues[46] = d46
				ps238.OverlayValues[47] = d47
				ps238.OverlayValues[48] = d48
				ps238.OverlayValues[49] = d49
				ps238.OverlayValues[52] = d52
				ps238.OverlayValues[53] = d53
				ps238.OverlayValues[54] = d54
				ps238.OverlayValues[109] = d109
				ps238.OverlayValues[110] = d110
				ps238.OverlayValues[111] = d111
				ps238.OverlayValues[112] = d112
				ps238.OverlayValues[113] = d113
				ps238.OverlayValues[114] = d114
				ps238.OverlayValues[115] = d115
				ps238.OverlayValues[116] = d116
				ps238.OverlayValues[117] = d117
				ps238.OverlayValues[118] = d118
				ps238.OverlayValues[119] = d119
				ps238.OverlayValues[120] = d120
				ps238.OverlayValues[121] = d121
				ps238.OverlayValues[122] = d122
				ps238.OverlayValues[123] = d123
				ps238.OverlayValues[124] = d124
				ps238.OverlayValues[125] = d125
				ps238.OverlayValues[126] = d126
				ps238.OverlayValues[127] = d127
				ps238.OverlayValues[128] = d128
				ps238.OverlayValues[129] = d129
				ps238.OverlayValues[130] = d130
				ps238.OverlayValues[131] = d131
				ps238.OverlayValues[132] = d132
				ps238.OverlayValues[133] = d133
				ps238.OverlayValues[134] = d134
				ps238.OverlayValues[135] = d135
				ps238.OverlayValues[136] = d136
				ps238.OverlayValues[137] = d137
				ps238.OverlayValues[140] = d140
				ps238.OverlayValues[225] = d225
				ps238.OverlayValues[226] = d226
				ps238.OverlayValues[227] = d227
				ps238.OverlayValues[228] = d228
				ps238.OverlayValues[230] = d230
				ps238.OverlayValues[231] = d231
				ps238.OverlayValues[232] = d232
				ps238.OverlayValues[233] = d233
				ps238.OverlayValues[234] = d234
				ps238.OverlayValues[235] = d235
				ps238.OverlayValues[236] = d236
				ps238.OverlayValues[237] = d237
				ps238.PhiValues = make([]scm.JITValueDesc, 1)
				d239 = d6
				ps238.PhiValues[0] = d239
				return bbs[2].RenderPS(ps238)
			}
			if ps.General {
			}
			ps240 := scm.PhiState{General: ps.General}
			ps240.OverlayValues = make([]scm.JITValueDesc, 240)
			ps240.OverlayValues[1] = d1
			ps240.OverlayValues[2] = d2
			ps240.OverlayValues[3] = d3
			ps240.OverlayValues[4] = d4
			ps240.OverlayValues[5] = d5
			ps240.OverlayValues[6] = d6
			ps240.OverlayValues[7] = d7
			ps240.OverlayValues[8] = d8
			ps240.OverlayValues[9] = d9
			ps240.OverlayValues[10] = d10
			ps240.OverlayValues[11] = d11
			ps240.OverlayValues[12] = d12
			ps240.OverlayValues[13] = d13
			ps240.OverlayValues[14] = d14
			ps240.OverlayValues[15] = d15
			ps240.OverlayValues[17] = d17
			ps240.OverlayValues[18] = d18
			ps240.OverlayValues[19] = d19
			ps240.OverlayValues[20] = d20
			ps240.OverlayValues[21] = d21
			ps240.OverlayValues[22] = d22
			ps240.OverlayValues[23] = d23
			ps240.OverlayValues[24] = d24
			ps240.OverlayValues[25] = d25
			ps240.OverlayValues[26] = d26
			ps240.OverlayValues[27] = d27
			ps240.OverlayValues[28] = d28
			ps240.OverlayValues[29] = d29
			ps240.OverlayValues[30] = d30
			ps240.OverlayValues[31] = d31
			ps240.OverlayValues[32] = d32
			ps240.OverlayValues[33] = d33
			ps240.OverlayValues[34] = d34
			ps240.OverlayValues[35] = d35
			ps240.OverlayValues[36] = d36
			ps240.OverlayValues[37] = d37
			ps240.OverlayValues[38] = d38
			ps240.OverlayValues[39] = d39
			ps240.OverlayValues[40] = d40
			ps240.OverlayValues[41] = d41
			ps240.OverlayValues[42] = d42
			ps240.OverlayValues[43] = d43
			ps240.OverlayValues[44] = d44
			ps240.OverlayValues[45] = d45
			ps240.OverlayValues[46] = d46
			ps240.OverlayValues[47] = d47
			ps240.OverlayValues[48] = d48
			ps240.OverlayValues[49] = d49
			ps240.OverlayValues[52] = d52
			ps240.OverlayValues[53] = d53
			ps240.OverlayValues[54] = d54
			ps240.OverlayValues[109] = d109
			ps240.OverlayValues[110] = d110
			ps240.OverlayValues[111] = d111
			ps240.OverlayValues[112] = d112
			ps240.OverlayValues[113] = d113
			ps240.OverlayValues[114] = d114
			ps240.OverlayValues[115] = d115
			ps240.OverlayValues[116] = d116
			ps240.OverlayValues[117] = d117
			ps240.OverlayValues[118] = d118
			ps240.OverlayValues[119] = d119
			ps240.OverlayValues[120] = d120
			ps240.OverlayValues[121] = d121
			ps240.OverlayValues[122] = d122
			ps240.OverlayValues[123] = d123
			ps240.OverlayValues[124] = d124
			ps240.OverlayValues[125] = d125
			ps240.OverlayValues[126] = d126
			ps240.OverlayValues[127] = d127
			ps240.OverlayValues[128] = d128
			ps240.OverlayValues[129] = d129
			ps240.OverlayValues[130] = d130
			ps240.OverlayValues[131] = d131
			ps240.OverlayValues[132] = d132
			ps240.OverlayValues[133] = d133
			ps240.OverlayValues[134] = d134
			ps240.OverlayValues[135] = d135
			ps240.OverlayValues[136] = d136
			ps240.OverlayValues[137] = d137
			ps240.OverlayValues[140] = d140
			ps240.OverlayValues[225] = d225
			ps240.OverlayValues[226] = d226
			ps240.OverlayValues[227] = d227
			ps240.OverlayValues[228] = d228
			ps240.OverlayValues[230] = d230
			ps240.OverlayValues[231] = d231
			ps240.OverlayValues[232] = d232
			ps240.OverlayValues[233] = d233
			ps240.OverlayValues[234] = d234
			ps240.OverlayValues[235] = d235
			ps240.OverlayValues[236] = d236
			ps240.OverlayValues[237] = d237
			ps240.OverlayValues[239] = d239
			return bbs[6].RenderPS(ps240)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d241 := ps.PhiValues[0]
				ctx.EnsureDesc(&d241)
				ctx.EmitStoreToStack(d241, int32(bbs[4].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d242 := ps.PhiValues[1]
				ctx.EnsureDesc(&d242)
				ctx.EmitStoreToStack(d242, int32(bbs[4].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d243 := ps.PhiValues[2]
				ctx.EnsureDesc(&d243)
				ctx.EmitStoreToStack(d243, int32(bbs[4].PhiBase)+int32(32))
			}
			ps.General = true
			return bbs[4].RenderPS(ps)
		}
		lbl21 := ctx.ReserveLabel()
		lbl22 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d235.Reg, 0)
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
		d244 = d6
		if d244.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d244)
		d245 = d244
		if d245.Loc == scm.LocImm {
			d245 = scm.JITValueDesc{Loc: scm.LocImm, Type: d245.Type, Imm: scm.NewInt(int64(uint64(d245.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d245.Reg, 32)
			ctx.EmitShrRegImm8(d245.Reg, 32)
		}
		ctx.EmitStoreToStack(d245, int32(bbs[2].PhiBase)+int32(0))
		if d6.Loc == scm.LocReg {
			ctx.UnprotectReg(d6.Reg)
		} else if d6.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d6.Reg)
			ctx.UnprotectReg(d6.Reg2)
		}
		ctx.EmitJmp(lbl3)
		ctx.MarkLabel(lbl22)
		ctx.EmitJmp(lbl7)
		ps246 := scm.PhiState{General: true}
		ps246.OverlayValues = make([]scm.JITValueDesc, 246)
		ps246.OverlayValues[1] = d1
		ps246.OverlayValues[2] = d2
		ps246.OverlayValues[3] = d3
		ps246.OverlayValues[4] = d4
		ps246.OverlayValues[5] = d5
		ps246.OverlayValues[6] = d6
		ps246.OverlayValues[7] = d7
		ps246.OverlayValues[8] = d8
		ps246.OverlayValues[9] = d9
		ps246.OverlayValues[10] = d10
		ps246.OverlayValues[11] = d11
		ps246.OverlayValues[12] = d12
		ps246.OverlayValues[13] = d13
		ps246.OverlayValues[14] = d14
		ps246.OverlayValues[15] = d15
		ps246.OverlayValues[17] = d17
		ps246.OverlayValues[18] = d18
		ps246.OverlayValues[19] = d19
		ps246.OverlayValues[20] = d20
		ps246.OverlayValues[21] = d21
		ps246.OverlayValues[22] = d22
		ps246.OverlayValues[23] = d23
		ps246.OverlayValues[24] = d24
		ps246.OverlayValues[25] = d25
		ps246.OverlayValues[26] = d26
		ps246.OverlayValues[27] = d27
		ps246.OverlayValues[28] = d28
		ps246.OverlayValues[29] = d29
		ps246.OverlayValues[30] = d30
		ps246.OverlayValues[31] = d31
		ps246.OverlayValues[32] = d32
		ps246.OverlayValues[33] = d33
		ps246.OverlayValues[34] = d34
		ps246.OverlayValues[35] = d35
		ps246.OverlayValues[36] = d36
		ps246.OverlayValues[37] = d37
		ps246.OverlayValues[38] = d38
		ps246.OverlayValues[39] = d39
		ps246.OverlayValues[40] = d40
		ps246.OverlayValues[41] = d41
		ps246.OverlayValues[42] = d42
		ps246.OverlayValues[43] = d43
		ps246.OverlayValues[44] = d44
		ps246.OverlayValues[45] = d45
		ps246.OverlayValues[46] = d46
		ps246.OverlayValues[47] = d47
		ps246.OverlayValues[48] = d48
		ps246.OverlayValues[49] = d49
		ps246.OverlayValues[52] = d52
		ps246.OverlayValues[53] = d53
		ps246.OverlayValues[54] = d54
		ps246.OverlayValues[109] = d109
		ps246.OverlayValues[110] = d110
		ps246.OverlayValues[111] = d111
		ps246.OverlayValues[112] = d112
		ps246.OverlayValues[113] = d113
		ps246.OverlayValues[114] = d114
		ps246.OverlayValues[115] = d115
		ps246.OverlayValues[116] = d116
		ps246.OverlayValues[117] = d117
		ps246.OverlayValues[118] = d118
		ps246.OverlayValues[119] = d119
		ps246.OverlayValues[120] = d120
		ps246.OverlayValues[121] = d121
		ps246.OverlayValues[122] = d122
		ps246.OverlayValues[123] = d123
		ps246.OverlayValues[124] = d124
		ps246.OverlayValues[125] = d125
		ps246.OverlayValues[126] = d126
		ps246.OverlayValues[127] = d127
		ps246.OverlayValues[128] = d128
		ps246.OverlayValues[129] = d129
		ps246.OverlayValues[130] = d130
		ps246.OverlayValues[131] = d131
		ps246.OverlayValues[132] = d132
		ps246.OverlayValues[133] = d133
		ps246.OverlayValues[134] = d134
		ps246.OverlayValues[135] = d135
		ps246.OverlayValues[136] = d136
		ps246.OverlayValues[137] = d137
		ps246.OverlayValues[140] = d140
		ps246.OverlayValues[225] = d225
		ps246.OverlayValues[226] = d226
		ps246.OverlayValues[227] = d227
		ps246.OverlayValues[228] = d228
		ps246.OverlayValues[230] = d230
		ps246.OverlayValues[231] = d231
		ps246.OverlayValues[232] = d232
		ps246.OverlayValues[233] = d233
		ps246.OverlayValues[234] = d234
		ps246.OverlayValues[235] = d235
		ps246.OverlayValues[236] = d236
		ps246.OverlayValues[237] = d237
		ps246.OverlayValues[239] = d239
		ps246.OverlayValues[241] = d241
		ps246.OverlayValues[242] = d242
		ps246.OverlayValues[243] = d243
		ps246.OverlayValues[244] = d244
		ps246.OverlayValues[245] = d245
		ps246.PhiValues = make([]scm.JITValueDesc, 1)
		d248 = d6
		ps246.PhiValues[0] = d248
		ps247 := scm.PhiState{General: true}
		ps247.OverlayValues = make([]scm.JITValueDesc, 249)
		ps247.OverlayValues[1] = d1
		ps247.OverlayValues[2] = d2
		ps247.OverlayValues[3] = d3
		ps247.OverlayValues[4] = d4
		ps247.OverlayValues[5] = d5
		ps247.OverlayValues[6] = d6
		ps247.OverlayValues[7] = d7
		ps247.OverlayValues[8] = d8
		ps247.OverlayValues[9] = d9
		ps247.OverlayValues[10] = d10
		ps247.OverlayValues[11] = d11
		ps247.OverlayValues[12] = d12
		ps247.OverlayValues[13] = d13
		ps247.OverlayValues[14] = d14
		ps247.OverlayValues[15] = d15
		ps247.OverlayValues[17] = d17
		ps247.OverlayValues[18] = d18
		ps247.OverlayValues[19] = d19
		ps247.OverlayValues[20] = d20
		ps247.OverlayValues[21] = d21
		ps247.OverlayValues[22] = d22
		ps247.OverlayValues[23] = d23
		ps247.OverlayValues[24] = d24
		ps247.OverlayValues[25] = d25
		ps247.OverlayValues[26] = d26
		ps247.OverlayValues[27] = d27
		ps247.OverlayValues[28] = d28
		ps247.OverlayValues[29] = d29
		ps247.OverlayValues[30] = d30
		ps247.OverlayValues[31] = d31
		ps247.OverlayValues[32] = d32
		ps247.OverlayValues[33] = d33
		ps247.OverlayValues[34] = d34
		ps247.OverlayValues[35] = d35
		ps247.OverlayValues[36] = d36
		ps247.OverlayValues[37] = d37
		ps247.OverlayValues[38] = d38
		ps247.OverlayValues[39] = d39
		ps247.OverlayValues[40] = d40
		ps247.OverlayValues[41] = d41
		ps247.OverlayValues[42] = d42
		ps247.OverlayValues[43] = d43
		ps247.OverlayValues[44] = d44
		ps247.OverlayValues[45] = d45
		ps247.OverlayValues[46] = d46
		ps247.OverlayValues[47] = d47
		ps247.OverlayValues[48] = d48
		ps247.OverlayValues[49] = d49
		ps247.OverlayValues[52] = d52
		ps247.OverlayValues[53] = d53
		ps247.OverlayValues[54] = d54
		ps247.OverlayValues[109] = d109
		ps247.OverlayValues[110] = d110
		ps247.OverlayValues[111] = d111
		ps247.OverlayValues[112] = d112
		ps247.OverlayValues[113] = d113
		ps247.OverlayValues[114] = d114
		ps247.OverlayValues[115] = d115
		ps247.OverlayValues[116] = d116
		ps247.OverlayValues[117] = d117
		ps247.OverlayValues[118] = d118
		ps247.OverlayValues[119] = d119
		ps247.OverlayValues[120] = d120
		ps247.OverlayValues[121] = d121
		ps247.OverlayValues[122] = d122
		ps247.OverlayValues[123] = d123
		ps247.OverlayValues[124] = d124
		ps247.OverlayValues[125] = d125
		ps247.OverlayValues[126] = d126
		ps247.OverlayValues[127] = d127
		ps247.OverlayValues[128] = d128
		ps247.OverlayValues[129] = d129
		ps247.OverlayValues[130] = d130
		ps247.OverlayValues[131] = d131
		ps247.OverlayValues[132] = d132
		ps247.OverlayValues[133] = d133
		ps247.OverlayValues[134] = d134
		ps247.OverlayValues[135] = d135
		ps247.OverlayValues[136] = d136
		ps247.OverlayValues[137] = d137
		ps247.OverlayValues[140] = d140
		ps247.OverlayValues[225] = d225
		ps247.OverlayValues[226] = d226
		ps247.OverlayValues[227] = d227
		ps247.OverlayValues[228] = d228
		ps247.OverlayValues[230] = d230
		ps247.OverlayValues[231] = d231
		ps247.OverlayValues[232] = d232
		ps247.OverlayValues[233] = d233
		ps247.OverlayValues[234] = d234
		ps247.OverlayValues[235] = d235
		ps247.OverlayValues[236] = d236
		ps247.OverlayValues[237] = d237
		ps247.OverlayValues[239] = d239
		ps247.OverlayValues[241] = d241
		ps247.OverlayValues[242] = d242
		ps247.OverlayValues[243] = d243
		ps247.OverlayValues[244] = d244
		ps247.OverlayValues[245] = d245
		ps247.OverlayValues[248] = d248
		snap249 := d1
		snap250 := d2
		snap251 := d3
		snap252 := d4
		snap253 := d5
		snap254 := d6
		snap255 := d7
		snap256 := d8
		snap257 := d9
		snap258 := d10
		snap259 := d11
		snap260 := d12
		snap261 := d13
		snap262 := d14
		snap263 := d15
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
		snap292 := d45
		snap293 := d46
		snap294 := d47
		snap295 := d48
		snap296 := d49
		snap297 := d52
		snap298 := d53
		snap299 := d54
		snap300 := d109
		snap301 := d110
		snap302 := d111
		snap303 := d112
		snap304 := d113
		snap305 := d114
		snap306 := d115
		snap307 := d116
		snap308 := d117
		snap309 := d118
		snap310 := d119
		snap311 := d120
		snap312 := d121
		snap313 := d122
		snap314 := d123
		snap315 := d124
		snap316 := d125
		snap317 := d126
		snap318 := d127
		snap319 := d128
		snap320 := d129
		snap321 := d130
		snap322 := d131
		snap323 := d132
		snap324 := d133
		snap325 := d134
		snap326 := d135
		snap327 := d136
		snap328 := d137
		snap329 := d140
		snap330 := d225
		snap331 := d226
		snap332 := d227
		snap333 := d228
		snap334 := d230
		snap335 := d231
		snap336 := d232
		snap337 := d233
		snap338 := d234
		snap339 := d235
		snap340 := d236
		snap341 := d237
		snap342 := d239
		snap343 := d241
		snap344 := d242
		snap345 := d243
		snap346 := d244
		snap347 := d245
		snap348 := d248
		alloc349 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps246)
		}
		ctx.RestoreAllocState(alloc349)
		d1 = snap249
		d2 = snap250
		d3 = snap251
		d4 = snap252
		d5 = snap253
		d6 = snap254
		d7 = snap255
		d8 = snap256
		d9 = snap257
		d10 = snap258
		d11 = snap259
		d12 = snap260
		d13 = snap261
		d14 = snap262
		d15 = snap263
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
		d45 = snap292
		d46 = snap293
		d47 = snap294
		d48 = snap295
		d49 = snap296
		d52 = snap297
		d53 = snap298
		d54 = snap299
		d109 = snap300
		d110 = snap301
		d111 = snap302
		d112 = snap303
		d113 = snap304
		d114 = snap305
		d115 = snap306
		d116 = snap307
		d117 = snap308
		d118 = snap309
		d119 = snap310
		d120 = snap311
		d121 = snap312
		d122 = snap313
		d123 = snap314
		d124 = snap315
		d125 = snap316
		d126 = snap317
		d127 = snap318
		d128 = snap319
		d129 = snap320
		d130 = snap321
		d131 = snap322
		d132 = snap323
		d133 = snap324
		d134 = snap325
		d135 = snap326
		d136 = snap327
		d137 = snap328
		d140 = snap329
		d225 = snap330
		d226 = snap331
		d227 = snap332
		d228 = snap333
		d230 = snap334
		d231 = snap335
		d232 = snap336
		d233 = snap337
		d234 = snap338
		d235 = snap339
		d236 = snap340
		d237 = snap341
		d239 = snap342
		d241 = snap343
		d242 = snap344
		d243 = snap345
		d244 = snap346
		d245 = snap347
		d248 = snap348
		if !bbs[6].Rendered {
			return bbs[6].RenderPS(ps247)
		}
		return result
		ctx.FreeDesc(&d234)
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
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
			d52 = ps.OverlayValues[52]
		}
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != scm.LocNone {
			d109 = ps.OverlayValues[109]
		}
		if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != scm.LocNone {
			d110 = ps.OverlayValues[110]
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
		if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != scm.LocNone {
			d140 = ps.OverlayValues[140]
		}
		if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != scm.LocNone {
			d225 = ps.OverlayValues[225]
		}
		if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != scm.LocNone {
			d226 = ps.OverlayValues[226]
		}
		if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
			d227 = ps.OverlayValues[227]
		}
		if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
			d228 = ps.OverlayValues[228]
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
		if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != scm.LocNone {
			d234 = ps.OverlayValues[234]
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
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
		}
		if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != scm.LocNone {
			d241 = ps.OverlayValues[241]
		}
		if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != scm.LocNone {
			d242 = ps.OverlayValues[242]
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
		if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != scm.LocNone {
			d248 = ps.OverlayValues[248]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		ctx.EnsureDesc(&d1)
		var d350 scm.JITValueDesc
		if d1.Loc == scm.LocImm {
			d350 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegReg(scratch, d1.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d350 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d350)
		}
		if d350.Loc == scm.LocImm {
			d350 = scm.JITValueDesc{Loc: scm.LocImm, Type: d350.Type, Imm: scm.NewInt(int64(uint64(d350.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d350.Reg, 32)
			ctx.EmitShrRegImm8(d350.Reg, 32)
		}
		if d350.Loc == scm.LocReg && d1.Loc == scm.LocReg && d350.Reg == d1.Reg {
			ctx.TransferReg(d1.Reg)
			d1.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d350)
		ctx.EmitStoreToStack(d350, int32(bbs[4].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d350)
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
			d351 = d1
			if d351.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d351)
			d352 = d351
			if d352.Loc == scm.LocImm {
				d352 = scm.JITValueDesc{Loc: scm.LocImm, Type: d352.Type, Imm: scm.NewInt(int64(uint64(d352.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d352.Reg, 32)
				ctx.EmitShrRegImm8(d352.Reg, 32)
			}
			ctx.EmitStoreToStack(d352, int32(bbs[4].PhiBase)+int32(16))
			d353 = d3
			if d353.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d353)
			d354 = d353
			if d354.Loc == scm.LocImm {
				d354 = scm.JITValueDesc{Loc: scm.LocImm, Type: d354.Type, Imm: scm.NewInt(int64(uint64(d354.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d354.Reg, 32)
				ctx.EmitShrRegImm8(d354.Reg, 32)
			}
			ctx.EmitStoreToStack(d354, int32(bbs[4].PhiBase)+int32(32))
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
		ps355 := scm.PhiState{General: ps.General}
		ps355.OverlayValues = make([]scm.JITValueDesc, 355)
		ps355.OverlayValues[1] = d1
		ps355.OverlayValues[2] = d2
		ps355.OverlayValues[3] = d3
		ps355.OverlayValues[4] = d4
		ps355.OverlayValues[5] = d5
		ps355.OverlayValues[6] = d6
		ps355.OverlayValues[7] = d7
		ps355.OverlayValues[8] = d8
		ps355.OverlayValues[9] = d9
		ps355.OverlayValues[10] = d10
		ps355.OverlayValues[11] = d11
		ps355.OverlayValues[12] = d12
		ps355.OverlayValues[13] = d13
		ps355.OverlayValues[14] = d14
		ps355.OverlayValues[15] = d15
		ps355.OverlayValues[17] = d17
		ps355.OverlayValues[18] = d18
		ps355.OverlayValues[19] = d19
		ps355.OverlayValues[20] = d20
		ps355.OverlayValues[21] = d21
		ps355.OverlayValues[22] = d22
		ps355.OverlayValues[23] = d23
		ps355.OverlayValues[24] = d24
		ps355.OverlayValues[25] = d25
		ps355.OverlayValues[26] = d26
		ps355.OverlayValues[27] = d27
		ps355.OverlayValues[28] = d28
		ps355.OverlayValues[29] = d29
		ps355.OverlayValues[30] = d30
		ps355.OverlayValues[31] = d31
		ps355.OverlayValues[32] = d32
		ps355.OverlayValues[33] = d33
		ps355.OverlayValues[34] = d34
		ps355.OverlayValues[35] = d35
		ps355.OverlayValues[36] = d36
		ps355.OverlayValues[37] = d37
		ps355.OverlayValues[38] = d38
		ps355.OverlayValues[39] = d39
		ps355.OverlayValues[40] = d40
		ps355.OverlayValues[41] = d41
		ps355.OverlayValues[42] = d42
		ps355.OverlayValues[43] = d43
		ps355.OverlayValues[44] = d44
		ps355.OverlayValues[45] = d45
		ps355.OverlayValues[46] = d46
		ps355.OverlayValues[47] = d47
		ps355.OverlayValues[48] = d48
		ps355.OverlayValues[49] = d49
		ps355.OverlayValues[52] = d52
		ps355.OverlayValues[53] = d53
		ps355.OverlayValues[54] = d54
		ps355.OverlayValues[109] = d109
		ps355.OverlayValues[110] = d110
		ps355.OverlayValues[111] = d111
		ps355.OverlayValues[112] = d112
		ps355.OverlayValues[113] = d113
		ps355.OverlayValues[114] = d114
		ps355.OverlayValues[115] = d115
		ps355.OverlayValues[116] = d116
		ps355.OverlayValues[117] = d117
		ps355.OverlayValues[118] = d118
		ps355.OverlayValues[119] = d119
		ps355.OverlayValues[120] = d120
		ps355.OverlayValues[121] = d121
		ps355.OverlayValues[122] = d122
		ps355.OverlayValues[123] = d123
		ps355.OverlayValues[124] = d124
		ps355.OverlayValues[125] = d125
		ps355.OverlayValues[126] = d126
		ps355.OverlayValues[127] = d127
		ps355.OverlayValues[128] = d128
		ps355.OverlayValues[129] = d129
		ps355.OverlayValues[130] = d130
		ps355.OverlayValues[131] = d131
		ps355.OverlayValues[132] = d132
		ps355.OverlayValues[133] = d133
		ps355.OverlayValues[134] = d134
		ps355.OverlayValues[135] = d135
		ps355.OverlayValues[136] = d136
		ps355.OverlayValues[137] = d137
		ps355.OverlayValues[140] = d140
		ps355.OverlayValues[225] = d225
		ps355.OverlayValues[226] = d226
		ps355.OverlayValues[227] = d227
		ps355.OverlayValues[228] = d228
		ps355.OverlayValues[230] = d230
		ps355.OverlayValues[231] = d231
		ps355.OverlayValues[232] = d232
		ps355.OverlayValues[233] = d233
		ps355.OverlayValues[234] = d234
		ps355.OverlayValues[235] = d235
		ps355.OverlayValues[236] = d236
		ps355.OverlayValues[237] = d237
		ps355.OverlayValues[239] = d239
		ps355.OverlayValues[241] = d241
		ps355.OverlayValues[242] = d242
		ps355.OverlayValues[243] = d243
		ps355.OverlayValues[244] = d244
		ps355.OverlayValues[245] = d245
		ps355.OverlayValues[248] = d248
		ps355.OverlayValues[350] = d350
		ps355.OverlayValues[351] = d351
		ps355.OverlayValues[352] = d352
		ps355.OverlayValues[353] = d353
		ps355.OverlayValues[354] = d354
		ps355.PhiValues = make([]scm.JITValueDesc, 3)
		d356 = d1
		ps355.PhiValues[1] = d356
		d357 = d3
		ps355.PhiValues[2] = d357
		if ps355.General && bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
			return result
		}
		return bbs[4].RenderPS(ps355)
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
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
			d52 = ps.OverlayValues[52]
		}
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != scm.LocNone {
			d109 = ps.OverlayValues[109]
		}
		if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != scm.LocNone {
			d110 = ps.OverlayValues[110]
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
		if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != scm.LocNone {
			d140 = ps.OverlayValues[140]
		}
		if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != scm.LocNone {
			d225 = ps.OverlayValues[225]
		}
		if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != scm.LocNone {
			d226 = ps.OverlayValues[226]
		}
		if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
			d227 = ps.OverlayValues[227]
		}
		if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
			d228 = ps.OverlayValues[228]
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
		if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != scm.LocNone {
			d234 = ps.OverlayValues[234]
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
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
		}
		if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != scm.LocNone {
			d241 = ps.OverlayValues[241]
		}
		if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != scm.LocNone {
			d242 = ps.OverlayValues[242]
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
		if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != scm.LocNone {
			d248 = ps.OverlayValues[248]
		}
		if len(ps.OverlayValues) > 350 && ps.OverlayValues[350].Loc != scm.LocNone {
			d350 = ps.OverlayValues[350]
		}
		if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != scm.LocNone {
			d351 = ps.OverlayValues[351]
		}
		if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != scm.LocNone {
			d352 = ps.OverlayValues[352]
		}
		if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != scm.LocNone {
			d353 = ps.OverlayValues[353]
		}
		if len(ps.OverlayValues) > 354 && ps.OverlayValues[354].Loc != scm.LocNone {
			d354 = ps.OverlayValues[354]
		}
		if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != scm.LocNone {
			d356 = ps.OverlayValues[356]
		}
		if len(ps.OverlayValues) > 357 && ps.OverlayValues[357].Loc != scm.LocNone {
			d357 = ps.OverlayValues[357]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		d358 = d5
		_ = d358
		ctx.StabilizeDescForControlFlow(&d358)
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
		ctx.EnsureDesc(&d358)
		ctx.EnsureDesc(&d358)
		var d359 scm.JITValueDesc
		if d358.Loc == scm.LocImm {
			d359 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d358.Imm.Int()))))}
		} else {
			r80 := ctx.AllocReg()
			ctx.EmitMovRegReg(r80, d358.Reg)
			ctx.EmitShlRegImm8(r80, 32)
			ctx.EmitShrRegImm8(r80, 32)
			d359 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
			ctx.BindReg(r80, &d359)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d360 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			r81 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r81, fieldAddr)
			d360 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r81}
			ctx.BindReg(r81, &d360)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			r82 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r82, thisptr.Reg, off)
			d360 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r82}
			ctx.BindReg(r82, &d360)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d360)
		ctx.EnsureDesc(&d360)
		var d361 scm.JITValueDesc
		if d360.Loc == scm.LocImm {
			d361 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d360.Imm.Int()))))}
		} else {
			r83 := ctx.AllocReg()
			ctx.EmitMovRegReg(r83, d360.Reg)
			ctx.EmitShlRegImm8(r83, 56)
			ctx.EmitShrRegImm8(r83, 56)
			d361 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
			ctx.BindReg(r83, &d361)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d359)
		ctx.EnsureDesc(&d361)
		ctx.EnsureDescsTogether(&d359, &d361)
		var d362 scm.JITValueDesc
		if d359.Loc == scm.LocImm && d361.Loc == scm.LocImm {
			d362 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d359.Imm.Int() * d361.Imm.Int())}
		} else if d359.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d361.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d359.Imm.Int()))
			ctx.EmitImulInt64(scratch, d361.Reg)
			d362 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d362)
		} else if d361.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d359.Reg)
			ctx.EmitMovRegReg(scratch, d359.Reg)
			if d361.Imm.Int() >= -2147483648 && d361.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d361.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d361.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d362 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d362)
		} else {
			r84 := ctx.AllocRegExcept(d359.Reg, d361.Reg)
			ctx.EmitMovRegReg(r84, d359.Reg)
			ctx.EmitImulInt64(r84, d361.Reg)
			d362 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r84}
			ctx.BindReg(r84, &d362)
		}
		if d362.Loc == scm.LocReg && d359.Loc == scm.LocReg && d362.Reg == d359.Reg {
			ctx.TransferReg(d359.Reg)
			d359.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d359)
		ctx.FreeDesc(&d361)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d362)
		var d363 scm.JITValueDesc
		if d362.Loc == scm.LocImm {
			d363 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d362.Imm.Int() / 64)}
		} else {
			r85 := ctx.AllocRegExcept(d362.Reg)
			ctx.EmitMovRegReg(r85, d362.Reg)
			ctx.EmitShrRegImm8(r85, 6)
			d363 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
			ctx.BindReg(r85, &d363)
		}
		if d363.Loc == scm.LocReg && d362.Loc == scm.LocReg && d363.Reg == d362.Reg {
			ctx.TransferReg(d362.Reg)
			d362.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d362)
		var d364 scm.JITValueDesc
		if d362.Loc == scm.LocImm {
			d364 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d362.Imm.Int() % 64)}
		} else {
			r86 := ctx.AllocRegExcept(d362.Reg)
			ctx.EmitMovRegReg(r86, d362.Reg)
			ctx.EmitAndRegImm32(r86, 63)
			d364 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
			ctx.BindReg(r86, &d364)
		}
		if d364.Loc == scm.LocReg && d362.Loc == scm.LocReg && d364.Reg == d362.Reg {
			ctx.TransferReg(d362.Reg)
			d362.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d362)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d365 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
			r87 := ctx.AllocReg()
			r88 := ctx.AllocRegExcept(r87)
			r89 := ctx.AllocRegExcept(r87, r88)
			ctx.EmitMovRegMem64(r87, fieldAddr)
			ctx.EmitMovRegMem64(r88, fieldAddr+8)
			ctx.EmitMovRegMem64(r89, fieldAddr+16)
			d365 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r87, Reg2: r88, Reg3: r89}
			ctx.BindReg(r87, &d365)
			ctx.BindReg(r88, &d365)
			ctx.BindReg(r89, &d365)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
			r90 := ctx.AllocReg()
			r91 := ctx.AllocRegExcept(r90)
			r92 := ctx.AllocRegExcept(r90, r91)
			ctx.EmitMovRegMem(r90, thisptr.Reg, off)
			ctx.EmitMovRegMem(r91, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r92, thisptr.Reg, off+16)
			d365 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r90, Reg2: r91, Reg3: r92}
			ctx.BindReg(r90, &d365)
			ctx.BindReg(r91, &d365)
			ctx.BindReg(r92, &d365)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d363)
		ctx.ReclaimUntrackedRegs()
		d367 = ctx.EmitSliceElementAddress(&d365, &d363, 8)
		ctx.EnsureDesc(&d367)
		ctx.EmitMovRegMem(d367.Reg, d367.Reg, 0)
		d366 = d367
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d366)
		ctx.EnsureDesc(&d364)
		var d368 scm.JITValueDesc
		if d366.Loc == scm.LocImm && d364.Loc == scm.LocImm {
			d368 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d366.Imm.Int()) << uint64(d364.Imm.Int())))}
		} else if d364.Loc == scm.LocImm {
			r93 := ctx.AllocRegExcept(d366.Reg)
			ctx.EmitMovRegReg(r93, d366.Reg)
			ctx.EmitShlRegImm8(r93, uint8(d364.Imm.Int()))
			d368 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r93}
			ctx.BindReg(r93, &d368)
		} else {
			{
				shiftSrc := d366.Reg
				r94 := ctx.AllocRegExcept(d366.Reg)
				ctx.EmitMovRegReg(r94, d366.Reg)
				shiftSrc = r94
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d364.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d364.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d364.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d368 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d368)
			}
		}
		if d368.Loc == scm.LocReg && d366.Loc == scm.LocReg && d368.Reg == d366.Reg {
			ctx.TransferReg(d366.Reg)
			d366.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d366)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d363)
		ctx.EnsureDesc(&d363)
		var d369 scm.JITValueDesc
		if d363.Loc == scm.LocImm {
			d369 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d363.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d363.Reg)
			ctx.EmitMovRegReg(scratch, d363.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d369 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d369)
		}
		if d369.Loc == scm.LocReg && d363.Loc == scm.LocReg && d369.Reg == d363.Reg {
			ctx.TransferReg(d363.Reg)
			d363.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d363)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d369)
		ctx.ReclaimUntrackedRegs()
		d371 = ctx.EmitSliceElementAddress(&d365, &d369, 8)
		ctx.EnsureDesc(&d371)
		ctx.EmitMovRegMem(d371.Reg, d371.Reg, 0)
		d370 = d371
		ctx.FreeDesc(&d369)
		ctx.ReclaimUntrackedRegs()
		d372 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d364)
		ctx.EnsureDescsTogether(&d372, &d364)
		var d373 scm.JITValueDesc
		if d372.Loc == scm.LocImm && d364.Loc == scm.LocImm {
			d373 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d372.Imm.Int() - d364.Imm.Int())}
		} else if d364.Loc == scm.LocImm && d364.Imm.Int() == 0 {
			r95 := ctx.AllocRegExcept(d372.Reg)
			ctx.EmitMovRegReg(r95, d372.Reg)
			d373 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r95}
			ctx.BindReg(r95, &d373)
		} else if d372.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d364.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d372.Imm.Int()))
			ctx.EmitSubInt64(scratch, d364.Reg)
			d373 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d373)
		} else if d364.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d372.Reg)
			ctx.EmitMovRegReg(scratch, d372.Reg)
			if d364.Imm.Int() >= -2147483648 && d364.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d364.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d364.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d373 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d373)
		} else {
			r96 := ctx.AllocRegExcept(d372.Reg, d364.Reg)
			ctx.EmitMovRegReg(r96, d372.Reg)
			ctx.EmitSubInt64(r96, d364.Reg)
			d373 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
			ctx.BindReg(r96, &d373)
		}
		if d373.Loc == scm.LocReg && d372.Loc == scm.LocReg && d373.Reg == d372.Reg {
			ctx.TransferReg(d372.Reg)
			d372.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d364)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d370)
		ctx.EnsureDesc(&d373)
		var d374 scm.JITValueDesc
		if d370.Loc == scm.LocImm && d373.Loc == scm.LocImm {
			d374 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d370.Imm.Int()) >> uint64(d373.Imm.Int())))}
		} else if d373.Loc == scm.LocImm {
			r97 := ctx.AllocRegExcept(d370.Reg)
			ctx.EmitMovRegReg(r97, d370.Reg)
			ctx.EmitShrRegImm8(r97, uint8(d373.Imm.Int()))
			d374 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
			ctx.BindReg(r97, &d374)
		} else {
			{
				shiftSrc := d370.Reg
				r98 := ctx.AllocRegExcept(d370.Reg)
				ctx.EmitMovRegReg(r98, d370.Reg)
				shiftSrc = r98
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d373.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d373.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d373.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d374 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d374)
			}
		}
		if d374.Loc == scm.LocReg && d370.Loc == scm.LocReg && d374.Reg == d370.Reg {
			ctx.TransferReg(d370.Reg)
			d370.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d370)
		ctx.FreeDesc(&d373)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d368)
		ctx.EnsureDesc(&d374)
		var d375 scm.JITValueDesc
		if d368.Loc == scm.LocImm && d374.Loc == scm.LocImm {
			d375 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d368.Imm.Int() | d374.Imm.Int())}
		} else if d368.Loc == scm.LocImm && d368.Imm.Int() == 0 {
			d375 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d374.Reg}
			ctx.BindReg(d374.Reg, &d375)
		} else if d374.Loc == scm.LocImm && d374.Imm.Int() == 0 {
			r99 := ctx.AllocRegExcept(d368.Reg)
			ctx.EmitMovRegReg(r99, d368.Reg)
			d375 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
			ctx.BindReg(r99, &d375)
		} else if d368.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d374.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d368.Imm.Int()))
			ctx.EmitOrInt64(scratch, d374.Reg)
			d375 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d375)
		} else if d374.Loc == scm.LocImm {
			r100 := ctx.AllocRegExcept(d368.Reg)
			ctx.EmitMovRegReg(r100, d368.Reg)
			if d374.Imm.Int() >= -2147483648 && d374.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r100, int32(d374.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d374.Imm.Int()))
				ctx.EmitOrInt64(r100, scm.RegR11)
			}
			d375 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r100}
			ctx.BindReg(r100, &d375)
		} else {
			r101 := ctx.AllocRegExcept(d368.Reg, d374.Reg)
			ctx.EmitMovRegReg(r101, d368.Reg)
			ctx.EmitOrInt64(r101, d374.Reg)
			d375 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r101}
			ctx.BindReg(r101, &d375)
		}
		if d375.Loc == scm.LocReg && d368.Loc == scm.LocReg && d375.Reg == d368.Reg {
			ctx.TransferReg(d368.Reg)
			d368.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d368)
		ctx.FreeDesc(&d374)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d360)
		ctx.EnsureDesc(&d360)
		var d376 scm.JITValueDesc
		if d360.Loc == scm.LocImm {
			d376 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d360.Imm.Int()))))}
		} else {
			r102 := ctx.AllocReg()
			ctx.EmitMovRegReg(r102, d360.Reg)
			ctx.EmitShlRegImm8(r102, 56)
			ctx.EmitShrRegImm8(r102, 56)
			d376 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r102}
			ctx.BindReg(r102, &d376)
		}
		ctx.ReclaimUntrackedRegs()
		d377 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d376)
		ctx.EnsureDescsTogether(&d377, &d376)
		var d378 scm.JITValueDesc
		if d377.Loc == scm.LocImm && d376.Loc == scm.LocImm {
			d378 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d377.Imm.Int() - d376.Imm.Int())}
		} else if d376.Loc == scm.LocImm && d376.Imm.Int() == 0 {
			r103 := ctx.AllocRegExcept(d377.Reg)
			ctx.EmitMovRegReg(r103, d377.Reg)
			d378 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r103}
			ctx.BindReg(r103, &d378)
		} else if d377.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d376.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d377.Imm.Int()))
			ctx.EmitSubInt64(scratch, d376.Reg)
			d378 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d378)
		} else if d376.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d377.Reg)
			ctx.EmitMovRegReg(scratch, d377.Reg)
			if d376.Imm.Int() >= -2147483648 && d376.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d376.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d376.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d378 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d378)
		} else {
			r104 := ctx.AllocRegExcept(d377.Reg, d376.Reg)
			ctx.EmitMovRegReg(r104, d377.Reg)
			ctx.EmitSubInt64(r104, d376.Reg)
			d378 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r104}
			ctx.BindReg(r104, &d378)
		}
		if d378.Loc == scm.LocReg && d377.Loc == scm.LocReg && d378.Reg == d377.Reg {
			ctx.TransferReg(d377.Reg)
			d377.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d376)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d375)
		ctx.EnsureDesc(&d378)
		var d379 scm.JITValueDesc
		if d375.Loc == scm.LocImm && d378.Loc == scm.LocImm {
			d379 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d375.Imm.Int()) >> uint64(d378.Imm.Int())))}
		} else if d378.Loc == scm.LocImm {
			r105 := ctx.AllocRegExcept(d375.Reg)
			ctx.EmitMovRegReg(r105, d375.Reg)
			ctx.EmitShrRegImm8(r105, uint8(d378.Imm.Int()))
			d379 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r105}
			ctx.BindReg(r105, &d379)
		} else {
			{
				shiftSrc := d375.Reg
				r106 := ctx.AllocRegExcept(d375.Reg)
				ctx.EmitMovRegReg(r106, d375.Reg)
				shiftSrc = r106
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d378.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d378.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d378.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d379 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d379)
			}
		}
		if d379.Loc == scm.LocReg && d375.Loc == scm.LocReg && d379.Reg == d375.Reg {
			ctx.TransferReg(d375.Reg)
			d375.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d375)
		ctx.FreeDesc(&d378)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d379)
		ctx.EnsureDesc(&d379)
		ctx.EnsureDesc(&d379)
		var d380 scm.JITValueDesc
		if d379.Loc == scm.LocImm {
			d380 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d379.Imm.Int()))))}
		} else {
			r107 := ctx.AllocReg()
			ctx.EmitMovRegReg(r107, d379.Reg)
			d380 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r107}
			ctx.BindReg(r107, &d380)
		}
		ctx.FreeDesc(&d379)
		ctx.EnsureDesc(&d380)
		ctx.EnsureDesc(&d45)
		ctx.EnsureDescsTogether(&d380, &d45)
		var d381 scm.JITValueDesc
		if d380.Loc == scm.LocImm && d45.Loc == scm.LocImm {
			d381 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d380.Imm.Int() + d45.Imm.Int())}
		} else if d45.Loc == scm.LocImm && d45.Imm.Int() == 0 {
			r108 := ctx.AllocRegExcept(d380.Reg)
			ctx.EmitMovRegReg(r108, d380.Reg)
			d381 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
			ctx.BindReg(r108, &d381)
		} else if d380.Loc == scm.LocImm && d380.Imm.Int() == 0 {
			d381 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d45.Reg}
			ctx.BindReg(d45.Reg, &d381)
		} else if d380.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d45.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d380.Imm.Int()))
			ctx.EmitAddInt64(scratch, d45.Reg)
			d381 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d381)
		} else if d45.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d380.Reg)
			ctx.EmitMovRegReg(scratch, d380.Reg)
			if d45.Imm.Int() >= -2147483648 && d45.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d45.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d45.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d381 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d381)
		} else {
			r109 := ctx.AllocRegExcept(d380.Reg, d45.Reg)
			ctx.EmitMovRegReg(r109, d380.Reg)
			ctx.EmitAddInt64(r109, d45.Reg)
			d381 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r109}
			ctx.BindReg(r109, &d381)
		}
		if d381.Loc == scm.LocReg && d380.Loc == scm.LocReg && d381.Reg == d380.Reg {
			ctx.TransferReg(d380.Reg)
			d380.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d380)
		ctx.EnsureDesc(&d381)
		ctx.EnsureDesc(&d381)
		var d382 scm.JITValueDesc
		if d381.Loc == scm.LocImm {
			d382 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d381.Imm.Int()))))}
		} else {
			r110 := ctx.AllocReg()
			ctx.EmitMovRegReg(r110, d381.Reg)
			ctx.EmitShlRegImm8(r110, 32)
			ctx.EmitShrRegImm8(r110, 32)
			d382 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
			ctx.BindReg(r110, &d382)
		}
		ctx.FreeDesc(&d381)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d382)
		ctx.EnsureDescsTogether(&idxInt, &d382)
		var d383 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm && d382.Loc == scm.LocImm {
			d383 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d382.Imm.Int()))}
		} else if d382.Loc == scm.LocImm {
			r111 := ctx.AllocRegExcept(idxInt.Reg)
			if d382.Imm.Int() >= -2147483648 && d382.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(idxInt.Reg, int32(d382.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d382.Imm.Int()))
				ctx.EmitCmpInt64(idxInt.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r111, scm.CondUnsignedBelow)
			d383 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r111}
			ctx.BindReg(r111, &d383)
		} else if idxInt.Loc == scm.LocImm {
			r112 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d382.Reg)
			ctx.EmitSetcc(r112, scm.CondUnsignedBelow)
			d383 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r112}
			ctx.BindReg(r112, &d383)
		} else {
			r113 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitCmpInt64(idxInt.Reg, d382.Reg)
			ctx.EmitSetcc(r113, scm.CondUnsignedBelow)
			d383 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r113}
			ctx.BindReg(r113, &d383)
		}
		ctx.FreeDesc(&d382)
		d384 = d383
		ctx.EnsureDesc(&d384)
		if d384.Loc != scm.LocImm && d384.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d384.Loc == scm.LocImm {
			if d384.Imm.Bool() {
				if ps.General {
				}
				ps385 := scm.PhiState{General: ps.General}
				ps385.OverlayValues = make([]scm.JITValueDesc, 385)
				ps385.OverlayValues[1] = d1
				ps385.OverlayValues[2] = d2
				ps385.OverlayValues[3] = d3
				ps385.OverlayValues[4] = d4
				ps385.OverlayValues[5] = d5
				ps385.OverlayValues[6] = d6
				ps385.OverlayValues[7] = d7
				ps385.OverlayValues[8] = d8
				ps385.OverlayValues[9] = d9
				ps385.OverlayValues[10] = d10
				ps385.OverlayValues[11] = d11
				ps385.OverlayValues[12] = d12
				ps385.OverlayValues[13] = d13
				ps385.OverlayValues[14] = d14
				ps385.OverlayValues[15] = d15
				ps385.OverlayValues[17] = d17
				ps385.OverlayValues[18] = d18
				ps385.OverlayValues[19] = d19
				ps385.OverlayValues[20] = d20
				ps385.OverlayValues[21] = d21
				ps385.OverlayValues[22] = d22
				ps385.OverlayValues[23] = d23
				ps385.OverlayValues[24] = d24
				ps385.OverlayValues[25] = d25
				ps385.OverlayValues[26] = d26
				ps385.OverlayValues[27] = d27
				ps385.OverlayValues[28] = d28
				ps385.OverlayValues[29] = d29
				ps385.OverlayValues[30] = d30
				ps385.OverlayValues[31] = d31
				ps385.OverlayValues[32] = d32
				ps385.OverlayValues[33] = d33
				ps385.OverlayValues[34] = d34
				ps385.OverlayValues[35] = d35
				ps385.OverlayValues[36] = d36
				ps385.OverlayValues[37] = d37
				ps385.OverlayValues[38] = d38
				ps385.OverlayValues[39] = d39
				ps385.OverlayValues[40] = d40
				ps385.OverlayValues[41] = d41
				ps385.OverlayValues[42] = d42
				ps385.OverlayValues[43] = d43
				ps385.OverlayValues[44] = d44
				ps385.OverlayValues[45] = d45
				ps385.OverlayValues[46] = d46
				ps385.OverlayValues[47] = d47
				ps385.OverlayValues[48] = d48
				ps385.OverlayValues[49] = d49
				ps385.OverlayValues[52] = d52
				ps385.OverlayValues[53] = d53
				ps385.OverlayValues[54] = d54
				ps385.OverlayValues[109] = d109
				ps385.OverlayValues[110] = d110
				ps385.OverlayValues[111] = d111
				ps385.OverlayValues[112] = d112
				ps385.OverlayValues[113] = d113
				ps385.OverlayValues[114] = d114
				ps385.OverlayValues[115] = d115
				ps385.OverlayValues[116] = d116
				ps385.OverlayValues[117] = d117
				ps385.OverlayValues[118] = d118
				ps385.OverlayValues[119] = d119
				ps385.OverlayValues[120] = d120
				ps385.OverlayValues[121] = d121
				ps385.OverlayValues[122] = d122
				ps385.OverlayValues[123] = d123
				ps385.OverlayValues[124] = d124
				ps385.OverlayValues[125] = d125
				ps385.OverlayValues[126] = d126
				ps385.OverlayValues[127] = d127
				ps385.OverlayValues[128] = d128
				ps385.OverlayValues[129] = d129
				ps385.OverlayValues[130] = d130
				ps385.OverlayValues[131] = d131
				ps385.OverlayValues[132] = d132
				ps385.OverlayValues[133] = d133
				ps385.OverlayValues[134] = d134
				ps385.OverlayValues[135] = d135
				ps385.OverlayValues[136] = d136
				ps385.OverlayValues[137] = d137
				ps385.OverlayValues[140] = d140
				ps385.OverlayValues[225] = d225
				ps385.OverlayValues[226] = d226
				ps385.OverlayValues[227] = d227
				ps385.OverlayValues[228] = d228
				ps385.OverlayValues[230] = d230
				ps385.OverlayValues[231] = d231
				ps385.OverlayValues[232] = d232
				ps385.OverlayValues[233] = d233
				ps385.OverlayValues[234] = d234
				ps385.OverlayValues[235] = d235
				ps385.OverlayValues[236] = d236
				ps385.OverlayValues[237] = d237
				ps385.OverlayValues[239] = d239
				ps385.OverlayValues[241] = d241
				ps385.OverlayValues[242] = d242
				ps385.OverlayValues[243] = d243
				ps385.OverlayValues[244] = d244
				ps385.OverlayValues[245] = d245
				ps385.OverlayValues[248] = d248
				ps385.OverlayValues[350] = d350
				ps385.OverlayValues[351] = d351
				ps385.OverlayValues[352] = d352
				ps385.OverlayValues[353] = d353
				ps385.OverlayValues[354] = d354
				ps385.OverlayValues[356] = d356
				ps385.OverlayValues[357] = d357
				ps385.OverlayValues[358] = d358
				ps385.OverlayValues[359] = d359
				ps385.OverlayValues[360] = d360
				ps385.OverlayValues[361] = d361
				ps385.OverlayValues[362] = d362
				ps385.OverlayValues[363] = d363
				ps385.OverlayValues[364] = d364
				ps385.OverlayValues[365] = d365
				ps385.OverlayValues[366] = d366
				ps385.OverlayValues[367] = d367
				ps385.OverlayValues[368] = d368
				ps385.OverlayValues[369] = d369
				ps385.OverlayValues[370] = d370
				ps385.OverlayValues[371] = d371
				ps385.OverlayValues[372] = d372
				ps385.OverlayValues[373] = d373
				ps385.OverlayValues[374] = d374
				ps385.OverlayValues[375] = d375
				ps385.OverlayValues[376] = d376
				ps385.OverlayValues[377] = d377
				ps385.OverlayValues[378] = d378
				ps385.OverlayValues[379] = d379
				ps385.OverlayValues[380] = d380
				ps385.OverlayValues[381] = d381
				ps385.OverlayValues[382] = d382
				ps385.OverlayValues[383] = d383
				ps385.OverlayValues[384] = d384
				return bbs[7].RenderPS(ps385)
			}
			if ps.General {
			}
			ps386 := scm.PhiState{General: ps.General}
			ps386.OverlayValues = make([]scm.JITValueDesc, 385)
			ps386.OverlayValues[1] = d1
			ps386.OverlayValues[2] = d2
			ps386.OverlayValues[3] = d3
			ps386.OverlayValues[4] = d4
			ps386.OverlayValues[5] = d5
			ps386.OverlayValues[6] = d6
			ps386.OverlayValues[7] = d7
			ps386.OverlayValues[8] = d8
			ps386.OverlayValues[9] = d9
			ps386.OverlayValues[10] = d10
			ps386.OverlayValues[11] = d11
			ps386.OverlayValues[12] = d12
			ps386.OverlayValues[13] = d13
			ps386.OverlayValues[14] = d14
			ps386.OverlayValues[15] = d15
			ps386.OverlayValues[17] = d17
			ps386.OverlayValues[18] = d18
			ps386.OverlayValues[19] = d19
			ps386.OverlayValues[20] = d20
			ps386.OverlayValues[21] = d21
			ps386.OverlayValues[22] = d22
			ps386.OverlayValues[23] = d23
			ps386.OverlayValues[24] = d24
			ps386.OverlayValues[25] = d25
			ps386.OverlayValues[26] = d26
			ps386.OverlayValues[27] = d27
			ps386.OverlayValues[28] = d28
			ps386.OverlayValues[29] = d29
			ps386.OverlayValues[30] = d30
			ps386.OverlayValues[31] = d31
			ps386.OverlayValues[32] = d32
			ps386.OverlayValues[33] = d33
			ps386.OverlayValues[34] = d34
			ps386.OverlayValues[35] = d35
			ps386.OverlayValues[36] = d36
			ps386.OverlayValues[37] = d37
			ps386.OverlayValues[38] = d38
			ps386.OverlayValues[39] = d39
			ps386.OverlayValues[40] = d40
			ps386.OverlayValues[41] = d41
			ps386.OverlayValues[42] = d42
			ps386.OverlayValues[43] = d43
			ps386.OverlayValues[44] = d44
			ps386.OverlayValues[45] = d45
			ps386.OverlayValues[46] = d46
			ps386.OverlayValues[47] = d47
			ps386.OverlayValues[48] = d48
			ps386.OverlayValues[49] = d49
			ps386.OverlayValues[52] = d52
			ps386.OverlayValues[53] = d53
			ps386.OverlayValues[54] = d54
			ps386.OverlayValues[109] = d109
			ps386.OverlayValues[110] = d110
			ps386.OverlayValues[111] = d111
			ps386.OverlayValues[112] = d112
			ps386.OverlayValues[113] = d113
			ps386.OverlayValues[114] = d114
			ps386.OverlayValues[115] = d115
			ps386.OverlayValues[116] = d116
			ps386.OverlayValues[117] = d117
			ps386.OverlayValues[118] = d118
			ps386.OverlayValues[119] = d119
			ps386.OverlayValues[120] = d120
			ps386.OverlayValues[121] = d121
			ps386.OverlayValues[122] = d122
			ps386.OverlayValues[123] = d123
			ps386.OverlayValues[124] = d124
			ps386.OverlayValues[125] = d125
			ps386.OverlayValues[126] = d126
			ps386.OverlayValues[127] = d127
			ps386.OverlayValues[128] = d128
			ps386.OverlayValues[129] = d129
			ps386.OverlayValues[130] = d130
			ps386.OverlayValues[131] = d131
			ps386.OverlayValues[132] = d132
			ps386.OverlayValues[133] = d133
			ps386.OverlayValues[134] = d134
			ps386.OverlayValues[135] = d135
			ps386.OverlayValues[136] = d136
			ps386.OverlayValues[137] = d137
			ps386.OverlayValues[140] = d140
			ps386.OverlayValues[225] = d225
			ps386.OverlayValues[226] = d226
			ps386.OverlayValues[227] = d227
			ps386.OverlayValues[228] = d228
			ps386.OverlayValues[230] = d230
			ps386.OverlayValues[231] = d231
			ps386.OverlayValues[232] = d232
			ps386.OverlayValues[233] = d233
			ps386.OverlayValues[234] = d234
			ps386.OverlayValues[235] = d235
			ps386.OverlayValues[236] = d236
			ps386.OverlayValues[237] = d237
			ps386.OverlayValues[239] = d239
			ps386.OverlayValues[241] = d241
			ps386.OverlayValues[242] = d242
			ps386.OverlayValues[243] = d243
			ps386.OverlayValues[244] = d244
			ps386.OverlayValues[245] = d245
			ps386.OverlayValues[248] = d248
			ps386.OverlayValues[350] = d350
			ps386.OverlayValues[351] = d351
			ps386.OverlayValues[352] = d352
			ps386.OverlayValues[353] = d353
			ps386.OverlayValues[354] = d354
			ps386.OverlayValues[356] = d356
			ps386.OverlayValues[357] = d357
			ps386.OverlayValues[358] = d358
			ps386.OverlayValues[359] = d359
			ps386.OverlayValues[360] = d360
			ps386.OverlayValues[361] = d361
			ps386.OverlayValues[362] = d362
			ps386.OverlayValues[363] = d363
			ps386.OverlayValues[364] = d364
			ps386.OverlayValues[365] = d365
			ps386.OverlayValues[366] = d366
			ps386.OverlayValues[367] = d367
			ps386.OverlayValues[368] = d368
			ps386.OverlayValues[369] = d369
			ps386.OverlayValues[370] = d370
			ps386.OverlayValues[371] = d371
			ps386.OverlayValues[372] = d372
			ps386.OverlayValues[373] = d373
			ps386.OverlayValues[374] = d374
			ps386.OverlayValues[375] = d375
			ps386.OverlayValues[376] = d376
			ps386.OverlayValues[377] = d377
			ps386.OverlayValues[378] = d378
			ps386.OverlayValues[379] = d379
			ps386.OverlayValues[380] = d380
			ps386.OverlayValues[381] = d381
			ps386.OverlayValues[382] = d382
			ps386.OverlayValues[383] = d383
			ps386.OverlayValues[384] = d384
			return bbs[9].RenderPS(ps386)
		}
		if !ps.General {
			ps.General = true
			return bbs[6].RenderPS(ps)
		}
		lbl24 := ctx.ReserveLabel()
		lbl25 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d384.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl24)
		ctx.EmitJmp(lbl25)
		ctx.MarkLabel(lbl24)
		ctx.EmitJmp(lbl8)
		ctx.MarkLabel(lbl25)
		ctx.EmitJmp(lbl10)
		ps387 := scm.PhiState{General: true}
		ps387.OverlayValues = make([]scm.JITValueDesc, 385)
		ps387.OverlayValues[1] = d1
		ps387.OverlayValues[2] = d2
		ps387.OverlayValues[3] = d3
		ps387.OverlayValues[4] = d4
		ps387.OverlayValues[5] = d5
		ps387.OverlayValues[6] = d6
		ps387.OverlayValues[7] = d7
		ps387.OverlayValues[8] = d8
		ps387.OverlayValues[9] = d9
		ps387.OverlayValues[10] = d10
		ps387.OverlayValues[11] = d11
		ps387.OverlayValues[12] = d12
		ps387.OverlayValues[13] = d13
		ps387.OverlayValues[14] = d14
		ps387.OverlayValues[15] = d15
		ps387.OverlayValues[17] = d17
		ps387.OverlayValues[18] = d18
		ps387.OverlayValues[19] = d19
		ps387.OverlayValues[20] = d20
		ps387.OverlayValues[21] = d21
		ps387.OverlayValues[22] = d22
		ps387.OverlayValues[23] = d23
		ps387.OverlayValues[24] = d24
		ps387.OverlayValues[25] = d25
		ps387.OverlayValues[26] = d26
		ps387.OverlayValues[27] = d27
		ps387.OverlayValues[28] = d28
		ps387.OverlayValues[29] = d29
		ps387.OverlayValues[30] = d30
		ps387.OverlayValues[31] = d31
		ps387.OverlayValues[32] = d32
		ps387.OverlayValues[33] = d33
		ps387.OverlayValues[34] = d34
		ps387.OverlayValues[35] = d35
		ps387.OverlayValues[36] = d36
		ps387.OverlayValues[37] = d37
		ps387.OverlayValues[38] = d38
		ps387.OverlayValues[39] = d39
		ps387.OverlayValues[40] = d40
		ps387.OverlayValues[41] = d41
		ps387.OverlayValues[42] = d42
		ps387.OverlayValues[43] = d43
		ps387.OverlayValues[44] = d44
		ps387.OverlayValues[45] = d45
		ps387.OverlayValues[46] = d46
		ps387.OverlayValues[47] = d47
		ps387.OverlayValues[48] = d48
		ps387.OverlayValues[49] = d49
		ps387.OverlayValues[52] = d52
		ps387.OverlayValues[53] = d53
		ps387.OverlayValues[54] = d54
		ps387.OverlayValues[109] = d109
		ps387.OverlayValues[110] = d110
		ps387.OverlayValues[111] = d111
		ps387.OverlayValues[112] = d112
		ps387.OverlayValues[113] = d113
		ps387.OverlayValues[114] = d114
		ps387.OverlayValues[115] = d115
		ps387.OverlayValues[116] = d116
		ps387.OverlayValues[117] = d117
		ps387.OverlayValues[118] = d118
		ps387.OverlayValues[119] = d119
		ps387.OverlayValues[120] = d120
		ps387.OverlayValues[121] = d121
		ps387.OverlayValues[122] = d122
		ps387.OverlayValues[123] = d123
		ps387.OverlayValues[124] = d124
		ps387.OverlayValues[125] = d125
		ps387.OverlayValues[126] = d126
		ps387.OverlayValues[127] = d127
		ps387.OverlayValues[128] = d128
		ps387.OverlayValues[129] = d129
		ps387.OverlayValues[130] = d130
		ps387.OverlayValues[131] = d131
		ps387.OverlayValues[132] = d132
		ps387.OverlayValues[133] = d133
		ps387.OverlayValues[134] = d134
		ps387.OverlayValues[135] = d135
		ps387.OverlayValues[136] = d136
		ps387.OverlayValues[137] = d137
		ps387.OverlayValues[140] = d140
		ps387.OverlayValues[225] = d225
		ps387.OverlayValues[226] = d226
		ps387.OverlayValues[227] = d227
		ps387.OverlayValues[228] = d228
		ps387.OverlayValues[230] = d230
		ps387.OverlayValues[231] = d231
		ps387.OverlayValues[232] = d232
		ps387.OverlayValues[233] = d233
		ps387.OverlayValues[234] = d234
		ps387.OverlayValues[235] = d235
		ps387.OverlayValues[236] = d236
		ps387.OverlayValues[237] = d237
		ps387.OverlayValues[239] = d239
		ps387.OverlayValues[241] = d241
		ps387.OverlayValues[242] = d242
		ps387.OverlayValues[243] = d243
		ps387.OverlayValues[244] = d244
		ps387.OverlayValues[245] = d245
		ps387.OverlayValues[248] = d248
		ps387.OverlayValues[350] = d350
		ps387.OverlayValues[351] = d351
		ps387.OverlayValues[352] = d352
		ps387.OverlayValues[353] = d353
		ps387.OverlayValues[354] = d354
		ps387.OverlayValues[356] = d356
		ps387.OverlayValues[357] = d357
		ps387.OverlayValues[358] = d358
		ps387.OverlayValues[359] = d359
		ps387.OverlayValues[360] = d360
		ps387.OverlayValues[361] = d361
		ps387.OverlayValues[362] = d362
		ps387.OverlayValues[363] = d363
		ps387.OverlayValues[364] = d364
		ps387.OverlayValues[365] = d365
		ps387.OverlayValues[366] = d366
		ps387.OverlayValues[367] = d367
		ps387.OverlayValues[368] = d368
		ps387.OverlayValues[369] = d369
		ps387.OverlayValues[370] = d370
		ps387.OverlayValues[371] = d371
		ps387.OverlayValues[372] = d372
		ps387.OverlayValues[373] = d373
		ps387.OverlayValues[374] = d374
		ps387.OverlayValues[375] = d375
		ps387.OverlayValues[376] = d376
		ps387.OverlayValues[377] = d377
		ps387.OverlayValues[378] = d378
		ps387.OverlayValues[379] = d379
		ps387.OverlayValues[380] = d380
		ps387.OverlayValues[381] = d381
		ps387.OverlayValues[382] = d382
		ps387.OverlayValues[383] = d383
		ps387.OverlayValues[384] = d384
		ps388 := scm.PhiState{General: true}
		ps388.OverlayValues = make([]scm.JITValueDesc, 385)
		ps388.OverlayValues[1] = d1
		ps388.OverlayValues[2] = d2
		ps388.OverlayValues[3] = d3
		ps388.OverlayValues[4] = d4
		ps388.OverlayValues[5] = d5
		ps388.OverlayValues[6] = d6
		ps388.OverlayValues[7] = d7
		ps388.OverlayValues[8] = d8
		ps388.OverlayValues[9] = d9
		ps388.OverlayValues[10] = d10
		ps388.OverlayValues[11] = d11
		ps388.OverlayValues[12] = d12
		ps388.OverlayValues[13] = d13
		ps388.OverlayValues[14] = d14
		ps388.OverlayValues[15] = d15
		ps388.OverlayValues[17] = d17
		ps388.OverlayValues[18] = d18
		ps388.OverlayValues[19] = d19
		ps388.OverlayValues[20] = d20
		ps388.OverlayValues[21] = d21
		ps388.OverlayValues[22] = d22
		ps388.OverlayValues[23] = d23
		ps388.OverlayValues[24] = d24
		ps388.OverlayValues[25] = d25
		ps388.OverlayValues[26] = d26
		ps388.OverlayValues[27] = d27
		ps388.OverlayValues[28] = d28
		ps388.OverlayValues[29] = d29
		ps388.OverlayValues[30] = d30
		ps388.OverlayValues[31] = d31
		ps388.OverlayValues[32] = d32
		ps388.OverlayValues[33] = d33
		ps388.OverlayValues[34] = d34
		ps388.OverlayValues[35] = d35
		ps388.OverlayValues[36] = d36
		ps388.OverlayValues[37] = d37
		ps388.OverlayValues[38] = d38
		ps388.OverlayValues[39] = d39
		ps388.OverlayValues[40] = d40
		ps388.OverlayValues[41] = d41
		ps388.OverlayValues[42] = d42
		ps388.OverlayValues[43] = d43
		ps388.OverlayValues[44] = d44
		ps388.OverlayValues[45] = d45
		ps388.OverlayValues[46] = d46
		ps388.OverlayValues[47] = d47
		ps388.OverlayValues[48] = d48
		ps388.OverlayValues[49] = d49
		ps388.OverlayValues[52] = d52
		ps388.OverlayValues[53] = d53
		ps388.OverlayValues[54] = d54
		ps388.OverlayValues[109] = d109
		ps388.OverlayValues[110] = d110
		ps388.OverlayValues[111] = d111
		ps388.OverlayValues[112] = d112
		ps388.OverlayValues[113] = d113
		ps388.OverlayValues[114] = d114
		ps388.OverlayValues[115] = d115
		ps388.OverlayValues[116] = d116
		ps388.OverlayValues[117] = d117
		ps388.OverlayValues[118] = d118
		ps388.OverlayValues[119] = d119
		ps388.OverlayValues[120] = d120
		ps388.OverlayValues[121] = d121
		ps388.OverlayValues[122] = d122
		ps388.OverlayValues[123] = d123
		ps388.OverlayValues[124] = d124
		ps388.OverlayValues[125] = d125
		ps388.OverlayValues[126] = d126
		ps388.OverlayValues[127] = d127
		ps388.OverlayValues[128] = d128
		ps388.OverlayValues[129] = d129
		ps388.OverlayValues[130] = d130
		ps388.OverlayValues[131] = d131
		ps388.OverlayValues[132] = d132
		ps388.OverlayValues[133] = d133
		ps388.OverlayValues[134] = d134
		ps388.OverlayValues[135] = d135
		ps388.OverlayValues[136] = d136
		ps388.OverlayValues[137] = d137
		ps388.OverlayValues[140] = d140
		ps388.OverlayValues[225] = d225
		ps388.OverlayValues[226] = d226
		ps388.OverlayValues[227] = d227
		ps388.OverlayValues[228] = d228
		ps388.OverlayValues[230] = d230
		ps388.OverlayValues[231] = d231
		ps388.OverlayValues[232] = d232
		ps388.OverlayValues[233] = d233
		ps388.OverlayValues[234] = d234
		ps388.OverlayValues[235] = d235
		ps388.OverlayValues[236] = d236
		ps388.OverlayValues[237] = d237
		ps388.OverlayValues[239] = d239
		ps388.OverlayValues[241] = d241
		ps388.OverlayValues[242] = d242
		ps388.OverlayValues[243] = d243
		ps388.OverlayValues[244] = d244
		ps388.OverlayValues[245] = d245
		ps388.OverlayValues[248] = d248
		ps388.OverlayValues[350] = d350
		ps388.OverlayValues[351] = d351
		ps388.OverlayValues[352] = d352
		ps388.OverlayValues[353] = d353
		ps388.OverlayValues[354] = d354
		ps388.OverlayValues[356] = d356
		ps388.OverlayValues[357] = d357
		ps388.OverlayValues[358] = d358
		ps388.OverlayValues[359] = d359
		ps388.OverlayValues[360] = d360
		ps388.OverlayValues[361] = d361
		ps388.OverlayValues[362] = d362
		ps388.OverlayValues[363] = d363
		ps388.OverlayValues[364] = d364
		ps388.OverlayValues[365] = d365
		ps388.OverlayValues[366] = d366
		ps388.OverlayValues[367] = d367
		ps388.OverlayValues[368] = d368
		ps388.OverlayValues[369] = d369
		ps388.OverlayValues[370] = d370
		ps388.OverlayValues[371] = d371
		ps388.OverlayValues[372] = d372
		ps388.OverlayValues[373] = d373
		ps388.OverlayValues[374] = d374
		ps388.OverlayValues[375] = d375
		ps388.OverlayValues[376] = d376
		ps388.OverlayValues[377] = d377
		ps388.OverlayValues[378] = d378
		ps388.OverlayValues[379] = d379
		ps388.OverlayValues[380] = d380
		ps388.OverlayValues[381] = d381
		ps388.OverlayValues[382] = d382
		ps388.OverlayValues[383] = d383
		ps388.OverlayValues[384] = d384
		snap389 := d1
		snap390 := d2
		snap391 := d3
		snap392 := d4
		snap393 := d5
		snap394 := d6
		snap395 := d7
		snap396 := d8
		snap397 := d9
		snap398 := d10
		snap399 := d11
		snap400 := d12
		snap401 := d13
		snap402 := d14
		snap403 := d15
		snap404 := d17
		snap405 := d18
		snap406 := d19
		snap407 := d20
		snap408 := d21
		snap409 := d22
		snap410 := d23
		snap411 := d24
		snap412 := d25
		snap413 := d26
		snap414 := d27
		snap415 := d28
		snap416 := d29
		snap417 := d30
		snap418 := d31
		snap419 := d32
		snap420 := d33
		snap421 := d34
		snap422 := d35
		snap423 := d36
		snap424 := d37
		snap425 := d38
		snap426 := d39
		snap427 := d40
		snap428 := d41
		snap429 := d42
		snap430 := d43
		snap431 := d44
		snap432 := d45
		snap433 := d46
		snap434 := d47
		snap435 := d48
		snap436 := d49
		snap437 := d52
		snap438 := d53
		snap439 := d54
		snap440 := d109
		snap441 := d110
		snap442 := d111
		snap443 := d112
		snap444 := d113
		snap445 := d114
		snap446 := d115
		snap447 := d116
		snap448 := d117
		snap449 := d118
		snap450 := d119
		snap451 := d120
		snap452 := d121
		snap453 := d122
		snap454 := d123
		snap455 := d124
		snap456 := d125
		snap457 := d126
		snap458 := d127
		snap459 := d128
		snap460 := d129
		snap461 := d130
		snap462 := d131
		snap463 := d132
		snap464 := d133
		snap465 := d134
		snap466 := d135
		snap467 := d136
		snap468 := d137
		snap469 := d140
		snap470 := d225
		snap471 := d226
		snap472 := d227
		snap473 := d228
		snap474 := d230
		snap475 := d231
		snap476 := d232
		snap477 := d233
		snap478 := d234
		snap479 := d235
		snap480 := d236
		snap481 := d237
		snap482 := d239
		snap483 := d241
		snap484 := d242
		snap485 := d243
		snap486 := d244
		snap487 := d245
		snap488 := d248
		snap489 := d350
		snap490 := d351
		snap491 := d352
		snap492 := d353
		snap493 := d354
		snap494 := d356
		snap495 := d357
		snap496 := d358
		snap497 := d359
		snap498 := d360
		snap499 := d361
		snap500 := d362
		snap501 := d363
		snap502 := d364
		snap503 := d365
		snap504 := d366
		snap505 := d367
		snap506 := d368
		snap507 := d369
		snap508 := d370
		snap509 := d371
		snap510 := d372
		snap511 := d373
		snap512 := d374
		snap513 := d375
		snap514 := d376
		snap515 := d377
		snap516 := d378
		snap517 := d379
		snap518 := d380
		snap519 := d381
		snap520 := d382
		snap521 := d383
		snap522 := d384
		alloc523 := ctx.SnapshotAllocState()
		if !bbs[9].Rendered {
			bbs[9].RenderPS(ps388)
		}
		ctx.RestoreAllocState(alloc523)
		d1 = snap389
		d2 = snap390
		d3 = snap391
		d4 = snap392
		d5 = snap393
		d6 = snap394
		d7 = snap395
		d8 = snap396
		d9 = snap397
		d10 = snap398
		d11 = snap399
		d12 = snap400
		d13 = snap401
		d14 = snap402
		d15 = snap403
		d17 = snap404
		d18 = snap405
		d19 = snap406
		d20 = snap407
		d21 = snap408
		d22 = snap409
		d23 = snap410
		d24 = snap411
		d25 = snap412
		d26 = snap413
		d27 = snap414
		d28 = snap415
		d29 = snap416
		d30 = snap417
		d31 = snap418
		d32 = snap419
		d33 = snap420
		d34 = snap421
		d35 = snap422
		d36 = snap423
		d37 = snap424
		d38 = snap425
		d39 = snap426
		d40 = snap427
		d41 = snap428
		d42 = snap429
		d43 = snap430
		d44 = snap431
		d45 = snap432
		d46 = snap433
		d47 = snap434
		d48 = snap435
		d49 = snap436
		d52 = snap437
		d53 = snap438
		d54 = snap439
		d109 = snap440
		d110 = snap441
		d111 = snap442
		d112 = snap443
		d113 = snap444
		d114 = snap445
		d115 = snap446
		d116 = snap447
		d117 = snap448
		d118 = snap449
		d119 = snap450
		d120 = snap451
		d121 = snap452
		d122 = snap453
		d123 = snap454
		d124 = snap455
		d125 = snap456
		d126 = snap457
		d127 = snap458
		d128 = snap459
		d129 = snap460
		d130 = snap461
		d131 = snap462
		d132 = snap463
		d133 = snap464
		d134 = snap465
		d135 = snap466
		d136 = snap467
		d137 = snap468
		d140 = snap469
		d225 = snap470
		d226 = snap471
		d227 = snap472
		d228 = snap473
		d230 = snap474
		d231 = snap475
		d232 = snap476
		d233 = snap477
		d234 = snap478
		d235 = snap479
		d236 = snap480
		d237 = snap481
		d239 = snap482
		d241 = snap483
		d242 = snap484
		d243 = snap485
		d244 = snap486
		d245 = snap487
		d248 = snap488
		d350 = snap489
		d351 = snap490
		d352 = snap491
		d353 = snap492
		d354 = snap493
		d356 = snap494
		d357 = snap495
		d358 = snap496
		d359 = snap497
		d360 = snap498
		d361 = snap499
		d362 = snap500
		d363 = snap501
		d364 = snap502
		d365 = snap503
		d366 = snap504
		d367 = snap505
		d368 = snap506
		d369 = snap507
		d370 = snap508
		d371 = snap509
		d372 = snap510
		d373 = snap511
		d374 = snap512
		d375 = snap513
		d376 = snap514
		d377 = snap515
		d378 = snap516
		d379 = snap517
		d380 = snap518
		d381 = snap519
		d382 = snap520
		d383 = snap521
		d384 = snap522
		if !bbs[7].Rendered {
			return bbs[7].RenderPS(ps387)
		}
		return result
		ctx.FreeDesc(&d383)
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
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
			d52 = ps.OverlayValues[52]
		}
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != scm.LocNone {
			d109 = ps.OverlayValues[109]
		}
		if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != scm.LocNone {
			d110 = ps.OverlayValues[110]
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
		if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != scm.LocNone {
			d140 = ps.OverlayValues[140]
		}
		if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != scm.LocNone {
			d225 = ps.OverlayValues[225]
		}
		if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != scm.LocNone {
			d226 = ps.OverlayValues[226]
		}
		if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
			d227 = ps.OverlayValues[227]
		}
		if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
			d228 = ps.OverlayValues[228]
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
		if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != scm.LocNone {
			d234 = ps.OverlayValues[234]
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
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
		}
		if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != scm.LocNone {
			d241 = ps.OverlayValues[241]
		}
		if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != scm.LocNone {
			d242 = ps.OverlayValues[242]
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
		if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != scm.LocNone {
			d248 = ps.OverlayValues[248]
		}
		if len(ps.OverlayValues) > 350 && ps.OverlayValues[350].Loc != scm.LocNone {
			d350 = ps.OverlayValues[350]
		}
		if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != scm.LocNone {
			d351 = ps.OverlayValues[351]
		}
		if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != scm.LocNone {
			d352 = ps.OverlayValues[352]
		}
		if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != scm.LocNone {
			d353 = ps.OverlayValues[353]
		}
		if len(ps.OverlayValues) > 354 && ps.OverlayValues[354].Loc != scm.LocNone {
			d354 = ps.OverlayValues[354]
		}
		if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != scm.LocNone {
			d356 = ps.OverlayValues[356]
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
		if len(ps.OverlayValues) > 362 && ps.OverlayValues[362].Loc != scm.LocNone {
			d362 = ps.OverlayValues[362]
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d5)
		var d524 scm.JITValueDesc
		if d5.Loc == scm.LocImm {
			d524 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d524 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d524)
		}
		if d524.Loc == scm.LocImm {
			d524 = scm.JITValueDesc{Loc: scm.LocImm, Type: d524.Type, Imm: scm.NewInt(int64(uint64(d524.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d524.Reg, 32)
			ctx.EmitShrRegImm8(d524.Reg, 32)
		}
		if d524.Loc == scm.LocReg && d5.Loc == scm.LocReg && d524.Reg == d5.Reg {
			ctx.TransferReg(d5.Reg)
			d5.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d524)
		ctx.EmitStoreToStack(d524, int32(bbs[8].PhiBase)+int32(16))
		ctx.StabilizeDescForControlFlow(&d524)
		if ps.General {
			ctx.SyncDesc(&d6)
			if d6.Loc == scm.LocReg {
				ctx.ProtectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.ProtectReg(d6.Reg)
				ctx.ProtectReg(d6.Reg2)
			}
			d525 = d6
			if d525.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d525)
			d526 = d525
			if d526.Loc == scm.LocImm {
				d526 = scm.JITValueDesc{Loc: scm.LocImm, Type: d526.Type, Imm: scm.NewInt(int64(uint64(d526.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d526.Reg, 32)
				ctx.EmitShrRegImm8(d526.Reg, 32)
			}
			ctx.EmitStoreToStack(d526, int32(bbs[8].PhiBase)+int32(0))
			if d6.Loc == scm.LocReg {
				ctx.UnprotectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d6.Reg)
				ctx.UnprotectReg(d6.Reg2)
			}
		}
		ps527 := scm.PhiState{General: ps.General}
		ps527.OverlayValues = make([]scm.JITValueDesc, 527)
		ps527.OverlayValues[1] = d1
		ps527.OverlayValues[2] = d2
		ps527.OverlayValues[3] = d3
		ps527.OverlayValues[4] = d4
		ps527.OverlayValues[5] = d5
		ps527.OverlayValues[6] = d6
		ps527.OverlayValues[7] = d7
		ps527.OverlayValues[8] = d8
		ps527.OverlayValues[9] = d9
		ps527.OverlayValues[10] = d10
		ps527.OverlayValues[11] = d11
		ps527.OverlayValues[12] = d12
		ps527.OverlayValues[13] = d13
		ps527.OverlayValues[14] = d14
		ps527.OverlayValues[15] = d15
		ps527.OverlayValues[17] = d17
		ps527.OverlayValues[18] = d18
		ps527.OverlayValues[19] = d19
		ps527.OverlayValues[20] = d20
		ps527.OverlayValues[21] = d21
		ps527.OverlayValues[22] = d22
		ps527.OverlayValues[23] = d23
		ps527.OverlayValues[24] = d24
		ps527.OverlayValues[25] = d25
		ps527.OverlayValues[26] = d26
		ps527.OverlayValues[27] = d27
		ps527.OverlayValues[28] = d28
		ps527.OverlayValues[29] = d29
		ps527.OverlayValues[30] = d30
		ps527.OverlayValues[31] = d31
		ps527.OverlayValues[32] = d32
		ps527.OverlayValues[33] = d33
		ps527.OverlayValues[34] = d34
		ps527.OverlayValues[35] = d35
		ps527.OverlayValues[36] = d36
		ps527.OverlayValues[37] = d37
		ps527.OverlayValues[38] = d38
		ps527.OverlayValues[39] = d39
		ps527.OverlayValues[40] = d40
		ps527.OverlayValues[41] = d41
		ps527.OverlayValues[42] = d42
		ps527.OverlayValues[43] = d43
		ps527.OverlayValues[44] = d44
		ps527.OverlayValues[45] = d45
		ps527.OverlayValues[46] = d46
		ps527.OverlayValues[47] = d47
		ps527.OverlayValues[48] = d48
		ps527.OverlayValues[49] = d49
		ps527.OverlayValues[52] = d52
		ps527.OverlayValues[53] = d53
		ps527.OverlayValues[54] = d54
		ps527.OverlayValues[109] = d109
		ps527.OverlayValues[110] = d110
		ps527.OverlayValues[111] = d111
		ps527.OverlayValues[112] = d112
		ps527.OverlayValues[113] = d113
		ps527.OverlayValues[114] = d114
		ps527.OverlayValues[115] = d115
		ps527.OverlayValues[116] = d116
		ps527.OverlayValues[117] = d117
		ps527.OverlayValues[118] = d118
		ps527.OverlayValues[119] = d119
		ps527.OverlayValues[120] = d120
		ps527.OverlayValues[121] = d121
		ps527.OverlayValues[122] = d122
		ps527.OverlayValues[123] = d123
		ps527.OverlayValues[124] = d124
		ps527.OverlayValues[125] = d125
		ps527.OverlayValues[126] = d126
		ps527.OverlayValues[127] = d127
		ps527.OverlayValues[128] = d128
		ps527.OverlayValues[129] = d129
		ps527.OverlayValues[130] = d130
		ps527.OverlayValues[131] = d131
		ps527.OverlayValues[132] = d132
		ps527.OverlayValues[133] = d133
		ps527.OverlayValues[134] = d134
		ps527.OverlayValues[135] = d135
		ps527.OverlayValues[136] = d136
		ps527.OverlayValues[137] = d137
		ps527.OverlayValues[140] = d140
		ps527.OverlayValues[225] = d225
		ps527.OverlayValues[226] = d226
		ps527.OverlayValues[227] = d227
		ps527.OverlayValues[228] = d228
		ps527.OverlayValues[230] = d230
		ps527.OverlayValues[231] = d231
		ps527.OverlayValues[232] = d232
		ps527.OverlayValues[233] = d233
		ps527.OverlayValues[234] = d234
		ps527.OverlayValues[235] = d235
		ps527.OverlayValues[236] = d236
		ps527.OverlayValues[237] = d237
		ps527.OverlayValues[239] = d239
		ps527.OverlayValues[241] = d241
		ps527.OverlayValues[242] = d242
		ps527.OverlayValues[243] = d243
		ps527.OverlayValues[244] = d244
		ps527.OverlayValues[245] = d245
		ps527.OverlayValues[248] = d248
		ps527.OverlayValues[350] = d350
		ps527.OverlayValues[351] = d351
		ps527.OverlayValues[352] = d352
		ps527.OverlayValues[353] = d353
		ps527.OverlayValues[354] = d354
		ps527.OverlayValues[356] = d356
		ps527.OverlayValues[357] = d357
		ps527.OverlayValues[358] = d358
		ps527.OverlayValues[359] = d359
		ps527.OverlayValues[360] = d360
		ps527.OverlayValues[361] = d361
		ps527.OverlayValues[362] = d362
		ps527.OverlayValues[363] = d363
		ps527.OverlayValues[364] = d364
		ps527.OverlayValues[365] = d365
		ps527.OverlayValues[366] = d366
		ps527.OverlayValues[367] = d367
		ps527.OverlayValues[368] = d368
		ps527.OverlayValues[369] = d369
		ps527.OverlayValues[370] = d370
		ps527.OverlayValues[371] = d371
		ps527.OverlayValues[372] = d372
		ps527.OverlayValues[373] = d373
		ps527.OverlayValues[374] = d374
		ps527.OverlayValues[375] = d375
		ps527.OverlayValues[376] = d376
		ps527.OverlayValues[377] = d377
		ps527.OverlayValues[378] = d378
		ps527.OverlayValues[379] = d379
		ps527.OverlayValues[380] = d380
		ps527.OverlayValues[381] = d381
		ps527.OverlayValues[382] = d382
		ps527.OverlayValues[383] = d383
		ps527.OverlayValues[384] = d384
		ps527.OverlayValues[524] = d524
		ps527.OverlayValues[525] = d525
		ps527.OverlayValues[526] = d526
		ps527.PhiValues = make([]scm.JITValueDesc, 2)
		d528 = d6
		ps527.PhiValues[0] = d528
		if ps527.General && bbs[8].Rendered {
			ctx.EmitJmp(lbl9)
			return result
		}
		return bbs[8].RenderPS(ps527)
		return result
	}
	bbs[8].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d529 := ps.PhiValues[0]
				ctx.EnsureDesc(&d529)
				ctx.EmitStoreToStack(d529, int32(bbs[8].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d530 := ps.PhiValues[1]
				ctx.EnsureDesc(&d530)
				ctx.EmitStoreToStack(d530, int32(bbs[8].PhiBase)+int32(16))
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
			d52 = ps.OverlayValues[52]
		}
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != scm.LocNone {
			d109 = ps.OverlayValues[109]
		}
		if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != scm.LocNone {
			d110 = ps.OverlayValues[110]
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
		if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != scm.LocNone {
			d140 = ps.OverlayValues[140]
		}
		if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != scm.LocNone {
			d225 = ps.OverlayValues[225]
		}
		if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != scm.LocNone {
			d226 = ps.OverlayValues[226]
		}
		if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
			d227 = ps.OverlayValues[227]
		}
		if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
			d228 = ps.OverlayValues[228]
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
		if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != scm.LocNone {
			d234 = ps.OverlayValues[234]
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
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
		}
		if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != scm.LocNone {
			d241 = ps.OverlayValues[241]
		}
		if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != scm.LocNone {
			d242 = ps.OverlayValues[242]
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
		if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != scm.LocNone {
			d248 = ps.OverlayValues[248]
		}
		if len(ps.OverlayValues) > 350 && ps.OverlayValues[350].Loc != scm.LocNone {
			d350 = ps.OverlayValues[350]
		}
		if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != scm.LocNone {
			d351 = ps.OverlayValues[351]
		}
		if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != scm.LocNone {
			d352 = ps.OverlayValues[352]
		}
		if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != scm.LocNone {
			d353 = ps.OverlayValues[353]
		}
		if len(ps.OverlayValues) > 354 && ps.OverlayValues[354].Loc != scm.LocNone {
			d354 = ps.OverlayValues[354]
		}
		if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != scm.LocNone {
			d356 = ps.OverlayValues[356]
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
		if len(ps.OverlayValues) > 362 && ps.OverlayValues[362].Loc != scm.LocNone {
			d362 = ps.OverlayValues[362]
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
		if len(ps.OverlayValues) > 524 && ps.OverlayValues[524].Loc != scm.LocNone {
			d524 = ps.OverlayValues[524]
		}
		if len(ps.OverlayValues) > 525 && ps.OverlayValues[525].Loc != scm.LocNone {
			d525 = ps.OverlayValues[525]
		}
		if len(ps.OverlayValues) > 526 && ps.OverlayValues[526].Loc != scm.LocNone {
			d526 = ps.OverlayValues[526]
		}
		if len(ps.OverlayValues) > 528 && ps.OverlayValues[528].Loc != scm.LocNone {
			d528 = ps.OverlayValues[528]
		}
		if len(ps.OverlayValues) > 529 && ps.OverlayValues[529].Loc != scm.LocNone {
			d529 = ps.OverlayValues[529]
		}
		if len(ps.OverlayValues) > 530 && ps.OverlayValues[530].Loc != scm.LocNone {
			d530 = ps.OverlayValues[530]
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
		var d531 scm.JITValueDesc
		if d8.Loc == scm.LocImm && d9.Loc == scm.LocImm {
			d531 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d8.Imm.Int()) == uint64(d9.Imm.Int()))}
		} else if d9.Loc == scm.LocImm {
			r114 := ctx.AllocRegExcept(d8.Reg)
			if d9.Imm.Int() >= -2147483648 && d9.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d8.Reg, int32(d9.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d9.Imm.Int()))
				ctx.EmitCmpInt64(d8.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r114, scm.CondEqual)
			d531 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r114}
			ctx.BindReg(r114, &d531)
		} else if d8.Loc == scm.LocImm {
			r115 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d8.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d9.Reg)
			ctx.EmitSetcc(r115, scm.CondEqual)
			d531 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r115}
			ctx.BindReg(r115, &d531)
		} else {
			r116 := ctx.AllocRegExcept(d8.Reg)
			ctx.EmitCmpInt64(d8.Reg, d9.Reg)
			ctx.EmitSetcc(r116, scm.CondEqual)
			d531 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r116}
			ctx.BindReg(r116, &d531)
		}
		d532 = d531
		ctx.EnsureDesc(&d532)
		if d532.Loc != scm.LocImm && d532.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d532.Loc == scm.LocImm {
			if d532.Imm.Bool() {
				if ps.General {
					ctx.SyncDesc(&d8)
					if d8.Loc == scm.LocReg {
						ctx.ProtectReg(d8.Reg)
					} else if d8.Loc == scm.LocRegPair {
						ctx.ProtectReg(d8.Reg)
						ctx.ProtectReg(d8.Reg2)
					}
					d533 = d8
					if d533.Loc == scm.LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d533)
					d534 = d533
					if d534.Loc == scm.LocImm {
						d534 = scm.JITValueDesc{Loc: scm.LocImm, Type: d534.Type, Imm: scm.NewInt(int64(uint64(d534.Imm.Int()) & 0xffffffff))}
					} else {
						ctx.EmitShlRegImm8(d534.Reg, 32)
						ctx.EmitShrRegImm8(d534.Reg, 32)
					}
					ctx.EmitStoreToStack(d534, int32(bbs[2].PhiBase)+int32(0))
					if d8.Loc == scm.LocReg {
						ctx.UnprotectReg(d8.Reg)
					} else if d8.Loc == scm.LocRegPair {
						ctx.UnprotectReg(d8.Reg)
						ctx.UnprotectReg(d8.Reg2)
					}
				}
				ps535 := scm.PhiState{General: ps.General}
				ps535.OverlayValues = make([]scm.JITValueDesc, 535)
				ps535.OverlayValues[1] = d1
				ps535.OverlayValues[2] = d2
				ps535.OverlayValues[3] = d3
				ps535.OverlayValues[4] = d4
				ps535.OverlayValues[5] = d5
				ps535.OverlayValues[6] = d6
				ps535.OverlayValues[7] = d7
				ps535.OverlayValues[8] = d8
				ps535.OverlayValues[9] = d9
				ps535.OverlayValues[10] = d10
				ps535.OverlayValues[11] = d11
				ps535.OverlayValues[12] = d12
				ps535.OverlayValues[13] = d13
				ps535.OverlayValues[14] = d14
				ps535.OverlayValues[15] = d15
				ps535.OverlayValues[17] = d17
				ps535.OverlayValues[18] = d18
				ps535.OverlayValues[19] = d19
				ps535.OverlayValues[20] = d20
				ps535.OverlayValues[21] = d21
				ps535.OverlayValues[22] = d22
				ps535.OverlayValues[23] = d23
				ps535.OverlayValues[24] = d24
				ps535.OverlayValues[25] = d25
				ps535.OverlayValues[26] = d26
				ps535.OverlayValues[27] = d27
				ps535.OverlayValues[28] = d28
				ps535.OverlayValues[29] = d29
				ps535.OverlayValues[30] = d30
				ps535.OverlayValues[31] = d31
				ps535.OverlayValues[32] = d32
				ps535.OverlayValues[33] = d33
				ps535.OverlayValues[34] = d34
				ps535.OverlayValues[35] = d35
				ps535.OverlayValues[36] = d36
				ps535.OverlayValues[37] = d37
				ps535.OverlayValues[38] = d38
				ps535.OverlayValues[39] = d39
				ps535.OverlayValues[40] = d40
				ps535.OverlayValues[41] = d41
				ps535.OverlayValues[42] = d42
				ps535.OverlayValues[43] = d43
				ps535.OverlayValues[44] = d44
				ps535.OverlayValues[45] = d45
				ps535.OverlayValues[46] = d46
				ps535.OverlayValues[47] = d47
				ps535.OverlayValues[48] = d48
				ps535.OverlayValues[49] = d49
				ps535.OverlayValues[52] = d52
				ps535.OverlayValues[53] = d53
				ps535.OverlayValues[54] = d54
				ps535.OverlayValues[109] = d109
				ps535.OverlayValues[110] = d110
				ps535.OverlayValues[111] = d111
				ps535.OverlayValues[112] = d112
				ps535.OverlayValues[113] = d113
				ps535.OverlayValues[114] = d114
				ps535.OverlayValues[115] = d115
				ps535.OverlayValues[116] = d116
				ps535.OverlayValues[117] = d117
				ps535.OverlayValues[118] = d118
				ps535.OverlayValues[119] = d119
				ps535.OverlayValues[120] = d120
				ps535.OverlayValues[121] = d121
				ps535.OverlayValues[122] = d122
				ps535.OverlayValues[123] = d123
				ps535.OverlayValues[124] = d124
				ps535.OverlayValues[125] = d125
				ps535.OverlayValues[126] = d126
				ps535.OverlayValues[127] = d127
				ps535.OverlayValues[128] = d128
				ps535.OverlayValues[129] = d129
				ps535.OverlayValues[130] = d130
				ps535.OverlayValues[131] = d131
				ps535.OverlayValues[132] = d132
				ps535.OverlayValues[133] = d133
				ps535.OverlayValues[134] = d134
				ps535.OverlayValues[135] = d135
				ps535.OverlayValues[136] = d136
				ps535.OverlayValues[137] = d137
				ps535.OverlayValues[140] = d140
				ps535.OverlayValues[225] = d225
				ps535.OverlayValues[226] = d226
				ps535.OverlayValues[227] = d227
				ps535.OverlayValues[228] = d228
				ps535.OverlayValues[230] = d230
				ps535.OverlayValues[231] = d231
				ps535.OverlayValues[232] = d232
				ps535.OverlayValues[233] = d233
				ps535.OverlayValues[234] = d234
				ps535.OverlayValues[235] = d235
				ps535.OverlayValues[236] = d236
				ps535.OverlayValues[237] = d237
				ps535.OverlayValues[239] = d239
				ps535.OverlayValues[241] = d241
				ps535.OverlayValues[242] = d242
				ps535.OverlayValues[243] = d243
				ps535.OverlayValues[244] = d244
				ps535.OverlayValues[245] = d245
				ps535.OverlayValues[248] = d248
				ps535.OverlayValues[350] = d350
				ps535.OverlayValues[351] = d351
				ps535.OverlayValues[352] = d352
				ps535.OverlayValues[353] = d353
				ps535.OverlayValues[354] = d354
				ps535.OverlayValues[356] = d356
				ps535.OverlayValues[357] = d357
				ps535.OverlayValues[358] = d358
				ps535.OverlayValues[359] = d359
				ps535.OverlayValues[360] = d360
				ps535.OverlayValues[361] = d361
				ps535.OverlayValues[362] = d362
				ps535.OverlayValues[363] = d363
				ps535.OverlayValues[364] = d364
				ps535.OverlayValues[365] = d365
				ps535.OverlayValues[366] = d366
				ps535.OverlayValues[367] = d367
				ps535.OverlayValues[368] = d368
				ps535.OverlayValues[369] = d369
				ps535.OverlayValues[370] = d370
				ps535.OverlayValues[371] = d371
				ps535.OverlayValues[372] = d372
				ps535.OverlayValues[373] = d373
				ps535.OverlayValues[374] = d374
				ps535.OverlayValues[375] = d375
				ps535.OverlayValues[376] = d376
				ps535.OverlayValues[377] = d377
				ps535.OverlayValues[378] = d378
				ps535.OverlayValues[379] = d379
				ps535.OverlayValues[380] = d380
				ps535.OverlayValues[381] = d381
				ps535.OverlayValues[382] = d382
				ps535.OverlayValues[383] = d383
				ps535.OverlayValues[384] = d384
				ps535.OverlayValues[524] = d524
				ps535.OverlayValues[525] = d525
				ps535.OverlayValues[526] = d526
				ps535.OverlayValues[528] = d528
				ps535.OverlayValues[529] = d529
				ps535.OverlayValues[530] = d530
				ps535.OverlayValues[531] = d531
				ps535.OverlayValues[532] = d532
				ps535.OverlayValues[533] = d533
				ps535.OverlayValues[534] = d534
				ps535.PhiValues = make([]scm.JITValueDesc, 1)
				d536 = d8
				ps535.PhiValues[0] = d536
				return bbs[2].RenderPS(ps535)
			}
			if ps.General {
			}
			ps537 := scm.PhiState{General: ps.General}
			ps537.OverlayValues = make([]scm.JITValueDesc, 537)
			ps537.OverlayValues[1] = d1
			ps537.OverlayValues[2] = d2
			ps537.OverlayValues[3] = d3
			ps537.OverlayValues[4] = d4
			ps537.OverlayValues[5] = d5
			ps537.OverlayValues[6] = d6
			ps537.OverlayValues[7] = d7
			ps537.OverlayValues[8] = d8
			ps537.OverlayValues[9] = d9
			ps537.OverlayValues[10] = d10
			ps537.OverlayValues[11] = d11
			ps537.OverlayValues[12] = d12
			ps537.OverlayValues[13] = d13
			ps537.OverlayValues[14] = d14
			ps537.OverlayValues[15] = d15
			ps537.OverlayValues[17] = d17
			ps537.OverlayValues[18] = d18
			ps537.OverlayValues[19] = d19
			ps537.OverlayValues[20] = d20
			ps537.OverlayValues[21] = d21
			ps537.OverlayValues[22] = d22
			ps537.OverlayValues[23] = d23
			ps537.OverlayValues[24] = d24
			ps537.OverlayValues[25] = d25
			ps537.OverlayValues[26] = d26
			ps537.OverlayValues[27] = d27
			ps537.OverlayValues[28] = d28
			ps537.OverlayValues[29] = d29
			ps537.OverlayValues[30] = d30
			ps537.OverlayValues[31] = d31
			ps537.OverlayValues[32] = d32
			ps537.OverlayValues[33] = d33
			ps537.OverlayValues[34] = d34
			ps537.OverlayValues[35] = d35
			ps537.OverlayValues[36] = d36
			ps537.OverlayValues[37] = d37
			ps537.OverlayValues[38] = d38
			ps537.OverlayValues[39] = d39
			ps537.OverlayValues[40] = d40
			ps537.OverlayValues[41] = d41
			ps537.OverlayValues[42] = d42
			ps537.OverlayValues[43] = d43
			ps537.OverlayValues[44] = d44
			ps537.OverlayValues[45] = d45
			ps537.OverlayValues[46] = d46
			ps537.OverlayValues[47] = d47
			ps537.OverlayValues[48] = d48
			ps537.OverlayValues[49] = d49
			ps537.OverlayValues[52] = d52
			ps537.OverlayValues[53] = d53
			ps537.OverlayValues[54] = d54
			ps537.OverlayValues[109] = d109
			ps537.OverlayValues[110] = d110
			ps537.OverlayValues[111] = d111
			ps537.OverlayValues[112] = d112
			ps537.OverlayValues[113] = d113
			ps537.OverlayValues[114] = d114
			ps537.OverlayValues[115] = d115
			ps537.OverlayValues[116] = d116
			ps537.OverlayValues[117] = d117
			ps537.OverlayValues[118] = d118
			ps537.OverlayValues[119] = d119
			ps537.OverlayValues[120] = d120
			ps537.OverlayValues[121] = d121
			ps537.OverlayValues[122] = d122
			ps537.OverlayValues[123] = d123
			ps537.OverlayValues[124] = d124
			ps537.OverlayValues[125] = d125
			ps537.OverlayValues[126] = d126
			ps537.OverlayValues[127] = d127
			ps537.OverlayValues[128] = d128
			ps537.OverlayValues[129] = d129
			ps537.OverlayValues[130] = d130
			ps537.OverlayValues[131] = d131
			ps537.OverlayValues[132] = d132
			ps537.OverlayValues[133] = d133
			ps537.OverlayValues[134] = d134
			ps537.OverlayValues[135] = d135
			ps537.OverlayValues[136] = d136
			ps537.OverlayValues[137] = d137
			ps537.OverlayValues[140] = d140
			ps537.OverlayValues[225] = d225
			ps537.OverlayValues[226] = d226
			ps537.OverlayValues[227] = d227
			ps537.OverlayValues[228] = d228
			ps537.OverlayValues[230] = d230
			ps537.OverlayValues[231] = d231
			ps537.OverlayValues[232] = d232
			ps537.OverlayValues[233] = d233
			ps537.OverlayValues[234] = d234
			ps537.OverlayValues[235] = d235
			ps537.OverlayValues[236] = d236
			ps537.OverlayValues[237] = d237
			ps537.OverlayValues[239] = d239
			ps537.OverlayValues[241] = d241
			ps537.OverlayValues[242] = d242
			ps537.OverlayValues[243] = d243
			ps537.OverlayValues[244] = d244
			ps537.OverlayValues[245] = d245
			ps537.OverlayValues[248] = d248
			ps537.OverlayValues[350] = d350
			ps537.OverlayValues[351] = d351
			ps537.OverlayValues[352] = d352
			ps537.OverlayValues[353] = d353
			ps537.OverlayValues[354] = d354
			ps537.OverlayValues[356] = d356
			ps537.OverlayValues[357] = d357
			ps537.OverlayValues[358] = d358
			ps537.OverlayValues[359] = d359
			ps537.OverlayValues[360] = d360
			ps537.OverlayValues[361] = d361
			ps537.OverlayValues[362] = d362
			ps537.OverlayValues[363] = d363
			ps537.OverlayValues[364] = d364
			ps537.OverlayValues[365] = d365
			ps537.OverlayValues[366] = d366
			ps537.OverlayValues[367] = d367
			ps537.OverlayValues[368] = d368
			ps537.OverlayValues[369] = d369
			ps537.OverlayValues[370] = d370
			ps537.OverlayValues[371] = d371
			ps537.OverlayValues[372] = d372
			ps537.OverlayValues[373] = d373
			ps537.OverlayValues[374] = d374
			ps537.OverlayValues[375] = d375
			ps537.OverlayValues[376] = d376
			ps537.OverlayValues[377] = d377
			ps537.OverlayValues[378] = d378
			ps537.OverlayValues[379] = d379
			ps537.OverlayValues[380] = d380
			ps537.OverlayValues[381] = d381
			ps537.OverlayValues[382] = d382
			ps537.OverlayValues[383] = d383
			ps537.OverlayValues[384] = d384
			ps537.OverlayValues[524] = d524
			ps537.OverlayValues[525] = d525
			ps537.OverlayValues[526] = d526
			ps537.OverlayValues[528] = d528
			ps537.OverlayValues[529] = d529
			ps537.OverlayValues[530] = d530
			ps537.OverlayValues[531] = d531
			ps537.OverlayValues[532] = d532
			ps537.OverlayValues[533] = d533
			ps537.OverlayValues[534] = d534
			ps537.OverlayValues[536] = d536
			return bbs[10].RenderPS(ps537)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d538 := ps.PhiValues[0]
				ctx.EnsureDesc(&d538)
				ctx.EmitStoreToStack(d538, int32(bbs[8].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d539 := ps.PhiValues[1]
				ctx.EnsureDesc(&d539)
				ctx.EmitStoreToStack(d539, int32(bbs[8].PhiBase)+int32(16))
			}
			ps.General = true
			return bbs[8].RenderPS(ps)
		}
		lbl26 := ctx.ReserveLabel()
		lbl27 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d532.Reg, 0)
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
		d540 = d8
		if d540.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d540)
		d541 = d540
		if d541.Loc == scm.LocImm {
			d541 = scm.JITValueDesc{Loc: scm.LocImm, Type: d541.Type, Imm: scm.NewInt(int64(uint64(d541.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d541.Reg, 32)
			ctx.EmitShrRegImm8(d541.Reg, 32)
		}
		ctx.EmitStoreToStack(d541, int32(bbs[2].PhiBase)+int32(0))
		if d8.Loc == scm.LocReg {
			ctx.UnprotectReg(d8.Reg)
		} else if d8.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d8.Reg)
			ctx.UnprotectReg(d8.Reg2)
		}
		ctx.EmitJmp(lbl3)
		ctx.MarkLabel(lbl27)
		ctx.EmitJmp(lbl11)
		ps542 := scm.PhiState{General: true}
		ps542.OverlayValues = make([]scm.JITValueDesc, 542)
		ps542.OverlayValues[1] = d1
		ps542.OverlayValues[2] = d2
		ps542.OverlayValues[3] = d3
		ps542.OverlayValues[4] = d4
		ps542.OverlayValues[5] = d5
		ps542.OverlayValues[6] = d6
		ps542.OverlayValues[7] = d7
		ps542.OverlayValues[8] = d8
		ps542.OverlayValues[9] = d9
		ps542.OverlayValues[10] = d10
		ps542.OverlayValues[11] = d11
		ps542.OverlayValues[12] = d12
		ps542.OverlayValues[13] = d13
		ps542.OverlayValues[14] = d14
		ps542.OverlayValues[15] = d15
		ps542.OverlayValues[17] = d17
		ps542.OverlayValues[18] = d18
		ps542.OverlayValues[19] = d19
		ps542.OverlayValues[20] = d20
		ps542.OverlayValues[21] = d21
		ps542.OverlayValues[22] = d22
		ps542.OverlayValues[23] = d23
		ps542.OverlayValues[24] = d24
		ps542.OverlayValues[25] = d25
		ps542.OverlayValues[26] = d26
		ps542.OverlayValues[27] = d27
		ps542.OverlayValues[28] = d28
		ps542.OverlayValues[29] = d29
		ps542.OverlayValues[30] = d30
		ps542.OverlayValues[31] = d31
		ps542.OverlayValues[32] = d32
		ps542.OverlayValues[33] = d33
		ps542.OverlayValues[34] = d34
		ps542.OverlayValues[35] = d35
		ps542.OverlayValues[36] = d36
		ps542.OverlayValues[37] = d37
		ps542.OverlayValues[38] = d38
		ps542.OverlayValues[39] = d39
		ps542.OverlayValues[40] = d40
		ps542.OverlayValues[41] = d41
		ps542.OverlayValues[42] = d42
		ps542.OverlayValues[43] = d43
		ps542.OverlayValues[44] = d44
		ps542.OverlayValues[45] = d45
		ps542.OverlayValues[46] = d46
		ps542.OverlayValues[47] = d47
		ps542.OverlayValues[48] = d48
		ps542.OverlayValues[49] = d49
		ps542.OverlayValues[52] = d52
		ps542.OverlayValues[53] = d53
		ps542.OverlayValues[54] = d54
		ps542.OverlayValues[109] = d109
		ps542.OverlayValues[110] = d110
		ps542.OverlayValues[111] = d111
		ps542.OverlayValues[112] = d112
		ps542.OverlayValues[113] = d113
		ps542.OverlayValues[114] = d114
		ps542.OverlayValues[115] = d115
		ps542.OverlayValues[116] = d116
		ps542.OverlayValues[117] = d117
		ps542.OverlayValues[118] = d118
		ps542.OverlayValues[119] = d119
		ps542.OverlayValues[120] = d120
		ps542.OverlayValues[121] = d121
		ps542.OverlayValues[122] = d122
		ps542.OverlayValues[123] = d123
		ps542.OverlayValues[124] = d124
		ps542.OverlayValues[125] = d125
		ps542.OverlayValues[126] = d126
		ps542.OverlayValues[127] = d127
		ps542.OverlayValues[128] = d128
		ps542.OverlayValues[129] = d129
		ps542.OverlayValues[130] = d130
		ps542.OverlayValues[131] = d131
		ps542.OverlayValues[132] = d132
		ps542.OverlayValues[133] = d133
		ps542.OverlayValues[134] = d134
		ps542.OverlayValues[135] = d135
		ps542.OverlayValues[136] = d136
		ps542.OverlayValues[137] = d137
		ps542.OverlayValues[140] = d140
		ps542.OverlayValues[225] = d225
		ps542.OverlayValues[226] = d226
		ps542.OverlayValues[227] = d227
		ps542.OverlayValues[228] = d228
		ps542.OverlayValues[230] = d230
		ps542.OverlayValues[231] = d231
		ps542.OverlayValues[232] = d232
		ps542.OverlayValues[233] = d233
		ps542.OverlayValues[234] = d234
		ps542.OverlayValues[235] = d235
		ps542.OverlayValues[236] = d236
		ps542.OverlayValues[237] = d237
		ps542.OverlayValues[239] = d239
		ps542.OverlayValues[241] = d241
		ps542.OverlayValues[242] = d242
		ps542.OverlayValues[243] = d243
		ps542.OverlayValues[244] = d244
		ps542.OverlayValues[245] = d245
		ps542.OverlayValues[248] = d248
		ps542.OverlayValues[350] = d350
		ps542.OverlayValues[351] = d351
		ps542.OverlayValues[352] = d352
		ps542.OverlayValues[353] = d353
		ps542.OverlayValues[354] = d354
		ps542.OverlayValues[356] = d356
		ps542.OverlayValues[357] = d357
		ps542.OverlayValues[358] = d358
		ps542.OverlayValues[359] = d359
		ps542.OverlayValues[360] = d360
		ps542.OverlayValues[361] = d361
		ps542.OverlayValues[362] = d362
		ps542.OverlayValues[363] = d363
		ps542.OverlayValues[364] = d364
		ps542.OverlayValues[365] = d365
		ps542.OverlayValues[366] = d366
		ps542.OverlayValues[367] = d367
		ps542.OverlayValues[368] = d368
		ps542.OverlayValues[369] = d369
		ps542.OverlayValues[370] = d370
		ps542.OverlayValues[371] = d371
		ps542.OverlayValues[372] = d372
		ps542.OverlayValues[373] = d373
		ps542.OverlayValues[374] = d374
		ps542.OverlayValues[375] = d375
		ps542.OverlayValues[376] = d376
		ps542.OverlayValues[377] = d377
		ps542.OverlayValues[378] = d378
		ps542.OverlayValues[379] = d379
		ps542.OverlayValues[380] = d380
		ps542.OverlayValues[381] = d381
		ps542.OverlayValues[382] = d382
		ps542.OverlayValues[383] = d383
		ps542.OverlayValues[384] = d384
		ps542.OverlayValues[524] = d524
		ps542.OverlayValues[525] = d525
		ps542.OverlayValues[526] = d526
		ps542.OverlayValues[528] = d528
		ps542.OverlayValues[529] = d529
		ps542.OverlayValues[530] = d530
		ps542.OverlayValues[531] = d531
		ps542.OverlayValues[532] = d532
		ps542.OverlayValues[533] = d533
		ps542.OverlayValues[534] = d534
		ps542.OverlayValues[536] = d536
		ps542.OverlayValues[538] = d538
		ps542.OverlayValues[539] = d539
		ps542.OverlayValues[540] = d540
		ps542.OverlayValues[541] = d541
		ps542.PhiValues = make([]scm.JITValueDesc, 1)
		d544 = d8
		ps542.PhiValues[0] = d544
		ps543 := scm.PhiState{General: true}
		ps543.OverlayValues = make([]scm.JITValueDesc, 545)
		ps543.OverlayValues[1] = d1
		ps543.OverlayValues[2] = d2
		ps543.OverlayValues[3] = d3
		ps543.OverlayValues[4] = d4
		ps543.OverlayValues[5] = d5
		ps543.OverlayValues[6] = d6
		ps543.OverlayValues[7] = d7
		ps543.OverlayValues[8] = d8
		ps543.OverlayValues[9] = d9
		ps543.OverlayValues[10] = d10
		ps543.OverlayValues[11] = d11
		ps543.OverlayValues[12] = d12
		ps543.OverlayValues[13] = d13
		ps543.OverlayValues[14] = d14
		ps543.OverlayValues[15] = d15
		ps543.OverlayValues[17] = d17
		ps543.OverlayValues[18] = d18
		ps543.OverlayValues[19] = d19
		ps543.OverlayValues[20] = d20
		ps543.OverlayValues[21] = d21
		ps543.OverlayValues[22] = d22
		ps543.OverlayValues[23] = d23
		ps543.OverlayValues[24] = d24
		ps543.OverlayValues[25] = d25
		ps543.OverlayValues[26] = d26
		ps543.OverlayValues[27] = d27
		ps543.OverlayValues[28] = d28
		ps543.OverlayValues[29] = d29
		ps543.OverlayValues[30] = d30
		ps543.OverlayValues[31] = d31
		ps543.OverlayValues[32] = d32
		ps543.OverlayValues[33] = d33
		ps543.OverlayValues[34] = d34
		ps543.OverlayValues[35] = d35
		ps543.OverlayValues[36] = d36
		ps543.OverlayValues[37] = d37
		ps543.OverlayValues[38] = d38
		ps543.OverlayValues[39] = d39
		ps543.OverlayValues[40] = d40
		ps543.OverlayValues[41] = d41
		ps543.OverlayValues[42] = d42
		ps543.OverlayValues[43] = d43
		ps543.OverlayValues[44] = d44
		ps543.OverlayValues[45] = d45
		ps543.OverlayValues[46] = d46
		ps543.OverlayValues[47] = d47
		ps543.OverlayValues[48] = d48
		ps543.OverlayValues[49] = d49
		ps543.OverlayValues[52] = d52
		ps543.OverlayValues[53] = d53
		ps543.OverlayValues[54] = d54
		ps543.OverlayValues[109] = d109
		ps543.OverlayValues[110] = d110
		ps543.OverlayValues[111] = d111
		ps543.OverlayValues[112] = d112
		ps543.OverlayValues[113] = d113
		ps543.OverlayValues[114] = d114
		ps543.OverlayValues[115] = d115
		ps543.OverlayValues[116] = d116
		ps543.OverlayValues[117] = d117
		ps543.OverlayValues[118] = d118
		ps543.OverlayValues[119] = d119
		ps543.OverlayValues[120] = d120
		ps543.OverlayValues[121] = d121
		ps543.OverlayValues[122] = d122
		ps543.OverlayValues[123] = d123
		ps543.OverlayValues[124] = d124
		ps543.OverlayValues[125] = d125
		ps543.OverlayValues[126] = d126
		ps543.OverlayValues[127] = d127
		ps543.OverlayValues[128] = d128
		ps543.OverlayValues[129] = d129
		ps543.OverlayValues[130] = d130
		ps543.OverlayValues[131] = d131
		ps543.OverlayValues[132] = d132
		ps543.OverlayValues[133] = d133
		ps543.OverlayValues[134] = d134
		ps543.OverlayValues[135] = d135
		ps543.OverlayValues[136] = d136
		ps543.OverlayValues[137] = d137
		ps543.OverlayValues[140] = d140
		ps543.OverlayValues[225] = d225
		ps543.OverlayValues[226] = d226
		ps543.OverlayValues[227] = d227
		ps543.OverlayValues[228] = d228
		ps543.OverlayValues[230] = d230
		ps543.OverlayValues[231] = d231
		ps543.OverlayValues[232] = d232
		ps543.OverlayValues[233] = d233
		ps543.OverlayValues[234] = d234
		ps543.OverlayValues[235] = d235
		ps543.OverlayValues[236] = d236
		ps543.OverlayValues[237] = d237
		ps543.OverlayValues[239] = d239
		ps543.OverlayValues[241] = d241
		ps543.OverlayValues[242] = d242
		ps543.OverlayValues[243] = d243
		ps543.OverlayValues[244] = d244
		ps543.OverlayValues[245] = d245
		ps543.OverlayValues[248] = d248
		ps543.OverlayValues[350] = d350
		ps543.OverlayValues[351] = d351
		ps543.OverlayValues[352] = d352
		ps543.OverlayValues[353] = d353
		ps543.OverlayValues[354] = d354
		ps543.OverlayValues[356] = d356
		ps543.OverlayValues[357] = d357
		ps543.OverlayValues[358] = d358
		ps543.OverlayValues[359] = d359
		ps543.OverlayValues[360] = d360
		ps543.OverlayValues[361] = d361
		ps543.OverlayValues[362] = d362
		ps543.OverlayValues[363] = d363
		ps543.OverlayValues[364] = d364
		ps543.OverlayValues[365] = d365
		ps543.OverlayValues[366] = d366
		ps543.OverlayValues[367] = d367
		ps543.OverlayValues[368] = d368
		ps543.OverlayValues[369] = d369
		ps543.OverlayValues[370] = d370
		ps543.OverlayValues[371] = d371
		ps543.OverlayValues[372] = d372
		ps543.OverlayValues[373] = d373
		ps543.OverlayValues[374] = d374
		ps543.OverlayValues[375] = d375
		ps543.OverlayValues[376] = d376
		ps543.OverlayValues[377] = d377
		ps543.OverlayValues[378] = d378
		ps543.OverlayValues[379] = d379
		ps543.OverlayValues[380] = d380
		ps543.OverlayValues[381] = d381
		ps543.OverlayValues[382] = d382
		ps543.OverlayValues[383] = d383
		ps543.OverlayValues[384] = d384
		ps543.OverlayValues[524] = d524
		ps543.OverlayValues[525] = d525
		ps543.OverlayValues[526] = d526
		ps543.OverlayValues[528] = d528
		ps543.OverlayValues[529] = d529
		ps543.OverlayValues[530] = d530
		ps543.OverlayValues[531] = d531
		ps543.OverlayValues[532] = d532
		ps543.OverlayValues[533] = d533
		ps543.OverlayValues[534] = d534
		ps543.OverlayValues[536] = d536
		ps543.OverlayValues[538] = d538
		ps543.OverlayValues[539] = d539
		ps543.OverlayValues[540] = d540
		ps543.OverlayValues[541] = d541
		ps543.OverlayValues[544] = d544
		snap545 := d1
		snap546 := d2
		snap547 := d3
		snap548 := d4
		snap549 := d5
		snap550 := d6
		snap551 := d7
		snap552 := d8
		snap553 := d9
		snap554 := d10
		snap555 := d11
		snap556 := d12
		snap557 := d13
		snap558 := d14
		snap559 := d15
		snap560 := d17
		snap561 := d18
		snap562 := d19
		snap563 := d20
		snap564 := d21
		snap565 := d22
		snap566 := d23
		snap567 := d24
		snap568 := d25
		snap569 := d26
		snap570 := d27
		snap571 := d28
		snap572 := d29
		snap573 := d30
		snap574 := d31
		snap575 := d32
		snap576 := d33
		snap577 := d34
		snap578 := d35
		snap579 := d36
		snap580 := d37
		snap581 := d38
		snap582 := d39
		snap583 := d40
		snap584 := d41
		snap585 := d42
		snap586 := d43
		snap587 := d44
		snap588 := d45
		snap589 := d46
		snap590 := d47
		snap591 := d48
		snap592 := d49
		snap593 := d52
		snap594 := d53
		snap595 := d54
		snap596 := d109
		snap597 := d110
		snap598 := d111
		snap599 := d112
		snap600 := d113
		snap601 := d114
		snap602 := d115
		snap603 := d116
		snap604 := d117
		snap605 := d118
		snap606 := d119
		snap607 := d120
		snap608 := d121
		snap609 := d122
		snap610 := d123
		snap611 := d124
		snap612 := d125
		snap613 := d126
		snap614 := d127
		snap615 := d128
		snap616 := d129
		snap617 := d130
		snap618 := d131
		snap619 := d132
		snap620 := d133
		snap621 := d134
		snap622 := d135
		snap623 := d136
		snap624 := d137
		snap625 := d140
		snap626 := d225
		snap627 := d226
		snap628 := d227
		snap629 := d228
		snap630 := d230
		snap631 := d231
		snap632 := d232
		snap633 := d233
		snap634 := d234
		snap635 := d235
		snap636 := d236
		snap637 := d237
		snap638 := d239
		snap639 := d241
		snap640 := d242
		snap641 := d243
		snap642 := d244
		snap643 := d245
		snap644 := d248
		snap645 := d350
		snap646 := d351
		snap647 := d352
		snap648 := d353
		snap649 := d354
		snap650 := d356
		snap651 := d357
		snap652 := d358
		snap653 := d359
		snap654 := d360
		snap655 := d361
		snap656 := d362
		snap657 := d363
		snap658 := d364
		snap659 := d365
		snap660 := d366
		snap661 := d367
		snap662 := d368
		snap663 := d369
		snap664 := d370
		snap665 := d371
		snap666 := d372
		snap667 := d373
		snap668 := d374
		snap669 := d375
		snap670 := d376
		snap671 := d377
		snap672 := d378
		snap673 := d379
		snap674 := d380
		snap675 := d381
		snap676 := d382
		snap677 := d383
		snap678 := d384
		snap679 := d524
		snap680 := d525
		snap681 := d526
		snap682 := d528
		snap683 := d529
		snap684 := d530
		snap685 := d531
		snap686 := d532
		snap687 := d533
		snap688 := d534
		snap689 := d536
		snap690 := d538
		snap691 := d539
		snap692 := d540
		snap693 := d541
		snap694 := d544
		alloc695 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps542)
		}
		ctx.RestoreAllocState(alloc695)
		d1 = snap545
		d2 = snap546
		d3 = snap547
		d4 = snap548
		d5 = snap549
		d6 = snap550
		d7 = snap551
		d8 = snap552
		d9 = snap553
		d10 = snap554
		d11 = snap555
		d12 = snap556
		d13 = snap557
		d14 = snap558
		d15 = snap559
		d17 = snap560
		d18 = snap561
		d19 = snap562
		d20 = snap563
		d21 = snap564
		d22 = snap565
		d23 = snap566
		d24 = snap567
		d25 = snap568
		d26 = snap569
		d27 = snap570
		d28 = snap571
		d29 = snap572
		d30 = snap573
		d31 = snap574
		d32 = snap575
		d33 = snap576
		d34 = snap577
		d35 = snap578
		d36 = snap579
		d37 = snap580
		d38 = snap581
		d39 = snap582
		d40 = snap583
		d41 = snap584
		d42 = snap585
		d43 = snap586
		d44 = snap587
		d45 = snap588
		d46 = snap589
		d47 = snap590
		d48 = snap591
		d49 = snap592
		d52 = snap593
		d53 = snap594
		d54 = snap595
		d109 = snap596
		d110 = snap597
		d111 = snap598
		d112 = snap599
		d113 = snap600
		d114 = snap601
		d115 = snap602
		d116 = snap603
		d117 = snap604
		d118 = snap605
		d119 = snap606
		d120 = snap607
		d121 = snap608
		d122 = snap609
		d123 = snap610
		d124 = snap611
		d125 = snap612
		d126 = snap613
		d127 = snap614
		d128 = snap615
		d129 = snap616
		d130 = snap617
		d131 = snap618
		d132 = snap619
		d133 = snap620
		d134 = snap621
		d135 = snap622
		d136 = snap623
		d137 = snap624
		d140 = snap625
		d225 = snap626
		d226 = snap627
		d227 = snap628
		d228 = snap629
		d230 = snap630
		d231 = snap631
		d232 = snap632
		d233 = snap633
		d234 = snap634
		d235 = snap635
		d236 = snap636
		d237 = snap637
		d239 = snap638
		d241 = snap639
		d242 = snap640
		d243 = snap641
		d244 = snap642
		d245 = snap643
		d248 = snap644
		d350 = snap645
		d351 = snap646
		d352 = snap647
		d353 = snap648
		d354 = snap649
		d356 = snap650
		d357 = snap651
		d358 = snap652
		d359 = snap653
		d360 = snap654
		d361 = snap655
		d362 = snap656
		d363 = snap657
		d364 = snap658
		d365 = snap659
		d366 = snap660
		d367 = snap661
		d368 = snap662
		d369 = snap663
		d370 = snap664
		d371 = snap665
		d372 = snap666
		d373 = snap667
		d374 = snap668
		d375 = snap669
		d376 = snap670
		d377 = snap671
		d378 = snap672
		d379 = snap673
		d380 = snap674
		d381 = snap675
		d382 = snap676
		d383 = snap677
		d384 = snap678
		d524 = snap679
		d525 = snap680
		d526 = snap681
		d528 = snap682
		d529 = snap683
		d530 = snap684
		d531 = snap685
		d532 = snap686
		d533 = snap687
		d534 = snap688
		d536 = snap689
		d538 = snap690
		d539 = snap691
		d540 = snap692
		d541 = snap693
		d544 = snap694
		if !bbs[10].Rendered {
			return bbs[10].RenderPS(ps543)
		}
		return result
		ctx.FreeDesc(&d531)
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
			d52 = ps.OverlayValues[52]
		}
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != scm.LocNone {
			d109 = ps.OverlayValues[109]
		}
		if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != scm.LocNone {
			d110 = ps.OverlayValues[110]
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
		if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != scm.LocNone {
			d140 = ps.OverlayValues[140]
		}
		if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != scm.LocNone {
			d225 = ps.OverlayValues[225]
		}
		if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != scm.LocNone {
			d226 = ps.OverlayValues[226]
		}
		if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
			d227 = ps.OverlayValues[227]
		}
		if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
			d228 = ps.OverlayValues[228]
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
		if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != scm.LocNone {
			d234 = ps.OverlayValues[234]
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
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
		}
		if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != scm.LocNone {
			d241 = ps.OverlayValues[241]
		}
		if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != scm.LocNone {
			d242 = ps.OverlayValues[242]
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
		if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != scm.LocNone {
			d248 = ps.OverlayValues[248]
		}
		if len(ps.OverlayValues) > 350 && ps.OverlayValues[350].Loc != scm.LocNone {
			d350 = ps.OverlayValues[350]
		}
		if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != scm.LocNone {
			d351 = ps.OverlayValues[351]
		}
		if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != scm.LocNone {
			d352 = ps.OverlayValues[352]
		}
		if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != scm.LocNone {
			d353 = ps.OverlayValues[353]
		}
		if len(ps.OverlayValues) > 354 && ps.OverlayValues[354].Loc != scm.LocNone {
			d354 = ps.OverlayValues[354]
		}
		if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != scm.LocNone {
			d356 = ps.OverlayValues[356]
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
		if len(ps.OverlayValues) > 362 && ps.OverlayValues[362].Loc != scm.LocNone {
			d362 = ps.OverlayValues[362]
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
		if len(ps.OverlayValues) > 524 && ps.OverlayValues[524].Loc != scm.LocNone {
			d524 = ps.OverlayValues[524]
		}
		if len(ps.OverlayValues) > 525 && ps.OverlayValues[525].Loc != scm.LocNone {
			d525 = ps.OverlayValues[525]
		}
		if len(ps.OverlayValues) > 526 && ps.OverlayValues[526].Loc != scm.LocNone {
			d526 = ps.OverlayValues[526]
		}
		if len(ps.OverlayValues) > 528 && ps.OverlayValues[528].Loc != scm.LocNone {
			d528 = ps.OverlayValues[528]
		}
		if len(ps.OverlayValues) > 529 && ps.OverlayValues[529].Loc != scm.LocNone {
			d529 = ps.OverlayValues[529]
		}
		if len(ps.OverlayValues) > 530 && ps.OverlayValues[530].Loc != scm.LocNone {
			d530 = ps.OverlayValues[530]
		}
		if len(ps.OverlayValues) > 531 && ps.OverlayValues[531].Loc != scm.LocNone {
			d531 = ps.OverlayValues[531]
		}
		if len(ps.OverlayValues) > 532 && ps.OverlayValues[532].Loc != scm.LocNone {
			d532 = ps.OverlayValues[532]
		}
		if len(ps.OverlayValues) > 533 && ps.OverlayValues[533].Loc != scm.LocNone {
			d533 = ps.OverlayValues[533]
		}
		if len(ps.OverlayValues) > 534 && ps.OverlayValues[534].Loc != scm.LocNone {
			d534 = ps.OverlayValues[534]
		}
		if len(ps.OverlayValues) > 536 && ps.OverlayValues[536].Loc != scm.LocNone {
			d536 = ps.OverlayValues[536]
		}
		if len(ps.OverlayValues) > 538 && ps.OverlayValues[538].Loc != scm.LocNone {
			d538 = ps.OverlayValues[538]
		}
		if len(ps.OverlayValues) > 539 && ps.OverlayValues[539].Loc != scm.LocNone {
			d539 = ps.OverlayValues[539]
		}
		if len(ps.OverlayValues) > 540 && ps.OverlayValues[540].Loc != scm.LocNone {
			d540 = ps.OverlayValues[540]
		}
		if len(ps.OverlayValues) > 541 && ps.OverlayValues[541].Loc != scm.LocNone {
			d541 = ps.OverlayValues[541]
		}
		if len(ps.OverlayValues) > 544 && ps.OverlayValues[544].Loc != scm.LocNone {
			d544 = ps.OverlayValues[544]
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
			d696 = d5
			if d696.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d696)
			d697 = d696
			if d697.Loc == scm.LocImm {
				d697 = scm.JITValueDesc{Loc: scm.LocImm, Type: d697.Type, Imm: scm.NewInt(int64(uint64(d697.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d697.Reg, 32)
				ctx.EmitShrRegImm8(d697.Reg, 32)
			}
			ctx.EmitStoreToStack(d697, int32(bbs[8].PhiBase)+int32(0))
			d698 = d7
			if d698.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d698)
			d699 = d698
			if d699.Loc == scm.LocImm {
				d699 = scm.JITValueDesc{Loc: scm.LocImm, Type: d699.Type, Imm: scm.NewInt(int64(uint64(d699.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d699.Reg, 32)
				ctx.EmitShrRegImm8(d699.Reg, 32)
			}
			ctx.EmitStoreToStack(d699, int32(bbs[8].PhiBase)+int32(16))
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
		ps700 := scm.PhiState{General: ps.General}
		ps700.OverlayValues = make([]scm.JITValueDesc, 700)
		ps700.OverlayValues[1] = d1
		ps700.OverlayValues[2] = d2
		ps700.OverlayValues[3] = d3
		ps700.OverlayValues[4] = d4
		ps700.OverlayValues[5] = d5
		ps700.OverlayValues[6] = d6
		ps700.OverlayValues[7] = d7
		ps700.OverlayValues[8] = d8
		ps700.OverlayValues[9] = d9
		ps700.OverlayValues[10] = d10
		ps700.OverlayValues[11] = d11
		ps700.OverlayValues[12] = d12
		ps700.OverlayValues[13] = d13
		ps700.OverlayValues[14] = d14
		ps700.OverlayValues[15] = d15
		ps700.OverlayValues[17] = d17
		ps700.OverlayValues[18] = d18
		ps700.OverlayValues[19] = d19
		ps700.OverlayValues[20] = d20
		ps700.OverlayValues[21] = d21
		ps700.OverlayValues[22] = d22
		ps700.OverlayValues[23] = d23
		ps700.OverlayValues[24] = d24
		ps700.OverlayValues[25] = d25
		ps700.OverlayValues[26] = d26
		ps700.OverlayValues[27] = d27
		ps700.OverlayValues[28] = d28
		ps700.OverlayValues[29] = d29
		ps700.OverlayValues[30] = d30
		ps700.OverlayValues[31] = d31
		ps700.OverlayValues[32] = d32
		ps700.OverlayValues[33] = d33
		ps700.OverlayValues[34] = d34
		ps700.OverlayValues[35] = d35
		ps700.OverlayValues[36] = d36
		ps700.OverlayValues[37] = d37
		ps700.OverlayValues[38] = d38
		ps700.OverlayValues[39] = d39
		ps700.OverlayValues[40] = d40
		ps700.OverlayValues[41] = d41
		ps700.OverlayValues[42] = d42
		ps700.OverlayValues[43] = d43
		ps700.OverlayValues[44] = d44
		ps700.OverlayValues[45] = d45
		ps700.OverlayValues[46] = d46
		ps700.OverlayValues[47] = d47
		ps700.OverlayValues[48] = d48
		ps700.OverlayValues[49] = d49
		ps700.OverlayValues[52] = d52
		ps700.OverlayValues[53] = d53
		ps700.OverlayValues[54] = d54
		ps700.OverlayValues[109] = d109
		ps700.OverlayValues[110] = d110
		ps700.OverlayValues[111] = d111
		ps700.OverlayValues[112] = d112
		ps700.OverlayValues[113] = d113
		ps700.OverlayValues[114] = d114
		ps700.OverlayValues[115] = d115
		ps700.OverlayValues[116] = d116
		ps700.OverlayValues[117] = d117
		ps700.OverlayValues[118] = d118
		ps700.OverlayValues[119] = d119
		ps700.OverlayValues[120] = d120
		ps700.OverlayValues[121] = d121
		ps700.OverlayValues[122] = d122
		ps700.OverlayValues[123] = d123
		ps700.OverlayValues[124] = d124
		ps700.OverlayValues[125] = d125
		ps700.OverlayValues[126] = d126
		ps700.OverlayValues[127] = d127
		ps700.OverlayValues[128] = d128
		ps700.OverlayValues[129] = d129
		ps700.OverlayValues[130] = d130
		ps700.OverlayValues[131] = d131
		ps700.OverlayValues[132] = d132
		ps700.OverlayValues[133] = d133
		ps700.OverlayValues[134] = d134
		ps700.OverlayValues[135] = d135
		ps700.OverlayValues[136] = d136
		ps700.OverlayValues[137] = d137
		ps700.OverlayValues[140] = d140
		ps700.OverlayValues[225] = d225
		ps700.OverlayValues[226] = d226
		ps700.OverlayValues[227] = d227
		ps700.OverlayValues[228] = d228
		ps700.OverlayValues[230] = d230
		ps700.OverlayValues[231] = d231
		ps700.OverlayValues[232] = d232
		ps700.OverlayValues[233] = d233
		ps700.OverlayValues[234] = d234
		ps700.OverlayValues[235] = d235
		ps700.OverlayValues[236] = d236
		ps700.OverlayValues[237] = d237
		ps700.OverlayValues[239] = d239
		ps700.OverlayValues[241] = d241
		ps700.OverlayValues[242] = d242
		ps700.OverlayValues[243] = d243
		ps700.OverlayValues[244] = d244
		ps700.OverlayValues[245] = d245
		ps700.OverlayValues[248] = d248
		ps700.OverlayValues[350] = d350
		ps700.OverlayValues[351] = d351
		ps700.OverlayValues[352] = d352
		ps700.OverlayValues[353] = d353
		ps700.OverlayValues[354] = d354
		ps700.OverlayValues[356] = d356
		ps700.OverlayValues[357] = d357
		ps700.OverlayValues[358] = d358
		ps700.OverlayValues[359] = d359
		ps700.OverlayValues[360] = d360
		ps700.OverlayValues[361] = d361
		ps700.OverlayValues[362] = d362
		ps700.OverlayValues[363] = d363
		ps700.OverlayValues[364] = d364
		ps700.OverlayValues[365] = d365
		ps700.OverlayValues[366] = d366
		ps700.OverlayValues[367] = d367
		ps700.OverlayValues[368] = d368
		ps700.OverlayValues[369] = d369
		ps700.OverlayValues[370] = d370
		ps700.OverlayValues[371] = d371
		ps700.OverlayValues[372] = d372
		ps700.OverlayValues[373] = d373
		ps700.OverlayValues[374] = d374
		ps700.OverlayValues[375] = d375
		ps700.OverlayValues[376] = d376
		ps700.OverlayValues[377] = d377
		ps700.OverlayValues[378] = d378
		ps700.OverlayValues[379] = d379
		ps700.OverlayValues[380] = d380
		ps700.OverlayValues[381] = d381
		ps700.OverlayValues[382] = d382
		ps700.OverlayValues[383] = d383
		ps700.OverlayValues[384] = d384
		ps700.OverlayValues[524] = d524
		ps700.OverlayValues[525] = d525
		ps700.OverlayValues[526] = d526
		ps700.OverlayValues[528] = d528
		ps700.OverlayValues[529] = d529
		ps700.OverlayValues[530] = d530
		ps700.OverlayValues[531] = d531
		ps700.OverlayValues[532] = d532
		ps700.OverlayValues[533] = d533
		ps700.OverlayValues[534] = d534
		ps700.OverlayValues[536] = d536
		ps700.OverlayValues[538] = d538
		ps700.OverlayValues[539] = d539
		ps700.OverlayValues[540] = d540
		ps700.OverlayValues[541] = d541
		ps700.OverlayValues[544] = d544
		ps700.OverlayValues[696] = d696
		ps700.OverlayValues[697] = d697
		ps700.OverlayValues[698] = d698
		ps700.OverlayValues[699] = d699
		ps700.PhiValues = make([]scm.JITValueDesc, 2)
		d701 = d5
		ps700.PhiValues[0] = d701
		d702 = d7
		ps700.PhiValues[1] = d702
		if ps700.General && bbs[8].Rendered {
			ctx.EmitJmp(lbl9)
			return result
		}
		return bbs[8].RenderPS(ps700)
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
			d52 = ps.OverlayValues[52]
		}
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != scm.LocNone {
			d109 = ps.OverlayValues[109]
		}
		if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != scm.LocNone {
			d110 = ps.OverlayValues[110]
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
		if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != scm.LocNone {
			d140 = ps.OverlayValues[140]
		}
		if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != scm.LocNone {
			d225 = ps.OverlayValues[225]
		}
		if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != scm.LocNone {
			d226 = ps.OverlayValues[226]
		}
		if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
			d227 = ps.OverlayValues[227]
		}
		if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
			d228 = ps.OverlayValues[228]
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
		if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != scm.LocNone {
			d234 = ps.OverlayValues[234]
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
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
		}
		if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != scm.LocNone {
			d241 = ps.OverlayValues[241]
		}
		if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != scm.LocNone {
			d242 = ps.OverlayValues[242]
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
		if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != scm.LocNone {
			d248 = ps.OverlayValues[248]
		}
		if len(ps.OverlayValues) > 350 && ps.OverlayValues[350].Loc != scm.LocNone {
			d350 = ps.OverlayValues[350]
		}
		if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != scm.LocNone {
			d351 = ps.OverlayValues[351]
		}
		if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != scm.LocNone {
			d352 = ps.OverlayValues[352]
		}
		if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != scm.LocNone {
			d353 = ps.OverlayValues[353]
		}
		if len(ps.OverlayValues) > 354 && ps.OverlayValues[354].Loc != scm.LocNone {
			d354 = ps.OverlayValues[354]
		}
		if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != scm.LocNone {
			d356 = ps.OverlayValues[356]
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
		if len(ps.OverlayValues) > 362 && ps.OverlayValues[362].Loc != scm.LocNone {
			d362 = ps.OverlayValues[362]
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
		if len(ps.OverlayValues) > 524 && ps.OverlayValues[524].Loc != scm.LocNone {
			d524 = ps.OverlayValues[524]
		}
		if len(ps.OverlayValues) > 525 && ps.OverlayValues[525].Loc != scm.LocNone {
			d525 = ps.OverlayValues[525]
		}
		if len(ps.OverlayValues) > 526 && ps.OverlayValues[526].Loc != scm.LocNone {
			d526 = ps.OverlayValues[526]
		}
		if len(ps.OverlayValues) > 528 && ps.OverlayValues[528].Loc != scm.LocNone {
			d528 = ps.OverlayValues[528]
		}
		if len(ps.OverlayValues) > 529 && ps.OverlayValues[529].Loc != scm.LocNone {
			d529 = ps.OverlayValues[529]
		}
		if len(ps.OverlayValues) > 530 && ps.OverlayValues[530].Loc != scm.LocNone {
			d530 = ps.OverlayValues[530]
		}
		if len(ps.OverlayValues) > 531 && ps.OverlayValues[531].Loc != scm.LocNone {
			d531 = ps.OverlayValues[531]
		}
		if len(ps.OverlayValues) > 532 && ps.OverlayValues[532].Loc != scm.LocNone {
			d532 = ps.OverlayValues[532]
		}
		if len(ps.OverlayValues) > 533 && ps.OverlayValues[533].Loc != scm.LocNone {
			d533 = ps.OverlayValues[533]
		}
		if len(ps.OverlayValues) > 534 && ps.OverlayValues[534].Loc != scm.LocNone {
			d534 = ps.OverlayValues[534]
		}
		if len(ps.OverlayValues) > 536 && ps.OverlayValues[536].Loc != scm.LocNone {
			d536 = ps.OverlayValues[536]
		}
		if len(ps.OverlayValues) > 538 && ps.OverlayValues[538].Loc != scm.LocNone {
			d538 = ps.OverlayValues[538]
		}
		if len(ps.OverlayValues) > 539 && ps.OverlayValues[539].Loc != scm.LocNone {
			d539 = ps.OverlayValues[539]
		}
		if len(ps.OverlayValues) > 540 && ps.OverlayValues[540].Loc != scm.LocNone {
			d540 = ps.OverlayValues[540]
		}
		if len(ps.OverlayValues) > 541 && ps.OverlayValues[541].Loc != scm.LocNone {
			d541 = ps.OverlayValues[541]
		}
		if len(ps.OverlayValues) > 544 && ps.OverlayValues[544].Loc != scm.LocNone {
			d544 = ps.OverlayValues[544]
		}
		if len(ps.OverlayValues) > 696 && ps.OverlayValues[696].Loc != scm.LocNone {
			d696 = ps.OverlayValues[696]
		}
		if len(ps.OverlayValues) > 697 && ps.OverlayValues[697].Loc != scm.LocNone {
			d697 = ps.OverlayValues[697]
		}
		if len(ps.OverlayValues) > 698 && ps.OverlayValues[698].Loc != scm.LocNone {
			d698 = ps.OverlayValues[698]
		}
		if len(ps.OverlayValues) > 699 && ps.OverlayValues[699].Loc != scm.LocNone {
			d699 = ps.OverlayValues[699]
		}
		if len(ps.OverlayValues) > 701 && ps.OverlayValues[701].Loc != scm.LocNone {
			d701 = ps.OverlayValues[701]
		}
		if len(ps.OverlayValues) > 702 && ps.OverlayValues[702].Loc != scm.LocNone {
			d702 = ps.OverlayValues[702]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d8)
		ctx.EnsureDesc(&d9)
		ctx.EnsureDescsTogether(&d8, &d9)
		var d703 scm.JITValueDesc
		if d8.Loc == scm.LocImm && d9.Loc == scm.LocImm {
			d703 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() + d9.Imm.Int())}
		} else if d9.Loc == scm.LocImm && d9.Imm.Int() == 0 {
			r117 := ctx.AllocRegExcept(d8.Reg)
			ctx.EmitMovRegReg(r117, d8.Reg)
			d703 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
			ctx.BindReg(r117, &d703)
		} else if d8.Loc == scm.LocImm && d8.Imm.Int() == 0 {
			d703 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d9.Reg}
			ctx.BindReg(d9.Reg, &d703)
		} else if d8.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d9.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d8.Imm.Int()))
			ctx.EmitAddInt64(scratch, d9.Reg)
			d703 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d703)
		} else if d9.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d8.Reg)
			ctx.EmitMovRegReg(scratch, d8.Reg)
			if d9.Imm.Int() >= -2147483648 && d9.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d9.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d9.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d703 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d703)
		} else {
			r118 := ctx.AllocRegExcept(d8.Reg, d9.Reg)
			ctx.EmitMovRegReg(r118, d8.Reg)
			ctx.EmitAddInt64(r118, d9.Reg)
			d703 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r118}
			ctx.BindReg(r118, &d703)
		}
		if d703.Loc == scm.LocImm {
			d703 = scm.JITValueDesc{Loc: scm.LocImm, Type: d703.Type, Imm: scm.NewInt(int64(uint64(d703.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d703.Reg, 32)
			ctx.EmitShrRegImm8(d703.Reg, 32)
		}
		if d703.Loc == scm.LocReg && d8.Loc == scm.LocReg && d703.Reg == d8.Reg {
			ctx.TransferReg(d8.Reg)
			d8.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d703)
		var d704 scm.JITValueDesc
		if d703.Loc == scm.LocImm {
			d704 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d703.Imm.Int() / 2)}
		} else {
			r119 := ctx.AllocRegExcept(d703.Reg)
			ctx.EmitMovRegReg(r119, d703.Reg)
			ctx.EmitShrRegImm8(r119, 1)
			d704 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
			ctx.BindReg(r119, &d704)
		}
		if d704.Loc == scm.LocImm {
			d704 = scm.JITValueDesc{Loc: scm.LocImm, Type: d704.Type, Imm: scm.NewInt(int64(uint64(d704.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d704.Reg, 32)
			ctx.EmitShrRegImm8(d704.Reg, 32)
		}
		if d704.Loc == scm.LocReg && d703.Loc == scm.LocReg && d704.Reg == d703.Reg {
			ctx.TransferReg(d703.Reg)
			d703.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d704)
		ctx.EmitStoreToStack(d704, int32(bbs[1].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d704)
		ctx.FreeDesc(&d703)
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
			d705 = d8
			if d705.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d705)
			d706 = d705
			if d706.Loc == scm.LocImm {
				d706 = scm.JITValueDesc{Loc: scm.LocImm, Type: d706.Type, Imm: scm.NewInt(int64(uint64(d706.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d706.Reg, 32)
				ctx.EmitShrRegImm8(d706.Reg, 32)
			}
			ctx.EmitStoreToStack(d706, int32(bbs[1].PhiBase)+int32(16))
			d707 = d9
			if d707.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d707)
			d708 = d707
			if d708.Loc == scm.LocImm {
				d708 = scm.JITValueDesc{Loc: scm.LocImm, Type: d708.Type, Imm: scm.NewInt(int64(uint64(d708.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d708.Reg, 32)
				ctx.EmitShrRegImm8(d708.Reg, 32)
			}
			ctx.EmitStoreToStack(d708, int32(bbs[1].PhiBase)+int32(32))
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
		ps709 := scm.PhiState{General: ps.General}
		ps709.OverlayValues = make([]scm.JITValueDesc, 709)
		ps709.OverlayValues[1] = d1
		ps709.OverlayValues[2] = d2
		ps709.OverlayValues[3] = d3
		ps709.OverlayValues[4] = d4
		ps709.OverlayValues[5] = d5
		ps709.OverlayValues[6] = d6
		ps709.OverlayValues[7] = d7
		ps709.OverlayValues[8] = d8
		ps709.OverlayValues[9] = d9
		ps709.OverlayValues[10] = d10
		ps709.OverlayValues[11] = d11
		ps709.OverlayValues[12] = d12
		ps709.OverlayValues[13] = d13
		ps709.OverlayValues[14] = d14
		ps709.OverlayValues[15] = d15
		ps709.OverlayValues[17] = d17
		ps709.OverlayValues[18] = d18
		ps709.OverlayValues[19] = d19
		ps709.OverlayValues[20] = d20
		ps709.OverlayValues[21] = d21
		ps709.OverlayValues[22] = d22
		ps709.OverlayValues[23] = d23
		ps709.OverlayValues[24] = d24
		ps709.OverlayValues[25] = d25
		ps709.OverlayValues[26] = d26
		ps709.OverlayValues[27] = d27
		ps709.OverlayValues[28] = d28
		ps709.OverlayValues[29] = d29
		ps709.OverlayValues[30] = d30
		ps709.OverlayValues[31] = d31
		ps709.OverlayValues[32] = d32
		ps709.OverlayValues[33] = d33
		ps709.OverlayValues[34] = d34
		ps709.OverlayValues[35] = d35
		ps709.OverlayValues[36] = d36
		ps709.OverlayValues[37] = d37
		ps709.OverlayValues[38] = d38
		ps709.OverlayValues[39] = d39
		ps709.OverlayValues[40] = d40
		ps709.OverlayValues[41] = d41
		ps709.OverlayValues[42] = d42
		ps709.OverlayValues[43] = d43
		ps709.OverlayValues[44] = d44
		ps709.OverlayValues[45] = d45
		ps709.OverlayValues[46] = d46
		ps709.OverlayValues[47] = d47
		ps709.OverlayValues[48] = d48
		ps709.OverlayValues[49] = d49
		ps709.OverlayValues[52] = d52
		ps709.OverlayValues[53] = d53
		ps709.OverlayValues[54] = d54
		ps709.OverlayValues[109] = d109
		ps709.OverlayValues[110] = d110
		ps709.OverlayValues[111] = d111
		ps709.OverlayValues[112] = d112
		ps709.OverlayValues[113] = d113
		ps709.OverlayValues[114] = d114
		ps709.OverlayValues[115] = d115
		ps709.OverlayValues[116] = d116
		ps709.OverlayValues[117] = d117
		ps709.OverlayValues[118] = d118
		ps709.OverlayValues[119] = d119
		ps709.OverlayValues[120] = d120
		ps709.OverlayValues[121] = d121
		ps709.OverlayValues[122] = d122
		ps709.OverlayValues[123] = d123
		ps709.OverlayValues[124] = d124
		ps709.OverlayValues[125] = d125
		ps709.OverlayValues[126] = d126
		ps709.OverlayValues[127] = d127
		ps709.OverlayValues[128] = d128
		ps709.OverlayValues[129] = d129
		ps709.OverlayValues[130] = d130
		ps709.OverlayValues[131] = d131
		ps709.OverlayValues[132] = d132
		ps709.OverlayValues[133] = d133
		ps709.OverlayValues[134] = d134
		ps709.OverlayValues[135] = d135
		ps709.OverlayValues[136] = d136
		ps709.OverlayValues[137] = d137
		ps709.OverlayValues[140] = d140
		ps709.OverlayValues[225] = d225
		ps709.OverlayValues[226] = d226
		ps709.OverlayValues[227] = d227
		ps709.OverlayValues[228] = d228
		ps709.OverlayValues[230] = d230
		ps709.OverlayValues[231] = d231
		ps709.OverlayValues[232] = d232
		ps709.OverlayValues[233] = d233
		ps709.OverlayValues[234] = d234
		ps709.OverlayValues[235] = d235
		ps709.OverlayValues[236] = d236
		ps709.OverlayValues[237] = d237
		ps709.OverlayValues[239] = d239
		ps709.OverlayValues[241] = d241
		ps709.OverlayValues[242] = d242
		ps709.OverlayValues[243] = d243
		ps709.OverlayValues[244] = d244
		ps709.OverlayValues[245] = d245
		ps709.OverlayValues[248] = d248
		ps709.OverlayValues[350] = d350
		ps709.OverlayValues[351] = d351
		ps709.OverlayValues[352] = d352
		ps709.OverlayValues[353] = d353
		ps709.OverlayValues[354] = d354
		ps709.OverlayValues[356] = d356
		ps709.OverlayValues[357] = d357
		ps709.OverlayValues[358] = d358
		ps709.OverlayValues[359] = d359
		ps709.OverlayValues[360] = d360
		ps709.OverlayValues[361] = d361
		ps709.OverlayValues[362] = d362
		ps709.OverlayValues[363] = d363
		ps709.OverlayValues[364] = d364
		ps709.OverlayValues[365] = d365
		ps709.OverlayValues[366] = d366
		ps709.OverlayValues[367] = d367
		ps709.OverlayValues[368] = d368
		ps709.OverlayValues[369] = d369
		ps709.OverlayValues[370] = d370
		ps709.OverlayValues[371] = d371
		ps709.OverlayValues[372] = d372
		ps709.OverlayValues[373] = d373
		ps709.OverlayValues[374] = d374
		ps709.OverlayValues[375] = d375
		ps709.OverlayValues[376] = d376
		ps709.OverlayValues[377] = d377
		ps709.OverlayValues[378] = d378
		ps709.OverlayValues[379] = d379
		ps709.OverlayValues[380] = d380
		ps709.OverlayValues[381] = d381
		ps709.OverlayValues[382] = d382
		ps709.OverlayValues[383] = d383
		ps709.OverlayValues[384] = d384
		ps709.OverlayValues[524] = d524
		ps709.OverlayValues[525] = d525
		ps709.OverlayValues[526] = d526
		ps709.OverlayValues[528] = d528
		ps709.OverlayValues[529] = d529
		ps709.OverlayValues[530] = d530
		ps709.OverlayValues[531] = d531
		ps709.OverlayValues[532] = d532
		ps709.OverlayValues[533] = d533
		ps709.OverlayValues[534] = d534
		ps709.OverlayValues[536] = d536
		ps709.OverlayValues[538] = d538
		ps709.OverlayValues[539] = d539
		ps709.OverlayValues[540] = d540
		ps709.OverlayValues[541] = d541
		ps709.OverlayValues[544] = d544
		ps709.OverlayValues[696] = d696
		ps709.OverlayValues[697] = d697
		ps709.OverlayValues[698] = d698
		ps709.OverlayValues[699] = d699
		ps709.OverlayValues[701] = d701
		ps709.OverlayValues[702] = d702
		ps709.OverlayValues[703] = d703
		ps709.OverlayValues[704] = d704
		ps709.OverlayValues[705] = d705
		ps709.OverlayValues[706] = d706
		ps709.OverlayValues[707] = d707
		ps709.OverlayValues[708] = d708
		ps709.PhiValues = make([]scm.JITValueDesc, 3)
		d710 = d8
		ps709.PhiValues[1] = d710
		d711 = d9
		ps709.PhiValues[2] = d711
		if ps709.General && bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		return bbs[1].RenderPS(ps709)
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
			d52 = ps.OverlayValues[52]
		}
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != scm.LocNone {
			d109 = ps.OverlayValues[109]
		}
		if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != scm.LocNone {
			d110 = ps.OverlayValues[110]
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
		if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != scm.LocNone {
			d140 = ps.OverlayValues[140]
		}
		if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != scm.LocNone {
			d225 = ps.OverlayValues[225]
		}
		if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != scm.LocNone {
			d226 = ps.OverlayValues[226]
		}
		if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
			d227 = ps.OverlayValues[227]
		}
		if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
			d228 = ps.OverlayValues[228]
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
		if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != scm.LocNone {
			d234 = ps.OverlayValues[234]
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
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
		}
		if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != scm.LocNone {
			d241 = ps.OverlayValues[241]
		}
		if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != scm.LocNone {
			d242 = ps.OverlayValues[242]
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
		if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != scm.LocNone {
			d248 = ps.OverlayValues[248]
		}
		if len(ps.OverlayValues) > 350 && ps.OverlayValues[350].Loc != scm.LocNone {
			d350 = ps.OverlayValues[350]
		}
		if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != scm.LocNone {
			d351 = ps.OverlayValues[351]
		}
		if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != scm.LocNone {
			d352 = ps.OverlayValues[352]
		}
		if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != scm.LocNone {
			d353 = ps.OverlayValues[353]
		}
		if len(ps.OverlayValues) > 354 && ps.OverlayValues[354].Loc != scm.LocNone {
			d354 = ps.OverlayValues[354]
		}
		if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != scm.LocNone {
			d356 = ps.OverlayValues[356]
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
		if len(ps.OverlayValues) > 362 && ps.OverlayValues[362].Loc != scm.LocNone {
			d362 = ps.OverlayValues[362]
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
		if len(ps.OverlayValues) > 524 && ps.OverlayValues[524].Loc != scm.LocNone {
			d524 = ps.OverlayValues[524]
		}
		if len(ps.OverlayValues) > 525 && ps.OverlayValues[525].Loc != scm.LocNone {
			d525 = ps.OverlayValues[525]
		}
		if len(ps.OverlayValues) > 526 && ps.OverlayValues[526].Loc != scm.LocNone {
			d526 = ps.OverlayValues[526]
		}
		if len(ps.OverlayValues) > 528 && ps.OverlayValues[528].Loc != scm.LocNone {
			d528 = ps.OverlayValues[528]
		}
		if len(ps.OverlayValues) > 529 && ps.OverlayValues[529].Loc != scm.LocNone {
			d529 = ps.OverlayValues[529]
		}
		if len(ps.OverlayValues) > 530 && ps.OverlayValues[530].Loc != scm.LocNone {
			d530 = ps.OverlayValues[530]
		}
		if len(ps.OverlayValues) > 531 && ps.OverlayValues[531].Loc != scm.LocNone {
			d531 = ps.OverlayValues[531]
		}
		if len(ps.OverlayValues) > 532 && ps.OverlayValues[532].Loc != scm.LocNone {
			d532 = ps.OverlayValues[532]
		}
		if len(ps.OverlayValues) > 533 && ps.OverlayValues[533].Loc != scm.LocNone {
			d533 = ps.OverlayValues[533]
		}
		if len(ps.OverlayValues) > 534 && ps.OverlayValues[534].Loc != scm.LocNone {
			d534 = ps.OverlayValues[534]
		}
		if len(ps.OverlayValues) > 536 && ps.OverlayValues[536].Loc != scm.LocNone {
			d536 = ps.OverlayValues[536]
		}
		if len(ps.OverlayValues) > 538 && ps.OverlayValues[538].Loc != scm.LocNone {
			d538 = ps.OverlayValues[538]
		}
		if len(ps.OverlayValues) > 539 && ps.OverlayValues[539].Loc != scm.LocNone {
			d539 = ps.OverlayValues[539]
		}
		if len(ps.OverlayValues) > 540 && ps.OverlayValues[540].Loc != scm.LocNone {
			d540 = ps.OverlayValues[540]
		}
		if len(ps.OverlayValues) > 541 && ps.OverlayValues[541].Loc != scm.LocNone {
			d541 = ps.OverlayValues[541]
		}
		if len(ps.OverlayValues) > 544 && ps.OverlayValues[544].Loc != scm.LocNone {
			d544 = ps.OverlayValues[544]
		}
		if len(ps.OverlayValues) > 696 && ps.OverlayValues[696].Loc != scm.LocNone {
			d696 = ps.OverlayValues[696]
		}
		if len(ps.OverlayValues) > 697 && ps.OverlayValues[697].Loc != scm.LocNone {
			d697 = ps.OverlayValues[697]
		}
		if len(ps.OverlayValues) > 698 && ps.OverlayValues[698].Loc != scm.LocNone {
			d698 = ps.OverlayValues[698]
		}
		if len(ps.OverlayValues) > 699 && ps.OverlayValues[699].Loc != scm.LocNone {
			d699 = ps.OverlayValues[699]
		}
		if len(ps.OverlayValues) > 701 && ps.OverlayValues[701].Loc != scm.LocNone {
			d701 = ps.OverlayValues[701]
		}
		if len(ps.OverlayValues) > 702 && ps.OverlayValues[702].Loc != scm.LocNone {
			d702 = ps.OverlayValues[702]
		}
		if len(ps.OverlayValues) > 703 && ps.OverlayValues[703].Loc != scm.LocNone {
			d703 = ps.OverlayValues[703]
		}
		if len(ps.OverlayValues) > 704 && ps.OverlayValues[704].Loc != scm.LocNone {
			d704 = ps.OverlayValues[704]
		}
		if len(ps.OverlayValues) > 705 && ps.OverlayValues[705].Loc != scm.LocNone {
			d705 = ps.OverlayValues[705]
		}
		if len(ps.OverlayValues) > 706 && ps.OverlayValues[706].Loc != scm.LocNone {
			d706 = ps.OverlayValues[706]
		}
		if len(ps.OverlayValues) > 707 && ps.OverlayValues[707].Loc != scm.LocNone {
			d707 = ps.OverlayValues[707]
		}
		if len(ps.OverlayValues) > 708 && ps.OverlayValues[708].Loc != scm.LocNone {
			d708 = ps.OverlayValues[708]
		}
		if len(ps.OverlayValues) > 710 && ps.OverlayValues[710].Loc != scm.LocNone {
			d710 = ps.OverlayValues[710]
		}
		if len(ps.OverlayValues) > 711 && ps.OverlayValues[711].Loc != scm.LocNone {
			d711 = ps.OverlayValues[711]
		}
		ctx.ReclaimUntrackedRegs()
		d712 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d713 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d713)
		ctx.BindReg(r1, &d713)
		ctx.EnsureDesc(&d712)
		if d712.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d712, &d713)
		} else {
			switch d712.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d713, d712)
			case scm.TagInt:
				ctx.EmitMakeInt(d713, d712)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d713, d712)
			case scm.TagNil:
				ctx.EmitMakeNil(d713)
			default:
				ctx.EmitMovPairToResult(&d712, &d713)
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
			d52 = ps.OverlayValues[52]
		}
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != scm.LocNone {
			d109 = ps.OverlayValues[109]
		}
		if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != scm.LocNone {
			d110 = ps.OverlayValues[110]
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
		if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != scm.LocNone {
			d140 = ps.OverlayValues[140]
		}
		if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != scm.LocNone {
			d225 = ps.OverlayValues[225]
		}
		if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != scm.LocNone {
			d226 = ps.OverlayValues[226]
		}
		if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
			d227 = ps.OverlayValues[227]
		}
		if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
			d228 = ps.OverlayValues[228]
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
		if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != scm.LocNone {
			d234 = ps.OverlayValues[234]
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
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
		}
		if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != scm.LocNone {
			d241 = ps.OverlayValues[241]
		}
		if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != scm.LocNone {
			d242 = ps.OverlayValues[242]
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
		if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != scm.LocNone {
			d248 = ps.OverlayValues[248]
		}
		if len(ps.OverlayValues) > 350 && ps.OverlayValues[350].Loc != scm.LocNone {
			d350 = ps.OverlayValues[350]
		}
		if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != scm.LocNone {
			d351 = ps.OverlayValues[351]
		}
		if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != scm.LocNone {
			d352 = ps.OverlayValues[352]
		}
		if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != scm.LocNone {
			d353 = ps.OverlayValues[353]
		}
		if len(ps.OverlayValues) > 354 && ps.OverlayValues[354].Loc != scm.LocNone {
			d354 = ps.OverlayValues[354]
		}
		if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != scm.LocNone {
			d356 = ps.OverlayValues[356]
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
		if len(ps.OverlayValues) > 362 && ps.OverlayValues[362].Loc != scm.LocNone {
			d362 = ps.OverlayValues[362]
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
		if len(ps.OverlayValues) > 524 && ps.OverlayValues[524].Loc != scm.LocNone {
			d524 = ps.OverlayValues[524]
		}
		if len(ps.OverlayValues) > 525 && ps.OverlayValues[525].Loc != scm.LocNone {
			d525 = ps.OverlayValues[525]
		}
		if len(ps.OverlayValues) > 526 && ps.OverlayValues[526].Loc != scm.LocNone {
			d526 = ps.OverlayValues[526]
		}
		if len(ps.OverlayValues) > 528 && ps.OverlayValues[528].Loc != scm.LocNone {
			d528 = ps.OverlayValues[528]
		}
		if len(ps.OverlayValues) > 529 && ps.OverlayValues[529].Loc != scm.LocNone {
			d529 = ps.OverlayValues[529]
		}
		if len(ps.OverlayValues) > 530 && ps.OverlayValues[530].Loc != scm.LocNone {
			d530 = ps.OverlayValues[530]
		}
		if len(ps.OverlayValues) > 531 && ps.OverlayValues[531].Loc != scm.LocNone {
			d531 = ps.OverlayValues[531]
		}
		if len(ps.OverlayValues) > 532 && ps.OverlayValues[532].Loc != scm.LocNone {
			d532 = ps.OverlayValues[532]
		}
		if len(ps.OverlayValues) > 533 && ps.OverlayValues[533].Loc != scm.LocNone {
			d533 = ps.OverlayValues[533]
		}
		if len(ps.OverlayValues) > 534 && ps.OverlayValues[534].Loc != scm.LocNone {
			d534 = ps.OverlayValues[534]
		}
		if len(ps.OverlayValues) > 536 && ps.OverlayValues[536].Loc != scm.LocNone {
			d536 = ps.OverlayValues[536]
		}
		if len(ps.OverlayValues) > 538 && ps.OverlayValues[538].Loc != scm.LocNone {
			d538 = ps.OverlayValues[538]
		}
		if len(ps.OverlayValues) > 539 && ps.OverlayValues[539].Loc != scm.LocNone {
			d539 = ps.OverlayValues[539]
		}
		if len(ps.OverlayValues) > 540 && ps.OverlayValues[540].Loc != scm.LocNone {
			d540 = ps.OverlayValues[540]
		}
		if len(ps.OverlayValues) > 541 && ps.OverlayValues[541].Loc != scm.LocNone {
			d541 = ps.OverlayValues[541]
		}
		if len(ps.OverlayValues) > 544 && ps.OverlayValues[544].Loc != scm.LocNone {
			d544 = ps.OverlayValues[544]
		}
		if len(ps.OverlayValues) > 696 && ps.OverlayValues[696].Loc != scm.LocNone {
			d696 = ps.OverlayValues[696]
		}
		if len(ps.OverlayValues) > 697 && ps.OverlayValues[697].Loc != scm.LocNone {
			d697 = ps.OverlayValues[697]
		}
		if len(ps.OverlayValues) > 698 && ps.OverlayValues[698].Loc != scm.LocNone {
			d698 = ps.OverlayValues[698]
		}
		if len(ps.OverlayValues) > 699 && ps.OverlayValues[699].Loc != scm.LocNone {
			d699 = ps.OverlayValues[699]
		}
		if len(ps.OverlayValues) > 701 && ps.OverlayValues[701].Loc != scm.LocNone {
			d701 = ps.OverlayValues[701]
		}
		if len(ps.OverlayValues) > 702 && ps.OverlayValues[702].Loc != scm.LocNone {
			d702 = ps.OverlayValues[702]
		}
		if len(ps.OverlayValues) > 703 && ps.OverlayValues[703].Loc != scm.LocNone {
			d703 = ps.OverlayValues[703]
		}
		if len(ps.OverlayValues) > 704 && ps.OverlayValues[704].Loc != scm.LocNone {
			d704 = ps.OverlayValues[704]
		}
		if len(ps.OverlayValues) > 705 && ps.OverlayValues[705].Loc != scm.LocNone {
			d705 = ps.OverlayValues[705]
		}
		if len(ps.OverlayValues) > 706 && ps.OverlayValues[706].Loc != scm.LocNone {
			d706 = ps.OverlayValues[706]
		}
		if len(ps.OverlayValues) > 707 && ps.OverlayValues[707].Loc != scm.LocNone {
			d707 = ps.OverlayValues[707]
		}
		if len(ps.OverlayValues) > 708 && ps.OverlayValues[708].Loc != scm.LocNone {
			d708 = ps.OverlayValues[708]
		}
		if len(ps.OverlayValues) > 710 && ps.OverlayValues[710].Loc != scm.LocNone {
			d710 = ps.OverlayValues[710]
		}
		if len(ps.OverlayValues) > 711 && ps.OverlayValues[711].Loc != scm.LocNone {
			d711 = ps.OverlayValues[711]
		}
		if len(ps.OverlayValues) > 712 && ps.OverlayValues[712].Loc != scm.LocNone {
			d712 = ps.OverlayValues[712]
		}
		if len(ps.OverlayValues) > 713 && ps.OverlayValues[713].Loc != scm.LocNone {
			d713 = ps.OverlayValues[713]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d4)
		d714 = d4
		_ = d714
		ctx.StabilizeDescForControlFlow(&d714)
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
		ctx.EnsureDesc(&d714)
		ctx.EnsureDesc(&d714)
		var d715 scm.JITValueDesc
		if d714.Loc == scm.LocImm {
			d715 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d714.Imm.Int()))))}
		} else {
			r120 := ctx.AllocReg()
			ctx.EmitMovRegReg(r120, d714.Reg)
			ctx.EmitShlRegImm8(r120, 32)
			ctx.EmitShrRegImm8(r120, 32)
			d715 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r120}
			ctx.BindReg(r120, &d715)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d716 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
			r121 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r121, fieldAddr)
			d716 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r121}
			ctx.BindReg(r121, &d716)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
			r122 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r122, thisptr.Reg, off)
			d716 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r122}
			ctx.BindReg(r122, &d716)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d716)
		ctx.EnsureDesc(&d716)
		var d717 scm.JITValueDesc
		if d716.Loc == scm.LocImm {
			d717 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d716.Imm.Int()))))}
		} else {
			r123 := ctx.AllocReg()
			ctx.EmitMovRegReg(r123, d716.Reg)
			ctx.EmitShlRegImm8(r123, 56)
			ctx.EmitShrRegImm8(r123, 56)
			d717 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
			ctx.BindReg(r123, &d717)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d715)
		ctx.EnsureDesc(&d717)
		ctx.EnsureDescsTogether(&d715, &d717)
		var d718 scm.JITValueDesc
		if d715.Loc == scm.LocImm && d717.Loc == scm.LocImm {
			d718 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d715.Imm.Int() * d717.Imm.Int())}
		} else if d715.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d717.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d715.Imm.Int()))
			ctx.EmitImulInt64(scratch, d717.Reg)
			d718 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d718)
		} else if d717.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d715.Reg)
			ctx.EmitMovRegReg(scratch, d715.Reg)
			if d717.Imm.Int() >= -2147483648 && d717.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d717.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d717.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d718 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d718)
		} else {
			r124 := ctx.AllocRegExcept(d715.Reg, d717.Reg)
			ctx.EmitMovRegReg(r124, d715.Reg)
			ctx.EmitImulInt64(r124, d717.Reg)
			d718 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
			ctx.BindReg(r124, &d718)
		}
		if d718.Loc == scm.LocReg && d715.Loc == scm.LocReg && d718.Reg == d715.Reg {
			ctx.TransferReg(d715.Reg)
			d715.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d715)
		ctx.FreeDesc(&d717)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d718)
		var d719 scm.JITValueDesc
		if d718.Loc == scm.LocImm {
			d719 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d718.Imm.Int() / 64)}
		} else {
			r125 := ctx.AllocRegExcept(d718.Reg)
			ctx.EmitMovRegReg(r125, d718.Reg)
			ctx.EmitShrRegImm8(r125, 6)
			d719 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
			ctx.BindReg(r125, &d719)
		}
		if d719.Loc == scm.LocReg && d718.Loc == scm.LocReg && d719.Reg == d718.Reg {
			ctx.TransferReg(d718.Reg)
			d718.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d718)
		var d720 scm.JITValueDesc
		if d718.Loc == scm.LocImm {
			d720 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d718.Imm.Int() % 64)}
		} else {
			r126 := ctx.AllocRegExcept(d718.Reg)
			ctx.EmitMovRegReg(r126, d718.Reg)
			ctx.EmitAndRegImm32(r126, 63)
			d720 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r126}
			ctx.BindReg(r126, &d720)
		}
		if d720.Loc == scm.LocReg && d718.Loc == scm.LocReg && d720.Reg == d718.Reg {
			ctx.TransferReg(d718.Reg)
			d718.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d718)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d721 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 0
			r127 := ctx.AllocReg()
			r128 := ctx.AllocRegExcept(r127)
			r129 := ctx.AllocRegExcept(r127, r128)
			ctx.EmitMovRegMem64(r127, fieldAddr)
			ctx.EmitMovRegMem64(r128, fieldAddr+8)
			ctx.EmitMovRegMem64(r129, fieldAddr+16)
			d721 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r127, Reg2: r128, Reg3: r129}
			ctx.BindReg(r127, &d721)
			ctx.BindReg(r128, &d721)
			ctx.BindReg(r129, &d721)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 0)
			r130 := ctx.AllocReg()
			r131 := ctx.AllocRegExcept(r130)
			r132 := ctx.AllocRegExcept(r130, r131)
			ctx.EmitMovRegMem(r130, thisptr.Reg, off)
			ctx.EmitMovRegMem(r131, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r132, thisptr.Reg, off+16)
			d721 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r130, Reg2: r131, Reg3: r132}
			ctx.BindReg(r130, &d721)
			ctx.BindReg(r131, &d721)
			ctx.BindReg(r132, &d721)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d719)
		ctx.ReclaimUntrackedRegs()
		d723 = ctx.EmitSliceElementAddress(&d721, &d719, 8)
		ctx.EnsureDesc(&d723)
		ctx.EmitMovRegMem(d723.Reg, d723.Reg, 0)
		d722 = d723
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d722)
		ctx.EnsureDesc(&d720)
		var d724 scm.JITValueDesc
		if d722.Loc == scm.LocImm && d720.Loc == scm.LocImm {
			d724 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d722.Imm.Int()) << uint64(d720.Imm.Int())))}
		} else if d720.Loc == scm.LocImm {
			r133 := ctx.AllocRegExcept(d722.Reg)
			ctx.EmitMovRegReg(r133, d722.Reg)
			ctx.EmitShlRegImm8(r133, uint8(d720.Imm.Int()))
			d724 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
			ctx.BindReg(r133, &d724)
		} else {
			{
				shiftSrc := d722.Reg
				r134 := ctx.AllocRegExcept(d722.Reg)
				ctx.EmitMovRegReg(r134, d722.Reg)
				shiftSrc = r134
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d720.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d720.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d720.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d724 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d724)
			}
		}
		if d724.Loc == scm.LocReg && d722.Loc == scm.LocReg && d724.Reg == d722.Reg {
			ctx.TransferReg(d722.Reg)
			d722.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d722)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d719)
		ctx.EnsureDesc(&d719)
		var d725 scm.JITValueDesc
		if d719.Loc == scm.LocImm {
			d725 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d719.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d719.Reg)
			ctx.EmitMovRegReg(scratch, d719.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d725 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d725)
		}
		if d725.Loc == scm.LocReg && d719.Loc == scm.LocReg && d725.Reg == d719.Reg {
			ctx.TransferReg(d719.Reg)
			d719.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d719)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d725)
		ctx.ReclaimUntrackedRegs()
		d727 = ctx.EmitSliceElementAddress(&d721, &d725, 8)
		ctx.EnsureDesc(&d727)
		ctx.EmitMovRegMem(d727.Reg, d727.Reg, 0)
		d726 = d727
		ctx.FreeDesc(&d725)
		ctx.ReclaimUntrackedRegs()
		d728 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d720)
		ctx.EnsureDescsTogether(&d728, &d720)
		var d729 scm.JITValueDesc
		if d728.Loc == scm.LocImm && d720.Loc == scm.LocImm {
			d729 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d728.Imm.Int() - d720.Imm.Int())}
		} else if d720.Loc == scm.LocImm && d720.Imm.Int() == 0 {
			r135 := ctx.AllocRegExcept(d728.Reg)
			ctx.EmitMovRegReg(r135, d728.Reg)
			d729 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r135}
			ctx.BindReg(r135, &d729)
		} else if d728.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d720.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d728.Imm.Int()))
			ctx.EmitSubInt64(scratch, d720.Reg)
			d729 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d729)
		} else if d720.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d728.Reg)
			ctx.EmitMovRegReg(scratch, d728.Reg)
			if d720.Imm.Int() >= -2147483648 && d720.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d720.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d720.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d729 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d729)
		} else {
			r136 := ctx.AllocRegExcept(d728.Reg, d720.Reg)
			ctx.EmitMovRegReg(r136, d728.Reg)
			ctx.EmitSubInt64(r136, d720.Reg)
			d729 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r136}
			ctx.BindReg(r136, &d729)
		}
		if d729.Loc == scm.LocReg && d728.Loc == scm.LocReg && d729.Reg == d728.Reg {
			ctx.TransferReg(d728.Reg)
			d728.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d720)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d726)
		ctx.EnsureDesc(&d729)
		var d730 scm.JITValueDesc
		if d726.Loc == scm.LocImm && d729.Loc == scm.LocImm {
			d730 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d726.Imm.Int()) >> uint64(d729.Imm.Int())))}
		} else if d729.Loc == scm.LocImm {
			r137 := ctx.AllocRegExcept(d726.Reg)
			ctx.EmitMovRegReg(r137, d726.Reg)
			ctx.EmitShrRegImm8(r137, uint8(d729.Imm.Int()))
			d730 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r137}
			ctx.BindReg(r137, &d730)
		} else {
			{
				shiftSrc := d726.Reg
				r138 := ctx.AllocRegExcept(d726.Reg)
				ctx.EmitMovRegReg(r138, d726.Reg)
				shiftSrc = r138
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d729.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d729.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d729.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d730 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d730)
			}
		}
		if d730.Loc == scm.LocReg && d726.Loc == scm.LocReg && d730.Reg == d726.Reg {
			ctx.TransferReg(d726.Reg)
			d726.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d726)
		ctx.FreeDesc(&d729)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d724)
		ctx.EnsureDesc(&d730)
		var d731 scm.JITValueDesc
		if d724.Loc == scm.LocImm && d730.Loc == scm.LocImm {
			d731 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d724.Imm.Int() | d730.Imm.Int())}
		} else if d724.Loc == scm.LocImm && d724.Imm.Int() == 0 {
			d731 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d730.Reg}
			ctx.BindReg(d730.Reg, &d731)
		} else if d730.Loc == scm.LocImm && d730.Imm.Int() == 0 {
			r139 := ctx.AllocRegExcept(d724.Reg)
			ctx.EmitMovRegReg(r139, d724.Reg)
			d731 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r139}
			ctx.BindReg(r139, &d731)
		} else if d724.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d730.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d724.Imm.Int()))
			ctx.EmitOrInt64(scratch, d730.Reg)
			d731 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d731)
		} else if d730.Loc == scm.LocImm {
			r140 := ctx.AllocRegExcept(d724.Reg)
			ctx.EmitMovRegReg(r140, d724.Reg)
			if d730.Imm.Int() >= -2147483648 && d730.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r140, int32(d730.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d730.Imm.Int()))
				ctx.EmitOrInt64(r140, scm.RegR11)
			}
			d731 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r140}
			ctx.BindReg(r140, &d731)
		} else {
			r141 := ctx.AllocRegExcept(d724.Reg, d730.Reg)
			ctx.EmitMovRegReg(r141, d724.Reg)
			ctx.EmitOrInt64(r141, d730.Reg)
			d731 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
			ctx.BindReg(r141, &d731)
		}
		if d731.Loc == scm.LocReg && d724.Loc == scm.LocReg && d731.Reg == d724.Reg {
			ctx.TransferReg(d724.Reg)
			d724.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d724)
		ctx.FreeDesc(&d730)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d716)
		ctx.EnsureDesc(&d716)
		var d732 scm.JITValueDesc
		if d716.Loc == scm.LocImm {
			d732 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d716.Imm.Int()))))}
		} else {
			r142 := ctx.AllocReg()
			ctx.EmitMovRegReg(r142, d716.Reg)
			ctx.EmitShlRegImm8(r142, 56)
			ctx.EmitShrRegImm8(r142, 56)
			d732 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r142}
			ctx.BindReg(r142, &d732)
		}
		ctx.ReclaimUntrackedRegs()
		d733 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d732)
		ctx.EnsureDescsTogether(&d733, &d732)
		var d734 scm.JITValueDesc
		if d733.Loc == scm.LocImm && d732.Loc == scm.LocImm {
			d734 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d733.Imm.Int() - d732.Imm.Int())}
		} else if d732.Loc == scm.LocImm && d732.Imm.Int() == 0 {
			r143 := ctx.AllocRegExcept(d733.Reg)
			ctx.EmitMovRegReg(r143, d733.Reg)
			d734 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r143}
			ctx.BindReg(r143, &d734)
		} else if d733.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d732.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d733.Imm.Int()))
			ctx.EmitSubInt64(scratch, d732.Reg)
			d734 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d734)
		} else if d732.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d733.Reg)
			ctx.EmitMovRegReg(scratch, d733.Reg)
			if d732.Imm.Int() >= -2147483648 && d732.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d732.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d732.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d734 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d734)
		} else {
			r144 := ctx.AllocRegExcept(d733.Reg, d732.Reg)
			ctx.EmitMovRegReg(r144, d733.Reg)
			ctx.EmitSubInt64(r144, d732.Reg)
			d734 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
			ctx.BindReg(r144, &d734)
		}
		if d734.Loc == scm.LocReg && d733.Loc == scm.LocReg && d734.Reg == d733.Reg {
			ctx.TransferReg(d733.Reg)
			d733.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d732)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d731)
		ctx.EnsureDesc(&d734)
		var d735 scm.JITValueDesc
		if d731.Loc == scm.LocImm && d734.Loc == scm.LocImm {
			d735 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d731.Imm.Int()) >> uint64(d734.Imm.Int())))}
		} else if d734.Loc == scm.LocImm {
			r145 := ctx.AllocRegExcept(d731.Reg)
			ctx.EmitMovRegReg(r145, d731.Reg)
			ctx.EmitShrRegImm8(r145, uint8(d734.Imm.Int()))
			d735 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r145}
			ctx.BindReg(r145, &d735)
		} else {
			{
				shiftSrc := d731.Reg
				r146 := ctx.AllocRegExcept(d731.Reg)
				ctx.EmitMovRegReg(r146, d731.Reg)
				shiftSrc = r146
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d734.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d734.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d734.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d735 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d735)
			}
		}
		if d735.Loc == scm.LocReg && d731.Loc == scm.LocReg && d735.Reg == d731.Reg {
			ctx.TransferReg(d731.Reg)
			d731.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d731)
		ctx.FreeDesc(&d734)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d735)
		ctx.EnsureDesc(&d735)
		ctx.EnsureDesc(&d735)
		var d736 scm.JITValueDesc
		if d735.Loc == scm.LocImm {
			d736 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d735.Imm.Int()))))}
		} else {
			r147 := ctx.AllocReg()
			ctx.EmitMovRegReg(r147, d735.Reg)
			d736 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r147}
			ctx.BindReg(r147, &d736)
		}
		ctx.FreeDesc(&d735)
		var d737 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 32
			r148 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r148, fieldAddr)
			d737 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r148}
			ctx.BindReg(r148, &d737)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 32)
			r149 := ctx.AllocReg()
			ctx.EmitMovRegMem(r149, thisptr.Reg, off)
			d737 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r149}
			ctx.BindReg(r149, &d737)
		}
		ctx.EnsureDesc(&d736)
		ctx.EnsureDesc(&d737)
		ctx.EnsureDescsTogether(&d736, &d737)
		var d738 scm.JITValueDesc
		if d736.Loc == scm.LocImm && d737.Loc == scm.LocImm {
			d738 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d736.Imm.Int() + d737.Imm.Int())}
		} else if d737.Loc == scm.LocImm && d737.Imm.Int() == 0 {
			r150 := ctx.AllocRegExcept(d736.Reg)
			ctx.EmitMovRegReg(r150, d736.Reg)
			d738 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r150}
			ctx.BindReg(r150, &d738)
		} else if d736.Loc == scm.LocImm && d736.Imm.Int() == 0 {
			d738 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d737.Reg}
			ctx.BindReg(d737.Reg, &d738)
		} else if d736.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d737.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d736.Imm.Int()))
			ctx.EmitAddInt64(scratch, d737.Reg)
			d738 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d738)
		} else if d737.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d736.Reg)
			ctx.EmitMovRegReg(scratch, d736.Reg)
			if d737.Imm.Int() >= -2147483648 && d737.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d737.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d737.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d738 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d738)
		} else {
			r151 := ctx.AllocRegExcept(d736.Reg, d737.Reg)
			ctx.EmitMovRegReg(r151, d736.Reg)
			ctx.EmitAddInt64(r151, d737.Reg)
			d738 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
			ctx.BindReg(r151, &d738)
		}
		if d738.Loc == scm.LocReg && d736.Loc == scm.LocReg && d738.Reg == d736.Reg {
			ctx.TransferReg(d736.Reg)
			d736.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d736)
		ctx.EnsureDesc(&d4)
		d739 = d4
		_ = d739
		ctx.StabilizeDescForControlFlow(&d739)
		bbpos_5_0 := int32(-1)
		_ = bbpos_5_0
		lbl29 := ctx.ReserveLabel()
		_ = lbl29
		bbpos_5_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl29)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d739)
		ctx.EnsureDesc(&d739)
		var d740 scm.JITValueDesc
		if d739.Loc == scm.LocImm {
			d740 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d739.Imm.Int()))))}
		} else {
			r152 := ctx.AllocReg()
			ctx.EmitMovRegReg(r152, d739.Reg)
			ctx.EmitShlRegImm8(r152, 32)
			ctx.EmitShrRegImm8(r152, 32)
			d740 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r152}
			ctx.BindReg(r152, &d740)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d741 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			r153 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r153, fieldAddr)
			d741 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r153}
			ctx.BindReg(r153, &d741)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			r154 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r154, thisptr.Reg, off)
			d741 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r154}
			ctx.BindReg(r154, &d741)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d741)
		ctx.EnsureDesc(&d741)
		var d742 scm.JITValueDesc
		if d741.Loc == scm.LocImm {
			d742 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d741.Imm.Int()))))}
		} else {
			r155 := ctx.AllocReg()
			ctx.EmitMovRegReg(r155, d741.Reg)
			ctx.EmitShlRegImm8(r155, 56)
			ctx.EmitShrRegImm8(r155, 56)
			d742 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r155}
			ctx.BindReg(r155, &d742)
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
			r156 := ctx.AllocRegExcept(d740.Reg, d742.Reg)
			ctx.EmitMovRegReg(r156, d740.Reg)
			ctx.EmitImulInt64(r156, d742.Reg)
			d743 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r156}
			ctx.BindReg(r156, &d743)
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
			r157 := ctx.AllocRegExcept(d743.Reg)
			ctx.EmitMovRegReg(r157, d743.Reg)
			ctx.EmitShrRegImm8(r157, 6)
			d744 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r157}
			ctx.BindReg(r157, &d744)
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
			r158 := ctx.AllocRegExcept(d743.Reg)
			ctx.EmitMovRegReg(r158, d743.Reg)
			ctx.EmitAndRegImm32(r158, 63)
			d745 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r158}
			ctx.BindReg(r158, &d745)
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
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
			r159 := ctx.AllocReg()
			r160 := ctx.AllocRegExcept(r159)
			r161 := ctx.AllocRegExcept(r159, r160)
			ctx.EmitMovRegMem64(r159, fieldAddr)
			ctx.EmitMovRegMem64(r160, fieldAddr+8)
			ctx.EmitMovRegMem64(r161, fieldAddr+16)
			d746 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r159, Reg2: r160, Reg3: r161}
			ctx.BindReg(r159, &d746)
			ctx.BindReg(r160, &d746)
			ctx.BindReg(r161, &d746)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
			r162 := ctx.AllocReg()
			r163 := ctx.AllocRegExcept(r162)
			r164 := ctx.AllocRegExcept(r162, r163)
			ctx.EmitMovRegMem(r162, thisptr.Reg, off)
			ctx.EmitMovRegMem(r163, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r164, thisptr.Reg, off+16)
			d746 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r162, Reg2: r163, Reg3: r164}
			ctx.BindReg(r162, &d746)
			ctx.BindReg(r163, &d746)
			ctx.BindReg(r164, &d746)
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
			r165 := ctx.AllocRegExcept(d747.Reg)
			ctx.EmitMovRegReg(r165, d747.Reg)
			ctx.EmitShlRegImm8(r165, uint8(d745.Imm.Int()))
			d749 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r165}
			ctx.BindReg(r165, &d749)
		} else {
			{
				shiftSrc := d747.Reg
				r166 := ctx.AllocRegExcept(d747.Reg)
				ctx.EmitMovRegReg(r166, d747.Reg)
				shiftSrc = r166
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
			r167 := ctx.AllocRegExcept(d753.Reg)
			ctx.EmitMovRegReg(r167, d753.Reg)
			d754 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r167}
			ctx.BindReg(r167, &d754)
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
			r168 := ctx.AllocRegExcept(d753.Reg, d745.Reg)
			ctx.EmitMovRegReg(r168, d753.Reg)
			ctx.EmitSubInt64(r168, d745.Reg)
			d754 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r168}
			ctx.BindReg(r168, &d754)
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
			r169 := ctx.AllocRegExcept(d751.Reg)
			ctx.EmitMovRegReg(r169, d751.Reg)
			ctx.EmitShrRegImm8(r169, uint8(d754.Imm.Int()))
			d755 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r169}
			ctx.BindReg(r169, &d755)
		} else {
			{
				shiftSrc := d751.Reg
				r170 := ctx.AllocRegExcept(d751.Reg)
				ctx.EmitMovRegReg(r170, d751.Reg)
				shiftSrc = r170
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
			r171 := ctx.AllocRegExcept(d749.Reg)
			ctx.EmitMovRegReg(r171, d749.Reg)
			d756 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r171}
			ctx.BindReg(r171, &d756)
		} else if d749.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d755.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d749.Imm.Int()))
			ctx.EmitOrInt64(scratch, d755.Reg)
			d756 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d756)
		} else if d755.Loc == scm.LocImm {
			r172 := ctx.AllocRegExcept(d749.Reg)
			ctx.EmitMovRegReg(r172, d749.Reg)
			if d755.Imm.Int() >= -2147483648 && d755.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r172, int32(d755.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d755.Imm.Int()))
				ctx.EmitOrInt64(r172, scm.RegR11)
			}
			d756 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
			ctx.BindReg(r172, &d756)
		} else {
			r173 := ctx.AllocRegExcept(d749.Reg, d755.Reg)
			ctx.EmitMovRegReg(r173, d749.Reg)
			ctx.EmitOrInt64(r173, d755.Reg)
			d756 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r173}
			ctx.BindReg(r173, &d756)
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
			r174 := ctx.AllocReg()
			ctx.EmitMovRegReg(r174, d741.Reg)
			ctx.EmitShlRegImm8(r174, 56)
			ctx.EmitShrRegImm8(r174, 56)
			d757 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r174}
			ctx.BindReg(r174, &d757)
		}
		ctx.ReclaimUntrackedRegs()
		d758 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d757)
		ctx.EnsureDescsTogether(&d758, &d757)
		var d759 scm.JITValueDesc
		if d758.Loc == scm.LocImm && d757.Loc == scm.LocImm {
			d759 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d758.Imm.Int() - d757.Imm.Int())}
		} else if d757.Loc == scm.LocImm && d757.Imm.Int() == 0 {
			r175 := ctx.AllocRegExcept(d758.Reg)
			ctx.EmitMovRegReg(r175, d758.Reg)
			d759 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r175}
			ctx.BindReg(r175, &d759)
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
			r176 := ctx.AllocRegExcept(d758.Reg, d757.Reg)
			ctx.EmitMovRegReg(r176, d758.Reg)
			ctx.EmitSubInt64(r176, d757.Reg)
			d759 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r176}
			ctx.BindReg(r176, &d759)
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
			r177 := ctx.AllocRegExcept(d756.Reg)
			ctx.EmitMovRegReg(r177, d756.Reg)
			ctx.EmitShrRegImm8(r177, uint8(d759.Imm.Int()))
			d760 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r177}
			ctx.BindReg(r177, &d760)
		} else {
			{
				shiftSrc := d756.Reg
				r178 := ctx.AllocRegExcept(d756.Reg)
				ctx.EmitMovRegReg(r178, d756.Reg)
				shiftSrc = r178
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
		ctx.FreeDesc(&d4)
		ctx.EnsureDesc(&d760)
		ctx.EnsureDesc(&d760)
		var d761 scm.JITValueDesc
		if d760.Loc == scm.LocImm {
			d761 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d760.Imm.Int()))))}
		} else {
			r179 := ctx.AllocReg()
			ctx.EmitMovRegReg(r179, d760.Reg)
			d761 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r179}
			ctx.BindReg(r179, &d761)
		}
		ctx.FreeDesc(&d760)
		ctx.EnsureDesc(&d761)
		ctx.EnsureDesc(&d45)
		ctx.EnsureDescsTogether(&d761, &d45)
		var d762 scm.JITValueDesc
		if d761.Loc == scm.LocImm && d45.Loc == scm.LocImm {
			d762 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d761.Imm.Int() + d45.Imm.Int())}
		} else if d45.Loc == scm.LocImm && d45.Imm.Int() == 0 {
			r180 := ctx.AllocRegExcept(d761.Reg)
			ctx.EmitMovRegReg(r180, d761.Reg)
			d762 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r180}
			ctx.BindReg(r180, &d762)
		} else if d761.Loc == scm.LocImm && d761.Imm.Int() == 0 {
			d762 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d45.Reg}
			ctx.BindReg(d45.Reg, &d762)
		} else if d761.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d45.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d761.Imm.Int()))
			ctx.EmitAddInt64(scratch, d45.Reg)
			d762 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d762)
		} else if d45.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d761.Reg)
			ctx.EmitMovRegReg(scratch, d761.Reg)
			if d45.Imm.Int() >= -2147483648 && d45.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d45.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d45.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d762 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d762)
		} else {
			r181 := ctx.AllocRegExcept(d761.Reg, d45.Reg)
			ctx.EmitMovRegReg(r181, d761.Reg)
			ctx.EmitAddInt64(r181, d45.Reg)
			d762 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r181}
			ctx.BindReg(r181, &d762)
		}
		if d762.Loc == scm.LocReg && d761.Loc == scm.LocReg && d762.Reg == d761.Reg {
			ctx.TransferReg(d761.Reg)
			d761.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d761)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		var d763 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm {
			d763 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(idxInt.Imm.Int()))))}
		} else {
			r182 := ctx.AllocReg()
			ctx.EmitMovRegReg(r182, idxInt.Reg)
			ctx.EmitShlRegImm8(r182, 32)
			ctx.EmitShrRegImm8(r182, 32)
			d763 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r182}
			ctx.BindReg(r182, &d763)
		}
		ctx.FreeDesc(&idxInt)
		ctx.EnsureDesc(&d763)
		ctx.EnsureDesc(&d762)
		ctx.EnsureDescsTogether(&d763, &d762)
		var d764 scm.JITValueDesc
		if d763.Loc == scm.LocImm && d762.Loc == scm.LocImm {
			d764 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d763.Imm.Int() - d762.Imm.Int())}
		} else if d762.Loc == scm.LocImm && d762.Imm.Int() == 0 {
			r183 := ctx.AllocRegExcept(d763.Reg)
			ctx.EmitMovRegReg(r183, d763.Reg)
			d764 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r183}
			ctx.BindReg(r183, &d764)
		} else if d763.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d762.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d763.Imm.Int()))
			ctx.EmitSubInt64(scratch, d762.Reg)
			d764 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d764)
		} else if d762.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d763.Reg)
			ctx.EmitMovRegReg(scratch, d763.Reg)
			if d762.Imm.Int() >= -2147483648 && d762.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d762.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d762.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d764 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d764)
		} else {
			r184 := ctx.AllocRegExcept(d763.Reg, d762.Reg)
			ctx.EmitMovRegReg(r184, d763.Reg)
			ctx.EmitSubInt64(r184, d762.Reg)
			d764 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r184}
			ctx.BindReg(r184, &d764)
		}
		if d764.Loc == scm.LocReg && d763.Loc == scm.LocReg && d764.Reg == d763.Reg {
			ctx.TransferReg(d763.Reg)
			d763.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d763)
		ctx.FreeDesc(&d762)
		ctx.EnsureDesc(&d764)
		ctx.EnsureDesc(&d738)
		ctx.EnsureDescsTogether(&d764, &d738)
		var d765 scm.JITValueDesc
		if d764.Loc == scm.LocImm && d738.Loc == scm.LocImm {
			d765 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d764.Imm.Int() * d738.Imm.Int())}
		} else if d764.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d738.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d764.Imm.Int()))
			ctx.EmitImulInt64(scratch, d738.Reg)
			d765 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d765)
		} else if d738.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d764.Reg)
			ctx.EmitMovRegReg(scratch, d764.Reg)
			if d738.Imm.Int() >= -2147483648 && d738.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d738.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d738.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d765 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d765)
		} else {
			r185 := ctx.AllocRegExcept(d764.Reg, d738.Reg)
			ctx.EmitMovRegReg(r185, d764.Reg)
			ctx.EmitImulInt64(r185, d738.Reg)
			d765 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r185}
			ctx.BindReg(r185, &d765)
		}
		if d765.Loc == scm.LocReg && d764.Loc == scm.LocReg && d765.Reg == d764.Reg {
			ctx.TransferReg(d764.Reg)
			d764.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d764)
		ctx.FreeDesc(&d738)
		ctx.EnsureDesc(&d135)
		ctx.EnsureDesc(&d765)
		ctx.EnsureDescsTogether(&d135, &d765)
		var d766 scm.JITValueDesc
		if d135.Loc == scm.LocImm && d765.Loc == scm.LocImm {
			d766 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d135.Imm.Int() + d765.Imm.Int())}
		} else if d765.Loc == scm.LocImm && d765.Imm.Int() == 0 {
			r186 := ctx.AllocRegExcept(d135.Reg)
			ctx.EmitMovRegReg(r186, d135.Reg)
			d766 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r186}
			ctx.BindReg(r186, &d766)
		} else if d135.Loc == scm.LocImm && d135.Imm.Int() == 0 {
			d766 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d765.Reg}
			ctx.BindReg(d765.Reg, &d766)
		} else if d135.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d765.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d135.Imm.Int()))
			ctx.EmitAddInt64(scratch, d765.Reg)
			d766 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d766)
		} else if d765.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d135.Reg)
			ctx.EmitMovRegReg(scratch, d135.Reg)
			if d765.Imm.Int() >= -2147483648 && d765.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d765.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d765.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d766 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d766)
		} else {
			r187 := ctx.AllocRegExcept(d135.Reg, d765.Reg)
			ctx.EmitMovRegReg(r187, d135.Reg)
			ctx.EmitAddInt64(r187, d765.Reg)
			d766 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r187}
			ctx.BindReg(r187, &d766)
		}
		if d766.Loc == scm.LocReg && d135.Loc == scm.LocReg && d766.Reg == d135.Reg {
			ctx.TransferReg(d135.Reg)
			d135.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d765)
		ctx.EnsureDesc(&d766)
		ctx.EnsureDesc(&d766)
		var d767 scm.JITValueDesc
		if d766.Loc == scm.LocImm {
			d767 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d766.Imm.Int()))}
		} else {
			ctx.EmitCvtInt64ToFloat64(scm.RegX0, d766.Reg)
			d767 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d766.Reg}
			ctx.BindReg(d766.Reg, &d767)
		}
		ctx.FreeDesc(&d766)
		ctx.EnsureDesc(&d767)
		d768 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d768)
		ctx.BindReg(r1, &d768)
		ctx.EnsureDesc(&d767)
		ctx.EmitMakeFloat(d768, d767)
		if d767.Loc == scm.LocReg {
			ctx.FreeReg(d767.Reg)
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
			d52 = ps.OverlayValues[52]
		}
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != scm.LocNone {
			d109 = ps.OverlayValues[109]
		}
		if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != scm.LocNone {
			d110 = ps.OverlayValues[110]
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
		if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != scm.LocNone {
			d140 = ps.OverlayValues[140]
		}
		if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != scm.LocNone {
			d225 = ps.OverlayValues[225]
		}
		if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != scm.LocNone {
			d226 = ps.OverlayValues[226]
		}
		if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
			d227 = ps.OverlayValues[227]
		}
		if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
			d228 = ps.OverlayValues[228]
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
		if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != scm.LocNone {
			d234 = ps.OverlayValues[234]
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
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
		}
		if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != scm.LocNone {
			d241 = ps.OverlayValues[241]
		}
		if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != scm.LocNone {
			d242 = ps.OverlayValues[242]
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
		if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != scm.LocNone {
			d248 = ps.OverlayValues[248]
		}
		if len(ps.OverlayValues) > 350 && ps.OverlayValues[350].Loc != scm.LocNone {
			d350 = ps.OverlayValues[350]
		}
		if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != scm.LocNone {
			d351 = ps.OverlayValues[351]
		}
		if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != scm.LocNone {
			d352 = ps.OverlayValues[352]
		}
		if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != scm.LocNone {
			d353 = ps.OverlayValues[353]
		}
		if len(ps.OverlayValues) > 354 && ps.OverlayValues[354].Loc != scm.LocNone {
			d354 = ps.OverlayValues[354]
		}
		if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != scm.LocNone {
			d356 = ps.OverlayValues[356]
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
		if len(ps.OverlayValues) > 362 && ps.OverlayValues[362].Loc != scm.LocNone {
			d362 = ps.OverlayValues[362]
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
		if len(ps.OverlayValues) > 524 && ps.OverlayValues[524].Loc != scm.LocNone {
			d524 = ps.OverlayValues[524]
		}
		if len(ps.OverlayValues) > 525 && ps.OverlayValues[525].Loc != scm.LocNone {
			d525 = ps.OverlayValues[525]
		}
		if len(ps.OverlayValues) > 526 && ps.OverlayValues[526].Loc != scm.LocNone {
			d526 = ps.OverlayValues[526]
		}
		if len(ps.OverlayValues) > 528 && ps.OverlayValues[528].Loc != scm.LocNone {
			d528 = ps.OverlayValues[528]
		}
		if len(ps.OverlayValues) > 529 && ps.OverlayValues[529].Loc != scm.LocNone {
			d529 = ps.OverlayValues[529]
		}
		if len(ps.OverlayValues) > 530 && ps.OverlayValues[530].Loc != scm.LocNone {
			d530 = ps.OverlayValues[530]
		}
		if len(ps.OverlayValues) > 531 && ps.OverlayValues[531].Loc != scm.LocNone {
			d531 = ps.OverlayValues[531]
		}
		if len(ps.OverlayValues) > 532 && ps.OverlayValues[532].Loc != scm.LocNone {
			d532 = ps.OverlayValues[532]
		}
		if len(ps.OverlayValues) > 533 && ps.OverlayValues[533].Loc != scm.LocNone {
			d533 = ps.OverlayValues[533]
		}
		if len(ps.OverlayValues) > 534 && ps.OverlayValues[534].Loc != scm.LocNone {
			d534 = ps.OverlayValues[534]
		}
		if len(ps.OverlayValues) > 536 && ps.OverlayValues[536].Loc != scm.LocNone {
			d536 = ps.OverlayValues[536]
		}
		if len(ps.OverlayValues) > 538 && ps.OverlayValues[538].Loc != scm.LocNone {
			d538 = ps.OverlayValues[538]
		}
		if len(ps.OverlayValues) > 539 && ps.OverlayValues[539].Loc != scm.LocNone {
			d539 = ps.OverlayValues[539]
		}
		if len(ps.OverlayValues) > 540 && ps.OverlayValues[540].Loc != scm.LocNone {
			d540 = ps.OverlayValues[540]
		}
		if len(ps.OverlayValues) > 541 && ps.OverlayValues[541].Loc != scm.LocNone {
			d541 = ps.OverlayValues[541]
		}
		if len(ps.OverlayValues) > 544 && ps.OverlayValues[544].Loc != scm.LocNone {
			d544 = ps.OverlayValues[544]
		}
		if len(ps.OverlayValues) > 696 && ps.OverlayValues[696].Loc != scm.LocNone {
			d696 = ps.OverlayValues[696]
		}
		if len(ps.OverlayValues) > 697 && ps.OverlayValues[697].Loc != scm.LocNone {
			d697 = ps.OverlayValues[697]
		}
		if len(ps.OverlayValues) > 698 && ps.OverlayValues[698].Loc != scm.LocNone {
			d698 = ps.OverlayValues[698]
		}
		if len(ps.OverlayValues) > 699 && ps.OverlayValues[699].Loc != scm.LocNone {
			d699 = ps.OverlayValues[699]
		}
		if len(ps.OverlayValues) > 701 && ps.OverlayValues[701].Loc != scm.LocNone {
			d701 = ps.OverlayValues[701]
		}
		if len(ps.OverlayValues) > 702 && ps.OverlayValues[702].Loc != scm.LocNone {
			d702 = ps.OverlayValues[702]
		}
		if len(ps.OverlayValues) > 703 && ps.OverlayValues[703].Loc != scm.LocNone {
			d703 = ps.OverlayValues[703]
		}
		if len(ps.OverlayValues) > 704 && ps.OverlayValues[704].Loc != scm.LocNone {
			d704 = ps.OverlayValues[704]
		}
		if len(ps.OverlayValues) > 705 && ps.OverlayValues[705].Loc != scm.LocNone {
			d705 = ps.OverlayValues[705]
		}
		if len(ps.OverlayValues) > 706 && ps.OverlayValues[706].Loc != scm.LocNone {
			d706 = ps.OverlayValues[706]
		}
		if len(ps.OverlayValues) > 707 && ps.OverlayValues[707].Loc != scm.LocNone {
			d707 = ps.OverlayValues[707]
		}
		if len(ps.OverlayValues) > 708 && ps.OverlayValues[708].Loc != scm.LocNone {
			d708 = ps.OverlayValues[708]
		}
		if len(ps.OverlayValues) > 710 && ps.OverlayValues[710].Loc != scm.LocNone {
			d710 = ps.OverlayValues[710]
		}
		if len(ps.OverlayValues) > 711 && ps.OverlayValues[711].Loc != scm.LocNone {
			d711 = ps.OverlayValues[711]
		}
		if len(ps.OverlayValues) > 712 && ps.OverlayValues[712].Loc != scm.LocNone {
			d712 = ps.OverlayValues[712]
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
		if len(ps.OverlayValues) > 717 && ps.OverlayValues[717].Loc != scm.LocNone {
			d717 = ps.OverlayValues[717]
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
		ctx.ReclaimUntrackedRegs()
		var d769 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 64
			r188 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r188, fieldAddr)
			d769 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r188}
			ctx.BindReg(r188, &d769)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 64)
			r189 := ctx.AllocReg()
			ctx.EmitMovRegMem(r189, thisptr.Reg, off)
			d769 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r189}
			ctx.BindReg(r189, &d769)
		}
		ctx.EnsureDesc(&d769)
		ctx.EnsureDesc(&d769)
		var d770 scm.JITValueDesc
		if d769.Loc == scm.LocImm {
			d770 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d769.Imm.Int()))))}
		} else {
			r190 := ctx.AllocReg()
			ctx.EmitMovRegReg(r190, d769.Reg)
			d770 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r190}
			ctx.BindReg(r190, &d770)
		}
		ctx.EnsureDesc(&d135)
		ctx.EnsureDesc(&d770)
		ctx.EnsureDescsTogether(&d135, &d770)
		var d771 scm.JITValueDesc
		if d135.Loc == scm.LocImm && d770.Loc == scm.LocImm {
			d771 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d135.Imm.Int() == d770.Imm.Int())}
		} else if d770.Loc == scm.LocImm {
			r191 := ctx.AllocRegExcept(d135.Reg)
			if d770.Imm.Int() >= -2147483648 && d770.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d135.Reg, int32(d770.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d770.Imm.Int()))
				ctx.EmitCmpInt64(d135.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r191, scm.CondEqual)
			d771 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r191}
			ctx.BindReg(r191, &d771)
		} else if d135.Loc == scm.LocImm {
			r192 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d135.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d770.Reg)
			ctx.EmitSetcc(r192, scm.CondEqual)
			d771 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r192}
			ctx.BindReg(r192, &d771)
		} else {
			r193 := ctx.AllocRegExcept(d135.Reg)
			ctx.EmitCmpInt64(d135.Reg, d770.Reg)
			ctx.EmitSetcc(r193, scm.CondEqual)
			d771 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r193}
			ctx.BindReg(r193, &d771)
		}
		ctx.FreeDesc(&d135)
		ctx.FreeDesc(&d770)
		d772 = d771
		ctx.EnsureDesc(&d772)
		if d772.Loc != scm.LocImm && d772.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d772.Loc == scm.LocImm {
			if d772.Imm.Bool() {
				if ps.General {
				}
				ps773 := scm.PhiState{General: ps.General}
				ps773.OverlayValues = make([]scm.JITValueDesc, 773)
				ps773.OverlayValues[1] = d1
				ps773.OverlayValues[2] = d2
				ps773.OverlayValues[3] = d3
				ps773.OverlayValues[4] = d4
				ps773.OverlayValues[5] = d5
				ps773.OverlayValues[6] = d6
				ps773.OverlayValues[7] = d7
				ps773.OverlayValues[8] = d8
				ps773.OverlayValues[9] = d9
				ps773.OverlayValues[10] = d10
				ps773.OverlayValues[11] = d11
				ps773.OverlayValues[12] = d12
				ps773.OverlayValues[13] = d13
				ps773.OverlayValues[14] = d14
				ps773.OverlayValues[15] = d15
				ps773.OverlayValues[17] = d17
				ps773.OverlayValues[18] = d18
				ps773.OverlayValues[19] = d19
				ps773.OverlayValues[20] = d20
				ps773.OverlayValues[21] = d21
				ps773.OverlayValues[22] = d22
				ps773.OverlayValues[23] = d23
				ps773.OverlayValues[24] = d24
				ps773.OverlayValues[25] = d25
				ps773.OverlayValues[26] = d26
				ps773.OverlayValues[27] = d27
				ps773.OverlayValues[28] = d28
				ps773.OverlayValues[29] = d29
				ps773.OverlayValues[30] = d30
				ps773.OverlayValues[31] = d31
				ps773.OverlayValues[32] = d32
				ps773.OverlayValues[33] = d33
				ps773.OverlayValues[34] = d34
				ps773.OverlayValues[35] = d35
				ps773.OverlayValues[36] = d36
				ps773.OverlayValues[37] = d37
				ps773.OverlayValues[38] = d38
				ps773.OverlayValues[39] = d39
				ps773.OverlayValues[40] = d40
				ps773.OverlayValues[41] = d41
				ps773.OverlayValues[42] = d42
				ps773.OverlayValues[43] = d43
				ps773.OverlayValues[44] = d44
				ps773.OverlayValues[45] = d45
				ps773.OverlayValues[46] = d46
				ps773.OverlayValues[47] = d47
				ps773.OverlayValues[48] = d48
				ps773.OverlayValues[49] = d49
				ps773.OverlayValues[52] = d52
				ps773.OverlayValues[53] = d53
				ps773.OverlayValues[54] = d54
				ps773.OverlayValues[109] = d109
				ps773.OverlayValues[110] = d110
				ps773.OverlayValues[111] = d111
				ps773.OverlayValues[112] = d112
				ps773.OverlayValues[113] = d113
				ps773.OverlayValues[114] = d114
				ps773.OverlayValues[115] = d115
				ps773.OverlayValues[116] = d116
				ps773.OverlayValues[117] = d117
				ps773.OverlayValues[118] = d118
				ps773.OverlayValues[119] = d119
				ps773.OverlayValues[120] = d120
				ps773.OverlayValues[121] = d121
				ps773.OverlayValues[122] = d122
				ps773.OverlayValues[123] = d123
				ps773.OverlayValues[124] = d124
				ps773.OverlayValues[125] = d125
				ps773.OverlayValues[126] = d126
				ps773.OverlayValues[127] = d127
				ps773.OverlayValues[128] = d128
				ps773.OverlayValues[129] = d129
				ps773.OverlayValues[130] = d130
				ps773.OverlayValues[131] = d131
				ps773.OverlayValues[132] = d132
				ps773.OverlayValues[133] = d133
				ps773.OverlayValues[134] = d134
				ps773.OverlayValues[135] = d135
				ps773.OverlayValues[136] = d136
				ps773.OverlayValues[137] = d137
				ps773.OverlayValues[140] = d140
				ps773.OverlayValues[225] = d225
				ps773.OverlayValues[226] = d226
				ps773.OverlayValues[227] = d227
				ps773.OverlayValues[228] = d228
				ps773.OverlayValues[230] = d230
				ps773.OverlayValues[231] = d231
				ps773.OverlayValues[232] = d232
				ps773.OverlayValues[233] = d233
				ps773.OverlayValues[234] = d234
				ps773.OverlayValues[235] = d235
				ps773.OverlayValues[236] = d236
				ps773.OverlayValues[237] = d237
				ps773.OverlayValues[239] = d239
				ps773.OverlayValues[241] = d241
				ps773.OverlayValues[242] = d242
				ps773.OverlayValues[243] = d243
				ps773.OverlayValues[244] = d244
				ps773.OverlayValues[245] = d245
				ps773.OverlayValues[248] = d248
				ps773.OverlayValues[350] = d350
				ps773.OverlayValues[351] = d351
				ps773.OverlayValues[352] = d352
				ps773.OverlayValues[353] = d353
				ps773.OverlayValues[354] = d354
				ps773.OverlayValues[356] = d356
				ps773.OverlayValues[357] = d357
				ps773.OverlayValues[358] = d358
				ps773.OverlayValues[359] = d359
				ps773.OverlayValues[360] = d360
				ps773.OverlayValues[361] = d361
				ps773.OverlayValues[362] = d362
				ps773.OverlayValues[363] = d363
				ps773.OverlayValues[364] = d364
				ps773.OverlayValues[365] = d365
				ps773.OverlayValues[366] = d366
				ps773.OverlayValues[367] = d367
				ps773.OverlayValues[368] = d368
				ps773.OverlayValues[369] = d369
				ps773.OverlayValues[370] = d370
				ps773.OverlayValues[371] = d371
				ps773.OverlayValues[372] = d372
				ps773.OverlayValues[373] = d373
				ps773.OverlayValues[374] = d374
				ps773.OverlayValues[375] = d375
				ps773.OverlayValues[376] = d376
				ps773.OverlayValues[377] = d377
				ps773.OverlayValues[378] = d378
				ps773.OverlayValues[379] = d379
				ps773.OverlayValues[380] = d380
				ps773.OverlayValues[381] = d381
				ps773.OverlayValues[382] = d382
				ps773.OverlayValues[383] = d383
				ps773.OverlayValues[384] = d384
				ps773.OverlayValues[524] = d524
				ps773.OverlayValues[525] = d525
				ps773.OverlayValues[526] = d526
				ps773.OverlayValues[528] = d528
				ps773.OverlayValues[529] = d529
				ps773.OverlayValues[530] = d530
				ps773.OverlayValues[531] = d531
				ps773.OverlayValues[532] = d532
				ps773.OverlayValues[533] = d533
				ps773.OverlayValues[534] = d534
				ps773.OverlayValues[536] = d536
				ps773.OverlayValues[538] = d538
				ps773.OverlayValues[539] = d539
				ps773.OverlayValues[540] = d540
				ps773.OverlayValues[541] = d541
				ps773.OverlayValues[544] = d544
				ps773.OverlayValues[696] = d696
				ps773.OverlayValues[697] = d697
				ps773.OverlayValues[698] = d698
				ps773.OverlayValues[699] = d699
				ps773.OverlayValues[701] = d701
				ps773.OverlayValues[702] = d702
				ps773.OverlayValues[703] = d703
				ps773.OverlayValues[704] = d704
				ps773.OverlayValues[705] = d705
				ps773.OverlayValues[706] = d706
				ps773.OverlayValues[707] = d707
				ps773.OverlayValues[708] = d708
				ps773.OverlayValues[710] = d710
				ps773.OverlayValues[711] = d711
				ps773.OverlayValues[712] = d712
				ps773.OverlayValues[713] = d713
				ps773.OverlayValues[714] = d714
				ps773.OverlayValues[715] = d715
				ps773.OverlayValues[716] = d716
				ps773.OverlayValues[717] = d717
				ps773.OverlayValues[718] = d718
				ps773.OverlayValues[719] = d719
				ps773.OverlayValues[720] = d720
				ps773.OverlayValues[721] = d721
				ps773.OverlayValues[722] = d722
				ps773.OverlayValues[723] = d723
				ps773.OverlayValues[724] = d724
				ps773.OverlayValues[725] = d725
				ps773.OverlayValues[726] = d726
				ps773.OverlayValues[727] = d727
				ps773.OverlayValues[728] = d728
				ps773.OverlayValues[729] = d729
				ps773.OverlayValues[730] = d730
				ps773.OverlayValues[731] = d731
				ps773.OverlayValues[732] = d732
				ps773.OverlayValues[733] = d733
				ps773.OverlayValues[734] = d734
				ps773.OverlayValues[735] = d735
				ps773.OverlayValues[736] = d736
				ps773.OverlayValues[737] = d737
				ps773.OverlayValues[738] = d738
				ps773.OverlayValues[739] = d739
				ps773.OverlayValues[740] = d740
				ps773.OverlayValues[741] = d741
				ps773.OverlayValues[742] = d742
				ps773.OverlayValues[743] = d743
				ps773.OverlayValues[744] = d744
				ps773.OverlayValues[745] = d745
				ps773.OverlayValues[746] = d746
				ps773.OverlayValues[747] = d747
				ps773.OverlayValues[748] = d748
				ps773.OverlayValues[749] = d749
				ps773.OverlayValues[750] = d750
				ps773.OverlayValues[751] = d751
				ps773.OverlayValues[752] = d752
				ps773.OverlayValues[753] = d753
				ps773.OverlayValues[754] = d754
				ps773.OverlayValues[755] = d755
				ps773.OverlayValues[756] = d756
				ps773.OverlayValues[757] = d757
				ps773.OverlayValues[758] = d758
				ps773.OverlayValues[759] = d759
				ps773.OverlayValues[760] = d760
				ps773.OverlayValues[761] = d761
				ps773.OverlayValues[762] = d762
				ps773.OverlayValues[763] = d763
				ps773.OverlayValues[764] = d764
				ps773.OverlayValues[765] = d765
				ps773.OverlayValues[766] = d766
				ps773.OverlayValues[767] = d767
				ps773.OverlayValues[768] = d768
				ps773.OverlayValues[769] = d769
				ps773.OverlayValues[770] = d770
				ps773.OverlayValues[771] = d771
				ps773.OverlayValues[772] = d772
				return bbs[11].RenderPS(ps773)
			}
			if ps.General {
			}
			ps774 := scm.PhiState{General: ps.General}
			ps774.OverlayValues = make([]scm.JITValueDesc, 773)
			ps774.OverlayValues[1] = d1
			ps774.OverlayValues[2] = d2
			ps774.OverlayValues[3] = d3
			ps774.OverlayValues[4] = d4
			ps774.OverlayValues[5] = d5
			ps774.OverlayValues[6] = d6
			ps774.OverlayValues[7] = d7
			ps774.OverlayValues[8] = d8
			ps774.OverlayValues[9] = d9
			ps774.OverlayValues[10] = d10
			ps774.OverlayValues[11] = d11
			ps774.OverlayValues[12] = d12
			ps774.OverlayValues[13] = d13
			ps774.OverlayValues[14] = d14
			ps774.OverlayValues[15] = d15
			ps774.OverlayValues[17] = d17
			ps774.OverlayValues[18] = d18
			ps774.OverlayValues[19] = d19
			ps774.OverlayValues[20] = d20
			ps774.OverlayValues[21] = d21
			ps774.OverlayValues[22] = d22
			ps774.OverlayValues[23] = d23
			ps774.OverlayValues[24] = d24
			ps774.OverlayValues[25] = d25
			ps774.OverlayValues[26] = d26
			ps774.OverlayValues[27] = d27
			ps774.OverlayValues[28] = d28
			ps774.OverlayValues[29] = d29
			ps774.OverlayValues[30] = d30
			ps774.OverlayValues[31] = d31
			ps774.OverlayValues[32] = d32
			ps774.OverlayValues[33] = d33
			ps774.OverlayValues[34] = d34
			ps774.OverlayValues[35] = d35
			ps774.OverlayValues[36] = d36
			ps774.OverlayValues[37] = d37
			ps774.OverlayValues[38] = d38
			ps774.OverlayValues[39] = d39
			ps774.OverlayValues[40] = d40
			ps774.OverlayValues[41] = d41
			ps774.OverlayValues[42] = d42
			ps774.OverlayValues[43] = d43
			ps774.OverlayValues[44] = d44
			ps774.OverlayValues[45] = d45
			ps774.OverlayValues[46] = d46
			ps774.OverlayValues[47] = d47
			ps774.OverlayValues[48] = d48
			ps774.OverlayValues[49] = d49
			ps774.OverlayValues[52] = d52
			ps774.OverlayValues[53] = d53
			ps774.OverlayValues[54] = d54
			ps774.OverlayValues[109] = d109
			ps774.OverlayValues[110] = d110
			ps774.OverlayValues[111] = d111
			ps774.OverlayValues[112] = d112
			ps774.OverlayValues[113] = d113
			ps774.OverlayValues[114] = d114
			ps774.OverlayValues[115] = d115
			ps774.OverlayValues[116] = d116
			ps774.OverlayValues[117] = d117
			ps774.OverlayValues[118] = d118
			ps774.OverlayValues[119] = d119
			ps774.OverlayValues[120] = d120
			ps774.OverlayValues[121] = d121
			ps774.OverlayValues[122] = d122
			ps774.OverlayValues[123] = d123
			ps774.OverlayValues[124] = d124
			ps774.OverlayValues[125] = d125
			ps774.OverlayValues[126] = d126
			ps774.OverlayValues[127] = d127
			ps774.OverlayValues[128] = d128
			ps774.OverlayValues[129] = d129
			ps774.OverlayValues[130] = d130
			ps774.OverlayValues[131] = d131
			ps774.OverlayValues[132] = d132
			ps774.OverlayValues[133] = d133
			ps774.OverlayValues[134] = d134
			ps774.OverlayValues[135] = d135
			ps774.OverlayValues[136] = d136
			ps774.OverlayValues[137] = d137
			ps774.OverlayValues[140] = d140
			ps774.OverlayValues[225] = d225
			ps774.OverlayValues[226] = d226
			ps774.OverlayValues[227] = d227
			ps774.OverlayValues[228] = d228
			ps774.OverlayValues[230] = d230
			ps774.OverlayValues[231] = d231
			ps774.OverlayValues[232] = d232
			ps774.OverlayValues[233] = d233
			ps774.OverlayValues[234] = d234
			ps774.OverlayValues[235] = d235
			ps774.OverlayValues[236] = d236
			ps774.OverlayValues[237] = d237
			ps774.OverlayValues[239] = d239
			ps774.OverlayValues[241] = d241
			ps774.OverlayValues[242] = d242
			ps774.OverlayValues[243] = d243
			ps774.OverlayValues[244] = d244
			ps774.OverlayValues[245] = d245
			ps774.OverlayValues[248] = d248
			ps774.OverlayValues[350] = d350
			ps774.OverlayValues[351] = d351
			ps774.OverlayValues[352] = d352
			ps774.OverlayValues[353] = d353
			ps774.OverlayValues[354] = d354
			ps774.OverlayValues[356] = d356
			ps774.OverlayValues[357] = d357
			ps774.OverlayValues[358] = d358
			ps774.OverlayValues[359] = d359
			ps774.OverlayValues[360] = d360
			ps774.OverlayValues[361] = d361
			ps774.OverlayValues[362] = d362
			ps774.OverlayValues[363] = d363
			ps774.OverlayValues[364] = d364
			ps774.OverlayValues[365] = d365
			ps774.OverlayValues[366] = d366
			ps774.OverlayValues[367] = d367
			ps774.OverlayValues[368] = d368
			ps774.OverlayValues[369] = d369
			ps774.OverlayValues[370] = d370
			ps774.OverlayValues[371] = d371
			ps774.OverlayValues[372] = d372
			ps774.OverlayValues[373] = d373
			ps774.OverlayValues[374] = d374
			ps774.OverlayValues[375] = d375
			ps774.OverlayValues[376] = d376
			ps774.OverlayValues[377] = d377
			ps774.OverlayValues[378] = d378
			ps774.OverlayValues[379] = d379
			ps774.OverlayValues[380] = d380
			ps774.OverlayValues[381] = d381
			ps774.OverlayValues[382] = d382
			ps774.OverlayValues[383] = d383
			ps774.OverlayValues[384] = d384
			ps774.OverlayValues[524] = d524
			ps774.OverlayValues[525] = d525
			ps774.OverlayValues[526] = d526
			ps774.OverlayValues[528] = d528
			ps774.OverlayValues[529] = d529
			ps774.OverlayValues[530] = d530
			ps774.OverlayValues[531] = d531
			ps774.OverlayValues[532] = d532
			ps774.OverlayValues[533] = d533
			ps774.OverlayValues[534] = d534
			ps774.OverlayValues[536] = d536
			ps774.OverlayValues[538] = d538
			ps774.OverlayValues[539] = d539
			ps774.OverlayValues[540] = d540
			ps774.OverlayValues[541] = d541
			ps774.OverlayValues[544] = d544
			ps774.OverlayValues[696] = d696
			ps774.OverlayValues[697] = d697
			ps774.OverlayValues[698] = d698
			ps774.OverlayValues[699] = d699
			ps774.OverlayValues[701] = d701
			ps774.OverlayValues[702] = d702
			ps774.OverlayValues[703] = d703
			ps774.OverlayValues[704] = d704
			ps774.OverlayValues[705] = d705
			ps774.OverlayValues[706] = d706
			ps774.OverlayValues[707] = d707
			ps774.OverlayValues[708] = d708
			ps774.OverlayValues[710] = d710
			ps774.OverlayValues[711] = d711
			ps774.OverlayValues[712] = d712
			ps774.OverlayValues[713] = d713
			ps774.OverlayValues[714] = d714
			ps774.OverlayValues[715] = d715
			ps774.OverlayValues[716] = d716
			ps774.OverlayValues[717] = d717
			ps774.OverlayValues[718] = d718
			ps774.OverlayValues[719] = d719
			ps774.OverlayValues[720] = d720
			ps774.OverlayValues[721] = d721
			ps774.OverlayValues[722] = d722
			ps774.OverlayValues[723] = d723
			ps774.OverlayValues[724] = d724
			ps774.OverlayValues[725] = d725
			ps774.OverlayValues[726] = d726
			ps774.OverlayValues[727] = d727
			ps774.OverlayValues[728] = d728
			ps774.OverlayValues[729] = d729
			ps774.OverlayValues[730] = d730
			ps774.OverlayValues[731] = d731
			ps774.OverlayValues[732] = d732
			ps774.OverlayValues[733] = d733
			ps774.OverlayValues[734] = d734
			ps774.OverlayValues[735] = d735
			ps774.OverlayValues[736] = d736
			ps774.OverlayValues[737] = d737
			ps774.OverlayValues[738] = d738
			ps774.OverlayValues[739] = d739
			ps774.OverlayValues[740] = d740
			ps774.OverlayValues[741] = d741
			ps774.OverlayValues[742] = d742
			ps774.OverlayValues[743] = d743
			ps774.OverlayValues[744] = d744
			ps774.OverlayValues[745] = d745
			ps774.OverlayValues[746] = d746
			ps774.OverlayValues[747] = d747
			ps774.OverlayValues[748] = d748
			ps774.OverlayValues[749] = d749
			ps774.OverlayValues[750] = d750
			ps774.OverlayValues[751] = d751
			ps774.OverlayValues[752] = d752
			ps774.OverlayValues[753] = d753
			ps774.OverlayValues[754] = d754
			ps774.OverlayValues[755] = d755
			ps774.OverlayValues[756] = d756
			ps774.OverlayValues[757] = d757
			ps774.OverlayValues[758] = d758
			ps774.OverlayValues[759] = d759
			ps774.OverlayValues[760] = d760
			ps774.OverlayValues[761] = d761
			ps774.OverlayValues[762] = d762
			ps774.OverlayValues[763] = d763
			ps774.OverlayValues[764] = d764
			ps774.OverlayValues[765] = d765
			ps774.OverlayValues[766] = d766
			ps774.OverlayValues[767] = d767
			ps774.OverlayValues[768] = d768
			ps774.OverlayValues[769] = d769
			ps774.OverlayValues[770] = d770
			ps774.OverlayValues[771] = d771
			ps774.OverlayValues[772] = d772
			return bbs[12].RenderPS(ps774)
		}
		if !ps.General {
			ps.General = true
			return bbs[13].RenderPS(ps)
		}
		lbl30 := ctx.ReserveLabel()
		lbl31 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d772.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl30)
		ctx.EmitJmp(lbl31)
		ctx.MarkLabel(lbl30)
		ctx.EmitJmp(lbl12)
		ctx.MarkLabel(lbl31)
		ctx.EmitJmp(lbl13)
		ps775 := scm.PhiState{General: true}
		ps775.OverlayValues = make([]scm.JITValueDesc, 773)
		ps775.OverlayValues[1] = d1
		ps775.OverlayValues[2] = d2
		ps775.OverlayValues[3] = d3
		ps775.OverlayValues[4] = d4
		ps775.OverlayValues[5] = d5
		ps775.OverlayValues[6] = d6
		ps775.OverlayValues[7] = d7
		ps775.OverlayValues[8] = d8
		ps775.OverlayValues[9] = d9
		ps775.OverlayValues[10] = d10
		ps775.OverlayValues[11] = d11
		ps775.OverlayValues[12] = d12
		ps775.OverlayValues[13] = d13
		ps775.OverlayValues[14] = d14
		ps775.OverlayValues[15] = d15
		ps775.OverlayValues[17] = d17
		ps775.OverlayValues[18] = d18
		ps775.OverlayValues[19] = d19
		ps775.OverlayValues[20] = d20
		ps775.OverlayValues[21] = d21
		ps775.OverlayValues[22] = d22
		ps775.OverlayValues[23] = d23
		ps775.OverlayValues[24] = d24
		ps775.OverlayValues[25] = d25
		ps775.OverlayValues[26] = d26
		ps775.OverlayValues[27] = d27
		ps775.OverlayValues[28] = d28
		ps775.OverlayValues[29] = d29
		ps775.OverlayValues[30] = d30
		ps775.OverlayValues[31] = d31
		ps775.OverlayValues[32] = d32
		ps775.OverlayValues[33] = d33
		ps775.OverlayValues[34] = d34
		ps775.OverlayValues[35] = d35
		ps775.OverlayValues[36] = d36
		ps775.OverlayValues[37] = d37
		ps775.OverlayValues[38] = d38
		ps775.OverlayValues[39] = d39
		ps775.OverlayValues[40] = d40
		ps775.OverlayValues[41] = d41
		ps775.OverlayValues[42] = d42
		ps775.OverlayValues[43] = d43
		ps775.OverlayValues[44] = d44
		ps775.OverlayValues[45] = d45
		ps775.OverlayValues[46] = d46
		ps775.OverlayValues[47] = d47
		ps775.OverlayValues[48] = d48
		ps775.OverlayValues[49] = d49
		ps775.OverlayValues[52] = d52
		ps775.OverlayValues[53] = d53
		ps775.OverlayValues[54] = d54
		ps775.OverlayValues[109] = d109
		ps775.OverlayValues[110] = d110
		ps775.OverlayValues[111] = d111
		ps775.OverlayValues[112] = d112
		ps775.OverlayValues[113] = d113
		ps775.OverlayValues[114] = d114
		ps775.OverlayValues[115] = d115
		ps775.OverlayValues[116] = d116
		ps775.OverlayValues[117] = d117
		ps775.OverlayValues[118] = d118
		ps775.OverlayValues[119] = d119
		ps775.OverlayValues[120] = d120
		ps775.OverlayValues[121] = d121
		ps775.OverlayValues[122] = d122
		ps775.OverlayValues[123] = d123
		ps775.OverlayValues[124] = d124
		ps775.OverlayValues[125] = d125
		ps775.OverlayValues[126] = d126
		ps775.OverlayValues[127] = d127
		ps775.OverlayValues[128] = d128
		ps775.OverlayValues[129] = d129
		ps775.OverlayValues[130] = d130
		ps775.OverlayValues[131] = d131
		ps775.OverlayValues[132] = d132
		ps775.OverlayValues[133] = d133
		ps775.OverlayValues[134] = d134
		ps775.OverlayValues[135] = d135
		ps775.OverlayValues[136] = d136
		ps775.OverlayValues[137] = d137
		ps775.OverlayValues[140] = d140
		ps775.OverlayValues[225] = d225
		ps775.OverlayValues[226] = d226
		ps775.OverlayValues[227] = d227
		ps775.OverlayValues[228] = d228
		ps775.OverlayValues[230] = d230
		ps775.OverlayValues[231] = d231
		ps775.OverlayValues[232] = d232
		ps775.OverlayValues[233] = d233
		ps775.OverlayValues[234] = d234
		ps775.OverlayValues[235] = d235
		ps775.OverlayValues[236] = d236
		ps775.OverlayValues[237] = d237
		ps775.OverlayValues[239] = d239
		ps775.OverlayValues[241] = d241
		ps775.OverlayValues[242] = d242
		ps775.OverlayValues[243] = d243
		ps775.OverlayValues[244] = d244
		ps775.OverlayValues[245] = d245
		ps775.OverlayValues[248] = d248
		ps775.OverlayValues[350] = d350
		ps775.OverlayValues[351] = d351
		ps775.OverlayValues[352] = d352
		ps775.OverlayValues[353] = d353
		ps775.OverlayValues[354] = d354
		ps775.OverlayValues[356] = d356
		ps775.OverlayValues[357] = d357
		ps775.OverlayValues[358] = d358
		ps775.OverlayValues[359] = d359
		ps775.OverlayValues[360] = d360
		ps775.OverlayValues[361] = d361
		ps775.OverlayValues[362] = d362
		ps775.OverlayValues[363] = d363
		ps775.OverlayValues[364] = d364
		ps775.OverlayValues[365] = d365
		ps775.OverlayValues[366] = d366
		ps775.OverlayValues[367] = d367
		ps775.OverlayValues[368] = d368
		ps775.OverlayValues[369] = d369
		ps775.OverlayValues[370] = d370
		ps775.OverlayValues[371] = d371
		ps775.OverlayValues[372] = d372
		ps775.OverlayValues[373] = d373
		ps775.OverlayValues[374] = d374
		ps775.OverlayValues[375] = d375
		ps775.OverlayValues[376] = d376
		ps775.OverlayValues[377] = d377
		ps775.OverlayValues[378] = d378
		ps775.OverlayValues[379] = d379
		ps775.OverlayValues[380] = d380
		ps775.OverlayValues[381] = d381
		ps775.OverlayValues[382] = d382
		ps775.OverlayValues[383] = d383
		ps775.OverlayValues[384] = d384
		ps775.OverlayValues[524] = d524
		ps775.OverlayValues[525] = d525
		ps775.OverlayValues[526] = d526
		ps775.OverlayValues[528] = d528
		ps775.OverlayValues[529] = d529
		ps775.OverlayValues[530] = d530
		ps775.OverlayValues[531] = d531
		ps775.OverlayValues[532] = d532
		ps775.OverlayValues[533] = d533
		ps775.OverlayValues[534] = d534
		ps775.OverlayValues[536] = d536
		ps775.OverlayValues[538] = d538
		ps775.OverlayValues[539] = d539
		ps775.OverlayValues[540] = d540
		ps775.OverlayValues[541] = d541
		ps775.OverlayValues[544] = d544
		ps775.OverlayValues[696] = d696
		ps775.OverlayValues[697] = d697
		ps775.OverlayValues[698] = d698
		ps775.OverlayValues[699] = d699
		ps775.OverlayValues[701] = d701
		ps775.OverlayValues[702] = d702
		ps775.OverlayValues[703] = d703
		ps775.OverlayValues[704] = d704
		ps775.OverlayValues[705] = d705
		ps775.OverlayValues[706] = d706
		ps775.OverlayValues[707] = d707
		ps775.OverlayValues[708] = d708
		ps775.OverlayValues[710] = d710
		ps775.OverlayValues[711] = d711
		ps775.OverlayValues[712] = d712
		ps775.OverlayValues[713] = d713
		ps775.OverlayValues[714] = d714
		ps775.OverlayValues[715] = d715
		ps775.OverlayValues[716] = d716
		ps775.OverlayValues[717] = d717
		ps775.OverlayValues[718] = d718
		ps775.OverlayValues[719] = d719
		ps775.OverlayValues[720] = d720
		ps775.OverlayValues[721] = d721
		ps775.OverlayValues[722] = d722
		ps775.OverlayValues[723] = d723
		ps775.OverlayValues[724] = d724
		ps775.OverlayValues[725] = d725
		ps775.OverlayValues[726] = d726
		ps775.OverlayValues[727] = d727
		ps775.OverlayValues[728] = d728
		ps775.OverlayValues[729] = d729
		ps775.OverlayValues[730] = d730
		ps775.OverlayValues[731] = d731
		ps775.OverlayValues[732] = d732
		ps775.OverlayValues[733] = d733
		ps775.OverlayValues[734] = d734
		ps775.OverlayValues[735] = d735
		ps775.OverlayValues[736] = d736
		ps775.OverlayValues[737] = d737
		ps775.OverlayValues[738] = d738
		ps775.OverlayValues[739] = d739
		ps775.OverlayValues[740] = d740
		ps775.OverlayValues[741] = d741
		ps775.OverlayValues[742] = d742
		ps775.OverlayValues[743] = d743
		ps775.OverlayValues[744] = d744
		ps775.OverlayValues[745] = d745
		ps775.OverlayValues[746] = d746
		ps775.OverlayValues[747] = d747
		ps775.OverlayValues[748] = d748
		ps775.OverlayValues[749] = d749
		ps775.OverlayValues[750] = d750
		ps775.OverlayValues[751] = d751
		ps775.OverlayValues[752] = d752
		ps775.OverlayValues[753] = d753
		ps775.OverlayValues[754] = d754
		ps775.OverlayValues[755] = d755
		ps775.OverlayValues[756] = d756
		ps775.OverlayValues[757] = d757
		ps775.OverlayValues[758] = d758
		ps775.OverlayValues[759] = d759
		ps775.OverlayValues[760] = d760
		ps775.OverlayValues[761] = d761
		ps775.OverlayValues[762] = d762
		ps775.OverlayValues[763] = d763
		ps775.OverlayValues[764] = d764
		ps775.OverlayValues[765] = d765
		ps775.OverlayValues[766] = d766
		ps775.OverlayValues[767] = d767
		ps775.OverlayValues[768] = d768
		ps775.OverlayValues[769] = d769
		ps775.OverlayValues[770] = d770
		ps775.OverlayValues[771] = d771
		ps775.OverlayValues[772] = d772
		ps776 := scm.PhiState{General: true}
		ps776.OverlayValues = make([]scm.JITValueDesc, 773)
		ps776.OverlayValues[1] = d1
		ps776.OverlayValues[2] = d2
		ps776.OverlayValues[3] = d3
		ps776.OverlayValues[4] = d4
		ps776.OverlayValues[5] = d5
		ps776.OverlayValues[6] = d6
		ps776.OverlayValues[7] = d7
		ps776.OverlayValues[8] = d8
		ps776.OverlayValues[9] = d9
		ps776.OverlayValues[10] = d10
		ps776.OverlayValues[11] = d11
		ps776.OverlayValues[12] = d12
		ps776.OverlayValues[13] = d13
		ps776.OverlayValues[14] = d14
		ps776.OverlayValues[15] = d15
		ps776.OverlayValues[17] = d17
		ps776.OverlayValues[18] = d18
		ps776.OverlayValues[19] = d19
		ps776.OverlayValues[20] = d20
		ps776.OverlayValues[21] = d21
		ps776.OverlayValues[22] = d22
		ps776.OverlayValues[23] = d23
		ps776.OverlayValues[24] = d24
		ps776.OverlayValues[25] = d25
		ps776.OverlayValues[26] = d26
		ps776.OverlayValues[27] = d27
		ps776.OverlayValues[28] = d28
		ps776.OverlayValues[29] = d29
		ps776.OverlayValues[30] = d30
		ps776.OverlayValues[31] = d31
		ps776.OverlayValues[32] = d32
		ps776.OverlayValues[33] = d33
		ps776.OverlayValues[34] = d34
		ps776.OverlayValues[35] = d35
		ps776.OverlayValues[36] = d36
		ps776.OverlayValues[37] = d37
		ps776.OverlayValues[38] = d38
		ps776.OverlayValues[39] = d39
		ps776.OverlayValues[40] = d40
		ps776.OverlayValues[41] = d41
		ps776.OverlayValues[42] = d42
		ps776.OverlayValues[43] = d43
		ps776.OverlayValues[44] = d44
		ps776.OverlayValues[45] = d45
		ps776.OverlayValues[46] = d46
		ps776.OverlayValues[47] = d47
		ps776.OverlayValues[48] = d48
		ps776.OverlayValues[49] = d49
		ps776.OverlayValues[52] = d52
		ps776.OverlayValues[53] = d53
		ps776.OverlayValues[54] = d54
		ps776.OverlayValues[109] = d109
		ps776.OverlayValues[110] = d110
		ps776.OverlayValues[111] = d111
		ps776.OverlayValues[112] = d112
		ps776.OverlayValues[113] = d113
		ps776.OverlayValues[114] = d114
		ps776.OverlayValues[115] = d115
		ps776.OverlayValues[116] = d116
		ps776.OverlayValues[117] = d117
		ps776.OverlayValues[118] = d118
		ps776.OverlayValues[119] = d119
		ps776.OverlayValues[120] = d120
		ps776.OverlayValues[121] = d121
		ps776.OverlayValues[122] = d122
		ps776.OverlayValues[123] = d123
		ps776.OverlayValues[124] = d124
		ps776.OverlayValues[125] = d125
		ps776.OverlayValues[126] = d126
		ps776.OverlayValues[127] = d127
		ps776.OverlayValues[128] = d128
		ps776.OverlayValues[129] = d129
		ps776.OverlayValues[130] = d130
		ps776.OverlayValues[131] = d131
		ps776.OverlayValues[132] = d132
		ps776.OverlayValues[133] = d133
		ps776.OverlayValues[134] = d134
		ps776.OverlayValues[135] = d135
		ps776.OverlayValues[136] = d136
		ps776.OverlayValues[137] = d137
		ps776.OverlayValues[140] = d140
		ps776.OverlayValues[225] = d225
		ps776.OverlayValues[226] = d226
		ps776.OverlayValues[227] = d227
		ps776.OverlayValues[228] = d228
		ps776.OverlayValues[230] = d230
		ps776.OverlayValues[231] = d231
		ps776.OverlayValues[232] = d232
		ps776.OverlayValues[233] = d233
		ps776.OverlayValues[234] = d234
		ps776.OverlayValues[235] = d235
		ps776.OverlayValues[236] = d236
		ps776.OverlayValues[237] = d237
		ps776.OverlayValues[239] = d239
		ps776.OverlayValues[241] = d241
		ps776.OverlayValues[242] = d242
		ps776.OverlayValues[243] = d243
		ps776.OverlayValues[244] = d244
		ps776.OverlayValues[245] = d245
		ps776.OverlayValues[248] = d248
		ps776.OverlayValues[350] = d350
		ps776.OverlayValues[351] = d351
		ps776.OverlayValues[352] = d352
		ps776.OverlayValues[353] = d353
		ps776.OverlayValues[354] = d354
		ps776.OverlayValues[356] = d356
		ps776.OverlayValues[357] = d357
		ps776.OverlayValues[358] = d358
		ps776.OverlayValues[359] = d359
		ps776.OverlayValues[360] = d360
		ps776.OverlayValues[361] = d361
		ps776.OverlayValues[362] = d362
		ps776.OverlayValues[363] = d363
		ps776.OverlayValues[364] = d364
		ps776.OverlayValues[365] = d365
		ps776.OverlayValues[366] = d366
		ps776.OverlayValues[367] = d367
		ps776.OverlayValues[368] = d368
		ps776.OverlayValues[369] = d369
		ps776.OverlayValues[370] = d370
		ps776.OverlayValues[371] = d371
		ps776.OverlayValues[372] = d372
		ps776.OverlayValues[373] = d373
		ps776.OverlayValues[374] = d374
		ps776.OverlayValues[375] = d375
		ps776.OverlayValues[376] = d376
		ps776.OverlayValues[377] = d377
		ps776.OverlayValues[378] = d378
		ps776.OverlayValues[379] = d379
		ps776.OverlayValues[380] = d380
		ps776.OverlayValues[381] = d381
		ps776.OverlayValues[382] = d382
		ps776.OverlayValues[383] = d383
		ps776.OverlayValues[384] = d384
		ps776.OverlayValues[524] = d524
		ps776.OverlayValues[525] = d525
		ps776.OverlayValues[526] = d526
		ps776.OverlayValues[528] = d528
		ps776.OverlayValues[529] = d529
		ps776.OverlayValues[530] = d530
		ps776.OverlayValues[531] = d531
		ps776.OverlayValues[532] = d532
		ps776.OverlayValues[533] = d533
		ps776.OverlayValues[534] = d534
		ps776.OverlayValues[536] = d536
		ps776.OverlayValues[538] = d538
		ps776.OverlayValues[539] = d539
		ps776.OverlayValues[540] = d540
		ps776.OverlayValues[541] = d541
		ps776.OverlayValues[544] = d544
		ps776.OverlayValues[696] = d696
		ps776.OverlayValues[697] = d697
		ps776.OverlayValues[698] = d698
		ps776.OverlayValues[699] = d699
		ps776.OverlayValues[701] = d701
		ps776.OverlayValues[702] = d702
		ps776.OverlayValues[703] = d703
		ps776.OverlayValues[704] = d704
		ps776.OverlayValues[705] = d705
		ps776.OverlayValues[706] = d706
		ps776.OverlayValues[707] = d707
		ps776.OverlayValues[708] = d708
		ps776.OverlayValues[710] = d710
		ps776.OverlayValues[711] = d711
		ps776.OverlayValues[712] = d712
		ps776.OverlayValues[713] = d713
		ps776.OverlayValues[714] = d714
		ps776.OverlayValues[715] = d715
		ps776.OverlayValues[716] = d716
		ps776.OverlayValues[717] = d717
		ps776.OverlayValues[718] = d718
		ps776.OverlayValues[719] = d719
		ps776.OverlayValues[720] = d720
		ps776.OverlayValues[721] = d721
		ps776.OverlayValues[722] = d722
		ps776.OverlayValues[723] = d723
		ps776.OverlayValues[724] = d724
		ps776.OverlayValues[725] = d725
		ps776.OverlayValues[726] = d726
		ps776.OverlayValues[727] = d727
		ps776.OverlayValues[728] = d728
		ps776.OverlayValues[729] = d729
		ps776.OverlayValues[730] = d730
		ps776.OverlayValues[731] = d731
		ps776.OverlayValues[732] = d732
		ps776.OverlayValues[733] = d733
		ps776.OverlayValues[734] = d734
		ps776.OverlayValues[735] = d735
		ps776.OverlayValues[736] = d736
		ps776.OverlayValues[737] = d737
		ps776.OverlayValues[738] = d738
		ps776.OverlayValues[739] = d739
		ps776.OverlayValues[740] = d740
		ps776.OverlayValues[741] = d741
		ps776.OverlayValues[742] = d742
		ps776.OverlayValues[743] = d743
		ps776.OverlayValues[744] = d744
		ps776.OverlayValues[745] = d745
		ps776.OverlayValues[746] = d746
		ps776.OverlayValues[747] = d747
		ps776.OverlayValues[748] = d748
		ps776.OverlayValues[749] = d749
		ps776.OverlayValues[750] = d750
		ps776.OverlayValues[751] = d751
		ps776.OverlayValues[752] = d752
		ps776.OverlayValues[753] = d753
		ps776.OverlayValues[754] = d754
		ps776.OverlayValues[755] = d755
		ps776.OverlayValues[756] = d756
		ps776.OverlayValues[757] = d757
		ps776.OverlayValues[758] = d758
		ps776.OverlayValues[759] = d759
		ps776.OverlayValues[760] = d760
		ps776.OverlayValues[761] = d761
		ps776.OverlayValues[762] = d762
		ps776.OverlayValues[763] = d763
		ps776.OverlayValues[764] = d764
		ps776.OverlayValues[765] = d765
		ps776.OverlayValues[766] = d766
		ps776.OverlayValues[767] = d767
		ps776.OverlayValues[768] = d768
		ps776.OverlayValues[769] = d769
		ps776.OverlayValues[770] = d770
		ps776.OverlayValues[771] = d771
		ps776.OverlayValues[772] = d772
		snap777 := d1
		snap778 := d2
		snap779 := d3
		snap780 := d4
		snap781 := d5
		snap782 := d6
		snap783 := d7
		snap784 := d8
		snap785 := d9
		snap786 := d10
		snap787 := d11
		snap788 := d12
		snap789 := d13
		snap790 := d14
		snap791 := d15
		snap792 := d17
		snap793 := d18
		snap794 := d19
		snap795 := d20
		snap796 := d21
		snap797 := d22
		snap798 := d23
		snap799 := d24
		snap800 := d25
		snap801 := d26
		snap802 := d27
		snap803 := d28
		snap804 := d29
		snap805 := d30
		snap806 := d31
		snap807 := d32
		snap808 := d33
		snap809 := d34
		snap810 := d35
		snap811 := d36
		snap812 := d37
		snap813 := d38
		snap814 := d39
		snap815 := d40
		snap816 := d41
		snap817 := d42
		snap818 := d43
		snap819 := d44
		snap820 := d45
		snap821 := d46
		snap822 := d47
		snap823 := d48
		snap824 := d49
		snap825 := d52
		snap826 := d53
		snap827 := d54
		snap828 := d109
		snap829 := d110
		snap830 := d111
		snap831 := d112
		snap832 := d113
		snap833 := d114
		snap834 := d115
		snap835 := d116
		snap836 := d117
		snap837 := d118
		snap838 := d119
		snap839 := d120
		snap840 := d121
		snap841 := d122
		snap842 := d123
		snap843 := d124
		snap844 := d125
		snap845 := d126
		snap846 := d127
		snap847 := d128
		snap848 := d129
		snap849 := d130
		snap850 := d131
		snap851 := d132
		snap852 := d133
		snap853 := d134
		snap854 := d135
		snap855 := d136
		snap856 := d137
		snap857 := d140
		snap858 := d225
		snap859 := d226
		snap860 := d227
		snap861 := d228
		snap862 := d230
		snap863 := d231
		snap864 := d232
		snap865 := d233
		snap866 := d234
		snap867 := d235
		snap868 := d236
		snap869 := d237
		snap870 := d239
		snap871 := d241
		snap872 := d242
		snap873 := d243
		snap874 := d244
		snap875 := d245
		snap876 := d248
		snap877 := d350
		snap878 := d351
		snap879 := d352
		snap880 := d353
		snap881 := d354
		snap882 := d356
		snap883 := d357
		snap884 := d358
		snap885 := d359
		snap886 := d360
		snap887 := d361
		snap888 := d362
		snap889 := d363
		snap890 := d364
		snap891 := d365
		snap892 := d366
		snap893 := d367
		snap894 := d368
		snap895 := d369
		snap896 := d370
		snap897 := d371
		snap898 := d372
		snap899 := d373
		snap900 := d374
		snap901 := d375
		snap902 := d376
		snap903 := d377
		snap904 := d378
		snap905 := d379
		snap906 := d380
		snap907 := d381
		snap908 := d382
		snap909 := d383
		snap910 := d384
		snap911 := d524
		snap912 := d525
		snap913 := d526
		snap914 := d528
		snap915 := d529
		snap916 := d530
		snap917 := d531
		snap918 := d532
		snap919 := d533
		snap920 := d534
		snap921 := d536
		snap922 := d538
		snap923 := d539
		snap924 := d540
		snap925 := d541
		snap926 := d544
		snap927 := d696
		snap928 := d697
		snap929 := d698
		snap930 := d699
		snap931 := d701
		snap932 := d702
		snap933 := d703
		snap934 := d704
		snap935 := d705
		snap936 := d706
		snap937 := d707
		snap938 := d708
		snap939 := d710
		snap940 := d711
		snap941 := d712
		snap942 := d713
		snap943 := d714
		snap944 := d715
		snap945 := d716
		snap946 := d717
		snap947 := d718
		snap948 := d719
		snap949 := d720
		snap950 := d721
		snap951 := d722
		snap952 := d723
		snap953 := d724
		snap954 := d725
		snap955 := d726
		snap956 := d727
		snap957 := d728
		snap958 := d729
		snap959 := d730
		snap960 := d731
		snap961 := d732
		snap962 := d733
		snap963 := d734
		snap964 := d735
		snap965 := d736
		snap966 := d737
		snap967 := d738
		snap968 := d739
		snap969 := d740
		snap970 := d741
		snap971 := d742
		snap972 := d743
		snap973 := d744
		snap974 := d745
		snap975 := d746
		snap976 := d747
		snap977 := d748
		snap978 := d749
		snap979 := d750
		snap980 := d751
		snap981 := d752
		snap982 := d753
		snap983 := d754
		snap984 := d755
		snap985 := d756
		snap986 := d757
		snap987 := d758
		snap988 := d759
		snap989 := d760
		snap990 := d761
		snap991 := d762
		snap992 := d763
		snap993 := d764
		snap994 := d765
		snap995 := d766
		snap996 := d767
		snap997 := d768
		snap998 := d769
		snap999 := d770
		snap1000 := d771
		snap1001 := d772
		alloc1002 := ctx.SnapshotAllocState()
		if !bbs[12].Rendered {
			bbs[12].RenderPS(ps776)
		}
		ctx.RestoreAllocState(alloc1002)
		d1 = snap777
		d2 = snap778
		d3 = snap779
		d4 = snap780
		d5 = snap781
		d6 = snap782
		d7 = snap783
		d8 = snap784
		d9 = snap785
		d10 = snap786
		d11 = snap787
		d12 = snap788
		d13 = snap789
		d14 = snap790
		d15 = snap791
		d17 = snap792
		d18 = snap793
		d19 = snap794
		d20 = snap795
		d21 = snap796
		d22 = snap797
		d23 = snap798
		d24 = snap799
		d25 = snap800
		d26 = snap801
		d27 = snap802
		d28 = snap803
		d29 = snap804
		d30 = snap805
		d31 = snap806
		d32 = snap807
		d33 = snap808
		d34 = snap809
		d35 = snap810
		d36 = snap811
		d37 = snap812
		d38 = snap813
		d39 = snap814
		d40 = snap815
		d41 = snap816
		d42 = snap817
		d43 = snap818
		d44 = snap819
		d45 = snap820
		d46 = snap821
		d47 = snap822
		d48 = snap823
		d49 = snap824
		d52 = snap825
		d53 = snap826
		d54 = snap827
		d109 = snap828
		d110 = snap829
		d111 = snap830
		d112 = snap831
		d113 = snap832
		d114 = snap833
		d115 = snap834
		d116 = snap835
		d117 = snap836
		d118 = snap837
		d119 = snap838
		d120 = snap839
		d121 = snap840
		d122 = snap841
		d123 = snap842
		d124 = snap843
		d125 = snap844
		d126 = snap845
		d127 = snap846
		d128 = snap847
		d129 = snap848
		d130 = snap849
		d131 = snap850
		d132 = snap851
		d133 = snap852
		d134 = snap853
		d135 = snap854
		d136 = snap855
		d137 = snap856
		d140 = snap857
		d225 = snap858
		d226 = snap859
		d227 = snap860
		d228 = snap861
		d230 = snap862
		d231 = snap863
		d232 = snap864
		d233 = snap865
		d234 = snap866
		d235 = snap867
		d236 = snap868
		d237 = snap869
		d239 = snap870
		d241 = snap871
		d242 = snap872
		d243 = snap873
		d244 = snap874
		d245 = snap875
		d248 = snap876
		d350 = snap877
		d351 = snap878
		d352 = snap879
		d353 = snap880
		d354 = snap881
		d356 = snap882
		d357 = snap883
		d358 = snap884
		d359 = snap885
		d360 = snap886
		d361 = snap887
		d362 = snap888
		d363 = snap889
		d364 = snap890
		d365 = snap891
		d366 = snap892
		d367 = snap893
		d368 = snap894
		d369 = snap895
		d370 = snap896
		d371 = snap897
		d372 = snap898
		d373 = snap899
		d374 = snap900
		d375 = snap901
		d376 = snap902
		d377 = snap903
		d378 = snap904
		d379 = snap905
		d380 = snap906
		d381 = snap907
		d382 = snap908
		d383 = snap909
		d384 = snap910
		d524 = snap911
		d525 = snap912
		d526 = snap913
		d528 = snap914
		d529 = snap915
		d530 = snap916
		d531 = snap917
		d532 = snap918
		d533 = snap919
		d534 = snap920
		d536 = snap921
		d538 = snap922
		d539 = snap923
		d540 = snap924
		d541 = snap925
		d544 = snap926
		d696 = snap927
		d697 = snap928
		d698 = snap929
		d699 = snap930
		d701 = snap931
		d702 = snap932
		d703 = snap933
		d704 = snap934
		d705 = snap935
		d706 = snap936
		d707 = snap937
		d708 = snap938
		d710 = snap939
		d711 = snap940
		d712 = snap941
		d713 = snap942
		d714 = snap943
		d715 = snap944
		d716 = snap945
		d717 = snap946
		d718 = snap947
		d719 = snap948
		d720 = snap949
		d721 = snap950
		d722 = snap951
		d723 = snap952
		d724 = snap953
		d725 = snap954
		d726 = snap955
		d727 = snap956
		d728 = snap957
		d729 = snap958
		d730 = snap959
		d731 = snap960
		d732 = snap961
		d733 = snap962
		d734 = snap963
		d735 = snap964
		d736 = snap965
		d737 = snap966
		d738 = snap967
		d739 = snap968
		d740 = snap969
		d741 = snap970
		d742 = snap971
		d743 = snap972
		d744 = snap973
		d745 = snap974
		d746 = snap975
		d747 = snap976
		d748 = snap977
		d749 = snap978
		d750 = snap979
		d751 = snap980
		d752 = snap981
		d753 = snap982
		d754 = snap983
		d755 = snap984
		d756 = snap985
		d757 = snap986
		d758 = snap987
		d759 = snap988
		d760 = snap989
		d761 = snap990
		d762 = snap991
		d763 = snap992
		d764 = snap993
		d765 = snap994
		d766 = snap995
		d767 = snap996
		d768 = snap997
		d769 = snap998
		d770 = snap999
		d771 = snap1000
		d772 = snap1001
		if !bbs[11].Rendered {
			return bbs[11].RenderPS(ps775)
		}
		return result
		ctx.FreeDesc(&d771)
		return result
	}
	ps1003 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps1003)
	ctx.MarkLabel(lbl0)
	d1004 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
	ctx.BindReg(r0, &d1004)
	ctx.BindReg(r1, &d1004)
	ctx.EmitMovPairToResult(&d1004, &result)
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
