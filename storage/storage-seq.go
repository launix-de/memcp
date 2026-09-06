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
	var d12 scm.JITValueDesc
	_ = d12
	var d13 scm.JITValueDesc
	_ = d13
	var d14 scm.JITValueDesc
	_ = d14
	var d15 scm.JITValueDesc
	_ = d15
	var d16 scm.JITValueDesc
	_ = d16
	var d17 scm.JITValueDesc
	_ = d17
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
	var d51 scm.JITValueDesc
	_ = d51
	var d54 scm.JITValueDesc
	_ = d54
	var d55 scm.JITValueDesc
	_ = d55
	var d56 scm.JITValueDesc
	_ = d56
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
	var d142 scm.JITValueDesc
	_ = d142
	var d227 scm.JITValueDesc
	_ = d227
	var d228 scm.JITValueDesc
	_ = d228
	var d229 scm.JITValueDesc
	_ = d229
	var d230 scm.JITValueDesc
	_ = d230
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
	var d238 scm.JITValueDesc
	_ = d238
	var d239 scm.JITValueDesc
	_ = d239
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
	var d250 scm.JITValueDesc
	_ = d250
	var d352 scm.JITValueDesc
	_ = d352
	var d353 scm.JITValueDesc
	_ = d353
	var d354 scm.JITValueDesc
	_ = d354
	var d355 scm.JITValueDesc
	_ = d355
	var d356 scm.JITValueDesc
	_ = d356
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
	var d385 scm.JITValueDesc
	_ = d385
	var d386 scm.JITValueDesc
	_ = d386
	var d526 scm.JITValueDesc
	_ = d526
	var d527 scm.JITValueDesc
	_ = d527
	var d528 scm.JITValueDesc
	_ = d528
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
	var d535 scm.JITValueDesc
	_ = d535
	var d536 scm.JITValueDesc
	_ = d536
	var d538 scm.JITValueDesc
	_ = d538
	var d540 scm.JITValueDesc
	_ = d540
	var d541 scm.JITValueDesc
	_ = d541
	var d542 scm.JITValueDesc
	_ = d542
	var d543 scm.JITValueDesc
	_ = d543
	var d546 scm.JITValueDesc
	_ = d546
	var d698 scm.JITValueDesc
	_ = d698
	var d699 scm.JITValueDesc
	_ = d699
	var d700 scm.JITValueDesc
	_ = d700
	var d701 scm.JITValueDesc
	_ = d701
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
	registerHomes1 := ctx.AllocRegisterHomes(1)
	defer ctx.ReleaseRegisterHomes(registerHomes1)
	var r0 Reg
	phiHomeOK2 := int(registerHomes1.Count) > 0
	if phiHomeOK2 {
		r0 = registerHomes1.Registers[0]
	}
	var d3 scm.JITValueDesc
	if phiHomeOK2 {
		d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0}
		ctx.BindReg(r0, &d3)
	} else {
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
	}
	_ = d3
	d4 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
	_ = d4
	d5 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
	_ = d5
	d6 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
	_ = d6
	d7 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
	_ = d7
	d8 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
	_ = d8
	d9 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
	_ = d9
	d10 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
	_ = d10
	d11 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
	_ = d11
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
	r1 := ctx.AllocReg()
	r2 := ctx.AllocRegExcept(r1)
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
			d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0}
			ctx.BindReg(r0, &d3)
		} else {
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		ctx.ReclaimUntrackedRegs()
		r3 := ctx.AllocReg()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).lastValue)
			ctx.EmitMovRegMem64(r3, fieldAddr)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
			ctx.EmitMovRegMem(r3, thisptr.Reg, off)
		}
		d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r3}
		ctx.BindReg(r3, &d12)
		ctx.EnsureDesc(&d12)
		ctx.EnsureDesc(&d12)
		var d13 scm.JITValueDesc
		if d12.Loc == scm.LocImm {
			d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d12.Imm.Int()))))}
		} else {
			r4 := ctx.AllocReg()
			ctx.EmitMovRegReg(r4, d12.Reg)
			ctx.EmitShlRegImm8(r4, 32)
			ctx.EmitShrRegImm8(r4, 32)
			d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r4}
			ctx.BindReg(r4, &d13)
		}
		ctx.StabilizeDescForControlFlow(&d13)
		ctx.FreeDesc(&d12)
		var d14 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).seqCount)
			r5 := ctx.AllocReg()
			ctx.EmitMovRegMem32(r5, fieldAddr)
			d14 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r5}
			ctx.BindReg(r5, &d14)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).seqCount))
			r6 := ctx.AllocReg()
			ctx.EmitMovRegMemL(r6, thisptr.Reg, off)
			d14 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r6}
			ctx.BindReg(r6, &d14)
		}
		ctx.EnsureDesc(&d14)
		ctx.EnsureDesc(&d14)
		var d15 scm.JITValueDesc
		if d14.Loc == scm.LocImm {
			d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d14.Reg)
			ctx.EmitMovRegReg(scratch, d14.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d15)
		}
		if d15.Loc == scm.LocImm {
			d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: d15.Type, Imm: scm.NewInt(int64(uint64(d15.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d15.Reg, 32)
			ctx.EmitShrRegImm8(d15.Reg, 32)
		}
		if d15.Loc == scm.LocReg && d14.Loc == scm.LocReg && d15.Reg == d14.Reg {
			ctx.TransferReg(d14.Reg)
			d14.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d15)
		ctx.EmitStoreToStack(d15, int32(bbs[1].PhiBase)+int32(32))
		ctx.StabilizeDescForControlFlow(&d15)
		if ps.General {
			ctx.SyncDesc(&d13)
			if d13.Loc == scm.LocReg {
				ctx.ProtectReg(d13.Reg)
			} else if d13.Loc == scm.LocRegPair {
				ctx.ProtectReg(d13.Reg)
				ctx.ProtectReg(d13.Reg2)
			}
			d16 = d13
			if d16.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d16)
			d17 = d16
			if d17.Loc == scm.LocImm {
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: d17.Type, Imm: scm.NewInt(int64(uint64(d17.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d17.Reg, 32)
				ctx.EmitShrRegImm8(d17.Reg, 32)
			}
			if phiHomeOK2 {
				ctx.EmitMovToReg(r0, d17)
			} else {
				ctx.EmitStoreToStack(d17, int32(bbs[1].PhiBase)+int32(0))
			}
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(bbs[1].PhiBase)+int32(16))
			if d13.Loc == scm.LocReg {
				ctx.UnprotectReg(d13.Reg)
			} else if d13.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d13.Reg)
				ctx.UnprotectReg(d13.Reg2)
			}
		}
		ps18 := scm.PhiState{General: ps.General}
		ps18.OverlayValues = make([]scm.JITValueDesc, 18)
		ps18.OverlayValues[3] = d3
		ps18.OverlayValues[4] = d4
		ps18.OverlayValues[5] = d5
		ps18.OverlayValues[6] = d6
		ps18.OverlayValues[7] = d7
		ps18.OverlayValues[8] = d8
		ps18.OverlayValues[9] = d9
		ps18.OverlayValues[10] = d10
		ps18.OverlayValues[11] = d11
		ps18.OverlayValues[12] = d12
		ps18.OverlayValues[13] = d13
		ps18.OverlayValues[14] = d14
		ps18.OverlayValues[15] = d15
		ps18.OverlayValues[16] = d16
		ps18.OverlayValues[17] = d17
		ps18.PhiValues = make([]scm.JITValueDesc, 3)
		d19 = d13
		ps18.PhiValues[0] = d19
		d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
		ps18.PhiValues[1] = d20
		if ps18.General && bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		return bbs[1].RenderPS(ps18)
		return result
	}
	bbs[1].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d21 := ps.PhiValues[0]
				ctx.EnsureDesc(&d21)
				ctx.EmitStoreToStack(d21, int32(bbs[1].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d22 := ps.PhiValues[1]
				ctx.EnsureDesc(&d22)
				ctx.EmitStoreToStack(d22, int32(bbs[1].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d23 := ps.PhiValues[2]
				ctx.EnsureDesc(&d23)
				ctx.EmitStoreToStack(d23, int32(bbs[1].PhiBase)+int32(32))
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
			d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0}
			ctx.BindReg(r0, &d3)
		} else {
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
			d16 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
			d17 = ps.OverlayValues[17]
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
		if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
			d3 = ps.PhiValues[0]
		}
		if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
			d4 = ps.PhiValues[1]
		}
		if !ps.General && len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
			d5 = ps.PhiValues[2]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d4)
		ctx.StabilizeDescForControlFlow(&d5)
		ctx.EnsureDesc(&d3)
		d24 = d3
		_ = d24
		ctx.StabilizeDescForControlFlow(&d24)
		ctx.StabilizeDescForControlFlow(&d3)
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		lbl15 := ctx.ReserveLabel()
		_ = lbl15
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl15)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d24)
		ctx.EnsureDesc(&d24)
		var d25 scm.JITValueDesc
		if d24.Loc == scm.LocImm {
			d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d24.Imm.Int()))))}
		} else {
			r7 := ctx.AllocReg()
			ctx.EmitMovRegReg(r7, d24.Reg)
			ctx.EmitShlRegImm8(r7, 32)
			ctx.EmitShrRegImm8(r7, 32)
			d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
			ctx.BindReg(r7, &d25)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d26 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			r8 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r8, fieldAddr)
			d26 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r8}
			ctx.BindReg(r8, &d26)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			r9 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r9, thisptr.Reg, off)
			d26 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r9}
			ctx.BindReg(r9, &d26)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d26)
		ctx.EnsureDesc(&d26)
		var d27 scm.JITValueDesc
		if d26.Loc == scm.LocImm {
			d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d26.Imm.Int()))))}
		} else {
			r10 := ctx.AllocReg()
			ctx.EmitMovRegReg(r10, d26.Reg)
			ctx.EmitShlRegImm8(r10, 56)
			ctx.EmitShrRegImm8(r10, 56)
			d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
			ctx.BindReg(r10, &d27)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d25)
		ctx.EnsureDesc(&d27)
		ctx.EnsureDescsTogether(&d25, &d27)
		var d28 scm.JITValueDesc
		if d25.Loc == scm.LocImm && d27.Loc == scm.LocImm {
			d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d25.Imm.Int() * d27.Imm.Int())}
		} else if d25.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d27.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d25.Imm.Int()))
			ctx.EmitImulInt64(scratch, d27.Reg)
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d28)
		} else if d27.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d25.Reg)
			ctx.EmitMovRegReg(scratch, d25.Reg)
			if d27.Imm.Int() >= -2147483648 && d27.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d27.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d27.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d28)
		} else {
			r11 := ctx.AllocRegExcept(d25.Reg, d27.Reg)
			ctx.EmitMovRegReg(r11, d25.Reg)
			ctx.EmitImulInt64(r11, d27.Reg)
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
			ctx.BindReg(r11, &d28)
		}
		if d28.Loc == scm.LocReg && d25.Loc == scm.LocReg && d28.Reg == d25.Reg {
			ctx.TransferReg(d25.Reg)
			d25.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d25)
		ctx.FreeDesc(&d27)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d28)
		var d29 scm.JITValueDesc
		if d28.Loc == scm.LocImm {
			d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d28.Imm.Int() / 64)}
		} else {
			r12 := ctx.AllocRegExcept(d28.Reg)
			ctx.EmitMovRegReg(r12, d28.Reg)
			ctx.EmitShrRegImm8(r12, 6)
			d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
			ctx.BindReg(r12, &d29)
		}
		if d29.Loc == scm.LocReg && d28.Loc == scm.LocReg && d29.Reg == d28.Reg {
			ctx.TransferReg(d28.Reg)
			d28.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d28)
		var d30 scm.JITValueDesc
		if d28.Loc == scm.LocImm {
			d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d28.Imm.Int() % 64)}
		} else {
			r13 := ctx.AllocRegExcept(d28.Reg)
			ctx.EmitMovRegReg(r13, d28.Reg)
			ctx.EmitAndRegImm32(r13, 63)
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
			ctx.BindReg(r13, &d30)
		}
		if d30.Loc == scm.LocReg && d28.Loc == scm.LocReg && d30.Reg == d28.Reg {
			ctx.TransferReg(d28.Reg)
			d28.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d28)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d31 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
			r14 := ctx.AllocReg()
			r15 := ctx.AllocRegExcept(r14)
			r16 := ctx.AllocRegExcept(r14, r15)
			ctx.EmitMovRegMem64(r14, fieldAddr)
			ctx.EmitMovRegMem64(r15, fieldAddr+8)
			ctx.EmitMovRegMem64(r16, fieldAddr+16)
			d31 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r14, Reg2: r15, Reg3: r16}
			ctx.BindReg(r14, &d31)
			ctx.BindReg(r15, &d31)
			ctx.BindReg(r16, &d31)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
			r17 := ctx.AllocReg()
			r18 := ctx.AllocRegExcept(r17)
			r19 := ctx.AllocRegExcept(r17, r18)
			ctx.EmitMovRegMem(r17, thisptr.Reg, off)
			ctx.EmitMovRegMem(r18, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r19, thisptr.Reg, off+16)
			d31 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r17, Reg2: r18, Reg3: r19}
			ctx.BindReg(r17, &d31)
			ctx.BindReg(r18, &d31)
			ctx.BindReg(r19, &d31)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d29)
		ctx.ReclaimUntrackedRegs()
		d33 = ctx.EmitSliceElementAddress(&d31, &d29, 8)
		ctx.EnsureDesc(&d33)
		ctx.EmitMovRegMem(d33.Reg, d33.Reg, 0)
		d32 = d33
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d32)
		ctx.EnsureDesc(&d30)
		var d34 scm.JITValueDesc
		if d32.Loc == scm.LocImm && d30.Loc == scm.LocImm {
			d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d32.Imm.Int()) << uint64(d30.Imm.Int())))}
		} else if d30.Loc == scm.LocImm {
			r20 := ctx.AllocRegExcept(d32.Reg)
			ctx.EmitMovRegReg(r20, d32.Reg)
			ctx.EmitShlRegImm8(r20, uint8(d30.Imm.Int()))
			d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
			ctx.BindReg(r20, &d34)
		} else {
			{
				shiftSrc := d32.Reg
				r21 := ctx.AllocRegExcept(d32.Reg)
				ctx.EmitMovRegReg(r21, d32.Reg)
				shiftSrc = r21
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d30.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d30.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d30.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d34)
			}
		}
		if d34.Loc == scm.LocReg && d32.Loc == scm.LocReg && d34.Reg == d32.Reg {
			ctx.TransferReg(d32.Reg)
			d32.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d32)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d29)
		ctx.EnsureDesc(&d29)
		var d35 scm.JITValueDesc
		if d29.Loc == scm.LocImm {
			d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d29.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d29.Reg)
			ctx.EmitMovRegReg(scratch, d29.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d35)
		}
		if d35.Loc == scm.LocReg && d29.Loc == scm.LocReg && d35.Reg == d29.Reg {
			ctx.TransferReg(d29.Reg)
			d29.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d29)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d35)
		ctx.ReclaimUntrackedRegs()
		d37 = ctx.EmitSliceElementAddress(&d31, &d35, 8)
		ctx.EnsureDesc(&d37)
		ctx.EmitMovRegMem(d37.Reg, d37.Reg, 0)
		d36 = d37
		ctx.FreeDesc(&d35)
		ctx.ReclaimUntrackedRegs()
		d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d30)
		ctx.EnsureDescsTogether(&d38, &d30)
		var d39 scm.JITValueDesc
		if d38.Loc == scm.LocImm && d30.Loc == scm.LocImm {
			d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d38.Imm.Int() - d30.Imm.Int())}
		} else if d30.Loc == scm.LocImm && d30.Imm.Int() == 0 {
			r22 := ctx.AllocRegExcept(d38.Reg)
			ctx.EmitMovRegReg(r22, d38.Reg)
			d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
			ctx.BindReg(r22, &d39)
		} else if d38.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d30.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d38.Imm.Int()))
			ctx.EmitSubInt64(scratch, d30.Reg)
			d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d39)
		} else if d30.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d38.Reg)
			ctx.EmitMovRegReg(scratch, d38.Reg)
			if d30.Imm.Int() >= -2147483648 && d30.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d30.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d30.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d39)
		} else {
			r23 := ctx.AllocRegExcept(d38.Reg, d30.Reg)
			ctx.EmitMovRegReg(r23, d38.Reg)
			ctx.EmitSubInt64(r23, d30.Reg)
			d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
			ctx.BindReg(r23, &d39)
		}
		if d39.Loc == scm.LocReg && d38.Loc == scm.LocReg && d39.Reg == d38.Reg {
			ctx.TransferReg(d38.Reg)
			d38.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d30)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d36)
		ctx.EnsureDesc(&d39)
		var d40 scm.JITValueDesc
		if d36.Loc == scm.LocImm && d39.Loc == scm.LocImm {
			d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d36.Imm.Int()) >> uint64(d39.Imm.Int())))}
		} else if d39.Loc == scm.LocImm {
			r24 := ctx.AllocRegExcept(d36.Reg)
			ctx.EmitMovRegReg(r24, d36.Reg)
			ctx.EmitShrRegImm8(r24, uint8(d39.Imm.Int()))
			d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d40)
		} else {
			{
				shiftSrc := d36.Reg
				r25 := ctx.AllocRegExcept(d36.Reg)
				ctx.EmitMovRegReg(r25, d36.Reg)
				shiftSrc = r25
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d39.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d39.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d39.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d40)
			}
		}
		if d40.Loc == scm.LocReg && d36.Loc == scm.LocReg && d40.Reg == d36.Reg {
			ctx.TransferReg(d36.Reg)
			d36.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d36)
		ctx.FreeDesc(&d39)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d34)
		ctx.EnsureDesc(&d40)
		var d41 scm.JITValueDesc
		if d34.Loc == scm.LocImm && d40.Loc == scm.LocImm {
			d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d34.Imm.Int() | d40.Imm.Int())}
		} else if d34.Loc == scm.LocImm && d34.Imm.Int() == 0 {
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d40.Reg}
			ctx.BindReg(d40.Reg, &d41)
		} else if d40.Loc == scm.LocImm && d40.Imm.Int() == 0 {
			r26 := ctx.AllocRegExcept(d34.Reg)
			ctx.EmitMovRegReg(r26, d34.Reg)
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
			ctx.BindReg(r26, &d41)
		} else if d34.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d40.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d34.Imm.Int()))
			ctx.EmitOrInt64(scratch, d40.Reg)
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d41)
		} else if d40.Loc == scm.LocImm {
			r27 := ctx.AllocRegExcept(d34.Reg)
			ctx.EmitMovRegReg(r27, d34.Reg)
			if d40.Imm.Int() >= -2147483648 && d40.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r27, int32(d40.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
				ctx.EmitOrInt64(r27, scm.RegR11)
			}
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
			ctx.BindReg(r27, &d41)
		} else {
			r28 := ctx.AllocRegExcept(d34.Reg, d40.Reg)
			ctx.EmitMovRegReg(r28, d34.Reg)
			ctx.EmitOrInt64(r28, d40.Reg)
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
			ctx.BindReg(r28, &d41)
		}
		if d41.Loc == scm.LocReg && d34.Loc == scm.LocReg && d41.Reg == d34.Reg {
			ctx.TransferReg(d34.Reg)
			d34.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d34)
		ctx.FreeDesc(&d40)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d26)
		ctx.EnsureDesc(&d26)
		var d42 scm.JITValueDesc
		if d26.Loc == scm.LocImm {
			d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d26.Imm.Int()))))}
		} else {
			r29 := ctx.AllocReg()
			ctx.EmitMovRegReg(r29, d26.Reg)
			ctx.EmitShlRegImm8(r29, 56)
			ctx.EmitShrRegImm8(r29, 56)
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
			ctx.BindReg(r29, &d42)
		}
		ctx.ReclaimUntrackedRegs()
		d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d42)
		ctx.EnsureDescsTogether(&d43, &d42)
		var d44 scm.JITValueDesc
		if d43.Loc == scm.LocImm && d42.Loc == scm.LocImm {
			d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d43.Imm.Int() - d42.Imm.Int())}
		} else if d42.Loc == scm.LocImm && d42.Imm.Int() == 0 {
			r30 := ctx.AllocRegExcept(d43.Reg)
			ctx.EmitMovRegReg(r30, d43.Reg)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
			ctx.BindReg(r30, &d44)
		} else if d43.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d42.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d43.Imm.Int()))
			ctx.EmitSubInt64(scratch, d42.Reg)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d44)
		} else if d42.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d43.Reg)
			ctx.EmitMovRegReg(scratch, d43.Reg)
			if d42.Imm.Int() >= -2147483648 && d42.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d42.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d42.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d44)
		} else {
			r31 := ctx.AllocRegExcept(d43.Reg, d42.Reg)
			ctx.EmitMovRegReg(r31, d43.Reg)
			ctx.EmitSubInt64(r31, d42.Reg)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
			ctx.BindReg(r31, &d44)
		}
		if d44.Loc == scm.LocReg && d43.Loc == scm.LocReg && d44.Reg == d43.Reg {
			ctx.TransferReg(d43.Reg)
			d43.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d42)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d41)
		ctx.EnsureDesc(&d44)
		var d45 scm.JITValueDesc
		if d41.Loc == scm.LocImm && d44.Loc == scm.LocImm {
			d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d41.Imm.Int()) >> uint64(d44.Imm.Int())))}
		} else if d44.Loc == scm.LocImm {
			r32 := ctx.AllocRegExcept(d41.Reg)
			ctx.EmitMovRegReg(r32, d41.Reg)
			ctx.EmitShrRegImm8(r32, uint8(d44.Imm.Int()))
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
			ctx.BindReg(r32, &d45)
		} else {
			{
				shiftSrc := d41.Reg
				r33 := ctx.AllocRegExcept(d41.Reg)
				ctx.EmitMovRegReg(r33, d41.Reg)
				shiftSrc = r33
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
		ctx.EnsureDesc(&d45)
		ctx.EnsureDesc(&d45)
		ctx.EnsureDesc(&d45)
		var d46 scm.JITValueDesc
		if d45.Loc == scm.LocImm {
			d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d45.Imm.Int()))))}
		} else {
			r34 := ctx.AllocReg()
			ctx.EmitMovRegReg(r34, d45.Reg)
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
			ctx.BindReg(r34, &d46)
		}
		ctx.FreeDesc(&d45)
		var d47 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
			r35 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r35, fieldAddr)
			d47 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r35}
			ctx.BindReg(r35, &d47)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
			r36 := ctx.AllocReg()
			ctx.EmitMovRegMem(r36, thisptr.Reg, off)
			d47 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r36}
			ctx.BindReg(r36, &d47)
		}
		ctx.EnsureDesc(&d46)
		ctx.EnsureDesc(&d47)
		ctx.EnsureDescsTogether(&d46, &d47)
		var d48 scm.JITValueDesc
		if d46.Loc == scm.LocImm && d47.Loc == scm.LocImm {
			d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d46.Imm.Int() + d47.Imm.Int())}
		} else if d47.Loc == scm.LocImm && d47.Imm.Int() == 0 {
			r37 := ctx.AllocRegExcept(d46.Reg)
			ctx.EmitMovRegReg(r37, d46.Reg)
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
			ctx.BindReg(r37, &d48)
		} else if d46.Loc == scm.LocImm && d46.Imm.Int() == 0 {
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d47.Reg}
			ctx.BindReg(d47.Reg, &d48)
		} else if d46.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d47.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d46.Imm.Int()))
			ctx.EmitAddInt64(scratch, d47.Reg)
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d48)
		} else if d47.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d46.Reg)
			ctx.EmitMovRegReg(scratch, d46.Reg)
			if d47.Imm.Int() >= -2147483648 && d47.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d47.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d47.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d48)
		} else {
			r38 := ctx.AllocRegExcept(d46.Reg, d47.Reg)
			ctx.EmitMovRegReg(r38, d46.Reg)
			ctx.EmitAddInt64(r38, d47.Reg)
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
			ctx.BindReg(r38, &d48)
		}
		if d48.Loc == scm.LocReg && d46.Loc == scm.LocReg && d48.Reg == d46.Reg {
			ctx.TransferReg(d46.Reg)
			d46.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d46)
		ctx.EnsureDesc(&d48)
		ctx.EnsureDesc(&d48)
		var d49 scm.JITValueDesc
		if d48.Loc == scm.LocImm {
			d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d48.Imm.Int()))))}
		} else {
			r39 := ctx.AllocReg()
			ctx.EmitMovRegReg(r39, d48.Reg)
			ctx.EmitShlRegImm8(r39, 32)
			ctx.EmitShrRegImm8(r39, 32)
			d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
			ctx.BindReg(r39, &d49)
		}
		ctx.FreeDesc(&d48)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d49)
		ctx.EnsureDescsTogether(&idxInt, &d49)
		var d50 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm && d49.Loc == scm.LocImm {
			d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d49.Imm.Int()))}
		} else if d49.Loc == scm.LocImm {
			r40 := ctx.AllocRegExcept(idxInt.Reg)
			if d49.Imm.Int() >= -2147483648 && d49.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(idxInt.Reg, int32(d49.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d49.Imm.Int()))
				ctx.EmitCmpInt64(idxInt.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r40, scm.CondUnsignedBelow)
			d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r40}
			ctx.BindReg(r40, &d50)
		} else if idxInt.Loc == scm.LocImm {
			r41 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d49.Reg)
			ctx.EmitSetcc(r41, scm.CondUnsignedBelow)
			d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r41}
			ctx.BindReg(r41, &d50)
		} else {
			r42 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitCmpInt64(idxInt.Reg, d49.Reg)
			ctx.EmitSetcc(r42, scm.CondUnsignedBelow)
			d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r42}
			ctx.BindReg(r42, &d50)
		}
		ctx.FreeDesc(&d49)
		d51 = d50
		ctx.EnsureDesc(&d51)
		if d51.Loc != scm.LocImm && d51.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d51.Loc == scm.LocImm {
			if d51.Imm.Bool() {
				if ps.General {
				}
				ps52 := scm.PhiState{General: ps.General}
				ps52.OverlayValues = make([]scm.JITValueDesc, 52)
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
				ps52.OverlayValues[16] = d16
				ps52.OverlayValues[17] = d17
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
				ps52.OverlayValues[51] = d51
				return bbs[3].RenderPS(ps52)
			}
			if ps.General {
			}
			ps53 := scm.PhiState{General: ps.General}
			ps53.OverlayValues = make([]scm.JITValueDesc, 52)
			ps53.OverlayValues[3] = d3
			ps53.OverlayValues[4] = d4
			ps53.OverlayValues[5] = d5
			ps53.OverlayValues[6] = d6
			ps53.OverlayValues[7] = d7
			ps53.OverlayValues[8] = d8
			ps53.OverlayValues[9] = d9
			ps53.OverlayValues[10] = d10
			ps53.OverlayValues[11] = d11
			ps53.OverlayValues[12] = d12
			ps53.OverlayValues[13] = d13
			ps53.OverlayValues[14] = d14
			ps53.OverlayValues[15] = d15
			ps53.OverlayValues[16] = d16
			ps53.OverlayValues[17] = d17
			ps53.OverlayValues[19] = d19
			ps53.OverlayValues[20] = d20
			ps53.OverlayValues[21] = d21
			ps53.OverlayValues[22] = d22
			ps53.OverlayValues[23] = d23
			ps53.OverlayValues[24] = d24
			ps53.OverlayValues[25] = d25
			ps53.OverlayValues[26] = d26
			ps53.OverlayValues[27] = d27
			ps53.OverlayValues[28] = d28
			ps53.OverlayValues[29] = d29
			ps53.OverlayValues[30] = d30
			ps53.OverlayValues[31] = d31
			ps53.OverlayValues[32] = d32
			ps53.OverlayValues[33] = d33
			ps53.OverlayValues[34] = d34
			ps53.OverlayValues[35] = d35
			ps53.OverlayValues[36] = d36
			ps53.OverlayValues[37] = d37
			ps53.OverlayValues[38] = d38
			ps53.OverlayValues[39] = d39
			ps53.OverlayValues[40] = d40
			ps53.OverlayValues[41] = d41
			ps53.OverlayValues[42] = d42
			ps53.OverlayValues[43] = d43
			ps53.OverlayValues[44] = d44
			ps53.OverlayValues[45] = d45
			ps53.OverlayValues[46] = d46
			ps53.OverlayValues[47] = d47
			ps53.OverlayValues[48] = d48
			ps53.OverlayValues[49] = d49
			ps53.OverlayValues[50] = d50
			ps53.OverlayValues[51] = d51
			return bbs[5].RenderPS(ps53)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d54 := ps.PhiValues[0]
				ctx.EnsureDesc(&d54)
				ctx.EmitStoreToStack(d54, int32(bbs[1].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d55 := ps.PhiValues[1]
				ctx.EnsureDesc(&d55)
				ctx.EmitStoreToStack(d55, int32(bbs[1].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d56 := ps.PhiValues[2]
				ctx.EnsureDesc(&d56)
				ctx.EmitStoreToStack(d56, int32(bbs[1].PhiBase)+int32(32))
			}
			ps.General = true
			return bbs[1].RenderPS(ps)
		}
		lbl16 := ctx.ReserveLabel()
		lbl17 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d51.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl16)
		ctx.EmitJmp(lbl17)
		ctx.MarkLabel(lbl16)
		ctx.EmitJmp(lbl4)
		ctx.MarkLabel(lbl17)
		ctx.EmitJmp(lbl6)
		ps57 := scm.PhiState{General: true}
		ps57.OverlayValues = make([]scm.JITValueDesc, 57)
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
		ps57.OverlayValues[16] = d16
		ps57.OverlayValues[17] = d17
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
		ps57.OverlayValues[51] = d51
		ps57.OverlayValues[54] = d54
		ps57.OverlayValues[55] = d55
		ps57.OverlayValues[56] = d56
		ps58 := scm.PhiState{General: true}
		ps58.OverlayValues = make([]scm.JITValueDesc, 57)
		ps58.OverlayValues[3] = d3
		ps58.OverlayValues[4] = d4
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
		ps58.OverlayValues[19] = d19
		ps58.OverlayValues[20] = d20
		ps58.OverlayValues[21] = d21
		ps58.OverlayValues[22] = d22
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
		ps58.OverlayValues[54] = d54
		ps58.OverlayValues[55] = d55
		ps58.OverlayValues[56] = d56
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
		snap72 := d16
		snap73 := d17
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
		snap105 := d50
		snap106 := d51
		snap107 := d54
		snap108 := d55
		snap109 := d56
		alloc110 := ctx.SnapshotAllocState()
		if !bbs[5].Rendered {
			bbs[5].RenderPS(ps58)
		}
		ctx.RestoreAllocState(alloc110)
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
		d16 = snap72
		d17 = snap73
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
		d50 = snap105
		d51 = snap106
		d54 = snap107
		d55 = snap108
		d56 = snap109
		if !bbs[3].Rendered {
			return bbs[3].RenderPS(ps57)
		}
		return result
		ctx.FreeDesc(&d50)
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
		if phiHomeOK2 {
			d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0}
			ctx.BindReg(r0, &d3)
		} else {
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
			d16 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
			d17 = ps.OverlayValues[17]
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
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
			d51 = ps.OverlayValues[51]
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
		if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
			d111 = ps.OverlayValues[111]
		}
		if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
			d6 = ps.PhiValues[0]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d6)
		ctx.EnsureDesc(&d6)
		ctx.EnsureDesc(&d6)
		var d112 scm.JITValueDesc
		if d6.Loc == scm.LocImm {
			d112 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(d6.Imm.Int()))))}
		} else {
			r43 := ctx.AllocReg()
			ctx.EmitMovRegReg(r43, d6.Reg)
			ctx.EmitShlRegImm8(r43, 32)
			ctx.EmitShrRegImm8(r43, 32)
			d112 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
			ctx.BindReg(r43, &d112)
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
		ctx.EnsureDesc(&d6)
		d113 = d6
		_ = d113
		ctx.StabilizeDescForControlFlow(&d113)
		ctx.StabilizeDescForControlFlow(&d6)
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
			r44 := ctx.AllocReg()
			ctx.EmitMovRegReg(r44, d113.Reg)
			ctx.EmitShlRegImm8(r44, 32)
			ctx.EmitShrRegImm8(r44, 32)
			d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
			ctx.BindReg(r44, &d114)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d115 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
			r45 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r45, fieldAddr)
			d115 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r45}
			ctx.BindReg(r45, &d115)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
			r46 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r46, thisptr.Reg, off)
			d115 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r46}
			ctx.BindReg(r46, &d115)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d115)
		ctx.EnsureDesc(&d115)
		var d116 scm.JITValueDesc
		if d115.Loc == scm.LocImm {
			d116 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d115.Imm.Int()))))}
		} else {
			r47 := ctx.AllocReg()
			ctx.EmitMovRegReg(r47, d115.Reg)
			ctx.EmitShlRegImm8(r47, 56)
			ctx.EmitShrRegImm8(r47, 56)
			d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
			ctx.BindReg(r47, &d116)
		}
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
			r48 := ctx.AllocRegExcept(d114.Reg, d116.Reg)
			ctx.EmitMovRegReg(r48, d114.Reg)
			ctx.EmitImulInt64(r48, d116.Reg)
			d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
			ctx.BindReg(r48, &d117)
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
			r49 := ctx.AllocRegExcept(d117.Reg)
			ctx.EmitMovRegReg(r49, d117.Reg)
			ctx.EmitShrRegImm8(r49, 6)
			d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
			ctx.BindReg(r49, &d118)
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
			r50 := ctx.AllocRegExcept(d117.Reg)
			ctx.EmitMovRegReg(r50, d117.Reg)
			ctx.EmitAndRegImm32(r50, 63)
			d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
			ctx.BindReg(r50, &d119)
		}
		if d119.Loc == scm.LocReg && d117.Loc == scm.LocReg && d119.Reg == d117.Reg {
			ctx.TransferReg(d117.Reg)
			d117.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d117)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d120 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 0
			r51 := ctx.AllocReg()
			r52 := ctx.AllocRegExcept(r51)
			r53 := ctx.AllocRegExcept(r51, r52)
			ctx.EmitMovRegMem64(r51, fieldAddr)
			ctx.EmitMovRegMem64(r52, fieldAddr+8)
			ctx.EmitMovRegMem64(r53, fieldAddr+16)
			d120 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r51, Reg2: r52, Reg3: r53}
			ctx.BindReg(r51, &d120)
			ctx.BindReg(r52, &d120)
			ctx.BindReg(r53, &d120)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 0)
			r54 := ctx.AllocReg()
			r55 := ctx.AllocRegExcept(r54)
			r56 := ctx.AllocRegExcept(r54, r55)
			ctx.EmitMovRegMem(r54, thisptr.Reg, off)
			ctx.EmitMovRegMem(r55, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r56, thisptr.Reg, off+16)
			d120 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r54, Reg2: r55, Reg3: r56}
			ctx.BindReg(r54, &d120)
			ctx.BindReg(r55, &d120)
			ctx.BindReg(r56, &d120)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d118)
		ctx.ReclaimUntrackedRegs()
		d122 = ctx.EmitSliceElementAddress(&d120, &d118, 8)
		ctx.EnsureDesc(&d122)
		ctx.EmitMovRegMem(d122.Reg, d122.Reg, 0)
		d121 = d122
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d121)
		ctx.EnsureDesc(&d119)
		var d123 scm.JITValueDesc
		if d121.Loc == scm.LocImm && d119.Loc == scm.LocImm {
			d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d121.Imm.Int()) << uint64(d119.Imm.Int())))}
		} else if d119.Loc == scm.LocImm {
			r57 := ctx.AllocRegExcept(d121.Reg)
			ctx.EmitMovRegReg(r57, d121.Reg)
			ctx.EmitShlRegImm8(r57, uint8(d119.Imm.Int()))
			d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r57}
			ctx.BindReg(r57, &d123)
		} else {
			{
				shiftSrc := d121.Reg
				r58 := ctx.AllocRegExcept(d121.Reg)
				ctx.EmitMovRegReg(r58, d121.Reg)
				shiftSrc = r58
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
		ctx.FreeDesc(&d124)
		ctx.ReclaimUntrackedRegs()
		d127 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d119)
		ctx.EnsureDescsTogether(&d127, &d119)
		var d128 scm.JITValueDesc
		if d127.Loc == scm.LocImm && d119.Loc == scm.LocImm {
			d128 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d127.Imm.Int() - d119.Imm.Int())}
		} else if d119.Loc == scm.LocImm && d119.Imm.Int() == 0 {
			r59 := ctx.AllocRegExcept(d127.Reg)
			ctx.EmitMovRegReg(r59, d127.Reg)
			d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r59}
			ctx.BindReg(r59, &d128)
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
			r60 := ctx.AllocRegExcept(d127.Reg, d119.Reg)
			ctx.EmitMovRegReg(r60, d127.Reg)
			ctx.EmitSubInt64(r60, d119.Reg)
			d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r60}
			ctx.BindReg(r60, &d128)
		}
		if d128.Loc == scm.LocReg && d127.Loc == scm.LocReg && d128.Reg == d127.Reg {
			ctx.TransferReg(d127.Reg)
			d127.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d119)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d125)
		ctx.EnsureDesc(&d128)
		var d129 scm.JITValueDesc
		if d125.Loc == scm.LocImm && d128.Loc == scm.LocImm {
			d129 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d125.Imm.Int()) >> uint64(d128.Imm.Int())))}
		} else if d128.Loc == scm.LocImm {
			r61 := ctx.AllocRegExcept(d125.Reg)
			ctx.EmitMovRegReg(r61, d125.Reg)
			ctx.EmitShrRegImm8(r61, uint8(d128.Imm.Int()))
			d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r61}
			ctx.BindReg(r61, &d129)
		} else {
			{
				shiftSrc := d125.Reg
				r62 := ctx.AllocRegExcept(d125.Reg)
				ctx.EmitMovRegReg(r62, d125.Reg)
				shiftSrc = r62
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
			r63 := ctx.AllocRegExcept(d123.Reg)
			ctx.EmitMovRegReg(r63, d123.Reg)
			d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r63}
			ctx.BindReg(r63, &d130)
		} else if d123.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d129.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d123.Imm.Int()))
			ctx.EmitOrInt64(scratch, d129.Reg)
			d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d130)
		} else if d129.Loc == scm.LocImm {
			r64 := ctx.AllocRegExcept(d123.Reg)
			ctx.EmitMovRegReg(r64, d123.Reg)
			if d129.Imm.Int() >= -2147483648 && d129.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r64, int32(d129.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d129.Imm.Int()))
				ctx.EmitOrInt64(r64, scm.RegR11)
			}
			d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r64}
			ctx.BindReg(r64, &d130)
		} else {
			r65 := ctx.AllocRegExcept(d123.Reg, d129.Reg)
			ctx.EmitMovRegReg(r65, d123.Reg)
			ctx.EmitOrInt64(r65, d129.Reg)
			d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
			ctx.BindReg(r65, &d130)
		}
		if d130.Loc == scm.LocReg && d123.Loc == scm.LocReg && d130.Reg == d123.Reg {
			ctx.TransferReg(d123.Reg)
			d123.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d123)
		ctx.FreeDesc(&d129)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d115)
		ctx.EnsureDesc(&d115)
		var d131 scm.JITValueDesc
		if d115.Loc == scm.LocImm {
			d131 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d115.Imm.Int()))))}
		} else {
			r66 := ctx.AllocReg()
			ctx.EmitMovRegReg(r66, d115.Reg)
			ctx.EmitShlRegImm8(r66, 56)
			ctx.EmitShrRegImm8(r66, 56)
			d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
			ctx.BindReg(r66, &d131)
		}
		ctx.ReclaimUntrackedRegs()
		d132 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d131)
		ctx.EnsureDescsTogether(&d132, &d131)
		var d133 scm.JITValueDesc
		if d132.Loc == scm.LocImm && d131.Loc == scm.LocImm {
			d133 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d132.Imm.Int() - d131.Imm.Int())}
		} else if d131.Loc == scm.LocImm && d131.Imm.Int() == 0 {
			r67 := ctx.AllocRegExcept(d132.Reg)
			ctx.EmitMovRegReg(r67, d132.Reg)
			d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r67}
			ctx.BindReg(r67, &d133)
		} else if d132.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d131.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d132.Imm.Int()))
			ctx.EmitSubInt64(scratch, d131.Reg)
			d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d133)
		} else if d131.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d132.Reg)
			ctx.EmitMovRegReg(scratch, d132.Reg)
			if d131.Imm.Int() >= -2147483648 && d131.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d131.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d131.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d133)
		} else {
			r68 := ctx.AllocRegExcept(d132.Reg, d131.Reg)
			ctx.EmitMovRegReg(r68, d132.Reg)
			ctx.EmitSubInt64(r68, d131.Reg)
			d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r68}
			ctx.BindReg(r68, &d133)
		}
		if d133.Loc == scm.LocReg && d132.Loc == scm.LocReg && d133.Reg == d132.Reg {
			ctx.TransferReg(d132.Reg)
			d132.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d131)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d130)
		ctx.EnsureDesc(&d133)
		var d134 scm.JITValueDesc
		if d130.Loc == scm.LocImm && d133.Loc == scm.LocImm {
			d134 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d130.Imm.Int()) >> uint64(d133.Imm.Int())))}
		} else if d133.Loc == scm.LocImm {
			r69 := ctx.AllocRegExcept(d130.Reg)
			ctx.EmitMovRegReg(r69, d130.Reg)
			ctx.EmitShrRegImm8(r69, uint8(d133.Imm.Int()))
			d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r69}
			ctx.BindReg(r69, &d134)
		} else {
			{
				shiftSrc := d130.Reg
				r70 := ctx.AllocRegExcept(d130.Reg)
				ctx.EmitMovRegReg(r70, d130.Reg)
				shiftSrc = r70
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d133.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d133.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d133.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d134)
			}
		}
		if d134.Loc == scm.LocReg && d130.Loc == scm.LocReg && d134.Reg == d130.Reg {
			ctx.TransferReg(d130.Reg)
			d130.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d130)
		ctx.FreeDesc(&d133)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d134)
		ctx.EnsureDesc(&d134)
		ctx.EnsureDesc(&d134)
		var d135 scm.JITValueDesc
		if d134.Loc == scm.LocImm {
			d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d134.Imm.Int()))))}
		} else {
			r71 := ctx.AllocReg()
			ctx.EmitMovRegReg(r71, d134.Reg)
			d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r71}
			ctx.BindReg(r71, &d135)
		}
		ctx.FreeDesc(&d134)
		var d136 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 32
			r72 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r72, fieldAddr)
			d136 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r72}
			ctx.BindReg(r72, &d136)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 32)
			r73 := ctx.AllocReg()
			ctx.EmitMovRegMem(r73, thisptr.Reg, off)
			d136 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r73}
			ctx.BindReg(r73, &d136)
		}
		ctx.EnsureDesc(&d135)
		ctx.EnsureDesc(&d136)
		ctx.EnsureDescsTogether(&d135, &d136)
		var d137 scm.JITValueDesc
		if d135.Loc == scm.LocImm && d136.Loc == scm.LocImm {
			d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d135.Imm.Int() + d136.Imm.Int())}
		} else if d136.Loc == scm.LocImm && d136.Imm.Int() == 0 {
			r74 := ctx.AllocRegExcept(d135.Reg)
			ctx.EmitMovRegReg(r74, d135.Reg)
			d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r74}
			ctx.BindReg(r74, &d137)
		} else if d135.Loc == scm.LocImm && d135.Imm.Int() == 0 {
			d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d136.Reg}
			ctx.BindReg(d136.Reg, &d137)
		} else if d135.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d136.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d135.Imm.Int()))
			ctx.EmitAddInt64(scratch, d136.Reg)
			d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d137)
		} else if d136.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d135.Reg)
			ctx.EmitMovRegReg(scratch, d135.Reg)
			if d136.Imm.Int() >= -2147483648 && d136.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d136.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d136.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d137)
		} else {
			r75 := ctx.AllocRegExcept(d135.Reg, d136.Reg)
			ctx.EmitMovRegReg(r75, d135.Reg)
			ctx.EmitAddInt64(r75, d136.Reg)
			d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
			ctx.BindReg(r75, &d137)
		}
		if d137.Loc == scm.LocReg && d135.Loc == scm.LocReg && d137.Reg == d135.Reg {
			ctx.TransferReg(d135.Reg)
			d135.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d137)
		ctx.FreeDesc(&d135)
		var d138 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 56
			r76 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r76, fieldAddr)
			d138 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r76}
			ctx.BindReg(r76, &d138)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 56)
			r77 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r77, thisptr.Reg, off)
			d138 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r77}
			ctx.BindReg(r77, &d138)
		}
		d139 = d138
		ctx.EnsureDesc(&d139)
		if d139.Loc != scm.LocImm && d139.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d139.Loc == scm.LocImm {
			if d139.Imm.Bool() {
				if ps.General {
				}
				ps140 := scm.PhiState{General: ps.General}
				ps140.OverlayValues = make([]scm.JITValueDesc, 140)
				ps140.OverlayValues[3] = d3
				ps140.OverlayValues[4] = d4
				ps140.OverlayValues[5] = d5
				ps140.OverlayValues[6] = d6
				ps140.OverlayValues[7] = d7
				ps140.OverlayValues[8] = d8
				ps140.OverlayValues[9] = d9
				ps140.OverlayValues[10] = d10
				ps140.OverlayValues[11] = d11
				ps140.OverlayValues[12] = d12
				ps140.OverlayValues[13] = d13
				ps140.OverlayValues[14] = d14
				ps140.OverlayValues[15] = d15
				ps140.OverlayValues[16] = d16
				ps140.OverlayValues[17] = d17
				ps140.OverlayValues[19] = d19
				ps140.OverlayValues[20] = d20
				ps140.OverlayValues[21] = d21
				ps140.OverlayValues[22] = d22
				ps140.OverlayValues[23] = d23
				ps140.OverlayValues[24] = d24
				ps140.OverlayValues[25] = d25
				ps140.OverlayValues[26] = d26
				ps140.OverlayValues[27] = d27
				ps140.OverlayValues[28] = d28
				ps140.OverlayValues[29] = d29
				ps140.OverlayValues[30] = d30
				ps140.OverlayValues[31] = d31
				ps140.OverlayValues[32] = d32
				ps140.OverlayValues[33] = d33
				ps140.OverlayValues[34] = d34
				ps140.OverlayValues[35] = d35
				ps140.OverlayValues[36] = d36
				ps140.OverlayValues[37] = d37
				ps140.OverlayValues[38] = d38
				ps140.OverlayValues[39] = d39
				ps140.OverlayValues[40] = d40
				ps140.OverlayValues[41] = d41
				ps140.OverlayValues[42] = d42
				ps140.OverlayValues[43] = d43
				ps140.OverlayValues[44] = d44
				ps140.OverlayValues[45] = d45
				ps140.OverlayValues[46] = d46
				ps140.OverlayValues[47] = d47
				ps140.OverlayValues[48] = d48
				ps140.OverlayValues[49] = d49
				ps140.OverlayValues[50] = d50
				ps140.OverlayValues[51] = d51
				ps140.OverlayValues[54] = d54
				ps140.OverlayValues[55] = d55
				ps140.OverlayValues[56] = d56
				ps140.OverlayValues[111] = d111
				ps140.OverlayValues[112] = d112
				ps140.OverlayValues[113] = d113
				ps140.OverlayValues[114] = d114
				ps140.OverlayValues[115] = d115
				ps140.OverlayValues[116] = d116
				ps140.OverlayValues[117] = d117
				ps140.OverlayValues[118] = d118
				ps140.OverlayValues[119] = d119
				ps140.OverlayValues[120] = d120
				ps140.OverlayValues[121] = d121
				ps140.OverlayValues[122] = d122
				ps140.OverlayValues[123] = d123
				ps140.OverlayValues[124] = d124
				ps140.OverlayValues[125] = d125
				ps140.OverlayValues[126] = d126
				ps140.OverlayValues[127] = d127
				ps140.OverlayValues[128] = d128
				ps140.OverlayValues[129] = d129
				ps140.OverlayValues[130] = d130
				ps140.OverlayValues[131] = d131
				ps140.OverlayValues[132] = d132
				ps140.OverlayValues[133] = d133
				ps140.OverlayValues[134] = d134
				ps140.OverlayValues[135] = d135
				ps140.OverlayValues[136] = d136
				ps140.OverlayValues[137] = d137
				ps140.OverlayValues[138] = d138
				ps140.OverlayValues[139] = d139
				return bbs[13].RenderPS(ps140)
			}
			if ps.General {
			}
			ps141 := scm.PhiState{General: ps.General}
			ps141.OverlayValues = make([]scm.JITValueDesc, 140)
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
			ps141.OverlayValues[16] = d16
			ps141.OverlayValues[17] = d17
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
			ps141.OverlayValues[51] = d51
			ps141.OverlayValues[54] = d54
			ps141.OverlayValues[55] = d55
			ps141.OverlayValues[56] = d56
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
			return bbs[12].RenderPS(ps141)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d142 := ps.PhiValues[0]
				ctx.EnsureDesc(&d142)
				ctx.EmitStoreToStack(d142, int32(bbs[2].PhiBase)+int32(0))
			}
			ps.General = true
			return bbs[2].RenderPS(ps)
		}
		lbl19 := ctx.ReserveLabel()
		lbl20 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d139.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl19)
		ctx.EmitJmp(lbl20)
		ctx.MarkLabel(lbl19)
		ctx.EmitJmp(lbl14)
		ctx.MarkLabel(lbl20)
		ctx.EmitJmp(lbl13)
		ps143 := scm.PhiState{General: true}
		ps143.OverlayValues = make([]scm.JITValueDesc, 143)
		ps143.OverlayValues[3] = d3
		ps143.OverlayValues[4] = d4
		ps143.OverlayValues[5] = d5
		ps143.OverlayValues[6] = d6
		ps143.OverlayValues[7] = d7
		ps143.OverlayValues[8] = d8
		ps143.OverlayValues[9] = d9
		ps143.OverlayValues[10] = d10
		ps143.OverlayValues[11] = d11
		ps143.OverlayValues[12] = d12
		ps143.OverlayValues[13] = d13
		ps143.OverlayValues[14] = d14
		ps143.OverlayValues[15] = d15
		ps143.OverlayValues[16] = d16
		ps143.OverlayValues[17] = d17
		ps143.OverlayValues[19] = d19
		ps143.OverlayValues[20] = d20
		ps143.OverlayValues[21] = d21
		ps143.OverlayValues[22] = d22
		ps143.OverlayValues[23] = d23
		ps143.OverlayValues[24] = d24
		ps143.OverlayValues[25] = d25
		ps143.OverlayValues[26] = d26
		ps143.OverlayValues[27] = d27
		ps143.OverlayValues[28] = d28
		ps143.OverlayValues[29] = d29
		ps143.OverlayValues[30] = d30
		ps143.OverlayValues[31] = d31
		ps143.OverlayValues[32] = d32
		ps143.OverlayValues[33] = d33
		ps143.OverlayValues[34] = d34
		ps143.OverlayValues[35] = d35
		ps143.OverlayValues[36] = d36
		ps143.OverlayValues[37] = d37
		ps143.OverlayValues[38] = d38
		ps143.OverlayValues[39] = d39
		ps143.OverlayValues[40] = d40
		ps143.OverlayValues[41] = d41
		ps143.OverlayValues[42] = d42
		ps143.OverlayValues[43] = d43
		ps143.OverlayValues[44] = d44
		ps143.OverlayValues[45] = d45
		ps143.OverlayValues[46] = d46
		ps143.OverlayValues[47] = d47
		ps143.OverlayValues[48] = d48
		ps143.OverlayValues[49] = d49
		ps143.OverlayValues[50] = d50
		ps143.OverlayValues[51] = d51
		ps143.OverlayValues[54] = d54
		ps143.OverlayValues[55] = d55
		ps143.OverlayValues[56] = d56
		ps143.OverlayValues[111] = d111
		ps143.OverlayValues[112] = d112
		ps143.OverlayValues[113] = d113
		ps143.OverlayValues[114] = d114
		ps143.OverlayValues[115] = d115
		ps143.OverlayValues[116] = d116
		ps143.OverlayValues[117] = d117
		ps143.OverlayValues[118] = d118
		ps143.OverlayValues[119] = d119
		ps143.OverlayValues[120] = d120
		ps143.OverlayValues[121] = d121
		ps143.OverlayValues[122] = d122
		ps143.OverlayValues[123] = d123
		ps143.OverlayValues[124] = d124
		ps143.OverlayValues[125] = d125
		ps143.OverlayValues[126] = d126
		ps143.OverlayValues[127] = d127
		ps143.OverlayValues[128] = d128
		ps143.OverlayValues[129] = d129
		ps143.OverlayValues[130] = d130
		ps143.OverlayValues[131] = d131
		ps143.OverlayValues[132] = d132
		ps143.OverlayValues[133] = d133
		ps143.OverlayValues[134] = d134
		ps143.OverlayValues[135] = d135
		ps143.OverlayValues[136] = d136
		ps143.OverlayValues[137] = d137
		ps143.OverlayValues[138] = d138
		ps143.OverlayValues[139] = d139
		ps143.OverlayValues[142] = d142
		ps144 := scm.PhiState{General: true}
		ps144.OverlayValues = make([]scm.JITValueDesc, 143)
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
		ps144.OverlayValues[16] = d16
		ps144.OverlayValues[17] = d17
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
		ps144.OverlayValues[51] = d51
		ps144.OverlayValues[54] = d54
		ps144.OverlayValues[55] = d55
		ps144.OverlayValues[56] = d56
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
		ps144.OverlayValues[142] = d142
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
		snap158 := d16
		snap159 := d17
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
		snap191 := d50
		snap192 := d51
		snap193 := d54
		snap194 := d55
		snap195 := d56
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
		snap223 := d138
		snap224 := d139
		snap225 := d142
		alloc226 := ctx.SnapshotAllocState()
		if !bbs[12].Rendered {
			bbs[12].RenderPS(ps144)
		}
		ctx.RestoreAllocState(alloc226)
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
		d16 = snap158
		d17 = snap159
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
		d50 = snap191
		d51 = snap192
		d54 = snap193
		d55 = snap194
		d56 = snap195
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
		d138 = snap223
		d139 = snap224
		d142 = snap225
		if !bbs[13].Rendered {
			return bbs[13].RenderPS(ps143)
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
			d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0}
			ctx.BindReg(r0, &d3)
		} else {
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
			d16 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
			d17 = ps.OverlayValues[17]
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
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
			d51 = ps.OverlayValues[51]
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
		if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != scm.LocNone {
			d142 = ps.OverlayValues[142]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d3)
		ctx.EnsureDesc(&d3)
		var d227 scm.JITValueDesc
		if d3.Loc == scm.LocImm {
			d227 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d3.Reg)
			ctx.EmitMovRegReg(scratch, d3.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d227 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d227)
		}
		if d227.Loc == scm.LocImm {
			d227 = scm.JITValueDesc{Loc: scm.LocImm, Type: d227.Type, Imm: scm.NewInt(int64(uint64(d227.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d227.Reg, 32)
			ctx.EmitShrRegImm8(d227.Reg, 32)
		}
		if d227.Loc == scm.LocReg && d3.Loc == scm.LocReg && d227.Reg == d3.Reg {
			ctx.TransferReg(d3.Reg)
			d3.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d227)
		ctx.EmitStoreToStack(d227, int32(bbs[4].PhiBase)+int32(32))
		ctx.StabilizeDescForControlFlow(&d227)
		ctx.EnsureDesc(&d3)
		ctx.EnsureDesc(&d3)
		var d228 scm.JITValueDesc
		if d3.Loc == scm.LocImm {
			d228 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d3.Reg)
			ctx.EmitMovRegReg(scratch, d3.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d228 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d228)
		}
		if d228.Loc == scm.LocImm {
			d228 = scm.JITValueDesc{Loc: scm.LocImm, Type: d228.Type, Imm: scm.NewInt(int64(uint64(d228.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d228.Reg, 32)
			ctx.EmitShrRegImm8(d228.Reg, 32)
		}
		if d228.Loc == scm.LocReg && d3.Loc == scm.LocReg && d228.Reg == d3.Reg {
			ctx.TransferReg(d3.Reg)
			d3.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d228)
		ctx.EmitStoreToStack(d228, int32(bbs[4].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d228)
		if ps.General {
			ctx.SyncDesc(&d4)
			if d4.Loc == scm.LocReg {
				ctx.ProtectReg(d4.Reg)
			} else if d4.Loc == scm.LocRegPair {
				ctx.ProtectReg(d4.Reg)
				ctx.ProtectReg(d4.Reg2)
			}
			d229 = d4
			if d229.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d229)
			d230 = d229
			if d230.Loc == scm.LocImm {
				d230 = scm.JITValueDesc{Loc: scm.LocImm, Type: d230.Type, Imm: scm.NewInt(int64(uint64(d230.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d230.Reg, 32)
				ctx.EmitShrRegImm8(d230.Reg, 32)
			}
			ctx.EmitStoreToStack(d230, int32(bbs[4].PhiBase)+int32(16))
			if d4.Loc == scm.LocReg {
				ctx.UnprotectReg(d4.Reg)
			} else if d4.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d4.Reg)
				ctx.UnprotectReg(d4.Reg2)
			}
		}
		ps231 := scm.PhiState{General: ps.General}
		ps231.OverlayValues = make([]scm.JITValueDesc, 231)
		ps231.OverlayValues[3] = d3
		ps231.OverlayValues[4] = d4
		ps231.OverlayValues[5] = d5
		ps231.OverlayValues[6] = d6
		ps231.OverlayValues[7] = d7
		ps231.OverlayValues[8] = d8
		ps231.OverlayValues[9] = d9
		ps231.OverlayValues[10] = d10
		ps231.OverlayValues[11] = d11
		ps231.OverlayValues[12] = d12
		ps231.OverlayValues[13] = d13
		ps231.OverlayValues[14] = d14
		ps231.OverlayValues[15] = d15
		ps231.OverlayValues[16] = d16
		ps231.OverlayValues[17] = d17
		ps231.OverlayValues[19] = d19
		ps231.OverlayValues[20] = d20
		ps231.OverlayValues[21] = d21
		ps231.OverlayValues[22] = d22
		ps231.OverlayValues[23] = d23
		ps231.OverlayValues[24] = d24
		ps231.OverlayValues[25] = d25
		ps231.OverlayValues[26] = d26
		ps231.OverlayValues[27] = d27
		ps231.OverlayValues[28] = d28
		ps231.OverlayValues[29] = d29
		ps231.OverlayValues[30] = d30
		ps231.OverlayValues[31] = d31
		ps231.OverlayValues[32] = d32
		ps231.OverlayValues[33] = d33
		ps231.OverlayValues[34] = d34
		ps231.OverlayValues[35] = d35
		ps231.OverlayValues[36] = d36
		ps231.OverlayValues[37] = d37
		ps231.OverlayValues[38] = d38
		ps231.OverlayValues[39] = d39
		ps231.OverlayValues[40] = d40
		ps231.OverlayValues[41] = d41
		ps231.OverlayValues[42] = d42
		ps231.OverlayValues[43] = d43
		ps231.OverlayValues[44] = d44
		ps231.OverlayValues[45] = d45
		ps231.OverlayValues[46] = d46
		ps231.OverlayValues[47] = d47
		ps231.OverlayValues[48] = d48
		ps231.OverlayValues[49] = d49
		ps231.OverlayValues[50] = d50
		ps231.OverlayValues[51] = d51
		ps231.OverlayValues[54] = d54
		ps231.OverlayValues[55] = d55
		ps231.OverlayValues[56] = d56
		ps231.OverlayValues[111] = d111
		ps231.OverlayValues[112] = d112
		ps231.OverlayValues[113] = d113
		ps231.OverlayValues[114] = d114
		ps231.OverlayValues[115] = d115
		ps231.OverlayValues[116] = d116
		ps231.OverlayValues[117] = d117
		ps231.OverlayValues[118] = d118
		ps231.OverlayValues[119] = d119
		ps231.OverlayValues[120] = d120
		ps231.OverlayValues[121] = d121
		ps231.OverlayValues[122] = d122
		ps231.OverlayValues[123] = d123
		ps231.OverlayValues[124] = d124
		ps231.OverlayValues[125] = d125
		ps231.OverlayValues[126] = d126
		ps231.OverlayValues[127] = d127
		ps231.OverlayValues[128] = d128
		ps231.OverlayValues[129] = d129
		ps231.OverlayValues[130] = d130
		ps231.OverlayValues[131] = d131
		ps231.OverlayValues[132] = d132
		ps231.OverlayValues[133] = d133
		ps231.OverlayValues[134] = d134
		ps231.OverlayValues[135] = d135
		ps231.OverlayValues[136] = d136
		ps231.OverlayValues[137] = d137
		ps231.OverlayValues[138] = d138
		ps231.OverlayValues[139] = d139
		ps231.OverlayValues[142] = d142
		ps231.OverlayValues[227] = d227
		ps231.OverlayValues[228] = d228
		ps231.OverlayValues[229] = d229
		ps231.OverlayValues[230] = d230
		ps231.PhiValues = make([]scm.JITValueDesc, 3)
		d232 = d4
		ps231.PhiValues[1] = d232
		if ps231.General && bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
			return result
		}
		return bbs[4].RenderPS(ps231)
		return result
	}
	bbs[4].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d233 := ps.PhiValues[0]
				ctx.EnsureDesc(&d233)
				ctx.EmitStoreToStack(d233, int32(bbs[4].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d234 := ps.PhiValues[1]
				ctx.EnsureDesc(&d234)
				ctx.EmitStoreToStack(d234, int32(bbs[4].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d235 := ps.PhiValues[2]
				ctx.EnsureDesc(&d235)
				ctx.EmitStoreToStack(d235, int32(bbs[4].PhiBase)+int32(32))
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
			d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0}
			ctx.BindReg(r0, &d3)
		} else {
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
			d16 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
			d17 = ps.OverlayValues[17]
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
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
			d51 = ps.OverlayValues[51]
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
		if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != scm.LocNone {
			d142 = ps.OverlayValues[142]
		}
		if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
			d227 = ps.OverlayValues[227]
		}
		if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
			d228 = ps.OverlayValues[228]
		}
		if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != scm.LocNone {
			d229 = ps.OverlayValues[229]
		}
		if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
			d230 = ps.OverlayValues[230]
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
		if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
			d7 = ps.PhiValues[0]
		}
		if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
			d8 = ps.PhiValues[1]
		}
		if !ps.General && len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
			d9 = ps.PhiValues[2]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d7)
		ctx.StabilizeDescForControlFlow(&d8)
		ctx.StabilizeDescForControlFlow(&d9)
		ctx.EnsureDesc(&d8)
		ctx.EnsureDesc(&d9)
		ctx.EnsureDescsTogether(&d8, &d9)
		var d236 scm.JITValueDesc
		if d8.Loc == scm.LocImm && d9.Loc == scm.LocImm {
			d236 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d8.Imm.Int()) == uint64(d9.Imm.Int()))}
		} else if d9.Loc == scm.LocImm {
			r78 := ctx.AllocRegExcept(d8.Reg)
			if d9.Imm.Int() >= -2147483648 && d9.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d8.Reg, int32(d9.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d9.Imm.Int()))
				ctx.EmitCmpInt64(d8.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r78, scm.CondEqual)
			d236 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r78}
			ctx.BindReg(r78, &d236)
		} else if d8.Loc == scm.LocImm {
			r79 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d8.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d9.Reg)
			ctx.EmitSetcc(r79, scm.CondEqual)
			d236 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r79}
			ctx.BindReg(r79, &d236)
		} else {
			r80 := ctx.AllocRegExcept(d8.Reg)
			ctx.EmitCmpInt64(d8.Reg, d9.Reg)
			ctx.EmitSetcc(r80, scm.CondEqual)
			d236 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r80}
			ctx.BindReg(r80, &d236)
		}
		d237 = d236
		ctx.EnsureDesc(&d237)
		if d237.Loc != scm.LocImm && d237.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d237.Loc == scm.LocImm {
			if d237.Imm.Bool() {
				if ps.General {
					ctx.SyncDesc(&d8)
					if d8.Loc == scm.LocReg {
						ctx.ProtectReg(d8.Reg)
					} else if d8.Loc == scm.LocRegPair {
						ctx.ProtectReg(d8.Reg)
						ctx.ProtectReg(d8.Reg2)
					}
					d238 = d8
					if d238.Loc == scm.LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d238)
					d239 = d238
					if d239.Loc == scm.LocImm {
						d239 = scm.JITValueDesc{Loc: scm.LocImm, Type: d239.Type, Imm: scm.NewInt(int64(uint64(d239.Imm.Int()) & 0xffffffff))}
					} else {
						ctx.EmitShlRegImm8(d239.Reg, 32)
						ctx.EmitShrRegImm8(d239.Reg, 32)
					}
					ctx.EmitStoreToStack(d239, int32(bbs[2].PhiBase)+int32(0))
					if d8.Loc == scm.LocReg {
						ctx.UnprotectReg(d8.Reg)
					} else if d8.Loc == scm.LocRegPair {
						ctx.UnprotectReg(d8.Reg)
						ctx.UnprotectReg(d8.Reg2)
					}
				}
				ps240 := scm.PhiState{General: ps.General}
				ps240.OverlayValues = make([]scm.JITValueDesc, 240)
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
				ps240.OverlayValues[16] = d16
				ps240.OverlayValues[17] = d17
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
				ps240.OverlayValues[50] = d50
				ps240.OverlayValues[51] = d51
				ps240.OverlayValues[54] = d54
				ps240.OverlayValues[55] = d55
				ps240.OverlayValues[56] = d56
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
				ps240.OverlayValues[138] = d138
				ps240.OverlayValues[139] = d139
				ps240.OverlayValues[142] = d142
				ps240.OverlayValues[227] = d227
				ps240.OverlayValues[228] = d228
				ps240.OverlayValues[229] = d229
				ps240.OverlayValues[230] = d230
				ps240.OverlayValues[232] = d232
				ps240.OverlayValues[233] = d233
				ps240.OverlayValues[234] = d234
				ps240.OverlayValues[235] = d235
				ps240.OverlayValues[236] = d236
				ps240.OverlayValues[237] = d237
				ps240.OverlayValues[238] = d238
				ps240.OverlayValues[239] = d239
				ps240.PhiValues = make([]scm.JITValueDesc, 1)
				d241 = d8
				ps240.PhiValues[0] = d241
				return bbs[2].RenderPS(ps240)
			}
			if ps.General {
			}
			ps242 := scm.PhiState{General: ps.General}
			ps242.OverlayValues = make([]scm.JITValueDesc, 242)
			ps242.OverlayValues[3] = d3
			ps242.OverlayValues[4] = d4
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
			ps242.OverlayValues[19] = d19
			ps242.OverlayValues[20] = d20
			ps242.OverlayValues[21] = d21
			ps242.OverlayValues[22] = d22
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
			ps242.OverlayValues[54] = d54
			ps242.OverlayValues[55] = d55
			ps242.OverlayValues[56] = d56
			ps242.OverlayValues[111] = d111
			ps242.OverlayValues[112] = d112
			ps242.OverlayValues[113] = d113
			ps242.OverlayValues[114] = d114
			ps242.OverlayValues[115] = d115
			ps242.OverlayValues[116] = d116
			ps242.OverlayValues[117] = d117
			ps242.OverlayValues[118] = d118
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
			ps242.OverlayValues[142] = d142
			ps242.OverlayValues[227] = d227
			ps242.OverlayValues[228] = d228
			ps242.OverlayValues[229] = d229
			ps242.OverlayValues[230] = d230
			ps242.OverlayValues[232] = d232
			ps242.OverlayValues[233] = d233
			ps242.OverlayValues[234] = d234
			ps242.OverlayValues[235] = d235
			ps242.OverlayValues[236] = d236
			ps242.OverlayValues[237] = d237
			ps242.OverlayValues[238] = d238
			ps242.OverlayValues[239] = d239
			ps242.OverlayValues[241] = d241
			return bbs[6].RenderPS(ps242)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d243 := ps.PhiValues[0]
				ctx.EnsureDesc(&d243)
				ctx.EmitStoreToStack(d243, int32(bbs[4].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d244 := ps.PhiValues[1]
				ctx.EnsureDesc(&d244)
				ctx.EmitStoreToStack(d244, int32(bbs[4].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d245 := ps.PhiValues[2]
				ctx.EnsureDesc(&d245)
				ctx.EmitStoreToStack(d245, int32(bbs[4].PhiBase)+int32(32))
			}
			ps.General = true
			return bbs[4].RenderPS(ps)
		}
		lbl21 := ctx.ReserveLabel()
		lbl22 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d237.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl21)
		ctx.EmitJmp(lbl22)
		ctx.MarkLabel(lbl21)
		ctx.SyncDesc(&d8)
		if d8.Loc == scm.LocReg {
			ctx.ProtectReg(d8.Reg)
		} else if d8.Loc == scm.LocRegPair {
			ctx.ProtectReg(d8.Reg)
			ctx.ProtectReg(d8.Reg2)
		}
		d246 = d8
		if d246.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d246)
		d247 = d246
		if d247.Loc == scm.LocImm {
			d247 = scm.JITValueDesc{Loc: scm.LocImm, Type: d247.Type, Imm: scm.NewInt(int64(uint64(d247.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d247.Reg, 32)
			ctx.EmitShrRegImm8(d247.Reg, 32)
		}
		ctx.EmitStoreToStack(d247, int32(bbs[2].PhiBase)+int32(0))
		if d8.Loc == scm.LocReg {
			ctx.UnprotectReg(d8.Reg)
		} else if d8.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d8.Reg)
			ctx.UnprotectReg(d8.Reg2)
		}
		ctx.EmitJmp(lbl3)
		ctx.MarkLabel(lbl22)
		ctx.EmitJmp(lbl7)
		ps248 := scm.PhiState{General: true}
		ps248.OverlayValues = make([]scm.JITValueDesc, 248)
		ps248.OverlayValues[3] = d3
		ps248.OverlayValues[4] = d4
		ps248.OverlayValues[5] = d5
		ps248.OverlayValues[6] = d6
		ps248.OverlayValues[7] = d7
		ps248.OverlayValues[8] = d8
		ps248.OverlayValues[9] = d9
		ps248.OverlayValues[10] = d10
		ps248.OverlayValues[11] = d11
		ps248.OverlayValues[12] = d12
		ps248.OverlayValues[13] = d13
		ps248.OverlayValues[14] = d14
		ps248.OverlayValues[15] = d15
		ps248.OverlayValues[16] = d16
		ps248.OverlayValues[17] = d17
		ps248.OverlayValues[19] = d19
		ps248.OverlayValues[20] = d20
		ps248.OverlayValues[21] = d21
		ps248.OverlayValues[22] = d22
		ps248.OverlayValues[23] = d23
		ps248.OverlayValues[24] = d24
		ps248.OverlayValues[25] = d25
		ps248.OverlayValues[26] = d26
		ps248.OverlayValues[27] = d27
		ps248.OverlayValues[28] = d28
		ps248.OverlayValues[29] = d29
		ps248.OverlayValues[30] = d30
		ps248.OverlayValues[31] = d31
		ps248.OverlayValues[32] = d32
		ps248.OverlayValues[33] = d33
		ps248.OverlayValues[34] = d34
		ps248.OverlayValues[35] = d35
		ps248.OverlayValues[36] = d36
		ps248.OverlayValues[37] = d37
		ps248.OverlayValues[38] = d38
		ps248.OverlayValues[39] = d39
		ps248.OverlayValues[40] = d40
		ps248.OverlayValues[41] = d41
		ps248.OverlayValues[42] = d42
		ps248.OverlayValues[43] = d43
		ps248.OverlayValues[44] = d44
		ps248.OverlayValues[45] = d45
		ps248.OverlayValues[46] = d46
		ps248.OverlayValues[47] = d47
		ps248.OverlayValues[48] = d48
		ps248.OverlayValues[49] = d49
		ps248.OverlayValues[50] = d50
		ps248.OverlayValues[51] = d51
		ps248.OverlayValues[54] = d54
		ps248.OverlayValues[55] = d55
		ps248.OverlayValues[56] = d56
		ps248.OverlayValues[111] = d111
		ps248.OverlayValues[112] = d112
		ps248.OverlayValues[113] = d113
		ps248.OverlayValues[114] = d114
		ps248.OverlayValues[115] = d115
		ps248.OverlayValues[116] = d116
		ps248.OverlayValues[117] = d117
		ps248.OverlayValues[118] = d118
		ps248.OverlayValues[119] = d119
		ps248.OverlayValues[120] = d120
		ps248.OverlayValues[121] = d121
		ps248.OverlayValues[122] = d122
		ps248.OverlayValues[123] = d123
		ps248.OverlayValues[124] = d124
		ps248.OverlayValues[125] = d125
		ps248.OverlayValues[126] = d126
		ps248.OverlayValues[127] = d127
		ps248.OverlayValues[128] = d128
		ps248.OverlayValues[129] = d129
		ps248.OverlayValues[130] = d130
		ps248.OverlayValues[131] = d131
		ps248.OverlayValues[132] = d132
		ps248.OverlayValues[133] = d133
		ps248.OverlayValues[134] = d134
		ps248.OverlayValues[135] = d135
		ps248.OverlayValues[136] = d136
		ps248.OverlayValues[137] = d137
		ps248.OverlayValues[138] = d138
		ps248.OverlayValues[139] = d139
		ps248.OverlayValues[142] = d142
		ps248.OverlayValues[227] = d227
		ps248.OverlayValues[228] = d228
		ps248.OverlayValues[229] = d229
		ps248.OverlayValues[230] = d230
		ps248.OverlayValues[232] = d232
		ps248.OverlayValues[233] = d233
		ps248.OverlayValues[234] = d234
		ps248.OverlayValues[235] = d235
		ps248.OverlayValues[236] = d236
		ps248.OverlayValues[237] = d237
		ps248.OverlayValues[238] = d238
		ps248.OverlayValues[239] = d239
		ps248.OverlayValues[241] = d241
		ps248.OverlayValues[243] = d243
		ps248.OverlayValues[244] = d244
		ps248.OverlayValues[245] = d245
		ps248.OverlayValues[246] = d246
		ps248.OverlayValues[247] = d247
		ps248.PhiValues = make([]scm.JITValueDesc, 1)
		d250 = d8
		ps248.PhiValues[0] = d250
		ps249 := scm.PhiState{General: true}
		ps249.OverlayValues = make([]scm.JITValueDesc, 251)
		ps249.OverlayValues[3] = d3
		ps249.OverlayValues[4] = d4
		ps249.OverlayValues[5] = d5
		ps249.OverlayValues[6] = d6
		ps249.OverlayValues[7] = d7
		ps249.OverlayValues[8] = d8
		ps249.OverlayValues[9] = d9
		ps249.OverlayValues[10] = d10
		ps249.OverlayValues[11] = d11
		ps249.OverlayValues[12] = d12
		ps249.OverlayValues[13] = d13
		ps249.OverlayValues[14] = d14
		ps249.OverlayValues[15] = d15
		ps249.OverlayValues[16] = d16
		ps249.OverlayValues[17] = d17
		ps249.OverlayValues[19] = d19
		ps249.OverlayValues[20] = d20
		ps249.OverlayValues[21] = d21
		ps249.OverlayValues[22] = d22
		ps249.OverlayValues[23] = d23
		ps249.OverlayValues[24] = d24
		ps249.OverlayValues[25] = d25
		ps249.OverlayValues[26] = d26
		ps249.OverlayValues[27] = d27
		ps249.OverlayValues[28] = d28
		ps249.OverlayValues[29] = d29
		ps249.OverlayValues[30] = d30
		ps249.OverlayValues[31] = d31
		ps249.OverlayValues[32] = d32
		ps249.OverlayValues[33] = d33
		ps249.OverlayValues[34] = d34
		ps249.OverlayValues[35] = d35
		ps249.OverlayValues[36] = d36
		ps249.OverlayValues[37] = d37
		ps249.OverlayValues[38] = d38
		ps249.OverlayValues[39] = d39
		ps249.OverlayValues[40] = d40
		ps249.OverlayValues[41] = d41
		ps249.OverlayValues[42] = d42
		ps249.OverlayValues[43] = d43
		ps249.OverlayValues[44] = d44
		ps249.OverlayValues[45] = d45
		ps249.OverlayValues[46] = d46
		ps249.OverlayValues[47] = d47
		ps249.OverlayValues[48] = d48
		ps249.OverlayValues[49] = d49
		ps249.OverlayValues[50] = d50
		ps249.OverlayValues[51] = d51
		ps249.OverlayValues[54] = d54
		ps249.OverlayValues[55] = d55
		ps249.OverlayValues[56] = d56
		ps249.OverlayValues[111] = d111
		ps249.OverlayValues[112] = d112
		ps249.OverlayValues[113] = d113
		ps249.OverlayValues[114] = d114
		ps249.OverlayValues[115] = d115
		ps249.OverlayValues[116] = d116
		ps249.OverlayValues[117] = d117
		ps249.OverlayValues[118] = d118
		ps249.OverlayValues[119] = d119
		ps249.OverlayValues[120] = d120
		ps249.OverlayValues[121] = d121
		ps249.OverlayValues[122] = d122
		ps249.OverlayValues[123] = d123
		ps249.OverlayValues[124] = d124
		ps249.OverlayValues[125] = d125
		ps249.OverlayValues[126] = d126
		ps249.OverlayValues[127] = d127
		ps249.OverlayValues[128] = d128
		ps249.OverlayValues[129] = d129
		ps249.OverlayValues[130] = d130
		ps249.OverlayValues[131] = d131
		ps249.OverlayValues[132] = d132
		ps249.OverlayValues[133] = d133
		ps249.OverlayValues[134] = d134
		ps249.OverlayValues[135] = d135
		ps249.OverlayValues[136] = d136
		ps249.OverlayValues[137] = d137
		ps249.OverlayValues[138] = d138
		ps249.OverlayValues[139] = d139
		ps249.OverlayValues[142] = d142
		ps249.OverlayValues[227] = d227
		ps249.OverlayValues[228] = d228
		ps249.OverlayValues[229] = d229
		ps249.OverlayValues[230] = d230
		ps249.OverlayValues[232] = d232
		ps249.OverlayValues[233] = d233
		ps249.OverlayValues[234] = d234
		ps249.OverlayValues[235] = d235
		ps249.OverlayValues[236] = d236
		ps249.OverlayValues[237] = d237
		ps249.OverlayValues[238] = d238
		ps249.OverlayValues[239] = d239
		ps249.OverlayValues[241] = d241
		ps249.OverlayValues[243] = d243
		ps249.OverlayValues[244] = d244
		ps249.OverlayValues[245] = d245
		ps249.OverlayValues[246] = d246
		ps249.OverlayValues[247] = d247
		ps249.OverlayValues[250] = d250
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
		snap264 := d16
		snap265 := d17
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
		snap297 := d50
		snap298 := d51
		snap299 := d54
		snap300 := d55
		snap301 := d56
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
		snap329 := d138
		snap330 := d139
		snap331 := d142
		snap332 := d227
		snap333 := d228
		snap334 := d229
		snap335 := d230
		snap336 := d232
		snap337 := d233
		snap338 := d234
		snap339 := d235
		snap340 := d236
		snap341 := d237
		snap342 := d238
		snap343 := d239
		snap344 := d241
		snap345 := d243
		snap346 := d244
		snap347 := d245
		snap348 := d246
		snap349 := d247
		snap350 := d250
		alloc351 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps248)
		}
		ctx.RestoreAllocState(alloc351)
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
		d16 = snap264
		d17 = snap265
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
		d50 = snap297
		d51 = snap298
		d54 = snap299
		d55 = snap300
		d56 = snap301
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
		d138 = snap329
		d139 = snap330
		d142 = snap331
		d227 = snap332
		d228 = snap333
		d229 = snap334
		d230 = snap335
		d232 = snap336
		d233 = snap337
		d234 = snap338
		d235 = snap339
		d236 = snap340
		d237 = snap341
		d238 = snap342
		d239 = snap343
		d241 = snap344
		d243 = snap345
		d244 = snap346
		d245 = snap347
		d246 = snap348
		d247 = snap349
		d250 = snap350
		if !bbs[6].Rendered {
			return bbs[6].RenderPS(ps249)
		}
		return result
		ctx.FreeDesc(&d236)
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
			d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0}
			ctx.BindReg(r0, &d3)
		} else {
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
			d16 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
			d17 = ps.OverlayValues[17]
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
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
			d51 = ps.OverlayValues[51]
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
		if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != scm.LocNone {
			d142 = ps.OverlayValues[142]
		}
		if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
			d227 = ps.OverlayValues[227]
		}
		if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
			d228 = ps.OverlayValues[228]
		}
		if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != scm.LocNone {
			d229 = ps.OverlayValues[229]
		}
		if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
			d230 = ps.OverlayValues[230]
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
		if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
			d238 = ps.OverlayValues[238]
		}
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
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
		if len(ps.OverlayValues) > 250 && ps.OverlayValues[250].Loc != scm.LocNone {
			d250 = ps.OverlayValues[250]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d3)
		ctx.EnsureDesc(&d3)
		var d352 scm.JITValueDesc
		if d3.Loc == scm.LocImm {
			d352 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d3.Reg)
			ctx.EmitMovRegReg(scratch, d3.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d352 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d352)
		}
		if d352.Loc == scm.LocImm {
			d352 = scm.JITValueDesc{Loc: scm.LocImm, Type: d352.Type, Imm: scm.NewInt(int64(uint64(d352.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d352.Reg, 32)
			ctx.EmitShrRegImm8(d352.Reg, 32)
		}
		if d352.Loc == scm.LocReg && d3.Loc == scm.LocReg && d352.Reg == d3.Reg {
			ctx.TransferReg(d3.Reg)
			d3.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d352)
		ctx.EmitStoreToStack(d352, int32(bbs[4].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d352)
		if ps.General {
			ctx.SyncDesc(&d3)
			if d3.Loc == scm.LocReg {
				ctx.ProtectReg(d3.Reg)
			} else if d3.Loc == scm.LocRegPair {
				ctx.ProtectReg(d3.Reg)
				ctx.ProtectReg(d3.Reg2)
			}
			ctx.SyncDesc(&d5)
			if d5.Loc == scm.LocReg {
				ctx.ProtectReg(d5.Reg)
			} else if d5.Loc == scm.LocRegPair {
				ctx.ProtectReg(d5.Reg)
				ctx.ProtectReg(d5.Reg2)
			}
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
			ctx.EmitStoreToStack(d354, int32(bbs[4].PhiBase)+int32(16))
			d355 = d5
			if d355.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d355)
			d356 = d355
			if d356.Loc == scm.LocImm {
				d356 = scm.JITValueDesc{Loc: scm.LocImm, Type: d356.Type, Imm: scm.NewInt(int64(uint64(d356.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d356.Reg, 32)
				ctx.EmitShrRegImm8(d356.Reg, 32)
			}
			ctx.EmitStoreToStack(d356, int32(bbs[4].PhiBase)+int32(32))
			if d3.Loc == scm.LocReg {
				ctx.UnprotectReg(d3.Reg)
			} else if d3.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d3.Reg)
				ctx.UnprotectReg(d3.Reg2)
			}
			if d5.Loc == scm.LocReg {
				ctx.UnprotectReg(d5.Reg)
			} else if d5.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d5.Reg)
				ctx.UnprotectReg(d5.Reg2)
			}
		}
		ps357 := scm.PhiState{General: ps.General}
		ps357.OverlayValues = make([]scm.JITValueDesc, 357)
		ps357.OverlayValues[3] = d3
		ps357.OverlayValues[4] = d4
		ps357.OverlayValues[5] = d5
		ps357.OverlayValues[6] = d6
		ps357.OverlayValues[7] = d7
		ps357.OverlayValues[8] = d8
		ps357.OverlayValues[9] = d9
		ps357.OverlayValues[10] = d10
		ps357.OverlayValues[11] = d11
		ps357.OverlayValues[12] = d12
		ps357.OverlayValues[13] = d13
		ps357.OverlayValues[14] = d14
		ps357.OverlayValues[15] = d15
		ps357.OverlayValues[16] = d16
		ps357.OverlayValues[17] = d17
		ps357.OverlayValues[19] = d19
		ps357.OverlayValues[20] = d20
		ps357.OverlayValues[21] = d21
		ps357.OverlayValues[22] = d22
		ps357.OverlayValues[23] = d23
		ps357.OverlayValues[24] = d24
		ps357.OverlayValues[25] = d25
		ps357.OverlayValues[26] = d26
		ps357.OverlayValues[27] = d27
		ps357.OverlayValues[28] = d28
		ps357.OverlayValues[29] = d29
		ps357.OverlayValues[30] = d30
		ps357.OverlayValues[31] = d31
		ps357.OverlayValues[32] = d32
		ps357.OverlayValues[33] = d33
		ps357.OverlayValues[34] = d34
		ps357.OverlayValues[35] = d35
		ps357.OverlayValues[36] = d36
		ps357.OverlayValues[37] = d37
		ps357.OverlayValues[38] = d38
		ps357.OverlayValues[39] = d39
		ps357.OverlayValues[40] = d40
		ps357.OverlayValues[41] = d41
		ps357.OverlayValues[42] = d42
		ps357.OverlayValues[43] = d43
		ps357.OverlayValues[44] = d44
		ps357.OverlayValues[45] = d45
		ps357.OverlayValues[46] = d46
		ps357.OverlayValues[47] = d47
		ps357.OverlayValues[48] = d48
		ps357.OverlayValues[49] = d49
		ps357.OverlayValues[50] = d50
		ps357.OverlayValues[51] = d51
		ps357.OverlayValues[54] = d54
		ps357.OverlayValues[55] = d55
		ps357.OverlayValues[56] = d56
		ps357.OverlayValues[111] = d111
		ps357.OverlayValues[112] = d112
		ps357.OverlayValues[113] = d113
		ps357.OverlayValues[114] = d114
		ps357.OverlayValues[115] = d115
		ps357.OverlayValues[116] = d116
		ps357.OverlayValues[117] = d117
		ps357.OverlayValues[118] = d118
		ps357.OverlayValues[119] = d119
		ps357.OverlayValues[120] = d120
		ps357.OverlayValues[121] = d121
		ps357.OverlayValues[122] = d122
		ps357.OverlayValues[123] = d123
		ps357.OverlayValues[124] = d124
		ps357.OverlayValues[125] = d125
		ps357.OverlayValues[126] = d126
		ps357.OverlayValues[127] = d127
		ps357.OverlayValues[128] = d128
		ps357.OverlayValues[129] = d129
		ps357.OverlayValues[130] = d130
		ps357.OverlayValues[131] = d131
		ps357.OverlayValues[132] = d132
		ps357.OverlayValues[133] = d133
		ps357.OverlayValues[134] = d134
		ps357.OverlayValues[135] = d135
		ps357.OverlayValues[136] = d136
		ps357.OverlayValues[137] = d137
		ps357.OverlayValues[138] = d138
		ps357.OverlayValues[139] = d139
		ps357.OverlayValues[142] = d142
		ps357.OverlayValues[227] = d227
		ps357.OverlayValues[228] = d228
		ps357.OverlayValues[229] = d229
		ps357.OverlayValues[230] = d230
		ps357.OverlayValues[232] = d232
		ps357.OverlayValues[233] = d233
		ps357.OverlayValues[234] = d234
		ps357.OverlayValues[235] = d235
		ps357.OverlayValues[236] = d236
		ps357.OverlayValues[237] = d237
		ps357.OverlayValues[238] = d238
		ps357.OverlayValues[239] = d239
		ps357.OverlayValues[241] = d241
		ps357.OverlayValues[243] = d243
		ps357.OverlayValues[244] = d244
		ps357.OverlayValues[245] = d245
		ps357.OverlayValues[246] = d246
		ps357.OverlayValues[247] = d247
		ps357.OverlayValues[250] = d250
		ps357.OverlayValues[352] = d352
		ps357.OverlayValues[353] = d353
		ps357.OverlayValues[354] = d354
		ps357.OverlayValues[355] = d355
		ps357.OverlayValues[356] = d356
		ps357.PhiValues = make([]scm.JITValueDesc, 3)
		d358 = d3
		ps357.PhiValues[1] = d358
		d359 = d5
		ps357.PhiValues[2] = d359
		if ps357.General && bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
			return result
		}
		return bbs[4].RenderPS(ps357)
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
			d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0}
			ctx.BindReg(r0, &d3)
		} else {
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
			d16 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
			d17 = ps.OverlayValues[17]
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
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
			d51 = ps.OverlayValues[51]
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
		if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != scm.LocNone {
			d142 = ps.OverlayValues[142]
		}
		if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
			d227 = ps.OverlayValues[227]
		}
		if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
			d228 = ps.OverlayValues[228]
		}
		if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != scm.LocNone {
			d229 = ps.OverlayValues[229]
		}
		if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
			d230 = ps.OverlayValues[230]
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
		if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
			d238 = ps.OverlayValues[238]
		}
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
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
		if len(ps.OverlayValues) > 250 && ps.OverlayValues[250].Loc != scm.LocNone {
			d250 = ps.OverlayValues[250]
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
		if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != scm.LocNone {
			d355 = ps.OverlayValues[355]
		}
		if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != scm.LocNone {
			d356 = ps.OverlayValues[356]
		}
		if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != scm.LocNone {
			d358 = ps.OverlayValues[358]
		}
		if len(ps.OverlayValues) > 359 && ps.OverlayValues[359].Loc != scm.LocNone {
			d359 = ps.OverlayValues[359]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d7)
		d360 = d7
		_ = d360
		ctx.StabilizeDescForControlFlow(&d360)
		ctx.StabilizeDescForControlFlow(&d7)
		bbpos_3_0 := int32(-1)
		_ = bbpos_3_0
		lbl23 := ctx.ReserveLabel()
		_ = lbl23
		bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl23)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d360)
		ctx.EnsureDesc(&d360)
		var d361 scm.JITValueDesc
		if d360.Loc == scm.LocImm {
			d361 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d360.Imm.Int()))))}
		} else {
			r81 := ctx.AllocReg()
			ctx.EmitMovRegReg(r81, d360.Reg)
			ctx.EmitShlRegImm8(r81, 32)
			ctx.EmitShrRegImm8(r81, 32)
			d361 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
			ctx.BindReg(r81, &d361)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d362 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			r82 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r82, fieldAddr)
			d362 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r82}
			ctx.BindReg(r82, &d362)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			r83 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r83, thisptr.Reg, off)
			d362 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r83}
			ctx.BindReg(r83, &d362)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d362)
		ctx.EnsureDesc(&d362)
		var d363 scm.JITValueDesc
		if d362.Loc == scm.LocImm {
			d363 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d362.Imm.Int()))))}
		} else {
			r84 := ctx.AllocReg()
			ctx.EmitMovRegReg(r84, d362.Reg)
			ctx.EmitShlRegImm8(r84, 56)
			ctx.EmitShrRegImm8(r84, 56)
			d363 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r84}
			ctx.BindReg(r84, &d363)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d361)
		ctx.EnsureDesc(&d363)
		ctx.EnsureDescsTogether(&d361, &d363)
		var d364 scm.JITValueDesc
		if d361.Loc == scm.LocImm && d363.Loc == scm.LocImm {
			d364 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d361.Imm.Int() * d363.Imm.Int())}
		} else if d361.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d363.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d361.Imm.Int()))
			ctx.EmitImulInt64(scratch, d363.Reg)
			d364 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d364)
		} else if d363.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d361.Reg)
			ctx.EmitMovRegReg(scratch, d361.Reg)
			if d363.Imm.Int() >= -2147483648 && d363.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d363.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d363.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d364 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d364)
		} else {
			r85 := ctx.AllocRegExcept(d361.Reg, d363.Reg)
			ctx.EmitMovRegReg(r85, d361.Reg)
			ctx.EmitImulInt64(r85, d363.Reg)
			d364 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
			ctx.BindReg(r85, &d364)
		}
		if d364.Loc == scm.LocReg && d361.Loc == scm.LocReg && d364.Reg == d361.Reg {
			ctx.TransferReg(d361.Reg)
			d361.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d361)
		ctx.FreeDesc(&d363)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d364)
		var d365 scm.JITValueDesc
		if d364.Loc == scm.LocImm {
			d365 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d364.Imm.Int() / 64)}
		} else {
			r86 := ctx.AllocRegExcept(d364.Reg)
			ctx.EmitMovRegReg(r86, d364.Reg)
			ctx.EmitShrRegImm8(r86, 6)
			d365 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
			ctx.BindReg(r86, &d365)
		}
		if d365.Loc == scm.LocReg && d364.Loc == scm.LocReg && d365.Reg == d364.Reg {
			ctx.TransferReg(d364.Reg)
			d364.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d364)
		var d366 scm.JITValueDesc
		if d364.Loc == scm.LocImm {
			d366 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d364.Imm.Int() % 64)}
		} else {
			r87 := ctx.AllocRegExcept(d364.Reg)
			ctx.EmitMovRegReg(r87, d364.Reg)
			ctx.EmitAndRegImm32(r87, 63)
			d366 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r87}
			ctx.BindReg(r87, &d366)
		}
		if d366.Loc == scm.LocReg && d364.Loc == scm.LocReg && d366.Reg == d364.Reg {
			ctx.TransferReg(d364.Reg)
			d364.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d364)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d367 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
			r88 := ctx.AllocReg()
			r89 := ctx.AllocRegExcept(r88)
			r90 := ctx.AllocRegExcept(r88, r89)
			ctx.EmitMovRegMem64(r88, fieldAddr)
			ctx.EmitMovRegMem64(r89, fieldAddr+8)
			ctx.EmitMovRegMem64(r90, fieldAddr+16)
			d367 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r88, Reg2: r89, Reg3: r90}
			ctx.BindReg(r88, &d367)
			ctx.BindReg(r89, &d367)
			ctx.BindReg(r90, &d367)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
			r91 := ctx.AllocReg()
			r92 := ctx.AllocRegExcept(r91)
			r93 := ctx.AllocRegExcept(r91, r92)
			ctx.EmitMovRegMem(r91, thisptr.Reg, off)
			ctx.EmitMovRegMem(r92, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r93, thisptr.Reg, off+16)
			d367 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r91, Reg2: r92, Reg3: r93}
			ctx.BindReg(r91, &d367)
			ctx.BindReg(r92, &d367)
			ctx.BindReg(r93, &d367)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d365)
		ctx.ReclaimUntrackedRegs()
		d369 = ctx.EmitSliceElementAddress(&d367, &d365, 8)
		ctx.EnsureDesc(&d369)
		ctx.EmitMovRegMem(d369.Reg, d369.Reg, 0)
		d368 = d369
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d368)
		ctx.EnsureDesc(&d366)
		var d370 scm.JITValueDesc
		if d368.Loc == scm.LocImm && d366.Loc == scm.LocImm {
			d370 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d368.Imm.Int()) << uint64(d366.Imm.Int())))}
		} else if d366.Loc == scm.LocImm {
			r94 := ctx.AllocRegExcept(d368.Reg)
			ctx.EmitMovRegReg(r94, d368.Reg)
			ctx.EmitShlRegImm8(r94, uint8(d366.Imm.Int()))
			d370 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
			ctx.BindReg(r94, &d370)
		} else {
			{
				shiftSrc := d368.Reg
				r95 := ctx.AllocRegExcept(d368.Reg)
				ctx.EmitMovRegReg(r95, d368.Reg)
				shiftSrc = r95
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d366.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d366.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d366.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d370 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d370)
			}
		}
		if d370.Loc == scm.LocReg && d368.Loc == scm.LocReg && d370.Reg == d368.Reg {
			ctx.TransferReg(d368.Reg)
			d368.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d368)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d365)
		ctx.EnsureDesc(&d365)
		var d371 scm.JITValueDesc
		if d365.Loc == scm.LocImm {
			d371 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d365.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d365.Reg)
			ctx.EmitMovRegReg(scratch, d365.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d371 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d371)
		}
		if d371.Loc == scm.LocReg && d365.Loc == scm.LocReg && d371.Reg == d365.Reg {
			ctx.TransferReg(d365.Reg)
			d365.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d365)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d371)
		ctx.ReclaimUntrackedRegs()
		d373 = ctx.EmitSliceElementAddress(&d367, &d371, 8)
		ctx.EnsureDesc(&d373)
		ctx.EmitMovRegMem(d373.Reg, d373.Reg, 0)
		d372 = d373
		ctx.FreeDesc(&d371)
		ctx.ReclaimUntrackedRegs()
		d374 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d366)
		ctx.EnsureDescsTogether(&d374, &d366)
		var d375 scm.JITValueDesc
		if d374.Loc == scm.LocImm && d366.Loc == scm.LocImm {
			d375 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d374.Imm.Int() - d366.Imm.Int())}
		} else if d366.Loc == scm.LocImm && d366.Imm.Int() == 0 {
			r96 := ctx.AllocRegExcept(d374.Reg)
			ctx.EmitMovRegReg(r96, d374.Reg)
			d375 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
			ctx.BindReg(r96, &d375)
		} else if d374.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d366.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d374.Imm.Int()))
			ctx.EmitSubInt64(scratch, d366.Reg)
			d375 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d375)
		} else if d366.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d374.Reg)
			ctx.EmitMovRegReg(scratch, d374.Reg)
			if d366.Imm.Int() >= -2147483648 && d366.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d366.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d366.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d375 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d375)
		} else {
			r97 := ctx.AllocRegExcept(d374.Reg, d366.Reg)
			ctx.EmitMovRegReg(r97, d374.Reg)
			ctx.EmitSubInt64(r97, d366.Reg)
			d375 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
			ctx.BindReg(r97, &d375)
		}
		if d375.Loc == scm.LocReg && d374.Loc == scm.LocReg && d375.Reg == d374.Reg {
			ctx.TransferReg(d374.Reg)
			d374.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d366)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d372)
		ctx.EnsureDesc(&d375)
		var d376 scm.JITValueDesc
		if d372.Loc == scm.LocImm && d375.Loc == scm.LocImm {
			d376 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d372.Imm.Int()) >> uint64(d375.Imm.Int())))}
		} else if d375.Loc == scm.LocImm {
			r98 := ctx.AllocRegExcept(d372.Reg)
			ctx.EmitMovRegReg(r98, d372.Reg)
			ctx.EmitShrRegImm8(r98, uint8(d375.Imm.Int()))
			d376 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r98}
			ctx.BindReg(r98, &d376)
		} else {
			{
				shiftSrc := d372.Reg
				r99 := ctx.AllocRegExcept(d372.Reg)
				ctx.EmitMovRegReg(r99, d372.Reg)
				shiftSrc = r99
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d375.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d375.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d375.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d376 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d376)
			}
		}
		if d376.Loc == scm.LocReg && d372.Loc == scm.LocReg && d376.Reg == d372.Reg {
			ctx.TransferReg(d372.Reg)
			d372.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d372)
		ctx.FreeDesc(&d375)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d370)
		ctx.EnsureDesc(&d376)
		var d377 scm.JITValueDesc
		if d370.Loc == scm.LocImm && d376.Loc == scm.LocImm {
			d377 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d370.Imm.Int() | d376.Imm.Int())}
		} else if d370.Loc == scm.LocImm && d370.Imm.Int() == 0 {
			d377 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d376.Reg}
			ctx.BindReg(d376.Reg, &d377)
		} else if d376.Loc == scm.LocImm && d376.Imm.Int() == 0 {
			r100 := ctx.AllocRegExcept(d370.Reg)
			ctx.EmitMovRegReg(r100, d370.Reg)
			d377 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r100}
			ctx.BindReg(r100, &d377)
		} else if d370.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d376.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d370.Imm.Int()))
			ctx.EmitOrInt64(scratch, d376.Reg)
			d377 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d377)
		} else if d376.Loc == scm.LocImm {
			r101 := ctx.AllocRegExcept(d370.Reg)
			ctx.EmitMovRegReg(r101, d370.Reg)
			if d376.Imm.Int() >= -2147483648 && d376.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r101, int32(d376.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d376.Imm.Int()))
				ctx.EmitOrInt64(r101, scm.RegR11)
			}
			d377 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r101}
			ctx.BindReg(r101, &d377)
		} else {
			r102 := ctx.AllocRegExcept(d370.Reg, d376.Reg)
			ctx.EmitMovRegReg(r102, d370.Reg)
			ctx.EmitOrInt64(r102, d376.Reg)
			d377 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r102}
			ctx.BindReg(r102, &d377)
		}
		if d377.Loc == scm.LocReg && d370.Loc == scm.LocReg && d377.Reg == d370.Reg {
			ctx.TransferReg(d370.Reg)
			d370.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d370)
		ctx.FreeDesc(&d376)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d362)
		ctx.EnsureDesc(&d362)
		var d378 scm.JITValueDesc
		if d362.Loc == scm.LocImm {
			d378 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d362.Imm.Int()))))}
		} else {
			r103 := ctx.AllocReg()
			ctx.EmitMovRegReg(r103, d362.Reg)
			ctx.EmitShlRegImm8(r103, 56)
			ctx.EmitShrRegImm8(r103, 56)
			d378 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r103}
			ctx.BindReg(r103, &d378)
		}
		ctx.ReclaimUntrackedRegs()
		d379 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d378)
		ctx.EnsureDescsTogether(&d379, &d378)
		var d380 scm.JITValueDesc
		if d379.Loc == scm.LocImm && d378.Loc == scm.LocImm {
			d380 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d379.Imm.Int() - d378.Imm.Int())}
		} else if d378.Loc == scm.LocImm && d378.Imm.Int() == 0 {
			r104 := ctx.AllocRegExcept(d379.Reg)
			ctx.EmitMovRegReg(r104, d379.Reg)
			d380 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r104}
			ctx.BindReg(r104, &d380)
		} else if d379.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d378.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d379.Imm.Int()))
			ctx.EmitSubInt64(scratch, d378.Reg)
			d380 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d380)
		} else if d378.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d379.Reg)
			ctx.EmitMovRegReg(scratch, d379.Reg)
			if d378.Imm.Int() >= -2147483648 && d378.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d378.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d378.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d380 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d380)
		} else {
			r105 := ctx.AllocRegExcept(d379.Reg, d378.Reg)
			ctx.EmitMovRegReg(r105, d379.Reg)
			ctx.EmitSubInt64(r105, d378.Reg)
			d380 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r105}
			ctx.BindReg(r105, &d380)
		}
		if d380.Loc == scm.LocReg && d379.Loc == scm.LocReg && d380.Reg == d379.Reg {
			ctx.TransferReg(d379.Reg)
			d379.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d378)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d377)
		ctx.EnsureDesc(&d380)
		var d381 scm.JITValueDesc
		if d377.Loc == scm.LocImm && d380.Loc == scm.LocImm {
			d381 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d377.Imm.Int()) >> uint64(d380.Imm.Int())))}
		} else if d380.Loc == scm.LocImm {
			r106 := ctx.AllocRegExcept(d377.Reg)
			ctx.EmitMovRegReg(r106, d377.Reg)
			ctx.EmitShrRegImm8(r106, uint8(d380.Imm.Int()))
			d381 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r106}
			ctx.BindReg(r106, &d381)
		} else {
			{
				shiftSrc := d377.Reg
				r107 := ctx.AllocRegExcept(d377.Reg)
				ctx.EmitMovRegReg(r107, d377.Reg)
				shiftSrc = r107
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
		ctx.EnsureDesc(&d381)
		ctx.EnsureDesc(&d381)
		ctx.EnsureDesc(&d381)
		var d382 scm.JITValueDesc
		if d381.Loc == scm.LocImm {
			d382 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d381.Imm.Int()))))}
		} else {
			r108 := ctx.AllocReg()
			ctx.EmitMovRegReg(r108, d381.Reg)
			d382 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
			ctx.BindReg(r108, &d382)
		}
		ctx.FreeDesc(&d381)
		ctx.EnsureDesc(&d382)
		ctx.EnsureDesc(&d47)
		ctx.EnsureDescsTogether(&d382, &d47)
		var d383 scm.JITValueDesc
		if d382.Loc == scm.LocImm && d47.Loc == scm.LocImm {
			d383 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d382.Imm.Int() + d47.Imm.Int())}
		} else if d47.Loc == scm.LocImm && d47.Imm.Int() == 0 {
			r109 := ctx.AllocRegExcept(d382.Reg)
			ctx.EmitMovRegReg(r109, d382.Reg)
			d383 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r109}
			ctx.BindReg(r109, &d383)
		} else if d382.Loc == scm.LocImm && d382.Imm.Int() == 0 {
			d383 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d47.Reg}
			ctx.BindReg(d47.Reg, &d383)
		} else if d382.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d47.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d382.Imm.Int()))
			ctx.EmitAddInt64(scratch, d47.Reg)
			d383 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d383)
		} else if d47.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d382.Reg)
			ctx.EmitMovRegReg(scratch, d382.Reg)
			if d47.Imm.Int() >= -2147483648 && d47.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d47.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d47.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d383 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d383)
		} else {
			r110 := ctx.AllocRegExcept(d382.Reg, d47.Reg)
			ctx.EmitMovRegReg(r110, d382.Reg)
			ctx.EmitAddInt64(r110, d47.Reg)
			d383 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
			ctx.BindReg(r110, &d383)
		}
		if d383.Loc == scm.LocReg && d382.Loc == scm.LocReg && d383.Reg == d382.Reg {
			ctx.TransferReg(d382.Reg)
			d382.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d382)
		ctx.EnsureDesc(&d383)
		ctx.EnsureDesc(&d383)
		var d384 scm.JITValueDesc
		if d383.Loc == scm.LocImm {
			d384 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d383.Imm.Int()))))}
		} else {
			r111 := ctx.AllocReg()
			ctx.EmitMovRegReg(r111, d383.Reg)
			ctx.EmitShlRegImm8(r111, 32)
			ctx.EmitShrRegImm8(r111, 32)
			d384 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r111}
			ctx.BindReg(r111, &d384)
		}
		ctx.FreeDesc(&d383)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d384)
		ctx.EnsureDescsTogether(&idxInt, &d384)
		var d385 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm && d384.Loc == scm.LocImm {
			d385 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d384.Imm.Int()))}
		} else if d384.Loc == scm.LocImm {
			r112 := ctx.AllocRegExcept(idxInt.Reg)
			if d384.Imm.Int() >= -2147483648 && d384.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(idxInt.Reg, int32(d384.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d384.Imm.Int()))
				ctx.EmitCmpInt64(idxInt.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r112, scm.CondUnsignedBelow)
			d385 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r112}
			ctx.BindReg(r112, &d385)
		} else if idxInt.Loc == scm.LocImm {
			r113 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d384.Reg)
			ctx.EmitSetcc(r113, scm.CondUnsignedBelow)
			d385 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r113}
			ctx.BindReg(r113, &d385)
		} else {
			r114 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitCmpInt64(idxInt.Reg, d384.Reg)
			ctx.EmitSetcc(r114, scm.CondUnsignedBelow)
			d385 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r114}
			ctx.BindReg(r114, &d385)
		}
		ctx.FreeDesc(&d384)
		d386 = d385
		ctx.EnsureDesc(&d386)
		if d386.Loc != scm.LocImm && d386.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d386.Loc == scm.LocImm {
			if d386.Imm.Bool() {
				if ps.General {
				}
				ps387 := scm.PhiState{General: ps.General}
				ps387.OverlayValues = make([]scm.JITValueDesc, 387)
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
				ps387.OverlayValues[16] = d16
				ps387.OverlayValues[17] = d17
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
				ps387.OverlayValues[50] = d50
				ps387.OverlayValues[51] = d51
				ps387.OverlayValues[54] = d54
				ps387.OverlayValues[55] = d55
				ps387.OverlayValues[56] = d56
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
				ps387.OverlayValues[138] = d138
				ps387.OverlayValues[139] = d139
				ps387.OverlayValues[142] = d142
				ps387.OverlayValues[227] = d227
				ps387.OverlayValues[228] = d228
				ps387.OverlayValues[229] = d229
				ps387.OverlayValues[230] = d230
				ps387.OverlayValues[232] = d232
				ps387.OverlayValues[233] = d233
				ps387.OverlayValues[234] = d234
				ps387.OverlayValues[235] = d235
				ps387.OverlayValues[236] = d236
				ps387.OverlayValues[237] = d237
				ps387.OverlayValues[238] = d238
				ps387.OverlayValues[239] = d239
				ps387.OverlayValues[241] = d241
				ps387.OverlayValues[243] = d243
				ps387.OverlayValues[244] = d244
				ps387.OverlayValues[245] = d245
				ps387.OverlayValues[246] = d246
				ps387.OverlayValues[247] = d247
				ps387.OverlayValues[250] = d250
				ps387.OverlayValues[352] = d352
				ps387.OverlayValues[353] = d353
				ps387.OverlayValues[354] = d354
				ps387.OverlayValues[355] = d355
				ps387.OverlayValues[356] = d356
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
				ps387.OverlayValues[385] = d385
				ps387.OverlayValues[386] = d386
				return bbs[7].RenderPS(ps387)
			}
			if ps.General {
			}
			ps388 := scm.PhiState{General: ps.General}
			ps388.OverlayValues = make([]scm.JITValueDesc, 387)
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
			ps388.OverlayValues[16] = d16
			ps388.OverlayValues[17] = d17
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
			ps388.OverlayValues[50] = d50
			ps388.OverlayValues[51] = d51
			ps388.OverlayValues[54] = d54
			ps388.OverlayValues[55] = d55
			ps388.OverlayValues[56] = d56
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
			ps388.OverlayValues[138] = d138
			ps388.OverlayValues[139] = d139
			ps388.OverlayValues[142] = d142
			ps388.OverlayValues[227] = d227
			ps388.OverlayValues[228] = d228
			ps388.OverlayValues[229] = d229
			ps388.OverlayValues[230] = d230
			ps388.OverlayValues[232] = d232
			ps388.OverlayValues[233] = d233
			ps388.OverlayValues[234] = d234
			ps388.OverlayValues[235] = d235
			ps388.OverlayValues[236] = d236
			ps388.OverlayValues[237] = d237
			ps388.OverlayValues[238] = d238
			ps388.OverlayValues[239] = d239
			ps388.OverlayValues[241] = d241
			ps388.OverlayValues[243] = d243
			ps388.OverlayValues[244] = d244
			ps388.OverlayValues[245] = d245
			ps388.OverlayValues[246] = d246
			ps388.OverlayValues[247] = d247
			ps388.OverlayValues[250] = d250
			ps388.OverlayValues[352] = d352
			ps388.OverlayValues[353] = d353
			ps388.OverlayValues[354] = d354
			ps388.OverlayValues[355] = d355
			ps388.OverlayValues[356] = d356
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
			ps388.OverlayValues[385] = d385
			ps388.OverlayValues[386] = d386
			return bbs[9].RenderPS(ps388)
		}
		if !ps.General {
			ps.General = true
			return bbs[6].RenderPS(ps)
		}
		lbl24 := ctx.ReserveLabel()
		lbl25 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d386.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl24)
		ctx.EmitJmp(lbl25)
		ctx.MarkLabel(lbl24)
		ctx.EmitJmp(lbl8)
		ctx.MarkLabel(lbl25)
		ctx.EmitJmp(lbl10)
		ps389 := scm.PhiState{General: true}
		ps389.OverlayValues = make([]scm.JITValueDesc, 387)
		ps389.OverlayValues[3] = d3
		ps389.OverlayValues[4] = d4
		ps389.OverlayValues[5] = d5
		ps389.OverlayValues[6] = d6
		ps389.OverlayValues[7] = d7
		ps389.OverlayValues[8] = d8
		ps389.OverlayValues[9] = d9
		ps389.OverlayValues[10] = d10
		ps389.OverlayValues[11] = d11
		ps389.OverlayValues[12] = d12
		ps389.OverlayValues[13] = d13
		ps389.OverlayValues[14] = d14
		ps389.OverlayValues[15] = d15
		ps389.OverlayValues[16] = d16
		ps389.OverlayValues[17] = d17
		ps389.OverlayValues[19] = d19
		ps389.OverlayValues[20] = d20
		ps389.OverlayValues[21] = d21
		ps389.OverlayValues[22] = d22
		ps389.OverlayValues[23] = d23
		ps389.OverlayValues[24] = d24
		ps389.OverlayValues[25] = d25
		ps389.OverlayValues[26] = d26
		ps389.OverlayValues[27] = d27
		ps389.OverlayValues[28] = d28
		ps389.OverlayValues[29] = d29
		ps389.OverlayValues[30] = d30
		ps389.OverlayValues[31] = d31
		ps389.OverlayValues[32] = d32
		ps389.OverlayValues[33] = d33
		ps389.OverlayValues[34] = d34
		ps389.OverlayValues[35] = d35
		ps389.OverlayValues[36] = d36
		ps389.OverlayValues[37] = d37
		ps389.OverlayValues[38] = d38
		ps389.OverlayValues[39] = d39
		ps389.OverlayValues[40] = d40
		ps389.OverlayValues[41] = d41
		ps389.OverlayValues[42] = d42
		ps389.OverlayValues[43] = d43
		ps389.OverlayValues[44] = d44
		ps389.OverlayValues[45] = d45
		ps389.OverlayValues[46] = d46
		ps389.OverlayValues[47] = d47
		ps389.OverlayValues[48] = d48
		ps389.OverlayValues[49] = d49
		ps389.OverlayValues[50] = d50
		ps389.OverlayValues[51] = d51
		ps389.OverlayValues[54] = d54
		ps389.OverlayValues[55] = d55
		ps389.OverlayValues[56] = d56
		ps389.OverlayValues[111] = d111
		ps389.OverlayValues[112] = d112
		ps389.OverlayValues[113] = d113
		ps389.OverlayValues[114] = d114
		ps389.OverlayValues[115] = d115
		ps389.OverlayValues[116] = d116
		ps389.OverlayValues[117] = d117
		ps389.OverlayValues[118] = d118
		ps389.OverlayValues[119] = d119
		ps389.OverlayValues[120] = d120
		ps389.OverlayValues[121] = d121
		ps389.OverlayValues[122] = d122
		ps389.OverlayValues[123] = d123
		ps389.OverlayValues[124] = d124
		ps389.OverlayValues[125] = d125
		ps389.OverlayValues[126] = d126
		ps389.OverlayValues[127] = d127
		ps389.OverlayValues[128] = d128
		ps389.OverlayValues[129] = d129
		ps389.OverlayValues[130] = d130
		ps389.OverlayValues[131] = d131
		ps389.OverlayValues[132] = d132
		ps389.OverlayValues[133] = d133
		ps389.OverlayValues[134] = d134
		ps389.OverlayValues[135] = d135
		ps389.OverlayValues[136] = d136
		ps389.OverlayValues[137] = d137
		ps389.OverlayValues[138] = d138
		ps389.OverlayValues[139] = d139
		ps389.OverlayValues[142] = d142
		ps389.OverlayValues[227] = d227
		ps389.OverlayValues[228] = d228
		ps389.OverlayValues[229] = d229
		ps389.OverlayValues[230] = d230
		ps389.OverlayValues[232] = d232
		ps389.OverlayValues[233] = d233
		ps389.OverlayValues[234] = d234
		ps389.OverlayValues[235] = d235
		ps389.OverlayValues[236] = d236
		ps389.OverlayValues[237] = d237
		ps389.OverlayValues[238] = d238
		ps389.OverlayValues[239] = d239
		ps389.OverlayValues[241] = d241
		ps389.OverlayValues[243] = d243
		ps389.OverlayValues[244] = d244
		ps389.OverlayValues[245] = d245
		ps389.OverlayValues[246] = d246
		ps389.OverlayValues[247] = d247
		ps389.OverlayValues[250] = d250
		ps389.OverlayValues[352] = d352
		ps389.OverlayValues[353] = d353
		ps389.OverlayValues[354] = d354
		ps389.OverlayValues[355] = d355
		ps389.OverlayValues[356] = d356
		ps389.OverlayValues[358] = d358
		ps389.OverlayValues[359] = d359
		ps389.OverlayValues[360] = d360
		ps389.OverlayValues[361] = d361
		ps389.OverlayValues[362] = d362
		ps389.OverlayValues[363] = d363
		ps389.OverlayValues[364] = d364
		ps389.OverlayValues[365] = d365
		ps389.OverlayValues[366] = d366
		ps389.OverlayValues[367] = d367
		ps389.OverlayValues[368] = d368
		ps389.OverlayValues[369] = d369
		ps389.OverlayValues[370] = d370
		ps389.OverlayValues[371] = d371
		ps389.OverlayValues[372] = d372
		ps389.OverlayValues[373] = d373
		ps389.OverlayValues[374] = d374
		ps389.OverlayValues[375] = d375
		ps389.OverlayValues[376] = d376
		ps389.OverlayValues[377] = d377
		ps389.OverlayValues[378] = d378
		ps389.OverlayValues[379] = d379
		ps389.OverlayValues[380] = d380
		ps389.OverlayValues[381] = d381
		ps389.OverlayValues[382] = d382
		ps389.OverlayValues[383] = d383
		ps389.OverlayValues[384] = d384
		ps389.OverlayValues[385] = d385
		ps389.OverlayValues[386] = d386
		ps390 := scm.PhiState{General: true}
		ps390.OverlayValues = make([]scm.JITValueDesc, 387)
		ps390.OverlayValues[3] = d3
		ps390.OverlayValues[4] = d4
		ps390.OverlayValues[5] = d5
		ps390.OverlayValues[6] = d6
		ps390.OverlayValues[7] = d7
		ps390.OverlayValues[8] = d8
		ps390.OverlayValues[9] = d9
		ps390.OverlayValues[10] = d10
		ps390.OverlayValues[11] = d11
		ps390.OverlayValues[12] = d12
		ps390.OverlayValues[13] = d13
		ps390.OverlayValues[14] = d14
		ps390.OverlayValues[15] = d15
		ps390.OverlayValues[16] = d16
		ps390.OverlayValues[17] = d17
		ps390.OverlayValues[19] = d19
		ps390.OverlayValues[20] = d20
		ps390.OverlayValues[21] = d21
		ps390.OverlayValues[22] = d22
		ps390.OverlayValues[23] = d23
		ps390.OverlayValues[24] = d24
		ps390.OverlayValues[25] = d25
		ps390.OverlayValues[26] = d26
		ps390.OverlayValues[27] = d27
		ps390.OverlayValues[28] = d28
		ps390.OverlayValues[29] = d29
		ps390.OverlayValues[30] = d30
		ps390.OverlayValues[31] = d31
		ps390.OverlayValues[32] = d32
		ps390.OverlayValues[33] = d33
		ps390.OverlayValues[34] = d34
		ps390.OverlayValues[35] = d35
		ps390.OverlayValues[36] = d36
		ps390.OverlayValues[37] = d37
		ps390.OverlayValues[38] = d38
		ps390.OverlayValues[39] = d39
		ps390.OverlayValues[40] = d40
		ps390.OverlayValues[41] = d41
		ps390.OverlayValues[42] = d42
		ps390.OverlayValues[43] = d43
		ps390.OverlayValues[44] = d44
		ps390.OverlayValues[45] = d45
		ps390.OverlayValues[46] = d46
		ps390.OverlayValues[47] = d47
		ps390.OverlayValues[48] = d48
		ps390.OverlayValues[49] = d49
		ps390.OverlayValues[50] = d50
		ps390.OverlayValues[51] = d51
		ps390.OverlayValues[54] = d54
		ps390.OverlayValues[55] = d55
		ps390.OverlayValues[56] = d56
		ps390.OverlayValues[111] = d111
		ps390.OverlayValues[112] = d112
		ps390.OverlayValues[113] = d113
		ps390.OverlayValues[114] = d114
		ps390.OverlayValues[115] = d115
		ps390.OverlayValues[116] = d116
		ps390.OverlayValues[117] = d117
		ps390.OverlayValues[118] = d118
		ps390.OverlayValues[119] = d119
		ps390.OverlayValues[120] = d120
		ps390.OverlayValues[121] = d121
		ps390.OverlayValues[122] = d122
		ps390.OverlayValues[123] = d123
		ps390.OverlayValues[124] = d124
		ps390.OverlayValues[125] = d125
		ps390.OverlayValues[126] = d126
		ps390.OverlayValues[127] = d127
		ps390.OverlayValues[128] = d128
		ps390.OverlayValues[129] = d129
		ps390.OverlayValues[130] = d130
		ps390.OverlayValues[131] = d131
		ps390.OverlayValues[132] = d132
		ps390.OverlayValues[133] = d133
		ps390.OverlayValues[134] = d134
		ps390.OverlayValues[135] = d135
		ps390.OverlayValues[136] = d136
		ps390.OverlayValues[137] = d137
		ps390.OverlayValues[138] = d138
		ps390.OverlayValues[139] = d139
		ps390.OverlayValues[142] = d142
		ps390.OverlayValues[227] = d227
		ps390.OverlayValues[228] = d228
		ps390.OverlayValues[229] = d229
		ps390.OverlayValues[230] = d230
		ps390.OverlayValues[232] = d232
		ps390.OverlayValues[233] = d233
		ps390.OverlayValues[234] = d234
		ps390.OverlayValues[235] = d235
		ps390.OverlayValues[236] = d236
		ps390.OverlayValues[237] = d237
		ps390.OverlayValues[238] = d238
		ps390.OverlayValues[239] = d239
		ps390.OverlayValues[241] = d241
		ps390.OverlayValues[243] = d243
		ps390.OverlayValues[244] = d244
		ps390.OverlayValues[245] = d245
		ps390.OverlayValues[246] = d246
		ps390.OverlayValues[247] = d247
		ps390.OverlayValues[250] = d250
		ps390.OverlayValues[352] = d352
		ps390.OverlayValues[353] = d353
		ps390.OverlayValues[354] = d354
		ps390.OverlayValues[355] = d355
		ps390.OverlayValues[356] = d356
		ps390.OverlayValues[358] = d358
		ps390.OverlayValues[359] = d359
		ps390.OverlayValues[360] = d360
		ps390.OverlayValues[361] = d361
		ps390.OverlayValues[362] = d362
		ps390.OverlayValues[363] = d363
		ps390.OverlayValues[364] = d364
		ps390.OverlayValues[365] = d365
		ps390.OverlayValues[366] = d366
		ps390.OverlayValues[367] = d367
		ps390.OverlayValues[368] = d368
		ps390.OverlayValues[369] = d369
		ps390.OverlayValues[370] = d370
		ps390.OverlayValues[371] = d371
		ps390.OverlayValues[372] = d372
		ps390.OverlayValues[373] = d373
		ps390.OverlayValues[374] = d374
		ps390.OverlayValues[375] = d375
		ps390.OverlayValues[376] = d376
		ps390.OverlayValues[377] = d377
		ps390.OverlayValues[378] = d378
		ps390.OverlayValues[379] = d379
		ps390.OverlayValues[380] = d380
		ps390.OverlayValues[381] = d381
		ps390.OverlayValues[382] = d382
		ps390.OverlayValues[383] = d383
		ps390.OverlayValues[384] = d384
		ps390.OverlayValues[385] = d385
		ps390.OverlayValues[386] = d386
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
		snap404 := d16
		snap405 := d17
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
		snap437 := d50
		snap438 := d51
		snap439 := d54
		snap440 := d55
		snap441 := d56
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
		snap469 := d138
		snap470 := d139
		snap471 := d142
		snap472 := d227
		snap473 := d228
		snap474 := d229
		snap475 := d230
		snap476 := d232
		snap477 := d233
		snap478 := d234
		snap479 := d235
		snap480 := d236
		snap481 := d237
		snap482 := d238
		snap483 := d239
		snap484 := d241
		snap485 := d243
		snap486 := d244
		snap487 := d245
		snap488 := d246
		snap489 := d247
		snap490 := d250
		snap491 := d352
		snap492 := d353
		snap493 := d354
		snap494 := d355
		snap495 := d356
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
		snap523 := d385
		snap524 := d386
		alloc525 := ctx.SnapshotAllocState()
		if !bbs[9].Rendered {
			bbs[9].RenderPS(ps390)
		}
		ctx.RestoreAllocState(alloc525)
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
		d16 = snap404
		d17 = snap405
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
		d50 = snap437
		d51 = snap438
		d54 = snap439
		d55 = snap440
		d56 = snap441
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
		d138 = snap469
		d139 = snap470
		d142 = snap471
		d227 = snap472
		d228 = snap473
		d229 = snap474
		d230 = snap475
		d232 = snap476
		d233 = snap477
		d234 = snap478
		d235 = snap479
		d236 = snap480
		d237 = snap481
		d238 = snap482
		d239 = snap483
		d241 = snap484
		d243 = snap485
		d244 = snap486
		d245 = snap487
		d246 = snap488
		d247 = snap489
		d250 = snap490
		d352 = snap491
		d353 = snap492
		d354 = snap493
		d355 = snap494
		d356 = snap495
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
		d385 = snap523
		d386 = snap524
		if !bbs[7].Rendered {
			return bbs[7].RenderPS(ps389)
		}
		return result
		ctx.FreeDesc(&d385)
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
			d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0}
			ctx.BindReg(r0, &d3)
		} else {
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
			d16 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
			d17 = ps.OverlayValues[17]
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
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
			d51 = ps.OverlayValues[51]
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
		if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != scm.LocNone {
			d142 = ps.OverlayValues[142]
		}
		if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
			d227 = ps.OverlayValues[227]
		}
		if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
			d228 = ps.OverlayValues[228]
		}
		if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != scm.LocNone {
			d229 = ps.OverlayValues[229]
		}
		if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
			d230 = ps.OverlayValues[230]
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
		if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
			d238 = ps.OverlayValues[238]
		}
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
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
		if len(ps.OverlayValues) > 250 && ps.OverlayValues[250].Loc != scm.LocNone {
			d250 = ps.OverlayValues[250]
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
		if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != scm.LocNone {
			d355 = ps.OverlayValues[355]
		}
		if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != scm.LocNone {
			d356 = ps.OverlayValues[356]
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
		if len(ps.OverlayValues) > 385 && ps.OverlayValues[385].Loc != scm.LocNone {
			d385 = ps.OverlayValues[385]
		}
		if len(ps.OverlayValues) > 386 && ps.OverlayValues[386].Loc != scm.LocNone {
			d386 = ps.OverlayValues[386]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d7)
		ctx.EnsureDesc(&d7)
		var d526 scm.JITValueDesc
		if d7.Loc == scm.LocImm {
			d526 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d7.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d7.Reg)
			ctx.EmitMovRegReg(scratch, d7.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d526 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d526)
		}
		if d526.Loc == scm.LocImm {
			d526 = scm.JITValueDesc{Loc: scm.LocImm, Type: d526.Type, Imm: scm.NewInt(int64(uint64(d526.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d526.Reg, 32)
			ctx.EmitShrRegImm8(d526.Reg, 32)
		}
		if d526.Loc == scm.LocReg && d7.Loc == scm.LocReg && d526.Reg == d7.Reg {
			ctx.TransferReg(d7.Reg)
			d7.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d526)
		ctx.EmitStoreToStack(d526, int32(bbs[8].PhiBase)+int32(16))
		ctx.StabilizeDescForControlFlow(&d526)
		if ps.General {
			ctx.SyncDesc(&d8)
			if d8.Loc == scm.LocReg {
				ctx.ProtectReg(d8.Reg)
			} else if d8.Loc == scm.LocRegPair {
				ctx.ProtectReg(d8.Reg)
				ctx.ProtectReg(d8.Reg2)
			}
			d527 = d8
			if d527.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d527)
			d528 = d527
			if d528.Loc == scm.LocImm {
				d528 = scm.JITValueDesc{Loc: scm.LocImm, Type: d528.Type, Imm: scm.NewInt(int64(uint64(d528.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d528.Reg, 32)
				ctx.EmitShrRegImm8(d528.Reg, 32)
			}
			ctx.EmitStoreToStack(d528, int32(bbs[8].PhiBase)+int32(0))
			if d8.Loc == scm.LocReg {
				ctx.UnprotectReg(d8.Reg)
			} else if d8.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d8.Reg)
				ctx.UnprotectReg(d8.Reg2)
			}
		}
		ps529 := scm.PhiState{General: ps.General}
		ps529.OverlayValues = make([]scm.JITValueDesc, 529)
		ps529.OverlayValues[3] = d3
		ps529.OverlayValues[4] = d4
		ps529.OverlayValues[5] = d5
		ps529.OverlayValues[6] = d6
		ps529.OverlayValues[7] = d7
		ps529.OverlayValues[8] = d8
		ps529.OverlayValues[9] = d9
		ps529.OverlayValues[10] = d10
		ps529.OverlayValues[11] = d11
		ps529.OverlayValues[12] = d12
		ps529.OverlayValues[13] = d13
		ps529.OverlayValues[14] = d14
		ps529.OverlayValues[15] = d15
		ps529.OverlayValues[16] = d16
		ps529.OverlayValues[17] = d17
		ps529.OverlayValues[19] = d19
		ps529.OverlayValues[20] = d20
		ps529.OverlayValues[21] = d21
		ps529.OverlayValues[22] = d22
		ps529.OverlayValues[23] = d23
		ps529.OverlayValues[24] = d24
		ps529.OverlayValues[25] = d25
		ps529.OverlayValues[26] = d26
		ps529.OverlayValues[27] = d27
		ps529.OverlayValues[28] = d28
		ps529.OverlayValues[29] = d29
		ps529.OverlayValues[30] = d30
		ps529.OverlayValues[31] = d31
		ps529.OverlayValues[32] = d32
		ps529.OverlayValues[33] = d33
		ps529.OverlayValues[34] = d34
		ps529.OverlayValues[35] = d35
		ps529.OverlayValues[36] = d36
		ps529.OverlayValues[37] = d37
		ps529.OverlayValues[38] = d38
		ps529.OverlayValues[39] = d39
		ps529.OverlayValues[40] = d40
		ps529.OverlayValues[41] = d41
		ps529.OverlayValues[42] = d42
		ps529.OverlayValues[43] = d43
		ps529.OverlayValues[44] = d44
		ps529.OverlayValues[45] = d45
		ps529.OverlayValues[46] = d46
		ps529.OverlayValues[47] = d47
		ps529.OverlayValues[48] = d48
		ps529.OverlayValues[49] = d49
		ps529.OverlayValues[50] = d50
		ps529.OverlayValues[51] = d51
		ps529.OverlayValues[54] = d54
		ps529.OverlayValues[55] = d55
		ps529.OverlayValues[56] = d56
		ps529.OverlayValues[111] = d111
		ps529.OverlayValues[112] = d112
		ps529.OverlayValues[113] = d113
		ps529.OverlayValues[114] = d114
		ps529.OverlayValues[115] = d115
		ps529.OverlayValues[116] = d116
		ps529.OverlayValues[117] = d117
		ps529.OverlayValues[118] = d118
		ps529.OverlayValues[119] = d119
		ps529.OverlayValues[120] = d120
		ps529.OverlayValues[121] = d121
		ps529.OverlayValues[122] = d122
		ps529.OverlayValues[123] = d123
		ps529.OverlayValues[124] = d124
		ps529.OverlayValues[125] = d125
		ps529.OverlayValues[126] = d126
		ps529.OverlayValues[127] = d127
		ps529.OverlayValues[128] = d128
		ps529.OverlayValues[129] = d129
		ps529.OverlayValues[130] = d130
		ps529.OverlayValues[131] = d131
		ps529.OverlayValues[132] = d132
		ps529.OverlayValues[133] = d133
		ps529.OverlayValues[134] = d134
		ps529.OverlayValues[135] = d135
		ps529.OverlayValues[136] = d136
		ps529.OverlayValues[137] = d137
		ps529.OverlayValues[138] = d138
		ps529.OverlayValues[139] = d139
		ps529.OverlayValues[142] = d142
		ps529.OverlayValues[227] = d227
		ps529.OverlayValues[228] = d228
		ps529.OverlayValues[229] = d229
		ps529.OverlayValues[230] = d230
		ps529.OverlayValues[232] = d232
		ps529.OverlayValues[233] = d233
		ps529.OverlayValues[234] = d234
		ps529.OverlayValues[235] = d235
		ps529.OverlayValues[236] = d236
		ps529.OverlayValues[237] = d237
		ps529.OverlayValues[238] = d238
		ps529.OverlayValues[239] = d239
		ps529.OverlayValues[241] = d241
		ps529.OverlayValues[243] = d243
		ps529.OverlayValues[244] = d244
		ps529.OverlayValues[245] = d245
		ps529.OverlayValues[246] = d246
		ps529.OverlayValues[247] = d247
		ps529.OverlayValues[250] = d250
		ps529.OverlayValues[352] = d352
		ps529.OverlayValues[353] = d353
		ps529.OverlayValues[354] = d354
		ps529.OverlayValues[355] = d355
		ps529.OverlayValues[356] = d356
		ps529.OverlayValues[358] = d358
		ps529.OverlayValues[359] = d359
		ps529.OverlayValues[360] = d360
		ps529.OverlayValues[361] = d361
		ps529.OverlayValues[362] = d362
		ps529.OverlayValues[363] = d363
		ps529.OverlayValues[364] = d364
		ps529.OverlayValues[365] = d365
		ps529.OverlayValues[366] = d366
		ps529.OverlayValues[367] = d367
		ps529.OverlayValues[368] = d368
		ps529.OverlayValues[369] = d369
		ps529.OverlayValues[370] = d370
		ps529.OverlayValues[371] = d371
		ps529.OverlayValues[372] = d372
		ps529.OverlayValues[373] = d373
		ps529.OverlayValues[374] = d374
		ps529.OverlayValues[375] = d375
		ps529.OverlayValues[376] = d376
		ps529.OverlayValues[377] = d377
		ps529.OverlayValues[378] = d378
		ps529.OverlayValues[379] = d379
		ps529.OverlayValues[380] = d380
		ps529.OverlayValues[381] = d381
		ps529.OverlayValues[382] = d382
		ps529.OverlayValues[383] = d383
		ps529.OverlayValues[384] = d384
		ps529.OverlayValues[385] = d385
		ps529.OverlayValues[386] = d386
		ps529.OverlayValues[526] = d526
		ps529.OverlayValues[527] = d527
		ps529.OverlayValues[528] = d528
		ps529.PhiValues = make([]scm.JITValueDesc, 2)
		d530 = d8
		ps529.PhiValues[0] = d530
		if ps529.General && bbs[8].Rendered {
			ctx.EmitJmp(lbl9)
			return result
		}
		return bbs[8].RenderPS(ps529)
		return result
	}
	bbs[8].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d531 := ps.PhiValues[0]
				ctx.EnsureDesc(&d531)
				ctx.EmitStoreToStack(d531, int32(bbs[8].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d532 := ps.PhiValues[1]
				ctx.EnsureDesc(&d532)
				ctx.EmitStoreToStack(d532, int32(bbs[8].PhiBase)+int32(16))
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
			d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0}
			ctx.BindReg(r0, &d3)
		} else {
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
			d16 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
			d17 = ps.OverlayValues[17]
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
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
			d51 = ps.OverlayValues[51]
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
		if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != scm.LocNone {
			d142 = ps.OverlayValues[142]
		}
		if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
			d227 = ps.OverlayValues[227]
		}
		if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
			d228 = ps.OverlayValues[228]
		}
		if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != scm.LocNone {
			d229 = ps.OverlayValues[229]
		}
		if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
			d230 = ps.OverlayValues[230]
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
		if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
			d238 = ps.OverlayValues[238]
		}
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
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
		if len(ps.OverlayValues) > 250 && ps.OverlayValues[250].Loc != scm.LocNone {
			d250 = ps.OverlayValues[250]
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
		if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != scm.LocNone {
			d355 = ps.OverlayValues[355]
		}
		if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != scm.LocNone {
			d356 = ps.OverlayValues[356]
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
		if len(ps.OverlayValues) > 385 && ps.OverlayValues[385].Loc != scm.LocNone {
			d385 = ps.OverlayValues[385]
		}
		if len(ps.OverlayValues) > 386 && ps.OverlayValues[386].Loc != scm.LocNone {
			d386 = ps.OverlayValues[386]
		}
		if len(ps.OverlayValues) > 526 && ps.OverlayValues[526].Loc != scm.LocNone {
			d526 = ps.OverlayValues[526]
		}
		if len(ps.OverlayValues) > 527 && ps.OverlayValues[527].Loc != scm.LocNone {
			d527 = ps.OverlayValues[527]
		}
		if len(ps.OverlayValues) > 528 && ps.OverlayValues[528].Loc != scm.LocNone {
			d528 = ps.OverlayValues[528]
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
		if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
			d10 = ps.PhiValues[0]
		}
		if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
			d11 = ps.PhiValues[1]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d10)
		ctx.StabilizeDescForControlFlow(&d11)
		ctx.EnsureDesc(&d10)
		ctx.EnsureDesc(&d11)
		ctx.EnsureDescsTogether(&d10, &d11)
		var d533 scm.JITValueDesc
		if d10.Loc == scm.LocImm && d11.Loc == scm.LocImm {
			d533 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d10.Imm.Int()) == uint64(d11.Imm.Int()))}
		} else if d11.Loc == scm.LocImm {
			r115 := ctx.AllocRegExcept(d10.Reg)
			if d11.Imm.Int() >= -2147483648 && d11.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d10.Reg, int32(d11.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d11.Imm.Int()))
				ctx.EmitCmpInt64(d10.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r115, scm.CondEqual)
			d533 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r115}
			ctx.BindReg(r115, &d533)
		} else if d10.Loc == scm.LocImm {
			r116 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d10.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d11.Reg)
			ctx.EmitSetcc(r116, scm.CondEqual)
			d533 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r116}
			ctx.BindReg(r116, &d533)
		} else {
			r117 := ctx.AllocRegExcept(d10.Reg)
			ctx.EmitCmpInt64(d10.Reg, d11.Reg)
			ctx.EmitSetcc(r117, scm.CondEqual)
			d533 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r117}
			ctx.BindReg(r117, &d533)
		}
		d534 = d533
		ctx.EnsureDesc(&d534)
		if d534.Loc != scm.LocImm && d534.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d534.Loc == scm.LocImm {
			if d534.Imm.Bool() {
				if ps.General {
					ctx.SyncDesc(&d10)
					if d10.Loc == scm.LocReg {
						ctx.ProtectReg(d10.Reg)
					} else if d10.Loc == scm.LocRegPair {
						ctx.ProtectReg(d10.Reg)
						ctx.ProtectReg(d10.Reg2)
					}
					d535 = d10
					if d535.Loc == scm.LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d535)
					d536 = d535
					if d536.Loc == scm.LocImm {
						d536 = scm.JITValueDesc{Loc: scm.LocImm, Type: d536.Type, Imm: scm.NewInt(int64(uint64(d536.Imm.Int()) & 0xffffffff))}
					} else {
						ctx.EmitShlRegImm8(d536.Reg, 32)
						ctx.EmitShrRegImm8(d536.Reg, 32)
					}
					ctx.EmitStoreToStack(d536, int32(bbs[2].PhiBase)+int32(0))
					if d10.Loc == scm.LocReg {
						ctx.UnprotectReg(d10.Reg)
					} else if d10.Loc == scm.LocRegPair {
						ctx.UnprotectReg(d10.Reg)
						ctx.UnprotectReg(d10.Reg2)
					}
				}
				ps537 := scm.PhiState{General: ps.General}
				ps537.OverlayValues = make([]scm.JITValueDesc, 537)
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
				ps537.OverlayValues[16] = d16
				ps537.OverlayValues[17] = d17
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
				ps537.OverlayValues[50] = d50
				ps537.OverlayValues[51] = d51
				ps537.OverlayValues[54] = d54
				ps537.OverlayValues[55] = d55
				ps537.OverlayValues[56] = d56
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
				ps537.OverlayValues[138] = d138
				ps537.OverlayValues[139] = d139
				ps537.OverlayValues[142] = d142
				ps537.OverlayValues[227] = d227
				ps537.OverlayValues[228] = d228
				ps537.OverlayValues[229] = d229
				ps537.OverlayValues[230] = d230
				ps537.OverlayValues[232] = d232
				ps537.OverlayValues[233] = d233
				ps537.OverlayValues[234] = d234
				ps537.OverlayValues[235] = d235
				ps537.OverlayValues[236] = d236
				ps537.OverlayValues[237] = d237
				ps537.OverlayValues[238] = d238
				ps537.OverlayValues[239] = d239
				ps537.OverlayValues[241] = d241
				ps537.OverlayValues[243] = d243
				ps537.OverlayValues[244] = d244
				ps537.OverlayValues[245] = d245
				ps537.OverlayValues[246] = d246
				ps537.OverlayValues[247] = d247
				ps537.OverlayValues[250] = d250
				ps537.OverlayValues[352] = d352
				ps537.OverlayValues[353] = d353
				ps537.OverlayValues[354] = d354
				ps537.OverlayValues[355] = d355
				ps537.OverlayValues[356] = d356
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
				ps537.OverlayValues[385] = d385
				ps537.OverlayValues[386] = d386
				ps537.OverlayValues[526] = d526
				ps537.OverlayValues[527] = d527
				ps537.OverlayValues[528] = d528
				ps537.OverlayValues[530] = d530
				ps537.OverlayValues[531] = d531
				ps537.OverlayValues[532] = d532
				ps537.OverlayValues[533] = d533
				ps537.OverlayValues[534] = d534
				ps537.OverlayValues[535] = d535
				ps537.OverlayValues[536] = d536
				ps537.PhiValues = make([]scm.JITValueDesc, 1)
				d538 = d10
				ps537.PhiValues[0] = d538
				return bbs[2].RenderPS(ps537)
			}
			if ps.General {
			}
			ps539 := scm.PhiState{General: ps.General}
			ps539.OverlayValues = make([]scm.JITValueDesc, 539)
			ps539.OverlayValues[3] = d3
			ps539.OverlayValues[4] = d4
			ps539.OverlayValues[5] = d5
			ps539.OverlayValues[6] = d6
			ps539.OverlayValues[7] = d7
			ps539.OverlayValues[8] = d8
			ps539.OverlayValues[9] = d9
			ps539.OverlayValues[10] = d10
			ps539.OverlayValues[11] = d11
			ps539.OverlayValues[12] = d12
			ps539.OverlayValues[13] = d13
			ps539.OverlayValues[14] = d14
			ps539.OverlayValues[15] = d15
			ps539.OverlayValues[16] = d16
			ps539.OverlayValues[17] = d17
			ps539.OverlayValues[19] = d19
			ps539.OverlayValues[20] = d20
			ps539.OverlayValues[21] = d21
			ps539.OverlayValues[22] = d22
			ps539.OverlayValues[23] = d23
			ps539.OverlayValues[24] = d24
			ps539.OverlayValues[25] = d25
			ps539.OverlayValues[26] = d26
			ps539.OverlayValues[27] = d27
			ps539.OverlayValues[28] = d28
			ps539.OverlayValues[29] = d29
			ps539.OverlayValues[30] = d30
			ps539.OverlayValues[31] = d31
			ps539.OverlayValues[32] = d32
			ps539.OverlayValues[33] = d33
			ps539.OverlayValues[34] = d34
			ps539.OverlayValues[35] = d35
			ps539.OverlayValues[36] = d36
			ps539.OverlayValues[37] = d37
			ps539.OverlayValues[38] = d38
			ps539.OverlayValues[39] = d39
			ps539.OverlayValues[40] = d40
			ps539.OverlayValues[41] = d41
			ps539.OverlayValues[42] = d42
			ps539.OverlayValues[43] = d43
			ps539.OverlayValues[44] = d44
			ps539.OverlayValues[45] = d45
			ps539.OverlayValues[46] = d46
			ps539.OverlayValues[47] = d47
			ps539.OverlayValues[48] = d48
			ps539.OverlayValues[49] = d49
			ps539.OverlayValues[50] = d50
			ps539.OverlayValues[51] = d51
			ps539.OverlayValues[54] = d54
			ps539.OverlayValues[55] = d55
			ps539.OverlayValues[56] = d56
			ps539.OverlayValues[111] = d111
			ps539.OverlayValues[112] = d112
			ps539.OverlayValues[113] = d113
			ps539.OverlayValues[114] = d114
			ps539.OverlayValues[115] = d115
			ps539.OverlayValues[116] = d116
			ps539.OverlayValues[117] = d117
			ps539.OverlayValues[118] = d118
			ps539.OverlayValues[119] = d119
			ps539.OverlayValues[120] = d120
			ps539.OverlayValues[121] = d121
			ps539.OverlayValues[122] = d122
			ps539.OverlayValues[123] = d123
			ps539.OverlayValues[124] = d124
			ps539.OverlayValues[125] = d125
			ps539.OverlayValues[126] = d126
			ps539.OverlayValues[127] = d127
			ps539.OverlayValues[128] = d128
			ps539.OverlayValues[129] = d129
			ps539.OverlayValues[130] = d130
			ps539.OverlayValues[131] = d131
			ps539.OverlayValues[132] = d132
			ps539.OverlayValues[133] = d133
			ps539.OverlayValues[134] = d134
			ps539.OverlayValues[135] = d135
			ps539.OverlayValues[136] = d136
			ps539.OverlayValues[137] = d137
			ps539.OverlayValues[138] = d138
			ps539.OverlayValues[139] = d139
			ps539.OverlayValues[142] = d142
			ps539.OverlayValues[227] = d227
			ps539.OverlayValues[228] = d228
			ps539.OverlayValues[229] = d229
			ps539.OverlayValues[230] = d230
			ps539.OverlayValues[232] = d232
			ps539.OverlayValues[233] = d233
			ps539.OverlayValues[234] = d234
			ps539.OverlayValues[235] = d235
			ps539.OverlayValues[236] = d236
			ps539.OverlayValues[237] = d237
			ps539.OverlayValues[238] = d238
			ps539.OverlayValues[239] = d239
			ps539.OverlayValues[241] = d241
			ps539.OverlayValues[243] = d243
			ps539.OverlayValues[244] = d244
			ps539.OverlayValues[245] = d245
			ps539.OverlayValues[246] = d246
			ps539.OverlayValues[247] = d247
			ps539.OverlayValues[250] = d250
			ps539.OverlayValues[352] = d352
			ps539.OverlayValues[353] = d353
			ps539.OverlayValues[354] = d354
			ps539.OverlayValues[355] = d355
			ps539.OverlayValues[356] = d356
			ps539.OverlayValues[358] = d358
			ps539.OverlayValues[359] = d359
			ps539.OverlayValues[360] = d360
			ps539.OverlayValues[361] = d361
			ps539.OverlayValues[362] = d362
			ps539.OverlayValues[363] = d363
			ps539.OverlayValues[364] = d364
			ps539.OverlayValues[365] = d365
			ps539.OverlayValues[366] = d366
			ps539.OverlayValues[367] = d367
			ps539.OverlayValues[368] = d368
			ps539.OverlayValues[369] = d369
			ps539.OverlayValues[370] = d370
			ps539.OverlayValues[371] = d371
			ps539.OverlayValues[372] = d372
			ps539.OverlayValues[373] = d373
			ps539.OverlayValues[374] = d374
			ps539.OverlayValues[375] = d375
			ps539.OverlayValues[376] = d376
			ps539.OverlayValues[377] = d377
			ps539.OverlayValues[378] = d378
			ps539.OverlayValues[379] = d379
			ps539.OverlayValues[380] = d380
			ps539.OverlayValues[381] = d381
			ps539.OverlayValues[382] = d382
			ps539.OverlayValues[383] = d383
			ps539.OverlayValues[384] = d384
			ps539.OverlayValues[385] = d385
			ps539.OverlayValues[386] = d386
			ps539.OverlayValues[526] = d526
			ps539.OverlayValues[527] = d527
			ps539.OverlayValues[528] = d528
			ps539.OverlayValues[530] = d530
			ps539.OverlayValues[531] = d531
			ps539.OverlayValues[532] = d532
			ps539.OverlayValues[533] = d533
			ps539.OverlayValues[534] = d534
			ps539.OverlayValues[535] = d535
			ps539.OverlayValues[536] = d536
			ps539.OverlayValues[538] = d538
			return bbs[10].RenderPS(ps539)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d540 := ps.PhiValues[0]
				ctx.EnsureDesc(&d540)
				ctx.EmitStoreToStack(d540, int32(bbs[8].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d541 := ps.PhiValues[1]
				ctx.EnsureDesc(&d541)
				ctx.EmitStoreToStack(d541, int32(bbs[8].PhiBase)+int32(16))
			}
			ps.General = true
			return bbs[8].RenderPS(ps)
		}
		lbl26 := ctx.ReserveLabel()
		lbl27 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d534.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl26)
		ctx.EmitJmp(lbl27)
		ctx.MarkLabel(lbl26)
		ctx.SyncDesc(&d10)
		if d10.Loc == scm.LocReg {
			ctx.ProtectReg(d10.Reg)
		} else if d10.Loc == scm.LocRegPair {
			ctx.ProtectReg(d10.Reg)
			ctx.ProtectReg(d10.Reg2)
		}
		d542 = d10
		if d542.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d542)
		d543 = d542
		if d543.Loc == scm.LocImm {
			d543 = scm.JITValueDesc{Loc: scm.LocImm, Type: d543.Type, Imm: scm.NewInt(int64(uint64(d543.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d543.Reg, 32)
			ctx.EmitShrRegImm8(d543.Reg, 32)
		}
		ctx.EmitStoreToStack(d543, int32(bbs[2].PhiBase)+int32(0))
		if d10.Loc == scm.LocReg {
			ctx.UnprotectReg(d10.Reg)
		} else if d10.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d10.Reg)
			ctx.UnprotectReg(d10.Reg2)
		}
		ctx.EmitJmp(lbl3)
		ctx.MarkLabel(lbl27)
		ctx.EmitJmp(lbl11)
		ps544 := scm.PhiState{General: true}
		ps544.OverlayValues = make([]scm.JITValueDesc, 544)
		ps544.OverlayValues[3] = d3
		ps544.OverlayValues[4] = d4
		ps544.OverlayValues[5] = d5
		ps544.OverlayValues[6] = d6
		ps544.OverlayValues[7] = d7
		ps544.OverlayValues[8] = d8
		ps544.OverlayValues[9] = d9
		ps544.OverlayValues[10] = d10
		ps544.OverlayValues[11] = d11
		ps544.OverlayValues[12] = d12
		ps544.OverlayValues[13] = d13
		ps544.OverlayValues[14] = d14
		ps544.OverlayValues[15] = d15
		ps544.OverlayValues[16] = d16
		ps544.OverlayValues[17] = d17
		ps544.OverlayValues[19] = d19
		ps544.OverlayValues[20] = d20
		ps544.OverlayValues[21] = d21
		ps544.OverlayValues[22] = d22
		ps544.OverlayValues[23] = d23
		ps544.OverlayValues[24] = d24
		ps544.OverlayValues[25] = d25
		ps544.OverlayValues[26] = d26
		ps544.OverlayValues[27] = d27
		ps544.OverlayValues[28] = d28
		ps544.OverlayValues[29] = d29
		ps544.OverlayValues[30] = d30
		ps544.OverlayValues[31] = d31
		ps544.OverlayValues[32] = d32
		ps544.OverlayValues[33] = d33
		ps544.OverlayValues[34] = d34
		ps544.OverlayValues[35] = d35
		ps544.OverlayValues[36] = d36
		ps544.OverlayValues[37] = d37
		ps544.OverlayValues[38] = d38
		ps544.OverlayValues[39] = d39
		ps544.OverlayValues[40] = d40
		ps544.OverlayValues[41] = d41
		ps544.OverlayValues[42] = d42
		ps544.OverlayValues[43] = d43
		ps544.OverlayValues[44] = d44
		ps544.OverlayValues[45] = d45
		ps544.OverlayValues[46] = d46
		ps544.OverlayValues[47] = d47
		ps544.OverlayValues[48] = d48
		ps544.OverlayValues[49] = d49
		ps544.OverlayValues[50] = d50
		ps544.OverlayValues[51] = d51
		ps544.OverlayValues[54] = d54
		ps544.OverlayValues[55] = d55
		ps544.OverlayValues[56] = d56
		ps544.OverlayValues[111] = d111
		ps544.OverlayValues[112] = d112
		ps544.OverlayValues[113] = d113
		ps544.OverlayValues[114] = d114
		ps544.OverlayValues[115] = d115
		ps544.OverlayValues[116] = d116
		ps544.OverlayValues[117] = d117
		ps544.OverlayValues[118] = d118
		ps544.OverlayValues[119] = d119
		ps544.OverlayValues[120] = d120
		ps544.OverlayValues[121] = d121
		ps544.OverlayValues[122] = d122
		ps544.OverlayValues[123] = d123
		ps544.OverlayValues[124] = d124
		ps544.OverlayValues[125] = d125
		ps544.OverlayValues[126] = d126
		ps544.OverlayValues[127] = d127
		ps544.OverlayValues[128] = d128
		ps544.OverlayValues[129] = d129
		ps544.OverlayValues[130] = d130
		ps544.OverlayValues[131] = d131
		ps544.OverlayValues[132] = d132
		ps544.OverlayValues[133] = d133
		ps544.OverlayValues[134] = d134
		ps544.OverlayValues[135] = d135
		ps544.OverlayValues[136] = d136
		ps544.OverlayValues[137] = d137
		ps544.OverlayValues[138] = d138
		ps544.OverlayValues[139] = d139
		ps544.OverlayValues[142] = d142
		ps544.OverlayValues[227] = d227
		ps544.OverlayValues[228] = d228
		ps544.OverlayValues[229] = d229
		ps544.OverlayValues[230] = d230
		ps544.OverlayValues[232] = d232
		ps544.OverlayValues[233] = d233
		ps544.OverlayValues[234] = d234
		ps544.OverlayValues[235] = d235
		ps544.OverlayValues[236] = d236
		ps544.OverlayValues[237] = d237
		ps544.OverlayValues[238] = d238
		ps544.OverlayValues[239] = d239
		ps544.OverlayValues[241] = d241
		ps544.OverlayValues[243] = d243
		ps544.OverlayValues[244] = d244
		ps544.OverlayValues[245] = d245
		ps544.OverlayValues[246] = d246
		ps544.OverlayValues[247] = d247
		ps544.OverlayValues[250] = d250
		ps544.OverlayValues[352] = d352
		ps544.OverlayValues[353] = d353
		ps544.OverlayValues[354] = d354
		ps544.OverlayValues[355] = d355
		ps544.OverlayValues[356] = d356
		ps544.OverlayValues[358] = d358
		ps544.OverlayValues[359] = d359
		ps544.OverlayValues[360] = d360
		ps544.OverlayValues[361] = d361
		ps544.OverlayValues[362] = d362
		ps544.OverlayValues[363] = d363
		ps544.OverlayValues[364] = d364
		ps544.OverlayValues[365] = d365
		ps544.OverlayValues[366] = d366
		ps544.OverlayValues[367] = d367
		ps544.OverlayValues[368] = d368
		ps544.OverlayValues[369] = d369
		ps544.OverlayValues[370] = d370
		ps544.OverlayValues[371] = d371
		ps544.OverlayValues[372] = d372
		ps544.OverlayValues[373] = d373
		ps544.OverlayValues[374] = d374
		ps544.OverlayValues[375] = d375
		ps544.OverlayValues[376] = d376
		ps544.OverlayValues[377] = d377
		ps544.OverlayValues[378] = d378
		ps544.OverlayValues[379] = d379
		ps544.OverlayValues[380] = d380
		ps544.OverlayValues[381] = d381
		ps544.OverlayValues[382] = d382
		ps544.OverlayValues[383] = d383
		ps544.OverlayValues[384] = d384
		ps544.OverlayValues[385] = d385
		ps544.OverlayValues[386] = d386
		ps544.OverlayValues[526] = d526
		ps544.OverlayValues[527] = d527
		ps544.OverlayValues[528] = d528
		ps544.OverlayValues[530] = d530
		ps544.OverlayValues[531] = d531
		ps544.OverlayValues[532] = d532
		ps544.OverlayValues[533] = d533
		ps544.OverlayValues[534] = d534
		ps544.OverlayValues[535] = d535
		ps544.OverlayValues[536] = d536
		ps544.OverlayValues[538] = d538
		ps544.OverlayValues[540] = d540
		ps544.OverlayValues[541] = d541
		ps544.OverlayValues[542] = d542
		ps544.OverlayValues[543] = d543
		ps544.PhiValues = make([]scm.JITValueDesc, 1)
		d546 = d10
		ps544.PhiValues[0] = d546
		ps545 := scm.PhiState{General: true}
		ps545.OverlayValues = make([]scm.JITValueDesc, 547)
		ps545.OverlayValues[3] = d3
		ps545.OverlayValues[4] = d4
		ps545.OverlayValues[5] = d5
		ps545.OverlayValues[6] = d6
		ps545.OverlayValues[7] = d7
		ps545.OverlayValues[8] = d8
		ps545.OverlayValues[9] = d9
		ps545.OverlayValues[10] = d10
		ps545.OverlayValues[11] = d11
		ps545.OverlayValues[12] = d12
		ps545.OverlayValues[13] = d13
		ps545.OverlayValues[14] = d14
		ps545.OverlayValues[15] = d15
		ps545.OverlayValues[16] = d16
		ps545.OverlayValues[17] = d17
		ps545.OverlayValues[19] = d19
		ps545.OverlayValues[20] = d20
		ps545.OverlayValues[21] = d21
		ps545.OverlayValues[22] = d22
		ps545.OverlayValues[23] = d23
		ps545.OverlayValues[24] = d24
		ps545.OverlayValues[25] = d25
		ps545.OverlayValues[26] = d26
		ps545.OverlayValues[27] = d27
		ps545.OverlayValues[28] = d28
		ps545.OverlayValues[29] = d29
		ps545.OverlayValues[30] = d30
		ps545.OverlayValues[31] = d31
		ps545.OverlayValues[32] = d32
		ps545.OverlayValues[33] = d33
		ps545.OverlayValues[34] = d34
		ps545.OverlayValues[35] = d35
		ps545.OverlayValues[36] = d36
		ps545.OverlayValues[37] = d37
		ps545.OverlayValues[38] = d38
		ps545.OverlayValues[39] = d39
		ps545.OverlayValues[40] = d40
		ps545.OverlayValues[41] = d41
		ps545.OverlayValues[42] = d42
		ps545.OverlayValues[43] = d43
		ps545.OverlayValues[44] = d44
		ps545.OverlayValues[45] = d45
		ps545.OverlayValues[46] = d46
		ps545.OverlayValues[47] = d47
		ps545.OverlayValues[48] = d48
		ps545.OverlayValues[49] = d49
		ps545.OverlayValues[50] = d50
		ps545.OverlayValues[51] = d51
		ps545.OverlayValues[54] = d54
		ps545.OverlayValues[55] = d55
		ps545.OverlayValues[56] = d56
		ps545.OverlayValues[111] = d111
		ps545.OverlayValues[112] = d112
		ps545.OverlayValues[113] = d113
		ps545.OverlayValues[114] = d114
		ps545.OverlayValues[115] = d115
		ps545.OverlayValues[116] = d116
		ps545.OverlayValues[117] = d117
		ps545.OverlayValues[118] = d118
		ps545.OverlayValues[119] = d119
		ps545.OverlayValues[120] = d120
		ps545.OverlayValues[121] = d121
		ps545.OverlayValues[122] = d122
		ps545.OverlayValues[123] = d123
		ps545.OverlayValues[124] = d124
		ps545.OverlayValues[125] = d125
		ps545.OverlayValues[126] = d126
		ps545.OverlayValues[127] = d127
		ps545.OverlayValues[128] = d128
		ps545.OverlayValues[129] = d129
		ps545.OverlayValues[130] = d130
		ps545.OverlayValues[131] = d131
		ps545.OverlayValues[132] = d132
		ps545.OverlayValues[133] = d133
		ps545.OverlayValues[134] = d134
		ps545.OverlayValues[135] = d135
		ps545.OverlayValues[136] = d136
		ps545.OverlayValues[137] = d137
		ps545.OverlayValues[138] = d138
		ps545.OverlayValues[139] = d139
		ps545.OverlayValues[142] = d142
		ps545.OverlayValues[227] = d227
		ps545.OverlayValues[228] = d228
		ps545.OverlayValues[229] = d229
		ps545.OverlayValues[230] = d230
		ps545.OverlayValues[232] = d232
		ps545.OverlayValues[233] = d233
		ps545.OverlayValues[234] = d234
		ps545.OverlayValues[235] = d235
		ps545.OverlayValues[236] = d236
		ps545.OverlayValues[237] = d237
		ps545.OverlayValues[238] = d238
		ps545.OverlayValues[239] = d239
		ps545.OverlayValues[241] = d241
		ps545.OverlayValues[243] = d243
		ps545.OverlayValues[244] = d244
		ps545.OverlayValues[245] = d245
		ps545.OverlayValues[246] = d246
		ps545.OverlayValues[247] = d247
		ps545.OverlayValues[250] = d250
		ps545.OverlayValues[352] = d352
		ps545.OverlayValues[353] = d353
		ps545.OverlayValues[354] = d354
		ps545.OverlayValues[355] = d355
		ps545.OverlayValues[356] = d356
		ps545.OverlayValues[358] = d358
		ps545.OverlayValues[359] = d359
		ps545.OverlayValues[360] = d360
		ps545.OverlayValues[361] = d361
		ps545.OverlayValues[362] = d362
		ps545.OverlayValues[363] = d363
		ps545.OverlayValues[364] = d364
		ps545.OverlayValues[365] = d365
		ps545.OverlayValues[366] = d366
		ps545.OverlayValues[367] = d367
		ps545.OverlayValues[368] = d368
		ps545.OverlayValues[369] = d369
		ps545.OverlayValues[370] = d370
		ps545.OverlayValues[371] = d371
		ps545.OverlayValues[372] = d372
		ps545.OverlayValues[373] = d373
		ps545.OverlayValues[374] = d374
		ps545.OverlayValues[375] = d375
		ps545.OverlayValues[376] = d376
		ps545.OverlayValues[377] = d377
		ps545.OverlayValues[378] = d378
		ps545.OverlayValues[379] = d379
		ps545.OverlayValues[380] = d380
		ps545.OverlayValues[381] = d381
		ps545.OverlayValues[382] = d382
		ps545.OverlayValues[383] = d383
		ps545.OverlayValues[384] = d384
		ps545.OverlayValues[385] = d385
		ps545.OverlayValues[386] = d386
		ps545.OverlayValues[526] = d526
		ps545.OverlayValues[527] = d527
		ps545.OverlayValues[528] = d528
		ps545.OverlayValues[530] = d530
		ps545.OverlayValues[531] = d531
		ps545.OverlayValues[532] = d532
		ps545.OverlayValues[533] = d533
		ps545.OverlayValues[534] = d534
		ps545.OverlayValues[535] = d535
		ps545.OverlayValues[536] = d536
		ps545.OverlayValues[538] = d538
		ps545.OverlayValues[540] = d540
		ps545.OverlayValues[541] = d541
		ps545.OverlayValues[542] = d542
		ps545.OverlayValues[543] = d543
		ps545.OverlayValues[546] = d546
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
		snap560 := d16
		snap561 := d17
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
		snap593 := d50
		snap594 := d51
		snap595 := d54
		snap596 := d55
		snap597 := d56
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
		snap625 := d138
		snap626 := d139
		snap627 := d142
		snap628 := d227
		snap629 := d228
		snap630 := d229
		snap631 := d230
		snap632 := d232
		snap633 := d233
		snap634 := d234
		snap635 := d235
		snap636 := d236
		snap637 := d237
		snap638 := d238
		snap639 := d239
		snap640 := d241
		snap641 := d243
		snap642 := d244
		snap643 := d245
		snap644 := d246
		snap645 := d247
		snap646 := d250
		snap647 := d352
		snap648 := d353
		snap649 := d354
		snap650 := d355
		snap651 := d356
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
		snap679 := d385
		snap680 := d386
		snap681 := d526
		snap682 := d527
		snap683 := d528
		snap684 := d530
		snap685 := d531
		snap686 := d532
		snap687 := d533
		snap688 := d534
		snap689 := d535
		snap690 := d536
		snap691 := d538
		snap692 := d540
		snap693 := d541
		snap694 := d542
		snap695 := d543
		snap696 := d546
		alloc697 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps544)
		}
		ctx.RestoreAllocState(alloc697)
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
		d16 = snap560
		d17 = snap561
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
		d50 = snap593
		d51 = snap594
		d54 = snap595
		d55 = snap596
		d56 = snap597
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
		d138 = snap625
		d139 = snap626
		d142 = snap627
		d227 = snap628
		d228 = snap629
		d229 = snap630
		d230 = snap631
		d232 = snap632
		d233 = snap633
		d234 = snap634
		d235 = snap635
		d236 = snap636
		d237 = snap637
		d238 = snap638
		d239 = snap639
		d241 = snap640
		d243 = snap641
		d244 = snap642
		d245 = snap643
		d246 = snap644
		d247 = snap645
		d250 = snap646
		d352 = snap647
		d353 = snap648
		d354 = snap649
		d355 = snap650
		d356 = snap651
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
		d385 = snap679
		d386 = snap680
		d526 = snap681
		d527 = snap682
		d528 = snap683
		d530 = snap684
		d531 = snap685
		d532 = snap686
		d533 = snap687
		d534 = snap688
		d535 = snap689
		d536 = snap690
		d538 = snap691
		d540 = snap692
		d541 = snap693
		d542 = snap694
		d543 = snap695
		d546 = snap696
		if !bbs[10].Rendered {
			return bbs[10].RenderPS(ps545)
		}
		return result
		ctx.FreeDesc(&d533)
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
			d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0}
			ctx.BindReg(r0, &d3)
		} else {
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
			d16 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
			d17 = ps.OverlayValues[17]
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
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
			d51 = ps.OverlayValues[51]
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
		if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != scm.LocNone {
			d142 = ps.OverlayValues[142]
		}
		if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
			d227 = ps.OverlayValues[227]
		}
		if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
			d228 = ps.OverlayValues[228]
		}
		if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != scm.LocNone {
			d229 = ps.OverlayValues[229]
		}
		if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
			d230 = ps.OverlayValues[230]
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
		if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
			d238 = ps.OverlayValues[238]
		}
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
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
		if len(ps.OverlayValues) > 250 && ps.OverlayValues[250].Loc != scm.LocNone {
			d250 = ps.OverlayValues[250]
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
		if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != scm.LocNone {
			d355 = ps.OverlayValues[355]
		}
		if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != scm.LocNone {
			d356 = ps.OverlayValues[356]
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
		if len(ps.OverlayValues) > 385 && ps.OverlayValues[385].Loc != scm.LocNone {
			d385 = ps.OverlayValues[385]
		}
		if len(ps.OverlayValues) > 386 && ps.OverlayValues[386].Loc != scm.LocNone {
			d386 = ps.OverlayValues[386]
		}
		if len(ps.OverlayValues) > 526 && ps.OverlayValues[526].Loc != scm.LocNone {
			d526 = ps.OverlayValues[526]
		}
		if len(ps.OverlayValues) > 527 && ps.OverlayValues[527].Loc != scm.LocNone {
			d527 = ps.OverlayValues[527]
		}
		if len(ps.OverlayValues) > 528 && ps.OverlayValues[528].Loc != scm.LocNone {
			d528 = ps.OverlayValues[528]
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
		if len(ps.OverlayValues) > 535 && ps.OverlayValues[535].Loc != scm.LocNone {
			d535 = ps.OverlayValues[535]
		}
		if len(ps.OverlayValues) > 536 && ps.OverlayValues[536].Loc != scm.LocNone {
			d536 = ps.OverlayValues[536]
		}
		if len(ps.OverlayValues) > 538 && ps.OverlayValues[538].Loc != scm.LocNone {
			d538 = ps.OverlayValues[538]
		}
		if len(ps.OverlayValues) > 540 && ps.OverlayValues[540].Loc != scm.LocNone {
			d540 = ps.OverlayValues[540]
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
		if len(ps.OverlayValues) > 546 && ps.OverlayValues[546].Loc != scm.LocNone {
			d546 = ps.OverlayValues[546]
		}
		ctx.ReclaimUntrackedRegs()
		if ps.General {
			ctx.SyncDesc(&d7)
			if d7.Loc == scm.LocReg {
				ctx.ProtectReg(d7.Reg)
			} else if d7.Loc == scm.LocRegPair {
				ctx.ProtectReg(d7.Reg)
				ctx.ProtectReg(d7.Reg2)
			}
			ctx.SyncDesc(&d9)
			if d9.Loc == scm.LocReg {
				ctx.ProtectReg(d9.Reg)
			} else if d9.Loc == scm.LocRegPair {
				ctx.ProtectReg(d9.Reg)
				ctx.ProtectReg(d9.Reg2)
			}
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
			ctx.EmitStoreToStack(d699, int32(bbs[8].PhiBase)+int32(0))
			d700 = d9
			if d700.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d700)
			d701 = d700
			if d701.Loc == scm.LocImm {
				d701 = scm.JITValueDesc{Loc: scm.LocImm, Type: d701.Type, Imm: scm.NewInt(int64(uint64(d701.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d701.Reg, 32)
				ctx.EmitShrRegImm8(d701.Reg, 32)
			}
			ctx.EmitStoreToStack(d701, int32(bbs[8].PhiBase)+int32(16))
			if d7.Loc == scm.LocReg {
				ctx.UnprotectReg(d7.Reg)
			} else if d7.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d7.Reg)
				ctx.UnprotectReg(d7.Reg2)
			}
			if d9.Loc == scm.LocReg {
				ctx.UnprotectReg(d9.Reg)
			} else if d9.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d9.Reg)
				ctx.UnprotectReg(d9.Reg2)
			}
		}
		ps702 := scm.PhiState{General: ps.General}
		ps702.OverlayValues = make([]scm.JITValueDesc, 702)
		ps702.OverlayValues[3] = d3
		ps702.OverlayValues[4] = d4
		ps702.OverlayValues[5] = d5
		ps702.OverlayValues[6] = d6
		ps702.OverlayValues[7] = d7
		ps702.OverlayValues[8] = d8
		ps702.OverlayValues[9] = d9
		ps702.OverlayValues[10] = d10
		ps702.OverlayValues[11] = d11
		ps702.OverlayValues[12] = d12
		ps702.OverlayValues[13] = d13
		ps702.OverlayValues[14] = d14
		ps702.OverlayValues[15] = d15
		ps702.OverlayValues[16] = d16
		ps702.OverlayValues[17] = d17
		ps702.OverlayValues[19] = d19
		ps702.OverlayValues[20] = d20
		ps702.OverlayValues[21] = d21
		ps702.OverlayValues[22] = d22
		ps702.OverlayValues[23] = d23
		ps702.OverlayValues[24] = d24
		ps702.OverlayValues[25] = d25
		ps702.OverlayValues[26] = d26
		ps702.OverlayValues[27] = d27
		ps702.OverlayValues[28] = d28
		ps702.OverlayValues[29] = d29
		ps702.OverlayValues[30] = d30
		ps702.OverlayValues[31] = d31
		ps702.OverlayValues[32] = d32
		ps702.OverlayValues[33] = d33
		ps702.OverlayValues[34] = d34
		ps702.OverlayValues[35] = d35
		ps702.OverlayValues[36] = d36
		ps702.OverlayValues[37] = d37
		ps702.OverlayValues[38] = d38
		ps702.OverlayValues[39] = d39
		ps702.OverlayValues[40] = d40
		ps702.OverlayValues[41] = d41
		ps702.OverlayValues[42] = d42
		ps702.OverlayValues[43] = d43
		ps702.OverlayValues[44] = d44
		ps702.OverlayValues[45] = d45
		ps702.OverlayValues[46] = d46
		ps702.OverlayValues[47] = d47
		ps702.OverlayValues[48] = d48
		ps702.OverlayValues[49] = d49
		ps702.OverlayValues[50] = d50
		ps702.OverlayValues[51] = d51
		ps702.OverlayValues[54] = d54
		ps702.OverlayValues[55] = d55
		ps702.OverlayValues[56] = d56
		ps702.OverlayValues[111] = d111
		ps702.OverlayValues[112] = d112
		ps702.OverlayValues[113] = d113
		ps702.OverlayValues[114] = d114
		ps702.OverlayValues[115] = d115
		ps702.OverlayValues[116] = d116
		ps702.OverlayValues[117] = d117
		ps702.OverlayValues[118] = d118
		ps702.OverlayValues[119] = d119
		ps702.OverlayValues[120] = d120
		ps702.OverlayValues[121] = d121
		ps702.OverlayValues[122] = d122
		ps702.OverlayValues[123] = d123
		ps702.OverlayValues[124] = d124
		ps702.OverlayValues[125] = d125
		ps702.OverlayValues[126] = d126
		ps702.OverlayValues[127] = d127
		ps702.OverlayValues[128] = d128
		ps702.OverlayValues[129] = d129
		ps702.OverlayValues[130] = d130
		ps702.OverlayValues[131] = d131
		ps702.OverlayValues[132] = d132
		ps702.OverlayValues[133] = d133
		ps702.OverlayValues[134] = d134
		ps702.OverlayValues[135] = d135
		ps702.OverlayValues[136] = d136
		ps702.OverlayValues[137] = d137
		ps702.OverlayValues[138] = d138
		ps702.OverlayValues[139] = d139
		ps702.OverlayValues[142] = d142
		ps702.OverlayValues[227] = d227
		ps702.OverlayValues[228] = d228
		ps702.OverlayValues[229] = d229
		ps702.OverlayValues[230] = d230
		ps702.OverlayValues[232] = d232
		ps702.OverlayValues[233] = d233
		ps702.OverlayValues[234] = d234
		ps702.OverlayValues[235] = d235
		ps702.OverlayValues[236] = d236
		ps702.OverlayValues[237] = d237
		ps702.OverlayValues[238] = d238
		ps702.OverlayValues[239] = d239
		ps702.OverlayValues[241] = d241
		ps702.OverlayValues[243] = d243
		ps702.OverlayValues[244] = d244
		ps702.OverlayValues[245] = d245
		ps702.OverlayValues[246] = d246
		ps702.OverlayValues[247] = d247
		ps702.OverlayValues[250] = d250
		ps702.OverlayValues[352] = d352
		ps702.OverlayValues[353] = d353
		ps702.OverlayValues[354] = d354
		ps702.OverlayValues[355] = d355
		ps702.OverlayValues[356] = d356
		ps702.OverlayValues[358] = d358
		ps702.OverlayValues[359] = d359
		ps702.OverlayValues[360] = d360
		ps702.OverlayValues[361] = d361
		ps702.OverlayValues[362] = d362
		ps702.OverlayValues[363] = d363
		ps702.OverlayValues[364] = d364
		ps702.OverlayValues[365] = d365
		ps702.OverlayValues[366] = d366
		ps702.OverlayValues[367] = d367
		ps702.OverlayValues[368] = d368
		ps702.OverlayValues[369] = d369
		ps702.OverlayValues[370] = d370
		ps702.OverlayValues[371] = d371
		ps702.OverlayValues[372] = d372
		ps702.OverlayValues[373] = d373
		ps702.OverlayValues[374] = d374
		ps702.OverlayValues[375] = d375
		ps702.OverlayValues[376] = d376
		ps702.OverlayValues[377] = d377
		ps702.OverlayValues[378] = d378
		ps702.OverlayValues[379] = d379
		ps702.OverlayValues[380] = d380
		ps702.OverlayValues[381] = d381
		ps702.OverlayValues[382] = d382
		ps702.OverlayValues[383] = d383
		ps702.OverlayValues[384] = d384
		ps702.OverlayValues[385] = d385
		ps702.OverlayValues[386] = d386
		ps702.OverlayValues[526] = d526
		ps702.OverlayValues[527] = d527
		ps702.OverlayValues[528] = d528
		ps702.OverlayValues[530] = d530
		ps702.OverlayValues[531] = d531
		ps702.OverlayValues[532] = d532
		ps702.OverlayValues[533] = d533
		ps702.OverlayValues[534] = d534
		ps702.OverlayValues[535] = d535
		ps702.OverlayValues[536] = d536
		ps702.OverlayValues[538] = d538
		ps702.OverlayValues[540] = d540
		ps702.OverlayValues[541] = d541
		ps702.OverlayValues[542] = d542
		ps702.OverlayValues[543] = d543
		ps702.OverlayValues[546] = d546
		ps702.OverlayValues[698] = d698
		ps702.OverlayValues[699] = d699
		ps702.OverlayValues[700] = d700
		ps702.OverlayValues[701] = d701
		ps702.PhiValues = make([]scm.JITValueDesc, 2)
		d703 = d7
		ps702.PhiValues[0] = d703
		d704 = d9
		ps702.PhiValues[1] = d704
		if ps702.General && bbs[8].Rendered {
			ctx.EmitJmp(lbl9)
			return result
		}
		return bbs[8].RenderPS(ps702)
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
			d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0}
			ctx.BindReg(r0, &d3)
		} else {
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
			d16 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
			d17 = ps.OverlayValues[17]
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
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
			d51 = ps.OverlayValues[51]
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
		if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != scm.LocNone {
			d142 = ps.OverlayValues[142]
		}
		if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
			d227 = ps.OverlayValues[227]
		}
		if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
			d228 = ps.OverlayValues[228]
		}
		if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != scm.LocNone {
			d229 = ps.OverlayValues[229]
		}
		if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
			d230 = ps.OverlayValues[230]
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
		if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
			d238 = ps.OverlayValues[238]
		}
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
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
		if len(ps.OverlayValues) > 250 && ps.OverlayValues[250].Loc != scm.LocNone {
			d250 = ps.OverlayValues[250]
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
		if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != scm.LocNone {
			d355 = ps.OverlayValues[355]
		}
		if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != scm.LocNone {
			d356 = ps.OverlayValues[356]
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
		if len(ps.OverlayValues) > 385 && ps.OverlayValues[385].Loc != scm.LocNone {
			d385 = ps.OverlayValues[385]
		}
		if len(ps.OverlayValues) > 386 && ps.OverlayValues[386].Loc != scm.LocNone {
			d386 = ps.OverlayValues[386]
		}
		if len(ps.OverlayValues) > 526 && ps.OverlayValues[526].Loc != scm.LocNone {
			d526 = ps.OverlayValues[526]
		}
		if len(ps.OverlayValues) > 527 && ps.OverlayValues[527].Loc != scm.LocNone {
			d527 = ps.OverlayValues[527]
		}
		if len(ps.OverlayValues) > 528 && ps.OverlayValues[528].Loc != scm.LocNone {
			d528 = ps.OverlayValues[528]
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
		if len(ps.OverlayValues) > 535 && ps.OverlayValues[535].Loc != scm.LocNone {
			d535 = ps.OverlayValues[535]
		}
		if len(ps.OverlayValues) > 536 && ps.OverlayValues[536].Loc != scm.LocNone {
			d536 = ps.OverlayValues[536]
		}
		if len(ps.OverlayValues) > 538 && ps.OverlayValues[538].Loc != scm.LocNone {
			d538 = ps.OverlayValues[538]
		}
		if len(ps.OverlayValues) > 540 && ps.OverlayValues[540].Loc != scm.LocNone {
			d540 = ps.OverlayValues[540]
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
		if len(ps.OverlayValues) > 546 && ps.OverlayValues[546].Loc != scm.LocNone {
			d546 = ps.OverlayValues[546]
		}
		if len(ps.OverlayValues) > 698 && ps.OverlayValues[698].Loc != scm.LocNone {
			d698 = ps.OverlayValues[698]
		}
		if len(ps.OverlayValues) > 699 && ps.OverlayValues[699].Loc != scm.LocNone {
			d699 = ps.OverlayValues[699]
		}
		if len(ps.OverlayValues) > 700 && ps.OverlayValues[700].Loc != scm.LocNone {
			d700 = ps.OverlayValues[700]
		}
		if len(ps.OverlayValues) > 701 && ps.OverlayValues[701].Loc != scm.LocNone {
			d701 = ps.OverlayValues[701]
		}
		if len(ps.OverlayValues) > 703 && ps.OverlayValues[703].Loc != scm.LocNone {
			d703 = ps.OverlayValues[703]
		}
		if len(ps.OverlayValues) > 704 && ps.OverlayValues[704].Loc != scm.LocNone {
			d704 = ps.OverlayValues[704]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d10)
		ctx.EnsureDesc(&d11)
		ctx.EnsureDescsTogether(&d10, &d11)
		var d705 scm.JITValueDesc
		if d10.Loc == scm.LocImm && d11.Loc == scm.LocImm {
			d705 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d10.Imm.Int() + d11.Imm.Int())}
		} else if d11.Loc == scm.LocImm && d11.Imm.Int() == 0 {
			r118 := ctx.AllocRegExcept(d10.Reg)
			ctx.EmitMovRegReg(r118, d10.Reg)
			d705 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r118}
			ctx.BindReg(r118, &d705)
		} else if d10.Loc == scm.LocImm && d10.Imm.Int() == 0 {
			d705 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d11.Reg}
			ctx.BindReg(d11.Reg, &d705)
		} else if d10.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d11.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d10.Imm.Int()))
			ctx.EmitAddInt64(scratch, d11.Reg)
			d705 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d705)
		} else if d11.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d10.Reg)
			ctx.EmitMovRegReg(scratch, d10.Reg)
			if d11.Imm.Int() >= -2147483648 && d11.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d11.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d11.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d705 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d705)
		} else {
			r119 := ctx.AllocRegExcept(d10.Reg, d11.Reg)
			ctx.EmitMovRegReg(r119, d10.Reg)
			ctx.EmitAddInt64(r119, d11.Reg)
			d705 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
			ctx.BindReg(r119, &d705)
		}
		if d705.Loc == scm.LocImm {
			d705 = scm.JITValueDesc{Loc: scm.LocImm, Type: d705.Type, Imm: scm.NewInt(int64(uint64(d705.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d705.Reg, 32)
			ctx.EmitShrRegImm8(d705.Reg, 32)
		}
		if d705.Loc == scm.LocReg && d10.Loc == scm.LocReg && d705.Reg == d10.Reg {
			ctx.TransferReg(d10.Reg)
			d10.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d705)
		var d706 scm.JITValueDesc
		if d705.Loc == scm.LocImm {
			d706 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d705.Imm.Int() / 2)}
		} else {
			r120 := ctx.AllocRegExcept(d705.Reg)
			ctx.EmitMovRegReg(r120, d705.Reg)
			ctx.EmitShrRegImm8(r120, 1)
			d706 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r120}
			ctx.BindReg(r120, &d706)
		}
		if d706.Loc == scm.LocImm {
			d706 = scm.JITValueDesc{Loc: scm.LocImm, Type: d706.Type, Imm: scm.NewInt(int64(uint64(d706.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d706.Reg, 32)
			ctx.EmitShrRegImm8(d706.Reg, 32)
		}
		if d706.Loc == scm.LocReg && d705.Loc == scm.LocReg && d706.Reg == d705.Reg {
			ctx.TransferReg(d705.Reg)
			d705.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d705)
		if ps.General {
			ctx.SyncDesc(&d10)
			if d10.Loc == scm.LocReg {
				ctx.ProtectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.ProtectReg(d10.Reg)
				ctx.ProtectReg(d10.Reg2)
			}
			ctx.SyncDesc(&d11)
			if d11.Loc == scm.LocReg {
				ctx.ProtectReg(d11.Reg)
			} else if d11.Loc == scm.LocRegPair {
				ctx.ProtectReg(d11.Reg)
				ctx.ProtectReg(d11.Reg2)
			}
			ctx.SyncDesc(&d706)
			if d706.Loc == scm.LocReg {
				ctx.ProtectReg(d706.Reg)
			} else if d706.Loc == scm.LocRegPair {
				ctx.ProtectReg(d706.Reg)
				ctx.ProtectReg(d706.Reg2)
			}
			d707 = d706
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
			if phiHomeOK2 {
				ctx.EmitMovToReg(r0, d708)
			} else {
				ctx.EmitStoreToStack(d708, int32(bbs[1].PhiBase)+int32(0))
			}
			d709 = d10
			if d709.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d709)
			d710 = d709
			if d710.Loc == scm.LocImm {
				d710 = scm.JITValueDesc{Loc: scm.LocImm, Type: d710.Type, Imm: scm.NewInt(int64(uint64(d710.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d710.Reg, 32)
				ctx.EmitShrRegImm8(d710.Reg, 32)
			}
			ctx.EmitStoreToStack(d710, int32(bbs[1].PhiBase)+int32(16))
			d711 = d11
			if d711.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d711)
			d712 = d711
			if d712.Loc == scm.LocImm {
				d712 = scm.JITValueDesc{Loc: scm.LocImm, Type: d712.Type, Imm: scm.NewInt(int64(uint64(d712.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d712.Reg, 32)
				ctx.EmitShrRegImm8(d712.Reg, 32)
			}
			ctx.EmitStoreToStack(d712, int32(bbs[1].PhiBase)+int32(32))
			if d10.Loc == scm.LocReg {
				ctx.UnprotectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d10.Reg)
				ctx.UnprotectReg(d10.Reg2)
			}
			if d11.Loc == scm.LocReg {
				ctx.UnprotectReg(d11.Reg)
			} else if d11.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d11.Reg)
				ctx.UnprotectReg(d11.Reg2)
			}
			if d706.Loc == scm.LocReg {
				ctx.UnprotectReg(d706.Reg)
			} else if d706.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d706.Reg)
				ctx.UnprotectReg(d706.Reg2)
			}
		}
		ps713 := scm.PhiState{General: ps.General}
		ps713.OverlayValues = make([]scm.JITValueDesc, 713)
		ps713.OverlayValues[3] = d3
		ps713.OverlayValues[4] = d4
		ps713.OverlayValues[5] = d5
		ps713.OverlayValues[6] = d6
		ps713.OverlayValues[7] = d7
		ps713.OverlayValues[8] = d8
		ps713.OverlayValues[9] = d9
		ps713.OverlayValues[10] = d10
		ps713.OverlayValues[11] = d11
		ps713.OverlayValues[12] = d12
		ps713.OverlayValues[13] = d13
		ps713.OverlayValues[14] = d14
		ps713.OverlayValues[15] = d15
		ps713.OverlayValues[16] = d16
		ps713.OverlayValues[17] = d17
		ps713.OverlayValues[19] = d19
		ps713.OverlayValues[20] = d20
		ps713.OverlayValues[21] = d21
		ps713.OverlayValues[22] = d22
		ps713.OverlayValues[23] = d23
		ps713.OverlayValues[24] = d24
		ps713.OverlayValues[25] = d25
		ps713.OverlayValues[26] = d26
		ps713.OverlayValues[27] = d27
		ps713.OverlayValues[28] = d28
		ps713.OverlayValues[29] = d29
		ps713.OverlayValues[30] = d30
		ps713.OverlayValues[31] = d31
		ps713.OverlayValues[32] = d32
		ps713.OverlayValues[33] = d33
		ps713.OverlayValues[34] = d34
		ps713.OverlayValues[35] = d35
		ps713.OverlayValues[36] = d36
		ps713.OverlayValues[37] = d37
		ps713.OverlayValues[38] = d38
		ps713.OverlayValues[39] = d39
		ps713.OverlayValues[40] = d40
		ps713.OverlayValues[41] = d41
		ps713.OverlayValues[42] = d42
		ps713.OverlayValues[43] = d43
		ps713.OverlayValues[44] = d44
		ps713.OverlayValues[45] = d45
		ps713.OverlayValues[46] = d46
		ps713.OverlayValues[47] = d47
		ps713.OverlayValues[48] = d48
		ps713.OverlayValues[49] = d49
		ps713.OverlayValues[50] = d50
		ps713.OverlayValues[51] = d51
		ps713.OverlayValues[54] = d54
		ps713.OverlayValues[55] = d55
		ps713.OverlayValues[56] = d56
		ps713.OverlayValues[111] = d111
		ps713.OverlayValues[112] = d112
		ps713.OverlayValues[113] = d113
		ps713.OverlayValues[114] = d114
		ps713.OverlayValues[115] = d115
		ps713.OverlayValues[116] = d116
		ps713.OverlayValues[117] = d117
		ps713.OverlayValues[118] = d118
		ps713.OverlayValues[119] = d119
		ps713.OverlayValues[120] = d120
		ps713.OverlayValues[121] = d121
		ps713.OverlayValues[122] = d122
		ps713.OverlayValues[123] = d123
		ps713.OverlayValues[124] = d124
		ps713.OverlayValues[125] = d125
		ps713.OverlayValues[126] = d126
		ps713.OverlayValues[127] = d127
		ps713.OverlayValues[128] = d128
		ps713.OverlayValues[129] = d129
		ps713.OverlayValues[130] = d130
		ps713.OverlayValues[131] = d131
		ps713.OverlayValues[132] = d132
		ps713.OverlayValues[133] = d133
		ps713.OverlayValues[134] = d134
		ps713.OverlayValues[135] = d135
		ps713.OverlayValues[136] = d136
		ps713.OverlayValues[137] = d137
		ps713.OverlayValues[138] = d138
		ps713.OverlayValues[139] = d139
		ps713.OverlayValues[142] = d142
		ps713.OverlayValues[227] = d227
		ps713.OverlayValues[228] = d228
		ps713.OverlayValues[229] = d229
		ps713.OverlayValues[230] = d230
		ps713.OverlayValues[232] = d232
		ps713.OverlayValues[233] = d233
		ps713.OverlayValues[234] = d234
		ps713.OverlayValues[235] = d235
		ps713.OverlayValues[236] = d236
		ps713.OverlayValues[237] = d237
		ps713.OverlayValues[238] = d238
		ps713.OverlayValues[239] = d239
		ps713.OverlayValues[241] = d241
		ps713.OverlayValues[243] = d243
		ps713.OverlayValues[244] = d244
		ps713.OverlayValues[245] = d245
		ps713.OverlayValues[246] = d246
		ps713.OverlayValues[247] = d247
		ps713.OverlayValues[250] = d250
		ps713.OverlayValues[352] = d352
		ps713.OverlayValues[353] = d353
		ps713.OverlayValues[354] = d354
		ps713.OverlayValues[355] = d355
		ps713.OverlayValues[356] = d356
		ps713.OverlayValues[358] = d358
		ps713.OverlayValues[359] = d359
		ps713.OverlayValues[360] = d360
		ps713.OverlayValues[361] = d361
		ps713.OverlayValues[362] = d362
		ps713.OverlayValues[363] = d363
		ps713.OverlayValues[364] = d364
		ps713.OverlayValues[365] = d365
		ps713.OverlayValues[366] = d366
		ps713.OverlayValues[367] = d367
		ps713.OverlayValues[368] = d368
		ps713.OverlayValues[369] = d369
		ps713.OverlayValues[370] = d370
		ps713.OverlayValues[371] = d371
		ps713.OverlayValues[372] = d372
		ps713.OverlayValues[373] = d373
		ps713.OverlayValues[374] = d374
		ps713.OverlayValues[375] = d375
		ps713.OverlayValues[376] = d376
		ps713.OverlayValues[377] = d377
		ps713.OverlayValues[378] = d378
		ps713.OverlayValues[379] = d379
		ps713.OverlayValues[380] = d380
		ps713.OverlayValues[381] = d381
		ps713.OverlayValues[382] = d382
		ps713.OverlayValues[383] = d383
		ps713.OverlayValues[384] = d384
		ps713.OverlayValues[385] = d385
		ps713.OverlayValues[386] = d386
		ps713.OverlayValues[526] = d526
		ps713.OverlayValues[527] = d527
		ps713.OverlayValues[528] = d528
		ps713.OverlayValues[530] = d530
		ps713.OverlayValues[531] = d531
		ps713.OverlayValues[532] = d532
		ps713.OverlayValues[533] = d533
		ps713.OverlayValues[534] = d534
		ps713.OverlayValues[535] = d535
		ps713.OverlayValues[536] = d536
		ps713.OverlayValues[538] = d538
		ps713.OverlayValues[540] = d540
		ps713.OverlayValues[541] = d541
		ps713.OverlayValues[542] = d542
		ps713.OverlayValues[543] = d543
		ps713.OverlayValues[546] = d546
		ps713.OverlayValues[698] = d698
		ps713.OverlayValues[699] = d699
		ps713.OverlayValues[700] = d700
		ps713.OverlayValues[701] = d701
		ps713.OverlayValues[703] = d703
		ps713.OverlayValues[704] = d704
		ps713.OverlayValues[705] = d705
		ps713.OverlayValues[706] = d706
		ps713.OverlayValues[707] = d707
		ps713.OverlayValues[708] = d708
		ps713.OverlayValues[709] = d709
		ps713.OverlayValues[710] = d710
		ps713.OverlayValues[711] = d711
		ps713.OverlayValues[712] = d712
		ps713.PhiValues = make([]scm.JITValueDesc, 3)
		d714 = d706
		ps713.PhiValues[0] = d714
		d715 = d10
		ps713.PhiValues[1] = d715
		d716 = d11
		ps713.PhiValues[2] = d716
		if ps713.General && bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		return bbs[1].RenderPS(ps713)
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
			d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0}
			ctx.BindReg(r0, &d3)
		} else {
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
			d16 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
			d17 = ps.OverlayValues[17]
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
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
			d51 = ps.OverlayValues[51]
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
		if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != scm.LocNone {
			d142 = ps.OverlayValues[142]
		}
		if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
			d227 = ps.OverlayValues[227]
		}
		if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
			d228 = ps.OverlayValues[228]
		}
		if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != scm.LocNone {
			d229 = ps.OverlayValues[229]
		}
		if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
			d230 = ps.OverlayValues[230]
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
		if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
			d238 = ps.OverlayValues[238]
		}
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
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
		if len(ps.OverlayValues) > 250 && ps.OverlayValues[250].Loc != scm.LocNone {
			d250 = ps.OverlayValues[250]
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
		if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != scm.LocNone {
			d355 = ps.OverlayValues[355]
		}
		if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != scm.LocNone {
			d356 = ps.OverlayValues[356]
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
		if len(ps.OverlayValues) > 385 && ps.OverlayValues[385].Loc != scm.LocNone {
			d385 = ps.OverlayValues[385]
		}
		if len(ps.OverlayValues) > 386 && ps.OverlayValues[386].Loc != scm.LocNone {
			d386 = ps.OverlayValues[386]
		}
		if len(ps.OverlayValues) > 526 && ps.OverlayValues[526].Loc != scm.LocNone {
			d526 = ps.OverlayValues[526]
		}
		if len(ps.OverlayValues) > 527 && ps.OverlayValues[527].Loc != scm.LocNone {
			d527 = ps.OverlayValues[527]
		}
		if len(ps.OverlayValues) > 528 && ps.OverlayValues[528].Loc != scm.LocNone {
			d528 = ps.OverlayValues[528]
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
		if len(ps.OverlayValues) > 535 && ps.OverlayValues[535].Loc != scm.LocNone {
			d535 = ps.OverlayValues[535]
		}
		if len(ps.OverlayValues) > 536 && ps.OverlayValues[536].Loc != scm.LocNone {
			d536 = ps.OverlayValues[536]
		}
		if len(ps.OverlayValues) > 538 && ps.OverlayValues[538].Loc != scm.LocNone {
			d538 = ps.OverlayValues[538]
		}
		if len(ps.OverlayValues) > 540 && ps.OverlayValues[540].Loc != scm.LocNone {
			d540 = ps.OverlayValues[540]
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
		if len(ps.OverlayValues) > 546 && ps.OverlayValues[546].Loc != scm.LocNone {
			d546 = ps.OverlayValues[546]
		}
		if len(ps.OverlayValues) > 698 && ps.OverlayValues[698].Loc != scm.LocNone {
			d698 = ps.OverlayValues[698]
		}
		if len(ps.OverlayValues) > 699 && ps.OverlayValues[699].Loc != scm.LocNone {
			d699 = ps.OverlayValues[699]
		}
		if len(ps.OverlayValues) > 700 && ps.OverlayValues[700].Loc != scm.LocNone {
			d700 = ps.OverlayValues[700]
		}
		if len(ps.OverlayValues) > 701 && ps.OverlayValues[701].Loc != scm.LocNone {
			d701 = ps.OverlayValues[701]
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
		if len(ps.OverlayValues) > 709 && ps.OverlayValues[709].Loc != scm.LocNone {
			d709 = ps.OverlayValues[709]
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
		if len(ps.OverlayValues) > 714 && ps.OverlayValues[714].Loc != scm.LocNone {
			d714 = ps.OverlayValues[714]
		}
		if len(ps.OverlayValues) > 715 && ps.OverlayValues[715].Loc != scm.LocNone {
			d715 = ps.OverlayValues[715]
		}
		if len(ps.OverlayValues) > 716 && ps.OverlayValues[716].Loc != scm.LocNone {
			d716 = ps.OverlayValues[716]
		}
		ctx.ReclaimUntrackedRegs()
		d717 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d718 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
		ctx.BindReg(r1, &d718)
		ctx.BindReg(r2, &d718)
		ctx.EnsureDesc(&d717)
		if d717.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d717, &d718)
		} else {
			switch d717.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d718, d717)
			case scm.TagInt:
				ctx.EmitMakeInt(d718, d717)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d718, d717)
			case scm.TagNil:
				ctx.EmitMakeNil(d718)
			default:
				ctx.EmitMovPairToResult(&d717, &d718)
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
			d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0}
			ctx.BindReg(r0, &d3)
		} else {
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
			d16 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
			d17 = ps.OverlayValues[17]
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
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
			d51 = ps.OverlayValues[51]
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
		if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != scm.LocNone {
			d142 = ps.OverlayValues[142]
		}
		if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
			d227 = ps.OverlayValues[227]
		}
		if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
			d228 = ps.OverlayValues[228]
		}
		if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != scm.LocNone {
			d229 = ps.OverlayValues[229]
		}
		if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
			d230 = ps.OverlayValues[230]
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
		if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
			d238 = ps.OverlayValues[238]
		}
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
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
		if len(ps.OverlayValues) > 250 && ps.OverlayValues[250].Loc != scm.LocNone {
			d250 = ps.OverlayValues[250]
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
		if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != scm.LocNone {
			d355 = ps.OverlayValues[355]
		}
		if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != scm.LocNone {
			d356 = ps.OverlayValues[356]
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
		if len(ps.OverlayValues) > 385 && ps.OverlayValues[385].Loc != scm.LocNone {
			d385 = ps.OverlayValues[385]
		}
		if len(ps.OverlayValues) > 386 && ps.OverlayValues[386].Loc != scm.LocNone {
			d386 = ps.OverlayValues[386]
		}
		if len(ps.OverlayValues) > 526 && ps.OverlayValues[526].Loc != scm.LocNone {
			d526 = ps.OverlayValues[526]
		}
		if len(ps.OverlayValues) > 527 && ps.OverlayValues[527].Loc != scm.LocNone {
			d527 = ps.OverlayValues[527]
		}
		if len(ps.OverlayValues) > 528 && ps.OverlayValues[528].Loc != scm.LocNone {
			d528 = ps.OverlayValues[528]
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
		if len(ps.OverlayValues) > 535 && ps.OverlayValues[535].Loc != scm.LocNone {
			d535 = ps.OverlayValues[535]
		}
		if len(ps.OverlayValues) > 536 && ps.OverlayValues[536].Loc != scm.LocNone {
			d536 = ps.OverlayValues[536]
		}
		if len(ps.OverlayValues) > 538 && ps.OverlayValues[538].Loc != scm.LocNone {
			d538 = ps.OverlayValues[538]
		}
		if len(ps.OverlayValues) > 540 && ps.OverlayValues[540].Loc != scm.LocNone {
			d540 = ps.OverlayValues[540]
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
		if len(ps.OverlayValues) > 546 && ps.OverlayValues[546].Loc != scm.LocNone {
			d546 = ps.OverlayValues[546]
		}
		if len(ps.OverlayValues) > 698 && ps.OverlayValues[698].Loc != scm.LocNone {
			d698 = ps.OverlayValues[698]
		}
		if len(ps.OverlayValues) > 699 && ps.OverlayValues[699].Loc != scm.LocNone {
			d699 = ps.OverlayValues[699]
		}
		if len(ps.OverlayValues) > 700 && ps.OverlayValues[700].Loc != scm.LocNone {
			d700 = ps.OverlayValues[700]
		}
		if len(ps.OverlayValues) > 701 && ps.OverlayValues[701].Loc != scm.LocNone {
			d701 = ps.OverlayValues[701]
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
		if len(ps.OverlayValues) > 709 && ps.OverlayValues[709].Loc != scm.LocNone {
			d709 = ps.OverlayValues[709]
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d6)
		d719 = d6
		_ = d719
		ctx.StabilizeDescForControlFlow(&d719)
		ctx.StabilizeDescForControlFlow(&d6)
		bbpos_4_0 := int32(-1)
		_ = bbpos_4_0
		lbl28 := ctx.ReserveLabel()
		_ = lbl28
		bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl28)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d719)
		ctx.EnsureDesc(&d719)
		var d720 scm.JITValueDesc
		if d719.Loc == scm.LocImm {
			d720 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d719.Imm.Int()))))}
		} else {
			r121 := ctx.AllocReg()
			ctx.EmitMovRegReg(r121, d719.Reg)
			ctx.EmitShlRegImm8(r121, 32)
			ctx.EmitShrRegImm8(r121, 32)
			d720 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r121}
			ctx.BindReg(r121, &d720)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d721 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
			r122 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r122, fieldAddr)
			d721 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r122}
			ctx.BindReg(r122, &d721)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
			r123 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r123, thisptr.Reg, off)
			d721 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r123}
			ctx.BindReg(r123, &d721)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d721)
		ctx.EnsureDesc(&d721)
		var d722 scm.JITValueDesc
		if d721.Loc == scm.LocImm {
			d722 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d721.Imm.Int()))))}
		} else {
			r124 := ctx.AllocReg()
			ctx.EmitMovRegReg(r124, d721.Reg)
			ctx.EmitShlRegImm8(r124, 56)
			ctx.EmitShrRegImm8(r124, 56)
			d722 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
			ctx.BindReg(r124, &d722)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d720)
		ctx.EnsureDesc(&d722)
		ctx.EnsureDescsTogether(&d720, &d722)
		var d723 scm.JITValueDesc
		if d720.Loc == scm.LocImm && d722.Loc == scm.LocImm {
			d723 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d720.Imm.Int() * d722.Imm.Int())}
		} else if d720.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d722.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d720.Imm.Int()))
			ctx.EmitImulInt64(scratch, d722.Reg)
			d723 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d723)
		} else if d722.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d720.Reg)
			ctx.EmitMovRegReg(scratch, d720.Reg)
			if d722.Imm.Int() >= -2147483648 && d722.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d722.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d722.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d723 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d723)
		} else {
			r125 := ctx.AllocRegExcept(d720.Reg, d722.Reg)
			ctx.EmitMovRegReg(r125, d720.Reg)
			ctx.EmitImulInt64(r125, d722.Reg)
			d723 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
			ctx.BindReg(r125, &d723)
		}
		if d723.Loc == scm.LocReg && d720.Loc == scm.LocReg && d723.Reg == d720.Reg {
			ctx.TransferReg(d720.Reg)
			d720.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d720)
		ctx.FreeDesc(&d722)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d723)
		var d724 scm.JITValueDesc
		if d723.Loc == scm.LocImm {
			d724 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d723.Imm.Int() / 64)}
		} else {
			r126 := ctx.AllocRegExcept(d723.Reg)
			ctx.EmitMovRegReg(r126, d723.Reg)
			ctx.EmitShrRegImm8(r126, 6)
			d724 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r126}
			ctx.BindReg(r126, &d724)
		}
		if d724.Loc == scm.LocReg && d723.Loc == scm.LocReg && d724.Reg == d723.Reg {
			ctx.TransferReg(d723.Reg)
			d723.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d723)
		var d725 scm.JITValueDesc
		if d723.Loc == scm.LocImm {
			d725 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d723.Imm.Int() % 64)}
		} else {
			r127 := ctx.AllocRegExcept(d723.Reg)
			ctx.EmitMovRegReg(r127, d723.Reg)
			ctx.EmitAndRegImm32(r127, 63)
			d725 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
			ctx.BindReg(r127, &d725)
		}
		if d725.Loc == scm.LocReg && d723.Loc == scm.LocReg && d725.Reg == d723.Reg {
			ctx.TransferReg(d723.Reg)
			d723.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d723)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d726 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 0
			r128 := ctx.AllocReg()
			r129 := ctx.AllocRegExcept(r128)
			r130 := ctx.AllocRegExcept(r128, r129)
			ctx.EmitMovRegMem64(r128, fieldAddr)
			ctx.EmitMovRegMem64(r129, fieldAddr+8)
			ctx.EmitMovRegMem64(r130, fieldAddr+16)
			d726 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r128, Reg2: r129, Reg3: r130}
			ctx.BindReg(r128, &d726)
			ctx.BindReg(r129, &d726)
			ctx.BindReg(r130, &d726)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 0)
			r131 := ctx.AllocReg()
			r132 := ctx.AllocRegExcept(r131)
			r133 := ctx.AllocRegExcept(r131, r132)
			ctx.EmitMovRegMem(r131, thisptr.Reg, off)
			ctx.EmitMovRegMem(r132, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r133, thisptr.Reg, off+16)
			d726 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r131, Reg2: r132, Reg3: r133}
			ctx.BindReg(r131, &d726)
			ctx.BindReg(r132, &d726)
			ctx.BindReg(r133, &d726)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d724)
		ctx.ReclaimUntrackedRegs()
		d728 = ctx.EmitSliceElementAddress(&d726, &d724, 8)
		ctx.EnsureDesc(&d728)
		ctx.EmitMovRegMem(d728.Reg, d728.Reg, 0)
		d727 = d728
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d727)
		ctx.EnsureDesc(&d725)
		var d729 scm.JITValueDesc
		if d727.Loc == scm.LocImm && d725.Loc == scm.LocImm {
			d729 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d727.Imm.Int()) << uint64(d725.Imm.Int())))}
		} else if d725.Loc == scm.LocImm {
			r134 := ctx.AllocRegExcept(d727.Reg)
			ctx.EmitMovRegReg(r134, d727.Reg)
			ctx.EmitShlRegImm8(r134, uint8(d725.Imm.Int()))
			d729 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r134}
			ctx.BindReg(r134, &d729)
		} else {
			{
				shiftSrc := d727.Reg
				r135 := ctx.AllocRegExcept(d727.Reg)
				ctx.EmitMovRegReg(r135, d727.Reg)
				shiftSrc = r135
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d725.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d725.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d725.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d729 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d729)
			}
		}
		if d729.Loc == scm.LocReg && d727.Loc == scm.LocReg && d729.Reg == d727.Reg {
			ctx.TransferReg(d727.Reg)
			d727.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d727)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d724)
		ctx.EnsureDesc(&d724)
		var d730 scm.JITValueDesc
		if d724.Loc == scm.LocImm {
			d730 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d724.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d724.Reg)
			ctx.EmitMovRegReg(scratch, d724.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d730 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d730)
		}
		if d730.Loc == scm.LocReg && d724.Loc == scm.LocReg && d730.Reg == d724.Reg {
			ctx.TransferReg(d724.Reg)
			d724.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d724)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d730)
		ctx.ReclaimUntrackedRegs()
		d732 = ctx.EmitSliceElementAddress(&d726, &d730, 8)
		ctx.EnsureDesc(&d732)
		ctx.EmitMovRegMem(d732.Reg, d732.Reg, 0)
		d731 = d732
		ctx.FreeDesc(&d730)
		ctx.ReclaimUntrackedRegs()
		d733 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d725)
		ctx.EnsureDescsTogether(&d733, &d725)
		var d734 scm.JITValueDesc
		if d733.Loc == scm.LocImm && d725.Loc == scm.LocImm {
			d734 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d733.Imm.Int() - d725.Imm.Int())}
		} else if d725.Loc == scm.LocImm && d725.Imm.Int() == 0 {
			r136 := ctx.AllocRegExcept(d733.Reg)
			ctx.EmitMovRegReg(r136, d733.Reg)
			d734 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r136}
			ctx.BindReg(r136, &d734)
		} else if d733.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d725.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d733.Imm.Int()))
			ctx.EmitSubInt64(scratch, d725.Reg)
			d734 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d734)
		} else if d725.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d733.Reg)
			ctx.EmitMovRegReg(scratch, d733.Reg)
			if d725.Imm.Int() >= -2147483648 && d725.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d725.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d725.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d734 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d734)
		} else {
			r137 := ctx.AllocRegExcept(d733.Reg, d725.Reg)
			ctx.EmitMovRegReg(r137, d733.Reg)
			ctx.EmitSubInt64(r137, d725.Reg)
			d734 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r137}
			ctx.BindReg(r137, &d734)
		}
		if d734.Loc == scm.LocReg && d733.Loc == scm.LocReg && d734.Reg == d733.Reg {
			ctx.TransferReg(d733.Reg)
			d733.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d725)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d731)
		ctx.EnsureDesc(&d734)
		var d735 scm.JITValueDesc
		if d731.Loc == scm.LocImm && d734.Loc == scm.LocImm {
			d735 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d731.Imm.Int()) >> uint64(d734.Imm.Int())))}
		} else if d734.Loc == scm.LocImm {
			r138 := ctx.AllocRegExcept(d731.Reg)
			ctx.EmitMovRegReg(r138, d731.Reg)
			ctx.EmitShrRegImm8(r138, uint8(d734.Imm.Int()))
			d735 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
			ctx.BindReg(r138, &d735)
		} else {
			{
				shiftSrc := d731.Reg
				r139 := ctx.AllocRegExcept(d731.Reg)
				ctx.EmitMovRegReg(r139, d731.Reg)
				shiftSrc = r139
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
		ctx.EnsureDesc(&d729)
		ctx.EnsureDesc(&d735)
		var d736 scm.JITValueDesc
		if d729.Loc == scm.LocImm && d735.Loc == scm.LocImm {
			d736 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d729.Imm.Int() | d735.Imm.Int())}
		} else if d729.Loc == scm.LocImm && d729.Imm.Int() == 0 {
			d736 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d735.Reg}
			ctx.BindReg(d735.Reg, &d736)
		} else if d735.Loc == scm.LocImm && d735.Imm.Int() == 0 {
			r140 := ctx.AllocRegExcept(d729.Reg)
			ctx.EmitMovRegReg(r140, d729.Reg)
			d736 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r140}
			ctx.BindReg(r140, &d736)
		} else if d729.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d735.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d729.Imm.Int()))
			ctx.EmitOrInt64(scratch, d735.Reg)
			d736 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d736)
		} else if d735.Loc == scm.LocImm {
			r141 := ctx.AllocRegExcept(d729.Reg)
			ctx.EmitMovRegReg(r141, d729.Reg)
			if d735.Imm.Int() >= -2147483648 && d735.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r141, int32(d735.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d735.Imm.Int()))
				ctx.EmitOrInt64(r141, scm.RegR11)
			}
			d736 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
			ctx.BindReg(r141, &d736)
		} else {
			r142 := ctx.AllocRegExcept(d729.Reg, d735.Reg)
			ctx.EmitMovRegReg(r142, d729.Reg)
			ctx.EmitOrInt64(r142, d735.Reg)
			d736 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r142}
			ctx.BindReg(r142, &d736)
		}
		if d736.Loc == scm.LocReg && d729.Loc == scm.LocReg && d736.Reg == d729.Reg {
			ctx.TransferReg(d729.Reg)
			d729.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d729)
		ctx.FreeDesc(&d735)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d721)
		ctx.EnsureDesc(&d721)
		var d737 scm.JITValueDesc
		if d721.Loc == scm.LocImm {
			d737 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d721.Imm.Int()))))}
		} else {
			r143 := ctx.AllocReg()
			ctx.EmitMovRegReg(r143, d721.Reg)
			ctx.EmitShlRegImm8(r143, 56)
			ctx.EmitShrRegImm8(r143, 56)
			d737 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r143}
			ctx.BindReg(r143, &d737)
		}
		ctx.ReclaimUntrackedRegs()
		d738 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d737)
		ctx.EnsureDescsTogether(&d738, &d737)
		var d739 scm.JITValueDesc
		if d738.Loc == scm.LocImm && d737.Loc == scm.LocImm {
			d739 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d738.Imm.Int() - d737.Imm.Int())}
		} else if d737.Loc == scm.LocImm && d737.Imm.Int() == 0 {
			r144 := ctx.AllocRegExcept(d738.Reg)
			ctx.EmitMovRegReg(r144, d738.Reg)
			d739 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
			ctx.BindReg(r144, &d739)
		} else if d738.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d737.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d738.Imm.Int()))
			ctx.EmitSubInt64(scratch, d737.Reg)
			d739 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d739)
		} else if d737.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d738.Reg)
			ctx.EmitMovRegReg(scratch, d738.Reg)
			if d737.Imm.Int() >= -2147483648 && d737.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d737.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d737.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d739 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d739)
		} else {
			r145 := ctx.AllocRegExcept(d738.Reg, d737.Reg)
			ctx.EmitMovRegReg(r145, d738.Reg)
			ctx.EmitSubInt64(r145, d737.Reg)
			d739 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r145}
			ctx.BindReg(r145, &d739)
		}
		if d739.Loc == scm.LocReg && d738.Loc == scm.LocReg && d739.Reg == d738.Reg {
			ctx.TransferReg(d738.Reg)
			d738.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d737)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d736)
		ctx.EnsureDesc(&d739)
		var d740 scm.JITValueDesc
		if d736.Loc == scm.LocImm && d739.Loc == scm.LocImm {
			d740 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d736.Imm.Int()) >> uint64(d739.Imm.Int())))}
		} else if d739.Loc == scm.LocImm {
			r146 := ctx.AllocRegExcept(d736.Reg)
			ctx.EmitMovRegReg(r146, d736.Reg)
			ctx.EmitShrRegImm8(r146, uint8(d739.Imm.Int()))
			d740 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r146}
			ctx.BindReg(r146, &d740)
		} else {
			{
				shiftSrc := d736.Reg
				r147 := ctx.AllocRegExcept(d736.Reg)
				ctx.EmitMovRegReg(r147, d736.Reg)
				shiftSrc = r147
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d739.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d739.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d739.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d740 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d740)
			}
		}
		if d740.Loc == scm.LocReg && d736.Loc == scm.LocReg && d740.Reg == d736.Reg {
			ctx.TransferReg(d736.Reg)
			d736.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d736)
		ctx.FreeDesc(&d739)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d740)
		ctx.EnsureDesc(&d740)
		ctx.EnsureDesc(&d740)
		var d741 scm.JITValueDesc
		if d740.Loc == scm.LocImm {
			d741 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d740.Imm.Int()))))}
		} else {
			r148 := ctx.AllocReg()
			ctx.EmitMovRegReg(r148, d740.Reg)
			d741 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r148}
			ctx.BindReg(r148, &d741)
		}
		ctx.FreeDesc(&d740)
		var d742 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 32
			r149 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r149, fieldAddr)
			d742 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r149}
			ctx.BindReg(r149, &d742)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 32)
			r150 := ctx.AllocReg()
			ctx.EmitMovRegMem(r150, thisptr.Reg, off)
			d742 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r150}
			ctx.BindReg(r150, &d742)
		}
		ctx.EnsureDesc(&d741)
		ctx.EnsureDesc(&d742)
		ctx.EnsureDescsTogether(&d741, &d742)
		var d743 scm.JITValueDesc
		if d741.Loc == scm.LocImm && d742.Loc == scm.LocImm {
			d743 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d741.Imm.Int() + d742.Imm.Int())}
		} else if d742.Loc == scm.LocImm && d742.Imm.Int() == 0 {
			r151 := ctx.AllocRegExcept(d741.Reg)
			ctx.EmitMovRegReg(r151, d741.Reg)
			d743 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
			ctx.BindReg(r151, &d743)
		} else if d741.Loc == scm.LocImm && d741.Imm.Int() == 0 {
			d743 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d742.Reg}
			ctx.BindReg(d742.Reg, &d743)
		} else if d741.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d742.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d741.Imm.Int()))
			ctx.EmitAddInt64(scratch, d742.Reg)
			d743 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d743)
		} else if d742.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d741.Reg)
			ctx.EmitMovRegReg(scratch, d741.Reg)
			if d742.Imm.Int() >= -2147483648 && d742.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d742.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d742.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d743 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d743)
		} else {
			r152 := ctx.AllocRegExcept(d741.Reg, d742.Reg)
			ctx.EmitMovRegReg(r152, d741.Reg)
			ctx.EmitAddInt64(r152, d742.Reg)
			d743 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r152}
			ctx.BindReg(r152, &d743)
		}
		if d743.Loc == scm.LocReg && d741.Loc == scm.LocReg && d743.Reg == d741.Reg {
			ctx.TransferReg(d741.Reg)
			d741.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d741)
		ctx.EnsureDesc(&d6)
		d744 = d6
		_ = d744
		ctx.StabilizeDescForControlFlow(&d744)
		bbpos_5_0 := int32(-1)
		_ = bbpos_5_0
		lbl29 := ctx.ReserveLabel()
		_ = lbl29
		bbpos_5_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl29)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d744)
		ctx.EnsureDesc(&d744)
		var d745 scm.JITValueDesc
		if d744.Loc == scm.LocImm {
			d745 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d744.Imm.Int()))))}
		} else {
			r153 := ctx.AllocReg()
			ctx.EmitMovRegReg(r153, d744.Reg)
			ctx.EmitShlRegImm8(r153, 32)
			ctx.EmitShrRegImm8(r153, 32)
			d745 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r153}
			ctx.BindReg(r153, &d745)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d746 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			r154 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r154, fieldAddr)
			d746 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r154}
			ctx.BindReg(r154, &d746)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			r155 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r155, thisptr.Reg, off)
			d746 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r155}
			ctx.BindReg(r155, &d746)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d746)
		ctx.EnsureDesc(&d746)
		var d747 scm.JITValueDesc
		if d746.Loc == scm.LocImm {
			d747 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d746.Imm.Int()))))}
		} else {
			r156 := ctx.AllocReg()
			ctx.EmitMovRegReg(r156, d746.Reg)
			ctx.EmitShlRegImm8(r156, 56)
			ctx.EmitShrRegImm8(r156, 56)
			d747 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r156}
			ctx.BindReg(r156, &d747)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d745)
		ctx.EnsureDesc(&d747)
		ctx.EnsureDescsTogether(&d745, &d747)
		var d748 scm.JITValueDesc
		if d745.Loc == scm.LocImm && d747.Loc == scm.LocImm {
			d748 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d745.Imm.Int() * d747.Imm.Int())}
		} else if d745.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d747.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d745.Imm.Int()))
			ctx.EmitImulInt64(scratch, d747.Reg)
			d748 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d748)
		} else if d747.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d745.Reg)
			ctx.EmitMovRegReg(scratch, d745.Reg)
			if d747.Imm.Int() >= -2147483648 && d747.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d747.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d747.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d748 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d748)
		} else {
			r157 := ctx.AllocRegExcept(d745.Reg, d747.Reg)
			ctx.EmitMovRegReg(r157, d745.Reg)
			ctx.EmitImulInt64(r157, d747.Reg)
			d748 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r157}
			ctx.BindReg(r157, &d748)
		}
		if d748.Loc == scm.LocReg && d745.Loc == scm.LocReg && d748.Reg == d745.Reg {
			ctx.TransferReg(d745.Reg)
			d745.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d745)
		ctx.FreeDesc(&d747)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d748)
		var d749 scm.JITValueDesc
		if d748.Loc == scm.LocImm {
			d749 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d748.Imm.Int() / 64)}
		} else {
			r158 := ctx.AllocRegExcept(d748.Reg)
			ctx.EmitMovRegReg(r158, d748.Reg)
			ctx.EmitShrRegImm8(r158, 6)
			d749 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r158}
			ctx.BindReg(r158, &d749)
		}
		if d749.Loc == scm.LocReg && d748.Loc == scm.LocReg && d749.Reg == d748.Reg {
			ctx.TransferReg(d748.Reg)
			d748.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d748)
		var d750 scm.JITValueDesc
		if d748.Loc == scm.LocImm {
			d750 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d748.Imm.Int() % 64)}
		} else {
			r159 := ctx.AllocRegExcept(d748.Reg)
			ctx.EmitMovRegReg(r159, d748.Reg)
			ctx.EmitAndRegImm32(r159, 63)
			d750 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
			ctx.BindReg(r159, &d750)
		}
		if d750.Loc == scm.LocReg && d748.Loc == scm.LocReg && d750.Reg == d748.Reg {
			ctx.TransferReg(d748.Reg)
			d748.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d748)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d751 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
			r160 := ctx.AllocReg()
			r161 := ctx.AllocRegExcept(r160)
			r162 := ctx.AllocRegExcept(r160, r161)
			ctx.EmitMovRegMem64(r160, fieldAddr)
			ctx.EmitMovRegMem64(r161, fieldAddr+8)
			ctx.EmitMovRegMem64(r162, fieldAddr+16)
			d751 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r160, Reg2: r161, Reg3: r162}
			ctx.BindReg(r160, &d751)
			ctx.BindReg(r161, &d751)
			ctx.BindReg(r162, &d751)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
			r163 := ctx.AllocReg()
			r164 := ctx.AllocRegExcept(r163)
			r165 := ctx.AllocRegExcept(r163, r164)
			ctx.EmitMovRegMem(r163, thisptr.Reg, off)
			ctx.EmitMovRegMem(r164, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r165, thisptr.Reg, off+16)
			d751 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r163, Reg2: r164, Reg3: r165}
			ctx.BindReg(r163, &d751)
			ctx.BindReg(r164, &d751)
			ctx.BindReg(r165, &d751)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d749)
		ctx.ReclaimUntrackedRegs()
		d753 = ctx.EmitSliceElementAddress(&d751, &d749, 8)
		ctx.EnsureDesc(&d753)
		ctx.EmitMovRegMem(d753.Reg, d753.Reg, 0)
		d752 = d753
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d752)
		ctx.EnsureDesc(&d750)
		var d754 scm.JITValueDesc
		if d752.Loc == scm.LocImm && d750.Loc == scm.LocImm {
			d754 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d752.Imm.Int()) << uint64(d750.Imm.Int())))}
		} else if d750.Loc == scm.LocImm {
			r166 := ctx.AllocRegExcept(d752.Reg)
			ctx.EmitMovRegReg(r166, d752.Reg)
			ctx.EmitShlRegImm8(r166, uint8(d750.Imm.Int()))
			d754 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r166}
			ctx.BindReg(r166, &d754)
		} else {
			{
				shiftSrc := d752.Reg
				r167 := ctx.AllocRegExcept(d752.Reg)
				ctx.EmitMovRegReg(r167, d752.Reg)
				shiftSrc = r167
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d750.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d750.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d750.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d754 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d754)
			}
		}
		if d754.Loc == scm.LocReg && d752.Loc == scm.LocReg && d754.Reg == d752.Reg {
			ctx.TransferReg(d752.Reg)
			d752.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d752)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d749)
		ctx.EnsureDesc(&d749)
		var d755 scm.JITValueDesc
		if d749.Loc == scm.LocImm {
			d755 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d749.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d749.Reg)
			ctx.EmitMovRegReg(scratch, d749.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d755 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d755)
		}
		if d755.Loc == scm.LocReg && d749.Loc == scm.LocReg && d755.Reg == d749.Reg {
			ctx.TransferReg(d749.Reg)
			d749.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d749)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d755)
		ctx.ReclaimUntrackedRegs()
		d757 = ctx.EmitSliceElementAddress(&d751, &d755, 8)
		ctx.EnsureDesc(&d757)
		ctx.EmitMovRegMem(d757.Reg, d757.Reg, 0)
		d756 = d757
		ctx.FreeDesc(&d755)
		ctx.ReclaimUntrackedRegs()
		d758 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d750)
		ctx.EnsureDescsTogether(&d758, &d750)
		var d759 scm.JITValueDesc
		if d758.Loc == scm.LocImm && d750.Loc == scm.LocImm {
			d759 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d758.Imm.Int() - d750.Imm.Int())}
		} else if d750.Loc == scm.LocImm && d750.Imm.Int() == 0 {
			r168 := ctx.AllocRegExcept(d758.Reg)
			ctx.EmitMovRegReg(r168, d758.Reg)
			d759 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r168}
			ctx.BindReg(r168, &d759)
		} else if d758.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d750.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d758.Imm.Int()))
			ctx.EmitSubInt64(scratch, d750.Reg)
			d759 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d759)
		} else if d750.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d758.Reg)
			ctx.EmitMovRegReg(scratch, d758.Reg)
			if d750.Imm.Int() >= -2147483648 && d750.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d750.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d750.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d759 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d759)
		} else {
			r169 := ctx.AllocRegExcept(d758.Reg, d750.Reg)
			ctx.EmitMovRegReg(r169, d758.Reg)
			ctx.EmitSubInt64(r169, d750.Reg)
			d759 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r169}
			ctx.BindReg(r169, &d759)
		}
		if d759.Loc == scm.LocReg && d758.Loc == scm.LocReg && d759.Reg == d758.Reg {
			ctx.TransferReg(d758.Reg)
			d758.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d750)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d756)
		ctx.EnsureDesc(&d759)
		var d760 scm.JITValueDesc
		if d756.Loc == scm.LocImm && d759.Loc == scm.LocImm {
			d760 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d756.Imm.Int()) >> uint64(d759.Imm.Int())))}
		} else if d759.Loc == scm.LocImm {
			r170 := ctx.AllocRegExcept(d756.Reg)
			ctx.EmitMovRegReg(r170, d756.Reg)
			ctx.EmitShrRegImm8(r170, uint8(d759.Imm.Int()))
			d760 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r170}
			ctx.BindReg(r170, &d760)
		} else {
			{
				shiftSrc := d756.Reg
				r171 := ctx.AllocRegExcept(d756.Reg)
				ctx.EmitMovRegReg(r171, d756.Reg)
				shiftSrc = r171
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
		ctx.EnsureDesc(&d754)
		ctx.EnsureDesc(&d760)
		var d761 scm.JITValueDesc
		if d754.Loc == scm.LocImm && d760.Loc == scm.LocImm {
			d761 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d754.Imm.Int() | d760.Imm.Int())}
		} else if d754.Loc == scm.LocImm && d754.Imm.Int() == 0 {
			d761 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d760.Reg}
			ctx.BindReg(d760.Reg, &d761)
		} else if d760.Loc == scm.LocImm && d760.Imm.Int() == 0 {
			r172 := ctx.AllocRegExcept(d754.Reg)
			ctx.EmitMovRegReg(r172, d754.Reg)
			d761 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
			ctx.BindReg(r172, &d761)
		} else if d754.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d760.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d754.Imm.Int()))
			ctx.EmitOrInt64(scratch, d760.Reg)
			d761 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d761)
		} else if d760.Loc == scm.LocImm {
			r173 := ctx.AllocRegExcept(d754.Reg)
			ctx.EmitMovRegReg(r173, d754.Reg)
			if d760.Imm.Int() >= -2147483648 && d760.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r173, int32(d760.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d760.Imm.Int()))
				ctx.EmitOrInt64(r173, scm.RegR11)
			}
			d761 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r173}
			ctx.BindReg(r173, &d761)
		} else {
			r174 := ctx.AllocRegExcept(d754.Reg, d760.Reg)
			ctx.EmitMovRegReg(r174, d754.Reg)
			ctx.EmitOrInt64(r174, d760.Reg)
			d761 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r174}
			ctx.BindReg(r174, &d761)
		}
		if d761.Loc == scm.LocReg && d754.Loc == scm.LocReg && d761.Reg == d754.Reg {
			ctx.TransferReg(d754.Reg)
			d754.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d754)
		ctx.FreeDesc(&d760)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d746)
		ctx.EnsureDesc(&d746)
		var d762 scm.JITValueDesc
		if d746.Loc == scm.LocImm {
			d762 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d746.Imm.Int()))))}
		} else {
			r175 := ctx.AllocReg()
			ctx.EmitMovRegReg(r175, d746.Reg)
			ctx.EmitShlRegImm8(r175, 56)
			ctx.EmitShrRegImm8(r175, 56)
			d762 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r175}
			ctx.BindReg(r175, &d762)
		}
		ctx.ReclaimUntrackedRegs()
		d763 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d762)
		ctx.EnsureDescsTogether(&d763, &d762)
		var d764 scm.JITValueDesc
		if d763.Loc == scm.LocImm && d762.Loc == scm.LocImm {
			d764 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d763.Imm.Int() - d762.Imm.Int())}
		} else if d762.Loc == scm.LocImm && d762.Imm.Int() == 0 {
			r176 := ctx.AllocRegExcept(d763.Reg)
			ctx.EmitMovRegReg(r176, d763.Reg)
			d764 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r176}
			ctx.BindReg(r176, &d764)
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
			r177 := ctx.AllocRegExcept(d763.Reg, d762.Reg)
			ctx.EmitMovRegReg(r177, d763.Reg)
			ctx.EmitSubInt64(r177, d762.Reg)
			d764 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r177}
			ctx.BindReg(r177, &d764)
		}
		if d764.Loc == scm.LocReg && d763.Loc == scm.LocReg && d764.Reg == d763.Reg {
			ctx.TransferReg(d763.Reg)
			d763.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d762)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d761)
		ctx.EnsureDesc(&d764)
		var d765 scm.JITValueDesc
		if d761.Loc == scm.LocImm && d764.Loc == scm.LocImm {
			d765 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d761.Imm.Int()) >> uint64(d764.Imm.Int())))}
		} else if d764.Loc == scm.LocImm {
			r178 := ctx.AllocRegExcept(d761.Reg)
			ctx.EmitMovRegReg(r178, d761.Reg)
			ctx.EmitShrRegImm8(r178, uint8(d764.Imm.Int()))
			d765 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r178}
			ctx.BindReg(r178, &d765)
		} else {
			{
				shiftSrc := d761.Reg
				r179 := ctx.AllocRegExcept(d761.Reg)
				ctx.EmitMovRegReg(r179, d761.Reg)
				shiftSrc = r179
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d764.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d764.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d764.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d765 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d765)
			}
		}
		if d765.Loc == scm.LocReg && d761.Loc == scm.LocReg && d765.Reg == d761.Reg {
			ctx.TransferReg(d761.Reg)
			d761.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d761)
		ctx.FreeDesc(&d764)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d765)
		ctx.FreeDesc(&d6)
		ctx.EnsureDesc(&d765)
		ctx.EnsureDesc(&d765)
		var d766 scm.JITValueDesc
		if d765.Loc == scm.LocImm {
			d766 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d765.Imm.Int()))))}
		} else {
			r180 := ctx.AllocReg()
			ctx.EmitMovRegReg(r180, d765.Reg)
			d766 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r180}
			ctx.BindReg(r180, &d766)
		}
		ctx.FreeDesc(&d765)
		ctx.EnsureDesc(&d766)
		ctx.EnsureDesc(&d47)
		ctx.EnsureDescsTogether(&d766, &d47)
		var d767 scm.JITValueDesc
		if d766.Loc == scm.LocImm && d47.Loc == scm.LocImm {
			d767 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d766.Imm.Int() + d47.Imm.Int())}
		} else if d47.Loc == scm.LocImm && d47.Imm.Int() == 0 {
			r181 := ctx.AllocRegExcept(d766.Reg)
			ctx.EmitMovRegReg(r181, d766.Reg)
			d767 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r181}
			ctx.BindReg(r181, &d767)
		} else if d766.Loc == scm.LocImm && d766.Imm.Int() == 0 {
			d767 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d47.Reg}
			ctx.BindReg(d47.Reg, &d767)
		} else if d766.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d47.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d766.Imm.Int()))
			ctx.EmitAddInt64(scratch, d47.Reg)
			d767 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d767)
		} else if d47.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d766.Reg)
			ctx.EmitMovRegReg(scratch, d766.Reg)
			if d47.Imm.Int() >= -2147483648 && d47.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d47.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d47.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d767 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d767)
		} else {
			r182 := ctx.AllocRegExcept(d766.Reg, d47.Reg)
			ctx.EmitMovRegReg(r182, d766.Reg)
			ctx.EmitAddInt64(r182, d47.Reg)
			d767 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r182}
			ctx.BindReg(r182, &d767)
		}
		if d767.Loc == scm.LocReg && d766.Loc == scm.LocReg && d767.Reg == d766.Reg {
			ctx.TransferReg(d766.Reg)
			d766.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d766)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		var d768 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm {
			d768 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(idxInt.Imm.Int()))))}
		} else {
			r183 := ctx.AllocReg()
			ctx.EmitMovRegReg(r183, idxInt.Reg)
			ctx.EmitShlRegImm8(r183, 32)
			ctx.EmitShrRegImm8(r183, 32)
			d768 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r183}
			ctx.BindReg(r183, &d768)
		}
		ctx.FreeDesc(&idxInt)
		ctx.EnsureDesc(&d768)
		ctx.EnsureDesc(&d767)
		ctx.EnsureDescsTogether(&d768, &d767)
		var d769 scm.JITValueDesc
		if d768.Loc == scm.LocImm && d767.Loc == scm.LocImm {
			d769 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d768.Imm.Int() - d767.Imm.Int())}
		} else if d767.Loc == scm.LocImm && d767.Imm.Int() == 0 {
			r184 := ctx.AllocRegExcept(d768.Reg)
			ctx.EmitMovRegReg(r184, d768.Reg)
			d769 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r184}
			ctx.BindReg(r184, &d769)
		} else if d768.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d767.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d768.Imm.Int()))
			ctx.EmitSubInt64(scratch, d767.Reg)
			d769 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d769)
		} else if d767.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d768.Reg)
			ctx.EmitMovRegReg(scratch, d768.Reg)
			if d767.Imm.Int() >= -2147483648 && d767.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d767.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d767.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d769 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d769)
		} else {
			r185 := ctx.AllocRegExcept(d768.Reg, d767.Reg)
			ctx.EmitMovRegReg(r185, d768.Reg)
			ctx.EmitSubInt64(r185, d767.Reg)
			d769 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r185}
			ctx.BindReg(r185, &d769)
		}
		if d769.Loc == scm.LocReg && d768.Loc == scm.LocReg && d769.Reg == d768.Reg {
			ctx.TransferReg(d768.Reg)
			d768.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d768)
		ctx.FreeDesc(&d767)
		ctx.EnsureDesc(&d769)
		ctx.EnsureDesc(&d743)
		ctx.EnsureDescsTogether(&d769, &d743)
		var d770 scm.JITValueDesc
		if d769.Loc == scm.LocImm && d743.Loc == scm.LocImm {
			d770 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d769.Imm.Int() * d743.Imm.Int())}
		} else if d769.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d743.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d769.Imm.Int()))
			ctx.EmitImulInt64(scratch, d743.Reg)
			d770 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d770)
		} else if d743.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d769.Reg)
			ctx.EmitMovRegReg(scratch, d769.Reg)
			if d743.Imm.Int() >= -2147483648 && d743.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d743.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d743.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d770 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d770)
		} else {
			r186 := ctx.AllocRegExcept(d769.Reg, d743.Reg)
			ctx.EmitMovRegReg(r186, d769.Reg)
			ctx.EmitImulInt64(r186, d743.Reg)
			d770 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r186}
			ctx.BindReg(r186, &d770)
		}
		if d770.Loc == scm.LocReg && d769.Loc == scm.LocReg && d770.Reg == d769.Reg {
			ctx.TransferReg(d769.Reg)
			d769.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d769)
		ctx.FreeDesc(&d743)
		ctx.EnsureDesc(&d137)
		ctx.EnsureDesc(&d770)
		ctx.EnsureDescsTogether(&d137, &d770)
		var d771 scm.JITValueDesc
		if d137.Loc == scm.LocImm && d770.Loc == scm.LocImm {
			d771 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d137.Imm.Int() + d770.Imm.Int())}
		} else if d770.Loc == scm.LocImm && d770.Imm.Int() == 0 {
			r187 := ctx.AllocRegExcept(d137.Reg)
			ctx.EmitMovRegReg(r187, d137.Reg)
			d771 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r187}
			ctx.BindReg(r187, &d771)
		} else if d137.Loc == scm.LocImm && d137.Imm.Int() == 0 {
			d771 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d770.Reg}
			ctx.BindReg(d770.Reg, &d771)
		} else if d137.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d770.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d137.Imm.Int()))
			ctx.EmitAddInt64(scratch, d770.Reg)
			d771 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d771)
		} else if d770.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d137.Reg)
			ctx.EmitMovRegReg(scratch, d137.Reg)
			if d770.Imm.Int() >= -2147483648 && d770.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d770.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d770.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d771 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d771)
		} else {
			r188 := ctx.AllocRegExcept(d137.Reg, d770.Reg)
			ctx.EmitMovRegReg(r188, d137.Reg)
			ctx.EmitAddInt64(r188, d770.Reg)
			d771 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r188}
			ctx.BindReg(r188, &d771)
		}
		if d771.Loc == scm.LocReg && d137.Loc == scm.LocReg && d771.Reg == d137.Reg {
			ctx.TransferReg(d137.Reg)
			d137.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d770)
		ctx.EnsureDesc(&d771)
		ctx.EnsureDesc(&d771)
		var d772 scm.JITValueDesc
		if d771.Loc == scm.LocImm {
			d772 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d771.Imm.Int()))}
		} else {
			ctx.EmitCvtInt64ToFloat64(scm.RegX0, d771.Reg)
			d772 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d771.Reg}
			ctx.BindReg(d771.Reg, &d772)
		}
		ctx.FreeDesc(&d771)
		ctx.EnsureDesc(&d772)
		d773 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
		ctx.BindReg(r1, &d773)
		ctx.BindReg(r2, &d773)
		ctx.EnsureDesc(&d772)
		ctx.EmitMakeFloat(d773, d772)
		if d772.Loc == scm.LocReg {
			ctx.FreeReg(d772.Reg)
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
			d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0}
			ctx.BindReg(r0, &d3)
		} else {
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
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
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
			d16 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
			d17 = ps.OverlayValues[17]
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
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
			d51 = ps.OverlayValues[51]
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
		if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != scm.LocNone {
			d142 = ps.OverlayValues[142]
		}
		if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
			d227 = ps.OverlayValues[227]
		}
		if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
			d228 = ps.OverlayValues[228]
		}
		if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != scm.LocNone {
			d229 = ps.OverlayValues[229]
		}
		if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
			d230 = ps.OverlayValues[230]
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
		if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
			d238 = ps.OverlayValues[238]
		}
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
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
		if len(ps.OverlayValues) > 250 && ps.OverlayValues[250].Loc != scm.LocNone {
			d250 = ps.OverlayValues[250]
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
		if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != scm.LocNone {
			d355 = ps.OverlayValues[355]
		}
		if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != scm.LocNone {
			d356 = ps.OverlayValues[356]
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
		if len(ps.OverlayValues) > 385 && ps.OverlayValues[385].Loc != scm.LocNone {
			d385 = ps.OverlayValues[385]
		}
		if len(ps.OverlayValues) > 386 && ps.OverlayValues[386].Loc != scm.LocNone {
			d386 = ps.OverlayValues[386]
		}
		if len(ps.OverlayValues) > 526 && ps.OverlayValues[526].Loc != scm.LocNone {
			d526 = ps.OverlayValues[526]
		}
		if len(ps.OverlayValues) > 527 && ps.OverlayValues[527].Loc != scm.LocNone {
			d527 = ps.OverlayValues[527]
		}
		if len(ps.OverlayValues) > 528 && ps.OverlayValues[528].Loc != scm.LocNone {
			d528 = ps.OverlayValues[528]
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
		if len(ps.OverlayValues) > 535 && ps.OverlayValues[535].Loc != scm.LocNone {
			d535 = ps.OverlayValues[535]
		}
		if len(ps.OverlayValues) > 536 && ps.OverlayValues[536].Loc != scm.LocNone {
			d536 = ps.OverlayValues[536]
		}
		if len(ps.OverlayValues) > 538 && ps.OverlayValues[538].Loc != scm.LocNone {
			d538 = ps.OverlayValues[538]
		}
		if len(ps.OverlayValues) > 540 && ps.OverlayValues[540].Loc != scm.LocNone {
			d540 = ps.OverlayValues[540]
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
		if len(ps.OverlayValues) > 546 && ps.OverlayValues[546].Loc != scm.LocNone {
			d546 = ps.OverlayValues[546]
		}
		if len(ps.OverlayValues) > 698 && ps.OverlayValues[698].Loc != scm.LocNone {
			d698 = ps.OverlayValues[698]
		}
		if len(ps.OverlayValues) > 699 && ps.OverlayValues[699].Loc != scm.LocNone {
			d699 = ps.OverlayValues[699]
		}
		if len(ps.OverlayValues) > 700 && ps.OverlayValues[700].Loc != scm.LocNone {
			d700 = ps.OverlayValues[700]
		}
		if len(ps.OverlayValues) > 701 && ps.OverlayValues[701].Loc != scm.LocNone {
			d701 = ps.OverlayValues[701]
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
		if len(ps.OverlayValues) > 709 && ps.OverlayValues[709].Loc != scm.LocNone {
			d709 = ps.OverlayValues[709]
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
		ctx.ReclaimUntrackedRegs()
		var d774 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 64
			r189 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r189, fieldAddr)
			d774 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r189}
			ctx.BindReg(r189, &d774)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 64)
			r190 := ctx.AllocReg()
			ctx.EmitMovRegMem(r190, thisptr.Reg, off)
			d774 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r190}
			ctx.BindReg(r190, &d774)
		}
		ctx.EnsureDesc(&d774)
		ctx.EnsureDesc(&d774)
		var d775 scm.JITValueDesc
		if d774.Loc == scm.LocImm {
			d775 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d774.Imm.Int()))))}
		} else {
			r191 := ctx.AllocReg()
			ctx.EmitMovRegReg(r191, d774.Reg)
			d775 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r191}
			ctx.BindReg(r191, &d775)
		}
		ctx.EnsureDesc(&d137)
		ctx.EnsureDesc(&d775)
		ctx.EnsureDescsTogether(&d137, &d775)
		var d776 scm.JITValueDesc
		if d137.Loc == scm.LocImm && d775.Loc == scm.LocImm {
			d776 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d137.Imm.Int() == d775.Imm.Int())}
		} else if d775.Loc == scm.LocImm {
			r192 := ctx.AllocRegExcept(d137.Reg)
			if d775.Imm.Int() >= -2147483648 && d775.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d137.Reg, int32(d775.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d775.Imm.Int()))
				ctx.EmitCmpInt64(d137.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r192, scm.CondEqual)
			d776 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r192}
			ctx.BindReg(r192, &d776)
		} else if d137.Loc == scm.LocImm {
			r193 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d137.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d775.Reg)
			ctx.EmitSetcc(r193, scm.CondEqual)
			d776 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r193}
			ctx.BindReg(r193, &d776)
		} else {
			r194 := ctx.AllocRegExcept(d137.Reg)
			ctx.EmitCmpInt64(d137.Reg, d775.Reg)
			ctx.EmitSetcc(r194, scm.CondEqual)
			d776 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r194}
			ctx.BindReg(r194, &d776)
		}
		ctx.FreeDesc(&d137)
		ctx.FreeDesc(&d775)
		d777 = d776
		ctx.EnsureDesc(&d777)
		if d777.Loc != scm.LocImm && d777.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d777.Loc == scm.LocImm {
			if d777.Imm.Bool() {
				if ps.General {
				}
				ps778 := scm.PhiState{General: ps.General}
				ps778.OverlayValues = make([]scm.JITValueDesc, 778)
				ps778.OverlayValues[3] = d3
				ps778.OverlayValues[4] = d4
				ps778.OverlayValues[5] = d5
				ps778.OverlayValues[6] = d6
				ps778.OverlayValues[7] = d7
				ps778.OverlayValues[8] = d8
				ps778.OverlayValues[9] = d9
				ps778.OverlayValues[10] = d10
				ps778.OverlayValues[11] = d11
				ps778.OverlayValues[12] = d12
				ps778.OverlayValues[13] = d13
				ps778.OverlayValues[14] = d14
				ps778.OverlayValues[15] = d15
				ps778.OverlayValues[16] = d16
				ps778.OverlayValues[17] = d17
				ps778.OverlayValues[19] = d19
				ps778.OverlayValues[20] = d20
				ps778.OverlayValues[21] = d21
				ps778.OverlayValues[22] = d22
				ps778.OverlayValues[23] = d23
				ps778.OverlayValues[24] = d24
				ps778.OverlayValues[25] = d25
				ps778.OverlayValues[26] = d26
				ps778.OverlayValues[27] = d27
				ps778.OverlayValues[28] = d28
				ps778.OverlayValues[29] = d29
				ps778.OverlayValues[30] = d30
				ps778.OverlayValues[31] = d31
				ps778.OverlayValues[32] = d32
				ps778.OverlayValues[33] = d33
				ps778.OverlayValues[34] = d34
				ps778.OverlayValues[35] = d35
				ps778.OverlayValues[36] = d36
				ps778.OverlayValues[37] = d37
				ps778.OverlayValues[38] = d38
				ps778.OverlayValues[39] = d39
				ps778.OverlayValues[40] = d40
				ps778.OverlayValues[41] = d41
				ps778.OverlayValues[42] = d42
				ps778.OverlayValues[43] = d43
				ps778.OverlayValues[44] = d44
				ps778.OverlayValues[45] = d45
				ps778.OverlayValues[46] = d46
				ps778.OverlayValues[47] = d47
				ps778.OverlayValues[48] = d48
				ps778.OverlayValues[49] = d49
				ps778.OverlayValues[50] = d50
				ps778.OverlayValues[51] = d51
				ps778.OverlayValues[54] = d54
				ps778.OverlayValues[55] = d55
				ps778.OverlayValues[56] = d56
				ps778.OverlayValues[111] = d111
				ps778.OverlayValues[112] = d112
				ps778.OverlayValues[113] = d113
				ps778.OverlayValues[114] = d114
				ps778.OverlayValues[115] = d115
				ps778.OverlayValues[116] = d116
				ps778.OverlayValues[117] = d117
				ps778.OverlayValues[118] = d118
				ps778.OverlayValues[119] = d119
				ps778.OverlayValues[120] = d120
				ps778.OverlayValues[121] = d121
				ps778.OverlayValues[122] = d122
				ps778.OverlayValues[123] = d123
				ps778.OverlayValues[124] = d124
				ps778.OverlayValues[125] = d125
				ps778.OverlayValues[126] = d126
				ps778.OverlayValues[127] = d127
				ps778.OverlayValues[128] = d128
				ps778.OverlayValues[129] = d129
				ps778.OverlayValues[130] = d130
				ps778.OverlayValues[131] = d131
				ps778.OverlayValues[132] = d132
				ps778.OverlayValues[133] = d133
				ps778.OverlayValues[134] = d134
				ps778.OverlayValues[135] = d135
				ps778.OverlayValues[136] = d136
				ps778.OverlayValues[137] = d137
				ps778.OverlayValues[138] = d138
				ps778.OverlayValues[139] = d139
				ps778.OverlayValues[142] = d142
				ps778.OverlayValues[227] = d227
				ps778.OverlayValues[228] = d228
				ps778.OverlayValues[229] = d229
				ps778.OverlayValues[230] = d230
				ps778.OverlayValues[232] = d232
				ps778.OverlayValues[233] = d233
				ps778.OverlayValues[234] = d234
				ps778.OverlayValues[235] = d235
				ps778.OverlayValues[236] = d236
				ps778.OverlayValues[237] = d237
				ps778.OverlayValues[238] = d238
				ps778.OverlayValues[239] = d239
				ps778.OverlayValues[241] = d241
				ps778.OverlayValues[243] = d243
				ps778.OverlayValues[244] = d244
				ps778.OverlayValues[245] = d245
				ps778.OverlayValues[246] = d246
				ps778.OverlayValues[247] = d247
				ps778.OverlayValues[250] = d250
				ps778.OverlayValues[352] = d352
				ps778.OverlayValues[353] = d353
				ps778.OverlayValues[354] = d354
				ps778.OverlayValues[355] = d355
				ps778.OverlayValues[356] = d356
				ps778.OverlayValues[358] = d358
				ps778.OverlayValues[359] = d359
				ps778.OverlayValues[360] = d360
				ps778.OverlayValues[361] = d361
				ps778.OverlayValues[362] = d362
				ps778.OverlayValues[363] = d363
				ps778.OverlayValues[364] = d364
				ps778.OverlayValues[365] = d365
				ps778.OverlayValues[366] = d366
				ps778.OverlayValues[367] = d367
				ps778.OverlayValues[368] = d368
				ps778.OverlayValues[369] = d369
				ps778.OverlayValues[370] = d370
				ps778.OverlayValues[371] = d371
				ps778.OverlayValues[372] = d372
				ps778.OverlayValues[373] = d373
				ps778.OverlayValues[374] = d374
				ps778.OverlayValues[375] = d375
				ps778.OverlayValues[376] = d376
				ps778.OverlayValues[377] = d377
				ps778.OverlayValues[378] = d378
				ps778.OverlayValues[379] = d379
				ps778.OverlayValues[380] = d380
				ps778.OverlayValues[381] = d381
				ps778.OverlayValues[382] = d382
				ps778.OverlayValues[383] = d383
				ps778.OverlayValues[384] = d384
				ps778.OverlayValues[385] = d385
				ps778.OverlayValues[386] = d386
				ps778.OverlayValues[526] = d526
				ps778.OverlayValues[527] = d527
				ps778.OverlayValues[528] = d528
				ps778.OverlayValues[530] = d530
				ps778.OverlayValues[531] = d531
				ps778.OverlayValues[532] = d532
				ps778.OverlayValues[533] = d533
				ps778.OverlayValues[534] = d534
				ps778.OverlayValues[535] = d535
				ps778.OverlayValues[536] = d536
				ps778.OverlayValues[538] = d538
				ps778.OverlayValues[540] = d540
				ps778.OverlayValues[541] = d541
				ps778.OverlayValues[542] = d542
				ps778.OverlayValues[543] = d543
				ps778.OverlayValues[546] = d546
				ps778.OverlayValues[698] = d698
				ps778.OverlayValues[699] = d699
				ps778.OverlayValues[700] = d700
				ps778.OverlayValues[701] = d701
				ps778.OverlayValues[703] = d703
				ps778.OverlayValues[704] = d704
				ps778.OverlayValues[705] = d705
				ps778.OverlayValues[706] = d706
				ps778.OverlayValues[707] = d707
				ps778.OverlayValues[708] = d708
				ps778.OverlayValues[709] = d709
				ps778.OverlayValues[710] = d710
				ps778.OverlayValues[711] = d711
				ps778.OverlayValues[712] = d712
				ps778.OverlayValues[714] = d714
				ps778.OverlayValues[715] = d715
				ps778.OverlayValues[716] = d716
				ps778.OverlayValues[717] = d717
				ps778.OverlayValues[718] = d718
				ps778.OverlayValues[719] = d719
				ps778.OverlayValues[720] = d720
				ps778.OverlayValues[721] = d721
				ps778.OverlayValues[722] = d722
				ps778.OverlayValues[723] = d723
				ps778.OverlayValues[724] = d724
				ps778.OverlayValues[725] = d725
				ps778.OverlayValues[726] = d726
				ps778.OverlayValues[727] = d727
				ps778.OverlayValues[728] = d728
				ps778.OverlayValues[729] = d729
				ps778.OverlayValues[730] = d730
				ps778.OverlayValues[731] = d731
				ps778.OverlayValues[732] = d732
				ps778.OverlayValues[733] = d733
				ps778.OverlayValues[734] = d734
				ps778.OverlayValues[735] = d735
				ps778.OverlayValues[736] = d736
				ps778.OverlayValues[737] = d737
				ps778.OverlayValues[738] = d738
				ps778.OverlayValues[739] = d739
				ps778.OverlayValues[740] = d740
				ps778.OverlayValues[741] = d741
				ps778.OverlayValues[742] = d742
				ps778.OverlayValues[743] = d743
				ps778.OverlayValues[744] = d744
				ps778.OverlayValues[745] = d745
				ps778.OverlayValues[746] = d746
				ps778.OverlayValues[747] = d747
				ps778.OverlayValues[748] = d748
				ps778.OverlayValues[749] = d749
				ps778.OverlayValues[750] = d750
				ps778.OverlayValues[751] = d751
				ps778.OverlayValues[752] = d752
				ps778.OverlayValues[753] = d753
				ps778.OverlayValues[754] = d754
				ps778.OverlayValues[755] = d755
				ps778.OverlayValues[756] = d756
				ps778.OverlayValues[757] = d757
				ps778.OverlayValues[758] = d758
				ps778.OverlayValues[759] = d759
				ps778.OverlayValues[760] = d760
				ps778.OverlayValues[761] = d761
				ps778.OverlayValues[762] = d762
				ps778.OverlayValues[763] = d763
				ps778.OverlayValues[764] = d764
				ps778.OverlayValues[765] = d765
				ps778.OverlayValues[766] = d766
				ps778.OverlayValues[767] = d767
				ps778.OverlayValues[768] = d768
				ps778.OverlayValues[769] = d769
				ps778.OverlayValues[770] = d770
				ps778.OverlayValues[771] = d771
				ps778.OverlayValues[772] = d772
				ps778.OverlayValues[773] = d773
				ps778.OverlayValues[774] = d774
				ps778.OverlayValues[775] = d775
				ps778.OverlayValues[776] = d776
				ps778.OverlayValues[777] = d777
				return bbs[11].RenderPS(ps778)
			}
			if ps.General {
			}
			ps779 := scm.PhiState{General: ps.General}
			ps779.OverlayValues = make([]scm.JITValueDesc, 778)
			ps779.OverlayValues[3] = d3
			ps779.OverlayValues[4] = d4
			ps779.OverlayValues[5] = d5
			ps779.OverlayValues[6] = d6
			ps779.OverlayValues[7] = d7
			ps779.OverlayValues[8] = d8
			ps779.OverlayValues[9] = d9
			ps779.OverlayValues[10] = d10
			ps779.OverlayValues[11] = d11
			ps779.OverlayValues[12] = d12
			ps779.OverlayValues[13] = d13
			ps779.OverlayValues[14] = d14
			ps779.OverlayValues[15] = d15
			ps779.OverlayValues[16] = d16
			ps779.OverlayValues[17] = d17
			ps779.OverlayValues[19] = d19
			ps779.OverlayValues[20] = d20
			ps779.OverlayValues[21] = d21
			ps779.OverlayValues[22] = d22
			ps779.OverlayValues[23] = d23
			ps779.OverlayValues[24] = d24
			ps779.OverlayValues[25] = d25
			ps779.OverlayValues[26] = d26
			ps779.OverlayValues[27] = d27
			ps779.OverlayValues[28] = d28
			ps779.OverlayValues[29] = d29
			ps779.OverlayValues[30] = d30
			ps779.OverlayValues[31] = d31
			ps779.OverlayValues[32] = d32
			ps779.OverlayValues[33] = d33
			ps779.OverlayValues[34] = d34
			ps779.OverlayValues[35] = d35
			ps779.OverlayValues[36] = d36
			ps779.OverlayValues[37] = d37
			ps779.OverlayValues[38] = d38
			ps779.OverlayValues[39] = d39
			ps779.OverlayValues[40] = d40
			ps779.OverlayValues[41] = d41
			ps779.OverlayValues[42] = d42
			ps779.OverlayValues[43] = d43
			ps779.OverlayValues[44] = d44
			ps779.OverlayValues[45] = d45
			ps779.OverlayValues[46] = d46
			ps779.OverlayValues[47] = d47
			ps779.OverlayValues[48] = d48
			ps779.OverlayValues[49] = d49
			ps779.OverlayValues[50] = d50
			ps779.OverlayValues[51] = d51
			ps779.OverlayValues[54] = d54
			ps779.OverlayValues[55] = d55
			ps779.OverlayValues[56] = d56
			ps779.OverlayValues[111] = d111
			ps779.OverlayValues[112] = d112
			ps779.OverlayValues[113] = d113
			ps779.OverlayValues[114] = d114
			ps779.OverlayValues[115] = d115
			ps779.OverlayValues[116] = d116
			ps779.OverlayValues[117] = d117
			ps779.OverlayValues[118] = d118
			ps779.OverlayValues[119] = d119
			ps779.OverlayValues[120] = d120
			ps779.OverlayValues[121] = d121
			ps779.OverlayValues[122] = d122
			ps779.OverlayValues[123] = d123
			ps779.OverlayValues[124] = d124
			ps779.OverlayValues[125] = d125
			ps779.OverlayValues[126] = d126
			ps779.OverlayValues[127] = d127
			ps779.OverlayValues[128] = d128
			ps779.OverlayValues[129] = d129
			ps779.OverlayValues[130] = d130
			ps779.OverlayValues[131] = d131
			ps779.OverlayValues[132] = d132
			ps779.OverlayValues[133] = d133
			ps779.OverlayValues[134] = d134
			ps779.OverlayValues[135] = d135
			ps779.OverlayValues[136] = d136
			ps779.OverlayValues[137] = d137
			ps779.OverlayValues[138] = d138
			ps779.OverlayValues[139] = d139
			ps779.OverlayValues[142] = d142
			ps779.OverlayValues[227] = d227
			ps779.OverlayValues[228] = d228
			ps779.OverlayValues[229] = d229
			ps779.OverlayValues[230] = d230
			ps779.OverlayValues[232] = d232
			ps779.OverlayValues[233] = d233
			ps779.OverlayValues[234] = d234
			ps779.OverlayValues[235] = d235
			ps779.OverlayValues[236] = d236
			ps779.OverlayValues[237] = d237
			ps779.OverlayValues[238] = d238
			ps779.OverlayValues[239] = d239
			ps779.OverlayValues[241] = d241
			ps779.OverlayValues[243] = d243
			ps779.OverlayValues[244] = d244
			ps779.OverlayValues[245] = d245
			ps779.OverlayValues[246] = d246
			ps779.OverlayValues[247] = d247
			ps779.OverlayValues[250] = d250
			ps779.OverlayValues[352] = d352
			ps779.OverlayValues[353] = d353
			ps779.OverlayValues[354] = d354
			ps779.OverlayValues[355] = d355
			ps779.OverlayValues[356] = d356
			ps779.OverlayValues[358] = d358
			ps779.OverlayValues[359] = d359
			ps779.OverlayValues[360] = d360
			ps779.OverlayValues[361] = d361
			ps779.OverlayValues[362] = d362
			ps779.OverlayValues[363] = d363
			ps779.OverlayValues[364] = d364
			ps779.OverlayValues[365] = d365
			ps779.OverlayValues[366] = d366
			ps779.OverlayValues[367] = d367
			ps779.OverlayValues[368] = d368
			ps779.OverlayValues[369] = d369
			ps779.OverlayValues[370] = d370
			ps779.OverlayValues[371] = d371
			ps779.OverlayValues[372] = d372
			ps779.OverlayValues[373] = d373
			ps779.OverlayValues[374] = d374
			ps779.OverlayValues[375] = d375
			ps779.OverlayValues[376] = d376
			ps779.OverlayValues[377] = d377
			ps779.OverlayValues[378] = d378
			ps779.OverlayValues[379] = d379
			ps779.OverlayValues[380] = d380
			ps779.OverlayValues[381] = d381
			ps779.OverlayValues[382] = d382
			ps779.OverlayValues[383] = d383
			ps779.OverlayValues[384] = d384
			ps779.OverlayValues[385] = d385
			ps779.OverlayValues[386] = d386
			ps779.OverlayValues[526] = d526
			ps779.OverlayValues[527] = d527
			ps779.OverlayValues[528] = d528
			ps779.OverlayValues[530] = d530
			ps779.OverlayValues[531] = d531
			ps779.OverlayValues[532] = d532
			ps779.OverlayValues[533] = d533
			ps779.OverlayValues[534] = d534
			ps779.OverlayValues[535] = d535
			ps779.OverlayValues[536] = d536
			ps779.OverlayValues[538] = d538
			ps779.OverlayValues[540] = d540
			ps779.OverlayValues[541] = d541
			ps779.OverlayValues[542] = d542
			ps779.OverlayValues[543] = d543
			ps779.OverlayValues[546] = d546
			ps779.OverlayValues[698] = d698
			ps779.OverlayValues[699] = d699
			ps779.OverlayValues[700] = d700
			ps779.OverlayValues[701] = d701
			ps779.OverlayValues[703] = d703
			ps779.OverlayValues[704] = d704
			ps779.OverlayValues[705] = d705
			ps779.OverlayValues[706] = d706
			ps779.OverlayValues[707] = d707
			ps779.OverlayValues[708] = d708
			ps779.OverlayValues[709] = d709
			ps779.OverlayValues[710] = d710
			ps779.OverlayValues[711] = d711
			ps779.OverlayValues[712] = d712
			ps779.OverlayValues[714] = d714
			ps779.OverlayValues[715] = d715
			ps779.OverlayValues[716] = d716
			ps779.OverlayValues[717] = d717
			ps779.OverlayValues[718] = d718
			ps779.OverlayValues[719] = d719
			ps779.OverlayValues[720] = d720
			ps779.OverlayValues[721] = d721
			ps779.OverlayValues[722] = d722
			ps779.OverlayValues[723] = d723
			ps779.OverlayValues[724] = d724
			ps779.OverlayValues[725] = d725
			ps779.OverlayValues[726] = d726
			ps779.OverlayValues[727] = d727
			ps779.OverlayValues[728] = d728
			ps779.OverlayValues[729] = d729
			ps779.OverlayValues[730] = d730
			ps779.OverlayValues[731] = d731
			ps779.OverlayValues[732] = d732
			ps779.OverlayValues[733] = d733
			ps779.OverlayValues[734] = d734
			ps779.OverlayValues[735] = d735
			ps779.OverlayValues[736] = d736
			ps779.OverlayValues[737] = d737
			ps779.OverlayValues[738] = d738
			ps779.OverlayValues[739] = d739
			ps779.OverlayValues[740] = d740
			ps779.OverlayValues[741] = d741
			ps779.OverlayValues[742] = d742
			ps779.OverlayValues[743] = d743
			ps779.OverlayValues[744] = d744
			ps779.OverlayValues[745] = d745
			ps779.OverlayValues[746] = d746
			ps779.OverlayValues[747] = d747
			ps779.OverlayValues[748] = d748
			ps779.OverlayValues[749] = d749
			ps779.OverlayValues[750] = d750
			ps779.OverlayValues[751] = d751
			ps779.OverlayValues[752] = d752
			ps779.OverlayValues[753] = d753
			ps779.OverlayValues[754] = d754
			ps779.OverlayValues[755] = d755
			ps779.OverlayValues[756] = d756
			ps779.OverlayValues[757] = d757
			ps779.OverlayValues[758] = d758
			ps779.OverlayValues[759] = d759
			ps779.OverlayValues[760] = d760
			ps779.OverlayValues[761] = d761
			ps779.OverlayValues[762] = d762
			ps779.OverlayValues[763] = d763
			ps779.OverlayValues[764] = d764
			ps779.OverlayValues[765] = d765
			ps779.OverlayValues[766] = d766
			ps779.OverlayValues[767] = d767
			ps779.OverlayValues[768] = d768
			ps779.OverlayValues[769] = d769
			ps779.OverlayValues[770] = d770
			ps779.OverlayValues[771] = d771
			ps779.OverlayValues[772] = d772
			ps779.OverlayValues[773] = d773
			ps779.OverlayValues[774] = d774
			ps779.OverlayValues[775] = d775
			ps779.OverlayValues[776] = d776
			ps779.OverlayValues[777] = d777
			return bbs[12].RenderPS(ps779)
		}
		if !ps.General {
			ps.General = true
			return bbs[13].RenderPS(ps)
		}
		lbl30 := ctx.ReserveLabel()
		lbl31 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d777.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl30)
		ctx.EmitJmp(lbl31)
		ctx.MarkLabel(lbl30)
		ctx.EmitJmp(lbl12)
		ctx.MarkLabel(lbl31)
		ctx.EmitJmp(lbl13)
		ps780 := scm.PhiState{General: true}
		ps780.OverlayValues = make([]scm.JITValueDesc, 778)
		ps780.OverlayValues[3] = d3
		ps780.OverlayValues[4] = d4
		ps780.OverlayValues[5] = d5
		ps780.OverlayValues[6] = d6
		ps780.OverlayValues[7] = d7
		ps780.OverlayValues[8] = d8
		ps780.OverlayValues[9] = d9
		ps780.OverlayValues[10] = d10
		ps780.OverlayValues[11] = d11
		ps780.OverlayValues[12] = d12
		ps780.OverlayValues[13] = d13
		ps780.OverlayValues[14] = d14
		ps780.OverlayValues[15] = d15
		ps780.OverlayValues[16] = d16
		ps780.OverlayValues[17] = d17
		ps780.OverlayValues[19] = d19
		ps780.OverlayValues[20] = d20
		ps780.OverlayValues[21] = d21
		ps780.OverlayValues[22] = d22
		ps780.OverlayValues[23] = d23
		ps780.OverlayValues[24] = d24
		ps780.OverlayValues[25] = d25
		ps780.OverlayValues[26] = d26
		ps780.OverlayValues[27] = d27
		ps780.OverlayValues[28] = d28
		ps780.OverlayValues[29] = d29
		ps780.OverlayValues[30] = d30
		ps780.OverlayValues[31] = d31
		ps780.OverlayValues[32] = d32
		ps780.OverlayValues[33] = d33
		ps780.OverlayValues[34] = d34
		ps780.OverlayValues[35] = d35
		ps780.OverlayValues[36] = d36
		ps780.OverlayValues[37] = d37
		ps780.OverlayValues[38] = d38
		ps780.OverlayValues[39] = d39
		ps780.OverlayValues[40] = d40
		ps780.OverlayValues[41] = d41
		ps780.OverlayValues[42] = d42
		ps780.OverlayValues[43] = d43
		ps780.OverlayValues[44] = d44
		ps780.OverlayValues[45] = d45
		ps780.OverlayValues[46] = d46
		ps780.OverlayValues[47] = d47
		ps780.OverlayValues[48] = d48
		ps780.OverlayValues[49] = d49
		ps780.OverlayValues[50] = d50
		ps780.OverlayValues[51] = d51
		ps780.OverlayValues[54] = d54
		ps780.OverlayValues[55] = d55
		ps780.OverlayValues[56] = d56
		ps780.OverlayValues[111] = d111
		ps780.OverlayValues[112] = d112
		ps780.OverlayValues[113] = d113
		ps780.OverlayValues[114] = d114
		ps780.OverlayValues[115] = d115
		ps780.OverlayValues[116] = d116
		ps780.OverlayValues[117] = d117
		ps780.OverlayValues[118] = d118
		ps780.OverlayValues[119] = d119
		ps780.OverlayValues[120] = d120
		ps780.OverlayValues[121] = d121
		ps780.OverlayValues[122] = d122
		ps780.OverlayValues[123] = d123
		ps780.OverlayValues[124] = d124
		ps780.OverlayValues[125] = d125
		ps780.OverlayValues[126] = d126
		ps780.OverlayValues[127] = d127
		ps780.OverlayValues[128] = d128
		ps780.OverlayValues[129] = d129
		ps780.OverlayValues[130] = d130
		ps780.OverlayValues[131] = d131
		ps780.OverlayValues[132] = d132
		ps780.OverlayValues[133] = d133
		ps780.OverlayValues[134] = d134
		ps780.OverlayValues[135] = d135
		ps780.OverlayValues[136] = d136
		ps780.OverlayValues[137] = d137
		ps780.OverlayValues[138] = d138
		ps780.OverlayValues[139] = d139
		ps780.OverlayValues[142] = d142
		ps780.OverlayValues[227] = d227
		ps780.OverlayValues[228] = d228
		ps780.OverlayValues[229] = d229
		ps780.OverlayValues[230] = d230
		ps780.OverlayValues[232] = d232
		ps780.OverlayValues[233] = d233
		ps780.OverlayValues[234] = d234
		ps780.OverlayValues[235] = d235
		ps780.OverlayValues[236] = d236
		ps780.OverlayValues[237] = d237
		ps780.OverlayValues[238] = d238
		ps780.OverlayValues[239] = d239
		ps780.OverlayValues[241] = d241
		ps780.OverlayValues[243] = d243
		ps780.OverlayValues[244] = d244
		ps780.OverlayValues[245] = d245
		ps780.OverlayValues[246] = d246
		ps780.OverlayValues[247] = d247
		ps780.OverlayValues[250] = d250
		ps780.OverlayValues[352] = d352
		ps780.OverlayValues[353] = d353
		ps780.OverlayValues[354] = d354
		ps780.OverlayValues[355] = d355
		ps780.OverlayValues[356] = d356
		ps780.OverlayValues[358] = d358
		ps780.OverlayValues[359] = d359
		ps780.OverlayValues[360] = d360
		ps780.OverlayValues[361] = d361
		ps780.OverlayValues[362] = d362
		ps780.OverlayValues[363] = d363
		ps780.OverlayValues[364] = d364
		ps780.OverlayValues[365] = d365
		ps780.OverlayValues[366] = d366
		ps780.OverlayValues[367] = d367
		ps780.OverlayValues[368] = d368
		ps780.OverlayValues[369] = d369
		ps780.OverlayValues[370] = d370
		ps780.OverlayValues[371] = d371
		ps780.OverlayValues[372] = d372
		ps780.OverlayValues[373] = d373
		ps780.OverlayValues[374] = d374
		ps780.OverlayValues[375] = d375
		ps780.OverlayValues[376] = d376
		ps780.OverlayValues[377] = d377
		ps780.OverlayValues[378] = d378
		ps780.OverlayValues[379] = d379
		ps780.OverlayValues[380] = d380
		ps780.OverlayValues[381] = d381
		ps780.OverlayValues[382] = d382
		ps780.OverlayValues[383] = d383
		ps780.OverlayValues[384] = d384
		ps780.OverlayValues[385] = d385
		ps780.OverlayValues[386] = d386
		ps780.OverlayValues[526] = d526
		ps780.OverlayValues[527] = d527
		ps780.OverlayValues[528] = d528
		ps780.OverlayValues[530] = d530
		ps780.OverlayValues[531] = d531
		ps780.OverlayValues[532] = d532
		ps780.OverlayValues[533] = d533
		ps780.OverlayValues[534] = d534
		ps780.OverlayValues[535] = d535
		ps780.OverlayValues[536] = d536
		ps780.OverlayValues[538] = d538
		ps780.OverlayValues[540] = d540
		ps780.OverlayValues[541] = d541
		ps780.OverlayValues[542] = d542
		ps780.OverlayValues[543] = d543
		ps780.OverlayValues[546] = d546
		ps780.OverlayValues[698] = d698
		ps780.OverlayValues[699] = d699
		ps780.OverlayValues[700] = d700
		ps780.OverlayValues[701] = d701
		ps780.OverlayValues[703] = d703
		ps780.OverlayValues[704] = d704
		ps780.OverlayValues[705] = d705
		ps780.OverlayValues[706] = d706
		ps780.OverlayValues[707] = d707
		ps780.OverlayValues[708] = d708
		ps780.OverlayValues[709] = d709
		ps780.OverlayValues[710] = d710
		ps780.OverlayValues[711] = d711
		ps780.OverlayValues[712] = d712
		ps780.OverlayValues[714] = d714
		ps780.OverlayValues[715] = d715
		ps780.OverlayValues[716] = d716
		ps780.OverlayValues[717] = d717
		ps780.OverlayValues[718] = d718
		ps780.OverlayValues[719] = d719
		ps780.OverlayValues[720] = d720
		ps780.OverlayValues[721] = d721
		ps780.OverlayValues[722] = d722
		ps780.OverlayValues[723] = d723
		ps780.OverlayValues[724] = d724
		ps780.OverlayValues[725] = d725
		ps780.OverlayValues[726] = d726
		ps780.OverlayValues[727] = d727
		ps780.OverlayValues[728] = d728
		ps780.OverlayValues[729] = d729
		ps780.OverlayValues[730] = d730
		ps780.OverlayValues[731] = d731
		ps780.OverlayValues[732] = d732
		ps780.OverlayValues[733] = d733
		ps780.OverlayValues[734] = d734
		ps780.OverlayValues[735] = d735
		ps780.OverlayValues[736] = d736
		ps780.OverlayValues[737] = d737
		ps780.OverlayValues[738] = d738
		ps780.OverlayValues[739] = d739
		ps780.OverlayValues[740] = d740
		ps780.OverlayValues[741] = d741
		ps780.OverlayValues[742] = d742
		ps780.OverlayValues[743] = d743
		ps780.OverlayValues[744] = d744
		ps780.OverlayValues[745] = d745
		ps780.OverlayValues[746] = d746
		ps780.OverlayValues[747] = d747
		ps780.OverlayValues[748] = d748
		ps780.OverlayValues[749] = d749
		ps780.OverlayValues[750] = d750
		ps780.OverlayValues[751] = d751
		ps780.OverlayValues[752] = d752
		ps780.OverlayValues[753] = d753
		ps780.OverlayValues[754] = d754
		ps780.OverlayValues[755] = d755
		ps780.OverlayValues[756] = d756
		ps780.OverlayValues[757] = d757
		ps780.OverlayValues[758] = d758
		ps780.OverlayValues[759] = d759
		ps780.OverlayValues[760] = d760
		ps780.OverlayValues[761] = d761
		ps780.OverlayValues[762] = d762
		ps780.OverlayValues[763] = d763
		ps780.OverlayValues[764] = d764
		ps780.OverlayValues[765] = d765
		ps780.OverlayValues[766] = d766
		ps780.OverlayValues[767] = d767
		ps780.OverlayValues[768] = d768
		ps780.OverlayValues[769] = d769
		ps780.OverlayValues[770] = d770
		ps780.OverlayValues[771] = d771
		ps780.OverlayValues[772] = d772
		ps780.OverlayValues[773] = d773
		ps780.OverlayValues[774] = d774
		ps780.OverlayValues[775] = d775
		ps780.OverlayValues[776] = d776
		ps780.OverlayValues[777] = d777
		ps781 := scm.PhiState{General: true}
		ps781.OverlayValues = make([]scm.JITValueDesc, 778)
		ps781.OverlayValues[3] = d3
		ps781.OverlayValues[4] = d4
		ps781.OverlayValues[5] = d5
		ps781.OverlayValues[6] = d6
		ps781.OverlayValues[7] = d7
		ps781.OverlayValues[8] = d8
		ps781.OverlayValues[9] = d9
		ps781.OverlayValues[10] = d10
		ps781.OverlayValues[11] = d11
		ps781.OverlayValues[12] = d12
		ps781.OverlayValues[13] = d13
		ps781.OverlayValues[14] = d14
		ps781.OverlayValues[15] = d15
		ps781.OverlayValues[16] = d16
		ps781.OverlayValues[17] = d17
		ps781.OverlayValues[19] = d19
		ps781.OverlayValues[20] = d20
		ps781.OverlayValues[21] = d21
		ps781.OverlayValues[22] = d22
		ps781.OverlayValues[23] = d23
		ps781.OverlayValues[24] = d24
		ps781.OverlayValues[25] = d25
		ps781.OverlayValues[26] = d26
		ps781.OverlayValues[27] = d27
		ps781.OverlayValues[28] = d28
		ps781.OverlayValues[29] = d29
		ps781.OverlayValues[30] = d30
		ps781.OverlayValues[31] = d31
		ps781.OverlayValues[32] = d32
		ps781.OverlayValues[33] = d33
		ps781.OverlayValues[34] = d34
		ps781.OverlayValues[35] = d35
		ps781.OverlayValues[36] = d36
		ps781.OverlayValues[37] = d37
		ps781.OverlayValues[38] = d38
		ps781.OverlayValues[39] = d39
		ps781.OverlayValues[40] = d40
		ps781.OverlayValues[41] = d41
		ps781.OverlayValues[42] = d42
		ps781.OverlayValues[43] = d43
		ps781.OverlayValues[44] = d44
		ps781.OverlayValues[45] = d45
		ps781.OverlayValues[46] = d46
		ps781.OverlayValues[47] = d47
		ps781.OverlayValues[48] = d48
		ps781.OverlayValues[49] = d49
		ps781.OverlayValues[50] = d50
		ps781.OverlayValues[51] = d51
		ps781.OverlayValues[54] = d54
		ps781.OverlayValues[55] = d55
		ps781.OverlayValues[56] = d56
		ps781.OverlayValues[111] = d111
		ps781.OverlayValues[112] = d112
		ps781.OverlayValues[113] = d113
		ps781.OverlayValues[114] = d114
		ps781.OverlayValues[115] = d115
		ps781.OverlayValues[116] = d116
		ps781.OverlayValues[117] = d117
		ps781.OverlayValues[118] = d118
		ps781.OverlayValues[119] = d119
		ps781.OverlayValues[120] = d120
		ps781.OverlayValues[121] = d121
		ps781.OverlayValues[122] = d122
		ps781.OverlayValues[123] = d123
		ps781.OverlayValues[124] = d124
		ps781.OverlayValues[125] = d125
		ps781.OverlayValues[126] = d126
		ps781.OverlayValues[127] = d127
		ps781.OverlayValues[128] = d128
		ps781.OverlayValues[129] = d129
		ps781.OverlayValues[130] = d130
		ps781.OverlayValues[131] = d131
		ps781.OverlayValues[132] = d132
		ps781.OverlayValues[133] = d133
		ps781.OverlayValues[134] = d134
		ps781.OverlayValues[135] = d135
		ps781.OverlayValues[136] = d136
		ps781.OverlayValues[137] = d137
		ps781.OverlayValues[138] = d138
		ps781.OverlayValues[139] = d139
		ps781.OverlayValues[142] = d142
		ps781.OverlayValues[227] = d227
		ps781.OverlayValues[228] = d228
		ps781.OverlayValues[229] = d229
		ps781.OverlayValues[230] = d230
		ps781.OverlayValues[232] = d232
		ps781.OverlayValues[233] = d233
		ps781.OverlayValues[234] = d234
		ps781.OverlayValues[235] = d235
		ps781.OverlayValues[236] = d236
		ps781.OverlayValues[237] = d237
		ps781.OverlayValues[238] = d238
		ps781.OverlayValues[239] = d239
		ps781.OverlayValues[241] = d241
		ps781.OverlayValues[243] = d243
		ps781.OverlayValues[244] = d244
		ps781.OverlayValues[245] = d245
		ps781.OverlayValues[246] = d246
		ps781.OverlayValues[247] = d247
		ps781.OverlayValues[250] = d250
		ps781.OverlayValues[352] = d352
		ps781.OverlayValues[353] = d353
		ps781.OverlayValues[354] = d354
		ps781.OverlayValues[355] = d355
		ps781.OverlayValues[356] = d356
		ps781.OverlayValues[358] = d358
		ps781.OverlayValues[359] = d359
		ps781.OverlayValues[360] = d360
		ps781.OverlayValues[361] = d361
		ps781.OverlayValues[362] = d362
		ps781.OverlayValues[363] = d363
		ps781.OverlayValues[364] = d364
		ps781.OverlayValues[365] = d365
		ps781.OverlayValues[366] = d366
		ps781.OverlayValues[367] = d367
		ps781.OverlayValues[368] = d368
		ps781.OverlayValues[369] = d369
		ps781.OverlayValues[370] = d370
		ps781.OverlayValues[371] = d371
		ps781.OverlayValues[372] = d372
		ps781.OverlayValues[373] = d373
		ps781.OverlayValues[374] = d374
		ps781.OverlayValues[375] = d375
		ps781.OverlayValues[376] = d376
		ps781.OverlayValues[377] = d377
		ps781.OverlayValues[378] = d378
		ps781.OverlayValues[379] = d379
		ps781.OverlayValues[380] = d380
		ps781.OverlayValues[381] = d381
		ps781.OverlayValues[382] = d382
		ps781.OverlayValues[383] = d383
		ps781.OverlayValues[384] = d384
		ps781.OverlayValues[385] = d385
		ps781.OverlayValues[386] = d386
		ps781.OverlayValues[526] = d526
		ps781.OverlayValues[527] = d527
		ps781.OverlayValues[528] = d528
		ps781.OverlayValues[530] = d530
		ps781.OverlayValues[531] = d531
		ps781.OverlayValues[532] = d532
		ps781.OverlayValues[533] = d533
		ps781.OverlayValues[534] = d534
		ps781.OverlayValues[535] = d535
		ps781.OverlayValues[536] = d536
		ps781.OverlayValues[538] = d538
		ps781.OverlayValues[540] = d540
		ps781.OverlayValues[541] = d541
		ps781.OverlayValues[542] = d542
		ps781.OverlayValues[543] = d543
		ps781.OverlayValues[546] = d546
		ps781.OverlayValues[698] = d698
		ps781.OverlayValues[699] = d699
		ps781.OverlayValues[700] = d700
		ps781.OverlayValues[701] = d701
		ps781.OverlayValues[703] = d703
		ps781.OverlayValues[704] = d704
		ps781.OverlayValues[705] = d705
		ps781.OverlayValues[706] = d706
		ps781.OverlayValues[707] = d707
		ps781.OverlayValues[708] = d708
		ps781.OverlayValues[709] = d709
		ps781.OverlayValues[710] = d710
		ps781.OverlayValues[711] = d711
		ps781.OverlayValues[712] = d712
		ps781.OverlayValues[714] = d714
		ps781.OverlayValues[715] = d715
		ps781.OverlayValues[716] = d716
		ps781.OverlayValues[717] = d717
		ps781.OverlayValues[718] = d718
		ps781.OverlayValues[719] = d719
		ps781.OverlayValues[720] = d720
		ps781.OverlayValues[721] = d721
		ps781.OverlayValues[722] = d722
		ps781.OverlayValues[723] = d723
		ps781.OverlayValues[724] = d724
		ps781.OverlayValues[725] = d725
		ps781.OverlayValues[726] = d726
		ps781.OverlayValues[727] = d727
		ps781.OverlayValues[728] = d728
		ps781.OverlayValues[729] = d729
		ps781.OverlayValues[730] = d730
		ps781.OverlayValues[731] = d731
		ps781.OverlayValues[732] = d732
		ps781.OverlayValues[733] = d733
		ps781.OverlayValues[734] = d734
		ps781.OverlayValues[735] = d735
		ps781.OverlayValues[736] = d736
		ps781.OverlayValues[737] = d737
		ps781.OverlayValues[738] = d738
		ps781.OverlayValues[739] = d739
		ps781.OverlayValues[740] = d740
		ps781.OverlayValues[741] = d741
		ps781.OverlayValues[742] = d742
		ps781.OverlayValues[743] = d743
		ps781.OverlayValues[744] = d744
		ps781.OverlayValues[745] = d745
		ps781.OverlayValues[746] = d746
		ps781.OverlayValues[747] = d747
		ps781.OverlayValues[748] = d748
		ps781.OverlayValues[749] = d749
		ps781.OverlayValues[750] = d750
		ps781.OverlayValues[751] = d751
		ps781.OverlayValues[752] = d752
		ps781.OverlayValues[753] = d753
		ps781.OverlayValues[754] = d754
		ps781.OverlayValues[755] = d755
		ps781.OverlayValues[756] = d756
		ps781.OverlayValues[757] = d757
		ps781.OverlayValues[758] = d758
		ps781.OverlayValues[759] = d759
		ps781.OverlayValues[760] = d760
		ps781.OverlayValues[761] = d761
		ps781.OverlayValues[762] = d762
		ps781.OverlayValues[763] = d763
		ps781.OverlayValues[764] = d764
		ps781.OverlayValues[765] = d765
		ps781.OverlayValues[766] = d766
		ps781.OverlayValues[767] = d767
		ps781.OverlayValues[768] = d768
		ps781.OverlayValues[769] = d769
		ps781.OverlayValues[770] = d770
		ps781.OverlayValues[771] = d771
		ps781.OverlayValues[772] = d772
		ps781.OverlayValues[773] = d773
		ps781.OverlayValues[774] = d774
		ps781.OverlayValues[775] = d775
		ps781.OverlayValues[776] = d776
		ps781.OverlayValues[777] = d777
		snap782 := d3
		snap783 := d4
		snap784 := d5
		snap785 := d6
		snap786 := d7
		snap787 := d8
		snap788 := d9
		snap789 := d10
		snap790 := d11
		snap791 := d12
		snap792 := d13
		snap793 := d14
		snap794 := d15
		snap795 := d16
		snap796 := d17
		snap797 := d19
		snap798 := d20
		snap799 := d21
		snap800 := d22
		snap801 := d23
		snap802 := d24
		snap803 := d25
		snap804 := d26
		snap805 := d27
		snap806 := d28
		snap807 := d29
		snap808 := d30
		snap809 := d31
		snap810 := d32
		snap811 := d33
		snap812 := d34
		snap813 := d35
		snap814 := d36
		snap815 := d37
		snap816 := d38
		snap817 := d39
		snap818 := d40
		snap819 := d41
		snap820 := d42
		snap821 := d43
		snap822 := d44
		snap823 := d45
		snap824 := d46
		snap825 := d47
		snap826 := d48
		snap827 := d49
		snap828 := d50
		snap829 := d51
		snap830 := d54
		snap831 := d55
		snap832 := d56
		snap833 := d111
		snap834 := d112
		snap835 := d113
		snap836 := d114
		snap837 := d115
		snap838 := d116
		snap839 := d117
		snap840 := d118
		snap841 := d119
		snap842 := d120
		snap843 := d121
		snap844 := d122
		snap845 := d123
		snap846 := d124
		snap847 := d125
		snap848 := d126
		snap849 := d127
		snap850 := d128
		snap851 := d129
		snap852 := d130
		snap853 := d131
		snap854 := d132
		snap855 := d133
		snap856 := d134
		snap857 := d135
		snap858 := d136
		snap859 := d137
		snap860 := d138
		snap861 := d139
		snap862 := d142
		snap863 := d227
		snap864 := d228
		snap865 := d229
		snap866 := d230
		snap867 := d232
		snap868 := d233
		snap869 := d234
		snap870 := d235
		snap871 := d236
		snap872 := d237
		snap873 := d238
		snap874 := d239
		snap875 := d241
		snap876 := d243
		snap877 := d244
		snap878 := d245
		snap879 := d246
		snap880 := d247
		snap881 := d250
		snap882 := d352
		snap883 := d353
		snap884 := d354
		snap885 := d355
		snap886 := d356
		snap887 := d358
		snap888 := d359
		snap889 := d360
		snap890 := d361
		snap891 := d362
		snap892 := d363
		snap893 := d364
		snap894 := d365
		snap895 := d366
		snap896 := d367
		snap897 := d368
		snap898 := d369
		snap899 := d370
		snap900 := d371
		snap901 := d372
		snap902 := d373
		snap903 := d374
		snap904 := d375
		snap905 := d376
		snap906 := d377
		snap907 := d378
		snap908 := d379
		snap909 := d380
		snap910 := d381
		snap911 := d382
		snap912 := d383
		snap913 := d384
		snap914 := d385
		snap915 := d386
		snap916 := d526
		snap917 := d527
		snap918 := d528
		snap919 := d530
		snap920 := d531
		snap921 := d532
		snap922 := d533
		snap923 := d534
		snap924 := d535
		snap925 := d536
		snap926 := d538
		snap927 := d540
		snap928 := d541
		snap929 := d542
		snap930 := d543
		snap931 := d546
		snap932 := d698
		snap933 := d699
		snap934 := d700
		snap935 := d701
		snap936 := d703
		snap937 := d704
		snap938 := d705
		snap939 := d706
		snap940 := d707
		snap941 := d708
		snap942 := d709
		snap943 := d710
		snap944 := d711
		snap945 := d712
		snap946 := d714
		snap947 := d715
		snap948 := d716
		snap949 := d717
		snap950 := d718
		snap951 := d719
		snap952 := d720
		snap953 := d721
		snap954 := d722
		snap955 := d723
		snap956 := d724
		snap957 := d725
		snap958 := d726
		snap959 := d727
		snap960 := d728
		snap961 := d729
		snap962 := d730
		snap963 := d731
		snap964 := d732
		snap965 := d733
		snap966 := d734
		snap967 := d735
		snap968 := d736
		snap969 := d737
		snap970 := d738
		snap971 := d739
		snap972 := d740
		snap973 := d741
		snap974 := d742
		snap975 := d743
		snap976 := d744
		snap977 := d745
		snap978 := d746
		snap979 := d747
		snap980 := d748
		snap981 := d749
		snap982 := d750
		snap983 := d751
		snap984 := d752
		snap985 := d753
		snap986 := d754
		snap987 := d755
		snap988 := d756
		snap989 := d757
		snap990 := d758
		snap991 := d759
		snap992 := d760
		snap993 := d761
		snap994 := d762
		snap995 := d763
		snap996 := d764
		snap997 := d765
		snap998 := d766
		snap999 := d767
		snap1000 := d768
		snap1001 := d769
		snap1002 := d770
		snap1003 := d771
		snap1004 := d772
		snap1005 := d773
		snap1006 := d774
		snap1007 := d775
		snap1008 := d776
		snap1009 := d777
		alloc1010 := ctx.SnapshotAllocState()
		if !bbs[12].Rendered {
			bbs[12].RenderPS(ps781)
		}
		ctx.RestoreAllocState(alloc1010)
		d3 = snap782
		d4 = snap783
		d5 = snap784
		d6 = snap785
		d7 = snap786
		d8 = snap787
		d9 = snap788
		d10 = snap789
		d11 = snap790
		d12 = snap791
		d13 = snap792
		d14 = snap793
		d15 = snap794
		d16 = snap795
		d17 = snap796
		d19 = snap797
		d20 = snap798
		d21 = snap799
		d22 = snap800
		d23 = snap801
		d24 = snap802
		d25 = snap803
		d26 = snap804
		d27 = snap805
		d28 = snap806
		d29 = snap807
		d30 = snap808
		d31 = snap809
		d32 = snap810
		d33 = snap811
		d34 = snap812
		d35 = snap813
		d36 = snap814
		d37 = snap815
		d38 = snap816
		d39 = snap817
		d40 = snap818
		d41 = snap819
		d42 = snap820
		d43 = snap821
		d44 = snap822
		d45 = snap823
		d46 = snap824
		d47 = snap825
		d48 = snap826
		d49 = snap827
		d50 = snap828
		d51 = snap829
		d54 = snap830
		d55 = snap831
		d56 = snap832
		d111 = snap833
		d112 = snap834
		d113 = snap835
		d114 = snap836
		d115 = snap837
		d116 = snap838
		d117 = snap839
		d118 = snap840
		d119 = snap841
		d120 = snap842
		d121 = snap843
		d122 = snap844
		d123 = snap845
		d124 = snap846
		d125 = snap847
		d126 = snap848
		d127 = snap849
		d128 = snap850
		d129 = snap851
		d130 = snap852
		d131 = snap853
		d132 = snap854
		d133 = snap855
		d134 = snap856
		d135 = snap857
		d136 = snap858
		d137 = snap859
		d138 = snap860
		d139 = snap861
		d142 = snap862
		d227 = snap863
		d228 = snap864
		d229 = snap865
		d230 = snap866
		d232 = snap867
		d233 = snap868
		d234 = snap869
		d235 = snap870
		d236 = snap871
		d237 = snap872
		d238 = snap873
		d239 = snap874
		d241 = snap875
		d243 = snap876
		d244 = snap877
		d245 = snap878
		d246 = snap879
		d247 = snap880
		d250 = snap881
		d352 = snap882
		d353 = snap883
		d354 = snap884
		d355 = snap885
		d356 = snap886
		d358 = snap887
		d359 = snap888
		d360 = snap889
		d361 = snap890
		d362 = snap891
		d363 = snap892
		d364 = snap893
		d365 = snap894
		d366 = snap895
		d367 = snap896
		d368 = snap897
		d369 = snap898
		d370 = snap899
		d371 = snap900
		d372 = snap901
		d373 = snap902
		d374 = snap903
		d375 = snap904
		d376 = snap905
		d377 = snap906
		d378 = snap907
		d379 = snap908
		d380 = snap909
		d381 = snap910
		d382 = snap911
		d383 = snap912
		d384 = snap913
		d385 = snap914
		d386 = snap915
		d526 = snap916
		d527 = snap917
		d528 = snap918
		d530 = snap919
		d531 = snap920
		d532 = snap921
		d533 = snap922
		d534 = snap923
		d535 = snap924
		d536 = snap925
		d538 = snap926
		d540 = snap927
		d541 = snap928
		d542 = snap929
		d543 = snap930
		d546 = snap931
		d698 = snap932
		d699 = snap933
		d700 = snap934
		d701 = snap935
		d703 = snap936
		d704 = snap937
		d705 = snap938
		d706 = snap939
		d707 = snap940
		d708 = snap941
		d709 = snap942
		d710 = snap943
		d711 = snap944
		d712 = snap945
		d714 = snap946
		d715 = snap947
		d716 = snap948
		d717 = snap949
		d718 = snap950
		d719 = snap951
		d720 = snap952
		d721 = snap953
		d722 = snap954
		d723 = snap955
		d724 = snap956
		d725 = snap957
		d726 = snap958
		d727 = snap959
		d728 = snap960
		d729 = snap961
		d730 = snap962
		d731 = snap963
		d732 = snap964
		d733 = snap965
		d734 = snap966
		d735 = snap967
		d736 = snap968
		d737 = snap969
		d738 = snap970
		d739 = snap971
		d740 = snap972
		d741 = snap973
		d742 = snap974
		d743 = snap975
		d744 = snap976
		d745 = snap977
		d746 = snap978
		d747 = snap979
		d748 = snap980
		d749 = snap981
		d750 = snap982
		d751 = snap983
		d752 = snap984
		d753 = snap985
		d754 = snap986
		d755 = snap987
		d756 = snap988
		d757 = snap989
		d758 = snap990
		d759 = snap991
		d760 = snap992
		d761 = snap993
		d762 = snap994
		d763 = snap995
		d764 = snap996
		d765 = snap997
		d766 = snap998
		d767 = snap999
		d768 = snap1000
		d769 = snap1001
		d770 = snap1002
		d771 = snap1003
		d772 = snap1004
		d773 = snap1005
		d774 = snap1006
		d775 = snap1007
		d776 = snap1008
		d777 = snap1009
		if !bbs[11].Rendered {
			return bbs[11].RenderPS(ps780)
		}
		return result
		ctx.FreeDesc(&d776)
		return result
	}
	ps1011 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps1011)
	ctx.MarkLabel(lbl0)
	d1012 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
	ctx.BindReg(r1, &d1012)
	ctx.BindReg(r2, &d1012)
	ctx.EmitMovPairToResult(&d1012, &result)
	ctx.FreeReg(r1)
	ctx.FreeReg(r2)
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
