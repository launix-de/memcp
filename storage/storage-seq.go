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
	var phiBase23 int32
	_ = phiBase23
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
	var d57 scm.JITValueDesc
	_ = d57
	var d58 scm.JITValueDesc
	_ = d58
	var d59 scm.JITValueDesc
	_ = d59
	var d62 scm.JITValueDesc
	_ = d62
	var d63 scm.JITValueDesc
	_ = d63
	var d64 scm.JITValueDesc
	_ = d64
	var d128 scm.JITValueDesc
	_ = d128
	var d129 scm.JITValueDesc
	_ = d129
	var d130 scm.JITValueDesc
	_ = d130
	var phiBase131 int32
	_ = phiBase131
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
	var d148 scm.JITValueDesc
	_ = d148
	var d149 scm.JITValueDesc
	_ = d149
	var d150 scm.JITValueDesc
	_ = d150
	var d151 scm.JITValueDesc
	_ = d151
	var d152 scm.JITValueDesc
	_ = d152
	var d153 scm.JITValueDesc
	_ = d153
	var d154 scm.JITValueDesc
	_ = d154
	var d155 scm.JITValueDesc
	_ = d155
	var d156 scm.JITValueDesc
	_ = d156
	var d157 scm.JITValueDesc
	_ = d157
	var d158 scm.JITValueDesc
	_ = d158
	var d159 scm.JITValueDesc
	_ = d159
	var d160 scm.JITValueDesc
	_ = d160
	var d161 scm.JITValueDesc
	_ = d161
	var d162 scm.JITValueDesc
	_ = d162
	var d163 scm.JITValueDesc
	_ = d163
	var d164 scm.JITValueDesc
	_ = d164
	var d165 scm.JITValueDesc
	_ = d165
	var d166 scm.JITValueDesc
	_ = d166
	var d169 scm.JITValueDesc
	_ = d169
	var d272 scm.JITValueDesc
	_ = d272
	var d273 scm.JITValueDesc
	_ = d273
	var d274 scm.JITValueDesc
	_ = d274
	var d275 scm.JITValueDesc
	_ = d275
	var d277 scm.JITValueDesc
	_ = d277
	var d278 scm.JITValueDesc
	_ = d278
	var d279 scm.JITValueDesc
	_ = d279
	var d280 scm.JITValueDesc
	_ = d280
	var d281 scm.JITValueDesc
	_ = d281
	var d282 scm.JITValueDesc
	_ = d282
	var d283 scm.JITValueDesc
	_ = d283
	var d284 scm.JITValueDesc
	_ = d284
	var d286 scm.JITValueDesc
	_ = d286
	var d288 scm.JITValueDesc
	_ = d288
	var d289 scm.JITValueDesc
	_ = d289
	var d290 scm.JITValueDesc
	_ = d290
	var d291 scm.JITValueDesc
	_ = d291
	var d292 scm.JITValueDesc
	_ = d292
	var d295 scm.JITValueDesc
	_ = d295
	var d415 scm.JITValueDesc
	_ = d415
	var d416 scm.JITValueDesc
	_ = d416
	var d417 scm.JITValueDesc
	_ = d417
	var d418 scm.JITValueDesc
	_ = d418
	var d419 scm.JITValueDesc
	_ = d419
	var d421 scm.JITValueDesc
	_ = d421
	var d422 scm.JITValueDesc
	_ = d422
	var d423 scm.JITValueDesc
	_ = d423
	var phiBase424 int32
	_ = phiBase424
	var d425 scm.JITValueDesc
	_ = d425
	var d426 scm.JITValueDesc
	_ = d426
	var d427 scm.JITValueDesc
	_ = d427
	var d428 scm.JITValueDesc
	_ = d428
	var d429 scm.JITValueDesc
	_ = d429
	var d430 scm.JITValueDesc
	_ = d430
	var d431 scm.JITValueDesc
	_ = d431
	var d432 scm.JITValueDesc
	_ = d432
	var d433 scm.JITValueDesc
	_ = d433
	var d434 scm.JITValueDesc
	_ = d434
	var d435 scm.JITValueDesc
	_ = d435
	var d436 scm.JITValueDesc
	_ = d436
	var d437 scm.JITValueDesc
	_ = d437
	var d438 scm.JITValueDesc
	_ = d438
	var d439 scm.JITValueDesc
	_ = d439
	var d440 scm.JITValueDesc
	_ = d440
	var d441 scm.JITValueDesc
	_ = d441
	var d442 scm.JITValueDesc
	_ = d442
	var d443 scm.JITValueDesc
	_ = d443
	var d444 scm.JITValueDesc
	_ = d444
	var d445 scm.JITValueDesc
	_ = d445
	var d446 scm.JITValueDesc
	_ = d446
	var d447 scm.JITValueDesc
	_ = d447
	var d448 scm.JITValueDesc
	_ = d448
	var d449 scm.JITValueDesc
	_ = d449
	var d450 scm.JITValueDesc
	_ = d450
	var d451 scm.JITValueDesc
	_ = d451
	var d452 scm.JITValueDesc
	_ = d452
	var d453 scm.JITValueDesc
	_ = d453
	var d454 scm.JITValueDesc
	_ = d454
	var d455 scm.JITValueDesc
	_ = d455
	var d456 scm.JITValueDesc
	_ = d456
	var d457 scm.JITValueDesc
	_ = d457
	var d458 scm.JITValueDesc
	_ = d458
	var d459 scm.JITValueDesc
	_ = d459
	var d626 scm.JITValueDesc
	_ = d626
	var d627 scm.JITValueDesc
	_ = d627
	var d628 scm.JITValueDesc
	_ = d628
	var d630 scm.JITValueDesc
	_ = d630
	var d631 scm.JITValueDesc
	_ = d631
	var d632 scm.JITValueDesc
	_ = d632
	var d633 scm.JITValueDesc
	_ = d633
	var d634 scm.JITValueDesc
	_ = d634
	var d635 scm.JITValueDesc
	_ = d635
	var d636 scm.JITValueDesc
	_ = d636
	var d638 scm.JITValueDesc
	_ = d638
	var d640 scm.JITValueDesc
	_ = d640
	var d641 scm.JITValueDesc
	_ = d641
	var d642 scm.JITValueDesc
	_ = d642
	var d643 scm.JITValueDesc
	_ = d643
	var d646 scm.JITValueDesc
	_ = d646
	var d825 scm.JITValueDesc
	_ = d825
	var d826 scm.JITValueDesc
	_ = d826
	var d827 scm.JITValueDesc
	_ = d827
	var d828 scm.JITValueDesc
	_ = d828
	var d830 scm.JITValueDesc
	_ = d830
	var d831 scm.JITValueDesc
	_ = d831
	var d832 scm.JITValueDesc
	_ = d832
	var d833 scm.JITValueDesc
	_ = d833
	var d834 scm.JITValueDesc
	_ = d834
	var d835 scm.JITValueDesc
	_ = d835
	var d836 scm.JITValueDesc
	_ = d836
	var d837 scm.JITValueDesc
	_ = d837
	var d839 scm.JITValueDesc
	_ = d839
	var d840 scm.JITValueDesc
	_ = d840
	var d841 scm.JITValueDesc
	_ = d841
	var d842 scm.JITValueDesc
	_ = d842
	var d843 scm.JITValueDesc
	_ = d843
	var phiBase844 int32
	_ = phiBase844
	var d845 scm.JITValueDesc
	_ = d845
	var d846 scm.JITValueDesc
	_ = d846
	var d847 scm.JITValueDesc
	_ = d847
	var d848 scm.JITValueDesc
	_ = d848
	var d849 scm.JITValueDesc
	_ = d849
	var d850 scm.JITValueDesc
	_ = d850
	var d851 scm.JITValueDesc
	_ = d851
	var d852 scm.JITValueDesc
	_ = d852
	var d853 scm.JITValueDesc
	_ = d853
	var d854 scm.JITValueDesc
	_ = d854
	var d855 scm.JITValueDesc
	_ = d855
	var d856 scm.JITValueDesc
	_ = d856
	var d857 scm.JITValueDesc
	_ = d857
	var d858 scm.JITValueDesc
	_ = d858
	var d859 scm.JITValueDesc
	_ = d859
	var d860 scm.JITValueDesc
	_ = d860
	var d861 scm.JITValueDesc
	_ = d861
	var d862 scm.JITValueDesc
	_ = d862
	var d863 scm.JITValueDesc
	_ = d863
	var d864 scm.JITValueDesc
	_ = d864
	var d865 scm.JITValueDesc
	_ = d865
	var d866 scm.JITValueDesc
	_ = d866
	var d867 scm.JITValueDesc
	_ = d867
	var d868 scm.JITValueDesc
	_ = d868
	var d869 scm.JITValueDesc
	_ = d869
	var d870 scm.JITValueDesc
	_ = d870
	var d871 scm.JITValueDesc
	_ = d871
	var d872 scm.JITValueDesc
	_ = d872
	var d873 scm.JITValueDesc
	_ = d873
	var d874 scm.JITValueDesc
	_ = d874
	var d875 scm.JITValueDesc
	_ = d875
	var d876 scm.JITValueDesc
	_ = d876
	var d877 scm.JITValueDesc
	_ = d877
	var d878 scm.JITValueDesc
	_ = d878
	var phiBase879 int32
	_ = phiBase879
	var d880 scm.JITValueDesc
	_ = d880
	var d881 scm.JITValueDesc
	_ = d881
	var d882 scm.JITValueDesc
	_ = d882
	var d883 scm.JITValueDesc
	_ = d883
	var d884 scm.JITValueDesc
	_ = d884
	var d885 scm.JITValueDesc
	_ = d885
	var d886 scm.JITValueDesc
	_ = d886
	var d887 scm.JITValueDesc
	_ = d887
	var d888 scm.JITValueDesc
	_ = d888
	var d889 scm.JITValueDesc
	_ = d889
	var d890 scm.JITValueDesc
	_ = d890
	var d891 scm.JITValueDesc
	_ = d891
	var d892 scm.JITValueDesc
	_ = d892
	var d893 scm.JITValueDesc
	_ = d893
	var d894 scm.JITValueDesc
	_ = d894
	var d895 scm.JITValueDesc
	_ = d895
	var d896 scm.JITValueDesc
	_ = d896
	var d897 scm.JITValueDesc
	_ = d897
	var d898 scm.JITValueDesc
	_ = d898
	var d899 scm.JITValueDesc
	_ = d899
	var d900 scm.JITValueDesc
	_ = d900
	var d901 scm.JITValueDesc
	_ = d901
	var d902 scm.JITValueDesc
	_ = d902
	var d903 scm.JITValueDesc
	_ = d903
	var d904 scm.JITValueDesc
	_ = d904
	var d905 scm.JITValueDesc
	_ = d905
	var d906 scm.JITValueDesc
	_ = d906
	var d907 scm.JITValueDesc
	_ = d907
	var d908 scm.JITValueDesc
	_ = d908
	var d909 scm.JITValueDesc
	_ = d909
	var d910 scm.JITValueDesc
	_ = d910
	var d911 scm.JITValueDesc
	_ = d911
	var d912 scm.JITValueDesc
	_ = d912
	var d913 scm.JITValueDesc
	_ = d913
	var d914 scm.JITValueDesc
	_ = d914
	var d915 scm.JITValueDesc
	_ = d915
	var d916 scm.JITValueDesc
	_ = d916
	var d917 scm.JITValueDesc
	_ = d917
	var d918 scm.JITValueDesc
	_ = d918
	var d919 scm.JITValueDesc
	_ = d919
	var d920 scm.JITValueDesc
	_ = d920
	var d921 scm.JITValueDesc
	_ = d921
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
	r0 := ctx.AllocReg()
	r1 := ctx.AllocRegExcept(r0)
	lbl0 := ctx.ReserveLabel()
	bbpos_0_0 := int32(-1)
	_ = bbpos_0_0
	lbl1 := ctx.ReserveLabel()
	bbpos_0_1 := int32(-1)
	_ = bbpos_0_1
	lbl2 := ctx.ReserveLabel()
	bbpos_0_2 := int32(-1)
	_ = bbpos_0_2
	lbl3 := ctx.ReserveLabel()
	bbpos_0_3 := int32(-1)
	_ = bbpos_0_3
	lbl4 := ctx.ReserveLabel()
	bbpos_0_4 := int32(-1)
	_ = bbpos_0_4
	lbl5 := ctx.ReserveLabel()
	bbpos_0_5 := int32(-1)
	_ = bbpos_0_5
	lbl6 := ctx.ReserveLabel()
	bbpos_0_6 := int32(-1)
	_ = bbpos_0_6
	lbl7 := ctx.ReserveLabel()
	bbpos_0_7 := int32(-1)
	_ = bbpos_0_7
	lbl8 := ctx.ReserveLabel()
	bbpos_0_8 := int32(-1)
	_ = bbpos_0_8
	lbl9 := ctx.ReserveLabel()
	bbpos_0_9 := int32(-1)
	_ = bbpos_0_9
	lbl10 := ctx.ReserveLabel()
	bbpos_0_10 := int32(-1)
	_ = bbpos_0_10
	lbl11 := ctx.ReserveLabel()
	bbpos_0_11 := int32(-1)
	_ = bbpos_0_11
	lbl12 := ctx.ReserveLabel()
	bbpos_0_12 := int32(-1)
	_ = bbpos_0_12
	lbl13 := ctx.ReserveLabel()
	bbpos_0_13 := int32(-1)
	_ = bbpos_0_13
	lbl14 := ctx.ReserveLabel()
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
		r6 := d1.Loc == scm.LocReg || d1.Loc == scm.LocRegPair || d1.Loc == scm.LocRegTriple
		r7 := d1.Reg
		if r6 {
			ctx.ProtectReg(r7)
		}
		r8 := d1.Loc == scm.LocRegPair || d1.Loc == scm.LocRegTriple
		r9 := d1.Reg2
		if r8 {
			ctx.ProtectReg(r9)
		}
		r10 := d1.Loc == scm.LocRegTriple
		r11 := d1.Reg3
		if r10 {
			ctx.ProtectReg(r11)
		}
		phiBase23 = ctx.AllocStack(int32(16))
		d24 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase23) + int32(0)}
		_ = d24
		lbl15 := ctx.ReserveLabel()
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		bbpos_1_1 := int32(-1)
		_ = bbpos_1_1
		bbpos_1_2 := int32(-1)
		_ = bbpos_1_2
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		d24 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d22)
		ctx.EnsureDesc(&d22)
		var d25 scm.JITValueDesc
		if d22.Loc == scm.LocImm {
			d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d22.Imm.Int()))))}
		} else {
			r12 := ctx.AllocReg()
			ctx.EmitMovRegReg(r12, d22.Reg)
			ctx.EmitShlRegImm8(r12, 32)
			ctx.EmitShrRegImm8(r12, 32)
			d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
			ctx.BindReg(r12, &d25)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d26 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			r13 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r13, fieldAddr)
			d26 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r13}
			ctx.BindReg(r13, &d26)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			r14 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r14, thisptr.Reg, off)
			d26 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r14}
			ctx.BindReg(r14, &d26)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d26)
		ctx.EnsureDesc(&d26)
		var d27 scm.JITValueDesc
		if d26.Loc == scm.LocImm {
			d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d26.Imm.Int()))))}
		} else {
			r15 := ctx.AllocReg()
			ctx.EmitMovRegReg(r15, d26.Reg)
			ctx.EmitShlRegImm8(r15, 56)
			ctx.EmitShrRegImm8(r15, 56)
			d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
			ctx.BindReg(r15, &d27)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d25)
		ctx.EnsureDesc(&d27)
		ctx.EnsureDesc(&d25)
		ctx.ProtectReg(d25.Reg)
		ctx.EnsureDesc(&d27)
		ctx.UnprotectReg(d25.Reg)
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
			r16 := ctx.AllocRegExcept(d25.Reg, d27.Reg)
			ctx.EmitMovRegReg(r16, d25.Reg)
			ctx.EmitImulInt64(r16, d27.Reg)
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
			ctx.BindReg(r16, &d28)
		}
		if d28.Loc == scm.LocReg && d25.Loc == scm.LocReg && d28.Reg == d25.Reg {
			ctx.TransferReg(d25.Reg)
			d25.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d28)
		ctx.FreeDesc(&d25)
		ctx.FreeDesc(&d27)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d29 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
			r17 := ctx.AllocReg()
			r18 := ctx.AllocRegExcept(r17)
			r19 := ctx.AllocRegExcept(r17, r18)
			ctx.EmitMovRegMem64(r17, fieldAddr)
			ctx.EmitMovRegMem64(r18, fieldAddr+8)
			ctx.EmitMovRegMem64(r19, fieldAddr+16)
			d29 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r17, Reg2: r18, Reg3: r19}
			ctx.BindReg(r17, &d29)
			ctx.BindReg(r18, &d29)
			ctx.BindReg(r19, &d29)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
			r20 := ctx.AllocReg()
			r21 := ctx.AllocRegExcept(r20)
			r22 := ctx.AllocRegExcept(r20, r21)
			ctx.EmitMovRegMem(r20, thisptr.Reg, off)
			ctx.EmitMovRegMem(r21, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r22, thisptr.Reg, off+16)
			d29 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r20, Reg2: r21, Reg3: r22}
			ctx.BindReg(r20, &d29)
			ctx.BindReg(r21, &d29)
			ctx.BindReg(r22, &d29)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d28)
		var d30 scm.JITValueDesc
		if d28.Loc == scm.LocImm {
			d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d28.Imm.Int() / 64)}
		} else {
			r23 := ctx.AllocRegExcept(d28.Reg)
			ctx.EmitMovRegReg(r23, d28.Reg)
			ctx.EmitShrRegImm8(r23, 6)
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
			ctx.BindReg(r23, &d30)
		}
		if d30.Loc == scm.LocReg && d28.Loc == scm.LocReg && d30.Reg == d28.Reg {
			ctx.TransferReg(d28.Reg)
			d28.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d30)
		ctx.ReclaimUntrackedRegs()
		d32 = ctx.EmitSliceElementAddress(&d29, &d30, 8)
		ctx.EnsureDesc(&d32)
		ctx.EmitMovRegMem(d32.Reg, d32.Reg, 0)
		d31 = d32
		ctx.FreeDesc(&d30)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d28)
		var d33 scm.JITValueDesc
		if d28.Loc == scm.LocImm {
			d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d28.Imm.Int() % 64)}
		} else {
			r24 := ctx.AllocRegExcept(d28.Reg)
			ctx.EmitMovRegReg(r24, d28.Reg)
			ctx.EmitAndRegImm32(r24, 63)
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d33)
		}
		if d33.Loc == scm.LocReg && d28.Loc == scm.LocReg && d33.Reg == d28.Reg {
			ctx.TransferReg(d28.Reg)
			d28.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d31)
		ctx.EnsureDesc(&d33)
		var d34 scm.JITValueDesc
		if d31.Loc == scm.LocImm && d33.Loc == scm.LocImm {
			d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d31.Imm.Int()) << uint64(d33.Imm.Int())))}
		} else if d33.Loc == scm.LocImm {
			r25 := ctx.AllocRegExcept(d31.Reg)
			ctx.EmitMovRegReg(r25, d31.Reg)
			ctx.EmitShlRegImm8(r25, uint8(d33.Imm.Int()))
			d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
			ctx.BindReg(r25, &d34)
		} else {
			{
				shiftSrc := d31.Reg
				r26 := ctx.AllocRegExcept(d31.Reg)
				ctx.EmitMovRegReg(r26, d31.Reg)
				shiftSrc = r26
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d33.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d33.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d33.Reg)
				}
				ctx.EmitShlRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d34)
			}
		}
		if d34.Loc == scm.LocReg && d31.Loc == scm.LocReg && d34.Reg == d31.Reg {
			ctx.TransferReg(d31.Reg)
			d31.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d34)
		ctx.EmitStoreToStack(d34, int32(phiBase23)+int32(0))
		ctx.StabilizeDescForControlFlow(&d34)
		ctx.FreeDesc(&d31)
		ctx.FreeDesc(&d33)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d28)
		var d35 scm.JITValueDesc
		if d28.Loc == scm.LocImm {
			d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d28.Imm.Int() % 64)}
		} else {
			r27 := ctx.AllocRegExcept(d28.Reg)
			ctx.EmitMovRegReg(r27, d28.Reg)
			ctx.EmitAndRegImm32(r27, 63)
			d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
			ctx.BindReg(r27, &d35)
		}
		if d35.Loc == scm.LocReg && d28.Loc == scm.LocReg && d35.Reg == d28.Reg {
			ctx.TransferReg(d28.Reg)
			d28.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d26)
		ctx.EnsureDesc(&d26)
		var d36 scm.JITValueDesc
		if d26.Loc == scm.LocImm {
			d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d26.Imm.Int()))))}
		} else {
			r28 := ctx.AllocReg()
			ctx.EmitMovRegReg(r28, d26.Reg)
			ctx.EmitShlRegImm8(r28, 56)
			ctx.EmitShrRegImm8(r28, 56)
			d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
			ctx.BindReg(r28, &d36)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d35)
		ctx.EnsureDesc(&d36)
		ctx.EnsureDesc(&d35)
		ctx.ProtectReg(d35.Reg)
		ctx.EnsureDesc(&d36)
		ctx.UnprotectReg(d35.Reg)
		var d37 scm.JITValueDesc
		if d35.Loc == scm.LocImm && d36.Loc == scm.LocImm {
			d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d35.Imm.Int() + d36.Imm.Int())}
		} else if d36.Loc == scm.LocImm && d36.Imm.Int() == 0 {
			r29 := ctx.AllocRegExcept(d35.Reg)
			ctx.EmitMovRegReg(r29, d35.Reg)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
			ctx.BindReg(r29, &d37)
		} else if d35.Loc == scm.LocImm && d35.Imm.Int() == 0 {
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d36.Reg}
			ctx.BindReg(d36.Reg, &d37)
		} else if d35.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d36.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d35.Imm.Int()))
			ctx.EmitAddInt64(scratch, d36.Reg)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d37)
		} else if d36.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d35.Reg)
			ctx.EmitMovRegReg(scratch, d35.Reg)
			if d36.Imm.Int() >= -2147483648 && d36.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d36.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d36.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d37)
		} else {
			r30 := ctx.AllocRegExcept(d35.Reg, d36.Reg)
			ctx.EmitMovRegReg(r30, d35.Reg)
			ctx.EmitAddInt64(r30, d36.Reg)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
			ctx.BindReg(r30, &d37)
		}
		if d37.Loc == scm.LocReg && d35.Loc == scm.LocReg && d37.Reg == d35.Reg {
			ctx.TransferReg(d35.Reg)
			d35.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d35)
		ctx.FreeDesc(&d36)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d37)
		var d38 scm.JITValueDesc
		if d37.Loc == scm.LocImm {
			d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d37.Imm.Int()) > uint64(0x40))}
		} else {
			r31 := ctx.AllocRegExcept(d37.Reg)
			ctx.EmitCmpRegImm32(d37.Reg, 64)
			ctx.EmitSetcc(r31, scm.CondUnsignedAbove)
			d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r31}
			ctx.BindReg(r31, &d38)
		}
		ctx.FreeDesc(&d37)
		ctx.ReclaimUntrackedRegs()
		d39 = d38
		ctx.EnsureDesc(&d39)
		if d39.Loc != scm.LocImm && d39.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		lbl16 := ctx.ReserveLabel()
		lbl17 := ctx.ReserveLabel()
		lbl18 := ctx.ReserveLabel()
		lbl19 := ctx.ReserveLabel()
		if d39.Loc == scm.LocImm {
			if d39.Imm.Bool() {
				ctx.MarkLabel(lbl18)
				ctx.EmitJmp(lbl16)
			} else {
				ctx.MarkLabel(lbl19)
				ctx.EmitJmp(lbl17)
			}
		} else {
			ctx.EmitCmpRegImm32(d39.Reg, 0)
			ctx.EmitJump(scm.CondNotEqual, lbl18)
			ctx.EmitJmp(lbl19)
			ctx.MarkLabel(lbl18)
			ctx.EmitJmp(lbl16)
			ctx.MarkLabel(lbl19)
			ctx.EmitJmp(lbl17)
		}
		ctx.FreeDesc(&d38)
		bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl17)
		ctx.ResolveFixups()
		d24 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d26)
		ctx.EnsureDesc(&d26)
		var d40 scm.JITValueDesc
		if d26.Loc == scm.LocImm {
			d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d26.Imm.Int()))))}
		} else {
			r32 := ctx.AllocReg()
			ctx.EmitMovRegReg(r32, d26.Reg)
			ctx.EmitShlRegImm8(r32, 56)
			ctx.EmitShrRegImm8(r32, 56)
			d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
			ctx.BindReg(r32, &d40)
		}
		ctx.ReclaimUntrackedRegs()
		d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d40)
		ctx.EnsureDesc(&d41)
		ctx.ProtectReg(d41.Reg)
		ctx.EnsureDesc(&d40)
		ctx.UnprotectReg(d41.Reg)
		var d42 scm.JITValueDesc
		if d41.Loc == scm.LocImm && d40.Loc == scm.LocImm {
			d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d41.Imm.Int() - d40.Imm.Int())}
		} else if d40.Loc == scm.LocImm && d40.Imm.Int() == 0 {
			r33 := ctx.AllocRegExcept(d41.Reg)
			ctx.EmitMovRegReg(r33, d41.Reg)
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
			ctx.BindReg(r33, &d42)
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
			r34 := ctx.AllocRegExcept(d41.Reg, d40.Reg)
			ctx.EmitMovRegReg(r34, d41.Reg)
			ctx.EmitSubInt64(r34, d40.Reg)
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
			ctx.BindReg(r34, &d42)
		}
		if d42.Loc == scm.LocReg && d41.Loc == scm.LocReg && d42.Reg == d41.Reg {
			ctx.TransferReg(d41.Reg)
			d41.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d40)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d24)
		ctx.EnsureDesc(&d42)
		var d43 scm.JITValueDesc
		if d24.Loc == scm.LocImm && d42.Loc == scm.LocImm {
			d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d24.Imm.Int()) >> uint64(d42.Imm.Int())))}
		} else if d42.Loc == scm.LocImm {
			r35 := ctx.AllocRegExcept(d24.Reg)
			ctx.EmitMovRegReg(r35, d24.Reg)
			ctx.EmitShrRegImm8(r35, uint8(d42.Imm.Int()))
			d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
			ctx.BindReg(r35, &d43)
		} else {
			{
				shiftSrc := d24.Reg
				r36 := ctx.AllocRegExcept(d24.Reg)
				ctx.EmitMovRegReg(r36, d24.Reg)
				shiftSrc = r36
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d42.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d42.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d42.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d43)
			}
		}
		if d43.Loc == scm.LocReg && d24.Loc == scm.LocReg && d43.Reg == d24.Reg {
			ctx.TransferReg(d24.Reg)
			d24.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d24)
		ctx.FreeDesc(&d42)
		ctx.ReclaimUntrackedRegs()
		r37 := ctx.AllocReg()
		ctx.EnsureDesc(&d43)
		ctx.EnsureDesc(&d43)
		if d43.Loc == scm.LocRegPair {
			panic("jit: scalar inline return has scm.LocRegPair")
		} else {
			ctx.EmitMovToReg(r37, d43)
		}
		ctx.EmitJmp(lbl15)
		bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl16)
		ctx.ResolveFixups()
		d24 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d28)
		var d44 scm.JITValueDesc
		if d28.Loc == scm.LocImm {
			d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d28.Imm.Int() / 64)}
		} else {
			r38 := ctx.AllocRegExcept(d28.Reg)
			ctx.EmitMovRegReg(r38, d28.Reg)
			ctx.EmitShrRegImm8(r38, 6)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
			ctx.BindReg(r38, &d44)
		}
		if d44.Loc == scm.LocReg && d28.Loc == scm.LocReg && d44.Reg == d28.Reg {
			ctx.TransferReg(d28.Reg)
			d28.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d44)
		ctx.EnsureDesc(&d44)
		var d45 scm.JITValueDesc
		if d44.Loc == scm.LocImm {
			d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d44.Reg)
			ctx.EmitMovRegReg(scratch, d44.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d45)
		}
		if d45.Loc == scm.LocReg && d44.Loc == scm.LocReg && d45.Reg == d44.Reg {
			ctx.TransferReg(d44.Reg)
			d44.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d44)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d45)
		ctx.ReclaimUntrackedRegs()
		d47 = ctx.EmitSliceElementAddress(&d29, &d45, 8)
		ctx.EnsureDesc(&d47)
		ctx.EmitMovRegMem(d47.Reg, d47.Reg, 0)
		d46 = d47
		ctx.FreeDesc(&d45)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d28)
		var d48 scm.JITValueDesc
		if d28.Loc == scm.LocImm {
			d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d28.Imm.Int() % 64)}
		} else {
			r39 := ctx.AllocRegExcept(d28.Reg)
			ctx.EmitMovRegReg(r39, d28.Reg)
			ctx.EmitAndRegImm32(r39, 63)
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
			ctx.BindReg(r39, &d48)
		}
		if d48.Loc == scm.LocReg && d28.Loc == scm.LocReg && d48.Reg == d28.Reg {
			ctx.TransferReg(d28.Reg)
			d28.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d28)
		ctx.ReclaimUntrackedRegs()
		d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d48)
		ctx.EnsureDesc(&d49)
		ctx.ProtectReg(d49.Reg)
		ctx.EnsureDesc(&d48)
		ctx.UnprotectReg(d49.Reg)
		var d50 scm.JITValueDesc
		if d49.Loc == scm.LocImm && d48.Loc == scm.LocImm {
			d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d49.Imm.Int() - d48.Imm.Int())}
		} else if d48.Loc == scm.LocImm && d48.Imm.Int() == 0 {
			r40 := ctx.AllocRegExcept(d49.Reg)
			ctx.EmitMovRegReg(r40, d49.Reg)
			d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
			ctx.BindReg(r40, &d50)
		} else if d49.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d48.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d49.Imm.Int()))
			ctx.EmitSubInt64(scratch, d48.Reg)
			d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d50)
		} else if d48.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d49.Reg)
			ctx.EmitMovRegReg(scratch, d49.Reg)
			if d48.Imm.Int() >= -2147483648 && d48.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d48.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d48.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d50)
		} else {
			r41 := ctx.AllocRegExcept(d49.Reg, d48.Reg)
			ctx.EmitMovRegReg(r41, d49.Reg)
			ctx.EmitSubInt64(r41, d48.Reg)
			d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
			ctx.BindReg(r41, &d50)
		}
		if d50.Loc == scm.LocReg && d49.Loc == scm.LocReg && d50.Reg == d49.Reg {
			ctx.TransferReg(d49.Reg)
			d49.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d48)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d46)
		ctx.EnsureDesc(&d50)
		var d51 scm.JITValueDesc
		if d46.Loc == scm.LocImm && d50.Loc == scm.LocImm {
			d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d46.Imm.Int()) >> uint64(d50.Imm.Int())))}
		} else if d50.Loc == scm.LocImm {
			r42 := ctx.AllocRegExcept(d46.Reg)
			ctx.EmitMovRegReg(r42, d46.Reg)
			ctx.EmitShrRegImm8(r42, uint8(d50.Imm.Int()))
			d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
			ctx.BindReg(r42, &d51)
		} else {
			{
				shiftSrc := d46.Reg
				r43 := ctx.AllocRegExcept(d46.Reg)
				ctx.EmitMovRegReg(r43, d46.Reg)
				shiftSrc = r43
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d50.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d50.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d50.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d51)
			}
		}
		if d51.Loc == scm.LocReg && d46.Loc == scm.LocReg && d51.Reg == d46.Reg {
			ctx.TransferReg(d46.Reg)
			d46.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d46)
		ctx.FreeDesc(&d50)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d34)
		ctx.EnsureDesc(&d51)
		var d52 scm.JITValueDesc
		if d34.Loc == scm.LocImm && d51.Loc == scm.LocImm {
			d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d34.Imm.Int() | d51.Imm.Int())}
		} else if d34.Loc == scm.LocImm && d34.Imm.Int() == 0 {
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d51.Reg}
			ctx.BindReg(d51.Reg, &d52)
		} else if d51.Loc == scm.LocImm && d51.Imm.Int() == 0 {
			r44 := ctx.AllocRegExcept(d34.Reg)
			ctx.EmitMovRegReg(r44, d34.Reg)
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
			ctx.BindReg(r44, &d52)
		} else if d34.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d51.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d34.Imm.Int()))
			ctx.EmitOrInt64(scratch, d51.Reg)
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d52)
		} else if d51.Loc == scm.LocImm {
			r45 := ctx.AllocRegExcept(d34.Reg)
			ctx.EmitMovRegReg(r45, d34.Reg)
			if d51.Imm.Int() >= -2147483648 && d51.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r45, int32(d51.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d51.Imm.Int()))
				ctx.EmitOrInt64(r45, scm.RegR11)
			}
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
			ctx.BindReg(r45, &d52)
		} else {
			r46 := ctx.AllocRegExcept(d34.Reg, d51.Reg)
			ctx.EmitMovRegReg(r46, d34.Reg)
			ctx.EmitOrInt64(r46, d51.Reg)
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
			ctx.BindReg(r46, &d52)
		}
		if d52.Loc == scm.LocReg && d34.Loc == scm.LocReg && d52.Reg == d34.Reg {
			ctx.TransferReg(d34.Reg)
			d34.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d52)
		ctx.EmitStoreToStack(d52, int32(phiBase23)+int32(0))
		ctx.StabilizeDescForControlFlow(&d52)
		ctx.FreeDesc(&d51)
		ctx.ReclaimUntrackedRegs()
		ctx.EmitJmp(lbl17)
		ctx.MarkLabel(lbl15)
		d53 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r37}
		ctx.BindReg(r37, &d53)
		ctx.BindReg(r37, &d53)
		if r6 {
			ctx.UnprotectReg(r7)
		}
		if r8 {
			ctx.UnprotectReg(r9)
		}
		if r10 {
			ctx.UnprotectReg(r11)
		}
		ctx.EnsureDesc(&d53)
		ctx.EnsureDesc(&d53)
		var d54 scm.JITValueDesc
		if d53.Loc == scm.LocImm {
			d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d53.Imm.Int()))))}
		} else {
			r47 := ctx.AllocReg()
			ctx.EmitMovRegReg(r47, d53.Reg)
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
			ctx.BindReg(r47, &d54)
		}
		ctx.FreeDesc(&d53)
		var d55 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
			r48 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r48, fieldAddr)
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r48}
			ctx.BindReg(r48, &d55)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
			r49 := ctx.AllocReg()
			ctx.EmitMovRegMem(r49, thisptr.Reg, off)
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r49}
			ctx.BindReg(r49, &d55)
		}
		ctx.EnsureDesc(&d54)
		ctx.EnsureDesc(&d55)
		ctx.EnsureDesc(&d54)
		ctx.ProtectReg(d54.Reg)
		ctx.EnsureDesc(&d55)
		ctx.UnprotectReg(d54.Reg)
		var d56 scm.JITValueDesc
		if d54.Loc == scm.LocImm && d55.Loc == scm.LocImm {
			d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d54.Imm.Int() + d55.Imm.Int())}
		} else if d55.Loc == scm.LocImm && d55.Imm.Int() == 0 {
			r50 := ctx.AllocRegExcept(d54.Reg)
			ctx.EmitMovRegReg(r50, d54.Reg)
			d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
			ctx.BindReg(r50, &d56)
		} else if d54.Loc == scm.LocImm && d54.Imm.Int() == 0 {
			d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d55.Reg}
			ctx.BindReg(d55.Reg, &d56)
		} else if d54.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d55.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d54.Imm.Int()))
			ctx.EmitAddInt64(scratch, d55.Reg)
			d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d56)
		} else if d55.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d54.Reg)
			ctx.EmitMovRegReg(scratch, d54.Reg)
			if d55.Imm.Int() >= -2147483648 && d55.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d55.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d55.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d56)
		} else {
			r51 := ctx.AllocRegExcept(d54.Reg, d55.Reg)
			ctx.EmitMovRegReg(r51, d54.Reg)
			ctx.EmitAddInt64(r51, d55.Reg)
			d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
			ctx.BindReg(r51, &d56)
		}
		if d56.Loc == scm.LocReg && d54.Loc == scm.LocReg && d56.Reg == d54.Reg {
			ctx.TransferReg(d54.Reg)
			d54.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d54)
		ctx.EnsureDesc(&d56)
		ctx.EnsureDesc(&d56)
		var d57 scm.JITValueDesc
		if d56.Loc == scm.LocImm {
			d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d56.Imm.Int()))))}
		} else {
			r52 := ctx.AllocReg()
			ctx.EmitMovRegReg(r52, d56.Reg)
			ctx.EmitShlRegImm8(r52, 32)
			ctx.EmitShrRegImm8(r52, 32)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
			ctx.BindReg(r52, &d57)
		}
		ctx.FreeDesc(&d56)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d57)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d57)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d57)
		var d58 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm && d57.Loc == scm.LocImm {
			d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d57.Imm.Int()))}
		} else if d57.Loc == scm.LocImm {
			r53 := ctx.AllocRegExcept(idxInt.Reg)
			if d57.Imm.Int() >= -2147483648 && d57.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(idxInt.Reg, int32(d57.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d57.Imm.Int()))
				ctx.EmitCmpInt64(idxInt.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r53, scm.CondUnsignedBelow)
			d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r53}
			ctx.BindReg(r53, &d58)
		} else if idxInt.Loc == scm.LocImm {
			r54 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d57.Reg)
			ctx.EmitSetcc(r54, scm.CondUnsignedBelow)
			d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r54}
			ctx.BindReg(r54, &d58)
		} else {
			r55 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitCmpInt64(idxInt.Reg, d57.Reg)
			ctx.EmitSetcc(r55, scm.CondUnsignedBelow)
			d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r55}
			ctx.BindReg(r55, &d58)
		}
		ctx.FreeDesc(&d57)
		d59 = d58
		ctx.EnsureDesc(&d59)
		if d59.Loc != scm.LocImm && d59.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d59.Loc == scm.LocImm {
			if d59.Imm.Bool() {
				if ps.General {
				}
				ps60 := scm.PhiState{General: ps.General}
				ps60.OverlayValues = make([]scm.JITValueDesc, 60)
				ps60.OverlayValues[1] = d1
				ps60.OverlayValues[2] = d2
				ps60.OverlayValues[3] = d3
				ps60.OverlayValues[4] = d4
				ps60.OverlayValues[5] = d5
				ps60.OverlayValues[6] = d6
				ps60.OverlayValues[7] = d7
				ps60.OverlayValues[8] = d8
				ps60.OverlayValues[9] = d9
				ps60.OverlayValues[10] = d10
				ps60.OverlayValues[11] = d11
				ps60.OverlayValues[12] = d12
				ps60.OverlayValues[13] = d13
				ps60.OverlayValues[14] = d14
				ps60.OverlayValues[15] = d15
				ps60.OverlayValues[17] = d17
				ps60.OverlayValues[18] = d18
				ps60.OverlayValues[19] = d19
				ps60.OverlayValues[20] = d20
				ps60.OverlayValues[21] = d21
				ps60.OverlayValues[22] = d22
				ps60.OverlayValues[24] = d24
				ps60.OverlayValues[25] = d25
				ps60.OverlayValues[26] = d26
				ps60.OverlayValues[27] = d27
				ps60.OverlayValues[28] = d28
				ps60.OverlayValues[29] = d29
				ps60.OverlayValues[30] = d30
				ps60.OverlayValues[31] = d31
				ps60.OverlayValues[32] = d32
				ps60.OverlayValues[33] = d33
				ps60.OverlayValues[34] = d34
				ps60.OverlayValues[35] = d35
				ps60.OverlayValues[36] = d36
				ps60.OverlayValues[37] = d37
				ps60.OverlayValues[38] = d38
				ps60.OverlayValues[39] = d39
				ps60.OverlayValues[40] = d40
				ps60.OverlayValues[41] = d41
				ps60.OverlayValues[42] = d42
				ps60.OverlayValues[43] = d43
				ps60.OverlayValues[44] = d44
				ps60.OverlayValues[45] = d45
				ps60.OverlayValues[46] = d46
				ps60.OverlayValues[47] = d47
				ps60.OverlayValues[48] = d48
				ps60.OverlayValues[49] = d49
				ps60.OverlayValues[50] = d50
				ps60.OverlayValues[51] = d51
				ps60.OverlayValues[52] = d52
				ps60.OverlayValues[53] = d53
				ps60.OverlayValues[54] = d54
				ps60.OverlayValues[55] = d55
				ps60.OverlayValues[56] = d56
				ps60.OverlayValues[57] = d57
				ps60.OverlayValues[58] = d58
				ps60.OverlayValues[59] = d59
				return bbs[3].RenderPS(ps60)
			}
			if ps.General {
			}
			ps61 := scm.PhiState{General: ps.General}
			ps61.OverlayValues = make([]scm.JITValueDesc, 60)
			ps61.OverlayValues[1] = d1
			ps61.OverlayValues[2] = d2
			ps61.OverlayValues[3] = d3
			ps61.OverlayValues[4] = d4
			ps61.OverlayValues[5] = d5
			ps61.OverlayValues[6] = d6
			ps61.OverlayValues[7] = d7
			ps61.OverlayValues[8] = d8
			ps61.OverlayValues[9] = d9
			ps61.OverlayValues[10] = d10
			ps61.OverlayValues[11] = d11
			ps61.OverlayValues[12] = d12
			ps61.OverlayValues[13] = d13
			ps61.OverlayValues[14] = d14
			ps61.OverlayValues[15] = d15
			ps61.OverlayValues[17] = d17
			ps61.OverlayValues[18] = d18
			ps61.OverlayValues[19] = d19
			ps61.OverlayValues[20] = d20
			ps61.OverlayValues[21] = d21
			ps61.OverlayValues[22] = d22
			ps61.OverlayValues[24] = d24
			ps61.OverlayValues[25] = d25
			ps61.OverlayValues[26] = d26
			ps61.OverlayValues[27] = d27
			ps61.OverlayValues[28] = d28
			ps61.OverlayValues[29] = d29
			ps61.OverlayValues[30] = d30
			ps61.OverlayValues[31] = d31
			ps61.OverlayValues[32] = d32
			ps61.OverlayValues[33] = d33
			ps61.OverlayValues[34] = d34
			ps61.OverlayValues[35] = d35
			ps61.OverlayValues[36] = d36
			ps61.OverlayValues[37] = d37
			ps61.OverlayValues[38] = d38
			ps61.OverlayValues[39] = d39
			ps61.OverlayValues[40] = d40
			ps61.OverlayValues[41] = d41
			ps61.OverlayValues[42] = d42
			ps61.OverlayValues[43] = d43
			ps61.OverlayValues[44] = d44
			ps61.OverlayValues[45] = d45
			ps61.OverlayValues[46] = d46
			ps61.OverlayValues[47] = d47
			ps61.OverlayValues[48] = d48
			ps61.OverlayValues[49] = d49
			ps61.OverlayValues[50] = d50
			ps61.OverlayValues[51] = d51
			ps61.OverlayValues[52] = d52
			ps61.OverlayValues[53] = d53
			ps61.OverlayValues[54] = d54
			ps61.OverlayValues[55] = d55
			ps61.OverlayValues[56] = d56
			ps61.OverlayValues[57] = d57
			ps61.OverlayValues[58] = d58
			ps61.OverlayValues[59] = d59
			return bbs[5].RenderPS(ps61)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d62 := ps.PhiValues[0]
				ctx.EnsureDesc(&d62)
				ctx.EmitStoreToStack(d62, int32(bbs[1].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d63 := ps.PhiValues[1]
				ctx.EnsureDesc(&d63)
				ctx.EmitStoreToStack(d63, int32(bbs[1].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d64 := ps.PhiValues[2]
				ctx.EnsureDesc(&d64)
				ctx.EmitStoreToStack(d64, int32(bbs[1].PhiBase)+int32(32))
			}
			ps.General = true
			return bbs[1].RenderPS(ps)
		}
		lbl20 := ctx.ReserveLabel()
		lbl21 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d59.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl20)
		ctx.EmitJmp(lbl21)
		ctx.MarkLabel(lbl20)
		ctx.EmitJmp(lbl4)
		ctx.MarkLabel(lbl21)
		ctx.EmitJmp(lbl6)
		ps65 := scm.PhiState{General: true}
		ps65.OverlayValues = make([]scm.JITValueDesc, 65)
		ps65.OverlayValues[1] = d1
		ps65.OverlayValues[2] = d2
		ps65.OverlayValues[3] = d3
		ps65.OverlayValues[4] = d4
		ps65.OverlayValues[5] = d5
		ps65.OverlayValues[6] = d6
		ps65.OverlayValues[7] = d7
		ps65.OverlayValues[8] = d8
		ps65.OverlayValues[9] = d9
		ps65.OverlayValues[10] = d10
		ps65.OverlayValues[11] = d11
		ps65.OverlayValues[12] = d12
		ps65.OverlayValues[13] = d13
		ps65.OverlayValues[14] = d14
		ps65.OverlayValues[15] = d15
		ps65.OverlayValues[17] = d17
		ps65.OverlayValues[18] = d18
		ps65.OverlayValues[19] = d19
		ps65.OverlayValues[20] = d20
		ps65.OverlayValues[21] = d21
		ps65.OverlayValues[22] = d22
		ps65.OverlayValues[24] = d24
		ps65.OverlayValues[25] = d25
		ps65.OverlayValues[26] = d26
		ps65.OverlayValues[27] = d27
		ps65.OverlayValues[28] = d28
		ps65.OverlayValues[29] = d29
		ps65.OverlayValues[30] = d30
		ps65.OverlayValues[31] = d31
		ps65.OverlayValues[32] = d32
		ps65.OverlayValues[33] = d33
		ps65.OverlayValues[34] = d34
		ps65.OverlayValues[35] = d35
		ps65.OverlayValues[36] = d36
		ps65.OverlayValues[37] = d37
		ps65.OverlayValues[38] = d38
		ps65.OverlayValues[39] = d39
		ps65.OverlayValues[40] = d40
		ps65.OverlayValues[41] = d41
		ps65.OverlayValues[42] = d42
		ps65.OverlayValues[43] = d43
		ps65.OverlayValues[44] = d44
		ps65.OverlayValues[45] = d45
		ps65.OverlayValues[46] = d46
		ps65.OverlayValues[47] = d47
		ps65.OverlayValues[48] = d48
		ps65.OverlayValues[49] = d49
		ps65.OverlayValues[50] = d50
		ps65.OverlayValues[51] = d51
		ps65.OverlayValues[52] = d52
		ps65.OverlayValues[53] = d53
		ps65.OverlayValues[54] = d54
		ps65.OverlayValues[55] = d55
		ps65.OverlayValues[56] = d56
		ps65.OverlayValues[57] = d57
		ps65.OverlayValues[58] = d58
		ps65.OverlayValues[59] = d59
		ps65.OverlayValues[62] = d62
		ps65.OverlayValues[63] = d63
		ps65.OverlayValues[64] = d64
		ps66 := scm.PhiState{General: true}
		ps66.OverlayValues = make([]scm.JITValueDesc, 65)
		ps66.OverlayValues[1] = d1
		ps66.OverlayValues[2] = d2
		ps66.OverlayValues[3] = d3
		ps66.OverlayValues[4] = d4
		ps66.OverlayValues[5] = d5
		ps66.OverlayValues[6] = d6
		ps66.OverlayValues[7] = d7
		ps66.OverlayValues[8] = d8
		ps66.OverlayValues[9] = d9
		ps66.OverlayValues[10] = d10
		ps66.OverlayValues[11] = d11
		ps66.OverlayValues[12] = d12
		ps66.OverlayValues[13] = d13
		ps66.OverlayValues[14] = d14
		ps66.OverlayValues[15] = d15
		ps66.OverlayValues[17] = d17
		ps66.OverlayValues[18] = d18
		ps66.OverlayValues[19] = d19
		ps66.OverlayValues[20] = d20
		ps66.OverlayValues[21] = d21
		ps66.OverlayValues[22] = d22
		ps66.OverlayValues[24] = d24
		ps66.OverlayValues[25] = d25
		ps66.OverlayValues[26] = d26
		ps66.OverlayValues[27] = d27
		ps66.OverlayValues[28] = d28
		ps66.OverlayValues[29] = d29
		ps66.OverlayValues[30] = d30
		ps66.OverlayValues[31] = d31
		ps66.OverlayValues[32] = d32
		ps66.OverlayValues[33] = d33
		ps66.OverlayValues[34] = d34
		ps66.OverlayValues[35] = d35
		ps66.OverlayValues[36] = d36
		ps66.OverlayValues[37] = d37
		ps66.OverlayValues[38] = d38
		ps66.OverlayValues[39] = d39
		ps66.OverlayValues[40] = d40
		ps66.OverlayValues[41] = d41
		ps66.OverlayValues[42] = d42
		ps66.OverlayValues[43] = d43
		ps66.OverlayValues[44] = d44
		ps66.OverlayValues[45] = d45
		ps66.OverlayValues[46] = d46
		ps66.OverlayValues[47] = d47
		ps66.OverlayValues[48] = d48
		ps66.OverlayValues[49] = d49
		ps66.OverlayValues[50] = d50
		ps66.OverlayValues[51] = d51
		ps66.OverlayValues[52] = d52
		ps66.OverlayValues[53] = d53
		ps66.OverlayValues[54] = d54
		ps66.OverlayValues[55] = d55
		ps66.OverlayValues[56] = d56
		ps66.OverlayValues[57] = d57
		ps66.OverlayValues[58] = d58
		ps66.OverlayValues[59] = d59
		ps66.OverlayValues[62] = d62
		ps66.OverlayValues[63] = d63
		ps66.OverlayValues[64] = d64
		snap67 := d1
		snap68 := d2
		snap69 := d3
		snap70 := d4
		snap71 := d5
		snap72 := d6
		snap73 := d7
		snap74 := d8
		snap75 := d9
		snap76 := d10
		snap77 := d11
		snap78 := d12
		snap79 := d13
		snap80 := d14
		snap81 := d15
		snap82 := d17
		snap83 := d18
		snap84 := d19
		snap85 := d20
		snap86 := d21
		snap87 := d22
		snap88 := d24
		snap89 := d25
		snap90 := d26
		snap91 := d27
		snap92 := d28
		snap93 := d29
		snap94 := d30
		snap95 := d31
		snap96 := d32
		snap97 := d33
		snap98 := d34
		snap99 := d35
		snap100 := d36
		snap101 := d37
		snap102 := d38
		snap103 := d39
		snap104 := d40
		snap105 := d41
		snap106 := d42
		snap107 := d43
		snap108 := d44
		snap109 := d45
		snap110 := d46
		snap111 := d47
		snap112 := d48
		snap113 := d49
		snap114 := d50
		snap115 := d51
		snap116 := d52
		snap117 := d53
		snap118 := d54
		snap119 := d55
		snap120 := d56
		snap121 := d57
		snap122 := d58
		snap123 := d59
		snap124 := d62
		snap125 := d63
		snap126 := d64
		alloc127 := ctx.SnapshotAllocState()
		if !bbs[5].Rendered {
			bbs[5].RenderPS(ps66)
		}
		ctx.RestoreAllocState(alloc127)
		d1 = snap67
		d2 = snap68
		d3 = snap69
		d4 = snap70
		d5 = snap71
		d6 = snap72
		d7 = snap73
		d8 = snap74
		d9 = snap75
		d10 = snap76
		d11 = snap77
		d12 = snap78
		d13 = snap79
		d14 = snap80
		d15 = snap81
		d17 = snap82
		d18 = snap83
		d19 = snap84
		d20 = snap85
		d21 = snap86
		d22 = snap87
		d24 = snap88
		d25 = snap89
		d26 = snap90
		d27 = snap91
		d28 = snap92
		d29 = snap93
		d30 = snap94
		d31 = snap95
		d32 = snap96
		d33 = snap97
		d34 = snap98
		d35 = snap99
		d36 = snap100
		d37 = snap101
		d38 = snap102
		d39 = snap103
		d40 = snap104
		d41 = snap105
		d42 = snap106
		d43 = snap107
		d44 = snap108
		d45 = snap109
		d46 = snap110
		d47 = snap111
		d48 = snap112
		d49 = snap113
		d50 = snap114
		d51 = snap115
		d52 = snap116
		d53 = snap117
		d54 = snap118
		d55 = snap119
		d56 = snap120
		d57 = snap121
		d58 = snap122
		d59 = snap123
		d62 = snap124
		d63 = snap125
		d64 = snap126
		if !bbs[3].Rendered {
			return bbs[3].RenderPS(ps65)
		}
		return result
		ctx.FreeDesc(&d58)
		return result
	}
	bbs[2].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d128 := ps.PhiValues[0]
				ctx.EnsureDesc(&d128)
				ctx.EmitStoreToStack(d128, int32(bbs[2].PhiBase)+int32(0))
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
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
		}
		if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
			d58 = ps.OverlayValues[58]
		}
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
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
		if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
			d128 = ps.OverlayValues[128]
		}
		if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
			d4 = ps.PhiValues[0]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d4)
		ctx.EnsureDesc(&d4)
		ctx.EnsureDesc(&d4)
		var d129 scm.JITValueDesc
		if d4.Loc == scm.LocImm {
			d129 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(d4.Imm.Int()))))}
		} else {
			r56 := ctx.AllocReg()
			ctx.EmitMovRegReg(r56, d4.Reg)
			ctx.EmitShlRegImm8(r56, 32)
			ctx.EmitShrRegImm8(r56, 32)
			d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r56}
			ctx.BindReg(r56, &d129)
		}
		ctx.EnsureDesc(&d129)
		if thisptr.Loc == scm.LocImm {
			baseReg := ctx.AllocReg()
			if d129.Loc == scm.LocReg {
				ctx.FreeReg(baseReg)
				baseReg = ctx.AllocRegExcept(d129.Reg)
			}
			ctx.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int())+unsafe.Offsetof((*StorageSeq)(nil).lastValue)))
			if d129.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d129.Imm.Int()))
				ctx.EmitStoreRegMem(scm.RegR11, baseReg, 0)
			} else {
				ctx.EmitStoreRegMem(d129.Reg, baseReg, 0)
			}
			ctx.FreeReg(baseReg)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
			if d129.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d129.Imm.Int()))
				ctx.EmitStoreRegMem(scm.RegR11, thisptr.Reg, off)
			} else {
				ctx.EmitStoreRegMem(d129.Reg, thisptr.Reg, off)
			}
		}
		ctx.FreeDesc(&d129)
		ctx.EnsureDesc(&d4)
		d130 = d4
		_ = d130
		ctx.StabilizeDescForControlFlow(&d130)
		r57 := d4.Loc == scm.LocReg || d4.Loc == scm.LocRegPair || d4.Loc == scm.LocRegTriple
		r58 := d4.Reg
		if r57 {
			ctx.ProtectReg(r58)
		}
		r59 := d4.Loc == scm.LocRegPair || d4.Loc == scm.LocRegTriple
		r60 := d4.Reg2
		if r59 {
			ctx.ProtectReg(r60)
		}
		r61 := d4.Loc == scm.LocRegTriple
		r62 := d4.Reg3
		if r61 {
			ctx.ProtectReg(r62)
		}
		phiBase131 = ctx.AllocStack(int32(16))
		d132 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase131) + int32(0)}
		_ = d132
		lbl22 := ctx.ReserveLabel()
		bbpos_2_0 := int32(-1)
		_ = bbpos_2_0
		bbpos_2_1 := int32(-1)
		_ = bbpos_2_1
		bbpos_2_2 := int32(-1)
		_ = bbpos_2_2
		bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		d132 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d130)
		ctx.EnsureDesc(&d130)
		var d133 scm.JITValueDesc
		if d130.Loc == scm.LocImm {
			d133 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d130.Imm.Int()))))}
		} else {
			r63 := ctx.AllocReg()
			ctx.EmitMovRegReg(r63, d130.Reg)
			ctx.EmitShlRegImm8(r63, 32)
			ctx.EmitShrRegImm8(r63, 32)
			d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r63}
			ctx.BindReg(r63, &d133)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d134 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
			r64 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r64, fieldAddr)
			d134 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r64}
			ctx.BindReg(r64, &d134)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
			r65 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r65, thisptr.Reg, off)
			d134 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r65}
			ctx.BindReg(r65, &d134)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d134)
		ctx.EnsureDesc(&d134)
		var d135 scm.JITValueDesc
		if d134.Loc == scm.LocImm {
			d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d134.Imm.Int()))))}
		} else {
			r66 := ctx.AllocReg()
			ctx.EmitMovRegReg(r66, d134.Reg)
			ctx.EmitShlRegImm8(r66, 56)
			ctx.EmitShrRegImm8(r66, 56)
			d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
			ctx.BindReg(r66, &d135)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d133)
		ctx.EnsureDesc(&d135)
		ctx.EnsureDesc(&d133)
		ctx.ProtectReg(d133.Reg)
		ctx.EnsureDesc(&d135)
		ctx.UnprotectReg(d133.Reg)
		var d136 scm.JITValueDesc
		if d133.Loc == scm.LocImm && d135.Loc == scm.LocImm {
			d136 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d133.Imm.Int() * d135.Imm.Int())}
		} else if d133.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d135.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d133.Imm.Int()))
			ctx.EmitImulInt64(scratch, d135.Reg)
			d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d136)
		} else if d135.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d133.Reg)
			ctx.EmitMovRegReg(scratch, d133.Reg)
			if d135.Imm.Int() >= -2147483648 && d135.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d135.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d135.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d136)
		} else {
			r67 := ctx.AllocRegExcept(d133.Reg, d135.Reg)
			ctx.EmitMovRegReg(r67, d133.Reg)
			ctx.EmitImulInt64(r67, d135.Reg)
			d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r67}
			ctx.BindReg(r67, &d136)
		}
		if d136.Loc == scm.LocReg && d133.Loc == scm.LocReg && d136.Reg == d133.Reg {
			ctx.TransferReg(d133.Reg)
			d133.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d136)
		ctx.FreeDesc(&d133)
		ctx.FreeDesc(&d135)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d137 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 0
			r68 := ctx.AllocReg()
			r69 := ctx.AllocRegExcept(r68)
			r70 := ctx.AllocRegExcept(r68, r69)
			ctx.EmitMovRegMem64(r68, fieldAddr)
			ctx.EmitMovRegMem64(r69, fieldAddr+8)
			ctx.EmitMovRegMem64(r70, fieldAddr+16)
			d137 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r68, Reg2: r69, Reg3: r70}
			ctx.BindReg(r68, &d137)
			ctx.BindReg(r69, &d137)
			ctx.BindReg(r70, &d137)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 0)
			r71 := ctx.AllocReg()
			r72 := ctx.AllocRegExcept(r71)
			r73 := ctx.AllocRegExcept(r71, r72)
			ctx.EmitMovRegMem(r71, thisptr.Reg, off)
			ctx.EmitMovRegMem(r72, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r73, thisptr.Reg, off+16)
			d137 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r71, Reg2: r72, Reg3: r73}
			ctx.BindReg(r71, &d137)
			ctx.BindReg(r72, &d137)
			ctx.BindReg(r73, &d137)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d136)
		var d138 scm.JITValueDesc
		if d136.Loc == scm.LocImm {
			d138 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d136.Imm.Int() / 64)}
		} else {
			r74 := ctx.AllocRegExcept(d136.Reg)
			ctx.EmitMovRegReg(r74, d136.Reg)
			ctx.EmitShrRegImm8(r74, 6)
			d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r74}
			ctx.BindReg(r74, &d138)
		}
		if d138.Loc == scm.LocReg && d136.Loc == scm.LocReg && d138.Reg == d136.Reg {
			ctx.TransferReg(d136.Reg)
			d136.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d138)
		ctx.ReclaimUntrackedRegs()
		d140 = ctx.EmitSliceElementAddress(&d137, &d138, 8)
		ctx.EnsureDesc(&d140)
		ctx.EmitMovRegMem(d140.Reg, d140.Reg, 0)
		d139 = d140
		ctx.FreeDesc(&d138)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d136)
		var d141 scm.JITValueDesc
		if d136.Loc == scm.LocImm {
			d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d136.Imm.Int() % 64)}
		} else {
			r75 := ctx.AllocRegExcept(d136.Reg)
			ctx.EmitMovRegReg(r75, d136.Reg)
			ctx.EmitAndRegImm32(r75, 63)
			d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
			ctx.BindReg(r75, &d141)
		}
		if d141.Loc == scm.LocReg && d136.Loc == scm.LocReg && d141.Reg == d136.Reg {
			ctx.TransferReg(d136.Reg)
			d136.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d139)
		ctx.EnsureDesc(&d141)
		var d142 scm.JITValueDesc
		if d139.Loc == scm.LocImm && d141.Loc == scm.LocImm {
			d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d139.Imm.Int()) << uint64(d141.Imm.Int())))}
		} else if d141.Loc == scm.LocImm {
			r76 := ctx.AllocRegExcept(d139.Reg)
			ctx.EmitMovRegReg(r76, d139.Reg)
			ctx.EmitShlRegImm8(r76, uint8(d141.Imm.Int()))
			d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r76}
			ctx.BindReg(r76, &d142)
		} else {
			{
				shiftSrc := d139.Reg
				r77 := ctx.AllocRegExcept(d139.Reg)
				ctx.EmitMovRegReg(r77, d139.Reg)
				shiftSrc = r77
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d141.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d141.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d141.Reg)
				}
				ctx.EmitShlRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d142)
			}
		}
		if d142.Loc == scm.LocReg && d139.Loc == scm.LocReg && d142.Reg == d139.Reg {
			ctx.TransferReg(d139.Reg)
			d139.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d142)
		ctx.EmitStoreToStack(d142, int32(phiBase131)+int32(0))
		ctx.StabilizeDescForControlFlow(&d142)
		ctx.FreeDesc(&d139)
		ctx.FreeDesc(&d141)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d136)
		var d143 scm.JITValueDesc
		if d136.Loc == scm.LocImm {
			d143 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d136.Imm.Int() % 64)}
		} else {
			r78 := ctx.AllocRegExcept(d136.Reg)
			ctx.EmitMovRegReg(r78, d136.Reg)
			ctx.EmitAndRegImm32(r78, 63)
			d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r78}
			ctx.BindReg(r78, &d143)
		}
		if d143.Loc == scm.LocReg && d136.Loc == scm.LocReg && d143.Reg == d136.Reg {
			ctx.TransferReg(d136.Reg)
			d136.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d134)
		ctx.EnsureDesc(&d134)
		var d144 scm.JITValueDesc
		if d134.Loc == scm.LocImm {
			d144 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d134.Imm.Int()))))}
		} else {
			r79 := ctx.AllocReg()
			ctx.EmitMovRegReg(r79, d134.Reg)
			ctx.EmitShlRegImm8(r79, 56)
			ctx.EmitShrRegImm8(r79, 56)
			d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r79}
			ctx.BindReg(r79, &d144)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d143)
		ctx.EnsureDesc(&d144)
		ctx.EnsureDesc(&d143)
		ctx.ProtectReg(d143.Reg)
		ctx.EnsureDesc(&d144)
		ctx.UnprotectReg(d143.Reg)
		var d145 scm.JITValueDesc
		if d143.Loc == scm.LocImm && d144.Loc == scm.LocImm {
			d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d143.Imm.Int() + d144.Imm.Int())}
		} else if d144.Loc == scm.LocImm && d144.Imm.Int() == 0 {
			r80 := ctx.AllocRegExcept(d143.Reg)
			ctx.EmitMovRegReg(r80, d143.Reg)
			d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
			ctx.BindReg(r80, &d145)
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
			r81 := ctx.AllocRegExcept(d143.Reg, d144.Reg)
			ctx.EmitMovRegReg(r81, d143.Reg)
			ctx.EmitAddInt64(r81, d144.Reg)
			d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
			ctx.BindReg(r81, &d145)
		}
		if d145.Loc == scm.LocReg && d143.Loc == scm.LocReg && d145.Reg == d143.Reg {
			ctx.TransferReg(d143.Reg)
			d143.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d143)
		ctx.FreeDesc(&d144)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d145)
		var d146 scm.JITValueDesc
		if d145.Loc == scm.LocImm {
			d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d145.Imm.Int()) > uint64(0x40))}
		} else {
			r82 := ctx.AllocRegExcept(d145.Reg)
			ctx.EmitCmpRegImm32(d145.Reg, 64)
			ctx.EmitSetcc(r82, scm.CondUnsignedAbove)
			d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r82}
			ctx.BindReg(r82, &d146)
		}
		ctx.FreeDesc(&d145)
		ctx.ReclaimUntrackedRegs()
		d147 = d146
		ctx.EnsureDesc(&d147)
		if d147.Loc != scm.LocImm && d147.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		lbl23 := ctx.ReserveLabel()
		lbl24 := ctx.ReserveLabel()
		lbl25 := ctx.ReserveLabel()
		lbl26 := ctx.ReserveLabel()
		if d147.Loc == scm.LocImm {
			if d147.Imm.Bool() {
				ctx.MarkLabel(lbl25)
				ctx.EmitJmp(lbl23)
			} else {
				ctx.MarkLabel(lbl26)
				ctx.EmitJmp(lbl24)
			}
		} else {
			ctx.EmitCmpRegImm32(d147.Reg, 0)
			ctx.EmitJump(scm.CondNotEqual, lbl25)
			ctx.EmitJmp(lbl26)
			ctx.MarkLabel(lbl25)
			ctx.EmitJmp(lbl23)
			ctx.MarkLabel(lbl26)
			ctx.EmitJmp(lbl24)
		}
		ctx.FreeDesc(&d146)
		bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl24)
		ctx.ResolveFixups()
		d132 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d134)
		ctx.EnsureDesc(&d134)
		var d148 scm.JITValueDesc
		if d134.Loc == scm.LocImm {
			d148 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d134.Imm.Int()))))}
		} else {
			r83 := ctx.AllocReg()
			ctx.EmitMovRegReg(r83, d134.Reg)
			ctx.EmitShlRegImm8(r83, 56)
			ctx.EmitShrRegImm8(r83, 56)
			d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
			ctx.BindReg(r83, &d148)
		}
		ctx.ReclaimUntrackedRegs()
		d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d148)
		ctx.EnsureDesc(&d149)
		ctx.ProtectReg(d149.Reg)
		ctx.EnsureDesc(&d148)
		ctx.UnprotectReg(d149.Reg)
		var d150 scm.JITValueDesc
		if d149.Loc == scm.LocImm && d148.Loc == scm.LocImm {
			d150 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d149.Imm.Int() - d148.Imm.Int())}
		} else if d148.Loc == scm.LocImm && d148.Imm.Int() == 0 {
			r84 := ctx.AllocRegExcept(d149.Reg)
			ctx.EmitMovRegReg(r84, d149.Reg)
			d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r84}
			ctx.BindReg(r84, &d150)
		} else if d149.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d148.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d149.Imm.Int()))
			ctx.EmitSubInt64(scratch, d148.Reg)
			d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d150)
		} else if d148.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d149.Reg)
			ctx.EmitMovRegReg(scratch, d149.Reg)
			if d148.Imm.Int() >= -2147483648 && d148.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d148.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d148.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d150)
		} else {
			r85 := ctx.AllocRegExcept(d149.Reg, d148.Reg)
			ctx.EmitMovRegReg(r85, d149.Reg)
			ctx.EmitSubInt64(r85, d148.Reg)
			d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
			ctx.BindReg(r85, &d150)
		}
		if d150.Loc == scm.LocReg && d149.Loc == scm.LocReg && d150.Reg == d149.Reg {
			ctx.TransferReg(d149.Reg)
			d149.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d148)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d132)
		ctx.EnsureDesc(&d150)
		var d151 scm.JITValueDesc
		if d132.Loc == scm.LocImm && d150.Loc == scm.LocImm {
			d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d132.Imm.Int()) >> uint64(d150.Imm.Int())))}
		} else if d150.Loc == scm.LocImm {
			r86 := ctx.AllocRegExcept(d132.Reg)
			ctx.EmitMovRegReg(r86, d132.Reg)
			ctx.EmitShrRegImm8(r86, uint8(d150.Imm.Int()))
			d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
			ctx.BindReg(r86, &d151)
		} else {
			{
				shiftSrc := d132.Reg
				r87 := ctx.AllocRegExcept(d132.Reg)
				ctx.EmitMovRegReg(r87, d132.Reg)
				shiftSrc = r87
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d150.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d150.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d150.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d151)
			}
		}
		if d151.Loc == scm.LocReg && d132.Loc == scm.LocReg && d151.Reg == d132.Reg {
			ctx.TransferReg(d132.Reg)
			d132.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d132)
		ctx.FreeDesc(&d150)
		ctx.ReclaimUntrackedRegs()
		r88 := ctx.AllocReg()
		ctx.EnsureDesc(&d151)
		ctx.EnsureDesc(&d151)
		if d151.Loc == scm.LocRegPair {
			panic("jit: scalar inline return has scm.LocRegPair")
		} else {
			ctx.EmitMovToReg(r88, d151)
		}
		ctx.EmitJmp(lbl22)
		bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl23)
		ctx.ResolveFixups()
		d132 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d136)
		var d152 scm.JITValueDesc
		if d136.Loc == scm.LocImm {
			d152 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d136.Imm.Int() / 64)}
		} else {
			r89 := ctx.AllocRegExcept(d136.Reg)
			ctx.EmitMovRegReg(r89, d136.Reg)
			ctx.EmitShrRegImm8(r89, 6)
			d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
			ctx.BindReg(r89, &d152)
		}
		if d152.Loc == scm.LocReg && d136.Loc == scm.LocReg && d152.Reg == d136.Reg {
			ctx.TransferReg(d136.Reg)
			d136.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d152)
		ctx.EnsureDesc(&d152)
		var d153 scm.JITValueDesc
		if d152.Loc == scm.LocImm {
			d153 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d152.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d152.Reg)
			ctx.EmitMovRegReg(scratch, d152.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d153)
		}
		if d153.Loc == scm.LocReg && d152.Loc == scm.LocReg && d153.Reg == d152.Reg {
			ctx.TransferReg(d152.Reg)
			d152.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d152)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d153)
		ctx.ReclaimUntrackedRegs()
		d155 = ctx.EmitSliceElementAddress(&d137, &d153, 8)
		ctx.EnsureDesc(&d155)
		ctx.EmitMovRegMem(d155.Reg, d155.Reg, 0)
		d154 = d155
		ctx.FreeDesc(&d153)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d136)
		var d156 scm.JITValueDesc
		if d136.Loc == scm.LocImm {
			d156 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d136.Imm.Int() % 64)}
		} else {
			r90 := ctx.AllocRegExcept(d136.Reg)
			ctx.EmitMovRegReg(r90, d136.Reg)
			ctx.EmitAndRegImm32(r90, 63)
			d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
			ctx.BindReg(r90, &d156)
		}
		if d156.Loc == scm.LocReg && d136.Loc == scm.LocReg && d156.Reg == d136.Reg {
			ctx.TransferReg(d136.Reg)
			d136.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d136)
		ctx.ReclaimUntrackedRegs()
		d157 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d156)
		ctx.EnsureDesc(&d157)
		ctx.ProtectReg(d157.Reg)
		ctx.EnsureDesc(&d156)
		ctx.UnprotectReg(d157.Reg)
		var d158 scm.JITValueDesc
		if d157.Loc == scm.LocImm && d156.Loc == scm.LocImm {
			d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d157.Imm.Int() - d156.Imm.Int())}
		} else if d156.Loc == scm.LocImm && d156.Imm.Int() == 0 {
			r91 := ctx.AllocRegExcept(d157.Reg)
			ctx.EmitMovRegReg(r91, d157.Reg)
			d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r91}
			ctx.BindReg(r91, &d158)
		} else if d157.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d156.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d157.Imm.Int()))
			ctx.EmitSubInt64(scratch, d156.Reg)
			d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d158)
		} else if d156.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d157.Reg)
			ctx.EmitMovRegReg(scratch, d157.Reg)
			if d156.Imm.Int() >= -2147483648 && d156.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d156.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d156.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d158)
		} else {
			r92 := ctx.AllocRegExcept(d157.Reg, d156.Reg)
			ctx.EmitMovRegReg(r92, d157.Reg)
			ctx.EmitSubInt64(r92, d156.Reg)
			d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r92}
			ctx.BindReg(r92, &d158)
		}
		if d158.Loc == scm.LocReg && d157.Loc == scm.LocReg && d158.Reg == d157.Reg {
			ctx.TransferReg(d157.Reg)
			d157.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d156)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d154)
		ctx.EnsureDesc(&d158)
		var d159 scm.JITValueDesc
		if d154.Loc == scm.LocImm && d158.Loc == scm.LocImm {
			d159 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d154.Imm.Int()) >> uint64(d158.Imm.Int())))}
		} else if d158.Loc == scm.LocImm {
			r93 := ctx.AllocRegExcept(d154.Reg)
			ctx.EmitMovRegReg(r93, d154.Reg)
			ctx.EmitShrRegImm8(r93, uint8(d158.Imm.Int()))
			d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r93}
			ctx.BindReg(r93, &d159)
		} else {
			{
				shiftSrc := d154.Reg
				r94 := ctx.AllocRegExcept(d154.Reg)
				ctx.EmitMovRegReg(r94, d154.Reg)
				shiftSrc = r94
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d158.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d158.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d158.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d159)
			}
		}
		if d159.Loc == scm.LocReg && d154.Loc == scm.LocReg && d159.Reg == d154.Reg {
			ctx.TransferReg(d154.Reg)
			d154.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d154)
		ctx.FreeDesc(&d158)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d142)
		ctx.EnsureDesc(&d159)
		var d160 scm.JITValueDesc
		if d142.Loc == scm.LocImm && d159.Loc == scm.LocImm {
			d160 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d142.Imm.Int() | d159.Imm.Int())}
		} else if d142.Loc == scm.LocImm && d142.Imm.Int() == 0 {
			d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d159.Reg}
			ctx.BindReg(d159.Reg, &d160)
		} else if d159.Loc == scm.LocImm && d159.Imm.Int() == 0 {
			r95 := ctx.AllocRegExcept(d142.Reg)
			ctx.EmitMovRegReg(r95, d142.Reg)
			d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r95}
			ctx.BindReg(r95, &d160)
		} else if d142.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d159.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d142.Imm.Int()))
			ctx.EmitOrInt64(scratch, d159.Reg)
			d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d160)
		} else if d159.Loc == scm.LocImm {
			r96 := ctx.AllocRegExcept(d142.Reg)
			ctx.EmitMovRegReg(r96, d142.Reg)
			if d159.Imm.Int() >= -2147483648 && d159.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r96, int32(d159.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d159.Imm.Int()))
				ctx.EmitOrInt64(r96, scm.RegR11)
			}
			d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
			ctx.BindReg(r96, &d160)
		} else {
			r97 := ctx.AllocRegExcept(d142.Reg, d159.Reg)
			ctx.EmitMovRegReg(r97, d142.Reg)
			ctx.EmitOrInt64(r97, d159.Reg)
			d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
			ctx.BindReg(r97, &d160)
		}
		if d160.Loc == scm.LocReg && d142.Loc == scm.LocReg && d160.Reg == d142.Reg {
			ctx.TransferReg(d142.Reg)
			d142.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d160)
		ctx.EmitStoreToStack(d160, int32(phiBase131)+int32(0))
		ctx.StabilizeDescForControlFlow(&d160)
		ctx.FreeDesc(&d159)
		ctx.ReclaimUntrackedRegs()
		ctx.EmitJmp(lbl24)
		ctx.MarkLabel(lbl22)
		d161 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r88}
		ctx.BindReg(r88, &d161)
		ctx.BindReg(r88, &d161)
		if r57 {
			ctx.UnprotectReg(r58)
		}
		if r59 {
			ctx.UnprotectReg(r60)
		}
		if r61 {
			ctx.UnprotectReg(r62)
		}
		ctx.EnsureDesc(&d161)
		ctx.EnsureDesc(&d161)
		var d162 scm.JITValueDesc
		if d161.Loc == scm.LocImm {
			d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d161.Imm.Int()))))}
		} else {
			r98 := ctx.AllocReg()
			ctx.EmitMovRegReg(r98, d161.Reg)
			d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r98}
			ctx.BindReg(r98, &d162)
		}
		ctx.FreeDesc(&d161)
		var d163 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 32
			r99 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r99, fieldAddr)
			d163 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r99}
			ctx.BindReg(r99, &d163)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 32)
			r100 := ctx.AllocReg()
			ctx.EmitMovRegMem(r100, thisptr.Reg, off)
			d163 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r100}
			ctx.BindReg(r100, &d163)
		}
		ctx.EnsureDesc(&d162)
		ctx.EnsureDesc(&d163)
		ctx.EnsureDesc(&d162)
		ctx.ProtectReg(d162.Reg)
		ctx.EnsureDesc(&d163)
		ctx.UnprotectReg(d162.Reg)
		var d164 scm.JITValueDesc
		if d162.Loc == scm.LocImm && d163.Loc == scm.LocImm {
			d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d162.Imm.Int() + d163.Imm.Int())}
		} else if d163.Loc == scm.LocImm && d163.Imm.Int() == 0 {
			r101 := ctx.AllocRegExcept(d162.Reg)
			ctx.EmitMovRegReg(r101, d162.Reg)
			d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r101}
			ctx.BindReg(r101, &d164)
		} else if d162.Loc == scm.LocImm && d162.Imm.Int() == 0 {
			d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d163.Reg}
			ctx.BindReg(d163.Reg, &d164)
		} else if d162.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d163.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d162.Imm.Int()))
			ctx.EmitAddInt64(scratch, d163.Reg)
			d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d164)
		} else if d163.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d162.Reg)
			ctx.EmitMovRegReg(scratch, d162.Reg)
			if d163.Imm.Int() >= -2147483648 && d163.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d163.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d163.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d164)
		} else {
			r102 := ctx.AllocRegExcept(d162.Reg, d163.Reg)
			ctx.EmitMovRegReg(r102, d162.Reg)
			ctx.EmitAddInt64(r102, d163.Reg)
			d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r102}
			ctx.BindReg(r102, &d164)
		}
		if d164.Loc == scm.LocReg && d162.Loc == scm.LocReg && d164.Reg == d162.Reg {
			ctx.TransferReg(d162.Reg)
			d162.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d164)
		ctx.FreeDesc(&d162)
		var d165 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 56
			r103 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r103, fieldAddr)
			d165 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r103}
			ctx.BindReg(r103, &d165)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 56)
			r104 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r104, thisptr.Reg, off)
			d165 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r104}
			ctx.BindReg(r104, &d165)
		}
		d166 = d165
		ctx.EnsureDesc(&d166)
		if d166.Loc != scm.LocImm && d166.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d166.Loc == scm.LocImm {
			if d166.Imm.Bool() {
				if ps.General {
				}
				ps167 := scm.PhiState{General: ps.General}
				ps167.OverlayValues = make([]scm.JITValueDesc, 167)
				ps167.OverlayValues[1] = d1
				ps167.OverlayValues[2] = d2
				ps167.OverlayValues[3] = d3
				ps167.OverlayValues[4] = d4
				ps167.OverlayValues[5] = d5
				ps167.OverlayValues[6] = d6
				ps167.OverlayValues[7] = d7
				ps167.OverlayValues[8] = d8
				ps167.OverlayValues[9] = d9
				ps167.OverlayValues[10] = d10
				ps167.OverlayValues[11] = d11
				ps167.OverlayValues[12] = d12
				ps167.OverlayValues[13] = d13
				ps167.OverlayValues[14] = d14
				ps167.OverlayValues[15] = d15
				ps167.OverlayValues[17] = d17
				ps167.OverlayValues[18] = d18
				ps167.OverlayValues[19] = d19
				ps167.OverlayValues[20] = d20
				ps167.OverlayValues[21] = d21
				ps167.OverlayValues[22] = d22
				ps167.OverlayValues[24] = d24
				ps167.OverlayValues[25] = d25
				ps167.OverlayValues[26] = d26
				ps167.OverlayValues[27] = d27
				ps167.OverlayValues[28] = d28
				ps167.OverlayValues[29] = d29
				ps167.OverlayValues[30] = d30
				ps167.OverlayValues[31] = d31
				ps167.OverlayValues[32] = d32
				ps167.OverlayValues[33] = d33
				ps167.OverlayValues[34] = d34
				ps167.OverlayValues[35] = d35
				ps167.OverlayValues[36] = d36
				ps167.OverlayValues[37] = d37
				ps167.OverlayValues[38] = d38
				ps167.OverlayValues[39] = d39
				ps167.OverlayValues[40] = d40
				ps167.OverlayValues[41] = d41
				ps167.OverlayValues[42] = d42
				ps167.OverlayValues[43] = d43
				ps167.OverlayValues[44] = d44
				ps167.OverlayValues[45] = d45
				ps167.OverlayValues[46] = d46
				ps167.OverlayValues[47] = d47
				ps167.OverlayValues[48] = d48
				ps167.OverlayValues[49] = d49
				ps167.OverlayValues[50] = d50
				ps167.OverlayValues[51] = d51
				ps167.OverlayValues[52] = d52
				ps167.OverlayValues[53] = d53
				ps167.OverlayValues[54] = d54
				ps167.OverlayValues[55] = d55
				ps167.OverlayValues[56] = d56
				ps167.OverlayValues[57] = d57
				ps167.OverlayValues[58] = d58
				ps167.OverlayValues[59] = d59
				ps167.OverlayValues[62] = d62
				ps167.OverlayValues[63] = d63
				ps167.OverlayValues[64] = d64
				ps167.OverlayValues[128] = d128
				ps167.OverlayValues[129] = d129
				ps167.OverlayValues[130] = d130
				ps167.OverlayValues[132] = d132
				ps167.OverlayValues[133] = d133
				ps167.OverlayValues[134] = d134
				ps167.OverlayValues[135] = d135
				ps167.OverlayValues[136] = d136
				ps167.OverlayValues[137] = d137
				ps167.OverlayValues[138] = d138
				ps167.OverlayValues[139] = d139
				ps167.OverlayValues[140] = d140
				ps167.OverlayValues[141] = d141
				ps167.OverlayValues[142] = d142
				ps167.OverlayValues[143] = d143
				ps167.OverlayValues[144] = d144
				ps167.OverlayValues[145] = d145
				ps167.OverlayValues[146] = d146
				ps167.OverlayValues[147] = d147
				ps167.OverlayValues[148] = d148
				ps167.OverlayValues[149] = d149
				ps167.OverlayValues[150] = d150
				ps167.OverlayValues[151] = d151
				ps167.OverlayValues[152] = d152
				ps167.OverlayValues[153] = d153
				ps167.OverlayValues[154] = d154
				ps167.OverlayValues[155] = d155
				ps167.OverlayValues[156] = d156
				ps167.OverlayValues[157] = d157
				ps167.OverlayValues[158] = d158
				ps167.OverlayValues[159] = d159
				ps167.OverlayValues[160] = d160
				ps167.OverlayValues[161] = d161
				ps167.OverlayValues[162] = d162
				ps167.OverlayValues[163] = d163
				ps167.OverlayValues[164] = d164
				ps167.OverlayValues[165] = d165
				ps167.OverlayValues[166] = d166
				return bbs[13].RenderPS(ps167)
			}
			if ps.General {
			}
			ps168 := scm.PhiState{General: ps.General}
			ps168.OverlayValues = make([]scm.JITValueDesc, 167)
			ps168.OverlayValues[1] = d1
			ps168.OverlayValues[2] = d2
			ps168.OverlayValues[3] = d3
			ps168.OverlayValues[4] = d4
			ps168.OverlayValues[5] = d5
			ps168.OverlayValues[6] = d6
			ps168.OverlayValues[7] = d7
			ps168.OverlayValues[8] = d8
			ps168.OverlayValues[9] = d9
			ps168.OverlayValues[10] = d10
			ps168.OverlayValues[11] = d11
			ps168.OverlayValues[12] = d12
			ps168.OverlayValues[13] = d13
			ps168.OverlayValues[14] = d14
			ps168.OverlayValues[15] = d15
			ps168.OverlayValues[17] = d17
			ps168.OverlayValues[18] = d18
			ps168.OverlayValues[19] = d19
			ps168.OverlayValues[20] = d20
			ps168.OverlayValues[21] = d21
			ps168.OverlayValues[22] = d22
			ps168.OverlayValues[24] = d24
			ps168.OverlayValues[25] = d25
			ps168.OverlayValues[26] = d26
			ps168.OverlayValues[27] = d27
			ps168.OverlayValues[28] = d28
			ps168.OverlayValues[29] = d29
			ps168.OverlayValues[30] = d30
			ps168.OverlayValues[31] = d31
			ps168.OverlayValues[32] = d32
			ps168.OverlayValues[33] = d33
			ps168.OverlayValues[34] = d34
			ps168.OverlayValues[35] = d35
			ps168.OverlayValues[36] = d36
			ps168.OverlayValues[37] = d37
			ps168.OverlayValues[38] = d38
			ps168.OverlayValues[39] = d39
			ps168.OverlayValues[40] = d40
			ps168.OverlayValues[41] = d41
			ps168.OverlayValues[42] = d42
			ps168.OverlayValues[43] = d43
			ps168.OverlayValues[44] = d44
			ps168.OverlayValues[45] = d45
			ps168.OverlayValues[46] = d46
			ps168.OverlayValues[47] = d47
			ps168.OverlayValues[48] = d48
			ps168.OverlayValues[49] = d49
			ps168.OverlayValues[50] = d50
			ps168.OverlayValues[51] = d51
			ps168.OverlayValues[52] = d52
			ps168.OverlayValues[53] = d53
			ps168.OverlayValues[54] = d54
			ps168.OverlayValues[55] = d55
			ps168.OverlayValues[56] = d56
			ps168.OverlayValues[57] = d57
			ps168.OverlayValues[58] = d58
			ps168.OverlayValues[59] = d59
			ps168.OverlayValues[62] = d62
			ps168.OverlayValues[63] = d63
			ps168.OverlayValues[64] = d64
			ps168.OverlayValues[128] = d128
			ps168.OverlayValues[129] = d129
			ps168.OverlayValues[130] = d130
			ps168.OverlayValues[132] = d132
			ps168.OverlayValues[133] = d133
			ps168.OverlayValues[134] = d134
			ps168.OverlayValues[135] = d135
			ps168.OverlayValues[136] = d136
			ps168.OverlayValues[137] = d137
			ps168.OverlayValues[138] = d138
			ps168.OverlayValues[139] = d139
			ps168.OverlayValues[140] = d140
			ps168.OverlayValues[141] = d141
			ps168.OverlayValues[142] = d142
			ps168.OverlayValues[143] = d143
			ps168.OverlayValues[144] = d144
			ps168.OverlayValues[145] = d145
			ps168.OverlayValues[146] = d146
			ps168.OverlayValues[147] = d147
			ps168.OverlayValues[148] = d148
			ps168.OverlayValues[149] = d149
			ps168.OverlayValues[150] = d150
			ps168.OverlayValues[151] = d151
			ps168.OverlayValues[152] = d152
			ps168.OverlayValues[153] = d153
			ps168.OverlayValues[154] = d154
			ps168.OverlayValues[155] = d155
			ps168.OverlayValues[156] = d156
			ps168.OverlayValues[157] = d157
			ps168.OverlayValues[158] = d158
			ps168.OverlayValues[159] = d159
			ps168.OverlayValues[160] = d160
			ps168.OverlayValues[161] = d161
			ps168.OverlayValues[162] = d162
			ps168.OverlayValues[163] = d163
			ps168.OverlayValues[164] = d164
			ps168.OverlayValues[165] = d165
			ps168.OverlayValues[166] = d166
			return bbs[12].RenderPS(ps168)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d169 := ps.PhiValues[0]
				ctx.EnsureDesc(&d169)
				ctx.EmitStoreToStack(d169, int32(bbs[2].PhiBase)+int32(0))
			}
			ps.General = true
			return bbs[2].RenderPS(ps)
		}
		lbl27 := ctx.ReserveLabel()
		lbl28 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d166.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl27)
		ctx.EmitJmp(lbl28)
		ctx.MarkLabel(lbl27)
		ctx.EmitJmp(lbl14)
		ctx.MarkLabel(lbl28)
		ctx.EmitJmp(lbl13)
		ps170 := scm.PhiState{General: true}
		ps170.OverlayValues = make([]scm.JITValueDesc, 170)
		ps170.OverlayValues[1] = d1
		ps170.OverlayValues[2] = d2
		ps170.OverlayValues[3] = d3
		ps170.OverlayValues[4] = d4
		ps170.OverlayValues[5] = d5
		ps170.OverlayValues[6] = d6
		ps170.OverlayValues[7] = d7
		ps170.OverlayValues[8] = d8
		ps170.OverlayValues[9] = d9
		ps170.OverlayValues[10] = d10
		ps170.OverlayValues[11] = d11
		ps170.OverlayValues[12] = d12
		ps170.OverlayValues[13] = d13
		ps170.OverlayValues[14] = d14
		ps170.OverlayValues[15] = d15
		ps170.OverlayValues[17] = d17
		ps170.OverlayValues[18] = d18
		ps170.OverlayValues[19] = d19
		ps170.OverlayValues[20] = d20
		ps170.OverlayValues[21] = d21
		ps170.OverlayValues[22] = d22
		ps170.OverlayValues[24] = d24
		ps170.OverlayValues[25] = d25
		ps170.OverlayValues[26] = d26
		ps170.OverlayValues[27] = d27
		ps170.OverlayValues[28] = d28
		ps170.OverlayValues[29] = d29
		ps170.OverlayValues[30] = d30
		ps170.OverlayValues[31] = d31
		ps170.OverlayValues[32] = d32
		ps170.OverlayValues[33] = d33
		ps170.OverlayValues[34] = d34
		ps170.OverlayValues[35] = d35
		ps170.OverlayValues[36] = d36
		ps170.OverlayValues[37] = d37
		ps170.OverlayValues[38] = d38
		ps170.OverlayValues[39] = d39
		ps170.OverlayValues[40] = d40
		ps170.OverlayValues[41] = d41
		ps170.OverlayValues[42] = d42
		ps170.OverlayValues[43] = d43
		ps170.OverlayValues[44] = d44
		ps170.OverlayValues[45] = d45
		ps170.OverlayValues[46] = d46
		ps170.OverlayValues[47] = d47
		ps170.OverlayValues[48] = d48
		ps170.OverlayValues[49] = d49
		ps170.OverlayValues[50] = d50
		ps170.OverlayValues[51] = d51
		ps170.OverlayValues[52] = d52
		ps170.OverlayValues[53] = d53
		ps170.OverlayValues[54] = d54
		ps170.OverlayValues[55] = d55
		ps170.OverlayValues[56] = d56
		ps170.OverlayValues[57] = d57
		ps170.OverlayValues[58] = d58
		ps170.OverlayValues[59] = d59
		ps170.OverlayValues[62] = d62
		ps170.OverlayValues[63] = d63
		ps170.OverlayValues[64] = d64
		ps170.OverlayValues[128] = d128
		ps170.OverlayValues[129] = d129
		ps170.OverlayValues[130] = d130
		ps170.OverlayValues[132] = d132
		ps170.OverlayValues[133] = d133
		ps170.OverlayValues[134] = d134
		ps170.OverlayValues[135] = d135
		ps170.OverlayValues[136] = d136
		ps170.OverlayValues[137] = d137
		ps170.OverlayValues[138] = d138
		ps170.OverlayValues[139] = d139
		ps170.OverlayValues[140] = d140
		ps170.OverlayValues[141] = d141
		ps170.OverlayValues[142] = d142
		ps170.OverlayValues[143] = d143
		ps170.OverlayValues[144] = d144
		ps170.OverlayValues[145] = d145
		ps170.OverlayValues[146] = d146
		ps170.OverlayValues[147] = d147
		ps170.OverlayValues[148] = d148
		ps170.OverlayValues[149] = d149
		ps170.OverlayValues[150] = d150
		ps170.OverlayValues[151] = d151
		ps170.OverlayValues[152] = d152
		ps170.OverlayValues[153] = d153
		ps170.OverlayValues[154] = d154
		ps170.OverlayValues[155] = d155
		ps170.OverlayValues[156] = d156
		ps170.OverlayValues[157] = d157
		ps170.OverlayValues[158] = d158
		ps170.OverlayValues[159] = d159
		ps170.OverlayValues[160] = d160
		ps170.OverlayValues[161] = d161
		ps170.OverlayValues[162] = d162
		ps170.OverlayValues[163] = d163
		ps170.OverlayValues[164] = d164
		ps170.OverlayValues[165] = d165
		ps170.OverlayValues[166] = d166
		ps170.OverlayValues[169] = d169
		ps171 := scm.PhiState{General: true}
		ps171.OverlayValues = make([]scm.JITValueDesc, 170)
		ps171.OverlayValues[1] = d1
		ps171.OverlayValues[2] = d2
		ps171.OverlayValues[3] = d3
		ps171.OverlayValues[4] = d4
		ps171.OverlayValues[5] = d5
		ps171.OverlayValues[6] = d6
		ps171.OverlayValues[7] = d7
		ps171.OverlayValues[8] = d8
		ps171.OverlayValues[9] = d9
		ps171.OverlayValues[10] = d10
		ps171.OverlayValues[11] = d11
		ps171.OverlayValues[12] = d12
		ps171.OverlayValues[13] = d13
		ps171.OverlayValues[14] = d14
		ps171.OverlayValues[15] = d15
		ps171.OverlayValues[17] = d17
		ps171.OverlayValues[18] = d18
		ps171.OverlayValues[19] = d19
		ps171.OverlayValues[20] = d20
		ps171.OverlayValues[21] = d21
		ps171.OverlayValues[22] = d22
		ps171.OverlayValues[24] = d24
		ps171.OverlayValues[25] = d25
		ps171.OverlayValues[26] = d26
		ps171.OverlayValues[27] = d27
		ps171.OverlayValues[28] = d28
		ps171.OverlayValues[29] = d29
		ps171.OverlayValues[30] = d30
		ps171.OverlayValues[31] = d31
		ps171.OverlayValues[32] = d32
		ps171.OverlayValues[33] = d33
		ps171.OverlayValues[34] = d34
		ps171.OverlayValues[35] = d35
		ps171.OverlayValues[36] = d36
		ps171.OverlayValues[37] = d37
		ps171.OverlayValues[38] = d38
		ps171.OverlayValues[39] = d39
		ps171.OverlayValues[40] = d40
		ps171.OverlayValues[41] = d41
		ps171.OverlayValues[42] = d42
		ps171.OverlayValues[43] = d43
		ps171.OverlayValues[44] = d44
		ps171.OverlayValues[45] = d45
		ps171.OverlayValues[46] = d46
		ps171.OverlayValues[47] = d47
		ps171.OverlayValues[48] = d48
		ps171.OverlayValues[49] = d49
		ps171.OverlayValues[50] = d50
		ps171.OverlayValues[51] = d51
		ps171.OverlayValues[52] = d52
		ps171.OverlayValues[53] = d53
		ps171.OverlayValues[54] = d54
		ps171.OverlayValues[55] = d55
		ps171.OverlayValues[56] = d56
		ps171.OverlayValues[57] = d57
		ps171.OverlayValues[58] = d58
		ps171.OverlayValues[59] = d59
		ps171.OverlayValues[62] = d62
		ps171.OverlayValues[63] = d63
		ps171.OverlayValues[64] = d64
		ps171.OverlayValues[128] = d128
		ps171.OverlayValues[129] = d129
		ps171.OverlayValues[130] = d130
		ps171.OverlayValues[132] = d132
		ps171.OverlayValues[133] = d133
		ps171.OverlayValues[134] = d134
		ps171.OverlayValues[135] = d135
		ps171.OverlayValues[136] = d136
		ps171.OverlayValues[137] = d137
		ps171.OverlayValues[138] = d138
		ps171.OverlayValues[139] = d139
		ps171.OverlayValues[140] = d140
		ps171.OverlayValues[141] = d141
		ps171.OverlayValues[142] = d142
		ps171.OverlayValues[143] = d143
		ps171.OverlayValues[144] = d144
		ps171.OverlayValues[145] = d145
		ps171.OverlayValues[146] = d146
		ps171.OverlayValues[147] = d147
		ps171.OverlayValues[148] = d148
		ps171.OverlayValues[149] = d149
		ps171.OverlayValues[150] = d150
		ps171.OverlayValues[151] = d151
		ps171.OverlayValues[152] = d152
		ps171.OverlayValues[153] = d153
		ps171.OverlayValues[154] = d154
		ps171.OverlayValues[155] = d155
		ps171.OverlayValues[156] = d156
		ps171.OverlayValues[157] = d157
		ps171.OverlayValues[158] = d158
		ps171.OverlayValues[159] = d159
		ps171.OverlayValues[160] = d160
		ps171.OverlayValues[161] = d161
		ps171.OverlayValues[162] = d162
		ps171.OverlayValues[163] = d163
		ps171.OverlayValues[164] = d164
		ps171.OverlayValues[165] = d165
		ps171.OverlayValues[166] = d166
		ps171.OverlayValues[169] = d169
		snap172 := d1
		snap173 := d2
		snap174 := d3
		snap175 := d4
		snap176 := d5
		snap177 := d6
		snap178 := d7
		snap179 := d8
		snap180 := d9
		snap181 := d10
		snap182 := d11
		snap183 := d12
		snap184 := d13
		snap185 := d14
		snap186 := d15
		snap187 := d17
		snap188 := d18
		snap189 := d19
		snap190 := d20
		snap191 := d21
		snap192 := d22
		snap193 := d24
		snap194 := d25
		snap195 := d26
		snap196 := d27
		snap197 := d28
		snap198 := d29
		snap199 := d30
		snap200 := d31
		snap201 := d32
		snap202 := d33
		snap203 := d34
		snap204 := d35
		snap205 := d36
		snap206 := d37
		snap207 := d38
		snap208 := d39
		snap209 := d40
		snap210 := d41
		snap211 := d42
		snap212 := d43
		snap213 := d44
		snap214 := d45
		snap215 := d46
		snap216 := d47
		snap217 := d48
		snap218 := d49
		snap219 := d50
		snap220 := d51
		snap221 := d52
		snap222 := d53
		snap223 := d54
		snap224 := d55
		snap225 := d56
		snap226 := d57
		snap227 := d58
		snap228 := d59
		snap229 := d62
		snap230 := d63
		snap231 := d64
		snap232 := d128
		snap233 := d129
		snap234 := d130
		snap235 := d132
		snap236 := d133
		snap237 := d134
		snap238 := d135
		snap239 := d136
		snap240 := d137
		snap241 := d138
		snap242 := d139
		snap243 := d140
		snap244 := d141
		snap245 := d142
		snap246 := d143
		snap247 := d144
		snap248 := d145
		snap249 := d146
		snap250 := d147
		snap251 := d148
		snap252 := d149
		snap253 := d150
		snap254 := d151
		snap255 := d152
		snap256 := d153
		snap257 := d154
		snap258 := d155
		snap259 := d156
		snap260 := d157
		snap261 := d158
		snap262 := d159
		snap263 := d160
		snap264 := d161
		snap265 := d162
		snap266 := d163
		snap267 := d164
		snap268 := d165
		snap269 := d166
		snap270 := d169
		alloc271 := ctx.SnapshotAllocState()
		if !bbs[12].Rendered {
			bbs[12].RenderPS(ps171)
		}
		ctx.RestoreAllocState(alloc271)
		d1 = snap172
		d2 = snap173
		d3 = snap174
		d4 = snap175
		d5 = snap176
		d6 = snap177
		d7 = snap178
		d8 = snap179
		d9 = snap180
		d10 = snap181
		d11 = snap182
		d12 = snap183
		d13 = snap184
		d14 = snap185
		d15 = snap186
		d17 = snap187
		d18 = snap188
		d19 = snap189
		d20 = snap190
		d21 = snap191
		d22 = snap192
		d24 = snap193
		d25 = snap194
		d26 = snap195
		d27 = snap196
		d28 = snap197
		d29 = snap198
		d30 = snap199
		d31 = snap200
		d32 = snap201
		d33 = snap202
		d34 = snap203
		d35 = snap204
		d36 = snap205
		d37 = snap206
		d38 = snap207
		d39 = snap208
		d40 = snap209
		d41 = snap210
		d42 = snap211
		d43 = snap212
		d44 = snap213
		d45 = snap214
		d46 = snap215
		d47 = snap216
		d48 = snap217
		d49 = snap218
		d50 = snap219
		d51 = snap220
		d52 = snap221
		d53 = snap222
		d54 = snap223
		d55 = snap224
		d56 = snap225
		d57 = snap226
		d58 = snap227
		d59 = snap228
		d62 = snap229
		d63 = snap230
		d64 = snap231
		d128 = snap232
		d129 = snap233
		d130 = snap234
		d132 = snap235
		d133 = snap236
		d134 = snap237
		d135 = snap238
		d136 = snap239
		d137 = snap240
		d138 = snap241
		d139 = snap242
		d140 = snap243
		d141 = snap244
		d142 = snap245
		d143 = snap246
		d144 = snap247
		d145 = snap248
		d146 = snap249
		d147 = snap250
		d148 = snap251
		d149 = snap252
		d150 = snap253
		d151 = snap254
		d152 = snap255
		d153 = snap256
		d154 = snap257
		d155 = snap258
		d156 = snap259
		d157 = snap260
		d158 = snap261
		d159 = snap262
		d160 = snap263
		d161 = snap264
		d162 = snap265
		d163 = snap266
		d164 = snap267
		d165 = snap268
		d166 = snap269
		d169 = snap270
		if !bbs[13].Rendered {
			return bbs[13].RenderPS(ps170)
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
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
		}
		if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
			d58 = ps.OverlayValues[58]
		}
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
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
		if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
			d128 = ps.OverlayValues[128]
		}
		if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
			d129 = ps.OverlayValues[129]
		}
		if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
			d130 = ps.OverlayValues[130]
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
		if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != scm.LocNone {
			d148 = ps.OverlayValues[148]
		}
		if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != scm.LocNone {
			d149 = ps.OverlayValues[149]
		}
		if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
			d150 = ps.OverlayValues[150]
		}
		if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != scm.LocNone {
			d151 = ps.OverlayValues[151]
		}
		if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != scm.LocNone {
			d152 = ps.OverlayValues[152]
		}
		if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
			d153 = ps.OverlayValues[153]
		}
		if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
			d154 = ps.OverlayValues[154]
		}
		if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != scm.LocNone {
			d155 = ps.OverlayValues[155]
		}
		if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != scm.LocNone {
			d156 = ps.OverlayValues[156]
		}
		if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != scm.LocNone {
			d157 = ps.OverlayValues[157]
		}
		if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != scm.LocNone {
			d158 = ps.OverlayValues[158]
		}
		if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != scm.LocNone {
			d159 = ps.OverlayValues[159]
		}
		if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != scm.LocNone {
			d160 = ps.OverlayValues[160]
		}
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != scm.LocNone {
			d161 = ps.OverlayValues[161]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != scm.LocNone {
			d162 = ps.OverlayValues[162]
		}
		if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != scm.LocNone {
			d163 = ps.OverlayValues[163]
		}
		if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != scm.LocNone {
			d164 = ps.OverlayValues[164]
		}
		if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != scm.LocNone {
			d165 = ps.OverlayValues[165]
		}
		if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != scm.LocNone {
			d166 = ps.OverlayValues[166]
		}
		if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != scm.LocNone {
			d169 = ps.OverlayValues[169]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		ctx.EnsureDesc(&d1)
		var d272 scm.JITValueDesc
		if d1.Loc == scm.LocImm {
			d272 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegReg(scratch, d1.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d272 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d272)
		}
		if d272.Loc == scm.LocImm {
			d272 = scm.JITValueDesc{Loc: scm.LocImm, Type: d272.Type, Imm: scm.NewInt(int64(uint64(d272.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d272.Reg, 32)
			ctx.EmitShrRegImm8(d272.Reg, 32)
		}
		if d272.Loc == scm.LocReg && d1.Loc == scm.LocReg && d272.Reg == d1.Reg {
			ctx.TransferReg(d1.Reg)
			d1.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d272)
		ctx.EmitStoreToStack(d272, int32(bbs[4].PhiBase)+int32(32))
		ctx.StabilizeDescForControlFlow(&d272)
		ctx.EnsureDesc(&d1)
		ctx.EnsureDesc(&d1)
		var d273 scm.JITValueDesc
		if d1.Loc == scm.LocImm {
			d273 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegReg(scratch, d1.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d273 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d273)
		}
		if d273.Loc == scm.LocImm {
			d273 = scm.JITValueDesc{Loc: scm.LocImm, Type: d273.Type, Imm: scm.NewInt(int64(uint64(d273.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d273.Reg, 32)
			ctx.EmitShrRegImm8(d273.Reg, 32)
		}
		if d273.Loc == scm.LocReg && d1.Loc == scm.LocReg && d273.Reg == d1.Reg {
			ctx.TransferReg(d1.Reg)
			d1.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d273)
		ctx.EmitStoreToStack(d273, int32(bbs[4].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d273)
		if ps.General {
			ctx.SyncDesc(&d2)
			if d2.Loc == scm.LocReg {
				ctx.ProtectReg(d2.Reg)
			} else if d2.Loc == scm.LocRegPair {
				ctx.ProtectReg(d2.Reg)
				ctx.ProtectReg(d2.Reg2)
			}
			d274 = d2
			if d274.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d274)
			d275 = d274
			if d275.Loc == scm.LocImm {
				d275 = scm.JITValueDesc{Loc: scm.LocImm, Type: d275.Type, Imm: scm.NewInt(int64(uint64(d275.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d275.Reg, 32)
				ctx.EmitShrRegImm8(d275.Reg, 32)
			}
			ctx.EmitStoreToStack(d275, int32(bbs[4].PhiBase)+int32(16))
			if d2.Loc == scm.LocReg {
				ctx.UnprotectReg(d2.Reg)
			} else if d2.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d2.Reg)
				ctx.UnprotectReg(d2.Reg2)
			}
		}
		ps276 := scm.PhiState{General: ps.General}
		ps276.OverlayValues = make([]scm.JITValueDesc, 276)
		ps276.OverlayValues[1] = d1
		ps276.OverlayValues[2] = d2
		ps276.OverlayValues[3] = d3
		ps276.OverlayValues[4] = d4
		ps276.OverlayValues[5] = d5
		ps276.OverlayValues[6] = d6
		ps276.OverlayValues[7] = d7
		ps276.OverlayValues[8] = d8
		ps276.OverlayValues[9] = d9
		ps276.OverlayValues[10] = d10
		ps276.OverlayValues[11] = d11
		ps276.OverlayValues[12] = d12
		ps276.OverlayValues[13] = d13
		ps276.OverlayValues[14] = d14
		ps276.OverlayValues[15] = d15
		ps276.OverlayValues[17] = d17
		ps276.OverlayValues[18] = d18
		ps276.OverlayValues[19] = d19
		ps276.OverlayValues[20] = d20
		ps276.OverlayValues[21] = d21
		ps276.OverlayValues[22] = d22
		ps276.OverlayValues[24] = d24
		ps276.OverlayValues[25] = d25
		ps276.OverlayValues[26] = d26
		ps276.OverlayValues[27] = d27
		ps276.OverlayValues[28] = d28
		ps276.OverlayValues[29] = d29
		ps276.OverlayValues[30] = d30
		ps276.OverlayValues[31] = d31
		ps276.OverlayValues[32] = d32
		ps276.OverlayValues[33] = d33
		ps276.OverlayValues[34] = d34
		ps276.OverlayValues[35] = d35
		ps276.OverlayValues[36] = d36
		ps276.OverlayValues[37] = d37
		ps276.OverlayValues[38] = d38
		ps276.OverlayValues[39] = d39
		ps276.OverlayValues[40] = d40
		ps276.OverlayValues[41] = d41
		ps276.OverlayValues[42] = d42
		ps276.OverlayValues[43] = d43
		ps276.OverlayValues[44] = d44
		ps276.OverlayValues[45] = d45
		ps276.OverlayValues[46] = d46
		ps276.OverlayValues[47] = d47
		ps276.OverlayValues[48] = d48
		ps276.OverlayValues[49] = d49
		ps276.OverlayValues[50] = d50
		ps276.OverlayValues[51] = d51
		ps276.OverlayValues[52] = d52
		ps276.OverlayValues[53] = d53
		ps276.OverlayValues[54] = d54
		ps276.OverlayValues[55] = d55
		ps276.OverlayValues[56] = d56
		ps276.OverlayValues[57] = d57
		ps276.OverlayValues[58] = d58
		ps276.OverlayValues[59] = d59
		ps276.OverlayValues[62] = d62
		ps276.OverlayValues[63] = d63
		ps276.OverlayValues[64] = d64
		ps276.OverlayValues[128] = d128
		ps276.OverlayValues[129] = d129
		ps276.OverlayValues[130] = d130
		ps276.OverlayValues[132] = d132
		ps276.OverlayValues[133] = d133
		ps276.OverlayValues[134] = d134
		ps276.OverlayValues[135] = d135
		ps276.OverlayValues[136] = d136
		ps276.OverlayValues[137] = d137
		ps276.OverlayValues[138] = d138
		ps276.OverlayValues[139] = d139
		ps276.OverlayValues[140] = d140
		ps276.OverlayValues[141] = d141
		ps276.OverlayValues[142] = d142
		ps276.OverlayValues[143] = d143
		ps276.OverlayValues[144] = d144
		ps276.OverlayValues[145] = d145
		ps276.OverlayValues[146] = d146
		ps276.OverlayValues[147] = d147
		ps276.OverlayValues[148] = d148
		ps276.OverlayValues[149] = d149
		ps276.OverlayValues[150] = d150
		ps276.OverlayValues[151] = d151
		ps276.OverlayValues[152] = d152
		ps276.OverlayValues[153] = d153
		ps276.OverlayValues[154] = d154
		ps276.OverlayValues[155] = d155
		ps276.OverlayValues[156] = d156
		ps276.OverlayValues[157] = d157
		ps276.OverlayValues[158] = d158
		ps276.OverlayValues[159] = d159
		ps276.OverlayValues[160] = d160
		ps276.OverlayValues[161] = d161
		ps276.OverlayValues[162] = d162
		ps276.OverlayValues[163] = d163
		ps276.OverlayValues[164] = d164
		ps276.OverlayValues[165] = d165
		ps276.OverlayValues[166] = d166
		ps276.OverlayValues[169] = d169
		ps276.OverlayValues[272] = d272
		ps276.OverlayValues[273] = d273
		ps276.OverlayValues[274] = d274
		ps276.OverlayValues[275] = d275
		ps276.PhiValues = make([]scm.JITValueDesc, 3)
		d277 = d2
		ps276.PhiValues[1] = d277
		if ps276.General && bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
			return result
		}
		return bbs[4].RenderPS(ps276)
		return result
	}
	bbs[4].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d278 := ps.PhiValues[0]
				ctx.EnsureDesc(&d278)
				ctx.EmitStoreToStack(d278, int32(bbs[4].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d279 := ps.PhiValues[1]
				ctx.EnsureDesc(&d279)
				ctx.EmitStoreToStack(d279, int32(bbs[4].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d280 := ps.PhiValues[2]
				ctx.EnsureDesc(&d280)
				ctx.EmitStoreToStack(d280, int32(bbs[4].PhiBase)+int32(32))
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
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
		}
		if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
			d58 = ps.OverlayValues[58]
		}
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
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
		if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
			d128 = ps.OverlayValues[128]
		}
		if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
			d129 = ps.OverlayValues[129]
		}
		if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
			d130 = ps.OverlayValues[130]
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
		if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != scm.LocNone {
			d148 = ps.OverlayValues[148]
		}
		if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != scm.LocNone {
			d149 = ps.OverlayValues[149]
		}
		if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
			d150 = ps.OverlayValues[150]
		}
		if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != scm.LocNone {
			d151 = ps.OverlayValues[151]
		}
		if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != scm.LocNone {
			d152 = ps.OverlayValues[152]
		}
		if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
			d153 = ps.OverlayValues[153]
		}
		if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
			d154 = ps.OverlayValues[154]
		}
		if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != scm.LocNone {
			d155 = ps.OverlayValues[155]
		}
		if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != scm.LocNone {
			d156 = ps.OverlayValues[156]
		}
		if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != scm.LocNone {
			d157 = ps.OverlayValues[157]
		}
		if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != scm.LocNone {
			d158 = ps.OverlayValues[158]
		}
		if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != scm.LocNone {
			d159 = ps.OverlayValues[159]
		}
		if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != scm.LocNone {
			d160 = ps.OverlayValues[160]
		}
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != scm.LocNone {
			d161 = ps.OverlayValues[161]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != scm.LocNone {
			d162 = ps.OverlayValues[162]
		}
		if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != scm.LocNone {
			d163 = ps.OverlayValues[163]
		}
		if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != scm.LocNone {
			d164 = ps.OverlayValues[164]
		}
		if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != scm.LocNone {
			d165 = ps.OverlayValues[165]
		}
		if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != scm.LocNone {
			d166 = ps.OverlayValues[166]
		}
		if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != scm.LocNone {
			d169 = ps.OverlayValues[169]
		}
		if len(ps.OverlayValues) > 272 && ps.OverlayValues[272].Loc != scm.LocNone {
			d272 = ps.OverlayValues[272]
		}
		if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != scm.LocNone {
			d273 = ps.OverlayValues[273]
		}
		if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != scm.LocNone {
			d274 = ps.OverlayValues[274]
		}
		if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != scm.LocNone {
			d275 = ps.OverlayValues[275]
		}
		if len(ps.OverlayValues) > 277 && ps.OverlayValues[277].Loc != scm.LocNone {
			d277 = ps.OverlayValues[277]
		}
		if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != scm.LocNone {
			d278 = ps.OverlayValues[278]
		}
		if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != scm.LocNone {
			d279 = ps.OverlayValues[279]
		}
		if len(ps.OverlayValues) > 280 && ps.OverlayValues[280].Loc != scm.LocNone {
			d280 = ps.OverlayValues[280]
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
		ctx.EnsureDesc(&d6)
		ctx.EnsureDesc(&d7)
		ctx.EnsureDesc(&d6)
		ctx.EnsureDesc(&d7)
		var d281 scm.JITValueDesc
		if d6.Loc == scm.LocImm && d7.Loc == scm.LocImm {
			d281 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d6.Imm.Int()) == uint64(d7.Imm.Int()))}
		} else if d7.Loc == scm.LocImm {
			r105 := ctx.AllocRegExcept(d6.Reg)
			if d7.Imm.Int() >= -2147483648 && d7.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d6.Reg, int32(d7.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d7.Imm.Int()))
				ctx.EmitCmpInt64(d6.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r105, scm.CondEqual)
			d281 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r105}
			ctx.BindReg(r105, &d281)
		} else if d6.Loc == scm.LocImm {
			r106 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d6.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d7.Reg)
			ctx.EmitSetcc(r106, scm.CondEqual)
			d281 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r106}
			ctx.BindReg(r106, &d281)
		} else {
			r107 := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitCmpInt64(d6.Reg, d7.Reg)
			ctx.EmitSetcc(r107, scm.CondEqual)
			d281 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r107}
			ctx.BindReg(r107, &d281)
		}
		d282 = d281
		ctx.EnsureDesc(&d282)
		if d282.Loc != scm.LocImm && d282.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d282.Loc == scm.LocImm {
			if d282.Imm.Bool() {
				if ps.General {
					ctx.SyncDesc(&d6)
					if d6.Loc == scm.LocReg {
						ctx.ProtectReg(d6.Reg)
					} else if d6.Loc == scm.LocRegPair {
						ctx.ProtectReg(d6.Reg)
						ctx.ProtectReg(d6.Reg2)
					}
					d283 = d6
					if d283.Loc == scm.LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d283)
					d284 = d283
					if d284.Loc == scm.LocImm {
						d284 = scm.JITValueDesc{Loc: scm.LocImm, Type: d284.Type, Imm: scm.NewInt(int64(uint64(d284.Imm.Int()) & 0xffffffff))}
					} else {
						ctx.EmitShlRegImm8(d284.Reg, 32)
						ctx.EmitShrRegImm8(d284.Reg, 32)
					}
					ctx.EmitStoreToStack(d284, int32(bbs[2].PhiBase)+int32(0))
					if d6.Loc == scm.LocReg {
						ctx.UnprotectReg(d6.Reg)
					} else if d6.Loc == scm.LocRegPair {
						ctx.UnprotectReg(d6.Reg)
						ctx.UnprotectReg(d6.Reg2)
					}
				}
				ps285 := scm.PhiState{General: ps.General}
				ps285.OverlayValues = make([]scm.JITValueDesc, 285)
				ps285.OverlayValues[1] = d1
				ps285.OverlayValues[2] = d2
				ps285.OverlayValues[3] = d3
				ps285.OverlayValues[4] = d4
				ps285.OverlayValues[5] = d5
				ps285.OverlayValues[6] = d6
				ps285.OverlayValues[7] = d7
				ps285.OverlayValues[8] = d8
				ps285.OverlayValues[9] = d9
				ps285.OverlayValues[10] = d10
				ps285.OverlayValues[11] = d11
				ps285.OverlayValues[12] = d12
				ps285.OverlayValues[13] = d13
				ps285.OverlayValues[14] = d14
				ps285.OverlayValues[15] = d15
				ps285.OverlayValues[17] = d17
				ps285.OverlayValues[18] = d18
				ps285.OverlayValues[19] = d19
				ps285.OverlayValues[20] = d20
				ps285.OverlayValues[21] = d21
				ps285.OverlayValues[22] = d22
				ps285.OverlayValues[24] = d24
				ps285.OverlayValues[25] = d25
				ps285.OverlayValues[26] = d26
				ps285.OverlayValues[27] = d27
				ps285.OverlayValues[28] = d28
				ps285.OverlayValues[29] = d29
				ps285.OverlayValues[30] = d30
				ps285.OverlayValues[31] = d31
				ps285.OverlayValues[32] = d32
				ps285.OverlayValues[33] = d33
				ps285.OverlayValues[34] = d34
				ps285.OverlayValues[35] = d35
				ps285.OverlayValues[36] = d36
				ps285.OverlayValues[37] = d37
				ps285.OverlayValues[38] = d38
				ps285.OverlayValues[39] = d39
				ps285.OverlayValues[40] = d40
				ps285.OverlayValues[41] = d41
				ps285.OverlayValues[42] = d42
				ps285.OverlayValues[43] = d43
				ps285.OverlayValues[44] = d44
				ps285.OverlayValues[45] = d45
				ps285.OverlayValues[46] = d46
				ps285.OverlayValues[47] = d47
				ps285.OverlayValues[48] = d48
				ps285.OverlayValues[49] = d49
				ps285.OverlayValues[50] = d50
				ps285.OverlayValues[51] = d51
				ps285.OverlayValues[52] = d52
				ps285.OverlayValues[53] = d53
				ps285.OverlayValues[54] = d54
				ps285.OverlayValues[55] = d55
				ps285.OverlayValues[56] = d56
				ps285.OverlayValues[57] = d57
				ps285.OverlayValues[58] = d58
				ps285.OverlayValues[59] = d59
				ps285.OverlayValues[62] = d62
				ps285.OverlayValues[63] = d63
				ps285.OverlayValues[64] = d64
				ps285.OverlayValues[128] = d128
				ps285.OverlayValues[129] = d129
				ps285.OverlayValues[130] = d130
				ps285.OverlayValues[132] = d132
				ps285.OverlayValues[133] = d133
				ps285.OverlayValues[134] = d134
				ps285.OverlayValues[135] = d135
				ps285.OverlayValues[136] = d136
				ps285.OverlayValues[137] = d137
				ps285.OverlayValues[138] = d138
				ps285.OverlayValues[139] = d139
				ps285.OverlayValues[140] = d140
				ps285.OverlayValues[141] = d141
				ps285.OverlayValues[142] = d142
				ps285.OverlayValues[143] = d143
				ps285.OverlayValues[144] = d144
				ps285.OverlayValues[145] = d145
				ps285.OverlayValues[146] = d146
				ps285.OverlayValues[147] = d147
				ps285.OverlayValues[148] = d148
				ps285.OverlayValues[149] = d149
				ps285.OverlayValues[150] = d150
				ps285.OverlayValues[151] = d151
				ps285.OverlayValues[152] = d152
				ps285.OverlayValues[153] = d153
				ps285.OverlayValues[154] = d154
				ps285.OverlayValues[155] = d155
				ps285.OverlayValues[156] = d156
				ps285.OverlayValues[157] = d157
				ps285.OverlayValues[158] = d158
				ps285.OverlayValues[159] = d159
				ps285.OverlayValues[160] = d160
				ps285.OverlayValues[161] = d161
				ps285.OverlayValues[162] = d162
				ps285.OverlayValues[163] = d163
				ps285.OverlayValues[164] = d164
				ps285.OverlayValues[165] = d165
				ps285.OverlayValues[166] = d166
				ps285.OverlayValues[169] = d169
				ps285.OverlayValues[272] = d272
				ps285.OverlayValues[273] = d273
				ps285.OverlayValues[274] = d274
				ps285.OverlayValues[275] = d275
				ps285.OverlayValues[277] = d277
				ps285.OverlayValues[278] = d278
				ps285.OverlayValues[279] = d279
				ps285.OverlayValues[280] = d280
				ps285.OverlayValues[281] = d281
				ps285.OverlayValues[282] = d282
				ps285.OverlayValues[283] = d283
				ps285.OverlayValues[284] = d284
				ps285.PhiValues = make([]scm.JITValueDesc, 1)
				d286 = d6
				ps285.PhiValues[0] = d286
				return bbs[2].RenderPS(ps285)
			}
			if ps.General {
			}
			ps287 := scm.PhiState{General: ps.General}
			ps287.OverlayValues = make([]scm.JITValueDesc, 287)
			ps287.OverlayValues[1] = d1
			ps287.OverlayValues[2] = d2
			ps287.OverlayValues[3] = d3
			ps287.OverlayValues[4] = d4
			ps287.OverlayValues[5] = d5
			ps287.OverlayValues[6] = d6
			ps287.OverlayValues[7] = d7
			ps287.OverlayValues[8] = d8
			ps287.OverlayValues[9] = d9
			ps287.OverlayValues[10] = d10
			ps287.OverlayValues[11] = d11
			ps287.OverlayValues[12] = d12
			ps287.OverlayValues[13] = d13
			ps287.OverlayValues[14] = d14
			ps287.OverlayValues[15] = d15
			ps287.OverlayValues[17] = d17
			ps287.OverlayValues[18] = d18
			ps287.OverlayValues[19] = d19
			ps287.OverlayValues[20] = d20
			ps287.OverlayValues[21] = d21
			ps287.OverlayValues[22] = d22
			ps287.OverlayValues[24] = d24
			ps287.OverlayValues[25] = d25
			ps287.OverlayValues[26] = d26
			ps287.OverlayValues[27] = d27
			ps287.OverlayValues[28] = d28
			ps287.OverlayValues[29] = d29
			ps287.OverlayValues[30] = d30
			ps287.OverlayValues[31] = d31
			ps287.OverlayValues[32] = d32
			ps287.OverlayValues[33] = d33
			ps287.OverlayValues[34] = d34
			ps287.OverlayValues[35] = d35
			ps287.OverlayValues[36] = d36
			ps287.OverlayValues[37] = d37
			ps287.OverlayValues[38] = d38
			ps287.OverlayValues[39] = d39
			ps287.OverlayValues[40] = d40
			ps287.OverlayValues[41] = d41
			ps287.OverlayValues[42] = d42
			ps287.OverlayValues[43] = d43
			ps287.OverlayValues[44] = d44
			ps287.OverlayValues[45] = d45
			ps287.OverlayValues[46] = d46
			ps287.OverlayValues[47] = d47
			ps287.OverlayValues[48] = d48
			ps287.OverlayValues[49] = d49
			ps287.OverlayValues[50] = d50
			ps287.OverlayValues[51] = d51
			ps287.OverlayValues[52] = d52
			ps287.OverlayValues[53] = d53
			ps287.OverlayValues[54] = d54
			ps287.OverlayValues[55] = d55
			ps287.OverlayValues[56] = d56
			ps287.OverlayValues[57] = d57
			ps287.OverlayValues[58] = d58
			ps287.OverlayValues[59] = d59
			ps287.OverlayValues[62] = d62
			ps287.OverlayValues[63] = d63
			ps287.OverlayValues[64] = d64
			ps287.OverlayValues[128] = d128
			ps287.OverlayValues[129] = d129
			ps287.OverlayValues[130] = d130
			ps287.OverlayValues[132] = d132
			ps287.OverlayValues[133] = d133
			ps287.OverlayValues[134] = d134
			ps287.OverlayValues[135] = d135
			ps287.OverlayValues[136] = d136
			ps287.OverlayValues[137] = d137
			ps287.OverlayValues[138] = d138
			ps287.OverlayValues[139] = d139
			ps287.OverlayValues[140] = d140
			ps287.OverlayValues[141] = d141
			ps287.OverlayValues[142] = d142
			ps287.OverlayValues[143] = d143
			ps287.OverlayValues[144] = d144
			ps287.OverlayValues[145] = d145
			ps287.OverlayValues[146] = d146
			ps287.OverlayValues[147] = d147
			ps287.OverlayValues[148] = d148
			ps287.OverlayValues[149] = d149
			ps287.OverlayValues[150] = d150
			ps287.OverlayValues[151] = d151
			ps287.OverlayValues[152] = d152
			ps287.OverlayValues[153] = d153
			ps287.OverlayValues[154] = d154
			ps287.OverlayValues[155] = d155
			ps287.OverlayValues[156] = d156
			ps287.OverlayValues[157] = d157
			ps287.OverlayValues[158] = d158
			ps287.OverlayValues[159] = d159
			ps287.OverlayValues[160] = d160
			ps287.OverlayValues[161] = d161
			ps287.OverlayValues[162] = d162
			ps287.OverlayValues[163] = d163
			ps287.OverlayValues[164] = d164
			ps287.OverlayValues[165] = d165
			ps287.OverlayValues[166] = d166
			ps287.OverlayValues[169] = d169
			ps287.OverlayValues[272] = d272
			ps287.OverlayValues[273] = d273
			ps287.OverlayValues[274] = d274
			ps287.OverlayValues[275] = d275
			ps287.OverlayValues[277] = d277
			ps287.OverlayValues[278] = d278
			ps287.OverlayValues[279] = d279
			ps287.OverlayValues[280] = d280
			ps287.OverlayValues[281] = d281
			ps287.OverlayValues[282] = d282
			ps287.OverlayValues[283] = d283
			ps287.OverlayValues[284] = d284
			ps287.OverlayValues[286] = d286
			return bbs[6].RenderPS(ps287)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d288 := ps.PhiValues[0]
				ctx.EnsureDesc(&d288)
				ctx.EmitStoreToStack(d288, int32(bbs[4].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d289 := ps.PhiValues[1]
				ctx.EnsureDesc(&d289)
				ctx.EmitStoreToStack(d289, int32(bbs[4].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d290 := ps.PhiValues[2]
				ctx.EnsureDesc(&d290)
				ctx.EmitStoreToStack(d290, int32(bbs[4].PhiBase)+int32(32))
			}
			ps.General = true
			return bbs[4].RenderPS(ps)
		}
		lbl29 := ctx.ReserveLabel()
		lbl30 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d282.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl29)
		ctx.EmitJmp(lbl30)
		ctx.MarkLabel(lbl29)
		ctx.SyncDesc(&d6)
		if d6.Loc == scm.LocReg {
			ctx.ProtectReg(d6.Reg)
		} else if d6.Loc == scm.LocRegPair {
			ctx.ProtectReg(d6.Reg)
			ctx.ProtectReg(d6.Reg2)
		}
		d291 = d6
		if d291.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d291)
		d292 = d291
		if d292.Loc == scm.LocImm {
			d292 = scm.JITValueDesc{Loc: scm.LocImm, Type: d292.Type, Imm: scm.NewInt(int64(uint64(d292.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d292.Reg, 32)
			ctx.EmitShrRegImm8(d292.Reg, 32)
		}
		ctx.EmitStoreToStack(d292, int32(bbs[2].PhiBase)+int32(0))
		if d6.Loc == scm.LocReg {
			ctx.UnprotectReg(d6.Reg)
		} else if d6.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d6.Reg)
			ctx.UnprotectReg(d6.Reg2)
		}
		ctx.EmitJmp(lbl3)
		ctx.MarkLabel(lbl30)
		ctx.EmitJmp(lbl7)
		ps293 := scm.PhiState{General: true}
		ps293.OverlayValues = make([]scm.JITValueDesc, 293)
		ps293.OverlayValues[1] = d1
		ps293.OverlayValues[2] = d2
		ps293.OverlayValues[3] = d3
		ps293.OverlayValues[4] = d4
		ps293.OverlayValues[5] = d5
		ps293.OverlayValues[6] = d6
		ps293.OverlayValues[7] = d7
		ps293.OverlayValues[8] = d8
		ps293.OverlayValues[9] = d9
		ps293.OverlayValues[10] = d10
		ps293.OverlayValues[11] = d11
		ps293.OverlayValues[12] = d12
		ps293.OverlayValues[13] = d13
		ps293.OverlayValues[14] = d14
		ps293.OverlayValues[15] = d15
		ps293.OverlayValues[17] = d17
		ps293.OverlayValues[18] = d18
		ps293.OverlayValues[19] = d19
		ps293.OverlayValues[20] = d20
		ps293.OverlayValues[21] = d21
		ps293.OverlayValues[22] = d22
		ps293.OverlayValues[24] = d24
		ps293.OverlayValues[25] = d25
		ps293.OverlayValues[26] = d26
		ps293.OverlayValues[27] = d27
		ps293.OverlayValues[28] = d28
		ps293.OverlayValues[29] = d29
		ps293.OverlayValues[30] = d30
		ps293.OverlayValues[31] = d31
		ps293.OverlayValues[32] = d32
		ps293.OverlayValues[33] = d33
		ps293.OverlayValues[34] = d34
		ps293.OverlayValues[35] = d35
		ps293.OverlayValues[36] = d36
		ps293.OverlayValues[37] = d37
		ps293.OverlayValues[38] = d38
		ps293.OverlayValues[39] = d39
		ps293.OverlayValues[40] = d40
		ps293.OverlayValues[41] = d41
		ps293.OverlayValues[42] = d42
		ps293.OverlayValues[43] = d43
		ps293.OverlayValues[44] = d44
		ps293.OverlayValues[45] = d45
		ps293.OverlayValues[46] = d46
		ps293.OverlayValues[47] = d47
		ps293.OverlayValues[48] = d48
		ps293.OverlayValues[49] = d49
		ps293.OverlayValues[50] = d50
		ps293.OverlayValues[51] = d51
		ps293.OverlayValues[52] = d52
		ps293.OverlayValues[53] = d53
		ps293.OverlayValues[54] = d54
		ps293.OverlayValues[55] = d55
		ps293.OverlayValues[56] = d56
		ps293.OverlayValues[57] = d57
		ps293.OverlayValues[58] = d58
		ps293.OverlayValues[59] = d59
		ps293.OverlayValues[62] = d62
		ps293.OverlayValues[63] = d63
		ps293.OverlayValues[64] = d64
		ps293.OverlayValues[128] = d128
		ps293.OverlayValues[129] = d129
		ps293.OverlayValues[130] = d130
		ps293.OverlayValues[132] = d132
		ps293.OverlayValues[133] = d133
		ps293.OverlayValues[134] = d134
		ps293.OverlayValues[135] = d135
		ps293.OverlayValues[136] = d136
		ps293.OverlayValues[137] = d137
		ps293.OverlayValues[138] = d138
		ps293.OverlayValues[139] = d139
		ps293.OverlayValues[140] = d140
		ps293.OverlayValues[141] = d141
		ps293.OverlayValues[142] = d142
		ps293.OverlayValues[143] = d143
		ps293.OverlayValues[144] = d144
		ps293.OverlayValues[145] = d145
		ps293.OverlayValues[146] = d146
		ps293.OverlayValues[147] = d147
		ps293.OverlayValues[148] = d148
		ps293.OverlayValues[149] = d149
		ps293.OverlayValues[150] = d150
		ps293.OverlayValues[151] = d151
		ps293.OverlayValues[152] = d152
		ps293.OverlayValues[153] = d153
		ps293.OverlayValues[154] = d154
		ps293.OverlayValues[155] = d155
		ps293.OverlayValues[156] = d156
		ps293.OverlayValues[157] = d157
		ps293.OverlayValues[158] = d158
		ps293.OverlayValues[159] = d159
		ps293.OverlayValues[160] = d160
		ps293.OverlayValues[161] = d161
		ps293.OverlayValues[162] = d162
		ps293.OverlayValues[163] = d163
		ps293.OverlayValues[164] = d164
		ps293.OverlayValues[165] = d165
		ps293.OverlayValues[166] = d166
		ps293.OverlayValues[169] = d169
		ps293.OverlayValues[272] = d272
		ps293.OverlayValues[273] = d273
		ps293.OverlayValues[274] = d274
		ps293.OverlayValues[275] = d275
		ps293.OverlayValues[277] = d277
		ps293.OverlayValues[278] = d278
		ps293.OverlayValues[279] = d279
		ps293.OverlayValues[280] = d280
		ps293.OverlayValues[281] = d281
		ps293.OverlayValues[282] = d282
		ps293.OverlayValues[283] = d283
		ps293.OverlayValues[284] = d284
		ps293.OverlayValues[286] = d286
		ps293.OverlayValues[288] = d288
		ps293.OverlayValues[289] = d289
		ps293.OverlayValues[290] = d290
		ps293.OverlayValues[291] = d291
		ps293.OverlayValues[292] = d292
		ps293.PhiValues = make([]scm.JITValueDesc, 1)
		d295 = d6
		ps293.PhiValues[0] = d295
		ps294 := scm.PhiState{General: true}
		ps294.OverlayValues = make([]scm.JITValueDesc, 296)
		ps294.OverlayValues[1] = d1
		ps294.OverlayValues[2] = d2
		ps294.OverlayValues[3] = d3
		ps294.OverlayValues[4] = d4
		ps294.OverlayValues[5] = d5
		ps294.OverlayValues[6] = d6
		ps294.OverlayValues[7] = d7
		ps294.OverlayValues[8] = d8
		ps294.OverlayValues[9] = d9
		ps294.OverlayValues[10] = d10
		ps294.OverlayValues[11] = d11
		ps294.OverlayValues[12] = d12
		ps294.OverlayValues[13] = d13
		ps294.OverlayValues[14] = d14
		ps294.OverlayValues[15] = d15
		ps294.OverlayValues[17] = d17
		ps294.OverlayValues[18] = d18
		ps294.OverlayValues[19] = d19
		ps294.OverlayValues[20] = d20
		ps294.OverlayValues[21] = d21
		ps294.OverlayValues[22] = d22
		ps294.OverlayValues[24] = d24
		ps294.OverlayValues[25] = d25
		ps294.OverlayValues[26] = d26
		ps294.OverlayValues[27] = d27
		ps294.OverlayValues[28] = d28
		ps294.OverlayValues[29] = d29
		ps294.OverlayValues[30] = d30
		ps294.OverlayValues[31] = d31
		ps294.OverlayValues[32] = d32
		ps294.OverlayValues[33] = d33
		ps294.OverlayValues[34] = d34
		ps294.OverlayValues[35] = d35
		ps294.OverlayValues[36] = d36
		ps294.OverlayValues[37] = d37
		ps294.OverlayValues[38] = d38
		ps294.OverlayValues[39] = d39
		ps294.OverlayValues[40] = d40
		ps294.OverlayValues[41] = d41
		ps294.OverlayValues[42] = d42
		ps294.OverlayValues[43] = d43
		ps294.OverlayValues[44] = d44
		ps294.OverlayValues[45] = d45
		ps294.OverlayValues[46] = d46
		ps294.OverlayValues[47] = d47
		ps294.OverlayValues[48] = d48
		ps294.OverlayValues[49] = d49
		ps294.OverlayValues[50] = d50
		ps294.OverlayValues[51] = d51
		ps294.OverlayValues[52] = d52
		ps294.OverlayValues[53] = d53
		ps294.OverlayValues[54] = d54
		ps294.OverlayValues[55] = d55
		ps294.OverlayValues[56] = d56
		ps294.OverlayValues[57] = d57
		ps294.OverlayValues[58] = d58
		ps294.OverlayValues[59] = d59
		ps294.OverlayValues[62] = d62
		ps294.OverlayValues[63] = d63
		ps294.OverlayValues[64] = d64
		ps294.OverlayValues[128] = d128
		ps294.OverlayValues[129] = d129
		ps294.OverlayValues[130] = d130
		ps294.OverlayValues[132] = d132
		ps294.OverlayValues[133] = d133
		ps294.OverlayValues[134] = d134
		ps294.OverlayValues[135] = d135
		ps294.OverlayValues[136] = d136
		ps294.OverlayValues[137] = d137
		ps294.OverlayValues[138] = d138
		ps294.OverlayValues[139] = d139
		ps294.OverlayValues[140] = d140
		ps294.OverlayValues[141] = d141
		ps294.OverlayValues[142] = d142
		ps294.OverlayValues[143] = d143
		ps294.OverlayValues[144] = d144
		ps294.OverlayValues[145] = d145
		ps294.OverlayValues[146] = d146
		ps294.OverlayValues[147] = d147
		ps294.OverlayValues[148] = d148
		ps294.OverlayValues[149] = d149
		ps294.OverlayValues[150] = d150
		ps294.OverlayValues[151] = d151
		ps294.OverlayValues[152] = d152
		ps294.OverlayValues[153] = d153
		ps294.OverlayValues[154] = d154
		ps294.OverlayValues[155] = d155
		ps294.OverlayValues[156] = d156
		ps294.OverlayValues[157] = d157
		ps294.OverlayValues[158] = d158
		ps294.OverlayValues[159] = d159
		ps294.OverlayValues[160] = d160
		ps294.OverlayValues[161] = d161
		ps294.OverlayValues[162] = d162
		ps294.OverlayValues[163] = d163
		ps294.OverlayValues[164] = d164
		ps294.OverlayValues[165] = d165
		ps294.OverlayValues[166] = d166
		ps294.OverlayValues[169] = d169
		ps294.OverlayValues[272] = d272
		ps294.OverlayValues[273] = d273
		ps294.OverlayValues[274] = d274
		ps294.OverlayValues[275] = d275
		ps294.OverlayValues[277] = d277
		ps294.OverlayValues[278] = d278
		ps294.OverlayValues[279] = d279
		ps294.OverlayValues[280] = d280
		ps294.OverlayValues[281] = d281
		ps294.OverlayValues[282] = d282
		ps294.OverlayValues[283] = d283
		ps294.OverlayValues[284] = d284
		ps294.OverlayValues[286] = d286
		ps294.OverlayValues[288] = d288
		ps294.OverlayValues[289] = d289
		ps294.OverlayValues[290] = d290
		ps294.OverlayValues[291] = d291
		ps294.OverlayValues[292] = d292
		ps294.OverlayValues[295] = d295
		snap296 := d1
		snap297 := d2
		snap298 := d3
		snap299 := d4
		snap300 := d5
		snap301 := d6
		snap302 := d7
		snap303 := d8
		snap304 := d9
		snap305 := d10
		snap306 := d11
		snap307 := d12
		snap308 := d13
		snap309 := d14
		snap310 := d15
		snap311 := d17
		snap312 := d18
		snap313 := d19
		snap314 := d20
		snap315 := d21
		snap316 := d22
		snap317 := d24
		snap318 := d25
		snap319 := d26
		snap320 := d27
		snap321 := d28
		snap322 := d29
		snap323 := d30
		snap324 := d31
		snap325 := d32
		snap326 := d33
		snap327 := d34
		snap328 := d35
		snap329 := d36
		snap330 := d37
		snap331 := d38
		snap332 := d39
		snap333 := d40
		snap334 := d41
		snap335 := d42
		snap336 := d43
		snap337 := d44
		snap338 := d45
		snap339 := d46
		snap340 := d47
		snap341 := d48
		snap342 := d49
		snap343 := d50
		snap344 := d51
		snap345 := d52
		snap346 := d53
		snap347 := d54
		snap348 := d55
		snap349 := d56
		snap350 := d57
		snap351 := d58
		snap352 := d59
		snap353 := d62
		snap354 := d63
		snap355 := d64
		snap356 := d128
		snap357 := d129
		snap358 := d130
		snap359 := d132
		snap360 := d133
		snap361 := d134
		snap362 := d135
		snap363 := d136
		snap364 := d137
		snap365 := d138
		snap366 := d139
		snap367 := d140
		snap368 := d141
		snap369 := d142
		snap370 := d143
		snap371 := d144
		snap372 := d145
		snap373 := d146
		snap374 := d147
		snap375 := d148
		snap376 := d149
		snap377 := d150
		snap378 := d151
		snap379 := d152
		snap380 := d153
		snap381 := d154
		snap382 := d155
		snap383 := d156
		snap384 := d157
		snap385 := d158
		snap386 := d159
		snap387 := d160
		snap388 := d161
		snap389 := d162
		snap390 := d163
		snap391 := d164
		snap392 := d165
		snap393 := d166
		snap394 := d169
		snap395 := d272
		snap396 := d273
		snap397 := d274
		snap398 := d275
		snap399 := d277
		snap400 := d278
		snap401 := d279
		snap402 := d280
		snap403 := d281
		snap404 := d282
		snap405 := d283
		snap406 := d284
		snap407 := d286
		snap408 := d288
		snap409 := d289
		snap410 := d290
		snap411 := d291
		snap412 := d292
		snap413 := d295
		alloc414 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps293)
		}
		ctx.RestoreAllocState(alloc414)
		d1 = snap296
		d2 = snap297
		d3 = snap298
		d4 = snap299
		d5 = snap300
		d6 = snap301
		d7 = snap302
		d8 = snap303
		d9 = snap304
		d10 = snap305
		d11 = snap306
		d12 = snap307
		d13 = snap308
		d14 = snap309
		d15 = snap310
		d17 = snap311
		d18 = snap312
		d19 = snap313
		d20 = snap314
		d21 = snap315
		d22 = snap316
		d24 = snap317
		d25 = snap318
		d26 = snap319
		d27 = snap320
		d28 = snap321
		d29 = snap322
		d30 = snap323
		d31 = snap324
		d32 = snap325
		d33 = snap326
		d34 = snap327
		d35 = snap328
		d36 = snap329
		d37 = snap330
		d38 = snap331
		d39 = snap332
		d40 = snap333
		d41 = snap334
		d42 = snap335
		d43 = snap336
		d44 = snap337
		d45 = snap338
		d46 = snap339
		d47 = snap340
		d48 = snap341
		d49 = snap342
		d50 = snap343
		d51 = snap344
		d52 = snap345
		d53 = snap346
		d54 = snap347
		d55 = snap348
		d56 = snap349
		d57 = snap350
		d58 = snap351
		d59 = snap352
		d62 = snap353
		d63 = snap354
		d64 = snap355
		d128 = snap356
		d129 = snap357
		d130 = snap358
		d132 = snap359
		d133 = snap360
		d134 = snap361
		d135 = snap362
		d136 = snap363
		d137 = snap364
		d138 = snap365
		d139 = snap366
		d140 = snap367
		d141 = snap368
		d142 = snap369
		d143 = snap370
		d144 = snap371
		d145 = snap372
		d146 = snap373
		d147 = snap374
		d148 = snap375
		d149 = snap376
		d150 = snap377
		d151 = snap378
		d152 = snap379
		d153 = snap380
		d154 = snap381
		d155 = snap382
		d156 = snap383
		d157 = snap384
		d158 = snap385
		d159 = snap386
		d160 = snap387
		d161 = snap388
		d162 = snap389
		d163 = snap390
		d164 = snap391
		d165 = snap392
		d166 = snap393
		d169 = snap394
		d272 = snap395
		d273 = snap396
		d274 = snap397
		d275 = snap398
		d277 = snap399
		d278 = snap400
		d279 = snap401
		d280 = snap402
		d281 = snap403
		d282 = snap404
		d283 = snap405
		d284 = snap406
		d286 = snap407
		d288 = snap408
		d289 = snap409
		d290 = snap410
		d291 = snap411
		d292 = snap412
		d295 = snap413
		if !bbs[6].Rendered {
			return bbs[6].RenderPS(ps294)
		}
		return result
		ctx.FreeDesc(&d281)
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
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
		}
		if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
			d58 = ps.OverlayValues[58]
		}
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
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
		if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
			d128 = ps.OverlayValues[128]
		}
		if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
			d129 = ps.OverlayValues[129]
		}
		if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
			d130 = ps.OverlayValues[130]
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
		if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != scm.LocNone {
			d148 = ps.OverlayValues[148]
		}
		if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != scm.LocNone {
			d149 = ps.OverlayValues[149]
		}
		if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
			d150 = ps.OverlayValues[150]
		}
		if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != scm.LocNone {
			d151 = ps.OverlayValues[151]
		}
		if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != scm.LocNone {
			d152 = ps.OverlayValues[152]
		}
		if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
			d153 = ps.OverlayValues[153]
		}
		if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
			d154 = ps.OverlayValues[154]
		}
		if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != scm.LocNone {
			d155 = ps.OverlayValues[155]
		}
		if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != scm.LocNone {
			d156 = ps.OverlayValues[156]
		}
		if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != scm.LocNone {
			d157 = ps.OverlayValues[157]
		}
		if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != scm.LocNone {
			d158 = ps.OverlayValues[158]
		}
		if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != scm.LocNone {
			d159 = ps.OverlayValues[159]
		}
		if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != scm.LocNone {
			d160 = ps.OverlayValues[160]
		}
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != scm.LocNone {
			d161 = ps.OverlayValues[161]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != scm.LocNone {
			d162 = ps.OverlayValues[162]
		}
		if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != scm.LocNone {
			d163 = ps.OverlayValues[163]
		}
		if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != scm.LocNone {
			d164 = ps.OverlayValues[164]
		}
		if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != scm.LocNone {
			d165 = ps.OverlayValues[165]
		}
		if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != scm.LocNone {
			d166 = ps.OverlayValues[166]
		}
		if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != scm.LocNone {
			d169 = ps.OverlayValues[169]
		}
		if len(ps.OverlayValues) > 272 && ps.OverlayValues[272].Loc != scm.LocNone {
			d272 = ps.OverlayValues[272]
		}
		if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != scm.LocNone {
			d273 = ps.OverlayValues[273]
		}
		if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != scm.LocNone {
			d274 = ps.OverlayValues[274]
		}
		if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != scm.LocNone {
			d275 = ps.OverlayValues[275]
		}
		if len(ps.OverlayValues) > 277 && ps.OverlayValues[277].Loc != scm.LocNone {
			d277 = ps.OverlayValues[277]
		}
		if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != scm.LocNone {
			d278 = ps.OverlayValues[278]
		}
		if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != scm.LocNone {
			d279 = ps.OverlayValues[279]
		}
		if len(ps.OverlayValues) > 280 && ps.OverlayValues[280].Loc != scm.LocNone {
			d280 = ps.OverlayValues[280]
		}
		if len(ps.OverlayValues) > 281 && ps.OverlayValues[281].Loc != scm.LocNone {
			d281 = ps.OverlayValues[281]
		}
		if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != scm.LocNone {
			d282 = ps.OverlayValues[282]
		}
		if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != scm.LocNone {
			d283 = ps.OverlayValues[283]
		}
		if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != scm.LocNone {
			d284 = ps.OverlayValues[284]
		}
		if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != scm.LocNone {
			d286 = ps.OverlayValues[286]
		}
		if len(ps.OverlayValues) > 288 && ps.OverlayValues[288].Loc != scm.LocNone {
			d288 = ps.OverlayValues[288]
		}
		if len(ps.OverlayValues) > 289 && ps.OverlayValues[289].Loc != scm.LocNone {
			d289 = ps.OverlayValues[289]
		}
		if len(ps.OverlayValues) > 290 && ps.OverlayValues[290].Loc != scm.LocNone {
			d290 = ps.OverlayValues[290]
		}
		if len(ps.OverlayValues) > 291 && ps.OverlayValues[291].Loc != scm.LocNone {
			d291 = ps.OverlayValues[291]
		}
		if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != scm.LocNone {
			d292 = ps.OverlayValues[292]
		}
		if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
			d295 = ps.OverlayValues[295]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		ctx.EnsureDesc(&d1)
		var d415 scm.JITValueDesc
		if d1.Loc == scm.LocImm {
			d415 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegReg(scratch, d1.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d415 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d415)
		}
		if d415.Loc == scm.LocImm {
			d415 = scm.JITValueDesc{Loc: scm.LocImm, Type: d415.Type, Imm: scm.NewInt(int64(uint64(d415.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d415.Reg, 32)
			ctx.EmitShrRegImm8(d415.Reg, 32)
		}
		if d415.Loc == scm.LocReg && d1.Loc == scm.LocReg && d415.Reg == d1.Reg {
			ctx.TransferReg(d1.Reg)
			d1.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d415)
		ctx.EmitStoreToStack(d415, int32(bbs[4].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d415)
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
			d416 = d1
			if d416.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d416)
			d417 = d416
			if d417.Loc == scm.LocImm {
				d417 = scm.JITValueDesc{Loc: scm.LocImm, Type: d417.Type, Imm: scm.NewInt(int64(uint64(d417.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d417.Reg, 32)
				ctx.EmitShrRegImm8(d417.Reg, 32)
			}
			ctx.EmitStoreToStack(d417, int32(bbs[4].PhiBase)+int32(16))
			d418 = d3
			if d418.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d418)
			d419 = d418
			if d419.Loc == scm.LocImm {
				d419 = scm.JITValueDesc{Loc: scm.LocImm, Type: d419.Type, Imm: scm.NewInt(int64(uint64(d419.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d419.Reg, 32)
				ctx.EmitShrRegImm8(d419.Reg, 32)
			}
			ctx.EmitStoreToStack(d419, int32(bbs[4].PhiBase)+int32(32))
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
		ps420 := scm.PhiState{General: ps.General}
		ps420.OverlayValues = make([]scm.JITValueDesc, 420)
		ps420.OverlayValues[1] = d1
		ps420.OverlayValues[2] = d2
		ps420.OverlayValues[3] = d3
		ps420.OverlayValues[4] = d4
		ps420.OverlayValues[5] = d5
		ps420.OverlayValues[6] = d6
		ps420.OverlayValues[7] = d7
		ps420.OverlayValues[8] = d8
		ps420.OverlayValues[9] = d9
		ps420.OverlayValues[10] = d10
		ps420.OverlayValues[11] = d11
		ps420.OverlayValues[12] = d12
		ps420.OverlayValues[13] = d13
		ps420.OverlayValues[14] = d14
		ps420.OverlayValues[15] = d15
		ps420.OverlayValues[17] = d17
		ps420.OverlayValues[18] = d18
		ps420.OverlayValues[19] = d19
		ps420.OverlayValues[20] = d20
		ps420.OverlayValues[21] = d21
		ps420.OverlayValues[22] = d22
		ps420.OverlayValues[24] = d24
		ps420.OverlayValues[25] = d25
		ps420.OverlayValues[26] = d26
		ps420.OverlayValues[27] = d27
		ps420.OverlayValues[28] = d28
		ps420.OverlayValues[29] = d29
		ps420.OverlayValues[30] = d30
		ps420.OverlayValues[31] = d31
		ps420.OverlayValues[32] = d32
		ps420.OverlayValues[33] = d33
		ps420.OverlayValues[34] = d34
		ps420.OverlayValues[35] = d35
		ps420.OverlayValues[36] = d36
		ps420.OverlayValues[37] = d37
		ps420.OverlayValues[38] = d38
		ps420.OverlayValues[39] = d39
		ps420.OverlayValues[40] = d40
		ps420.OverlayValues[41] = d41
		ps420.OverlayValues[42] = d42
		ps420.OverlayValues[43] = d43
		ps420.OverlayValues[44] = d44
		ps420.OverlayValues[45] = d45
		ps420.OverlayValues[46] = d46
		ps420.OverlayValues[47] = d47
		ps420.OverlayValues[48] = d48
		ps420.OverlayValues[49] = d49
		ps420.OverlayValues[50] = d50
		ps420.OverlayValues[51] = d51
		ps420.OverlayValues[52] = d52
		ps420.OverlayValues[53] = d53
		ps420.OverlayValues[54] = d54
		ps420.OverlayValues[55] = d55
		ps420.OverlayValues[56] = d56
		ps420.OverlayValues[57] = d57
		ps420.OverlayValues[58] = d58
		ps420.OverlayValues[59] = d59
		ps420.OverlayValues[62] = d62
		ps420.OverlayValues[63] = d63
		ps420.OverlayValues[64] = d64
		ps420.OverlayValues[128] = d128
		ps420.OverlayValues[129] = d129
		ps420.OverlayValues[130] = d130
		ps420.OverlayValues[132] = d132
		ps420.OverlayValues[133] = d133
		ps420.OverlayValues[134] = d134
		ps420.OverlayValues[135] = d135
		ps420.OverlayValues[136] = d136
		ps420.OverlayValues[137] = d137
		ps420.OverlayValues[138] = d138
		ps420.OverlayValues[139] = d139
		ps420.OverlayValues[140] = d140
		ps420.OverlayValues[141] = d141
		ps420.OverlayValues[142] = d142
		ps420.OverlayValues[143] = d143
		ps420.OverlayValues[144] = d144
		ps420.OverlayValues[145] = d145
		ps420.OverlayValues[146] = d146
		ps420.OverlayValues[147] = d147
		ps420.OverlayValues[148] = d148
		ps420.OverlayValues[149] = d149
		ps420.OverlayValues[150] = d150
		ps420.OverlayValues[151] = d151
		ps420.OverlayValues[152] = d152
		ps420.OverlayValues[153] = d153
		ps420.OverlayValues[154] = d154
		ps420.OverlayValues[155] = d155
		ps420.OverlayValues[156] = d156
		ps420.OverlayValues[157] = d157
		ps420.OverlayValues[158] = d158
		ps420.OverlayValues[159] = d159
		ps420.OverlayValues[160] = d160
		ps420.OverlayValues[161] = d161
		ps420.OverlayValues[162] = d162
		ps420.OverlayValues[163] = d163
		ps420.OverlayValues[164] = d164
		ps420.OverlayValues[165] = d165
		ps420.OverlayValues[166] = d166
		ps420.OverlayValues[169] = d169
		ps420.OverlayValues[272] = d272
		ps420.OverlayValues[273] = d273
		ps420.OverlayValues[274] = d274
		ps420.OverlayValues[275] = d275
		ps420.OverlayValues[277] = d277
		ps420.OverlayValues[278] = d278
		ps420.OverlayValues[279] = d279
		ps420.OverlayValues[280] = d280
		ps420.OverlayValues[281] = d281
		ps420.OverlayValues[282] = d282
		ps420.OverlayValues[283] = d283
		ps420.OverlayValues[284] = d284
		ps420.OverlayValues[286] = d286
		ps420.OverlayValues[288] = d288
		ps420.OverlayValues[289] = d289
		ps420.OverlayValues[290] = d290
		ps420.OverlayValues[291] = d291
		ps420.OverlayValues[292] = d292
		ps420.OverlayValues[295] = d295
		ps420.OverlayValues[415] = d415
		ps420.OverlayValues[416] = d416
		ps420.OverlayValues[417] = d417
		ps420.OverlayValues[418] = d418
		ps420.OverlayValues[419] = d419
		ps420.PhiValues = make([]scm.JITValueDesc, 3)
		d421 = d1
		ps420.PhiValues[1] = d421
		d422 = d3
		ps420.PhiValues[2] = d422
		if ps420.General && bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
			return result
		}
		return bbs[4].RenderPS(ps420)
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
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
		}
		if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
			d58 = ps.OverlayValues[58]
		}
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
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
		if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
			d128 = ps.OverlayValues[128]
		}
		if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
			d129 = ps.OverlayValues[129]
		}
		if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
			d130 = ps.OverlayValues[130]
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
		if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != scm.LocNone {
			d148 = ps.OverlayValues[148]
		}
		if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != scm.LocNone {
			d149 = ps.OverlayValues[149]
		}
		if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
			d150 = ps.OverlayValues[150]
		}
		if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != scm.LocNone {
			d151 = ps.OverlayValues[151]
		}
		if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != scm.LocNone {
			d152 = ps.OverlayValues[152]
		}
		if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
			d153 = ps.OverlayValues[153]
		}
		if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
			d154 = ps.OverlayValues[154]
		}
		if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != scm.LocNone {
			d155 = ps.OverlayValues[155]
		}
		if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != scm.LocNone {
			d156 = ps.OverlayValues[156]
		}
		if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != scm.LocNone {
			d157 = ps.OverlayValues[157]
		}
		if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != scm.LocNone {
			d158 = ps.OverlayValues[158]
		}
		if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != scm.LocNone {
			d159 = ps.OverlayValues[159]
		}
		if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != scm.LocNone {
			d160 = ps.OverlayValues[160]
		}
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != scm.LocNone {
			d161 = ps.OverlayValues[161]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != scm.LocNone {
			d162 = ps.OverlayValues[162]
		}
		if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != scm.LocNone {
			d163 = ps.OverlayValues[163]
		}
		if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != scm.LocNone {
			d164 = ps.OverlayValues[164]
		}
		if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != scm.LocNone {
			d165 = ps.OverlayValues[165]
		}
		if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != scm.LocNone {
			d166 = ps.OverlayValues[166]
		}
		if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != scm.LocNone {
			d169 = ps.OverlayValues[169]
		}
		if len(ps.OverlayValues) > 272 && ps.OverlayValues[272].Loc != scm.LocNone {
			d272 = ps.OverlayValues[272]
		}
		if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != scm.LocNone {
			d273 = ps.OverlayValues[273]
		}
		if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != scm.LocNone {
			d274 = ps.OverlayValues[274]
		}
		if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != scm.LocNone {
			d275 = ps.OverlayValues[275]
		}
		if len(ps.OverlayValues) > 277 && ps.OverlayValues[277].Loc != scm.LocNone {
			d277 = ps.OverlayValues[277]
		}
		if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != scm.LocNone {
			d278 = ps.OverlayValues[278]
		}
		if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != scm.LocNone {
			d279 = ps.OverlayValues[279]
		}
		if len(ps.OverlayValues) > 280 && ps.OverlayValues[280].Loc != scm.LocNone {
			d280 = ps.OverlayValues[280]
		}
		if len(ps.OverlayValues) > 281 && ps.OverlayValues[281].Loc != scm.LocNone {
			d281 = ps.OverlayValues[281]
		}
		if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != scm.LocNone {
			d282 = ps.OverlayValues[282]
		}
		if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != scm.LocNone {
			d283 = ps.OverlayValues[283]
		}
		if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != scm.LocNone {
			d284 = ps.OverlayValues[284]
		}
		if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != scm.LocNone {
			d286 = ps.OverlayValues[286]
		}
		if len(ps.OverlayValues) > 288 && ps.OverlayValues[288].Loc != scm.LocNone {
			d288 = ps.OverlayValues[288]
		}
		if len(ps.OverlayValues) > 289 && ps.OverlayValues[289].Loc != scm.LocNone {
			d289 = ps.OverlayValues[289]
		}
		if len(ps.OverlayValues) > 290 && ps.OverlayValues[290].Loc != scm.LocNone {
			d290 = ps.OverlayValues[290]
		}
		if len(ps.OverlayValues) > 291 && ps.OverlayValues[291].Loc != scm.LocNone {
			d291 = ps.OverlayValues[291]
		}
		if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != scm.LocNone {
			d292 = ps.OverlayValues[292]
		}
		if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
			d295 = ps.OverlayValues[295]
		}
		if len(ps.OverlayValues) > 415 && ps.OverlayValues[415].Loc != scm.LocNone {
			d415 = ps.OverlayValues[415]
		}
		if len(ps.OverlayValues) > 416 && ps.OverlayValues[416].Loc != scm.LocNone {
			d416 = ps.OverlayValues[416]
		}
		if len(ps.OverlayValues) > 417 && ps.OverlayValues[417].Loc != scm.LocNone {
			d417 = ps.OverlayValues[417]
		}
		if len(ps.OverlayValues) > 418 && ps.OverlayValues[418].Loc != scm.LocNone {
			d418 = ps.OverlayValues[418]
		}
		if len(ps.OverlayValues) > 419 && ps.OverlayValues[419].Loc != scm.LocNone {
			d419 = ps.OverlayValues[419]
		}
		if len(ps.OverlayValues) > 421 && ps.OverlayValues[421].Loc != scm.LocNone {
			d421 = ps.OverlayValues[421]
		}
		if len(ps.OverlayValues) > 422 && ps.OverlayValues[422].Loc != scm.LocNone {
			d422 = ps.OverlayValues[422]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		d423 = d5
		_ = d423
		ctx.StabilizeDescForControlFlow(&d423)
		r108 := d5.Loc == scm.LocReg || d5.Loc == scm.LocRegPair || d5.Loc == scm.LocRegTriple
		r109 := d5.Reg
		if r108 {
			ctx.ProtectReg(r109)
		}
		r110 := d5.Loc == scm.LocRegPair || d5.Loc == scm.LocRegTriple
		r111 := d5.Reg2
		if r110 {
			ctx.ProtectReg(r111)
		}
		r112 := d5.Loc == scm.LocRegTriple
		r113 := d5.Reg3
		if r112 {
			ctx.ProtectReg(r113)
		}
		phiBase424 = ctx.AllocStack(int32(16))
		d425 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase424) + int32(0)}
		_ = d425
		lbl31 := ctx.ReserveLabel()
		bbpos_3_0 := int32(-1)
		_ = bbpos_3_0
		bbpos_3_1 := int32(-1)
		_ = bbpos_3_1
		bbpos_3_2 := int32(-1)
		_ = bbpos_3_2
		bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		d425 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d423)
		ctx.EnsureDesc(&d423)
		var d426 scm.JITValueDesc
		if d423.Loc == scm.LocImm {
			d426 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d423.Imm.Int()))))}
		} else {
			r114 := ctx.AllocReg()
			ctx.EmitMovRegReg(r114, d423.Reg)
			ctx.EmitShlRegImm8(r114, 32)
			ctx.EmitShrRegImm8(r114, 32)
			d426 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r114}
			ctx.BindReg(r114, &d426)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d427 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			r115 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r115, fieldAddr)
			d427 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r115}
			ctx.BindReg(r115, &d427)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			r116 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r116, thisptr.Reg, off)
			d427 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r116}
			ctx.BindReg(r116, &d427)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d427)
		ctx.EnsureDesc(&d427)
		var d428 scm.JITValueDesc
		if d427.Loc == scm.LocImm {
			d428 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d427.Imm.Int()))))}
		} else {
			r117 := ctx.AllocReg()
			ctx.EmitMovRegReg(r117, d427.Reg)
			ctx.EmitShlRegImm8(r117, 56)
			ctx.EmitShrRegImm8(r117, 56)
			d428 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
			ctx.BindReg(r117, &d428)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d426)
		ctx.EnsureDesc(&d428)
		ctx.EnsureDesc(&d426)
		ctx.ProtectReg(d426.Reg)
		ctx.EnsureDesc(&d428)
		ctx.UnprotectReg(d426.Reg)
		var d429 scm.JITValueDesc
		if d426.Loc == scm.LocImm && d428.Loc == scm.LocImm {
			d429 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d426.Imm.Int() * d428.Imm.Int())}
		} else if d426.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d428.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d426.Imm.Int()))
			ctx.EmitImulInt64(scratch, d428.Reg)
			d429 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d429)
		} else if d428.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d426.Reg)
			ctx.EmitMovRegReg(scratch, d426.Reg)
			if d428.Imm.Int() >= -2147483648 && d428.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d428.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d428.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d429 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d429)
		} else {
			r118 := ctx.AllocRegExcept(d426.Reg, d428.Reg)
			ctx.EmitMovRegReg(r118, d426.Reg)
			ctx.EmitImulInt64(r118, d428.Reg)
			d429 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r118}
			ctx.BindReg(r118, &d429)
		}
		if d429.Loc == scm.LocReg && d426.Loc == scm.LocReg && d429.Reg == d426.Reg {
			ctx.TransferReg(d426.Reg)
			d426.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d429)
		ctx.FreeDesc(&d426)
		ctx.FreeDesc(&d428)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d430 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
			r119 := ctx.AllocReg()
			r120 := ctx.AllocRegExcept(r119)
			r121 := ctx.AllocRegExcept(r119, r120)
			ctx.EmitMovRegMem64(r119, fieldAddr)
			ctx.EmitMovRegMem64(r120, fieldAddr+8)
			ctx.EmitMovRegMem64(r121, fieldAddr+16)
			d430 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r119, Reg2: r120, Reg3: r121}
			ctx.BindReg(r119, &d430)
			ctx.BindReg(r120, &d430)
			ctx.BindReg(r121, &d430)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
			r122 := ctx.AllocReg()
			r123 := ctx.AllocRegExcept(r122)
			r124 := ctx.AllocRegExcept(r122, r123)
			ctx.EmitMovRegMem(r122, thisptr.Reg, off)
			ctx.EmitMovRegMem(r123, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r124, thisptr.Reg, off+16)
			d430 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r122, Reg2: r123, Reg3: r124}
			ctx.BindReg(r122, &d430)
			ctx.BindReg(r123, &d430)
			ctx.BindReg(r124, &d430)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d429)
		var d431 scm.JITValueDesc
		if d429.Loc == scm.LocImm {
			d431 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d429.Imm.Int() / 64)}
		} else {
			r125 := ctx.AllocRegExcept(d429.Reg)
			ctx.EmitMovRegReg(r125, d429.Reg)
			ctx.EmitShrRegImm8(r125, 6)
			d431 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
			ctx.BindReg(r125, &d431)
		}
		if d431.Loc == scm.LocReg && d429.Loc == scm.LocReg && d431.Reg == d429.Reg {
			ctx.TransferReg(d429.Reg)
			d429.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d431)
		ctx.ReclaimUntrackedRegs()
		d433 = ctx.EmitSliceElementAddress(&d430, &d431, 8)
		ctx.EnsureDesc(&d433)
		ctx.EmitMovRegMem(d433.Reg, d433.Reg, 0)
		d432 = d433
		ctx.FreeDesc(&d431)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d429)
		var d434 scm.JITValueDesc
		if d429.Loc == scm.LocImm {
			d434 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d429.Imm.Int() % 64)}
		} else {
			r126 := ctx.AllocRegExcept(d429.Reg)
			ctx.EmitMovRegReg(r126, d429.Reg)
			ctx.EmitAndRegImm32(r126, 63)
			d434 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r126}
			ctx.BindReg(r126, &d434)
		}
		if d434.Loc == scm.LocReg && d429.Loc == scm.LocReg && d434.Reg == d429.Reg {
			ctx.TransferReg(d429.Reg)
			d429.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d432)
		ctx.EnsureDesc(&d434)
		var d435 scm.JITValueDesc
		if d432.Loc == scm.LocImm && d434.Loc == scm.LocImm {
			d435 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d432.Imm.Int()) << uint64(d434.Imm.Int())))}
		} else if d434.Loc == scm.LocImm {
			r127 := ctx.AllocRegExcept(d432.Reg)
			ctx.EmitMovRegReg(r127, d432.Reg)
			ctx.EmitShlRegImm8(r127, uint8(d434.Imm.Int()))
			d435 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
			ctx.BindReg(r127, &d435)
		} else {
			{
				shiftSrc := d432.Reg
				r128 := ctx.AllocRegExcept(d432.Reg)
				ctx.EmitMovRegReg(r128, d432.Reg)
				shiftSrc = r128
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d434.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d434.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d434.Reg)
				}
				ctx.EmitShlRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d435 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d435)
			}
		}
		if d435.Loc == scm.LocReg && d432.Loc == scm.LocReg && d435.Reg == d432.Reg {
			ctx.TransferReg(d432.Reg)
			d432.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d435)
		ctx.EmitStoreToStack(d435, int32(phiBase424)+int32(0))
		ctx.StabilizeDescForControlFlow(&d435)
		ctx.FreeDesc(&d432)
		ctx.FreeDesc(&d434)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d429)
		var d436 scm.JITValueDesc
		if d429.Loc == scm.LocImm {
			d436 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d429.Imm.Int() % 64)}
		} else {
			r129 := ctx.AllocRegExcept(d429.Reg)
			ctx.EmitMovRegReg(r129, d429.Reg)
			ctx.EmitAndRegImm32(r129, 63)
			d436 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
			ctx.BindReg(r129, &d436)
		}
		if d436.Loc == scm.LocReg && d429.Loc == scm.LocReg && d436.Reg == d429.Reg {
			ctx.TransferReg(d429.Reg)
			d429.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d427)
		ctx.EnsureDesc(&d427)
		var d437 scm.JITValueDesc
		if d427.Loc == scm.LocImm {
			d437 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d427.Imm.Int()))))}
		} else {
			r130 := ctx.AllocReg()
			ctx.EmitMovRegReg(r130, d427.Reg)
			ctx.EmitShlRegImm8(r130, 56)
			ctx.EmitShrRegImm8(r130, 56)
			d437 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
			ctx.BindReg(r130, &d437)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d436)
		ctx.EnsureDesc(&d437)
		ctx.EnsureDesc(&d436)
		ctx.ProtectReg(d436.Reg)
		ctx.EnsureDesc(&d437)
		ctx.UnprotectReg(d436.Reg)
		var d438 scm.JITValueDesc
		if d436.Loc == scm.LocImm && d437.Loc == scm.LocImm {
			d438 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d436.Imm.Int() + d437.Imm.Int())}
		} else if d437.Loc == scm.LocImm && d437.Imm.Int() == 0 {
			r131 := ctx.AllocRegExcept(d436.Reg)
			ctx.EmitMovRegReg(r131, d436.Reg)
			d438 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r131}
			ctx.BindReg(r131, &d438)
		} else if d436.Loc == scm.LocImm && d436.Imm.Int() == 0 {
			d438 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d437.Reg}
			ctx.BindReg(d437.Reg, &d438)
		} else if d436.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d437.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d436.Imm.Int()))
			ctx.EmitAddInt64(scratch, d437.Reg)
			d438 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d438)
		} else if d437.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d436.Reg)
			ctx.EmitMovRegReg(scratch, d436.Reg)
			if d437.Imm.Int() >= -2147483648 && d437.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d437.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d437.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d438 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d438)
		} else {
			r132 := ctx.AllocRegExcept(d436.Reg, d437.Reg)
			ctx.EmitMovRegReg(r132, d436.Reg)
			ctx.EmitAddInt64(r132, d437.Reg)
			d438 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
			ctx.BindReg(r132, &d438)
		}
		if d438.Loc == scm.LocReg && d436.Loc == scm.LocReg && d438.Reg == d436.Reg {
			ctx.TransferReg(d436.Reg)
			d436.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d436)
		ctx.FreeDesc(&d437)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d438)
		var d439 scm.JITValueDesc
		if d438.Loc == scm.LocImm {
			d439 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d438.Imm.Int()) > uint64(0x40))}
		} else {
			r133 := ctx.AllocRegExcept(d438.Reg)
			ctx.EmitCmpRegImm32(d438.Reg, 64)
			ctx.EmitSetcc(r133, scm.CondUnsignedAbove)
			d439 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r133}
			ctx.BindReg(r133, &d439)
		}
		ctx.FreeDesc(&d438)
		ctx.ReclaimUntrackedRegs()
		d440 = d439
		ctx.EnsureDesc(&d440)
		if d440.Loc != scm.LocImm && d440.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		lbl32 := ctx.ReserveLabel()
		lbl33 := ctx.ReserveLabel()
		lbl34 := ctx.ReserveLabel()
		lbl35 := ctx.ReserveLabel()
		if d440.Loc == scm.LocImm {
			if d440.Imm.Bool() {
				ctx.MarkLabel(lbl34)
				ctx.EmitJmp(lbl32)
			} else {
				ctx.MarkLabel(lbl35)
				ctx.EmitJmp(lbl33)
			}
		} else {
			ctx.EmitCmpRegImm32(d440.Reg, 0)
			ctx.EmitJump(scm.CondNotEqual, lbl34)
			ctx.EmitJmp(lbl35)
			ctx.MarkLabel(lbl34)
			ctx.EmitJmp(lbl32)
			ctx.MarkLabel(lbl35)
			ctx.EmitJmp(lbl33)
		}
		ctx.FreeDesc(&d439)
		bbpos_3_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl33)
		ctx.ResolveFixups()
		d425 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d427)
		ctx.EnsureDesc(&d427)
		var d441 scm.JITValueDesc
		if d427.Loc == scm.LocImm {
			d441 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d427.Imm.Int()))))}
		} else {
			r134 := ctx.AllocReg()
			ctx.EmitMovRegReg(r134, d427.Reg)
			ctx.EmitShlRegImm8(r134, 56)
			ctx.EmitShrRegImm8(r134, 56)
			d441 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r134}
			ctx.BindReg(r134, &d441)
		}
		ctx.ReclaimUntrackedRegs()
		d442 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d441)
		ctx.EnsureDesc(&d442)
		ctx.ProtectReg(d442.Reg)
		ctx.EnsureDesc(&d441)
		ctx.UnprotectReg(d442.Reg)
		var d443 scm.JITValueDesc
		if d442.Loc == scm.LocImm && d441.Loc == scm.LocImm {
			d443 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d442.Imm.Int() - d441.Imm.Int())}
		} else if d441.Loc == scm.LocImm && d441.Imm.Int() == 0 {
			r135 := ctx.AllocRegExcept(d442.Reg)
			ctx.EmitMovRegReg(r135, d442.Reg)
			d443 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r135}
			ctx.BindReg(r135, &d443)
		} else if d442.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d441.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d442.Imm.Int()))
			ctx.EmitSubInt64(scratch, d441.Reg)
			d443 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d443)
		} else if d441.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d442.Reg)
			ctx.EmitMovRegReg(scratch, d442.Reg)
			if d441.Imm.Int() >= -2147483648 && d441.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d441.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d441.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d443 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d443)
		} else {
			r136 := ctx.AllocRegExcept(d442.Reg, d441.Reg)
			ctx.EmitMovRegReg(r136, d442.Reg)
			ctx.EmitSubInt64(r136, d441.Reg)
			d443 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r136}
			ctx.BindReg(r136, &d443)
		}
		if d443.Loc == scm.LocReg && d442.Loc == scm.LocReg && d443.Reg == d442.Reg {
			ctx.TransferReg(d442.Reg)
			d442.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d441)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d425)
		ctx.EnsureDesc(&d443)
		var d444 scm.JITValueDesc
		if d425.Loc == scm.LocImm && d443.Loc == scm.LocImm {
			d444 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d425.Imm.Int()) >> uint64(d443.Imm.Int())))}
		} else if d443.Loc == scm.LocImm {
			r137 := ctx.AllocRegExcept(d425.Reg)
			ctx.EmitMovRegReg(r137, d425.Reg)
			ctx.EmitShrRegImm8(r137, uint8(d443.Imm.Int()))
			d444 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r137}
			ctx.BindReg(r137, &d444)
		} else {
			{
				shiftSrc := d425.Reg
				r138 := ctx.AllocRegExcept(d425.Reg)
				ctx.EmitMovRegReg(r138, d425.Reg)
				shiftSrc = r138
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d443.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d443.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d443.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d444 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d444)
			}
		}
		if d444.Loc == scm.LocReg && d425.Loc == scm.LocReg && d444.Reg == d425.Reg {
			ctx.TransferReg(d425.Reg)
			d425.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d425)
		ctx.FreeDesc(&d443)
		ctx.ReclaimUntrackedRegs()
		r139 := ctx.AllocReg()
		ctx.EnsureDesc(&d444)
		ctx.EnsureDesc(&d444)
		if d444.Loc == scm.LocRegPair {
			panic("jit: scalar inline return has scm.LocRegPair")
		} else {
			ctx.EmitMovToReg(r139, d444)
		}
		ctx.EmitJmp(lbl31)
		bbpos_3_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl32)
		ctx.ResolveFixups()
		d425 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d429)
		var d445 scm.JITValueDesc
		if d429.Loc == scm.LocImm {
			d445 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d429.Imm.Int() / 64)}
		} else {
			r140 := ctx.AllocRegExcept(d429.Reg)
			ctx.EmitMovRegReg(r140, d429.Reg)
			ctx.EmitShrRegImm8(r140, 6)
			d445 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r140}
			ctx.BindReg(r140, &d445)
		}
		if d445.Loc == scm.LocReg && d429.Loc == scm.LocReg && d445.Reg == d429.Reg {
			ctx.TransferReg(d429.Reg)
			d429.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d445)
		ctx.EnsureDesc(&d445)
		var d446 scm.JITValueDesc
		if d445.Loc == scm.LocImm {
			d446 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d445.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d445.Reg)
			ctx.EmitMovRegReg(scratch, d445.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d446 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d446)
		}
		if d446.Loc == scm.LocReg && d445.Loc == scm.LocReg && d446.Reg == d445.Reg {
			ctx.TransferReg(d445.Reg)
			d445.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d445)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d446)
		ctx.ReclaimUntrackedRegs()
		d448 = ctx.EmitSliceElementAddress(&d430, &d446, 8)
		ctx.EnsureDesc(&d448)
		ctx.EmitMovRegMem(d448.Reg, d448.Reg, 0)
		d447 = d448
		ctx.FreeDesc(&d446)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d429)
		var d449 scm.JITValueDesc
		if d429.Loc == scm.LocImm {
			d449 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d429.Imm.Int() % 64)}
		} else {
			r141 := ctx.AllocRegExcept(d429.Reg)
			ctx.EmitMovRegReg(r141, d429.Reg)
			ctx.EmitAndRegImm32(r141, 63)
			d449 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
			ctx.BindReg(r141, &d449)
		}
		if d449.Loc == scm.LocReg && d429.Loc == scm.LocReg && d449.Reg == d429.Reg {
			ctx.TransferReg(d429.Reg)
			d429.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d429)
		ctx.ReclaimUntrackedRegs()
		d450 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d449)
		ctx.EnsureDesc(&d450)
		ctx.ProtectReg(d450.Reg)
		ctx.EnsureDesc(&d449)
		ctx.UnprotectReg(d450.Reg)
		var d451 scm.JITValueDesc
		if d450.Loc == scm.LocImm && d449.Loc == scm.LocImm {
			d451 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d450.Imm.Int() - d449.Imm.Int())}
		} else if d449.Loc == scm.LocImm && d449.Imm.Int() == 0 {
			r142 := ctx.AllocRegExcept(d450.Reg)
			ctx.EmitMovRegReg(r142, d450.Reg)
			d451 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r142}
			ctx.BindReg(r142, &d451)
		} else if d450.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d449.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d450.Imm.Int()))
			ctx.EmitSubInt64(scratch, d449.Reg)
			d451 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d451)
		} else if d449.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d450.Reg)
			ctx.EmitMovRegReg(scratch, d450.Reg)
			if d449.Imm.Int() >= -2147483648 && d449.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d449.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d449.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d451 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d451)
		} else {
			r143 := ctx.AllocRegExcept(d450.Reg, d449.Reg)
			ctx.EmitMovRegReg(r143, d450.Reg)
			ctx.EmitSubInt64(r143, d449.Reg)
			d451 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r143}
			ctx.BindReg(r143, &d451)
		}
		if d451.Loc == scm.LocReg && d450.Loc == scm.LocReg && d451.Reg == d450.Reg {
			ctx.TransferReg(d450.Reg)
			d450.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d449)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d447)
		ctx.EnsureDesc(&d451)
		var d452 scm.JITValueDesc
		if d447.Loc == scm.LocImm && d451.Loc == scm.LocImm {
			d452 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d447.Imm.Int()) >> uint64(d451.Imm.Int())))}
		} else if d451.Loc == scm.LocImm {
			r144 := ctx.AllocRegExcept(d447.Reg)
			ctx.EmitMovRegReg(r144, d447.Reg)
			ctx.EmitShrRegImm8(r144, uint8(d451.Imm.Int()))
			d452 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
			ctx.BindReg(r144, &d452)
		} else {
			{
				shiftSrc := d447.Reg
				r145 := ctx.AllocRegExcept(d447.Reg)
				ctx.EmitMovRegReg(r145, d447.Reg)
				shiftSrc = r145
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d451.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d451.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d451.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d452 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d452)
			}
		}
		if d452.Loc == scm.LocReg && d447.Loc == scm.LocReg && d452.Reg == d447.Reg {
			ctx.TransferReg(d447.Reg)
			d447.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d447)
		ctx.FreeDesc(&d451)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d435)
		ctx.EnsureDesc(&d452)
		var d453 scm.JITValueDesc
		if d435.Loc == scm.LocImm && d452.Loc == scm.LocImm {
			d453 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d435.Imm.Int() | d452.Imm.Int())}
		} else if d435.Loc == scm.LocImm && d435.Imm.Int() == 0 {
			d453 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d452.Reg}
			ctx.BindReg(d452.Reg, &d453)
		} else if d452.Loc == scm.LocImm && d452.Imm.Int() == 0 {
			r146 := ctx.AllocRegExcept(d435.Reg)
			ctx.EmitMovRegReg(r146, d435.Reg)
			d453 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r146}
			ctx.BindReg(r146, &d453)
		} else if d435.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d452.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d435.Imm.Int()))
			ctx.EmitOrInt64(scratch, d452.Reg)
			d453 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d453)
		} else if d452.Loc == scm.LocImm {
			r147 := ctx.AllocRegExcept(d435.Reg)
			ctx.EmitMovRegReg(r147, d435.Reg)
			if d452.Imm.Int() >= -2147483648 && d452.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r147, int32(d452.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d452.Imm.Int()))
				ctx.EmitOrInt64(r147, scm.RegR11)
			}
			d453 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r147}
			ctx.BindReg(r147, &d453)
		} else {
			r148 := ctx.AllocRegExcept(d435.Reg, d452.Reg)
			ctx.EmitMovRegReg(r148, d435.Reg)
			ctx.EmitOrInt64(r148, d452.Reg)
			d453 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r148}
			ctx.BindReg(r148, &d453)
		}
		if d453.Loc == scm.LocReg && d435.Loc == scm.LocReg && d453.Reg == d435.Reg {
			ctx.TransferReg(d435.Reg)
			d435.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d453)
		ctx.EmitStoreToStack(d453, int32(phiBase424)+int32(0))
		ctx.StabilizeDescForControlFlow(&d453)
		ctx.FreeDesc(&d452)
		ctx.ReclaimUntrackedRegs()
		ctx.EmitJmp(lbl33)
		ctx.MarkLabel(lbl31)
		d454 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r139}
		ctx.BindReg(r139, &d454)
		ctx.BindReg(r139, &d454)
		if r108 {
			ctx.UnprotectReg(r109)
		}
		if r110 {
			ctx.UnprotectReg(r111)
		}
		if r112 {
			ctx.UnprotectReg(r113)
		}
		ctx.EnsureDesc(&d454)
		ctx.EnsureDesc(&d454)
		var d455 scm.JITValueDesc
		if d454.Loc == scm.LocImm {
			d455 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d454.Imm.Int()))))}
		} else {
			r149 := ctx.AllocReg()
			ctx.EmitMovRegReg(r149, d454.Reg)
			d455 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r149}
			ctx.BindReg(r149, &d455)
		}
		ctx.FreeDesc(&d454)
		ctx.EnsureDesc(&d455)
		ctx.EnsureDesc(&d55)
		ctx.EnsureDesc(&d455)
		ctx.ProtectReg(d455.Reg)
		ctx.EnsureDesc(&d55)
		ctx.UnprotectReg(d455.Reg)
		var d456 scm.JITValueDesc
		if d455.Loc == scm.LocImm && d55.Loc == scm.LocImm {
			d456 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d455.Imm.Int() + d55.Imm.Int())}
		} else if d55.Loc == scm.LocImm && d55.Imm.Int() == 0 {
			r150 := ctx.AllocRegExcept(d455.Reg)
			ctx.EmitMovRegReg(r150, d455.Reg)
			d456 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r150}
			ctx.BindReg(r150, &d456)
		} else if d455.Loc == scm.LocImm && d455.Imm.Int() == 0 {
			d456 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d55.Reg}
			ctx.BindReg(d55.Reg, &d456)
		} else if d455.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d55.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d455.Imm.Int()))
			ctx.EmitAddInt64(scratch, d55.Reg)
			d456 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d456)
		} else if d55.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d455.Reg)
			ctx.EmitMovRegReg(scratch, d455.Reg)
			if d55.Imm.Int() >= -2147483648 && d55.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d55.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d55.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d456 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d456)
		} else {
			r151 := ctx.AllocRegExcept(d455.Reg, d55.Reg)
			ctx.EmitMovRegReg(r151, d455.Reg)
			ctx.EmitAddInt64(r151, d55.Reg)
			d456 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
			ctx.BindReg(r151, &d456)
		}
		if d456.Loc == scm.LocReg && d455.Loc == scm.LocReg && d456.Reg == d455.Reg {
			ctx.TransferReg(d455.Reg)
			d455.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d455)
		ctx.EnsureDesc(&d456)
		ctx.EnsureDesc(&d456)
		var d457 scm.JITValueDesc
		if d456.Loc == scm.LocImm {
			d457 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d456.Imm.Int()))))}
		} else {
			r152 := ctx.AllocReg()
			ctx.EmitMovRegReg(r152, d456.Reg)
			ctx.EmitShlRegImm8(r152, 32)
			ctx.EmitShrRegImm8(r152, 32)
			d457 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r152}
			ctx.BindReg(r152, &d457)
		}
		ctx.FreeDesc(&d456)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d457)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d457)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d457)
		var d458 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm && d457.Loc == scm.LocImm {
			d458 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d457.Imm.Int()))}
		} else if d457.Loc == scm.LocImm {
			r153 := ctx.AllocRegExcept(idxInt.Reg)
			if d457.Imm.Int() >= -2147483648 && d457.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(idxInt.Reg, int32(d457.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d457.Imm.Int()))
				ctx.EmitCmpInt64(idxInt.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r153, scm.CondUnsignedBelow)
			d458 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r153}
			ctx.BindReg(r153, &d458)
		} else if idxInt.Loc == scm.LocImm {
			r154 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d457.Reg)
			ctx.EmitSetcc(r154, scm.CondUnsignedBelow)
			d458 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r154}
			ctx.BindReg(r154, &d458)
		} else {
			r155 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitCmpInt64(idxInt.Reg, d457.Reg)
			ctx.EmitSetcc(r155, scm.CondUnsignedBelow)
			d458 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r155}
			ctx.BindReg(r155, &d458)
		}
		ctx.FreeDesc(&d457)
		d459 = d458
		ctx.EnsureDesc(&d459)
		if d459.Loc != scm.LocImm && d459.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d459.Loc == scm.LocImm {
			if d459.Imm.Bool() {
				if ps.General {
				}
				ps460 := scm.PhiState{General: ps.General}
				ps460.OverlayValues = make([]scm.JITValueDesc, 460)
				ps460.OverlayValues[1] = d1
				ps460.OverlayValues[2] = d2
				ps460.OverlayValues[3] = d3
				ps460.OverlayValues[4] = d4
				ps460.OverlayValues[5] = d5
				ps460.OverlayValues[6] = d6
				ps460.OverlayValues[7] = d7
				ps460.OverlayValues[8] = d8
				ps460.OverlayValues[9] = d9
				ps460.OverlayValues[10] = d10
				ps460.OverlayValues[11] = d11
				ps460.OverlayValues[12] = d12
				ps460.OverlayValues[13] = d13
				ps460.OverlayValues[14] = d14
				ps460.OverlayValues[15] = d15
				ps460.OverlayValues[17] = d17
				ps460.OverlayValues[18] = d18
				ps460.OverlayValues[19] = d19
				ps460.OverlayValues[20] = d20
				ps460.OverlayValues[21] = d21
				ps460.OverlayValues[22] = d22
				ps460.OverlayValues[24] = d24
				ps460.OverlayValues[25] = d25
				ps460.OverlayValues[26] = d26
				ps460.OverlayValues[27] = d27
				ps460.OverlayValues[28] = d28
				ps460.OverlayValues[29] = d29
				ps460.OverlayValues[30] = d30
				ps460.OverlayValues[31] = d31
				ps460.OverlayValues[32] = d32
				ps460.OverlayValues[33] = d33
				ps460.OverlayValues[34] = d34
				ps460.OverlayValues[35] = d35
				ps460.OverlayValues[36] = d36
				ps460.OverlayValues[37] = d37
				ps460.OverlayValues[38] = d38
				ps460.OverlayValues[39] = d39
				ps460.OverlayValues[40] = d40
				ps460.OverlayValues[41] = d41
				ps460.OverlayValues[42] = d42
				ps460.OverlayValues[43] = d43
				ps460.OverlayValues[44] = d44
				ps460.OverlayValues[45] = d45
				ps460.OverlayValues[46] = d46
				ps460.OverlayValues[47] = d47
				ps460.OverlayValues[48] = d48
				ps460.OverlayValues[49] = d49
				ps460.OverlayValues[50] = d50
				ps460.OverlayValues[51] = d51
				ps460.OverlayValues[52] = d52
				ps460.OverlayValues[53] = d53
				ps460.OverlayValues[54] = d54
				ps460.OverlayValues[55] = d55
				ps460.OverlayValues[56] = d56
				ps460.OverlayValues[57] = d57
				ps460.OverlayValues[58] = d58
				ps460.OverlayValues[59] = d59
				ps460.OverlayValues[62] = d62
				ps460.OverlayValues[63] = d63
				ps460.OverlayValues[64] = d64
				ps460.OverlayValues[128] = d128
				ps460.OverlayValues[129] = d129
				ps460.OverlayValues[130] = d130
				ps460.OverlayValues[132] = d132
				ps460.OverlayValues[133] = d133
				ps460.OverlayValues[134] = d134
				ps460.OverlayValues[135] = d135
				ps460.OverlayValues[136] = d136
				ps460.OverlayValues[137] = d137
				ps460.OverlayValues[138] = d138
				ps460.OverlayValues[139] = d139
				ps460.OverlayValues[140] = d140
				ps460.OverlayValues[141] = d141
				ps460.OverlayValues[142] = d142
				ps460.OverlayValues[143] = d143
				ps460.OverlayValues[144] = d144
				ps460.OverlayValues[145] = d145
				ps460.OverlayValues[146] = d146
				ps460.OverlayValues[147] = d147
				ps460.OverlayValues[148] = d148
				ps460.OverlayValues[149] = d149
				ps460.OverlayValues[150] = d150
				ps460.OverlayValues[151] = d151
				ps460.OverlayValues[152] = d152
				ps460.OverlayValues[153] = d153
				ps460.OverlayValues[154] = d154
				ps460.OverlayValues[155] = d155
				ps460.OverlayValues[156] = d156
				ps460.OverlayValues[157] = d157
				ps460.OverlayValues[158] = d158
				ps460.OverlayValues[159] = d159
				ps460.OverlayValues[160] = d160
				ps460.OverlayValues[161] = d161
				ps460.OverlayValues[162] = d162
				ps460.OverlayValues[163] = d163
				ps460.OverlayValues[164] = d164
				ps460.OverlayValues[165] = d165
				ps460.OverlayValues[166] = d166
				ps460.OverlayValues[169] = d169
				ps460.OverlayValues[272] = d272
				ps460.OverlayValues[273] = d273
				ps460.OverlayValues[274] = d274
				ps460.OverlayValues[275] = d275
				ps460.OverlayValues[277] = d277
				ps460.OverlayValues[278] = d278
				ps460.OverlayValues[279] = d279
				ps460.OverlayValues[280] = d280
				ps460.OverlayValues[281] = d281
				ps460.OverlayValues[282] = d282
				ps460.OverlayValues[283] = d283
				ps460.OverlayValues[284] = d284
				ps460.OverlayValues[286] = d286
				ps460.OverlayValues[288] = d288
				ps460.OverlayValues[289] = d289
				ps460.OverlayValues[290] = d290
				ps460.OverlayValues[291] = d291
				ps460.OverlayValues[292] = d292
				ps460.OverlayValues[295] = d295
				ps460.OverlayValues[415] = d415
				ps460.OverlayValues[416] = d416
				ps460.OverlayValues[417] = d417
				ps460.OverlayValues[418] = d418
				ps460.OverlayValues[419] = d419
				ps460.OverlayValues[421] = d421
				ps460.OverlayValues[422] = d422
				ps460.OverlayValues[423] = d423
				ps460.OverlayValues[425] = d425
				ps460.OverlayValues[426] = d426
				ps460.OverlayValues[427] = d427
				ps460.OverlayValues[428] = d428
				ps460.OverlayValues[429] = d429
				ps460.OverlayValues[430] = d430
				ps460.OverlayValues[431] = d431
				ps460.OverlayValues[432] = d432
				ps460.OverlayValues[433] = d433
				ps460.OverlayValues[434] = d434
				ps460.OverlayValues[435] = d435
				ps460.OverlayValues[436] = d436
				ps460.OverlayValues[437] = d437
				ps460.OverlayValues[438] = d438
				ps460.OverlayValues[439] = d439
				ps460.OverlayValues[440] = d440
				ps460.OverlayValues[441] = d441
				ps460.OverlayValues[442] = d442
				ps460.OverlayValues[443] = d443
				ps460.OverlayValues[444] = d444
				ps460.OverlayValues[445] = d445
				ps460.OverlayValues[446] = d446
				ps460.OverlayValues[447] = d447
				ps460.OverlayValues[448] = d448
				ps460.OverlayValues[449] = d449
				ps460.OverlayValues[450] = d450
				ps460.OverlayValues[451] = d451
				ps460.OverlayValues[452] = d452
				ps460.OverlayValues[453] = d453
				ps460.OverlayValues[454] = d454
				ps460.OverlayValues[455] = d455
				ps460.OverlayValues[456] = d456
				ps460.OverlayValues[457] = d457
				ps460.OverlayValues[458] = d458
				ps460.OverlayValues[459] = d459
				return bbs[7].RenderPS(ps460)
			}
			if ps.General {
			}
			ps461 := scm.PhiState{General: ps.General}
			ps461.OverlayValues = make([]scm.JITValueDesc, 460)
			ps461.OverlayValues[1] = d1
			ps461.OverlayValues[2] = d2
			ps461.OverlayValues[3] = d3
			ps461.OverlayValues[4] = d4
			ps461.OverlayValues[5] = d5
			ps461.OverlayValues[6] = d6
			ps461.OverlayValues[7] = d7
			ps461.OverlayValues[8] = d8
			ps461.OverlayValues[9] = d9
			ps461.OverlayValues[10] = d10
			ps461.OverlayValues[11] = d11
			ps461.OverlayValues[12] = d12
			ps461.OverlayValues[13] = d13
			ps461.OverlayValues[14] = d14
			ps461.OverlayValues[15] = d15
			ps461.OverlayValues[17] = d17
			ps461.OverlayValues[18] = d18
			ps461.OverlayValues[19] = d19
			ps461.OverlayValues[20] = d20
			ps461.OverlayValues[21] = d21
			ps461.OverlayValues[22] = d22
			ps461.OverlayValues[24] = d24
			ps461.OverlayValues[25] = d25
			ps461.OverlayValues[26] = d26
			ps461.OverlayValues[27] = d27
			ps461.OverlayValues[28] = d28
			ps461.OverlayValues[29] = d29
			ps461.OverlayValues[30] = d30
			ps461.OverlayValues[31] = d31
			ps461.OverlayValues[32] = d32
			ps461.OverlayValues[33] = d33
			ps461.OverlayValues[34] = d34
			ps461.OverlayValues[35] = d35
			ps461.OverlayValues[36] = d36
			ps461.OverlayValues[37] = d37
			ps461.OverlayValues[38] = d38
			ps461.OverlayValues[39] = d39
			ps461.OverlayValues[40] = d40
			ps461.OverlayValues[41] = d41
			ps461.OverlayValues[42] = d42
			ps461.OverlayValues[43] = d43
			ps461.OverlayValues[44] = d44
			ps461.OverlayValues[45] = d45
			ps461.OverlayValues[46] = d46
			ps461.OverlayValues[47] = d47
			ps461.OverlayValues[48] = d48
			ps461.OverlayValues[49] = d49
			ps461.OverlayValues[50] = d50
			ps461.OverlayValues[51] = d51
			ps461.OverlayValues[52] = d52
			ps461.OverlayValues[53] = d53
			ps461.OverlayValues[54] = d54
			ps461.OverlayValues[55] = d55
			ps461.OverlayValues[56] = d56
			ps461.OverlayValues[57] = d57
			ps461.OverlayValues[58] = d58
			ps461.OverlayValues[59] = d59
			ps461.OverlayValues[62] = d62
			ps461.OverlayValues[63] = d63
			ps461.OverlayValues[64] = d64
			ps461.OverlayValues[128] = d128
			ps461.OverlayValues[129] = d129
			ps461.OverlayValues[130] = d130
			ps461.OverlayValues[132] = d132
			ps461.OverlayValues[133] = d133
			ps461.OverlayValues[134] = d134
			ps461.OverlayValues[135] = d135
			ps461.OverlayValues[136] = d136
			ps461.OverlayValues[137] = d137
			ps461.OverlayValues[138] = d138
			ps461.OverlayValues[139] = d139
			ps461.OverlayValues[140] = d140
			ps461.OverlayValues[141] = d141
			ps461.OverlayValues[142] = d142
			ps461.OverlayValues[143] = d143
			ps461.OverlayValues[144] = d144
			ps461.OverlayValues[145] = d145
			ps461.OverlayValues[146] = d146
			ps461.OverlayValues[147] = d147
			ps461.OverlayValues[148] = d148
			ps461.OverlayValues[149] = d149
			ps461.OverlayValues[150] = d150
			ps461.OverlayValues[151] = d151
			ps461.OverlayValues[152] = d152
			ps461.OverlayValues[153] = d153
			ps461.OverlayValues[154] = d154
			ps461.OverlayValues[155] = d155
			ps461.OverlayValues[156] = d156
			ps461.OverlayValues[157] = d157
			ps461.OverlayValues[158] = d158
			ps461.OverlayValues[159] = d159
			ps461.OverlayValues[160] = d160
			ps461.OverlayValues[161] = d161
			ps461.OverlayValues[162] = d162
			ps461.OverlayValues[163] = d163
			ps461.OverlayValues[164] = d164
			ps461.OverlayValues[165] = d165
			ps461.OverlayValues[166] = d166
			ps461.OverlayValues[169] = d169
			ps461.OverlayValues[272] = d272
			ps461.OverlayValues[273] = d273
			ps461.OverlayValues[274] = d274
			ps461.OverlayValues[275] = d275
			ps461.OverlayValues[277] = d277
			ps461.OverlayValues[278] = d278
			ps461.OverlayValues[279] = d279
			ps461.OverlayValues[280] = d280
			ps461.OverlayValues[281] = d281
			ps461.OverlayValues[282] = d282
			ps461.OverlayValues[283] = d283
			ps461.OverlayValues[284] = d284
			ps461.OverlayValues[286] = d286
			ps461.OverlayValues[288] = d288
			ps461.OverlayValues[289] = d289
			ps461.OverlayValues[290] = d290
			ps461.OverlayValues[291] = d291
			ps461.OverlayValues[292] = d292
			ps461.OverlayValues[295] = d295
			ps461.OverlayValues[415] = d415
			ps461.OverlayValues[416] = d416
			ps461.OverlayValues[417] = d417
			ps461.OverlayValues[418] = d418
			ps461.OverlayValues[419] = d419
			ps461.OverlayValues[421] = d421
			ps461.OverlayValues[422] = d422
			ps461.OverlayValues[423] = d423
			ps461.OverlayValues[425] = d425
			ps461.OverlayValues[426] = d426
			ps461.OverlayValues[427] = d427
			ps461.OverlayValues[428] = d428
			ps461.OverlayValues[429] = d429
			ps461.OverlayValues[430] = d430
			ps461.OverlayValues[431] = d431
			ps461.OverlayValues[432] = d432
			ps461.OverlayValues[433] = d433
			ps461.OverlayValues[434] = d434
			ps461.OverlayValues[435] = d435
			ps461.OverlayValues[436] = d436
			ps461.OverlayValues[437] = d437
			ps461.OverlayValues[438] = d438
			ps461.OverlayValues[439] = d439
			ps461.OverlayValues[440] = d440
			ps461.OverlayValues[441] = d441
			ps461.OverlayValues[442] = d442
			ps461.OverlayValues[443] = d443
			ps461.OverlayValues[444] = d444
			ps461.OverlayValues[445] = d445
			ps461.OverlayValues[446] = d446
			ps461.OverlayValues[447] = d447
			ps461.OverlayValues[448] = d448
			ps461.OverlayValues[449] = d449
			ps461.OverlayValues[450] = d450
			ps461.OverlayValues[451] = d451
			ps461.OverlayValues[452] = d452
			ps461.OverlayValues[453] = d453
			ps461.OverlayValues[454] = d454
			ps461.OverlayValues[455] = d455
			ps461.OverlayValues[456] = d456
			ps461.OverlayValues[457] = d457
			ps461.OverlayValues[458] = d458
			ps461.OverlayValues[459] = d459
			return bbs[9].RenderPS(ps461)
		}
		if !ps.General {
			ps.General = true
			return bbs[6].RenderPS(ps)
		}
		lbl36 := ctx.ReserveLabel()
		lbl37 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d459.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl36)
		ctx.EmitJmp(lbl37)
		ctx.MarkLabel(lbl36)
		ctx.EmitJmp(lbl8)
		ctx.MarkLabel(lbl37)
		ctx.EmitJmp(lbl10)
		ps462 := scm.PhiState{General: true}
		ps462.OverlayValues = make([]scm.JITValueDesc, 460)
		ps462.OverlayValues[1] = d1
		ps462.OverlayValues[2] = d2
		ps462.OverlayValues[3] = d3
		ps462.OverlayValues[4] = d4
		ps462.OverlayValues[5] = d5
		ps462.OverlayValues[6] = d6
		ps462.OverlayValues[7] = d7
		ps462.OverlayValues[8] = d8
		ps462.OverlayValues[9] = d9
		ps462.OverlayValues[10] = d10
		ps462.OverlayValues[11] = d11
		ps462.OverlayValues[12] = d12
		ps462.OverlayValues[13] = d13
		ps462.OverlayValues[14] = d14
		ps462.OverlayValues[15] = d15
		ps462.OverlayValues[17] = d17
		ps462.OverlayValues[18] = d18
		ps462.OverlayValues[19] = d19
		ps462.OverlayValues[20] = d20
		ps462.OverlayValues[21] = d21
		ps462.OverlayValues[22] = d22
		ps462.OverlayValues[24] = d24
		ps462.OverlayValues[25] = d25
		ps462.OverlayValues[26] = d26
		ps462.OverlayValues[27] = d27
		ps462.OverlayValues[28] = d28
		ps462.OverlayValues[29] = d29
		ps462.OverlayValues[30] = d30
		ps462.OverlayValues[31] = d31
		ps462.OverlayValues[32] = d32
		ps462.OverlayValues[33] = d33
		ps462.OverlayValues[34] = d34
		ps462.OverlayValues[35] = d35
		ps462.OverlayValues[36] = d36
		ps462.OverlayValues[37] = d37
		ps462.OverlayValues[38] = d38
		ps462.OverlayValues[39] = d39
		ps462.OverlayValues[40] = d40
		ps462.OverlayValues[41] = d41
		ps462.OverlayValues[42] = d42
		ps462.OverlayValues[43] = d43
		ps462.OverlayValues[44] = d44
		ps462.OverlayValues[45] = d45
		ps462.OverlayValues[46] = d46
		ps462.OverlayValues[47] = d47
		ps462.OverlayValues[48] = d48
		ps462.OverlayValues[49] = d49
		ps462.OverlayValues[50] = d50
		ps462.OverlayValues[51] = d51
		ps462.OverlayValues[52] = d52
		ps462.OverlayValues[53] = d53
		ps462.OverlayValues[54] = d54
		ps462.OverlayValues[55] = d55
		ps462.OverlayValues[56] = d56
		ps462.OverlayValues[57] = d57
		ps462.OverlayValues[58] = d58
		ps462.OverlayValues[59] = d59
		ps462.OverlayValues[62] = d62
		ps462.OverlayValues[63] = d63
		ps462.OverlayValues[64] = d64
		ps462.OverlayValues[128] = d128
		ps462.OverlayValues[129] = d129
		ps462.OverlayValues[130] = d130
		ps462.OverlayValues[132] = d132
		ps462.OverlayValues[133] = d133
		ps462.OverlayValues[134] = d134
		ps462.OverlayValues[135] = d135
		ps462.OverlayValues[136] = d136
		ps462.OverlayValues[137] = d137
		ps462.OverlayValues[138] = d138
		ps462.OverlayValues[139] = d139
		ps462.OverlayValues[140] = d140
		ps462.OverlayValues[141] = d141
		ps462.OverlayValues[142] = d142
		ps462.OverlayValues[143] = d143
		ps462.OverlayValues[144] = d144
		ps462.OverlayValues[145] = d145
		ps462.OverlayValues[146] = d146
		ps462.OverlayValues[147] = d147
		ps462.OverlayValues[148] = d148
		ps462.OverlayValues[149] = d149
		ps462.OverlayValues[150] = d150
		ps462.OverlayValues[151] = d151
		ps462.OverlayValues[152] = d152
		ps462.OverlayValues[153] = d153
		ps462.OverlayValues[154] = d154
		ps462.OverlayValues[155] = d155
		ps462.OverlayValues[156] = d156
		ps462.OverlayValues[157] = d157
		ps462.OverlayValues[158] = d158
		ps462.OverlayValues[159] = d159
		ps462.OverlayValues[160] = d160
		ps462.OverlayValues[161] = d161
		ps462.OverlayValues[162] = d162
		ps462.OverlayValues[163] = d163
		ps462.OverlayValues[164] = d164
		ps462.OverlayValues[165] = d165
		ps462.OverlayValues[166] = d166
		ps462.OverlayValues[169] = d169
		ps462.OverlayValues[272] = d272
		ps462.OverlayValues[273] = d273
		ps462.OverlayValues[274] = d274
		ps462.OverlayValues[275] = d275
		ps462.OverlayValues[277] = d277
		ps462.OverlayValues[278] = d278
		ps462.OverlayValues[279] = d279
		ps462.OverlayValues[280] = d280
		ps462.OverlayValues[281] = d281
		ps462.OverlayValues[282] = d282
		ps462.OverlayValues[283] = d283
		ps462.OverlayValues[284] = d284
		ps462.OverlayValues[286] = d286
		ps462.OverlayValues[288] = d288
		ps462.OverlayValues[289] = d289
		ps462.OverlayValues[290] = d290
		ps462.OverlayValues[291] = d291
		ps462.OverlayValues[292] = d292
		ps462.OverlayValues[295] = d295
		ps462.OverlayValues[415] = d415
		ps462.OverlayValues[416] = d416
		ps462.OverlayValues[417] = d417
		ps462.OverlayValues[418] = d418
		ps462.OverlayValues[419] = d419
		ps462.OverlayValues[421] = d421
		ps462.OverlayValues[422] = d422
		ps462.OverlayValues[423] = d423
		ps462.OverlayValues[425] = d425
		ps462.OverlayValues[426] = d426
		ps462.OverlayValues[427] = d427
		ps462.OverlayValues[428] = d428
		ps462.OverlayValues[429] = d429
		ps462.OverlayValues[430] = d430
		ps462.OverlayValues[431] = d431
		ps462.OverlayValues[432] = d432
		ps462.OverlayValues[433] = d433
		ps462.OverlayValues[434] = d434
		ps462.OverlayValues[435] = d435
		ps462.OverlayValues[436] = d436
		ps462.OverlayValues[437] = d437
		ps462.OverlayValues[438] = d438
		ps462.OverlayValues[439] = d439
		ps462.OverlayValues[440] = d440
		ps462.OverlayValues[441] = d441
		ps462.OverlayValues[442] = d442
		ps462.OverlayValues[443] = d443
		ps462.OverlayValues[444] = d444
		ps462.OverlayValues[445] = d445
		ps462.OverlayValues[446] = d446
		ps462.OverlayValues[447] = d447
		ps462.OverlayValues[448] = d448
		ps462.OverlayValues[449] = d449
		ps462.OverlayValues[450] = d450
		ps462.OverlayValues[451] = d451
		ps462.OverlayValues[452] = d452
		ps462.OverlayValues[453] = d453
		ps462.OverlayValues[454] = d454
		ps462.OverlayValues[455] = d455
		ps462.OverlayValues[456] = d456
		ps462.OverlayValues[457] = d457
		ps462.OverlayValues[458] = d458
		ps462.OverlayValues[459] = d459
		ps463 := scm.PhiState{General: true}
		ps463.OverlayValues = make([]scm.JITValueDesc, 460)
		ps463.OverlayValues[1] = d1
		ps463.OverlayValues[2] = d2
		ps463.OverlayValues[3] = d3
		ps463.OverlayValues[4] = d4
		ps463.OverlayValues[5] = d5
		ps463.OverlayValues[6] = d6
		ps463.OverlayValues[7] = d7
		ps463.OverlayValues[8] = d8
		ps463.OverlayValues[9] = d9
		ps463.OverlayValues[10] = d10
		ps463.OverlayValues[11] = d11
		ps463.OverlayValues[12] = d12
		ps463.OverlayValues[13] = d13
		ps463.OverlayValues[14] = d14
		ps463.OverlayValues[15] = d15
		ps463.OverlayValues[17] = d17
		ps463.OverlayValues[18] = d18
		ps463.OverlayValues[19] = d19
		ps463.OverlayValues[20] = d20
		ps463.OverlayValues[21] = d21
		ps463.OverlayValues[22] = d22
		ps463.OverlayValues[24] = d24
		ps463.OverlayValues[25] = d25
		ps463.OverlayValues[26] = d26
		ps463.OverlayValues[27] = d27
		ps463.OverlayValues[28] = d28
		ps463.OverlayValues[29] = d29
		ps463.OverlayValues[30] = d30
		ps463.OverlayValues[31] = d31
		ps463.OverlayValues[32] = d32
		ps463.OverlayValues[33] = d33
		ps463.OverlayValues[34] = d34
		ps463.OverlayValues[35] = d35
		ps463.OverlayValues[36] = d36
		ps463.OverlayValues[37] = d37
		ps463.OverlayValues[38] = d38
		ps463.OverlayValues[39] = d39
		ps463.OverlayValues[40] = d40
		ps463.OverlayValues[41] = d41
		ps463.OverlayValues[42] = d42
		ps463.OverlayValues[43] = d43
		ps463.OverlayValues[44] = d44
		ps463.OverlayValues[45] = d45
		ps463.OverlayValues[46] = d46
		ps463.OverlayValues[47] = d47
		ps463.OverlayValues[48] = d48
		ps463.OverlayValues[49] = d49
		ps463.OverlayValues[50] = d50
		ps463.OverlayValues[51] = d51
		ps463.OverlayValues[52] = d52
		ps463.OverlayValues[53] = d53
		ps463.OverlayValues[54] = d54
		ps463.OverlayValues[55] = d55
		ps463.OverlayValues[56] = d56
		ps463.OverlayValues[57] = d57
		ps463.OverlayValues[58] = d58
		ps463.OverlayValues[59] = d59
		ps463.OverlayValues[62] = d62
		ps463.OverlayValues[63] = d63
		ps463.OverlayValues[64] = d64
		ps463.OverlayValues[128] = d128
		ps463.OverlayValues[129] = d129
		ps463.OverlayValues[130] = d130
		ps463.OverlayValues[132] = d132
		ps463.OverlayValues[133] = d133
		ps463.OverlayValues[134] = d134
		ps463.OverlayValues[135] = d135
		ps463.OverlayValues[136] = d136
		ps463.OverlayValues[137] = d137
		ps463.OverlayValues[138] = d138
		ps463.OverlayValues[139] = d139
		ps463.OverlayValues[140] = d140
		ps463.OverlayValues[141] = d141
		ps463.OverlayValues[142] = d142
		ps463.OverlayValues[143] = d143
		ps463.OverlayValues[144] = d144
		ps463.OverlayValues[145] = d145
		ps463.OverlayValues[146] = d146
		ps463.OverlayValues[147] = d147
		ps463.OverlayValues[148] = d148
		ps463.OverlayValues[149] = d149
		ps463.OverlayValues[150] = d150
		ps463.OverlayValues[151] = d151
		ps463.OverlayValues[152] = d152
		ps463.OverlayValues[153] = d153
		ps463.OverlayValues[154] = d154
		ps463.OverlayValues[155] = d155
		ps463.OverlayValues[156] = d156
		ps463.OverlayValues[157] = d157
		ps463.OverlayValues[158] = d158
		ps463.OverlayValues[159] = d159
		ps463.OverlayValues[160] = d160
		ps463.OverlayValues[161] = d161
		ps463.OverlayValues[162] = d162
		ps463.OverlayValues[163] = d163
		ps463.OverlayValues[164] = d164
		ps463.OverlayValues[165] = d165
		ps463.OverlayValues[166] = d166
		ps463.OverlayValues[169] = d169
		ps463.OverlayValues[272] = d272
		ps463.OverlayValues[273] = d273
		ps463.OverlayValues[274] = d274
		ps463.OverlayValues[275] = d275
		ps463.OverlayValues[277] = d277
		ps463.OverlayValues[278] = d278
		ps463.OverlayValues[279] = d279
		ps463.OverlayValues[280] = d280
		ps463.OverlayValues[281] = d281
		ps463.OverlayValues[282] = d282
		ps463.OverlayValues[283] = d283
		ps463.OverlayValues[284] = d284
		ps463.OverlayValues[286] = d286
		ps463.OverlayValues[288] = d288
		ps463.OverlayValues[289] = d289
		ps463.OverlayValues[290] = d290
		ps463.OverlayValues[291] = d291
		ps463.OverlayValues[292] = d292
		ps463.OverlayValues[295] = d295
		ps463.OverlayValues[415] = d415
		ps463.OverlayValues[416] = d416
		ps463.OverlayValues[417] = d417
		ps463.OverlayValues[418] = d418
		ps463.OverlayValues[419] = d419
		ps463.OverlayValues[421] = d421
		ps463.OverlayValues[422] = d422
		ps463.OverlayValues[423] = d423
		ps463.OverlayValues[425] = d425
		ps463.OverlayValues[426] = d426
		ps463.OverlayValues[427] = d427
		ps463.OverlayValues[428] = d428
		ps463.OverlayValues[429] = d429
		ps463.OverlayValues[430] = d430
		ps463.OverlayValues[431] = d431
		ps463.OverlayValues[432] = d432
		ps463.OverlayValues[433] = d433
		ps463.OverlayValues[434] = d434
		ps463.OverlayValues[435] = d435
		ps463.OverlayValues[436] = d436
		ps463.OverlayValues[437] = d437
		ps463.OverlayValues[438] = d438
		ps463.OverlayValues[439] = d439
		ps463.OverlayValues[440] = d440
		ps463.OverlayValues[441] = d441
		ps463.OverlayValues[442] = d442
		ps463.OverlayValues[443] = d443
		ps463.OverlayValues[444] = d444
		ps463.OverlayValues[445] = d445
		ps463.OverlayValues[446] = d446
		ps463.OverlayValues[447] = d447
		ps463.OverlayValues[448] = d448
		ps463.OverlayValues[449] = d449
		ps463.OverlayValues[450] = d450
		ps463.OverlayValues[451] = d451
		ps463.OverlayValues[452] = d452
		ps463.OverlayValues[453] = d453
		ps463.OverlayValues[454] = d454
		ps463.OverlayValues[455] = d455
		ps463.OverlayValues[456] = d456
		ps463.OverlayValues[457] = d457
		ps463.OverlayValues[458] = d458
		ps463.OverlayValues[459] = d459
		snap464 := d1
		snap465 := d2
		snap466 := d3
		snap467 := d4
		snap468 := d5
		snap469 := d6
		snap470 := d7
		snap471 := d8
		snap472 := d9
		snap473 := d10
		snap474 := d11
		snap475 := d12
		snap476 := d13
		snap477 := d14
		snap478 := d15
		snap479 := d17
		snap480 := d18
		snap481 := d19
		snap482 := d20
		snap483 := d21
		snap484 := d22
		snap485 := d24
		snap486 := d25
		snap487 := d26
		snap488 := d27
		snap489 := d28
		snap490 := d29
		snap491 := d30
		snap492 := d31
		snap493 := d32
		snap494 := d33
		snap495 := d34
		snap496 := d35
		snap497 := d36
		snap498 := d37
		snap499 := d38
		snap500 := d39
		snap501 := d40
		snap502 := d41
		snap503 := d42
		snap504 := d43
		snap505 := d44
		snap506 := d45
		snap507 := d46
		snap508 := d47
		snap509 := d48
		snap510 := d49
		snap511 := d50
		snap512 := d51
		snap513 := d52
		snap514 := d53
		snap515 := d54
		snap516 := d55
		snap517 := d56
		snap518 := d57
		snap519 := d58
		snap520 := d59
		snap521 := d62
		snap522 := d63
		snap523 := d64
		snap524 := d128
		snap525 := d129
		snap526 := d130
		snap527 := d132
		snap528 := d133
		snap529 := d134
		snap530 := d135
		snap531 := d136
		snap532 := d137
		snap533 := d138
		snap534 := d139
		snap535 := d140
		snap536 := d141
		snap537 := d142
		snap538 := d143
		snap539 := d144
		snap540 := d145
		snap541 := d146
		snap542 := d147
		snap543 := d148
		snap544 := d149
		snap545 := d150
		snap546 := d151
		snap547 := d152
		snap548 := d153
		snap549 := d154
		snap550 := d155
		snap551 := d156
		snap552 := d157
		snap553 := d158
		snap554 := d159
		snap555 := d160
		snap556 := d161
		snap557 := d162
		snap558 := d163
		snap559 := d164
		snap560 := d165
		snap561 := d166
		snap562 := d169
		snap563 := d272
		snap564 := d273
		snap565 := d274
		snap566 := d275
		snap567 := d277
		snap568 := d278
		snap569 := d279
		snap570 := d280
		snap571 := d281
		snap572 := d282
		snap573 := d283
		snap574 := d284
		snap575 := d286
		snap576 := d288
		snap577 := d289
		snap578 := d290
		snap579 := d291
		snap580 := d292
		snap581 := d295
		snap582 := d415
		snap583 := d416
		snap584 := d417
		snap585 := d418
		snap586 := d419
		snap587 := d421
		snap588 := d422
		snap589 := d423
		snap590 := d425
		snap591 := d426
		snap592 := d427
		snap593 := d428
		snap594 := d429
		snap595 := d430
		snap596 := d431
		snap597 := d432
		snap598 := d433
		snap599 := d434
		snap600 := d435
		snap601 := d436
		snap602 := d437
		snap603 := d438
		snap604 := d439
		snap605 := d440
		snap606 := d441
		snap607 := d442
		snap608 := d443
		snap609 := d444
		snap610 := d445
		snap611 := d446
		snap612 := d447
		snap613 := d448
		snap614 := d449
		snap615 := d450
		snap616 := d451
		snap617 := d452
		snap618 := d453
		snap619 := d454
		snap620 := d455
		snap621 := d456
		snap622 := d457
		snap623 := d458
		snap624 := d459
		alloc625 := ctx.SnapshotAllocState()
		if !bbs[9].Rendered {
			bbs[9].RenderPS(ps463)
		}
		ctx.RestoreAllocState(alloc625)
		d1 = snap464
		d2 = snap465
		d3 = snap466
		d4 = snap467
		d5 = snap468
		d6 = snap469
		d7 = snap470
		d8 = snap471
		d9 = snap472
		d10 = snap473
		d11 = snap474
		d12 = snap475
		d13 = snap476
		d14 = snap477
		d15 = snap478
		d17 = snap479
		d18 = snap480
		d19 = snap481
		d20 = snap482
		d21 = snap483
		d22 = snap484
		d24 = snap485
		d25 = snap486
		d26 = snap487
		d27 = snap488
		d28 = snap489
		d29 = snap490
		d30 = snap491
		d31 = snap492
		d32 = snap493
		d33 = snap494
		d34 = snap495
		d35 = snap496
		d36 = snap497
		d37 = snap498
		d38 = snap499
		d39 = snap500
		d40 = snap501
		d41 = snap502
		d42 = snap503
		d43 = snap504
		d44 = snap505
		d45 = snap506
		d46 = snap507
		d47 = snap508
		d48 = snap509
		d49 = snap510
		d50 = snap511
		d51 = snap512
		d52 = snap513
		d53 = snap514
		d54 = snap515
		d55 = snap516
		d56 = snap517
		d57 = snap518
		d58 = snap519
		d59 = snap520
		d62 = snap521
		d63 = snap522
		d64 = snap523
		d128 = snap524
		d129 = snap525
		d130 = snap526
		d132 = snap527
		d133 = snap528
		d134 = snap529
		d135 = snap530
		d136 = snap531
		d137 = snap532
		d138 = snap533
		d139 = snap534
		d140 = snap535
		d141 = snap536
		d142 = snap537
		d143 = snap538
		d144 = snap539
		d145 = snap540
		d146 = snap541
		d147 = snap542
		d148 = snap543
		d149 = snap544
		d150 = snap545
		d151 = snap546
		d152 = snap547
		d153 = snap548
		d154 = snap549
		d155 = snap550
		d156 = snap551
		d157 = snap552
		d158 = snap553
		d159 = snap554
		d160 = snap555
		d161 = snap556
		d162 = snap557
		d163 = snap558
		d164 = snap559
		d165 = snap560
		d166 = snap561
		d169 = snap562
		d272 = snap563
		d273 = snap564
		d274 = snap565
		d275 = snap566
		d277 = snap567
		d278 = snap568
		d279 = snap569
		d280 = snap570
		d281 = snap571
		d282 = snap572
		d283 = snap573
		d284 = snap574
		d286 = snap575
		d288 = snap576
		d289 = snap577
		d290 = snap578
		d291 = snap579
		d292 = snap580
		d295 = snap581
		d415 = snap582
		d416 = snap583
		d417 = snap584
		d418 = snap585
		d419 = snap586
		d421 = snap587
		d422 = snap588
		d423 = snap589
		d425 = snap590
		d426 = snap591
		d427 = snap592
		d428 = snap593
		d429 = snap594
		d430 = snap595
		d431 = snap596
		d432 = snap597
		d433 = snap598
		d434 = snap599
		d435 = snap600
		d436 = snap601
		d437 = snap602
		d438 = snap603
		d439 = snap604
		d440 = snap605
		d441 = snap606
		d442 = snap607
		d443 = snap608
		d444 = snap609
		d445 = snap610
		d446 = snap611
		d447 = snap612
		d448 = snap613
		d449 = snap614
		d450 = snap615
		d451 = snap616
		d452 = snap617
		d453 = snap618
		d454 = snap619
		d455 = snap620
		d456 = snap621
		d457 = snap622
		d458 = snap623
		d459 = snap624
		if !bbs[7].Rendered {
			return bbs[7].RenderPS(ps462)
		}
		return result
		ctx.FreeDesc(&d458)
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
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
		}
		if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
			d58 = ps.OverlayValues[58]
		}
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
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
		if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
			d128 = ps.OverlayValues[128]
		}
		if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
			d129 = ps.OverlayValues[129]
		}
		if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
			d130 = ps.OverlayValues[130]
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
		if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != scm.LocNone {
			d148 = ps.OverlayValues[148]
		}
		if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != scm.LocNone {
			d149 = ps.OverlayValues[149]
		}
		if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
			d150 = ps.OverlayValues[150]
		}
		if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != scm.LocNone {
			d151 = ps.OverlayValues[151]
		}
		if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != scm.LocNone {
			d152 = ps.OverlayValues[152]
		}
		if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
			d153 = ps.OverlayValues[153]
		}
		if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
			d154 = ps.OverlayValues[154]
		}
		if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != scm.LocNone {
			d155 = ps.OverlayValues[155]
		}
		if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != scm.LocNone {
			d156 = ps.OverlayValues[156]
		}
		if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != scm.LocNone {
			d157 = ps.OverlayValues[157]
		}
		if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != scm.LocNone {
			d158 = ps.OverlayValues[158]
		}
		if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != scm.LocNone {
			d159 = ps.OverlayValues[159]
		}
		if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != scm.LocNone {
			d160 = ps.OverlayValues[160]
		}
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != scm.LocNone {
			d161 = ps.OverlayValues[161]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != scm.LocNone {
			d162 = ps.OverlayValues[162]
		}
		if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != scm.LocNone {
			d163 = ps.OverlayValues[163]
		}
		if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != scm.LocNone {
			d164 = ps.OverlayValues[164]
		}
		if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != scm.LocNone {
			d165 = ps.OverlayValues[165]
		}
		if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != scm.LocNone {
			d166 = ps.OverlayValues[166]
		}
		if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != scm.LocNone {
			d169 = ps.OverlayValues[169]
		}
		if len(ps.OverlayValues) > 272 && ps.OverlayValues[272].Loc != scm.LocNone {
			d272 = ps.OverlayValues[272]
		}
		if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != scm.LocNone {
			d273 = ps.OverlayValues[273]
		}
		if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != scm.LocNone {
			d274 = ps.OverlayValues[274]
		}
		if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != scm.LocNone {
			d275 = ps.OverlayValues[275]
		}
		if len(ps.OverlayValues) > 277 && ps.OverlayValues[277].Loc != scm.LocNone {
			d277 = ps.OverlayValues[277]
		}
		if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != scm.LocNone {
			d278 = ps.OverlayValues[278]
		}
		if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != scm.LocNone {
			d279 = ps.OverlayValues[279]
		}
		if len(ps.OverlayValues) > 280 && ps.OverlayValues[280].Loc != scm.LocNone {
			d280 = ps.OverlayValues[280]
		}
		if len(ps.OverlayValues) > 281 && ps.OverlayValues[281].Loc != scm.LocNone {
			d281 = ps.OverlayValues[281]
		}
		if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != scm.LocNone {
			d282 = ps.OverlayValues[282]
		}
		if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != scm.LocNone {
			d283 = ps.OverlayValues[283]
		}
		if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != scm.LocNone {
			d284 = ps.OverlayValues[284]
		}
		if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != scm.LocNone {
			d286 = ps.OverlayValues[286]
		}
		if len(ps.OverlayValues) > 288 && ps.OverlayValues[288].Loc != scm.LocNone {
			d288 = ps.OverlayValues[288]
		}
		if len(ps.OverlayValues) > 289 && ps.OverlayValues[289].Loc != scm.LocNone {
			d289 = ps.OverlayValues[289]
		}
		if len(ps.OverlayValues) > 290 && ps.OverlayValues[290].Loc != scm.LocNone {
			d290 = ps.OverlayValues[290]
		}
		if len(ps.OverlayValues) > 291 && ps.OverlayValues[291].Loc != scm.LocNone {
			d291 = ps.OverlayValues[291]
		}
		if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != scm.LocNone {
			d292 = ps.OverlayValues[292]
		}
		if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
			d295 = ps.OverlayValues[295]
		}
		if len(ps.OverlayValues) > 415 && ps.OverlayValues[415].Loc != scm.LocNone {
			d415 = ps.OverlayValues[415]
		}
		if len(ps.OverlayValues) > 416 && ps.OverlayValues[416].Loc != scm.LocNone {
			d416 = ps.OverlayValues[416]
		}
		if len(ps.OverlayValues) > 417 && ps.OverlayValues[417].Loc != scm.LocNone {
			d417 = ps.OverlayValues[417]
		}
		if len(ps.OverlayValues) > 418 && ps.OverlayValues[418].Loc != scm.LocNone {
			d418 = ps.OverlayValues[418]
		}
		if len(ps.OverlayValues) > 419 && ps.OverlayValues[419].Loc != scm.LocNone {
			d419 = ps.OverlayValues[419]
		}
		if len(ps.OverlayValues) > 421 && ps.OverlayValues[421].Loc != scm.LocNone {
			d421 = ps.OverlayValues[421]
		}
		if len(ps.OverlayValues) > 422 && ps.OverlayValues[422].Loc != scm.LocNone {
			d422 = ps.OverlayValues[422]
		}
		if len(ps.OverlayValues) > 423 && ps.OverlayValues[423].Loc != scm.LocNone {
			d423 = ps.OverlayValues[423]
		}
		if len(ps.OverlayValues) > 425 && ps.OverlayValues[425].Loc != scm.LocNone {
			d425 = ps.OverlayValues[425]
		}
		if len(ps.OverlayValues) > 426 && ps.OverlayValues[426].Loc != scm.LocNone {
			d426 = ps.OverlayValues[426]
		}
		if len(ps.OverlayValues) > 427 && ps.OverlayValues[427].Loc != scm.LocNone {
			d427 = ps.OverlayValues[427]
		}
		if len(ps.OverlayValues) > 428 && ps.OverlayValues[428].Loc != scm.LocNone {
			d428 = ps.OverlayValues[428]
		}
		if len(ps.OverlayValues) > 429 && ps.OverlayValues[429].Loc != scm.LocNone {
			d429 = ps.OverlayValues[429]
		}
		if len(ps.OverlayValues) > 430 && ps.OverlayValues[430].Loc != scm.LocNone {
			d430 = ps.OverlayValues[430]
		}
		if len(ps.OverlayValues) > 431 && ps.OverlayValues[431].Loc != scm.LocNone {
			d431 = ps.OverlayValues[431]
		}
		if len(ps.OverlayValues) > 432 && ps.OverlayValues[432].Loc != scm.LocNone {
			d432 = ps.OverlayValues[432]
		}
		if len(ps.OverlayValues) > 433 && ps.OverlayValues[433].Loc != scm.LocNone {
			d433 = ps.OverlayValues[433]
		}
		if len(ps.OverlayValues) > 434 && ps.OverlayValues[434].Loc != scm.LocNone {
			d434 = ps.OverlayValues[434]
		}
		if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != scm.LocNone {
			d435 = ps.OverlayValues[435]
		}
		if len(ps.OverlayValues) > 436 && ps.OverlayValues[436].Loc != scm.LocNone {
			d436 = ps.OverlayValues[436]
		}
		if len(ps.OverlayValues) > 437 && ps.OverlayValues[437].Loc != scm.LocNone {
			d437 = ps.OverlayValues[437]
		}
		if len(ps.OverlayValues) > 438 && ps.OverlayValues[438].Loc != scm.LocNone {
			d438 = ps.OverlayValues[438]
		}
		if len(ps.OverlayValues) > 439 && ps.OverlayValues[439].Loc != scm.LocNone {
			d439 = ps.OverlayValues[439]
		}
		if len(ps.OverlayValues) > 440 && ps.OverlayValues[440].Loc != scm.LocNone {
			d440 = ps.OverlayValues[440]
		}
		if len(ps.OverlayValues) > 441 && ps.OverlayValues[441].Loc != scm.LocNone {
			d441 = ps.OverlayValues[441]
		}
		if len(ps.OverlayValues) > 442 && ps.OverlayValues[442].Loc != scm.LocNone {
			d442 = ps.OverlayValues[442]
		}
		if len(ps.OverlayValues) > 443 && ps.OverlayValues[443].Loc != scm.LocNone {
			d443 = ps.OverlayValues[443]
		}
		if len(ps.OverlayValues) > 444 && ps.OverlayValues[444].Loc != scm.LocNone {
			d444 = ps.OverlayValues[444]
		}
		if len(ps.OverlayValues) > 445 && ps.OverlayValues[445].Loc != scm.LocNone {
			d445 = ps.OverlayValues[445]
		}
		if len(ps.OverlayValues) > 446 && ps.OverlayValues[446].Loc != scm.LocNone {
			d446 = ps.OverlayValues[446]
		}
		if len(ps.OverlayValues) > 447 && ps.OverlayValues[447].Loc != scm.LocNone {
			d447 = ps.OverlayValues[447]
		}
		if len(ps.OverlayValues) > 448 && ps.OverlayValues[448].Loc != scm.LocNone {
			d448 = ps.OverlayValues[448]
		}
		if len(ps.OverlayValues) > 449 && ps.OverlayValues[449].Loc != scm.LocNone {
			d449 = ps.OverlayValues[449]
		}
		if len(ps.OverlayValues) > 450 && ps.OverlayValues[450].Loc != scm.LocNone {
			d450 = ps.OverlayValues[450]
		}
		if len(ps.OverlayValues) > 451 && ps.OverlayValues[451].Loc != scm.LocNone {
			d451 = ps.OverlayValues[451]
		}
		if len(ps.OverlayValues) > 452 && ps.OverlayValues[452].Loc != scm.LocNone {
			d452 = ps.OverlayValues[452]
		}
		if len(ps.OverlayValues) > 453 && ps.OverlayValues[453].Loc != scm.LocNone {
			d453 = ps.OverlayValues[453]
		}
		if len(ps.OverlayValues) > 454 && ps.OverlayValues[454].Loc != scm.LocNone {
			d454 = ps.OverlayValues[454]
		}
		if len(ps.OverlayValues) > 455 && ps.OverlayValues[455].Loc != scm.LocNone {
			d455 = ps.OverlayValues[455]
		}
		if len(ps.OverlayValues) > 456 && ps.OverlayValues[456].Loc != scm.LocNone {
			d456 = ps.OverlayValues[456]
		}
		if len(ps.OverlayValues) > 457 && ps.OverlayValues[457].Loc != scm.LocNone {
			d457 = ps.OverlayValues[457]
		}
		if len(ps.OverlayValues) > 458 && ps.OverlayValues[458].Loc != scm.LocNone {
			d458 = ps.OverlayValues[458]
		}
		if len(ps.OverlayValues) > 459 && ps.OverlayValues[459].Loc != scm.LocNone {
			d459 = ps.OverlayValues[459]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d5)
		var d626 scm.JITValueDesc
		if d5.Loc == scm.LocImm {
			d626 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d626 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d626)
		}
		if d626.Loc == scm.LocImm {
			d626 = scm.JITValueDesc{Loc: scm.LocImm, Type: d626.Type, Imm: scm.NewInt(int64(uint64(d626.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d626.Reg, 32)
			ctx.EmitShrRegImm8(d626.Reg, 32)
		}
		if d626.Loc == scm.LocReg && d5.Loc == scm.LocReg && d626.Reg == d5.Reg {
			ctx.TransferReg(d5.Reg)
			d5.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d626)
		ctx.EmitStoreToStack(d626, int32(bbs[8].PhiBase)+int32(16))
		ctx.StabilizeDescForControlFlow(&d626)
		if ps.General {
			ctx.SyncDesc(&d6)
			if d6.Loc == scm.LocReg {
				ctx.ProtectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.ProtectReg(d6.Reg)
				ctx.ProtectReg(d6.Reg2)
			}
			d627 = d6
			if d627.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d627)
			d628 = d627
			if d628.Loc == scm.LocImm {
				d628 = scm.JITValueDesc{Loc: scm.LocImm, Type: d628.Type, Imm: scm.NewInt(int64(uint64(d628.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d628.Reg, 32)
				ctx.EmitShrRegImm8(d628.Reg, 32)
			}
			ctx.EmitStoreToStack(d628, int32(bbs[8].PhiBase)+int32(0))
			if d6.Loc == scm.LocReg {
				ctx.UnprotectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d6.Reg)
				ctx.UnprotectReg(d6.Reg2)
			}
		}
		ps629 := scm.PhiState{General: ps.General}
		ps629.OverlayValues = make([]scm.JITValueDesc, 629)
		ps629.OverlayValues[1] = d1
		ps629.OverlayValues[2] = d2
		ps629.OverlayValues[3] = d3
		ps629.OverlayValues[4] = d4
		ps629.OverlayValues[5] = d5
		ps629.OverlayValues[6] = d6
		ps629.OverlayValues[7] = d7
		ps629.OverlayValues[8] = d8
		ps629.OverlayValues[9] = d9
		ps629.OverlayValues[10] = d10
		ps629.OverlayValues[11] = d11
		ps629.OverlayValues[12] = d12
		ps629.OverlayValues[13] = d13
		ps629.OverlayValues[14] = d14
		ps629.OverlayValues[15] = d15
		ps629.OverlayValues[17] = d17
		ps629.OverlayValues[18] = d18
		ps629.OverlayValues[19] = d19
		ps629.OverlayValues[20] = d20
		ps629.OverlayValues[21] = d21
		ps629.OverlayValues[22] = d22
		ps629.OverlayValues[24] = d24
		ps629.OverlayValues[25] = d25
		ps629.OverlayValues[26] = d26
		ps629.OverlayValues[27] = d27
		ps629.OverlayValues[28] = d28
		ps629.OverlayValues[29] = d29
		ps629.OverlayValues[30] = d30
		ps629.OverlayValues[31] = d31
		ps629.OverlayValues[32] = d32
		ps629.OverlayValues[33] = d33
		ps629.OverlayValues[34] = d34
		ps629.OverlayValues[35] = d35
		ps629.OverlayValues[36] = d36
		ps629.OverlayValues[37] = d37
		ps629.OverlayValues[38] = d38
		ps629.OverlayValues[39] = d39
		ps629.OverlayValues[40] = d40
		ps629.OverlayValues[41] = d41
		ps629.OverlayValues[42] = d42
		ps629.OverlayValues[43] = d43
		ps629.OverlayValues[44] = d44
		ps629.OverlayValues[45] = d45
		ps629.OverlayValues[46] = d46
		ps629.OverlayValues[47] = d47
		ps629.OverlayValues[48] = d48
		ps629.OverlayValues[49] = d49
		ps629.OverlayValues[50] = d50
		ps629.OverlayValues[51] = d51
		ps629.OverlayValues[52] = d52
		ps629.OverlayValues[53] = d53
		ps629.OverlayValues[54] = d54
		ps629.OverlayValues[55] = d55
		ps629.OverlayValues[56] = d56
		ps629.OverlayValues[57] = d57
		ps629.OverlayValues[58] = d58
		ps629.OverlayValues[59] = d59
		ps629.OverlayValues[62] = d62
		ps629.OverlayValues[63] = d63
		ps629.OverlayValues[64] = d64
		ps629.OverlayValues[128] = d128
		ps629.OverlayValues[129] = d129
		ps629.OverlayValues[130] = d130
		ps629.OverlayValues[132] = d132
		ps629.OverlayValues[133] = d133
		ps629.OverlayValues[134] = d134
		ps629.OverlayValues[135] = d135
		ps629.OverlayValues[136] = d136
		ps629.OverlayValues[137] = d137
		ps629.OverlayValues[138] = d138
		ps629.OverlayValues[139] = d139
		ps629.OverlayValues[140] = d140
		ps629.OverlayValues[141] = d141
		ps629.OverlayValues[142] = d142
		ps629.OverlayValues[143] = d143
		ps629.OverlayValues[144] = d144
		ps629.OverlayValues[145] = d145
		ps629.OverlayValues[146] = d146
		ps629.OverlayValues[147] = d147
		ps629.OverlayValues[148] = d148
		ps629.OverlayValues[149] = d149
		ps629.OverlayValues[150] = d150
		ps629.OverlayValues[151] = d151
		ps629.OverlayValues[152] = d152
		ps629.OverlayValues[153] = d153
		ps629.OverlayValues[154] = d154
		ps629.OverlayValues[155] = d155
		ps629.OverlayValues[156] = d156
		ps629.OverlayValues[157] = d157
		ps629.OverlayValues[158] = d158
		ps629.OverlayValues[159] = d159
		ps629.OverlayValues[160] = d160
		ps629.OverlayValues[161] = d161
		ps629.OverlayValues[162] = d162
		ps629.OverlayValues[163] = d163
		ps629.OverlayValues[164] = d164
		ps629.OverlayValues[165] = d165
		ps629.OverlayValues[166] = d166
		ps629.OverlayValues[169] = d169
		ps629.OverlayValues[272] = d272
		ps629.OverlayValues[273] = d273
		ps629.OverlayValues[274] = d274
		ps629.OverlayValues[275] = d275
		ps629.OverlayValues[277] = d277
		ps629.OverlayValues[278] = d278
		ps629.OverlayValues[279] = d279
		ps629.OverlayValues[280] = d280
		ps629.OverlayValues[281] = d281
		ps629.OverlayValues[282] = d282
		ps629.OverlayValues[283] = d283
		ps629.OverlayValues[284] = d284
		ps629.OverlayValues[286] = d286
		ps629.OverlayValues[288] = d288
		ps629.OverlayValues[289] = d289
		ps629.OverlayValues[290] = d290
		ps629.OverlayValues[291] = d291
		ps629.OverlayValues[292] = d292
		ps629.OverlayValues[295] = d295
		ps629.OverlayValues[415] = d415
		ps629.OverlayValues[416] = d416
		ps629.OverlayValues[417] = d417
		ps629.OverlayValues[418] = d418
		ps629.OverlayValues[419] = d419
		ps629.OverlayValues[421] = d421
		ps629.OverlayValues[422] = d422
		ps629.OverlayValues[423] = d423
		ps629.OverlayValues[425] = d425
		ps629.OverlayValues[426] = d426
		ps629.OverlayValues[427] = d427
		ps629.OverlayValues[428] = d428
		ps629.OverlayValues[429] = d429
		ps629.OverlayValues[430] = d430
		ps629.OverlayValues[431] = d431
		ps629.OverlayValues[432] = d432
		ps629.OverlayValues[433] = d433
		ps629.OverlayValues[434] = d434
		ps629.OverlayValues[435] = d435
		ps629.OverlayValues[436] = d436
		ps629.OverlayValues[437] = d437
		ps629.OverlayValues[438] = d438
		ps629.OverlayValues[439] = d439
		ps629.OverlayValues[440] = d440
		ps629.OverlayValues[441] = d441
		ps629.OverlayValues[442] = d442
		ps629.OverlayValues[443] = d443
		ps629.OverlayValues[444] = d444
		ps629.OverlayValues[445] = d445
		ps629.OverlayValues[446] = d446
		ps629.OverlayValues[447] = d447
		ps629.OverlayValues[448] = d448
		ps629.OverlayValues[449] = d449
		ps629.OverlayValues[450] = d450
		ps629.OverlayValues[451] = d451
		ps629.OverlayValues[452] = d452
		ps629.OverlayValues[453] = d453
		ps629.OverlayValues[454] = d454
		ps629.OverlayValues[455] = d455
		ps629.OverlayValues[456] = d456
		ps629.OverlayValues[457] = d457
		ps629.OverlayValues[458] = d458
		ps629.OverlayValues[459] = d459
		ps629.OverlayValues[626] = d626
		ps629.OverlayValues[627] = d627
		ps629.OverlayValues[628] = d628
		ps629.PhiValues = make([]scm.JITValueDesc, 2)
		d630 = d6
		ps629.PhiValues[0] = d630
		if ps629.General && bbs[8].Rendered {
			ctx.EmitJmp(lbl9)
			return result
		}
		return bbs[8].RenderPS(ps629)
		return result
	}
	bbs[8].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d631 := ps.PhiValues[0]
				ctx.EnsureDesc(&d631)
				ctx.EmitStoreToStack(d631, int32(bbs[8].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d632 := ps.PhiValues[1]
				ctx.EnsureDesc(&d632)
				ctx.EmitStoreToStack(d632, int32(bbs[8].PhiBase)+int32(16))
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
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
		}
		if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
			d58 = ps.OverlayValues[58]
		}
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
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
		if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
			d128 = ps.OverlayValues[128]
		}
		if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
			d129 = ps.OverlayValues[129]
		}
		if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
			d130 = ps.OverlayValues[130]
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
		if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != scm.LocNone {
			d148 = ps.OverlayValues[148]
		}
		if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != scm.LocNone {
			d149 = ps.OverlayValues[149]
		}
		if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
			d150 = ps.OverlayValues[150]
		}
		if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != scm.LocNone {
			d151 = ps.OverlayValues[151]
		}
		if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != scm.LocNone {
			d152 = ps.OverlayValues[152]
		}
		if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
			d153 = ps.OverlayValues[153]
		}
		if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
			d154 = ps.OverlayValues[154]
		}
		if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != scm.LocNone {
			d155 = ps.OverlayValues[155]
		}
		if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != scm.LocNone {
			d156 = ps.OverlayValues[156]
		}
		if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != scm.LocNone {
			d157 = ps.OverlayValues[157]
		}
		if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != scm.LocNone {
			d158 = ps.OverlayValues[158]
		}
		if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != scm.LocNone {
			d159 = ps.OverlayValues[159]
		}
		if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != scm.LocNone {
			d160 = ps.OverlayValues[160]
		}
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != scm.LocNone {
			d161 = ps.OverlayValues[161]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != scm.LocNone {
			d162 = ps.OverlayValues[162]
		}
		if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != scm.LocNone {
			d163 = ps.OverlayValues[163]
		}
		if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != scm.LocNone {
			d164 = ps.OverlayValues[164]
		}
		if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != scm.LocNone {
			d165 = ps.OverlayValues[165]
		}
		if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != scm.LocNone {
			d166 = ps.OverlayValues[166]
		}
		if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != scm.LocNone {
			d169 = ps.OverlayValues[169]
		}
		if len(ps.OverlayValues) > 272 && ps.OverlayValues[272].Loc != scm.LocNone {
			d272 = ps.OverlayValues[272]
		}
		if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != scm.LocNone {
			d273 = ps.OverlayValues[273]
		}
		if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != scm.LocNone {
			d274 = ps.OverlayValues[274]
		}
		if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != scm.LocNone {
			d275 = ps.OverlayValues[275]
		}
		if len(ps.OverlayValues) > 277 && ps.OverlayValues[277].Loc != scm.LocNone {
			d277 = ps.OverlayValues[277]
		}
		if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != scm.LocNone {
			d278 = ps.OverlayValues[278]
		}
		if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != scm.LocNone {
			d279 = ps.OverlayValues[279]
		}
		if len(ps.OverlayValues) > 280 && ps.OverlayValues[280].Loc != scm.LocNone {
			d280 = ps.OverlayValues[280]
		}
		if len(ps.OverlayValues) > 281 && ps.OverlayValues[281].Loc != scm.LocNone {
			d281 = ps.OverlayValues[281]
		}
		if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != scm.LocNone {
			d282 = ps.OverlayValues[282]
		}
		if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != scm.LocNone {
			d283 = ps.OverlayValues[283]
		}
		if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != scm.LocNone {
			d284 = ps.OverlayValues[284]
		}
		if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != scm.LocNone {
			d286 = ps.OverlayValues[286]
		}
		if len(ps.OverlayValues) > 288 && ps.OverlayValues[288].Loc != scm.LocNone {
			d288 = ps.OverlayValues[288]
		}
		if len(ps.OverlayValues) > 289 && ps.OverlayValues[289].Loc != scm.LocNone {
			d289 = ps.OverlayValues[289]
		}
		if len(ps.OverlayValues) > 290 && ps.OverlayValues[290].Loc != scm.LocNone {
			d290 = ps.OverlayValues[290]
		}
		if len(ps.OverlayValues) > 291 && ps.OverlayValues[291].Loc != scm.LocNone {
			d291 = ps.OverlayValues[291]
		}
		if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != scm.LocNone {
			d292 = ps.OverlayValues[292]
		}
		if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
			d295 = ps.OverlayValues[295]
		}
		if len(ps.OverlayValues) > 415 && ps.OverlayValues[415].Loc != scm.LocNone {
			d415 = ps.OverlayValues[415]
		}
		if len(ps.OverlayValues) > 416 && ps.OverlayValues[416].Loc != scm.LocNone {
			d416 = ps.OverlayValues[416]
		}
		if len(ps.OverlayValues) > 417 && ps.OverlayValues[417].Loc != scm.LocNone {
			d417 = ps.OverlayValues[417]
		}
		if len(ps.OverlayValues) > 418 && ps.OverlayValues[418].Loc != scm.LocNone {
			d418 = ps.OverlayValues[418]
		}
		if len(ps.OverlayValues) > 419 && ps.OverlayValues[419].Loc != scm.LocNone {
			d419 = ps.OverlayValues[419]
		}
		if len(ps.OverlayValues) > 421 && ps.OverlayValues[421].Loc != scm.LocNone {
			d421 = ps.OverlayValues[421]
		}
		if len(ps.OverlayValues) > 422 && ps.OverlayValues[422].Loc != scm.LocNone {
			d422 = ps.OverlayValues[422]
		}
		if len(ps.OverlayValues) > 423 && ps.OverlayValues[423].Loc != scm.LocNone {
			d423 = ps.OverlayValues[423]
		}
		if len(ps.OverlayValues) > 425 && ps.OverlayValues[425].Loc != scm.LocNone {
			d425 = ps.OverlayValues[425]
		}
		if len(ps.OverlayValues) > 426 && ps.OverlayValues[426].Loc != scm.LocNone {
			d426 = ps.OverlayValues[426]
		}
		if len(ps.OverlayValues) > 427 && ps.OverlayValues[427].Loc != scm.LocNone {
			d427 = ps.OverlayValues[427]
		}
		if len(ps.OverlayValues) > 428 && ps.OverlayValues[428].Loc != scm.LocNone {
			d428 = ps.OverlayValues[428]
		}
		if len(ps.OverlayValues) > 429 && ps.OverlayValues[429].Loc != scm.LocNone {
			d429 = ps.OverlayValues[429]
		}
		if len(ps.OverlayValues) > 430 && ps.OverlayValues[430].Loc != scm.LocNone {
			d430 = ps.OverlayValues[430]
		}
		if len(ps.OverlayValues) > 431 && ps.OverlayValues[431].Loc != scm.LocNone {
			d431 = ps.OverlayValues[431]
		}
		if len(ps.OverlayValues) > 432 && ps.OverlayValues[432].Loc != scm.LocNone {
			d432 = ps.OverlayValues[432]
		}
		if len(ps.OverlayValues) > 433 && ps.OverlayValues[433].Loc != scm.LocNone {
			d433 = ps.OverlayValues[433]
		}
		if len(ps.OverlayValues) > 434 && ps.OverlayValues[434].Loc != scm.LocNone {
			d434 = ps.OverlayValues[434]
		}
		if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != scm.LocNone {
			d435 = ps.OverlayValues[435]
		}
		if len(ps.OverlayValues) > 436 && ps.OverlayValues[436].Loc != scm.LocNone {
			d436 = ps.OverlayValues[436]
		}
		if len(ps.OverlayValues) > 437 && ps.OverlayValues[437].Loc != scm.LocNone {
			d437 = ps.OverlayValues[437]
		}
		if len(ps.OverlayValues) > 438 && ps.OverlayValues[438].Loc != scm.LocNone {
			d438 = ps.OverlayValues[438]
		}
		if len(ps.OverlayValues) > 439 && ps.OverlayValues[439].Loc != scm.LocNone {
			d439 = ps.OverlayValues[439]
		}
		if len(ps.OverlayValues) > 440 && ps.OverlayValues[440].Loc != scm.LocNone {
			d440 = ps.OverlayValues[440]
		}
		if len(ps.OverlayValues) > 441 && ps.OverlayValues[441].Loc != scm.LocNone {
			d441 = ps.OverlayValues[441]
		}
		if len(ps.OverlayValues) > 442 && ps.OverlayValues[442].Loc != scm.LocNone {
			d442 = ps.OverlayValues[442]
		}
		if len(ps.OverlayValues) > 443 && ps.OverlayValues[443].Loc != scm.LocNone {
			d443 = ps.OverlayValues[443]
		}
		if len(ps.OverlayValues) > 444 && ps.OverlayValues[444].Loc != scm.LocNone {
			d444 = ps.OverlayValues[444]
		}
		if len(ps.OverlayValues) > 445 && ps.OverlayValues[445].Loc != scm.LocNone {
			d445 = ps.OverlayValues[445]
		}
		if len(ps.OverlayValues) > 446 && ps.OverlayValues[446].Loc != scm.LocNone {
			d446 = ps.OverlayValues[446]
		}
		if len(ps.OverlayValues) > 447 && ps.OverlayValues[447].Loc != scm.LocNone {
			d447 = ps.OverlayValues[447]
		}
		if len(ps.OverlayValues) > 448 && ps.OverlayValues[448].Loc != scm.LocNone {
			d448 = ps.OverlayValues[448]
		}
		if len(ps.OverlayValues) > 449 && ps.OverlayValues[449].Loc != scm.LocNone {
			d449 = ps.OverlayValues[449]
		}
		if len(ps.OverlayValues) > 450 && ps.OverlayValues[450].Loc != scm.LocNone {
			d450 = ps.OverlayValues[450]
		}
		if len(ps.OverlayValues) > 451 && ps.OverlayValues[451].Loc != scm.LocNone {
			d451 = ps.OverlayValues[451]
		}
		if len(ps.OverlayValues) > 452 && ps.OverlayValues[452].Loc != scm.LocNone {
			d452 = ps.OverlayValues[452]
		}
		if len(ps.OverlayValues) > 453 && ps.OverlayValues[453].Loc != scm.LocNone {
			d453 = ps.OverlayValues[453]
		}
		if len(ps.OverlayValues) > 454 && ps.OverlayValues[454].Loc != scm.LocNone {
			d454 = ps.OverlayValues[454]
		}
		if len(ps.OverlayValues) > 455 && ps.OverlayValues[455].Loc != scm.LocNone {
			d455 = ps.OverlayValues[455]
		}
		if len(ps.OverlayValues) > 456 && ps.OverlayValues[456].Loc != scm.LocNone {
			d456 = ps.OverlayValues[456]
		}
		if len(ps.OverlayValues) > 457 && ps.OverlayValues[457].Loc != scm.LocNone {
			d457 = ps.OverlayValues[457]
		}
		if len(ps.OverlayValues) > 458 && ps.OverlayValues[458].Loc != scm.LocNone {
			d458 = ps.OverlayValues[458]
		}
		if len(ps.OverlayValues) > 459 && ps.OverlayValues[459].Loc != scm.LocNone {
			d459 = ps.OverlayValues[459]
		}
		if len(ps.OverlayValues) > 626 && ps.OverlayValues[626].Loc != scm.LocNone {
			d626 = ps.OverlayValues[626]
		}
		if len(ps.OverlayValues) > 627 && ps.OverlayValues[627].Loc != scm.LocNone {
			d627 = ps.OverlayValues[627]
		}
		if len(ps.OverlayValues) > 628 && ps.OverlayValues[628].Loc != scm.LocNone {
			d628 = ps.OverlayValues[628]
		}
		if len(ps.OverlayValues) > 630 && ps.OverlayValues[630].Loc != scm.LocNone {
			d630 = ps.OverlayValues[630]
		}
		if len(ps.OverlayValues) > 631 && ps.OverlayValues[631].Loc != scm.LocNone {
			d631 = ps.OverlayValues[631]
		}
		if len(ps.OverlayValues) > 632 && ps.OverlayValues[632].Loc != scm.LocNone {
			d632 = ps.OverlayValues[632]
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
		ctx.EnsureDesc(&d8)
		ctx.EnsureDesc(&d9)
		ctx.EnsureDesc(&d8)
		ctx.EnsureDesc(&d9)
		var d633 scm.JITValueDesc
		if d8.Loc == scm.LocImm && d9.Loc == scm.LocImm {
			d633 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d8.Imm.Int()) == uint64(d9.Imm.Int()))}
		} else if d9.Loc == scm.LocImm {
			r156 := ctx.AllocRegExcept(d8.Reg)
			if d9.Imm.Int() >= -2147483648 && d9.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d8.Reg, int32(d9.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d9.Imm.Int()))
				ctx.EmitCmpInt64(d8.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r156, scm.CondEqual)
			d633 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r156}
			ctx.BindReg(r156, &d633)
		} else if d8.Loc == scm.LocImm {
			r157 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d8.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d9.Reg)
			ctx.EmitSetcc(r157, scm.CondEqual)
			d633 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r157}
			ctx.BindReg(r157, &d633)
		} else {
			r158 := ctx.AllocRegExcept(d8.Reg)
			ctx.EmitCmpInt64(d8.Reg, d9.Reg)
			ctx.EmitSetcc(r158, scm.CondEqual)
			d633 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r158}
			ctx.BindReg(r158, &d633)
		}
		d634 = d633
		ctx.EnsureDesc(&d634)
		if d634.Loc != scm.LocImm && d634.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d634.Loc == scm.LocImm {
			if d634.Imm.Bool() {
				if ps.General {
					ctx.SyncDesc(&d8)
					if d8.Loc == scm.LocReg {
						ctx.ProtectReg(d8.Reg)
					} else if d8.Loc == scm.LocRegPair {
						ctx.ProtectReg(d8.Reg)
						ctx.ProtectReg(d8.Reg2)
					}
					d635 = d8
					if d635.Loc == scm.LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d635)
					d636 = d635
					if d636.Loc == scm.LocImm {
						d636 = scm.JITValueDesc{Loc: scm.LocImm, Type: d636.Type, Imm: scm.NewInt(int64(uint64(d636.Imm.Int()) & 0xffffffff))}
					} else {
						ctx.EmitShlRegImm8(d636.Reg, 32)
						ctx.EmitShrRegImm8(d636.Reg, 32)
					}
					ctx.EmitStoreToStack(d636, int32(bbs[2].PhiBase)+int32(0))
					if d8.Loc == scm.LocReg {
						ctx.UnprotectReg(d8.Reg)
					} else if d8.Loc == scm.LocRegPair {
						ctx.UnprotectReg(d8.Reg)
						ctx.UnprotectReg(d8.Reg2)
					}
				}
				ps637 := scm.PhiState{General: ps.General}
				ps637.OverlayValues = make([]scm.JITValueDesc, 637)
				ps637.OverlayValues[1] = d1
				ps637.OverlayValues[2] = d2
				ps637.OverlayValues[3] = d3
				ps637.OverlayValues[4] = d4
				ps637.OverlayValues[5] = d5
				ps637.OverlayValues[6] = d6
				ps637.OverlayValues[7] = d7
				ps637.OverlayValues[8] = d8
				ps637.OverlayValues[9] = d9
				ps637.OverlayValues[10] = d10
				ps637.OverlayValues[11] = d11
				ps637.OverlayValues[12] = d12
				ps637.OverlayValues[13] = d13
				ps637.OverlayValues[14] = d14
				ps637.OverlayValues[15] = d15
				ps637.OverlayValues[17] = d17
				ps637.OverlayValues[18] = d18
				ps637.OverlayValues[19] = d19
				ps637.OverlayValues[20] = d20
				ps637.OverlayValues[21] = d21
				ps637.OverlayValues[22] = d22
				ps637.OverlayValues[24] = d24
				ps637.OverlayValues[25] = d25
				ps637.OverlayValues[26] = d26
				ps637.OverlayValues[27] = d27
				ps637.OverlayValues[28] = d28
				ps637.OverlayValues[29] = d29
				ps637.OverlayValues[30] = d30
				ps637.OverlayValues[31] = d31
				ps637.OverlayValues[32] = d32
				ps637.OverlayValues[33] = d33
				ps637.OverlayValues[34] = d34
				ps637.OverlayValues[35] = d35
				ps637.OverlayValues[36] = d36
				ps637.OverlayValues[37] = d37
				ps637.OverlayValues[38] = d38
				ps637.OverlayValues[39] = d39
				ps637.OverlayValues[40] = d40
				ps637.OverlayValues[41] = d41
				ps637.OverlayValues[42] = d42
				ps637.OverlayValues[43] = d43
				ps637.OverlayValues[44] = d44
				ps637.OverlayValues[45] = d45
				ps637.OverlayValues[46] = d46
				ps637.OverlayValues[47] = d47
				ps637.OverlayValues[48] = d48
				ps637.OverlayValues[49] = d49
				ps637.OverlayValues[50] = d50
				ps637.OverlayValues[51] = d51
				ps637.OverlayValues[52] = d52
				ps637.OverlayValues[53] = d53
				ps637.OverlayValues[54] = d54
				ps637.OverlayValues[55] = d55
				ps637.OverlayValues[56] = d56
				ps637.OverlayValues[57] = d57
				ps637.OverlayValues[58] = d58
				ps637.OverlayValues[59] = d59
				ps637.OverlayValues[62] = d62
				ps637.OverlayValues[63] = d63
				ps637.OverlayValues[64] = d64
				ps637.OverlayValues[128] = d128
				ps637.OverlayValues[129] = d129
				ps637.OverlayValues[130] = d130
				ps637.OverlayValues[132] = d132
				ps637.OverlayValues[133] = d133
				ps637.OverlayValues[134] = d134
				ps637.OverlayValues[135] = d135
				ps637.OverlayValues[136] = d136
				ps637.OverlayValues[137] = d137
				ps637.OverlayValues[138] = d138
				ps637.OverlayValues[139] = d139
				ps637.OverlayValues[140] = d140
				ps637.OverlayValues[141] = d141
				ps637.OverlayValues[142] = d142
				ps637.OverlayValues[143] = d143
				ps637.OverlayValues[144] = d144
				ps637.OverlayValues[145] = d145
				ps637.OverlayValues[146] = d146
				ps637.OverlayValues[147] = d147
				ps637.OverlayValues[148] = d148
				ps637.OverlayValues[149] = d149
				ps637.OverlayValues[150] = d150
				ps637.OverlayValues[151] = d151
				ps637.OverlayValues[152] = d152
				ps637.OverlayValues[153] = d153
				ps637.OverlayValues[154] = d154
				ps637.OverlayValues[155] = d155
				ps637.OverlayValues[156] = d156
				ps637.OverlayValues[157] = d157
				ps637.OverlayValues[158] = d158
				ps637.OverlayValues[159] = d159
				ps637.OverlayValues[160] = d160
				ps637.OverlayValues[161] = d161
				ps637.OverlayValues[162] = d162
				ps637.OverlayValues[163] = d163
				ps637.OverlayValues[164] = d164
				ps637.OverlayValues[165] = d165
				ps637.OverlayValues[166] = d166
				ps637.OverlayValues[169] = d169
				ps637.OverlayValues[272] = d272
				ps637.OverlayValues[273] = d273
				ps637.OverlayValues[274] = d274
				ps637.OverlayValues[275] = d275
				ps637.OverlayValues[277] = d277
				ps637.OverlayValues[278] = d278
				ps637.OverlayValues[279] = d279
				ps637.OverlayValues[280] = d280
				ps637.OverlayValues[281] = d281
				ps637.OverlayValues[282] = d282
				ps637.OverlayValues[283] = d283
				ps637.OverlayValues[284] = d284
				ps637.OverlayValues[286] = d286
				ps637.OverlayValues[288] = d288
				ps637.OverlayValues[289] = d289
				ps637.OverlayValues[290] = d290
				ps637.OverlayValues[291] = d291
				ps637.OverlayValues[292] = d292
				ps637.OverlayValues[295] = d295
				ps637.OverlayValues[415] = d415
				ps637.OverlayValues[416] = d416
				ps637.OverlayValues[417] = d417
				ps637.OverlayValues[418] = d418
				ps637.OverlayValues[419] = d419
				ps637.OverlayValues[421] = d421
				ps637.OverlayValues[422] = d422
				ps637.OverlayValues[423] = d423
				ps637.OverlayValues[425] = d425
				ps637.OverlayValues[426] = d426
				ps637.OverlayValues[427] = d427
				ps637.OverlayValues[428] = d428
				ps637.OverlayValues[429] = d429
				ps637.OverlayValues[430] = d430
				ps637.OverlayValues[431] = d431
				ps637.OverlayValues[432] = d432
				ps637.OverlayValues[433] = d433
				ps637.OverlayValues[434] = d434
				ps637.OverlayValues[435] = d435
				ps637.OverlayValues[436] = d436
				ps637.OverlayValues[437] = d437
				ps637.OverlayValues[438] = d438
				ps637.OverlayValues[439] = d439
				ps637.OverlayValues[440] = d440
				ps637.OverlayValues[441] = d441
				ps637.OverlayValues[442] = d442
				ps637.OverlayValues[443] = d443
				ps637.OverlayValues[444] = d444
				ps637.OverlayValues[445] = d445
				ps637.OverlayValues[446] = d446
				ps637.OverlayValues[447] = d447
				ps637.OverlayValues[448] = d448
				ps637.OverlayValues[449] = d449
				ps637.OverlayValues[450] = d450
				ps637.OverlayValues[451] = d451
				ps637.OverlayValues[452] = d452
				ps637.OverlayValues[453] = d453
				ps637.OverlayValues[454] = d454
				ps637.OverlayValues[455] = d455
				ps637.OverlayValues[456] = d456
				ps637.OverlayValues[457] = d457
				ps637.OverlayValues[458] = d458
				ps637.OverlayValues[459] = d459
				ps637.OverlayValues[626] = d626
				ps637.OverlayValues[627] = d627
				ps637.OverlayValues[628] = d628
				ps637.OverlayValues[630] = d630
				ps637.OverlayValues[631] = d631
				ps637.OverlayValues[632] = d632
				ps637.OverlayValues[633] = d633
				ps637.OverlayValues[634] = d634
				ps637.OverlayValues[635] = d635
				ps637.OverlayValues[636] = d636
				ps637.PhiValues = make([]scm.JITValueDesc, 1)
				d638 = d8
				ps637.PhiValues[0] = d638
				return bbs[2].RenderPS(ps637)
			}
			if ps.General {
			}
			ps639 := scm.PhiState{General: ps.General}
			ps639.OverlayValues = make([]scm.JITValueDesc, 639)
			ps639.OverlayValues[1] = d1
			ps639.OverlayValues[2] = d2
			ps639.OverlayValues[3] = d3
			ps639.OverlayValues[4] = d4
			ps639.OverlayValues[5] = d5
			ps639.OverlayValues[6] = d6
			ps639.OverlayValues[7] = d7
			ps639.OverlayValues[8] = d8
			ps639.OverlayValues[9] = d9
			ps639.OverlayValues[10] = d10
			ps639.OverlayValues[11] = d11
			ps639.OverlayValues[12] = d12
			ps639.OverlayValues[13] = d13
			ps639.OverlayValues[14] = d14
			ps639.OverlayValues[15] = d15
			ps639.OverlayValues[17] = d17
			ps639.OverlayValues[18] = d18
			ps639.OverlayValues[19] = d19
			ps639.OverlayValues[20] = d20
			ps639.OverlayValues[21] = d21
			ps639.OverlayValues[22] = d22
			ps639.OverlayValues[24] = d24
			ps639.OverlayValues[25] = d25
			ps639.OverlayValues[26] = d26
			ps639.OverlayValues[27] = d27
			ps639.OverlayValues[28] = d28
			ps639.OverlayValues[29] = d29
			ps639.OverlayValues[30] = d30
			ps639.OverlayValues[31] = d31
			ps639.OverlayValues[32] = d32
			ps639.OverlayValues[33] = d33
			ps639.OverlayValues[34] = d34
			ps639.OverlayValues[35] = d35
			ps639.OverlayValues[36] = d36
			ps639.OverlayValues[37] = d37
			ps639.OverlayValues[38] = d38
			ps639.OverlayValues[39] = d39
			ps639.OverlayValues[40] = d40
			ps639.OverlayValues[41] = d41
			ps639.OverlayValues[42] = d42
			ps639.OverlayValues[43] = d43
			ps639.OverlayValues[44] = d44
			ps639.OverlayValues[45] = d45
			ps639.OverlayValues[46] = d46
			ps639.OverlayValues[47] = d47
			ps639.OverlayValues[48] = d48
			ps639.OverlayValues[49] = d49
			ps639.OverlayValues[50] = d50
			ps639.OverlayValues[51] = d51
			ps639.OverlayValues[52] = d52
			ps639.OverlayValues[53] = d53
			ps639.OverlayValues[54] = d54
			ps639.OverlayValues[55] = d55
			ps639.OverlayValues[56] = d56
			ps639.OverlayValues[57] = d57
			ps639.OverlayValues[58] = d58
			ps639.OverlayValues[59] = d59
			ps639.OverlayValues[62] = d62
			ps639.OverlayValues[63] = d63
			ps639.OverlayValues[64] = d64
			ps639.OverlayValues[128] = d128
			ps639.OverlayValues[129] = d129
			ps639.OverlayValues[130] = d130
			ps639.OverlayValues[132] = d132
			ps639.OverlayValues[133] = d133
			ps639.OverlayValues[134] = d134
			ps639.OverlayValues[135] = d135
			ps639.OverlayValues[136] = d136
			ps639.OverlayValues[137] = d137
			ps639.OverlayValues[138] = d138
			ps639.OverlayValues[139] = d139
			ps639.OverlayValues[140] = d140
			ps639.OverlayValues[141] = d141
			ps639.OverlayValues[142] = d142
			ps639.OverlayValues[143] = d143
			ps639.OverlayValues[144] = d144
			ps639.OverlayValues[145] = d145
			ps639.OverlayValues[146] = d146
			ps639.OverlayValues[147] = d147
			ps639.OverlayValues[148] = d148
			ps639.OverlayValues[149] = d149
			ps639.OverlayValues[150] = d150
			ps639.OverlayValues[151] = d151
			ps639.OverlayValues[152] = d152
			ps639.OverlayValues[153] = d153
			ps639.OverlayValues[154] = d154
			ps639.OverlayValues[155] = d155
			ps639.OverlayValues[156] = d156
			ps639.OverlayValues[157] = d157
			ps639.OverlayValues[158] = d158
			ps639.OverlayValues[159] = d159
			ps639.OverlayValues[160] = d160
			ps639.OverlayValues[161] = d161
			ps639.OverlayValues[162] = d162
			ps639.OverlayValues[163] = d163
			ps639.OverlayValues[164] = d164
			ps639.OverlayValues[165] = d165
			ps639.OverlayValues[166] = d166
			ps639.OverlayValues[169] = d169
			ps639.OverlayValues[272] = d272
			ps639.OverlayValues[273] = d273
			ps639.OverlayValues[274] = d274
			ps639.OverlayValues[275] = d275
			ps639.OverlayValues[277] = d277
			ps639.OverlayValues[278] = d278
			ps639.OverlayValues[279] = d279
			ps639.OverlayValues[280] = d280
			ps639.OverlayValues[281] = d281
			ps639.OverlayValues[282] = d282
			ps639.OverlayValues[283] = d283
			ps639.OverlayValues[284] = d284
			ps639.OverlayValues[286] = d286
			ps639.OverlayValues[288] = d288
			ps639.OverlayValues[289] = d289
			ps639.OverlayValues[290] = d290
			ps639.OverlayValues[291] = d291
			ps639.OverlayValues[292] = d292
			ps639.OverlayValues[295] = d295
			ps639.OverlayValues[415] = d415
			ps639.OverlayValues[416] = d416
			ps639.OverlayValues[417] = d417
			ps639.OverlayValues[418] = d418
			ps639.OverlayValues[419] = d419
			ps639.OverlayValues[421] = d421
			ps639.OverlayValues[422] = d422
			ps639.OverlayValues[423] = d423
			ps639.OverlayValues[425] = d425
			ps639.OverlayValues[426] = d426
			ps639.OverlayValues[427] = d427
			ps639.OverlayValues[428] = d428
			ps639.OverlayValues[429] = d429
			ps639.OverlayValues[430] = d430
			ps639.OverlayValues[431] = d431
			ps639.OverlayValues[432] = d432
			ps639.OverlayValues[433] = d433
			ps639.OverlayValues[434] = d434
			ps639.OverlayValues[435] = d435
			ps639.OverlayValues[436] = d436
			ps639.OverlayValues[437] = d437
			ps639.OverlayValues[438] = d438
			ps639.OverlayValues[439] = d439
			ps639.OverlayValues[440] = d440
			ps639.OverlayValues[441] = d441
			ps639.OverlayValues[442] = d442
			ps639.OverlayValues[443] = d443
			ps639.OverlayValues[444] = d444
			ps639.OverlayValues[445] = d445
			ps639.OverlayValues[446] = d446
			ps639.OverlayValues[447] = d447
			ps639.OverlayValues[448] = d448
			ps639.OverlayValues[449] = d449
			ps639.OverlayValues[450] = d450
			ps639.OverlayValues[451] = d451
			ps639.OverlayValues[452] = d452
			ps639.OverlayValues[453] = d453
			ps639.OverlayValues[454] = d454
			ps639.OverlayValues[455] = d455
			ps639.OverlayValues[456] = d456
			ps639.OverlayValues[457] = d457
			ps639.OverlayValues[458] = d458
			ps639.OverlayValues[459] = d459
			ps639.OverlayValues[626] = d626
			ps639.OverlayValues[627] = d627
			ps639.OverlayValues[628] = d628
			ps639.OverlayValues[630] = d630
			ps639.OverlayValues[631] = d631
			ps639.OverlayValues[632] = d632
			ps639.OverlayValues[633] = d633
			ps639.OverlayValues[634] = d634
			ps639.OverlayValues[635] = d635
			ps639.OverlayValues[636] = d636
			ps639.OverlayValues[638] = d638
			return bbs[10].RenderPS(ps639)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d640 := ps.PhiValues[0]
				ctx.EnsureDesc(&d640)
				ctx.EmitStoreToStack(d640, int32(bbs[8].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d641 := ps.PhiValues[1]
				ctx.EnsureDesc(&d641)
				ctx.EmitStoreToStack(d641, int32(bbs[8].PhiBase)+int32(16))
			}
			ps.General = true
			return bbs[8].RenderPS(ps)
		}
		lbl38 := ctx.ReserveLabel()
		lbl39 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d634.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl38)
		ctx.EmitJmp(lbl39)
		ctx.MarkLabel(lbl38)
		ctx.SyncDesc(&d8)
		if d8.Loc == scm.LocReg {
			ctx.ProtectReg(d8.Reg)
		} else if d8.Loc == scm.LocRegPair {
			ctx.ProtectReg(d8.Reg)
			ctx.ProtectReg(d8.Reg2)
		}
		d642 = d8
		if d642.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d642)
		d643 = d642
		if d643.Loc == scm.LocImm {
			d643 = scm.JITValueDesc{Loc: scm.LocImm, Type: d643.Type, Imm: scm.NewInt(int64(uint64(d643.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d643.Reg, 32)
			ctx.EmitShrRegImm8(d643.Reg, 32)
		}
		ctx.EmitStoreToStack(d643, int32(bbs[2].PhiBase)+int32(0))
		if d8.Loc == scm.LocReg {
			ctx.UnprotectReg(d8.Reg)
		} else if d8.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d8.Reg)
			ctx.UnprotectReg(d8.Reg2)
		}
		ctx.EmitJmp(lbl3)
		ctx.MarkLabel(lbl39)
		ctx.EmitJmp(lbl11)
		ps644 := scm.PhiState{General: true}
		ps644.OverlayValues = make([]scm.JITValueDesc, 644)
		ps644.OverlayValues[1] = d1
		ps644.OverlayValues[2] = d2
		ps644.OverlayValues[3] = d3
		ps644.OverlayValues[4] = d4
		ps644.OverlayValues[5] = d5
		ps644.OverlayValues[6] = d6
		ps644.OverlayValues[7] = d7
		ps644.OverlayValues[8] = d8
		ps644.OverlayValues[9] = d9
		ps644.OverlayValues[10] = d10
		ps644.OverlayValues[11] = d11
		ps644.OverlayValues[12] = d12
		ps644.OverlayValues[13] = d13
		ps644.OverlayValues[14] = d14
		ps644.OverlayValues[15] = d15
		ps644.OverlayValues[17] = d17
		ps644.OverlayValues[18] = d18
		ps644.OverlayValues[19] = d19
		ps644.OverlayValues[20] = d20
		ps644.OverlayValues[21] = d21
		ps644.OverlayValues[22] = d22
		ps644.OverlayValues[24] = d24
		ps644.OverlayValues[25] = d25
		ps644.OverlayValues[26] = d26
		ps644.OverlayValues[27] = d27
		ps644.OverlayValues[28] = d28
		ps644.OverlayValues[29] = d29
		ps644.OverlayValues[30] = d30
		ps644.OverlayValues[31] = d31
		ps644.OverlayValues[32] = d32
		ps644.OverlayValues[33] = d33
		ps644.OverlayValues[34] = d34
		ps644.OverlayValues[35] = d35
		ps644.OverlayValues[36] = d36
		ps644.OverlayValues[37] = d37
		ps644.OverlayValues[38] = d38
		ps644.OverlayValues[39] = d39
		ps644.OverlayValues[40] = d40
		ps644.OverlayValues[41] = d41
		ps644.OverlayValues[42] = d42
		ps644.OverlayValues[43] = d43
		ps644.OverlayValues[44] = d44
		ps644.OverlayValues[45] = d45
		ps644.OverlayValues[46] = d46
		ps644.OverlayValues[47] = d47
		ps644.OverlayValues[48] = d48
		ps644.OverlayValues[49] = d49
		ps644.OverlayValues[50] = d50
		ps644.OverlayValues[51] = d51
		ps644.OverlayValues[52] = d52
		ps644.OverlayValues[53] = d53
		ps644.OverlayValues[54] = d54
		ps644.OverlayValues[55] = d55
		ps644.OverlayValues[56] = d56
		ps644.OverlayValues[57] = d57
		ps644.OverlayValues[58] = d58
		ps644.OverlayValues[59] = d59
		ps644.OverlayValues[62] = d62
		ps644.OverlayValues[63] = d63
		ps644.OverlayValues[64] = d64
		ps644.OverlayValues[128] = d128
		ps644.OverlayValues[129] = d129
		ps644.OverlayValues[130] = d130
		ps644.OverlayValues[132] = d132
		ps644.OverlayValues[133] = d133
		ps644.OverlayValues[134] = d134
		ps644.OverlayValues[135] = d135
		ps644.OverlayValues[136] = d136
		ps644.OverlayValues[137] = d137
		ps644.OverlayValues[138] = d138
		ps644.OverlayValues[139] = d139
		ps644.OverlayValues[140] = d140
		ps644.OverlayValues[141] = d141
		ps644.OverlayValues[142] = d142
		ps644.OverlayValues[143] = d143
		ps644.OverlayValues[144] = d144
		ps644.OverlayValues[145] = d145
		ps644.OverlayValues[146] = d146
		ps644.OverlayValues[147] = d147
		ps644.OverlayValues[148] = d148
		ps644.OverlayValues[149] = d149
		ps644.OverlayValues[150] = d150
		ps644.OverlayValues[151] = d151
		ps644.OverlayValues[152] = d152
		ps644.OverlayValues[153] = d153
		ps644.OverlayValues[154] = d154
		ps644.OverlayValues[155] = d155
		ps644.OverlayValues[156] = d156
		ps644.OverlayValues[157] = d157
		ps644.OverlayValues[158] = d158
		ps644.OverlayValues[159] = d159
		ps644.OverlayValues[160] = d160
		ps644.OverlayValues[161] = d161
		ps644.OverlayValues[162] = d162
		ps644.OverlayValues[163] = d163
		ps644.OverlayValues[164] = d164
		ps644.OverlayValues[165] = d165
		ps644.OverlayValues[166] = d166
		ps644.OverlayValues[169] = d169
		ps644.OverlayValues[272] = d272
		ps644.OverlayValues[273] = d273
		ps644.OverlayValues[274] = d274
		ps644.OverlayValues[275] = d275
		ps644.OverlayValues[277] = d277
		ps644.OverlayValues[278] = d278
		ps644.OverlayValues[279] = d279
		ps644.OverlayValues[280] = d280
		ps644.OverlayValues[281] = d281
		ps644.OverlayValues[282] = d282
		ps644.OverlayValues[283] = d283
		ps644.OverlayValues[284] = d284
		ps644.OverlayValues[286] = d286
		ps644.OverlayValues[288] = d288
		ps644.OverlayValues[289] = d289
		ps644.OverlayValues[290] = d290
		ps644.OverlayValues[291] = d291
		ps644.OverlayValues[292] = d292
		ps644.OverlayValues[295] = d295
		ps644.OverlayValues[415] = d415
		ps644.OverlayValues[416] = d416
		ps644.OverlayValues[417] = d417
		ps644.OverlayValues[418] = d418
		ps644.OverlayValues[419] = d419
		ps644.OverlayValues[421] = d421
		ps644.OverlayValues[422] = d422
		ps644.OverlayValues[423] = d423
		ps644.OverlayValues[425] = d425
		ps644.OverlayValues[426] = d426
		ps644.OverlayValues[427] = d427
		ps644.OverlayValues[428] = d428
		ps644.OverlayValues[429] = d429
		ps644.OverlayValues[430] = d430
		ps644.OverlayValues[431] = d431
		ps644.OverlayValues[432] = d432
		ps644.OverlayValues[433] = d433
		ps644.OverlayValues[434] = d434
		ps644.OverlayValues[435] = d435
		ps644.OverlayValues[436] = d436
		ps644.OverlayValues[437] = d437
		ps644.OverlayValues[438] = d438
		ps644.OverlayValues[439] = d439
		ps644.OverlayValues[440] = d440
		ps644.OverlayValues[441] = d441
		ps644.OverlayValues[442] = d442
		ps644.OverlayValues[443] = d443
		ps644.OverlayValues[444] = d444
		ps644.OverlayValues[445] = d445
		ps644.OverlayValues[446] = d446
		ps644.OverlayValues[447] = d447
		ps644.OverlayValues[448] = d448
		ps644.OverlayValues[449] = d449
		ps644.OverlayValues[450] = d450
		ps644.OverlayValues[451] = d451
		ps644.OverlayValues[452] = d452
		ps644.OverlayValues[453] = d453
		ps644.OverlayValues[454] = d454
		ps644.OverlayValues[455] = d455
		ps644.OverlayValues[456] = d456
		ps644.OverlayValues[457] = d457
		ps644.OverlayValues[458] = d458
		ps644.OverlayValues[459] = d459
		ps644.OverlayValues[626] = d626
		ps644.OverlayValues[627] = d627
		ps644.OverlayValues[628] = d628
		ps644.OverlayValues[630] = d630
		ps644.OverlayValues[631] = d631
		ps644.OverlayValues[632] = d632
		ps644.OverlayValues[633] = d633
		ps644.OverlayValues[634] = d634
		ps644.OverlayValues[635] = d635
		ps644.OverlayValues[636] = d636
		ps644.OverlayValues[638] = d638
		ps644.OverlayValues[640] = d640
		ps644.OverlayValues[641] = d641
		ps644.OverlayValues[642] = d642
		ps644.OverlayValues[643] = d643
		ps644.PhiValues = make([]scm.JITValueDesc, 1)
		d646 = d8
		ps644.PhiValues[0] = d646
		ps645 := scm.PhiState{General: true}
		ps645.OverlayValues = make([]scm.JITValueDesc, 647)
		ps645.OverlayValues[1] = d1
		ps645.OverlayValues[2] = d2
		ps645.OverlayValues[3] = d3
		ps645.OverlayValues[4] = d4
		ps645.OverlayValues[5] = d5
		ps645.OverlayValues[6] = d6
		ps645.OverlayValues[7] = d7
		ps645.OverlayValues[8] = d8
		ps645.OverlayValues[9] = d9
		ps645.OverlayValues[10] = d10
		ps645.OverlayValues[11] = d11
		ps645.OverlayValues[12] = d12
		ps645.OverlayValues[13] = d13
		ps645.OverlayValues[14] = d14
		ps645.OverlayValues[15] = d15
		ps645.OverlayValues[17] = d17
		ps645.OverlayValues[18] = d18
		ps645.OverlayValues[19] = d19
		ps645.OverlayValues[20] = d20
		ps645.OverlayValues[21] = d21
		ps645.OverlayValues[22] = d22
		ps645.OverlayValues[24] = d24
		ps645.OverlayValues[25] = d25
		ps645.OverlayValues[26] = d26
		ps645.OverlayValues[27] = d27
		ps645.OverlayValues[28] = d28
		ps645.OverlayValues[29] = d29
		ps645.OverlayValues[30] = d30
		ps645.OverlayValues[31] = d31
		ps645.OverlayValues[32] = d32
		ps645.OverlayValues[33] = d33
		ps645.OverlayValues[34] = d34
		ps645.OverlayValues[35] = d35
		ps645.OverlayValues[36] = d36
		ps645.OverlayValues[37] = d37
		ps645.OverlayValues[38] = d38
		ps645.OverlayValues[39] = d39
		ps645.OverlayValues[40] = d40
		ps645.OverlayValues[41] = d41
		ps645.OverlayValues[42] = d42
		ps645.OverlayValues[43] = d43
		ps645.OverlayValues[44] = d44
		ps645.OverlayValues[45] = d45
		ps645.OverlayValues[46] = d46
		ps645.OverlayValues[47] = d47
		ps645.OverlayValues[48] = d48
		ps645.OverlayValues[49] = d49
		ps645.OverlayValues[50] = d50
		ps645.OverlayValues[51] = d51
		ps645.OverlayValues[52] = d52
		ps645.OverlayValues[53] = d53
		ps645.OverlayValues[54] = d54
		ps645.OverlayValues[55] = d55
		ps645.OverlayValues[56] = d56
		ps645.OverlayValues[57] = d57
		ps645.OverlayValues[58] = d58
		ps645.OverlayValues[59] = d59
		ps645.OverlayValues[62] = d62
		ps645.OverlayValues[63] = d63
		ps645.OverlayValues[64] = d64
		ps645.OverlayValues[128] = d128
		ps645.OverlayValues[129] = d129
		ps645.OverlayValues[130] = d130
		ps645.OverlayValues[132] = d132
		ps645.OverlayValues[133] = d133
		ps645.OverlayValues[134] = d134
		ps645.OverlayValues[135] = d135
		ps645.OverlayValues[136] = d136
		ps645.OverlayValues[137] = d137
		ps645.OverlayValues[138] = d138
		ps645.OverlayValues[139] = d139
		ps645.OverlayValues[140] = d140
		ps645.OverlayValues[141] = d141
		ps645.OverlayValues[142] = d142
		ps645.OverlayValues[143] = d143
		ps645.OverlayValues[144] = d144
		ps645.OverlayValues[145] = d145
		ps645.OverlayValues[146] = d146
		ps645.OverlayValues[147] = d147
		ps645.OverlayValues[148] = d148
		ps645.OverlayValues[149] = d149
		ps645.OverlayValues[150] = d150
		ps645.OverlayValues[151] = d151
		ps645.OverlayValues[152] = d152
		ps645.OverlayValues[153] = d153
		ps645.OverlayValues[154] = d154
		ps645.OverlayValues[155] = d155
		ps645.OverlayValues[156] = d156
		ps645.OverlayValues[157] = d157
		ps645.OverlayValues[158] = d158
		ps645.OverlayValues[159] = d159
		ps645.OverlayValues[160] = d160
		ps645.OverlayValues[161] = d161
		ps645.OverlayValues[162] = d162
		ps645.OverlayValues[163] = d163
		ps645.OverlayValues[164] = d164
		ps645.OverlayValues[165] = d165
		ps645.OverlayValues[166] = d166
		ps645.OverlayValues[169] = d169
		ps645.OverlayValues[272] = d272
		ps645.OverlayValues[273] = d273
		ps645.OverlayValues[274] = d274
		ps645.OverlayValues[275] = d275
		ps645.OverlayValues[277] = d277
		ps645.OverlayValues[278] = d278
		ps645.OverlayValues[279] = d279
		ps645.OverlayValues[280] = d280
		ps645.OverlayValues[281] = d281
		ps645.OverlayValues[282] = d282
		ps645.OverlayValues[283] = d283
		ps645.OverlayValues[284] = d284
		ps645.OverlayValues[286] = d286
		ps645.OverlayValues[288] = d288
		ps645.OverlayValues[289] = d289
		ps645.OverlayValues[290] = d290
		ps645.OverlayValues[291] = d291
		ps645.OverlayValues[292] = d292
		ps645.OverlayValues[295] = d295
		ps645.OverlayValues[415] = d415
		ps645.OverlayValues[416] = d416
		ps645.OverlayValues[417] = d417
		ps645.OverlayValues[418] = d418
		ps645.OverlayValues[419] = d419
		ps645.OverlayValues[421] = d421
		ps645.OverlayValues[422] = d422
		ps645.OverlayValues[423] = d423
		ps645.OverlayValues[425] = d425
		ps645.OverlayValues[426] = d426
		ps645.OverlayValues[427] = d427
		ps645.OverlayValues[428] = d428
		ps645.OverlayValues[429] = d429
		ps645.OverlayValues[430] = d430
		ps645.OverlayValues[431] = d431
		ps645.OverlayValues[432] = d432
		ps645.OverlayValues[433] = d433
		ps645.OverlayValues[434] = d434
		ps645.OverlayValues[435] = d435
		ps645.OverlayValues[436] = d436
		ps645.OverlayValues[437] = d437
		ps645.OverlayValues[438] = d438
		ps645.OverlayValues[439] = d439
		ps645.OverlayValues[440] = d440
		ps645.OverlayValues[441] = d441
		ps645.OverlayValues[442] = d442
		ps645.OverlayValues[443] = d443
		ps645.OverlayValues[444] = d444
		ps645.OverlayValues[445] = d445
		ps645.OverlayValues[446] = d446
		ps645.OverlayValues[447] = d447
		ps645.OverlayValues[448] = d448
		ps645.OverlayValues[449] = d449
		ps645.OverlayValues[450] = d450
		ps645.OverlayValues[451] = d451
		ps645.OverlayValues[452] = d452
		ps645.OverlayValues[453] = d453
		ps645.OverlayValues[454] = d454
		ps645.OverlayValues[455] = d455
		ps645.OverlayValues[456] = d456
		ps645.OverlayValues[457] = d457
		ps645.OverlayValues[458] = d458
		ps645.OverlayValues[459] = d459
		ps645.OverlayValues[626] = d626
		ps645.OverlayValues[627] = d627
		ps645.OverlayValues[628] = d628
		ps645.OverlayValues[630] = d630
		ps645.OverlayValues[631] = d631
		ps645.OverlayValues[632] = d632
		ps645.OverlayValues[633] = d633
		ps645.OverlayValues[634] = d634
		ps645.OverlayValues[635] = d635
		ps645.OverlayValues[636] = d636
		ps645.OverlayValues[638] = d638
		ps645.OverlayValues[640] = d640
		ps645.OverlayValues[641] = d641
		ps645.OverlayValues[642] = d642
		ps645.OverlayValues[643] = d643
		ps645.OverlayValues[646] = d646
		snap647 := d1
		snap648 := d2
		snap649 := d3
		snap650 := d4
		snap651 := d5
		snap652 := d6
		snap653 := d7
		snap654 := d8
		snap655 := d9
		snap656 := d10
		snap657 := d11
		snap658 := d12
		snap659 := d13
		snap660 := d14
		snap661 := d15
		snap662 := d17
		snap663 := d18
		snap664 := d19
		snap665 := d20
		snap666 := d21
		snap667 := d22
		snap668 := d24
		snap669 := d25
		snap670 := d26
		snap671 := d27
		snap672 := d28
		snap673 := d29
		snap674 := d30
		snap675 := d31
		snap676 := d32
		snap677 := d33
		snap678 := d34
		snap679 := d35
		snap680 := d36
		snap681 := d37
		snap682 := d38
		snap683 := d39
		snap684 := d40
		snap685 := d41
		snap686 := d42
		snap687 := d43
		snap688 := d44
		snap689 := d45
		snap690 := d46
		snap691 := d47
		snap692 := d48
		snap693 := d49
		snap694 := d50
		snap695 := d51
		snap696 := d52
		snap697 := d53
		snap698 := d54
		snap699 := d55
		snap700 := d56
		snap701 := d57
		snap702 := d58
		snap703 := d59
		snap704 := d62
		snap705 := d63
		snap706 := d64
		snap707 := d128
		snap708 := d129
		snap709 := d130
		snap710 := d132
		snap711 := d133
		snap712 := d134
		snap713 := d135
		snap714 := d136
		snap715 := d137
		snap716 := d138
		snap717 := d139
		snap718 := d140
		snap719 := d141
		snap720 := d142
		snap721 := d143
		snap722 := d144
		snap723 := d145
		snap724 := d146
		snap725 := d147
		snap726 := d148
		snap727 := d149
		snap728 := d150
		snap729 := d151
		snap730 := d152
		snap731 := d153
		snap732 := d154
		snap733 := d155
		snap734 := d156
		snap735 := d157
		snap736 := d158
		snap737 := d159
		snap738 := d160
		snap739 := d161
		snap740 := d162
		snap741 := d163
		snap742 := d164
		snap743 := d165
		snap744 := d166
		snap745 := d169
		snap746 := d272
		snap747 := d273
		snap748 := d274
		snap749 := d275
		snap750 := d277
		snap751 := d278
		snap752 := d279
		snap753 := d280
		snap754 := d281
		snap755 := d282
		snap756 := d283
		snap757 := d284
		snap758 := d286
		snap759 := d288
		snap760 := d289
		snap761 := d290
		snap762 := d291
		snap763 := d292
		snap764 := d295
		snap765 := d415
		snap766 := d416
		snap767 := d417
		snap768 := d418
		snap769 := d419
		snap770 := d421
		snap771 := d422
		snap772 := d423
		snap773 := d425
		snap774 := d426
		snap775 := d427
		snap776 := d428
		snap777 := d429
		snap778 := d430
		snap779 := d431
		snap780 := d432
		snap781 := d433
		snap782 := d434
		snap783 := d435
		snap784 := d436
		snap785 := d437
		snap786 := d438
		snap787 := d439
		snap788 := d440
		snap789 := d441
		snap790 := d442
		snap791 := d443
		snap792 := d444
		snap793 := d445
		snap794 := d446
		snap795 := d447
		snap796 := d448
		snap797 := d449
		snap798 := d450
		snap799 := d451
		snap800 := d452
		snap801 := d453
		snap802 := d454
		snap803 := d455
		snap804 := d456
		snap805 := d457
		snap806 := d458
		snap807 := d459
		snap808 := d626
		snap809 := d627
		snap810 := d628
		snap811 := d630
		snap812 := d631
		snap813 := d632
		snap814 := d633
		snap815 := d634
		snap816 := d635
		snap817 := d636
		snap818 := d638
		snap819 := d640
		snap820 := d641
		snap821 := d642
		snap822 := d643
		snap823 := d646
		alloc824 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps644)
		}
		ctx.RestoreAllocState(alloc824)
		d1 = snap647
		d2 = snap648
		d3 = snap649
		d4 = snap650
		d5 = snap651
		d6 = snap652
		d7 = snap653
		d8 = snap654
		d9 = snap655
		d10 = snap656
		d11 = snap657
		d12 = snap658
		d13 = snap659
		d14 = snap660
		d15 = snap661
		d17 = snap662
		d18 = snap663
		d19 = snap664
		d20 = snap665
		d21 = snap666
		d22 = snap667
		d24 = snap668
		d25 = snap669
		d26 = snap670
		d27 = snap671
		d28 = snap672
		d29 = snap673
		d30 = snap674
		d31 = snap675
		d32 = snap676
		d33 = snap677
		d34 = snap678
		d35 = snap679
		d36 = snap680
		d37 = snap681
		d38 = snap682
		d39 = snap683
		d40 = snap684
		d41 = snap685
		d42 = snap686
		d43 = snap687
		d44 = snap688
		d45 = snap689
		d46 = snap690
		d47 = snap691
		d48 = snap692
		d49 = snap693
		d50 = snap694
		d51 = snap695
		d52 = snap696
		d53 = snap697
		d54 = snap698
		d55 = snap699
		d56 = snap700
		d57 = snap701
		d58 = snap702
		d59 = snap703
		d62 = snap704
		d63 = snap705
		d64 = snap706
		d128 = snap707
		d129 = snap708
		d130 = snap709
		d132 = snap710
		d133 = snap711
		d134 = snap712
		d135 = snap713
		d136 = snap714
		d137 = snap715
		d138 = snap716
		d139 = snap717
		d140 = snap718
		d141 = snap719
		d142 = snap720
		d143 = snap721
		d144 = snap722
		d145 = snap723
		d146 = snap724
		d147 = snap725
		d148 = snap726
		d149 = snap727
		d150 = snap728
		d151 = snap729
		d152 = snap730
		d153 = snap731
		d154 = snap732
		d155 = snap733
		d156 = snap734
		d157 = snap735
		d158 = snap736
		d159 = snap737
		d160 = snap738
		d161 = snap739
		d162 = snap740
		d163 = snap741
		d164 = snap742
		d165 = snap743
		d166 = snap744
		d169 = snap745
		d272 = snap746
		d273 = snap747
		d274 = snap748
		d275 = snap749
		d277 = snap750
		d278 = snap751
		d279 = snap752
		d280 = snap753
		d281 = snap754
		d282 = snap755
		d283 = snap756
		d284 = snap757
		d286 = snap758
		d288 = snap759
		d289 = snap760
		d290 = snap761
		d291 = snap762
		d292 = snap763
		d295 = snap764
		d415 = snap765
		d416 = snap766
		d417 = snap767
		d418 = snap768
		d419 = snap769
		d421 = snap770
		d422 = snap771
		d423 = snap772
		d425 = snap773
		d426 = snap774
		d427 = snap775
		d428 = snap776
		d429 = snap777
		d430 = snap778
		d431 = snap779
		d432 = snap780
		d433 = snap781
		d434 = snap782
		d435 = snap783
		d436 = snap784
		d437 = snap785
		d438 = snap786
		d439 = snap787
		d440 = snap788
		d441 = snap789
		d442 = snap790
		d443 = snap791
		d444 = snap792
		d445 = snap793
		d446 = snap794
		d447 = snap795
		d448 = snap796
		d449 = snap797
		d450 = snap798
		d451 = snap799
		d452 = snap800
		d453 = snap801
		d454 = snap802
		d455 = snap803
		d456 = snap804
		d457 = snap805
		d458 = snap806
		d459 = snap807
		d626 = snap808
		d627 = snap809
		d628 = snap810
		d630 = snap811
		d631 = snap812
		d632 = snap813
		d633 = snap814
		d634 = snap815
		d635 = snap816
		d636 = snap817
		d638 = snap818
		d640 = snap819
		d641 = snap820
		d642 = snap821
		d643 = snap822
		d646 = snap823
		if !bbs[10].Rendered {
			return bbs[10].RenderPS(ps645)
		}
		return result
		ctx.FreeDesc(&d633)
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
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
		}
		if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
			d58 = ps.OverlayValues[58]
		}
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
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
		if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
			d128 = ps.OverlayValues[128]
		}
		if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
			d129 = ps.OverlayValues[129]
		}
		if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
			d130 = ps.OverlayValues[130]
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
		if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != scm.LocNone {
			d148 = ps.OverlayValues[148]
		}
		if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != scm.LocNone {
			d149 = ps.OverlayValues[149]
		}
		if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
			d150 = ps.OverlayValues[150]
		}
		if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != scm.LocNone {
			d151 = ps.OverlayValues[151]
		}
		if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != scm.LocNone {
			d152 = ps.OverlayValues[152]
		}
		if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
			d153 = ps.OverlayValues[153]
		}
		if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
			d154 = ps.OverlayValues[154]
		}
		if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != scm.LocNone {
			d155 = ps.OverlayValues[155]
		}
		if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != scm.LocNone {
			d156 = ps.OverlayValues[156]
		}
		if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != scm.LocNone {
			d157 = ps.OverlayValues[157]
		}
		if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != scm.LocNone {
			d158 = ps.OverlayValues[158]
		}
		if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != scm.LocNone {
			d159 = ps.OverlayValues[159]
		}
		if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != scm.LocNone {
			d160 = ps.OverlayValues[160]
		}
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != scm.LocNone {
			d161 = ps.OverlayValues[161]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != scm.LocNone {
			d162 = ps.OverlayValues[162]
		}
		if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != scm.LocNone {
			d163 = ps.OverlayValues[163]
		}
		if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != scm.LocNone {
			d164 = ps.OverlayValues[164]
		}
		if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != scm.LocNone {
			d165 = ps.OverlayValues[165]
		}
		if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != scm.LocNone {
			d166 = ps.OverlayValues[166]
		}
		if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != scm.LocNone {
			d169 = ps.OverlayValues[169]
		}
		if len(ps.OverlayValues) > 272 && ps.OverlayValues[272].Loc != scm.LocNone {
			d272 = ps.OverlayValues[272]
		}
		if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != scm.LocNone {
			d273 = ps.OverlayValues[273]
		}
		if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != scm.LocNone {
			d274 = ps.OverlayValues[274]
		}
		if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != scm.LocNone {
			d275 = ps.OverlayValues[275]
		}
		if len(ps.OverlayValues) > 277 && ps.OverlayValues[277].Loc != scm.LocNone {
			d277 = ps.OverlayValues[277]
		}
		if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != scm.LocNone {
			d278 = ps.OverlayValues[278]
		}
		if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != scm.LocNone {
			d279 = ps.OverlayValues[279]
		}
		if len(ps.OverlayValues) > 280 && ps.OverlayValues[280].Loc != scm.LocNone {
			d280 = ps.OverlayValues[280]
		}
		if len(ps.OverlayValues) > 281 && ps.OverlayValues[281].Loc != scm.LocNone {
			d281 = ps.OverlayValues[281]
		}
		if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != scm.LocNone {
			d282 = ps.OverlayValues[282]
		}
		if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != scm.LocNone {
			d283 = ps.OverlayValues[283]
		}
		if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != scm.LocNone {
			d284 = ps.OverlayValues[284]
		}
		if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != scm.LocNone {
			d286 = ps.OverlayValues[286]
		}
		if len(ps.OverlayValues) > 288 && ps.OverlayValues[288].Loc != scm.LocNone {
			d288 = ps.OverlayValues[288]
		}
		if len(ps.OverlayValues) > 289 && ps.OverlayValues[289].Loc != scm.LocNone {
			d289 = ps.OverlayValues[289]
		}
		if len(ps.OverlayValues) > 290 && ps.OverlayValues[290].Loc != scm.LocNone {
			d290 = ps.OverlayValues[290]
		}
		if len(ps.OverlayValues) > 291 && ps.OverlayValues[291].Loc != scm.LocNone {
			d291 = ps.OverlayValues[291]
		}
		if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != scm.LocNone {
			d292 = ps.OverlayValues[292]
		}
		if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
			d295 = ps.OverlayValues[295]
		}
		if len(ps.OverlayValues) > 415 && ps.OverlayValues[415].Loc != scm.LocNone {
			d415 = ps.OverlayValues[415]
		}
		if len(ps.OverlayValues) > 416 && ps.OverlayValues[416].Loc != scm.LocNone {
			d416 = ps.OverlayValues[416]
		}
		if len(ps.OverlayValues) > 417 && ps.OverlayValues[417].Loc != scm.LocNone {
			d417 = ps.OverlayValues[417]
		}
		if len(ps.OverlayValues) > 418 && ps.OverlayValues[418].Loc != scm.LocNone {
			d418 = ps.OverlayValues[418]
		}
		if len(ps.OverlayValues) > 419 && ps.OverlayValues[419].Loc != scm.LocNone {
			d419 = ps.OverlayValues[419]
		}
		if len(ps.OverlayValues) > 421 && ps.OverlayValues[421].Loc != scm.LocNone {
			d421 = ps.OverlayValues[421]
		}
		if len(ps.OverlayValues) > 422 && ps.OverlayValues[422].Loc != scm.LocNone {
			d422 = ps.OverlayValues[422]
		}
		if len(ps.OverlayValues) > 423 && ps.OverlayValues[423].Loc != scm.LocNone {
			d423 = ps.OverlayValues[423]
		}
		if len(ps.OverlayValues) > 425 && ps.OverlayValues[425].Loc != scm.LocNone {
			d425 = ps.OverlayValues[425]
		}
		if len(ps.OverlayValues) > 426 && ps.OverlayValues[426].Loc != scm.LocNone {
			d426 = ps.OverlayValues[426]
		}
		if len(ps.OverlayValues) > 427 && ps.OverlayValues[427].Loc != scm.LocNone {
			d427 = ps.OverlayValues[427]
		}
		if len(ps.OverlayValues) > 428 && ps.OverlayValues[428].Loc != scm.LocNone {
			d428 = ps.OverlayValues[428]
		}
		if len(ps.OverlayValues) > 429 && ps.OverlayValues[429].Loc != scm.LocNone {
			d429 = ps.OverlayValues[429]
		}
		if len(ps.OverlayValues) > 430 && ps.OverlayValues[430].Loc != scm.LocNone {
			d430 = ps.OverlayValues[430]
		}
		if len(ps.OverlayValues) > 431 && ps.OverlayValues[431].Loc != scm.LocNone {
			d431 = ps.OverlayValues[431]
		}
		if len(ps.OverlayValues) > 432 && ps.OverlayValues[432].Loc != scm.LocNone {
			d432 = ps.OverlayValues[432]
		}
		if len(ps.OverlayValues) > 433 && ps.OverlayValues[433].Loc != scm.LocNone {
			d433 = ps.OverlayValues[433]
		}
		if len(ps.OverlayValues) > 434 && ps.OverlayValues[434].Loc != scm.LocNone {
			d434 = ps.OverlayValues[434]
		}
		if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != scm.LocNone {
			d435 = ps.OverlayValues[435]
		}
		if len(ps.OverlayValues) > 436 && ps.OverlayValues[436].Loc != scm.LocNone {
			d436 = ps.OverlayValues[436]
		}
		if len(ps.OverlayValues) > 437 && ps.OverlayValues[437].Loc != scm.LocNone {
			d437 = ps.OverlayValues[437]
		}
		if len(ps.OverlayValues) > 438 && ps.OverlayValues[438].Loc != scm.LocNone {
			d438 = ps.OverlayValues[438]
		}
		if len(ps.OverlayValues) > 439 && ps.OverlayValues[439].Loc != scm.LocNone {
			d439 = ps.OverlayValues[439]
		}
		if len(ps.OverlayValues) > 440 && ps.OverlayValues[440].Loc != scm.LocNone {
			d440 = ps.OverlayValues[440]
		}
		if len(ps.OverlayValues) > 441 && ps.OverlayValues[441].Loc != scm.LocNone {
			d441 = ps.OverlayValues[441]
		}
		if len(ps.OverlayValues) > 442 && ps.OverlayValues[442].Loc != scm.LocNone {
			d442 = ps.OverlayValues[442]
		}
		if len(ps.OverlayValues) > 443 && ps.OverlayValues[443].Loc != scm.LocNone {
			d443 = ps.OverlayValues[443]
		}
		if len(ps.OverlayValues) > 444 && ps.OverlayValues[444].Loc != scm.LocNone {
			d444 = ps.OverlayValues[444]
		}
		if len(ps.OverlayValues) > 445 && ps.OverlayValues[445].Loc != scm.LocNone {
			d445 = ps.OverlayValues[445]
		}
		if len(ps.OverlayValues) > 446 && ps.OverlayValues[446].Loc != scm.LocNone {
			d446 = ps.OverlayValues[446]
		}
		if len(ps.OverlayValues) > 447 && ps.OverlayValues[447].Loc != scm.LocNone {
			d447 = ps.OverlayValues[447]
		}
		if len(ps.OverlayValues) > 448 && ps.OverlayValues[448].Loc != scm.LocNone {
			d448 = ps.OverlayValues[448]
		}
		if len(ps.OverlayValues) > 449 && ps.OverlayValues[449].Loc != scm.LocNone {
			d449 = ps.OverlayValues[449]
		}
		if len(ps.OverlayValues) > 450 && ps.OverlayValues[450].Loc != scm.LocNone {
			d450 = ps.OverlayValues[450]
		}
		if len(ps.OverlayValues) > 451 && ps.OverlayValues[451].Loc != scm.LocNone {
			d451 = ps.OverlayValues[451]
		}
		if len(ps.OverlayValues) > 452 && ps.OverlayValues[452].Loc != scm.LocNone {
			d452 = ps.OverlayValues[452]
		}
		if len(ps.OverlayValues) > 453 && ps.OverlayValues[453].Loc != scm.LocNone {
			d453 = ps.OverlayValues[453]
		}
		if len(ps.OverlayValues) > 454 && ps.OverlayValues[454].Loc != scm.LocNone {
			d454 = ps.OverlayValues[454]
		}
		if len(ps.OverlayValues) > 455 && ps.OverlayValues[455].Loc != scm.LocNone {
			d455 = ps.OverlayValues[455]
		}
		if len(ps.OverlayValues) > 456 && ps.OverlayValues[456].Loc != scm.LocNone {
			d456 = ps.OverlayValues[456]
		}
		if len(ps.OverlayValues) > 457 && ps.OverlayValues[457].Loc != scm.LocNone {
			d457 = ps.OverlayValues[457]
		}
		if len(ps.OverlayValues) > 458 && ps.OverlayValues[458].Loc != scm.LocNone {
			d458 = ps.OverlayValues[458]
		}
		if len(ps.OverlayValues) > 459 && ps.OverlayValues[459].Loc != scm.LocNone {
			d459 = ps.OverlayValues[459]
		}
		if len(ps.OverlayValues) > 626 && ps.OverlayValues[626].Loc != scm.LocNone {
			d626 = ps.OverlayValues[626]
		}
		if len(ps.OverlayValues) > 627 && ps.OverlayValues[627].Loc != scm.LocNone {
			d627 = ps.OverlayValues[627]
		}
		if len(ps.OverlayValues) > 628 && ps.OverlayValues[628].Loc != scm.LocNone {
			d628 = ps.OverlayValues[628]
		}
		if len(ps.OverlayValues) > 630 && ps.OverlayValues[630].Loc != scm.LocNone {
			d630 = ps.OverlayValues[630]
		}
		if len(ps.OverlayValues) > 631 && ps.OverlayValues[631].Loc != scm.LocNone {
			d631 = ps.OverlayValues[631]
		}
		if len(ps.OverlayValues) > 632 && ps.OverlayValues[632].Loc != scm.LocNone {
			d632 = ps.OverlayValues[632]
		}
		if len(ps.OverlayValues) > 633 && ps.OverlayValues[633].Loc != scm.LocNone {
			d633 = ps.OverlayValues[633]
		}
		if len(ps.OverlayValues) > 634 && ps.OverlayValues[634].Loc != scm.LocNone {
			d634 = ps.OverlayValues[634]
		}
		if len(ps.OverlayValues) > 635 && ps.OverlayValues[635].Loc != scm.LocNone {
			d635 = ps.OverlayValues[635]
		}
		if len(ps.OverlayValues) > 636 && ps.OverlayValues[636].Loc != scm.LocNone {
			d636 = ps.OverlayValues[636]
		}
		if len(ps.OverlayValues) > 638 && ps.OverlayValues[638].Loc != scm.LocNone {
			d638 = ps.OverlayValues[638]
		}
		if len(ps.OverlayValues) > 640 && ps.OverlayValues[640].Loc != scm.LocNone {
			d640 = ps.OverlayValues[640]
		}
		if len(ps.OverlayValues) > 641 && ps.OverlayValues[641].Loc != scm.LocNone {
			d641 = ps.OverlayValues[641]
		}
		if len(ps.OverlayValues) > 642 && ps.OverlayValues[642].Loc != scm.LocNone {
			d642 = ps.OverlayValues[642]
		}
		if len(ps.OverlayValues) > 643 && ps.OverlayValues[643].Loc != scm.LocNone {
			d643 = ps.OverlayValues[643]
		}
		if len(ps.OverlayValues) > 646 && ps.OverlayValues[646].Loc != scm.LocNone {
			d646 = ps.OverlayValues[646]
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
			d825 = d5
			if d825.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d825)
			d826 = d825
			if d826.Loc == scm.LocImm {
				d826 = scm.JITValueDesc{Loc: scm.LocImm, Type: d826.Type, Imm: scm.NewInt(int64(uint64(d826.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d826.Reg, 32)
				ctx.EmitShrRegImm8(d826.Reg, 32)
			}
			ctx.EmitStoreToStack(d826, int32(bbs[8].PhiBase)+int32(0))
			d827 = d7
			if d827.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d827)
			d828 = d827
			if d828.Loc == scm.LocImm {
				d828 = scm.JITValueDesc{Loc: scm.LocImm, Type: d828.Type, Imm: scm.NewInt(int64(uint64(d828.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d828.Reg, 32)
				ctx.EmitShrRegImm8(d828.Reg, 32)
			}
			ctx.EmitStoreToStack(d828, int32(bbs[8].PhiBase)+int32(16))
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
		ps829 := scm.PhiState{General: ps.General}
		ps829.OverlayValues = make([]scm.JITValueDesc, 829)
		ps829.OverlayValues[1] = d1
		ps829.OverlayValues[2] = d2
		ps829.OverlayValues[3] = d3
		ps829.OverlayValues[4] = d4
		ps829.OverlayValues[5] = d5
		ps829.OverlayValues[6] = d6
		ps829.OverlayValues[7] = d7
		ps829.OverlayValues[8] = d8
		ps829.OverlayValues[9] = d9
		ps829.OverlayValues[10] = d10
		ps829.OverlayValues[11] = d11
		ps829.OverlayValues[12] = d12
		ps829.OverlayValues[13] = d13
		ps829.OverlayValues[14] = d14
		ps829.OverlayValues[15] = d15
		ps829.OverlayValues[17] = d17
		ps829.OverlayValues[18] = d18
		ps829.OverlayValues[19] = d19
		ps829.OverlayValues[20] = d20
		ps829.OverlayValues[21] = d21
		ps829.OverlayValues[22] = d22
		ps829.OverlayValues[24] = d24
		ps829.OverlayValues[25] = d25
		ps829.OverlayValues[26] = d26
		ps829.OverlayValues[27] = d27
		ps829.OverlayValues[28] = d28
		ps829.OverlayValues[29] = d29
		ps829.OverlayValues[30] = d30
		ps829.OverlayValues[31] = d31
		ps829.OverlayValues[32] = d32
		ps829.OverlayValues[33] = d33
		ps829.OverlayValues[34] = d34
		ps829.OverlayValues[35] = d35
		ps829.OverlayValues[36] = d36
		ps829.OverlayValues[37] = d37
		ps829.OverlayValues[38] = d38
		ps829.OverlayValues[39] = d39
		ps829.OverlayValues[40] = d40
		ps829.OverlayValues[41] = d41
		ps829.OverlayValues[42] = d42
		ps829.OverlayValues[43] = d43
		ps829.OverlayValues[44] = d44
		ps829.OverlayValues[45] = d45
		ps829.OverlayValues[46] = d46
		ps829.OverlayValues[47] = d47
		ps829.OverlayValues[48] = d48
		ps829.OverlayValues[49] = d49
		ps829.OverlayValues[50] = d50
		ps829.OverlayValues[51] = d51
		ps829.OverlayValues[52] = d52
		ps829.OverlayValues[53] = d53
		ps829.OverlayValues[54] = d54
		ps829.OverlayValues[55] = d55
		ps829.OverlayValues[56] = d56
		ps829.OverlayValues[57] = d57
		ps829.OverlayValues[58] = d58
		ps829.OverlayValues[59] = d59
		ps829.OverlayValues[62] = d62
		ps829.OverlayValues[63] = d63
		ps829.OverlayValues[64] = d64
		ps829.OverlayValues[128] = d128
		ps829.OverlayValues[129] = d129
		ps829.OverlayValues[130] = d130
		ps829.OverlayValues[132] = d132
		ps829.OverlayValues[133] = d133
		ps829.OverlayValues[134] = d134
		ps829.OverlayValues[135] = d135
		ps829.OverlayValues[136] = d136
		ps829.OverlayValues[137] = d137
		ps829.OverlayValues[138] = d138
		ps829.OverlayValues[139] = d139
		ps829.OverlayValues[140] = d140
		ps829.OverlayValues[141] = d141
		ps829.OverlayValues[142] = d142
		ps829.OverlayValues[143] = d143
		ps829.OverlayValues[144] = d144
		ps829.OverlayValues[145] = d145
		ps829.OverlayValues[146] = d146
		ps829.OverlayValues[147] = d147
		ps829.OverlayValues[148] = d148
		ps829.OverlayValues[149] = d149
		ps829.OverlayValues[150] = d150
		ps829.OverlayValues[151] = d151
		ps829.OverlayValues[152] = d152
		ps829.OverlayValues[153] = d153
		ps829.OverlayValues[154] = d154
		ps829.OverlayValues[155] = d155
		ps829.OverlayValues[156] = d156
		ps829.OverlayValues[157] = d157
		ps829.OverlayValues[158] = d158
		ps829.OverlayValues[159] = d159
		ps829.OverlayValues[160] = d160
		ps829.OverlayValues[161] = d161
		ps829.OverlayValues[162] = d162
		ps829.OverlayValues[163] = d163
		ps829.OverlayValues[164] = d164
		ps829.OverlayValues[165] = d165
		ps829.OverlayValues[166] = d166
		ps829.OverlayValues[169] = d169
		ps829.OverlayValues[272] = d272
		ps829.OverlayValues[273] = d273
		ps829.OverlayValues[274] = d274
		ps829.OverlayValues[275] = d275
		ps829.OverlayValues[277] = d277
		ps829.OverlayValues[278] = d278
		ps829.OverlayValues[279] = d279
		ps829.OverlayValues[280] = d280
		ps829.OverlayValues[281] = d281
		ps829.OverlayValues[282] = d282
		ps829.OverlayValues[283] = d283
		ps829.OverlayValues[284] = d284
		ps829.OverlayValues[286] = d286
		ps829.OverlayValues[288] = d288
		ps829.OverlayValues[289] = d289
		ps829.OverlayValues[290] = d290
		ps829.OverlayValues[291] = d291
		ps829.OverlayValues[292] = d292
		ps829.OverlayValues[295] = d295
		ps829.OverlayValues[415] = d415
		ps829.OverlayValues[416] = d416
		ps829.OverlayValues[417] = d417
		ps829.OverlayValues[418] = d418
		ps829.OverlayValues[419] = d419
		ps829.OverlayValues[421] = d421
		ps829.OverlayValues[422] = d422
		ps829.OverlayValues[423] = d423
		ps829.OverlayValues[425] = d425
		ps829.OverlayValues[426] = d426
		ps829.OverlayValues[427] = d427
		ps829.OverlayValues[428] = d428
		ps829.OverlayValues[429] = d429
		ps829.OverlayValues[430] = d430
		ps829.OverlayValues[431] = d431
		ps829.OverlayValues[432] = d432
		ps829.OverlayValues[433] = d433
		ps829.OverlayValues[434] = d434
		ps829.OverlayValues[435] = d435
		ps829.OverlayValues[436] = d436
		ps829.OverlayValues[437] = d437
		ps829.OverlayValues[438] = d438
		ps829.OverlayValues[439] = d439
		ps829.OverlayValues[440] = d440
		ps829.OverlayValues[441] = d441
		ps829.OverlayValues[442] = d442
		ps829.OverlayValues[443] = d443
		ps829.OverlayValues[444] = d444
		ps829.OverlayValues[445] = d445
		ps829.OverlayValues[446] = d446
		ps829.OverlayValues[447] = d447
		ps829.OverlayValues[448] = d448
		ps829.OverlayValues[449] = d449
		ps829.OverlayValues[450] = d450
		ps829.OverlayValues[451] = d451
		ps829.OverlayValues[452] = d452
		ps829.OverlayValues[453] = d453
		ps829.OverlayValues[454] = d454
		ps829.OverlayValues[455] = d455
		ps829.OverlayValues[456] = d456
		ps829.OverlayValues[457] = d457
		ps829.OverlayValues[458] = d458
		ps829.OverlayValues[459] = d459
		ps829.OverlayValues[626] = d626
		ps829.OverlayValues[627] = d627
		ps829.OverlayValues[628] = d628
		ps829.OverlayValues[630] = d630
		ps829.OverlayValues[631] = d631
		ps829.OverlayValues[632] = d632
		ps829.OverlayValues[633] = d633
		ps829.OverlayValues[634] = d634
		ps829.OverlayValues[635] = d635
		ps829.OverlayValues[636] = d636
		ps829.OverlayValues[638] = d638
		ps829.OverlayValues[640] = d640
		ps829.OverlayValues[641] = d641
		ps829.OverlayValues[642] = d642
		ps829.OverlayValues[643] = d643
		ps829.OverlayValues[646] = d646
		ps829.OverlayValues[825] = d825
		ps829.OverlayValues[826] = d826
		ps829.OverlayValues[827] = d827
		ps829.OverlayValues[828] = d828
		ps829.PhiValues = make([]scm.JITValueDesc, 2)
		d830 = d5
		ps829.PhiValues[0] = d830
		d831 = d7
		ps829.PhiValues[1] = d831
		if ps829.General && bbs[8].Rendered {
			ctx.EmitJmp(lbl9)
			return result
		}
		return bbs[8].RenderPS(ps829)
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
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
		}
		if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
			d58 = ps.OverlayValues[58]
		}
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
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
		if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
			d128 = ps.OverlayValues[128]
		}
		if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
			d129 = ps.OverlayValues[129]
		}
		if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
			d130 = ps.OverlayValues[130]
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
		if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != scm.LocNone {
			d148 = ps.OverlayValues[148]
		}
		if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != scm.LocNone {
			d149 = ps.OverlayValues[149]
		}
		if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
			d150 = ps.OverlayValues[150]
		}
		if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != scm.LocNone {
			d151 = ps.OverlayValues[151]
		}
		if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != scm.LocNone {
			d152 = ps.OverlayValues[152]
		}
		if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
			d153 = ps.OverlayValues[153]
		}
		if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
			d154 = ps.OverlayValues[154]
		}
		if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != scm.LocNone {
			d155 = ps.OverlayValues[155]
		}
		if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != scm.LocNone {
			d156 = ps.OverlayValues[156]
		}
		if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != scm.LocNone {
			d157 = ps.OverlayValues[157]
		}
		if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != scm.LocNone {
			d158 = ps.OverlayValues[158]
		}
		if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != scm.LocNone {
			d159 = ps.OverlayValues[159]
		}
		if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != scm.LocNone {
			d160 = ps.OverlayValues[160]
		}
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != scm.LocNone {
			d161 = ps.OverlayValues[161]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != scm.LocNone {
			d162 = ps.OverlayValues[162]
		}
		if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != scm.LocNone {
			d163 = ps.OverlayValues[163]
		}
		if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != scm.LocNone {
			d164 = ps.OverlayValues[164]
		}
		if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != scm.LocNone {
			d165 = ps.OverlayValues[165]
		}
		if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != scm.LocNone {
			d166 = ps.OverlayValues[166]
		}
		if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != scm.LocNone {
			d169 = ps.OverlayValues[169]
		}
		if len(ps.OverlayValues) > 272 && ps.OverlayValues[272].Loc != scm.LocNone {
			d272 = ps.OverlayValues[272]
		}
		if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != scm.LocNone {
			d273 = ps.OverlayValues[273]
		}
		if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != scm.LocNone {
			d274 = ps.OverlayValues[274]
		}
		if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != scm.LocNone {
			d275 = ps.OverlayValues[275]
		}
		if len(ps.OverlayValues) > 277 && ps.OverlayValues[277].Loc != scm.LocNone {
			d277 = ps.OverlayValues[277]
		}
		if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != scm.LocNone {
			d278 = ps.OverlayValues[278]
		}
		if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != scm.LocNone {
			d279 = ps.OverlayValues[279]
		}
		if len(ps.OverlayValues) > 280 && ps.OverlayValues[280].Loc != scm.LocNone {
			d280 = ps.OverlayValues[280]
		}
		if len(ps.OverlayValues) > 281 && ps.OverlayValues[281].Loc != scm.LocNone {
			d281 = ps.OverlayValues[281]
		}
		if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != scm.LocNone {
			d282 = ps.OverlayValues[282]
		}
		if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != scm.LocNone {
			d283 = ps.OverlayValues[283]
		}
		if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != scm.LocNone {
			d284 = ps.OverlayValues[284]
		}
		if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != scm.LocNone {
			d286 = ps.OverlayValues[286]
		}
		if len(ps.OverlayValues) > 288 && ps.OverlayValues[288].Loc != scm.LocNone {
			d288 = ps.OverlayValues[288]
		}
		if len(ps.OverlayValues) > 289 && ps.OverlayValues[289].Loc != scm.LocNone {
			d289 = ps.OverlayValues[289]
		}
		if len(ps.OverlayValues) > 290 && ps.OverlayValues[290].Loc != scm.LocNone {
			d290 = ps.OverlayValues[290]
		}
		if len(ps.OverlayValues) > 291 && ps.OverlayValues[291].Loc != scm.LocNone {
			d291 = ps.OverlayValues[291]
		}
		if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != scm.LocNone {
			d292 = ps.OverlayValues[292]
		}
		if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
			d295 = ps.OverlayValues[295]
		}
		if len(ps.OverlayValues) > 415 && ps.OverlayValues[415].Loc != scm.LocNone {
			d415 = ps.OverlayValues[415]
		}
		if len(ps.OverlayValues) > 416 && ps.OverlayValues[416].Loc != scm.LocNone {
			d416 = ps.OverlayValues[416]
		}
		if len(ps.OverlayValues) > 417 && ps.OverlayValues[417].Loc != scm.LocNone {
			d417 = ps.OverlayValues[417]
		}
		if len(ps.OverlayValues) > 418 && ps.OverlayValues[418].Loc != scm.LocNone {
			d418 = ps.OverlayValues[418]
		}
		if len(ps.OverlayValues) > 419 && ps.OverlayValues[419].Loc != scm.LocNone {
			d419 = ps.OverlayValues[419]
		}
		if len(ps.OverlayValues) > 421 && ps.OverlayValues[421].Loc != scm.LocNone {
			d421 = ps.OverlayValues[421]
		}
		if len(ps.OverlayValues) > 422 && ps.OverlayValues[422].Loc != scm.LocNone {
			d422 = ps.OverlayValues[422]
		}
		if len(ps.OverlayValues) > 423 && ps.OverlayValues[423].Loc != scm.LocNone {
			d423 = ps.OverlayValues[423]
		}
		if len(ps.OverlayValues) > 425 && ps.OverlayValues[425].Loc != scm.LocNone {
			d425 = ps.OverlayValues[425]
		}
		if len(ps.OverlayValues) > 426 && ps.OverlayValues[426].Loc != scm.LocNone {
			d426 = ps.OverlayValues[426]
		}
		if len(ps.OverlayValues) > 427 && ps.OverlayValues[427].Loc != scm.LocNone {
			d427 = ps.OverlayValues[427]
		}
		if len(ps.OverlayValues) > 428 && ps.OverlayValues[428].Loc != scm.LocNone {
			d428 = ps.OverlayValues[428]
		}
		if len(ps.OverlayValues) > 429 && ps.OverlayValues[429].Loc != scm.LocNone {
			d429 = ps.OverlayValues[429]
		}
		if len(ps.OverlayValues) > 430 && ps.OverlayValues[430].Loc != scm.LocNone {
			d430 = ps.OverlayValues[430]
		}
		if len(ps.OverlayValues) > 431 && ps.OverlayValues[431].Loc != scm.LocNone {
			d431 = ps.OverlayValues[431]
		}
		if len(ps.OverlayValues) > 432 && ps.OverlayValues[432].Loc != scm.LocNone {
			d432 = ps.OverlayValues[432]
		}
		if len(ps.OverlayValues) > 433 && ps.OverlayValues[433].Loc != scm.LocNone {
			d433 = ps.OverlayValues[433]
		}
		if len(ps.OverlayValues) > 434 && ps.OverlayValues[434].Loc != scm.LocNone {
			d434 = ps.OverlayValues[434]
		}
		if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != scm.LocNone {
			d435 = ps.OverlayValues[435]
		}
		if len(ps.OverlayValues) > 436 && ps.OverlayValues[436].Loc != scm.LocNone {
			d436 = ps.OverlayValues[436]
		}
		if len(ps.OverlayValues) > 437 && ps.OverlayValues[437].Loc != scm.LocNone {
			d437 = ps.OverlayValues[437]
		}
		if len(ps.OverlayValues) > 438 && ps.OverlayValues[438].Loc != scm.LocNone {
			d438 = ps.OverlayValues[438]
		}
		if len(ps.OverlayValues) > 439 && ps.OverlayValues[439].Loc != scm.LocNone {
			d439 = ps.OverlayValues[439]
		}
		if len(ps.OverlayValues) > 440 && ps.OverlayValues[440].Loc != scm.LocNone {
			d440 = ps.OverlayValues[440]
		}
		if len(ps.OverlayValues) > 441 && ps.OverlayValues[441].Loc != scm.LocNone {
			d441 = ps.OverlayValues[441]
		}
		if len(ps.OverlayValues) > 442 && ps.OverlayValues[442].Loc != scm.LocNone {
			d442 = ps.OverlayValues[442]
		}
		if len(ps.OverlayValues) > 443 && ps.OverlayValues[443].Loc != scm.LocNone {
			d443 = ps.OverlayValues[443]
		}
		if len(ps.OverlayValues) > 444 && ps.OverlayValues[444].Loc != scm.LocNone {
			d444 = ps.OverlayValues[444]
		}
		if len(ps.OverlayValues) > 445 && ps.OverlayValues[445].Loc != scm.LocNone {
			d445 = ps.OverlayValues[445]
		}
		if len(ps.OverlayValues) > 446 && ps.OverlayValues[446].Loc != scm.LocNone {
			d446 = ps.OverlayValues[446]
		}
		if len(ps.OverlayValues) > 447 && ps.OverlayValues[447].Loc != scm.LocNone {
			d447 = ps.OverlayValues[447]
		}
		if len(ps.OverlayValues) > 448 && ps.OverlayValues[448].Loc != scm.LocNone {
			d448 = ps.OverlayValues[448]
		}
		if len(ps.OverlayValues) > 449 && ps.OverlayValues[449].Loc != scm.LocNone {
			d449 = ps.OverlayValues[449]
		}
		if len(ps.OverlayValues) > 450 && ps.OverlayValues[450].Loc != scm.LocNone {
			d450 = ps.OverlayValues[450]
		}
		if len(ps.OverlayValues) > 451 && ps.OverlayValues[451].Loc != scm.LocNone {
			d451 = ps.OverlayValues[451]
		}
		if len(ps.OverlayValues) > 452 && ps.OverlayValues[452].Loc != scm.LocNone {
			d452 = ps.OverlayValues[452]
		}
		if len(ps.OverlayValues) > 453 && ps.OverlayValues[453].Loc != scm.LocNone {
			d453 = ps.OverlayValues[453]
		}
		if len(ps.OverlayValues) > 454 && ps.OverlayValues[454].Loc != scm.LocNone {
			d454 = ps.OverlayValues[454]
		}
		if len(ps.OverlayValues) > 455 && ps.OverlayValues[455].Loc != scm.LocNone {
			d455 = ps.OverlayValues[455]
		}
		if len(ps.OverlayValues) > 456 && ps.OverlayValues[456].Loc != scm.LocNone {
			d456 = ps.OverlayValues[456]
		}
		if len(ps.OverlayValues) > 457 && ps.OverlayValues[457].Loc != scm.LocNone {
			d457 = ps.OverlayValues[457]
		}
		if len(ps.OverlayValues) > 458 && ps.OverlayValues[458].Loc != scm.LocNone {
			d458 = ps.OverlayValues[458]
		}
		if len(ps.OverlayValues) > 459 && ps.OverlayValues[459].Loc != scm.LocNone {
			d459 = ps.OverlayValues[459]
		}
		if len(ps.OverlayValues) > 626 && ps.OverlayValues[626].Loc != scm.LocNone {
			d626 = ps.OverlayValues[626]
		}
		if len(ps.OverlayValues) > 627 && ps.OverlayValues[627].Loc != scm.LocNone {
			d627 = ps.OverlayValues[627]
		}
		if len(ps.OverlayValues) > 628 && ps.OverlayValues[628].Loc != scm.LocNone {
			d628 = ps.OverlayValues[628]
		}
		if len(ps.OverlayValues) > 630 && ps.OverlayValues[630].Loc != scm.LocNone {
			d630 = ps.OverlayValues[630]
		}
		if len(ps.OverlayValues) > 631 && ps.OverlayValues[631].Loc != scm.LocNone {
			d631 = ps.OverlayValues[631]
		}
		if len(ps.OverlayValues) > 632 && ps.OverlayValues[632].Loc != scm.LocNone {
			d632 = ps.OverlayValues[632]
		}
		if len(ps.OverlayValues) > 633 && ps.OverlayValues[633].Loc != scm.LocNone {
			d633 = ps.OverlayValues[633]
		}
		if len(ps.OverlayValues) > 634 && ps.OverlayValues[634].Loc != scm.LocNone {
			d634 = ps.OverlayValues[634]
		}
		if len(ps.OverlayValues) > 635 && ps.OverlayValues[635].Loc != scm.LocNone {
			d635 = ps.OverlayValues[635]
		}
		if len(ps.OverlayValues) > 636 && ps.OverlayValues[636].Loc != scm.LocNone {
			d636 = ps.OverlayValues[636]
		}
		if len(ps.OverlayValues) > 638 && ps.OverlayValues[638].Loc != scm.LocNone {
			d638 = ps.OverlayValues[638]
		}
		if len(ps.OverlayValues) > 640 && ps.OverlayValues[640].Loc != scm.LocNone {
			d640 = ps.OverlayValues[640]
		}
		if len(ps.OverlayValues) > 641 && ps.OverlayValues[641].Loc != scm.LocNone {
			d641 = ps.OverlayValues[641]
		}
		if len(ps.OverlayValues) > 642 && ps.OverlayValues[642].Loc != scm.LocNone {
			d642 = ps.OverlayValues[642]
		}
		if len(ps.OverlayValues) > 643 && ps.OverlayValues[643].Loc != scm.LocNone {
			d643 = ps.OverlayValues[643]
		}
		if len(ps.OverlayValues) > 646 && ps.OverlayValues[646].Loc != scm.LocNone {
			d646 = ps.OverlayValues[646]
		}
		if len(ps.OverlayValues) > 825 && ps.OverlayValues[825].Loc != scm.LocNone {
			d825 = ps.OverlayValues[825]
		}
		if len(ps.OverlayValues) > 826 && ps.OverlayValues[826].Loc != scm.LocNone {
			d826 = ps.OverlayValues[826]
		}
		if len(ps.OverlayValues) > 827 && ps.OverlayValues[827].Loc != scm.LocNone {
			d827 = ps.OverlayValues[827]
		}
		if len(ps.OverlayValues) > 828 && ps.OverlayValues[828].Loc != scm.LocNone {
			d828 = ps.OverlayValues[828]
		}
		if len(ps.OverlayValues) > 830 && ps.OverlayValues[830].Loc != scm.LocNone {
			d830 = ps.OverlayValues[830]
		}
		if len(ps.OverlayValues) > 831 && ps.OverlayValues[831].Loc != scm.LocNone {
			d831 = ps.OverlayValues[831]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d8)
		ctx.EnsureDesc(&d9)
		ctx.EnsureDesc(&d8)
		ctx.ProtectReg(d8.Reg)
		ctx.EnsureDesc(&d9)
		ctx.UnprotectReg(d8.Reg)
		var d832 scm.JITValueDesc
		if d8.Loc == scm.LocImm && d9.Loc == scm.LocImm {
			d832 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() + d9.Imm.Int())}
		} else if d9.Loc == scm.LocImm && d9.Imm.Int() == 0 {
			r159 := ctx.AllocRegExcept(d8.Reg)
			ctx.EmitMovRegReg(r159, d8.Reg)
			d832 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
			ctx.BindReg(r159, &d832)
		} else if d8.Loc == scm.LocImm && d8.Imm.Int() == 0 {
			d832 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d9.Reg}
			ctx.BindReg(d9.Reg, &d832)
		} else if d8.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d9.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d8.Imm.Int()))
			ctx.EmitAddInt64(scratch, d9.Reg)
			d832 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d832)
		} else if d9.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d8.Reg)
			ctx.EmitMovRegReg(scratch, d8.Reg)
			if d9.Imm.Int() >= -2147483648 && d9.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d9.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d9.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d832 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d832)
		} else {
			r160 := ctx.AllocRegExcept(d8.Reg, d9.Reg)
			ctx.EmitMovRegReg(r160, d8.Reg)
			ctx.EmitAddInt64(r160, d9.Reg)
			d832 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r160}
			ctx.BindReg(r160, &d832)
		}
		if d832.Loc == scm.LocImm {
			d832 = scm.JITValueDesc{Loc: scm.LocImm, Type: d832.Type, Imm: scm.NewInt(int64(uint64(d832.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d832.Reg, 32)
			ctx.EmitShrRegImm8(d832.Reg, 32)
		}
		if d832.Loc == scm.LocReg && d8.Loc == scm.LocReg && d832.Reg == d8.Reg {
			ctx.TransferReg(d8.Reg)
			d8.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d832)
		var d833 scm.JITValueDesc
		if d832.Loc == scm.LocImm {
			d833 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d832.Imm.Int() / 2)}
		} else {
			r161 := ctx.AllocRegExcept(d832.Reg)
			ctx.EmitMovRegReg(r161, d832.Reg)
			ctx.EmitShrRegImm8(r161, 1)
			d833 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r161}
			ctx.BindReg(r161, &d833)
		}
		if d833.Loc == scm.LocImm {
			d833 = scm.JITValueDesc{Loc: scm.LocImm, Type: d833.Type, Imm: scm.NewInt(int64(uint64(d833.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d833.Reg, 32)
			ctx.EmitShrRegImm8(d833.Reg, 32)
		}
		if d833.Loc == scm.LocReg && d832.Loc == scm.LocReg && d833.Reg == d832.Reg {
			ctx.TransferReg(d832.Reg)
			d832.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d833)
		ctx.EmitStoreToStack(d833, int32(bbs[1].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d833)
		ctx.FreeDesc(&d832)
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
			d834 = d8
			if d834.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d834)
			d835 = d834
			if d835.Loc == scm.LocImm {
				d835 = scm.JITValueDesc{Loc: scm.LocImm, Type: d835.Type, Imm: scm.NewInt(int64(uint64(d835.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d835.Reg, 32)
				ctx.EmitShrRegImm8(d835.Reg, 32)
			}
			ctx.EmitStoreToStack(d835, int32(bbs[1].PhiBase)+int32(16))
			d836 = d9
			if d836.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d836)
			d837 = d836
			if d837.Loc == scm.LocImm {
				d837 = scm.JITValueDesc{Loc: scm.LocImm, Type: d837.Type, Imm: scm.NewInt(int64(uint64(d837.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d837.Reg, 32)
				ctx.EmitShrRegImm8(d837.Reg, 32)
			}
			ctx.EmitStoreToStack(d837, int32(bbs[1].PhiBase)+int32(32))
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
		ps838 := scm.PhiState{General: ps.General}
		ps838.OverlayValues = make([]scm.JITValueDesc, 838)
		ps838.OverlayValues[1] = d1
		ps838.OverlayValues[2] = d2
		ps838.OverlayValues[3] = d3
		ps838.OverlayValues[4] = d4
		ps838.OverlayValues[5] = d5
		ps838.OverlayValues[6] = d6
		ps838.OverlayValues[7] = d7
		ps838.OverlayValues[8] = d8
		ps838.OverlayValues[9] = d9
		ps838.OverlayValues[10] = d10
		ps838.OverlayValues[11] = d11
		ps838.OverlayValues[12] = d12
		ps838.OverlayValues[13] = d13
		ps838.OverlayValues[14] = d14
		ps838.OverlayValues[15] = d15
		ps838.OverlayValues[17] = d17
		ps838.OverlayValues[18] = d18
		ps838.OverlayValues[19] = d19
		ps838.OverlayValues[20] = d20
		ps838.OverlayValues[21] = d21
		ps838.OverlayValues[22] = d22
		ps838.OverlayValues[24] = d24
		ps838.OverlayValues[25] = d25
		ps838.OverlayValues[26] = d26
		ps838.OverlayValues[27] = d27
		ps838.OverlayValues[28] = d28
		ps838.OverlayValues[29] = d29
		ps838.OverlayValues[30] = d30
		ps838.OverlayValues[31] = d31
		ps838.OverlayValues[32] = d32
		ps838.OverlayValues[33] = d33
		ps838.OverlayValues[34] = d34
		ps838.OverlayValues[35] = d35
		ps838.OverlayValues[36] = d36
		ps838.OverlayValues[37] = d37
		ps838.OverlayValues[38] = d38
		ps838.OverlayValues[39] = d39
		ps838.OverlayValues[40] = d40
		ps838.OverlayValues[41] = d41
		ps838.OverlayValues[42] = d42
		ps838.OverlayValues[43] = d43
		ps838.OverlayValues[44] = d44
		ps838.OverlayValues[45] = d45
		ps838.OverlayValues[46] = d46
		ps838.OverlayValues[47] = d47
		ps838.OverlayValues[48] = d48
		ps838.OverlayValues[49] = d49
		ps838.OverlayValues[50] = d50
		ps838.OverlayValues[51] = d51
		ps838.OverlayValues[52] = d52
		ps838.OverlayValues[53] = d53
		ps838.OverlayValues[54] = d54
		ps838.OverlayValues[55] = d55
		ps838.OverlayValues[56] = d56
		ps838.OverlayValues[57] = d57
		ps838.OverlayValues[58] = d58
		ps838.OverlayValues[59] = d59
		ps838.OverlayValues[62] = d62
		ps838.OverlayValues[63] = d63
		ps838.OverlayValues[64] = d64
		ps838.OverlayValues[128] = d128
		ps838.OverlayValues[129] = d129
		ps838.OverlayValues[130] = d130
		ps838.OverlayValues[132] = d132
		ps838.OverlayValues[133] = d133
		ps838.OverlayValues[134] = d134
		ps838.OverlayValues[135] = d135
		ps838.OverlayValues[136] = d136
		ps838.OverlayValues[137] = d137
		ps838.OverlayValues[138] = d138
		ps838.OverlayValues[139] = d139
		ps838.OverlayValues[140] = d140
		ps838.OverlayValues[141] = d141
		ps838.OverlayValues[142] = d142
		ps838.OverlayValues[143] = d143
		ps838.OverlayValues[144] = d144
		ps838.OverlayValues[145] = d145
		ps838.OverlayValues[146] = d146
		ps838.OverlayValues[147] = d147
		ps838.OverlayValues[148] = d148
		ps838.OverlayValues[149] = d149
		ps838.OverlayValues[150] = d150
		ps838.OverlayValues[151] = d151
		ps838.OverlayValues[152] = d152
		ps838.OverlayValues[153] = d153
		ps838.OverlayValues[154] = d154
		ps838.OverlayValues[155] = d155
		ps838.OverlayValues[156] = d156
		ps838.OverlayValues[157] = d157
		ps838.OverlayValues[158] = d158
		ps838.OverlayValues[159] = d159
		ps838.OverlayValues[160] = d160
		ps838.OverlayValues[161] = d161
		ps838.OverlayValues[162] = d162
		ps838.OverlayValues[163] = d163
		ps838.OverlayValues[164] = d164
		ps838.OverlayValues[165] = d165
		ps838.OverlayValues[166] = d166
		ps838.OverlayValues[169] = d169
		ps838.OverlayValues[272] = d272
		ps838.OverlayValues[273] = d273
		ps838.OverlayValues[274] = d274
		ps838.OverlayValues[275] = d275
		ps838.OverlayValues[277] = d277
		ps838.OverlayValues[278] = d278
		ps838.OverlayValues[279] = d279
		ps838.OverlayValues[280] = d280
		ps838.OverlayValues[281] = d281
		ps838.OverlayValues[282] = d282
		ps838.OverlayValues[283] = d283
		ps838.OverlayValues[284] = d284
		ps838.OverlayValues[286] = d286
		ps838.OverlayValues[288] = d288
		ps838.OverlayValues[289] = d289
		ps838.OverlayValues[290] = d290
		ps838.OverlayValues[291] = d291
		ps838.OverlayValues[292] = d292
		ps838.OverlayValues[295] = d295
		ps838.OverlayValues[415] = d415
		ps838.OverlayValues[416] = d416
		ps838.OverlayValues[417] = d417
		ps838.OverlayValues[418] = d418
		ps838.OverlayValues[419] = d419
		ps838.OverlayValues[421] = d421
		ps838.OverlayValues[422] = d422
		ps838.OverlayValues[423] = d423
		ps838.OverlayValues[425] = d425
		ps838.OverlayValues[426] = d426
		ps838.OverlayValues[427] = d427
		ps838.OverlayValues[428] = d428
		ps838.OverlayValues[429] = d429
		ps838.OverlayValues[430] = d430
		ps838.OverlayValues[431] = d431
		ps838.OverlayValues[432] = d432
		ps838.OverlayValues[433] = d433
		ps838.OverlayValues[434] = d434
		ps838.OverlayValues[435] = d435
		ps838.OverlayValues[436] = d436
		ps838.OverlayValues[437] = d437
		ps838.OverlayValues[438] = d438
		ps838.OverlayValues[439] = d439
		ps838.OverlayValues[440] = d440
		ps838.OverlayValues[441] = d441
		ps838.OverlayValues[442] = d442
		ps838.OverlayValues[443] = d443
		ps838.OverlayValues[444] = d444
		ps838.OverlayValues[445] = d445
		ps838.OverlayValues[446] = d446
		ps838.OverlayValues[447] = d447
		ps838.OverlayValues[448] = d448
		ps838.OverlayValues[449] = d449
		ps838.OverlayValues[450] = d450
		ps838.OverlayValues[451] = d451
		ps838.OverlayValues[452] = d452
		ps838.OverlayValues[453] = d453
		ps838.OverlayValues[454] = d454
		ps838.OverlayValues[455] = d455
		ps838.OverlayValues[456] = d456
		ps838.OverlayValues[457] = d457
		ps838.OverlayValues[458] = d458
		ps838.OverlayValues[459] = d459
		ps838.OverlayValues[626] = d626
		ps838.OverlayValues[627] = d627
		ps838.OverlayValues[628] = d628
		ps838.OverlayValues[630] = d630
		ps838.OverlayValues[631] = d631
		ps838.OverlayValues[632] = d632
		ps838.OverlayValues[633] = d633
		ps838.OverlayValues[634] = d634
		ps838.OverlayValues[635] = d635
		ps838.OverlayValues[636] = d636
		ps838.OverlayValues[638] = d638
		ps838.OverlayValues[640] = d640
		ps838.OverlayValues[641] = d641
		ps838.OverlayValues[642] = d642
		ps838.OverlayValues[643] = d643
		ps838.OverlayValues[646] = d646
		ps838.OverlayValues[825] = d825
		ps838.OverlayValues[826] = d826
		ps838.OverlayValues[827] = d827
		ps838.OverlayValues[828] = d828
		ps838.OverlayValues[830] = d830
		ps838.OverlayValues[831] = d831
		ps838.OverlayValues[832] = d832
		ps838.OverlayValues[833] = d833
		ps838.OverlayValues[834] = d834
		ps838.OverlayValues[835] = d835
		ps838.OverlayValues[836] = d836
		ps838.OverlayValues[837] = d837
		ps838.PhiValues = make([]scm.JITValueDesc, 3)
		d839 = d8
		ps838.PhiValues[1] = d839
		d840 = d9
		ps838.PhiValues[2] = d840
		if ps838.General && bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		return bbs[1].RenderPS(ps838)
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
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
		}
		if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
			d58 = ps.OverlayValues[58]
		}
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
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
		if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
			d128 = ps.OverlayValues[128]
		}
		if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
			d129 = ps.OverlayValues[129]
		}
		if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
			d130 = ps.OverlayValues[130]
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
		if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != scm.LocNone {
			d148 = ps.OverlayValues[148]
		}
		if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != scm.LocNone {
			d149 = ps.OverlayValues[149]
		}
		if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
			d150 = ps.OverlayValues[150]
		}
		if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != scm.LocNone {
			d151 = ps.OverlayValues[151]
		}
		if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != scm.LocNone {
			d152 = ps.OverlayValues[152]
		}
		if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
			d153 = ps.OverlayValues[153]
		}
		if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
			d154 = ps.OverlayValues[154]
		}
		if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != scm.LocNone {
			d155 = ps.OverlayValues[155]
		}
		if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != scm.LocNone {
			d156 = ps.OverlayValues[156]
		}
		if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != scm.LocNone {
			d157 = ps.OverlayValues[157]
		}
		if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != scm.LocNone {
			d158 = ps.OverlayValues[158]
		}
		if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != scm.LocNone {
			d159 = ps.OverlayValues[159]
		}
		if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != scm.LocNone {
			d160 = ps.OverlayValues[160]
		}
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != scm.LocNone {
			d161 = ps.OverlayValues[161]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != scm.LocNone {
			d162 = ps.OverlayValues[162]
		}
		if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != scm.LocNone {
			d163 = ps.OverlayValues[163]
		}
		if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != scm.LocNone {
			d164 = ps.OverlayValues[164]
		}
		if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != scm.LocNone {
			d165 = ps.OverlayValues[165]
		}
		if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != scm.LocNone {
			d166 = ps.OverlayValues[166]
		}
		if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != scm.LocNone {
			d169 = ps.OverlayValues[169]
		}
		if len(ps.OverlayValues) > 272 && ps.OverlayValues[272].Loc != scm.LocNone {
			d272 = ps.OverlayValues[272]
		}
		if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != scm.LocNone {
			d273 = ps.OverlayValues[273]
		}
		if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != scm.LocNone {
			d274 = ps.OverlayValues[274]
		}
		if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != scm.LocNone {
			d275 = ps.OverlayValues[275]
		}
		if len(ps.OverlayValues) > 277 && ps.OverlayValues[277].Loc != scm.LocNone {
			d277 = ps.OverlayValues[277]
		}
		if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != scm.LocNone {
			d278 = ps.OverlayValues[278]
		}
		if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != scm.LocNone {
			d279 = ps.OverlayValues[279]
		}
		if len(ps.OverlayValues) > 280 && ps.OverlayValues[280].Loc != scm.LocNone {
			d280 = ps.OverlayValues[280]
		}
		if len(ps.OverlayValues) > 281 && ps.OverlayValues[281].Loc != scm.LocNone {
			d281 = ps.OverlayValues[281]
		}
		if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != scm.LocNone {
			d282 = ps.OverlayValues[282]
		}
		if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != scm.LocNone {
			d283 = ps.OverlayValues[283]
		}
		if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != scm.LocNone {
			d284 = ps.OverlayValues[284]
		}
		if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != scm.LocNone {
			d286 = ps.OverlayValues[286]
		}
		if len(ps.OverlayValues) > 288 && ps.OverlayValues[288].Loc != scm.LocNone {
			d288 = ps.OverlayValues[288]
		}
		if len(ps.OverlayValues) > 289 && ps.OverlayValues[289].Loc != scm.LocNone {
			d289 = ps.OverlayValues[289]
		}
		if len(ps.OverlayValues) > 290 && ps.OverlayValues[290].Loc != scm.LocNone {
			d290 = ps.OverlayValues[290]
		}
		if len(ps.OverlayValues) > 291 && ps.OverlayValues[291].Loc != scm.LocNone {
			d291 = ps.OverlayValues[291]
		}
		if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != scm.LocNone {
			d292 = ps.OverlayValues[292]
		}
		if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
			d295 = ps.OverlayValues[295]
		}
		if len(ps.OverlayValues) > 415 && ps.OverlayValues[415].Loc != scm.LocNone {
			d415 = ps.OverlayValues[415]
		}
		if len(ps.OverlayValues) > 416 && ps.OverlayValues[416].Loc != scm.LocNone {
			d416 = ps.OverlayValues[416]
		}
		if len(ps.OverlayValues) > 417 && ps.OverlayValues[417].Loc != scm.LocNone {
			d417 = ps.OverlayValues[417]
		}
		if len(ps.OverlayValues) > 418 && ps.OverlayValues[418].Loc != scm.LocNone {
			d418 = ps.OverlayValues[418]
		}
		if len(ps.OverlayValues) > 419 && ps.OverlayValues[419].Loc != scm.LocNone {
			d419 = ps.OverlayValues[419]
		}
		if len(ps.OverlayValues) > 421 && ps.OverlayValues[421].Loc != scm.LocNone {
			d421 = ps.OverlayValues[421]
		}
		if len(ps.OverlayValues) > 422 && ps.OverlayValues[422].Loc != scm.LocNone {
			d422 = ps.OverlayValues[422]
		}
		if len(ps.OverlayValues) > 423 && ps.OverlayValues[423].Loc != scm.LocNone {
			d423 = ps.OverlayValues[423]
		}
		if len(ps.OverlayValues) > 425 && ps.OverlayValues[425].Loc != scm.LocNone {
			d425 = ps.OverlayValues[425]
		}
		if len(ps.OverlayValues) > 426 && ps.OverlayValues[426].Loc != scm.LocNone {
			d426 = ps.OverlayValues[426]
		}
		if len(ps.OverlayValues) > 427 && ps.OverlayValues[427].Loc != scm.LocNone {
			d427 = ps.OverlayValues[427]
		}
		if len(ps.OverlayValues) > 428 && ps.OverlayValues[428].Loc != scm.LocNone {
			d428 = ps.OverlayValues[428]
		}
		if len(ps.OverlayValues) > 429 && ps.OverlayValues[429].Loc != scm.LocNone {
			d429 = ps.OverlayValues[429]
		}
		if len(ps.OverlayValues) > 430 && ps.OverlayValues[430].Loc != scm.LocNone {
			d430 = ps.OverlayValues[430]
		}
		if len(ps.OverlayValues) > 431 && ps.OverlayValues[431].Loc != scm.LocNone {
			d431 = ps.OverlayValues[431]
		}
		if len(ps.OverlayValues) > 432 && ps.OverlayValues[432].Loc != scm.LocNone {
			d432 = ps.OverlayValues[432]
		}
		if len(ps.OverlayValues) > 433 && ps.OverlayValues[433].Loc != scm.LocNone {
			d433 = ps.OverlayValues[433]
		}
		if len(ps.OverlayValues) > 434 && ps.OverlayValues[434].Loc != scm.LocNone {
			d434 = ps.OverlayValues[434]
		}
		if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != scm.LocNone {
			d435 = ps.OverlayValues[435]
		}
		if len(ps.OverlayValues) > 436 && ps.OverlayValues[436].Loc != scm.LocNone {
			d436 = ps.OverlayValues[436]
		}
		if len(ps.OverlayValues) > 437 && ps.OverlayValues[437].Loc != scm.LocNone {
			d437 = ps.OverlayValues[437]
		}
		if len(ps.OverlayValues) > 438 && ps.OverlayValues[438].Loc != scm.LocNone {
			d438 = ps.OverlayValues[438]
		}
		if len(ps.OverlayValues) > 439 && ps.OverlayValues[439].Loc != scm.LocNone {
			d439 = ps.OverlayValues[439]
		}
		if len(ps.OverlayValues) > 440 && ps.OverlayValues[440].Loc != scm.LocNone {
			d440 = ps.OverlayValues[440]
		}
		if len(ps.OverlayValues) > 441 && ps.OverlayValues[441].Loc != scm.LocNone {
			d441 = ps.OverlayValues[441]
		}
		if len(ps.OverlayValues) > 442 && ps.OverlayValues[442].Loc != scm.LocNone {
			d442 = ps.OverlayValues[442]
		}
		if len(ps.OverlayValues) > 443 && ps.OverlayValues[443].Loc != scm.LocNone {
			d443 = ps.OverlayValues[443]
		}
		if len(ps.OverlayValues) > 444 && ps.OverlayValues[444].Loc != scm.LocNone {
			d444 = ps.OverlayValues[444]
		}
		if len(ps.OverlayValues) > 445 && ps.OverlayValues[445].Loc != scm.LocNone {
			d445 = ps.OverlayValues[445]
		}
		if len(ps.OverlayValues) > 446 && ps.OverlayValues[446].Loc != scm.LocNone {
			d446 = ps.OverlayValues[446]
		}
		if len(ps.OverlayValues) > 447 && ps.OverlayValues[447].Loc != scm.LocNone {
			d447 = ps.OverlayValues[447]
		}
		if len(ps.OverlayValues) > 448 && ps.OverlayValues[448].Loc != scm.LocNone {
			d448 = ps.OverlayValues[448]
		}
		if len(ps.OverlayValues) > 449 && ps.OverlayValues[449].Loc != scm.LocNone {
			d449 = ps.OverlayValues[449]
		}
		if len(ps.OverlayValues) > 450 && ps.OverlayValues[450].Loc != scm.LocNone {
			d450 = ps.OverlayValues[450]
		}
		if len(ps.OverlayValues) > 451 && ps.OverlayValues[451].Loc != scm.LocNone {
			d451 = ps.OverlayValues[451]
		}
		if len(ps.OverlayValues) > 452 && ps.OverlayValues[452].Loc != scm.LocNone {
			d452 = ps.OverlayValues[452]
		}
		if len(ps.OverlayValues) > 453 && ps.OverlayValues[453].Loc != scm.LocNone {
			d453 = ps.OverlayValues[453]
		}
		if len(ps.OverlayValues) > 454 && ps.OverlayValues[454].Loc != scm.LocNone {
			d454 = ps.OverlayValues[454]
		}
		if len(ps.OverlayValues) > 455 && ps.OverlayValues[455].Loc != scm.LocNone {
			d455 = ps.OverlayValues[455]
		}
		if len(ps.OverlayValues) > 456 && ps.OverlayValues[456].Loc != scm.LocNone {
			d456 = ps.OverlayValues[456]
		}
		if len(ps.OverlayValues) > 457 && ps.OverlayValues[457].Loc != scm.LocNone {
			d457 = ps.OverlayValues[457]
		}
		if len(ps.OverlayValues) > 458 && ps.OverlayValues[458].Loc != scm.LocNone {
			d458 = ps.OverlayValues[458]
		}
		if len(ps.OverlayValues) > 459 && ps.OverlayValues[459].Loc != scm.LocNone {
			d459 = ps.OverlayValues[459]
		}
		if len(ps.OverlayValues) > 626 && ps.OverlayValues[626].Loc != scm.LocNone {
			d626 = ps.OverlayValues[626]
		}
		if len(ps.OverlayValues) > 627 && ps.OverlayValues[627].Loc != scm.LocNone {
			d627 = ps.OverlayValues[627]
		}
		if len(ps.OverlayValues) > 628 && ps.OverlayValues[628].Loc != scm.LocNone {
			d628 = ps.OverlayValues[628]
		}
		if len(ps.OverlayValues) > 630 && ps.OverlayValues[630].Loc != scm.LocNone {
			d630 = ps.OverlayValues[630]
		}
		if len(ps.OverlayValues) > 631 && ps.OverlayValues[631].Loc != scm.LocNone {
			d631 = ps.OverlayValues[631]
		}
		if len(ps.OverlayValues) > 632 && ps.OverlayValues[632].Loc != scm.LocNone {
			d632 = ps.OverlayValues[632]
		}
		if len(ps.OverlayValues) > 633 && ps.OverlayValues[633].Loc != scm.LocNone {
			d633 = ps.OverlayValues[633]
		}
		if len(ps.OverlayValues) > 634 && ps.OverlayValues[634].Loc != scm.LocNone {
			d634 = ps.OverlayValues[634]
		}
		if len(ps.OverlayValues) > 635 && ps.OverlayValues[635].Loc != scm.LocNone {
			d635 = ps.OverlayValues[635]
		}
		if len(ps.OverlayValues) > 636 && ps.OverlayValues[636].Loc != scm.LocNone {
			d636 = ps.OverlayValues[636]
		}
		if len(ps.OverlayValues) > 638 && ps.OverlayValues[638].Loc != scm.LocNone {
			d638 = ps.OverlayValues[638]
		}
		if len(ps.OverlayValues) > 640 && ps.OverlayValues[640].Loc != scm.LocNone {
			d640 = ps.OverlayValues[640]
		}
		if len(ps.OverlayValues) > 641 && ps.OverlayValues[641].Loc != scm.LocNone {
			d641 = ps.OverlayValues[641]
		}
		if len(ps.OverlayValues) > 642 && ps.OverlayValues[642].Loc != scm.LocNone {
			d642 = ps.OverlayValues[642]
		}
		if len(ps.OverlayValues) > 643 && ps.OverlayValues[643].Loc != scm.LocNone {
			d643 = ps.OverlayValues[643]
		}
		if len(ps.OverlayValues) > 646 && ps.OverlayValues[646].Loc != scm.LocNone {
			d646 = ps.OverlayValues[646]
		}
		if len(ps.OverlayValues) > 825 && ps.OverlayValues[825].Loc != scm.LocNone {
			d825 = ps.OverlayValues[825]
		}
		if len(ps.OverlayValues) > 826 && ps.OverlayValues[826].Loc != scm.LocNone {
			d826 = ps.OverlayValues[826]
		}
		if len(ps.OverlayValues) > 827 && ps.OverlayValues[827].Loc != scm.LocNone {
			d827 = ps.OverlayValues[827]
		}
		if len(ps.OverlayValues) > 828 && ps.OverlayValues[828].Loc != scm.LocNone {
			d828 = ps.OverlayValues[828]
		}
		if len(ps.OverlayValues) > 830 && ps.OverlayValues[830].Loc != scm.LocNone {
			d830 = ps.OverlayValues[830]
		}
		if len(ps.OverlayValues) > 831 && ps.OverlayValues[831].Loc != scm.LocNone {
			d831 = ps.OverlayValues[831]
		}
		if len(ps.OverlayValues) > 832 && ps.OverlayValues[832].Loc != scm.LocNone {
			d832 = ps.OverlayValues[832]
		}
		if len(ps.OverlayValues) > 833 && ps.OverlayValues[833].Loc != scm.LocNone {
			d833 = ps.OverlayValues[833]
		}
		if len(ps.OverlayValues) > 834 && ps.OverlayValues[834].Loc != scm.LocNone {
			d834 = ps.OverlayValues[834]
		}
		if len(ps.OverlayValues) > 835 && ps.OverlayValues[835].Loc != scm.LocNone {
			d835 = ps.OverlayValues[835]
		}
		if len(ps.OverlayValues) > 836 && ps.OverlayValues[836].Loc != scm.LocNone {
			d836 = ps.OverlayValues[836]
		}
		if len(ps.OverlayValues) > 837 && ps.OverlayValues[837].Loc != scm.LocNone {
			d837 = ps.OverlayValues[837]
		}
		if len(ps.OverlayValues) > 839 && ps.OverlayValues[839].Loc != scm.LocNone {
			d839 = ps.OverlayValues[839]
		}
		if len(ps.OverlayValues) > 840 && ps.OverlayValues[840].Loc != scm.LocNone {
			d840 = ps.OverlayValues[840]
		}
		ctx.ReclaimUntrackedRegs()
		d841 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d842 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d842)
		ctx.BindReg(r1, &d842)
		ctx.EnsureDesc(&d841)
		if d841.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d841, &d842)
		} else {
			switch d841.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d842, d841)
			case scm.TagInt:
				ctx.EmitMakeInt(d842, d841)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d842, d841)
			case scm.TagNil:
				ctx.EmitMakeNil(d842)
			default:
				ctx.EmitMovPairToResult(&d841, &d842)
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
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
		}
		if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
			d58 = ps.OverlayValues[58]
		}
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
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
		if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
			d128 = ps.OverlayValues[128]
		}
		if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
			d129 = ps.OverlayValues[129]
		}
		if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
			d130 = ps.OverlayValues[130]
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
		if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != scm.LocNone {
			d148 = ps.OverlayValues[148]
		}
		if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != scm.LocNone {
			d149 = ps.OverlayValues[149]
		}
		if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
			d150 = ps.OverlayValues[150]
		}
		if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != scm.LocNone {
			d151 = ps.OverlayValues[151]
		}
		if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != scm.LocNone {
			d152 = ps.OverlayValues[152]
		}
		if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
			d153 = ps.OverlayValues[153]
		}
		if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
			d154 = ps.OverlayValues[154]
		}
		if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != scm.LocNone {
			d155 = ps.OverlayValues[155]
		}
		if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != scm.LocNone {
			d156 = ps.OverlayValues[156]
		}
		if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != scm.LocNone {
			d157 = ps.OverlayValues[157]
		}
		if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != scm.LocNone {
			d158 = ps.OverlayValues[158]
		}
		if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != scm.LocNone {
			d159 = ps.OverlayValues[159]
		}
		if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != scm.LocNone {
			d160 = ps.OverlayValues[160]
		}
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != scm.LocNone {
			d161 = ps.OverlayValues[161]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != scm.LocNone {
			d162 = ps.OverlayValues[162]
		}
		if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != scm.LocNone {
			d163 = ps.OverlayValues[163]
		}
		if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != scm.LocNone {
			d164 = ps.OverlayValues[164]
		}
		if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != scm.LocNone {
			d165 = ps.OverlayValues[165]
		}
		if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != scm.LocNone {
			d166 = ps.OverlayValues[166]
		}
		if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != scm.LocNone {
			d169 = ps.OverlayValues[169]
		}
		if len(ps.OverlayValues) > 272 && ps.OverlayValues[272].Loc != scm.LocNone {
			d272 = ps.OverlayValues[272]
		}
		if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != scm.LocNone {
			d273 = ps.OverlayValues[273]
		}
		if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != scm.LocNone {
			d274 = ps.OverlayValues[274]
		}
		if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != scm.LocNone {
			d275 = ps.OverlayValues[275]
		}
		if len(ps.OverlayValues) > 277 && ps.OverlayValues[277].Loc != scm.LocNone {
			d277 = ps.OverlayValues[277]
		}
		if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != scm.LocNone {
			d278 = ps.OverlayValues[278]
		}
		if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != scm.LocNone {
			d279 = ps.OverlayValues[279]
		}
		if len(ps.OverlayValues) > 280 && ps.OverlayValues[280].Loc != scm.LocNone {
			d280 = ps.OverlayValues[280]
		}
		if len(ps.OverlayValues) > 281 && ps.OverlayValues[281].Loc != scm.LocNone {
			d281 = ps.OverlayValues[281]
		}
		if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != scm.LocNone {
			d282 = ps.OverlayValues[282]
		}
		if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != scm.LocNone {
			d283 = ps.OverlayValues[283]
		}
		if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != scm.LocNone {
			d284 = ps.OverlayValues[284]
		}
		if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != scm.LocNone {
			d286 = ps.OverlayValues[286]
		}
		if len(ps.OverlayValues) > 288 && ps.OverlayValues[288].Loc != scm.LocNone {
			d288 = ps.OverlayValues[288]
		}
		if len(ps.OverlayValues) > 289 && ps.OverlayValues[289].Loc != scm.LocNone {
			d289 = ps.OverlayValues[289]
		}
		if len(ps.OverlayValues) > 290 && ps.OverlayValues[290].Loc != scm.LocNone {
			d290 = ps.OverlayValues[290]
		}
		if len(ps.OverlayValues) > 291 && ps.OverlayValues[291].Loc != scm.LocNone {
			d291 = ps.OverlayValues[291]
		}
		if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != scm.LocNone {
			d292 = ps.OverlayValues[292]
		}
		if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
			d295 = ps.OverlayValues[295]
		}
		if len(ps.OverlayValues) > 415 && ps.OverlayValues[415].Loc != scm.LocNone {
			d415 = ps.OverlayValues[415]
		}
		if len(ps.OverlayValues) > 416 && ps.OverlayValues[416].Loc != scm.LocNone {
			d416 = ps.OverlayValues[416]
		}
		if len(ps.OverlayValues) > 417 && ps.OverlayValues[417].Loc != scm.LocNone {
			d417 = ps.OverlayValues[417]
		}
		if len(ps.OverlayValues) > 418 && ps.OverlayValues[418].Loc != scm.LocNone {
			d418 = ps.OverlayValues[418]
		}
		if len(ps.OverlayValues) > 419 && ps.OverlayValues[419].Loc != scm.LocNone {
			d419 = ps.OverlayValues[419]
		}
		if len(ps.OverlayValues) > 421 && ps.OverlayValues[421].Loc != scm.LocNone {
			d421 = ps.OverlayValues[421]
		}
		if len(ps.OverlayValues) > 422 && ps.OverlayValues[422].Loc != scm.LocNone {
			d422 = ps.OverlayValues[422]
		}
		if len(ps.OverlayValues) > 423 && ps.OverlayValues[423].Loc != scm.LocNone {
			d423 = ps.OverlayValues[423]
		}
		if len(ps.OverlayValues) > 425 && ps.OverlayValues[425].Loc != scm.LocNone {
			d425 = ps.OverlayValues[425]
		}
		if len(ps.OverlayValues) > 426 && ps.OverlayValues[426].Loc != scm.LocNone {
			d426 = ps.OverlayValues[426]
		}
		if len(ps.OverlayValues) > 427 && ps.OverlayValues[427].Loc != scm.LocNone {
			d427 = ps.OverlayValues[427]
		}
		if len(ps.OverlayValues) > 428 && ps.OverlayValues[428].Loc != scm.LocNone {
			d428 = ps.OverlayValues[428]
		}
		if len(ps.OverlayValues) > 429 && ps.OverlayValues[429].Loc != scm.LocNone {
			d429 = ps.OverlayValues[429]
		}
		if len(ps.OverlayValues) > 430 && ps.OverlayValues[430].Loc != scm.LocNone {
			d430 = ps.OverlayValues[430]
		}
		if len(ps.OverlayValues) > 431 && ps.OverlayValues[431].Loc != scm.LocNone {
			d431 = ps.OverlayValues[431]
		}
		if len(ps.OverlayValues) > 432 && ps.OverlayValues[432].Loc != scm.LocNone {
			d432 = ps.OverlayValues[432]
		}
		if len(ps.OverlayValues) > 433 && ps.OverlayValues[433].Loc != scm.LocNone {
			d433 = ps.OverlayValues[433]
		}
		if len(ps.OverlayValues) > 434 && ps.OverlayValues[434].Loc != scm.LocNone {
			d434 = ps.OverlayValues[434]
		}
		if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != scm.LocNone {
			d435 = ps.OverlayValues[435]
		}
		if len(ps.OverlayValues) > 436 && ps.OverlayValues[436].Loc != scm.LocNone {
			d436 = ps.OverlayValues[436]
		}
		if len(ps.OverlayValues) > 437 && ps.OverlayValues[437].Loc != scm.LocNone {
			d437 = ps.OverlayValues[437]
		}
		if len(ps.OverlayValues) > 438 && ps.OverlayValues[438].Loc != scm.LocNone {
			d438 = ps.OverlayValues[438]
		}
		if len(ps.OverlayValues) > 439 && ps.OverlayValues[439].Loc != scm.LocNone {
			d439 = ps.OverlayValues[439]
		}
		if len(ps.OverlayValues) > 440 && ps.OverlayValues[440].Loc != scm.LocNone {
			d440 = ps.OverlayValues[440]
		}
		if len(ps.OverlayValues) > 441 && ps.OverlayValues[441].Loc != scm.LocNone {
			d441 = ps.OverlayValues[441]
		}
		if len(ps.OverlayValues) > 442 && ps.OverlayValues[442].Loc != scm.LocNone {
			d442 = ps.OverlayValues[442]
		}
		if len(ps.OverlayValues) > 443 && ps.OverlayValues[443].Loc != scm.LocNone {
			d443 = ps.OverlayValues[443]
		}
		if len(ps.OverlayValues) > 444 && ps.OverlayValues[444].Loc != scm.LocNone {
			d444 = ps.OverlayValues[444]
		}
		if len(ps.OverlayValues) > 445 && ps.OverlayValues[445].Loc != scm.LocNone {
			d445 = ps.OverlayValues[445]
		}
		if len(ps.OverlayValues) > 446 && ps.OverlayValues[446].Loc != scm.LocNone {
			d446 = ps.OverlayValues[446]
		}
		if len(ps.OverlayValues) > 447 && ps.OverlayValues[447].Loc != scm.LocNone {
			d447 = ps.OverlayValues[447]
		}
		if len(ps.OverlayValues) > 448 && ps.OverlayValues[448].Loc != scm.LocNone {
			d448 = ps.OverlayValues[448]
		}
		if len(ps.OverlayValues) > 449 && ps.OverlayValues[449].Loc != scm.LocNone {
			d449 = ps.OverlayValues[449]
		}
		if len(ps.OverlayValues) > 450 && ps.OverlayValues[450].Loc != scm.LocNone {
			d450 = ps.OverlayValues[450]
		}
		if len(ps.OverlayValues) > 451 && ps.OverlayValues[451].Loc != scm.LocNone {
			d451 = ps.OverlayValues[451]
		}
		if len(ps.OverlayValues) > 452 && ps.OverlayValues[452].Loc != scm.LocNone {
			d452 = ps.OverlayValues[452]
		}
		if len(ps.OverlayValues) > 453 && ps.OverlayValues[453].Loc != scm.LocNone {
			d453 = ps.OverlayValues[453]
		}
		if len(ps.OverlayValues) > 454 && ps.OverlayValues[454].Loc != scm.LocNone {
			d454 = ps.OverlayValues[454]
		}
		if len(ps.OverlayValues) > 455 && ps.OverlayValues[455].Loc != scm.LocNone {
			d455 = ps.OverlayValues[455]
		}
		if len(ps.OverlayValues) > 456 && ps.OverlayValues[456].Loc != scm.LocNone {
			d456 = ps.OverlayValues[456]
		}
		if len(ps.OverlayValues) > 457 && ps.OverlayValues[457].Loc != scm.LocNone {
			d457 = ps.OverlayValues[457]
		}
		if len(ps.OverlayValues) > 458 && ps.OverlayValues[458].Loc != scm.LocNone {
			d458 = ps.OverlayValues[458]
		}
		if len(ps.OverlayValues) > 459 && ps.OverlayValues[459].Loc != scm.LocNone {
			d459 = ps.OverlayValues[459]
		}
		if len(ps.OverlayValues) > 626 && ps.OverlayValues[626].Loc != scm.LocNone {
			d626 = ps.OverlayValues[626]
		}
		if len(ps.OverlayValues) > 627 && ps.OverlayValues[627].Loc != scm.LocNone {
			d627 = ps.OverlayValues[627]
		}
		if len(ps.OverlayValues) > 628 && ps.OverlayValues[628].Loc != scm.LocNone {
			d628 = ps.OverlayValues[628]
		}
		if len(ps.OverlayValues) > 630 && ps.OverlayValues[630].Loc != scm.LocNone {
			d630 = ps.OverlayValues[630]
		}
		if len(ps.OverlayValues) > 631 && ps.OverlayValues[631].Loc != scm.LocNone {
			d631 = ps.OverlayValues[631]
		}
		if len(ps.OverlayValues) > 632 && ps.OverlayValues[632].Loc != scm.LocNone {
			d632 = ps.OverlayValues[632]
		}
		if len(ps.OverlayValues) > 633 && ps.OverlayValues[633].Loc != scm.LocNone {
			d633 = ps.OverlayValues[633]
		}
		if len(ps.OverlayValues) > 634 && ps.OverlayValues[634].Loc != scm.LocNone {
			d634 = ps.OverlayValues[634]
		}
		if len(ps.OverlayValues) > 635 && ps.OverlayValues[635].Loc != scm.LocNone {
			d635 = ps.OverlayValues[635]
		}
		if len(ps.OverlayValues) > 636 && ps.OverlayValues[636].Loc != scm.LocNone {
			d636 = ps.OverlayValues[636]
		}
		if len(ps.OverlayValues) > 638 && ps.OverlayValues[638].Loc != scm.LocNone {
			d638 = ps.OverlayValues[638]
		}
		if len(ps.OverlayValues) > 640 && ps.OverlayValues[640].Loc != scm.LocNone {
			d640 = ps.OverlayValues[640]
		}
		if len(ps.OverlayValues) > 641 && ps.OverlayValues[641].Loc != scm.LocNone {
			d641 = ps.OverlayValues[641]
		}
		if len(ps.OverlayValues) > 642 && ps.OverlayValues[642].Loc != scm.LocNone {
			d642 = ps.OverlayValues[642]
		}
		if len(ps.OverlayValues) > 643 && ps.OverlayValues[643].Loc != scm.LocNone {
			d643 = ps.OverlayValues[643]
		}
		if len(ps.OverlayValues) > 646 && ps.OverlayValues[646].Loc != scm.LocNone {
			d646 = ps.OverlayValues[646]
		}
		if len(ps.OverlayValues) > 825 && ps.OverlayValues[825].Loc != scm.LocNone {
			d825 = ps.OverlayValues[825]
		}
		if len(ps.OverlayValues) > 826 && ps.OverlayValues[826].Loc != scm.LocNone {
			d826 = ps.OverlayValues[826]
		}
		if len(ps.OverlayValues) > 827 && ps.OverlayValues[827].Loc != scm.LocNone {
			d827 = ps.OverlayValues[827]
		}
		if len(ps.OverlayValues) > 828 && ps.OverlayValues[828].Loc != scm.LocNone {
			d828 = ps.OverlayValues[828]
		}
		if len(ps.OverlayValues) > 830 && ps.OverlayValues[830].Loc != scm.LocNone {
			d830 = ps.OverlayValues[830]
		}
		if len(ps.OverlayValues) > 831 && ps.OverlayValues[831].Loc != scm.LocNone {
			d831 = ps.OverlayValues[831]
		}
		if len(ps.OverlayValues) > 832 && ps.OverlayValues[832].Loc != scm.LocNone {
			d832 = ps.OverlayValues[832]
		}
		if len(ps.OverlayValues) > 833 && ps.OverlayValues[833].Loc != scm.LocNone {
			d833 = ps.OverlayValues[833]
		}
		if len(ps.OverlayValues) > 834 && ps.OverlayValues[834].Loc != scm.LocNone {
			d834 = ps.OverlayValues[834]
		}
		if len(ps.OverlayValues) > 835 && ps.OverlayValues[835].Loc != scm.LocNone {
			d835 = ps.OverlayValues[835]
		}
		if len(ps.OverlayValues) > 836 && ps.OverlayValues[836].Loc != scm.LocNone {
			d836 = ps.OverlayValues[836]
		}
		if len(ps.OverlayValues) > 837 && ps.OverlayValues[837].Loc != scm.LocNone {
			d837 = ps.OverlayValues[837]
		}
		if len(ps.OverlayValues) > 839 && ps.OverlayValues[839].Loc != scm.LocNone {
			d839 = ps.OverlayValues[839]
		}
		if len(ps.OverlayValues) > 840 && ps.OverlayValues[840].Loc != scm.LocNone {
			d840 = ps.OverlayValues[840]
		}
		if len(ps.OverlayValues) > 841 && ps.OverlayValues[841].Loc != scm.LocNone {
			d841 = ps.OverlayValues[841]
		}
		if len(ps.OverlayValues) > 842 && ps.OverlayValues[842].Loc != scm.LocNone {
			d842 = ps.OverlayValues[842]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d4)
		d843 = d4
		_ = d843
		ctx.StabilizeDescForControlFlow(&d843)
		r162 := d4.Loc == scm.LocReg || d4.Loc == scm.LocRegPair || d4.Loc == scm.LocRegTriple
		r163 := d4.Reg
		if r162 {
			ctx.ProtectReg(r163)
		}
		r164 := d4.Loc == scm.LocRegPair || d4.Loc == scm.LocRegTriple
		r165 := d4.Reg2
		if r164 {
			ctx.ProtectReg(r165)
		}
		r166 := d4.Loc == scm.LocRegTriple
		r167 := d4.Reg3
		if r166 {
			ctx.ProtectReg(r167)
		}
		phiBase844 = ctx.AllocStack(int32(16))
		d845 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase844) + int32(0)}
		_ = d845
		lbl40 := ctx.ReserveLabel()
		bbpos_4_0 := int32(-1)
		_ = bbpos_4_0
		bbpos_4_1 := int32(-1)
		_ = bbpos_4_1
		bbpos_4_2 := int32(-1)
		_ = bbpos_4_2
		bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		d845 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d843)
		ctx.EnsureDesc(&d843)
		var d846 scm.JITValueDesc
		if d843.Loc == scm.LocImm {
			d846 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d843.Imm.Int()))))}
		} else {
			r168 := ctx.AllocReg()
			ctx.EmitMovRegReg(r168, d843.Reg)
			ctx.EmitShlRegImm8(r168, 32)
			ctx.EmitShrRegImm8(r168, 32)
			d846 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r168}
			ctx.BindReg(r168, &d846)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d847 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
			r169 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r169, fieldAddr)
			d847 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r169}
			ctx.BindReg(r169, &d847)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
			r170 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r170, thisptr.Reg, off)
			d847 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r170}
			ctx.BindReg(r170, &d847)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d847)
		ctx.EnsureDesc(&d847)
		var d848 scm.JITValueDesc
		if d847.Loc == scm.LocImm {
			d848 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d847.Imm.Int()))))}
		} else {
			r171 := ctx.AllocReg()
			ctx.EmitMovRegReg(r171, d847.Reg)
			ctx.EmitShlRegImm8(r171, 56)
			ctx.EmitShrRegImm8(r171, 56)
			d848 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r171}
			ctx.BindReg(r171, &d848)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d846)
		ctx.EnsureDesc(&d848)
		ctx.EnsureDesc(&d846)
		ctx.ProtectReg(d846.Reg)
		ctx.EnsureDesc(&d848)
		ctx.UnprotectReg(d846.Reg)
		var d849 scm.JITValueDesc
		if d846.Loc == scm.LocImm && d848.Loc == scm.LocImm {
			d849 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d846.Imm.Int() * d848.Imm.Int())}
		} else if d846.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d848.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d846.Imm.Int()))
			ctx.EmitImulInt64(scratch, d848.Reg)
			d849 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d849)
		} else if d848.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d846.Reg)
			ctx.EmitMovRegReg(scratch, d846.Reg)
			if d848.Imm.Int() >= -2147483648 && d848.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d848.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d848.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d849 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d849)
		} else {
			r172 := ctx.AllocRegExcept(d846.Reg, d848.Reg)
			ctx.EmitMovRegReg(r172, d846.Reg)
			ctx.EmitImulInt64(r172, d848.Reg)
			d849 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
			ctx.BindReg(r172, &d849)
		}
		if d849.Loc == scm.LocReg && d846.Loc == scm.LocReg && d849.Reg == d846.Reg {
			ctx.TransferReg(d846.Reg)
			d846.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d849)
		ctx.FreeDesc(&d846)
		ctx.FreeDesc(&d848)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d850 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 0
			r173 := ctx.AllocReg()
			r174 := ctx.AllocRegExcept(r173)
			r175 := ctx.AllocRegExcept(r173, r174)
			ctx.EmitMovRegMem64(r173, fieldAddr)
			ctx.EmitMovRegMem64(r174, fieldAddr+8)
			ctx.EmitMovRegMem64(r175, fieldAddr+16)
			d850 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r173, Reg2: r174, Reg3: r175}
			ctx.BindReg(r173, &d850)
			ctx.BindReg(r174, &d850)
			ctx.BindReg(r175, &d850)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 0)
			r176 := ctx.AllocReg()
			r177 := ctx.AllocRegExcept(r176)
			r178 := ctx.AllocRegExcept(r176, r177)
			ctx.EmitMovRegMem(r176, thisptr.Reg, off)
			ctx.EmitMovRegMem(r177, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r178, thisptr.Reg, off+16)
			d850 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r176, Reg2: r177, Reg3: r178}
			ctx.BindReg(r176, &d850)
			ctx.BindReg(r177, &d850)
			ctx.BindReg(r178, &d850)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d849)
		var d851 scm.JITValueDesc
		if d849.Loc == scm.LocImm {
			d851 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d849.Imm.Int() / 64)}
		} else {
			r179 := ctx.AllocRegExcept(d849.Reg)
			ctx.EmitMovRegReg(r179, d849.Reg)
			ctx.EmitShrRegImm8(r179, 6)
			d851 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r179}
			ctx.BindReg(r179, &d851)
		}
		if d851.Loc == scm.LocReg && d849.Loc == scm.LocReg && d851.Reg == d849.Reg {
			ctx.TransferReg(d849.Reg)
			d849.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d851)
		ctx.ReclaimUntrackedRegs()
		d853 = ctx.EmitSliceElementAddress(&d850, &d851, 8)
		ctx.EnsureDesc(&d853)
		ctx.EmitMovRegMem(d853.Reg, d853.Reg, 0)
		d852 = d853
		ctx.FreeDesc(&d851)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d849)
		var d854 scm.JITValueDesc
		if d849.Loc == scm.LocImm {
			d854 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d849.Imm.Int() % 64)}
		} else {
			r180 := ctx.AllocRegExcept(d849.Reg)
			ctx.EmitMovRegReg(r180, d849.Reg)
			ctx.EmitAndRegImm32(r180, 63)
			d854 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r180}
			ctx.BindReg(r180, &d854)
		}
		if d854.Loc == scm.LocReg && d849.Loc == scm.LocReg && d854.Reg == d849.Reg {
			ctx.TransferReg(d849.Reg)
			d849.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d852)
		ctx.EnsureDesc(&d854)
		var d855 scm.JITValueDesc
		if d852.Loc == scm.LocImm && d854.Loc == scm.LocImm {
			d855 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d852.Imm.Int()) << uint64(d854.Imm.Int())))}
		} else if d854.Loc == scm.LocImm {
			r181 := ctx.AllocRegExcept(d852.Reg)
			ctx.EmitMovRegReg(r181, d852.Reg)
			ctx.EmitShlRegImm8(r181, uint8(d854.Imm.Int()))
			d855 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r181}
			ctx.BindReg(r181, &d855)
		} else {
			{
				shiftSrc := d852.Reg
				r182 := ctx.AllocRegExcept(d852.Reg)
				ctx.EmitMovRegReg(r182, d852.Reg)
				shiftSrc = r182
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d854.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d854.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d854.Reg)
				}
				ctx.EmitShlRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d855 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d855)
			}
		}
		if d855.Loc == scm.LocReg && d852.Loc == scm.LocReg && d855.Reg == d852.Reg {
			ctx.TransferReg(d852.Reg)
			d852.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d855)
		ctx.EmitStoreToStack(d855, int32(phiBase844)+int32(0))
		ctx.StabilizeDescForControlFlow(&d855)
		ctx.FreeDesc(&d852)
		ctx.FreeDesc(&d854)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d849)
		var d856 scm.JITValueDesc
		if d849.Loc == scm.LocImm {
			d856 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d849.Imm.Int() % 64)}
		} else {
			r183 := ctx.AllocRegExcept(d849.Reg)
			ctx.EmitMovRegReg(r183, d849.Reg)
			ctx.EmitAndRegImm32(r183, 63)
			d856 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r183}
			ctx.BindReg(r183, &d856)
		}
		if d856.Loc == scm.LocReg && d849.Loc == scm.LocReg && d856.Reg == d849.Reg {
			ctx.TransferReg(d849.Reg)
			d849.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d847)
		ctx.EnsureDesc(&d847)
		var d857 scm.JITValueDesc
		if d847.Loc == scm.LocImm {
			d857 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d847.Imm.Int()))))}
		} else {
			r184 := ctx.AllocReg()
			ctx.EmitMovRegReg(r184, d847.Reg)
			ctx.EmitShlRegImm8(r184, 56)
			ctx.EmitShrRegImm8(r184, 56)
			d857 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r184}
			ctx.BindReg(r184, &d857)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d856)
		ctx.EnsureDesc(&d857)
		ctx.EnsureDesc(&d856)
		ctx.ProtectReg(d856.Reg)
		ctx.EnsureDesc(&d857)
		ctx.UnprotectReg(d856.Reg)
		var d858 scm.JITValueDesc
		if d856.Loc == scm.LocImm && d857.Loc == scm.LocImm {
			d858 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d856.Imm.Int() + d857.Imm.Int())}
		} else if d857.Loc == scm.LocImm && d857.Imm.Int() == 0 {
			r185 := ctx.AllocRegExcept(d856.Reg)
			ctx.EmitMovRegReg(r185, d856.Reg)
			d858 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r185}
			ctx.BindReg(r185, &d858)
		} else if d856.Loc == scm.LocImm && d856.Imm.Int() == 0 {
			d858 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d857.Reg}
			ctx.BindReg(d857.Reg, &d858)
		} else if d856.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d857.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d856.Imm.Int()))
			ctx.EmitAddInt64(scratch, d857.Reg)
			d858 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d858)
		} else if d857.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d856.Reg)
			ctx.EmitMovRegReg(scratch, d856.Reg)
			if d857.Imm.Int() >= -2147483648 && d857.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d857.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d857.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d858 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d858)
		} else {
			r186 := ctx.AllocRegExcept(d856.Reg, d857.Reg)
			ctx.EmitMovRegReg(r186, d856.Reg)
			ctx.EmitAddInt64(r186, d857.Reg)
			d858 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r186}
			ctx.BindReg(r186, &d858)
		}
		if d858.Loc == scm.LocReg && d856.Loc == scm.LocReg && d858.Reg == d856.Reg {
			ctx.TransferReg(d856.Reg)
			d856.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d856)
		ctx.FreeDesc(&d857)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d858)
		var d859 scm.JITValueDesc
		if d858.Loc == scm.LocImm {
			d859 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d858.Imm.Int()) > uint64(0x40))}
		} else {
			r187 := ctx.AllocRegExcept(d858.Reg)
			ctx.EmitCmpRegImm32(d858.Reg, 64)
			ctx.EmitSetcc(r187, scm.CondUnsignedAbove)
			d859 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r187}
			ctx.BindReg(r187, &d859)
		}
		ctx.FreeDesc(&d858)
		ctx.ReclaimUntrackedRegs()
		d860 = d859
		ctx.EnsureDesc(&d860)
		if d860.Loc != scm.LocImm && d860.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		lbl41 := ctx.ReserveLabel()
		lbl42 := ctx.ReserveLabel()
		lbl43 := ctx.ReserveLabel()
		lbl44 := ctx.ReserveLabel()
		if d860.Loc == scm.LocImm {
			if d860.Imm.Bool() {
				ctx.MarkLabel(lbl43)
				ctx.EmitJmp(lbl41)
			} else {
				ctx.MarkLabel(lbl44)
				ctx.EmitJmp(lbl42)
			}
		} else {
			ctx.EmitCmpRegImm32(d860.Reg, 0)
			ctx.EmitJump(scm.CondNotEqual, lbl43)
			ctx.EmitJmp(lbl44)
			ctx.MarkLabel(lbl43)
			ctx.EmitJmp(lbl41)
			ctx.MarkLabel(lbl44)
			ctx.EmitJmp(lbl42)
		}
		ctx.FreeDesc(&d859)
		bbpos_4_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl42)
		ctx.ResolveFixups()
		d845 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d847)
		ctx.EnsureDesc(&d847)
		var d861 scm.JITValueDesc
		if d847.Loc == scm.LocImm {
			d861 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d847.Imm.Int()))))}
		} else {
			r188 := ctx.AllocReg()
			ctx.EmitMovRegReg(r188, d847.Reg)
			ctx.EmitShlRegImm8(r188, 56)
			ctx.EmitShrRegImm8(r188, 56)
			d861 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r188}
			ctx.BindReg(r188, &d861)
		}
		ctx.ReclaimUntrackedRegs()
		d862 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d861)
		ctx.EnsureDesc(&d862)
		ctx.ProtectReg(d862.Reg)
		ctx.EnsureDesc(&d861)
		ctx.UnprotectReg(d862.Reg)
		var d863 scm.JITValueDesc
		if d862.Loc == scm.LocImm && d861.Loc == scm.LocImm {
			d863 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d862.Imm.Int() - d861.Imm.Int())}
		} else if d861.Loc == scm.LocImm && d861.Imm.Int() == 0 {
			r189 := ctx.AllocRegExcept(d862.Reg)
			ctx.EmitMovRegReg(r189, d862.Reg)
			d863 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r189}
			ctx.BindReg(r189, &d863)
		} else if d862.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d861.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d862.Imm.Int()))
			ctx.EmitSubInt64(scratch, d861.Reg)
			d863 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d863)
		} else if d861.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d862.Reg)
			ctx.EmitMovRegReg(scratch, d862.Reg)
			if d861.Imm.Int() >= -2147483648 && d861.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d861.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d861.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d863 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d863)
		} else {
			r190 := ctx.AllocRegExcept(d862.Reg, d861.Reg)
			ctx.EmitMovRegReg(r190, d862.Reg)
			ctx.EmitSubInt64(r190, d861.Reg)
			d863 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r190}
			ctx.BindReg(r190, &d863)
		}
		if d863.Loc == scm.LocReg && d862.Loc == scm.LocReg && d863.Reg == d862.Reg {
			ctx.TransferReg(d862.Reg)
			d862.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d861)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d845)
		ctx.EnsureDesc(&d863)
		var d864 scm.JITValueDesc
		if d845.Loc == scm.LocImm && d863.Loc == scm.LocImm {
			d864 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d845.Imm.Int()) >> uint64(d863.Imm.Int())))}
		} else if d863.Loc == scm.LocImm {
			r191 := ctx.AllocRegExcept(d845.Reg)
			ctx.EmitMovRegReg(r191, d845.Reg)
			ctx.EmitShrRegImm8(r191, uint8(d863.Imm.Int()))
			d864 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r191}
			ctx.BindReg(r191, &d864)
		} else {
			{
				shiftSrc := d845.Reg
				r192 := ctx.AllocRegExcept(d845.Reg)
				ctx.EmitMovRegReg(r192, d845.Reg)
				shiftSrc = r192
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d863.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d863.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d863.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d864 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d864)
			}
		}
		if d864.Loc == scm.LocReg && d845.Loc == scm.LocReg && d864.Reg == d845.Reg {
			ctx.TransferReg(d845.Reg)
			d845.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d845)
		ctx.FreeDesc(&d863)
		ctx.ReclaimUntrackedRegs()
		r193 := ctx.AllocReg()
		ctx.EnsureDesc(&d864)
		ctx.EnsureDesc(&d864)
		if d864.Loc == scm.LocRegPair {
			panic("jit: scalar inline return has scm.LocRegPair")
		} else {
			ctx.EmitMovToReg(r193, d864)
		}
		ctx.EmitJmp(lbl40)
		bbpos_4_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl41)
		ctx.ResolveFixups()
		d845 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d849)
		var d865 scm.JITValueDesc
		if d849.Loc == scm.LocImm {
			d865 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d849.Imm.Int() / 64)}
		} else {
			r194 := ctx.AllocRegExcept(d849.Reg)
			ctx.EmitMovRegReg(r194, d849.Reg)
			ctx.EmitShrRegImm8(r194, 6)
			d865 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r194}
			ctx.BindReg(r194, &d865)
		}
		if d865.Loc == scm.LocReg && d849.Loc == scm.LocReg && d865.Reg == d849.Reg {
			ctx.TransferReg(d849.Reg)
			d849.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d865)
		ctx.EnsureDesc(&d865)
		var d866 scm.JITValueDesc
		if d865.Loc == scm.LocImm {
			d866 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d865.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d865.Reg)
			ctx.EmitMovRegReg(scratch, d865.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d866 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d866)
		}
		if d866.Loc == scm.LocReg && d865.Loc == scm.LocReg && d866.Reg == d865.Reg {
			ctx.TransferReg(d865.Reg)
			d865.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d865)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d866)
		ctx.ReclaimUntrackedRegs()
		d868 = ctx.EmitSliceElementAddress(&d850, &d866, 8)
		ctx.EnsureDesc(&d868)
		ctx.EmitMovRegMem(d868.Reg, d868.Reg, 0)
		d867 = d868
		ctx.FreeDesc(&d866)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d849)
		var d869 scm.JITValueDesc
		if d849.Loc == scm.LocImm {
			d869 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d849.Imm.Int() % 64)}
		} else {
			r195 := ctx.AllocRegExcept(d849.Reg)
			ctx.EmitMovRegReg(r195, d849.Reg)
			ctx.EmitAndRegImm32(r195, 63)
			d869 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r195}
			ctx.BindReg(r195, &d869)
		}
		if d869.Loc == scm.LocReg && d849.Loc == scm.LocReg && d869.Reg == d849.Reg {
			ctx.TransferReg(d849.Reg)
			d849.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d849)
		ctx.ReclaimUntrackedRegs()
		d870 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d869)
		ctx.EnsureDesc(&d870)
		ctx.ProtectReg(d870.Reg)
		ctx.EnsureDesc(&d869)
		ctx.UnprotectReg(d870.Reg)
		var d871 scm.JITValueDesc
		if d870.Loc == scm.LocImm && d869.Loc == scm.LocImm {
			d871 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d870.Imm.Int() - d869.Imm.Int())}
		} else if d869.Loc == scm.LocImm && d869.Imm.Int() == 0 {
			r196 := ctx.AllocRegExcept(d870.Reg)
			ctx.EmitMovRegReg(r196, d870.Reg)
			d871 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r196}
			ctx.BindReg(r196, &d871)
		} else if d870.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d869.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d870.Imm.Int()))
			ctx.EmitSubInt64(scratch, d869.Reg)
			d871 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d871)
		} else if d869.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d870.Reg)
			ctx.EmitMovRegReg(scratch, d870.Reg)
			if d869.Imm.Int() >= -2147483648 && d869.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d869.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d869.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d871 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d871)
		} else {
			r197 := ctx.AllocRegExcept(d870.Reg, d869.Reg)
			ctx.EmitMovRegReg(r197, d870.Reg)
			ctx.EmitSubInt64(r197, d869.Reg)
			d871 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r197}
			ctx.BindReg(r197, &d871)
		}
		if d871.Loc == scm.LocReg && d870.Loc == scm.LocReg && d871.Reg == d870.Reg {
			ctx.TransferReg(d870.Reg)
			d870.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d869)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d867)
		ctx.EnsureDesc(&d871)
		var d872 scm.JITValueDesc
		if d867.Loc == scm.LocImm && d871.Loc == scm.LocImm {
			d872 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d867.Imm.Int()) >> uint64(d871.Imm.Int())))}
		} else if d871.Loc == scm.LocImm {
			r198 := ctx.AllocRegExcept(d867.Reg)
			ctx.EmitMovRegReg(r198, d867.Reg)
			ctx.EmitShrRegImm8(r198, uint8(d871.Imm.Int()))
			d872 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r198}
			ctx.BindReg(r198, &d872)
		} else {
			{
				shiftSrc := d867.Reg
				r199 := ctx.AllocRegExcept(d867.Reg)
				ctx.EmitMovRegReg(r199, d867.Reg)
				shiftSrc = r199
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d871.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d871.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d871.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d872 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d872)
			}
		}
		if d872.Loc == scm.LocReg && d867.Loc == scm.LocReg && d872.Reg == d867.Reg {
			ctx.TransferReg(d867.Reg)
			d867.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d867)
		ctx.FreeDesc(&d871)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d855)
		ctx.EnsureDesc(&d872)
		var d873 scm.JITValueDesc
		if d855.Loc == scm.LocImm && d872.Loc == scm.LocImm {
			d873 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d855.Imm.Int() | d872.Imm.Int())}
		} else if d855.Loc == scm.LocImm && d855.Imm.Int() == 0 {
			d873 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d872.Reg}
			ctx.BindReg(d872.Reg, &d873)
		} else if d872.Loc == scm.LocImm && d872.Imm.Int() == 0 {
			r200 := ctx.AllocRegExcept(d855.Reg)
			ctx.EmitMovRegReg(r200, d855.Reg)
			d873 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r200}
			ctx.BindReg(r200, &d873)
		} else if d855.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d872.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d855.Imm.Int()))
			ctx.EmitOrInt64(scratch, d872.Reg)
			d873 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d873)
		} else if d872.Loc == scm.LocImm {
			r201 := ctx.AllocRegExcept(d855.Reg)
			ctx.EmitMovRegReg(r201, d855.Reg)
			if d872.Imm.Int() >= -2147483648 && d872.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r201, int32(d872.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d872.Imm.Int()))
				ctx.EmitOrInt64(r201, scm.RegR11)
			}
			d873 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r201}
			ctx.BindReg(r201, &d873)
		} else {
			r202 := ctx.AllocRegExcept(d855.Reg, d872.Reg)
			ctx.EmitMovRegReg(r202, d855.Reg)
			ctx.EmitOrInt64(r202, d872.Reg)
			d873 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r202}
			ctx.BindReg(r202, &d873)
		}
		if d873.Loc == scm.LocReg && d855.Loc == scm.LocReg && d873.Reg == d855.Reg {
			ctx.TransferReg(d855.Reg)
			d855.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d873)
		ctx.EmitStoreToStack(d873, int32(phiBase844)+int32(0))
		ctx.StabilizeDescForControlFlow(&d873)
		ctx.FreeDesc(&d872)
		ctx.ReclaimUntrackedRegs()
		ctx.EmitJmp(lbl42)
		ctx.MarkLabel(lbl40)
		d874 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r193}
		ctx.BindReg(r193, &d874)
		ctx.BindReg(r193, &d874)
		if r162 {
			ctx.UnprotectReg(r163)
		}
		if r164 {
			ctx.UnprotectReg(r165)
		}
		if r166 {
			ctx.UnprotectReg(r167)
		}
		ctx.EnsureDesc(&d874)
		ctx.EnsureDesc(&d874)
		var d875 scm.JITValueDesc
		if d874.Loc == scm.LocImm {
			d875 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d874.Imm.Int()))))}
		} else {
			r203 := ctx.AllocReg()
			ctx.EmitMovRegReg(r203, d874.Reg)
			d875 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r203}
			ctx.BindReg(r203, &d875)
		}
		ctx.FreeDesc(&d874)
		var d876 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 32
			r204 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r204, fieldAddr)
			d876 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r204}
			ctx.BindReg(r204, &d876)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 32)
			r205 := ctx.AllocReg()
			ctx.EmitMovRegMem(r205, thisptr.Reg, off)
			d876 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r205}
			ctx.BindReg(r205, &d876)
		}
		ctx.EnsureDesc(&d875)
		ctx.EnsureDesc(&d876)
		ctx.EnsureDesc(&d875)
		ctx.ProtectReg(d875.Reg)
		ctx.EnsureDesc(&d876)
		ctx.UnprotectReg(d875.Reg)
		var d877 scm.JITValueDesc
		if d875.Loc == scm.LocImm && d876.Loc == scm.LocImm {
			d877 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d875.Imm.Int() + d876.Imm.Int())}
		} else if d876.Loc == scm.LocImm && d876.Imm.Int() == 0 {
			r206 := ctx.AllocRegExcept(d875.Reg)
			ctx.EmitMovRegReg(r206, d875.Reg)
			d877 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r206}
			ctx.BindReg(r206, &d877)
		} else if d875.Loc == scm.LocImm && d875.Imm.Int() == 0 {
			d877 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d876.Reg}
			ctx.BindReg(d876.Reg, &d877)
		} else if d875.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d876.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d875.Imm.Int()))
			ctx.EmitAddInt64(scratch, d876.Reg)
			d877 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d877)
		} else if d876.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d875.Reg)
			ctx.EmitMovRegReg(scratch, d875.Reg)
			if d876.Imm.Int() >= -2147483648 && d876.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d876.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d876.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d877 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d877)
		} else {
			r207 := ctx.AllocRegExcept(d875.Reg, d876.Reg)
			ctx.EmitMovRegReg(r207, d875.Reg)
			ctx.EmitAddInt64(r207, d876.Reg)
			d877 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r207}
			ctx.BindReg(r207, &d877)
		}
		if d877.Loc == scm.LocReg && d875.Loc == scm.LocReg && d877.Reg == d875.Reg {
			ctx.TransferReg(d875.Reg)
			d875.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d875)
		ctx.EnsureDesc(&d4)
		d878 = d4
		_ = d878
		ctx.StabilizeDescForControlFlow(&d878)
		phiBase879 = ctx.AllocStack(int32(16))
		d880 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase879) + int32(0)}
		_ = d880
		lbl45 := ctx.ReserveLabel()
		bbpos_5_0 := int32(-1)
		_ = bbpos_5_0
		bbpos_5_1 := int32(-1)
		_ = bbpos_5_1
		bbpos_5_2 := int32(-1)
		_ = bbpos_5_2
		bbpos_5_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		d880 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d878)
		ctx.EnsureDesc(&d878)
		var d881 scm.JITValueDesc
		if d878.Loc == scm.LocImm {
			d881 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d878.Imm.Int()))))}
		} else {
			r208 := ctx.AllocReg()
			ctx.EmitMovRegReg(r208, d878.Reg)
			ctx.EmitShlRegImm8(r208, 32)
			ctx.EmitShrRegImm8(r208, 32)
			d881 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r208}
			ctx.BindReg(r208, &d881)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d882 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			r209 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r209, fieldAddr)
			d882 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r209}
			ctx.BindReg(r209, &d882)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			r210 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r210, thisptr.Reg, off)
			d882 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r210}
			ctx.BindReg(r210, &d882)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d882)
		ctx.EnsureDesc(&d882)
		var d883 scm.JITValueDesc
		if d882.Loc == scm.LocImm {
			d883 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d882.Imm.Int()))))}
		} else {
			r211 := ctx.AllocReg()
			ctx.EmitMovRegReg(r211, d882.Reg)
			ctx.EmitShlRegImm8(r211, 56)
			ctx.EmitShrRegImm8(r211, 56)
			d883 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r211}
			ctx.BindReg(r211, &d883)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d881)
		ctx.EnsureDesc(&d883)
		ctx.EnsureDesc(&d881)
		ctx.ProtectReg(d881.Reg)
		ctx.EnsureDesc(&d883)
		ctx.UnprotectReg(d881.Reg)
		var d884 scm.JITValueDesc
		if d881.Loc == scm.LocImm && d883.Loc == scm.LocImm {
			d884 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d881.Imm.Int() * d883.Imm.Int())}
		} else if d881.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d883.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d881.Imm.Int()))
			ctx.EmitImulInt64(scratch, d883.Reg)
			d884 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d884)
		} else if d883.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d881.Reg)
			ctx.EmitMovRegReg(scratch, d881.Reg)
			if d883.Imm.Int() >= -2147483648 && d883.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d883.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d883.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d884 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d884)
		} else {
			r212 := ctx.AllocRegExcept(d881.Reg, d883.Reg)
			ctx.EmitMovRegReg(r212, d881.Reg)
			ctx.EmitImulInt64(r212, d883.Reg)
			d884 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r212}
			ctx.BindReg(r212, &d884)
		}
		if d884.Loc == scm.LocReg && d881.Loc == scm.LocReg && d884.Reg == d881.Reg {
			ctx.TransferReg(d881.Reg)
			d881.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d884)
		ctx.FreeDesc(&d881)
		ctx.FreeDesc(&d883)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d885 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
			r213 := ctx.AllocReg()
			r214 := ctx.AllocRegExcept(r213)
			r215 := ctx.AllocRegExcept(r213, r214)
			ctx.EmitMovRegMem64(r213, fieldAddr)
			ctx.EmitMovRegMem64(r214, fieldAddr+8)
			ctx.EmitMovRegMem64(r215, fieldAddr+16)
			d885 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r213, Reg2: r214, Reg3: r215}
			ctx.BindReg(r213, &d885)
			ctx.BindReg(r214, &d885)
			ctx.BindReg(r215, &d885)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
			r216 := ctx.AllocReg()
			r217 := ctx.AllocRegExcept(r216)
			r218 := ctx.AllocRegExcept(r216, r217)
			ctx.EmitMovRegMem(r216, thisptr.Reg, off)
			ctx.EmitMovRegMem(r217, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r218, thisptr.Reg, off+16)
			d885 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r216, Reg2: r217, Reg3: r218}
			ctx.BindReg(r216, &d885)
			ctx.BindReg(r217, &d885)
			ctx.BindReg(r218, &d885)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d884)
		var d886 scm.JITValueDesc
		if d884.Loc == scm.LocImm {
			d886 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d884.Imm.Int() / 64)}
		} else {
			r219 := ctx.AllocRegExcept(d884.Reg)
			ctx.EmitMovRegReg(r219, d884.Reg)
			ctx.EmitShrRegImm8(r219, 6)
			d886 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r219}
			ctx.BindReg(r219, &d886)
		}
		if d886.Loc == scm.LocReg && d884.Loc == scm.LocReg && d886.Reg == d884.Reg {
			ctx.TransferReg(d884.Reg)
			d884.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d886)
		ctx.ReclaimUntrackedRegs()
		d888 = ctx.EmitSliceElementAddress(&d885, &d886, 8)
		ctx.EnsureDesc(&d888)
		ctx.EmitMovRegMem(d888.Reg, d888.Reg, 0)
		d887 = d888
		ctx.FreeDesc(&d886)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d884)
		var d889 scm.JITValueDesc
		if d884.Loc == scm.LocImm {
			d889 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d884.Imm.Int() % 64)}
		} else {
			r220 := ctx.AllocRegExcept(d884.Reg)
			ctx.EmitMovRegReg(r220, d884.Reg)
			ctx.EmitAndRegImm32(r220, 63)
			d889 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r220}
			ctx.BindReg(r220, &d889)
		}
		if d889.Loc == scm.LocReg && d884.Loc == scm.LocReg && d889.Reg == d884.Reg {
			ctx.TransferReg(d884.Reg)
			d884.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d887)
		ctx.EnsureDesc(&d889)
		var d890 scm.JITValueDesc
		if d887.Loc == scm.LocImm && d889.Loc == scm.LocImm {
			d890 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d887.Imm.Int()) << uint64(d889.Imm.Int())))}
		} else if d889.Loc == scm.LocImm {
			r221 := ctx.AllocRegExcept(d887.Reg)
			ctx.EmitMovRegReg(r221, d887.Reg)
			ctx.EmitShlRegImm8(r221, uint8(d889.Imm.Int()))
			d890 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r221}
			ctx.BindReg(r221, &d890)
		} else {
			{
				shiftSrc := d887.Reg
				r222 := ctx.AllocRegExcept(d887.Reg)
				ctx.EmitMovRegReg(r222, d887.Reg)
				shiftSrc = r222
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d889.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d889.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d889.Reg)
				}
				ctx.EmitShlRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d890 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d890)
			}
		}
		if d890.Loc == scm.LocReg && d887.Loc == scm.LocReg && d890.Reg == d887.Reg {
			ctx.TransferReg(d887.Reg)
			d887.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d890)
		ctx.EmitStoreToStack(d890, int32(phiBase879)+int32(0))
		ctx.StabilizeDescForControlFlow(&d890)
		ctx.FreeDesc(&d887)
		ctx.FreeDesc(&d889)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d884)
		var d891 scm.JITValueDesc
		if d884.Loc == scm.LocImm {
			d891 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d884.Imm.Int() % 64)}
		} else {
			r223 := ctx.AllocRegExcept(d884.Reg)
			ctx.EmitMovRegReg(r223, d884.Reg)
			ctx.EmitAndRegImm32(r223, 63)
			d891 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r223}
			ctx.BindReg(r223, &d891)
		}
		if d891.Loc == scm.LocReg && d884.Loc == scm.LocReg && d891.Reg == d884.Reg {
			ctx.TransferReg(d884.Reg)
			d884.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d882)
		ctx.EnsureDesc(&d882)
		var d892 scm.JITValueDesc
		if d882.Loc == scm.LocImm {
			d892 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d882.Imm.Int()))))}
		} else {
			r224 := ctx.AllocReg()
			ctx.EmitMovRegReg(r224, d882.Reg)
			ctx.EmitShlRegImm8(r224, 56)
			ctx.EmitShrRegImm8(r224, 56)
			d892 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r224}
			ctx.BindReg(r224, &d892)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d891)
		ctx.EnsureDesc(&d892)
		ctx.EnsureDesc(&d891)
		ctx.ProtectReg(d891.Reg)
		ctx.EnsureDesc(&d892)
		ctx.UnprotectReg(d891.Reg)
		var d893 scm.JITValueDesc
		if d891.Loc == scm.LocImm && d892.Loc == scm.LocImm {
			d893 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d891.Imm.Int() + d892.Imm.Int())}
		} else if d892.Loc == scm.LocImm && d892.Imm.Int() == 0 {
			r225 := ctx.AllocRegExcept(d891.Reg)
			ctx.EmitMovRegReg(r225, d891.Reg)
			d893 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r225}
			ctx.BindReg(r225, &d893)
		} else if d891.Loc == scm.LocImm && d891.Imm.Int() == 0 {
			d893 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d892.Reg}
			ctx.BindReg(d892.Reg, &d893)
		} else if d891.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d892.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d891.Imm.Int()))
			ctx.EmitAddInt64(scratch, d892.Reg)
			d893 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d893)
		} else if d892.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d891.Reg)
			ctx.EmitMovRegReg(scratch, d891.Reg)
			if d892.Imm.Int() >= -2147483648 && d892.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d892.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d892.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d893 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d893)
		} else {
			r226 := ctx.AllocRegExcept(d891.Reg, d892.Reg)
			ctx.EmitMovRegReg(r226, d891.Reg)
			ctx.EmitAddInt64(r226, d892.Reg)
			d893 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r226}
			ctx.BindReg(r226, &d893)
		}
		if d893.Loc == scm.LocReg && d891.Loc == scm.LocReg && d893.Reg == d891.Reg {
			ctx.TransferReg(d891.Reg)
			d891.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d891)
		ctx.FreeDesc(&d892)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d893)
		var d894 scm.JITValueDesc
		if d893.Loc == scm.LocImm {
			d894 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d893.Imm.Int()) > uint64(0x40))}
		} else {
			r227 := ctx.AllocRegExcept(d893.Reg)
			ctx.EmitCmpRegImm32(d893.Reg, 64)
			ctx.EmitSetcc(r227, scm.CondUnsignedAbove)
			d894 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r227}
			ctx.BindReg(r227, &d894)
		}
		ctx.FreeDesc(&d893)
		ctx.ReclaimUntrackedRegs()
		d895 = d894
		ctx.EnsureDesc(&d895)
		if d895.Loc != scm.LocImm && d895.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		lbl46 := ctx.ReserveLabel()
		lbl47 := ctx.ReserveLabel()
		lbl48 := ctx.ReserveLabel()
		lbl49 := ctx.ReserveLabel()
		if d895.Loc == scm.LocImm {
			if d895.Imm.Bool() {
				ctx.MarkLabel(lbl48)
				ctx.EmitJmp(lbl46)
			} else {
				ctx.MarkLabel(lbl49)
				ctx.EmitJmp(lbl47)
			}
		} else {
			ctx.EmitCmpRegImm32(d895.Reg, 0)
			ctx.EmitJump(scm.CondNotEqual, lbl48)
			ctx.EmitJmp(lbl49)
			ctx.MarkLabel(lbl48)
			ctx.EmitJmp(lbl46)
			ctx.MarkLabel(lbl49)
			ctx.EmitJmp(lbl47)
		}
		ctx.FreeDesc(&d894)
		bbpos_5_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl47)
		ctx.ResolveFixups()
		d880 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d882)
		ctx.EnsureDesc(&d882)
		var d896 scm.JITValueDesc
		if d882.Loc == scm.LocImm {
			d896 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d882.Imm.Int()))))}
		} else {
			r228 := ctx.AllocReg()
			ctx.EmitMovRegReg(r228, d882.Reg)
			ctx.EmitShlRegImm8(r228, 56)
			ctx.EmitShrRegImm8(r228, 56)
			d896 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r228}
			ctx.BindReg(r228, &d896)
		}
		ctx.ReclaimUntrackedRegs()
		d897 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d896)
		ctx.EnsureDesc(&d897)
		ctx.ProtectReg(d897.Reg)
		ctx.EnsureDesc(&d896)
		ctx.UnprotectReg(d897.Reg)
		var d898 scm.JITValueDesc
		if d897.Loc == scm.LocImm && d896.Loc == scm.LocImm {
			d898 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d897.Imm.Int() - d896.Imm.Int())}
		} else if d896.Loc == scm.LocImm && d896.Imm.Int() == 0 {
			r229 := ctx.AllocRegExcept(d897.Reg)
			ctx.EmitMovRegReg(r229, d897.Reg)
			d898 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r229}
			ctx.BindReg(r229, &d898)
		} else if d897.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d896.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d897.Imm.Int()))
			ctx.EmitSubInt64(scratch, d896.Reg)
			d898 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d898)
		} else if d896.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d897.Reg)
			ctx.EmitMovRegReg(scratch, d897.Reg)
			if d896.Imm.Int() >= -2147483648 && d896.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d896.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d896.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d898 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d898)
		} else {
			r230 := ctx.AllocRegExcept(d897.Reg, d896.Reg)
			ctx.EmitMovRegReg(r230, d897.Reg)
			ctx.EmitSubInt64(r230, d896.Reg)
			d898 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r230}
			ctx.BindReg(r230, &d898)
		}
		if d898.Loc == scm.LocReg && d897.Loc == scm.LocReg && d898.Reg == d897.Reg {
			ctx.TransferReg(d897.Reg)
			d897.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d896)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d880)
		ctx.EnsureDesc(&d898)
		var d899 scm.JITValueDesc
		if d880.Loc == scm.LocImm && d898.Loc == scm.LocImm {
			d899 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d880.Imm.Int()) >> uint64(d898.Imm.Int())))}
		} else if d898.Loc == scm.LocImm {
			r231 := ctx.AllocRegExcept(d880.Reg)
			ctx.EmitMovRegReg(r231, d880.Reg)
			ctx.EmitShrRegImm8(r231, uint8(d898.Imm.Int()))
			d899 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r231}
			ctx.BindReg(r231, &d899)
		} else {
			{
				shiftSrc := d880.Reg
				r232 := ctx.AllocRegExcept(d880.Reg)
				ctx.EmitMovRegReg(r232, d880.Reg)
				shiftSrc = r232
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d898.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d898.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d898.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d899 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d899)
			}
		}
		if d899.Loc == scm.LocReg && d880.Loc == scm.LocReg && d899.Reg == d880.Reg {
			ctx.TransferReg(d880.Reg)
			d880.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d880)
		ctx.FreeDesc(&d898)
		ctx.ReclaimUntrackedRegs()
		r233 := ctx.AllocReg()
		ctx.EnsureDesc(&d899)
		ctx.EnsureDesc(&d899)
		if d899.Loc == scm.LocRegPair {
			panic("jit: scalar inline return has scm.LocRegPair")
		} else {
			ctx.EmitMovToReg(r233, d899)
		}
		ctx.EmitJmp(lbl45)
		bbpos_5_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl46)
		ctx.ResolveFixups()
		d880 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d884)
		var d900 scm.JITValueDesc
		if d884.Loc == scm.LocImm {
			d900 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d884.Imm.Int() / 64)}
		} else {
			r234 := ctx.AllocRegExcept(d884.Reg)
			ctx.EmitMovRegReg(r234, d884.Reg)
			ctx.EmitShrRegImm8(r234, 6)
			d900 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
			ctx.BindReg(r234, &d900)
		}
		if d900.Loc == scm.LocReg && d884.Loc == scm.LocReg && d900.Reg == d884.Reg {
			ctx.TransferReg(d884.Reg)
			d884.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d900)
		ctx.EnsureDesc(&d900)
		var d901 scm.JITValueDesc
		if d900.Loc == scm.LocImm {
			d901 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d900.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d900.Reg)
			ctx.EmitMovRegReg(scratch, d900.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d901 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d901)
		}
		if d901.Loc == scm.LocReg && d900.Loc == scm.LocReg && d901.Reg == d900.Reg {
			ctx.TransferReg(d900.Reg)
			d900.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d900)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d901)
		ctx.ReclaimUntrackedRegs()
		d903 = ctx.EmitSliceElementAddress(&d885, &d901, 8)
		ctx.EnsureDesc(&d903)
		ctx.EmitMovRegMem(d903.Reg, d903.Reg, 0)
		d902 = d903
		ctx.FreeDesc(&d901)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d884)
		var d904 scm.JITValueDesc
		if d884.Loc == scm.LocImm {
			d904 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d884.Imm.Int() % 64)}
		} else {
			r235 := ctx.AllocRegExcept(d884.Reg)
			ctx.EmitMovRegReg(r235, d884.Reg)
			ctx.EmitAndRegImm32(r235, 63)
			d904 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r235}
			ctx.BindReg(r235, &d904)
		}
		if d904.Loc == scm.LocReg && d884.Loc == scm.LocReg && d904.Reg == d884.Reg {
			ctx.TransferReg(d884.Reg)
			d884.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d884)
		ctx.ReclaimUntrackedRegs()
		d905 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d904)
		ctx.EnsureDesc(&d905)
		ctx.ProtectReg(d905.Reg)
		ctx.EnsureDesc(&d904)
		ctx.UnprotectReg(d905.Reg)
		var d906 scm.JITValueDesc
		if d905.Loc == scm.LocImm && d904.Loc == scm.LocImm {
			d906 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d905.Imm.Int() - d904.Imm.Int())}
		} else if d904.Loc == scm.LocImm && d904.Imm.Int() == 0 {
			r236 := ctx.AllocRegExcept(d905.Reg)
			ctx.EmitMovRegReg(r236, d905.Reg)
			d906 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r236}
			ctx.BindReg(r236, &d906)
		} else if d905.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d904.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d905.Imm.Int()))
			ctx.EmitSubInt64(scratch, d904.Reg)
			d906 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d906)
		} else if d904.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d905.Reg)
			ctx.EmitMovRegReg(scratch, d905.Reg)
			if d904.Imm.Int() >= -2147483648 && d904.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d904.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d904.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d906 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d906)
		} else {
			r237 := ctx.AllocRegExcept(d905.Reg, d904.Reg)
			ctx.EmitMovRegReg(r237, d905.Reg)
			ctx.EmitSubInt64(r237, d904.Reg)
			d906 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r237}
			ctx.BindReg(r237, &d906)
		}
		if d906.Loc == scm.LocReg && d905.Loc == scm.LocReg && d906.Reg == d905.Reg {
			ctx.TransferReg(d905.Reg)
			d905.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d904)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d902)
		ctx.EnsureDesc(&d906)
		var d907 scm.JITValueDesc
		if d902.Loc == scm.LocImm && d906.Loc == scm.LocImm {
			d907 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d902.Imm.Int()) >> uint64(d906.Imm.Int())))}
		} else if d906.Loc == scm.LocImm {
			r238 := ctx.AllocRegExcept(d902.Reg)
			ctx.EmitMovRegReg(r238, d902.Reg)
			ctx.EmitShrRegImm8(r238, uint8(d906.Imm.Int()))
			d907 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r238}
			ctx.BindReg(r238, &d907)
		} else {
			{
				shiftSrc := d902.Reg
				r239 := ctx.AllocRegExcept(d902.Reg)
				ctx.EmitMovRegReg(r239, d902.Reg)
				shiftSrc = r239
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d906.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d906.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d906.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d907 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d907)
			}
		}
		if d907.Loc == scm.LocReg && d902.Loc == scm.LocReg && d907.Reg == d902.Reg {
			ctx.TransferReg(d902.Reg)
			d902.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d902)
		ctx.FreeDesc(&d906)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d890)
		ctx.EnsureDesc(&d907)
		var d908 scm.JITValueDesc
		if d890.Loc == scm.LocImm && d907.Loc == scm.LocImm {
			d908 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d890.Imm.Int() | d907.Imm.Int())}
		} else if d890.Loc == scm.LocImm && d890.Imm.Int() == 0 {
			d908 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d907.Reg}
			ctx.BindReg(d907.Reg, &d908)
		} else if d907.Loc == scm.LocImm && d907.Imm.Int() == 0 {
			r240 := ctx.AllocRegExcept(d890.Reg)
			ctx.EmitMovRegReg(r240, d890.Reg)
			d908 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r240}
			ctx.BindReg(r240, &d908)
		} else if d890.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d907.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d890.Imm.Int()))
			ctx.EmitOrInt64(scratch, d907.Reg)
			d908 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d908)
		} else if d907.Loc == scm.LocImm {
			r241 := ctx.AllocRegExcept(d890.Reg)
			ctx.EmitMovRegReg(r241, d890.Reg)
			if d907.Imm.Int() >= -2147483648 && d907.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r241, int32(d907.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d907.Imm.Int()))
				ctx.EmitOrInt64(r241, scm.RegR11)
			}
			d908 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r241}
			ctx.BindReg(r241, &d908)
		} else {
			r242 := ctx.AllocRegExcept(d890.Reg, d907.Reg)
			ctx.EmitMovRegReg(r242, d890.Reg)
			ctx.EmitOrInt64(r242, d907.Reg)
			d908 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r242}
			ctx.BindReg(r242, &d908)
		}
		if d908.Loc == scm.LocReg && d890.Loc == scm.LocReg && d908.Reg == d890.Reg {
			ctx.TransferReg(d890.Reg)
			d890.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d908)
		ctx.EmitStoreToStack(d908, int32(phiBase879)+int32(0))
		ctx.StabilizeDescForControlFlow(&d908)
		ctx.FreeDesc(&d907)
		ctx.ReclaimUntrackedRegs()
		ctx.EmitJmp(lbl47)
		ctx.MarkLabel(lbl45)
		d909 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r233}
		ctx.BindReg(r233, &d909)
		ctx.BindReg(r233, &d909)
		ctx.FreeDesc(&d4)
		ctx.EnsureDesc(&d909)
		ctx.EnsureDesc(&d909)
		var d910 scm.JITValueDesc
		if d909.Loc == scm.LocImm {
			d910 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d909.Imm.Int()))))}
		} else {
			r243 := ctx.AllocReg()
			ctx.EmitMovRegReg(r243, d909.Reg)
			d910 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r243}
			ctx.BindReg(r243, &d910)
		}
		ctx.FreeDesc(&d909)
		ctx.EnsureDesc(&d910)
		ctx.EnsureDesc(&d55)
		ctx.EnsureDesc(&d910)
		ctx.ProtectReg(d910.Reg)
		ctx.EnsureDesc(&d55)
		ctx.UnprotectReg(d910.Reg)
		var d911 scm.JITValueDesc
		if d910.Loc == scm.LocImm && d55.Loc == scm.LocImm {
			d911 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d910.Imm.Int() + d55.Imm.Int())}
		} else if d55.Loc == scm.LocImm && d55.Imm.Int() == 0 {
			r244 := ctx.AllocRegExcept(d910.Reg)
			ctx.EmitMovRegReg(r244, d910.Reg)
			d911 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r244}
			ctx.BindReg(r244, &d911)
		} else if d910.Loc == scm.LocImm && d910.Imm.Int() == 0 {
			d911 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d55.Reg}
			ctx.BindReg(d55.Reg, &d911)
		} else if d910.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d55.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d910.Imm.Int()))
			ctx.EmitAddInt64(scratch, d55.Reg)
			d911 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d911)
		} else if d55.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d910.Reg)
			ctx.EmitMovRegReg(scratch, d910.Reg)
			if d55.Imm.Int() >= -2147483648 && d55.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d55.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d55.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d911 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d911)
		} else {
			r245 := ctx.AllocRegExcept(d910.Reg, d55.Reg)
			ctx.EmitMovRegReg(r245, d910.Reg)
			ctx.EmitAddInt64(r245, d55.Reg)
			d911 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r245}
			ctx.BindReg(r245, &d911)
		}
		if d911.Loc == scm.LocReg && d910.Loc == scm.LocReg && d911.Reg == d910.Reg {
			ctx.TransferReg(d910.Reg)
			d910.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d910)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		var d912 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm {
			d912 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(idxInt.Imm.Int()))))}
		} else {
			r246 := ctx.AllocReg()
			ctx.EmitMovRegReg(r246, idxInt.Reg)
			ctx.EmitShlRegImm8(r246, 32)
			ctx.EmitShrRegImm8(r246, 32)
			d912 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r246}
			ctx.BindReg(r246, &d912)
		}
		ctx.FreeDesc(&idxInt)
		ctx.EnsureDesc(&d912)
		ctx.EnsureDesc(&d911)
		ctx.EnsureDesc(&d912)
		ctx.ProtectReg(d912.Reg)
		ctx.EnsureDesc(&d911)
		ctx.UnprotectReg(d912.Reg)
		var d913 scm.JITValueDesc
		if d912.Loc == scm.LocImm && d911.Loc == scm.LocImm {
			d913 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d912.Imm.Int() - d911.Imm.Int())}
		} else if d911.Loc == scm.LocImm && d911.Imm.Int() == 0 {
			r247 := ctx.AllocRegExcept(d912.Reg)
			ctx.EmitMovRegReg(r247, d912.Reg)
			d913 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r247}
			ctx.BindReg(r247, &d913)
		} else if d912.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d911.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d912.Imm.Int()))
			ctx.EmitSubInt64(scratch, d911.Reg)
			d913 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d913)
		} else if d911.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d912.Reg)
			ctx.EmitMovRegReg(scratch, d912.Reg)
			if d911.Imm.Int() >= -2147483648 && d911.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d911.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d911.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d913 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d913)
		} else {
			r248 := ctx.AllocRegExcept(d912.Reg, d911.Reg)
			ctx.EmitMovRegReg(r248, d912.Reg)
			ctx.EmitSubInt64(r248, d911.Reg)
			d913 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r248}
			ctx.BindReg(r248, &d913)
		}
		if d913.Loc == scm.LocReg && d912.Loc == scm.LocReg && d913.Reg == d912.Reg {
			ctx.TransferReg(d912.Reg)
			d912.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d912)
		ctx.FreeDesc(&d911)
		ctx.EnsureDesc(&d913)
		ctx.EnsureDesc(&d877)
		ctx.EnsureDesc(&d913)
		ctx.ProtectReg(d913.Reg)
		ctx.EnsureDesc(&d877)
		ctx.UnprotectReg(d913.Reg)
		var d914 scm.JITValueDesc
		if d913.Loc == scm.LocImm && d877.Loc == scm.LocImm {
			d914 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d913.Imm.Int() * d877.Imm.Int())}
		} else if d913.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d877.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d913.Imm.Int()))
			ctx.EmitImulInt64(scratch, d877.Reg)
			d914 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d914)
		} else if d877.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d913.Reg)
			ctx.EmitMovRegReg(scratch, d913.Reg)
			if d877.Imm.Int() >= -2147483648 && d877.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d877.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d877.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d914 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d914)
		} else {
			r249 := ctx.AllocRegExcept(d913.Reg, d877.Reg)
			ctx.EmitMovRegReg(r249, d913.Reg)
			ctx.EmitImulInt64(r249, d877.Reg)
			d914 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r249}
			ctx.BindReg(r249, &d914)
		}
		if d914.Loc == scm.LocReg && d913.Loc == scm.LocReg && d914.Reg == d913.Reg {
			ctx.TransferReg(d913.Reg)
			d913.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d913)
		ctx.FreeDesc(&d877)
		ctx.EnsureDesc(&d164)
		ctx.EnsureDesc(&d914)
		ctx.EnsureDesc(&d164)
		ctx.ProtectReg(d164.Reg)
		ctx.EnsureDesc(&d914)
		ctx.UnprotectReg(d164.Reg)
		var d915 scm.JITValueDesc
		if d164.Loc == scm.LocImm && d914.Loc == scm.LocImm {
			d915 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d164.Imm.Int() + d914.Imm.Int())}
		} else if d914.Loc == scm.LocImm && d914.Imm.Int() == 0 {
			r250 := ctx.AllocRegExcept(d164.Reg)
			ctx.EmitMovRegReg(r250, d164.Reg)
			d915 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r250}
			ctx.BindReg(r250, &d915)
		} else if d164.Loc == scm.LocImm && d164.Imm.Int() == 0 {
			d915 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d914.Reg}
			ctx.BindReg(d914.Reg, &d915)
		} else if d164.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d914.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d164.Imm.Int()))
			ctx.EmitAddInt64(scratch, d914.Reg)
			d915 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d915)
		} else if d914.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d164.Reg)
			ctx.EmitMovRegReg(scratch, d164.Reg)
			if d914.Imm.Int() >= -2147483648 && d914.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d914.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d914.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d915 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d915)
		} else {
			r251 := ctx.AllocRegExcept(d164.Reg, d914.Reg)
			ctx.EmitMovRegReg(r251, d164.Reg)
			ctx.EmitAddInt64(r251, d914.Reg)
			d915 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r251}
			ctx.BindReg(r251, &d915)
		}
		if d915.Loc == scm.LocReg && d164.Loc == scm.LocReg && d915.Reg == d164.Reg {
			ctx.TransferReg(d164.Reg)
			d164.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d914)
		ctx.EnsureDesc(&d915)
		ctx.EnsureDesc(&d915)
		var d916 scm.JITValueDesc
		if d915.Loc == scm.LocImm {
			d916 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d915.Imm.Int()))}
		} else {
			ctx.EmitCvtInt64ToFloat64(scm.RegX0, d915.Reg)
			d916 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d915.Reg}
			ctx.BindReg(d915.Reg, &d916)
		}
		ctx.FreeDesc(&d915)
		ctx.EnsureDesc(&d916)
		d917 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d917)
		ctx.BindReg(r1, &d917)
		ctx.EnsureDesc(&d916)
		ctx.EmitMakeFloat(d917, d916)
		if d916.Loc == scm.LocReg {
			ctx.FreeReg(d916.Reg)
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
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
		}
		if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
			d58 = ps.OverlayValues[58]
		}
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
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
		if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
			d128 = ps.OverlayValues[128]
		}
		if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
			d129 = ps.OverlayValues[129]
		}
		if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
			d130 = ps.OverlayValues[130]
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
		if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != scm.LocNone {
			d148 = ps.OverlayValues[148]
		}
		if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != scm.LocNone {
			d149 = ps.OverlayValues[149]
		}
		if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
			d150 = ps.OverlayValues[150]
		}
		if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != scm.LocNone {
			d151 = ps.OverlayValues[151]
		}
		if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != scm.LocNone {
			d152 = ps.OverlayValues[152]
		}
		if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
			d153 = ps.OverlayValues[153]
		}
		if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
			d154 = ps.OverlayValues[154]
		}
		if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != scm.LocNone {
			d155 = ps.OverlayValues[155]
		}
		if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != scm.LocNone {
			d156 = ps.OverlayValues[156]
		}
		if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != scm.LocNone {
			d157 = ps.OverlayValues[157]
		}
		if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != scm.LocNone {
			d158 = ps.OverlayValues[158]
		}
		if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != scm.LocNone {
			d159 = ps.OverlayValues[159]
		}
		if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != scm.LocNone {
			d160 = ps.OverlayValues[160]
		}
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != scm.LocNone {
			d161 = ps.OverlayValues[161]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != scm.LocNone {
			d162 = ps.OverlayValues[162]
		}
		if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != scm.LocNone {
			d163 = ps.OverlayValues[163]
		}
		if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != scm.LocNone {
			d164 = ps.OverlayValues[164]
		}
		if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != scm.LocNone {
			d165 = ps.OverlayValues[165]
		}
		if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != scm.LocNone {
			d166 = ps.OverlayValues[166]
		}
		if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != scm.LocNone {
			d169 = ps.OverlayValues[169]
		}
		if len(ps.OverlayValues) > 272 && ps.OverlayValues[272].Loc != scm.LocNone {
			d272 = ps.OverlayValues[272]
		}
		if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != scm.LocNone {
			d273 = ps.OverlayValues[273]
		}
		if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != scm.LocNone {
			d274 = ps.OverlayValues[274]
		}
		if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != scm.LocNone {
			d275 = ps.OverlayValues[275]
		}
		if len(ps.OverlayValues) > 277 && ps.OverlayValues[277].Loc != scm.LocNone {
			d277 = ps.OverlayValues[277]
		}
		if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != scm.LocNone {
			d278 = ps.OverlayValues[278]
		}
		if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != scm.LocNone {
			d279 = ps.OverlayValues[279]
		}
		if len(ps.OverlayValues) > 280 && ps.OverlayValues[280].Loc != scm.LocNone {
			d280 = ps.OverlayValues[280]
		}
		if len(ps.OverlayValues) > 281 && ps.OverlayValues[281].Loc != scm.LocNone {
			d281 = ps.OverlayValues[281]
		}
		if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != scm.LocNone {
			d282 = ps.OverlayValues[282]
		}
		if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != scm.LocNone {
			d283 = ps.OverlayValues[283]
		}
		if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != scm.LocNone {
			d284 = ps.OverlayValues[284]
		}
		if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != scm.LocNone {
			d286 = ps.OverlayValues[286]
		}
		if len(ps.OverlayValues) > 288 && ps.OverlayValues[288].Loc != scm.LocNone {
			d288 = ps.OverlayValues[288]
		}
		if len(ps.OverlayValues) > 289 && ps.OverlayValues[289].Loc != scm.LocNone {
			d289 = ps.OverlayValues[289]
		}
		if len(ps.OverlayValues) > 290 && ps.OverlayValues[290].Loc != scm.LocNone {
			d290 = ps.OverlayValues[290]
		}
		if len(ps.OverlayValues) > 291 && ps.OverlayValues[291].Loc != scm.LocNone {
			d291 = ps.OverlayValues[291]
		}
		if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != scm.LocNone {
			d292 = ps.OverlayValues[292]
		}
		if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
			d295 = ps.OverlayValues[295]
		}
		if len(ps.OverlayValues) > 415 && ps.OverlayValues[415].Loc != scm.LocNone {
			d415 = ps.OverlayValues[415]
		}
		if len(ps.OverlayValues) > 416 && ps.OverlayValues[416].Loc != scm.LocNone {
			d416 = ps.OverlayValues[416]
		}
		if len(ps.OverlayValues) > 417 && ps.OverlayValues[417].Loc != scm.LocNone {
			d417 = ps.OverlayValues[417]
		}
		if len(ps.OverlayValues) > 418 && ps.OverlayValues[418].Loc != scm.LocNone {
			d418 = ps.OverlayValues[418]
		}
		if len(ps.OverlayValues) > 419 && ps.OverlayValues[419].Loc != scm.LocNone {
			d419 = ps.OverlayValues[419]
		}
		if len(ps.OverlayValues) > 421 && ps.OverlayValues[421].Loc != scm.LocNone {
			d421 = ps.OverlayValues[421]
		}
		if len(ps.OverlayValues) > 422 && ps.OverlayValues[422].Loc != scm.LocNone {
			d422 = ps.OverlayValues[422]
		}
		if len(ps.OverlayValues) > 423 && ps.OverlayValues[423].Loc != scm.LocNone {
			d423 = ps.OverlayValues[423]
		}
		if len(ps.OverlayValues) > 425 && ps.OverlayValues[425].Loc != scm.LocNone {
			d425 = ps.OverlayValues[425]
		}
		if len(ps.OverlayValues) > 426 && ps.OverlayValues[426].Loc != scm.LocNone {
			d426 = ps.OverlayValues[426]
		}
		if len(ps.OverlayValues) > 427 && ps.OverlayValues[427].Loc != scm.LocNone {
			d427 = ps.OverlayValues[427]
		}
		if len(ps.OverlayValues) > 428 && ps.OverlayValues[428].Loc != scm.LocNone {
			d428 = ps.OverlayValues[428]
		}
		if len(ps.OverlayValues) > 429 && ps.OverlayValues[429].Loc != scm.LocNone {
			d429 = ps.OverlayValues[429]
		}
		if len(ps.OverlayValues) > 430 && ps.OverlayValues[430].Loc != scm.LocNone {
			d430 = ps.OverlayValues[430]
		}
		if len(ps.OverlayValues) > 431 && ps.OverlayValues[431].Loc != scm.LocNone {
			d431 = ps.OverlayValues[431]
		}
		if len(ps.OverlayValues) > 432 && ps.OverlayValues[432].Loc != scm.LocNone {
			d432 = ps.OverlayValues[432]
		}
		if len(ps.OverlayValues) > 433 && ps.OverlayValues[433].Loc != scm.LocNone {
			d433 = ps.OverlayValues[433]
		}
		if len(ps.OverlayValues) > 434 && ps.OverlayValues[434].Loc != scm.LocNone {
			d434 = ps.OverlayValues[434]
		}
		if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != scm.LocNone {
			d435 = ps.OverlayValues[435]
		}
		if len(ps.OverlayValues) > 436 && ps.OverlayValues[436].Loc != scm.LocNone {
			d436 = ps.OverlayValues[436]
		}
		if len(ps.OverlayValues) > 437 && ps.OverlayValues[437].Loc != scm.LocNone {
			d437 = ps.OverlayValues[437]
		}
		if len(ps.OverlayValues) > 438 && ps.OverlayValues[438].Loc != scm.LocNone {
			d438 = ps.OverlayValues[438]
		}
		if len(ps.OverlayValues) > 439 && ps.OverlayValues[439].Loc != scm.LocNone {
			d439 = ps.OverlayValues[439]
		}
		if len(ps.OverlayValues) > 440 && ps.OverlayValues[440].Loc != scm.LocNone {
			d440 = ps.OverlayValues[440]
		}
		if len(ps.OverlayValues) > 441 && ps.OverlayValues[441].Loc != scm.LocNone {
			d441 = ps.OverlayValues[441]
		}
		if len(ps.OverlayValues) > 442 && ps.OverlayValues[442].Loc != scm.LocNone {
			d442 = ps.OverlayValues[442]
		}
		if len(ps.OverlayValues) > 443 && ps.OverlayValues[443].Loc != scm.LocNone {
			d443 = ps.OverlayValues[443]
		}
		if len(ps.OverlayValues) > 444 && ps.OverlayValues[444].Loc != scm.LocNone {
			d444 = ps.OverlayValues[444]
		}
		if len(ps.OverlayValues) > 445 && ps.OverlayValues[445].Loc != scm.LocNone {
			d445 = ps.OverlayValues[445]
		}
		if len(ps.OverlayValues) > 446 && ps.OverlayValues[446].Loc != scm.LocNone {
			d446 = ps.OverlayValues[446]
		}
		if len(ps.OverlayValues) > 447 && ps.OverlayValues[447].Loc != scm.LocNone {
			d447 = ps.OverlayValues[447]
		}
		if len(ps.OverlayValues) > 448 && ps.OverlayValues[448].Loc != scm.LocNone {
			d448 = ps.OverlayValues[448]
		}
		if len(ps.OverlayValues) > 449 && ps.OverlayValues[449].Loc != scm.LocNone {
			d449 = ps.OverlayValues[449]
		}
		if len(ps.OverlayValues) > 450 && ps.OverlayValues[450].Loc != scm.LocNone {
			d450 = ps.OverlayValues[450]
		}
		if len(ps.OverlayValues) > 451 && ps.OverlayValues[451].Loc != scm.LocNone {
			d451 = ps.OverlayValues[451]
		}
		if len(ps.OverlayValues) > 452 && ps.OverlayValues[452].Loc != scm.LocNone {
			d452 = ps.OverlayValues[452]
		}
		if len(ps.OverlayValues) > 453 && ps.OverlayValues[453].Loc != scm.LocNone {
			d453 = ps.OverlayValues[453]
		}
		if len(ps.OverlayValues) > 454 && ps.OverlayValues[454].Loc != scm.LocNone {
			d454 = ps.OverlayValues[454]
		}
		if len(ps.OverlayValues) > 455 && ps.OverlayValues[455].Loc != scm.LocNone {
			d455 = ps.OverlayValues[455]
		}
		if len(ps.OverlayValues) > 456 && ps.OverlayValues[456].Loc != scm.LocNone {
			d456 = ps.OverlayValues[456]
		}
		if len(ps.OverlayValues) > 457 && ps.OverlayValues[457].Loc != scm.LocNone {
			d457 = ps.OverlayValues[457]
		}
		if len(ps.OverlayValues) > 458 && ps.OverlayValues[458].Loc != scm.LocNone {
			d458 = ps.OverlayValues[458]
		}
		if len(ps.OverlayValues) > 459 && ps.OverlayValues[459].Loc != scm.LocNone {
			d459 = ps.OverlayValues[459]
		}
		if len(ps.OverlayValues) > 626 && ps.OverlayValues[626].Loc != scm.LocNone {
			d626 = ps.OverlayValues[626]
		}
		if len(ps.OverlayValues) > 627 && ps.OverlayValues[627].Loc != scm.LocNone {
			d627 = ps.OverlayValues[627]
		}
		if len(ps.OverlayValues) > 628 && ps.OverlayValues[628].Loc != scm.LocNone {
			d628 = ps.OverlayValues[628]
		}
		if len(ps.OverlayValues) > 630 && ps.OverlayValues[630].Loc != scm.LocNone {
			d630 = ps.OverlayValues[630]
		}
		if len(ps.OverlayValues) > 631 && ps.OverlayValues[631].Loc != scm.LocNone {
			d631 = ps.OverlayValues[631]
		}
		if len(ps.OverlayValues) > 632 && ps.OverlayValues[632].Loc != scm.LocNone {
			d632 = ps.OverlayValues[632]
		}
		if len(ps.OverlayValues) > 633 && ps.OverlayValues[633].Loc != scm.LocNone {
			d633 = ps.OverlayValues[633]
		}
		if len(ps.OverlayValues) > 634 && ps.OverlayValues[634].Loc != scm.LocNone {
			d634 = ps.OverlayValues[634]
		}
		if len(ps.OverlayValues) > 635 && ps.OverlayValues[635].Loc != scm.LocNone {
			d635 = ps.OverlayValues[635]
		}
		if len(ps.OverlayValues) > 636 && ps.OverlayValues[636].Loc != scm.LocNone {
			d636 = ps.OverlayValues[636]
		}
		if len(ps.OverlayValues) > 638 && ps.OverlayValues[638].Loc != scm.LocNone {
			d638 = ps.OverlayValues[638]
		}
		if len(ps.OverlayValues) > 640 && ps.OverlayValues[640].Loc != scm.LocNone {
			d640 = ps.OverlayValues[640]
		}
		if len(ps.OverlayValues) > 641 && ps.OverlayValues[641].Loc != scm.LocNone {
			d641 = ps.OverlayValues[641]
		}
		if len(ps.OverlayValues) > 642 && ps.OverlayValues[642].Loc != scm.LocNone {
			d642 = ps.OverlayValues[642]
		}
		if len(ps.OverlayValues) > 643 && ps.OverlayValues[643].Loc != scm.LocNone {
			d643 = ps.OverlayValues[643]
		}
		if len(ps.OverlayValues) > 646 && ps.OverlayValues[646].Loc != scm.LocNone {
			d646 = ps.OverlayValues[646]
		}
		if len(ps.OverlayValues) > 825 && ps.OverlayValues[825].Loc != scm.LocNone {
			d825 = ps.OverlayValues[825]
		}
		if len(ps.OverlayValues) > 826 && ps.OverlayValues[826].Loc != scm.LocNone {
			d826 = ps.OverlayValues[826]
		}
		if len(ps.OverlayValues) > 827 && ps.OverlayValues[827].Loc != scm.LocNone {
			d827 = ps.OverlayValues[827]
		}
		if len(ps.OverlayValues) > 828 && ps.OverlayValues[828].Loc != scm.LocNone {
			d828 = ps.OverlayValues[828]
		}
		if len(ps.OverlayValues) > 830 && ps.OverlayValues[830].Loc != scm.LocNone {
			d830 = ps.OverlayValues[830]
		}
		if len(ps.OverlayValues) > 831 && ps.OverlayValues[831].Loc != scm.LocNone {
			d831 = ps.OverlayValues[831]
		}
		if len(ps.OverlayValues) > 832 && ps.OverlayValues[832].Loc != scm.LocNone {
			d832 = ps.OverlayValues[832]
		}
		if len(ps.OverlayValues) > 833 && ps.OverlayValues[833].Loc != scm.LocNone {
			d833 = ps.OverlayValues[833]
		}
		if len(ps.OverlayValues) > 834 && ps.OverlayValues[834].Loc != scm.LocNone {
			d834 = ps.OverlayValues[834]
		}
		if len(ps.OverlayValues) > 835 && ps.OverlayValues[835].Loc != scm.LocNone {
			d835 = ps.OverlayValues[835]
		}
		if len(ps.OverlayValues) > 836 && ps.OverlayValues[836].Loc != scm.LocNone {
			d836 = ps.OverlayValues[836]
		}
		if len(ps.OverlayValues) > 837 && ps.OverlayValues[837].Loc != scm.LocNone {
			d837 = ps.OverlayValues[837]
		}
		if len(ps.OverlayValues) > 839 && ps.OverlayValues[839].Loc != scm.LocNone {
			d839 = ps.OverlayValues[839]
		}
		if len(ps.OverlayValues) > 840 && ps.OverlayValues[840].Loc != scm.LocNone {
			d840 = ps.OverlayValues[840]
		}
		if len(ps.OverlayValues) > 841 && ps.OverlayValues[841].Loc != scm.LocNone {
			d841 = ps.OverlayValues[841]
		}
		if len(ps.OverlayValues) > 842 && ps.OverlayValues[842].Loc != scm.LocNone {
			d842 = ps.OverlayValues[842]
		}
		if len(ps.OverlayValues) > 843 && ps.OverlayValues[843].Loc != scm.LocNone {
			d843 = ps.OverlayValues[843]
		}
		if len(ps.OverlayValues) > 845 && ps.OverlayValues[845].Loc != scm.LocNone {
			d845 = ps.OverlayValues[845]
		}
		if len(ps.OverlayValues) > 846 && ps.OverlayValues[846].Loc != scm.LocNone {
			d846 = ps.OverlayValues[846]
		}
		if len(ps.OverlayValues) > 847 && ps.OverlayValues[847].Loc != scm.LocNone {
			d847 = ps.OverlayValues[847]
		}
		if len(ps.OverlayValues) > 848 && ps.OverlayValues[848].Loc != scm.LocNone {
			d848 = ps.OverlayValues[848]
		}
		if len(ps.OverlayValues) > 849 && ps.OverlayValues[849].Loc != scm.LocNone {
			d849 = ps.OverlayValues[849]
		}
		if len(ps.OverlayValues) > 850 && ps.OverlayValues[850].Loc != scm.LocNone {
			d850 = ps.OverlayValues[850]
		}
		if len(ps.OverlayValues) > 851 && ps.OverlayValues[851].Loc != scm.LocNone {
			d851 = ps.OverlayValues[851]
		}
		if len(ps.OverlayValues) > 852 && ps.OverlayValues[852].Loc != scm.LocNone {
			d852 = ps.OverlayValues[852]
		}
		if len(ps.OverlayValues) > 853 && ps.OverlayValues[853].Loc != scm.LocNone {
			d853 = ps.OverlayValues[853]
		}
		if len(ps.OverlayValues) > 854 && ps.OverlayValues[854].Loc != scm.LocNone {
			d854 = ps.OverlayValues[854]
		}
		if len(ps.OverlayValues) > 855 && ps.OverlayValues[855].Loc != scm.LocNone {
			d855 = ps.OverlayValues[855]
		}
		if len(ps.OverlayValues) > 856 && ps.OverlayValues[856].Loc != scm.LocNone {
			d856 = ps.OverlayValues[856]
		}
		if len(ps.OverlayValues) > 857 && ps.OverlayValues[857].Loc != scm.LocNone {
			d857 = ps.OverlayValues[857]
		}
		if len(ps.OverlayValues) > 858 && ps.OverlayValues[858].Loc != scm.LocNone {
			d858 = ps.OverlayValues[858]
		}
		if len(ps.OverlayValues) > 859 && ps.OverlayValues[859].Loc != scm.LocNone {
			d859 = ps.OverlayValues[859]
		}
		if len(ps.OverlayValues) > 860 && ps.OverlayValues[860].Loc != scm.LocNone {
			d860 = ps.OverlayValues[860]
		}
		if len(ps.OverlayValues) > 861 && ps.OverlayValues[861].Loc != scm.LocNone {
			d861 = ps.OverlayValues[861]
		}
		if len(ps.OverlayValues) > 862 && ps.OverlayValues[862].Loc != scm.LocNone {
			d862 = ps.OverlayValues[862]
		}
		if len(ps.OverlayValues) > 863 && ps.OverlayValues[863].Loc != scm.LocNone {
			d863 = ps.OverlayValues[863]
		}
		if len(ps.OverlayValues) > 864 && ps.OverlayValues[864].Loc != scm.LocNone {
			d864 = ps.OverlayValues[864]
		}
		if len(ps.OverlayValues) > 865 && ps.OverlayValues[865].Loc != scm.LocNone {
			d865 = ps.OverlayValues[865]
		}
		if len(ps.OverlayValues) > 866 && ps.OverlayValues[866].Loc != scm.LocNone {
			d866 = ps.OverlayValues[866]
		}
		if len(ps.OverlayValues) > 867 && ps.OverlayValues[867].Loc != scm.LocNone {
			d867 = ps.OverlayValues[867]
		}
		if len(ps.OverlayValues) > 868 && ps.OverlayValues[868].Loc != scm.LocNone {
			d868 = ps.OverlayValues[868]
		}
		if len(ps.OverlayValues) > 869 && ps.OverlayValues[869].Loc != scm.LocNone {
			d869 = ps.OverlayValues[869]
		}
		if len(ps.OverlayValues) > 870 && ps.OverlayValues[870].Loc != scm.LocNone {
			d870 = ps.OverlayValues[870]
		}
		if len(ps.OverlayValues) > 871 && ps.OverlayValues[871].Loc != scm.LocNone {
			d871 = ps.OverlayValues[871]
		}
		if len(ps.OverlayValues) > 872 && ps.OverlayValues[872].Loc != scm.LocNone {
			d872 = ps.OverlayValues[872]
		}
		if len(ps.OverlayValues) > 873 && ps.OverlayValues[873].Loc != scm.LocNone {
			d873 = ps.OverlayValues[873]
		}
		if len(ps.OverlayValues) > 874 && ps.OverlayValues[874].Loc != scm.LocNone {
			d874 = ps.OverlayValues[874]
		}
		if len(ps.OverlayValues) > 875 && ps.OverlayValues[875].Loc != scm.LocNone {
			d875 = ps.OverlayValues[875]
		}
		if len(ps.OverlayValues) > 876 && ps.OverlayValues[876].Loc != scm.LocNone {
			d876 = ps.OverlayValues[876]
		}
		if len(ps.OverlayValues) > 877 && ps.OverlayValues[877].Loc != scm.LocNone {
			d877 = ps.OverlayValues[877]
		}
		if len(ps.OverlayValues) > 878 && ps.OverlayValues[878].Loc != scm.LocNone {
			d878 = ps.OverlayValues[878]
		}
		if len(ps.OverlayValues) > 880 && ps.OverlayValues[880].Loc != scm.LocNone {
			d880 = ps.OverlayValues[880]
		}
		if len(ps.OverlayValues) > 881 && ps.OverlayValues[881].Loc != scm.LocNone {
			d881 = ps.OverlayValues[881]
		}
		if len(ps.OverlayValues) > 882 && ps.OverlayValues[882].Loc != scm.LocNone {
			d882 = ps.OverlayValues[882]
		}
		if len(ps.OverlayValues) > 883 && ps.OverlayValues[883].Loc != scm.LocNone {
			d883 = ps.OverlayValues[883]
		}
		if len(ps.OverlayValues) > 884 && ps.OverlayValues[884].Loc != scm.LocNone {
			d884 = ps.OverlayValues[884]
		}
		if len(ps.OverlayValues) > 885 && ps.OverlayValues[885].Loc != scm.LocNone {
			d885 = ps.OverlayValues[885]
		}
		if len(ps.OverlayValues) > 886 && ps.OverlayValues[886].Loc != scm.LocNone {
			d886 = ps.OverlayValues[886]
		}
		if len(ps.OverlayValues) > 887 && ps.OverlayValues[887].Loc != scm.LocNone {
			d887 = ps.OverlayValues[887]
		}
		if len(ps.OverlayValues) > 888 && ps.OverlayValues[888].Loc != scm.LocNone {
			d888 = ps.OverlayValues[888]
		}
		if len(ps.OverlayValues) > 889 && ps.OverlayValues[889].Loc != scm.LocNone {
			d889 = ps.OverlayValues[889]
		}
		if len(ps.OverlayValues) > 890 && ps.OverlayValues[890].Loc != scm.LocNone {
			d890 = ps.OverlayValues[890]
		}
		if len(ps.OverlayValues) > 891 && ps.OverlayValues[891].Loc != scm.LocNone {
			d891 = ps.OverlayValues[891]
		}
		if len(ps.OverlayValues) > 892 && ps.OverlayValues[892].Loc != scm.LocNone {
			d892 = ps.OverlayValues[892]
		}
		if len(ps.OverlayValues) > 893 && ps.OverlayValues[893].Loc != scm.LocNone {
			d893 = ps.OverlayValues[893]
		}
		if len(ps.OverlayValues) > 894 && ps.OverlayValues[894].Loc != scm.LocNone {
			d894 = ps.OverlayValues[894]
		}
		if len(ps.OverlayValues) > 895 && ps.OverlayValues[895].Loc != scm.LocNone {
			d895 = ps.OverlayValues[895]
		}
		if len(ps.OverlayValues) > 896 && ps.OverlayValues[896].Loc != scm.LocNone {
			d896 = ps.OverlayValues[896]
		}
		if len(ps.OverlayValues) > 897 && ps.OverlayValues[897].Loc != scm.LocNone {
			d897 = ps.OverlayValues[897]
		}
		if len(ps.OverlayValues) > 898 && ps.OverlayValues[898].Loc != scm.LocNone {
			d898 = ps.OverlayValues[898]
		}
		if len(ps.OverlayValues) > 899 && ps.OverlayValues[899].Loc != scm.LocNone {
			d899 = ps.OverlayValues[899]
		}
		if len(ps.OverlayValues) > 900 && ps.OverlayValues[900].Loc != scm.LocNone {
			d900 = ps.OverlayValues[900]
		}
		if len(ps.OverlayValues) > 901 && ps.OverlayValues[901].Loc != scm.LocNone {
			d901 = ps.OverlayValues[901]
		}
		if len(ps.OverlayValues) > 902 && ps.OverlayValues[902].Loc != scm.LocNone {
			d902 = ps.OverlayValues[902]
		}
		if len(ps.OverlayValues) > 903 && ps.OverlayValues[903].Loc != scm.LocNone {
			d903 = ps.OverlayValues[903]
		}
		if len(ps.OverlayValues) > 904 && ps.OverlayValues[904].Loc != scm.LocNone {
			d904 = ps.OverlayValues[904]
		}
		if len(ps.OverlayValues) > 905 && ps.OverlayValues[905].Loc != scm.LocNone {
			d905 = ps.OverlayValues[905]
		}
		if len(ps.OverlayValues) > 906 && ps.OverlayValues[906].Loc != scm.LocNone {
			d906 = ps.OverlayValues[906]
		}
		if len(ps.OverlayValues) > 907 && ps.OverlayValues[907].Loc != scm.LocNone {
			d907 = ps.OverlayValues[907]
		}
		if len(ps.OverlayValues) > 908 && ps.OverlayValues[908].Loc != scm.LocNone {
			d908 = ps.OverlayValues[908]
		}
		if len(ps.OverlayValues) > 909 && ps.OverlayValues[909].Loc != scm.LocNone {
			d909 = ps.OverlayValues[909]
		}
		if len(ps.OverlayValues) > 910 && ps.OverlayValues[910].Loc != scm.LocNone {
			d910 = ps.OverlayValues[910]
		}
		if len(ps.OverlayValues) > 911 && ps.OverlayValues[911].Loc != scm.LocNone {
			d911 = ps.OverlayValues[911]
		}
		if len(ps.OverlayValues) > 912 && ps.OverlayValues[912].Loc != scm.LocNone {
			d912 = ps.OverlayValues[912]
		}
		if len(ps.OverlayValues) > 913 && ps.OverlayValues[913].Loc != scm.LocNone {
			d913 = ps.OverlayValues[913]
		}
		if len(ps.OverlayValues) > 914 && ps.OverlayValues[914].Loc != scm.LocNone {
			d914 = ps.OverlayValues[914]
		}
		if len(ps.OverlayValues) > 915 && ps.OverlayValues[915].Loc != scm.LocNone {
			d915 = ps.OverlayValues[915]
		}
		if len(ps.OverlayValues) > 916 && ps.OverlayValues[916].Loc != scm.LocNone {
			d916 = ps.OverlayValues[916]
		}
		if len(ps.OverlayValues) > 917 && ps.OverlayValues[917].Loc != scm.LocNone {
			d917 = ps.OverlayValues[917]
		}
		ctx.ReclaimUntrackedRegs()
		var d918 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 64
			r252 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r252, fieldAddr)
			d918 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r252}
			ctx.BindReg(r252, &d918)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 64)
			r253 := ctx.AllocReg()
			ctx.EmitMovRegMem(r253, thisptr.Reg, off)
			d918 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r253}
			ctx.BindReg(r253, &d918)
		}
		ctx.EnsureDesc(&d918)
		ctx.EnsureDesc(&d918)
		var d919 scm.JITValueDesc
		if d918.Loc == scm.LocImm {
			d919 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d918.Imm.Int()))))}
		} else {
			r254 := ctx.AllocReg()
			ctx.EmitMovRegReg(r254, d918.Reg)
			d919 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r254}
			ctx.BindReg(r254, &d919)
		}
		ctx.EnsureDesc(&d164)
		ctx.EnsureDesc(&d919)
		ctx.EnsureDesc(&d164)
		ctx.EnsureDesc(&d919)
		ctx.EnsureDesc(&d164)
		ctx.EnsureDesc(&d919)
		var d920 scm.JITValueDesc
		if d164.Loc == scm.LocImm && d919.Loc == scm.LocImm {
			d920 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d164.Imm.Int() == d919.Imm.Int())}
		} else if d919.Loc == scm.LocImm {
			r255 := ctx.AllocRegExcept(d164.Reg)
			if d919.Imm.Int() >= -2147483648 && d919.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d164.Reg, int32(d919.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d919.Imm.Int()))
				ctx.EmitCmpInt64(d164.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r255, scm.CondEqual)
			d920 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r255}
			ctx.BindReg(r255, &d920)
		} else if d164.Loc == scm.LocImm {
			r256 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d164.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d919.Reg)
			ctx.EmitSetcc(r256, scm.CondEqual)
			d920 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r256}
			ctx.BindReg(r256, &d920)
		} else {
			r257 := ctx.AllocRegExcept(d164.Reg)
			ctx.EmitCmpInt64(d164.Reg, d919.Reg)
			ctx.EmitSetcc(r257, scm.CondEqual)
			d920 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r257}
			ctx.BindReg(r257, &d920)
		}
		ctx.FreeDesc(&d164)
		ctx.FreeDesc(&d919)
		d921 = d920
		ctx.EnsureDesc(&d921)
		if d921.Loc != scm.LocImm && d921.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d921.Loc == scm.LocImm {
			if d921.Imm.Bool() {
				if ps.General {
				}
				ps922 := scm.PhiState{General: ps.General}
				ps922.OverlayValues = make([]scm.JITValueDesc, 922)
				ps922.OverlayValues[1] = d1
				ps922.OverlayValues[2] = d2
				ps922.OverlayValues[3] = d3
				ps922.OverlayValues[4] = d4
				ps922.OverlayValues[5] = d5
				ps922.OverlayValues[6] = d6
				ps922.OverlayValues[7] = d7
				ps922.OverlayValues[8] = d8
				ps922.OverlayValues[9] = d9
				ps922.OverlayValues[10] = d10
				ps922.OverlayValues[11] = d11
				ps922.OverlayValues[12] = d12
				ps922.OverlayValues[13] = d13
				ps922.OverlayValues[14] = d14
				ps922.OverlayValues[15] = d15
				ps922.OverlayValues[17] = d17
				ps922.OverlayValues[18] = d18
				ps922.OverlayValues[19] = d19
				ps922.OverlayValues[20] = d20
				ps922.OverlayValues[21] = d21
				ps922.OverlayValues[22] = d22
				ps922.OverlayValues[24] = d24
				ps922.OverlayValues[25] = d25
				ps922.OverlayValues[26] = d26
				ps922.OverlayValues[27] = d27
				ps922.OverlayValues[28] = d28
				ps922.OverlayValues[29] = d29
				ps922.OverlayValues[30] = d30
				ps922.OverlayValues[31] = d31
				ps922.OverlayValues[32] = d32
				ps922.OverlayValues[33] = d33
				ps922.OverlayValues[34] = d34
				ps922.OverlayValues[35] = d35
				ps922.OverlayValues[36] = d36
				ps922.OverlayValues[37] = d37
				ps922.OverlayValues[38] = d38
				ps922.OverlayValues[39] = d39
				ps922.OverlayValues[40] = d40
				ps922.OverlayValues[41] = d41
				ps922.OverlayValues[42] = d42
				ps922.OverlayValues[43] = d43
				ps922.OverlayValues[44] = d44
				ps922.OverlayValues[45] = d45
				ps922.OverlayValues[46] = d46
				ps922.OverlayValues[47] = d47
				ps922.OverlayValues[48] = d48
				ps922.OverlayValues[49] = d49
				ps922.OverlayValues[50] = d50
				ps922.OverlayValues[51] = d51
				ps922.OverlayValues[52] = d52
				ps922.OverlayValues[53] = d53
				ps922.OverlayValues[54] = d54
				ps922.OverlayValues[55] = d55
				ps922.OverlayValues[56] = d56
				ps922.OverlayValues[57] = d57
				ps922.OverlayValues[58] = d58
				ps922.OverlayValues[59] = d59
				ps922.OverlayValues[62] = d62
				ps922.OverlayValues[63] = d63
				ps922.OverlayValues[64] = d64
				ps922.OverlayValues[128] = d128
				ps922.OverlayValues[129] = d129
				ps922.OverlayValues[130] = d130
				ps922.OverlayValues[132] = d132
				ps922.OverlayValues[133] = d133
				ps922.OverlayValues[134] = d134
				ps922.OverlayValues[135] = d135
				ps922.OverlayValues[136] = d136
				ps922.OverlayValues[137] = d137
				ps922.OverlayValues[138] = d138
				ps922.OverlayValues[139] = d139
				ps922.OverlayValues[140] = d140
				ps922.OverlayValues[141] = d141
				ps922.OverlayValues[142] = d142
				ps922.OverlayValues[143] = d143
				ps922.OverlayValues[144] = d144
				ps922.OverlayValues[145] = d145
				ps922.OverlayValues[146] = d146
				ps922.OverlayValues[147] = d147
				ps922.OverlayValues[148] = d148
				ps922.OverlayValues[149] = d149
				ps922.OverlayValues[150] = d150
				ps922.OverlayValues[151] = d151
				ps922.OverlayValues[152] = d152
				ps922.OverlayValues[153] = d153
				ps922.OverlayValues[154] = d154
				ps922.OverlayValues[155] = d155
				ps922.OverlayValues[156] = d156
				ps922.OverlayValues[157] = d157
				ps922.OverlayValues[158] = d158
				ps922.OverlayValues[159] = d159
				ps922.OverlayValues[160] = d160
				ps922.OverlayValues[161] = d161
				ps922.OverlayValues[162] = d162
				ps922.OverlayValues[163] = d163
				ps922.OverlayValues[164] = d164
				ps922.OverlayValues[165] = d165
				ps922.OverlayValues[166] = d166
				ps922.OverlayValues[169] = d169
				ps922.OverlayValues[272] = d272
				ps922.OverlayValues[273] = d273
				ps922.OverlayValues[274] = d274
				ps922.OverlayValues[275] = d275
				ps922.OverlayValues[277] = d277
				ps922.OverlayValues[278] = d278
				ps922.OverlayValues[279] = d279
				ps922.OverlayValues[280] = d280
				ps922.OverlayValues[281] = d281
				ps922.OverlayValues[282] = d282
				ps922.OverlayValues[283] = d283
				ps922.OverlayValues[284] = d284
				ps922.OverlayValues[286] = d286
				ps922.OverlayValues[288] = d288
				ps922.OverlayValues[289] = d289
				ps922.OverlayValues[290] = d290
				ps922.OverlayValues[291] = d291
				ps922.OverlayValues[292] = d292
				ps922.OverlayValues[295] = d295
				ps922.OverlayValues[415] = d415
				ps922.OverlayValues[416] = d416
				ps922.OverlayValues[417] = d417
				ps922.OverlayValues[418] = d418
				ps922.OverlayValues[419] = d419
				ps922.OverlayValues[421] = d421
				ps922.OverlayValues[422] = d422
				ps922.OverlayValues[423] = d423
				ps922.OverlayValues[425] = d425
				ps922.OverlayValues[426] = d426
				ps922.OverlayValues[427] = d427
				ps922.OverlayValues[428] = d428
				ps922.OverlayValues[429] = d429
				ps922.OverlayValues[430] = d430
				ps922.OverlayValues[431] = d431
				ps922.OverlayValues[432] = d432
				ps922.OverlayValues[433] = d433
				ps922.OverlayValues[434] = d434
				ps922.OverlayValues[435] = d435
				ps922.OverlayValues[436] = d436
				ps922.OverlayValues[437] = d437
				ps922.OverlayValues[438] = d438
				ps922.OverlayValues[439] = d439
				ps922.OverlayValues[440] = d440
				ps922.OverlayValues[441] = d441
				ps922.OverlayValues[442] = d442
				ps922.OverlayValues[443] = d443
				ps922.OverlayValues[444] = d444
				ps922.OverlayValues[445] = d445
				ps922.OverlayValues[446] = d446
				ps922.OverlayValues[447] = d447
				ps922.OverlayValues[448] = d448
				ps922.OverlayValues[449] = d449
				ps922.OverlayValues[450] = d450
				ps922.OverlayValues[451] = d451
				ps922.OverlayValues[452] = d452
				ps922.OverlayValues[453] = d453
				ps922.OverlayValues[454] = d454
				ps922.OverlayValues[455] = d455
				ps922.OverlayValues[456] = d456
				ps922.OverlayValues[457] = d457
				ps922.OverlayValues[458] = d458
				ps922.OverlayValues[459] = d459
				ps922.OverlayValues[626] = d626
				ps922.OverlayValues[627] = d627
				ps922.OverlayValues[628] = d628
				ps922.OverlayValues[630] = d630
				ps922.OverlayValues[631] = d631
				ps922.OverlayValues[632] = d632
				ps922.OverlayValues[633] = d633
				ps922.OverlayValues[634] = d634
				ps922.OverlayValues[635] = d635
				ps922.OverlayValues[636] = d636
				ps922.OverlayValues[638] = d638
				ps922.OverlayValues[640] = d640
				ps922.OverlayValues[641] = d641
				ps922.OverlayValues[642] = d642
				ps922.OverlayValues[643] = d643
				ps922.OverlayValues[646] = d646
				ps922.OverlayValues[825] = d825
				ps922.OverlayValues[826] = d826
				ps922.OverlayValues[827] = d827
				ps922.OverlayValues[828] = d828
				ps922.OverlayValues[830] = d830
				ps922.OverlayValues[831] = d831
				ps922.OverlayValues[832] = d832
				ps922.OverlayValues[833] = d833
				ps922.OverlayValues[834] = d834
				ps922.OverlayValues[835] = d835
				ps922.OverlayValues[836] = d836
				ps922.OverlayValues[837] = d837
				ps922.OverlayValues[839] = d839
				ps922.OverlayValues[840] = d840
				ps922.OverlayValues[841] = d841
				ps922.OverlayValues[842] = d842
				ps922.OverlayValues[843] = d843
				ps922.OverlayValues[845] = d845
				ps922.OverlayValues[846] = d846
				ps922.OverlayValues[847] = d847
				ps922.OverlayValues[848] = d848
				ps922.OverlayValues[849] = d849
				ps922.OverlayValues[850] = d850
				ps922.OverlayValues[851] = d851
				ps922.OverlayValues[852] = d852
				ps922.OverlayValues[853] = d853
				ps922.OverlayValues[854] = d854
				ps922.OverlayValues[855] = d855
				ps922.OverlayValues[856] = d856
				ps922.OverlayValues[857] = d857
				ps922.OverlayValues[858] = d858
				ps922.OverlayValues[859] = d859
				ps922.OverlayValues[860] = d860
				ps922.OverlayValues[861] = d861
				ps922.OverlayValues[862] = d862
				ps922.OverlayValues[863] = d863
				ps922.OverlayValues[864] = d864
				ps922.OverlayValues[865] = d865
				ps922.OverlayValues[866] = d866
				ps922.OverlayValues[867] = d867
				ps922.OverlayValues[868] = d868
				ps922.OverlayValues[869] = d869
				ps922.OverlayValues[870] = d870
				ps922.OverlayValues[871] = d871
				ps922.OverlayValues[872] = d872
				ps922.OverlayValues[873] = d873
				ps922.OverlayValues[874] = d874
				ps922.OverlayValues[875] = d875
				ps922.OverlayValues[876] = d876
				ps922.OverlayValues[877] = d877
				ps922.OverlayValues[878] = d878
				ps922.OverlayValues[880] = d880
				ps922.OverlayValues[881] = d881
				ps922.OverlayValues[882] = d882
				ps922.OverlayValues[883] = d883
				ps922.OverlayValues[884] = d884
				ps922.OverlayValues[885] = d885
				ps922.OverlayValues[886] = d886
				ps922.OverlayValues[887] = d887
				ps922.OverlayValues[888] = d888
				ps922.OverlayValues[889] = d889
				ps922.OverlayValues[890] = d890
				ps922.OverlayValues[891] = d891
				ps922.OverlayValues[892] = d892
				ps922.OverlayValues[893] = d893
				ps922.OverlayValues[894] = d894
				ps922.OverlayValues[895] = d895
				ps922.OverlayValues[896] = d896
				ps922.OverlayValues[897] = d897
				ps922.OverlayValues[898] = d898
				ps922.OverlayValues[899] = d899
				ps922.OverlayValues[900] = d900
				ps922.OverlayValues[901] = d901
				ps922.OverlayValues[902] = d902
				ps922.OverlayValues[903] = d903
				ps922.OverlayValues[904] = d904
				ps922.OverlayValues[905] = d905
				ps922.OverlayValues[906] = d906
				ps922.OverlayValues[907] = d907
				ps922.OverlayValues[908] = d908
				ps922.OverlayValues[909] = d909
				ps922.OverlayValues[910] = d910
				ps922.OverlayValues[911] = d911
				ps922.OverlayValues[912] = d912
				ps922.OverlayValues[913] = d913
				ps922.OverlayValues[914] = d914
				ps922.OverlayValues[915] = d915
				ps922.OverlayValues[916] = d916
				ps922.OverlayValues[917] = d917
				ps922.OverlayValues[918] = d918
				ps922.OverlayValues[919] = d919
				ps922.OverlayValues[920] = d920
				ps922.OverlayValues[921] = d921
				return bbs[11].RenderPS(ps922)
			}
			if ps.General {
			}
			ps923 := scm.PhiState{General: ps.General}
			ps923.OverlayValues = make([]scm.JITValueDesc, 922)
			ps923.OverlayValues[1] = d1
			ps923.OverlayValues[2] = d2
			ps923.OverlayValues[3] = d3
			ps923.OverlayValues[4] = d4
			ps923.OverlayValues[5] = d5
			ps923.OverlayValues[6] = d6
			ps923.OverlayValues[7] = d7
			ps923.OverlayValues[8] = d8
			ps923.OverlayValues[9] = d9
			ps923.OverlayValues[10] = d10
			ps923.OverlayValues[11] = d11
			ps923.OverlayValues[12] = d12
			ps923.OverlayValues[13] = d13
			ps923.OverlayValues[14] = d14
			ps923.OverlayValues[15] = d15
			ps923.OverlayValues[17] = d17
			ps923.OverlayValues[18] = d18
			ps923.OverlayValues[19] = d19
			ps923.OverlayValues[20] = d20
			ps923.OverlayValues[21] = d21
			ps923.OverlayValues[22] = d22
			ps923.OverlayValues[24] = d24
			ps923.OverlayValues[25] = d25
			ps923.OverlayValues[26] = d26
			ps923.OverlayValues[27] = d27
			ps923.OverlayValues[28] = d28
			ps923.OverlayValues[29] = d29
			ps923.OverlayValues[30] = d30
			ps923.OverlayValues[31] = d31
			ps923.OverlayValues[32] = d32
			ps923.OverlayValues[33] = d33
			ps923.OverlayValues[34] = d34
			ps923.OverlayValues[35] = d35
			ps923.OverlayValues[36] = d36
			ps923.OverlayValues[37] = d37
			ps923.OverlayValues[38] = d38
			ps923.OverlayValues[39] = d39
			ps923.OverlayValues[40] = d40
			ps923.OverlayValues[41] = d41
			ps923.OverlayValues[42] = d42
			ps923.OverlayValues[43] = d43
			ps923.OverlayValues[44] = d44
			ps923.OverlayValues[45] = d45
			ps923.OverlayValues[46] = d46
			ps923.OverlayValues[47] = d47
			ps923.OverlayValues[48] = d48
			ps923.OverlayValues[49] = d49
			ps923.OverlayValues[50] = d50
			ps923.OverlayValues[51] = d51
			ps923.OverlayValues[52] = d52
			ps923.OverlayValues[53] = d53
			ps923.OverlayValues[54] = d54
			ps923.OverlayValues[55] = d55
			ps923.OverlayValues[56] = d56
			ps923.OverlayValues[57] = d57
			ps923.OverlayValues[58] = d58
			ps923.OverlayValues[59] = d59
			ps923.OverlayValues[62] = d62
			ps923.OverlayValues[63] = d63
			ps923.OverlayValues[64] = d64
			ps923.OverlayValues[128] = d128
			ps923.OverlayValues[129] = d129
			ps923.OverlayValues[130] = d130
			ps923.OverlayValues[132] = d132
			ps923.OverlayValues[133] = d133
			ps923.OverlayValues[134] = d134
			ps923.OverlayValues[135] = d135
			ps923.OverlayValues[136] = d136
			ps923.OverlayValues[137] = d137
			ps923.OverlayValues[138] = d138
			ps923.OverlayValues[139] = d139
			ps923.OverlayValues[140] = d140
			ps923.OverlayValues[141] = d141
			ps923.OverlayValues[142] = d142
			ps923.OverlayValues[143] = d143
			ps923.OverlayValues[144] = d144
			ps923.OverlayValues[145] = d145
			ps923.OverlayValues[146] = d146
			ps923.OverlayValues[147] = d147
			ps923.OverlayValues[148] = d148
			ps923.OverlayValues[149] = d149
			ps923.OverlayValues[150] = d150
			ps923.OverlayValues[151] = d151
			ps923.OverlayValues[152] = d152
			ps923.OverlayValues[153] = d153
			ps923.OverlayValues[154] = d154
			ps923.OverlayValues[155] = d155
			ps923.OverlayValues[156] = d156
			ps923.OverlayValues[157] = d157
			ps923.OverlayValues[158] = d158
			ps923.OverlayValues[159] = d159
			ps923.OverlayValues[160] = d160
			ps923.OverlayValues[161] = d161
			ps923.OverlayValues[162] = d162
			ps923.OverlayValues[163] = d163
			ps923.OverlayValues[164] = d164
			ps923.OverlayValues[165] = d165
			ps923.OverlayValues[166] = d166
			ps923.OverlayValues[169] = d169
			ps923.OverlayValues[272] = d272
			ps923.OverlayValues[273] = d273
			ps923.OverlayValues[274] = d274
			ps923.OverlayValues[275] = d275
			ps923.OverlayValues[277] = d277
			ps923.OverlayValues[278] = d278
			ps923.OverlayValues[279] = d279
			ps923.OverlayValues[280] = d280
			ps923.OverlayValues[281] = d281
			ps923.OverlayValues[282] = d282
			ps923.OverlayValues[283] = d283
			ps923.OverlayValues[284] = d284
			ps923.OverlayValues[286] = d286
			ps923.OverlayValues[288] = d288
			ps923.OverlayValues[289] = d289
			ps923.OverlayValues[290] = d290
			ps923.OverlayValues[291] = d291
			ps923.OverlayValues[292] = d292
			ps923.OverlayValues[295] = d295
			ps923.OverlayValues[415] = d415
			ps923.OverlayValues[416] = d416
			ps923.OverlayValues[417] = d417
			ps923.OverlayValues[418] = d418
			ps923.OverlayValues[419] = d419
			ps923.OverlayValues[421] = d421
			ps923.OverlayValues[422] = d422
			ps923.OverlayValues[423] = d423
			ps923.OverlayValues[425] = d425
			ps923.OverlayValues[426] = d426
			ps923.OverlayValues[427] = d427
			ps923.OverlayValues[428] = d428
			ps923.OverlayValues[429] = d429
			ps923.OverlayValues[430] = d430
			ps923.OverlayValues[431] = d431
			ps923.OverlayValues[432] = d432
			ps923.OverlayValues[433] = d433
			ps923.OverlayValues[434] = d434
			ps923.OverlayValues[435] = d435
			ps923.OverlayValues[436] = d436
			ps923.OverlayValues[437] = d437
			ps923.OverlayValues[438] = d438
			ps923.OverlayValues[439] = d439
			ps923.OverlayValues[440] = d440
			ps923.OverlayValues[441] = d441
			ps923.OverlayValues[442] = d442
			ps923.OverlayValues[443] = d443
			ps923.OverlayValues[444] = d444
			ps923.OverlayValues[445] = d445
			ps923.OverlayValues[446] = d446
			ps923.OverlayValues[447] = d447
			ps923.OverlayValues[448] = d448
			ps923.OverlayValues[449] = d449
			ps923.OverlayValues[450] = d450
			ps923.OverlayValues[451] = d451
			ps923.OverlayValues[452] = d452
			ps923.OverlayValues[453] = d453
			ps923.OverlayValues[454] = d454
			ps923.OverlayValues[455] = d455
			ps923.OverlayValues[456] = d456
			ps923.OverlayValues[457] = d457
			ps923.OverlayValues[458] = d458
			ps923.OverlayValues[459] = d459
			ps923.OverlayValues[626] = d626
			ps923.OverlayValues[627] = d627
			ps923.OverlayValues[628] = d628
			ps923.OverlayValues[630] = d630
			ps923.OverlayValues[631] = d631
			ps923.OverlayValues[632] = d632
			ps923.OverlayValues[633] = d633
			ps923.OverlayValues[634] = d634
			ps923.OverlayValues[635] = d635
			ps923.OverlayValues[636] = d636
			ps923.OverlayValues[638] = d638
			ps923.OverlayValues[640] = d640
			ps923.OverlayValues[641] = d641
			ps923.OverlayValues[642] = d642
			ps923.OverlayValues[643] = d643
			ps923.OverlayValues[646] = d646
			ps923.OverlayValues[825] = d825
			ps923.OverlayValues[826] = d826
			ps923.OverlayValues[827] = d827
			ps923.OverlayValues[828] = d828
			ps923.OverlayValues[830] = d830
			ps923.OverlayValues[831] = d831
			ps923.OverlayValues[832] = d832
			ps923.OverlayValues[833] = d833
			ps923.OverlayValues[834] = d834
			ps923.OverlayValues[835] = d835
			ps923.OverlayValues[836] = d836
			ps923.OverlayValues[837] = d837
			ps923.OverlayValues[839] = d839
			ps923.OverlayValues[840] = d840
			ps923.OverlayValues[841] = d841
			ps923.OverlayValues[842] = d842
			ps923.OverlayValues[843] = d843
			ps923.OverlayValues[845] = d845
			ps923.OverlayValues[846] = d846
			ps923.OverlayValues[847] = d847
			ps923.OverlayValues[848] = d848
			ps923.OverlayValues[849] = d849
			ps923.OverlayValues[850] = d850
			ps923.OverlayValues[851] = d851
			ps923.OverlayValues[852] = d852
			ps923.OverlayValues[853] = d853
			ps923.OverlayValues[854] = d854
			ps923.OverlayValues[855] = d855
			ps923.OverlayValues[856] = d856
			ps923.OverlayValues[857] = d857
			ps923.OverlayValues[858] = d858
			ps923.OverlayValues[859] = d859
			ps923.OverlayValues[860] = d860
			ps923.OverlayValues[861] = d861
			ps923.OverlayValues[862] = d862
			ps923.OverlayValues[863] = d863
			ps923.OverlayValues[864] = d864
			ps923.OverlayValues[865] = d865
			ps923.OverlayValues[866] = d866
			ps923.OverlayValues[867] = d867
			ps923.OverlayValues[868] = d868
			ps923.OverlayValues[869] = d869
			ps923.OverlayValues[870] = d870
			ps923.OverlayValues[871] = d871
			ps923.OverlayValues[872] = d872
			ps923.OverlayValues[873] = d873
			ps923.OverlayValues[874] = d874
			ps923.OverlayValues[875] = d875
			ps923.OverlayValues[876] = d876
			ps923.OverlayValues[877] = d877
			ps923.OverlayValues[878] = d878
			ps923.OverlayValues[880] = d880
			ps923.OverlayValues[881] = d881
			ps923.OverlayValues[882] = d882
			ps923.OverlayValues[883] = d883
			ps923.OverlayValues[884] = d884
			ps923.OverlayValues[885] = d885
			ps923.OverlayValues[886] = d886
			ps923.OverlayValues[887] = d887
			ps923.OverlayValues[888] = d888
			ps923.OverlayValues[889] = d889
			ps923.OverlayValues[890] = d890
			ps923.OverlayValues[891] = d891
			ps923.OverlayValues[892] = d892
			ps923.OverlayValues[893] = d893
			ps923.OverlayValues[894] = d894
			ps923.OverlayValues[895] = d895
			ps923.OverlayValues[896] = d896
			ps923.OverlayValues[897] = d897
			ps923.OverlayValues[898] = d898
			ps923.OverlayValues[899] = d899
			ps923.OverlayValues[900] = d900
			ps923.OverlayValues[901] = d901
			ps923.OverlayValues[902] = d902
			ps923.OverlayValues[903] = d903
			ps923.OverlayValues[904] = d904
			ps923.OverlayValues[905] = d905
			ps923.OverlayValues[906] = d906
			ps923.OverlayValues[907] = d907
			ps923.OverlayValues[908] = d908
			ps923.OverlayValues[909] = d909
			ps923.OverlayValues[910] = d910
			ps923.OverlayValues[911] = d911
			ps923.OverlayValues[912] = d912
			ps923.OverlayValues[913] = d913
			ps923.OverlayValues[914] = d914
			ps923.OverlayValues[915] = d915
			ps923.OverlayValues[916] = d916
			ps923.OverlayValues[917] = d917
			ps923.OverlayValues[918] = d918
			ps923.OverlayValues[919] = d919
			ps923.OverlayValues[920] = d920
			ps923.OverlayValues[921] = d921
			return bbs[12].RenderPS(ps923)
		}
		if !ps.General {
			ps.General = true
			return bbs[13].RenderPS(ps)
		}
		lbl50 := ctx.ReserveLabel()
		lbl51 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d921.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl50)
		ctx.EmitJmp(lbl51)
		ctx.MarkLabel(lbl50)
		ctx.EmitJmp(lbl12)
		ctx.MarkLabel(lbl51)
		ctx.EmitJmp(lbl13)
		ps924 := scm.PhiState{General: true}
		ps924.OverlayValues = make([]scm.JITValueDesc, 922)
		ps924.OverlayValues[1] = d1
		ps924.OverlayValues[2] = d2
		ps924.OverlayValues[3] = d3
		ps924.OverlayValues[4] = d4
		ps924.OverlayValues[5] = d5
		ps924.OverlayValues[6] = d6
		ps924.OverlayValues[7] = d7
		ps924.OverlayValues[8] = d8
		ps924.OverlayValues[9] = d9
		ps924.OverlayValues[10] = d10
		ps924.OverlayValues[11] = d11
		ps924.OverlayValues[12] = d12
		ps924.OverlayValues[13] = d13
		ps924.OverlayValues[14] = d14
		ps924.OverlayValues[15] = d15
		ps924.OverlayValues[17] = d17
		ps924.OverlayValues[18] = d18
		ps924.OverlayValues[19] = d19
		ps924.OverlayValues[20] = d20
		ps924.OverlayValues[21] = d21
		ps924.OverlayValues[22] = d22
		ps924.OverlayValues[24] = d24
		ps924.OverlayValues[25] = d25
		ps924.OverlayValues[26] = d26
		ps924.OverlayValues[27] = d27
		ps924.OverlayValues[28] = d28
		ps924.OverlayValues[29] = d29
		ps924.OverlayValues[30] = d30
		ps924.OverlayValues[31] = d31
		ps924.OverlayValues[32] = d32
		ps924.OverlayValues[33] = d33
		ps924.OverlayValues[34] = d34
		ps924.OverlayValues[35] = d35
		ps924.OverlayValues[36] = d36
		ps924.OverlayValues[37] = d37
		ps924.OverlayValues[38] = d38
		ps924.OverlayValues[39] = d39
		ps924.OverlayValues[40] = d40
		ps924.OverlayValues[41] = d41
		ps924.OverlayValues[42] = d42
		ps924.OverlayValues[43] = d43
		ps924.OverlayValues[44] = d44
		ps924.OverlayValues[45] = d45
		ps924.OverlayValues[46] = d46
		ps924.OverlayValues[47] = d47
		ps924.OverlayValues[48] = d48
		ps924.OverlayValues[49] = d49
		ps924.OverlayValues[50] = d50
		ps924.OverlayValues[51] = d51
		ps924.OverlayValues[52] = d52
		ps924.OverlayValues[53] = d53
		ps924.OverlayValues[54] = d54
		ps924.OverlayValues[55] = d55
		ps924.OverlayValues[56] = d56
		ps924.OverlayValues[57] = d57
		ps924.OverlayValues[58] = d58
		ps924.OverlayValues[59] = d59
		ps924.OverlayValues[62] = d62
		ps924.OverlayValues[63] = d63
		ps924.OverlayValues[64] = d64
		ps924.OverlayValues[128] = d128
		ps924.OverlayValues[129] = d129
		ps924.OverlayValues[130] = d130
		ps924.OverlayValues[132] = d132
		ps924.OverlayValues[133] = d133
		ps924.OverlayValues[134] = d134
		ps924.OverlayValues[135] = d135
		ps924.OverlayValues[136] = d136
		ps924.OverlayValues[137] = d137
		ps924.OverlayValues[138] = d138
		ps924.OverlayValues[139] = d139
		ps924.OverlayValues[140] = d140
		ps924.OverlayValues[141] = d141
		ps924.OverlayValues[142] = d142
		ps924.OverlayValues[143] = d143
		ps924.OverlayValues[144] = d144
		ps924.OverlayValues[145] = d145
		ps924.OverlayValues[146] = d146
		ps924.OverlayValues[147] = d147
		ps924.OverlayValues[148] = d148
		ps924.OverlayValues[149] = d149
		ps924.OverlayValues[150] = d150
		ps924.OverlayValues[151] = d151
		ps924.OverlayValues[152] = d152
		ps924.OverlayValues[153] = d153
		ps924.OverlayValues[154] = d154
		ps924.OverlayValues[155] = d155
		ps924.OverlayValues[156] = d156
		ps924.OverlayValues[157] = d157
		ps924.OverlayValues[158] = d158
		ps924.OverlayValues[159] = d159
		ps924.OverlayValues[160] = d160
		ps924.OverlayValues[161] = d161
		ps924.OverlayValues[162] = d162
		ps924.OverlayValues[163] = d163
		ps924.OverlayValues[164] = d164
		ps924.OverlayValues[165] = d165
		ps924.OverlayValues[166] = d166
		ps924.OverlayValues[169] = d169
		ps924.OverlayValues[272] = d272
		ps924.OverlayValues[273] = d273
		ps924.OverlayValues[274] = d274
		ps924.OverlayValues[275] = d275
		ps924.OverlayValues[277] = d277
		ps924.OverlayValues[278] = d278
		ps924.OverlayValues[279] = d279
		ps924.OverlayValues[280] = d280
		ps924.OverlayValues[281] = d281
		ps924.OverlayValues[282] = d282
		ps924.OverlayValues[283] = d283
		ps924.OverlayValues[284] = d284
		ps924.OverlayValues[286] = d286
		ps924.OverlayValues[288] = d288
		ps924.OverlayValues[289] = d289
		ps924.OverlayValues[290] = d290
		ps924.OverlayValues[291] = d291
		ps924.OverlayValues[292] = d292
		ps924.OverlayValues[295] = d295
		ps924.OverlayValues[415] = d415
		ps924.OverlayValues[416] = d416
		ps924.OverlayValues[417] = d417
		ps924.OverlayValues[418] = d418
		ps924.OverlayValues[419] = d419
		ps924.OverlayValues[421] = d421
		ps924.OverlayValues[422] = d422
		ps924.OverlayValues[423] = d423
		ps924.OverlayValues[425] = d425
		ps924.OverlayValues[426] = d426
		ps924.OverlayValues[427] = d427
		ps924.OverlayValues[428] = d428
		ps924.OverlayValues[429] = d429
		ps924.OverlayValues[430] = d430
		ps924.OverlayValues[431] = d431
		ps924.OverlayValues[432] = d432
		ps924.OverlayValues[433] = d433
		ps924.OverlayValues[434] = d434
		ps924.OverlayValues[435] = d435
		ps924.OverlayValues[436] = d436
		ps924.OverlayValues[437] = d437
		ps924.OverlayValues[438] = d438
		ps924.OverlayValues[439] = d439
		ps924.OverlayValues[440] = d440
		ps924.OverlayValues[441] = d441
		ps924.OverlayValues[442] = d442
		ps924.OverlayValues[443] = d443
		ps924.OverlayValues[444] = d444
		ps924.OverlayValues[445] = d445
		ps924.OverlayValues[446] = d446
		ps924.OverlayValues[447] = d447
		ps924.OverlayValues[448] = d448
		ps924.OverlayValues[449] = d449
		ps924.OverlayValues[450] = d450
		ps924.OverlayValues[451] = d451
		ps924.OverlayValues[452] = d452
		ps924.OverlayValues[453] = d453
		ps924.OverlayValues[454] = d454
		ps924.OverlayValues[455] = d455
		ps924.OverlayValues[456] = d456
		ps924.OverlayValues[457] = d457
		ps924.OverlayValues[458] = d458
		ps924.OverlayValues[459] = d459
		ps924.OverlayValues[626] = d626
		ps924.OverlayValues[627] = d627
		ps924.OverlayValues[628] = d628
		ps924.OverlayValues[630] = d630
		ps924.OverlayValues[631] = d631
		ps924.OverlayValues[632] = d632
		ps924.OverlayValues[633] = d633
		ps924.OverlayValues[634] = d634
		ps924.OverlayValues[635] = d635
		ps924.OverlayValues[636] = d636
		ps924.OverlayValues[638] = d638
		ps924.OverlayValues[640] = d640
		ps924.OverlayValues[641] = d641
		ps924.OverlayValues[642] = d642
		ps924.OverlayValues[643] = d643
		ps924.OverlayValues[646] = d646
		ps924.OverlayValues[825] = d825
		ps924.OverlayValues[826] = d826
		ps924.OverlayValues[827] = d827
		ps924.OverlayValues[828] = d828
		ps924.OverlayValues[830] = d830
		ps924.OverlayValues[831] = d831
		ps924.OverlayValues[832] = d832
		ps924.OverlayValues[833] = d833
		ps924.OverlayValues[834] = d834
		ps924.OverlayValues[835] = d835
		ps924.OverlayValues[836] = d836
		ps924.OverlayValues[837] = d837
		ps924.OverlayValues[839] = d839
		ps924.OverlayValues[840] = d840
		ps924.OverlayValues[841] = d841
		ps924.OverlayValues[842] = d842
		ps924.OverlayValues[843] = d843
		ps924.OverlayValues[845] = d845
		ps924.OverlayValues[846] = d846
		ps924.OverlayValues[847] = d847
		ps924.OverlayValues[848] = d848
		ps924.OverlayValues[849] = d849
		ps924.OverlayValues[850] = d850
		ps924.OverlayValues[851] = d851
		ps924.OverlayValues[852] = d852
		ps924.OverlayValues[853] = d853
		ps924.OverlayValues[854] = d854
		ps924.OverlayValues[855] = d855
		ps924.OverlayValues[856] = d856
		ps924.OverlayValues[857] = d857
		ps924.OverlayValues[858] = d858
		ps924.OverlayValues[859] = d859
		ps924.OverlayValues[860] = d860
		ps924.OverlayValues[861] = d861
		ps924.OverlayValues[862] = d862
		ps924.OverlayValues[863] = d863
		ps924.OverlayValues[864] = d864
		ps924.OverlayValues[865] = d865
		ps924.OverlayValues[866] = d866
		ps924.OverlayValues[867] = d867
		ps924.OverlayValues[868] = d868
		ps924.OverlayValues[869] = d869
		ps924.OverlayValues[870] = d870
		ps924.OverlayValues[871] = d871
		ps924.OverlayValues[872] = d872
		ps924.OverlayValues[873] = d873
		ps924.OverlayValues[874] = d874
		ps924.OverlayValues[875] = d875
		ps924.OverlayValues[876] = d876
		ps924.OverlayValues[877] = d877
		ps924.OverlayValues[878] = d878
		ps924.OverlayValues[880] = d880
		ps924.OverlayValues[881] = d881
		ps924.OverlayValues[882] = d882
		ps924.OverlayValues[883] = d883
		ps924.OverlayValues[884] = d884
		ps924.OverlayValues[885] = d885
		ps924.OverlayValues[886] = d886
		ps924.OverlayValues[887] = d887
		ps924.OverlayValues[888] = d888
		ps924.OverlayValues[889] = d889
		ps924.OverlayValues[890] = d890
		ps924.OverlayValues[891] = d891
		ps924.OverlayValues[892] = d892
		ps924.OverlayValues[893] = d893
		ps924.OverlayValues[894] = d894
		ps924.OverlayValues[895] = d895
		ps924.OverlayValues[896] = d896
		ps924.OverlayValues[897] = d897
		ps924.OverlayValues[898] = d898
		ps924.OverlayValues[899] = d899
		ps924.OverlayValues[900] = d900
		ps924.OverlayValues[901] = d901
		ps924.OverlayValues[902] = d902
		ps924.OverlayValues[903] = d903
		ps924.OverlayValues[904] = d904
		ps924.OverlayValues[905] = d905
		ps924.OverlayValues[906] = d906
		ps924.OverlayValues[907] = d907
		ps924.OverlayValues[908] = d908
		ps924.OverlayValues[909] = d909
		ps924.OverlayValues[910] = d910
		ps924.OverlayValues[911] = d911
		ps924.OverlayValues[912] = d912
		ps924.OverlayValues[913] = d913
		ps924.OverlayValues[914] = d914
		ps924.OverlayValues[915] = d915
		ps924.OverlayValues[916] = d916
		ps924.OverlayValues[917] = d917
		ps924.OverlayValues[918] = d918
		ps924.OverlayValues[919] = d919
		ps924.OverlayValues[920] = d920
		ps924.OverlayValues[921] = d921
		ps925 := scm.PhiState{General: true}
		ps925.OverlayValues = make([]scm.JITValueDesc, 922)
		ps925.OverlayValues[1] = d1
		ps925.OverlayValues[2] = d2
		ps925.OverlayValues[3] = d3
		ps925.OverlayValues[4] = d4
		ps925.OverlayValues[5] = d5
		ps925.OverlayValues[6] = d6
		ps925.OverlayValues[7] = d7
		ps925.OverlayValues[8] = d8
		ps925.OverlayValues[9] = d9
		ps925.OverlayValues[10] = d10
		ps925.OverlayValues[11] = d11
		ps925.OverlayValues[12] = d12
		ps925.OverlayValues[13] = d13
		ps925.OverlayValues[14] = d14
		ps925.OverlayValues[15] = d15
		ps925.OverlayValues[17] = d17
		ps925.OverlayValues[18] = d18
		ps925.OverlayValues[19] = d19
		ps925.OverlayValues[20] = d20
		ps925.OverlayValues[21] = d21
		ps925.OverlayValues[22] = d22
		ps925.OverlayValues[24] = d24
		ps925.OverlayValues[25] = d25
		ps925.OverlayValues[26] = d26
		ps925.OverlayValues[27] = d27
		ps925.OverlayValues[28] = d28
		ps925.OverlayValues[29] = d29
		ps925.OverlayValues[30] = d30
		ps925.OverlayValues[31] = d31
		ps925.OverlayValues[32] = d32
		ps925.OverlayValues[33] = d33
		ps925.OverlayValues[34] = d34
		ps925.OverlayValues[35] = d35
		ps925.OverlayValues[36] = d36
		ps925.OverlayValues[37] = d37
		ps925.OverlayValues[38] = d38
		ps925.OverlayValues[39] = d39
		ps925.OverlayValues[40] = d40
		ps925.OverlayValues[41] = d41
		ps925.OverlayValues[42] = d42
		ps925.OverlayValues[43] = d43
		ps925.OverlayValues[44] = d44
		ps925.OverlayValues[45] = d45
		ps925.OverlayValues[46] = d46
		ps925.OverlayValues[47] = d47
		ps925.OverlayValues[48] = d48
		ps925.OverlayValues[49] = d49
		ps925.OverlayValues[50] = d50
		ps925.OverlayValues[51] = d51
		ps925.OverlayValues[52] = d52
		ps925.OverlayValues[53] = d53
		ps925.OverlayValues[54] = d54
		ps925.OverlayValues[55] = d55
		ps925.OverlayValues[56] = d56
		ps925.OverlayValues[57] = d57
		ps925.OverlayValues[58] = d58
		ps925.OverlayValues[59] = d59
		ps925.OverlayValues[62] = d62
		ps925.OverlayValues[63] = d63
		ps925.OverlayValues[64] = d64
		ps925.OverlayValues[128] = d128
		ps925.OverlayValues[129] = d129
		ps925.OverlayValues[130] = d130
		ps925.OverlayValues[132] = d132
		ps925.OverlayValues[133] = d133
		ps925.OverlayValues[134] = d134
		ps925.OverlayValues[135] = d135
		ps925.OverlayValues[136] = d136
		ps925.OverlayValues[137] = d137
		ps925.OverlayValues[138] = d138
		ps925.OverlayValues[139] = d139
		ps925.OverlayValues[140] = d140
		ps925.OverlayValues[141] = d141
		ps925.OverlayValues[142] = d142
		ps925.OverlayValues[143] = d143
		ps925.OverlayValues[144] = d144
		ps925.OverlayValues[145] = d145
		ps925.OverlayValues[146] = d146
		ps925.OverlayValues[147] = d147
		ps925.OverlayValues[148] = d148
		ps925.OverlayValues[149] = d149
		ps925.OverlayValues[150] = d150
		ps925.OverlayValues[151] = d151
		ps925.OverlayValues[152] = d152
		ps925.OverlayValues[153] = d153
		ps925.OverlayValues[154] = d154
		ps925.OverlayValues[155] = d155
		ps925.OverlayValues[156] = d156
		ps925.OverlayValues[157] = d157
		ps925.OverlayValues[158] = d158
		ps925.OverlayValues[159] = d159
		ps925.OverlayValues[160] = d160
		ps925.OverlayValues[161] = d161
		ps925.OverlayValues[162] = d162
		ps925.OverlayValues[163] = d163
		ps925.OverlayValues[164] = d164
		ps925.OverlayValues[165] = d165
		ps925.OverlayValues[166] = d166
		ps925.OverlayValues[169] = d169
		ps925.OverlayValues[272] = d272
		ps925.OverlayValues[273] = d273
		ps925.OverlayValues[274] = d274
		ps925.OverlayValues[275] = d275
		ps925.OverlayValues[277] = d277
		ps925.OverlayValues[278] = d278
		ps925.OverlayValues[279] = d279
		ps925.OverlayValues[280] = d280
		ps925.OverlayValues[281] = d281
		ps925.OverlayValues[282] = d282
		ps925.OverlayValues[283] = d283
		ps925.OverlayValues[284] = d284
		ps925.OverlayValues[286] = d286
		ps925.OverlayValues[288] = d288
		ps925.OverlayValues[289] = d289
		ps925.OverlayValues[290] = d290
		ps925.OverlayValues[291] = d291
		ps925.OverlayValues[292] = d292
		ps925.OverlayValues[295] = d295
		ps925.OverlayValues[415] = d415
		ps925.OverlayValues[416] = d416
		ps925.OverlayValues[417] = d417
		ps925.OverlayValues[418] = d418
		ps925.OverlayValues[419] = d419
		ps925.OverlayValues[421] = d421
		ps925.OverlayValues[422] = d422
		ps925.OverlayValues[423] = d423
		ps925.OverlayValues[425] = d425
		ps925.OverlayValues[426] = d426
		ps925.OverlayValues[427] = d427
		ps925.OverlayValues[428] = d428
		ps925.OverlayValues[429] = d429
		ps925.OverlayValues[430] = d430
		ps925.OverlayValues[431] = d431
		ps925.OverlayValues[432] = d432
		ps925.OverlayValues[433] = d433
		ps925.OverlayValues[434] = d434
		ps925.OverlayValues[435] = d435
		ps925.OverlayValues[436] = d436
		ps925.OverlayValues[437] = d437
		ps925.OverlayValues[438] = d438
		ps925.OverlayValues[439] = d439
		ps925.OverlayValues[440] = d440
		ps925.OverlayValues[441] = d441
		ps925.OverlayValues[442] = d442
		ps925.OverlayValues[443] = d443
		ps925.OverlayValues[444] = d444
		ps925.OverlayValues[445] = d445
		ps925.OverlayValues[446] = d446
		ps925.OverlayValues[447] = d447
		ps925.OverlayValues[448] = d448
		ps925.OverlayValues[449] = d449
		ps925.OverlayValues[450] = d450
		ps925.OverlayValues[451] = d451
		ps925.OverlayValues[452] = d452
		ps925.OverlayValues[453] = d453
		ps925.OverlayValues[454] = d454
		ps925.OverlayValues[455] = d455
		ps925.OverlayValues[456] = d456
		ps925.OverlayValues[457] = d457
		ps925.OverlayValues[458] = d458
		ps925.OverlayValues[459] = d459
		ps925.OverlayValues[626] = d626
		ps925.OverlayValues[627] = d627
		ps925.OverlayValues[628] = d628
		ps925.OverlayValues[630] = d630
		ps925.OverlayValues[631] = d631
		ps925.OverlayValues[632] = d632
		ps925.OverlayValues[633] = d633
		ps925.OverlayValues[634] = d634
		ps925.OverlayValues[635] = d635
		ps925.OverlayValues[636] = d636
		ps925.OverlayValues[638] = d638
		ps925.OverlayValues[640] = d640
		ps925.OverlayValues[641] = d641
		ps925.OverlayValues[642] = d642
		ps925.OverlayValues[643] = d643
		ps925.OverlayValues[646] = d646
		ps925.OverlayValues[825] = d825
		ps925.OverlayValues[826] = d826
		ps925.OverlayValues[827] = d827
		ps925.OverlayValues[828] = d828
		ps925.OverlayValues[830] = d830
		ps925.OverlayValues[831] = d831
		ps925.OverlayValues[832] = d832
		ps925.OverlayValues[833] = d833
		ps925.OverlayValues[834] = d834
		ps925.OverlayValues[835] = d835
		ps925.OverlayValues[836] = d836
		ps925.OverlayValues[837] = d837
		ps925.OverlayValues[839] = d839
		ps925.OverlayValues[840] = d840
		ps925.OverlayValues[841] = d841
		ps925.OverlayValues[842] = d842
		ps925.OverlayValues[843] = d843
		ps925.OverlayValues[845] = d845
		ps925.OverlayValues[846] = d846
		ps925.OverlayValues[847] = d847
		ps925.OverlayValues[848] = d848
		ps925.OverlayValues[849] = d849
		ps925.OverlayValues[850] = d850
		ps925.OverlayValues[851] = d851
		ps925.OverlayValues[852] = d852
		ps925.OverlayValues[853] = d853
		ps925.OverlayValues[854] = d854
		ps925.OverlayValues[855] = d855
		ps925.OverlayValues[856] = d856
		ps925.OverlayValues[857] = d857
		ps925.OverlayValues[858] = d858
		ps925.OverlayValues[859] = d859
		ps925.OverlayValues[860] = d860
		ps925.OverlayValues[861] = d861
		ps925.OverlayValues[862] = d862
		ps925.OverlayValues[863] = d863
		ps925.OverlayValues[864] = d864
		ps925.OverlayValues[865] = d865
		ps925.OverlayValues[866] = d866
		ps925.OverlayValues[867] = d867
		ps925.OverlayValues[868] = d868
		ps925.OverlayValues[869] = d869
		ps925.OverlayValues[870] = d870
		ps925.OverlayValues[871] = d871
		ps925.OverlayValues[872] = d872
		ps925.OverlayValues[873] = d873
		ps925.OverlayValues[874] = d874
		ps925.OverlayValues[875] = d875
		ps925.OverlayValues[876] = d876
		ps925.OverlayValues[877] = d877
		ps925.OverlayValues[878] = d878
		ps925.OverlayValues[880] = d880
		ps925.OverlayValues[881] = d881
		ps925.OverlayValues[882] = d882
		ps925.OverlayValues[883] = d883
		ps925.OverlayValues[884] = d884
		ps925.OverlayValues[885] = d885
		ps925.OverlayValues[886] = d886
		ps925.OverlayValues[887] = d887
		ps925.OverlayValues[888] = d888
		ps925.OverlayValues[889] = d889
		ps925.OverlayValues[890] = d890
		ps925.OverlayValues[891] = d891
		ps925.OverlayValues[892] = d892
		ps925.OverlayValues[893] = d893
		ps925.OverlayValues[894] = d894
		ps925.OverlayValues[895] = d895
		ps925.OverlayValues[896] = d896
		ps925.OverlayValues[897] = d897
		ps925.OverlayValues[898] = d898
		ps925.OverlayValues[899] = d899
		ps925.OverlayValues[900] = d900
		ps925.OverlayValues[901] = d901
		ps925.OverlayValues[902] = d902
		ps925.OverlayValues[903] = d903
		ps925.OverlayValues[904] = d904
		ps925.OverlayValues[905] = d905
		ps925.OverlayValues[906] = d906
		ps925.OverlayValues[907] = d907
		ps925.OverlayValues[908] = d908
		ps925.OverlayValues[909] = d909
		ps925.OverlayValues[910] = d910
		ps925.OverlayValues[911] = d911
		ps925.OverlayValues[912] = d912
		ps925.OverlayValues[913] = d913
		ps925.OverlayValues[914] = d914
		ps925.OverlayValues[915] = d915
		ps925.OverlayValues[916] = d916
		ps925.OverlayValues[917] = d917
		ps925.OverlayValues[918] = d918
		ps925.OverlayValues[919] = d919
		ps925.OverlayValues[920] = d920
		ps925.OverlayValues[921] = d921
		snap926 := d1
		snap927 := d2
		snap928 := d3
		snap929 := d4
		snap930 := d5
		snap931 := d6
		snap932 := d7
		snap933 := d8
		snap934 := d9
		snap935 := d10
		snap936 := d11
		snap937 := d12
		snap938 := d13
		snap939 := d14
		snap940 := d15
		snap941 := d17
		snap942 := d18
		snap943 := d19
		snap944 := d20
		snap945 := d21
		snap946 := d22
		snap947 := d24
		snap948 := d25
		snap949 := d26
		snap950 := d27
		snap951 := d28
		snap952 := d29
		snap953 := d30
		snap954 := d31
		snap955 := d32
		snap956 := d33
		snap957 := d34
		snap958 := d35
		snap959 := d36
		snap960 := d37
		snap961 := d38
		snap962 := d39
		snap963 := d40
		snap964 := d41
		snap965 := d42
		snap966 := d43
		snap967 := d44
		snap968 := d45
		snap969 := d46
		snap970 := d47
		snap971 := d48
		snap972 := d49
		snap973 := d50
		snap974 := d51
		snap975 := d52
		snap976 := d53
		snap977 := d54
		snap978 := d55
		snap979 := d56
		snap980 := d57
		snap981 := d58
		snap982 := d59
		snap983 := d62
		snap984 := d63
		snap985 := d64
		snap986 := d128
		snap987 := d129
		snap988 := d130
		snap989 := d132
		snap990 := d133
		snap991 := d134
		snap992 := d135
		snap993 := d136
		snap994 := d137
		snap995 := d138
		snap996 := d139
		snap997 := d140
		snap998 := d141
		snap999 := d142
		snap1000 := d143
		snap1001 := d144
		snap1002 := d145
		snap1003 := d146
		snap1004 := d147
		snap1005 := d148
		snap1006 := d149
		snap1007 := d150
		snap1008 := d151
		snap1009 := d152
		snap1010 := d153
		snap1011 := d154
		snap1012 := d155
		snap1013 := d156
		snap1014 := d157
		snap1015 := d158
		snap1016 := d159
		snap1017 := d160
		snap1018 := d161
		snap1019 := d162
		snap1020 := d163
		snap1021 := d164
		snap1022 := d165
		snap1023 := d166
		snap1024 := d169
		snap1025 := d272
		snap1026 := d273
		snap1027 := d274
		snap1028 := d275
		snap1029 := d277
		snap1030 := d278
		snap1031 := d279
		snap1032 := d280
		snap1033 := d281
		snap1034 := d282
		snap1035 := d283
		snap1036 := d284
		snap1037 := d286
		snap1038 := d288
		snap1039 := d289
		snap1040 := d290
		snap1041 := d291
		snap1042 := d292
		snap1043 := d295
		snap1044 := d415
		snap1045 := d416
		snap1046 := d417
		snap1047 := d418
		snap1048 := d419
		snap1049 := d421
		snap1050 := d422
		snap1051 := d423
		snap1052 := d425
		snap1053 := d426
		snap1054 := d427
		snap1055 := d428
		snap1056 := d429
		snap1057 := d430
		snap1058 := d431
		snap1059 := d432
		snap1060 := d433
		snap1061 := d434
		snap1062 := d435
		snap1063 := d436
		snap1064 := d437
		snap1065 := d438
		snap1066 := d439
		snap1067 := d440
		snap1068 := d441
		snap1069 := d442
		snap1070 := d443
		snap1071 := d444
		snap1072 := d445
		snap1073 := d446
		snap1074 := d447
		snap1075 := d448
		snap1076 := d449
		snap1077 := d450
		snap1078 := d451
		snap1079 := d452
		snap1080 := d453
		snap1081 := d454
		snap1082 := d455
		snap1083 := d456
		snap1084 := d457
		snap1085 := d458
		snap1086 := d459
		snap1087 := d626
		snap1088 := d627
		snap1089 := d628
		snap1090 := d630
		snap1091 := d631
		snap1092 := d632
		snap1093 := d633
		snap1094 := d634
		snap1095 := d635
		snap1096 := d636
		snap1097 := d638
		snap1098 := d640
		snap1099 := d641
		snap1100 := d642
		snap1101 := d643
		snap1102 := d646
		snap1103 := d825
		snap1104 := d826
		snap1105 := d827
		snap1106 := d828
		snap1107 := d830
		snap1108 := d831
		snap1109 := d832
		snap1110 := d833
		snap1111 := d834
		snap1112 := d835
		snap1113 := d836
		snap1114 := d837
		snap1115 := d839
		snap1116 := d840
		snap1117 := d841
		snap1118 := d842
		snap1119 := d843
		snap1120 := d845
		snap1121 := d846
		snap1122 := d847
		snap1123 := d848
		snap1124 := d849
		snap1125 := d850
		snap1126 := d851
		snap1127 := d852
		snap1128 := d853
		snap1129 := d854
		snap1130 := d855
		snap1131 := d856
		snap1132 := d857
		snap1133 := d858
		snap1134 := d859
		snap1135 := d860
		snap1136 := d861
		snap1137 := d862
		snap1138 := d863
		snap1139 := d864
		snap1140 := d865
		snap1141 := d866
		snap1142 := d867
		snap1143 := d868
		snap1144 := d869
		snap1145 := d870
		snap1146 := d871
		snap1147 := d872
		snap1148 := d873
		snap1149 := d874
		snap1150 := d875
		snap1151 := d876
		snap1152 := d877
		snap1153 := d878
		snap1154 := d880
		snap1155 := d881
		snap1156 := d882
		snap1157 := d883
		snap1158 := d884
		snap1159 := d885
		snap1160 := d886
		snap1161 := d887
		snap1162 := d888
		snap1163 := d889
		snap1164 := d890
		snap1165 := d891
		snap1166 := d892
		snap1167 := d893
		snap1168 := d894
		snap1169 := d895
		snap1170 := d896
		snap1171 := d897
		snap1172 := d898
		snap1173 := d899
		snap1174 := d900
		snap1175 := d901
		snap1176 := d902
		snap1177 := d903
		snap1178 := d904
		snap1179 := d905
		snap1180 := d906
		snap1181 := d907
		snap1182 := d908
		snap1183 := d909
		snap1184 := d910
		snap1185 := d911
		snap1186 := d912
		snap1187 := d913
		snap1188 := d914
		snap1189 := d915
		snap1190 := d916
		snap1191 := d917
		snap1192 := d918
		snap1193 := d919
		snap1194 := d920
		snap1195 := d921
		alloc1196 := ctx.SnapshotAllocState()
		if !bbs[12].Rendered {
			bbs[12].RenderPS(ps925)
		}
		ctx.RestoreAllocState(alloc1196)
		d1 = snap926
		d2 = snap927
		d3 = snap928
		d4 = snap929
		d5 = snap930
		d6 = snap931
		d7 = snap932
		d8 = snap933
		d9 = snap934
		d10 = snap935
		d11 = snap936
		d12 = snap937
		d13 = snap938
		d14 = snap939
		d15 = snap940
		d17 = snap941
		d18 = snap942
		d19 = snap943
		d20 = snap944
		d21 = snap945
		d22 = snap946
		d24 = snap947
		d25 = snap948
		d26 = snap949
		d27 = snap950
		d28 = snap951
		d29 = snap952
		d30 = snap953
		d31 = snap954
		d32 = snap955
		d33 = snap956
		d34 = snap957
		d35 = snap958
		d36 = snap959
		d37 = snap960
		d38 = snap961
		d39 = snap962
		d40 = snap963
		d41 = snap964
		d42 = snap965
		d43 = snap966
		d44 = snap967
		d45 = snap968
		d46 = snap969
		d47 = snap970
		d48 = snap971
		d49 = snap972
		d50 = snap973
		d51 = snap974
		d52 = snap975
		d53 = snap976
		d54 = snap977
		d55 = snap978
		d56 = snap979
		d57 = snap980
		d58 = snap981
		d59 = snap982
		d62 = snap983
		d63 = snap984
		d64 = snap985
		d128 = snap986
		d129 = snap987
		d130 = snap988
		d132 = snap989
		d133 = snap990
		d134 = snap991
		d135 = snap992
		d136 = snap993
		d137 = snap994
		d138 = snap995
		d139 = snap996
		d140 = snap997
		d141 = snap998
		d142 = snap999
		d143 = snap1000
		d144 = snap1001
		d145 = snap1002
		d146 = snap1003
		d147 = snap1004
		d148 = snap1005
		d149 = snap1006
		d150 = snap1007
		d151 = snap1008
		d152 = snap1009
		d153 = snap1010
		d154 = snap1011
		d155 = snap1012
		d156 = snap1013
		d157 = snap1014
		d158 = snap1015
		d159 = snap1016
		d160 = snap1017
		d161 = snap1018
		d162 = snap1019
		d163 = snap1020
		d164 = snap1021
		d165 = snap1022
		d166 = snap1023
		d169 = snap1024
		d272 = snap1025
		d273 = snap1026
		d274 = snap1027
		d275 = snap1028
		d277 = snap1029
		d278 = snap1030
		d279 = snap1031
		d280 = snap1032
		d281 = snap1033
		d282 = snap1034
		d283 = snap1035
		d284 = snap1036
		d286 = snap1037
		d288 = snap1038
		d289 = snap1039
		d290 = snap1040
		d291 = snap1041
		d292 = snap1042
		d295 = snap1043
		d415 = snap1044
		d416 = snap1045
		d417 = snap1046
		d418 = snap1047
		d419 = snap1048
		d421 = snap1049
		d422 = snap1050
		d423 = snap1051
		d425 = snap1052
		d426 = snap1053
		d427 = snap1054
		d428 = snap1055
		d429 = snap1056
		d430 = snap1057
		d431 = snap1058
		d432 = snap1059
		d433 = snap1060
		d434 = snap1061
		d435 = snap1062
		d436 = snap1063
		d437 = snap1064
		d438 = snap1065
		d439 = snap1066
		d440 = snap1067
		d441 = snap1068
		d442 = snap1069
		d443 = snap1070
		d444 = snap1071
		d445 = snap1072
		d446 = snap1073
		d447 = snap1074
		d448 = snap1075
		d449 = snap1076
		d450 = snap1077
		d451 = snap1078
		d452 = snap1079
		d453 = snap1080
		d454 = snap1081
		d455 = snap1082
		d456 = snap1083
		d457 = snap1084
		d458 = snap1085
		d459 = snap1086
		d626 = snap1087
		d627 = snap1088
		d628 = snap1089
		d630 = snap1090
		d631 = snap1091
		d632 = snap1092
		d633 = snap1093
		d634 = snap1094
		d635 = snap1095
		d636 = snap1096
		d638 = snap1097
		d640 = snap1098
		d641 = snap1099
		d642 = snap1100
		d643 = snap1101
		d646 = snap1102
		d825 = snap1103
		d826 = snap1104
		d827 = snap1105
		d828 = snap1106
		d830 = snap1107
		d831 = snap1108
		d832 = snap1109
		d833 = snap1110
		d834 = snap1111
		d835 = snap1112
		d836 = snap1113
		d837 = snap1114
		d839 = snap1115
		d840 = snap1116
		d841 = snap1117
		d842 = snap1118
		d843 = snap1119
		d845 = snap1120
		d846 = snap1121
		d847 = snap1122
		d848 = snap1123
		d849 = snap1124
		d850 = snap1125
		d851 = snap1126
		d852 = snap1127
		d853 = snap1128
		d854 = snap1129
		d855 = snap1130
		d856 = snap1131
		d857 = snap1132
		d858 = snap1133
		d859 = snap1134
		d860 = snap1135
		d861 = snap1136
		d862 = snap1137
		d863 = snap1138
		d864 = snap1139
		d865 = snap1140
		d866 = snap1141
		d867 = snap1142
		d868 = snap1143
		d869 = snap1144
		d870 = snap1145
		d871 = snap1146
		d872 = snap1147
		d873 = snap1148
		d874 = snap1149
		d875 = snap1150
		d876 = snap1151
		d877 = snap1152
		d878 = snap1153
		d880 = snap1154
		d881 = snap1155
		d882 = snap1156
		d883 = snap1157
		d884 = snap1158
		d885 = snap1159
		d886 = snap1160
		d887 = snap1161
		d888 = snap1162
		d889 = snap1163
		d890 = snap1164
		d891 = snap1165
		d892 = snap1166
		d893 = snap1167
		d894 = snap1168
		d895 = snap1169
		d896 = snap1170
		d897 = snap1171
		d898 = snap1172
		d899 = snap1173
		d900 = snap1174
		d901 = snap1175
		d902 = snap1176
		d903 = snap1177
		d904 = snap1178
		d905 = snap1179
		d906 = snap1180
		d907 = snap1181
		d908 = snap1182
		d909 = snap1183
		d910 = snap1184
		d911 = snap1185
		d912 = snap1186
		d913 = snap1187
		d914 = snap1188
		d915 = snap1189
		d916 = snap1190
		d917 = snap1191
		d918 = snap1192
		d919 = snap1193
		d920 = snap1194
		d921 = snap1195
		if !bbs[11].Rendered {
			return bbs[11].RenderPS(ps924)
		}
		return result
		ctx.FreeDesc(&d920)
		return result
	}
	ps1197 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps1197)
	ctx.MarkLabel(lbl0)
	d1198 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
	ctx.BindReg(r0, &d1198)
	ctx.BindReg(r1, &d1198)
	ctx.EmitMovPairToResult(&d1198, &result)
	ctx.FreeReg(r0)
	ctx.FreeReg(r1)
	ctx.ResolveFixups()
	if idxPinned {
		ctx.UnprotectReg(idxPinnedReg)
	}
	ctx.FreeStack(int32(144))
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
