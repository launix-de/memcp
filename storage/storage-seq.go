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
			var d60 scm.JITValueDesc
			_ = d60
			var d61 scm.JITValueDesc
			_ = d61
			var d62 scm.JITValueDesc
			_ = d62
			var d63 scm.JITValueDesc
			_ = d63
			var d66 scm.JITValueDesc
			_ = d66
			var d67 scm.JITValueDesc
			_ = d67
			var d68 scm.JITValueDesc
			_ = d68
			var d136 scm.JITValueDesc
			_ = d136
			var d137 scm.JITValueDesc
			_ = d137
			var d138 scm.JITValueDesc
			_ = d138
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
			var d173 scm.JITValueDesc
			_ = d173
			var d174 scm.JITValueDesc
			_ = d174
			var d175 scm.JITValueDesc
			_ = d175
			var d178 scm.JITValueDesc
			_ = d178
			var d286 scm.JITValueDesc
			_ = d286
			var d287 scm.JITValueDesc
			_ = d287
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
			var d293 scm.JITValueDesc
			_ = d293
			var d295 scm.JITValueDesc
			_ = d295
			var d296 scm.JITValueDesc
			_ = d296
			var d297 scm.JITValueDesc
			_ = d297
			var d298 scm.JITValueDesc
			_ = d298
			var d299 scm.JITValueDesc
			_ = d299
			var d300 scm.JITValueDesc
			_ = d300
			var d301 scm.JITValueDesc
			_ = d301
			var d302 scm.JITValueDesc
			_ = d302
			var d303 scm.JITValueDesc
			_ = d303
			var d304 scm.JITValueDesc
			_ = d304
			var d306 scm.JITValueDesc
			_ = d306
			var d308 scm.JITValueDesc
			_ = d308
			var d309 scm.JITValueDesc
			_ = d309
			var d310 scm.JITValueDesc
			_ = d310
			var d311 scm.JITValueDesc
			_ = d311
			var d312 scm.JITValueDesc
			_ = d312
			var d315 scm.JITValueDesc
			_ = d315
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
			var d454 scm.JITValueDesc
			_ = d454
			var d455 scm.JITValueDesc
			_ = d455
			var d456 scm.JITValueDesc
			_ = d456
			var d457 scm.JITValueDesc
			_ = d457
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
			var d465 scm.JITValueDesc
			_ = d465
			var d466 scm.JITValueDesc
			_ = d466
			var d467 scm.JITValueDesc
			_ = d467
			var d468 scm.JITValueDesc
			_ = d468
			var d469 scm.JITValueDesc
			_ = d469
			var d470 scm.JITValueDesc
			_ = d470
			var d471 scm.JITValueDesc
			_ = d471
			var d472 scm.JITValueDesc
			_ = d472
			var d473 scm.JITValueDesc
			_ = d473
			var d474 scm.JITValueDesc
			_ = d474
			var d475 scm.JITValueDesc
			_ = d475
			var d476 scm.JITValueDesc
			_ = d476
			var d477 scm.JITValueDesc
			_ = d477
			var d478 scm.JITValueDesc
			_ = d478
			var d479 scm.JITValueDesc
			_ = d479
			var d480 scm.JITValueDesc
			_ = d480
			var d481 scm.JITValueDesc
			_ = d481
			var d482 scm.JITValueDesc
			_ = d482
			var d483 scm.JITValueDesc
			_ = d483
			var d484 scm.JITValueDesc
			_ = d484
			var d485 scm.JITValueDesc
			_ = d485
			var d486 scm.JITValueDesc
			_ = d486
			var d487 scm.JITValueDesc
			_ = d487
			var d488 scm.JITValueDesc
			_ = d488
			var d489 scm.JITValueDesc
			_ = d489
			var d490 scm.JITValueDesc
			_ = d490
			var d491 scm.JITValueDesc
			_ = d491
			var d492 scm.JITValueDesc
			_ = d492
			var d493 scm.JITValueDesc
			_ = d493
			var d494 scm.JITValueDesc
			_ = d494
			var d676 scm.JITValueDesc
			_ = d676
			var d677 scm.JITValueDesc
			_ = d677
			var d678 scm.JITValueDesc
			_ = d678
			var d679 scm.JITValueDesc
			_ = d679
			var d680 scm.JITValueDesc
			_ = d680
			var d682 scm.JITValueDesc
			_ = d682
			var d683 scm.JITValueDesc
			_ = d683
			var d684 scm.JITValueDesc
			_ = d684
			var d685 scm.JITValueDesc
			_ = d685
			var d686 scm.JITValueDesc
			_ = d686
			var d687 scm.JITValueDesc
			_ = d687
			var d688 scm.JITValueDesc
			_ = d688
			var d689 scm.JITValueDesc
			_ = d689
			var d691 scm.JITValueDesc
			_ = d691
			var d693 scm.JITValueDesc
			_ = d693
			var d694 scm.JITValueDesc
			_ = d694
			var d695 scm.JITValueDesc
			_ = d695
			var d696 scm.JITValueDesc
			_ = d696
			var d699 scm.JITValueDesc
			_ = d699
			var d896 scm.JITValueDesc
			_ = d896
			var d897 scm.JITValueDesc
			_ = d897
			var d898 scm.JITValueDesc
			_ = d898
			var d899 scm.JITValueDesc
			_ = d899
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
			var d918 scm.JITValueDesc
			_ = d918
			var d919 scm.JITValueDesc
			_ = d919
			var d920 scm.JITValueDesc
			_ = d920
			var d921 scm.JITValueDesc
			_ = d921
			var d922 scm.JITValueDesc
			_ = d922
			var d923 scm.JITValueDesc
			_ = d923
			var d924 scm.JITValueDesc
			_ = d924
			var d925 scm.JITValueDesc
			_ = d925
			var d926 scm.JITValueDesc
			_ = d926
			var d927 scm.JITValueDesc
			_ = d927
			var d928 scm.JITValueDesc
			_ = d928
			var d929 scm.JITValueDesc
			_ = d929
			var d930 scm.JITValueDesc
			_ = d930
			var d931 scm.JITValueDesc
			_ = d931
			var d932 scm.JITValueDesc
			_ = d932
			var d933 scm.JITValueDesc
			_ = d933
			var d934 scm.JITValueDesc
			_ = d934
			var d935 scm.JITValueDesc
			_ = d935
			var d936 scm.JITValueDesc
			_ = d936
			var d937 scm.JITValueDesc
			_ = d937
			var d938 scm.JITValueDesc
			_ = d938
			var d939 scm.JITValueDesc
			_ = d939
			var d940 scm.JITValueDesc
			_ = d940
			var d941 scm.JITValueDesc
			_ = d941
			var d942 scm.JITValueDesc
			_ = d942
			var d943 scm.JITValueDesc
			_ = d943
			var d944 scm.JITValueDesc
			_ = d944
			var d945 scm.JITValueDesc
			_ = d945
			var d946 scm.JITValueDesc
			_ = d946
			var d947 scm.JITValueDesc
			_ = d947
			var d948 scm.JITValueDesc
			_ = d948
			var d949 scm.JITValueDesc
			_ = d949
			var d950 scm.JITValueDesc
			_ = d950
			var d951 scm.JITValueDesc
			_ = d951
			var d952 scm.JITValueDesc
			_ = d952
			var d954 scm.JITValueDesc
			_ = d954
			var d955 scm.JITValueDesc
			_ = d955
			var d956 scm.JITValueDesc
			_ = d956
			var d957 scm.JITValueDesc
			_ = d957
			var d958 scm.JITValueDesc
			_ = d958
			var d959 scm.JITValueDesc
			_ = d959
			var d960 scm.JITValueDesc
			_ = d960
			var d961 scm.JITValueDesc
			_ = d961
			var d962 scm.JITValueDesc
			_ = d962
			var d963 scm.JITValueDesc
			_ = d963
			var d964 scm.JITValueDesc
			_ = d964
			var d965 scm.JITValueDesc
			_ = d965
			var d966 scm.JITValueDesc
			_ = d966
			var d967 scm.JITValueDesc
			_ = d967
			var d968 scm.JITValueDesc
			_ = d968
			var d969 scm.JITValueDesc
			_ = d969
			var d970 scm.JITValueDesc
			_ = d970
			var d971 scm.JITValueDesc
			_ = d971
			var d972 scm.JITValueDesc
			_ = d972
			var d973 scm.JITValueDesc
			_ = d973
			var d974 scm.JITValueDesc
			_ = d974
			var d975 scm.JITValueDesc
			_ = d975
			var d976 scm.JITValueDesc
			_ = d976
			var d977 scm.JITValueDesc
			_ = d977
			var d978 scm.JITValueDesc
			_ = d978
			var d979 scm.JITValueDesc
			_ = d979
			var d980 scm.JITValueDesc
			_ = d980
			var d981 scm.JITValueDesc
			_ = d981
			var d982 scm.JITValueDesc
			_ = d982
			var d983 scm.JITValueDesc
			_ = d983
			var d984 scm.JITValueDesc
			_ = d984
			var d985 scm.JITValueDesc
			_ = d985
			var d986 scm.JITValueDesc
			_ = d986
			var d987 scm.JITValueDesc
			_ = d987
			var d988 scm.JITValueDesc
			_ = d988
			var d989 scm.JITValueDesc
			_ = d989
			var d990 scm.JITValueDesc
			_ = d990
			var d991 scm.JITValueDesc
			_ = d991
			var d992 scm.JITValueDesc
			_ = d992
			var d993 scm.JITValueDesc
			_ = d993
			var d994 scm.JITValueDesc
			_ = d994
			var d995 scm.JITValueDesc
			_ = d995
			var d996 scm.JITValueDesc
			_ = d996
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
				if idxInt.Loc != scm.LocReg { panic("jit: idxInt not in register") }
				ctx.EmitShlRegImm8(idxInt.Reg, 32)
				ctx.EmitShrRegImm8(idxInt.Reg, 32)
				ctx.BindReg(idxInt.Reg, &idxInt)
			}
			idxPinned := idxInt.Loc == scm.LocReg
			idxPinnedReg := idxInt.Reg
			if idxPinned { ctx.ProtectReg(idxPinnedReg) }
			phiBase0 := ctx.AllocStack(int32(144))
			d1 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0)+int32(0)}
			d2 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0)+int32(16)}
			d3 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0)+int32(32)}
			d4 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0)+int32(48)}
			d5 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0)+int32(64)}
			d6 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0)+int32(80)}
			d7 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0)+int32(96)}
			d8 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0)+int32(112)}
			d9 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0)+int32(128)}
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
			ctx.EnsureDesc(&d11)
			if d11.Loc == scm.LocReg {
				ctx.ProtectReg(d11.Reg)
			} else if d11.Loc == scm.LocRegPair {
				ctx.ProtectReg(d11.Reg)
				ctx.ProtectReg(d11.Reg2)
			}
			ctx.EnsureDesc(&d13)
			if d13.Loc == scm.LocReg {
				ctx.ProtectReg(d13.Reg)
			} else if d13.Loc == scm.LocRegPair {
				ctx.ProtectReg(d13.Reg)
				ctx.ProtectReg(d13.Reg2)
			}
			d14 = d11
			if d14.Loc == scm.LocNone { panic("jit: phi source has no location") }
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
			d16 = d13
			if d16.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d16)
			d17 = d16
			if d17.Loc == scm.LocImm {
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: d17.Type, Imm: scm.NewInt(int64(uint64(d17.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d17.Reg, 32)
				ctx.EmitShrRegImm8(d17.Reg, 32)
			}
			ctx.EmitStoreToStack(d17, int32(bbs[1].PhiBase)+int32(32))
			if d11.Loc == scm.LocReg {
				ctx.UnprotectReg(d11.Reg)
			} else if d11.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d11.Reg)
				ctx.UnprotectReg(d11.Reg2)
			}
			if d13.Loc == scm.LocReg {
				ctx.UnprotectReg(d13.Reg)
			} else if d13.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d13.Reg)
				ctx.UnprotectReg(d13.Reg2)
			}
			ps18 := scm.PhiState{General: ps.General}
			ps18.OverlayValues = make([]scm.JITValueDesc, 18)
			ps18.OverlayValues[1] = d1
			ps18.OverlayValues[2] = d2
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
			d19 = d11
			ps18.PhiValues[0] = d19
			d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
			ps18.PhiValues[1] = d20
			d21 = d13
			ps18.PhiValues[2] = d21
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
					d22 := ps.PhiValues[0]
					ctx.EnsureDesc(&d22)
					ctx.EmitStoreToStack(d22, int32(bbs[1].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
					d23 := ps.PhiValues[1]
					ctx.EnsureDesc(&d23)
					ctx.EmitStoreToStack(d23, int32(bbs[1].PhiBase)+int32(16))
				}
				if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
					d24 := ps.PhiValues[2]
					ctx.EnsureDesc(&d24)
					ctx.EmitStoreToStack(d24, int32(bbs[1].PhiBase)+int32(32))
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
			ctx.EnsureDesc(&d1)
			d25 = d1
			_ = d25
			r6 := d1.Loc == scm.LocReg
			r7 := d1.Reg
			if r6 { ctx.ProtectReg(r7) }
			phiBase26 := ctx.AllocStack(int32(16))
			d27 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase26)+int32(144)}
			lbl15 := ctx.ReserveLabel()
			bbpos_1_0 := int32(-1)
			_ = bbpos_1_0
			bbpos_1_1 := int32(-1)
			_ = bbpos_1_1
			bbpos_1_2 := int32(-1)
			_ = bbpos_1_2
			bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d27 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(144)}
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d25)
			var d28 scm.JITValueDesc
			if d25.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d25.Imm.Int()))))}
			} else {
				r8 := ctx.AllocReg()
				ctx.EmitMovRegReg(r8, d25.Reg)
				ctx.EmitShlRegImm8(r8, 32)
				ctx.EmitShrRegImm8(r8, 32)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
				ctx.BindReg(r8, &d28)
			}
			var d29 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				r9 := ctx.AllocReg()
				ctx.EmitMovRegMem8(r9, fieldAddr)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r9}
				ctx.BindReg(r9, &d29)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r10 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r10, thisptr.Reg, off)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r10}
				ctx.BindReg(r10, &d29)
			}
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d29)
			var d30 scm.JITValueDesc
			if d29.Loc == scm.LocImm {
				d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d29.Imm.Int()))))}
			} else {
				r11 := ctx.AllocReg()
				ctx.EmitMovRegReg(r11, d29.Reg)
				ctx.EmitShlRegImm8(r11, 56)
				ctx.EmitShrRegImm8(r11, 56)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
				ctx.BindReg(r11, &d30)
			}
			ctx.EnsureDesc(&d28)
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d28)
			ctx.ProtectReg(d28.Reg)
			ctx.EnsureDesc(&d30)
			ctx.UnprotectReg(d28.Reg)
			var d31 scm.JITValueDesc
			if d28.Loc == scm.LocImm && d30.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d28.Imm.Int() * d30.Imm.Int())}
			} else if d28.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d30.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d28.Imm.Int()))
				ctx.EmitImulInt64(scratch, d30.Reg)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d31)
			} else if d30.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d28.Reg)
				ctx.EmitMovRegReg(scratch, d28.Reg)
				if d30.Imm.Int() >= -2147483648 && d30.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d30.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d30.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d31)
			} else {
				r12 := ctx.AllocRegExcept(d28.Reg, d30.Reg)
				ctx.EmitMovRegReg(r12, d28.Reg)
				ctx.EmitImulInt64(r12, d30.Reg)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
				ctx.BindReg(r12, &d31)
			}
			if d31.Loc == scm.LocReg && d28.Loc == scm.LocReg && d31.Reg == d28.Reg {
				ctx.TransferReg(d28.Reg)
				d28.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d28)
			ctx.FreeDesc(&d30)
			var d32 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				r13 := ctx.AllocReg()
				r14 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r13, fieldAddr)
				ctx.EmitMovRegMem64(r14, fieldAddr+8)
				d32 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r13, Reg2: r14}
				ctx.BindReg(r13, &d32)
				ctx.BindReg(r14, &d32)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				r15 := ctx.AllocReg()
				r16 := ctx.AllocReg()
				ctx.EmitMovRegMem(r15, thisptr.Reg, off)
				ctx.EmitMovRegMem(r16, thisptr.Reg, off+8)
				d32 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r15, Reg2: r16}
				ctx.BindReg(r15, &d32)
				ctx.BindReg(r16, &d32)
			}
			ctx.EnsureDesc(&d31)
			var d33 scm.JITValueDesc
			if d31.Loc == scm.LocImm {
				d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d31.Imm.Int() / 64)}
			} else {
				r17 := ctx.AllocRegExcept(d31.Reg)
				ctx.EmitMovRegReg(r17, d31.Reg)
				ctx.EmitShrRegImm8(r17, 6)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
				ctx.BindReg(r17, &d33)
			}
			if d33.Loc == scm.LocReg && d31.Loc == scm.LocReg && d33.Reg == d31.Reg {
				ctx.TransferReg(d31.Reg)
				d31.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d33)
			r18 := ctx.AllocReg()
			ctx.EnsureDesc(&d33)
			ctx.EnsureDesc(&d32)
			if d33.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r18, uint64(d33.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r18, d33.Reg)
				ctx.EmitShlRegImm8(r18, 3)
			}
			if d32.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d32.Imm.Int()))
				ctx.EmitAddInt64(r18, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r18, d32.Reg)
			}
			r19 := ctx.AllocRegExcept(r18)
			ctx.EmitMovRegMem(r19, r18, 0)
			ctx.FreeReg(r18)
			d34 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r19}
			ctx.BindReg(r19, &d34)
			ctx.FreeDesc(&d33)
			ctx.EnsureDesc(&d31)
			var d35 scm.JITValueDesc
			if d31.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d31.Imm.Int() % 64)}
			} else {
				r20 := ctx.AllocRegExcept(d31.Reg)
				ctx.EmitMovRegReg(r20, d31.Reg)
				ctx.EmitAndRegImm32(r20, 63)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
				ctx.BindReg(r20, &d35)
			}
			if d35.Loc == scm.LocReg && d31.Loc == scm.LocReg && d35.Reg == d31.Reg {
				ctx.TransferReg(d31.Reg)
				d31.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d34)
			ctx.EnsureDesc(&d35)
			var d36 scm.JITValueDesc
			if d34.Loc == scm.LocImm && d35.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d34.Imm.Int()) << uint64(d35.Imm.Int())))}
			} else if d35.Loc == scm.LocImm {
				r21 := ctx.AllocRegExcept(d34.Reg)
				ctx.EmitMovRegReg(r21, d34.Reg)
				ctx.EmitShlRegImm8(r21, uint8(d35.Imm.Int()))
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d36)
			} else {
				{
					shiftSrc := d34.Reg
					r22 := ctx.AllocRegExcept(d34.Reg)
					ctx.EmitMovRegReg(r22, d34.Reg)
					shiftSrc = r22
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d35.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d35.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d35.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d36)
				}
			}
			if d36.Loc == scm.LocReg && d34.Loc == scm.LocReg && d36.Reg == d34.Reg {
				ctx.TransferReg(d34.Reg)
				d34.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d34)
			ctx.FreeDesc(&d35)
			ctx.EnsureDesc(&d31)
			var d37 scm.JITValueDesc
			if d31.Loc == scm.LocImm {
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d31.Imm.Int() % 64)}
			} else {
				r23 := ctx.AllocRegExcept(d31.Reg)
				ctx.EmitMovRegReg(r23, d31.Reg)
				ctx.EmitAndRegImm32(r23, 63)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
				ctx.BindReg(r23, &d37)
			}
			if d37.Loc == scm.LocReg && d31.Loc == scm.LocReg && d37.Reg == d31.Reg {
				ctx.TransferReg(d31.Reg)
				d31.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d29)
			var d38 scm.JITValueDesc
			if d29.Loc == scm.LocImm {
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d29.Imm.Int()))))}
			} else {
				r24 := ctx.AllocReg()
				ctx.EmitMovRegReg(r24, d29.Reg)
				ctx.EmitShlRegImm8(r24, 56)
				ctx.EmitShrRegImm8(r24, 56)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
				ctx.BindReg(r24, &d38)
			}
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d37)
			ctx.ProtectReg(d37.Reg)
			ctx.EnsureDesc(&d38)
			ctx.UnprotectReg(d37.Reg)
			var d39 scm.JITValueDesc
			if d37.Loc == scm.LocImm && d38.Loc == scm.LocImm {
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d37.Imm.Int() + d38.Imm.Int())}
			} else if d38.Loc == scm.LocImm && d38.Imm.Int() == 0 {
				r25 := ctx.AllocRegExcept(d37.Reg)
				ctx.EmitMovRegReg(r25, d37.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
				ctx.BindReg(r25, &d39)
			} else if d37.Loc == scm.LocImm && d37.Imm.Int() == 0 {
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d38.Reg}
				ctx.BindReg(d38.Reg, &d39)
			} else if d37.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d38.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d37.Imm.Int()))
				ctx.EmitAddInt64(scratch, d38.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d39)
			} else if d38.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d37.Reg)
				ctx.EmitMovRegReg(scratch, d37.Reg)
				if d38.Imm.Int() >= -2147483648 && d38.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d38.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d38.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d39)
			} else {
				r26 := ctx.AllocRegExcept(d37.Reg, d38.Reg)
				ctx.EmitMovRegReg(r26, d37.Reg)
				ctx.EmitAddInt64(r26, d38.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
				ctx.BindReg(r26, &d39)
			}
			if d39.Loc == scm.LocReg && d37.Loc == scm.LocReg && d39.Reg == d37.Reg {
				ctx.TransferReg(d37.Reg)
				d37.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d37)
			ctx.FreeDesc(&d38)
			ctx.EnsureDesc(&d39)
			var d40 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d39.Imm.Int()) > uint64(64))}
			} else {
				r27 := ctx.AllocRegExcept(d39.Reg)
				ctx.EmitCmpRegImm32(d39.Reg, 64)
				ctx.EmitSetcc(r27, scm.CcA)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r27}
				ctx.BindReg(r27, &d40)
			}
			ctx.FreeDesc(&d39)
			d41 = d40
			ctx.EnsureDesc(&d41)
			if d41.Loc != scm.LocImm && d41.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl16 := ctx.ReserveLabel()
			lbl17 := ctx.ReserveLabel()
			lbl18 := ctx.ReserveLabel()
			lbl19 := ctx.ReserveLabel()
			if d41.Loc == scm.LocImm {
				if d41.Imm.Bool() {
					ctx.MarkLabel(lbl18)
					ctx.EmitJmp(lbl16)
				} else {
					ctx.MarkLabel(lbl19)
			ctx.EnsureDesc(&d36)
			if d36.Loc == scm.LocReg {
				ctx.ProtectReg(d36.Reg)
			} else if d36.Loc == scm.LocRegPair {
				ctx.ProtectReg(d36.Reg)
				ctx.ProtectReg(d36.Reg2)
			}
			d42 = d36
			if d42.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d42)
			ctx.EmitStoreToStack(d42, int32(bbs[2].PhiBase)+int32(0))
			if d36.Loc == scm.LocReg {
				ctx.UnprotectReg(d36.Reg)
			} else if d36.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d36.Reg)
				ctx.UnprotectReg(d36.Reg2)
			}
					ctx.EmitJmp(lbl17)
				}
			} else {
				ctx.EmitCmpRegImm32(d41.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl18)
				ctx.EmitJmp(lbl19)
				ctx.MarkLabel(lbl18)
				ctx.EmitJmp(lbl16)
				ctx.MarkLabel(lbl19)
			ctx.EnsureDesc(&d36)
			if d36.Loc == scm.LocReg {
				ctx.ProtectReg(d36.Reg)
			} else if d36.Loc == scm.LocRegPair {
				ctx.ProtectReg(d36.Reg)
				ctx.ProtectReg(d36.Reg2)
			}
			d43 = d36
			if d43.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d43)
			ctx.EmitStoreToStack(d43, int32(bbs[2].PhiBase)+int32(0))
			if d36.Loc == scm.LocReg {
				ctx.UnprotectReg(d36.Reg)
			} else if d36.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d36.Reg)
				ctx.UnprotectReg(d36.Reg2)
			}
				ctx.EmitJmp(lbl17)
			}
			ctx.FreeDesc(&d40)
			bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl17)
			ctx.ResolveFixups()
			d27 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(144)}
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d29)
			var d44 scm.JITValueDesc
			if d29.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d29.Imm.Int()))))}
			} else {
				r28 := ctx.AllocReg()
				ctx.EmitMovRegReg(r28, d29.Reg)
				ctx.EmitShlRegImm8(r28, 56)
				ctx.EmitShrRegImm8(r28, 56)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d44)
			}
			d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d44)
			ctx.EnsureDesc(&d45)
			ctx.ProtectReg(d45.Reg)
			ctx.EnsureDesc(&d44)
			ctx.UnprotectReg(d45.Reg)
			var d46 scm.JITValueDesc
			if d45.Loc == scm.LocImm && d44.Loc == scm.LocImm {
				d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d45.Imm.Int() - d44.Imm.Int())}
			} else if d44.Loc == scm.LocImm && d44.Imm.Int() == 0 {
				r29 := ctx.AllocRegExcept(d45.Reg)
				ctx.EmitMovRegReg(r29, d45.Reg)
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d46)
			} else if d45.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d44.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d45.Imm.Int()))
				ctx.EmitSubInt64(scratch, d44.Reg)
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d46)
			} else if d44.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d45.Reg)
				ctx.EmitMovRegReg(scratch, d45.Reg)
				if d44.Imm.Int() >= -2147483648 && d44.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d44.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d44.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d46)
			} else {
				r30 := ctx.AllocRegExcept(d45.Reg, d44.Reg)
				ctx.EmitMovRegReg(r30, d45.Reg)
				ctx.EmitSubInt64(r30, d44.Reg)
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
				ctx.BindReg(r30, &d46)
			}
			if d46.Loc == scm.LocReg && d45.Loc == scm.LocReg && d46.Reg == d45.Reg {
				ctx.TransferReg(d45.Reg)
				d45.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d44)
			ctx.EnsureDesc(&d27)
			ctx.EnsureDesc(&d46)
			var d47 scm.JITValueDesc
			if d27.Loc == scm.LocImm && d46.Loc == scm.LocImm {
				d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d27.Imm.Int()) >> uint64(d46.Imm.Int())))}
			} else if d46.Loc == scm.LocImm {
				r31 := ctx.AllocRegExcept(d27.Reg)
				ctx.EmitMovRegReg(r31, d27.Reg)
				ctx.EmitShrRegImm8(r31, uint8(d46.Imm.Int()))
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
				ctx.BindReg(r31, &d47)
			} else {
				{
					shiftSrc := d27.Reg
					r32 := ctx.AllocRegExcept(d27.Reg)
					ctx.EmitMovRegReg(r32, d27.Reg)
					shiftSrc = r32
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d46.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d46.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d46.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d47)
				}
			}
			if d47.Loc == scm.LocReg && d27.Loc == scm.LocReg && d47.Reg == d27.Reg {
				ctx.TransferReg(d27.Reg)
				d27.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d27)
			ctx.FreeDesc(&d46)
			r33 := ctx.AllocReg()
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d47)
			if d47.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r33, d47)
			}
			ctx.EmitJmp(lbl15)
			bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl16)
			ctx.ResolveFixups()
			d27 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(144)}
			ctx.EnsureDesc(&d31)
			var d48 scm.JITValueDesc
			if d31.Loc == scm.LocImm {
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d31.Imm.Int() / 64)}
			} else {
				r34 := ctx.AllocRegExcept(d31.Reg)
				ctx.EmitMovRegReg(r34, d31.Reg)
				ctx.EmitShrRegImm8(r34, 6)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
				ctx.BindReg(r34, &d48)
			}
			if d48.Loc == scm.LocReg && d31.Loc == scm.LocReg && d48.Reg == d31.Reg {
				ctx.TransferReg(d31.Reg)
				d31.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d48)
			var d49 scm.JITValueDesc
			if d48.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d48.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d48.Reg)
				ctx.EmitMovRegReg(scratch, d48.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d49)
			}
			if d49.Loc == scm.LocReg && d48.Loc == scm.LocReg && d49.Reg == d48.Reg {
				ctx.TransferReg(d48.Reg)
				d48.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d48)
			ctx.EnsureDesc(&d49)
			r35 := ctx.AllocReg()
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d32)
			if d49.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r35, uint64(d49.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r35, d49.Reg)
				ctx.EmitShlRegImm8(r35, 3)
			}
			if d32.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d32.Imm.Int()))
				ctx.EmitAddInt64(r35, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r35, d32.Reg)
			}
			r36 := ctx.AllocRegExcept(r35)
			ctx.EmitMovRegMem(r36, r35, 0)
			ctx.FreeReg(r35)
			d50 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r36}
			ctx.BindReg(r36, &d50)
			ctx.FreeDesc(&d49)
			ctx.EnsureDesc(&d31)
			var d51 scm.JITValueDesc
			if d31.Loc == scm.LocImm {
				d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d31.Imm.Int() % 64)}
			} else {
				r37 := ctx.AllocRegExcept(d31.Reg)
				ctx.EmitMovRegReg(r37, d31.Reg)
				ctx.EmitAndRegImm32(r37, 63)
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d51)
			}
			if d51.Loc == scm.LocReg && d31.Loc == scm.LocReg && d51.Reg == d31.Reg {
				ctx.TransferReg(d31.Reg)
				d31.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d31)
			d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d51)
			ctx.EnsureDesc(&d52)
			ctx.ProtectReg(d52.Reg)
			ctx.EnsureDesc(&d51)
			ctx.UnprotectReg(d52.Reg)
			var d53 scm.JITValueDesc
			if d52.Loc == scm.LocImm && d51.Loc == scm.LocImm {
				d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() - d51.Imm.Int())}
			} else if d51.Loc == scm.LocImm && d51.Imm.Int() == 0 {
				r38 := ctx.AllocRegExcept(d52.Reg)
				ctx.EmitMovRegReg(r38, d52.Reg)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
				ctx.BindReg(r38, &d53)
			} else if d52.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d51.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d52.Imm.Int()))
				ctx.EmitSubInt64(scratch, d51.Reg)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d53)
			} else if d51.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d52.Reg)
				ctx.EmitMovRegReg(scratch, d52.Reg)
				if d51.Imm.Int() >= -2147483648 && d51.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d51.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d51.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d53)
			} else {
				r39 := ctx.AllocRegExcept(d52.Reg, d51.Reg)
				ctx.EmitMovRegReg(r39, d52.Reg)
				ctx.EmitSubInt64(r39, d51.Reg)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d53)
			}
			if d53.Loc == scm.LocReg && d52.Loc == scm.LocReg && d53.Reg == d52.Reg {
				ctx.TransferReg(d52.Reg)
				d52.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d51)
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d53)
			var d54 scm.JITValueDesc
			if d50.Loc == scm.LocImm && d53.Loc == scm.LocImm {
				d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d50.Imm.Int()) >> uint64(d53.Imm.Int())))}
			} else if d53.Loc == scm.LocImm {
				r40 := ctx.AllocRegExcept(d50.Reg)
				ctx.EmitMovRegReg(r40, d50.Reg)
				ctx.EmitShrRegImm8(r40, uint8(d53.Imm.Int()))
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
				ctx.BindReg(r40, &d54)
			} else {
				{
					shiftSrc := d50.Reg
					r41 := ctx.AllocRegExcept(d50.Reg)
					ctx.EmitMovRegReg(r41, d50.Reg)
					shiftSrc = r41
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d53.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d53.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d53.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d54)
				}
			}
			if d54.Loc == scm.LocReg && d50.Loc == scm.LocReg && d54.Reg == d50.Reg {
				ctx.TransferReg(d50.Reg)
				d50.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d50)
			ctx.FreeDesc(&d53)
			ctx.EnsureDesc(&d36)
			ctx.EnsureDesc(&d54)
			var d55 scm.JITValueDesc
			if d36.Loc == scm.LocImm && d54.Loc == scm.LocImm {
				d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d36.Imm.Int() | d54.Imm.Int())}
			} else if d36.Loc == scm.LocImm && d36.Imm.Int() == 0 {
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d54.Reg}
				ctx.BindReg(d54.Reg, &d55)
			} else if d54.Loc == scm.LocImm && d54.Imm.Int() == 0 {
				r42 := ctx.AllocRegExcept(d36.Reg)
				ctx.EmitMovRegReg(r42, d36.Reg)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d55)
			} else if d36.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d54.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d36.Imm.Int()))
				ctx.EmitOrInt64(scratch, d54.Reg)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d55)
			} else if d54.Loc == scm.LocImm {
				r43 := ctx.AllocRegExcept(d36.Reg)
				ctx.EmitMovRegReg(r43, d36.Reg)
				if d54.Imm.Int() >= -2147483648 && d54.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r43, int32(d54.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d54.Imm.Int()))
					ctx.EmitOrInt64(r43, scm.RegR11)
				}
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
				ctx.BindReg(r43, &d55)
			} else {
				r44 := ctx.AllocRegExcept(d36.Reg, d54.Reg)
				ctx.EmitMovRegReg(r44, d36.Reg)
				ctx.EmitOrInt64(r44, d54.Reg)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d55)
			}
			if d55.Loc == scm.LocReg && d36.Loc == scm.LocReg && d55.Reg == d36.Reg {
				ctx.TransferReg(d36.Reg)
				d36.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d54)
			ctx.EnsureDesc(&d55)
			if d55.Loc == scm.LocReg {
				ctx.ProtectReg(d55.Reg)
			} else if d55.Loc == scm.LocRegPair {
				ctx.ProtectReg(d55.Reg)
				ctx.ProtectReg(d55.Reg2)
			}
			d56 = d55
			if d56.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d56)
			ctx.EmitStoreToStack(d56, int32(bbs[2].PhiBase)+int32(0))
			if d55.Loc == scm.LocReg {
				ctx.UnprotectReg(d55.Reg)
			} else if d55.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d55.Reg)
				ctx.UnprotectReg(d55.Reg2)
			}
			ctx.EmitJmp(lbl17)
			ctx.MarkLabel(lbl15)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r33}
			ctx.BindReg(r33, &d57)
			ctx.BindReg(r33, &d57)
			if r6 { ctx.UnprotectReg(r7) }
			ctx.EnsureDesc(&d57)
			ctx.EnsureDesc(&d57)
			var d58 scm.JITValueDesc
			if d57.Loc == scm.LocImm {
				d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d57.Imm.Int()))))}
			} else {
				r45 := ctx.AllocReg()
				ctx.EmitMovRegReg(r45, d57.Reg)
				d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
				ctx.BindReg(r45, &d58)
			}
			ctx.FreeDesc(&d57)
			var d59 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				r46 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r46, fieldAddr)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r46}
				ctx.BindReg(r46, &d59)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r47 := ctx.AllocReg()
				ctx.EmitMovRegMem(r47, thisptr.Reg, off)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r47}
				ctx.BindReg(r47, &d59)
			}
			ctx.EnsureDesc(&d58)
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d58)
			ctx.ProtectReg(d58.Reg)
			ctx.EnsureDesc(&d59)
			ctx.UnprotectReg(d58.Reg)
			var d60 scm.JITValueDesc
			if d58.Loc == scm.LocImm && d59.Loc == scm.LocImm {
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d58.Imm.Int() + d59.Imm.Int())}
			} else if d59.Loc == scm.LocImm && d59.Imm.Int() == 0 {
				r48 := ctx.AllocRegExcept(d58.Reg)
				ctx.EmitMovRegReg(r48, d58.Reg)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
				ctx.BindReg(r48, &d60)
			} else if d58.Loc == scm.LocImm && d58.Imm.Int() == 0 {
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d59.Reg}
				ctx.BindReg(d59.Reg, &d60)
			} else if d58.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d59.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d58.Imm.Int()))
				ctx.EmitAddInt64(scratch, d59.Reg)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d60)
			} else if d59.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d58.Reg)
				ctx.EmitMovRegReg(scratch, d58.Reg)
				if d59.Imm.Int() >= -2147483648 && d59.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d59.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d59.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d60)
			} else {
				r49 := ctx.AllocRegExcept(d58.Reg, d59.Reg)
				ctx.EmitMovRegReg(r49, d58.Reg)
				ctx.EmitAddInt64(r49, d59.Reg)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
				ctx.BindReg(r49, &d60)
			}
			if d60.Loc == scm.LocReg && d58.Loc == scm.LocReg && d60.Reg == d58.Reg {
				ctx.TransferReg(d58.Reg)
				d58.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d58)
			ctx.EnsureDesc(&d60)
			ctx.EnsureDesc(&d60)
			var d61 scm.JITValueDesc
			if d60.Loc == scm.LocImm {
				d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d60.Imm.Int()))))}
			} else {
				r50 := ctx.AllocReg()
				ctx.EmitMovRegReg(r50, d60.Reg)
				ctx.EmitShlRegImm8(r50, 32)
				ctx.EmitShrRegImm8(r50, 32)
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
				ctx.BindReg(r50, &d61)
			}
			ctx.FreeDesc(&d60)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d61)
			var d62 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d61.Loc == scm.LocImm {
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d61.Imm.Int()))}
			} else if d61.Loc == scm.LocImm {
				r51 := ctx.AllocRegExcept(idxInt.Reg)
				if d61.Imm.Int() >= -2147483648 && d61.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(idxInt.Reg, int32(d61.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d61.Imm.Int()))
					ctx.EmitCmpInt64(idxInt.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r51, scm.CcB)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r51}
				ctx.BindReg(r51, &d62)
			} else if idxInt.Loc == scm.LocImm {
				r52 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d61.Reg)
				ctx.EmitSetcc(r52, scm.CcB)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r52}
				ctx.BindReg(r52, &d62)
			} else {
				r53 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.EmitCmpInt64(idxInt.Reg, d61.Reg)
				ctx.EmitSetcc(r53, scm.CcB)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r53}
				ctx.BindReg(r53, &d62)
			}
			ctx.FreeDesc(&d61)
			d63 = d62
			ctx.EnsureDesc(&d63)
			if d63.Loc != scm.LocImm && d63.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d63.Loc == scm.LocImm {
				if d63.Imm.Bool() {
			ps64 := scm.PhiState{General: ps.General}
			ps64.OverlayValues = make([]scm.JITValueDesc, 64)
			ps64.OverlayValues[1] = d1
			ps64.OverlayValues[2] = d2
			ps64.OverlayValues[3] = d3
			ps64.OverlayValues[4] = d4
			ps64.OverlayValues[5] = d5
			ps64.OverlayValues[6] = d6
			ps64.OverlayValues[7] = d7
			ps64.OverlayValues[8] = d8
			ps64.OverlayValues[9] = d9
			ps64.OverlayValues[10] = d10
			ps64.OverlayValues[11] = d11
			ps64.OverlayValues[12] = d12
			ps64.OverlayValues[13] = d13
			ps64.OverlayValues[14] = d14
			ps64.OverlayValues[15] = d15
			ps64.OverlayValues[16] = d16
			ps64.OverlayValues[17] = d17
			ps64.OverlayValues[19] = d19
			ps64.OverlayValues[20] = d20
			ps64.OverlayValues[21] = d21
			ps64.OverlayValues[22] = d22
			ps64.OverlayValues[23] = d23
			ps64.OverlayValues[24] = d24
			ps64.OverlayValues[25] = d25
			ps64.OverlayValues[27] = d27
			ps64.OverlayValues[28] = d28
			ps64.OverlayValues[29] = d29
			ps64.OverlayValues[30] = d30
			ps64.OverlayValues[31] = d31
			ps64.OverlayValues[32] = d32
			ps64.OverlayValues[33] = d33
			ps64.OverlayValues[34] = d34
			ps64.OverlayValues[35] = d35
			ps64.OverlayValues[36] = d36
			ps64.OverlayValues[37] = d37
			ps64.OverlayValues[38] = d38
			ps64.OverlayValues[39] = d39
			ps64.OverlayValues[40] = d40
			ps64.OverlayValues[41] = d41
			ps64.OverlayValues[42] = d42
			ps64.OverlayValues[43] = d43
			ps64.OverlayValues[44] = d44
			ps64.OverlayValues[45] = d45
			ps64.OverlayValues[46] = d46
			ps64.OverlayValues[47] = d47
			ps64.OverlayValues[48] = d48
			ps64.OverlayValues[49] = d49
			ps64.OverlayValues[50] = d50
			ps64.OverlayValues[51] = d51
			ps64.OverlayValues[52] = d52
			ps64.OverlayValues[53] = d53
			ps64.OverlayValues[54] = d54
			ps64.OverlayValues[55] = d55
			ps64.OverlayValues[56] = d56
			ps64.OverlayValues[57] = d57
			ps64.OverlayValues[58] = d58
			ps64.OverlayValues[59] = d59
			ps64.OverlayValues[60] = d60
			ps64.OverlayValues[61] = d61
			ps64.OverlayValues[62] = d62
			ps64.OverlayValues[63] = d63
					return bbs[3].RenderPS(ps64)
				}
			ps65 := scm.PhiState{General: ps.General}
			ps65.OverlayValues = make([]scm.JITValueDesc, 64)
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
			ps65.OverlayValues[16] = d16
			ps65.OverlayValues[17] = d17
			ps65.OverlayValues[19] = d19
			ps65.OverlayValues[20] = d20
			ps65.OverlayValues[21] = d21
			ps65.OverlayValues[22] = d22
			ps65.OverlayValues[23] = d23
			ps65.OverlayValues[24] = d24
			ps65.OverlayValues[25] = d25
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
			ps65.OverlayValues[60] = d60
			ps65.OverlayValues[61] = d61
			ps65.OverlayValues[62] = d62
			ps65.OverlayValues[63] = d63
				return bbs[5].RenderPS(ps65)
			}
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
					d66 := ps.PhiValues[0]
					ctx.EnsureDesc(&d66)
					ctx.EmitStoreToStack(d66, int32(bbs[1].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
					d67 := ps.PhiValues[1]
					ctx.EnsureDesc(&d67)
					ctx.EmitStoreToStack(d67, int32(bbs[1].PhiBase)+int32(16))
				}
				if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
					d68 := ps.PhiValues[2]
					ctx.EnsureDesc(&d68)
					ctx.EmitStoreToStack(d68, int32(bbs[1].PhiBase)+int32(32))
				}
				ps.General = true
				return bbs[1].RenderPS(ps)
			}
			lbl20 := ctx.ReserveLabel()
			lbl21 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d63.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl20)
			ctx.EmitJmp(lbl21)
			ctx.MarkLabel(lbl20)
			ctx.EmitJmp(lbl4)
			ctx.MarkLabel(lbl21)
			ctx.EmitJmp(lbl6)
			ps69 := scm.PhiState{General: true}
			ps69.OverlayValues = make([]scm.JITValueDesc, 69)
			ps69.OverlayValues[1] = d1
			ps69.OverlayValues[2] = d2
			ps69.OverlayValues[3] = d3
			ps69.OverlayValues[4] = d4
			ps69.OverlayValues[5] = d5
			ps69.OverlayValues[6] = d6
			ps69.OverlayValues[7] = d7
			ps69.OverlayValues[8] = d8
			ps69.OverlayValues[9] = d9
			ps69.OverlayValues[10] = d10
			ps69.OverlayValues[11] = d11
			ps69.OverlayValues[12] = d12
			ps69.OverlayValues[13] = d13
			ps69.OverlayValues[14] = d14
			ps69.OverlayValues[15] = d15
			ps69.OverlayValues[16] = d16
			ps69.OverlayValues[17] = d17
			ps69.OverlayValues[19] = d19
			ps69.OverlayValues[20] = d20
			ps69.OverlayValues[21] = d21
			ps69.OverlayValues[22] = d22
			ps69.OverlayValues[23] = d23
			ps69.OverlayValues[24] = d24
			ps69.OverlayValues[25] = d25
			ps69.OverlayValues[27] = d27
			ps69.OverlayValues[28] = d28
			ps69.OverlayValues[29] = d29
			ps69.OverlayValues[30] = d30
			ps69.OverlayValues[31] = d31
			ps69.OverlayValues[32] = d32
			ps69.OverlayValues[33] = d33
			ps69.OverlayValues[34] = d34
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
			ps69.OverlayValues[66] = d66
			ps69.OverlayValues[67] = d67
			ps69.OverlayValues[68] = d68
			ps70 := scm.PhiState{General: true}
			ps70.OverlayValues = make([]scm.JITValueDesc, 69)
			ps70.OverlayValues[1] = d1
			ps70.OverlayValues[2] = d2
			ps70.OverlayValues[3] = d3
			ps70.OverlayValues[4] = d4
			ps70.OverlayValues[5] = d5
			ps70.OverlayValues[6] = d6
			ps70.OverlayValues[7] = d7
			ps70.OverlayValues[8] = d8
			ps70.OverlayValues[9] = d9
			ps70.OverlayValues[10] = d10
			ps70.OverlayValues[11] = d11
			ps70.OverlayValues[12] = d12
			ps70.OverlayValues[13] = d13
			ps70.OverlayValues[14] = d14
			ps70.OverlayValues[15] = d15
			ps70.OverlayValues[16] = d16
			ps70.OverlayValues[17] = d17
			ps70.OverlayValues[19] = d19
			ps70.OverlayValues[20] = d20
			ps70.OverlayValues[21] = d21
			ps70.OverlayValues[22] = d22
			ps70.OverlayValues[23] = d23
			ps70.OverlayValues[24] = d24
			ps70.OverlayValues[25] = d25
			ps70.OverlayValues[27] = d27
			ps70.OverlayValues[28] = d28
			ps70.OverlayValues[29] = d29
			ps70.OverlayValues[30] = d30
			ps70.OverlayValues[31] = d31
			ps70.OverlayValues[32] = d32
			ps70.OverlayValues[33] = d33
			ps70.OverlayValues[34] = d34
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
			ps70.OverlayValues[66] = d66
			ps70.OverlayValues[67] = d67
			ps70.OverlayValues[68] = d68
			snap71 := d1
			snap72 := d2
			snap73 := d3
			snap74 := d4
			snap75 := d5
			snap76 := d6
			snap77 := d7
			snap78 := d8
			snap79 := d9
			snap80 := d10
			snap81 := d11
			snap82 := d12
			snap83 := d13
			snap84 := d14
			snap85 := d15
			snap86 := d16
			snap87 := d17
			snap88 := d19
			snap89 := d20
			snap90 := d21
			snap91 := d22
			snap92 := d23
			snap93 := d24
			snap94 := d25
			snap95 := d27
			snap96 := d28
			snap97 := d29
			snap98 := d30
			snap99 := d31
			snap100 := d32
			snap101 := d33
			snap102 := d34
			snap103 := d35
			snap104 := d36
			snap105 := d37
			snap106 := d38
			snap107 := d39
			snap108 := d40
			snap109 := d41
			snap110 := d42
			snap111 := d43
			snap112 := d44
			snap113 := d45
			snap114 := d46
			snap115 := d47
			snap116 := d48
			snap117 := d49
			snap118 := d50
			snap119 := d51
			snap120 := d52
			snap121 := d53
			snap122 := d54
			snap123 := d55
			snap124 := d56
			snap125 := d57
			snap126 := d58
			snap127 := d59
			snap128 := d60
			snap129 := d61
			snap130 := d62
			snap131 := d63
			snap132 := d66
			snap133 := d67
			snap134 := d68
			alloc135 := ctx.SnapshotAllocState()
			if !bbs[5].Rendered {
				bbs[5].RenderPS(ps70)
			}
			ctx.RestoreAllocState(alloc135)
			d1 = snap71
			d2 = snap72
			d3 = snap73
			d4 = snap74
			d5 = snap75
			d6 = snap76
			d7 = snap77
			d8 = snap78
			d9 = snap79
			d10 = snap80
			d11 = snap81
			d12 = snap82
			d13 = snap83
			d14 = snap84
			d15 = snap85
			d16 = snap86
			d17 = snap87
			d19 = snap88
			d20 = snap89
			d21 = snap90
			d22 = snap91
			d23 = snap92
			d24 = snap93
			d25 = snap94
			d27 = snap95
			d28 = snap96
			d29 = snap97
			d30 = snap98
			d31 = snap99
			d32 = snap100
			d33 = snap101
			d34 = snap102
			d35 = snap103
			d36 = snap104
			d37 = snap105
			d38 = snap106
			d39 = snap107
			d40 = snap108
			d41 = snap109
			d42 = snap110
			d43 = snap111
			d44 = snap112
			d45 = snap113
			d46 = snap114
			d47 = snap115
			d48 = snap116
			d49 = snap117
			d50 = snap118
			d51 = snap119
			d52 = snap120
			d53 = snap121
			d54 = snap122
			d55 = snap123
			d56 = snap124
			d57 = snap125
			d58 = snap126
			d59 = snap127
			d60 = snap128
			d61 = snap129
			d62 = snap130
			d63 = snap131
			d66 = snap132
			d67 = snap133
			d68 = snap134
			if !bbs[3].Rendered {
				return bbs[3].RenderPS(ps69)
			}
			return result
			ctx.FreeDesc(&d62)
			return result
			}
			bbs[2].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
					d136 := ps.PhiValues[0]
					ctx.EnsureDesc(&d136)
					ctx.EmitStoreToStack(d136, int32(bbs[2].PhiBase)+int32(0))
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
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
			}
			if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != scm.LocNone {
				d136 = ps.OverlayValues[136]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d4 = ps.PhiValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d4)
			var d137 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(d4.Imm.Int()))))}
			} else {
				r54 := ctx.AllocReg()
				ctx.EmitMovRegReg(r54, d4.Reg)
				ctx.EmitShlRegImm8(r54, 32)
				ctx.EmitShrRegImm8(r54, 32)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
				ctx.BindReg(r54, &d137)
			}
			ctx.EnsureDesc(&d137)
			if thisptr.Loc == scm.LocImm {
				baseReg := ctx.AllocReg()
				if d137.Loc == scm.LocReg {
					ctx.FreeReg(baseReg)
					baseReg = ctx.AllocRegExcept(d137.Reg)
				}
				ctx.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).lastValue)))
				if d137.Loc == scm.LocImm {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d137.Imm.Int()))
					ctx.EmitStoreRegMem(scm.RegR11, baseReg, 0)
				} else {
					ctx.EmitStoreRegMem(d137.Reg, baseReg, 0)
				}
				ctx.FreeReg(baseReg)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
				if d137.Loc == scm.LocImm {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d137.Imm.Int()))
					ctx.EmitStoreRegMem(scm.RegR11, thisptr.Reg, off)
				} else {
					ctx.EmitStoreRegMem(d137.Reg, thisptr.Reg, off)
				}
			}
			ctx.FreeDesc(&d137)
			ctx.EnsureDesc(&d4)
			d138 = d4
			_ = d138
			r55 := d4.Loc == scm.LocReg
			r56 := d4.Reg
			if r55 { ctx.ProtectReg(r56) }
			phiBase139 := ctx.AllocStack(int32(16))
			d140 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase139)+int32(160)}
			lbl22 := ctx.ReserveLabel()
			bbpos_2_0 := int32(-1)
			_ = bbpos_2_0
			bbpos_2_1 := int32(-1)
			_ = bbpos_2_1
			bbpos_2_2 := int32(-1)
			_ = bbpos_2_2
			bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d140 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(160)}
			ctx.EnsureDesc(&d138)
			ctx.EnsureDesc(&d138)
			var d141 scm.JITValueDesc
			if d138.Loc == scm.LocImm {
				d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d138.Imm.Int()))))}
			} else {
				r57 := ctx.AllocReg()
				ctx.EmitMovRegReg(r57, d138.Reg)
				ctx.EmitShlRegImm8(r57, 32)
				ctx.EmitShrRegImm8(r57, 32)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r57}
				ctx.BindReg(r57, &d141)
			}
			var d142 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				r58 := ctx.AllocReg()
				ctx.EmitMovRegMem8(r58, fieldAddr)
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r58}
				ctx.BindReg(r58, &d142)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r59 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r59, thisptr.Reg, off)
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r59}
				ctx.BindReg(r59, &d142)
			}
			ctx.EnsureDesc(&d142)
			ctx.EnsureDesc(&d142)
			var d143 scm.JITValueDesc
			if d142.Loc == scm.LocImm {
				d143 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d142.Imm.Int()))))}
			} else {
				r60 := ctx.AllocReg()
				ctx.EmitMovRegReg(r60, d142.Reg)
				ctx.EmitShlRegImm8(r60, 56)
				ctx.EmitShrRegImm8(r60, 56)
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r60}
				ctx.BindReg(r60, &d143)
			}
			ctx.EnsureDesc(&d141)
			ctx.EnsureDesc(&d143)
			ctx.EnsureDesc(&d141)
			ctx.ProtectReg(d141.Reg)
			ctx.EnsureDesc(&d143)
			ctx.UnprotectReg(d141.Reg)
			var d144 scm.JITValueDesc
			if d141.Loc == scm.LocImm && d143.Loc == scm.LocImm {
				d144 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d141.Imm.Int() * d143.Imm.Int())}
			} else if d141.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d143.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d141.Imm.Int()))
				ctx.EmitImulInt64(scratch, d143.Reg)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d144)
			} else if d143.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d141.Reg)
				ctx.EmitMovRegReg(scratch, d141.Reg)
				if d143.Imm.Int() >= -2147483648 && d143.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d143.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d143.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d144)
			} else {
				r61 := ctx.AllocRegExcept(d141.Reg, d143.Reg)
				ctx.EmitMovRegReg(r61, d141.Reg)
				ctx.EmitImulInt64(r61, d143.Reg)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r61}
				ctx.BindReg(r61, &d144)
			}
			if d144.Loc == scm.LocReg && d141.Loc == scm.LocReg && d144.Reg == d141.Reg {
				ctx.TransferReg(d141.Reg)
				d141.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d141)
			ctx.FreeDesc(&d143)
			var d145 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 0
				r62 := ctx.AllocReg()
				r63 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r62, fieldAddr)
				ctx.EmitMovRegMem64(r63, fieldAddr+8)
				d145 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r62, Reg2: r63}
				ctx.BindReg(r62, &d145)
				ctx.BindReg(r63, &d145)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 0)
				r64 := ctx.AllocReg()
				r65 := ctx.AllocReg()
				ctx.EmitMovRegMem(r64, thisptr.Reg, off)
				ctx.EmitMovRegMem(r65, thisptr.Reg, off+8)
				d145 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r64, Reg2: r65}
				ctx.BindReg(r64, &d145)
				ctx.BindReg(r65, &d145)
			}
			ctx.EnsureDesc(&d144)
			var d146 scm.JITValueDesc
			if d144.Loc == scm.LocImm {
				d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d144.Imm.Int() / 64)}
			} else {
				r66 := ctx.AllocRegExcept(d144.Reg)
				ctx.EmitMovRegReg(r66, d144.Reg)
				ctx.EmitShrRegImm8(r66, 6)
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
				ctx.BindReg(r66, &d146)
			}
			if d146.Loc == scm.LocReg && d144.Loc == scm.LocReg && d146.Reg == d144.Reg {
				ctx.TransferReg(d144.Reg)
				d144.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d146)
			r67 := ctx.AllocReg()
			ctx.EnsureDesc(&d146)
			ctx.EnsureDesc(&d145)
			if d146.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r67, uint64(d146.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r67, d146.Reg)
				ctx.EmitShlRegImm8(r67, 3)
			}
			if d145.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d145.Imm.Int()))
				ctx.EmitAddInt64(r67, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r67, d145.Reg)
			}
			r68 := ctx.AllocRegExcept(r67)
			ctx.EmitMovRegMem(r68, r67, 0)
			ctx.FreeReg(r67)
			d147 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r68}
			ctx.BindReg(r68, &d147)
			ctx.FreeDesc(&d146)
			ctx.EnsureDesc(&d144)
			var d148 scm.JITValueDesc
			if d144.Loc == scm.LocImm {
				d148 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d144.Imm.Int() % 64)}
			} else {
				r69 := ctx.AllocRegExcept(d144.Reg)
				ctx.EmitMovRegReg(r69, d144.Reg)
				ctx.EmitAndRegImm32(r69, 63)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r69}
				ctx.BindReg(r69, &d148)
			}
			if d148.Loc == scm.LocReg && d144.Loc == scm.LocReg && d148.Reg == d144.Reg {
				ctx.TransferReg(d144.Reg)
				d144.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d147)
			ctx.EnsureDesc(&d148)
			var d149 scm.JITValueDesc
			if d147.Loc == scm.LocImm && d148.Loc == scm.LocImm {
				d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d147.Imm.Int()) << uint64(d148.Imm.Int())))}
			} else if d148.Loc == scm.LocImm {
				r70 := ctx.AllocRegExcept(d147.Reg)
				ctx.EmitMovRegReg(r70, d147.Reg)
				ctx.EmitShlRegImm8(r70, uint8(d148.Imm.Int()))
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r70}
				ctx.BindReg(r70, &d149)
			} else {
				{
					shiftSrc := d147.Reg
					r71 := ctx.AllocRegExcept(d147.Reg)
					ctx.EmitMovRegReg(r71, d147.Reg)
					shiftSrc = r71
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d148.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d148.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d148.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d149)
				}
			}
			if d149.Loc == scm.LocReg && d147.Loc == scm.LocReg && d149.Reg == d147.Reg {
				ctx.TransferReg(d147.Reg)
				d147.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d147)
			ctx.FreeDesc(&d148)
			ctx.EnsureDesc(&d144)
			var d150 scm.JITValueDesc
			if d144.Loc == scm.LocImm {
				d150 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d144.Imm.Int() % 64)}
			} else {
				r72 := ctx.AllocRegExcept(d144.Reg)
				ctx.EmitMovRegReg(r72, d144.Reg)
				ctx.EmitAndRegImm32(r72, 63)
				d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r72}
				ctx.BindReg(r72, &d150)
			}
			if d150.Loc == scm.LocReg && d144.Loc == scm.LocReg && d150.Reg == d144.Reg {
				ctx.TransferReg(d144.Reg)
				d144.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d142)
			ctx.EnsureDesc(&d142)
			var d151 scm.JITValueDesc
			if d142.Loc == scm.LocImm {
				d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d142.Imm.Int()))))}
			} else {
				r73 := ctx.AllocReg()
				ctx.EmitMovRegReg(r73, d142.Reg)
				ctx.EmitShlRegImm8(r73, 56)
				ctx.EmitShrRegImm8(r73, 56)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
				ctx.BindReg(r73, &d151)
			}
			ctx.EnsureDesc(&d150)
			ctx.EnsureDesc(&d151)
			ctx.EnsureDesc(&d150)
			ctx.ProtectReg(d150.Reg)
			ctx.EnsureDesc(&d151)
			ctx.UnprotectReg(d150.Reg)
			var d152 scm.JITValueDesc
			if d150.Loc == scm.LocImm && d151.Loc == scm.LocImm {
				d152 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d150.Imm.Int() + d151.Imm.Int())}
			} else if d151.Loc == scm.LocImm && d151.Imm.Int() == 0 {
				r74 := ctx.AllocRegExcept(d150.Reg)
				ctx.EmitMovRegReg(r74, d150.Reg)
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r74}
				ctx.BindReg(r74, &d152)
			} else if d150.Loc == scm.LocImm && d150.Imm.Int() == 0 {
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d151.Reg}
				ctx.BindReg(d151.Reg, &d152)
			} else if d150.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d151.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d150.Imm.Int()))
				ctx.EmitAddInt64(scratch, d151.Reg)
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d152)
			} else if d151.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d150.Reg)
				ctx.EmitMovRegReg(scratch, d150.Reg)
				if d151.Imm.Int() >= -2147483648 && d151.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d151.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d151.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d152)
			} else {
				r75 := ctx.AllocRegExcept(d150.Reg, d151.Reg)
				ctx.EmitMovRegReg(r75, d150.Reg)
				ctx.EmitAddInt64(r75, d151.Reg)
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
				ctx.BindReg(r75, &d152)
			}
			if d152.Loc == scm.LocReg && d150.Loc == scm.LocReg && d152.Reg == d150.Reg {
				ctx.TransferReg(d150.Reg)
				d150.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d150)
			ctx.FreeDesc(&d151)
			ctx.EnsureDesc(&d152)
			var d153 scm.JITValueDesc
			if d152.Loc == scm.LocImm {
				d153 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d152.Imm.Int()) > uint64(64))}
			} else {
				r76 := ctx.AllocRegExcept(d152.Reg)
				ctx.EmitCmpRegImm32(d152.Reg, 64)
				ctx.EmitSetcc(r76, scm.CcA)
				d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r76}
				ctx.BindReg(r76, &d153)
			}
			ctx.FreeDesc(&d152)
			d154 = d153
			ctx.EnsureDesc(&d154)
			if d154.Loc != scm.LocImm && d154.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl23 := ctx.ReserveLabel()
			lbl24 := ctx.ReserveLabel()
			lbl25 := ctx.ReserveLabel()
			lbl26 := ctx.ReserveLabel()
			if d154.Loc == scm.LocImm {
				if d154.Imm.Bool() {
					ctx.MarkLabel(lbl25)
					ctx.EmitJmp(lbl23)
				} else {
					ctx.MarkLabel(lbl26)
			ctx.EnsureDesc(&d149)
			if d149.Loc == scm.LocReg {
				ctx.ProtectReg(d149.Reg)
			} else if d149.Loc == scm.LocRegPair {
				ctx.ProtectReg(d149.Reg)
				ctx.ProtectReg(d149.Reg2)
			}
			d155 = d149
			if d155.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d155)
			ctx.EmitStoreToStack(d155, int32(bbs[2].PhiBase)+int32(0))
			if d149.Loc == scm.LocReg {
				ctx.UnprotectReg(d149.Reg)
			} else if d149.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d149.Reg)
				ctx.UnprotectReg(d149.Reg2)
			}
					ctx.EmitJmp(lbl24)
				}
			} else {
				ctx.EmitCmpRegImm32(d154.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl25)
				ctx.EmitJmp(lbl26)
				ctx.MarkLabel(lbl25)
				ctx.EmitJmp(lbl23)
				ctx.MarkLabel(lbl26)
			ctx.EnsureDesc(&d149)
			if d149.Loc == scm.LocReg {
				ctx.ProtectReg(d149.Reg)
			} else if d149.Loc == scm.LocRegPair {
				ctx.ProtectReg(d149.Reg)
				ctx.ProtectReg(d149.Reg2)
			}
			d156 = d149
			if d156.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d156)
			ctx.EmitStoreToStack(d156, int32(bbs[2].PhiBase)+int32(0))
			if d149.Loc == scm.LocReg {
				ctx.UnprotectReg(d149.Reg)
			} else if d149.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d149.Reg)
				ctx.UnprotectReg(d149.Reg2)
			}
				ctx.EmitJmp(lbl24)
			}
			ctx.FreeDesc(&d153)
			bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl24)
			ctx.ResolveFixups()
			d140 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(160)}
			ctx.EnsureDesc(&d142)
			ctx.EnsureDesc(&d142)
			var d157 scm.JITValueDesc
			if d142.Loc == scm.LocImm {
				d157 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d142.Imm.Int()))))}
			} else {
				r77 := ctx.AllocReg()
				ctx.EmitMovRegReg(r77, d142.Reg)
				ctx.EmitShlRegImm8(r77, 56)
				ctx.EmitShrRegImm8(r77, 56)
				d157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r77}
				ctx.BindReg(r77, &d157)
			}
			d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d157)
			ctx.EnsureDesc(&d158)
			ctx.ProtectReg(d158.Reg)
			ctx.EnsureDesc(&d157)
			ctx.UnprotectReg(d158.Reg)
			var d159 scm.JITValueDesc
			if d158.Loc == scm.LocImm && d157.Loc == scm.LocImm {
				d159 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d158.Imm.Int() - d157.Imm.Int())}
			} else if d157.Loc == scm.LocImm && d157.Imm.Int() == 0 {
				r78 := ctx.AllocRegExcept(d158.Reg)
				ctx.EmitMovRegReg(r78, d158.Reg)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r78}
				ctx.BindReg(r78, &d159)
			} else if d158.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d157.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d158.Imm.Int()))
				ctx.EmitSubInt64(scratch, d157.Reg)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d159)
			} else if d157.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d158.Reg)
				ctx.EmitMovRegReg(scratch, d158.Reg)
				if d157.Imm.Int() >= -2147483648 && d157.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d157.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d157.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d159)
			} else {
				r79 := ctx.AllocRegExcept(d158.Reg, d157.Reg)
				ctx.EmitMovRegReg(r79, d158.Reg)
				ctx.EmitSubInt64(r79, d157.Reg)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r79}
				ctx.BindReg(r79, &d159)
			}
			if d159.Loc == scm.LocReg && d158.Loc == scm.LocReg && d159.Reg == d158.Reg {
				ctx.TransferReg(d158.Reg)
				d158.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d157)
			ctx.EnsureDesc(&d140)
			ctx.EnsureDesc(&d159)
			var d160 scm.JITValueDesc
			if d140.Loc == scm.LocImm && d159.Loc == scm.LocImm {
				d160 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d140.Imm.Int()) >> uint64(d159.Imm.Int())))}
			} else if d159.Loc == scm.LocImm {
				r80 := ctx.AllocRegExcept(d140.Reg)
				ctx.EmitMovRegReg(r80, d140.Reg)
				ctx.EmitShrRegImm8(r80, uint8(d159.Imm.Int()))
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
				ctx.BindReg(r80, &d160)
			} else {
				{
					shiftSrc := d140.Reg
					r81 := ctx.AllocRegExcept(d140.Reg)
					ctx.EmitMovRegReg(r81, d140.Reg)
					shiftSrc = r81
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d159.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d159.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d159.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d160)
				}
			}
			if d160.Loc == scm.LocReg && d140.Loc == scm.LocReg && d160.Reg == d140.Reg {
				ctx.TransferReg(d140.Reg)
				d140.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d140)
			ctx.FreeDesc(&d159)
			r82 := ctx.AllocReg()
			ctx.EnsureDesc(&d160)
			ctx.EnsureDesc(&d160)
			if d160.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r82, d160)
			}
			ctx.EmitJmp(lbl22)
			bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl23)
			ctx.ResolveFixups()
			d140 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(160)}
			ctx.EnsureDesc(&d144)
			var d161 scm.JITValueDesc
			if d144.Loc == scm.LocImm {
				d161 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d144.Imm.Int() / 64)}
			} else {
				r83 := ctx.AllocRegExcept(d144.Reg)
				ctx.EmitMovRegReg(r83, d144.Reg)
				ctx.EmitShrRegImm8(r83, 6)
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
				ctx.BindReg(r83, &d161)
			}
			if d161.Loc == scm.LocReg && d144.Loc == scm.LocReg && d161.Reg == d144.Reg {
				ctx.TransferReg(d144.Reg)
				d144.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d161)
			ctx.EnsureDesc(&d161)
			var d162 scm.JITValueDesc
			if d161.Loc == scm.LocImm {
				d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d161.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d161.Reg)
				ctx.EmitMovRegReg(scratch, d161.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d162)
			}
			if d162.Loc == scm.LocReg && d161.Loc == scm.LocReg && d162.Reg == d161.Reg {
				ctx.TransferReg(d161.Reg)
				d161.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d161)
			ctx.EnsureDesc(&d162)
			r84 := ctx.AllocReg()
			ctx.EnsureDesc(&d162)
			ctx.EnsureDesc(&d145)
			if d162.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r84, uint64(d162.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r84, d162.Reg)
				ctx.EmitShlRegImm8(r84, 3)
			}
			if d145.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d145.Imm.Int()))
				ctx.EmitAddInt64(r84, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r84, d145.Reg)
			}
			r85 := ctx.AllocRegExcept(r84)
			ctx.EmitMovRegMem(r85, r84, 0)
			ctx.FreeReg(r84)
			d163 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r85}
			ctx.BindReg(r85, &d163)
			ctx.FreeDesc(&d162)
			ctx.EnsureDesc(&d144)
			var d164 scm.JITValueDesc
			if d144.Loc == scm.LocImm {
				d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d144.Imm.Int() % 64)}
			} else {
				r86 := ctx.AllocRegExcept(d144.Reg)
				ctx.EmitMovRegReg(r86, d144.Reg)
				ctx.EmitAndRegImm32(r86, 63)
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
				ctx.BindReg(r86, &d164)
			}
			if d164.Loc == scm.LocReg && d144.Loc == scm.LocReg && d164.Reg == d144.Reg {
				ctx.TransferReg(d144.Reg)
				d144.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d144)
			d165 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d164)
			ctx.EnsureDesc(&d165)
			ctx.ProtectReg(d165.Reg)
			ctx.EnsureDesc(&d164)
			ctx.UnprotectReg(d165.Reg)
			var d166 scm.JITValueDesc
			if d165.Loc == scm.LocImm && d164.Loc == scm.LocImm {
				d166 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d165.Imm.Int() - d164.Imm.Int())}
			} else if d164.Loc == scm.LocImm && d164.Imm.Int() == 0 {
				r87 := ctx.AllocRegExcept(d165.Reg)
				ctx.EmitMovRegReg(r87, d165.Reg)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r87}
				ctx.BindReg(r87, &d166)
			} else if d165.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d164.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d165.Imm.Int()))
				ctx.EmitSubInt64(scratch, d164.Reg)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d166)
			} else if d164.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d165.Reg)
				ctx.EmitMovRegReg(scratch, d165.Reg)
				if d164.Imm.Int() >= -2147483648 && d164.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d164.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d164.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d166)
			} else {
				r88 := ctx.AllocRegExcept(d165.Reg, d164.Reg)
				ctx.EmitMovRegReg(r88, d165.Reg)
				ctx.EmitSubInt64(r88, d164.Reg)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
				ctx.BindReg(r88, &d166)
			}
			if d166.Loc == scm.LocReg && d165.Loc == scm.LocReg && d166.Reg == d165.Reg {
				ctx.TransferReg(d165.Reg)
				d165.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d164)
			ctx.EnsureDesc(&d163)
			ctx.EnsureDesc(&d166)
			var d167 scm.JITValueDesc
			if d163.Loc == scm.LocImm && d166.Loc == scm.LocImm {
				d167 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d163.Imm.Int()) >> uint64(d166.Imm.Int())))}
			} else if d166.Loc == scm.LocImm {
				r89 := ctx.AllocRegExcept(d163.Reg)
				ctx.EmitMovRegReg(r89, d163.Reg)
				ctx.EmitShrRegImm8(r89, uint8(d166.Imm.Int()))
				d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
				ctx.BindReg(r89, &d167)
			} else {
				{
					shiftSrc := d163.Reg
					r90 := ctx.AllocRegExcept(d163.Reg)
					ctx.EmitMovRegReg(r90, d163.Reg)
					shiftSrc = r90
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d166.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d166.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d166.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d167)
				}
			}
			if d167.Loc == scm.LocReg && d163.Loc == scm.LocReg && d167.Reg == d163.Reg {
				ctx.TransferReg(d163.Reg)
				d163.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d163)
			ctx.FreeDesc(&d166)
			ctx.EnsureDesc(&d149)
			ctx.EnsureDesc(&d167)
			var d168 scm.JITValueDesc
			if d149.Loc == scm.LocImm && d167.Loc == scm.LocImm {
				d168 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d149.Imm.Int() | d167.Imm.Int())}
			} else if d149.Loc == scm.LocImm && d149.Imm.Int() == 0 {
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d167.Reg}
				ctx.BindReg(d167.Reg, &d168)
			} else if d167.Loc == scm.LocImm && d167.Imm.Int() == 0 {
				r91 := ctx.AllocRegExcept(d149.Reg)
				ctx.EmitMovRegReg(r91, d149.Reg)
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r91}
				ctx.BindReg(r91, &d168)
			} else if d149.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d167.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d149.Imm.Int()))
				ctx.EmitOrInt64(scratch, d167.Reg)
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d168)
			} else if d167.Loc == scm.LocImm {
				r92 := ctx.AllocRegExcept(d149.Reg)
				ctx.EmitMovRegReg(r92, d149.Reg)
				if d167.Imm.Int() >= -2147483648 && d167.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r92, int32(d167.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d167.Imm.Int()))
					ctx.EmitOrInt64(r92, scm.RegR11)
				}
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r92}
				ctx.BindReg(r92, &d168)
			} else {
				r93 := ctx.AllocRegExcept(d149.Reg, d167.Reg)
				ctx.EmitMovRegReg(r93, d149.Reg)
				ctx.EmitOrInt64(r93, d167.Reg)
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r93}
				ctx.BindReg(r93, &d168)
			}
			if d168.Loc == scm.LocReg && d149.Loc == scm.LocReg && d168.Reg == d149.Reg {
				ctx.TransferReg(d149.Reg)
				d149.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d167)
			ctx.EnsureDesc(&d168)
			if d168.Loc == scm.LocReg {
				ctx.ProtectReg(d168.Reg)
			} else if d168.Loc == scm.LocRegPair {
				ctx.ProtectReg(d168.Reg)
				ctx.ProtectReg(d168.Reg2)
			}
			d169 = d168
			if d169.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d169)
			ctx.EmitStoreToStack(d169, int32(bbs[2].PhiBase)+int32(0))
			if d168.Loc == scm.LocReg {
				ctx.UnprotectReg(d168.Reg)
			} else if d168.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d168.Reg)
				ctx.UnprotectReg(d168.Reg2)
			}
			ctx.EmitJmp(lbl24)
			ctx.MarkLabel(lbl22)
			d170 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r82}
			ctx.BindReg(r82, &d170)
			ctx.BindReg(r82, &d170)
			if r55 { ctx.UnprotectReg(r56) }
			ctx.EnsureDesc(&d170)
			ctx.EnsureDesc(&d170)
			var d171 scm.JITValueDesc
			if d170.Loc == scm.LocImm {
				d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d170.Imm.Int()))))}
			} else {
				r94 := ctx.AllocReg()
				ctx.EmitMovRegReg(r94, d170.Reg)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
				ctx.BindReg(r94, &d171)
			}
			ctx.FreeDesc(&d170)
			var d172 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 32
				r95 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r95, fieldAddr)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r95}
				ctx.BindReg(r95, &d172)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 32)
				r96 := ctx.AllocReg()
				ctx.EmitMovRegMem(r96, thisptr.Reg, off)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r96}
				ctx.BindReg(r96, &d172)
			}
			ctx.EnsureDesc(&d171)
			ctx.EnsureDesc(&d172)
			ctx.EnsureDesc(&d171)
			ctx.ProtectReg(d171.Reg)
			ctx.EnsureDesc(&d172)
			ctx.UnprotectReg(d171.Reg)
			var d173 scm.JITValueDesc
			if d171.Loc == scm.LocImm && d172.Loc == scm.LocImm {
				d173 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d171.Imm.Int() + d172.Imm.Int())}
			} else if d172.Loc == scm.LocImm && d172.Imm.Int() == 0 {
				r97 := ctx.AllocRegExcept(d171.Reg)
				ctx.EmitMovRegReg(r97, d171.Reg)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
				ctx.BindReg(r97, &d173)
			} else if d171.Loc == scm.LocImm && d171.Imm.Int() == 0 {
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d172.Reg}
				ctx.BindReg(d172.Reg, &d173)
			} else if d171.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d172.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d171.Imm.Int()))
				ctx.EmitAddInt64(scratch, d172.Reg)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d173)
			} else if d172.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d171.Reg)
				ctx.EmitMovRegReg(scratch, d171.Reg)
				if d172.Imm.Int() >= -2147483648 && d172.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d172.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d172.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d173)
			} else {
				r98 := ctx.AllocRegExcept(d171.Reg, d172.Reg)
				ctx.EmitMovRegReg(r98, d171.Reg)
				ctx.EmitAddInt64(r98, d172.Reg)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r98}
				ctx.BindReg(r98, &d173)
			}
			if d173.Loc == scm.LocReg && d171.Loc == scm.LocReg && d173.Reg == d171.Reg {
				ctx.TransferReg(d171.Reg)
				d171.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d171)
			var d174 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 56
				r99 := ctx.AllocReg()
				ctx.EmitMovRegMem8(r99, fieldAddr)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r99}
				ctx.BindReg(r99, &d174)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 56)
				r100 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r100, thisptr.Reg, off)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r100}
				ctx.BindReg(r100, &d174)
			}
			d175 = d174
			ctx.EnsureDesc(&d175)
			if d175.Loc != scm.LocImm && d175.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d175.Loc == scm.LocImm {
				if d175.Imm.Bool() {
			ps176 := scm.PhiState{General: ps.General}
			ps176.OverlayValues = make([]scm.JITValueDesc, 176)
			ps176.OverlayValues[1] = d1
			ps176.OverlayValues[2] = d2
			ps176.OverlayValues[3] = d3
			ps176.OverlayValues[4] = d4
			ps176.OverlayValues[5] = d5
			ps176.OverlayValues[6] = d6
			ps176.OverlayValues[7] = d7
			ps176.OverlayValues[8] = d8
			ps176.OverlayValues[9] = d9
			ps176.OverlayValues[10] = d10
			ps176.OverlayValues[11] = d11
			ps176.OverlayValues[12] = d12
			ps176.OverlayValues[13] = d13
			ps176.OverlayValues[14] = d14
			ps176.OverlayValues[15] = d15
			ps176.OverlayValues[16] = d16
			ps176.OverlayValues[17] = d17
			ps176.OverlayValues[19] = d19
			ps176.OverlayValues[20] = d20
			ps176.OverlayValues[21] = d21
			ps176.OverlayValues[22] = d22
			ps176.OverlayValues[23] = d23
			ps176.OverlayValues[24] = d24
			ps176.OverlayValues[25] = d25
			ps176.OverlayValues[27] = d27
			ps176.OverlayValues[28] = d28
			ps176.OverlayValues[29] = d29
			ps176.OverlayValues[30] = d30
			ps176.OverlayValues[31] = d31
			ps176.OverlayValues[32] = d32
			ps176.OverlayValues[33] = d33
			ps176.OverlayValues[34] = d34
			ps176.OverlayValues[35] = d35
			ps176.OverlayValues[36] = d36
			ps176.OverlayValues[37] = d37
			ps176.OverlayValues[38] = d38
			ps176.OverlayValues[39] = d39
			ps176.OverlayValues[40] = d40
			ps176.OverlayValues[41] = d41
			ps176.OverlayValues[42] = d42
			ps176.OverlayValues[43] = d43
			ps176.OverlayValues[44] = d44
			ps176.OverlayValues[45] = d45
			ps176.OverlayValues[46] = d46
			ps176.OverlayValues[47] = d47
			ps176.OverlayValues[48] = d48
			ps176.OverlayValues[49] = d49
			ps176.OverlayValues[50] = d50
			ps176.OverlayValues[51] = d51
			ps176.OverlayValues[52] = d52
			ps176.OverlayValues[53] = d53
			ps176.OverlayValues[54] = d54
			ps176.OverlayValues[55] = d55
			ps176.OverlayValues[56] = d56
			ps176.OverlayValues[57] = d57
			ps176.OverlayValues[58] = d58
			ps176.OverlayValues[59] = d59
			ps176.OverlayValues[60] = d60
			ps176.OverlayValues[61] = d61
			ps176.OverlayValues[62] = d62
			ps176.OverlayValues[63] = d63
			ps176.OverlayValues[66] = d66
			ps176.OverlayValues[67] = d67
			ps176.OverlayValues[68] = d68
			ps176.OverlayValues[136] = d136
			ps176.OverlayValues[137] = d137
			ps176.OverlayValues[138] = d138
			ps176.OverlayValues[140] = d140
			ps176.OverlayValues[141] = d141
			ps176.OverlayValues[142] = d142
			ps176.OverlayValues[143] = d143
			ps176.OverlayValues[144] = d144
			ps176.OverlayValues[145] = d145
			ps176.OverlayValues[146] = d146
			ps176.OverlayValues[147] = d147
			ps176.OverlayValues[148] = d148
			ps176.OverlayValues[149] = d149
			ps176.OverlayValues[150] = d150
			ps176.OverlayValues[151] = d151
			ps176.OverlayValues[152] = d152
			ps176.OverlayValues[153] = d153
			ps176.OverlayValues[154] = d154
			ps176.OverlayValues[155] = d155
			ps176.OverlayValues[156] = d156
			ps176.OverlayValues[157] = d157
			ps176.OverlayValues[158] = d158
			ps176.OverlayValues[159] = d159
			ps176.OverlayValues[160] = d160
			ps176.OverlayValues[161] = d161
			ps176.OverlayValues[162] = d162
			ps176.OverlayValues[163] = d163
			ps176.OverlayValues[164] = d164
			ps176.OverlayValues[165] = d165
			ps176.OverlayValues[166] = d166
			ps176.OverlayValues[167] = d167
			ps176.OverlayValues[168] = d168
			ps176.OverlayValues[169] = d169
			ps176.OverlayValues[170] = d170
			ps176.OverlayValues[171] = d171
			ps176.OverlayValues[172] = d172
			ps176.OverlayValues[173] = d173
			ps176.OverlayValues[174] = d174
			ps176.OverlayValues[175] = d175
					return bbs[13].RenderPS(ps176)
				}
			ps177 := scm.PhiState{General: ps.General}
			ps177.OverlayValues = make([]scm.JITValueDesc, 176)
			ps177.OverlayValues[1] = d1
			ps177.OverlayValues[2] = d2
			ps177.OverlayValues[3] = d3
			ps177.OverlayValues[4] = d4
			ps177.OverlayValues[5] = d5
			ps177.OverlayValues[6] = d6
			ps177.OverlayValues[7] = d7
			ps177.OverlayValues[8] = d8
			ps177.OverlayValues[9] = d9
			ps177.OverlayValues[10] = d10
			ps177.OverlayValues[11] = d11
			ps177.OverlayValues[12] = d12
			ps177.OverlayValues[13] = d13
			ps177.OverlayValues[14] = d14
			ps177.OverlayValues[15] = d15
			ps177.OverlayValues[16] = d16
			ps177.OverlayValues[17] = d17
			ps177.OverlayValues[19] = d19
			ps177.OverlayValues[20] = d20
			ps177.OverlayValues[21] = d21
			ps177.OverlayValues[22] = d22
			ps177.OverlayValues[23] = d23
			ps177.OverlayValues[24] = d24
			ps177.OverlayValues[25] = d25
			ps177.OverlayValues[27] = d27
			ps177.OverlayValues[28] = d28
			ps177.OverlayValues[29] = d29
			ps177.OverlayValues[30] = d30
			ps177.OverlayValues[31] = d31
			ps177.OverlayValues[32] = d32
			ps177.OverlayValues[33] = d33
			ps177.OverlayValues[34] = d34
			ps177.OverlayValues[35] = d35
			ps177.OverlayValues[36] = d36
			ps177.OverlayValues[37] = d37
			ps177.OverlayValues[38] = d38
			ps177.OverlayValues[39] = d39
			ps177.OverlayValues[40] = d40
			ps177.OverlayValues[41] = d41
			ps177.OverlayValues[42] = d42
			ps177.OverlayValues[43] = d43
			ps177.OverlayValues[44] = d44
			ps177.OverlayValues[45] = d45
			ps177.OverlayValues[46] = d46
			ps177.OverlayValues[47] = d47
			ps177.OverlayValues[48] = d48
			ps177.OverlayValues[49] = d49
			ps177.OverlayValues[50] = d50
			ps177.OverlayValues[51] = d51
			ps177.OverlayValues[52] = d52
			ps177.OverlayValues[53] = d53
			ps177.OverlayValues[54] = d54
			ps177.OverlayValues[55] = d55
			ps177.OverlayValues[56] = d56
			ps177.OverlayValues[57] = d57
			ps177.OverlayValues[58] = d58
			ps177.OverlayValues[59] = d59
			ps177.OverlayValues[60] = d60
			ps177.OverlayValues[61] = d61
			ps177.OverlayValues[62] = d62
			ps177.OverlayValues[63] = d63
			ps177.OverlayValues[66] = d66
			ps177.OverlayValues[67] = d67
			ps177.OverlayValues[68] = d68
			ps177.OverlayValues[136] = d136
			ps177.OverlayValues[137] = d137
			ps177.OverlayValues[138] = d138
			ps177.OverlayValues[140] = d140
			ps177.OverlayValues[141] = d141
			ps177.OverlayValues[142] = d142
			ps177.OverlayValues[143] = d143
			ps177.OverlayValues[144] = d144
			ps177.OverlayValues[145] = d145
			ps177.OverlayValues[146] = d146
			ps177.OverlayValues[147] = d147
			ps177.OverlayValues[148] = d148
			ps177.OverlayValues[149] = d149
			ps177.OverlayValues[150] = d150
			ps177.OverlayValues[151] = d151
			ps177.OverlayValues[152] = d152
			ps177.OverlayValues[153] = d153
			ps177.OverlayValues[154] = d154
			ps177.OverlayValues[155] = d155
			ps177.OverlayValues[156] = d156
			ps177.OverlayValues[157] = d157
			ps177.OverlayValues[158] = d158
			ps177.OverlayValues[159] = d159
			ps177.OverlayValues[160] = d160
			ps177.OverlayValues[161] = d161
			ps177.OverlayValues[162] = d162
			ps177.OverlayValues[163] = d163
			ps177.OverlayValues[164] = d164
			ps177.OverlayValues[165] = d165
			ps177.OverlayValues[166] = d166
			ps177.OverlayValues[167] = d167
			ps177.OverlayValues[168] = d168
			ps177.OverlayValues[169] = d169
			ps177.OverlayValues[170] = d170
			ps177.OverlayValues[171] = d171
			ps177.OverlayValues[172] = d172
			ps177.OverlayValues[173] = d173
			ps177.OverlayValues[174] = d174
			ps177.OverlayValues[175] = d175
				return bbs[12].RenderPS(ps177)
			}
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
					d178 := ps.PhiValues[0]
					ctx.EnsureDesc(&d178)
					ctx.EmitStoreToStack(d178, int32(bbs[2].PhiBase)+int32(0))
				}
				ps.General = true
				return bbs[2].RenderPS(ps)
			}
			lbl27 := ctx.ReserveLabel()
			lbl28 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d175.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl27)
			ctx.EmitJmp(lbl28)
			ctx.MarkLabel(lbl27)
			ctx.EmitJmp(lbl14)
			ctx.MarkLabel(lbl28)
			ctx.EmitJmp(lbl13)
			ps179 := scm.PhiState{General: true}
			ps179.OverlayValues = make([]scm.JITValueDesc, 179)
			ps179.OverlayValues[1] = d1
			ps179.OverlayValues[2] = d2
			ps179.OverlayValues[3] = d3
			ps179.OverlayValues[4] = d4
			ps179.OverlayValues[5] = d5
			ps179.OverlayValues[6] = d6
			ps179.OverlayValues[7] = d7
			ps179.OverlayValues[8] = d8
			ps179.OverlayValues[9] = d9
			ps179.OverlayValues[10] = d10
			ps179.OverlayValues[11] = d11
			ps179.OverlayValues[12] = d12
			ps179.OverlayValues[13] = d13
			ps179.OverlayValues[14] = d14
			ps179.OverlayValues[15] = d15
			ps179.OverlayValues[16] = d16
			ps179.OverlayValues[17] = d17
			ps179.OverlayValues[19] = d19
			ps179.OverlayValues[20] = d20
			ps179.OverlayValues[21] = d21
			ps179.OverlayValues[22] = d22
			ps179.OverlayValues[23] = d23
			ps179.OverlayValues[24] = d24
			ps179.OverlayValues[25] = d25
			ps179.OverlayValues[27] = d27
			ps179.OverlayValues[28] = d28
			ps179.OverlayValues[29] = d29
			ps179.OverlayValues[30] = d30
			ps179.OverlayValues[31] = d31
			ps179.OverlayValues[32] = d32
			ps179.OverlayValues[33] = d33
			ps179.OverlayValues[34] = d34
			ps179.OverlayValues[35] = d35
			ps179.OverlayValues[36] = d36
			ps179.OverlayValues[37] = d37
			ps179.OverlayValues[38] = d38
			ps179.OverlayValues[39] = d39
			ps179.OverlayValues[40] = d40
			ps179.OverlayValues[41] = d41
			ps179.OverlayValues[42] = d42
			ps179.OverlayValues[43] = d43
			ps179.OverlayValues[44] = d44
			ps179.OverlayValues[45] = d45
			ps179.OverlayValues[46] = d46
			ps179.OverlayValues[47] = d47
			ps179.OverlayValues[48] = d48
			ps179.OverlayValues[49] = d49
			ps179.OverlayValues[50] = d50
			ps179.OverlayValues[51] = d51
			ps179.OverlayValues[52] = d52
			ps179.OverlayValues[53] = d53
			ps179.OverlayValues[54] = d54
			ps179.OverlayValues[55] = d55
			ps179.OverlayValues[56] = d56
			ps179.OverlayValues[57] = d57
			ps179.OverlayValues[58] = d58
			ps179.OverlayValues[59] = d59
			ps179.OverlayValues[60] = d60
			ps179.OverlayValues[61] = d61
			ps179.OverlayValues[62] = d62
			ps179.OverlayValues[63] = d63
			ps179.OverlayValues[66] = d66
			ps179.OverlayValues[67] = d67
			ps179.OverlayValues[68] = d68
			ps179.OverlayValues[136] = d136
			ps179.OverlayValues[137] = d137
			ps179.OverlayValues[138] = d138
			ps179.OverlayValues[140] = d140
			ps179.OverlayValues[141] = d141
			ps179.OverlayValues[142] = d142
			ps179.OverlayValues[143] = d143
			ps179.OverlayValues[144] = d144
			ps179.OverlayValues[145] = d145
			ps179.OverlayValues[146] = d146
			ps179.OverlayValues[147] = d147
			ps179.OverlayValues[148] = d148
			ps179.OverlayValues[149] = d149
			ps179.OverlayValues[150] = d150
			ps179.OverlayValues[151] = d151
			ps179.OverlayValues[152] = d152
			ps179.OverlayValues[153] = d153
			ps179.OverlayValues[154] = d154
			ps179.OverlayValues[155] = d155
			ps179.OverlayValues[156] = d156
			ps179.OverlayValues[157] = d157
			ps179.OverlayValues[158] = d158
			ps179.OverlayValues[159] = d159
			ps179.OverlayValues[160] = d160
			ps179.OverlayValues[161] = d161
			ps179.OverlayValues[162] = d162
			ps179.OverlayValues[163] = d163
			ps179.OverlayValues[164] = d164
			ps179.OverlayValues[165] = d165
			ps179.OverlayValues[166] = d166
			ps179.OverlayValues[167] = d167
			ps179.OverlayValues[168] = d168
			ps179.OverlayValues[169] = d169
			ps179.OverlayValues[170] = d170
			ps179.OverlayValues[171] = d171
			ps179.OverlayValues[172] = d172
			ps179.OverlayValues[173] = d173
			ps179.OverlayValues[174] = d174
			ps179.OverlayValues[175] = d175
			ps179.OverlayValues[178] = d178
			ps180 := scm.PhiState{General: true}
			ps180.OverlayValues = make([]scm.JITValueDesc, 179)
			ps180.OverlayValues[1] = d1
			ps180.OverlayValues[2] = d2
			ps180.OverlayValues[3] = d3
			ps180.OverlayValues[4] = d4
			ps180.OverlayValues[5] = d5
			ps180.OverlayValues[6] = d6
			ps180.OverlayValues[7] = d7
			ps180.OverlayValues[8] = d8
			ps180.OverlayValues[9] = d9
			ps180.OverlayValues[10] = d10
			ps180.OverlayValues[11] = d11
			ps180.OverlayValues[12] = d12
			ps180.OverlayValues[13] = d13
			ps180.OverlayValues[14] = d14
			ps180.OverlayValues[15] = d15
			ps180.OverlayValues[16] = d16
			ps180.OverlayValues[17] = d17
			ps180.OverlayValues[19] = d19
			ps180.OverlayValues[20] = d20
			ps180.OverlayValues[21] = d21
			ps180.OverlayValues[22] = d22
			ps180.OverlayValues[23] = d23
			ps180.OverlayValues[24] = d24
			ps180.OverlayValues[25] = d25
			ps180.OverlayValues[27] = d27
			ps180.OverlayValues[28] = d28
			ps180.OverlayValues[29] = d29
			ps180.OverlayValues[30] = d30
			ps180.OverlayValues[31] = d31
			ps180.OverlayValues[32] = d32
			ps180.OverlayValues[33] = d33
			ps180.OverlayValues[34] = d34
			ps180.OverlayValues[35] = d35
			ps180.OverlayValues[36] = d36
			ps180.OverlayValues[37] = d37
			ps180.OverlayValues[38] = d38
			ps180.OverlayValues[39] = d39
			ps180.OverlayValues[40] = d40
			ps180.OverlayValues[41] = d41
			ps180.OverlayValues[42] = d42
			ps180.OverlayValues[43] = d43
			ps180.OverlayValues[44] = d44
			ps180.OverlayValues[45] = d45
			ps180.OverlayValues[46] = d46
			ps180.OverlayValues[47] = d47
			ps180.OverlayValues[48] = d48
			ps180.OverlayValues[49] = d49
			ps180.OverlayValues[50] = d50
			ps180.OverlayValues[51] = d51
			ps180.OverlayValues[52] = d52
			ps180.OverlayValues[53] = d53
			ps180.OverlayValues[54] = d54
			ps180.OverlayValues[55] = d55
			ps180.OverlayValues[56] = d56
			ps180.OverlayValues[57] = d57
			ps180.OverlayValues[58] = d58
			ps180.OverlayValues[59] = d59
			ps180.OverlayValues[60] = d60
			ps180.OverlayValues[61] = d61
			ps180.OverlayValues[62] = d62
			ps180.OverlayValues[63] = d63
			ps180.OverlayValues[66] = d66
			ps180.OverlayValues[67] = d67
			ps180.OverlayValues[68] = d68
			ps180.OverlayValues[136] = d136
			ps180.OverlayValues[137] = d137
			ps180.OverlayValues[138] = d138
			ps180.OverlayValues[140] = d140
			ps180.OverlayValues[141] = d141
			ps180.OverlayValues[142] = d142
			ps180.OverlayValues[143] = d143
			ps180.OverlayValues[144] = d144
			ps180.OverlayValues[145] = d145
			ps180.OverlayValues[146] = d146
			ps180.OverlayValues[147] = d147
			ps180.OverlayValues[148] = d148
			ps180.OverlayValues[149] = d149
			ps180.OverlayValues[150] = d150
			ps180.OverlayValues[151] = d151
			ps180.OverlayValues[152] = d152
			ps180.OverlayValues[153] = d153
			ps180.OverlayValues[154] = d154
			ps180.OverlayValues[155] = d155
			ps180.OverlayValues[156] = d156
			ps180.OverlayValues[157] = d157
			ps180.OverlayValues[158] = d158
			ps180.OverlayValues[159] = d159
			ps180.OverlayValues[160] = d160
			ps180.OverlayValues[161] = d161
			ps180.OverlayValues[162] = d162
			ps180.OverlayValues[163] = d163
			ps180.OverlayValues[164] = d164
			ps180.OverlayValues[165] = d165
			ps180.OverlayValues[166] = d166
			ps180.OverlayValues[167] = d167
			ps180.OverlayValues[168] = d168
			ps180.OverlayValues[169] = d169
			ps180.OverlayValues[170] = d170
			ps180.OverlayValues[171] = d171
			ps180.OverlayValues[172] = d172
			ps180.OverlayValues[173] = d173
			ps180.OverlayValues[174] = d174
			ps180.OverlayValues[175] = d175
			ps180.OverlayValues[178] = d178
			snap181 := d1
			snap182 := d2
			snap183 := d3
			snap184 := d4
			snap185 := d5
			snap186 := d6
			snap187 := d7
			snap188 := d8
			snap189 := d9
			snap190 := d10
			snap191 := d11
			snap192 := d12
			snap193 := d13
			snap194 := d14
			snap195 := d15
			snap196 := d16
			snap197 := d17
			snap198 := d19
			snap199 := d20
			snap200 := d21
			snap201 := d22
			snap202 := d23
			snap203 := d24
			snap204 := d25
			snap205 := d27
			snap206 := d28
			snap207 := d29
			snap208 := d30
			snap209 := d31
			snap210 := d32
			snap211 := d33
			snap212 := d34
			snap213 := d35
			snap214 := d36
			snap215 := d37
			snap216 := d38
			snap217 := d39
			snap218 := d40
			snap219 := d41
			snap220 := d42
			snap221 := d43
			snap222 := d44
			snap223 := d45
			snap224 := d46
			snap225 := d47
			snap226 := d48
			snap227 := d49
			snap228 := d50
			snap229 := d51
			snap230 := d52
			snap231 := d53
			snap232 := d54
			snap233 := d55
			snap234 := d56
			snap235 := d57
			snap236 := d58
			snap237 := d59
			snap238 := d60
			snap239 := d61
			snap240 := d62
			snap241 := d63
			snap242 := d66
			snap243 := d67
			snap244 := d68
			snap245 := d136
			snap246 := d137
			snap247 := d138
			snap248 := d140
			snap249 := d141
			snap250 := d142
			snap251 := d143
			snap252 := d144
			snap253 := d145
			snap254 := d146
			snap255 := d147
			snap256 := d148
			snap257 := d149
			snap258 := d150
			snap259 := d151
			snap260 := d152
			snap261 := d153
			snap262 := d154
			snap263 := d155
			snap264 := d156
			snap265 := d157
			snap266 := d158
			snap267 := d159
			snap268 := d160
			snap269 := d161
			snap270 := d162
			snap271 := d163
			snap272 := d164
			snap273 := d165
			snap274 := d166
			snap275 := d167
			snap276 := d168
			snap277 := d169
			snap278 := d170
			snap279 := d171
			snap280 := d172
			snap281 := d173
			snap282 := d174
			snap283 := d175
			snap284 := d178
			alloc285 := ctx.SnapshotAllocState()
			if !bbs[12].Rendered {
				bbs[12].RenderPS(ps180)
			}
			ctx.RestoreAllocState(alloc285)
			d1 = snap181
			d2 = snap182
			d3 = snap183
			d4 = snap184
			d5 = snap185
			d6 = snap186
			d7 = snap187
			d8 = snap188
			d9 = snap189
			d10 = snap190
			d11 = snap191
			d12 = snap192
			d13 = snap193
			d14 = snap194
			d15 = snap195
			d16 = snap196
			d17 = snap197
			d19 = snap198
			d20 = snap199
			d21 = snap200
			d22 = snap201
			d23 = snap202
			d24 = snap203
			d25 = snap204
			d27 = snap205
			d28 = snap206
			d29 = snap207
			d30 = snap208
			d31 = snap209
			d32 = snap210
			d33 = snap211
			d34 = snap212
			d35 = snap213
			d36 = snap214
			d37 = snap215
			d38 = snap216
			d39 = snap217
			d40 = snap218
			d41 = snap219
			d42 = snap220
			d43 = snap221
			d44 = snap222
			d45 = snap223
			d46 = snap224
			d47 = snap225
			d48 = snap226
			d49 = snap227
			d50 = snap228
			d51 = snap229
			d52 = snap230
			d53 = snap231
			d54 = snap232
			d55 = snap233
			d56 = snap234
			d57 = snap235
			d58 = snap236
			d59 = snap237
			d60 = snap238
			d61 = snap239
			d62 = snap240
			d63 = snap241
			d66 = snap242
			d67 = snap243
			d68 = snap244
			d136 = snap245
			d137 = snap246
			d138 = snap247
			d140 = snap248
			d141 = snap249
			d142 = snap250
			d143 = snap251
			d144 = snap252
			d145 = snap253
			d146 = snap254
			d147 = snap255
			d148 = snap256
			d149 = snap257
			d150 = snap258
			d151 = snap259
			d152 = snap260
			d153 = snap261
			d154 = snap262
			d155 = snap263
			d156 = snap264
			d157 = snap265
			d158 = snap266
			d159 = snap267
			d160 = snap268
			d161 = snap269
			d162 = snap270
			d163 = snap271
			d164 = snap272
			d165 = snap273
			d166 = snap274
			d167 = snap275
			d168 = snap276
			d169 = snap277
			d170 = snap278
			d171 = snap279
			d172 = snap280
			d173 = snap281
			d174 = snap282
			d175 = snap283
			d178 = snap284
			if !bbs[13].Rendered {
				return bbs[13].RenderPS(ps179)
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
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
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
			if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != scm.LocNone {
				d173 = ps.OverlayValues[173]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
			}
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
				d178 = ps.OverlayValues[178]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			var d286 scm.JITValueDesc
			if d1.Loc == scm.LocImm {
				d286 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.EmitMovRegReg(scratch, d1.Reg)
				ctx.EmitSubRegImm32(scratch, int32(1))
				d286 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d286)
			}
			if d286.Loc == scm.LocImm {
				d286 = scm.JITValueDesc{Loc: scm.LocImm, Type: d286.Type, Imm: scm.NewInt(int64(uint64(d286.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d286.Reg, 32)
				ctx.EmitShrRegImm8(d286.Reg, 32)
			}
			if d286.Loc == scm.LocReg && d1.Loc == scm.LocReg && d286.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			var d287 scm.JITValueDesc
			if d1.Loc == scm.LocImm {
				d287 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.EmitMovRegReg(scratch, d1.Reg)
				ctx.EmitSubRegImm32(scratch, int32(1))
				d287 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d287)
			}
			if d287.Loc == scm.LocImm {
				d287 = scm.JITValueDesc{Loc: scm.LocImm, Type: d287.Type, Imm: scm.NewInt(int64(uint64(d287.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d287.Reg, 32)
				ctx.EmitShrRegImm8(d287.Reg, 32)
			}
			if d287.Loc == scm.LocReg && d1.Loc == scm.LocReg && d287.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d2)
			if d2.Loc == scm.LocReg {
				ctx.ProtectReg(d2.Reg)
			} else if d2.Loc == scm.LocRegPair {
				ctx.ProtectReg(d2.Reg)
				ctx.ProtectReg(d2.Reg2)
			}
			ctx.EnsureDesc(&d286)
			if d286.Loc == scm.LocReg {
				ctx.ProtectReg(d286.Reg)
			} else if d286.Loc == scm.LocRegPair {
				ctx.ProtectReg(d286.Reg)
				ctx.ProtectReg(d286.Reg2)
			}
			ctx.EnsureDesc(&d287)
			if d287.Loc == scm.LocReg {
				ctx.ProtectReg(d287.Reg)
			} else if d287.Loc == scm.LocRegPair {
				ctx.ProtectReg(d287.Reg)
				ctx.ProtectReg(d287.Reg2)
			}
			d288 = d287
			if d288.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d288)
			d289 = d288
			if d289.Loc == scm.LocImm {
				d289 = scm.JITValueDesc{Loc: scm.LocImm, Type: d289.Type, Imm: scm.NewInt(int64(uint64(d289.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d289.Reg, 32)
				ctx.EmitShrRegImm8(d289.Reg, 32)
			}
			ctx.EmitStoreToStack(d289, int32(bbs[4].PhiBase)+int32(0))
			d290 = d2
			if d290.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d290)
			d291 = d290
			if d291.Loc == scm.LocImm {
				d291 = scm.JITValueDesc{Loc: scm.LocImm, Type: d291.Type, Imm: scm.NewInt(int64(uint64(d291.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d291.Reg, 32)
				ctx.EmitShrRegImm8(d291.Reg, 32)
			}
			ctx.EmitStoreToStack(d291, int32(bbs[4].PhiBase)+int32(16))
			d292 = d286
			if d292.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d292)
			d293 = d292
			if d293.Loc == scm.LocImm {
				d293 = scm.JITValueDesc{Loc: scm.LocImm, Type: d293.Type, Imm: scm.NewInt(int64(uint64(d293.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d293.Reg, 32)
				ctx.EmitShrRegImm8(d293.Reg, 32)
			}
			ctx.EmitStoreToStack(d293, int32(bbs[4].PhiBase)+int32(32))
			if d2.Loc == scm.LocReg {
				ctx.UnprotectReg(d2.Reg)
			} else if d2.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d2.Reg)
				ctx.UnprotectReg(d2.Reg2)
			}
			if d286.Loc == scm.LocReg {
				ctx.UnprotectReg(d286.Reg)
			} else if d286.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d286.Reg)
				ctx.UnprotectReg(d286.Reg2)
			}
			if d287.Loc == scm.LocReg {
				ctx.UnprotectReg(d287.Reg)
			} else if d287.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d287.Reg)
				ctx.UnprotectReg(d287.Reg2)
			}
			ps294 := scm.PhiState{General: ps.General}
			ps294.OverlayValues = make([]scm.JITValueDesc, 294)
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
			ps294.OverlayValues[16] = d16
			ps294.OverlayValues[17] = d17
			ps294.OverlayValues[19] = d19
			ps294.OverlayValues[20] = d20
			ps294.OverlayValues[21] = d21
			ps294.OverlayValues[22] = d22
			ps294.OverlayValues[23] = d23
			ps294.OverlayValues[24] = d24
			ps294.OverlayValues[25] = d25
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
			ps294.OverlayValues[60] = d60
			ps294.OverlayValues[61] = d61
			ps294.OverlayValues[62] = d62
			ps294.OverlayValues[63] = d63
			ps294.OverlayValues[66] = d66
			ps294.OverlayValues[67] = d67
			ps294.OverlayValues[68] = d68
			ps294.OverlayValues[136] = d136
			ps294.OverlayValues[137] = d137
			ps294.OverlayValues[138] = d138
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
			ps294.OverlayValues[167] = d167
			ps294.OverlayValues[168] = d168
			ps294.OverlayValues[169] = d169
			ps294.OverlayValues[170] = d170
			ps294.OverlayValues[171] = d171
			ps294.OverlayValues[172] = d172
			ps294.OverlayValues[173] = d173
			ps294.OverlayValues[174] = d174
			ps294.OverlayValues[175] = d175
			ps294.OverlayValues[178] = d178
			ps294.OverlayValues[286] = d286
			ps294.OverlayValues[287] = d287
			ps294.OverlayValues[288] = d288
			ps294.OverlayValues[289] = d289
			ps294.OverlayValues[290] = d290
			ps294.OverlayValues[291] = d291
			ps294.OverlayValues[292] = d292
			ps294.OverlayValues[293] = d293
			ps294.PhiValues = make([]scm.JITValueDesc, 3)
			d295 = d287
			ps294.PhiValues[0] = d295
			d296 = d2
			ps294.PhiValues[1] = d296
			d297 = d286
			ps294.PhiValues[2] = d297
			if ps294.General && bbs[4].Rendered {
				ctx.EmitJmp(lbl5)
				return result
			}
			return bbs[4].RenderPS(ps294)
			return result
			}
			bbs[4].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
					d298 := ps.PhiValues[0]
					ctx.EnsureDesc(&d298)
					ctx.EmitStoreToStack(d298, int32(bbs[4].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
					d299 := ps.PhiValues[1]
					ctx.EnsureDesc(&d299)
					ctx.EmitStoreToStack(d299, int32(bbs[4].PhiBase)+int32(16))
				}
				if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
					d300 := ps.PhiValues[2]
					ctx.EnsureDesc(&d300)
					ctx.EmitStoreToStack(d300, int32(bbs[4].PhiBase)+int32(32))
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
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
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
			if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != scm.LocNone {
				d173 = ps.OverlayValues[173]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
			}
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
				d178 = ps.OverlayValues[178]
			}
			if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != scm.LocNone {
				d286 = ps.OverlayValues[286]
			}
			if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != scm.LocNone {
				d287 = ps.OverlayValues[287]
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
			if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != scm.LocNone {
				d293 = ps.OverlayValues[293]
			}
			if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
				d295 = ps.OverlayValues[295]
			}
			if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != scm.LocNone {
				d296 = ps.OverlayValues[296]
			}
			if len(ps.OverlayValues) > 297 && ps.OverlayValues[297].Loc != scm.LocNone {
				d297 = ps.OverlayValues[297]
			}
			if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != scm.LocNone {
				d298 = ps.OverlayValues[298]
			}
			if len(ps.OverlayValues) > 299 && ps.OverlayValues[299].Loc != scm.LocNone {
				d299 = ps.OverlayValues[299]
			}
			if len(ps.OverlayValues) > 300 && ps.OverlayValues[300].Loc != scm.LocNone {
				d300 = ps.OverlayValues[300]
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
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d7)
			var d301 scm.JITValueDesc
			if d6.Loc == scm.LocImm && d7.Loc == scm.LocImm {
				d301 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d6.Imm.Int()) == uint64(d7.Imm.Int()))}
			} else if d7.Loc == scm.LocImm {
				r101 := ctx.AllocRegExcept(d6.Reg)
				if d7.Imm.Int() >= -2147483648 && d7.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d6.Reg, int32(d7.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d7.Imm.Int()))
					ctx.EmitCmpInt64(d6.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r101, scm.CcE)
				d301 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r101}
				ctx.BindReg(r101, &d301)
			} else if d6.Loc == scm.LocImm {
				r102 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d6.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d7.Reg)
				ctx.EmitSetcc(r102, scm.CcE)
				d301 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r102}
				ctx.BindReg(r102, &d301)
			} else {
				r103 := ctx.AllocRegExcept(d6.Reg)
				ctx.EmitCmpInt64(d6.Reg, d7.Reg)
				ctx.EmitSetcc(r103, scm.CcE)
				d301 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r103}
				ctx.BindReg(r103, &d301)
			}
			d302 = d301
			ctx.EnsureDesc(&d302)
			if d302.Loc != scm.LocImm && d302.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d302.Loc == scm.LocImm {
				if d302.Imm.Bool() {
			ctx.EnsureDesc(&d6)
			if d6.Loc == scm.LocReg {
				ctx.ProtectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.ProtectReg(d6.Reg)
				ctx.ProtectReg(d6.Reg2)
			}
			d303 = d6
			if d303.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d303)
			d304 = d303
			if d304.Loc == scm.LocImm {
				d304 = scm.JITValueDesc{Loc: scm.LocImm, Type: d304.Type, Imm: scm.NewInt(int64(uint64(d304.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d304.Reg, 32)
				ctx.EmitShrRegImm8(d304.Reg, 32)
			}
			ctx.EmitStoreToStack(d304, int32(bbs[2].PhiBase)+int32(0))
			if d6.Loc == scm.LocReg {
				ctx.UnprotectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d6.Reg)
				ctx.UnprotectReg(d6.Reg2)
			}
			ps305 := scm.PhiState{General: ps.General}
			ps305.OverlayValues = make([]scm.JITValueDesc, 305)
			ps305.OverlayValues[1] = d1
			ps305.OverlayValues[2] = d2
			ps305.OverlayValues[3] = d3
			ps305.OverlayValues[4] = d4
			ps305.OverlayValues[5] = d5
			ps305.OverlayValues[6] = d6
			ps305.OverlayValues[7] = d7
			ps305.OverlayValues[8] = d8
			ps305.OverlayValues[9] = d9
			ps305.OverlayValues[10] = d10
			ps305.OverlayValues[11] = d11
			ps305.OverlayValues[12] = d12
			ps305.OverlayValues[13] = d13
			ps305.OverlayValues[14] = d14
			ps305.OverlayValues[15] = d15
			ps305.OverlayValues[16] = d16
			ps305.OverlayValues[17] = d17
			ps305.OverlayValues[19] = d19
			ps305.OverlayValues[20] = d20
			ps305.OverlayValues[21] = d21
			ps305.OverlayValues[22] = d22
			ps305.OverlayValues[23] = d23
			ps305.OverlayValues[24] = d24
			ps305.OverlayValues[25] = d25
			ps305.OverlayValues[27] = d27
			ps305.OverlayValues[28] = d28
			ps305.OverlayValues[29] = d29
			ps305.OverlayValues[30] = d30
			ps305.OverlayValues[31] = d31
			ps305.OverlayValues[32] = d32
			ps305.OverlayValues[33] = d33
			ps305.OverlayValues[34] = d34
			ps305.OverlayValues[35] = d35
			ps305.OverlayValues[36] = d36
			ps305.OverlayValues[37] = d37
			ps305.OverlayValues[38] = d38
			ps305.OverlayValues[39] = d39
			ps305.OverlayValues[40] = d40
			ps305.OverlayValues[41] = d41
			ps305.OverlayValues[42] = d42
			ps305.OverlayValues[43] = d43
			ps305.OverlayValues[44] = d44
			ps305.OverlayValues[45] = d45
			ps305.OverlayValues[46] = d46
			ps305.OverlayValues[47] = d47
			ps305.OverlayValues[48] = d48
			ps305.OverlayValues[49] = d49
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
			ps305.OverlayValues[66] = d66
			ps305.OverlayValues[67] = d67
			ps305.OverlayValues[68] = d68
			ps305.OverlayValues[136] = d136
			ps305.OverlayValues[137] = d137
			ps305.OverlayValues[138] = d138
			ps305.OverlayValues[140] = d140
			ps305.OverlayValues[141] = d141
			ps305.OverlayValues[142] = d142
			ps305.OverlayValues[143] = d143
			ps305.OverlayValues[144] = d144
			ps305.OverlayValues[145] = d145
			ps305.OverlayValues[146] = d146
			ps305.OverlayValues[147] = d147
			ps305.OverlayValues[148] = d148
			ps305.OverlayValues[149] = d149
			ps305.OverlayValues[150] = d150
			ps305.OverlayValues[151] = d151
			ps305.OverlayValues[152] = d152
			ps305.OverlayValues[153] = d153
			ps305.OverlayValues[154] = d154
			ps305.OverlayValues[155] = d155
			ps305.OverlayValues[156] = d156
			ps305.OverlayValues[157] = d157
			ps305.OverlayValues[158] = d158
			ps305.OverlayValues[159] = d159
			ps305.OverlayValues[160] = d160
			ps305.OverlayValues[161] = d161
			ps305.OverlayValues[162] = d162
			ps305.OverlayValues[163] = d163
			ps305.OverlayValues[164] = d164
			ps305.OverlayValues[165] = d165
			ps305.OverlayValues[166] = d166
			ps305.OverlayValues[167] = d167
			ps305.OverlayValues[168] = d168
			ps305.OverlayValues[169] = d169
			ps305.OverlayValues[170] = d170
			ps305.OverlayValues[171] = d171
			ps305.OverlayValues[172] = d172
			ps305.OverlayValues[173] = d173
			ps305.OverlayValues[174] = d174
			ps305.OverlayValues[175] = d175
			ps305.OverlayValues[178] = d178
			ps305.OverlayValues[286] = d286
			ps305.OverlayValues[287] = d287
			ps305.OverlayValues[288] = d288
			ps305.OverlayValues[289] = d289
			ps305.OverlayValues[290] = d290
			ps305.OverlayValues[291] = d291
			ps305.OverlayValues[292] = d292
			ps305.OverlayValues[293] = d293
			ps305.OverlayValues[295] = d295
			ps305.OverlayValues[296] = d296
			ps305.OverlayValues[297] = d297
			ps305.OverlayValues[298] = d298
			ps305.OverlayValues[299] = d299
			ps305.OverlayValues[300] = d300
			ps305.OverlayValues[301] = d301
			ps305.OverlayValues[302] = d302
			ps305.OverlayValues[303] = d303
			ps305.OverlayValues[304] = d304
			ps305.PhiValues = make([]scm.JITValueDesc, 1)
			d306 = d6
			ps305.PhiValues[0] = d306
					return bbs[2].RenderPS(ps305)
				}
			ps307 := scm.PhiState{General: ps.General}
			ps307.OverlayValues = make([]scm.JITValueDesc, 307)
			ps307.OverlayValues[1] = d1
			ps307.OverlayValues[2] = d2
			ps307.OverlayValues[3] = d3
			ps307.OverlayValues[4] = d4
			ps307.OverlayValues[5] = d5
			ps307.OverlayValues[6] = d6
			ps307.OverlayValues[7] = d7
			ps307.OverlayValues[8] = d8
			ps307.OverlayValues[9] = d9
			ps307.OverlayValues[10] = d10
			ps307.OverlayValues[11] = d11
			ps307.OverlayValues[12] = d12
			ps307.OverlayValues[13] = d13
			ps307.OverlayValues[14] = d14
			ps307.OverlayValues[15] = d15
			ps307.OverlayValues[16] = d16
			ps307.OverlayValues[17] = d17
			ps307.OverlayValues[19] = d19
			ps307.OverlayValues[20] = d20
			ps307.OverlayValues[21] = d21
			ps307.OverlayValues[22] = d22
			ps307.OverlayValues[23] = d23
			ps307.OverlayValues[24] = d24
			ps307.OverlayValues[25] = d25
			ps307.OverlayValues[27] = d27
			ps307.OverlayValues[28] = d28
			ps307.OverlayValues[29] = d29
			ps307.OverlayValues[30] = d30
			ps307.OverlayValues[31] = d31
			ps307.OverlayValues[32] = d32
			ps307.OverlayValues[33] = d33
			ps307.OverlayValues[34] = d34
			ps307.OverlayValues[35] = d35
			ps307.OverlayValues[36] = d36
			ps307.OverlayValues[37] = d37
			ps307.OverlayValues[38] = d38
			ps307.OverlayValues[39] = d39
			ps307.OverlayValues[40] = d40
			ps307.OverlayValues[41] = d41
			ps307.OverlayValues[42] = d42
			ps307.OverlayValues[43] = d43
			ps307.OverlayValues[44] = d44
			ps307.OverlayValues[45] = d45
			ps307.OverlayValues[46] = d46
			ps307.OverlayValues[47] = d47
			ps307.OverlayValues[48] = d48
			ps307.OverlayValues[49] = d49
			ps307.OverlayValues[50] = d50
			ps307.OverlayValues[51] = d51
			ps307.OverlayValues[52] = d52
			ps307.OverlayValues[53] = d53
			ps307.OverlayValues[54] = d54
			ps307.OverlayValues[55] = d55
			ps307.OverlayValues[56] = d56
			ps307.OverlayValues[57] = d57
			ps307.OverlayValues[58] = d58
			ps307.OverlayValues[59] = d59
			ps307.OverlayValues[60] = d60
			ps307.OverlayValues[61] = d61
			ps307.OverlayValues[62] = d62
			ps307.OverlayValues[63] = d63
			ps307.OverlayValues[66] = d66
			ps307.OverlayValues[67] = d67
			ps307.OverlayValues[68] = d68
			ps307.OverlayValues[136] = d136
			ps307.OverlayValues[137] = d137
			ps307.OverlayValues[138] = d138
			ps307.OverlayValues[140] = d140
			ps307.OverlayValues[141] = d141
			ps307.OverlayValues[142] = d142
			ps307.OverlayValues[143] = d143
			ps307.OverlayValues[144] = d144
			ps307.OverlayValues[145] = d145
			ps307.OverlayValues[146] = d146
			ps307.OverlayValues[147] = d147
			ps307.OverlayValues[148] = d148
			ps307.OverlayValues[149] = d149
			ps307.OverlayValues[150] = d150
			ps307.OverlayValues[151] = d151
			ps307.OverlayValues[152] = d152
			ps307.OverlayValues[153] = d153
			ps307.OverlayValues[154] = d154
			ps307.OverlayValues[155] = d155
			ps307.OverlayValues[156] = d156
			ps307.OverlayValues[157] = d157
			ps307.OverlayValues[158] = d158
			ps307.OverlayValues[159] = d159
			ps307.OverlayValues[160] = d160
			ps307.OverlayValues[161] = d161
			ps307.OverlayValues[162] = d162
			ps307.OverlayValues[163] = d163
			ps307.OverlayValues[164] = d164
			ps307.OverlayValues[165] = d165
			ps307.OverlayValues[166] = d166
			ps307.OverlayValues[167] = d167
			ps307.OverlayValues[168] = d168
			ps307.OverlayValues[169] = d169
			ps307.OverlayValues[170] = d170
			ps307.OverlayValues[171] = d171
			ps307.OverlayValues[172] = d172
			ps307.OverlayValues[173] = d173
			ps307.OverlayValues[174] = d174
			ps307.OverlayValues[175] = d175
			ps307.OverlayValues[178] = d178
			ps307.OverlayValues[286] = d286
			ps307.OverlayValues[287] = d287
			ps307.OverlayValues[288] = d288
			ps307.OverlayValues[289] = d289
			ps307.OverlayValues[290] = d290
			ps307.OverlayValues[291] = d291
			ps307.OverlayValues[292] = d292
			ps307.OverlayValues[293] = d293
			ps307.OverlayValues[295] = d295
			ps307.OverlayValues[296] = d296
			ps307.OverlayValues[297] = d297
			ps307.OverlayValues[298] = d298
			ps307.OverlayValues[299] = d299
			ps307.OverlayValues[300] = d300
			ps307.OverlayValues[301] = d301
			ps307.OverlayValues[302] = d302
			ps307.OverlayValues[303] = d303
			ps307.OverlayValues[304] = d304
			ps307.OverlayValues[306] = d306
				return bbs[6].RenderPS(ps307)
			}
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
					d308 := ps.PhiValues[0]
					ctx.EnsureDesc(&d308)
					ctx.EmitStoreToStack(d308, int32(bbs[4].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
					d309 := ps.PhiValues[1]
					ctx.EnsureDesc(&d309)
					ctx.EmitStoreToStack(d309, int32(bbs[4].PhiBase)+int32(16))
				}
				if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
					d310 := ps.PhiValues[2]
					ctx.EnsureDesc(&d310)
					ctx.EmitStoreToStack(d310, int32(bbs[4].PhiBase)+int32(32))
				}
				ps.General = true
				return bbs[4].RenderPS(ps)
			}
			lbl29 := ctx.ReserveLabel()
			lbl30 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d302.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl29)
			ctx.EmitJmp(lbl30)
			ctx.MarkLabel(lbl29)
			ctx.EnsureDesc(&d6)
			if d6.Loc == scm.LocReg {
				ctx.ProtectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.ProtectReg(d6.Reg)
				ctx.ProtectReg(d6.Reg2)
			}
			d311 = d6
			if d311.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d311)
			d312 = d311
			if d312.Loc == scm.LocImm {
				d312 = scm.JITValueDesc{Loc: scm.LocImm, Type: d312.Type, Imm: scm.NewInt(int64(uint64(d312.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d312.Reg, 32)
				ctx.EmitShrRegImm8(d312.Reg, 32)
			}
			ctx.EmitStoreToStack(d312, int32(bbs[2].PhiBase)+int32(0))
			if d6.Loc == scm.LocReg {
				ctx.UnprotectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d6.Reg)
				ctx.UnprotectReg(d6.Reg2)
			}
			ctx.EmitJmp(lbl3)
			ctx.MarkLabel(lbl30)
			ctx.EmitJmp(lbl7)
			ps313 := scm.PhiState{General: true}
			ps313.OverlayValues = make([]scm.JITValueDesc, 313)
			ps313.OverlayValues[1] = d1
			ps313.OverlayValues[2] = d2
			ps313.OverlayValues[3] = d3
			ps313.OverlayValues[4] = d4
			ps313.OverlayValues[5] = d5
			ps313.OverlayValues[6] = d6
			ps313.OverlayValues[7] = d7
			ps313.OverlayValues[8] = d8
			ps313.OverlayValues[9] = d9
			ps313.OverlayValues[10] = d10
			ps313.OverlayValues[11] = d11
			ps313.OverlayValues[12] = d12
			ps313.OverlayValues[13] = d13
			ps313.OverlayValues[14] = d14
			ps313.OverlayValues[15] = d15
			ps313.OverlayValues[16] = d16
			ps313.OverlayValues[17] = d17
			ps313.OverlayValues[19] = d19
			ps313.OverlayValues[20] = d20
			ps313.OverlayValues[21] = d21
			ps313.OverlayValues[22] = d22
			ps313.OverlayValues[23] = d23
			ps313.OverlayValues[24] = d24
			ps313.OverlayValues[25] = d25
			ps313.OverlayValues[27] = d27
			ps313.OverlayValues[28] = d28
			ps313.OverlayValues[29] = d29
			ps313.OverlayValues[30] = d30
			ps313.OverlayValues[31] = d31
			ps313.OverlayValues[32] = d32
			ps313.OverlayValues[33] = d33
			ps313.OverlayValues[34] = d34
			ps313.OverlayValues[35] = d35
			ps313.OverlayValues[36] = d36
			ps313.OverlayValues[37] = d37
			ps313.OverlayValues[38] = d38
			ps313.OverlayValues[39] = d39
			ps313.OverlayValues[40] = d40
			ps313.OverlayValues[41] = d41
			ps313.OverlayValues[42] = d42
			ps313.OverlayValues[43] = d43
			ps313.OverlayValues[44] = d44
			ps313.OverlayValues[45] = d45
			ps313.OverlayValues[46] = d46
			ps313.OverlayValues[47] = d47
			ps313.OverlayValues[48] = d48
			ps313.OverlayValues[49] = d49
			ps313.OverlayValues[50] = d50
			ps313.OverlayValues[51] = d51
			ps313.OverlayValues[52] = d52
			ps313.OverlayValues[53] = d53
			ps313.OverlayValues[54] = d54
			ps313.OverlayValues[55] = d55
			ps313.OverlayValues[56] = d56
			ps313.OverlayValues[57] = d57
			ps313.OverlayValues[58] = d58
			ps313.OverlayValues[59] = d59
			ps313.OverlayValues[60] = d60
			ps313.OverlayValues[61] = d61
			ps313.OverlayValues[62] = d62
			ps313.OverlayValues[63] = d63
			ps313.OverlayValues[66] = d66
			ps313.OverlayValues[67] = d67
			ps313.OverlayValues[68] = d68
			ps313.OverlayValues[136] = d136
			ps313.OverlayValues[137] = d137
			ps313.OverlayValues[138] = d138
			ps313.OverlayValues[140] = d140
			ps313.OverlayValues[141] = d141
			ps313.OverlayValues[142] = d142
			ps313.OverlayValues[143] = d143
			ps313.OverlayValues[144] = d144
			ps313.OverlayValues[145] = d145
			ps313.OverlayValues[146] = d146
			ps313.OverlayValues[147] = d147
			ps313.OverlayValues[148] = d148
			ps313.OverlayValues[149] = d149
			ps313.OverlayValues[150] = d150
			ps313.OverlayValues[151] = d151
			ps313.OverlayValues[152] = d152
			ps313.OverlayValues[153] = d153
			ps313.OverlayValues[154] = d154
			ps313.OverlayValues[155] = d155
			ps313.OverlayValues[156] = d156
			ps313.OverlayValues[157] = d157
			ps313.OverlayValues[158] = d158
			ps313.OverlayValues[159] = d159
			ps313.OverlayValues[160] = d160
			ps313.OverlayValues[161] = d161
			ps313.OverlayValues[162] = d162
			ps313.OverlayValues[163] = d163
			ps313.OverlayValues[164] = d164
			ps313.OverlayValues[165] = d165
			ps313.OverlayValues[166] = d166
			ps313.OverlayValues[167] = d167
			ps313.OverlayValues[168] = d168
			ps313.OverlayValues[169] = d169
			ps313.OverlayValues[170] = d170
			ps313.OverlayValues[171] = d171
			ps313.OverlayValues[172] = d172
			ps313.OverlayValues[173] = d173
			ps313.OverlayValues[174] = d174
			ps313.OverlayValues[175] = d175
			ps313.OverlayValues[178] = d178
			ps313.OverlayValues[286] = d286
			ps313.OverlayValues[287] = d287
			ps313.OverlayValues[288] = d288
			ps313.OverlayValues[289] = d289
			ps313.OverlayValues[290] = d290
			ps313.OverlayValues[291] = d291
			ps313.OverlayValues[292] = d292
			ps313.OverlayValues[293] = d293
			ps313.OverlayValues[295] = d295
			ps313.OverlayValues[296] = d296
			ps313.OverlayValues[297] = d297
			ps313.OverlayValues[298] = d298
			ps313.OverlayValues[299] = d299
			ps313.OverlayValues[300] = d300
			ps313.OverlayValues[301] = d301
			ps313.OverlayValues[302] = d302
			ps313.OverlayValues[303] = d303
			ps313.OverlayValues[304] = d304
			ps313.OverlayValues[306] = d306
			ps313.OverlayValues[308] = d308
			ps313.OverlayValues[309] = d309
			ps313.OverlayValues[310] = d310
			ps313.OverlayValues[311] = d311
			ps313.OverlayValues[312] = d312
			ps313.PhiValues = make([]scm.JITValueDesc, 1)
			d315 = d6
			ps313.PhiValues[0] = d315
			ps314 := scm.PhiState{General: true}
			ps314.OverlayValues = make([]scm.JITValueDesc, 316)
			ps314.OverlayValues[1] = d1
			ps314.OverlayValues[2] = d2
			ps314.OverlayValues[3] = d3
			ps314.OverlayValues[4] = d4
			ps314.OverlayValues[5] = d5
			ps314.OverlayValues[6] = d6
			ps314.OverlayValues[7] = d7
			ps314.OverlayValues[8] = d8
			ps314.OverlayValues[9] = d9
			ps314.OverlayValues[10] = d10
			ps314.OverlayValues[11] = d11
			ps314.OverlayValues[12] = d12
			ps314.OverlayValues[13] = d13
			ps314.OverlayValues[14] = d14
			ps314.OverlayValues[15] = d15
			ps314.OverlayValues[16] = d16
			ps314.OverlayValues[17] = d17
			ps314.OverlayValues[19] = d19
			ps314.OverlayValues[20] = d20
			ps314.OverlayValues[21] = d21
			ps314.OverlayValues[22] = d22
			ps314.OverlayValues[23] = d23
			ps314.OverlayValues[24] = d24
			ps314.OverlayValues[25] = d25
			ps314.OverlayValues[27] = d27
			ps314.OverlayValues[28] = d28
			ps314.OverlayValues[29] = d29
			ps314.OverlayValues[30] = d30
			ps314.OverlayValues[31] = d31
			ps314.OverlayValues[32] = d32
			ps314.OverlayValues[33] = d33
			ps314.OverlayValues[34] = d34
			ps314.OverlayValues[35] = d35
			ps314.OverlayValues[36] = d36
			ps314.OverlayValues[37] = d37
			ps314.OverlayValues[38] = d38
			ps314.OverlayValues[39] = d39
			ps314.OverlayValues[40] = d40
			ps314.OverlayValues[41] = d41
			ps314.OverlayValues[42] = d42
			ps314.OverlayValues[43] = d43
			ps314.OverlayValues[44] = d44
			ps314.OverlayValues[45] = d45
			ps314.OverlayValues[46] = d46
			ps314.OverlayValues[47] = d47
			ps314.OverlayValues[48] = d48
			ps314.OverlayValues[49] = d49
			ps314.OverlayValues[50] = d50
			ps314.OverlayValues[51] = d51
			ps314.OverlayValues[52] = d52
			ps314.OverlayValues[53] = d53
			ps314.OverlayValues[54] = d54
			ps314.OverlayValues[55] = d55
			ps314.OverlayValues[56] = d56
			ps314.OverlayValues[57] = d57
			ps314.OverlayValues[58] = d58
			ps314.OverlayValues[59] = d59
			ps314.OverlayValues[60] = d60
			ps314.OverlayValues[61] = d61
			ps314.OverlayValues[62] = d62
			ps314.OverlayValues[63] = d63
			ps314.OverlayValues[66] = d66
			ps314.OverlayValues[67] = d67
			ps314.OverlayValues[68] = d68
			ps314.OverlayValues[136] = d136
			ps314.OverlayValues[137] = d137
			ps314.OverlayValues[138] = d138
			ps314.OverlayValues[140] = d140
			ps314.OverlayValues[141] = d141
			ps314.OverlayValues[142] = d142
			ps314.OverlayValues[143] = d143
			ps314.OverlayValues[144] = d144
			ps314.OverlayValues[145] = d145
			ps314.OverlayValues[146] = d146
			ps314.OverlayValues[147] = d147
			ps314.OverlayValues[148] = d148
			ps314.OverlayValues[149] = d149
			ps314.OverlayValues[150] = d150
			ps314.OverlayValues[151] = d151
			ps314.OverlayValues[152] = d152
			ps314.OverlayValues[153] = d153
			ps314.OverlayValues[154] = d154
			ps314.OverlayValues[155] = d155
			ps314.OverlayValues[156] = d156
			ps314.OverlayValues[157] = d157
			ps314.OverlayValues[158] = d158
			ps314.OverlayValues[159] = d159
			ps314.OverlayValues[160] = d160
			ps314.OverlayValues[161] = d161
			ps314.OverlayValues[162] = d162
			ps314.OverlayValues[163] = d163
			ps314.OverlayValues[164] = d164
			ps314.OverlayValues[165] = d165
			ps314.OverlayValues[166] = d166
			ps314.OverlayValues[167] = d167
			ps314.OverlayValues[168] = d168
			ps314.OverlayValues[169] = d169
			ps314.OverlayValues[170] = d170
			ps314.OverlayValues[171] = d171
			ps314.OverlayValues[172] = d172
			ps314.OverlayValues[173] = d173
			ps314.OverlayValues[174] = d174
			ps314.OverlayValues[175] = d175
			ps314.OverlayValues[178] = d178
			ps314.OverlayValues[286] = d286
			ps314.OverlayValues[287] = d287
			ps314.OverlayValues[288] = d288
			ps314.OverlayValues[289] = d289
			ps314.OverlayValues[290] = d290
			ps314.OverlayValues[291] = d291
			ps314.OverlayValues[292] = d292
			ps314.OverlayValues[293] = d293
			ps314.OverlayValues[295] = d295
			ps314.OverlayValues[296] = d296
			ps314.OverlayValues[297] = d297
			ps314.OverlayValues[298] = d298
			ps314.OverlayValues[299] = d299
			ps314.OverlayValues[300] = d300
			ps314.OverlayValues[301] = d301
			ps314.OverlayValues[302] = d302
			ps314.OverlayValues[303] = d303
			ps314.OverlayValues[304] = d304
			ps314.OverlayValues[306] = d306
			ps314.OverlayValues[308] = d308
			ps314.OverlayValues[309] = d309
			ps314.OverlayValues[310] = d310
			ps314.OverlayValues[311] = d311
			ps314.OverlayValues[312] = d312
			ps314.OverlayValues[315] = d315
			snap316 := d1
			snap317 := d2
			snap318 := d3
			snap319 := d4
			snap320 := d5
			snap321 := d6
			snap322 := d7
			snap323 := d8
			snap324 := d9
			snap325 := d10
			snap326 := d11
			snap327 := d12
			snap328 := d13
			snap329 := d14
			snap330 := d15
			snap331 := d16
			snap332 := d17
			snap333 := d19
			snap334 := d20
			snap335 := d21
			snap336 := d22
			snap337 := d23
			snap338 := d24
			snap339 := d25
			snap340 := d27
			snap341 := d28
			snap342 := d29
			snap343 := d30
			snap344 := d31
			snap345 := d32
			snap346 := d33
			snap347 := d34
			snap348 := d35
			snap349 := d36
			snap350 := d37
			snap351 := d38
			snap352 := d39
			snap353 := d40
			snap354 := d41
			snap355 := d42
			snap356 := d43
			snap357 := d44
			snap358 := d45
			snap359 := d46
			snap360 := d47
			snap361 := d48
			snap362 := d49
			snap363 := d50
			snap364 := d51
			snap365 := d52
			snap366 := d53
			snap367 := d54
			snap368 := d55
			snap369 := d56
			snap370 := d57
			snap371 := d58
			snap372 := d59
			snap373 := d60
			snap374 := d61
			snap375 := d62
			snap376 := d63
			snap377 := d66
			snap378 := d67
			snap379 := d68
			snap380 := d136
			snap381 := d137
			snap382 := d138
			snap383 := d140
			snap384 := d141
			snap385 := d142
			snap386 := d143
			snap387 := d144
			snap388 := d145
			snap389 := d146
			snap390 := d147
			snap391 := d148
			snap392 := d149
			snap393 := d150
			snap394 := d151
			snap395 := d152
			snap396 := d153
			snap397 := d154
			snap398 := d155
			snap399 := d156
			snap400 := d157
			snap401 := d158
			snap402 := d159
			snap403 := d160
			snap404 := d161
			snap405 := d162
			snap406 := d163
			snap407 := d164
			snap408 := d165
			snap409 := d166
			snap410 := d167
			snap411 := d168
			snap412 := d169
			snap413 := d170
			snap414 := d171
			snap415 := d172
			snap416 := d173
			snap417 := d174
			snap418 := d175
			snap419 := d178
			snap420 := d286
			snap421 := d287
			snap422 := d288
			snap423 := d289
			snap424 := d290
			snap425 := d291
			snap426 := d292
			snap427 := d293
			snap428 := d295
			snap429 := d296
			snap430 := d297
			snap431 := d298
			snap432 := d299
			snap433 := d300
			snap434 := d301
			snap435 := d302
			snap436 := d303
			snap437 := d304
			snap438 := d306
			snap439 := d308
			snap440 := d309
			snap441 := d310
			snap442 := d311
			snap443 := d312
			snap444 := d315
			alloc445 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps313)
			}
			ctx.RestoreAllocState(alloc445)
			d1 = snap316
			d2 = snap317
			d3 = snap318
			d4 = snap319
			d5 = snap320
			d6 = snap321
			d7 = snap322
			d8 = snap323
			d9 = snap324
			d10 = snap325
			d11 = snap326
			d12 = snap327
			d13 = snap328
			d14 = snap329
			d15 = snap330
			d16 = snap331
			d17 = snap332
			d19 = snap333
			d20 = snap334
			d21 = snap335
			d22 = snap336
			d23 = snap337
			d24 = snap338
			d25 = snap339
			d27 = snap340
			d28 = snap341
			d29 = snap342
			d30 = snap343
			d31 = snap344
			d32 = snap345
			d33 = snap346
			d34 = snap347
			d35 = snap348
			d36 = snap349
			d37 = snap350
			d38 = snap351
			d39 = snap352
			d40 = snap353
			d41 = snap354
			d42 = snap355
			d43 = snap356
			d44 = snap357
			d45 = snap358
			d46 = snap359
			d47 = snap360
			d48 = snap361
			d49 = snap362
			d50 = snap363
			d51 = snap364
			d52 = snap365
			d53 = snap366
			d54 = snap367
			d55 = snap368
			d56 = snap369
			d57 = snap370
			d58 = snap371
			d59 = snap372
			d60 = snap373
			d61 = snap374
			d62 = snap375
			d63 = snap376
			d66 = snap377
			d67 = snap378
			d68 = snap379
			d136 = snap380
			d137 = snap381
			d138 = snap382
			d140 = snap383
			d141 = snap384
			d142 = snap385
			d143 = snap386
			d144 = snap387
			d145 = snap388
			d146 = snap389
			d147 = snap390
			d148 = snap391
			d149 = snap392
			d150 = snap393
			d151 = snap394
			d152 = snap395
			d153 = snap396
			d154 = snap397
			d155 = snap398
			d156 = snap399
			d157 = snap400
			d158 = snap401
			d159 = snap402
			d160 = snap403
			d161 = snap404
			d162 = snap405
			d163 = snap406
			d164 = snap407
			d165 = snap408
			d166 = snap409
			d167 = snap410
			d168 = snap411
			d169 = snap412
			d170 = snap413
			d171 = snap414
			d172 = snap415
			d173 = snap416
			d174 = snap417
			d175 = snap418
			d178 = snap419
			d286 = snap420
			d287 = snap421
			d288 = snap422
			d289 = snap423
			d290 = snap424
			d291 = snap425
			d292 = snap426
			d293 = snap427
			d295 = snap428
			d296 = snap429
			d297 = snap430
			d298 = snap431
			d299 = snap432
			d300 = snap433
			d301 = snap434
			d302 = snap435
			d303 = snap436
			d304 = snap437
			d306 = snap438
			d308 = snap439
			d309 = snap440
			d310 = snap441
			d311 = snap442
			d312 = snap443
			d315 = snap444
			if !bbs[6].Rendered {
				return bbs[6].RenderPS(ps314)
			}
			return result
			ctx.FreeDesc(&d301)
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
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
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
			if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != scm.LocNone {
				d173 = ps.OverlayValues[173]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
			}
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
				d178 = ps.OverlayValues[178]
			}
			if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != scm.LocNone {
				d286 = ps.OverlayValues[286]
			}
			if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != scm.LocNone {
				d287 = ps.OverlayValues[287]
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
			if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != scm.LocNone {
				d293 = ps.OverlayValues[293]
			}
			if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
				d295 = ps.OverlayValues[295]
			}
			if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != scm.LocNone {
				d296 = ps.OverlayValues[296]
			}
			if len(ps.OverlayValues) > 297 && ps.OverlayValues[297].Loc != scm.LocNone {
				d297 = ps.OverlayValues[297]
			}
			if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != scm.LocNone {
				d298 = ps.OverlayValues[298]
			}
			if len(ps.OverlayValues) > 299 && ps.OverlayValues[299].Loc != scm.LocNone {
				d299 = ps.OverlayValues[299]
			}
			if len(ps.OverlayValues) > 300 && ps.OverlayValues[300].Loc != scm.LocNone {
				d300 = ps.OverlayValues[300]
			}
			if len(ps.OverlayValues) > 301 && ps.OverlayValues[301].Loc != scm.LocNone {
				d301 = ps.OverlayValues[301]
			}
			if len(ps.OverlayValues) > 302 && ps.OverlayValues[302].Loc != scm.LocNone {
				d302 = ps.OverlayValues[302]
			}
			if len(ps.OverlayValues) > 303 && ps.OverlayValues[303].Loc != scm.LocNone {
				d303 = ps.OverlayValues[303]
			}
			if len(ps.OverlayValues) > 304 && ps.OverlayValues[304].Loc != scm.LocNone {
				d304 = ps.OverlayValues[304]
			}
			if len(ps.OverlayValues) > 306 && ps.OverlayValues[306].Loc != scm.LocNone {
				d306 = ps.OverlayValues[306]
			}
			if len(ps.OverlayValues) > 308 && ps.OverlayValues[308].Loc != scm.LocNone {
				d308 = ps.OverlayValues[308]
			}
			if len(ps.OverlayValues) > 309 && ps.OverlayValues[309].Loc != scm.LocNone {
				d309 = ps.OverlayValues[309]
			}
			if len(ps.OverlayValues) > 310 && ps.OverlayValues[310].Loc != scm.LocNone {
				d310 = ps.OverlayValues[310]
			}
			if len(ps.OverlayValues) > 311 && ps.OverlayValues[311].Loc != scm.LocNone {
				d311 = ps.OverlayValues[311]
			}
			if len(ps.OverlayValues) > 312 && ps.OverlayValues[312].Loc != scm.LocNone {
				d312 = ps.OverlayValues[312]
			}
			if len(ps.OverlayValues) > 315 && ps.OverlayValues[315].Loc != scm.LocNone {
				d315 = ps.OverlayValues[315]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			var d446 scm.JITValueDesc
			if d1.Loc == scm.LocImm {
				d446 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.EmitMovRegReg(scratch, d1.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d446 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d446)
			}
			if d446.Loc == scm.LocImm {
				d446 = scm.JITValueDesc{Loc: scm.LocImm, Type: d446.Type, Imm: scm.NewInt(int64(uint64(d446.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d446.Reg, 32)
				ctx.EmitShrRegImm8(d446.Reg, 32)
			}
			if d446.Loc == scm.LocReg && d1.Loc == scm.LocReg && d446.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d1)
			if d1.Loc == scm.LocReg {
				ctx.ProtectReg(d1.Reg)
			} else if d1.Loc == scm.LocRegPair {
				ctx.ProtectReg(d1.Reg)
				ctx.ProtectReg(d1.Reg2)
			}
			ctx.EnsureDesc(&d3)
			if d3.Loc == scm.LocReg {
				ctx.ProtectReg(d3.Reg)
			} else if d3.Loc == scm.LocRegPair {
				ctx.ProtectReg(d3.Reg)
				ctx.ProtectReg(d3.Reg2)
			}
			ctx.EnsureDesc(&d446)
			if d446.Loc == scm.LocReg {
				ctx.ProtectReg(d446.Reg)
			} else if d446.Loc == scm.LocRegPair {
				ctx.ProtectReg(d446.Reg)
				ctx.ProtectReg(d446.Reg2)
			}
			d447 = d446
			if d447.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d447)
			d448 = d447
			if d448.Loc == scm.LocImm {
				d448 = scm.JITValueDesc{Loc: scm.LocImm, Type: d448.Type, Imm: scm.NewInt(int64(uint64(d448.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d448.Reg, 32)
				ctx.EmitShrRegImm8(d448.Reg, 32)
			}
			ctx.EmitStoreToStack(d448, int32(bbs[4].PhiBase)+int32(0))
			d449 = d1
			if d449.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d449)
			d450 = d449
			if d450.Loc == scm.LocImm {
				d450 = scm.JITValueDesc{Loc: scm.LocImm, Type: d450.Type, Imm: scm.NewInt(int64(uint64(d450.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d450.Reg, 32)
				ctx.EmitShrRegImm8(d450.Reg, 32)
			}
			ctx.EmitStoreToStack(d450, int32(bbs[4].PhiBase)+int32(16))
			d451 = d3
			if d451.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d451)
			d452 = d451
			if d452.Loc == scm.LocImm {
				d452 = scm.JITValueDesc{Loc: scm.LocImm, Type: d452.Type, Imm: scm.NewInt(int64(uint64(d452.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d452.Reg, 32)
				ctx.EmitShrRegImm8(d452.Reg, 32)
			}
			ctx.EmitStoreToStack(d452, int32(bbs[4].PhiBase)+int32(32))
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
			if d446.Loc == scm.LocReg {
				ctx.UnprotectReg(d446.Reg)
			} else if d446.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d446.Reg)
				ctx.UnprotectReg(d446.Reg2)
			}
			ps453 := scm.PhiState{General: ps.General}
			ps453.OverlayValues = make([]scm.JITValueDesc, 453)
			ps453.OverlayValues[1] = d1
			ps453.OverlayValues[2] = d2
			ps453.OverlayValues[3] = d3
			ps453.OverlayValues[4] = d4
			ps453.OverlayValues[5] = d5
			ps453.OverlayValues[6] = d6
			ps453.OverlayValues[7] = d7
			ps453.OverlayValues[8] = d8
			ps453.OverlayValues[9] = d9
			ps453.OverlayValues[10] = d10
			ps453.OverlayValues[11] = d11
			ps453.OverlayValues[12] = d12
			ps453.OverlayValues[13] = d13
			ps453.OverlayValues[14] = d14
			ps453.OverlayValues[15] = d15
			ps453.OverlayValues[16] = d16
			ps453.OverlayValues[17] = d17
			ps453.OverlayValues[19] = d19
			ps453.OverlayValues[20] = d20
			ps453.OverlayValues[21] = d21
			ps453.OverlayValues[22] = d22
			ps453.OverlayValues[23] = d23
			ps453.OverlayValues[24] = d24
			ps453.OverlayValues[25] = d25
			ps453.OverlayValues[27] = d27
			ps453.OverlayValues[28] = d28
			ps453.OverlayValues[29] = d29
			ps453.OverlayValues[30] = d30
			ps453.OverlayValues[31] = d31
			ps453.OverlayValues[32] = d32
			ps453.OverlayValues[33] = d33
			ps453.OverlayValues[34] = d34
			ps453.OverlayValues[35] = d35
			ps453.OverlayValues[36] = d36
			ps453.OverlayValues[37] = d37
			ps453.OverlayValues[38] = d38
			ps453.OverlayValues[39] = d39
			ps453.OverlayValues[40] = d40
			ps453.OverlayValues[41] = d41
			ps453.OverlayValues[42] = d42
			ps453.OverlayValues[43] = d43
			ps453.OverlayValues[44] = d44
			ps453.OverlayValues[45] = d45
			ps453.OverlayValues[46] = d46
			ps453.OverlayValues[47] = d47
			ps453.OverlayValues[48] = d48
			ps453.OverlayValues[49] = d49
			ps453.OverlayValues[50] = d50
			ps453.OverlayValues[51] = d51
			ps453.OverlayValues[52] = d52
			ps453.OverlayValues[53] = d53
			ps453.OverlayValues[54] = d54
			ps453.OverlayValues[55] = d55
			ps453.OverlayValues[56] = d56
			ps453.OverlayValues[57] = d57
			ps453.OverlayValues[58] = d58
			ps453.OverlayValues[59] = d59
			ps453.OverlayValues[60] = d60
			ps453.OverlayValues[61] = d61
			ps453.OverlayValues[62] = d62
			ps453.OverlayValues[63] = d63
			ps453.OverlayValues[66] = d66
			ps453.OverlayValues[67] = d67
			ps453.OverlayValues[68] = d68
			ps453.OverlayValues[136] = d136
			ps453.OverlayValues[137] = d137
			ps453.OverlayValues[138] = d138
			ps453.OverlayValues[140] = d140
			ps453.OverlayValues[141] = d141
			ps453.OverlayValues[142] = d142
			ps453.OverlayValues[143] = d143
			ps453.OverlayValues[144] = d144
			ps453.OverlayValues[145] = d145
			ps453.OverlayValues[146] = d146
			ps453.OverlayValues[147] = d147
			ps453.OverlayValues[148] = d148
			ps453.OverlayValues[149] = d149
			ps453.OverlayValues[150] = d150
			ps453.OverlayValues[151] = d151
			ps453.OverlayValues[152] = d152
			ps453.OverlayValues[153] = d153
			ps453.OverlayValues[154] = d154
			ps453.OverlayValues[155] = d155
			ps453.OverlayValues[156] = d156
			ps453.OverlayValues[157] = d157
			ps453.OverlayValues[158] = d158
			ps453.OverlayValues[159] = d159
			ps453.OverlayValues[160] = d160
			ps453.OverlayValues[161] = d161
			ps453.OverlayValues[162] = d162
			ps453.OverlayValues[163] = d163
			ps453.OverlayValues[164] = d164
			ps453.OverlayValues[165] = d165
			ps453.OverlayValues[166] = d166
			ps453.OverlayValues[167] = d167
			ps453.OverlayValues[168] = d168
			ps453.OverlayValues[169] = d169
			ps453.OverlayValues[170] = d170
			ps453.OverlayValues[171] = d171
			ps453.OverlayValues[172] = d172
			ps453.OverlayValues[173] = d173
			ps453.OverlayValues[174] = d174
			ps453.OverlayValues[175] = d175
			ps453.OverlayValues[178] = d178
			ps453.OverlayValues[286] = d286
			ps453.OverlayValues[287] = d287
			ps453.OverlayValues[288] = d288
			ps453.OverlayValues[289] = d289
			ps453.OverlayValues[290] = d290
			ps453.OverlayValues[291] = d291
			ps453.OverlayValues[292] = d292
			ps453.OverlayValues[293] = d293
			ps453.OverlayValues[295] = d295
			ps453.OverlayValues[296] = d296
			ps453.OverlayValues[297] = d297
			ps453.OverlayValues[298] = d298
			ps453.OverlayValues[299] = d299
			ps453.OverlayValues[300] = d300
			ps453.OverlayValues[301] = d301
			ps453.OverlayValues[302] = d302
			ps453.OverlayValues[303] = d303
			ps453.OverlayValues[304] = d304
			ps453.OverlayValues[306] = d306
			ps453.OverlayValues[308] = d308
			ps453.OverlayValues[309] = d309
			ps453.OverlayValues[310] = d310
			ps453.OverlayValues[311] = d311
			ps453.OverlayValues[312] = d312
			ps453.OverlayValues[315] = d315
			ps453.OverlayValues[446] = d446
			ps453.OverlayValues[447] = d447
			ps453.OverlayValues[448] = d448
			ps453.OverlayValues[449] = d449
			ps453.OverlayValues[450] = d450
			ps453.OverlayValues[451] = d451
			ps453.OverlayValues[452] = d452
			ps453.PhiValues = make([]scm.JITValueDesc, 3)
			d454 = d446
			ps453.PhiValues[0] = d454
			d455 = d1
			ps453.PhiValues[1] = d455
			d456 = d3
			ps453.PhiValues[2] = d456
			if ps453.General && bbs[4].Rendered {
				ctx.EmitJmp(lbl5)
				return result
			}
			return bbs[4].RenderPS(ps453)
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
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
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
			if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != scm.LocNone {
				d173 = ps.OverlayValues[173]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
			}
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
				d178 = ps.OverlayValues[178]
			}
			if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != scm.LocNone {
				d286 = ps.OverlayValues[286]
			}
			if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != scm.LocNone {
				d287 = ps.OverlayValues[287]
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
			if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != scm.LocNone {
				d293 = ps.OverlayValues[293]
			}
			if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
				d295 = ps.OverlayValues[295]
			}
			if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != scm.LocNone {
				d296 = ps.OverlayValues[296]
			}
			if len(ps.OverlayValues) > 297 && ps.OverlayValues[297].Loc != scm.LocNone {
				d297 = ps.OverlayValues[297]
			}
			if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != scm.LocNone {
				d298 = ps.OverlayValues[298]
			}
			if len(ps.OverlayValues) > 299 && ps.OverlayValues[299].Loc != scm.LocNone {
				d299 = ps.OverlayValues[299]
			}
			if len(ps.OverlayValues) > 300 && ps.OverlayValues[300].Loc != scm.LocNone {
				d300 = ps.OverlayValues[300]
			}
			if len(ps.OverlayValues) > 301 && ps.OverlayValues[301].Loc != scm.LocNone {
				d301 = ps.OverlayValues[301]
			}
			if len(ps.OverlayValues) > 302 && ps.OverlayValues[302].Loc != scm.LocNone {
				d302 = ps.OverlayValues[302]
			}
			if len(ps.OverlayValues) > 303 && ps.OverlayValues[303].Loc != scm.LocNone {
				d303 = ps.OverlayValues[303]
			}
			if len(ps.OverlayValues) > 304 && ps.OverlayValues[304].Loc != scm.LocNone {
				d304 = ps.OverlayValues[304]
			}
			if len(ps.OverlayValues) > 306 && ps.OverlayValues[306].Loc != scm.LocNone {
				d306 = ps.OverlayValues[306]
			}
			if len(ps.OverlayValues) > 308 && ps.OverlayValues[308].Loc != scm.LocNone {
				d308 = ps.OverlayValues[308]
			}
			if len(ps.OverlayValues) > 309 && ps.OverlayValues[309].Loc != scm.LocNone {
				d309 = ps.OverlayValues[309]
			}
			if len(ps.OverlayValues) > 310 && ps.OverlayValues[310].Loc != scm.LocNone {
				d310 = ps.OverlayValues[310]
			}
			if len(ps.OverlayValues) > 311 && ps.OverlayValues[311].Loc != scm.LocNone {
				d311 = ps.OverlayValues[311]
			}
			if len(ps.OverlayValues) > 312 && ps.OverlayValues[312].Loc != scm.LocNone {
				d312 = ps.OverlayValues[312]
			}
			if len(ps.OverlayValues) > 315 && ps.OverlayValues[315].Loc != scm.LocNone {
				d315 = ps.OverlayValues[315]
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
			if len(ps.OverlayValues) > 454 && ps.OverlayValues[454].Loc != scm.LocNone {
				d454 = ps.OverlayValues[454]
			}
			if len(ps.OverlayValues) > 455 && ps.OverlayValues[455].Loc != scm.LocNone {
				d455 = ps.OverlayValues[455]
			}
			if len(ps.OverlayValues) > 456 && ps.OverlayValues[456].Loc != scm.LocNone {
				d456 = ps.OverlayValues[456]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d5)
			d457 = d5
			_ = d457
			r104 := d5.Loc == scm.LocReg
			r105 := d5.Reg
			if r104 { ctx.ProtectReg(r105) }
			phiBase458 := ctx.AllocStack(int32(16))
			d459 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase458)+int32(176)}
			lbl31 := ctx.ReserveLabel()
			bbpos_3_0 := int32(-1)
			_ = bbpos_3_0
			bbpos_3_1 := int32(-1)
			_ = bbpos_3_1
			bbpos_3_2 := int32(-1)
			_ = bbpos_3_2
			bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d459 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(176)}
			ctx.EnsureDesc(&d457)
			ctx.EnsureDesc(&d457)
			var d460 scm.JITValueDesc
			if d457.Loc == scm.LocImm {
				d460 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d457.Imm.Int()))))}
			} else {
				r106 := ctx.AllocReg()
				ctx.EmitMovRegReg(r106, d457.Reg)
				ctx.EmitShlRegImm8(r106, 32)
				ctx.EmitShrRegImm8(r106, 32)
				d460 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r106}
				ctx.BindReg(r106, &d460)
			}
			var d461 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				r107 := ctx.AllocReg()
				ctx.EmitMovRegMem8(r107, fieldAddr)
				d461 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r107}
				ctx.BindReg(r107, &d461)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r108 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r108, thisptr.Reg, off)
				d461 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r108}
				ctx.BindReg(r108, &d461)
			}
			ctx.EnsureDesc(&d461)
			ctx.EnsureDesc(&d461)
			var d462 scm.JITValueDesc
			if d461.Loc == scm.LocImm {
				d462 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d461.Imm.Int()))))}
			} else {
				r109 := ctx.AllocReg()
				ctx.EmitMovRegReg(r109, d461.Reg)
				ctx.EmitShlRegImm8(r109, 56)
				ctx.EmitShrRegImm8(r109, 56)
				d462 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r109}
				ctx.BindReg(r109, &d462)
			}
			ctx.EnsureDesc(&d460)
			ctx.EnsureDesc(&d462)
			ctx.EnsureDesc(&d460)
			ctx.ProtectReg(d460.Reg)
			ctx.EnsureDesc(&d462)
			ctx.UnprotectReg(d460.Reg)
			var d463 scm.JITValueDesc
			if d460.Loc == scm.LocImm && d462.Loc == scm.LocImm {
				d463 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d460.Imm.Int() * d462.Imm.Int())}
			} else if d460.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d462.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d460.Imm.Int()))
				ctx.EmitImulInt64(scratch, d462.Reg)
				d463 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d463)
			} else if d462.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d460.Reg)
				ctx.EmitMovRegReg(scratch, d460.Reg)
				if d462.Imm.Int() >= -2147483648 && d462.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d462.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d462.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d463 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d463)
			} else {
				r110 := ctx.AllocRegExcept(d460.Reg, d462.Reg)
				ctx.EmitMovRegReg(r110, d460.Reg)
				ctx.EmitImulInt64(r110, d462.Reg)
				d463 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
				ctx.BindReg(r110, &d463)
			}
			if d463.Loc == scm.LocReg && d460.Loc == scm.LocReg && d463.Reg == d460.Reg {
				ctx.TransferReg(d460.Reg)
				d460.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d460)
			ctx.FreeDesc(&d462)
			var d464 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				r111 := ctx.AllocReg()
				r112 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r111, fieldAddr)
				ctx.EmitMovRegMem64(r112, fieldAddr+8)
				d464 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r111, Reg2: r112}
				ctx.BindReg(r111, &d464)
				ctx.BindReg(r112, &d464)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				r113 := ctx.AllocReg()
				r114 := ctx.AllocReg()
				ctx.EmitMovRegMem(r113, thisptr.Reg, off)
				ctx.EmitMovRegMem(r114, thisptr.Reg, off+8)
				d464 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r113, Reg2: r114}
				ctx.BindReg(r113, &d464)
				ctx.BindReg(r114, &d464)
			}
			ctx.EnsureDesc(&d463)
			var d465 scm.JITValueDesc
			if d463.Loc == scm.LocImm {
				d465 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d463.Imm.Int() / 64)}
			} else {
				r115 := ctx.AllocRegExcept(d463.Reg)
				ctx.EmitMovRegReg(r115, d463.Reg)
				ctx.EmitShrRegImm8(r115, 6)
				d465 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
				ctx.BindReg(r115, &d465)
			}
			if d465.Loc == scm.LocReg && d463.Loc == scm.LocReg && d465.Reg == d463.Reg {
				ctx.TransferReg(d463.Reg)
				d463.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d465)
			r116 := ctx.AllocReg()
			ctx.EnsureDesc(&d465)
			ctx.EnsureDesc(&d464)
			if d465.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r116, uint64(d465.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r116, d465.Reg)
				ctx.EmitShlRegImm8(r116, 3)
			}
			if d464.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d464.Imm.Int()))
				ctx.EmitAddInt64(r116, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r116, d464.Reg)
			}
			r117 := ctx.AllocRegExcept(r116)
			ctx.EmitMovRegMem(r117, r116, 0)
			ctx.FreeReg(r116)
			d466 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r117}
			ctx.BindReg(r117, &d466)
			ctx.FreeDesc(&d465)
			ctx.EnsureDesc(&d463)
			var d467 scm.JITValueDesc
			if d463.Loc == scm.LocImm {
				d467 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d463.Imm.Int() % 64)}
			} else {
				r118 := ctx.AllocRegExcept(d463.Reg)
				ctx.EmitMovRegReg(r118, d463.Reg)
				ctx.EmitAndRegImm32(r118, 63)
				d467 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r118}
				ctx.BindReg(r118, &d467)
			}
			if d467.Loc == scm.LocReg && d463.Loc == scm.LocReg && d467.Reg == d463.Reg {
				ctx.TransferReg(d463.Reg)
				d463.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d466)
			ctx.EnsureDesc(&d467)
			var d468 scm.JITValueDesc
			if d466.Loc == scm.LocImm && d467.Loc == scm.LocImm {
				d468 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d466.Imm.Int()) << uint64(d467.Imm.Int())))}
			} else if d467.Loc == scm.LocImm {
				r119 := ctx.AllocRegExcept(d466.Reg)
				ctx.EmitMovRegReg(r119, d466.Reg)
				ctx.EmitShlRegImm8(r119, uint8(d467.Imm.Int()))
				d468 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
				ctx.BindReg(r119, &d468)
			} else {
				{
					shiftSrc := d466.Reg
					r120 := ctx.AllocRegExcept(d466.Reg)
					ctx.EmitMovRegReg(r120, d466.Reg)
					shiftSrc = r120
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d467.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d467.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d467.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d468 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d468)
				}
			}
			if d468.Loc == scm.LocReg && d466.Loc == scm.LocReg && d468.Reg == d466.Reg {
				ctx.TransferReg(d466.Reg)
				d466.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d466)
			ctx.FreeDesc(&d467)
			ctx.EnsureDesc(&d463)
			var d469 scm.JITValueDesc
			if d463.Loc == scm.LocImm {
				d469 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d463.Imm.Int() % 64)}
			} else {
				r121 := ctx.AllocRegExcept(d463.Reg)
				ctx.EmitMovRegReg(r121, d463.Reg)
				ctx.EmitAndRegImm32(r121, 63)
				d469 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r121}
				ctx.BindReg(r121, &d469)
			}
			if d469.Loc == scm.LocReg && d463.Loc == scm.LocReg && d469.Reg == d463.Reg {
				ctx.TransferReg(d463.Reg)
				d463.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d461)
			ctx.EnsureDesc(&d461)
			var d470 scm.JITValueDesc
			if d461.Loc == scm.LocImm {
				d470 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d461.Imm.Int()))))}
			} else {
				r122 := ctx.AllocReg()
				ctx.EmitMovRegReg(r122, d461.Reg)
				ctx.EmitShlRegImm8(r122, 56)
				ctx.EmitShrRegImm8(r122, 56)
				d470 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
				ctx.BindReg(r122, &d470)
			}
			ctx.EnsureDesc(&d469)
			ctx.EnsureDesc(&d470)
			ctx.EnsureDesc(&d469)
			ctx.ProtectReg(d469.Reg)
			ctx.EnsureDesc(&d470)
			ctx.UnprotectReg(d469.Reg)
			var d471 scm.JITValueDesc
			if d469.Loc == scm.LocImm && d470.Loc == scm.LocImm {
				d471 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d469.Imm.Int() + d470.Imm.Int())}
			} else if d470.Loc == scm.LocImm && d470.Imm.Int() == 0 {
				r123 := ctx.AllocRegExcept(d469.Reg)
				ctx.EmitMovRegReg(r123, d469.Reg)
				d471 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
				ctx.BindReg(r123, &d471)
			} else if d469.Loc == scm.LocImm && d469.Imm.Int() == 0 {
				d471 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d470.Reg}
				ctx.BindReg(d470.Reg, &d471)
			} else if d469.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d470.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d469.Imm.Int()))
				ctx.EmitAddInt64(scratch, d470.Reg)
				d471 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d471)
			} else if d470.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d469.Reg)
				ctx.EmitMovRegReg(scratch, d469.Reg)
				if d470.Imm.Int() >= -2147483648 && d470.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d470.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d470.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d471 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d471)
			} else {
				r124 := ctx.AllocRegExcept(d469.Reg, d470.Reg)
				ctx.EmitMovRegReg(r124, d469.Reg)
				ctx.EmitAddInt64(r124, d470.Reg)
				d471 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
				ctx.BindReg(r124, &d471)
			}
			if d471.Loc == scm.LocReg && d469.Loc == scm.LocReg && d471.Reg == d469.Reg {
				ctx.TransferReg(d469.Reg)
				d469.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d469)
			ctx.FreeDesc(&d470)
			ctx.EnsureDesc(&d471)
			var d472 scm.JITValueDesc
			if d471.Loc == scm.LocImm {
				d472 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d471.Imm.Int()) > uint64(64))}
			} else {
				r125 := ctx.AllocRegExcept(d471.Reg)
				ctx.EmitCmpRegImm32(d471.Reg, 64)
				ctx.EmitSetcc(r125, scm.CcA)
				d472 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r125}
				ctx.BindReg(r125, &d472)
			}
			ctx.FreeDesc(&d471)
			d473 = d472
			ctx.EnsureDesc(&d473)
			if d473.Loc != scm.LocImm && d473.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl32 := ctx.ReserveLabel()
			lbl33 := ctx.ReserveLabel()
			lbl34 := ctx.ReserveLabel()
			lbl35 := ctx.ReserveLabel()
			if d473.Loc == scm.LocImm {
				if d473.Imm.Bool() {
					ctx.MarkLabel(lbl34)
					ctx.EmitJmp(lbl32)
				} else {
					ctx.MarkLabel(lbl35)
			ctx.EnsureDesc(&d468)
			if d468.Loc == scm.LocReg {
				ctx.ProtectReg(d468.Reg)
			} else if d468.Loc == scm.LocRegPair {
				ctx.ProtectReg(d468.Reg)
				ctx.ProtectReg(d468.Reg2)
			}
			d474 = d468
			if d474.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d474)
			ctx.EmitStoreToStack(d474, int32(bbs[2].PhiBase)+int32(0))
			if d468.Loc == scm.LocReg {
				ctx.UnprotectReg(d468.Reg)
			} else if d468.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d468.Reg)
				ctx.UnprotectReg(d468.Reg2)
			}
					ctx.EmitJmp(lbl33)
				}
			} else {
				ctx.EmitCmpRegImm32(d473.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl34)
				ctx.EmitJmp(lbl35)
				ctx.MarkLabel(lbl34)
				ctx.EmitJmp(lbl32)
				ctx.MarkLabel(lbl35)
			ctx.EnsureDesc(&d468)
			if d468.Loc == scm.LocReg {
				ctx.ProtectReg(d468.Reg)
			} else if d468.Loc == scm.LocRegPair {
				ctx.ProtectReg(d468.Reg)
				ctx.ProtectReg(d468.Reg2)
			}
			d475 = d468
			if d475.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d475)
			ctx.EmitStoreToStack(d475, int32(bbs[2].PhiBase)+int32(0))
			if d468.Loc == scm.LocReg {
				ctx.UnprotectReg(d468.Reg)
			} else if d468.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d468.Reg)
				ctx.UnprotectReg(d468.Reg2)
			}
				ctx.EmitJmp(lbl33)
			}
			ctx.FreeDesc(&d472)
			bbpos_3_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl33)
			ctx.ResolveFixups()
			d459 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(176)}
			ctx.EnsureDesc(&d461)
			ctx.EnsureDesc(&d461)
			var d476 scm.JITValueDesc
			if d461.Loc == scm.LocImm {
				d476 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d461.Imm.Int()))))}
			} else {
				r126 := ctx.AllocReg()
				ctx.EmitMovRegReg(r126, d461.Reg)
				ctx.EmitShlRegImm8(r126, 56)
				ctx.EmitShrRegImm8(r126, 56)
				d476 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r126}
				ctx.BindReg(r126, &d476)
			}
			d477 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d476)
			ctx.EnsureDesc(&d477)
			ctx.ProtectReg(d477.Reg)
			ctx.EnsureDesc(&d476)
			ctx.UnprotectReg(d477.Reg)
			var d478 scm.JITValueDesc
			if d477.Loc == scm.LocImm && d476.Loc == scm.LocImm {
				d478 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d477.Imm.Int() - d476.Imm.Int())}
			} else if d476.Loc == scm.LocImm && d476.Imm.Int() == 0 {
				r127 := ctx.AllocRegExcept(d477.Reg)
				ctx.EmitMovRegReg(r127, d477.Reg)
				d478 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
				ctx.BindReg(r127, &d478)
			} else if d477.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d476.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d477.Imm.Int()))
				ctx.EmitSubInt64(scratch, d476.Reg)
				d478 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d478)
			} else if d476.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d477.Reg)
				ctx.EmitMovRegReg(scratch, d477.Reg)
				if d476.Imm.Int() >= -2147483648 && d476.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d476.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d476.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d478 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d478)
			} else {
				r128 := ctx.AllocRegExcept(d477.Reg, d476.Reg)
				ctx.EmitMovRegReg(r128, d477.Reg)
				ctx.EmitSubInt64(r128, d476.Reg)
				d478 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
				ctx.BindReg(r128, &d478)
			}
			if d478.Loc == scm.LocReg && d477.Loc == scm.LocReg && d478.Reg == d477.Reg {
				ctx.TransferReg(d477.Reg)
				d477.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d476)
			ctx.EnsureDesc(&d459)
			ctx.EnsureDesc(&d478)
			var d479 scm.JITValueDesc
			if d459.Loc == scm.LocImm && d478.Loc == scm.LocImm {
				d479 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d459.Imm.Int()) >> uint64(d478.Imm.Int())))}
			} else if d478.Loc == scm.LocImm {
				r129 := ctx.AllocRegExcept(d459.Reg)
				ctx.EmitMovRegReg(r129, d459.Reg)
				ctx.EmitShrRegImm8(r129, uint8(d478.Imm.Int()))
				d479 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
				ctx.BindReg(r129, &d479)
			} else {
				{
					shiftSrc := d459.Reg
					r130 := ctx.AllocRegExcept(d459.Reg)
					ctx.EmitMovRegReg(r130, d459.Reg)
					shiftSrc = r130
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d478.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d478.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d478.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d479 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d479)
				}
			}
			if d479.Loc == scm.LocReg && d459.Loc == scm.LocReg && d479.Reg == d459.Reg {
				ctx.TransferReg(d459.Reg)
				d459.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d459)
			ctx.FreeDesc(&d478)
			r131 := ctx.AllocReg()
			ctx.EnsureDesc(&d479)
			ctx.EnsureDesc(&d479)
			if d479.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r131, d479)
			}
			ctx.EmitJmp(lbl31)
			bbpos_3_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl32)
			ctx.ResolveFixups()
			d459 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(176)}
			ctx.EnsureDesc(&d463)
			var d480 scm.JITValueDesc
			if d463.Loc == scm.LocImm {
				d480 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d463.Imm.Int() / 64)}
			} else {
				r132 := ctx.AllocRegExcept(d463.Reg)
				ctx.EmitMovRegReg(r132, d463.Reg)
				ctx.EmitShrRegImm8(r132, 6)
				d480 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
				ctx.BindReg(r132, &d480)
			}
			if d480.Loc == scm.LocReg && d463.Loc == scm.LocReg && d480.Reg == d463.Reg {
				ctx.TransferReg(d463.Reg)
				d463.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d480)
			ctx.EnsureDesc(&d480)
			var d481 scm.JITValueDesc
			if d480.Loc == scm.LocImm {
				d481 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d480.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d480.Reg)
				ctx.EmitMovRegReg(scratch, d480.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d481 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d481)
			}
			if d481.Loc == scm.LocReg && d480.Loc == scm.LocReg && d481.Reg == d480.Reg {
				ctx.TransferReg(d480.Reg)
				d480.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d480)
			ctx.EnsureDesc(&d481)
			r133 := ctx.AllocReg()
			ctx.EnsureDesc(&d481)
			ctx.EnsureDesc(&d464)
			if d481.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r133, uint64(d481.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r133, d481.Reg)
				ctx.EmitShlRegImm8(r133, 3)
			}
			if d464.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d464.Imm.Int()))
				ctx.EmitAddInt64(r133, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r133, d464.Reg)
			}
			r134 := ctx.AllocRegExcept(r133)
			ctx.EmitMovRegMem(r134, r133, 0)
			ctx.FreeReg(r133)
			d482 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r134}
			ctx.BindReg(r134, &d482)
			ctx.FreeDesc(&d481)
			ctx.EnsureDesc(&d463)
			var d483 scm.JITValueDesc
			if d463.Loc == scm.LocImm {
				d483 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d463.Imm.Int() % 64)}
			} else {
				r135 := ctx.AllocRegExcept(d463.Reg)
				ctx.EmitMovRegReg(r135, d463.Reg)
				ctx.EmitAndRegImm32(r135, 63)
				d483 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r135}
				ctx.BindReg(r135, &d483)
			}
			if d483.Loc == scm.LocReg && d463.Loc == scm.LocReg && d483.Reg == d463.Reg {
				ctx.TransferReg(d463.Reg)
				d463.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d463)
			d484 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d483)
			ctx.EnsureDesc(&d484)
			ctx.ProtectReg(d484.Reg)
			ctx.EnsureDesc(&d483)
			ctx.UnprotectReg(d484.Reg)
			var d485 scm.JITValueDesc
			if d484.Loc == scm.LocImm && d483.Loc == scm.LocImm {
				d485 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d484.Imm.Int() - d483.Imm.Int())}
			} else if d483.Loc == scm.LocImm && d483.Imm.Int() == 0 {
				r136 := ctx.AllocRegExcept(d484.Reg)
				ctx.EmitMovRegReg(r136, d484.Reg)
				d485 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r136}
				ctx.BindReg(r136, &d485)
			} else if d484.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d483.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d484.Imm.Int()))
				ctx.EmitSubInt64(scratch, d483.Reg)
				d485 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d485)
			} else if d483.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d484.Reg)
				ctx.EmitMovRegReg(scratch, d484.Reg)
				if d483.Imm.Int() >= -2147483648 && d483.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d483.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d483.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d485 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d485)
			} else {
				r137 := ctx.AllocRegExcept(d484.Reg, d483.Reg)
				ctx.EmitMovRegReg(r137, d484.Reg)
				ctx.EmitSubInt64(r137, d483.Reg)
				d485 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r137}
				ctx.BindReg(r137, &d485)
			}
			if d485.Loc == scm.LocReg && d484.Loc == scm.LocReg && d485.Reg == d484.Reg {
				ctx.TransferReg(d484.Reg)
				d484.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d483)
			ctx.EnsureDesc(&d482)
			ctx.EnsureDesc(&d485)
			var d486 scm.JITValueDesc
			if d482.Loc == scm.LocImm && d485.Loc == scm.LocImm {
				d486 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d482.Imm.Int()) >> uint64(d485.Imm.Int())))}
			} else if d485.Loc == scm.LocImm {
				r138 := ctx.AllocRegExcept(d482.Reg)
				ctx.EmitMovRegReg(r138, d482.Reg)
				ctx.EmitShrRegImm8(r138, uint8(d485.Imm.Int()))
				d486 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
				ctx.BindReg(r138, &d486)
			} else {
				{
					shiftSrc := d482.Reg
					r139 := ctx.AllocRegExcept(d482.Reg)
					ctx.EmitMovRegReg(r139, d482.Reg)
					shiftSrc = r139
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d485.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d485.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d485.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d486 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d486)
				}
			}
			if d486.Loc == scm.LocReg && d482.Loc == scm.LocReg && d486.Reg == d482.Reg {
				ctx.TransferReg(d482.Reg)
				d482.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d482)
			ctx.FreeDesc(&d485)
			ctx.EnsureDesc(&d468)
			ctx.EnsureDesc(&d486)
			var d487 scm.JITValueDesc
			if d468.Loc == scm.LocImm && d486.Loc == scm.LocImm {
				d487 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d468.Imm.Int() | d486.Imm.Int())}
			} else if d468.Loc == scm.LocImm && d468.Imm.Int() == 0 {
				d487 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d486.Reg}
				ctx.BindReg(d486.Reg, &d487)
			} else if d486.Loc == scm.LocImm && d486.Imm.Int() == 0 {
				r140 := ctx.AllocRegExcept(d468.Reg)
				ctx.EmitMovRegReg(r140, d468.Reg)
				d487 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r140}
				ctx.BindReg(r140, &d487)
			} else if d468.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d486.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d468.Imm.Int()))
				ctx.EmitOrInt64(scratch, d486.Reg)
				d487 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d487)
			} else if d486.Loc == scm.LocImm {
				r141 := ctx.AllocRegExcept(d468.Reg)
				ctx.EmitMovRegReg(r141, d468.Reg)
				if d486.Imm.Int() >= -2147483648 && d486.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r141, int32(d486.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d486.Imm.Int()))
					ctx.EmitOrInt64(r141, scm.RegR11)
				}
				d487 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
				ctx.BindReg(r141, &d487)
			} else {
				r142 := ctx.AllocRegExcept(d468.Reg, d486.Reg)
				ctx.EmitMovRegReg(r142, d468.Reg)
				ctx.EmitOrInt64(r142, d486.Reg)
				d487 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r142}
				ctx.BindReg(r142, &d487)
			}
			if d487.Loc == scm.LocReg && d468.Loc == scm.LocReg && d487.Reg == d468.Reg {
				ctx.TransferReg(d468.Reg)
				d468.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d486)
			ctx.EnsureDesc(&d487)
			if d487.Loc == scm.LocReg {
				ctx.ProtectReg(d487.Reg)
			} else if d487.Loc == scm.LocRegPair {
				ctx.ProtectReg(d487.Reg)
				ctx.ProtectReg(d487.Reg2)
			}
			d488 = d487
			if d488.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d488)
			ctx.EmitStoreToStack(d488, int32(bbs[2].PhiBase)+int32(0))
			if d487.Loc == scm.LocReg {
				ctx.UnprotectReg(d487.Reg)
			} else if d487.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d487.Reg)
				ctx.UnprotectReg(d487.Reg2)
			}
			ctx.EmitJmp(lbl33)
			ctx.MarkLabel(lbl31)
			d489 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r131}
			ctx.BindReg(r131, &d489)
			ctx.BindReg(r131, &d489)
			if r104 { ctx.UnprotectReg(r105) }
			ctx.EnsureDesc(&d489)
			ctx.EnsureDesc(&d489)
			var d490 scm.JITValueDesc
			if d489.Loc == scm.LocImm {
				d490 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d489.Imm.Int()))))}
			} else {
				r143 := ctx.AllocReg()
				ctx.EmitMovRegReg(r143, d489.Reg)
				d490 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r143}
				ctx.BindReg(r143, &d490)
			}
			ctx.FreeDesc(&d489)
			ctx.EnsureDesc(&d490)
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d490)
			ctx.ProtectReg(d490.Reg)
			ctx.EnsureDesc(&d59)
			ctx.UnprotectReg(d490.Reg)
			var d491 scm.JITValueDesc
			if d490.Loc == scm.LocImm && d59.Loc == scm.LocImm {
				d491 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d490.Imm.Int() + d59.Imm.Int())}
			} else if d59.Loc == scm.LocImm && d59.Imm.Int() == 0 {
				r144 := ctx.AllocRegExcept(d490.Reg)
				ctx.EmitMovRegReg(r144, d490.Reg)
				d491 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
				ctx.BindReg(r144, &d491)
			} else if d490.Loc == scm.LocImm && d490.Imm.Int() == 0 {
				d491 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d59.Reg}
				ctx.BindReg(d59.Reg, &d491)
			} else if d490.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d59.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d490.Imm.Int()))
				ctx.EmitAddInt64(scratch, d59.Reg)
				d491 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d491)
			} else if d59.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d490.Reg)
				ctx.EmitMovRegReg(scratch, d490.Reg)
				if d59.Imm.Int() >= -2147483648 && d59.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d59.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d59.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d491 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d491)
			} else {
				r145 := ctx.AllocRegExcept(d490.Reg, d59.Reg)
				ctx.EmitMovRegReg(r145, d490.Reg)
				ctx.EmitAddInt64(r145, d59.Reg)
				d491 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r145}
				ctx.BindReg(r145, &d491)
			}
			if d491.Loc == scm.LocReg && d490.Loc == scm.LocReg && d491.Reg == d490.Reg {
				ctx.TransferReg(d490.Reg)
				d490.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d490)
			ctx.EnsureDesc(&d491)
			ctx.EnsureDesc(&d491)
			var d492 scm.JITValueDesc
			if d491.Loc == scm.LocImm {
				d492 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d491.Imm.Int()))))}
			} else {
				r146 := ctx.AllocReg()
				ctx.EmitMovRegReg(r146, d491.Reg)
				ctx.EmitShlRegImm8(r146, 32)
				ctx.EmitShrRegImm8(r146, 32)
				d492 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r146}
				ctx.BindReg(r146, &d492)
			}
			ctx.FreeDesc(&d491)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d492)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d492)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d492)
			var d493 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d492.Loc == scm.LocImm {
				d493 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d492.Imm.Int()))}
			} else if d492.Loc == scm.LocImm {
				r147 := ctx.AllocRegExcept(idxInt.Reg)
				if d492.Imm.Int() >= -2147483648 && d492.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(idxInt.Reg, int32(d492.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d492.Imm.Int()))
					ctx.EmitCmpInt64(idxInt.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r147, scm.CcB)
				d493 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r147}
				ctx.BindReg(r147, &d493)
			} else if idxInt.Loc == scm.LocImm {
				r148 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d492.Reg)
				ctx.EmitSetcc(r148, scm.CcB)
				d493 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r148}
				ctx.BindReg(r148, &d493)
			} else {
				r149 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.EmitCmpInt64(idxInt.Reg, d492.Reg)
				ctx.EmitSetcc(r149, scm.CcB)
				d493 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r149}
				ctx.BindReg(r149, &d493)
			}
			ctx.FreeDesc(&d492)
			d494 = d493
			ctx.EnsureDesc(&d494)
			if d494.Loc != scm.LocImm && d494.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d494.Loc == scm.LocImm {
				if d494.Imm.Bool() {
			ps495 := scm.PhiState{General: ps.General}
			ps495.OverlayValues = make([]scm.JITValueDesc, 495)
			ps495.OverlayValues[1] = d1
			ps495.OverlayValues[2] = d2
			ps495.OverlayValues[3] = d3
			ps495.OverlayValues[4] = d4
			ps495.OverlayValues[5] = d5
			ps495.OverlayValues[6] = d6
			ps495.OverlayValues[7] = d7
			ps495.OverlayValues[8] = d8
			ps495.OverlayValues[9] = d9
			ps495.OverlayValues[10] = d10
			ps495.OverlayValues[11] = d11
			ps495.OverlayValues[12] = d12
			ps495.OverlayValues[13] = d13
			ps495.OverlayValues[14] = d14
			ps495.OverlayValues[15] = d15
			ps495.OverlayValues[16] = d16
			ps495.OverlayValues[17] = d17
			ps495.OverlayValues[19] = d19
			ps495.OverlayValues[20] = d20
			ps495.OverlayValues[21] = d21
			ps495.OverlayValues[22] = d22
			ps495.OverlayValues[23] = d23
			ps495.OverlayValues[24] = d24
			ps495.OverlayValues[25] = d25
			ps495.OverlayValues[27] = d27
			ps495.OverlayValues[28] = d28
			ps495.OverlayValues[29] = d29
			ps495.OverlayValues[30] = d30
			ps495.OverlayValues[31] = d31
			ps495.OverlayValues[32] = d32
			ps495.OverlayValues[33] = d33
			ps495.OverlayValues[34] = d34
			ps495.OverlayValues[35] = d35
			ps495.OverlayValues[36] = d36
			ps495.OverlayValues[37] = d37
			ps495.OverlayValues[38] = d38
			ps495.OverlayValues[39] = d39
			ps495.OverlayValues[40] = d40
			ps495.OverlayValues[41] = d41
			ps495.OverlayValues[42] = d42
			ps495.OverlayValues[43] = d43
			ps495.OverlayValues[44] = d44
			ps495.OverlayValues[45] = d45
			ps495.OverlayValues[46] = d46
			ps495.OverlayValues[47] = d47
			ps495.OverlayValues[48] = d48
			ps495.OverlayValues[49] = d49
			ps495.OverlayValues[50] = d50
			ps495.OverlayValues[51] = d51
			ps495.OverlayValues[52] = d52
			ps495.OverlayValues[53] = d53
			ps495.OverlayValues[54] = d54
			ps495.OverlayValues[55] = d55
			ps495.OverlayValues[56] = d56
			ps495.OverlayValues[57] = d57
			ps495.OverlayValues[58] = d58
			ps495.OverlayValues[59] = d59
			ps495.OverlayValues[60] = d60
			ps495.OverlayValues[61] = d61
			ps495.OverlayValues[62] = d62
			ps495.OverlayValues[63] = d63
			ps495.OverlayValues[66] = d66
			ps495.OverlayValues[67] = d67
			ps495.OverlayValues[68] = d68
			ps495.OverlayValues[136] = d136
			ps495.OverlayValues[137] = d137
			ps495.OverlayValues[138] = d138
			ps495.OverlayValues[140] = d140
			ps495.OverlayValues[141] = d141
			ps495.OverlayValues[142] = d142
			ps495.OverlayValues[143] = d143
			ps495.OverlayValues[144] = d144
			ps495.OverlayValues[145] = d145
			ps495.OverlayValues[146] = d146
			ps495.OverlayValues[147] = d147
			ps495.OverlayValues[148] = d148
			ps495.OverlayValues[149] = d149
			ps495.OverlayValues[150] = d150
			ps495.OverlayValues[151] = d151
			ps495.OverlayValues[152] = d152
			ps495.OverlayValues[153] = d153
			ps495.OverlayValues[154] = d154
			ps495.OverlayValues[155] = d155
			ps495.OverlayValues[156] = d156
			ps495.OverlayValues[157] = d157
			ps495.OverlayValues[158] = d158
			ps495.OverlayValues[159] = d159
			ps495.OverlayValues[160] = d160
			ps495.OverlayValues[161] = d161
			ps495.OverlayValues[162] = d162
			ps495.OverlayValues[163] = d163
			ps495.OverlayValues[164] = d164
			ps495.OverlayValues[165] = d165
			ps495.OverlayValues[166] = d166
			ps495.OverlayValues[167] = d167
			ps495.OverlayValues[168] = d168
			ps495.OverlayValues[169] = d169
			ps495.OverlayValues[170] = d170
			ps495.OverlayValues[171] = d171
			ps495.OverlayValues[172] = d172
			ps495.OverlayValues[173] = d173
			ps495.OverlayValues[174] = d174
			ps495.OverlayValues[175] = d175
			ps495.OverlayValues[178] = d178
			ps495.OverlayValues[286] = d286
			ps495.OverlayValues[287] = d287
			ps495.OverlayValues[288] = d288
			ps495.OverlayValues[289] = d289
			ps495.OverlayValues[290] = d290
			ps495.OverlayValues[291] = d291
			ps495.OverlayValues[292] = d292
			ps495.OverlayValues[293] = d293
			ps495.OverlayValues[295] = d295
			ps495.OverlayValues[296] = d296
			ps495.OverlayValues[297] = d297
			ps495.OverlayValues[298] = d298
			ps495.OverlayValues[299] = d299
			ps495.OverlayValues[300] = d300
			ps495.OverlayValues[301] = d301
			ps495.OverlayValues[302] = d302
			ps495.OverlayValues[303] = d303
			ps495.OverlayValues[304] = d304
			ps495.OverlayValues[306] = d306
			ps495.OverlayValues[308] = d308
			ps495.OverlayValues[309] = d309
			ps495.OverlayValues[310] = d310
			ps495.OverlayValues[311] = d311
			ps495.OverlayValues[312] = d312
			ps495.OverlayValues[315] = d315
			ps495.OverlayValues[446] = d446
			ps495.OverlayValues[447] = d447
			ps495.OverlayValues[448] = d448
			ps495.OverlayValues[449] = d449
			ps495.OverlayValues[450] = d450
			ps495.OverlayValues[451] = d451
			ps495.OverlayValues[452] = d452
			ps495.OverlayValues[454] = d454
			ps495.OverlayValues[455] = d455
			ps495.OverlayValues[456] = d456
			ps495.OverlayValues[457] = d457
			ps495.OverlayValues[459] = d459
			ps495.OverlayValues[460] = d460
			ps495.OverlayValues[461] = d461
			ps495.OverlayValues[462] = d462
			ps495.OverlayValues[463] = d463
			ps495.OverlayValues[464] = d464
			ps495.OverlayValues[465] = d465
			ps495.OverlayValues[466] = d466
			ps495.OverlayValues[467] = d467
			ps495.OverlayValues[468] = d468
			ps495.OverlayValues[469] = d469
			ps495.OverlayValues[470] = d470
			ps495.OverlayValues[471] = d471
			ps495.OverlayValues[472] = d472
			ps495.OverlayValues[473] = d473
			ps495.OverlayValues[474] = d474
			ps495.OverlayValues[475] = d475
			ps495.OverlayValues[476] = d476
			ps495.OverlayValues[477] = d477
			ps495.OverlayValues[478] = d478
			ps495.OverlayValues[479] = d479
			ps495.OverlayValues[480] = d480
			ps495.OverlayValues[481] = d481
			ps495.OverlayValues[482] = d482
			ps495.OverlayValues[483] = d483
			ps495.OverlayValues[484] = d484
			ps495.OverlayValues[485] = d485
			ps495.OverlayValues[486] = d486
			ps495.OverlayValues[487] = d487
			ps495.OverlayValues[488] = d488
			ps495.OverlayValues[489] = d489
			ps495.OverlayValues[490] = d490
			ps495.OverlayValues[491] = d491
			ps495.OverlayValues[492] = d492
			ps495.OverlayValues[493] = d493
			ps495.OverlayValues[494] = d494
					return bbs[7].RenderPS(ps495)
				}
			ps496 := scm.PhiState{General: ps.General}
			ps496.OverlayValues = make([]scm.JITValueDesc, 495)
			ps496.OverlayValues[1] = d1
			ps496.OverlayValues[2] = d2
			ps496.OverlayValues[3] = d3
			ps496.OverlayValues[4] = d4
			ps496.OverlayValues[5] = d5
			ps496.OverlayValues[6] = d6
			ps496.OverlayValues[7] = d7
			ps496.OverlayValues[8] = d8
			ps496.OverlayValues[9] = d9
			ps496.OverlayValues[10] = d10
			ps496.OverlayValues[11] = d11
			ps496.OverlayValues[12] = d12
			ps496.OverlayValues[13] = d13
			ps496.OverlayValues[14] = d14
			ps496.OverlayValues[15] = d15
			ps496.OverlayValues[16] = d16
			ps496.OverlayValues[17] = d17
			ps496.OverlayValues[19] = d19
			ps496.OverlayValues[20] = d20
			ps496.OverlayValues[21] = d21
			ps496.OverlayValues[22] = d22
			ps496.OverlayValues[23] = d23
			ps496.OverlayValues[24] = d24
			ps496.OverlayValues[25] = d25
			ps496.OverlayValues[27] = d27
			ps496.OverlayValues[28] = d28
			ps496.OverlayValues[29] = d29
			ps496.OverlayValues[30] = d30
			ps496.OverlayValues[31] = d31
			ps496.OverlayValues[32] = d32
			ps496.OverlayValues[33] = d33
			ps496.OverlayValues[34] = d34
			ps496.OverlayValues[35] = d35
			ps496.OverlayValues[36] = d36
			ps496.OverlayValues[37] = d37
			ps496.OverlayValues[38] = d38
			ps496.OverlayValues[39] = d39
			ps496.OverlayValues[40] = d40
			ps496.OverlayValues[41] = d41
			ps496.OverlayValues[42] = d42
			ps496.OverlayValues[43] = d43
			ps496.OverlayValues[44] = d44
			ps496.OverlayValues[45] = d45
			ps496.OverlayValues[46] = d46
			ps496.OverlayValues[47] = d47
			ps496.OverlayValues[48] = d48
			ps496.OverlayValues[49] = d49
			ps496.OverlayValues[50] = d50
			ps496.OverlayValues[51] = d51
			ps496.OverlayValues[52] = d52
			ps496.OverlayValues[53] = d53
			ps496.OverlayValues[54] = d54
			ps496.OverlayValues[55] = d55
			ps496.OverlayValues[56] = d56
			ps496.OverlayValues[57] = d57
			ps496.OverlayValues[58] = d58
			ps496.OverlayValues[59] = d59
			ps496.OverlayValues[60] = d60
			ps496.OverlayValues[61] = d61
			ps496.OverlayValues[62] = d62
			ps496.OverlayValues[63] = d63
			ps496.OverlayValues[66] = d66
			ps496.OverlayValues[67] = d67
			ps496.OverlayValues[68] = d68
			ps496.OverlayValues[136] = d136
			ps496.OverlayValues[137] = d137
			ps496.OverlayValues[138] = d138
			ps496.OverlayValues[140] = d140
			ps496.OverlayValues[141] = d141
			ps496.OverlayValues[142] = d142
			ps496.OverlayValues[143] = d143
			ps496.OverlayValues[144] = d144
			ps496.OverlayValues[145] = d145
			ps496.OverlayValues[146] = d146
			ps496.OverlayValues[147] = d147
			ps496.OverlayValues[148] = d148
			ps496.OverlayValues[149] = d149
			ps496.OverlayValues[150] = d150
			ps496.OverlayValues[151] = d151
			ps496.OverlayValues[152] = d152
			ps496.OverlayValues[153] = d153
			ps496.OverlayValues[154] = d154
			ps496.OverlayValues[155] = d155
			ps496.OverlayValues[156] = d156
			ps496.OverlayValues[157] = d157
			ps496.OverlayValues[158] = d158
			ps496.OverlayValues[159] = d159
			ps496.OverlayValues[160] = d160
			ps496.OverlayValues[161] = d161
			ps496.OverlayValues[162] = d162
			ps496.OverlayValues[163] = d163
			ps496.OverlayValues[164] = d164
			ps496.OverlayValues[165] = d165
			ps496.OverlayValues[166] = d166
			ps496.OverlayValues[167] = d167
			ps496.OverlayValues[168] = d168
			ps496.OverlayValues[169] = d169
			ps496.OverlayValues[170] = d170
			ps496.OverlayValues[171] = d171
			ps496.OverlayValues[172] = d172
			ps496.OverlayValues[173] = d173
			ps496.OverlayValues[174] = d174
			ps496.OverlayValues[175] = d175
			ps496.OverlayValues[178] = d178
			ps496.OverlayValues[286] = d286
			ps496.OverlayValues[287] = d287
			ps496.OverlayValues[288] = d288
			ps496.OverlayValues[289] = d289
			ps496.OverlayValues[290] = d290
			ps496.OverlayValues[291] = d291
			ps496.OverlayValues[292] = d292
			ps496.OverlayValues[293] = d293
			ps496.OverlayValues[295] = d295
			ps496.OverlayValues[296] = d296
			ps496.OverlayValues[297] = d297
			ps496.OverlayValues[298] = d298
			ps496.OverlayValues[299] = d299
			ps496.OverlayValues[300] = d300
			ps496.OverlayValues[301] = d301
			ps496.OverlayValues[302] = d302
			ps496.OverlayValues[303] = d303
			ps496.OverlayValues[304] = d304
			ps496.OverlayValues[306] = d306
			ps496.OverlayValues[308] = d308
			ps496.OverlayValues[309] = d309
			ps496.OverlayValues[310] = d310
			ps496.OverlayValues[311] = d311
			ps496.OverlayValues[312] = d312
			ps496.OverlayValues[315] = d315
			ps496.OverlayValues[446] = d446
			ps496.OverlayValues[447] = d447
			ps496.OverlayValues[448] = d448
			ps496.OverlayValues[449] = d449
			ps496.OverlayValues[450] = d450
			ps496.OverlayValues[451] = d451
			ps496.OverlayValues[452] = d452
			ps496.OverlayValues[454] = d454
			ps496.OverlayValues[455] = d455
			ps496.OverlayValues[456] = d456
			ps496.OverlayValues[457] = d457
			ps496.OverlayValues[459] = d459
			ps496.OverlayValues[460] = d460
			ps496.OverlayValues[461] = d461
			ps496.OverlayValues[462] = d462
			ps496.OverlayValues[463] = d463
			ps496.OverlayValues[464] = d464
			ps496.OverlayValues[465] = d465
			ps496.OverlayValues[466] = d466
			ps496.OverlayValues[467] = d467
			ps496.OverlayValues[468] = d468
			ps496.OverlayValues[469] = d469
			ps496.OverlayValues[470] = d470
			ps496.OverlayValues[471] = d471
			ps496.OverlayValues[472] = d472
			ps496.OverlayValues[473] = d473
			ps496.OverlayValues[474] = d474
			ps496.OverlayValues[475] = d475
			ps496.OverlayValues[476] = d476
			ps496.OverlayValues[477] = d477
			ps496.OverlayValues[478] = d478
			ps496.OverlayValues[479] = d479
			ps496.OverlayValues[480] = d480
			ps496.OverlayValues[481] = d481
			ps496.OverlayValues[482] = d482
			ps496.OverlayValues[483] = d483
			ps496.OverlayValues[484] = d484
			ps496.OverlayValues[485] = d485
			ps496.OverlayValues[486] = d486
			ps496.OverlayValues[487] = d487
			ps496.OverlayValues[488] = d488
			ps496.OverlayValues[489] = d489
			ps496.OverlayValues[490] = d490
			ps496.OverlayValues[491] = d491
			ps496.OverlayValues[492] = d492
			ps496.OverlayValues[493] = d493
			ps496.OverlayValues[494] = d494
				return bbs[9].RenderPS(ps496)
			}
			if !ps.General {
				ps.General = true
				return bbs[6].RenderPS(ps)
			}
			lbl36 := ctx.ReserveLabel()
			lbl37 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d494.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl36)
			ctx.EmitJmp(lbl37)
			ctx.MarkLabel(lbl36)
			ctx.EmitJmp(lbl8)
			ctx.MarkLabel(lbl37)
			ctx.EmitJmp(lbl10)
			ps497 := scm.PhiState{General: true}
			ps497.OverlayValues = make([]scm.JITValueDesc, 495)
			ps497.OverlayValues[1] = d1
			ps497.OverlayValues[2] = d2
			ps497.OverlayValues[3] = d3
			ps497.OverlayValues[4] = d4
			ps497.OverlayValues[5] = d5
			ps497.OverlayValues[6] = d6
			ps497.OverlayValues[7] = d7
			ps497.OverlayValues[8] = d8
			ps497.OverlayValues[9] = d9
			ps497.OverlayValues[10] = d10
			ps497.OverlayValues[11] = d11
			ps497.OverlayValues[12] = d12
			ps497.OverlayValues[13] = d13
			ps497.OverlayValues[14] = d14
			ps497.OverlayValues[15] = d15
			ps497.OverlayValues[16] = d16
			ps497.OverlayValues[17] = d17
			ps497.OverlayValues[19] = d19
			ps497.OverlayValues[20] = d20
			ps497.OverlayValues[21] = d21
			ps497.OverlayValues[22] = d22
			ps497.OverlayValues[23] = d23
			ps497.OverlayValues[24] = d24
			ps497.OverlayValues[25] = d25
			ps497.OverlayValues[27] = d27
			ps497.OverlayValues[28] = d28
			ps497.OverlayValues[29] = d29
			ps497.OverlayValues[30] = d30
			ps497.OverlayValues[31] = d31
			ps497.OverlayValues[32] = d32
			ps497.OverlayValues[33] = d33
			ps497.OverlayValues[34] = d34
			ps497.OverlayValues[35] = d35
			ps497.OverlayValues[36] = d36
			ps497.OverlayValues[37] = d37
			ps497.OverlayValues[38] = d38
			ps497.OverlayValues[39] = d39
			ps497.OverlayValues[40] = d40
			ps497.OverlayValues[41] = d41
			ps497.OverlayValues[42] = d42
			ps497.OverlayValues[43] = d43
			ps497.OverlayValues[44] = d44
			ps497.OverlayValues[45] = d45
			ps497.OverlayValues[46] = d46
			ps497.OverlayValues[47] = d47
			ps497.OverlayValues[48] = d48
			ps497.OverlayValues[49] = d49
			ps497.OverlayValues[50] = d50
			ps497.OverlayValues[51] = d51
			ps497.OverlayValues[52] = d52
			ps497.OverlayValues[53] = d53
			ps497.OverlayValues[54] = d54
			ps497.OverlayValues[55] = d55
			ps497.OverlayValues[56] = d56
			ps497.OverlayValues[57] = d57
			ps497.OverlayValues[58] = d58
			ps497.OverlayValues[59] = d59
			ps497.OverlayValues[60] = d60
			ps497.OverlayValues[61] = d61
			ps497.OverlayValues[62] = d62
			ps497.OverlayValues[63] = d63
			ps497.OverlayValues[66] = d66
			ps497.OverlayValues[67] = d67
			ps497.OverlayValues[68] = d68
			ps497.OverlayValues[136] = d136
			ps497.OverlayValues[137] = d137
			ps497.OverlayValues[138] = d138
			ps497.OverlayValues[140] = d140
			ps497.OverlayValues[141] = d141
			ps497.OverlayValues[142] = d142
			ps497.OverlayValues[143] = d143
			ps497.OverlayValues[144] = d144
			ps497.OverlayValues[145] = d145
			ps497.OverlayValues[146] = d146
			ps497.OverlayValues[147] = d147
			ps497.OverlayValues[148] = d148
			ps497.OverlayValues[149] = d149
			ps497.OverlayValues[150] = d150
			ps497.OverlayValues[151] = d151
			ps497.OverlayValues[152] = d152
			ps497.OverlayValues[153] = d153
			ps497.OverlayValues[154] = d154
			ps497.OverlayValues[155] = d155
			ps497.OverlayValues[156] = d156
			ps497.OverlayValues[157] = d157
			ps497.OverlayValues[158] = d158
			ps497.OverlayValues[159] = d159
			ps497.OverlayValues[160] = d160
			ps497.OverlayValues[161] = d161
			ps497.OverlayValues[162] = d162
			ps497.OverlayValues[163] = d163
			ps497.OverlayValues[164] = d164
			ps497.OverlayValues[165] = d165
			ps497.OverlayValues[166] = d166
			ps497.OverlayValues[167] = d167
			ps497.OverlayValues[168] = d168
			ps497.OverlayValues[169] = d169
			ps497.OverlayValues[170] = d170
			ps497.OverlayValues[171] = d171
			ps497.OverlayValues[172] = d172
			ps497.OverlayValues[173] = d173
			ps497.OverlayValues[174] = d174
			ps497.OverlayValues[175] = d175
			ps497.OverlayValues[178] = d178
			ps497.OverlayValues[286] = d286
			ps497.OverlayValues[287] = d287
			ps497.OverlayValues[288] = d288
			ps497.OverlayValues[289] = d289
			ps497.OverlayValues[290] = d290
			ps497.OverlayValues[291] = d291
			ps497.OverlayValues[292] = d292
			ps497.OverlayValues[293] = d293
			ps497.OverlayValues[295] = d295
			ps497.OverlayValues[296] = d296
			ps497.OverlayValues[297] = d297
			ps497.OverlayValues[298] = d298
			ps497.OverlayValues[299] = d299
			ps497.OverlayValues[300] = d300
			ps497.OverlayValues[301] = d301
			ps497.OverlayValues[302] = d302
			ps497.OverlayValues[303] = d303
			ps497.OverlayValues[304] = d304
			ps497.OverlayValues[306] = d306
			ps497.OverlayValues[308] = d308
			ps497.OverlayValues[309] = d309
			ps497.OverlayValues[310] = d310
			ps497.OverlayValues[311] = d311
			ps497.OverlayValues[312] = d312
			ps497.OverlayValues[315] = d315
			ps497.OverlayValues[446] = d446
			ps497.OverlayValues[447] = d447
			ps497.OverlayValues[448] = d448
			ps497.OverlayValues[449] = d449
			ps497.OverlayValues[450] = d450
			ps497.OverlayValues[451] = d451
			ps497.OverlayValues[452] = d452
			ps497.OverlayValues[454] = d454
			ps497.OverlayValues[455] = d455
			ps497.OverlayValues[456] = d456
			ps497.OverlayValues[457] = d457
			ps497.OverlayValues[459] = d459
			ps497.OverlayValues[460] = d460
			ps497.OverlayValues[461] = d461
			ps497.OverlayValues[462] = d462
			ps497.OverlayValues[463] = d463
			ps497.OverlayValues[464] = d464
			ps497.OverlayValues[465] = d465
			ps497.OverlayValues[466] = d466
			ps497.OverlayValues[467] = d467
			ps497.OverlayValues[468] = d468
			ps497.OverlayValues[469] = d469
			ps497.OverlayValues[470] = d470
			ps497.OverlayValues[471] = d471
			ps497.OverlayValues[472] = d472
			ps497.OverlayValues[473] = d473
			ps497.OverlayValues[474] = d474
			ps497.OverlayValues[475] = d475
			ps497.OverlayValues[476] = d476
			ps497.OverlayValues[477] = d477
			ps497.OverlayValues[478] = d478
			ps497.OverlayValues[479] = d479
			ps497.OverlayValues[480] = d480
			ps497.OverlayValues[481] = d481
			ps497.OverlayValues[482] = d482
			ps497.OverlayValues[483] = d483
			ps497.OverlayValues[484] = d484
			ps497.OverlayValues[485] = d485
			ps497.OverlayValues[486] = d486
			ps497.OverlayValues[487] = d487
			ps497.OverlayValues[488] = d488
			ps497.OverlayValues[489] = d489
			ps497.OverlayValues[490] = d490
			ps497.OverlayValues[491] = d491
			ps497.OverlayValues[492] = d492
			ps497.OverlayValues[493] = d493
			ps497.OverlayValues[494] = d494
			ps498 := scm.PhiState{General: true}
			ps498.OverlayValues = make([]scm.JITValueDesc, 495)
			ps498.OverlayValues[1] = d1
			ps498.OverlayValues[2] = d2
			ps498.OverlayValues[3] = d3
			ps498.OverlayValues[4] = d4
			ps498.OverlayValues[5] = d5
			ps498.OverlayValues[6] = d6
			ps498.OverlayValues[7] = d7
			ps498.OverlayValues[8] = d8
			ps498.OverlayValues[9] = d9
			ps498.OverlayValues[10] = d10
			ps498.OverlayValues[11] = d11
			ps498.OverlayValues[12] = d12
			ps498.OverlayValues[13] = d13
			ps498.OverlayValues[14] = d14
			ps498.OverlayValues[15] = d15
			ps498.OverlayValues[16] = d16
			ps498.OverlayValues[17] = d17
			ps498.OverlayValues[19] = d19
			ps498.OverlayValues[20] = d20
			ps498.OverlayValues[21] = d21
			ps498.OverlayValues[22] = d22
			ps498.OverlayValues[23] = d23
			ps498.OverlayValues[24] = d24
			ps498.OverlayValues[25] = d25
			ps498.OverlayValues[27] = d27
			ps498.OverlayValues[28] = d28
			ps498.OverlayValues[29] = d29
			ps498.OverlayValues[30] = d30
			ps498.OverlayValues[31] = d31
			ps498.OverlayValues[32] = d32
			ps498.OverlayValues[33] = d33
			ps498.OverlayValues[34] = d34
			ps498.OverlayValues[35] = d35
			ps498.OverlayValues[36] = d36
			ps498.OverlayValues[37] = d37
			ps498.OverlayValues[38] = d38
			ps498.OverlayValues[39] = d39
			ps498.OverlayValues[40] = d40
			ps498.OverlayValues[41] = d41
			ps498.OverlayValues[42] = d42
			ps498.OverlayValues[43] = d43
			ps498.OverlayValues[44] = d44
			ps498.OverlayValues[45] = d45
			ps498.OverlayValues[46] = d46
			ps498.OverlayValues[47] = d47
			ps498.OverlayValues[48] = d48
			ps498.OverlayValues[49] = d49
			ps498.OverlayValues[50] = d50
			ps498.OverlayValues[51] = d51
			ps498.OverlayValues[52] = d52
			ps498.OverlayValues[53] = d53
			ps498.OverlayValues[54] = d54
			ps498.OverlayValues[55] = d55
			ps498.OverlayValues[56] = d56
			ps498.OverlayValues[57] = d57
			ps498.OverlayValues[58] = d58
			ps498.OverlayValues[59] = d59
			ps498.OverlayValues[60] = d60
			ps498.OverlayValues[61] = d61
			ps498.OverlayValues[62] = d62
			ps498.OverlayValues[63] = d63
			ps498.OverlayValues[66] = d66
			ps498.OverlayValues[67] = d67
			ps498.OverlayValues[68] = d68
			ps498.OverlayValues[136] = d136
			ps498.OverlayValues[137] = d137
			ps498.OverlayValues[138] = d138
			ps498.OverlayValues[140] = d140
			ps498.OverlayValues[141] = d141
			ps498.OverlayValues[142] = d142
			ps498.OverlayValues[143] = d143
			ps498.OverlayValues[144] = d144
			ps498.OverlayValues[145] = d145
			ps498.OverlayValues[146] = d146
			ps498.OverlayValues[147] = d147
			ps498.OverlayValues[148] = d148
			ps498.OverlayValues[149] = d149
			ps498.OverlayValues[150] = d150
			ps498.OverlayValues[151] = d151
			ps498.OverlayValues[152] = d152
			ps498.OverlayValues[153] = d153
			ps498.OverlayValues[154] = d154
			ps498.OverlayValues[155] = d155
			ps498.OverlayValues[156] = d156
			ps498.OverlayValues[157] = d157
			ps498.OverlayValues[158] = d158
			ps498.OverlayValues[159] = d159
			ps498.OverlayValues[160] = d160
			ps498.OverlayValues[161] = d161
			ps498.OverlayValues[162] = d162
			ps498.OverlayValues[163] = d163
			ps498.OverlayValues[164] = d164
			ps498.OverlayValues[165] = d165
			ps498.OverlayValues[166] = d166
			ps498.OverlayValues[167] = d167
			ps498.OverlayValues[168] = d168
			ps498.OverlayValues[169] = d169
			ps498.OverlayValues[170] = d170
			ps498.OverlayValues[171] = d171
			ps498.OverlayValues[172] = d172
			ps498.OverlayValues[173] = d173
			ps498.OverlayValues[174] = d174
			ps498.OverlayValues[175] = d175
			ps498.OverlayValues[178] = d178
			ps498.OverlayValues[286] = d286
			ps498.OverlayValues[287] = d287
			ps498.OverlayValues[288] = d288
			ps498.OverlayValues[289] = d289
			ps498.OverlayValues[290] = d290
			ps498.OverlayValues[291] = d291
			ps498.OverlayValues[292] = d292
			ps498.OverlayValues[293] = d293
			ps498.OverlayValues[295] = d295
			ps498.OverlayValues[296] = d296
			ps498.OverlayValues[297] = d297
			ps498.OverlayValues[298] = d298
			ps498.OverlayValues[299] = d299
			ps498.OverlayValues[300] = d300
			ps498.OverlayValues[301] = d301
			ps498.OverlayValues[302] = d302
			ps498.OverlayValues[303] = d303
			ps498.OverlayValues[304] = d304
			ps498.OverlayValues[306] = d306
			ps498.OverlayValues[308] = d308
			ps498.OverlayValues[309] = d309
			ps498.OverlayValues[310] = d310
			ps498.OverlayValues[311] = d311
			ps498.OverlayValues[312] = d312
			ps498.OverlayValues[315] = d315
			ps498.OverlayValues[446] = d446
			ps498.OverlayValues[447] = d447
			ps498.OverlayValues[448] = d448
			ps498.OverlayValues[449] = d449
			ps498.OverlayValues[450] = d450
			ps498.OverlayValues[451] = d451
			ps498.OverlayValues[452] = d452
			ps498.OverlayValues[454] = d454
			ps498.OverlayValues[455] = d455
			ps498.OverlayValues[456] = d456
			ps498.OverlayValues[457] = d457
			ps498.OverlayValues[459] = d459
			ps498.OverlayValues[460] = d460
			ps498.OverlayValues[461] = d461
			ps498.OverlayValues[462] = d462
			ps498.OverlayValues[463] = d463
			ps498.OverlayValues[464] = d464
			ps498.OverlayValues[465] = d465
			ps498.OverlayValues[466] = d466
			ps498.OverlayValues[467] = d467
			ps498.OverlayValues[468] = d468
			ps498.OverlayValues[469] = d469
			ps498.OverlayValues[470] = d470
			ps498.OverlayValues[471] = d471
			ps498.OverlayValues[472] = d472
			ps498.OverlayValues[473] = d473
			ps498.OverlayValues[474] = d474
			ps498.OverlayValues[475] = d475
			ps498.OverlayValues[476] = d476
			ps498.OverlayValues[477] = d477
			ps498.OverlayValues[478] = d478
			ps498.OverlayValues[479] = d479
			ps498.OverlayValues[480] = d480
			ps498.OverlayValues[481] = d481
			ps498.OverlayValues[482] = d482
			ps498.OverlayValues[483] = d483
			ps498.OverlayValues[484] = d484
			ps498.OverlayValues[485] = d485
			ps498.OverlayValues[486] = d486
			ps498.OverlayValues[487] = d487
			ps498.OverlayValues[488] = d488
			ps498.OverlayValues[489] = d489
			ps498.OverlayValues[490] = d490
			ps498.OverlayValues[491] = d491
			ps498.OverlayValues[492] = d492
			ps498.OverlayValues[493] = d493
			ps498.OverlayValues[494] = d494
			snap499 := d1
			snap500 := d2
			snap501 := d3
			snap502 := d4
			snap503 := d5
			snap504 := d6
			snap505 := d7
			snap506 := d8
			snap507 := d9
			snap508 := d10
			snap509 := d11
			snap510 := d12
			snap511 := d13
			snap512 := d14
			snap513 := d15
			snap514 := d16
			snap515 := d17
			snap516 := d19
			snap517 := d20
			snap518 := d21
			snap519 := d22
			snap520 := d23
			snap521 := d24
			snap522 := d25
			snap523 := d27
			snap524 := d28
			snap525 := d29
			snap526 := d30
			snap527 := d31
			snap528 := d32
			snap529 := d33
			snap530 := d34
			snap531 := d35
			snap532 := d36
			snap533 := d37
			snap534 := d38
			snap535 := d39
			snap536 := d40
			snap537 := d41
			snap538 := d42
			snap539 := d43
			snap540 := d44
			snap541 := d45
			snap542 := d46
			snap543 := d47
			snap544 := d48
			snap545 := d49
			snap546 := d50
			snap547 := d51
			snap548 := d52
			snap549 := d53
			snap550 := d54
			snap551 := d55
			snap552 := d56
			snap553 := d57
			snap554 := d58
			snap555 := d59
			snap556 := d60
			snap557 := d61
			snap558 := d62
			snap559 := d63
			snap560 := d66
			snap561 := d67
			snap562 := d68
			snap563 := d136
			snap564 := d137
			snap565 := d138
			snap566 := d140
			snap567 := d141
			snap568 := d142
			snap569 := d143
			snap570 := d144
			snap571 := d145
			snap572 := d146
			snap573 := d147
			snap574 := d148
			snap575 := d149
			snap576 := d150
			snap577 := d151
			snap578 := d152
			snap579 := d153
			snap580 := d154
			snap581 := d155
			snap582 := d156
			snap583 := d157
			snap584 := d158
			snap585 := d159
			snap586 := d160
			snap587 := d161
			snap588 := d162
			snap589 := d163
			snap590 := d164
			snap591 := d165
			snap592 := d166
			snap593 := d167
			snap594 := d168
			snap595 := d169
			snap596 := d170
			snap597 := d171
			snap598 := d172
			snap599 := d173
			snap600 := d174
			snap601 := d175
			snap602 := d178
			snap603 := d286
			snap604 := d287
			snap605 := d288
			snap606 := d289
			snap607 := d290
			snap608 := d291
			snap609 := d292
			snap610 := d293
			snap611 := d295
			snap612 := d296
			snap613 := d297
			snap614 := d298
			snap615 := d299
			snap616 := d300
			snap617 := d301
			snap618 := d302
			snap619 := d303
			snap620 := d304
			snap621 := d306
			snap622 := d308
			snap623 := d309
			snap624 := d310
			snap625 := d311
			snap626 := d312
			snap627 := d315
			snap628 := d446
			snap629 := d447
			snap630 := d448
			snap631 := d449
			snap632 := d450
			snap633 := d451
			snap634 := d452
			snap635 := d454
			snap636 := d455
			snap637 := d456
			snap638 := d457
			snap639 := d459
			snap640 := d460
			snap641 := d461
			snap642 := d462
			snap643 := d463
			snap644 := d464
			snap645 := d465
			snap646 := d466
			snap647 := d467
			snap648 := d468
			snap649 := d469
			snap650 := d470
			snap651 := d471
			snap652 := d472
			snap653 := d473
			snap654 := d474
			snap655 := d475
			snap656 := d476
			snap657 := d477
			snap658 := d478
			snap659 := d479
			snap660 := d480
			snap661 := d481
			snap662 := d482
			snap663 := d483
			snap664 := d484
			snap665 := d485
			snap666 := d486
			snap667 := d487
			snap668 := d488
			snap669 := d489
			snap670 := d490
			snap671 := d491
			snap672 := d492
			snap673 := d493
			snap674 := d494
			alloc675 := ctx.SnapshotAllocState()
			if !bbs[9].Rendered {
				bbs[9].RenderPS(ps498)
			}
			ctx.RestoreAllocState(alloc675)
			d1 = snap499
			d2 = snap500
			d3 = snap501
			d4 = snap502
			d5 = snap503
			d6 = snap504
			d7 = snap505
			d8 = snap506
			d9 = snap507
			d10 = snap508
			d11 = snap509
			d12 = snap510
			d13 = snap511
			d14 = snap512
			d15 = snap513
			d16 = snap514
			d17 = snap515
			d19 = snap516
			d20 = snap517
			d21 = snap518
			d22 = snap519
			d23 = snap520
			d24 = snap521
			d25 = snap522
			d27 = snap523
			d28 = snap524
			d29 = snap525
			d30 = snap526
			d31 = snap527
			d32 = snap528
			d33 = snap529
			d34 = snap530
			d35 = snap531
			d36 = snap532
			d37 = snap533
			d38 = snap534
			d39 = snap535
			d40 = snap536
			d41 = snap537
			d42 = snap538
			d43 = snap539
			d44 = snap540
			d45 = snap541
			d46 = snap542
			d47 = snap543
			d48 = snap544
			d49 = snap545
			d50 = snap546
			d51 = snap547
			d52 = snap548
			d53 = snap549
			d54 = snap550
			d55 = snap551
			d56 = snap552
			d57 = snap553
			d58 = snap554
			d59 = snap555
			d60 = snap556
			d61 = snap557
			d62 = snap558
			d63 = snap559
			d66 = snap560
			d67 = snap561
			d68 = snap562
			d136 = snap563
			d137 = snap564
			d138 = snap565
			d140 = snap566
			d141 = snap567
			d142 = snap568
			d143 = snap569
			d144 = snap570
			d145 = snap571
			d146 = snap572
			d147 = snap573
			d148 = snap574
			d149 = snap575
			d150 = snap576
			d151 = snap577
			d152 = snap578
			d153 = snap579
			d154 = snap580
			d155 = snap581
			d156 = snap582
			d157 = snap583
			d158 = snap584
			d159 = snap585
			d160 = snap586
			d161 = snap587
			d162 = snap588
			d163 = snap589
			d164 = snap590
			d165 = snap591
			d166 = snap592
			d167 = snap593
			d168 = snap594
			d169 = snap595
			d170 = snap596
			d171 = snap597
			d172 = snap598
			d173 = snap599
			d174 = snap600
			d175 = snap601
			d178 = snap602
			d286 = snap603
			d287 = snap604
			d288 = snap605
			d289 = snap606
			d290 = snap607
			d291 = snap608
			d292 = snap609
			d293 = snap610
			d295 = snap611
			d296 = snap612
			d297 = snap613
			d298 = snap614
			d299 = snap615
			d300 = snap616
			d301 = snap617
			d302 = snap618
			d303 = snap619
			d304 = snap620
			d306 = snap621
			d308 = snap622
			d309 = snap623
			d310 = snap624
			d311 = snap625
			d312 = snap626
			d315 = snap627
			d446 = snap628
			d447 = snap629
			d448 = snap630
			d449 = snap631
			d450 = snap632
			d451 = snap633
			d452 = snap634
			d454 = snap635
			d455 = snap636
			d456 = snap637
			d457 = snap638
			d459 = snap639
			d460 = snap640
			d461 = snap641
			d462 = snap642
			d463 = snap643
			d464 = snap644
			d465 = snap645
			d466 = snap646
			d467 = snap647
			d468 = snap648
			d469 = snap649
			d470 = snap650
			d471 = snap651
			d472 = snap652
			d473 = snap653
			d474 = snap654
			d475 = snap655
			d476 = snap656
			d477 = snap657
			d478 = snap658
			d479 = snap659
			d480 = snap660
			d481 = snap661
			d482 = snap662
			d483 = snap663
			d484 = snap664
			d485 = snap665
			d486 = snap666
			d487 = snap667
			d488 = snap668
			d489 = snap669
			d490 = snap670
			d491 = snap671
			d492 = snap672
			d493 = snap673
			d494 = snap674
			if !bbs[7].Rendered {
				return bbs[7].RenderPS(ps497)
			}
			return result
			ctx.FreeDesc(&d493)
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
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
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
			if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != scm.LocNone {
				d173 = ps.OverlayValues[173]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
			}
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
				d178 = ps.OverlayValues[178]
			}
			if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != scm.LocNone {
				d286 = ps.OverlayValues[286]
			}
			if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != scm.LocNone {
				d287 = ps.OverlayValues[287]
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
			if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != scm.LocNone {
				d293 = ps.OverlayValues[293]
			}
			if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
				d295 = ps.OverlayValues[295]
			}
			if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != scm.LocNone {
				d296 = ps.OverlayValues[296]
			}
			if len(ps.OverlayValues) > 297 && ps.OverlayValues[297].Loc != scm.LocNone {
				d297 = ps.OverlayValues[297]
			}
			if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != scm.LocNone {
				d298 = ps.OverlayValues[298]
			}
			if len(ps.OverlayValues) > 299 && ps.OverlayValues[299].Loc != scm.LocNone {
				d299 = ps.OverlayValues[299]
			}
			if len(ps.OverlayValues) > 300 && ps.OverlayValues[300].Loc != scm.LocNone {
				d300 = ps.OverlayValues[300]
			}
			if len(ps.OverlayValues) > 301 && ps.OverlayValues[301].Loc != scm.LocNone {
				d301 = ps.OverlayValues[301]
			}
			if len(ps.OverlayValues) > 302 && ps.OverlayValues[302].Loc != scm.LocNone {
				d302 = ps.OverlayValues[302]
			}
			if len(ps.OverlayValues) > 303 && ps.OverlayValues[303].Loc != scm.LocNone {
				d303 = ps.OverlayValues[303]
			}
			if len(ps.OverlayValues) > 304 && ps.OverlayValues[304].Loc != scm.LocNone {
				d304 = ps.OverlayValues[304]
			}
			if len(ps.OverlayValues) > 306 && ps.OverlayValues[306].Loc != scm.LocNone {
				d306 = ps.OverlayValues[306]
			}
			if len(ps.OverlayValues) > 308 && ps.OverlayValues[308].Loc != scm.LocNone {
				d308 = ps.OverlayValues[308]
			}
			if len(ps.OverlayValues) > 309 && ps.OverlayValues[309].Loc != scm.LocNone {
				d309 = ps.OverlayValues[309]
			}
			if len(ps.OverlayValues) > 310 && ps.OverlayValues[310].Loc != scm.LocNone {
				d310 = ps.OverlayValues[310]
			}
			if len(ps.OverlayValues) > 311 && ps.OverlayValues[311].Loc != scm.LocNone {
				d311 = ps.OverlayValues[311]
			}
			if len(ps.OverlayValues) > 312 && ps.OverlayValues[312].Loc != scm.LocNone {
				d312 = ps.OverlayValues[312]
			}
			if len(ps.OverlayValues) > 315 && ps.OverlayValues[315].Loc != scm.LocNone {
				d315 = ps.OverlayValues[315]
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
			if len(ps.OverlayValues) > 459 && ps.OverlayValues[459].Loc != scm.LocNone {
				d459 = ps.OverlayValues[459]
			}
			if len(ps.OverlayValues) > 460 && ps.OverlayValues[460].Loc != scm.LocNone {
				d460 = ps.OverlayValues[460]
			}
			if len(ps.OverlayValues) > 461 && ps.OverlayValues[461].Loc != scm.LocNone {
				d461 = ps.OverlayValues[461]
			}
			if len(ps.OverlayValues) > 462 && ps.OverlayValues[462].Loc != scm.LocNone {
				d462 = ps.OverlayValues[462]
			}
			if len(ps.OverlayValues) > 463 && ps.OverlayValues[463].Loc != scm.LocNone {
				d463 = ps.OverlayValues[463]
			}
			if len(ps.OverlayValues) > 464 && ps.OverlayValues[464].Loc != scm.LocNone {
				d464 = ps.OverlayValues[464]
			}
			if len(ps.OverlayValues) > 465 && ps.OverlayValues[465].Loc != scm.LocNone {
				d465 = ps.OverlayValues[465]
			}
			if len(ps.OverlayValues) > 466 && ps.OverlayValues[466].Loc != scm.LocNone {
				d466 = ps.OverlayValues[466]
			}
			if len(ps.OverlayValues) > 467 && ps.OverlayValues[467].Loc != scm.LocNone {
				d467 = ps.OverlayValues[467]
			}
			if len(ps.OverlayValues) > 468 && ps.OverlayValues[468].Loc != scm.LocNone {
				d468 = ps.OverlayValues[468]
			}
			if len(ps.OverlayValues) > 469 && ps.OverlayValues[469].Loc != scm.LocNone {
				d469 = ps.OverlayValues[469]
			}
			if len(ps.OverlayValues) > 470 && ps.OverlayValues[470].Loc != scm.LocNone {
				d470 = ps.OverlayValues[470]
			}
			if len(ps.OverlayValues) > 471 && ps.OverlayValues[471].Loc != scm.LocNone {
				d471 = ps.OverlayValues[471]
			}
			if len(ps.OverlayValues) > 472 && ps.OverlayValues[472].Loc != scm.LocNone {
				d472 = ps.OverlayValues[472]
			}
			if len(ps.OverlayValues) > 473 && ps.OverlayValues[473].Loc != scm.LocNone {
				d473 = ps.OverlayValues[473]
			}
			if len(ps.OverlayValues) > 474 && ps.OverlayValues[474].Loc != scm.LocNone {
				d474 = ps.OverlayValues[474]
			}
			if len(ps.OverlayValues) > 475 && ps.OverlayValues[475].Loc != scm.LocNone {
				d475 = ps.OverlayValues[475]
			}
			if len(ps.OverlayValues) > 476 && ps.OverlayValues[476].Loc != scm.LocNone {
				d476 = ps.OverlayValues[476]
			}
			if len(ps.OverlayValues) > 477 && ps.OverlayValues[477].Loc != scm.LocNone {
				d477 = ps.OverlayValues[477]
			}
			if len(ps.OverlayValues) > 478 && ps.OverlayValues[478].Loc != scm.LocNone {
				d478 = ps.OverlayValues[478]
			}
			if len(ps.OverlayValues) > 479 && ps.OverlayValues[479].Loc != scm.LocNone {
				d479 = ps.OverlayValues[479]
			}
			if len(ps.OverlayValues) > 480 && ps.OverlayValues[480].Loc != scm.LocNone {
				d480 = ps.OverlayValues[480]
			}
			if len(ps.OverlayValues) > 481 && ps.OverlayValues[481].Loc != scm.LocNone {
				d481 = ps.OverlayValues[481]
			}
			if len(ps.OverlayValues) > 482 && ps.OverlayValues[482].Loc != scm.LocNone {
				d482 = ps.OverlayValues[482]
			}
			if len(ps.OverlayValues) > 483 && ps.OverlayValues[483].Loc != scm.LocNone {
				d483 = ps.OverlayValues[483]
			}
			if len(ps.OverlayValues) > 484 && ps.OverlayValues[484].Loc != scm.LocNone {
				d484 = ps.OverlayValues[484]
			}
			if len(ps.OverlayValues) > 485 && ps.OverlayValues[485].Loc != scm.LocNone {
				d485 = ps.OverlayValues[485]
			}
			if len(ps.OverlayValues) > 486 && ps.OverlayValues[486].Loc != scm.LocNone {
				d486 = ps.OverlayValues[486]
			}
			if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != scm.LocNone {
				d487 = ps.OverlayValues[487]
			}
			if len(ps.OverlayValues) > 488 && ps.OverlayValues[488].Loc != scm.LocNone {
				d488 = ps.OverlayValues[488]
			}
			if len(ps.OverlayValues) > 489 && ps.OverlayValues[489].Loc != scm.LocNone {
				d489 = ps.OverlayValues[489]
			}
			if len(ps.OverlayValues) > 490 && ps.OverlayValues[490].Loc != scm.LocNone {
				d490 = ps.OverlayValues[490]
			}
			if len(ps.OverlayValues) > 491 && ps.OverlayValues[491].Loc != scm.LocNone {
				d491 = ps.OverlayValues[491]
			}
			if len(ps.OverlayValues) > 492 && ps.OverlayValues[492].Loc != scm.LocNone {
				d492 = ps.OverlayValues[492]
			}
			if len(ps.OverlayValues) > 493 && ps.OverlayValues[493].Loc != scm.LocNone {
				d493 = ps.OverlayValues[493]
			}
			if len(ps.OverlayValues) > 494 && ps.OverlayValues[494].Loc != scm.LocNone {
				d494 = ps.OverlayValues[494]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d5)
			var d676 scm.JITValueDesc
			if d5.Loc == scm.LocImm {
				d676 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d5.Reg)
				ctx.EmitMovRegReg(scratch, d5.Reg)
				ctx.EmitSubRegImm32(scratch, int32(1))
				d676 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d676)
			}
			if d676.Loc == scm.LocImm {
				d676 = scm.JITValueDesc{Loc: scm.LocImm, Type: d676.Type, Imm: scm.NewInt(int64(uint64(d676.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d676.Reg, 32)
				ctx.EmitShrRegImm8(d676.Reg, 32)
			}
			if d676.Loc == scm.LocReg && d5.Loc == scm.LocReg && d676.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d6)
			if d6.Loc == scm.LocReg {
				ctx.ProtectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.ProtectReg(d6.Reg)
				ctx.ProtectReg(d6.Reg2)
			}
			ctx.EnsureDesc(&d676)
			if d676.Loc == scm.LocReg {
				ctx.ProtectReg(d676.Reg)
			} else if d676.Loc == scm.LocRegPair {
				ctx.ProtectReg(d676.Reg)
				ctx.ProtectReg(d676.Reg2)
			}
			d677 = d6
			if d677.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d677)
			d678 = d677
			if d678.Loc == scm.LocImm {
				d678 = scm.JITValueDesc{Loc: scm.LocImm, Type: d678.Type, Imm: scm.NewInt(int64(uint64(d678.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d678.Reg, 32)
				ctx.EmitShrRegImm8(d678.Reg, 32)
			}
			ctx.EmitStoreToStack(d678, int32(bbs[8].PhiBase)+int32(0))
			d679 = d676
			if d679.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d679)
			d680 = d679
			if d680.Loc == scm.LocImm {
				d680 = scm.JITValueDesc{Loc: scm.LocImm, Type: d680.Type, Imm: scm.NewInt(int64(uint64(d680.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d680.Reg, 32)
				ctx.EmitShrRegImm8(d680.Reg, 32)
			}
			ctx.EmitStoreToStack(d680, int32(bbs[8].PhiBase)+int32(16))
			if d6.Loc == scm.LocReg {
				ctx.UnprotectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d6.Reg)
				ctx.UnprotectReg(d6.Reg2)
			}
			if d676.Loc == scm.LocReg {
				ctx.UnprotectReg(d676.Reg)
			} else if d676.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d676.Reg)
				ctx.UnprotectReg(d676.Reg2)
			}
			ps681 := scm.PhiState{General: ps.General}
			ps681.OverlayValues = make([]scm.JITValueDesc, 681)
			ps681.OverlayValues[1] = d1
			ps681.OverlayValues[2] = d2
			ps681.OverlayValues[3] = d3
			ps681.OverlayValues[4] = d4
			ps681.OverlayValues[5] = d5
			ps681.OverlayValues[6] = d6
			ps681.OverlayValues[7] = d7
			ps681.OverlayValues[8] = d8
			ps681.OverlayValues[9] = d9
			ps681.OverlayValues[10] = d10
			ps681.OverlayValues[11] = d11
			ps681.OverlayValues[12] = d12
			ps681.OverlayValues[13] = d13
			ps681.OverlayValues[14] = d14
			ps681.OverlayValues[15] = d15
			ps681.OverlayValues[16] = d16
			ps681.OverlayValues[17] = d17
			ps681.OverlayValues[19] = d19
			ps681.OverlayValues[20] = d20
			ps681.OverlayValues[21] = d21
			ps681.OverlayValues[22] = d22
			ps681.OverlayValues[23] = d23
			ps681.OverlayValues[24] = d24
			ps681.OverlayValues[25] = d25
			ps681.OverlayValues[27] = d27
			ps681.OverlayValues[28] = d28
			ps681.OverlayValues[29] = d29
			ps681.OverlayValues[30] = d30
			ps681.OverlayValues[31] = d31
			ps681.OverlayValues[32] = d32
			ps681.OverlayValues[33] = d33
			ps681.OverlayValues[34] = d34
			ps681.OverlayValues[35] = d35
			ps681.OverlayValues[36] = d36
			ps681.OverlayValues[37] = d37
			ps681.OverlayValues[38] = d38
			ps681.OverlayValues[39] = d39
			ps681.OverlayValues[40] = d40
			ps681.OverlayValues[41] = d41
			ps681.OverlayValues[42] = d42
			ps681.OverlayValues[43] = d43
			ps681.OverlayValues[44] = d44
			ps681.OverlayValues[45] = d45
			ps681.OverlayValues[46] = d46
			ps681.OverlayValues[47] = d47
			ps681.OverlayValues[48] = d48
			ps681.OverlayValues[49] = d49
			ps681.OverlayValues[50] = d50
			ps681.OverlayValues[51] = d51
			ps681.OverlayValues[52] = d52
			ps681.OverlayValues[53] = d53
			ps681.OverlayValues[54] = d54
			ps681.OverlayValues[55] = d55
			ps681.OverlayValues[56] = d56
			ps681.OverlayValues[57] = d57
			ps681.OverlayValues[58] = d58
			ps681.OverlayValues[59] = d59
			ps681.OverlayValues[60] = d60
			ps681.OverlayValues[61] = d61
			ps681.OverlayValues[62] = d62
			ps681.OverlayValues[63] = d63
			ps681.OverlayValues[66] = d66
			ps681.OverlayValues[67] = d67
			ps681.OverlayValues[68] = d68
			ps681.OverlayValues[136] = d136
			ps681.OverlayValues[137] = d137
			ps681.OverlayValues[138] = d138
			ps681.OverlayValues[140] = d140
			ps681.OverlayValues[141] = d141
			ps681.OverlayValues[142] = d142
			ps681.OverlayValues[143] = d143
			ps681.OverlayValues[144] = d144
			ps681.OverlayValues[145] = d145
			ps681.OverlayValues[146] = d146
			ps681.OverlayValues[147] = d147
			ps681.OverlayValues[148] = d148
			ps681.OverlayValues[149] = d149
			ps681.OverlayValues[150] = d150
			ps681.OverlayValues[151] = d151
			ps681.OverlayValues[152] = d152
			ps681.OverlayValues[153] = d153
			ps681.OverlayValues[154] = d154
			ps681.OverlayValues[155] = d155
			ps681.OverlayValues[156] = d156
			ps681.OverlayValues[157] = d157
			ps681.OverlayValues[158] = d158
			ps681.OverlayValues[159] = d159
			ps681.OverlayValues[160] = d160
			ps681.OverlayValues[161] = d161
			ps681.OverlayValues[162] = d162
			ps681.OverlayValues[163] = d163
			ps681.OverlayValues[164] = d164
			ps681.OverlayValues[165] = d165
			ps681.OverlayValues[166] = d166
			ps681.OverlayValues[167] = d167
			ps681.OverlayValues[168] = d168
			ps681.OverlayValues[169] = d169
			ps681.OverlayValues[170] = d170
			ps681.OverlayValues[171] = d171
			ps681.OverlayValues[172] = d172
			ps681.OverlayValues[173] = d173
			ps681.OverlayValues[174] = d174
			ps681.OverlayValues[175] = d175
			ps681.OverlayValues[178] = d178
			ps681.OverlayValues[286] = d286
			ps681.OverlayValues[287] = d287
			ps681.OverlayValues[288] = d288
			ps681.OverlayValues[289] = d289
			ps681.OverlayValues[290] = d290
			ps681.OverlayValues[291] = d291
			ps681.OverlayValues[292] = d292
			ps681.OverlayValues[293] = d293
			ps681.OverlayValues[295] = d295
			ps681.OverlayValues[296] = d296
			ps681.OverlayValues[297] = d297
			ps681.OverlayValues[298] = d298
			ps681.OverlayValues[299] = d299
			ps681.OverlayValues[300] = d300
			ps681.OverlayValues[301] = d301
			ps681.OverlayValues[302] = d302
			ps681.OverlayValues[303] = d303
			ps681.OverlayValues[304] = d304
			ps681.OverlayValues[306] = d306
			ps681.OverlayValues[308] = d308
			ps681.OverlayValues[309] = d309
			ps681.OverlayValues[310] = d310
			ps681.OverlayValues[311] = d311
			ps681.OverlayValues[312] = d312
			ps681.OverlayValues[315] = d315
			ps681.OverlayValues[446] = d446
			ps681.OverlayValues[447] = d447
			ps681.OverlayValues[448] = d448
			ps681.OverlayValues[449] = d449
			ps681.OverlayValues[450] = d450
			ps681.OverlayValues[451] = d451
			ps681.OverlayValues[452] = d452
			ps681.OverlayValues[454] = d454
			ps681.OverlayValues[455] = d455
			ps681.OverlayValues[456] = d456
			ps681.OverlayValues[457] = d457
			ps681.OverlayValues[459] = d459
			ps681.OverlayValues[460] = d460
			ps681.OverlayValues[461] = d461
			ps681.OverlayValues[462] = d462
			ps681.OverlayValues[463] = d463
			ps681.OverlayValues[464] = d464
			ps681.OverlayValues[465] = d465
			ps681.OverlayValues[466] = d466
			ps681.OverlayValues[467] = d467
			ps681.OverlayValues[468] = d468
			ps681.OverlayValues[469] = d469
			ps681.OverlayValues[470] = d470
			ps681.OverlayValues[471] = d471
			ps681.OverlayValues[472] = d472
			ps681.OverlayValues[473] = d473
			ps681.OverlayValues[474] = d474
			ps681.OverlayValues[475] = d475
			ps681.OverlayValues[476] = d476
			ps681.OverlayValues[477] = d477
			ps681.OverlayValues[478] = d478
			ps681.OverlayValues[479] = d479
			ps681.OverlayValues[480] = d480
			ps681.OverlayValues[481] = d481
			ps681.OverlayValues[482] = d482
			ps681.OverlayValues[483] = d483
			ps681.OverlayValues[484] = d484
			ps681.OverlayValues[485] = d485
			ps681.OverlayValues[486] = d486
			ps681.OverlayValues[487] = d487
			ps681.OverlayValues[488] = d488
			ps681.OverlayValues[489] = d489
			ps681.OverlayValues[490] = d490
			ps681.OverlayValues[491] = d491
			ps681.OverlayValues[492] = d492
			ps681.OverlayValues[493] = d493
			ps681.OverlayValues[494] = d494
			ps681.OverlayValues[676] = d676
			ps681.OverlayValues[677] = d677
			ps681.OverlayValues[678] = d678
			ps681.OverlayValues[679] = d679
			ps681.OverlayValues[680] = d680
			ps681.PhiValues = make([]scm.JITValueDesc, 2)
			d682 = d6
			ps681.PhiValues[0] = d682
			d683 = d676
			ps681.PhiValues[1] = d683
			if ps681.General && bbs[8].Rendered {
				ctx.EmitJmp(lbl9)
				return result
			}
			return bbs[8].RenderPS(ps681)
			return result
			}
			bbs[8].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
					d684 := ps.PhiValues[0]
					ctx.EnsureDesc(&d684)
					ctx.EmitStoreToStack(d684, int32(bbs[8].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
					d685 := ps.PhiValues[1]
					ctx.EnsureDesc(&d685)
					ctx.EmitStoreToStack(d685, int32(bbs[8].PhiBase)+int32(16))
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
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
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
			if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != scm.LocNone {
				d173 = ps.OverlayValues[173]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
			}
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
				d178 = ps.OverlayValues[178]
			}
			if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != scm.LocNone {
				d286 = ps.OverlayValues[286]
			}
			if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != scm.LocNone {
				d287 = ps.OverlayValues[287]
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
			if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != scm.LocNone {
				d293 = ps.OverlayValues[293]
			}
			if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
				d295 = ps.OverlayValues[295]
			}
			if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != scm.LocNone {
				d296 = ps.OverlayValues[296]
			}
			if len(ps.OverlayValues) > 297 && ps.OverlayValues[297].Loc != scm.LocNone {
				d297 = ps.OverlayValues[297]
			}
			if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != scm.LocNone {
				d298 = ps.OverlayValues[298]
			}
			if len(ps.OverlayValues) > 299 && ps.OverlayValues[299].Loc != scm.LocNone {
				d299 = ps.OverlayValues[299]
			}
			if len(ps.OverlayValues) > 300 && ps.OverlayValues[300].Loc != scm.LocNone {
				d300 = ps.OverlayValues[300]
			}
			if len(ps.OverlayValues) > 301 && ps.OverlayValues[301].Loc != scm.LocNone {
				d301 = ps.OverlayValues[301]
			}
			if len(ps.OverlayValues) > 302 && ps.OverlayValues[302].Loc != scm.LocNone {
				d302 = ps.OverlayValues[302]
			}
			if len(ps.OverlayValues) > 303 && ps.OverlayValues[303].Loc != scm.LocNone {
				d303 = ps.OverlayValues[303]
			}
			if len(ps.OverlayValues) > 304 && ps.OverlayValues[304].Loc != scm.LocNone {
				d304 = ps.OverlayValues[304]
			}
			if len(ps.OverlayValues) > 306 && ps.OverlayValues[306].Loc != scm.LocNone {
				d306 = ps.OverlayValues[306]
			}
			if len(ps.OverlayValues) > 308 && ps.OverlayValues[308].Loc != scm.LocNone {
				d308 = ps.OverlayValues[308]
			}
			if len(ps.OverlayValues) > 309 && ps.OverlayValues[309].Loc != scm.LocNone {
				d309 = ps.OverlayValues[309]
			}
			if len(ps.OverlayValues) > 310 && ps.OverlayValues[310].Loc != scm.LocNone {
				d310 = ps.OverlayValues[310]
			}
			if len(ps.OverlayValues) > 311 && ps.OverlayValues[311].Loc != scm.LocNone {
				d311 = ps.OverlayValues[311]
			}
			if len(ps.OverlayValues) > 312 && ps.OverlayValues[312].Loc != scm.LocNone {
				d312 = ps.OverlayValues[312]
			}
			if len(ps.OverlayValues) > 315 && ps.OverlayValues[315].Loc != scm.LocNone {
				d315 = ps.OverlayValues[315]
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
			if len(ps.OverlayValues) > 459 && ps.OverlayValues[459].Loc != scm.LocNone {
				d459 = ps.OverlayValues[459]
			}
			if len(ps.OverlayValues) > 460 && ps.OverlayValues[460].Loc != scm.LocNone {
				d460 = ps.OverlayValues[460]
			}
			if len(ps.OverlayValues) > 461 && ps.OverlayValues[461].Loc != scm.LocNone {
				d461 = ps.OverlayValues[461]
			}
			if len(ps.OverlayValues) > 462 && ps.OverlayValues[462].Loc != scm.LocNone {
				d462 = ps.OverlayValues[462]
			}
			if len(ps.OverlayValues) > 463 && ps.OverlayValues[463].Loc != scm.LocNone {
				d463 = ps.OverlayValues[463]
			}
			if len(ps.OverlayValues) > 464 && ps.OverlayValues[464].Loc != scm.LocNone {
				d464 = ps.OverlayValues[464]
			}
			if len(ps.OverlayValues) > 465 && ps.OverlayValues[465].Loc != scm.LocNone {
				d465 = ps.OverlayValues[465]
			}
			if len(ps.OverlayValues) > 466 && ps.OverlayValues[466].Loc != scm.LocNone {
				d466 = ps.OverlayValues[466]
			}
			if len(ps.OverlayValues) > 467 && ps.OverlayValues[467].Loc != scm.LocNone {
				d467 = ps.OverlayValues[467]
			}
			if len(ps.OverlayValues) > 468 && ps.OverlayValues[468].Loc != scm.LocNone {
				d468 = ps.OverlayValues[468]
			}
			if len(ps.OverlayValues) > 469 && ps.OverlayValues[469].Loc != scm.LocNone {
				d469 = ps.OverlayValues[469]
			}
			if len(ps.OverlayValues) > 470 && ps.OverlayValues[470].Loc != scm.LocNone {
				d470 = ps.OverlayValues[470]
			}
			if len(ps.OverlayValues) > 471 && ps.OverlayValues[471].Loc != scm.LocNone {
				d471 = ps.OverlayValues[471]
			}
			if len(ps.OverlayValues) > 472 && ps.OverlayValues[472].Loc != scm.LocNone {
				d472 = ps.OverlayValues[472]
			}
			if len(ps.OverlayValues) > 473 && ps.OverlayValues[473].Loc != scm.LocNone {
				d473 = ps.OverlayValues[473]
			}
			if len(ps.OverlayValues) > 474 && ps.OverlayValues[474].Loc != scm.LocNone {
				d474 = ps.OverlayValues[474]
			}
			if len(ps.OverlayValues) > 475 && ps.OverlayValues[475].Loc != scm.LocNone {
				d475 = ps.OverlayValues[475]
			}
			if len(ps.OverlayValues) > 476 && ps.OverlayValues[476].Loc != scm.LocNone {
				d476 = ps.OverlayValues[476]
			}
			if len(ps.OverlayValues) > 477 && ps.OverlayValues[477].Loc != scm.LocNone {
				d477 = ps.OverlayValues[477]
			}
			if len(ps.OverlayValues) > 478 && ps.OverlayValues[478].Loc != scm.LocNone {
				d478 = ps.OverlayValues[478]
			}
			if len(ps.OverlayValues) > 479 && ps.OverlayValues[479].Loc != scm.LocNone {
				d479 = ps.OverlayValues[479]
			}
			if len(ps.OverlayValues) > 480 && ps.OverlayValues[480].Loc != scm.LocNone {
				d480 = ps.OverlayValues[480]
			}
			if len(ps.OverlayValues) > 481 && ps.OverlayValues[481].Loc != scm.LocNone {
				d481 = ps.OverlayValues[481]
			}
			if len(ps.OverlayValues) > 482 && ps.OverlayValues[482].Loc != scm.LocNone {
				d482 = ps.OverlayValues[482]
			}
			if len(ps.OverlayValues) > 483 && ps.OverlayValues[483].Loc != scm.LocNone {
				d483 = ps.OverlayValues[483]
			}
			if len(ps.OverlayValues) > 484 && ps.OverlayValues[484].Loc != scm.LocNone {
				d484 = ps.OverlayValues[484]
			}
			if len(ps.OverlayValues) > 485 && ps.OverlayValues[485].Loc != scm.LocNone {
				d485 = ps.OverlayValues[485]
			}
			if len(ps.OverlayValues) > 486 && ps.OverlayValues[486].Loc != scm.LocNone {
				d486 = ps.OverlayValues[486]
			}
			if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != scm.LocNone {
				d487 = ps.OverlayValues[487]
			}
			if len(ps.OverlayValues) > 488 && ps.OverlayValues[488].Loc != scm.LocNone {
				d488 = ps.OverlayValues[488]
			}
			if len(ps.OverlayValues) > 489 && ps.OverlayValues[489].Loc != scm.LocNone {
				d489 = ps.OverlayValues[489]
			}
			if len(ps.OverlayValues) > 490 && ps.OverlayValues[490].Loc != scm.LocNone {
				d490 = ps.OverlayValues[490]
			}
			if len(ps.OverlayValues) > 491 && ps.OverlayValues[491].Loc != scm.LocNone {
				d491 = ps.OverlayValues[491]
			}
			if len(ps.OverlayValues) > 492 && ps.OverlayValues[492].Loc != scm.LocNone {
				d492 = ps.OverlayValues[492]
			}
			if len(ps.OverlayValues) > 493 && ps.OverlayValues[493].Loc != scm.LocNone {
				d493 = ps.OverlayValues[493]
			}
			if len(ps.OverlayValues) > 494 && ps.OverlayValues[494].Loc != scm.LocNone {
				d494 = ps.OverlayValues[494]
			}
			if len(ps.OverlayValues) > 676 && ps.OverlayValues[676].Loc != scm.LocNone {
				d676 = ps.OverlayValues[676]
			}
			if len(ps.OverlayValues) > 677 && ps.OverlayValues[677].Loc != scm.LocNone {
				d677 = ps.OverlayValues[677]
			}
			if len(ps.OverlayValues) > 678 && ps.OverlayValues[678].Loc != scm.LocNone {
				d678 = ps.OverlayValues[678]
			}
			if len(ps.OverlayValues) > 679 && ps.OverlayValues[679].Loc != scm.LocNone {
				d679 = ps.OverlayValues[679]
			}
			if len(ps.OverlayValues) > 680 && ps.OverlayValues[680].Loc != scm.LocNone {
				d680 = ps.OverlayValues[680]
			}
			if len(ps.OverlayValues) > 682 && ps.OverlayValues[682].Loc != scm.LocNone {
				d682 = ps.OverlayValues[682]
			}
			if len(ps.OverlayValues) > 683 && ps.OverlayValues[683].Loc != scm.LocNone {
				d683 = ps.OverlayValues[683]
			}
			if len(ps.OverlayValues) > 684 && ps.OverlayValues[684].Loc != scm.LocNone {
				d684 = ps.OverlayValues[684]
			}
			if len(ps.OverlayValues) > 685 && ps.OverlayValues[685].Loc != scm.LocNone {
				d685 = ps.OverlayValues[685]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d8 = ps.PhiValues[0]
			}
			if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d9 = ps.PhiValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d9)
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d9)
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d9)
			var d686 scm.JITValueDesc
			if d8.Loc == scm.LocImm && d9.Loc == scm.LocImm {
				d686 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d8.Imm.Int()) == uint64(d9.Imm.Int()))}
			} else if d9.Loc == scm.LocImm {
				r150 := ctx.AllocRegExcept(d8.Reg)
				if d9.Imm.Int() >= -2147483648 && d9.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d8.Reg, int32(d9.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d9.Imm.Int()))
					ctx.EmitCmpInt64(d8.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r150, scm.CcE)
				d686 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r150}
				ctx.BindReg(r150, &d686)
			} else if d8.Loc == scm.LocImm {
				r151 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d8.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d9.Reg)
				ctx.EmitSetcc(r151, scm.CcE)
				d686 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r151}
				ctx.BindReg(r151, &d686)
			} else {
				r152 := ctx.AllocRegExcept(d8.Reg)
				ctx.EmitCmpInt64(d8.Reg, d9.Reg)
				ctx.EmitSetcc(r152, scm.CcE)
				d686 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r152}
				ctx.BindReg(r152, &d686)
			}
			d687 = d686
			ctx.EnsureDesc(&d687)
			if d687.Loc != scm.LocImm && d687.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d687.Loc == scm.LocImm {
				if d687.Imm.Bool() {
			ctx.EnsureDesc(&d8)
			if d8.Loc == scm.LocReg {
				ctx.ProtectReg(d8.Reg)
			} else if d8.Loc == scm.LocRegPair {
				ctx.ProtectReg(d8.Reg)
				ctx.ProtectReg(d8.Reg2)
			}
			d688 = d8
			if d688.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d688)
			d689 = d688
			if d689.Loc == scm.LocImm {
				d689 = scm.JITValueDesc{Loc: scm.LocImm, Type: d689.Type, Imm: scm.NewInt(int64(uint64(d689.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d689.Reg, 32)
				ctx.EmitShrRegImm8(d689.Reg, 32)
			}
			ctx.EmitStoreToStack(d689, int32(bbs[2].PhiBase)+int32(0))
			if d8.Loc == scm.LocReg {
				ctx.UnprotectReg(d8.Reg)
			} else if d8.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d8.Reg)
				ctx.UnprotectReg(d8.Reg2)
			}
			ps690 := scm.PhiState{General: ps.General}
			ps690.OverlayValues = make([]scm.JITValueDesc, 690)
			ps690.OverlayValues[1] = d1
			ps690.OverlayValues[2] = d2
			ps690.OverlayValues[3] = d3
			ps690.OverlayValues[4] = d4
			ps690.OverlayValues[5] = d5
			ps690.OverlayValues[6] = d6
			ps690.OverlayValues[7] = d7
			ps690.OverlayValues[8] = d8
			ps690.OverlayValues[9] = d9
			ps690.OverlayValues[10] = d10
			ps690.OverlayValues[11] = d11
			ps690.OverlayValues[12] = d12
			ps690.OverlayValues[13] = d13
			ps690.OverlayValues[14] = d14
			ps690.OverlayValues[15] = d15
			ps690.OverlayValues[16] = d16
			ps690.OverlayValues[17] = d17
			ps690.OverlayValues[19] = d19
			ps690.OverlayValues[20] = d20
			ps690.OverlayValues[21] = d21
			ps690.OverlayValues[22] = d22
			ps690.OverlayValues[23] = d23
			ps690.OverlayValues[24] = d24
			ps690.OverlayValues[25] = d25
			ps690.OverlayValues[27] = d27
			ps690.OverlayValues[28] = d28
			ps690.OverlayValues[29] = d29
			ps690.OverlayValues[30] = d30
			ps690.OverlayValues[31] = d31
			ps690.OverlayValues[32] = d32
			ps690.OverlayValues[33] = d33
			ps690.OverlayValues[34] = d34
			ps690.OverlayValues[35] = d35
			ps690.OverlayValues[36] = d36
			ps690.OverlayValues[37] = d37
			ps690.OverlayValues[38] = d38
			ps690.OverlayValues[39] = d39
			ps690.OverlayValues[40] = d40
			ps690.OverlayValues[41] = d41
			ps690.OverlayValues[42] = d42
			ps690.OverlayValues[43] = d43
			ps690.OverlayValues[44] = d44
			ps690.OverlayValues[45] = d45
			ps690.OverlayValues[46] = d46
			ps690.OverlayValues[47] = d47
			ps690.OverlayValues[48] = d48
			ps690.OverlayValues[49] = d49
			ps690.OverlayValues[50] = d50
			ps690.OverlayValues[51] = d51
			ps690.OverlayValues[52] = d52
			ps690.OverlayValues[53] = d53
			ps690.OverlayValues[54] = d54
			ps690.OverlayValues[55] = d55
			ps690.OverlayValues[56] = d56
			ps690.OverlayValues[57] = d57
			ps690.OverlayValues[58] = d58
			ps690.OverlayValues[59] = d59
			ps690.OverlayValues[60] = d60
			ps690.OverlayValues[61] = d61
			ps690.OverlayValues[62] = d62
			ps690.OverlayValues[63] = d63
			ps690.OverlayValues[66] = d66
			ps690.OverlayValues[67] = d67
			ps690.OverlayValues[68] = d68
			ps690.OverlayValues[136] = d136
			ps690.OverlayValues[137] = d137
			ps690.OverlayValues[138] = d138
			ps690.OverlayValues[140] = d140
			ps690.OverlayValues[141] = d141
			ps690.OverlayValues[142] = d142
			ps690.OverlayValues[143] = d143
			ps690.OverlayValues[144] = d144
			ps690.OverlayValues[145] = d145
			ps690.OverlayValues[146] = d146
			ps690.OverlayValues[147] = d147
			ps690.OverlayValues[148] = d148
			ps690.OverlayValues[149] = d149
			ps690.OverlayValues[150] = d150
			ps690.OverlayValues[151] = d151
			ps690.OverlayValues[152] = d152
			ps690.OverlayValues[153] = d153
			ps690.OverlayValues[154] = d154
			ps690.OverlayValues[155] = d155
			ps690.OverlayValues[156] = d156
			ps690.OverlayValues[157] = d157
			ps690.OverlayValues[158] = d158
			ps690.OverlayValues[159] = d159
			ps690.OverlayValues[160] = d160
			ps690.OverlayValues[161] = d161
			ps690.OverlayValues[162] = d162
			ps690.OverlayValues[163] = d163
			ps690.OverlayValues[164] = d164
			ps690.OverlayValues[165] = d165
			ps690.OverlayValues[166] = d166
			ps690.OverlayValues[167] = d167
			ps690.OverlayValues[168] = d168
			ps690.OverlayValues[169] = d169
			ps690.OverlayValues[170] = d170
			ps690.OverlayValues[171] = d171
			ps690.OverlayValues[172] = d172
			ps690.OverlayValues[173] = d173
			ps690.OverlayValues[174] = d174
			ps690.OverlayValues[175] = d175
			ps690.OverlayValues[178] = d178
			ps690.OverlayValues[286] = d286
			ps690.OverlayValues[287] = d287
			ps690.OverlayValues[288] = d288
			ps690.OverlayValues[289] = d289
			ps690.OverlayValues[290] = d290
			ps690.OverlayValues[291] = d291
			ps690.OverlayValues[292] = d292
			ps690.OverlayValues[293] = d293
			ps690.OverlayValues[295] = d295
			ps690.OverlayValues[296] = d296
			ps690.OverlayValues[297] = d297
			ps690.OverlayValues[298] = d298
			ps690.OverlayValues[299] = d299
			ps690.OverlayValues[300] = d300
			ps690.OverlayValues[301] = d301
			ps690.OverlayValues[302] = d302
			ps690.OverlayValues[303] = d303
			ps690.OverlayValues[304] = d304
			ps690.OverlayValues[306] = d306
			ps690.OverlayValues[308] = d308
			ps690.OverlayValues[309] = d309
			ps690.OverlayValues[310] = d310
			ps690.OverlayValues[311] = d311
			ps690.OverlayValues[312] = d312
			ps690.OverlayValues[315] = d315
			ps690.OverlayValues[446] = d446
			ps690.OverlayValues[447] = d447
			ps690.OverlayValues[448] = d448
			ps690.OverlayValues[449] = d449
			ps690.OverlayValues[450] = d450
			ps690.OverlayValues[451] = d451
			ps690.OverlayValues[452] = d452
			ps690.OverlayValues[454] = d454
			ps690.OverlayValues[455] = d455
			ps690.OverlayValues[456] = d456
			ps690.OverlayValues[457] = d457
			ps690.OverlayValues[459] = d459
			ps690.OverlayValues[460] = d460
			ps690.OverlayValues[461] = d461
			ps690.OverlayValues[462] = d462
			ps690.OverlayValues[463] = d463
			ps690.OverlayValues[464] = d464
			ps690.OverlayValues[465] = d465
			ps690.OverlayValues[466] = d466
			ps690.OverlayValues[467] = d467
			ps690.OverlayValues[468] = d468
			ps690.OverlayValues[469] = d469
			ps690.OverlayValues[470] = d470
			ps690.OverlayValues[471] = d471
			ps690.OverlayValues[472] = d472
			ps690.OverlayValues[473] = d473
			ps690.OverlayValues[474] = d474
			ps690.OverlayValues[475] = d475
			ps690.OverlayValues[476] = d476
			ps690.OverlayValues[477] = d477
			ps690.OverlayValues[478] = d478
			ps690.OverlayValues[479] = d479
			ps690.OverlayValues[480] = d480
			ps690.OverlayValues[481] = d481
			ps690.OverlayValues[482] = d482
			ps690.OverlayValues[483] = d483
			ps690.OverlayValues[484] = d484
			ps690.OverlayValues[485] = d485
			ps690.OverlayValues[486] = d486
			ps690.OverlayValues[487] = d487
			ps690.OverlayValues[488] = d488
			ps690.OverlayValues[489] = d489
			ps690.OverlayValues[490] = d490
			ps690.OverlayValues[491] = d491
			ps690.OverlayValues[492] = d492
			ps690.OverlayValues[493] = d493
			ps690.OverlayValues[494] = d494
			ps690.OverlayValues[676] = d676
			ps690.OverlayValues[677] = d677
			ps690.OverlayValues[678] = d678
			ps690.OverlayValues[679] = d679
			ps690.OverlayValues[680] = d680
			ps690.OverlayValues[682] = d682
			ps690.OverlayValues[683] = d683
			ps690.OverlayValues[684] = d684
			ps690.OverlayValues[685] = d685
			ps690.OverlayValues[686] = d686
			ps690.OverlayValues[687] = d687
			ps690.OverlayValues[688] = d688
			ps690.OverlayValues[689] = d689
			ps690.PhiValues = make([]scm.JITValueDesc, 1)
			d691 = d8
			ps690.PhiValues[0] = d691
					return bbs[2].RenderPS(ps690)
				}
			ps692 := scm.PhiState{General: ps.General}
			ps692.OverlayValues = make([]scm.JITValueDesc, 692)
			ps692.OverlayValues[1] = d1
			ps692.OverlayValues[2] = d2
			ps692.OverlayValues[3] = d3
			ps692.OverlayValues[4] = d4
			ps692.OverlayValues[5] = d5
			ps692.OverlayValues[6] = d6
			ps692.OverlayValues[7] = d7
			ps692.OverlayValues[8] = d8
			ps692.OverlayValues[9] = d9
			ps692.OverlayValues[10] = d10
			ps692.OverlayValues[11] = d11
			ps692.OverlayValues[12] = d12
			ps692.OverlayValues[13] = d13
			ps692.OverlayValues[14] = d14
			ps692.OverlayValues[15] = d15
			ps692.OverlayValues[16] = d16
			ps692.OverlayValues[17] = d17
			ps692.OverlayValues[19] = d19
			ps692.OverlayValues[20] = d20
			ps692.OverlayValues[21] = d21
			ps692.OverlayValues[22] = d22
			ps692.OverlayValues[23] = d23
			ps692.OverlayValues[24] = d24
			ps692.OverlayValues[25] = d25
			ps692.OverlayValues[27] = d27
			ps692.OverlayValues[28] = d28
			ps692.OverlayValues[29] = d29
			ps692.OverlayValues[30] = d30
			ps692.OverlayValues[31] = d31
			ps692.OverlayValues[32] = d32
			ps692.OverlayValues[33] = d33
			ps692.OverlayValues[34] = d34
			ps692.OverlayValues[35] = d35
			ps692.OverlayValues[36] = d36
			ps692.OverlayValues[37] = d37
			ps692.OverlayValues[38] = d38
			ps692.OverlayValues[39] = d39
			ps692.OverlayValues[40] = d40
			ps692.OverlayValues[41] = d41
			ps692.OverlayValues[42] = d42
			ps692.OverlayValues[43] = d43
			ps692.OverlayValues[44] = d44
			ps692.OverlayValues[45] = d45
			ps692.OverlayValues[46] = d46
			ps692.OverlayValues[47] = d47
			ps692.OverlayValues[48] = d48
			ps692.OverlayValues[49] = d49
			ps692.OverlayValues[50] = d50
			ps692.OverlayValues[51] = d51
			ps692.OverlayValues[52] = d52
			ps692.OverlayValues[53] = d53
			ps692.OverlayValues[54] = d54
			ps692.OverlayValues[55] = d55
			ps692.OverlayValues[56] = d56
			ps692.OverlayValues[57] = d57
			ps692.OverlayValues[58] = d58
			ps692.OverlayValues[59] = d59
			ps692.OverlayValues[60] = d60
			ps692.OverlayValues[61] = d61
			ps692.OverlayValues[62] = d62
			ps692.OverlayValues[63] = d63
			ps692.OverlayValues[66] = d66
			ps692.OverlayValues[67] = d67
			ps692.OverlayValues[68] = d68
			ps692.OverlayValues[136] = d136
			ps692.OverlayValues[137] = d137
			ps692.OverlayValues[138] = d138
			ps692.OverlayValues[140] = d140
			ps692.OverlayValues[141] = d141
			ps692.OverlayValues[142] = d142
			ps692.OverlayValues[143] = d143
			ps692.OverlayValues[144] = d144
			ps692.OverlayValues[145] = d145
			ps692.OverlayValues[146] = d146
			ps692.OverlayValues[147] = d147
			ps692.OverlayValues[148] = d148
			ps692.OverlayValues[149] = d149
			ps692.OverlayValues[150] = d150
			ps692.OverlayValues[151] = d151
			ps692.OverlayValues[152] = d152
			ps692.OverlayValues[153] = d153
			ps692.OverlayValues[154] = d154
			ps692.OverlayValues[155] = d155
			ps692.OverlayValues[156] = d156
			ps692.OverlayValues[157] = d157
			ps692.OverlayValues[158] = d158
			ps692.OverlayValues[159] = d159
			ps692.OverlayValues[160] = d160
			ps692.OverlayValues[161] = d161
			ps692.OverlayValues[162] = d162
			ps692.OverlayValues[163] = d163
			ps692.OverlayValues[164] = d164
			ps692.OverlayValues[165] = d165
			ps692.OverlayValues[166] = d166
			ps692.OverlayValues[167] = d167
			ps692.OverlayValues[168] = d168
			ps692.OverlayValues[169] = d169
			ps692.OverlayValues[170] = d170
			ps692.OverlayValues[171] = d171
			ps692.OverlayValues[172] = d172
			ps692.OverlayValues[173] = d173
			ps692.OverlayValues[174] = d174
			ps692.OverlayValues[175] = d175
			ps692.OverlayValues[178] = d178
			ps692.OverlayValues[286] = d286
			ps692.OverlayValues[287] = d287
			ps692.OverlayValues[288] = d288
			ps692.OverlayValues[289] = d289
			ps692.OverlayValues[290] = d290
			ps692.OverlayValues[291] = d291
			ps692.OverlayValues[292] = d292
			ps692.OverlayValues[293] = d293
			ps692.OverlayValues[295] = d295
			ps692.OverlayValues[296] = d296
			ps692.OverlayValues[297] = d297
			ps692.OverlayValues[298] = d298
			ps692.OverlayValues[299] = d299
			ps692.OverlayValues[300] = d300
			ps692.OverlayValues[301] = d301
			ps692.OverlayValues[302] = d302
			ps692.OverlayValues[303] = d303
			ps692.OverlayValues[304] = d304
			ps692.OverlayValues[306] = d306
			ps692.OverlayValues[308] = d308
			ps692.OverlayValues[309] = d309
			ps692.OverlayValues[310] = d310
			ps692.OverlayValues[311] = d311
			ps692.OverlayValues[312] = d312
			ps692.OverlayValues[315] = d315
			ps692.OverlayValues[446] = d446
			ps692.OverlayValues[447] = d447
			ps692.OverlayValues[448] = d448
			ps692.OverlayValues[449] = d449
			ps692.OverlayValues[450] = d450
			ps692.OverlayValues[451] = d451
			ps692.OverlayValues[452] = d452
			ps692.OverlayValues[454] = d454
			ps692.OverlayValues[455] = d455
			ps692.OverlayValues[456] = d456
			ps692.OverlayValues[457] = d457
			ps692.OverlayValues[459] = d459
			ps692.OverlayValues[460] = d460
			ps692.OverlayValues[461] = d461
			ps692.OverlayValues[462] = d462
			ps692.OverlayValues[463] = d463
			ps692.OverlayValues[464] = d464
			ps692.OverlayValues[465] = d465
			ps692.OverlayValues[466] = d466
			ps692.OverlayValues[467] = d467
			ps692.OverlayValues[468] = d468
			ps692.OverlayValues[469] = d469
			ps692.OverlayValues[470] = d470
			ps692.OverlayValues[471] = d471
			ps692.OverlayValues[472] = d472
			ps692.OverlayValues[473] = d473
			ps692.OverlayValues[474] = d474
			ps692.OverlayValues[475] = d475
			ps692.OverlayValues[476] = d476
			ps692.OverlayValues[477] = d477
			ps692.OverlayValues[478] = d478
			ps692.OverlayValues[479] = d479
			ps692.OverlayValues[480] = d480
			ps692.OverlayValues[481] = d481
			ps692.OverlayValues[482] = d482
			ps692.OverlayValues[483] = d483
			ps692.OverlayValues[484] = d484
			ps692.OverlayValues[485] = d485
			ps692.OverlayValues[486] = d486
			ps692.OverlayValues[487] = d487
			ps692.OverlayValues[488] = d488
			ps692.OverlayValues[489] = d489
			ps692.OverlayValues[490] = d490
			ps692.OverlayValues[491] = d491
			ps692.OverlayValues[492] = d492
			ps692.OverlayValues[493] = d493
			ps692.OverlayValues[494] = d494
			ps692.OverlayValues[676] = d676
			ps692.OverlayValues[677] = d677
			ps692.OverlayValues[678] = d678
			ps692.OverlayValues[679] = d679
			ps692.OverlayValues[680] = d680
			ps692.OverlayValues[682] = d682
			ps692.OverlayValues[683] = d683
			ps692.OverlayValues[684] = d684
			ps692.OverlayValues[685] = d685
			ps692.OverlayValues[686] = d686
			ps692.OverlayValues[687] = d687
			ps692.OverlayValues[688] = d688
			ps692.OverlayValues[689] = d689
			ps692.OverlayValues[691] = d691
				return bbs[10].RenderPS(ps692)
			}
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
					d693 := ps.PhiValues[0]
					ctx.EnsureDesc(&d693)
					ctx.EmitStoreToStack(d693, int32(bbs[8].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
					d694 := ps.PhiValues[1]
					ctx.EnsureDesc(&d694)
					ctx.EmitStoreToStack(d694, int32(bbs[8].PhiBase)+int32(16))
				}
				ps.General = true
				return bbs[8].RenderPS(ps)
			}
			lbl38 := ctx.ReserveLabel()
			lbl39 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d687.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl38)
			ctx.EmitJmp(lbl39)
			ctx.MarkLabel(lbl38)
			ctx.EnsureDesc(&d8)
			if d8.Loc == scm.LocReg {
				ctx.ProtectReg(d8.Reg)
			} else if d8.Loc == scm.LocRegPair {
				ctx.ProtectReg(d8.Reg)
				ctx.ProtectReg(d8.Reg2)
			}
			d695 = d8
			if d695.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d695)
			d696 = d695
			if d696.Loc == scm.LocImm {
				d696 = scm.JITValueDesc{Loc: scm.LocImm, Type: d696.Type, Imm: scm.NewInt(int64(uint64(d696.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d696.Reg, 32)
				ctx.EmitShrRegImm8(d696.Reg, 32)
			}
			ctx.EmitStoreToStack(d696, int32(bbs[2].PhiBase)+int32(0))
			if d8.Loc == scm.LocReg {
				ctx.UnprotectReg(d8.Reg)
			} else if d8.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d8.Reg)
				ctx.UnprotectReg(d8.Reg2)
			}
			ctx.EmitJmp(lbl3)
			ctx.MarkLabel(lbl39)
			ctx.EmitJmp(lbl11)
			ps697 := scm.PhiState{General: true}
			ps697.OverlayValues = make([]scm.JITValueDesc, 697)
			ps697.OverlayValues[1] = d1
			ps697.OverlayValues[2] = d2
			ps697.OverlayValues[3] = d3
			ps697.OverlayValues[4] = d4
			ps697.OverlayValues[5] = d5
			ps697.OverlayValues[6] = d6
			ps697.OverlayValues[7] = d7
			ps697.OverlayValues[8] = d8
			ps697.OverlayValues[9] = d9
			ps697.OverlayValues[10] = d10
			ps697.OverlayValues[11] = d11
			ps697.OverlayValues[12] = d12
			ps697.OverlayValues[13] = d13
			ps697.OverlayValues[14] = d14
			ps697.OverlayValues[15] = d15
			ps697.OverlayValues[16] = d16
			ps697.OverlayValues[17] = d17
			ps697.OverlayValues[19] = d19
			ps697.OverlayValues[20] = d20
			ps697.OverlayValues[21] = d21
			ps697.OverlayValues[22] = d22
			ps697.OverlayValues[23] = d23
			ps697.OverlayValues[24] = d24
			ps697.OverlayValues[25] = d25
			ps697.OverlayValues[27] = d27
			ps697.OverlayValues[28] = d28
			ps697.OverlayValues[29] = d29
			ps697.OverlayValues[30] = d30
			ps697.OverlayValues[31] = d31
			ps697.OverlayValues[32] = d32
			ps697.OverlayValues[33] = d33
			ps697.OverlayValues[34] = d34
			ps697.OverlayValues[35] = d35
			ps697.OverlayValues[36] = d36
			ps697.OverlayValues[37] = d37
			ps697.OverlayValues[38] = d38
			ps697.OverlayValues[39] = d39
			ps697.OverlayValues[40] = d40
			ps697.OverlayValues[41] = d41
			ps697.OverlayValues[42] = d42
			ps697.OverlayValues[43] = d43
			ps697.OverlayValues[44] = d44
			ps697.OverlayValues[45] = d45
			ps697.OverlayValues[46] = d46
			ps697.OverlayValues[47] = d47
			ps697.OverlayValues[48] = d48
			ps697.OverlayValues[49] = d49
			ps697.OverlayValues[50] = d50
			ps697.OverlayValues[51] = d51
			ps697.OverlayValues[52] = d52
			ps697.OverlayValues[53] = d53
			ps697.OverlayValues[54] = d54
			ps697.OverlayValues[55] = d55
			ps697.OverlayValues[56] = d56
			ps697.OverlayValues[57] = d57
			ps697.OverlayValues[58] = d58
			ps697.OverlayValues[59] = d59
			ps697.OverlayValues[60] = d60
			ps697.OverlayValues[61] = d61
			ps697.OverlayValues[62] = d62
			ps697.OverlayValues[63] = d63
			ps697.OverlayValues[66] = d66
			ps697.OverlayValues[67] = d67
			ps697.OverlayValues[68] = d68
			ps697.OverlayValues[136] = d136
			ps697.OverlayValues[137] = d137
			ps697.OverlayValues[138] = d138
			ps697.OverlayValues[140] = d140
			ps697.OverlayValues[141] = d141
			ps697.OverlayValues[142] = d142
			ps697.OverlayValues[143] = d143
			ps697.OverlayValues[144] = d144
			ps697.OverlayValues[145] = d145
			ps697.OverlayValues[146] = d146
			ps697.OverlayValues[147] = d147
			ps697.OverlayValues[148] = d148
			ps697.OverlayValues[149] = d149
			ps697.OverlayValues[150] = d150
			ps697.OverlayValues[151] = d151
			ps697.OverlayValues[152] = d152
			ps697.OverlayValues[153] = d153
			ps697.OverlayValues[154] = d154
			ps697.OverlayValues[155] = d155
			ps697.OverlayValues[156] = d156
			ps697.OverlayValues[157] = d157
			ps697.OverlayValues[158] = d158
			ps697.OverlayValues[159] = d159
			ps697.OverlayValues[160] = d160
			ps697.OverlayValues[161] = d161
			ps697.OverlayValues[162] = d162
			ps697.OverlayValues[163] = d163
			ps697.OverlayValues[164] = d164
			ps697.OverlayValues[165] = d165
			ps697.OverlayValues[166] = d166
			ps697.OverlayValues[167] = d167
			ps697.OverlayValues[168] = d168
			ps697.OverlayValues[169] = d169
			ps697.OverlayValues[170] = d170
			ps697.OverlayValues[171] = d171
			ps697.OverlayValues[172] = d172
			ps697.OverlayValues[173] = d173
			ps697.OverlayValues[174] = d174
			ps697.OverlayValues[175] = d175
			ps697.OverlayValues[178] = d178
			ps697.OverlayValues[286] = d286
			ps697.OverlayValues[287] = d287
			ps697.OverlayValues[288] = d288
			ps697.OverlayValues[289] = d289
			ps697.OverlayValues[290] = d290
			ps697.OverlayValues[291] = d291
			ps697.OverlayValues[292] = d292
			ps697.OverlayValues[293] = d293
			ps697.OverlayValues[295] = d295
			ps697.OverlayValues[296] = d296
			ps697.OverlayValues[297] = d297
			ps697.OverlayValues[298] = d298
			ps697.OverlayValues[299] = d299
			ps697.OverlayValues[300] = d300
			ps697.OverlayValues[301] = d301
			ps697.OverlayValues[302] = d302
			ps697.OverlayValues[303] = d303
			ps697.OverlayValues[304] = d304
			ps697.OverlayValues[306] = d306
			ps697.OverlayValues[308] = d308
			ps697.OverlayValues[309] = d309
			ps697.OverlayValues[310] = d310
			ps697.OverlayValues[311] = d311
			ps697.OverlayValues[312] = d312
			ps697.OverlayValues[315] = d315
			ps697.OverlayValues[446] = d446
			ps697.OverlayValues[447] = d447
			ps697.OverlayValues[448] = d448
			ps697.OverlayValues[449] = d449
			ps697.OverlayValues[450] = d450
			ps697.OverlayValues[451] = d451
			ps697.OverlayValues[452] = d452
			ps697.OverlayValues[454] = d454
			ps697.OverlayValues[455] = d455
			ps697.OverlayValues[456] = d456
			ps697.OverlayValues[457] = d457
			ps697.OverlayValues[459] = d459
			ps697.OverlayValues[460] = d460
			ps697.OverlayValues[461] = d461
			ps697.OverlayValues[462] = d462
			ps697.OverlayValues[463] = d463
			ps697.OverlayValues[464] = d464
			ps697.OverlayValues[465] = d465
			ps697.OverlayValues[466] = d466
			ps697.OverlayValues[467] = d467
			ps697.OverlayValues[468] = d468
			ps697.OverlayValues[469] = d469
			ps697.OverlayValues[470] = d470
			ps697.OverlayValues[471] = d471
			ps697.OverlayValues[472] = d472
			ps697.OverlayValues[473] = d473
			ps697.OverlayValues[474] = d474
			ps697.OverlayValues[475] = d475
			ps697.OverlayValues[476] = d476
			ps697.OverlayValues[477] = d477
			ps697.OverlayValues[478] = d478
			ps697.OverlayValues[479] = d479
			ps697.OverlayValues[480] = d480
			ps697.OverlayValues[481] = d481
			ps697.OverlayValues[482] = d482
			ps697.OverlayValues[483] = d483
			ps697.OverlayValues[484] = d484
			ps697.OverlayValues[485] = d485
			ps697.OverlayValues[486] = d486
			ps697.OverlayValues[487] = d487
			ps697.OverlayValues[488] = d488
			ps697.OverlayValues[489] = d489
			ps697.OverlayValues[490] = d490
			ps697.OverlayValues[491] = d491
			ps697.OverlayValues[492] = d492
			ps697.OverlayValues[493] = d493
			ps697.OverlayValues[494] = d494
			ps697.OverlayValues[676] = d676
			ps697.OverlayValues[677] = d677
			ps697.OverlayValues[678] = d678
			ps697.OverlayValues[679] = d679
			ps697.OverlayValues[680] = d680
			ps697.OverlayValues[682] = d682
			ps697.OverlayValues[683] = d683
			ps697.OverlayValues[684] = d684
			ps697.OverlayValues[685] = d685
			ps697.OverlayValues[686] = d686
			ps697.OverlayValues[687] = d687
			ps697.OverlayValues[688] = d688
			ps697.OverlayValues[689] = d689
			ps697.OverlayValues[691] = d691
			ps697.OverlayValues[693] = d693
			ps697.OverlayValues[694] = d694
			ps697.OverlayValues[695] = d695
			ps697.OverlayValues[696] = d696
			ps697.PhiValues = make([]scm.JITValueDesc, 1)
			d699 = d8
			ps697.PhiValues[0] = d699
			ps698 := scm.PhiState{General: true}
			ps698.OverlayValues = make([]scm.JITValueDesc, 700)
			ps698.OverlayValues[1] = d1
			ps698.OverlayValues[2] = d2
			ps698.OverlayValues[3] = d3
			ps698.OverlayValues[4] = d4
			ps698.OverlayValues[5] = d5
			ps698.OverlayValues[6] = d6
			ps698.OverlayValues[7] = d7
			ps698.OverlayValues[8] = d8
			ps698.OverlayValues[9] = d9
			ps698.OverlayValues[10] = d10
			ps698.OverlayValues[11] = d11
			ps698.OverlayValues[12] = d12
			ps698.OverlayValues[13] = d13
			ps698.OverlayValues[14] = d14
			ps698.OverlayValues[15] = d15
			ps698.OverlayValues[16] = d16
			ps698.OverlayValues[17] = d17
			ps698.OverlayValues[19] = d19
			ps698.OverlayValues[20] = d20
			ps698.OverlayValues[21] = d21
			ps698.OverlayValues[22] = d22
			ps698.OverlayValues[23] = d23
			ps698.OverlayValues[24] = d24
			ps698.OverlayValues[25] = d25
			ps698.OverlayValues[27] = d27
			ps698.OverlayValues[28] = d28
			ps698.OverlayValues[29] = d29
			ps698.OverlayValues[30] = d30
			ps698.OverlayValues[31] = d31
			ps698.OverlayValues[32] = d32
			ps698.OverlayValues[33] = d33
			ps698.OverlayValues[34] = d34
			ps698.OverlayValues[35] = d35
			ps698.OverlayValues[36] = d36
			ps698.OverlayValues[37] = d37
			ps698.OverlayValues[38] = d38
			ps698.OverlayValues[39] = d39
			ps698.OverlayValues[40] = d40
			ps698.OverlayValues[41] = d41
			ps698.OverlayValues[42] = d42
			ps698.OverlayValues[43] = d43
			ps698.OverlayValues[44] = d44
			ps698.OverlayValues[45] = d45
			ps698.OverlayValues[46] = d46
			ps698.OverlayValues[47] = d47
			ps698.OverlayValues[48] = d48
			ps698.OverlayValues[49] = d49
			ps698.OverlayValues[50] = d50
			ps698.OverlayValues[51] = d51
			ps698.OverlayValues[52] = d52
			ps698.OverlayValues[53] = d53
			ps698.OverlayValues[54] = d54
			ps698.OverlayValues[55] = d55
			ps698.OverlayValues[56] = d56
			ps698.OverlayValues[57] = d57
			ps698.OverlayValues[58] = d58
			ps698.OverlayValues[59] = d59
			ps698.OverlayValues[60] = d60
			ps698.OverlayValues[61] = d61
			ps698.OverlayValues[62] = d62
			ps698.OverlayValues[63] = d63
			ps698.OverlayValues[66] = d66
			ps698.OverlayValues[67] = d67
			ps698.OverlayValues[68] = d68
			ps698.OverlayValues[136] = d136
			ps698.OverlayValues[137] = d137
			ps698.OverlayValues[138] = d138
			ps698.OverlayValues[140] = d140
			ps698.OverlayValues[141] = d141
			ps698.OverlayValues[142] = d142
			ps698.OverlayValues[143] = d143
			ps698.OverlayValues[144] = d144
			ps698.OverlayValues[145] = d145
			ps698.OverlayValues[146] = d146
			ps698.OverlayValues[147] = d147
			ps698.OverlayValues[148] = d148
			ps698.OverlayValues[149] = d149
			ps698.OverlayValues[150] = d150
			ps698.OverlayValues[151] = d151
			ps698.OverlayValues[152] = d152
			ps698.OverlayValues[153] = d153
			ps698.OverlayValues[154] = d154
			ps698.OverlayValues[155] = d155
			ps698.OverlayValues[156] = d156
			ps698.OverlayValues[157] = d157
			ps698.OverlayValues[158] = d158
			ps698.OverlayValues[159] = d159
			ps698.OverlayValues[160] = d160
			ps698.OverlayValues[161] = d161
			ps698.OverlayValues[162] = d162
			ps698.OverlayValues[163] = d163
			ps698.OverlayValues[164] = d164
			ps698.OverlayValues[165] = d165
			ps698.OverlayValues[166] = d166
			ps698.OverlayValues[167] = d167
			ps698.OverlayValues[168] = d168
			ps698.OverlayValues[169] = d169
			ps698.OverlayValues[170] = d170
			ps698.OverlayValues[171] = d171
			ps698.OverlayValues[172] = d172
			ps698.OverlayValues[173] = d173
			ps698.OverlayValues[174] = d174
			ps698.OverlayValues[175] = d175
			ps698.OverlayValues[178] = d178
			ps698.OverlayValues[286] = d286
			ps698.OverlayValues[287] = d287
			ps698.OverlayValues[288] = d288
			ps698.OverlayValues[289] = d289
			ps698.OverlayValues[290] = d290
			ps698.OverlayValues[291] = d291
			ps698.OverlayValues[292] = d292
			ps698.OverlayValues[293] = d293
			ps698.OverlayValues[295] = d295
			ps698.OverlayValues[296] = d296
			ps698.OverlayValues[297] = d297
			ps698.OverlayValues[298] = d298
			ps698.OverlayValues[299] = d299
			ps698.OverlayValues[300] = d300
			ps698.OverlayValues[301] = d301
			ps698.OverlayValues[302] = d302
			ps698.OverlayValues[303] = d303
			ps698.OverlayValues[304] = d304
			ps698.OverlayValues[306] = d306
			ps698.OverlayValues[308] = d308
			ps698.OverlayValues[309] = d309
			ps698.OverlayValues[310] = d310
			ps698.OverlayValues[311] = d311
			ps698.OverlayValues[312] = d312
			ps698.OverlayValues[315] = d315
			ps698.OverlayValues[446] = d446
			ps698.OverlayValues[447] = d447
			ps698.OverlayValues[448] = d448
			ps698.OverlayValues[449] = d449
			ps698.OverlayValues[450] = d450
			ps698.OverlayValues[451] = d451
			ps698.OverlayValues[452] = d452
			ps698.OverlayValues[454] = d454
			ps698.OverlayValues[455] = d455
			ps698.OverlayValues[456] = d456
			ps698.OverlayValues[457] = d457
			ps698.OverlayValues[459] = d459
			ps698.OverlayValues[460] = d460
			ps698.OverlayValues[461] = d461
			ps698.OverlayValues[462] = d462
			ps698.OverlayValues[463] = d463
			ps698.OverlayValues[464] = d464
			ps698.OverlayValues[465] = d465
			ps698.OverlayValues[466] = d466
			ps698.OverlayValues[467] = d467
			ps698.OverlayValues[468] = d468
			ps698.OverlayValues[469] = d469
			ps698.OverlayValues[470] = d470
			ps698.OverlayValues[471] = d471
			ps698.OverlayValues[472] = d472
			ps698.OverlayValues[473] = d473
			ps698.OverlayValues[474] = d474
			ps698.OverlayValues[475] = d475
			ps698.OverlayValues[476] = d476
			ps698.OverlayValues[477] = d477
			ps698.OverlayValues[478] = d478
			ps698.OverlayValues[479] = d479
			ps698.OverlayValues[480] = d480
			ps698.OverlayValues[481] = d481
			ps698.OverlayValues[482] = d482
			ps698.OverlayValues[483] = d483
			ps698.OverlayValues[484] = d484
			ps698.OverlayValues[485] = d485
			ps698.OverlayValues[486] = d486
			ps698.OverlayValues[487] = d487
			ps698.OverlayValues[488] = d488
			ps698.OverlayValues[489] = d489
			ps698.OverlayValues[490] = d490
			ps698.OverlayValues[491] = d491
			ps698.OverlayValues[492] = d492
			ps698.OverlayValues[493] = d493
			ps698.OverlayValues[494] = d494
			ps698.OverlayValues[676] = d676
			ps698.OverlayValues[677] = d677
			ps698.OverlayValues[678] = d678
			ps698.OverlayValues[679] = d679
			ps698.OverlayValues[680] = d680
			ps698.OverlayValues[682] = d682
			ps698.OverlayValues[683] = d683
			ps698.OverlayValues[684] = d684
			ps698.OverlayValues[685] = d685
			ps698.OverlayValues[686] = d686
			ps698.OverlayValues[687] = d687
			ps698.OverlayValues[688] = d688
			ps698.OverlayValues[689] = d689
			ps698.OverlayValues[691] = d691
			ps698.OverlayValues[693] = d693
			ps698.OverlayValues[694] = d694
			ps698.OverlayValues[695] = d695
			ps698.OverlayValues[696] = d696
			ps698.OverlayValues[699] = d699
			snap700 := d1
			snap701 := d2
			snap702 := d3
			snap703 := d4
			snap704 := d5
			snap705 := d6
			snap706 := d7
			snap707 := d8
			snap708 := d9
			snap709 := d10
			snap710 := d11
			snap711 := d12
			snap712 := d13
			snap713 := d14
			snap714 := d15
			snap715 := d16
			snap716 := d17
			snap717 := d19
			snap718 := d20
			snap719 := d21
			snap720 := d22
			snap721 := d23
			snap722 := d24
			snap723 := d25
			snap724 := d27
			snap725 := d28
			snap726 := d29
			snap727 := d30
			snap728 := d31
			snap729 := d32
			snap730 := d33
			snap731 := d34
			snap732 := d35
			snap733 := d36
			snap734 := d37
			snap735 := d38
			snap736 := d39
			snap737 := d40
			snap738 := d41
			snap739 := d42
			snap740 := d43
			snap741 := d44
			snap742 := d45
			snap743 := d46
			snap744 := d47
			snap745 := d48
			snap746 := d49
			snap747 := d50
			snap748 := d51
			snap749 := d52
			snap750 := d53
			snap751 := d54
			snap752 := d55
			snap753 := d56
			snap754 := d57
			snap755 := d58
			snap756 := d59
			snap757 := d60
			snap758 := d61
			snap759 := d62
			snap760 := d63
			snap761 := d66
			snap762 := d67
			snap763 := d68
			snap764 := d136
			snap765 := d137
			snap766 := d138
			snap767 := d140
			snap768 := d141
			snap769 := d142
			snap770 := d143
			snap771 := d144
			snap772 := d145
			snap773 := d146
			snap774 := d147
			snap775 := d148
			snap776 := d149
			snap777 := d150
			snap778 := d151
			snap779 := d152
			snap780 := d153
			snap781 := d154
			snap782 := d155
			snap783 := d156
			snap784 := d157
			snap785 := d158
			snap786 := d159
			snap787 := d160
			snap788 := d161
			snap789 := d162
			snap790 := d163
			snap791 := d164
			snap792 := d165
			snap793 := d166
			snap794 := d167
			snap795 := d168
			snap796 := d169
			snap797 := d170
			snap798 := d171
			snap799 := d172
			snap800 := d173
			snap801 := d174
			snap802 := d175
			snap803 := d178
			snap804 := d286
			snap805 := d287
			snap806 := d288
			snap807 := d289
			snap808 := d290
			snap809 := d291
			snap810 := d292
			snap811 := d293
			snap812 := d295
			snap813 := d296
			snap814 := d297
			snap815 := d298
			snap816 := d299
			snap817 := d300
			snap818 := d301
			snap819 := d302
			snap820 := d303
			snap821 := d304
			snap822 := d306
			snap823 := d308
			snap824 := d309
			snap825 := d310
			snap826 := d311
			snap827 := d312
			snap828 := d315
			snap829 := d446
			snap830 := d447
			snap831 := d448
			snap832 := d449
			snap833 := d450
			snap834 := d451
			snap835 := d452
			snap836 := d454
			snap837 := d455
			snap838 := d456
			snap839 := d457
			snap840 := d459
			snap841 := d460
			snap842 := d461
			snap843 := d462
			snap844 := d463
			snap845 := d464
			snap846 := d465
			snap847 := d466
			snap848 := d467
			snap849 := d468
			snap850 := d469
			snap851 := d470
			snap852 := d471
			snap853 := d472
			snap854 := d473
			snap855 := d474
			snap856 := d475
			snap857 := d476
			snap858 := d477
			snap859 := d478
			snap860 := d479
			snap861 := d480
			snap862 := d481
			snap863 := d482
			snap864 := d483
			snap865 := d484
			snap866 := d485
			snap867 := d486
			snap868 := d487
			snap869 := d488
			snap870 := d489
			snap871 := d490
			snap872 := d491
			snap873 := d492
			snap874 := d493
			snap875 := d494
			snap876 := d676
			snap877 := d677
			snap878 := d678
			snap879 := d679
			snap880 := d680
			snap881 := d682
			snap882 := d683
			snap883 := d684
			snap884 := d685
			snap885 := d686
			snap886 := d687
			snap887 := d688
			snap888 := d689
			snap889 := d691
			snap890 := d693
			snap891 := d694
			snap892 := d695
			snap893 := d696
			snap894 := d699
			alloc895 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps697)
			}
			ctx.RestoreAllocState(alloc895)
			d1 = snap700
			d2 = snap701
			d3 = snap702
			d4 = snap703
			d5 = snap704
			d6 = snap705
			d7 = snap706
			d8 = snap707
			d9 = snap708
			d10 = snap709
			d11 = snap710
			d12 = snap711
			d13 = snap712
			d14 = snap713
			d15 = snap714
			d16 = snap715
			d17 = snap716
			d19 = snap717
			d20 = snap718
			d21 = snap719
			d22 = snap720
			d23 = snap721
			d24 = snap722
			d25 = snap723
			d27 = snap724
			d28 = snap725
			d29 = snap726
			d30 = snap727
			d31 = snap728
			d32 = snap729
			d33 = snap730
			d34 = snap731
			d35 = snap732
			d36 = snap733
			d37 = snap734
			d38 = snap735
			d39 = snap736
			d40 = snap737
			d41 = snap738
			d42 = snap739
			d43 = snap740
			d44 = snap741
			d45 = snap742
			d46 = snap743
			d47 = snap744
			d48 = snap745
			d49 = snap746
			d50 = snap747
			d51 = snap748
			d52 = snap749
			d53 = snap750
			d54 = snap751
			d55 = snap752
			d56 = snap753
			d57 = snap754
			d58 = snap755
			d59 = snap756
			d60 = snap757
			d61 = snap758
			d62 = snap759
			d63 = snap760
			d66 = snap761
			d67 = snap762
			d68 = snap763
			d136 = snap764
			d137 = snap765
			d138 = snap766
			d140 = snap767
			d141 = snap768
			d142 = snap769
			d143 = snap770
			d144 = snap771
			d145 = snap772
			d146 = snap773
			d147 = snap774
			d148 = snap775
			d149 = snap776
			d150 = snap777
			d151 = snap778
			d152 = snap779
			d153 = snap780
			d154 = snap781
			d155 = snap782
			d156 = snap783
			d157 = snap784
			d158 = snap785
			d159 = snap786
			d160 = snap787
			d161 = snap788
			d162 = snap789
			d163 = snap790
			d164 = snap791
			d165 = snap792
			d166 = snap793
			d167 = snap794
			d168 = snap795
			d169 = snap796
			d170 = snap797
			d171 = snap798
			d172 = snap799
			d173 = snap800
			d174 = snap801
			d175 = snap802
			d178 = snap803
			d286 = snap804
			d287 = snap805
			d288 = snap806
			d289 = snap807
			d290 = snap808
			d291 = snap809
			d292 = snap810
			d293 = snap811
			d295 = snap812
			d296 = snap813
			d297 = snap814
			d298 = snap815
			d299 = snap816
			d300 = snap817
			d301 = snap818
			d302 = snap819
			d303 = snap820
			d304 = snap821
			d306 = snap822
			d308 = snap823
			d309 = snap824
			d310 = snap825
			d311 = snap826
			d312 = snap827
			d315 = snap828
			d446 = snap829
			d447 = snap830
			d448 = snap831
			d449 = snap832
			d450 = snap833
			d451 = snap834
			d452 = snap835
			d454 = snap836
			d455 = snap837
			d456 = snap838
			d457 = snap839
			d459 = snap840
			d460 = snap841
			d461 = snap842
			d462 = snap843
			d463 = snap844
			d464 = snap845
			d465 = snap846
			d466 = snap847
			d467 = snap848
			d468 = snap849
			d469 = snap850
			d470 = snap851
			d471 = snap852
			d472 = snap853
			d473 = snap854
			d474 = snap855
			d475 = snap856
			d476 = snap857
			d477 = snap858
			d478 = snap859
			d479 = snap860
			d480 = snap861
			d481 = snap862
			d482 = snap863
			d483 = snap864
			d484 = snap865
			d485 = snap866
			d486 = snap867
			d487 = snap868
			d488 = snap869
			d489 = snap870
			d490 = snap871
			d491 = snap872
			d492 = snap873
			d493 = snap874
			d494 = snap875
			d676 = snap876
			d677 = snap877
			d678 = snap878
			d679 = snap879
			d680 = snap880
			d682 = snap881
			d683 = snap882
			d684 = snap883
			d685 = snap884
			d686 = snap885
			d687 = snap886
			d688 = snap887
			d689 = snap888
			d691 = snap889
			d693 = snap890
			d694 = snap891
			d695 = snap892
			d696 = snap893
			d699 = snap894
			if !bbs[10].Rendered {
				return bbs[10].RenderPS(ps698)
			}
			return result
			ctx.FreeDesc(&d686)
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
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
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
			if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != scm.LocNone {
				d173 = ps.OverlayValues[173]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
			}
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
				d178 = ps.OverlayValues[178]
			}
			if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != scm.LocNone {
				d286 = ps.OverlayValues[286]
			}
			if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != scm.LocNone {
				d287 = ps.OverlayValues[287]
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
			if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != scm.LocNone {
				d293 = ps.OverlayValues[293]
			}
			if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
				d295 = ps.OverlayValues[295]
			}
			if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != scm.LocNone {
				d296 = ps.OverlayValues[296]
			}
			if len(ps.OverlayValues) > 297 && ps.OverlayValues[297].Loc != scm.LocNone {
				d297 = ps.OverlayValues[297]
			}
			if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != scm.LocNone {
				d298 = ps.OverlayValues[298]
			}
			if len(ps.OverlayValues) > 299 && ps.OverlayValues[299].Loc != scm.LocNone {
				d299 = ps.OverlayValues[299]
			}
			if len(ps.OverlayValues) > 300 && ps.OverlayValues[300].Loc != scm.LocNone {
				d300 = ps.OverlayValues[300]
			}
			if len(ps.OverlayValues) > 301 && ps.OverlayValues[301].Loc != scm.LocNone {
				d301 = ps.OverlayValues[301]
			}
			if len(ps.OverlayValues) > 302 && ps.OverlayValues[302].Loc != scm.LocNone {
				d302 = ps.OverlayValues[302]
			}
			if len(ps.OverlayValues) > 303 && ps.OverlayValues[303].Loc != scm.LocNone {
				d303 = ps.OverlayValues[303]
			}
			if len(ps.OverlayValues) > 304 && ps.OverlayValues[304].Loc != scm.LocNone {
				d304 = ps.OverlayValues[304]
			}
			if len(ps.OverlayValues) > 306 && ps.OverlayValues[306].Loc != scm.LocNone {
				d306 = ps.OverlayValues[306]
			}
			if len(ps.OverlayValues) > 308 && ps.OverlayValues[308].Loc != scm.LocNone {
				d308 = ps.OverlayValues[308]
			}
			if len(ps.OverlayValues) > 309 && ps.OverlayValues[309].Loc != scm.LocNone {
				d309 = ps.OverlayValues[309]
			}
			if len(ps.OverlayValues) > 310 && ps.OverlayValues[310].Loc != scm.LocNone {
				d310 = ps.OverlayValues[310]
			}
			if len(ps.OverlayValues) > 311 && ps.OverlayValues[311].Loc != scm.LocNone {
				d311 = ps.OverlayValues[311]
			}
			if len(ps.OverlayValues) > 312 && ps.OverlayValues[312].Loc != scm.LocNone {
				d312 = ps.OverlayValues[312]
			}
			if len(ps.OverlayValues) > 315 && ps.OverlayValues[315].Loc != scm.LocNone {
				d315 = ps.OverlayValues[315]
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
			if len(ps.OverlayValues) > 459 && ps.OverlayValues[459].Loc != scm.LocNone {
				d459 = ps.OverlayValues[459]
			}
			if len(ps.OverlayValues) > 460 && ps.OverlayValues[460].Loc != scm.LocNone {
				d460 = ps.OverlayValues[460]
			}
			if len(ps.OverlayValues) > 461 && ps.OverlayValues[461].Loc != scm.LocNone {
				d461 = ps.OverlayValues[461]
			}
			if len(ps.OverlayValues) > 462 && ps.OverlayValues[462].Loc != scm.LocNone {
				d462 = ps.OverlayValues[462]
			}
			if len(ps.OverlayValues) > 463 && ps.OverlayValues[463].Loc != scm.LocNone {
				d463 = ps.OverlayValues[463]
			}
			if len(ps.OverlayValues) > 464 && ps.OverlayValues[464].Loc != scm.LocNone {
				d464 = ps.OverlayValues[464]
			}
			if len(ps.OverlayValues) > 465 && ps.OverlayValues[465].Loc != scm.LocNone {
				d465 = ps.OverlayValues[465]
			}
			if len(ps.OverlayValues) > 466 && ps.OverlayValues[466].Loc != scm.LocNone {
				d466 = ps.OverlayValues[466]
			}
			if len(ps.OverlayValues) > 467 && ps.OverlayValues[467].Loc != scm.LocNone {
				d467 = ps.OverlayValues[467]
			}
			if len(ps.OverlayValues) > 468 && ps.OverlayValues[468].Loc != scm.LocNone {
				d468 = ps.OverlayValues[468]
			}
			if len(ps.OverlayValues) > 469 && ps.OverlayValues[469].Loc != scm.LocNone {
				d469 = ps.OverlayValues[469]
			}
			if len(ps.OverlayValues) > 470 && ps.OverlayValues[470].Loc != scm.LocNone {
				d470 = ps.OverlayValues[470]
			}
			if len(ps.OverlayValues) > 471 && ps.OverlayValues[471].Loc != scm.LocNone {
				d471 = ps.OverlayValues[471]
			}
			if len(ps.OverlayValues) > 472 && ps.OverlayValues[472].Loc != scm.LocNone {
				d472 = ps.OverlayValues[472]
			}
			if len(ps.OverlayValues) > 473 && ps.OverlayValues[473].Loc != scm.LocNone {
				d473 = ps.OverlayValues[473]
			}
			if len(ps.OverlayValues) > 474 && ps.OverlayValues[474].Loc != scm.LocNone {
				d474 = ps.OverlayValues[474]
			}
			if len(ps.OverlayValues) > 475 && ps.OverlayValues[475].Loc != scm.LocNone {
				d475 = ps.OverlayValues[475]
			}
			if len(ps.OverlayValues) > 476 && ps.OverlayValues[476].Loc != scm.LocNone {
				d476 = ps.OverlayValues[476]
			}
			if len(ps.OverlayValues) > 477 && ps.OverlayValues[477].Loc != scm.LocNone {
				d477 = ps.OverlayValues[477]
			}
			if len(ps.OverlayValues) > 478 && ps.OverlayValues[478].Loc != scm.LocNone {
				d478 = ps.OverlayValues[478]
			}
			if len(ps.OverlayValues) > 479 && ps.OverlayValues[479].Loc != scm.LocNone {
				d479 = ps.OverlayValues[479]
			}
			if len(ps.OverlayValues) > 480 && ps.OverlayValues[480].Loc != scm.LocNone {
				d480 = ps.OverlayValues[480]
			}
			if len(ps.OverlayValues) > 481 && ps.OverlayValues[481].Loc != scm.LocNone {
				d481 = ps.OverlayValues[481]
			}
			if len(ps.OverlayValues) > 482 && ps.OverlayValues[482].Loc != scm.LocNone {
				d482 = ps.OverlayValues[482]
			}
			if len(ps.OverlayValues) > 483 && ps.OverlayValues[483].Loc != scm.LocNone {
				d483 = ps.OverlayValues[483]
			}
			if len(ps.OverlayValues) > 484 && ps.OverlayValues[484].Loc != scm.LocNone {
				d484 = ps.OverlayValues[484]
			}
			if len(ps.OverlayValues) > 485 && ps.OverlayValues[485].Loc != scm.LocNone {
				d485 = ps.OverlayValues[485]
			}
			if len(ps.OverlayValues) > 486 && ps.OverlayValues[486].Loc != scm.LocNone {
				d486 = ps.OverlayValues[486]
			}
			if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != scm.LocNone {
				d487 = ps.OverlayValues[487]
			}
			if len(ps.OverlayValues) > 488 && ps.OverlayValues[488].Loc != scm.LocNone {
				d488 = ps.OverlayValues[488]
			}
			if len(ps.OverlayValues) > 489 && ps.OverlayValues[489].Loc != scm.LocNone {
				d489 = ps.OverlayValues[489]
			}
			if len(ps.OverlayValues) > 490 && ps.OverlayValues[490].Loc != scm.LocNone {
				d490 = ps.OverlayValues[490]
			}
			if len(ps.OverlayValues) > 491 && ps.OverlayValues[491].Loc != scm.LocNone {
				d491 = ps.OverlayValues[491]
			}
			if len(ps.OverlayValues) > 492 && ps.OverlayValues[492].Loc != scm.LocNone {
				d492 = ps.OverlayValues[492]
			}
			if len(ps.OverlayValues) > 493 && ps.OverlayValues[493].Loc != scm.LocNone {
				d493 = ps.OverlayValues[493]
			}
			if len(ps.OverlayValues) > 494 && ps.OverlayValues[494].Loc != scm.LocNone {
				d494 = ps.OverlayValues[494]
			}
			if len(ps.OverlayValues) > 676 && ps.OverlayValues[676].Loc != scm.LocNone {
				d676 = ps.OverlayValues[676]
			}
			if len(ps.OverlayValues) > 677 && ps.OverlayValues[677].Loc != scm.LocNone {
				d677 = ps.OverlayValues[677]
			}
			if len(ps.OverlayValues) > 678 && ps.OverlayValues[678].Loc != scm.LocNone {
				d678 = ps.OverlayValues[678]
			}
			if len(ps.OverlayValues) > 679 && ps.OverlayValues[679].Loc != scm.LocNone {
				d679 = ps.OverlayValues[679]
			}
			if len(ps.OverlayValues) > 680 && ps.OverlayValues[680].Loc != scm.LocNone {
				d680 = ps.OverlayValues[680]
			}
			if len(ps.OverlayValues) > 682 && ps.OverlayValues[682].Loc != scm.LocNone {
				d682 = ps.OverlayValues[682]
			}
			if len(ps.OverlayValues) > 683 && ps.OverlayValues[683].Loc != scm.LocNone {
				d683 = ps.OverlayValues[683]
			}
			if len(ps.OverlayValues) > 684 && ps.OverlayValues[684].Loc != scm.LocNone {
				d684 = ps.OverlayValues[684]
			}
			if len(ps.OverlayValues) > 685 && ps.OverlayValues[685].Loc != scm.LocNone {
				d685 = ps.OverlayValues[685]
			}
			if len(ps.OverlayValues) > 686 && ps.OverlayValues[686].Loc != scm.LocNone {
				d686 = ps.OverlayValues[686]
			}
			if len(ps.OverlayValues) > 687 && ps.OverlayValues[687].Loc != scm.LocNone {
				d687 = ps.OverlayValues[687]
			}
			if len(ps.OverlayValues) > 688 && ps.OverlayValues[688].Loc != scm.LocNone {
				d688 = ps.OverlayValues[688]
			}
			if len(ps.OverlayValues) > 689 && ps.OverlayValues[689].Loc != scm.LocNone {
				d689 = ps.OverlayValues[689]
			}
			if len(ps.OverlayValues) > 691 && ps.OverlayValues[691].Loc != scm.LocNone {
				d691 = ps.OverlayValues[691]
			}
			if len(ps.OverlayValues) > 693 && ps.OverlayValues[693].Loc != scm.LocNone {
				d693 = ps.OverlayValues[693]
			}
			if len(ps.OverlayValues) > 694 && ps.OverlayValues[694].Loc != scm.LocNone {
				d694 = ps.OverlayValues[694]
			}
			if len(ps.OverlayValues) > 695 && ps.OverlayValues[695].Loc != scm.LocNone {
				d695 = ps.OverlayValues[695]
			}
			if len(ps.OverlayValues) > 696 && ps.OverlayValues[696].Loc != scm.LocNone {
				d696 = ps.OverlayValues[696]
			}
			if len(ps.OverlayValues) > 699 && ps.OverlayValues[699].Loc != scm.LocNone {
				d699 = ps.OverlayValues[699]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d5)
			if d5.Loc == scm.LocReg {
				ctx.ProtectReg(d5.Reg)
			} else if d5.Loc == scm.LocRegPair {
				ctx.ProtectReg(d5.Reg)
				ctx.ProtectReg(d5.Reg2)
			}
			ctx.EnsureDesc(&d7)
			if d7.Loc == scm.LocReg {
				ctx.ProtectReg(d7.Reg)
			} else if d7.Loc == scm.LocRegPair {
				ctx.ProtectReg(d7.Reg)
				ctx.ProtectReg(d7.Reg2)
			}
			d896 = d5
			if d896.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d896)
			d897 = d896
			if d897.Loc == scm.LocImm {
				d897 = scm.JITValueDesc{Loc: scm.LocImm, Type: d897.Type, Imm: scm.NewInt(int64(uint64(d897.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d897.Reg, 32)
				ctx.EmitShrRegImm8(d897.Reg, 32)
			}
			ctx.EmitStoreToStack(d897, int32(bbs[8].PhiBase)+int32(0))
			d898 = d7
			if d898.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d898)
			d899 = d898
			if d899.Loc == scm.LocImm {
				d899 = scm.JITValueDesc{Loc: scm.LocImm, Type: d899.Type, Imm: scm.NewInt(int64(uint64(d899.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d899.Reg, 32)
				ctx.EmitShrRegImm8(d899.Reg, 32)
			}
			ctx.EmitStoreToStack(d899, int32(bbs[8].PhiBase)+int32(16))
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
			ps900 := scm.PhiState{General: ps.General}
			ps900.OverlayValues = make([]scm.JITValueDesc, 900)
			ps900.OverlayValues[1] = d1
			ps900.OverlayValues[2] = d2
			ps900.OverlayValues[3] = d3
			ps900.OverlayValues[4] = d4
			ps900.OverlayValues[5] = d5
			ps900.OverlayValues[6] = d6
			ps900.OverlayValues[7] = d7
			ps900.OverlayValues[8] = d8
			ps900.OverlayValues[9] = d9
			ps900.OverlayValues[10] = d10
			ps900.OverlayValues[11] = d11
			ps900.OverlayValues[12] = d12
			ps900.OverlayValues[13] = d13
			ps900.OverlayValues[14] = d14
			ps900.OverlayValues[15] = d15
			ps900.OverlayValues[16] = d16
			ps900.OverlayValues[17] = d17
			ps900.OverlayValues[19] = d19
			ps900.OverlayValues[20] = d20
			ps900.OverlayValues[21] = d21
			ps900.OverlayValues[22] = d22
			ps900.OverlayValues[23] = d23
			ps900.OverlayValues[24] = d24
			ps900.OverlayValues[25] = d25
			ps900.OverlayValues[27] = d27
			ps900.OverlayValues[28] = d28
			ps900.OverlayValues[29] = d29
			ps900.OverlayValues[30] = d30
			ps900.OverlayValues[31] = d31
			ps900.OverlayValues[32] = d32
			ps900.OverlayValues[33] = d33
			ps900.OverlayValues[34] = d34
			ps900.OverlayValues[35] = d35
			ps900.OverlayValues[36] = d36
			ps900.OverlayValues[37] = d37
			ps900.OverlayValues[38] = d38
			ps900.OverlayValues[39] = d39
			ps900.OverlayValues[40] = d40
			ps900.OverlayValues[41] = d41
			ps900.OverlayValues[42] = d42
			ps900.OverlayValues[43] = d43
			ps900.OverlayValues[44] = d44
			ps900.OverlayValues[45] = d45
			ps900.OverlayValues[46] = d46
			ps900.OverlayValues[47] = d47
			ps900.OverlayValues[48] = d48
			ps900.OverlayValues[49] = d49
			ps900.OverlayValues[50] = d50
			ps900.OverlayValues[51] = d51
			ps900.OverlayValues[52] = d52
			ps900.OverlayValues[53] = d53
			ps900.OverlayValues[54] = d54
			ps900.OverlayValues[55] = d55
			ps900.OverlayValues[56] = d56
			ps900.OverlayValues[57] = d57
			ps900.OverlayValues[58] = d58
			ps900.OverlayValues[59] = d59
			ps900.OverlayValues[60] = d60
			ps900.OverlayValues[61] = d61
			ps900.OverlayValues[62] = d62
			ps900.OverlayValues[63] = d63
			ps900.OverlayValues[66] = d66
			ps900.OverlayValues[67] = d67
			ps900.OverlayValues[68] = d68
			ps900.OverlayValues[136] = d136
			ps900.OverlayValues[137] = d137
			ps900.OverlayValues[138] = d138
			ps900.OverlayValues[140] = d140
			ps900.OverlayValues[141] = d141
			ps900.OverlayValues[142] = d142
			ps900.OverlayValues[143] = d143
			ps900.OverlayValues[144] = d144
			ps900.OverlayValues[145] = d145
			ps900.OverlayValues[146] = d146
			ps900.OverlayValues[147] = d147
			ps900.OverlayValues[148] = d148
			ps900.OverlayValues[149] = d149
			ps900.OverlayValues[150] = d150
			ps900.OverlayValues[151] = d151
			ps900.OverlayValues[152] = d152
			ps900.OverlayValues[153] = d153
			ps900.OverlayValues[154] = d154
			ps900.OverlayValues[155] = d155
			ps900.OverlayValues[156] = d156
			ps900.OverlayValues[157] = d157
			ps900.OverlayValues[158] = d158
			ps900.OverlayValues[159] = d159
			ps900.OverlayValues[160] = d160
			ps900.OverlayValues[161] = d161
			ps900.OverlayValues[162] = d162
			ps900.OverlayValues[163] = d163
			ps900.OverlayValues[164] = d164
			ps900.OverlayValues[165] = d165
			ps900.OverlayValues[166] = d166
			ps900.OverlayValues[167] = d167
			ps900.OverlayValues[168] = d168
			ps900.OverlayValues[169] = d169
			ps900.OverlayValues[170] = d170
			ps900.OverlayValues[171] = d171
			ps900.OverlayValues[172] = d172
			ps900.OverlayValues[173] = d173
			ps900.OverlayValues[174] = d174
			ps900.OverlayValues[175] = d175
			ps900.OverlayValues[178] = d178
			ps900.OverlayValues[286] = d286
			ps900.OverlayValues[287] = d287
			ps900.OverlayValues[288] = d288
			ps900.OverlayValues[289] = d289
			ps900.OverlayValues[290] = d290
			ps900.OverlayValues[291] = d291
			ps900.OverlayValues[292] = d292
			ps900.OverlayValues[293] = d293
			ps900.OverlayValues[295] = d295
			ps900.OverlayValues[296] = d296
			ps900.OverlayValues[297] = d297
			ps900.OverlayValues[298] = d298
			ps900.OverlayValues[299] = d299
			ps900.OverlayValues[300] = d300
			ps900.OverlayValues[301] = d301
			ps900.OverlayValues[302] = d302
			ps900.OverlayValues[303] = d303
			ps900.OverlayValues[304] = d304
			ps900.OverlayValues[306] = d306
			ps900.OverlayValues[308] = d308
			ps900.OverlayValues[309] = d309
			ps900.OverlayValues[310] = d310
			ps900.OverlayValues[311] = d311
			ps900.OverlayValues[312] = d312
			ps900.OverlayValues[315] = d315
			ps900.OverlayValues[446] = d446
			ps900.OverlayValues[447] = d447
			ps900.OverlayValues[448] = d448
			ps900.OverlayValues[449] = d449
			ps900.OverlayValues[450] = d450
			ps900.OverlayValues[451] = d451
			ps900.OverlayValues[452] = d452
			ps900.OverlayValues[454] = d454
			ps900.OverlayValues[455] = d455
			ps900.OverlayValues[456] = d456
			ps900.OverlayValues[457] = d457
			ps900.OverlayValues[459] = d459
			ps900.OverlayValues[460] = d460
			ps900.OverlayValues[461] = d461
			ps900.OverlayValues[462] = d462
			ps900.OverlayValues[463] = d463
			ps900.OverlayValues[464] = d464
			ps900.OverlayValues[465] = d465
			ps900.OverlayValues[466] = d466
			ps900.OverlayValues[467] = d467
			ps900.OverlayValues[468] = d468
			ps900.OverlayValues[469] = d469
			ps900.OverlayValues[470] = d470
			ps900.OverlayValues[471] = d471
			ps900.OverlayValues[472] = d472
			ps900.OverlayValues[473] = d473
			ps900.OverlayValues[474] = d474
			ps900.OverlayValues[475] = d475
			ps900.OverlayValues[476] = d476
			ps900.OverlayValues[477] = d477
			ps900.OverlayValues[478] = d478
			ps900.OverlayValues[479] = d479
			ps900.OverlayValues[480] = d480
			ps900.OverlayValues[481] = d481
			ps900.OverlayValues[482] = d482
			ps900.OverlayValues[483] = d483
			ps900.OverlayValues[484] = d484
			ps900.OverlayValues[485] = d485
			ps900.OverlayValues[486] = d486
			ps900.OverlayValues[487] = d487
			ps900.OverlayValues[488] = d488
			ps900.OverlayValues[489] = d489
			ps900.OverlayValues[490] = d490
			ps900.OverlayValues[491] = d491
			ps900.OverlayValues[492] = d492
			ps900.OverlayValues[493] = d493
			ps900.OverlayValues[494] = d494
			ps900.OverlayValues[676] = d676
			ps900.OverlayValues[677] = d677
			ps900.OverlayValues[678] = d678
			ps900.OverlayValues[679] = d679
			ps900.OverlayValues[680] = d680
			ps900.OverlayValues[682] = d682
			ps900.OverlayValues[683] = d683
			ps900.OverlayValues[684] = d684
			ps900.OverlayValues[685] = d685
			ps900.OverlayValues[686] = d686
			ps900.OverlayValues[687] = d687
			ps900.OverlayValues[688] = d688
			ps900.OverlayValues[689] = d689
			ps900.OverlayValues[691] = d691
			ps900.OverlayValues[693] = d693
			ps900.OverlayValues[694] = d694
			ps900.OverlayValues[695] = d695
			ps900.OverlayValues[696] = d696
			ps900.OverlayValues[699] = d699
			ps900.OverlayValues[896] = d896
			ps900.OverlayValues[897] = d897
			ps900.OverlayValues[898] = d898
			ps900.OverlayValues[899] = d899
			ps900.PhiValues = make([]scm.JITValueDesc, 2)
			d901 = d5
			ps900.PhiValues[0] = d901
			d902 = d7
			ps900.PhiValues[1] = d902
			if ps900.General && bbs[8].Rendered {
				ctx.EmitJmp(lbl9)
				return result
			}
			return bbs[8].RenderPS(ps900)
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
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
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
			if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != scm.LocNone {
				d173 = ps.OverlayValues[173]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
			}
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
				d178 = ps.OverlayValues[178]
			}
			if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != scm.LocNone {
				d286 = ps.OverlayValues[286]
			}
			if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != scm.LocNone {
				d287 = ps.OverlayValues[287]
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
			if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != scm.LocNone {
				d293 = ps.OverlayValues[293]
			}
			if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
				d295 = ps.OverlayValues[295]
			}
			if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != scm.LocNone {
				d296 = ps.OverlayValues[296]
			}
			if len(ps.OverlayValues) > 297 && ps.OverlayValues[297].Loc != scm.LocNone {
				d297 = ps.OverlayValues[297]
			}
			if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != scm.LocNone {
				d298 = ps.OverlayValues[298]
			}
			if len(ps.OverlayValues) > 299 && ps.OverlayValues[299].Loc != scm.LocNone {
				d299 = ps.OverlayValues[299]
			}
			if len(ps.OverlayValues) > 300 && ps.OverlayValues[300].Loc != scm.LocNone {
				d300 = ps.OverlayValues[300]
			}
			if len(ps.OverlayValues) > 301 && ps.OverlayValues[301].Loc != scm.LocNone {
				d301 = ps.OverlayValues[301]
			}
			if len(ps.OverlayValues) > 302 && ps.OverlayValues[302].Loc != scm.LocNone {
				d302 = ps.OverlayValues[302]
			}
			if len(ps.OverlayValues) > 303 && ps.OverlayValues[303].Loc != scm.LocNone {
				d303 = ps.OverlayValues[303]
			}
			if len(ps.OverlayValues) > 304 && ps.OverlayValues[304].Loc != scm.LocNone {
				d304 = ps.OverlayValues[304]
			}
			if len(ps.OverlayValues) > 306 && ps.OverlayValues[306].Loc != scm.LocNone {
				d306 = ps.OverlayValues[306]
			}
			if len(ps.OverlayValues) > 308 && ps.OverlayValues[308].Loc != scm.LocNone {
				d308 = ps.OverlayValues[308]
			}
			if len(ps.OverlayValues) > 309 && ps.OverlayValues[309].Loc != scm.LocNone {
				d309 = ps.OverlayValues[309]
			}
			if len(ps.OverlayValues) > 310 && ps.OverlayValues[310].Loc != scm.LocNone {
				d310 = ps.OverlayValues[310]
			}
			if len(ps.OverlayValues) > 311 && ps.OverlayValues[311].Loc != scm.LocNone {
				d311 = ps.OverlayValues[311]
			}
			if len(ps.OverlayValues) > 312 && ps.OverlayValues[312].Loc != scm.LocNone {
				d312 = ps.OverlayValues[312]
			}
			if len(ps.OverlayValues) > 315 && ps.OverlayValues[315].Loc != scm.LocNone {
				d315 = ps.OverlayValues[315]
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
			if len(ps.OverlayValues) > 459 && ps.OverlayValues[459].Loc != scm.LocNone {
				d459 = ps.OverlayValues[459]
			}
			if len(ps.OverlayValues) > 460 && ps.OverlayValues[460].Loc != scm.LocNone {
				d460 = ps.OverlayValues[460]
			}
			if len(ps.OverlayValues) > 461 && ps.OverlayValues[461].Loc != scm.LocNone {
				d461 = ps.OverlayValues[461]
			}
			if len(ps.OverlayValues) > 462 && ps.OverlayValues[462].Loc != scm.LocNone {
				d462 = ps.OverlayValues[462]
			}
			if len(ps.OverlayValues) > 463 && ps.OverlayValues[463].Loc != scm.LocNone {
				d463 = ps.OverlayValues[463]
			}
			if len(ps.OverlayValues) > 464 && ps.OverlayValues[464].Loc != scm.LocNone {
				d464 = ps.OverlayValues[464]
			}
			if len(ps.OverlayValues) > 465 && ps.OverlayValues[465].Loc != scm.LocNone {
				d465 = ps.OverlayValues[465]
			}
			if len(ps.OverlayValues) > 466 && ps.OverlayValues[466].Loc != scm.LocNone {
				d466 = ps.OverlayValues[466]
			}
			if len(ps.OverlayValues) > 467 && ps.OverlayValues[467].Loc != scm.LocNone {
				d467 = ps.OverlayValues[467]
			}
			if len(ps.OverlayValues) > 468 && ps.OverlayValues[468].Loc != scm.LocNone {
				d468 = ps.OverlayValues[468]
			}
			if len(ps.OverlayValues) > 469 && ps.OverlayValues[469].Loc != scm.LocNone {
				d469 = ps.OverlayValues[469]
			}
			if len(ps.OverlayValues) > 470 && ps.OverlayValues[470].Loc != scm.LocNone {
				d470 = ps.OverlayValues[470]
			}
			if len(ps.OverlayValues) > 471 && ps.OverlayValues[471].Loc != scm.LocNone {
				d471 = ps.OverlayValues[471]
			}
			if len(ps.OverlayValues) > 472 && ps.OverlayValues[472].Loc != scm.LocNone {
				d472 = ps.OverlayValues[472]
			}
			if len(ps.OverlayValues) > 473 && ps.OverlayValues[473].Loc != scm.LocNone {
				d473 = ps.OverlayValues[473]
			}
			if len(ps.OverlayValues) > 474 && ps.OverlayValues[474].Loc != scm.LocNone {
				d474 = ps.OverlayValues[474]
			}
			if len(ps.OverlayValues) > 475 && ps.OverlayValues[475].Loc != scm.LocNone {
				d475 = ps.OverlayValues[475]
			}
			if len(ps.OverlayValues) > 476 && ps.OverlayValues[476].Loc != scm.LocNone {
				d476 = ps.OverlayValues[476]
			}
			if len(ps.OverlayValues) > 477 && ps.OverlayValues[477].Loc != scm.LocNone {
				d477 = ps.OverlayValues[477]
			}
			if len(ps.OverlayValues) > 478 && ps.OverlayValues[478].Loc != scm.LocNone {
				d478 = ps.OverlayValues[478]
			}
			if len(ps.OverlayValues) > 479 && ps.OverlayValues[479].Loc != scm.LocNone {
				d479 = ps.OverlayValues[479]
			}
			if len(ps.OverlayValues) > 480 && ps.OverlayValues[480].Loc != scm.LocNone {
				d480 = ps.OverlayValues[480]
			}
			if len(ps.OverlayValues) > 481 && ps.OverlayValues[481].Loc != scm.LocNone {
				d481 = ps.OverlayValues[481]
			}
			if len(ps.OverlayValues) > 482 && ps.OverlayValues[482].Loc != scm.LocNone {
				d482 = ps.OverlayValues[482]
			}
			if len(ps.OverlayValues) > 483 && ps.OverlayValues[483].Loc != scm.LocNone {
				d483 = ps.OverlayValues[483]
			}
			if len(ps.OverlayValues) > 484 && ps.OverlayValues[484].Loc != scm.LocNone {
				d484 = ps.OverlayValues[484]
			}
			if len(ps.OverlayValues) > 485 && ps.OverlayValues[485].Loc != scm.LocNone {
				d485 = ps.OverlayValues[485]
			}
			if len(ps.OverlayValues) > 486 && ps.OverlayValues[486].Loc != scm.LocNone {
				d486 = ps.OverlayValues[486]
			}
			if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != scm.LocNone {
				d487 = ps.OverlayValues[487]
			}
			if len(ps.OverlayValues) > 488 && ps.OverlayValues[488].Loc != scm.LocNone {
				d488 = ps.OverlayValues[488]
			}
			if len(ps.OverlayValues) > 489 && ps.OverlayValues[489].Loc != scm.LocNone {
				d489 = ps.OverlayValues[489]
			}
			if len(ps.OverlayValues) > 490 && ps.OverlayValues[490].Loc != scm.LocNone {
				d490 = ps.OverlayValues[490]
			}
			if len(ps.OverlayValues) > 491 && ps.OverlayValues[491].Loc != scm.LocNone {
				d491 = ps.OverlayValues[491]
			}
			if len(ps.OverlayValues) > 492 && ps.OverlayValues[492].Loc != scm.LocNone {
				d492 = ps.OverlayValues[492]
			}
			if len(ps.OverlayValues) > 493 && ps.OverlayValues[493].Loc != scm.LocNone {
				d493 = ps.OverlayValues[493]
			}
			if len(ps.OverlayValues) > 494 && ps.OverlayValues[494].Loc != scm.LocNone {
				d494 = ps.OverlayValues[494]
			}
			if len(ps.OverlayValues) > 676 && ps.OverlayValues[676].Loc != scm.LocNone {
				d676 = ps.OverlayValues[676]
			}
			if len(ps.OverlayValues) > 677 && ps.OverlayValues[677].Loc != scm.LocNone {
				d677 = ps.OverlayValues[677]
			}
			if len(ps.OverlayValues) > 678 && ps.OverlayValues[678].Loc != scm.LocNone {
				d678 = ps.OverlayValues[678]
			}
			if len(ps.OverlayValues) > 679 && ps.OverlayValues[679].Loc != scm.LocNone {
				d679 = ps.OverlayValues[679]
			}
			if len(ps.OverlayValues) > 680 && ps.OverlayValues[680].Loc != scm.LocNone {
				d680 = ps.OverlayValues[680]
			}
			if len(ps.OverlayValues) > 682 && ps.OverlayValues[682].Loc != scm.LocNone {
				d682 = ps.OverlayValues[682]
			}
			if len(ps.OverlayValues) > 683 && ps.OverlayValues[683].Loc != scm.LocNone {
				d683 = ps.OverlayValues[683]
			}
			if len(ps.OverlayValues) > 684 && ps.OverlayValues[684].Loc != scm.LocNone {
				d684 = ps.OverlayValues[684]
			}
			if len(ps.OverlayValues) > 685 && ps.OverlayValues[685].Loc != scm.LocNone {
				d685 = ps.OverlayValues[685]
			}
			if len(ps.OverlayValues) > 686 && ps.OverlayValues[686].Loc != scm.LocNone {
				d686 = ps.OverlayValues[686]
			}
			if len(ps.OverlayValues) > 687 && ps.OverlayValues[687].Loc != scm.LocNone {
				d687 = ps.OverlayValues[687]
			}
			if len(ps.OverlayValues) > 688 && ps.OverlayValues[688].Loc != scm.LocNone {
				d688 = ps.OverlayValues[688]
			}
			if len(ps.OverlayValues) > 689 && ps.OverlayValues[689].Loc != scm.LocNone {
				d689 = ps.OverlayValues[689]
			}
			if len(ps.OverlayValues) > 691 && ps.OverlayValues[691].Loc != scm.LocNone {
				d691 = ps.OverlayValues[691]
			}
			if len(ps.OverlayValues) > 693 && ps.OverlayValues[693].Loc != scm.LocNone {
				d693 = ps.OverlayValues[693]
			}
			if len(ps.OverlayValues) > 694 && ps.OverlayValues[694].Loc != scm.LocNone {
				d694 = ps.OverlayValues[694]
			}
			if len(ps.OverlayValues) > 695 && ps.OverlayValues[695].Loc != scm.LocNone {
				d695 = ps.OverlayValues[695]
			}
			if len(ps.OverlayValues) > 696 && ps.OverlayValues[696].Loc != scm.LocNone {
				d696 = ps.OverlayValues[696]
			}
			if len(ps.OverlayValues) > 699 && ps.OverlayValues[699].Loc != scm.LocNone {
				d699 = ps.OverlayValues[699]
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
			if len(ps.OverlayValues) > 901 && ps.OverlayValues[901].Loc != scm.LocNone {
				d901 = ps.OverlayValues[901]
			}
			if len(ps.OverlayValues) > 902 && ps.OverlayValues[902].Loc != scm.LocNone {
				d902 = ps.OverlayValues[902]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d9)
			ctx.EnsureDesc(&d8)
			ctx.ProtectReg(d8.Reg)
			ctx.EnsureDesc(&d9)
			ctx.UnprotectReg(d8.Reg)
			var d903 scm.JITValueDesc
			if d8.Loc == scm.LocImm && d9.Loc == scm.LocImm {
				d903 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() + d9.Imm.Int())}
			} else if d9.Loc == scm.LocImm && d9.Imm.Int() == 0 {
				r153 := ctx.AllocRegExcept(d8.Reg)
				ctx.EmitMovRegReg(r153, d8.Reg)
				d903 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r153}
				ctx.BindReg(r153, &d903)
			} else if d8.Loc == scm.LocImm && d8.Imm.Int() == 0 {
				d903 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d9.Reg}
				ctx.BindReg(d9.Reg, &d903)
			} else if d8.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d9.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d8.Imm.Int()))
				ctx.EmitAddInt64(scratch, d9.Reg)
				d903 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d903)
			} else if d9.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d8.Reg)
				ctx.EmitMovRegReg(scratch, d8.Reg)
				if d9.Imm.Int() >= -2147483648 && d9.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d9.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d9.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d903 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d903)
			} else {
				r154 := ctx.AllocRegExcept(d8.Reg, d9.Reg)
				ctx.EmitMovRegReg(r154, d8.Reg)
				ctx.EmitAddInt64(r154, d9.Reg)
				d903 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r154}
				ctx.BindReg(r154, &d903)
			}
			if d903.Loc == scm.LocImm {
				d903 = scm.JITValueDesc{Loc: scm.LocImm, Type: d903.Type, Imm: scm.NewInt(int64(uint64(d903.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d903.Reg, 32)
				ctx.EmitShrRegImm8(d903.Reg, 32)
			}
			if d903.Loc == scm.LocReg && d8.Loc == scm.LocReg && d903.Reg == d8.Reg {
				ctx.TransferReg(d8.Reg)
				d8.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d903)
			var d904 scm.JITValueDesc
			if d903.Loc == scm.LocImm {
				d904 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d903.Imm.Int() / 2)}
			} else {
				r155 := ctx.AllocRegExcept(d903.Reg)
				ctx.EmitMovRegReg(r155, d903.Reg)
				ctx.EmitShrRegImm8(r155, 1)
				d904 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r155}
				ctx.BindReg(r155, &d904)
			}
			if d904.Loc == scm.LocImm {
				d904 = scm.JITValueDesc{Loc: scm.LocImm, Type: d904.Type, Imm: scm.NewInt(int64(uint64(d904.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d904.Reg, 32)
				ctx.EmitShrRegImm8(d904.Reg, 32)
			}
			if d904.Loc == scm.LocReg && d903.Loc == scm.LocReg && d904.Reg == d903.Reg {
				ctx.TransferReg(d903.Reg)
				d903.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d903)
			ctx.EnsureDesc(&d8)
			if d8.Loc == scm.LocReg {
				ctx.ProtectReg(d8.Reg)
			} else if d8.Loc == scm.LocRegPair {
				ctx.ProtectReg(d8.Reg)
				ctx.ProtectReg(d8.Reg2)
			}
			ctx.EnsureDesc(&d9)
			if d9.Loc == scm.LocReg {
				ctx.ProtectReg(d9.Reg)
			} else if d9.Loc == scm.LocRegPair {
				ctx.ProtectReg(d9.Reg)
				ctx.ProtectReg(d9.Reg2)
			}
			ctx.EnsureDesc(&d904)
			if d904.Loc == scm.LocReg {
				ctx.ProtectReg(d904.Reg)
			} else if d904.Loc == scm.LocRegPair {
				ctx.ProtectReg(d904.Reg)
				ctx.ProtectReg(d904.Reg2)
			}
			d905 = d904
			if d905.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d905)
			d906 = d905
			if d906.Loc == scm.LocImm {
				d906 = scm.JITValueDesc{Loc: scm.LocImm, Type: d906.Type, Imm: scm.NewInt(int64(uint64(d906.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d906.Reg, 32)
				ctx.EmitShrRegImm8(d906.Reg, 32)
			}
			ctx.EmitStoreToStack(d906, int32(bbs[1].PhiBase)+int32(0))
			d907 = d8
			if d907.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d907)
			d908 = d907
			if d908.Loc == scm.LocImm {
				d908 = scm.JITValueDesc{Loc: scm.LocImm, Type: d908.Type, Imm: scm.NewInt(int64(uint64(d908.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d908.Reg, 32)
				ctx.EmitShrRegImm8(d908.Reg, 32)
			}
			ctx.EmitStoreToStack(d908, int32(bbs[1].PhiBase)+int32(16))
			d909 = d9
			if d909.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d909)
			d910 = d909
			if d910.Loc == scm.LocImm {
				d910 = scm.JITValueDesc{Loc: scm.LocImm, Type: d910.Type, Imm: scm.NewInt(int64(uint64(d910.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d910.Reg, 32)
				ctx.EmitShrRegImm8(d910.Reg, 32)
			}
			ctx.EmitStoreToStack(d910, int32(bbs[1].PhiBase)+int32(32))
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
			if d904.Loc == scm.LocReg {
				ctx.UnprotectReg(d904.Reg)
			} else if d904.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d904.Reg)
				ctx.UnprotectReg(d904.Reg2)
			}
			ps911 := scm.PhiState{General: ps.General}
			ps911.OverlayValues = make([]scm.JITValueDesc, 911)
			ps911.OverlayValues[1] = d1
			ps911.OverlayValues[2] = d2
			ps911.OverlayValues[3] = d3
			ps911.OverlayValues[4] = d4
			ps911.OverlayValues[5] = d5
			ps911.OverlayValues[6] = d6
			ps911.OverlayValues[7] = d7
			ps911.OverlayValues[8] = d8
			ps911.OverlayValues[9] = d9
			ps911.OverlayValues[10] = d10
			ps911.OverlayValues[11] = d11
			ps911.OverlayValues[12] = d12
			ps911.OverlayValues[13] = d13
			ps911.OverlayValues[14] = d14
			ps911.OverlayValues[15] = d15
			ps911.OverlayValues[16] = d16
			ps911.OverlayValues[17] = d17
			ps911.OverlayValues[19] = d19
			ps911.OverlayValues[20] = d20
			ps911.OverlayValues[21] = d21
			ps911.OverlayValues[22] = d22
			ps911.OverlayValues[23] = d23
			ps911.OverlayValues[24] = d24
			ps911.OverlayValues[25] = d25
			ps911.OverlayValues[27] = d27
			ps911.OverlayValues[28] = d28
			ps911.OverlayValues[29] = d29
			ps911.OverlayValues[30] = d30
			ps911.OverlayValues[31] = d31
			ps911.OverlayValues[32] = d32
			ps911.OverlayValues[33] = d33
			ps911.OverlayValues[34] = d34
			ps911.OverlayValues[35] = d35
			ps911.OverlayValues[36] = d36
			ps911.OverlayValues[37] = d37
			ps911.OverlayValues[38] = d38
			ps911.OverlayValues[39] = d39
			ps911.OverlayValues[40] = d40
			ps911.OverlayValues[41] = d41
			ps911.OverlayValues[42] = d42
			ps911.OverlayValues[43] = d43
			ps911.OverlayValues[44] = d44
			ps911.OverlayValues[45] = d45
			ps911.OverlayValues[46] = d46
			ps911.OverlayValues[47] = d47
			ps911.OverlayValues[48] = d48
			ps911.OverlayValues[49] = d49
			ps911.OverlayValues[50] = d50
			ps911.OverlayValues[51] = d51
			ps911.OverlayValues[52] = d52
			ps911.OverlayValues[53] = d53
			ps911.OverlayValues[54] = d54
			ps911.OverlayValues[55] = d55
			ps911.OverlayValues[56] = d56
			ps911.OverlayValues[57] = d57
			ps911.OverlayValues[58] = d58
			ps911.OverlayValues[59] = d59
			ps911.OverlayValues[60] = d60
			ps911.OverlayValues[61] = d61
			ps911.OverlayValues[62] = d62
			ps911.OverlayValues[63] = d63
			ps911.OverlayValues[66] = d66
			ps911.OverlayValues[67] = d67
			ps911.OverlayValues[68] = d68
			ps911.OverlayValues[136] = d136
			ps911.OverlayValues[137] = d137
			ps911.OverlayValues[138] = d138
			ps911.OverlayValues[140] = d140
			ps911.OverlayValues[141] = d141
			ps911.OverlayValues[142] = d142
			ps911.OverlayValues[143] = d143
			ps911.OverlayValues[144] = d144
			ps911.OverlayValues[145] = d145
			ps911.OverlayValues[146] = d146
			ps911.OverlayValues[147] = d147
			ps911.OverlayValues[148] = d148
			ps911.OverlayValues[149] = d149
			ps911.OverlayValues[150] = d150
			ps911.OverlayValues[151] = d151
			ps911.OverlayValues[152] = d152
			ps911.OverlayValues[153] = d153
			ps911.OverlayValues[154] = d154
			ps911.OverlayValues[155] = d155
			ps911.OverlayValues[156] = d156
			ps911.OverlayValues[157] = d157
			ps911.OverlayValues[158] = d158
			ps911.OverlayValues[159] = d159
			ps911.OverlayValues[160] = d160
			ps911.OverlayValues[161] = d161
			ps911.OverlayValues[162] = d162
			ps911.OverlayValues[163] = d163
			ps911.OverlayValues[164] = d164
			ps911.OverlayValues[165] = d165
			ps911.OverlayValues[166] = d166
			ps911.OverlayValues[167] = d167
			ps911.OverlayValues[168] = d168
			ps911.OverlayValues[169] = d169
			ps911.OverlayValues[170] = d170
			ps911.OverlayValues[171] = d171
			ps911.OverlayValues[172] = d172
			ps911.OverlayValues[173] = d173
			ps911.OverlayValues[174] = d174
			ps911.OverlayValues[175] = d175
			ps911.OverlayValues[178] = d178
			ps911.OverlayValues[286] = d286
			ps911.OverlayValues[287] = d287
			ps911.OverlayValues[288] = d288
			ps911.OverlayValues[289] = d289
			ps911.OverlayValues[290] = d290
			ps911.OverlayValues[291] = d291
			ps911.OverlayValues[292] = d292
			ps911.OverlayValues[293] = d293
			ps911.OverlayValues[295] = d295
			ps911.OverlayValues[296] = d296
			ps911.OverlayValues[297] = d297
			ps911.OverlayValues[298] = d298
			ps911.OverlayValues[299] = d299
			ps911.OverlayValues[300] = d300
			ps911.OverlayValues[301] = d301
			ps911.OverlayValues[302] = d302
			ps911.OverlayValues[303] = d303
			ps911.OverlayValues[304] = d304
			ps911.OverlayValues[306] = d306
			ps911.OverlayValues[308] = d308
			ps911.OverlayValues[309] = d309
			ps911.OverlayValues[310] = d310
			ps911.OverlayValues[311] = d311
			ps911.OverlayValues[312] = d312
			ps911.OverlayValues[315] = d315
			ps911.OverlayValues[446] = d446
			ps911.OverlayValues[447] = d447
			ps911.OverlayValues[448] = d448
			ps911.OverlayValues[449] = d449
			ps911.OverlayValues[450] = d450
			ps911.OverlayValues[451] = d451
			ps911.OverlayValues[452] = d452
			ps911.OverlayValues[454] = d454
			ps911.OverlayValues[455] = d455
			ps911.OverlayValues[456] = d456
			ps911.OverlayValues[457] = d457
			ps911.OverlayValues[459] = d459
			ps911.OverlayValues[460] = d460
			ps911.OverlayValues[461] = d461
			ps911.OverlayValues[462] = d462
			ps911.OverlayValues[463] = d463
			ps911.OverlayValues[464] = d464
			ps911.OverlayValues[465] = d465
			ps911.OverlayValues[466] = d466
			ps911.OverlayValues[467] = d467
			ps911.OverlayValues[468] = d468
			ps911.OverlayValues[469] = d469
			ps911.OverlayValues[470] = d470
			ps911.OverlayValues[471] = d471
			ps911.OverlayValues[472] = d472
			ps911.OverlayValues[473] = d473
			ps911.OverlayValues[474] = d474
			ps911.OverlayValues[475] = d475
			ps911.OverlayValues[476] = d476
			ps911.OverlayValues[477] = d477
			ps911.OverlayValues[478] = d478
			ps911.OverlayValues[479] = d479
			ps911.OverlayValues[480] = d480
			ps911.OverlayValues[481] = d481
			ps911.OverlayValues[482] = d482
			ps911.OverlayValues[483] = d483
			ps911.OverlayValues[484] = d484
			ps911.OverlayValues[485] = d485
			ps911.OverlayValues[486] = d486
			ps911.OverlayValues[487] = d487
			ps911.OverlayValues[488] = d488
			ps911.OverlayValues[489] = d489
			ps911.OverlayValues[490] = d490
			ps911.OverlayValues[491] = d491
			ps911.OverlayValues[492] = d492
			ps911.OverlayValues[493] = d493
			ps911.OverlayValues[494] = d494
			ps911.OverlayValues[676] = d676
			ps911.OverlayValues[677] = d677
			ps911.OverlayValues[678] = d678
			ps911.OverlayValues[679] = d679
			ps911.OverlayValues[680] = d680
			ps911.OverlayValues[682] = d682
			ps911.OverlayValues[683] = d683
			ps911.OverlayValues[684] = d684
			ps911.OverlayValues[685] = d685
			ps911.OverlayValues[686] = d686
			ps911.OverlayValues[687] = d687
			ps911.OverlayValues[688] = d688
			ps911.OverlayValues[689] = d689
			ps911.OverlayValues[691] = d691
			ps911.OverlayValues[693] = d693
			ps911.OverlayValues[694] = d694
			ps911.OverlayValues[695] = d695
			ps911.OverlayValues[696] = d696
			ps911.OverlayValues[699] = d699
			ps911.OverlayValues[896] = d896
			ps911.OverlayValues[897] = d897
			ps911.OverlayValues[898] = d898
			ps911.OverlayValues[899] = d899
			ps911.OverlayValues[901] = d901
			ps911.OverlayValues[902] = d902
			ps911.OverlayValues[903] = d903
			ps911.OverlayValues[904] = d904
			ps911.OverlayValues[905] = d905
			ps911.OverlayValues[906] = d906
			ps911.OverlayValues[907] = d907
			ps911.OverlayValues[908] = d908
			ps911.OverlayValues[909] = d909
			ps911.OverlayValues[910] = d910
			ps911.PhiValues = make([]scm.JITValueDesc, 3)
			d912 = d904
			ps911.PhiValues[0] = d912
			d913 = d8
			ps911.PhiValues[1] = d913
			d914 = d9
			ps911.PhiValues[2] = d914
			if ps911.General && bbs[1].Rendered {
				ctx.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps911)
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
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
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
			if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != scm.LocNone {
				d173 = ps.OverlayValues[173]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
			}
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
				d178 = ps.OverlayValues[178]
			}
			if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != scm.LocNone {
				d286 = ps.OverlayValues[286]
			}
			if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != scm.LocNone {
				d287 = ps.OverlayValues[287]
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
			if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != scm.LocNone {
				d293 = ps.OverlayValues[293]
			}
			if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
				d295 = ps.OverlayValues[295]
			}
			if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != scm.LocNone {
				d296 = ps.OverlayValues[296]
			}
			if len(ps.OverlayValues) > 297 && ps.OverlayValues[297].Loc != scm.LocNone {
				d297 = ps.OverlayValues[297]
			}
			if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != scm.LocNone {
				d298 = ps.OverlayValues[298]
			}
			if len(ps.OverlayValues) > 299 && ps.OverlayValues[299].Loc != scm.LocNone {
				d299 = ps.OverlayValues[299]
			}
			if len(ps.OverlayValues) > 300 && ps.OverlayValues[300].Loc != scm.LocNone {
				d300 = ps.OverlayValues[300]
			}
			if len(ps.OverlayValues) > 301 && ps.OverlayValues[301].Loc != scm.LocNone {
				d301 = ps.OverlayValues[301]
			}
			if len(ps.OverlayValues) > 302 && ps.OverlayValues[302].Loc != scm.LocNone {
				d302 = ps.OverlayValues[302]
			}
			if len(ps.OverlayValues) > 303 && ps.OverlayValues[303].Loc != scm.LocNone {
				d303 = ps.OverlayValues[303]
			}
			if len(ps.OverlayValues) > 304 && ps.OverlayValues[304].Loc != scm.LocNone {
				d304 = ps.OverlayValues[304]
			}
			if len(ps.OverlayValues) > 306 && ps.OverlayValues[306].Loc != scm.LocNone {
				d306 = ps.OverlayValues[306]
			}
			if len(ps.OverlayValues) > 308 && ps.OverlayValues[308].Loc != scm.LocNone {
				d308 = ps.OverlayValues[308]
			}
			if len(ps.OverlayValues) > 309 && ps.OverlayValues[309].Loc != scm.LocNone {
				d309 = ps.OverlayValues[309]
			}
			if len(ps.OverlayValues) > 310 && ps.OverlayValues[310].Loc != scm.LocNone {
				d310 = ps.OverlayValues[310]
			}
			if len(ps.OverlayValues) > 311 && ps.OverlayValues[311].Loc != scm.LocNone {
				d311 = ps.OverlayValues[311]
			}
			if len(ps.OverlayValues) > 312 && ps.OverlayValues[312].Loc != scm.LocNone {
				d312 = ps.OverlayValues[312]
			}
			if len(ps.OverlayValues) > 315 && ps.OverlayValues[315].Loc != scm.LocNone {
				d315 = ps.OverlayValues[315]
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
			if len(ps.OverlayValues) > 459 && ps.OverlayValues[459].Loc != scm.LocNone {
				d459 = ps.OverlayValues[459]
			}
			if len(ps.OverlayValues) > 460 && ps.OverlayValues[460].Loc != scm.LocNone {
				d460 = ps.OverlayValues[460]
			}
			if len(ps.OverlayValues) > 461 && ps.OverlayValues[461].Loc != scm.LocNone {
				d461 = ps.OverlayValues[461]
			}
			if len(ps.OverlayValues) > 462 && ps.OverlayValues[462].Loc != scm.LocNone {
				d462 = ps.OverlayValues[462]
			}
			if len(ps.OverlayValues) > 463 && ps.OverlayValues[463].Loc != scm.LocNone {
				d463 = ps.OverlayValues[463]
			}
			if len(ps.OverlayValues) > 464 && ps.OverlayValues[464].Loc != scm.LocNone {
				d464 = ps.OverlayValues[464]
			}
			if len(ps.OverlayValues) > 465 && ps.OverlayValues[465].Loc != scm.LocNone {
				d465 = ps.OverlayValues[465]
			}
			if len(ps.OverlayValues) > 466 && ps.OverlayValues[466].Loc != scm.LocNone {
				d466 = ps.OverlayValues[466]
			}
			if len(ps.OverlayValues) > 467 && ps.OverlayValues[467].Loc != scm.LocNone {
				d467 = ps.OverlayValues[467]
			}
			if len(ps.OverlayValues) > 468 && ps.OverlayValues[468].Loc != scm.LocNone {
				d468 = ps.OverlayValues[468]
			}
			if len(ps.OverlayValues) > 469 && ps.OverlayValues[469].Loc != scm.LocNone {
				d469 = ps.OverlayValues[469]
			}
			if len(ps.OverlayValues) > 470 && ps.OverlayValues[470].Loc != scm.LocNone {
				d470 = ps.OverlayValues[470]
			}
			if len(ps.OverlayValues) > 471 && ps.OverlayValues[471].Loc != scm.LocNone {
				d471 = ps.OverlayValues[471]
			}
			if len(ps.OverlayValues) > 472 && ps.OverlayValues[472].Loc != scm.LocNone {
				d472 = ps.OverlayValues[472]
			}
			if len(ps.OverlayValues) > 473 && ps.OverlayValues[473].Loc != scm.LocNone {
				d473 = ps.OverlayValues[473]
			}
			if len(ps.OverlayValues) > 474 && ps.OverlayValues[474].Loc != scm.LocNone {
				d474 = ps.OverlayValues[474]
			}
			if len(ps.OverlayValues) > 475 && ps.OverlayValues[475].Loc != scm.LocNone {
				d475 = ps.OverlayValues[475]
			}
			if len(ps.OverlayValues) > 476 && ps.OverlayValues[476].Loc != scm.LocNone {
				d476 = ps.OverlayValues[476]
			}
			if len(ps.OverlayValues) > 477 && ps.OverlayValues[477].Loc != scm.LocNone {
				d477 = ps.OverlayValues[477]
			}
			if len(ps.OverlayValues) > 478 && ps.OverlayValues[478].Loc != scm.LocNone {
				d478 = ps.OverlayValues[478]
			}
			if len(ps.OverlayValues) > 479 && ps.OverlayValues[479].Loc != scm.LocNone {
				d479 = ps.OverlayValues[479]
			}
			if len(ps.OverlayValues) > 480 && ps.OverlayValues[480].Loc != scm.LocNone {
				d480 = ps.OverlayValues[480]
			}
			if len(ps.OverlayValues) > 481 && ps.OverlayValues[481].Loc != scm.LocNone {
				d481 = ps.OverlayValues[481]
			}
			if len(ps.OverlayValues) > 482 && ps.OverlayValues[482].Loc != scm.LocNone {
				d482 = ps.OverlayValues[482]
			}
			if len(ps.OverlayValues) > 483 && ps.OverlayValues[483].Loc != scm.LocNone {
				d483 = ps.OverlayValues[483]
			}
			if len(ps.OverlayValues) > 484 && ps.OverlayValues[484].Loc != scm.LocNone {
				d484 = ps.OverlayValues[484]
			}
			if len(ps.OverlayValues) > 485 && ps.OverlayValues[485].Loc != scm.LocNone {
				d485 = ps.OverlayValues[485]
			}
			if len(ps.OverlayValues) > 486 && ps.OverlayValues[486].Loc != scm.LocNone {
				d486 = ps.OverlayValues[486]
			}
			if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != scm.LocNone {
				d487 = ps.OverlayValues[487]
			}
			if len(ps.OverlayValues) > 488 && ps.OverlayValues[488].Loc != scm.LocNone {
				d488 = ps.OverlayValues[488]
			}
			if len(ps.OverlayValues) > 489 && ps.OverlayValues[489].Loc != scm.LocNone {
				d489 = ps.OverlayValues[489]
			}
			if len(ps.OverlayValues) > 490 && ps.OverlayValues[490].Loc != scm.LocNone {
				d490 = ps.OverlayValues[490]
			}
			if len(ps.OverlayValues) > 491 && ps.OverlayValues[491].Loc != scm.LocNone {
				d491 = ps.OverlayValues[491]
			}
			if len(ps.OverlayValues) > 492 && ps.OverlayValues[492].Loc != scm.LocNone {
				d492 = ps.OverlayValues[492]
			}
			if len(ps.OverlayValues) > 493 && ps.OverlayValues[493].Loc != scm.LocNone {
				d493 = ps.OverlayValues[493]
			}
			if len(ps.OverlayValues) > 494 && ps.OverlayValues[494].Loc != scm.LocNone {
				d494 = ps.OverlayValues[494]
			}
			if len(ps.OverlayValues) > 676 && ps.OverlayValues[676].Loc != scm.LocNone {
				d676 = ps.OverlayValues[676]
			}
			if len(ps.OverlayValues) > 677 && ps.OverlayValues[677].Loc != scm.LocNone {
				d677 = ps.OverlayValues[677]
			}
			if len(ps.OverlayValues) > 678 && ps.OverlayValues[678].Loc != scm.LocNone {
				d678 = ps.OverlayValues[678]
			}
			if len(ps.OverlayValues) > 679 && ps.OverlayValues[679].Loc != scm.LocNone {
				d679 = ps.OverlayValues[679]
			}
			if len(ps.OverlayValues) > 680 && ps.OverlayValues[680].Loc != scm.LocNone {
				d680 = ps.OverlayValues[680]
			}
			if len(ps.OverlayValues) > 682 && ps.OverlayValues[682].Loc != scm.LocNone {
				d682 = ps.OverlayValues[682]
			}
			if len(ps.OverlayValues) > 683 && ps.OverlayValues[683].Loc != scm.LocNone {
				d683 = ps.OverlayValues[683]
			}
			if len(ps.OverlayValues) > 684 && ps.OverlayValues[684].Loc != scm.LocNone {
				d684 = ps.OverlayValues[684]
			}
			if len(ps.OverlayValues) > 685 && ps.OverlayValues[685].Loc != scm.LocNone {
				d685 = ps.OverlayValues[685]
			}
			if len(ps.OverlayValues) > 686 && ps.OverlayValues[686].Loc != scm.LocNone {
				d686 = ps.OverlayValues[686]
			}
			if len(ps.OverlayValues) > 687 && ps.OverlayValues[687].Loc != scm.LocNone {
				d687 = ps.OverlayValues[687]
			}
			if len(ps.OverlayValues) > 688 && ps.OverlayValues[688].Loc != scm.LocNone {
				d688 = ps.OverlayValues[688]
			}
			if len(ps.OverlayValues) > 689 && ps.OverlayValues[689].Loc != scm.LocNone {
				d689 = ps.OverlayValues[689]
			}
			if len(ps.OverlayValues) > 691 && ps.OverlayValues[691].Loc != scm.LocNone {
				d691 = ps.OverlayValues[691]
			}
			if len(ps.OverlayValues) > 693 && ps.OverlayValues[693].Loc != scm.LocNone {
				d693 = ps.OverlayValues[693]
			}
			if len(ps.OverlayValues) > 694 && ps.OverlayValues[694].Loc != scm.LocNone {
				d694 = ps.OverlayValues[694]
			}
			if len(ps.OverlayValues) > 695 && ps.OverlayValues[695].Loc != scm.LocNone {
				d695 = ps.OverlayValues[695]
			}
			if len(ps.OverlayValues) > 696 && ps.OverlayValues[696].Loc != scm.LocNone {
				d696 = ps.OverlayValues[696]
			}
			if len(ps.OverlayValues) > 699 && ps.OverlayValues[699].Loc != scm.LocNone {
				d699 = ps.OverlayValues[699]
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
			if len(ps.OverlayValues) > 912 && ps.OverlayValues[912].Loc != scm.LocNone {
				d912 = ps.OverlayValues[912]
			}
			if len(ps.OverlayValues) > 913 && ps.OverlayValues[913].Loc != scm.LocNone {
				d913 = ps.OverlayValues[913]
			}
			if len(ps.OverlayValues) > 914 && ps.OverlayValues[914].Loc != scm.LocNone {
				d914 = ps.OverlayValues[914]
			}
			ctx.ReclaimUntrackedRegs()
			d915 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d915)
			ctx.BindReg(r1, &d915)
			ctx.EmitMakeNil(d915)
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
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
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
			if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != scm.LocNone {
				d173 = ps.OverlayValues[173]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
			}
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
				d178 = ps.OverlayValues[178]
			}
			if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != scm.LocNone {
				d286 = ps.OverlayValues[286]
			}
			if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != scm.LocNone {
				d287 = ps.OverlayValues[287]
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
			if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != scm.LocNone {
				d293 = ps.OverlayValues[293]
			}
			if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
				d295 = ps.OverlayValues[295]
			}
			if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != scm.LocNone {
				d296 = ps.OverlayValues[296]
			}
			if len(ps.OverlayValues) > 297 && ps.OverlayValues[297].Loc != scm.LocNone {
				d297 = ps.OverlayValues[297]
			}
			if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != scm.LocNone {
				d298 = ps.OverlayValues[298]
			}
			if len(ps.OverlayValues) > 299 && ps.OverlayValues[299].Loc != scm.LocNone {
				d299 = ps.OverlayValues[299]
			}
			if len(ps.OverlayValues) > 300 && ps.OverlayValues[300].Loc != scm.LocNone {
				d300 = ps.OverlayValues[300]
			}
			if len(ps.OverlayValues) > 301 && ps.OverlayValues[301].Loc != scm.LocNone {
				d301 = ps.OverlayValues[301]
			}
			if len(ps.OverlayValues) > 302 && ps.OverlayValues[302].Loc != scm.LocNone {
				d302 = ps.OverlayValues[302]
			}
			if len(ps.OverlayValues) > 303 && ps.OverlayValues[303].Loc != scm.LocNone {
				d303 = ps.OverlayValues[303]
			}
			if len(ps.OverlayValues) > 304 && ps.OverlayValues[304].Loc != scm.LocNone {
				d304 = ps.OverlayValues[304]
			}
			if len(ps.OverlayValues) > 306 && ps.OverlayValues[306].Loc != scm.LocNone {
				d306 = ps.OverlayValues[306]
			}
			if len(ps.OverlayValues) > 308 && ps.OverlayValues[308].Loc != scm.LocNone {
				d308 = ps.OverlayValues[308]
			}
			if len(ps.OverlayValues) > 309 && ps.OverlayValues[309].Loc != scm.LocNone {
				d309 = ps.OverlayValues[309]
			}
			if len(ps.OverlayValues) > 310 && ps.OverlayValues[310].Loc != scm.LocNone {
				d310 = ps.OverlayValues[310]
			}
			if len(ps.OverlayValues) > 311 && ps.OverlayValues[311].Loc != scm.LocNone {
				d311 = ps.OverlayValues[311]
			}
			if len(ps.OverlayValues) > 312 && ps.OverlayValues[312].Loc != scm.LocNone {
				d312 = ps.OverlayValues[312]
			}
			if len(ps.OverlayValues) > 315 && ps.OverlayValues[315].Loc != scm.LocNone {
				d315 = ps.OverlayValues[315]
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
			if len(ps.OverlayValues) > 459 && ps.OverlayValues[459].Loc != scm.LocNone {
				d459 = ps.OverlayValues[459]
			}
			if len(ps.OverlayValues) > 460 && ps.OverlayValues[460].Loc != scm.LocNone {
				d460 = ps.OverlayValues[460]
			}
			if len(ps.OverlayValues) > 461 && ps.OverlayValues[461].Loc != scm.LocNone {
				d461 = ps.OverlayValues[461]
			}
			if len(ps.OverlayValues) > 462 && ps.OverlayValues[462].Loc != scm.LocNone {
				d462 = ps.OverlayValues[462]
			}
			if len(ps.OverlayValues) > 463 && ps.OverlayValues[463].Loc != scm.LocNone {
				d463 = ps.OverlayValues[463]
			}
			if len(ps.OverlayValues) > 464 && ps.OverlayValues[464].Loc != scm.LocNone {
				d464 = ps.OverlayValues[464]
			}
			if len(ps.OverlayValues) > 465 && ps.OverlayValues[465].Loc != scm.LocNone {
				d465 = ps.OverlayValues[465]
			}
			if len(ps.OverlayValues) > 466 && ps.OverlayValues[466].Loc != scm.LocNone {
				d466 = ps.OverlayValues[466]
			}
			if len(ps.OverlayValues) > 467 && ps.OverlayValues[467].Loc != scm.LocNone {
				d467 = ps.OverlayValues[467]
			}
			if len(ps.OverlayValues) > 468 && ps.OverlayValues[468].Loc != scm.LocNone {
				d468 = ps.OverlayValues[468]
			}
			if len(ps.OverlayValues) > 469 && ps.OverlayValues[469].Loc != scm.LocNone {
				d469 = ps.OverlayValues[469]
			}
			if len(ps.OverlayValues) > 470 && ps.OverlayValues[470].Loc != scm.LocNone {
				d470 = ps.OverlayValues[470]
			}
			if len(ps.OverlayValues) > 471 && ps.OverlayValues[471].Loc != scm.LocNone {
				d471 = ps.OverlayValues[471]
			}
			if len(ps.OverlayValues) > 472 && ps.OverlayValues[472].Loc != scm.LocNone {
				d472 = ps.OverlayValues[472]
			}
			if len(ps.OverlayValues) > 473 && ps.OverlayValues[473].Loc != scm.LocNone {
				d473 = ps.OverlayValues[473]
			}
			if len(ps.OverlayValues) > 474 && ps.OverlayValues[474].Loc != scm.LocNone {
				d474 = ps.OverlayValues[474]
			}
			if len(ps.OverlayValues) > 475 && ps.OverlayValues[475].Loc != scm.LocNone {
				d475 = ps.OverlayValues[475]
			}
			if len(ps.OverlayValues) > 476 && ps.OverlayValues[476].Loc != scm.LocNone {
				d476 = ps.OverlayValues[476]
			}
			if len(ps.OverlayValues) > 477 && ps.OverlayValues[477].Loc != scm.LocNone {
				d477 = ps.OverlayValues[477]
			}
			if len(ps.OverlayValues) > 478 && ps.OverlayValues[478].Loc != scm.LocNone {
				d478 = ps.OverlayValues[478]
			}
			if len(ps.OverlayValues) > 479 && ps.OverlayValues[479].Loc != scm.LocNone {
				d479 = ps.OverlayValues[479]
			}
			if len(ps.OverlayValues) > 480 && ps.OverlayValues[480].Loc != scm.LocNone {
				d480 = ps.OverlayValues[480]
			}
			if len(ps.OverlayValues) > 481 && ps.OverlayValues[481].Loc != scm.LocNone {
				d481 = ps.OverlayValues[481]
			}
			if len(ps.OverlayValues) > 482 && ps.OverlayValues[482].Loc != scm.LocNone {
				d482 = ps.OverlayValues[482]
			}
			if len(ps.OverlayValues) > 483 && ps.OverlayValues[483].Loc != scm.LocNone {
				d483 = ps.OverlayValues[483]
			}
			if len(ps.OverlayValues) > 484 && ps.OverlayValues[484].Loc != scm.LocNone {
				d484 = ps.OverlayValues[484]
			}
			if len(ps.OverlayValues) > 485 && ps.OverlayValues[485].Loc != scm.LocNone {
				d485 = ps.OverlayValues[485]
			}
			if len(ps.OverlayValues) > 486 && ps.OverlayValues[486].Loc != scm.LocNone {
				d486 = ps.OverlayValues[486]
			}
			if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != scm.LocNone {
				d487 = ps.OverlayValues[487]
			}
			if len(ps.OverlayValues) > 488 && ps.OverlayValues[488].Loc != scm.LocNone {
				d488 = ps.OverlayValues[488]
			}
			if len(ps.OverlayValues) > 489 && ps.OverlayValues[489].Loc != scm.LocNone {
				d489 = ps.OverlayValues[489]
			}
			if len(ps.OverlayValues) > 490 && ps.OverlayValues[490].Loc != scm.LocNone {
				d490 = ps.OverlayValues[490]
			}
			if len(ps.OverlayValues) > 491 && ps.OverlayValues[491].Loc != scm.LocNone {
				d491 = ps.OverlayValues[491]
			}
			if len(ps.OverlayValues) > 492 && ps.OverlayValues[492].Loc != scm.LocNone {
				d492 = ps.OverlayValues[492]
			}
			if len(ps.OverlayValues) > 493 && ps.OverlayValues[493].Loc != scm.LocNone {
				d493 = ps.OverlayValues[493]
			}
			if len(ps.OverlayValues) > 494 && ps.OverlayValues[494].Loc != scm.LocNone {
				d494 = ps.OverlayValues[494]
			}
			if len(ps.OverlayValues) > 676 && ps.OverlayValues[676].Loc != scm.LocNone {
				d676 = ps.OverlayValues[676]
			}
			if len(ps.OverlayValues) > 677 && ps.OverlayValues[677].Loc != scm.LocNone {
				d677 = ps.OverlayValues[677]
			}
			if len(ps.OverlayValues) > 678 && ps.OverlayValues[678].Loc != scm.LocNone {
				d678 = ps.OverlayValues[678]
			}
			if len(ps.OverlayValues) > 679 && ps.OverlayValues[679].Loc != scm.LocNone {
				d679 = ps.OverlayValues[679]
			}
			if len(ps.OverlayValues) > 680 && ps.OverlayValues[680].Loc != scm.LocNone {
				d680 = ps.OverlayValues[680]
			}
			if len(ps.OverlayValues) > 682 && ps.OverlayValues[682].Loc != scm.LocNone {
				d682 = ps.OverlayValues[682]
			}
			if len(ps.OverlayValues) > 683 && ps.OverlayValues[683].Loc != scm.LocNone {
				d683 = ps.OverlayValues[683]
			}
			if len(ps.OverlayValues) > 684 && ps.OverlayValues[684].Loc != scm.LocNone {
				d684 = ps.OverlayValues[684]
			}
			if len(ps.OverlayValues) > 685 && ps.OverlayValues[685].Loc != scm.LocNone {
				d685 = ps.OverlayValues[685]
			}
			if len(ps.OverlayValues) > 686 && ps.OverlayValues[686].Loc != scm.LocNone {
				d686 = ps.OverlayValues[686]
			}
			if len(ps.OverlayValues) > 687 && ps.OverlayValues[687].Loc != scm.LocNone {
				d687 = ps.OverlayValues[687]
			}
			if len(ps.OverlayValues) > 688 && ps.OverlayValues[688].Loc != scm.LocNone {
				d688 = ps.OverlayValues[688]
			}
			if len(ps.OverlayValues) > 689 && ps.OverlayValues[689].Loc != scm.LocNone {
				d689 = ps.OverlayValues[689]
			}
			if len(ps.OverlayValues) > 691 && ps.OverlayValues[691].Loc != scm.LocNone {
				d691 = ps.OverlayValues[691]
			}
			if len(ps.OverlayValues) > 693 && ps.OverlayValues[693].Loc != scm.LocNone {
				d693 = ps.OverlayValues[693]
			}
			if len(ps.OverlayValues) > 694 && ps.OverlayValues[694].Loc != scm.LocNone {
				d694 = ps.OverlayValues[694]
			}
			if len(ps.OverlayValues) > 695 && ps.OverlayValues[695].Loc != scm.LocNone {
				d695 = ps.OverlayValues[695]
			}
			if len(ps.OverlayValues) > 696 && ps.OverlayValues[696].Loc != scm.LocNone {
				d696 = ps.OverlayValues[696]
			}
			if len(ps.OverlayValues) > 699 && ps.OverlayValues[699].Loc != scm.LocNone {
				d699 = ps.OverlayValues[699]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d4)
			d916 = d4
			_ = d916
			r156 := d4.Loc == scm.LocReg
			r157 := d4.Reg
			if r156 { ctx.ProtectReg(r157) }
			phiBase917 := ctx.AllocStack(int32(16))
			d918 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase917)+int32(192)}
			lbl40 := ctx.ReserveLabel()
			bbpos_4_0 := int32(-1)
			_ = bbpos_4_0
			bbpos_4_1 := int32(-1)
			_ = bbpos_4_1
			bbpos_4_2 := int32(-1)
			_ = bbpos_4_2
			bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d918 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(192)}
			ctx.EnsureDesc(&d916)
			ctx.EnsureDesc(&d916)
			var d919 scm.JITValueDesc
			if d916.Loc == scm.LocImm {
				d919 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d916.Imm.Int()))))}
			} else {
				r158 := ctx.AllocReg()
				ctx.EmitMovRegReg(r158, d916.Reg)
				ctx.EmitShlRegImm8(r158, 32)
				ctx.EmitShrRegImm8(r158, 32)
				d919 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r158}
				ctx.BindReg(r158, &d919)
			}
			var d920 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				r159 := ctx.AllocReg()
				ctx.EmitMovRegMem8(r159, fieldAddr)
				d920 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r159}
				ctx.BindReg(r159, &d920)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r160 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r160, thisptr.Reg, off)
				d920 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r160}
				ctx.BindReg(r160, &d920)
			}
			ctx.EnsureDesc(&d920)
			ctx.EnsureDesc(&d920)
			var d921 scm.JITValueDesc
			if d920.Loc == scm.LocImm {
				d921 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d920.Imm.Int()))))}
			} else {
				r161 := ctx.AllocReg()
				ctx.EmitMovRegReg(r161, d920.Reg)
				ctx.EmitShlRegImm8(r161, 56)
				ctx.EmitShrRegImm8(r161, 56)
				d921 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r161}
				ctx.BindReg(r161, &d921)
			}
			ctx.EnsureDesc(&d919)
			ctx.EnsureDesc(&d921)
			ctx.EnsureDesc(&d919)
			ctx.ProtectReg(d919.Reg)
			ctx.EnsureDesc(&d921)
			ctx.UnprotectReg(d919.Reg)
			var d922 scm.JITValueDesc
			if d919.Loc == scm.LocImm && d921.Loc == scm.LocImm {
				d922 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d919.Imm.Int() * d921.Imm.Int())}
			} else if d919.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d921.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d919.Imm.Int()))
				ctx.EmitImulInt64(scratch, d921.Reg)
				d922 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d922)
			} else if d921.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d919.Reg)
				ctx.EmitMovRegReg(scratch, d919.Reg)
				if d921.Imm.Int() >= -2147483648 && d921.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d921.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d921.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d922 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d922)
			} else {
				r162 := ctx.AllocRegExcept(d919.Reg, d921.Reg)
				ctx.EmitMovRegReg(r162, d919.Reg)
				ctx.EmitImulInt64(r162, d921.Reg)
				d922 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r162}
				ctx.BindReg(r162, &d922)
			}
			if d922.Loc == scm.LocReg && d919.Loc == scm.LocReg && d922.Reg == d919.Reg {
				ctx.TransferReg(d919.Reg)
				d919.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d919)
			ctx.FreeDesc(&d921)
			var d923 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 0
				r163 := ctx.AllocReg()
				r164 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r163, fieldAddr)
				ctx.EmitMovRegMem64(r164, fieldAddr+8)
				d923 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r163, Reg2: r164}
				ctx.BindReg(r163, &d923)
				ctx.BindReg(r164, &d923)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 0)
				r165 := ctx.AllocReg()
				r166 := ctx.AllocReg()
				ctx.EmitMovRegMem(r165, thisptr.Reg, off)
				ctx.EmitMovRegMem(r166, thisptr.Reg, off+8)
				d923 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r165, Reg2: r166}
				ctx.BindReg(r165, &d923)
				ctx.BindReg(r166, &d923)
			}
			ctx.EnsureDesc(&d922)
			var d924 scm.JITValueDesc
			if d922.Loc == scm.LocImm {
				d924 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d922.Imm.Int() / 64)}
			} else {
				r167 := ctx.AllocRegExcept(d922.Reg)
				ctx.EmitMovRegReg(r167, d922.Reg)
				ctx.EmitShrRegImm8(r167, 6)
				d924 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r167}
				ctx.BindReg(r167, &d924)
			}
			if d924.Loc == scm.LocReg && d922.Loc == scm.LocReg && d924.Reg == d922.Reg {
				ctx.TransferReg(d922.Reg)
				d922.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d924)
			r168 := ctx.AllocReg()
			ctx.EnsureDesc(&d924)
			ctx.EnsureDesc(&d923)
			if d924.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r168, uint64(d924.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r168, d924.Reg)
				ctx.EmitShlRegImm8(r168, 3)
			}
			if d923.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d923.Imm.Int()))
				ctx.EmitAddInt64(r168, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r168, d923.Reg)
			}
			r169 := ctx.AllocRegExcept(r168)
			ctx.EmitMovRegMem(r169, r168, 0)
			ctx.FreeReg(r168)
			d925 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r169}
			ctx.BindReg(r169, &d925)
			ctx.FreeDesc(&d924)
			ctx.EnsureDesc(&d922)
			var d926 scm.JITValueDesc
			if d922.Loc == scm.LocImm {
				d926 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d922.Imm.Int() % 64)}
			} else {
				r170 := ctx.AllocRegExcept(d922.Reg)
				ctx.EmitMovRegReg(r170, d922.Reg)
				ctx.EmitAndRegImm32(r170, 63)
				d926 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r170}
				ctx.BindReg(r170, &d926)
			}
			if d926.Loc == scm.LocReg && d922.Loc == scm.LocReg && d926.Reg == d922.Reg {
				ctx.TransferReg(d922.Reg)
				d922.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d925)
			ctx.EnsureDesc(&d926)
			var d927 scm.JITValueDesc
			if d925.Loc == scm.LocImm && d926.Loc == scm.LocImm {
				d927 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d925.Imm.Int()) << uint64(d926.Imm.Int())))}
			} else if d926.Loc == scm.LocImm {
				r171 := ctx.AllocRegExcept(d925.Reg)
				ctx.EmitMovRegReg(r171, d925.Reg)
				ctx.EmitShlRegImm8(r171, uint8(d926.Imm.Int()))
				d927 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r171}
				ctx.BindReg(r171, &d927)
			} else {
				{
					shiftSrc := d925.Reg
					r172 := ctx.AllocRegExcept(d925.Reg)
					ctx.EmitMovRegReg(r172, d925.Reg)
					shiftSrc = r172
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d926.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d926.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d926.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d927 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d927)
				}
			}
			if d927.Loc == scm.LocReg && d925.Loc == scm.LocReg && d927.Reg == d925.Reg {
				ctx.TransferReg(d925.Reg)
				d925.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d925)
			ctx.FreeDesc(&d926)
			ctx.EnsureDesc(&d922)
			var d928 scm.JITValueDesc
			if d922.Loc == scm.LocImm {
				d928 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d922.Imm.Int() % 64)}
			} else {
				r173 := ctx.AllocRegExcept(d922.Reg)
				ctx.EmitMovRegReg(r173, d922.Reg)
				ctx.EmitAndRegImm32(r173, 63)
				d928 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r173}
				ctx.BindReg(r173, &d928)
			}
			if d928.Loc == scm.LocReg && d922.Loc == scm.LocReg && d928.Reg == d922.Reg {
				ctx.TransferReg(d922.Reg)
				d922.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d920)
			ctx.EnsureDesc(&d920)
			var d929 scm.JITValueDesc
			if d920.Loc == scm.LocImm {
				d929 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d920.Imm.Int()))))}
			} else {
				r174 := ctx.AllocReg()
				ctx.EmitMovRegReg(r174, d920.Reg)
				ctx.EmitShlRegImm8(r174, 56)
				ctx.EmitShrRegImm8(r174, 56)
				d929 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r174}
				ctx.BindReg(r174, &d929)
			}
			ctx.EnsureDesc(&d928)
			ctx.EnsureDesc(&d929)
			ctx.EnsureDesc(&d928)
			ctx.ProtectReg(d928.Reg)
			ctx.EnsureDesc(&d929)
			ctx.UnprotectReg(d928.Reg)
			var d930 scm.JITValueDesc
			if d928.Loc == scm.LocImm && d929.Loc == scm.LocImm {
				d930 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d928.Imm.Int() + d929.Imm.Int())}
			} else if d929.Loc == scm.LocImm && d929.Imm.Int() == 0 {
				r175 := ctx.AllocRegExcept(d928.Reg)
				ctx.EmitMovRegReg(r175, d928.Reg)
				d930 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r175}
				ctx.BindReg(r175, &d930)
			} else if d928.Loc == scm.LocImm && d928.Imm.Int() == 0 {
				d930 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d929.Reg}
				ctx.BindReg(d929.Reg, &d930)
			} else if d928.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d929.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d928.Imm.Int()))
				ctx.EmitAddInt64(scratch, d929.Reg)
				d930 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d930)
			} else if d929.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d928.Reg)
				ctx.EmitMovRegReg(scratch, d928.Reg)
				if d929.Imm.Int() >= -2147483648 && d929.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d929.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d929.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d930 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d930)
			} else {
				r176 := ctx.AllocRegExcept(d928.Reg, d929.Reg)
				ctx.EmitMovRegReg(r176, d928.Reg)
				ctx.EmitAddInt64(r176, d929.Reg)
				d930 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r176}
				ctx.BindReg(r176, &d930)
			}
			if d930.Loc == scm.LocReg && d928.Loc == scm.LocReg && d930.Reg == d928.Reg {
				ctx.TransferReg(d928.Reg)
				d928.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d928)
			ctx.FreeDesc(&d929)
			ctx.EnsureDesc(&d930)
			var d931 scm.JITValueDesc
			if d930.Loc == scm.LocImm {
				d931 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d930.Imm.Int()) > uint64(64))}
			} else {
				r177 := ctx.AllocRegExcept(d930.Reg)
				ctx.EmitCmpRegImm32(d930.Reg, 64)
				ctx.EmitSetcc(r177, scm.CcA)
				d931 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r177}
				ctx.BindReg(r177, &d931)
			}
			ctx.FreeDesc(&d930)
			d932 = d931
			ctx.EnsureDesc(&d932)
			if d932.Loc != scm.LocImm && d932.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl41 := ctx.ReserveLabel()
			lbl42 := ctx.ReserveLabel()
			lbl43 := ctx.ReserveLabel()
			lbl44 := ctx.ReserveLabel()
			if d932.Loc == scm.LocImm {
				if d932.Imm.Bool() {
					ctx.MarkLabel(lbl43)
					ctx.EmitJmp(lbl41)
				} else {
					ctx.MarkLabel(lbl44)
			ctx.EnsureDesc(&d927)
			if d927.Loc == scm.LocReg {
				ctx.ProtectReg(d927.Reg)
			} else if d927.Loc == scm.LocRegPair {
				ctx.ProtectReg(d927.Reg)
				ctx.ProtectReg(d927.Reg2)
			}
			d933 = d927
			if d933.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d933)
			ctx.EmitStoreToStack(d933, int32(bbs[2].PhiBase)+int32(0))
			if d927.Loc == scm.LocReg {
				ctx.UnprotectReg(d927.Reg)
			} else if d927.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d927.Reg)
				ctx.UnprotectReg(d927.Reg2)
			}
					ctx.EmitJmp(lbl42)
				}
			} else {
				ctx.EmitCmpRegImm32(d932.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl43)
				ctx.EmitJmp(lbl44)
				ctx.MarkLabel(lbl43)
				ctx.EmitJmp(lbl41)
				ctx.MarkLabel(lbl44)
			ctx.EnsureDesc(&d927)
			if d927.Loc == scm.LocReg {
				ctx.ProtectReg(d927.Reg)
			} else if d927.Loc == scm.LocRegPair {
				ctx.ProtectReg(d927.Reg)
				ctx.ProtectReg(d927.Reg2)
			}
			d934 = d927
			if d934.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d934)
			ctx.EmitStoreToStack(d934, int32(bbs[2].PhiBase)+int32(0))
			if d927.Loc == scm.LocReg {
				ctx.UnprotectReg(d927.Reg)
			} else if d927.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d927.Reg)
				ctx.UnprotectReg(d927.Reg2)
			}
				ctx.EmitJmp(lbl42)
			}
			ctx.FreeDesc(&d931)
			bbpos_4_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl42)
			ctx.ResolveFixups()
			d918 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(192)}
			ctx.EnsureDesc(&d920)
			ctx.EnsureDesc(&d920)
			var d935 scm.JITValueDesc
			if d920.Loc == scm.LocImm {
				d935 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d920.Imm.Int()))))}
			} else {
				r178 := ctx.AllocReg()
				ctx.EmitMovRegReg(r178, d920.Reg)
				ctx.EmitShlRegImm8(r178, 56)
				ctx.EmitShrRegImm8(r178, 56)
				d935 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r178}
				ctx.BindReg(r178, &d935)
			}
			d936 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d935)
			ctx.EnsureDesc(&d936)
			ctx.ProtectReg(d936.Reg)
			ctx.EnsureDesc(&d935)
			ctx.UnprotectReg(d936.Reg)
			var d937 scm.JITValueDesc
			if d936.Loc == scm.LocImm && d935.Loc == scm.LocImm {
				d937 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d936.Imm.Int() - d935.Imm.Int())}
			} else if d935.Loc == scm.LocImm && d935.Imm.Int() == 0 {
				r179 := ctx.AllocRegExcept(d936.Reg)
				ctx.EmitMovRegReg(r179, d936.Reg)
				d937 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r179}
				ctx.BindReg(r179, &d937)
			} else if d936.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d935.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d936.Imm.Int()))
				ctx.EmitSubInt64(scratch, d935.Reg)
				d937 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d937)
			} else if d935.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d936.Reg)
				ctx.EmitMovRegReg(scratch, d936.Reg)
				if d935.Imm.Int() >= -2147483648 && d935.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d935.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d935.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d937 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d937)
			} else {
				r180 := ctx.AllocRegExcept(d936.Reg, d935.Reg)
				ctx.EmitMovRegReg(r180, d936.Reg)
				ctx.EmitSubInt64(r180, d935.Reg)
				d937 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r180}
				ctx.BindReg(r180, &d937)
			}
			if d937.Loc == scm.LocReg && d936.Loc == scm.LocReg && d937.Reg == d936.Reg {
				ctx.TransferReg(d936.Reg)
				d936.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d935)
			ctx.EnsureDesc(&d918)
			ctx.EnsureDesc(&d937)
			var d938 scm.JITValueDesc
			if d918.Loc == scm.LocImm && d937.Loc == scm.LocImm {
				d938 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d918.Imm.Int()) >> uint64(d937.Imm.Int())))}
			} else if d937.Loc == scm.LocImm {
				r181 := ctx.AllocRegExcept(d918.Reg)
				ctx.EmitMovRegReg(r181, d918.Reg)
				ctx.EmitShrRegImm8(r181, uint8(d937.Imm.Int()))
				d938 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r181}
				ctx.BindReg(r181, &d938)
			} else {
				{
					shiftSrc := d918.Reg
					r182 := ctx.AllocRegExcept(d918.Reg)
					ctx.EmitMovRegReg(r182, d918.Reg)
					shiftSrc = r182
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d937.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d937.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d937.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d938 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d938)
				}
			}
			if d938.Loc == scm.LocReg && d918.Loc == scm.LocReg && d938.Reg == d918.Reg {
				ctx.TransferReg(d918.Reg)
				d918.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d918)
			ctx.FreeDesc(&d937)
			r183 := ctx.AllocReg()
			ctx.EnsureDesc(&d938)
			ctx.EnsureDesc(&d938)
			if d938.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r183, d938)
			}
			ctx.EmitJmp(lbl40)
			bbpos_4_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl41)
			ctx.ResolveFixups()
			d918 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(192)}
			ctx.EnsureDesc(&d922)
			var d939 scm.JITValueDesc
			if d922.Loc == scm.LocImm {
				d939 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d922.Imm.Int() / 64)}
			} else {
				r184 := ctx.AllocRegExcept(d922.Reg)
				ctx.EmitMovRegReg(r184, d922.Reg)
				ctx.EmitShrRegImm8(r184, 6)
				d939 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r184}
				ctx.BindReg(r184, &d939)
			}
			if d939.Loc == scm.LocReg && d922.Loc == scm.LocReg && d939.Reg == d922.Reg {
				ctx.TransferReg(d922.Reg)
				d922.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d939)
			ctx.EnsureDesc(&d939)
			var d940 scm.JITValueDesc
			if d939.Loc == scm.LocImm {
				d940 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d939.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d939.Reg)
				ctx.EmitMovRegReg(scratch, d939.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d940 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d940)
			}
			if d940.Loc == scm.LocReg && d939.Loc == scm.LocReg && d940.Reg == d939.Reg {
				ctx.TransferReg(d939.Reg)
				d939.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d939)
			ctx.EnsureDesc(&d940)
			r185 := ctx.AllocReg()
			ctx.EnsureDesc(&d940)
			ctx.EnsureDesc(&d923)
			if d940.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r185, uint64(d940.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r185, d940.Reg)
				ctx.EmitShlRegImm8(r185, 3)
			}
			if d923.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d923.Imm.Int()))
				ctx.EmitAddInt64(r185, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r185, d923.Reg)
			}
			r186 := ctx.AllocRegExcept(r185)
			ctx.EmitMovRegMem(r186, r185, 0)
			ctx.FreeReg(r185)
			d941 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r186}
			ctx.BindReg(r186, &d941)
			ctx.FreeDesc(&d940)
			ctx.EnsureDesc(&d922)
			var d942 scm.JITValueDesc
			if d922.Loc == scm.LocImm {
				d942 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d922.Imm.Int() % 64)}
			} else {
				r187 := ctx.AllocRegExcept(d922.Reg)
				ctx.EmitMovRegReg(r187, d922.Reg)
				ctx.EmitAndRegImm32(r187, 63)
				d942 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r187}
				ctx.BindReg(r187, &d942)
			}
			if d942.Loc == scm.LocReg && d922.Loc == scm.LocReg && d942.Reg == d922.Reg {
				ctx.TransferReg(d922.Reg)
				d922.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d922)
			d943 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d942)
			ctx.EnsureDesc(&d943)
			ctx.ProtectReg(d943.Reg)
			ctx.EnsureDesc(&d942)
			ctx.UnprotectReg(d943.Reg)
			var d944 scm.JITValueDesc
			if d943.Loc == scm.LocImm && d942.Loc == scm.LocImm {
				d944 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d943.Imm.Int() - d942.Imm.Int())}
			} else if d942.Loc == scm.LocImm && d942.Imm.Int() == 0 {
				r188 := ctx.AllocRegExcept(d943.Reg)
				ctx.EmitMovRegReg(r188, d943.Reg)
				d944 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r188}
				ctx.BindReg(r188, &d944)
			} else if d943.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d942.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d943.Imm.Int()))
				ctx.EmitSubInt64(scratch, d942.Reg)
				d944 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d944)
			} else if d942.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d943.Reg)
				ctx.EmitMovRegReg(scratch, d943.Reg)
				if d942.Imm.Int() >= -2147483648 && d942.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d942.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d942.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d944 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d944)
			} else {
				r189 := ctx.AllocRegExcept(d943.Reg, d942.Reg)
				ctx.EmitMovRegReg(r189, d943.Reg)
				ctx.EmitSubInt64(r189, d942.Reg)
				d944 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r189}
				ctx.BindReg(r189, &d944)
			}
			if d944.Loc == scm.LocReg && d943.Loc == scm.LocReg && d944.Reg == d943.Reg {
				ctx.TransferReg(d943.Reg)
				d943.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d942)
			ctx.EnsureDesc(&d941)
			ctx.EnsureDesc(&d944)
			var d945 scm.JITValueDesc
			if d941.Loc == scm.LocImm && d944.Loc == scm.LocImm {
				d945 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d941.Imm.Int()) >> uint64(d944.Imm.Int())))}
			} else if d944.Loc == scm.LocImm {
				r190 := ctx.AllocRegExcept(d941.Reg)
				ctx.EmitMovRegReg(r190, d941.Reg)
				ctx.EmitShrRegImm8(r190, uint8(d944.Imm.Int()))
				d945 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r190}
				ctx.BindReg(r190, &d945)
			} else {
				{
					shiftSrc := d941.Reg
					r191 := ctx.AllocRegExcept(d941.Reg)
					ctx.EmitMovRegReg(r191, d941.Reg)
					shiftSrc = r191
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d944.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d944.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d944.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d945 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d945)
				}
			}
			if d945.Loc == scm.LocReg && d941.Loc == scm.LocReg && d945.Reg == d941.Reg {
				ctx.TransferReg(d941.Reg)
				d941.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d941)
			ctx.FreeDesc(&d944)
			ctx.EnsureDesc(&d927)
			ctx.EnsureDesc(&d945)
			var d946 scm.JITValueDesc
			if d927.Loc == scm.LocImm && d945.Loc == scm.LocImm {
				d946 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d927.Imm.Int() | d945.Imm.Int())}
			} else if d927.Loc == scm.LocImm && d927.Imm.Int() == 0 {
				d946 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d945.Reg}
				ctx.BindReg(d945.Reg, &d946)
			} else if d945.Loc == scm.LocImm && d945.Imm.Int() == 0 {
				r192 := ctx.AllocRegExcept(d927.Reg)
				ctx.EmitMovRegReg(r192, d927.Reg)
				d946 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r192}
				ctx.BindReg(r192, &d946)
			} else if d927.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d945.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d927.Imm.Int()))
				ctx.EmitOrInt64(scratch, d945.Reg)
				d946 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d946)
			} else if d945.Loc == scm.LocImm {
				r193 := ctx.AllocRegExcept(d927.Reg)
				ctx.EmitMovRegReg(r193, d927.Reg)
				if d945.Imm.Int() >= -2147483648 && d945.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r193, int32(d945.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d945.Imm.Int()))
					ctx.EmitOrInt64(r193, scm.RegR11)
				}
				d946 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r193}
				ctx.BindReg(r193, &d946)
			} else {
				r194 := ctx.AllocRegExcept(d927.Reg, d945.Reg)
				ctx.EmitMovRegReg(r194, d927.Reg)
				ctx.EmitOrInt64(r194, d945.Reg)
				d946 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r194}
				ctx.BindReg(r194, &d946)
			}
			if d946.Loc == scm.LocReg && d927.Loc == scm.LocReg && d946.Reg == d927.Reg {
				ctx.TransferReg(d927.Reg)
				d927.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d945)
			ctx.EnsureDesc(&d946)
			if d946.Loc == scm.LocReg {
				ctx.ProtectReg(d946.Reg)
			} else if d946.Loc == scm.LocRegPair {
				ctx.ProtectReg(d946.Reg)
				ctx.ProtectReg(d946.Reg2)
			}
			d947 = d946
			if d947.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d947)
			ctx.EmitStoreToStack(d947, int32(bbs[2].PhiBase)+int32(0))
			if d946.Loc == scm.LocReg {
				ctx.UnprotectReg(d946.Reg)
			} else if d946.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d946.Reg)
				ctx.UnprotectReg(d946.Reg2)
			}
			ctx.EmitJmp(lbl42)
			ctx.MarkLabel(lbl40)
			d948 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r183}
			ctx.BindReg(r183, &d948)
			ctx.BindReg(r183, &d948)
			if r156 { ctx.UnprotectReg(r157) }
			ctx.EnsureDesc(&d948)
			ctx.EnsureDesc(&d948)
			var d949 scm.JITValueDesc
			if d948.Loc == scm.LocImm {
				d949 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d948.Imm.Int()))))}
			} else {
				r195 := ctx.AllocReg()
				ctx.EmitMovRegReg(r195, d948.Reg)
				d949 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r195}
				ctx.BindReg(r195, &d949)
			}
			ctx.FreeDesc(&d948)
			var d950 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 32
				r196 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r196, fieldAddr)
				d950 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r196}
				ctx.BindReg(r196, &d950)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 32)
				r197 := ctx.AllocReg()
				ctx.EmitMovRegMem(r197, thisptr.Reg, off)
				d950 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r197}
				ctx.BindReg(r197, &d950)
			}
			ctx.EnsureDesc(&d949)
			ctx.EnsureDesc(&d950)
			ctx.EnsureDesc(&d949)
			ctx.ProtectReg(d949.Reg)
			ctx.EnsureDesc(&d950)
			ctx.UnprotectReg(d949.Reg)
			var d951 scm.JITValueDesc
			if d949.Loc == scm.LocImm && d950.Loc == scm.LocImm {
				d951 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d949.Imm.Int() + d950.Imm.Int())}
			} else if d950.Loc == scm.LocImm && d950.Imm.Int() == 0 {
				r198 := ctx.AllocRegExcept(d949.Reg)
				ctx.EmitMovRegReg(r198, d949.Reg)
				d951 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r198}
				ctx.BindReg(r198, &d951)
			} else if d949.Loc == scm.LocImm && d949.Imm.Int() == 0 {
				d951 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d950.Reg}
				ctx.BindReg(d950.Reg, &d951)
			} else if d949.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d950.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d949.Imm.Int()))
				ctx.EmitAddInt64(scratch, d950.Reg)
				d951 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d951)
			} else if d950.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d949.Reg)
				ctx.EmitMovRegReg(scratch, d949.Reg)
				if d950.Imm.Int() >= -2147483648 && d950.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d950.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d950.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d951 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d951)
			} else {
				r199 := ctx.AllocRegExcept(d949.Reg, d950.Reg)
				ctx.EmitMovRegReg(r199, d949.Reg)
				ctx.EmitAddInt64(r199, d950.Reg)
				d951 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r199}
				ctx.BindReg(r199, &d951)
			}
			if d951.Loc == scm.LocReg && d949.Loc == scm.LocReg && d951.Reg == d949.Reg {
				ctx.TransferReg(d949.Reg)
				d949.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d949)
			ctx.EnsureDesc(&d4)
			d952 = d4
			_ = d952
			r200 := d4.Loc == scm.LocReg
			r201 := d4.Reg
			if r200 { ctx.ProtectReg(r201) }
			phiBase953 := ctx.AllocStack(int32(16))
			d954 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase953)+int32(208)}
			lbl45 := ctx.ReserveLabel()
			bbpos_5_0 := int32(-1)
			_ = bbpos_5_0
			bbpos_5_1 := int32(-1)
			_ = bbpos_5_1
			bbpos_5_2 := int32(-1)
			_ = bbpos_5_2
			bbpos_5_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d954 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(208)}
			ctx.EnsureDesc(&d952)
			ctx.EnsureDesc(&d952)
			var d955 scm.JITValueDesc
			if d952.Loc == scm.LocImm {
				d955 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d952.Imm.Int()))))}
			} else {
				r202 := ctx.AllocReg()
				ctx.EmitMovRegReg(r202, d952.Reg)
				ctx.EmitShlRegImm8(r202, 32)
				ctx.EmitShrRegImm8(r202, 32)
				d955 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r202}
				ctx.BindReg(r202, &d955)
			}
			var d956 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				r203 := ctx.AllocReg()
				ctx.EmitMovRegMem8(r203, fieldAddr)
				d956 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r203}
				ctx.BindReg(r203, &d956)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r204 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r204, thisptr.Reg, off)
				d956 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r204}
				ctx.BindReg(r204, &d956)
			}
			ctx.EnsureDesc(&d956)
			ctx.EnsureDesc(&d956)
			var d957 scm.JITValueDesc
			if d956.Loc == scm.LocImm {
				d957 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d956.Imm.Int()))))}
			} else {
				r205 := ctx.AllocReg()
				ctx.EmitMovRegReg(r205, d956.Reg)
				ctx.EmitShlRegImm8(r205, 56)
				ctx.EmitShrRegImm8(r205, 56)
				d957 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r205}
				ctx.BindReg(r205, &d957)
			}
			ctx.EnsureDesc(&d955)
			ctx.EnsureDesc(&d957)
			ctx.EnsureDesc(&d955)
			ctx.ProtectReg(d955.Reg)
			ctx.EnsureDesc(&d957)
			ctx.UnprotectReg(d955.Reg)
			var d958 scm.JITValueDesc
			if d955.Loc == scm.LocImm && d957.Loc == scm.LocImm {
				d958 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d955.Imm.Int() * d957.Imm.Int())}
			} else if d955.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d957.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d955.Imm.Int()))
				ctx.EmitImulInt64(scratch, d957.Reg)
				d958 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d958)
			} else if d957.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d955.Reg)
				ctx.EmitMovRegReg(scratch, d955.Reg)
				if d957.Imm.Int() >= -2147483648 && d957.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d957.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d957.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d958 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d958)
			} else {
				r206 := ctx.AllocRegExcept(d955.Reg, d957.Reg)
				ctx.EmitMovRegReg(r206, d955.Reg)
				ctx.EmitImulInt64(r206, d957.Reg)
				d958 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r206}
				ctx.BindReg(r206, &d958)
			}
			if d958.Loc == scm.LocReg && d955.Loc == scm.LocReg && d958.Reg == d955.Reg {
				ctx.TransferReg(d955.Reg)
				d955.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d955)
			ctx.FreeDesc(&d957)
			var d959 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				r207 := ctx.AllocReg()
				r208 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r207, fieldAddr)
				ctx.EmitMovRegMem64(r208, fieldAddr+8)
				d959 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r207, Reg2: r208}
				ctx.BindReg(r207, &d959)
				ctx.BindReg(r208, &d959)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				r209 := ctx.AllocReg()
				r210 := ctx.AllocReg()
				ctx.EmitMovRegMem(r209, thisptr.Reg, off)
				ctx.EmitMovRegMem(r210, thisptr.Reg, off+8)
				d959 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r209, Reg2: r210}
				ctx.BindReg(r209, &d959)
				ctx.BindReg(r210, &d959)
			}
			ctx.EnsureDesc(&d958)
			var d960 scm.JITValueDesc
			if d958.Loc == scm.LocImm {
				d960 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d958.Imm.Int() / 64)}
			} else {
				r211 := ctx.AllocRegExcept(d958.Reg)
				ctx.EmitMovRegReg(r211, d958.Reg)
				ctx.EmitShrRegImm8(r211, 6)
				d960 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r211}
				ctx.BindReg(r211, &d960)
			}
			if d960.Loc == scm.LocReg && d958.Loc == scm.LocReg && d960.Reg == d958.Reg {
				ctx.TransferReg(d958.Reg)
				d958.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d960)
			r212 := ctx.AllocReg()
			ctx.EnsureDesc(&d960)
			ctx.EnsureDesc(&d959)
			if d960.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r212, uint64(d960.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r212, d960.Reg)
				ctx.EmitShlRegImm8(r212, 3)
			}
			if d959.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d959.Imm.Int()))
				ctx.EmitAddInt64(r212, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r212, d959.Reg)
			}
			r213 := ctx.AllocRegExcept(r212)
			ctx.EmitMovRegMem(r213, r212, 0)
			ctx.FreeReg(r212)
			d961 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r213}
			ctx.BindReg(r213, &d961)
			ctx.FreeDesc(&d960)
			ctx.EnsureDesc(&d958)
			var d962 scm.JITValueDesc
			if d958.Loc == scm.LocImm {
				d962 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d958.Imm.Int() % 64)}
			} else {
				r214 := ctx.AllocRegExcept(d958.Reg)
				ctx.EmitMovRegReg(r214, d958.Reg)
				ctx.EmitAndRegImm32(r214, 63)
				d962 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r214}
				ctx.BindReg(r214, &d962)
			}
			if d962.Loc == scm.LocReg && d958.Loc == scm.LocReg && d962.Reg == d958.Reg {
				ctx.TransferReg(d958.Reg)
				d958.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d961)
			ctx.EnsureDesc(&d962)
			var d963 scm.JITValueDesc
			if d961.Loc == scm.LocImm && d962.Loc == scm.LocImm {
				d963 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d961.Imm.Int()) << uint64(d962.Imm.Int())))}
			} else if d962.Loc == scm.LocImm {
				r215 := ctx.AllocRegExcept(d961.Reg)
				ctx.EmitMovRegReg(r215, d961.Reg)
				ctx.EmitShlRegImm8(r215, uint8(d962.Imm.Int()))
				d963 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r215}
				ctx.BindReg(r215, &d963)
			} else {
				{
					shiftSrc := d961.Reg
					r216 := ctx.AllocRegExcept(d961.Reg)
					ctx.EmitMovRegReg(r216, d961.Reg)
					shiftSrc = r216
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d962.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d962.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d962.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d963 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d963)
				}
			}
			if d963.Loc == scm.LocReg && d961.Loc == scm.LocReg && d963.Reg == d961.Reg {
				ctx.TransferReg(d961.Reg)
				d961.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d961)
			ctx.FreeDesc(&d962)
			ctx.EnsureDesc(&d958)
			var d964 scm.JITValueDesc
			if d958.Loc == scm.LocImm {
				d964 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d958.Imm.Int() % 64)}
			} else {
				r217 := ctx.AllocRegExcept(d958.Reg)
				ctx.EmitMovRegReg(r217, d958.Reg)
				ctx.EmitAndRegImm32(r217, 63)
				d964 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r217}
				ctx.BindReg(r217, &d964)
			}
			if d964.Loc == scm.LocReg && d958.Loc == scm.LocReg && d964.Reg == d958.Reg {
				ctx.TransferReg(d958.Reg)
				d958.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d956)
			ctx.EnsureDesc(&d956)
			var d965 scm.JITValueDesc
			if d956.Loc == scm.LocImm {
				d965 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d956.Imm.Int()))))}
			} else {
				r218 := ctx.AllocReg()
				ctx.EmitMovRegReg(r218, d956.Reg)
				ctx.EmitShlRegImm8(r218, 56)
				ctx.EmitShrRegImm8(r218, 56)
				d965 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r218}
				ctx.BindReg(r218, &d965)
			}
			ctx.EnsureDesc(&d964)
			ctx.EnsureDesc(&d965)
			ctx.EnsureDesc(&d964)
			ctx.ProtectReg(d964.Reg)
			ctx.EnsureDesc(&d965)
			ctx.UnprotectReg(d964.Reg)
			var d966 scm.JITValueDesc
			if d964.Loc == scm.LocImm && d965.Loc == scm.LocImm {
				d966 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d964.Imm.Int() + d965.Imm.Int())}
			} else if d965.Loc == scm.LocImm && d965.Imm.Int() == 0 {
				r219 := ctx.AllocRegExcept(d964.Reg)
				ctx.EmitMovRegReg(r219, d964.Reg)
				d966 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r219}
				ctx.BindReg(r219, &d966)
			} else if d964.Loc == scm.LocImm && d964.Imm.Int() == 0 {
				d966 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d965.Reg}
				ctx.BindReg(d965.Reg, &d966)
			} else if d964.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d965.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d964.Imm.Int()))
				ctx.EmitAddInt64(scratch, d965.Reg)
				d966 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d966)
			} else if d965.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d964.Reg)
				ctx.EmitMovRegReg(scratch, d964.Reg)
				if d965.Imm.Int() >= -2147483648 && d965.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d965.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d965.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d966 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d966)
			} else {
				r220 := ctx.AllocRegExcept(d964.Reg, d965.Reg)
				ctx.EmitMovRegReg(r220, d964.Reg)
				ctx.EmitAddInt64(r220, d965.Reg)
				d966 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r220}
				ctx.BindReg(r220, &d966)
			}
			if d966.Loc == scm.LocReg && d964.Loc == scm.LocReg && d966.Reg == d964.Reg {
				ctx.TransferReg(d964.Reg)
				d964.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d964)
			ctx.FreeDesc(&d965)
			ctx.EnsureDesc(&d966)
			var d967 scm.JITValueDesc
			if d966.Loc == scm.LocImm {
				d967 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d966.Imm.Int()) > uint64(64))}
			} else {
				r221 := ctx.AllocRegExcept(d966.Reg)
				ctx.EmitCmpRegImm32(d966.Reg, 64)
				ctx.EmitSetcc(r221, scm.CcA)
				d967 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r221}
				ctx.BindReg(r221, &d967)
			}
			ctx.FreeDesc(&d966)
			d968 = d967
			ctx.EnsureDesc(&d968)
			if d968.Loc != scm.LocImm && d968.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl46 := ctx.ReserveLabel()
			lbl47 := ctx.ReserveLabel()
			lbl48 := ctx.ReserveLabel()
			lbl49 := ctx.ReserveLabel()
			if d968.Loc == scm.LocImm {
				if d968.Imm.Bool() {
					ctx.MarkLabel(lbl48)
					ctx.EmitJmp(lbl46)
				} else {
					ctx.MarkLabel(lbl49)
			ctx.EnsureDesc(&d963)
			if d963.Loc == scm.LocReg {
				ctx.ProtectReg(d963.Reg)
			} else if d963.Loc == scm.LocRegPair {
				ctx.ProtectReg(d963.Reg)
				ctx.ProtectReg(d963.Reg2)
			}
			d969 = d963
			if d969.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d969)
			ctx.EmitStoreToStack(d969, int32(bbs[2].PhiBase)+int32(0))
			if d963.Loc == scm.LocReg {
				ctx.UnprotectReg(d963.Reg)
			} else if d963.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d963.Reg)
				ctx.UnprotectReg(d963.Reg2)
			}
					ctx.EmitJmp(lbl47)
				}
			} else {
				ctx.EmitCmpRegImm32(d968.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl48)
				ctx.EmitJmp(lbl49)
				ctx.MarkLabel(lbl48)
				ctx.EmitJmp(lbl46)
				ctx.MarkLabel(lbl49)
			ctx.EnsureDesc(&d963)
			if d963.Loc == scm.LocReg {
				ctx.ProtectReg(d963.Reg)
			} else if d963.Loc == scm.LocRegPair {
				ctx.ProtectReg(d963.Reg)
				ctx.ProtectReg(d963.Reg2)
			}
			d970 = d963
			if d970.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d970)
			ctx.EmitStoreToStack(d970, int32(bbs[2].PhiBase)+int32(0))
			if d963.Loc == scm.LocReg {
				ctx.UnprotectReg(d963.Reg)
			} else if d963.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d963.Reg)
				ctx.UnprotectReg(d963.Reg2)
			}
				ctx.EmitJmp(lbl47)
			}
			ctx.FreeDesc(&d967)
			bbpos_5_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl47)
			ctx.ResolveFixups()
			d954 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(208)}
			ctx.EnsureDesc(&d956)
			ctx.EnsureDesc(&d956)
			var d971 scm.JITValueDesc
			if d956.Loc == scm.LocImm {
				d971 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d956.Imm.Int()))))}
			} else {
				r222 := ctx.AllocReg()
				ctx.EmitMovRegReg(r222, d956.Reg)
				ctx.EmitShlRegImm8(r222, 56)
				ctx.EmitShrRegImm8(r222, 56)
				d971 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r222}
				ctx.BindReg(r222, &d971)
			}
			d972 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d971)
			ctx.EnsureDesc(&d972)
			ctx.ProtectReg(d972.Reg)
			ctx.EnsureDesc(&d971)
			ctx.UnprotectReg(d972.Reg)
			var d973 scm.JITValueDesc
			if d972.Loc == scm.LocImm && d971.Loc == scm.LocImm {
				d973 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d972.Imm.Int() - d971.Imm.Int())}
			} else if d971.Loc == scm.LocImm && d971.Imm.Int() == 0 {
				r223 := ctx.AllocRegExcept(d972.Reg)
				ctx.EmitMovRegReg(r223, d972.Reg)
				d973 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r223}
				ctx.BindReg(r223, &d973)
			} else if d972.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d971.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d972.Imm.Int()))
				ctx.EmitSubInt64(scratch, d971.Reg)
				d973 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d973)
			} else if d971.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d972.Reg)
				ctx.EmitMovRegReg(scratch, d972.Reg)
				if d971.Imm.Int() >= -2147483648 && d971.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d971.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d971.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d973 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d973)
			} else {
				r224 := ctx.AllocRegExcept(d972.Reg, d971.Reg)
				ctx.EmitMovRegReg(r224, d972.Reg)
				ctx.EmitSubInt64(r224, d971.Reg)
				d973 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r224}
				ctx.BindReg(r224, &d973)
			}
			if d973.Loc == scm.LocReg && d972.Loc == scm.LocReg && d973.Reg == d972.Reg {
				ctx.TransferReg(d972.Reg)
				d972.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d971)
			ctx.EnsureDesc(&d954)
			ctx.EnsureDesc(&d973)
			var d974 scm.JITValueDesc
			if d954.Loc == scm.LocImm && d973.Loc == scm.LocImm {
				d974 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d954.Imm.Int()) >> uint64(d973.Imm.Int())))}
			} else if d973.Loc == scm.LocImm {
				r225 := ctx.AllocRegExcept(d954.Reg)
				ctx.EmitMovRegReg(r225, d954.Reg)
				ctx.EmitShrRegImm8(r225, uint8(d973.Imm.Int()))
				d974 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r225}
				ctx.BindReg(r225, &d974)
			} else {
				{
					shiftSrc := d954.Reg
					r226 := ctx.AllocRegExcept(d954.Reg)
					ctx.EmitMovRegReg(r226, d954.Reg)
					shiftSrc = r226
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d973.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d973.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d973.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d974 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d974)
				}
			}
			if d974.Loc == scm.LocReg && d954.Loc == scm.LocReg && d974.Reg == d954.Reg {
				ctx.TransferReg(d954.Reg)
				d954.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d954)
			ctx.FreeDesc(&d973)
			r227 := ctx.AllocReg()
			ctx.EnsureDesc(&d974)
			ctx.EnsureDesc(&d974)
			if d974.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r227, d974)
			}
			ctx.EmitJmp(lbl45)
			bbpos_5_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl46)
			ctx.ResolveFixups()
			d954 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(208)}
			ctx.EnsureDesc(&d958)
			var d975 scm.JITValueDesc
			if d958.Loc == scm.LocImm {
				d975 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d958.Imm.Int() / 64)}
			} else {
				r228 := ctx.AllocRegExcept(d958.Reg)
				ctx.EmitMovRegReg(r228, d958.Reg)
				ctx.EmitShrRegImm8(r228, 6)
				d975 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r228}
				ctx.BindReg(r228, &d975)
			}
			if d975.Loc == scm.LocReg && d958.Loc == scm.LocReg && d975.Reg == d958.Reg {
				ctx.TransferReg(d958.Reg)
				d958.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d975)
			ctx.EnsureDesc(&d975)
			var d976 scm.JITValueDesc
			if d975.Loc == scm.LocImm {
				d976 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d975.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d975.Reg)
				ctx.EmitMovRegReg(scratch, d975.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d976 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d976)
			}
			if d976.Loc == scm.LocReg && d975.Loc == scm.LocReg && d976.Reg == d975.Reg {
				ctx.TransferReg(d975.Reg)
				d975.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d975)
			ctx.EnsureDesc(&d976)
			r229 := ctx.AllocReg()
			ctx.EnsureDesc(&d976)
			ctx.EnsureDesc(&d959)
			if d976.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r229, uint64(d976.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r229, d976.Reg)
				ctx.EmitShlRegImm8(r229, 3)
			}
			if d959.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d959.Imm.Int()))
				ctx.EmitAddInt64(r229, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r229, d959.Reg)
			}
			r230 := ctx.AllocRegExcept(r229)
			ctx.EmitMovRegMem(r230, r229, 0)
			ctx.FreeReg(r229)
			d977 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r230}
			ctx.BindReg(r230, &d977)
			ctx.FreeDesc(&d976)
			ctx.EnsureDesc(&d958)
			var d978 scm.JITValueDesc
			if d958.Loc == scm.LocImm {
				d978 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d958.Imm.Int() % 64)}
			} else {
				r231 := ctx.AllocRegExcept(d958.Reg)
				ctx.EmitMovRegReg(r231, d958.Reg)
				ctx.EmitAndRegImm32(r231, 63)
				d978 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r231}
				ctx.BindReg(r231, &d978)
			}
			if d978.Loc == scm.LocReg && d958.Loc == scm.LocReg && d978.Reg == d958.Reg {
				ctx.TransferReg(d958.Reg)
				d958.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d958)
			d979 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d978)
			ctx.EnsureDesc(&d979)
			ctx.ProtectReg(d979.Reg)
			ctx.EnsureDesc(&d978)
			ctx.UnprotectReg(d979.Reg)
			var d980 scm.JITValueDesc
			if d979.Loc == scm.LocImm && d978.Loc == scm.LocImm {
				d980 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d979.Imm.Int() - d978.Imm.Int())}
			} else if d978.Loc == scm.LocImm && d978.Imm.Int() == 0 {
				r232 := ctx.AllocRegExcept(d979.Reg)
				ctx.EmitMovRegReg(r232, d979.Reg)
				d980 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r232}
				ctx.BindReg(r232, &d980)
			} else if d979.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d978.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d979.Imm.Int()))
				ctx.EmitSubInt64(scratch, d978.Reg)
				d980 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d980)
			} else if d978.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d979.Reg)
				ctx.EmitMovRegReg(scratch, d979.Reg)
				if d978.Imm.Int() >= -2147483648 && d978.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d978.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d978.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d980 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d980)
			} else {
				r233 := ctx.AllocRegExcept(d979.Reg, d978.Reg)
				ctx.EmitMovRegReg(r233, d979.Reg)
				ctx.EmitSubInt64(r233, d978.Reg)
				d980 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r233}
				ctx.BindReg(r233, &d980)
			}
			if d980.Loc == scm.LocReg && d979.Loc == scm.LocReg && d980.Reg == d979.Reg {
				ctx.TransferReg(d979.Reg)
				d979.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d978)
			ctx.EnsureDesc(&d977)
			ctx.EnsureDesc(&d980)
			var d981 scm.JITValueDesc
			if d977.Loc == scm.LocImm && d980.Loc == scm.LocImm {
				d981 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d977.Imm.Int()) >> uint64(d980.Imm.Int())))}
			} else if d980.Loc == scm.LocImm {
				r234 := ctx.AllocRegExcept(d977.Reg)
				ctx.EmitMovRegReg(r234, d977.Reg)
				ctx.EmitShrRegImm8(r234, uint8(d980.Imm.Int()))
				d981 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
				ctx.BindReg(r234, &d981)
			} else {
				{
					shiftSrc := d977.Reg
					r235 := ctx.AllocRegExcept(d977.Reg)
					ctx.EmitMovRegReg(r235, d977.Reg)
					shiftSrc = r235
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d980.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d980.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d980.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d981 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d981)
				}
			}
			if d981.Loc == scm.LocReg && d977.Loc == scm.LocReg && d981.Reg == d977.Reg {
				ctx.TransferReg(d977.Reg)
				d977.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d977)
			ctx.FreeDesc(&d980)
			ctx.EnsureDesc(&d963)
			ctx.EnsureDesc(&d981)
			var d982 scm.JITValueDesc
			if d963.Loc == scm.LocImm && d981.Loc == scm.LocImm {
				d982 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d963.Imm.Int() | d981.Imm.Int())}
			} else if d963.Loc == scm.LocImm && d963.Imm.Int() == 0 {
				d982 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d981.Reg}
				ctx.BindReg(d981.Reg, &d982)
			} else if d981.Loc == scm.LocImm && d981.Imm.Int() == 0 {
				r236 := ctx.AllocRegExcept(d963.Reg)
				ctx.EmitMovRegReg(r236, d963.Reg)
				d982 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r236}
				ctx.BindReg(r236, &d982)
			} else if d963.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d981.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d963.Imm.Int()))
				ctx.EmitOrInt64(scratch, d981.Reg)
				d982 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d982)
			} else if d981.Loc == scm.LocImm {
				r237 := ctx.AllocRegExcept(d963.Reg)
				ctx.EmitMovRegReg(r237, d963.Reg)
				if d981.Imm.Int() >= -2147483648 && d981.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r237, int32(d981.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d981.Imm.Int()))
					ctx.EmitOrInt64(r237, scm.RegR11)
				}
				d982 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r237}
				ctx.BindReg(r237, &d982)
			} else {
				r238 := ctx.AllocRegExcept(d963.Reg, d981.Reg)
				ctx.EmitMovRegReg(r238, d963.Reg)
				ctx.EmitOrInt64(r238, d981.Reg)
				d982 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r238}
				ctx.BindReg(r238, &d982)
			}
			if d982.Loc == scm.LocReg && d963.Loc == scm.LocReg && d982.Reg == d963.Reg {
				ctx.TransferReg(d963.Reg)
				d963.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d981)
			ctx.EnsureDesc(&d982)
			if d982.Loc == scm.LocReg {
				ctx.ProtectReg(d982.Reg)
			} else if d982.Loc == scm.LocRegPair {
				ctx.ProtectReg(d982.Reg)
				ctx.ProtectReg(d982.Reg2)
			}
			d983 = d982
			if d983.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d983)
			ctx.EmitStoreToStack(d983, int32(bbs[2].PhiBase)+int32(0))
			if d982.Loc == scm.LocReg {
				ctx.UnprotectReg(d982.Reg)
			} else if d982.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d982.Reg)
				ctx.UnprotectReg(d982.Reg2)
			}
			ctx.EmitJmp(lbl47)
			ctx.MarkLabel(lbl45)
			d984 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r227}
			ctx.BindReg(r227, &d984)
			ctx.BindReg(r227, &d984)
			if r200 { ctx.UnprotectReg(r201) }
			ctx.FreeDesc(&d4)
			ctx.EnsureDesc(&d984)
			ctx.EnsureDesc(&d984)
			var d985 scm.JITValueDesc
			if d984.Loc == scm.LocImm {
				d985 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d984.Imm.Int()))))}
			} else {
				r239 := ctx.AllocReg()
				ctx.EmitMovRegReg(r239, d984.Reg)
				d985 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r239}
				ctx.BindReg(r239, &d985)
			}
			ctx.FreeDesc(&d984)
			ctx.EnsureDesc(&d985)
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d985)
			ctx.ProtectReg(d985.Reg)
			ctx.EnsureDesc(&d59)
			ctx.UnprotectReg(d985.Reg)
			var d986 scm.JITValueDesc
			if d985.Loc == scm.LocImm && d59.Loc == scm.LocImm {
				d986 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d985.Imm.Int() + d59.Imm.Int())}
			} else if d59.Loc == scm.LocImm && d59.Imm.Int() == 0 {
				r240 := ctx.AllocRegExcept(d985.Reg)
				ctx.EmitMovRegReg(r240, d985.Reg)
				d986 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r240}
				ctx.BindReg(r240, &d986)
			} else if d985.Loc == scm.LocImm && d985.Imm.Int() == 0 {
				d986 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d59.Reg}
				ctx.BindReg(d59.Reg, &d986)
			} else if d985.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d59.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d985.Imm.Int()))
				ctx.EmitAddInt64(scratch, d59.Reg)
				d986 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d986)
			} else if d59.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d985.Reg)
				ctx.EmitMovRegReg(scratch, d985.Reg)
				if d59.Imm.Int() >= -2147483648 && d59.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d59.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d59.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d986 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d986)
			} else {
				r241 := ctx.AllocRegExcept(d985.Reg, d59.Reg)
				ctx.EmitMovRegReg(r241, d985.Reg)
				ctx.EmitAddInt64(r241, d59.Reg)
				d986 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r241}
				ctx.BindReg(r241, &d986)
			}
			if d986.Loc == scm.LocReg && d985.Loc == scm.LocReg && d986.Reg == d985.Reg {
				ctx.TransferReg(d985.Reg)
				d985.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d985)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&idxInt)
			var d987 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d987 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(idxInt.Imm.Int()))))}
			} else {
				r242 := ctx.AllocReg()
				ctx.EmitMovRegReg(r242, idxInt.Reg)
				ctx.EmitShlRegImm8(r242, 32)
				ctx.EmitShrRegImm8(r242, 32)
				d987 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r242}
				ctx.BindReg(r242, &d987)
			}
			ctx.FreeDesc(&idxInt)
			ctx.EnsureDesc(&d987)
			ctx.EnsureDesc(&d986)
			ctx.EnsureDesc(&d987)
			ctx.ProtectReg(d987.Reg)
			ctx.EnsureDesc(&d986)
			ctx.UnprotectReg(d987.Reg)
			var d988 scm.JITValueDesc
			if d987.Loc == scm.LocImm && d986.Loc == scm.LocImm {
				d988 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d987.Imm.Int() - d986.Imm.Int())}
			} else if d986.Loc == scm.LocImm && d986.Imm.Int() == 0 {
				r243 := ctx.AllocRegExcept(d987.Reg)
				ctx.EmitMovRegReg(r243, d987.Reg)
				d988 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r243}
				ctx.BindReg(r243, &d988)
			} else if d987.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d986.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d987.Imm.Int()))
				ctx.EmitSubInt64(scratch, d986.Reg)
				d988 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d988)
			} else if d986.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d987.Reg)
				ctx.EmitMovRegReg(scratch, d987.Reg)
				if d986.Imm.Int() >= -2147483648 && d986.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d986.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d986.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d988 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d988)
			} else {
				r244 := ctx.AllocRegExcept(d987.Reg, d986.Reg)
				ctx.EmitMovRegReg(r244, d987.Reg)
				ctx.EmitSubInt64(r244, d986.Reg)
				d988 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r244}
				ctx.BindReg(r244, &d988)
			}
			if d988.Loc == scm.LocReg && d987.Loc == scm.LocReg && d988.Reg == d987.Reg {
				ctx.TransferReg(d987.Reg)
				d987.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d987)
			ctx.FreeDesc(&d986)
			ctx.EnsureDesc(&d988)
			ctx.EnsureDesc(&d951)
			ctx.EnsureDesc(&d988)
			ctx.ProtectReg(d988.Reg)
			ctx.EnsureDesc(&d951)
			ctx.UnprotectReg(d988.Reg)
			var d989 scm.JITValueDesc
			if d988.Loc == scm.LocImm && d951.Loc == scm.LocImm {
				d989 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d988.Imm.Int() * d951.Imm.Int())}
			} else if d988.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d951.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d988.Imm.Int()))
				ctx.EmitImulInt64(scratch, d951.Reg)
				d989 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d989)
			} else if d951.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d988.Reg)
				ctx.EmitMovRegReg(scratch, d988.Reg)
				if d951.Imm.Int() >= -2147483648 && d951.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d951.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d951.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d989 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d989)
			} else {
				r245 := ctx.AllocRegExcept(d988.Reg, d951.Reg)
				ctx.EmitMovRegReg(r245, d988.Reg)
				ctx.EmitImulInt64(r245, d951.Reg)
				d989 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r245}
				ctx.BindReg(r245, &d989)
			}
			if d989.Loc == scm.LocReg && d988.Loc == scm.LocReg && d989.Reg == d988.Reg {
				ctx.TransferReg(d988.Reg)
				d988.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d988)
			ctx.FreeDesc(&d951)
			ctx.EnsureDesc(&d173)
			ctx.EnsureDesc(&d989)
			ctx.EnsureDesc(&d173)
			ctx.ProtectReg(d173.Reg)
			ctx.EnsureDesc(&d989)
			ctx.UnprotectReg(d173.Reg)
			var d990 scm.JITValueDesc
			if d173.Loc == scm.LocImm && d989.Loc == scm.LocImm {
				d990 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d173.Imm.Int() + d989.Imm.Int())}
			} else if d989.Loc == scm.LocImm && d989.Imm.Int() == 0 {
				r246 := ctx.AllocRegExcept(d173.Reg)
				ctx.EmitMovRegReg(r246, d173.Reg)
				d990 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r246}
				ctx.BindReg(r246, &d990)
			} else if d173.Loc == scm.LocImm && d173.Imm.Int() == 0 {
				d990 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d989.Reg}
				ctx.BindReg(d989.Reg, &d990)
			} else if d173.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d989.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d173.Imm.Int()))
				ctx.EmitAddInt64(scratch, d989.Reg)
				d990 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d990)
			} else if d989.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d173.Reg)
				ctx.EmitMovRegReg(scratch, d173.Reg)
				if d989.Imm.Int() >= -2147483648 && d989.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d989.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d989.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d990 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d990)
			} else {
				r247 := ctx.AllocRegExcept(d173.Reg, d989.Reg)
				ctx.EmitMovRegReg(r247, d173.Reg)
				ctx.EmitAddInt64(r247, d989.Reg)
				d990 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r247}
				ctx.BindReg(r247, &d990)
			}
			if d990.Loc == scm.LocReg && d173.Loc == scm.LocReg && d990.Reg == d173.Reg {
				ctx.TransferReg(d173.Reg)
				d173.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d989)
			ctx.EnsureDesc(&d990)
			ctx.EnsureDesc(&d990)
			var d991 scm.JITValueDesc
			if d990.Loc == scm.LocImm {
				d991 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d990.Imm.Int()))}
			} else {
				ctx.EmitCvtInt64ToFloat64(scm.RegX0, d990.Reg)
				d991 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d990.Reg}
				ctx.BindReg(d990.Reg, &d991)
			}
			ctx.FreeDesc(&d990)
			ctx.EnsureDesc(&d991)
			d992 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d992)
			ctx.BindReg(r1, &d992)
			ctx.EnsureDesc(&d991)
			ctx.EmitMakeFloat(d992, d991)
			if d991.Loc == scm.LocReg { ctx.FreeReg(d991.Reg) }
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
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
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
			if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != scm.LocNone {
				d173 = ps.OverlayValues[173]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
			}
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
				d178 = ps.OverlayValues[178]
			}
			if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != scm.LocNone {
				d286 = ps.OverlayValues[286]
			}
			if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != scm.LocNone {
				d287 = ps.OverlayValues[287]
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
			if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != scm.LocNone {
				d293 = ps.OverlayValues[293]
			}
			if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
				d295 = ps.OverlayValues[295]
			}
			if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != scm.LocNone {
				d296 = ps.OverlayValues[296]
			}
			if len(ps.OverlayValues) > 297 && ps.OverlayValues[297].Loc != scm.LocNone {
				d297 = ps.OverlayValues[297]
			}
			if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != scm.LocNone {
				d298 = ps.OverlayValues[298]
			}
			if len(ps.OverlayValues) > 299 && ps.OverlayValues[299].Loc != scm.LocNone {
				d299 = ps.OverlayValues[299]
			}
			if len(ps.OverlayValues) > 300 && ps.OverlayValues[300].Loc != scm.LocNone {
				d300 = ps.OverlayValues[300]
			}
			if len(ps.OverlayValues) > 301 && ps.OverlayValues[301].Loc != scm.LocNone {
				d301 = ps.OverlayValues[301]
			}
			if len(ps.OverlayValues) > 302 && ps.OverlayValues[302].Loc != scm.LocNone {
				d302 = ps.OverlayValues[302]
			}
			if len(ps.OverlayValues) > 303 && ps.OverlayValues[303].Loc != scm.LocNone {
				d303 = ps.OverlayValues[303]
			}
			if len(ps.OverlayValues) > 304 && ps.OverlayValues[304].Loc != scm.LocNone {
				d304 = ps.OverlayValues[304]
			}
			if len(ps.OverlayValues) > 306 && ps.OverlayValues[306].Loc != scm.LocNone {
				d306 = ps.OverlayValues[306]
			}
			if len(ps.OverlayValues) > 308 && ps.OverlayValues[308].Loc != scm.LocNone {
				d308 = ps.OverlayValues[308]
			}
			if len(ps.OverlayValues) > 309 && ps.OverlayValues[309].Loc != scm.LocNone {
				d309 = ps.OverlayValues[309]
			}
			if len(ps.OverlayValues) > 310 && ps.OverlayValues[310].Loc != scm.LocNone {
				d310 = ps.OverlayValues[310]
			}
			if len(ps.OverlayValues) > 311 && ps.OverlayValues[311].Loc != scm.LocNone {
				d311 = ps.OverlayValues[311]
			}
			if len(ps.OverlayValues) > 312 && ps.OverlayValues[312].Loc != scm.LocNone {
				d312 = ps.OverlayValues[312]
			}
			if len(ps.OverlayValues) > 315 && ps.OverlayValues[315].Loc != scm.LocNone {
				d315 = ps.OverlayValues[315]
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
			if len(ps.OverlayValues) > 459 && ps.OverlayValues[459].Loc != scm.LocNone {
				d459 = ps.OverlayValues[459]
			}
			if len(ps.OverlayValues) > 460 && ps.OverlayValues[460].Loc != scm.LocNone {
				d460 = ps.OverlayValues[460]
			}
			if len(ps.OverlayValues) > 461 && ps.OverlayValues[461].Loc != scm.LocNone {
				d461 = ps.OverlayValues[461]
			}
			if len(ps.OverlayValues) > 462 && ps.OverlayValues[462].Loc != scm.LocNone {
				d462 = ps.OverlayValues[462]
			}
			if len(ps.OverlayValues) > 463 && ps.OverlayValues[463].Loc != scm.LocNone {
				d463 = ps.OverlayValues[463]
			}
			if len(ps.OverlayValues) > 464 && ps.OverlayValues[464].Loc != scm.LocNone {
				d464 = ps.OverlayValues[464]
			}
			if len(ps.OverlayValues) > 465 && ps.OverlayValues[465].Loc != scm.LocNone {
				d465 = ps.OverlayValues[465]
			}
			if len(ps.OverlayValues) > 466 && ps.OverlayValues[466].Loc != scm.LocNone {
				d466 = ps.OverlayValues[466]
			}
			if len(ps.OverlayValues) > 467 && ps.OverlayValues[467].Loc != scm.LocNone {
				d467 = ps.OverlayValues[467]
			}
			if len(ps.OverlayValues) > 468 && ps.OverlayValues[468].Loc != scm.LocNone {
				d468 = ps.OverlayValues[468]
			}
			if len(ps.OverlayValues) > 469 && ps.OverlayValues[469].Loc != scm.LocNone {
				d469 = ps.OverlayValues[469]
			}
			if len(ps.OverlayValues) > 470 && ps.OverlayValues[470].Loc != scm.LocNone {
				d470 = ps.OverlayValues[470]
			}
			if len(ps.OverlayValues) > 471 && ps.OverlayValues[471].Loc != scm.LocNone {
				d471 = ps.OverlayValues[471]
			}
			if len(ps.OverlayValues) > 472 && ps.OverlayValues[472].Loc != scm.LocNone {
				d472 = ps.OverlayValues[472]
			}
			if len(ps.OverlayValues) > 473 && ps.OverlayValues[473].Loc != scm.LocNone {
				d473 = ps.OverlayValues[473]
			}
			if len(ps.OverlayValues) > 474 && ps.OverlayValues[474].Loc != scm.LocNone {
				d474 = ps.OverlayValues[474]
			}
			if len(ps.OverlayValues) > 475 && ps.OverlayValues[475].Loc != scm.LocNone {
				d475 = ps.OverlayValues[475]
			}
			if len(ps.OverlayValues) > 476 && ps.OverlayValues[476].Loc != scm.LocNone {
				d476 = ps.OverlayValues[476]
			}
			if len(ps.OverlayValues) > 477 && ps.OverlayValues[477].Loc != scm.LocNone {
				d477 = ps.OverlayValues[477]
			}
			if len(ps.OverlayValues) > 478 && ps.OverlayValues[478].Loc != scm.LocNone {
				d478 = ps.OverlayValues[478]
			}
			if len(ps.OverlayValues) > 479 && ps.OverlayValues[479].Loc != scm.LocNone {
				d479 = ps.OverlayValues[479]
			}
			if len(ps.OverlayValues) > 480 && ps.OverlayValues[480].Loc != scm.LocNone {
				d480 = ps.OverlayValues[480]
			}
			if len(ps.OverlayValues) > 481 && ps.OverlayValues[481].Loc != scm.LocNone {
				d481 = ps.OverlayValues[481]
			}
			if len(ps.OverlayValues) > 482 && ps.OverlayValues[482].Loc != scm.LocNone {
				d482 = ps.OverlayValues[482]
			}
			if len(ps.OverlayValues) > 483 && ps.OverlayValues[483].Loc != scm.LocNone {
				d483 = ps.OverlayValues[483]
			}
			if len(ps.OverlayValues) > 484 && ps.OverlayValues[484].Loc != scm.LocNone {
				d484 = ps.OverlayValues[484]
			}
			if len(ps.OverlayValues) > 485 && ps.OverlayValues[485].Loc != scm.LocNone {
				d485 = ps.OverlayValues[485]
			}
			if len(ps.OverlayValues) > 486 && ps.OverlayValues[486].Loc != scm.LocNone {
				d486 = ps.OverlayValues[486]
			}
			if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != scm.LocNone {
				d487 = ps.OverlayValues[487]
			}
			if len(ps.OverlayValues) > 488 && ps.OverlayValues[488].Loc != scm.LocNone {
				d488 = ps.OverlayValues[488]
			}
			if len(ps.OverlayValues) > 489 && ps.OverlayValues[489].Loc != scm.LocNone {
				d489 = ps.OverlayValues[489]
			}
			if len(ps.OverlayValues) > 490 && ps.OverlayValues[490].Loc != scm.LocNone {
				d490 = ps.OverlayValues[490]
			}
			if len(ps.OverlayValues) > 491 && ps.OverlayValues[491].Loc != scm.LocNone {
				d491 = ps.OverlayValues[491]
			}
			if len(ps.OverlayValues) > 492 && ps.OverlayValues[492].Loc != scm.LocNone {
				d492 = ps.OverlayValues[492]
			}
			if len(ps.OverlayValues) > 493 && ps.OverlayValues[493].Loc != scm.LocNone {
				d493 = ps.OverlayValues[493]
			}
			if len(ps.OverlayValues) > 494 && ps.OverlayValues[494].Loc != scm.LocNone {
				d494 = ps.OverlayValues[494]
			}
			if len(ps.OverlayValues) > 676 && ps.OverlayValues[676].Loc != scm.LocNone {
				d676 = ps.OverlayValues[676]
			}
			if len(ps.OverlayValues) > 677 && ps.OverlayValues[677].Loc != scm.LocNone {
				d677 = ps.OverlayValues[677]
			}
			if len(ps.OverlayValues) > 678 && ps.OverlayValues[678].Loc != scm.LocNone {
				d678 = ps.OverlayValues[678]
			}
			if len(ps.OverlayValues) > 679 && ps.OverlayValues[679].Loc != scm.LocNone {
				d679 = ps.OverlayValues[679]
			}
			if len(ps.OverlayValues) > 680 && ps.OverlayValues[680].Loc != scm.LocNone {
				d680 = ps.OverlayValues[680]
			}
			if len(ps.OverlayValues) > 682 && ps.OverlayValues[682].Loc != scm.LocNone {
				d682 = ps.OverlayValues[682]
			}
			if len(ps.OverlayValues) > 683 && ps.OverlayValues[683].Loc != scm.LocNone {
				d683 = ps.OverlayValues[683]
			}
			if len(ps.OverlayValues) > 684 && ps.OverlayValues[684].Loc != scm.LocNone {
				d684 = ps.OverlayValues[684]
			}
			if len(ps.OverlayValues) > 685 && ps.OverlayValues[685].Loc != scm.LocNone {
				d685 = ps.OverlayValues[685]
			}
			if len(ps.OverlayValues) > 686 && ps.OverlayValues[686].Loc != scm.LocNone {
				d686 = ps.OverlayValues[686]
			}
			if len(ps.OverlayValues) > 687 && ps.OverlayValues[687].Loc != scm.LocNone {
				d687 = ps.OverlayValues[687]
			}
			if len(ps.OverlayValues) > 688 && ps.OverlayValues[688].Loc != scm.LocNone {
				d688 = ps.OverlayValues[688]
			}
			if len(ps.OverlayValues) > 689 && ps.OverlayValues[689].Loc != scm.LocNone {
				d689 = ps.OverlayValues[689]
			}
			if len(ps.OverlayValues) > 691 && ps.OverlayValues[691].Loc != scm.LocNone {
				d691 = ps.OverlayValues[691]
			}
			if len(ps.OverlayValues) > 693 && ps.OverlayValues[693].Loc != scm.LocNone {
				d693 = ps.OverlayValues[693]
			}
			if len(ps.OverlayValues) > 694 && ps.OverlayValues[694].Loc != scm.LocNone {
				d694 = ps.OverlayValues[694]
			}
			if len(ps.OverlayValues) > 695 && ps.OverlayValues[695].Loc != scm.LocNone {
				d695 = ps.OverlayValues[695]
			}
			if len(ps.OverlayValues) > 696 && ps.OverlayValues[696].Loc != scm.LocNone {
				d696 = ps.OverlayValues[696]
			}
			if len(ps.OverlayValues) > 699 && ps.OverlayValues[699].Loc != scm.LocNone {
				d699 = ps.OverlayValues[699]
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
			if len(ps.OverlayValues) > 918 && ps.OverlayValues[918].Loc != scm.LocNone {
				d918 = ps.OverlayValues[918]
			}
			if len(ps.OverlayValues) > 919 && ps.OverlayValues[919].Loc != scm.LocNone {
				d919 = ps.OverlayValues[919]
			}
			if len(ps.OverlayValues) > 920 && ps.OverlayValues[920].Loc != scm.LocNone {
				d920 = ps.OverlayValues[920]
			}
			if len(ps.OverlayValues) > 921 && ps.OverlayValues[921].Loc != scm.LocNone {
				d921 = ps.OverlayValues[921]
			}
			if len(ps.OverlayValues) > 922 && ps.OverlayValues[922].Loc != scm.LocNone {
				d922 = ps.OverlayValues[922]
			}
			if len(ps.OverlayValues) > 923 && ps.OverlayValues[923].Loc != scm.LocNone {
				d923 = ps.OverlayValues[923]
			}
			if len(ps.OverlayValues) > 924 && ps.OverlayValues[924].Loc != scm.LocNone {
				d924 = ps.OverlayValues[924]
			}
			if len(ps.OverlayValues) > 925 && ps.OverlayValues[925].Loc != scm.LocNone {
				d925 = ps.OverlayValues[925]
			}
			if len(ps.OverlayValues) > 926 && ps.OverlayValues[926].Loc != scm.LocNone {
				d926 = ps.OverlayValues[926]
			}
			if len(ps.OverlayValues) > 927 && ps.OverlayValues[927].Loc != scm.LocNone {
				d927 = ps.OverlayValues[927]
			}
			if len(ps.OverlayValues) > 928 && ps.OverlayValues[928].Loc != scm.LocNone {
				d928 = ps.OverlayValues[928]
			}
			if len(ps.OverlayValues) > 929 && ps.OverlayValues[929].Loc != scm.LocNone {
				d929 = ps.OverlayValues[929]
			}
			if len(ps.OverlayValues) > 930 && ps.OverlayValues[930].Loc != scm.LocNone {
				d930 = ps.OverlayValues[930]
			}
			if len(ps.OverlayValues) > 931 && ps.OverlayValues[931].Loc != scm.LocNone {
				d931 = ps.OverlayValues[931]
			}
			if len(ps.OverlayValues) > 932 && ps.OverlayValues[932].Loc != scm.LocNone {
				d932 = ps.OverlayValues[932]
			}
			if len(ps.OverlayValues) > 933 && ps.OverlayValues[933].Loc != scm.LocNone {
				d933 = ps.OverlayValues[933]
			}
			if len(ps.OverlayValues) > 934 && ps.OverlayValues[934].Loc != scm.LocNone {
				d934 = ps.OverlayValues[934]
			}
			if len(ps.OverlayValues) > 935 && ps.OverlayValues[935].Loc != scm.LocNone {
				d935 = ps.OverlayValues[935]
			}
			if len(ps.OverlayValues) > 936 && ps.OverlayValues[936].Loc != scm.LocNone {
				d936 = ps.OverlayValues[936]
			}
			if len(ps.OverlayValues) > 937 && ps.OverlayValues[937].Loc != scm.LocNone {
				d937 = ps.OverlayValues[937]
			}
			if len(ps.OverlayValues) > 938 && ps.OverlayValues[938].Loc != scm.LocNone {
				d938 = ps.OverlayValues[938]
			}
			if len(ps.OverlayValues) > 939 && ps.OverlayValues[939].Loc != scm.LocNone {
				d939 = ps.OverlayValues[939]
			}
			if len(ps.OverlayValues) > 940 && ps.OverlayValues[940].Loc != scm.LocNone {
				d940 = ps.OverlayValues[940]
			}
			if len(ps.OverlayValues) > 941 && ps.OverlayValues[941].Loc != scm.LocNone {
				d941 = ps.OverlayValues[941]
			}
			if len(ps.OverlayValues) > 942 && ps.OverlayValues[942].Loc != scm.LocNone {
				d942 = ps.OverlayValues[942]
			}
			if len(ps.OverlayValues) > 943 && ps.OverlayValues[943].Loc != scm.LocNone {
				d943 = ps.OverlayValues[943]
			}
			if len(ps.OverlayValues) > 944 && ps.OverlayValues[944].Loc != scm.LocNone {
				d944 = ps.OverlayValues[944]
			}
			if len(ps.OverlayValues) > 945 && ps.OverlayValues[945].Loc != scm.LocNone {
				d945 = ps.OverlayValues[945]
			}
			if len(ps.OverlayValues) > 946 && ps.OverlayValues[946].Loc != scm.LocNone {
				d946 = ps.OverlayValues[946]
			}
			if len(ps.OverlayValues) > 947 && ps.OverlayValues[947].Loc != scm.LocNone {
				d947 = ps.OverlayValues[947]
			}
			if len(ps.OverlayValues) > 948 && ps.OverlayValues[948].Loc != scm.LocNone {
				d948 = ps.OverlayValues[948]
			}
			if len(ps.OverlayValues) > 949 && ps.OverlayValues[949].Loc != scm.LocNone {
				d949 = ps.OverlayValues[949]
			}
			if len(ps.OverlayValues) > 950 && ps.OverlayValues[950].Loc != scm.LocNone {
				d950 = ps.OverlayValues[950]
			}
			if len(ps.OverlayValues) > 951 && ps.OverlayValues[951].Loc != scm.LocNone {
				d951 = ps.OverlayValues[951]
			}
			if len(ps.OverlayValues) > 952 && ps.OverlayValues[952].Loc != scm.LocNone {
				d952 = ps.OverlayValues[952]
			}
			if len(ps.OverlayValues) > 954 && ps.OverlayValues[954].Loc != scm.LocNone {
				d954 = ps.OverlayValues[954]
			}
			if len(ps.OverlayValues) > 955 && ps.OverlayValues[955].Loc != scm.LocNone {
				d955 = ps.OverlayValues[955]
			}
			if len(ps.OverlayValues) > 956 && ps.OverlayValues[956].Loc != scm.LocNone {
				d956 = ps.OverlayValues[956]
			}
			if len(ps.OverlayValues) > 957 && ps.OverlayValues[957].Loc != scm.LocNone {
				d957 = ps.OverlayValues[957]
			}
			if len(ps.OverlayValues) > 958 && ps.OverlayValues[958].Loc != scm.LocNone {
				d958 = ps.OverlayValues[958]
			}
			if len(ps.OverlayValues) > 959 && ps.OverlayValues[959].Loc != scm.LocNone {
				d959 = ps.OverlayValues[959]
			}
			if len(ps.OverlayValues) > 960 && ps.OverlayValues[960].Loc != scm.LocNone {
				d960 = ps.OverlayValues[960]
			}
			if len(ps.OverlayValues) > 961 && ps.OverlayValues[961].Loc != scm.LocNone {
				d961 = ps.OverlayValues[961]
			}
			if len(ps.OverlayValues) > 962 && ps.OverlayValues[962].Loc != scm.LocNone {
				d962 = ps.OverlayValues[962]
			}
			if len(ps.OverlayValues) > 963 && ps.OverlayValues[963].Loc != scm.LocNone {
				d963 = ps.OverlayValues[963]
			}
			if len(ps.OverlayValues) > 964 && ps.OverlayValues[964].Loc != scm.LocNone {
				d964 = ps.OverlayValues[964]
			}
			if len(ps.OverlayValues) > 965 && ps.OverlayValues[965].Loc != scm.LocNone {
				d965 = ps.OverlayValues[965]
			}
			if len(ps.OverlayValues) > 966 && ps.OverlayValues[966].Loc != scm.LocNone {
				d966 = ps.OverlayValues[966]
			}
			if len(ps.OverlayValues) > 967 && ps.OverlayValues[967].Loc != scm.LocNone {
				d967 = ps.OverlayValues[967]
			}
			if len(ps.OverlayValues) > 968 && ps.OverlayValues[968].Loc != scm.LocNone {
				d968 = ps.OverlayValues[968]
			}
			if len(ps.OverlayValues) > 969 && ps.OverlayValues[969].Loc != scm.LocNone {
				d969 = ps.OverlayValues[969]
			}
			if len(ps.OverlayValues) > 970 && ps.OverlayValues[970].Loc != scm.LocNone {
				d970 = ps.OverlayValues[970]
			}
			if len(ps.OverlayValues) > 971 && ps.OverlayValues[971].Loc != scm.LocNone {
				d971 = ps.OverlayValues[971]
			}
			if len(ps.OverlayValues) > 972 && ps.OverlayValues[972].Loc != scm.LocNone {
				d972 = ps.OverlayValues[972]
			}
			if len(ps.OverlayValues) > 973 && ps.OverlayValues[973].Loc != scm.LocNone {
				d973 = ps.OverlayValues[973]
			}
			if len(ps.OverlayValues) > 974 && ps.OverlayValues[974].Loc != scm.LocNone {
				d974 = ps.OverlayValues[974]
			}
			if len(ps.OverlayValues) > 975 && ps.OverlayValues[975].Loc != scm.LocNone {
				d975 = ps.OverlayValues[975]
			}
			if len(ps.OverlayValues) > 976 && ps.OverlayValues[976].Loc != scm.LocNone {
				d976 = ps.OverlayValues[976]
			}
			if len(ps.OverlayValues) > 977 && ps.OverlayValues[977].Loc != scm.LocNone {
				d977 = ps.OverlayValues[977]
			}
			if len(ps.OverlayValues) > 978 && ps.OverlayValues[978].Loc != scm.LocNone {
				d978 = ps.OverlayValues[978]
			}
			if len(ps.OverlayValues) > 979 && ps.OverlayValues[979].Loc != scm.LocNone {
				d979 = ps.OverlayValues[979]
			}
			if len(ps.OverlayValues) > 980 && ps.OverlayValues[980].Loc != scm.LocNone {
				d980 = ps.OverlayValues[980]
			}
			if len(ps.OverlayValues) > 981 && ps.OverlayValues[981].Loc != scm.LocNone {
				d981 = ps.OverlayValues[981]
			}
			if len(ps.OverlayValues) > 982 && ps.OverlayValues[982].Loc != scm.LocNone {
				d982 = ps.OverlayValues[982]
			}
			if len(ps.OverlayValues) > 983 && ps.OverlayValues[983].Loc != scm.LocNone {
				d983 = ps.OverlayValues[983]
			}
			if len(ps.OverlayValues) > 984 && ps.OverlayValues[984].Loc != scm.LocNone {
				d984 = ps.OverlayValues[984]
			}
			if len(ps.OverlayValues) > 985 && ps.OverlayValues[985].Loc != scm.LocNone {
				d985 = ps.OverlayValues[985]
			}
			if len(ps.OverlayValues) > 986 && ps.OverlayValues[986].Loc != scm.LocNone {
				d986 = ps.OverlayValues[986]
			}
			if len(ps.OverlayValues) > 987 && ps.OverlayValues[987].Loc != scm.LocNone {
				d987 = ps.OverlayValues[987]
			}
			if len(ps.OverlayValues) > 988 && ps.OverlayValues[988].Loc != scm.LocNone {
				d988 = ps.OverlayValues[988]
			}
			if len(ps.OverlayValues) > 989 && ps.OverlayValues[989].Loc != scm.LocNone {
				d989 = ps.OverlayValues[989]
			}
			if len(ps.OverlayValues) > 990 && ps.OverlayValues[990].Loc != scm.LocNone {
				d990 = ps.OverlayValues[990]
			}
			if len(ps.OverlayValues) > 991 && ps.OverlayValues[991].Loc != scm.LocNone {
				d991 = ps.OverlayValues[991]
			}
			if len(ps.OverlayValues) > 992 && ps.OverlayValues[992].Loc != scm.LocNone {
				d992 = ps.OverlayValues[992]
			}
			ctx.ReclaimUntrackedRegs()
			var d993 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 64
				r248 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r248, fieldAddr)
				d993 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r248}
				ctx.BindReg(r248, &d993)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 64)
				r249 := ctx.AllocReg()
				ctx.EmitMovRegMem(r249, thisptr.Reg, off)
				d993 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r249}
				ctx.BindReg(r249, &d993)
			}
			ctx.EnsureDesc(&d993)
			ctx.EnsureDesc(&d993)
			var d994 scm.JITValueDesc
			if d993.Loc == scm.LocImm {
				d994 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d993.Imm.Int()))))}
			} else {
				r250 := ctx.AllocReg()
				ctx.EmitMovRegReg(r250, d993.Reg)
				d994 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r250}
				ctx.BindReg(r250, &d994)
			}
			ctx.EnsureDesc(&d173)
			ctx.EnsureDesc(&d994)
			ctx.EnsureDesc(&d173)
			ctx.EnsureDesc(&d994)
			ctx.EnsureDesc(&d173)
			ctx.EnsureDesc(&d994)
			var d995 scm.JITValueDesc
			if d173.Loc == scm.LocImm && d994.Loc == scm.LocImm {
				d995 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d173.Imm.Int() == d994.Imm.Int())}
			} else if d994.Loc == scm.LocImm {
				r251 := ctx.AllocRegExcept(d173.Reg)
				if d994.Imm.Int() >= -2147483648 && d994.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d173.Reg, int32(d994.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d994.Imm.Int()))
					ctx.EmitCmpInt64(d173.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r251, scm.CcE)
				d995 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r251}
				ctx.BindReg(r251, &d995)
			} else if d173.Loc == scm.LocImm {
				r252 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d173.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d994.Reg)
				ctx.EmitSetcc(r252, scm.CcE)
				d995 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r252}
				ctx.BindReg(r252, &d995)
			} else {
				r253 := ctx.AllocRegExcept(d173.Reg)
				ctx.EmitCmpInt64(d173.Reg, d994.Reg)
				ctx.EmitSetcc(r253, scm.CcE)
				d995 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r253}
				ctx.BindReg(r253, &d995)
			}
			ctx.FreeDesc(&d173)
			ctx.FreeDesc(&d994)
			d996 = d995
			ctx.EnsureDesc(&d996)
			if d996.Loc != scm.LocImm && d996.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d996.Loc == scm.LocImm {
				if d996.Imm.Bool() {
			ps997 := scm.PhiState{General: ps.General}
			ps997.OverlayValues = make([]scm.JITValueDesc, 997)
			ps997.OverlayValues[1] = d1
			ps997.OverlayValues[2] = d2
			ps997.OverlayValues[3] = d3
			ps997.OverlayValues[4] = d4
			ps997.OverlayValues[5] = d5
			ps997.OverlayValues[6] = d6
			ps997.OverlayValues[7] = d7
			ps997.OverlayValues[8] = d8
			ps997.OverlayValues[9] = d9
			ps997.OverlayValues[10] = d10
			ps997.OverlayValues[11] = d11
			ps997.OverlayValues[12] = d12
			ps997.OverlayValues[13] = d13
			ps997.OverlayValues[14] = d14
			ps997.OverlayValues[15] = d15
			ps997.OverlayValues[16] = d16
			ps997.OverlayValues[17] = d17
			ps997.OverlayValues[19] = d19
			ps997.OverlayValues[20] = d20
			ps997.OverlayValues[21] = d21
			ps997.OverlayValues[22] = d22
			ps997.OverlayValues[23] = d23
			ps997.OverlayValues[24] = d24
			ps997.OverlayValues[25] = d25
			ps997.OverlayValues[27] = d27
			ps997.OverlayValues[28] = d28
			ps997.OverlayValues[29] = d29
			ps997.OverlayValues[30] = d30
			ps997.OverlayValues[31] = d31
			ps997.OverlayValues[32] = d32
			ps997.OverlayValues[33] = d33
			ps997.OverlayValues[34] = d34
			ps997.OverlayValues[35] = d35
			ps997.OverlayValues[36] = d36
			ps997.OverlayValues[37] = d37
			ps997.OverlayValues[38] = d38
			ps997.OverlayValues[39] = d39
			ps997.OverlayValues[40] = d40
			ps997.OverlayValues[41] = d41
			ps997.OverlayValues[42] = d42
			ps997.OverlayValues[43] = d43
			ps997.OverlayValues[44] = d44
			ps997.OverlayValues[45] = d45
			ps997.OverlayValues[46] = d46
			ps997.OverlayValues[47] = d47
			ps997.OverlayValues[48] = d48
			ps997.OverlayValues[49] = d49
			ps997.OverlayValues[50] = d50
			ps997.OverlayValues[51] = d51
			ps997.OverlayValues[52] = d52
			ps997.OverlayValues[53] = d53
			ps997.OverlayValues[54] = d54
			ps997.OverlayValues[55] = d55
			ps997.OverlayValues[56] = d56
			ps997.OverlayValues[57] = d57
			ps997.OverlayValues[58] = d58
			ps997.OverlayValues[59] = d59
			ps997.OverlayValues[60] = d60
			ps997.OverlayValues[61] = d61
			ps997.OverlayValues[62] = d62
			ps997.OverlayValues[63] = d63
			ps997.OverlayValues[66] = d66
			ps997.OverlayValues[67] = d67
			ps997.OverlayValues[68] = d68
			ps997.OverlayValues[136] = d136
			ps997.OverlayValues[137] = d137
			ps997.OverlayValues[138] = d138
			ps997.OverlayValues[140] = d140
			ps997.OverlayValues[141] = d141
			ps997.OverlayValues[142] = d142
			ps997.OverlayValues[143] = d143
			ps997.OverlayValues[144] = d144
			ps997.OverlayValues[145] = d145
			ps997.OverlayValues[146] = d146
			ps997.OverlayValues[147] = d147
			ps997.OverlayValues[148] = d148
			ps997.OverlayValues[149] = d149
			ps997.OverlayValues[150] = d150
			ps997.OverlayValues[151] = d151
			ps997.OverlayValues[152] = d152
			ps997.OverlayValues[153] = d153
			ps997.OverlayValues[154] = d154
			ps997.OverlayValues[155] = d155
			ps997.OverlayValues[156] = d156
			ps997.OverlayValues[157] = d157
			ps997.OverlayValues[158] = d158
			ps997.OverlayValues[159] = d159
			ps997.OverlayValues[160] = d160
			ps997.OverlayValues[161] = d161
			ps997.OverlayValues[162] = d162
			ps997.OverlayValues[163] = d163
			ps997.OverlayValues[164] = d164
			ps997.OverlayValues[165] = d165
			ps997.OverlayValues[166] = d166
			ps997.OverlayValues[167] = d167
			ps997.OverlayValues[168] = d168
			ps997.OverlayValues[169] = d169
			ps997.OverlayValues[170] = d170
			ps997.OverlayValues[171] = d171
			ps997.OverlayValues[172] = d172
			ps997.OverlayValues[173] = d173
			ps997.OverlayValues[174] = d174
			ps997.OverlayValues[175] = d175
			ps997.OverlayValues[178] = d178
			ps997.OverlayValues[286] = d286
			ps997.OverlayValues[287] = d287
			ps997.OverlayValues[288] = d288
			ps997.OverlayValues[289] = d289
			ps997.OverlayValues[290] = d290
			ps997.OverlayValues[291] = d291
			ps997.OverlayValues[292] = d292
			ps997.OverlayValues[293] = d293
			ps997.OverlayValues[295] = d295
			ps997.OverlayValues[296] = d296
			ps997.OverlayValues[297] = d297
			ps997.OverlayValues[298] = d298
			ps997.OverlayValues[299] = d299
			ps997.OverlayValues[300] = d300
			ps997.OverlayValues[301] = d301
			ps997.OverlayValues[302] = d302
			ps997.OverlayValues[303] = d303
			ps997.OverlayValues[304] = d304
			ps997.OverlayValues[306] = d306
			ps997.OverlayValues[308] = d308
			ps997.OverlayValues[309] = d309
			ps997.OverlayValues[310] = d310
			ps997.OverlayValues[311] = d311
			ps997.OverlayValues[312] = d312
			ps997.OverlayValues[315] = d315
			ps997.OverlayValues[446] = d446
			ps997.OverlayValues[447] = d447
			ps997.OverlayValues[448] = d448
			ps997.OverlayValues[449] = d449
			ps997.OverlayValues[450] = d450
			ps997.OverlayValues[451] = d451
			ps997.OverlayValues[452] = d452
			ps997.OverlayValues[454] = d454
			ps997.OverlayValues[455] = d455
			ps997.OverlayValues[456] = d456
			ps997.OverlayValues[457] = d457
			ps997.OverlayValues[459] = d459
			ps997.OverlayValues[460] = d460
			ps997.OverlayValues[461] = d461
			ps997.OverlayValues[462] = d462
			ps997.OverlayValues[463] = d463
			ps997.OverlayValues[464] = d464
			ps997.OverlayValues[465] = d465
			ps997.OverlayValues[466] = d466
			ps997.OverlayValues[467] = d467
			ps997.OverlayValues[468] = d468
			ps997.OverlayValues[469] = d469
			ps997.OverlayValues[470] = d470
			ps997.OverlayValues[471] = d471
			ps997.OverlayValues[472] = d472
			ps997.OverlayValues[473] = d473
			ps997.OverlayValues[474] = d474
			ps997.OverlayValues[475] = d475
			ps997.OverlayValues[476] = d476
			ps997.OverlayValues[477] = d477
			ps997.OverlayValues[478] = d478
			ps997.OverlayValues[479] = d479
			ps997.OverlayValues[480] = d480
			ps997.OverlayValues[481] = d481
			ps997.OverlayValues[482] = d482
			ps997.OverlayValues[483] = d483
			ps997.OverlayValues[484] = d484
			ps997.OverlayValues[485] = d485
			ps997.OverlayValues[486] = d486
			ps997.OverlayValues[487] = d487
			ps997.OverlayValues[488] = d488
			ps997.OverlayValues[489] = d489
			ps997.OverlayValues[490] = d490
			ps997.OverlayValues[491] = d491
			ps997.OverlayValues[492] = d492
			ps997.OverlayValues[493] = d493
			ps997.OverlayValues[494] = d494
			ps997.OverlayValues[676] = d676
			ps997.OverlayValues[677] = d677
			ps997.OverlayValues[678] = d678
			ps997.OverlayValues[679] = d679
			ps997.OverlayValues[680] = d680
			ps997.OverlayValues[682] = d682
			ps997.OverlayValues[683] = d683
			ps997.OverlayValues[684] = d684
			ps997.OverlayValues[685] = d685
			ps997.OverlayValues[686] = d686
			ps997.OverlayValues[687] = d687
			ps997.OverlayValues[688] = d688
			ps997.OverlayValues[689] = d689
			ps997.OverlayValues[691] = d691
			ps997.OverlayValues[693] = d693
			ps997.OverlayValues[694] = d694
			ps997.OverlayValues[695] = d695
			ps997.OverlayValues[696] = d696
			ps997.OverlayValues[699] = d699
			ps997.OverlayValues[896] = d896
			ps997.OverlayValues[897] = d897
			ps997.OverlayValues[898] = d898
			ps997.OverlayValues[899] = d899
			ps997.OverlayValues[901] = d901
			ps997.OverlayValues[902] = d902
			ps997.OverlayValues[903] = d903
			ps997.OverlayValues[904] = d904
			ps997.OverlayValues[905] = d905
			ps997.OverlayValues[906] = d906
			ps997.OverlayValues[907] = d907
			ps997.OverlayValues[908] = d908
			ps997.OverlayValues[909] = d909
			ps997.OverlayValues[910] = d910
			ps997.OverlayValues[912] = d912
			ps997.OverlayValues[913] = d913
			ps997.OverlayValues[914] = d914
			ps997.OverlayValues[915] = d915
			ps997.OverlayValues[916] = d916
			ps997.OverlayValues[918] = d918
			ps997.OverlayValues[919] = d919
			ps997.OverlayValues[920] = d920
			ps997.OverlayValues[921] = d921
			ps997.OverlayValues[922] = d922
			ps997.OverlayValues[923] = d923
			ps997.OverlayValues[924] = d924
			ps997.OverlayValues[925] = d925
			ps997.OverlayValues[926] = d926
			ps997.OverlayValues[927] = d927
			ps997.OverlayValues[928] = d928
			ps997.OverlayValues[929] = d929
			ps997.OverlayValues[930] = d930
			ps997.OverlayValues[931] = d931
			ps997.OverlayValues[932] = d932
			ps997.OverlayValues[933] = d933
			ps997.OverlayValues[934] = d934
			ps997.OverlayValues[935] = d935
			ps997.OverlayValues[936] = d936
			ps997.OverlayValues[937] = d937
			ps997.OverlayValues[938] = d938
			ps997.OverlayValues[939] = d939
			ps997.OverlayValues[940] = d940
			ps997.OverlayValues[941] = d941
			ps997.OverlayValues[942] = d942
			ps997.OverlayValues[943] = d943
			ps997.OverlayValues[944] = d944
			ps997.OverlayValues[945] = d945
			ps997.OverlayValues[946] = d946
			ps997.OverlayValues[947] = d947
			ps997.OverlayValues[948] = d948
			ps997.OverlayValues[949] = d949
			ps997.OverlayValues[950] = d950
			ps997.OverlayValues[951] = d951
			ps997.OverlayValues[952] = d952
			ps997.OverlayValues[954] = d954
			ps997.OverlayValues[955] = d955
			ps997.OverlayValues[956] = d956
			ps997.OverlayValues[957] = d957
			ps997.OverlayValues[958] = d958
			ps997.OverlayValues[959] = d959
			ps997.OverlayValues[960] = d960
			ps997.OverlayValues[961] = d961
			ps997.OverlayValues[962] = d962
			ps997.OverlayValues[963] = d963
			ps997.OverlayValues[964] = d964
			ps997.OverlayValues[965] = d965
			ps997.OverlayValues[966] = d966
			ps997.OverlayValues[967] = d967
			ps997.OverlayValues[968] = d968
			ps997.OverlayValues[969] = d969
			ps997.OverlayValues[970] = d970
			ps997.OverlayValues[971] = d971
			ps997.OverlayValues[972] = d972
			ps997.OverlayValues[973] = d973
			ps997.OverlayValues[974] = d974
			ps997.OverlayValues[975] = d975
			ps997.OverlayValues[976] = d976
			ps997.OverlayValues[977] = d977
			ps997.OverlayValues[978] = d978
			ps997.OverlayValues[979] = d979
			ps997.OverlayValues[980] = d980
			ps997.OverlayValues[981] = d981
			ps997.OverlayValues[982] = d982
			ps997.OverlayValues[983] = d983
			ps997.OverlayValues[984] = d984
			ps997.OverlayValues[985] = d985
			ps997.OverlayValues[986] = d986
			ps997.OverlayValues[987] = d987
			ps997.OverlayValues[988] = d988
			ps997.OverlayValues[989] = d989
			ps997.OverlayValues[990] = d990
			ps997.OverlayValues[991] = d991
			ps997.OverlayValues[992] = d992
			ps997.OverlayValues[993] = d993
			ps997.OverlayValues[994] = d994
			ps997.OverlayValues[995] = d995
			ps997.OverlayValues[996] = d996
					return bbs[11].RenderPS(ps997)
				}
			ps998 := scm.PhiState{General: ps.General}
			ps998.OverlayValues = make([]scm.JITValueDesc, 997)
			ps998.OverlayValues[1] = d1
			ps998.OverlayValues[2] = d2
			ps998.OverlayValues[3] = d3
			ps998.OverlayValues[4] = d4
			ps998.OverlayValues[5] = d5
			ps998.OverlayValues[6] = d6
			ps998.OverlayValues[7] = d7
			ps998.OverlayValues[8] = d8
			ps998.OverlayValues[9] = d9
			ps998.OverlayValues[10] = d10
			ps998.OverlayValues[11] = d11
			ps998.OverlayValues[12] = d12
			ps998.OverlayValues[13] = d13
			ps998.OverlayValues[14] = d14
			ps998.OverlayValues[15] = d15
			ps998.OverlayValues[16] = d16
			ps998.OverlayValues[17] = d17
			ps998.OverlayValues[19] = d19
			ps998.OverlayValues[20] = d20
			ps998.OverlayValues[21] = d21
			ps998.OverlayValues[22] = d22
			ps998.OverlayValues[23] = d23
			ps998.OverlayValues[24] = d24
			ps998.OverlayValues[25] = d25
			ps998.OverlayValues[27] = d27
			ps998.OverlayValues[28] = d28
			ps998.OverlayValues[29] = d29
			ps998.OverlayValues[30] = d30
			ps998.OverlayValues[31] = d31
			ps998.OverlayValues[32] = d32
			ps998.OverlayValues[33] = d33
			ps998.OverlayValues[34] = d34
			ps998.OverlayValues[35] = d35
			ps998.OverlayValues[36] = d36
			ps998.OverlayValues[37] = d37
			ps998.OverlayValues[38] = d38
			ps998.OverlayValues[39] = d39
			ps998.OverlayValues[40] = d40
			ps998.OverlayValues[41] = d41
			ps998.OverlayValues[42] = d42
			ps998.OverlayValues[43] = d43
			ps998.OverlayValues[44] = d44
			ps998.OverlayValues[45] = d45
			ps998.OverlayValues[46] = d46
			ps998.OverlayValues[47] = d47
			ps998.OverlayValues[48] = d48
			ps998.OverlayValues[49] = d49
			ps998.OverlayValues[50] = d50
			ps998.OverlayValues[51] = d51
			ps998.OverlayValues[52] = d52
			ps998.OverlayValues[53] = d53
			ps998.OverlayValues[54] = d54
			ps998.OverlayValues[55] = d55
			ps998.OverlayValues[56] = d56
			ps998.OverlayValues[57] = d57
			ps998.OverlayValues[58] = d58
			ps998.OverlayValues[59] = d59
			ps998.OverlayValues[60] = d60
			ps998.OverlayValues[61] = d61
			ps998.OverlayValues[62] = d62
			ps998.OverlayValues[63] = d63
			ps998.OverlayValues[66] = d66
			ps998.OverlayValues[67] = d67
			ps998.OverlayValues[68] = d68
			ps998.OverlayValues[136] = d136
			ps998.OverlayValues[137] = d137
			ps998.OverlayValues[138] = d138
			ps998.OverlayValues[140] = d140
			ps998.OverlayValues[141] = d141
			ps998.OverlayValues[142] = d142
			ps998.OverlayValues[143] = d143
			ps998.OverlayValues[144] = d144
			ps998.OverlayValues[145] = d145
			ps998.OverlayValues[146] = d146
			ps998.OverlayValues[147] = d147
			ps998.OverlayValues[148] = d148
			ps998.OverlayValues[149] = d149
			ps998.OverlayValues[150] = d150
			ps998.OverlayValues[151] = d151
			ps998.OverlayValues[152] = d152
			ps998.OverlayValues[153] = d153
			ps998.OverlayValues[154] = d154
			ps998.OverlayValues[155] = d155
			ps998.OverlayValues[156] = d156
			ps998.OverlayValues[157] = d157
			ps998.OverlayValues[158] = d158
			ps998.OverlayValues[159] = d159
			ps998.OverlayValues[160] = d160
			ps998.OverlayValues[161] = d161
			ps998.OverlayValues[162] = d162
			ps998.OverlayValues[163] = d163
			ps998.OverlayValues[164] = d164
			ps998.OverlayValues[165] = d165
			ps998.OverlayValues[166] = d166
			ps998.OverlayValues[167] = d167
			ps998.OverlayValues[168] = d168
			ps998.OverlayValues[169] = d169
			ps998.OverlayValues[170] = d170
			ps998.OverlayValues[171] = d171
			ps998.OverlayValues[172] = d172
			ps998.OverlayValues[173] = d173
			ps998.OverlayValues[174] = d174
			ps998.OverlayValues[175] = d175
			ps998.OverlayValues[178] = d178
			ps998.OverlayValues[286] = d286
			ps998.OverlayValues[287] = d287
			ps998.OverlayValues[288] = d288
			ps998.OverlayValues[289] = d289
			ps998.OverlayValues[290] = d290
			ps998.OverlayValues[291] = d291
			ps998.OverlayValues[292] = d292
			ps998.OverlayValues[293] = d293
			ps998.OverlayValues[295] = d295
			ps998.OverlayValues[296] = d296
			ps998.OverlayValues[297] = d297
			ps998.OverlayValues[298] = d298
			ps998.OverlayValues[299] = d299
			ps998.OverlayValues[300] = d300
			ps998.OverlayValues[301] = d301
			ps998.OverlayValues[302] = d302
			ps998.OverlayValues[303] = d303
			ps998.OverlayValues[304] = d304
			ps998.OverlayValues[306] = d306
			ps998.OverlayValues[308] = d308
			ps998.OverlayValues[309] = d309
			ps998.OverlayValues[310] = d310
			ps998.OverlayValues[311] = d311
			ps998.OverlayValues[312] = d312
			ps998.OverlayValues[315] = d315
			ps998.OverlayValues[446] = d446
			ps998.OverlayValues[447] = d447
			ps998.OverlayValues[448] = d448
			ps998.OverlayValues[449] = d449
			ps998.OverlayValues[450] = d450
			ps998.OverlayValues[451] = d451
			ps998.OverlayValues[452] = d452
			ps998.OverlayValues[454] = d454
			ps998.OverlayValues[455] = d455
			ps998.OverlayValues[456] = d456
			ps998.OverlayValues[457] = d457
			ps998.OverlayValues[459] = d459
			ps998.OverlayValues[460] = d460
			ps998.OverlayValues[461] = d461
			ps998.OverlayValues[462] = d462
			ps998.OverlayValues[463] = d463
			ps998.OverlayValues[464] = d464
			ps998.OverlayValues[465] = d465
			ps998.OverlayValues[466] = d466
			ps998.OverlayValues[467] = d467
			ps998.OverlayValues[468] = d468
			ps998.OverlayValues[469] = d469
			ps998.OverlayValues[470] = d470
			ps998.OverlayValues[471] = d471
			ps998.OverlayValues[472] = d472
			ps998.OverlayValues[473] = d473
			ps998.OverlayValues[474] = d474
			ps998.OverlayValues[475] = d475
			ps998.OverlayValues[476] = d476
			ps998.OverlayValues[477] = d477
			ps998.OverlayValues[478] = d478
			ps998.OverlayValues[479] = d479
			ps998.OverlayValues[480] = d480
			ps998.OverlayValues[481] = d481
			ps998.OverlayValues[482] = d482
			ps998.OverlayValues[483] = d483
			ps998.OverlayValues[484] = d484
			ps998.OverlayValues[485] = d485
			ps998.OverlayValues[486] = d486
			ps998.OverlayValues[487] = d487
			ps998.OverlayValues[488] = d488
			ps998.OverlayValues[489] = d489
			ps998.OverlayValues[490] = d490
			ps998.OverlayValues[491] = d491
			ps998.OverlayValues[492] = d492
			ps998.OverlayValues[493] = d493
			ps998.OverlayValues[494] = d494
			ps998.OverlayValues[676] = d676
			ps998.OverlayValues[677] = d677
			ps998.OverlayValues[678] = d678
			ps998.OverlayValues[679] = d679
			ps998.OverlayValues[680] = d680
			ps998.OverlayValues[682] = d682
			ps998.OverlayValues[683] = d683
			ps998.OverlayValues[684] = d684
			ps998.OverlayValues[685] = d685
			ps998.OverlayValues[686] = d686
			ps998.OverlayValues[687] = d687
			ps998.OverlayValues[688] = d688
			ps998.OverlayValues[689] = d689
			ps998.OverlayValues[691] = d691
			ps998.OverlayValues[693] = d693
			ps998.OverlayValues[694] = d694
			ps998.OverlayValues[695] = d695
			ps998.OverlayValues[696] = d696
			ps998.OverlayValues[699] = d699
			ps998.OverlayValues[896] = d896
			ps998.OverlayValues[897] = d897
			ps998.OverlayValues[898] = d898
			ps998.OverlayValues[899] = d899
			ps998.OverlayValues[901] = d901
			ps998.OverlayValues[902] = d902
			ps998.OverlayValues[903] = d903
			ps998.OverlayValues[904] = d904
			ps998.OverlayValues[905] = d905
			ps998.OverlayValues[906] = d906
			ps998.OverlayValues[907] = d907
			ps998.OverlayValues[908] = d908
			ps998.OverlayValues[909] = d909
			ps998.OverlayValues[910] = d910
			ps998.OverlayValues[912] = d912
			ps998.OverlayValues[913] = d913
			ps998.OverlayValues[914] = d914
			ps998.OverlayValues[915] = d915
			ps998.OverlayValues[916] = d916
			ps998.OverlayValues[918] = d918
			ps998.OverlayValues[919] = d919
			ps998.OverlayValues[920] = d920
			ps998.OverlayValues[921] = d921
			ps998.OverlayValues[922] = d922
			ps998.OverlayValues[923] = d923
			ps998.OverlayValues[924] = d924
			ps998.OverlayValues[925] = d925
			ps998.OverlayValues[926] = d926
			ps998.OverlayValues[927] = d927
			ps998.OverlayValues[928] = d928
			ps998.OverlayValues[929] = d929
			ps998.OverlayValues[930] = d930
			ps998.OverlayValues[931] = d931
			ps998.OverlayValues[932] = d932
			ps998.OverlayValues[933] = d933
			ps998.OverlayValues[934] = d934
			ps998.OverlayValues[935] = d935
			ps998.OverlayValues[936] = d936
			ps998.OverlayValues[937] = d937
			ps998.OverlayValues[938] = d938
			ps998.OverlayValues[939] = d939
			ps998.OverlayValues[940] = d940
			ps998.OverlayValues[941] = d941
			ps998.OverlayValues[942] = d942
			ps998.OverlayValues[943] = d943
			ps998.OverlayValues[944] = d944
			ps998.OverlayValues[945] = d945
			ps998.OverlayValues[946] = d946
			ps998.OverlayValues[947] = d947
			ps998.OverlayValues[948] = d948
			ps998.OverlayValues[949] = d949
			ps998.OverlayValues[950] = d950
			ps998.OverlayValues[951] = d951
			ps998.OverlayValues[952] = d952
			ps998.OverlayValues[954] = d954
			ps998.OverlayValues[955] = d955
			ps998.OverlayValues[956] = d956
			ps998.OverlayValues[957] = d957
			ps998.OverlayValues[958] = d958
			ps998.OverlayValues[959] = d959
			ps998.OverlayValues[960] = d960
			ps998.OverlayValues[961] = d961
			ps998.OverlayValues[962] = d962
			ps998.OverlayValues[963] = d963
			ps998.OverlayValues[964] = d964
			ps998.OverlayValues[965] = d965
			ps998.OverlayValues[966] = d966
			ps998.OverlayValues[967] = d967
			ps998.OverlayValues[968] = d968
			ps998.OverlayValues[969] = d969
			ps998.OverlayValues[970] = d970
			ps998.OverlayValues[971] = d971
			ps998.OverlayValues[972] = d972
			ps998.OverlayValues[973] = d973
			ps998.OverlayValues[974] = d974
			ps998.OverlayValues[975] = d975
			ps998.OverlayValues[976] = d976
			ps998.OverlayValues[977] = d977
			ps998.OverlayValues[978] = d978
			ps998.OverlayValues[979] = d979
			ps998.OverlayValues[980] = d980
			ps998.OverlayValues[981] = d981
			ps998.OverlayValues[982] = d982
			ps998.OverlayValues[983] = d983
			ps998.OverlayValues[984] = d984
			ps998.OverlayValues[985] = d985
			ps998.OverlayValues[986] = d986
			ps998.OverlayValues[987] = d987
			ps998.OverlayValues[988] = d988
			ps998.OverlayValues[989] = d989
			ps998.OverlayValues[990] = d990
			ps998.OverlayValues[991] = d991
			ps998.OverlayValues[992] = d992
			ps998.OverlayValues[993] = d993
			ps998.OverlayValues[994] = d994
			ps998.OverlayValues[995] = d995
			ps998.OverlayValues[996] = d996
				return bbs[12].RenderPS(ps998)
			}
			if !ps.General {
				ps.General = true
				return bbs[13].RenderPS(ps)
			}
			lbl50 := ctx.ReserveLabel()
			lbl51 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d996.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl50)
			ctx.EmitJmp(lbl51)
			ctx.MarkLabel(lbl50)
			ctx.EmitJmp(lbl12)
			ctx.MarkLabel(lbl51)
			ctx.EmitJmp(lbl13)
			ps999 := scm.PhiState{General: true}
			ps999.OverlayValues = make([]scm.JITValueDesc, 997)
			ps999.OverlayValues[1] = d1
			ps999.OverlayValues[2] = d2
			ps999.OverlayValues[3] = d3
			ps999.OverlayValues[4] = d4
			ps999.OverlayValues[5] = d5
			ps999.OverlayValues[6] = d6
			ps999.OverlayValues[7] = d7
			ps999.OverlayValues[8] = d8
			ps999.OverlayValues[9] = d9
			ps999.OverlayValues[10] = d10
			ps999.OverlayValues[11] = d11
			ps999.OverlayValues[12] = d12
			ps999.OverlayValues[13] = d13
			ps999.OverlayValues[14] = d14
			ps999.OverlayValues[15] = d15
			ps999.OverlayValues[16] = d16
			ps999.OverlayValues[17] = d17
			ps999.OverlayValues[19] = d19
			ps999.OverlayValues[20] = d20
			ps999.OverlayValues[21] = d21
			ps999.OverlayValues[22] = d22
			ps999.OverlayValues[23] = d23
			ps999.OverlayValues[24] = d24
			ps999.OverlayValues[25] = d25
			ps999.OverlayValues[27] = d27
			ps999.OverlayValues[28] = d28
			ps999.OverlayValues[29] = d29
			ps999.OverlayValues[30] = d30
			ps999.OverlayValues[31] = d31
			ps999.OverlayValues[32] = d32
			ps999.OverlayValues[33] = d33
			ps999.OverlayValues[34] = d34
			ps999.OverlayValues[35] = d35
			ps999.OverlayValues[36] = d36
			ps999.OverlayValues[37] = d37
			ps999.OverlayValues[38] = d38
			ps999.OverlayValues[39] = d39
			ps999.OverlayValues[40] = d40
			ps999.OverlayValues[41] = d41
			ps999.OverlayValues[42] = d42
			ps999.OverlayValues[43] = d43
			ps999.OverlayValues[44] = d44
			ps999.OverlayValues[45] = d45
			ps999.OverlayValues[46] = d46
			ps999.OverlayValues[47] = d47
			ps999.OverlayValues[48] = d48
			ps999.OverlayValues[49] = d49
			ps999.OverlayValues[50] = d50
			ps999.OverlayValues[51] = d51
			ps999.OverlayValues[52] = d52
			ps999.OverlayValues[53] = d53
			ps999.OverlayValues[54] = d54
			ps999.OverlayValues[55] = d55
			ps999.OverlayValues[56] = d56
			ps999.OverlayValues[57] = d57
			ps999.OverlayValues[58] = d58
			ps999.OverlayValues[59] = d59
			ps999.OverlayValues[60] = d60
			ps999.OverlayValues[61] = d61
			ps999.OverlayValues[62] = d62
			ps999.OverlayValues[63] = d63
			ps999.OverlayValues[66] = d66
			ps999.OverlayValues[67] = d67
			ps999.OverlayValues[68] = d68
			ps999.OverlayValues[136] = d136
			ps999.OverlayValues[137] = d137
			ps999.OverlayValues[138] = d138
			ps999.OverlayValues[140] = d140
			ps999.OverlayValues[141] = d141
			ps999.OverlayValues[142] = d142
			ps999.OverlayValues[143] = d143
			ps999.OverlayValues[144] = d144
			ps999.OverlayValues[145] = d145
			ps999.OverlayValues[146] = d146
			ps999.OverlayValues[147] = d147
			ps999.OverlayValues[148] = d148
			ps999.OverlayValues[149] = d149
			ps999.OverlayValues[150] = d150
			ps999.OverlayValues[151] = d151
			ps999.OverlayValues[152] = d152
			ps999.OverlayValues[153] = d153
			ps999.OverlayValues[154] = d154
			ps999.OverlayValues[155] = d155
			ps999.OverlayValues[156] = d156
			ps999.OverlayValues[157] = d157
			ps999.OverlayValues[158] = d158
			ps999.OverlayValues[159] = d159
			ps999.OverlayValues[160] = d160
			ps999.OverlayValues[161] = d161
			ps999.OverlayValues[162] = d162
			ps999.OverlayValues[163] = d163
			ps999.OverlayValues[164] = d164
			ps999.OverlayValues[165] = d165
			ps999.OverlayValues[166] = d166
			ps999.OverlayValues[167] = d167
			ps999.OverlayValues[168] = d168
			ps999.OverlayValues[169] = d169
			ps999.OverlayValues[170] = d170
			ps999.OverlayValues[171] = d171
			ps999.OverlayValues[172] = d172
			ps999.OverlayValues[173] = d173
			ps999.OverlayValues[174] = d174
			ps999.OverlayValues[175] = d175
			ps999.OverlayValues[178] = d178
			ps999.OverlayValues[286] = d286
			ps999.OverlayValues[287] = d287
			ps999.OverlayValues[288] = d288
			ps999.OverlayValues[289] = d289
			ps999.OverlayValues[290] = d290
			ps999.OverlayValues[291] = d291
			ps999.OverlayValues[292] = d292
			ps999.OverlayValues[293] = d293
			ps999.OverlayValues[295] = d295
			ps999.OverlayValues[296] = d296
			ps999.OverlayValues[297] = d297
			ps999.OverlayValues[298] = d298
			ps999.OverlayValues[299] = d299
			ps999.OverlayValues[300] = d300
			ps999.OverlayValues[301] = d301
			ps999.OverlayValues[302] = d302
			ps999.OverlayValues[303] = d303
			ps999.OverlayValues[304] = d304
			ps999.OverlayValues[306] = d306
			ps999.OverlayValues[308] = d308
			ps999.OverlayValues[309] = d309
			ps999.OverlayValues[310] = d310
			ps999.OverlayValues[311] = d311
			ps999.OverlayValues[312] = d312
			ps999.OverlayValues[315] = d315
			ps999.OverlayValues[446] = d446
			ps999.OverlayValues[447] = d447
			ps999.OverlayValues[448] = d448
			ps999.OverlayValues[449] = d449
			ps999.OverlayValues[450] = d450
			ps999.OverlayValues[451] = d451
			ps999.OverlayValues[452] = d452
			ps999.OverlayValues[454] = d454
			ps999.OverlayValues[455] = d455
			ps999.OverlayValues[456] = d456
			ps999.OverlayValues[457] = d457
			ps999.OverlayValues[459] = d459
			ps999.OverlayValues[460] = d460
			ps999.OverlayValues[461] = d461
			ps999.OverlayValues[462] = d462
			ps999.OverlayValues[463] = d463
			ps999.OverlayValues[464] = d464
			ps999.OverlayValues[465] = d465
			ps999.OverlayValues[466] = d466
			ps999.OverlayValues[467] = d467
			ps999.OverlayValues[468] = d468
			ps999.OverlayValues[469] = d469
			ps999.OverlayValues[470] = d470
			ps999.OverlayValues[471] = d471
			ps999.OverlayValues[472] = d472
			ps999.OverlayValues[473] = d473
			ps999.OverlayValues[474] = d474
			ps999.OverlayValues[475] = d475
			ps999.OverlayValues[476] = d476
			ps999.OverlayValues[477] = d477
			ps999.OverlayValues[478] = d478
			ps999.OverlayValues[479] = d479
			ps999.OverlayValues[480] = d480
			ps999.OverlayValues[481] = d481
			ps999.OverlayValues[482] = d482
			ps999.OverlayValues[483] = d483
			ps999.OverlayValues[484] = d484
			ps999.OverlayValues[485] = d485
			ps999.OverlayValues[486] = d486
			ps999.OverlayValues[487] = d487
			ps999.OverlayValues[488] = d488
			ps999.OverlayValues[489] = d489
			ps999.OverlayValues[490] = d490
			ps999.OverlayValues[491] = d491
			ps999.OverlayValues[492] = d492
			ps999.OverlayValues[493] = d493
			ps999.OverlayValues[494] = d494
			ps999.OverlayValues[676] = d676
			ps999.OverlayValues[677] = d677
			ps999.OverlayValues[678] = d678
			ps999.OverlayValues[679] = d679
			ps999.OverlayValues[680] = d680
			ps999.OverlayValues[682] = d682
			ps999.OverlayValues[683] = d683
			ps999.OverlayValues[684] = d684
			ps999.OverlayValues[685] = d685
			ps999.OverlayValues[686] = d686
			ps999.OverlayValues[687] = d687
			ps999.OverlayValues[688] = d688
			ps999.OverlayValues[689] = d689
			ps999.OverlayValues[691] = d691
			ps999.OverlayValues[693] = d693
			ps999.OverlayValues[694] = d694
			ps999.OverlayValues[695] = d695
			ps999.OverlayValues[696] = d696
			ps999.OverlayValues[699] = d699
			ps999.OverlayValues[896] = d896
			ps999.OverlayValues[897] = d897
			ps999.OverlayValues[898] = d898
			ps999.OverlayValues[899] = d899
			ps999.OverlayValues[901] = d901
			ps999.OverlayValues[902] = d902
			ps999.OverlayValues[903] = d903
			ps999.OverlayValues[904] = d904
			ps999.OverlayValues[905] = d905
			ps999.OverlayValues[906] = d906
			ps999.OverlayValues[907] = d907
			ps999.OverlayValues[908] = d908
			ps999.OverlayValues[909] = d909
			ps999.OverlayValues[910] = d910
			ps999.OverlayValues[912] = d912
			ps999.OverlayValues[913] = d913
			ps999.OverlayValues[914] = d914
			ps999.OverlayValues[915] = d915
			ps999.OverlayValues[916] = d916
			ps999.OverlayValues[918] = d918
			ps999.OverlayValues[919] = d919
			ps999.OverlayValues[920] = d920
			ps999.OverlayValues[921] = d921
			ps999.OverlayValues[922] = d922
			ps999.OverlayValues[923] = d923
			ps999.OverlayValues[924] = d924
			ps999.OverlayValues[925] = d925
			ps999.OverlayValues[926] = d926
			ps999.OverlayValues[927] = d927
			ps999.OverlayValues[928] = d928
			ps999.OverlayValues[929] = d929
			ps999.OverlayValues[930] = d930
			ps999.OverlayValues[931] = d931
			ps999.OverlayValues[932] = d932
			ps999.OverlayValues[933] = d933
			ps999.OverlayValues[934] = d934
			ps999.OverlayValues[935] = d935
			ps999.OverlayValues[936] = d936
			ps999.OverlayValues[937] = d937
			ps999.OverlayValues[938] = d938
			ps999.OverlayValues[939] = d939
			ps999.OverlayValues[940] = d940
			ps999.OverlayValues[941] = d941
			ps999.OverlayValues[942] = d942
			ps999.OverlayValues[943] = d943
			ps999.OverlayValues[944] = d944
			ps999.OverlayValues[945] = d945
			ps999.OverlayValues[946] = d946
			ps999.OverlayValues[947] = d947
			ps999.OverlayValues[948] = d948
			ps999.OverlayValues[949] = d949
			ps999.OverlayValues[950] = d950
			ps999.OverlayValues[951] = d951
			ps999.OverlayValues[952] = d952
			ps999.OverlayValues[954] = d954
			ps999.OverlayValues[955] = d955
			ps999.OverlayValues[956] = d956
			ps999.OverlayValues[957] = d957
			ps999.OverlayValues[958] = d958
			ps999.OverlayValues[959] = d959
			ps999.OverlayValues[960] = d960
			ps999.OverlayValues[961] = d961
			ps999.OverlayValues[962] = d962
			ps999.OverlayValues[963] = d963
			ps999.OverlayValues[964] = d964
			ps999.OverlayValues[965] = d965
			ps999.OverlayValues[966] = d966
			ps999.OverlayValues[967] = d967
			ps999.OverlayValues[968] = d968
			ps999.OverlayValues[969] = d969
			ps999.OverlayValues[970] = d970
			ps999.OverlayValues[971] = d971
			ps999.OverlayValues[972] = d972
			ps999.OverlayValues[973] = d973
			ps999.OverlayValues[974] = d974
			ps999.OverlayValues[975] = d975
			ps999.OverlayValues[976] = d976
			ps999.OverlayValues[977] = d977
			ps999.OverlayValues[978] = d978
			ps999.OverlayValues[979] = d979
			ps999.OverlayValues[980] = d980
			ps999.OverlayValues[981] = d981
			ps999.OverlayValues[982] = d982
			ps999.OverlayValues[983] = d983
			ps999.OverlayValues[984] = d984
			ps999.OverlayValues[985] = d985
			ps999.OverlayValues[986] = d986
			ps999.OverlayValues[987] = d987
			ps999.OverlayValues[988] = d988
			ps999.OverlayValues[989] = d989
			ps999.OverlayValues[990] = d990
			ps999.OverlayValues[991] = d991
			ps999.OverlayValues[992] = d992
			ps999.OverlayValues[993] = d993
			ps999.OverlayValues[994] = d994
			ps999.OverlayValues[995] = d995
			ps999.OverlayValues[996] = d996
			ps1000 := scm.PhiState{General: true}
			ps1000.OverlayValues = make([]scm.JITValueDesc, 997)
			ps1000.OverlayValues[1] = d1
			ps1000.OverlayValues[2] = d2
			ps1000.OverlayValues[3] = d3
			ps1000.OverlayValues[4] = d4
			ps1000.OverlayValues[5] = d5
			ps1000.OverlayValues[6] = d6
			ps1000.OverlayValues[7] = d7
			ps1000.OverlayValues[8] = d8
			ps1000.OverlayValues[9] = d9
			ps1000.OverlayValues[10] = d10
			ps1000.OverlayValues[11] = d11
			ps1000.OverlayValues[12] = d12
			ps1000.OverlayValues[13] = d13
			ps1000.OverlayValues[14] = d14
			ps1000.OverlayValues[15] = d15
			ps1000.OverlayValues[16] = d16
			ps1000.OverlayValues[17] = d17
			ps1000.OverlayValues[19] = d19
			ps1000.OverlayValues[20] = d20
			ps1000.OverlayValues[21] = d21
			ps1000.OverlayValues[22] = d22
			ps1000.OverlayValues[23] = d23
			ps1000.OverlayValues[24] = d24
			ps1000.OverlayValues[25] = d25
			ps1000.OverlayValues[27] = d27
			ps1000.OverlayValues[28] = d28
			ps1000.OverlayValues[29] = d29
			ps1000.OverlayValues[30] = d30
			ps1000.OverlayValues[31] = d31
			ps1000.OverlayValues[32] = d32
			ps1000.OverlayValues[33] = d33
			ps1000.OverlayValues[34] = d34
			ps1000.OverlayValues[35] = d35
			ps1000.OverlayValues[36] = d36
			ps1000.OverlayValues[37] = d37
			ps1000.OverlayValues[38] = d38
			ps1000.OverlayValues[39] = d39
			ps1000.OverlayValues[40] = d40
			ps1000.OverlayValues[41] = d41
			ps1000.OverlayValues[42] = d42
			ps1000.OverlayValues[43] = d43
			ps1000.OverlayValues[44] = d44
			ps1000.OverlayValues[45] = d45
			ps1000.OverlayValues[46] = d46
			ps1000.OverlayValues[47] = d47
			ps1000.OverlayValues[48] = d48
			ps1000.OverlayValues[49] = d49
			ps1000.OverlayValues[50] = d50
			ps1000.OverlayValues[51] = d51
			ps1000.OverlayValues[52] = d52
			ps1000.OverlayValues[53] = d53
			ps1000.OverlayValues[54] = d54
			ps1000.OverlayValues[55] = d55
			ps1000.OverlayValues[56] = d56
			ps1000.OverlayValues[57] = d57
			ps1000.OverlayValues[58] = d58
			ps1000.OverlayValues[59] = d59
			ps1000.OverlayValues[60] = d60
			ps1000.OverlayValues[61] = d61
			ps1000.OverlayValues[62] = d62
			ps1000.OverlayValues[63] = d63
			ps1000.OverlayValues[66] = d66
			ps1000.OverlayValues[67] = d67
			ps1000.OverlayValues[68] = d68
			ps1000.OverlayValues[136] = d136
			ps1000.OverlayValues[137] = d137
			ps1000.OverlayValues[138] = d138
			ps1000.OverlayValues[140] = d140
			ps1000.OverlayValues[141] = d141
			ps1000.OverlayValues[142] = d142
			ps1000.OverlayValues[143] = d143
			ps1000.OverlayValues[144] = d144
			ps1000.OverlayValues[145] = d145
			ps1000.OverlayValues[146] = d146
			ps1000.OverlayValues[147] = d147
			ps1000.OverlayValues[148] = d148
			ps1000.OverlayValues[149] = d149
			ps1000.OverlayValues[150] = d150
			ps1000.OverlayValues[151] = d151
			ps1000.OverlayValues[152] = d152
			ps1000.OverlayValues[153] = d153
			ps1000.OverlayValues[154] = d154
			ps1000.OverlayValues[155] = d155
			ps1000.OverlayValues[156] = d156
			ps1000.OverlayValues[157] = d157
			ps1000.OverlayValues[158] = d158
			ps1000.OverlayValues[159] = d159
			ps1000.OverlayValues[160] = d160
			ps1000.OverlayValues[161] = d161
			ps1000.OverlayValues[162] = d162
			ps1000.OverlayValues[163] = d163
			ps1000.OverlayValues[164] = d164
			ps1000.OverlayValues[165] = d165
			ps1000.OverlayValues[166] = d166
			ps1000.OverlayValues[167] = d167
			ps1000.OverlayValues[168] = d168
			ps1000.OverlayValues[169] = d169
			ps1000.OverlayValues[170] = d170
			ps1000.OverlayValues[171] = d171
			ps1000.OverlayValues[172] = d172
			ps1000.OverlayValues[173] = d173
			ps1000.OverlayValues[174] = d174
			ps1000.OverlayValues[175] = d175
			ps1000.OverlayValues[178] = d178
			ps1000.OverlayValues[286] = d286
			ps1000.OverlayValues[287] = d287
			ps1000.OverlayValues[288] = d288
			ps1000.OverlayValues[289] = d289
			ps1000.OverlayValues[290] = d290
			ps1000.OverlayValues[291] = d291
			ps1000.OverlayValues[292] = d292
			ps1000.OverlayValues[293] = d293
			ps1000.OverlayValues[295] = d295
			ps1000.OverlayValues[296] = d296
			ps1000.OverlayValues[297] = d297
			ps1000.OverlayValues[298] = d298
			ps1000.OverlayValues[299] = d299
			ps1000.OverlayValues[300] = d300
			ps1000.OverlayValues[301] = d301
			ps1000.OverlayValues[302] = d302
			ps1000.OverlayValues[303] = d303
			ps1000.OverlayValues[304] = d304
			ps1000.OverlayValues[306] = d306
			ps1000.OverlayValues[308] = d308
			ps1000.OverlayValues[309] = d309
			ps1000.OverlayValues[310] = d310
			ps1000.OverlayValues[311] = d311
			ps1000.OverlayValues[312] = d312
			ps1000.OverlayValues[315] = d315
			ps1000.OverlayValues[446] = d446
			ps1000.OverlayValues[447] = d447
			ps1000.OverlayValues[448] = d448
			ps1000.OverlayValues[449] = d449
			ps1000.OverlayValues[450] = d450
			ps1000.OverlayValues[451] = d451
			ps1000.OverlayValues[452] = d452
			ps1000.OverlayValues[454] = d454
			ps1000.OverlayValues[455] = d455
			ps1000.OverlayValues[456] = d456
			ps1000.OverlayValues[457] = d457
			ps1000.OverlayValues[459] = d459
			ps1000.OverlayValues[460] = d460
			ps1000.OverlayValues[461] = d461
			ps1000.OverlayValues[462] = d462
			ps1000.OverlayValues[463] = d463
			ps1000.OverlayValues[464] = d464
			ps1000.OverlayValues[465] = d465
			ps1000.OverlayValues[466] = d466
			ps1000.OverlayValues[467] = d467
			ps1000.OverlayValues[468] = d468
			ps1000.OverlayValues[469] = d469
			ps1000.OverlayValues[470] = d470
			ps1000.OverlayValues[471] = d471
			ps1000.OverlayValues[472] = d472
			ps1000.OverlayValues[473] = d473
			ps1000.OverlayValues[474] = d474
			ps1000.OverlayValues[475] = d475
			ps1000.OverlayValues[476] = d476
			ps1000.OverlayValues[477] = d477
			ps1000.OverlayValues[478] = d478
			ps1000.OverlayValues[479] = d479
			ps1000.OverlayValues[480] = d480
			ps1000.OverlayValues[481] = d481
			ps1000.OverlayValues[482] = d482
			ps1000.OverlayValues[483] = d483
			ps1000.OverlayValues[484] = d484
			ps1000.OverlayValues[485] = d485
			ps1000.OverlayValues[486] = d486
			ps1000.OverlayValues[487] = d487
			ps1000.OverlayValues[488] = d488
			ps1000.OverlayValues[489] = d489
			ps1000.OverlayValues[490] = d490
			ps1000.OverlayValues[491] = d491
			ps1000.OverlayValues[492] = d492
			ps1000.OverlayValues[493] = d493
			ps1000.OverlayValues[494] = d494
			ps1000.OverlayValues[676] = d676
			ps1000.OverlayValues[677] = d677
			ps1000.OverlayValues[678] = d678
			ps1000.OverlayValues[679] = d679
			ps1000.OverlayValues[680] = d680
			ps1000.OverlayValues[682] = d682
			ps1000.OverlayValues[683] = d683
			ps1000.OverlayValues[684] = d684
			ps1000.OverlayValues[685] = d685
			ps1000.OverlayValues[686] = d686
			ps1000.OverlayValues[687] = d687
			ps1000.OverlayValues[688] = d688
			ps1000.OverlayValues[689] = d689
			ps1000.OverlayValues[691] = d691
			ps1000.OverlayValues[693] = d693
			ps1000.OverlayValues[694] = d694
			ps1000.OverlayValues[695] = d695
			ps1000.OverlayValues[696] = d696
			ps1000.OverlayValues[699] = d699
			ps1000.OverlayValues[896] = d896
			ps1000.OverlayValues[897] = d897
			ps1000.OverlayValues[898] = d898
			ps1000.OverlayValues[899] = d899
			ps1000.OverlayValues[901] = d901
			ps1000.OverlayValues[902] = d902
			ps1000.OverlayValues[903] = d903
			ps1000.OverlayValues[904] = d904
			ps1000.OverlayValues[905] = d905
			ps1000.OverlayValues[906] = d906
			ps1000.OverlayValues[907] = d907
			ps1000.OverlayValues[908] = d908
			ps1000.OverlayValues[909] = d909
			ps1000.OverlayValues[910] = d910
			ps1000.OverlayValues[912] = d912
			ps1000.OverlayValues[913] = d913
			ps1000.OverlayValues[914] = d914
			ps1000.OverlayValues[915] = d915
			ps1000.OverlayValues[916] = d916
			ps1000.OverlayValues[918] = d918
			ps1000.OverlayValues[919] = d919
			ps1000.OverlayValues[920] = d920
			ps1000.OverlayValues[921] = d921
			ps1000.OverlayValues[922] = d922
			ps1000.OverlayValues[923] = d923
			ps1000.OverlayValues[924] = d924
			ps1000.OverlayValues[925] = d925
			ps1000.OverlayValues[926] = d926
			ps1000.OverlayValues[927] = d927
			ps1000.OverlayValues[928] = d928
			ps1000.OverlayValues[929] = d929
			ps1000.OverlayValues[930] = d930
			ps1000.OverlayValues[931] = d931
			ps1000.OverlayValues[932] = d932
			ps1000.OverlayValues[933] = d933
			ps1000.OverlayValues[934] = d934
			ps1000.OverlayValues[935] = d935
			ps1000.OverlayValues[936] = d936
			ps1000.OverlayValues[937] = d937
			ps1000.OverlayValues[938] = d938
			ps1000.OverlayValues[939] = d939
			ps1000.OverlayValues[940] = d940
			ps1000.OverlayValues[941] = d941
			ps1000.OverlayValues[942] = d942
			ps1000.OverlayValues[943] = d943
			ps1000.OverlayValues[944] = d944
			ps1000.OverlayValues[945] = d945
			ps1000.OverlayValues[946] = d946
			ps1000.OverlayValues[947] = d947
			ps1000.OverlayValues[948] = d948
			ps1000.OverlayValues[949] = d949
			ps1000.OverlayValues[950] = d950
			ps1000.OverlayValues[951] = d951
			ps1000.OverlayValues[952] = d952
			ps1000.OverlayValues[954] = d954
			ps1000.OverlayValues[955] = d955
			ps1000.OverlayValues[956] = d956
			ps1000.OverlayValues[957] = d957
			ps1000.OverlayValues[958] = d958
			ps1000.OverlayValues[959] = d959
			ps1000.OverlayValues[960] = d960
			ps1000.OverlayValues[961] = d961
			ps1000.OverlayValues[962] = d962
			ps1000.OverlayValues[963] = d963
			ps1000.OverlayValues[964] = d964
			ps1000.OverlayValues[965] = d965
			ps1000.OverlayValues[966] = d966
			ps1000.OverlayValues[967] = d967
			ps1000.OverlayValues[968] = d968
			ps1000.OverlayValues[969] = d969
			ps1000.OverlayValues[970] = d970
			ps1000.OverlayValues[971] = d971
			ps1000.OverlayValues[972] = d972
			ps1000.OverlayValues[973] = d973
			ps1000.OverlayValues[974] = d974
			ps1000.OverlayValues[975] = d975
			ps1000.OverlayValues[976] = d976
			ps1000.OverlayValues[977] = d977
			ps1000.OverlayValues[978] = d978
			ps1000.OverlayValues[979] = d979
			ps1000.OverlayValues[980] = d980
			ps1000.OverlayValues[981] = d981
			ps1000.OverlayValues[982] = d982
			ps1000.OverlayValues[983] = d983
			ps1000.OverlayValues[984] = d984
			ps1000.OverlayValues[985] = d985
			ps1000.OverlayValues[986] = d986
			ps1000.OverlayValues[987] = d987
			ps1000.OverlayValues[988] = d988
			ps1000.OverlayValues[989] = d989
			ps1000.OverlayValues[990] = d990
			ps1000.OverlayValues[991] = d991
			ps1000.OverlayValues[992] = d992
			ps1000.OverlayValues[993] = d993
			ps1000.OverlayValues[994] = d994
			ps1000.OverlayValues[995] = d995
			ps1000.OverlayValues[996] = d996
			snap1001 := d1
			snap1002 := d2
			snap1003 := d3
			snap1004 := d4
			snap1005 := d5
			snap1006 := d6
			snap1007 := d7
			snap1008 := d8
			snap1009 := d9
			snap1010 := d10
			snap1011 := d11
			snap1012 := d12
			snap1013 := d13
			snap1014 := d14
			snap1015 := d15
			snap1016 := d16
			snap1017 := d17
			snap1018 := d19
			snap1019 := d20
			snap1020 := d21
			snap1021 := d22
			snap1022 := d23
			snap1023 := d24
			snap1024 := d25
			snap1025 := d27
			snap1026 := d28
			snap1027 := d29
			snap1028 := d30
			snap1029 := d31
			snap1030 := d32
			snap1031 := d33
			snap1032 := d34
			snap1033 := d35
			snap1034 := d36
			snap1035 := d37
			snap1036 := d38
			snap1037 := d39
			snap1038 := d40
			snap1039 := d41
			snap1040 := d42
			snap1041 := d43
			snap1042 := d44
			snap1043 := d45
			snap1044 := d46
			snap1045 := d47
			snap1046 := d48
			snap1047 := d49
			snap1048 := d50
			snap1049 := d51
			snap1050 := d52
			snap1051 := d53
			snap1052 := d54
			snap1053 := d55
			snap1054 := d56
			snap1055 := d57
			snap1056 := d58
			snap1057 := d59
			snap1058 := d60
			snap1059 := d61
			snap1060 := d62
			snap1061 := d63
			snap1062 := d66
			snap1063 := d67
			snap1064 := d68
			snap1065 := d136
			snap1066 := d137
			snap1067 := d138
			snap1068 := d140
			snap1069 := d141
			snap1070 := d142
			snap1071 := d143
			snap1072 := d144
			snap1073 := d145
			snap1074 := d146
			snap1075 := d147
			snap1076 := d148
			snap1077 := d149
			snap1078 := d150
			snap1079 := d151
			snap1080 := d152
			snap1081 := d153
			snap1082 := d154
			snap1083 := d155
			snap1084 := d156
			snap1085 := d157
			snap1086 := d158
			snap1087 := d159
			snap1088 := d160
			snap1089 := d161
			snap1090 := d162
			snap1091 := d163
			snap1092 := d164
			snap1093 := d165
			snap1094 := d166
			snap1095 := d167
			snap1096 := d168
			snap1097 := d169
			snap1098 := d170
			snap1099 := d171
			snap1100 := d172
			snap1101 := d173
			snap1102 := d174
			snap1103 := d175
			snap1104 := d178
			snap1105 := d286
			snap1106 := d287
			snap1107 := d288
			snap1108 := d289
			snap1109 := d290
			snap1110 := d291
			snap1111 := d292
			snap1112 := d293
			snap1113 := d295
			snap1114 := d296
			snap1115 := d297
			snap1116 := d298
			snap1117 := d299
			snap1118 := d300
			snap1119 := d301
			snap1120 := d302
			snap1121 := d303
			snap1122 := d304
			snap1123 := d306
			snap1124 := d308
			snap1125 := d309
			snap1126 := d310
			snap1127 := d311
			snap1128 := d312
			snap1129 := d315
			snap1130 := d446
			snap1131 := d447
			snap1132 := d448
			snap1133 := d449
			snap1134 := d450
			snap1135 := d451
			snap1136 := d452
			snap1137 := d454
			snap1138 := d455
			snap1139 := d456
			snap1140 := d457
			snap1141 := d459
			snap1142 := d460
			snap1143 := d461
			snap1144 := d462
			snap1145 := d463
			snap1146 := d464
			snap1147 := d465
			snap1148 := d466
			snap1149 := d467
			snap1150 := d468
			snap1151 := d469
			snap1152 := d470
			snap1153 := d471
			snap1154 := d472
			snap1155 := d473
			snap1156 := d474
			snap1157 := d475
			snap1158 := d476
			snap1159 := d477
			snap1160 := d478
			snap1161 := d479
			snap1162 := d480
			snap1163 := d481
			snap1164 := d482
			snap1165 := d483
			snap1166 := d484
			snap1167 := d485
			snap1168 := d486
			snap1169 := d487
			snap1170 := d488
			snap1171 := d489
			snap1172 := d490
			snap1173 := d491
			snap1174 := d492
			snap1175 := d493
			snap1176 := d494
			snap1177 := d676
			snap1178 := d677
			snap1179 := d678
			snap1180 := d679
			snap1181 := d680
			snap1182 := d682
			snap1183 := d683
			snap1184 := d684
			snap1185 := d685
			snap1186 := d686
			snap1187 := d687
			snap1188 := d688
			snap1189 := d689
			snap1190 := d691
			snap1191 := d693
			snap1192 := d694
			snap1193 := d695
			snap1194 := d696
			snap1195 := d699
			snap1196 := d896
			snap1197 := d897
			snap1198 := d898
			snap1199 := d899
			snap1200 := d901
			snap1201 := d902
			snap1202 := d903
			snap1203 := d904
			snap1204 := d905
			snap1205 := d906
			snap1206 := d907
			snap1207 := d908
			snap1208 := d909
			snap1209 := d910
			snap1210 := d912
			snap1211 := d913
			snap1212 := d914
			snap1213 := d915
			snap1214 := d916
			snap1215 := d918
			snap1216 := d919
			snap1217 := d920
			snap1218 := d921
			snap1219 := d922
			snap1220 := d923
			snap1221 := d924
			snap1222 := d925
			snap1223 := d926
			snap1224 := d927
			snap1225 := d928
			snap1226 := d929
			snap1227 := d930
			snap1228 := d931
			snap1229 := d932
			snap1230 := d933
			snap1231 := d934
			snap1232 := d935
			snap1233 := d936
			snap1234 := d937
			snap1235 := d938
			snap1236 := d939
			snap1237 := d940
			snap1238 := d941
			snap1239 := d942
			snap1240 := d943
			snap1241 := d944
			snap1242 := d945
			snap1243 := d946
			snap1244 := d947
			snap1245 := d948
			snap1246 := d949
			snap1247 := d950
			snap1248 := d951
			snap1249 := d952
			snap1250 := d954
			snap1251 := d955
			snap1252 := d956
			snap1253 := d957
			snap1254 := d958
			snap1255 := d959
			snap1256 := d960
			snap1257 := d961
			snap1258 := d962
			snap1259 := d963
			snap1260 := d964
			snap1261 := d965
			snap1262 := d966
			snap1263 := d967
			snap1264 := d968
			snap1265 := d969
			snap1266 := d970
			snap1267 := d971
			snap1268 := d972
			snap1269 := d973
			snap1270 := d974
			snap1271 := d975
			snap1272 := d976
			snap1273 := d977
			snap1274 := d978
			snap1275 := d979
			snap1276 := d980
			snap1277 := d981
			snap1278 := d982
			snap1279 := d983
			snap1280 := d984
			snap1281 := d985
			snap1282 := d986
			snap1283 := d987
			snap1284 := d988
			snap1285 := d989
			snap1286 := d990
			snap1287 := d991
			snap1288 := d992
			snap1289 := d993
			snap1290 := d994
			snap1291 := d995
			snap1292 := d996
			alloc1293 := ctx.SnapshotAllocState()
			if !bbs[12].Rendered {
				bbs[12].RenderPS(ps1000)
			}
			ctx.RestoreAllocState(alloc1293)
			d1 = snap1001
			d2 = snap1002
			d3 = snap1003
			d4 = snap1004
			d5 = snap1005
			d6 = snap1006
			d7 = snap1007
			d8 = snap1008
			d9 = snap1009
			d10 = snap1010
			d11 = snap1011
			d12 = snap1012
			d13 = snap1013
			d14 = snap1014
			d15 = snap1015
			d16 = snap1016
			d17 = snap1017
			d19 = snap1018
			d20 = snap1019
			d21 = snap1020
			d22 = snap1021
			d23 = snap1022
			d24 = snap1023
			d25 = snap1024
			d27 = snap1025
			d28 = snap1026
			d29 = snap1027
			d30 = snap1028
			d31 = snap1029
			d32 = snap1030
			d33 = snap1031
			d34 = snap1032
			d35 = snap1033
			d36 = snap1034
			d37 = snap1035
			d38 = snap1036
			d39 = snap1037
			d40 = snap1038
			d41 = snap1039
			d42 = snap1040
			d43 = snap1041
			d44 = snap1042
			d45 = snap1043
			d46 = snap1044
			d47 = snap1045
			d48 = snap1046
			d49 = snap1047
			d50 = snap1048
			d51 = snap1049
			d52 = snap1050
			d53 = snap1051
			d54 = snap1052
			d55 = snap1053
			d56 = snap1054
			d57 = snap1055
			d58 = snap1056
			d59 = snap1057
			d60 = snap1058
			d61 = snap1059
			d62 = snap1060
			d63 = snap1061
			d66 = snap1062
			d67 = snap1063
			d68 = snap1064
			d136 = snap1065
			d137 = snap1066
			d138 = snap1067
			d140 = snap1068
			d141 = snap1069
			d142 = snap1070
			d143 = snap1071
			d144 = snap1072
			d145 = snap1073
			d146 = snap1074
			d147 = snap1075
			d148 = snap1076
			d149 = snap1077
			d150 = snap1078
			d151 = snap1079
			d152 = snap1080
			d153 = snap1081
			d154 = snap1082
			d155 = snap1083
			d156 = snap1084
			d157 = snap1085
			d158 = snap1086
			d159 = snap1087
			d160 = snap1088
			d161 = snap1089
			d162 = snap1090
			d163 = snap1091
			d164 = snap1092
			d165 = snap1093
			d166 = snap1094
			d167 = snap1095
			d168 = snap1096
			d169 = snap1097
			d170 = snap1098
			d171 = snap1099
			d172 = snap1100
			d173 = snap1101
			d174 = snap1102
			d175 = snap1103
			d178 = snap1104
			d286 = snap1105
			d287 = snap1106
			d288 = snap1107
			d289 = snap1108
			d290 = snap1109
			d291 = snap1110
			d292 = snap1111
			d293 = snap1112
			d295 = snap1113
			d296 = snap1114
			d297 = snap1115
			d298 = snap1116
			d299 = snap1117
			d300 = snap1118
			d301 = snap1119
			d302 = snap1120
			d303 = snap1121
			d304 = snap1122
			d306 = snap1123
			d308 = snap1124
			d309 = snap1125
			d310 = snap1126
			d311 = snap1127
			d312 = snap1128
			d315 = snap1129
			d446 = snap1130
			d447 = snap1131
			d448 = snap1132
			d449 = snap1133
			d450 = snap1134
			d451 = snap1135
			d452 = snap1136
			d454 = snap1137
			d455 = snap1138
			d456 = snap1139
			d457 = snap1140
			d459 = snap1141
			d460 = snap1142
			d461 = snap1143
			d462 = snap1144
			d463 = snap1145
			d464 = snap1146
			d465 = snap1147
			d466 = snap1148
			d467 = snap1149
			d468 = snap1150
			d469 = snap1151
			d470 = snap1152
			d471 = snap1153
			d472 = snap1154
			d473 = snap1155
			d474 = snap1156
			d475 = snap1157
			d476 = snap1158
			d477 = snap1159
			d478 = snap1160
			d479 = snap1161
			d480 = snap1162
			d481 = snap1163
			d482 = snap1164
			d483 = snap1165
			d484 = snap1166
			d485 = snap1167
			d486 = snap1168
			d487 = snap1169
			d488 = snap1170
			d489 = snap1171
			d490 = snap1172
			d491 = snap1173
			d492 = snap1174
			d493 = snap1175
			d494 = snap1176
			d676 = snap1177
			d677 = snap1178
			d678 = snap1179
			d679 = snap1180
			d680 = snap1181
			d682 = snap1182
			d683 = snap1183
			d684 = snap1184
			d685 = snap1185
			d686 = snap1186
			d687 = snap1187
			d688 = snap1188
			d689 = snap1189
			d691 = snap1190
			d693 = snap1191
			d694 = snap1192
			d695 = snap1193
			d696 = snap1194
			d699 = snap1195
			d896 = snap1196
			d897 = snap1197
			d898 = snap1198
			d899 = snap1199
			d901 = snap1200
			d902 = snap1201
			d903 = snap1202
			d904 = snap1203
			d905 = snap1204
			d906 = snap1205
			d907 = snap1206
			d908 = snap1207
			d909 = snap1208
			d910 = snap1209
			d912 = snap1210
			d913 = snap1211
			d914 = snap1212
			d915 = snap1213
			d916 = snap1214
			d918 = snap1215
			d919 = snap1216
			d920 = snap1217
			d921 = snap1218
			d922 = snap1219
			d923 = snap1220
			d924 = snap1221
			d925 = snap1222
			d926 = snap1223
			d927 = snap1224
			d928 = snap1225
			d929 = snap1226
			d930 = snap1227
			d931 = snap1228
			d932 = snap1229
			d933 = snap1230
			d934 = snap1231
			d935 = snap1232
			d936 = snap1233
			d937 = snap1234
			d938 = snap1235
			d939 = snap1236
			d940 = snap1237
			d941 = snap1238
			d942 = snap1239
			d943 = snap1240
			d944 = snap1241
			d945 = snap1242
			d946 = snap1243
			d947 = snap1244
			d948 = snap1245
			d949 = snap1246
			d950 = snap1247
			d951 = snap1248
			d952 = snap1249
			d954 = snap1250
			d955 = snap1251
			d956 = snap1252
			d957 = snap1253
			d958 = snap1254
			d959 = snap1255
			d960 = snap1256
			d961 = snap1257
			d962 = snap1258
			d963 = snap1259
			d964 = snap1260
			d965 = snap1261
			d966 = snap1262
			d967 = snap1263
			d968 = snap1264
			d969 = snap1265
			d970 = snap1266
			d971 = snap1267
			d972 = snap1268
			d973 = snap1269
			d974 = snap1270
			d975 = snap1271
			d976 = snap1272
			d977 = snap1273
			d978 = snap1274
			d979 = snap1275
			d980 = snap1276
			d981 = snap1277
			d982 = snap1278
			d983 = snap1279
			d984 = snap1280
			d985 = snap1281
			d986 = snap1282
			d987 = snap1283
			d988 = snap1284
			d989 = snap1285
			d990 = snap1286
			d991 = snap1287
			d992 = snap1288
			d993 = snap1289
			d994 = snap1290
			d995 = snap1291
			d996 = snap1292
			if !bbs[11].Rendered {
				return bbs[11].RenderPS(ps999)
			}
			return result
			ctx.FreeDesc(&d995)
			return result
			}
			ps1294 := scm.PhiState{General: false}
			_ = bbs[0].RenderPS(ps1294)
			ctx.MarkLabel(lbl0)
			d1295 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d1295)
			ctx.BindReg(r1, &d1295)
			ctx.EmitMovPairToResult(&d1295, &result)
			ctx.FreeReg(r0)
			ctx.FreeReg(r1)
			ctx.ResolveFixups()
			if idxPinned { ctx.UnprotectReg(idxPinnedReg) }
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
