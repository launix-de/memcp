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
	var d132 scm.JITValueDesc
	_ = d132
	var d133 scm.JITValueDesc
	_ = d133
	var d134 scm.JITValueDesc
	_ = d134
	var phiBase135 int32
	_ = phiBase135
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
	var d282 scm.JITValueDesc
	_ = d282
	var d283 scm.JITValueDesc
	_ = d283
	var d284 scm.JITValueDesc
	_ = d284
	var d285 scm.JITValueDesc
	_ = d285
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
	var d294 scm.JITValueDesc
	_ = d294
	var d296 scm.JITValueDesc
	_ = d296
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
	var d305 scm.JITValueDesc
	_ = d305
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
	var d435 scm.JITValueDesc
	_ = d435
	var d436 scm.JITValueDesc
	_ = d436
	var d437 scm.JITValueDesc
	_ = d437
	var phiBase438 int32
	_ = phiBase438
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
	var d648 scm.JITValueDesc
	_ = d648
	var d649 scm.JITValueDesc
	_ = d649
	var d650 scm.JITValueDesc
	_ = d650
	var d652 scm.JITValueDesc
	_ = d652
	var d653 scm.JITValueDesc
	_ = d653
	var d654 scm.JITValueDesc
	_ = d654
	var d655 scm.JITValueDesc
	_ = d655
	var d656 scm.JITValueDesc
	_ = d656
	var d657 scm.JITValueDesc
	_ = d657
	var d658 scm.JITValueDesc
	_ = d658
	var d660 scm.JITValueDesc
	_ = d660
	var d662 scm.JITValueDesc
	_ = d662
	var d663 scm.JITValueDesc
	_ = d663
	var d664 scm.JITValueDesc
	_ = d664
	var d665 scm.JITValueDesc
	_ = d665
	var d668 scm.JITValueDesc
	_ = d668
	var d853 scm.JITValueDesc
	_ = d853
	var d854 scm.JITValueDesc
	_ = d854
	var d855 scm.JITValueDesc
	_ = d855
	var d856 scm.JITValueDesc
	_ = d856
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
	var phiBase872 int32
	_ = phiBase872
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
	var d879 scm.JITValueDesc
	_ = d879
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
	var phiBase909 int32
	_ = phiBase909
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
		phiBase23 = ctx.AllocStack(int32(16))
		d24 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase23) + int32(0)}
		_ = d24
		lbl15 := ctx.ReserveLabel()
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		lbl16 := ctx.ReserveLabel()
		_ = lbl16
		bbpos_1_1 := int32(-1)
		_ = bbpos_1_1
		lbl17 := ctx.ReserveLabel()
		_ = lbl17
		bbpos_1_2 := int32(-1)
		_ = bbpos_1_2
		lbl18 := ctx.ReserveLabel()
		_ = lbl18
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl16)
		ctx.ResolveFixups()
		d24 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d22)
		ctx.EnsureDesc(&d22)
		var d25 scm.JITValueDesc
		if d22.Loc == scm.LocImm {
			d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d22.Imm.Int()))))}
		} else {
			r6 := ctx.AllocReg()
			ctx.EmitMovRegReg(r6, d22.Reg)
			ctx.EmitShlRegImm8(r6, 32)
			ctx.EmitShrRegImm8(r6, 32)
			d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r6}
			ctx.BindReg(r6, &d25)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d26 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			r7 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r7, fieldAddr)
			d26 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r7}
			ctx.BindReg(r7, &d26)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			r8 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r8, thisptr.Reg, off)
			d26 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r8}
			ctx.BindReg(r8, &d26)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d26)
		ctx.EnsureDesc(&d26)
		var d27 scm.JITValueDesc
		if d26.Loc == scm.LocImm {
			d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d26.Imm.Int()))))}
		} else {
			r9 := ctx.AllocReg()
			ctx.EmitMovRegReg(r9, d26.Reg)
			ctx.EmitShlRegImm8(r9, 56)
			ctx.EmitShrRegImm8(r9, 56)
			d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
			ctx.BindReg(r9, &d27)
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
			r10 := ctx.AllocRegExcept(d25.Reg, d27.Reg)
			ctx.EmitMovRegReg(r10, d25.Reg)
			ctx.EmitImulInt64(r10, d27.Reg)
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
			ctx.BindReg(r10, &d28)
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
			r11 := ctx.AllocReg()
			r12 := ctx.AllocRegExcept(r11)
			r13 := ctx.AllocRegExcept(r11, r12)
			ctx.EmitMovRegMem64(r11, fieldAddr)
			ctx.EmitMovRegMem64(r12, fieldAddr+8)
			ctx.EmitMovRegMem64(r13, fieldAddr+16)
			d29 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r11, Reg2: r12, Reg3: r13}
			ctx.BindReg(r11, &d29)
			ctx.BindReg(r12, &d29)
			ctx.BindReg(r13, &d29)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
			r14 := ctx.AllocReg()
			r15 := ctx.AllocRegExcept(r14)
			r16 := ctx.AllocRegExcept(r14, r15)
			ctx.EmitMovRegMem(r14, thisptr.Reg, off)
			ctx.EmitMovRegMem(r15, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r16, thisptr.Reg, off+16)
			d29 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r14, Reg2: r15, Reg3: r16}
			ctx.BindReg(r14, &d29)
			ctx.BindReg(r15, &d29)
			ctx.BindReg(r16, &d29)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d28)
		var d30 scm.JITValueDesc
		if d28.Loc == scm.LocImm {
			d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d28.Imm.Int() / 64)}
		} else {
			r17 := ctx.AllocRegExcept(d28.Reg)
			ctx.EmitMovRegReg(r17, d28.Reg)
			ctx.EmitShrRegImm8(r17, 6)
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			ctx.BindReg(r17, &d30)
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
			r18 := ctx.AllocRegExcept(d28.Reg)
			ctx.EmitMovRegReg(r18, d28.Reg)
			ctx.EmitAndRegImm32(r18, 63)
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			ctx.BindReg(r18, &d33)
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
			r19 := ctx.AllocRegExcept(d31.Reg)
			ctx.EmitMovRegReg(r19, d31.Reg)
			ctx.EmitShlRegImm8(r19, uint8(d33.Imm.Int()))
			d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d34)
		} else {
			{
				shiftSrc := d31.Reg
				r20 := ctx.AllocRegExcept(d31.Reg)
				ctx.EmitMovRegReg(r20, d31.Reg)
				shiftSrc = r20
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
		ctx.StabilizeDescForControlFlow(&d34)
		ctx.FreeDesc(&d31)
		ctx.FreeDesc(&d33)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d28)
		var d35 scm.JITValueDesc
		if d28.Loc == scm.LocImm {
			d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d28.Imm.Int() % 64)}
		} else {
			r21 := ctx.AllocRegExcept(d28.Reg)
			ctx.EmitMovRegReg(r21, d28.Reg)
			ctx.EmitAndRegImm32(r21, 63)
			d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			ctx.BindReg(r21, &d35)
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
			r22 := ctx.AllocReg()
			ctx.EmitMovRegReg(r22, d26.Reg)
			ctx.EmitShlRegImm8(r22, 56)
			ctx.EmitShrRegImm8(r22, 56)
			d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
			ctx.BindReg(r22, &d36)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d35)
		ctx.EnsureDesc(&d36)
		ctx.EnsureDescsTogether(&d35, &d36)
		var d37 scm.JITValueDesc
		if d35.Loc == scm.LocImm && d36.Loc == scm.LocImm {
			d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d35.Imm.Int() + d36.Imm.Int())}
		} else if d36.Loc == scm.LocImm && d36.Imm.Int() == 0 {
			r23 := ctx.AllocRegExcept(d35.Reg)
			ctx.EmitMovRegReg(r23, d35.Reg)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
			ctx.BindReg(r23, &d37)
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
			r24 := ctx.AllocRegExcept(d35.Reg, d36.Reg)
			ctx.EmitMovRegReg(r24, d35.Reg)
			ctx.EmitAddInt64(r24, d36.Reg)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d37)
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
			r25 := ctx.AllocRegExcept(d37.Reg)
			ctx.EmitCmpRegImm32(d37.Reg, 64)
			ctx.EmitSetcc(r25, scm.CondUnsignedAbove)
			d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r25}
			ctx.BindReg(r25, &d38)
		}
		ctx.FreeDesc(&d37)
		ctx.ReclaimUntrackedRegs()
		d39 = d38
		ctx.EnsureDesc(&d39)
		if d39.Loc != scm.LocImm && d39.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		lbl19 := ctx.ReserveLabel()
		lbl20 := ctx.ReserveLabel()
		if d39.Loc == scm.LocImm {
			if d39.Imm.Bool() {
				ctx.MarkLabel(lbl19)
				ctx.EmitJmp(lbl17)
			} else {
				ctx.MarkLabel(lbl20)
				ctx.SyncDesc(&d34)
				if d34.Loc == scm.LocReg {
					ctx.ProtectReg(d34.Reg)
				} else if d34.Loc == scm.LocRegPair {
					ctx.ProtectReg(d34.Reg)
					ctx.ProtectReg(d34.Reg2)
				}
				d40 = d34
				if d40.Loc == scm.LocNone {
					panic("jit: phi source has no location")
				}
				ctx.EnsureDesc(&d40)
				ctx.EmitStoreToStack(d40, int32(phiBase23)+int32(0))
				if d34.Loc == scm.LocReg {
					ctx.UnprotectReg(d34.Reg)
				} else if d34.Loc == scm.LocRegPair {
					ctx.UnprotectReg(d34.Reg)
					ctx.UnprotectReg(d34.Reg2)
				}
				ctx.EmitJmp(lbl18)
			}
		} else {
			ctx.EmitCmpRegImm32(d39.Reg, 0)
			ctx.EmitJump(scm.CondNotEqual, lbl19)
			ctx.EmitJmp(lbl20)
			ctx.MarkLabel(lbl19)
			ctx.EmitJmp(lbl17)
			ctx.MarkLabel(lbl20)
			ctx.SyncDesc(&d34)
			if d34.Loc == scm.LocReg {
				ctx.ProtectReg(d34.Reg)
			} else if d34.Loc == scm.LocRegPair {
				ctx.ProtectReg(d34.Reg)
				ctx.ProtectReg(d34.Reg2)
			}
			d41 = d34
			if d41.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d41)
			ctx.EmitStoreToStack(d41, int32(phiBase23)+int32(0))
			if d34.Loc == scm.LocReg {
				ctx.UnprotectReg(d34.Reg)
			} else if d34.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d34.Reg)
				ctx.UnprotectReg(d34.Reg2)
			}
			ctx.EmitJmp(lbl18)
		}
		ctx.FreeDesc(&d38)
		bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl18)
		ctx.ResolveFixups()
		d24 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d26)
		ctx.EnsureDesc(&d26)
		var d42 scm.JITValueDesc
		if d26.Loc == scm.LocImm {
			d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d26.Imm.Int()))))}
		} else {
			r26 := ctx.AllocReg()
			ctx.EmitMovRegReg(r26, d26.Reg)
			ctx.EmitShlRegImm8(r26, 56)
			ctx.EmitShrRegImm8(r26, 56)
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
			ctx.BindReg(r26, &d42)
		}
		ctx.ReclaimUntrackedRegs()
		d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d42)
		ctx.EnsureDescsTogether(&d43, &d42)
		var d44 scm.JITValueDesc
		if d43.Loc == scm.LocImm && d42.Loc == scm.LocImm {
			d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d43.Imm.Int() - d42.Imm.Int())}
		} else if d42.Loc == scm.LocImm && d42.Imm.Int() == 0 {
			r27 := ctx.AllocRegExcept(d43.Reg)
			ctx.EmitMovRegReg(r27, d43.Reg)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
			ctx.BindReg(r27, &d44)
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
			r28 := ctx.AllocRegExcept(d43.Reg, d42.Reg)
			ctx.EmitMovRegReg(r28, d43.Reg)
			ctx.EmitSubInt64(r28, d42.Reg)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
			ctx.BindReg(r28, &d44)
		}
		if d44.Loc == scm.LocReg && d43.Loc == scm.LocReg && d44.Reg == d43.Reg {
			ctx.TransferReg(d43.Reg)
			d43.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d42)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d24)
		ctx.EnsureDesc(&d44)
		var d45 scm.JITValueDesc
		if d24.Loc == scm.LocImm && d44.Loc == scm.LocImm {
			d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d24.Imm.Int()) >> uint64(d44.Imm.Int())))}
		} else if d44.Loc == scm.LocImm {
			r29 := ctx.AllocRegExcept(d24.Reg)
			ctx.EmitMovRegReg(r29, d24.Reg)
			ctx.EmitShrRegImm8(r29, uint8(d44.Imm.Int()))
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
			ctx.BindReg(r29, &d45)
		} else {
			{
				shiftSrc := d24.Reg
				r30 := ctx.AllocRegExcept(d24.Reg)
				ctx.EmitMovRegReg(r30, d24.Reg)
				shiftSrc = r30
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d44.Reg != scm.RegRCX
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
		if d45.Loc == scm.LocReg && d24.Loc == scm.LocReg && d45.Reg == d24.Reg {
			ctx.TransferReg(d24.Reg)
			d24.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d24)
		ctx.FreeDesc(&d44)
		ctx.ReclaimUntrackedRegs()
		r31 := ctx.AllocReg()
		ctx.EnsureDesc(&d45)
		ctx.EnsureDesc(&d45)
		if d45.Loc == scm.LocRegPair {
			panic("jit: scalar inline return has scm.LocRegPair")
		} else {
			ctx.EmitMovToReg(r31, d45)
		}
		ctx.EmitJmp(lbl15)
		bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl17)
		ctx.ResolveFixups()
		d24 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d28)
		var d46 scm.JITValueDesc
		if d28.Loc == scm.LocImm {
			d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d28.Imm.Int() / 64)}
		} else {
			r32 := ctx.AllocRegExcept(d28.Reg)
			ctx.EmitMovRegReg(r32, d28.Reg)
			ctx.EmitShrRegImm8(r32, 6)
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
			ctx.BindReg(r32, &d46)
		}
		if d46.Loc == scm.LocReg && d28.Loc == scm.LocReg && d46.Reg == d28.Reg {
			ctx.TransferReg(d28.Reg)
			d28.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d47)
		ctx.ReclaimUntrackedRegs()
		d49 = ctx.EmitSliceElementAddress(&d29, &d47, 8)
		ctx.EnsureDesc(&d49)
		ctx.EmitMovRegMem(d49.Reg, d49.Reg, 0)
		d48 = d49
		ctx.FreeDesc(&d47)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d28)
		var d50 scm.JITValueDesc
		if d28.Loc == scm.LocImm {
			d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d28.Imm.Int() % 64)}
		} else {
			r33 := ctx.AllocRegExcept(d28.Reg)
			ctx.EmitMovRegReg(r33, d28.Reg)
			ctx.EmitAndRegImm32(r33, 63)
			d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
			ctx.BindReg(r33, &d50)
		}
		if d50.Loc == scm.LocReg && d28.Loc == scm.LocReg && d50.Reg == d28.Reg {
			ctx.TransferReg(d28.Reg)
			d28.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d28)
		ctx.ReclaimUntrackedRegs()
		d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d50)
		ctx.EnsureDescsTogether(&d51, &d50)
		var d52 scm.JITValueDesc
		if d51.Loc == scm.LocImm && d50.Loc == scm.LocImm {
			d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d51.Imm.Int() - d50.Imm.Int())}
		} else if d50.Loc == scm.LocImm && d50.Imm.Int() == 0 {
			r34 := ctx.AllocRegExcept(d51.Reg)
			ctx.EmitMovRegReg(r34, d51.Reg)
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
			ctx.BindReg(r34, &d52)
		} else if d51.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d50.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d51.Imm.Int()))
			ctx.EmitSubInt64(scratch, d50.Reg)
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d52)
		} else if d50.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d51.Reg)
			ctx.EmitMovRegReg(scratch, d51.Reg)
			if d50.Imm.Int() >= -2147483648 && d50.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d50.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d50.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d52)
		} else {
			r35 := ctx.AllocRegExcept(d51.Reg, d50.Reg)
			ctx.EmitMovRegReg(r35, d51.Reg)
			ctx.EmitSubInt64(r35, d50.Reg)
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
			ctx.BindReg(r35, &d52)
		}
		if d52.Loc == scm.LocReg && d51.Loc == scm.LocReg && d52.Reg == d51.Reg {
			ctx.TransferReg(d51.Reg)
			d51.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d50)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d48)
		ctx.EnsureDesc(&d52)
		var d53 scm.JITValueDesc
		if d48.Loc == scm.LocImm && d52.Loc == scm.LocImm {
			d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d48.Imm.Int()) >> uint64(d52.Imm.Int())))}
		} else if d52.Loc == scm.LocImm {
			r36 := ctx.AllocRegExcept(d48.Reg)
			ctx.EmitMovRegReg(r36, d48.Reg)
			ctx.EmitShrRegImm8(r36, uint8(d52.Imm.Int()))
			d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
			ctx.BindReg(r36, &d53)
		} else {
			{
				shiftSrc := d48.Reg
				r37 := ctx.AllocRegExcept(d48.Reg)
				ctx.EmitMovRegReg(r37, d48.Reg)
				shiftSrc = r37
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d52.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d52.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d52.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d53)
			}
		}
		if d53.Loc == scm.LocReg && d48.Loc == scm.LocReg && d53.Reg == d48.Reg {
			ctx.TransferReg(d48.Reg)
			d48.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d48)
		ctx.FreeDesc(&d52)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d34)
		ctx.EnsureDesc(&d53)
		var d54 scm.JITValueDesc
		if d34.Loc == scm.LocImm && d53.Loc == scm.LocImm {
			d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d34.Imm.Int() | d53.Imm.Int())}
		} else if d34.Loc == scm.LocImm && d34.Imm.Int() == 0 {
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d53.Reg}
			ctx.BindReg(d53.Reg, &d54)
		} else if d53.Loc == scm.LocImm && d53.Imm.Int() == 0 {
			r38 := ctx.AllocRegExcept(d34.Reg)
			ctx.EmitMovRegReg(r38, d34.Reg)
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
			ctx.BindReg(r38, &d54)
		} else if d34.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d53.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d34.Imm.Int()))
			ctx.EmitOrInt64(scratch, d53.Reg)
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d54)
		} else if d53.Loc == scm.LocImm {
			r39 := ctx.AllocRegExcept(d34.Reg)
			ctx.EmitMovRegReg(r39, d34.Reg)
			if d53.Imm.Int() >= -2147483648 && d53.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r39, int32(d53.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d53.Imm.Int()))
				ctx.EmitOrInt64(r39, scm.RegR11)
			}
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
			ctx.BindReg(r39, &d54)
		} else {
			r40 := ctx.AllocRegExcept(d34.Reg, d53.Reg)
			ctx.EmitMovRegReg(r40, d34.Reg)
			ctx.EmitOrInt64(r40, d53.Reg)
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
			ctx.BindReg(r40, &d54)
		}
		if d54.Loc == scm.LocReg && d34.Loc == scm.LocReg && d54.Reg == d34.Reg {
			ctx.TransferReg(d34.Reg)
			d34.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d54)
		ctx.EmitStoreToStack(d54, int32(phiBase23)+int32(0))
		ctx.StabilizeDescForControlFlow(&d54)
		ctx.FreeDesc(&d53)
		ctx.ReclaimUntrackedRegs()
		ctx.EmitJmp(lbl18)
		ctx.MarkLabel(lbl15)
		d55 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r31}
		ctx.BindReg(r31, &d55)
		ctx.BindReg(r31, &d55)
		ctx.EnsureDesc(&d55)
		ctx.EnsureDesc(&d55)
		var d56 scm.JITValueDesc
		if d55.Loc == scm.LocImm {
			d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d55.Imm.Int()))))}
		} else {
			r41 := ctx.AllocReg()
			ctx.EmitMovRegReg(r41, d55.Reg)
			d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
			ctx.BindReg(r41, &d56)
		}
		ctx.FreeDesc(&d55)
		var d57 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
			r42 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r42, fieldAddr)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r42}
			ctx.BindReg(r42, &d57)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
			r43 := ctx.AllocReg()
			ctx.EmitMovRegMem(r43, thisptr.Reg, off)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r43}
			ctx.BindReg(r43, &d57)
		}
		ctx.EnsureDesc(&d56)
		ctx.EnsureDesc(&d57)
		ctx.EnsureDescsTogether(&d56, &d57)
		var d58 scm.JITValueDesc
		if d56.Loc == scm.LocImm && d57.Loc == scm.LocImm {
			d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d56.Imm.Int() + d57.Imm.Int())}
		} else if d57.Loc == scm.LocImm && d57.Imm.Int() == 0 {
			r44 := ctx.AllocRegExcept(d56.Reg)
			ctx.EmitMovRegReg(r44, d56.Reg)
			d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
			ctx.BindReg(r44, &d58)
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
			r45 := ctx.AllocRegExcept(d56.Reg, d57.Reg)
			ctx.EmitMovRegReg(r45, d56.Reg)
			ctx.EmitAddInt64(r45, d57.Reg)
			d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
			ctx.BindReg(r45, &d58)
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
			r46 := ctx.AllocReg()
			ctx.EmitMovRegReg(r46, d58.Reg)
			ctx.EmitShlRegImm8(r46, 32)
			ctx.EmitShrRegImm8(r46, 32)
			d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
			ctx.BindReg(r46, &d59)
		}
		ctx.FreeDesc(&d58)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d59)
		ctx.EnsureDescsTogether(&idxInt, &d59)
		var d60 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm && d59.Loc == scm.LocImm {
			d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d59.Imm.Int()))}
		} else if d59.Loc == scm.LocImm {
			r47 := ctx.AllocRegExcept(idxInt.Reg)
			if d59.Imm.Int() >= -2147483648 && d59.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(idxInt.Reg, int32(d59.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d59.Imm.Int()))
				ctx.EmitCmpInt64(idxInt.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r47, scm.CondUnsignedBelow)
			d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r47}
			ctx.BindReg(r47, &d60)
		} else if idxInt.Loc == scm.LocImm {
			r48 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d59.Reg)
			ctx.EmitSetcc(r48, scm.CondUnsignedBelow)
			d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r48}
			ctx.BindReg(r48, &d60)
		} else {
			r49 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitCmpInt64(idxInt.Reg, d59.Reg)
			ctx.EmitSetcc(r49, scm.CondUnsignedBelow)
			d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r49}
			ctx.BindReg(r49, &d60)
		}
		ctx.FreeDesc(&d59)
		d61 = d60
		ctx.EnsureDesc(&d61)
		if d61.Loc != scm.LocImm && d61.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d61.Loc == scm.LocImm {
			if d61.Imm.Bool() {
				if ps.General {
				}
				ps62 := scm.PhiState{General: ps.General}
				ps62.OverlayValues = make([]scm.JITValueDesc, 62)
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
				ps62.OverlayValues[17] = d17
				ps62.OverlayValues[18] = d18
				ps62.OverlayValues[19] = d19
				ps62.OverlayValues[20] = d20
				ps62.OverlayValues[21] = d21
				ps62.OverlayValues[22] = d22
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
			if ps.General {
			}
			ps63 := scm.PhiState{General: ps.General}
			ps63.OverlayValues = make([]scm.JITValueDesc, 62)
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
			ps63.OverlayValues[17] = d17
			ps63.OverlayValues[18] = d18
			ps63.OverlayValues[19] = d19
			ps63.OverlayValues[20] = d20
			ps63.OverlayValues[21] = d21
			ps63.OverlayValues[22] = d22
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
		lbl21 := ctx.ReserveLabel()
		lbl22 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d61.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl21)
		ctx.EmitJmp(lbl22)
		ctx.MarkLabel(lbl21)
		ctx.EmitJmp(lbl4)
		ctx.MarkLabel(lbl22)
		ctx.EmitJmp(lbl6)
		ps67 := scm.PhiState{General: true}
		ps67.OverlayValues = make([]scm.JITValueDesc, 67)
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
		ps67.OverlayValues[17] = d17
		ps67.OverlayValues[18] = d18
		ps67.OverlayValues[19] = d19
		ps67.OverlayValues[20] = d20
		ps67.OverlayValues[21] = d21
		ps67.OverlayValues[22] = d22
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
		ps68.OverlayValues[17] = d17
		ps68.OverlayValues[18] = d18
		ps68.OverlayValues[19] = d19
		ps68.OverlayValues[20] = d20
		ps68.OverlayValues[21] = d21
		ps68.OverlayValues[22] = d22
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
		snap69 := d1
		snap70 := d2
		snap71 := d3
		snap72 := d4
		snap73 := d5
		snap74 := d6
		snap75 := d7
		snap76 := d8
		snap77 := d9
		snap78 := d10
		snap79 := d11
		snap80 := d12
		snap81 := d13
		snap82 := d14
		snap83 := d15
		snap84 := d17
		snap85 := d18
		snap86 := d19
		snap87 := d20
		snap88 := d21
		snap89 := d22
		snap90 := d24
		snap91 := d25
		snap92 := d26
		snap93 := d27
		snap94 := d28
		snap95 := d29
		snap96 := d30
		snap97 := d31
		snap98 := d32
		snap99 := d33
		snap100 := d34
		snap101 := d35
		snap102 := d36
		snap103 := d37
		snap104 := d38
		snap105 := d39
		snap106 := d40
		snap107 := d41
		snap108 := d42
		snap109 := d43
		snap110 := d44
		snap111 := d45
		snap112 := d46
		snap113 := d47
		snap114 := d48
		snap115 := d49
		snap116 := d50
		snap117 := d51
		snap118 := d52
		snap119 := d53
		snap120 := d54
		snap121 := d55
		snap122 := d56
		snap123 := d57
		snap124 := d58
		snap125 := d59
		snap126 := d60
		snap127 := d61
		snap128 := d64
		snap129 := d65
		snap130 := d66
		alloc131 := ctx.SnapshotAllocState()
		if !bbs[5].Rendered {
			bbs[5].RenderPS(ps68)
		}
		ctx.RestoreAllocState(alloc131)
		d1 = snap69
		d2 = snap70
		d3 = snap71
		d4 = snap72
		d5 = snap73
		d6 = snap74
		d7 = snap75
		d8 = snap76
		d9 = snap77
		d10 = snap78
		d11 = snap79
		d12 = snap80
		d13 = snap81
		d14 = snap82
		d15 = snap83
		d17 = snap84
		d18 = snap85
		d19 = snap86
		d20 = snap87
		d21 = snap88
		d22 = snap89
		d24 = snap90
		d25 = snap91
		d26 = snap92
		d27 = snap93
		d28 = snap94
		d29 = snap95
		d30 = snap96
		d31 = snap97
		d32 = snap98
		d33 = snap99
		d34 = snap100
		d35 = snap101
		d36 = snap102
		d37 = snap103
		d38 = snap104
		d39 = snap105
		d40 = snap106
		d41 = snap107
		d42 = snap108
		d43 = snap109
		d44 = snap110
		d45 = snap111
		d46 = snap112
		d47 = snap113
		d48 = snap114
		d49 = snap115
		d50 = snap116
		d51 = snap117
		d52 = snap118
		d53 = snap119
		d54 = snap120
		d55 = snap121
		d56 = snap122
		d57 = snap123
		d58 = snap124
		d59 = snap125
		d60 = snap126
		d61 = snap127
		d64 = snap128
		d65 = snap129
		d66 = snap130
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
				d132 := ps.PhiValues[0]
				ctx.EnsureDesc(&d132)
				ctx.EmitStoreToStack(d132, int32(bbs[2].PhiBase)+int32(0))
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
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
			d4 = ps.PhiValues[0]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d4)
		ctx.EnsureDesc(&d4)
		ctx.EnsureDesc(&d4)
		var d133 scm.JITValueDesc
		if d4.Loc == scm.LocImm {
			d133 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(d4.Imm.Int()))))}
		} else {
			r50 := ctx.AllocReg()
			ctx.EmitMovRegReg(r50, d4.Reg)
			ctx.EmitShlRegImm8(r50, 32)
			ctx.EmitShrRegImm8(r50, 32)
			d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
			ctx.BindReg(r50, &d133)
		}
		ctx.EnsureDesc(&d133)
		if thisptr.Loc == scm.LocImm {
			baseReg := ctx.AllocReg()
			if d133.Loc == scm.LocReg {
				ctx.FreeReg(baseReg)
				baseReg = ctx.AllocRegExcept(d133.Reg)
			}
			ctx.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int())+unsafe.Offsetof((*StorageSeq)(nil).lastValue)))
			if d133.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d133.Imm.Int()))
				ctx.EmitStoreRegMem(scm.RegR11, baseReg, 0)
			} else {
				ctx.EmitStoreRegMem(d133.Reg, baseReg, 0)
			}
			ctx.FreeReg(baseReg)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
			if d133.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d133.Imm.Int()))
				ctx.EmitStoreRegMem(scm.RegR11, thisptr.Reg, off)
			} else {
				ctx.EmitStoreRegMem(d133.Reg, thisptr.Reg, off)
			}
		}
		ctx.FreeDesc(&d133)
		ctx.EnsureDesc(&d4)
		d134 = d4
		_ = d134
		ctx.StabilizeDescForControlFlow(&d134)
		ctx.StabilizeDescForControlFlow(&d4)
		phiBase135 = ctx.AllocStack(int32(16))
		d136 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase135) + int32(0)}
		_ = d136
		lbl23 := ctx.ReserveLabel()
		bbpos_2_0 := int32(-1)
		_ = bbpos_2_0
		lbl24 := ctx.ReserveLabel()
		_ = lbl24
		bbpos_2_1 := int32(-1)
		_ = bbpos_2_1
		lbl25 := ctx.ReserveLabel()
		_ = lbl25
		bbpos_2_2 := int32(-1)
		_ = bbpos_2_2
		lbl26 := ctx.ReserveLabel()
		_ = lbl26
		bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl24)
		ctx.ResolveFixups()
		d136 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d134)
		ctx.EnsureDesc(&d134)
		var d137 scm.JITValueDesc
		if d134.Loc == scm.LocImm {
			d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d134.Imm.Int()))))}
		} else {
			r51 := ctx.AllocReg()
			ctx.EmitMovRegReg(r51, d134.Reg)
			ctx.EmitShlRegImm8(r51, 32)
			ctx.EmitShrRegImm8(r51, 32)
			d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
			ctx.BindReg(r51, &d137)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d138 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
			r52 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r52, fieldAddr)
			d138 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r52}
			ctx.BindReg(r52, &d138)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
			r53 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r53, thisptr.Reg, off)
			d138 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r53}
			ctx.BindReg(r53, &d138)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d138)
		ctx.EnsureDesc(&d138)
		var d139 scm.JITValueDesc
		if d138.Loc == scm.LocImm {
			d139 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d138.Imm.Int()))))}
		} else {
			r54 := ctx.AllocReg()
			ctx.EmitMovRegReg(r54, d138.Reg)
			ctx.EmitShlRegImm8(r54, 56)
			ctx.EmitShrRegImm8(r54, 56)
			d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
			ctx.BindReg(r54, &d139)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d137)
		ctx.EnsureDesc(&d139)
		ctx.EnsureDescsTogether(&d137, &d139)
		var d140 scm.JITValueDesc
		if d137.Loc == scm.LocImm && d139.Loc == scm.LocImm {
			d140 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d137.Imm.Int() * d139.Imm.Int())}
		} else if d137.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d139.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d137.Imm.Int()))
			ctx.EmitImulInt64(scratch, d139.Reg)
			d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d140)
		} else if d139.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d137.Reg)
			ctx.EmitMovRegReg(scratch, d137.Reg)
			if d139.Imm.Int() >= -2147483648 && d139.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d139.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d139.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d140)
		} else {
			r55 := ctx.AllocRegExcept(d137.Reg, d139.Reg)
			ctx.EmitMovRegReg(r55, d137.Reg)
			ctx.EmitImulInt64(r55, d139.Reg)
			d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
			ctx.BindReg(r55, &d140)
		}
		if d140.Loc == scm.LocReg && d137.Loc == scm.LocReg && d140.Reg == d137.Reg {
			ctx.TransferReg(d137.Reg)
			d137.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d140)
		ctx.FreeDesc(&d137)
		ctx.FreeDesc(&d139)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d141 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 0
			r56 := ctx.AllocReg()
			r57 := ctx.AllocRegExcept(r56)
			r58 := ctx.AllocRegExcept(r56, r57)
			ctx.EmitMovRegMem64(r56, fieldAddr)
			ctx.EmitMovRegMem64(r57, fieldAddr+8)
			ctx.EmitMovRegMem64(r58, fieldAddr+16)
			d141 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r56, Reg2: r57, Reg3: r58}
			ctx.BindReg(r56, &d141)
			ctx.BindReg(r57, &d141)
			ctx.BindReg(r58, &d141)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 0)
			r59 := ctx.AllocReg()
			r60 := ctx.AllocRegExcept(r59)
			r61 := ctx.AllocRegExcept(r59, r60)
			ctx.EmitMovRegMem(r59, thisptr.Reg, off)
			ctx.EmitMovRegMem(r60, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r61, thisptr.Reg, off+16)
			d141 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r59, Reg2: r60, Reg3: r61}
			ctx.BindReg(r59, &d141)
			ctx.BindReg(r60, &d141)
			ctx.BindReg(r61, &d141)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d140)
		var d142 scm.JITValueDesc
		if d140.Loc == scm.LocImm {
			d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d140.Imm.Int() / 64)}
		} else {
			r62 := ctx.AllocRegExcept(d140.Reg)
			ctx.EmitMovRegReg(r62, d140.Reg)
			ctx.EmitShrRegImm8(r62, 6)
			d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r62}
			ctx.BindReg(r62, &d142)
		}
		if d142.Loc == scm.LocReg && d140.Loc == scm.LocReg && d142.Reg == d140.Reg {
			ctx.TransferReg(d140.Reg)
			d140.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d142)
		ctx.ReclaimUntrackedRegs()
		d144 = ctx.EmitSliceElementAddress(&d141, &d142, 8)
		ctx.EnsureDesc(&d144)
		ctx.EmitMovRegMem(d144.Reg, d144.Reg, 0)
		d143 = d144
		ctx.FreeDesc(&d142)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d140)
		var d145 scm.JITValueDesc
		if d140.Loc == scm.LocImm {
			d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d140.Imm.Int() % 64)}
		} else {
			r63 := ctx.AllocRegExcept(d140.Reg)
			ctx.EmitMovRegReg(r63, d140.Reg)
			ctx.EmitAndRegImm32(r63, 63)
			d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r63}
			ctx.BindReg(r63, &d145)
		}
		if d145.Loc == scm.LocReg && d140.Loc == scm.LocReg && d145.Reg == d140.Reg {
			ctx.TransferReg(d140.Reg)
			d140.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d143)
		ctx.EnsureDesc(&d145)
		var d146 scm.JITValueDesc
		if d143.Loc == scm.LocImm && d145.Loc == scm.LocImm {
			d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d143.Imm.Int()) << uint64(d145.Imm.Int())))}
		} else if d145.Loc == scm.LocImm {
			r64 := ctx.AllocRegExcept(d143.Reg)
			ctx.EmitMovRegReg(r64, d143.Reg)
			ctx.EmitShlRegImm8(r64, uint8(d145.Imm.Int()))
			d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r64}
			ctx.BindReg(r64, &d146)
		} else {
			{
				shiftSrc := d143.Reg
				r65 := ctx.AllocRegExcept(d143.Reg)
				ctx.EmitMovRegReg(r65, d143.Reg)
				shiftSrc = r65
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d145.Reg != scm.RegRCX
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
		if d146.Loc == scm.LocReg && d143.Loc == scm.LocReg && d146.Reg == d143.Reg {
			ctx.TransferReg(d143.Reg)
			d143.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d146)
		ctx.FreeDesc(&d143)
		ctx.FreeDesc(&d145)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d140)
		var d147 scm.JITValueDesc
		if d140.Loc == scm.LocImm {
			d147 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d140.Imm.Int() % 64)}
		} else {
			r66 := ctx.AllocRegExcept(d140.Reg)
			ctx.EmitMovRegReg(r66, d140.Reg)
			ctx.EmitAndRegImm32(r66, 63)
			d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
			ctx.BindReg(r66, &d147)
		}
		if d147.Loc == scm.LocReg && d140.Loc == scm.LocReg && d147.Reg == d140.Reg {
			ctx.TransferReg(d140.Reg)
			d140.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d138)
		ctx.EnsureDesc(&d138)
		var d148 scm.JITValueDesc
		if d138.Loc == scm.LocImm {
			d148 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d138.Imm.Int()))))}
		} else {
			r67 := ctx.AllocReg()
			ctx.EmitMovRegReg(r67, d138.Reg)
			ctx.EmitShlRegImm8(r67, 56)
			ctx.EmitShrRegImm8(r67, 56)
			d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r67}
			ctx.BindReg(r67, &d148)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d147)
		ctx.EnsureDesc(&d148)
		ctx.EnsureDescsTogether(&d147, &d148)
		var d149 scm.JITValueDesc
		if d147.Loc == scm.LocImm && d148.Loc == scm.LocImm {
			d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d147.Imm.Int() + d148.Imm.Int())}
		} else if d148.Loc == scm.LocImm && d148.Imm.Int() == 0 {
			r68 := ctx.AllocRegExcept(d147.Reg)
			ctx.EmitMovRegReg(r68, d147.Reg)
			d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r68}
			ctx.BindReg(r68, &d149)
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
			r69 := ctx.AllocRegExcept(d147.Reg, d148.Reg)
			ctx.EmitMovRegReg(r69, d147.Reg)
			ctx.EmitAddInt64(r69, d148.Reg)
			d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r69}
			ctx.BindReg(r69, &d149)
		}
		if d149.Loc == scm.LocReg && d147.Loc == scm.LocReg && d149.Reg == d147.Reg {
			ctx.TransferReg(d147.Reg)
			d147.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d147)
		ctx.FreeDesc(&d148)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d149)
		var d150 scm.JITValueDesc
		if d149.Loc == scm.LocImm {
			d150 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d149.Imm.Int()) > uint64(0x40))}
		} else {
			r70 := ctx.AllocRegExcept(d149.Reg)
			ctx.EmitCmpRegImm32(d149.Reg, 64)
			ctx.EmitSetcc(r70, scm.CondUnsignedAbove)
			d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r70}
			ctx.BindReg(r70, &d150)
		}
		ctx.FreeDesc(&d149)
		ctx.ReclaimUntrackedRegs()
		d151 = d150
		ctx.EnsureDesc(&d151)
		if d151.Loc != scm.LocImm && d151.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		lbl27 := ctx.ReserveLabel()
		lbl28 := ctx.ReserveLabel()
		if d151.Loc == scm.LocImm {
			if d151.Imm.Bool() {
				ctx.MarkLabel(lbl27)
				ctx.EmitJmp(lbl25)
			} else {
				ctx.MarkLabel(lbl28)
				ctx.SyncDesc(&d146)
				if d146.Loc == scm.LocReg {
					ctx.ProtectReg(d146.Reg)
				} else if d146.Loc == scm.LocRegPair {
					ctx.ProtectReg(d146.Reg)
					ctx.ProtectReg(d146.Reg2)
				}
				d152 = d146
				if d152.Loc == scm.LocNone {
					panic("jit: phi source has no location")
				}
				ctx.EnsureDesc(&d152)
				ctx.EmitStoreToStack(d152, int32(phiBase135)+int32(0))
				if d146.Loc == scm.LocReg {
					ctx.UnprotectReg(d146.Reg)
				} else if d146.Loc == scm.LocRegPair {
					ctx.UnprotectReg(d146.Reg)
					ctx.UnprotectReg(d146.Reg2)
				}
				ctx.EmitJmp(lbl26)
			}
		} else {
			ctx.EmitCmpRegImm32(d151.Reg, 0)
			ctx.EmitJump(scm.CondNotEqual, lbl27)
			ctx.EmitJmp(lbl28)
			ctx.MarkLabel(lbl27)
			ctx.EmitJmp(lbl25)
			ctx.MarkLabel(lbl28)
			ctx.SyncDesc(&d146)
			if d146.Loc == scm.LocReg {
				ctx.ProtectReg(d146.Reg)
			} else if d146.Loc == scm.LocRegPair {
				ctx.ProtectReg(d146.Reg)
				ctx.ProtectReg(d146.Reg2)
			}
			d153 = d146
			if d153.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d153)
			ctx.EmitStoreToStack(d153, int32(phiBase135)+int32(0))
			if d146.Loc == scm.LocReg {
				ctx.UnprotectReg(d146.Reg)
			} else if d146.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d146.Reg)
				ctx.UnprotectReg(d146.Reg2)
			}
			ctx.EmitJmp(lbl26)
		}
		ctx.FreeDesc(&d150)
		bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl26)
		ctx.ResolveFixups()
		d136 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d138)
		ctx.EnsureDesc(&d138)
		var d154 scm.JITValueDesc
		if d138.Loc == scm.LocImm {
			d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d138.Imm.Int()))))}
		} else {
			r71 := ctx.AllocReg()
			ctx.EmitMovRegReg(r71, d138.Reg)
			ctx.EmitShlRegImm8(r71, 56)
			ctx.EmitShrRegImm8(r71, 56)
			d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r71}
			ctx.BindReg(r71, &d154)
		}
		ctx.ReclaimUntrackedRegs()
		d155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d154)
		ctx.EnsureDescsTogether(&d155, &d154)
		var d156 scm.JITValueDesc
		if d155.Loc == scm.LocImm && d154.Loc == scm.LocImm {
			d156 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d155.Imm.Int() - d154.Imm.Int())}
		} else if d154.Loc == scm.LocImm && d154.Imm.Int() == 0 {
			r72 := ctx.AllocRegExcept(d155.Reg)
			ctx.EmitMovRegReg(r72, d155.Reg)
			d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r72}
			ctx.BindReg(r72, &d156)
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
			r73 := ctx.AllocRegExcept(d155.Reg, d154.Reg)
			ctx.EmitMovRegReg(r73, d155.Reg)
			ctx.EmitSubInt64(r73, d154.Reg)
			d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
			ctx.BindReg(r73, &d156)
		}
		if d156.Loc == scm.LocReg && d155.Loc == scm.LocReg && d156.Reg == d155.Reg {
			ctx.TransferReg(d155.Reg)
			d155.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d154)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d136)
		ctx.EnsureDesc(&d156)
		var d157 scm.JITValueDesc
		if d136.Loc == scm.LocImm && d156.Loc == scm.LocImm {
			d157 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d136.Imm.Int()) >> uint64(d156.Imm.Int())))}
		} else if d156.Loc == scm.LocImm {
			r74 := ctx.AllocRegExcept(d136.Reg)
			ctx.EmitMovRegReg(r74, d136.Reg)
			ctx.EmitShrRegImm8(r74, uint8(d156.Imm.Int()))
			d157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r74}
			ctx.BindReg(r74, &d157)
		} else {
			{
				shiftSrc := d136.Reg
				r75 := ctx.AllocRegExcept(d136.Reg)
				ctx.EmitMovRegReg(r75, d136.Reg)
				shiftSrc = r75
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d156.Reg != scm.RegRCX
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
		if d157.Loc == scm.LocReg && d136.Loc == scm.LocReg && d157.Reg == d136.Reg {
			ctx.TransferReg(d136.Reg)
			d136.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d136)
		ctx.FreeDesc(&d156)
		ctx.ReclaimUntrackedRegs()
		r76 := ctx.AllocReg()
		ctx.EnsureDesc(&d157)
		ctx.EnsureDesc(&d157)
		if d157.Loc == scm.LocRegPair {
			panic("jit: scalar inline return has scm.LocRegPair")
		} else {
			ctx.EmitMovToReg(r76, d157)
		}
		ctx.EmitJmp(lbl23)
		bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl25)
		ctx.ResolveFixups()
		d136 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d140)
		var d158 scm.JITValueDesc
		if d140.Loc == scm.LocImm {
			d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d140.Imm.Int() / 64)}
		} else {
			r77 := ctx.AllocRegExcept(d140.Reg)
			ctx.EmitMovRegReg(r77, d140.Reg)
			ctx.EmitShrRegImm8(r77, 6)
			d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r77}
			ctx.BindReg(r77, &d158)
		}
		if d158.Loc == scm.LocReg && d140.Loc == scm.LocReg && d158.Reg == d140.Reg {
			ctx.TransferReg(d140.Reg)
			d140.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d159)
		ctx.ReclaimUntrackedRegs()
		d161 = ctx.EmitSliceElementAddress(&d141, &d159, 8)
		ctx.EnsureDesc(&d161)
		ctx.EmitMovRegMem(d161.Reg, d161.Reg, 0)
		d160 = d161
		ctx.FreeDesc(&d159)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d140)
		var d162 scm.JITValueDesc
		if d140.Loc == scm.LocImm {
			d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d140.Imm.Int() % 64)}
		} else {
			r78 := ctx.AllocRegExcept(d140.Reg)
			ctx.EmitMovRegReg(r78, d140.Reg)
			ctx.EmitAndRegImm32(r78, 63)
			d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r78}
			ctx.BindReg(r78, &d162)
		}
		if d162.Loc == scm.LocReg && d140.Loc == scm.LocReg && d162.Reg == d140.Reg {
			ctx.TransferReg(d140.Reg)
			d140.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d140)
		ctx.ReclaimUntrackedRegs()
		d163 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d162)
		ctx.EnsureDescsTogether(&d163, &d162)
		var d164 scm.JITValueDesc
		if d163.Loc == scm.LocImm && d162.Loc == scm.LocImm {
			d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d163.Imm.Int() - d162.Imm.Int())}
		} else if d162.Loc == scm.LocImm && d162.Imm.Int() == 0 {
			r79 := ctx.AllocRegExcept(d163.Reg)
			ctx.EmitMovRegReg(r79, d163.Reg)
			d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r79}
			ctx.BindReg(r79, &d164)
		} else if d163.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d162.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d163.Imm.Int()))
			ctx.EmitSubInt64(scratch, d162.Reg)
			d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d164)
		} else if d162.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d163.Reg)
			ctx.EmitMovRegReg(scratch, d163.Reg)
			if d162.Imm.Int() >= -2147483648 && d162.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d162.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d162.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d164)
		} else {
			r80 := ctx.AllocRegExcept(d163.Reg, d162.Reg)
			ctx.EmitMovRegReg(r80, d163.Reg)
			ctx.EmitSubInt64(r80, d162.Reg)
			d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
			ctx.BindReg(r80, &d164)
		}
		if d164.Loc == scm.LocReg && d163.Loc == scm.LocReg && d164.Reg == d163.Reg {
			ctx.TransferReg(d163.Reg)
			d163.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d162)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d160)
		ctx.EnsureDesc(&d164)
		var d165 scm.JITValueDesc
		if d160.Loc == scm.LocImm && d164.Loc == scm.LocImm {
			d165 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d160.Imm.Int()) >> uint64(d164.Imm.Int())))}
		} else if d164.Loc == scm.LocImm {
			r81 := ctx.AllocRegExcept(d160.Reg)
			ctx.EmitMovRegReg(r81, d160.Reg)
			ctx.EmitShrRegImm8(r81, uint8(d164.Imm.Int()))
			d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
			ctx.BindReg(r81, &d165)
		} else {
			{
				shiftSrc := d160.Reg
				r82 := ctx.AllocRegExcept(d160.Reg)
				ctx.EmitMovRegReg(r82, d160.Reg)
				shiftSrc = r82
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d164.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d164.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d164.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d165)
			}
		}
		if d165.Loc == scm.LocReg && d160.Loc == scm.LocReg && d165.Reg == d160.Reg {
			ctx.TransferReg(d160.Reg)
			d160.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d160)
		ctx.FreeDesc(&d164)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d146)
		ctx.EnsureDesc(&d165)
		var d166 scm.JITValueDesc
		if d146.Loc == scm.LocImm && d165.Loc == scm.LocImm {
			d166 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d146.Imm.Int() | d165.Imm.Int())}
		} else if d146.Loc == scm.LocImm && d146.Imm.Int() == 0 {
			d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d165.Reg}
			ctx.BindReg(d165.Reg, &d166)
		} else if d165.Loc == scm.LocImm && d165.Imm.Int() == 0 {
			r83 := ctx.AllocRegExcept(d146.Reg)
			ctx.EmitMovRegReg(r83, d146.Reg)
			d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
			ctx.BindReg(r83, &d166)
		} else if d146.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d165.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d146.Imm.Int()))
			ctx.EmitOrInt64(scratch, d165.Reg)
			d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d166)
		} else if d165.Loc == scm.LocImm {
			r84 := ctx.AllocRegExcept(d146.Reg)
			ctx.EmitMovRegReg(r84, d146.Reg)
			if d165.Imm.Int() >= -2147483648 && d165.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r84, int32(d165.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d165.Imm.Int()))
				ctx.EmitOrInt64(r84, scm.RegR11)
			}
			d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r84}
			ctx.BindReg(r84, &d166)
		} else {
			r85 := ctx.AllocRegExcept(d146.Reg, d165.Reg)
			ctx.EmitMovRegReg(r85, d146.Reg)
			ctx.EmitOrInt64(r85, d165.Reg)
			d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
			ctx.BindReg(r85, &d166)
		}
		if d166.Loc == scm.LocReg && d146.Loc == scm.LocReg && d166.Reg == d146.Reg {
			ctx.TransferReg(d146.Reg)
			d146.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d166)
		ctx.EmitStoreToStack(d166, int32(phiBase135)+int32(0))
		ctx.StabilizeDescForControlFlow(&d166)
		ctx.FreeDesc(&d165)
		ctx.ReclaimUntrackedRegs()
		ctx.EmitJmp(lbl26)
		ctx.MarkLabel(lbl23)
		d167 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r76}
		ctx.BindReg(r76, &d167)
		ctx.BindReg(r76, &d167)
		ctx.EnsureDesc(&d167)
		ctx.EnsureDesc(&d167)
		var d168 scm.JITValueDesc
		if d167.Loc == scm.LocImm {
			d168 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d167.Imm.Int()))))}
		} else {
			r86 := ctx.AllocReg()
			ctx.EmitMovRegReg(r86, d167.Reg)
			d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
			ctx.BindReg(r86, &d168)
		}
		ctx.FreeDesc(&d167)
		var d169 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 32
			r87 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r87, fieldAddr)
			d169 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r87}
			ctx.BindReg(r87, &d169)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 32)
			r88 := ctx.AllocReg()
			ctx.EmitMovRegMem(r88, thisptr.Reg, off)
			d169 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r88}
			ctx.BindReg(r88, &d169)
		}
		ctx.EnsureDesc(&d168)
		ctx.EnsureDesc(&d169)
		ctx.EnsureDescsTogether(&d168, &d169)
		var d170 scm.JITValueDesc
		if d168.Loc == scm.LocImm && d169.Loc == scm.LocImm {
			d170 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d168.Imm.Int() + d169.Imm.Int())}
		} else if d169.Loc == scm.LocImm && d169.Imm.Int() == 0 {
			r89 := ctx.AllocRegExcept(d168.Reg)
			ctx.EmitMovRegReg(r89, d168.Reg)
			d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
			ctx.BindReg(r89, &d170)
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
			r90 := ctx.AllocRegExcept(d168.Reg, d169.Reg)
			ctx.EmitMovRegReg(r90, d168.Reg)
			ctx.EmitAddInt64(r90, d169.Reg)
			d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
			ctx.BindReg(r90, &d170)
		}
		if d170.Loc == scm.LocReg && d168.Loc == scm.LocReg && d170.Reg == d168.Reg {
			ctx.TransferReg(d168.Reg)
			d168.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d170)
		ctx.FreeDesc(&d168)
		var d171 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 56
			r91 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r91, fieldAddr)
			d171 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r91}
			ctx.BindReg(r91, &d171)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 56)
			r92 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r92, thisptr.Reg, off)
			d171 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r92}
			ctx.BindReg(r92, &d171)
		}
		d172 = d171
		ctx.EnsureDesc(&d172)
		if d172.Loc != scm.LocImm && d172.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
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
				ps173.OverlayValues[17] = d17
				ps173.OverlayValues[18] = d18
				ps173.OverlayValues[19] = d19
				ps173.OverlayValues[20] = d20
				ps173.OverlayValues[21] = d21
				ps173.OverlayValues[22] = d22
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
				ps173.OverlayValues[132] = d132
				ps173.OverlayValues[133] = d133
				ps173.OverlayValues[134] = d134
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
			if ps.General {
			}
			ps174 := scm.PhiState{General: ps.General}
			ps174.OverlayValues = make([]scm.JITValueDesc, 173)
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
			ps174.OverlayValues[17] = d17
			ps174.OverlayValues[18] = d18
			ps174.OverlayValues[19] = d19
			ps174.OverlayValues[20] = d20
			ps174.OverlayValues[21] = d21
			ps174.OverlayValues[22] = d22
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
			ps174.OverlayValues[132] = d132
			ps174.OverlayValues[133] = d133
			ps174.OverlayValues[134] = d134
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
		lbl29 := ctx.ReserveLabel()
		lbl30 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d172.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl29)
		ctx.EmitJmp(lbl30)
		ctx.MarkLabel(lbl29)
		ctx.EmitJmp(lbl14)
		ctx.MarkLabel(lbl30)
		ctx.EmitJmp(lbl13)
		ps176 := scm.PhiState{General: true}
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
		ps176.OverlayValues[17] = d17
		ps176.OverlayValues[18] = d18
		ps176.OverlayValues[19] = d19
		ps176.OverlayValues[20] = d20
		ps176.OverlayValues[21] = d21
		ps176.OverlayValues[22] = d22
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
		ps176.OverlayValues[132] = d132
		ps176.OverlayValues[133] = d133
		ps176.OverlayValues[134] = d134
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
		ps177.OverlayValues[17] = d17
		ps177.OverlayValues[18] = d18
		ps177.OverlayValues[19] = d19
		ps177.OverlayValues[20] = d20
		ps177.OverlayValues[21] = d21
		ps177.OverlayValues[22] = d22
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
		ps177.OverlayValues[132] = d132
		ps177.OverlayValues[133] = d133
		ps177.OverlayValues[134] = d134
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
		snap178 := d1
		snap179 := d2
		snap180 := d3
		snap181 := d4
		snap182 := d5
		snap183 := d6
		snap184 := d7
		snap185 := d8
		snap186 := d9
		snap187 := d10
		snap188 := d11
		snap189 := d12
		snap190 := d13
		snap191 := d14
		snap192 := d15
		snap193 := d17
		snap194 := d18
		snap195 := d19
		snap196 := d20
		snap197 := d21
		snap198 := d22
		snap199 := d24
		snap200 := d25
		snap201 := d26
		snap202 := d27
		snap203 := d28
		snap204 := d29
		snap205 := d30
		snap206 := d31
		snap207 := d32
		snap208 := d33
		snap209 := d34
		snap210 := d35
		snap211 := d36
		snap212 := d37
		snap213 := d38
		snap214 := d39
		snap215 := d40
		snap216 := d41
		snap217 := d42
		snap218 := d43
		snap219 := d44
		snap220 := d45
		snap221 := d46
		snap222 := d47
		snap223 := d48
		snap224 := d49
		snap225 := d50
		snap226 := d51
		snap227 := d52
		snap228 := d53
		snap229 := d54
		snap230 := d55
		snap231 := d56
		snap232 := d57
		snap233 := d58
		snap234 := d59
		snap235 := d60
		snap236 := d61
		snap237 := d64
		snap238 := d65
		snap239 := d66
		snap240 := d132
		snap241 := d133
		snap242 := d134
		snap243 := d136
		snap244 := d137
		snap245 := d138
		snap246 := d139
		snap247 := d140
		snap248 := d141
		snap249 := d142
		snap250 := d143
		snap251 := d144
		snap252 := d145
		snap253 := d146
		snap254 := d147
		snap255 := d148
		snap256 := d149
		snap257 := d150
		snap258 := d151
		snap259 := d152
		snap260 := d153
		snap261 := d154
		snap262 := d155
		snap263 := d156
		snap264 := d157
		snap265 := d158
		snap266 := d159
		snap267 := d160
		snap268 := d161
		snap269 := d162
		snap270 := d163
		snap271 := d164
		snap272 := d165
		snap273 := d166
		snap274 := d167
		snap275 := d168
		snap276 := d169
		snap277 := d170
		snap278 := d171
		snap279 := d172
		snap280 := d175
		alloc281 := ctx.SnapshotAllocState()
		if !bbs[12].Rendered {
			bbs[12].RenderPS(ps177)
		}
		ctx.RestoreAllocState(alloc281)
		d1 = snap178
		d2 = snap179
		d3 = snap180
		d4 = snap181
		d5 = snap182
		d6 = snap183
		d7 = snap184
		d8 = snap185
		d9 = snap186
		d10 = snap187
		d11 = snap188
		d12 = snap189
		d13 = snap190
		d14 = snap191
		d15 = snap192
		d17 = snap193
		d18 = snap194
		d19 = snap195
		d20 = snap196
		d21 = snap197
		d22 = snap198
		d24 = snap199
		d25 = snap200
		d26 = snap201
		d27 = snap202
		d28 = snap203
		d29 = snap204
		d30 = snap205
		d31 = snap206
		d32 = snap207
		d33 = snap208
		d34 = snap209
		d35 = snap210
		d36 = snap211
		d37 = snap212
		d38 = snap213
		d39 = snap214
		d40 = snap215
		d41 = snap216
		d42 = snap217
		d43 = snap218
		d44 = snap219
		d45 = snap220
		d46 = snap221
		d47 = snap222
		d48 = snap223
		d49 = snap224
		d50 = snap225
		d51 = snap226
		d52 = snap227
		d53 = snap228
		d54 = snap229
		d55 = snap230
		d56 = snap231
		d57 = snap232
		d58 = snap233
		d59 = snap234
		d60 = snap235
		d61 = snap236
		d64 = snap237
		d65 = snap238
		d66 = snap239
		d132 = snap240
		d133 = snap241
		d134 = snap242
		d136 = snap243
		d137 = snap244
		d138 = snap245
		d139 = snap246
		d140 = snap247
		d141 = snap248
		d142 = snap249
		d143 = snap250
		d144 = snap251
		d145 = snap252
		d146 = snap253
		d147 = snap254
		d148 = snap255
		d149 = snap256
		d150 = snap257
		d151 = snap258
		d152 = snap259
		d153 = snap260
		d154 = snap261
		d155 = snap262
		d156 = snap263
		d157 = snap264
		d158 = snap265
		d159 = snap266
		d160 = snap267
		d161 = snap268
		d162 = snap269
		d163 = snap270
		d164 = snap271
		d165 = snap272
		d166 = snap273
		d167 = snap274
		d168 = snap275
		d169 = snap276
		d170 = snap277
		d171 = snap278
		d172 = snap279
		d175 = snap280
		if !bbs[13].Rendered {
			return bbs[13].RenderPS(ps176)
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
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
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
		ctx.EnsureDesc(&d1)
		ctx.EnsureDesc(&d1)
		var d282 scm.JITValueDesc
		if d1.Loc == scm.LocImm {
			d282 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegReg(scratch, d1.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d282 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d282)
		}
		if d282.Loc == scm.LocImm {
			d282 = scm.JITValueDesc{Loc: scm.LocImm, Type: d282.Type, Imm: scm.NewInt(int64(uint64(d282.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d282.Reg, 32)
			ctx.EmitShrRegImm8(d282.Reg, 32)
		}
		if d282.Loc == scm.LocReg && d1.Loc == scm.LocReg && d282.Reg == d1.Reg {
			ctx.TransferReg(d1.Reg)
			d1.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d282)
		ctx.EmitStoreToStack(d282, int32(bbs[4].PhiBase)+int32(32))
		ctx.StabilizeDescForControlFlow(&d282)
		ctx.EnsureDesc(&d1)
		ctx.EnsureDesc(&d1)
		var d283 scm.JITValueDesc
		if d1.Loc == scm.LocImm {
			d283 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegReg(scratch, d1.Reg)
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
		if d283.Loc == scm.LocReg && d1.Loc == scm.LocReg && d283.Reg == d1.Reg {
			ctx.TransferReg(d1.Reg)
			d1.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d283)
		ctx.EmitStoreToStack(d283, int32(bbs[4].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d283)
		if ps.General {
			ctx.SyncDesc(&d2)
			if d2.Loc == scm.LocReg {
				ctx.ProtectReg(d2.Reg)
			} else if d2.Loc == scm.LocRegPair {
				ctx.ProtectReg(d2.Reg)
				ctx.ProtectReg(d2.Reg2)
			}
			d284 = d2
			if d284.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d284)
			d285 = d284
			if d285.Loc == scm.LocImm {
				d285 = scm.JITValueDesc{Loc: scm.LocImm, Type: d285.Type, Imm: scm.NewInt(int64(uint64(d285.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d285.Reg, 32)
				ctx.EmitShrRegImm8(d285.Reg, 32)
			}
			ctx.EmitStoreToStack(d285, int32(bbs[4].PhiBase)+int32(16))
			if d2.Loc == scm.LocReg {
				ctx.UnprotectReg(d2.Reg)
			} else if d2.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d2.Reg)
				ctx.UnprotectReg(d2.Reg2)
			}
		}
		ps286 := scm.PhiState{General: ps.General}
		ps286.OverlayValues = make([]scm.JITValueDesc, 286)
		ps286.OverlayValues[1] = d1
		ps286.OverlayValues[2] = d2
		ps286.OverlayValues[3] = d3
		ps286.OverlayValues[4] = d4
		ps286.OverlayValues[5] = d5
		ps286.OverlayValues[6] = d6
		ps286.OverlayValues[7] = d7
		ps286.OverlayValues[8] = d8
		ps286.OverlayValues[9] = d9
		ps286.OverlayValues[10] = d10
		ps286.OverlayValues[11] = d11
		ps286.OverlayValues[12] = d12
		ps286.OverlayValues[13] = d13
		ps286.OverlayValues[14] = d14
		ps286.OverlayValues[15] = d15
		ps286.OverlayValues[17] = d17
		ps286.OverlayValues[18] = d18
		ps286.OverlayValues[19] = d19
		ps286.OverlayValues[20] = d20
		ps286.OverlayValues[21] = d21
		ps286.OverlayValues[22] = d22
		ps286.OverlayValues[24] = d24
		ps286.OverlayValues[25] = d25
		ps286.OverlayValues[26] = d26
		ps286.OverlayValues[27] = d27
		ps286.OverlayValues[28] = d28
		ps286.OverlayValues[29] = d29
		ps286.OverlayValues[30] = d30
		ps286.OverlayValues[31] = d31
		ps286.OverlayValues[32] = d32
		ps286.OverlayValues[33] = d33
		ps286.OverlayValues[34] = d34
		ps286.OverlayValues[35] = d35
		ps286.OverlayValues[36] = d36
		ps286.OverlayValues[37] = d37
		ps286.OverlayValues[38] = d38
		ps286.OverlayValues[39] = d39
		ps286.OverlayValues[40] = d40
		ps286.OverlayValues[41] = d41
		ps286.OverlayValues[42] = d42
		ps286.OverlayValues[43] = d43
		ps286.OverlayValues[44] = d44
		ps286.OverlayValues[45] = d45
		ps286.OverlayValues[46] = d46
		ps286.OverlayValues[47] = d47
		ps286.OverlayValues[48] = d48
		ps286.OverlayValues[49] = d49
		ps286.OverlayValues[50] = d50
		ps286.OverlayValues[51] = d51
		ps286.OverlayValues[52] = d52
		ps286.OverlayValues[53] = d53
		ps286.OverlayValues[54] = d54
		ps286.OverlayValues[55] = d55
		ps286.OverlayValues[56] = d56
		ps286.OverlayValues[57] = d57
		ps286.OverlayValues[58] = d58
		ps286.OverlayValues[59] = d59
		ps286.OverlayValues[60] = d60
		ps286.OverlayValues[61] = d61
		ps286.OverlayValues[64] = d64
		ps286.OverlayValues[65] = d65
		ps286.OverlayValues[66] = d66
		ps286.OverlayValues[132] = d132
		ps286.OverlayValues[133] = d133
		ps286.OverlayValues[134] = d134
		ps286.OverlayValues[136] = d136
		ps286.OverlayValues[137] = d137
		ps286.OverlayValues[138] = d138
		ps286.OverlayValues[139] = d139
		ps286.OverlayValues[140] = d140
		ps286.OverlayValues[141] = d141
		ps286.OverlayValues[142] = d142
		ps286.OverlayValues[143] = d143
		ps286.OverlayValues[144] = d144
		ps286.OverlayValues[145] = d145
		ps286.OverlayValues[146] = d146
		ps286.OverlayValues[147] = d147
		ps286.OverlayValues[148] = d148
		ps286.OverlayValues[149] = d149
		ps286.OverlayValues[150] = d150
		ps286.OverlayValues[151] = d151
		ps286.OverlayValues[152] = d152
		ps286.OverlayValues[153] = d153
		ps286.OverlayValues[154] = d154
		ps286.OverlayValues[155] = d155
		ps286.OverlayValues[156] = d156
		ps286.OverlayValues[157] = d157
		ps286.OverlayValues[158] = d158
		ps286.OverlayValues[159] = d159
		ps286.OverlayValues[160] = d160
		ps286.OverlayValues[161] = d161
		ps286.OverlayValues[162] = d162
		ps286.OverlayValues[163] = d163
		ps286.OverlayValues[164] = d164
		ps286.OverlayValues[165] = d165
		ps286.OverlayValues[166] = d166
		ps286.OverlayValues[167] = d167
		ps286.OverlayValues[168] = d168
		ps286.OverlayValues[169] = d169
		ps286.OverlayValues[170] = d170
		ps286.OverlayValues[171] = d171
		ps286.OverlayValues[172] = d172
		ps286.OverlayValues[175] = d175
		ps286.OverlayValues[282] = d282
		ps286.OverlayValues[283] = d283
		ps286.OverlayValues[284] = d284
		ps286.OverlayValues[285] = d285
		ps286.PhiValues = make([]scm.JITValueDesc, 3)
		d287 = d2
		ps286.PhiValues[1] = d287
		if ps286.General && bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
			return result
		}
		return bbs[4].RenderPS(ps286)
		return result
	}
	bbs[4].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
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
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
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
		if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != scm.LocNone {
			d282 = ps.OverlayValues[282]
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
		var d291 scm.JITValueDesc
		if d6.Loc == scm.LocImm && d7.Loc == scm.LocImm {
			d291 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d6.Imm.Int()) == uint64(d7.Imm.Int()))}
		} else if d7.Loc == scm.LocImm {
			r93 := ctx.AllocRegExcept(d6.Reg)
			if d7.Imm.Int() >= -2147483648 && d7.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d6.Reg, int32(d7.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d7.Imm.Int()))
				ctx.EmitCmpInt64(d6.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r93, scm.CondEqual)
			d291 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r93}
			ctx.BindReg(r93, &d291)
		} else if d6.Loc == scm.LocImm {
			r94 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d6.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d7.Reg)
			ctx.EmitSetcc(r94, scm.CondEqual)
			d291 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r94}
			ctx.BindReg(r94, &d291)
		} else {
			r95 := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitCmpInt64(d6.Reg, d7.Reg)
			ctx.EmitSetcc(r95, scm.CondEqual)
			d291 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r95}
			ctx.BindReg(r95, &d291)
		}
		d292 = d291
		ctx.EnsureDesc(&d292)
		if d292.Loc != scm.LocImm && d292.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d292.Loc == scm.LocImm {
			if d292.Imm.Bool() {
				if ps.General {
					ctx.SyncDesc(&d6)
					if d6.Loc == scm.LocReg {
						ctx.ProtectReg(d6.Reg)
					} else if d6.Loc == scm.LocRegPair {
						ctx.ProtectReg(d6.Reg)
						ctx.ProtectReg(d6.Reg2)
					}
					d293 = d6
					if d293.Loc == scm.LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d293)
					d294 = d293
					if d294.Loc == scm.LocImm {
						d294 = scm.JITValueDesc{Loc: scm.LocImm, Type: d294.Type, Imm: scm.NewInt(int64(uint64(d294.Imm.Int()) & 0xffffffff))}
					} else {
						ctx.EmitShlRegImm8(d294.Reg, 32)
						ctx.EmitShrRegImm8(d294.Reg, 32)
					}
					ctx.EmitStoreToStack(d294, int32(bbs[2].PhiBase)+int32(0))
					if d6.Loc == scm.LocReg {
						ctx.UnprotectReg(d6.Reg)
					} else if d6.Loc == scm.LocRegPair {
						ctx.UnprotectReg(d6.Reg)
						ctx.UnprotectReg(d6.Reg2)
					}
				}
				ps295 := scm.PhiState{General: ps.General}
				ps295.OverlayValues = make([]scm.JITValueDesc, 295)
				ps295.OverlayValues[1] = d1
				ps295.OverlayValues[2] = d2
				ps295.OverlayValues[3] = d3
				ps295.OverlayValues[4] = d4
				ps295.OverlayValues[5] = d5
				ps295.OverlayValues[6] = d6
				ps295.OverlayValues[7] = d7
				ps295.OverlayValues[8] = d8
				ps295.OverlayValues[9] = d9
				ps295.OverlayValues[10] = d10
				ps295.OverlayValues[11] = d11
				ps295.OverlayValues[12] = d12
				ps295.OverlayValues[13] = d13
				ps295.OverlayValues[14] = d14
				ps295.OverlayValues[15] = d15
				ps295.OverlayValues[17] = d17
				ps295.OverlayValues[18] = d18
				ps295.OverlayValues[19] = d19
				ps295.OverlayValues[20] = d20
				ps295.OverlayValues[21] = d21
				ps295.OverlayValues[22] = d22
				ps295.OverlayValues[24] = d24
				ps295.OverlayValues[25] = d25
				ps295.OverlayValues[26] = d26
				ps295.OverlayValues[27] = d27
				ps295.OverlayValues[28] = d28
				ps295.OverlayValues[29] = d29
				ps295.OverlayValues[30] = d30
				ps295.OverlayValues[31] = d31
				ps295.OverlayValues[32] = d32
				ps295.OverlayValues[33] = d33
				ps295.OverlayValues[34] = d34
				ps295.OverlayValues[35] = d35
				ps295.OverlayValues[36] = d36
				ps295.OverlayValues[37] = d37
				ps295.OverlayValues[38] = d38
				ps295.OverlayValues[39] = d39
				ps295.OverlayValues[40] = d40
				ps295.OverlayValues[41] = d41
				ps295.OverlayValues[42] = d42
				ps295.OverlayValues[43] = d43
				ps295.OverlayValues[44] = d44
				ps295.OverlayValues[45] = d45
				ps295.OverlayValues[46] = d46
				ps295.OverlayValues[47] = d47
				ps295.OverlayValues[48] = d48
				ps295.OverlayValues[49] = d49
				ps295.OverlayValues[50] = d50
				ps295.OverlayValues[51] = d51
				ps295.OverlayValues[52] = d52
				ps295.OverlayValues[53] = d53
				ps295.OverlayValues[54] = d54
				ps295.OverlayValues[55] = d55
				ps295.OverlayValues[56] = d56
				ps295.OverlayValues[57] = d57
				ps295.OverlayValues[58] = d58
				ps295.OverlayValues[59] = d59
				ps295.OverlayValues[60] = d60
				ps295.OverlayValues[61] = d61
				ps295.OverlayValues[64] = d64
				ps295.OverlayValues[65] = d65
				ps295.OverlayValues[66] = d66
				ps295.OverlayValues[132] = d132
				ps295.OverlayValues[133] = d133
				ps295.OverlayValues[134] = d134
				ps295.OverlayValues[136] = d136
				ps295.OverlayValues[137] = d137
				ps295.OverlayValues[138] = d138
				ps295.OverlayValues[139] = d139
				ps295.OverlayValues[140] = d140
				ps295.OverlayValues[141] = d141
				ps295.OverlayValues[142] = d142
				ps295.OverlayValues[143] = d143
				ps295.OverlayValues[144] = d144
				ps295.OverlayValues[145] = d145
				ps295.OverlayValues[146] = d146
				ps295.OverlayValues[147] = d147
				ps295.OverlayValues[148] = d148
				ps295.OverlayValues[149] = d149
				ps295.OverlayValues[150] = d150
				ps295.OverlayValues[151] = d151
				ps295.OverlayValues[152] = d152
				ps295.OverlayValues[153] = d153
				ps295.OverlayValues[154] = d154
				ps295.OverlayValues[155] = d155
				ps295.OverlayValues[156] = d156
				ps295.OverlayValues[157] = d157
				ps295.OverlayValues[158] = d158
				ps295.OverlayValues[159] = d159
				ps295.OverlayValues[160] = d160
				ps295.OverlayValues[161] = d161
				ps295.OverlayValues[162] = d162
				ps295.OverlayValues[163] = d163
				ps295.OverlayValues[164] = d164
				ps295.OverlayValues[165] = d165
				ps295.OverlayValues[166] = d166
				ps295.OverlayValues[167] = d167
				ps295.OverlayValues[168] = d168
				ps295.OverlayValues[169] = d169
				ps295.OverlayValues[170] = d170
				ps295.OverlayValues[171] = d171
				ps295.OverlayValues[172] = d172
				ps295.OverlayValues[175] = d175
				ps295.OverlayValues[282] = d282
				ps295.OverlayValues[283] = d283
				ps295.OverlayValues[284] = d284
				ps295.OverlayValues[285] = d285
				ps295.OverlayValues[287] = d287
				ps295.OverlayValues[288] = d288
				ps295.OverlayValues[289] = d289
				ps295.OverlayValues[290] = d290
				ps295.OverlayValues[291] = d291
				ps295.OverlayValues[292] = d292
				ps295.OverlayValues[293] = d293
				ps295.OverlayValues[294] = d294
				ps295.PhiValues = make([]scm.JITValueDesc, 1)
				d296 = d6
				ps295.PhiValues[0] = d296
				return bbs[2].RenderPS(ps295)
			}
			if ps.General {
			}
			ps297 := scm.PhiState{General: ps.General}
			ps297.OverlayValues = make([]scm.JITValueDesc, 297)
			ps297.OverlayValues[1] = d1
			ps297.OverlayValues[2] = d2
			ps297.OverlayValues[3] = d3
			ps297.OverlayValues[4] = d4
			ps297.OverlayValues[5] = d5
			ps297.OverlayValues[6] = d6
			ps297.OverlayValues[7] = d7
			ps297.OverlayValues[8] = d8
			ps297.OverlayValues[9] = d9
			ps297.OverlayValues[10] = d10
			ps297.OverlayValues[11] = d11
			ps297.OverlayValues[12] = d12
			ps297.OverlayValues[13] = d13
			ps297.OverlayValues[14] = d14
			ps297.OverlayValues[15] = d15
			ps297.OverlayValues[17] = d17
			ps297.OverlayValues[18] = d18
			ps297.OverlayValues[19] = d19
			ps297.OverlayValues[20] = d20
			ps297.OverlayValues[21] = d21
			ps297.OverlayValues[22] = d22
			ps297.OverlayValues[24] = d24
			ps297.OverlayValues[25] = d25
			ps297.OverlayValues[26] = d26
			ps297.OverlayValues[27] = d27
			ps297.OverlayValues[28] = d28
			ps297.OverlayValues[29] = d29
			ps297.OverlayValues[30] = d30
			ps297.OverlayValues[31] = d31
			ps297.OverlayValues[32] = d32
			ps297.OverlayValues[33] = d33
			ps297.OverlayValues[34] = d34
			ps297.OverlayValues[35] = d35
			ps297.OverlayValues[36] = d36
			ps297.OverlayValues[37] = d37
			ps297.OverlayValues[38] = d38
			ps297.OverlayValues[39] = d39
			ps297.OverlayValues[40] = d40
			ps297.OverlayValues[41] = d41
			ps297.OverlayValues[42] = d42
			ps297.OverlayValues[43] = d43
			ps297.OverlayValues[44] = d44
			ps297.OverlayValues[45] = d45
			ps297.OverlayValues[46] = d46
			ps297.OverlayValues[47] = d47
			ps297.OverlayValues[48] = d48
			ps297.OverlayValues[49] = d49
			ps297.OverlayValues[50] = d50
			ps297.OverlayValues[51] = d51
			ps297.OverlayValues[52] = d52
			ps297.OverlayValues[53] = d53
			ps297.OverlayValues[54] = d54
			ps297.OverlayValues[55] = d55
			ps297.OverlayValues[56] = d56
			ps297.OverlayValues[57] = d57
			ps297.OverlayValues[58] = d58
			ps297.OverlayValues[59] = d59
			ps297.OverlayValues[60] = d60
			ps297.OverlayValues[61] = d61
			ps297.OverlayValues[64] = d64
			ps297.OverlayValues[65] = d65
			ps297.OverlayValues[66] = d66
			ps297.OverlayValues[132] = d132
			ps297.OverlayValues[133] = d133
			ps297.OverlayValues[134] = d134
			ps297.OverlayValues[136] = d136
			ps297.OverlayValues[137] = d137
			ps297.OverlayValues[138] = d138
			ps297.OverlayValues[139] = d139
			ps297.OverlayValues[140] = d140
			ps297.OverlayValues[141] = d141
			ps297.OverlayValues[142] = d142
			ps297.OverlayValues[143] = d143
			ps297.OverlayValues[144] = d144
			ps297.OverlayValues[145] = d145
			ps297.OverlayValues[146] = d146
			ps297.OverlayValues[147] = d147
			ps297.OverlayValues[148] = d148
			ps297.OverlayValues[149] = d149
			ps297.OverlayValues[150] = d150
			ps297.OverlayValues[151] = d151
			ps297.OverlayValues[152] = d152
			ps297.OverlayValues[153] = d153
			ps297.OverlayValues[154] = d154
			ps297.OverlayValues[155] = d155
			ps297.OverlayValues[156] = d156
			ps297.OverlayValues[157] = d157
			ps297.OverlayValues[158] = d158
			ps297.OverlayValues[159] = d159
			ps297.OverlayValues[160] = d160
			ps297.OverlayValues[161] = d161
			ps297.OverlayValues[162] = d162
			ps297.OverlayValues[163] = d163
			ps297.OverlayValues[164] = d164
			ps297.OverlayValues[165] = d165
			ps297.OverlayValues[166] = d166
			ps297.OverlayValues[167] = d167
			ps297.OverlayValues[168] = d168
			ps297.OverlayValues[169] = d169
			ps297.OverlayValues[170] = d170
			ps297.OverlayValues[171] = d171
			ps297.OverlayValues[172] = d172
			ps297.OverlayValues[175] = d175
			ps297.OverlayValues[282] = d282
			ps297.OverlayValues[283] = d283
			ps297.OverlayValues[284] = d284
			ps297.OverlayValues[285] = d285
			ps297.OverlayValues[287] = d287
			ps297.OverlayValues[288] = d288
			ps297.OverlayValues[289] = d289
			ps297.OverlayValues[290] = d290
			ps297.OverlayValues[291] = d291
			ps297.OverlayValues[292] = d292
			ps297.OverlayValues[293] = d293
			ps297.OverlayValues[294] = d294
			ps297.OverlayValues[296] = d296
			return bbs[6].RenderPS(ps297)
		}
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
			ps.General = true
			return bbs[4].RenderPS(ps)
		}
		lbl31 := ctx.ReserveLabel()
		lbl32 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d292.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl31)
		ctx.EmitJmp(lbl32)
		ctx.MarkLabel(lbl31)
		ctx.SyncDesc(&d6)
		if d6.Loc == scm.LocReg {
			ctx.ProtectReg(d6.Reg)
		} else if d6.Loc == scm.LocRegPair {
			ctx.ProtectReg(d6.Reg)
			ctx.ProtectReg(d6.Reg2)
		}
		d301 = d6
		if d301.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d301)
		d302 = d301
		if d302.Loc == scm.LocImm {
			d302 = scm.JITValueDesc{Loc: scm.LocImm, Type: d302.Type, Imm: scm.NewInt(int64(uint64(d302.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d302.Reg, 32)
			ctx.EmitShrRegImm8(d302.Reg, 32)
		}
		ctx.EmitStoreToStack(d302, int32(bbs[2].PhiBase)+int32(0))
		if d6.Loc == scm.LocReg {
			ctx.UnprotectReg(d6.Reg)
		} else if d6.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d6.Reg)
			ctx.UnprotectReg(d6.Reg2)
		}
		ctx.EmitJmp(lbl3)
		ctx.MarkLabel(lbl32)
		ctx.EmitJmp(lbl7)
		ps303 := scm.PhiState{General: true}
		ps303.OverlayValues = make([]scm.JITValueDesc, 303)
		ps303.OverlayValues[1] = d1
		ps303.OverlayValues[2] = d2
		ps303.OverlayValues[3] = d3
		ps303.OverlayValues[4] = d4
		ps303.OverlayValues[5] = d5
		ps303.OverlayValues[6] = d6
		ps303.OverlayValues[7] = d7
		ps303.OverlayValues[8] = d8
		ps303.OverlayValues[9] = d9
		ps303.OverlayValues[10] = d10
		ps303.OverlayValues[11] = d11
		ps303.OverlayValues[12] = d12
		ps303.OverlayValues[13] = d13
		ps303.OverlayValues[14] = d14
		ps303.OverlayValues[15] = d15
		ps303.OverlayValues[17] = d17
		ps303.OverlayValues[18] = d18
		ps303.OverlayValues[19] = d19
		ps303.OverlayValues[20] = d20
		ps303.OverlayValues[21] = d21
		ps303.OverlayValues[22] = d22
		ps303.OverlayValues[24] = d24
		ps303.OverlayValues[25] = d25
		ps303.OverlayValues[26] = d26
		ps303.OverlayValues[27] = d27
		ps303.OverlayValues[28] = d28
		ps303.OverlayValues[29] = d29
		ps303.OverlayValues[30] = d30
		ps303.OverlayValues[31] = d31
		ps303.OverlayValues[32] = d32
		ps303.OverlayValues[33] = d33
		ps303.OverlayValues[34] = d34
		ps303.OverlayValues[35] = d35
		ps303.OverlayValues[36] = d36
		ps303.OverlayValues[37] = d37
		ps303.OverlayValues[38] = d38
		ps303.OverlayValues[39] = d39
		ps303.OverlayValues[40] = d40
		ps303.OverlayValues[41] = d41
		ps303.OverlayValues[42] = d42
		ps303.OverlayValues[43] = d43
		ps303.OverlayValues[44] = d44
		ps303.OverlayValues[45] = d45
		ps303.OverlayValues[46] = d46
		ps303.OverlayValues[47] = d47
		ps303.OverlayValues[48] = d48
		ps303.OverlayValues[49] = d49
		ps303.OverlayValues[50] = d50
		ps303.OverlayValues[51] = d51
		ps303.OverlayValues[52] = d52
		ps303.OverlayValues[53] = d53
		ps303.OverlayValues[54] = d54
		ps303.OverlayValues[55] = d55
		ps303.OverlayValues[56] = d56
		ps303.OverlayValues[57] = d57
		ps303.OverlayValues[58] = d58
		ps303.OverlayValues[59] = d59
		ps303.OverlayValues[60] = d60
		ps303.OverlayValues[61] = d61
		ps303.OverlayValues[64] = d64
		ps303.OverlayValues[65] = d65
		ps303.OverlayValues[66] = d66
		ps303.OverlayValues[132] = d132
		ps303.OverlayValues[133] = d133
		ps303.OverlayValues[134] = d134
		ps303.OverlayValues[136] = d136
		ps303.OverlayValues[137] = d137
		ps303.OverlayValues[138] = d138
		ps303.OverlayValues[139] = d139
		ps303.OverlayValues[140] = d140
		ps303.OverlayValues[141] = d141
		ps303.OverlayValues[142] = d142
		ps303.OverlayValues[143] = d143
		ps303.OverlayValues[144] = d144
		ps303.OverlayValues[145] = d145
		ps303.OverlayValues[146] = d146
		ps303.OverlayValues[147] = d147
		ps303.OverlayValues[148] = d148
		ps303.OverlayValues[149] = d149
		ps303.OverlayValues[150] = d150
		ps303.OverlayValues[151] = d151
		ps303.OverlayValues[152] = d152
		ps303.OverlayValues[153] = d153
		ps303.OverlayValues[154] = d154
		ps303.OverlayValues[155] = d155
		ps303.OverlayValues[156] = d156
		ps303.OverlayValues[157] = d157
		ps303.OverlayValues[158] = d158
		ps303.OverlayValues[159] = d159
		ps303.OverlayValues[160] = d160
		ps303.OverlayValues[161] = d161
		ps303.OverlayValues[162] = d162
		ps303.OverlayValues[163] = d163
		ps303.OverlayValues[164] = d164
		ps303.OverlayValues[165] = d165
		ps303.OverlayValues[166] = d166
		ps303.OverlayValues[167] = d167
		ps303.OverlayValues[168] = d168
		ps303.OverlayValues[169] = d169
		ps303.OverlayValues[170] = d170
		ps303.OverlayValues[171] = d171
		ps303.OverlayValues[172] = d172
		ps303.OverlayValues[175] = d175
		ps303.OverlayValues[282] = d282
		ps303.OverlayValues[283] = d283
		ps303.OverlayValues[284] = d284
		ps303.OverlayValues[285] = d285
		ps303.OverlayValues[287] = d287
		ps303.OverlayValues[288] = d288
		ps303.OverlayValues[289] = d289
		ps303.OverlayValues[290] = d290
		ps303.OverlayValues[291] = d291
		ps303.OverlayValues[292] = d292
		ps303.OverlayValues[293] = d293
		ps303.OverlayValues[294] = d294
		ps303.OverlayValues[296] = d296
		ps303.OverlayValues[298] = d298
		ps303.OverlayValues[299] = d299
		ps303.OverlayValues[300] = d300
		ps303.OverlayValues[301] = d301
		ps303.OverlayValues[302] = d302
		ps303.PhiValues = make([]scm.JITValueDesc, 1)
		d305 = d6
		ps303.PhiValues[0] = d305
		ps304 := scm.PhiState{General: true}
		ps304.OverlayValues = make([]scm.JITValueDesc, 306)
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
		ps304.OverlayValues[17] = d17
		ps304.OverlayValues[18] = d18
		ps304.OverlayValues[19] = d19
		ps304.OverlayValues[20] = d20
		ps304.OverlayValues[21] = d21
		ps304.OverlayValues[22] = d22
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
		ps304.OverlayValues[132] = d132
		ps304.OverlayValues[133] = d133
		ps304.OverlayValues[134] = d134
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
		ps304.OverlayValues[282] = d282
		ps304.OverlayValues[283] = d283
		ps304.OverlayValues[284] = d284
		ps304.OverlayValues[285] = d285
		ps304.OverlayValues[287] = d287
		ps304.OverlayValues[288] = d288
		ps304.OverlayValues[289] = d289
		ps304.OverlayValues[290] = d290
		ps304.OverlayValues[291] = d291
		ps304.OverlayValues[292] = d292
		ps304.OverlayValues[293] = d293
		ps304.OverlayValues[294] = d294
		ps304.OverlayValues[296] = d296
		ps304.OverlayValues[298] = d298
		ps304.OverlayValues[299] = d299
		ps304.OverlayValues[300] = d300
		ps304.OverlayValues[301] = d301
		ps304.OverlayValues[302] = d302
		ps304.OverlayValues[305] = d305
		snap306 := d1
		snap307 := d2
		snap308 := d3
		snap309 := d4
		snap310 := d5
		snap311 := d6
		snap312 := d7
		snap313 := d8
		snap314 := d9
		snap315 := d10
		snap316 := d11
		snap317 := d12
		snap318 := d13
		snap319 := d14
		snap320 := d15
		snap321 := d17
		snap322 := d18
		snap323 := d19
		snap324 := d20
		snap325 := d21
		snap326 := d22
		snap327 := d24
		snap328 := d25
		snap329 := d26
		snap330 := d27
		snap331 := d28
		snap332 := d29
		snap333 := d30
		snap334 := d31
		snap335 := d32
		snap336 := d33
		snap337 := d34
		snap338 := d35
		snap339 := d36
		snap340 := d37
		snap341 := d38
		snap342 := d39
		snap343 := d40
		snap344 := d41
		snap345 := d42
		snap346 := d43
		snap347 := d44
		snap348 := d45
		snap349 := d46
		snap350 := d47
		snap351 := d48
		snap352 := d49
		snap353 := d50
		snap354 := d51
		snap355 := d52
		snap356 := d53
		snap357 := d54
		snap358 := d55
		snap359 := d56
		snap360 := d57
		snap361 := d58
		snap362 := d59
		snap363 := d60
		snap364 := d61
		snap365 := d64
		snap366 := d65
		snap367 := d66
		snap368 := d132
		snap369 := d133
		snap370 := d134
		snap371 := d136
		snap372 := d137
		snap373 := d138
		snap374 := d139
		snap375 := d140
		snap376 := d141
		snap377 := d142
		snap378 := d143
		snap379 := d144
		snap380 := d145
		snap381 := d146
		snap382 := d147
		snap383 := d148
		snap384 := d149
		snap385 := d150
		snap386 := d151
		snap387 := d152
		snap388 := d153
		snap389 := d154
		snap390 := d155
		snap391 := d156
		snap392 := d157
		snap393 := d158
		snap394 := d159
		snap395 := d160
		snap396 := d161
		snap397 := d162
		snap398 := d163
		snap399 := d164
		snap400 := d165
		snap401 := d166
		snap402 := d167
		snap403 := d168
		snap404 := d169
		snap405 := d170
		snap406 := d171
		snap407 := d172
		snap408 := d175
		snap409 := d282
		snap410 := d283
		snap411 := d284
		snap412 := d285
		snap413 := d287
		snap414 := d288
		snap415 := d289
		snap416 := d290
		snap417 := d291
		snap418 := d292
		snap419 := d293
		snap420 := d294
		snap421 := d296
		snap422 := d298
		snap423 := d299
		snap424 := d300
		snap425 := d301
		snap426 := d302
		snap427 := d305
		alloc428 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps303)
		}
		ctx.RestoreAllocState(alloc428)
		d1 = snap306
		d2 = snap307
		d3 = snap308
		d4 = snap309
		d5 = snap310
		d6 = snap311
		d7 = snap312
		d8 = snap313
		d9 = snap314
		d10 = snap315
		d11 = snap316
		d12 = snap317
		d13 = snap318
		d14 = snap319
		d15 = snap320
		d17 = snap321
		d18 = snap322
		d19 = snap323
		d20 = snap324
		d21 = snap325
		d22 = snap326
		d24 = snap327
		d25 = snap328
		d26 = snap329
		d27 = snap330
		d28 = snap331
		d29 = snap332
		d30 = snap333
		d31 = snap334
		d32 = snap335
		d33 = snap336
		d34 = snap337
		d35 = snap338
		d36 = snap339
		d37 = snap340
		d38 = snap341
		d39 = snap342
		d40 = snap343
		d41 = snap344
		d42 = snap345
		d43 = snap346
		d44 = snap347
		d45 = snap348
		d46 = snap349
		d47 = snap350
		d48 = snap351
		d49 = snap352
		d50 = snap353
		d51 = snap354
		d52 = snap355
		d53 = snap356
		d54 = snap357
		d55 = snap358
		d56 = snap359
		d57 = snap360
		d58 = snap361
		d59 = snap362
		d60 = snap363
		d61 = snap364
		d64 = snap365
		d65 = snap366
		d66 = snap367
		d132 = snap368
		d133 = snap369
		d134 = snap370
		d136 = snap371
		d137 = snap372
		d138 = snap373
		d139 = snap374
		d140 = snap375
		d141 = snap376
		d142 = snap377
		d143 = snap378
		d144 = snap379
		d145 = snap380
		d146 = snap381
		d147 = snap382
		d148 = snap383
		d149 = snap384
		d150 = snap385
		d151 = snap386
		d152 = snap387
		d153 = snap388
		d154 = snap389
		d155 = snap390
		d156 = snap391
		d157 = snap392
		d158 = snap393
		d159 = snap394
		d160 = snap395
		d161 = snap396
		d162 = snap397
		d163 = snap398
		d164 = snap399
		d165 = snap400
		d166 = snap401
		d167 = snap402
		d168 = snap403
		d169 = snap404
		d170 = snap405
		d171 = snap406
		d172 = snap407
		d175 = snap408
		d282 = snap409
		d283 = snap410
		d284 = snap411
		d285 = snap412
		d287 = snap413
		d288 = snap414
		d289 = snap415
		d290 = snap416
		d291 = snap417
		d292 = snap418
		d293 = snap419
		d294 = snap420
		d296 = snap421
		d298 = snap422
		d299 = snap423
		d300 = snap424
		d301 = snap425
		d302 = snap426
		d305 = snap427
		if !bbs[6].Rendered {
			return bbs[6].RenderPS(ps304)
		}
		return result
		ctx.FreeDesc(&d291)
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
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
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
		if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != scm.LocNone {
			d282 = ps.OverlayValues[282]
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
		if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
			d294 = ps.OverlayValues[294]
		}
		if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != scm.LocNone {
			d296 = ps.OverlayValues[296]
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
		if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != scm.LocNone {
			d305 = ps.OverlayValues[305]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		ctx.EnsureDesc(&d1)
		var d429 scm.JITValueDesc
		if d1.Loc == scm.LocImm {
			d429 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegReg(scratch, d1.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d429 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d429)
		}
		if d429.Loc == scm.LocImm {
			d429 = scm.JITValueDesc{Loc: scm.LocImm, Type: d429.Type, Imm: scm.NewInt(int64(uint64(d429.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d429.Reg, 32)
			ctx.EmitShrRegImm8(d429.Reg, 32)
		}
		if d429.Loc == scm.LocReg && d1.Loc == scm.LocReg && d429.Reg == d1.Reg {
			ctx.TransferReg(d1.Reg)
			d1.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d429)
		ctx.EmitStoreToStack(d429, int32(bbs[4].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d429)
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
			d430 = d1
			if d430.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d430)
			d431 = d430
			if d431.Loc == scm.LocImm {
				d431 = scm.JITValueDesc{Loc: scm.LocImm, Type: d431.Type, Imm: scm.NewInt(int64(uint64(d431.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d431.Reg, 32)
				ctx.EmitShrRegImm8(d431.Reg, 32)
			}
			ctx.EmitStoreToStack(d431, int32(bbs[4].PhiBase)+int32(16))
			d432 = d3
			if d432.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d432)
			d433 = d432
			if d433.Loc == scm.LocImm {
				d433 = scm.JITValueDesc{Loc: scm.LocImm, Type: d433.Type, Imm: scm.NewInt(int64(uint64(d433.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d433.Reg, 32)
				ctx.EmitShrRegImm8(d433.Reg, 32)
			}
			ctx.EmitStoreToStack(d433, int32(bbs[4].PhiBase)+int32(32))
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
		ps434 := scm.PhiState{General: ps.General}
		ps434.OverlayValues = make([]scm.JITValueDesc, 434)
		ps434.OverlayValues[1] = d1
		ps434.OverlayValues[2] = d2
		ps434.OverlayValues[3] = d3
		ps434.OverlayValues[4] = d4
		ps434.OverlayValues[5] = d5
		ps434.OverlayValues[6] = d6
		ps434.OverlayValues[7] = d7
		ps434.OverlayValues[8] = d8
		ps434.OverlayValues[9] = d9
		ps434.OverlayValues[10] = d10
		ps434.OverlayValues[11] = d11
		ps434.OverlayValues[12] = d12
		ps434.OverlayValues[13] = d13
		ps434.OverlayValues[14] = d14
		ps434.OverlayValues[15] = d15
		ps434.OverlayValues[17] = d17
		ps434.OverlayValues[18] = d18
		ps434.OverlayValues[19] = d19
		ps434.OverlayValues[20] = d20
		ps434.OverlayValues[21] = d21
		ps434.OverlayValues[22] = d22
		ps434.OverlayValues[24] = d24
		ps434.OverlayValues[25] = d25
		ps434.OverlayValues[26] = d26
		ps434.OverlayValues[27] = d27
		ps434.OverlayValues[28] = d28
		ps434.OverlayValues[29] = d29
		ps434.OverlayValues[30] = d30
		ps434.OverlayValues[31] = d31
		ps434.OverlayValues[32] = d32
		ps434.OverlayValues[33] = d33
		ps434.OverlayValues[34] = d34
		ps434.OverlayValues[35] = d35
		ps434.OverlayValues[36] = d36
		ps434.OverlayValues[37] = d37
		ps434.OverlayValues[38] = d38
		ps434.OverlayValues[39] = d39
		ps434.OverlayValues[40] = d40
		ps434.OverlayValues[41] = d41
		ps434.OverlayValues[42] = d42
		ps434.OverlayValues[43] = d43
		ps434.OverlayValues[44] = d44
		ps434.OverlayValues[45] = d45
		ps434.OverlayValues[46] = d46
		ps434.OverlayValues[47] = d47
		ps434.OverlayValues[48] = d48
		ps434.OverlayValues[49] = d49
		ps434.OverlayValues[50] = d50
		ps434.OverlayValues[51] = d51
		ps434.OverlayValues[52] = d52
		ps434.OverlayValues[53] = d53
		ps434.OverlayValues[54] = d54
		ps434.OverlayValues[55] = d55
		ps434.OverlayValues[56] = d56
		ps434.OverlayValues[57] = d57
		ps434.OverlayValues[58] = d58
		ps434.OverlayValues[59] = d59
		ps434.OverlayValues[60] = d60
		ps434.OverlayValues[61] = d61
		ps434.OverlayValues[64] = d64
		ps434.OverlayValues[65] = d65
		ps434.OverlayValues[66] = d66
		ps434.OverlayValues[132] = d132
		ps434.OverlayValues[133] = d133
		ps434.OverlayValues[134] = d134
		ps434.OverlayValues[136] = d136
		ps434.OverlayValues[137] = d137
		ps434.OverlayValues[138] = d138
		ps434.OverlayValues[139] = d139
		ps434.OverlayValues[140] = d140
		ps434.OverlayValues[141] = d141
		ps434.OverlayValues[142] = d142
		ps434.OverlayValues[143] = d143
		ps434.OverlayValues[144] = d144
		ps434.OverlayValues[145] = d145
		ps434.OverlayValues[146] = d146
		ps434.OverlayValues[147] = d147
		ps434.OverlayValues[148] = d148
		ps434.OverlayValues[149] = d149
		ps434.OverlayValues[150] = d150
		ps434.OverlayValues[151] = d151
		ps434.OverlayValues[152] = d152
		ps434.OverlayValues[153] = d153
		ps434.OverlayValues[154] = d154
		ps434.OverlayValues[155] = d155
		ps434.OverlayValues[156] = d156
		ps434.OverlayValues[157] = d157
		ps434.OverlayValues[158] = d158
		ps434.OverlayValues[159] = d159
		ps434.OverlayValues[160] = d160
		ps434.OverlayValues[161] = d161
		ps434.OverlayValues[162] = d162
		ps434.OverlayValues[163] = d163
		ps434.OverlayValues[164] = d164
		ps434.OverlayValues[165] = d165
		ps434.OverlayValues[166] = d166
		ps434.OverlayValues[167] = d167
		ps434.OverlayValues[168] = d168
		ps434.OverlayValues[169] = d169
		ps434.OverlayValues[170] = d170
		ps434.OverlayValues[171] = d171
		ps434.OverlayValues[172] = d172
		ps434.OverlayValues[175] = d175
		ps434.OverlayValues[282] = d282
		ps434.OverlayValues[283] = d283
		ps434.OverlayValues[284] = d284
		ps434.OverlayValues[285] = d285
		ps434.OverlayValues[287] = d287
		ps434.OverlayValues[288] = d288
		ps434.OverlayValues[289] = d289
		ps434.OverlayValues[290] = d290
		ps434.OverlayValues[291] = d291
		ps434.OverlayValues[292] = d292
		ps434.OverlayValues[293] = d293
		ps434.OverlayValues[294] = d294
		ps434.OverlayValues[296] = d296
		ps434.OverlayValues[298] = d298
		ps434.OverlayValues[299] = d299
		ps434.OverlayValues[300] = d300
		ps434.OverlayValues[301] = d301
		ps434.OverlayValues[302] = d302
		ps434.OverlayValues[305] = d305
		ps434.OverlayValues[429] = d429
		ps434.OverlayValues[430] = d430
		ps434.OverlayValues[431] = d431
		ps434.OverlayValues[432] = d432
		ps434.OverlayValues[433] = d433
		ps434.PhiValues = make([]scm.JITValueDesc, 3)
		d435 = d1
		ps434.PhiValues[1] = d435
		d436 = d3
		ps434.PhiValues[2] = d436
		if ps434.General && bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
			return result
		}
		return bbs[4].RenderPS(ps434)
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
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
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
		if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != scm.LocNone {
			d282 = ps.OverlayValues[282]
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
		if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
			d294 = ps.OverlayValues[294]
		}
		if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != scm.LocNone {
			d296 = ps.OverlayValues[296]
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
		if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != scm.LocNone {
			d305 = ps.OverlayValues[305]
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
		if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != scm.LocNone {
			d435 = ps.OverlayValues[435]
		}
		if len(ps.OverlayValues) > 436 && ps.OverlayValues[436].Loc != scm.LocNone {
			d436 = ps.OverlayValues[436]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		d437 = d5
		_ = d437
		ctx.StabilizeDescForControlFlow(&d437)
		ctx.StabilizeDescForControlFlow(&d5)
		phiBase438 = ctx.AllocStack(int32(16))
		d439 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase438) + int32(0)}
		_ = d439
		lbl33 := ctx.ReserveLabel()
		bbpos_3_0 := int32(-1)
		_ = bbpos_3_0
		lbl34 := ctx.ReserveLabel()
		_ = lbl34
		bbpos_3_1 := int32(-1)
		_ = bbpos_3_1
		lbl35 := ctx.ReserveLabel()
		_ = lbl35
		bbpos_3_2 := int32(-1)
		_ = bbpos_3_2
		lbl36 := ctx.ReserveLabel()
		_ = lbl36
		bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl34)
		ctx.ResolveFixups()
		d439 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d437)
		ctx.EnsureDesc(&d437)
		var d440 scm.JITValueDesc
		if d437.Loc == scm.LocImm {
			d440 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d437.Imm.Int()))))}
		} else {
			r96 := ctx.AllocReg()
			ctx.EmitMovRegReg(r96, d437.Reg)
			ctx.EmitShlRegImm8(r96, 32)
			ctx.EmitShrRegImm8(r96, 32)
			d440 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
			ctx.BindReg(r96, &d440)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d441 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			r97 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r97, fieldAddr)
			d441 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r97}
			ctx.BindReg(r97, &d441)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			r98 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r98, thisptr.Reg, off)
			d441 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r98}
			ctx.BindReg(r98, &d441)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d441)
		ctx.EnsureDesc(&d441)
		var d442 scm.JITValueDesc
		if d441.Loc == scm.LocImm {
			d442 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d441.Imm.Int()))))}
		} else {
			r99 := ctx.AllocReg()
			ctx.EmitMovRegReg(r99, d441.Reg)
			ctx.EmitShlRegImm8(r99, 56)
			ctx.EmitShrRegImm8(r99, 56)
			d442 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
			ctx.BindReg(r99, &d442)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d440)
		ctx.EnsureDesc(&d442)
		ctx.EnsureDescsTogether(&d440, &d442)
		var d443 scm.JITValueDesc
		if d440.Loc == scm.LocImm && d442.Loc == scm.LocImm {
			d443 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d440.Imm.Int() * d442.Imm.Int())}
		} else if d440.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d442.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d440.Imm.Int()))
			ctx.EmitImulInt64(scratch, d442.Reg)
			d443 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d443)
		} else if d442.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d440.Reg)
			ctx.EmitMovRegReg(scratch, d440.Reg)
			if d442.Imm.Int() >= -2147483648 && d442.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d442.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d442.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d443 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d443)
		} else {
			r100 := ctx.AllocRegExcept(d440.Reg, d442.Reg)
			ctx.EmitMovRegReg(r100, d440.Reg)
			ctx.EmitImulInt64(r100, d442.Reg)
			d443 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r100}
			ctx.BindReg(r100, &d443)
		}
		if d443.Loc == scm.LocReg && d440.Loc == scm.LocReg && d443.Reg == d440.Reg {
			ctx.TransferReg(d440.Reg)
			d440.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d443)
		ctx.FreeDesc(&d440)
		ctx.FreeDesc(&d442)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d444 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
			r101 := ctx.AllocReg()
			r102 := ctx.AllocRegExcept(r101)
			r103 := ctx.AllocRegExcept(r101, r102)
			ctx.EmitMovRegMem64(r101, fieldAddr)
			ctx.EmitMovRegMem64(r102, fieldAddr+8)
			ctx.EmitMovRegMem64(r103, fieldAddr+16)
			d444 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r101, Reg2: r102, Reg3: r103}
			ctx.BindReg(r101, &d444)
			ctx.BindReg(r102, &d444)
			ctx.BindReg(r103, &d444)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
			r104 := ctx.AllocReg()
			r105 := ctx.AllocRegExcept(r104)
			r106 := ctx.AllocRegExcept(r104, r105)
			ctx.EmitMovRegMem(r104, thisptr.Reg, off)
			ctx.EmitMovRegMem(r105, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r106, thisptr.Reg, off+16)
			d444 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r104, Reg2: r105, Reg3: r106}
			ctx.BindReg(r104, &d444)
			ctx.BindReg(r105, &d444)
			ctx.BindReg(r106, &d444)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d443)
		var d445 scm.JITValueDesc
		if d443.Loc == scm.LocImm {
			d445 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d443.Imm.Int() / 64)}
		} else {
			r107 := ctx.AllocRegExcept(d443.Reg)
			ctx.EmitMovRegReg(r107, d443.Reg)
			ctx.EmitShrRegImm8(r107, 6)
			d445 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r107}
			ctx.BindReg(r107, &d445)
		}
		if d445.Loc == scm.LocReg && d443.Loc == scm.LocReg && d445.Reg == d443.Reg {
			ctx.TransferReg(d443.Reg)
			d443.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d445)
		ctx.ReclaimUntrackedRegs()
		d447 = ctx.EmitSliceElementAddress(&d444, &d445, 8)
		ctx.EnsureDesc(&d447)
		ctx.EmitMovRegMem(d447.Reg, d447.Reg, 0)
		d446 = d447
		ctx.FreeDesc(&d445)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d443)
		var d448 scm.JITValueDesc
		if d443.Loc == scm.LocImm {
			d448 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d443.Imm.Int() % 64)}
		} else {
			r108 := ctx.AllocRegExcept(d443.Reg)
			ctx.EmitMovRegReg(r108, d443.Reg)
			ctx.EmitAndRegImm32(r108, 63)
			d448 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
			ctx.BindReg(r108, &d448)
		}
		if d448.Loc == scm.LocReg && d443.Loc == scm.LocReg && d448.Reg == d443.Reg {
			ctx.TransferReg(d443.Reg)
			d443.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d446)
		ctx.EnsureDesc(&d448)
		var d449 scm.JITValueDesc
		if d446.Loc == scm.LocImm && d448.Loc == scm.LocImm {
			d449 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d446.Imm.Int()) << uint64(d448.Imm.Int())))}
		} else if d448.Loc == scm.LocImm {
			r109 := ctx.AllocRegExcept(d446.Reg)
			ctx.EmitMovRegReg(r109, d446.Reg)
			ctx.EmitShlRegImm8(r109, uint8(d448.Imm.Int()))
			d449 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r109}
			ctx.BindReg(r109, &d449)
		} else {
			{
				shiftSrc := d446.Reg
				r110 := ctx.AllocRegExcept(d446.Reg)
				ctx.EmitMovRegReg(r110, d446.Reg)
				shiftSrc = r110
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d448.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d448.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d448.Reg)
				}
				ctx.EmitShlRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d449 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d449)
			}
		}
		if d449.Loc == scm.LocReg && d446.Loc == scm.LocReg && d449.Reg == d446.Reg {
			ctx.TransferReg(d446.Reg)
			d446.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d449)
		ctx.FreeDesc(&d446)
		ctx.FreeDesc(&d448)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d443)
		var d450 scm.JITValueDesc
		if d443.Loc == scm.LocImm {
			d450 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d443.Imm.Int() % 64)}
		} else {
			r111 := ctx.AllocRegExcept(d443.Reg)
			ctx.EmitMovRegReg(r111, d443.Reg)
			ctx.EmitAndRegImm32(r111, 63)
			d450 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r111}
			ctx.BindReg(r111, &d450)
		}
		if d450.Loc == scm.LocReg && d443.Loc == scm.LocReg && d450.Reg == d443.Reg {
			ctx.TransferReg(d443.Reg)
			d443.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d441)
		ctx.EnsureDesc(&d441)
		var d451 scm.JITValueDesc
		if d441.Loc == scm.LocImm {
			d451 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d441.Imm.Int()))))}
		} else {
			r112 := ctx.AllocReg()
			ctx.EmitMovRegReg(r112, d441.Reg)
			ctx.EmitShlRegImm8(r112, 56)
			ctx.EmitShrRegImm8(r112, 56)
			d451 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r112}
			ctx.BindReg(r112, &d451)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d450)
		ctx.EnsureDesc(&d451)
		ctx.EnsureDescsTogether(&d450, &d451)
		var d452 scm.JITValueDesc
		if d450.Loc == scm.LocImm && d451.Loc == scm.LocImm {
			d452 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d450.Imm.Int() + d451.Imm.Int())}
		} else if d451.Loc == scm.LocImm && d451.Imm.Int() == 0 {
			r113 := ctx.AllocRegExcept(d450.Reg)
			ctx.EmitMovRegReg(r113, d450.Reg)
			d452 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r113}
			ctx.BindReg(r113, &d452)
		} else if d450.Loc == scm.LocImm && d450.Imm.Int() == 0 {
			d452 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d451.Reg}
			ctx.BindReg(d451.Reg, &d452)
		} else if d450.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d451.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d450.Imm.Int()))
			ctx.EmitAddInt64(scratch, d451.Reg)
			d452 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d452)
		} else if d451.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d450.Reg)
			ctx.EmitMovRegReg(scratch, d450.Reg)
			if d451.Imm.Int() >= -2147483648 && d451.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d451.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d451.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d452 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d452)
		} else {
			r114 := ctx.AllocRegExcept(d450.Reg, d451.Reg)
			ctx.EmitMovRegReg(r114, d450.Reg)
			ctx.EmitAddInt64(r114, d451.Reg)
			d452 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r114}
			ctx.BindReg(r114, &d452)
		}
		if d452.Loc == scm.LocReg && d450.Loc == scm.LocReg && d452.Reg == d450.Reg {
			ctx.TransferReg(d450.Reg)
			d450.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d450)
		ctx.FreeDesc(&d451)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d452)
		var d453 scm.JITValueDesc
		if d452.Loc == scm.LocImm {
			d453 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d452.Imm.Int()) > uint64(0x40))}
		} else {
			r115 := ctx.AllocRegExcept(d452.Reg)
			ctx.EmitCmpRegImm32(d452.Reg, 64)
			ctx.EmitSetcc(r115, scm.CondUnsignedAbove)
			d453 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r115}
			ctx.BindReg(r115, &d453)
		}
		ctx.FreeDesc(&d452)
		ctx.ReclaimUntrackedRegs()
		d454 = d453
		ctx.EnsureDesc(&d454)
		if d454.Loc != scm.LocImm && d454.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		lbl37 := ctx.ReserveLabel()
		lbl38 := ctx.ReserveLabel()
		if d454.Loc == scm.LocImm {
			if d454.Imm.Bool() {
				ctx.MarkLabel(lbl37)
				ctx.EmitJmp(lbl35)
			} else {
				ctx.MarkLabel(lbl38)
				ctx.SyncDesc(&d449)
				if d449.Loc == scm.LocReg {
					ctx.ProtectReg(d449.Reg)
				} else if d449.Loc == scm.LocRegPair {
					ctx.ProtectReg(d449.Reg)
					ctx.ProtectReg(d449.Reg2)
				}
				d455 = d449
				if d455.Loc == scm.LocNone {
					panic("jit: phi source has no location")
				}
				ctx.EnsureDesc(&d455)
				ctx.EmitStoreToStack(d455, int32(phiBase438)+int32(0))
				if d449.Loc == scm.LocReg {
					ctx.UnprotectReg(d449.Reg)
				} else if d449.Loc == scm.LocRegPair {
					ctx.UnprotectReg(d449.Reg)
					ctx.UnprotectReg(d449.Reg2)
				}
				ctx.EmitJmp(lbl36)
			}
		} else {
			ctx.EmitCmpRegImm32(d454.Reg, 0)
			ctx.EmitJump(scm.CondNotEqual, lbl37)
			ctx.EmitJmp(lbl38)
			ctx.MarkLabel(lbl37)
			ctx.EmitJmp(lbl35)
			ctx.MarkLabel(lbl38)
			ctx.SyncDesc(&d449)
			if d449.Loc == scm.LocReg {
				ctx.ProtectReg(d449.Reg)
			} else if d449.Loc == scm.LocRegPair {
				ctx.ProtectReg(d449.Reg)
				ctx.ProtectReg(d449.Reg2)
			}
			d456 = d449
			if d456.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d456)
			ctx.EmitStoreToStack(d456, int32(phiBase438)+int32(0))
			if d449.Loc == scm.LocReg {
				ctx.UnprotectReg(d449.Reg)
			} else if d449.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d449.Reg)
				ctx.UnprotectReg(d449.Reg2)
			}
			ctx.EmitJmp(lbl36)
		}
		ctx.FreeDesc(&d453)
		bbpos_3_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl36)
		ctx.ResolveFixups()
		d439 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d441)
		ctx.EnsureDesc(&d441)
		var d457 scm.JITValueDesc
		if d441.Loc == scm.LocImm {
			d457 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d441.Imm.Int()))))}
		} else {
			r116 := ctx.AllocReg()
			ctx.EmitMovRegReg(r116, d441.Reg)
			ctx.EmitShlRegImm8(r116, 56)
			ctx.EmitShrRegImm8(r116, 56)
			d457 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r116}
			ctx.BindReg(r116, &d457)
		}
		ctx.ReclaimUntrackedRegs()
		d458 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d457)
		ctx.EnsureDescsTogether(&d458, &d457)
		var d459 scm.JITValueDesc
		if d458.Loc == scm.LocImm && d457.Loc == scm.LocImm {
			d459 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d458.Imm.Int() - d457.Imm.Int())}
		} else if d457.Loc == scm.LocImm && d457.Imm.Int() == 0 {
			r117 := ctx.AllocRegExcept(d458.Reg)
			ctx.EmitMovRegReg(r117, d458.Reg)
			d459 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
			ctx.BindReg(r117, &d459)
		} else if d458.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d457.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d458.Imm.Int()))
			ctx.EmitSubInt64(scratch, d457.Reg)
			d459 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d459)
		} else if d457.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d458.Reg)
			ctx.EmitMovRegReg(scratch, d458.Reg)
			if d457.Imm.Int() >= -2147483648 && d457.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d457.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d457.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d459 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d459)
		} else {
			r118 := ctx.AllocRegExcept(d458.Reg, d457.Reg)
			ctx.EmitMovRegReg(r118, d458.Reg)
			ctx.EmitSubInt64(r118, d457.Reg)
			d459 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r118}
			ctx.BindReg(r118, &d459)
		}
		if d459.Loc == scm.LocReg && d458.Loc == scm.LocReg && d459.Reg == d458.Reg {
			ctx.TransferReg(d458.Reg)
			d458.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d457)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d439)
		ctx.EnsureDesc(&d459)
		var d460 scm.JITValueDesc
		if d439.Loc == scm.LocImm && d459.Loc == scm.LocImm {
			d460 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d439.Imm.Int()) >> uint64(d459.Imm.Int())))}
		} else if d459.Loc == scm.LocImm {
			r119 := ctx.AllocRegExcept(d439.Reg)
			ctx.EmitMovRegReg(r119, d439.Reg)
			ctx.EmitShrRegImm8(r119, uint8(d459.Imm.Int()))
			d460 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
			ctx.BindReg(r119, &d460)
		} else {
			{
				shiftSrc := d439.Reg
				r120 := ctx.AllocRegExcept(d439.Reg)
				ctx.EmitMovRegReg(r120, d439.Reg)
				shiftSrc = r120
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d459.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d459.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d459.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d460 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d460)
			}
		}
		if d460.Loc == scm.LocReg && d439.Loc == scm.LocReg && d460.Reg == d439.Reg {
			ctx.TransferReg(d439.Reg)
			d439.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d439)
		ctx.FreeDesc(&d459)
		ctx.ReclaimUntrackedRegs()
		r121 := ctx.AllocReg()
		ctx.EnsureDesc(&d460)
		ctx.EnsureDesc(&d460)
		if d460.Loc == scm.LocRegPair {
			panic("jit: scalar inline return has scm.LocRegPair")
		} else {
			ctx.EmitMovToReg(r121, d460)
		}
		ctx.EmitJmp(lbl33)
		bbpos_3_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl35)
		ctx.ResolveFixups()
		d439 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d443)
		var d461 scm.JITValueDesc
		if d443.Loc == scm.LocImm {
			d461 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d443.Imm.Int() / 64)}
		} else {
			r122 := ctx.AllocRegExcept(d443.Reg)
			ctx.EmitMovRegReg(r122, d443.Reg)
			ctx.EmitShrRegImm8(r122, 6)
			d461 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
			ctx.BindReg(r122, &d461)
		}
		if d461.Loc == scm.LocReg && d443.Loc == scm.LocReg && d461.Reg == d443.Reg {
			ctx.TransferReg(d443.Reg)
			d443.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d461)
		ctx.EnsureDesc(&d461)
		var d462 scm.JITValueDesc
		if d461.Loc == scm.LocImm {
			d462 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d461.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d461.Reg)
			ctx.EmitMovRegReg(scratch, d461.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d462 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d462)
		}
		if d462.Loc == scm.LocReg && d461.Loc == scm.LocReg && d462.Reg == d461.Reg {
			ctx.TransferReg(d461.Reg)
			d461.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d461)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d462)
		ctx.ReclaimUntrackedRegs()
		d464 = ctx.EmitSliceElementAddress(&d444, &d462, 8)
		ctx.EnsureDesc(&d464)
		ctx.EmitMovRegMem(d464.Reg, d464.Reg, 0)
		d463 = d464
		ctx.FreeDesc(&d462)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d443)
		var d465 scm.JITValueDesc
		if d443.Loc == scm.LocImm {
			d465 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d443.Imm.Int() % 64)}
		} else {
			r123 := ctx.AllocRegExcept(d443.Reg)
			ctx.EmitMovRegReg(r123, d443.Reg)
			ctx.EmitAndRegImm32(r123, 63)
			d465 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
			ctx.BindReg(r123, &d465)
		}
		if d465.Loc == scm.LocReg && d443.Loc == scm.LocReg && d465.Reg == d443.Reg {
			ctx.TransferReg(d443.Reg)
			d443.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d443)
		ctx.ReclaimUntrackedRegs()
		d466 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d465)
		ctx.EnsureDescsTogether(&d466, &d465)
		var d467 scm.JITValueDesc
		if d466.Loc == scm.LocImm && d465.Loc == scm.LocImm {
			d467 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d466.Imm.Int() - d465.Imm.Int())}
		} else if d465.Loc == scm.LocImm && d465.Imm.Int() == 0 {
			r124 := ctx.AllocRegExcept(d466.Reg)
			ctx.EmitMovRegReg(r124, d466.Reg)
			d467 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
			ctx.BindReg(r124, &d467)
		} else if d466.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d465.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d466.Imm.Int()))
			ctx.EmitSubInt64(scratch, d465.Reg)
			d467 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d467)
		} else if d465.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d466.Reg)
			ctx.EmitMovRegReg(scratch, d466.Reg)
			if d465.Imm.Int() >= -2147483648 && d465.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d465.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d465.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d467 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d467)
		} else {
			r125 := ctx.AllocRegExcept(d466.Reg, d465.Reg)
			ctx.EmitMovRegReg(r125, d466.Reg)
			ctx.EmitSubInt64(r125, d465.Reg)
			d467 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
			ctx.BindReg(r125, &d467)
		}
		if d467.Loc == scm.LocReg && d466.Loc == scm.LocReg && d467.Reg == d466.Reg {
			ctx.TransferReg(d466.Reg)
			d466.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d465)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d463)
		ctx.EnsureDesc(&d467)
		var d468 scm.JITValueDesc
		if d463.Loc == scm.LocImm && d467.Loc == scm.LocImm {
			d468 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d463.Imm.Int()) >> uint64(d467.Imm.Int())))}
		} else if d467.Loc == scm.LocImm {
			r126 := ctx.AllocRegExcept(d463.Reg)
			ctx.EmitMovRegReg(r126, d463.Reg)
			ctx.EmitShrRegImm8(r126, uint8(d467.Imm.Int()))
			d468 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r126}
			ctx.BindReg(r126, &d468)
		} else {
			{
				shiftSrc := d463.Reg
				r127 := ctx.AllocRegExcept(d463.Reg)
				ctx.EmitMovRegReg(r127, d463.Reg)
				shiftSrc = r127
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d467.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d467.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d467.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d468 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d468)
			}
		}
		if d468.Loc == scm.LocReg && d463.Loc == scm.LocReg && d468.Reg == d463.Reg {
			ctx.TransferReg(d463.Reg)
			d463.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d463)
		ctx.FreeDesc(&d467)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d449)
		ctx.EnsureDesc(&d468)
		var d469 scm.JITValueDesc
		if d449.Loc == scm.LocImm && d468.Loc == scm.LocImm {
			d469 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d449.Imm.Int() | d468.Imm.Int())}
		} else if d449.Loc == scm.LocImm && d449.Imm.Int() == 0 {
			d469 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d468.Reg}
			ctx.BindReg(d468.Reg, &d469)
		} else if d468.Loc == scm.LocImm && d468.Imm.Int() == 0 {
			r128 := ctx.AllocRegExcept(d449.Reg)
			ctx.EmitMovRegReg(r128, d449.Reg)
			d469 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
			ctx.BindReg(r128, &d469)
		} else if d449.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d468.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d449.Imm.Int()))
			ctx.EmitOrInt64(scratch, d468.Reg)
			d469 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d469)
		} else if d468.Loc == scm.LocImm {
			r129 := ctx.AllocRegExcept(d449.Reg)
			ctx.EmitMovRegReg(r129, d449.Reg)
			if d468.Imm.Int() >= -2147483648 && d468.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r129, int32(d468.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d468.Imm.Int()))
				ctx.EmitOrInt64(r129, scm.RegR11)
			}
			d469 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
			ctx.BindReg(r129, &d469)
		} else {
			r130 := ctx.AllocRegExcept(d449.Reg, d468.Reg)
			ctx.EmitMovRegReg(r130, d449.Reg)
			ctx.EmitOrInt64(r130, d468.Reg)
			d469 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
			ctx.BindReg(r130, &d469)
		}
		if d469.Loc == scm.LocReg && d449.Loc == scm.LocReg && d469.Reg == d449.Reg {
			ctx.TransferReg(d449.Reg)
			d449.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d469)
		ctx.EmitStoreToStack(d469, int32(phiBase438)+int32(0))
		ctx.StabilizeDescForControlFlow(&d469)
		ctx.FreeDesc(&d468)
		ctx.ReclaimUntrackedRegs()
		ctx.EmitJmp(lbl36)
		ctx.MarkLabel(lbl33)
		d470 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r121}
		ctx.BindReg(r121, &d470)
		ctx.BindReg(r121, &d470)
		ctx.EnsureDesc(&d470)
		ctx.EnsureDesc(&d470)
		var d471 scm.JITValueDesc
		if d470.Loc == scm.LocImm {
			d471 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d470.Imm.Int()))))}
		} else {
			r131 := ctx.AllocReg()
			ctx.EmitMovRegReg(r131, d470.Reg)
			d471 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r131}
			ctx.BindReg(r131, &d471)
		}
		ctx.FreeDesc(&d470)
		ctx.EnsureDesc(&d471)
		ctx.EnsureDesc(&d57)
		ctx.EnsureDescsTogether(&d471, &d57)
		var d472 scm.JITValueDesc
		if d471.Loc == scm.LocImm && d57.Loc == scm.LocImm {
			d472 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d471.Imm.Int() + d57.Imm.Int())}
		} else if d57.Loc == scm.LocImm && d57.Imm.Int() == 0 {
			r132 := ctx.AllocRegExcept(d471.Reg)
			ctx.EmitMovRegReg(r132, d471.Reg)
			d472 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
			ctx.BindReg(r132, &d472)
		} else if d471.Loc == scm.LocImm && d471.Imm.Int() == 0 {
			d472 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d57.Reg}
			ctx.BindReg(d57.Reg, &d472)
		} else if d471.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d57.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d471.Imm.Int()))
			ctx.EmitAddInt64(scratch, d57.Reg)
			d472 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d472)
		} else if d57.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d471.Reg)
			ctx.EmitMovRegReg(scratch, d471.Reg)
			if d57.Imm.Int() >= -2147483648 && d57.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d57.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d57.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d472 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d472)
		} else {
			r133 := ctx.AllocRegExcept(d471.Reg, d57.Reg)
			ctx.EmitMovRegReg(r133, d471.Reg)
			ctx.EmitAddInt64(r133, d57.Reg)
			d472 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
			ctx.BindReg(r133, &d472)
		}
		if d472.Loc == scm.LocReg && d471.Loc == scm.LocReg && d472.Reg == d471.Reg {
			ctx.TransferReg(d471.Reg)
			d471.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d471)
		ctx.EnsureDesc(&d472)
		ctx.EnsureDesc(&d472)
		var d473 scm.JITValueDesc
		if d472.Loc == scm.LocImm {
			d473 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d472.Imm.Int()))))}
		} else {
			r134 := ctx.AllocReg()
			ctx.EmitMovRegReg(r134, d472.Reg)
			ctx.EmitShlRegImm8(r134, 32)
			ctx.EmitShrRegImm8(r134, 32)
			d473 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r134}
			ctx.BindReg(r134, &d473)
		}
		ctx.FreeDesc(&d472)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d473)
		ctx.EnsureDescsTogether(&idxInt, &d473)
		var d474 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm && d473.Loc == scm.LocImm {
			d474 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d473.Imm.Int()))}
		} else if d473.Loc == scm.LocImm {
			r135 := ctx.AllocRegExcept(idxInt.Reg)
			if d473.Imm.Int() >= -2147483648 && d473.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(idxInt.Reg, int32(d473.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d473.Imm.Int()))
				ctx.EmitCmpInt64(idxInt.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r135, scm.CondUnsignedBelow)
			d474 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r135}
			ctx.BindReg(r135, &d474)
		} else if idxInt.Loc == scm.LocImm {
			r136 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d473.Reg)
			ctx.EmitSetcc(r136, scm.CondUnsignedBelow)
			d474 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r136}
			ctx.BindReg(r136, &d474)
		} else {
			r137 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitCmpInt64(idxInt.Reg, d473.Reg)
			ctx.EmitSetcc(r137, scm.CondUnsignedBelow)
			d474 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r137}
			ctx.BindReg(r137, &d474)
		}
		ctx.FreeDesc(&d473)
		d475 = d474
		ctx.EnsureDesc(&d475)
		if d475.Loc != scm.LocImm && d475.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d475.Loc == scm.LocImm {
			if d475.Imm.Bool() {
				if ps.General {
				}
				ps476 := scm.PhiState{General: ps.General}
				ps476.OverlayValues = make([]scm.JITValueDesc, 476)
				ps476.OverlayValues[1] = d1
				ps476.OverlayValues[2] = d2
				ps476.OverlayValues[3] = d3
				ps476.OverlayValues[4] = d4
				ps476.OverlayValues[5] = d5
				ps476.OverlayValues[6] = d6
				ps476.OverlayValues[7] = d7
				ps476.OverlayValues[8] = d8
				ps476.OverlayValues[9] = d9
				ps476.OverlayValues[10] = d10
				ps476.OverlayValues[11] = d11
				ps476.OverlayValues[12] = d12
				ps476.OverlayValues[13] = d13
				ps476.OverlayValues[14] = d14
				ps476.OverlayValues[15] = d15
				ps476.OverlayValues[17] = d17
				ps476.OverlayValues[18] = d18
				ps476.OverlayValues[19] = d19
				ps476.OverlayValues[20] = d20
				ps476.OverlayValues[21] = d21
				ps476.OverlayValues[22] = d22
				ps476.OverlayValues[24] = d24
				ps476.OverlayValues[25] = d25
				ps476.OverlayValues[26] = d26
				ps476.OverlayValues[27] = d27
				ps476.OverlayValues[28] = d28
				ps476.OverlayValues[29] = d29
				ps476.OverlayValues[30] = d30
				ps476.OverlayValues[31] = d31
				ps476.OverlayValues[32] = d32
				ps476.OverlayValues[33] = d33
				ps476.OverlayValues[34] = d34
				ps476.OverlayValues[35] = d35
				ps476.OverlayValues[36] = d36
				ps476.OverlayValues[37] = d37
				ps476.OverlayValues[38] = d38
				ps476.OverlayValues[39] = d39
				ps476.OverlayValues[40] = d40
				ps476.OverlayValues[41] = d41
				ps476.OverlayValues[42] = d42
				ps476.OverlayValues[43] = d43
				ps476.OverlayValues[44] = d44
				ps476.OverlayValues[45] = d45
				ps476.OverlayValues[46] = d46
				ps476.OverlayValues[47] = d47
				ps476.OverlayValues[48] = d48
				ps476.OverlayValues[49] = d49
				ps476.OverlayValues[50] = d50
				ps476.OverlayValues[51] = d51
				ps476.OverlayValues[52] = d52
				ps476.OverlayValues[53] = d53
				ps476.OverlayValues[54] = d54
				ps476.OverlayValues[55] = d55
				ps476.OverlayValues[56] = d56
				ps476.OverlayValues[57] = d57
				ps476.OverlayValues[58] = d58
				ps476.OverlayValues[59] = d59
				ps476.OverlayValues[60] = d60
				ps476.OverlayValues[61] = d61
				ps476.OverlayValues[64] = d64
				ps476.OverlayValues[65] = d65
				ps476.OverlayValues[66] = d66
				ps476.OverlayValues[132] = d132
				ps476.OverlayValues[133] = d133
				ps476.OverlayValues[134] = d134
				ps476.OverlayValues[136] = d136
				ps476.OverlayValues[137] = d137
				ps476.OverlayValues[138] = d138
				ps476.OverlayValues[139] = d139
				ps476.OverlayValues[140] = d140
				ps476.OverlayValues[141] = d141
				ps476.OverlayValues[142] = d142
				ps476.OverlayValues[143] = d143
				ps476.OverlayValues[144] = d144
				ps476.OverlayValues[145] = d145
				ps476.OverlayValues[146] = d146
				ps476.OverlayValues[147] = d147
				ps476.OverlayValues[148] = d148
				ps476.OverlayValues[149] = d149
				ps476.OverlayValues[150] = d150
				ps476.OverlayValues[151] = d151
				ps476.OverlayValues[152] = d152
				ps476.OverlayValues[153] = d153
				ps476.OverlayValues[154] = d154
				ps476.OverlayValues[155] = d155
				ps476.OverlayValues[156] = d156
				ps476.OverlayValues[157] = d157
				ps476.OverlayValues[158] = d158
				ps476.OverlayValues[159] = d159
				ps476.OverlayValues[160] = d160
				ps476.OverlayValues[161] = d161
				ps476.OverlayValues[162] = d162
				ps476.OverlayValues[163] = d163
				ps476.OverlayValues[164] = d164
				ps476.OverlayValues[165] = d165
				ps476.OverlayValues[166] = d166
				ps476.OverlayValues[167] = d167
				ps476.OverlayValues[168] = d168
				ps476.OverlayValues[169] = d169
				ps476.OverlayValues[170] = d170
				ps476.OverlayValues[171] = d171
				ps476.OverlayValues[172] = d172
				ps476.OverlayValues[175] = d175
				ps476.OverlayValues[282] = d282
				ps476.OverlayValues[283] = d283
				ps476.OverlayValues[284] = d284
				ps476.OverlayValues[285] = d285
				ps476.OverlayValues[287] = d287
				ps476.OverlayValues[288] = d288
				ps476.OverlayValues[289] = d289
				ps476.OverlayValues[290] = d290
				ps476.OverlayValues[291] = d291
				ps476.OverlayValues[292] = d292
				ps476.OverlayValues[293] = d293
				ps476.OverlayValues[294] = d294
				ps476.OverlayValues[296] = d296
				ps476.OverlayValues[298] = d298
				ps476.OverlayValues[299] = d299
				ps476.OverlayValues[300] = d300
				ps476.OverlayValues[301] = d301
				ps476.OverlayValues[302] = d302
				ps476.OverlayValues[305] = d305
				ps476.OverlayValues[429] = d429
				ps476.OverlayValues[430] = d430
				ps476.OverlayValues[431] = d431
				ps476.OverlayValues[432] = d432
				ps476.OverlayValues[433] = d433
				ps476.OverlayValues[435] = d435
				ps476.OverlayValues[436] = d436
				ps476.OverlayValues[437] = d437
				ps476.OverlayValues[439] = d439
				ps476.OverlayValues[440] = d440
				ps476.OverlayValues[441] = d441
				ps476.OverlayValues[442] = d442
				ps476.OverlayValues[443] = d443
				ps476.OverlayValues[444] = d444
				ps476.OverlayValues[445] = d445
				ps476.OverlayValues[446] = d446
				ps476.OverlayValues[447] = d447
				ps476.OverlayValues[448] = d448
				ps476.OverlayValues[449] = d449
				ps476.OverlayValues[450] = d450
				ps476.OverlayValues[451] = d451
				ps476.OverlayValues[452] = d452
				ps476.OverlayValues[453] = d453
				ps476.OverlayValues[454] = d454
				ps476.OverlayValues[455] = d455
				ps476.OverlayValues[456] = d456
				ps476.OverlayValues[457] = d457
				ps476.OverlayValues[458] = d458
				ps476.OverlayValues[459] = d459
				ps476.OverlayValues[460] = d460
				ps476.OverlayValues[461] = d461
				ps476.OverlayValues[462] = d462
				ps476.OverlayValues[463] = d463
				ps476.OverlayValues[464] = d464
				ps476.OverlayValues[465] = d465
				ps476.OverlayValues[466] = d466
				ps476.OverlayValues[467] = d467
				ps476.OverlayValues[468] = d468
				ps476.OverlayValues[469] = d469
				ps476.OverlayValues[470] = d470
				ps476.OverlayValues[471] = d471
				ps476.OverlayValues[472] = d472
				ps476.OverlayValues[473] = d473
				ps476.OverlayValues[474] = d474
				ps476.OverlayValues[475] = d475
				return bbs[7].RenderPS(ps476)
			}
			if ps.General {
			}
			ps477 := scm.PhiState{General: ps.General}
			ps477.OverlayValues = make([]scm.JITValueDesc, 476)
			ps477.OverlayValues[1] = d1
			ps477.OverlayValues[2] = d2
			ps477.OverlayValues[3] = d3
			ps477.OverlayValues[4] = d4
			ps477.OverlayValues[5] = d5
			ps477.OverlayValues[6] = d6
			ps477.OverlayValues[7] = d7
			ps477.OverlayValues[8] = d8
			ps477.OverlayValues[9] = d9
			ps477.OverlayValues[10] = d10
			ps477.OverlayValues[11] = d11
			ps477.OverlayValues[12] = d12
			ps477.OverlayValues[13] = d13
			ps477.OverlayValues[14] = d14
			ps477.OverlayValues[15] = d15
			ps477.OverlayValues[17] = d17
			ps477.OverlayValues[18] = d18
			ps477.OverlayValues[19] = d19
			ps477.OverlayValues[20] = d20
			ps477.OverlayValues[21] = d21
			ps477.OverlayValues[22] = d22
			ps477.OverlayValues[24] = d24
			ps477.OverlayValues[25] = d25
			ps477.OverlayValues[26] = d26
			ps477.OverlayValues[27] = d27
			ps477.OverlayValues[28] = d28
			ps477.OverlayValues[29] = d29
			ps477.OverlayValues[30] = d30
			ps477.OverlayValues[31] = d31
			ps477.OverlayValues[32] = d32
			ps477.OverlayValues[33] = d33
			ps477.OverlayValues[34] = d34
			ps477.OverlayValues[35] = d35
			ps477.OverlayValues[36] = d36
			ps477.OverlayValues[37] = d37
			ps477.OverlayValues[38] = d38
			ps477.OverlayValues[39] = d39
			ps477.OverlayValues[40] = d40
			ps477.OverlayValues[41] = d41
			ps477.OverlayValues[42] = d42
			ps477.OverlayValues[43] = d43
			ps477.OverlayValues[44] = d44
			ps477.OverlayValues[45] = d45
			ps477.OverlayValues[46] = d46
			ps477.OverlayValues[47] = d47
			ps477.OverlayValues[48] = d48
			ps477.OverlayValues[49] = d49
			ps477.OverlayValues[50] = d50
			ps477.OverlayValues[51] = d51
			ps477.OverlayValues[52] = d52
			ps477.OverlayValues[53] = d53
			ps477.OverlayValues[54] = d54
			ps477.OverlayValues[55] = d55
			ps477.OverlayValues[56] = d56
			ps477.OverlayValues[57] = d57
			ps477.OverlayValues[58] = d58
			ps477.OverlayValues[59] = d59
			ps477.OverlayValues[60] = d60
			ps477.OverlayValues[61] = d61
			ps477.OverlayValues[64] = d64
			ps477.OverlayValues[65] = d65
			ps477.OverlayValues[66] = d66
			ps477.OverlayValues[132] = d132
			ps477.OverlayValues[133] = d133
			ps477.OverlayValues[134] = d134
			ps477.OverlayValues[136] = d136
			ps477.OverlayValues[137] = d137
			ps477.OverlayValues[138] = d138
			ps477.OverlayValues[139] = d139
			ps477.OverlayValues[140] = d140
			ps477.OverlayValues[141] = d141
			ps477.OverlayValues[142] = d142
			ps477.OverlayValues[143] = d143
			ps477.OverlayValues[144] = d144
			ps477.OverlayValues[145] = d145
			ps477.OverlayValues[146] = d146
			ps477.OverlayValues[147] = d147
			ps477.OverlayValues[148] = d148
			ps477.OverlayValues[149] = d149
			ps477.OverlayValues[150] = d150
			ps477.OverlayValues[151] = d151
			ps477.OverlayValues[152] = d152
			ps477.OverlayValues[153] = d153
			ps477.OverlayValues[154] = d154
			ps477.OverlayValues[155] = d155
			ps477.OverlayValues[156] = d156
			ps477.OverlayValues[157] = d157
			ps477.OverlayValues[158] = d158
			ps477.OverlayValues[159] = d159
			ps477.OverlayValues[160] = d160
			ps477.OverlayValues[161] = d161
			ps477.OverlayValues[162] = d162
			ps477.OverlayValues[163] = d163
			ps477.OverlayValues[164] = d164
			ps477.OverlayValues[165] = d165
			ps477.OverlayValues[166] = d166
			ps477.OverlayValues[167] = d167
			ps477.OverlayValues[168] = d168
			ps477.OverlayValues[169] = d169
			ps477.OverlayValues[170] = d170
			ps477.OverlayValues[171] = d171
			ps477.OverlayValues[172] = d172
			ps477.OverlayValues[175] = d175
			ps477.OverlayValues[282] = d282
			ps477.OverlayValues[283] = d283
			ps477.OverlayValues[284] = d284
			ps477.OverlayValues[285] = d285
			ps477.OverlayValues[287] = d287
			ps477.OverlayValues[288] = d288
			ps477.OverlayValues[289] = d289
			ps477.OverlayValues[290] = d290
			ps477.OverlayValues[291] = d291
			ps477.OverlayValues[292] = d292
			ps477.OverlayValues[293] = d293
			ps477.OverlayValues[294] = d294
			ps477.OverlayValues[296] = d296
			ps477.OverlayValues[298] = d298
			ps477.OverlayValues[299] = d299
			ps477.OverlayValues[300] = d300
			ps477.OverlayValues[301] = d301
			ps477.OverlayValues[302] = d302
			ps477.OverlayValues[305] = d305
			ps477.OverlayValues[429] = d429
			ps477.OverlayValues[430] = d430
			ps477.OverlayValues[431] = d431
			ps477.OverlayValues[432] = d432
			ps477.OverlayValues[433] = d433
			ps477.OverlayValues[435] = d435
			ps477.OverlayValues[436] = d436
			ps477.OverlayValues[437] = d437
			ps477.OverlayValues[439] = d439
			ps477.OverlayValues[440] = d440
			ps477.OverlayValues[441] = d441
			ps477.OverlayValues[442] = d442
			ps477.OverlayValues[443] = d443
			ps477.OverlayValues[444] = d444
			ps477.OverlayValues[445] = d445
			ps477.OverlayValues[446] = d446
			ps477.OverlayValues[447] = d447
			ps477.OverlayValues[448] = d448
			ps477.OverlayValues[449] = d449
			ps477.OverlayValues[450] = d450
			ps477.OverlayValues[451] = d451
			ps477.OverlayValues[452] = d452
			ps477.OverlayValues[453] = d453
			ps477.OverlayValues[454] = d454
			ps477.OverlayValues[455] = d455
			ps477.OverlayValues[456] = d456
			ps477.OverlayValues[457] = d457
			ps477.OverlayValues[458] = d458
			ps477.OverlayValues[459] = d459
			ps477.OverlayValues[460] = d460
			ps477.OverlayValues[461] = d461
			ps477.OverlayValues[462] = d462
			ps477.OverlayValues[463] = d463
			ps477.OverlayValues[464] = d464
			ps477.OverlayValues[465] = d465
			ps477.OverlayValues[466] = d466
			ps477.OverlayValues[467] = d467
			ps477.OverlayValues[468] = d468
			ps477.OverlayValues[469] = d469
			ps477.OverlayValues[470] = d470
			ps477.OverlayValues[471] = d471
			ps477.OverlayValues[472] = d472
			ps477.OverlayValues[473] = d473
			ps477.OverlayValues[474] = d474
			ps477.OverlayValues[475] = d475
			return bbs[9].RenderPS(ps477)
		}
		if !ps.General {
			ps.General = true
			return bbs[6].RenderPS(ps)
		}
		lbl39 := ctx.ReserveLabel()
		lbl40 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d475.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl39)
		ctx.EmitJmp(lbl40)
		ctx.MarkLabel(lbl39)
		ctx.EmitJmp(lbl8)
		ctx.MarkLabel(lbl40)
		ctx.EmitJmp(lbl10)
		ps478 := scm.PhiState{General: true}
		ps478.OverlayValues = make([]scm.JITValueDesc, 476)
		ps478.OverlayValues[1] = d1
		ps478.OverlayValues[2] = d2
		ps478.OverlayValues[3] = d3
		ps478.OverlayValues[4] = d4
		ps478.OverlayValues[5] = d5
		ps478.OverlayValues[6] = d6
		ps478.OverlayValues[7] = d7
		ps478.OverlayValues[8] = d8
		ps478.OverlayValues[9] = d9
		ps478.OverlayValues[10] = d10
		ps478.OverlayValues[11] = d11
		ps478.OverlayValues[12] = d12
		ps478.OverlayValues[13] = d13
		ps478.OverlayValues[14] = d14
		ps478.OverlayValues[15] = d15
		ps478.OverlayValues[17] = d17
		ps478.OverlayValues[18] = d18
		ps478.OverlayValues[19] = d19
		ps478.OverlayValues[20] = d20
		ps478.OverlayValues[21] = d21
		ps478.OverlayValues[22] = d22
		ps478.OverlayValues[24] = d24
		ps478.OverlayValues[25] = d25
		ps478.OverlayValues[26] = d26
		ps478.OverlayValues[27] = d27
		ps478.OverlayValues[28] = d28
		ps478.OverlayValues[29] = d29
		ps478.OverlayValues[30] = d30
		ps478.OverlayValues[31] = d31
		ps478.OverlayValues[32] = d32
		ps478.OverlayValues[33] = d33
		ps478.OverlayValues[34] = d34
		ps478.OverlayValues[35] = d35
		ps478.OverlayValues[36] = d36
		ps478.OverlayValues[37] = d37
		ps478.OverlayValues[38] = d38
		ps478.OverlayValues[39] = d39
		ps478.OverlayValues[40] = d40
		ps478.OverlayValues[41] = d41
		ps478.OverlayValues[42] = d42
		ps478.OverlayValues[43] = d43
		ps478.OverlayValues[44] = d44
		ps478.OverlayValues[45] = d45
		ps478.OverlayValues[46] = d46
		ps478.OverlayValues[47] = d47
		ps478.OverlayValues[48] = d48
		ps478.OverlayValues[49] = d49
		ps478.OverlayValues[50] = d50
		ps478.OverlayValues[51] = d51
		ps478.OverlayValues[52] = d52
		ps478.OverlayValues[53] = d53
		ps478.OverlayValues[54] = d54
		ps478.OverlayValues[55] = d55
		ps478.OverlayValues[56] = d56
		ps478.OverlayValues[57] = d57
		ps478.OverlayValues[58] = d58
		ps478.OverlayValues[59] = d59
		ps478.OverlayValues[60] = d60
		ps478.OverlayValues[61] = d61
		ps478.OverlayValues[64] = d64
		ps478.OverlayValues[65] = d65
		ps478.OverlayValues[66] = d66
		ps478.OverlayValues[132] = d132
		ps478.OverlayValues[133] = d133
		ps478.OverlayValues[134] = d134
		ps478.OverlayValues[136] = d136
		ps478.OverlayValues[137] = d137
		ps478.OverlayValues[138] = d138
		ps478.OverlayValues[139] = d139
		ps478.OverlayValues[140] = d140
		ps478.OverlayValues[141] = d141
		ps478.OverlayValues[142] = d142
		ps478.OverlayValues[143] = d143
		ps478.OverlayValues[144] = d144
		ps478.OverlayValues[145] = d145
		ps478.OverlayValues[146] = d146
		ps478.OverlayValues[147] = d147
		ps478.OverlayValues[148] = d148
		ps478.OverlayValues[149] = d149
		ps478.OverlayValues[150] = d150
		ps478.OverlayValues[151] = d151
		ps478.OverlayValues[152] = d152
		ps478.OverlayValues[153] = d153
		ps478.OverlayValues[154] = d154
		ps478.OverlayValues[155] = d155
		ps478.OverlayValues[156] = d156
		ps478.OverlayValues[157] = d157
		ps478.OverlayValues[158] = d158
		ps478.OverlayValues[159] = d159
		ps478.OverlayValues[160] = d160
		ps478.OverlayValues[161] = d161
		ps478.OverlayValues[162] = d162
		ps478.OverlayValues[163] = d163
		ps478.OverlayValues[164] = d164
		ps478.OverlayValues[165] = d165
		ps478.OverlayValues[166] = d166
		ps478.OverlayValues[167] = d167
		ps478.OverlayValues[168] = d168
		ps478.OverlayValues[169] = d169
		ps478.OverlayValues[170] = d170
		ps478.OverlayValues[171] = d171
		ps478.OverlayValues[172] = d172
		ps478.OverlayValues[175] = d175
		ps478.OverlayValues[282] = d282
		ps478.OverlayValues[283] = d283
		ps478.OverlayValues[284] = d284
		ps478.OverlayValues[285] = d285
		ps478.OverlayValues[287] = d287
		ps478.OverlayValues[288] = d288
		ps478.OverlayValues[289] = d289
		ps478.OverlayValues[290] = d290
		ps478.OverlayValues[291] = d291
		ps478.OverlayValues[292] = d292
		ps478.OverlayValues[293] = d293
		ps478.OverlayValues[294] = d294
		ps478.OverlayValues[296] = d296
		ps478.OverlayValues[298] = d298
		ps478.OverlayValues[299] = d299
		ps478.OverlayValues[300] = d300
		ps478.OverlayValues[301] = d301
		ps478.OverlayValues[302] = d302
		ps478.OverlayValues[305] = d305
		ps478.OverlayValues[429] = d429
		ps478.OverlayValues[430] = d430
		ps478.OverlayValues[431] = d431
		ps478.OverlayValues[432] = d432
		ps478.OverlayValues[433] = d433
		ps478.OverlayValues[435] = d435
		ps478.OverlayValues[436] = d436
		ps478.OverlayValues[437] = d437
		ps478.OverlayValues[439] = d439
		ps478.OverlayValues[440] = d440
		ps478.OverlayValues[441] = d441
		ps478.OverlayValues[442] = d442
		ps478.OverlayValues[443] = d443
		ps478.OverlayValues[444] = d444
		ps478.OverlayValues[445] = d445
		ps478.OverlayValues[446] = d446
		ps478.OverlayValues[447] = d447
		ps478.OverlayValues[448] = d448
		ps478.OverlayValues[449] = d449
		ps478.OverlayValues[450] = d450
		ps478.OverlayValues[451] = d451
		ps478.OverlayValues[452] = d452
		ps478.OverlayValues[453] = d453
		ps478.OverlayValues[454] = d454
		ps478.OverlayValues[455] = d455
		ps478.OverlayValues[456] = d456
		ps478.OverlayValues[457] = d457
		ps478.OverlayValues[458] = d458
		ps478.OverlayValues[459] = d459
		ps478.OverlayValues[460] = d460
		ps478.OverlayValues[461] = d461
		ps478.OverlayValues[462] = d462
		ps478.OverlayValues[463] = d463
		ps478.OverlayValues[464] = d464
		ps478.OverlayValues[465] = d465
		ps478.OverlayValues[466] = d466
		ps478.OverlayValues[467] = d467
		ps478.OverlayValues[468] = d468
		ps478.OverlayValues[469] = d469
		ps478.OverlayValues[470] = d470
		ps478.OverlayValues[471] = d471
		ps478.OverlayValues[472] = d472
		ps478.OverlayValues[473] = d473
		ps478.OverlayValues[474] = d474
		ps478.OverlayValues[475] = d475
		ps479 := scm.PhiState{General: true}
		ps479.OverlayValues = make([]scm.JITValueDesc, 476)
		ps479.OverlayValues[1] = d1
		ps479.OverlayValues[2] = d2
		ps479.OverlayValues[3] = d3
		ps479.OverlayValues[4] = d4
		ps479.OverlayValues[5] = d5
		ps479.OverlayValues[6] = d6
		ps479.OverlayValues[7] = d7
		ps479.OverlayValues[8] = d8
		ps479.OverlayValues[9] = d9
		ps479.OverlayValues[10] = d10
		ps479.OverlayValues[11] = d11
		ps479.OverlayValues[12] = d12
		ps479.OverlayValues[13] = d13
		ps479.OverlayValues[14] = d14
		ps479.OverlayValues[15] = d15
		ps479.OverlayValues[17] = d17
		ps479.OverlayValues[18] = d18
		ps479.OverlayValues[19] = d19
		ps479.OverlayValues[20] = d20
		ps479.OverlayValues[21] = d21
		ps479.OverlayValues[22] = d22
		ps479.OverlayValues[24] = d24
		ps479.OverlayValues[25] = d25
		ps479.OverlayValues[26] = d26
		ps479.OverlayValues[27] = d27
		ps479.OverlayValues[28] = d28
		ps479.OverlayValues[29] = d29
		ps479.OverlayValues[30] = d30
		ps479.OverlayValues[31] = d31
		ps479.OverlayValues[32] = d32
		ps479.OverlayValues[33] = d33
		ps479.OverlayValues[34] = d34
		ps479.OverlayValues[35] = d35
		ps479.OverlayValues[36] = d36
		ps479.OverlayValues[37] = d37
		ps479.OverlayValues[38] = d38
		ps479.OverlayValues[39] = d39
		ps479.OverlayValues[40] = d40
		ps479.OverlayValues[41] = d41
		ps479.OverlayValues[42] = d42
		ps479.OverlayValues[43] = d43
		ps479.OverlayValues[44] = d44
		ps479.OverlayValues[45] = d45
		ps479.OverlayValues[46] = d46
		ps479.OverlayValues[47] = d47
		ps479.OverlayValues[48] = d48
		ps479.OverlayValues[49] = d49
		ps479.OverlayValues[50] = d50
		ps479.OverlayValues[51] = d51
		ps479.OverlayValues[52] = d52
		ps479.OverlayValues[53] = d53
		ps479.OverlayValues[54] = d54
		ps479.OverlayValues[55] = d55
		ps479.OverlayValues[56] = d56
		ps479.OverlayValues[57] = d57
		ps479.OverlayValues[58] = d58
		ps479.OverlayValues[59] = d59
		ps479.OverlayValues[60] = d60
		ps479.OverlayValues[61] = d61
		ps479.OverlayValues[64] = d64
		ps479.OverlayValues[65] = d65
		ps479.OverlayValues[66] = d66
		ps479.OverlayValues[132] = d132
		ps479.OverlayValues[133] = d133
		ps479.OverlayValues[134] = d134
		ps479.OverlayValues[136] = d136
		ps479.OverlayValues[137] = d137
		ps479.OverlayValues[138] = d138
		ps479.OverlayValues[139] = d139
		ps479.OverlayValues[140] = d140
		ps479.OverlayValues[141] = d141
		ps479.OverlayValues[142] = d142
		ps479.OverlayValues[143] = d143
		ps479.OverlayValues[144] = d144
		ps479.OverlayValues[145] = d145
		ps479.OverlayValues[146] = d146
		ps479.OverlayValues[147] = d147
		ps479.OverlayValues[148] = d148
		ps479.OverlayValues[149] = d149
		ps479.OverlayValues[150] = d150
		ps479.OverlayValues[151] = d151
		ps479.OverlayValues[152] = d152
		ps479.OverlayValues[153] = d153
		ps479.OverlayValues[154] = d154
		ps479.OverlayValues[155] = d155
		ps479.OverlayValues[156] = d156
		ps479.OverlayValues[157] = d157
		ps479.OverlayValues[158] = d158
		ps479.OverlayValues[159] = d159
		ps479.OverlayValues[160] = d160
		ps479.OverlayValues[161] = d161
		ps479.OverlayValues[162] = d162
		ps479.OverlayValues[163] = d163
		ps479.OverlayValues[164] = d164
		ps479.OverlayValues[165] = d165
		ps479.OverlayValues[166] = d166
		ps479.OverlayValues[167] = d167
		ps479.OverlayValues[168] = d168
		ps479.OverlayValues[169] = d169
		ps479.OverlayValues[170] = d170
		ps479.OverlayValues[171] = d171
		ps479.OverlayValues[172] = d172
		ps479.OverlayValues[175] = d175
		ps479.OverlayValues[282] = d282
		ps479.OverlayValues[283] = d283
		ps479.OverlayValues[284] = d284
		ps479.OverlayValues[285] = d285
		ps479.OverlayValues[287] = d287
		ps479.OverlayValues[288] = d288
		ps479.OverlayValues[289] = d289
		ps479.OverlayValues[290] = d290
		ps479.OverlayValues[291] = d291
		ps479.OverlayValues[292] = d292
		ps479.OverlayValues[293] = d293
		ps479.OverlayValues[294] = d294
		ps479.OverlayValues[296] = d296
		ps479.OverlayValues[298] = d298
		ps479.OverlayValues[299] = d299
		ps479.OverlayValues[300] = d300
		ps479.OverlayValues[301] = d301
		ps479.OverlayValues[302] = d302
		ps479.OverlayValues[305] = d305
		ps479.OverlayValues[429] = d429
		ps479.OverlayValues[430] = d430
		ps479.OverlayValues[431] = d431
		ps479.OverlayValues[432] = d432
		ps479.OverlayValues[433] = d433
		ps479.OverlayValues[435] = d435
		ps479.OverlayValues[436] = d436
		ps479.OverlayValues[437] = d437
		ps479.OverlayValues[439] = d439
		ps479.OverlayValues[440] = d440
		ps479.OverlayValues[441] = d441
		ps479.OverlayValues[442] = d442
		ps479.OverlayValues[443] = d443
		ps479.OverlayValues[444] = d444
		ps479.OverlayValues[445] = d445
		ps479.OverlayValues[446] = d446
		ps479.OverlayValues[447] = d447
		ps479.OverlayValues[448] = d448
		ps479.OverlayValues[449] = d449
		ps479.OverlayValues[450] = d450
		ps479.OverlayValues[451] = d451
		ps479.OverlayValues[452] = d452
		ps479.OverlayValues[453] = d453
		ps479.OverlayValues[454] = d454
		ps479.OverlayValues[455] = d455
		ps479.OverlayValues[456] = d456
		ps479.OverlayValues[457] = d457
		ps479.OverlayValues[458] = d458
		ps479.OverlayValues[459] = d459
		ps479.OverlayValues[460] = d460
		ps479.OverlayValues[461] = d461
		ps479.OverlayValues[462] = d462
		ps479.OverlayValues[463] = d463
		ps479.OverlayValues[464] = d464
		ps479.OverlayValues[465] = d465
		ps479.OverlayValues[466] = d466
		ps479.OverlayValues[467] = d467
		ps479.OverlayValues[468] = d468
		ps479.OverlayValues[469] = d469
		ps479.OverlayValues[470] = d470
		ps479.OverlayValues[471] = d471
		ps479.OverlayValues[472] = d472
		ps479.OverlayValues[473] = d473
		ps479.OverlayValues[474] = d474
		ps479.OverlayValues[475] = d475
		snap480 := d1
		snap481 := d2
		snap482 := d3
		snap483 := d4
		snap484 := d5
		snap485 := d6
		snap486 := d7
		snap487 := d8
		snap488 := d9
		snap489 := d10
		snap490 := d11
		snap491 := d12
		snap492 := d13
		snap493 := d14
		snap494 := d15
		snap495 := d17
		snap496 := d18
		snap497 := d19
		snap498 := d20
		snap499 := d21
		snap500 := d22
		snap501 := d24
		snap502 := d25
		snap503 := d26
		snap504 := d27
		snap505 := d28
		snap506 := d29
		snap507 := d30
		snap508 := d31
		snap509 := d32
		snap510 := d33
		snap511 := d34
		snap512 := d35
		snap513 := d36
		snap514 := d37
		snap515 := d38
		snap516 := d39
		snap517 := d40
		snap518 := d41
		snap519 := d42
		snap520 := d43
		snap521 := d44
		snap522 := d45
		snap523 := d46
		snap524 := d47
		snap525 := d48
		snap526 := d49
		snap527 := d50
		snap528 := d51
		snap529 := d52
		snap530 := d53
		snap531 := d54
		snap532 := d55
		snap533 := d56
		snap534 := d57
		snap535 := d58
		snap536 := d59
		snap537 := d60
		snap538 := d61
		snap539 := d64
		snap540 := d65
		snap541 := d66
		snap542 := d132
		snap543 := d133
		snap544 := d134
		snap545 := d136
		snap546 := d137
		snap547 := d138
		snap548 := d139
		snap549 := d140
		snap550 := d141
		snap551 := d142
		snap552 := d143
		snap553 := d144
		snap554 := d145
		snap555 := d146
		snap556 := d147
		snap557 := d148
		snap558 := d149
		snap559 := d150
		snap560 := d151
		snap561 := d152
		snap562 := d153
		snap563 := d154
		snap564 := d155
		snap565 := d156
		snap566 := d157
		snap567 := d158
		snap568 := d159
		snap569 := d160
		snap570 := d161
		snap571 := d162
		snap572 := d163
		snap573 := d164
		snap574 := d165
		snap575 := d166
		snap576 := d167
		snap577 := d168
		snap578 := d169
		snap579 := d170
		snap580 := d171
		snap581 := d172
		snap582 := d175
		snap583 := d282
		snap584 := d283
		snap585 := d284
		snap586 := d285
		snap587 := d287
		snap588 := d288
		snap589 := d289
		snap590 := d290
		snap591 := d291
		snap592 := d292
		snap593 := d293
		snap594 := d294
		snap595 := d296
		snap596 := d298
		snap597 := d299
		snap598 := d300
		snap599 := d301
		snap600 := d302
		snap601 := d305
		snap602 := d429
		snap603 := d430
		snap604 := d431
		snap605 := d432
		snap606 := d433
		snap607 := d435
		snap608 := d436
		snap609 := d437
		snap610 := d439
		snap611 := d440
		snap612 := d441
		snap613 := d442
		snap614 := d443
		snap615 := d444
		snap616 := d445
		snap617 := d446
		snap618 := d447
		snap619 := d448
		snap620 := d449
		snap621 := d450
		snap622 := d451
		snap623 := d452
		snap624 := d453
		snap625 := d454
		snap626 := d455
		snap627 := d456
		snap628 := d457
		snap629 := d458
		snap630 := d459
		snap631 := d460
		snap632 := d461
		snap633 := d462
		snap634 := d463
		snap635 := d464
		snap636 := d465
		snap637 := d466
		snap638 := d467
		snap639 := d468
		snap640 := d469
		snap641 := d470
		snap642 := d471
		snap643 := d472
		snap644 := d473
		snap645 := d474
		snap646 := d475
		alloc647 := ctx.SnapshotAllocState()
		if !bbs[9].Rendered {
			bbs[9].RenderPS(ps479)
		}
		ctx.RestoreAllocState(alloc647)
		d1 = snap480
		d2 = snap481
		d3 = snap482
		d4 = snap483
		d5 = snap484
		d6 = snap485
		d7 = snap486
		d8 = snap487
		d9 = snap488
		d10 = snap489
		d11 = snap490
		d12 = snap491
		d13 = snap492
		d14 = snap493
		d15 = snap494
		d17 = snap495
		d18 = snap496
		d19 = snap497
		d20 = snap498
		d21 = snap499
		d22 = snap500
		d24 = snap501
		d25 = snap502
		d26 = snap503
		d27 = snap504
		d28 = snap505
		d29 = snap506
		d30 = snap507
		d31 = snap508
		d32 = snap509
		d33 = snap510
		d34 = snap511
		d35 = snap512
		d36 = snap513
		d37 = snap514
		d38 = snap515
		d39 = snap516
		d40 = snap517
		d41 = snap518
		d42 = snap519
		d43 = snap520
		d44 = snap521
		d45 = snap522
		d46 = snap523
		d47 = snap524
		d48 = snap525
		d49 = snap526
		d50 = snap527
		d51 = snap528
		d52 = snap529
		d53 = snap530
		d54 = snap531
		d55 = snap532
		d56 = snap533
		d57 = snap534
		d58 = snap535
		d59 = snap536
		d60 = snap537
		d61 = snap538
		d64 = snap539
		d65 = snap540
		d66 = snap541
		d132 = snap542
		d133 = snap543
		d134 = snap544
		d136 = snap545
		d137 = snap546
		d138 = snap547
		d139 = snap548
		d140 = snap549
		d141 = snap550
		d142 = snap551
		d143 = snap552
		d144 = snap553
		d145 = snap554
		d146 = snap555
		d147 = snap556
		d148 = snap557
		d149 = snap558
		d150 = snap559
		d151 = snap560
		d152 = snap561
		d153 = snap562
		d154 = snap563
		d155 = snap564
		d156 = snap565
		d157 = snap566
		d158 = snap567
		d159 = snap568
		d160 = snap569
		d161 = snap570
		d162 = snap571
		d163 = snap572
		d164 = snap573
		d165 = snap574
		d166 = snap575
		d167 = snap576
		d168 = snap577
		d169 = snap578
		d170 = snap579
		d171 = snap580
		d172 = snap581
		d175 = snap582
		d282 = snap583
		d283 = snap584
		d284 = snap585
		d285 = snap586
		d287 = snap587
		d288 = snap588
		d289 = snap589
		d290 = snap590
		d291 = snap591
		d292 = snap592
		d293 = snap593
		d294 = snap594
		d296 = snap595
		d298 = snap596
		d299 = snap597
		d300 = snap598
		d301 = snap599
		d302 = snap600
		d305 = snap601
		d429 = snap602
		d430 = snap603
		d431 = snap604
		d432 = snap605
		d433 = snap606
		d435 = snap607
		d436 = snap608
		d437 = snap609
		d439 = snap610
		d440 = snap611
		d441 = snap612
		d442 = snap613
		d443 = snap614
		d444 = snap615
		d445 = snap616
		d446 = snap617
		d447 = snap618
		d448 = snap619
		d449 = snap620
		d450 = snap621
		d451 = snap622
		d452 = snap623
		d453 = snap624
		d454 = snap625
		d455 = snap626
		d456 = snap627
		d457 = snap628
		d458 = snap629
		d459 = snap630
		d460 = snap631
		d461 = snap632
		d462 = snap633
		d463 = snap634
		d464 = snap635
		d465 = snap636
		d466 = snap637
		d467 = snap638
		d468 = snap639
		d469 = snap640
		d470 = snap641
		d471 = snap642
		d472 = snap643
		d473 = snap644
		d474 = snap645
		d475 = snap646
		if !bbs[7].Rendered {
			return bbs[7].RenderPS(ps478)
		}
		return result
		ctx.FreeDesc(&d474)
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
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
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
		if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != scm.LocNone {
			d282 = ps.OverlayValues[282]
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
		if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
			d294 = ps.OverlayValues[294]
		}
		if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != scm.LocNone {
			d296 = ps.OverlayValues[296]
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
		if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != scm.LocNone {
			d305 = ps.OverlayValues[305]
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
		if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != scm.LocNone {
			d435 = ps.OverlayValues[435]
		}
		if len(ps.OverlayValues) > 436 && ps.OverlayValues[436].Loc != scm.LocNone {
			d436 = ps.OverlayValues[436]
		}
		if len(ps.OverlayValues) > 437 && ps.OverlayValues[437].Loc != scm.LocNone {
			d437 = ps.OverlayValues[437]
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d5)
		var d648 scm.JITValueDesc
		if d5.Loc == scm.LocImm {
			d648 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d648 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d648)
		}
		if d648.Loc == scm.LocImm {
			d648 = scm.JITValueDesc{Loc: scm.LocImm, Type: d648.Type, Imm: scm.NewInt(int64(uint64(d648.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d648.Reg, 32)
			ctx.EmitShrRegImm8(d648.Reg, 32)
		}
		if d648.Loc == scm.LocReg && d5.Loc == scm.LocReg && d648.Reg == d5.Reg {
			ctx.TransferReg(d5.Reg)
			d5.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d648)
		ctx.EmitStoreToStack(d648, int32(bbs[8].PhiBase)+int32(16))
		ctx.StabilizeDescForControlFlow(&d648)
		if ps.General {
			ctx.SyncDesc(&d6)
			if d6.Loc == scm.LocReg {
				ctx.ProtectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.ProtectReg(d6.Reg)
				ctx.ProtectReg(d6.Reg2)
			}
			d649 = d6
			if d649.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d649)
			d650 = d649
			if d650.Loc == scm.LocImm {
				d650 = scm.JITValueDesc{Loc: scm.LocImm, Type: d650.Type, Imm: scm.NewInt(int64(uint64(d650.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d650.Reg, 32)
				ctx.EmitShrRegImm8(d650.Reg, 32)
			}
			ctx.EmitStoreToStack(d650, int32(bbs[8].PhiBase)+int32(0))
			if d6.Loc == scm.LocReg {
				ctx.UnprotectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d6.Reg)
				ctx.UnprotectReg(d6.Reg2)
			}
		}
		ps651 := scm.PhiState{General: ps.General}
		ps651.OverlayValues = make([]scm.JITValueDesc, 651)
		ps651.OverlayValues[1] = d1
		ps651.OverlayValues[2] = d2
		ps651.OverlayValues[3] = d3
		ps651.OverlayValues[4] = d4
		ps651.OverlayValues[5] = d5
		ps651.OverlayValues[6] = d6
		ps651.OverlayValues[7] = d7
		ps651.OverlayValues[8] = d8
		ps651.OverlayValues[9] = d9
		ps651.OverlayValues[10] = d10
		ps651.OverlayValues[11] = d11
		ps651.OverlayValues[12] = d12
		ps651.OverlayValues[13] = d13
		ps651.OverlayValues[14] = d14
		ps651.OverlayValues[15] = d15
		ps651.OverlayValues[17] = d17
		ps651.OverlayValues[18] = d18
		ps651.OverlayValues[19] = d19
		ps651.OverlayValues[20] = d20
		ps651.OverlayValues[21] = d21
		ps651.OverlayValues[22] = d22
		ps651.OverlayValues[24] = d24
		ps651.OverlayValues[25] = d25
		ps651.OverlayValues[26] = d26
		ps651.OverlayValues[27] = d27
		ps651.OverlayValues[28] = d28
		ps651.OverlayValues[29] = d29
		ps651.OverlayValues[30] = d30
		ps651.OverlayValues[31] = d31
		ps651.OverlayValues[32] = d32
		ps651.OverlayValues[33] = d33
		ps651.OverlayValues[34] = d34
		ps651.OverlayValues[35] = d35
		ps651.OverlayValues[36] = d36
		ps651.OverlayValues[37] = d37
		ps651.OverlayValues[38] = d38
		ps651.OverlayValues[39] = d39
		ps651.OverlayValues[40] = d40
		ps651.OverlayValues[41] = d41
		ps651.OverlayValues[42] = d42
		ps651.OverlayValues[43] = d43
		ps651.OverlayValues[44] = d44
		ps651.OverlayValues[45] = d45
		ps651.OverlayValues[46] = d46
		ps651.OverlayValues[47] = d47
		ps651.OverlayValues[48] = d48
		ps651.OverlayValues[49] = d49
		ps651.OverlayValues[50] = d50
		ps651.OverlayValues[51] = d51
		ps651.OverlayValues[52] = d52
		ps651.OverlayValues[53] = d53
		ps651.OverlayValues[54] = d54
		ps651.OverlayValues[55] = d55
		ps651.OverlayValues[56] = d56
		ps651.OverlayValues[57] = d57
		ps651.OverlayValues[58] = d58
		ps651.OverlayValues[59] = d59
		ps651.OverlayValues[60] = d60
		ps651.OverlayValues[61] = d61
		ps651.OverlayValues[64] = d64
		ps651.OverlayValues[65] = d65
		ps651.OverlayValues[66] = d66
		ps651.OverlayValues[132] = d132
		ps651.OverlayValues[133] = d133
		ps651.OverlayValues[134] = d134
		ps651.OverlayValues[136] = d136
		ps651.OverlayValues[137] = d137
		ps651.OverlayValues[138] = d138
		ps651.OverlayValues[139] = d139
		ps651.OverlayValues[140] = d140
		ps651.OverlayValues[141] = d141
		ps651.OverlayValues[142] = d142
		ps651.OverlayValues[143] = d143
		ps651.OverlayValues[144] = d144
		ps651.OverlayValues[145] = d145
		ps651.OverlayValues[146] = d146
		ps651.OverlayValues[147] = d147
		ps651.OverlayValues[148] = d148
		ps651.OverlayValues[149] = d149
		ps651.OverlayValues[150] = d150
		ps651.OverlayValues[151] = d151
		ps651.OverlayValues[152] = d152
		ps651.OverlayValues[153] = d153
		ps651.OverlayValues[154] = d154
		ps651.OverlayValues[155] = d155
		ps651.OverlayValues[156] = d156
		ps651.OverlayValues[157] = d157
		ps651.OverlayValues[158] = d158
		ps651.OverlayValues[159] = d159
		ps651.OverlayValues[160] = d160
		ps651.OverlayValues[161] = d161
		ps651.OverlayValues[162] = d162
		ps651.OverlayValues[163] = d163
		ps651.OverlayValues[164] = d164
		ps651.OverlayValues[165] = d165
		ps651.OverlayValues[166] = d166
		ps651.OverlayValues[167] = d167
		ps651.OverlayValues[168] = d168
		ps651.OverlayValues[169] = d169
		ps651.OverlayValues[170] = d170
		ps651.OverlayValues[171] = d171
		ps651.OverlayValues[172] = d172
		ps651.OverlayValues[175] = d175
		ps651.OverlayValues[282] = d282
		ps651.OverlayValues[283] = d283
		ps651.OverlayValues[284] = d284
		ps651.OverlayValues[285] = d285
		ps651.OverlayValues[287] = d287
		ps651.OverlayValues[288] = d288
		ps651.OverlayValues[289] = d289
		ps651.OverlayValues[290] = d290
		ps651.OverlayValues[291] = d291
		ps651.OverlayValues[292] = d292
		ps651.OverlayValues[293] = d293
		ps651.OverlayValues[294] = d294
		ps651.OverlayValues[296] = d296
		ps651.OverlayValues[298] = d298
		ps651.OverlayValues[299] = d299
		ps651.OverlayValues[300] = d300
		ps651.OverlayValues[301] = d301
		ps651.OverlayValues[302] = d302
		ps651.OverlayValues[305] = d305
		ps651.OverlayValues[429] = d429
		ps651.OverlayValues[430] = d430
		ps651.OverlayValues[431] = d431
		ps651.OverlayValues[432] = d432
		ps651.OverlayValues[433] = d433
		ps651.OverlayValues[435] = d435
		ps651.OverlayValues[436] = d436
		ps651.OverlayValues[437] = d437
		ps651.OverlayValues[439] = d439
		ps651.OverlayValues[440] = d440
		ps651.OverlayValues[441] = d441
		ps651.OverlayValues[442] = d442
		ps651.OverlayValues[443] = d443
		ps651.OverlayValues[444] = d444
		ps651.OverlayValues[445] = d445
		ps651.OverlayValues[446] = d446
		ps651.OverlayValues[447] = d447
		ps651.OverlayValues[448] = d448
		ps651.OverlayValues[449] = d449
		ps651.OverlayValues[450] = d450
		ps651.OverlayValues[451] = d451
		ps651.OverlayValues[452] = d452
		ps651.OverlayValues[453] = d453
		ps651.OverlayValues[454] = d454
		ps651.OverlayValues[455] = d455
		ps651.OverlayValues[456] = d456
		ps651.OverlayValues[457] = d457
		ps651.OverlayValues[458] = d458
		ps651.OverlayValues[459] = d459
		ps651.OverlayValues[460] = d460
		ps651.OverlayValues[461] = d461
		ps651.OverlayValues[462] = d462
		ps651.OverlayValues[463] = d463
		ps651.OverlayValues[464] = d464
		ps651.OverlayValues[465] = d465
		ps651.OverlayValues[466] = d466
		ps651.OverlayValues[467] = d467
		ps651.OverlayValues[468] = d468
		ps651.OverlayValues[469] = d469
		ps651.OverlayValues[470] = d470
		ps651.OverlayValues[471] = d471
		ps651.OverlayValues[472] = d472
		ps651.OverlayValues[473] = d473
		ps651.OverlayValues[474] = d474
		ps651.OverlayValues[475] = d475
		ps651.OverlayValues[648] = d648
		ps651.OverlayValues[649] = d649
		ps651.OverlayValues[650] = d650
		ps651.PhiValues = make([]scm.JITValueDesc, 2)
		d652 = d6
		ps651.PhiValues[0] = d652
		if ps651.General && bbs[8].Rendered {
			ctx.EmitJmp(lbl9)
			return result
		}
		return bbs[8].RenderPS(ps651)
		return result
	}
	bbs[8].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d653 := ps.PhiValues[0]
				ctx.EnsureDesc(&d653)
				ctx.EmitStoreToStack(d653, int32(bbs[8].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d654 := ps.PhiValues[1]
				ctx.EnsureDesc(&d654)
				ctx.EmitStoreToStack(d654, int32(bbs[8].PhiBase)+int32(16))
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
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
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
		if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != scm.LocNone {
			d282 = ps.OverlayValues[282]
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
		if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
			d294 = ps.OverlayValues[294]
		}
		if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != scm.LocNone {
			d296 = ps.OverlayValues[296]
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
		if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != scm.LocNone {
			d305 = ps.OverlayValues[305]
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
		if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != scm.LocNone {
			d435 = ps.OverlayValues[435]
		}
		if len(ps.OverlayValues) > 436 && ps.OverlayValues[436].Loc != scm.LocNone {
			d436 = ps.OverlayValues[436]
		}
		if len(ps.OverlayValues) > 437 && ps.OverlayValues[437].Loc != scm.LocNone {
			d437 = ps.OverlayValues[437]
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
		if len(ps.OverlayValues) > 648 && ps.OverlayValues[648].Loc != scm.LocNone {
			d648 = ps.OverlayValues[648]
		}
		if len(ps.OverlayValues) > 649 && ps.OverlayValues[649].Loc != scm.LocNone {
			d649 = ps.OverlayValues[649]
		}
		if len(ps.OverlayValues) > 650 && ps.OverlayValues[650].Loc != scm.LocNone {
			d650 = ps.OverlayValues[650]
		}
		if len(ps.OverlayValues) > 652 && ps.OverlayValues[652].Loc != scm.LocNone {
			d652 = ps.OverlayValues[652]
		}
		if len(ps.OverlayValues) > 653 && ps.OverlayValues[653].Loc != scm.LocNone {
			d653 = ps.OverlayValues[653]
		}
		if len(ps.OverlayValues) > 654 && ps.OverlayValues[654].Loc != scm.LocNone {
			d654 = ps.OverlayValues[654]
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
		var d655 scm.JITValueDesc
		if d8.Loc == scm.LocImm && d9.Loc == scm.LocImm {
			d655 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d8.Imm.Int()) == uint64(d9.Imm.Int()))}
		} else if d9.Loc == scm.LocImm {
			r138 := ctx.AllocRegExcept(d8.Reg)
			if d9.Imm.Int() >= -2147483648 && d9.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d8.Reg, int32(d9.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d9.Imm.Int()))
				ctx.EmitCmpInt64(d8.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r138, scm.CondEqual)
			d655 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r138}
			ctx.BindReg(r138, &d655)
		} else if d8.Loc == scm.LocImm {
			r139 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d8.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d9.Reg)
			ctx.EmitSetcc(r139, scm.CondEqual)
			d655 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r139}
			ctx.BindReg(r139, &d655)
		} else {
			r140 := ctx.AllocRegExcept(d8.Reg)
			ctx.EmitCmpInt64(d8.Reg, d9.Reg)
			ctx.EmitSetcc(r140, scm.CondEqual)
			d655 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r140}
			ctx.BindReg(r140, &d655)
		}
		d656 = d655
		ctx.EnsureDesc(&d656)
		if d656.Loc != scm.LocImm && d656.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d656.Loc == scm.LocImm {
			if d656.Imm.Bool() {
				if ps.General {
					ctx.SyncDesc(&d8)
					if d8.Loc == scm.LocReg {
						ctx.ProtectReg(d8.Reg)
					} else if d8.Loc == scm.LocRegPair {
						ctx.ProtectReg(d8.Reg)
						ctx.ProtectReg(d8.Reg2)
					}
					d657 = d8
					if d657.Loc == scm.LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d657)
					d658 = d657
					if d658.Loc == scm.LocImm {
						d658 = scm.JITValueDesc{Loc: scm.LocImm, Type: d658.Type, Imm: scm.NewInt(int64(uint64(d658.Imm.Int()) & 0xffffffff))}
					} else {
						ctx.EmitShlRegImm8(d658.Reg, 32)
						ctx.EmitShrRegImm8(d658.Reg, 32)
					}
					ctx.EmitStoreToStack(d658, int32(bbs[2].PhiBase)+int32(0))
					if d8.Loc == scm.LocReg {
						ctx.UnprotectReg(d8.Reg)
					} else if d8.Loc == scm.LocRegPair {
						ctx.UnprotectReg(d8.Reg)
						ctx.UnprotectReg(d8.Reg2)
					}
				}
				ps659 := scm.PhiState{General: ps.General}
				ps659.OverlayValues = make([]scm.JITValueDesc, 659)
				ps659.OverlayValues[1] = d1
				ps659.OverlayValues[2] = d2
				ps659.OverlayValues[3] = d3
				ps659.OverlayValues[4] = d4
				ps659.OverlayValues[5] = d5
				ps659.OverlayValues[6] = d6
				ps659.OverlayValues[7] = d7
				ps659.OverlayValues[8] = d8
				ps659.OverlayValues[9] = d9
				ps659.OverlayValues[10] = d10
				ps659.OverlayValues[11] = d11
				ps659.OverlayValues[12] = d12
				ps659.OverlayValues[13] = d13
				ps659.OverlayValues[14] = d14
				ps659.OverlayValues[15] = d15
				ps659.OverlayValues[17] = d17
				ps659.OverlayValues[18] = d18
				ps659.OverlayValues[19] = d19
				ps659.OverlayValues[20] = d20
				ps659.OverlayValues[21] = d21
				ps659.OverlayValues[22] = d22
				ps659.OverlayValues[24] = d24
				ps659.OverlayValues[25] = d25
				ps659.OverlayValues[26] = d26
				ps659.OverlayValues[27] = d27
				ps659.OverlayValues[28] = d28
				ps659.OverlayValues[29] = d29
				ps659.OverlayValues[30] = d30
				ps659.OverlayValues[31] = d31
				ps659.OverlayValues[32] = d32
				ps659.OverlayValues[33] = d33
				ps659.OverlayValues[34] = d34
				ps659.OverlayValues[35] = d35
				ps659.OverlayValues[36] = d36
				ps659.OverlayValues[37] = d37
				ps659.OverlayValues[38] = d38
				ps659.OverlayValues[39] = d39
				ps659.OverlayValues[40] = d40
				ps659.OverlayValues[41] = d41
				ps659.OverlayValues[42] = d42
				ps659.OverlayValues[43] = d43
				ps659.OverlayValues[44] = d44
				ps659.OverlayValues[45] = d45
				ps659.OverlayValues[46] = d46
				ps659.OverlayValues[47] = d47
				ps659.OverlayValues[48] = d48
				ps659.OverlayValues[49] = d49
				ps659.OverlayValues[50] = d50
				ps659.OverlayValues[51] = d51
				ps659.OverlayValues[52] = d52
				ps659.OverlayValues[53] = d53
				ps659.OverlayValues[54] = d54
				ps659.OverlayValues[55] = d55
				ps659.OverlayValues[56] = d56
				ps659.OverlayValues[57] = d57
				ps659.OverlayValues[58] = d58
				ps659.OverlayValues[59] = d59
				ps659.OverlayValues[60] = d60
				ps659.OverlayValues[61] = d61
				ps659.OverlayValues[64] = d64
				ps659.OverlayValues[65] = d65
				ps659.OverlayValues[66] = d66
				ps659.OverlayValues[132] = d132
				ps659.OverlayValues[133] = d133
				ps659.OverlayValues[134] = d134
				ps659.OverlayValues[136] = d136
				ps659.OverlayValues[137] = d137
				ps659.OverlayValues[138] = d138
				ps659.OverlayValues[139] = d139
				ps659.OverlayValues[140] = d140
				ps659.OverlayValues[141] = d141
				ps659.OverlayValues[142] = d142
				ps659.OverlayValues[143] = d143
				ps659.OverlayValues[144] = d144
				ps659.OverlayValues[145] = d145
				ps659.OverlayValues[146] = d146
				ps659.OverlayValues[147] = d147
				ps659.OverlayValues[148] = d148
				ps659.OverlayValues[149] = d149
				ps659.OverlayValues[150] = d150
				ps659.OverlayValues[151] = d151
				ps659.OverlayValues[152] = d152
				ps659.OverlayValues[153] = d153
				ps659.OverlayValues[154] = d154
				ps659.OverlayValues[155] = d155
				ps659.OverlayValues[156] = d156
				ps659.OverlayValues[157] = d157
				ps659.OverlayValues[158] = d158
				ps659.OverlayValues[159] = d159
				ps659.OverlayValues[160] = d160
				ps659.OverlayValues[161] = d161
				ps659.OverlayValues[162] = d162
				ps659.OverlayValues[163] = d163
				ps659.OverlayValues[164] = d164
				ps659.OverlayValues[165] = d165
				ps659.OverlayValues[166] = d166
				ps659.OverlayValues[167] = d167
				ps659.OverlayValues[168] = d168
				ps659.OverlayValues[169] = d169
				ps659.OverlayValues[170] = d170
				ps659.OverlayValues[171] = d171
				ps659.OverlayValues[172] = d172
				ps659.OverlayValues[175] = d175
				ps659.OverlayValues[282] = d282
				ps659.OverlayValues[283] = d283
				ps659.OverlayValues[284] = d284
				ps659.OverlayValues[285] = d285
				ps659.OverlayValues[287] = d287
				ps659.OverlayValues[288] = d288
				ps659.OverlayValues[289] = d289
				ps659.OverlayValues[290] = d290
				ps659.OverlayValues[291] = d291
				ps659.OverlayValues[292] = d292
				ps659.OverlayValues[293] = d293
				ps659.OverlayValues[294] = d294
				ps659.OverlayValues[296] = d296
				ps659.OverlayValues[298] = d298
				ps659.OverlayValues[299] = d299
				ps659.OverlayValues[300] = d300
				ps659.OverlayValues[301] = d301
				ps659.OverlayValues[302] = d302
				ps659.OverlayValues[305] = d305
				ps659.OverlayValues[429] = d429
				ps659.OverlayValues[430] = d430
				ps659.OverlayValues[431] = d431
				ps659.OverlayValues[432] = d432
				ps659.OverlayValues[433] = d433
				ps659.OverlayValues[435] = d435
				ps659.OverlayValues[436] = d436
				ps659.OverlayValues[437] = d437
				ps659.OverlayValues[439] = d439
				ps659.OverlayValues[440] = d440
				ps659.OverlayValues[441] = d441
				ps659.OverlayValues[442] = d442
				ps659.OverlayValues[443] = d443
				ps659.OverlayValues[444] = d444
				ps659.OverlayValues[445] = d445
				ps659.OverlayValues[446] = d446
				ps659.OverlayValues[447] = d447
				ps659.OverlayValues[448] = d448
				ps659.OverlayValues[449] = d449
				ps659.OverlayValues[450] = d450
				ps659.OverlayValues[451] = d451
				ps659.OverlayValues[452] = d452
				ps659.OverlayValues[453] = d453
				ps659.OverlayValues[454] = d454
				ps659.OverlayValues[455] = d455
				ps659.OverlayValues[456] = d456
				ps659.OverlayValues[457] = d457
				ps659.OverlayValues[458] = d458
				ps659.OverlayValues[459] = d459
				ps659.OverlayValues[460] = d460
				ps659.OverlayValues[461] = d461
				ps659.OverlayValues[462] = d462
				ps659.OverlayValues[463] = d463
				ps659.OverlayValues[464] = d464
				ps659.OverlayValues[465] = d465
				ps659.OverlayValues[466] = d466
				ps659.OverlayValues[467] = d467
				ps659.OverlayValues[468] = d468
				ps659.OverlayValues[469] = d469
				ps659.OverlayValues[470] = d470
				ps659.OverlayValues[471] = d471
				ps659.OverlayValues[472] = d472
				ps659.OverlayValues[473] = d473
				ps659.OverlayValues[474] = d474
				ps659.OverlayValues[475] = d475
				ps659.OverlayValues[648] = d648
				ps659.OverlayValues[649] = d649
				ps659.OverlayValues[650] = d650
				ps659.OverlayValues[652] = d652
				ps659.OverlayValues[653] = d653
				ps659.OverlayValues[654] = d654
				ps659.OverlayValues[655] = d655
				ps659.OverlayValues[656] = d656
				ps659.OverlayValues[657] = d657
				ps659.OverlayValues[658] = d658
				ps659.PhiValues = make([]scm.JITValueDesc, 1)
				d660 = d8
				ps659.PhiValues[0] = d660
				return bbs[2].RenderPS(ps659)
			}
			if ps.General {
			}
			ps661 := scm.PhiState{General: ps.General}
			ps661.OverlayValues = make([]scm.JITValueDesc, 661)
			ps661.OverlayValues[1] = d1
			ps661.OverlayValues[2] = d2
			ps661.OverlayValues[3] = d3
			ps661.OverlayValues[4] = d4
			ps661.OverlayValues[5] = d5
			ps661.OverlayValues[6] = d6
			ps661.OverlayValues[7] = d7
			ps661.OverlayValues[8] = d8
			ps661.OverlayValues[9] = d9
			ps661.OverlayValues[10] = d10
			ps661.OverlayValues[11] = d11
			ps661.OverlayValues[12] = d12
			ps661.OverlayValues[13] = d13
			ps661.OverlayValues[14] = d14
			ps661.OverlayValues[15] = d15
			ps661.OverlayValues[17] = d17
			ps661.OverlayValues[18] = d18
			ps661.OverlayValues[19] = d19
			ps661.OverlayValues[20] = d20
			ps661.OverlayValues[21] = d21
			ps661.OverlayValues[22] = d22
			ps661.OverlayValues[24] = d24
			ps661.OverlayValues[25] = d25
			ps661.OverlayValues[26] = d26
			ps661.OverlayValues[27] = d27
			ps661.OverlayValues[28] = d28
			ps661.OverlayValues[29] = d29
			ps661.OverlayValues[30] = d30
			ps661.OverlayValues[31] = d31
			ps661.OverlayValues[32] = d32
			ps661.OverlayValues[33] = d33
			ps661.OverlayValues[34] = d34
			ps661.OverlayValues[35] = d35
			ps661.OverlayValues[36] = d36
			ps661.OverlayValues[37] = d37
			ps661.OverlayValues[38] = d38
			ps661.OverlayValues[39] = d39
			ps661.OverlayValues[40] = d40
			ps661.OverlayValues[41] = d41
			ps661.OverlayValues[42] = d42
			ps661.OverlayValues[43] = d43
			ps661.OverlayValues[44] = d44
			ps661.OverlayValues[45] = d45
			ps661.OverlayValues[46] = d46
			ps661.OverlayValues[47] = d47
			ps661.OverlayValues[48] = d48
			ps661.OverlayValues[49] = d49
			ps661.OverlayValues[50] = d50
			ps661.OverlayValues[51] = d51
			ps661.OverlayValues[52] = d52
			ps661.OverlayValues[53] = d53
			ps661.OverlayValues[54] = d54
			ps661.OverlayValues[55] = d55
			ps661.OverlayValues[56] = d56
			ps661.OverlayValues[57] = d57
			ps661.OverlayValues[58] = d58
			ps661.OverlayValues[59] = d59
			ps661.OverlayValues[60] = d60
			ps661.OverlayValues[61] = d61
			ps661.OverlayValues[64] = d64
			ps661.OverlayValues[65] = d65
			ps661.OverlayValues[66] = d66
			ps661.OverlayValues[132] = d132
			ps661.OverlayValues[133] = d133
			ps661.OverlayValues[134] = d134
			ps661.OverlayValues[136] = d136
			ps661.OverlayValues[137] = d137
			ps661.OverlayValues[138] = d138
			ps661.OverlayValues[139] = d139
			ps661.OverlayValues[140] = d140
			ps661.OverlayValues[141] = d141
			ps661.OverlayValues[142] = d142
			ps661.OverlayValues[143] = d143
			ps661.OverlayValues[144] = d144
			ps661.OverlayValues[145] = d145
			ps661.OverlayValues[146] = d146
			ps661.OverlayValues[147] = d147
			ps661.OverlayValues[148] = d148
			ps661.OverlayValues[149] = d149
			ps661.OverlayValues[150] = d150
			ps661.OverlayValues[151] = d151
			ps661.OverlayValues[152] = d152
			ps661.OverlayValues[153] = d153
			ps661.OverlayValues[154] = d154
			ps661.OverlayValues[155] = d155
			ps661.OverlayValues[156] = d156
			ps661.OverlayValues[157] = d157
			ps661.OverlayValues[158] = d158
			ps661.OverlayValues[159] = d159
			ps661.OverlayValues[160] = d160
			ps661.OverlayValues[161] = d161
			ps661.OverlayValues[162] = d162
			ps661.OverlayValues[163] = d163
			ps661.OverlayValues[164] = d164
			ps661.OverlayValues[165] = d165
			ps661.OverlayValues[166] = d166
			ps661.OverlayValues[167] = d167
			ps661.OverlayValues[168] = d168
			ps661.OverlayValues[169] = d169
			ps661.OverlayValues[170] = d170
			ps661.OverlayValues[171] = d171
			ps661.OverlayValues[172] = d172
			ps661.OverlayValues[175] = d175
			ps661.OverlayValues[282] = d282
			ps661.OverlayValues[283] = d283
			ps661.OverlayValues[284] = d284
			ps661.OverlayValues[285] = d285
			ps661.OverlayValues[287] = d287
			ps661.OverlayValues[288] = d288
			ps661.OverlayValues[289] = d289
			ps661.OverlayValues[290] = d290
			ps661.OverlayValues[291] = d291
			ps661.OverlayValues[292] = d292
			ps661.OverlayValues[293] = d293
			ps661.OverlayValues[294] = d294
			ps661.OverlayValues[296] = d296
			ps661.OverlayValues[298] = d298
			ps661.OverlayValues[299] = d299
			ps661.OverlayValues[300] = d300
			ps661.OverlayValues[301] = d301
			ps661.OverlayValues[302] = d302
			ps661.OverlayValues[305] = d305
			ps661.OverlayValues[429] = d429
			ps661.OverlayValues[430] = d430
			ps661.OverlayValues[431] = d431
			ps661.OverlayValues[432] = d432
			ps661.OverlayValues[433] = d433
			ps661.OverlayValues[435] = d435
			ps661.OverlayValues[436] = d436
			ps661.OverlayValues[437] = d437
			ps661.OverlayValues[439] = d439
			ps661.OverlayValues[440] = d440
			ps661.OverlayValues[441] = d441
			ps661.OverlayValues[442] = d442
			ps661.OverlayValues[443] = d443
			ps661.OverlayValues[444] = d444
			ps661.OverlayValues[445] = d445
			ps661.OverlayValues[446] = d446
			ps661.OverlayValues[447] = d447
			ps661.OverlayValues[448] = d448
			ps661.OverlayValues[449] = d449
			ps661.OverlayValues[450] = d450
			ps661.OverlayValues[451] = d451
			ps661.OverlayValues[452] = d452
			ps661.OverlayValues[453] = d453
			ps661.OverlayValues[454] = d454
			ps661.OverlayValues[455] = d455
			ps661.OverlayValues[456] = d456
			ps661.OverlayValues[457] = d457
			ps661.OverlayValues[458] = d458
			ps661.OverlayValues[459] = d459
			ps661.OverlayValues[460] = d460
			ps661.OverlayValues[461] = d461
			ps661.OverlayValues[462] = d462
			ps661.OverlayValues[463] = d463
			ps661.OverlayValues[464] = d464
			ps661.OverlayValues[465] = d465
			ps661.OverlayValues[466] = d466
			ps661.OverlayValues[467] = d467
			ps661.OverlayValues[468] = d468
			ps661.OverlayValues[469] = d469
			ps661.OverlayValues[470] = d470
			ps661.OverlayValues[471] = d471
			ps661.OverlayValues[472] = d472
			ps661.OverlayValues[473] = d473
			ps661.OverlayValues[474] = d474
			ps661.OverlayValues[475] = d475
			ps661.OverlayValues[648] = d648
			ps661.OverlayValues[649] = d649
			ps661.OverlayValues[650] = d650
			ps661.OverlayValues[652] = d652
			ps661.OverlayValues[653] = d653
			ps661.OverlayValues[654] = d654
			ps661.OverlayValues[655] = d655
			ps661.OverlayValues[656] = d656
			ps661.OverlayValues[657] = d657
			ps661.OverlayValues[658] = d658
			ps661.OverlayValues[660] = d660
			return bbs[10].RenderPS(ps661)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d662 := ps.PhiValues[0]
				ctx.EnsureDesc(&d662)
				ctx.EmitStoreToStack(d662, int32(bbs[8].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d663 := ps.PhiValues[1]
				ctx.EnsureDesc(&d663)
				ctx.EmitStoreToStack(d663, int32(bbs[8].PhiBase)+int32(16))
			}
			ps.General = true
			return bbs[8].RenderPS(ps)
		}
		lbl41 := ctx.ReserveLabel()
		lbl42 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d656.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl41)
		ctx.EmitJmp(lbl42)
		ctx.MarkLabel(lbl41)
		ctx.SyncDesc(&d8)
		if d8.Loc == scm.LocReg {
			ctx.ProtectReg(d8.Reg)
		} else if d8.Loc == scm.LocRegPair {
			ctx.ProtectReg(d8.Reg)
			ctx.ProtectReg(d8.Reg2)
		}
		d664 = d8
		if d664.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d664)
		d665 = d664
		if d665.Loc == scm.LocImm {
			d665 = scm.JITValueDesc{Loc: scm.LocImm, Type: d665.Type, Imm: scm.NewInt(int64(uint64(d665.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d665.Reg, 32)
			ctx.EmitShrRegImm8(d665.Reg, 32)
		}
		ctx.EmitStoreToStack(d665, int32(bbs[2].PhiBase)+int32(0))
		if d8.Loc == scm.LocReg {
			ctx.UnprotectReg(d8.Reg)
		} else if d8.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d8.Reg)
			ctx.UnprotectReg(d8.Reg2)
		}
		ctx.EmitJmp(lbl3)
		ctx.MarkLabel(lbl42)
		ctx.EmitJmp(lbl11)
		ps666 := scm.PhiState{General: true}
		ps666.OverlayValues = make([]scm.JITValueDesc, 666)
		ps666.OverlayValues[1] = d1
		ps666.OverlayValues[2] = d2
		ps666.OverlayValues[3] = d3
		ps666.OverlayValues[4] = d4
		ps666.OverlayValues[5] = d5
		ps666.OverlayValues[6] = d6
		ps666.OverlayValues[7] = d7
		ps666.OverlayValues[8] = d8
		ps666.OverlayValues[9] = d9
		ps666.OverlayValues[10] = d10
		ps666.OverlayValues[11] = d11
		ps666.OverlayValues[12] = d12
		ps666.OverlayValues[13] = d13
		ps666.OverlayValues[14] = d14
		ps666.OverlayValues[15] = d15
		ps666.OverlayValues[17] = d17
		ps666.OverlayValues[18] = d18
		ps666.OverlayValues[19] = d19
		ps666.OverlayValues[20] = d20
		ps666.OverlayValues[21] = d21
		ps666.OverlayValues[22] = d22
		ps666.OverlayValues[24] = d24
		ps666.OverlayValues[25] = d25
		ps666.OverlayValues[26] = d26
		ps666.OverlayValues[27] = d27
		ps666.OverlayValues[28] = d28
		ps666.OverlayValues[29] = d29
		ps666.OverlayValues[30] = d30
		ps666.OverlayValues[31] = d31
		ps666.OverlayValues[32] = d32
		ps666.OverlayValues[33] = d33
		ps666.OverlayValues[34] = d34
		ps666.OverlayValues[35] = d35
		ps666.OverlayValues[36] = d36
		ps666.OverlayValues[37] = d37
		ps666.OverlayValues[38] = d38
		ps666.OverlayValues[39] = d39
		ps666.OverlayValues[40] = d40
		ps666.OverlayValues[41] = d41
		ps666.OverlayValues[42] = d42
		ps666.OverlayValues[43] = d43
		ps666.OverlayValues[44] = d44
		ps666.OverlayValues[45] = d45
		ps666.OverlayValues[46] = d46
		ps666.OverlayValues[47] = d47
		ps666.OverlayValues[48] = d48
		ps666.OverlayValues[49] = d49
		ps666.OverlayValues[50] = d50
		ps666.OverlayValues[51] = d51
		ps666.OverlayValues[52] = d52
		ps666.OverlayValues[53] = d53
		ps666.OverlayValues[54] = d54
		ps666.OverlayValues[55] = d55
		ps666.OverlayValues[56] = d56
		ps666.OverlayValues[57] = d57
		ps666.OverlayValues[58] = d58
		ps666.OverlayValues[59] = d59
		ps666.OverlayValues[60] = d60
		ps666.OverlayValues[61] = d61
		ps666.OverlayValues[64] = d64
		ps666.OverlayValues[65] = d65
		ps666.OverlayValues[66] = d66
		ps666.OverlayValues[132] = d132
		ps666.OverlayValues[133] = d133
		ps666.OverlayValues[134] = d134
		ps666.OverlayValues[136] = d136
		ps666.OverlayValues[137] = d137
		ps666.OverlayValues[138] = d138
		ps666.OverlayValues[139] = d139
		ps666.OverlayValues[140] = d140
		ps666.OverlayValues[141] = d141
		ps666.OverlayValues[142] = d142
		ps666.OverlayValues[143] = d143
		ps666.OverlayValues[144] = d144
		ps666.OverlayValues[145] = d145
		ps666.OverlayValues[146] = d146
		ps666.OverlayValues[147] = d147
		ps666.OverlayValues[148] = d148
		ps666.OverlayValues[149] = d149
		ps666.OverlayValues[150] = d150
		ps666.OverlayValues[151] = d151
		ps666.OverlayValues[152] = d152
		ps666.OverlayValues[153] = d153
		ps666.OverlayValues[154] = d154
		ps666.OverlayValues[155] = d155
		ps666.OverlayValues[156] = d156
		ps666.OverlayValues[157] = d157
		ps666.OverlayValues[158] = d158
		ps666.OverlayValues[159] = d159
		ps666.OverlayValues[160] = d160
		ps666.OverlayValues[161] = d161
		ps666.OverlayValues[162] = d162
		ps666.OverlayValues[163] = d163
		ps666.OverlayValues[164] = d164
		ps666.OverlayValues[165] = d165
		ps666.OverlayValues[166] = d166
		ps666.OverlayValues[167] = d167
		ps666.OverlayValues[168] = d168
		ps666.OverlayValues[169] = d169
		ps666.OverlayValues[170] = d170
		ps666.OverlayValues[171] = d171
		ps666.OverlayValues[172] = d172
		ps666.OverlayValues[175] = d175
		ps666.OverlayValues[282] = d282
		ps666.OverlayValues[283] = d283
		ps666.OverlayValues[284] = d284
		ps666.OverlayValues[285] = d285
		ps666.OverlayValues[287] = d287
		ps666.OverlayValues[288] = d288
		ps666.OverlayValues[289] = d289
		ps666.OverlayValues[290] = d290
		ps666.OverlayValues[291] = d291
		ps666.OverlayValues[292] = d292
		ps666.OverlayValues[293] = d293
		ps666.OverlayValues[294] = d294
		ps666.OverlayValues[296] = d296
		ps666.OverlayValues[298] = d298
		ps666.OverlayValues[299] = d299
		ps666.OverlayValues[300] = d300
		ps666.OverlayValues[301] = d301
		ps666.OverlayValues[302] = d302
		ps666.OverlayValues[305] = d305
		ps666.OverlayValues[429] = d429
		ps666.OverlayValues[430] = d430
		ps666.OverlayValues[431] = d431
		ps666.OverlayValues[432] = d432
		ps666.OverlayValues[433] = d433
		ps666.OverlayValues[435] = d435
		ps666.OverlayValues[436] = d436
		ps666.OverlayValues[437] = d437
		ps666.OverlayValues[439] = d439
		ps666.OverlayValues[440] = d440
		ps666.OverlayValues[441] = d441
		ps666.OverlayValues[442] = d442
		ps666.OverlayValues[443] = d443
		ps666.OverlayValues[444] = d444
		ps666.OverlayValues[445] = d445
		ps666.OverlayValues[446] = d446
		ps666.OverlayValues[447] = d447
		ps666.OverlayValues[448] = d448
		ps666.OverlayValues[449] = d449
		ps666.OverlayValues[450] = d450
		ps666.OverlayValues[451] = d451
		ps666.OverlayValues[452] = d452
		ps666.OverlayValues[453] = d453
		ps666.OverlayValues[454] = d454
		ps666.OverlayValues[455] = d455
		ps666.OverlayValues[456] = d456
		ps666.OverlayValues[457] = d457
		ps666.OverlayValues[458] = d458
		ps666.OverlayValues[459] = d459
		ps666.OverlayValues[460] = d460
		ps666.OverlayValues[461] = d461
		ps666.OverlayValues[462] = d462
		ps666.OverlayValues[463] = d463
		ps666.OverlayValues[464] = d464
		ps666.OverlayValues[465] = d465
		ps666.OverlayValues[466] = d466
		ps666.OverlayValues[467] = d467
		ps666.OverlayValues[468] = d468
		ps666.OverlayValues[469] = d469
		ps666.OverlayValues[470] = d470
		ps666.OverlayValues[471] = d471
		ps666.OverlayValues[472] = d472
		ps666.OverlayValues[473] = d473
		ps666.OverlayValues[474] = d474
		ps666.OverlayValues[475] = d475
		ps666.OverlayValues[648] = d648
		ps666.OverlayValues[649] = d649
		ps666.OverlayValues[650] = d650
		ps666.OverlayValues[652] = d652
		ps666.OverlayValues[653] = d653
		ps666.OverlayValues[654] = d654
		ps666.OverlayValues[655] = d655
		ps666.OverlayValues[656] = d656
		ps666.OverlayValues[657] = d657
		ps666.OverlayValues[658] = d658
		ps666.OverlayValues[660] = d660
		ps666.OverlayValues[662] = d662
		ps666.OverlayValues[663] = d663
		ps666.OverlayValues[664] = d664
		ps666.OverlayValues[665] = d665
		ps666.PhiValues = make([]scm.JITValueDesc, 1)
		d668 = d8
		ps666.PhiValues[0] = d668
		ps667 := scm.PhiState{General: true}
		ps667.OverlayValues = make([]scm.JITValueDesc, 669)
		ps667.OverlayValues[1] = d1
		ps667.OverlayValues[2] = d2
		ps667.OverlayValues[3] = d3
		ps667.OverlayValues[4] = d4
		ps667.OverlayValues[5] = d5
		ps667.OverlayValues[6] = d6
		ps667.OverlayValues[7] = d7
		ps667.OverlayValues[8] = d8
		ps667.OverlayValues[9] = d9
		ps667.OverlayValues[10] = d10
		ps667.OverlayValues[11] = d11
		ps667.OverlayValues[12] = d12
		ps667.OverlayValues[13] = d13
		ps667.OverlayValues[14] = d14
		ps667.OverlayValues[15] = d15
		ps667.OverlayValues[17] = d17
		ps667.OverlayValues[18] = d18
		ps667.OverlayValues[19] = d19
		ps667.OverlayValues[20] = d20
		ps667.OverlayValues[21] = d21
		ps667.OverlayValues[22] = d22
		ps667.OverlayValues[24] = d24
		ps667.OverlayValues[25] = d25
		ps667.OverlayValues[26] = d26
		ps667.OverlayValues[27] = d27
		ps667.OverlayValues[28] = d28
		ps667.OverlayValues[29] = d29
		ps667.OverlayValues[30] = d30
		ps667.OverlayValues[31] = d31
		ps667.OverlayValues[32] = d32
		ps667.OverlayValues[33] = d33
		ps667.OverlayValues[34] = d34
		ps667.OverlayValues[35] = d35
		ps667.OverlayValues[36] = d36
		ps667.OverlayValues[37] = d37
		ps667.OverlayValues[38] = d38
		ps667.OverlayValues[39] = d39
		ps667.OverlayValues[40] = d40
		ps667.OverlayValues[41] = d41
		ps667.OverlayValues[42] = d42
		ps667.OverlayValues[43] = d43
		ps667.OverlayValues[44] = d44
		ps667.OverlayValues[45] = d45
		ps667.OverlayValues[46] = d46
		ps667.OverlayValues[47] = d47
		ps667.OverlayValues[48] = d48
		ps667.OverlayValues[49] = d49
		ps667.OverlayValues[50] = d50
		ps667.OverlayValues[51] = d51
		ps667.OverlayValues[52] = d52
		ps667.OverlayValues[53] = d53
		ps667.OverlayValues[54] = d54
		ps667.OverlayValues[55] = d55
		ps667.OverlayValues[56] = d56
		ps667.OverlayValues[57] = d57
		ps667.OverlayValues[58] = d58
		ps667.OverlayValues[59] = d59
		ps667.OverlayValues[60] = d60
		ps667.OverlayValues[61] = d61
		ps667.OverlayValues[64] = d64
		ps667.OverlayValues[65] = d65
		ps667.OverlayValues[66] = d66
		ps667.OverlayValues[132] = d132
		ps667.OverlayValues[133] = d133
		ps667.OverlayValues[134] = d134
		ps667.OverlayValues[136] = d136
		ps667.OverlayValues[137] = d137
		ps667.OverlayValues[138] = d138
		ps667.OverlayValues[139] = d139
		ps667.OverlayValues[140] = d140
		ps667.OverlayValues[141] = d141
		ps667.OverlayValues[142] = d142
		ps667.OverlayValues[143] = d143
		ps667.OverlayValues[144] = d144
		ps667.OverlayValues[145] = d145
		ps667.OverlayValues[146] = d146
		ps667.OverlayValues[147] = d147
		ps667.OverlayValues[148] = d148
		ps667.OverlayValues[149] = d149
		ps667.OverlayValues[150] = d150
		ps667.OverlayValues[151] = d151
		ps667.OverlayValues[152] = d152
		ps667.OverlayValues[153] = d153
		ps667.OverlayValues[154] = d154
		ps667.OverlayValues[155] = d155
		ps667.OverlayValues[156] = d156
		ps667.OverlayValues[157] = d157
		ps667.OverlayValues[158] = d158
		ps667.OverlayValues[159] = d159
		ps667.OverlayValues[160] = d160
		ps667.OverlayValues[161] = d161
		ps667.OverlayValues[162] = d162
		ps667.OverlayValues[163] = d163
		ps667.OverlayValues[164] = d164
		ps667.OverlayValues[165] = d165
		ps667.OverlayValues[166] = d166
		ps667.OverlayValues[167] = d167
		ps667.OverlayValues[168] = d168
		ps667.OverlayValues[169] = d169
		ps667.OverlayValues[170] = d170
		ps667.OverlayValues[171] = d171
		ps667.OverlayValues[172] = d172
		ps667.OverlayValues[175] = d175
		ps667.OverlayValues[282] = d282
		ps667.OverlayValues[283] = d283
		ps667.OverlayValues[284] = d284
		ps667.OverlayValues[285] = d285
		ps667.OverlayValues[287] = d287
		ps667.OverlayValues[288] = d288
		ps667.OverlayValues[289] = d289
		ps667.OverlayValues[290] = d290
		ps667.OverlayValues[291] = d291
		ps667.OverlayValues[292] = d292
		ps667.OverlayValues[293] = d293
		ps667.OverlayValues[294] = d294
		ps667.OverlayValues[296] = d296
		ps667.OverlayValues[298] = d298
		ps667.OverlayValues[299] = d299
		ps667.OverlayValues[300] = d300
		ps667.OverlayValues[301] = d301
		ps667.OverlayValues[302] = d302
		ps667.OverlayValues[305] = d305
		ps667.OverlayValues[429] = d429
		ps667.OverlayValues[430] = d430
		ps667.OverlayValues[431] = d431
		ps667.OverlayValues[432] = d432
		ps667.OverlayValues[433] = d433
		ps667.OverlayValues[435] = d435
		ps667.OverlayValues[436] = d436
		ps667.OverlayValues[437] = d437
		ps667.OverlayValues[439] = d439
		ps667.OverlayValues[440] = d440
		ps667.OverlayValues[441] = d441
		ps667.OverlayValues[442] = d442
		ps667.OverlayValues[443] = d443
		ps667.OverlayValues[444] = d444
		ps667.OverlayValues[445] = d445
		ps667.OverlayValues[446] = d446
		ps667.OverlayValues[447] = d447
		ps667.OverlayValues[448] = d448
		ps667.OverlayValues[449] = d449
		ps667.OverlayValues[450] = d450
		ps667.OverlayValues[451] = d451
		ps667.OverlayValues[452] = d452
		ps667.OverlayValues[453] = d453
		ps667.OverlayValues[454] = d454
		ps667.OverlayValues[455] = d455
		ps667.OverlayValues[456] = d456
		ps667.OverlayValues[457] = d457
		ps667.OverlayValues[458] = d458
		ps667.OverlayValues[459] = d459
		ps667.OverlayValues[460] = d460
		ps667.OverlayValues[461] = d461
		ps667.OverlayValues[462] = d462
		ps667.OverlayValues[463] = d463
		ps667.OverlayValues[464] = d464
		ps667.OverlayValues[465] = d465
		ps667.OverlayValues[466] = d466
		ps667.OverlayValues[467] = d467
		ps667.OverlayValues[468] = d468
		ps667.OverlayValues[469] = d469
		ps667.OverlayValues[470] = d470
		ps667.OverlayValues[471] = d471
		ps667.OverlayValues[472] = d472
		ps667.OverlayValues[473] = d473
		ps667.OverlayValues[474] = d474
		ps667.OverlayValues[475] = d475
		ps667.OverlayValues[648] = d648
		ps667.OverlayValues[649] = d649
		ps667.OverlayValues[650] = d650
		ps667.OverlayValues[652] = d652
		ps667.OverlayValues[653] = d653
		ps667.OverlayValues[654] = d654
		ps667.OverlayValues[655] = d655
		ps667.OverlayValues[656] = d656
		ps667.OverlayValues[657] = d657
		ps667.OverlayValues[658] = d658
		ps667.OverlayValues[660] = d660
		ps667.OverlayValues[662] = d662
		ps667.OverlayValues[663] = d663
		ps667.OverlayValues[664] = d664
		ps667.OverlayValues[665] = d665
		ps667.OverlayValues[668] = d668
		snap669 := d1
		snap670 := d2
		snap671 := d3
		snap672 := d4
		snap673 := d5
		snap674 := d6
		snap675 := d7
		snap676 := d8
		snap677 := d9
		snap678 := d10
		snap679 := d11
		snap680 := d12
		snap681 := d13
		snap682 := d14
		snap683 := d15
		snap684 := d17
		snap685 := d18
		snap686 := d19
		snap687 := d20
		snap688 := d21
		snap689 := d22
		snap690 := d24
		snap691 := d25
		snap692 := d26
		snap693 := d27
		snap694 := d28
		snap695 := d29
		snap696 := d30
		snap697 := d31
		snap698 := d32
		snap699 := d33
		snap700 := d34
		snap701 := d35
		snap702 := d36
		snap703 := d37
		snap704 := d38
		snap705 := d39
		snap706 := d40
		snap707 := d41
		snap708 := d42
		snap709 := d43
		snap710 := d44
		snap711 := d45
		snap712 := d46
		snap713 := d47
		snap714 := d48
		snap715 := d49
		snap716 := d50
		snap717 := d51
		snap718 := d52
		snap719 := d53
		snap720 := d54
		snap721 := d55
		snap722 := d56
		snap723 := d57
		snap724 := d58
		snap725 := d59
		snap726 := d60
		snap727 := d61
		snap728 := d64
		snap729 := d65
		snap730 := d66
		snap731 := d132
		snap732 := d133
		snap733 := d134
		snap734 := d136
		snap735 := d137
		snap736 := d138
		snap737 := d139
		snap738 := d140
		snap739 := d141
		snap740 := d142
		snap741 := d143
		snap742 := d144
		snap743 := d145
		snap744 := d146
		snap745 := d147
		snap746 := d148
		snap747 := d149
		snap748 := d150
		snap749 := d151
		snap750 := d152
		snap751 := d153
		snap752 := d154
		snap753 := d155
		snap754 := d156
		snap755 := d157
		snap756 := d158
		snap757 := d159
		snap758 := d160
		snap759 := d161
		snap760 := d162
		snap761 := d163
		snap762 := d164
		snap763 := d165
		snap764 := d166
		snap765 := d167
		snap766 := d168
		snap767 := d169
		snap768 := d170
		snap769 := d171
		snap770 := d172
		snap771 := d175
		snap772 := d282
		snap773 := d283
		snap774 := d284
		snap775 := d285
		snap776 := d287
		snap777 := d288
		snap778 := d289
		snap779 := d290
		snap780 := d291
		snap781 := d292
		snap782 := d293
		snap783 := d294
		snap784 := d296
		snap785 := d298
		snap786 := d299
		snap787 := d300
		snap788 := d301
		snap789 := d302
		snap790 := d305
		snap791 := d429
		snap792 := d430
		snap793 := d431
		snap794 := d432
		snap795 := d433
		snap796 := d435
		snap797 := d436
		snap798 := d437
		snap799 := d439
		snap800 := d440
		snap801 := d441
		snap802 := d442
		snap803 := d443
		snap804 := d444
		snap805 := d445
		snap806 := d446
		snap807 := d447
		snap808 := d448
		snap809 := d449
		snap810 := d450
		snap811 := d451
		snap812 := d452
		snap813 := d453
		snap814 := d454
		snap815 := d455
		snap816 := d456
		snap817 := d457
		snap818 := d458
		snap819 := d459
		snap820 := d460
		snap821 := d461
		snap822 := d462
		snap823 := d463
		snap824 := d464
		snap825 := d465
		snap826 := d466
		snap827 := d467
		snap828 := d468
		snap829 := d469
		snap830 := d470
		snap831 := d471
		snap832 := d472
		snap833 := d473
		snap834 := d474
		snap835 := d475
		snap836 := d648
		snap837 := d649
		snap838 := d650
		snap839 := d652
		snap840 := d653
		snap841 := d654
		snap842 := d655
		snap843 := d656
		snap844 := d657
		snap845 := d658
		snap846 := d660
		snap847 := d662
		snap848 := d663
		snap849 := d664
		snap850 := d665
		snap851 := d668
		alloc852 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps666)
		}
		ctx.RestoreAllocState(alloc852)
		d1 = snap669
		d2 = snap670
		d3 = snap671
		d4 = snap672
		d5 = snap673
		d6 = snap674
		d7 = snap675
		d8 = snap676
		d9 = snap677
		d10 = snap678
		d11 = snap679
		d12 = snap680
		d13 = snap681
		d14 = snap682
		d15 = snap683
		d17 = snap684
		d18 = snap685
		d19 = snap686
		d20 = snap687
		d21 = snap688
		d22 = snap689
		d24 = snap690
		d25 = snap691
		d26 = snap692
		d27 = snap693
		d28 = snap694
		d29 = snap695
		d30 = snap696
		d31 = snap697
		d32 = snap698
		d33 = snap699
		d34 = snap700
		d35 = snap701
		d36 = snap702
		d37 = snap703
		d38 = snap704
		d39 = snap705
		d40 = snap706
		d41 = snap707
		d42 = snap708
		d43 = snap709
		d44 = snap710
		d45 = snap711
		d46 = snap712
		d47 = snap713
		d48 = snap714
		d49 = snap715
		d50 = snap716
		d51 = snap717
		d52 = snap718
		d53 = snap719
		d54 = snap720
		d55 = snap721
		d56 = snap722
		d57 = snap723
		d58 = snap724
		d59 = snap725
		d60 = snap726
		d61 = snap727
		d64 = snap728
		d65 = snap729
		d66 = snap730
		d132 = snap731
		d133 = snap732
		d134 = snap733
		d136 = snap734
		d137 = snap735
		d138 = snap736
		d139 = snap737
		d140 = snap738
		d141 = snap739
		d142 = snap740
		d143 = snap741
		d144 = snap742
		d145 = snap743
		d146 = snap744
		d147 = snap745
		d148 = snap746
		d149 = snap747
		d150 = snap748
		d151 = snap749
		d152 = snap750
		d153 = snap751
		d154 = snap752
		d155 = snap753
		d156 = snap754
		d157 = snap755
		d158 = snap756
		d159 = snap757
		d160 = snap758
		d161 = snap759
		d162 = snap760
		d163 = snap761
		d164 = snap762
		d165 = snap763
		d166 = snap764
		d167 = snap765
		d168 = snap766
		d169 = snap767
		d170 = snap768
		d171 = snap769
		d172 = snap770
		d175 = snap771
		d282 = snap772
		d283 = snap773
		d284 = snap774
		d285 = snap775
		d287 = snap776
		d288 = snap777
		d289 = snap778
		d290 = snap779
		d291 = snap780
		d292 = snap781
		d293 = snap782
		d294 = snap783
		d296 = snap784
		d298 = snap785
		d299 = snap786
		d300 = snap787
		d301 = snap788
		d302 = snap789
		d305 = snap790
		d429 = snap791
		d430 = snap792
		d431 = snap793
		d432 = snap794
		d433 = snap795
		d435 = snap796
		d436 = snap797
		d437 = snap798
		d439 = snap799
		d440 = snap800
		d441 = snap801
		d442 = snap802
		d443 = snap803
		d444 = snap804
		d445 = snap805
		d446 = snap806
		d447 = snap807
		d448 = snap808
		d449 = snap809
		d450 = snap810
		d451 = snap811
		d452 = snap812
		d453 = snap813
		d454 = snap814
		d455 = snap815
		d456 = snap816
		d457 = snap817
		d458 = snap818
		d459 = snap819
		d460 = snap820
		d461 = snap821
		d462 = snap822
		d463 = snap823
		d464 = snap824
		d465 = snap825
		d466 = snap826
		d467 = snap827
		d468 = snap828
		d469 = snap829
		d470 = snap830
		d471 = snap831
		d472 = snap832
		d473 = snap833
		d474 = snap834
		d475 = snap835
		d648 = snap836
		d649 = snap837
		d650 = snap838
		d652 = snap839
		d653 = snap840
		d654 = snap841
		d655 = snap842
		d656 = snap843
		d657 = snap844
		d658 = snap845
		d660 = snap846
		d662 = snap847
		d663 = snap848
		d664 = snap849
		d665 = snap850
		d668 = snap851
		if !bbs[10].Rendered {
			return bbs[10].RenderPS(ps667)
		}
		return result
		ctx.FreeDesc(&d655)
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
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
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
		if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != scm.LocNone {
			d282 = ps.OverlayValues[282]
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
		if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
			d294 = ps.OverlayValues[294]
		}
		if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != scm.LocNone {
			d296 = ps.OverlayValues[296]
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
		if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != scm.LocNone {
			d305 = ps.OverlayValues[305]
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
		if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != scm.LocNone {
			d435 = ps.OverlayValues[435]
		}
		if len(ps.OverlayValues) > 436 && ps.OverlayValues[436].Loc != scm.LocNone {
			d436 = ps.OverlayValues[436]
		}
		if len(ps.OverlayValues) > 437 && ps.OverlayValues[437].Loc != scm.LocNone {
			d437 = ps.OverlayValues[437]
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
		if len(ps.OverlayValues) > 648 && ps.OverlayValues[648].Loc != scm.LocNone {
			d648 = ps.OverlayValues[648]
		}
		if len(ps.OverlayValues) > 649 && ps.OverlayValues[649].Loc != scm.LocNone {
			d649 = ps.OverlayValues[649]
		}
		if len(ps.OverlayValues) > 650 && ps.OverlayValues[650].Loc != scm.LocNone {
			d650 = ps.OverlayValues[650]
		}
		if len(ps.OverlayValues) > 652 && ps.OverlayValues[652].Loc != scm.LocNone {
			d652 = ps.OverlayValues[652]
		}
		if len(ps.OverlayValues) > 653 && ps.OverlayValues[653].Loc != scm.LocNone {
			d653 = ps.OverlayValues[653]
		}
		if len(ps.OverlayValues) > 654 && ps.OverlayValues[654].Loc != scm.LocNone {
			d654 = ps.OverlayValues[654]
		}
		if len(ps.OverlayValues) > 655 && ps.OverlayValues[655].Loc != scm.LocNone {
			d655 = ps.OverlayValues[655]
		}
		if len(ps.OverlayValues) > 656 && ps.OverlayValues[656].Loc != scm.LocNone {
			d656 = ps.OverlayValues[656]
		}
		if len(ps.OverlayValues) > 657 && ps.OverlayValues[657].Loc != scm.LocNone {
			d657 = ps.OverlayValues[657]
		}
		if len(ps.OverlayValues) > 658 && ps.OverlayValues[658].Loc != scm.LocNone {
			d658 = ps.OverlayValues[658]
		}
		if len(ps.OverlayValues) > 660 && ps.OverlayValues[660].Loc != scm.LocNone {
			d660 = ps.OverlayValues[660]
		}
		if len(ps.OverlayValues) > 662 && ps.OverlayValues[662].Loc != scm.LocNone {
			d662 = ps.OverlayValues[662]
		}
		if len(ps.OverlayValues) > 663 && ps.OverlayValues[663].Loc != scm.LocNone {
			d663 = ps.OverlayValues[663]
		}
		if len(ps.OverlayValues) > 664 && ps.OverlayValues[664].Loc != scm.LocNone {
			d664 = ps.OverlayValues[664]
		}
		if len(ps.OverlayValues) > 665 && ps.OverlayValues[665].Loc != scm.LocNone {
			d665 = ps.OverlayValues[665]
		}
		if len(ps.OverlayValues) > 668 && ps.OverlayValues[668].Loc != scm.LocNone {
			d668 = ps.OverlayValues[668]
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
			d853 = d5
			if d853.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d853)
			d854 = d853
			if d854.Loc == scm.LocImm {
				d854 = scm.JITValueDesc{Loc: scm.LocImm, Type: d854.Type, Imm: scm.NewInt(int64(uint64(d854.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d854.Reg, 32)
				ctx.EmitShrRegImm8(d854.Reg, 32)
			}
			ctx.EmitStoreToStack(d854, int32(bbs[8].PhiBase)+int32(0))
			d855 = d7
			if d855.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d855)
			d856 = d855
			if d856.Loc == scm.LocImm {
				d856 = scm.JITValueDesc{Loc: scm.LocImm, Type: d856.Type, Imm: scm.NewInt(int64(uint64(d856.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d856.Reg, 32)
				ctx.EmitShrRegImm8(d856.Reg, 32)
			}
			ctx.EmitStoreToStack(d856, int32(bbs[8].PhiBase)+int32(16))
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
		ps857 := scm.PhiState{General: ps.General}
		ps857.OverlayValues = make([]scm.JITValueDesc, 857)
		ps857.OverlayValues[1] = d1
		ps857.OverlayValues[2] = d2
		ps857.OverlayValues[3] = d3
		ps857.OverlayValues[4] = d4
		ps857.OverlayValues[5] = d5
		ps857.OverlayValues[6] = d6
		ps857.OverlayValues[7] = d7
		ps857.OverlayValues[8] = d8
		ps857.OverlayValues[9] = d9
		ps857.OverlayValues[10] = d10
		ps857.OverlayValues[11] = d11
		ps857.OverlayValues[12] = d12
		ps857.OverlayValues[13] = d13
		ps857.OverlayValues[14] = d14
		ps857.OverlayValues[15] = d15
		ps857.OverlayValues[17] = d17
		ps857.OverlayValues[18] = d18
		ps857.OverlayValues[19] = d19
		ps857.OverlayValues[20] = d20
		ps857.OverlayValues[21] = d21
		ps857.OverlayValues[22] = d22
		ps857.OverlayValues[24] = d24
		ps857.OverlayValues[25] = d25
		ps857.OverlayValues[26] = d26
		ps857.OverlayValues[27] = d27
		ps857.OverlayValues[28] = d28
		ps857.OverlayValues[29] = d29
		ps857.OverlayValues[30] = d30
		ps857.OverlayValues[31] = d31
		ps857.OverlayValues[32] = d32
		ps857.OverlayValues[33] = d33
		ps857.OverlayValues[34] = d34
		ps857.OverlayValues[35] = d35
		ps857.OverlayValues[36] = d36
		ps857.OverlayValues[37] = d37
		ps857.OverlayValues[38] = d38
		ps857.OverlayValues[39] = d39
		ps857.OverlayValues[40] = d40
		ps857.OverlayValues[41] = d41
		ps857.OverlayValues[42] = d42
		ps857.OverlayValues[43] = d43
		ps857.OverlayValues[44] = d44
		ps857.OverlayValues[45] = d45
		ps857.OverlayValues[46] = d46
		ps857.OverlayValues[47] = d47
		ps857.OverlayValues[48] = d48
		ps857.OverlayValues[49] = d49
		ps857.OverlayValues[50] = d50
		ps857.OverlayValues[51] = d51
		ps857.OverlayValues[52] = d52
		ps857.OverlayValues[53] = d53
		ps857.OverlayValues[54] = d54
		ps857.OverlayValues[55] = d55
		ps857.OverlayValues[56] = d56
		ps857.OverlayValues[57] = d57
		ps857.OverlayValues[58] = d58
		ps857.OverlayValues[59] = d59
		ps857.OverlayValues[60] = d60
		ps857.OverlayValues[61] = d61
		ps857.OverlayValues[64] = d64
		ps857.OverlayValues[65] = d65
		ps857.OverlayValues[66] = d66
		ps857.OverlayValues[132] = d132
		ps857.OverlayValues[133] = d133
		ps857.OverlayValues[134] = d134
		ps857.OverlayValues[136] = d136
		ps857.OverlayValues[137] = d137
		ps857.OverlayValues[138] = d138
		ps857.OverlayValues[139] = d139
		ps857.OverlayValues[140] = d140
		ps857.OverlayValues[141] = d141
		ps857.OverlayValues[142] = d142
		ps857.OverlayValues[143] = d143
		ps857.OverlayValues[144] = d144
		ps857.OverlayValues[145] = d145
		ps857.OverlayValues[146] = d146
		ps857.OverlayValues[147] = d147
		ps857.OverlayValues[148] = d148
		ps857.OverlayValues[149] = d149
		ps857.OverlayValues[150] = d150
		ps857.OverlayValues[151] = d151
		ps857.OverlayValues[152] = d152
		ps857.OverlayValues[153] = d153
		ps857.OverlayValues[154] = d154
		ps857.OverlayValues[155] = d155
		ps857.OverlayValues[156] = d156
		ps857.OverlayValues[157] = d157
		ps857.OverlayValues[158] = d158
		ps857.OverlayValues[159] = d159
		ps857.OverlayValues[160] = d160
		ps857.OverlayValues[161] = d161
		ps857.OverlayValues[162] = d162
		ps857.OverlayValues[163] = d163
		ps857.OverlayValues[164] = d164
		ps857.OverlayValues[165] = d165
		ps857.OverlayValues[166] = d166
		ps857.OverlayValues[167] = d167
		ps857.OverlayValues[168] = d168
		ps857.OverlayValues[169] = d169
		ps857.OverlayValues[170] = d170
		ps857.OverlayValues[171] = d171
		ps857.OverlayValues[172] = d172
		ps857.OverlayValues[175] = d175
		ps857.OverlayValues[282] = d282
		ps857.OverlayValues[283] = d283
		ps857.OverlayValues[284] = d284
		ps857.OverlayValues[285] = d285
		ps857.OverlayValues[287] = d287
		ps857.OverlayValues[288] = d288
		ps857.OverlayValues[289] = d289
		ps857.OverlayValues[290] = d290
		ps857.OverlayValues[291] = d291
		ps857.OverlayValues[292] = d292
		ps857.OverlayValues[293] = d293
		ps857.OverlayValues[294] = d294
		ps857.OverlayValues[296] = d296
		ps857.OverlayValues[298] = d298
		ps857.OverlayValues[299] = d299
		ps857.OverlayValues[300] = d300
		ps857.OverlayValues[301] = d301
		ps857.OverlayValues[302] = d302
		ps857.OverlayValues[305] = d305
		ps857.OverlayValues[429] = d429
		ps857.OverlayValues[430] = d430
		ps857.OverlayValues[431] = d431
		ps857.OverlayValues[432] = d432
		ps857.OverlayValues[433] = d433
		ps857.OverlayValues[435] = d435
		ps857.OverlayValues[436] = d436
		ps857.OverlayValues[437] = d437
		ps857.OverlayValues[439] = d439
		ps857.OverlayValues[440] = d440
		ps857.OverlayValues[441] = d441
		ps857.OverlayValues[442] = d442
		ps857.OverlayValues[443] = d443
		ps857.OverlayValues[444] = d444
		ps857.OverlayValues[445] = d445
		ps857.OverlayValues[446] = d446
		ps857.OverlayValues[447] = d447
		ps857.OverlayValues[448] = d448
		ps857.OverlayValues[449] = d449
		ps857.OverlayValues[450] = d450
		ps857.OverlayValues[451] = d451
		ps857.OverlayValues[452] = d452
		ps857.OverlayValues[453] = d453
		ps857.OverlayValues[454] = d454
		ps857.OverlayValues[455] = d455
		ps857.OverlayValues[456] = d456
		ps857.OverlayValues[457] = d457
		ps857.OverlayValues[458] = d458
		ps857.OverlayValues[459] = d459
		ps857.OverlayValues[460] = d460
		ps857.OverlayValues[461] = d461
		ps857.OverlayValues[462] = d462
		ps857.OverlayValues[463] = d463
		ps857.OverlayValues[464] = d464
		ps857.OverlayValues[465] = d465
		ps857.OverlayValues[466] = d466
		ps857.OverlayValues[467] = d467
		ps857.OverlayValues[468] = d468
		ps857.OverlayValues[469] = d469
		ps857.OverlayValues[470] = d470
		ps857.OverlayValues[471] = d471
		ps857.OverlayValues[472] = d472
		ps857.OverlayValues[473] = d473
		ps857.OverlayValues[474] = d474
		ps857.OverlayValues[475] = d475
		ps857.OverlayValues[648] = d648
		ps857.OverlayValues[649] = d649
		ps857.OverlayValues[650] = d650
		ps857.OverlayValues[652] = d652
		ps857.OverlayValues[653] = d653
		ps857.OverlayValues[654] = d654
		ps857.OverlayValues[655] = d655
		ps857.OverlayValues[656] = d656
		ps857.OverlayValues[657] = d657
		ps857.OverlayValues[658] = d658
		ps857.OverlayValues[660] = d660
		ps857.OverlayValues[662] = d662
		ps857.OverlayValues[663] = d663
		ps857.OverlayValues[664] = d664
		ps857.OverlayValues[665] = d665
		ps857.OverlayValues[668] = d668
		ps857.OverlayValues[853] = d853
		ps857.OverlayValues[854] = d854
		ps857.OverlayValues[855] = d855
		ps857.OverlayValues[856] = d856
		ps857.PhiValues = make([]scm.JITValueDesc, 2)
		d858 = d5
		ps857.PhiValues[0] = d858
		d859 = d7
		ps857.PhiValues[1] = d859
		if ps857.General && bbs[8].Rendered {
			ctx.EmitJmp(lbl9)
			return result
		}
		return bbs[8].RenderPS(ps857)
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
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
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
		if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != scm.LocNone {
			d282 = ps.OverlayValues[282]
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
		if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
			d294 = ps.OverlayValues[294]
		}
		if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != scm.LocNone {
			d296 = ps.OverlayValues[296]
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
		if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != scm.LocNone {
			d305 = ps.OverlayValues[305]
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
		if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != scm.LocNone {
			d435 = ps.OverlayValues[435]
		}
		if len(ps.OverlayValues) > 436 && ps.OverlayValues[436].Loc != scm.LocNone {
			d436 = ps.OverlayValues[436]
		}
		if len(ps.OverlayValues) > 437 && ps.OverlayValues[437].Loc != scm.LocNone {
			d437 = ps.OverlayValues[437]
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
		if len(ps.OverlayValues) > 648 && ps.OverlayValues[648].Loc != scm.LocNone {
			d648 = ps.OverlayValues[648]
		}
		if len(ps.OverlayValues) > 649 && ps.OverlayValues[649].Loc != scm.LocNone {
			d649 = ps.OverlayValues[649]
		}
		if len(ps.OverlayValues) > 650 && ps.OverlayValues[650].Loc != scm.LocNone {
			d650 = ps.OverlayValues[650]
		}
		if len(ps.OverlayValues) > 652 && ps.OverlayValues[652].Loc != scm.LocNone {
			d652 = ps.OverlayValues[652]
		}
		if len(ps.OverlayValues) > 653 && ps.OverlayValues[653].Loc != scm.LocNone {
			d653 = ps.OverlayValues[653]
		}
		if len(ps.OverlayValues) > 654 && ps.OverlayValues[654].Loc != scm.LocNone {
			d654 = ps.OverlayValues[654]
		}
		if len(ps.OverlayValues) > 655 && ps.OverlayValues[655].Loc != scm.LocNone {
			d655 = ps.OverlayValues[655]
		}
		if len(ps.OverlayValues) > 656 && ps.OverlayValues[656].Loc != scm.LocNone {
			d656 = ps.OverlayValues[656]
		}
		if len(ps.OverlayValues) > 657 && ps.OverlayValues[657].Loc != scm.LocNone {
			d657 = ps.OverlayValues[657]
		}
		if len(ps.OverlayValues) > 658 && ps.OverlayValues[658].Loc != scm.LocNone {
			d658 = ps.OverlayValues[658]
		}
		if len(ps.OverlayValues) > 660 && ps.OverlayValues[660].Loc != scm.LocNone {
			d660 = ps.OverlayValues[660]
		}
		if len(ps.OverlayValues) > 662 && ps.OverlayValues[662].Loc != scm.LocNone {
			d662 = ps.OverlayValues[662]
		}
		if len(ps.OverlayValues) > 663 && ps.OverlayValues[663].Loc != scm.LocNone {
			d663 = ps.OverlayValues[663]
		}
		if len(ps.OverlayValues) > 664 && ps.OverlayValues[664].Loc != scm.LocNone {
			d664 = ps.OverlayValues[664]
		}
		if len(ps.OverlayValues) > 665 && ps.OverlayValues[665].Loc != scm.LocNone {
			d665 = ps.OverlayValues[665]
		}
		if len(ps.OverlayValues) > 668 && ps.OverlayValues[668].Loc != scm.LocNone {
			d668 = ps.OverlayValues[668]
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
		if len(ps.OverlayValues) > 858 && ps.OverlayValues[858].Loc != scm.LocNone {
			d858 = ps.OverlayValues[858]
		}
		if len(ps.OverlayValues) > 859 && ps.OverlayValues[859].Loc != scm.LocNone {
			d859 = ps.OverlayValues[859]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d8)
		ctx.EnsureDesc(&d9)
		ctx.EnsureDescsTogether(&d8, &d9)
		var d860 scm.JITValueDesc
		if d8.Loc == scm.LocImm && d9.Loc == scm.LocImm {
			d860 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() + d9.Imm.Int())}
		} else if d9.Loc == scm.LocImm && d9.Imm.Int() == 0 {
			r141 := ctx.AllocRegExcept(d8.Reg)
			ctx.EmitMovRegReg(r141, d8.Reg)
			d860 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
			ctx.BindReg(r141, &d860)
		} else if d8.Loc == scm.LocImm && d8.Imm.Int() == 0 {
			d860 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d9.Reg}
			ctx.BindReg(d9.Reg, &d860)
		} else if d8.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d9.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d8.Imm.Int()))
			ctx.EmitAddInt64(scratch, d9.Reg)
			d860 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d860)
		} else if d9.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d8.Reg)
			ctx.EmitMovRegReg(scratch, d8.Reg)
			if d9.Imm.Int() >= -2147483648 && d9.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d9.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d9.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d860 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d860)
		} else {
			r142 := ctx.AllocRegExcept(d8.Reg, d9.Reg)
			ctx.EmitMovRegReg(r142, d8.Reg)
			ctx.EmitAddInt64(r142, d9.Reg)
			d860 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r142}
			ctx.BindReg(r142, &d860)
		}
		if d860.Loc == scm.LocImm {
			d860 = scm.JITValueDesc{Loc: scm.LocImm, Type: d860.Type, Imm: scm.NewInt(int64(uint64(d860.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d860.Reg, 32)
			ctx.EmitShrRegImm8(d860.Reg, 32)
		}
		if d860.Loc == scm.LocReg && d8.Loc == scm.LocReg && d860.Reg == d8.Reg {
			ctx.TransferReg(d8.Reg)
			d8.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d860)
		var d861 scm.JITValueDesc
		if d860.Loc == scm.LocImm {
			d861 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d860.Imm.Int() / 2)}
		} else {
			r143 := ctx.AllocRegExcept(d860.Reg)
			ctx.EmitMovRegReg(r143, d860.Reg)
			ctx.EmitShrRegImm8(r143, 1)
			d861 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r143}
			ctx.BindReg(r143, &d861)
		}
		if d861.Loc == scm.LocImm {
			d861 = scm.JITValueDesc{Loc: scm.LocImm, Type: d861.Type, Imm: scm.NewInt(int64(uint64(d861.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d861.Reg, 32)
			ctx.EmitShrRegImm8(d861.Reg, 32)
		}
		if d861.Loc == scm.LocReg && d860.Loc == scm.LocReg && d861.Reg == d860.Reg {
			ctx.TransferReg(d860.Reg)
			d860.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d861)
		ctx.EmitStoreToStack(d861, int32(bbs[1].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d861)
		ctx.FreeDesc(&d860)
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
			d862 = d8
			if d862.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d862)
			d863 = d862
			if d863.Loc == scm.LocImm {
				d863 = scm.JITValueDesc{Loc: scm.LocImm, Type: d863.Type, Imm: scm.NewInt(int64(uint64(d863.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d863.Reg, 32)
				ctx.EmitShrRegImm8(d863.Reg, 32)
			}
			ctx.EmitStoreToStack(d863, int32(bbs[1].PhiBase)+int32(16))
			d864 = d9
			if d864.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d864)
			d865 = d864
			if d865.Loc == scm.LocImm {
				d865 = scm.JITValueDesc{Loc: scm.LocImm, Type: d865.Type, Imm: scm.NewInt(int64(uint64(d865.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d865.Reg, 32)
				ctx.EmitShrRegImm8(d865.Reg, 32)
			}
			ctx.EmitStoreToStack(d865, int32(bbs[1].PhiBase)+int32(32))
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
		ps866 := scm.PhiState{General: ps.General}
		ps866.OverlayValues = make([]scm.JITValueDesc, 866)
		ps866.OverlayValues[1] = d1
		ps866.OverlayValues[2] = d2
		ps866.OverlayValues[3] = d3
		ps866.OverlayValues[4] = d4
		ps866.OverlayValues[5] = d5
		ps866.OverlayValues[6] = d6
		ps866.OverlayValues[7] = d7
		ps866.OverlayValues[8] = d8
		ps866.OverlayValues[9] = d9
		ps866.OverlayValues[10] = d10
		ps866.OverlayValues[11] = d11
		ps866.OverlayValues[12] = d12
		ps866.OverlayValues[13] = d13
		ps866.OverlayValues[14] = d14
		ps866.OverlayValues[15] = d15
		ps866.OverlayValues[17] = d17
		ps866.OverlayValues[18] = d18
		ps866.OverlayValues[19] = d19
		ps866.OverlayValues[20] = d20
		ps866.OverlayValues[21] = d21
		ps866.OverlayValues[22] = d22
		ps866.OverlayValues[24] = d24
		ps866.OverlayValues[25] = d25
		ps866.OverlayValues[26] = d26
		ps866.OverlayValues[27] = d27
		ps866.OverlayValues[28] = d28
		ps866.OverlayValues[29] = d29
		ps866.OverlayValues[30] = d30
		ps866.OverlayValues[31] = d31
		ps866.OverlayValues[32] = d32
		ps866.OverlayValues[33] = d33
		ps866.OverlayValues[34] = d34
		ps866.OverlayValues[35] = d35
		ps866.OverlayValues[36] = d36
		ps866.OverlayValues[37] = d37
		ps866.OverlayValues[38] = d38
		ps866.OverlayValues[39] = d39
		ps866.OverlayValues[40] = d40
		ps866.OverlayValues[41] = d41
		ps866.OverlayValues[42] = d42
		ps866.OverlayValues[43] = d43
		ps866.OverlayValues[44] = d44
		ps866.OverlayValues[45] = d45
		ps866.OverlayValues[46] = d46
		ps866.OverlayValues[47] = d47
		ps866.OverlayValues[48] = d48
		ps866.OverlayValues[49] = d49
		ps866.OverlayValues[50] = d50
		ps866.OverlayValues[51] = d51
		ps866.OverlayValues[52] = d52
		ps866.OverlayValues[53] = d53
		ps866.OverlayValues[54] = d54
		ps866.OverlayValues[55] = d55
		ps866.OverlayValues[56] = d56
		ps866.OverlayValues[57] = d57
		ps866.OverlayValues[58] = d58
		ps866.OverlayValues[59] = d59
		ps866.OverlayValues[60] = d60
		ps866.OverlayValues[61] = d61
		ps866.OverlayValues[64] = d64
		ps866.OverlayValues[65] = d65
		ps866.OverlayValues[66] = d66
		ps866.OverlayValues[132] = d132
		ps866.OverlayValues[133] = d133
		ps866.OverlayValues[134] = d134
		ps866.OverlayValues[136] = d136
		ps866.OverlayValues[137] = d137
		ps866.OverlayValues[138] = d138
		ps866.OverlayValues[139] = d139
		ps866.OverlayValues[140] = d140
		ps866.OverlayValues[141] = d141
		ps866.OverlayValues[142] = d142
		ps866.OverlayValues[143] = d143
		ps866.OverlayValues[144] = d144
		ps866.OverlayValues[145] = d145
		ps866.OverlayValues[146] = d146
		ps866.OverlayValues[147] = d147
		ps866.OverlayValues[148] = d148
		ps866.OverlayValues[149] = d149
		ps866.OverlayValues[150] = d150
		ps866.OverlayValues[151] = d151
		ps866.OverlayValues[152] = d152
		ps866.OverlayValues[153] = d153
		ps866.OverlayValues[154] = d154
		ps866.OverlayValues[155] = d155
		ps866.OverlayValues[156] = d156
		ps866.OverlayValues[157] = d157
		ps866.OverlayValues[158] = d158
		ps866.OverlayValues[159] = d159
		ps866.OverlayValues[160] = d160
		ps866.OverlayValues[161] = d161
		ps866.OverlayValues[162] = d162
		ps866.OverlayValues[163] = d163
		ps866.OverlayValues[164] = d164
		ps866.OverlayValues[165] = d165
		ps866.OverlayValues[166] = d166
		ps866.OverlayValues[167] = d167
		ps866.OverlayValues[168] = d168
		ps866.OverlayValues[169] = d169
		ps866.OverlayValues[170] = d170
		ps866.OverlayValues[171] = d171
		ps866.OverlayValues[172] = d172
		ps866.OverlayValues[175] = d175
		ps866.OverlayValues[282] = d282
		ps866.OverlayValues[283] = d283
		ps866.OverlayValues[284] = d284
		ps866.OverlayValues[285] = d285
		ps866.OverlayValues[287] = d287
		ps866.OverlayValues[288] = d288
		ps866.OverlayValues[289] = d289
		ps866.OverlayValues[290] = d290
		ps866.OverlayValues[291] = d291
		ps866.OverlayValues[292] = d292
		ps866.OverlayValues[293] = d293
		ps866.OverlayValues[294] = d294
		ps866.OverlayValues[296] = d296
		ps866.OverlayValues[298] = d298
		ps866.OverlayValues[299] = d299
		ps866.OverlayValues[300] = d300
		ps866.OverlayValues[301] = d301
		ps866.OverlayValues[302] = d302
		ps866.OverlayValues[305] = d305
		ps866.OverlayValues[429] = d429
		ps866.OverlayValues[430] = d430
		ps866.OverlayValues[431] = d431
		ps866.OverlayValues[432] = d432
		ps866.OverlayValues[433] = d433
		ps866.OverlayValues[435] = d435
		ps866.OverlayValues[436] = d436
		ps866.OverlayValues[437] = d437
		ps866.OverlayValues[439] = d439
		ps866.OverlayValues[440] = d440
		ps866.OverlayValues[441] = d441
		ps866.OverlayValues[442] = d442
		ps866.OverlayValues[443] = d443
		ps866.OverlayValues[444] = d444
		ps866.OverlayValues[445] = d445
		ps866.OverlayValues[446] = d446
		ps866.OverlayValues[447] = d447
		ps866.OverlayValues[448] = d448
		ps866.OverlayValues[449] = d449
		ps866.OverlayValues[450] = d450
		ps866.OverlayValues[451] = d451
		ps866.OverlayValues[452] = d452
		ps866.OverlayValues[453] = d453
		ps866.OverlayValues[454] = d454
		ps866.OverlayValues[455] = d455
		ps866.OverlayValues[456] = d456
		ps866.OverlayValues[457] = d457
		ps866.OverlayValues[458] = d458
		ps866.OverlayValues[459] = d459
		ps866.OverlayValues[460] = d460
		ps866.OverlayValues[461] = d461
		ps866.OverlayValues[462] = d462
		ps866.OverlayValues[463] = d463
		ps866.OverlayValues[464] = d464
		ps866.OverlayValues[465] = d465
		ps866.OverlayValues[466] = d466
		ps866.OverlayValues[467] = d467
		ps866.OverlayValues[468] = d468
		ps866.OverlayValues[469] = d469
		ps866.OverlayValues[470] = d470
		ps866.OverlayValues[471] = d471
		ps866.OverlayValues[472] = d472
		ps866.OverlayValues[473] = d473
		ps866.OverlayValues[474] = d474
		ps866.OverlayValues[475] = d475
		ps866.OverlayValues[648] = d648
		ps866.OverlayValues[649] = d649
		ps866.OverlayValues[650] = d650
		ps866.OverlayValues[652] = d652
		ps866.OverlayValues[653] = d653
		ps866.OverlayValues[654] = d654
		ps866.OverlayValues[655] = d655
		ps866.OverlayValues[656] = d656
		ps866.OverlayValues[657] = d657
		ps866.OverlayValues[658] = d658
		ps866.OverlayValues[660] = d660
		ps866.OverlayValues[662] = d662
		ps866.OverlayValues[663] = d663
		ps866.OverlayValues[664] = d664
		ps866.OverlayValues[665] = d665
		ps866.OverlayValues[668] = d668
		ps866.OverlayValues[853] = d853
		ps866.OverlayValues[854] = d854
		ps866.OverlayValues[855] = d855
		ps866.OverlayValues[856] = d856
		ps866.OverlayValues[858] = d858
		ps866.OverlayValues[859] = d859
		ps866.OverlayValues[860] = d860
		ps866.OverlayValues[861] = d861
		ps866.OverlayValues[862] = d862
		ps866.OverlayValues[863] = d863
		ps866.OverlayValues[864] = d864
		ps866.OverlayValues[865] = d865
		ps866.PhiValues = make([]scm.JITValueDesc, 3)
		d867 = d8
		ps866.PhiValues[1] = d867
		d868 = d9
		ps866.PhiValues[2] = d868
		if ps866.General && bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		return bbs[1].RenderPS(ps866)
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
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
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
		if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != scm.LocNone {
			d282 = ps.OverlayValues[282]
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
		if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
			d294 = ps.OverlayValues[294]
		}
		if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != scm.LocNone {
			d296 = ps.OverlayValues[296]
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
		if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != scm.LocNone {
			d305 = ps.OverlayValues[305]
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
		if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != scm.LocNone {
			d435 = ps.OverlayValues[435]
		}
		if len(ps.OverlayValues) > 436 && ps.OverlayValues[436].Loc != scm.LocNone {
			d436 = ps.OverlayValues[436]
		}
		if len(ps.OverlayValues) > 437 && ps.OverlayValues[437].Loc != scm.LocNone {
			d437 = ps.OverlayValues[437]
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
		if len(ps.OverlayValues) > 648 && ps.OverlayValues[648].Loc != scm.LocNone {
			d648 = ps.OverlayValues[648]
		}
		if len(ps.OverlayValues) > 649 && ps.OverlayValues[649].Loc != scm.LocNone {
			d649 = ps.OverlayValues[649]
		}
		if len(ps.OverlayValues) > 650 && ps.OverlayValues[650].Loc != scm.LocNone {
			d650 = ps.OverlayValues[650]
		}
		if len(ps.OverlayValues) > 652 && ps.OverlayValues[652].Loc != scm.LocNone {
			d652 = ps.OverlayValues[652]
		}
		if len(ps.OverlayValues) > 653 && ps.OverlayValues[653].Loc != scm.LocNone {
			d653 = ps.OverlayValues[653]
		}
		if len(ps.OverlayValues) > 654 && ps.OverlayValues[654].Loc != scm.LocNone {
			d654 = ps.OverlayValues[654]
		}
		if len(ps.OverlayValues) > 655 && ps.OverlayValues[655].Loc != scm.LocNone {
			d655 = ps.OverlayValues[655]
		}
		if len(ps.OverlayValues) > 656 && ps.OverlayValues[656].Loc != scm.LocNone {
			d656 = ps.OverlayValues[656]
		}
		if len(ps.OverlayValues) > 657 && ps.OverlayValues[657].Loc != scm.LocNone {
			d657 = ps.OverlayValues[657]
		}
		if len(ps.OverlayValues) > 658 && ps.OverlayValues[658].Loc != scm.LocNone {
			d658 = ps.OverlayValues[658]
		}
		if len(ps.OverlayValues) > 660 && ps.OverlayValues[660].Loc != scm.LocNone {
			d660 = ps.OverlayValues[660]
		}
		if len(ps.OverlayValues) > 662 && ps.OverlayValues[662].Loc != scm.LocNone {
			d662 = ps.OverlayValues[662]
		}
		if len(ps.OverlayValues) > 663 && ps.OverlayValues[663].Loc != scm.LocNone {
			d663 = ps.OverlayValues[663]
		}
		if len(ps.OverlayValues) > 664 && ps.OverlayValues[664].Loc != scm.LocNone {
			d664 = ps.OverlayValues[664]
		}
		if len(ps.OverlayValues) > 665 && ps.OverlayValues[665].Loc != scm.LocNone {
			d665 = ps.OverlayValues[665]
		}
		if len(ps.OverlayValues) > 668 && ps.OverlayValues[668].Loc != scm.LocNone {
			d668 = ps.OverlayValues[668]
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
		if len(ps.OverlayValues) > 867 && ps.OverlayValues[867].Loc != scm.LocNone {
			d867 = ps.OverlayValues[867]
		}
		if len(ps.OverlayValues) > 868 && ps.OverlayValues[868].Loc != scm.LocNone {
			d868 = ps.OverlayValues[868]
		}
		ctx.ReclaimUntrackedRegs()
		d869 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d870 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d870)
		ctx.BindReg(r1, &d870)
		ctx.EnsureDesc(&d869)
		if d869.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d869, &d870)
		} else {
			switch d869.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d870, d869)
			case scm.TagInt:
				ctx.EmitMakeInt(d870, d869)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d870, d869)
			case scm.TagNil:
				ctx.EmitMakeNil(d870)
			default:
				ctx.EmitMovPairToResult(&d869, &d870)
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
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
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
		if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != scm.LocNone {
			d282 = ps.OverlayValues[282]
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
		if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
			d294 = ps.OverlayValues[294]
		}
		if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != scm.LocNone {
			d296 = ps.OverlayValues[296]
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
		if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != scm.LocNone {
			d305 = ps.OverlayValues[305]
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
		if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != scm.LocNone {
			d435 = ps.OverlayValues[435]
		}
		if len(ps.OverlayValues) > 436 && ps.OverlayValues[436].Loc != scm.LocNone {
			d436 = ps.OverlayValues[436]
		}
		if len(ps.OverlayValues) > 437 && ps.OverlayValues[437].Loc != scm.LocNone {
			d437 = ps.OverlayValues[437]
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
		if len(ps.OverlayValues) > 648 && ps.OverlayValues[648].Loc != scm.LocNone {
			d648 = ps.OverlayValues[648]
		}
		if len(ps.OverlayValues) > 649 && ps.OverlayValues[649].Loc != scm.LocNone {
			d649 = ps.OverlayValues[649]
		}
		if len(ps.OverlayValues) > 650 && ps.OverlayValues[650].Loc != scm.LocNone {
			d650 = ps.OverlayValues[650]
		}
		if len(ps.OverlayValues) > 652 && ps.OverlayValues[652].Loc != scm.LocNone {
			d652 = ps.OverlayValues[652]
		}
		if len(ps.OverlayValues) > 653 && ps.OverlayValues[653].Loc != scm.LocNone {
			d653 = ps.OverlayValues[653]
		}
		if len(ps.OverlayValues) > 654 && ps.OverlayValues[654].Loc != scm.LocNone {
			d654 = ps.OverlayValues[654]
		}
		if len(ps.OverlayValues) > 655 && ps.OverlayValues[655].Loc != scm.LocNone {
			d655 = ps.OverlayValues[655]
		}
		if len(ps.OverlayValues) > 656 && ps.OverlayValues[656].Loc != scm.LocNone {
			d656 = ps.OverlayValues[656]
		}
		if len(ps.OverlayValues) > 657 && ps.OverlayValues[657].Loc != scm.LocNone {
			d657 = ps.OverlayValues[657]
		}
		if len(ps.OverlayValues) > 658 && ps.OverlayValues[658].Loc != scm.LocNone {
			d658 = ps.OverlayValues[658]
		}
		if len(ps.OverlayValues) > 660 && ps.OverlayValues[660].Loc != scm.LocNone {
			d660 = ps.OverlayValues[660]
		}
		if len(ps.OverlayValues) > 662 && ps.OverlayValues[662].Loc != scm.LocNone {
			d662 = ps.OverlayValues[662]
		}
		if len(ps.OverlayValues) > 663 && ps.OverlayValues[663].Loc != scm.LocNone {
			d663 = ps.OverlayValues[663]
		}
		if len(ps.OverlayValues) > 664 && ps.OverlayValues[664].Loc != scm.LocNone {
			d664 = ps.OverlayValues[664]
		}
		if len(ps.OverlayValues) > 665 && ps.OverlayValues[665].Loc != scm.LocNone {
			d665 = ps.OverlayValues[665]
		}
		if len(ps.OverlayValues) > 668 && ps.OverlayValues[668].Loc != scm.LocNone {
			d668 = ps.OverlayValues[668]
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d4)
		d871 = d4
		_ = d871
		ctx.StabilizeDescForControlFlow(&d871)
		ctx.StabilizeDescForControlFlow(&d4)
		phiBase872 = ctx.AllocStack(int32(16))
		d873 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase872) + int32(0)}
		_ = d873
		lbl43 := ctx.ReserveLabel()
		bbpos_4_0 := int32(-1)
		_ = bbpos_4_0
		lbl44 := ctx.ReserveLabel()
		_ = lbl44
		bbpos_4_1 := int32(-1)
		_ = bbpos_4_1
		lbl45 := ctx.ReserveLabel()
		_ = lbl45
		bbpos_4_2 := int32(-1)
		_ = bbpos_4_2
		lbl46 := ctx.ReserveLabel()
		_ = lbl46
		bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl44)
		ctx.ResolveFixups()
		d873 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d871)
		ctx.EnsureDesc(&d871)
		var d874 scm.JITValueDesc
		if d871.Loc == scm.LocImm {
			d874 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d871.Imm.Int()))))}
		} else {
			r144 := ctx.AllocReg()
			ctx.EmitMovRegReg(r144, d871.Reg)
			ctx.EmitShlRegImm8(r144, 32)
			ctx.EmitShrRegImm8(r144, 32)
			d874 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
			ctx.BindReg(r144, &d874)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d875 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
			r145 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r145, fieldAddr)
			d875 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r145}
			ctx.BindReg(r145, &d875)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
			r146 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r146, thisptr.Reg, off)
			d875 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r146}
			ctx.BindReg(r146, &d875)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d875)
		ctx.EnsureDesc(&d875)
		var d876 scm.JITValueDesc
		if d875.Loc == scm.LocImm {
			d876 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d875.Imm.Int()))))}
		} else {
			r147 := ctx.AllocReg()
			ctx.EmitMovRegReg(r147, d875.Reg)
			ctx.EmitShlRegImm8(r147, 56)
			ctx.EmitShrRegImm8(r147, 56)
			d876 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r147}
			ctx.BindReg(r147, &d876)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d874)
		ctx.EnsureDesc(&d876)
		ctx.EnsureDescsTogether(&d874, &d876)
		var d877 scm.JITValueDesc
		if d874.Loc == scm.LocImm && d876.Loc == scm.LocImm {
			d877 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d874.Imm.Int() * d876.Imm.Int())}
		} else if d874.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d876.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d874.Imm.Int()))
			ctx.EmitImulInt64(scratch, d876.Reg)
			d877 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d877)
		} else if d876.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d874.Reg)
			ctx.EmitMovRegReg(scratch, d874.Reg)
			if d876.Imm.Int() >= -2147483648 && d876.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d876.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d876.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d877 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d877)
		} else {
			r148 := ctx.AllocRegExcept(d874.Reg, d876.Reg)
			ctx.EmitMovRegReg(r148, d874.Reg)
			ctx.EmitImulInt64(r148, d876.Reg)
			d877 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r148}
			ctx.BindReg(r148, &d877)
		}
		if d877.Loc == scm.LocReg && d874.Loc == scm.LocReg && d877.Reg == d874.Reg {
			ctx.TransferReg(d874.Reg)
			d874.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d877)
		ctx.FreeDesc(&d874)
		ctx.FreeDesc(&d876)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d878 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 0
			r149 := ctx.AllocReg()
			r150 := ctx.AllocRegExcept(r149)
			r151 := ctx.AllocRegExcept(r149, r150)
			ctx.EmitMovRegMem64(r149, fieldAddr)
			ctx.EmitMovRegMem64(r150, fieldAddr+8)
			ctx.EmitMovRegMem64(r151, fieldAddr+16)
			d878 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r149, Reg2: r150, Reg3: r151}
			ctx.BindReg(r149, &d878)
			ctx.BindReg(r150, &d878)
			ctx.BindReg(r151, &d878)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 0)
			r152 := ctx.AllocReg()
			r153 := ctx.AllocRegExcept(r152)
			r154 := ctx.AllocRegExcept(r152, r153)
			ctx.EmitMovRegMem(r152, thisptr.Reg, off)
			ctx.EmitMovRegMem(r153, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r154, thisptr.Reg, off+16)
			d878 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r152, Reg2: r153, Reg3: r154}
			ctx.BindReg(r152, &d878)
			ctx.BindReg(r153, &d878)
			ctx.BindReg(r154, &d878)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d877)
		var d879 scm.JITValueDesc
		if d877.Loc == scm.LocImm {
			d879 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d877.Imm.Int() / 64)}
		} else {
			r155 := ctx.AllocRegExcept(d877.Reg)
			ctx.EmitMovRegReg(r155, d877.Reg)
			ctx.EmitShrRegImm8(r155, 6)
			d879 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r155}
			ctx.BindReg(r155, &d879)
		}
		if d879.Loc == scm.LocReg && d877.Loc == scm.LocReg && d879.Reg == d877.Reg {
			ctx.TransferReg(d877.Reg)
			d877.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d879)
		ctx.ReclaimUntrackedRegs()
		d881 = ctx.EmitSliceElementAddress(&d878, &d879, 8)
		ctx.EnsureDesc(&d881)
		ctx.EmitMovRegMem(d881.Reg, d881.Reg, 0)
		d880 = d881
		ctx.FreeDesc(&d879)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d877)
		var d882 scm.JITValueDesc
		if d877.Loc == scm.LocImm {
			d882 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d877.Imm.Int() % 64)}
		} else {
			r156 := ctx.AllocRegExcept(d877.Reg)
			ctx.EmitMovRegReg(r156, d877.Reg)
			ctx.EmitAndRegImm32(r156, 63)
			d882 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r156}
			ctx.BindReg(r156, &d882)
		}
		if d882.Loc == scm.LocReg && d877.Loc == scm.LocReg && d882.Reg == d877.Reg {
			ctx.TransferReg(d877.Reg)
			d877.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d880)
		ctx.EnsureDesc(&d882)
		var d883 scm.JITValueDesc
		if d880.Loc == scm.LocImm && d882.Loc == scm.LocImm {
			d883 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d880.Imm.Int()) << uint64(d882.Imm.Int())))}
		} else if d882.Loc == scm.LocImm {
			r157 := ctx.AllocRegExcept(d880.Reg)
			ctx.EmitMovRegReg(r157, d880.Reg)
			ctx.EmitShlRegImm8(r157, uint8(d882.Imm.Int()))
			d883 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r157}
			ctx.BindReg(r157, &d883)
		} else {
			{
				shiftSrc := d880.Reg
				r158 := ctx.AllocRegExcept(d880.Reg)
				ctx.EmitMovRegReg(r158, d880.Reg)
				shiftSrc = r158
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d882.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d882.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d882.Reg)
				}
				ctx.EmitShlRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d883 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d883)
			}
		}
		if d883.Loc == scm.LocReg && d880.Loc == scm.LocReg && d883.Reg == d880.Reg {
			ctx.TransferReg(d880.Reg)
			d880.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d883)
		ctx.FreeDesc(&d880)
		ctx.FreeDesc(&d882)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d877)
		var d884 scm.JITValueDesc
		if d877.Loc == scm.LocImm {
			d884 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d877.Imm.Int() % 64)}
		} else {
			r159 := ctx.AllocRegExcept(d877.Reg)
			ctx.EmitMovRegReg(r159, d877.Reg)
			ctx.EmitAndRegImm32(r159, 63)
			d884 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
			ctx.BindReg(r159, &d884)
		}
		if d884.Loc == scm.LocReg && d877.Loc == scm.LocReg && d884.Reg == d877.Reg {
			ctx.TransferReg(d877.Reg)
			d877.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d875)
		ctx.EnsureDesc(&d875)
		var d885 scm.JITValueDesc
		if d875.Loc == scm.LocImm {
			d885 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d875.Imm.Int()))))}
		} else {
			r160 := ctx.AllocReg()
			ctx.EmitMovRegReg(r160, d875.Reg)
			ctx.EmitShlRegImm8(r160, 56)
			ctx.EmitShrRegImm8(r160, 56)
			d885 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r160}
			ctx.BindReg(r160, &d885)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d884)
		ctx.EnsureDesc(&d885)
		ctx.EnsureDescsTogether(&d884, &d885)
		var d886 scm.JITValueDesc
		if d884.Loc == scm.LocImm && d885.Loc == scm.LocImm {
			d886 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d884.Imm.Int() + d885.Imm.Int())}
		} else if d885.Loc == scm.LocImm && d885.Imm.Int() == 0 {
			r161 := ctx.AllocRegExcept(d884.Reg)
			ctx.EmitMovRegReg(r161, d884.Reg)
			d886 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r161}
			ctx.BindReg(r161, &d886)
		} else if d884.Loc == scm.LocImm && d884.Imm.Int() == 0 {
			d886 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d885.Reg}
			ctx.BindReg(d885.Reg, &d886)
		} else if d884.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d885.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d884.Imm.Int()))
			ctx.EmitAddInt64(scratch, d885.Reg)
			d886 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d886)
		} else if d885.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d884.Reg)
			ctx.EmitMovRegReg(scratch, d884.Reg)
			if d885.Imm.Int() >= -2147483648 && d885.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d885.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d885.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d886 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d886)
		} else {
			r162 := ctx.AllocRegExcept(d884.Reg, d885.Reg)
			ctx.EmitMovRegReg(r162, d884.Reg)
			ctx.EmitAddInt64(r162, d885.Reg)
			d886 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r162}
			ctx.BindReg(r162, &d886)
		}
		if d886.Loc == scm.LocReg && d884.Loc == scm.LocReg && d886.Reg == d884.Reg {
			ctx.TransferReg(d884.Reg)
			d884.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d884)
		ctx.FreeDesc(&d885)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d886)
		var d887 scm.JITValueDesc
		if d886.Loc == scm.LocImm {
			d887 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d886.Imm.Int()) > uint64(0x40))}
		} else {
			r163 := ctx.AllocRegExcept(d886.Reg)
			ctx.EmitCmpRegImm32(d886.Reg, 64)
			ctx.EmitSetcc(r163, scm.CondUnsignedAbove)
			d887 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r163}
			ctx.BindReg(r163, &d887)
		}
		ctx.FreeDesc(&d886)
		ctx.ReclaimUntrackedRegs()
		d888 = d887
		ctx.EnsureDesc(&d888)
		if d888.Loc != scm.LocImm && d888.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		lbl47 := ctx.ReserveLabel()
		lbl48 := ctx.ReserveLabel()
		if d888.Loc == scm.LocImm {
			if d888.Imm.Bool() {
				ctx.MarkLabel(lbl47)
				ctx.EmitJmp(lbl45)
			} else {
				ctx.MarkLabel(lbl48)
				ctx.SyncDesc(&d883)
				if d883.Loc == scm.LocReg {
					ctx.ProtectReg(d883.Reg)
				} else if d883.Loc == scm.LocRegPair {
					ctx.ProtectReg(d883.Reg)
					ctx.ProtectReg(d883.Reg2)
				}
				d889 = d883
				if d889.Loc == scm.LocNone {
					panic("jit: phi source has no location")
				}
				ctx.EnsureDesc(&d889)
				ctx.EmitStoreToStack(d889, int32(phiBase872)+int32(0))
				if d883.Loc == scm.LocReg {
					ctx.UnprotectReg(d883.Reg)
				} else if d883.Loc == scm.LocRegPair {
					ctx.UnprotectReg(d883.Reg)
					ctx.UnprotectReg(d883.Reg2)
				}
				ctx.EmitJmp(lbl46)
			}
		} else {
			ctx.EmitCmpRegImm32(d888.Reg, 0)
			ctx.EmitJump(scm.CondNotEqual, lbl47)
			ctx.EmitJmp(lbl48)
			ctx.MarkLabel(lbl47)
			ctx.EmitJmp(lbl45)
			ctx.MarkLabel(lbl48)
			ctx.SyncDesc(&d883)
			if d883.Loc == scm.LocReg {
				ctx.ProtectReg(d883.Reg)
			} else if d883.Loc == scm.LocRegPair {
				ctx.ProtectReg(d883.Reg)
				ctx.ProtectReg(d883.Reg2)
			}
			d890 = d883
			if d890.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d890)
			ctx.EmitStoreToStack(d890, int32(phiBase872)+int32(0))
			if d883.Loc == scm.LocReg {
				ctx.UnprotectReg(d883.Reg)
			} else if d883.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d883.Reg)
				ctx.UnprotectReg(d883.Reg2)
			}
			ctx.EmitJmp(lbl46)
		}
		ctx.FreeDesc(&d887)
		bbpos_4_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl46)
		ctx.ResolveFixups()
		d873 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d875)
		ctx.EnsureDesc(&d875)
		var d891 scm.JITValueDesc
		if d875.Loc == scm.LocImm {
			d891 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d875.Imm.Int()))))}
		} else {
			r164 := ctx.AllocReg()
			ctx.EmitMovRegReg(r164, d875.Reg)
			ctx.EmitShlRegImm8(r164, 56)
			ctx.EmitShrRegImm8(r164, 56)
			d891 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r164}
			ctx.BindReg(r164, &d891)
		}
		ctx.ReclaimUntrackedRegs()
		d892 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d891)
		ctx.EnsureDescsTogether(&d892, &d891)
		var d893 scm.JITValueDesc
		if d892.Loc == scm.LocImm && d891.Loc == scm.LocImm {
			d893 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d892.Imm.Int() - d891.Imm.Int())}
		} else if d891.Loc == scm.LocImm && d891.Imm.Int() == 0 {
			r165 := ctx.AllocRegExcept(d892.Reg)
			ctx.EmitMovRegReg(r165, d892.Reg)
			d893 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r165}
			ctx.BindReg(r165, &d893)
		} else if d892.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d891.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d892.Imm.Int()))
			ctx.EmitSubInt64(scratch, d891.Reg)
			d893 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d893)
		} else if d891.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d892.Reg)
			ctx.EmitMovRegReg(scratch, d892.Reg)
			if d891.Imm.Int() >= -2147483648 && d891.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d891.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d891.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d893 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d893)
		} else {
			r166 := ctx.AllocRegExcept(d892.Reg, d891.Reg)
			ctx.EmitMovRegReg(r166, d892.Reg)
			ctx.EmitSubInt64(r166, d891.Reg)
			d893 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r166}
			ctx.BindReg(r166, &d893)
		}
		if d893.Loc == scm.LocReg && d892.Loc == scm.LocReg && d893.Reg == d892.Reg {
			ctx.TransferReg(d892.Reg)
			d892.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d891)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d873)
		ctx.EnsureDesc(&d893)
		var d894 scm.JITValueDesc
		if d873.Loc == scm.LocImm && d893.Loc == scm.LocImm {
			d894 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d873.Imm.Int()) >> uint64(d893.Imm.Int())))}
		} else if d893.Loc == scm.LocImm {
			r167 := ctx.AllocRegExcept(d873.Reg)
			ctx.EmitMovRegReg(r167, d873.Reg)
			ctx.EmitShrRegImm8(r167, uint8(d893.Imm.Int()))
			d894 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r167}
			ctx.BindReg(r167, &d894)
		} else {
			{
				shiftSrc := d873.Reg
				r168 := ctx.AllocRegExcept(d873.Reg)
				ctx.EmitMovRegReg(r168, d873.Reg)
				shiftSrc = r168
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d893.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d893.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d893.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d894 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d894)
			}
		}
		if d894.Loc == scm.LocReg && d873.Loc == scm.LocReg && d894.Reg == d873.Reg {
			ctx.TransferReg(d873.Reg)
			d873.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d873)
		ctx.FreeDesc(&d893)
		ctx.ReclaimUntrackedRegs()
		r169 := ctx.AllocReg()
		ctx.EnsureDesc(&d894)
		ctx.EnsureDesc(&d894)
		if d894.Loc == scm.LocRegPair {
			panic("jit: scalar inline return has scm.LocRegPair")
		} else {
			ctx.EmitMovToReg(r169, d894)
		}
		ctx.EmitJmp(lbl43)
		bbpos_4_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl45)
		ctx.ResolveFixups()
		d873 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d877)
		var d895 scm.JITValueDesc
		if d877.Loc == scm.LocImm {
			d895 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d877.Imm.Int() / 64)}
		} else {
			r170 := ctx.AllocRegExcept(d877.Reg)
			ctx.EmitMovRegReg(r170, d877.Reg)
			ctx.EmitShrRegImm8(r170, 6)
			d895 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r170}
			ctx.BindReg(r170, &d895)
		}
		if d895.Loc == scm.LocReg && d877.Loc == scm.LocReg && d895.Reg == d877.Reg {
			ctx.TransferReg(d877.Reg)
			d877.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d895)
		ctx.EnsureDesc(&d895)
		var d896 scm.JITValueDesc
		if d895.Loc == scm.LocImm {
			d896 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d895.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d895.Reg)
			ctx.EmitMovRegReg(scratch, d895.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d896 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d896)
		}
		if d896.Loc == scm.LocReg && d895.Loc == scm.LocReg && d896.Reg == d895.Reg {
			ctx.TransferReg(d895.Reg)
			d895.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d895)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d896)
		ctx.ReclaimUntrackedRegs()
		d898 = ctx.EmitSliceElementAddress(&d878, &d896, 8)
		ctx.EnsureDesc(&d898)
		ctx.EmitMovRegMem(d898.Reg, d898.Reg, 0)
		d897 = d898
		ctx.FreeDesc(&d896)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d877)
		var d899 scm.JITValueDesc
		if d877.Loc == scm.LocImm {
			d899 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d877.Imm.Int() % 64)}
		} else {
			r171 := ctx.AllocRegExcept(d877.Reg)
			ctx.EmitMovRegReg(r171, d877.Reg)
			ctx.EmitAndRegImm32(r171, 63)
			d899 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r171}
			ctx.BindReg(r171, &d899)
		}
		if d899.Loc == scm.LocReg && d877.Loc == scm.LocReg && d899.Reg == d877.Reg {
			ctx.TransferReg(d877.Reg)
			d877.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d877)
		ctx.ReclaimUntrackedRegs()
		d900 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d899)
		ctx.EnsureDescsTogether(&d900, &d899)
		var d901 scm.JITValueDesc
		if d900.Loc == scm.LocImm && d899.Loc == scm.LocImm {
			d901 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d900.Imm.Int() - d899.Imm.Int())}
		} else if d899.Loc == scm.LocImm && d899.Imm.Int() == 0 {
			r172 := ctx.AllocRegExcept(d900.Reg)
			ctx.EmitMovRegReg(r172, d900.Reg)
			d901 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
			ctx.BindReg(r172, &d901)
		} else if d900.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d899.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d900.Imm.Int()))
			ctx.EmitSubInt64(scratch, d899.Reg)
			d901 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d901)
		} else if d899.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d900.Reg)
			ctx.EmitMovRegReg(scratch, d900.Reg)
			if d899.Imm.Int() >= -2147483648 && d899.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d899.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d899.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d901 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d901)
		} else {
			r173 := ctx.AllocRegExcept(d900.Reg, d899.Reg)
			ctx.EmitMovRegReg(r173, d900.Reg)
			ctx.EmitSubInt64(r173, d899.Reg)
			d901 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r173}
			ctx.BindReg(r173, &d901)
		}
		if d901.Loc == scm.LocReg && d900.Loc == scm.LocReg && d901.Reg == d900.Reg {
			ctx.TransferReg(d900.Reg)
			d900.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d899)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d897)
		ctx.EnsureDesc(&d901)
		var d902 scm.JITValueDesc
		if d897.Loc == scm.LocImm && d901.Loc == scm.LocImm {
			d902 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d897.Imm.Int()) >> uint64(d901.Imm.Int())))}
		} else if d901.Loc == scm.LocImm {
			r174 := ctx.AllocRegExcept(d897.Reg)
			ctx.EmitMovRegReg(r174, d897.Reg)
			ctx.EmitShrRegImm8(r174, uint8(d901.Imm.Int()))
			d902 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r174}
			ctx.BindReg(r174, &d902)
		} else {
			{
				shiftSrc := d897.Reg
				r175 := ctx.AllocRegExcept(d897.Reg)
				ctx.EmitMovRegReg(r175, d897.Reg)
				shiftSrc = r175
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d901.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d901.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d901.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d902 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d902)
			}
		}
		if d902.Loc == scm.LocReg && d897.Loc == scm.LocReg && d902.Reg == d897.Reg {
			ctx.TransferReg(d897.Reg)
			d897.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d897)
		ctx.FreeDesc(&d901)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d883)
		ctx.EnsureDesc(&d902)
		var d903 scm.JITValueDesc
		if d883.Loc == scm.LocImm && d902.Loc == scm.LocImm {
			d903 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d883.Imm.Int() | d902.Imm.Int())}
		} else if d883.Loc == scm.LocImm && d883.Imm.Int() == 0 {
			d903 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d902.Reg}
			ctx.BindReg(d902.Reg, &d903)
		} else if d902.Loc == scm.LocImm && d902.Imm.Int() == 0 {
			r176 := ctx.AllocRegExcept(d883.Reg)
			ctx.EmitMovRegReg(r176, d883.Reg)
			d903 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r176}
			ctx.BindReg(r176, &d903)
		} else if d883.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d902.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d883.Imm.Int()))
			ctx.EmitOrInt64(scratch, d902.Reg)
			d903 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d903)
		} else if d902.Loc == scm.LocImm {
			r177 := ctx.AllocRegExcept(d883.Reg)
			ctx.EmitMovRegReg(r177, d883.Reg)
			if d902.Imm.Int() >= -2147483648 && d902.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r177, int32(d902.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d902.Imm.Int()))
				ctx.EmitOrInt64(r177, scm.RegR11)
			}
			d903 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r177}
			ctx.BindReg(r177, &d903)
		} else {
			r178 := ctx.AllocRegExcept(d883.Reg, d902.Reg)
			ctx.EmitMovRegReg(r178, d883.Reg)
			ctx.EmitOrInt64(r178, d902.Reg)
			d903 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r178}
			ctx.BindReg(r178, &d903)
		}
		if d903.Loc == scm.LocReg && d883.Loc == scm.LocReg && d903.Reg == d883.Reg {
			ctx.TransferReg(d883.Reg)
			d883.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d903)
		ctx.EmitStoreToStack(d903, int32(phiBase872)+int32(0))
		ctx.StabilizeDescForControlFlow(&d903)
		ctx.FreeDesc(&d902)
		ctx.ReclaimUntrackedRegs()
		ctx.EmitJmp(lbl46)
		ctx.MarkLabel(lbl43)
		d904 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r169}
		ctx.BindReg(r169, &d904)
		ctx.BindReg(r169, &d904)
		ctx.EnsureDesc(&d904)
		ctx.EnsureDesc(&d904)
		var d905 scm.JITValueDesc
		if d904.Loc == scm.LocImm {
			d905 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d904.Imm.Int()))))}
		} else {
			r179 := ctx.AllocReg()
			ctx.EmitMovRegReg(r179, d904.Reg)
			d905 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r179}
			ctx.BindReg(r179, &d905)
		}
		ctx.FreeDesc(&d904)
		var d906 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 32
			r180 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r180, fieldAddr)
			d906 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r180}
			ctx.BindReg(r180, &d906)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 32)
			r181 := ctx.AllocReg()
			ctx.EmitMovRegMem(r181, thisptr.Reg, off)
			d906 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r181}
			ctx.BindReg(r181, &d906)
		}
		ctx.EnsureDesc(&d905)
		ctx.EnsureDesc(&d906)
		ctx.EnsureDescsTogether(&d905, &d906)
		var d907 scm.JITValueDesc
		if d905.Loc == scm.LocImm && d906.Loc == scm.LocImm {
			d907 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d905.Imm.Int() + d906.Imm.Int())}
		} else if d906.Loc == scm.LocImm && d906.Imm.Int() == 0 {
			r182 := ctx.AllocRegExcept(d905.Reg)
			ctx.EmitMovRegReg(r182, d905.Reg)
			d907 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r182}
			ctx.BindReg(r182, &d907)
		} else if d905.Loc == scm.LocImm && d905.Imm.Int() == 0 {
			d907 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d906.Reg}
			ctx.BindReg(d906.Reg, &d907)
		} else if d905.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d906.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d905.Imm.Int()))
			ctx.EmitAddInt64(scratch, d906.Reg)
			d907 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d907)
		} else if d906.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d905.Reg)
			ctx.EmitMovRegReg(scratch, d905.Reg)
			if d906.Imm.Int() >= -2147483648 && d906.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d906.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d906.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d907 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d907)
		} else {
			r183 := ctx.AllocRegExcept(d905.Reg, d906.Reg)
			ctx.EmitMovRegReg(r183, d905.Reg)
			ctx.EmitAddInt64(r183, d906.Reg)
			d907 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r183}
			ctx.BindReg(r183, &d907)
		}
		if d907.Loc == scm.LocReg && d905.Loc == scm.LocReg && d907.Reg == d905.Reg {
			ctx.TransferReg(d905.Reg)
			d905.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d905)
		ctx.EnsureDesc(&d4)
		d908 = d4
		_ = d908
		ctx.StabilizeDescForControlFlow(&d908)
		phiBase909 = ctx.AllocStack(int32(16))
		d910 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase909) + int32(0)}
		_ = d910
		lbl49 := ctx.ReserveLabel()
		bbpos_5_0 := int32(-1)
		_ = bbpos_5_0
		lbl50 := ctx.ReserveLabel()
		_ = lbl50
		bbpos_5_1 := int32(-1)
		_ = bbpos_5_1
		lbl51 := ctx.ReserveLabel()
		_ = lbl51
		bbpos_5_2 := int32(-1)
		_ = bbpos_5_2
		lbl52 := ctx.ReserveLabel()
		_ = lbl52
		bbpos_5_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl50)
		ctx.ResolveFixups()
		d910 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d908)
		ctx.EnsureDesc(&d908)
		var d911 scm.JITValueDesc
		if d908.Loc == scm.LocImm {
			d911 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d908.Imm.Int()))))}
		} else {
			r184 := ctx.AllocReg()
			ctx.EmitMovRegReg(r184, d908.Reg)
			ctx.EmitShlRegImm8(r184, 32)
			ctx.EmitShrRegImm8(r184, 32)
			d911 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r184}
			ctx.BindReg(r184, &d911)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d912 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			r185 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r185, fieldAddr)
			d912 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r185}
			ctx.BindReg(r185, &d912)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			r186 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r186, thisptr.Reg, off)
			d912 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r186}
			ctx.BindReg(r186, &d912)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d912)
		ctx.EnsureDesc(&d912)
		var d913 scm.JITValueDesc
		if d912.Loc == scm.LocImm {
			d913 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d912.Imm.Int()))))}
		} else {
			r187 := ctx.AllocReg()
			ctx.EmitMovRegReg(r187, d912.Reg)
			ctx.EmitShlRegImm8(r187, 56)
			ctx.EmitShrRegImm8(r187, 56)
			d913 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r187}
			ctx.BindReg(r187, &d913)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d911)
		ctx.EnsureDesc(&d913)
		ctx.EnsureDescsTogether(&d911, &d913)
		var d914 scm.JITValueDesc
		if d911.Loc == scm.LocImm && d913.Loc == scm.LocImm {
			d914 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d911.Imm.Int() * d913.Imm.Int())}
		} else if d911.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d913.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d911.Imm.Int()))
			ctx.EmitImulInt64(scratch, d913.Reg)
			d914 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d914)
		} else if d913.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d911.Reg)
			ctx.EmitMovRegReg(scratch, d911.Reg)
			if d913.Imm.Int() >= -2147483648 && d913.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d913.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d913.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d914 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d914)
		} else {
			r188 := ctx.AllocRegExcept(d911.Reg, d913.Reg)
			ctx.EmitMovRegReg(r188, d911.Reg)
			ctx.EmitImulInt64(r188, d913.Reg)
			d914 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r188}
			ctx.BindReg(r188, &d914)
		}
		if d914.Loc == scm.LocReg && d911.Loc == scm.LocReg && d914.Reg == d911.Reg {
			ctx.TransferReg(d911.Reg)
			d911.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d914)
		ctx.FreeDesc(&d911)
		ctx.FreeDesc(&d913)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d915 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
			r189 := ctx.AllocReg()
			r190 := ctx.AllocRegExcept(r189)
			r191 := ctx.AllocRegExcept(r189, r190)
			ctx.EmitMovRegMem64(r189, fieldAddr)
			ctx.EmitMovRegMem64(r190, fieldAddr+8)
			ctx.EmitMovRegMem64(r191, fieldAddr+16)
			d915 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r189, Reg2: r190, Reg3: r191}
			ctx.BindReg(r189, &d915)
			ctx.BindReg(r190, &d915)
			ctx.BindReg(r191, &d915)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
			r192 := ctx.AllocReg()
			r193 := ctx.AllocRegExcept(r192)
			r194 := ctx.AllocRegExcept(r192, r193)
			ctx.EmitMovRegMem(r192, thisptr.Reg, off)
			ctx.EmitMovRegMem(r193, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r194, thisptr.Reg, off+16)
			d915 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r192, Reg2: r193, Reg3: r194}
			ctx.BindReg(r192, &d915)
			ctx.BindReg(r193, &d915)
			ctx.BindReg(r194, &d915)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d914)
		var d916 scm.JITValueDesc
		if d914.Loc == scm.LocImm {
			d916 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d914.Imm.Int() / 64)}
		} else {
			r195 := ctx.AllocRegExcept(d914.Reg)
			ctx.EmitMovRegReg(r195, d914.Reg)
			ctx.EmitShrRegImm8(r195, 6)
			d916 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r195}
			ctx.BindReg(r195, &d916)
		}
		if d916.Loc == scm.LocReg && d914.Loc == scm.LocReg && d916.Reg == d914.Reg {
			ctx.TransferReg(d914.Reg)
			d914.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d916)
		ctx.ReclaimUntrackedRegs()
		d918 = ctx.EmitSliceElementAddress(&d915, &d916, 8)
		ctx.EnsureDesc(&d918)
		ctx.EmitMovRegMem(d918.Reg, d918.Reg, 0)
		d917 = d918
		ctx.FreeDesc(&d916)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d914)
		var d919 scm.JITValueDesc
		if d914.Loc == scm.LocImm {
			d919 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d914.Imm.Int() % 64)}
		} else {
			r196 := ctx.AllocRegExcept(d914.Reg)
			ctx.EmitMovRegReg(r196, d914.Reg)
			ctx.EmitAndRegImm32(r196, 63)
			d919 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r196}
			ctx.BindReg(r196, &d919)
		}
		if d919.Loc == scm.LocReg && d914.Loc == scm.LocReg && d919.Reg == d914.Reg {
			ctx.TransferReg(d914.Reg)
			d914.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d917)
		ctx.EnsureDesc(&d919)
		var d920 scm.JITValueDesc
		if d917.Loc == scm.LocImm && d919.Loc == scm.LocImm {
			d920 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d917.Imm.Int()) << uint64(d919.Imm.Int())))}
		} else if d919.Loc == scm.LocImm {
			r197 := ctx.AllocRegExcept(d917.Reg)
			ctx.EmitMovRegReg(r197, d917.Reg)
			ctx.EmitShlRegImm8(r197, uint8(d919.Imm.Int()))
			d920 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r197}
			ctx.BindReg(r197, &d920)
		} else {
			{
				shiftSrc := d917.Reg
				r198 := ctx.AllocRegExcept(d917.Reg)
				ctx.EmitMovRegReg(r198, d917.Reg)
				shiftSrc = r198
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d919.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d919.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d919.Reg)
				}
				ctx.EmitShlRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d920 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d920)
			}
		}
		if d920.Loc == scm.LocReg && d917.Loc == scm.LocReg && d920.Reg == d917.Reg {
			ctx.TransferReg(d917.Reg)
			d917.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d920)
		ctx.FreeDesc(&d917)
		ctx.FreeDesc(&d919)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d914)
		var d921 scm.JITValueDesc
		if d914.Loc == scm.LocImm {
			d921 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d914.Imm.Int() % 64)}
		} else {
			r199 := ctx.AllocRegExcept(d914.Reg)
			ctx.EmitMovRegReg(r199, d914.Reg)
			ctx.EmitAndRegImm32(r199, 63)
			d921 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r199}
			ctx.BindReg(r199, &d921)
		}
		if d921.Loc == scm.LocReg && d914.Loc == scm.LocReg && d921.Reg == d914.Reg {
			ctx.TransferReg(d914.Reg)
			d914.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d912)
		ctx.EnsureDesc(&d912)
		var d922 scm.JITValueDesc
		if d912.Loc == scm.LocImm {
			d922 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d912.Imm.Int()))))}
		} else {
			r200 := ctx.AllocReg()
			ctx.EmitMovRegReg(r200, d912.Reg)
			ctx.EmitShlRegImm8(r200, 56)
			ctx.EmitShrRegImm8(r200, 56)
			d922 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r200}
			ctx.BindReg(r200, &d922)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d921)
		ctx.EnsureDesc(&d922)
		ctx.EnsureDescsTogether(&d921, &d922)
		var d923 scm.JITValueDesc
		if d921.Loc == scm.LocImm && d922.Loc == scm.LocImm {
			d923 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d921.Imm.Int() + d922.Imm.Int())}
		} else if d922.Loc == scm.LocImm && d922.Imm.Int() == 0 {
			r201 := ctx.AllocRegExcept(d921.Reg)
			ctx.EmitMovRegReg(r201, d921.Reg)
			d923 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r201}
			ctx.BindReg(r201, &d923)
		} else if d921.Loc == scm.LocImm && d921.Imm.Int() == 0 {
			d923 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d922.Reg}
			ctx.BindReg(d922.Reg, &d923)
		} else if d921.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d922.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d921.Imm.Int()))
			ctx.EmitAddInt64(scratch, d922.Reg)
			d923 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d923)
		} else if d922.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d921.Reg)
			ctx.EmitMovRegReg(scratch, d921.Reg)
			if d922.Imm.Int() >= -2147483648 && d922.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d922.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d922.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d923 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d923)
		} else {
			r202 := ctx.AllocRegExcept(d921.Reg, d922.Reg)
			ctx.EmitMovRegReg(r202, d921.Reg)
			ctx.EmitAddInt64(r202, d922.Reg)
			d923 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r202}
			ctx.BindReg(r202, &d923)
		}
		if d923.Loc == scm.LocReg && d921.Loc == scm.LocReg && d923.Reg == d921.Reg {
			ctx.TransferReg(d921.Reg)
			d921.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d921)
		ctx.FreeDesc(&d922)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d923)
		var d924 scm.JITValueDesc
		if d923.Loc == scm.LocImm {
			d924 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d923.Imm.Int()) > uint64(0x40))}
		} else {
			r203 := ctx.AllocRegExcept(d923.Reg)
			ctx.EmitCmpRegImm32(d923.Reg, 64)
			ctx.EmitSetcc(r203, scm.CondUnsignedAbove)
			d924 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r203}
			ctx.BindReg(r203, &d924)
		}
		ctx.FreeDesc(&d923)
		ctx.ReclaimUntrackedRegs()
		d925 = d924
		ctx.EnsureDesc(&d925)
		if d925.Loc != scm.LocImm && d925.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		lbl53 := ctx.ReserveLabel()
		lbl54 := ctx.ReserveLabel()
		if d925.Loc == scm.LocImm {
			if d925.Imm.Bool() {
				ctx.MarkLabel(lbl53)
				ctx.EmitJmp(lbl51)
			} else {
				ctx.MarkLabel(lbl54)
				ctx.SyncDesc(&d920)
				if d920.Loc == scm.LocReg {
					ctx.ProtectReg(d920.Reg)
				} else if d920.Loc == scm.LocRegPair {
					ctx.ProtectReg(d920.Reg)
					ctx.ProtectReg(d920.Reg2)
				}
				d926 = d920
				if d926.Loc == scm.LocNone {
					panic("jit: phi source has no location")
				}
				ctx.EnsureDesc(&d926)
				ctx.EmitStoreToStack(d926, int32(phiBase909)+int32(0))
				if d920.Loc == scm.LocReg {
					ctx.UnprotectReg(d920.Reg)
				} else if d920.Loc == scm.LocRegPair {
					ctx.UnprotectReg(d920.Reg)
					ctx.UnprotectReg(d920.Reg2)
				}
				ctx.EmitJmp(lbl52)
			}
		} else {
			ctx.EmitCmpRegImm32(d925.Reg, 0)
			ctx.EmitJump(scm.CondNotEqual, lbl53)
			ctx.EmitJmp(lbl54)
			ctx.MarkLabel(lbl53)
			ctx.EmitJmp(lbl51)
			ctx.MarkLabel(lbl54)
			ctx.SyncDesc(&d920)
			if d920.Loc == scm.LocReg {
				ctx.ProtectReg(d920.Reg)
			} else if d920.Loc == scm.LocRegPair {
				ctx.ProtectReg(d920.Reg)
				ctx.ProtectReg(d920.Reg2)
			}
			d927 = d920
			if d927.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d927)
			ctx.EmitStoreToStack(d927, int32(phiBase909)+int32(0))
			if d920.Loc == scm.LocReg {
				ctx.UnprotectReg(d920.Reg)
			} else if d920.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d920.Reg)
				ctx.UnprotectReg(d920.Reg2)
			}
			ctx.EmitJmp(lbl52)
		}
		ctx.FreeDesc(&d924)
		bbpos_5_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl52)
		ctx.ResolveFixups()
		d910 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d912)
		ctx.EnsureDesc(&d912)
		var d928 scm.JITValueDesc
		if d912.Loc == scm.LocImm {
			d928 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d912.Imm.Int()))))}
		} else {
			r204 := ctx.AllocReg()
			ctx.EmitMovRegReg(r204, d912.Reg)
			ctx.EmitShlRegImm8(r204, 56)
			ctx.EmitShrRegImm8(r204, 56)
			d928 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r204}
			ctx.BindReg(r204, &d928)
		}
		ctx.ReclaimUntrackedRegs()
		d929 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d928)
		ctx.EnsureDescsTogether(&d929, &d928)
		var d930 scm.JITValueDesc
		if d929.Loc == scm.LocImm && d928.Loc == scm.LocImm {
			d930 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d929.Imm.Int() - d928.Imm.Int())}
		} else if d928.Loc == scm.LocImm && d928.Imm.Int() == 0 {
			r205 := ctx.AllocRegExcept(d929.Reg)
			ctx.EmitMovRegReg(r205, d929.Reg)
			d930 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r205}
			ctx.BindReg(r205, &d930)
		} else if d929.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d928.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d929.Imm.Int()))
			ctx.EmitSubInt64(scratch, d928.Reg)
			d930 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d930)
		} else if d928.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d929.Reg)
			ctx.EmitMovRegReg(scratch, d929.Reg)
			if d928.Imm.Int() >= -2147483648 && d928.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d928.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d928.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d930 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d930)
		} else {
			r206 := ctx.AllocRegExcept(d929.Reg, d928.Reg)
			ctx.EmitMovRegReg(r206, d929.Reg)
			ctx.EmitSubInt64(r206, d928.Reg)
			d930 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r206}
			ctx.BindReg(r206, &d930)
		}
		if d930.Loc == scm.LocReg && d929.Loc == scm.LocReg && d930.Reg == d929.Reg {
			ctx.TransferReg(d929.Reg)
			d929.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d928)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d910)
		ctx.EnsureDesc(&d930)
		var d931 scm.JITValueDesc
		if d910.Loc == scm.LocImm && d930.Loc == scm.LocImm {
			d931 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d910.Imm.Int()) >> uint64(d930.Imm.Int())))}
		} else if d930.Loc == scm.LocImm {
			r207 := ctx.AllocRegExcept(d910.Reg)
			ctx.EmitMovRegReg(r207, d910.Reg)
			ctx.EmitShrRegImm8(r207, uint8(d930.Imm.Int()))
			d931 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r207}
			ctx.BindReg(r207, &d931)
		} else {
			{
				shiftSrc := d910.Reg
				r208 := ctx.AllocRegExcept(d910.Reg)
				ctx.EmitMovRegReg(r208, d910.Reg)
				shiftSrc = r208
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d930.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d930.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d930.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d931 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d931)
			}
		}
		if d931.Loc == scm.LocReg && d910.Loc == scm.LocReg && d931.Reg == d910.Reg {
			ctx.TransferReg(d910.Reg)
			d910.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d910)
		ctx.FreeDesc(&d930)
		ctx.ReclaimUntrackedRegs()
		r209 := ctx.AllocReg()
		ctx.EnsureDesc(&d931)
		ctx.EnsureDesc(&d931)
		if d931.Loc == scm.LocRegPair {
			panic("jit: scalar inline return has scm.LocRegPair")
		} else {
			ctx.EmitMovToReg(r209, d931)
		}
		ctx.EmitJmp(lbl49)
		bbpos_5_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl51)
		ctx.ResolveFixups()
		d910 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d914)
		var d932 scm.JITValueDesc
		if d914.Loc == scm.LocImm {
			d932 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d914.Imm.Int() / 64)}
		} else {
			r210 := ctx.AllocRegExcept(d914.Reg)
			ctx.EmitMovRegReg(r210, d914.Reg)
			ctx.EmitShrRegImm8(r210, 6)
			d932 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r210}
			ctx.BindReg(r210, &d932)
		}
		if d932.Loc == scm.LocReg && d914.Loc == scm.LocReg && d932.Reg == d914.Reg {
			ctx.TransferReg(d914.Reg)
			d914.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d932)
		ctx.EnsureDesc(&d932)
		var d933 scm.JITValueDesc
		if d932.Loc == scm.LocImm {
			d933 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d932.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d932.Reg)
			ctx.EmitMovRegReg(scratch, d932.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d933 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d933)
		}
		if d933.Loc == scm.LocReg && d932.Loc == scm.LocReg && d933.Reg == d932.Reg {
			ctx.TransferReg(d932.Reg)
			d932.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d932)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d933)
		ctx.ReclaimUntrackedRegs()
		d935 = ctx.EmitSliceElementAddress(&d915, &d933, 8)
		ctx.EnsureDesc(&d935)
		ctx.EmitMovRegMem(d935.Reg, d935.Reg, 0)
		d934 = d935
		ctx.FreeDesc(&d933)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d914)
		var d936 scm.JITValueDesc
		if d914.Loc == scm.LocImm {
			d936 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d914.Imm.Int() % 64)}
		} else {
			r211 := ctx.AllocRegExcept(d914.Reg)
			ctx.EmitMovRegReg(r211, d914.Reg)
			ctx.EmitAndRegImm32(r211, 63)
			d936 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r211}
			ctx.BindReg(r211, &d936)
		}
		if d936.Loc == scm.LocReg && d914.Loc == scm.LocReg && d936.Reg == d914.Reg {
			ctx.TransferReg(d914.Reg)
			d914.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d914)
		ctx.ReclaimUntrackedRegs()
		d937 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d936)
		ctx.EnsureDescsTogether(&d937, &d936)
		var d938 scm.JITValueDesc
		if d937.Loc == scm.LocImm && d936.Loc == scm.LocImm {
			d938 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d937.Imm.Int() - d936.Imm.Int())}
		} else if d936.Loc == scm.LocImm && d936.Imm.Int() == 0 {
			r212 := ctx.AllocRegExcept(d937.Reg)
			ctx.EmitMovRegReg(r212, d937.Reg)
			d938 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r212}
			ctx.BindReg(r212, &d938)
		} else if d937.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d936.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d937.Imm.Int()))
			ctx.EmitSubInt64(scratch, d936.Reg)
			d938 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d938)
		} else if d936.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d937.Reg)
			ctx.EmitMovRegReg(scratch, d937.Reg)
			if d936.Imm.Int() >= -2147483648 && d936.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d936.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d936.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d938 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d938)
		} else {
			r213 := ctx.AllocRegExcept(d937.Reg, d936.Reg)
			ctx.EmitMovRegReg(r213, d937.Reg)
			ctx.EmitSubInt64(r213, d936.Reg)
			d938 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r213}
			ctx.BindReg(r213, &d938)
		}
		if d938.Loc == scm.LocReg && d937.Loc == scm.LocReg && d938.Reg == d937.Reg {
			ctx.TransferReg(d937.Reg)
			d937.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d936)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d934)
		ctx.EnsureDesc(&d938)
		var d939 scm.JITValueDesc
		if d934.Loc == scm.LocImm && d938.Loc == scm.LocImm {
			d939 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d934.Imm.Int()) >> uint64(d938.Imm.Int())))}
		} else if d938.Loc == scm.LocImm {
			r214 := ctx.AllocRegExcept(d934.Reg)
			ctx.EmitMovRegReg(r214, d934.Reg)
			ctx.EmitShrRegImm8(r214, uint8(d938.Imm.Int()))
			d939 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r214}
			ctx.BindReg(r214, &d939)
		} else {
			{
				shiftSrc := d934.Reg
				r215 := ctx.AllocRegExcept(d934.Reg)
				ctx.EmitMovRegReg(r215, d934.Reg)
				shiftSrc = r215
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d938.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d938.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d938.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d939 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d939)
			}
		}
		if d939.Loc == scm.LocReg && d934.Loc == scm.LocReg && d939.Reg == d934.Reg {
			ctx.TransferReg(d934.Reg)
			d934.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d934)
		ctx.FreeDesc(&d938)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d920)
		ctx.EnsureDesc(&d939)
		var d940 scm.JITValueDesc
		if d920.Loc == scm.LocImm && d939.Loc == scm.LocImm {
			d940 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d920.Imm.Int() | d939.Imm.Int())}
		} else if d920.Loc == scm.LocImm && d920.Imm.Int() == 0 {
			d940 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d939.Reg}
			ctx.BindReg(d939.Reg, &d940)
		} else if d939.Loc == scm.LocImm && d939.Imm.Int() == 0 {
			r216 := ctx.AllocRegExcept(d920.Reg)
			ctx.EmitMovRegReg(r216, d920.Reg)
			d940 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r216}
			ctx.BindReg(r216, &d940)
		} else if d920.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d939.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d920.Imm.Int()))
			ctx.EmitOrInt64(scratch, d939.Reg)
			d940 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d940)
		} else if d939.Loc == scm.LocImm {
			r217 := ctx.AllocRegExcept(d920.Reg)
			ctx.EmitMovRegReg(r217, d920.Reg)
			if d939.Imm.Int() >= -2147483648 && d939.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r217, int32(d939.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d939.Imm.Int()))
				ctx.EmitOrInt64(r217, scm.RegR11)
			}
			d940 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r217}
			ctx.BindReg(r217, &d940)
		} else {
			r218 := ctx.AllocRegExcept(d920.Reg, d939.Reg)
			ctx.EmitMovRegReg(r218, d920.Reg)
			ctx.EmitOrInt64(r218, d939.Reg)
			d940 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r218}
			ctx.BindReg(r218, &d940)
		}
		if d940.Loc == scm.LocReg && d920.Loc == scm.LocReg && d940.Reg == d920.Reg {
			ctx.TransferReg(d920.Reg)
			d920.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d940)
		ctx.EmitStoreToStack(d940, int32(phiBase909)+int32(0))
		ctx.StabilizeDescForControlFlow(&d940)
		ctx.FreeDesc(&d939)
		ctx.ReclaimUntrackedRegs()
		ctx.EmitJmp(lbl52)
		ctx.MarkLabel(lbl49)
		d941 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r209}
		ctx.BindReg(r209, &d941)
		ctx.BindReg(r209, &d941)
		ctx.FreeDesc(&d4)
		ctx.EnsureDesc(&d941)
		ctx.EnsureDesc(&d941)
		var d942 scm.JITValueDesc
		if d941.Loc == scm.LocImm {
			d942 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d941.Imm.Int()))))}
		} else {
			r219 := ctx.AllocReg()
			ctx.EmitMovRegReg(r219, d941.Reg)
			d942 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r219}
			ctx.BindReg(r219, &d942)
		}
		ctx.FreeDesc(&d941)
		ctx.EnsureDesc(&d942)
		ctx.EnsureDesc(&d57)
		ctx.EnsureDescsTogether(&d942, &d57)
		var d943 scm.JITValueDesc
		if d942.Loc == scm.LocImm && d57.Loc == scm.LocImm {
			d943 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d942.Imm.Int() + d57.Imm.Int())}
		} else if d57.Loc == scm.LocImm && d57.Imm.Int() == 0 {
			r220 := ctx.AllocRegExcept(d942.Reg)
			ctx.EmitMovRegReg(r220, d942.Reg)
			d943 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r220}
			ctx.BindReg(r220, &d943)
		} else if d942.Loc == scm.LocImm && d942.Imm.Int() == 0 {
			d943 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d57.Reg}
			ctx.BindReg(d57.Reg, &d943)
		} else if d942.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d57.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d942.Imm.Int()))
			ctx.EmitAddInt64(scratch, d57.Reg)
			d943 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d943)
		} else if d57.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d942.Reg)
			ctx.EmitMovRegReg(scratch, d942.Reg)
			if d57.Imm.Int() >= -2147483648 && d57.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d57.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d57.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d943 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d943)
		} else {
			r221 := ctx.AllocRegExcept(d942.Reg, d57.Reg)
			ctx.EmitMovRegReg(r221, d942.Reg)
			ctx.EmitAddInt64(r221, d57.Reg)
			d943 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r221}
			ctx.BindReg(r221, &d943)
		}
		if d943.Loc == scm.LocReg && d942.Loc == scm.LocReg && d943.Reg == d942.Reg {
			ctx.TransferReg(d942.Reg)
			d942.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d942)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		var d944 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm {
			d944 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(idxInt.Imm.Int()))))}
		} else {
			r222 := ctx.AllocReg()
			ctx.EmitMovRegReg(r222, idxInt.Reg)
			ctx.EmitShlRegImm8(r222, 32)
			ctx.EmitShrRegImm8(r222, 32)
			d944 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r222}
			ctx.BindReg(r222, &d944)
		}
		ctx.FreeDesc(&idxInt)
		ctx.EnsureDesc(&d944)
		ctx.EnsureDesc(&d943)
		ctx.EnsureDescsTogether(&d944, &d943)
		var d945 scm.JITValueDesc
		if d944.Loc == scm.LocImm && d943.Loc == scm.LocImm {
			d945 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d944.Imm.Int() - d943.Imm.Int())}
		} else if d943.Loc == scm.LocImm && d943.Imm.Int() == 0 {
			r223 := ctx.AllocRegExcept(d944.Reg)
			ctx.EmitMovRegReg(r223, d944.Reg)
			d945 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r223}
			ctx.BindReg(r223, &d945)
		} else if d944.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d943.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d944.Imm.Int()))
			ctx.EmitSubInt64(scratch, d943.Reg)
			d945 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d945)
		} else if d943.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d944.Reg)
			ctx.EmitMovRegReg(scratch, d944.Reg)
			if d943.Imm.Int() >= -2147483648 && d943.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d943.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d943.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d945 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d945)
		} else {
			r224 := ctx.AllocRegExcept(d944.Reg, d943.Reg)
			ctx.EmitMovRegReg(r224, d944.Reg)
			ctx.EmitSubInt64(r224, d943.Reg)
			d945 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r224}
			ctx.BindReg(r224, &d945)
		}
		if d945.Loc == scm.LocReg && d944.Loc == scm.LocReg && d945.Reg == d944.Reg {
			ctx.TransferReg(d944.Reg)
			d944.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d944)
		ctx.FreeDesc(&d943)
		ctx.EnsureDesc(&d945)
		ctx.EnsureDesc(&d907)
		ctx.EnsureDescsTogether(&d945, &d907)
		var d946 scm.JITValueDesc
		if d945.Loc == scm.LocImm && d907.Loc == scm.LocImm {
			d946 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d945.Imm.Int() * d907.Imm.Int())}
		} else if d945.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d907.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d945.Imm.Int()))
			ctx.EmitImulInt64(scratch, d907.Reg)
			d946 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d946)
		} else if d907.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d945.Reg)
			ctx.EmitMovRegReg(scratch, d945.Reg)
			if d907.Imm.Int() >= -2147483648 && d907.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d907.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d907.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d946 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d946)
		} else {
			r225 := ctx.AllocRegExcept(d945.Reg, d907.Reg)
			ctx.EmitMovRegReg(r225, d945.Reg)
			ctx.EmitImulInt64(r225, d907.Reg)
			d946 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r225}
			ctx.BindReg(r225, &d946)
		}
		if d946.Loc == scm.LocReg && d945.Loc == scm.LocReg && d946.Reg == d945.Reg {
			ctx.TransferReg(d945.Reg)
			d945.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d945)
		ctx.FreeDesc(&d907)
		ctx.EnsureDesc(&d170)
		ctx.EnsureDesc(&d946)
		ctx.EnsureDescsTogether(&d170, &d946)
		var d947 scm.JITValueDesc
		if d170.Loc == scm.LocImm && d946.Loc == scm.LocImm {
			d947 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d170.Imm.Int() + d946.Imm.Int())}
		} else if d946.Loc == scm.LocImm && d946.Imm.Int() == 0 {
			r226 := ctx.AllocRegExcept(d170.Reg)
			ctx.EmitMovRegReg(r226, d170.Reg)
			d947 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r226}
			ctx.BindReg(r226, &d947)
		} else if d170.Loc == scm.LocImm && d170.Imm.Int() == 0 {
			d947 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d946.Reg}
			ctx.BindReg(d946.Reg, &d947)
		} else if d170.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d946.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d170.Imm.Int()))
			ctx.EmitAddInt64(scratch, d946.Reg)
			d947 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d947)
		} else if d946.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d170.Reg)
			ctx.EmitMovRegReg(scratch, d170.Reg)
			if d946.Imm.Int() >= -2147483648 && d946.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d946.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d946.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d947 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d947)
		} else {
			r227 := ctx.AllocRegExcept(d170.Reg, d946.Reg)
			ctx.EmitMovRegReg(r227, d170.Reg)
			ctx.EmitAddInt64(r227, d946.Reg)
			d947 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r227}
			ctx.BindReg(r227, &d947)
		}
		if d947.Loc == scm.LocReg && d170.Loc == scm.LocReg && d947.Reg == d170.Reg {
			ctx.TransferReg(d170.Reg)
			d170.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d946)
		ctx.EnsureDesc(&d947)
		ctx.EnsureDesc(&d947)
		var d948 scm.JITValueDesc
		if d947.Loc == scm.LocImm {
			d948 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d947.Imm.Int()))}
		} else {
			ctx.EmitCvtInt64ToFloat64(scm.RegX0, d947.Reg)
			d948 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d947.Reg}
			ctx.BindReg(d947.Reg, &d948)
		}
		ctx.FreeDesc(&d947)
		ctx.EnsureDesc(&d948)
		d949 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d949)
		ctx.BindReg(r1, &d949)
		ctx.EnsureDesc(&d948)
		ctx.EmitMakeFloat(d949, d948)
		if d948.Loc == scm.LocReg {
			ctx.FreeReg(d948.Reg)
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
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
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
		if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != scm.LocNone {
			d282 = ps.OverlayValues[282]
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
		if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
			d294 = ps.OverlayValues[294]
		}
		if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != scm.LocNone {
			d296 = ps.OverlayValues[296]
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
		if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != scm.LocNone {
			d305 = ps.OverlayValues[305]
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
		if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != scm.LocNone {
			d435 = ps.OverlayValues[435]
		}
		if len(ps.OverlayValues) > 436 && ps.OverlayValues[436].Loc != scm.LocNone {
			d436 = ps.OverlayValues[436]
		}
		if len(ps.OverlayValues) > 437 && ps.OverlayValues[437].Loc != scm.LocNone {
			d437 = ps.OverlayValues[437]
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
		if len(ps.OverlayValues) > 648 && ps.OverlayValues[648].Loc != scm.LocNone {
			d648 = ps.OverlayValues[648]
		}
		if len(ps.OverlayValues) > 649 && ps.OverlayValues[649].Loc != scm.LocNone {
			d649 = ps.OverlayValues[649]
		}
		if len(ps.OverlayValues) > 650 && ps.OverlayValues[650].Loc != scm.LocNone {
			d650 = ps.OverlayValues[650]
		}
		if len(ps.OverlayValues) > 652 && ps.OverlayValues[652].Loc != scm.LocNone {
			d652 = ps.OverlayValues[652]
		}
		if len(ps.OverlayValues) > 653 && ps.OverlayValues[653].Loc != scm.LocNone {
			d653 = ps.OverlayValues[653]
		}
		if len(ps.OverlayValues) > 654 && ps.OverlayValues[654].Loc != scm.LocNone {
			d654 = ps.OverlayValues[654]
		}
		if len(ps.OverlayValues) > 655 && ps.OverlayValues[655].Loc != scm.LocNone {
			d655 = ps.OverlayValues[655]
		}
		if len(ps.OverlayValues) > 656 && ps.OverlayValues[656].Loc != scm.LocNone {
			d656 = ps.OverlayValues[656]
		}
		if len(ps.OverlayValues) > 657 && ps.OverlayValues[657].Loc != scm.LocNone {
			d657 = ps.OverlayValues[657]
		}
		if len(ps.OverlayValues) > 658 && ps.OverlayValues[658].Loc != scm.LocNone {
			d658 = ps.OverlayValues[658]
		}
		if len(ps.OverlayValues) > 660 && ps.OverlayValues[660].Loc != scm.LocNone {
			d660 = ps.OverlayValues[660]
		}
		if len(ps.OverlayValues) > 662 && ps.OverlayValues[662].Loc != scm.LocNone {
			d662 = ps.OverlayValues[662]
		}
		if len(ps.OverlayValues) > 663 && ps.OverlayValues[663].Loc != scm.LocNone {
			d663 = ps.OverlayValues[663]
		}
		if len(ps.OverlayValues) > 664 && ps.OverlayValues[664].Loc != scm.LocNone {
			d664 = ps.OverlayValues[664]
		}
		if len(ps.OverlayValues) > 665 && ps.OverlayValues[665].Loc != scm.LocNone {
			d665 = ps.OverlayValues[665]
		}
		if len(ps.OverlayValues) > 668 && ps.OverlayValues[668].Loc != scm.LocNone {
			d668 = ps.OverlayValues[668]
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
		if len(ps.OverlayValues) > 879 && ps.OverlayValues[879].Loc != scm.LocNone {
			d879 = ps.OverlayValues[879]
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
		ctx.ReclaimUntrackedRegs()
		var d950 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 64
			r228 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r228, fieldAddr)
			d950 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r228}
			ctx.BindReg(r228, &d950)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 64)
			r229 := ctx.AllocReg()
			ctx.EmitMovRegMem(r229, thisptr.Reg, off)
			d950 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r229}
			ctx.BindReg(r229, &d950)
		}
		ctx.EnsureDesc(&d950)
		ctx.EnsureDesc(&d950)
		var d951 scm.JITValueDesc
		if d950.Loc == scm.LocImm {
			d951 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d950.Imm.Int()))))}
		} else {
			r230 := ctx.AllocReg()
			ctx.EmitMovRegReg(r230, d950.Reg)
			d951 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r230}
			ctx.BindReg(r230, &d951)
		}
		ctx.EnsureDesc(&d170)
		ctx.EnsureDesc(&d951)
		ctx.EnsureDescsTogether(&d170, &d951)
		var d952 scm.JITValueDesc
		if d170.Loc == scm.LocImm && d951.Loc == scm.LocImm {
			d952 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d170.Imm.Int() == d951.Imm.Int())}
		} else if d951.Loc == scm.LocImm {
			r231 := ctx.AllocRegExcept(d170.Reg)
			if d951.Imm.Int() >= -2147483648 && d951.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d170.Reg, int32(d951.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d951.Imm.Int()))
				ctx.EmitCmpInt64(d170.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r231, scm.CondEqual)
			d952 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r231}
			ctx.BindReg(r231, &d952)
		} else if d170.Loc == scm.LocImm {
			r232 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d170.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d951.Reg)
			ctx.EmitSetcc(r232, scm.CondEqual)
			d952 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r232}
			ctx.BindReg(r232, &d952)
		} else {
			r233 := ctx.AllocRegExcept(d170.Reg)
			ctx.EmitCmpInt64(d170.Reg, d951.Reg)
			ctx.EmitSetcc(r233, scm.CondEqual)
			d952 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r233}
			ctx.BindReg(r233, &d952)
		}
		ctx.FreeDesc(&d170)
		ctx.FreeDesc(&d951)
		d953 = d952
		ctx.EnsureDesc(&d953)
		if d953.Loc != scm.LocImm && d953.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d953.Loc == scm.LocImm {
			if d953.Imm.Bool() {
				if ps.General {
				}
				ps954 := scm.PhiState{General: ps.General}
				ps954.OverlayValues = make([]scm.JITValueDesc, 954)
				ps954.OverlayValues[1] = d1
				ps954.OverlayValues[2] = d2
				ps954.OverlayValues[3] = d3
				ps954.OverlayValues[4] = d4
				ps954.OverlayValues[5] = d5
				ps954.OverlayValues[6] = d6
				ps954.OverlayValues[7] = d7
				ps954.OverlayValues[8] = d8
				ps954.OverlayValues[9] = d9
				ps954.OverlayValues[10] = d10
				ps954.OverlayValues[11] = d11
				ps954.OverlayValues[12] = d12
				ps954.OverlayValues[13] = d13
				ps954.OverlayValues[14] = d14
				ps954.OverlayValues[15] = d15
				ps954.OverlayValues[17] = d17
				ps954.OverlayValues[18] = d18
				ps954.OverlayValues[19] = d19
				ps954.OverlayValues[20] = d20
				ps954.OverlayValues[21] = d21
				ps954.OverlayValues[22] = d22
				ps954.OverlayValues[24] = d24
				ps954.OverlayValues[25] = d25
				ps954.OverlayValues[26] = d26
				ps954.OverlayValues[27] = d27
				ps954.OverlayValues[28] = d28
				ps954.OverlayValues[29] = d29
				ps954.OverlayValues[30] = d30
				ps954.OverlayValues[31] = d31
				ps954.OverlayValues[32] = d32
				ps954.OverlayValues[33] = d33
				ps954.OverlayValues[34] = d34
				ps954.OverlayValues[35] = d35
				ps954.OverlayValues[36] = d36
				ps954.OverlayValues[37] = d37
				ps954.OverlayValues[38] = d38
				ps954.OverlayValues[39] = d39
				ps954.OverlayValues[40] = d40
				ps954.OverlayValues[41] = d41
				ps954.OverlayValues[42] = d42
				ps954.OverlayValues[43] = d43
				ps954.OverlayValues[44] = d44
				ps954.OverlayValues[45] = d45
				ps954.OverlayValues[46] = d46
				ps954.OverlayValues[47] = d47
				ps954.OverlayValues[48] = d48
				ps954.OverlayValues[49] = d49
				ps954.OverlayValues[50] = d50
				ps954.OverlayValues[51] = d51
				ps954.OverlayValues[52] = d52
				ps954.OverlayValues[53] = d53
				ps954.OverlayValues[54] = d54
				ps954.OverlayValues[55] = d55
				ps954.OverlayValues[56] = d56
				ps954.OverlayValues[57] = d57
				ps954.OverlayValues[58] = d58
				ps954.OverlayValues[59] = d59
				ps954.OverlayValues[60] = d60
				ps954.OverlayValues[61] = d61
				ps954.OverlayValues[64] = d64
				ps954.OverlayValues[65] = d65
				ps954.OverlayValues[66] = d66
				ps954.OverlayValues[132] = d132
				ps954.OverlayValues[133] = d133
				ps954.OverlayValues[134] = d134
				ps954.OverlayValues[136] = d136
				ps954.OverlayValues[137] = d137
				ps954.OverlayValues[138] = d138
				ps954.OverlayValues[139] = d139
				ps954.OverlayValues[140] = d140
				ps954.OverlayValues[141] = d141
				ps954.OverlayValues[142] = d142
				ps954.OverlayValues[143] = d143
				ps954.OverlayValues[144] = d144
				ps954.OverlayValues[145] = d145
				ps954.OverlayValues[146] = d146
				ps954.OverlayValues[147] = d147
				ps954.OverlayValues[148] = d148
				ps954.OverlayValues[149] = d149
				ps954.OverlayValues[150] = d150
				ps954.OverlayValues[151] = d151
				ps954.OverlayValues[152] = d152
				ps954.OverlayValues[153] = d153
				ps954.OverlayValues[154] = d154
				ps954.OverlayValues[155] = d155
				ps954.OverlayValues[156] = d156
				ps954.OverlayValues[157] = d157
				ps954.OverlayValues[158] = d158
				ps954.OverlayValues[159] = d159
				ps954.OverlayValues[160] = d160
				ps954.OverlayValues[161] = d161
				ps954.OverlayValues[162] = d162
				ps954.OverlayValues[163] = d163
				ps954.OverlayValues[164] = d164
				ps954.OverlayValues[165] = d165
				ps954.OverlayValues[166] = d166
				ps954.OverlayValues[167] = d167
				ps954.OverlayValues[168] = d168
				ps954.OverlayValues[169] = d169
				ps954.OverlayValues[170] = d170
				ps954.OverlayValues[171] = d171
				ps954.OverlayValues[172] = d172
				ps954.OverlayValues[175] = d175
				ps954.OverlayValues[282] = d282
				ps954.OverlayValues[283] = d283
				ps954.OverlayValues[284] = d284
				ps954.OverlayValues[285] = d285
				ps954.OverlayValues[287] = d287
				ps954.OverlayValues[288] = d288
				ps954.OverlayValues[289] = d289
				ps954.OverlayValues[290] = d290
				ps954.OverlayValues[291] = d291
				ps954.OverlayValues[292] = d292
				ps954.OverlayValues[293] = d293
				ps954.OverlayValues[294] = d294
				ps954.OverlayValues[296] = d296
				ps954.OverlayValues[298] = d298
				ps954.OverlayValues[299] = d299
				ps954.OverlayValues[300] = d300
				ps954.OverlayValues[301] = d301
				ps954.OverlayValues[302] = d302
				ps954.OverlayValues[305] = d305
				ps954.OverlayValues[429] = d429
				ps954.OverlayValues[430] = d430
				ps954.OverlayValues[431] = d431
				ps954.OverlayValues[432] = d432
				ps954.OverlayValues[433] = d433
				ps954.OverlayValues[435] = d435
				ps954.OverlayValues[436] = d436
				ps954.OverlayValues[437] = d437
				ps954.OverlayValues[439] = d439
				ps954.OverlayValues[440] = d440
				ps954.OverlayValues[441] = d441
				ps954.OverlayValues[442] = d442
				ps954.OverlayValues[443] = d443
				ps954.OverlayValues[444] = d444
				ps954.OverlayValues[445] = d445
				ps954.OverlayValues[446] = d446
				ps954.OverlayValues[447] = d447
				ps954.OverlayValues[448] = d448
				ps954.OverlayValues[449] = d449
				ps954.OverlayValues[450] = d450
				ps954.OverlayValues[451] = d451
				ps954.OverlayValues[452] = d452
				ps954.OverlayValues[453] = d453
				ps954.OverlayValues[454] = d454
				ps954.OverlayValues[455] = d455
				ps954.OverlayValues[456] = d456
				ps954.OverlayValues[457] = d457
				ps954.OverlayValues[458] = d458
				ps954.OverlayValues[459] = d459
				ps954.OverlayValues[460] = d460
				ps954.OverlayValues[461] = d461
				ps954.OverlayValues[462] = d462
				ps954.OverlayValues[463] = d463
				ps954.OverlayValues[464] = d464
				ps954.OverlayValues[465] = d465
				ps954.OverlayValues[466] = d466
				ps954.OverlayValues[467] = d467
				ps954.OverlayValues[468] = d468
				ps954.OverlayValues[469] = d469
				ps954.OverlayValues[470] = d470
				ps954.OverlayValues[471] = d471
				ps954.OverlayValues[472] = d472
				ps954.OverlayValues[473] = d473
				ps954.OverlayValues[474] = d474
				ps954.OverlayValues[475] = d475
				ps954.OverlayValues[648] = d648
				ps954.OverlayValues[649] = d649
				ps954.OverlayValues[650] = d650
				ps954.OverlayValues[652] = d652
				ps954.OverlayValues[653] = d653
				ps954.OverlayValues[654] = d654
				ps954.OverlayValues[655] = d655
				ps954.OverlayValues[656] = d656
				ps954.OverlayValues[657] = d657
				ps954.OverlayValues[658] = d658
				ps954.OverlayValues[660] = d660
				ps954.OverlayValues[662] = d662
				ps954.OverlayValues[663] = d663
				ps954.OverlayValues[664] = d664
				ps954.OverlayValues[665] = d665
				ps954.OverlayValues[668] = d668
				ps954.OverlayValues[853] = d853
				ps954.OverlayValues[854] = d854
				ps954.OverlayValues[855] = d855
				ps954.OverlayValues[856] = d856
				ps954.OverlayValues[858] = d858
				ps954.OverlayValues[859] = d859
				ps954.OverlayValues[860] = d860
				ps954.OverlayValues[861] = d861
				ps954.OverlayValues[862] = d862
				ps954.OverlayValues[863] = d863
				ps954.OverlayValues[864] = d864
				ps954.OverlayValues[865] = d865
				ps954.OverlayValues[867] = d867
				ps954.OverlayValues[868] = d868
				ps954.OverlayValues[869] = d869
				ps954.OverlayValues[870] = d870
				ps954.OverlayValues[871] = d871
				ps954.OverlayValues[873] = d873
				ps954.OverlayValues[874] = d874
				ps954.OverlayValues[875] = d875
				ps954.OverlayValues[876] = d876
				ps954.OverlayValues[877] = d877
				ps954.OverlayValues[878] = d878
				ps954.OverlayValues[879] = d879
				ps954.OverlayValues[880] = d880
				ps954.OverlayValues[881] = d881
				ps954.OverlayValues[882] = d882
				ps954.OverlayValues[883] = d883
				ps954.OverlayValues[884] = d884
				ps954.OverlayValues[885] = d885
				ps954.OverlayValues[886] = d886
				ps954.OverlayValues[887] = d887
				ps954.OverlayValues[888] = d888
				ps954.OverlayValues[889] = d889
				ps954.OverlayValues[890] = d890
				ps954.OverlayValues[891] = d891
				ps954.OverlayValues[892] = d892
				ps954.OverlayValues[893] = d893
				ps954.OverlayValues[894] = d894
				ps954.OverlayValues[895] = d895
				ps954.OverlayValues[896] = d896
				ps954.OverlayValues[897] = d897
				ps954.OverlayValues[898] = d898
				ps954.OverlayValues[899] = d899
				ps954.OverlayValues[900] = d900
				ps954.OverlayValues[901] = d901
				ps954.OverlayValues[902] = d902
				ps954.OverlayValues[903] = d903
				ps954.OverlayValues[904] = d904
				ps954.OverlayValues[905] = d905
				ps954.OverlayValues[906] = d906
				ps954.OverlayValues[907] = d907
				ps954.OverlayValues[908] = d908
				ps954.OverlayValues[910] = d910
				ps954.OverlayValues[911] = d911
				ps954.OverlayValues[912] = d912
				ps954.OverlayValues[913] = d913
				ps954.OverlayValues[914] = d914
				ps954.OverlayValues[915] = d915
				ps954.OverlayValues[916] = d916
				ps954.OverlayValues[917] = d917
				ps954.OverlayValues[918] = d918
				ps954.OverlayValues[919] = d919
				ps954.OverlayValues[920] = d920
				ps954.OverlayValues[921] = d921
				ps954.OverlayValues[922] = d922
				ps954.OverlayValues[923] = d923
				ps954.OverlayValues[924] = d924
				ps954.OverlayValues[925] = d925
				ps954.OverlayValues[926] = d926
				ps954.OverlayValues[927] = d927
				ps954.OverlayValues[928] = d928
				ps954.OverlayValues[929] = d929
				ps954.OverlayValues[930] = d930
				ps954.OverlayValues[931] = d931
				ps954.OverlayValues[932] = d932
				ps954.OverlayValues[933] = d933
				ps954.OverlayValues[934] = d934
				ps954.OverlayValues[935] = d935
				ps954.OverlayValues[936] = d936
				ps954.OverlayValues[937] = d937
				ps954.OverlayValues[938] = d938
				ps954.OverlayValues[939] = d939
				ps954.OverlayValues[940] = d940
				ps954.OverlayValues[941] = d941
				ps954.OverlayValues[942] = d942
				ps954.OverlayValues[943] = d943
				ps954.OverlayValues[944] = d944
				ps954.OverlayValues[945] = d945
				ps954.OverlayValues[946] = d946
				ps954.OverlayValues[947] = d947
				ps954.OverlayValues[948] = d948
				ps954.OverlayValues[949] = d949
				ps954.OverlayValues[950] = d950
				ps954.OverlayValues[951] = d951
				ps954.OverlayValues[952] = d952
				ps954.OverlayValues[953] = d953
				return bbs[11].RenderPS(ps954)
			}
			if ps.General {
			}
			ps955 := scm.PhiState{General: ps.General}
			ps955.OverlayValues = make([]scm.JITValueDesc, 954)
			ps955.OverlayValues[1] = d1
			ps955.OverlayValues[2] = d2
			ps955.OverlayValues[3] = d3
			ps955.OverlayValues[4] = d4
			ps955.OverlayValues[5] = d5
			ps955.OverlayValues[6] = d6
			ps955.OverlayValues[7] = d7
			ps955.OverlayValues[8] = d8
			ps955.OverlayValues[9] = d9
			ps955.OverlayValues[10] = d10
			ps955.OverlayValues[11] = d11
			ps955.OverlayValues[12] = d12
			ps955.OverlayValues[13] = d13
			ps955.OverlayValues[14] = d14
			ps955.OverlayValues[15] = d15
			ps955.OverlayValues[17] = d17
			ps955.OverlayValues[18] = d18
			ps955.OverlayValues[19] = d19
			ps955.OverlayValues[20] = d20
			ps955.OverlayValues[21] = d21
			ps955.OverlayValues[22] = d22
			ps955.OverlayValues[24] = d24
			ps955.OverlayValues[25] = d25
			ps955.OverlayValues[26] = d26
			ps955.OverlayValues[27] = d27
			ps955.OverlayValues[28] = d28
			ps955.OverlayValues[29] = d29
			ps955.OverlayValues[30] = d30
			ps955.OverlayValues[31] = d31
			ps955.OverlayValues[32] = d32
			ps955.OverlayValues[33] = d33
			ps955.OverlayValues[34] = d34
			ps955.OverlayValues[35] = d35
			ps955.OverlayValues[36] = d36
			ps955.OverlayValues[37] = d37
			ps955.OverlayValues[38] = d38
			ps955.OverlayValues[39] = d39
			ps955.OverlayValues[40] = d40
			ps955.OverlayValues[41] = d41
			ps955.OverlayValues[42] = d42
			ps955.OverlayValues[43] = d43
			ps955.OverlayValues[44] = d44
			ps955.OverlayValues[45] = d45
			ps955.OverlayValues[46] = d46
			ps955.OverlayValues[47] = d47
			ps955.OverlayValues[48] = d48
			ps955.OverlayValues[49] = d49
			ps955.OverlayValues[50] = d50
			ps955.OverlayValues[51] = d51
			ps955.OverlayValues[52] = d52
			ps955.OverlayValues[53] = d53
			ps955.OverlayValues[54] = d54
			ps955.OverlayValues[55] = d55
			ps955.OverlayValues[56] = d56
			ps955.OverlayValues[57] = d57
			ps955.OverlayValues[58] = d58
			ps955.OverlayValues[59] = d59
			ps955.OverlayValues[60] = d60
			ps955.OverlayValues[61] = d61
			ps955.OverlayValues[64] = d64
			ps955.OverlayValues[65] = d65
			ps955.OverlayValues[66] = d66
			ps955.OverlayValues[132] = d132
			ps955.OverlayValues[133] = d133
			ps955.OverlayValues[134] = d134
			ps955.OverlayValues[136] = d136
			ps955.OverlayValues[137] = d137
			ps955.OverlayValues[138] = d138
			ps955.OverlayValues[139] = d139
			ps955.OverlayValues[140] = d140
			ps955.OverlayValues[141] = d141
			ps955.OverlayValues[142] = d142
			ps955.OverlayValues[143] = d143
			ps955.OverlayValues[144] = d144
			ps955.OverlayValues[145] = d145
			ps955.OverlayValues[146] = d146
			ps955.OverlayValues[147] = d147
			ps955.OverlayValues[148] = d148
			ps955.OverlayValues[149] = d149
			ps955.OverlayValues[150] = d150
			ps955.OverlayValues[151] = d151
			ps955.OverlayValues[152] = d152
			ps955.OverlayValues[153] = d153
			ps955.OverlayValues[154] = d154
			ps955.OverlayValues[155] = d155
			ps955.OverlayValues[156] = d156
			ps955.OverlayValues[157] = d157
			ps955.OverlayValues[158] = d158
			ps955.OverlayValues[159] = d159
			ps955.OverlayValues[160] = d160
			ps955.OverlayValues[161] = d161
			ps955.OverlayValues[162] = d162
			ps955.OverlayValues[163] = d163
			ps955.OverlayValues[164] = d164
			ps955.OverlayValues[165] = d165
			ps955.OverlayValues[166] = d166
			ps955.OverlayValues[167] = d167
			ps955.OverlayValues[168] = d168
			ps955.OverlayValues[169] = d169
			ps955.OverlayValues[170] = d170
			ps955.OverlayValues[171] = d171
			ps955.OverlayValues[172] = d172
			ps955.OverlayValues[175] = d175
			ps955.OverlayValues[282] = d282
			ps955.OverlayValues[283] = d283
			ps955.OverlayValues[284] = d284
			ps955.OverlayValues[285] = d285
			ps955.OverlayValues[287] = d287
			ps955.OverlayValues[288] = d288
			ps955.OverlayValues[289] = d289
			ps955.OverlayValues[290] = d290
			ps955.OverlayValues[291] = d291
			ps955.OverlayValues[292] = d292
			ps955.OverlayValues[293] = d293
			ps955.OverlayValues[294] = d294
			ps955.OverlayValues[296] = d296
			ps955.OverlayValues[298] = d298
			ps955.OverlayValues[299] = d299
			ps955.OverlayValues[300] = d300
			ps955.OverlayValues[301] = d301
			ps955.OverlayValues[302] = d302
			ps955.OverlayValues[305] = d305
			ps955.OverlayValues[429] = d429
			ps955.OverlayValues[430] = d430
			ps955.OverlayValues[431] = d431
			ps955.OverlayValues[432] = d432
			ps955.OverlayValues[433] = d433
			ps955.OverlayValues[435] = d435
			ps955.OverlayValues[436] = d436
			ps955.OverlayValues[437] = d437
			ps955.OverlayValues[439] = d439
			ps955.OverlayValues[440] = d440
			ps955.OverlayValues[441] = d441
			ps955.OverlayValues[442] = d442
			ps955.OverlayValues[443] = d443
			ps955.OverlayValues[444] = d444
			ps955.OverlayValues[445] = d445
			ps955.OverlayValues[446] = d446
			ps955.OverlayValues[447] = d447
			ps955.OverlayValues[448] = d448
			ps955.OverlayValues[449] = d449
			ps955.OverlayValues[450] = d450
			ps955.OverlayValues[451] = d451
			ps955.OverlayValues[452] = d452
			ps955.OverlayValues[453] = d453
			ps955.OverlayValues[454] = d454
			ps955.OverlayValues[455] = d455
			ps955.OverlayValues[456] = d456
			ps955.OverlayValues[457] = d457
			ps955.OverlayValues[458] = d458
			ps955.OverlayValues[459] = d459
			ps955.OverlayValues[460] = d460
			ps955.OverlayValues[461] = d461
			ps955.OverlayValues[462] = d462
			ps955.OverlayValues[463] = d463
			ps955.OverlayValues[464] = d464
			ps955.OverlayValues[465] = d465
			ps955.OverlayValues[466] = d466
			ps955.OverlayValues[467] = d467
			ps955.OverlayValues[468] = d468
			ps955.OverlayValues[469] = d469
			ps955.OverlayValues[470] = d470
			ps955.OverlayValues[471] = d471
			ps955.OverlayValues[472] = d472
			ps955.OverlayValues[473] = d473
			ps955.OverlayValues[474] = d474
			ps955.OverlayValues[475] = d475
			ps955.OverlayValues[648] = d648
			ps955.OverlayValues[649] = d649
			ps955.OverlayValues[650] = d650
			ps955.OverlayValues[652] = d652
			ps955.OverlayValues[653] = d653
			ps955.OverlayValues[654] = d654
			ps955.OverlayValues[655] = d655
			ps955.OverlayValues[656] = d656
			ps955.OverlayValues[657] = d657
			ps955.OverlayValues[658] = d658
			ps955.OverlayValues[660] = d660
			ps955.OverlayValues[662] = d662
			ps955.OverlayValues[663] = d663
			ps955.OverlayValues[664] = d664
			ps955.OverlayValues[665] = d665
			ps955.OverlayValues[668] = d668
			ps955.OverlayValues[853] = d853
			ps955.OverlayValues[854] = d854
			ps955.OverlayValues[855] = d855
			ps955.OverlayValues[856] = d856
			ps955.OverlayValues[858] = d858
			ps955.OverlayValues[859] = d859
			ps955.OverlayValues[860] = d860
			ps955.OverlayValues[861] = d861
			ps955.OverlayValues[862] = d862
			ps955.OverlayValues[863] = d863
			ps955.OverlayValues[864] = d864
			ps955.OverlayValues[865] = d865
			ps955.OverlayValues[867] = d867
			ps955.OverlayValues[868] = d868
			ps955.OverlayValues[869] = d869
			ps955.OverlayValues[870] = d870
			ps955.OverlayValues[871] = d871
			ps955.OverlayValues[873] = d873
			ps955.OverlayValues[874] = d874
			ps955.OverlayValues[875] = d875
			ps955.OverlayValues[876] = d876
			ps955.OverlayValues[877] = d877
			ps955.OverlayValues[878] = d878
			ps955.OverlayValues[879] = d879
			ps955.OverlayValues[880] = d880
			ps955.OverlayValues[881] = d881
			ps955.OverlayValues[882] = d882
			ps955.OverlayValues[883] = d883
			ps955.OverlayValues[884] = d884
			ps955.OverlayValues[885] = d885
			ps955.OverlayValues[886] = d886
			ps955.OverlayValues[887] = d887
			ps955.OverlayValues[888] = d888
			ps955.OverlayValues[889] = d889
			ps955.OverlayValues[890] = d890
			ps955.OverlayValues[891] = d891
			ps955.OverlayValues[892] = d892
			ps955.OverlayValues[893] = d893
			ps955.OverlayValues[894] = d894
			ps955.OverlayValues[895] = d895
			ps955.OverlayValues[896] = d896
			ps955.OverlayValues[897] = d897
			ps955.OverlayValues[898] = d898
			ps955.OverlayValues[899] = d899
			ps955.OverlayValues[900] = d900
			ps955.OverlayValues[901] = d901
			ps955.OverlayValues[902] = d902
			ps955.OverlayValues[903] = d903
			ps955.OverlayValues[904] = d904
			ps955.OverlayValues[905] = d905
			ps955.OverlayValues[906] = d906
			ps955.OverlayValues[907] = d907
			ps955.OverlayValues[908] = d908
			ps955.OverlayValues[910] = d910
			ps955.OverlayValues[911] = d911
			ps955.OverlayValues[912] = d912
			ps955.OverlayValues[913] = d913
			ps955.OverlayValues[914] = d914
			ps955.OverlayValues[915] = d915
			ps955.OverlayValues[916] = d916
			ps955.OverlayValues[917] = d917
			ps955.OverlayValues[918] = d918
			ps955.OverlayValues[919] = d919
			ps955.OverlayValues[920] = d920
			ps955.OverlayValues[921] = d921
			ps955.OverlayValues[922] = d922
			ps955.OverlayValues[923] = d923
			ps955.OverlayValues[924] = d924
			ps955.OverlayValues[925] = d925
			ps955.OverlayValues[926] = d926
			ps955.OverlayValues[927] = d927
			ps955.OverlayValues[928] = d928
			ps955.OverlayValues[929] = d929
			ps955.OverlayValues[930] = d930
			ps955.OverlayValues[931] = d931
			ps955.OverlayValues[932] = d932
			ps955.OverlayValues[933] = d933
			ps955.OverlayValues[934] = d934
			ps955.OverlayValues[935] = d935
			ps955.OverlayValues[936] = d936
			ps955.OverlayValues[937] = d937
			ps955.OverlayValues[938] = d938
			ps955.OverlayValues[939] = d939
			ps955.OverlayValues[940] = d940
			ps955.OverlayValues[941] = d941
			ps955.OverlayValues[942] = d942
			ps955.OverlayValues[943] = d943
			ps955.OverlayValues[944] = d944
			ps955.OverlayValues[945] = d945
			ps955.OverlayValues[946] = d946
			ps955.OverlayValues[947] = d947
			ps955.OverlayValues[948] = d948
			ps955.OverlayValues[949] = d949
			ps955.OverlayValues[950] = d950
			ps955.OverlayValues[951] = d951
			ps955.OverlayValues[952] = d952
			ps955.OverlayValues[953] = d953
			return bbs[12].RenderPS(ps955)
		}
		if !ps.General {
			ps.General = true
			return bbs[13].RenderPS(ps)
		}
		lbl55 := ctx.ReserveLabel()
		lbl56 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d953.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl55)
		ctx.EmitJmp(lbl56)
		ctx.MarkLabel(lbl55)
		ctx.EmitJmp(lbl12)
		ctx.MarkLabel(lbl56)
		ctx.EmitJmp(lbl13)
		ps956 := scm.PhiState{General: true}
		ps956.OverlayValues = make([]scm.JITValueDesc, 954)
		ps956.OverlayValues[1] = d1
		ps956.OverlayValues[2] = d2
		ps956.OverlayValues[3] = d3
		ps956.OverlayValues[4] = d4
		ps956.OverlayValues[5] = d5
		ps956.OverlayValues[6] = d6
		ps956.OverlayValues[7] = d7
		ps956.OverlayValues[8] = d8
		ps956.OverlayValues[9] = d9
		ps956.OverlayValues[10] = d10
		ps956.OverlayValues[11] = d11
		ps956.OverlayValues[12] = d12
		ps956.OverlayValues[13] = d13
		ps956.OverlayValues[14] = d14
		ps956.OverlayValues[15] = d15
		ps956.OverlayValues[17] = d17
		ps956.OverlayValues[18] = d18
		ps956.OverlayValues[19] = d19
		ps956.OverlayValues[20] = d20
		ps956.OverlayValues[21] = d21
		ps956.OverlayValues[22] = d22
		ps956.OverlayValues[24] = d24
		ps956.OverlayValues[25] = d25
		ps956.OverlayValues[26] = d26
		ps956.OverlayValues[27] = d27
		ps956.OverlayValues[28] = d28
		ps956.OverlayValues[29] = d29
		ps956.OverlayValues[30] = d30
		ps956.OverlayValues[31] = d31
		ps956.OverlayValues[32] = d32
		ps956.OverlayValues[33] = d33
		ps956.OverlayValues[34] = d34
		ps956.OverlayValues[35] = d35
		ps956.OverlayValues[36] = d36
		ps956.OverlayValues[37] = d37
		ps956.OverlayValues[38] = d38
		ps956.OverlayValues[39] = d39
		ps956.OverlayValues[40] = d40
		ps956.OverlayValues[41] = d41
		ps956.OverlayValues[42] = d42
		ps956.OverlayValues[43] = d43
		ps956.OverlayValues[44] = d44
		ps956.OverlayValues[45] = d45
		ps956.OverlayValues[46] = d46
		ps956.OverlayValues[47] = d47
		ps956.OverlayValues[48] = d48
		ps956.OverlayValues[49] = d49
		ps956.OverlayValues[50] = d50
		ps956.OverlayValues[51] = d51
		ps956.OverlayValues[52] = d52
		ps956.OverlayValues[53] = d53
		ps956.OverlayValues[54] = d54
		ps956.OverlayValues[55] = d55
		ps956.OverlayValues[56] = d56
		ps956.OverlayValues[57] = d57
		ps956.OverlayValues[58] = d58
		ps956.OverlayValues[59] = d59
		ps956.OverlayValues[60] = d60
		ps956.OverlayValues[61] = d61
		ps956.OverlayValues[64] = d64
		ps956.OverlayValues[65] = d65
		ps956.OverlayValues[66] = d66
		ps956.OverlayValues[132] = d132
		ps956.OverlayValues[133] = d133
		ps956.OverlayValues[134] = d134
		ps956.OverlayValues[136] = d136
		ps956.OverlayValues[137] = d137
		ps956.OverlayValues[138] = d138
		ps956.OverlayValues[139] = d139
		ps956.OverlayValues[140] = d140
		ps956.OverlayValues[141] = d141
		ps956.OverlayValues[142] = d142
		ps956.OverlayValues[143] = d143
		ps956.OverlayValues[144] = d144
		ps956.OverlayValues[145] = d145
		ps956.OverlayValues[146] = d146
		ps956.OverlayValues[147] = d147
		ps956.OverlayValues[148] = d148
		ps956.OverlayValues[149] = d149
		ps956.OverlayValues[150] = d150
		ps956.OverlayValues[151] = d151
		ps956.OverlayValues[152] = d152
		ps956.OverlayValues[153] = d153
		ps956.OverlayValues[154] = d154
		ps956.OverlayValues[155] = d155
		ps956.OverlayValues[156] = d156
		ps956.OverlayValues[157] = d157
		ps956.OverlayValues[158] = d158
		ps956.OverlayValues[159] = d159
		ps956.OverlayValues[160] = d160
		ps956.OverlayValues[161] = d161
		ps956.OverlayValues[162] = d162
		ps956.OverlayValues[163] = d163
		ps956.OverlayValues[164] = d164
		ps956.OverlayValues[165] = d165
		ps956.OverlayValues[166] = d166
		ps956.OverlayValues[167] = d167
		ps956.OverlayValues[168] = d168
		ps956.OverlayValues[169] = d169
		ps956.OverlayValues[170] = d170
		ps956.OverlayValues[171] = d171
		ps956.OverlayValues[172] = d172
		ps956.OverlayValues[175] = d175
		ps956.OverlayValues[282] = d282
		ps956.OverlayValues[283] = d283
		ps956.OverlayValues[284] = d284
		ps956.OverlayValues[285] = d285
		ps956.OverlayValues[287] = d287
		ps956.OverlayValues[288] = d288
		ps956.OverlayValues[289] = d289
		ps956.OverlayValues[290] = d290
		ps956.OverlayValues[291] = d291
		ps956.OverlayValues[292] = d292
		ps956.OverlayValues[293] = d293
		ps956.OverlayValues[294] = d294
		ps956.OverlayValues[296] = d296
		ps956.OverlayValues[298] = d298
		ps956.OverlayValues[299] = d299
		ps956.OverlayValues[300] = d300
		ps956.OverlayValues[301] = d301
		ps956.OverlayValues[302] = d302
		ps956.OverlayValues[305] = d305
		ps956.OverlayValues[429] = d429
		ps956.OverlayValues[430] = d430
		ps956.OverlayValues[431] = d431
		ps956.OverlayValues[432] = d432
		ps956.OverlayValues[433] = d433
		ps956.OverlayValues[435] = d435
		ps956.OverlayValues[436] = d436
		ps956.OverlayValues[437] = d437
		ps956.OverlayValues[439] = d439
		ps956.OverlayValues[440] = d440
		ps956.OverlayValues[441] = d441
		ps956.OverlayValues[442] = d442
		ps956.OverlayValues[443] = d443
		ps956.OverlayValues[444] = d444
		ps956.OverlayValues[445] = d445
		ps956.OverlayValues[446] = d446
		ps956.OverlayValues[447] = d447
		ps956.OverlayValues[448] = d448
		ps956.OverlayValues[449] = d449
		ps956.OverlayValues[450] = d450
		ps956.OverlayValues[451] = d451
		ps956.OverlayValues[452] = d452
		ps956.OverlayValues[453] = d453
		ps956.OverlayValues[454] = d454
		ps956.OverlayValues[455] = d455
		ps956.OverlayValues[456] = d456
		ps956.OverlayValues[457] = d457
		ps956.OverlayValues[458] = d458
		ps956.OverlayValues[459] = d459
		ps956.OverlayValues[460] = d460
		ps956.OverlayValues[461] = d461
		ps956.OverlayValues[462] = d462
		ps956.OverlayValues[463] = d463
		ps956.OverlayValues[464] = d464
		ps956.OverlayValues[465] = d465
		ps956.OverlayValues[466] = d466
		ps956.OverlayValues[467] = d467
		ps956.OverlayValues[468] = d468
		ps956.OverlayValues[469] = d469
		ps956.OverlayValues[470] = d470
		ps956.OverlayValues[471] = d471
		ps956.OverlayValues[472] = d472
		ps956.OverlayValues[473] = d473
		ps956.OverlayValues[474] = d474
		ps956.OverlayValues[475] = d475
		ps956.OverlayValues[648] = d648
		ps956.OverlayValues[649] = d649
		ps956.OverlayValues[650] = d650
		ps956.OverlayValues[652] = d652
		ps956.OverlayValues[653] = d653
		ps956.OverlayValues[654] = d654
		ps956.OverlayValues[655] = d655
		ps956.OverlayValues[656] = d656
		ps956.OverlayValues[657] = d657
		ps956.OverlayValues[658] = d658
		ps956.OverlayValues[660] = d660
		ps956.OverlayValues[662] = d662
		ps956.OverlayValues[663] = d663
		ps956.OverlayValues[664] = d664
		ps956.OverlayValues[665] = d665
		ps956.OverlayValues[668] = d668
		ps956.OverlayValues[853] = d853
		ps956.OverlayValues[854] = d854
		ps956.OverlayValues[855] = d855
		ps956.OverlayValues[856] = d856
		ps956.OverlayValues[858] = d858
		ps956.OverlayValues[859] = d859
		ps956.OverlayValues[860] = d860
		ps956.OverlayValues[861] = d861
		ps956.OverlayValues[862] = d862
		ps956.OverlayValues[863] = d863
		ps956.OverlayValues[864] = d864
		ps956.OverlayValues[865] = d865
		ps956.OverlayValues[867] = d867
		ps956.OverlayValues[868] = d868
		ps956.OverlayValues[869] = d869
		ps956.OverlayValues[870] = d870
		ps956.OverlayValues[871] = d871
		ps956.OverlayValues[873] = d873
		ps956.OverlayValues[874] = d874
		ps956.OverlayValues[875] = d875
		ps956.OverlayValues[876] = d876
		ps956.OverlayValues[877] = d877
		ps956.OverlayValues[878] = d878
		ps956.OverlayValues[879] = d879
		ps956.OverlayValues[880] = d880
		ps956.OverlayValues[881] = d881
		ps956.OverlayValues[882] = d882
		ps956.OverlayValues[883] = d883
		ps956.OverlayValues[884] = d884
		ps956.OverlayValues[885] = d885
		ps956.OverlayValues[886] = d886
		ps956.OverlayValues[887] = d887
		ps956.OverlayValues[888] = d888
		ps956.OverlayValues[889] = d889
		ps956.OverlayValues[890] = d890
		ps956.OverlayValues[891] = d891
		ps956.OverlayValues[892] = d892
		ps956.OverlayValues[893] = d893
		ps956.OverlayValues[894] = d894
		ps956.OverlayValues[895] = d895
		ps956.OverlayValues[896] = d896
		ps956.OverlayValues[897] = d897
		ps956.OverlayValues[898] = d898
		ps956.OverlayValues[899] = d899
		ps956.OverlayValues[900] = d900
		ps956.OverlayValues[901] = d901
		ps956.OverlayValues[902] = d902
		ps956.OverlayValues[903] = d903
		ps956.OverlayValues[904] = d904
		ps956.OverlayValues[905] = d905
		ps956.OverlayValues[906] = d906
		ps956.OverlayValues[907] = d907
		ps956.OverlayValues[908] = d908
		ps956.OverlayValues[910] = d910
		ps956.OverlayValues[911] = d911
		ps956.OverlayValues[912] = d912
		ps956.OverlayValues[913] = d913
		ps956.OverlayValues[914] = d914
		ps956.OverlayValues[915] = d915
		ps956.OverlayValues[916] = d916
		ps956.OverlayValues[917] = d917
		ps956.OverlayValues[918] = d918
		ps956.OverlayValues[919] = d919
		ps956.OverlayValues[920] = d920
		ps956.OverlayValues[921] = d921
		ps956.OverlayValues[922] = d922
		ps956.OverlayValues[923] = d923
		ps956.OverlayValues[924] = d924
		ps956.OverlayValues[925] = d925
		ps956.OverlayValues[926] = d926
		ps956.OverlayValues[927] = d927
		ps956.OverlayValues[928] = d928
		ps956.OverlayValues[929] = d929
		ps956.OverlayValues[930] = d930
		ps956.OverlayValues[931] = d931
		ps956.OverlayValues[932] = d932
		ps956.OverlayValues[933] = d933
		ps956.OverlayValues[934] = d934
		ps956.OverlayValues[935] = d935
		ps956.OverlayValues[936] = d936
		ps956.OverlayValues[937] = d937
		ps956.OverlayValues[938] = d938
		ps956.OverlayValues[939] = d939
		ps956.OverlayValues[940] = d940
		ps956.OverlayValues[941] = d941
		ps956.OverlayValues[942] = d942
		ps956.OverlayValues[943] = d943
		ps956.OverlayValues[944] = d944
		ps956.OverlayValues[945] = d945
		ps956.OverlayValues[946] = d946
		ps956.OverlayValues[947] = d947
		ps956.OverlayValues[948] = d948
		ps956.OverlayValues[949] = d949
		ps956.OverlayValues[950] = d950
		ps956.OverlayValues[951] = d951
		ps956.OverlayValues[952] = d952
		ps956.OverlayValues[953] = d953
		ps957 := scm.PhiState{General: true}
		ps957.OverlayValues = make([]scm.JITValueDesc, 954)
		ps957.OverlayValues[1] = d1
		ps957.OverlayValues[2] = d2
		ps957.OverlayValues[3] = d3
		ps957.OverlayValues[4] = d4
		ps957.OverlayValues[5] = d5
		ps957.OverlayValues[6] = d6
		ps957.OverlayValues[7] = d7
		ps957.OverlayValues[8] = d8
		ps957.OverlayValues[9] = d9
		ps957.OverlayValues[10] = d10
		ps957.OverlayValues[11] = d11
		ps957.OverlayValues[12] = d12
		ps957.OverlayValues[13] = d13
		ps957.OverlayValues[14] = d14
		ps957.OverlayValues[15] = d15
		ps957.OverlayValues[17] = d17
		ps957.OverlayValues[18] = d18
		ps957.OverlayValues[19] = d19
		ps957.OverlayValues[20] = d20
		ps957.OverlayValues[21] = d21
		ps957.OverlayValues[22] = d22
		ps957.OverlayValues[24] = d24
		ps957.OverlayValues[25] = d25
		ps957.OverlayValues[26] = d26
		ps957.OverlayValues[27] = d27
		ps957.OverlayValues[28] = d28
		ps957.OverlayValues[29] = d29
		ps957.OverlayValues[30] = d30
		ps957.OverlayValues[31] = d31
		ps957.OverlayValues[32] = d32
		ps957.OverlayValues[33] = d33
		ps957.OverlayValues[34] = d34
		ps957.OverlayValues[35] = d35
		ps957.OverlayValues[36] = d36
		ps957.OverlayValues[37] = d37
		ps957.OverlayValues[38] = d38
		ps957.OverlayValues[39] = d39
		ps957.OverlayValues[40] = d40
		ps957.OverlayValues[41] = d41
		ps957.OverlayValues[42] = d42
		ps957.OverlayValues[43] = d43
		ps957.OverlayValues[44] = d44
		ps957.OverlayValues[45] = d45
		ps957.OverlayValues[46] = d46
		ps957.OverlayValues[47] = d47
		ps957.OverlayValues[48] = d48
		ps957.OverlayValues[49] = d49
		ps957.OverlayValues[50] = d50
		ps957.OverlayValues[51] = d51
		ps957.OverlayValues[52] = d52
		ps957.OverlayValues[53] = d53
		ps957.OverlayValues[54] = d54
		ps957.OverlayValues[55] = d55
		ps957.OverlayValues[56] = d56
		ps957.OverlayValues[57] = d57
		ps957.OverlayValues[58] = d58
		ps957.OverlayValues[59] = d59
		ps957.OverlayValues[60] = d60
		ps957.OverlayValues[61] = d61
		ps957.OverlayValues[64] = d64
		ps957.OverlayValues[65] = d65
		ps957.OverlayValues[66] = d66
		ps957.OverlayValues[132] = d132
		ps957.OverlayValues[133] = d133
		ps957.OverlayValues[134] = d134
		ps957.OverlayValues[136] = d136
		ps957.OverlayValues[137] = d137
		ps957.OverlayValues[138] = d138
		ps957.OverlayValues[139] = d139
		ps957.OverlayValues[140] = d140
		ps957.OverlayValues[141] = d141
		ps957.OverlayValues[142] = d142
		ps957.OverlayValues[143] = d143
		ps957.OverlayValues[144] = d144
		ps957.OverlayValues[145] = d145
		ps957.OverlayValues[146] = d146
		ps957.OverlayValues[147] = d147
		ps957.OverlayValues[148] = d148
		ps957.OverlayValues[149] = d149
		ps957.OverlayValues[150] = d150
		ps957.OverlayValues[151] = d151
		ps957.OverlayValues[152] = d152
		ps957.OverlayValues[153] = d153
		ps957.OverlayValues[154] = d154
		ps957.OverlayValues[155] = d155
		ps957.OverlayValues[156] = d156
		ps957.OverlayValues[157] = d157
		ps957.OverlayValues[158] = d158
		ps957.OverlayValues[159] = d159
		ps957.OverlayValues[160] = d160
		ps957.OverlayValues[161] = d161
		ps957.OverlayValues[162] = d162
		ps957.OverlayValues[163] = d163
		ps957.OverlayValues[164] = d164
		ps957.OverlayValues[165] = d165
		ps957.OverlayValues[166] = d166
		ps957.OverlayValues[167] = d167
		ps957.OverlayValues[168] = d168
		ps957.OverlayValues[169] = d169
		ps957.OverlayValues[170] = d170
		ps957.OverlayValues[171] = d171
		ps957.OverlayValues[172] = d172
		ps957.OverlayValues[175] = d175
		ps957.OverlayValues[282] = d282
		ps957.OverlayValues[283] = d283
		ps957.OverlayValues[284] = d284
		ps957.OverlayValues[285] = d285
		ps957.OverlayValues[287] = d287
		ps957.OverlayValues[288] = d288
		ps957.OverlayValues[289] = d289
		ps957.OverlayValues[290] = d290
		ps957.OverlayValues[291] = d291
		ps957.OverlayValues[292] = d292
		ps957.OverlayValues[293] = d293
		ps957.OverlayValues[294] = d294
		ps957.OverlayValues[296] = d296
		ps957.OverlayValues[298] = d298
		ps957.OverlayValues[299] = d299
		ps957.OverlayValues[300] = d300
		ps957.OverlayValues[301] = d301
		ps957.OverlayValues[302] = d302
		ps957.OverlayValues[305] = d305
		ps957.OverlayValues[429] = d429
		ps957.OverlayValues[430] = d430
		ps957.OverlayValues[431] = d431
		ps957.OverlayValues[432] = d432
		ps957.OverlayValues[433] = d433
		ps957.OverlayValues[435] = d435
		ps957.OverlayValues[436] = d436
		ps957.OverlayValues[437] = d437
		ps957.OverlayValues[439] = d439
		ps957.OverlayValues[440] = d440
		ps957.OverlayValues[441] = d441
		ps957.OverlayValues[442] = d442
		ps957.OverlayValues[443] = d443
		ps957.OverlayValues[444] = d444
		ps957.OverlayValues[445] = d445
		ps957.OverlayValues[446] = d446
		ps957.OverlayValues[447] = d447
		ps957.OverlayValues[448] = d448
		ps957.OverlayValues[449] = d449
		ps957.OverlayValues[450] = d450
		ps957.OverlayValues[451] = d451
		ps957.OverlayValues[452] = d452
		ps957.OverlayValues[453] = d453
		ps957.OverlayValues[454] = d454
		ps957.OverlayValues[455] = d455
		ps957.OverlayValues[456] = d456
		ps957.OverlayValues[457] = d457
		ps957.OverlayValues[458] = d458
		ps957.OverlayValues[459] = d459
		ps957.OverlayValues[460] = d460
		ps957.OverlayValues[461] = d461
		ps957.OverlayValues[462] = d462
		ps957.OverlayValues[463] = d463
		ps957.OverlayValues[464] = d464
		ps957.OverlayValues[465] = d465
		ps957.OverlayValues[466] = d466
		ps957.OverlayValues[467] = d467
		ps957.OverlayValues[468] = d468
		ps957.OverlayValues[469] = d469
		ps957.OverlayValues[470] = d470
		ps957.OverlayValues[471] = d471
		ps957.OverlayValues[472] = d472
		ps957.OverlayValues[473] = d473
		ps957.OverlayValues[474] = d474
		ps957.OverlayValues[475] = d475
		ps957.OverlayValues[648] = d648
		ps957.OverlayValues[649] = d649
		ps957.OverlayValues[650] = d650
		ps957.OverlayValues[652] = d652
		ps957.OverlayValues[653] = d653
		ps957.OverlayValues[654] = d654
		ps957.OverlayValues[655] = d655
		ps957.OverlayValues[656] = d656
		ps957.OverlayValues[657] = d657
		ps957.OverlayValues[658] = d658
		ps957.OverlayValues[660] = d660
		ps957.OverlayValues[662] = d662
		ps957.OverlayValues[663] = d663
		ps957.OverlayValues[664] = d664
		ps957.OverlayValues[665] = d665
		ps957.OverlayValues[668] = d668
		ps957.OverlayValues[853] = d853
		ps957.OverlayValues[854] = d854
		ps957.OverlayValues[855] = d855
		ps957.OverlayValues[856] = d856
		ps957.OverlayValues[858] = d858
		ps957.OverlayValues[859] = d859
		ps957.OverlayValues[860] = d860
		ps957.OverlayValues[861] = d861
		ps957.OverlayValues[862] = d862
		ps957.OverlayValues[863] = d863
		ps957.OverlayValues[864] = d864
		ps957.OverlayValues[865] = d865
		ps957.OverlayValues[867] = d867
		ps957.OverlayValues[868] = d868
		ps957.OverlayValues[869] = d869
		ps957.OverlayValues[870] = d870
		ps957.OverlayValues[871] = d871
		ps957.OverlayValues[873] = d873
		ps957.OverlayValues[874] = d874
		ps957.OverlayValues[875] = d875
		ps957.OverlayValues[876] = d876
		ps957.OverlayValues[877] = d877
		ps957.OverlayValues[878] = d878
		ps957.OverlayValues[879] = d879
		ps957.OverlayValues[880] = d880
		ps957.OverlayValues[881] = d881
		ps957.OverlayValues[882] = d882
		ps957.OverlayValues[883] = d883
		ps957.OverlayValues[884] = d884
		ps957.OverlayValues[885] = d885
		ps957.OverlayValues[886] = d886
		ps957.OverlayValues[887] = d887
		ps957.OverlayValues[888] = d888
		ps957.OverlayValues[889] = d889
		ps957.OverlayValues[890] = d890
		ps957.OverlayValues[891] = d891
		ps957.OverlayValues[892] = d892
		ps957.OverlayValues[893] = d893
		ps957.OverlayValues[894] = d894
		ps957.OverlayValues[895] = d895
		ps957.OverlayValues[896] = d896
		ps957.OverlayValues[897] = d897
		ps957.OverlayValues[898] = d898
		ps957.OverlayValues[899] = d899
		ps957.OverlayValues[900] = d900
		ps957.OverlayValues[901] = d901
		ps957.OverlayValues[902] = d902
		ps957.OverlayValues[903] = d903
		ps957.OverlayValues[904] = d904
		ps957.OverlayValues[905] = d905
		ps957.OverlayValues[906] = d906
		ps957.OverlayValues[907] = d907
		ps957.OverlayValues[908] = d908
		ps957.OverlayValues[910] = d910
		ps957.OverlayValues[911] = d911
		ps957.OverlayValues[912] = d912
		ps957.OverlayValues[913] = d913
		ps957.OverlayValues[914] = d914
		ps957.OverlayValues[915] = d915
		ps957.OverlayValues[916] = d916
		ps957.OverlayValues[917] = d917
		ps957.OverlayValues[918] = d918
		ps957.OverlayValues[919] = d919
		ps957.OverlayValues[920] = d920
		ps957.OverlayValues[921] = d921
		ps957.OverlayValues[922] = d922
		ps957.OverlayValues[923] = d923
		ps957.OverlayValues[924] = d924
		ps957.OverlayValues[925] = d925
		ps957.OverlayValues[926] = d926
		ps957.OverlayValues[927] = d927
		ps957.OverlayValues[928] = d928
		ps957.OverlayValues[929] = d929
		ps957.OverlayValues[930] = d930
		ps957.OverlayValues[931] = d931
		ps957.OverlayValues[932] = d932
		ps957.OverlayValues[933] = d933
		ps957.OverlayValues[934] = d934
		ps957.OverlayValues[935] = d935
		ps957.OverlayValues[936] = d936
		ps957.OverlayValues[937] = d937
		ps957.OverlayValues[938] = d938
		ps957.OverlayValues[939] = d939
		ps957.OverlayValues[940] = d940
		ps957.OverlayValues[941] = d941
		ps957.OverlayValues[942] = d942
		ps957.OverlayValues[943] = d943
		ps957.OverlayValues[944] = d944
		ps957.OverlayValues[945] = d945
		ps957.OverlayValues[946] = d946
		ps957.OverlayValues[947] = d947
		ps957.OverlayValues[948] = d948
		ps957.OverlayValues[949] = d949
		ps957.OverlayValues[950] = d950
		ps957.OverlayValues[951] = d951
		ps957.OverlayValues[952] = d952
		ps957.OverlayValues[953] = d953
		snap958 := d1
		snap959 := d2
		snap960 := d3
		snap961 := d4
		snap962 := d5
		snap963 := d6
		snap964 := d7
		snap965 := d8
		snap966 := d9
		snap967 := d10
		snap968 := d11
		snap969 := d12
		snap970 := d13
		snap971 := d14
		snap972 := d15
		snap973 := d17
		snap974 := d18
		snap975 := d19
		snap976 := d20
		snap977 := d21
		snap978 := d22
		snap979 := d24
		snap980 := d25
		snap981 := d26
		snap982 := d27
		snap983 := d28
		snap984 := d29
		snap985 := d30
		snap986 := d31
		snap987 := d32
		snap988 := d33
		snap989 := d34
		snap990 := d35
		snap991 := d36
		snap992 := d37
		snap993 := d38
		snap994 := d39
		snap995 := d40
		snap996 := d41
		snap997 := d42
		snap998 := d43
		snap999 := d44
		snap1000 := d45
		snap1001 := d46
		snap1002 := d47
		snap1003 := d48
		snap1004 := d49
		snap1005 := d50
		snap1006 := d51
		snap1007 := d52
		snap1008 := d53
		snap1009 := d54
		snap1010 := d55
		snap1011 := d56
		snap1012 := d57
		snap1013 := d58
		snap1014 := d59
		snap1015 := d60
		snap1016 := d61
		snap1017 := d64
		snap1018 := d65
		snap1019 := d66
		snap1020 := d132
		snap1021 := d133
		snap1022 := d134
		snap1023 := d136
		snap1024 := d137
		snap1025 := d138
		snap1026 := d139
		snap1027 := d140
		snap1028 := d141
		snap1029 := d142
		snap1030 := d143
		snap1031 := d144
		snap1032 := d145
		snap1033 := d146
		snap1034 := d147
		snap1035 := d148
		snap1036 := d149
		snap1037 := d150
		snap1038 := d151
		snap1039 := d152
		snap1040 := d153
		snap1041 := d154
		snap1042 := d155
		snap1043 := d156
		snap1044 := d157
		snap1045 := d158
		snap1046 := d159
		snap1047 := d160
		snap1048 := d161
		snap1049 := d162
		snap1050 := d163
		snap1051 := d164
		snap1052 := d165
		snap1053 := d166
		snap1054 := d167
		snap1055 := d168
		snap1056 := d169
		snap1057 := d170
		snap1058 := d171
		snap1059 := d172
		snap1060 := d175
		snap1061 := d282
		snap1062 := d283
		snap1063 := d284
		snap1064 := d285
		snap1065 := d287
		snap1066 := d288
		snap1067 := d289
		snap1068 := d290
		snap1069 := d291
		snap1070 := d292
		snap1071 := d293
		snap1072 := d294
		snap1073 := d296
		snap1074 := d298
		snap1075 := d299
		snap1076 := d300
		snap1077 := d301
		snap1078 := d302
		snap1079 := d305
		snap1080 := d429
		snap1081 := d430
		snap1082 := d431
		snap1083 := d432
		snap1084 := d433
		snap1085 := d435
		snap1086 := d436
		snap1087 := d437
		snap1088 := d439
		snap1089 := d440
		snap1090 := d441
		snap1091 := d442
		snap1092 := d443
		snap1093 := d444
		snap1094 := d445
		snap1095 := d446
		snap1096 := d447
		snap1097 := d448
		snap1098 := d449
		snap1099 := d450
		snap1100 := d451
		snap1101 := d452
		snap1102 := d453
		snap1103 := d454
		snap1104 := d455
		snap1105 := d456
		snap1106 := d457
		snap1107 := d458
		snap1108 := d459
		snap1109 := d460
		snap1110 := d461
		snap1111 := d462
		snap1112 := d463
		snap1113 := d464
		snap1114 := d465
		snap1115 := d466
		snap1116 := d467
		snap1117 := d468
		snap1118 := d469
		snap1119 := d470
		snap1120 := d471
		snap1121 := d472
		snap1122 := d473
		snap1123 := d474
		snap1124 := d475
		snap1125 := d648
		snap1126 := d649
		snap1127 := d650
		snap1128 := d652
		snap1129 := d653
		snap1130 := d654
		snap1131 := d655
		snap1132 := d656
		snap1133 := d657
		snap1134 := d658
		snap1135 := d660
		snap1136 := d662
		snap1137 := d663
		snap1138 := d664
		snap1139 := d665
		snap1140 := d668
		snap1141 := d853
		snap1142 := d854
		snap1143 := d855
		snap1144 := d856
		snap1145 := d858
		snap1146 := d859
		snap1147 := d860
		snap1148 := d861
		snap1149 := d862
		snap1150 := d863
		snap1151 := d864
		snap1152 := d865
		snap1153 := d867
		snap1154 := d868
		snap1155 := d869
		snap1156 := d870
		snap1157 := d871
		snap1158 := d873
		snap1159 := d874
		snap1160 := d875
		snap1161 := d876
		snap1162 := d877
		snap1163 := d878
		snap1164 := d879
		snap1165 := d880
		snap1166 := d881
		snap1167 := d882
		snap1168 := d883
		snap1169 := d884
		snap1170 := d885
		snap1171 := d886
		snap1172 := d887
		snap1173 := d888
		snap1174 := d889
		snap1175 := d890
		snap1176 := d891
		snap1177 := d892
		snap1178 := d893
		snap1179 := d894
		snap1180 := d895
		snap1181 := d896
		snap1182 := d897
		snap1183 := d898
		snap1184 := d899
		snap1185 := d900
		snap1186 := d901
		snap1187 := d902
		snap1188 := d903
		snap1189 := d904
		snap1190 := d905
		snap1191 := d906
		snap1192 := d907
		snap1193 := d908
		snap1194 := d910
		snap1195 := d911
		snap1196 := d912
		snap1197 := d913
		snap1198 := d914
		snap1199 := d915
		snap1200 := d916
		snap1201 := d917
		snap1202 := d918
		snap1203 := d919
		snap1204 := d920
		snap1205 := d921
		snap1206 := d922
		snap1207 := d923
		snap1208 := d924
		snap1209 := d925
		snap1210 := d926
		snap1211 := d927
		snap1212 := d928
		snap1213 := d929
		snap1214 := d930
		snap1215 := d931
		snap1216 := d932
		snap1217 := d933
		snap1218 := d934
		snap1219 := d935
		snap1220 := d936
		snap1221 := d937
		snap1222 := d938
		snap1223 := d939
		snap1224 := d940
		snap1225 := d941
		snap1226 := d942
		snap1227 := d943
		snap1228 := d944
		snap1229 := d945
		snap1230 := d946
		snap1231 := d947
		snap1232 := d948
		snap1233 := d949
		snap1234 := d950
		snap1235 := d951
		snap1236 := d952
		snap1237 := d953
		alloc1238 := ctx.SnapshotAllocState()
		if !bbs[12].Rendered {
			bbs[12].RenderPS(ps957)
		}
		ctx.RestoreAllocState(alloc1238)
		d1 = snap958
		d2 = snap959
		d3 = snap960
		d4 = snap961
		d5 = snap962
		d6 = snap963
		d7 = snap964
		d8 = snap965
		d9 = snap966
		d10 = snap967
		d11 = snap968
		d12 = snap969
		d13 = snap970
		d14 = snap971
		d15 = snap972
		d17 = snap973
		d18 = snap974
		d19 = snap975
		d20 = snap976
		d21 = snap977
		d22 = snap978
		d24 = snap979
		d25 = snap980
		d26 = snap981
		d27 = snap982
		d28 = snap983
		d29 = snap984
		d30 = snap985
		d31 = snap986
		d32 = snap987
		d33 = snap988
		d34 = snap989
		d35 = snap990
		d36 = snap991
		d37 = snap992
		d38 = snap993
		d39 = snap994
		d40 = snap995
		d41 = snap996
		d42 = snap997
		d43 = snap998
		d44 = snap999
		d45 = snap1000
		d46 = snap1001
		d47 = snap1002
		d48 = snap1003
		d49 = snap1004
		d50 = snap1005
		d51 = snap1006
		d52 = snap1007
		d53 = snap1008
		d54 = snap1009
		d55 = snap1010
		d56 = snap1011
		d57 = snap1012
		d58 = snap1013
		d59 = snap1014
		d60 = snap1015
		d61 = snap1016
		d64 = snap1017
		d65 = snap1018
		d66 = snap1019
		d132 = snap1020
		d133 = snap1021
		d134 = snap1022
		d136 = snap1023
		d137 = snap1024
		d138 = snap1025
		d139 = snap1026
		d140 = snap1027
		d141 = snap1028
		d142 = snap1029
		d143 = snap1030
		d144 = snap1031
		d145 = snap1032
		d146 = snap1033
		d147 = snap1034
		d148 = snap1035
		d149 = snap1036
		d150 = snap1037
		d151 = snap1038
		d152 = snap1039
		d153 = snap1040
		d154 = snap1041
		d155 = snap1042
		d156 = snap1043
		d157 = snap1044
		d158 = snap1045
		d159 = snap1046
		d160 = snap1047
		d161 = snap1048
		d162 = snap1049
		d163 = snap1050
		d164 = snap1051
		d165 = snap1052
		d166 = snap1053
		d167 = snap1054
		d168 = snap1055
		d169 = snap1056
		d170 = snap1057
		d171 = snap1058
		d172 = snap1059
		d175 = snap1060
		d282 = snap1061
		d283 = snap1062
		d284 = snap1063
		d285 = snap1064
		d287 = snap1065
		d288 = snap1066
		d289 = snap1067
		d290 = snap1068
		d291 = snap1069
		d292 = snap1070
		d293 = snap1071
		d294 = snap1072
		d296 = snap1073
		d298 = snap1074
		d299 = snap1075
		d300 = snap1076
		d301 = snap1077
		d302 = snap1078
		d305 = snap1079
		d429 = snap1080
		d430 = snap1081
		d431 = snap1082
		d432 = snap1083
		d433 = snap1084
		d435 = snap1085
		d436 = snap1086
		d437 = snap1087
		d439 = snap1088
		d440 = snap1089
		d441 = snap1090
		d442 = snap1091
		d443 = snap1092
		d444 = snap1093
		d445 = snap1094
		d446 = snap1095
		d447 = snap1096
		d448 = snap1097
		d449 = snap1098
		d450 = snap1099
		d451 = snap1100
		d452 = snap1101
		d453 = snap1102
		d454 = snap1103
		d455 = snap1104
		d456 = snap1105
		d457 = snap1106
		d458 = snap1107
		d459 = snap1108
		d460 = snap1109
		d461 = snap1110
		d462 = snap1111
		d463 = snap1112
		d464 = snap1113
		d465 = snap1114
		d466 = snap1115
		d467 = snap1116
		d468 = snap1117
		d469 = snap1118
		d470 = snap1119
		d471 = snap1120
		d472 = snap1121
		d473 = snap1122
		d474 = snap1123
		d475 = snap1124
		d648 = snap1125
		d649 = snap1126
		d650 = snap1127
		d652 = snap1128
		d653 = snap1129
		d654 = snap1130
		d655 = snap1131
		d656 = snap1132
		d657 = snap1133
		d658 = snap1134
		d660 = snap1135
		d662 = snap1136
		d663 = snap1137
		d664 = snap1138
		d665 = snap1139
		d668 = snap1140
		d853 = snap1141
		d854 = snap1142
		d855 = snap1143
		d856 = snap1144
		d858 = snap1145
		d859 = snap1146
		d860 = snap1147
		d861 = snap1148
		d862 = snap1149
		d863 = snap1150
		d864 = snap1151
		d865 = snap1152
		d867 = snap1153
		d868 = snap1154
		d869 = snap1155
		d870 = snap1156
		d871 = snap1157
		d873 = snap1158
		d874 = snap1159
		d875 = snap1160
		d876 = snap1161
		d877 = snap1162
		d878 = snap1163
		d879 = snap1164
		d880 = snap1165
		d881 = snap1166
		d882 = snap1167
		d883 = snap1168
		d884 = snap1169
		d885 = snap1170
		d886 = snap1171
		d887 = snap1172
		d888 = snap1173
		d889 = snap1174
		d890 = snap1175
		d891 = snap1176
		d892 = snap1177
		d893 = snap1178
		d894 = snap1179
		d895 = snap1180
		d896 = snap1181
		d897 = snap1182
		d898 = snap1183
		d899 = snap1184
		d900 = snap1185
		d901 = snap1186
		d902 = snap1187
		d903 = snap1188
		d904 = snap1189
		d905 = snap1190
		d906 = snap1191
		d907 = snap1192
		d908 = snap1193
		d910 = snap1194
		d911 = snap1195
		d912 = snap1196
		d913 = snap1197
		d914 = snap1198
		d915 = snap1199
		d916 = snap1200
		d917 = snap1201
		d918 = snap1202
		d919 = snap1203
		d920 = snap1204
		d921 = snap1205
		d922 = snap1206
		d923 = snap1207
		d924 = snap1208
		d925 = snap1209
		d926 = snap1210
		d927 = snap1211
		d928 = snap1212
		d929 = snap1213
		d930 = snap1214
		d931 = snap1215
		d932 = snap1216
		d933 = snap1217
		d934 = snap1218
		d935 = snap1219
		d936 = snap1220
		d937 = snap1221
		d938 = snap1222
		d939 = snap1223
		d940 = snap1224
		d941 = snap1225
		d942 = snap1226
		d943 = snap1227
		d944 = snap1228
		d945 = snap1229
		d946 = snap1230
		d947 = snap1231
		d948 = snap1232
		d949 = snap1233
		d950 = snap1234
		d951 = snap1235
		d952 = snap1236
		d953 = snap1237
		if !bbs[11].Rendered {
			return bbs[11].RenderPS(ps956)
		}
		return result
		ctx.FreeDesc(&d952)
		return result
	}
	ps1239 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps1239)
	ctx.MarkLabel(lbl0)
	d1240 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
	ctx.BindReg(r0, &d1240)
	ctx.BindReg(r1, &d1240)
	ctx.EmitMovPairToResult(&d1240, &result)
	ctx.FreeReg(r0)
	ctx.FreeReg(r1)
	ctx.ResolveFixups()
	if idxPinned {
		ctx.UnprotectReg(idxPinnedReg)
	}
	if resultRegsProtected {
		ctx.UnprotectReg(result.Reg2)
		ctx.UnprotectReg(result.Reg)
	}
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
