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
	var d57 scm.JITValueDesc
	_ = d57
	var d60 scm.JITValueDesc
	_ = d60
	var d61 scm.JITValueDesc
	_ = d61
	var d62 scm.JITValueDesc
	_ = d62
	var d177 scm.JITValueDesc
	_ = d177
	var d178 scm.JITValueDesc
	_ = d178
	var d179 scm.JITValueDesc
	_ = d179
	var d180 scm.JITValueDesc
	_ = d180
	var d181 scm.JITValueDesc
	_ = d181
	var d182 scm.JITValueDesc
	_ = d182
	var d183 scm.JITValueDesc
	_ = d183
	var d184 scm.JITValueDesc
	_ = d184
	var d185 scm.JITValueDesc
	_ = d185
	var d186 scm.JITValueDesc
	_ = d186
	var d187 scm.JITValueDesc
	_ = d187
	var d188 scm.JITValueDesc
	_ = d188
	var d189 scm.JITValueDesc
	_ = d189
	var d190 scm.JITValueDesc
	_ = d190
	var d191 scm.JITValueDesc
	_ = d191
	var d192 scm.JITValueDesc
	_ = d192
	var d193 scm.JITValueDesc
	_ = d193
	var d194 scm.JITValueDesc
	_ = d194
	var d195 scm.JITValueDesc
	_ = d195
	var d196 scm.JITValueDesc
	_ = d196
	var d197 scm.JITValueDesc
	_ = d197
	var d198 scm.JITValueDesc
	_ = d198
	var d199 scm.JITValueDesc
	_ = d199
	var d200 scm.JITValueDesc
	_ = d200
	var d201 scm.JITValueDesc
	_ = d201
	var d202 scm.JITValueDesc
	_ = d202
	var d203 scm.JITValueDesc
	_ = d203
	var d204 scm.JITValueDesc
	_ = d204
	var d205 scm.JITValueDesc
	_ = d205
	var d206 scm.JITValueDesc
	_ = d206
	var d209 scm.JITValueDesc
	_ = d209
	var d386 scm.JITValueDesc
	_ = d386
	var d387 scm.JITValueDesc
	_ = d387
	var d388 scm.JITValueDesc
	_ = d388
	var d389 scm.JITValueDesc
	_ = d389
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
	var d400 scm.JITValueDesc
	_ = d400
	var d402 scm.JITValueDesc
	_ = d402
	var d403 scm.JITValueDesc
	_ = d403
	var d404 scm.JITValueDesc
	_ = d404
	var d508 scm.JITValueDesc
	_ = d508
	var d509 scm.JITValueDesc
	_ = d509
	var d512 scm.JITValueDesc
	_ = d512
	var d619 scm.JITValueDesc
	_ = d619
	var d620 scm.JITValueDesc
	_ = d620
	var d621 scm.JITValueDesc
	_ = d621
	var d622 scm.JITValueDesc
	_ = d622
	var d623 scm.JITValueDesc
	_ = d623
	var d625 scm.JITValueDesc
	_ = d625
	var d626 scm.JITValueDesc
	_ = d626
	var d627 scm.JITValueDesc
	_ = d627
	var d628 scm.JITValueDesc
	_ = d628
	var d629 scm.JITValueDesc
	_ = d629
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
	var d637 scm.JITValueDesc
	_ = d637
	var d638 scm.JITValueDesc
	_ = d638
	var d639 scm.JITValueDesc
	_ = d639
	var d640 scm.JITValueDesc
	_ = d640
	var d641 scm.JITValueDesc
	_ = d641
	var d642 scm.JITValueDesc
	_ = d642
	var d643 scm.JITValueDesc
	_ = d643
	var d644 scm.JITValueDesc
	_ = d644
	var d645 scm.JITValueDesc
	_ = d645
	var d646 scm.JITValueDesc
	_ = d646
	var d647 scm.JITValueDesc
	_ = d647
	var d648 scm.JITValueDesc
	_ = d648
	var d649 scm.JITValueDesc
	_ = d649
	var d650 scm.JITValueDesc
	_ = d650
	var d651 scm.JITValueDesc
	_ = d651
	var d652 scm.JITValueDesc
	_ = d652
	var d653 scm.JITValueDesc
	_ = d653
	var d654 scm.JITValueDesc
	_ = d654
	var d655 scm.JITValueDesc
	_ = d655
	var d944 scm.JITValueDesc
	_ = d944
	var d945 scm.JITValueDesc
	_ = d945
	var d946 scm.JITValueDesc
	_ = d946
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
	var d956 scm.JITValueDesc
	_ = d956
	var d958 scm.JITValueDesc
	_ = d958
	var d959 scm.JITValueDesc
	_ = d959
	var d1115 scm.JITValueDesc
	_ = d1115
	var d1116 scm.JITValueDesc
	_ = d1116
	var d1119 scm.JITValueDesc
	_ = d1119
	var d1278 scm.JITValueDesc
	_ = d1278
	var d1279 scm.JITValueDesc
	_ = d1279
	var d1280 scm.JITValueDesc
	_ = d1280
	var d1281 scm.JITValueDesc
	_ = d1281
	var d1283 scm.JITValueDesc
	_ = d1283
	var d1284 scm.JITValueDesc
	_ = d1284
	var d1285 scm.JITValueDesc
	_ = d1285
	var d1286 scm.JITValueDesc
	_ = d1286
	var d1287 scm.JITValueDesc
	_ = d1287
	var d1288 scm.JITValueDesc
	_ = d1288
	var d1289 scm.JITValueDesc
	_ = d1289
	var d1290 scm.JITValueDesc
	_ = d1290
	var d1291 scm.JITValueDesc
	_ = d1291
	var d1292 scm.JITValueDesc
	_ = d1292
	var d1294 scm.JITValueDesc
	_ = d1294
	var d1295 scm.JITValueDesc
	_ = d1295
	var d1296 scm.JITValueDesc
	_ = d1296
	var d1297 scm.JITValueDesc
	_ = d1297
	var d1298 scm.JITValueDesc
	_ = d1298
	var d1299 scm.JITValueDesc
	_ = d1299
	var d1300 scm.JITValueDesc
	_ = d1300
	var d1301 scm.JITValueDesc
	_ = d1301
	var d1302 scm.JITValueDesc
	_ = d1302
	var d1303 scm.JITValueDesc
	_ = d1303
	var d1304 scm.JITValueDesc
	_ = d1304
	var d1305 scm.JITValueDesc
	_ = d1305
	var d1306 scm.JITValueDesc
	_ = d1306
	var d1307 scm.JITValueDesc
	_ = d1307
	var d1308 scm.JITValueDesc
	_ = d1308
	var d1309 scm.JITValueDesc
	_ = d1309
	var d1310 scm.JITValueDesc
	_ = d1310
	var d1311 scm.JITValueDesc
	_ = d1311
	var d1312 scm.JITValueDesc
	_ = d1312
	var d1313 scm.JITValueDesc
	_ = d1313
	var d1314 scm.JITValueDesc
	_ = d1314
	var d1315 scm.JITValueDesc
	_ = d1315
	var d1316 scm.JITValueDesc
	_ = d1316
	var d1317 scm.JITValueDesc
	_ = d1317
	var d1318 scm.JITValueDesc
	_ = d1318
	var d1319 scm.JITValueDesc
	_ = d1319
	var d1320 scm.JITValueDesc
	_ = d1320
	var d1321 scm.JITValueDesc
	_ = d1321
	var d1322 scm.JITValueDesc
	_ = d1322
	var d1323 scm.JITValueDesc
	_ = d1323
	var d1324 scm.JITValueDesc
	_ = d1324
	var d1325 scm.JITValueDesc
	_ = d1325
	var d1326 scm.JITValueDesc
	_ = d1326
	var d1327 scm.JITValueDesc
	_ = d1327
	var d1328 scm.JITValueDesc
	_ = d1328
	var d1329 scm.JITValueDesc
	_ = d1329
	var d1330 scm.JITValueDesc
	_ = d1330
	var d1331 scm.JITValueDesc
	_ = d1331
	var d1332 scm.JITValueDesc
	_ = d1332
	var d1333 scm.JITValueDesc
	_ = d1333
	var d1334 scm.JITValueDesc
	_ = d1334
	var d1335 scm.JITValueDesc
	_ = d1335
	var d1336 scm.JITValueDesc
	_ = d1336
	var d1337 scm.JITValueDesc
	_ = d1337
	var d1338 scm.JITValueDesc
	_ = d1338
	var d1339 scm.JITValueDesc
	_ = d1339
	var d1340 scm.JITValueDesc
	_ = d1340
	var d1341 scm.JITValueDesc
	_ = d1341
	var d1342 scm.JITValueDesc
	_ = d1342
	var d1343 scm.JITValueDesc
	_ = d1343
	var d1344 scm.JITValueDesc
	_ = d1344
	var d1345 scm.JITValueDesc
	_ = d1345
	var d1346 scm.JITValueDesc
	_ = d1346
	var d1347 scm.JITValueDesc
	_ = d1347
	var d1348 scm.JITValueDesc
	_ = d1348
	var d1349 scm.JITValueDesc
	_ = d1349
	var d1350 scm.JITValueDesc
	_ = d1350
	var d1351 scm.JITValueDesc
	_ = d1351
	var d1352 scm.JITValueDesc
	_ = d1352
	var d1353 scm.JITValueDesc
	_ = d1353
	var d1354 scm.JITValueDesc
	_ = d1354
	var d1355 scm.JITValueDesc
	_ = d1355
	var d1356 scm.JITValueDesc
	_ = d1356
	var d1357 scm.JITValueDesc
	_ = d1357
	var d1358 scm.JITValueDesc
	_ = d1358
	var d1359 scm.JITValueDesc
	_ = d1359
	var d1360 scm.JITValueDesc
	_ = d1360
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
			val := *(*uint32)(unsafe.Pointer(fieldAddr))
			d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).seqCount))
			r7 := ctx.AllocReg()
			ctx.EmitMovRegMemL(r7, thisptr.Reg, off)
			d16 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r7}
			ctx.BindReg(r7, &d16)
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
		ctx.FreeDesc(&d16)
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
				ctx.EnsureDesc(&d26)
				if phiHomeOK2 {
					ctx.EmitMovToReg(r0, d26)
				} else {
					ctx.EmitStoreToStack(d26, int32(bbs[1].PhiBase)+int32(0))
				}
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d27 := ps.PhiValues[1]
				ctx.EnsureDesc(&d27)
				if phiHomeOK3 {
					ctx.EmitMovToReg(r1, d27)
				} else {
					ctx.EmitStoreToStack(d27, int32(bbs[1].PhiBase)+int32(16))
				}
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d28 := ps.PhiValues[2]
				ctx.EnsureDesc(&d28)
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
			r8 := ctx.AllocReg()
			ctx.EmitMovRegReg(r8, d29.Reg)
			ctx.EmitShlRegImm8(r8, 32)
			ctx.EmitShrRegImm8(r8, 32)
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
			ctx.BindReg(r8, &d30)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d31 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48)
			r9 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r9, thisptr.Reg, off)
			d31 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r9}
			ctx.BindReg(r9, &d31)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d31)
		ctx.EnsureDesc(&d31)
		var d32 scm.JITValueDesc
		if d31.Loc == scm.LocImm {
			d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d31.Imm.Int()))))}
		} else {
			r10 := ctx.AllocReg()
			ctx.EmitMovRegReg(r10, d31.Reg)
			ctx.EmitShlRegImm8(r10, 56)
			ctx.EmitShrRegImm8(r10, 56)
			d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
			ctx.BindReg(r10, &d32)
		}
		ctx.FreeDesc(&d31)
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
			r11 := ctx.AllocRegExcept(d30.Reg, d32.Reg)
			ctx.EmitMovRegReg(r11, d30.Reg)
			ctx.EmitImulInt64(r11, d32.Reg)
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
			ctx.BindReg(r11, &d33)
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
			r12 := ctx.AllocRegExcept(d33.Reg)
			ctx.EmitMovRegReg(r12, d33.Reg)
			ctx.EmitShrRegImm8(r12, 6)
			d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
			ctx.BindReg(r12, &d34)
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
			r13 := ctx.AllocRegExcept(d33.Reg)
			ctx.EmitMovRegReg(r13, d33.Reg)
			ctx.EmitAndRegImm32(r13, 63)
			d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
			ctx.BindReg(r13, &d35)
		}
		if d35.Loc == scm.LocReg && d33.Loc == scm.LocReg && d35.Reg == d33.Reg {
			ctx.TransferReg(d33.Reg)
			d33.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d33)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d36 scm.JITValueDesc
		r14 := ctx.AllocReg()
		r15 := ctx.AllocRegExcept(r14)
		r16 := ctx.AllocRegExcept(r14, r15)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			ctx.EmitMovRegImm64(r14, uint64(dataPtr))
			ctx.EmitMovRegImm64(r15, uint64(sliceLen))
			ctx.EmitMovRegImm64(r16, uint64(sliceCap))
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			ctx.EmitMovRegMem(r14, thisptr.Reg, off)
			ctx.EmitMovRegMem(r15, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r16, thisptr.Reg, off+16)
		}
		d36 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r14, Reg2: r15, Reg3: r16}
		ctx.BindReg(r14, &d36)
		ctx.BindReg(r15, &d36)
		ctx.BindReg(r16, &d36)
		ctx.BindReg(r14, &d36)
		ctx.BindReg(r15, &d36)
		ctx.BindReg(r16, &d36)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d34)
		ctx.ReclaimUntrackedRegs()
		d38 = ctx.EmitSliceElementAddress(&d36, &d34, 8)
		ctx.EnsureDesc(&d38)
		ctx.EmitMovRegMem(d38.Reg, d38.Reg, 0)
		d37 = d38
		d37.Type = scm.TagInt
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d37)
		ctx.EnsureDesc(&d35)
		ctx.EnsureDescsTogether(&d37, &d35)
		var d39 scm.JITValueDesc
		if d37.Loc == scm.LocImm && d35.Loc == scm.LocImm {
			d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d37.Imm.Int()) << uint64(d35.Imm.Int())))}
		} else if d35.Loc == scm.LocImm {
			r17 := ctx.AllocRegExcept(d37.Reg)
			ctx.EmitMovRegReg(r17, d37.Reg)
			ctx.EmitShlRegImm8(r17, uint8(d35.Imm.Int()))
			d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			ctx.BindReg(r17, &d39)
		} else {
			{
				shiftSrc := d37.Reg
				r18 := ctx.AllocRegExcept(d37.Reg, d35.Reg)
				ctx.EmitMovRegReg(r18, d37.Reg)
				shiftSrc = r18
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
		d41.Type = scm.TagInt
		ctx.FreeDesc(&d40)
		ctx.ReclaimUntrackedRegs()
		d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d35)
		ctx.EnsureDescsTogether(&d43, &d35)
		var d44 scm.JITValueDesc
		if d43.Loc == scm.LocImm && d35.Loc == scm.LocImm {
			d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d43.Imm.Int() - d35.Imm.Int())}
		} else if d35.Loc == scm.LocImm && d35.Imm.Int() == 0 {
			r19 := ctx.AllocRegExcept(d43.Reg)
			ctx.EmitMovRegReg(r19, d43.Reg)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d44)
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
			r20 := ctx.AllocRegExcept(d43.Reg, d35.Reg)
			ctx.EmitMovRegReg(r20, d43.Reg)
			ctx.EmitSubInt64(r20, d35.Reg)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
			ctx.BindReg(r20, &d44)
		}
		if d44.Loc == scm.LocReg && d43.Loc == scm.LocReg && d44.Reg == d43.Reg {
			ctx.TransferReg(d43.Reg)
			d43.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d35)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d41)
		ctx.EnsureDesc(&d44)
		ctx.EnsureDescsTogether(&d41, &d44)
		var d45 scm.JITValueDesc
		if d41.Loc == scm.LocImm && d44.Loc == scm.LocImm {
			d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d41.Imm.Int()) >> uint64(d44.Imm.Int())))}
		} else if d44.Loc == scm.LocImm {
			r21 := ctx.AllocRegExcept(d41.Reg)
			ctx.EmitMovRegReg(r21, d41.Reg)
			ctx.EmitShrRegImm8(r21, uint8(d44.Imm.Int()))
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			ctx.BindReg(r21, &d45)
		} else {
			{
				shiftSrc := d41.Reg
				r22 := ctx.AllocRegExcept(d41.Reg, d44.Reg)
				ctx.EmitMovRegReg(r22, d41.Reg)
				shiftSrc = r22
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
			r23 := ctx.AllocRegExcept(d39.Reg)
			ctx.EmitMovRegReg(r23, d39.Reg)
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
			ctx.BindReg(r23, &d46)
		} else if d39.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d45.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d39.Imm.Int()))
			ctx.EmitOrInt64(scratch, d45.Reg)
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d46)
		} else if d45.Loc == scm.LocImm {
			r24 := ctx.AllocRegExcept(d39.Reg)
			ctx.EmitMovRegReg(r24, d39.Reg)
			if d45.Imm.Int() >= -2147483648 && d45.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r24, int32(d45.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d45.Imm.Int()))
				ctx.EmitOrInt64(r24, scm.RegR11)
			}
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d46)
		} else {
			r25 := ctx.AllocRegExcept(d39.Reg, d45.Reg)
			ctx.EmitMovRegReg(r25, d39.Reg)
			ctx.EmitOrInt64(r25, d45.Reg)
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
			ctx.BindReg(r25, &d46)
		}
		if d46.Loc == scm.LocReg && d39.Loc == scm.LocReg && d46.Reg == d39.Reg {
			ctx.TransferReg(d39.Reg)
			d39.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d39)
		ctx.FreeDesc(&d45)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d47 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48)
			r26 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r26, thisptr.Reg, off)
			d47 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r26}
			ctx.BindReg(r26, &d47)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d47)
		ctx.EnsureDesc(&d47)
		var d48 scm.JITValueDesc
		if d47.Loc == scm.LocImm {
			d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d47.Imm.Int()))))}
		} else {
			r27 := ctx.AllocReg()
			ctx.EmitMovRegReg(r27, d47.Reg)
			ctx.EmitShlRegImm8(r27, 56)
			ctx.EmitShrRegImm8(r27, 56)
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
			ctx.BindReg(r27, &d48)
		}
		ctx.FreeDesc(&d47)
		ctx.ReclaimUntrackedRegs()
		d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d48)
		ctx.EnsureDescsTogether(&d49, &d48)
		var d50 scm.JITValueDesc
		if d49.Loc == scm.LocImm && d48.Loc == scm.LocImm {
			d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d49.Imm.Int() - d48.Imm.Int())}
		} else if d48.Loc == scm.LocImm && d48.Imm.Int() == 0 {
			r28 := ctx.AllocRegExcept(d49.Reg)
			ctx.EmitMovRegReg(r28, d49.Reg)
			d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
			ctx.BindReg(r28, &d50)
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
			r29 := ctx.AllocRegExcept(d49.Reg, d48.Reg)
			ctx.EmitMovRegReg(r29, d49.Reg)
			ctx.EmitSubInt64(r29, d48.Reg)
			d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
			ctx.BindReg(r29, &d50)
		}
		if d50.Loc == scm.LocReg && d49.Loc == scm.LocReg && d50.Reg == d49.Reg {
			ctx.TransferReg(d49.Reg)
			d49.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d48)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d46)
		ctx.EnsureDesc(&d50)
		ctx.EnsureDescsTogether(&d46, &d50)
		var d51 scm.JITValueDesc
		if d46.Loc == scm.LocImm && d50.Loc == scm.LocImm {
			d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d46.Imm.Int()) >> uint64(d50.Imm.Int())))}
		} else if d50.Loc == scm.LocImm {
			r30 := ctx.AllocRegExcept(d46.Reg)
			ctx.EmitMovRegReg(r30, d46.Reg)
			ctx.EmitShrRegImm8(r30, uint8(d50.Imm.Int()))
			d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
			ctx.BindReg(r30, &d51)
		} else {
			{
				shiftSrc := d46.Reg
				r31 := ctx.AllocRegExcept(d46.Reg, d50.Reg)
				ctx.EmitMovRegReg(r31, d46.Reg)
				shiftSrc = r31
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d50.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d50.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d50.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
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
		ctx.EnsureDesc(&d51)
		ctx.EnsureDesc(&d51)
		ctx.EnsureDesc(&d51)
		var d52 scm.JITValueDesc
		if d51.Loc == scm.LocImm {
			d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d51.Imm.Int()))))}
		} else {
			r32 := ctx.AllocReg()
			ctx.EmitMovRegReg(r32, d51.Reg)
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
			ctx.BindReg(r32, &d52)
		}
		ctx.FreeDesc(&d51)
		var d53 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56)
			r33 := ctx.AllocReg()
			ctx.EmitMovRegMem(r33, thisptr.Reg, off)
			d53 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r33}
			ctx.BindReg(r33, &d53)
		}
		ctx.EnsureDesc(&d52)
		ctx.EnsureDesc(&d53)
		ctx.EnsureDescsTogether(&d52, &d53)
		var d54 scm.JITValueDesc
		if d52.Loc == scm.LocImm && d53.Loc == scm.LocImm {
			d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() + d53.Imm.Int())}
		} else if d53.Loc == scm.LocImm && d53.Imm.Int() == 0 {
			r34 := ctx.AllocRegExcept(d52.Reg)
			ctx.EmitMovRegReg(r34, d52.Reg)
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
			ctx.BindReg(r34, &d54)
		} else if d52.Loc == scm.LocImm && d52.Imm.Int() == 0 {
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d53.Reg}
			ctx.BindReg(d53.Reg, &d54)
		} else if d52.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d53.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d52.Imm.Int()))
			ctx.EmitAddInt64(scratch, d53.Reg)
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d54)
		} else if d53.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d52.Reg)
			ctx.EmitMovRegReg(scratch, d52.Reg)
			if d53.Imm.Int() >= -2147483648 && d53.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d53.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d53.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d54)
		} else {
			r35 := ctx.AllocRegExcept(d52.Reg, d53.Reg)
			ctx.EmitMovRegReg(r35, d52.Reg)
			ctx.EmitAddInt64(r35, d53.Reg)
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
			ctx.BindReg(r35, &d54)
		}
		if d54.Loc == scm.LocReg && d52.Loc == scm.LocReg && d54.Reg == d52.Reg {
			ctx.TransferReg(d52.Reg)
			d52.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d52)
		ctx.FreeDesc(&d53)
		ctx.EnsureDesc(&d54)
		ctx.EnsureDesc(&d54)
		var d55 scm.JITValueDesc
		if d54.Loc == scm.LocImm {
			d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d54.Imm.Int()))))}
		} else {
			r36 := ctx.AllocReg()
			ctx.EmitMovRegReg(r36, d54.Reg)
			ctx.EmitShlRegImm8(r36, 32)
			ctx.EmitShrRegImm8(r36, 32)
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
			ctx.BindReg(r36, &d55)
		}
		ctx.FreeDesc(&d54)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d55)
		ctx.EnsureDescsTogether(&idxInt, &d55)
		var d56 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm && d55.Loc == scm.LocImm {
			d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d55.Imm.Int()))}
		} else if d55.Loc == scm.LocImm {
			r37 := ctx.AllocRegExcept(idxInt.Reg)
			if d55.Imm.Int() >= -2147483648 && d55.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(idxInt.Reg, int32(d55.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d55.Imm.Int()))
				ctx.EmitCmpInt64(idxInt.Reg, scm.RegR11)
			}
			d56 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r37, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r37, &d56)
		} else if idxInt.Loc == scm.LocImm {
			r38 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d55.Reg)
			d56 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r38, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r38, &d56)
		} else {
			r39 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitCmpInt64(idxInt.Reg, d55.Reg)
			d56 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r39, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r39, &d56)
		}
		ctx.FreeDesc(&d55)
		d57 = d56
		ctx.EnsureDesc(&d57)
		if d57.Loc != scm.LocImm && d57.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d57.Loc == scm.LocImm {
			if d57.Imm.Bool() {
				if ps.General {
				}
				ps58 := scm.PhiState{General: ps.General}
				ps58.OverlayValues = make([]scm.JITValueDesc, 58)
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
				ps58.OverlayValues[57] = d57
				return bbs[3].RenderPS(ps58)
			}
			if ps.General {
			}
			ps59 := scm.PhiState{General: ps.General}
			ps59.OverlayValues = make([]scm.JITValueDesc, 58)
			ps59.OverlayValues[5] = d5
			ps59.OverlayValues[6] = d6
			ps59.OverlayValues[7] = d7
			ps59.OverlayValues[8] = d8
			ps59.OverlayValues[9] = d9
			ps59.OverlayValues[10] = d10
			ps59.OverlayValues[11] = d11
			ps59.OverlayValues[12] = d12
			ps59.OverlayValues[13] = d13
			ps59.OverlayValues[14] = d14
			ps59.OverlayValues[15] = d15
			ps59.OverlayValues[16] = d16
			ps59.OverlayValues[17] = d17
			ps59.OverlayValues[18] = d18
			ps59.OverlayValues[19] = d19
			ps59.OverlayValues[20] = d20
			ps59.OverlayValues[21] = d21
			ps59.OverlayValues[23] = d23
			ps59.OverlayValues[24] = d24
			ps59.OverlayValues[25] = d25
			ps59.OverlayValues[26] = d26
			ps59.OverlayValues[27] = d27
			ps59.OverlayValues[28] = d28
			ps59.OverlayValues[29] = d29
			ps59.OverlayValues[30] = d30
			ps59.OverlayValues[31] = d31
			ps59.OverlayValues[32] = d32
			ps59.OverlayValues[33] = d33
			ps59.OverlayValues[34] = d34
			ps59.OverlayValues[35] = d35
			ps59.OverlayValues[36] = d36
			ps59.OverlayValues[37] = d37
			ps59.OverlayValues[38] = d38
			ps59.OverlayValues[39] = d39
			ps59.OverlayValues[40] = d40
			ps59.OverlayValues[41] = d41
			ps59.OverlayValues[42] = d42
			ps59.OverlayValues[43] = d43
			ps59.OverlayValues[44] = d44
			ps59.OverlayValues[45] = d45
			ps59.OverlayValues[46] = d46
			ps59.OverlayValues[47] = d47
			ps59.OverlayValues[48] = d48
			ps59.OverlayValues[49] = d49
			ps59.OverlayValues[50] = d50
			ps59.OverlayValues[51] = d51
			ps59.OverlayValues[52] = d52
			ps59.OverlayValues[53] = d53
			ps59.OverlayValues[54] = d54
			ps59.OverlayValues[55] = d55
			ps59.OverlayValues[56] = d56
			ps59.OverlayValues[57] = d57
			return bbs[5].RenderPS(ps59)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d60 := ps.PhiValues[0]
				ctx.EnsureDesc(&d60)
				if phiHomeOK2 {
					ctx.EmitMovToReg(r0, d60)
				} else {
					ctx.EmitStoreToStack(d60, int32(bbs[1].PhiBase)+int32(0))
				}
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d61 := ps.PhiValues[1]
				ctx.EnsureDesc(&d61)
				if phiHomeOK3 {
					ctx.EmitMovToReg(r1, d61)
				} else {
					ctx.EmitStoreToStack(d61, int32(bbs[1].PhiBase)+int32(16))
				}
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d62 := ps.PhiValues[2]
				ctx.EnsureDesc(&d62)
				if phiHomeOK4 {
					ctx.EmitMovToReg(r2, d62)
				} else {
					ctx.EmitStoreToStack(d62, int32(bbs[1].PhiBase)+int32(32))
				}
			}
			ps.General = true
			return bbs[1].RenderPS(ps)
		}
		lbl16 := ctx.ReserveLabel()
		lbl17 := ctx.ReserveLabel()
		ctx.EmitJump(d57.Condition, lbl16)
		ctx.EmitJmp(lbl17)
		snap63 := d5
		snap64 := d6
		snap65 := d7
		snap66 := d8
		snap67 := d9
		snap68 := d10
		snap69 := d11
		snap70 := d12
		snap71 := d13
		snap72 := d14
		snap73 := d15
		snap74 := d16
		snap75 := d17
		snap76 := d18
		snap77 := d19
		snap78 := d20
		snap79 := d21
		snap80 := d23
		snap81 := d24
		snap82 := d25
		snap83 := d26
		snap84 := d27
		snap85 := d28
		snap86 := d29
		snap87 := d30
		snap88 := d31
		snap89 := d32
		snap90 := d33
		snap91 := d34
		snap92 := d35
		snap93 := d36
		snap94 := d37
		snap95 := d38
		snap96 := d39
		snap97 := d40
		snap98 := d41
		snap99 := d42
		snap100 := d43
		snap101 := d44
		snap102 := d45
		snap103 := d46
		snap104 := d47
		snap105 := d48
		snap106 := d49
		snap107 := d50
		snap108 := d51
		snap109 := d52
		snap110 := d53
		snap111 := d54
		snap112 := d55
		snap113 := d56
		snap114 := d57
		snap115 := d60
		snap116 := d61
		snap117 := d62
		alloc118 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl16)
		ctx.EmitJmp(lbl4)
		ctx.RestoreAllocState(alloc118)
		d5 = snap63
		d6 = snap64
		d7 = snap65
		d8 = snap66
		d9 = snap67
		d10 = snap68
		d11 = snap69
		d12 = snap70
		d13 = snap71
		d14 = snap72
		d15 = snap73
		d16 = snap74
		d17 = snap75
		d18 = snap76
		d19 = snap77
		d20 = snap78
		d21 = snap79
		d23 = snap80
		d24 = snap81
		d25 = snap82
		d26 = snap83
		d27 = snap84
		d28 = snap85
		d29 = snap86
		d30 = snap87
		d31 = snap88
		d32 = snap89
		d33 = snap90
		d34 = snap91
		d35 = snap92
		d36 = snap93
		d37 = snap94
		d38 = snap95
		d39 = snap96
		d40 = snap97
		d41 = snap98
		d42 = snap99
		d43 = snap100
		d44 = snap101
		d45 = snap102
		d46 = snap103
		d47 = snap104
		d48 = snap105
		d49 = snap106
		d50 = snap107
		d51 = snap108
		d52 = snap109
		d53 = snap110
		d54 = snap111
		d55 = snap112
		d56 = snap113
		d57 = snap114
		d60 = snap115
		d61 = snap116
		d62 = snap117
		ctx.MarkLabel(lbl17)
		ctx.EmitJmp(lbl6)
		ctx.RestoreAllocState(alloc118)
		d5 = snap63
		d6 = snap64
		d7 = snap65
		d8 = snap66
		d9 = snap67
		d10 = snap68
		d11 = snap69
		d12 = snap70
		d13 = snap71
		d14 = snap72
		d15 = snap73
		d16 = snap74
		d17 = snap75
		d18 = snap76
		d19 = snap77
		d20 = snap78
		d21 = snap79
		d23 = snap80
		d24 = snap81
		d25 = snap82
		d26 = snap83
		d27 = snap84
		d28 = snap85
		d29 = snap86
		d30 = snap87
		d31 = snap88
		d32 = snap89
		d33 = snap90
		d34 = snap91
		d35 = snap92
		d36 = snap93
		d37 = snap94
		d38 = snap95
		d39 = snap96
		d40 = snap97
		d41 = snap98
		d42 = snap99
		d43 = snap100
		d44 = snap101
		d45 = snap102
		d46 = snap103
		d47 = snap104
		d48 = snap105
		d49 = snap106
		d50 = snap107
		d51 = snap108
		d52 = snap109
		d53 = snap110
		d54 = snap111
		d55 = snap112
		d56 = snap113
		d57 = snap114
		d60 = snap115
		d61 = snap116
		d62 = snap117
		ps119 := scm.PhiState{General: true}
		ps119.OverlayValues = make([]scm.JITValueDesc, 63)
		ps119.OverlayValues[5] = d5
		ps119.OverlayValues[6] = d6
		ps119.OverlayValues[7] = d7
		ps119.OverlayValues[8] = d8
		ps119.OverlayValues[9] = d9
		ps119.OverlayValues[10] = d10
		ps119.OverlayValues[11] = d11
		ps119.OverlayValues[12] = d12
		ps119.OverlayValues[13] = d13
		ps119.OverlayValues[14] = d14
		ps119.OverlayValues[15] = d15
		ps119.OverlayValues[16] = d16
		ps119.OverlayValues[17] = d17
		ps119.OverlayValues[18] = d18
		ps119.OverlayValues[19] = d19
		ps119.OverlayValues[20] = d20
		ps119.OverlayValues[21] = d21
		ps119.OverlayValues[23] = d23
		ps119.OverlayValues[24] = d24
		ps119.OverlayValues[25] = d25
		ps119.OverlayValues[26] = d26
		ps119.OverlayValues[27] = d27
		ps119.OverlayValues[28] = d28
		ps119.OverlayValues[29] = d29
		ps119.OverlayValues[30] = d30
		ps119.OverlayValues[31] = d31
		ps119.OverlayValues[32] = d32
		ps119.OverlayValues[33] = d33
		ps119.OverlayValues[34] = d34
		ps119.OverlayValues[35] = d35
		ps119.OverlayValues[36] = d36
		ps119.OverlayValues[37] = d37
		ps119.OverlayValues[38] = d38
		ps119.OverlayValues[39] = d39
		ps119.OverlayValues[40] = d40
		ps119.OverlayValues[41] = d41
		ps119.OverlayValues[42] = d42
		ps119.OverlayValues[43] = d43
		ps119.OverlayValues[44] = d44
		ps119.OverlayValues[45] = d45
		ps119.OverlayValues[46] = d46
		ps119.OverlayValues[47] = d47
		ps119.OverlayValues[48] = d48
		ps119.OverlayValues[49] = d49
		ps119.OverlayValues[50] = d50
		ps119.OverlayValues[51] = d51
		ps119.OverlayValues[52] = d52
		ps119.OverlayValues[53] = d53
		ps119.OverlayValues[54] = d54
		ps119.OverlayValues[55] = d55
		ps119.OverlayValues[56] = d56
		ps119.OverlayValues[57] = d57
		ps119.OverlayValues[60] = d60
		ps119.OverlayValues[61] = d61
		ps119.OverlayValues[62] = d62
		ps120 := scm.PhiState{General: true}
		ps120.OverlayValues = make([]scm.JITValueDesc, 63)
		ps120.OverlayValues[5] = d5
		ps120.OverlayValues[6] = d6
		ps120.OverlayValues[7] = d7
		ps120.OverlayValues[8] = d8
		ps120.OverlayValues[9] = d9
		ps120.OverlayValues[10] = d10
		ps120.OverlayValues[11] = d11
		ps120.OverlayValues[12] = d12
		ps120.OverlayValues[13] = d13
		ps120.OverlayValues[14] = d14
		ps120.OverlayValues[15] = d15
		ps120.OverlayValues[16] = d16
		ps120.OverlayValues[17] = d17
		ps120.OverlayValues[18] = d18
		ps120.OverlayValues[19] = d19
		ps120.OverlayValues[20] = d20
		ps120.OverlayValues[21] = d21
		ps120.OverlayValues[23] = d23
		ps120.OverlayValues[24] = d24
		ps120.OverlayValues[25] = d25
		ps120.OverlayValues[26] = d26
		ps120.OverlayValues[27] = d27
		ps120.OverlayValues[28] = d28
		ps120.OverlayValues[29] = d29
		ps120.OverlayValues[30] = d30
		ps120.OverlayValues[31] = d31
		ps120.OverlayValues[32] = d32
		ps120.OverlayValues[33] = d33
		ps120.OverlayValues[34] = d34
		ps120.OverlayValues[35] = d35
		ps120.OverlayValues[36] = d36
		ps120.OverlayValues[37] = d37
		ps120.OverlayValues[38] = d38
		ps120.OverlayValues[39] = d39
		ps120.OverlayValues[40] = d40
		ps120.OverlayValues[41] = d41
		ps120.OverlayValues[42] = d42
		ps120.OverlayValues[43] = d43
		ps120.OverlayValues[44] = d44
		ps120.OverlayValues[45] = d45
		ps120.OverlayValues[46] = d46
		ps120.OverlayValues[47] = d47
		ps120.OverlayValues[48] = d48
		ps120.OverlayValues[49] = d49
		ps120.OverlayValues[50] = d50
		ps120.OverlayValues[51] = d51
		ps120.OverlayValues[52] = d52
		ps120.OverlayValues[53] = d53
		ps120.OverlayValues[54] = d54
		ps120.OverlayValues[55] = d55
		ps120.OverlayValues[56] = d56
		ps120.OverlayValues[57] = d57
		ps120.OverlayValues[60] = d60
		ps120.OverlayValues[61] = d61
		ps120.OverlayValues[62] = d62
		snap121 := d5
		snap122 := d6
		snap123 := d7
		snap124 := d8
		snap125 := d9
		snap126 := d10
		snap127 := d11
		snap128 := d12
		snap129 := d13
		snap130 := d14
		snap131 := d15
		snap132 := d16
		snap133 := d17
		snap134 := d18
		snap135 := d19
		snap136 := d20
		snap137 := d21
		snap138 := d23
		snap139 := d24
		snap140 := d25
		snap141 := d26
		snap142 := d27
		snap143 := d28
		snap144 := d29
		snap145 := d30
		snap146 := d31
		snap147 := d32
		snap148 := d33
		snap149 := d34
		snap150 := d35
		snap151 := d36
		snap152 := d37
		snap153 := d38
		snap154 := d39
		snap155 := d40
		snap156 := d41
		snap157 := d42
		snap158 := d43
		snap159 := d44
		snap160 := d45
		snap161 := d46
		snap162 := d47
		snap163 := d48
		snap164 := d49
		snap165 := d50
		snap166 := d51
		snap167 := d52
		snap168 := d53
		snap169 := d54
		snap170 := d55
		snap171 := d56
		snap172 := d57
		snap173 := d60
		snap174 := d61
		snap175 := d62
		alloc176 := ctx.SnapshotAllocState()
		if !bbs[5].Rendered {
			bbs[5].RenderPS(ps120)
		}
		ctx.RestoreAllocState(alloc176)
		d5 = snap121
		d6 = snap122
		d7 = snap123
		d8 = snap124
		d9 = snap125
		d10 = snap126
		d11 = snap127
		d12 = snap128
		d13 = snap129
		d14 = snap130
		d15 = snap131
		d16 = snap132
		d17 = snap133
		d18 = snap134
		d19 = snap135
		d20 = snap136
		d21 = snap137
		d23 = snap138
		d24 = snap139
		d25 = snap140
		d26 = snap141
		d27 = snap142
		d28 = snap143
		d29 = snap144
		d30 = snap145
		d31 = snap146
		d32 = snap147
		d33 = snap148
		d34 = snap149
		d35 = snap150
		d36 = snap151
		d37 = snap152
		d38 = snap153
		d39 = snap154
		d40 = snap155
		d41 = snap156
		d42 = snap157
		d43 = snap158
		d44 = snap159
		d45 = snap160
		d46 = snap161
		d47 = snap162
		d48 = snap163
		d49 = snap164
		d50 = snap165
		d51 = snap166
		d52 = snap167
		d53 = snap168
		d54 = snap169
		d55 = snap170
		d56 = snap171
		d57 = snap172
		d60 = snap173
		d61 = snap174
		d62 = snap175
		if !bbs[3].Rendered {
			return bbs[3].RenderPS(ps119)
		}
		return result
		return result
	}
	bbs[2].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d177 := ps.PhiValues[0]
				ctx.EnsureDesc(&d177)
				ctx.EmitStoreToStack(d177, int32(bbs[2].PhiBase)+int32(0))
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
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
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
		if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != scm.LocNone {
			d177 = ps.OverlayValues[177]
		}
		if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
			d8 = ps.PhiValues[0]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d8)
		ctx.EnsureDesc(&d8)
		ctx.EnsureDesc(&d8)
		var d178 scm.JITValueDesc
		if d8.Loc == scm.LocImm {
			d178 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(d8.Imm.Int()))))}
		} else {
			r40 := ctx.AllocReg()
			ctx.EmitMovRegReg(r40, d8.Reg)
			ctx.EmitShlRegImm8(r40, 32)
			ctx.EmitShrRegImm8(r40, 32)
			d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
			ctx.BindReg(r40, &d178)
		}
		ctx.EnsureDesc(&d178)
		if thisptr.Loc == scm.LocImm {
			baseReg := ctx.AllocReg()
			if d178.Loc == scm.LocReg {
				ctx.FreeReg(baseReg)
				baseReg = ctx.AllocRegExcept(d178.Reg)
			}
			ctx.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int())+unsafe.Offsetof((*StorageSeq)(nil).lastValue)))
			if d178.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d178.Imm.Int()))
				ctx.EmitStoreRegMem(scm.RegR11, baseReg, 0)
			} else {
				ctx.EmitStoreRegMem(d178.Reg, baseReg, 0)
			}
			ctx.FreeReg(baseReg)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
			if d178.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d178.Imm.Int()))
				ctx.EmitStoreRegMem(scm.RegR11, thisptr.Reg, off)
			} else {
				ctx.EmitStoreRegMem(d178.Reg, thisptr.Reg, off)
			}
		}
		ctx.FreeDesc(&d178)
		ctx.EnsureDesc(&d8)
		d179 = d8
		_ = d179
		ctx.StabilizeDescForControlFlow(&d179)
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
		ctx.EnsureDesc(&d179)
		ctx.EnsureDesc(&d179)
		var d180 scm.JITValueDesc
		if d179.Loc == scm.LocImm {
			d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d179.Imm.Int()))))}
		} else {
			r41 := ctx.AllocReg()
			ctx.EmitMovRegReg(r41, d179.Reg)
			ctx.EmitShlRegImm8(r41, 32)
			ctx.EmitShrRegImm8(r41, 32)
			d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
			ctx.BindReg(r41, &d180)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d181 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d181 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 48)
			r42 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r42, thisptr.Reg, off)
			d181 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r42}
			ctx.BindReg(r42, &d181)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d181)
		ctx.EnsureDesc(&d181)
		var d182 scm.JITValueDesc
		if d181.Loc == scm.LocImm {
			d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d181.Imm.Int()))))}
		} else {
			r43 := ctx.AllocReg()
			ctx.EmitMovRegReg(r43, d181.Reg)
			ctx.EmitShlRegImm8(r43, 56)
			ctx.EmitShrRegImm8(r43, 56)
			d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
			ctx.BindReg(r43, &d182)
		}
		ctx.FreeDesc(&d181)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d180)
		ctx.EnsureDesc(&d182)
		ctx.EnsureDescsTogether(&d180, &d182)
		var d183 scm.JITValueDesc
		if d180.Loc == scm.LocImm && d182.Loc == scm.LocImm {
			d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d180.Imm.Int() * d182.Imm.Int())}
		} else if d180.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d182.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d180.Imm.Int()))
			ctx.EmitImulInt64(scratch, d182.Reg)
			d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d183)
		} else if d182.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d180.Reg)
			ctx.EmitMovRegReg(scratch, d180.Reg)
			if d182.Imm.Int() >= -2147483648 && d182.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d182.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d182.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d183)
		} else {
			r44 := ctx.AllocRegExcept(d180.Reg, d182.Reg)
			ctx.EmitMovRegReg(r44, d180.Reg)
			ctx.EmitImulInt64(r44, d182.Reg)
			d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
			ctx.BindReg(r44, &d183)
		}
		if d183.Loc == scm.LocReg && d180.Loc == scm.LocReg && d183.Reg == d180.Reg {
			ctx.TransferReg(d180.Reg)
			d180.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d180)
		ctx.FreeDesc(&d182)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d183)
		var d184 scm.JITValueDesc
		if d183.Loc == scm.LocImm {
			d184 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d183.Imm.Int() / 64)}
		} else {
			r45 := ctx.AllocRegExcept(d183.Reg)
			ctx.EmitMovRegReg(r45, d183.Reg)
			ctx.EmitShrRegImm8(r45, 6)
			d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
			ctx.BindReg(r45, &d184)
		}
		if d184.Loc == scm.LocReg && d183.Loc == scm.LocReg && d184.Reg == d183.Reg {
			ctx.TransferReg(d183.Reg)
			d183.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d183)
		var d185 scm.JITValueDesc
		if d183.Loc == scm.LocImm {
			d185 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d183.Imm.Int() % 64)}
		} else {
			r46 := ctx.AllocRegExcept(d183.Reg)
			ctx.EmitMovRegReg(r46, d183.Reg)
			ctx.EmitAndRegImm32(r46, 63)
			d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
			ctx.BindReg(r46, &d185)
		}
		if d185.Loc == scm.LocReg && d183.Loc == scm.LocReg && d185.Reg == d183.Reg {
			ctx.TransferReg(d183.Reg)
			d183.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d183)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d186 scm.JITValueDesc
		r47 := ctx.AllocReg()
		r48 := ctx.AllocRegExcept(r47)
		r49 := ctx.AllocRegExcept(r47, r48)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			ctx.EmitMovRegImm64(r47, uint64(dataPtr))
			ctx.EmitMovRegImm64(r48, uint64(sliceLen))
			ctx.EmitMovRegImm64(r49, uint64(sliceCap))
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
			ctx.EmitMovRegMem(r47, thisptr.Reg, off)
			ctx.EmitMovRegMem(r48, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r49, thisptr.Reg, off+16)
		}
		d186 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r47, Reg2: r48, Reg3: r49}
		ctx.BindReg(r47, &d186)
		ctx.BindReg(r48, &d186)
		ctx.BindReg(r49, &d186)
		ctx.BindReg(r47, &d186)
		ctx.BindReg(r48, &d186)
		ctx.BindReg(r49, &d186)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d184)
		ctx.ReclaimUntrackedRegs()
		d188 = ctx.EmitSliceElementAddress(&d186, &d184, 8)
		ctx.EnsureDesc(&d188)
		ctx.EmitMovRegMem(d188.Reg, d188.Reg, 0)
		d187 = d188
		d187.Type = scm.TagInt
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d187)
		ctx.EnsureDesc(&d185)
		ctx.EnsureDescsTogether(&d187, &d185)
		var d189 scm.JITValueDesc
		if d187.Loc == scm.LocImm && d185.Loc == scm.LocImm {
			d189 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d187.Imm.Int()) << uint64(d185.Imm.Int())))}
		} else if d185.Loc == scm.LocImm {
			r50 := ctx.AllocRegExcept(d187.Reg)
			ctx.EmitMovRegReg(r50, d187.Reg)
			ctx.EmitShlRegImm8(r50, uint8(d185.Imm.Int()))
			d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
			ctx.BindReg(r50, &d189)
		} else {
			{
				shiftSrc := d187.Reg
				r51 := ctx.AllocRegExcept(d187.Reg, d185.Reg)
				ctx.EmitMovRegReg(r51, d187.Reg)
				shiftSrc = r51
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d185.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d185.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d185.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d189)
			}
		}
		if d189.Loc == scm.LocReg && d187.Loc == scm.LocReg && d189.Reg == d187.Reg {
			ctx.TransferReg(d187.Reg)
			d187.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d187)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d184)
		ctx.EnsureDesc(&d184)
		var d190 scm.JITValueDesc
		if d184.Loc == scm.LocImm {
			d190 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d184.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d184.Reg)
			ctx.EmitMovRegReg(scratch, d184.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d190)
		}
		if d190.Loc == scm.LocReg && d184.Loc == scm.LocReg && d190.Reg == d184.Reg {
			ctx.TransferReg(d184.Reg)
			d184.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d184)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d190)
		ctx.ReclaimUntrackedRegs()
		d192 = ctx.EmitSliceElementAddress(&d186, &d190, 8)
		ctx.EnsureDesc(&d192)
		ctx.EmitMovRegMem(d192.Reg, d192.Reg, 0)
		d191 = d192
		d191.Type = scm.TagInt
		ctx.FreeDesc(&d190)
		ctx.ReclaimUntrackedRegs()
		d193 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d185)
		ctx.EnsureDescsTogether(&d193, &d185)
		var d194 scm.JITValueDesc
		if d193.Loc == scm.LocImm && d185.Loc == scm.LocImm {
			d194 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d193.Imm.Int() - d185.Imm.Int())}
		} else if d185.Loc == scm.LocImm && d185.Imm.Int() == 0 {
			r52 := ctx.AllocRegExcept(d193.Reg)
			ctx.EmitMovRegReg(r52, d193.Reg)
			d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
			ctx.BindReg(r52, &d194)
		} else if d193.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d185.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d193.Imm.Int()))
			ctx.EmitSubInt64(scratch, d185.Reg)
			d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d194)
		} else if d185.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d193.Reg)
			ctx.EmitMovRegReg(scratch, d193.Reg)
			if d185.Imm.Int() >= -2147483648 && d185.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d185.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d185.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d194)
		} else {
			r53 := ctx.AllocRegExcept(d193.Reg, d185.Reg)
			ctx.EmitMovRegReg(r53, d193.Reg)
			ctx.EmitSubInt64(r53, d185.Reg)
			d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
			ctx.BindReg(r53, &d194)
		}
		if d194.Loc == scm.LocReg && d193.Loc == scm.LocReg && d194.Reg == d193.Reg {
			ctx.TransferReg(d193.Reg)
			d193.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d185)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d191)
		ctx.EnsureDesc(&d194)
		ctx.EnsureDescsTogether(&d191, &d194)
		var d195 scm.JITValueDesc
		if d191.Loc == scm.LocImm && d194.Loc == scm.LocImm {
			d195 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d191.Imm.Int()) >> uint64(d194.Imm.Int())))}
		} else if d194.Loc == scm.LocImm {
			r54 := ctx.AllocRegExcept(d191.Reg)
			ctx.EmitMovRegReg(r54, d191.Reg)
			ctx.EmitShrRegImm8(r54, uint8(d194.Imm.Int()))
			d195 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
			ctx.BindReg(r54, &d195)
		} else {
			{
				shiftSrc := d191.Reg
				r55 := ctx.AllocRegExcept(d191.Reg, d194.Reg)
				ctx.EmitMovRegReg(r55, d191.Reg)
				shiftSrc = r55
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d194.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d194.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d194.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d195 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d195)
			}
		}
		if d195.Loc == scm.LocReg && d191.Loc == scm.LocReg && d195.Reg == d191.Reg {
			ctx.TransferReg(d191.Reg)
			d191.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d191)
		ctx.FreeDesc(&d194)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d189)
		ctx.EnsureDesc(&d195)
		var d196 scm.JITValueDesc
		if d189.Loc == scm.LocImm && d195.Loc == scm.LocImm {
			d196 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d189.Imm.Int() | d195.Imm.Int())}
		} else if d189.Loc == scm.LocImm && d189.Imm.Int() == 0 {
			d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d195.Reg}
			ctx.BindReg(d195.Reg, &d196)
		} else if d195.Loc == scm.LocImm && d195.Imm.Int() == 0 {
			r56 := ctx.AllocRegExcept(d189.Reg)
			ctx.EmitMovRegReg(r56, d189.Reg)
			d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r56}
			ctx.BindReg(r56, &d196)
		} else if d189.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d195.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d189.Imm.Int()))
			ctx.EmitOrInt64(scratch, d195.Reg)
			d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d196)
		} else if d195.Loc == scm.LocImm {
			r57 := ctx.AllocRegExcept(d189.Reg)
			ctx.EmitMovRegReg(r57, d189.Reg)
			if d195.Imm.Int() >= -2147483648 && d195.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r57, int32(d195.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d195.Imm.Int()))
				ctx.EmitOrInt64(r57, scm.RegR11)
			}
			d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r57}
			ctx.BindReg(r57, &d196)
		} else {
			r58 := ctx.AllocRegExcept(d189.Reg, d195.Reg)
			ctx.EmitMovRegReg(r58, d189.Reg)
			ctx.EmitOrInt64(r58, d195.Reg)
			d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r58}
			ctx.BindReg(r58, &d196)
		}
		if d196.Loc == scm.LocReg && d189.Loc == scm.LocReg && d196.Reg == d189.Reg {
			ctx.TransferReg(d189.Reg)
			d189.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d189)
		ctx.FreeDesc(&d195)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d197 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d197 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 48)
			r59 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r59, thisptr.Reg, off)
			d197 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r59}
			ctx.BindReg(r59, &d197)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d197)
		ctx.EnsureDesc(&d197)
		var d198 scm.JITValueDesc
		if d197.Loc == scm.LocImm {
			d198 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d197.Imm.Int()))))}
		} else {
			r60 := ctx.AllocReg()
			ctx.EmitMovRegReg(r60, d197.Reg)
			ctx.EmitShlRegImm8(r60, 56)
			ctx.EmitShrRegImm8(r60, 56)
			d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r60}
			ctx.BindReg(r60, &d198)
		}
		ctx.FreeDesc(&d197)
		ctx.ReclaimUntrackedRegs()
		d199 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d198)
		ctx.EnsureDescsTogether(&d199, &d198)
		var d200 scm.JITValueDesc
		if d199.Loc == scm.LocImm && d198.Loc == scm.LocImm {
			d200 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d199.Imm.Int() - d198.Imm.Int())}
		} else if d198.Loc == scm.LocImm && d198.Imm.Int() == 0 {
			r61 := ctx.AllocRegExcept(d199.Reg)
			ctx.EmitMovRegReg(r61, d199.Reg)
			d200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r61}
			ctx.BindReg(r61, &d200)
		} else if d199.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d198.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d199.Imm.Int()))
			ctx.EmitSubInt64(scratch, d198.Reg)
			d200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d200)
		} else if d198.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d199.Reg)
			ctx.EmitMovRegReg(scratch, d199.Reg)
			if d198.Imm.Int() >= -2147483648 && d198.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d198.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d198.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d200)
		} else {
			r62 := ctx.AllocRegExcept(d199.Reg, d198.Reg)
			ctx.EmitMovRegReg(r62, d199.Reg)
			ctx.EmitSubInt64(r62, d198.Reg)
			d200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r62}
			ctx.BindReg(r62, &d200)
		}
		if d200.Loc == scm.LocReg && d199.Loc == scm.LocReg && d200.Reg == d199.Reg {
			ctx.TransferReg(d199.Reg)
			d199.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d198)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d196)
		ctx.EnsureDesc(&d200)
		ctx.EnsureDescsTogether(&d196, &d200)
		var d201 scm.JITValueDesc
		if d196.Loc == scm.LocImm && d200.Loc == scm.LocImm {
			d201 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d196.Imm.Int()) >> uint64(d200.Imm.Int())))}
		} else if d200.Loc == scm.LocImm {
			r63 := ctx.AllocRegExcept(d196.Reg)
			ctx.EmitMovRegReg(r63, d196.Reg)
			ctx.EmitShrRegImm8(r63, uint8(d200.Imm.Int()))
			d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r63}
			ctx.BindReg(r63, &d201)
		} else {
			{
				shiftSrc := d196.Reg
				r64 := ctx.AllocRegExcept(d196.Reg, d200.Reg)
				ctx.EmitMovRegReg(r64, d196.Reg)
				shiftSrc = r64
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d200.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d200.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d200.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d201)
			}
		}
		if d201.Loc == scm.LocReg && d196.Loc == scm.LocReg && d201.Reg == d196.Reg {
			ctx.TransferReg(d196.Reg)
			d196.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d196)
		ctx.FreeDesc(&d200)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d201)
		ctx.EnsureDesc(&d201)
		ctx.EnsureDesc(&d201)
		var d202 scm.JITValueDesc
		if d201.Loc == scm.LocImm {
			d202 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d201.Imm.Int()))))}
		} else {
			r65 := ctx.AllocReg()
			ctx.EmitMovRegReg(r65, d201.Reg)
			d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
			ctx.BindReg(r65, &d202)
		}
		ctx.FreeDesc(&d201)
		var d203 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d203 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 56)
			r66 := ctx.AllocReg()
			ctx.EmitMovRegMem(r66, thisptr.Reg, off)
			d203 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r66}
			ctx.BindReg(r66, &d203)
		}
		ctx.EnsureDesc(&d202)
		ctx.EnsureDesc(&d203)
		ctx.EnsureDescsTogether(&d202, &d203)
		var d204 scm.JITValueDesc
		if d202.Loc == scm.LocImm && d203.Loc == scm.LocImm {
			d204 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d202.Imm.Int() + d203.Imm.Int())}
		} else if d203.Loc == scm.LocImm && d203.Imm.Int() == 0 {
			r67 := ctx.AllocRegExcept(d202.Reg)
			ctx.EmitMovRegReg(r67, d202.Reg)
			d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r67}
			ctx.BindReg(r67, &d204)
		} else if d202.Loc == scm.LocImm && d202.Imm.Int() == 0 {
			d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d203.Reg}
			ctx.BindReg(d203.Reg, &d204)
		} else if d202.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d203.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d202.Imm.Int()))
			ctx.EmitAddInt64(scratch, d203.Reg)
			d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d204)
		} else if d203.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d202.Reg)
			ctx.EmitMovRegReg(scratch, d202.Reg)
			if d203.Imm.Int() >= -2147483648 && d203.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d203.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d203.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d204)
		} else {
			r68 := ctx.AllocRegExcept(d202.Reg, d203.Reg)
			ctx.EmitMovRegReg(r68, d202.Reg)
			ctx.EmitAddInt64(r68, d203.Reg)
			d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r68}
			ctx.BindReg(r68, &d204)
		}
		if d204.Loc == scm.LocReg && d202.Loc == scm.LocReg && d204.Reg == d202.Reg {
			ctx.TransferReg(d202.Reg)
			d202.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d204)
		ctx.FreeDesc(&d202)
		ctx.FreeDesc(&d203)
		var d205 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 80
			val := *(*bool)(unsafe.Pointer(fieldAddr))
			d205 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 80)
			r69 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r69, thisptr.Reg, off)
			d205 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r69}
			ctx.BindReg(r69, &d205)
		}
		d206 = d205
		ctx.EnsureDesc(&d206)
		if d206.Loc != scm.LocImm && d206.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d206.Loc == scm.LocImm {
			if d206.Imm.Bool() {
				if ps.General {
				}
				ps207 := scm.PhiState{General: ps.General}
				ps207.OverlayValues = make([]scm.JITValueDesc, 207)
				ps207.OverlayValues[5] = d5
				ps207.OverlayValues[6] = d6
				ps207.OverlayValues[7] = d7
				ps207.OverlayValues[8] = d8
				ps207.OverlayValues[9] = d9
				ps207.OverlayValues[10] = d10
				ps207.OverlayValues[11] = d11
				ps207.OverlayValues[12] = d12
				ps207.OverlayValues[13] = d13
				ps207.OverlayValues[14] = d14
				ps207.OverlayValues[15] = d15
				ps207.OverlayValues[16] = d16
				ps207.OverlayValues[17] = d17
				ps207.OverlayValues[18] = d18
				ps207.OverlayValues[19] = d19
				ps207.OverlayValues[20] = d20
				ps207.OverlayValues[21] = d21
				ps207.OverlayValues[23] = d23
				ps207.OverlayValues[24] = d24
				ps207.OverlayValues[25] = d25
				ps207.OverlayValues[26] = d26
				ps207.OverlayValues[27] = d27
				ps207.OverlayValues[28] = d28
				ps207.OverlayValues[29] = d29
				ps207.OverlayValues[30] = d30
				ps207.OverlayValues[31] = d31
				ps207.OverlayValues[32] = d32
				ps207.OverlayValues[33] = d33
				ps207.OverlayValues[34] = d34
				ps207.OverlayValues[35] = d35
				ps207.OverlayValues[36] = d36
				ps207.OverlayValues[37] = d37
				ps207.OverlayValues[38] = d38
				ps207.OverlayValues[39] = d39
				ps207.OverlayValues[40] = d40
				ps207.OverlayValues[41] = d41
				ps207.OverlayValues[42] = d42
				ps207.OverlayValues[43] = d43
				ps207.OverlayValues[44] = d44
				ps207.OverlayValues[45] = d45
				ps207.OverlayValues[46] = d46
				ps207.OverlayValues[47] = d47
				ps207.OverlayValues[48] = d48
				ps207.OverlayValues[49] = d49
				ps207.OverlayValues[50] = d50
				ps207.OverlayValues[51] = d51
				ps207.OverlayValues[52] = d52
				ps207.OverlayValues[53] = d53
				ps207.OverlayValues[54] = d54
				ps207.OverlayValues[55] = d55
				ps207.OverlayValues[56] = d56
				ps207.OverlayValues[57] = d57
				ps207.OverlayValues[60] = d60
				ps207.OverlayValues[61] = d61
				ps207.OverlayValues[62] = d62
				ps207.OverlayValues[177] = d177
				ps207.OverlayValues[178] = d178
				ps207.OverlayValues[179] = d179
				ps207.OverlayValues[180] = d180
				ps207.OverlayValues[181] = d181
				ps207.OverlayValues[182] = d182
				ps207.OverlayValues[183] = d183
				ps207.OverlayValues[184] = d184
				ps207.OverlayValues[185] = d185
				ps207.OverlayValues[186] = d186
				ps207.OverlayValues[187] = d187
				ps207.OverlayValues[188] = d188
				ps207.OverlayValues[189] = d189
				ps207.OverlayValues[190] = d190
				ps207.OverlayValues[191] = d191
				ps207.OverlayValues[192] = d192
				ps207.OverlayValues[193] = d193
				ps207.OverlayValues[194] = d194
				ps207.OverlayValues[195] = d195
				ps207.OverlayValues[196] = d196
				ps207.OverlayValues[197] = d197
				ps207.OverlayValues[198] = d198
				ps207.OverlayValues[199] = d199
				ps207.OverlayValues[200] = d200
				ps207.OverlayValues[201] = d201
				ps207.OverlayValues[202] = d202
				ps207.OverlayValues[203] = d203
				ps207.OverlayValues[204] = d204
				ps207.OverlayValues[205] = d205
				ps207.OverlayValues[206] = d206
				return bbs[13].RenderPS(ps207)
			}
			if ps.General {
			}
			ps208 := scm.PhiState{General: ps.General}
			ps208.OverlayValues = make([]scm.JITValueDesc, 207)
			ps208.OverlayValues[5] = d5
			ps208.OverlayValues[6] = d6
			ps208.OverlayValues[7] = d7
			ps208.OverlayValues[8] = d8
			ps208.OverlayValues[9] = d9
			ps208.OverlayValues[10] = d10
			ps208.OverlayValues[11] = d11
			ps208.OverlayValues[12] = d12
			ps208.OverlayValues[13] = d13
			ps208.OverlayValues[14] = d14
			ps208.OverlayValues[15] = d15
			ps208.OverlayValues[16] = d16
			ps208.OverlayValues[17] = d17
			ps208.OverlayValues[18] = d18
			ps208.OverlayValues[19] = d19
			ps208.OverlayValues[20] = d20
			ps208.OverlayValues[21] = d21
			ps208.OverlayValues[23] = d23
			ps208.OverlayValues[24] = d24
			ps208.OverlayValues[25] = d25
			ps208.OverlayValues[26] = d26
			ps208.OverlayValues[27] = d27
			ps208.OverlayValues[28] = d28
			ps208.OverlayValues[29] = d29
			ps208.OverlayValues[30] = d30
			ps208.OverlayValues[31] = d31
			ps208.OverlayValues[32] = d32
			ps208.OverlayValues[33] = d33
			ps208.OverlayValues[34] = d34
			ps208.OverlayValues[35] = d35
			ps208.OverlayValues[36] = d36
			ps208.OverlayValues[37] = d37
			ps208.OverlayValues[38] = d38
			ps208.OverlayValues[39] = d39
			ps208.OverlayValues[40] = d40
			ps208.OverlayValues[41] = d41
			ps208.OverlayValues[42] = d42
			ps208.OverlayValues[43] = d43
			ps208.OverlayValues[44] = d44
			ps208.OverlayValues[45] = d45
			ps208.OverlayValues[46] = d46
			ps208.OverlayValues[47] = d47
			ps208.OverlayValues[48] = d48
			ps208.OverlayValues[49] = d49
			ps208.OverlayValues[50] = d50
			ps208.OverlayValues[51] = d51
			ps208.OverlayValues[52] = d52
			ps208.OverlayValues[53] = d53
			ps208.OverlayValues[54] = d54
			ps208.OverlayValues[55] = d55
			ps208.OverlayValues[56] = d56
			ps208.OverlayValues[57] = d57
			ps208.OverlayValues[60] = d60
			ps208.OverlayValues[61] = d61
			ps208.OverlayValues[62] = d62
			ps208.OverlayValues[177] = d177
			ps208.OverlayValues[178] = d178
			ps208.OverlayValues[179] = d179
			ps208.OverlayValues[180] = d180
			ps208.OverlayValues[181] = d181
			ps208.OverlayValues[182] = d182
			ps208.OverlayValues[183] = d183
			ps208.OverlayValues[184] = d184
			ps208.OverlayValues[185] = d185
			ps208.OverlayValues[186] = d186
			ps208.OverlayValues[187] = d187
			ps208.OverlayValues[188] = d188
			ps208.OverlayValues[189] = d189
			ps208.OverlayValues[190] = d190
			ps208.OverlayValues[191] = d191
			ps208.OverlayValues[192] = d192
			ps208.OverlayValues[193] = d193
			ps208.OverlayValues[194] = d194
			ps208.OverlayValues[195] = d195
			ps208.OverlayValues[196] = d196
			ps208.OverlayValues[197] = d197
			ps208.OverlayValues[198] = d198
			ps208.OverlayValues[199] = d199
			ps208.OverlayValues[200] = d200
			ps208.OverlayValues[201] = d201
			ps208.OverlayValues[202] = d202
			ps208.OverlayValues[203] = d203
			ps208.OverlayValues[204] = d204
			ps208.OverlayValues[205] = d205
			ps208.OverlayValues[206] = d206
			return bbs[12].RenderPS(ps208)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d209 := ps.PhiValues[0]
				ctx.EnsureDesc(&d209)
				ctx.EmitStoreToStack(d209, int32(bbs[2].PhiBase)+int32(0))
			}
			ps.General = true
			return bbs[2].RenderPS(ps)
		}
		lbl19 := ctx.ReserveLabel()
		lbl20 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d206.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl19)
		ctx.EmitJmp(lbl20)
		snap210 := d5
		snap211 := d6
		snap212 := d7
		snap213 := d8
		snap214 := d9
		snap215 := d10
		snap216 := d11
		snap217 := d12
		snap218 := d13
		snap219 := d14
		snap220 := d15
		snap221 := d16
		snap222 := d17
		snap223 := d18
		snap224 := d19
		snap225 := d20
		snap226 := d21
		snap227 := d23
		snap228 := d24
		snap229 := d25
		snap230 := d26
		snap231 := d27
		snap232 := d28
		snap233 := d29
		snap234 := d30
		snap235 := d31
		snap236 := d32
		snap237 := d33
		snap238 := d34
		snap239 := d35
		snap240 := d36
		snap241 := d37
		snap242 := d38
		snap243 := d39
		snap244 := d40
		snap245 := d41
		snap246 := d42
		snap247 := d43
		snap248 := d44
		snap249 := d45
		snap250 := d46
		snap251 := d47
		snap252 := d48
		snap253 := d49
		snap254 := d50
		snap255 := d51
		snap256 := d52
		snap257 := d53
		snap258 := d54
		snap259 := d55
		snap260 := d56
		snap261 := d57
		snap262 := d60
		snap263 := d61
		snap264 := d62
		snap265 := d177
		snap266 := d178
		snap267 := d179
		snap268 := d180
		snap269 := d181
		snap270 := d182
		snap271 := d183
		snap272 := d184
		snap273 := d185
		snap274 := d186
		snap275 := d187
		snap276 := d188
		snap277 := d189
		snap278 := d190
		snap279 := d191
		snap280 := d192
		snap281 := d193
		snap282 := d194
		snap283 := d195
		snap284 := d196
		snap285 := d197
		snap286 := d198
		snap287 := d199
		snap288 := d200
		snap289 := d201
		snap290 := d202
		snap291 := d203
		snap292 := d204
		snap293 := d205
		snap294 := d206
		snap295 := d209
		alloc296 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl19)
		ctx.EmitJmp(lbl14)
		ctx.RestoreAllocState(alloc296)
		d5 = snap210
		d6 = snap211
		d7 = snap212
		d8 = snap213
		d9 = snap214
		d10 = snap215
		d11 = snap216
		d12 = snap217
		d13 = snap218
		d14 = snap219
		d15 = snap220
		d16 = snap221
		d17 = snap222
		d18 = snap223
		d19 = snap224
		d20 = snap225
		d21 = snap226
		d23 = snap227
		d24 = snap228
		d25 = snap229
		d26 = snap230
		d27 = snap231
		d28 = snap232
		d29 = snap233
		d30 = snap234
		d31 = snap235
		d32 = snap236
		d33 = snap237
		d34 = snap238
		d35 = snap239
		d36 = snap240
		d37 = snap241
		d38 = snap242
		d39 = snap243
		d40 = snap244
		d41 = snap245
		d42 = snap246
		d43 = snap247
		d44 = snap248
		d45 = snap249
		d46 = snap250
		d47 = snap251
		d48 = snap252
		d49 = snap253
		d50 = snap254
		d51 = snap255
		d52 = snap256
		d53 = snap257
		d54 = snap258
		d55 = snap259
		d56 = snap260
		d57 = snap261
		d60 = snap262
		d61 = snap263
		d62 = snap264
		d177 = snap265
		d178 = snap266
		d179 = snap267
		d180 = snap268
		d181 = snap269
		d182 = snap270
		d183 = snap271
		d184 = snap272
		d185 = snap273
		d186 = snap274
		d187 = snap275
		d188 = snap276
		d189 = snap277
		d190 = snap278
		d191 = snap279
		d192 = snap280
		d193 = snap281
		d194 = snap282
		d195 = snap283
		d196 = snap284
		d197 = snap285
		d198 = snap286
		d199 = snap287
		d200 = snap288
		d201 = snap289
		d202 = snap290
		d203 = snap291
		d204 = snap292
		d205 = snap293
		d206 = snap294
		d209 = snap295
		ctx.MarkLabel(lbl20)
		ctx.EmitJmp(lbl13)
		ctx.RestoreAllocState(alloc296)
		d5 = snap210
		d6 = snap211
		d7 = snap212
		d8 = snap213
		d9 = snap214
		d10 = snap215
		d11 = snap216
		d12 = snap217
		d13 = snap218
		d14 = snap219
		d15 = snap220
		d16 = snap221
		d17 = snap222
		d18 = snap223
		d19 = snap224
		d20 = snap225
		d21 = snap226
		d23 = snap227
		d24 = snap228
		d25 = snap229
		d26 = snap230
		d27 = snap231
		d28 = snap232
		d29 = snap233
		d30 = snap234
		d31 = snap235
		d32 = snap236
		d33 = snap237
		d34 = snap238
		d35 = snap239
		d36 = snap240
		d37 = snap241
		d38 = snap242
		d39 = snap243
		d40 = snap244
		d41 = snap245
		d42 = snap246
		d43 = snap247
		d44 = snap248
		d45 = snap249
		d46 = snap250
		d47 = snap251
		d48 = snap252
		d49 = snap253
		d50 = snap254
		d51 = snap255
		d52 = snap256
		d53 = snap257
		d54 = snap258
		d55 = snap259
		d56 = snap260
		d57 = snap261
		d60 = snap262
		d61 = snap263
		d62 = snap264
		d177 = snap265
		d178 = snap266
		d179 = snap267
		d180 = snap268
		d181 = snap269
		d182 = snap270
		d183 = snap271
		d184 = snap272
		d185 = snap273
		d186 = snap274
		d187 = snap275
		d188 = snap276
		d189 = snap277
		d190 = snap278
		d191 = snap279
		d192 = snap280
		d193 = snap281
		d194 = snap282
		d195 = snap283
		d196 = snap284
		d197 = snap285
		d198 = snap286
		d199 = snap287
		d200 = snap288
		d201 = snap289
		d202 = snap290
		d203 = snap291
		d204 = snap292
		d205 = snap293
		d206 = snap294
		d209 = snap295
		ps297 := scm.PhiState{General: true}
		ps297.OverlayValues = make([]scm.JITValueDesc, 210)
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
		ps297.OverlayValues[16] = d16
		ps297.OverlayValues[17] = d17
		ps297.OverlayValues[18] = d18
		ps297.OverlayValues[19] = d19
		ps297.OverlayValues[20] = d20
		ps297.OverlayValues[21] = d21
		ps297.OverlayValues[23] = d23
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
		ps297.OverlayValues[60] = d60
		ps297.OverlayValues[61] = d61
		ps297.OverlayValues[62] = d62
		ps297.OverlayValues[177] = d177
		ps297.OverlayValues[178] = d178
		ps297.OverlayValues[179] = d179
		ps297.OverlayValues[180] = d180
		ps297.OverlayValues[181] = d181
		ps297.OverlayValues[182] = d182
		ps297.OverlayValues[183] = d183
		ps297.OverlayValues[184] = d184
		ps297.OverlayValues[185] = d185
		ps297.OverlayValues[186] = d186
		ps297.OverlayValues[187] = d187
		ps297.OverlayValues[188] = d188
		ps297.OverlayValues[189] = d189
		ps297.OverlayValues[190] = d190
		ps297.OverlayValues[191] = d191
		ps297.OverlayValues[192] = d192
		ps297.OverlayValues[193] = d193
		ps297.OverlayValues[194] = d194
		ps297.OverlayValues[195] = d195
		ps297.OverlayValues[196] = d196
		ps297.OverlayValues[197] = d197
		ps297.OverlayValues[198] = d198
		ps297.OverlayValues[199] = d199
		ps297.OverlayValues[200] = d200
		ps297.OverlayValues[201] = d201
		ps297.OverlayValues[202] = d202
		ps297.OverlayValues[203] = d203
		ps297.OverlayValues[204] = d204
		ps297.OverlayValues[205] = d205
		ps297.OverlayValues[206] = d206
		ps297.OverlayValues[209] = d209
		ps298 := scm.PhiState{General: true}
		ps298.OverlayValues = make([]scm.JITValueDesc, 210)
		ps298.OverlayValues[5] = d5
		ps298.OverlayValues[6] = d6
		ps298.OverlayValues[7] = d7
		ps298.OverlayValues[8] = d8
		ps298.OverlayValues[9] = d9
		ps298.OverlayValues[10] = d10
		ps298.OverlayValues[11] = d11
		ps298.OverlayValues[12] = d12
		ps298.OverlayValues[13] = d13
		ps298.OverlayValues[14] = d14
		ps298.OverlayValues[15] = d15
		ps298.OverlayValues[16] = d16
		ps298.OverlayValues[17] = d17
		ps298.OverlayValues[18] = d18
		ps298.OverlayValues[19] = d19
		ps298.OverlayValues[20] = d20
		ps298.OverlayValues[21] = d21
		ps298.OverlayValues[23] = d23
		ps298.OverlayValues[24] = d24
		ps298.OverlayValues[25] = d25
		ps298.OverlayValues[26] = d26
		ps298.OverlayValues[27] = d27
		ps298.OverlayValues[28] = d28
		ps298.OverlayValues[29] = d29
		ps298.OverlayValues[30] = d30
		ps298.OverlayValues[31] = d31
		ps298.OverlayValues[32] = d32
		ps298.OverlayValues[33] = d33
		ps298.OverlayValues[34] = d34
		ps298.OverlayValues[35] = d35
		ps298.OverlayValues[36] = d36
		ps298.OverlayValues[37] = d37
		ps298.OverlayValues[38] = d38
		ps298.OverlayValues[39] = d39
		ps298.OverlayValues[40] = d40
		ps298.OverlayValues[41] = d41
		ps298.OverlayValues[42] = d42
		ps298.OverlayValues[43] = d43
		ps298.OverlayValues[44] = d44
		ps298.OverlayValues[45] = d45
		ps298.OverlayValues[46] = d46
		ps298.OverlayValues[47] = d47
		ps298.OverlayValues[48] = d48
		ps298.OverlayValues[49] = d49
		ps298.OverlayValues[50] = d50
		ps298.OverlayValues[51] = d51
		ps298.OverlayValues[52] = d52
		ps298.OverlayValues[53] = d53
		ps298.OverlayValues[54] = d54
		ps298.OverlayValues[55] = d55
		ps298.OverlayValues[56] = d56
		ps298.OverlayValues[57] = d57
		ps298.OverlayValues[60] = d60
		ps298.OverlayValues[61] = d61
		ps298.OverlayValues[62] = d62
		ps298.OverlayValues[177] = d177
		ps298.OverlayValues[178] = d178
		ps298.OverlayValues[179] = d179
		ps298.OverlayValues[180] = d180
		ps298.OverlayValues[181] = d181
		ps298.OverlayValues[182] = d182
		ps298.OverlayValues[183] = d183
		ps298.OverlayValues[184] = d184
		ps298.OverlayValues[185] = d185
		ps298.OverlayValues[186] = d186
		ps298.OverlayValues[187] = d187
		ps298.OverlayValues[188] = d188
		ps298.OverlayValues[189] = d189
		ps298.OverlayValues[190] = d190
		ps298.OverlayValues[191] = d191
		ps298.OverlayValues[192] = d192
		ps298.OverlayValues[193] = d193
		ps298.OverlayValues[194] = d194
		ps298.OverlayValues[195] = d195
		ps298.OverlayValues[196] = d196
		ps298.OverlayValues[197] = d197
		ps298.OverlayValues[198] = d198
		ps298.OverlayValues[199] = d199
		ps298.OverlayValues[200] = d200
		ps298.OverlayValues[201] = d201
		ps298.OverlayValues[202] = d202
		ps298.OverlayValues[203] = d203
		ps298.OverlayValues[204] = d204
		ps298.OverlayValues[205] = d205
		ps298.OverlayValues[206] = d206
		ps298.OverlayValues[209] = d209
		snap299 := d5
		snap300 := d6
		snap301 := d7
		snap302 := d8
		snap303 := d9
		snap304 := d10
		snap305 := d11
		snap306 := d12
		snap307 := d13
		snap308 := d14
		snap309 := d15
		snap310 := d16
		snap311 := d17
		snap312 := d18
		snap313 := d19
		snap314 := d20
		snap315 := d21
		snap316 := d23
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
		snap351 := d60
		snap352 := d61
		snap353 := d62
		snap354 := d177
		snap355 := d178
		snap356 := d179
		snap357 := d180
		snap358 := d181
		snap359 := d182
		snap360 := d183
		snap361 := d184
		snap362 := d185
		snap363 := d186
		snap364 := d187
		snap365 := d188
		snap366 := d189
		snap367 := d190
		snap368 := d191
		snap369 := d192
		snap370 := d193
		snap371 := d194
		snap372 := d195
		snap373 := d196
		snap374 := d197
		snap375 := d198
		snap376 := d199
		snap377 := d200
		snap378 := d201
		snap379 := d202
		snap380 := d203
		snap381 := d204
		snap382 := d205
		snap383 := d206
		snap384 := d209
		alloc385 := ctx.SnapshotAllocState()
		if !bbs[12].Rendered {
			bbs[12].RenderPS(ps298)
		}
		ctx.RestoreAllocState(alloc385)
		d5 = snap299
		d6 = snap300
		d7 = snap301
		d8 = snap302
		d9 = snap303
		d10 = snap304
		d11 = snap305
		d12 = snap306
		d13 = snap307
		d14 = snap308
		d15 = snap309
		d16 = snap310
		d17 = snap311
		d18 = snap312
		d19 = snap313
		d20 = snap314
		d21 = snap315
		d23 = snap316
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
		d60 = snap351
		d61 = snap352
		d62 = snap353
		d177 = snap354
		d178 = snap355
		d179 = snap356
		d180 = snap357
		d181 = snap358
		d182 = snap359
		d183 = snap360
		d184 = snap361
		d185 = snap362
		d186 = snap363
		d187 = snap364
		d188 = snap365
		d189 = snap366
		d190 = snap367
		d191 = snap368
		d192 = snap369
		d193 = snap370
		d194 = snap371
		d195 = snap372
		d196 = snap373
		d197 = snap374
		d198 = snap375
		d199 = snap376
		d200 = snap377
		d201 = snap378
		d202 = snap379
		d203 = snap380
		d204 = snap381
		d205 = snap382
		d206 = snap383
		d209 = snap384
		if !bbs[13].Rendered {
			return bbs[13].RenderPS(ps297)
		}
		return result
		ctx.FreeDesc(&d205)
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
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
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
		if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != scm.LocNone {
			d177 = ps.OverlayValues[177]
		}
		if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
			d178 = ps.OverlayValues[178]
		}
		if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != scm.LocNone {
			d179 = ps.OverlayValues[179]
		}
		if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != scm.LocNone {
			d180 = ps.OverlayValues[180]
		}
		if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != scm.LocNone {
			d181 = ps.OverlayValues[181]
		}
		if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != scm.LocNone {
			d182 = ps.OverlayValues[182]
		}
		if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != scm.LocNone {
			d183 = ps.OverlayValues[183]
		}
		if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
			d184 = ps.OverlayValues[184]
		}
		if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != scm.LocNone {
			d185 = ps.OverlayValues[185]
		}
		if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != scm.LocNone {
			d186 = ps.OverlayValues[186]
		}
		if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
			d187 = ps.OverlayValues[187]
		}
		if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != scm.LocNone {
			d188 = ps.OverlayValues[188]
		}
		if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != scm.LocNone {
			d189 = ps.OverlayValues[189]
		}
		if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != scm.LocNone {
			d190 = ps.OverlayValues[190]
		}
		if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != scm.LocNone {
			d191 = ps.OverlayValues[191]
		}
		if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != scm.LocNone {
			d192 = ps.OverlayValues[192]
		}
		if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != scm.LocNone {
			d193 = ps.OverlayValues[193]
		}
		if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != scm.LocNone {
			d194 = ps.OverlayValues[194]
		}
		if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != scm.LocNone {
			d195 = ps.OverlayValues[195]
		}
		if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
			d196 = ps.OverlayValues[196]
		}
		if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != scm.LocNone {
			d197 = ps.OverlayValues[197]
		}
		if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != scm.LocNone {
			d198 = ps.OverlayValues[198]
		}
		if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
			d199 = ps.OverlayValues[199]
		}
		if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
			d200 = ps.OverlayValues[200]
		}
		if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
			d201 = ps.OverlayValues[201]
		}
		if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != scm.LocNone {
			d202 = ps.OverlayValues[202]
		}
		if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != scm.LocNone {
			d203 = ps.OverlayValues[203]
		}
		if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != scm.LocNone {
			d204 = ps.OverlayValues[204]
		}
		if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != scm.LocNone {
			d205 = ps.OverlayValues[205]
		}
		if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != scm.LocNone {
			d206 = ps.OverlayValues[206]
		}
		if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != scm.LocNone {
			d209 = ps.OverlayValues[209]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d5)
		var d386 scm.JITValueDesc
		if d5.Loc == scm.LocImm {
			d386 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d386 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d386)
		}
		if d386.Loc == scm.LocImm {
			d386 = scm.JITValueDesc{Loc: scm.LocImm, Type: d386.Type, Imm: scm.NewInt(int64(uint64(d386.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d386.Reg, 32)
			ctx.EmitShrRegImm8(d386.Reg, 32)
		}
		if d386.Loc == scm.LocReg && d5.Loc == scm.LocReg && d386.Reg == d5.Reg {
			ctx.TransferReg(d5.Reg)
			d5.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d386)
		ctx.EmitStoreToStack(d386, int32(bbs[4].PhiBase)+int32(32))
		ctx.StabilizeDescForControlFlow(&d386)
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d5)
		var d387 scm.JITValueDesc
		if d5.Loc == scm.LocImm {
			d387 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d387 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d387)
		}
		if d387.Loc == scm.LocImm {
			d387 = scm.JITValueDesc{Loc: scm.LocImm, Type: d387.Type, Imm: scm.NewInt(int64(uint64(d387.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d387.Reg, 32)
			ctx.EmitShrRegImm8(d387.Reg, 32)
		}
		if d387.Loc == scm.LocReg && d5.Loc == scm.LocReg && d387.Reg == d5.Reg {
			ctx.TransferReg(d5.Reg)
			d5.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d387)
		ctx.EmitStoreToStack(d387, int32(bbs[4].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d387)
		if ps.General {
			ctx.SyncDesc(&d6)
			if d6.Loc == scm.LocReg {
				ctx.ProtectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.ProtectReg(d6.Reg)
				ctx.ProtectReg(d6.Reg2)
			}
			d388 = d6
			if d388.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d388)
			d389 = d388
			if d389.Loc == scm.LocImm {
				d389 = scm.JITValueDesc{Loc: scm.LocImm, Type: d389.Type, Imm: scm.NewInt(int64(uint64(d389.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d389.Reg, 32)
				ctx.EmitShrRegImm8(d389.Reg, 32)
			}
			ctx.EmitStoreToStack(d389, int32(bbs[4].PhiBase)+int32(16))
			if d6.Loc == scm.LocReg {
				ctx.UnprotectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d6.Reg)
				ctx.UnprotectReg(d6.Reg2)
			}
		}
		ps390 := scm.PhiState{General: ps.General}
		ps390.OverlayValues = make([]scm.JITValueDesc, 390)
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
		ps390.OverlayValues[18] = d18
		ps390.OverlayValues[19] = d19
		ps390.OverlayValues[20] = d20
		ps390.OverlayValues[21] = d21
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
		ps390.OverlayValues[52] = d52
		ps390.OverlayValues[53] = d53
		ps390.OverlayValues[54] = d54
		ps390.OverlayValues[55] = d55
		ps390.OverlayValues[56] = d56
		ps390.OverlayValues[57] = d57
		ps390.OverlayValues[60] = d60
		ps390.OverlayValues[61] = d61
		ps390.OverlayValues[62] = d62
		ps390.OverlayValues[177] = d177
		ps390.OverlayValues[178] = d178
		ps390.OverlayValues[179] = d179
		ps390.OverlayValues[180] = d180
		ps390.OverlayValues[181] = d181
		ps390.OverlayValues[182] = d182
		ps390.OverlayValues[183] = d183
		ps390.OverlayValues[184] = d184
		ps390.OverlayValues[185] = d185
		ps390.OverlayValues[186] = d186
		ps390.OverlayValues[187] = d187
		ps390.OverlayValues[188] = d188
		ps390.OverlayValues[189] = d189
		ps390.OverlayValues[190] = d190
		ps390.OverlayValues[191] = d191
		ps390.OverlayValues[192] = d192
		ps390.OverlayValues[193] = d193
		ps390.OverlayValues[194] = d194
		ps390.OverlayValues[195] = d195
		ps390.OverlayValues[196] = d196
		ps390.OverlayValues[197] = d197
		ps390.OverlayValues[198] = d198
		ps390.OverlayValues[199] = d199
		ps390.OverlayValues[200] = d200
		ps390.OverlayValues[201] = d201
		ps390.OverlayValues[202] = d202
		ps390.OverlayValues[203] = d203
		ps390.OverlayValues[204] = d204
		ps390.OverlayValues[205] = d205
		ps390.OverlayValues[206] = d206
		ps390.OverlayValues[209] = d209
		ps390.OverlayValues[386] = d386
		ps390.OverlayValues[387] = d387
		ps390.OverlayValues[388] = d388
		ps390.OverlayValues[389] = d389
		ps390.PhiValues = make([]scm.JITValueDesc, 3)
		d391 = d6
		ps390.PhiValues[1] = d391
		if ps390.General && bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
			return result
		}
		return bbs[4].RenderPS(ps390)
		return result
	}
	bbs[4].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d392 := ps.PhiValues[0]
				ctx.EnsureDesc(&d392)
				ctx.EmitStoreToStack(d392, int32(bbs[4].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d393 := ps.PhiValues[1]
				ctx.EnsureDesc(&d393)
				ctx.EmitStoreToStack(d393, int32(bbs[4].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d394 := ps.PhiValues[2]
				ctx.EnsureDesc(&d394)
				ctx.EmitStoreToStack(d394, int32(bbs[4].PhiBase)+int32(32))
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
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
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
		if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != scm.LocNone {
			d177 = ps.OverlayValues[177]
		}
		if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
			d178 = ps.OverlayValues[178]
		}
		if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != scm.LocNone {
			d179 = ps.OverlayValues[179]
		}
		if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != scm.LocNone {
			d180 = ps.OverlayValues[180]
		}
		if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != scm.LocNone {
			d181 = ps.OverlayValues[181]
		}
		if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != scm.LocNone {
			d182 = ps.OverlayValues[182]
		}
		if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != scm.LocNone {
			d183 = ps.OverlayValues[183]
		}
		if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
			d184 = ps.OverlayValues[184]
		}
		if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != scm.LocNone {
			d185 = ps.OverlayValues[185]
		}
		if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != scm.LocNone {
			d186 = ps.OverlayValues[186]
		}
		if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
			d187 = ps.OverlayValues[187]
		}
		if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != scm.LocNone {
			d188 = ps.OverlayValues[188]
		}
		if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != scm.LocNone {
			d189 = ps.OverlayValues[189]
		}
		if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != scm.LocNone {
			d190 = ps.OverlayValues[190]
		}
		if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != scm.LocNone {
			d191 = ps.OverlayValues[191]
		}
		if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != scm.LocNone {
			d192 = ps.OverlayValues[192]
		}
		if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != scm.LocNone {
			d193 = ps.OverlayValues[193]
		}
		if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != scm.LocNone {
			d194 = ps.OverlayValues[194]
		}
		if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != scm.LocNone {
			d195 = ps.OverlayValues[195]
		}
		if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
			d196 = ps.OverlayValues[196]
		}
		if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != scm.LocNone {
			d197 = ps.OverlayValues[197]
		}
		if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != scm.LocNone {
			d198 = ps.OverlayValues[198]
		}
		if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
			d199 = ps.OverlayValues[199]
		}
		if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
			d200 = ps.OverlayValues[200]
		}
		if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
			d201 = ps.OverlayValues[201]
		}
		if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != scm.LocNone {
			d202 = ps.OverlayValues[202]
		}
		if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != scm.LocNone {
			d203 = ps.OverlayValues[203]
		}
		if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != scm.LocNone {
			d204 = ps.OverlayValues[204]
		}
		if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != scm.LocNone {
			d205 = ps.OverlayValues[205]
		}
		if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != scm.LocNone {
			d206 = ps.OverlayValues[206]
		}
		if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != scm.LocNone {
			d209 = ps.OverlayValues[209]
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
		var d395 scm.JITValueDesc
		if d10.Loc == scm.LocImm && d11.Loc == scm.LocImm {
			d395 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d10.Imm.Int()) == uint64(d11.Imm.Int()))}
		} else if d11.Loc == scm.LocImm {
			r70 := ctx.AllocRegExcept(d10.Reg)
			if d11.Imm.Int() >= -2147483648 && d11.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d10.Reg, int32(d11.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d11.Imm.Int()))
				ctx.EmitCmpInt64(d10.Reg, scm.RegR11)
			}
			d395 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r70, Condition: scm.CondEqual}
			ctx.BindReg(r70, &d395)
		} else if d10.Loc == scm.LocImm {
			r71 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d10.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d11.Reg)
			d395 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r71, Condition: scm.CondEqual}
			ctx.BindReg(r71, &d395)
		} else {
			r72 := ctx.AllocRegExcept(d10.Reg)
			ctx.EmitCmpInt64(d10.Reg, d11.Reg)
			d395 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r72, Condition: scm.CondEqual}
			ctx.BindReg(r72, &d395)
		}
		d396 = d395
		ctx.EnsureDesc(&d396)
		if d396.Loc != scm.LocImm && d396.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d396.Loc == scm.LocImm {
			if d396.Imm.Bool() {
				if ps.General {
					ctx.SyncDesc(&d10)
					if d10.Loc == scm.LocReg {
						ctx.ProtectReg(d10.Reg)
					} else if d10.Loc == scm.LocRegPair {
						ctx.ProtectReg(d10.Reg)
						ctx.ProtectReg(d10.Reg2)
					}
					d397 = d10
					if d397.Loc == scm.LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d397)
					d398 = d397
					if d398.Loc == scm.LocImm {
						d398 = scm.JITValueDesc{Loc: scm.LocImm, Type: d398.Type, Imm: scm.NewInt(int64(uint64(d398.Imm.Int()) & 0xffffffff))}
					} else {
						ctx.EmitShlRegImm8(d398.Reg, 32)
						ctx.EmitShrRegImm8(d398.Reg, 32)
					}
					ctx.EmitStoreToStack(d398, int32(bbs[2].PhiBase)+int32(0))
					if d10.Loc == scm.LocReg {
						ctx.UnprotectReg(d10.Reg)
					} else if d10.Loc == scm.LocRegPair {
						ctx.UnprotectReg(d10.Reg)
						ctx.UnprotectReg(d10.Reg2)
					}
				}
				ps399 := scm.PhiState{General: ps.General}
				ps399.OverlayValues = make([]scm.JITValueDesc, 399)
				ps399.OverlayValues[5] = d5
				ps399.OverlayValues[6] = d6
				ps399.OverlayValues[7] = d7
				ps399.OverlayValues[8] = d8
				ps399.OverlayValues[9] = d9
				ps399.OverlayValues[10] = d10
				ps399.OverlayValues[11] = d11
				ps399.OverlayValues[12] = d12
				ps399.OverlayValues[13] = d13
				ps399.OverlayValues[14] = d14
				ps399.OverlayValues[15] = d15
				ps399.OverlayValues[16] = d16
				ps399.OverlayValues[17] = d17
				ps399.OverlayValues[18] = d18
				ps399.OverlayValues[19] = d19
				ps399.OverlayValues[20] = d20
				ps399.OverlayValues[21] = d21
				ps399.OverlayValues[23] = d23
				ps399.OverlayValues[24] = d24
				ps399.OverlayValues[25] = d25
				ps399.OverlayValues[26] = d26
				ps399.OverlayValues[27] = d27
				ps399.OverlayValues[28] = d28
				ps399.OverlayValues[29] = d29
				ps399.OverlayValues[30] = d30
				ps399.OverlayValues[31] = d31
				ps399.OverlayValues[32] = d32
				ps399.OverlayValues[33] = d33
				ps399.OverlayValues[34] = d34
				ps399.OverlayValues[35] = d35
				ps399.OverlayValues[36] = d36
				ps399.OverlayValues[37] = d37
				ps399.OverlayValues[38] = d38
				ps399.OverlayValues[39] = d39
				ps399.OverlayValues[40] = d40
				ps399.OverlayValues[41] = d41
				ps399.OverlayValues[42] = d42
				ps399.OverlayValues[43] = d43
				ps399.OverlayValues[44] = d44
				ps399.OverlayValues[45] = d45
				ps399.OverlayValues[46] = d46
				ps399.OverlayValues[47] = d47
				ps399.OverlayValues[48] = d48
				ps399.OverlayValues[49] = d49
				ps399.OverlayValues[50] = d50
				ps399.OverlayValues[51] = d51
				ps399.OverlayValues[52] = d52
				ps399.OverlayValues[53] = d53
				ps399.OverlayValues[54] = d54
				ps399.OverlayValues[55] = d55
				ps399.OverlayValues[56] = d56
				ps399.OverlayValues[57] = d57
				ps399.OverlayValues[60] = d60
				ps399.OverlayValues[61] = d61
				ps399.OverlayValues[62] = d62
				ps399.OverlayValues[177] = d177
				ps399.OverlayValues[178] = d178
				ps399.OverlayValues[179] = d179
				ps399.OverlayValues[180] = d180
				ps399.OverlayValues[181] = d181
				ps399.OverlayValues[182] = d182
				ps399.OverlayValues[183] = d183
				ps399.OverlayValues[184] = d184
				ps399.OverlayValues[185] = d185
				ps399.OverlayValues[186] = d186
				ps399.OverlayValues[187] = d187
				ps399.OverlayValues[188] = d188
				ps399.OverlayValues[189] = d189
				ps399.OverlayValues[190] = d190
				ps399.OverlayValues[191] = d191
				ps399.OverlayValues[192] = d192
				ps399.OverlayValues[193] = d193
				ps399.OverlayValues[194] = d194
				ps399.OverlayValues[195] = d195
				ps399.OverlayValues[196] = d196
				ps399.OverlayValues[197] = d197
				ps399.OverlayValues[198] = d198
				ps399.OverlayValues[199] = d199
				ps399.OverlayValues[200] = d200
				ps399.OverlayValues[201] = d201
				ps399.OverlayValues[202] = d202
				ps399.OverlayValues[203] = d203
				ps399.OverlayValues[204] = d204
				ps399.OverlayValues[205] = d205
				ps399.OverlayValues[206] = d206
				ps399.OverlayValues[209] = d209
				ps399.OverlayValues[386] = d386
				ps399.OverlayValues[387] = d387
				ps399.OverlayValues[388] = d388
				ps399.OverlayValues[389] = d389
				ps399.OverlayValues[391] = d391
				ps399.OverlayValues[392] = d392
				ps399.OverlayValues[393] = d393
				ps399.OverlayValues[394] = d394
				ps399.OverlayValues[395] = d395
				ps399.OverlayValues[396] = d396
				ps399.OverlayValues[397] = d397
				ps399.OverlayValues[398] = d398
				ps399.PhiValues = make([]scm.JITValueDesc, 1)
				d400 = d10
				ps399.PhiValues[0] = d400
				return bbs[2].RenderPS(ps399)
			}
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
			ps401.OverlayValues[57] = d57
			ps401.OverlayValues[60] = d60
			ps401.OverlayValues[61] = d61
			ps401.OverlayValues[62] = d62
			ps401.OverlayValues[177] = d177
			ps401.OverlayValues[178] = d178
			ps401.OverlayValues[179] = d179
			ps401.OverlayValues[180] = d180
			ps401.OverlayValues[181] = d181
			ps401.OverlayValues[182] = d182
			ps401.OverlayValues[183] = d183
			ps401.OverlayValues[184] = d184
			ps401.OverlayValues[185] = d185
			ps401.OverlayValues[186] = d186
			ps401.OverlayValues[187] = d187
			ps401.OverlayValues[188] = d188
			ps401.OverlayValues[189] = d189
			ps401.OverlayValues[190] = d190
			ps401.OverlayValues[191] = d191
			ps401.OverlayValues[192] = d192
			ps401.OverlayValues[193] = d193
			ps401.OverlayValues[194] = d194
			ps401.OverlayValues[195] = d195
			ps401.OverlayValues[196] = d196
			ps401.OverlayValues[197] = d197
			ps401.OverlayValues[198] = d198
			ps401.OverlayValues[199] = d199
			ps401.OverlayValues[200] = d200
			ps401.OverlayValues[201] = d201
			ps401.OverlayValues[202] = d202
			ps401.OverlayValues[203] = d203
			ps401.OverlayValues[204] = d204
			ps401.OverlayValues[205] = d205
			ps401.OverlayValues[206] = d206
			ps401.OverlayValues[209] = d209
			ps401.OverlayValues[386] = d386
			ps401.OverlayValues[387] = d387
			ps401.OverlayValues[388] = d388
			ps401.OverlayValues[389] = d389
			ps401.OverlayValues[391] = d391
			ps401.OverlayValues[392] = d392
			ps401.OverlayValues[393] = d393
			ps401.OverlayValues[394] = d394
			ps401.OverlayValues[395] = d395
			ps401.OverlayValues[396] = d396
			ps401.OverlayValues[397] = d397
			ps401.OverlayValues[398] = d398
			ps401.OverlayValues[400] = d400
			return bbs[6].RenderPS(ps401)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d402 := ps.PhiValues[0]
				ctx.EnsureDesc(&d402)
				ctx.EmitStoreToStack(d402, int32(bbs[4].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d403 := ps.PhiValues[1]
				ctx.EnsureDesc(&d403)
				ctx.EmitStoreToStack(d403, int32(bbs[4].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d404 := ps.PhiValues[2]
				ctx.EnsureDesc(&d404)
				ctx.EmitStoreToStack(d404, int32(bbs[4].PhiBase)+int32(32))
			}
			ps.General = true
			return bbs[4].RenderPS(ps)
		}
		lbl21 := ctx.ReserveLabel()
		lbl22 := ctx.ReserveLabel()
		ctx.EmitJump(d396.Condition, lbl21)
		ctx.EmitJmp(lbl22)
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
		snap456 := d57
		snap457 := d60
		snap458 := d61
		snap459 := d62
		snap460 := d177
		snap461 := d178
		snap462 := d179
		snap463 := d180
		snap464 := d181
		snap465 := d182
		snap466 := d183
		snap467 := d184
		snap468 := d185
		snap469 := d186
		snap470 := d187
		snap471 := d188
		snap472 := d189
		snap473 := d190
		snap474 := d191
		snap475 := d192
		snap476 := d193
		snap477 := d194
		snap478 := d195
		snap479 := d196
		snap480 := d197
		snap481 := d198
		snap482 := d199
		snap483 := d200
		snap484 := d201
		snap485 := d202
		snap486 := d203
		snap487 := d204
		snap488 := d205
		snap489 := d206
		snap490 := d209
		snap491 := d386
		snap492 := d387
		snap493 := d388
		snap494 := d389
		snap495 := d391
		snap496 := d392
		snap497 := d393
		snap498 := d394
		snap499 := d395
		snap500 := d396
		snap501 := d397
		snap502 := d398
		snap503 := d400
		snap504 := d402
		snap505 := d403
		snap506 := d404
		alloc507 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl21)
		ctx.SyncDesc(&d10)
		if d10.Loc == scm.LocReg {
			ctx.ProtectReg(d10.Reg)
		} else if d10.Loc == scm.LocRegPair {
			ctx.ProtectReg(d10.Reg)
			ctx.ProtectReg(d10.Reg2)
		}
		d508 = d10
		if d508.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d508)
		d509 = d508
		if d509.Loc == scm.LocImm {
			d509 = scm.JITValueDesc{Loc: scm.LocImm, Type: d509.Type, Imm: scm.NewInt(int64(uint64(d509.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d509.Reg, 32)
			ctx.EmitShrRegImm8(d509.Reg, 32)
		}
		ctx.EmitStoreToStack(d509, int32(bbs[2].PhiBase)+int32(0))
		if d10.Loc == scm.LocReg {
			ctx.UnprotectReg(d10.Reg)
		} else if d10.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d10.Reg)
			ctx.UnprotectReg(d10.Reg2)
		}
		ctx.EmitJmp(lbl3)
		ctx.RestoreAllocState(alloc507)
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
		d57 = snap456
		d60 = snap457
		d61 = snap458
		d62 = snap459
		d177 = snap460
		d178 = snap461
		d179 = snap462
		d180 = snap463
		d181 = snap464
		d182 = snap465
		d183 = snap466
		d184 = snap467
		d185 = snap468
		d186 = snap469
		d187 = snap470
		d188 = snap471
		d189 = snap472
		d190 = snap473
		d191 = snap474
		d192 = snap475
		d193 = snap476
		d194 = snap477
		d195 = snap478
		d196 = snap479
		d197 = snap480
		d198 = snap481
		d199 = snap482
		d200 = snap483
		d201 = snap484
		d202 = snap485
		d203 = snap486
		d204 = snap487
		d205 = snap488
		d206 = snap489
		d209 = snap490
		d386 = snap491
		d387 = snap492
		d388 = snap493
		d389 = snap494
		d391 = snap495
		d392 = snap496
		d393 = snap497
		d394 = snap498
		d395 = snap499
		d396 = snap500
		d397 = snap501
		d398 = snap502
		d400 = snap503
		d402 = snap504
		d403 = snap505
		d404 = snap506
		ctx.MarkLabel(lbl22)
		ctx.EmitJmp(lbl7)
		ctx.RestoreAllocState(alloc507)
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
		d57 = snap456
		d60 = snap457
		d61 = snap458
		d62 = snap459
		d177 = snap460
		d178 = snap461
		d179 = snap462
		d180 = snap463
		d181 = snap464
		d182 = snap465
		d183 = snap466
		d184 = snap467
		d185 = snap468
		d186 = snap469
		d187 = snap470
		d188 = snap471
		d189 = snap472
		d190 = snap473
		d191 = snap474
		d192 = snap475
		d193 = snap476
		d194 = snap477
		d195 = snap478
		d196 = snap479
		d197 = snap480
		d198 = snap481
		d199 = snap482
		d200 = snap483
		d201 = snap484
		d202 = snap485
		d203 = snap486
		d204 = snap487
		d205 = snap488
		d206 = snap489
		d209 = snap490
		d386 = snap491
		d387 = snap492
		d388 = snap493
		d389 = snap494
		d391 = snap495
		d392 = snap496
		d393 = snap497
		d394 = snap498
		d395 = snap499
		d396 = snap500
		d397 = snap501
		d398 = snap502
		d400 = snap503
		d402 = snap504
		d403 = snap505
		d404 = snap506
		ps510 := scm.PhiState{General: true}
		ps510.OverlayValues = make([]scm.JITValueDesc, 510)
		ps510.OverlayValues[5] = d5
		ps510.OverlayValues[6] = d6
		ps510.OverlayValues[7] = d7
		ps510.OverlayValues[8] = d8
		ps510.OverlayValues[9] = d9
		ps510.OverlayValues[10] = d10
		ps510.OverlayValues[11] = d11
		ps510.OverlayValues[12] = d12
		ps510.OverlayValues[13] = d13
		ps510.OverlayValues[14] = d14
		ps510.OverlayValues[15] = d15
		ps510.OverlayValues[16] = d16
		ps510.OverlayValues[17] = d17
		ps510.OverlayValues[18] = d18
		ps510.OverlayValues[19] = d19
		ps510.OverlayValues[20] = d20
		ps510.OverlayValues[21] = d21
		ps510.OverlayValues[23] = d23
		ps510.OverlayValues[24] = d24
		ps510.OverlayValues[25] = d25
		ps510.OverlayValues[26] = d26
		ps510.OverlayValues[27] = d27
		ps510.OverlayValues[28] = d28
		ps510.OverlayValues[29] = d29
		ps510.OverlayValues[30] = d30
		ps510.OverlayValues[31] = d31
		ps510.OverlayValues[32] = d32
		ps510.OverlayValues[33] = d33
		ps510.OverlayValues[34] = d34
		ps510.OverlayValues[35] = d35
		ps510.OverlayValues[36] = d36
		ps510.OverlayValues[37] = d37
		ps510.OverlayValues[38] = d38
		ps510.OverlayValues[39] = d39
		ps510.OverlayValues[40] = d40
		ps510.OverlayValues[41] = d41
		ps510.OverlayValues[42] = d42
		ps510.OverlayValues[43] = d43
		ps510.OverlayValues[44] = d44
		ps510.OverlayValues[45] = d45
		ps510.OverlayValues[46] = d46
		ps510.OverlayValues[47] = d47
		ps510.OverlayValues[48] = d48
		ps510.OverlayValues[49] = d49
		ps510.OverlayValues[50] = d50
		ps510.OverlayValues[51] = d51
		ps510.OverlayValues[52] = d52
		ps510.OverlayValues[53] = d53
		ps510.OverlayValues[54] = d54
		ps510.OverlayValues[55] = d55
		ps510.OverlayValues[56] = d56
		ps510.OverlayValues[57] = d57
		ps510.OverlayValues[60] = d60
		ps510.OverlayValues[61] = d61
		ps510.OverlayValues[62] = d62
		ps510.OverlayValues[177] = d177
		ps510.OverlayValues[178] = d178
		ps510.OverlayValues[179] = d179
		ps510.OverlayValues[180] = d180
		ps510.OverlayValues[181] = d181
		ps510.OverlayValues[182] = d182
		ps510.OverlayValues[183] = d183
		ps510.OverlayValues[184] = d184
		ps510.OverlayValues[185] = d185
		ps510.OverlayValues[186] = d186
		ps510.OverlayValues[187] = d187
		ps510.OverlayValues[188] = d188
		ps510.OverlayValues[189] = d189
		ps510.OverlayValues[190] = d190
		ps510.OverlayValues[191] = d191
		ps510.OverlayValues[192] = d192
		ps510.OverlayValues[193] = d193
		ps510.OverlayValues[194] = d194
		ps510.OverlayValues[195] = d195
		ps510.OverlayValues[196] = d196
		ps510.OverlayValues[197] = d197
		ps510.OverlayValues[198] = d198
		ps510.OverlayValues[199] = d199
		ps510.OverlayValues[200] = d200
		ps510.OverlayValues[201] = d201
		ps510.OverlayValues[202] = d202
		ps510.OverlayValues[203] = d203
		ps510.OverlayValues[204] = d204
		ps510.OverlayValues[205] = d205
		ps510.OverlayValues[206] = d206
		ps510.OverlayValues[209] = d209
		ps510.OverlayValues[386] = d386
		ps510.OverlayValues[387] = d387
		ps510.OverlayValues[388] = d388
		ps510.OverlayValues[389] = d389
		ps510.OverlayValues[391] = d391
		ps510.OverlayValues[392] = d392
		ps510.OverlayValues[393] = d393
		ps510.OverlayValues[394] = d394
		ps510.OverlayValues[395] = d395
		ps510.OverlayValues[396] = d396
		ps510.OverlayValues[397] = d397
		ps510.OverlayValues[398] = d398
		ps510.OverlayValues[400] = d400
		ps510.OverlayValues[402] = d402
		ps510.OverlayValues[403] = d403
		ps510.OverlayValues[404] = d404
		ps510.OverlayValues[508] = d508
		ps510.OverlayValues[509] = d509
		ps510.PhiValues = make([]scm.JITValueDesc, 1)
		d512 = d10
		ps510.PhiValues[0] = d512
		ps511 := scm.PhiState{General: true}
		ps511.OverlayValues = make([]scm.JITValueDesc, 513)
		ps511.OverlayValues[5] = d5
		ps511.OverlayValues[6] = d6
		ps511.OverlayValues[7] = d7
		ps511.OverlayValues[8] = d8
		ps511.OverlayValues[9] = d9
		ps511.OverlayValues[10] = d10
		ps511.OverlayValues[11] = d11
		ps511.OverlayValues[12] = d12
		ps511.OverlayValues[13] = d13
		ps511.OverlayValues[14] = d14
		ps511.OverlayValues[15] = d15
		ps511.OverlayValues[16] = d16
		ps511.OverlayValues[17] = d17
		ps511.OverlayValues[18] = d18
		ps511.OverlayValues[19] = d19
		ps511.OverlayValues[20] = d20
		ps511.OverlayValues[21] = d21
		ps511.OverlayValues[23] = d23
		ps511.OverlayValues[24] = d24
		ps511.OverlayValues[25] = d25
		ps511.OverlayValues[26] = d26
		ps511.OverlayValues[27] = d27
		ps511.OverlayValues[28] = d28
		ps511.OverlayValues[29] = d29
		ps511.OverlayValues[30] = d30
		ps511.OverlayValues[31] = d31
		ps511.OverlayValues[32] = d32
		ps511.OverlayValues[33] = d33
		ps511.OverlayValues[34] = d34
		ps511.OverlayValues[35] = d35
		ps511.OverlayValues[36] = d36
		ps511.OverlayValues[37] = d37
		ps511.OverlayValues[38] = d38
		ps511.OverlayValues[39] = d39
		ps511.OverlayValues[40] = d40
		ps511.OverlayValues[41] = d41
		ps511.OverlayValues[42] = d42
		ps511.OverlayValues[43] = d43
		ps511.OverlayValues[44] = d44
		ps511.OverlayValues[45] = d45
		ps511.OverlayValues[46] = d46
		ps511.OverlayValues[47] = d47
		ps511.OverlayValues[48] = d48
		ps511.OverlayValues[49] = d49
		ps511.OverlayValues[50] = d50
		ps511.OverlayValues[51] = d51
		ps511.OverlayValues[52] = d52
		ps511.OverlayValues[53] = d53
		ps511.OverlayValues[54] = d54
		ps511.OverlayValues[55] = d55
		ps511.OverlayValues[56] = d56
		ps511.OverlayValues[57] = d57
		ps511.OverlayValues[60] = d60
		ps511.OverlayValues[61] = d61
		ps511.OverlayValues[62] = d62
		ps511.OverlayValues[177] = d177
		ps511.OverlayValues[178] = d178
		ps511.OverlayValues[179] = d179
		ps511.OverlayValues[180] = d180
		ps511.OverlayValues[181] = d181
		ps511.OverlayValues[182] = d182
		ps511.OverlayValues[183] = d183
		ps511.OverlayValues[184] = d184
		ps511.OverlayValues[185] = d185
		ps511.OverlayValues[186] = d186
		ps511.OverlayValues[187] = d187
		ps511.OverlayValues[188] = d188
		ps511.OverlayValues[189] = d189
		ps511.OverlayValues[190] = d190
		ps511.OverlayValues[191] = d191
		ps511.OverlayValues[192] = d192
		ps511.OverlayValues[193] = d193
		ps511.OverlayValues[194] = d194
		ps511.OverlayValues[195] = d195
		ps511.OverlayValues[196] = d196
		ps511.OverlayValues[197] = d197
		ps511.OverlayValues[198] = d198
		ps511.OverlayValues[199] = d199
		ps511.OverlayValues[200] = d200
		ps511.OverlayValues[201] = d201
		ps511.OverlayValues[202] = d202
		ps511.OverlayValues[203] = d203
		ps511.OverlayValues[204] = d204
		ps511.OverlayValues[205] = d205
		ps511.OverlayValues[206] = d206
		ps511.OverlayValues[209] = d209
		ps511.OverlayValues[386] = d386
		ps511.OverlayValues[387] = d387
		ps511.OverlayValues[388] = d388
		ps511.OverlayValues[389] = d389
		ps511.OverlayValues[391] = d391
		ps511.OverlayValues[392] = d392
		ps511.OverlayValues[393] = d393
		ps511.OverlayValues[394] = d394
		ps511.OverlayValues[395] = d395
		ps511.OverlayValues[396] = d396
		ps511.OverlayValues[397] = d397
		ps511.OverlayValues[398] = d398
		ps511.OverlayValues[400] = d400
		ps511.OverlayValues[402] = d402
		ps511.OverlayValues[403] = d403
		ps511.OverlayValues[404] = d404
		ps511.OverlayValues[508] = d508
		ps511.OverlayValues[509] = d509
		ps511.OverlayValues[512] = d512
		snap513 := d5
		snap514 := d6
		snap515 := d7
		snap516 := d8
		snap517 := d9
		snap518 := d10
		snap519 := d11
		snap520 := d12
		snap521 := d13
		snap522 := d14
		snap523 := d15
		snap524 := d16
		snap525 := d17
		snap526 := d18
		snap527 := d19
		snap528 := d20
		snap529 := d21
		snap530 := d23
		snap531 := d24
		snap532 := d25
		snap533 := d26
		snap534 := d27
		snap535 := d28
		snap536 := d29
		snap537 := d30
		snap538 := d31
		snap539 := d32
		snap540 := d33
		snap541 := d34
		snap542 := d35
		snap543 := d36
		snap544 := d37
		snap545 := d38
		snap546 := d39
		snap547 := d40
		snap548 := d41
		snap549 := d42
		snap550 := d43
		snap551 := d44
		snap552 := d45
		snap553 := d46
		snap554 := d47
		snap555 := d48
		snap556 := d49
		snap557 := d50
		snap558 := d51
		snap559 := d52
		snap560 := d53
		snap561 := d54
		snap562 := d55
		snap563 := d56
		snap564 := d57
		snap565 := d60
		snap566 := d61
		snap567 := d62
		snap568 := d177
		snap569 := d178
		snap570 := d179
		snap571 := d180
		snap572 := d181
		snap573 := d182
		snap574 := d183
		snap575 := d184
		snap576 := d185
		snap577 := d186
		snap578 := d187
		snap579 := d188
		snap580 := d189
		snap581 := d190
		snap582 := d191
		snap583 := d192
		snap584 := d193
		snap585 := d194
		snap586 := d195
		snap587 := d196
		snap588 := d197
		snap589 := d198
		snap590 := d199
		snap591 := d200
		snap592 := d201
		snap593 := d202
		snap594 := d203
		snap595 := d204
		snap596 := d205
		snap597 := d206
		snap598 := d209
		snap599 := d386
		snap600 := d387
		snap601 := d388
		snap602 := d389
		snap603 := d391
		snap604 := d392
		snap605 := d393
		snap606 := d394
		snap607 := d395
		snap608 := d396
		snap609 := d397
		snap610 := d398
		snap611 := d400
		snap612 := d402
		snap613 := d403
		snap614 := d404
		snap615 := d508
		snap616 := d509
		snap617 := d512
		alloc618 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps510)
		}
		ctx.RestoreAllocState(alloc618)
		d5 = snap513
		d6 = snap514
		d7 = snap515
		d8 = snap516
		d9 = snap517
		d10 = snap518
		d11 = snap519
		d12 = snap520
		d13 = snap521
		d14 = snap522
		d15 = snap523
		d16 = snap524
		d17 = snap525
		d18 = snap526
		d19 = snap527
		d20 = snap528
		d21 = snap529
		d23 = snap530
		d24 = snap531
		d25 = snap532
		d26 = snap533
		d27 = snap534
		d28 = snap535
		d29 = snap536
		d30 = snap537
		d31 = snap538
		d32 = snap539
		d33 = snap540
		d34 = snap541
		d35 = snap542
		d36 = snap543
		d37 = snap544
		d38 = snap545
		d39 = snap546
		d40 = snap547
		d41 = snap548
		d42 = snap549
		d43 = snap550
		d44 = snap551
		d45 = snap552
		d46 = snap553
		d47 = snap554
		d48 = snap555
		d49 = snap556
		d50 = snap557
		d51 = snap558
		d52 = snap559
		d53 = snap560
		d54 = snap561
		d55 = snap562
		d56 = snap563
		d57 = snap564
		d60 = snap565
		d61 = snap566
		d62 = snap567
		d177 = snap568
		d178 = snap569
		d179 = snap570
		d180 = snap571
		d181 = snap572
		d182 = snap573
		d183 = snap574
		d184 = snap575
		d185 = snap576
		d186 = snap577
		d187 = snap578
		d188 = snap579
		d189 = snap580
		d190 = snap581
		d191 = snap582
		d192 = snap583
		d193 = snap584
		d194 = snap585
		d195 = snap586
		d196 = snap587
		d197 = snap588
		d198 = snap589
		d199 = snap590
		d200 = snap591
		d201 = snap592
		d202 = snap593
		d203 = snap594
		d204 = snap595
		d205 = snap596
		d206 = snap597
		d209 = snap598
		d386 = snap599
		d387 = snap600
		d388 = snap601
		d389 = snap602
		d391 = snap603
		d392 = snap604
		d393 = snap605
		d394 = snap606
		d395 = snap607
		d396 = snap608
		d397 = snap609
		d398 = snap610
		d400 = snap611
		d402 = snap612
		d403 = snap613
		d404 = snap614
		d508 = snap615
		d509 = snap616
		d512 = snap617
		if !bbs[6].Rendered {
			return bbs[6].RenderPS(ps511)
		}
		return result
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
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
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
		if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != scm.LocNone {
			d177 = ps.OverlayValues[177]
		}
		if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
			d178 = ps.OverlayValues[178]
		}
		if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != scm.LocNone {
			d179 = ps.OverlayValues[179]
		}
		if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != scm.LocNone {
			d180 = ps.OverlayValues[180]
		}
		if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != scm.LocNone {
			d181 = ps.OverlayValues[181]
		}
		if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != scm.LocNone {
			d182 = ps.OverlayValues[182]
		}
		if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != scm.LocNone {
			d183 = ps.OverlayValues[183]
		}
		if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
			d184 = ps.OverlayValues[184]
		}
		if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != scm.LocNone {
			d185 = ps.OverlayValues[185]
		}
		if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != scm.LocNone {
			d186 = ps.OverlayValues[186]
		}
		if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
			d187 = ps.OverlayValues[187]
		}
		if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != scm.LocNone {
			d188 = ps.OverlayValues[188]
		}
		if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != scm.LocNone {
			d189 = ps.OverlayValues[189]
		}
		if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != scm.LocNone {
			d190 = ps.OverlayValues[190]
		}
		if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != scm.LocNone {
			d191 = ps.OverlayValues[191]
		}
		if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != scm.LocNone {
			d192 = ps.OverlayValues[192]
		}
		if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != scm.LocNone {
			d193 = ps.OverlayValues[193]
		}
		if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != scm.LocNone {
			d194 = ps.OverlayValues[194]
		}
		if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != scm.LocNone {
			d195 = ps.OverlayValues[195]
		}
		if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
			d196 = ps.OverlayValues[196]
		}
		if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != scm.LocNone {
			d197 = ps.OverlayValues[197]
		}
		if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != scm.LocNone {
			d198 = ps.OverlayValues[198]
		}
		if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
			d199 = ps.OverlayValues[199]
		}
		if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
			d200 = ps.OverlayValues[200]
		}
		if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
			d201 = ps.OverlayValues[201]
		}
		if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != scm.LocNone {
			d202 = ps.OverlayValues[202]
		}
		if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != scm.LocNone {
			d203 = ps.OverlayValues[203]
		}
		if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != scm.LocNone {
			d204 = ps.OverlayValues[204]
		}
		if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != scm.LocNone {
			d205 = ps.OverlayValues[205]
		}
		if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != scm.LocNone {
			d206 = ps.OverlayValues[206]
		}
		if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != scm.LocNone {
			d209 = ps.OverlayValues[209]
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
		if len(ps.OverlayValues) > 400 && ps.OverlayValues[400].Loc != scm.LocNone {
			d400 = ps.OverlayValues[400]
		}
		if len(ps.OverlayValues) > 402 && ps.OverlayValues[402].Loc != scm.LocNone {
			d402 = ps.OverlayValues[402]
		}
		if len(ps.OverlayValues) > 403 && ps.OverlayValues[403].Loc != scm.LocNone {
			d403 = ps.OverlayValues[403]
		}
		if len(ps.OverlayValues) > 404 && ps.OverlayValues[404].Loc != scm.LocNone {
			d404 = ps.OverlayValues[404]
		}
		if len(ps.OverlayValues) > 508 && ps.OverlayValues[508].Loc != scm.LocNone {
			d508 = ps.OverlayValues[508]
		}
		if len(ps.OverlayValues) > 509 && ps.OverlayValues[509].Loc != scm.LocNone {
			d509 = ps.OverlayValues[509]
		}
		if len(ps.OverlayValues) > 512 && ps.OverlayValues[512].Loc != scm.LocNone {
			d512 = ps.OverlayValues[512]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d5)
		var d619 scm.JITValueDesc
		if d5.Loc == scm.LocImm {
			d619 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d619 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d619)
		}
		if d619.Loc == scm.LocImm {
			d619 = scm.JITValueDesc{Loc: scm.LocImm, Type: d619.Type, Imm: scm.NewInt(int64(uint64(d619.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d619.Reg, 32)
			ctx.EmitShrRegImm8(d619.Reg, 32)
		}
		if d619.Loc == scm.LocReg && d5.Loc == scm.LocReg && d619.Reg == d5.Reg {
			ctx.TransferReg(d5.Reg)
			d5.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d619)
		ctx.EmitStoreToStack(d619, int32(bbs[4].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d619)
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
			d620 = d5
			if d620.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d620)
			d621 = d620
			if d621.Loc == scm.LocImm {
				d621 = scm.JITValueDesc{Loc: scm.LocImm, Type: d621.Type, Imm: scm.NewInt(int64(uint64(d621.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d621.Reg, 32)
				ctx.EmitShrRegImm8(d621.Reg, 32)
			}
			ctx.EmitStoreToStack(d621, int32(bbs[4].PhiBase)+int32(16))
			d622 = d7
			if d622.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d622)
			d623 = d622
			if d623.Loc == scm.LocImm {
				d623 = scm.JITValueDesc{Loc: scm.LocImm, Type: d623.Type, Imm: scm.NewInt(int64(uint64(d623.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d623.Reg, 32)
				ctx.EmitShrRegImm8(d623.Reg, 32)
			}
			ctx.EmitStoreToStack(d623, int32(bbs[4].PhiBase)+int32(32))
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
		ps624 := scm.PhiState{General: ps.General}
		ps624.OverlayValues = make([]scm.JITValueDesc, 624)
		ps624.OverlayValues[5] = d5
		ps624.OverlayValues[6] = d6
		ps624.OverlayValues[7] = d7
		ps624.OverlayValues[8] = d8
		ps624.OverlayValues[9] = d9
		ps624.OverlayValues[10] = d10
		ps624.OverlayValues[11] = d11
		ps624.OverlayValues[12] = d12
		ps624.OverlayValues[13] = d13
		ps624.OverlayValues[14] = d14
		ps624.OverlayValues[15] = d15
		ps624.OverlayValues[16] = d16
		ps624.OverlayValues[17] = d17
		ps624.OverlayValues[18] = d18
		ps624.OverlayValues[19] = d19
		ps624.OverlayValues[20] = d20
		ps624.OverlayValues[21] = d21
		ps624.OverlayValues[23] = d23
		ps624.OverlayValues[24] = d24
		ps624.OverlayValues[25] = d25
		ps624.OverlayValues[26] = d26
		ps624.OverlayValues[27] = d27
		ps624.OverlayValues[28] = d28
		ps624.OverlayValues[29] = d29
		ps624.OverlayValues[30] = d30
		ps624.OverlayValues[31] = d31
		ps624.OverlayValues[32] = d32
		ps624.OverlayValues[33] = d33
		ps624.OverlayValues[34] = d34
		ps624.OverlayValues[35] = d35
		ps624.OverlayValues[36] = d36
		ps624.OverlayValues[37] = d37
		ps624.OverlayValues[38] = d38
		ps624.OverlayValues[39] = d39
		ps624.OverlayValues[40] = d40
		ps624.OverlayValues[41] = d41
		ps624.OverlayValues[42] = d42
		ps624.OverlayValues[43] = d43
		ps624.OverlayValues[44] = d44
		ps624.OverlayValues[45] = d45
		ps624.OverlayValues[46] = d46
		ps624.OverlayValues[47] = d47
		ps624.OverlayValues[48] = d48
		ps624.OverlayValues[49] = d49
		ps624.OverlayValues[50] = d50
		ps624.OverlayValues[51] = d51
		ps624.OverlayValues[52] = d52
		ps624.OverlayValues[53] = d53
		ps624.OverlayValues[54] = d54
		ps624.OverlayValues[55] = d55
		ps624.OverlayValues[56] = d56
		ps624.OverlayValues[57] = d57
		ps624.OverlayValues[60] = d60
		ps624.OverlayValues[61] = d61
		ps624.OverlayValues[62] = d62
		ps624.OverlayValues[177] = d177
		ps624.OverlayValues[178] = d178
		ps624.OverlayValues[179] = d179
		ps624.OverlayValues[180] = d180
		ps624.OverlayValues[181] = d181
		ps624.OverlayValues[182] = d182
		ps624.OverlayValues[183] = d183
		ps624.OverlayValues[184] = d184
		ps624.OverlayValues[185] = d185
		ps624.OverlayValues[186] = d186
		ps624.OverlayValues[187] = d187
		ps624.OverlayValues[188] = d188
		ps624.OverlayValues[189] = d189
		ps624.OverlayValues[190] = d190
		ps624.OverlayValues[191] = d191
		ps624.OverlayValues[192] = d192
		ps624.OverlayValues[193] = d193
		ps624.OverlayValues[194] = d194
		ps624.OverlayValues[195] = d195
		ps624.OverlayValues[196] = d196
		ps624.OverlayValues[197] = d197
		ps624.OverlayValues[198] = d198
		ps624.OverlayValues[199] = d199
		ps624.OverlayValues[200] = d200
		ps624.OverlayValues[201] = d201
		ps624.OverlayValues[202] = d202
		ps624.OverlayValues[203] = d203
		ps624.OverlayValues[204] = d204
		ps624.OverlayValues[205] = d205
		ps624.OverlayValues[206] = d206
		ps624.OverlayValues[209] = d209
		ps624.OverlayValues[386] = d386
		ps624.OverlayValues[387] = d387
		ps624.OverlayValues[388] = d388
		ps624.OverlayValues[389] = d389
		ps624.OverlayValues[391] = d391
		ps624.OverlayValues[392] = d392
		ps624.OverlayValues[393] = d393
		ps624.OverlayValues[394] = d394
		ps624.OverlayValues[395] = d395
		ps624.OverlayValues[396] = d396
		ps624.OverlayValues[397] = d397
		ps624.OverlayValues[398] = d398
		ps624.OverlayValues[400] = d400
		ps624.OverlayValues[402] = d402
		ps624.OverlayValues[403] = d403
		ps624.OverlayValues[404] = d404
		ps624.OverlayValues[508] = d508
		ps624.OverlayValues[509] = d509
		ps624.OverlayValues[512] = d512
		ps624.OverlayValues[619] = d619
		ps624.OverlayValues[620] = d620
		ps624.OverlayValues[621] = d621
		ps624.OverlayValues[622] = d622
		ps624.OverlayValues[623] = d623
		ps624.PhiValues = make([]scm.JITValueDesc, 3)
		d625 = d5
		ps624.PhiValues[1] = d625
		d626 = d7
		ps624.PhiValues[2] = d626
		if ps624.General && bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
			return result
		}
		return bbs[4].RenderPS(ps624)
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
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
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
		if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != scm.LocNone {
			d177 = ps.OverlayValues[177]
		}
		if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
			d178 = ps.OverlayValues[178]
		}
		if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != scm.LocNone {
			d179 = ps.OverlayValues[179]
		}
		if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != scm.LocNone {
			d180 = ps.OverlayValues[180]
		}
		if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != scm.LocNone {
			d181 = ps.OverlayValues[181]
		}
		if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != scm.LocNone {
			d182 = ps.OverlayValues[182]
		}
		if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != scm.LocNone {
			d183 = ps.OverlayValues[183]
		}
		if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
			d184 = ps.OverlayValues[184]
		}
		if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != scm.LocNone {
			d185 = ps.OverlayValues[185]
		}
		if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != scm.LocNone {
			d186 = ps.OverlayValues[186]
		}
		if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
			d187 = ps.OverlayValues[187]
		}
		if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != scm.LocNone {
			d188 = ps.OverlayValues[188]
		}
		if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != scm.LocNone {
			d189 = ps.OverlayValues[189]
		}
		if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != scm.LocNone {
			d190 = ps.OverlayValues[190]
		}
		if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != scm.LocNone {
			d191 = ps.OverlayValues[191]
		}
		if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != scm.LocNone {
			d192 = ps.OverlayValues[192]
		}
		if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != scm.LocNone {
			d193 = ps.OverlayValues[193]
		}
		if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != scm.LocNone {
			d194 = ps.OverlayValues[194]
		}
		if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != scm.LocNone {
			d195 = ps.OverlayValues[195]
		}
		if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
			d196 = ps.OverlayValues[196]
		}
		if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != scm.LocNone {
			d197 = ps.OverlayValues[197]
		}
		if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != scm.LocNone {
			d198 = ps.OverlayValues[198]
		}
		if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
			d199 = ps.OverlayValues[199]
		}
		if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
			d200 = ps.OverlayValues[200]
		}
		if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
			d201 = ps.OverlayValues[201]
		}
		if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != scm.LocNone {
			d202 = ps.OverlayValues[202]
		}
		if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != scm.LocNone {
			d203 = ps.OverlayValues[203]
		}
		if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != scm.LocNone {
			d204 = ps.OverlayValues[204]
		}
		if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != scm.LocNone {
			d205 = ps.OverlayValues[205]
		}
		if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != scm.LocNone {
			d206 = ps.OverlayValues[206]
		}
		if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != scm.LocNone {
			d209 = ps.OverlayValues[209]
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
		if len(ps.OverlayValues) > 400 && ps.OverlayValues[400].Loc != scm.LocNone {
			d400 = ps.OverlayValues[400]
		}
		if len(ps.OverlayValues) > 402 && ps.OverlayValues[402].Loc != scm.LocNone {
			d402 = ps.OverlayValues[402]
		}
		if len(ps.OverlayValues) > 403 && ps.OverlayValues[403].Loc != scm.LocNone {
			d403 = ps.OverlayValues[403]
		}
		if len(ps.OverlayValues) > 404 && ps.OverlayValues[404].Loc != scm.LocNone {
			d404 = ps.OverlayValues[404]
		}
		if len(ps.OverlayValues) > 508 && ps.OverlayValues[508].Loc != scm.LocNone {
			d508 = ps.OverlayValues[508]
		}
		if len(ps.OverlayValues) > 509 && ps.OverlayValues[509].Loc != scm.LocNone {
			d509 = ps.OverlayValues[509]
		}
		if len(ps.OverlayValues) > 512 && ps.OverlayValues[512].Loc != scm.LocNone {
			d512 = ps.OverlayValues[512]
		}
		if len(ps.OverlayValues) > 619 && ps.OverlayValues[619].Loc != scm.LocNone {
			d619 = ps.OverlayValues[619]
		}
		if len(ps.OverlayValues) > 620 && ps.OverlayValues[620].Loc != scm.LocNone {
			d620 = ps.OverlayValues[620]
		}
		if len(ps.OverlayValues) > 621 && ps.OverlayValues[621].Loc != scm.LocNone {
			d621 = ps.OverlayValues[621]
		}
		if len(ps.OverlayValues) > 622 && ps.OverlayValues[622].Loc != scm.LocNone {
			d622 = ps.OverlayValues[622]
		}
		if len(ps.OverlayValues) > 623 && ps.OverlayValues[623].Loc != scm.LocNone {
			d623 = ps.OverlayValues[623]
		}
		if len(ps.OverlayValues) > 625 && ps.OverlayValues[625].Loc != scm.LocNone {
			d625 = ps.OverlayValues[625]
		}
		if len(ps.OverlayValues) > 626 && ps.OverlayValues[626].Loc != scm.LocNone {
			d626 = ps.OverlayValues[626]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d9)
		d627 = d9
		_ = d627
		ctx.StabilizeDescForControlFlow(&d627)
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
		ctx.EnsureDesc(&d627)
		ctx.EnsureDesc(&d627)
		var d628 scm.JITValueDesc
		if d627.Loc == scm.LocImm {
			d628 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d627.Imm.Int()))))}
		} else {
			r73 := ctx.AllocReg()
			ctx.EmitMovRegReg(r73, d627.Reg)
			ctx.EmitShlRegImm8(r73, 32)
			ctx.EmitShrRegImm8(r73, 32)
			d628 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
			ctx.BindReg(r73, &d628)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d629 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d629 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48)
			r74 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r74, thisptr.Reg, off)
			d629 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r74}
			ctx.BindReg(r74, &d629)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d629)
		ctx.EnsureDesc(&d629)
		var d630 scm.JITValueDesc
		if d629.Loc == scm.LocImm {
			d630 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d629.Imm.Int()))))}
		} else {
			r75 := ctx.AllocReg()
			ctx.EmitMovRegReg(r75, d629.Reg)
			ctx.EmitShlRegImm8(r75, 56)
			ctx.EmitShrRegImm8(r75, 56)
			d630 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
			ctx.BindReg(r75, &d630)
		}
		ctx.FreeDesc(&d629)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d628)
		ctx.EnsureDesc(&d630)
		ctx.EnsureDescsTogether(&d628, &d630)
		var d631 scm.JITValueDesc
		if d628.Loc == scm.LocImm && d630.Loc == scm.LocImm {
			d631 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d628.Imm.Int() * d630.Imm.Int())}
		} else if d628.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d630.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d628.Imm.Int()))
			ctx.EmitImulInt64(scratch, d630.Reg)
			d631 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d631)
		} else if d630.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d628.Reg)
			ctx.EmitMovRegReg(scratch, d628.Reg)
			if d630.Imm.Int() >= -2147483648 && d630.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d630.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d630.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d631 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d631)
		} else {
			r76 := ctx.AllocRegExcept(d628.Reg, d630.Reg)
			ctx.EmitMovRegReg(r76, d628.Reg)
			ctx.EmitImulInt64(r76, d630.Reg)
			d631 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r76}
			ctx.BindReg(r76, &d631)
		}
		if d631.Loc == scm.LocReg && d628.Loc == scm.LocReg && d631.Reg == d628.Reg {
			ctx.TransferReg(d628.Reg)
			d628.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d628)
		ctx.FreeDesc(&d630)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d631)
		var d632 scm.JITValueDesc
		if d631.Loc == scm.LocImm {
			d632 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d631.Imm.Int() / 64)}
		} else {
			r77 := ctx.AllocRegExcept(d631.Reg)
			ctx.EmitMovRegReg(r77, d631.Reg)
			ctx.EmitShrRegImm8(r77, 6)
			d632 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r77}
			ctx.BindReg(r77, &d632)
		}
		if d632.Loc == scm.LocReg && d631.Loc == scm.LocReg && d632.Reg == d631.Reg {
			ctx.TransferReg(d631.Reg)
			d631.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d631)
		var d633 scm.JITValueDesc
		if d631.Loc == scm.LocImm {
			d633 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d631.Imm.Int() % 64)}
		} else {
			r78 := ctx.AllocRegExcept(d631.Reg)
			ctx.EmitMovRegReg(r78, d631.Reg)
			ctx.EmitAndRegImm32(r78, 63)
			d633 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r78}
			ctx.BindReg(r78, &d633)
		}
		if d633.Loc == scm.LocReg && d631.Loc == scm.LocReg && d633.Reg == d631.Reg {
			ctx.TransferReg(d631.Reg)
			d631.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d631)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d634 scm.JITValueDesc
		r79 := ctx.AllocReg()
		r80 := ctx.AllocRegExcept(r79)
		r81 := ctx.AllocRegExcept(r79, r80)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			ctx.EmitMovRegImm64(r79, uint64(dataPtr))
			ctx.EmitMovRegImm64(r80, uint64(sliceLen))
			ctx.EmitMovRegImm64(r81, uint64(sliceCap))
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			ctx.EmitMovRegMem(r79, thisptr.Reg, off)
			ctx.EmitMovRegMem(r80, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r81, thisptr.Reg, off+16)
		}
		d634 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r79, Reg2: r80, Reg3: r81}
		ctx.BindReg(r79, &d634)
		ctx.BindReg(r80, &d634)
		ctx.BindReg(r81, &d634)
		ctx.BindReg(r79, &d634)
		ctx.BindReg(r80, &d634)
		ctx.BindReg(r81, &d634)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d632)
		ctx.ReclaimUntrackedRegs()
		d636 = ctx.EmitSliceElementAddress(&d634, &d632, 8)
		ctx.EnsureDesc(&d636)
		ctx.EmitMovRegMem(d636.Reg, d636.Reg, 0)
		d635 = d636
		d635.Type = scm.TagInt
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d635)
		ctx.EnsureDesc(&d633)
		ctx.EnsureDescsTogether(&d635, &d633)
		var d637 scm.JITValueDesc
		if d635.Loc == scm.LocImm && d633.Loc == scm.LocImm {
			d637 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d635.Imm.Int()) << uint64(d633.Imm.Int())))}
		} else if d633.Loc == scm.LocImm {
			r82 := ctx.AllocRegExcept(d635.Reg)
			ctx.EmitMovRegReg(r82, d635.Reg)
			ctx.EmitShlRegImm8(r82, uint8(d633.Imm.Int()))
			d637 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r82}
			ctx.BindReg(r82, &d637)
		} else {
			{
				shiftSrc := d635.Reg
				r83 := ctx.AllocRegExcept(d635.Reg, d633.Reg)
				ctx.EmitMovRegReg(r83, d635.Reg)
				shiftSrc = r83
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d633.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d633.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d633.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d637 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d637)
			}
		}
		if d637.Loc == scm.LocReg && d635.Loc == scm.LocReg && d637.Reg == d635.Reg {
			ctx.TransferReg(d635.Reg)
			d635.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d635)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d632)
		ctx.EnsureDesc(&d632)
		var d638 scm.JITValueDesc
		if d632.Loc == scm.LocImm {
			d638 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d632.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d632.Reg)
			ctx.EmitMovRegReg(scratch, d632.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d638 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d638)
		}
		if d638.Loc == scm.LocReg && d632.Loc == scm.LocReg && d638.Reg == d632.Reg {
			ctx.TransferReg(d632.Reg)
			d632.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d632)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d638)
		ctx.ReclaimUntrackedRegs()
		d640 = ctx.EmitSliceElementAddress(&d634, &d638, 8)
		ctx.EnsureDesc(&d640)
		ctx.EmitMovRegMem(d640.Reg, d640.Reg, 0)
		d639 = d640
		d639.Type = scm.TagInt
		ctx.FreeDesc(&d638)
		ctx.ReclaimUntrackedRegs()
		d641 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d633)
		ctx.EnsureDescsTogether(&d641, &d633)
		var d642 scm.JITValueDesc
		if d641.Loc == scm.LocImm && d633.Loc == scm.LocImm {
			d642 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d641.Imm.Int() - d633.Imm.Int())}
		} else if d633.Loc == scm.LocImm && d633.Imm.Int() == 0 {
			r84 := ctx.AllocRegExcept(d641.Reg)
			ctx.EmitMovRegReg(r84, d641.Reg)
			d642 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r84}
			ctx.BindReg(r84, &d642)
		} else if d641.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d633.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d641.Imm.Int()))
			ctx.EmitSubInt64(scratch, d633.Reg)
			d642 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d642)
		} else if d633.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d641.Reg)
			ctx.EmitMovRegReg(scratch, d641.Reg)
			if d633.Imm.Int() >= -2147483648 && d633.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d633.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d633.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d642 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d642)
		} else {
			r85 := ctx.AllocRegExcept(d641.Reg, d633.Reg)
			ctx.EmitMovRegReg(r85, d641.Reg)
			ctx.EmitSubInt64(r85, d633.Reg)
			d642 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
			ctx.BindReg(r85, &d642)
		}
		if d642.Loc == scm.LocReg && d641.Loc == scm.LocReg && d642.Reg == d641.Reg {
			ctx.TransferReg(d641.Reg)
			d641.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d633)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d639)
		ctx.EnsureDesc(&d642)
		ctx.EnsureDescsTogether(&d639, &d642)
		var d643 scm.JITValueDesc
		if d639.Loc == scm.LocImm && d642.Loc == scm.LocImm {
			d643 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d639.Imm.Int()) >> uint64(d642.Imm.Int())))}
		} else if d642.Loc == scm.LocImm {
			r86 := ctx.AllocRegExcept(d639.Reg)
			ctx.EmitMovRegReg(r86, d639.Reg)
			ctx.EmitShrRegImm8(r86, uint8(d642.Imm.Int()))
			d643 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
			ctx.BindReg(r86, &d643)
		} else {
			{
				shiftSrc := d639.Reg
				r87 := ctx.AllocRegExcept(d639.Reg, d642.Reg)
				ctx.EmitMovRegReg(r87, d639.Reg)
				shiftSrc = r87
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d642.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d642.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d642.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d643 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d643)
			}
		}
		if d643.Loc == scm.LocReg && d639.Loc == scm.LocReg && d643.Reg == d639.Reg {
			ctx.TransferReg(d639.Reg)
			d639.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d639)
		ctx.FreeDesc(&d642)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d637)
		ctx.EnsureDesc(&d643)
		var d644 scm.JITValueDesc
		if d637.Loc == scm.LocImm && d643.Loc == scm.LocImm {
			d644 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d637.Imm.Int() | d643.Imm.Int())}
		} else if d637.Loc == scm.LocImm && d637.Imm.Int() == 0 {
			d644 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d643.Reg}
			ctx.BindReg(d643.Reg, &d644)
		} else if d643.Loc == scm.LocImm && d643.Imm.Int() == 0 {
			r88 := ctx.AllocRegExcept(d637.Reg)
			ctx.EmitMovRegReg(r88, d637.Reg)
			d644 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
			ctx.BindReg(r88, &d644)
		} else if d637.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d643.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d637.Imm.Int()))
			ctx.EmitOrInt64(scratch, d643.Reg)
			d644 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d644)
		} else if d643.Loc == scm.LocImm {
			r89 := ctx.AllocRegExcept(d637.Reg)
			ctx.EmitMovRegReg(r89, d637.Reg)
			if d643.Imm.Int() >= -2147483648 && d643.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r89, int32(d643.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d643.Imm.Int()))
				ctx.EmitOrInt64(r89, scm.RegR11)
			}
			d644 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
			ctx.BindReg(r89, &d644)
		} else {
			r90 := ctx.AllocRegExcept(d637.Reg, d643.Reg)
			ctx.EmitMovRegReg(r90, d637.Reg)
			ctx.EmitOrInt64(r90, d643.Reg)
			d644 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
			ctx.BindReg(r90, &d644)
		}
		if d644.Loc == scm.LocReg && d637.Loc == scm.LocReg && d644.Reg == d637.Reg {
			ctx.TransferReg(d637.Reg)
			d637.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d637)
		ctx.FreeDesc(&d643)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d645 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d645 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48)
			r91 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r91, thisptr.Reg, off)
			d645 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r91}
			ctx.BindReg(r91, &d645)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d645)
		ctx.EnsureDesc(&d645)
		var d646 scm.JITValueDesc
		if d645.Loc == scm.LocImm {
			d646 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d645.Imm.Int()))))}
		} else {
			r92 := ctx.AllocReg()
			ctx.EmitMovRegReg(r92, d645.Reg)
			ctx.EmitShlRegImm8(r92, 56)
			ctx.EmitShrRegImm8(r92, 56)
			d646 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r92}
			ctx.BindReg(r92, &d646)
		}
		ctx.FreeDesc(&d645)
		ctx.ReclaimUntrackedRegs()
		d647 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d646)
		ctx.EnsureDescsTogether(&d647, &d646)
		var d648 scm.JITValueDesc
		if d647.Loc == scm.LocImm && d646.Loc == scm.LocImm {
			d648 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d647.Imm.Int() - d646.Imm.Int())}
		} else if d646.Loc == scm.LocImm && d646.Imm.Int() == 0 {
			r93 := ctx.AllocRegExcept(d647.Reg)
			ctx.EmitMovRegReg(r93, d647.Reg)
			d648 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r93}
			ctx.BindReg(r93, &d648)
		} else if d647.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d646.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d647.Imm.Int()))
			ctx.EmitSubInt64(scratch, d646.Reg)
			d648 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d648)
		} else if d646.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d647.Reg)
			ctx.EmitMovRegReg(scratch, d647.Reg)
			if d646.Imm.Int() >= -2147483648 && d646.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d646.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d646.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d648 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d648)
		} else {
			r94 := ctx.AllocRegExcept(d647.Reg, d646.Reg)
			ctx.EmitMovRegReg(r94, d647.Reg)
			ctx.EmitSubInt64(r94, d646.Reg)
			d648 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
			ctx.BindReg(r94, &d648)
		}
		if d648.Loc == scm.LocReg && d647.Loc == scm.LocReg && d648.Reg == d647.Reg {
			ctx.TransferReg(d647.Reg)
			d647.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d646)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d644)
		ctx.EnsureDesc(&d648)
		ctx.EnsureDescsTogether(&d644, &d648)
		var d649 scm.JITValueDesc
		if d644.Loc == scm.LocImm && d648.Loc == scm.LocImm {
			d649 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d644.Imm.Int()) >> uint64(d648.Imm.Int())))}
		} else if d648.Loc == scm.LocImm {
			r95 := ctx.AllocRegExcept(d644.Reg)
			ctx.EmitMovRegReg(r95, d644.Reg)
			ctx.EmitShrRegImm8(r95, uint8(d648.Imm.Int()))
			d649 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r95}
			ctx.BindReg(r95, &d649)
		} else {
			{
				shiftSrc := d644.Reg
				r96 := ctx.AllocRegExcept(d644.Reg, d648.Reg)
				ctx.EmitMovRegReg(r96, d644.Reg)
				shiftSrc = r96
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d648.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d648.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d648.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d649 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d649)
			}
		}
		if d649.Loc == scm.LocReg && d644.Loc == scm.LocReg && d649.Reg == d644.Reg {
			ctx.TransferReg(d644.Reg)
			d644.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d644)
		ctx.FreeDesc(&d648)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d649)
		ctx.EnsureDesc(&d649)
		ctx.EnsureDesc(&d649)
		var d650 scm.JITValueDesc
		if d649.Loc == scm.LocImm {
			d650 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d649.Imm.Int()))))}
		} else {
			r97 := ctx.AllocReg()
			ctx.EmitMovRegReg(r97, d649.Reg)
			d650 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
			ctx.BindReg(r97, &d650)
		}
		ctx.FreeDesc(&d649)
		var d651 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d651 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56)
			r98 := ctx.AllocReg()
			ctx.EmitMovRegMem(r98, thisptr.Reg, off)
			d651 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r98}
			ctx.BindReg(r98, &d651)
		}
		ctx.EnsureDesc(&d650)
		ctx.EnsureDesc(&d651)
		ctx.EnsureDescsTogether(&d650, &d651)
		var d652 scm.JITValueDesc
		if d650.Loc == scm.LocImm && d651.Loc == scm.LocImm {
			d652 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d650.Imm.Int() + d651.Imm.Int())}
		} else if d651.Loc == scm.LocImm && d651.Imm.Int() == 0 {
			r99 := ctx.AllocRegExcept(d650.Reg)
			ctx.EmitMovRegReg(r99, d650.Reg)
			d652 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
			ctx.BindReg(r99, &d652)
		} else if d650.Loc == scm.LocImm && d650.Imm.Int() == 0 {
			d652 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d651.Reg}
			ctx.BindReg(d651.Reg, &d652)
		} else if d650.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d651.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d650.Imm.Int()))
			ctx.EmitAddInt64(scratch, d651.Reg)
			d652 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d652)
		} else if d651.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d650.Reg)
			ctx.EmitMovRegReg(scratch, d650.Reg)
			if d651.Imm.Int() >= -2147483648 && d651.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d651.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d651.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d652 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d652)
		} else {
			r100 := ctx.AllocRegExcept(d650.Reg, d651.Reg)
			ctx.EmitMovRegReg(r100, d650.Reg)
			ctx.EmitAddInt64(r100, d651.Reg)
			d652 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r100}
			ctx.BindReg(r100, &d652)
		}
		if d652.Loc == scm.LocReg && d650.Loc == scm.LocReg && d652.Reg == d650.Reg {
			ctx.TransferReg(d650.Reg)
			d650.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d650)
		ctx.FreeDesc(&d651)
		ctx.EnsureDesc(&d652)
		ctx.EnsureDesc(&d652)
		var d653 scm.JITValueDesc
		if d652.Loc == scm.LocImm {
			d653 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d652.Imm.Int()))))}
		} else {
			r101 := ctx.AllocReg()
			ctx.EmitMovRegReg(r101, d652.Reg)
			ctx.EmitShlRegImm8(r101, 32)
			ctx.EmitShrRegImm8(r101, 32)
			d653 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r101}
			ctx.BindReg(r101, &d653)
		}
		ctx.FreeDesc(&d652)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d653)
		ctx.EnsureDescsTogether(&idxInt, &d653)
		var d654 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm && d653.Loc == scm.LocImm {
			d654 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d653.Imm.Int()))}
		} else if d653.Loc == scm.LocImm {
			r102 := ctx.AllocRegExcept(idxInt.Reg)
			if d653.Imm.Int() >= -2147483648 && d653.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(idxInt.Reg, int32(d653.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d653.Imm.Int()))
				ctx.EmitCmpInt64(idxInt.Reg, scm.RegR11)
			}
			d654 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r102, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r102, &d654)
		} else if idxInt.Loc == scm.LocImm {
			r103 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d653.Reg)
			d654 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r103, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r103, &d654)
		} else {
			r104 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitCmpInt64(idxInt.Reg, d653.Reg)
			d654 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r104, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r104, &d654)
		}
		ctx.FreeDesc(&d653)
		d655 = d654
		ctx.EnsureDesc(&d655)
		if d655.Loc != scm.LocImm && d655.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d655.Loc == scm.LocImm {
			if d655.Imm.Bool() {
				if ps.General {
				}
				ps656 := scm.PhiState{General: ps.General}
				ps656.OverlayValues = make([]scm.JITValueDesc, 656)
				ps656.OverlayValues[5] = d5
				ps656.OverlayValues[6] = d6
				ps656.OverlayValues[7] = d7
				ps656.OverlayValues[8] = d8
				ps656.OverlayValues[9] = d9
				ps656.OverlayValues[10] = d10
				ps656.OverlayValues[11] = d11
				ps656.OverlayValues[12] = d12
				ps656.OverlayValues[13] = d13
				ps656.OverlayValues[14] = d14
				ps656.OverlayValues[15] = d15
				ps656.OverlayValues[16] = d16
				ps656.OverlayValues[17] = d17
				ps656.OverlayValues[18] = d18
				ps656.OverlayValues[19] = d19
				ps656.OverlayValues[20] = d20
				ps656.OverlayValues[21] = d21
				ps656.OverlayValues[23] = d23
				ps656.OverlayValues[24] = d24
				ps656.OverlayValues[25] = d25
				ps656.OverlayValues[26] = d26
				ps656.OverlayValues[27] = d27
				ps656.OverlayValues[28] = d28
				ps656.OverlayValues[29] = d29
				ps656.OverlayValues[30] = d30
				ps656.OverlayValues[31] = d31
				ps656.OverlayValues[32] = d32
				ps656.OverlayValues[33] = d33
				ps656.OverlayValues[34] = d34
				ps656.OverlayValues[35] = d35
				ps656.OverlayValues[36] = d36
				ps656.OverlayValues[37] = d37
				ps656.OverlayValues[38] = d38
				ps656.OverlayValues[39] = d39
				ps656.OverlayValues[40] = d40
				ps656.OverlayValues[41] = d41
				ps656.OverlayValues[42] = d42
				ps656.OverlayValues[43] = d43
				ps656.OverlayValues[44] = d44
				ps656.OverlayValues[45] = d45
				ps656.OverlayValues[46] = d46
				ps656.OverlayValues[47] = d47
				ps656.OverlayValues[48] = d48
				ps656.OverlayValues[49] = d49
				ps656.OverlayValues[50] = d50
				ps656.OverlayValues[51] = d51
				ps656.OverlayValues[52] = d52
				ps656.OverlayValues[53] = d53
				ps656.OverlayValues[54] = d54
				ps656.OverlayValues[55] = d55
				ps656.OverlayValues[56] = d56
				ps656.OverlayValues[57] = d57
				ps656.OverlayValues[60] = d60
				ps656.OverlayValues[61] = d61
				ps656.OverlayValues[62] = d62
				ps656.OverlayValues[177] = d177
				ps656.OverlayValues[178] = d178
				ps656.OverlayValues[179] = d179
				ps656.OverlayValues[180] = d180
				ps656.OverlayValues[181] = d181
				ps656.OverlayValues[182] = d182
				ps656.OverlayValues[183] = d183
				ps656.OverlayValues[184] = d184
				ps656.OverlayValues[185] = d185
				ps656.OverlayValues[186] = d186
				ps656.OverlayValues[187] = d187
				ps656.OverlayValues[188] = d188
				ps656.OverlayValues[189] = d189
				ps656.OverlayValues[190] = d190
				ps656.OverlayValues[191] = d191
				ps656.OverlayValues[192] = d192
				ps656.OverlayValues[193] = d193
				ps656.OverlayValues[194] = d194
				ps656.OverlayValues[195] = d195
				ps656.OverlayValues[196] = d196
				ps656.OverlayValues[197] = d197
				ps656.OverlayValues[198] = d198
				ps656.OverlayValues[199] = d199
				ps656.OverlayValues[200] = d200
				ps656.OverlayValues[201] = d201
				ps656.OverlayValues[202] = d202
				ps656.OverlayValues[203] = d203
				ps656.OverlayValues[204] = d204
				ps656.OverlayValues[205] = d205
				ps656.OverlayValues[206] = d206
				ps656.OverlayValues[209] = d209
				ps656.OverlayValues[386] = d386
				ps656.OverlayValues[387] = d387
				ps656.OverlayValues[388] = d388
				ps656.OverlayValues[389] = d389
				ps656.OverlayValues[391] = d391
				ps656.OverlayValues[392] = d392
				ps656.OverlayValues[393] = d393
				ps656.OverlayValues[394] = d394
				ps656.OverlayValues[395] = d395
				ps656.OverlayValues[396] = d396
				ps656.OverlayValues[397] = d397
				ps656.OverlayValues[398] = d398
				ps656.OverlayValues[400] = d400
				ps656.OverlayValues[402] = d402
				ps656.OverlayValues[403] = d403
				ps656.OverlayValues[404] = d404
				ps656.OverlayValues[508] = d508
				ps656.OverlayValues[509] = d509
				ps656.OverlayValues[512] = d512
				ps656.OverlayValues[619] = d619
				ps656.OverlayValues[620] = d620
				ps656.OverlayValues[621] = d621
				ps656.OverlayValues[622] = d622
				ps656.OverlayValues[623] = d623
				ps656.OverlayValues[625] = d625
				ps656.OverlayValues[626] = d626
				ps656.OverlayValues[627] = d627
				ps656.OverlayValues[628] = d628
				ps656.OverlayValues[629] = d629
				ps656.OverlayValues[630] = d630
				ps656.OverlayValues[631] = d631
				ps656.OverlayValues[632] = d632
				ps656.OverlayValues[633] = d633
				ps656.OverlayValues[634] = d634
				ps656.OverlayValues[635] = d635
				ps656.OverlayValues[636] = d636
				ps656.OverlayValues[637] = d637
				ps656.OverlayValues[638] = d638
				ps656.OverlayValues[639] = d639
				ps656.OverlayValues[640] = d640
				ps656.OverlayValues[641] = d641
				ps656.OverlayValues[642] = d642
				ps656.OverlayValues[643] = d643
				ps656.OverlayValues[644] = d644
				ps656.OverlayValues[645] = d645
				ps656.OverlayValues[646] = d646
				ps656.OverlayValues[647] = d647
				ps656.OverlayValues[648] = d648
				ps656.OverlayValues[649] = d649
				ps656.OverlayValues[650] = d650
				ps656.OverlayValues[651] = d651
				ps656.OverlayValues[652] = d652
				ps656.OverlayValues[653] = d653
				ps656.OverlayValues[654] = d654
				ps656.OverlayValues[655] = d655
				return bbs[7].RenderPS(ps656)
			}
			if ps.General {
			}
			ps657 := scm.PhiState{General: ps.General}
			ps657.OverlayValues = make([]scm.JITValueDesc, 656)
			ps657.OverlayValues[5] = d5
			ps657.OverlayValues[6] = d6
			ps657.OverlayValues[7] = d7
			ps657.OverlayValues[8] = d8
			ps657.OverlayValues[9] = d9
			ps657.OverlayValues[10] = d10
			ps657.OverlayValues[11] = d11
			ps657.OverlayValues[12] = d12
			ps657.OverlayValues[13] = d13
			ps657.OverlayValues[14] = d14
			ps657.OverlayValues[15] = d15
			ps657.OverlayValues[16] = d16
			ps657.OverlayValues[17] = d17
			ps657.OverlayValues[18] = d18
			ps657.OverlayValues[19] = d19
			ps657.OverlayValues[20] = d20
			ps657.OverlayValues[21] = d21
			ps657.OverlayValues[23] = d23
			ps657.OverlayValues[24] = d24
			ps657.OverlayValues[25] = d25
			ps657.OverlayValues[26] = d26
			ps657.OverlayValues[27] = d27
			ps657.OverlayValues[28] = d28
			ps657.OverlayValues[29] = d29
			ps657.OverlayValues[30] = d30
			ps657.OverlayValues[31] = d31
			ps657.OverlayValues[32] = d32
			ps657.OverlayValues[33] = d33
			ps657.OverlayValues[34] = d34
			ps657.OverlayValues[35] = d35
			ps657.OverlayValues[36] = d36
			ps657.OverlayValues[37] = d37
			ps657.OverlayValues[38] = d38
			ps657.OverlayValues[39] = d39
			ps657.OverlayValues[40] = d40
			ps657.OverlayValues[41] = d41
			ps657.OverlayValues[42] = d42
			ps657.OverlayValues[43] = d43
			ps657.OverlayValues[44] = d44
			ps657.OverlayValues[45] = d45
			ps657.OverlayValues[46] = d46
			ps657.OverlayValues[47] = d47
			ps657.OverlayValues[48] = d48
			ps657.OverlayValues[49] = d49
			ps657.OverlayValues[50] = d50
			ps657.OverlayValues[51] = d51
			ps657.OverlayValues[52] = d52
			ps657.OverlayValues[53] = d53
			ps657.OverlayValues[54] = d54
			ps657.OverlayValues[55] = d55
			ps657.OverlayValues[56] = d56
			ps657.OverlayValues[57] = d57
			ps657.OverlayValues[60] = d60
			ps657.OverlayValues[61] = d61
			ps657.OverlayValues[62] = d62
			ps657.OverlayValues[177] = d177
			ps657.OverlayValues[178] = d178
			ps657.OverlayValues[179] = d179
			ps657.OverlayValues[180] = d180
			ps657.OverlayValues[181] = d181
			ps657.OverlayValues[182] = d182
			ps657.OverlayValues[183] = d183
			ps657.OverlayValues[184] = d184
			ps657.OverlayValues[185] = d185
			ps657.OverlayValues[186] = d186
			ps657.OverlayValues[187] = d187
			ps657.OverlayValues[188] = d188
			ps657.OverlayValues[189] = d189
			ps657.OverlayValues[190] = d190
			ps657.OverlayValues[191] = d191
			ps657.OverlayValues[192] = d192
			ps657.OverlayValues[193] = d193
			ps657.OverlayValues[194] = d194
			ps657.OverlayValues[195] = d195
			ps657.OverlayValues[196] = d196
			ps657.OverlayValues[197] = d197
			ps657.OverlayValues[198] = d198
			ps657.OverlayValues[199] = d199
			ps657.OverlayValues[200] = d200
			ps657.OverlayValues[201] = d201
			ps657.OverlayValues[202] = d202
			ps657.OverlayValues[203] = d203
			ps657.OverlayValues[204] = d204
			ps657.OverlayValues[205] = d205
			ps657.OverlayValues[206] = d206
			ps657.OverlayValues[209] = d209
			ps657.OverlayValues[386] = d386
			ps657.OverlayValues[387] = d387
			ps657.OverlayValues[388] = d388
			ps657.OverlayValues[389] = d389
			ps657.OverlayValues[391] = d391
			ps657.OverlayValues[392] = d392
			ps657.OverlayValues[393] = d393
			ps657.OverlayValues[394] = d394
			ps657.OverlayValues[395] = d395
			ps657.OverlayValues[396] = d396
			ps657.OverlayValues[397] = d397
			ps657.OverlayValues[398] = d398
			ps657.OverlayValues[400] = d400
			ps657.OverlayValues[402] = d402
			ps657.OverlayValues[403] = d403
			ps657.OverlayValues[404] = d404
			ps657.OverlayValues[508] = d508
			ps657.OverlayValues[509] = d509
			ps657.OverlayValues[512] = d512
			ps657.OverlayValues[619] = d619
			ps657.OverlayValues[620] = d620
			ps657.OverlayValues[621] = d621
			ps657.OverlayValues[622] = d622
			ps657.OverlayValues[623] = d623
			ps657.OverlayValues[625] = d625
			ps657.OverlayValues[626] = d626
			ps657.OverlayValues[627] = d627
			ps657.OverlayValues[628] = d628
			ps657.OverlayValues[629] = d629
			ps657.OverlayValues[630] = d630
			ps657.OverlayValues[631] = d631
			ps657.OverlayValues[632] = d632
			ps657.OverlayValues[633] = d633
			ps657.OverlayValues[634] = d634
			ps657.OverlayValues[635] = d635
			ps657.OverlayValues[636] = d636
			ps657.OverlayValues[637] = d637
			ps657.OverlayValues[638] = d638
			ps657.OverlayValues[639] = d639
			ps657.OverlayValues[640] = d640
			ps657.OverlayValues[641] = d641
			ps657.OverlayValues[642] = d642
			ps657.OverlayValues[643] = d643
			ps657.OverlayValues[644] = d644
			ps657.OverlayValues[645] = d645
			ps657.OverlayValues[646] = d646
			ps657.OverlayValues[647] = d647
			ps657.OverlayValues[648] = d648
			ps657.OverlayValues[649] = d649
			ps657.OverlayValues[650] = d650
			ps657.OverlayValues[651] = d651
			ps657.OverlayValues[652] = d652
			ps657.OverlayValues[653] = d653
			ps657.OverlayValues[654] = d654
			ps657.OverlayValues[655] = d655
			return bbs[9].RenderPS(ps657)
		}
		if !ps.General {
			ps.General = true
			return bbs[6].RenderPS(ps)
		}
		lbl24 := ctx.ReserveLabel()
		lbl25 := ctx.ReserveLabel()
		ctx.EmitJump(d655.Condition, lbl24)
		ctx.EmitJmp(lbl25)
		snap658 := d5
		snap659 := d6
		snap660 := d7
		snap661 := d8
		snap662 := d9
		snap663 := d10
		snap664 := d11
		snap665 := d12
		snap666 := d13
		snap667 := d14
		snap668 := d15
		snap669 := d16
		snap670 := d17
		snap671 := d18
		snap672 := d19
		snap673 := d20
		snap674 := d21
		snap675 := d23
		snap676 := d24
		snap677 := d25
		snap678 := d26
		snap679 := d27
		snap680 := d28
		snap681 := d29
		snap682 := d30
		snap683 := d31
		snap684 := d32
		snap685 := d33
		snap686 := d34
		snap687 := d35
		snap688 := d36
		snap689 := d37
		snap690 := d38
		snap691 := d39
		snap692 := d40
		snap693 := d41
		snap694 := d42
		snap695 := d43
		snap696 := d44
		snap697 := d45
		snap698 := d46
		snap699 := d47
		snap700 := d48
		snap701 := d49
		snap702 := d50
		snap703 := d51
		snap704 := d52
		snap705 := d53
		snap706 := d54
		snap707 := d55
		snap708 := d56
		snap709 := d57
		snap710 := d60
		snap711 := d61
		snap712 := d62
		snap713 := d177
		snap714 := d178
		snap715 := d179
		snap716 := d180
		snap717 := d181
		snap718 := d182
		snap719 := d183
		snap720 := d184
		snap721 := d185
		snap722 := d186
		snap723 := d187
		snap724 := d188
		snap725 := d189
		snap726 := d190
		snap727 := d191
		snap728 := d192
		snap729 := d193
		snap730 := d194
		snap731 := d195
		snap732 := d196
		snap733 := d197
		snap734 := d198
		snap735 := d199
		snap736 := d200
		snap737 := d201
		snap738 := d202
		snap739 := d203
		snap740 := d204
		snap741 := d205
		snap742 := d206
		snap743 := d209
		snap744 := d386
		snap745 := d387
		snap746 := d388
		snap747 := d389
		snap748 := d391
		snap749 := d392
		snap750 := d393
		snap751 := d394
		snap752 := d395
		snap753 := d396
		snap754 := d397
		snap755 := d398
		snap756 := d400
		snap757 := d402
		snap758 := d403
		snap759 := d404
		snap760 := d508
		snap761 := d509
		snap762 := d512
		snap763 := d619
		snap764 := d620
		snap765 := d621
		snap766 := d622
		snap767 := d623
		snap768 := d625
		snap769 := d626
		snap770 := d627
		snap771 := d628
		snap772 := d629
		snap773 := d630
		snap774 := d631
		snap775 := d632
		snap776 := d633
		snap777 := d634
		snap778 := d635
		snap779 := d636
		snap780 := d637
		snap781 := d638
		snap782 := d639
		snap783 := d640
		snap784 := d641
		snap785 := d642
		snap786 := d643
		snap787 := d644
		snap788 := d645
		snap789 := d646
		snap790 := d647
		snap791 := d648
		snap792 := d649
		snap793 := d650
		snap794 := d651
		snap795 := d652
		snap796 := d653
		snap797 := d654
		snap798 := d655
		alloc799 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl24)
		ctx.EmitJmp(lbl8)
		ctx.RestoreAllocState(alloc799)
		d5 = snap658
		d6 = snap659
		d7 = snap660
		d8 = snap661
		d9 = snap662
		d10 = snap663
		d11 = snap664
		d12 = snap665
		d13 = snap666
		d14 = snap667
		d15 = snap668
		d16 = snap669
		d17 = snap670
		d18 = snap671
		d19 = snap672
		d20 = snap673
		d21 = snap674
		d23 = snap675
		d24 = snap676
		d25 = snap677
		d26 = snap678
		d27 = snap679
		d28 = snap680
		d29 = snap681
		d30 = snap682
		d31 = snap683
		d32 = snap684
		d33 = snap685
		d34 = snap686
		d35 = snap687
		d36 = snap688
		d37 = snap689
		d38 = snap690
		d39 = snap691
		d40 = snap692
		d41 = snap693
		d42 = snap694
		d43 = snap695
		d44 = snap696
		d45 = snap697
		d46 = snap698
		d47 = snap699
		d48 = snap700
		d49 = snap701
		d50 = snap702
		d51 = snap703
		d52 = snap704
		d53 = snap705
		d54 = snap706
		d55 = snap707
		d56 = snap708
		d57 = snap709
		d60 = snap710
		d61 = snap711
		d62 = snap712
		d177 = snap713
		d178 = snap714
		d179 = snap715
		d180 = snap716
		d181 = snap717
		d182 = snap718
		d183 = snap719
		d184 = snap720
		d185 = snap721
		d186 = snap722
		d187 = snap723
		d188 = snap724
		d189 = snap725
		d190 = snap726
		d191 = snap727
		d192 = snap728
		d193 = snap729
		d194 = snap730
		d195 = snap731
		d196 = snap732
		d197 = snap733
		d198 = snap734
		d199 = snap735
		d200 = snap736
		d201 = snap737
		d202 = snap738
		d203 = snap739
		d204 = snap740
		d205 = snap741
		d206 = snap742
		d209 = snap743
		d386 = snap744
		d387 = snap745
		d388 = snap746
		d389 = snap747
		d391 = snap748
		d392 = snap749
		d393 = snap750
		d394 = snap751
		d395 = snap752
		d396 = snap753
		d397 = snap754
		d398 = snap755
		d400 = snap756
		d402 = snap757
		d403 = snap758
		d404 = snap759
		d508 = snap760
		d509 = snap761
		d512 = snap762
		d619 = snap763
		d620 = snap764
		d621 = snap765
		d622 = snap766
		d623 = snap767
		d625 = snap768
		d626 = snap769
		d627 = snap770
		d628 = snap771
		d629 = snap772
		d630 = snap773
		d631 = snap774
		d632 = snap775
		d633 = snap776
		d634 = snap777
		d635 = snap778
		d636 = snap779
		d637 = snap780
		d638 = snap781
		d639 = snap782
		d640 = snap783
		d641 = snap784
		d642 = snap785
		d643 = snap786
		d644 = snap787
		d645 = snap788
		d646 = snap789
		d647 = snap790
		d648 = snap791
		d649 = snap792
		d650 = snap793
		d651 = snap794
		d652 = snap795
		d653 = snap796
		d654 = snap797
		d655 = snap798
		ctx.MarkLabel(lbl25)
		ctx.EmitJmp(lbl10)
		ctx.RestoreAllocState(alloc799)
		d5 = snap658
		d6 = snap659
		d7 = snap660
		d8 = snap661
		d9 = snap662
		d10 = snap663
		d11 = snap664
		d12 = snap665
		d13 = snap666
		d14 = snap667
		d15 = snap668
		d16 = snap669
		d17 = snap670
		d18 = snap671
		d19 = snap672
		d20 = snap673
		d21 = snap674
		d23 = snap675
		d24 = snap676
		d25 = snap677
		d26 = snap678
		d27 = snap679
		d28 = snap680
		d29 = snap681
		d30 = snap682
		d31 = snap683
		d32 = snap684
		d33 = snap685
		d34 = snap686
		d35 = snap687
		d36 = snap688
		d37 = snap689
		d38 = snap690
		d39 = snap691
		d40 = snap692
		d41 = snap693
		d42 = snap694
		d43 = snap695
		d44 = snap696
		d45 = snap697
		d46 = snap698
		d47 = snap699
		d48 = snap700
		d49 = snap701
		d50 = snap702
		d51 = snap703
		d52 = snap704
		d53 = snap705
		d54 = snap706
		d55 = snap707
		d56 = snap708
		d57 = snap709
		d60 = snap710
		d61 = snap711
		d62 = snap712
		d177 = snap713
		d178 = snap714
		d179 = snap715
		d180 = snap716
		d181 = snap717
		d182 = snap718
		d183 = snap719
		d184 = snap720
		d185 = snap721
		d186 = snap722
		d187 = snap723
		d188 = snap724
		d189 = snap725
		d190 = snap726
		d191 = snap727
		d192 = snap728
		d193 = snap729
		d194 = snap730
		d195 = snap731
		d196 = snap732
		d197 = snap733
		d198 = snap734
		d199 = snap735
		d200 = snap736
		d201 = snap737
		d202 = snap738
		d203 = snap739
		d204 = snap740
		d205 = snap741
		d206 = snap742
		d209 = snap743
		d386 = snap744
		d387 = snap745
		d388 = snap746
		d389 = snap747
		d391 = snap748
		d392 = snap749
		d393 = snap750
		d394 = snap751
		d395 = snap752
		d396 = snap753
		d397 = snap754
		d398 = snap755
		d400 = snap756
		d402 = snap757
		d403 = snap758
		d404 = snap759
		d508 = snap760
		d509 = snap761
		d512 = snap762
		d619 = snap763
		d620 = snap764
		d621 = snap765
		d622 = snap766
		d623 = snap767
		d625 = snap768
		d626 = snap769
		d627 = snap770
		d628 = snap771
		d629 = snap772
		d630 = snap773
		d631 = snap774
		d632 = snap775
		d633 = snap776
		d634 = snap777
		d635 = snap778
		d636 = snap779
		d637 = snap780
		d638 = snap781
		d639 = snap782
		d640 = snap783
		d641 = snap784
		d642 = snap785
		d643 = snap786
		d644 = snap787
		d645 = snap788
		d646 = snap789
		d647 = snap790
		d648 = snap791
		d649 = snap792
		d650 = snap793
		d651 = snap794
		d652 = snap795
		d653 = snap796
		d654 = snap797
		d655 = snap798
		ps800 := scm.PhiState{General: true}
		ps800.OverlayValues = make([]scm.JITValueDesc, 656)
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
		ps800.OverlayValues[57] = d57
		ps800.OverlayValues[60] = d60
		ps800.OverlayValues[61] = d61
		ps800.OverlayValues[62] = d62
		ps800.OverlayValues[177] = d177
		ps800.OverlayValues[178] = d178
		ps800.OverlayValues[179] = d179
		ps800.OverlayValues[180] = d180
		ps800.OverlayValues[181] = d181
		ps800.OverlayValues[182] = d182
		ps800.OverlayValues[183] = d183
		ps800.OverlayValues[184] = d184
		ps800.OverlayValues[185] = d185
		ps800.OverlayValues[186] = d186
		ps800.OverlayValues[187] = d187
		ps800.OverlayValues[188] = d188
		ps800.OverlayValues[189] = d189
		ps800.OverlayValues[190] = d190
		ps800.OverlayValues[191] = d191
		ps800.OverlayValues[192] = d192
		ps800.OverlayValues[193] = d193
		ps800.OverlayValues[194] = d194
		ps800.OverlayValues[195] = d195
		ps800.OverlayValues[196] = d196
		ps800.OverlayValues[197] = d197
		ps800.OverlayValues[198] = d198
		ps800.OverlayValues[199] = d199
		ps800.OverlayValues[200] = d200
		ps800.OverlayValues[201] = d201
		ps800.OverlayValues[202] = d202
		ps800.OverlayValues[203] = d203
		ps800.OverlayValues[204] = d204
		ps800.OverlayValues[205] = d205
		ps800.OverlayValues[206] = d206
		ps800.OverlayValues[209] = d209
		ps800.OverlayValues[386] = d386
		ps800.OverlayValues[387] = d387
		ps800.OverlayValues[388] = d388
		ps800.OverlayValues[389] = d389
		ps800.OverlayValues[391] = d391
		ps800.OverlayValues[392] = d392
		ps800.OverlayValues[393] = d393
		ps800.OverlayValues[394] = d394
		ps800.OverlayValues[395] = d395
		ps800.OverlayValues[396] = d396
		ps800.OverlayValues[397] = d397
		ps800.OverlayValues[398] = d398
		ps800.OverlayValues[400] = d400
		ps800.OverlayValues[402] = d402
		ps800.OverlayValues[403] = d403
		ps800.OverlayValues[404] = d404
		ps800.OverlayValues[508] = d508
		ps800.OverlayValues[509] = d509
		ps800.OverlayValues[512] = d512
		ps800.OverlayValues[619] = d619
		ps800.OverlayValues[620] = d620
		ps800.OverlayValues[621] = d621
		ps800.OverlayValues[622] = d622
		ps800.OverlayValues[623] = d623
		ps800.OverlayValues[625] = d625
		ps800.OverlayValues[626] = d626
		ps800.OverlayValues[627] = d627
		ps800.OverlayValues[628] = d628
		ps800.OverlayValues[629] = d629
		ps800.OverlayValues[630] = d630
		ps800.OverlayValues[631] = d631
		ps800.OverlayValues[632] = d632
		ps800.OverlayValues[633] = d633
		ps800.OverlayValues[634] = d634
		ps800.OverlayValues[635] = d635
		ps800.OverlayValues[636] = d636
		ps800.OverlayValues[637] = d637
		ps800.OverlayValues[638] = d638
		ps800.OverlayValues[639] = d639
		ps800.OverlayValues[640] = d640
		ps800.OverlayValues[641] = d641
		ps800.OverlayValues[642] = d642
		ps800.OverlayValues[643] = d643
		ps800.OverlayValues[644] = d644
		ps800.OverlayValues[645] = d645
		ps800.OverlayValues[646] = d646
		ps800.OverlayValues[647] = d647
		ps800.OverlayValues[648] = d648
		ps800.OverlayValues[649] = d649
		ps800.OverlayValues[650] = d650
		ps800.OverlayValues[651] = d651
		ps800.OverlayValues[652] = d652
		ps800.OverlayValues[653] = d653
		ps800.OverlayValues[654] = d654
		ps800.OverlayValues[655] = d655
		ps801 := scm.PhiState{General: true}
		ps801.OverlayValues = make([]scm.JITValueDesc, 656)
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
		ps801.OverlayValues[57] = d57
		ps801.OverlayValues[60] = d60
		ps801.OverlayValues[61] = d61
		ps801.OverlayValues[62] = d62
		ps801.OverlayValues[177] = d177
		ps801.OverlayValues[178] = d178
		ps801.OverlayValues[179] = d179
		ps801.OverlayValues[180] = d180
		ps801.OverlayValues[181] = d181
		ps801.OverlayValues[182] = d182
		ps801.OverlayValues[183] = d183
		ps801.OverlayValues[184] = d184
		ps801.OverlayValues[185] = d185
		ps801.OverlayValues[186] = d186
		ps801.OverlayValues[187] = d187
		ps801.OverlayValues[188] = d188
		ps801.OverlayValues[189] = d189
		ps801.OverlayValues[190] = d190
		ps801.OverlayValues[191] = d191
		ps801.OverlayValues[192] = d192
		ps801.OverlayValues[193] = d193
		ps801.OverlayValues[194] = d194
		ps801.OverlayValues[195] = d195
		ps801.OverlayValues[196] = d196
		ps801.OverlayValues[197] = d197
		ps801.OverlayValues[198] = d198
		ps801.OverlayValues[199] = d199
		ps801.OverlayValues[200] = d200
		ps801.OverlayValues[201] = d201
		ps801.OverlayValues[202] = d202
		ps801.OverlayValues[203] = d203
		ps801.OverlayValues[204] = d204
		ps801.OverlayValues[205] = d205
		ps801.OverlayValues[206] = d206
		ps801.OverlayValues[209] = d209
		ps801.OverlayValues[386] = d386
		ps801.OverlayValues[387] = d387
		ps801.OverlayValues[388] = d388
		ps801.OverlayValues[389] = d389
		ps801.OverlayValues[391] = d391
		ps801.OverlayValues[392] = d392
		ps801.OverlayValues[393] = d393
		ps801.OverlayValues[394] = d394
		ps801.OverlayValues[395] = d395
		ps801.OverlayValues[396] = d396
		ps801.OverlayValues[397] = d397
		ps801.OverlayValues[398] = d398
		ps801.OverlayValues[400] = d400
		ps801.OverlayValues[402] = d402
		ps801.OverlayValues[403] = d403
		ps801.OverlayValues[404] = d404
		ps801.OverlayValues[508] = d508
		ps801.OverlayValues[509] = d509
		ps801.OverlayValues[512] = d512
		ps801.OverlayValues[619] = d619
		ps801.OverlayValues[620] = d620
		ps801.OverlayValues[621] = d621
		ps801.OverlayValues[622] = d622
		ps801.OverlayValues[623] = d623
		ps801.OverlayValues[625] = d625
		ps801.OverlayValues[626] = d626
		ps801.OverlayValues[627] = d627
		ps801.OverlayValues[628] = d628
		ps801.OverlayValues[629] = d629
		ps801.OverlayValues[630] = d630
		ps801.OverlayValues[631] = d631
		ps801.OverlayValues[632] = d632
		ps801.OverlayValues[633] = d633
		ps801.OverlayValues[634] = d634
		ps801.OverlayValues[635] = d635
		ps801.OverlayValues[636] = d636
		ps801.OverlayValues[637] = d637
		ps801.OverlayValues[638] = d638
		ps801.OverlayValues[639] = d639
		ps801.OverlayValues[640] = d640
		ps801.OverlayValues[641] = d641
		ps801.OverlayValues[642] = d642
		ps801.OverlayValues[643] = d643
		ps801.OverlayValues[644] = d644
		ps801.OverlayValues[645] = d645
		ps801.OverlayValues[646] = d646
		ps801.OverlayValues[647] = d647
		ps801.OverlayValues[648] = d648
		ps801.OverlayValues[649] = d649
		ps801.OverlayValues[650] = d650
		ps801.OverlayValues[651] = d651
		ps801.OverlayValues[652] = d652
		ps801.OverlayValues[653] = d653
		ps801.OverlayValues[654] = d654
		ps801.OverlayValues[655] = d655
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
		snap853 := d57
		snap854 := d60
		snap855 := d61
		snap856 := d62
		snap857 := d177
		snap858 := d178
		snap859 := d179
		snap860 := d180
		snap861 := d181
		snap862 := d182
		snap863 := d183
		snap864 := d184
		snap865 := d185
		snap866 := d186
		snap867 := d187
		snap868 := d188
		snap869 := d189
		snap870 := d190
		snap871 := d191
		snap872 := d192
		snap873 := d193
		snap874 := d194
		snap875 := d195
		snap876 := d196
		snap877 := d197
		snap878 := d198
		snap879 := d199
		snap880 := d200
		snap881 := d201
		snap882 := d202
		snap883 := d203
		snap884 := d204
		snap885 := d205
		snap886 := d206
		snap887 := d209
		snap888 := d386
		snap889 := d387
		snap890 := d388
		snap891 := d389
		snap892 := d391
		snap893 := d392
		snap894 := d393
		snap895 := d394
		snap896 := d395
		snap897 := d396
		snap898 := d397
		snap899 := d398
		snap900 := d400
		snap901 := d402
		snap902 := d403
		snap903 := d404
		snap904 := d508
		snap905 := d509
		snap906 := d512
		snap907 := d619
		snap908 := d620
		snap909 := d621
		snap910 := d622
		snap911 := d623
		snap912 := d625
		snap913 := d626
		snap914 := d627
		snap915 := d628
		snap916 := d629
		snap917 := d630
		snap918 := d631
		snap919 := d632
		snap920 := d633
		snap921 := d634
		snap922 := d635
		snap923 := d636
		snap924 := d637
		snap925 := d638
		snap926 := d639
		snap927 := d640
		snap928 := d641
		snap929 := d642
		snap930 := d643
		snap931 := d644
		snap932 := d645
		snap933 := d646
		snap934 := d647
		snap935 := d648
		snap936 := d649
		snap937 := d650
		snap938 := d651
		snap939 := d652
		snap940 := d653
		snap941 := d654
		snap942 := d655
		alloc943 := ctx.SnapshotAllocState()
		if !bbs[9].Rendered {
			bbs[9].RenderPS(ps801)
		}
		ctx.RestoreAllocState(alloc943)
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
		d57 = snap853
		d60 = snap854
		d61 = snap855
		d62 = snap856
		d177 = snap857
		d178 = snap858
		d179 = snap859
		d180 = snap860
		d181 = snap861
		d182 = snap862
		d183 = snap863
		d184 = snap864
		d185 = snap865
		d186 = snap866
		d187 = snap867
		d188 = snap868
		d189 = snap869
		d190 = snap870
		d191 = snap871
		d192 = snap872
		d193 = snap873
		d194 = snap874
		d195 = snap875
		d196 = snap876
		d197 = snap877
		d198 = snap878
		d199 = snap879
		d200 = snap880
		d201 = snap881
		d202 = snap882
		d203 = snap883
		d204 = snap884
		d205 = snap885
		d206 = snap886
		d209 = snap887
		d386 = snap888
		d387 = snap889
		d388 = snap890
		d389 = snap891
		d391 = snap892
		d392 = snap893
		d393 = snap894
		d394 = snap895
		d395 = snap896
		d396 = snap897
		d397 = snap898
		d398 = snap899
		d400 = snap900
		d402 = snap901
		d403 = snap902
		d404 = snap903
		d508 = snap904
		d509 = snap905
		d512 = snap906
		d619 = snap907
		d620 = snap908
		d621 = snap909
		d622 = snap910
		d623 = snap911
		d625 = snap912
		d626 = snap913
		d627 = snap914
		d628 = snap915
		d629 = snap916
		d630 = snap917
		d631 = snap918
		d632 = snap919
		d633 = snap920
		d634 = snap921
		d635 = snap922
		d636 = snap923
		d637 = snap924
		d638 = snap925
		d639 = snap926
		d640 = snap927
		d641 = snap928
		d642 = snap929
		d643 = snap930
		d644 = snap931
		d645 = snap932
		d646 = snap933
		d647 = snap934
		d648 = snap935
		d649 = snap936
		d650 = snap937
		d651 = snap938
		d652 = snap939
		d653 = snap940
		d654 = snap941
		d655 = snap942
		if !bbs[7].Rendered {
			return bbs[7].RenderPS(ps800)
		}
		return result
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
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
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
		if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != scm.LocNone {
			d177 = ps.OverlayValues[177]
		}
		if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
			d178 = ps.OverlayValues[178]
		}
		if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != scm.LocNone {
			d179 = ps.OverlayValues[179]
		}
		if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != scm.LocNone {
			d180 = ps.OverlayValues[180]
		}
		if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != scm.LocNone {
			d181 = ps.OverlayValues[181]
		}
		if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != scm.LocNone {
			d182 = ps.OverlayValues[182]
		}
		if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != scm.LocNone {
			d183 = ps.OverlayValues[183]
		}
		if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
			d184 = ps.OverlayValues[184]
		}
		if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != scm.LocNone {
			d185 = ps.OverlayValues[185]
		}
		if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != scm.LocNone {
			d186 = ps.OverlayValues[186]
		}
		if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
			d187 = ps.OverlayValues[187]
		}
		if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != scm.LocNone {
			d188 = ps.OverlayValues[188]
		}
		if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != scm.LocNone {
			d189 = ps.OverlayValues[189]
		}
		if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != scm.LocNone {
			d190 = ps.OverlayValues[190]
		}
		if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != scm.LocNone {
			d191 = ps.OverlayValues[191]
		}
		if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != scm.LocNone {
			d192 = ps.OverlayValues[192]
		}
		if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != scm.LocNone {
			d193 = ps.OverlayValues[193]
		}
		if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != scm.LocNone {
			d194 = ps.OverlayValues[194]
		}
		if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != scm.LocNone {
			d195 = ps.OverlayValues[195]
		}
		if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
			d196 = ps.OverlayValues[196]
		}
		if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != scm.LocNone {
			d197 = ps.OverlayValues[197]
		}
		if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != scm.LocNone {
			d198 = ps.OverlayValues[198]
		}
		if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
			d199 = ps.OverlayValues[199]
		}
		if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
			d200 = ps.OverlayValues[200]
		}
		if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
			d201 = ps.OverlayValues[201]
		}
		if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != scm.LocNone {
			d202 = ps.OverlayValues[202]
		}
		if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != scm.LocNone {
			d203 = ps.OverlayValues[203]
		}
		if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != scm.LocNone {
			d204 = ps.OverlayValues[204]
		}
		if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != scm.LocNone {
			d205 = ps.OverlayValues[205]
		}
		if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != scm.LocNone {
			d206 = ps.OverlayValues[206]
		}
		if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != scm.LocNone {
			d209 = ps.OverlayValues[209]
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
		if len(ps.OverlayValues) > 400 && ps.OverlayValues[400].Loc != scm.LocNone {
			d400 = ps.OverlayValues[400]
		}
		if len(ps.OverlayValues) > 402 && ps.OverlayValues[402].Loc != scm.LocNone {
			d402 = ps.OverlayValues[402]
		}
		if len(ps.OverlayValues) > 403 && ps.OverlayValues[403].Loc != scm.LocNone {
			d403 = ps.OverlayValues[403]
		}
		if len(ps.OverlayValues) > 404 && ps.OverlayValues[404].Loc != scm.LocNone {
			d404 = ps.OverlayValues[404]
		}
		if len(ps.OverlayValues) > 508 && ps.OverlayValues[508].Loc != scm.LocNone {
			d508 = ps.OverlayValues[508]
		}
		if len(ps.OverlayValues) > 509 && ps.OverlayValues[509].Loc != scm.LocNone {
			d509 = ps.OverlayValues[509]
		}
		if len(ps.OverlayValues) > 512 && ps.OverlayValues[512].Loc != scm.LocNone {
			d512 = ps.OverlayValues[512]
		}
		if len(ps.OverlayValues) > 619 && ps.OverlayValues[619].Loc != scm.LocNone {
			d619 = ps.OverlayValues[619]
		}
		if len(ps.OverlayValues) > 620 && ps.OverlayValues[620].Loc != scm.LocNone {
			d620 = ps.OverlayValues[620]
		}
		if len(ps.OverlayValues) > 621 && ps.OverlayValues[621].Loc != scm.LocNone {
			d621 = ps.OverlayValues[621]
		}
		if len(ps.OverlayValues) > 622 && ps.OverlayValues[622].Loc != scm.LocNone {
			d622 = ps.OverlayValues[622]
		}
		if len(ps.OverlayValues) > 623 && ps.OverlayValues[623].Loc != scm.LocNone {
			d623 = ps.OverlayValues[623]
		}
		if len(ps.OverlayValues) > 625 && ps.OverlayValues[625].Loc != scm.LocNone {
			d625 = ps.OverlayValues[625]
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
		if len(ps.OverlayValues) > 629 && ps.OverlayValues[629].Loc != scm.LocNone {
			d629 = ps.OverlayValues[629]
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
		if len(ps.OverlayValues) > 637 && ps.OverlayValues[637].Loc != scm.LocNone {
			d637 = ps.OverlayValues[637]
		}
		if len(ps.OverlayValues) > 638 && ps.OverlayValues[638].Loc != scm.LocNone {
			d638 = ps.OverlayValues[638]
		}
		if len(ps.OverlayValues) > 639 && ps.OverlayValues[639].Loc != scm.LocNone {
			d639 = ps.OverlayValues[639]
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
		if len(ps.OverlayValues) > 644 && ps.OverlayValues[644].Loc != scm.LocNone {
			d644 = ps.OverlayValues[644]
		}
		if len(ps.OverlayValues) > 645 && ps.OverlayValues[645].Loc != scm.LocNone {
			d645 = ps.OverlayValues[645]
		}
		if len(ps.OverlayValues) > 646 && ps.OverlayValues[646].Loc != scm.LocNone {
			d646 = ps.OverlayValues[646]
		}
		if len(ps.OverlayValues) > 647 && ps.OverlayValues[647].Loc != scm.LocNone {
			d647 = ps.OverlayValues[647]
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
		if len(ps.OverlayValues) > 651 && ps.OverlayValues[651].Loc != scm.LocNone {
			d651 = ps.OverlayValues[651]
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d9)
		ctx.EnsureDesc(&d9)
		var d944 scm.JITValueDesc
		if d9.Loc == scm.LocImm {
			d944 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d9.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d9.Reg)
			ctx.EmitMovRegReg(scratch, d9.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d944 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d944)
		}
		if d944.Loc == scm.LocImm {
			d944 = scm.JITValueDesc{Loc: scm.LocImm, Type: d944.Type, Imm: scm.NewInt(int64(uint64(d944.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d944.Reg, 32)
			ctx.EmitShrRegImm8(d944.Reg, 32)
		}
		if d944.Loc == scm.LocReg && d9.Loc == scm.LocReg && d944.Reg == d9.Reg {
			ctx.TransferReg(d9.Reg)
			d9.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d944)
		ctx.EmitStoreToStack(d944, int32(bbs[8].PhiBase)+int32(16))
		ctx.StabilizeDescForControlFlow(&d944)
		if ps.General {
			ctx.SyncDesc(&d10)
			if d10.Loc == scm.LocReg {
				ctx.ProtectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.ProtectReg(d10.Reg)
				ctx.ProtectReg(d10.Reg2)
			}
			d945 = d10
			if d945.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d945)
			d946 = d945
			if d946.Loc == scm.LocImm {
				d946 = scm.JITValueDesc{Loc: scm.LocImm, Type: d946.Type, Imm: scm.NewInt(int64(uint64(d946.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d946.Reg, 32)
				ctx.EmitShrRegImm8(d946.Reg, 32)
			}
			ctx.EmitStoreToStack(d946, int32(bbs[8].PhiBase)+int32(0))
			if d10.Loc == scm.LocReg {
				ctx.UnprotectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d10.Reg)
				ctx.UnprotectReg(d10.Reg2)
			}
		}
		ps947 := scm.PhiState{General: ps.General}
		ps947.OverlayValues = make([]scm.JITValueDesc, 947)
		ps947.OverlayValues[5] = d5
		ps947.OverlayValues[6] = d6
		ps947.OverlayValues[7] = d7
		ps947.OverlayValues[8] = d8
		ps947.OverlayValues[9] = d9
		ps947.OverlayValues[10] = d10
		ps947.OverlayValues[11] = d11
		ps947.OverlayValues[12] = d12
		ps947.OverlayValues[13] = d13
		ps947.OverlayValues[14] = d14
		ps947.OverlayValues[15] = d15
		ps947.OverlayValues[16] = d16
		ps947.OverlayValues[17] = d17
		ps947.OverlayValues[18] = d18
		ps947.OverlayValues[19] = d19
		ps947.OverlayValues[20] = d20
		ps947.OverlayValues[21] = d21
		ps947.OverlayValues[23] = d23
		ps947.OverlayValues[24] = d24
		ps947.OverlayValues[25] = d25
		ps947.OverlayValues[26] = d26
		ps947.OverlayValues[27] = d27
		ps947.OverlayValues[28] = d28
		ps947.OverlayValues[29] = d29
		ps947.OverlayValues[30] = d30
		ps947.OverlayValues[31] = d31
		ps947.OverlayValues[32] = d32
		ps947.OverlayValues[33] = d33
		ps947.OverlayValues[34] = d34
		ps947.OverlayValues[35] = d35
		ps947.OverlayValues[36] = d36
		ps947.OverlayValues[37] = d37
		ps947.OverlayValues[38] = d38
		ps947.OverlayValues[39] = d39
		ps947.OverlayValues[40] = d40
		ps947.OverlayValues[41] = d41
		ps947.OverlayValues[42] = d42
		ps947.OverlayValues[43] = d43
		ps947.OverlayValues[44] = d44
		ps947.OverlayValues[45] = d45
		ps947.OverlayValues[46] = d46
		ps947.OverlayValues[47] = d47
		ps947.OverlayValues[48] = d48
		ps947.OverlayValues[49] = d49
		ps947.OverlayValues[50] = d50
		ps947.OverlayValues[51] = d51
		ps947.OverlayValues[52] = d52
		ps947.OverlayValues[53] = d53
		ps947.OverlayValues[54] = d54
		ps947.OverlayValues[55] = d55
		ps947.OverlayValues[56] = d56
		ps947.OverlayValues[57] = d57
		ps947.OverlayValues[60] = d60
		ps947.OverlayValues[61] = d61
		ps947.OverlayValues[62] = d62
		ps947.OverlayValues[177] = d177
		ps947.OverlayValues[178] = d178
		ps947.OverlayValues[179] = d179
		ps947.OverlayValues[180] = d180
		ps947.OverlayValues[181] = d181
		ps947.OverlayValues[182] = d182
		ps947.OverlayValues[183] = d183
		ps947.OverlayValues[184] = d184
		ps947.OverlayValues[185] = d185
		ps947.OverlayValues[186] = d186
		ps947.OverlayValues[187] = d187
		ps947.OverlayValues[188] = d188
		ps947.OverlayValues[189] = d189
		ps947.OverlayValues[190] = d190
		ps947.OverlayValues[191] = d191
		ps947.OverlayValues[192] = d192
		ps947.OverlayValues[193] = d193
		ps947.OverlayValues[194] = d194
		ps947.OverlayValues[195] = d195
		ps947.OverlayValues[196] = d196
		ps947.OverlayValues[197] = d197
		ps947.OverlayValues[198] = d198
		ps947.OverlayValues[199] = d199
		ps947.OverlayValues[200] = d200
		ps947.OverlayValues[201] = d201
		ps947.OverlayValues[202] = d202
		ps947.OverlayValues[203] = d203
		ps947.OverlayValues[204] = d204
		ps947.OverlayValues[205] = d205
		ps947.OverlayValues[206] = d206
		ps947.OverlayValues[209] = d209
		ps947.OverlayValues[386] = d386
		ps947.OverlayValues[387] = d387
		ps947.OverlayValues[388] = d388
		ps947.OverlayValues[389] = d389
		ps947.OverlayValues[391] = d391
		ps947.OverlayValues[392] = d392
		ps947.OverlayValues[393] = d393
		ps947.OverlayValues[394] = d394
		ps947.OverlayValues[395] = d395
		ps947.OverlayValues[396] = d396
		ps947.OverlayValues[397] = d397
		ps947.OverlayValues[398] = d398
		ps947.OverlayValues[400] = d400
		ps947.OverlayValues[402] = d402
		ps947.OverlayValues[403] = d403
		ps947.OverlayValues[404] = d404
		ps947.OverlayValues[508] = d508
		ps947.OverlayValues[509] = d509
		ps947.OverlayValues[512] = d512
		ps947.OverlayValues[619] = d619
		ps947.OverlayValues[620] = d620
		ps947.OverlayValues[621] = d621
		ps947.OverlayValues[622] = d622
		ps947.OverlayValues[623] = d623
		ps947.OverlayValues[625] = d625
		ps947.OverlayValues[626] = d626
		ps947.OverlayValues[627] = d627
		ps947.OverlayValues[628] = d628
		ps947.OverlayValues[629] = d629
		ps947.OverlayValues[630] = d630
		ps947.OverlayValues[631] = d631
		ps947.OverlayValues[632] = d632
		ps947.OverlayValues[633] = d633
		ps947.OverlayValues[634] = d634
		ps947.OverlayValues[635] = d635
		ps947.OverlayValues[636] = d636
		ps947.OverlayValues[637] = d637
		ps947.OverlayValues[638] = d638
		ps947.OverlayValues[639] = d639
		ps947.OverlayValues[640] = d640
		ps947.OverlayValues[641] = d641
		ps947.OverlayValues[642] = d642
		ps947.OverlayValues[643] = d643
		ps947.OverlayValues[644] = d644
		ps947.OverlayValues[645] = d645
		ps947.OverlayValues[646] = d646
		ps947.OverlayValues[647] = d647
		ps947.OverlayValues[648] = d648
		ps947.OverlayValues[649] = d649
		ps947.OverlayValues[650] = d650
		ps947.OverlayValues[651] = d651
		ps947.OverlayValues[652] = d652
		ps947.OverlayValues[653] = d653
		ps947.OverlayValues[654] = d654
		ps947.OverlayValues[655] = d655
		ps947.OverlayValues[944] = d944
		ps947.OverlayValues[945] = d945
		ps947.OverlayValues[946] = d946
		ps947.PhiValues = make([]scm.JITValueDesc, 2)
		d948 = d10
		ps947.PhiValues[0] = d948
		if ps947.General && bbs[8].Rendered {
			ctx.EmitJmp(lbl9)
			return result
		}
		return bbs[8].RenderPS(ps947)
		return result
	}
	bbs[8].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d949 := ps.PhiValues[0]
				ctx.EnsureDesc(&d949)
				ctx.EmitStoreToStack(d949, int32(bbs[8].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d950 := ps.PhiValues[1]
				ctx.EnsureDesc(&d950)
				ctx.EmitStoreToStack(d950, int32(bbs[8].PhiBase)+int32(16))
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
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
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
		if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != scm.LocNone {
			d177 = ps.OverlayValues[177]
		}
		if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
			d178 = ps.OverlayValues[178]
		}
		if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != scm.LocNone {
			d179 = ps.OverlayValues[179]
		}
		if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != scm.LocNone {
			d180 = ps.OverlayValues[180]
		}
		if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != scm.LocNone {
			d181 = ps.OverlayValues[181]
		}
		if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != scm.LocNone {
			d182 = ps.OverlayValues[182]
		}
		if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != scm.LocNone {
			d183 = ps.OverlayValues[183]
		}
		if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
			d184 = ps.OverlayValues[184]
		}
		if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != scm.LocNone {
			d185 = ps.OverlayValues[185]
		}
		if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != scm.LocNone {
			d186 = ps.OverlayValues[186]
		}
		if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
			d187 = ps.OverlayValues[187]
		}
		if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != scm.LocNone {
			d188 = ps.OverlayValues[188]
		}
		if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != scm.LocNone {
			d189 = ps.OverlayValues[189]
		}
		if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != scm.LocNone {
			d190 = ps.OverlayValues[190]
		}
		if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != scm.LocNone {
			d191 = ps.OverlayValues[191]
		}
		if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != scm.LocNone {
			d192 = ps.OverlayValues[192]
		}
		if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != scm.LocNone {
			d193 = ps.OverlayValues[193]
		}
		if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != scm.LocNone {
			d194 = ps.OverlayValues[194]
		}
		if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != scm.LocNone {
			d195 = ps.OverlayValues[195]
		}
		if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
			d196 = ps.OverlayValues[196]
		}
		if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != scm.LocNone {
			d197 = ps.OverlayValues[197]
		}
		if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != scm.LocNone {
			d198 = ps.OverlayValues[198]
		}
		if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
			d199 = ps.OverlayValues[199]
		}
		if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
			d200 = ps.OverlayValues[200]
		}
		if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
			d201 = ps.OverlayValues[201]
		}
		if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != scm.LocNone {
			d202 = ps.OverlayValues[202]
		}
		if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != scm.LocNone {
			d203 = ps.OverlayValues[203]
		}
		if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != scm.LocNone {
			d204 = ps.OverlayValues[204]
		}
		if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != scm.LocNone {
			d205 = ps.OverlayValues[205]
		}
		if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != scm.LocNone {
			d206 = ps.OverlayValues[206]
		}
		if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != scm.LocNone {
			d209 = ps.OverlayValues[209]
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
		if len(ps.OverlayValues) > 400 && ps.OverlayValues[400].Loc != scm.LocNone {
			d400 = ps.OverlayValues[400]
		}
		if len(ps.OverlayValues) > 402 && ps.OverlayValues[402].Loc != scm.LocNone {
			d402 = ps.OverlayValues[402]
		}
		if len(ps.OverlayValues) > 403 && ps.OverlayValues[403].Loc != scm.LocNone {
			d403 = ps.OverlayValues[403]
		}
		if len(ps.OverlayValues) > 404 && ps.OverlayValues[404].Loc != scm.LocNone {
			d404 = ps.OverlayValues[404]
		}
		if len(ps.OverlayValues) > 508 && ps.OverlayValues[508].Loc != scm.LocNone {
			d508 = ps.OverlayValues[508]
		}
		if len(ps.OverlayValues) > 509 && ps.OverlayValues[509].Loc != scm.LocNone {
			d509 = ps.OverlayValues[509]
		}
		if len(ps.OverlayValues) > 512 && ps.OverlayValues[512].Loc != scm.LocNone {
			d512 = ps.OverlayValues[512]
		}
		if len(ps.OverlayValues) > 619 && ps.OverlayValues[619].Loc != scm.LocNone {
			d619 = ps.OverlayValues[619]
		}
		if len(ps.OverlayValues) > 620 && ps.OverlayValues[620].Loc != scm.LocNone {
			d620 = ps.OverlayValues[620]
		}
		if len(ps.OverlayValues) > 621 && ps.OverlayValues[621].Loc != scm.LocNone {
			d621 = ps.OverlayValues[621]
		}
		if len(ps.OverlayValues) > 622 && ps.OverlayValues[622].Loc != scm.LocNone {
			d622 = ps.OverlayValues[622]
		}
		if len(ps.OverlayValues) > 623 && ps.OverlayValues[623].Loc != scm.LocNone {
			d623 = ps.OverlayValues[623]
		}
		if len(ps.OverlayValues) > 625 && ps.OverlayValues[625].Loc != scm.LocNone {
			d625 = ps.OverlayValues[625]
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
		if len(ps.OverlayValues) > 629 && ps.OverlayValues[629].Loc != scm.LocNone {
			d629 = ps.OverlayValues[629]
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
		if len(ps.OverlayValues) > 637 && ps.OverlayValues[637].Loc != scm.LocNone {
			d637 = ps.OverlayValues[637]
		}
		if len(ps.OverlayValues) > 638 && ps.OverlayValues[638].Loc != scm.LocNone {
			d638 = ps.OverlayValues[638]
		}
		if len(ps.OverlayValues) > 639 && ps.OverlayValues[639].Loc != scm.LocNone {
			d639 = ps.OverlayValues[639]
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
		if len(ps.OverlayValues) > 644 && ps.OverlayValues[644].Loc != scm.LocNone {
			d644 = ps.OverlayValues[644]
		}
		if len(ps.OverlayValues) > 645 && ps.OverlayValues[645].Loc != scm.LocNone {
			d645 = ps.OverlayValues[645]
		}
		if len(ps.OverlayValues) > 646 && ps.OverlayValues[646].Loc != scm.LocNone {
			d646 = ps.OverlayValues[646]
		}
		if len(ps.OverlayValues) > 647 && ps.OverlayValues[647].Loc != scm.LocNone {
			d647 = ps.OverlayValues[647]
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
		if len(ps.OverlayValues) > 651 && ps.OverlayValues[651].Loc != scm.LocNone {
			d651 = ps.OverlayValues[651]
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
		if len(ps.OverlayValues) > 944 && ps.OverlayValues[944].Loc != scm.LocNone {
			d944 = ps.OverlayValues[944]
		}
		if len(ps.OverlayValues) > 945 && ps.OverlayValues[945].Loc != scm.LocNone {
			d945 = ps.OverlayValues[945]
		}
		if len(ps.OverlayValues) > 946 && ps.OverlayValues[946].Loc != scm.LocNone {
			d946 = ps.OverlayValues[946]
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
		var d951 scm.JITValueDesc
		if d12.Loc == scm.LocImm && d13.Loc == scm.LocImm {
			d951 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d12.Imm.Int()) == uint64(d13.Imm.Int()))}
		} else if d13.Loc == scm.LocImm {
			r105 := ctx.AllocRegExcept(d12.Reg)
			if d13.Imm.Int() >= -2147483648 && d13.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d12.Reg, int32(d13.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
				ctx.EmitCmpInt64(d12.Reg, scm.RegR11)
			}
			d951 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r105, Condition: scm.CondEqual}
			ctx.BindReg(r105, &d951)
		} else if d12.Loc == scm.LocImm {
			r106 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d12.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d13.Reg)
			d951 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r106, Condition: scm.CondEqual}
			ctx.BindReg(r106, &d951)
		} else {
			r107 := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitCmpInt64(d12.Reg, d13.Reg)
			d951 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r107, Condition: scm.CondEqual}
			ctx.BindReg(r107, &d951)
		}
		d952 = d951
		ctx.EnsureDesc(&d952)
		if d952.Loc != scm.LocImm && d952.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d952.Loc == scm.LocImm {
			if d952.Imm.Bool() {
				if ps.General {
					ctx.SyncDesc(&d12)
					if d12.Loc == scm.LocReg {
						ctx.ProtectReg(d12.Reg)
					} else if d12.Loc == scm.LocRegPair {
						ctx.ProtectReg(d12.Reg)
						ctx.ProtectReg(d12.Reg2)
					}
					d953 = d12
					if d953.Loc == scm.LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d953)
					d954 = d953
					if d954.Loc == scm.LocImm {
						d954 = scm.JITValueDesc{Loc: scm.LocImm, Type: d954.Type, Imm: scm.NewInt(int64(uint64(d954.Imm.Int()) & 0xffffffff))}
					} else {
						ctx.EmitShlRegImm8(d954.Reg, 32)
						ctx.EmitShrRegImm8(d954.Reg, 32)
					}
					ctx.EmitStoreToStack(d954, int32(bbs[2].PhiBase)+int32(0))
					if d12.Loc == scm.LocReg {
						ctx.UnprotectReg(d12.Reg)
					} else if d12.Loc == scm.LocRegPair {
						ctx.UnprotectReg(d12.Reg)
						ctx.UnprotectReg(d12.Reg2)
					}
				}
				ps955 := scm.PhiState{General: ps.General}
				ps955.OverlayValues = make([]scm.JITValueDesc, 955)
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
				ps955.OverlayValues[16] = d16
				ps955.OverlayValues[17] = d17
				ps955.OverlayValues[18] = d18
				ps955.OverlayValues[19] = d19
				ps955.OverlayValues[20] = d20
				ps955.OverlayValues[21] = d21
				ps955.OverlayValues[23] = d23
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
				ps955.OverlayValues[60] = d60
				ps955.OverlayValues[61] = d61
				ps955.OverlayValues[62] = d62
				ps955.OverlayValues[177] = d177
				ps955.OverlayValues[178] = d178
				ps955.OverlayValues[179] = d179
				ps955.OverlayValues[180] = d180
				ps955.OverlayValues[181] = d181
				ps955.OverlayValues[182] = d182
				ps955.OverlayValues[183] = d183
				ps955.OverlayValues[184] = d184
				ps955.OverlayValues[185] = d185
				ps955.OverlayValues[186] = d186
				ps955.OverlayValues[187] = d187
				ps955.OverlayValues[188] = d188
				ps955.OverlayValues[189] = d189
				ps955.OverlayValues[190] = d190
				ps955.OverlayValues[191] = d191
				ps955.OverlayValues[192] = d192
				ps955.OverlayValues[193] = d193
				ps955.OverlayValues[194] = d194
				ps955.OverlayValues[195] = d195
				ps955.OverlayValues[196] = d196
				ps955.OverlayValues[197] = d197
				ps955.OverlayValues[198] = d198
				ps955.OverlayValues[199] = d199
				ps955.OverlayValues[200] = d200
				ps955.OverlayValues[201] = d201
				ps955.OverlayValues[202] = d202
				ps955.OverlayValues[203] = d203
				ps955.OverlayValues[204] = d204
				ps955.OverlayValues[205] = d205
				ps955.OverlayValues[206] = d206
				ps955.OverlayValues[209] = d209
				ps955.OverlayValues[386] = d386
				ps955.OverlayValues[387] = d387
				ps955.OverlayValues[388] = d388
				ps955.OverlayValues[389] = d389
				ps955.OverlayValues[391] = d391
				ps955.OverlayValues[392] = d392
				ps955.OverlayValues[393] = d393
				ps955.OverlayValues[394] = d394
				ps955.OverlayValues[395] = d395
				ps955.OverlayValues[396] = d396
				ps955.OverlayValues[397] = d397
				ps955.OverlayValues[398] = d398
				ps955.OverlayValues[400] = d400
				ps955.OverlayValues[402] = d402
				ps955.OverlayValues[403] = d403
				ps955.OverlayValues[404] = d404
				ps955.OverlayValues[508] = d508
				ps955.OverlayValues[509] = d509
				ps955.OverlayValues[512] = d512
				ps955.OverlayValues[619] = d619
				ps955.OverlayValues[620] = d620
				ps955.OverlayValues[621] = d621
				ps955.OverlayValues[622] = d622
				ps955.OverlayValues[623] = d623
				ps955.OverlayValues[625] = d625
				ps955.OverlayValues[626] = d626
				ps955.OverlayValues[627] = d627
				ps955.OverlayValues[628] = d628
				ps955.OverlayValues[629] = d629
				ps955.OverlayValues[630] = d630
				ps955.OverlayValues[631] = d631
				ps955.OverlayValues[632] = d632
				ps955.OverlayValues[633] = d633
				ps955.OverlayValues[634] = d634
				ps955.OverlayValues[635] = d635
				ps955.OverlayValues[636] = d636
				ps955.OverlayValues[637] = d637
				ps955.OverlayValues[638] = d638
				ps955.OverlayValues[639] = d639
				ps955.OverlayValues[640] = d640
				ps955.OverlayValues[641] = d641
				ps955.OverlayValues[642] = d642
				ps955.OverlayValues[643] = d643
				ps955.OverlayValues[644] = d644
				ps955.OverlayValues[645] = d645
				ps955.OverlayValues[646] = d646
				ps955.OverlayValues[647] = d647
				ps955.OverlayValues[648] = d648
				ps955.OverlayValues[649] = d649
				ps955.OverlayValues[650] = d650
				ps955.OverlayValues[651] = d651
				ps955.OverlayValues[652] = d652
				ps955.OverlayValues[653] = d653
				ps955.OverlayValues[654] = d654
				ps955.OverlayValues[655] = d655
				ps955.OverlayValues[944] = d944
				ps955.OverlayValues[945] = d945
				ps955.OverlayValues[946] = d946
				ps955.OverlayValues[948] = d948
				ps955.OverlayValues[949] = d949
				ps955.OverlayValues[950] = d950
				ps955.OverlayValues[951] = d951
				ps955.OverlayValues[952] = d952
				ps955.OverlayValues[953] = d953
				ps955.OverlayValues[954] = d954
				ps955.PhiValues = make([]scm.JITValueDesc, 1)
				d956 = d12
				ps955.PhiValues[0] = d956
				return bbs[2].RenderPS(ps955)
			}
			if ps.General {
			}
			ps957 := scm.PhiState{General: ps.General}
			ps957.OverlayValues = make([]scm.JITValueDesc, 957)
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
			ps957.OverlayValues[16] = d16
			ps957.OverlayValues[17] = d17
			ps957.OverlayValues[18] = d18
			ps957.OverlayValues[19] = d19
			ps957.OverlayValues[20] = d20
			ps957.OverlayValues[21] = d21
			ps957.OverlayValues[23] = d23
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
			ps957.OverlayValues[60] = d60
			ps957.OverlayValues[61] = d61
			ps957.OverlayValues[62] = d62
			ps957.OverlayValues[177] = d177
			ps957.OverlayValues[178] = d178
			ps957.OverlayValues[179] = d179
			ps957.OverlayValues[180] = d180
			ps957.OverlayValues[181] = d181
			ps957.OverlayValues[182] = d182
			ps957.OverlayValues[183] = d183
			ps957.OverlayValues[184] = d184
			ps957.OverlayValues[185] = d185
			ps957.OverlayValues[186] = d186
			ps957.OverlayValues[187] = d187
			ps957.OverlayValues[188] = d188
			ps957.OverlayValues[189] = d189
			ps957.OverlayValues[190] = d190
			ps957.OverlayValues[191] = d191
			ps957.OverlayValues[192] = d192
			ps957.OverlayValues[193] = d193
			ps957.OverlayValues[194] = d194
			ps957.OverlayValues[195] = d195
			ps957.OverlayValues[196] = d196
			ps957.OverlayValues[197] = d197
			ps957.OverlayValues[198] = d198
			ps957.OverlayValues[199] = d199
			ps957.OverlayValues[200] = d200
			ps957.OverlayValues[201] = d201
			ps957.OverlayValues[202] = d202
			ps957.OverlayValues[203] = d203
			ps957.OverlayValues[204] = d204
			ps957.OverlayValues[205] = d205
			ps957.OverlayValues[206] = d206
			ps957.OverlayValues[209] = d209
			ps957.OverlayValues[386] = d386
			ps957.OverlayValues[387] = d387
			ps957.OverlayValues[388] = d388
			ps957.OverlayValues[389] = d389
			ps957.OverlayValues[391] = d391
			ps957.OverlayValues[392] = d392
			ps957.OverlayValues[393] = d393
			ps957.OverlayValues[394] = d394
			ps957.OverlayValues[395] = d395
			ps957.OverlayValues[396] = d396
			ps957.OverlayValues[397] = d397
			ps957.OverlayValues[398] = d398
			ps957.OverlayValues[400] = d400
			ps957.OverlayValues[402] = d402
			ps957.OverlayValues[403] = d403
			ps957.OverlayValues[404] = d404
			ps957.OverlayValues[508] = d508
			ps957.OverlayValues[509] = d509
			ps957.OverlayValues[512] = d512
			ps957.OverlayValues[619] = d619
			ps957.OverlayValues[620] = d620
			ps957.OverlayValues[621] = d621
			ps957.OverlayValues[622] = d622
			ps957.OverlayValues[623] = d623
			ps957.OverlayValues[625] = d625
			ps957.OverlayValues[626] = d626
			ps957.OverlayValues[627] = d627
			ps957.OverlayValues[628] = d628
			ps957.OverlayValues[629] = d629
			ps957.OverlayValues[630] = d630
			ps957.OverlayValues[631] = d631
			ps957.OverlayValues[632] = d632
			ps957.OverlayValues[633] = d633
			ps957.OverlayValues[634] = d634
			ps957.OverlayValues[635] = d635
			ps957.OverlayValues[636] = d636
			ps957.OverlayValues[637] = d637
			ps957.OverlayValues[638] = d638
			ps957.OverlayValues[639] = d639
			ps957.OverlayValues[640] = d640
			ps957.OverlayValues[641] = d641
			ps957.OverlayValues[642] = d642
			ps957.OverlayValues[643] = d643
			ps957.OverlayValues[644] = d644
			ps957.OverlayValues[645] = d645
			ps957.OverlayValues[646] = d646
			ps957.OverlayValues[647] = d647
			ps957.OverlayValues[648] = d648
			ps957.OverlayValues[649] = d649
			ps957.OverlayValues[650] = d650
			ps957.OverlayValues[651] = d651
			ps957.OverlayValues[652] = d652
			ps957.OverlayValues[653] = d653
			ps957.OverlayValues[654] = d654
			ps957.OverlayValues[655] = d655
			ps957.OverlayValues[944] = d944
			ps957.OverlayValues[945] = d945
			ps957.OverlayValues[946] = d946
			ps957.OverlayValues[948] = d948
			ps957.OverlayValues[949] = d949
			ps957.OverlayValues[950] = d950
			ps957.OverlayValues[951] = d951
			ps957.OverlayValues[952] = d952
			ps957.OverlayValues[953] = d953
			ps957.OverlayValues[954] = d954
			ps957.OverlayValues[956] = d956
			return bbs[10].RenderPS(ps957)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d958 := ps.PhiValues[0]
				ctx.EnsureDesc(&d958)
				ctx.EmitStoreToStack(d958, int32(bbs[8].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d959 := ps.PhiValues[1]
				ctx.EnsureDesc(&d959)
				ctx.EmitStoreToStack(d959, int32(bbs[8].PhiBase)+int32(16))
			}
			ps.General = true
			return bbs[8].RenderPS(ps)
		}
		lbl26 := ctx.ReserveLabel()
		lbl27 := ctx.ReserveLabel()
		ctx.EmitJump(d952.Condition, lbl26)
		ctx.EmitJmp(lbl27)
		snap960 := d5
		snap961 := d6
		snap962 := d7
		snap963 := d8
		snap964 := d9
		snap965 := d10
		snap966 := d11
		snap967 := d12
		snap968 := d13
		snap969 := d14
		snap970 := d15
		snap971 := d16
		snap972 := d17
		snap973 := d18
		snap974 := d19
		snap975 := d20
		snap976 := d21
		snap977 := d23
		snap978 := d24
		snap979 := d25
		snap980 := d26
		snap981 := d27
		snap982 := d28
		snap983 := d29
		snap984 := d30
		snap985 := d31
		snap986 := d32
		snap987 := d33
		snap988 := d34
		snap989 := d35
		snap990 := d36
		snap991 := d37
		snap992 := d38
		snap993 := d39
		snap994 := d40
		snap995 := d41
		snap996 := d42
		snap997 := d43
		snap998 := d44
		snap999 := d45
		snap1000 := d46
		snap1001 := d47
		snap1002 := d48
		snap1003 := d49
		snap1004 := d50
		snap1005 := d51
		snap1006 := d52
		snap1007 := d53
		snap1008 := d54
		snap1009 := d55
		snap1010 := d56
		snap1011 := d57
		snap1012 := d60
		snap1013 := d61
		snap1014 := d62
		snap1015 := d177
		snap1016 := d178
		snap1017 := d179
		snap1018 := d180
		snap1019 := d181
		snap1020 := d182
		snap1021 := d183
		snap1022 := d184
		snap1023 := d185
		snap1024 := d186
		snap1025 := d187
		snap1026 := d188
		snap1027 := d189
		snap1028 := d190
		snap1029 := d191
		snap1030 := d192
		snap1031 := d193
		snap1032 := d194
		snap1033 := d195
		snap1034 := d196
		snap1035 := d197
		snap1036 := d198
		snap1037 := d199
		snap1038 := d200
		snap1039 := d201
		snap1040 := d202
		snap1041 := d203
		snap1042 := d204
		snap1043 := d205
		snap1044 := d206
		snap1045 := d209
		snap1046 := d386
		snap1047 := d387
		snap1048 := d388
		snap1049 := d389
		snap1050 := d391
		snap1051 := d392
		snap1052 := d393
		snap1053 := d394
		snap1054 := d395
		snap1055 := d396
		snap1056 := d397
		snap1057 := d398
		snap1058 := d400
		snap1059 := d402
		snap1060 := d403
		snap1061 := d404
		snap1062 := d508
		snap1063 := d509
		snap1064 := d512
		snap1065 := d619
		snap1066 := d620
		snap1067 := d621
		snap1068 := d622
		snap1069 := d623
		snap1070 := d625
		snap1071 := d626
		snap1072 := d627
		snap1073 := d628
		snap1074 := d629
		snap1075 := d630
		snap1076 := d631
		snap1077 := d632
		snap1078 := d633
		snap1079 := d634
		snap1080 := d635
		snap1081 := d636
		snap1082 := d637
		snap1083 := d638
		snap1084 := d639
		snap1085 := d640
		snap1086 := d641
		snap1087 := d642
		snap1088 := d643
		snap1089 := d644
		snap1090 := d645
		snap1091 := d646
		snap1092 := d647
		snap1093 := d648
		snap1094 := d649
		snap1095 := d650
		snap1096 := d651
		snap1097 := d652
		snap1098 := d653
		snap1099 := d654
		snap1100 := d655
		snap1101 := d944
		snap1102 := d945
		snap1103 := d946
		snap1104 := d948
		snap1105 := d949
		snap1106 := d950
		snap1107 := d951
		snap1108 := d952
		snap1109 := d953
		snap1110 := d954
		snap1111 := d956
		snap1112 := d958
		snap1113 := d959
		alloc1114 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl26)
		ctx.SyncDesc(&d12)
		if d12.Loc == scm.LocReg {
			ctx.ProtectReg(d12.Reg)
		} else if d12.Loc == scm.LocRegPair {
			ctx.ProtectReg(d12.Reg)
			ctx.ProtectReg(d12.Reg2)
		}
		d1115 = d12
		if d1115.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d1115)
		d1116 = d1115
		if d1116.Loc == scm.LocImm {
			d1116 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1116.Type, Imm: scm.NewInt(int64(uint64(d1116.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d1116.Reg, 32)
			ctx.EmitShrRegImm8(d1116.Reg, 32)
		}
		ctx.EmitStoreToStack(d1116, int32(bbs[2].PhiBase)+int32(0))
		if d12.Loc == scm.LocReg {
			ctx.UnprotectReg(d12.Reg)
		} else if d12.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d12.Reg)
			ctx.UnprotectReg(d12.Reg2)
		}
		ctx.EmitJmp(lbl3)
		ctx.RestoreAllocState(alloc1114)
		d5 = snap960
		d6 = snap961
		d7 = snap962
		d8 = snap963
		d9 = snap964
		d10 = snap965
		d11 = snap966
		d12 = snap967
		d13 = snap968
		d14 = snap969
		d15 = snap970
		d16 = snap971
		d17 = snap972
		d18 = snap973
		d19 = snap974
		d20 = snap975
		d21 = snap976
		d23 = snap977
		d24 = snap978
		d25 = snap979
		d26 = snap980
		d27 = snap981
		d28 = snap982
		d29 = snap983
		d30 = snap984
		d31 = snap985
		d32 = snap986
		d33 = snap987
		d34 = snap988
		d35 = snap989
		d36 = snap990
		d37 = snap991
		d38 = snap992
		d39 = snap993
		d40 = snap994
		d41 = snap995
		d42 = snap996
		d43 = snap997
		d44 = snap998
		d45 = snap999
		d46 = snap1000
		d47 = snap1001
		d48 = snap1002
		d49 = snap1003
		d50 = snap1004
		d51 = snap1005
		d52 = snap1006
		d53 = snap1007
		d54 = snap1008
		d55 = snap1009
		d56 = snap1010
		d57 = snap1011
		d60 = snap1012
		d61 = snap1013
		d62 = snap1014
		d177 = snap1015
		d178 = snap1016
		d179 = snap1017
		d180 = snap1018
		d181 = snap1019
		d182 = snap1020
		d183 = snap1021
		d184 = snap1022
		d185 = snap1023
		d186 = snap1024
		d187 = snap1025
		d188 = snap1026
		d189 = snap1027
		d190 = snap1028
		d191 = snap1029
		d192 = snap1030
		d193 = snap1031
		d194 = snap1032
		d195 = snap1033
		d196 = snap1034
		d197 = snap1035
		d198 = snap1036
		d199 = snap1037
		d200 = snap1038
		d201 = snap1039
		d202 = snap1040
		d203 = snap1041
		d204 = snap1042
		d205 = snap1043
		d206 = snap1044
		d209 = snap1045
		d386 = snap1046
		d387 = snap1047
		d388 = snap1048
		d389 = snap1049
		d391 = snap1050
		d392 = snap1051
		d393 = snap1052
		d394 = snap1053
		d395 = snap1054
		d396 = snap1055
		d397 = snap1056
		d398 = snap1057
		d400 = snap1058
		d402 = snap1059
		d403 = snap1060
		d404 = snap1061
		d508 = snap1062
		d509 = snap1063
		d512 = snap1064
		d619 = snap1065
		d620 = snap1066
		d621 = snap1067
		d622 = snap1068
		d623 = snap1069
		d625 = snap1070
		d626 = snap1071
		d627 = snap1072
		d628 = snap1073
		d629 = snap1074
		d630 = snap1075
		d631 = snap1076
		d632 = snap1077
		d633 = snap1078
		d634 = snap1079
		d635 = snap1080
		d636 = snap1081
		d637 = snap1082
		d638 = snap1083
		d639 = snap1084
		d640 = snap1085
		d641 = snap1086
		d642 = snap1087
		d643 = snap1088
		d644 = snap1089
		d645 = snap1090
		d646 = snap1091
		d647 = snap1092
		d648 = snap1093
		d649 = snap1094
		d650 = snap1095
		d651 = snap1096
		d652 = snap1097
		d653 = snap1098
		d654 = snap1099
		d655 = snap1100
		d944 = snap1101
		d945 = snap1102
		d946 = snap1103
		d948 = snap1104
		d949 = snap1105
		d950 = snap1106
		d951 = snap1107
		d952 = snap1108
		d953 = snap1109
		d954 = snap1110
		d956 = snap1111
		d958 = snap1112
		d959 = snap1113
		ctx.MarkLabel(lbl27)
		ctx.EmitJmp(lbl11)
		ctx.RestoreAllocState(alloc1114)
		d5 = snap960
		d6 = snap961
		d7 = snap962
		d8 = snap963
		d9 = snap964
		d10 = snap965
		d11 = snap966
		d12 = snap967
		d13 = snap968
		d14 = snap969
		d15 = snap970
		d16 = snap971
		d17 = snap972
		d18 = snap973
		d19 = snap974
		d20 = snap975
		d21 = snap976
		d23 = snap977
		d24 = snap978
		d25 = snap979
		d26 = snap980
		d27 = snap981
		d28 = snap982
		d29 = snap983
		d30 = snap984
		d31 = snap985
		d32 = snap986
		d33 = snap987
		d34 = snap988
		d35 = snap989
		d36 = snap990
		d37 = snap991
		d38 = snap992
		d39 = snap993
		d40 = snap994
		d41 = snap995
		d42 = snap996
		d43 = snap997
		d44 = snap998
		d45 = snap999
		d46 = snap1000
		d47 = snap1001
		d48 = snap1002
		d49 = snap1003
		d50 = snap1004
		d51 = snap1005
		d52 = snap1006
		d53 = snap1007
		d54 = snap1008
		d55 = snap1009
		d56 = snap1010
		d57 = snap1011
		d60 = snap1012
		d61 = snap1013
		d62 = snap1014
		d177 = snap1015
		d178 = snap1016
		d179 = snap1017
		d180 = snap1018
		d181 = snap1019
		d182 = snap1020
		d183 = snap1021
		d184 = snap1022
		d185 = snap1023
		d186 = snap1024
		d187 = snap1025
		d188 = snap1026
		d189 = snap1027
		d190 = snap1028
		d191 = snap1029
		d192 = snap1030
		d193 = snap1031
		d194 = snap1032
		d195 = snap1033
		d196 = snap1034
		d197 = snap1035
		d198 = snap1036
		d199 = snap1037
		d200 = snap1038
		d201 = snap1039
		d202 = snap1040
		d203 = snap1041
		d204 = snap1042
		d205 = snap1043
		d206 = snap1044
		d209 = snap1045
		d386 = snap1046
		d387 = snap1047
		d388 = snap1048
		d389 = snap1049
		d391 = snap1050
		d392 = snap1051
		d393 = snap1052
		d394 = snap1053
		d395 = snap1054
		d396 = snap1055
		d397 = snap1056
		d398 = snap1057
		d400 = snap1058
		d402 = snap1059
		d403 = snap1060
		d404 = snap1061
		d508 = snap1062
		d509 = snap1063
		d512 = snap1064
		d619 = snap1065
		d620 = snap1066
		d621 = snap1067
		d622 = snap1068
		d623 = snap1069
		d625 = snap1070
		d626 = snap1071
		d627 = snap1072
		d628 = snap1073
		d629 = snap1074
		d630 = snap1075
		d631 = snap1076
		d632 = snap1077
		d633 = snap1078
		d634 = snap1079
		d635 = snap1080
		d636 = snap1081
		d637 = snap1082
		d638 = snap1083
		d639 = snap1084
		d640 = snap1085
		d641 = snap1086
		d642 = snap1087
		d643 = snap1088
		d644 = snap1089
		d645 = snap1090
		d646 = snap1091
		d647 = snap1092
		d648 = snap1093
		d649 = snap1094
		d650 = snap1095
		d651 = snap1096
		d652 = snap1097
		d653 = snap1098
		d654 = snap1099
		d655 = snap1100
		d944 = snap1101
		d945 = snap1102
		d946 = snap1103
		d948 = snap1104
		d949 = snap1105
		d950 = snap1106
		d951 = snap1107
		d952 = snap1108
		d953 = snap1109
		d954 = snap1110
		d956 = snap1111
		d958 = snap1112
		d959 = snap1113
		ps1117 := scm.PhiState{General: true}
		ps1117.OverlayValues = make([]scm.JITValueDesc, 1117)
		ps1117.OverlayValues[5] = d5
		ps1117.OverlayValues[6] = d6
		ps1117.OverlayValues[7] = d7
		ps1117.OverlayValues[8] = d8
		ps1117.OverlayValues[9] = d9
		ps1117.OverlayValues[10] = d10
		ps1117.OverlayValues[11] = d11
		ps1117.OverlayValues[12] = d12
		ps1117.OverlayValues[13] = d13
		ps1117.OverlayValues[14] = d14
		ps1117.OverlayValues[15] = d15
		ps1117.OverlayValues[16] = d16
		ps1117.OverlayValues[17] = d17
		ps1117.OverlayValues[18] = d18
		ps1117.OverlayValues[19] = d19
		ps1117.OverlayValues[20] = d20
		ps1117.OverlayValues[21] = d21
		ps1117.OverlayValues[23] = d23
		ps1117.OverlayValues[24] = d24
		ps1117.OverlayValues[25] = d25
		ps1117.OverlayValues[26] = d26
		ps1117.OverlayValues[27] = d27
		ps1117.OverlayValues[28] = d28
		ps1117.OverlayValues[29] = d29
		ps1117.OverlayValues[30] = d30
		ps1117.OverlayValues[31] = d31
		ps1117.OverlayValues[32] = d32
		ps1117.OverlayValues[33] = d33
		ps1117.OverlayValues[34] = d34
		ps1117.OverlayValues[35] = d35
		ps1117.OverlayValues[36] = d36
		ps1117.OverlayValues[37] = d37
		ps1117.OverlayValues[38] = d38
		ps1117.OverlayValues[39] = d39
		ps1117.OverlayValues[40] = d40
		ps1117.OverlayValues[41] = d41
		ps1117.OverlayValues[42] = d42
		ps1117.OverlayValues[43] = d43
		ps1117.OverlayValues[44] = d44
		ps1117.OverlayValues[45] = d45
		ps1117.OverlayValues[46] = d46
		ps1117.OverlayValues[47] = d47
		ps1117.OverlayValues[48] = d48
		ps1117.OverlayValues[49] = d49
		ps1117.OverlayValues[50] = d50
		ps1117.OverlayValues[51] = d51
		ps1117.OverlayValues[52] = d52
		ps1117.OverlayValues[53] = d53
		ps1117.OverlayValues[54] = d54
		ps1117.OverlayValues[55] = d55
		ps1117.OverlayValues[56] = d56
		ps1117.OverlayValues[57] = d57
		ps1117.OverlayValues[60] = d60
		ps1117.OverlayValues[61] = d61
		ps1117.OverlayValues[62] = d62
		ps1117.OverlayValues[177] = d177
		ps1117.OverlayValues[178] = d178
		ps1117.OverlayValues[179] = d179
		ps1117.OverlayValues[180] = d180
		ps1117.OverlayValues[181] = d181
		ps1117.OverlayValues[182] = d182
		ps1117.OverlayValues[183] = d183
		ps1117.OverlayValues[184] = d184
		ps1117.OverlayValues[185] = d185
		ps1117.OverlayValues[186] = d186
		ps1117.OverlayValues[187] = d187
		ps1117.OverlayValues[188] = d188
		ps1117.OverlayValues[189] = d189
		ps1117.OverlayValues[190] = d190
		ps1117.OverlayValues[191] = d191
		ps1117.OverlayValues[192] = d192
		ps1117.OverlayValues[193] = d193
		ps1117.OverlayValues[194] = d194
		ps1117.OverlayValues[195] = d195
		ps1117.OverlayValues[196] = d196
		ps1117.OverlayValues[197] = d197
		ps1117.OverlayValues[198] = d198
		ps1117.OverlayValues[199] = d199
		ps1117.OverlayValues[200] = d200
		ps1117.OverlayValues[201] = d201
		ps1117.OverlayValues[202] = d202
		ps1117.OverlayValues[203] = d203
		ps1117.OverlayValues[204] = d204
		ps1117.OverlayValues[205] = d205
		ps1117.OverlayValues[206] = d206
		ps1117.OverlayValues[209] = d209
		ps1117.OverlayValues[386] = d386
		ps1117.OverlayValues[387] = d387
		ps1117.OverlayValues[388] = d388
		ps1117.OverlayValues[389] = d389
		ps1117.OverlayValues[391] = d391
		ps1117.OverlayValues[392] = d392
		ps1117.OverlayValues[393] = d393
		ps1117.OverlayValues[394] = d394
		ps1117.OverlayValues[395] = d395
		ps1117.OverlayValues[396] = d396
		ps1117.OverlayValues[397] = d397
		ps1117.OverlayValues[398] = d398
		ps1117.OverlayValues[400] = d400
		ps1117.OverlayValues[402] = d402
		ps1117.OverlayValues[403] = d403
		ps1117.OverlayValues[404] = d404
		ps1117.OverlayValues[508] = d508
		ps1117.OverlayValues[509] = d509
		ps1117.OverlayValues[512] = d512
		ps1117.OverlayValues[619] = d619
		ps1117.OverlayValues[620] = d620
		ps1117.OverlayValues[621] = d621
		ps1117.OverlayValues[622] = d622
		ps1117.OverlayValues[623] = d623
		ps1117.OverlayValues[625] = d625
		ps1117.OverlayValues[626] = d626
		ps1117.OverlayValues[627] = d627
		ps1117.OverlayValues[628] = d628
		ps1117.OverlayValues[629] = d629
		ps1117.OverlayValues[630] = d630
		ps1117.OverlayValues[631] = d631
		ps1117.OverlayValues[632] = d632
		ps1117.OverlayValues[633] = d633
		ps1117.OverlayValues[634] = d634
		ps1117.OverlayValues[635] = d635
		ps1117.OverlayValues[636] = d636
		ps1117.OverlayValues[637] = d637
		ps1117.OverlayValues[638] = d638
		ps1117.OverlayValues[639] = d639
		ps1117.OverlayValues[640] = d640
		ps1117.OverlayValues[641] = d641
		ps1117.OverlayValues[642] = d642
		ps1117.OverlayValues[643] = d643
		ps1117.OverlayValues[644] = d644
		ps1117.OverlayValues[645] = d645
		ps1117.OverlayValues[646] = d646
		ps1117.OverlayValues[647] = d647
		ps1117.OverlayValues[648] = d648
		ps1117.OverlayValues[649] = d649
		ps1117.OverlayValues[650] = d650
		ps1117.OverlayValues[651] = d651
		ps1117.OverlayValues[652] = d652
		ps1117.OverlayValues[653] = d653
		ps1117.OverlayValues[654] = d654
		ps1117.OverlayValues[655] = d655
		ps1117.OverlayValues[944] = d944
		ps1117.OverlayValues[945] = d945
		ps1117.OverlayValues[946] = d946
		ps1117.OverlayValues[948] = d948
		ps1117.OverlayValues[949] = d949
		ps1117.OverlayValues[950] = d950
		ps1117.OverlayValues[951] = d951
		ps1117.OverlayValues[952] = d952
		ps1117.OverlayValues[953] = d953
		ps1117.OverlayValues[954] = d954
		ps1117.OverlayValues[956] = d956
		ps1117.OverlayValues[958] = d958
		ps1117.OverlayValues[959] = d959
		ps1117.OverlayValues[1115] = d1115
		ps1117.OverlayValues[1116] = d1116
		ps1117.PhiValues = make([]scm.JITValueDesc, 1)
		d1119 = d12
		ps1117.PhiValues[0] = d1119
		ps1118 := scm.PhiState{General: true}
		ps1118.OverlayValues = make([]scm.JITValueDesc, 1120)
		ps1118.OverlayValues[5] = d5
		ps1118.OverlayValues[6] = d6
		ps1118.OverlayValues[7] = d7
		ps1118.OverlayValues[8] = d8
		ps1118.OverlayValues[9] = d9
		ps1118.OverlayValues[10] = d10
		ps1118.OverlayValues[11] = d11
		ps1118.OverlayValues[12] = d12
		ps1118.OverlayValues[13] = d13
		ps1118.OverlayValues[14] = d14
		ps1118.OverlayValues[15] = d15
		ps1118.OverlayValues[16] = d16
		ps1118.OverlayValues[17] = d17
		ps1118.OverlayValues[18] = d18
		ps1118.OverlayValues[19] = d19
		ps1118.OverlayValues[20] = d20
		ps1118.OverlayValues[21] = d21
		ps1118.OverlayValues[23] = d23
		ps1118.OverlayValues[24] = d24
		ps1118.OverlayValues[25] = d25
		ps1118.OverlayValues[26] = d26
		ps1118.OverlayValues[27] = d27
		ps1118.OverlayValues[28] = d28
		ps1118.OverlayValues[29] = d29
		ps1118.OverlayValues[30] = d30
		ps1118.OverlayValues[31] = d31
		ps1118.OverlayValues[32] = d32
		ps1118.OverlayValues[33] = d33
		ps1118.OverlayValues[34] = d34
		ps1118.OverlayValues[35] = d35
		ps1118.OverlayValues[36] = d36
		ps1118.OverlayValues[37] = d37
		ps1118.OverlayValues[38] = d38
		ps1118.OverlayValues[39] = d39
		ps1118.OverlayValues[40] = d40
		ps1118.OverlayValues[41] = d41
		ps1118.OverlayValues[42] = d42
		ps1118.OverlayValues[43] = d43
		ps1118.OverlayValues[44] = d44
		ps1118.OverlayValues[45] = d45
		ps1118.OverlayValues[46] = d46
		ps1118.OverlayValues[47] = d47
		ps1118.OverlayValues[48] = d48
		ps1118.OverlayValues[49] = d49
		ps1118.OverlayValues[50] = d50
		ps1118.OverlayValues[51] = d51
		ps1118.OverlayValues[52] = d52
		ps1118.OverlayValues[53] = d53
		ps1118.OverlayValues[54] = d54
		ps1118.OverlayValues[55] = d55
		ps1118.OverlayValues[56] = d56
		ps1118.OverlayValues[57] = d57
		ps1118.OverlayValues[60] = d60
		ps1118.OverlayValues[61] = d61
		ps1118.OverlayValues[62] = d62
		ps1118.OverlayValues[177] = d177
		ps1118.OverlayValues[178] = d178
		ps1118.OverlayValues[179] = d179
		ps1118.OverlayValues[180] = d180
		ps1118.OverlayValues[181] = d181
		ps1118.OverlayValues[182] = d182
		ps1118.OverlayValues[183] = d183
		ps1118.OverlayValues[184] = d184
		ps1118.OverlayValues[185] = d185
		ps1118.OverlayValues[186] = d186
		ps1118.OverlayValues[187] = d187
		ps1118.OverlayValues[188] = d188
		ps1118.OverlayValues[189] = d189
		ps1118.OverlayValues[190] = d190
		ps1118.OverlayValues[191] = d191
		ps1118.OverlayValues[192] = d192
		ps1118.OverlayValues[193] = d193
		ps1118.OverlayValues[194] = d194
		ps1118.OverlayValues[195] = d195
		ps1118.OverlayValues[196] = d196
		ps1118.OverlayValues[197] = d197
		ps1118.OverlayValues[198] = d198
		ps1118.OverlayValues[199] = d199
		ps1118.OverlayValues[200] = d200
		ps1118.OverlayValues[201] = d201
		ps1118.OverlayValues[202] = d202
		ps1118.OverlayValues[203] = d203
		ps1118.OverlayValues[204] = d204
		ps1118.OverlayValues[205] = d205
		ps1118.OverlayValues[206] = d206
		ps1118.OverlayValues[209] = d209
		ps1118.OverlayValues[386] = d386
		ps1118.OverlayValues[387] = d387
		ps1118.OverlayValues[388] = d388
		ps1118.OverlayValues[389] = d389
		ps1118.OverlayValues[391] = d391
		ps1118.OverlayValues[392] = d392
		ps1118.OverlayValues[393] = d393
		ps1118.OverlayValues[394] = d394
		ps1118.OverlayValues[395] = d395
		ps1118.OverlayValues[396] = d396
		ps1118.OverlayValues[397] = d397
		ps1118.OverlayValues[398] = d398
		ps1118.OverlayValues[400] = d400
		ps1118.OverlayValues[402] = d402
		ps1118.OverlayValues[403] = d403
		ps1118.OverlayValues[404] = d404
		ps1118.OverlayValues[508] = d508
		ps1118.OverlayValues[509] = d509
		ps1118.OverlayValues[512] = d512
		ps1118.OverlayValues[619] = d619
		ps1118.OverlayValues[620] = d620
		ps1118.OverlayValues[621] = d621
		ps1118.OverlayValues[622] = d622
		ps1118.OverlayValues[623] = d623
		ps1118.OverlayValues[625] = d625
		ps1118.OverlayValues[626] = d626
		ps1118.OverlayValues[627] = d627
		ps1118.OverlayValues[628] = d628
		ps1118.OverlayValues[629] = d629
		ps1118.OverlayValues[630] = d630
		ps1118.OverlayValues[631] = d631
		ps1118.OverlayValues[632] = d632
		ps1118.OverlayValues[633] = d633
		ps1118.OverlayValues[634] = d634
		ps1118.OverlayValues[635] = d635
		ps1118.OverlayValues[636] = d636
		ps1118.OverlayValues[637] = d637
		ps1118.OverlayValues[638] = d638
		ps1118.OverlayValues[639] = d639
		ps1118.OverlayValues[640] = d640
		ps1118.OverlayValues[641] = d641
		ps1118.OverlayValues[642] = d642
		ps1118.OverlayValues[643] = d643
		ps1118.OverlayValues[644] = d644
		ps1118.OverlayValues[645] = d645
		ps1118.OverlayValues[646] = d646
		ps1118.OverlayValues[647] = d647
		ps1118.OverlayValues[648] = d648
		ps1118.OverlayValues[649] = d649
		ps1118.OverlayValues[650] = d650
		ps1118.OverlayValues[651] = d651
		ps1118.OverlayValues[652] = d652
		ps1118.OverlayValues[653] = d653
		ps1118.OverlayValues[654] = d654
		ps1118.OverlayValues[655] = d655
		ps1118.OverlayValues[944] = d944
		ps1118.OverlayValues[945] = d945
		ps1118.OverlayValues[946] = d946
		ps1118.OverlayValues[948] = d948
		ps1118.OverlayValues[949] = d949
		ps1118.OverlayValues[950] = d950
		ps1118.OverlayValues[951] = d951
		ps1118.OverlayValues[952] = d952
		ps1118.OverlayValues[953] = d953
		ps1118.OverlayValues[954] = d954
		ps1118.OverlayValues[956] = d956
		ps1118.OverlayValues[958] = d958
		ps1118.OverlayValues[959] = d959
		ps1118.OverlayValues[1115] = d1115
		ps1118.OverlayValues[1116] = d1116
		ps1118.OverlayValues[1119] = d1119
		snap1120 := d5
		snap1121 := d6
		snap1122 := d7
		snap1123 := d8
		snap1124 := d9
		snap1125 := d10
		snap1126 := d11
		snap1127 := d12
		snap1128 := d13
		snap1129 := d14
		snap1130 := d15
		snap1131 := d16
		snap1132 := d17
		snap1133 := d18
		snap1134 := d19
		snap1135 := d20
		snap1136 := d21
		snap1137 := d23
		snap1138 := d24
		snap1139 := d25
		snap1140 := d26
		snap1141 := d27
		snap1142 := d28
		snap1143 := d29
		snap1144 := d30
		snap1145 := d31
		snap1146 := d32
		snap1147 := d33
		snap1148 := d34
		snap1149 := d35
		snap1150 := d36
		snap1151 := d37
		snap1152 := d38
		snap1153 := d39
		snap1154 := d40
		snap1155 := d41
		snap1156 := d42
		snap1157 := d43
		snap1158 := d44
		snap1159 := d45
		snap1160 := d46
		snap1161 := d47
		snap1162 := d48
		snap1163 := d49
		snap1164 := d50
		snap1165 := d51
		snap1166 := d52
		snap1167 := d53
		snap1168 := d54
		snap1169 := d55
		snap1170 := d56
		snap1171 := d57
		snap1172 := d60
		snap1173 := d61
		snap1174 := d62
		snap1175 := d177
		snap1176 := d178
		snap1177 := d179
		snap1178 := d180
		snap1179 := d181
		snap1180 := d182
		snap1181 := d183
		snap1182 := d184
		snap1183 := d185
		snap1184 := d186
		snap1185 := d187
		snap1186 := d188
		snap1187 := d189
		snap1188 := d190
		snap1189 := d191
		snap1190 := d192
		snap1191 := d193
		snap1192 := d194
		snap1193 := d195
		snap1194 := d196
		snap1195 := d197
		snap1196 := d198
		snap1197 := d199
		snap1198 := d200
		snap1199 := d201
		snap1200 := d202
		snap1201 := d203
		snap1202 := d204
		snap1203 := d205
		snap1204 := d206
		snap1205 := d209
		snap1206 := d386
		snap1207 := d387
		snap1208 := d388
		snap1209 := d389
		snap1210 := d391
		snap1211 := d392
		snap1212 := d393
		snap1213 := d394
		snap1214 := d395
		snap1215 := d396
		snap1216 := d397
		snap1217 := d398
		snap1218 := d400
		snap1219 := d402
		snap1220 := d403
		snap1221 := d404
		snap1222 := d508
		snap1223 := d509
		snap1224 := d512
		snap1225 := d619
		snap1226 := d620
		snap1227 := d621
		snap1228 := d622
		snap1229 := d623
		snap1230 := d625
		snap1231 := d626
		snap1232 := d627
		snap1233 := d628
		snap1234 := d629
		snap1235 := d630
		snap1236 := d631
		snap1237 := d632
		snap1238 := d633
		snap1239 := d634
		snap1240 := d635
		snap1241 := d636
		snap1242 := d637
		snap1243 := d638
		snap1244 := d639
		snap1245 := d640
		snap1246 := d641
		snap1247 := d642
		snap1248 := d643
		snap1249 := d644
		snap1250 := d645
		snap1251 := d646
		snap1252 := d647
		snap1253 := d648
		snap1254 := d649
		snap1255 := d650
		snap1256 := d651
		snap1257 := d652
		snap1258 := d653
		snap1259 := d654
		snap1260 := d655
		snap1261 := d944
		snap1262 := d945
		snap1263 := d946
		snap1264 := d948
		snap1265 := d949
		snap1266 := d950
		snap1267 := d951
		snap1268 := d952
		snap1269 := d953
		snap1270 := d954
		snap1271 := d956
		snap1272 := d958
		snap1273 := d959
		snap1274 := d1115
		snap1275 := d1116
		snap1276 := d1119
		alloc1277 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps1117)
		}
		ctx.RestoreAllocState(alloc1277)
		d5 = snap1120
		d6 = snap1121
		d7 = snap1122
		d8 = snap1123
		d9 = snap1124
		d10 = snap1125
		d11 = snap1126
		d12 = snap1127
		d13 = snap1128
		d14 = snap1129
		d15 = snap1130
		d16 = snap1131
		d17 = snap1132
		d18 = snap1133
		d19 = snap1134
		d20 = snap1135
		d21 = snap1136
		d23 = snap1137
		d24 = snap1138
		d25 = snap1139
		d26 = snap1140
		d27 = snap1141
		d28 = snap1142
		d29 = snap1143
		d30 = snap1144
		d31 = snap1145
		d32 = snap1146
		d33 = snap1147
		d34 = snap1148
		d35 = snap1149
		d36 = snap1150
		d37 = snap1151
		d38 = snap1152
		d39 = snap1153
		d40 = snap1154
		d41 = snap1155
		d42 = snap1156
		d43 = snap1157
		d44 = snap1158
		d45 = snap1159
		d46 = snap1160
		d47 = snap1161
		d48 = snap1162
		d49 = snap1163
		d50 = snap1164
		d51 = snap1165
		d52 = snap1166
		d53 = snap1167
		d54 = snap1168
		d55 = snap1169
		d56 = snap1170
		d57 = snap1171
		d60 = snap1172
		d61 = snap1173
		d62 = snap1174
		d177 = snap1175
		d178 = snap1176
		d179 = snap1177
		d180 = snap1178
		d181 = snap1179
		d182 = snap1180
		d183 = snap1181
		d184 = snap1182
		d185 = snap1183
		d186 = snap1184
		d187 = snap1185
		d188 = snap1186
		d189 = snap1187
		d190 = snap1188
		d191 = snap1189
		d192 = snap1190
		d193 = snap1191
		d194 = snap1192
		d195 = snap1193
		d196 = snap1194
		d197 = snap1195
		d198 = snap1196
		d199 = snap1197
		d200 = snap1198
		d201 = snap1199
		d202 = snap1200
		d203 = snap1201
		d204 = snap1202
		d205 = snap1203
		d206 = snap1204
		d209 = snap1205
		d386 = snap1206
		d387 = snap1207
		d388 = snap1208
		d389 = snap1209
		d391 = snap1210
		d392 = snap1211
		d393 = snap1212
		d394 = snap1213
		d395 = snap1214
		d396 = snap1215
		d397 = snap1216
		d398 = snap1217
		d400 = snap1218
		d402 = snap1219
		d403 = snap1220
		d404 = snap1221
		d508 = snap1222
		d509 = snap1223
		d512 = snap1224
		d619 = snap1225
		d620 = snap1226
		d621 = snap1227
		d622 = snap1228
		d623 = snap1229
		d625 = snap1230
		d626 = snap1231
		d627 = snap1232
		d628 = snap1233
		d629 = snap1234
		d630 = snap1235
		d631 = snap1236
		d632 = snap1237
		d633 = snap1238
		d634 = snap1239
		d635 = snap1240
		d636 = snap1241
		d637 = snap1242
		d638 = snap1243
		d639 = snap1244
		d640 = snap1245
		d641 = snap1246
		d642 = snap1247
		d643 = snap1248
		d644 = snap1249
		d645 = snap1250
		d646 = snap1251
		d647 = snap1252
		d648 = snap1253
		d649 = snap1254
		d650 = snap1255
		d651 = snap1256
		d652 = snap1257
		d653 = snap1258
		d654 = snap1259
		d655 = snap1260
		d944 = snap1261
		d945 = snap1262
		d946 = snap1263
		d948 = snap1264
		d949 = snap1265
		d950 = snap1266
		d951 = snap1267
		d952 = snap1268
		d953 = snap1269
		d954 = snap1270
		d956 = snap1271
		d958 = snap1272
		d959 = snap1273
		d1115 = snap1274
		d1116 = snap1275
		d1119 = snap1276
		if !bbs[10].Rendered {
			return bbs[10].RenderPS(ps1118)
		}
		return result
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
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
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
		if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != scm.LocNone {
			d177 = ps.OverlayValues[177]
		}
		if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
			d178 = ps.OverlayValues[178]
		}
		if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != scm.LocNone {
			d179 = ps.OverlayValues[179]
		}
		if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != scm.LocNone {
			d180 = ps.OverlayValues[180]
		}
		if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != scm.LocNone {
			d181 = ps.OverlayValues[181]
		}
		if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != scm.LocNone {
			d182 = ps.OverlayValues[182]
		}
		if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != scm.LocNone {
			d183 = ps.OverlayValues[183]
		}
		if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
			d184 = ps.OverlayValues[184]
		}
		if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != scm.LocNone {
			d185 = ps.OverlayValues[185]
		}
		if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != scm.LocNone {
			d186 = ps.OverlayValues[186]
		}
		if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
			d187 = ps.OverlayValues[187]
		}
		if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != scm.LocNone {
			d188 = ps.OverlayValues[188]
		}
		if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != scm.LocNone {
			d189 = ps.OverlayValues[189]
		}
		if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != scm.LocNone {
			d190 = ps.OverlayValues[190]
		}
		if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != scm.LocNone {
			d191 = ps.OverlayValues[191]
		}
		if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != scm.LocNone {
			d192 = ps.OverlayValues[192]
		}
		if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != scm.LocNone {
			d193 = ps.OverlayValues[193]
		}
		if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != scm.LocNone {
			d194 = ps.OverlayValues[194]
		}
		if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != scm.LocNone {
			d195 = ps.OverlayValues[195]
		}
		if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
			d196 = ps.OverlayValues[196]
		}
		if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != scm.LocNone {
			d197 = ps.OverlayValues[197]
		}
		if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != scm.LocNone {
			d198 = ps.OverlayValues[198]
		}
		if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
			d199 = ps.OverlayValues[199]
		}
		if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
			d200 = ps.OverlayValues[200]
		}
		if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
			d201 = ps.OverlayValues[201]
		}
		if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != scm.LocNone {
			d202 = ps.OverlayValues[202]
		}
		if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != scm.LocNone {
			d203 = ps.OverlayValues[203]
		}
		if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != scm.LocNone {
			d204 = ps.OverlayValues[204]
		}
		if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != scm.LocNone {
			d205 = ps.OverlayValues[205]
		}
		if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != scm.LocNone {
			d206 = ps.OverlayValues[206]
		}
		if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != scm.LocNone {
			d209 = ps.OverlayValues[209]
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
		if len(ps.OverlayValues) > 400 && ps.OverlayValues[400].Loc != scm.LocNone {
			d400 = ps.OverlayValues[400]
		}
		if len(ps.OverlayValues) > 402 && ps.OverlayValues[402].Loc != scm.LocNone {
			d402 = ps.OverlayValues[402]
		}
		if len(ps.OverlayValues) > 403 && ps.OverlayValues[403].Loc != scm.LocNone {
			d403 = ps.OverlayValues[403]
		}
		if len(ps.OverlayValues) > 404 && ps.OverlayValues[404].Loc != scm.LocNone {
			d404 = ps.OverlayValues[404]
		}
		if len(ps.OverlayValues) > 508 && ps.OverlayValues[508].Loc != scm.LocNone {
			d508 = ps.OverlayValues[508]
		}
		if len(ps.OverlayValues) > 509 && ps.OverlayValues[509].Loc != scm.LocNone {
			d509 = ps.OverlayValues[509]
		}
		if len(ps.OverlayValues) > 512 && ps.OverlayValues[512].Loc != scm.LocNone {
			d512 = ps.OverlayValues[512]
		}
		if len(ps.OverlayValues) > 619 && ps.OverlayValues[619].Loc != scm.LocNone {
			d619 = ps.OverlayValues[619]
		}
		if len(ps.OverlayValues) > 620 && ps.OverlayValues[620].Loc != scm.LocNone {
			d620 = ps.OverlayValues[620]
		}
		if len(ps.OverlayValues) > 621 && ps.OverlayValues[621].Loc != scm.LocNone {
			d621 = ps.OverlayValues[621]
		}
		if len(ps.OverlayValues) > 622 && ps.OverlayValues[622].Loc != scm.LocNone {
			d622 = ps.OverlayValues[622]
		}
		if len(ps.OverlayValues) > 623 && ps.OverlayValues[623].Loc != scm.LocNone {
			d623 = ps.OverlayValues[623]
		}
		if len(ps.OverlayValues) > 625 && ps.OverlayValues[625].Loc != scm.LocNone {
			d625 = ps.OverlayValues[625]
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
		if len(ps.OverlayValues) > 629 && ps.OverlayValues[629].Loc != scm.LocNone {
			d629 = ps.OverlayValues[629]
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
		if len(ps.OverlayValues) > 637 && ps.OverlayValues[637].Loc != scm.LocNone {
			d637 = ps.OverlayValues[637]
		}
		if len(ps.OverlayValues) > 638 && ps.OverlayValues[638].Loc != scm.LocNone {
			d638 = ps.OverlayValues[638]
		}
		if len(ps.OverlayValues) > 639 && ps.OverlayValues[639].Loc != scm.LocNone {
			d639 = ps.OverlayValues[639]
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
		if len(ps.OverlayValues) > 644 && ps.OverlayValues[644].Loc != scm.LocNone {
			d644 = ps.OverlayValues[644]
		}
		if len(ps.OverlayValues) > 645 && ps.OverlayValues[645].Loc != scm.LocNone {
			d645 = ps.OverlayValues[645]
		}
		if len(ps.OverlayValues) > 646 && ps.OverlayValues[646].Loc != scm.LocNone {
			d646 = ps.OverlayValues[646]
		}
		if len(ps.OverlayValues) > 647 && ps.OverlayValues[647].Loc != scm.LocNone {
			d647 = ps.OverlayValues[647]
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
		if len(ps.OverlayValues) > 651 && ps.OverlayValues[651].Loc != scm.LocNone {
			d651 = ps.OverlayValues[651]
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
		if len(ps.OverlayValues) > 944 && ps.OverlayValues[944].Loc != scm.LocNone {
			d944 = ps.OverlayValues[944]
		}
		if len(ps.OverlayValues) > 945 && ps.OverlayValues[945].Loc != scm.LocNone {
			d945 = ps.OverlayValues[945]
		}
		if len(ps.OverlayValues) > 946 && ps.OverlayValues[946].Loc != scm.LocNone {
			d946 = ps.OverlayValues[946]
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
		if len(ps.OverlayValues) > 956 && ps.OverlayValues[956].Loc != scm.LocNone {
			d956 = ps.OverlayValues[956]
		}
		if len(ps.OverlayValues) > 958 && ps.OverlayValues[958].Loc != scm.LocNone {
			d958 = ps.OverlayValues[958]
		}
		if len(ps.OverlayValues) > 959 && ps.OverlayValues[959].Loc != scm.LocNone {
			d959 = ps.OverlayValues[959]
		}
		if len(ps.OverlayValues) > 1115 && ps.OverlayValues[1115].Loc != scm.LocNone {
			d1115 = ps.OverlayValues[1115]
		}
		if len(ps.OverlayValues) > 1116 && ps.OverlayValues[1116].Loc != scm.LocNone {
			d1116 = ps.OverlayValues[1116]
		}
		if len(ps.OverlayValues) > 1119 && ps.OverlayValues[1119].Loc != scm.LocNone {
			d1119 = ps.OverlayValues[1119]
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
			d1278 = d9
			if d1278.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d1278)
			d1279 = d1278
			if d1279.Loc == scm.LocImm {
				d1279 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1279.Type, Imm: scm.NewInt(int64(uint64(d1279.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d1279.Reg, 32)
				ctx.EmitShrRegImm8(d1279.Reg, 32)
			}
			ctx.EmitStoreToStack(d1279, int32(bbs[8].PhiBase)+int32(0))
			d1280 = d11
			if d1280.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d1280)
			d1281 = d1280
			if d1281.Loc == scm.LocImm {
				d1281 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1281.Type, Imm: scm.NewInt(int64(uint64(d1281.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d1281.Reg, 32)
				ctx.EmitShrRegImm8(d1281.Reg, 32)
			}
			ctx.EmitStoreToStack(d1281, int32(bbs[8].PhiBase)+int32(16))
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
		ps1282 := scm.PhiState{General: ps.General}
		ps1282.OverlayValues = make([]scm.JITValueDesc, 1282)
		ps1282.OverlayValues[5] = d5
		ps1282.OverlayValues[6] = d6
		ps1282.OverlayValues[7] = d7
		ps1282.OverlayValues[8] = d8
		ps1282.OverlayValues[9] = d9
		ps1282.OverlayValues[10] = d10
		ps1282.OverlayValues[11] = d11
		ps1282.OverlayValues[12] = d12
		ps1282.OverlayValues[13] = d13
		ps1282.OverlayValues[14] = d14
		ps1282.OverlayValues[15] = d15
		ps1282.OverlayValues[16] = d16
		ps1282.OverlayValues[17] = d17
		ps1282.OverlayValues[18] = d18
		ps1282.OverlayValues[19] = d19
		ps1282.OverlayValues[20] = d20
		ps1282.OverlayValues[21] = d21
		ps1282.OverlayValues[23] = d23
		ps1282.OverlayValues[24] = d24
		ps1282.OverlayValues[25] = d25
		ps1282.OverlayValues[26] = d26
		ps1282.OverlayValues[27] = d27
		ps1282.OverlayValues[28] = d28
		ps1282.OverlayValues[29] = d29
		ps1282.OverlayValues[30] = d30
		ps1282.OverlayValues[31] = d31
		ps1282.OverlayValues[32] = d32
		ps1282.OverlayValues[33] = d33
		ps1282.OverlayValues[34] = d34
		ps1282.OverlayValues[35] = d35
		ps1282.OverlayValues[36] = d36
		ps1282.OverlayValues[37] = d37
		ps1282.OverlayValues[38] = d38
		ps1282.OverlayValues[39] = d39
		ps1282.OverlayValues[40] = d40
		ps1282.OverlayValues[41] = d41
		ps1282.OverlayValues[42] = d42
		ps1282.OverlayValues[43] = d43
		ps1282.OverlayValues[44] = d44
		ps1282.OverlayValues[45] = d45
		ps1282.OverlayValues[46] = d46
		ps1282.OverlayValues[47] = d47
		ps1282.OverlayValues[48] = d48
		ps1282.OverlayValues[49] = d49
		ps1282.OverlayValues[50] = d50
		ps1282.OverlayValues[51] = d51
		ps1282.OverlayValues[52] = d52
		ps1282.OverlayValues[53] = d53
		ps1282.OverlayValues[54] = d54
		ps1282.OverlayValues[55] = d55
		ps1282.OverlayValues[56] = d56
		ps1282.OverlayValues[57] = d57
		ps1282.OverlayValues[60] = d60
		ps1282.OverlayValues[61] = d61
		ps1282.OverlayValues[62] = d62
		ps1282.OverlayValues[177] = d177
		ps1282.OverlayValues[178] = d178
		ps1282.OverlayValues[179] = d179
		ps1282.OverlayValues[180] = d180
		ps1282.OverlayValues[181] = d181
		ps1282.OverlayValues[182] = d182
		ps1282.OverlayValues[183] = d183
		ps1282.OverlayValues[184] = d184
		ps1282.OverlayValues[185] = d185
		ps1282.OverlayValues[186] = d186
		ps1282.OverlayValues[187] = d187
		ps1282.OverlayValues[188] = d188
		ps1282.OverlayValues[189] = d189
		ps1282.OverlayValues[190] = d190
		ps1282.OverlayValues[191] = d191
		ps1282.OverlayValues[192] = d192
		ps1282.OverlayValues[193] = d193
		ps1282.OverlayValues[194] = d194
		ps1282.OverlayValues[195] = d195
		ps1282.OverlayValues[196] = d196
		ps1282.OverlayValues[197] = d197
		ps1282.OverlayValues[198] = d198
		ps1282.OverlayValues[199] = d199
		ps1282.OverlayValues[200] = d200
		ps1282.OverlayValues[201] = d201
		ps1282.OverlayValues[202] = d202
		ps1282.OverlayValues[203] = d203
		ps1282.OverlayValues[204] = d204
		ps1282.OverlayValues[205] = d205
		ps1282.OverlayValues[206] = d206
		ps1282.OverlayValues[209] = d209
		ps1282.OverlayValues[386] = d386
		ps1282.OverlayValues[387] = d387
		ps1282.OverlayValues[388] = d388
		ps1282.OverlayValues[389] = d389
		ps1282.OverlayValues[391] = d391
		ps1282.OverlayValues[392] = d392
		ps1282.OverlayValues[393] = d393
		ps1282.OverlayValues[394] = d394
		ps1282.OverlayValues[395] = d395
		ps1282.OverlayValues[396] = d396
		ps1282.OverlayValues[397] = d397
		ps1282.OverlayValues[398] = d398
		ps1282.OverlayValues[400] = d400
		ps1282.OverlayValues[402] = d402
		ps1282.OverlayValues[403] = d403
		ps1282.OverlayValues[404] = d404
		ps1282.OverlayValues[508] = d508
		ps1282.OverlayValues[509] = d509
		ps1282.OverlayValues[512] = d512
		ps1282.OverlayValues[619] = d619
		ps1282.OverlayValues[620] = d620
		ps1282.OverlayValues[621] = d621
		ps1282.OverlayValues[622] = d622
		ps1282.OverlayValues[623] = d623
		ps1282.OverlayValues[625] = d625
		ps1282.OverlayValues[626] = d626
		ps1282.OverlayValues[627] = d627
		ps1282.OverlayValues[628] = d628
		ps1282.OverlayValues[629] = d629
		ps1282.OverlayValues[630] = d630
		ps1282.OverlayValues[631] = d631
		ps1282.OverlayValues[632] = d632
		ps1282.OverlayValues[633] = d633
		ps1282.OverlayValues[634] = d634
		ps1282.OverlayValues[635] = d635
		ps1282.OverlayValues[636] = d636
		ps1282.OverlayValues[637] = d637
		ps1282.OverlayValues[638] = d638
		ps1282.OverlayValues[639] = d639
		ps1282.OverlayValues[640] = d640
		ps1282.OverlayValues[641] = d641
		ps1282.OverlayValues[642] = d642
		ps1282.OverlayValues[643] = d643
		ps1282.OverlayValues[644] = d644
		ps1282.OverlayValues[645] = d645
		ps1282.OverlayValues[646] = d646
		ps1282.OverlayValues[647] = d647
		ps1282.OverlayValues[648] = d648
		ps1282.OverlayValues[649] = d649
		ps1282.OverlayValues[650] = d650
		ps1282.OverlayValues[651] = d651
		ps1282.OverlayValues[652] = d652
		ps1282.OverlayValues[653] = d653
		ps1282.OverlayValues[654] = d654
		ps1282.OverlayValues[655] = d655
		ps1282.OverlayValues[944] = d944
		ps1282.OverlayValues[945] = d945
		ps1282.OverlayValues[946] = d946
		ps1282.OverlayValues[948] = d948
		ps1282.OverlayValues[949] = d949
		ps1282.OverlayValues[950] = d950
		ps1282.OverlayValues[951] = d951
		ps1282.OverlayValues[952] = d952
		ps1282.OverlayValues[953] = d953
		ps1282.OverlayValues[954] = d954
		ps1282.OverlayValues[956] = d956
		ps1282.OverlayValues[958] = d958
		ps1282.OverlayValues[959] = d959
		ps1282.OverlayValues[1115] = d1115
		ps1282.OverlayValues[1116] = d1116
		ps1282.OverlayValues[1119] = d1119
		ps1282.OverlayValues[1278] = d1278
		ps1282.OverlayValues[1279] = d1279
		ps1282.OverlayValues[1280] = d1280
		ps1282.OverlayValues[1281] = d1281
		ps1282.PhiValues = make([]scm.JITValueDesc, 2)
		d1283 = d9
		ps1282.PhiValues[0] = d1283
		d1284 = d11
		ps1282.PhiValues[1] = d1284
		if ps1282.General && bbs[8].Rendered {
			ctx.EmitJmp(lbl9)
			return result
		}
		return bbs[8].RenderPS(ps1282)
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
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
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
		if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != scm.LocNone {
			d177 = ps.OverlayValues[177]
		}
		if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
			d178 = ps.OverlayValues[178]
		}
		if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != scm.LocNone {
			d179 = ps.OverlayValues[179]
		}
		if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != scm.LocNone {
			d180 = ps.OverlayValues[180]
		}
		if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != scm.LocNone {
			d181 = ps.OverlayValues[181]
		}
		if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != scm.LocNone {
			d182 = ps.OverlayValues[182]
		}
		if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != scm.LocNone {
			d183 = ps.OverlayValues[183]
		}
		if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
			d184 = ps.OverlayValues[184]
		}
		if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != scm.LocNone {
			d185 = ps.OverlayValues[185]
		}
		if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != scm.LocNone {
			d186 = ps.OverlayValues[186]
		}
		if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
			d187 = ps.OverlayValues[187]
		}
		if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != scm.LocNone {
			d188 = ps.OverlayValues[188]
		}
		if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != scm.LocNone {
			d189 = ps.OverlayValues[189]
		}
		if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != scm.LocNone {
			d190 = ps.OverlayValues[190]
		}
		if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != scm.LocNone {
			d191 = ps.OverlayValues[191]
		}
		if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != scm.LocNone {
			d192 = ps.OverlayValues[192]
		}
		if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != scm.LocNone {
			d193 = ps.OverlayValues[193]
		}
		if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != scm.LocNone {
			d194 = ps.OverlayValues[194]
		}
		if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != scm.LocNone {
			d195 = ps.OverlayValues[195]
		}
		if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
			d196 = ps.OverlayValues[196]
		}
		if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != scm.LocNone {
			d197 = ps.OverlayValues[197]
		}
		if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != scm.LocNone {
			d198 = ps.OverlayValues[198]
		}
		if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
			d199 = ps.OverlayValues[199]
		}
		if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
			d200 = ps.OverlayValues[200]
		}
		if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
			d201 = ps.OverlayValues[201]
		}
		if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != scm.LocNone {
			d202 = ps.OverlayValues[202]
		}
		if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != scm.LocNone {
			d203 = ps.OverlayValues[203]
		}
		if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != scm.LocNone {
			d204 = ps.OverlayValues[204]
		}
		if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != scm.LocNone {
			d205 = ps.OverlayValues[205]
		}
		if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != scm.LocNone {
			d206 = ps.OverlayValues[206]
		}
		if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != scm.LocNone {
			d209 = ps.OverlayValues[209]
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
		if len(ps.OverlayValues) > 400 && ps.OverlayValues[400].Loc != scm.LocNone {
			d400 = ps.OverlayValues[400]
		}
		if len(ps.OverlayValues) > 402 && ps.OverlayValues[402].Loc != scm.LocNone {
			d402 = ps.OverlayValues[402]
		}
		if len(ps.OverlayValues) > 403 && ps.OverlayValues[403].Loc != scm.LocNone {
			d403 = ps.OverlayValues[403]
		}
		if len(ps.OverlayValues) > 404 && ps.OverlayValues[404].Loc != scm.LocNone {
			d404 = ps.OverlayValues[404]
		}
		if len(ps.OverlayValues) > 508 && ps.OverlayValues[508].Loc != scm.LocNone {
			d508 = ps.OverlayValues[508]
		}
		if len(ps.OverlayValues) > 509 && ps.OverlayValues[509].Loc != scm.LocNone {
			d509 = ps.OverlayValues[509]
		}
		if len(ps.OverlayValues) > 512 && ps.OverlayValues[512].Loc != scm.LocNone {
			d512 = ps.OverlayValues[512]
		}
		if len(ps.OverlayValues) > 619 && ps.OverlayValues[619].Loc != scm.LocNone {
			d619 = ps.OverlayValues[619]
		}
		if len(ps.OverlayValues) > 620 && ps.OverlayValues[620].Loc != scm.LocNone {
			d620 = ps.OverlayValues[620]
		}
		if len(ps.OverlayValues) > 621 && ps.OverlayValues[621].Loc != scm.LocNone {
			d621 = ps.OverlayValues[621]
		}
		if len(ps.OverlayValues) > 622 && ps.OverlayValues[622].Loc != scm.LocNone {
			d622 = ps.OverlayValues[622]
		}
		if len(ps.OverlayValues) > 623 && ps.OverlayValues[623].Loc != scm.LocNone {
			d623 = ps.OverlayValues[623]
		}
		if len(ps.OverlayValues) > 625 && ps.OverlayValues[625].Loc != scm.LocNone {
			d625 = ps.OverlayValues[625]
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
		if len(ps.OverlayValues) > 629 && ps.OverlayValues[629].Loc != scm.LocNone {
			d629 = ps.OverlayValues[629]
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
		if len(ps.OverlayValues) > 637 && ps.OverlayValues[637].Loc != scm.LocNone {
			d637 = ps.OverlayValues[637]
		}
		if len(ps.OverlayValues) > 638 && ps.OverlayValues[638].Loc != scm.LocNone {
			d638 = ps.OverlayValues[638]
		}
		if len(ps.OverlayValues) > 639 && ps.OverlayValues[639].Loc != scm.LocNone {
			d639 = ps.OverlayValues[639]
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
		if len(ps.OverlayValues) > 644 && ps.OverlayValues[644].Loc != scm.LocNone {
			d644 = ps.OverlayValues[644]
		}
		if len(ps.OverlayValues) > 645 && ps.OverlayValues[645].Loc != scm.LocNone {
			d645 = ps.OverlayValues[645]
		}
		if len(ps.OverlayValues) > 646 && ps.OverlayValues[646].Loc != scm.LocNone {
			d646 = ps.OverlayValues[646]
		}
		if len(ps.OverlayValues) > 647 && ps.OverlayValues[647].Loc != scm.LocNone {
			d647 = ps.OverlayValues[647]
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
		if len(ps.OverlayValues) > 651 && ps.OverlayValues[651].Loc != scm.LocNone {
			d651 = ps.OverlayValues[651]
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
		if len(ps.OverlayValues) > 944 && ps.OverlayValues[944].Loc != scm.LocNone {
			d944 = ps.OverlayValues[944]
		}
		if len(ps.OverlayValues) > 945 && ps.OverlayValues[945].Loc != scm.LocNone {
			d945 = ps.OverlayValues[945]
		}
		if len(ps.OverlayValues) > 946 && ps.OverlayValues[946].Loc != scm.LocNone {
			d946 = ps.OverlayValues[946]
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
		if len(ps.OverlayValues) > 956 && ps.OverlayValues[956].Loc != scm.LocNone {
			d956 = ps.OverlayValues[956]
		}
		if len(ps.OverlayValues) > 958 && ps.OverlayValues[958].Loc != scm.LocNone {
			d958 = ps.OverlayValues[958]
		}
		if len(ps.OverlayValues) > 959 && ps.OverlayValues[959].Loc != scm.LocNone {
			d959 = ps.OverlayValues[959]
		}
		if len(ps.OverlayValues) > 1115 && ps.OverlayValues[1115].Loc != scm.LocNone {
			d1115 = ps.OverlayValues[1115]
		}
		if len(ps.OverlayValues) > 1116 && ps.OverlayValues[1116].Loc != scm.LocNone {
			d1116 = ps.OverlayValues[1116]
		}
		if len(ps.OverlayValues) > 1119 && ps.OverlayValues[1119].Loc != scm.LocNone {
			d1119 = ps.OverlayValues[1119]
		}
		if len(ps.OverlayValues) > 1278 && ps.OverlayValues[1278].Loc != scm.LocNone {
			d1278 = ps.OverlayValues[1278]
		}
		if len(ps.OverlayValues) > 1279 && ps.OverlayValues[1279].Loc != scm.LocNone {
			d1279 = ps.OverlayValues[1279]
		}
		if len(ps.OverlayValues) > 1280 && ps.OverlayValues[1280].Loc != scm.LocNone {
			d1280 = ps.OverlayValues[1280]
		}
		if len(ps.OverlayValues) > 1281 && ps.OverlayValues[1281].Loc != scm.LocNone {
			d1281 = ps.OverlayValues[1281]
		}
		if len(ps.OverlayValues) > 1283 && ps.OverlayValues[1283].Loc != scm.LocNone {
			d1283 = ps.OverlayValues[1283]
		}
		if len(ps.OverlayValues) > 1284 && ps.OverlayValues[1284].Loc != scm.LocNone {
			d1284 = ps.OverlayValues[1284]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d12)
		ctx.EnsureDesc(&d13)
		ctx.EnsureDescsTogether(&d12, &d13)
		var d1285 scm.JITValueDesc
		if d12.Loc == scm.LocImm && d13.Loc == scm.LocImm {
			d1285 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() + d13.Imm.Int())}
		} else if d13.Loc == scm.LocImm && d13.Imm.Int() == 0 {
			r108 := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitMovRegReg(r108, d12.Reg)
			d1285 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
			ctx.BindReg(r108, &d1285)
		} else if d12.Loc == scm.LocImm && d12.Imm.Int() == 0 {
			d1285 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d13.Reg}
			ctx.BindReg(d13.Reg, &d1285)
		} else if d12.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d13.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d12.Imm.Int()))
			ctx.EmitAddInt64(scratch, d13.Reg)
			d1285 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1285)
		} else if d13.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitMovRegReg(scratch, d12.Reg)
			if d13.Imm.Int() >= -2147483648 && d13.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d13.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d1285 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1285)
		} else {
			r109 := ctx.AllocRegExcept(d12.Reg, d13.Reg)
			ctx.EmitMovRegReg(r109, d12.Reg)
			ctx.EmitAddInt64(r109, d13.Reg)
			d1285 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r109}
			ctx.BindReg(r109, &d1285)
		}
		if d1285.Loc == scm.LocImm {
			d1285 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1285.Type, Imm: scm.NewInt(int64(uint64(d1285.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d1285.Reg, 32)
			ctx.EmitShrRegImm8(d1285.Reg, 32)
		}
		if d1285.Loc == scm.LocReg && d12.Loc == scm.LocReg && d1285.Reg == d12.Reg {
			ctx.TransferReg(d12.Reg)
			d12.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d1285)
		var d1286 scm.JITValueDesc
		if d1285.Loc == scm.LocImm {
			d1286 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1285.Imm.Int() / 2)}
		} else {
			r110 := ctx.AllocRegExcept(d1285.Reg)
			ctx.EmitMovRegReg(r110, d1285.Reg)
			ctx.EmitShrRegImm8(r110, 1)
			d1286 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
			ctx.BindReg(r110, &d1286)
		}
		if d1286.Loc == scm.LocImm {
			d1286 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1286.Type, Imm: scm.NewInt(int64(uint64(d1286.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d1286.Reg, 32)
			ctx.EmitShrRegImm8(d1286.Reg, 32)
		}
		if d1286.Loc == scm.LocReg && d1285.Loc == scm.LocReg && d1286.Reg == d1285.Reg {
			ctx.TransferReg(d1285.Reg)
			d1285.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1285)
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
			ctx.SyncDesc(&d1286)
			if d1286.Loc == scm.LocReg {
				ctx.ProtectReg(d1286.Reg)
			} else if d1286.Loc == scm.LocRegPair {
				ctx.ProtectReg(d1286.Reg)
				ctx.ProtectReg(d1286.Reg2)
			}
			d1287 = d1286
			if d1287.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d1287)
			d1288 = d1287
			if d1288.Loc == scm.LocImm {
				d1288 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1288.Type, Imm: scm.NewInt(int64(uint64(d1288.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d1288.Reg, 32)
				ctx.EmitShrRegImm8(d1288.Reg, 32)
			}
			if phiHomeOK2 {
				ctx.EmitMovToReg(r0, d1288)
			} else {
				ctx.EmitStoreToStack(d1288, int32(bbs[1].PhiBase)+int32(0))
			}
			d1289 = d12
			if d1289.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d1289)
			d1290 = d1289
			if d1290.Loc == scm.LocImm {
				d1290 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1290.Type, Imm: scm.NewInt(int64(uint64(d1290.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d1290.Reg, 32)
				ctx.EmitShrRegImm8(d1290.Reg, 32)
			}
			if phiHomeOK3 {
				ctx.EmitMovToReg(r1, d1290)
			} else {
				ctx.EmitStoreToStack(d1290, int32(bbs[1].PhiBase)+int32(16))
			}
			d1291 = d13
			if d1291.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d1291)
			d1292 = d1291
			if d1292.Loc == scm.LocImm {
				d1292 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1292.Type, Imm: scm.NewInt(int64(uint64(d1292.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d1292.Reg, 32)
				ctx.EmitShrRegImm8(d1292.Reg, 32)
			}
			if phiHomeOK4 {
				ctx.EmitMovToReg(r2, d1292)
			} else {
				ctx.EmitStoreToStack(d1292, int32(bbs[1].PhiBase)+int32(32))
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
			if d1286.Loc == scm.LocReg {
				ctx.UnprotectReg(d1286.Reg)
			} else if d1286.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d1286.Reg)
				ctx.UnprotectReg(d1286.Reg2)
			}
		}
		ps1293 := scm.PhiState{General: ps.General}
		ps1293.OverlayValues = make([]scm.JITValueDesc, 1293)
		ps1293.OverlayValues[5] = d5
		ps1293.OverlayValues[6] = d6
		ps1293.OverlayValues[7] = d7
		ps1293.OverlayValues[8] = d8
		ps1293.OverlayValues[9] = d9
		ps1293.OverlayValues[10] = d10
		ps1293.OverlayValues[11] = d11
		ps1293.OverlayValues[12] = d12
		ps1293.OverlayValues[13] = d13
		ps1293.OverlayValues[14] = d14
		ps1293.OverlayValues[15] = d15
		ps1293.OverlayValues[16] = d16
		ps1293.OverlayValues[17] = d17
		ps1293.OverlayValues[18] = d18
		ps1293.OverlayValues[19] = d19
		ps1293.OverlayValues[20] = d20
		ps1293.OverlayValues[21] = d21
		ps1293.OverlayValues[23] = d23
		ps1293.OverlayValues[24] = d24
		ps1293.OverlayValues[25] = d25
		ps1293.OverlayValues[26] = d26
		ps1293.OverlayValues[27] = d27
		ps1293.OverlayValues[28] = d28
		ps1293.OverlayValues[29] = d29
		ps1293.OverlayValues[30] = d30
		ps1293.OverlayValues[31] = d31
		ps1293.OverlayValues[32] = d32
		ps1293.OverlayValues[33] = d33
		ps1293.OverlayValues[34] = d34
		ps1293.OverlayValues[35] = d35
		ps1293.OverlayValues[36] = d36
		ps1293.OverlayValues[37] = d37
		ps1293.OverlayValues[38] = d38
		ps1293.OverlayValues[39] = d39
		ps1293.OverlayValues[40] = d40
		ps1293.OverlayValues[41] = d41
		ps1293.OverlayValues[42] = d42
		ps1293.OverlayValues[43] = d43
		ps1293.OverlayValues[44] = d44
		ps1293.OverlayValues[45] = d45
		ps1293.OverlayValues[46] = d46
		ps1293.OverlayValues[47] = d47
		ps1293.OverlayValues[48] = d48
		ps1293.OverlayValues[49] = d49
		ps1293.OverlayValues[50] = d50
		ps1293.OverlayValues[51] = d51
		ps1293.OverlayValues[52] = d52
		ps1293.OverlayValues[53] = d53
		ps1293.OverlayValues[54] = d54
		ps1293.OverlayValues[55] = d55
		ps1293.OverlayValues[56] = d56
		ps1293.OverlayValues[57] = d57
		ps1293.OverlayValues[60] = d60
		ps1293.OverlayValues[61] = d61
		ps1293.OverlayValues[62] = d62
		ps1293.OverlayValues[177] = d177
		ps1293.OverlayValues[178] = d178
		ps1293.OverlayValues[179] = d179
		ps1293.OverlayValues[180] = d180
		ps1293.OverlayValues[181] = d181
		ps1293.OverlayValues[182] = d182
		ps1293.OverlayValues[183] = d183
		ps1293.OverlayValues[184] = d184
		ps1293.OverlayValues[185] = d185
		ps1293.OverlayValues[186] = d186
		ps1293.OverlayValues[187] = d187
		ps1293.OverlayValues[188] = d188
		ps1293.OverlayValues[189] = d189
		ps1293.OverlayValues[190] = d190
		ps1293.OverlayValues[191] = d191
		ps1293.OverlayValues[192] = d192
		ps1293.OverlayValues[193] = d193
		ps1293.OverlayValues[194] = d194
		ps1293.OverlayValues[195] = d195
		ps1293.OverlayValues[196] = d196
		ps1293.OverlayValues[197] = d197
		ps1293.OverlayValues[198] = d198
		ps1293.OverlayValues[199] = d199
		ps1293.OverlayValues[200] = d200
		ps1293.OverlayValues[201] = d201
		ps1293.OverlayValues[202] = d202
		ps1293.OverlayValues[203] = d203
		ps1293.OverlayValues[204] = d204
		ps1293.OverlayValues[205] = d205
		ps1293.OverlayValues[206] = d206
		ps1293.OverlayValues[209] = d209
		ps1293.OverlayValues[386] = d386
		ps1293.OverlayValues[387] = d387
		ps1293.OverlayValues[388] = d388
		ps1293.OverlayValues[389] = d389
		ps1293.OverlayValues[391] = d391
		ps1293.OverlayValues[392] = d392
		ps1293.OverlayValues[393] = d393
		ps1293.OverlayValues[394] = d394
		ps1293.OverlayValues[395] = d395
		ps1293.OverlayValues[396] = d396
		ps1293.OverlayValues[397] = d397
		ps1293.OverlayValues[398] = d398
		ps1293.OverlayValues[400] = d400
		ps1293.OverlayValues[402] = d402
		ps1293.OverlayValues[403] = d403
		ps1293.OverlayValues[404] = d404
		ps1293.OverlayValues[508] = d508
		ps1293.OverlayValues[509] = d509
		ps1293.OverlayValues[512] = d512
		ps1293.OverlayValues[619] = d619
		ps1293.OverlayValues[620] = d620
		ps1293.OverlayValues[621] = d621
		ps1293.OverlayValues[622] = d622
		ps1293.OverlayValues[623] = d623
		ps1293.OverlayValues[625] = d625
		ps1293.OverlayValues[626] = d626
		ps1293.OverlayValues[627] = d627
		ps1293.OverlayValues[628] = d628
		ps1293.OverlayValues[629] = d629
		ps1293.OverlayValues[630] = d630
		ps1293.OverlayValues[631] = d631
		ps1293.OverlayValues[632] = d632
		ps1293.OverlayValues[633] = d633
		ps1293.OverlayValues[634] = d634
		ps1293.OverlayValues[635] = d635
		ps1293.OverlayValues[636] = d636
		ps1293.OverlayValues[637] = d637
		ps1293.OverlayValues[638] = d638
		ps1293.OverlayValues[639] = d639
		ps1293.OverlayValues[640] = d640
		ps1293.OverlayValues[641] = d641
		ps1293.OverlayValues[642] = d642
		ps1293.OverlayValues[643] = d643
		ps1293.OverlayValues[644] = d644
		ps1293.OverlayValues[645] = d645
		ps1293.OverlayValues[646] = d646
		ps1293.OverlayValues[647] = d647
		ps1293.OverlayValues[648] = d648
		ps1293.OverlayValues[649] = d649
		ps1293.OverlayValues[650] = d650
		ps1293.OverlayValues[651] = d651
		ps1293.OverlayValues[652] = d652
		ps1293.OverlayValues[653] = d653
		ps1293.OverlayValues[654] = d654
		ps1293.OverlayValues[655] = d655
		ps1293.OverlayValues[944] = d944
		ps1293.OverlayValues[945] = d945
		ps1293.OverlayValues[946] = d946
		ps1293.OverlayValues[948] = d948
		ps1293.OverlayValues[949] = d949
		ps1293.OverlayValues[950] = d950
		ps1293.OverlayValues[951] = d951
		ps1293.OverlayValues[952] = d952
		ps1293.OverlayValues[953] = d953
		ps1293.OverlayValues[954] = d954
		ps1293.OverlayValues[956] = d956
		ps1293.OverlayValues[958] = d958
		ps1293.OverlayValues[959] = d959
		ps1293.OverlayValues[1115] = d1115
		ps1293.OverlayValues[1116] = d1116
		ps1293.OverlayValues[1119] = d1119
		ps1293.OverlayValues[1278] = d1278
		ps1293.OverlayValues[1279] = d1279
		ps1293.OverlayValues[1280] = d1280
		ps1293.OverlayValues[1281] = d1281
		ps1293.OverlayValues[1283] = d1283
		ps1293.OverlayValues[1284] = d1284
		ps1293.OverlayValues[1285] = d1285
		ps1293.OverlayValues[1286] = d1286
		ps1293.OverlayValues[1287] = d1287
		ps1293.OverlayValues[1288] = d1288
		ps1293.OverlayValues[1289] = d1289
		ps1293.OverlayValues[1290] = d1290
		ps1293.OverlayValues[1291] = d1291
		ps1293.OverlayValues[1292] = d1292
		ps1293.PhiValues = make([]scm.JITValueDesc, 3)
		d1294 = d1286
		ps1293.PhiValues[0] = d1294
		d1295 = d12
		ps1293.PhiValues[1] = d1295
		d1296 = d13
		ps1293.PhiValues[2] = d1296
		if ps1293.General && bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		return bbs[1].RenderPS(ps1293)
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
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
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
		if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != scm.LocNone {
			d177 = ps.OverlayValues[177]
		}
		if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
			d178 = ps.OverlayValues[178]
		}
		if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != scm.LocNone {
			d179 = ps.OverlayValues[179]
		}
		if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != scm.LocNone {
			d180 = ps.OverlayValues[180]
		}
		if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != scm.LocNone {
			d181 = ps.OverlayValues[181]
		}
		if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != scm.LocNone {
			d182 = ps.OverlayValues[182]
		}
		if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != scm.LocNone {
			d183 = ps.OverlayValues[183]
		}
		if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
			d184 = ps.OverlayValues[184]
		}
		if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != scm.LocNone {
			d185 = ps.OverlayValues[185]
		}
		if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != scm.LocNone {
			d186 = ps.OverlayValues[186]
		}
		if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
			d187 = ps.OverlayValues[187]
		}
		if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != scm.LocNone {
			d188 = ps.OverlayValues[188]
		}
		if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != scm.LocNone {
			d189 = ps.OverlayValues[189]
		}
		if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != scm.LocNone {
			d190 = ps.OverlayValues[190]
		}
		if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != scm.LocNone {
			d191 = ps.OverlayValues[191]
		}
		if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != scm.LocNone {
			d192 = ps.OverlayValues[192]
		}
		if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != scm.LocNone {
			d193 = ps.OverlayValues[193]
		}
		if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != scm.LocNone {
			d194 = ps.OverlayValues[194]
		}
		if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != scm.LocNone {
			d195 = ps.OverlayValues[195]
		}
		if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
			d196 = ps.OverlayValues[196]
		}
		if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != scm.LocNone {
			d197 = ps.OverlayValues[197]
		}
		if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != scm.LocNone {
			d198 = ps.OverlayValues[198]
		}
		if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
			d199 = ps.OverlayValues[199]
		}
		if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
			d200 = ps.OverlayValues[200]
		}
		if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
			d201 = ps.OverlayValues[201]
		}
		if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != scm.LocNone {
			d202 = ps.OverlayValues[202]
		}
		if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != scm.LocNone {
			d203 = ps.OverlayValues[203]
		}
		if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != scm.LocNone {
			d204 = ps.OverlayValues[204]
		}
		if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != scm.LocNone {
			d205 = ps.OverlayValues[205]
		}
		if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != scm.LocNone {
			d206 = ps.OverlayValues[206]
		}
		if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != scm.LocNone {
			d209 = ps.OverlayValues[209]
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
		if len(ps.OverlayValues) > 400 && ps.OverlayValues[400].Loc != scm.LocNone {
			d400 = ps.OverlayValues[400]
		}
		if len(ps.OverlayValues) > 402 && ps.OverlayValues[402].Loc != scm.LocNone {
			d402 = ps.OverlayValues[402]
		}
		if len(ps.OverlayValues) > 403 && ps.OverlayValues[403].Loc != scm.LocNone {
			d403 = ps.OverlayValues[403]
		}
		if len(ps.OverlayValues) > 404 && ps.OverlayValues[404].Loc != scm.LocNone {
			d404 = ps.OverlayValues[404]
		}
		if len(ps.OverlayValues) > 508 && ps.OverlayValues[508].Loc != scm.LocNone {
			d508 = ps.OverlayValues[508]
		}
		if len(ps.OverlayValues) > 509 && ps.OverlayValues[509].Loc != scm.LocNone {
			d509 = ps.OverlayValues[509]
		}
		if len(ps.OverlayValues) > 512 && ps.OverlayValues[512].Loc != scm.LocNone {
			d512 = ps.OverlayValues[512]
		}
		if len(ps.OverlayValues) > 619 && ps.OverlayValues[619].Loc != scm.LocNone {
			d619 = ps.OverlayValues[619]
		}
		if len(ps.OverlayValues) > 620 && ps.OverlayValues[620].Loc != scm.LocNone {
			d620 = ps.OverlayValues[620]
		}
		if len(ps.OverlayValues) > 621 && ps.OverlayValues[621].Loc != scm.LocNone {
			d621 = ps.OverlayValues[621]
		}
		if len(ps.OverlayValues) > 622 && ps.OverlayValues[622].Loc != scm.LocNone {
			d622 = ps.OverlayValues[622]
		}
		if len(ps.OverlayValues) > 623 && ps.OverlayValues[623].Loc != scm.LocNone {
			d623 = ps.OverlayValues[623]
		}
		if len(ps.OverlayValues) > 625 && ps.OverlayValues[625].Loc != scm.LocNone {
			d625 = ps.OverlayValues[625]
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
		if len(ps.OverlayValues) > 629 && ps.OverlayValues[629].Loc != scm.LocNone {
			d629 = ps.OverlayValues[629]
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
		if len(ps.OverlayValues) > 637 && ps.OverlayValues[637].Loc != scm.LocNone {
			d637 = ps.OverlayValues[637]
		}
		if len(ps.OverlayValues) > 638 && ps.OverlayValues[638].Loc != scm.LocNone {
			d638 = ps.OverlayValues[638]
		}
		if len(ps.OverlayValues) > 639 && ps.OverlayValues[639].Loc != scm.LocNone {
			d639 = ps.OverlayValues[639]
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
		if len(ps.OverlayValues) > 644 && ps.OverlayValues[644].Loc != scm.LocNone {
			d644 = ps.OverlayValues[644]
		}
		if len(ps.OverlayValues) > 645 && ps.OverlayValues[645].Loc != scm.LocNone {
			d645 = ps.OverlayValues[645]
		}
		if len(ps.OverlayValues) > 646 && ps.OverlayValues[646].Loc != scm.LocNone {
			d646 = ps.OverlayValues[646]
		}
		if len(ps.OverlayValues) > 647 && ps.OverlayValues[647].Loc != scm.LocNone {
			d647 = ps.OverlayValues[647]
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
		if len(ps.OverlayValues) > 651 && ps.OverlayValues[651].Loc != scm.LocNone {
			d651 = ps.OverlayValues[651]
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
		if len(ps.OverlayValues) > 944 && ps.OverlayValues[944].Loc != scm.LocNone {
			d944 = ps.OverlayValues[944]
		}
		if len(ps.OverlayValues) > 945 && ps.OverlayValues[945].Loc != scm.LocNone {
			d945 = ps.OverlayValues[945]
		}
		if len(ps.OverlayValues) > 946 && ps.OverlayValues[946].Loc != scm.LocNone {
			d946 = ps.OverlayValues[946]
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
		if len(ps.OverlayValues) > 956 && ps.OverlayValues[956].Loc != scm.LocNone {
			d956 = ps.OverlayValues[956]
		}
		if len(ps.OverlayValues) > 958 && ps.OverlayValues[958].Loc != scm.LocNone {
			d958 = ps.OverlayValues[958]
		}
		if len(ps.OverlayValues) > 959 && ps.OverlayValues[959].Loc != scm.LocNone {
			d959 = ps.OverlayValues[959]
		}
		if len(ps.OverlayValues) > 1115 && ps.OverlayValues[1115].Loc != scm.LocNone {
			d1115 = ps.OverlayValues[1115]
		}
		if len(ps.OverlayValues) > 1116 && ps.OverlayValues[1116].Loc != scm.LocNone {
			d1116 = ps.OverlayValues[1116]
		}
		if len(ps.OverlayValues) > 1119 && ps.OverlayValues[1119].Loc != scm.LocNone {
			d1119 = ps.OverlayValues[1119]
		}
		if len(ps.OverlayValues) > 1278 && ps.OverlayValues[1278].Loc != scm.LocNone {
			d1278 = ps.OverlayValues[1278]
		}
		if len(ps.OverlayValues) > 1279 && ps.OverlayValues[1279].Loc != scm.LocNone {
			d1279 = ps.OverlayValues[1279]
		}
		if len(ps.OverlayValues) > 1280 && ps.OverlayValues[1280].Loc != scm.LocNone {
			d1280 = ps.OverlayValues[1280]
		}
		if len(ps.OverlayValues) > 1281 && ps.OverlayValues[1281].Loc != scm.LocNone {
			d1281 = ps.OverlayValues[1281]
		}
		if len(ps.OverlayValues) > 1283 && ps.OverlayValues[1283].Loc != scm.LocNone {
			d1283 = ps.OverlayValues[1283]
		}
		if len(ps.OverlayValues) > 1284 && ps.OverlayValues[1284].Loc != scm.LocNone {
			d1284 = ps.OverlayValues[1284]
		}
		if len(ps.OverlayValues) > 1285 && ps.OverlayValues[1285].Loc != scm.LocNone {
			d1285 = ps.OverlayValues[1285]
		}
		if len(ps.OverlayValues) > 1286 && ps.OverlayValues[1286].Loc != scm.LocNone {
			d1286 = ps.OverlayValues[1286]
		}
		if len(ps.OverlayValues) > 1287 && ps.OverlayValues[1287].Loc != scm.LocNone {
			d1287 = ps.OverlayValues[1287]
		}
		if len(ps.OverlayValues) > 1288 && ps.OverlayValues[1288].Loc != scm.LocNone {
			d1288 = ps.OverlayValues[1288]
		}
		if len(ps.OverlayValues) > 1289 && ps.OverlayValues[1289].Loc != scm.LocNone {
			d1289 = ps.OverlayValues[1289]
		}
		if len(ps.OverlayValues) > 1290 && ps.OverlayValues[1290].Loc != scm.LocNone {
			d1290 = ps.OverlayValues[1290]
		}
		if len(ps.OverlayValues) > 1291 && ps.OverlayValues[1291].Loc != scm.LocNone {
			d1291 = ps.OverlayValues[1291]
		}
		if len(ps.OverlayValues) > 1292 && ps.OverlayValues[1292].Loc != scm.LocNone {
			d1292 = ps.OverlayValues[1292]
		}
		if len(ps.OverlayValues) > 1294 && ps.OverlayValues[1294].Loc != scm.LocNone {
			d1294 = ps.OverlayValues[1294]
		}
		if len(ps.OverlayValues) > 1295 && ps.OverlayValues[1295].Loc != scm.LocNone {
			d1295 = ps.OverlayValues[1295]
		}
		if len(ps.OverlayValues) > 1296 && ps.OverlayValues[1296].Loc != scm.LocNone {
			d1296 = ps.OverlayValues[1296]
		}
		ctx.ReclaimUntrackedRegs()
		d1297 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d1298 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r3, Reg2: r4}
		ctx.BindReg(r3, &d1298)
		ctx.BindReg(r4, &d1298)
		ctx.EnsureDesc(&d1297)
		if d1297.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d1297, &d1298)
		} else {
			switch d1297.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d1298, d1297)
			case scm.TagInt:
				ctx.EmitMakeInt(d1298, d1297)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d1298, d1297)
			case scm.TagNil:
				ctx.EmitMakeNil(d1298)
			default:
				ctx.EmitMovPairToResult(&d1297, &d1298)
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
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
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
		if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != scm.LocNone {
			d177 = ps.OverlayValues[177]
		}
		if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
			d178 = ps.OverlayValues[178]
		}
		if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != scm.LocNone {
			d179 = ps.OverlayValues[179]
		}
		if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != scm.LocNone {
			d180 = ps.OverlayValues[180]
		}
		if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != scm.LocNone {
			d181 = ps.OverlayValues[181]
		}
		if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != scm.LocNone {
			d182 = ps.OverlayValues[182]
		}
		if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != scm.LocNone {
			d183 = ps.OverlayValues[183]
		}
		if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
			d184 = ps.OverlayValues[184]
		}
		if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != scm.LocNone {
			d185 = ps.OverlayValues[185]
		}
		if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != scm.LocNone {
			d186 = ps.OverlayValues[186]
		}
		if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
			d187 = ps.OverlayValues[187]
		}
		if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != scm.LocNone {
			d188 = ps.OverlayValues[188]
		}
		if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != scm.LocNone {
			d189 = ps.OverlayValues[189]
		}
		if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != scm.LocNone {
			d190 = ps.OverlayValues[190]
		}
		if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != scm.LocNone {
			d191 = ps.OverlayValues[191]
		}
		if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != scm.LocNone {
			d192 = ps.OverlayValues[192]
		}
		if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != scm.LocNone {
			d193 = ps.OverlayValues[193]
		}
		if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != scm.LocNone {
			d194 = ps.OverlayValues[194]
		}
		if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != scm.LocNone {
			d195 = ps.OverlayValues[195]
		}
		if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
			d196 = ps.OverlayValues[196]
		}
		if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != scm.LocNone {
			d197 = ps.OverlayValues[197]
		}
		if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != scm.LocNone {
			d198 = ps.OverlayValues[198]
		}
		if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
			d199 = ps.OverlayValues[199]
		}
		if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
			d200 = ps.OverlayValues[200]
		}
		if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
			d201 = ps.OverlayValues[201]
		}
		if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != scm.LocNone {
			d202 = ps.OverlayValues[202]
		}
		if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != scm.LocNone {
			d203 = ps.OverlayValues[203]
		}
		if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != scm.LocNone {
			d204 = ps.OverlayValues[204]
		}
		if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != scm.LocNone {
			d205 = ps.OverlayValues[205]
		}
		if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != scm.LocNone {
			d206 = ps.OverlayValues[206]
		}
		if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != scm.LocNone {
			d209 = ps.OverlayValues[209]
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
		if len(ps.OverlayValues) > 400 && ps.OverlayValues[400].Loc != scm.LocNone {
			d400 = ps.OverlayValues[400]
		}
		if len(ps.OverlayValues) > 402 && ps.OverlayValues[402].Loc != scm.LocNone {
			d402 = ps.OverlayValues[402]
		}
		if len(ps.OverlayValues) > 403 && ps.OverlayValues[403].Loc != scm.LocNone {
			d403 = ps.OverlayValues[403]
		}
		if len(ps.OverlayValues) > 404 && ps.OverlayValues[404].Loc != scm.LocNone {
			d404 = ps.OverlayValues[404]
		}
		if len(ps.OverlayValues) > 508 && ps.OverlayValues[508].Loc != scm.LocNone {
			d508 = ps.OverlayValues[508]
		}
		if len(ps.OverlayValues) > 509 && ps.OverlayValues[509].Loc != scm.LocNone {
			d509 = ps.OverlayValues[509]
		}
		if len(ps.OverlayValues) > 512 && ps.OverlayValues[512].Loc != scm.LocNone {
			d512 = ps.OverlayValues[512]
		}
		if len(ps.OverlayValues) > 619 && ps.OverlayValues[619].Loc != scm.LocNone {
			d619 = ps.OverlayValues[619]
		}
		if len(ps.OverlayValues) > 620 && ps.OverlayValues[620].Loc != scm.LocNone {
			d620 = ps.OverlayValues[620]
		}
		if len(ps.OverlayValues) > 621 && ps.OverlayValues[621].Loc != scm.LocNone {
			d621 = ps.OverlayValues[621]
		}
		if len(ps.OverlayValues) > 622 && ps.OverlayValues[622].Loc != scm.LocNone {
			d622 = ps.OverlayValues[622]
		}
		if len(ps.OverlayValues) > 623 && ps.OverlayValues[623].Loc != scm.LocNone {
			d623 = ps.OverlayValues[623]
		}
		if len(ps.OverlayValues) > 625 && ps.OverlayValues[625].Loc != scm.LocNone {
			d625 = ps.OverlayValues[625]
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
		if len(ps.OverlayValues) > 629 && ps.OverlayValues[629].Loc != scm.LocNone {
			d629 = ps.OverlayValues[629]
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
		if len(ps.OverlayValues) > 637 && ps.OverlayValues[637].Loc != scm.LocNone {
			d637 = ps.OverlayValues[637]
		}
		if len(ps.OverlayValues) > 638 && ps.OverlayValues[638].Loc != scm.LocNone {
			d638 = ps.OverlayValues[638]
		}
		if len(ps.OverlayValues) > 639 && ps.OverlayValues[639].Loc != scm.LocNone {
			d639 = ps.OverlayValues[639]
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
		if len(ps.OverlayValues) > 644 && ps.OverlayValues[644].Loc != scm.LocNone {
			d644 = ps.OverlayValues[644]
		}
		if len(ps.OverlayValues) > 645 && ps.OverlayValues[645].Loc != scm.LocNone {
			d645 = ps.OverlayValues[645]
		}
		if len(ps.OverlayValues) > 646 && ps.OverlayValues[646].Loc != scm.LocNone {
			d646 = ps.OverlayValues[646]
		}
		if len(ps.OverlayValues) > 647 && ps.OverlayValues[647].Loc != scm.LocNone {
			d647 = ps.OverlayValues[647]
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
		if len(ps.OverlayValues) > 651 && ps.OverlayValues[651].Loc != scm.LocNone {
			d651 = ps.OverlayValues[651]
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
		if len(ps.OverlayValues) > 944 && ps.OverlayValues[944].Loc != scm.LocNone {
			d944 = ps.OverlayValues[944]
		}
		if len(ps.OverlayValues) > 945 && ps.OverlayValues[945].Loc != scm.LocNone {
			d945 = ps.OverlayValues[945]
		}
		if len(ps.OverlayValues) > 946 && ps.OverlayValues[946].Loc != scm.LocNone {
			d946 = ps.OverlayValues[946]
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
		if len(ps.OverlayValues) > 956 && ps.OverlayValues[956].Loc != scm.LocNone {
			d956 = ps.OverlayValues[956]
		}
		if len(ps.OverlayValues) > 958 && ps.OverlayValues[958].Loc != scm.LocNone {
			d958 = ps.OverlayValues[958]
		}
		if len(ps.OverlayValues) > 959 && ps.OverlayValues[959].Loc != scm.LocNone {
			d959 = ps.OverlayValues[959]
		}
		if len(ps.OverlayValues) > 1115 && ps.OverlayValues[1115].Loc != scm.LocNone {
			d1115 = ps.OverlayValues[1115]
		}
		if len(ps.OverlayValues) > 1116 && ps.OverlayValues[1116].Loc != scm.LocNone {
			d1116 = ps.OverlayValues[1116]
		}
		if len(ps.OverlayValues) > 1119 && ps.OverlayValues[1119].Loc != scm.LocNone {
			d1119 = ps.OverlayValues[1119]
		}
		if len(ps.OverlayValues) > 1278 && ps.OverlayValues[1278].Loc != scm.LocNone {
			d1278 = ps.OverlayValues[1278]
		}
		if len(ps.OverlayValues) > 1279 && ps.OverlayValues[1279].Loc != scm.LocNone {
			d1279 = ps.OverlayValues[1279]
		}
		if len(ps.OverlayValues) > 1280 && ps.OverlayValues[1280].Loc != scm.LocNone {
			d1280 = ps.OverlayValues[1280]
		}
		if len(ps.OverlayValues) > 1281 && ps.OverlayValues[1281].Loc != scm.LocNone {
			d1281 = ps.OverlayValues[1281]
		}
		if len(ps.OverlayValues) > 1283 && ps.OverlayValues[1283].Loc != scm.LocNone {
			d1283 = ps.OverlayValues[1283]
		}
		if len(ps.OverlayValues) > 1284 && ps.OverlayValues[1284].Loc != scm.LocNone {
			d1284 = ps.OverlayValues[1284]
		}
		if len(ps.OverlayValues) > 1285 && ps.OverlayValues[1285].Loc != scm.LocNone {
			d1285 = ps.OverlayValues[1285]
		}
		if len(ps.OverlayValues) > 1286 && ps.OverlayValues[1286].Loc != scm.LocNone {
			d1286 = ps.OverlayValues[1286]
		}
		if len(ps.OverlayValues) > 1287 && ps.OverlayValues[1287].Loc != scm.LocNone {
			d1287 = ps.OverlayValues[1287]
		}
		if len(ps.OverlayValues) > 1288 && ps.OverlayValues[1288].Loc != scm.LocNone {
			d1288 = ps.OverlayValues[1288]
		}
		if len(ps.OverlayValues) > 1289 && ps.OverlayValues[1289].Loc != scm.LocNone {
			d1289 = ps.OverlayValues[1289]
		}
		if len(ps.OverlayValues) > 1290 && ps.OverlayValues[1290].Loc != scm.LocNone {
			d1290 = ps.OverlayValues[1290]
		}
		if len(ps.OverlayValues) > 1291 && ps.OverlayValues[1291].Loc != scm.LocNone {
			d1291 = ps.OverlayValues[1291]
		}
		if len(ps.OverlayValues) > 1292 && ps.OverlayValues[1292].Loc != scm.LocNone {
			d1292 = ps.OverlayValues[1292]
		}
		if len(ps.OverlayValues) > 1294 && ps.OverlayValues[1294].Loc != scm.LocNone {
			d1294 = ps.OverlayValues[1294]
		}
		if len(ps.OverlayValues) > 1295 && ps.OverlayValues[1295].Loc != scm.LocNone {
			d1295 = ps.OverlayValues[1295]
		}
		if len(ps.OverlayValues) > 1296 && ps.OverlayValues[1296].Loc != scm.LocNone {
			d1296 = ps.OverlayValues[1296]
		}
		if len(ps.OverlayValues) > 1297 && ps.OverlayValues[1297].Loc != scm.LocNone {
			d1297 = ps.OverlayValues[1297]
		}
		if len(ps.OverlayValues) > 1298 && ps.OverlayValues[1298].Loc != scm.LocNone {
			d1298 = ps.OverlayValues[1298]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d8)
		d1299 = d8
		_ = d1299
		ctx.StabilizeDescForControlFlow(&d1299)
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
		ctx.EnsureDesc(&d1299)
		ctx.EnsureDesc(&d1299)
		var d1300 scm.JITValueDesc
		if d1299.Loc == scm.LocImm {
			d1300 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d1299.Imm.Int()))))}
		} else {
			r111 := ctx.AllocReg()
			ctx.EmitMovRegReg(r111, d1299.Reg)
			ctx.EmitShlRegImm8(r111, 32)
			ctx.EmitShrRegImm8(r111, 32)
			d1300 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r111}
			ctx.BindReg(r111, &d1300)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d1301 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d1301 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 48)
			r112 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r112, thisptr.Reg, off)
			d1301 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r112}
			ctx.BindReg(r112, &d1301)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1301)
		ctx.EnsureDesc(&d1301)
		var d1302 scm.JITValueDesc
		if d1301.Loc == scm.LocImm {
			d1302 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d1301.Imm.Int()))))}
		} else {
			r113 := ctx.AllocReg()
			ctx.EmitMovRegReg(r113, d1301.Reg)
			ctx.EmitShlRegImm8(r113, 56)
			ctx.EmitShrRegImm8(r113, 56)
			d1302 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r113}
			ctx.BindReg(r113, &d1302)
		}
		ctx.FreeDesc(&d1301)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1300)
		ctx.EnsureDesc(&d1302)
		ctx.EnsureDescsTogether(&d1300, &d1302)
		var d1303 scm.JITValueDesc
		if d1300.Loc == scm.LocImm && d1302.Loc == scm.LocImm {
			d1303 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1300.Imm.Int() * d1302.Imm.Int())}
		} else if d1300.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1302.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1300.Imm.Int()))
			ctx.EmitImulInt64(scratch, d1302.Reg)
			d1303 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1303)
		} else if d1302.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1300.Reg)
			ctx.EmitMovRegReg(scratch, d1300.Reg)
			if d1302.Imm.Int() >= -2147483648 && d1302.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d1302.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1302.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d1303 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1303)
		} else {
			r114 := ctx.AllocRegExcept(d1300.Reg, d1302.Reg)
			ctx.EmitMovRegReg(r114, d1300.Reg)
			ctx.EmitImulInt64(r114, d1302.Reg)
			d1303 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r114}
			ctx.BindReg(r114, &d1303)
		}
		if d1303.Loc == scm.LocReg && d1300.Loc == scm.LocReg && d1303.Reg == d1300.Reg {
			ctx.TransferReg(d1300.Reg)
			d1300.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1300)
		ctx.FreeDesc(&d1302)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1303)
		var d1304 scm.JITValueDesc
		if d1303.Loc == scm.LocImm {
			d1304 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1303.Imm.Int() / 64)}
		} else {
			r115 := ctx.AllocRegExcept(d1303.Reg)
			ctx.EmitMovRegReg(r115, d1303.Reg)
			ctx.EmitShrRegImm8(r115, 6)
			d1304 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
			ctx.BindReg(r115, &d1304)
		}
		if d1304.Loc == scm.LocReg && d1303.Loc == scm.LocReg && d1304.Reg == d1303.Reg {
			ctx.TransferReg(d1303.Reg)
			d1303.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1303)
		var d1305 scm.JITValueDesc
		if d1303.Loc == scm.LocImm {
			d1305 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1303.Imm.Int() % 64)}
		} else {
			r116 := ctx.AllocRegExcept(d1303.Reg)
			ctx.EmitMovRegReg(r116, d1303.Reg)
			ctx.EmitAndRegImm32(r116, 63)
			d1305 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r116}
			ctx.BindReg(r116, &d1305)
		}
		if d1305.Loc == scm.LocReg && d1303.Loc == scm.LocReg && d1305.Reg == d1303.Reg {
			ctx.TransferReg(d1303.Reg)
			d1303.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1303)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d1306 scm.JITValueDesc
		r117 := ctx.AllocReg()
		r118 := ctx.AllocRegExcept(r117)
		r119 := ctx.AllocRegExcept(r117, r118)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			ctx.EmitMovRegImm64(r117, uint64(dataPtr))
			ctx.EmitMovRegImm64(r118, uint64(sliceLen))
			ctx.EmitMovRegImm64(r119, uint64(sliceCap))
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
			ctx.EmitMovRegMem(r117, thisptr.Reg, off)
			ctx.EmitMovRegMem(r118, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r119, thisptr.Reg, off+16)
		}
		d1306 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r117, Reg2: r118, Reg3: r119}
		ctx.BindReg(r117, &d1306)
		ctx.BindReg(r118, &d1306)
		ctx.BindReg(r119, &d1306)
		ctx.BindReg(r117, &d1306)
		ctx.BindReg(r118, &d1306)
		ctx.BindReg(r119, &d1306)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1304)
		ctx.ReclaimUntrackedRegs()
		d1308 = ctx.EmitSliceElementAddress(&d1306, &d1304, 8)
		ctx.EnsureDesc(&d1308)
		ctx.EmitMovRegMem(d1308.Reg, d1308.Reg, 0)
		d1307 = d1308
		d1307.Type = scm.TagInt
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1307)
		ctx.EnsureDesc(&d1305)
		ctx.EnsureDescsTogether(&d1307, &d1305)
		var d1309 scm.JITValueDesc
		if d1307.Loc == scm.LocImm && d1305.Loc == scm.LocImm {
			d1309 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1307.Imm.Int()) << uint64(d1305.Imm.Int())))}
		} else if d1305.Loc == scm.LocImm {
			r120 := ctx.AllocRegExcept(d1307.Reg)
			ctx.EmitMovRegReg(r120, d1307.Reg)
			ctx.EmitShlRegImm8(r120, uint8(d1305.Imm.Int()))
			d1309 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r120}
			ctx.BindReg(r120, &d1309)
		} else {
			{
				shiftSrc := d1307.Reg
				r121 := ctx.AllocRegExcept(d1307.Reg, d1305.Reg)
				ctx.EmitMovRegReg(r121, d1307.Reg)
				shiftSrc = r121
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d1305.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d1305.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d1305.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d1309 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d1309)
			}
		}
		if d1309.Loc == scm.LocReg && d1307.Loc == scm.LocReg && d1309.Reg == d1307.Reg {
			ctx.TransferReg(d1307.Reg)
			d1307.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1307)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1304)
		ctx.EnsureDesc(&d1304)
		var d1310 scm.JITValueDesc
		if d1304.Loc == scm.LocImm {
			d1310 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1304.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d1304.Reg)
			ctx.EmitMovRegReg(scratch, d1304.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d1310 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1310)
		}
		if d1310.Loc == scm.LocReg && d1304.Loc == scm.LocReg && d1310.Reg == d1304.Reg {
			ctx.TransferReg(d1304.Reg)
			d1304.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1304)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1310)
		ctx.ReclaimUntrackedRegs()
		d1312 = ctx.EmitSliceElementAddress(&d1306, &d1310, 8)
		ctx.EnsureDesc(&d1312)
		ctx.EmitMovRegMem(d1312.Reg, d1312.Reg, 0)
		d1311 = d1312
		d1311.Type = scm.TagInt
		ctx.FreeDesc(&d1310)
		ctx.ReclaimUntrackedRegs()
		d1313 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d1305)
		ctx.EnsureDescsTogether(&d1313, &d1305)
		var d1314 scm.JITValueDesc
		if d1313.Loc == scm.LocImm && d1305.Loc == scm.LocImm {
			d1314 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1313.Imm.Int() - d1305.Imm.Int())}
		} else if d1305.Loc == scm.LocImm && d1305.Imm.Int() == 0 {
			r122 := ctx.AllocRegExcept(d1313.Reg)
			ctx.EmitMovRegReg(r122, d1313.Reg)
			d1314 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
			ctx.BindReg(r122, &d1314)
		} else if d1313.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1305.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1313.Imm.Int()))
			ctx.EmitSubInt64(scratch, d1305.Reg)
			d1314 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1314)
		} else if d1305.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1313.Reg)
			ctx.EmitMovRegReg(scratch, d1313.Reg)
			if d1305.Imm.Int() >= -2147483648 && d1305.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d1305.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1305.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d1314 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1314)
		} else {
			r123 := ctx.AllocRegExcept(d1313.Reg, d1305.Reg)
			ctx.EmitMovRegReg(r123, d1313.Reg)
			ctx.EmitSubInt64(r123, d1305.Reg)
			d1314 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
			ctx.BindReg(r123, &d1314)
		}
		if d1314.Loc == scm.LocReg && d1313.Loc == scm.LocReg && d1314.Reg == d1313.Reg {
			ctx.TransferReg(d1313.Reg)
			d1313.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1305)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1311)
		ctx.EnsureDesc(&d1314)
		ctx.EnsureDescsTogether(&d1311, &d1314)
		var d1315 scm.JITValueDesc
		if d1311.Loc == scm.LocImm && d1314.Loc == scm.LocImm {
			d1315 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1311.Imm.Int()) >> uint64(d1314.Imm.Int())))}
		} else if d1314.Loc == scm.LocImm {
			r124 := ctx.AllocRegExcept(d1311.Reg)
			ctx.EmitMovRegReg(r124, d1311.Reg)
			ctx.EmitShrRegImm8(r124, uint8(d1314.Imm.Int()))
			d1315 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
			ctx.BindReg(r124, &d1315)
		} else {
			{
				shiftSrc := d1311.Reg
				r125 := ctx.AllocRegExcept(d1311.Reg, d1314.Reg)
				ctx.EmitMovRegReg(r125, d1311.Reg)
				shiftSrc = r125
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d1314.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d1314.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d1314.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d1315 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d1315)
			}
		}
		if d1315.Loc == scm.LocReg && d1311.Loc == scm.LocReg && d1315.Reg == d1311.Reg {
			ctx.TransferReg(d1311.Reg)
			d1311.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1311)
		ctx.FreeDesc(&d1314)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1309)
		ctx.EnsureDesc(&d1315)
		var d1316 scm.JITValueDesc
		if d1309.Loc == scm.LocImm && d1315.Loc == scm.LocImm {
			d1316 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1309.Imm.Int() | d1315.Imm.Int())}
		} else if d1309.Loc == scm.LocImm && d1309.Imm.Int() == 0 {
			d1316 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1315.Reg}
			ctx.BindReg(d1315.Reg, &d1316)
		} else if d1315.Loc == scm.LocImm && d1315.Imm.Int() == 0 {
			r126 := ctx.AllocRegExcept(d1309.Reg)
			ctx.EmitMovRegReg(r126, d1309.Reg)
			d1316 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r126}
			ctx.BindReg(r126, &d1316)
		} else if d1309.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1315.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1309.Imm.Int()))
			ctx.EmitOrInt64(scratch, d1315.Reg)
			d1316 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1316)
		} else if d1315.Loc == scm.LocImm {
			r127 := ctx.AllocRegExcept(d1309.Reg)
			ctx.EmitMovRegReg(r127, d1309.Reg)
			if d1315.Imm.Int() >= -2147483648 && d1315.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r127, int32(d1315.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1315.Imm.Int()))
				ctx.EmitOrInt64(r127, scm.RegR11)
			}
			d1316 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
			ctx.BindReg(r127, &d1316)
		} else {
			r128 := ctx.AllocRegExcept(d1309.Reg, d1315.Reg)
			ctx.EmitMovRegReg(r128, d1309.Reg)
			ctx.EmitOrInt64(r128, d1315.Reg)
			d1316 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
			ctx.BindReg(r128, &d1316)
		}
		if d1316.Loc == scm.LocReg && d1309.Loc == scm.LocReg && d1316.Reg == d1309.Reg {
			ctx.TransferReg(d1309.Reg)
			d1309.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1309)
		ctx.FreeDesc(&d1315)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d1317 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d1317 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 48)
			r129 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r129, thisptr.Reg, off)
			d1317 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r129}
			ctx.BindReg(r129, &d1317)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1317)
		ctx.EnsureDesc(&d1317)
		var d1318 scm.JITValueDesc
		if d1317.Loc == scm.LocImm {
			d1318 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d1317.Imm.Int()))))}
		} else {
			r130 := ctx.AllocReg()
			ctx.EmitMovRegReg(r130, d1317.Reg)
			ctx.EmitShlRegImm8(r130, 56)
			ctx.EmitShrRegImm8(r130, 56)
			d1318 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
			ctx.BindReg(r130, &d1318)
		}
		ctx.FreeDesc(&d1317)
		ctx.ReclaimUntrackedRegs()
		d1319 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d1318)
		ctx.EnsureDescsTogether(&d1319, &d1318)
		var d1320 scm.JITValueDesc
		if d1319.Loc == scm.LocImm && d1318.Loc == scm.LocImm {
			d1320 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1319.Imm.Int() - d1318.Imm.Int())}
		} else if d1318.Loc == scm.LocImm && d1318.Imm.Int() == 0 {
			r131 := ctx.AllocRegExcept(d1319.Reg)
			ctx.EmitMovRegReg(r131, d1319.Reg)
			d1320 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r131}
			ctx.BindReg(r131, &d1320)
		} else if d1319.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1318.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1319.Imm.Int()))
			ctx.EmitSubInt64(scratch, d1318.Reg)
			d1320 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1320)
		} else if d1318.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1319.Reg)
			ctx.EmitMovRegReg(scratch, d1319.Reg)
			if d1318.Imm.Int() >= -2147483648 && d1318.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d1318.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1318.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d1320 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1320)
		} else {
			r132 := ctx.AllocRegExcept(d1319.Reg, d1318.Reg)
			ctx.EmitMovRegReg(r132, d1319.Reg)
			ctx.EmitSubInt64(r132, d1318.Reg)
			d1320 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
			ctx.BindReg(r132, &d1320)
		}
		if d1320.Loc == scm.LocReg && d1319.Loc == scm.LocReg && d1320.Reg == d1319.Reg {
			ctx.TransferReg(d1319.Reg)
			d1319.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1318)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1316)
		ctx.EnsureDesc(&d1320)
		ctx.EnsureDescsTogether(&d1316, &d1320)
		var d1321 scm.JITValueDesc
		if d1316.Loc == scm.LocImm && d1320.Loc == scm.LocImm {
			d1321 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1316.Imm.Int()) >> uint64(d1320.Imm.Int())))}
		} else if d1320.Loc == scm.LocImm {
			r133 := ctx.AllocRegExcept(d1316.Reg)
			ctx.EmitMovRegReg(r133, d1316.Reg)
			ctx.EmitShrRegImm8(r133, uint8(d1320.Imm.Int()))
			d1321 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
			ctx.BindReg(r133, &d1321)
		} else {
			{
				shiftSrc := d1316.Reg
				r134 := ctx.AllocRegExcept(d1316.Reg, d1320.Reg)
				ctx.EmitMovRegReg(r134, d1316.Reg)
				shiftSrc = r134
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d1320.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d1320.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d1320.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d1321 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d1321)
			}
		}
		if d1321.Loc == scm.LocReg && d1316.Loc == scm.LocReg && d1321.Reg == d1316.Reg {
			ctx.TransferReg(d1316.Reg)
			d1316.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1316)
		ctx.FreeDesc(&d1320)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1321)
		ctx.EnsureDesc(&d1321)
		ctx.EnsureDesc(&d1321)
		var d1322 scm.JITValueDesc
		if d1321.Loc == scm.LocImm {
			d1322 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d1321.Imm.Int()))))}
		} else {
			r135 := ctx.AllocReg()
			ctx.EmitMovRegReg(r135, d1321.Reg)
			d1322 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r135}
			ctx.BindReg(r135, &d1322)
		}
		ctx.FreeDesc(&d1321)
		var d1323 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d1323 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 56)
			r136 := ctx.AllocReg()
			ctx.EmitMovRegMem(r136, thisptr.Reg, off)
			d1323 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r136}
			ctx.BindReg(r136, &d1323)
		}
		ctx.EnsureDesc(&d1322)
		ctx.EnsureDesc(&d1323)
		ctx.EnsureDescsTogether(&d1322, &d1323)
		var d1324 scm.JITValueDesc
		if d1322.Loc == scm.LocImm && d1323.Loc == scm.LocImm {
			d1324 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1322.Imm.Int() + d1323.Imm.Int())}
		} else if d1323.Loc == scm.LocImm && d1323.Imm.Int() == 0 {
			r137 := ctx.AllocRegExcept(d1322.Reg)
			ctx.EmitMovRegReg(r137, d1322.Reg)
			d1324 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r137}
			ctx.BindReg(r137, &d1324)
		} else if d1322.Loc == scm.LocImm && d1322.Imm.Int() == 0 {
			d1324 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1323.Reg}
			ctx.BindReg(d1323.Reg, &d1324)
		} else if d1322.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1323.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1322.Imm.Int()))
			ctx.EmitAddInt64(scratch, d1323.Reg)
			d1324 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1324)
		} else if d1323.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1322.Reg)
			ctx.EmitMovRegReg(scratch, d1322.Reg)
			if d1323.Imm.Int() >= -2147483648 && d1323.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d1323.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1323.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d1324 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1324)
		} else {
			r138 := ctx.AllocRegExcept(d1322.Reg, d1323.Reg)
			ctx.EmitMovRegReg(r138, d1322.Reg)
			ctx.EmitAddInt64(r138, d1323.Reg)
			d1324 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
			ctx.BindReg(r138, &d1324)
		}
		if d1324.Loc == scm.LocReg && d1322.Loc == scm.LocReg && d1324.Reg == d1322.Reg {
			ctx.TransferReg(d1322.Reg)
			d1322.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1322)
		ctx.FreeDesc(&d1323)
		ctx.EnsureDesc(&d8)
		d1325 = d8
		_ = d1325
		ctx.StabilizeDescForControlFlow(&d1325)
		ctx.StabilizeDescForControlFlow(&d8)
		bbpos_5_0 := int32(-1)
		_ = bbpos_5_0
		lbl29 := ctx.ReserveLabel()
		_ = lbl29
		bbpos_5_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl29)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1325)
		ctx.EnsureDesc(&d1325)
		var d1326 scm.JITValueDesc
		if d1325.Loc == scm.LocImm {
			d1326 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d1325.Imm.Int()))))}
		} else {
			r139 := ctx.AllocReg()
			ctx.EmitMovRegReg(r139, d1325.Reg)
			ctx.EmitShlRegImm8(r139, 32)
			ctx.EmitShrRegImm8(r139, 32)
			d1326 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r139}
			ctx.BindReg(r139, &d1326)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d1327 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d1327 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48)
			r140 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r140, thisptr.Reg, off)
			d1327 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r140}
			ctx.BindReg(r140, &d1327)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1327)
		ctx.EnsureDesc(&d1327)
		var d1328 scm.JITValueDesc
		if d1327.Loc == scm.LocImm {
			d1328 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d1327.Imm.Int()))))}
		} else {
			r141 := ctx.AllocReg()
			ctx.EmitMovRegReg(r141, d1327.Reg)
			ctx.EmitShlRegImm8(r141, 56)
			ctx.EmitShrRegImm8(r141, 56)
			d1328 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
			ctx.BindReg(r141, &d1328)
		}
		ctx.FreeDesc(&d1327)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1326)
		ctx.EnsureDesc(&d1328)
		ctx.EnsureDescsTogether(&d1326, &d1328)
		var d1329 scm.JITValueDesc
		if d1326.Loc == scm.LocImm && d1328.Loc == scm.LocImm {
			d1329 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1326.Imm.Int() * d1328.Imm.Int())}
		} else if d1326.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1328.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1326.Imm.Int()))
			ctx.EmitImulInt64(scratch, d1328.Reg)
			d1329 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1329)
		} else if d1328.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1326.Reg)
			ctx.EmitMovRegReg(scratch, d1326.Reg)
			if d1328.Imm.Int() >= -2147483648 && d1328.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d1328.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1328.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d1329 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1329)
		} else {
			r142 := ctx.AllocRegExcept(d1326.Reg, d1328.Reg)
			ctx.EmitMovRegReg(r142, d1326.Reg)
			ctx.EmitImulInt64(r142, d1328.Reg)
			d1329 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r142}
			ctx.BindReg(r142, &d1329)
		}
		if d1329.Loc == scm.LocReg && d1326.Loc == scm.LocReg && d1329.Reg == d1326.Reg {
			ctx.TransferReg(d1326.Reg)
			d1326.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1326)
		ctx.FreeDesc(&d1328)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1329)
		var d1330 scm.JITValueDesc
		if d1329.Loc == scm.LocImm {
			d1330 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1329.Imm.Int() / 64)}
		} else {
			r143 := ctx.AllocRegExcept(d1329.Reg)
			ctx.EmitMovRegReg(r143, d1329.Reg)
			ctx.EmitShrRegImm8(r143, 6)
			d1330 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r143}
			ctx.BindReg(r143, &d1330)
		}
		if d1330.Loc == scm.LocReg && d1329.Loc == scm.LocReg && d1330.Reg == d1329.Reg {
			ctx.TransferReg(d1329.Reg)
			d1329.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1329)
		var d1331 scm.JITValueDesc
		if d1329.Loc == scm.LocImm {
			d1331 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1329.Imm.Int() % 64)}
		} else {
			r144 := ctx.AllocRegExcept(d1329.Reg)
			ctx.EmitMovRegReg(r144, d1329.Reg)
			ctx.EmitAndRegImm32(r144, 63)
			d1331 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
			ctx.BindReg(r144, &d1331)
		}
		if d1331.Loc == scm.LocReg && d1329.Loc == scm.LocReg && d1331.Reg == d1329.Reg {
			ctx.TransferReg(d1329.Reg)
			d1329.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1329)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d1332 scm.JITValueDesc
		r145 := ctx.AllocReg()
		r146 := ctx.AllocRegExcept(r145)
		r147 := ctx.AllocRegExcept(r145, r146)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			ctx.EmitMovRegImm64(r145, uint64(dataPtr))
			ctx.EmitMovRegImm64(r146, uint64(sliceLen))
			ctx.EmitMovRegImm64(r147, uint64(sliceCap))
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			ctx.EmitMovRegMem(r145, thisptr.Reg, off)
			ctx.EmitMovRegMem(r146, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r147, thisptr.Reg, off+16)
		}
		d1332 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r145, Reg2: r146, Reg3: r147}
		ctx.BindReg(r145, &d1332)
		ctx.BindReg(r146, &d1332)
		ctx.BindReg(r147, &d1332)
		ctx.BindReg(r145, &d1332)
		ctx.BindReg(r146, &d1332)
		ctx.BindReg(r147, &d1332)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1330)
		ctx.ReclaimUntrackedRegs()
		d1334 = ctx.EmitSliceElementAddress(&d1332, &d1330, 8)
		ctx.EnsureDesc(&d1334)
		ctx.EmitMovRegMem(d1334.Reg, d1334.Reg, 0)
		d1333 = d1334
		d1333.Type = scm.TagInt
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1333)
		ctx.EnsureDesc(&d1331)
		ctx.EnsureDescsTogether(&d1333, &d1331)
		var d1335 scm.JITValueDesc
		if d1333.Loc == scm.LocImm && d1331.Loc == scm.LocImm {
			d1335 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1333.Imm.Int()) << uint64(d1331.Imm.Int())))}
		} else if d1331.Loc == scm.LocImm {
			r148 := ctx.AllocRegExcept(d1333.Reg)
			ctx.EmitMovRegReg(r148, d1333.Reg)
			ctx.EmitShlRegImm8(r148, uint8(d1331.Imm.Int()))
			d1335 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r148}
			ctx.BindReg(r148, &d1335)
		} else {
			{
				shiftSrc := d1333.Reg
				r149 := ctx.AllocRegExcept(d1333.Reg, d1331.Reg)
				ctx.EmitMovRegReg(r149, d1333.Reg)
				shiftSrc = r149
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d1331.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d1331.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d1331.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d1335 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d1335)
			}
		}
		if d1335.Loc == scm.LocReg && d1333.Loc == scm.LocReg && d1335.Reg == d1333.Reg {
			ctx.TransferReg(d1333.Reg)
			d1333.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1333)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1330)
		ctx.EnsureDesc(&d1330)
		var d1336 scm.JITValueDesc
		if d1330.Loc == scm.LocImm {
			d1336 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1330.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d1330.Reg)
			ctx.EmitMovRegReg(scratch, d1330.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d1336 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1336)
		}
		if d1336.Loc == scm.LocReg && d1330.Loc == scm.LocReg && d1336.Reg == d1330.Reg {
			ctx.TransferReg(d1330.Reg)
			d1330.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1330)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1336)
		ctx.ReclaimUntrackedRegs()
		d1338 = ctx.EmitSliceElementAddress(&d1332, &d1336, 8)
		ctx.EnsureDesc(&d1338)
		ctx.EmitMovRegMem(d1338.Reg, d1338.Reg, 0)
		d1337 = d1338
		d1337.Type = scm.TagInt
		ctx.FreeDesc(&d1336)
		ctx.ReclaimUntrackedRegs()
		d1339 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d1331)
		ctx.EnsureDescsTogether(&d1339, &d1331)
		var d1340 scm.JITValueDesc
		if d1339.Loc == scm.LocImm && d1331.Loc == scm.LocImm {
			d1340 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1339.Imm.Int() - d1331.Imm.Int())}
		} else if d1331.Loc == scm.LocImm && d1331.Imm.Int() == 0 {
			r150 := ctx.AllocRegExcept(d1339.Reg)
			ctx.EmitMovRegReg(r150, d1339.Reg)
			d1340 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r150}
			ctx.BindReg(r150, &d1340)
		} else if d1339.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1331.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1339.Imm.Int()))
			ctx.EmitSubInt64(scratch, d1331.Reg)
			d1340 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1340)
		} else if d1331.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1339.Reg)
			ctx.EmitMovRegReg(scratch, d1339.Reg)
			if d1331.Imm.Int() >= -2147483648 && d1331.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d1331.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1331.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d1340 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1340)
		} else {
			r151 := ctx.AllocRegExcept(d1339.Reg, d1331.Reg)
			ctx.EmitMovRegReg(r151, d1339.Reg)
			ctx.EmitSubInt64(r151, d1331.Reg)
			d1340 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
			ctx.BindReg(r151, &d1340)
		}
		if d1340.Loc == scm.LocReg && d1339.Loc == scm.LocReg && d1340.Reg == d1339.Reg {
			ctx.TransferReg(d1339.Reg)
			d1339.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1331)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1337)
		ctx.EnsureDesc(&d1340)
		ctx.EnsureDescsTogether(&d1337, &d1340)
		var d1341 scm.JITValueDesc
		if d1337.Loc == scm.LocImm && d1340.Loc == scm.LocImm {
			d1341 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1337.Imm.Int()) >> uint64(d1340.Imm.Int())))}
		} else if d1340.Loc == scm.LocImm {
			r152 := ctx.AllocRegExcept(d1337.Reg)
			ctx.EmitMovRegReg(r152, d1337.Reg)
			ctx.EmitShrRegImm8(r152, uint8(d1340.Imm.Int()))
			d1341 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r152}
			ctx.BindReg(r152, &d1341)
		} else {
			{
				shiftSrc := d1337.Reg
				r153 := ctx.AllocRegExcept(d1337.Reg, d1340.Reg)
				ctx.EmitMovRegReg(r153, d1337.Reg)
				shiftSrc = r153
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d1340.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d1340.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d1340.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d1341 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d1341)
			}
		}
		if d1341.Loc == scm.LocReg && d1337.Loc == scm.LocReg && d1341.Reg == d1337.Reg {
			ctx.TransferReg(d1337.Reg)
			d1337.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1337)
		ctx.FreeDesc(&d1340)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1335)
		ctx.EnsureDesc(&d1341)
		var d1342 scm.JITValueDesc
		if d1335.Loc == scm.LocImm && d1341.Loc == scm.LocImm {
			d1342 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1335.Imm.Int() | d1341.Imm.Int())}
		} else if d1335.Loc == scm.LocImm && d1335.Imm.Int() == 0 {
			d1342 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1341.Reg}
			ctx.BindReg(d1341.Reg, &d1342)
		} else if d1341.Loc == scm.LocImm && d1341.Imm.Int() == 0 {
			r154 := ctx.AllocRegExcept(d1335.Reg)
			ctx.EmitMovRegReg(r154, d1335.Reg)
			d1342 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r154}
			ctx.BindReg(r154, &d1342)
		} else if d1335.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1341.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1335.Imm.Int()))
			ctx.EmitOrInt64(scratch, d1341.Reg)
			d1342 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1342)
		} else if d1341.Loc == scm.LocImm {
			r155 := ctx.AllocRegExcept(d1335.Reg)
			ctx.EmitMovRegReg(r155, d1335.Reg)
			if d1341.Imm.Int() >= -2147483648 && d1341.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r155, int32(d1341.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1341.Imm.Int()))
				ctx.EmitOrInt64(r155, scm.RegR11)
			}
			d1342 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r155}
			ctx.BindReg(r155, &d1342)
		} else {
			r156 := ctx.AllocRegExcept(d1335.Reg, d1341.Reg)
			ctx.EmitMovRegReg(r156, d1335.Reg)
			ctx.EmitOrInt64(r156, d1341.Reg)
			d1342 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r156}
			ctx.BindReg(r156, &d1342)
		}
		if d1342.Loc == scm.LocReg && d1335.Loc == scm.LocReg && d1342.Reg == d1335.Reg {
			ctx.TransferReg(d1335.Reg)
			d1335.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1335)
		ctx.FreeDesc(&d1341)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d1343 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d1343 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48)
			r157 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r157, thisptr.Reg, off)
			d1343 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r157}
			ctx.BindReg(r157, &d1343)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1343)
		ctx.EnsureDesc(&d1343)
		var d1344 scm.JITValueDesc
		if d1343.Loc == scm.LocImm {
			d1344 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d1343.Imm.Int()))))}
		} else {
			r158 := ctx.AllocReg()
			ctx.EmitMovRegReg(r158, d1343.Reg)
			ctx.EmitShlRegImm8(r158, 56)
			ctx.EmitShrRegImm8(r158, 56)
			d1344 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r158}
			ctx.BindReg(r158, &d1344)
		}
		ctx.FreeDesc(&d1343)
		ctx.ReclaimUntrackedRegs()
		d1345 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d1344)
		ctx.EnsureDescsTogether(&d1345, &d1344)
		var d1346 scm.JITValueDesc
		if d1345.Loc == scm.LocImm && d1344.Loc == scm.LocImm {
			d1346 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1345.Imm.Int() - d1344.Imm.Int())}
		} else if d1344.Loc == scm.LocImm && d1344.Imm.Int() == 0 {
			r159 := ctx.AllocRegExcept(d1345.Reg)
			ctx.EmitMovRegReg(r159, d1345.Reg)
			d1346 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
			ctx.BindReg(r159, &d1346)
		} else if d1345.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1344.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1345.Imm.Int()))
			ctx.EmitSubInt64(scratch, d1344.Reg)
			d1346 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1346)
		} else if d1344.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1345.Reg)
			ctx.EmitMovRegReg(scratch, d1345.Reg)
			if d1344.Imm.Int() >= -2147483648 && d1344.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d1344.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1344.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d1346 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1346)
		} else {
			r160 := ctx.AllocRegExcept(d1345.Reg, d1344.Reg)
			ctx.EmitMovRegReg(r160, d1345.Reg)
			ctx.EmitSubInt64(r160, d1344.Reg)
			d1346 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r160}
			ctx.BindReg(r160, &d1346)
		}
		if d1346.Loc == scm.LocReg && d1345.Loc == scm.LocReg && d1346.Reg == d1345.Reg {
			ctx.TransferReg(d1345.Reg)
			d1345.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1344)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1342)
		ctx.EnsureDesc(&d1346)
		ctx.EnsureDescsTogether(&d1342, &d1346)
		var d1347 scm.JITValueDesc
		if d1342.Loc == scm.LocImm && d1346.Loc == scm.LocImm {
			d1347 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1342.Imm.Int()) >> uint64(d1346.Imm.Int())))}
		} else if d1346.Loc == scm.LocImm {
			r161 := ctx.AllocRegExcept(d1342.Reg)
			ctx.EmitMovRegReg(r161, d1342.Reg)
			ctx.EmitShrRegImm8(r161, uint8(d1346.Imm.Int()))
			d1347 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r161}
			ctx.BindReg(r161, &d1347)
		} else {
			{
				shiftSrc := d1342.Reg
				r162 := ctx.AllocRegExcept(d1342.Reg, d1346.Reg)
				ctx.EmitMovRegReg(r162, d1342.Reg)
				shiftSrc = r162
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d1346.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d1346.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d1346.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d1347 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d1347)
			}
		}
		if d1347.Loc == scm.LocReg && d1342.Loc == scm.LocReg && d1347.Reg == d1342.Reg {
			ctx.TransferReg(d1342.Reg)
			d1342.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1342)
		ctx.FreeDesc(&d1346)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1347)
		ctx.EnsureDesc(&d1347)
		ctx.EnsureDesc(&d1347)
		var d1348 scm.JITValueDesc
		if d1347.Loc == scm.LocImm {
			d1348 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d1347.Imm.Int()))))}
		} else {
			r163 := ctx.AllocReg()
			ctx.EmitMovRegReg(r163, d1347.Reg)
			d1348 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r163}
			ctx.BindReg(r163, &d1348)
		}
		ctx.FreeDesc(&d1347)
		var d1349 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d1349 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56)
			r164 := ctx.AllocReg()
			ctx.EmitMovRegMem(r164, thisptr.Reg, off)
			d1349 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r164}
			ctx.BindReg(r164, &d1349)
		}
		ctx.EnsureDesc(&d1348)
		ctx.EnsureDesc(&d1349)
		ctx.EnsureDescsTogether(&d1348, &d1349)
		var d1350 scm.JITValueDesc
		if d1348.Loc == scm.LocImm && d1349.Loc == scm.LocImm {
			d1350 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1348.Imm.Int() + d1349.Imm.Int())}
		} else if d1349.Loc == scm.LocImm && d1349.Imm.Int() == 0 {
			r165 := ctx.AllocRegExcept(d1348.Reg)
			ctx.EmitMovRegReg(r165, d1348.Reg)
			d1350 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r165}
			ctx.BindReg(r165, &d1350)
		} else if d1348.Loc == scm.LocImm && d1348.Imm.Int() == 0 {
			d1350 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1349.Reg}
			ctx.BindReg(d1349.Reg, &d1350)
		} else if d1348.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1349.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1348.Imm.Int()))
			ctx.EmitAddInt64(scratch, d1349.Reg)
			d1350 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1350)
		} else if d1349.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1348.Reg)
			ctx.EmitMovRegReg(scratch, d1348.Reg)
			if d1349.Imm.Int() >= -2147483648 && d1349.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d1349.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1349.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d1350 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1350)
		} else {
			r166 := ctx.AllocRegExcept(d1348.Reg, d1349.Reg)
			ctx.EmitMovRegReg(r166, d1348.Reg)
			ctx.EmitAddInt64(r166, d1349.Reg)
			d1350 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r166}
			ctx.BindReg(r166, &d1350)
		}
		if d1350.Loc == scm.LocReg && d1348.Loc == scm.LocReg && d1350.Reg == d1348.Reg {
			ctx.TransferReg(d1348.Reg)
			d1348.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1348)
		ctx.FreeDesc(&d1349)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		var d1351 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm {
			d1351 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(idxInt.Imm.Int()))))}
		} else {
			r167 := ctx.AllocReg()
			ctx.EmitMovRegReg(r167, idxInt.Reg)
			ctx.EmitShlRegImm8(r167, 32)
			ctx.EmitShrRegImm8(r167, 32)
			d1351 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r167}
			ctx.BindReg(r167, &d1351)
		}
		ctx.FreeDesc(&idxInt)
		ctx.EnsureDesc(&d1351)
		ctx.EnsureDesc(&d1350)
		ctx.EnsureDescsTogether(&d1351, &d1350)
		var d1352 scm.JITValueDesc
		if d1351.Loc == scm.LocImm && d1350.Loc == scm.LocImm {
			d1352 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1351.Imm.Int() - d1350.Imm.Int())}
		} else if d1350.Loc == scm.LocImm && d1350.Imm.Int() == 0 {
			r168 := ctx.AllocRegExcept(d1351.Reg)
			ctx.EmitMovRegReg(r168, d1351.Reg)
			d1352 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r168}
			ctx.BindReg(r168, &d1352)
		} else if d1351.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1350.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1351.Imm.Int()))
			ctx.EmitSubInt64(scratch, d1350.Reg)
			d1352 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1352)
		} else if d1350.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1351.Reg)
			ctx.EmitMovRegReg(scratch, d1351.Reg)
			if d1350.Imm.Int() >= -2147483648 && d1350.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d1350.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1350.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d1352 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1352)
		} else {
			r169 := ctx.AllocRegExcept(d1351.Reg, d1350.Reg)
			ctx.EmitMovRegReg(r169, d1351.Reg)
			ctx.EmitSubInt64(r169, d1350.Reg)
			d1352 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r169}
			ctx.BindReg(r169, &d1352)
		}
		if d1352.Loc == scm.LocReg && d1351.Loc == scm.LocReg && d1352.Reg == d1351.Reg {
			ctx.TransferReg(d1351.Reg)
			d1351.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1351)
		ctx.FreeDesc(&d1350)
		ctx.EnsureDesc(&d1352)
		ctx.EnsureDesc(&d1324)
		ctx.EnsureDescsTogether(&d1352, &d1324)
		var d1353 scm.JITValueDesc
		if d1352.Loc == scm.LocImm && d1324.Loc == scm.LocImm {
			d1353 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1352.Imm.Int() * d1324.Imm.Int())}
		} else if d1352.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1324.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1352.Imm.Int()))
			ctx.EmitImulInt64(scratch, d1324.Reg)
			d1353 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1353)
		} else if d1324.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1352.Reg)
			ctx.EmitMovRegReg(scratch, d1352.Reg)
			if d1324.Imm.Int() >= -2147483648 && d1324.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d1324.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1324.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d1353 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1353)
		} else {
			r170 := ctx.AllocRegExcept(d1352.Reg, d1324.Reg)
			ctx.EmitMovRegReg(r170, d1352.Reg)
			ctx.EmitImulInt64(r170, d1324.Reg)
			d1353 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r170}
			ctx.BindReg(r170, &d1353)
		}
		if d1353.Loc == scm.LocReg && d1352.Loc == scm.LocReg && d1353.Reg == d1352.Reg {
			ctx.TransferReg(d1352.Reg)
			d1352.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1352)
		ctx.FreeDesc(&d1324)
		ctx.EnsureDesc(&d204)
		ctx.EnsureDesc(&d1353)
		ctx.EnsureDescsTogether(&d204, &d1353)
		var d1354 scm.JITValueDesc
		if d204.Loc == scm.LocImm && d1353.Loc == scm.LocImm {
			d1354 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d204.Imm.Int() + d1353.Imm.Int())}
		} else if d1353.Loc == scm.LocImm && d1353.Imm.Int() == 0 {
			r171 := ctx.AllocRegExcept(d204.Reg)
			ctx.EmitMovRegReg(r171, d204.Reg)
			d1354 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r171}
			ctx.BindReg(r171, &d1354)
		} else if d204.Loc == scm.LocImm && d204.Imm.Int() == 0 {
			d1354 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1353.Reg}
			ctx.BindReg(d1353.Reg, &d1354)
		} else if d204.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1353.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d204.Imm.Int()))
			ctx.EmitAddInt64(scratch, d1353.Reg)
			d1354 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1354)
		} else if d1353.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d204.Reg)
			ctx.EmitMovRegReg(scratch, d204.Reg)
			if d1353.Imm.Int() >= -2147483648 && d1353.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d1353.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1353.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d1354 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1354)
		} else {
			r172 := ctx.AllocRegExcept(d204.Reg, d1353.Reg)
			ctx.EmitMovRegReg(r172, d204.Reg)
			ctx.EmitAddInt64(r172, d1353.Reg)
			d1354 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
			ctx.BindReg(r172, &d1354)
		}
		if d1354.Loc == scm.LocReg && d204.Loc == scm.LocReg && d1354.Reg == d204.Reg {
			ctx.TransferReg(d204.Reg)
			d204.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1353)
		ctx.EnsureDesc(&d1354)
		ctx.EnsureDesc(&d1354)
		var d1355 scm.JITValueDesc
		if d1354.Loc == scm.LocImm {
			d1355 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d1354.Imm.Int()))}
		} else {
			r173 := ctx.AllocRegExcept(d1354.Reg)
			ctx.EmitMovRegReg(r173, d1354.Reg)
			ctx.EmitCvtInt64ToFloat64(scm.RegX0, r173)
			d1355 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: r173}
			ctx.BindReg(r173, &d1355)
		}
		ctx.FreeDesc(&d1354)
		ctx.EnsureDesc(&d1355)
		d1356 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r3, Reg2: r4}
		ctx.BindReg(r3, &d1356)
		ctx.BindReg(r4, &d1356)
		ctx.EnsureDesc(&d1355)
		ctx.EmitMakeFloat(d1356, d1355)
		if d1355.Loc == scm.LocReg {
			ctx.FreeReg(d1355.Reg)
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
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
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
		if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != scm.LocNone {
			d177 = ps.OverlayValues[177]
		}
		if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
			d178 = ps.OverlayValues[178]
		}
		if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != scm.LocNone {
			d179 = ps.OverlayValues[179]
		}
		if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != scm.LocNone {
			d180 = ps.OverlayValues[180]
		}
		if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != scm.LocNone {
			d181 = ps.OverlayValues[181]
		}
		if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != scm.LocNone {
			d182 = ps.OverlayValues[182]
		}
		if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != scm.LocNone {
			d183 = ps.OverlayValues[183]
		}
		if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
			d184 = ps.OverlayValues[184]
		}
		if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != scm.LocNone {
			d185 = ps.OverlayValues[185]
		}
		if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != scm.LocNone {
			d186 = ps.OverlayValues[186]
		}
		if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
			d187 = ps.OverlayValues[187]
		}
		if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != scm.LocNone {
			d188 = ps.OverlayValues[188]
		}
		if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != scm.LocNone {
			d189 = ps.OverlayValues[189]
		}
		if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != scm.LocNone {
			d190 = ps.OverlayValues[190]
		}
		if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != scm.LocNone {
			d191 = ps.OverlayValues[191]
		}
		if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != scm.LocNone {
			d192 = ps.OverlayValues[192]
		}
		if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != scm.LocNone {
			d193 = ps.OverlayValues[193]
		}
		if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != scm.LocNone {
			d194 = ps.OverlayValues[194]
		}
		if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != scm.LocNone {
			d195 = ps.OverlayValues[195]
		}
		if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
			d196 = ps.OverlayValues[196]
		}
		if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != scm.LocNone {
			d197 = ps.OverlayValues[197]
		}
		if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != scm.LocNone {
			d198 = ps.OverlayValues[198]
		}
		if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
			d199 = ps.OverlayValues[199]
		}
		if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
			d200 = ps.OverlayValues[200]
		}
		if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
			d201 = ps.OverlayValues[201]
		}
		if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != scm.LocNone {
			d202 = ps.OverlayValues[202]
		}
		if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != scm.LocNone {
			d203 = ps.OverlayValues[203]
		}
		if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != scm.LocNone {
			d204 = ps.OverlayValues[204]
		}
		if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != scm.LocNone {
			d205 = ps.OverlayValues[205]
		}
		if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != scm.LocNone {
			d206 = ps.OverlayValues[206]
		}
		if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != scm.LocNone {
			d209 = ps.OverlayValues[209]
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
		if len(ps.OverlayValues) > 400 && ps.OverlayValues[400].Loc != scm.LocNone {
			d400 = ps.OverlayValues[400]
		}
		if len(ps.OverlayValues) > 402 && ps.OverlayValues[402].Loc != scm.LocNone {
			d402 = ps.OverlayValues[402]
		}
		if len(ps.OverlayValues) > 403 && ps.OverlayValues[403].Loc != scm.LocNone {
			d403 = ps.OverlayValues[403]
		}
		if len(ps.OverlayValues) > 404 && ps.OverlayValues[404].Loc != scm.LocNone {
			d404 = ps.OverlayValues[404]
		}
		if len(ps.OverlayValues) > 508 && ps.OverlayValues[508].Loc != scm.LocNone {
			d508 = ps.OverlayValues[508]
		}
		if len(ps.OverlayValues) > 509 && ps.OverlayValues[509].Loc != scm.LocNone {
			d509 = ps.OverlayValues[509]
		}
		if len(ps.OverlayValues) > 512 && ps.OverlayValues[512].Loc != scm.LocNone {
			d512 = ps.OverlayValues[512]
		}
		if len(ps.OverlayValues) > 619 && ps.OverlayValues[619].Loc != scm.LocNone {
			d619 = ps.OverlayValues[619]
		}
		if len(ps.OverlayValues) > 620 && ps.OverlayValues[620].Loc != scm.LocNone {
			d620 = ps.OverlayValues[620]
		}
		if len(ps.OverlayValues) > 621 && ps.OverlayValues[621].Loc != scm.LocNone {
			d621 = ps.OverlayValues[621]
		}
		if len(ps.OverlayValues) > 622 && ps.OverlayValues[622].Loc != scm.LocNone {
			d622 = ps.OverlayValues[622]
		}
		if len(ps.OverlayValues) > 623 && ps.OverlayValues[623].Loc != scm.LocNone {
			d623 = ps.OverlayValues[623]
		}
		if len(ps.OverlayValues) > 625 && ps.OverlayValues[625].Loc != scm.LocNone {
			d625 = ps.OverlayValues[625]
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
		if len(ps.OverlayValues) > 629 && ps.OverlayValues[629].Loc != scm.LocNone {
			d629 = ps.OverlayValues[629]
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
		if len(ps.OverlayValues) > 637 && ps.OverlayValues[637].Loc != scm.LocNone {
			d637 = ps.OverlayValues[637]
		}
		if len(ps.OverlayValues) > 638 && ps.OverlayValues[638].Loc != scm.LocNone {
			d638 = ps.OverlayValues[638]
		}
		if len(ps.OverlayValues) > 639 && ps.OverlayValues[639].Loc != scm.LocNone {
			d639 = ps.OverlayValues[639]
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
		if len(ps.OverlayValues) > 644 && ps.OverlayValues[644].Loc != scm.LocNone {
			d644 = ps.OverlayValues[644]
		}
		if len(ps.OverlayValues) > 645 && ps.OverlayValues[645].Loc != scm.LocNone {
			d645 = ps.OverlayValues[645]
		}
		if len(ps.OverlayValues) > 646 && ps.OverlayValues[646].Loc != scm.LocNone {
			d646 = ps.OverlayValues[646]
		}
		if len(ps.OverlayValues) > 647 && ps.OverlayValues[647].Loc != scm.LocNone {
			d647 = ps.OverlayValues[647]
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
		if len(ps.OverlayValues) > 651 && ps.OverlayValues[651].Loc != scm.LocNone {
			d651 = ps.OverlayValues[651]
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
		if len(ps.OverlayValues) > 944 && ps.OverlayValues[944].Loc != scm.LocNone {
			d944 = ps.OverlayValues[944]
		}
		if len(ps.OverlayValues) > 945 && ps.OverlayValues[945].Loc != scm.LocNone {
			d945 = ps.OverlayValues[945]
		}
		if len(ps.OverlayValues) > 946 && ps.OverlayValues[946].Loc != scm.LocNone {
			d946 = ps.OverlayValues[946]
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
		if len(ps.OverlayValues) > 956 && ps.OverlayValues[956].Loc != scm.LocNone {
			d956 = ps.OverlayValues[956]
		}
		if len(ps.OverlayValues) > 958 && ps.OverlayValues[958].Loc != scm.LocNone {
			d958 = ps.OverlayValues[958]
		}
		if len(ps.OverlayValues) > 959 && ps.OverlayValues[959].Loc != scm.LocNone {
			d959 = ps.OverlayValues[959]
		}
		if len(ps.OverlayValues) > 1115 && ps.OverlayValues[1115].Loc != scm.LocNone {
			d1115 = ps.OverlayValues[1115]
		}
		if len(ps.OverlayValues) > 1116 && ps.OverlayValues[1116].Loc != scm.LocNone {
			d1116 = ps.OverlayValues[1116]
		}
		if len(ps.OverlayValues) > 1119 && ps.OverlayValues[1119].Loc != scm.LocNone {
			d1119 = ps.OverlayValues[1119]
		}
		if len(ps.OverlayValues) > 1278 && ps.OverlayValues[1278].Loc != scm.LocNone {
			d1278 = ps.OverlayValues[1278]
		}
		if len(ps.OverlayValues) > 1279 && ps.OverlayValues[1279].Loc != scm.LocNone {
			d1279 = ps.OverlayValues[1279]
		}
		if len(ps.OverlayValues) > 1280 && ps.OverlayValues[1280].Loc != scm.LocNone {
			d1280 = ps.OverlayValues[1280]
		}
		if len(ps.OverlayValues) > 1281 && ps.OverlayValues[1281].Loc != scm.LocNone {
			d1281 = ps.OverlayValues[1281]
		}
		if len(ps.OverlayValues) > 1283 && ps.OverlayValues[1283].Loc != scm.LocNone {
			d1283 = ps.OverlayValues[1283]
		}
		if len(ps.OverlayValues) > 1284 && ps.OverlayValues[1284].Loc != scm.LocNone {
			d1284 = ps.OverlayValues[1284]
		}
		if len(ps.OverlayValues) > 1285 && ps.OverlayValues[1285].Loc != scm.LocNone {
			d1285 = ps.OverlayValues[1285]
		}
		if len(ps.OverlayValues) > 1286 && ps.OverlayValues[1286].Loc != scm.LocNone {
			d1286 = ps.OverlayValues[1286]
		}
		if len(ps.OverlayValues) > 1287 && ps.OverlayValues[1287].Loc != scm.LocNone {
			d1287 = ps.OverlayValues[1287]
		}
		if len(ps.OverlayValues) > 1288 && ps.OverlayValues[1288].Loc != scm.LocNone {
			d1288 = ps.OverlayValues[1288]
		}
		if len(ps.OverlayValues) > 1289 && ps.OverlayValues[1289].Loc != scm.LocNone {
			d1289 = ps.OverlayValues[1289]
		}
		if len(ps.OverlayValues) > 1290 && ps.OverlayValues[1290].Loc != scm.LocNone {
			d1290 = ps.OverlayValues[1290]
		}
		if len(ps.OverlayValues) > 1291 && ps.OverlayValues[1291].Loc != scm.LocNone {
			d1291 = ps.OverlayValues[1291]
		}
		if len(ps.OverlayValues) > 1292 && ps.OverlayValues[1292].Loc != scm.LocNone {
			d1292 = ps.OverlayValues[1292]
		}
		if len(ps.OverlayValues) > 1294 && ps.OverlayValues[1294].Loc != scm.LocNone {
			d1294 = ps.OverlayValues[1294]
		}
		if len(ps.OverlayValues) > 1295 && ps.OverlayValues[1295].Loc != scm.LocNone {
			d1295 = ps.OverlayValues[1295]
		}
		if len(ps.OverlayValues) > 1296 && ps.OverlayValues[1296].Loc != scm.LocNone {
			d1296 = ps.OverlayValues[1296]
		}
		if len(ps.OverlayValues) > 1297 && ps.OverlayValues[1297].Loc != scm.LocNone {
			d1297 = ps.OverlayValues[1297]
		}
		if len(ps.OverlayValues) > 1298 && ps.OverlayValues[1298].Loc != scm.LocNone {
			d1298 = ps.OverlayValues[1298]
		}
		if len(ps.OverlayValues) > 1299 && ps.OverlayValues[1299].Loc != scm.LocNone {
			d1299 = ps.OverlayValues[1299]
		}
		if len(ps.OverlayValues) > 1300 && ps.OverlayValues[1300].Loc != scm.LocNone {
			d1300 = ps.OverlayValues[1300]
		}
		if len(ps.OverlayValues) > 1301 && ps.OverlayValues[1301].Loc != scm.LocNone {
			d1301 = ps.OverlayValues[1301]
		}
		if len(ps.OverlayValues) > 1302 && ps.OverlayValues[1302].Loc != scm.LocNone {
			d1302 = ps.OverlayValues[1302]
		}
		if len(ps.OverlayValues) > 1303 && ps.OverlayValues[1303].Loc != scm.LocNone {
			d1303 = ps.OverlayValues[1303]
		}
		if len(ps.OverlayValues) > 1304 && ps.OverlayValues[1304].Loc != scm.LocNone {
			d1304 = ps.OverlayValues[1304]
		}
		if len(ps.OverlayValues) > 1305 && ps.OverlayValues[1305].Loc != scm.LocNone {
			d1305 = ps.OverlayValues[1305]
		}
		if len(ps.OverlayValues) > 1306 && ps.OverlayValues[1306].Loc != scm.LocNone {
			d1306 = ps.OverlayValues[1306]
		}
		if len(ps.OverlayValues) > 1307 && ps.OverlayValues[1307].Loc != scm.LocNone {
			d1307 = ps.OverlayValues[1307]
		}
		if len(ps.OverlayValues) > 1308 && ps.OverlayValues[1308].Loc != scm.LocNone {
			d1308 = ps.OverlayValues[1308]
		}
		if len(ps.OverlayValues) > 1309 && ps.OverlayValues[1309].Loc != scm.LocNone {
			d1309 = ps.OverlayValues[1309]
		}
		if len(ps.OverlayValues) > 1310 && ps.OverlayValues[1310].Loc != scm.LocNone {
			d1310 = ps.OverlayValues[1310]
		}
		if len(ps.OverlayValues) > 1311 && ps.OverlayValues[1311].Loc != scm.LocNone {
			d1311 = ps.OverlayValues[1311]
		}
		if len(ps.OverlayValues) > 1312 && ps.OverlayValues[1312].Loc != scm.LocNone {
			d1312 = ps.OverlayValues[1312]
		}
		if len(ps.OverlayValues) > 1313 && ps.OverlayValues[1313].Loc != scm.LocNone {
			d1313 = ps.OverlayValues[1313]
		}
		if len(ps.OverlayValues) > 1314 && ps.OverlayValues[1314].Loc != scm.LocNone {
			d1314 = ps.OverlayValues[1314]
		}
		if len(ps.OverlayValues) > 1315 && ps.OverlayValues[1315].Loc != scm.LocNone {
			d1315 = ps.OverlayValues[1315]
		}
		if len(ps.OverlayValues) > 1316 && ps.OverlayValues[1316].Loc != scm.LocNone {
			d1316 = ps.OverlayValues[1316]
		}
		if len(ps.OverlayValues) > 1317 && ps.OverlayValues[1317].Loc != scm.LocNone {
			d1317 = ps.OverlayValues[1317]
		}
		if len(ps.OverlayValues) > 1318 && ps.OverlayValues[1318].Loc != scm.LocNone {
			d1318 = ps.OverlayValues[1318]
		}
		if len(ps.OverlayValues) > 1319 && ps.OverlayValues[1319].Loc != scm.LocNone {
			d1319 = ps.OverlayValues[1319]
		}
		if len(ps.OverlayValues) > 1320 && ps.OverlayValues[1320].Loc != scm.LocNone {
			d1320 = ps.OverlayValues[1320]
		}
		if len(ps.OverlayValues) > 1321 && ps.OverlayValues[1321].Loc != scm.LocNone {
			d1321 = ps.OverlayValues[1321]
		}
		if len(ps.OverlayValues) > 1322 && ps.OverlayValues[1322].Loc != scm.LocNone {
			d1322 = ps.OverlayValues[1322]
		}
		if len(ps.OverlayValues) > 1323 && ps.OverlayValues[1323].Loc != scm.LocNone {
			d1323 = ps.OverlayValues[1323]
		}
		if len(ps.OverlayValues) > 1324 && ps.OverlayValues[1324].Loc != scm.LocNone {
			d1324 = ps.OverlayValues[1324]
		}
		if len(ps.OverlayValues) > 1325 && ps.OverlayValues[1325].Loc != scm.LocNone {
			d1325 = ps.OverlayValues[1325]
		}
		if len(ps.OverlayValues) > 1326 && ps.OverlayValues[1326].Loc != scm.LocNone {
			d1326 = ps.OverlayValues[1326]
		}
		if len(ps.OverlayValues) > 1327 && ps.OverlayValues[1327].Loc != scm.LocNone {
			d1327 = ps.OverlayValues[1327]
		}
		if len(ps.OverlayValues) > 1328 && ps.OverlayValues[1328].Loc != scm.LocNone {
			d1328 = ps.OverlayValues[1328]
		}
		if len(ps.OverlayValues) > 1329 && ps.OverlayValues[1329].Loc != scm.LocNone {
			d1329 = ps.OverlayValues[1329]
		}
		if len(ps.OverlayValues) > 1330 && ps.OverlayValues[1330].Loc != scm.LocNone {
			d1330 = ps.OverlayValues[1330]
		}
		if len(ps.OverlayValues) > 1331 && ps.OverlayValues[1331].Loc != scm.LocNone {
			d1331 = ps.OverlayValues[1331]
		}
		if len(ps.OverlayValues) > 1332 && ps.OverlayValues[1332].Loc != scm.LocNone {
			d1332 = ps.OverlayValues[1332]
		}
		if len(ps.OverlayValues) > 1333 && ps.OverlayValues[1333].Loc != scm.LocNone {
			d1333 = ps.OverlayValues[1333]
		}
		if len(ps.OverlayValues) > 1334 && ps.OverlayValues[1334].Loc != scm.LocNone {
			d1334 = ps.OverlayValues[1334]
		}
		if len(ps.OverlayValues) > 1335 && ps.OverlayValues[1335].Loc != scm.LocNone {
			d1335 = ps.OverlayValues[1335]
		}
		if len(ps.OverlayValues) > 1336 && ps.OverlayValues[1336].Loc != scm.LocNone {
			d1336 = ps.OverlayValues[1336]
		}
		if len(ps.OverlayValues) > 1337 && ps.OverlayValues[1337].Loc != scm.LocNone {
			d1337 = ps.OverlayValues[1337]
		}
		if len(ps.OverlayValues) > 1338 && ps.OverlayValues[1338].Loc != scm.LocNone {
			d1338 = ps.OverlayValues[1338]
		}
		if len(ps.OverlayValues) > 1339 && ps.OverlayValues[1339].Loc != scm.LocNone {
			d1339 = ps.OverlayValues[1339]
		}
		if len(ps.OverlayValues) > 1340 && ps.OverlayValues[1340].Loc != scm.LocNone {
			d1340 = ps.OverlayValues[1340]
		}
		if len(ps.OverlayValues) > 1341 && ps.OverlayValues[1341].Loc != scm.LocNone {
			d1341 = ps.OverlayValues[1341]
		}
		if len(ps.OverlayValues) > 1342 && ps.OverlayValues[1342].Loc != scm.LocNone {
			d1342 = ps.OverlayValues[1342]
		}
		if len(ps.OverlayValues) > 1343 && ps.OverlayValues[1343].Loc != scm.LocNone {
			d1343 = ps.OverlayValues[1343]
		}
		if len(ps.OverlayValues) > 1344 && ps.OverlayValues[1344].Loc != scm.LocNone {
			d1344 = ps.OverlayValues[1344]
		}
		if len(ps.OverlayValues) > 1345 && ps.OverlayValues[1345].Loc != scm.LocNone {
			d1345 = ps.OverlayValues[1345]
		}
		if len(ps.OverlayValues) > 1346 && ps.OverlayValues[1346].Loc != scm.LocNone {
			d1346 = ps.OverlayValues[1346]
		}
		if len(ps.OverlayValues) > 1347 && ps.OverlayValues[1347].Loc != scm.LocNone {
			d1347 = ps.OverlayValues[1347]
		}
		if len(ps.OverlayValues) > 1348 && ps.OverlayValues[1348].Loc != scm.LocNone {
			d1348 = ps.OverlayValues[1348]
		}
		if len(ps.OverlayValues) > 1349 && ps.OverlayValues[1349].Loc != scm.LocNone {
			d1349 = ps.OverlayValues[1349]
		}
		if len(ps.OverlayValues) > 1350 && ps.OverlayValues[1350].Loc != scm.LocNone {
			d1350 = ps.OverlayValues[1350]
		}
		if len(ps.OverlayValues) > 1351 && ps.OverlayValues[1351].Loc != scm.LocNone {
			d1351 = ps.OverlayValues[1351]
		}
		if len(ps.OverlayValues) > 1352 && ps.OverlayValues[1352].Loc != scm.LocNone {
			d1352 = ps.OverlayValues[1352]
		}
		if len(ps.OverlayValues) > 1353 && ps.OverlayValues[1353].Loc != scm.LocNone {
			d1353 = ps.OverlayValues[1353]
		}
		if len(ps.OverlayValues) > 1354 && ps.OverlayValues[1354].Loc != scm.LocNone {
			d1354 = ps.OverlayValues[1354]
		}
		if len(ps.OverlayValues) > 1355 && ps.OverlayValues[1355].Loc != scm.LocNone {
			d1355 = ps.OverlayValues[1355]
		}
		if len(ps.OverlayValues) > 1356 && ps.OverlayValues[1356].Loc != scm.LocNone {
			d1356 = ps.OverlayValues[1356]
		}
		ctx.ReclaimUntrackedRegs()
		var d1357 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 88
			val := *(*uint64)(unsafe.Pointer(fieldAddr))
			d1357 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 88)
			r174 := ctx.AllocReg()
			ctx.EmitMovRegMem(r174, thisptr.Reg, off)
			d1357 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r174}
			ctx.BindReg(r174, &d1357)
		}
		ctx.EnsureDesc(&d1357)
		ctx.EnsureDesc(&d1357)
		var d1358 scm.JITValueDesc
		if d1357.Loc == scm.LocImm {
			d1358 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d1357.Imm.Int()))))}
		} else {
			r175 := ctx.AllocReg()
			ctx.EmitMovRegReg(r175, d1357.Reg)
			d1358 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r175}
			ctx.BindReg(r175, &d1358)
		}
		ctx.FreeDesc(&d1357)
		ctx.EnsureDesc(&d204)
		ctx.EnsureDesc(&d1358)
		ctx.EnsureDescsTogether(&d204, &d1358)
		var d1359 scm.JITValueDesc
		if d204.Loc == scm.LocImm && d1358.Loc == scm.LocImm {
			d1359 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d204.Imm.Int() == d1358.Imm.Int())}
		} else if d1358.Loc == scm.LocImm {
			r176 := ctx.AllocRegExcept(d204.Reg)
			if d1358.Imm.Int() >= -2147483648 && d1358.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d204.Reg, int32(d1358.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1358.Imm.Int()))
				ctx.EmitCmpInt64(d204.Reg, scm.RegR11)
			}
			d1359 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r176, Condition: scm.CondEqual}
			ctx.BindReg(r176, &d1359)
		} else if d204.Loc == scm.LocImm {
			r177 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d204.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d1358.Reg)
			d1359 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r177, Condition: scm.CondEqual}
			ctx.BindReg(r177, &d1359)
		} else {
			r178 := ctx.AllocRegExcept(d204.Reg)
			ctx.EmitCmpInt64(d204.Reg, d1358.Reg)
			d1359 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r178, Condition: scm.CondEqual}
			ctx.BindReg(r178, &d1359)
		}
		ctx.FreeDesc(&d1358)
		d1360 = d1359
		ctx.EnsureDesc(&d1360)
		if d1360.Loc != scm.LocImm && d1360.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d1360.Loc == scm.LocImm {
			if d1360.Imm.Bool() {
				if ps.General {
				}
				ps1361 := scm.PhiState{General: ps.General}
				ps1361.OverlayValues = make([]scm.JITValueDesc, 1361)
				ps1361.OverlayValues[5] = d5
				ps1361.OverlayValues[6] = d6
				ps1361.OverlayValues[7] = d7
				ps1361.OverlayValues[8] = d8
				ps1361.OverlayValues[9] = d9
				ps1361.OverlayValues[10] = d10
				ps1361.OverlayValues[11] = d11
				ps1361.OverlayValues[12] = d12
				ps1361.OverlayValues[13] = d13
				ps1361.OverlayValues[14] = d14
				ps1361.OverlayValues[15] = d15
				ps1361.OverlayValues[16] = d16
				ps1361.OverlayValues[17] = d17
				ps1361.OverlayValues[18] = d18
				ps1361.OverlayValues[19] = d19
				ps1361.OverlayValues[20] = d20
				ps1361.OverlayValues[21] = d21
				ps1361.OverlayValues[23] = d23
				ps1361.OverlayValues[24] = d24
				ps1361.OverlayValues[25] = d25
				ps1361.OverlayValues[26] = d26
				ps1361.OverlayValues[27] = d27
				ps1361.OverlayValues[28] = d28
				ps1361.OverlayValues[29] = d29
				ps1361.OverlayValues[30] = d30
				ps1361.OverlayValues[31] = d31
				ps1361.OverlayValues[32] = d32
				ps1361.OverlayValues[33] = d33
				ps1361.OverlayValues[34] = d34
				ps1361.OverlayValues[35] = d35
				ps1361.OverlayValues[36] = d36
				ps1361.OverlayValues[37] = d37
				ps1361.OverlayValues[38] = d38
				ps1361.OverlayValues[39] = d39
				ps1361.OverlayValues[40] = d40
				ps1361.OverlayValues[41] = d41
				ps1361.OverlayValues[42] = d42
				ps1361.OverlayValues[43] = d43
				ps1361.OverlayValues[44] = d44
				ps1361.OverlayValues[45] = d45
				ps1361.OverlayValues[46] = d46
				ps1361.OverlayValues[47] = d47
				ps1361.OverlayValues[48] = d48
				ps1361.OverlayValues[49] = d49
				ps1361.OverlayValues[50] = d50
				ps1361.OverlayValues[51] = d51
				ps1361.OverlayValues[52] = d52
				ps1361.OverlayValues[53] = d53
				ps1361.OverlayValues[54] = d54
				ps1361.OverlayValues[55] = d55
				ps1361.OverlayValues[56] = d56
				ps1361.OverlayValues[57] = d57
				ps1361.OverlayValues[60] = d60
				ps1361.OverlayValues[61] = d61
				ps1361.OverlayValues[62] = d62
				ps1361.OverlayValues[177] = d177
				ps1361.OverlayValues[178] = d178
				ps1361.OverlayValues[179] = d179
				ps1361.OverlayValues[180] = d180
				ps1361.OverlayValues[181] = d181
				ps1361.OverlayValues[182] = d182
				ps1361.OverlayValues[183] = d183
				ps1361.OverlayValues[184] = d184
				ps1361.OverlayValues[185] = d185
				ps1361.OverlayValues[186] = d186
				ps1361.OverlayValues[187] = d187
				ps1361.OverlayValues[188] = d188
				ps1361.OverlayValues[189] = d189
				ps1361.OverlayValues[190] = d190
				ps1361.OverlayValues[191] = d191
				ps1361.OverlayValues[192] = d192
				ps1361.OverlayValues[193] = d193
				ps1361.OverlayValues[194] = d194
				ps1361.OverlayValues[195] = d195
				ps1361.OverlayValues[196] = d196
				ps1361.OverlayValues[197] = d197
				ps1361.OverlayValues[198] = d198
				ps1361.OverlayValues[199] = d199
				ps1361.OverlayValues[200] = d200
				ps1361.OverlayValues[201] = d201
				ps1361.OverlayValues[202] = d202
				ps1361.OverlayValues[203] = d203
				ps1361.OverlayValues[204] = d204
				ps1361.OverlayValues[205] = d205
				ps1361.OverlayValues[206] = d206
				ps1361.OverlayValues[209] = d209
				ps1361.OverlayValues[386] = d386
				ps1361.OverlayValues[387] = d387
				ps1361.OverlayValues[388] = d388
				ps1361.OverlayValues[389] = d389
				ps1361.OverlayValues[391] = d391
				ps1361.OverlayValues[392] = d392
				ps1361.OverlayValues[393] = d393
				ps1361.OverlayValues[394] = d394
				ps1361.OverlayValues[395] = d395
				ps1361.OverlayValues[396] = d396
				ps1361.OverlayValues[397] = d397
				ps1361.OverlayValues[398] = d398
				ps1361.OverlayValues[400] = d400
				ps1361.OverlayValues[402] = d402
				ps1361.OverlayValues[403] = d403
				ps1361.OverlayValues[404] = d404
				ps1361.OverlayValues[508] = d508
				ps1361.OverlayValues[509] = d509
				ps1361.OverlayValues[512] = d512
				ps1361.OverlayValues[619] = d619
				ps1361.OverlayValues[620] = d620
				ps1361.OverlayValues[621] = d621
				ps1361.OverlayValues[622] = d622
				ps1361.OverlayValues[623] = d623
				ps1361.OverlayValues[625] = d625
				ps1361.OverlayValues[626] = d626
				ps1361.OverlayValues[627] = d627
				ps1361.OverlayValues[628] = d628
				ps1361.OverlayValues[629] = d629
				ps1361.OverlayValues[630] = d630
				ps1361.OverlayValues[631] = d631
				ps1361.OverlayValues[632] = d632
				ps1361.OverlayValues[633] = d633
				ps1361.OverlayValues[634] = d634
				ps1361.OverlayValues[635] = d635
				ps1361.OverlayValues[636] = d636
				ps1361.OverlayValues[637] = d637
				ps1361.OverlayValues[638] = d638
				ps1361.OverlayValues[639] = d639
				ps1361.OverlayValues[640] = d640
				ps1361.OverlayValues[641] = d641
				ps1361.OverlayValues[642] = d642
				ps1361.OverlayValues[643] = d643
				ps1361.OverlayValues[644] = d644
				ps1361.OverlayValues[645] = d645
				ps1361.OverlayValues[646] = d646
				ps1361.OverlayValues[647] = d647
				ps1361.OverlayValues[648] = d648
				ps1361.OverlayValues[649] = d649
				ps1361.OverlayValues[650] = d650
				ps1361.OverlayValues[651] = d651
				ps1361.OverlayValues[652] = d652
				ps1361.OverlayValues[653] = d653
				ps1361.OverlayValues[654] = d654
				ps1361.OverlayValues[655] = d655
				ps1361.OverlayValues[944] = d944
				ps1361.OverlayValues[945] = d945
				ps1361.OverlayValues[946] = d946
				ps1361.OverlayValues[948] = d948
				ps1361.OverlayValues[949] = d949
				ps1361.OverlayValues[950] = d950
				ps1361.OverlayValues[951] = d951
				ps1361.OverlayValues[952] = d952
				ps1361.OverlayValues[953] = d953
				ps1361.OverlayValues[954] = d954
				ps1361.OverlayValues[956] = d956
				ps1361.OverlayValues[958] = d958
				ps1361.OverlayValues[959] = d959
				ps1361.OverlayValues[1115] = d1115
				ps1361.OverlayValues[1116] = d1116
				ps1361.OverlayValues[1119] = d1119
				ps1361.OverlayValues[1278] = d1278
				ps1361.OverlayValues[1279] = d1279
				ps1361.OverlayValues[1280] = d1280
				ps1361.OverlayValues[1281] = d1281
				ps1361.OverlayValues[1283] = d1283
				ps1361.OverlayValues[1284] = d1284
				ps1361.OverlayValues[1285] = d1285
				ps1361.OverlayValues[1286] = d1286
				ps1361.OverlayValues[1287] = d1287
				ps1361.OverlayValues[1288] = d1288
				ps1361.OverlayValues[1289] = d1289
				ps1361.OverlayValues[1290] = d1290
				ps1361.OverlayValues[1291] = d1291
				ps1361.OverlayValues[1292] = d1292
				ps1361.OverlayValues[1294] = d1294
				ps1361.OverlayValues[1295] = d1295
				ps1361.OverlayValues[1296] = d1296
				ps1361.OverlayValues[1297] = d1297
				ps1361.OverlayValues[1298] = d1298
				ps1361.OverlayValues[1299] = d1299
				ps1361.OverlayValues[1300] = d1300
				ps1361.OverlayValues[1301] = d1301
				ps1361.OverlayValues[1302] = d1302
				ps1361.OverlayValues[1303] = d1303
				ps1361.OverlayValues[1304] = d1304
				ps1361.OverlayValues[1305] = d1305
				ps1361.OverlayValues[1306] = d1306
				ps1361.OverlayValues[1307] = d1307
				ps1361.OverlayValues[1308] = d1308
				ps1361.OverlayValues[1309] = d1309
				ps1361.OverlayValues[1310] = d1310
				ps1361.OverlayValues[1311] = d1311
				ps1361.OverlayValues[1312] = d1312
				ps1361.OverlayValues[1313] = d1313
				ps1361.OverlayValues[1314] = d1314
				ps1361.OverlayValues[1315] = d1315
				ps1361.OverlayValues[1316] = d1316
				ps1361.OverlayValues[1317] = d1317
				ps1361.OverlayValues[1318] = d1318
				ps1361.OverlayValues[1319] = d1319
				ps1361.OverlayValues[1320] = d1320
				ps1361.OverlayValues[1321] = d1321
				ps1361.OverlayValues[1322] = d1322
				ps1361.OverlayValues[1323] = d1323
				ps1361.OverlayValues[1324] = d1324
				ps1361.OverlayValues[1325] = d1325
				ps1361.OverlayValues[1326] = d1326
				ps1361.OverlayValues[1327] = d1327
				ps1361.OverlayValues[1328] = d1328
				ps1361.OverlayValues[1329] = d1329
				ps1361.OverlayValues[1330] = d1330
				ps1361.OverlayValues[1331] = d1331
				ps1361.OverlayValues[1332] = d1332
				ps1361.OverlayValues[1333] = d1333
				ps1361.OverlayValues[1334] = d1334
				ps1361.OverlayValues[1335] = d1335
				ps1361.OverlayValues[1336] = d1336
				ps1361.OverlayValues[1337] = d1337
				ps1361.OverlayValues[1338] = d1338
				ps1361.OverlayValues[1339] = d1339
				ps1361.OverlayValues[1340] = d1340
				ps1361.OverlayValues[1341] = d1341
				ps1361.OverlayValues[1342] = d1342
				ps1361.OverlayValues[1343] = d1343
				ps1361.OverlayValues[1344] = d1344
				ps1361.OverlayValues[1345] = d1345
				ps1361.OverlayValues[1346] = d1346
				ps1361.OverlayValues[1347] = d1347
				ps1361.OverlayValues[1348] = d1348
				ps1361.OverlayValues[1349] = d1349
				ps1361.OverlayValues[1350] = d1350
				ps1361.OverlayValues[1351] = d1351
				ps1361.OverlayValues[1352] = d1352
				ps1361.OverlayValues[1353] = d1353
				ps1361.OverlayValues[1354] = d1354
				ps1361.OverlayValues[1355] = d1355
				ps1361.OverlayValues[1356] = d1356
				ps1361.OverlayValues[1357] = d1357
				ps1361.OverlayValues[1358] = d1358
				ps1361.OverlayValues[1359] = d1359
				ps1361.OverlayValues[1360] = d1360
				return bbs[11].RenderPS(ps1361)
			}
			if ps.General {
			}
			ps1362 := scm.PhiState{General: ps.General}
			ps1362.OverlayValues = make([]scm.JITValueDesc, 1361)
			ps1362.OverlayValues[5] = d5
			ps1362.OverlayValues[6] = d6
			ps1362.OverlayValues[7] = d7
			ps1362.OverlayValues[8] = d8
			ps1362.OverlayValues[9] = d9
			ps1362.OverlayValues[10] = d10
			ps1362.OverlayValues[11] = d11
			ps1362.OverlayValues[12] = d12
			ps1362.OverlayValues[13] = d13
			ps1362.OverlayValues[14] = d14
			ps1362.OverlayValues[15] = d15
			ps1362.OverlayValues[16] = d16
			ps1362.OverlayValues[17] = d17
			ps1362.OverlayValues[18] = d18
			ps1362.OverlayValues[19] = d19
			ps1362.OverlayValues[20] = d20
			ps1362.OverlayValues[21] = d21
			ps1362.OverlayValues[23] = d23
			ps1362.OverlayValues[24] = d24
			ps1362.OverlayValues[25] = d25
			ps1362.OverlayValues[26] = d26
			ps1362.OverlayValues[27] = d27
			ps1362.OverlayValues[28] = d28
			ps1362.OverlayValues[29] = d29
			ps1362.OverlayValues[30] = d30
			ps1362.OverlayValues[31] = d31
			ps1362.OverlayValues[32] = d32
			ps1362.OverlayValues[33] = d33
			ps1362.OverlayValues[34] = d34
			ps1362.OverlayValues[35] = d35
			ps1362.OverlayValues[36] = d36
			ps1362.OverlayValues[37] = d37
			ps1362.OverlayValues[38] = d38
			ps1362.OverlayValues[39] = d39
			ps1362.OverlayValues[40] = d40
			ps1362.OverlayValues[41] = d41
			ps1362.OverlayValues[42] = d42
			ps1362.OverlayValues[43] = d43
			ps1362.OverlayValues[44] = d44
			ps1362.OverlayValues[45] = d45
			ps1362.OverlayValues[46] = d46
			ps1362.OverlayValues[47] = d47
			ps1362.OverlayValues[48] = d48
			ps1362.OverlayValues[49] = d49
			ps1362.OverlayValues[50] = d50
			ps1362.OverlayValues[51] = d51
			ps1362.OverlayValues[52] = d52
			ps1362.OverlayValues[53] = d53
			ps1362.OverlayValues[54] = d54
			ps1362.OverlayValues[55] = d55
			ps1362.OverlayValues[56] = d56
			ps1362.OverlayValues[57] = d57
			ps1362.OverlayValues[60] = d60
			ps1362.OverlayValues[61] = d61
			ps1362.OverlayValues[62] = d62
			ps1362.OverlayValues[177] = d177
			ps1362.OverlayValues[178] = d178
			ps1362.OverlayValues[179] = d179
			ps1362.OverlayValues[180] = d180
			ps1362.OverlayValues[181] = d181
			ps1362.OverlayValues[182] = d182
			ps1362.OverlayValues[183] = d183
			ps1362.OverlayValues[184] = d184
			ps1362.OverlayValues[185] = d185
			ps1362.OverlayValues[186] = d186
			ps1362.OverlayValues[187] = d187
			ps1362.OverlayValues[188] = d188
			ps1362.OverlayValues[189] = d189
			ps1362.OverlayValues[190] = d190
			ps1362.OverlayValues[191] = d191
			ps1362.OverlayValues[192] = d192
			ps1362.OverlayValues[193] = d193
			ps1362.OverlayValues[194] = d194
			ps1362.OverlayValues[195] = d195
			ps1362.OverlayValues[196] = d196
			ps1362.OverlayValues[197] = d197
			ps1362.OverlayValues[198] = d198
			ps1362.OverlayValues[199] = d199
			ps1362.OverlayValues[200] = d200
			ps1362.OverlayValues[201] = d201
			ps1362.OverlayValues[202] = d202
			ps1362.OverlayValues[203] = d203
			ps1362.OverlayValues[204] = d204
			ps1362.OverlayValues[205] = d205
			ps1362.OverlayValues[206] = d206
			ps1362.OverlayValues[209] = d209
			ps1362.OverlayValues[386] = d386
			ps1362.OverlayValues[387] = d387
			ps1362.OverlayValues[388] = d388
			ps1362.OverlayValues[389] = d389
			ps1362.OverlayValues[391] = d391
			ps1362.OverlayValues[392] = d392
			ps1362.OverlayValues[393] = d393
			ps1362.OverlayValues[394] = d394
			ps1362.OverlayValues[395] = d395
			ps1362.OverlayValues[396] = d396
			ps1362.OverlayValues[397] = d397
			ps1362.OverlayValues[398] = d398
			ps1362.OverlayValues[400] = d400
			ps1362.OverlayValues[402] = d402
			ps1362.OverlayValues[403] = d403
			ps1362.OverlayValues[404] = d404
			ps1362.OverlayValues[508] = d508
			ps1362.OverlayValues[509] = d509
			ps1362.OverlayValues[512] = d512
			ps1362.OverlayValues[619] = d619
			ps1362.OverlayValues[620] = d620
			ps1362.OverlayValues[621] = d621
			ps1362.OverlayValues[622] = d622
			ps1362.OverlayValues[623] = d623
			ps1362.OverlayValues[625] = d625
			ps1362.OverlayValues[626] = d626
			ps1362.OverlayValues[627] = d627
			ps1362.OverlayValues[628] = d628
			ps1362.OverlayValues[629] = d629
			ps1362.OverlayValues[630] = d630
			ps1362.OverlayValues[631] = d631
			ps1362.OverlayValues[632] = d632
			ps1362.OverlayValues[633] = d633
			ps1362.OverlayValues[634] = d634
			ps1362.OverlayValues[635] = d635
			ps1362.OverlayValues[636] = d636
			ps1362.OverlayValues[637] = d637
			ps1362.OverlayValues[638] = d638
			ps1362.OverlayValues[639] = d639
			ps1362.OverlayValues[640] = d640
			ps1362.OverlayValues[641] = d641
			ps1362.OverlayValues[642] = d642
			ps1362.OverlayValues[643] = d643
			ps1362.OverlayValues[644] = d644
			ps1362.OverlayValues[645] = d645
			ps1362.OverlayValues[646] = d646
			ps1362.OverlayValues[647] = d647
			ps1362.OverlayValues[648] = d648
			ps1362.OverlayValues[649] = d649
			ps1362.OverlayValues[650] = d650
			ps1362.OverlayValues[651] = d651
			ps1362.OverlayValues[652] = d652
			ps1362.OverlayValues[653] = d653
			ps1362.OverlayValues[654] = d654
			ps1362.OverlayValues[655] = d655
			ps1362.OverlayValues[944] = d944
			ps1362.OverlayValues[945] = d945
			ps1362.OverlayValues[946] = d946
			ps1362.OverlayValues[948] = d948
			ps1362.OverlayValues[949] = d949
			ps1362.OverlayValues[950] = d950
			ps1362.OverlayValues[951] = d951
			ps1362.OverlayValues[952] = d952
			ps1362.OverlayValues[953] = d953
			ps1362.OverlayValues[954] = d954
			ps1362.OverlayValues[956] = d956
			ps1362.OverlayValues[958] = d958
			ps1362.OverlayValues[959] = d959
			ps1362.OverlayValues[1115] = d1115
			ps1362.OverlayValues[1116] = d1116
			ps1362.OverlayValues[1119] = d1119
			ps1362.OverlayValues[1278] = d1278
			ps1362.OverlayValues[1279] = d1279
			ps1362.OverlayValues[1280] = d1280
			ps1362.OverlayValues[1281] = d1281
			ps1362.OverlayValues[1283] = d1283
			ps1362.OverlayValues[1284] = d1284
			ps1362.OverlayValues[1285] = d1285
			ps1362.OverlayValues[1286] = d1286
			ps1362.OverlayValues[1287] = d1287
			ps1362.OverlayValues[1288] = d1288
			ps1362.OverlayValues[1289] = d1289
			ps1362.OverlayValues[1290] = d1290
			ps1362.OverlayValues[1291] = d1291
			ps1362.OverlayValues[1292] = d1292
			ps1362.OverlayValues[1294] = d1294
			ps1362.OverlayValues[1295] = d1295
			ps1362.OverlayValues[1296] = d1296
			ps1362.OverlayValues[1297] = d1297
			ps1362.OverlayValues[1298] = d1298
			ps1362.OverlayValues[1299] = d1299
			ps1362.OverlayValues[1300] = d1300
			ps1362.OverlayValues[1301] = d1301
			ps1362.OverlayValues[1302] = d1302
			ps1362.OverlayValues[1303] = d1303
			ps1362.OverlayValues[1304] = d1304
			ps1362.OverlayValues[1305] = d1305
			ps1362.OverlayValues[1306] = d1306
			ps1362.OverlayValues[1307] = d1307
			ps1362.OverlayValues[1308] = d1308
			ps1362.OverlayValues[1309] = d1309
			ps1362.OverlayValues[1310] = d1310
			ps1362.OverlayValues[1311] = d1311
			ps1362.OverlayValues[1312] = d1312
			ps1362.OverlayValues[1313] = d1313
			ps1362.OverlayValues[1314] = d1314
			ps1362.OverlayValues[1315] = d1315
			ps1362.OverlayValues[1316] = d1316
			ps1362.OverlayValues[1317] = d1317
			ps1362.OverlayValues[1318] = d1318
			ps1362.OverlayValues[1319] = d1319
			ps1362.OverlayValues[1320] = d1320
			ps1362.OverlayValues[1321] = d1321
			ps1362.OverlayValues[1322] = d1322
			ps1362.OverlayValues[1323] = d1323
			ps1362.OverlayValues[1324] = d1324
			ps1362.OverlayValues[1325] = d1325
			ps1362.OverlayValues[1326] = d1326
			ps1362.OverlayValues[1327] = d1327
			ps1362.OverlayValues[1328] = d1328
			ps1362.OverlayValues[1329] = d1329
			ps1362.OverlayValues[1330] = d1330
			ps1362.OverlayValues[1331] = d1331
			ps1362.OverlayValues[1332] = d1332
			ps1362.OverlayValues[1333] = d1333
			ps1362.OverlayValues[1334] = d1334
			ps1362.OverlayValues[1335] = d1335
			ps1362.OverlayValues[1336] = d1336
			ps1362.OverlayValues[1337] = d1337
			ps1362.OverlayValues[1338] = d1338
			ps1362.OverlayValues[1339] = d1339
			ps1362.OverlayValues[1340] = d1340
			ps1362.OverlayValues[1341] = d1341
			ps1362.OverlayValues[1342] = d1342
			ps1362.OverlayValues[1343] = d1343
			ps1362.OverlayValues[1344] = d1344
			ps1362.OverlayValues[1345] = d1345
			ps1362.OverlayValues[1346] = d1346
			ps1362.OverlayValues[1347] = d1347
			ps1362.OverlayValues[1348] = d1348
			ps1362.OverlayValues[1349] = d1349
			ps1362.OverlayValues[1350] = d1350
			ps1362.OverlayValues[1351] = d1351
			ps1362.OverlayValues[1352] = d1352
			ps1362.OverlayValues[1353] = d1353
			ps1362.OverlayValues[1354] = d1354
			ps1362.OverlayValues[1355] = d1355
			ps1362.OverlayValues[1356] = d1356
			ps1362.OverlayValues[1357] = d1357
			ps1362.OverlayValues[1358] = d1358
			ps1362.OverlayValues[1359] = d1359
			ps1362.OverlayValues[1360] = d1360
			return bbs[12].RenderPS(ps1362)
		}
		if !ps.General {
			ps.General = true
			return bbs[13].RenderPS(ps)
		}
		lbl30 := ctx.ReserveLabel()
		lbl31 := ctx.ReserveLabel()
		ctx.EmitJump(d1360.Condition, lbl30)
		ctx.EmitJmp(lbl31)
		snap1363 := d5
		snap1364 := d6
		snap1365 := d7
		snap1366 := d8
		snap1367 := d9
		snap1368 := d10
		snap1369 := d11
		snap1370 := d12
		snap1371 := d13
		snap1372 := d14
		snap1373 := d15
		snap1374 := d16
		snap1375 := d17
		snap1376 := d18
		snap1377 := d19
		snap1378 := d20
		snap1379 := d21
		snap1380 := d23
		snap1381 := d24
		snap1382 := d25
		snap1383 := d26
		snap1384 := d27
		snap1385 := d28
		snap1386 := d29
		snap1387 := d30
		snap1388 := d31
		snap1389 := d32
		snap1390 := d33
		snap1391 := d34
		snap1392 := d35
		snap1393 := d36
		snap1394 := d37
		snap1395 := d38
		snap1396 := d39
		snap1397 := d40
		snap1398 := d41
		snap1399 := d42
		snap1400 := d43
		snap1401 := d44
		snap1402 := d45
		snap1403 := d46
		snap1404 := d47
		snap1405 := d48
		snap1406 := d49
		snap1407 := d50
		snap1408 := d51
		snap1409 := d52
		snap1410 := d53
		snap1411 := d54
		snap1412 := d55
		snap1413 := d56
		snap1414 := d57
		snap1415 := d60
		snap1416 := d61
		snap1417 := d62
		snap1418 := d177
		snap1419 := d178
		snap1420 := d179
		snap1421 := d180
		snap1422 := d181
		snap1423 := d182
		snap1424 := d183
		snap1425 := d184
		snap1426 := d185
		snap1427 := d186
		snap1428 := d187
		snap1429 := d188
		snap1430 := d189
		snap1431 := d190
		snap1432 := d191
		snap1433 := d192
		snap1434 := d193
		snap1435 := d194
		snap1436 := d195
		snap1437 := d196
		snap1438 := d197
		snap1439 := d198
		snap1440 := d199
		snap1441 := d200
		snap1442 := d201
		snap1443 := d202
		snap1444 := d203
		snap1445 := d204
		snap1446 := d205
		snap1447 := d206
		snap1448 := d209
		snap1449 := d386
		snap1450 := d387
		snap1451 := d388
		snap1452 := d389
		snap1453 := d391
		snap1454 := d392
		snap1455 := d393
		snap1456 := d394
		snap1457 := d395
		snap1458 := d396
		snap1459 := d397
		snap1460 := d398
		snap1461 := d400
		snap1462 := d402
		snap1463 := d403
		snap1464 := d404
		snap1465 := d508
		snap1466 := d509
		snap1467 := d512
		snap1468 := d619
		snap1469 := d620
		snap1470 := d621
		snap1471 := d622
		snap1472 := d623
		snap1473 := d625
		snap1474 := d626
		snap1475 := d627
		snap1476 := d628
		snap1477 := d629
		snap1478 := d630
		snap1479 := d631
		snap1480 := d632
		snap1481 := d633
		snap1482 := d634
		snap1483 := d635
		snap1484 := d636
		snap1485 := d637
		snap1486 := d638
		snap1487 := d639
		snap1488 := d640
		snap1489 := d641
		snap1490 := d642
		snap1491 := d643
		snap1492 := d644
		snap1493 := d645
		snap1494 := d646
		snap1495 := d647
		snap1496 := d648
		snap1497 := d649
		snap1498 := d650
		snap1499 := d651
		snap1500 := d652
		snap1501 := d653
		snap1502 := d654
		snap1503 := d655
		snap1504 := d944
		snap1505 := d945
		snap1506 := d946
		snap1507 := d948
		snap1508 := d949
		snap1509 := d950
		snap1510 := d951
		snap1511 := d952
		snap1512 := d953
		snap1513 := d954
		snap1514 := d956
		snap1515 := d958
		snap1516 := d959
		snap1517 := d1115
		snap1518 := d1116
		snap1519 := d1119
		snap1520 := d1278
		snap1521 := d1279
		snap1522 := d1280
		snap1523 := d1281
		snap1524 := d1283
		snap1525 := d1284
		snap1526 := d1285
		snap1527 := d1286
		snap1528 := d1287
		snap1529 := d1288
		snap1530 := d1289
		snap1531 := d1290
		snap1532 := d1291
		snap1533 := d1292
		snap1534 := d1294
		snap1535 := d1295
		snap1536 := d1296
		snap1537 := d1297
		snap1538 := d1298
		snap1539 := d1299
		snap1540 := d1300
		snap1541 := d1301
		snap1542 := d1302
		snap1543 := d1303
		snap1544 := d1304
		snap1545 := d1305
		snap1546 := d1306
		snap1547 := d1307
		snap1548 := d1308
		snap1549 := d1309
		snap1550 := d1310
		snap1551 := d1311
		snap1552 := d1312
		snap1553 := d1313
		snap1554 := d1314
		snap1555 := d1315
		snap1556 := d1316
		snap1557 := d1317
		snap1558 := d1318
		snap1559 := d1319
		snap1560 := d1320
		snap1561 := d1321
		snap1562 := d1322
		snap1563 := d1323
		snap1564 := d1324
		snap1565 := d1325
		snap1566 := d1326
		snap1567 := d1327
		snap1568 := d1328
		snap1569 := d1329
		snap1570 := d1330
		snap1571 := d1331
		snap1572 := d1332
		snap1573 := d1333
		snap1574 := d1334
		snap1575 := d1335
		snap1576 := d1336
		snap1577 := d1337
		snap1578 := d1338
		snap1579 := d1339
		snap1580 := d1340
		snap1581 := d1341
		snap1582 := d1342
		snap1583 := d1343
		snap1584 := d1344
		snap1585 := d1345
		snap1586 := d1346
		snap1587 := d1347
		snap1588 := d1348
		snap1589 := d1349
		snap1590 := d1350
		snap1591 := d1351
		snap1592 := d1352
		snap1593 := d1353
		snap1594 := d1354
		snap1595 := d1355
		snap1596 := d1356
		snap1597 := d1357
		snap1598 := d1358
		snap1599 := d1359
		snap1600 := d1360
		alloc1601 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl30)
		ctx.EmitJmp(lbl12)
		ctx.RestoreAllocState(alloc1601)
		d5 = snap1363
		d6 = snap1364
		d7 = snap1365
		d8 = snap1366
		d9 = snap1367
		d10 = snap1368
		d11 = snap1369
		d12 = snap1370
		d13 = snap1371
		d14 = snap1372
		d15 = snap1373
		d16 = snap1374
		d17 = snap1375
		d18 = snap1376
		d19 = snap1377
		d20 = snap1378
		d21 = snap1379
		d23 = snap1380
		d24 = snap1381
		d25 = snap1382
		d26 = snap1383
		d27 = snap1384
		d28 = snap1385
		d29 = snap1386
		d30 = snap1387
		d31 = snap1388
		d32 = snap1389
		d33 = snap1390
		d34 = snap1391
		d35 = snap1392
		d36 = snap1393
		d37 = snap1394
		d38 = snap1395
		d39 = snap1396
		d40 = snap1397
		d41 = snap1398
		d42 = snap1399
		d43 = snap1400
		d44 = snap1401
		d45 = snap1402
		d46 = snap1403
		d47 = snap1404
		d48 = snap1405
		d49 = snap1406
		d50 = snap1407
		d51 = snap1408
		d52 = snap1409
		d53 = snap1410
		d54 = snap1411
		d55 = snap1412
		d56 = snap1413
		d57 = snap1414
		d60 = snap1415
		d61 = snap1416
		d62 = snap1417
		d177 = snap1418
		d178 = snap1419
		d179 = snap1420
		d180 = snap1421
		d181 = snap1422
		d182 = snap1423
		d183 = snap1424
		d184 = snap1425
		d185 = snap1426
		d186 = snap1427
		d187 = snap1428
		d188 = snap1429
		d189 = snap1430
		d190 = snap1431
		d191 = snap1432
		d192 = snap1433
		d193 = snap1434
		d194 = snap1435
		d195 = snap1436
		d196 = snap1437
		d197 = snap1438
		d198 = snap1439
		d199 = snap1440
		d200 = snap1441
		d201 = snap1442
		d202 = snap1443
		d203 = snap1444
		d204 = snap1445
		d205 = snap1446
		d206 = snap1447
		d209 = snap1448
		d386 = snap1449
		d387 = snap1450
		d388 = snap1451
		d389 = snap1452
		d391 = snap1453
		d392 = snap1454
		d393 = snap1455
		d394 = snap1456
		d395 = snap1457
		d396 = snap1458
		d397 = snap1459
		d398 = snap1460
		d400 = snap1461
		d402 = snap1462
		d403 = snap1463
		d404 = snap1464
		d508 = snap1465
		d509 = snap1466
		d512 = snap1467
		d619 = snap1468
		d620 = snap1469
		d621 = snap1470
		d622 = snap1471
		d623 = snap1472
		d625 = snap1473
		d626 = snap1474
		d627 = snap1475
		d628 = snap1476
		d629 = snap1477
		d630 = snap1478
		d631 = snap1479
		d632 = snap1480
		d633 = snap1481
		d634 = snap1482
		d635 = snap1483
		d636 = snap1484
		d637 = snap1485
		d638 = snap1486
		d639 = snap1487
		d640 = snap1488
		d641 = snap1489
		d642 = snap1490
		d643 = snap1491
		d644 = snap1492
		d645 = snap1493
		d646 = snap1494
		d647 = snap1495
		d648 = snap1496
		d649 = snap1497
		d650 = snap1498
		d651 = snap1499
		d652 = snap1500
		d653 = snap1501
		d654 = snap1502
		d655 = snap1503
		d944 = snap1504
		d945 = snap1505
		d946 = snap1506
		d948 = snap1507
		d949 = snap1508
		d950 = snap1509
		d951 = snap1510
		d952 = snap1511
		d953 = snap1512
		d954 = snap1513
		d956 = snap1514
		d958 = snap1515
		d959 = snap1516
		d1115 = snap1517
		d1116 = snap1518
		d1119 = snap1519
		d1278 = snap1520
		d1279 = snap1521
		d1280 = snap1522
		d1281 = snap1523
		d1283 = snap1524
		d1284 = snap1525
		d1285 = snap1526
		d1286 = snap1527
		d1287 = snap1528
		d1288 = snap1529
		d1289 = snap1530
		d1290 = snap1531
		d1291 = snap1532
		d1292 = snap1533
		d1294 = snap1534
		d1295 = snap1535
		d1296 = snap1536
		d1297 = snap1537
		d1298 = snap1538
		d1299 = snap1539
		d1300 = snap1540
		d1301 = snap1541
		d1302 = snap1542
		d1303 = snap1543
		d1304 = snap1544
		d1305 = snap1545
		d1306 = snap1546
		d1307 = snap1547
		d1308 = snap1548
		d1309 = snap1549
		d1310 = snap1550
		d1311 = snap1551
		d1312 = snap1552
		d1313 = snap1553
		d1314 = snap1554
		d1315 = snap1555
		d1316 = snap1556
		d1317 = snap1557
		d1318 = snap1558
		d1319 = snap1559
		d1320 = snap1560
		d1321 = snap1561
		d1322 = snap1562
		d1323 = snap1563
		d1324 = snap1564
		d1325 = snap1565
		d1326 = snap1566
		d1327 = snap1567
		d1328 = snap1568
		d1329 = snap1569
		d1330 = snap1570
		d1331 = snap1571
		d1332 = snap1572
		d1333 = snap1573
		d1334 = snap1574
		d1335 = snap1575
		d1336 = snap1576
		d1337 = snap1577
		d1338 = snap1578
		d1339 = snap1579
		d1340 = snap1580
		d1341 = snap1581
		d1342 = snap1582
		d1343 = snap1583
		d1344 = snap1584
		d1345 = snap1585
		d1346 = snap1586
		d1347 = snap1587
		d1348 = snap1588
		d1349 = snap1589
		d1350 = snap1590
		d1351 = snap1591
		d1352 = snap1592
		d1353 = snap1593
		d1354 = snap1594
		d1355 = snap1595
		d1356 = snap1596
		d1357 = snap1597
		d1358 = snap1598
		d1359 = snap1599
		d1360 = snap1600
		ctx.MarkLabel(lbl31)
		ctx.EmitJmp(lbl13)
		ctx.RestoreAllocState(alloc1601)
		d5 = snap1363
		d6 = snap1364
		d7 = snap1365
		d8 = snap1366
		d9 = snap1367
		d10 = snap1368
		d11 = snap1369
		d12 = snap1370
		d13 = snap1371
		d14 = snap1372
		d15 = snap1373
		d16 = snap1374
		d17 = snap1375
		d18 = snap1376
		d19 = snap1377
		d20 = snap1378
		d21 = snap1379
		d23 = snap1380
		d24 = snap1381
		d25 = snap1382
		d26 = snap1383
		d27 = snap1384
		d28 = snap1385
		d29 = snap1386
		d30 = snap1387
		d31 = snap1388
		d32 = snap1389
		d33 = snap1390
		d34 = snap1391
		d35 = snap1392
		d36 = snap1393
		d37 = snap1394
		d38 = snap1395
		d39 = snap1396
		d40 = snap1397
		d41 = snap1398
		d42 = snap1399
		d43 = snap1400
		d44 = snap1401
		d45 = snap1402
		d46 = snap1403
		d47 = snap1404
		d48 = snap1405
		d49 = snap1406
		d50 = snap1407
		d51 = snap1408
		d52 = snap1409
		d53 = snap1410
		d54 = snap1411
		d55 = snap1412
		d56 = snap1413
		d57 = snap1414
		d60 = snap1415
		d61 = snap1416
		d62 = snap1417
		d177 = snap1418
		d178 = snap1419
		d179 = snap1420
		d180 = snap1421
		d181 = snap1422
		d182 = snap1423
		d183 = snap1424
		d184 = snap1425
		d185 = snap1426
		d186 = snap1427
		d187 = snap1428
		d188 = snap1429
		d189 = snap1430
		d190 = snap1431
		d191 = snap1432
		d192 = snap1433
		d193 = snap1434
		d194 = snap1435
		d195 = snap1436
		d196 = snap1437
		d197 = snap1438
		d198 = snap1439
		d199 = snap1440
		d200 = snap1441
		d201 = snap1442
		d202 = snap1443
		d203 = snap1444
		d204 = snap1445
		d205 = snap1446
		d206 = snap1447
		d209 = snap1448
		d386 = snap1449
		d387 = snap1450
		d388 = snap1451
		d389 = snap1452
		d391 = snap1453
		d392 = snap1454
		d393 = snap1455
		d394 = snap1456
		d395 = snap1457
		d396 = snap1458
		d397 = snap1459
		d398 = snap1460
		d400 = snap1461
		d402 = snap1462
		d403 = snap1463
		d404 = snap1464
		d508 = snap1465
		d509 = snap1466
		d512 = snap1467
		d619 = snap1468
		d620 = snap1469
		d621 = snap1470
		d622 = snap1471
		d623 = snap1472
		d625 = snap1473
		d626 = snap1474
		d627 = snap1475
		d628 = snap1476
		d629 = snap1477
		d630 = snap1478
		d631 = snap1479
		d632 = snap1480
		d633 = snap1481
		d634 = snap1482
		d635 = snap1483
		d636 = snap1484
		d637 = snap1485
		d638 = snap1486
		d639 = snap1487
		d640 = snap1488
		d641 = snap1489
		d642 = snap1490
		d643 = snap1491
		d644 = snap1492
		d645 = snap1493
		d646 = snap1494
		d647 = snap1495
		d648 = snap1496
		d649 = snap1497
		d650 = snap1498
		d651 = snap1499
		d652 = snap1500
		d653 = snap1501
		d654 = snap1502
		d655 = snap1503
		d944 = snap1504
		d945 = snap1505
		d946 = snap1506
		d948 = snap1507
		d949 = snap1508
		d950 = snap1509
		d951 = snap1510
		d952 = snap1511
		d953 = snap1512
		d954 = snap1513
		d956 = snap1514
		d958 = snap1515
		d959 = snap1516
		d1115 = snap1517
		d1116 = snap1518
		d1119 = snap1519
		d1278 = snap1520
		d1279 = snap1521
		d1280 = snap1522
		d1281 = snap1523
		d1283 = snap1524
		d1284 = snap1525
		d1285 = snap1526
		d1286 = snap1527
		d1287 = snap1528
		d1288 = snap1529
		d1289 = snap1530
		d1290 = snap1531
		d1291 = snap1532
		d1292 = snap1533
		d1294 = snap1534
		d1295 = snap1535
		d1296 = snap1536
		d1297 = snap1537
		d1298 = snap1538
		d1299 = snap1539
		d1300 = snap1540
		d1301 = snap1541
		d1302 = snap1542
		d1303 = snap1543
		d1304 = snap1544
		d1305 = snap1545
		d1306 = snap1546
		d1307 = snap1547
		d1308 = snap1548
		d1309 = snap1549
		d1310 = snap1550
		d1311 = snap1551
		d1312 = snap1552
		d1313 = snap1553
		d1314 = snap1554
		d1315 = snap1555
		d1316 = snap1556
		d1317 = snap1557
		d1318 = snap1558
		d1319 = snap1559
		d1320 = snap1560
		d1321 = snap1561
		d1322 = snap1562
		d1323 = snap1563
		d1324 = snap1564
		d1325 = snap1565
		d1326 = snap1566
		d1327 = snap1567
		d1328 = snap1568
		d1329 = snap1569
		d1330 = snap1570
		d1331 = snap1571
		d1332 = snap1572
		d1333 = snap1573
		d1334 = snap1574
		d1335 = snap1575
		d1336 = snap1576
		d1337 = snap1577
		d1338 = snap1578
		d1339 = snap1579
		d1340 = snap1580
		d1341 = snap1581
		d1342 = snap1582
		d1343 = snap1583
		d1344 = snap1584
		d1345 = snap1585
		d1346 = snap1586
		d1347 = snap1587
		d1348 = snap1588
		d1349 = snap1589
		d1350 = snap1590
		d1351 = snap1591
		d1352 = snap1592
		d1353 = snap1593
		d1354 = snap1594
		d1355 = snap1595
		d1356 = snap1596
		d1357 = snap1597
		d1358 = snap1598
		d1359 = snap1599
		d1360 = snap1600
		ps1602 := scm.PhiState{General: true}
		ps1602.OverlayValues = make([]scm.JITValueDesc, 1361)
		ps1602.OverlayValues[5] = d5
		ps1602.OverlayValues[6] = d6
		ps1602.OverlayValues[7] = d7
		ps1602.OverlayValues[8] = d8
		ps1602.OverlayValues[9] = d9
		ps1602.OverlayValues[10] = d10
		ps1602.OverlayValues[11] = d11
		ps1602.OverlayValues[12] = d12
		ps1602.OverlayValues[13] = d13
		ps1602.OverlayValues[14] = d14
		ps1602.OverlayValues[15] = d15
		ps1602.OverlayValues[16] = d16
		ps1602.OverlayValues[17] = d17
		ps1602.OverlayValues[18] = d18
		ps1602.OverlayValues[19] = d19
		ps1602.OverlayValues[20] = d20
		ps1602.OverlayValues[21] = d21
		ps1602.OverlayValues[23] = d23
		ps1602.OverlayValues[24] = d24
		ps1602.OverlayValues[25] = d25
		ps1602.OverlayValues[26] = d26
		ps1602.OverlayValues[27] = d27
		ps1602.OverlayValues[28] = d28
		ps1602.OverlayValues[29] = d29
		ps1602.OverlayValues[30] = d30
		ps1602.OverlayValues[31] = d31
		ps1602.OverlayValues[32] = d32
		ps1602.OverlayValues[33] = d33
		ps1602.OverlayValues[34] = d34
		ps1602.OverlayValues[35] = d35
		ps1602.OverlayValues[36] = d36
		ps1602.OverlayValues[37] = d37
		ps1602.OverlayValues[38] = d38
		ps1602.OverlayValues[39] = d39
		ps1602.OverlayValues[40] = d40
		ps1602.OverlayValues[41] = d41
		ps1602.OverlayValues[42] = d42
		ps1602.OverlayValues[43] = d43
		ps1602.OverlayValues[44] = d44
		ps1602.OverlayValues[45] = d45
		ps1602.OverlayValues[46] = d46
		ps1602.OverlayValues[47] = d47
		ps1602.OverlayValues[48] = d48
		ps1602.OverlayValues[49] = d49
		ps1602.OverlayValues[50] = d50
		ps1602.OverlayValues[51] = d51
		ps1602.OverlayValues[52] = d52
		ps1602.OverlayValues[53] = d53
		ps1602.OverlayValues[54] = d54
		ps1602.OverlayValues[55] = d55
		ps1602.OverlayValues[56] = d56
		ps1602.OverlayValues[57] = d57
		ps1602.OverlayValues[60] = d60
		ps1602.OverlayValues[61] = d61
		ps1602.OverlayValues[62] = d62
		ps1602.OverlayValues[177] = d177
		ps1602.OverlayValues[178] = d178
		ps1602.OverlayValues[179] = d179
		ps1602.OverlayValues[180] = d180
		ps1602.OverlayValues[181] = d181
		ps1602.OverlayValues[182] = d182
		ps1602.OverlayValues[183] = d183
		ps1602.OverlayValues[184] = d184
		ps1602.OverlayValues[185] = d185
		ps1602.OverlayValues[186] = d186
		ps1602.OverlayValues[187] = d187
		ps1602.OverlayValues[188] = d188
		ps1602.OverlayValues[189] = d189
		ps1602.OverlayValues[190] = d190
		ps1602.OverlayValues[191] = d191
		ps1602.OverlayValues[192] = d192
		ps1602.OverlayValues[193] = d193
		ps1602.OverlayValues[194] = d194
		ps1602.OverlayValues[195] = d195
		ps1602.OverlayValues[196] = d196
		ps1602.OverlayValues[197] = d197
		ps1602.OverlayValues[198] = d198
		ps1602.OverlayValues[199] = d199
		ps1602.OverlayValues[200] = d200
		ps1602.OverlayValues[201] = d201
		ps1602.OverlayValues[202] = d202
		ps1602.OverlayValues[203] = d203
		ps1602.OverlayValues[204] = d204
		ps1602.OverlayValues[205] = d205
		ps1602.OverlayValues[206] = d206
		ps1602.OverlayValues[209] = d209
		ps1602.OverlayValues[386] = d386
		ps1602.OverlayValues[387] = d387
		ps1602.OverlayValues[388] = d388
		ps1602.OverlayValues[389] = d389
		ps1602.OverlayValues[391] = d391
		ps1602.OverlayValues[392] = d392
		ps1602.OverlayValues[393] = d393
		ps1602.OverlayValues[394] = d394
		ps1602.OverlayValues[395] = d395
		ps1602.OverlayValues[396] = d396
		ps1602.OverlayValues[397] = d397
		ps1602.OverlayValues[398] = d398
		ps1602.OverlayValues[400] = d400
		ps1602.OverlayValues[402] = d402
		ps1602.OverlayValues[403] = d403
		ps1602.OverlayValues[404] = d404
		ps1602.OverlayValues[508] = d508
		ps1602.OverlayValues[509] = d509
		ps1602.OverlayValues[512] = d512
		ps1602.OverlayValues[619] = d619
		ps1602.OverlayValues[620] = d620
		ps1602.OverlayValues[621] = d621
		ps1602.OverlayValues[622] = d622
		ps1602.OverlayValues[623] = d623
		ps1602.OverlayValues[625] = d625
		ps1602.OverlayValues[626] = d626
		ps1602.OverlayValues[627] = d627
		ps1602.OverlayValues[628] = d628
		ps1602.OverlayValues[629] = d629
		ps1602.OverlayValues[630] = d630
		ps1602.OverlayValues[631] = d631
		ps1602.OverlayValues[632] = d632
		ps1602.OverlayValues[633] = d633
		ps1602.OverlayValues[634] = d634
		ps1602.OverlayValues[635] = d635
		ps1602.OverlayValues[636] = d636
		ps1602.OverlayValues[637] = d637
		ps1602.OverlayValues[638] = d638
		ps1602.OverlayValues[639] = d639
		ps1602.OverlayValues[640] = d640
		ps1602.OverlayValues[641] = d641
		ps1602.OverlayValues[642] = d642
		ps1602.OverlayValues[643] = d643
		ps1602.OverlayValues[644] = d644
		ps1602.OverlayValues[645] = d645
		ps1602.OverlayValues[646] = d646
		ps1602.OverlayValues[647] = d647
		ps1602.OverlayValues[648] = d648
		ps1602.OverlayValues[649] = d649
		ps1602.OverlayValues[650] = d650
		ps1602.OverlayValues[651] = d651
		ps1602.OverlayValues[652] = d652
		ps1602.OverlayValues[653] = d653
		ps1602.OverlayValues[654] = d654
		ps1602.OverlayValues[655] = d655
		ps1602.OverlayValues[944] = d944
		ps1602.OverlayValues[945] = d945
		ps1602.OverlayValues[946] = d946
		ps1602.OverlayValues[948] = d948
		ps1602.OverlayValues[949] = d949
		ps1602.OverlayValues[950] = d950
		ps1602.OverlayValues[951] = d951
		ps1602.OverlayValues[952] = d952
		ps1602.OverlayValues[953] = d953
		ps1602.OverlayValues[954] = d954
		ps1602.OverlayValues[956] = d956
		ps1602.OverlayValues[958] = d958
		ps1602.OverlayValues[959] = d959
		ps1602.OverlayValues[1115] = d1115
		ps1602.OverlayValues[1116] = d1116
		ps1602.OverlayValues[1119] = d1119
		ps1602.OverlayValues[1278] = d1278
		ps1602.OverlayValues[1279] = d1279
		ps1602.OverlayValues[1280] = d1280
		ps1602.OverlayValues[1281] = d1281
		ps1602.OverlayValues[1283] = d1283
		ps1602.OverlayValues[1284] = d1284
		ps1602.OverlayValues[1285] = d1285
		ps1602.OverlayValues[1286] = d1286
		ps1602.OverlayValues[1287] = d1287
		ps1602.OverlayValues[1288] = d1288
		ps1602.OverlayValues[1289] = d1289
		ps1602.OverlayValues[1290] = d1290
		ps1602.OverlayValues[1291] = d1291
		ps1602.OverlayValues[1292] = d1292
		ps1602.OverlayValues[1294] = d1294
		ps1602.OverlayValues[1295] = d1295
		ps1602.OverlayValues[1296] = d1296
		ps1602.OverlayValues[1297] = d1297
		ps1602.OverlayValues[1298] = d1298
		ps1602.OverlayValues[1299] = d1299
		ps1602.OverlayValues[1300] = d1300
		ps1602.OverlayValues[1301] = d1301
		ps1602.OverlayValues[1302] = d1302
		ps1602.OverlayValues[1303] = d1303
		ps1602.OverlayValues[1304] = d1304
		ps1602.OverlayValues[1305] = d1305
		ps1602.OverlayValues[1306] = d1306
		ps1602.OverlayValues[1307] = d1307
		ps1602.OverlayValues[1308] = d1308
		ps1602.OverlayValues[1309] = d1309
		ps1602.OverlayValues[1310] = d1310
		ps1602.OverlayValues[1311] = d1311
		ps1602.OverlayValues[1312] = d1312
		ps1602.OverlayValues[1313] = d1313
		ps1602.OverlayValues[1314] = d1314
		ps1602.OverlayValues[1315] = d1315
		ps1602.OverlayValues[1316] = d1316
		ps1602.OverlayValues[1317] = d1317
		ps1602.OverlayValues[1318] = d1318
		ps1602.OverlayValues[1319] = d1319
		ps1602.OverlayValues[1320] = d1320
		ps1602.OverlayValues[1321] = d1321
		ps1602.OverlayValues[1322] = d1322
		ps1602.OverlayValues[1323] = d1323
		ps1602.OverlayValues[1324] = d1324
		ps1602.OverlayValues[1325] = d1325
		ps1602.OverlayValues[1326] = d1326
		ps1602.OverlayValues[1327] = d1327
		ps1602.OverlayValues[1328] = d1328
		ps1602.OverlayValues[1329] = d1329
		ps1602.OverlayValues[1330] = d1330
		ps1602.OverlayValues[1331] = d1331
		ps1602.OverlayValues[1332] = d1332
		ps1602.OverlayValues[1333] = d1333
		ps1602.OverlayValues[1334] = d1334
		ps1602.OverlayValues[1335] = d1335
		ps1602.OverlayValues[1336] = d1336
		ps1602.OverlayValues[1337] = d1337
		ps1602.OverlayValues[1338] = d1338
		ps1602.OverlayValues[1339] = d1339
		ps1602.OverlayValues[1340] = d1340
		ps1602.OverlayValues[1341] = d1341
		ps1602.OverlayValues[1342] = d1342
		ps1602.OverlayValues[1343] = d1343
		ps1602.OverlayValues[1344] = d1344
		ps1602.OverlayValues[1345] = d1345
		ps1602.OverlayValues[1346] = d1346
		ps1602.OverlayValues[1347] = d1347
		ps1602.OverlayValues[1348] = d1348
		ps1602.OverlayValues[1349] = d1349
		ps1602.OverlayValues[1350] = d1350
		ps1602.OverlayValues[1351] = d1351
		ps1602.OverlayValues[1352] = d1352
		ps1602.OverlayValues[1353] = d1353
		ps1602.OverlayValues[1354] = d1354
		ps1602.OverlayValues[1355] = d1355
		ps1602.OverlayValues[1356] = d1356
		ps1602.OverlayValues[1357] = d1357
		ps1602.OverlayValues[1358] = d1358
		ps1602.OverlayValues[1359] = d1359
		ps1602.OverlayValues[1360] = d1360
		ps1603 := scm.PhiState{General: true}
		ps1603.OverlayValues = make([]scm.JITValueDesc, 1361)
		ps1603.OverlayValues[5] = d5
		ps1603.OverlayValues[6] = d6
		ps1603.OverlayValues[7] = d7
		ps1603.OverlayValues[8] = d8
		ps1603.OverlayValues[9] = d9
		ps1603.OverlayValues[10] = d10
		ps1603.OverlayValues[11] = d11
		ps1603.OverlayValues[12] = d12
		ps1603.OverlayValues[13] = d13
		ps1603.OverlayValues[14] = d14
		ps1603.OverlayValues[15] = d15
		ps1603.OverlayValues[16] = d16
		ps1603.OverlayValues[17] = d17
		ps1603.OverlayValues[18] = d18
		ps1603.OverlayValues[19] = d19
		ps1603.OverlayValues[20] = d20
		ps1603.OverlayValues[21] = d21
		ps1603.OverlayValues[23] = d23
		ps1603.OverlayValues[24] = d24
		ps1603.OverlayValues[25] = d25
		ps1603.OverlayValues[26] = d26
		ps1603.OverlayValues[27] = d27
		ps1603.OverlayValues[28] = d28
		ps1603.OverlayValues[29] = d29
		ps1603.OverlayValues[30] = d30
		ps1603.OverlayValues[31] = d31
		ps1603.OverlayValues[32] = d32
		ps1603.OverlayValues[33] = d33
		ps1603.OverlayValues[34] = d34
		ps1603.OverlayValues[35] = d35
		ps1603.OverlayValues[36] = d36
		ps1603.OverlayValues[37] = d37
		ps1603.OverlayValues[38] = d38
		ps1603.OverlayValues[39] = d39
		ps1603.OverlayValues[40] = d40
		ps1603.OverlayValues[41] = d41
		ps1603.OverlayValues[42] = d42
		ps1603.OverlayValues[43] = d43
		ps1603.OverlayValues[44] = d44
		ps1603.OverlayValues[45] = d45
		ps1603.OverlayValues[46] = d46
		ps1603.OverlayValues[47] = d47
		ps1603.OverlayValues[48] = d48
		ps1603.OverlayValues[49] = d49
		ps1603.OverlayValues[50] = d50
		ps1603.OverlayValues[51] = d51
		ps1603.OverlayValues[52] = d52
		ps1603.OverlayValues[53] = d53
		ps1603.OverlayValues[54] = d54
		ps1603.OverlayValues[55] = d55
		ps1603.OverlayValues[56] = d56
		ps1603.OverlayValues[57] = d57
		ps1603.OverlayValues[60] = d60
		ps1603.OverlayValues[61] = d61
		ps1603.OverlayValues[62] = d62
		ps1603.OverlayValues[177] = d177
		ps1603.OverlayValues[178] = d178
		ps1603.OverlayValues[179] = d179
		ps1603.OverlayValues[180] = d180
		ps1603.OverlayValues[181] = d181
		ps1603.OverlayValues[182] = d182
		ps1603.OverlayValues[183] = d183
		ps1603.OverlayValues[184] = d184
		ps1603.OverlayValues[185] = d185
		ps1603.OverlayValues[186] = d186
		ps1603.OverlayValues[187] = d187
		ps1603.OverlayValues[188] = d188
		ps1603.OverlayValues[189] = d189
		ps1603.OverlayValues[190] = d190
		ps1603.OverlayValues[191] = d191
		ps1603.OverlayValues[192] = d192
		ps1603.OverlayValues[193] = d193
		ps1603.OverlayValues[194] = d194
		ps1603.OverlayValues[195] = d195
		ps1603.OverlayValues[196] = d196
		ps1603.OverlayValues[197] = d197
		ps1603.OverlayValues[198] = d198
		ps1603.OverlayValues[199] = d199
		ps1603.OverlayValues[200] = d200
		ps1603.OverlayValues[201] = d201
		ps1603.OverlayValues[202] = d202
		ps1603.OverlayValues[203] = d203
		ps1603.OverlayValues[204] = d204
		ps1603.OverlayValues[205] = d205
		ps1603.OverlayValues[206] = d206
		ps1603.OverlayValues[209] = d209
		ps1603.OverlayValues[386] = d386
		ps1603.OverlayValues[387] = d387
		ps1603.OverlayValues[388] = d388
		ps1603.OverlayValues[389] = d389
		ps1603.OverlayValues[391] = d391
		ps1603.OverlayValues[392] = d392
		ps1603.OverlayValues[393] = d393
		ps1603.OverlayValues[394] = d394
		ps1603.OverlayValues[395] = d395
		ps1603.OverlayValues[396] = d396
		ps1603.OverlayValues[397] = d397
		ps1603.OverlayValues[398] = d398
		ps1603.OverlayValues[400] = d400
		ps1603.OverlayValues[402] = d402
		ps1603.OverlayValues[403] = d403
		ps1603.OverlayValues[404] = d404
		ps1603.OverlayValues[508] = d508
		ps1603.OverlayValues[509] = d509
		ps1603.OverlayValues[512] = d512
		ps1603.OverlayValues[619] = d619
		ps1603.OverlayValues[620] = d620
		ps1603.OverlayValues[621] = d621
		ps1603.OverlayValues[622] = d622
		ps1603.OverlayValues[623] = d623
		ps1603.OverlayValues[625] = d625
		ps1603.OverlayValues[626] = d626
		ps1603.OverlayValues[627] = d627
		ps1603.OverlayValues[628] = d628
		ps1603.OverlayValues[629] = d629
		ps1603.OverlayValues[630] = d630
		ps1603.OverlayValues[631] = d631
		ps1603.OverlayValues[632] = d632
		ps1603.OverlayValues[633] = d633
		ps1603.OverlayValues[634] = d634
		ps1603.OverlayValues[635] = d635
		ps1603.OverlayValues[636] = d636
		ps1603.OverlayValues[637] = d637
		ps1603.OverlayValues[638] = d638
		ps1603.OverlayValues[639] = d639
		ps1603.OverlayValues[640] = d640
		ps1603.OverlayValues[641] = d641
		ps1603.OverlayValues[642] = d642
		ps1603.OverlayValues[643] = d643
		ps1603.OverlayValues[644] = d644
		ps1603.OverlayValues[645] = d645
		ps1603.OverlayValues[646] = d646
		ps1603.OverlayValues[647] = d647
		ps1603.OverlayValues[648] = d648
		ps1603.OverlayValues[649] = d649
		ps1603.OverlayValues[650] = d650
		ps1603.OverlayValues[651] = d651
		ps1603.OverlayValues[652] = d652
		ps1603.OverlayValues[653] = d653
		ps1603.OverlayValues[654] = d654
		ps1603.OverlayValues[655] = d655
		ps1603.OverlayValues[944] = d944
		ps1603.OverlayValues[945] = d945
		ps1603.OverlayValues[946] = d946
		ps1603.OverlayValues[948] = d948
		ps1603.OverlayValues[949] = d949
		ps1603.OverlayValues[950] = d950
		ps1603.OverlayValues[951] = d951
		ps1603.OverlayValues[952] = d952
		ps1603.OverlayValues[953] = d953
		ps1603.OverlayValues[954] = d954
		ps1603.OverlayValues[956] = d956
		ps1603.OverlayValues[958] = d958
		ps1603.OverlayValues[959] = d959
		ps1603.OverlayValues[1115] = d1115
		ps1603.OverlayValues[1116] = d1116
		ps1603.OverlayValues[1119] = d1119
		ps1603.OverlayValues[1278] = d1278
		ps1603.OverlayValues[1279] = d1279
		ps1603.OverlayValues[1280] = d1280
		ps1603.OverlayValues[1281] = d1281
		ps1603.OverlayValues[1283] = d1283
		ps1603.OverlayValues[1284] = d1284
		ps1603.OverlayValues[1285] = d1285
		ps1603.OverlayValues[1286] = d1286
		ps1603.OverlayValues[1287] = d1287
		ps1603.OverlayValues[1288] = d1288
		ps1603.OverlayValues[1289] = d1289
		ps1603.OverlayValues[1290] = d1290
		ps1603.OverlayValues[1291] = d1291
		ps1603.OverlayValues[1292] = d1292
		ps1603.OverlayValues[1294] = d1294
		ps1603.OverlayValues[1295] = d1295
		ps1603.OverlayValues[1296] = d1296
		ps1603.OverlayValues[1297] = d1297
		ps1603.OverlayValues[1298] = d1298
		ps1603.OverlayValues[1299] = d1299
		ps1603.OverlayValues[1300] = d1300
		ps1603.OverlayValues[1301] = d1301
		ps1603.OverlayValues[1302] = d1302
		ps1603.OverlayValues[1303] = d1303
		ps1603.OverlayValues[1304] = d1304
		ps1603.OverlayValues[1305] = d1305
		ps1603.OverlayValues[1306] = d1306
		ps1603.OverlayValues[1307] = d1307
		ps1603.OverlayValues[1308] = d1308
		ps1603.OverlayValues[1309] = d1309
		ps1603.OverlayValues[1310] = d1310
		ps1603.OverlayValues[1311] = d1311
		ps1603.OverlayValues[1312] = d1312
		ps1603.OverlayValues[1313] = d1313
		ps1603.OverlayValues[1314] = d1314
		ps1603.OverlayValues[1315] = d1315
		ps1603.OverlayValues[1316] = d1316
		ps1603.OverlayValues[1317] = d1317
		ps1603.OverlayValues[1318] = d1318
		ps1603.OverlayValues[1319] = d1319
		ps1603.OverlayValues[1320] = d1320
		ps1603.OverlayValues[1321] = d1321
		ps1603.OverlayValues[1322] = d1322
		ps1603.OverlayValues[1323] = d1323
		ps1603.OverlayValues[1324] = d1324
		ps1603.OverlayValues[1325] = d1325
		ps1603.OverlayValues[1326] = d1326
		ps1603.OverlayValues[1327] = d1327
		ps1603.OverlayValues[1328] = d1328
		ps1603.OverlayValues[1329] = d1329
		ps1603.OverlayValues[1330] = d1330
		ps1603.OverlayValues[1331] = d1331
		ps1603.OverlayValues[1332] = d1332
		ps1603.OverlayValues[1333] = d1333
		ps1603.OverlayValues[1334] = d1334
		ps1603.OverlayValues[1335] = d1335
		ps1603.OverlayValues[1336] = d1336
		ps1603.OverlayValues[1337] = d1337
		ps1603.OverlayValues[1338] = d1338
		ps1603.OverlayValues[1339] = d1339
		ps1603.OverlayValues[1340] = d1340
		ps1603.OverlayValues[1341] = d1341
		ps1603.OverlayValues[1342] = d1342
		ps1603.OverlayValues[1343] = d1343
		ps1603.OverlayValues[1344] = d1344
		ps1603.OverlayValues[1345] = d1345
		ps1603.OverlayValues[1346] = d1346
		ps1603.OverlayValues[1347] = d1347
		ps1603.OverlayValues[1348] = d1348
		ps1603.OverlayValues[1349] = d1349
		ps1603.OverlayValues[1350] = d1350
		ps1603.OverlayValues[1351] = d1351
		ps1603.OverlayValues[1352] = d1352
		ps1603.OverlayValues[1353] = d1353
		ps1603.OverlayValues[1354] = d1354
		ps1603.OverlayValues[1355] = d1355
		ps1603.OverlayValues[1356] = d1356
		ps1603.OverlayValues[1357] = d1357
		ps1603.OverlayValues[1358] = d1358
		ps1603.OverlayValues[1359] = d1359
		ps1603.OverlayValues[1360] = d1360
		snap1604 := d5
		snap1605 := d6
		snap1606 := d7
		snap1607 := d8
		snap1608 := d9
		snap1609 := d10
		snap1610 := d11
		snap1611 := d12
		snap1612 := d13
		snap1613 := d14
		snap1614 := d15
		snap1615 := d16
		snap1616 := d17
		snap1617 := d18
		snap1618 := d19
		snap1619 := d20
		snap1620 := d21
		snap1621 := d23
		snap1622 := d24
		snap1623 := d25
		snap1624 := d26
		snap1625 := d27
		snap1626 := d28
		snap1627 := d29
		snap1628 := d30
		snap1629 := d31
		snap1630 := d32
		snap1631 := d33
		snap1632 := d34
		snap1633 := d35
		snap1634 := d36
		snap1635 := d37
		snap1636 := d38
		snap1637 := d39
		snap1638 := d40
		snap1639 := d41
		snap1640 := d42
		snap1641 := d43
		snap1642 := d44
		snap1643 := d45
		snap1644 := d46
		snap1645 := d47
		snap1646 := d48
		snap1647 := d49
		snap1648 := d50
		snap1649 := d51
		snap1650 := d52
		snap1651 := d53
		snap1652 := d54
		snap1653 := d55
		snap1654 := d56
		snap1655 := d57
		snap1656 := d60
		snap1657 := d61
		snap1658 := d62
		snap1659 := d177
		snap1660 := d178
		snap1661 := d179
		snap1662 := d180
		snap1663 := d181
		snap1664 := d182
		snap1665 := d183
		snap1666 := d184
		snap1667 := d185
		snap1668 := d186
		snap1669 := d187
		snap1670 := d188
		snap1671 := d189
		snap1672 := d190
		snap1673 := d191
		snap1674 := d192
		snap1675 := d193
		snap1676 := d194
		snap1677 := d195
		snap1678 := d196
		snap1679 := d197
		snap1680 := d198
		snap1681 := d199
		snap1682 := d200
		snap1683 := d201
		snap1684 := d202
		snap1685 := d203
		snap1686 := d204
		snap1687 := d205
		snap1688 := d206
		snap1689 := d209
		snap1690 := d386
		snap1691 := d387
		snap1692 := d388
		snap1693 := d389
		snap1694 := d391
		snap1695 := d392
		snap1696 := d393
		snap1697 := d394
		snap1698 := d395
		snap1699 := d396
		snap1700 := d397
		snap1701 := d398
		snap1702 := d400
		snap1703 := d402
		snap1704 := d403
		snap1705 := d404
		snap1706 := d508
		snap1707 := d509
		snap1708 := d512
		snap1709 := d619
		snap1710 := d620
		snap1711 := d621
		snap1712 := d622
		snap1713 := d623
		snap1714 := d625
		snap1715 := d626
		snap1716 := d627
		snap1717 := d628
		snap1718 := d629
		snap1719 := d630
		snap1720 := d631
		snap1721 := d632
		snap1722 := d633
		snap1723 := d634
		snap1724 := d635
		snap1725 := d636
		snap1726 := d637
		snap1727 := d638
		snap1728 := d639
		snap1729 := d640
		snap1730 := d641
		snap1731 := d642
		snap1732 := d643
		snap1733 := d644
		snap1734 := d645
		snap1735 := d646
		snap1736 := d647
		snap1737 := d648
		snap1738 := d649
		snap1739 := d650
		snap1740 := d651
		snap1741 := d652
		snap1742 := d653
		snap1743 := d654
		snap1744 := d655
		snap1745 := d944
		snap1746 := d945
		snap1747 := d946
		snap1748 := d948
		snap1749 := d949
		snap1750 := d950
		snap1751 := d951
		snap1752 := d952
		snap1753 := d953
		snap1754 := d954
		snap1755 := d956
		snap1756 := d958
		snap1757 := d959
		snap1758 := d1115
		snap1759 := d1116
		snap1760 := d1119
		snap1761 := d1278
		snap1762 := d1279
		snap1763 := d1280
		snap1764 := d1281
		snap1765 := d1283
		snap1766 := d1284
		snap1767 := d1285
		snap1768 := d1286
		snap1769 := d1287
		snap1770 := d1288
		snap1771 := d1289
		snap1772 := d1290
		snap1773 := d1291
		snap1774 := d1292
		snap1775 := d1294
		snap1776 := d1295
		snap1777 := d1296
		snap1778 := d1297
		snap1779 := d1298
		snap1780 := d1299
		snap1781 := d1300
		snap1782 := d1301
		snap1783 := d1302
		snap1784 := d1303
		snap1785 := d1304
		snap1786 := d1305
		snap1787 := d1306
		snap1788 := d1307
		snap1789 := d1308
		snap1790 := d1309
		snap1791 := d1310
		snap1792 := d1311
		snap1793 := d1312
		snap1794 := d1313
		snap1795 := d1314
		snap1796 := d1315
		snap1797 := d1316
		snap1798 := d1317
		snap1799 := d1318
		snap1800 := d1319
		snap1801 := d1320
		snap1802 := d1321
		snap1803 := d1322
		snap1804 := d1323
		snap1805 := d1324
		snap1806 := d1325
		snap1807 := d1326
		snap1808 := d1327
		snap1809 := d1328
		snap1810 := d1329
		snap1811 := d1330
		snap1812 := d1331
		snap1813 := d1332
		snap1814 := d1333
		snap1815 := d1334
		snap1816 := d1335
		snap1817 := d1336
		snap1818 := d1337
		snap1819 := d1338
		snap1820 := d1339
		snap1821 := d1340
		snap1822 := d1341
		snap1823 := d1342
		snap1824 := d1343
		snap1825 := d1344
		snap1826 := d1345
		snap1827 := d1346
		snap1828 := d1347
		snap1829 := d1348
		snap1830 := d1349
		snap1831 := d1350
		snap1832 := d1351
		snap1833 := d1352
		snap1834 := d1353
		snap1835 := d1354
		snap1836 := d1355
		snap1837 := d1356
		snap1838 := d1357
		snap1839 := d1358
		snap1840 := d1359
		snap1841 := d1360
		alloc1842 := ctx.SnapshotAllocState()
		if !bbs[12].Rendered {
			bbs[12].RenderPS(ps1603)
		}
		ctx.RestoreAllocState(alloc1842)
		d5 = snap1604
		d6 = snap1605
		d7 = snap1606
		d8 = snap1607
		d9 = snap1608
		d10 = snap1609
		d11 = snap1610
		d12 = snap1611
		d13 = snap1612
		d14 = snap1613
		d15 = snap1614
		d16 = snap1615
		d17 = snap1616
		d18 = snap1617
		d19 = snap1618
		d20 = snap1619
		d21 = snap1620
		d23 = snap1621
		d24 = snap1622
		d25 = snap1623
		d26 = snap1624
		d27 = snap1625
		d28 = snap1626
		d29 = snap1627
		d30 = snap1628
		d31 = snap1629
		d32 = snap1630
		d33 = snap1631
		d34 = snap1632
		d35 = snap1633
		d36 = snap1634
		d37 = snap1635
		d38 = snap1636
		d39 = snap1637
		d40 = snap1638
		d41 = snap1639
		d42 = snap1640
		d43 = snap1641
		d44 = snap1642
		d45 = snap1643
		d46 = snap1644
		d47 = snap1645
		d48 = snap1646
		d49 = snap1647
		d50 = snap1648
		d51 = snap1649
		d52 = snap1650
		d53 = snap1651
		d54 = snap1652
		d55 = snap1653
		d56 = snap1654
		d57 = snap1655
		d60 = snap1656
		d61 = snap1657
		d62 = snap1658
		d177 = snap1659
		d178 = snap1660
		d179 = snap1661
		d180 = snap1662
		d181 = snap1663
		d182 = snap1664
		d183 = snap1665
		d184 = snap1666
		d185 = snap1667
		d186 = snap1668
		d187 = snap1669
		d188 = snap1670
		d189 = snap1671
		d190 = snap1672
		d191 = snap1673
		d192 = snap1674
		d193 = snap1675
		d194 = snap1676
		d195 = snap1677
		d196 = snap1678
		d197 = snap1679
		d198 = snap1680
		d199 = snap1681
		d200 = snap1682
		d201 = snap1683
		d202 = snap1684
		d203 = snap1685
		d204 = snap1686
		d205 = snap1687
		d206 = snap1688
		d209 = snap1689
		d386 = snap1690
		d387 = snap1691
		d388 = snap1692
		d389 = snap1693
		d391 = snap1694
		d392 = snap1695
		d393 = snap1696
		d394 = snap1697
		d395 = snap1698
		d396 = snap1699
		d397 = snap1700
		d398 = snap1701
		d400 = snap1702
		d402 = snap1703
		d403 = snap1704
		d404 = snap1705
		d508 = snap1706
		d509 = snap1707
		d512 = snap1708
		d619 = snap1709
		d620 = snap1710
		d621 = snap1711
		d622 = snap1712
		d623 = snap1713
		d625 = snap1714
		d626 = snap1715
		d627 = snap1716
		d628 = snap1717
		d629 = snap1718
		d630 = snap1719
		d631 = snap1720
		d632 = snap1721
		d633 = snap1722
		d634 = snap1723
		d635 = snap1724
		d636 = snap1725
		d637 = snap1726
		d638 = snap1727
		d639 = snap1728
		d640 = snap1729
		d641 = snap1730
		d642 = snap1731
		d643 = snap1732
		d644 = snap1733
		d645 = snap1734
		d646 = snap1735
		d647 = snap1736
		d648 = snap1737
		d649 = snap1738
		d650 = snap1739
		d651 = snap1740
		d652 = snap1741
		d653 = snap1742
		d654 = snap1743
		d655 = snap1744
		d944 = snap1745
		d945 = snap1746
		d946 = snap1747
		d948 = snap1748
		d949 = snap1749
		d950 = snap1750
		d951 = snap1751
		d952 = snap1752
		d953 = snap1753
		d954 = snap1754
		d956 = snap1755
		d958 = snap1756
		d959 = snap1757
		d1115 = snap1758
		d1116 = snap1759
		d1119 = snap1760
		d1278 = snap1761
		d1279 = snap1762
		d1280 = snap1763
		d1281 = snap1764
		d1283 = snap1765
		d1284 = snap1766
		d1285 = snap1767
		d1286 = snap1768
		d1287 = snap1769
		d1288 = snap1770
		d1289 = snap1771
		d1290 = snap1772
		d1291 = snap1773
		d1292 = snap1774
		d1294 = snap1775
		d1295 = snap1776
		d1296 = snap1777
		d1297 = snap1778
		d1298 = snap1779
		d1299 = snap1780
		d1300 = snap1781
		d1301 = snap1782
		d1302 = snap1783
		d1303 = snap1784
		d1304 = snap1785
		d1305 = snap1786
		d1306 = snap1787
		d1307 = snap1788
		d1308 = snap1789
		d1309 = snap1790
		d1310 = snap1791
		d1311 = snap1792
		d1312 = snap1793
		d1313 = snap1794
		d1314 = snap1795
		d1315 = snap1796
		d1316 = snap1797
		d1317 = snap1798
		d1318 = snap1799
		d1319 = snap1800
		d1320 = snap1801
		d1321 = snap1802
		d1322 = snap1803
		d1323 = snap1804
		d1324 = snap1805
		d1325 = snap1806
		d1326 = snap1807
		d1327 = snap1808
		d1328 = snap1809
		d1329 = snap1810
		d1330 = snap1811
		d1331 = snap1812
		d1332 = snap1813
		d1333 = snap1814
		d1334 = snap1815
		d1335 = snap1816
		d1336 = snap1817
		d1337 = snap1818
		d1338 = snap1819
		d1339 = snap1820
		d1340 = snap1821
		d1341 = snap1822
		d1342 = snap1823
		d1343 = snap1824
		d1344 = snap1825
		d1345 = snap1826
		d1346 = snap1827
		d1347 = snap1828
		d1348 = snap1829
		d1349 = snap1830
		d1350 = snap1831
		d1351 = snap1832
		d1352 = snap1833
		d1353 = snap1834
		d1354 = snap1835
		d1355 = snap1836
		d1356 = snap1837
		d1357 = snap1838
		d1358 = snap1839
		d1359 = snap1840
		d1360 = snap1841
		if !bbs[11].Rendered {
			return bbs[11].RenderPS(ps1602)
		}
		return result
		return result
	}
	ps1843 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps1843)
	ctx.MarkLabel(lbl0)
	d1844 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r3, Reg2: r4}
	ctx.BindReg(r3, &d1844)
	ctx.BindReg(r4, &d1844)
	ctx.EmitMovPairToResult(&d1844, &result)
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
