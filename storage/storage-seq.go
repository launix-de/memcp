/*
Copyright (C) 2023  Carl-Philip Hänsch

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
			var d14 scm.JITValueDesc
			_ = d14
			var d15 scm.JITValueDesc
			_ = d15
			var d16 scm.JITValueDesc
			_ = d16
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
			var d64 scm.JITValueDesc
			_ = d64
			var d65 scm.JITValueDesc
			_ = d65
			var d66 scm.JITValueDesc
			_ = d66
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
			var d175 scm.JITValueDesc
			_ = d175
			var d283 scm.JITValueDesc
			_ = d283
			var d284 scm.JITValueDesc
			_ = d284
			var d285 scm.JITValueDesc
			_ = d285
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
			var d292 scm.JITValueDesc
			_ = d292
			var d293 scm.JITValueDesc
			_ = d293
			var d294 scm.JITValueDesc
			_ = d294
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
			var d303 scm.JITValueDesc
			_ = d303
			var d305 scm.JITValueDesc
			_ = d305
			var d306 scm.JITValueDesc
			_ = d306
			var d307 scm.JITValueDesc
			_ = d307
			var d308 scm.JITValueDesc
			_ = d308
			var d309 scm.JITValueDesc
			_ = d309
			var d312 scm.JITValueDesc
			_ = d312
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
			var d672 scm.JITValueDesc
			_ = d672
			var d673 scm.JITValueDesc
			_ = d673
			var d674 scm.JITValueDesc
			_ = d674
			var d675 scm.JITValueDesc
			_ = d675
			var d676 scm.JITValueDesc
			_ = d676
			var d678 scm.JITValueDesc
			_ = d678
			var d679 scm.JITValueDesc
			_ = d679
			var d680 scm.JITValueDesc
			_ = d680
			var d681 scm.JITValueDesc
			_ = d681
			var d682 scm.JITValueDesc
			_ = d682
			var d683 scm.JITValueDesc
			_ = d683
			var d684 scm.JITValueDesc
			_ = d684
			var d685 scm.JITValueDesc
			_ = d685
			var d687 scm.JITValueDesc
			_ = d687
			var d689 scm.JITValueDesc
			_ = d689
			var d690 scm.JITValueDesc
			_ = d690
			var d691 scm.JITValueDesc
			_ = d691
			var d692 scm.JITValueDesc
			_ = d692
			var d695 scm.JITValueDesc
			_ = d695
			var d892 scm.JITValueDesc
			_ = d892
			var d893 scm.JITValueDesc
			_ = d893
			var d894 scm.JITValueDesc
			_ = d894
			var d895 scm.JITValueDesc
			_ = d895
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
			var d953 scm.JITValueDesc
			_ = d953
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
			r0 := ctx.EmitSubRSP32Fixup()
			_ = r0
			if thisptr.MemPtr == 0 && (thisptr.Loc == scm.LocStack || thisptr.Loc == scm.LocStackPair) {
				thisptr.StackOff += int32(144)
			}
			if idxInt.MemPtr == 0 && (idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair) {
				idxInt.StackOff += int32(144)
			}
			if result.MemPtr == 0 && (result.Loc == scm.LocStack || result.Loc == scm.LocStackPair) {
				result.StackOff += int32(144)
			}
			d0 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			d2 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			d3 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
			d4 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
			d5 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
			d6 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
			d7 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
			d8 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
			var bbs [14]scm.BBDescriptor
			bbs[1].PhiBase = int32(0)
			bbs[1].PhiCount = uint16(3)
			bbs[2].PhiBase = int32(48)
			bbs[2].PhiCount = uint16(1)
			bbs[4].PhiBase = int32(64)
			bbs[4].PhiCount = uint16(3)
			bbs[8].PhiBase = int32(112)
			bbs[8].PhiCount = uint16(2)
			if result.Loc == scm.LocAny {
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			r1 := ctx.AllocReg()
			r2 := ctx.AllocRegExcept(r1)
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
				if bbs[0].VisitCount >= 2 {
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
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			ctx.ReclaimUntrackedRegs()
			r3 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).lastValue)
				ctx.EmitMovRegMem64(r3, fieldAddr)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
				ctx.EmitMovRegMem(r3, thisptr.Reg, off)
			}
			d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r3}
			ctx.BindReg(r3, &d9)
			ctx.EnsureDesc(&d9)
			ctx.EnsureDesc(&d9)
			var d10 scm.JITValueDesc
			if d9.Loc == scm.LocImm {
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d9.Imm.Int()))))}
			} else {
				r4 := ctx.AllocReg()
				ctx.EmitMovRegReg(r4, d9.Reg)
				ctx.EmitShlRegImm8(r4, 32)
				ctx.EmitShrRegImm8(r4, 32)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r4}
				ctx.BindReg(r4, &d10)
			}
			ctx.FreeDesc(&d9)
			var d11 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).seqCount)
				r5 := ctx.AllocReg()
				ctx.EmitMovRegMem32(r5, fieldAddr)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r5}
				ctx.BindReg(r5, &d11)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).seqCount))
				r6 := ctx.AllocReg()
				ctx.EmitMovRegMemL(r6, thisptr.Reg, off)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r6}
				ctx.BindReg(r6, &d11)
			}
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d11)
			var d12 scm.JITValueDesc
			if d11.Loc == scm.LocImm {
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d11.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d11.Reg)
				ctx.EmitMovRegReg(scratch, d11.Reg)
				ctx.EmitSubRegImm32(scratch, int32(1))
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d12)
			}
			if d12.Loc == scm.LocImm {
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: d12.Type, Imm: scm.NewInt(int64(uint64(d12.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d12.Reg, 32)
				ctx.EmitShrRegImm8(d12.Reg, 32)
			}
			if d12.Loc == scm.LocReg && d11.Loc == scm.LocReg && d12.Reg == d11.Reg {
				ctx.TransferReg(d11.Reg)
				d11.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d10)
			if d10.Loc == scm.LocReg {
				ctx.ProtectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.ProtectReg(d10.Reg)
				ctx.ProtectReg(d10.Reg2)
			}
			ctx.EnsureDesc(&d12)
			if d12.Loc == scm.LocReg {
				ctx.ProtectReg(d12.Reg)
			} else if d12.Loc == scm.LocRegPair {
				ctx.ProtectReg(d12.Reg)
				ctx.ProtectReg(d12.Reg2)
			}
			d13 = d10
			if d13.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d13)
			d14 = d13
			if d14.Loc == scm.LocImm {
				d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: d14.Type, Imm: scm.NewInt(int64(uint64(d14.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d14.Reg, 32)
				ctx.EmitShrRegImm8(d14.Reg, 32)
			}
			ctx.EmitStoreToStack(d14, int32(bbs[1].PhiBase)+int32(0))
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(bbs[1].PhiBase)+int32(16))
			d15 = d12
			if d15.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d15)
			d16 = d15
			if d16.Loc == scm.LocImm {
				d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: d16.Type, Imm: scm.NewInt(int64(uint64(d16.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d16.Reg, 32)
				ctx.EmitShrRegImm8(d16.Reg, 32)
			}
			ctx.EmitStoreToStack(d16, int32(bbs[1].PhiBase)+int32(32))
			if d10.Loc == scm.LocReg {
				ctx.UnprotectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d10.Reg)
				ctx.UnprotectReg(d10.Reg2)
			}
			if d12.Loc == scm.LocReg {
				ctx.UnprotectReg(d12.Reg)
			} else if d12.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d12.Reg)
				ctx.UnprotectReg(d12.Reg2)
			}
			ps17 := scm.PhiState{General: ps.General}
			ps17.OverlayValues = make([]scm.JITValueDesc, 17)
			ps17.OverlayValues[0] = d0
			ps17.OverlayValues[1] = d1
			ps17.OverlayValues[2] = d2
			ps17.OverlayValues[3] = d3
			ps17.OverlayValues[4] = d4
			ps17.OverlayValues[5] = d5
			ps17.OverlayValues[6] = d6
			ps17.OverlayValues[7] = d7
			ps17.OverlayValues[8] = d8
			ps17.OverlayValues[9] = d9
			ps17.OverlayValues[10] = d10
			ps17.OverlayValues[11] = d11
			ps17.OverlayValues[12] = d12
			ps17.OverlayValues[13] = d13
			ps17.OverlayValues[14] = d14
			ps17.OverlayValues[15] = d15
			ps17.OverlayValues[16] = d16
			ps17.PhiValues = make([]scm.JITValueDesc, 3)
			d18 = d10
			ps17.PhiValues[0] = d18
			d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
			ps17.PhiValues[1] = d19
			d20 = d12
			ps17.PhiValues[2] = d20
			if ps17.General && bbs[1].Rendered {
				ctx.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps17)
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
				if bbs[1].VisitCount >= 2 {
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
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
				d14 = ps.OverlayValues[14]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
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
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d0 = ps.PhiValues[0]
			}
			if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d1 = ps.PhiValues[1]
			}
			if !ps.General && len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d2 = ps.PhiValues[2]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			d24 = d0
			_ = d24
			r7 := d0.Loc == scm.LocReg
			r8 := d0.Reg
			if r7 { ctx.ProtectReg(r8) }
			d25 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(144)}
			lbl15 := ctx.ReserveLabel()
			bbpos_1_0 := int32(-1)
			_ = bbpos_1_0
			bbpos_1_1 := int32(-1)
			_ = bbpos_1_1
			bbpos_1_2 := int32(-1)
			_ = bbpos_1_2
			bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d25 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(144)}
			ctx.EnsureDesc(&d24)
			ctx.EnsureDesc(&d24)
			var d26 scm.JITValueDesc
			if d24.Loc == scm.LocImm {
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d24.Imm.Int()))))}
			} else {
				r9 := ctx.AllocReg()
				ctx.EmitMovRegReg(r9, d24.Reg)
				ctx.EmitShlRegImm8(r9, 32)
				ctx.EmitShrRegImm8(r9, 32)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
				ctx.BindReg(r9, &d26)
			}
			var d27 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				r10 := ctx.AllocReg()
				ctx.EmitMovRegMem8(r10, fieldAddr)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r10}
				ctx.BindReg(r10, &d27)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r11 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r11, thisptr.Reg, off)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r11}
				ctx.BindReg(r11, &d27)
			}
			ctx.EnsureDesc(&d27)
			ctx.EnsureDesc(&d27)
			var d28 scm.JITValueDesc
			if d27.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d27.Imm.Int()))))}
			} else {
				r12 := ctx.AllocReg()
				ctx.EmitMovRegReg(r12, d27.Reg)
				ctx.EmitShlRegImm8(r12, 56)
				ctx.EmitShrRegImm8(r12, 56)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
				ctx.BindReg(r12, &d28)
			}
			ctx.EnsureDesc(&d26)
			ctx.EnsureDesc(&d28)
			ctx.EnsureDesc(&d26)
			ctx.EnsureDesc(&d28)
			ctx.EnsureDesc(&d26)
			ctx.EnsureDesc(&d28)
			var d29 scm.JITValueDesc
			if d26.Loc == scm.LocImm && d28.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d26.Imm.Int() * d28.Imm.Int())}
			} else if d26.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d28.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d26.Imm.Int()))
				ctx.EmitImulInt64(scratch, d28.Reg)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d29)
			} else if d28.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d26.Reg)
				ctx.EmitMovRegReg(scratch, d26.Reg)
				if d28.Imm.Int() >= -2147483648 && d28.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d28.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d28.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d29)
			} else {
				r13 := ctx.AllocRegExcept(d26.Reg, d28.Reg)
				ctx.EmitMovRegReg(r13, d26.Reg)
				ctx.EmitImulInt64(r13, d28.Reg)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
				ctx.BindReg(r13, &d29)
			}
			if d29.Loc == scm.LocReg && d26.Loc == scm.LocReg && d29.Reg == d26.Reg {
				ctx.TransferReg(d26.Reg)
				d26.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d26)
			ctx.FreeDesc(&d28)
			var d30 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				r14 := ctx.AllocReg()
				r15 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r14, fieldAddr)
				ctx.EmitMovRegMem64(r15, fieldAddr+8)
				d30 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r14, Reg2: r15}
				ctx.BindReg(r14, &d30)
				ctx.BindReg(r15, &d30)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				r16 := ctx.AllocReg()
				r17 := ctx.AllocReg()
				ctx.EmitMovRegMem(r16, thisptr.Reg, off)
				ctx.EmitMovRegMem(r17, thisptr.Reg, off+8)
				d30 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r16, Reg2: r17}
				ctx.BindReg(r16, &d30)
				ctx.BindReg(r17, &d30)
			}
			ctx.EnsureDesc(&d29)
			var d31 scm.JITValueDesc
			if d29.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d29.Imm.Int() / 64)}
			} else {
				r18 := ctx.AllocRegExcept(d29.Reg)
				ctx.EmitMovRegReg(r18, d29.Reg)
				ctx.EmitShrRegImm8(r18, 6)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
				ctx.BindReg(r18, &d31)
			}
			if d31.Loc == scm.LocReg && d29.Loc == scm.LocReg && d31.Reg == d29.Reg {
				ctx.TransferReg(d29.Reg)
				d29.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d31)
			r19 := ctx.AllocReg()
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d30)
			if d31.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r19, uint64(d31.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r19, d31.Reg)
				ctx.EmitShlRegImm8(r19, 3)
			}
			if d30.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d30.Imm.Int()))
				ctx.EmitAddInt64(r19, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r19, d30.Reg)
			}
			r20 := ctx.AllocRegExcept(r19)
			ctx.EmitMovRegMem(r20, r19, 0)
			ctx.FreeReg(r19)
			d32 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r20}
			ctx.BindReg(r20, &d32)
			ctx.FreeDesc(&d31)
			ctx.EnsureDesc(&d29)
			var d33 scm.JITValueDesc
			if d29.Loc == scm.LocImm {
				d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d29.Imm.Int() % 64)}
			} else {
				r21 := ctx.AllocRegExcept(d29.Reg)
				ctx.EmitMovRegReg(r21, d29.Reg)
				ctx.EmitAndRegImm32(r21, 63)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d33)
			}
			if d33.Loc == scm.LocReg && d29.Loc == scm.LocReg && d33.Reg == d29.Reg {
				ctx.TransferReg(d29.Reg)
				d29.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d33)
			var d34 scm.JITValueDesc
			if d32.Loc == scm.LocImm && d33.Loc == scm.LocImm {
				d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d32.Imm.Int()) << uint64(d33.Imm.Int())))}
			} else if d33.Loc == scm.LocImm {
				r22 := ctx.AllocRegExcept(d32.Reg)
				ctx.EmitMovRegReg(r22, d32.Reg)
				ctx.EmitShlRegImm8(r22, uint8(d33.Imm.Int()))
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d34)
			} else {
				{
					shiftSrc := d32.Reg
					r23 := ctx.AllocRegExcept(d32.Reg)
					ctx.EmitMovRegReg(r23, d32.Reg)
					shiftSrc = r23
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d33.Reg != scm.RegRCX
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
			if d34.Loc == scm.LocReg && d32.Loc == scm.LocReg && d34.Reg == d32.Reg {
				ctx.TransferReg(d32.Reg)
				d32.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d32)
			ctx.FreeDesc(&d33)
			ctx.EnsureDesc(&d29)
			var d35 scm.JITValueDesc
			if d29.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d29.Imm.Int() % 64)}
			} else {
				r24 := ctx.AllocRegExcept(d29.Reg)
				ctx.EmitMovRegReg(r24, d29.Reg)
				ctx.EmitAndRegImm32(r24, 63)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
				ctx.BindReg(r24, &d35)
			}
			if d35.Loc == scm.LocReg && d29.Loc == scm.LocReg && d35.Reg == d29.Reg {
				ctx.TransferReg(d29.Reg)
				d29.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d27)
			ctx.EnsureDesc(&d27)
			var d36 scm.JITValueDesc
			if d27.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d27.Imm.Int()))))}
			} else {
				r25 := ctx.AllocReg()
				ctx.EmitMovRegReg(r25, d27.Reg)
				ctx.EmitShlRegImm8(r25, 56)
				ctx.EmitShrRegImm8(r25, 56)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
				ctx.BindReg(r25, &d36)
			}
			ctx.EnsureDesc(&d35)
			ctx.EnsureDesc(&d36)
			ctx.EnsureDesc(&d35)
			ctx.EnsureDesc(&d36)
			ctx.EnsureDesc(&d35)
			ctx.EnsureDesc(&d36)
			var d37 scm.JITValueDesc
			if d35.Loc == scm.LocImm && d36.Loc == scm.LocImm {
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d35.Imm.Int() + d36.Imm.Int())}
			} else if d36.Loc == scm.LocImm && d36.Imm.Int() == 0 {
				r26 := ctx.AllocRegExcept(d35.Reg)
				ctx.EmitMovRegReg(r26, d35.Reg)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
				ctx.BindReg(r26, &d37)
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
				r27 := ctx.AllocRegExcept(d35.Reg, d36.Reg)
				ctx.EmitMovRegReg(r27, d35.Reg)
				ctx.EmitAddInt64(r27, d36.Reg)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d37)
			}
			if d37.Loc == scm.LocReg && d35.Loc == scm.LocReg && d37.Reg == d35.Reg {
				ctx.TransferReg(d35.Reg)
				d35.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d35)
			ctx.FreeDesc(&d36)
			ctx.EnsureDesc(&d37)
			var d38 scm.JITValueDesc
			if d37.Loc == scm.LocImm {
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d37.Imm.Int()) > uint64(64))}
			} else {
				r28 := ctx.AllocRegExcept(d37.Reg)
				ctx.EmitCmpRegImm32(d37.Reg, 64)
				ctx.EmitSetcc(r28, scm.CcA)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r28}
				ctx.BindReg(r28, &d38)
			}
			ctx.FreeDesc(&d37)
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
			ctx.EnsureDesc(&d34)
			if d34.Loc == scm.LocReg {
				ctx.ProtectReg(d34.Reg)
			} else if d34.Loc == scm.LocRegPair {
				ctx.ProtectReg(d34.Reg)
				ctx.ProtectReg(d34.Reg2)
			}
			d40 = d34
			if d40.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d40)
			ctx.EmitStoreToStack(d40, int32(bbs[2].PhiBase)+int32(0))
			if d34.Loc == scm.LocReg {
				ctx.UnprotectReg(d34.Reg)
			} else if d34.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d34.Reg)
				ctx.UnprotectReg(d34.Reg2)
			}
					ctx.EmitJmp(lbl17)
				}
			} else {
				ctx.EmitCmpRegImm32(d39.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl18)
				ctx.EmitJmp(lbl19)
				ctx.MarkLabel(lbl18)
				ctx.EmitJmp(lbl16)
				ctx.MarkLabel(lbl19)
			ctx.EnsureDesc(&d34)
			if d34.Loc == scm.LocReg {
				ctx.ProtectReg(d34.Reg)
			} else if d34.Loc == scm.LocRegPair {
				ctx.ProtectReg(d34.Reg)
				ctx.ProtectReg(d34.Reg2)
			}
			d41 = d34
			if d41.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d41)
			ctx.EmitStoreToStack(d41, int32(bbs[2].PhiBase)+int32(0))
			if d34.Loc == scm.LocReg {
				ctx.UnprotectReg(d34.Reg)
			} else if d34.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d34.Reg)
				ctx.UnprotectReg(d34.Reg2)
			}
				ctx.EmitJmp(lbl17)
			}
			ctx.FreeDesc(&d38)
			bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl17)
			ctx.ResolveFixups()
			d25 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(144)}
			ctx.EnsureDesc(&d27)
			ctx.EnsureDesc(&d27)
			var d42 scm.JITValueDesc
			if d27.Loc == scm.LocImm {
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d27.Imm.Int()))))}
			} else {
				r29 := ctx.AllocReg()
				ctx.EmitMovRegReg(r29, d27.Reg)
				ctx.EmitShlRegImm8(r29, 56)
				ctx.EmitShrRegImm8(r29, 56)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d42)
			}
			d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d43)
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d43)
			ctx.EnsureDesc(&d42)
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
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d44)
			var d45 scm.JITValueDesc
			if d25.Loc == scm.LocImm && d44.Loc == scm.LocImm {
				d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d25.Imm.Int()) >> uint64(d44.Imm.Int())))}
			} else if d44.Loc == scm.LocImm {
				r32 := ctx.AllocRegExcept(d25.Reg)
				ctx.EmitMovRegReg(r32, d25.Reg)
				ctx.EmitShrRegImm8(r32, uint8(d44.Imm.Int()))
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
				ctx.BindReg(r32, &d45)
			} else {
				{
					shiftSrc := d25.Reg
					r33 := ctx.AllocRegExcept(d25.Reg)
					ctx.EmitMovRegReg(r33, d25.Reg)
					shiftSrc = r33
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d44.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d44.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d44.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d45)
				}
			}
			if d45.Loc == scm.LocReg && d25.Loc == scm.LocReg && d45.Reg == d25.Reg {
				ctx.TransferReg(d25.Reg)
				d25.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d25)
			ctx.FreeDesc(&d44)
			r34 := ctx.AllocReg()
			ctx.EnsureDesc(&d45)
			ctx.EnsureDesc(&d45)
			if d45.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r34, d45)
			}
			ctx.EmitJmp(lbl15)
			bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl16)
			ctx.ResolveFixups()
			d25 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(144)}
			ctx.EnsureDesc(&d29)
			var d46 scm.JITValueDesc
			if d29.Loc == scm.LocImm {
				d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d29.Imm.Int() / 64)}
			} else {
				r35 := ctx.AllocRegExcept(d29.Reg)
				ctx.EmitMovRegReg(r35, d29.Reg)
				ctx.EmitShrRegImm8(r35, 6)
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d46)
			}
			if d46.Loc == scm.LocReg && d29.Loc == scm.LocReg && d46.Reg == d29.Reg {
				ctx.TransferReg(d29.Reg)
				d29.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d46)
			ctx.EnsureDesc(&d46)
			var d47 scm.JITValueDesc
			if d46.Loc == scm.LocImm {
				d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d46.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d46.Reg)
				ctx.EmitMovRegReg(scratch, d46.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d47)
			}
			if d47.Loc == scm.LocReg && d46.Loc == scm.LocReg && d47.Reg == d46.Reg {
				ctx.TransferReg(d46.Reg)
				d46.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d46)
			ctx.EnsureDesc(&d47)
			r36 := ctx.AllocReg()
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d30)
			if d47.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r36, uint64(d47.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r36, d47.Reg)
				ctx.EmitShlRegImm8(r36, 3)
			}
			if d30.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d30.Imm.Int()))
				ctx.EmitAddInt64(r36, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r36, d30.Reg)
			}
			r37 := ctx.AllocRegExcept(r36)
			ctx.EmitMovRegMem(r37, r36, 0)
			ctx.FreeReg(r36)
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r37}
			ctx.BindReg(r37, &d48)
			ctx.FreeDesc(&d47)
			ctx.EnsureDesc(&d29)
			var d49 scm.JITValueDesc
			if d29.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d29.Imm.Int() % 64)}
			} else {
				r38 := ctx.AllocRegExcept(d29.Reg)
				ctx.EmitMovRegReg(r38, d29.Reg)
				ctx.EmitAndRegImm32(r38, 63)
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
				ctx.BindReg(r38, &d49)
			}
			if d49.Loc == scm.LocReg && d29.Loc == scm.LocReg && d49.Reg == d29.Reg {
				ctx.TransferReg(d29.Reg)
				d29.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d29)
			d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d49)
			var d51 scm.JITValueDesc
			if d50.Loc == scm.LocImm && d49.Loc == scm.LocImm {
				d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d50.Imm.Int() - d49.Imm.Int())}
			} else if d49.Loc == scm.LocImm && d49.Imm.Int() == 0 {
				r39 := ctx.AllocRegExcept(d50.Reg)
				ctx.EmitMovRegReg(r39, d50.Reg)
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d51)
			} else if d50.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d49.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d50.Imm.Int()))
				ctx.EmitSubInt64(scratch, d49.Reg)
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d51)
			} else if d49.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d50.Reg)
				ctx.EmitMovRegReg(scratch, d50.Reg)
				if d49.Imm.Int() >= -2147483648 && d49.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d49.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d49.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d51)
			} else {
				r40 := ctx.AllocRegExcept(d50.Reg, d49.Reg)
				ctx.EmitMovRegReg(r40, d50.Reg)
				ctx.EmitSubInt64(r40, d49.Reg)
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
				ctx.BindReg(r40, &d51)
			}
			if d51.Loc == scm.LocReg && d50.Loc == scm.LocReg && d51.Reg == d50.Reg {
				ctx.TransferReg(d50.Reg)
				d50.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d49)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d51)
			var d52 scm.JITValueDesc
			if d48.Loc == scm.LocImm && d51.Loc == scm.LocImm {
				d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d48.Imm.Int()) >> uint64(d51.Imm.Int())))}
			} else if d51.Loc == scm.LocImm {
				r41 := ctx.AllocRegExcept(d48.Reg)
				ctx.EmitMovRegReg(r41, d48.Reg)
				ctx.EmitShrRegImm8(r41, uint8(d51.Imm.Int()))
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d52)
			} else {
				{
					shiftSrc := d48.Reg
					r42 := ctx.AllocRegExcept(d48.Reg)
					ctx.EmitMovRegReg(r42, d48.Reg)
					shiftSrc = r42
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d51.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d51.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d51.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d52)
				}
			}
			if d52.Loc == scm.LocReg && d48.Loc == scm.LocReg && d52.Reg == d48.Reg {
				ctx.TransferReg(d48.Reg)
				d48.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d48)
			ctx.FreeDesc(&d51)
			ctx.EnsureDesc(&d34)
			ctx.EnsureDesc(&d52)
			var d53 scm.JITValueDesc
			if d34.Loc == scm.LocImm && d52.Loc == scm.LocImm {
				d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d34.Imm.Int() | d52.Imm.Int())}
			} else if d34.Loc == scm.LocImm && d34.Imm.Int() == 0 {
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d52.Reg}
				ctx.BindReg(d52.Reg, &d53)
			} else if d52.Loc == scm.LocImm && d52.Imm.Int() == 0 {
				r43 := ctx.AllocRegExcept(d34.Reg)
				ctx.EmitMovRegReg(r43, d34.Reg)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
				ctx.BindReg(r43, &d53)
			} else if d34.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d52.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d34.Imm.Int()))
				ctx.EmitOrInt64(scratch, d52.Reg)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d53)
			} else if d52.Loc == scm.LocImm {
				r44 := ctx.AllocRegExcept(d34.Reg)
				ctx.EmitMovRegReg(r44, d34.Reg)
				if d52.Imm.Int() >= -2147483648 && d52.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r44, int32(d52.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d52.Imm.Int()))
					ctx.EmitOrInt64(r44, scm.RegR11)
				}
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d53)
			} else {
				r45 := ctx.AllocRegExcept(d34.Reg, d52.Reg)
				ctx.EmitMovRegReg(r45, d34.Reg)
				ctx.EmitOrInt64(r45, d52.Reg)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
				ctx.BindReg(r45, &d53)
			}
			if d53.Loc == scm.LocReg && d34.Loc == scm.LocReg && d53.Reg == d34.Reg {
				ctx.TransferReg(d34.Reg)
				d34.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d52)
			ctx.EnsureDesc(&d53)
			if d53.Loc == scm.LocReg {
				ctx.ProtectReg(d53.Reg)
			} else if d53.Loc == scm.LocRegPair {
				ctx.ProtectReg(d53.Reg)
				ctx.ProtectReg(d53.Reg2)
			}
			d54 = d53
			if d54.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d54)
			ctx.EmitStoreToStack(d54, int32(bbs[2].PhiBase)+int32(0))
			if d53.Loc == scm.LocReg {
				ctx.UnprotectReg(d53.Reg)
			} else if d53.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d53.Reg)
				ctx.UnprotectReg(d53.Reg2)
			}
			ctx.EmitJmp(lbl17)
			ctx.MarkLabel(lbl15)
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r34}
			ctx.BindReg(r34, &d55)
			ctx.BindReg(r34, &d55)
			if r7 { ctx.UnprotectReg(r8) }
			ctx.EnsureDesc(&d55)
			ctx.EnsureDesc(&d55)
			var d56 scm.JITValueDesc
			if d55.Loc == scm.LocImm {
				d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d55.Imm.Int()))))}
			} else {
				r46 := ctx.AllocReg()
				ctx.EmitMovRegReg(r46, d55.Reg)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d56)
			}
			ctx.FreeDesc(&d55)
			var d57 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				r47 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r47, fieldAddr)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r47}
				ctx.BindReg(r47, &d57)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r48 := ctx.AllocReg()
				ctx.EmitMovRegMem(r48, thisptr.Reg, off)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r48}
				ctx.BindReg(r48, &d57)
			}
			ctx.EnsureDesc(&d56)
			ctx.EnsureDesc(&d57)
			ctx.EnsureDesc(&d56)
			ctx.EnsureDesc(&d57)
			ctx.EnsureDesc(&d56)
			ctx.EnsureDesc(&d57)
			var d58 scm.JITValueDesc
			if d56.Loc == scm.LocImm && d57.Loc == scm.LocImm {
				d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d56.Imm.Int() + d57.Imm.Int())}
			} else if d57.Loc == scm.LocImm && d57.Imm.Int() == 0 {
				r49 := ctx.AllocRegExcept(d56.Reg)
				ctx.EmitMovRegReg(r49, d56.Reg)
				d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
				ctx.BindReg(r49, &d58)
			} else if d56.Loc == scm.LocImm && d56.Imm.Int() == 0 {
				d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d57.Reg}
				ctx.BindReg(d57.Reg, &d58)
			} else if d56.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d57.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d56.Imm.Int()))
				ctx.EmitAddInt64(scratch, d57.Reg)
				d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d58)
			} else if d57.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d56.Reg)
				ctx.EmitMovRegReg(scratch, d56.Reg)
				if d57.Imm.Int() >= -2147483648 && d57.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d57.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d57.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d58)
			} else {
				r50 := ctx.AllocRegExcept(d56.Reg, d57.Reg)
				ctx.EmitMovRegReg(r50, d56.Reg)
				ctx.EmitAddInt64(r50, d57.Reg)
				d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
				ctx.BindReg(r50, &d58)
			}
			if d58.Loc == scm.LocReg && d56.Loc == scm.LocReg && d58.Reg == d56.Reg {
				ctx.TransferReg(d56.Reg)
				d56.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d56)
			ctx.EnsureDesc(&d58)
			ctx.EnsureDesc(&d58)
			var d59 scm.JITValueDesc
			if d58.Loc == scm.LocImm {
				d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d58.Imm.Int()))))}
			} else {
				r51 := ctx.AllocReg()
				ctx.EmitMovRegReg(r51, d58.Reg)
				ctx.EmitShlRegImm8(r51, 32)
				ctx.EmitShrRegImm8(r51, 32)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
				ctx.BindReg(r51, &d59)
			}
			ctx.FreeDesc(&d58)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d59)
			var d60 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d59.Loc == scm.LocImm {
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d59.Imm.Int()))}
			} else if d59.Loc == scm.LocImm {
				r52 := ctx.AllocRegExcept(idxInt.Reg)
				if d59.Imm.Int() >= -2147483648 && d59.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(idxInt.Reg, int32(d59.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d59.Imm.Int()))
					ctx.EmitCmpInt64(idxInt.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r52, scm.CcB)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r52}
				ctx.BindReg(r52, &d60)
			} else if idxInt.Loc == scm.LocImm {
				r53 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d59.Reg)
				ctx.EmitSetcc(r53, scm.CcB)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r53}
				ctx.BindReg(r53, &d60)
			} else {
				r54 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.EmitCmpInt64(idxInt.Reg, d59.Reg)
				ctx.EmitSetcc(r54, scm.CcB)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r54}
				ctx.BindReg(r54, &d60)
			}
			ctx.FreeDesc(&d59)
			d61 = d60
			ctx.EnsureDesc(&d61)
			if d61.Loc != scm.LocImm && d61.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d61.Loc == scm.LocImm {
				if d61.Imm.Bool() {
			ps62 := scm.PhiState{General: ps.General}
			ps62.OverlayValues = make([]scm.JITValueDesc, 62)
			ps62.OverlayValues[0] = d0
			ps62.OverlayValues[1] = d1
			ps62.OverlayValues[2] = d2
			ps62.OverlayValues[3] = d3
			ps62.OverlayValues[4] = d4
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
			ps62.OverlayValues[18] = d18
			ps62.OverlayValues[19] = d19
			ps62.OverlayValues[20] = d20
			ps62.OverlayValues[21] = d21
			ps62.OverlayValues[22] = d22
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
			ps62.OverlayValues[57] = d57
			ps62.OverlayValues[58] = d58
			ps62.OverlayValues[59] = d59
			ps62.OverlayValues[60] = d60
			ps62.OverlayValues[61] = d61
					return bbs[3].RenderPS(ps62)
				}
			ps63 := scm.PhiState{General: ps.General}
			ps63.OverlayValues = make([]scm.JITValueDesc, 62)
			ps63.OverlayValues[0] = d0
			ps63.OverlayValues[1] = d1
			ps63.OverlayValues[2] = d2
			ps63.OverlayValues[3] = d3
			ps63.OverlayValues[4] = d4
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
			ps63.OverlayValues[18] = d18
			ps63.OverlayValues[19] = d19
			ps63.OverlayValues[20] = d20
			ps63.OverlayValues[21] = d21
			ps63.OverlayValues[22] = d22
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
			ps63.OverlayValues[57] = d57
			ps63.OverlayValues[58] = d58
			ps63.OverlayValues[59] = d59
			ps63.OverlayValues[60] = d60
			ps63.OverlayValues[61] = d61
				return bbs[5].RenderPS(ps63)
			}
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
					d64 := ps.PhiValues[0]
					ctx.EnsureDesc(&d64)
					ctx.EmitStoreToStack(d64, int32(bbs[1].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
					d65 := ps.PhiValues[1]
					ctx.EnsureDesc(&d65)
					ctx.EmitStoreToStack(d65, int32(bbs[1].PhiBase)+int32(16))
				}
				if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
					d66 := ps.PhiValues[2]
					ctx.EnsureDesc(&d66)
					ctx.EmitStoreToStack(d66, int32(bbs[1].PhiBase)+int32(32))
				}
				ps.General = true
				return bbs[1].RenderPS(ps)
			}
			lbl20 := ctx.ReserveLabel()
			lbl21 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d61.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl20)
			ctx.EmitJmp(lbl21)
			ctx.MarkLabel(lbl20)
			ctx.EmitJmp(lbl4)
			ctx.MarkLabel(lbl21)
			ctx.EmitJmp(lbl6)
			ps67 := scm.PhiState{General: true}
			ps67.OverlayValues = make([]scm.JITValueDesc, 67)
			ps67.OverlayValues[0] = d0
			ps67.OverlayValues[1] = d1
			ps67.OverlayValues[2] = d2
			ps67.OverlayValues[3] = d3
			ps67.OverlayValues[4] = d4
			ps67.OverlayValues[5] = d5
			ps67.OverlayValues[6] = d6
			ps67.OverlayValues[7] = d7
			ps67.OverlayValues[8] = d8
			ps67.OverlayValues[9] = d9
			ps67.OverlayValues[10] = d10
			ps67.OverlayValues[11] = d11
			ps67.OverlayValues[12] = d12
			ps67.OverlayValues[13] = d13
			ps67.OverlayValues[14] = d14
			ps67.OverlayValues[15] = d15
			ps67.OverlayValues[16] = d16
			ps67.OverlayValues[18] = d18
			ps67.OverlayValues[19] = d19
			ps67.OverlayValues[20] = d20
			ps67.OverlayValues[21] = d21
			ps67.OverlayValues[22] = d22
			ps67.OverlayValues[23] = d23
			ps67.OverlayValues[24] = d24
			ps67.OverlayValues[25] = d25
			ps67.OverlayValues[26] = d26
			ps67.OverlayValues[27] = d27
			ps67.OverlayValues[28] = d28
			ps67.OverlayValues[29] = d29
			ps67.OverlayValues[30] = d30
			ps67.OverlayValues[31] = d31
			ps67.OverlayValues[32] = d32
			ps67.OverlayValues[33] = d33
			ps67.OverlayValues[34] = d34
			ps67.OverlayValues[35] = d35
			ps67.OverlayValues[36] = d36
			ps67.OverlayValues[37] = d37
			ps67.OverlayValues[38] = d38
			ps67.OverlayValues[39] = d39
			ps67.OverlayValues[40] = d40
			ps67.OverlayValues[41] = d41
			ps67.OverlayValues[42] = d42
			ps67.OverlayValues[43] = d43
			ps67.OverlayValues[44] = d44
			ps67.OverlayValues[45] = d45
			ps67.OverlayValues[46] = d46
			ps67.OverlayValues[47] = d47
			ps67.OverlayValues[48] = d48
			ps67.OverlayValues[49] = d49
			ps67.OverlayValues[50] = d50
			ps67.OverlayValues[51] = d51
			ps67.OverlayValues[52] = d52
			ps67.OverlayValues[53] = d53
			ps67.OverlayValues[54] = d54
			ps67.OverlayValues[55] = d55
			ps67.OverlayValues[56] = d56
			ps67.OverlayValues[57] = d57
			ps67.OverlayValues[58] = d58
			ps67.OverlayValues[59] = d59
			ps67.OverlayValues[60] = d60
			ps67.OverlayValues[61] = d61
			ps67.OverlayValues[64] = d64
			ps67.OverlayValues[65] = d65
			ps67.OverlayValues[66] = d66
			ps68 := scm.PhiState{General: true}
			ps68.OverlayValues = make([]scm.JITValueDesc, 67)
			ps68.OverlayValues[0] = d0
			ps68.OverlayValues[1] = d1
			ps68.OverlayValues[2] = d2
			ps68.OverlayValues[3] = d3
			ps68.OverlayValues[4] = d4
			ps68.OverlayValues[5] = d5
			ps68.OverlayValues[6] = d6
			ps68.OverlayValues[7] = d7
			ps68.OverlayValues[8] = d8
			ps68.OverlayValues[9] = d9
			ps68.OverlayValues[10] = d10
			ps68.OverlayValues[11] = d11
			ps68.OverlayValues[12] = d12
			ps68.OverlayValues[13] = d13
			ps68.OverlayValues[14] = d14
			ps68.OverlayValues[15] = d15
			ps68.OverlayValues[16] = d16
			ps68.OverlayValues[18] = d18
			ps68.OverlayValues[19] = d19
			ps68.OverlayValues[20] = d20
			ps68.OverlayValues[21] = d21
			ps68.OverlayValues[22] = d22
			ps68.OverlayValues[23] = d23
			ps68.OverlayValues[24] = d24
			ps68.OverlayValues[25] = d25
			ps68.OverlayValues[26] = d26
			ps68.OverlayValues[27] = d27
			ps68.OverlayValues[28] = d28
			ps68.OverlayValues[29] = d29
			ps68.OverlayValues[30] = d30
			ps68.OverlayValues[31] = d31
			ps68.OverlayValues[32] = d32
			ps68.OverlayValues[33] = d33
			ps68.OverlayValues[34] = d34
			ps68.OverlayValues[35] = d35
			ps68.OverlayValues[36] = d36
			ps68.OverlayValues[37] = d37
			ps68.OverlayValues[38] = d38
			ps68.OverlayValues[39] = d39
			ps68.OverlayValues[40] = d40
			ps68.OverlayValues[41] = d41
			ps68.OverlayValues[42] = d42
			ps68.OverlayValues[43] = d43
			ps68.OverlayValues[44] = d44
			ps68.OverlayValues[45] = d45
			ps68.OverlayValues[46] = d46
			ps68.OverlayValues[47] = d47
			ps68.OverlayValues[48] = d48
			ps68.OverlayValues[49] = d49
			ps68.OverlayValues[50] = d50
			ps68.OverlayValues[51] = d51
			ps68.OverlayValues[52] = d52
			ps68.OverlayValues[53] = d53
			ps68.OverlayValues[54] = d54
			ps68.OverlayValues[55] = d55
			ps68.OverlayValues[56] = d56
			ps68.OverlayValues[57] = d57
			ps68.OverlayValues[58] = d58
			ps68.OverlayValues[59] = d59
			ps68.OverlayValues[60] = d60
			ps68.OverlayValues[61] = d61
			ps68.OverlayValues[64] = d64
			ps68.OverlayValues[65] = d65
			ps68.OverlayValues[66] = d66
			snap69 := d0
			snap70 := d1
			snap71 := d2
			snap72 := d3
			snap73 := d4
			snap74 := d5
			snap75 := d6
			snap76 := d7
			snap77 := d8
			snap78 := d9
			snap79 := d10
			snap80 := d11
			snap81 := d12
			snap82 := d13
			snap83 := d14
			snap84 := d15
			snap85 := d16
			snap86 := d18
			snap87 := d19
			snap88 := d20
			snap89 := d21
			snap90 := d22
			snap91 := d23
			snap92 := d24
			snap93 := d25
			snap94 := d26
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
			snap130 := d64
			snap131 := d65
			snap132 := d66
			alloc133 := ctx.SnapshotAllocState()
			if !bbs[5].Rendered {
				bbs[5].RenderPS(ps68)
			}
			ctx.RestoreAllocState(alloc133)
			d0 = snap69
			d1 = snap70
			d2 = snap71
			d3 = snap72
			d4 = snap73
			d5 = snap74
			d6 = snap75
			d7 = snap76
			d8 = snap77
			d9 = snap78
			d10 = snap79
			d11 = snap80
			d12 = snap81
			d13 = snap82
			d14 = snap83
			d15 = snap84
			d16 = snap85
			d18 = snap86
			d19 = snap87
			d20 = snap88
			d21 = snap89
			d22 = snap90
			d23 = snap91
			d24 = snap92
			d25 = snap93
			d26 = snap94
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
			d64 = snap130
			d65 = snap131
			d66 = snap132
			if !bbs[3].Rendered {
				return bbs[3].RenderPS(ps67)
			}
			return result
			ctx.FreeDesc(&d60)
			return result
			}
			bbs[2].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
					d134 := ps.PhiValues[0]
					ctx.EnsureDesc(&d134)
					ctx.EmitStoreToStack(d134, int32(bbs[2].PhiBase)+int32(0))
				}
				if bbs[2].VisitCount >= 2 {
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
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
				d14 = ps.OverlayValues[14]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
				d134 = ps.OverlayValues[134]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d3 = ps.PhiValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d3)
			var d135 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(d3.Imm.Int()))))}
			} else {
				r55 := ctx.AllocReg()
				ctx.EmitMovRegReg(r55, d3.Reg)
				ctx.EmitShlRegImm8(r55, 32)
				ctx.EmitShrRegImm8(r55, 32)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
				ctx.BindReg(r55, &d135)
			}
			ctx.EnsureDesc(&d135)
			if thisptr.Loc == scm.LocImm {
				baseReg := ctx.AllocReg()
				if d135.Loc == scm.LocReg {
					ctx.FreeReg(baseReg)
					baseReg = ctx.AllocRegExcept(d135.Reg)
				}
				ctx.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).lastValue)))
				if d135.Loc == scm.LocImm {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d135.Imm.Int()))
					ctx.EmitStoreRegMem(scm.RegR11, baseReg, 0)
				} else {
					ctx.EmitStoreRegMem(d135.Reg, baseReg, 0)
				}
				ctx.FreeReg(baseReg)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
				if d135.Loc == scm.LocImm {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d135.Imm.Int()))
					ctx.EmitStoreRegMem(scm.RegR11, thisptr.Reg, off)
				} else {
					ctx.EmitStoreRegMem(d135.Reg, thisptr.Reg, off)
				}
			}
			ctx.FreeDesc(&d135)
			ctx.EnsureDesc(&d3)
			d136 = d3
			_ = d136
			r56 := d3.Loc == scm.LocReg
			r57 := d3.Reg
			if r56 { ctx.ProtectReg(r57) }
			d137 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(160)}
			lbl22 := ctx.ReserveLabel()
			bbpos_2_0 := int32(-1)
			_ = bbpos_2_0
			bbpos_2_1 := int32(-1)
			_ = bbpos_2_1
			bbpos_2_2 := int32(-1)
			_ = bbpos_2_2
			bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d137 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(160)}
			ctx.EnsureDesc(&d136)
			ctx.EnsureDesc(&d136)
			var d138 scm.JITValueDesc
			if d136.Loc == scm.LocImm {
				d138 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d136.Imm.Int()))))}
			} else {
				r58 := ctx.AllocReg()
				ctx.EmitMovRegReg(r58, d136.Reg)
				ctx.EmitShlRegImm8(r58, 32)
				ctx.EmitShrRegImm8(r58, 32)
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r58}
				ctx.BindReg(r58, &d138)
			}
			var d139 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				r59 := ctx.AllocReg()
				ctx.EmitMovRegMem8(r59, fieldAddr)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r59}
				ctx.BindReg(r59, &d139)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r60 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r60, thisptr.Reg, off)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r60}
				ctx.BindReg(r60, &d139)
			}
			ctx.EnsureDesc(&d139)
			ctx.EnsureDesc(&d139)
			var d140 scm.JITValueDesc
			if d139.Loc == scm.LocImm {
				d140 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d139.Imm.Int()))))}
			} else {
				r61 := ctx.AllocReg()
				ctx.EmitMovRegReg(r61, d139.Reg)
				ctx.EmitShlRegImm8(r61, 56)
				ctx.EmitShrRegImm8(r61, 56)
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r61}
				ctx.BindReg(r61, &d140)
			}
			ctx.EnsureDesc(&d138)
			ctx.EnsureDesc(&d140)
			ctx.EnsureDesc(&d138)
			ctx.EnsureDesc(&d140)
			ctx.EnsureDesc(&d138)
			ctx.EnsureDesc(&d140)
			var d141 scm.JITValueDesc
			if d138.Loc == scm.LocImm && d140.Loc == scm.LocImm {
				d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d138.Imm.Int() * d140.Imm.Int())}
			} else if d138.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d140.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d138.Imm.Int()))
				ctx.EmitImulInt64(scratch, d140.Reg)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d141)
			} else if d140.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d138.Reg)
				ctx.EmitMovRegReg(scratch, d138.Reg)
				if d140.Imm.Int() >= -2147483648 && d140.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d140.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d140.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d141)
			} else {
				r62 := ctx.AllocRegExcept(d138.Reg, d140.Reg)
				ctx.EmitMovRegReg(r62, d138.Reg)
				ctx.EmitImulInt64(r62, d140.Reg)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r62}
				ctx.BindReg(r62, &d141)
			}
			if d141.Loc == scm.LocReg && d138.Loc == scm.LocReg && d141.Reg == d138.Reg {
				ctx.TransferReg(d138.Reg)
				d138.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d138)
			ctx.FreeDesc(&d140)
			var d142 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 0
				r63 := ctx.AllocReg()
				r64 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r63, fieldAddr)
				ctx.EmitMovRegMem64(r64, fieldAddr+8)
				d142 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r63, Reg2: r64}
				ctx.BindReg(r63, &d142)
				ctx.BindReg(r64, &d142)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 0)
				r65 := ctx.AllocReg()
				r66 := ctx.AllocReg()
				ctx.EmitMovRegMem(r65, thisptr.Reg, off)
				ctx.EmitMovRegMem(r66, thisptr.Reg, off+8)
				d142 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r65, Reg2: r66}
				ctx.BindReg(r65, &d142)
				ctx.BindReg(r66, &d142)
			}
			ctx.EnsureDesc(&d141)
			var d143 scm.JITValueDesc
			if d141.Loc == scm.LocImm {
				d143 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d141.Imm.Int() / 64)}
			} else {
				r67 := ctx.AllocRegExcept(d141.Reg)
				ctx.EmitMovRegReg(r67, d141.Reg)
				ctx.EmitShrRegImm8(r67, 6)
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r67}
				ctx.BindReg(r67, &d143)
			}
			if d143.Loc == scm.LocReg && d141.Loc == scm.LocReg && d143.Reg == d141.Reg {
				ctx.TransferReg(d141.Reg)
				d141.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d143)
			r68 := ctx.AllocReg()
			ctx.EnsureDesc(&d143)
			ctx.EnsureDesc(&d142)
			if d143.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r68, uint64(d143.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r68, d143.Reg)
				ctx.EmitShlRegImm8(r68, 3)
			}
			if d142.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d142.Imm.Int()))
				ctx.EmitAddInt64(r68, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r68, d142.Reg)
			}
			r69 := ctx.AllocRegExcept(r68)
			ctx.EmitMovRegMem(r69, r68, 0)
			ctx.FreeReg(r68)
			d144 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r69}
			ctx.BindReg(r69, &d144)
			ctx.FreeDesc(&d143)
			ctx.EnsureDesc(&d141)
			var d145 scm.JITValueDesc
			if d141.Loc == scm.LocImm {
				d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d141.Imm.Int() % 64)}
			} else {
				r70 := ctx.AllocRegExcept(d141.Reg)
				ctx.EmitMovRegReg(r70, d141.Reg)
				ctx.EmitAndRegImm32(r70, 63)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r70}
				ctx.BindReg(r70, &d145)
			}
			if d145.Loc == scm.LocReg && d141.Loc == scm.LocReg && d145.Reg == d141.Reg {
				ctx.TransferReg(d141.Reg)
				d141.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d144)
			ctx.EnsureDesc(&d145)
			var d146 scm.JITValueDesc
			if d144.Loc == scm.LocImm && d145.Loc == scm.LocImm {
				d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d144.Imm.Int()) << uint64(d145.Imm.Int())))}
			} else if d145.Loc == scm.LocImm {
				r71 := ctx.AllocRegExcept(d144.Reg)
				ctx.EmitMovRegReg(r71, d144.Reg)
				ctx.EmitShlRegImm8(r71, uint8(d145.Imm.Int()))
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r71}
				ctx.BindReg(r71, &d146)
			} else {
				{
					shiftSrc := d144.Reg
					r72 := ctx.AllocRegExcept(d144.Reg)
					ctx.EmitMovRegReg(r72, d144.Reg)
					shiftSrc = r72
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d145.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d145.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d145.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d146)
				}
			}
			if d146.Loc == scm.LocReg && d144.Loc == scm.LocReg && d146.Reg == d144.Reg {
				ctx.TransferReg(d144.Reg)
				d144.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d144)
			ctx.FreeDesc(&d145)
			ctx.EnsureDesc(&d141)
			var d147 scm.JITValueDesc
			if d141.Loc == scm.LocImm {
				d147 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d141.Imm.Int() % 64)}
			} else {
				r73 := ctx.AllocRegExcept(d141.Reg)
				ctx.EmitMovRegReg(r73, d141.Reg)
				ctx.EmitAndRegImm32(r73, 63)
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
				ctx.BindReg(r73, &d147)
			}
			if d147.Loc == scm.LocReg && d141.Loc == scm.LocReg && d147.Reg == d141.Reg {
				ctx.TransferReg(d141.Reg)
				d141.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d139)
			ctx.EnsureDesc(&d139)
			var d148 scm.JITValueDesc
			if d139.Loc == scm.LocImm {
				d148 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d139.Imm.Int()))))}
			} else {
				r74 := ctx.AllocReg()
				ctx.EmitMovRegReg(r74, d139.Reg)
				ctx.EmitShlRegImm8(r74, 56)
				ctx.EmitShrRegImm8(r74, 56)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r74}
				ctx.BindReg(r74, &d148)
			}
			ctx.EnsureDesc(&d147)
			ctx.EnsureDesc(&d148)
			ctx.EnsureDesc(&d147)
			ctx.EnsureDesc(&d148)
			ctx.EnsureDesc(&d147)
			ctx.EnsureDesc(&d148)
			var d149 scm.JITValueDesc
			if d147.Loc == scm.LocImm && d148.Loc == scm.LocImm {
				d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d147.Imm.Int() + d148.Imm.Int())}
			} else if d148.Loc == scm.LocImm && d148.Imm.Int() == 0 {
				r75 := ctx.AllocRegExcept(d147.Reg)
				ctx.EmitMovRegReg(r75, d147.Reg)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
				ctx.BindReg(r75, &d149)
			} else if d147.Loc == scm.LocImm && d147.Imm.Int() == 0 {
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d148.Reg}
				ctx.BindReg(d148.Reg, &d149)
			} else if d147.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d148.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d147.Imm.Int()))
				ctx.EmitAddInt64(scratch, d148.Reg)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d149)
			} else if d148.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d147.Reg)
				ctx.EmitMovRegReg(scratch, d147.Reg)
				if d148.Imm.Int() >= -2147483648 && d148.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d148.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d148.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d149)
			} else {
				r76 := ctx.AllocRegExcept(d147.Reg, d148.Reg)
				ctx.EmitMovRegReg(r76, d147.Reg)
				ctx.EmitAddInt64(r76, d148.Reg)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r76}
				ctx.BindReg(r76, &d149)
			}
			if d149.Loc == scm.LocReg && d147.Loc == scm.LocReg && d149.Reg == d147.Reg {
				ctx.TransferReg(d147.Reg)
				d147.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d147)
			ctx.FreeDesc(&d148)
			ctx.EnsureDesc(&d149)
			var d150 scm.JITValueDesc
			if d149.Loc == scm.LocImm {
				d150 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d149.Imm.Int()) > uint64(64))}
			} else {
				r77 := ctx.AllocRegExcept(d149.Reg)
				ctx.EmitCmpRegImm32(d149.Reg, 64)
				ctx.EmitSetcc(r77, scm.CcA)
				d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r77}
				ctx.BindReg(r77, &d150)
			}
			ctx.FreeDesc(&d149)
			d151 = d150
			ctx.EnsureDesc(&d151)
			if d151.Loc != scm.LocImm && d151.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl23 := ctx.ReserveLabel()
			lbl24 := ctx.ReserveLabel()
			lbl25 := ctx.ReserveLabel()
			lbl26 := ctx.ReserveLabel()
			if d151.Loc == scm.LocImm {
				if d151.Imm.Bool() {
					ctx.MarkLabel(lbl25)
					ctx.EmitJmp(lbl23)
				} else {
					ctx.MarkLabel(lbl26)
			ctx.EnsureDesc(&d146)
			if d146.Loc == scm.LocReg {
				ctx.ProtectReg(d146.Reg)
			} else if d146.Loc == scm.LocRegPair {
				ctx.ProtectReg(d146.Reg)
				ctx.ProtectReg(d146.Reg2)
			}
			d152 = d146
			if d152.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d152)
			ctx.EmitStoreToStack(d152, int32(bbs[2].PhiBase)+int32(0))
			if d146.Loc == scm.LocReg {
				ctx.UnprotectReg(d146.Reg)
			} else if d146.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d146.Reg)
				ctx.UnprotectReg(d146.Reg2)
			}
					ctx.EmitJmp(lbl24)
				}
			} else {
				ctx.EmitCmpRegImm32(d151.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl25)
				ctx.EmitJmp(lbl26)
				ctx.MarkLabel(lbl25)
				ctx.EmitJmp(lbl23)
				ctx.MarkLabel(lbl26)
			ctx.EnsureDesc(&d146)
			if d146.Loc == scm.LocReg {
				ctx.ProtectReg(d146.Reg)
			} else if d146.Loc == scm.LocRegPair {
				ctx.ProtectReg(d146.Reg)
				ctx.ProtectReg(d146.Reg2)
			}
			d153 = d146
			if d153.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d153)
			ctx.EmitStoreToStack(d153, int32(bbs[2].PhiBase)+int32(0))
			if d146.Loc == scm.LocReg {
				ctx.UnprotectReg(d146.Reg)
			} else if d146.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d146.Reg)
				ctx.UnprotectReg(d146.Reg2)
			}
				ctx.EmitJmp(lbl24)
			}
			ctx.FreeDesc(&d150)
			bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl24)
			ctx.ResolveFixups()
			d137 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(160)}
			ctx.EnsureDesc(&d139)
			ctx.EnsureDesc(&d139)
			var d154 scm.JITValueDesc
			if d139.Loc == scm.LocImm {
				d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d139.Imm.Int()))))}
			} else {
				r78 := ctx.AllocReg()
				ctx.EmitMovRegReg(r78, d139.Reg)
				ctx.EmitShlRegImm8(r78, 56)
				ctx.EmitShrRegImm8(r78, 56)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r78}
				ctx.BindReg(r78, &d154)
			}
			d155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d154)
			ctx.EnsureDesc(&d155)
			ctx.EnsureDesc(&d154)
			ctx.EnsureDesc(&d155)
			ctx.EnsureDesc(&d154)
			var d156 scm.JITValueDesc
			if d155.Loc == scm.LocImm && d154.Loc == scm.LocImm {
				d156 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d155.Imm.Int() - d154.Imm.Int())}
			} else if d154.Loc == scm.LocImm && d154.Imm.Int() == 0 {
				r79 := ctx.AllocRegExcept(d155.Reg)
				ctx.EmitMovRegReg(r79, d155.Reg)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r79}
				ctx.BindReg(r79, &d156)
			} else if d155.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d154.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d155.Imm.Int()))
				ctx.EmitSubInt64(scratch, d154.Reg)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d156)
			} else if d154.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d155.Reg)
				ctx.EmitMovRegReg(scratch, d155.Reg)
				if d154.Imm.Int() >= -2147483648 && d154.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d154.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d154.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d156)
			} else {
				r80 := ctx.AllocRegExcept(d155.Reg, d154.Reg)
				ctx.EmitMovRegReg(r80, d155.Reg)
				ctx.EmitSubInt64(r80, d154.Reg)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
				ctx.BindReg(r80, &d156)
			}
			if d156.Loc == scm.LocReg && d155.Loc == scm.LocReg && d156.Reg == d155.Reg {
				ctx.TransferReg(d155.Reg)
				d155.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d154)
			ctx.EnsureDesc(&d137)
			ctx.EnsureDesc(&d156)
			var d157 scm.JITValueDesc
			if d137.Loc == scm.LocImm && d156.Loc == scm.LocImm {
				d157 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d137.Imm.Int()) >> uint64(d156.Imm.Int())))}
			} else if d156.Loc == scm.LocImm {
				r81 := ctx.AllocRegExcept(d137.Reg)
				ctx.EmitMovRegReg(r81, d137.Reg)
				ctx.EmitShrRegImm8(r81, uint8(d156.Imm.Int()))
				d157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
				ctx.BindReg(r81, &d157)
			} else {
				{
					shiftSrc := d137.Reg
					r82 := ctx.AllocRegExcept(d137.Reg)
					ctx.EmitMovRegReg(r82, d137.Reg)
					shiftSrc = r82
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d156.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d156.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d156.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d157)
				}
			}
			if d157.Loc == scm.LocReg && d137.Loc == scm.LocReg && d157.Reg == d137.Reg {
				ctx.TransferReg(d137.Reg)
				d137.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d137)
			ctx.FreeDesc(&d156)
			r83 := ctx.AllocReg()
			ctx.EnsureDesc(&d157)
			ctx.EnsureDesc(&d157)
			if d157.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r83, d157)
			}
			ctx.EmitJmp(lbl22)
			bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl23)
			ctx.ResolveFixups()
			d137 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(160)}
			ctx.EnsureDesc(&d141)
			var d158 scm.JITValueDesc
			if d141.Loc == scm.LocImm {
				d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d141.Imm.Int() / 64)}
			} else {
				r84 := ctx.AllocRegExcept(d141.Reg)
				ctx.EmitMovRegReg(r84, d141.Reg)
				ctx.EmitShrRegImm8(r84, 6)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r84}
				ctx.BindReg(r84, &d158)
			}
			if d158.Loc == scm.LocReg && d141.Loc == scm.LocReg && d158.Reg == d141.Reg {
				ctx.TransferReg(d141.Reg)
				d141.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d158)
			ctx.EnsureDesc(&d158)
			var d159 scm.JITValueDesc
			if d158.Loc == scm.LocImm {
				d159 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d158.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d158.Reg)
				ctx.EmitMovRegReg(scratch, d158.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d159)
			}
			if d159.Loc == scm.LocReg && d158.Loc == scm.LocReg && d159.Reg == d158.Reg {
				ctx.TransferReg(d158.Reg)
				d158.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d158)
			ctx.EnsureDesc(&d159)
			r85 := ctx.AllocReg()
			ctx.EnsureDesc(&d159)
			ctx.EnsureDesc(&d142)
			if d159.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r85, uint64(d159.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r85, d159.Reg)
				ctx.EmitShlRegImm8(r85, 3)
			}
			if d142.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d142.Imm.Int()))
				ctx.EmitAddInt64(r85, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r85, d142.Reg)
			}
			r86 := ctx.AllocRegExcept(r85)
			ctx.EmitMovRegMem(r86, r85, 0)
			ctx.FreeReg(r85)
			d160 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r86}
			ctx.BindReg(r86, &d160)
			ctx.FreeDesc(&d159)
			ctx.EnsureDesc(&d141)
			var d161 scm.JITValueDesc
			if d141.Loc == scm.LocImm {
				d161 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d141.Imm.Int() % 64)}
			} else {
				r87 := ctx.AllocRegExcept(d141.Reg)
				ctx.EmitMovRegReg(r87, d141.Reg)
				ctx.EmitAndRegImm32(r87, 63)
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r87}
				ctx.BindReg(r87, &d161)
			}
			if d161.Loc == scm.LocReg && d141.Loc == scm.LocReg && d161.Reg == d141.Reg {
				ctx.TransferReg(d141.Reg)
				d141.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d141)
			d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d161)
			ctx.EnsureDesc(&d162)
			ctx.EnsureDesc(&d161)
			ctx.EnsureDesc(&d162)
			ctx.EnsureDesc(&d161)
			var d163 scm.JITValueDesc
			if d162.Loc == scm.LocImm && d161.Loc == scm.LocImm {
				d163 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d162.Imm.Int() - d161.Imm.Int())}
			} else if d161.Loc == scm.LocImm && d161.Imm.Int() == 0 {
				r88 := ctx.AllocRegExcept(d162.Reg)
				ctx.EmitMovRegReg(r88, d162.Reg)
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
				ctx.BindReg(r88, &d163)
			} else if d162.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d161.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d162.Imm.Int()))
				ctx.EmitSubInt64(scratch, d161.Reg)
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d163)
			} else if d161.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d162.Reg)
				ctx.EmitMovRegReg(scratch, d162.Reg)
				if d161.Imm.Int() >= -2147483648 && d161.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d161.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d161.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d163)
			} else {
				r89 := ctx.AllocRegExcept(d162.Reg, d161.Reg)
				ctx.EmitMovRegReg(r89, d162.Reg)
				ctx.EmitSubInt64(r89, d161.Reg)
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
				ctx.BindReg(r89, &d163)
			}
			if d163.Loc == scm.LocReg && d162.Loc == scm.LocReg && d163.Reg == d162.Reg {
				ctx.TransferReg(d162.Reg)
				d162.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d161)
			ctx.EnsureDesc(&d160)
			ctx.EnsureDesc(&d163)
			var d164 scm.JITValueDesc
			if d160.Loc == scm.LocImm && d163.Loc == scm.LocImm {
				d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d160.Imm.Int()) >> uint64(d163.Imm.Int())))}
			} else if d163.Loc == scm.LocImm {
				r90 := ctx.AllocRegExcept(d160.Reg)
				ctx.EmitMovRegReg(r90, d160.Reg)
				ctx.EmitShrRegImm8(r90, uint8(d163.Imm.Int()))
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
				ctx.BindReg(r90, &d164)
			} else {
				{
					shiftSrc := d160.Reg
					r91 := ctx.AllocRegExcept(d160.Reg)
					ctx.EmitMovRegReg(r91, d160.Reg)
					shiftSrc = r91
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d163.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d163.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d163.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d164)
				}
			}
			if d164.Loc == scm.LocReg && d160.Loc == scm.LocReg && d164.Reg == d160.Reg {
				ctx.TransferReg(d160.Reg)
				d160.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d160)
			ctx.FreeDesc(&d163)
			ctx.EnsureDesc(&d146)
			ctx.EnsureDesc(&d164)
			var d165 scm.JITValueDesc
			if d146.Loc == scm.LocImm && d164.Loc == scm.LocImm {
				d165 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d146.Imm.Int() | d164.Imm.Int())}
			} else if d146.Loc == scm.LocImm && d146.Imm.Int() == 0 {
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d164.Reg}
				ctx.BindReg(d164.Reg, &d165)
			} else if d164.Loc == scm.LocImm && d164.Imm.Int() == 0 {
				r92 := ctx.AllocRegExcept(d146.Reg)
				ctx.EmitMovRegReg(r92, d146.Reg)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r92}
				ctx.BindReg(r92, &d165)
			} else if d146.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d164.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d146.Imm.Int()))
				ctx.EmitOrInt64(scratch, d164.Reg)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d165)
			} else if d164.Loc == scm.LocImm {
				r93 := ctx.AllocRegExcept(d146.Reg)
				ctx.EmitMovRegReg(r93, d146.Reg)
				if d164.Imm.Int() >= -2147483648 && d164.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r93, int32(d164.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d164.Imm.Int()))
					ctx.EmitOrInt64(r93, scm.RegR11)
				}
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r93}
				ctx.BindReg(r93, &d165)
			} else {
				r94 := ctx.AllocRegExcept(d146.Reg, d164.Reg)
				ctx.EmitMovRegReg(r94, d146.Reg)
				ctx.EmitOrInt64(r94, d164.Reg)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
				ctx.BindReg(r94, &d165)
			}
			if d165.Loc == scm.LocReg && d146.Loc == scm.LocReg && d165.Reg == d146.Reg {
				ctx.TransferReg(d146.Reg)
				d146.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d164)
			ctx.EnsureDesc(&d165)
			if d165.Loc == scm.LocReg {
				ctx.ProtectReg(d165.Reg)
			} else if d165.Loc == scm.LocRegPair {
				ctx.ProtectReg(d165.Reg)
				ctx.ProtectReg(d165.Reg2)
			}
			d166 = d165
			if d166.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d166)
			ctx.EmitStoreToStack(d166, int32(bbs[2].PhiBase)+int32(0))
			if d165.Loc == scm.LocReg {
				ctx.UnprotectReg(d165.Reg)
			} else if d165.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d165.Reg)
				ctx.UnprotectReg(d165.Reg2)
			}
			ctx.EmitJmp(lbl24)
			ctx.MarkLabel(lbl22)
			d167 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r83}
			ctx.BindReg(r83, &d167)
			ctx.BindReg(r83, &d167)
			if r56 { ctx.UnprotectReg(r57) }
			ctx.EnsureDesc(&d167)
			ctx.EnsureDesc(&d167)
			var d168 scm.JITValueDesc
			if d167.Loc == scm.LocImm {
				d168 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d167.Imm.Int()))))}
			} else {
				r95 := ctx.AllocReg()
				ctx.EmitMovRegReg(r95, d167.Reg)
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r95}
				ctx.BindReg(r95, &d168)
			}
			ctx.FreeDesc(&d167)
			var d169 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 32
				r96 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r96, fieldAddr)
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r96}
				ctx.BindReg(r96, &d169)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 32)
				r97 := ctx.AllocReg()
				ctx.EmitMovRegMem(r97, thisptr.Reg, off)
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r97}
				ctx.BindReg(r97, &d169)
			}
			ctx.EnsureDesc(&d168)
			ctx.EnsureDesc(&d169)
			ctx.EnsureDesc(&d168)
			ctx.EnsureDesc(&d169)
			ctx.EnsureDesc(&d168)
			ctx.EnsureDesc(&d169)
			var d170 scm.JITValueDesc
			if d168.Loc == scm.LocImm && d169.Loc == scm.LocImm {
				d170 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d168.Imm.Int() + d169.Imm.Int())}
			} else if d169.Loc == scm.LocImm && d169.Imm.Int() == 0 {
				r98 := ctx.AllocRegExcept(d168.Reg)
				ctx.EmitMovRegReg(r98, d168.Reg)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r98}
				ctx.BindReg(r98, &d170)
			} else if d168.Loc == scm.LocImm && d168.Imm.Int() == 0 {
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d169.Reg}
				ctx.BindReg(d169.Reg, &d170)
			} else if d168.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d169.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d168.Imm.Int()))
				ctx.EmitAddInt64(scratch, d169.Reg)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d170)
			} else if d169.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d168.Reg)
				ctx.EmitMovRegReg(scratch, d168.Reg)
				if d169.Imm.Int() >= -2147483648 && d169.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d169.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d169.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d170)
			} else {
				r99 := ctx.AllocRegExcept(d168.Reg, d169.Reg)
				ctx.EmitMovRegReg(r99, d168.Reg)
				ctx.EmitAddInt64(r99, d169.Reg)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
				ctx.BindReg(r99, &d170)
			}
			if d170.Loc == scm.LocReg && d168.Loc == scm.LocReg && d170.Reg == d168.Reg {
				ctx.TransferReg(d168.Reg)
				d168.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d168)
			var d171 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 56
				r100 := ctx.AllocReg()
				ctx.EmitMovRegMem8(r100, fieldAddr)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r100}
				ctx.BindReg(r100, &d171)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 56)
				r101 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r101, thisptr.Reg, off)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r101}
				ctx.BindReg(r101, &d171)
			}
			d172 = d171
			ctx.EnsureDesc(&d172)
			if d172.Loc != scm.LocImm && d172.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d172.Loc == scm.LocImm {
				if d172.Imm.Bool() {
			ps173 := scm.PhiState{General: ps.General}
			ps173.OverlayValues = make([]scm.JITValueDesc, 173)
			ps173.OverlayValues[0] = d0
			ps173.OverlayValues[1] = d1
			ps173.OverlayValues[2] = d2
			ps173.OverlayValues[3] = d3
			ps173.OverlayValues[4] = d4
			ps173.OverlayValues[5] = d5
			ps173.OverlayValues[6] = d6
			ps173.OverlayValues[7] = d7
			ps173.OverlayValues[8] = d8
			ps173.OverlayValues[9] = d9
			ps173.OverlayValues[10] = d10
			ps173.OverlayValues[11] = d11
			ps173.OverlayValues[12] = d12
			ps173.OverlayValues[13] = d13
			ps173.OverlayValues[14] = d14
			ps173.OverlayValues[15] = d15
			ps173.OverlayValues[16] = d16
			ps173.OverlayValues[18] = d18
			ps173.OverlayValues[19] = d19
			ps173.OverlayValues[20] = d20
			ps173.OverlayValues[21] = d21
			ps173.OverlayValues[22] = d22
			ps173.OverlayValues[23] = d23
			ps173.OverlayValues[24] = d24
			ps173.OverlayValues[25] = d25
			ps173.OverlayValues[26] = d26
			ps173.OverlayValues[27] = d27
			ps173.OverlayValues[28] = d28
			ps173.OverlayValues[29] = d29
			ps173.OverlayValues[30] = d30
			ps173.OverlayValues[31] = d31
			ps173.OverlayValues[32] = d32
			ps173.OverlayValues[33] = d33
			ps173.OverlayValues[34] = d34
			ps173.OverlayValues[35] = d35
			ps173.OverlayValues[36] = d36
			ps173.OverlayValues[37] = d37
			ps173.OverlayValues[38] = d38
			ps173.OverlayValues[39] = d39
			ps173.OverlayValues[40] = d40
			ps173.OverlayValues[41] = d41
			ps173.OverlayValues[42] = d42
			ps173.OverlayValues[43] = d43
			ps173.OverlayValues[44] = d44
			ps173.OverlayValues[45] = d45
			ps173.OverlayValues[46] = d46
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
			ps173.OverlayValues[64] = d64
			ps173.OverlayValues[65] = d65
			ps173.OverlayValues[66] = d66
			ps173.OverlayValues[134] = d134
			ps173.OverlayValues[135] = d135
			ps173.OverlayValues[136] = d136
			ps173.OverlayValues[137] = d137
			ps173.OverlayValues[138] = d138
			ps173.OverlayValues[139] = d139
			ps173.OverlayValues[140] = d140
			ps173.OverlayValues[141] = d141
			ps173.OverlayValues[142] = d142
			ps173.OverlayValues[143] = d143
			ps173.OverlayValues[144] = d144
			ps173.OverlayValues[145] = d145
			ps173.OverlayValues[146] = d146
			ps173.OverlayValues[147] = d147
			ps173.OverlayValues[148] = d148
			ps173.OverlayValues[149] = d149
			ps173.OverlayValues[150] = d150
			ps173.OverlayValues[151] = d151
			ps173.OverlayValues[152] = d152
			ps173.OverlayValues[153] = d153
			ps173.OverlayValues[154] = d154
			ps173.OverlayValues[155] = d155
			ps173.OverlayValues[156] = d156
			ps173.OverlayValues[157] = d157
			ps173.OverlayValues[158] = d158
			ps173.OverlayValues[159] = d159
			ps173.OverlayValues[160] = d160
			ps173.OverlayValues[161] = d161
			ps173.OverlayValues[162] = d162
			ps173.OverlayValues[163] = d163
			ps173.OverlayValues[164] = d164
			ps173.OverlayValues[165] = d165
			ps173.OverlayValues[166] = d166
			ps173.OverlayValues[167] = d167
			ps173.OverlayValues[168] = d168
			ps173.OverlayValues[169] = d169
			ps173.OverlayValues[170] = d170
			ps173.OverlayValues[171] = d171
			ps173.OverlayValues[172] = d172
					return bbs[13].RenderPS(ps173)
				}
			ps174 := scm.PhiState{General: ps.General}
			ps174.OverlayValues = make([]scm.JITValueDesc, 173)
			ps174.OverlayValues[0] = d0
			ps174.OverlayValues[1] = d1
			ps174.OverlayValues[2] = d2
			ps174.OverlayValues[3] = d3
			ps174.OverlayValues[4] = d4
			ps174.OverlayValues[5] = d5
			ps174.OverlayValues[6] = d6
			ps174.OverlayValues[7] = d7
			ps174.OverlayValues[8] = d8
			ps174.OverlayValues[9] = d9
			ps174.OverlayValues[10] = d10
			ps174.OverlayValues[11] = d11
			ps174.OverlayValues[12] = d12
			ps174.OverlayValues[13] = d13
			ps174.OverlayValues[14] = d14
			ps174.OverlayValues[15] = d15
			ps174.OverlayValues[16] = d16
			ps174.OverlayValues[18] = d18
			ps174.OverlayValues[19] = d19
			ps174.OverlayValues[20] = d20
			ps174.OverlayValues[21] = d21
			ps174.OverlayValues[22] = d22
			ps174.OverlayValues[23] = d23
			ps174.OverlayValues[24] = d24
			ps174.OverlayValues[25] = d25
			ps174.OverlayValues[26] = d26
			ps174.OverlayValues[27] = d27
			ps174.OverlayValues[28] = d28
			ps174.OverlayValues[29] = d29
			ps174.OverlayValues[30] = d30
			ps174.OverlayValues[31] = d31
			ps174.OverlayValues[32] = d32
			ps174.OverlayValues[33] = d33
			ps174.OverlayValues[34] = d34
			ps174.OverlayValues[35] = d35
			ps174.OverlayValues[36] = d36
			ps174.OverlayValues[37] = d37
			ps174.OverlayValues[38] = d38
			ps174.OverlayValues[39] = d39
			ps174.OverlayValues[40] = d40
			ps174.OverlayValues[41] = d41
			ps174.OverlayValues[42] = d42
			ps174.OverlayValues[43] = d43
			ps174.OverlayValues[44] = d44
			ps174.OverlayValues[45] = d45
			ps174.OverlayValues[46] = d46
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
			ps174.OverlayValues[64] = d64
			ps174.OverlayValues[65] = d65
			ps174.OverlayValues[66] = d66
			ps174.OverlayValues[134] = d134
			ps174.OverlayValues[135] = d135
			ps174.OverlayValues[136] = d136
			ps174.OverlayValues[137] = d137
			ps174.OverlayValues[138] = d138
			ps174.OverlayValues[139] = d139
			ps174.OverlayValues[140] = d140
			ps174.OverlayValues[141] = d141
			ps174.OverlayValues[142] = d142
			ps174.OverlayValues[143] = d143
			ps174.OverlayValues[144] = d144
			ps174.OverlayValues[145] = d145
			ps174.OverlayValues[146] = d146
			ps174.OverlayValues[147] = d147
			ps174.OverlayValues[148] = d148
			ps174.OverlayValues[149] = d149
			ps174.OverlayValues[150] = d150
			ps174.OverlayValues[151] = d151
			ps174.OverlayValues[152] = d152
			ps174.OverlayValues[153] = d153
			ps174.OverlayValues[154] = d154
			ps174.OverlayValues[155] = d155
			ps174.OverlayValues[156] = d156
			ps174.OverlayValues[157] = d157
			ps174.OverlayValues[158] = d158
			ps174.OverlayValues[159] = d159
			ps174.OverlayValues[160] = d160
			ps174.OverlayValues[161] = d161
			ps174.OverlayValues[162] = d162
			ps174.OverlayValues[163] = d163
			ps174.OverlayValues[164] = d164
			ps174.OverlayValues[165] = d165
			ps174.OverlayValues[166] = d166
			ps174.OverlayValues[167] = d167
			ps174.OverlayValues[168] = d168
			ps174.OverlayValues[169] = d169
			ps174.OverlayValues[170] = d170
			ps174.OverlayValues[171] = d171
			ps174.OverlayValues[172] = d172
				return bbs[12].RenderPS(ps174)
			}
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
					d175 := ps.PhiValues[0]
					ctx.EnsureDesc(&d175)
					ctx.EmitStoreToStack(d175, int32(bbs[2].PhiBase)+int32(0))
				}
				ps.General = true
				return bbs[2].RenderPS(ps)
			}
			lbl27 := ctx.ReserveLabel()
			lbl28 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d172.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl27)
			ctx.EmitJmp(lbl28)
			ctx.MarkLabel(lbl27)
			ctx.EmitJmp(lbl14)
			ctx.MarkLabel(lbl28)
			ctx.EmitJmp(lbl13)
			ps176 := scm.PhiState{General: true}
			ps176.OverlayValues = make([]scm.JITValueDesc, 176)
			ps176.OverlayValues[0] = d0
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
			ps176.OverlayValues[18] = d18
			ps176.OverlayValues[19] = d19
			ps176.OverlayValues[20] = d20
			ps176.OverlayValues[21] = d21
			ps176.OverlayValues[22] = d22
			ps176.OverlayValues[23] = d23
			ps176.OverlayValues[24] = d24
			ps176.OverlayValues[25] = d25
			ps176.OverlayValues[26] = d26
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
			ps176.OverlayValues[64] = d64
			ps176.OverlayValues[65] = d65
			ps176.OverlayValues[66] = d66
			ps176.OverlayValues[134] = d134
			ps176.OverlayValues[135] = d135
			ps176.OverlayValues[136] = d136
			ps176.OverlayValues[137] = d137
			ps176.OverlayValues[138] = d138
			ps176.OverlayValues[139] = d139
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
			ps176.OverlayValues[175] = d175
			ps177 := scm.PhiState{General: true}
			ps177.OverlayValues = make([]scm.JITValueDesc, 176)
			ps177.OverlayValues[0] = d0
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
			ps177.OverlayValues[18] = d18
			ps177.OverlayValues[19] = d19
			ps177.OverlayValues[20] = d20
			ps177.OverlayValues[21] = d21
			ps177.OverlayValues[22] = d22
			ps177.OverlayValues[23] = d23
			ps177.OverlayValues[24] = d24
			ps177.OverlayValues[25] = d25
			ps177.OverlayValues[26] = d26
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
			ps177.OverlayValues[64] = d64
			ps177.OverlayValues[65] = d65
			ps177.OverlayValues[66] = d66
			ps177.OverlayValues[134] = d134
			ps177.OverlayValues[135] = d135
			ps177.OverlayValues[136] = d136
			ps177.OverlayValues[137] = d137
			ps177.OverlayValues[138] = d138
			ps177.OverlayValues[139] = d139
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
			ps177.OverlayValues[175] = d175
			snap178 := d0
			snap179 := d1
			snap180 := d2
			snap181 := d3
			snap182 := d4
			snap183 := d5
			snap184 := d6
			snap185 := d7
			snap186 := d8
			snap187 := d9
			snap188 := d10
			snap189 := d11
			snap190 := d12
			snap191 := d13
			snap192 := d14
			snap193 := d15
			snap194 := d16
			snap195 := d18
			snap196 := d19
			snap197 := d20
			snap198 := d21
			snap199 := d22
			snap200 := d23
			snap201 := d24
			snap202 := d25
			snap203 := d26
			snap204 := d27
			snap205 := d28
			snap206 := d29
			snap207 := d30
			snap208 := d31
			snap209 := d32
			snap210 := d33
			snap211 := d34
			snap212 := d35
			snap213 := d36
			snap214 := d37
			snap215 := d38
			snap216 := d39
			snap217 := d40
			snap218 := d41
			snap219 := d42
			snap220 := d43
			snap221 := d44
			snap222 := d45
			snap223 := d46
			snap224 := d47
			snap225 := d48
			snap226 := d49
			snap227 := d50
			snap228 := d51
			snap229 := d52
			snap230 := d53
			snap231 := d54
			snap232 := d55
			snap233 := d56
			snap234 := d57
			snap235 := d58
			snap236 := d59
			snap237 := d60
			snap238 := d61
			snap239 := d64
			snap240 := d65
			snap241 := d66
			snap242 := d134
			snap243 := d135
			snap244 := d136
			snap245 := d137
			snap246 := d138
			snap247 := d139
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
			snap281 := d175
			alloc282 := ctx.SnapshotAllocState()
			if !bbs[12].Rendered {
				bbs[12].RenderPS(ps177)
			}
			ctx.RestoreAllocState(alloc282)
			d0 = snap178
			d1 = snap179
			d2 = snap180
			d3 = snap181
			d4 = snap182
			d5 = snap183
			d6 = snap184
			d7 = snap185
			d8 = snap186
			d9 = snap187
			d10 = snap188
			d11 = snap189
			d12 = snap190
			d13 = snap191
			d14 = snap192
			d15 = snap193
			d16 = snap194
			d18 = snap195
			d19 = snap196
			d20 = snap197
			d21 = snap198
			d22 = snap199
			d23 = snap200
			d24 = snap201
			d25 = snap202
			d26 = snap203
			d27 = snap204
			d28 = snap205
			d29 = snap206
			d30 = snap207
			d31 = snap208
			d32 = snap209
			d33 = snap210
			d34 = snap211
			d35 = snap212
			d36 = snap213
			d37 = snap214
			d38 = snap215
			d39 = snap216
			d40 = snap217
			d41 = snap218
			d42 = snap219
			d43 = snap220
			d44 = snap221
			d45 = snap222
			d46 = snap223
			d47 = snap224
			d48 = snap225
			d49 = snap226
			d50 = snap227
			d51 = snap228
			d52 = snap229
			d53 = snap230
			d54 = snap231
			d55 = snap232
			d56 = snap233
			d57 = snap234
			d58 = snap235
			d59 = snap236
			d60 = snap237
			d61 = snap238
			d64 = snap239
			d65 = snap240
			d66 = snap241
			d134 = snap242
			d135 = snap243
			d136 = snap244
			d137 = snap245
			d138 = snap246
			d139 = snap247
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
			d175 = snap281
			if !bbs[13].Rendered {
				return bbs[13].RenderPS(ps176)
			}
			return result
			return result
			}
			bbs[3].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[3].VisitCount >= 2 {
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
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
				d14 = ps.OverlayValues[14]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
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
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			var d283 scm.JITValueDesc
			if d0.Loc == scm.LocImm {
				d283 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d0.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d0.Reg)
				ctx.EmitMovRegReg(scratch, d0.Reg)
				ctx.EmitSubRegImm32(scratch, int32(1))
				d283 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d283)
			}
			if d283.Loc == scm.LocImm {
				d283 = scm.JITValueDesc{Loc: scm.LocImm, Type: d283.Type, Imm: scm.NewInt(int64(uint64(d283.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d283.Reg, 32)
				ctx.EmitShrRegImm8(d283.Reg, 32)
			}
			if d283.Loc == scm.LocReg && d0.Loc == scm.LocReg && d283.Reg == d0.Reg {
				ctx.TransferReg(d0.Reg)
				d0.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			var d284 scm.JITValueDesc
			if d0.Loc == scm.LocImm {
				d284 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d0.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d0.Reg)
				ctx.EmitMovRegReg(scratch, d0.Reg)
				ctx.EmitSubRegImm32(scratch, int32(1))
				d284 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d284)
			}
			if d284.Loc == scm.LocImm {
				d284 = scm.JITValueDesc{Loc: scm.LocImm, Type: d284.Type, Imm: scm.NewInt(int64(uint64(d284.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d284.Reg, 32)
				ctx.EmitShrRegImm8(d284.Reg, 32)
			}
			if d284.Loc == scm.LocReg && d0.Loc == scm.LocReg && d284.Reg == d0.Reg {
				ctx.TransferReg(d0.Reg)
				d0.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d1)
			if d1.Loc == scm.LocReg {
				ctx.ProtectReg(d1.Reg)
			} else if d1.Loc == scm.LocRegPair {
				ctx.ProtectReg(d1.Reg)
				ctx.ProtectReg(d1.Reg2)
			}
			ctx.EnsureDesc(&d283)
			if d283.Loc == scm.LocReg {
				ctx.ProtectReg(d283.Reg)
			} else if d283.Loc == scm.LocRegPair {
				ctx.ProtectReg(d283.Reg)
				ctx.ProtectReg(d283.Reg2)
			}
			ctx.EnsureDesc(&d284)
			if d284.Loc == scm.LocReg {
				ctx.ProtectReg(d284.Reg)
			} else if d284.Loc == scm.LocRegPair {
				ctx.ProtectReg(d284.Reg)
				ctx.ProtectReg(d284.Reg2)
			}
			d285 = d284
			if d285.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d285)
			d286 = d285
			if d286.Loc == scm.LocImm {
				d286 = scm.JITValueDesc{Loc: scm.LocImm, Type: d286.Type, Imm: scm.NewInt(int64(uint64(d286.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d286.Reg, 32)
				ctx.EmitShrRegImm8(d286.Reg, 32)
			}
			ctx.EmitStoreToStack(d286, int32(bbs[4].PhiBase)+int32(0))
			d287 = d1
			if d287.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d287)
			d288 = d287
			if d288.Loc == scm.LocImm {
				d288 = scm.JITValueDesc{Loc: scm.LocImm, Type: d288.Type, Imm: scm.NewInt(int64(uint64(d288.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d288.Reg, 32)
				ctx.EmitShrRegImm8(d288.Reg, 32)
			}
			ctx.EmitStoreToStack(d288, int32(bbs[4].PhiBase)+int32(16))
			d289 = d283
			if d289.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d289)
			d290 = d289
			if d290.Loc == scm.LocImm {
				d290 = scm.JITValueDesc{Loc: scm.LocImm, Type: d290.Type, Imm: scm.NewInt(int64(uint64(d290.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d290.Reg, 32)
				ctx.EmitShrRegImm8(d290.Reg, 32)
			}
			ctx.EmitStoreToStack(d290, int32(bbs[4].PhiBase)+int32(32))
			if d1.Loc == scm.LocReg {
				ctx.UnprotectReg(d1.Reg)
			} else if d1.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d1.Reg)
				ctx.UnprotectReg(d1.Reg2)
			}
			if d283.Loc == scm.LocReg {
				ctx.UnprotectReg(d283.Reg)
			} else if d283.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d283.Reg)
				ctx.UnprotectReg(d283.Reg2)
			}
			if d284.Loc == scm.LocReg {
				ctx.UnprotectReg(d284.Reg)
			} else if d284.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d284.Reg)
				ctx.UnprotectReg(d284.Reg2)
			}
			ps291 := scm.PhiState{General: ps.General}
			ps291.OverlayValues = make([]scm.JITValueDesc, 291)
			ps291.OverlayValues[0] = d0
			ps291.OverlayValues[1] = d1
			ps291.OverlayValues[2] = d2
			ps291.OverlayValues[3] = d3
			ps291.OverlayValues[4] = d4
			ps291.OverlayValues[5] = d5
			ps291.OverlayValues[6] = d6
			ps291.OverlayValues[7] = d7
			ps291.OverlayValues[8] = d8
			ps291.OverlayValues[9] = d9
			ps291.OverlayValues[10] = d10
			ps291.OverlayValues[11] = d11
			ps291.OverlayValues[12] = d12
			ps291.OverlayValues[13] = d13
			ps291.OverlayValues[14] = d14
			ps291.OverlayValues[15] = d15
			ps291.OverlayValues[16] = d16
			ps291.OverlayValues[18] = d18
			ps291.OverlayValues[19] = d19
			ps291.OverlayValues[20] = d20
			ps291.OverlayValues[21] = d21
			ps291.OverlayValues[22] = d22
			ps291.OverlayValues[23] = d23
			ps291.OverlayValues[24] = d24
			ps291.OverlayValues[25] = d25
			ps291.OverlayValues[26] = d26
			ps291.OverlayValues[27] = d27
			ps291.OverlayValues[28] = d28
			ps291.OverlayValues[29] = d29
			ps291.OverlayValues[30] = d30
			ps291.OverlayValues[31] = d31
			ps291.OverlayValues[32] = d32
			ps291.OverlayValues[33] = d33
			ps291.OverlayValues[34] = d34
			ps291.OverlayValues[35] = d35
			ps291.OverlayValues[36] = d36
			ps291.OverlayValues[37] = d37
			ps291.OverlayValues[38] = d38
			ps291.OverlayValues[39] = d39
			ps291.OverlayValues[40] = d40
			ps291.OverlayValues[41] = d41
			ps291.OverlayValues[42] = d42
			ps291.OverlayValues[43] = d43
			ps291.OverlayValues[44] = d44
			ps291.OverlayValues[45] = d45
			ps291.OverlayValues[46] = d46
			ps291.OverlayValues[47] = d47
			ps291.OverlayValues[48] = d48
			ps291.OverlayValues[49] = d49
			ps291.OverlayValues[50] = d50
			ps291.OverlayValues[51] = d51
			ps291.OverlayValues[52] = d52
			ps291.OverlayValues[53] = d53
			ps291.OverlayValues[54] = d54
			ps291.OverlayValues[55] = d55
			ps291.OverlayValues[56] = d56
			ps291.OverlayValues[57] = d57
			ps291.OverlayValues[58] = d58
			ps291.OverlayValues[59] = d59
			ps291.OverlayValues[60] = d60
			ps291.OverlayValues[61] = d61
			ps291.OverlayValues[64] = d64
			ps291.OverlayValues[65] = d65
			ps291.OverlayValues[66] = d66
			ps291.OverlayValues[134] = d134
			ps291.OverlayValues[135] = d135
			ps291.OverlayValues[136] = d136
			ps291.OverlayValues[137] = d137
			ps291.OverlayValues[138] = d138
			ps291.OverlayValues[139] = d139
			ps291.OverlayValues[140] = d140
			ps291.OverlayValues[141] = d141
			ps291.OverlayValues[142] = d142
			ps291.OverlayValues[143] = d143
			ps291.OverlayValues[144] = d144
			ps291.OverlayValues[145] = d145
			ps291.OverlayValues[146] = d146
			ps291.OverlayValues[147] = d147
			ps291.OverlayValues[148] = d148
			ps291.OverlayValues[149] = d149
			ps291.OverlayValues[150] = d150
			ps291.OverlayValues[151] = d151
			ps291.OverlayValues[152] = d152
			ps291.OverlayValues[153] = d153
			ps291.OverlayValues[154] = d154
			ps291.OverlayValues[155] = d155
			ps291.OverlayValues[156] = d156
			ps291.OverlayValues[157] = d157
			ps291.OverlayValues[158] = d158
			ps291.OverlayValues[159] = d159
			ps291.OverlayValues[160] = d160
			ps291.OverlayValues[161] = d161
			ps291.OverlayValues[162] = d162
			ps291.OverlayValues[163] = d163
			ps291.OverlayValues[164] = d164
			ps291.OverlayValues[165] = d165
			ps291.OverlayValues[166] = d166
			ps291.OverlayValues[167] = d167
			ps291.OverlayValues[168] = d168
			ps291.OverlayValues[169] = d169
			ps291.OverlayValues[170] = d170
			ps291.OverlayValues[171] = d171
			ps291.OverlayValues[172] = d172
			ps291.OverlayValues[175] = d175
			ps291.OverlayValues[283] = d283
			ps291.OverlayValues[284] = d284
			ps291.OverlayValues[285] = d285
			ps291.OverlayValues[286] = d286
			ps291.OverlayValues[287] = d287
			ps291.OverlayValues[288] = d288
			ps291.OverlayValues[289] = d289
			ps291.OverlayValues[290] = d290
			ps291.PhiValues = make([]scm.JITValueDesc, 3)
			d292 = d284
			ps291.PhiValues[0] = d292
			d293 = d1
			ps291.PhiValues[1] = d293
			d294 = d283
			ps291.PhiValues[2] = d294
			if ps291.General && bbs[4].Rendered {
				ctx.EmitJmp(lbl5)
				return result
			}
			return bbs[4].RenderPS(ps291)
			return result
			}
			bbs[4].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
					d295 := ps.PhiValues[0]
					ctx.EnsureDesc(&d295)
					ctx.EmitStoreToStack(d295, int32(bbs[4].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
					d296 := ps.PhiValues[1]
					ctx.EnsureDesc(&d296)
					ctx.EmitStoreToStack(d296, int32(bbs[4].PhiBase)+int32(16))
				}
				if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
					d297 := ps.PhiValues[2]
					ctx.EnsureDesc(&d297)
					ctx.EmitStoreToStack(d297, int32(bbs[4].PhiBase)+int32(32))
				}
				if bbs[4].VisitCount >= 2 {
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
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
				d14 = ps.OverlayValues[14]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
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
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != scm.LocNone {
				d283 = ps.OverlayValues[283]
			}
			if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != scm.LocNone {
				d284 = ps.OverlayValues[284]
			}
			if len(ps.OverlayValues) > 285 && ps.OverlayValues[285].Loc != scm.LocNone {
				d285 = ps.OverlayValues[285]
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
			if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != scm.LocNone {
				d292 = ps.OverlayValues[292]
			}
			if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != scm.LocNone {
				d293 = ps.OverlayValues[293]
			}
			if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
				d294 = ps.OverlayValues[294]
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
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d4 = ps.PhiValues[0]
			}
			if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d5 = ps.PhiValues[1]
			}
			if !ps.General && len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d6 = ps.PhiValues[2]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d6)
			var d298 scm.JITValueDesc
			if d5.Loc == scm.LocImm && d6.Loc == scm.LocImm {
				d298 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d5.Imm.Int()) == uint64(d6.Imm.Int()))}
			} else if d6.Loc == scm.LocImm {
				r102 := ctx.AllocRegExcept(d5.Reg)
				if d6.Imm.Int() >= -2147483648 && d6.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d5.Reg, int32(d6.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d6.Imm.Int()))
					ctx.EmitCmpInt64(d5.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r102, scm.CcE)
				d298 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r102}
				ctx.BindReg(r102, &d298)
			} else if d5.Loc == scm.LocImm {
				r103 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d5.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d6.Reg)
				ctx.EmitSetcc(r103, scm.CcE)
				d298 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r103}
				ctx.BindReg(r103, &d298)
			} else {
				r104 := ctx.AllocRegExcept(d5.Reg)
				ctx.EmitCmpInt64(d5.Reg, d6.Reg)
				ctx.EmitSetcc(r104, scm.CcE)
				d298 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r104}
				ctx.BindReg(r104, &d298)
			}
			d299 = d298
			ctx.EnsureDesc(&d299)
			if d299.Loc != scm.LocImm && d299.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d299.Loc == scm.LocImm {
				if d299.Imm.Bool() {
			ctx.EnsureDesc(&d5)
			if d5.Loc == scm.LocReg {
				ctx.ProtectReg(d5.Reg)
			} else if d5.Loc == scm.LocRegPair {
				ctx.ProtectReg(d5.Reg)
				ctx.ProtectReg(d5.Reg2)
			}
			d300 = d5
			if d300.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d300)
			d301 = d300
			if d301.Loc == scm.LocImm {
				d301 = scm.JITValueDesc{Loc: scm.LocImm, Type: d301.Type, Imm: scm.NewInt(int64(uint64(d301.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d301.Reg, 32)
				ctx.EmitShrRegImm8(d301.Reg, 32)
			}
			ctx.EmitStoreToStack(d301, int32(bbs[2].PhiBase)+int32(0))
			if d5.Loc == scm.LocReg {
				ctx.UnprotectReg(d5.Reg)
			} else if d5.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d5.Reg)
				ctx.UnprotectReg(d5.Reg2)
			}
			ps302 := scm.PhiState{General: ps.General}
			ps302.OverlayValues = make([]scm.JITValueDesc, 302)
			ps302.OverlayValues[0] = d0
			ps302.OverlayValues[1] = d1
			ps302.OverlayValues[2] = d2
			ps302.OverlayValues[3] = d3
			ps302.OverlayValues[4] = d4
			ps302.OverlayValues[5] = d5
			ps302.OverlayValues[6] = d6
			ps302.OverlayValues[7] = d7
			ps302.OverlayValues[8] = d8
			ps302.OverlayValues[9] = d9
			ps302.OverlayValues[10] = d10
			ps302.OverlayValues[11] = d11
			ps302.OverlayValues[12] = d12
			ps302.OverlayValues[13] = d13
			ps302.OverlayValues[14] = d14
			ps302.OverlayValues[15] = d15
			ps302.OverlayValues[16] = d16
			ps302.OverlayValues[18] = d18
			ps302.OverlayValues[19] = d19
			ps302.OverlayValues[20] = d20
			ps302.OverlayValues[21] = d21
			ps302.OverlayValues[22] = d22
			ps302.OverlayValues[23] = d23
			ps302.OverlayValues[24] = d24
			ps302.OverlayValues[25] = d25
			ps302.OverlayValues[26] = d26
			ps302.OverlayValues[27] = d27
			ps302.OverlayValues[28] = d28
			ps302.OverlayValues[29] = d29
			ps302.OverlayValues[30] = d30
			ps302.OverlayValues[31] = d31
			ps302.OverlayValues[32] = d32
			ps302.OverlayValues[33] = d33
			ps302.OverlayValues[34] = d34
			ps302.OverlayValues[35] = d35
			ps302.OverlayValues[36] = d36
			ps302.OverlayValues[37] = d37
			ps302.OverlayValues[38] = d38
			ps302.OverlayValues[39] = d39
			ps302.OverlayValues[40] = d40
			ps302.OverlayValues[41] = d41
			ps302.OverlayValues[42] = d42
			ps302.OverlayValues[43] = d43
			ps302.OverlayValues[44] = d44
			ps302.OverlayValues[45] = d45
			ps302.OverlayValues[46] = d46
			ps302.OverlayValues[47] = d47
			ps302.OverlayValues[48] = d48
			ps302.OverlayValues[49] = d49
			ps302.OverlayValues[50] = d50
			ps302.OverlayValues[51] = d51
			ps302.OverlayValues[52] = d52
			ps302.OverlayValues[53] = d53
			ps302.OverlayValues[54] = d54
			ps302.OverlayValues[55] = d55
			ps302.OverlayValues[56] = d56
			ps302.OverlayValues[57] = d57
			ps302.OverlayValues[58] = d58
			ps302.OverlayValues[59] = d59
			ps302.OverlayValues[60] = d60
			ps302.OverlayValues[61] = d61
			ps302.OverlayValues[64] = d64
			ps302.OverlayValues[65] = d65
			ps302.OverlayValues[66] = d66
			ps302.OverlayValues[134] = d134
			ps302.OverlayValues[135] = d135
			ps302.OverlayValues[136] = d136
			ps302.OverlayValues[137] = d137
			ps302.OverlayValues[138] = d138
			ps302.OverlayValues[139] = d139
			ps302.OverlayValues[140] = d140
			ps302.OverlayValues[141] = d141
			ps302.OverlayValues[142] = d142
			ps302.OverlayValues[143] = d143
			ps302.OverlayValues[144] = d144
			ps302.OverlayValues[145] = d145
			ps302.OverlayValues[146] = d146
			ps302.OverlayValues[147] = d147
			ps302.OverlayValues[148] = d148
			ps302.OverlayValues[149] = d149
			ps302.OverlayValues[150] = d150
			ps302.OverlayValues[151] = d151
			ps302.OverlayValues[152] = d152
			ps302.OverlayValues[153] = d153
			ps302.OverlayValues[154] = d154
			ps302.OverlayValues[155] = d155
			ps302.OverlayValues[156] = d156
			ps302.OverlayValues[157] = d157
			ps302.OverlayValues[158] = d158
			ps302.OverlayValues[159] = d159
			ps302.OverlayValues[160] = d160
			ps302.OverlayValues[161] = d161
			ps302.OverlayValues[162] = d162
			ps302.OverlayValues[163] = d163
			ps302.OverlayValues[164] = d164
			ps302.OverlayValues[165] = d165
			ps302.OverlayValues[166] = d166
			ps302.OverlayValues[167] = d167
			ps302.OverlayValues[168] = d168
			ps302.OverlayValues[169] = d169
			ps302.OverlayValues[170] = d170
			ps302.OverlayValues[171] = d171
			ps302.OverlayValues[172] = d172
			ps302.OverlayValues[175] = d175
			ps302.OverlayValues[283] = d283
			ps302.OverlayValues[284] = d284
			ps302.OverlayValues[285] = d285
			ps302.OverlayValues[286] = d286
			ps302.OverlayValues[287] = d287
			ps302.OverlayValues[288] = d288
			ps302.OverlayValues[289] = d289
			ps302.OverlayValues[290] = d290
			ps302.OverlayValues[292] = d292
			ps302.OverlayValues[293] = d293
			ps302.OverlayValues[294] = d294
			ps302.OverlayValues[295] = d295
			ps302.OverlayValues[296] = d296
			ps302.OverlayValues[297] = d297
			ps302.OverlayValues[298] = d298
			ps302.OverlayValues[299] = d299
			ps302.OverlayValues[300] = d300
			ps302.OverlayValues[301] = d301
			ps302.PhiValues = make([]scm.JITValueDesc, 1)
			d303 = d5
			ps302.PhiValues[0] = d303
					return bbs[2].RenderPS(ps302)
				}
			ps304 := scm.PhiState{General: ps.General}
			ps304.OverlayValues = make([]scm.JITValueDesc, 304)
			ps304.OverlayValues[0] = d0
			ps304.OverlayValues[1] = d1
			ps304.OverlayValues[2] = d2
			ps304.OverlayValues[3] = d3
			ps304.OverlayValues[4] = d4
			ps304.OverlayValues[5] = d5
			ps304.OverlayValues[6] = d6
			ps304.OverlayValues[7] = d7
			ps304.OverlayValues[8] = d8
			ps304.OverlayValues[9] = d9
			ps304.OverlayValues[10] = d10
			ps304.OverlayValues[11] = d11
			ps304.OverlayValues[12] = d12
			ps304.OverlayValues[13] = d13
			ps304.OverlayValues[14] = d14
			ps304.OverlayValues[15] = d15
			ps304.OverlayValues[16] = d16
			ps304.OverlayValues[18] = d18
			ps304.OverlayValues[19] = d19
			ps304.OverlayValues[20] = d20
			ps304.OverlayValues[21] = d21
			ps304.OverlayValues[22] = d22
			ps304.OverlayValues[23] = d23
			ps304.OverlayValues[24] = d24
			ps304.OverlayValues[25] = d25
			ps304.OverlayValues[26] = d26
			ps304.OverlayValues[27] = d27
			ps304.OverlayValues[28] = d28
			ps304.OverlayValues[29] = d29
			ps304.OverlayValues[30] = d30
			ps304.OverlayValues[31] = d31
			ps304.OverlayValues[32] = d32
			ps304.OverlayValues[33] = d33
			ps304.OverlayValues[34] = d34
			ps304.OverlayValues[35] = d35
			ps304.OverlayValues[36] = d36
			ps304.OverlayValues[37] = d37
			ps304.OverlayValues[38] = d38
			ps304.OverlayValues[39] = d39
			ps304.OverlayValues[40] = d40
			ps304.OverlayValues[41] = d41
			ps304.OverlayValues[42] = d42
			ps304.OverlayValues[43] = d43
			ps304.OverlayValues[44] = d44
			ps304.OverlayValues[45] = d45
			ps304.OverlayValues[46] = d46
			ps304.OverlayValues[47] = d47
			ps304.OverlayValues[48] = d48
			ps304.OverlayValues[49] = d49
			ps304.OverlayValues[50] = d50
			ps304.OverlayValues[51] = d51
			ps304.OverlayValues[52] = d52
			ps304.OverlayValues[53] = d53
			ps304.OverlayValues[54] = d54
			ps304.OverlayValues[55] = d55
			ps304.OverlayValues[56] = d56
			ps304.OverlayValues[57] = d57
			ps304.OverlayValues[58] = d58
			ps304.OverlayValues[59] = d59
			ps304.OverlayValues[60] = d60
			ps304.OverlayValues[61] = d61
			ps304.OverlayValues[64] = d64
			ps304.OverlayValues[65] = d65
			ps304.OverlayValues[66] = d66
			ps304.OverlayValues[134] = d134
			ps304.OverlayValues[135] = d135
			ps304.OverlayValues[136] = d136
			ps304.OverlayValues[137] = d137
			ps304.OverlayValues[138] = d138
			ps304.OverlayValues[139] = d139
			ps304.OverlayValues[140] = d140
			ps304.OverlayValues[141] = d141
			ps304.OverlayValues[142] = d142
			ps304.OverlayValues[143] = d143
			ps304.OverlayValues[144] = d144
			ps304.OverlayValues[145] = d145
			ps304.OverlayValues[146] = d146
			ps304.OverlayValues[147] = d147
			ps304.OverlayValues[148] = d148
			ps304.OverlayValues[149] = d149
			ps304.OverlayValues[150] = d150
			ps304.OverlayValues[151] = d151
			ps304.OverlayValues[152] = d152
			ps304.OverlayValues[153] = d153
			ps304.OverlayValues[154] = d154
			ps304.OverlayValues[155] = d155
			ps304.OverlayValues[156] = d156
			ps304.OverlayValues[157] = d157
			ps304.OverlayValues[158] = d158
			ps304.OverlayValues[159] = d159
			ps304.OverlayValues[160] = d160
			ps304.OverlayValues[161] = d161
			ps304.OverlayValues[162] = d162
			ps304.OverlayValues[163] = d163
			ps304.OverlayValues[164] = d164
			ps304.OverlayValues[165] = d165
			ps304.OverlayValues[166] = d166
			ps304.OverlayValues[167] = d167
			ps304.OverlayValues[168] = d168
			ps304.OverlayValues[169] = d169
			ps304.OverlayValues[170] = d170
			ps304.OverlayValues[171] = d171
			ps304.OverlayValues[172] = d172
			ps304.OverlayValues[175] = d175
			ps304.OverlayValues[283] = d283
			ps304.OverlayValues[284] = d284
			ps304.OverlayValues[285] = d285
			ps304.OverlayValues[286] = d286
			ps304.OverlayValues[287] = d287
			ps304.OverlayValues[288] = d288
			ps304.OverlayValues[289] = d289
			ps304.OverlayValues[290] = d290
			ps304.OverlayValues[292] = d292
			ps304.OverlayValues[293] = d293
			ps304.OverlayValues[294] = d294
			ps304.OverlayValues[295] = d295
			ps304.OverlayValues[296] = d296
			ps304.OverlayValues[297] = d297
			ps304.OverlayValues[298] = d298
			ps304.OverlayValues[299] = d299
			ps304.OverlayValues[300] = d300
			ps304.OverlayValues[301] = d301
			ps304.OverlayValues[303] = d303
				return bbs[6].RenderPS(ps304)
			}
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
					d305 := ps.PhiValues[0]
					ctx.EnsureDesc(&d305)
					ctx.EmitStoreToStack(d305, int32(bbs[4].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
					d306 := ps.PhiValues[1]
					ctx.EnsureDesc(&d306)
					ctx.EmitStoreToStack(d306, int32(bbs[4].PhiBase)+int32(16))
				}
				if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
					d307 := ps.PhiValues[2]
					ctx.EnsureDesc(&d307)
					ctx.EmitStoreToStack(d307, int32(bbs[4].PhiBase)+int32(32))
				}
				ps.General = true
				return bbs[4].RenderPS(ps)
			}
			lbl29 := ctx.ReserveLabel()
			lbl30 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d299.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl29)
			ctx.EmitJmp(lbl30)
			ctx.MarkLabel(lbl29)
			ctx.EnsureDesc(&d5)
			if d5.Loc == scm.LocReg {
				ctx.ProtectReg(d5.Reg)
			} else if d5.Loc == scm.LocRegPair {
				ctx.ProtectReg(d5.Reg)
				ctx.ProtectReg(d5.Reg2)
			}
			d308 = d5
			if d308.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d308)
			d309 = d308
			if d309.Loc == scm.LocImm {
				d309 = scm.JITValueDesc{Loc: scm.LocImm, Type: d309.Type, Imm: scm.NewInt(int64(uint64(d309.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d309.Reg, 32)
				ctx.EmitShrRegImm8(d309.Reg, 32)
			}
			ctx.EmitStoreToStack(d309, int32(bbs[2].PhiBase)+int32(0))
			if d5.Loc == scm.LocReg {
				ctx.UnprotectReg(d5.Reg)
			} else if d5.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d5.Reg)
				ctx.UnprotectReg(d5.Reg2)
			}
			ctx.EmitJmp(lbl3)
			ctx.MarkLabel(lbl30)
			ctx.EmitJmp(lbl7)
			ps310 := scm.PhiState{General: true}
			ps310.OverlayValues = make([]scm.JITValueDesc, 310)
			ps310.OverlayValues[0] = d0
			ps310.OverlayValues[1] = d1
			ps310.OverlayValues[2] = d2
			ps310.OverlayValues[3] = d3
			ps310.OverlayValues[4] = d4
			ps310.OverlayValues[5] = d5
			ps310.OverlayValues[6] = d6
			ps310.OverlayValues[7] = d7
			ps310.OverlayValues[8] = d8
			ps310.OverlayValues[9] = d9
			ps310.OverlayValues[10] = d10
			ps310.OverlayValues[11] = d11
			ps310.OverlayValues[12] = d12
			ps310.OverlayValues[13] = d13
			ps310.OverlayValues[14] = d14
			ps310.OverlayValues[15] = d15
			ps310.OverlayValues[16] = d16
			ps310.OverlayValues[18] = d18
			ps310.OverlayValues[19] = d19
			ps310.OverlayValues[20] = d20
			ps310.OverlayValues[21] = d21
			ps310.OverlayValues[22] = d22
			ps310.OverlayValues[23] = d23
			ps310.OverlayValues[24] = d24
			ps310.OverlayValues[25] = d25
			ps310.OverlayValues[26] = d26
			ps310.OverlayValues[27] = d27
			ps310.OverlayValues[28] = d28
			ps310.OverlayValues[29] = d29
			ps310.OverlayValues[30] = d30
			ps310.OverlayValues[31] = d31
			ps310.OverlayValues[32] = d32
			ps310.OverlayValues[33] = d33
			ps310.OverlayValues[34] = d34
			ps310.OverlayValues[35] = d35
			ps310.OverlayValues[36] = d36
			ps310.OverlayValues[37] = d37
			ps310.OverlayValues[38] = d38
			ps310.OverlayValues[39] = d39
			ps310.OverlayValues[40] = d40
			ps310.OverlayValues[41] = d41
			ps310.OverlayValues[42] = d42
			ps310.OverlayValues[43] = d43
			ps310.OverlayValues[44] = d44
			ps310.OverlayValues[45] = d45
			ps310.OverlayValues[46] = d46
			ps310.OverlayValues[47] = d47
			ps310.OverlayValues[48] = d48
			ps310.OverlayValues[49] = d49
			ps310.OverlayValues[50] = d50
			ps310.OverlayValues[51] = d51
			ps310.OverlayValues[52] = d52
			ps310.OverlayValues[53] = d53
			ps310.OverlayValues[54] = d54
			ps310.OverlayValues[55] = d55
			ps310.OverlayValues[56] = d56
			ps310.OverlayValues[57] = d57
			ps310.OverlayValues[58] = d58
			ps310.OverlayValues[59] = d59
			ps310.OverlayValues[60] = d60
			ps310.OverlayValues[61] = d61
			ps310.OverlayValues[64] = d64
			ps310.OverlayValues[65] = d65
			ps310.OverlayValues[66] = d66
			ps310.OverlayValues[134] = d134
			ps310.OverlayValues[135] = d135
			ps310.OverlayValues[136] = d136
			ps310.OverlayValues[137] = d137
			ps310.OverlayValues[138] = d138
			ps310.OverlayValues[139] = d139
			ps310.OverlayValues[140] = d140
			ps310.OverlayValues[141] = d141
			ps310.OverlayValues[142] = d142
			ps310.OverlayValues[143] = d143
			ps310.OverlayValues[144] = d144
			ps310.OverlayValues[145] = d145
			ps310.OverlayValues[146] = d146
			ps310.OverlayValues[147] = d147
			ps310.OverlayValues[148] = d148
			ps310.OverlayValues[149] = d149
			ps310.OverlayValues[150] = d150
			ps310.OverlayValues[151] = d151
			ps310.OverlayValues[152] = d152
			ps310.OverlayValues[153] = d153
			ps310.OverlayValues[154] = d154
			ps310.OverlayValues[155] = d155
			ps310.OverlayValues[156] = d156
			ps310.OverlayValues[157] = d157
			ps310.OverlayValues[158] = d158
			ps310.OverlayValues[159] = d159
			ps310.OverlayValues[160] = d160
			ps310.OverlayValues[161] = d161
			ps310.OverlayValues[162] = d162
			ps310.OverlayValues[163] = d163
			ps310.OverlayValues[164] = d164
			ps310.OverlayValues[165] = d165
			ps310.OverlayValues[166] = d166
			ps310.OverlayValues[167] = d167
			ps310.OverlayValues[168] = d168
			ps310.OverlayValues[169] = d169
			ps310.OverlayValues[170] = d170
			ps310.OverlayValues[171] = d171
			ps310.OverlayValues[172] = d172
			ps310.OverlayValues[175] = d175
			ps310.OverlayValues[283] = d283
			ps310.OverlayValues[284] = d284
			ps310.OverlayValues[285] = d285
			ps310.OverlayValues[286] = d286
			ps310.OverlayValues[287] = d287
			ps310.OverlayValues[288] = d288
			ps310.OverlayValues[289] = d289
			ps310.OverlayValues[290] = d290
			ps310.OverlayValues[292] = d292
			ps310.OverlayValues[293] = d293
			ps310.OverlayValues[294] = d294
			ps310.OverlayValues[295] = d295
			ps310.OverlayValues[296] = d296
			ps310.OverlayValues[297] = d297
			ps310.OverlayValues[298] = d298
			ps310.OverlayValues[299] = d299
			ps310.OverlayValues[300] = d300
			ps310.OverlayValues[301] = d301
			ps310.OverlayValues[303] = d303
			ps310.OverlayValues[305] = d305
			ps310.OverlayValues[306] = d306
			ps310.OverlayValues[307] = d307
			ps310.OverlayValues[308] = d308
			ps310.OverlayValues[309] = d309
			ps310.PhiValues = make([]scm.JITValueDesc, 1)
			d312 = d5
			ps310.PhiValues[0] = d312
			ps311 := scm.PhiState{General: true}
			ps311.OverlayValues = make([]scm.JITValueDesc, 313)
			ps311.OverlayValues[0] = d0
			ps311.OverlayValues[1] = d1
			ps311.OverlayValues[2] = d2
			ps311.OverlayValues[3] = d3
			ps311.OverlayValues[4] = d4
			ps311.OverlayValues[5] = d5
			ps311.OverlayValues[6] = d6
			ps311.OverlayValues[7] = d7
			ps311.OverlayValues[8] = d8
			ps311.OverlayValues[9] = d9
			ps311.OverlayValues[10] = d10
			ps311.OverlayValues[11] = d11
			ps311.OverlayValues[12] = d12
			ps311.OverlayValues[13] = d13
			ps311.OverlayValues[14] = d14
			ps311.OverlayValues[15] = d15
			ps311.OverlayValues[16] = d16
			ps311.OverlayValues[18] = d18
			ps311.OverlayValues[19] = d19
			ps311.OverlayValues[20] = d20
			ps311.OverlayValues[21] = d21
			ps311.OverlayValues[22] = d22
			ps311.OverlayValues[23] = d23
			ps311.OverlayValues[24] = d24
			ps311.OverlayValues[25] = d25
			ps311.OverlayValues[26] = d26
			ps311.OverlayValues[27] = d27
			ps311.OverlayValues[28] = d28
			ps311.OverlayValues[29] = d29
			ps311.OverlayValues[30] = d30
			ps311.OverlayValues[31] = d31
			ps311.OverlayValues[32] = d32
			ps311.OverlayValues[33] = d33
			ps311.OverlayValues[34] = d34
			ps311.OverlayValues[35] = d35
			ps311.OverlayValues[36] = d36
			ps311.OverlayValues[37] = d37
			ps311.OverlayValues[38] = d38
			ps311.OverlayValues[39] = d39
			ps311.OverlayValues[40] = d40
			ps311.OverlayValues[41] = d41
			ps311.OverlayValues[42] = d42
			ps311.OverlayValues[43] = d43
			ps311.OverlayValues[44] = d44
			ps311.OverlayValues[45] = d45
			ps311.OverlayValues[46] = d46
			ps311.OverlayValues[47] = d47
			ps311.OverlayValues[48] = d48
			ps311.OverlayValues[49] = d49
			ps311.OverlayValues[50] = d50
			ps311.OverlayValues[51] = d51
			ps311.OverlayValues[52] = d52
			ps311.OverlayValues[53] = d53
			ps311.OverlayValues[54] = d54
			ps311.OverlayValues[55] = d55
			ps311.OverlayValues[56] = d56
			ps311.OverlayValues[57] = d57
			ps311.OverlayValues[58] = d58
			ps311.OverlayValues[59] = d59
			ps311.OverlayValues[60] = d60
			ps311.OverlayValues[61] = d61
			ps311.OverlayValues[64] = d64
			ps311.OverlayValues[65] = d65
			ps311.OverlayValues[66] = d66
			ps311.OverlayValues[134] = d134
			ps311.OverlayValues[135] = d135
			ps311.OverlayValues[136] = d136
			ps311.OverlayValues[137] = d137
			ps311.OverlayValues[138] = d138
			ps311.OverlayValues[139] = d139
			ps311.OverlayValues[140] = d140
			ps311.OverlayValues[141] = d141
			ps311.OverlayValues[142] = d142
			ps311.OverlayValues[143] = d143
			ps311.OverlayValues[144] = d144
			ps311.OverlayValues[145] = d145
			ps311.OverlayValues[146] = d146
			ps311.OverlayValues[147] = d147
			ps311.OverlayValues[148] = d148
			ps311.OverlayValues[149] = d149
			ps311.OverlayValues[150] = d150
			ps311.OverlayValues[151] = d151
			ps311.OverlayValues[152] = d152
			ps311.OverlayValues[153] = d153
			ps311.OverlayValues[154] = d154
			ps311.OverlayValues[155] = d155
			ps311.OverlayValues[156] = d156
			ps311.OverlayValues[157] = d157
			ps311.OverlayValues[158] = d158
			ps311.OverlayValues[159] = d159
			ps311.OverlayValues[160] = d160
			ps311.OverlayValues[161] = d161
			ps311.OverlayValues[162] = d162
			ps311.OverlayValues[163] = d163
			ps311.OverlayValues[164] = d164
			ps311.OverlayValues[165] = d165
			ps311.OverlayValues[166] = d166
			ps311.OverlayValues[167] = d167
			ps311.OverlayValues[168] = d168
			ps311.OverlayValues[169] = d169
			ps311.OverlayValues[170] = d170
			ps311.OverlayValues[171] = d171
			ps311.OverlayValues[172] = d172
			ps311.OverlayValues[175] = d175
			ps311.OverlayValues[283] = d283
			ps311.OverlayValues[284] = d284
			ps311.OverlayValues[285] = d285
			ps311.OverlayValues[286] = d286
			ps311.OverlayValues[287] = d287
			ps311.OverlayValues[288] = d288
			ps311.OverlayValues[289] = d289
			ps311.OverlayValues[290] = d290
			ps311.OverlayValues[292] = d292
			ps311.OverlayValues[293] = d293
			ps311.OverlayValues[294] = d294
			ps311.OverlayValues[295] = d295
			ps311.OverlayValues[296] = d296
			ps311.OverlayValues[297] = d297
			ps311.OverlayValues[298] = d298
			ps311.OverlayValues[299] = d299
			ps311.OverlayValues[300] = d300
			ps311.OverlayValues[301] = d301
			ps311.OverlayValues[303] = d303
			ps311.OverlayValues[305] = d305
			ps311.OverlayValues[306] = d306
			ps311.OverlayValues[307] = d307
			ps311.OverlayValues[308] = d308
			ps311.OverlayValues[309] = d309
			ps311.OverlayValues[312] = d312
			snap313 := d0
			snap314 := d1
			snap315 := d2
			snap316 := d3
			snap317 := d4
			snap318 := d5
			snap319 := d6
			snap320 := d7
			snap321 := d8
			snap322 := d9
			snap323 := d10
			snap324 := d11
			snap325 := d12
			snap326 := d13
			snap327 := d14
			snap328 := d15
			snap329 := d16
			snap330 := d18
			snap331 := d19
			snap332 := d20
			snap333 := d21
			snap334 := d22
			snap335 := d23
			snap336 := d24
			snap337 := d25
			snap338 := d26
			snap339 := d27
			snap340 := d28
			snap341 := d29
			snap342 := d30
			snap343 := d31
			snap344 := d32
			snap345 := d33
			snap346 := d34
			snap347 := d35
			snap348 := d36
			snap349 := d37
			snap350 := d38
			snap351 := d39
			snap352 := d40
			snap353 := d41
			snap354 := d42
			snap355 := d43
			snap356 := d44
			snap357 := d45
			snap358 := d46
			snap359 := d47
			snap360 := d48
			snap361 := d49
			snap362 := d50
			snap363 := d51
			snap364 := d52
			snap365 := d53
			snap366 := d54
			snap367 := d55
			snap368 := d56
			snap369 := d57
			snap370 := d58
			snap371 := d59
			snap372 := d60
			snap373 := d61
			snap374 := d64
			snap375 := d65
			snap376 := d66
			snap377 := d134
			snap378 := d135
			snap379 := d136
			snap380 := d137
			snap381 := d138
			snap382 := d139
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
			snap416 := d175
			snap417 := d283
			snap418 := d284
			snap419 := d285
			snap420 := d286
			snap421 := d287
			snap422 := d288
			snap423 := d289
			snap424 := d290
			snap425 := d292
			snap426 := d293
			snap427 := d294
			snap428 := d295
			snap429 := d296
			snap430 := d297
			snap431 := d298
			snap432 := d299
			snap433 := d300
			snap434 := d301
			snap435 := d303
			snap436 := d305
			snap437 := d306
			snap438 := d307
			snap439 := d308
			snap440 := d309
			snap441 := d312
			alloc442 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps310)
			}
			ctx.RestoreAllocState(alloc442)
			d0 = snap313
			d1 = snap314
			d2 = snap315
			d3 = snap316
			d4 = snap317
			d5 = snap318
			d6 = snap319
			d7 = snap320
			d8 = snap321
			d9 = snap322
			d10 = snap323
			d11 = snap324
			d12 = snap325
			d13 = snap326
			d14 = snap327
			d15 = snap328
			d16 = snap329
			d18 = snap330
			d19 = snap331
			d20 = snap332
			d21 = snap333
			d22 = snap334
			d23 = snap335
			d24 = snap336
			d25 = snap337
			d26 = snap338
			d27 = snap339
			d28 = snap340
			d29 = snap341
			d30 = snap342
			d31 = snap343
			d32 = snap344
			d33 = snap345
			d34 = snap346
			d35 = snap347
			d36 = snap348
			d37 = snap349
			d38 = snap350
			d39 = snap351
			d40 = snap352
			d41 = snap353
			d42 = snap354
			d43 = snap355
			d44 = snap356
			d45 = snap357
			d46 = snap358
			d47 = snap359
			d48 = snap360
			d49 = snap361
			d50 = snap362
			d51 = snap363
			d52 = snap364
			d53 = snap365
			d54 = snap366
			d55 = snap367
			d56 = snap368
			d57 = snap369
			d58 = snap370
			d59 = snap371
			d60 = snap372
			d61 = snap373
			d64 = snap374
			d65 = snap375
			d66 = snap376
			d134 = snap377
			d135 = snap378
			d136 = snap379
			d137 = snap380
			d138 = snap381
			d139 = snap382
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
			d175 = snap416
			d283 = snap417
			d284 = snap418
			d285 = snap419
			d286 = snap420
			d287 = snap421
			d288 = snap422
			d289 = snap423
			d290 = snap424
			d292 = snap425
			d293 = snap426
			d294 = snap427
			d295 = snap428
			d296 = snap429
			d297 = snap430
			d298 = snap431
			d299 = snap432
			d300 = snap433
			d301 = snap434
			d303 = snap435
			d305 = snap436
			d306 = snap437
			d307 = snap438
			d308 = snap439
			d309 = snap440
			d312 = snap441
			if !bbs[6].Rendered {
				return bbs[6].RenderPS(ps311)
			}
			return result
			ctx.FreeDesc(&d298)
			return result
			}
			bbs[5].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[5].VisitCount >= 2 {
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
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
				d14 = ps.OverlayValues[14]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
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
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != scm.LocNone {
				d283 = ps.OverlayValues[283]
			}
			if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != scm.LocNone {
				d284 = ps.OverlayValues[284]
			}
			if len(ps.OverlayValues) > 285 && ps.OverlayValues[285].Loc != scm.LocNone {
				d285 = ps.OverlayValues[285]
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
			if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != scm.LocNone {
				d292 = ps.OverlayValues[292]
			}
			if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != scm.LocNone {
				d293 = ps.OverlayValues[293]
			}
			if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
				d294 = ps.OverlayValues[294]
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
			if len(ps.OverlayValues) > 303 && ps.OverlayValues[303].Loc != scm.LocNone {
				d303 = ps.OverlayValues[303]
			}
			if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != scm.LocNone {
				d305 = ps.OverlayValues[305]
			}
			if len(ps.OverlayValues) > 306 && ps.OverlayValues[306].Loc != scm.LocNone {
				d306 = ps.OverlayValues[306]
			}
			if len(ps.OverlayValues) > 307 && ps.OverlayValues[307].Loc != scm.LocNone {
				d307 = ps.OverlayValues[307]
			}
			if len(ps.OverlayValues) > 308 && ps.OverlayValues[308].Loc != scm.LocNone {
				d308 = ps.OverlayValues[308]
			}
			if len(ps.OverlayValues) > 309 && ps.OverlayValues[309].Loc != scm.LocNone {
				d309 = ps.OverlayValues[309]
			}
			if len(ps.OverlayValues) > 312 && ps.OverlayValues[312].Loc != scm.LocNone {
				d312 = ps.OverlayValues[312]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			var d443 scm.JITValueDesc
			if d0.Loc == scm.LocImm {
				d443 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d0.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d0.Reg)
				ctx.EmitMovRegReg(scratch, d0.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d443 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d443)
			}
			if d443.Loc == scm.LocImm {
				d443 = scm.JITValueDesc{Loc: scm.LocImm, Type: d443.Type, Imm: scm.NewInt(int64(uint64(d443.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d443.Reg, 32)
				ctx.EmitShrRegImm8(d443.Reg, 32)
			}
			if d443.Loc == scm.LocReg && d0.Loc == scm.LocReg && d443.Reg == d0.Reg {
				ctx.TransferReg(d0.Reg)
				d0.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d0)
			if d0.Loc == scm.LocReg {
				ctx.ProtectReg(d0.Reg)
			} else if d0.Loc == scm.LocRegPair {
				ctx.ProtectReg(d0.Reg)
				ctx.ProtectReg(d0.Reg2)
			}
			ctx.EnsureDesc(&d2)
			if d2.Loc == scm.LocReg {
				ctx.ProtectReg(d2.Reg)
			} else if d2.Loc == scm.LocRegPair {
				ctx.ProtectReg(d2.Reg)
				ctx.ProtectReg(d2.Reg2)
			}
			ctx.EnsureDesc(&d443)
			if d443.Loc == scm.LocReg {
				ctx.ProtectReg(d443.Reg)
			} else if d443.Loc == scm.LocRegPair {
				ctx.ProtectReg(d443.Reg)
				ctx.ProtectReg(d443.Reg2)
			}
			d444 = d443
			if d444.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d444)
			d445 = d444
			if d445.Loc == scm.LocImm {
				d445 = scm.JITValueDesc{Loc: scm.LocImm, Type: d445.Type, Imm: scm.NewInt(int64(uint64(d445.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d445.Reg, 32)
				ctx.EmitShrRegImm8(d445.Reg, 32)
			}
			ctx.EmitStoreToStack(d445, int32(bbs[4].PhiBase)+int32(0))
			d446 = d0
			if d446.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d446)
			d447 = d446
			if d447.Loc == scm.LocImm {
				d447 = scm.JITValueDesc{Loc: scm.LocImm, Type: d447.Type, Imm: scm.NewInt(int64(uint64(d447.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d447.Reg, 32)
				ctx.EmitShrRegImm8(d447.Reg, 32)
			}
			ctx.EmitStoreToStack(d447, int32(bbs[4].PhiBase)+int32(16))
			d448 = d2
			if d448.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d448)
			d449 = d448
			if d449.Loc == scm.LocImm {
				d449 = scm.JITValueDesc{Loc: scm.LocImm, Type: d449.Type, Imm: scm.NewInt(int64(uint64(d449.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d449.Reg, 32)
				ctx.EmitShrRegImm8(d449.Reg, 32)
			}
			ctx.EmitStoreToStack(d449, int32(bbs[4].PhiBase)+int32(32))
			if d0.Loc == scm.LocReg {
				ctx.UnprotectReg(d0.Reg)
			} else if d0.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d0.Reg)
				ctx.UnprotectReg(d0.Reg2)
			}
			if d2.Loc == scm.LocReg {
				ctx.UnprotectReg(d2.Reg)
			} else if d2.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d2.Reg)
				ctx.UnprotectReg(d2.Reg2)
			}
			if d443.Loc == scm.LocReg {
				ctx.UnprotectReg(d443.Reg)
			} else if d443.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d443.Reg)
				ctx.UnprotectReg(d443.Reg2)
			}
			ps450 := scm.PhiState{General: ps.General}
			ps450.OverlayValues = make([]scm.JITValueDesc, 450)
			ps450.OverlayValues[0] = d0
			ps450.OverlayValues[1] = d1
			ps450.OverlayValues[2] = d2
			ps450.OverlayValues[3] = d3
			ps450.OverlayValues[4] = d4
			ps450.OverlayValues[5] = d5
			ps450.OverlayValues[6] = d6
			ps450.OverlayValues[7] = d7
			ps450.OverlayValues[8] = d8
			ps450.OverlayValues[9] = d9
			ps450.OverlayValues[10] = d10
			ps450.OverlayValues[11] = d11
			ps450.OverlayValues[12] = d12
			ps450.OverlayValues[13] = d13
			ps450.OverlayValues[14] = d14
			ps450.OverlayValues[15] = d15
			ps450.OverlayValues[16] = d16
			ps450.OverlayValues[18] = d18
			ps450.OverlayValues[19] = d19
			ps450.OverlayValues[20] = d20
			ps450.OverlayValues[21] = d21
			ps450.OverlayValues[22] = d22
			ps450.OverlayValues[23] = d23
			ps450.OverlayValues[24] = d24
			ps450.OverlayValues[25] = d25
			ps450.OverlayValues[26] = d26
			ps450.OverlayValues[27] = d27
			ps450.OverlayValues[28] = d28
			ps450.OverlayValues[29] = d29
			ps450.OverlayValues[30] = d30
			ps450.OverlayValues[31] = d31
			ps450.OverlayValues[32] = d32
			ps450.OverlayValues[33] = d33
			ps450.OverlayValues[34] = d34
			ps450.OverlayValues[35] = d35
			ps450.OverlayValues[36] = d36
			ps450.OverlayValues[37] = d37
			ps450.OverlayValues[38] = d38
			ps450.OverlayValues[39] = d39
			ps450.OverlayValues[40] = d40
			ps450.OverlayValues[41] = d41
			ps450.OverlayValues[42] = d42
			ps450.OverlayValues[43] = d43
			ps450.OverlayValues[44] = d44
			ps450.OverlayValues[45] = d45
			ps450.OverlayValues[46] = d46
			ps450.OverlayValues[47] = d47
			ps450.OverlayValues[48] = d48
			ps450.OverlayValues[49] = d49
			ps450.OverlayValues[50] = d50
			ps450.OverlayValues[51] = d51
			ps450.OverlayValues[52] = d52
			ps450.OverlayValues[53] = d53
			ps450.OverlayValues[54] = d54
			ps450.OverlayValues[55] = d55
			ps450.OverlayValues[56] = d56
			ps450.OverlayValues[57] = d57
			ps450.OverlayValues[58] = d58
			ps450.OverlayValues[59] = d59
			ps450.OverlayValues[60] = d60
			ps450.OverlayValues[61] = d61
			ps450.OverlayValues[64] = d64
			ps450.OverlayValues[65] = d65
			ps450.OverlayValues[66] = d66
			ps450.OverlayValues[134] = d134
			ps450.OverlayValues[135] = d135
			ps450.OverlayValues[136] = d136
			ps450.OverlayValues[137] = d137
			ps450.OverlayValues[138] = d138
			ps450.OverlayValues[139] = d139
			ps450.OverlayValues[140] = d140
			ps450.OverlayValues[141] = d141
			ps450.OverlayValues[142] = d142
			ps450.OverlayValues[143] = d143
			ps450.OverlayValues[144] = d144
			ps450.OverlayValues[145] = d145
			ps450.OverlayValues[146] = d146
			ps450.OverlayValues[147] = d147
			ps450.OverlayValues[148] = d148
			ps450.OverlayValues[149] = d149
			ps450.OverlayValues[150] = d150
			ps450.OverlayValues[151] = d151
			ps450.OverlayValues[152] = d152
			ps450.OverlayValues[153] = d153
			ps450.OverlayValues[154] = d154
			ps450.OverlayValues[155] = d155
			ps450.OverlayValues[156] = d156
			ps450.OverlayValues[157] = d157
			ps450.OverlayValues[158] = d158
			ps450.OverlayValues[159] = d159
			ps450.OverlayValues[160] = d160
			ps450.OverlayValues[161] = d161
			ps450.OverlayValues[162] = d162
			ps450.OverlayValues[163] = d163
			ps450.OverlayValues[164] = d164
			ps450.OverlayValues[165] = d165
			ps450.OverlayValues[166] = d166
			ps450.OverlayValues[167] = d167
			ps450.OverlayValues[168] = d168
			ps450.OverlayValues[169] = d169
			ps450.OverlayValues[170] = d170
			ps450.OverlayValues[171] = d171
			ps450.OverlayValues[172] = d172
			ps450.OverlayValues[175] = d175
			ps450.OverlayValues[283] = d283
			ps450.OverlayValues[284] = d284
			ps450.OverlayValues[285] = d285
			ps450.OverlayValues[286] = d286
			ps450.OverlayValues[287] = d287
			ps450.OverlayValues[288] = d288
			ps450.OverlayValues[289] = d289
			ps450.OverlayValues[290] = d290
			ps450.OverlayValues[292] = d292
			ps450.OverlayValues[293] = d293
			ps450.OverlayValues[294] = d294
			ps450.OverlayValues[295] = d295
			ps450.OverlayValues[296] = d296
			ps450.OverlayValues[297] = d297
			ps450.OverlayValues[298] = d298
			ps450.OverlayValues[299] = d299
			ps450.OverlayValues[300] = d300
			ps450.OverlayValues[301] = d301
			ps450.OverlayValues[303] = d303
			ps450.OverlayValues[305] = d305
			ps450.OverlayValues[306] = d306
			ps450.OverlayValues[307] = d307
			ps450.OverlayValues[308] = d308
			ps450.OverlayValues[309] = d309
			ps450.OverlayValues[312] = d312
			ps450.OverlayValues[443] = d443
			ps450.OverlayValues[444] = d444
			ps450.OverlayValues[445] = d445
			ps450.OverlayValues[446] = d446
			ps450.OverlayValues[447] = d447
			ps450.OverlayValues[448] = d448
			ps450.OverlayValues[449] = d449
			ps450.PhiValues = make([]scm.JITValueDesc, 3)
			d451 = d443
			ps450.PhiValues[0] = d451
			d452 = d0
			ps450.PhiValues[1] = d452
			d453 = d2
			ps450.PhiValues[2] = d453
			if ps450.General && bbs[4].Rendered {
				ctx.EmitJmp(lbl5)
				return result
			}
			return bbs[4].RenderPS(ps450)
			return result
			}
			bbs[6].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[6].VisitCount >= 2 {
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
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
				d14 = ps.OverlayValues[14]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
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
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != scm.LocNone {
				d283 = ps.OverlayValues[283]
			}
			if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != scm.LocNone {
				d284 = ps.OverlayValues[284]
			}
			if len(ps.OverlayValues) > 285 && ps.OverlayValues[285].Loc != scm.LocNone {
				d285 = ps.OverlayValues[285]
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
			if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != scm.LocNone {
				d292 = ps.OverlayValues[292]
			}
			if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != scm.LocNone {
				d293 = ps.OverlayValues[293]
			}
			if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
				d294 = ps.OverlayValues[294]
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
			if len(ps.OverlayValues) > 303 && ps.OverlayValues[303].Loc != scm.LocNone {
				d303 = ps.OverlayValues[303]
			}
			if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != scm.LocNone {
				d305 = ps.OverlayValues[305]
			}
			if len(ps.OverlayValues) > 306 && ps.OverlayValues[306].Loc != scm.LocNone {
				d306 = ps.OverlayValues[306]
			}
			if len(ps.OverlayValues) > 307 && ps.OverlayValues[307].Loc != scm.LocNone {
				d307 = ps.OverlayValues[307]
			}
			if len(ps.OverlayValues) > 308 && ps.OverlayValues[308].Loc != scm.LocNone {
				d308 = ps.OverlayValues[308]
			}
			if len(ps.OverlayValues) > 309 && ps.OverlayValues[309].Loc != scm.LocNone {
				d309 = ps.OverlayValues[309]
			}
			if len(ps.OverlayValues) > 312 && ps.OverlayValues[312].Loc != scm.LocNone {
				d312 = ps.OverlayValues[312]
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
			if len(ps.OverlayValues) > 451 && ps.OverlayValues[451].Loc != scm.LocNone {
				d451 = ps.OverlayValues[451]
			}
			if len(ps.OverlayValues) > 452 && ps.OverlayValues[452].Loc != scm.LocNone {
				d452 = ps.OverlayValues[452]
			}
			if len(ps.OverlayValues) > 453 && ps.OverlayValues[453].Loc != scm.LocNone {
				d453 = ps.OverlayValues[453]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d4)
			d454 = d4
			_ = d454
			r105 := d4.Loc == scm.LocReg
			r106 := d4.Reg
			if r105 { ctx.ProtectReg(r106) }
			d455 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(176)}
			lbl31 := ctx.ReserveLabel()
			bbpos_3_0 := int32(-1)
			_ = bbpos_3_0
			bbpos_3_1 := int32(-1)
			_ = bbpos_3_1
			bbpos_3_2 := int32(-1)
			_ = bbpos_3_2
			bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d455 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(176)}
			ctx.EnsureDesc(&d454)
			ctx.EnsureDesc(&d454)
			var d456 scm.JITValueDesc
			if d454.Loc == scm.LocImm {
				d456 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d454.Imm.Int()))))}
			} else {
				r107 := ctx.AllocReg()
				ctx.EmitMovRegReg(r107, d454.Reg)
				ctx.EmitShlRegImm8(r107, 32)
				ctx.EmitShrRegImm8(r107, 32)
				d456 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r107}
				ctx.BindReg(r107, &d456)
			}
			var d457 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				r108 := ctx.AllocReg()
				ctx.EmitMovRegMem8(r108, fieldAddr)
				d457 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r108}
				ctx.BindReg(r108, &d457)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r109 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r109, thisptr.Reg, off)
				d457 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r109}
				ctx.BindReg(r109, &d457)
			}
			ctx.EnsureDesc(&d457)
			ctx.EnsureDesc(&d457)
			var d458 scm.JITValueDesc
			if d457.Loc == scm.LocImm {
				d458 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d457.Imm.Int()))))}
			} else {
				r110 := ctx.AllocReg()
				ctx.EmitMovRegReg(r110, d457.Reg)
				ctx.EmitShlRegImm8(r110, 56)
				ctx.EmitShrRegImm8(r110, 56)
				d458 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
				ctx.BindReg(r110, &d458)
			}
			ctx.EnsureDesc(&d456)
			ctx.EnsureDesc(&d458)
			ctx.EnsureDesc(&d456)
			ctx.EnsureDesc(&d458)
			ctx.EnsureDesc(&d456)
			ctx.EnsureDesc(&d458)
			var d459 scm.JITValueDesc
			if d456.Loc == scm.LocImm && d458.Loc == scm.LocImm {
				d459 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d456.Imm.Int() * d458.Imm.Int())}
			} else if d456.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d458.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d456.Imm.Int()))
				ctx.EmitImulInt64(scratch, d458.Reg)
				d459 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d459)
			} else if d458.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d456.Reg)
				ctx.EmitMovRegReg(scratch, d456.Reg)
				if d458.Imm.Int() >= -2147483648 && d458.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d458.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d458.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d459 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d459)
			} else {
				r111 := ctx.AllocRegExcept(d456.Reg, d458.Reg)
				ctx.EmitMovRegReg(r111, d456.Reg)
				ctx.EmitImulInt64(r111, d458.Reg)
				d459 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r111}
				ctx.BindReg(r111, &d459)
			}
			if d459.Loc == scm.LocReg && d456.Loc == scm.LocReg && d459.Reg == d456.Reg {
				ctx.TransferReg(d456.Reg)
				d456.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d456)
			ctx.FreeDesc(&d458)
			var d460 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				r112 := ctx.AllocReg()
				r113 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r112, fieldAddr)
				ctx.EmitMovRegMem64(r113, fieldAddr+8)
				d460 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r112, Reg2: r113}
				ctx.BindReg(r112, &d460)
				ctx.BindReg(r113, &d460)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				r114 := ctx.AllocReg()
				r115 := ctx.AllocReg()
				ctx.EmitMovRegMem(r114, thisptr.Reg, off)
				ctx.EmitMovRegMem(r115, thisptr.Reg, off+8)
				d460 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r114, Reg2: r115}
				ctx.BindReg(r114, &d460)
				ctx.BindReg(r115, &d460)
			}
			ctx.EnsureDesc(&d459)
			var d461 scm.JITValueDesc
			if d459.Loc == scm.LocImm {
				d461 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d459.Imm.Int() / 64)}
			} else {
				r116 := ctx.AllocRegExcept(d459.Reg)
				ctx.EmitMovRegReg(r116, d459.Reg)
				ctx.EmitShrRegImm8(r116, 6)
				d461 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r116}
				ctx.BindReg(r116, &d461)
			}
			if d461.Loc == scm.LocReg && d459.Loc == scm.LocReg && d461.Reg == d459.Reg {
				ctx.TransferReg(d459.Reg)
				d459.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d461)
			r117 := ctx.AllocReg()
			ctx.EnsureDesc(&d461)
			ctx.EnsureDesc(&d460)
			if d461.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r117, uint64(d461.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r117, d461.Reg)
				ctx.EmitShlRegImm8(r117, 3)
			}
			if d460.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d460.Imm.Int()))
				ctx.EmitAddInt64(r117, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r117, d460.Reg)
			}
			r118 := ctx.AllocRegExcept(r117)
			ctx.EmitMovRegMem(r118, r117, 0)
			ctx.FreeReg(r117)
			d462 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r118}
			ctx.BindReg(r118, &d462)
			ctx.FreeDesc(&d461)
			ctx.EnsureDesc(&d459)
			var d463 scm.JITValueDesc
			if d459.Loc == scm.LocImm {
				d463 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d459.Imm.Int() % 64)}
			} else {
				r119 := ctx.AllocRegExcept(d459.Reg)
				ctx.EmitMovRegReg(r119, d459.Reg)
				ctx.EmitAndRegImm32(r119, 63)
				d463 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
				ctx.BindReg(r119, &d463)
			}
			if d463.Loc == scm.LocReg && d459.Loc == scm.LocReg && d463.Reg == d459.Reg {
				ctx.TransferReg(d459.Reg)
				d459.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d462)
			ctx.EnsureDesc(&d463)
			var d464 scm.JITValueDesc
			if d462.Loc == scm.LocImm && d463.Loc == scm.LocImm {
				d464 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d462.Imm.Int()) << uint64(d463.Imm.Int())))}
			} else if d463.Loc == scm.LocImm {
				r120 := ctx.AllocRegExcept(d462.Reg)
				ctx.EmitMovRegReg(r120, d462.Reg)
				ctx.EmitShlRegImm8(r120, uint8(d463.Imm.Int()))
				d464 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r120}
				ctx.BindReg(r120, &d464)
			} else {
				{
					shiftSrc := d462.Reg
					r121 := ctx.AllocRegExcept(d462.Reg)
					ctx.EmitMovRegReg(r121, d462.Reg)
					shiftSrc = r121
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d463.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d463.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d463.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d464 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d464)
				}
			}
			if d464.Loc == scm.LocReg && d462.Loc == scm.LocReg && d464.Reg == d462.Reg {
				ctx.TransferReg(d462.Reg)
				d462.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d462)
			ctx.FreeDesc(&d463)
			ctx.EnsureDesc(&d459)
			var d465 scm.JITValueDesc
			if d459.Loc == scm.LocImm {
				d465 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d459.Imm.Int() % 64)}
			} else {
				r122 := ctx.AllocRegExcept(d459.Reg)
				ctx.EmitMovRegReg(r122, d459.Reg)
				ctx.EmitAndRegImm32(r122, 63)
				d465 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
				ctx.BindReg(r122, &d465)
			}
			if d465.Loc == scm.LocReg && d459.Loc == scm.LocReg && d465.Reg == d459.Reg {
				ctx.TransferReg(d459.Reg)
				d459.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d457)
			ctx.EnsureDesc(&d457)
			var d466 scm.JITValueDesc
			if d457.Loc == scm.LocImm {
				d466 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d457.Imm.Int()))))}
			} else {
				r123 := ctx.AllocReg()
				ctx.EmitMovRegReg(r123, d457.Reg)
				ctx.EmitShlRegImm8(r123, 56)
				ctx.EmitShrRegImm8(r123, 56)
				d466 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
				ctx.BindReg(r123, &d466)
			}
			ctx.EnsureDesc(&d465)
			ctx.EnsureDesc(&d466)
			ctx.EnsureDesc(&d465)
			ctx.EnsureDesc(&d466)
			ctx.EnsureDesc(&d465)
			ctx.EnsureDesc(&d466)
			var d467 scm.JITValueDesc
			if d465.Loc == scm.LocImm && d466.Loc == scm.LocImm {
				d467 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d465.Imm.Int() + d466.Imm.Int())}
			} else if d466.Loc == scm.LocImm && d466.Imm.Int() == 0 {
				r124 := ctx.AllocRegExcept(d465.Reg)
				ctx.EmitMovRegReg(r124, d465.Reg)
				d467 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
				ctx.BindReg(r124, &d467)
			} else if d465.Loc == scm.LocImm && d465.Imm.Int() == 0 {
				d467 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d466.Reg}
				ctx.BindReg(d466.Reg, &d467)
			} else if d465.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d466.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d465.Imm.Int()))
				ctx.EmitAddInt64(scratch, d466.Reg)
				d467 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d467)
			} else if d466.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d465.Reg)
				ctx.EmitMovRegReg(scratch, d465.Reg)
				if d466.Imm.Int() >= -2147483648 && d466.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d466.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d466.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d467 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d467)
			} else {
				r125 := ctx.AllocRegExcept(d465.Reg, d466.Reg)
				ctx.EmitMovRegReg(r125, d465.Reg)
				ctx.EmitAddInt64(r125, d466.Reg)
				d467 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
				ctx.BindReg(r125, &d467)
			}
			if d467.Loc == scm.LocReg && d465.Loc == scm.LocReg && d467.Reg == d465.Reg {
				ctx.TransferReg(d465.Reg)
				d465.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d465)
			ctx.FreeDesc(&d466)
			ctx.EnsureDesc(&d467)
			var d468 scm.JITValueDesc
			if d467.Loc == scm.LocImm {
				d468 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d467.Imm.Int()) > uint64(64))}
			} else {
				r126 := ctx.AllocRegExcept(d467.Reg)
				ctx.EmitCmpRegImm32(d467.Reg, 64)
				ctx.EmitSetcc(r126, scm.CcA)
				d468 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r126}
				ctx.BindReg(r126, &d468)
			}
			ctx.FreeDesc(&d467)
			d469 = d468
			ctx.EnsureDesc(&d469)
			if d469.Loc != scm.LocImm && d469.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl32 := ctx.ReserveLabel()
			lbl33 := ctx.ReserveLabel()
			lbl34 := ctx.ReserveLabel()
			lbl35 := ctx.ReserveLabel()
			if d469.Loc == scm.LocImm {
				if d469.Imm.Bool() {
					ctx.MarkLabel(lbl34)
					ctx.EmitJmp(lbl32)
				} else {
					ctx.MarkLabel(lbl35)
			ctx.EnsureDesc(&d464)
			if d464.Loc == scm.LocReg {
				ctx.ProtectReg(d464.Reg)
			} else if d464.Loc == scm.LocRegPair {
				ctx.ProtectReg(d464.Reg)
				ctx.ProtectReg(d464.Reg2)
			}
			d470 = d464
			if d470.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d470)
			ctx.EmitStoreToStack(d470, int32(bbs[2].PhiBase)+int32(0))
			if d464.Loc == scm.LocReg {
				ctx.UnprotectReg(d464.Reg)
			} else if d464.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d464.Reg)
				ctx.UnprotectReg(d464.Reg2)
			}
					ctx.EmitJmp(lbl33)
				}
			} else {
				ctx.EmitCmpRegImm32(d469.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl34)
				ctx.EmitJmp(lbl35)
				ctx.MarkLabel(lbl34)
				ctx.EmitJmp(lbl32)
				ctx.MarkLabel(lbl35)
			ctx.EnsureDesc(&d464)
			if d464.Loc == scm.LocReg {
				ctx.ProtectReg(d464.Reg)
			} else if d464.Loc == scm.LocRegPair {
				ctx.ProtectReg(d464.Reg)
				ctx.ProtectReg(d464.Reg2)
			}
			d471 = d464
			if d471.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d471)
			ctx.EmitStoreToStack(d471, int32(bbs[2].PhiBase)+int32(0))
			if d464.Loc == scm.LocReg {
				ctx.UnprotectReg(d464.Reg)
			} else if d464.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d464.Reg)
				ctx.UnprotectReg(d464.Reg2)
			}
				ctx.EmitJmp(lbl33)
			}
			ctx.FreeDesc(&d468)
			bbpos_3_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl33)
			ctx.ResolveFixups()
			d455 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(176)}
			ctx.EnsureDesc(&d457)
			ctx.EnsureDesc(&d457)
			var d472 scm.JITValueDesc
			if d457.Loc == scm.LocImm {
				d472 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d457.Imm.Int()))))}
			} else {
				r127 := ctx.AllocReg()
				ctx.EmitMovRegReg(r127, d457.Reg)
				ctx.EmitShlRegImm8(r127, 56)
				ctx.EmitShrRegImm8(r127, 56)
				d472 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
				ctx.BindReg(r127, &d472)
			}
			d473 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d472)
			ctx.EnsureDesc(&d473)
			ctx.EnsureDesc(&d472)
			ctx.EnsureDesc(&d473)
			ctx.EnsureDesc(&d472)
			var d474 scm.JITValueDesc
			if d473.Loc == scm.LocImm && d472.Loc == scm.LocImm {
				d474 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d473.Imm.Int() - d472.Imm.Int())}
			} else if d472.Loc == scm.LocImm && d472.Imm.Int() == 0 {
				r128 := ctx.AllocRegExcept(d473.Reg)
				ctx.EmitMovRegReg(r128, d473.Reg)
				d474 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
				ctx.BindReg(r128, &d474)
			} else if d473.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d472.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d473.Imm.Int()))
				ctx.EmitSubInt64(scratch, d472.Reg)
				d474 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d474)
			} else if d472.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d473.Reg)
				ctx.EmitMovRegReg(scratch, d473.Reg)
				if d472.Imm.Int() >= -2147483648 && d472.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d472.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d472.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d474 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d474)
			} else {
				r129 := ctx.AllocRegExcept(d473.Reg, d472.Reg)
				ctx.EmitMovRegReg(r129, d473.Reg)
				ctx.EmitSubInt64(r129, d472.Reg)
				d474 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
				ctx.BindReg(r129, &d474)
			}
			if d474.Loc == scm.LocReg && d473.Loc == scm.LocReg && d474.Reg == d473.Reg {
				ctx.TransferReg(d473.Reg)
				d473.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d472)
			ctx.EnsureDesc(&d455)
			ctx.EnsureDesc(&d474)
			var d475 scm.JITValueDesc
			if d455.Loc == scm.LocImm && d474.Loc == scm.LocImm {
				d475 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d455.Imm.Int()) >> uint64(d474.Imm.Int())))}
			} else if d474.Loc == scm.LocImm {
				r130 := ctx.AllocRegExcept(d455.Reg)
				ctx.EmitMovRegReg(r130, d455.Reg)
				ctx.EmitShrRegImm8(r130, uint8(d474.Imm.Int()))
				d475 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
				ctx.BindReg(r130, &d475)
			} else {
				{
					shiftSrc := d455.Reg
					r131 := ctx.AllocRegExcept(d455.Reg)
					ctx.EmitMovRegReg(r131, d455.Reg)
					shiftSrc = r131
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d474.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d474.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d474.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d475 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d475)
				}
			}
			if d475.Loc == scm.LocReg && d455.Loc == scm.LocReg && d475.Reg == d455.Reg {
				ctx.TransferReg(d455.Reg)
				d455.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d455)
			ctx.FreeDesc(&d474)
			r132 := ctx.AllocReg()
			ctx.EnsureDesc(&d475)
			ctx.EnsureDesc(&d475)
			if d475.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r132, d475)
			}
			ctx.EmitJmp(lbl31)
			bbpos_3_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl32)
			ctx.ResolveFixups()
			d455 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(176)}
			ctx.EnsureDesc(&d459)
			var d476 scm.JITValueDesc
			if d459.Loc == scm.LocImm {
				d476 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d459.Imm.Int() / 64)}
			} else {
				r133 := ctx.AllocRegExcept(d459.Reg)
				ctx.EmitMovRegReg(r133, d459.Reg)
				ctx.EmitShrRegImm8(r133, 6)
				d476 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
				ctx.BindReg(r133, &d476)
			}
			if d476.Loc == scm.LocReg && d459.Loc == scm.LocReg && d476.Reg == d459.Reg {
				ctx.TransferReg(d459.Reg)
				d459.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d476)
			ctx.EnsureDesc(&d476)
			var d477 scm.JITValueDesc
			if d476.Loc == scm.LocImm {
				d477 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d476.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d476.Reg)
				ctx.EmitMovRegReg(scratch, d476.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d477 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d477)
			}
			if d477.Loc == scm.LocReg && d476.Loc == scm.LocReg && d477.Reg == d476.Reg {
				ctx.TransferReg(d476.Reg)
				d476.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d476)
			ctx.EnsureDesc(&d477)
			r134 := ctx.AllocReg()
			ctx.EnsureDesc(&d477)
			ctx.EnsureDesc(&d460)
			if d477.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r134, uint64(d477.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r134, d477.Reg)
				ctx.EmitShlRegImm8(r134, 3)
			}
			if d460.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d460.Imm.Int()))
				ctx.EmitAddInt64(r134, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r134, d460.Reg)
			}
			r135 := ctx.AllocRegExcept(r134)
			ctx.EmitMovRegMem(r135, r134, 0)
			ctx.FreeReg(r134)
			d478 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r135}
			ctx.BindReg(r135, &d478)
			ctx.FreeDesc(&d477)
			ctx.EnsureDesc(&d459)
			var d479 scm.JITValueDesc
			if d459.Loc == scm.LocImm {
				d479 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d459.Imm.Int() % 64)}
			} else {
				r136 := ctx.AllocRegExcept(d459.Reg)
				ctx.EmitMovRegReg(r136, d459.Reg)
				ctx.EmitAndRegImm32(r136, 63)
				d479 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r136}
				ctx.BindReg(r136, &d479)
			}
			if d479.Loc == scm.LocReg && d459.Loc == scm.LocReg && d479.Reg == d459.Reg {
				ctx.TransferReg(d459.Reg)
				d459.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d459)
			d480 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d479)
			ctx.EnsureDesc(&d480)
			ctx.EnsureDesc(&d479)
			ctx.EnsureDesc(&d480)
			ctx.EnsureDesc(&d479)
			var d481 scm.JITValueDesc
			if d480.Loc == scm.LocImm && d479.Loc == scm.LocImm {
				d481 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d480.Imm.Int() - d479.Imm.Int())}
			} else if d479.Loc == scm.LocImm && d479.Imm.Int() == 0 {
				r137 := ctx.AllocRegExcept(d480.Reg)
				ctx.EmitMovRegReg(r137, d480.Reg)
				d481 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r137}
				ctx.BindReg(r137, &d481)
			} else if d480.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d479.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d480.Imm.Int()))
				ctx.EmitSubInt64(scratch, d479.Reg)
				d481 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d481)
			} else if d479.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d480.Reg)
				ctx.EmitMovRegReg(scratch, d480.Reg)
				if d479.Imm.Int() >= -2147483648 && d479.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d479.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d479.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d481 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d481)
			} else {
				r138 := ctx.AllocRegExcept(d480.Reg, d479.Reg)
				ctx.EmitMovRegReg(r138, d480.Reg)
				ctx.EmitSubInt64(r138, d479.Reg)
				d481 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
				ctx.BindReg(r138, &d481)
			}
			if d481.Loc == scm.LocReg && d480.Loc == scm.LocReg && d481.Reg == d480.Reg {
				ctx.TransferReg(d480.Reg)
				d480.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d479)
			ctx.EnsureDesc(&d478)
			ctx.EnsureDesc(&d481)
			var d482 scm.JITValueDesc
			if d478.Loc == scm.LocImm && d481.Loc == scm.LocImm {
				d482 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d478.Imm.Int()) >> uint64(d481.Imm.Int())))}
			} else if d481.Loc == scm.LocImm {
				r139 := ctx.AllocRegExcept(d478.Reg)
				ctx.EmitMovRegReg(r139, d478.Reg)
				ctx.EmitShrRegImm8(r139, uint8(d481.Imm.Int()))
				d482 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r139}
				ctx.BindReg(r139, &d482)
			} else {
				{
					shiftSrc := d478.Reg
					r140 := ctx.AllocRegExcept(d478.Reg)
					ctx.EmitMovRegReg(r140, d478.Reg)
					shiftSrc = r140
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d481.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d481.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d481.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d482 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d482)
				}
			}
			if d482.Loc == scm.LocReg && d478.Loc == scm.LocReg && d482.Reg == d478.Reg {
				ctx.TransferReg(d478.Reg)
				d478.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d478)
			ctx.FreeDesc(&d481)
			ctx.EnsureDesc(&d464)
			ctx.EnsureDesc(&d482)
			var d483 scm.JITValueDesc
			if d464.Loc == scm.LocImm && d482.Loc == scm.LocImm {
				d483 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d464.Imm.Int() | d482.Imm.Int())}
			} else if d464.Loc == scm.LocImm && d464.Imm.Int() == 0 {
				d483 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d482.Reg}
				ctx.BindReg(d482.Reg, &d483)
			} else if d482.Loc == scm.LocImm && d482.Imm.Int() == 0 {
				r141 := ctx.AllocRegExcept(d464.Reg)
				ctx.EmitMovRegReg(r141, d464.Reg)
				d483 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
				ctx.BindReg(r141, &d483)
			} else if d464.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d482.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d464.Imm.Int()))
				ctx.EmitOrInt64(scratch, d482.Reg)
				d483 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d483)
			} else if d482.Loc == scm.LocImm {
				r142 := ctx.AllocRegExcept(d464.Reg)
				ctx.EmitMovRegReg(r142, d464.Reg)
				if d482.Imm.Int() >= -2147483648 && d482.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r142, int32(d482.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d482.Imm.Int()))
					ctx.EmitOrInt64(r142, scm.RegR11)
				}
				d483 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r142}
				ctx.BindReg(r142, &d483)
			} else {
				r143 := ctx.AllocRegExcept(d464.Reg, d482.Reg)
				ctx.EmitMovRegReg(r143, d464.Reg)
				ctx.EmitOrInt64(r143, d482.Reg)
				d483 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r143}
				ctx.BindReg(r143, &d483)
			}
			if d483.Loc == scm.LocReg && d464.Loc == scm.LocReg && d483.Reg == d464.Reg {
				ctx.TransferReg(d464.Reg)
				d464.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d482)
			ctx.EnsureDesc(&d483)
			if d483.Loc == scm.LocReg {
				ctx.ProtectReg(d483.Reg)
			} else if d483.Loc == scm.LocRegPair {
				ctx.ProtectReg(d483.Reg)
				ctx.ProtectReg(d483.Reg2)
			}
			d484 = d483
			if d484.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d484)
			ctx.EmitStoreToStack(d484, int32(bbs[2].PhiBase)+int32(0))
			if d483.Loc == scm.LocReg {
				ctx.UnprotectReg(d483.Reg)
			} else if d483.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d483.Reg)
				ctx.UnprotectReg(d483.Reg2)
			}
			ctx.EmitJmp(lbl33)
			ctx.MarkLabel(lbl31)
			d485 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r132}
			ctx.BindReg(r132, &d485)
			ctx.BindReg(r132, &d485)
			if r105 { ctx.UnprotectReg(r106) }
			ctx.EnsureDesc(&d485)
			ctx.EnsureDesc(&d485)
			var d486 scm.JITValueDesc
			if d485.Loc == scm.LocImm {
				d486 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d485.Imm.Int()))))}
			} else {
				r144 := ctx.AllocReg()
				ctx.EmitMovRegReg(r144, d485.Reg)
				d486 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
				ctx.BindReg(r144, &d486)
			}
			ctx.FreeDesc(&d485)
			ctx.EnsureDesc(&d486)
			ctx.EnsureDesc(&d57)
			ctx.EnsureDesc(&d486)
			ctx.EnsureDesc(&d57)
			ctx.EnsureDesc(&d486)
			ctx.EnsureDesc(&d57)
			var d487 scm.JITValueDesc
			if d486.Loc == scm.LocImm && d57.Loc == scm.LocImm {
				d487 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d486.Imm.Int() + d57.Imm.Int())}
			} else if d57.Loc == scm.LocImm && d57.Imm.Int() == 0 {
				r145 := ctx.AllocRegExcept(d486.Reg)
				ctx.EmitMovRegReg(r145, d486.Reg)
				d487 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r145}
				ctx.BindReg(r145, &d487)
			} else if d486.Loc == scm.LocImm && d486.Imm.Int() == 0 {
				d487 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d57.Reg}
				ctx.BindReg(d57.Reg, &d487)
			} else if d486.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d57.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d486.Imm.Int()))
				ctx.EmitAddInt64(scratch, d57.Reg)
				d487 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d487)
			} else if d57.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d486.Reg)
				ctx.EmitMovRegReg(scratch, d486.Reg)
				if d57.Imm.Int() >= -2147483648 && d57.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d57.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d57.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d487 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d487)
			} else {
				r146 := ctx.AllocRegExcept(d486.Reg, d57.Reg)
				ctx.EmitMovRegReg(r146, d486.Reg)
				ctx.EmitAddInt64(r146, d57.Reg)
				d487 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r146}
				ctx.BindReg(r146, &d487)
			}
			if d487.Loc == scm.LocReg && d486.Loc == scm.LocReg && d487.Reg == d486.Reg {
				ctx.TransferReg(d486.Reg)
				d486.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d486)
			ctx.EnsureDesc(&d487)
			ctx.EnsureDesc(&d487)
			var d488 scm.JITValueDesc
			if d487.Loc == scm.LocImm {
				d488 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d487.Imm.Int()))))}
			} else {
				r147 := ctx.AllocReg()
				ctx.EmitMovRegReg(r147, d487.Reg)
				ctx.EmitShlRegImm8(r147, 32)
				ctx.EmitShrRegImm8(r147, 32)
				d488 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r147}
				ctx.BindReg(r147, &d488)
			}
			ctx.FreeDesc(&d487)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d488)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d488)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d488)
			var d489 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d488.Loc == scm.LocImm {
				d489 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d488.Imm.Int()))}
			} else if d488.Loc == scm.LocImm {
				r148 := ctx.AllocRegExcept(idxInt.Reg)
				if d488.Imm.Int() >= -2147483648 && d488.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(idxInt.Reg, int32(d488.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d488.Imm.Int()))
					ctx.EmitCmpInt64(idxInt.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r148, scm.CcB)
				d489 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r148}
				ctx.BindReg(r148, &d489)
			} else if idxInt.Loc == scm.LocImm {
				r149 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d488.Reg)
				ctx.EmitSetcc(r149, scm.CcB)
				d489 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r149}
				ctx.BindReg(r149, &d489)
			} else {
				r150 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.EmitCmpInt64(idxInt.Reg, d488.Reg)
				ctx.EmitSetcc(r150, scm.CcB)
				d489 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r150}
				ctx.BindReg(r150, &d489)
			}
			ctx.FreeDesc(&d488)
			d490 = d489
			ctx.EnsureDesc(&d490)
			if d490.Loc != scm.LocImm && d490.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d490.Loc == scm.LocImm {
				if d490.Imm.Bool() {
			ps491 := scm.PhiState{General: ps.General}
			ps491.OverlayValues = make([]scm.JITValueDesc, 491)
			ps491.OverlayValues[0] = d0
			ps491.OverlayValues[1] = d1
			ps491.OverlayValues[2] = d2
			ps491.OverlayValues[3] = d3
			ps491.OverlayValues[4] = d4
			ps491.OverlayValues[5] = d5
			ps491.OverlayValues[6] = d6
			ps491.OverlayValues[7] = d7
			ps491.OverlayValues[8] = d8
			ps491.OverlayValues[9] = d9
			ps491.OverlayValues[10] = d10
			ps491.OverlayValues[11] = d11
			ps491.OverlayValues[12] = d12
			ps491.OverlayValues[13] = d13
			ps491.OverlayValues[14] = d14
			ps491.OverlayValues[15] = d15
			ps491.OverlayValues[16] = d16
			ps491.OverlayValues[18] = d18
			ps491.OverlayValues[19] = d19
			ps491.OverlayValues[20] = d20
			ps491.OverlayValues[21] = d21
			ps491.OverlayValues[22] = d22
			ps491.OverlayValues[23] = d23
			ps491.OverlayValues[24] = d24
			ps491.OverlayValues[25] = d25
			ps491.OverlayValues[26] = d26
			ps491.OverlayValues[27] = d27
			ps491.OverlayValues[28] = d28
			ps491.OverlayValues[29] = d29
			ps491.OverlayValues[30] = d30
			ps491.OverlayValues[31] = d31
			ps491.OverlayValues[32] = d32
			ps491.OverlayValues[33] = d33
			ps491.OverlayValues[34] = d34
			ps491.OverlayValues[35] = d35
			ps491.OverlayValues[36] = d36
			ps491.OverlayValues[37] = d37
			ps491.OverlayValues[38] = d38
			ps491.OverlayValues[39] = d39
			ps491.OverlayValues[40] = d40
			ps491.OverlayValues[41] = d41
			ps491.OverlayValues[42] = d42
			ps491.OverlayValues[43] = d43
			ps491.OverlayValues[44] = d44
			ps491.OverlayValues[45] = d45
			ps491.OverlayValues[46] = d46
			ps491.OverlayValues[47] = d47
			ps491.OverlayValues[48] = d48
			ps491.OverlayValues[49] = d49
			ps491.OverlayValues[50] = d50
			ps491.OverlayValues[51] = d51
			ps491.OverlayValues[52] = d52
			ps491.OverlayValues[53] = d53
			ps491.OverlayValues[54] = d54
			ps491.OverlayValues[55] = d55
			ps491.OverlayValues[56] = d56
			ps491.OverlayValues[57] = d57
			ps491.OverlayValues[58] = d58
			ps491.OverlayValues[59] = d59
			ps491.OverlayValues[60] = d60
			ps491.OverlayValues[61] = d61
			ps491.OverlayValues[64] = d64
			ps491.OverlayValues[65] = d65
			ps491.OverlayValues[66] = d66
			ps491.OverlayValues[134] = d134
			ps491.OverlayValues[135] = d135
			ps491.OverlayValues[136] = d136
			ps491.OverlayValues[137] = d137
			ps491.OverlayValues[138] = d138
			ps491.OverlayValues[139] = d139
			ps491.OverlayValues[140] = d140
			ps491.OverlayValues[141] = d141
			ps491.OverlayValues[142] = d142
			ps491.OverlayValues[143] = d143
			ps491.OverlayValues[144] = d144
			ps491.OverlayValues[145] = d145
			ps491.OverlayValues[146] = d146
			ps491.OverlayValues[147] = d147
			ps491.OverlayValues[148] = d148
			ps491.OverlayValues[149] = d149
			ps491.OverlayValues[150] = d150
			ps491.OverlayValues[151] = d151
			ps491.OverlayValues[152] = d152
			ps491.OverlayValues[153] = d153
			ps491.OverlayValues[154] = d154
			ps491.OverlayValues[155] = d155
			ps491.OverlayValues[156] = d156
			ps491.OverlayValues[157] = d157
			ps491.OverlayValues[158] = d158
			ps491.OverlayValues[159] = d159
			ps491.OverlayValues[160] = d160
			ps491.OverlayValues[161] = d161
			ps491.OverlayValues[162] = d162
			ps491.OverlayValues[163] = d163
			ps491.OverlayValues[164] = d164
			ps491.OverlayValues[165] = d165
			ps491.OverlayValues[166] = d166
			ps491.OverlayValues[167] = d167
			ps491.OverlayValues[168] = d168
			ps491.OverlayValues[169] = d169
			ps491.OverlayValues[170] = d170
			ps491.OverlayValues[171] = d171
			ps491.OverlayValues[172] = d172
			ps491.OverlayValues[175] = d175
			ps491.OverlayValues[283] = d283
			ps491.OverlayValues[284] = d284
			ps491.OverlayValues[285] = d285
			ps491.OverlayValues[286] = d286
			ps491.OverlayValues[287] = d287
			ps491.OverlayValues[288] = d288
			ps491.OverlayValues[289] = d289
			ps491.OverlayValues[290] = d290
			ps491.OverlayValues[292] = d292
			ps491.OverlayValues[293] = d293
			ps491.OverlayValues[294] = d294
			ps491.OverlayValues[295] = d295
			ps491.OverlayValues[296] = d296
			ps491.OverlayValues[297] = d297
			ps491.OverlayValues[298] = d298
			ps491.OverlayValues[299] = d299
			ps491.OverlayValues[300] = d300
			ps491.OverlayValues[301] = d301
			ps491.OverlayValues[303] = d303
			ps491.OverlayValues[305] = d305
			ps491.OverlayValues[306] = d306
			ps491.OverlayValues[307] = d307
			ps491.OverlayValues[308] = d308
			ps491.OverlayValues[309] = d309
			ps491.OverlayValues[312] = d312
			ps491.OverlayValues[443] = d443
			ps491.OverlayValues[444] = d444
			ps491.OverlayValues[445] = d445
			ps491.OverlayValues[446] = d446
			ps491.OverlayValues[447] = d447
			ps491.OverlayValues[448] = d448
			ps491.OverlayValues[449] = d449
			ps491.OverlayValues[451] = d451
			ps491.OverlayValues[452] = d452
			ps491.OverlayValues[453] = d453
			ps491.OverlayValues[454] = d454
			ps491.OverlayValues[455] = d455
			ps491.OverlayValues[456] = d456
			ps491.OverlayValues[457] = d457
			ps491.OverlayValues[458] = d458
			ps491.OverlayValues[459] = d459
			ps491.OverlayValues[460] = d460
			ps491.OverlayValues[461] = d461
			ps491.OverlayValues[462] = d462
			ps491.OverlayValues[463] = d463
			ps491.OverlayValues[464] = d464
			ps491.OverlayValues[465] = d465
			ps491.OverlayValues[466] = d466
			ps491.OverlayValues[467] = d467
			ps491.OverlayValues[468] = d468
			ps491.OverlayValues[469] = d469
			ps491.OverlayValues[470] = d470
			ps491.OverlayValues[471] = d471
			ps491.OverlayValues[472] = d472
			ps491.OverlayValues[473] = d473
			ps491.OverlayValues[474] = d474
			ps491.OverlayValues[475] = d475
			ps491.OverlayValues[476] = d476
			ps491.OverlayValues[477] = d477
			ps491.OverlayValues[478] = d478
			ps491.OverlayValues[479] = d479
			ps491.OverlayValues[480] = d480
			ps491.OverlayValues[481] = d481
			ps491.OverlayValues[482] = d482
			ps491.OverlayValues[483] = d483
			ps491.OverlayValues[484] = d484
			ps491.OverlayValues[485] = d485
			ps491.OverlayValues[486] = d486
			ps491.OverlayValues[487] = d487
			ps491.OverlayValues[488] = d488
			ps491.OverlayValues[489] = d489
			ps491.OverlayValues[490] = d490
					return bbs[7].RenderPS(ps491)
				}
			ps492 := scm.PhiState{General: ps.General}
			ps492.OverlayValues = make([]scm.JITValueDesc, 491)
			ps492.OverlayValues[0] = d0
			ps492.OverlayValues[1] = d1
			ps492.OverlayValues[2] = d2
			ps492.OverlayValues[3] = d3
			ps492.OverlayValues[4] = d4
			ps492.OverlayValues[5] = d5
			ps492.OverlayValues[6] = d6
			ps492.OverlayValues[7] = d7
			ps492.OverlayValues[8] = d8
			ps492.OverlayValues[9] = d9
			ps492.OverlayValues[10] = d10
			ps492.OverlayValues[11] = d11
			ps492.OverlayValues[12] = d12
			ps492.OverlayValues[13] = d13
			ps492.OverlayValues[14] = d14
			ps492.OverlayValues[15] = d15
			ps492.OverlayValues[16] = d16
			ps492.OverlayValues[18] = d18
			ps492.OverlayValues[19] = d19
			ps492.OverlayValues[20] = d20
			ps492.OverlayValues[21] = d21
			ps492.OverlayValues[22] = d22
			ps492.OverlayValues[23] = d23
			ps492.OverlayValues[24] = d24
			ps492.OverlayValues[25] = d25
			ps492.OverlayValues[26] = d26
			ps492.OverlayValues[27] = d27
			ps492.OverlayValues[28] = d28
			ps492.OverlayValues[29] = d29
			ps492.OverlayValues[30] = d30
			ps492.OverlayValues[31] = d31
			ps492.OverlayValues[32] = d32
			ps492.OverlayValues[33] = d33
			ps492.OverlayValues[34] = d34
			ps492.OverlayValues[35] = d35
			ps492.OverlayValues[36] = d36
			ps492.OverlayValues[37] = d37
			ps492.OverlayValues[38] = d38
			ps492.OverlayValues[39] = d39
			ps492.OverlayValues[40] = d40
			ps492.OverlayValues[41] = d41
			ps492.OverlayValues[42] = d42
			ps492.OverlayValues[43] = d43
			ps492.OverlayValues[44] = d44
			ps492.OverlayValues[45] = d45
			ps492.OverlayValues[46] = d46
			ps492.OverlayValues[47] = d47
			ps492.OverlayValues[48] = d48
			ps492.OverlayValues[49] = d49
			ps492.OverlayValues[50] = d50
			ps492.OverlayValues[51] = d51
			ps492.OverlayValues[52] = d52
			ps492.OverlayValues[53] = d53
			ps492.OverlayValues[54] = d54
			ps492.OverlayValues[55] = d55
			ps492.OverlayValues[56] = d56
			ps492.OverlayValues[57] = d57
			ps492.OverlayValues[58] = d58
			ps492.OverlayValues[59] = d59
			ps492.OverlayValues[60] = d60
			ps492.OverlayValues[61] = d61
			ps492.OverlayValues[64] = d64
			ps492.OverlayValues[65] = d65
			ps492.OverlayValues[66] = d66
			ps492.OverlayValues[134] = d134
			ps492.OverlayValues[135] = d135
			ps492.OverlayValues[136] = d136
			ps492.OverlayValues[137] = d137
			ps492.OverlayValues[138] = d138
			ps492.OverlayValues[139] = d139
			ps492.OverlayValues[140] = d140
			ps492.OverlayValues[141] = d141
			ps492.OverlayValues[142] = d142
			ps492.OverlayValues[143] = d143
			ps492.OverlayValues[144] = d144
			ps492.OverlayValues[145] = d145
			ps492.OverlayValues[146] = d146
			ps492.OverlayValues[147] = d147
			ps492.OverlayValues[148] = d148
			ps492.OverlayValues[149] = d149
			ps492.OverlayValues[150] = d150
			ps492.OverlayValues[151] = d151
			ps492.OverlayValues[152] = d152
			ps492.OverlayValues[153] = d153
			ps492.OverlayValues[154] = d154
			ps492.OverlayValues[155] = d155
			ps492.OverlayValues[156] = d156
			ps492.OverlayValues[157] = d157
			ps492.OverlayValues[158] = d158
			ps492.OverlayValues[159] = d159
			ps492.OverlayValues[160] = d160
			ps492.OverlayValues[161] = d161
			ps492.OverlayValues[162] = d162
			ps492.OverlayValues[163] = d163
			ps492.OverlayValues[164] = d164
			ps492.OverlayValues[165] = d165
			ps492.OverlayValues[166] = d166
			ps492.OverlayValues[167] = d167
			ps492.OverlayValues[168] = d168
			ps492.OverlayValues[169] = d169
			ps492.OverlayValues[170] = d170
			ps492.OverlayValues[171] = d171
			ps492.OverlayValues[172] = d172
			ps492.OverlayValues[175] = d175
			ps492.OverlayValues[283] = d283
			ps492.OverlayValues[284] = d284
			ps492.OverlayValues[285] = d285
			ps492.OverlayValues[286] = d286
			ps492.OverlayValues[287] = d287
			ps492.OverlayValues[288] = d288
			ps492.OverlayValues[289] = d289
			ps492.OverlayValues[290] = d290
			ps492.OverlayValues[292] = d292
			ps492.OverlayValues[293] = d293
			ps492.OverlayValues[294] = d294
			ps492.OverlayValues[295] = d295
			ps492.OverlayValues[296] = d296
			ps492.OverlayValues[297] = d297
			ps492.OverlayValues[298] = d298
			ps492.OverlayValues[299] = d299
			ps492.OverlayValues[300] = d300
			ps492.OverlayValues[301] = d301
			ps492.OverlayValues[303] = d303
			ps492.OverlayValues[305] = d305
			ps492.OverlayValues[306] = d306
			ps492.OverlayValues[307] = d307
			ps492.OverlayValues[308] = d308
			ps492.OverlayValues[309] = d309
			ps492.OverlayValues[312] = d312
			ps492.OverlayValues[443] = d443
			ps492.OverlayValues[444] = d444
			ps492.OverlayValues[445] = d445
			ps492.OverlayValues[446] = d446
			ps492.OverlayValues[447] = d447
			ps492.OverlayValues[448] = d448
			ps492.OverlayValues[449] = d449
			ps492.OverlayValues[451] = d451
			ps492.OverlayValues[452] = d452
			ps492.OverlayValues[453] = d453
			ps492.OverlayValues[454] = d454
			ps492.OverlayValues[455] = d455
			ps492.OverlayValues[456] = d456
			ps492.OverlayValues[457] = d457
			ps492.OverlayValues[458] = d458
			ps492.OverlayValues[459] = d459
			ps492.OverlayValues[460] = d460
			ps492.OverlayValues[461] = d461
			ps492.OverlayValues[462] = d462
			ps492.OverlayValues[463] = d463
			ps492.OverlayValues[464] = d464
			ps492.OverlayValues[465] = d465
			ps492.OverlayValues[466] = d466
			ps492.OverlayValues[467] = d467
			ps492.OverlayValues[468] = d468
			ps492.OverlayValues[469] = d469
			ps492.OverlayValues[470] = d470
			ps492.OverlayValues[471] = d471
			ps492.OverlayValues[472] = d472
			ps492.OverlayValues[473] = d473
			ps492.OverlayValues[474] = d474
			ps492.OverlayValues[475] = d475
			ps492.OverlayValues[476] = d476
			ps492.OverlayValues[477] = d477
			ps492.OverlayValues[478] = d478
			ps492.OverlayValues[479] = d479
			ps492.OverlayValues[480] = d480
			ps492.OverlayValues[481] = d481
			ps492.OverlayValues[482] = d482
			ps492.OverlayValues[483] = d483
			ps492.OverlayValues[484] = d484
			ps492.OverlayValues[485] = d485
			ps492.OverlayValues[486] = d486
			ps492.OverlayValues[487] = d487
			ps492.OverlayValues[488] = d488
			ps492.OverlayValues[489] = d489
			ps492.OverlayValues[490] = d490
				return bbs[9].RenderPS(ps492)
			}
			if !ps.General {
				ps.General = true
				return bbs[6].RenderPS(ps)
			}
			lbl36 := ctx.ReserveLabel()
			lbl37 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d490.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl36)
			ctx.EmitJmp(lbl37)
			ctx.MarkLabel(lbl36)
			ctx.EmitJmp(lbl8)
			ctx.MarkLabel(lbl37)
			ctx.EmitJmp(lbl10)
			ps493 := scm.PhiState{General: true}
			ps493.OverlayValues = make([]scm.JITValueDesc, 491)
			ps493.OverlayValues[0] = d0
			ps493.OverlayValues[1] = d1
			ps493.OverlayValues[2] = d2
			ps493.OverlayValues[3] = d3
			ps493.OverlayValues[4] = d4
			ps493.OverlayValues[5] = d5
			ps493.OverlayValues[6] = d6
			ps493.OverlayValues[7] = d7
			ps493.OverlayValues[8] = d8
			ps493.OverlayValues[9] = d9
			ps493.OverlayValues[10] = d10
			ps493.OverlayValues[11] = d11
			ps493.OverlayValues[12] = d12
			ps493.OverlayValues[13] = d13
			ps493.OverlayValues[14] = d14
			ps493.OverlayValues[15] = d15
			ps493.OverlayValues[16] = d16
			ps493.OverlayValues[18] = d18
			ps493.OverlayValues[19] = d19
			ps493.OverlayValues[20] = d20
			ps493.OverlayValues[21] = d21
			ps493.OverlayValues[22] = d22
			ps493.OverlayValues[23] = d23
			ps493.OverlayValues[24] = d24
			ps493.OverlayValues[25] = d25
			ps493.OverlayValues[26] = d26
			ps493.OverlayValues[27] = d27
			ps493.OverlayValues[28] = d28
			ps493.OverlayValues[29] = d29
			ps493.OverlayValues[30] = d30
			ps493.OverlayValues[31] = d31
			ps493.OverlayValues[32] = d32
			ps493.OverlayValues[33] = d33
			ps493.OverlayValues[34] = d34
			ps493.OverlayValues[35] = d35
			ps493.OverlayValues[36] = d36
			ps493.OverlayValues[37] = d37
			ps493.OverlayValues[38] = d38
			ps493.OverlayValues[39] = d39
			ps493.OverlayValues[40] = d40
			ps493.OverlayValues[41] = d41
			ps493.OverlayValues[42] = d42
			ps493.OverlayValues[43] = d43
			ps493.OverlayValues[44] = d44
			ps493.OverlayValues[45] = d45
			ps493.OverlayValues[46] = d46
			ps493.OverlayValues[47] = d47
			ps493.OverlayValues[48] = d48
			ps493.OverlayValues[49] = d49
			ps493.OverlayValues[50] = d50
			ps493.OverlayValues[51] = d51
			ps493.OverlayValues[52] = d52
			ps493.OverlayValues[53] = d53
			ps493.OverlayValues[54] = d54
			ps493.OverlayValues[55] = d55
			ps493.OverlayValues[56] = d56
			ps493.OverlayValues[57] = d57
			ps493.OverlayValues[58] = d58
			ps493.OverlayValues[59] = d59
			ps493.OverlayValues[60] = d60
			ps493.OverlayValues[61] = d61
			ps493.OverlayValues[64] = d64
			ps493.OverlayValues[65] = d65
			ps493.OverlayValues[66] = d66
			ps493.OverlayValues[134] = d134
			ps493.OverlayValues[135] = d135
			ps493.OverlayValues[136] = d136
			ps493.OverlayValues[137] = d137
			ps493.OverlayValues[138] = d138
			ps493.OverlayValues[139] = d139
			ps493.OverlayValues[140] = d140
			ps493.OverlayValues[141] = d141
			ps493.OverlayValues[142] = d142
			ps493.OverlayValues[143] = d143
			ps493.OverlayValues[144] = d144
			ps493.OverlayValues[145] = d145
			ps493.OverlayValues[146] = d146
			ps493.OverlayValues[147] = d147
			ps493.OverlayValues[148] = d148
			ps493.OverlayValues[149] = d149
			ps493.OverlayValues[150] = d150
			ps493.OverlayValues[151] = d151
			ps493.OverlayValues[152] = d152
			ps493.OverlayValues[153] = d153
			ps493.OverlayValues[154] = d154
			ps493.OverlayValues[155] = d155
			ps493.OverlayValues[156] = d156
			ps493.OverlayValues[157] = d157
			ps493.OverlayValues[158] = d158
			ps493.OverlayValues[159] = d159
			ps493.OverlayValues[160] = d160
			ps493.OverlayValues[161] = d161
			ps493.OverlayValues[162] = d162
			ps493.OverlayValues[163] = d163
			ps493.OverlayValues[164] = d164
			ps493.OverlayValues[165] = d165
			ps493.OverlayValues[166] = d166
			ps493.OverlayValues[167] = d167
			ps493.OverlayValues[168] = d168
			ps493.OverlayValues[169] = d169
			ps493.OverlayValues[170] = d170
			ps493.OverlayValues[171] = d171
			ps493.OverlayValues[172] = d172
			ps493.OverlayValues[175] = d175
			ps493.OverlayValues[283] = d283
			ps493.OverlayValues[284] = d284
			ps493.OverlayValues[285] = d285
			ps493.OverlayValues[286] = d286
			ps493.OverlayValues[287] = d287
			ps493.OverlayValues[288] = d288
			ps493.OverlayValues[289] = d289
			ps493.OverlayValues[290] = d290
			ps493.OverlayValues[292] = d292
			ps493.OverlayValues[293] = d293
			ps493.OverlayValues[294] = d294
			ps493.OverlayValues[295] = d295
			ps493.OverlayValues[296] = d296
			ps493.OverlayValues[297] = d297
			ps493.OverlayValues[298] = d298
			ps493.OverlayValues[299] = d299
			ps493.OverlayValues[300] = d300
			ps493.OverlayValues[301] = d301
			ps493.OverlayValues[303] = d303
			ps493.OverlayValues[305] = d305
			ps493.OverlayValues[306] = d306
			ps493.OverlayValues[307] = d307
			ps493.OverlayValues[308] = d308
			ps493.OverlayValues[309] = d309
			ps493.OverlayValues[312] = d312
			ps493.OverlayValues[443] = d443
			ps493.OverlayValues[444] = d444
			ps493.OverlayValues[445] = d445
			ps493.OverlayValues[446] = d446
			ps493.OverlayValues[447] = d447
			ps493.OverlayValues[448] = d448
			ps493.OverlayValues[449] = d449
			ps493.OverlayValues[451] = d451
			ps493.OverlayValues[452] = d452
			ps493.OverlayValues[453] = d453
			ps493.OverlayValues[454] = d454
			ps493.OverlayValues[455] = d455
			ps493.OverlayValues[456] = d456
			ps493.OverlayValues[457] = d457
			ps493.OverlayValues[458] = d458
			ps493.OverlayValues[459] = d459
			ps493.OverlayValues[460] = d460
			ps493.OverlayValues[461] = d461
			ps493.OverlayValues[462] = d462
			ps493.OverlayValues[463] = d463
			ps493.OverlayValues[464] = d464
			ps493.OverlayValues[465] = d465
			ps493.OverlayValues[466] = d466
			ps493.OverlayValues[467] = d467
			ps493.OverlayValues[468] = d468
			ps493.OverlayValues[469] = d469
			ps493.OverlayValues[470] = d470
			ps493.OverlayValues[471] = d471
			ps493.OverlayValues[472] = d472
			ps493.OverlayValues[473] = d473
			ps493.OverlayValues[474] = d474
			ps493.OverlayValues[475] = d475
			ps493.OverlayValues[476] = d476
			ps493.OverlayValues[477] = d477
			ps493.OverlayValues[478] = d478
			ps493.OverlayValues[479] = d479
			ps493.OverlayValues[480] = d480
			ps493.OverlayValues[481] = d481
			ps493.OverlayValues[482] = d482
			ps493.OverlayValues[483] = d483
			ps493.OverlayValues[484] = d484
			ps493.OverlayValues[485] = d485
			ps493.OverlayValues[486] = d486
			ps493.OverlayValues[487] = d487
			ps493.OverlayValues[488] = d488
			ps493.OverlayValues[489] = d489
			ps493.OverlayValues[490] = d490
			ps494 := scm.PhiState{General: true}
			ps494.OverlayValues = make([]scm.JITValueDesc, 491)
			ps494.OverlayValues[0] = d0
			ps494.OverlayValues[1] = d1
			ps494.OverlayValues[2] = d2
			ps494.OverlayValues[3] = d3
			ps494.OverlayValues[4] = d4
			ps494.OverlayValues[5] = d5
			ps494.OverlayValues[6] = d6
			ps494.OverlayValues[7] = d7
			ps494.OverlayValues[8] = d8
			ps494.OverlayValues[9] = d9
			ps494.OverlayValues[10] = d10
			ps494.OverlayValues[11] = d11
			ps494.OverlayValues[12] = d12
			ps494.OverlayValues[13] = d13
			ps494.OverlayValues[14] = d14
			ps494.OverlayValues[15] = d15
			ps494.OverlayValues[16] = d16
			ps494.OverlayValues[18] = d18
			ps494.OverlayValues[19] = d19
			ps494.OverlayValues[20] = d20
			ps494.OverlayValues[21] = d21
			ps494.OverlayValues[22] = d22
			ps494.OverlayValues[23] = d23
			ps494.OverlayValues[24] = d24
			ps494.OverlayValues[25] = d25
			ps494.OverlayValues[26] = d26
			ps494.OverlayValues[27] = d27
			ps494.OverlayValues[28] = d28
			ps494.OverlayValues[29] = d29
			ps494.OverlayValues[30] = d30
			ps494.OverlayValues[31] = d31
			ps494.OverlayValues[32] = d32
			ps494.OverlayValues[33] = d33
			ps494.OverlayValues[34] = d34
			ps494.OverlayValues[35] = d35
			ps494.OverlayValues[36] = d36
			ps494.OverlayValues[37] = d37
			ps494.OverlayValues[38] = d38
			ps494.OverlayValues[39] = d39
			ps494.OverlayValues[40] = d40
			ps494.OverlayValues[41] = d41
			ps494.OverlayValues[42] = d42
			ps494.OverlayValues[43] = d43
			ps494.OverlayValues[44] = d44
			ps494.OverlayValues[45] = d45
			ps494.OverlayValues[46] = d46
			ps494.OverlayValues[47] = d47
			ps494.OverlayValues[48] = d48
			ps494.OverlayValues[49] = d49
			ps494.OverlayValues[50] = d50
			ps494.OverlayValues[51] = d51
			ps494.OverlayValues[52] = d52
			ps494.OverlayValues[53] = d53
			ps494.OverlayValues[54] = d54
			ps494.OverlayValues[55] = d55
			ps494.OverlayValues[56] = d56
			ps494.OverlayValues[57] = d57
			ps494.OverlayValues[58] = d58
			ps494.OverlayValues[59] = d59
			ps494.OverlayValues[60] = d60
			ps494.OverlayValues[61] = d61
			ps494.OverlayValues[64] = d64
			ps494.OverlayValues[65] = d65
			ps494.OverlayValues[66] = d66
			ps494.OverlayValues[134] = d134
			ps494.OverlayValues[135] = d135
			ps494.OverlayValues[136] = d136
			ps494.OverlayValues[137] = d137
			ps494.OverlayValues[138] = d138
			ps494.OverlayValues[139] = d139
			ps494.OverlayValues[140] = d140
			ps494.OverlayValues[141] = d141
			ps494.OverlayValues[142] = d142
			ps494.OverlayValues[143] = d143
			ps494.OverlayValues[144] = d144
			ps494.OverlayValues[145] = d145
			ps494.OverlayValues[146] = d146
			ps494.OverlayValues[147] = d147
			ps494.OverlayValues[148] = d148
			ps494.OverlayValues[149] = d149
			ps494.OverlayValues[150] = d150
			ps494.OverlayValues[151] = d151
			ps494.OverlayValues[152] = d152
			ps494.OverlayValues[153] = d153
			ps494.OverlayValues[154] = d154
			ps494.OverlayValues[155] = d155
			ps494.OverlayValues[156] = d156
			ps494.OverlayValues[157] = d157
			ps494.OverlayValues[158] = d158
			ps494.OverlayValues[159] = d159
			ps494.OverlayValues[160] = d160
			ps494.OverlayValues[161] = d161
			ps494.OverlayValues[162] = d162
			ps494.OverlayValues[163] = d163
			ps494.OverlayValues[164] = d164
			ps494.OverlayValues[165] = d165
			ps494.OverlayValues[166] = d166
			ps494.OverlayValues[167] = d167
			ps494.OverlayValues[168] = d168
			ps494.OverlayValues[169] = d169
			ps494.OverlayValues[170] = d170
			ps494.OverlayValues[171] = d171
			ps494.OverlayValues[172] = d172
			ps494.OverlayValues[175] = d175
			ps494.OverlayValues[283] = d283
			ps494.OverlayValues[284] = d284
			ps494.OverlayValues[285] = d285
			ps494.OverlayValues[286] = d286
			ps494.OverlayValues[287] = d287
			ps494.OverlayValues[288] = d288
			ps494.OverlayValues[289] = d289
			ps494.OverlayValues[290] = d290
			ps494.OverlayValues[292] = d292
			ps494.OverlayValues[293] = d293
			ps494.OverlayValues[294] = d294
			ps494.OverlayValues[295] = d295
			ps494.OverlayValues[296] = d296
			ps494.OverlayValues[297] = d297
			ps494.OverlayValues[298] = d298
			ps494.OverlayValues[299] = d299
			ps494.OverlayValues[300] = d300
			ps494.OverlayValues[301] = d301
			ps494.OverlayValues[303] = d303
			ps494.OverlayValues[305] = d305
			ps494.OverlayValues[306] = d306
			ps494.OverlayValues[307] = d307
			ps494.OverlayValues[308] = d308
			ps494.OverlayValues[309] = d309
			ps494.OverlayValues[312] = d312
			ps494.OverlayValues[443] = d443
			ps494.OverlayValues[444] = d444
			ps494.OverlayValues[445] = d445
			ps494.OverlayValues[446] = d446
			ps494.OverlayValues[447] = d447
			ps494.OverlayValues[448] = d448
			ps494.OverlayValues[449] = d449
			ps494.OverlayValues[451] = d451
			ps494.OverlayValues[452] = d452
			ps494.OverlayValues[453] = d453
			ps494.OverlayValues[454] = d454
			ps494.OverlayValues[455] = d455
			ps494.OverlayValues[456] = d456
			ps494.OverlayValues[457] = d457
			ps494.OverlayValues[458] = d458
			ps494.OverlayValues[459] = d459
			ps494.OverlayValues[460] = d460
			ps494.OverlayValues[461] = d461
			ps494.OverlayValues[462] = d462
			ps494.OverlayValues[463] = d463
			ps494.OverlayValues[464] = d464
			ps494.OverlayValues[465] = d465
			ps494.OverlayValues[466] = d466
			ps494.OverlayValues[467] = d467
			ps494.OverlayValues[468] = d468
			ps494.OverlayValues[469] = d469
			ps494.OverlayValues[470] = d470
			ps494.OverlayValues[471] = d471
			ps494.OverlayValues[472] = d472
			ps494.OverlayValues[473] = d473
			ps494.OverlayValues[474] = d474
			ps494.OverlayValues[475] = d475
			ps494.OverlayValues[476] = d476
			ps494.OverlayValues[477] = d477
			ps494.OverlayValues[478] = d478
			ps494.OverlayValues[479] = d479
			ps494.OverlayValues[480] = d480
			ps494.OverlayValues[481] = d481
			ps494.OverlayValues[482] = d482
			ps494.OverlayValues[483] = d483
			ps494.OverlayValues[484] = d484
			ps494.OverlayValues[485] = d485
			ps494.OverlayValues[486] = d486
			ps494.OverlayValues[487] = d487
			ps494.OverlayValues[488] = d488
			ps494.OverlayValues[489] = d489
			ps494.OverlayValues[490] = d490
			snap495 := d0
			snap496 := d1
			snap497 := d2
			snap498 := d3
			snap499 := d4
			snap500 := d5
			snap501 := d6
			snap502 := d7
			snap503 := d8
			snap504 := d9
			snap505 := d10
			snap506 := d11
			snap507 := d12
			snap508 := d13
			snap509 := d14
			snap510 := d15
			snap511 := d16
			snap512 := d18
			snap513 := d19
			snap514 := d20
			snap515 := d21
			snap516 := d22
			snap517 := d23
			snap518 := d24
			snap519 := d25
			snap520 := d26
			snap521 := d27
			snap522 := d28
			snap523 := d29
			snap524 := d30
			snap525 := d31
			snap526 := d32
			snap527 := d33
			snap528 := d34
			snap529 := d35
			snap530 := d36
			snap531 := d37
			snap532 := d38
			snap533 := d39
			snap534 := d40
			snap535 := d41
			snap536 := d42
			snap537 := d43
			snap538 := d44
			snap539 := d45
			snap540 := d46
			snap541 := d47
			snap542 := d48
			snap543 := d49
			snap544 := d50
			snap545 := d51
			snap546 := d52
			snap547 := d53
			snap548 := d54
			snap549 := d55
			snap550 := d56
			snap551 := d57
			snap552 := d58
			snap553 := d59
			snap554 := d60
			snap555 := d61
			snap556 := d64
			snap557 := d65
			snap558 := d66
			snap559 := d134
			snap560 := d135
			snap561 := d136
			snap562 := d137
			snap563 := d138
			snap564 := d139
			snap565 := d140
			snap566 := d141
			snap567 := d142
			snap568 := d143
			snap569 := d144
			snap570 := d145
			snap571 := d146
			snap572 := d147
			snap573 := d148
			snap574 := d149
			snap575 := d150
			snap576 := d151
			snap577 := d152
			snap578 := d153
			snap579 := d154
			snap580 := d155
			snap581 := d156
			snap582 := d157
			snap583 := d158
			snap584 := d159
			snap585 := d160
			snap586 := d161
			snap587 := d162
			snap588 := d163
			snap589 := d164
			snap590 := d165
			snap591 := d166
			snap592 := d167
			snap593 := d168
			snap594 := d169
			snap595 := d170
			snap596 := d171
			snap597 := d172
			snap598 := d175
			snap599 := d283
			snap600 := d284
			snap601 := d285
			snap602 := d286
			snap603 := d287
			snap604 := d288
			snap605 := d289
			snap606 := d290
			snap607 := d292
			snap608 := d293
			snap609 := d294
			snap610 := d295
			snap611 := d296
			snap612 := d297
			snap613 := d298
			snap614 := d299
			snap615 := d300
			snap616 := d301
			snap617 := d303
			snap618 := d305
			snap619 := d306
			snap620 := d307
			snap621 := d308
			snap622 := d309
			snap623 := d312
			snap624 := d443
			snap625 := d444
			snap626 := d445
			snap627 := d446
			snap628 := d447
			snap629 := d448
			snap630 := d449
			snap631 := d451
			snap632 := d452
			snap633 := d453
			snap634 := d454
			snap635 := d455
			snap636 := d456
			snap637 := d457
			snap638 := d458
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
			alloc671 := ctx.SnapshotAllocState()
			if !bbs[9].Rendered {
				bbs[9].RenderPS(ps494)
			}
			ctx.RestoreAllocState(alloc671)
			d0 = snap495
			d1 = snap496
			d2 = snap497
			d3 = snap498
			d4 = snap499
			d5 = snap500
			d6 = snap501
			d7 = snap502
			d8 = snap503
			d9 = snap504
			d10 = snap505
			d11 = snap506
			d12 = snap507
			d13 = snap508
			d14 = snap509
			d15 = snap510
			d16 = snap511
			d18 = snap512
			d19 = snap513
			d20 = snap514
			d21 = snap515
			d22 = snap516
			d23 = snap517
			d24 = snap518
			d25 = snap519
			d26 = snap520
			d27 = snap521
			d28 = snap522
			d29 = snap523
			d30 = snap524
			d31 = snap525
			d32 = snap526
			d33 = snap527
			d34 = snap528
			d35 = snap529
			d36 = snap530
			d37 = snap531
			d38 = snap532
			d39 = snap533
			d40 = snap534
			d41 = snap535
			d42 = snap536
			d43 = snap537
			d44 = snap538
			d45 = snap539
			d46 = snap540
			d47 = snap541
			d48 = snap542
			d49 = snap543
			d50 = snap544
			d51 = snap545
			d52 = snap546
			d53 = snap547
			d54 = snap548
			d55 = snap549
			d56 = snap550
			d57 = snap551
			d58 = snap552
			d59 = snap553
			d60 = snap554
			d61 = snap555
			d64 = snap556
			d65 = snap557
			d66 = snap558
			d134 = snap559
			d135 = snap560
			d136 = snap561
			d137 = snap562
			d138 = snap563
			d139 = snap564
			d140 = snap565
			d141 = snap566
			d142 = snap567
			d143 = snap568
			d144 = snap569
			d145 = snap570
			d146 = snap571
			d147 = snap572
			d148 = snap573
			d149 = snap574
			d150 = snap575
			d151 = snap576
			d152 = snap577
			d153 = snap578
			d154 = snap579
			d155 = snap580
			d156 = snap581
			d157 = snap582
			d158 = snap583
			d159 = snap584
			d160 = snap585
			d161 = snap586
			d162 = snap587
			d163 = snap588
			d164 = snap589
			d165 = snap590
			d166 = snap591
			d167 = snap592
			d168 = snap593
			d169 = snap594
			d170 = snap595
			d171 = snap596
			d172 = snap597
			d175 = snap598
			d283 = snap599
			d284 = snap600
			d285 = snap601
			d286 = snap602
			d287 = snap603
			d288 = snap604
			d289 = snap605
			d290 = snap606
			d292 = snap607
			d293 = snap608
			d294 = snap609
			d295 = snap610
			d296 = snap611
			d297 = snap612
			d298 = snap613
			d299 = snap614
			d300 = snap615
			d301 = snap616
			d303 = snap617
			d305 = snap618
			d306 = snap619
			d307 = snap620
			d308 = snap621
			d309 = snap622
			d312 = snap623
			d443 = snap624
			d444 = snap625
			d445 = snap626
			d446 = snap627
			d447 = snap628
			d448 = snap629
			d449 = snap630
			d451 = snap631
			d452 = snap632
			d453 = snap633
			d454 = snap634
			d455 = snap635
			d456 = snap636
			d457 = snap637
			d458 = snap638
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
			if !bbs[7].Rendered {
				return bbs[7].RenderPS(ps493)
			}
			return result
			ctx.FreeDesc(&d489)
			return result
			}
			bbs[7].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[7].VisitCount >= 2 {
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
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
				d14 = ps.OverlayValues[14]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
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
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != scm.LocNone {
				d283 = ps.OverlayValues[283]
			}
			if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != scm.LocNone {
				d284 = ps.OverlayValues[284]
			}
			if len(ps.OverlayValues) > 285 && ps.OverlayValues[285].Loc != scm.LocNone {
				d285 = ps.OverlayValues[285]
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
			if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != scm.LocNone {
				d292 = ps.OverlayValues[292]
			}
			if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != scm.LocNone {
				d293 = ps.OverlayValues[293]
			}
			if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
				d294 = ps.OverlayValues[294]
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
			if len(ps.OverlayValues) > 303 && ps.OverlayValues[303].Loc != scm.LocNone {
				d303 = ps.OverlayValues[303]
			}
			if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != scm.LocNone {
				d305 = ps.OverlayValues[305]
			}
			if len(ps.OverlayValues) > 306 && ps.OverlayValues[306].Loc != scm.LocNone {
				d306 = ps.OverlayValues[306]
			}
			if len(ps.OverlayValues) > 307 && ps.OverlayValues[307].Loc != scm.LocNone {
				d307 = ps.OverlayValues[307]
			}
			if len(ps.OverlayValues) > 308 && ps.OverlayValues[308].Loc != scm.LocNone {
				d308 = ps.OverlayValues[308]
			}
			if len(ps.OverlayValues) > 309 && ps.OverlayValues[309].Loc != scm.LocNone {
				d309 = ps.OverlayValues[309]
			}
			if len(ps.OverlayValues) > 312 && ps.OverlayValues[312].Loc != scm.LocNone {
				d312 = ps.OverlayValues[312]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d4)
			var d672 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d672 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d4.Reg)
				ctx.EmitMovRegReg(scratch, d4.Reg)
				ctx.EmitSubRegImm32(scratch, int32(1))
				d672 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d672)
			}
			if d672.Loc == scm.LocImm {
				d672 = scm.JITValueDesc{Loc: scm.LocImm, Type: d672.Type, Imm: scm.NewInt(int64(uint64(d672.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d672.Reg, 32)
				ctx.EmitShrRegImm8(d672.Reg, 32)
			}
			if d672.Loc == scm.LocReg && d4.Loc == scm.LocReg && d672.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d5)
			if d5.Loc == scm.LocReg {
				ctx.ProtectReg(d5.Reg)
			} else if d5.Loc == scm.LocRegPair {
				ctx.ProtectReg(d5.Reg)
				ctx.ProtectReg(d5.Reg2)
			}
			ctx.EnsureDesc(&d672)
			if d672.Loc == scm.LocReg {
				ctx.ProtectReg(d672.Reg)
			} else if d672.Loc == scm.LocRegPair {
				ctx.ProtectReg(d672.Reg)
				ctx.ProtectReg(d672.Reg2)
			}
			d673 = d5
			if d673.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d673)
			d674 = d673
			if d674.Loc == scm.LocImm {
				d674 = scm.JITValueDesc{Loc: scm.LocImm, Type: d674.Type, Imm: scm.NewInt(int64(uint64(d674.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d674.Reg, 32)
				ctx.EmitShrRegImm8(d674.Reg, 32)
			}
			ctx.EmitStoreToStack(d674, int32(bbs[8].PhiBase)+int32(0))
			d675 = d672
			if d675.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d675)
			d676 = d675
			if d676.Loc == scm.LocImm {
				d676 = scm.JITValueDesc{Loc: scm.LocImm, Type: d676.Type, Imm: scm.NewInt(int64(uint64(d676.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d676.Reg, 32)
				ctx.EmitShrRegImm8(d676.Reg, 32)
			}
			ctx.EmitStoreToStack(d676, int32(bbs[8].PhiBase)+int32(16))
			if d5.Loc == scm.LocReg {
				ctx.UnprotectReg(d5.Reg)
			} else if d5.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d5.Reg)
				ctx.UnprotectReg(d5.Reg2)
			}
			if d672.Loc == scm.LocReg {
				ctx.UnprotectReg(d672.Reg)
			} else if d672.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d672.Reg)
				ctx.UnprotectReg(d672.Reg2)
			}
			ps677 := scm.PhiState{General: ps.General}
			ps677.OverlayValues = make([]scm.JITValueDesc, 677)
			ps677.OverlayValues[0] = d0
			ps677.OverlayValues[1] = d1
			ps677.OverlayValues[2] = d2
			ps677.OverlayValues[3] = d3
			ps677.OverlayValues[4] = d4
			ps677.OverlayValues[5] = d5
			ps677.OverlayValues[6] = d6
			ps677.OverlayValues[7] = d7
			ps677.OverlayValues[8] = d8
			ps677.OverlayValues[9] = d9
			ps677.OverlayValues[10] = d10
			ps677.OverlayValues[11] = d11
			ps677.OverlayValues[12] = d12
			ps677.OverlayValues[13] = d13
			ps677.OverlayValues[14] = d14
			ps677.OverlayValues[15] = d15
			ps677.OverlayValues[16] = d16
			ps677.OverlayValues[18] = d18
			ps677.OverlayValues[19] = d19
			ps677.OverlayValues[20] = d20
			ps677.OverlayValues[21] = d21
			ps677.OverlayValues[22] = d22
			ps677.OverlayValues[23] = d23
			ps677.OverlayValues[24] = d24
			ps677.OverlayValues[25] = d25
			ps677.OverlayValues[26] = d26
			ps677.OverlayValues[27] = d27
			ps677.OverlayValues[28] = d28
			ps677.OverlayValues[29] = d29
			ps677.OverlayValues[30] = d30
			ps677.OverlayValues[31] = d31
			ps677.OverlayValues[32] = d32
			ps677.OverlayValues[33] = d33
			ps677.OverlayValues[34] = d34
			ps677.OverlayValues[35] = d35
			ps677.OverlayValues[36] = d36
			ps677.OverlayValues[37] = d37
			ps677.OverlayValues[38] = d38
			ps677.OverlayValues[39] = d39
			ps677.OverlayValues[40] = d40
			ps677.OverlayValues[41] = d41
			ps677.OverlayValues[42] = d42
			ps677.OverlayValues[43] = d43
			ps677.OverlayValues[44] = d44
			ps677.OverlayValues[45] = d45
			ps677.OverlayValues[46] = d46
			ps677.OverlayValues[47] = d47
			ps677.OverlayValues[48] = d48
			ps677.OverlayValues[49] = d49
			ps677.OverlayValues[50] = d50
			ps677.OverlayValues[51] = d51
			ps677.OverlayValues[52] = d52
			ps677.OverlayValues[53] = d53
			ps677.OverlayValues[54] = d54
			ps677.OverlayValues[55] = d55
			ps677.OverlayValues[56] = d56
			ps677.OverlayValues[57] = d57
			ps677.OverlayValues[58] = d58
			ps677.OverlayValues[59] = d59
			ps677.OverlayValues[60] = d60
			ps677.OverlayValues[61] = d61
			ps677.OverlayValues[64] = d64
			ps677.OverlayValues[65] = d65
			ps677.OverlayValues[66] = d66
			ps677.OverlayValues[134] = d134
			ps677.OverlayValues[135] = d135
			ps677.OverlayValues[136] = d136
			ps677.OverlayValues[137] = d137
			ps677.OverlayValues[138] = d138
			ps677.OverlayValues[139] = d139
			ps677.OverlayValues[140] = d140
			ps677.OverlayValues[141] = d141
			ps677.OverlayValues[142] = d142
			ps677.OverlayValues[143] = d143
			ps677.OverlayValues[144] = d144
			ps677.OverlayValues[145] = d145
			ps677.OverlayValues[146] = d146
			ps677.OverlayValues[147] = d147
			ps677.OverlayValues[148] = d148
			ps677.OverlayValues[149] = d149
			ps677.OverlayValues[150] = d150
			ps677.OverlayValues[151] = d151
			ps677.OverlayValues[152] = d152
			ps677.OverlayValues[153] = d153
			ps677.OverlayValues[154] = d154
			ps677.OverlayValues[155] = d155
			ps677.OverlayValues[156] = d156
			ps677.OverlayValues[157] = d157
			ps677.OverlayValues[158] = d158
			ps677.OverlayValues[159] = d159
			ps677.OverlayValues[160] = d160
			ps677.OverlayValues[161] = d161
			ps677.OverlayValues[162] = d162
			ps677.OverlayValues[163] = d163
			ps677.OverlayValues[164] = d164
			ps677.OverlayValues[165] = d165
			ps677.OverlayValues[166] = d166
			ps677.OverlayValues[167] = d167
			ps677.OverlayValues[168] = d168
			ps677.OverlayValues[169] = d169
			ps677.OverlayValues[170] = d170
			ps677.OverlayValues[171] = d171
			ps677.OverlayValues[172] = d172
			ps677.OverlayValues[175] = d175
			ps677.OverlayValues[283] = d283
			ps677.OverlayValues[284] = d284
			ps677.OverlayValues[285] = d285
			ps677.OverlayValues[286] = d286
			ps677.OverlayValues[287] = d287
			ps677.OverlayValues[288] = d288
			ps677.OverlayValues[289] = d289
			ps677.OverlayValues[290] = d290
			ps677.OverlayValues[292] = d292
			ps677.OverlayValues[293] = d293
			ps677.OverlayValues[294] = d294
			ps677.OverlayValues[295] = d295
			ps677.OverlayValues[296] = d296
			ps677.OverlayValues[297] = d297
			ps677.OverlayValues[298] = d298
			ps677.OverlayValues[299] = d299
			ps677.OverlayValues[300] = d300
			ps677.OverlayValues[301] = d301
			ps677.OverlayValues[303] = d303
			ps677.OverlayValues[305] = d305
			ps677.OverlayValues[306] = d306
			ps677.OverlayValues[307] = d307
			ps677.OverlayValues[308] = d308
			ps677.OverlayValues[309] = d309
			ps677.OverlayValues[312] = d312
			ps677.OverlayValues[443] = d443
			ps677.OverlayValues[444] = d444
			ps677.OverlayValues[445] = d445
			ps677.OverlayValues[446] = d446
			ps677.OverlayValues[447] = d447
			ps677.OverlayValues[448] = d448
			ps677.OverlayValues[449] = d449
			ps677.OverlayValues[451] = d451
			ps677.OverlayValues[452] = d452
			ps677.OverlayValues[453] = d453
			ps677.OverlayValues[454] = d454
			ps677.OverlayValues[455] = d455
			ps677.OverlayValues[456] = d456
			ps677.OverlayValues[457] = d457
			ps677.OverlayValues[458] = d458
			ps677.OverlayValues[459] = d459
			ps677.OverlayValues[460] = d460
			ps677.OverlayValues[461] = d461
			ps677.OverlayValues[462] = d462
			ps677.OverlayValues[463] = d463
			ps677.OverlayValues[464] = d464
			ps677.OverlayValues[465] = d465
			ps677.OverlayValues[466] = d466
			ps677.OverlayValues[467] = d467
			ps677.OverlayValues[468] = d468
			ps677.OverlayValues[469] = d469
			ps677.OverlayValues[470] = d470
			ps677.OverlayValues[471] = d471
			ps677.OverlayValues[472] = d472
			ps677.OverlayValues[473] = d473
			ps677.OverlayValues[474] = d474
			ps677.OverlayValues[475] = d475
			ps677.OverlayValues[476] = d476
			ps677.OverlayValues[477] = d477
			ps677.OverlayValues[478] = d478
			ps677.OverlayValues[479] = d479
			ps677.OverlayValues[480] = d480
			ps677.OverlayValues[481] = d481
			ps677.OverlayValues[482] = d482
			ps677.OverlayValues[483] = d483
			ps677.OverlayValues[484] = d484
			ps677.OverlayValues[485] = d485
			ps677.OverlayValues[486] = d486
			ps677.OverlayValues[487] = d487
			ps677.OverlayValues[488] = d488
			ps677.OverlayValues[489] = d489
			ps677.OverlayValues[490] = d490
			ps677.OverlayValues[672] = d672
			ps677.OverlayValues[673] = d673
			ps677.OverlayValues[674] = d674
			ps677.OverlayValues[675] = d675
			ps677.OverlayValues[676] = d676
			ps677.PhiValues = make([]scm.JITValueDesc, 2)
			d678 = d5
			ps677.PhiValues[0] = d678
			d679 = d672
			ps677.PhiValues[1] = d679
			if ps677.General && bbs[8].Rendered {
				ctx.EmitJmp(lbl9)
				return result
			}
			return bbs[8].RenderPS(ps677)
			return result
			}
			bbs[8].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
					d680 := ps.PhiValues[0]
					ctx.EnsureDesc(&d680)
					ctx.EmitStoreToStack(d680, int32(bbs[8].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
					d681 := ps.PhiValues[1]
					ctx.EnsureDesc(&d681)
					ctx.EmitStoreToStack(d681, int32(bbs[8].PhiBase)+int32(16))
				}
				if bbs[8].VisitCount >= 2 {
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
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
				d14 = ps.OverlayValues[14]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
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
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != scm.LocNone {
				d283 = ps.OverlayValues[283]
			}
			if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != scm.LocNone {
				d284 = ps.OverlayValues[284]
			}
			if len(ps.OverlayValues) > 285 && ps.OverlayValues[285].Loc != scm.LocNone {
				d285 = ps.OverlayValues[285]
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
			if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != scm.LocNone {
				d292 = ps.OverlayValues[292]
			}
			if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != scm.LocNone {
				d293 = ps.OverlayValues[293]
			}
			if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
				d294 = ps.OverlayValues[294]
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
			if len(ps.OverlayValues) > 303 && ps.OverlayValues[303].Loc != scm.LocNone {
				d303 = ps.OverlayValues[303]
			}
			if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != scm.LocNone {
				d305 = ps.OverlayValues[305]
			}
			if len(ps.OverlayValues) > 306 && ps.OverlayValues[306].Loc != scm.LocNone {
				d306 = ps.OverlayValues[306]
			}
			if len(ps.OverlayValues) > 307 && ps.OverlayValues[307].Loc != scm.LocNone {
				d307 = ps.OverlayValues[307]
			}
			if len(ps.OverlayValues) > 308 && ps.OverlayValues[308].Loc != scm.LocNone {
				d308 = ps.OverlayValues[308]
			}
			if len(ps.OverlayValues) > 309 && ps.OverlayValues[309].Loc != scm.LocNone {
				d309 = ps.OverlayValues[309]
			}
			if len(ps.OverlayValues) > 312 && ps.OverlayValues[312].Loc != scm.LocNone {
				d312 = ps.OverlayValues[312]
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
			if len(ps.OverlayValues) > 672 && ps.OverlayValues[672].Loc != scm.LocNone {
				d672 = ps.OverlayValues[672]
			}
			if len(ps.OverlayValues) > 673 && ps.OverlayValues[673].Loc != scm.LocNone {
				d673 = ps.OverlayValues[673]
			}
			if len(ps.OverlayValues) > 674 && ps.OverlayValues[674].Loc != scm.LocNone {
				d674 = ps.OverlayValues[674]
			}
			if len(ps.OverlayValues) > 675 && ps.OverlayValues[675].Loc != scm.LocNone {
				d675 = ps.OverlayValues[675]
			}
			if len(ps.OverlayValues) > 676 && ps.OverlayValues[676].Loc != scm.LocNone {
				d676 = ps.OverlayValues[676]
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
			if len(ps.OverlayValues) > 681 && ps.OverlayValues[681].Loc != scm.LocNone {
				d681 = ps.OverlayValues[681]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d7 = ps.PhiValues[0]
			}
			if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d8 = ps.PhiValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d8)
			var d682 scm.JITValueDesc
			if d7.Loc == scm.LocImm && d8.Loc == scm.LocImm {
				d682 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d7.Imm.Int()) == uint64(d8.Imm.Int()))}
			} else if d8.Loc == scm.LocImm {
				r151 := ctx.AllocRegExcept(d7.Reg)
				if d8.Imm.Int() >= -2147483648 && d8.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d7.Reg, int32(d8.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d8.Imm.Int()))
					ctx.EmitCmpInt64(d7.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r151, scm.CcE)
				d682 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r151}
				ctx.BindReg(r151, &d682)
			} else if d7.Loc == scm.LocImm {
				r152 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d7.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d8.Reg)
				ctx.EmitSetcc(r152, scm.CcE)
				d682 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r152}
				ctx.BindReg(r152, &d682)
			} else {
				r153 := ctx.AllocRegExcept(d7.Reg)
				ctx.EmitCmpInt64(d7.Reg, d8.Reg)
				ctx.EmitSetcc(r153, scm.CcE)
				d682 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r153}
				ctx.BindReg(r153, &d682)
			}
			d683 = d682
			ctx.EnsureDesc(&d683)
			if d683.Loc != scm.LocImm && d683.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d683.Loc == scm.LocImm {
				if d683.Imm.Bool() {
			ctx.EnsureDesc(&d7)
			if d7.Loc == scm.LocReg {
				ctx.ProtectReg(d7.Reg)
			} else if d7.Loc == scm.LocRegPair {
				ctx.ProtectReg(d7.Reg)
				ctx.ProtectReg(d7.Reg2)
			}
			d684 = d7
			if d684.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d684)
			d685 = d684
			if d685.Loc == scm.LocImm {
				d685 = scm.JITValueDesc{Loc: scm.LocImm, Type: d685.Type, Imm: scm.NewInt(int64(uint64(d685.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d685.Reg, 32)
				ctx.EmitShrRegImm8(d685.Reg, 32)
			}
			ctx.EmitStoreToStack(d685, int32(bbs[2].PhiBase)+int32(0))
			if d7.Loc == scm.LocReg {
				ctx.UnprotectReg(d7.Reg)
			} else if d7.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d7.Reg)
				ctx.UnprotectReg(d7.Reg2)
			}
			ps686 := scm.PhiState{General: ps.General}
			ps686.OverlayValues = make([]scm.JITValueDesc, 686)
			ps686.OverlayValues[0] = d0
			ps686.OverlayValues[1] = d1
			ps686.OverlayValues[2] = d2
			ps686.OverlayValues[3] = d3
			ps686.OverlayValues[4] = d4
			ps686.OverlayValues[5] = d5
			ps686.OverlayValues[6] = d6
			ps686.OverlayValues[7] = d7
			ps686.OverlayValues[8] = d8
			ps686.OverlayValues[9] = d9
			ps686.OverlayValues[10] = d10
			ps686.OverlayValues[11] = d11
			ps686.OverlayValues[12] = d12
			ps686.OverlayValues[13] = d13
			ps686.OverlayValues[14] = d14
			ps686.OverlayValues[15] = d15
			ps686.OverlayValues[16] = d16
			ps686.OverlayValues[18] = d18
			ps686.OverlayValues[19] = d19
			ps686.OverlayValues[20] = d20
			ps686.OverlayValues[21] = d21
			ps686.OverlayValues[22] = d22
			ps686.OverlayValues[23] = d23
			ps686.OverlayValues[24] = d24
			ps686.OverlayValues[25] = d25
			ps686.OverlayValues[26] = d26
			ps686.OverlayValues[27] = d27
			ps686.OverlayValues[28] = d28
			ps686.OverlayValues[29] = d29
			ps686.OverlayValues[30] = d30
			ps686.OverlayValues[31] = d31
			ps686.OverlayValues[32] = d32
			ps686.OverlayValues[33] = d33
			ps686.OverlayValues[34] = d34
			ps686.OverlayValues[35] = d35
			ps686.OverlayValues[36] = d36
			ps686.OverlayValues[37] = d37
			ps686.OverlayValues[38] = d38
			ps686.OverlayValues[39] = d39
			ps686.OverlayValues[40] = d40
			ps686.OverlayValues[41] = d41
			ps686.OverlayValues[42] = d42
			ps686.OverlayValues[43] = d43
			ps686.OverlayValues[44] = d44
			ps686.OverlayValues[45] = d45
			ps686.OverlayValues[46] = d46
			ps686.OverlayValues[47] = d47
			ps686.OverlayValues[48] = d48
			ps686.OverlayValues[49] = d49
			ps686.OverlayValues[50] = d50
			ps686.OverlayValues[51] = d51
			ps686.OverlayValues[52] = d52
			ps686.OverlayValues[53] = d53
			ps686.OverlayValues[54] = d54
			ps686.OverlayValues[55] = d55
			ps686.OverlayValues[56] = d56
			ps686.OverlayValues[57] = d57
			ps686.OverlayValues[58] = d58
			ps686.OverlayValues[59] = d59
			ps686.OverlayValues[60] = d60
			ps686.OverlayValues[61] = d61
			ps686.OverlayValues[64] = d64
			ps686.OverlayValues[65] = d65
			ps686.OverlayValues[66] = d66
			ps686.OverlayValues[134] = d134
			ps686.OverlayValues[135] = d135
			ps686.OverlayValues[136] = d136
			ps686.OverlayValues[137] = d137
			ps686.OverlayValues[138] = d138
			ps686.OverlayValues[139] = d139
			ps686.OverlayValues[140] = d140
			ps686.OverlayValues[141] = d141
			ps686.OverlayValues[142] = d142
			ps686.OverlayValues[143] = d143
			ps686.OverlayValues[144] = d144
			ps686.OverlayValues[145] = d145
			ps686.OverlayValues[146] = d146
			ps686.OverlayValues[147] = d147
			ps686.OverlayValues[148] = d148
			ps686.OverlayValues[149] = d149
			ps686.OverlayValues[150] = d150
			ps686.OverlayValues[151] = d151
			ps686.OverlayValues[152] = d152
			ps686.OverlayValues[153] = d153
			ps686.OverlayValues[154] = d154
			ps686.OverlayValues[155] = d155
			ps686.OverlayValues[156] = d156
			ps686.OverlayValues[157] = d157
			ps686.OverlayValues[158] = d158
			ps686.OverlayValues[159] = d159
			ps686.OverlayValues[160] = d160
			ps686.OverlayValues[161] = d161
			ps686.OverlayValues[162] = d162
			ps686.OverlayValues[163] = d163
			ps686.OverlayValues[164] = d164
			ps686.OverlayValues[165] = d165
			ps686.OverlayValues[166] = d166
			ps686.OverlayValues[167] = d167
			ps686.OverlayValues[168] = d168
			ps686.OverlayValues[169] = d169
			ps686.OverlayValues[170] = d170
			ps686.OverlayValues[171] = d171
			ps686.OverlayValues[172] = d172
			ps686.OverlayValues[175] = d175
			ps686.OverlayValues[283] = d283
			ps686.OverlayValues[284] = d284
			ps686.OverlayValues[285] = d285
			ps686.OverlayValues[286] = d286
			ps686.OverlayValues[287] = d287
			ps686.OverlayValues[288] = d288
			ps686.OverlayValues[289] = d289
			ps686.OverlayValues[290] = d290
			ps686.OverlayValues[292] = d292
			ps686.OverlayValues[293] = d293
			ps686.OverlayValues[294] = d294
			ps686.OverlayValues[295] = d295
			ps686.OverlayValues[296] = d296
			ps686.OverlayValues[297] = d297
			ps686.OverlayValues[298] = d298
			ps686.OverlayValues[299] = d299
			ps686.OverlayValues[300] = d300
			ps686.OverlayValues[301] = d301
			ps686.OverlayValues[303] = d303
			ps686.OverlayValues[305] = d305
			ps686.OverlayValues[306] = d306
			ps686.OverlayValues[307] = d307
			ps686.OverlayValues[308] = d308
			ps686.OverlayValues[309] = d309
			ps686.OverlayValues[312] = d312
			ps686.OverlayValues[443] = d443
			ps686.OverlayValues[444] = d444
			ps686.OverlayValues[445] = d445
			ps686.OverlayValues[446] = d446
			ps686.OverlayValues[447] = d447
			ps686.OverlayValues[448] = d448
			ps686.OverlayValues[449] = d449
			ps686.OverlayValues[451] = d451
			ps686.OverlayValues[452] = d452
			ps686.OverlayValues[453] = d453
			ps686.OverlayValues[454] = d454
			ps686.OverlayValues[455] = d455
			ps686.OverlayValues[456] = d456
			ps686.OverlayValues[457] = d457
			ps686.OverlayValues[458] = d458
			ps686.OverlayValues[459] = d459
			ps686.OverlayValues[460] = d460
			ps686.OverlayValues[461] = d461
			ps686.OverlayValues[462] = d462
			ps686.OverlayValues[463] = d463
			ps686.OverlayValues[464] = d464
			ps686.OverlayValues[465] = d465
			ps686.OverlayValues[466] = d466
			ps686.OverlayValues[467] = d467
			ps686.OverlayValues[468] = d468
			ps686.OverlayValues[469] = d469
			ps686.OverlayValues[470] = d470
			ps686.OverlayValues[471] = d471
			ps686.OverlayValues[472] = d472
			ps686.OverlayValues[473] = d473
			ps686.OverlayValues[474] = d474
			ps686.OverlayValues[475] = d475
			ps686.OverlayValues[476] = d476
			ps686.OverlayValues[477] = d477
			ps686.OverlayValues[478] = d478
			ps686.OverlayValues[479] = d479
			ps686.OverlayValues[480] = d480
			ps686.OverlayValues[481] = d481
			ps686.OverlayValues[482] = d482
			ps686.OverlayValues[483] = d483
			ps686.OverlayValues[484] = d484
			ps686.OverlayValues[485] = d485
			ps686.OverlayValues[486] = d486
			ps686.OverlayValues[487] = d487
			ps686.OverlayValues[488] = d488
			ps686.OverlayValues[489] = d489
			ps686.OverlayValues[490] = d490
			ps686.OverlayValues[672] = d672
			ps686.OverlayValues[673] = d673
			ps686.OverlayValues[674] = d674
			ps686.OverlayValues[675] = d675
			ps686.OverlayValues[676] = d676
			ps686.OverlayValues[678] = d678
			ps686.OverlayValues[679] = d679
			ps686.OverlayValues[680] = d680
			ps686.OverlayValues[681] = d681
			ps686.OverlayValues[682] = d682
			ps686.OverlayValues[683] = d683
			ps686.OverlayValues[684] = d684
			ps686.OverlayValues[685] = d685
			ps686.PhiValues = make([]scm.JITValueDesc, 1)
			d687 = d7
			ps686.PhiValues[0] = d687
					return bbs[2].RenderPS(ps686)
				}
			ps688 := scm.PhiState{General: ps.General}
			ps688.OverlayValues = make([]scm.JITValueDesc, 688)
			ps688.OverlayValues[0] = d0
			ps688.OverlayValues[1] = d1
			ps688.OverlayValues[2] = d2
			ps688.OverlayValues[3] = d3
			ps688.OverlayValues[4] = d4
			ps688.OverlayValues[5] = d5
			ps688.OverlayValues[6] = d6
			ps688.OverlayValues[7] = d7
			ps688.OverlayValues[8] = d8
			ps688.OverlayValues[9] = d9
			ps688.OverlayValues[10] = d10
			ps688.OverlayValues[11] = d11
			ps688.OverlayValues[12] = d12
			ps688.OverlayValues[13] = d13
			ps688.OverlayValues[14] = d14
			ps688.OverlayValues[15] = d15
			ps688.OverlayValues[16] = d16
			ps688.OverlayValues[18] = d18
			ps688.OverlayValues[19] = d19
			ps688.OverlayValues[20] = d20
			ps688.OverlayValues[21] = d21
			ps688.OverlayValues[22] = d22
			ps688.OverlayValues[23] = d23
			ps688.OverlayValues[24] = d24
			ps688.OverlayValues[25] = d25
			ps688.OverlayValues[26] = d26
			ps688.OverlayValues[27] = d27
			ps688.OverlayValues[28] = d28
			ps688.OverlayValues[29] = d29
			ps688.OverlayValues[30] = d30
			ps688.OverlayValues[31] = d31
			ps688.OverlayValues[32] = d32
			ps688.OverlayValues[33] = d33
			ps688.OverlayValues[34] = d34
			ps688.OverlayValues[35] = d35
			ps688.OverlayValues[36] = d36
			ps688.OverlayValues[37] = d37
			ps688.OverlayValues[38] = d38
			ps688.OverlayValues[39] = d39
			ps688.OverlayValues[40] = d40
			ps688.OverlayValues[41] = d41
			ps688.OverlayValues[42] = d42
			ps688.OverlayValues[43] = d43
			ps688.OverlayValues[44] = d44
			ps688.OverlayValues[45] = d45
			ps688.OverlayValues[46] = d46
			ps688.OverlayValues[47] = d47
			ps688.OverlayValues[48] = d48
			ps688.OverlayValues[49] = d49
			ps688.OverlayValues[50] = d50
			ps688.OverlayValues[51] = d51
			ps688.OverlayValues[52] = d52
			ps688.OverlayValues[53] = d53
			ps688.OverlayValues[54] = d54
			ps688.OverlayValues[55] = d55
			ps688.OverlayValues[56] = d56
			ps688.OverlayValues[57] = d57
			ps688.OverlayValues[58] = d58
			ps688.OverlayValues[59] = d59
			ps688.OverlayValues[60] = d60
			ps688.OverlayValues[61] = d61
			ps688.OverlayValues[64] = d64
			ps688.OverlayValues[65] = d65
			ps688.OverlayValues[66] = d66
			ps688.OverlayValues[134] = d134
			ps688.OverlayValues[135] = d135
			ps688.OverlayValues[136] = d136
			ps688.OverlayValues[137] = d137
			ps688.OverlayValues[138] = d138
			ps688.OverlayValues[139] = d139
			ps688.OverlayValues[140] = d140
			ps688.OverlayValues[141] = d141
			ps688.OverlayValues[142] = d142
			ps688.OverlayValues[143] = d143
			ps688.OverlayValues[144] = d144
			ps688.OverlayValues[145] = d145
			ps688.OverlayValues[146] = d146
			ps688.OverlayValues[147] = d147
			ps688.OverlayValues[148] = d148
			ps688.OverlayValues[149] = d149
			ps688.OverlayValues[150] = d150
			ps688.OverlayValues[151] = d151
			ps688.OverlayValues[152] = d152
			ps688.OverlayValues[153] = d153
			ps688.OverlayValues[154] = d154
			ps688.OverlayValues[155] = d155
			ps688.OverlayValues[156] = d156
			ps688.OverlayValues[157] = d157
			ps688.OverlayValues[158] = d158
			ps688.OverlayValues[159] = d159
			ps688.OverlayValues[160] = d160
			ps688.OverlayValues[161] = d161
			ps688.OverlayValues[162] = d162
			ps688.OverlayValues[163] = d163
			ps688.OverlayValues[164] = d164
			ps688.OverlayValues[165] = d165
			ps688.OverlayValues[166] = d166
			ps688.OverlayValues[167] = d167
			ps688.OverlayValues[168] = d168
			ps688.OverlayValues[169] = d169
			ps688.OverlayValues[170] = d170
			ps688.OverlayValues[171] = d171
			ps688.OverlayValues[172] = d172
			ps688.OverlayValues[175] = d175
			ps688.OverlayValues[283] = d283
			ps688.OverlayValues[284] = d284
			ps688.OverlayValues[285] = d285
			ps688.OverlayValues[286] = d286
			ps688.OverlayValues[287] = d287
			ps688.OverlayValues[288] = d288
			ps688.OverlayValues[289] = d289
			ps688.OverlayValues[290] = d290
			ps688.OverlayValues[292] = d292
			ps688.OverlayValues[293] = d293
			ps688.OverlayValues[294] = d294
			ps688.OverlayValues[295] = d295
			ps688.OverlayValues[296] = d296
			ps688.OverlayValues[297] = d297
			ps688.OverlayValues[298] = d298
			ps688.OverlayValues[299] = d299
			ps688.OverlayValues[300] = d300
			ps688.OverlayValues[301] = d301
			ps688.OverlayValues[303] = d303
			ps688.OverlayValues[305] = d305
			ps688.OverlayValues[306] = d306
			ps688.OverlayValues[307] = d307
			ps688.OverlayValues[308] = d308
			ps688.OverlayValues[309] = d309
			ps688.OverlayValues[312] = d312
			ps688.OverlayValues[443] = d443
			ps688.OverlayValues[444] = d444
			ps688.OverlayValues[445] = d445
			ps688.OverlayValues[446] = d446
			ps688.OverlayValues[447] = d447
			ps688.OverlayValues[448] = d448
			ps688.OverlayValues[449] = d449
			ps688.OverlayValues[451] = d451
			ps688.OverlayValues[452] = d452
			ps688.OverlayValues[453] = d453
			ps688.OverlayValues[454] = d454
			ps688.OverlayValues[455] = d455
			ps688.OverlayValues[456] = d456
			ps688.OverlayValues[457] = d457
			ps688.OverlayValues[458] = d458
			ps688.OverlayValues[459] = d459
			ps688.OverlayValues[460] = d460
			ps688.OverlayValues[461] = d461
			ps688.OverlayValues[462] = d462
			ps688.OverlayValues[463] = d463
			ps688.OverlayValues[464] = d464
			ps688.OverlayValues[465] = d465
			ps688.OverlayValues[466] = d466
			ps688.OverlayValues[467] = d467
			ps688.OverlayValues[468] = d468
			ps688.OverlayValues[469] = d469
			ps688.OverlayValues[470] = d470
			ps688.OverlayValues[471] = d471
			ps688.OverlayValues[472] = d472
			ps688.OverlayValues[473] = d473
			ps688.OverlayValues[474] = d474
			ps688.OverlayValues[475] = d475
			ps688.OverlayValues[476] = d476
			ps688.OverlayValues[477] = d477
			ps688.OverlayValues[478] = d478
			ps688.OverlayValues[479] = d479
			ps688.OverlayValues[480] = d480
			ps688.OverlayValues[481] = d481
			ps688.OverlayValues[482] = d482
			ps688.OverlayValues[483] = d483
			ps688.OverlayValues[484] = d484
			ps688.OverlayValues[485] = d485
			ps688.OverlayValues[486] = d486
			ps688.OverlayValues[487] = d487
			ps688.OverlayValues[488] = d488
			ps688.OverlayValues[489] = d489
			ps688.OverlayValues[490] = d490
			ps688.OverlayValues[672] = d672
			ps688.OverlayValues[673] = d673
			ps688.OverlayValues[674] = d674
			ps688.OverlayValues[675] = d675
			ps688.OverlayValues[676] = d676
			ps688.OverlayValues[678] = d678
			ps688.OverlayValues[679] = d679
			ps688.OverlayValues[680] = d680
			ps688.OverlayValues[681] = d681
			ps688.OverlayValues[682] = d682
			ps688.OverlayValues[683] = d683
			ps688.OverlayValues[684] = d684
			ps688.OverlayValues[685] = d685
			ps688.OverlayValues[687] = d687
				return bbs[10].RenderPS(ps688)
			}
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
					d689 := ps.PhiValues[0]
					ctx.EnsureDesc(&d689)
					ctx.EmitStoreToStack(d689, int32(bbs[8].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
					d690 := ps.PhiValues[1]
					ctx.EnsureDesc(&d690)
					ctx.EmitStoreToStack(d690, int32(bbs[8].PhiBase)+int32(16))
				}
				ps.General = true
				return bbs[8].RenderPS(ps)
			}
			lbl38 := ctx.ReserveLabel()
			lbl39 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d683.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl38)
			ctx.EmitJmp(lbl39)
			ctx.MarkLabel(lbl38)
			ctx.EnsureDesc(&d7)
			if d7.Loc == scm.LocReg {
				ctx.ProtectReg(d7.Reg)
			} else if d7.Loc == scm.LocRegPair {
				ctx.ProtectReg(d7.Reg)
				ctx.ProtectReg(d7.Reg2)
			}
			d691 = d7
			if d691.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d691)
			d692 = d691
			if d692.Loc == scm.LocImm {
				d692 = scm.JITValueDesc{Loc: scm.LocImm, Type: d692.Type, Imm: scm.NewInt(int64(uint64(d692.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d692.Reg, 32)
				ctx.EmitShrRegImm8(d692.Reg, 32)
			}
			ctx.EmitStoreToStack(d692, int32(bbs[2].PhiBase)+int32(0))
			if d7.Loc == scm.LocReg {
				ctx.UnprotectReg(d7.Reg)
			} else if d7.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d7.Reg)
				ctx.UnprotectReg(d7.Reg2)
			}
			ctx.EmitJmp(lbl3)
			ctx.MarkLabel(lbl39)
			ctx.EmitJmp(lbl11)
			ps693 := scm.PhiState{General: true}
			ps693.OverlayValues = make([]scm.JITValueDesc, 693)
			ps693.OverlayValues[0] = d0
			ps693.OverlayValues[1] = d1
			ps693.OverlayValues[2] = d2
			ps693.OverlayValues[3] = d3
			ps693.OverlayValues[4] = d4
			ps693.OverlayValues[5] = d5
			ps693.OverlayValues[6] = d6
			ps693.OverlayValues[7] = d7
			ps693.OverlayValues[8] = d8
			ps693.OverlayValues[9] = d9
			ps693.OverlayValues[10] = d10
			ps693.OverlayValues[11] = d11
			ps693.OverlayValues[12] = d12
			ps693.OverlayValues[13] = d13
			ps693.OverlayValues[14] = d14
			ps693.OverlayValues[15] = d15
			ps693.OverlayValues[16] = d16
			ps693.OverlayValues[18] = d18
			ps693.OverlayValues[19] = d19
			ps693.OverlayValues[20] = d20
			ps693.OverlayValues[21] = d21
			ps693.OverlayValues[22] = d22
			ps693.OverlayValues[23] = d23
			ps693.OverlayValues[24] = d24
			ps693.OverlayValues[25] = d25
			ps693.OverlayValues[26] = d26
			ps693.OverlayValues[27] = d27
			ps693.OverlayValues[28] = d28
			ps693.OverlayValues[29] = d29
			ps693.OverlayValues[30] = d30
			ps693.OverlayValues[31] = d31
			ps693.OverlayValues[32] = d32
			ps693.OverlayValues[33] = d33
			ps693.OverlayValues[34] = d34
			ps693.OverlayValues[35] = d35
			ps693.OverlayValues[36] = d36
			ps693.OverlayValues[37] = d37
			ps693.OverlayValues[38] = d38
			ps693.OverlayValues[39] = d39
			ps693.OverlayValues[40] = d40
			ps693.OverlayValues[41] = d41
			ps693.OverlayValues[42] = d42
			ps693.OverlayValues[43] = d43
			ps693.OverlayValues[44] = d44
			ps693.OverlayValues[45] = d45
			ps693.OverlayValues[46] = d46
			ps693.OverlayValues[47] = d47
			ps693.OverlayValues[48] = d48
			ps693.OverlayValues[49] = d49
			ps693.OverlayValues[50] = d50
			ps693.OverlayValues[51] = d51
			ps693.OverlayValues[52] = d52
			ps693.OverlayValues[53] = d53
			ps693.OverlayValues[54] = d54
			ps693.OverlayValues[55] = d55
			ps693.OverlayValues[56] = d56
			ps693.OverlayValues[57] = d57
			ps693.OverlayValues[58] = d58
			ps693.OverlayValues[59] = d59
			ps693.OverlayValues[60] = d60
			ps693.OverlayValues[61] = d61
			ps693.OverlayValues[64] = d64
			ps693.OverlayValues[65] = d65
			ps693.OverlayValues[66] = d66
			ps693.OverlayValues[134] = d134
			ps693.OverlayValues[135] = d135
			ps693.OverlayValues[136] = d136
			ps693.OverlayValues[137] = d137
			ps693.OverlayValues[138] = d138
			ps693.OverlayValues[139] = d139
			ps693.OverlayValues[140] = d140
			ps693.OverlayValues[141] = d141
			ps693.OverlayValues[142] = d142
			ps693.OverlayValues[143] = d143
			ps693.OverlayValues[144] = d144
			ps693.OverlayValues[145] = d145
			ps693.OverlayValues[146] = d146
			ps693.OverlayValues[147] = d147
			ps693.OverlayValues[148] = d148
			ps693.OverlayValues[149] = d149
			ps693.OverlayValues[150] = d150
			ps693.OverlayValues[151] = d151
			ps693.OverlayValues[152] = d152
			ps693.OverlayValues[153] = d153
			ps693.OverlayValues[154] = d154
			ps693.OverlayValues[155] = d155
			ps693.OverlayValues[156] = d156
			ps693.OverlayValues[157] = d157
			ps693.OverlayValues[158] = d158
			ps693.OverlayValues[159] = d159
			ps693.OverlayValues[160] = d160
			ps693.OverlayValues[161] = d161
			ps693.OverlayValues[162] = d162
			ps693.OverlayValues[163] = d163
			ps693.OverlayValues[164] = d164
			ps693.OverlayValues[165] = d165
			ps693.OverlayValues[166] = d166
			ps693.OverlayValues[167] = d167
			ps693.OverlayValues[168] = d168
			ps693.OverlayValues[169] = d169
			ps693.OverlayValues[170] = d170
			ps693.OverlayValues[171] = d171
			ps693.OverlayValues[172] = d172
			ps693.OverlayValues[175] = d175
			ps693.OverlayValues[283] = d283
			ps693.OverlayValues[284] = d284
			ps693.OverlayValues[285] = d285
			ps693.OverlayValues[286] = d286
			ps693.OverlayValues[287] = d287
			ps693.OverlayValues[288] = d288
			ps693.OverlayValues[289] = d289
			ps693.OverlayValues[290] = d290
			ps693.OverlayValues[292] = d292
			ps693.OverlayValues[293] = d293
			ps693.OverlayValues[294] = d294
			ps693.OverlayValues[295] = d295
			ps693.OverlayValues[296] = d296
			ps693.OverlayValues[297] = d297
			ps693.OverlayValues[298] = d298
			ps693.OverlayValues[299] = d299
			ps693.OverlayValues[300] = d300
			ps693.OverlayValues[301] = d301
			ps693.OverlayValues[303] = d303
			ps693.OverlayValues[305] = d305
			ps693.OverlayValues[306] = d306
			ps693.OverlayValues[307] = d307
			ps693.OverlayValues[308] = d308
			ps693.OverlayValues[309] = d309
			ps693.OverlayValues[312] = d312
			ps693.OverlayValues[443] = d443
			ps693.OverlayValues[444] = d444
			ps693.OverlayValues[445] = d445
			ps693.OverlayValues[446] = d446
			ps693.OverlayValues[447] = d447
			ps693.OverlayValues[448] = d448
			ps693.OverlayValues[449] = d449
			ps693.OverlayValues[451] = d451
			ps693.OverlayValues[452] = d452
			ps693.OverlayValues[453] = d453
			ps693.OverlayValues[454] = d454
			ps693.OverlayValues[455] = d455
			ps693.OverlayValues[456] = d456
			ps693.OverlayValues[457] = d457
			ps693.OverlayValues[458] = d458
			ps693.OverlayValues[459] = d459
			ps693.OverlayValues[460] = d460
			ps693.OverlayValues[461] = d461
			ps693.OverlayValues[462] = d462
			ps693.OverlayValues[463] = d463
			ps693.OverlayValues[464] = d464
			ps693.OverlayValues[465] = d465
			ps693.OverlayValues[466] = d466
			ps693.OverlayValues[467] = d467
			ps693.OverlayValues[468] = d468
			ps693.OverlayValues[469] = d469
			ps693.OverlayValues[470] = d470
			ps693.OverlayValues[471] = d471
			ps693.OverlayValues[472] = d472
			ps693.OverlayValues[473] = d473
			ps693.OverlayValues[474] = d474
			ps693.OverlayValues[475] = d475
			ps693.OverlayValues[476] = d476
			ps693.OverlayValues[477] = d477
			ps693.OverlayValues[478] = d478
			ps693.OverlayValues[479] = d479
			ps693.OverlayValues[480] = d480
			ps693.OverlayValues[481] = d481
			ps693.OverlayValues[482] = d482
			ps693.OverlayValues[483] = d483
			ps693.OverlayValues[484] = d484
			ps693.OverlayValues[485] = d485
			ps693.OverlayValues[486] = d486
			ps693.OverlayValues[487] = d487
			ps693.OverlayValues[488] = d488
			ps693.OverlayValues[489] = d489
			ps693.OverlayValues[490] = d490
			ps693.OverlayValues[672] = d672
			ps693.OverlayValues[673] = d673
			ps693.OverlayValues[674] = d674
			ps693.OverlayValues[675] = d675
			ps693.OverlayValues[676] = d676
			ps693.OverlayValues[678] = d678
			ps693.OverlayValues[679] = d679
			ps693.OverlayValues[680] = d680
			ps693.OverlayValues[681] = d681
			ps693.OverlayValues[682] = d682
			ps693.OverlayValues[683] = d683
			ps693.OverlayValues[684] = d684
			ps693.OverlayValues[685] = d685
			ps693.OverlayValues[687] = d687
			ps693.OverlayValues[689] = d689
			ps693.OverlayValues[690] = d690
			ps693.OverlayValues[691] = d691
			ps693.OverlayValues[692] = d692
			ps693.PhiValues = make([]scm.JITValueDesc, 1)
			d695 = d7
			ps693.PhiValues[0] = d695
			ps694 := scm.PhiState{General: true}
			ps694.OverlayValues = make([]scm.JITValueDesc, 696)
			ps694.OverlayValues[0] = d0
			ps694.OverlayValues[1] = d1
			ps694.OverlayValues[2] = d2
			ps694.OverlayValues[3] = d3
			ps694.OverlayValues[4] = d4
			ps694.OverlayValues[5] = d5
			ps694.OverlayValues[6] = d6
			ps694.OverlayValues[7] = d7
			ps694.OverlayValues[8] = d8
			ps694.OverlayValues[9] = d9
			ps694.OverlayValues[10] = d10
			ps694.OverlayValues[11] = d11
			ps694.OverlayValues[12] = d12
			ps694.OverlayValues[13] = d13
			ps694.OverlayValues[14] = d14
			ps694.OverlayValues[15] = d15
			ps694.OverlayValues[16] = d16
			ps694.OverlayValues[18] = d18
			ps694.OverlayValues[19] = d19
			ps694.OverlayValues[20] = d20
			ps694.OverlayValues[21] = d21
			ps694.OverlayValues[22] = d22
			ps694.OverlayValues[23] = d23
			ps694.OverlayValues[24] = d24
			ps694.OverlayValues[25] = d25
			ps694.OverlayValues[26] = d26
			ps694.OverlayValues[27] = d27
			ps694.OverlayValues[28] = d28
			ps694.OverlayValues[29] = d29
			ps694.OverlayValues[30] = d30
			ps694.OverlayValues[31] = d31
			ps694.OverlayValues[32] = d32
			ps694.OverlayValues[33] = d33
			ps694.OverlayValues[34] = d34
			ps694.OverlayValues[35] = d35
			ps694.OverlayValues[36] = d36
			ps694.OverlayValues[37] = d37
			ps694.OverlayValues[38] = d38
			ps694.OverlayValues[39] = d39
			ps694.OverlayValues[40] = d40
			ps694.OverlayValues[41] = d41
			ps694.OverlayValues[42] = d42
			ps694.OverlayValues[43] = d43
			ps694.OverlayValues[44] = d44
			ps694.OverlayValues[45] = d45
			ps694.OverlayValues[46] = d46
			ps694.OverlayValues[47] = d47
			ps694.OverlayValues[48] = d48
			ps694.OverlayValues[49] = d49
			ps694.OverlayValues[50] = d50
			ps694.OverlayValues[51] = d51
			ps694.OverlayValues[52] = d52
			ps694.OverlayValues[53] = d53
			ps694.OverlayValues[54] = d54
			ps694.OverlayValues[55] = d55
			ps694.OverlayValues[56] = d56
			ps694.OverlayValues[57] = d57
			ps694.OverlayValues[58] = d58
			ps694.OverlayValues[59] = d59
			ps694.OverlayValues[60] = d60
			ps694.OverlayValues[61] = d61
			ps694.OverlayValues[64] = d64
			ps694.OverlayValues[65] = d65
			ps694.OverlayValues[66] = d66
			ps694.OverlayValues[134] = d134
			ps694.OverlayValues[135] = d135
			ps694.OverlayValues[136] = d136
			ps694.OverlayValues[137] = d137
			ps694.OverlayValues[138] = d138
			ps694.OverlayValues[139] = d139
			ps694.OverlayValues[140] = d140
			ps694.OverlayValues[141] = d141
			ps694.OverlayValues[142] = d142
			ps694.OverlayValues[143] = d143
			ps694.OverlayValues[144] = d144
			ps694.OverlayValues[145] = d145
			ps694.OverlayValues[146] = d146
			ps694.OverlayValues[147] = d147
			ps694.OverlayValues[148] = d148
			ps694.OverlayValues[149] = d149
			ps694.OverlayValues[150] = d150
			ps694.OverlayValues[151] = d151
			ps694.OverlayValues[152] = d152
			ps694.OverlayValues[153] = d153
			ps694.OverlayValues[154] = d154
			ps694.OverlayValues[155] = d155
			ps694.OverlayValues[156] = d156
			ps694.OverlayValues[157] = d157
			ps694.OverlayValues[158] = d158
			ps694.OverlayValues[159] = d159
			ps694.OverlayValues[160] = d160
			ps694.OverlayValues[161] = d161
			ps694.OverlayValues[162] = d162
			ps694.OverlayValues[163] = d163
			ps694.OverlayValues[164] = d164
			ps694.OverlayValues[165] = d165
			ps694.OverlayValues[166] = d166
			ps694.OverlayValues[167] = d167
			ps694.OverlayValues[168] = d168
			ps694.OverlayValues[169] = d169
			ps694.OverlayValues[170] = d170
			ps694.OverlayValues[171] = d171
			ps694.OverlayValues[172] = d172
			ps694.OverlayValues[175] = d175
			ps694.OverlayValues[283] = d283
			ps694.OverlayValues[284] = d284
			ps694.OverlayValues[285] = d285
			ps694.OverlayValues[286] = d286
			ps694.OverlayValues[287] = d287
			ps694.OverlayValues[288] = d288
			ps694.OverlayValues[289] = d289
			ps694.OverlayValues[290] = d290
			ps694.OverlayValues[292] = d292
			ps694.OverlayValues[293] = d293
			ps694.OverlayValues[294] = d294
			ps694.OverlayValues[295] = d295
			ps694.OverlayValues[296] = d296
			ps694.OverlayValues[297] = d297
			ps694.OverlayValues[298] = d298
			ps694.OverlayValues[299] = d299
			ps694.OverlayValues[300] = d300
			ps694.OverlayValues[301] = d301
			ps694.OverlayValues[303] = d303
			ps694.OverlayValues[305] = d305
			ps694.OverlayValues[306] = d306
			ps694.OverlayValues[307] = d307
			ps694.OverlayValues[308] = d308
			ps694.OverlayValues[309] = d309
			ps694.OverlayValues[312] = d312
			ps694.OverlayValues[443] = d443
			ps694.OverlayValues[444] = d444
			ps694.OverlayValues[445] = d445
			ps694.OverlayValues[446] = d446
			ps694.OverlayValues[447] = d447
			ps694.OverlayValues[448] = d448
			ps694.OverlayValues[449] = d449
			ps694.OverlayValues[451] = d451
			ps694.OverlayValues[452] = d452
			ps694.OverlayValues[453] = d453
			ps694.OverlayValues[454] = d454
			ps694.OverlayValues[455] = d455
			ps694.OverlayValues[456] = d456
			ps694.OverlayValues[457] = d457
			ps694.OverlayValues[458] = d458
			ps694.OverlayValues[459] = d459
			ps694.OverlayValues[460] = d460
			ps694.OverlayValues[461] = d461
			ps694.OverlayValues[462] = d462
			ps694.OverlayValues[463] = d463
			ps694.OverlayValues[464] = d464
			ps694.OverlayValues[465] = d465
			ps694.OverlayValues[466] = d466
			ps694.OverlayValues[467] = d467
			ps694.OverlayValues[468] = d468
			ps694.OverlayValues[469] = d469
			ps694.OverlayValues[470] = d470
			ps694.OverlayValues[471] = d471
			ps694.OverlayValues[472] = d472
			ps694.OverlayValues[473] = d473
			ps694.OverlayValues[474] = d474
			ps694.OverlayValues[475] = d475
			ps694.OverlayValues[476] = d476
			ps694.OverlayValues[477] = d477
			ps694.OverlayValues[478] = d478
			ps694.OverlayValues[479] = d479
			ps694.OverlayValues[480] = d480
			ps694.OverlayValues[481] = d481
			ps694.OverlayValues[482] = d482
			ps694.OverlayValues[483] = d483
			ps694.OverlayValues[484] = d484
			ps694.OverlayValues[485] = d485
			ps694.OverlayValues[486] = d486
			ps694.OverlayValues[487] = d487
			ps694.OverlayValues[488] = d488
			ps694.OverlayValues[489] = d489
			ps694.OverlayValues[490] = d490
			ps694.OverlayValues[672] = d672
			ps694.OverlayValues[673] = d673
			ps694.OverlayValues[674] = d674
			ps694.OverlayValues[675] = d675
			ps694.OverlayValues[676] = d676
			ps694.OverlayValues[678] = d678
			ps694.OverlayValues[679] = d679
			ps694.OverlayValues[680] = d680
			ps694.OverlayValues[681] = d681
			ps694.OverlayValues[682] = d682
			ps694.OverlayValues[683] = d683
			ps694.OverlayValues[684] = d684
			ps694.OverlayValues[685] = d685
			ps694.OverlayValues[687] = d687
			ps694.OverlayValues[689] = d689
			ps694.OverlayValues[690] = d690
			ps694.OverlayValues[691] = d691
			ps694.OverlayValues[692] = d692
			ps694.OverlayValues[695] = d695
			snap696 := d0
			snap697 := d1
			snap698 := d2
			snap699 := d3
			snap700 := d4
			snap701 := d5
			snap702 := d6
			snap703 := d7
			snap704 := d8
			snap705 := d9
			snap706 := d10
			snap707 := d11
			snap708 := d12
			snap709 := d13
			snap710 := d14
			snap711 := d15
			snap712 := d16
			snap713 := d18
			snap714 := d19
			snap715 := d20
			snap716 := d21
			snap717 := d22
			snap718 := d23
			snap719 := d24
			snap720 := d25
			snap721 := d26
			snap722 := d27
			snap723 := d28
			snap724 := d29
			snap725 := d30
			snap726 := d31
			snap727 := d32
			snap728 := d33
			snap729 := d34
			snap730 := d35
			snap731 := d36
			snap732 := d37
			snap733 := d38
			snap734 := d39
			snap735 := d40
			snap736 := d41
			snap737 := d42
			snap738 := d43
			snap739 := d44
			snap740 := d45
			snap741 := d46
			snap742 := d47
			snap743 := d48
			snap744 := d49
			snap745 := d50
			snap746 := d51
			snap747 := d52
			snap748 := d53
			snap749 := d54
			snap750 := d55
			snap751 := d56
			snap752 := d57
			snap753 := d58
			snap754 := d59
			snap755 := d60
			snap756 := d61
			snap757 := d64
			snap758 := d65
			snap759 := d66
			snap760 := d134
			snap761 := d135
			snap762 := d136
			snap763 := d137
			snap764 := d138
			snap765 := d139
			snap766 := d140
			snap767 := d141
			snap768 := d142
			snap769 := d143
			snap770 := d144
			snap771 := d145
			snap772 := d146
			snap773 := d147
			snap774 := d148
			snap775 := d149
			snap776 := d150
			snap777 := d151
			snap778 := d152
			snap779 := d153
			snap780 := d154
			snap781 := d155
			snap782 := d156
			snap783 := d157
			snap784 := d158
			snap785 := d159
			snap786 := d160
			snap787 := d161
			snap788 := d162
			snap789 := d163
			snap790 := d164
			snap791 := d165
			snap792 := d166
			snap793 := d167
			snap794 := d168
			snap795 := d169
			snap796 := d170
			snap797 := d171
			snap798 := d172
			snap799 := d175
			snap800 := d283
			snap801 := d284
			snap802 := d285
			snap803 := d286
			snap804 := d287
			snap805 := d288
			snap806 := d289
			snap807 := d290
			snap808 := d292
			snap809 := d293
			snap810 := d294
			snap811 := d295
			snap812 := d296
			snap813 := d297
			snap814 := d298
			snap815 := d299
			snap816 := d300
			snap817 := d301
			snap818 := d303
			snap819 := d305
			snap820 := d306
			snap821 := d307
			snap822 := d308
			snap823 := d309
			snap824 := d312
			snap825 := d443
			snap826 := d444
			snap827 := d445
			snap828 := d446
			snap829 := d447
			snap830 := d448
			snap831 := d449
			snap832 := d451
			snap833 := d452
			snap834 := d453
			snap835 := d454
			snap836 := d455
			snap837 := d456
			snap838 := d457
			snap839 := d458
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
			snap872 := d672
			snap873 := d673
			snap874 := d674
			snap875 := d675
			snap876 := d676
			snap877 := d678
			snap878 := d679
			snap879 := d680
			snap880 := d681
			snap881 := d682
			snap882 := d683
			snap883 := d684
			snap884 := d685
			snap885 := d687
			snap886 := d689
			snap887 := d690
			snap888 := d691
			snap889 := d692
			snap890 := d695
			alloc891 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps693)
			}
			ctx.RestoreAllocState(alloc891)
			d0 = snap696
			d1 = snap697
			d2 = snap698
			d3 = snap699
			d4 = snap700
			d5 = snap701
			d6 = snap702
			d7 = snap703
			d8 = snap704
			d9 = snap705
			d10 = snap706
			d11 = snap707
			d12 = snap708
			d13 = snap709
			d14 = snap710
			d15 = snap711
			d16 = snap712
			d18 = snap713
			d19 = snap714
			d20 = snap715
			d21 = snap716
			d22 = snap717
			d23 = snap718
			d24 = snap719
			d25 = snap720
			d26 = snap721
			d27 = snap722
			d28 = snap723
			d29 = snap724
			d30 = snap725
			d31 = snap726
			d32 = snap727
			d33 = snap728
			d34 = snap729
			d35 = snap730
			d36 = snap731
			d37 = snap732
			d38 = snap733
			d39 = snap734
			d40 = snap735
			d41 = snap736
			d42 = snap737
			d43 = snap738
			d44 = snap739
			d45 = snap740
			d46 = snap741
			d47 = snap742
			d48 = snap743
			d49 = snap744
			d50 = snap745
			d51 = snap746
			d52 = snap747
			d53 = snap748
			d54 = snap749
			d55 = snap750
			d56 = snap751
			d57 = snap752
			d58 = snap753
			d59 = snap754
			d60 = snap755
			d61 = snap756
			d64 = snap757
			d65 = snap758
			d66 = snap759
			d134 = snap760
			d135 = snap761
			d136 = snap762
			d137 = snap763
			d138 = snap764
			d139 = snap765
			d140 = snap766
			d141 = snap767
			d142 = snap768
			d143 = snap769
			d144 = snap770
			d145 = snap771
			d146 = snap772
			d147 = snap773
			d148 = snap774
			d149 = snap775
			d150 = snap776
			d151 = snap777
			d152 = snap778
			d153 = snap779
			d154 = snap780
			d155 = snap781
			d156 = snap782
			d157 = snap783
			d158 = snap784
			d159 = snap785
			d160 = snap786
			d161 = snap787
			d162 = snap788
			d163 = snap789
			d164 = snap790
			d165 = snap791
			d166 = snap792
			d167 = snap793
			d168 = snap794
			d169 = snap795
			d170 = snap796
			d171 = snap797
			d172 = snap798
			d175 = snap799
			d283 = snap800
			d284 = snap801
			d285 = snap802
			d286 = snap803
			d287 = snap804
			d288 = snap805
			d289 = snap806
			d290 = snap807
			d292 = snap808
			d293 = snap809
			d294 = snap810
			d295 = snap811
			d296 = snap812
			d297 = snap813
			d298 = snap814
			d299 = snap815
			d300 = snap816
			d301 = snap817
			d303 = snap818
			d305 = snap819
			d306 = snap820
			d307 = snap821
			d308 = snap822
			d309 = snap823
			d312 = snap824
			d443 = snap825
			d444 = snap826
			d445 = snap827
			d446 = snap828
			d447 = snap829
			d448 = snap830
			d449 = snap831
			d451 = snap832
			d452 = snap833
			d453 = snap834
			d454 = snap835
			d455 = snap836
			d456 = snap837
			d457 = snap838
			d458 = snap839
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
			d672 = snap872
			d673 = snap873
			d674 = snap874
			d675 = snap875
			d676 = snap876
			d678 = snap877
			d679 = snap878
			d680 = snap879
			d681 = snap880
			d682 = snap881
			d683 = snap882
			d684 = snap883
			d685 = snap884
			d687 = snap885
			d689 = snap886
			d690 = snap887
			d691 = snap888
			d692 = snap889
			d695 = snap890
			if !bbs[10].Rendered {
				return bbs[10].RenderPS(ps694)
			}
			return result
			ctx.FreeDesc(&d682)
			return result
			}
			bbs[9].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[9].VisitCount >= 2 {
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
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
				d14 = ps.OverlayValues[14]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
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
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != scm.LocNone {
				d283 = ps.OverlayValues[283]
			}
			if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != scm.LocNone {
				d284 = ps.OverlayValues[284]
			}
			if len(ps.OverlayValues) > 285 && ps.OverlayValues[285].Loc != scm.LocNone {
				d285 = ps.OverlayValues[285]
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
			if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != scm.LocNone {
				d292 = ps.OverlayValues[292]
			}
			if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != scm.LocNone {
				d293 = ps.OverlayValues[293]
			}
			if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
				d294 = ps.OverlayValues[294]
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
			if len(ps.OverlayValues) > 303 && ps.OverlayValues[303].Loc != scm.LocNone {
				d303 = ps.OverlayValues[303]
			}
			if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != scm.LocNone {
				d305 = ps.OverlayValues[305]
			}
			if len(ps.OverlayValues) > 306 && ps.OverlayValues[306].Loc != scm.LocNone {
				d306 = ps.OverlayValues[306]
			}
			if len(ps.OverlayValues) > 307 && ps.OverlayValues[307].Loc != scm.LocNone {
				d307 = ps.OverlayValues[307]
			}
			if len(ps.OverlayValues) > 308 && ps.OverlayValues[308].Loc != scm.LocNone {
				d308 = ps.OverlayValues[308]
			}
			if len(ps.OverlayValues) > 309 && ps.OverlayValues[309].Loc != scm.LocNone {
				d309 = ps.OverlayValues[309]
			}
			if len(ps.OverlayValues) > 312 && ps.OverlayValues[312].Loc != scm.LocNone {
				d312 = ps.OverlayValues[312]
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
			if len(ps.OverlayValues) > 672 && ps.OverlayValues[672].Loc != scm.LocNone {
				d672 = ps.OverlayValues[672]
			}
			if len(ps.OverlayValues) > 673 && ps.OverlayValues[673].Loc != scm.LocNone {
				d673 = ps.OverlayValues[673]
			}
			if len(ps.OverlayValues) > 674 && ps.OverlayValues[674].Loc != scm.LocNone {
				d674 = ps.OverlayValues[674]
			}
			if len(ps.OverlayValues) > 675 && ps.OverlayValues[675].Loc != scm.LocNone {
				d675 = ps.OverlayValues[675]
			}
			if len(ps.OverlayValues) > 676 && ps.OverlayValues[676].Loc != scm.LocNone {
				d676 = ps.OverlayValues[676]
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
			if len(ps.OverlayValues) > 681 && ps.OverlayValues[681].Loc != scm.LocNone {
				d681 = ps.OverlayValues[681]
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
			if len(ps.OverlayValues) > 687 && ps.OverlayValues[687].Loc != scm.LocNone {
				d687 = ps.OverlayValues[687]
			}
			if len(ps.OverlayValues) > 689 && ps.OverlayValues[689].Loc != scm.LocNone {
				d689 = ps.OverlayValues[689]
			}
			if len(ps.OverlayValues) > 690 && ps.OverlayValues[690].Loc != scm.LocNone {
				d690 = ps.OverlayValues[690]
			}
			if len(ps.OverlayValues) > 691 && ps.OverlayValues[691].Loc != scm.LocNone {
				d691 = ps.OverlayValues[691]
			}
			if len(ps.OverlayValues) > 692 && ps.OverlayValues[692].Loc != scm.LocNone {
				d692 = ps.OverlayValues[692]
			}
			if len(ps.OverlayValues) > 695 && ps.OverlayValues[695].Loc != scm.LocNone {
				d695 = ps.OverlayValues[695]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d4)
			if d4.Loc == scm.LocReg {
				ctx.ProtectReg(d4.Reg)
			} else if d4.Loc == scm.LocRegPair {
				ctx.ProtectReg(d4.Reg)
				ctx.ProtectReg(d4.Reg2)
			}
			ctx.EnsureDesc(&d6)
			if d6.Loc == scm.LocReg {
				ctx.ProtectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.ProtectReg(d6.Reg)
				ctx.ProtectReg(d6.Reg2)
			}
			d892 = d4
			if d892.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d892)
			d893 = d892
			if d893.Loc == scm.LocImm {
				d893 = scm.JITValueDesc{Loc: scm.LocImm, Type: d893.Type, Imm: scm.NewInt(int64(uint64(d893.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d893.Reg, 32)
				ctx.EmitShrRegImm8(d893.Reg, 32)
			}
			ctx.EmitStoreToStack(d893, int32(bbs[8].PhiBase)+int32(0))
			d894 = d6
			if d894.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d894)
			d895 = d894
			if d895.Loc == scm.LocImm {
				d895 = scm.JITValueDesc{Loc: scm.LocImm, Type: d895.Type, Imm: scm.NewInt(int64(uint64(d895.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d895.Reg, 32)
				ctx.EmitShrRegImm8(d895.Reg, 32)
			}
			ctx.EmitStoreToStack(d895, int32(bbs[8].PhiBase)+int32(16))
			if d4.Loc == scm.LocReg {
				ctx.UnprotectReg(d4.Reg)
			} else if d4.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d4.Reg)
				ctx.UnprotectReg(d4.Reg2)
			}
			if d6.Loc == scm.LocReg {
				ctx.UnprotectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d6.Reg)
				ctx.UnprotectReg(d6.Reg2)
			}
			ps896 := scm.PhiState{General: ps.General}
			ps896.OverlayValues = make([]scm.JITValueDesc, 896)
			ps896.OverlayValues[0] = d0
			ps896.OverlayValues[1] = d1
			ps896.OverlayValues[2] = d2
			ps896.OverlayValues[3] = d3
			ps896.OverlayValues[4] = d4
			ps896.OverlayValues[5] = d5
			ps896.OverlayValues[6] = d6
			ps896.OverlayValues[7] = d7
			ps896.OverlayValues[8] = d8
			ps896.OverlayValues[9] = d9
			ps896.OverlayValues[10] = d10
			ps896.OverlayValues[11] = d11
			ps896.OverlayValues[12] = d12
			ps896.OverlayValues[13] = d13
			ps896.OverlayValues[14] = d14
			ps896.OverlayValues[15] = d15
			ps896.OverlayValues[16] = d16
			ps896.OverlayValues[18] = d18
			ps896.OverlayValues[19] = d19
			ps896.OverlayValues[20] = d20
			ps896.OverlayValues[21] = d21
			ps896.OverlayValues[22] = d22
			ps896.OverlayValues[23] = d23
			ps896.OverlayValues[24] = d24
			ps896.OverlayValues[25] = d25
			ps896.OverlayValues[26] = d26
			ps896.OverlayValues[27] = d27
			ps896.OverlayValues[28] = d28
			ps896.OverlayValues[29] = d29
			ps896.OverlayValues[30] = d30
			ps896.OverlayValues[31] = d31
			ps896.OverlayValues[32] = d32
			ps896.OverlayValues[33] = d33
			ps896.OverlayValues[34] = d34
			ps896.OverlayValues[35] = d35
			ps896.OverlayValues[36] = d36
			ps896.OverlayValues[37] = d37
			ps896.OverlayValues[38] = d38
			ps896.OverlayValues[39] = d39
			ps896.OverlayValues[40] = d40
			ps896.OverlayValues[41] = d41
			ps896.OverlayValues[42] = d42
			ps896.OverlayValues[43] = d43
			ps896.OverlayValues[44] = d44
			ps896.OverlayValues[45] = d45
			ps896.OverlayValues[46] = d46
			ps896.OverlayValues[47] = d47
			ps896.OverlayValues[48] = d48
			ps896.OverlayValues[49] = d49
			ps896.OverlayValues[50] = d50
			ps896.OverlayValues[51] = d51
			ps896.OverlayValues[52] = d52
			ps896.OverlayValues[53] = d53
			ps896.OverlayValues[54] = d54
			ps896.OverlayValues[55] = d55
			ps896.OverlayValues[56] = d56
			ps896.OverlayValues[57] = d57
			ps896.OverlayValues[58] = d58
			ps896.OverlayValues[59] = d59
			ps896.OverlayValues[60] = d60
			ps896.OverlayValues[61] = d61
			ps896.OverlayValues[64] = d64
			ps896.OverlayValues[65] = d65
			ps896.OverlayValues[66] = d66
			ps896.OverlayValues[134] = d134
			ps896.OverlayValues[135] = d135
			ps896.OverlayValues[136] = d136
			ps896.OverlayValues[137] = d137
			ps896.OverlayValues[138] = d138
			ps896.OverlayValues[139] = d139
			ps896.OverlayValues[140] = d140
			ps896.OverlayValues[141] = d141
			ps896.OverlayValues[142] = d142
			ps896.OverlayValues[143] = d143
			ps896.OverlayValues[144] = d144
			ps896.OverlayValues[145] = d145
			ps896.OverlayValues[146] = d146
			ps896.OverlayValues[147] = d147
			ps896.OverlayValues[148] = d148
			ps896.OverlayValues[149] = d149
			ps896.OverlayValues[150] = d150
			ps896.OverlayValues[151] = d151
			ps896.OverlayValues[152] = d152
			ps896.OverlayValues[153] = d153
			ps896.OverlayValues[154] = d154
			ps896.OverlayValues[155] = d155
			ps896.OverlayValues[156] = d156
			ps896.OverlayValues[157] = d157
			ps896.OverlayValues[158] = d158
			ps896.OverlayValues[159] = d159
			ps896.OverlayValues[160] = d160
			ps896.OverlayValues[161] = d161
			ps896.OverlayValues[162] = d162
			ps896.OverlayValues[163] = d163
			ps896.OverlayValues[164] = d164
			ps896.OverlayValues[165] = d165
			ps896.OverlayValues[166] = d166
			ps896.OverlayValues[167] = d167
			ps896.OverlayValues[168] = d168
			ps896.OverlayValues[169] = d169
			ps896.OverlayValues[170] = d170
			ps896.OverlayValues[171] = d171
			ps896.OverlayValues[172] = d172
			ps896.OverlayValues[175] = d175
			ps896.OverlayValues[283] = d283
			ps896.OverlayValues[284] = d284
			ps896.OverlayValues[285] = d285
			ps896.OverlayValues[286] = d286
			ps896.OverlayValues[287] = d287
			ps896.OverlayValues[288] = d288
			ps896.OverlayValues[289] = d289
			ps896.OverlayValues[290] = d290
			ps896.OverlayValues[292] = d292
			ps896.OverlayValues[293] = d293
			ps896.OverlayValues[294] = d294
			ps896.OverlayValues[295] = d295
			ps896.OverlayValues[296] = d296
			ps896.OverlayValues[297] = d297
			ps896.OverlayValues[298] = d298
			ps896.OverlayValues[299] = d299
			ps896.OverlayValues[300] = d300
			ps896.OverlayValues[301] = d301
			ps896.OverlayValues[303] = d303
			ps896.OverlayValues[305] = d305
			ps896.OverlayValues[306] = d306
			ps896.OverlayValues[307] = d307
			ps896.OverlayValues[308] = d308
			ps896.OverlayValues[309] = d309
			ps896.OverlayValues[312] = d312
			ps896.OverlayValues[443] = d443
			ps896.OverlayValues[444] = d444
			ps896.OverlayValues[445] = d445
			ps896.OverlayValues[446] = d446
			ps896.OverlayValues[447] = d447
			ps896.OverlayValues[448] = d448
			ps896.OverlayValues[449] = d449
			ps896.OverlayValues[451] = d451
			ps896.OverlayValues[452] = d452
			ps896.OverlayValues[453] = d453
			ps896.OverlayValues[454] = d454
			ps896.OverlayValues[455] = d455
			ps896.OverlayValues[456] = d456
			ps896.OverlayValues[457] = d457
			ps896.OverlayValues[458] = d458
			ps896.OverlayValues[459] = d459
			ps896.OverlayValues[460] = d460
			ps896.OverlayValues[461] = d461
			ps896.OverlayValues[462] = d462
			ps896.OverlayValues[463] = d463
			ps896.OverlayValues[464] = d464
			ps896.OverlayValues[465] = d465
			ps896.OverlayValues[466] = d466
			ps896.OverlayValues[467] = d467
			ps896.OverlayValues[468] = d468
			ps896.OverlayValues[469] = d469
			ps896.OverlayValues[470] = d470
			ps896.OverlayValues[471] = d471
			ps896.OverlayValues[472] = d472
			ps896.OverlayValues[473] = d473
			ps896.OverlayValues[474] = d474
			ps896.OverlayValues[475] = d475
			ps896.OverlayValues[476] = d476
			ps896.OverlayValues[477] = d477
			ps896.OverlayValues[478] = d478
			ps896.OverlayValues[479] = d479
			ps896.OverlayValues[480] = d480
			ps896.OverlayValues[481] = d481
			ps896.OverlayValues[482] = d482
			ps896.OverlayValues[483] = d483
			ps896.OverlayValues[484] = d484
			ps896.OverlayValues[485] = d485
			ps896.OverlayValues[486] = d486
			ps896.OverlayValues[487] = d487
			ps896.OverlayValues[488] = d488
			ps896.OverlayValues[489] = d489
			ps896.OverlayValues[490] = d490
			ps896.OverlayValues[672] = d672
			ps896.OverlayValues[673] = d673
			ps896.OverlayValues[674] = d674
			ps896.OverlayValues[675] = d675
			ps896.OverlayValues[676] = d676
			ps896.OverlayValues[678] = d678
			ps896.OverlayValues[679] = d679
			ps896.OverlayValues[680] = d680
			ps896.OverlayValues[681] = d681
			ps896.OverlayValues[682] = d682
			ps896.OverlayValues[683] = d683
			ps896.OverlayValues[684] = d684
			ps896.OverlayValues[685] = d685
			ps896.OverlayValues[687] = d687
			ps896.OverlayValues[689] = d689
			ps896.OverlayValues[690] = d690
			ps896.OverlayValues[691] = d691
			ps896.OverlayValues[692] = d692
			ps896.OverlayValues[695] = d695
			ps896.OverlayValues[892] = d892
			ps896.OverlayValues[893] = d893
			ps896.OverlayValues[894] = d894
			ps896.OverlayValues[895] = d895
			ps896.PhiValues = make([]scm.JITValueDesc, 2)
			d897 = d4
			ps896.PhiValues[0] = d897
			d898 = d6
			ps896.PhiValues[1] = d898
			if ps896.General && bbs[8].Rendered {
				ctx.EmitJmp(lbl9)
				return result
			}
			return bbs[8].RenderPS(ps896)
			return result
			}
			bbs[10].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[10].VisitCount >= 2 {
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
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
				d14 = ps.OverlayValues[14]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
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
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != scm.LocNone {
				d283 = ps.OverlayValues[283]
			}
			if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != scm.LocNone {
				d284 = ps.OverlayValues[284]
			}
			if len(ps.OverlayValues) > 285 && ps.OverlayValues[285].Loc != scm.LocNone {
				d285 = ps.OverlayValues[285]
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
			if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != scm.LocNone {
				d292 = ps.OverlayValues[292]
			}
			if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != scm.LocNone {
				d293 = ps.OverlayValues[293]
			}
			if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
				d294 = ps.OverlayValues[294]
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
			if len(ps.OverlayValues) > 303 && ps.OverlayValues[303].Loc != scm.LocNone {
				d303 = ps.OverlayValues[303]
			}
			if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != scm.LocNone {
				d305 = ps.OverlayValues[305]
			}
			if len(ps.OverlayValues) > 306 && ps.OverlayValues[306].Loc != scm.LocNone {
				d306 = ps.OverlayValues[306]
			}
			if len(ps.OverlayValues) > 307 && ps.OverlayValues[307].Loc != scm.LocNone {
				d307 = ps.OverlayValues[307]
			}
			if len(ps.OverlayValues) > 308 && ps.OverlayValues[308].Loc != scm.LocNone {
				d308 = ps.OverlayValues[308]
			}
			if len(ps.OverlayValues) > 309 && ps.OverlayValues[309].Loc != scm.LocNone {
				d309 = ps.OverlayValues[309]
			}
			if len(ps.OverlayValues) > 312 && ps.OverlayValues[312].Loc != scm.LocNone {
				d312 = ps.OverlayValues[312]
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
			if len(ps.OverlayValues) > 672 && ps.OverlayValues[672].Loc != scm.LocNone {
				d672 = ps.OverlayValues[672]
			}
			if len(ps.OverlayValues) > 673 && ps.OverlayValues[673].Loc != scm.LocNone {
				d673 = ps.OverlayValues[673]
			}
			if len(ps.OverlayValues) > 674 && ps.OverlayValues[674].Loc != scm.LocNone {
				d674 = ps.OverlayValues[674]
			}
			if len(ps.OverlayValues) > 675 && ps.OverlayValues[675].Loc != scm.LocNone {
				d675 = ps.OverlayValues[675]
			}
			if len(ps.OverlayValues) > 676 && ps.OverlayValues[676].Loc != scm.LocNone {
				d676 = ps.OverlayValues[676]
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
			if len(ps.OverlayValues) > 681 && ps.OverlayValues[681].Loc != scm.LocNone {
				d681 = ps.OverlayValues[681]
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
			if len(ps.OverlayValues) > 687 && ps.OverlayValues[687].Loc != scm.LocNone {
				d687 = ps.OverlayValues[687]
			}
			if len(ps.OverlayValues) > 689 && ps.OverlayValues[689].Loc != scm.LocNone {
				d689 = ps.OverlayValues[689]
			}
			if len(ps.OverlayValues) > 690 && ps.OverlayValues[690].Loc != scm.LocNone {
				d690 = ps.OverlayValues[690]
			}
			if len(ps.OverlayValues) > 691 && ps.OverlayValues[691].Loc != scm.LocNone {
				d691 = ps.OverlayValues[691]
			}
			if len(ps.OverlayValues) > 692 && ps.OverlayValues[692].Loc != scm.LocNone {
				d692 = ps.OverlayValues[692]
			}
			if len(ps.OverlayValues) > 695 && ps.OverlayValues[695].Loc != scm.LocNone {
				d695 = ps.OverlayValues[695]
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
			if len(ps.OverlayValues) > 897 && ps.OverlayValues[897].Loc != scm.LocNone {
				d897 = ps.OverlayValues[897]
			}
			if len(ps.OverlayValues) > 898 && ps.OverlayValues[898].Loc != scm.LocNone {
				d898 = ps.OverlayValues[898]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d8)
			var d899 scm.JITValueDesc
			if d7.Loc == scm.LocImm && d8.Loc == scm.LocImm {
				d899 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d7.Imm.Int() + d8.Imm.Int())}
			} else if d8.Loc == scm.LocImm && d8.Imm.Int() == 0 {
				r154 := ctx.AllocRegExcept(d7.Reg)
				ctx.EmitMovRegReg(r154, d7.Reg)
				d899 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r154}
				ctx.BindReg(r154, &d899)
			} else if d7.Loc == scm.LocImm && d7.Imm.Int() == 0 {
				d899 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d8.Reg}
				ctx.BindReg(d8.Reg, &d899)
			} else if d7.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d8.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d7.Imm.Int()))
				ctx.EmitAddInt64(scratch, d8.Reg)
				d899 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d899)
			} else if d8.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d7.Reg)
				ctx.EmitMovRegReg(scratch, d7.Reg)
				if d8.Imm.Int() >= -2147483648 && d8.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d8.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d8.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d899 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d899)
			} else {
				r155 := ctx.AllocRegExcept(d7.Reg, d8.Reg)
				ctx.EmitMovRegReg(r155, d7.Reg)
				ctx.EmitAddInt64(r155, d8.Reg)
				d899 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r155}
				ctx.BindReg(r155, &d899)
			}
			if d899.Loc == scm.LocImm {
				d899 = scm.JITValueDesc{Loc: scm.LocImm, Type: d899.Type, Imm: scm.NewInt(int64(uint64(d899.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d899.Reg, 32)
				ctx.EmitShrRegImm8(d899.Reg, 32)
			}
			if d899.Loc == scm.LocReg && d7.Loc == scm.LocReg && d899.Reg == d7.Reg {
				ctx.TransferReg(d7.Reg)
				d7.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d899)
			var d900 scm.JITValueDesc
			if d899.Loc == scm.LocImm {
				d900 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d899.Imm.Int() / 2)}
			} else {
				r156 := ctx.AllocRegExcept(d899.Reg)
				ctx.EmitMovRegReg(r156, d899.Reg)
				ctx.EmitShrRegImm8(r156, 1)
				d900 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r156}
				ctx.BindReg(r156, &d900)
			}
			if d900.Loc == scm.LocImm {
				d900 = scm.JITValueDesc{Loc: scm.LocImm, Type: d900.Type, Imm: scm.NewInt(int64(uint64(d900.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d900.Reg, 32)
				ctx.EmitShrRegImm8(d900.Reg, 32)
			}
			if d900.Loc == scm.LocReg && d899.Loc == scm.LocReg && d900.Reg == d899.Reg {
				ctx.TransferReg(d899.Reg)
				d899.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d899)
			ctx.EnsureDesc(&d7)
			if d7.Loc == scm.LocReg {
				ctx.ProtectReg(d7.Reg)
			} else if d7.Loc == scm.LocRegPair {
				ctx.ProtectReg(d7.Reg)
				ctx.ProtectReg(d7.Reg2)
			}
			ctx.EnsureDesc(&d8)
			if d8.Loc == scm.LocReg {
				ctx.ProtectReg(d8.Reg)
			} else if d8.Loc == scm.LocRegPair {
				ctx.ProtectReg(d8.Reg)
				ctx.ProtectReg(d8.Reg2)
			}
			ctx.EnsureDesc(&d900)
			if d900.Loc == scm.LocReg {
				ctx.ProtectReg(d900.Reg)
			} else if d900.Loc == scm.LocRegPair {
				ctx.ProtectReg(d900.Reg)
				ctx.ProtectReg(d900.Reg2)
			}
			d901 = d900
			if d901.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d901)
			d902 = d901
			if d902.Loc == scm.LocImm {
				d902 = scm.JITValueDesc{Loc: scm.LocImm, Type: d902.Type, Imm: scm.NewInt(int64(uint64(d902.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d902.Reg, 32)
				ctx.EmitShrRegImm8(d902.Reg, 32)
			}
			ctx.EmitStoreToStack(d902, int32(bbs[1].PhiBase)+int32(0))
			d903 = d7
			if d903.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d903)
			d904 = d903
			if d904.Loc == scm.LocImm {
				d904 = scm.JITValueDesc{Loc: scm.LocImm, Type: d904.Type, Imm: scm.NewInt(int64(uint64(d904.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d904.Reg, 32)
				ctx.EmitShrRegImm8(d904.Reg, 32)
			}
			ctx.EmitStoreToStack(d904, int32(bbs[1].PhiBase)+int32(16))
			d905 = d8
			if d905.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d905)
			d906 = d905
			if d906.Loc == scm.LocImm {
				d906 = scm.JITValueDesc{Loc: scm.LocImm, Type: d906.Type, Imm: scm.NewInt(int64(uint64(d906.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d906.Reg, 32)
				ctx.EmitShrRegImm8(d906.Reg, 32)
			}
			ctx.EmitStoreToStack(d906, int32(bbs[1].PhiBase)+int32(32))
			if d7.Loc == scm.LocReg {
				ctx.UnprotectReg(d7.Reg)
			} else if d7.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d7.Reg)
				ctx.UnprotectReg(d7.Reg2)
			}
			if d8.Loc == scm.LocReg {
				ctx.UnprotectReg(d8.Reg)
			} else if d8.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d8.Reg)
				ctx.UnprotectReg(d8.Reg2)
			}
			if d900.Loc == scm.LocReg {
				ctx.UnprotectReg(d900.Reg)
			} else if d900.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d900.Reg)
				ctx.UnprotectReg(d900.Reg2)
			}
			ps907 := scm.PhiState{General: ps.General}
			ps907.OverlayValues = make([]scm.JITValueDesc, 907)
			ps907.OverlayValues[0] = d0
			ps907.OverlayValues[1] = d1
			ps907.OverlayValues[2] = d2
			ps907.OverlayValues[3] = d3
			ps907.OverlayValues[4] = d4
			ps907.OverlayValues[5] = d5
			ps907.OverlayValues[6] = d6
			ps907.OverlayValues[7] = d7
			ps907.OverlayValues[8] = d8
			ps907.OverlayValues[9] = d9
			ps907.OverlayValues[10] = d10
			ps907.OverlayValues[11] = d11
			ps907.OverlayValues[12] = d12
			ps907.OverlayValues[13] = d13
			ps907.OverlayValues[14] = d14
			ps907.OverlayValues[15] = d15
			ps907.OverlayValues[16] = d16
			ps907.OverlayValues[18] = d18
			ps907.OverlayValues[19] = d19
			ps907.OverlayValues[20] = d20
			ps907.OverlayValues[21] = d21
			ps907.OverlayValues[22] = d22
			ps907.OverlayValues[23] = d23
			ps907.OverlayValues[24] = d24
			ps907.OverlayValues[25] = d25
			ps907.OverlayValues[26] = d26
			ps907.OverlayValues[27] = d27
			ps907.OverlayValues[28] = d28
			ps907.OverlayValues[29] = d29
			ps907.OverlayValues[30] = d30
			ps907.OverlayValues[31] = d31
			ps907.OverlayValues[32] = d32
			ps907.OverlayValues[33] = d33
			ps907.OverlayValues[34] = d34
			ps907.OverlayValues[35] = d35
			ps907.OverlayValues[36] = d36
			ps907.OverlayValues[37] = d37
			ps907.OverlayValues[38] = d38
			ps907.OverlayValues[39] = d39
			ps907.OverlayValues[40] = d40
			ps907.OverlayValues[41] = d41
			ps907.OverlayValues[42] = d42
			ps907.OverlayValues[43] = d43
			ps907.OverlayValues[44] = d44
			ps907.OverlayValues[45] = d45
			ps907.OverlayValues[46] = d46
			ps907.OverlayValues[47] = d47
			ps907.OverlayValues[48] = d48
			ps907.OverlayValues[49] = d49
			ps907.OverlayValues[50] = d50
			ps907.OverlayValues[51] = d51
			ps907.OverlayValues[52] = d52
			ps907.OverlayValues[53] = d53
			ps907.OverlayValues[54] = d54
			ps907.OverlayValues[55] = d55
			ps907.OverlayValues[56] = d56
			ps907.OverlayValues[57] = d57
			ps907.OverlayValues[58] = d58
			ps907.OverlayValues[59] = d59
			ps907.OverlayValues[60] = d60
			ps907.OverlayValues[61] = d61
			ps907.OverlayValues[64] = d64
			ps907.OverlayValues[65] = d65
			ps907.OverlayValues[66] = d66
			ps907.OverlayValues[134] = d134
			ps907.OverlayValues[135] = d135
			ps907.OverlayValues[136] = d136
			ps907.OverlayValues[137] = d137
			ps907.OverlayValues[138] = d138
			ps907.OverlayValues[139] = d139
			ps907.OverlayValues[140] = d140
			ps907.OverlayValues[141] = d141
			ps907.OverlayValues[142] = d142
			ps907.OverlayValues[143] = d143
			ps907.OverlayValues[144] = d144
			ps907.OverlayValues[145] = d145
			ps907.OverlayValues[146] = d146
			ps907.OverlayValues[147] = d147
			ps907.OverlayValues[148] = d148
			ps907.OverlayValues[149] = d149
			ps907.OverlayValues[150] = d150
			ps907.OverlayValues[151] = d151
			ps907.OverlayValues[152] = d152
			ps907.OverlayValues[153] = d153
			ps907.OverlayValues[154] = d154
			ps907.OverlayValues[155] = d155
			ps907.OverlayValues[156] = d156
			ps907.OverlayValues[157] = d157
			ps907.OverlayValues[158] = d158
			ps907.OverlayValues[159] = d159
			ps907.OverlayValues[160] = d160
			ps907.OverlayValues[161] = d161
			ps907.OverlayValues[162] = d162
			ps907.OverlayValues[163] = d163
			ps907.OverlayValues[164] = d164
			ps907.OverlayValues[165] = d165
			ps907.OverlayValues[166] = d166
			ps907.OverlayValues[167] = d167
			ps907.OverlayValues[168] = d168
			ps907.OverlayValues[169] = d169
			ps907.OverlayValues[170] = d170
			ps907.OverlayValues[171] = d171
			ps907.OverlayValues[172] = d172
			ps907.OverlayValues[175] = d175
			ps907.OverlayValues[283] = d283
			ps907.OverlayValues[284] = d284
			ps907.OverlayValues[285] = d285
			ps907.OverlayValues[286] = d286
			ps907.OverlayValues[287] = d287
			ps907.OverlayValues[288] = d288
			ps907.OverlayValues[289] = d289
			ps907.OverlayValues[290] = d290
			ps907.OverlayValues[292] = d292
			ps907.OverlayValues[293] = d293
			ps907.OverlayValues[294] = d294
			ps907.OverlayValues[295] = d295
			ps907.OverlayValues[296] = d296
			ps907.OverlayValues[297] = d297
			ps907.OverlayValues[298] = d298
			ps907.OverlayValues[299] = d299
			ps907.OverlayValues[300] = d300
			ps907.OverlayValues[301] = d301
			ps907.OverlayValues[303] = d303
			ps907.OverlayValues[305] = d305
			ps907.OverlayValues[306] = d306
			ps907.OverlayValues[307] = d307
			ps907.OverlayValues[308] = d308
			ps907.OverlayValues[309] = d309
			ps907.OverlayValues[312] = d312
			ps907.OverlayValues[443] = d443
			ps907.OverlayValues[444] = d444
			ps907.OverlayValues[445] = d445
			ps907.OverlayValues[446] = d446
			ps907.OverlayValues[447] = d447
			ps907.OverlayValues[448] = d448
			ps907.OverlayValues[449] = d449
			ps907.OverlayValues[451] = d451
			ps907.OverlayValues[452] = d452
			ps907.OverlayValues[453] = d453
			ps907.OverlayValues[454] = d454
			ps907.OverlayValues[455] = d455
			ps907.OverlayValues[456] = d456
			ps907.OverlayValues[457] = d457
			ps907.OverlayValues[458] = d458
			ps907.OverlayValues[459] = d459
			ps907.OverlayValues[460] = d460
			ps907.OverlayValues[461] = d461
			ps907.OverlayValues[462] = d462
			ps907.OverlayValues[463] = d463
			ps907.OverlayValues[464] = d464
			ps907.OverlayValues[465] = d465
			ps907.OverlayValues[466] = d466
			ps907.OverlayValues[467] = d467
			ps907.OverlayValues[468] = d468
			ps907.OverlayValues[469] = d469
			ps907.OverlayValues[470] = d470
			ps907.OverlayValues[471] = d471
			ps907.OverlayValues[472] = d472
			ps907.OverlayValues[473] = d473
			ps907.OverlayValues[474] = d474
			ps907.OverlayValues[475] = d475
			ps907.OverlayValues[476] = d476
			ps907.OverlayValues[477] = d477
			ps907.OverlayValues[478] = d478
			ps907.OverlayValues[479] = d479
			ps907.OverlayValues[480] = d480
			ps907.OverlayValues[481] = d481
			ps907.OverlayValues[482] = d482
			ps907.OverlayValues[483] = d483
			ps907.OverlayValues[484] = d484
			ps907.OverlayValues[485] = d485
			ps907.OverlayValues[486] = d486
			ps907.OverlayValues[487] = d487
			ps907.OverlayValues[488] = d488
			ps907.OverlayValues[489] = d489
			ps907.OverlayValues[490] = d490
			ps907.OverlayValues[672] = d672
			ps907.OverlayValues[673] = d673
			ps907.OverlayValues[674] = d674
			ps907.OverlayValues[675] = d675
			ps907.OverlayValues[676] = d676
			ps907.OverlayValues[678] = d678
			ps907.OverlayValues[679] = d679
			ps907.OverlayValues[680] = d680
			ps907.OverlayValues[681] = d681
			ps907.OverlayValues[682] = d682
			ps907.OverlayValues[683] = d683
			ps907.OverlayValues[684] = d684
			ps907.OverlayValues[685] = d685
			ps907.OverlayValues[687] = d687
			ps907.OverlayValues[689] = d689
			ps907.OverlayValues[690] = d690
			ps907.OverlayValues[691] = d691
			ps907.OverlayValues[692] = d692
			ps907.OverlayValues[695] = d695
			ps907.OverlayValues[892] = d892
			ps907.OverlayValues[893] = d893
			ps907.OverlayValues[894] = d894
			ps907.OverlayValues[895] = d895
			ps907.OverlayValues[897] = d897
			ps907.OverlayValues[898] = d898
			ps907.OverlayValues[899] = d899
			ps907.OverlayValues[900] = d900
			ps907.OverlayValues[901] = d901
			ps907.OverlayValues[902] = d902
			ps907.OverlayValues[903] = d903
			ps907.OverlayValues[904] = d904
			ps907.OverlayValues[905] = d905
			ps907.OverlayValues[906] = d906
			ps907.PhiValues = make([]scm.JITValueDesc, 3)
			d908 = d900
			ps907.PhiValues[0] = d908
			d909 = d7
			ps907.PhiValues[1] = d909
			d910 = d8
			ps907.PhiValues[2] = d910
			if ps907.General && bbs[1].Rendered {
				ctx.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps907)
			return result
			}
			bbs[11].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[11].VisitCount >= 2 {
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
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
				d14 = ps.OverlayValues[14]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
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
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != scm.LocNone {
				d283 = ps.OverlayValues[283]
			}
			if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != scm.LocNone {
				d284 = ps.OverlayValues[284]
			}
			if len(ps.OverlayValues) > 285 && ps.OverlayValues[285].Loc != scm.LocNone {
				d285 = ps.OverlayValues[285]
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
			if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != scm.LocNone {
				d292 = ps.OverlayValues[292]
			}
			if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != scm.LocNone {
				d293 = ps.OverlayValues[293]
			}
			if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
				d294 = ps.OverlayValues[294]
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
			if len(ps.OverlayValues) > 303 && ps.OverlayValues[303].Loc != scm.LocNone {
				d303 = ps.OverlayValues[303]
			}
			if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != scm.LocNone {
				d305 = ps.OverlayValues[305]
			}
			if len(ps.OverlayValues) > 306 && ps.OverlayValues[306].Loc != scm.LocNone {
				d306 = ps.OverlayValues[306]
			}
			if len(ps.OverlayValues) > 307 && ps.OverlayValues[307].Loc != scm.LocNone {
				d307 = ps.OverlayValues[307]
			}
			if len(ps.OverlayValues) > 308 && ps.OverlayValues[308].Loc != scm.LocNone {
				d308 = ps.OverlayValues[308]
			}
			if len(ps.OverlayValues) > 309 && ps.OverlayValues[309].Loc != scm.LocNone {
				d309 = ps.OverlayValues[309]
			}
			if len(ps.OverlayValues) > 312 && ps.OverlayValues[312].Loc != scm.LocNone {
				d312 = ps.OverlayValues[312]
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
			if len(ps.OverlayValues) > 672 && ps.OverlayValues[672].Loc != scm.LocNone {
				d672 = ps.OverlayValues[672]
			}
			if len(ps.OverlayValues) > 673 && ps.OverlayValues[673].Loc != scm.LocNone {
				d673 = ps.OverlayValues[673]
			}
			if len(ps.OverlayValues) > 674 && ps.OverlayValues[674].Loc != scm.LocNone {
				d674 = ps.OverlayValues[674]
			}
			if len(ps.OverlayValues) > 675 && ps.OverlayValues[675].Loc != scm.LocNone {
				d675 = ps.OverlayValues[675]
			}
			if len(ps.OverlayValues) > 676 && ps.OverlayValues[676].Loc != scm.LocNone {
				d676 = ps.OverlayValues[676]
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
			if len(ps.OverlayValues) > 681 && ps.OverlayValues[681].Loc != scm.LocNone {
				d681 = ps.OverlayValues[681]
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
			if len(ps.OverlayValues) > 687 && ps.OverlayValues[687].Loc != scm.LocNone {
				d687 = ps.OverlayValues[687]
			}
			if len(ps.OverlayValues) > 689 && ps.OverlayValues[689].Loc != scm.LocNone {
				d689 = ps.OverlayValues[689]
			}
			if len(ps.OverlayValues) > 690 && ps.OverlayValues[690].Loc != scm.LocNone {
				d690 = ps.OverlayValues[690]
			}
			if len(ps.OverlayValues) > 691 && ps.OverlayValues[691].Loc != scm.LocNone {
				d691 = ps.OverlayValues[691]
			}
			if len(ps.OverlayValues) > 692 && ps.OverlayValues[692].Loc != scm.LocNone {
				d692 = ps.OverlayValues[692]
			}
			if len(ps.OverlayValues) > 695 && ps.OverlayValues[695].Loc != scm.LocNone {
				d695 = ps.OverlayValues[695]
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
			if len(ps.OverlayValues) > 908 && ps.OverlayValues[908].Loc != scm.LocNone {
				d908 = ps.OverlayValues[908]
			}
			if len(ps.OverlayValues) > 909 && ps.OverlayValues[909].Loc != scm.LocNone {
				d909 = ps.OverlayValues[909]
			}
			if len(ps.OverlayValues) > 910 && ps.OverlayValues[910].Loc != scm.LocNone {
				d910 = ps.OverlayValues[910]
			}
			ctx.ReclaimUntrackedRegs()
			d911 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d911)
			ctx.BindReg(r2, &d911)
			ctx.EmitMakeNil(d911)
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[12].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[12].VisitCount >= 2 {
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
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
				d14 = ps.OverlayValues[14]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
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
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != scm.LocNone {
				d283 = ps.OverlayValues[283]
			}
			if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != scm.LocNone {
				d284 = ps.OverlayValues[284]
			}
			if len(ps.OverlayValues) > 285 && ps.OverlayValues[285].Loc != scm.LocNone {
				d285 = ps.OverlayValues[285]
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
			if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != scm.LocNone {
				d292 = ps.OverlayValues[292]
			}
			if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != scm.LocNone {
				d293 = ps.OverlayValues[293]
			}
			if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
				d294 = ps.OverlayValues[294]
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
			if len(ps.OverlayValues) > 303 && ps.OverlayValues[303].Loc != scm.LocNone {
				d303 = ps.OverlayValues[303]
			}
			if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != scm.LocNone {
				d305 = ps.OverlayValues[305]
			}
			if len(ps.OverlayValues) > 306 && ps.OverlayValues[306].Loc != scm.LocNone {
				d306 = ps.OverlayValues[306]
			}
			if len(ps.OverlayValues) > 307 && ps.OverlayValues[307].Loc != scm.LocNone {
				d307 = ps.OverlayValues[307]
			}
			if len(ps.OverlayValues) > 308 && ps.OverlayValues[308].Loc != scm.LocNone {
				d308 = ps.OverlayValues[308]
			}
			if len(ps.OverlayValues) > 309 && ps.OverlayValues[309].Loc != scm.LocNone {
				d309 = ps.OverlayValues[309]
			}
			if len(ps.OverlayValues) > 312 && ps.OverlayValues[312].Loc != scm.LocNone {
				d312 = ps.OverlayValues[312]
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
			if len(ps.OverlayValues) > 672 && ps.OverlayValues[672].Loc != scm.LocNone {
				d672 = ps.OverlayValues[672]
			}
			if len(ps.OverlayValues) > 673 && ps.OverlayValues[673].Loc != scm.LocNone {
				d673 = ps.OverlayValues[673]
			}
			if len(ps.OverlayValues) > 674 && ps.OverlayValues[674].Loc != scm.LocNone {
				d674 = ps.OverlayValues[674]
			}
			if len(ps.OverlayValues) > 675 && ps.OverlayValues[675].Loc != scm.LocNone {
				d675 = ps.OverlayValues[675]
			}
			if len(ps.OverlayValues) > 676 && ps.OverlayValues[676].Loc != scm.LocNone {
				d676 = ps.OverlayValues[676]
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
			if len(ps.OverlayValues) > 681 && ps.OverlayValues[681].Loc != scm.LocNone {
				d681 = ps.OverlayValues[681]
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
			if len(ps.OverlayValues) > 687 && ps.OverlayValues[687].Loc != scm.LocNone {
				d687 = ps.OverlayValues[687]
			}
			if len(ps.OverlayValues) > 689 && ps.OverlayValues[689].Loc != scm.LocNone {
				d689 = ps.OverlayValues[689]
			}
			if len(ps.OverlayValues) > 690 && ps.OverlayValues[690].Loc != scm.LocNone {
				d690 = ps.OverlayValues[690]
			}
			if len(ps.OverlayValues) > 691 && ps.OverlayValues[691].Loc != scm.LocNone {
				d691 = ps.OverlayValues[691]
			}
			if len(ps.OverlayValues) > 692 && ps.OverlayValues[692].Loc != scm.LocNone {
				d692 = ps.OverlayValues[692]
			}
			if len(ps.OverlayValues) > 695 && ps.OverlayValues[695].Loc != scm.LocNone {
				d695 = ps.OverlayValues[695]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d3)
			d912 = d3
			_ = d912
			r157 := d3.Loc == scm.LocReg
			r158 := d3.Reg
			if r157 { ctx.ProtectReg(r158) }
			d913 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(192)}
			lbl40 := ctx.ReserveLabel()
			bbpos_4_0 := int32(-1)
			_ = bbpos_4_0
			bbpos_4_1 := int32(-1)
			_ = bbpos_4_1
			bbpos_4_2 := int32(-1)
			_ = bbpos_4_2
			bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d913 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(192)}
			ctx.EnsureDesc(&d912)
			ctx.EnsureDesc(&d912)
			var d914 scm.JITValueDesc
			if d912.Loc == scm.LocImm {
				d914 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d912.Imm.Int()))))}
			} else {
				r159 := ctx.AllocReg()
				ctx.EmitMovRegReg(r159, d912.Reg)
				ctx.EmitShlRegImm8(r159, 32)
				ctx.EmitShrRegImm8(r159, 32)
				d914 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
				ctx.BindReg(r159, &d914)
			}
			var d915 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				r160 := ctx.AllocReg()
				ctx.EmitMovRegMem8(r160, fieldAddr)
				d915 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r160}
				ctx.BindReg(r160, &d915)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r161 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r161, thisptr.Reg, off)
				d915 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r161}
				ctx.BindReg(r161, &d915)
			}
			ctx.EnsureDesc(&d915)
			ctx.EnsureDesc(&d915)
			var d916 scm.JITValueDesc
			if d915.Loc == scm.LocImm {
				d916 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d915.Imm.Int()))))}
			} else {
				r162 := ctx.AllocReg()
				ctx.EmitMovRegReg(r162, d915.Reg)
				ctx.EmitShlRegImm8(r162, 56)
				ctx.EmitShrRegImm8(r162, 56)
				d916 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r162}
				ctx.BindReg(r162, &d916)
			}
			ctx.EnsureDesc(&d914)
			ctx.EnsureDesc(&d916)
			ctx.EnsureDesc(&d914)
			ctx.EnsureDesc(&d916)
			ctx.EnsureDesc(&d914)
			ctx.EnsureDesc(&d916)
			var d917 scm.JITValueDesc
			if d914.Loc == scm.LocImm && d916.Loc == scm.LocImm {
				d917 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d914.Imm.Int() * d916.Imm.Int())}
			} else if d914.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d916.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d914.Imm.Int()))
				ctx.EmitImulInt64(scratch, d916.Reg)
				d917 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d917)
			} else if d916.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d914.Reg)
				ctx.EmitMovRegReg(scratch, d914.Reg)
				if d916.Imm.Int() >= -2147483648 && d916.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d916.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d916.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d917 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d917)
			} else {
				r163 := ctx.AllocRegExcept(d914.Reg, d916.Reg)
				ctx.EmitMovRegReg(r163, d914.Reg)
				ctx.EmitImulInt64(r163, d916.Reg)
				d917 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r163}
				ctx.BindReg(r163, &d917)
			}
			if d917.Loc == scm.LocReg && d914.Loc == scm.LocReg && d917.Reg == d914.Reg {
				ctx.TransferReg(d914.Reg)
				d914.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d914)
			ctx.FreeDesc(&d916)
			var d918 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 0
				r164 := ctx.AllocReg()
				r165 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r164, fieldAddr)
				ctx.EmitMovRegMem64(r165, fieldAddr+8)
				d918 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r164, Reg2: r165}
				ctx.BindReg(r164, &d918)
				ctx.BindReg(r165, &d918)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 0)
				r166 := ctx.AllocReg()
				r167 := ctx.AllocReg()
				ctx.EmitMovRegMem(r166, thisptr.Reg, off)
				ctx.EmitMovRegMem(r167, thisptr.Reg, off+8)
				d918 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r166, Reg2: r167}
				ctx.BindReg(r166, &d918)
				ctx.BindReg(r167, &d918)
			}
			ctx.EnsureDesc(&d917)
			var d919 scm.JITValueDesc
			if d917.Loc == scm.LocImm {
				d919 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d917.Imm.Int() / 64)}
			} else {
				r168 := ctx.AllocRegExcept(d917.Reg)
				ctx.EmitMovRegReg(r168, d917.Reg)
				ctx.EmitShrRegImm8(r168, 6)
				d919 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r168}
				ctx.BindReg(r168, &d919)
			}
			if d919.Loc == scm.LocReg && d917.Loc == scm.LocReg && d919.Reg == d917.Reg {
				ctx.TransferReg(d917.Reg)
				d917.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d919)
			r169 := ctx.AllocReg()
			ctx.EnsureDesc(&d919)
			ctx.EnsureDesc(&d918)
			if d919.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r169, uint64(d919.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r169, d919.Reg)
				ctx.EmitShlRegImm8(r169, 3)
			}
			if d918.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d918.Imm.Int()))
				ctx.EmitAddInt64(r169, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r169, d918.Reg)
			}
			r170 := ctx.AllocRegExcept(r169)
			ctx.EmitMovRegMem(r170, r169, 0)
			ctx.FreeReg(r169)
			d920 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r170}
			ctx.BindReg(r170, &d920)
			ctx.FreeDesc(&d919)
			ctx.EnsureDesc(&d917)
			var d921 scm.JITValueDesc
			if d917.Loc == scm.LocImm {
				d921 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d917.Imm.Int() % 64)}
			} else {
				r171 := ctx.AllocRegExcept(d917.Reg)
				ctx.EmitMovRegReg(r171, d917.Reg)
				ctx.EmitAndRegImm32(r171, 63)
				d921 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r171}
				ctx.BindReg(r171, &d921)
			}
			if d921.Loc == scm.LocReg && d917.Loc == scm.LocReg && d921.Reg == d917.Reg {
				ctx.TransferReg(d917.Reg)
				d917.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d920)
			ctx.EnsureDesc(&d921)
			var d922 scm.JITValueDesc
			if d920.Loc == scm.LocImm && d921.Loc == scm.LocImm {
				d922 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d920.Imm.Int()) << uint64(d921.Imm.Int())))}
			} else if d921.Loc == scm.LocImm {
				r172 := ctx.AllocRegExcept(d920.Reg)
				ctx.EmitMovRegReg(r172, d920.Reg)
				ctx.EmitShlRegImm8(r172, uint8(d921.Imm.Int()))
				d922 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
				ctx.BindReg(r172, &d922)
			} else {
				{
					shiftSrc := d920.Reg
					r173 := ctx.AllocRegExcept(d920.Reg)
					ctx.EmitMovRegReg(r173, d920.Reg)
					shiftSrc = r173
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d921.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d921.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d921.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d922 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d922)
				}
			}
			if d922.Loc == scm.LocReg && d920.Loc == scm.LocReg && d922.Reg == d920.Reg {
				ctx.TransferReg(d920.Reg)
				d920.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d920)
			ctx.FreeDesc(&d921)
			ctx.EnsureDesc(&d917)
			var d923 scm.JITValueDesc
			if d917.Loc == scm.LocImm {
				d923 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d917.Imm.Int() % 64)}
			} else {
				r174 := ctx.AllocRegExcept(d917.Reg)
				ctx.EmitMovRegReg(r174, d917.Reg)
				ctx.EmitAndRegImm32(r174, 63)
				d923 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r174}
				ctx.BindReg(r174, &d923)
			}
			if d923.Loc == scm.LocReg && d917.Loc == scm.LocReg && d923.Reg == d917.Reg {
				ctx.TransferReg(d917.Reg)
				d917.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d915)
			ctx.EnsureDesc(&d915)
			var d924 scm.JITValueDesc
			if d915.Loc == scm.LocImm {
				d924 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d915.Imm.Int()))))}
			} else {
				r175 := ctx.AllocReg()
				ctx.EmitMovRegReg(r175, d915.Reg)
				ctx.EmitShlRegImm8(r175, 56)
				ctx.EmitShrRegImm8(r175, 56)
				d924 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r175}
				ctx.BindReg(r175, &d924)
			}
			ctx.EnsureDesc(&d923)
			ctx.EnsureDesc(&d924)
			ctx.EnsureDesc(&d923)
			ctx.EnsureDesc(&d924)
			ctx.EnsureDesc(&d923)
			ctx.EnsureDesc(&d924)
			var d925 scm.JITValueDesc
			if d923.Loc == scm.LocImm && d924.Loc == scm.LocImm {
				d925 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d923.Imm.Int() + d924.Imm.Int())}
			} else if d924.Loc == scm.LocImm && d924.Imm.Int() == 0 {
				r176 := ctx.AllocRegExcept(d923.Reg)
				ctx.EmitMovRegReg(r176, d923.Reg)
				d925 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r176}
				ctx.BindReg(r176, &d925)
			} else if d923.Loc == scm.LocImm && d923.Imm.Int() == 0 {
				d925 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d924.Reg}
				ctx.BindReg(d924.Reg, &d925)
			} else if d923.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d924.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d923.Imm.Int()))
				ctx.EmitAddInt64(scratch, d924.Reg)
				d925 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d925)
			} else if d924.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d923.Reg)
				ctx.EmitMovRegReg(scratch, d923.Reg)
				if d924.Imm.Int() >= -2147483648 && d924.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d924.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d924.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d925 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d925)
			} else {
				r177 := ctx.AllocRegExcept(d923.Reg, d924.Reg)
				ctx.EmitMovRegReg(r177, d923.Reg)
				ctx.EmitAddInt64(r177, d924.Reg)
				d925 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r177}
				ctx.BindReg(r177, &d925)
			}
			if d925.Loc == scm.LocReg && d923.Loc == scm.LocReg && d925.Reg == d923.Reg {
				ctx.TransferReg(d923.Reg)
				d923.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d923)
			ctx.FreeDesc(&d924)
			ctx.EnsureDesc(&d925)
			var d926 scm.JITValueDesc
			if d925.Loc == scm.LocImm {
				d926 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d925.Imm.Int()) > uint64(64))}
			} else {
				r178 := ctx.AllocRegExcept(d925.Reg)
				ctx.EmitCmpRegImm32(d925.Reg, 64)
				ctx.EmitSetcc(r178, scm.CcA)
				d926 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r178}
				ctx.BindReg(r178, &d926)
			}
			ctx.FreeDesc(&d925)
			d927 = d926
			ctx.EnsureDesc(&d927)
			if d927.Loc != scm.LocImm && d927.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl41 := ctx.ReserveLabel()
			lbl42 := ctx.ReserveLabel()
			lbl43 := ctx.ReserveLabel()
			lbl44 := ctx.ReserveLabel()
			if d927.Loc == scm.LocImm {
				if d927.Imm.Bool() {
					ctx.MarkLabel(lbl43)
					ctx.EmitJmp(lbl41)
				} else {
					ctx.MarkLabel(lbl44)
			ctx.EnsureDesc(&d922)
			if d922.Loc == scm.LocReg {
				ctx.ProtectReg(d922.Reg)
			} else if d922.Loc == scm.LocRegPair {
				ctx.ProtectReg(d922.Reg)
				ctx.ProtectReg(d922.Reg2)
			}
			d928 = d922
			if d928.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d928)
			ctx.EmitStoreToStack(d928, int32(bbs[2].PhiBase)+int32(0))
			if d922.Loc == scm.LocReg {
				ctx.UnprotectReg(d922.Reg)
			} else if d922.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d922.Reg)
				ctx.UnprotectReg(d922.Reg2)
			}
					ctx.EmitJmp(lbl42)
				}
			} else {
				ctx.EmitCmpRegImm32(d927.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl43)
				ctx.EmitJmp(lbl44)
				ctx.MarkLabel(lbl43)
				ctx.EmitJmp(lbl41)
				ctx.MarkLabel(lbl44)
			ctx.EnsureDesc(&d922)
			if d922.Loc == scm.LocReg {
				ctx.ProtectReg(d922.Reg)
			} else if d922.Loc == scm.LocRegPair {
				ctx.ProtectReg(d922.Reg)
				ctx.ProtectReg(d922.Reg2)
			}
			d929 = d922
			if d929.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d929)
			ctx.EmitStoreToStack(d929, int32(bbs[2].PhiBase)+int32(0))
			if d922.Loc == scm.LocReg {
				ctx.UnprotectReg(d922.Reg)
			} else if d922.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d922.Reg)
				ctx.UnprotectReg(d922.Reg2)
			}
				ctx.EmitJmp(lbl42)
			}
			ctx.FreeDesc(&d926)
			bbpos_4_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl42)
			ctx.ResolveFixups()
			d913 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(192)}
			ctx.EnsureDesc(&d915)
			ctx.EnsureDesc(&d915)
			var d930 scm.JITValueDesc
			if d915.Loc == scm.LocImm {
				d930 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d915.Imm.Int()))))}
			} else {
				r179 := ctx.AllocReg()
				ctx.EmitMovRegReg(r179, d915.Reg)
				ctx.EmitShlRegImm8(r179, 56)
				ctx.EmitShrRegImm8(r179, 56)
				d930 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r179}
				ctx.BindReg(r179, &d930)
			}
			d931 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d930)
			ctx.EnsureDesc(&d931)
			ctx.EnsureDesc(&d930)
			ctx.EnsureDesc(&d931)
			ctx.EnsureDesc(&d930)
			var d932 scm.JITValueDesc
			if d931.Loc == scm.LocImm && d930.Loc == scm.LocImm {
				d932 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d931.Imm.Int() - d930.Imm.Int())}
			} else if d930.Loc == scm.LocImm && d930.Imm.Int() == 0 {
				r180 := ctx.AllocRegExcept(d931.Reg)
				ctx.EmitMovRegReg(r180, d931.Reg)
				d932 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r180}
				ctx.BindReg(r180, &d932)
			} else if d931.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d930.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d931.Imm.Int()))
				ctx.EmitSubInt64(scratch, d930.Reg)
				d932 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d932)
			} else if d930.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d931.Reg)
				ctx.EmitMovRegReg(scratch, d931.Reg)
				if d930.Imm.Int() >= -2147483648 && d930.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d930.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d930.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d932 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d932)
			} else {
				r181 := ctx.AllocRegExcept(d931.Reg, d930.Reg)
				ctx.EmitMovRegReg(r181, d931.Reg)
				ctx.EmitSubInt64(r181, d930.Reg)
				d932 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r181}
				ctx.BindReg(r181, &d932)
			}
			if d932.Loc == scm.LocReg && d931.Loc == scm.LocReg && d932.Reg == d931.Reg {
				ctx.TransferReg(d931.Reg)
				d931.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d930)
			ctx.EnsureDesc(&d913)
			ctx.EnsureDesc(&d932)
			var d933 scm.JITValueDesc
			if d913.Loc == scm.LocImm && d932.Loc == scm.LocImm {
				d933 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d913.Imm.Int()) >> uint64(d932.Imm.Int())))}
			} else if d932.Loc == scm.LocImm {
				r182 := ctx.AllocRegExcept(d913.Reg)
				ctx.EmitMovRegReg(r182, d913.Reg)
				ctx.EmitShrRegImm8(r182, uint8(d932.Imm.Int()))
				d933 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r182}
				ctx.BindReg(r182, &d933)
			} else {
				{
					shiftSrc := d913.Reg
					r183 := ctx.AllocRegExcept(d913.Reg)
					ctx.EmitMovRegReg(r183, d913.Reg)
					shiftSrc = r183
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d932.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d932.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d932.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d933 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d933)
				}
			}
			if d933.Loc == scm.LocReg && d913.Loc == scm.LocReg && d933.Reg == d913.Reg {
				ctx.TransferReg(d913.Reg)
				d913.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d913)
			ctx.FreeDesc(&d932)
			r184 := ctx.AllocReg()
			ctx.EnsureDesc(&d933)
			ctx.EnsureDesc(&d933)
			if d933.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r184, d933)
			}
			ctx.EmitJmp(lbl40)
			bbpos_4_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl41)
			ctx.ResolveFixups()
			d913 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(192)}
			ctx.EnsureDesc(&d917)
			var d934 scm.JITValueDesc
			if d917.Loc == scm.LocImm {
				d934 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d917.Imm.Int() / 64)}
			} else {
				r185 := ctx.AllocRegExcept(d917.Reg)
				ctx.EmitMovRegReg(r185, d917.Reg)
				ctx.EmitShrRegImm8(r185, 6)
				d934 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r185}
				ctx.BindReg(r185, &d934)
			}
			if d934.Loc == scm.LocReg && d917.Loc == scm.LocReg && d934.Reg == d917.Reg {
				ctx.TransferReg(d917.Reg)
				d917.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d934)
			ctx.EnsureDesc(&d934)
			var d935 scm.JITValueDesc
			if d934.Loc == scm.LocImm {
				d935 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d934.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d934.Reg)
				ctx.EmitMovRegReg(scratch, d934.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d935 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d935)
			}
			if d935.Loc == scm.LocReg && d934.Loc == scm.LocReg && d935.Reg == d934.Reg {
				ctx.TransferReg(d934.Reg)
				d934.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d934)
			ctx.EnsureDesc(&d935)
			r186 := ctx.AllocReg()
			ctx.EnsureDesc(&d935)
			ctx.EnsureDesc(&d918)
			if d935.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r186, uint64(d935.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r186, d935.Reg)
				ctx.EmitShlRegImm8(r186, 3)
			}
			if d918.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d918.Imm.Int()))
				ctx.EmitAddInt64(r186, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r186, d918.Reg)
			}
			r187 := ctx.AllocRegExcept(r186)
			ctx.EmitMovRegMem(r187, r186, 0)
			ctx.FreeReg(r186)
			d936 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r187}
			ctx.BindReg(r187, &d936)
			ctx.FreeDesc(&d935)
			ctx.EnsureDesc(&d917)
			var d937 scm.JITValueDesc
			if d917.Loc == scm.LocImm {
				d937 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d917.Imm.Int() % 64)}
			} else {
				r188 := ctx.AllocRegExcept(d917.Reg)
				ctx.EmitMovRegReg(r188, d917.Reg)
				ctx.EmitAndRegImm32(r188, 63)
				d937 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r188}
				ctx.BindReg(r188, &d937)
			}
			if d937.Loc == scm.LocReg && d917.Loc == scm.LocReg && d937.Reg == d917.Reg {
				ctx.TransferReg(d917.Reg)
				d917.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d917)
			d938 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d937)
			ctx.EnsureDesc(&d938)
			ctx.EnsureDesc(&d937)
			ctx.EnsureDesc(&d938)
			ctx.EnsureDesc(&d937)
			var d939 scm.JITValueDesc
			if d938.Loc == scm.LocImm && d937.Loc == scm.LocImm {
				d939 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d938.Imm.Int() - d937.Imm.Int())}
			} else if d937.Loc == scm.LocImm && d937.Imm.Int() == 0 {
				r189 := ctx.AllocRegExcept(d938.Reg)
				ctx.EmitMovRegReg(r189, d938.Reg)
				d939 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r189}
				ctx.BindReg(r189, &d939)
			} else if d938.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d937.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d938.Imm.Int()))
				ctx.EmitSubInt64(scratch, d937.Reg)
				d939 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d939)
			} else if d937.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d938.Reg)
				ctx.EmitMovRegReg(scratch, d938.Reg)
				if d937.Imm.Int() >= -2147483648 && d937.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d937.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d937.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d939 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d939)
			} else {
				r190 := ctx.AllocRegExcept(d938.Reg, d937.Reg)
				ctx.EmitMovRegReg(r190, d938.Reg)
				ctx.EmitSubInt64(r190, d937.Reg)
				d939 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r190}
				ctx.BindReg(r190, &d939)
			}
			if d939.Loc == scm.LocReg && d938.Loc == scm.LocReg && d939.Reg == d938.Reg {
				ctx.TransferReg(d938.Reg)
				d938.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d937)
			ctx.EnsureDesc(&d936)
			ctx.EnsureDesc(&d939)
			var d940 scm.JITValueDesc
			if d936.Loc == scm.LocImm && d939.Loc == scm.LocImm {
				d940 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d936.Imm.Int()) >> uint64(d939.Imm.Int())))}
			} else if d939.Loc == scm.LocImm {
				r191 := ctx.AllocRegExcept(d936.Reg)
				ctx.EmitMovRegReg(r191, d936.Reg)
				ctx.EmitShrRegImm8(r191, uint8(d939.Imm.Int()))
				d940 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r191}
				ctx.BindReg(r191, &d940)
			} else {
				{
					shiftSrc := d936.Reg
					r192 := ctx.AllocRegExcept(d936.Reg)
					ctx.EmitMovRegReg(r192, d936.Reg)
					shiftSrc = r192
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d939.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d939.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d939.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d940 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d940)
				}
			}
			if d940.Loc == scm.LocReg && d936.Loc == scm.LocReg && d940.Reg == d936.Reg {
				ctx.TransferReg(d936.Reg)
				d936.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d936)
			ctx.FreeDesc(&d939)
			ctx.EnsureDesc(&d922)
			ctx.EnsureDesc(&d940)
			var d941 scm.JITValueDesc
			if d922.Loc == scm.LocImm && d940.Loc == scm.LocImm {
				d941 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d922.Imm.Int() | d940.Imm.Int())}
			} else if d922.Loc == scm.LocImm && d922.Imm.Int() == 0 {
				d941 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d940.Reg}
				ctx.BindReg(d940.Reg, &d941)
			} else if d940.Loc == scm.LocImm && d940.Imm.Int() == 0 {
				r193 := ctx.AllocRegExcept(d922.Reg)
				ctx.EmitMovRegReg(r193, d922.Reg)
				d941 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r193}
				ctx.BindReg(r193, &d941)
			} else if d922.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d940.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d922.Imm.Int()))
				ctx.EmitOrInt64(scratch, d940.Reg)
				d941 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d941)
			} else if d940.Loc == scm.LocImm {
				r194 := ctx.AllocRegExcept(d922.Reg)
				ctx.EmitMovRegReg(r194, d922.Reg)
				if d940.Imm.Int() >= -2147483648 && d940.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r194, int32(d940.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d940.Imm.Int()))
					ctx.EmitOrInt64(r194, scm.RegR11)
				}
				d941 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r194}
				ctx.BindReg(r194, &d941)
			} else {
				r195 := ctx.AllocRegExcept(d922.Reg, d940.Reg)
				ctx.EmitMovRegReg(r195, d922.Reg)
				ctx.EmitOrInt64(r195, d940.Reg)
				d941 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r195}
				ctx.BindReg(r195, &d941)
			}
			if d941.Loc == scm.LocReg && d922.Loc == scm.LocReg && d941.Reg == d922.Reg {
				ctx.TransferReg(d922.Reg)
				d922.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d940)
			ctx.EnsureDesc(&d941)
			if d941.Loc == scm.LocReg {
				ctx.ProtectReg(d941.Reg)
			} else if d941.Loc == scm.LocRegPair {
				ctx.ProtectReg(d941.Reg)
				ctx.ProtectReg(d941.Reg2)
			}
			d942 = d941
			if d942.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d942)
			ctx.EmitStoreToStack(d942, int32(bbs[2].PhiBase)+int32(0))
			if d941.Loc == scm.LocReg {
				ctx.UnprotectReg(d941.Reg)
			} else if d941.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d941.Reg)
				ctx.UnprotectReg(d941.Reg2)
			}
			ctx.EmitJmp(lbl42)
			ctx.MarkLabel(lbl40)
			d943 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r184}
			ctx.BindReg(r184, &d943)
			ctx.BindReg(r184, &d943)
			if r157 { ctx.UnprotectReg(r158) }
			ctx.EnsureDesc(&d943)
			ctx.EnsureDesc(&d943)
			var d944 scm.JITValueDesc
			if d943.Loc == scm.LocImm {
				d944 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d943.Imm.Int()))))}
			} else {
				r196 := ctx.AllocReg()
				ctx.EmitMovRegReg(r196, d943.Reg)
				d944 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r196}
				ctx.BindReg(r196, &d944)
			}
			ctx.FreeDesc(&d943)
			var d945 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 32
				r197 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r197, fieldAddr)
				d945 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r197}
				ctx.BindReg(r197, &d945)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 32)
				r198 := ctx.AllocReg()
				ctx.EmitMovRegMem(r198, thisptr.Reg, off)
				d945 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r198}
				ctx.BindReg(r198, &d945)
			}
			ctx.EnsureDesc(&d944)
			ctx.EnsureDesc(&d945)
			ctx.EnsureDesc(&d944)
			ctx.EnsureDesc(&d945)
			ctx.EnsureDesc(&d944)
			ctx.EnsureDesc(&d945)
			var d946 scm.JITValueDesc
			if d944.Loc == scm.LocImm && d945.Loc == scm.LocImm {
				d946 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d944.Imm.Int() + d945.Imm.Int())}
			} else if d945.Loc == scm.LocImm && d945.Imm.Int() == 0 {
				r199 := ctx.AllocRegExcept(d944.Reg)
				ctx.EmitMovRegReg(r199, d944.Reg)
				d946 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r199}
				ctx.BindReg(r199, &d946)
			} else if d944.Loc == scm.LocImm && d944.Imm.Int() == 0 {
				d946 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d945.Reg}
				ctx.BindReg(d945.Reg, &d946)
			} else if d944.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d945.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d944.Imm.Int()))
				ctx.EmitAddInt64(scratch, d945.Reg)
				d946 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d946)
			} else if d945.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d944.Reg)
				ctx.EmitMovRegReg(scratch, d944.Reg)
				if d945.Imm.Int() >= -2147483648 && d945.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d945.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d945.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d946 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d946)
			} else {
				r200 := ctx.AllocRegExcept(d944.Reg, d945.Reg)
				ctx.EmitMovRegReg(r200, d944.Reg)
				ctx.EmitAddInt64(r200, d945.Reg)
				d946 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r200}
				ctx.BindReg(r200, &d946)
			}
			if d946.Loc == scm.LocReg && d944.Loc == scm.LocReg && d946.Reg == d944.Reg {
				ctx.TransferReg(d944.Reg)
				d944.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d944)
			ctx.EnsureDesc(&d3)
			d947 = d3
			_ = d947
			r201 := d3.Loc == scm.LocReg
			r202 := d3.Reg
			if r201 { ctx.ProtectReg(r202) }
			d948 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(208)}
			lbl45 := ctx.ReserveLabel()
			bbpos_5_0 := int32(-1)
			_ = bbpos_5_0
			bbpos_5_1 := int32(-1)
			_ = bbpos_5_1
			bbpos_5_2 := int32(-1)
			_ = bbpos_5_2
			bbpos_5_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d948 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(208)}
			ctx.EnsureDesc(&d947)
			ctx.EnsureDesc(&d947)
			var d949 scm.JITValueDesc
			if d947.Loc == scm.LocImm {
				d949 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d947.Imm.Int()))))}
			} else {
				r203 := ctx.AllocReg()
				ctx.EmitMovRegReg(r203, d947.Reg)
				ctx.EmitShlRegImm8(r203, 32)
				ctx.EmitShrRegImm8(r203, 32)
				d949 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r203}
				ctx.BindReg(r203, &d949)
			}
			var d950 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				r204 := ctx.AllocReg()
				ctx.EmitMovRegMem8(r204, fieldAddr)
				d950 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r204}
				ctx.BindReg(r204, &d950)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r205 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r205, thisptr.Reg, off)
				d950 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r205}
				ctx.BindReg(r205, &d950)
			}
			ctx.EnsureDesc(&d950)
			ctx.EnsureDesc(&d950)
			var d951 scm.JITValueDesc
			if d950.Loc == scm.LocImm {
				d951 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d950.Imm.Int()))))}
			} else {
				r206 := ctx.AllocReg()
				ctx.EmitMovRegReg(r206, d950.Reg)
				ctx.EmitShlRegImm8(r206, 56)
				ctx.EmitShrRegImm8(r206, 56)
				d951 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r206}
				ctx.BindReg(r206, &d951)
			}
			ctx.EnsureDesc(&d949)
			ctx.EnsureDesc(&d951)
			ctx.EnsureDesc(&d949)
			ctx.EnsureDesc(&d951)
			ctx.EnsureDesc(&d949)
			ctx.EnsureDesc(&d951)
			var d952 scm.JITValueDesc
			if d949.Loc == scm.LocImm && d951.Loc == scm.LocImm {
				d952 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d949.Imm.Int() * d951.Imm.Int())}
			} else if d949.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d951.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d949.Imm.Int()))
				ctx.EmitImulInt64(scratch, d951.Reg)
				d952 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d952)
			} else if d951.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d949.Reg)
				ctx.EmitMovRegReg(scratch, d949.Reg)
				if d951.Imm.Int() >= -2147483648 && d951.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d951.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d951.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d952 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d952)
			} else {
				r207 := ctx.AllocRegExcept(d949.Reg, d951.Reg)
				ctx.EmitMovRegReg(r207, d949.Reg)
				ctx.EmitImulInt64(r207, d951.Reg)
				d952 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r207}
				ctx.BindReg(r207, &d952)
			}
			if d952.Loc == scm.LocReg && d949.Loc == scm.LocReg && d952.Reg == d949.Reg {
				ctx.TransferReg(d949.Reg)
				d949.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d949)
			ctx.FreeDesc(&d951)
			var d953 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				r208 := ctx.AllocReg()
				r209 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r208, fieldAddr)
				ctx.EmitMovRegMem64(r209, fieldAddr+8)
				d953 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r208, Reg2: r209}
				ctx.BindReg(r208, &d953)
				ctx.BindReg(r209, &d953)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				r210 := ctx.AllocReg()
				r211 := ctx.AllocReg()
				ctx.EmitMovRegMem(r210, thisptr.Reg, off)
				ctx.EmitMovRegMem(r211, thisptr.Reg, off+8)
				d953 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r210, Reg2: r211}
				ctx.BindReg(r210, &d953)
				ctx.BindReg(r211, &d953)
			}
			ctx.EnsureDesc(&d952)
			var d954 scm.JITValueDesc
			if d952.Loc == scm.LocImm {
				d954 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d952.Imm.Int() / 64)}
			} else {
				r212 := ctx.AllocRegExcept(d952.Reg)
				ctx.EmitMovRegReg(r212, d952.Reg)
				ctx.EmitShrRegImm8(r212, 6)
				d954 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r212}
				ctx.BindReg(r212, &d954)
			}
			if d954.Loc == scm.LocReg && d952.Loc == scm.LocReg && d954.Reg == d952.Reg {
				ctx.TransferReg(d952.Reg)
				d952.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d954)
			r213 := ctx.AllocReg()
			ctx.EnsureDesc(&d954)
			ctx.EnsureDesc(&d953)
			if d954.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r213, uint64(d954.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r213, d954.Reg)
				ctx.EmitShlRegImm8(r213, 3)
			}
			if d953.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d953.Imm.Int()))
				ctx.EmitAddInt64(r213, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r213, d953.Reg)
			}
			r214 := ctx.AllocRegExcept(r213)
			ctx.EmitMovRegMem(r214, r213, 0)
			ctx.FreeReg(r213)
			d955 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r214}
			ctx.BindReg(r214, &d955)
			ctx.FreeDesc(&d954)
			ctx.EnsureDesc(&d952)
			var d956 scm.JITValueDesc
			if d952.Loc == scm.LocImm {
				d956 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d952.Imm.Int() % 64)}
			} else {
				r215 := ctx.AllocRegExcept(d952.Reg)
				ctx.EmitMovRegReg(r215, d952.Reg)
				ctx.EmitAndRegImm32(r215, 63)
				d956 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r215}
				ctx.BindReg(r215, &d956)
			}
			if d956.Loc == scm.LocReg && d952.Loc == scm.LocReg && d956.Reg == d952.Reg {
				ctx.TransferReg(d952.Reg)
				d952.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d955)
			ctx.EnsureDesc(&d956)
			var d957 scm.JITValueDesc
			if d955.Loc == scm.LocImm && d956.Loc == scm.LocImm {
				d957 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d955.Imm.Int()) << uint64(d956.Imm.Int())))}
			} else if d956.Loc == scm.LocImm {
				r216 := ctx.AllocRegExcept(d955.Reg)
				ctx.EmitMovRegReg(r216, d955.Reg)
				ctx.EmitShlRegImm8(r216, uint8(d956.Imm.Int()))
				d957 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r216}
				ctx.BindReg(r216, &d957)
			} else {
				{
					shiftSrc := d955.Reg
					r217 := ctx.AllocRegExcept(d955.Reg)
					ctx.EmitMovRegReg(r217, d955.Reg)
					shiftSrc = r217
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d956.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d956.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d956.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d957 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d957)
				}
			}
			if d957.Loc == scm.LocReg && d955.Loc == scm.LocReg && d957.Reg == d955.Reg {
				ctx.TransferReg(d955.Reg)
				d955.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d955)
			ctx.FreeDesc(&d956)
			ctx.EnsureDesc(&d952)
			var d958 scm.JITValueDesc
			if d952.Loc == scm.LocImm {
				d958 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d952.Imm.Int() % 64)}
			} else {
				r218 := ctx.AllocRegExcept(d952.Reg)
				ctx.EmitMovRegReg(r218, d952.Reg)
				ctx.EmitAndRegImm32(r218, 63)
				d958 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r218}
				ctx.BindReg(r218, &d958)
			}
			if d958.Loc == scm.LocReg && d952.Loc == scm.LocReg && d958.Reg == d952.Reg {
				ctx.TransferReg(d952.Reg)
				d952.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d950)
			ctx.EnsureDesc(&d950)
			var d959 scm.JITValueDesc
			if d950.Loc == scm.LocImm {
				d959 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d950.Imm.Int()))))}
			} else {
				r219 := ctx.AllocReg()
				ctx.EmitMovRegReg(r219, d950.Reg)
				ctx.EmitShlRegImm8(r219, 56)
				ctx.EmitShrRegImm8(r219, 56)
				d959 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r219}
				ctx.BindReg(r219, &d959)
			}
			ctx.EnsureDesc(&d958)
			ctx.EnsureDesc(&d959)
			ctx.EnsureDesc(&d958)
			ctx.EnsureDesc(&d959)
			ctx.EnsureDesc(&d958)
			ctx.EnsureDesc(&d959)
			var d960 scm.JITValueDesc
			if d958.Loc == scm.LocImm && d959.Loc == scm.LocImm {
				d960 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d958.Imm.Int() + d959.Imm.Int())}
			} else if d959.Loc == scm.LocImm && d959.Imm.Int() == 0 {
				r220 := ctx.AllocRegExcept(d958.Reg)
				ctx.EmitMovRegReg(r220, d958.Reg)
				d960 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r220}
				ctx.BindReg(r220, &d960)
			} else if d958.Loc == scm.LocImm && d958.Imm.Int() == 0 {
				d960 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d959.Reg}
				ctx.BindReg(d959.Reg, &d960)
			} else if d958.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d959.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d958.Imm.Int()))
				ctx.EmitAddInt64(scratch, d959.Reg)
				d960 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d960)
			} else if d959.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d958.Reg)
				ctx.EmitMovRegReg(scratch, d958.Reg)
				if d959.Imm.Int() >= -2147483648 && d959.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d959.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d959.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d960 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d960)
			} else {
				r221 := ctx.AllocRegExcept(d958.Reg, d959.Reg)
				ctx.EmitMovRegReg(r221, d958.Reg)
				ctx.EmitAddInt64(r221, d959.Reg)
				d960 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r221}
				ctx.BindReg(r221, &d960)
			}
			if d960.Loc == scm.LocReg && d958.Loc == scm.LocReg && d960.Reg == d958.Reg {
				ctx.TransferReg(d958.Reg)
				d958.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d958)
			ctx.FreeDesc(&d959)
			ctx.EnsureDesc(&d960)
			var d961 scm.JITValueDesc
			if d960.Loc == scm.LocImm {
				d961 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d960.Imm.Int()) > uint64(64))}
			} else {
				r222 := ctx.AllocRegExcept(d960.Reg)
				ctx.EmitCmpRegImm32(d960.Reg, 64)
				ctx.EmitSetcc(r222, scm.CcA)
				d961 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r222}
				ctx.BindReg(r222, &d961)
			}
			ctx.FreeDesc(&d960)
			d962 = d961
			ctx.EnsureDesc(&d962)
			if d962.Loc != scm.LocImm && d962.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl46 := ctx.ReserveLabel()
			lbl47 := ctx.ReserveLabel()
			lbl48 := ctx.ReserveLabel()
			lbl49 := ctx.ReserveLabel()
			if d962.Loc == scm.LocImm {
				if d962.Imm.Bool() {
					ctx.MarkLabel(lbl48)
					ctx.EmitJmp(lbl46)
				} else {
					ctx.MarkLabel(lbl49)
			ctx.EnsureDesc(&d957)
			if d957.Loc == scm.LocReg {
				ctx.ProtectReg(d957.Reg)
			} else if d957.Loc == scm.LocRegPair {
				ctx.ProtectReg(d957.Reg)
				ctx.ProtectReg(d957.Reg2)
			}
			d963 = d957
			if d963.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d963)
			ctx.EmitStoreToStack(d963, int32(bbs[2].PhiBase)+int32(0))
			if d957.Loc == scm.LocReg {
				ctx.UnprotectReg(d957.Reg)
			} else if d957.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d957.Reg)
				ctx.UnprotectReg(d957.Reg2)
			}
					ctx.EmitJmp(lbl47)
				}
			} else {
				ctx.EmitCmpRegImm32(d962.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl48)
				ctx.EmitJmp(lbl49)
				ctx.MarkLabel(lbl48)
				ctx.EmitJmp(lbl46)
				ctx.MarkLabel(lbl49)
			ctx.EnsureDesc(&d957)
			if d957.Loc == scm.LocReg {
				ctx.ProtectReg(d957.Reg)
			} else if d957.Loc == scm.LocRegPair {
				ctx.ProtectReg(d957.Reg)
				ctx.ProtectReg(d957.Reg2)
			}
			d964 = d957
			if d964.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d964)
			ctx.EmitStoreToStack(d964, int32(bbs[2].PhiBase)+int32(0))
			if d957.Loc == scm.LocReg {
				ctx.UnprotectReg(d957.Reg)
			} else if d957.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d957.Reg)
				ctx.UnprotectReg(d957.Reg2)
			}
				ctx.EmitJmp(lbl47)
			}
			ctx.FreeDesc(&d961)
			bbpos_5_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl47)
			ctx.ResolveFixups()
			d948 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(208)}
			ctx.EnsureDesc(&d950)
			ctx.EnsureDesc(&d950)
			var d965 scm.JITValueDesc
			if d950.Loc == scm.LocImm {
				d965 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d950.Imm.Int()))))}
			} else {
				r223 := ctx.AllocReg()
				ctx.EmitMovRegReg(r223, d950.Reg)
				ctx.EmitShlRegImm8(r223, 56)
				ctx.EmitShrRegImm8(r223, 56)
				d965 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r223}
				ctx.BindReg(r223, &d965)
			}
			d966 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d965)
			ctx.EnsureDesc(&d966)
			ctx.EnsureDesc(&d965)
			ctx.EnsureDesc(&d966)
			ctx.EnsureDesc(&d965)
			var d967 scm.JITValueDesc
			if d966.Loc == scm.LocImm && d965.Loc == scm.LocImm {
				d967 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d966.Imm.Int() - d965.Imm.Int())}
			} else if d965.Loc == scm.LocImm && d965.Imm.Int() == 0 {
				r224 := ctx.AllocRegExcept(d966.Reg)
				ctx.EmitMovRegReg(r224, d966.Reg)
				d967 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r224}
				ctx.BindReg(r224, &d967)
			} else if d966.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d965.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d966.Imm.Int()))
				ctx.EmitSubInt64(scratch, d965.Reg)
				d967 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d967)
			} else if d965.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d966.Reg)
				ctx.EmitMovRegReg(scratch, d966.Reg)
				if d965.Imm.Int() >= -2147483648 && d965.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d965.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d965.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d967 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d967)
			} else {
				r225 := ctx.AllocRegExcept(d966.Reg, d965.Reg)
				ctx.EmitMovRegReg(r225, d966.Reg)
				ctx.EmitSubInt64(r225, d965.Reg)
				d967 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r225}
				ctx.BindReg(r225, &d967)
			}
			if d967.Loc == scm.LocReg && d966.Loc == scm.LocReg && d967.Reg == d966.Reg {
				ctx.TransferReg(d966.Reg)
				d966.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d965)
			ctx.EnsureDesc(&d948)
			ctx.EnsureDesc(&d967)
			var d968 scm.JITValueDesc
			if d948.Loc == scm.LocImm && d967.Loc == scm.LocImm {
				d968 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d948.Imm.Int()) >> uint64(d967.Imm.Int())))}
			} else if d967.Loc == scm.LocImm {
				r226 := ctx.AllocRegExcept(d948.Reg)
				ctx.EmitMovRegReg(r226, d948.Reg)
				ctx.EmitShrRegImm8(r226, uint8(d967.Imm.Int()))
				d968 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r226}
				ctx.BindReg(r226, &d968)
			} else {
				{
					shiftSrc := d948.Reg
					r227 := ctx.AllocRegExcept(d948.Reg)
					ctx.EmitMovRegReg(r227, d948.Reg)
					shiftSrc = r227
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d967.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d967.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d967.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d968 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d968)
				}
			}
			if d968.Loc == scm.LocReg && d948.Loc == scm.LocReg && d968.Reg == d948.Reg {
				ctx.TransferReg(d948.Reg)
				d948.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d948)
			ctx.FreeDesc(&d967)
			r228 := ctx.AllocReg()
			ctx.EnsureDesc(&d968)
			ctx.EnsureDesc(&d968)
			if d968.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r228, d968)
			}
			ctx.EmitJmp(lbl45)
			bbpos_5_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl46)
			ctx.ResolveFixups()
			d948 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(208)}
			ctx.EnsureDesc(&d952)
			var d969 scm.JITValueDesc
			if d952.Loc == scm.LocImm {
				d969 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d952.Imm.Int() / 64)}
			} else {
				r229 := ctx.AllocRegExcept(d952.Reg)
				ctx.EmitMovRegReg(r229, d952.Reg)
				ctx.EmitShrRegImm8(r229, 6)
				d969 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r229}
				ctx.BindReg(r229, &d969)
			}
			if d969.Loc == scm.LocReg && d952.Loc == scm.LocReg && d969.Reg == d952.Reg {
				ctx.TransferReg(d952.Reg)
				d952.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d969)
			ctx.EnsureDesc(&d969)
			var d970 scm.JITValueDesc
			if d969.Loc == scm.LocImm {
				d970 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d969.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d969.Reg)
				ctx.EmitMovRegReg(scratch, d969.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d970 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d970)
			}
			if d970.Loc == scm.LocReg && d969.Loc == scm.LocReg && d970.Reg == d969.Reg {
				ctx.TransferReg(d969.Reg)
				d969.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d969)
			ctx.EnsureDesc(&d970)
			r230 := ctx.AllocReg()
			ctx.EnsureDesc(&d970)
			ctx.EnsureDesc(&d953)
			if d970.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r230, uint64(d970.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r230, d970.Reg)
				ctx.EmitShlRegImm8(r230, 3)
			}
			if d953.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d953.Imm.Int()))
				ctx.EmitAddInt64(r230, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r230, d953.Reg)
			}
			r231 := ctx.AllocRegExcept(r230)
			ctx.EmitMovRegMem(r231, r230, 0)
			ctx.FreeReg(r230)
			d971 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r231}
			ctx.BindReg(r231, &d971)
			ctx.FreeDesc(&d970)
			ctx.EnsureDesc(&d952)
			var d972 scm.JITValueDesc
			if d952.Loc == scm.LocImm {
				d972 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d952.Imm.Int() % 64)}
			} else {
				r232 := ctx.AllocRegExcept(d952.Reg)
				ctx.EmitMovRegReg(r232, d952.Reg)
				ctx.EmitAndRegImm32(r232, 63)
				d972 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r232}
				ctx.BindReg(r232, &d972)
			}
			if d972.Loc == scm.LocReg && d952.Loc == scm.LocReg && d972.Reg == d952.Reg {
				ctx.TransferReg(d952.Reg)
				d952.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d952)
			d973 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d972)
			ctx.EnsureDesc(&d973)
			ctx.EnsureDesc(&d972)
			ctx.EnsureDesc(&d973)
			ctx.EnsureDesc(&d972)
			var d974 scm.JITValueDesc
			if d973.Loc == scm.LocImm && d972.Loc == scm.LocImm {
				d974 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d973.Imm.Int() - d972.Imm.Int())}
			} else if d972.Loc == scm.LocImm && d972.Imm.Int() == 0 {
				r233 := ctx.AllocRegExcept(d973.Reg)
				ctx.EmitMovRegReg(r233, d973.Reg)
				d974 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r233}
				ctx.BindReg(r233, &d974)
			} else if d973.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d972.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d973.Imm.Int()))
				ctx.EmitSubInt64(scratch, d972.Reg)
				d974 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d974)
			} else if d972.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d973.Reg)
				ctx.EmitMovRegReg(scratch, d973.Reg)
				if d972.Imm.Int() >= -2147483648 && d972.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d972.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d972.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d974 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d974)
			} else {
				r234 := ctx.AllocRegExcept(d973.Reg, d972.Reg)
				ctx.EmitMovRegReg(r234, d973.Reg)
				ctx.EmitSubInt64(r234, d972.Reg)
				d974 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
				ctx.BindReg(r234, &d974)
			}
			if d974.Loc == scm.LocReg && d973.Loc == scm.LocReg && d974.Reg == d973.Reg {
				ctx.TransferReg(d973.Reg)
				d973.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d972)
			ctx.EnsureDesc(&d971)
			ctx.EnsureDesc(&d974)
			var d975 scm.JITValueDesc
			if d971.Loc == scm.LocImm && d974.Loc == scm.LocImm {
				d975 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d971.Imm.Int()) >> uint64(d974.Imm.Int())))}
			} else if d974.Loc == scm.LocImm {
				r235 := ctx.AllocRegExcept(d971.Reg)
				ctx.EmitMovRegReg(r235, d971.Reg)
				ctx.EmitShrRegImm8(r235, uint8(d974.Imm.Int()))
				d975 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r235}
				ctx.BindReg(r235, &d975)
			} else {
				{
					shiftSrc := d971.Reg
					r236 := ctx.AllocRegExcept(d971.Reg)
					ctx.EmitMovRegReg(r236, d971.Reg)
					shiftSrc = r236
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d974.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d974.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d974.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d975 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d975)
				}
			}
			if d975.Loc == scm.LocReg && d971.Loc == scm.LocReg && d975.Reg == d971.Reg {
				ctx.TransferReg(d971.Reg)
				d971.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d971)
			ctx.FreeDesc(&d974)
			ctx.EnsureDesc(&d957)
			ctx.EnsureDesc(&d975)
			var d976 scm.JITValueDesc
			if d957.Loc == scm.LocImm && d975.Loc == scm.LocImm {
				d976 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d957.Imm.Int() | d975.Imm.Int())}
			} else if d957.Loc == scm.LocImm && d957.Imm.Int() == 0 {
				d976 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d975.Reg}
				ctx.BindReg(d975.Reg, &d976)
			} else if d975.Loc == scm.LocImm && d975.Imm.Int() == 0 {
				r237 := ctx.AllocRegExcept(d957.Reg)
				ctx.EmitMovRegReg(r237, d957.Reg)
				d976 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r237}
				ctx.BindReg(r237, &d976)
			} else if d957.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d975.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d957.Imm.Int()))
				ctx.EmitOrInt64(scratch, d975.Reg)
				d976 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d976)
			} else if d975.Loc == scm.LocImm {
				r238 := ctx.AllocRegExcept(d957.Reg)
				ctx.EmitMovRegReg(r238, d957.Reg)
				if d975.Imm.Int() >= -2147483648 && d975.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r238, int32(d975.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d975.Imm.Int()))
					ctx.EmitOrInt64(r238, scm.RegR11)
				}
				d976 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r238}
				ctx.BindReg(r238, &d976)
			} else {
				r239 := ctx.AllocRegExcept(d957.Reg, d975.Reg)
				ctx.EmitMovRegReg(r239, d957.Reg)
				ctx.EmitOrInt64(r239, d975.Reg)
				d976 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r239}
				ctx.BindReg(r239, &d976)
			}
			if d976.Loc == scm.LocReg && d957.Loc == scm.LocReg && d976.Reg == d957.Reg {
				ctx.TransferReg(d957.Reg)
				d957.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d975)
			ctx.EnsureDesc(&d976)
			if d976.Loc == scm.LocReg {
				ctx.ProtectReg(d976.Reg)
			} else if d976.Loc == scm.LocRegPair {
				ctx.ProtectReg(d976.Reg)
				ctx.ProtectReg(d976.Reg2)
			}
			d977 = d976
			if d977.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d977)
			ctx.EmitStoreToStack(d977, int32(bbs[2].PhiBase)+int32(0))
			if d976.Loc == scm.LocReg {
				ctx.UnprotectReg(d976.Reg)
			} else if d976.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d976.Reg)
				ctx.UnprotectReg(d976.Reg2)
			}
			ctx.EmitJmp(lbl47)
			ctx.MarkLabel(lbl45)
			d978 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r228}
			ctx.BindReg(r228, &d978)
			ctx.BindReg(r228, &d978)
			if r201 { ctx.UnprotectReg(r202) }
			ctx.FreeDesc(&d3)
			ctx.EnsureDesc(&d978)
			ctx.EnsureDesc(&d978)
			var d979 scm.JITValueDesc
			if d978.Loc == scm.LocImm {
				d979 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d978.Imm.Int()))))}
			} else {
				r240 := ctx.AllocReg()
				ctx.EmitMovRegReg(r240, d978.Reg)
				d979 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r240}
				ctx.BindReg(r240, &d979)
			}
			ctx.FreeDesc(&d978)
			ctx.EnsureDesc(&d979)
			ctx.EnsureDesc(&d57)
			ctx.EnsureDesc(&d979)
			ctx.EnsureDesc(&d57)
			ctx.EnsureDesc(&d979)
			ctx.EnsureDesc(&d57)
			var d980 scm.JITValueDesc
			if d979.Loc == scm.LocImm && d57.Loc == scm.LocImm {
				d980 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d979.Imm.Int() + d57.Imm.Int())}
			} else if d57.Loc == scm.LocImm && d57.Imm.Int() == 0 {
				r241 := ctx.AllocRegExcept(d979.Reg)
				ctx.EmitMovRegReg(r241, d979.Reg)
				d980 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r241}
				ctx.BindReg(r241, &d980)
			} else if d979.Loc == scm.LocImm && d979.Imm.Int() == 0 {
				d980 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d57.Reg}
				ctx.BindReg(d57.Reg, &d980)
			} else if d979.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d57.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d979.Imm.Int()))
				ctx.EmitAddInt64(scratch, d57.Reg)
				d980 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d980)
			} else if d57.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d979.Reg)
				ctx.EmitMovRegReg(scratch, d979.Reg)
				if d57.Imm.Int() >= -2147483648 && d57.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d57.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d57.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d980 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d980)
			} else {
				r242 := ctx.AllocRegExcept(d979.Reg, d57.Reg)
				ctx.EmitMovRegReg(r242, d979.Reg)
				ctx.EmitAddInt64(r242, d57.Reg)
				d980 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r242}
				ctx.BindReg(r242, &d980)
			}
			if d980.Loc == scm.LocReg && d979.Loc == scm.LocReg && d980.Reg == d979.Reg {
				ctx.TransferReg(d979.Reg)
				d979.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d979)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&idxInt)
			var d981 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d981 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(idxInt.Imm.Int()))))}
			} else {
				r243 := ctx.AllocReg()
				ctx.EmitMovRegReg(r243, idxInt.Reg)
				ctx.EmitShlRegImm8(r243, 32)
				ctx.EmitShrRegImm8(r243, 32)
				d981 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r243}
				ctx.BindReg(r243, &d981)
			}
			ctx.FreeDesc(&idxInt)
			ctx.EnsureDesc(&d981)
			ctx.EnsureDesc(&d980)
			ctx.EnsureDesc(&d981)
			ctx.EnsureDesc(&d980)
			ctx.EnsureDesc(&d981)
			ctx.EnsureDesc(&d980)
			var d982 scm.JITValueDesc
			if d981.Loc == scm.LocImm && d980.Loc == scm.LocImm {
				d982 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d981.Imm.Int() - d980.Imm.Int())}
			} else if d980.Loc == scm.LocImm && d980.Imm.Int() == 0 {
				r244 := ctx.AllocRegExcept(d981.Reg)
				ctx.EmitMovRegReg(r244, d981.Reg)
				d982 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r244}
				ctx.BindReg(r244, &d982)
			} else if d981.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d980.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d981.Imm.Int()))
				ctx.EmitSubInt64(scratch, d980.Reg)
				d982 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d982)
			} else if d980.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d981.Reg)
				ctx.EmitMovRegReg(scratch, d981.Reg)
				if d980.Imm.Int() >= -2147483648 && d980.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d980.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d980.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d982 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d982)
			} else {
				r245 := ctx.AllocRegExcept(d981.Reg, d980.Reg)
				ctx.EmitMovRegReg(r245, d981.Reg)
				ctx.EmitSubInt64(r245, d980.Reg)
				d982 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r245}
				ctx.BindReg(r245, &d982)
			}
			if d982.Loc == scm.LocReg && d981.Loc == scm.LocReg && d982.Reg == d981.Reg {
				ctx.TransferReg(d981.Reg)
				d981.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d981)
			ctx.FreeDesc(&d980)
			ctx.EnsureDesc(&d982)
			ctx.EnsureDesc(&d946)
			ctx.EnsureDesc(&d982)
			ctx.EnsureDesc(&d946)
			ctx.EnsureDesc(&d982)
			ctx.EnsureDesc(&d946)
			var d983 scm.JITValueDesc
			if d982.Loc == scm.LocImm && d946.Loc == scm.LocImm {
				d983 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d982.Imm.Int() * d946.Imm.Int())}
			} else if d982.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d946.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d982.Imm.Int()))
				ctx.EmitImulInt64(scratch, d946.Reg)
				d983 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d983)
			} else if d946.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d982.Reg)
				ctx.EmitMovRegReg(scratch, d982.Reg)
				if d946.Imm.Int() >= -2147483648 && d946.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d946.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d946.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d983 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d983)
			} else {
				r246 := ctx.AllocRegExcept(d982.Reg, d946.Reg)
				ctx.EmitMovRegReg(r246, d982.Reg)
				ctx.EmitImulInt64(r246, d946.Reg)
				d983 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r246}
				ctx.BindReg(r246, &d983)
			}
			if d983.Loc == scm.LocReg && d982.Loc == scm.LocReg && d983.Reg == d982.Reg {
				ctx.TransferReg(d982.Reg)
				d982.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d982)
			ctx.FreeDesc(&d946)
			ctx.EnsureDesc(&d170)
			ctx.EnsureDesc(&d983)
			ctx.EnsureDesc(&d170)
			ctx.EnsureDesc(&d983)
			ctx.EnsureDesc(&d170)
			ctx.EnsureDesc(&d983)
			var d984 scm.JITValueDesc
			if d170.Loc == scm.LocImm && d983.Loc == scm.LocImm {
				d984 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d170.Imm.Int() + d983.Imm.Int())}
			} else if d983.Loc == scm.LocImm && d983.Imm.Int() == 0 {
				r247 := ctx.AllocRegExcept(d170.Reg)
				ctx.EmitMovRegReg(r247, d170.Reg)
				d984 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r247}
				ctx.BindReg(r247, &d984)
			} else if d170.Loc == scm.LocImm && d170.Imm.Int() == 0 {
				d984 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d983.Reg}
				ctx.BindReg(d983.Reg, &d984)
			} else if d170.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d983.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d170.Imm.Int()))
				ctx.EmitAddInt64(scratch, d983.Reg)
				d984 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d984)
			} else if d983.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d170.Reg)
				ctx.EmitMovRegReg(scratch, d170.Reg)
				if d983.Imm.Int() >= -2147483648 && d983.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d983.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d983.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d984 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d984)
			} else {
				r248 := ctx.AllocRegExcept(d170.Reg, d983.Reg)
				ctx.EmitMovRegReg(r248, d170.Reg)
				ctx.EmitAddInt64(r248, d983.Reg)
				d984 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r248}
				ctx.BindReg(r248, &d984)
			}
			if d984.Loc == scm.LocReg && d170.Loc == scm.LocReg && d984.Reg == d170.Reg {
				ctx.TransferReg(d170.Reg)
				d170.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d983)
			ctx.EnsureDesc(&d984)
			ctx.EnsureDesc(&d984)
			var d985 scm.JITValueDesc
			if d984.Loc == scm.LocImm {
				d985 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d984.Imm.Int()))}
			} else {
				ctx.EmitCvtInt64ToFloat64(scm.RegX0, d984.Reg)
				d985 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d984.Reg}
				ctx.BindReg(d984.Reg, &d985)
			}
			ctx.FreeDesc(&d984)
			ctx.EnsureDesc(&d985)
			d986 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d986)
			ctx.BindReg(r2, &d986)
			ctx.EnsureDesc(&d985)
			ctx.EmitMakeFloat(d986, d985)
			if d985.Loc == scm.LocReg { ctx.FreeReg(d985.Reg) }
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[13].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[13].VisitCount >= 2 {
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
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
				d14 = ps.OverlayValues[14]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
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
			if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
				d175 = ps.OverlayValues[175]
			}
			if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != scm.LocNone {
				d283 = ps.OverlayValues[283]
			}
			if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != scm.LocNone {
				d284 = ps.OverlayValues[284]
			}
			if len(ps.OverlayValues) > 285 && ps.OverlayValues[285].Loc != scm.LocNone {
				d285 = ps.OverlayValues[285]
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
			if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != scm.LocNone {
				d292 = ps.OverlayValues[292]
			}
			if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != scm.LocNone {
				d293 = ps.OverlayValues[293]
			}
			if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
				d294 = ps.OverlayValues[294]
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
			if len(ps.OverlayValues) > 303 && ps.OverlayValues[303].Loc != scm.LocNone {
				d303 = ps.OverlayValues[303]
			}
			if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != scm.LocNone {
				d305 = ps.OverlayValues[305]
			}
			if len(ps.OverlayValues) > 306 && ps.OverlayValues[306].Loc != scm.LocNone {
				d306 = ps.OverlayValues[306]
			}
			if len(ps.OverlayValues) > 307 && ps.OverlayValues[307].Loc != scm.LocNone {
				d307 = ps.OverlayValues[307]
			}
			if len(ps.OverlayValues) > 308 && ps.OverlayValues[308].Loc != scm.LocNone {
				d308 = ps.OverlayValues[308]
			}
			if len(ps.OverlayValues) > 309 && ps.OverlayValues[309].Loc != scm.LocNone {
				d309 = ps.OverlayValues[309]
			}
			if len(ps.OverlayValues) > 312 && ps.OverlayValues[312].Loc != scm.LocNone {
				d312 = ps.OverlayValues[312]
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
			if len(ps.OverlayValues) > 672 && ps.OverlayValues[672].Loc != scm.LocNone {
				d672 = ps.OverlayValues[672]
			}
			if len(ps.OverlayValues) > 673 && ps.OverlayValues[673].Loc != scm.LocNone {
				d673 = ps.OverlayValues[673]
			}
			if len(ps.OverlayValues) > 674 && ps.OverlayValues[674].Loc != scm.LocNone {
				d674 = ps.OverlayValues[674]
			}
			if len(ps.OverlayValues) > 675 && ps.OverlayValues[675].Loc != scm.LocNone {
				d675 = ps.OverlayValues[675]
			}
			if len(ps.OverlayValues) > 676 && ps.OverlayValues[676].Loc != scm.LocNone {
				d676 = ps.OverlayValues[676]
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
			if len(ps.OverlayValues) > 681 && ps.OverlayValues[681].Loc != scm.LocNone {
				d681 = ps.OverlayValues[681]
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
			if len(ps.OverlayValues) > 687 && ps.OverlayValues[687].Loc != scm.LocNone {
				d687 = ps.OverlayValues[687]
			}
			if len(ps.OverlayValues) > 689 && ps.OverlayValues[689].Loc != scm.LocNone {
				d689 = ps.OverlayValues[689]
			}
			if len(ps.OverlayValues) > 690 && ps.OverlayValues[690].Loc != scm.LocNone {
				d690 = ps.OverlayValues[690]
			}
			if len(ps.OverlayValues) > 691 && ps.OverlayValues[691].Loc != scm.LocNone {
				d691 = ps.OverlayValues[691]
			}
			if len(ps.OverlayValues) > 692 && ps.OverlayValues[692].Loc != scm.LocNone {
				d692 = ps.OverlayValues[692]
			}
			if len(ps.OverlayValues) > 695 && ps.OverlayValues[695].Loc != scm.LocNone {
				d695 = ps.OverlayValues[695]
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
			if len(ps.OverlayValues) > 953 && ps.OverlayValues[953].Loc != scm.LocNone {
				d953 = ps.OverlayValues[953]
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
			ctx.ReclaimUntrackedRegs()
			var d987 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 64
				r249 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r249, fieldAddr)
				d987 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r249}
				ctx.BindReg(r249, &d987)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 64)
				r250 := ctx.AllocReg()
				ctx.EmitMovRegMem(r250, thisptr.Reg, off)
				d987 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r250}
				ctx.BindReg(r250, &d987)
			}
			ctx.EnsureDesc(&d987)
			ctx.EnsureDesc(&d987)
			var d988 scm.JITValueDesc
			if d987.Loc == scm.LocImm {
				d988 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d987.Imm.Int()))))}
			} else {
				r251 := ctx.AllocReg()
				ctx.EmitMovRegReg(r251, d987.Reg)
				d988 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r251}
				ctx.BindReg(r251, &d988)
			}
			ctx.EnsureDesc(&d170)
			ctx.EnsureDesc(&d988)
			ctx.EnsureDesc(&d170)
			ctx.EnsureDesc(&d988)
			ctx.EnsureDesc(&d170)
			ctx.EnsureDesc(&d988)
			var d989 scm.JITValueDesc
			if d170.Loc == scm.LocImm && d988.Loc == scm.LocImm {
				d989 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d170.Imm.Int() == d988.Imm.Int())}
			} else if d988.Loc == scm.LocImm {
				r252 := ctx.AllocRegExcept(d170.Reg)
				if d988.Imm.Int() >= -2147483648 && d988.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d170.Reg, int32(d988.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d988.Imm.Int()))
					ctx.EmitCmpInt64(d170.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r252, scm.CcE)
				d989 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r252}
				ctx.BindReg(r252, &d989)
			} else if d170.Loc == scm.LocImm {
				r253 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d170.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d988.Reg)
				ctx.EmitSetcc(r253, scm.CcE)
				d989 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r253}
				ctx.BindReg(r253, &d989)
			} else {
				r254 := ctx.AllocRegExcept(d170.Reg)
				ctx.EmitCmpInt64(d170.Reg, d988.Reg)
				ctx.EmitSetcc(r254, scm.CcE)
				d989 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r254}
				ctx.BindReg(r254, &d989)
			}
			ctx.FreeDesc(&d170)
			ctx.FreeDesc(&d988)
			d990 = d989
			ctx.EnsureDesc(&d990)
			if d990.Loc != scm.LocImm && d990.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d990.Loc == scm.LocImm {
				if d990.Imm.Bool() {
			ps991 := scm.PhiState{General: ps.General}
			ps991.OverlayValues = make([]scm.JITValueDesc, 991)
			ps991.OverlayValues[0] = d0
			ps991.OverlayValues[1] = d1
			ps991.OverlayValues[2] = d2
			ps991.OverlayValues[3] = d3
			ps991.OverlayValues[4] = d4
			ps991.OverlayValues[5] = d5
			ps991.OverlayValues[6] = d6
			ps991.OverlayValues[7] = d7
			ps991.OverlayValues[8] = d8
			ps991.OverlayValues[9] = d9
			ps991.OverlayValues[10] = d10
			ps991.OverlayValues[11] = d11
			ps991.OverlayValues[12] = d12
			ps991.OverlayValues[13] = d13
			ps991.OverlayValues[14] = d14
			ps991.OverlayValues[15] = d15
			ps991.OverlayValues[16] = d16
			ps991.OverlayValues[18] = d18
			ps991.OverlayValues[19] = d19
			ps991.OverlayValues[20] = d20
			ps991.OverlayValues[21] = d21
			ps991.OverlayValues[22] = d22
			ps991.OverlayValues[23] = d23
			ps991.OverlayValues[24] = d24
			ps991.OverlayValues[25] = d25
			ps991.OverlayValues[26] = d26
			ps991.OverlayValues[27] = d27
			ps991.OverlayValues[28] = d28
			ps991.OverlayValues[29] = d29
			ps991.OverlayValues[30] = d30
			ps991.OverlayValues[31] = d31
			ps991.OverlayValues[32] = d32
			ps991.OverlayValues[33] = d33
			ps991.OverlayValues[34] = d34
			ps991.OverlayValues[35] = d35
			ps991.OverlayValues[36] = d36
			ps991.OverlayValues[37] = d37
			ps991.OverlayValues[38] = d38
			ps991.OverlayValues[39] = d39
			ps991.OverlayValues[40] = d40
			ps991.OverlayValues[41] = d41
			ps991.OverlayValues[42] = d42
			ps991.OverlayValues[43] = d43
			ps991.OverlayValues[44] = d44
			ps991.OverlayValues[45] = d45
			ps991.OverlayValues[46] = d46
			ps991.OverlayValues[47] = d47
			ps991.OverlayValues[48] = d48
			ps991.OverlayValues[49] = d49
			ps991.OverlayValues[50] = d50
			ps991.OverlayValues[51] = d51
			ps991.OverlayValues[52] = d52
			ps991.OverlayValues[53] = d53
			ps991.OverlayValues[54] = d54
			ps991.OverlayValues[55] = d55
			ps991.OverlayValues[56] = d56
			ps991.OverlayValues[57] = d57
			ps991.OverlayValues[58] = d58
			ps991.OverlayValues[59] = d59
			ps991.OverlayValues[60] = d60
			ps991.OverlayValues[61] = d61
			ps991.OverlayValues[64] = d64
			ps991.OverlayValues[65] = d65
			ps991.OverlayValues[66] = d66
			ps991.OverlayValues[134] = d134
			ps991.OverlayValues[135] = d135
			ps991.OverlayValues[136] = d136
			ps991.OverlayValues[137] = d137
			ps991.OverlayValues[138] = d138
			ps991.OverlayValues[139] = d139
			ps991.OverlayValues[140] = d140
			ps991.OverlayValues[141] = d141
			ps991.OverlayValues[142] = d142
			ps991.OverlayValues[143] = d143
			ps991.OverlayValues[144] = d144
			ps991.OverlayValues[145] = d145
			ps991.OverlayValues[146] = d146
			ps991.OverlayValues[147] = d147
			ps991.OverlayValues[148] = d148
			ps991.OverlayValues[149] = d149
			ps991.OverlayValues[150] = d150
			ps991.OverlayValues[151] = d151
			ps991.OverlayValues[152] = d152
			ps991.OverlayValues[153] = d153
			ps991.OverlayValues[154] = d154
			ps991.OverlayValues[155] = d155
			ps991.OverlayValues[156] = d156
			ps991.OverlayValues[157] = d157
			ps991.OverlayValues[158] = d158
			ps991.OverlayValues[159] = d159
			ps991.OverlayValues[160] = d160
			ps991.OverlayValues[161] = d161
			ps991.OverlayValues[162] = d162
			ps991.OverlayValues[163] = d163
			ps991.OverlayValues[164] = d164
			ps991.OverlayValues[165] = d165
			ps991.OverlayValues[166] = d166
			ps991.OverlayValues[167] = d167
			ps991.OverlayValues[168] = d168
			ps991.OverlayValues[169] = d169
			ps991.OverlayValues[170] = d170
			ps991.OverlayValues[171] = d171
			ps991.OverlayValues[172] = d172
			ps991.OverlayValues[175] = d175
			ps991.OverlayValues[283] = d283
			ps991.OverlayValues[284] = d284
			ps991.OverlayValues[285] = d285
			ps991.OverlayValues[286] = d286
			ps991.OverlayValues[287] = d287
			ps991.OverlayValues[288] = d288
			ps991.OverlayValues[289] = d289
			ps991.OverlayValues[290] = d290
			ps991.OverlayValues[292] = d292
			ps991.OverlayValues[293] = d293
			ps991.OverlayValues[294] = d294
			ps991.OverlayValues[295] = d295
			ps991.OverlayValues[296] = d296
			ps991.OverlayValues[297] = d297
			ps991.OverlayValues[298] = d298
			ps991.OverlayValues[299] = d299
			ps991.OverlayValues[300] = d300
			ps991.OverlayValues[301] = d301
			ps991.OverlayValues[303] = d303
			ps991.OverlayValues[305] = d305
			ps991.OverlayValues[306] = d306
			ps991.OverlayValues[307] = d307
			ps991.OverlayValues[308] = d308
			ps991.OverlayValues[309] = d309
			ps991.OverlayValues[312] = d312
			ps991.OverlayValues[443] = d443
			ps991.OverlayValues[444] = d444
			ps991.OverlayValues[445] = d445
			ps991.OverlayValues[446] = d446
			ps991.OverlayValues[447] = d447
			ps991.OverlayValues[448] = d448
			ps991.OverlayValues[449] = d449
			ps991.OverlayValues[451] = d451
			ps991.OverlayValues[452] = d452
			ps991.OverlayValues[453] = d453
			ps991.OverlayValues[454] = d454
			ps991.OverlayValues[455] = d455
			ps991.OverlayValues[456] = d456
			ps991.OverlayValues[457] = d457
			ps991.OverlayValues[458] = d458
			ps991.OverlayValues[459] = d459
			ps991.OverlayValues[460] = d460
			ps991.OverlayValues[461] = d461
			ps991.OverlayValues[462] = d462
			ps991.OverlayValues[463] = d463
			ps991.OverlayValues[464] = d464
			ps991.OverlayValues[465] = d465
			ps991.OverlayValues[466] = d466
			ps991.OverlayValues[467] = d467
			ps991.OverlayValues[468] = d468
			ps991.OverlayValues[469] = d469
			ps991.OverlayValues[470] = d470
			ps991.OverlayValues[471] = d471
			ps991.OverlayValues[472] = d472
			ps991.OverlayValues[473] = d473
			ps991.OverlayValues[474] = d474
			ps991.OverlayValues[475] = d475
			ps991.OverlayValues[476] = d476
			ps991.OverlayValues[477] = d477
			ps991.OverlayValues[478] = d478
			ps991.OverlayValues[479] = d479
			ps991.OverlayValues[480] = d480
			ps991.OverlayValues[481] = d481
			ps991.OverlayValues[482] = d482
			ps991.OverlayValues[483] = d483
			ps991.OverlayValues[484] = d484
			ps991.OverlayValues[485] = d485
			ps991.OverlayValues[486] = d486
			ps991.OverlayValues[487] = d487
			ps991.OverlayValues[488] = d488
			ps991.OverlayValues[489] = d489
			ps991.OverlayValues[490] = d490
			ps991.OverlayValues[672] = d672
			ps991.OverlayValues[673] = d673
			ps991.OverlayValues[674] = d674
			ps991.OverlayValues[675] = d675
			ps991.OverlayValues[676] = d676
			ps991.OverlayValues[678] = d678
			ps991.OverlayValues[679] = d679
			ps991.OverlayValues[680] = d680
			ps991.OverlayValues[681] = d681
			ps991.OverlayValues[682] = d682
			ps991.OverlayValues[683] = d683
			ps991.OverlayValues[684] = d684
			ps991.OverlayValues[685] = d685
			ps991.OverlayValues[687] = d687
			ps991.OverlayValues[689] = d689
			ps991.OverlayValues[690] = d690
			ps991.OverlayValues[691] = d691
			ps991.OverlayValues[692] = d692
			ps991.OverlayValues[695] = d695
			ps991.OverlayValues[892] = d892
			ps991.OverlayValues[893] = d893
			ps991.OverlayValues[894] = d894
			ps991.OverlayValues[895] = d895
			ps991.OverlayValues[897] = d897
			ps991.OverlayValues[898] = d898
			ps991.OverlayValues[899] = d899
			ps991.OverlayValues[900] = d900
			ps991.OverlayValues[901] = d901
			ps991.OverlayValues[902] = d902
			ps991.OverlayValues[903] = d903
			ps991.OverlayValues[904] = d904
			ps991.OverlayValues[905] = d905
			ps991.OverlayValues[906] = d906
			ps991.OverlayValues[908] = d908
			ps991.OverlayValues[909] = d909
			ps991.OverlayValues[910] = d910
			ps991.OverlayValues[911] = d911
			ps991.OverlayValues[912] = d912
			ps991.OverlayValues[913] = d913
			ps991.OverlayValues[914] = d914
			ps991.OverlayValues[915] = d915
			ps991.OverlayValues[916] = d916
			ps991.OverlayValues[917] = d917
			ps991.OverlayValues[918] = d918
			ps991.OverlayValues[919] = d919
			ps991.OverlayValues[920] = d920
			ps991.OverlayValues[921] = d921
			ps991.OverlayValues[922] = d922
			ps991.OverlayValues[923] = d923
			ps991.OverlayValues[924] = d924
			ps991.OverlayValues[925] = d925
			ps991.OverlayValues[926] = d926
			ps991.OverlayValues[927] = d927
			ps991.OverlayValues[928] = d928
			ps991.OverlayValues[929] = d929
			ps991.OverlayValues[930] = d930
			ps991.OverlayValues[931] = d931
			ps991.OverlayValues[932] = d932
			ps991.OverlayValues[933] = d933
			ps991.OverlayValues[934] = d934
			ps991.OverlayValues[935] = d935
			ps991.OverlayValues[936] = d936
			ps991.OverlayValues[937] = d937
			ps991.OverlayValues[938] = d938
			ps991.OverlayValues[939] = d939
			ps991.OverlayValues[940] = d940
			ps991.OverlayValues[941] = d941
			ps991.OverlayValues[942] = d942
			ps991.OverlayValues[943] = d943
			ps991.OverlayValues[944] = d944
			ps991.OverlayValues[945] = d945
			ps991.OverlayValues[946] = d946
			ps991.OverlayValues[947] = d947
			ps991.OverlayValues[948] = d948
			ps991.OverlayValues[949] = d949
			ps991.OverlayValues[950] = d950
			ps991.OverlayValues[951] = d951
			ps991.OverlayValues[952] = d952
			ps991.OverlayValues[953] = d953
			ps991.OverlayValues[954] = d954
			ps991.OverlayValues[955] = d955
			ps991.OverlayValues[956] = d956
			ps991.OverlayValues[957] = d957
			ps991.OverlayValues[958] = d958
			ps991.OverlayValues[959] = d959
			ps991.OverlayValues[960] = d960
			ps991.OverlayValues[961] = d961
			ps991.OverlayValues[962] = d962
			ps991.OverlayValues[963] = d963
			ps991.OverlayValues[964] = d964
			ps991.OverlayValues[965] = d965
			ps991.OverlayValues[966] = d966
			ps991.OverlayValues[967] = d967
			ps991.OverlayValues[968] = d968
			ps991.OverlayValues[969] = d969
			ps991.OverlayValues[970] = d970
			ps991.OverlayValues[971] = d971
			ps991.OverlayValues[972] = d972
			ps991.OverlayValues[973] = d973
			ps991.OverlayValues[974] = d974
			ps991.OverlayValues[975] = d975
			ps991.OverlayValues[976] = d976
			ps991.OverlayValues[977] = d977
			ps991.OverlayValues[978] = d978
			ps991.OverlayValues[979] = d979
			ps991.OverlayValues[980] = d980
			ps991.OverlayValues[981] = d981
			ps991.OverlayValues[982] = d982
			ps991.OverlayValues[983] = d983
			ps991.OverlayValues[984] = d984
			ps991.OverlayValues[985] = d985
			ps991.OverlayValues[986] = d986
			ps991.OverlayValues[987] = d987
			ps991.OverlayValues[988] = d988
			ps991.OverlayValues[989] = d989
			ps991.OverlayValues[990] = d990
					return bbs[11].RenderPS(ps991)
				}
			ps992 := scm.PhiState{General: ps.General}
			ps992.OverlayValues = make([]scm.JITValueDesc, 991)
			ps992.OverlayValues[0] = d0
			ps992.OverlayValues[1] = d1
			ps992.OverlayValues[2] = d2
			ps992.OverlayValues[3] = d3
			ps992.OverlayValues[4] = d4
			ps992.OverlayValues[5] = d5
			ps992.OverlayValues[6] = d6
			ps992.OverlayValues[7] = d7
			ps992.OverlayValues[8] = d8
			ps992.OverlayValues[9] = d9
			ps992.OverlayValues[10] = d10
			ps992.OverlayValues[11] = d11
			ps992.OverlayValues[12] = d12
			ps992.OverlayValues[13] = d13
			ps992.OverlayValues[14] = d14
			ps992.OverlayValues[15] = d15
			ps992.OverlayValues[16] = d16
			ps992.OverlayValues[18] = d18
			ps992.OverlayValues[19] = d19
			ps992.OverlayValues[20] = d20
			ps992.OverlayValues[21] = d21
			ps992.OverlayValues[22] = d22
			ps992.OverlayValues[23] = d23
			ps992.OverlayValues[24] = d24
			ps992.OverlayValues[25] = d25
			ps992.OverlayValues[26] = d26
			ps992.OverlayValues[27] = d27
			ps992.OverlayValues[28] = d28
			ps992.OverlayValues[29] = d29
			ps992.OverlayValues[30] = d30
			ps992.OverlayValues[31] = d31
			ps992.OverlayValues[32] = d32
			ps992.OverlayValues[33] = d33
			ps992.OverlayValues[34] = d34
			ps992.OverlayValues[35] = d35
			ps992.OverlayValues[36] = d36
			ps992.OverlayValues[37] = d37
			ps992.OverlayValues[38] = d38
			ps992.OverlayValues[39] = d39
			ps992.OverlayValues[40] = d40
			ps992.OverlayValues[41] = d41
			ps992.OverlayValues[42] = d42
			ps992.OverlayValues[43] = d43
			ps992.OverlayValues[44] = d44
			ps992.OverlayValues[45] = d45
			ps992.OverlayValues[46] = d46
			ps992.OverlayValues[47] = d47
			ps992.OverlayValues[48] = d48
			ps992.OverlayValues[49] = d49
			ps992.OverlayValues[50] = d50
			ps992.OverlayValues[51] = d51
			ps992.OverlayValues[52] = d52
			ps992.OverlayValues[53] = d53
			ps992.OverlayValues[54] = d54
			ps992.OverlayValues[55] = d55
			ps992.OverlayValues[56] = d56
			ps992.OverlayValues[57] = d57
			ps992.OverlayValues[58] = d58
			ps992.OverlayValues[59] = d59
			ps992.OverlayValues[60] = d60
			ps992.OverlayValues[61] = d61
			ps992.OverlayValues[64] = d64
			ps992.OverlayValues[65] = d65
			ps992.OverlayValues[66] = d66
			ps992.OverlayValues[134] = d134
			ps992.OverlayValues[135] = d135
			ps992.OverlayValues[136] = d136
			ps992.OverlayValues[137] = d137
			ps992.OverlayValues[138] = d138
			ps992.OverlayValues[139] = d139
			ps992.OverlayValues[140] = d140
			ps992.OverlayValues[141] = d141
			ps992.OverlayValues[142] = d142
			ps992.OverlayValues[143] = d143
			ps992.OverlayValues[144] = d144
			ps992.OverlayValues[145] = d145
			ps992.OverlayValues[146] = d146
			ps992.OverlayValues[147] = d147
			ps992.OverlayValues[148] = d148
			ps992.OverlayValues[149] = d149
			ps992.OverlayValues[150] = d150
			ps992.OverlayValues[151] = d151
			ps992.OverlayValues[152] = d152
			ps992.OverlayValues[153] = d153
			ps992.OverlayValues[154] = d154
			ps992.OverlayValues[155] = d155
			ps992.OverlayValues[156] = d156
			ps992.OverlayValues[157] = d157
			ps992.OverlayValues[158] = d158
			ps992.OverlayValues[159] = d159
			ps992.OverlayValues[160] = d160
			ps992.OverlayValues[161] = d161
			ps992.OverlayValues[162] = d162
			ps992.OverlayValues[163] = d163
			ps992.OverlayValues[164] = d164
			ps992.OverlayValues[165] = d165
			ps992.OverlayValues[166] = d166
			ps992.OverlayValues[167] = d167
			ps992.OverlayValues[168] = d168
			ps992.OverlayValues[169] = d169
			ps992.OverlayValues[170] = d170
			ps992.OverlayValues[171] = d171
			ps992.OverlayValues[172] = d172
			ps992.OverlayValues[175] = d175
			ps992.OverlayValues[283] = d283
			ps992.OverlayValues[284] = d284
			ps992.OverlayValues[285] = d285
			ps992.OverlayValues[286] = d286
			ps992.OverlayValues[287] = d287
			ps992.OverlayValues[288] = d288
			ps992.OverlayValues[289] = d289
			ps992.OverlayValues[290] = d290
			ps992.OverlayValues[292] = d292
			ps992.OverlayValues[293] = d293
			ps992.OverlayValues[294] = d294
			ps992.OverlayValues[295] = d295
			ps992.OverlayValues[296] = d296
			ps992.OverlayValues[297] = d297
			ps992.OverlayValues[298] = d298
			ps992.OverlayValues[299] = d299
			ps992.OverlayValues[300] = d300
			ps992.OverlayValues[301] = d301
			ps992.OverlayValues[303] = d303
			ps992.OverlayValues[305] = d305
			ps992.OverlayValues[306] = d306
			ps992.OverlayValues[307] = d307
			ps992.OverlayValues[308] = d308
			ps992.OverlayValues[309] = d309
			ps992.OverlayValues[312] = d312
			ps992.OverlayValues[443] = d443
			ps992.OverlayValues[444] = d444
			ps992.OverlayValues[445] = d445
			ps992.OverlayValues[446] = d446
			ps992.OverlayValues[447] = d447
			ps992.OverlayValues[448] = d448
			ps992.OverlayValues[449] = d449
			ps992.OverlayValues[451] = d451
			ps992.OverlayValues[452] = d452
			ps992.OverlayValues[453] = d453
			ps992.OverlayValues[454] = d454
			ps992.OverlayValues[455] = d455
			ps992.OverlayValues[456] = d456
			ps992.OverlayValues[457] = d457
			ps992.OverlayValues[458] = d458
			ps992.OverlayValues[459] = d459
			ps992.OverlayValues[460] = d460
			ps992.OverlayValues[461] = d461
			ps992.OverlayValues[462] = d462
			ps992.OverlayValues[463] = d463
			ps992.OverlayValues[464] = d464
			ps992.OverlayValues[465] = d465
			ps992.OverlayValues[466] = d466
			ps992.OverlayValues[467] = d467
			ps992.OverlayValues[468] = d468
			ps992.OverlayValues[469] = d469
			ps992.OverlayValues[470] = d470
			ps992.OverlayValues[471] = d471
			ps992.OverlayValues[472] = d472
			ps992.OverlayValues[473] = d473
			ps992.OverlayValues[474] = d474
			ps992.OverlayValues[475] = d475
			ps992.OverlayValues[476] = d476
			ps992.OverlayValues[477] = d477
			ps992.OverlayValues[478] = d478
			ps992.OverlayValues[479] = d479
			ps992.OverlayValues[480] = d480
			ps992.OverlayValues[481] = d481
			ps992.OverlayValues[482] = d482
			ps992.OverlayValues[483] = d483
			ps992.OverlayValues[484] = d484
			ps992.OverlayValues[485] = d485
			ps992.OverlayValues[486] = d486
			ps992.OverlayValues[487] = d487
			ps992.OverlayValues[488] = d488
			ps992.OverlayValues[489] = d489
			ps992.OverlayValues[490] = d490
			ps992.OverlayValues[672] = d672
			ps992.OverlayValues[673] = d673
			ps992.OverlayValues[674] = d674
			ps992.OverlayValues[675] = d675
			ps992.OverlayValues[676] = d676
			ps992.OverlayValues[678] = d678
			ps992.OverlayValues[679] = d679
			ps992.OverlayValues[680] = d680
			ps992.OverlayValues[681] = d681
			ps992.OverlayValues[682] = d682
			ps992.OverlayValues[683] = d683
			ps992.OverlayValues[684] = d684
			ps992.OverlayValues[685] = d685
			ps992.OverlayValues[687] = d687
			ps992.OverlayValues[689] = d689
			ps992.OverlayValues[690] = d690
			ps992.OverlayValues[691] = d691
			ps992.OverlayValues[692] = d692
			ps992.OverlayValues[695] = d695
			ps992.OverlayValues[892] = d892
			ps992.OverlayValues[893] = d893
			ps992.OverlayValues[894] = d894
			ps992.OverlayValues[895] = d895
			ps992.OverlayValues[897] = d897
			ps992.OverlayValues[898] = d898
			ps992.OverlayValues[899] = d899
			ps992.OverlayValues[900] = d900
			ps992.OverlayValues[901] = d901
			ps992.OverlayValues[902] = d902
			ps992.OverlayValues[903] = d903
			ps992.OverlayValues[904] = d904
			ps992.OverlayValues[905] = d905
			ps992.OverlayValues[906] = d906
			ps992.OverlayValues[908] = d908
			ps992.OverlayValues[909] = d909
			ps992.OverlayValues[910] = d910
			ps992.OverlayValues[911] = d911
			ps992.OverlayValues[912] = d912
			ps992.OverlayValues[913] = d913
			ps992.OverlayValues[914] = d914
			ps992.OverlayValues[915] = d915
			ps992.OverlayValues[916] = d916
			ps992.OverlayValues[917] = d917
			ps992.OverlayValues[918] = d918
			ps992.OverlayValues[919] = d919
			ps992.OverlayValues[920] = d920
			ps992.OverlayValues[921] = d921
			ps992.OverlayValues[922] = d922
			ps992.OverlayValues[923] = d923
			ps992.OverlayValues[924] = d924
			ps992.OverlayValues[925] = d925
			ps992.OverlayValues[926] = d926
			ps992.OverlayValues[927] = d927
			ps992.OverlayValues[928] = d928
			ps992.OverlayValues[929] = d929
			ps992.OverlayValues[930] = d930
			ps992.OverlayValues[931] = d931
			ps992.OverlayValues[932] = d932
			ps992.OverlayValues[933] = d933
			ps992.OverlayValues[934] = d934
			ps992.OverlayValues[935] = d935
			ps992.OverlayValues[936] = d936
			ps992.OverlayValues[937] = d937
			ps992.OverlayValues[938] = d938
			ps992.OverlayValues[939] = d939
			ps992.OverlayValues[940] = d940
			ps992.OverlayValues[941] = d941
			ps992.OverlayValues[942] = d942
			ps992.OverlayValues[943] = d943
			ps992.OverlayValues[944] = d944
			ps992.OverlayValues[945] = d945
			ps992.OverlayValues[946] = d946
			ps992.OverlayValues[947] = d947
			ps992.OverlayValues[948] = d948
			ps992.OverlayValues[949] = d949
			ps992.OverlayValues[950] = d950
			ps992.OverlayValues[951] = d951
			ps992.OverlayValues[952] = d952
			ps992.OverlayValues[953] = d953
			ps992.OverlayValues[954] = d954
			ps992.OverlayValues[955] = d955
			ps992.OverlayValues[956] = d956
			ps992.OverlayValues[957] = d957
			ps992.OverlayValues[958] = d958
			ps992.OverlayValues[959] = d959
			ps992.OverlayValues[960] = d960
			ps992.OverlayValues[961] = d961
			ps992.OverlayValues[962] = d962
			ps992.OverlayValues[963] = d963
			ps992.OverlayValues[964] = d964
			ps992.OverlayValues[965] = d965
			ps992.OverlayValues[966] = d966
			ps992.OverlayValues[967] = d967
			ps992.OverlayValues[968] = d968
			ps992.OverlayValues[969] = d969
			ps992.OverlayValues[970] = d970
			ps992.OverlayValues[971] = d971
			ps992.OverlayValues[972] = d972
			ps992.OverlayValues[973] = d973
			ps992.OverlayValues[974] = d974
			ps992.OverlayValues[975] = d975
			ps992.OverlayValues[976] = d976
			ps992.OverlayValues[977] = d977
			ps992.OverlayValues[978] = d978
			ps992.OverlayValues[979] = d979
			ps992.OverlayValues[980] = d980
			ps992.OverlayValues[981] = d981
			ps992.OverlayValues[982] = d982
			ps992.OverlayValues[983] = d983
			ps992.OverlayValues[984] = d984
			ps992.OverlayValues[985] = d985
			ps992.OverlayValues[986] = d986
			ps992.OverlayValues[987] = d987
			ps992.OverlayValues[988] = d988
			ps992.OverlayValues[989] = d989
			ps992.OverlayValues[990] = d990
				return bbs[12].RenderPS(ps992)
			}
			if !ps.General {
				ps.General = true
				return bbs[13].RenderPS(ps)
			}
			lbl50 := ctx.ReserveLabel()
			lbl51 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d990.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl50)
			ctx.EmitJmp(lbl51)
			ctx.MarkLabel(lbl50)
			ctx.EmitJmp(lbl12)
			ctx.MarkLabel(lbl51)
			ctx.EmitJmp(lbl13)
			ps993 := scm.PhiState{General: true}
			ps993.OverlayValues = make([]scm.JITValueDesc, 991)
			ps993.OverlayValues[0] = d0
			ps993.OverlayValues[1] = d1
			ps993.OverlayValues[2] = d2
			ps993.OverlayValues[3] = d3
			ps993.OverlayValues[4] = d4
			ps993.OverlayValues[5] = d5
			ps993.OverlayValues[6] = d6
			ps993.OverlayValues[7] = d7
			ps993.OverlayValues[8] = d8
			ps993.OverlayValues[9] = d9
			ps993.OverlayValues[10] = d10
			ps993.OverlayValues[11] = d11
			ps993.OverlayValues[12] = d12
			ps993.OverlayValues[13] = d13
			ps993.OverlayValues[14] = d14
			ps993.OverlayValues[15] = d15
			ps993.OverlayValues[16] = d16
			ps993.OverlayValues[18] = d18
			ps993.OverlayValues[19] = d19
			ps993.OverlayValues[20] = d20
			ps993.OverlayValues[21] = d21
			ps993.OverlayValues[22] = d22
			ps993.OverlayValues[23] = d23
			ps993.OverlayValues[24] = d24
			ps993.OverlayValues[25] = d25
			ps993.OverlayValues[26] = d26
			ps993.OverlayValues[27] = d27
			ps993.OverlayValues[28] = d28
			ps993.OverlayValues[29] = d29
			ps993.OverlayValues[30] = d30
			ps993.OverlayValues[31] = d31
			ps993.OverlayValues[32] = d32
			ps993.OverlayValues[33] = d33
			ps993.OverlayValues[34] = d34
			ps993.OverlayValues[35] = d35
			ps993.OverlayValues[36] = d36
			ps993.OverlayValues[37] = d37
			ps993.OverlayValues[38] = d38
			ps993.OverlayValues[39] = d39
			ps993.OverlayValues[40] = d40
			ps993.OverlayValues[41] = d41
			ps993.OverlayValues[42] = d42
			ps993.OverlayValues[43] = d43
			ps993.OverlayValues[44] = d44
			ps993.OverlayValues[45] = d45
			ps993.OverlayValues[46] = d46
			ps993.OverlayValues[47] = d47
			ps993.OverlayValues[48] = d48
			ps993.OverlayValues[49] = d49
			ps993.OverlayValues[50] = d50
			ps993.OverlayValues[51] = d51
			ps993.OverlayValues[52] = d52
			ps993.OverlayValues[53] = d53
			ps993.OverlayValues[54] = d54
			ps993.OverlayValues[55] = d55
			ps993.OverlayValues[56] = d56
			ps993.OverlayValues[57] = d57
			ps993.OverlayValues[58] = d58
			ps993.OverlayValues[59] = d59
			ps993.OverlayValues[60] = d60
			ps993.OverlayValues[61] = d61
			ps993.OverlayValues[64] = d64
			ps993.OverlayValues[65] = d65
			ps993.OverlayValues[66] = d66
			ps993.OverlayValues[134] = d134
			ps993.OverlayValues[135] = d135
			ps993.OverlayValues[136] = d136
			ps993.OverlayValues[137] = d137
			ps993.OverlayValues[138] = d138
			ps993.OverlayValues[139] = d139
			ps993.OverlayValues[140] = d140
			ps993.OverlayValues[141] = d141
			ps993.OverlayValues[142] = d142
			ps993.OverlayValues[143] = d143
			ps993.OverlayValues[144] = d144
			ps993.OverlayValues[145] = d145
			ps993.OverlayValues[146] = d146
			ps993.OverlayValues[147] = d147
			ps993.OverlayValues[148] = d148
			ps993.OverlayValues[149] = d149
			ps993.OverlayValues[150] = d150
			ps993.OverlayValues[151] = d151
			ps993.OverlayValues[152] = d152
			ps993.OverlayValues[153] = d153
			ps993.OverlayValues[154] = d154
			ps993.OverlayValues[155] = d155
			ps993.OverlayValues[156] = d156
			ps993.OverlayValues[157] = d157
			ps993.OverlayValues[158] = d158
			ps993.OverlayValues[159] = d159
			ps993.OverlayValues[160] = d160
			ps993.OverlayValues[161] = d161
			ps993.OverlayValues[162] = d162
			ps993.OverlayValues[163] = d163
			ps993.OverlayValues[164] = d164
			ps993.OverlayValues[165] = d165
			ps993.OverlayValues[166] = d166
			ps993.OverlayValues[167] = d167
			ps993.OverlayValues[168] = d168
			ps993.OverlayValues[169] = d169
			ps993.OverlayValues[170] = d170
			ps993.OverlayValues[171] = d171
			ps993.OverlayValues[172] = d172
			ps993.OverlayValues[175] = d175
			ps993.OverlayValues[283] = d283
			ps993.OverlayValues[284] = d284
			ps993.OverlayValues[285] = d285
			ps993.OverlayValues[286] = d286
			ps993.OverlayValues[287] = d287
			ps993.OverlayValues[288] = d288
			ps993.OverlayValues[289] = d289
			ps993.OverlayValues[290] = d290
			ps993.OverlayValues[292] = d292
			ps993.OverlayValues[293] = d293
			ps993.OverlayValues[294] = d294
			ps993.OverlayValues[295] = d295
			ps993.OverlayValues[296] = d296
			ps993.OverlayValues[297] = d297
			ps993.OverlayValues[298] = d298
			ps993.OverlayValues[299] = d299
			ps993.OverlayValues[300] = d300
			ps993.OverlayValues[301] = d301
			ps993.OverlayValues[303] = d303
			ps993.OverlayValues[305] = d305
			ps993.OverlayValues[306] = d306
			ps993.OverlayValues[307] = d307
			ps993.OverlayValues[308] = d308
			ps993.OverlayValues[309] = d309
			ps993.OverlayValues[312] = d312
			ps993.OverlayValues[443] = d443
			ps993.OverlayValues[444] = d444
			ps993.OverlayValues[445] = d445
			ps993.OverlayValues[446] = d446
			ps993.OverlayValues[447] = d447
			ps993.OverlayValues[448] = d448
			ps993.OverlayValues[449] = d449
			ps993.OverlayValues[451] = d451
			ps993.OverlayValues[452] = d452
			ps993.OverlayValues[453] = d453
			ps993.OverlayValues[454] = d454
			ps993.OverlayValues[455] = d455
			ps993.OverlayValues[456] = d456
			ps993.OverlayValues[457] = d457
			ps993.OverlayValues[458] = d458
			ps993.OverlayValues[459] = d459
			ps993.OverlayValues[460] = d460
			ps993.OverlayValues[461] = d461
			ps993.OverlayValues[462] = d462
			ps993.OverlayValues[463] = d463
			ps993.OverlayValues[464] = d464
			ps993.OverlayValues[465] = d465
			ps993.OverlayValues[466] = d466
			ps993.OverlayValues[467] = d467
			ps993.OverlayValues[468] = d468
			ps993.OverlayValues[469] = d469
			ps993.OverlayValues[470] = d470
			ps993.OverlayValues[471] = d471
			ps993.OverlayValues[472] = d472
			ps993.OverlayValues[473] = d473
			ps993.OverlayValues[474] = d474
			ps993.OverlayValues[475] = d475
			ps993.OverlayValues[476] = d476
			ps993.OverlayValues[477] = d477
			ps993.OverlayValues[478] = d478
			ps993.OverlayValues[479] = d479
			ps993.OverlayValues[480] = d480
			ps993.OverlayValues[481] = d481
			ps993.OverlayValues[482] = d482
			ps993.OverlayValues[483] = d483
			ps993.OverlayValues[484] = d484
			ps993.OverlayValues[485] = d485
			ps993.OverlayValues[486] = d486
			ps993.OverlayValues[487] = d487
			ps993.OverlayValues[488] = d488
			ps993.OverlayValues[489] = d489
			ps993.OverlayValues[490] = d490
			ps993.OverlayValues[672] = d672
			ps993.OverlayValues[673] = d673
			ps993.OverlayValues[674] = d674
			ps993.OverlayValues[675] = d675
			ps993.OverlayValues[676] = d676
			ps993.OverlayValues[678] = d678
			ps993.OverlayValues[679] = d679
			ps993.OverlayValues[680] = d680
			ps993.OverlayValues[681] = d681
			ps993.OverlayValues[682] = d682
			ps993.OverlayValues[683] = d683
			ps993.OverlayValues[684] = d684
			ps993.OverlayValues[685] = d685
			ps993.OverlayValues[687] = d687
			ps993.OverlayValues[689] = d689
			ps993.OverlayValues[690] = d690
			ps993.OverlayValues[691] = d691
			ps993.OverlayValues[692] = d692
			ps993.OverlayValues[695] = d695
			ps993.OverlayValues[892] = d892
			ps993.OverlayValues[893] = d893
			ps993.OverlayValues[894] = d894
			ps993.OverlayValues[895] = d895
			ps993.OverlayValues[897] = d897
			ps993.OverlayValues[898] = d898
			ps993.OverlayValues[899] = d899
			ps993.OverlayValues[900] = d900
			ps993.OverlayValues[901] = d901
			ps993.OverlayValues[902] = d902
			ps993.OverlayValues[903] = d903
			ps993.OverlayValues[904] = d904
			ps993.OverlayValues[905] = d905
			ps993.OverlayValues[906] = d906
			ps993.OverlayValues[908] = d908
			ps993.OverlayValues[909] = d909
			ps993.OverlayValues[910] = d910
			ps993.OverlayValues[911] = d911
			ps993.OverlayValues[912] = d912
			ps993.OverlayValues[913] = d913
			ps993.OverlayValues[914] = d914
			ps993.OverlayValues[915] = d915
			ps993.OverlayValues[916] = d916
			ps993.OverlayValues[917] = d917
			ps993.OverlayValues[918] = d918
			ps993.OverlayValues[919] = d919
			ps993.OverlayValues[920] = d920
			ps993.OverlayValues[921] = d921
			ps993.OverlayValues[922] = d922
			ps993.OverlayValues[923] = d923
			ps993.OverlayValues[924] = d924
			ps993.OverlayValues[925] = d925
			ps993.OverlayValues[926] = d926
			ps993.OverlayValues[927] = d927
			ps993.OverlayValues[928] = d928
			ps993.OverlayValues[929] = d929
			ps993.OverlayValues[930] = d930
			ps993.OverlayValues[931] = d931
			ps993.OverlayValues[932] = d932
			ps993.OverlayValues[933] = d933
			ps993.OverlayValues[934] = d934
			ps993.OverlayValues[935] = d935
			ps993.OverlayValues[936] = d936
			ps993.OverlayValues[937] = d937
			ps993.OverlayValues[938] = d938
			ps993.OverlayValues[939] = d939
			ps993.OverlayValues[940] = d940
			ps993.OverlayValues[941] = d941
			ps993.OverlayValues[942] = d942
			ps993.OverlayValues[943] = d943
			ps993.OverlayValues[944] = d944
			ps993.OverlayValues[945] = d945
			ps993.OverlayValues[946] = d946
			ps993.OverlayValues[947] = d947
			ps993.OverlayValues[948] = d948
			ps993.OverlayValues[949] = d949
			ps993.OverlayValues[950] = d950
			ps993.OverlayValues[951] = d951
			ps993.OverlayValues[952] = d952
			ps993.OverlayValues[953] = d953
			ps993.OverlayValues[954] = d954
			ps993.OverlayValues[955] = d955
			ps993.OverlayValues[956] = d956
			ps993.OverlayValues[957] = d957
			ps993.OverlayValues[958] = d958
			ps993.OverlayValues[959] = d959
			ps993.OverlayValues[960] = d960
			ps993.OverlayValues[961] = d961
			ps993.OverlayValues[962] = d962
			ps993.OverlayValues[963] = d963
			ps993.OverlayValues[964] = d964
			ps993.OverlayValues[965] = d965
			ps993.OverlayValues[966] = d966
			ps993.OverlayValues[967] = d967
			ps993.OverlayValues[968] = d968
			ps993.OverlayValues[969] = d969
			ps993.OverlayValues[970] = d970
			ps993.OverlayValues[971] = d971
			ps993.OverlayValues[972] = d972
			ps993.OverlayValues[973] = d973
			ps993.OverlayValues[974] = d974
			ps993.OverlayValues[975] = d975
			ps993.OverlayValues[976] = d976
			ps993.OverlayValues[977] = d977
			ps993.OverlayValues[978] = d978
			ps993.OverlayValues[979] = d979
			ps993.OverlayValues[980] = d980
			ps993.OverlayValues[981] = d981
			ps993.OverlayValues[982] = d982
			ps993.OverlayValues[983] = d983
			ps993.OverlayValues[984] = d984
			ps993.OverlayValues[985] = d985
			ps993.OverlayValues[986] = d986
			ps993.OverlayValues[987] = d987
			ps993.OverlayValues[988] = d988
			ps993.OverlayValues[989] = d989
			ps993.OverlayValues[990] = d990
			ps994 := scm.PhiState{General: true}
			ps994.OverlayValues = make([]scm.JITValueDesc, 991)
			ps994.OverlayValues[0] = d0
			ps994.OverlayValues[1] = d1
			ps994.OverlayValues[2] = d2
			ps994.OverlayValues[3] = d3
			ps994.OverlayValues[4] = d4
			ps994.OverlayValues[5] = d5
			ps994.OverlayValues[6] = d6
			ps994.OverlayValues[7] = d7
			ps994.OverlayValues[8] = d8
			ps994.OverlayValues[9] = d9
			ps994.OverlayValues[10] = d10
			ps994.OverlayValues[11] = d11
			ps994.OverlayValues[12] = d12
			ps994.OverlayValues[13] = d13
			ps994.OverlayValues[14] = d14
			ps994.OverlayValues[15] = d15
			ps994.OverlayValues[16] = d16
			ps994.OverlayValues[18] = d18
			ps994.OverlayValues[19] = d19
			ps994.OverlayValues[20] = d20
			ps994.OverlayValues[21] = d21
			ps994.OverlayValues[22] = d22
			ps994.OverlayValues[23] = d23
			ps994.OverlayValues[24] = d24
			ps994.OverlayValues[25] = d25
			ps994.OverlayValues[26] = d26
			ps994.OverlayValues[27] = d27
			ps994.OverlayValues[28] = d28
			ps994.OverlayValues[29] = d29
			ps994.OverlayValues[30] = d30
			ps994.OverlayValues[31] = d31
			ps994.OverlayValues[32] = d32
			ps994.OverlayValues[33] = d33
			ps994.OverlayValues[34] = d34
			ps994.OverlayValues[35] = d35
			ps994.OverlayValues[36] = d36
			ps994.OverlayValues[37] = d37
			ps994.OverlayValues[38] = d38
			ps994.OverlayValues[39] = d39
			ps994.OverlayValues[40] = d40
			ps994.OverlayValues[41] = d41
			ps994.OverlayValues[42] = d42
			ps994.OverlayValues[43] = d43
			ps994.OverlayValues[44] = d44
			ps994.OverlayValues[45] = d45
			ps994.OverlayValues[46] = d46
			ps994.OverlayValues[47] = d47
			ps994.OverlayValues[48] = d48
			ps994.OverlayValues[49] = d49
			ps994.OverlayValues[50] = d50
			ps994.OverlayValues[51] = d51
			ps994.OverlayValues[52] = d52
			ps994.OverlayValues[53] = d53
			ps994.OverlayValues[54] = d54
			ps994.OverlayValues[55] = d55
			ps994.OverlayValues[56] = d56
			ps994.OverlayValues[57] = d57
			ps994.OverlayValues[58] = d58
			ps994.OverlayValues[59] = d59
			ps994.OverlayValues[60] = d60
			ps994.OverlayValues[61] = d61
			ps994.OverlayValues[64] = d64
			ps994.OverlayValues[65] = d65
			ps994.OverlayValues[66] = d66
			ps994.OverlayValues[134] = d134
			ps994.OverlayValues[135] = d135
			ps994.OverlayValues[136] = d136
			ps994.OverlayValues[137] = d137
			ps994.OverlayValues[138] = d138
			ps994.OverlayValues[139] = d139
			ps994.OverlayValues[140] = d140
			ps994.OverlayValues[141] = d141
			ps994.OverlayValues[142] = d142
			ps994.OverlayValues[143] = d143
			ps994.OverlayValues[144] = d144
			ps994.OverlayValues[145] = d145
			ps994.OverlayValues[146] = d146
			ps994.OverlayValues[147] = d147
			ps994.OverlayValues[148] = d148
			ps994.OverlayValues[149] = d149
			ps994.OverlayValues[150] = d150
			ps994.OverlayValues[151] = d151
			ps994.OverlayValues[152] = d152
			ps994.OverlayValues[153] = d153
			ps994.OverlayValues[154] = d154
			ps994.OverlayValues[155] = d155
			ps994.OverlayValues[156] = d156
			ps994.OverlayValues[157] = d157
			ps994.OverlayValues[158] = d158
			ps994.OverlayValues[159] = d159
			ps994.OverlayValues[160] = d160
			ps994.OverlayValues[161] = d161
			ps994.OverlayValues[162] = d162
			ps994.OverlayValues[163] = d163
			ps994.OverlayValues[164] = d164
			ps994.OverlayValues[165] = d165
			ps994.OverlayValues[166] = d166
			ps994.OverlayValues[167] = d167
			ps994.OverlayValues[168] = d168
			ps994.OverlayValues[169] = d169
			ps994.OverlayValues[170] = d170
			ps994.OverlayValues[171] = d171
			ps994.OverlayValues[172] = d172
			ps994.OverlayValues[175] = d175
			ps994.OverlayValues[283] = d283
			ps994.OverlayValues[284] = d284
			ps994.OverlayValues[285] = d285
			ps994.OverlayValues[286] = d286
			ps994.OverlayValues[287] = d287
			ps994.OverlayValues[288] = d288
			ps994.OverlayValues[289] = d289
			ps994.OverlayValues[290] = d290
			ps994.OverlayValues[292] = d292
			ps994.OverlayValues[293] = d293
			ps994.OverlayValues[294] = d294
			ps994.OverlayValues[295] = d295
			ps994.OverlayValues[296] = d296
			ps994.OverlayValues[297] = d297
			ps994.OverlayValues[298] = d298
			ps994.OverlayValues[299] = d299
			ps994.OverlayValues[300] = d300
			ps994.OverlayValues[301] = d301
			ps994.OverlayValues[303] = d303
			ps994.OverlayValues[305] = d305
			ps994.OverlayValues[306] = d306
			ps994.OverlayValues[307] = d307
			ps994.OverlayValues[308] = d308
			ps994.OverlayValues[309] = d309
			ps994.OverlayValues[312] = d312
			ps994.OverlayValues[443] = d443
			ps994.OverlayValues[444] = d444
			ps994.OverlayValues[445] = d445
			ps994.OverlayValues[446] = d446
			ps994.OverlayValues[447] = d447
			ps994.OverlayValues[448] = d448
			ps994.OverlayValues[449] = d449
			ps994.OverlayValues[451] = d451
			ps994.OverlayValues[452] = d452
			ps994.OverlayValues[453] = d453
			ps994.OverlayValues[454] = d454
			ps994.OverlayValues[455] = d455
			ps994.OverlayValues[456] = d456
			ps994.OverlayValues[457] = d457
			ps994.OverlayValues[458] = d458
			ps994.OverlayValues[459] = d459
			ps994.OverlayValues[460] = d460
			ps994.OverlayValues[461] = d461
			ps994.OverlayValues[462] = d462
			ps994.OverlayValues[463] = d463
			ps994.OverlayValues[464] = d464
			ps994.OverlayValues[465] = d465
			ps994.OverlayValues[466] = d466
			ps994.OverlayValues[467] = d467
			ps994.OverlayValues[468] = d468
			ps994.OverlayValues[469] = d469
			ps994.OverlayValues[470] = d470
			ps994.OverlayValues[471] = d471
			ps994.OverlayValues[472] = d472
			ps994.OverlayValues[473] = d473
			ps994.OverlayValues[474] = d474
			ps994.OverlayValues[475] = d475
			ps994.OverlayValues[476] = d476
			ps994.OverlayValues[477] = d477
			ps994.OverlayValues[478] = d478
			ps994.OverlayValues[479] = d479
			ps994.OverlayValues[480] = d480
			ps994.OverlayValues[481] = d481
			ps994.OverlayValues[482] = d482
			ps994.OverlayValues[483] = d483
			ps994.OverlayValues[484] = d484
			ps994.OverlayValues[485] = d485
			ps994.OverlayValues[486] = d486
			ps994.OverlayValues[487] = d487
			ps994.OverlayValues[488] = d488
			ps994.OverlayValues[489] = d489
			ps994.OverlayValues[490] = d490
			ps994.OverlayValues[672] = d672
			ps994.OverlayValues[673] = d673
			ps994.OverlayValues[674] = d674
			ps994.OverlayValues[675] = d675
			ps994.OverlayValues[676] = d676
			ps994.OverlayValues[678] = d678
			ps994.OverlayValues[679] = d679
			ps994.OverlayValues[680] = d680
			ps994.OverlayValues[681] = d681
			ps994.OverlayValues[682] = d682
			ps994.OverlayValues[683] = d683
			ps994.OverlayValues[684] = d684
			ps994.OverlayValues[685] = d685
			ps994.OverlayValues[687] = d687
			ps994.OverlayValues[689] = d689
			ps994.OverlayValues[690] = d690
			ps994.OverlayValues[691] = d691
			ps994.OverlayValues[692] = d692
			ps994.OverlayValues[695] = d695
			ps994.OverlayValues[892] = d892
			ps994.OverlayValues[893] = d893
			ps994.OverlayValues[894] = d894
			ps994.OverlayValues[895] = d895
			ps994.OverlayValues[897] = d897
			ps994.OverlayValues[898] = d898
			ps994.OverlayValues[899] = d899
			ps994.OverlayValues[900] = d900
			ps994.OverlayValues[901] = d901
			ps994.OverlayValues[902] = d902
			ps994.OverlayValues[903] = d903
			ps994.OverlayValues[904] = d904
			ps994.OverlayValues[905] = d905
			ps994.OverlayValues[906] = d906
			ps994.OverlayValues[908] = d908
			ps994.OverlayValues[909] = d909
			ps994.OverlayValues[910] = d910
			ps994.OverlayValues[911] = d911
			ps994.OverlayValues[912] = d912
			ps994.OverlayValues[913] = d913
			ps994.OverlayValues[914] = d914
			ps994.OverlayValues[915] = d915
			ps994.OverlayValues[916] = d916
			ps994.OverlayValues[917] = d917
			ps994.OverlayValues[918] = d918
			ps994.OverlayValues[919] = d919
			ps994.OverlayValues[920] = d920
			ps994.OverlayValues[921] = d921
			ps994.OverlayValues[922] = d922
			ps994.OverlayValues[923] = d923
			ps994.OverlayValues[924] = d924
			ps994.OverlayValues[925] = d925
			ps994.OverlayValues[926] = d926
			ps994.OverlayValues[927] = d927
			ps994.OverlayValues[928] = d928
			ps994.OverlayValues[929] = d929
			ps994.OverlayValues[930] = d930
			ps994.OverlayValues[931] = d931
			ps994.OverlayValues[932] = d932
			ps994.OverlayValues[933] = d933
			ps994.OverlayValues[934] = d934
			ps994.OverlayValues[935] = d935
			ps994.OverlayValues[936] = d936
			ps994.OverlayValues[937] = d937
			ps994.OverlayValues[938] = d938
			ps994.OverlayValues[939] = d939
			ps994.OverlayValues[940] = d940
			ps994.OverlayValues[941] = d941
			ps994.OverlayValues[942] = d942
			ps994.OverlayValues[943] = d943
			ps994.OverlayValues[944] = d944
			ps994.OverlayValues[945] = d945
			ps994.OverlayValues[946] = d946
			ps994.OverlayValues[947] = d947
			ps994.OverlayValues[948] = d948
			ps994.OverlayValues[949] = d949
			ps994.OverlayValues[950] = d950
			ps994.OverlayValues[951] = d951
			ps994.OverlayValues[952] = d952
			ps994.OverlayValues[953] = d953
			ps994.OverlayValues[954] = d954
			ps994.OverlayValues[955] = d955
			ps994.OverlayValues[956] = d956
			ps994.OverlayValues[957] = d957
			ps994.OverlayValues[958] = d958
			ps994.OverlayValues[959] = d959
			ps994.OverlayValues[960] = d960
			ps994.OverlayValues[961] = d961
			ps994.OverlayValues[962] = d962
			ps994.OverlayValues[963] = d963
			ps994.OverlayValues[964] = d964
			ps994.OverlayValues[965] = d965
			ps994.OverlayValues[966] = d966
			ps994.OverlayValues[967] = d967
			ps994.OverlayValues[968] = d968
			ps994.OverlayValues[969] = d969
			ps994.OverlayValues[970] = d970
			ps994.OverlayValues[971] = d971
			ps994.OverlayValues[972] = d972
			ps994.OverlayValues[973] = d973
			ps994.OverlayValues[974] = d974
			ps994.OverlayValues[975] = d975
			ps994.OverlayValues[976] = d976
			ps994.OverlayValues[977] = d977
			ps994.OverlayValues[978] = d978
			ps994.OverlayValues[979] = d979
			ps994.OverlayValues[980] = d980
			ps994.OverlayValues[981] = d981
			ps994.OverlayValues[982] = d982
			ps994.OverlayValues[983] = d983
			ps994.OverlayValues[984] = d984
			ps994.OverlayValues[985] = d985
			ps994.OverlayValues[986] = d986
			ps994.OverlayValues[987] = d987
			ps994.OverlayValues[988] = d988
			ps994.OverlayValues[989] = d989
			ps994.OverlayValues[990] = d990
			snap995 := d0
			snap996 := d1
			snap997 := d2
			snap998 := d3
			snap999 := d4
			snap1000 := d5
			snap1001 := d6
			snap1002 := d7
			snap1003 := d8
			snap1004 := d9
			snap1005 := d10
			snap1006 := d11
			snap1007 := d12
			snap1008 := d13
			snap1009 := d14
			snap1010 := d15
			snap1011 := d16
			snap1012 := d18
			snap1013 := d19
			snap1014 := d20
			snap1015 := d21
			snap1016 := d22
			snap1017 := d23
			snap1018 := d24
			snap1019 := d25
			snap1020 := d26
			snap1021 := d27
			snap1022 := d28
			snap1023 := d29
			snap1024 := d30
			snap1025 := d31
			snap1026 := d32
			snap1027 := d33
			snap1028 := d34
			snap1029 := d35
			snap1030 := d36
			snap1031 := d37
			snap1032 := d38
			snap1033 := d39
			snap1034 := d40
			snap1035 := d41
			snap1036 := d42
			snap1037 := d43
			snap1038 := d44
			snap1039 := d45
			snap1040 := d46
			snap1041 := d47
			snap1042 := d48
			snap1043 := d49
			snap1044 := d50
			snap1045 := d51
			snap1046 := d52
			snap1047 := d53
			snap1048 := d54
			snap1049 := d55
			snap1050 := d56
			snap1051 := d57
			snap1052 := d58
			snap1053 := d59
			snap1054 := d60
			snap1055 := d61
			snap1056 := d64
			snap1057 := d65
			snap1058 := d66
			snap1059 := d134
			snap1060 := d135
			snap1061 := d136
			snap1062 := d137
			snap1063 := d138
			snap1064 := d139
			snap1065 := d140
			snap1066 := d141
			snap1067 := d142
			snap1068 := d143
			snap1069 := d144
			snap1070 := d145
			snap1071 := d146
			snap1072 := d147
			snap1073 := d148
			snap1074 := d149
			snap1075 := d150
			snap1076 := d151
			snap1077 := d152
			snap1078 := d153
			snap1079 := d154
			snap1080 := d155
			snap1081 := d156
			snap1082 := d157
			snap1083 := d158
			snap1084 := d159
			snap1085 := d160
			snap1086 := d161
			snap1087 := d162
			snap1088 := d163
			snap1089 := d164
			snap1090 := d165
			snap1091 := d166
			snap1092 := d167
			snap1093 := d168
			snap1094 := d169
			snap1095 := d170
			snap1096 := d171
			snap1097 := d172
			snap1098 := d175
			snap1099 := d283
			snap1100 := d284
			snap1101 := d285
			snap1102 := d286
			snap1103 := d287
			snap1104 := d288
			snap1105 := d289
			snap1106 := d290
			snap1107 := d292
			snap1108 := d293
			snap1109 := d294
			snap1110 := d295
			snap1111 := d296
			snap1112 := d297
			snap1113 := d298
			snap1114 := d299
			snap1115 := d300
			snap1116 := d301
			snap1117 := d303
			snap1118 := d305
			snap1119 := d306
			snap1120 := d307
			snap1121 := d308
			snap1122 := d309
			snap1123 := d312
			snap1124 := d443
			snap1125 := d444
			snap1126 := d445
			snap1127 := d446
			snap1128 := d447
			snap1129 := d448
			snap1130 := d449
			snap1131 := d451
			snap1132 := d452
			snap1133 := d453
			snap1134 := d454
			snap1135 := d455
			snap1136 := d456
			snap1137 := d457
			snap1138 := d458
			snap1139 := d459
			snap1140 := d460
			snap1141 := d461
			snap1142 := d462
			snap1143 := d463
			snap1144 := d464
			snap1145 := d465
			snap1146 := d466
			snap1147 := d467
			snap1148 := d468
			snap1149 := d469
			snap1150 := d470
			snap1151 := d471
			snap1152 := d472
			snap1153 := d473
			snap1154 := d474
			snap1155 := d475
			snap1156 := d476
			snap1157 := d477
			snap1158 := d478
			snap1159 := d479
			snap1160 := d480
			snap1161 := d481
			snap1162 := d482
			snap1163 := d483
			snap1164 := d484
			snap1165 := d485
			snap1166 := d486
			snap1167 := d487
			snap1168 := d488
			snap1169 := d489
			snap1170 := d490
			snap1171 := d672
			snap1172 := d673
			snap1173 := d674
			snap1174 := d675
			snap1175 := d676
			snap1176 := d678
			snap1177 := d679
			snap1178 := d680
			snap1179 := d681
			snap1180 := d682
			snap1181 := d683
			snap1182 := d684
			snap1183 := d685
			snap1184 := d687
			snap1185 := d689
			snap1186 := d690
			snap1187 := d691
			snap1188 := d692
			snap1189 := d695
			snap1190 := d892
			snap1191 := d893
			snap1192 := d894
			snap1193 := d895
			snap1194 := d897
			snap1195 := d898
			snap1196 := d899
			snap1197 := d900
			snap1198 := d901
			snap1199 := d902
			snap1200 := d903
			snap1201 := d904
			snap1202 := d905
			snap1203 := d906
			snap1204 := d908
			snap1205 := d909
			snap1206 := d910
			snap1207 := d911
			snap1208 := d912
			snap1209 := d913
			snap1210 := d914
			snap1211 := d915
			snap1212 := d916
			snap1213 := d917
			snap1214 := d918
			snap1215 := d919
			snap1216 := d920
			snap1217 := d921
			snap1218 := d922
			snap1219 := d923
			snap1220 := d924
			snap1221 := d925
			snap1222 := d926
			snap1223 := d927
			snap1224 := d928
			snap1225 := d929
			snap1226 := d930
			snap1227 := d931
			snap1228 := d932
			snap1229 := d933
			snap1230 := d934
			snap1231 := d935
			snap1232 := d936
			snap1233 := d937
			snap1234 := d938
			snap1235 := d939
			snap1236 := d940
			snap1237 := d941
			snap1238 := d942
			snap1239 := d943
			snap1240 := d944
			snap1241 := d945
			snap1242 := d946
			snap1243 := d947
			snap1244 := d948
			snap1245 := d949
			snap1246 := d950
			snap1247 := d951
			snap1248 := d952
			snap1249 := d953
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
			alloc1287 := ctx.SnapshotAllocState()
			if !bbs[12].Rendered {
				bbs[12].RenderPS(ps994)
			}
			ctx.RestoreAllocState(alloc1287)
			d0 = snap995
			d1 = snap996
			d2 = snap997
			d3 = snap998
			d4 = snap999
			d5 = snap1000
			d6 = snap1001
			d7 = snap1002
			d8 = snap1003
			d9 = snap1004
			d10 = snap1005
			d11 = snap1006
			d12 = snap1007
			d13 = snap1008
			d14 = snap1009
			d15 = snap1010
			d16 = snap1011
			d18 = snap1012
			d19 = snap1013
			d20 = snap1014
			d21 = snap1015
			d22 = snap1016
			d23 = snap1017
			d24 = snap1018
			d25 = snap1019
			d26 = snap1020
			d27 = snap1021
			d28 = snap1022
			d29 = snap1023
			d30 = snap1024
			d31 = snap1025
			d32 = snap1026
			d33 = snap1027
			d34 = snap1028
			d35 = snap1029
			d36 = snap1030
			d37 = snap1031
			d38 = snap1032
			d39 = snap1033
			d40 = snap1034
			d41 = snap1035
			d42 = snap1036
			d43 = snap1037
			d44 = snap1038
			d45 = snap1039
			d46 = snap1040
			d47 = snap1041
			d48 = snap1042
			d49 = snap1043
			d50 = snap1044
			d51 = snap1045
			d52 = snap1046
			d53 = snap1047
			d54 = snap1048
			d55 = snap1049
			d56 = snap1050
			d57 = snap1051
			d58 = snap1052
			d59 = snap1053
			d60 = snap1054
			d61 = snap1055
			d64 = snap1056
			d65 = snap1057
			d66 = snap1058
			d134 = snap1059
			d135 = snap1060
			d136 = snap1061
			d137 = snap1062
			d138 = snap1063
			d139 = snap1064
			d140 = snap1065
			d141 = snap1066
			d142 = snap1067
			d143 = snap1068
			d144 = snap1069
			d145 = snap1070
			d146 = snap1071
			d147 = snap1072
			d148 = snap1073
			d149 = snap1074
			d150 = snap1075
			d151 = snap1076
			d152 = snap1077
			d153 = snap1078
			d154 = snap1079
			d155 = snap1080
			d156 = snap1081
			d157 = snap1082
			d158 = snap1083
			d159 = snap1084
			d160 = snap1085
			d161 = snap1086
			d162 = snap1087
			d163 = snap1088
			d164 = snap1089
			d165 = snap1090
			d166 = snap1091
			d167 = snap1092
			d168 = snap1093
			d169 = snap1094
			d170 = snap1095
			d171 = snap1096
			d172 = snap1097
			d175 = snap1098
			d283 = snap1099
			d284 = snap1100
			d285 = snap1101
			d286 = snap1102
			d287 = snap1103
			d288 = snap1104
			d289 = snap1105
			d290 = snap1106
			d292 = snap1107
			d293 = snap1108
			d294 = snap1109
			d295 = snap1110
			d296 = snap1111
			d297 = snap1112
			d298 = snap1113
			d299 = snap1114
			d300 = snap1115
			d301 = snap1116
			d303 = snap1117
			d305 = snap1118
			d306 = snap1119
			d307 = snap1120
			d308 = snap1121
			d309 = snap1122
			d312 = snap1123
			d443 = snap1124
			d444 = snap1125
			d445 = snap1126
			d446 = snap1127
			d447 = snap1128
			d448 = snap1129
			d449 = snap1130
			d451 = snap1131
			d452 = snap1132
			d453 = snap1133
			d454 = snap1134
			d455 = snap1135
			d456 = snap1136
			d457 = snap1137
			d458 = snap1138
			d459 = snap1139
			d460 = snap1140
			d461 = snap1141
			d462 = snap1142
			d463 = snap1143
			d464 = snap1144
			d465 = snap1145
			d466 = snap1146
			d467 = snap1147
			d468 = snap1148
			d469 = snap1149
			d470 = snap1150
			d471 = snap1151
			d472 = snap1152
			d473 = snap1153
			d474 = snap1154
			d475 = snap1155
			d476 = snap1156
			d477 = snap1157
			d478 = snap1158
			d479 = snap1159
			d480 = snap1160
			d481 = snap1161
			d482 = snap1162
			d483 = snap1163
			d484 = snap1164
			d485 = snap1165
			d486 = snap1166
			d487 = snap1167
			d488 = snap1168
			d489 = snap1169
			d490 = snap1170
			d672 = snap1171
			d673 = snap1172
			d674 = snap1173
			d675 = snap1174
			d676 = snap1175
			d678 = snap1176
			d679 = snap1177
			d680 = snap1178
			d681 = snap1179
			d682 = snap1180
			d683 = snap1181
			d684 = snap1182
			d685 = snap1183
			d687 = snap1184
			d689 = snap1185
			d690 = snap1186
			d691 = snap1187
			d692 = snap1188
			d695 = snap1189
			d892 = snap1190
			d893 = snap1191
			d894 = snap1192
			d895 = snap1193
			d897 = snap1194
			d898 = snap1195
			d899 = snap1196
			d900 = snap1197
			d901 = snap1198
			d902 = snap1199
			d903 = snap1200
			d904 = snap1201
			d905 = snap1202
			d906 = snap1203
			d908 = snap1204
			d909 = snap1205
			d910 = snap1206
			d911 = snap1207
			d912 = snap1208
			d913 = snap1209
			d914 = snap1210
			d915 = snap1211
			d916 = snap1212
			d917 = snap1213
			d918 = snap1214
			d919 = snap1215
			d920 = snap1216
			d921 = snap1217
			d922 = snap1218
			d923 = snap1219
			d924 = snap1220
			d925 = snap1221
			d926 = snap1222
			d927 = snap1223
			d928 = snap1224
			d929 = snap1225
			d930 = snap1226
			d931 = snap1227
			d932 = snap1228
			d933 = snap1229
			d934 = snap1230
			d935 = snap1231
			d936 = snap1232
			d937 = snap1233
			d938 = snap1234
			d939 = snap1235
			d940 = snap1236
			d941 = snap1237
			d942 = snap1238
			d943 = snap1239
			d944 = snap1240
			d945 = snap1241
			d946 = snap1242
			d947 = snap1243
			d948 = snap1244
			d949 = snap1245
			d950 = snap1246
			d951 = snap1247
			d952 = snap1248
			d953 = snap1249
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
			if !bbs[11].Rendered {
				return bbs[11].RenderPS(ps993)
			}
			return result
			ctx.FreeDesc(&d989)
			return result
			}
			ps1288 := scm.PhiState{General: false}
			_ = bbs[0].RenderPS(ps1288)
			ctx.MarkLabel(lbl0)
			d1289 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d1289)
			ctx.BindReg(r2, &d1289)
			ctx.EmitMovPairToResult(&d1289, &result)
			ctx.FreeReg(r1)
			ctx.FreeReg(r2)
			ctx.ResolveFixups()
			if idxPinned { ctx.UnprotectReg(idxPinnedReg) }
			ctx.PatchInt32(r0, int32(224))
			ctx.EmitAddRSP32(int32(224))
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
