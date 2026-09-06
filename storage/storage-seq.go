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
	var d176 scm.JITValueDesc
	_ = d176
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
	var d196 scm.JITValueDesc
	_ = d196
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
	var d381 scm.JITValueDesc
	_ = d381
	var d383 scm.JITValueDesc
	_ = d383
	var d384 scm.JITValueDesc
	_ = d384
	var d385 scm.JITValueDesc
	_ = d385
	var d486 scm.JITValueDesc
	_ = d486
	var d487 scm.JITValueDesc
	_ = d487
	var d490 scm.JITValueDesc
	_ = d490
	var d594 scm.JITValueDesc
	_ = d594
	var d595 scm.JITValueDesc
	_ = d595
	var d596 scm.JITValueDesc
	_ = d596
	var d597 scm.JITValueDesc
	_ = d597
	var d598 scm.JITValueDesc
	_ = d598
	var d600 scm.JITValueDesc
	_ = d600
	var d601 scm.JITValueDesc
	_ = d601
	var d602 scm.JITValueDesc
	_ = d602
	var d603 scm.JITValueDesc
	_ = d603
	var d604 scm.JITValueDesc
	_ = d604
	var d605 scm.JITValueDesc
	_ = d605
	var d606 scm.JITValueDesc
	_ = d606
	var d607 scm.JITValueDesc
	_ = d607
	var d608 scm.JITValueDesc
	_ = d608
	var d609 scm.JITValueDesc
	_ = d609
	var d610 scm.JITValueDesc
	_ = d610
	var d611 scm.JITValueDesc
	_ = d611
	var d612 scm.JITValueDesc
	_ = d612
	var d613 scm.JITValueDesc
	_ = d613
	var d614 scm.JITValueDesc
	_ = d614
	var d615 scm.JITValueDesc
	_ = d615
	var d616 scm.JITValueDesc
	_ = d616
	var d617 scm.JITValueDesc
	_ = d617
	var d618 scm.JITValueDesc
	_ = d618
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
	var d624 scm.JITValueDesc
	_ = d624
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
	var d913 scm.JITValueDesc
	_ = d913
	var d914 scm.JITValueDesc
	_ = d914
	var d915 scm.JITValueDesc
	_ = d915
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
	var d925 scm.JITValueDesc
	_ = d925
	var d927 scm.JITValueDesc
	_ = d927
	var d928 scm.JITValueDesc
	_ = d928
	var d1081 scm.JITValueDesc
	_ = d1081
	var d1082 scm.JITValueDesc
	_ = d1082
	var d1085 scm.JITValueDesc
	_ = d1085
	var d1241 scm.JITValueDesc
	_ = d1241
	var d1242 scm.JITValueDesc
	_ = d1242
	var d1243 scm.JITValueDesc
	_ = d1243
	var d1244 scm.JITValueDesc
	_ = d1244
	var d1246 scm.JITValueDesc
	_ = d1246
	var d1247 scm.JITValueDesc
	_ = d1247
	var d1248 scm.JITValueDesc
	_ = d1248
	var d1249 scm.JITValueDesc
	_ = d1249
	var d1250 scm.JITValueDesc
	_ = d1250
	var d1251 scm.JITValueDesc
	_ = d1251
	var d1252 scm.JITValueDesc
	_ = d1252
	var d1253 scm.JITValueDesc
	_ = d1253
	var d1255 scm.JITValueDesc
	_ = d1255
	var d1256 scm.JITValueDesc
	_ = d1256
	var d1257 scm.JITValueDesc
	_ = d1257
	var d1258 scm.JITValueDesc
	_ = d1258
	var d1259 scm.JITValueDesc
	_ = d1259
	var d1260 scm.JITValueDesc
	_ = d1260
	var d1261 scm.JITValueDesc
	_ = d1261
	var d1262 scm.JITValueDesc
	_ = d1262
	var d1263 scm.JITValueDesc
	_ = d1263
	var d1264 scm.JITValueDesc
	_ = d1264
	var d1265 scm.JITValueDesc
	_ = d1265
	var d1266 scm.JITValueDesc
	_ = d1266
	var d1267 scm.JITValueDesc
	_ = d1267
	var d1268 scm.JITValueDesc
	_ = d1268
	var d1269 scm.JITValueDesc
	_ = d1269
	var d1270 scm.JITValueDesc
	_ = d1270
	var d1271 scm.JITValueDesc
	_ = d1271
	var d1272 scm.JITValueDesc
	_ = d1272
	var d1273 scm.JITValueDesc
	_ = d1273
	var d1274 scm.JITValueDesc
	_ = d1274
	var d1275 scm.JITValueDesc
	_ = d1275
	var d1276 scm.JITValueDesc
	_ = d1276
	var d1277 scm.JITValueDesc
	_ = d1277
	var d1278 scm.JITValueDesc
	_ = d1278
	var d1279 scm.JITValueDesc
	_ = d1279
	var d1280 scm.JITValueDesc
	_ = d1280
	var d1281 scm.JITValueDesc
	_ = d1281
	var d1282 scm.JITValueDesc
	_ = d1282
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
	var d1293 scm.JITValueDesc
	_ = d1293
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
		snap56 := d1
		snap57 := d2
		snap58 := d3
		snap59 := d4
		snap60 := d5
		snap61 := d6
		snap62 := d7
		snap63 := d8
		snap64 := d9
		snap65 := d10
		snap66 := d11
		snap67 := d12
		snap68 := d13
		snap69 := d14
		snap70 := d15
		snap71 := d17
		snap72 := d18
		snap73 := d19
		snap74 := d20
		snap75 := d21
		snap76 := d22
		snap77 := d23
		snap78 := d24
		snap79 := d25
		snap80 := d26
		snap81 := d27
		snap82 := d28
		snap83 := d29
		snap84 := d30
		snap85 := d31
		snap86 := d32
		snap87 := d33
		snap88 := d34
		snap89 := d35
		snap90 := d36
		snap91 := d37
		snap92 := d38
		snap93 := d39
		snap94 := d40
		snap95 := d41
		snap96 := d42
		snap97 := d43
		snap98 := d44
		snap99 := d45
		snap100 := d46
		snap101 := d47
		snap102 := d48
		snap103 := d49
		snap104 := d50
		snap105 := d53
		snap106 := d54
		snap107 := d55
		alloc108 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl16)
		ctx.EmitJmp(lbl4)
		ctx.RestoreAllocState(alloc108)
		d1 = snap56
		d2 = snap57
		d3 = snap58
		d4 = snap59
		d5 = snap60
		d6 = snap61
		d7 = snap62
		d8 = snap63
		d9 = snap64
		d10 = snap65
		d11 = snap66
		d12 = snap67
		d13 = snap68
		d14 = snap69
		d15 = snap70
		d17 = snap71
		d18 = snap72
		d19 = snap73
		d20 = snap74
		d21 = snap75
		d22 = snap76
		d23 = snap77
		d24 = snap78
		d25 = snap79
		d26 = snap80
		d27 = snap81
		d28 = snap82
		d29 = snap83
		d30 = snap84
		d31 = snap85
		d32 = snap86
		d33 = snap87
		d34 = snap88
		d35 = snap89
		d36 = snap90
		d37 = snap91
		d38 = snap92
		d39 = snap93
		d40 = snap94
		d41 = snap95
		d42 = snap96
		d43 = snap97
		d44 = snap98
		d45 = snap99
		d46 = snap100
		d47 = snap101
		d48 = snap102
		d49 = snap103
		d50 = snap104
		d53 = snap105
		d54 = snap106
		d55 = snap107
		ctx.MarkLabel(lbl17)
		ctx.EmitJmp(lbl6)
		ctx.RestoreAllocState(alloc108)
		d1 = snap56
		d2 = snap57
		d3 = snap58
		d4 = snap59
		d5 = snap60
		d6 = snap61
		d7 = snap62
		d8 = snap63
		d9 = snap64
		d10 = snap65
		d11 = snap66
		d12 = snap67
		d13 = snap68
		d14 = snap69
		d15 = snap70
		d17 = snap71
		d18 = snap72
		d19 = snap73
		d20 = snap74
		d21 = snap75
		d22 = snap76
		d23 = snap77
		d24 = snap78
		d25 = snap79
		d26 = snap80
		d27 = snap81
		d28 = snap82
		d29 = snap83
		d30 = snap84
		d31 = snap85
		d32 = snap86
		d33 = snap87
		d34 = snap88
		d35 = snap89
		d36 = snap90
		d37 = snap91
		d38 = snap92
		d39 = snap93
		d40 = snap94
		d41 = snap95
		d42 = snap96
		d43 = snap97
		d44 = snap98
		d45 = snap99
		d46 = snap100
		d47 = snap101
		d48 = snap102
		d49 = snap103
		d50 = snap104
		d53 = snap105
		d54 = snap106
		d55 = snap107
		ps109 := scm.PhiState{General: true}
		ps109.OverlayValues = make([]scm.JITValueDesc, 56)
		ps109.OverlayValues[1] = d1
		ps109.OverlayValues[2] = d2
		ps109.OverlayValues[3] = d3
		ps109.OverlayValues[4] = d4
		ps109.OverlayValues[5] = d5
		ps109.OverlayValues[6] = d6
		ps109.OverlayValues[7] = d7
		ps109.OverlayValues[8] = d8
		ps109.OverlayValues[9] = d9
		ps109.OverlayValues[10] = d10
		ps109.OverlayValues[11] = d11
		ps109.OverlayValues[12] = d12
		ps109.OverlayValues[13] = d13
		ps109.OverlayValues[14] = d14
		ps109.OverlayValues[15] = d15
		ps109.OverlayValues[17] = d17
		ps109.OverlayValues[18] = d18
		ps109.OverlayValues[19] = d19
		ps109.OverlayValues[20] = d20
		ps109.OverlayValues[21] = d21
		ps109.OverlayValues[22] = d22
		ps109.OverlayValues[23] = d23
		ps109.OverlayValues[24] = d24
		ps109.OverlayValues[25] = d25
		ps109.OverlayValues[26] = d26
		ps109.OverlayValues[27] = d27
		ps109.OverlayValues[28] = d28
		ps109.OverlayValues[29] = d29
		ps109.OverlayValues[30] = d30
		ps109.OverlayValues[31] = d31
		ps109.OverlayValues[32] = d32
		ps109.OverlayValues[33] = d33
		ps109.OverlayValues[34] = d34
		ps109.OverlayValues[35] = d35
		ps109.OverlayValues[36] = d36
		ps109.OverlayValues[37] = d37
		ps109.OverlayValues[38] = d38
		ps109.OverlayValues[39] = d39
		ps109.OverlayValues[40] = d40
		ps109.OverlayValues[41] = d41
		ps109.OverlayValues[42] = d42
		ps109.OverlayValues[43] = d43
		ps109.OverlayValues[44] = d44
		ps109.OverlayValues[45] = d45
		ps109.OverlayValues[46] = d46
		ps109.OverlayValues[47] = d47
		ps109.OverlayValues[48] = d48
		ps109.OverlayValues[49] = d49
		ps109.OverlayValues[50] = d50
		ps109.OverlayValues[53] = d53
		ps109.OverlayValues[54] = d54
		ps109.OverlayValues[55] = d55
		ps110 := scm.PhiState{General: true}
		ps110.OverlayValues = make([]scm.JITValueDesc, 56)
		ps110.OverlayValues[1] = d1
		ps110.OverlayValues[2] = d2
		ps110.OverlayValues[3] = d3
		ps110.OverlayValues[4] = d4
		ps110.OverlayValues[5] = d5
		ps110.OverlayValues[6] = d6
		ps110.OverlayValues[7] = d7
		ps110.OverlayValues[8] = d8
		ps110.OverlayValues[9] = d9
		ps110.OverlayValues[10] = d10
		ps110.OverlayValues[11] = d11
		ps110.OverlayValues[12] = d12
		ps110.OverlayValues[13] = d13
		ps110.OverlayValues[14] = d14
		ps110.OverlayValues[15] = d15
		ps110.OverlayValues[17] = d17
		ps110.OverlayValues[18] = d18
		ps110.OverlayValues[19] = d19
		ps110.OverlayValues[20] = d20
		ps110.OverlayValues[21] = d21
		ps110.OverlayValues[22] = d22
		ps110.OverlayValues[23] = d23
		ps110.OverlayValues[24] = d24
		ps110.OverlayValues[25] = d25
		ps110.OverlayValues[26] = d26
		ps110.OverlayValues[27] = d27
		ps110.OverlayValues[28] = d28
		ps110.OverlayValues[29] = d29
		ps110.OverlayValues[30] = d30
		ps110.OverlayValues[31] = d31
		ps110.OverlayValues[32] = d32
		ps110.OverlayValues[33] = d33
		ps110.OverlayValues[34] = d34
		ps110.OverlayValues[35] = d35
		ps110.OverlayValues[36] = d36
		ps110.OverlayValues[37] = d37
		ps110.OverlayValues[38] = d38
		ps110.OverlayValues[39] = d39
		ps110.OverlayValues[40] = d40
		ps110.OverlayValues[41] = d41
		ps110.OverlayValues[42] = d42
		ps110.OverlayValues[43] = d43
		ps110.OverlayValues[44] = d44
		ps110.OverlayValues[45] = d45
		ps110.OverlayValues[46] = d46
		ps110.OverlayValues[47] = d47
		ps110.OverlayValues[48] = d48
		ps110.OverlayValues[49] = d49
		ps110.OverlayValues[50] = d50
		ps110.OverlayValues[53] = d53
		ps110.OverlayValues[54] = d54
		ps110.OverlayValues[55] = d55
		snap111 := d1
		snap112 := d2
		snap113 := d3
		snap114 := d4
		snap115 := d5
		snap116 := d6
		snap117 := d7
		snap118 := d8
		snap119 := d9
		snap120 := d10
		snap121 := d11
		snap122 := d12
		snap123 := d13
		snap124 := d14
		snap125 := d15
		snap126 := d17
		snap127 := d18
		snap128 := d19
		snap129 := d20
		snap130 := d21
		snap131 := d22
		snap132 := d23
		snap133 := d24
		snap134 := d25
		snap135 := d26
		snap136 := d27
		snap137 := d28
		snap138 := d29
		snap139 := d30
		snap140 := d31
		snap141 := d32
		snap142 := d33
		snap143 := d34
		snap144 := d35
		snap145 := d36
		snap146 := d37
		snap147 := d38
		snap148 := d39
		snap149 := d40
		snap150 := d41
		snap151 := d42
		snap152 := d43
		snap153 := d44
		snap154 := d45
		snap155 := d46
		snap156 := d47
		snap157 := d48
		snap158 := d49
		snap159 := d50
		snap160 := d53
		snap161 := d54
		snap162 := d55
		alloc163 := ctx.SnapshotAllocState()
		if !bbs[5].Rendered {
			bbs[5].RenderPS(ps110)
		}
		ctx.RestoreAllocState(alloc163)
		d1 = snap111
		d2 = snap112
		d3 = snap113
		d4 = snap114
		d5 = snap115
		d6 = snap116
		d7 = snap117
		d8 = snap118
		d9 = snap119
		d10 = snap120
		d11 = snap121
		d12 = snap122
		d13 = snap123
		d14 = snap124
		d15 = snap125
		d17 = snap126
		d18 = snap127
		d19 = snap128
		d20 = snap129
		d21 = snap130
		d22 = snap131
		d23 = snap132
		d24 = snap133
		d25 = snap134
		d26 = snap135
		d27 = snap136
		d28 = snap137
		d29 = snap138
		d30 = snap139
		d31 = snap140
		d32 = snap141
		d33 = snap142
		d34 = snap143
		d35 = snap144
		d36 = snap145
		d37 = snap146
		d38 = snap147
		d39 = snap148
		d40 = snap149
		d41 = snap150
		d42 = snap151
		d43 = snap152
		d44 = snap153
		d45 = snap154
		d46 = snap155
		d47 = snap156
		d48 = snap157
		d49 = snap158
		d50 = snap159
		d53 = snap160
		d54 = snap161
		d55 = snap162
		if !bbs[3].Rendered {
			return bbs[3].RenderPS(ps109)
		}
		return result
		ctx.FreeDesc(&d49)
		return result
	}
	bbs[2].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d164 := ps.PhiValues[0]
				ctx.EnsureDesc(&d164)
				ctx.EmitStoreToStack(d164, int32(bbs[2].PhiBase)+int32(0))
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
		if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != scm.LocNone {
			d164 = ps.OverlayValues[164]
		}
		if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
			d4 = ps.PhiValues[0]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d4)
		ctx.EnsureDesc(&d4)
		ctx.EnsureDesc(&d4)
		var d165 scm.JITValueDesc
		if d4.Loc == scm.LocImm {
			d165 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(d4.Imm.Int()))))}
		} else {
			r37 := ctx.AllocReg()
			ctx.EmitMovRegReg(r37, d4.Reg)
			ctx.EmitShlRegImm8(r37, 32)
			ctx.EmitShrRegImm8(r37, 32)
			d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
			ctx.BindReg(r37, &d165)
		}
		ctx.EnsureDesc(&d165)
		if thisptr.Loc == scm.LocImm {
			baseReg := ctx.AllocReg()
			if d165.Loc == scm.LocReg {
				ctx.FreeReg(baseReg)
				baseReg = ctx.AllocRegExcept(d165.Reg)
			}
			ctx.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int())+unsafe.Offsetof((*StorageSeq)(nil).lastValue)))
			if d165.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d165.Imm.Int()))
				ctx.EmitStoreRegMem(scm.RegR11, baseReg, 0)
			} else {
				ctx.EmitStoreRegMem(d165.Reg, baseReg, 0)
			}
			ctx.FreeReg(baseReg)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
			if d165.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d165.Imm.Int()))
				ctx.EmitStoreRegMem(scm.RegR11, thisptr.Reg, off)
			} else {
				ctx.EmitStoreRegMem(d165.Reg, thisptr.Reg, off)
			}
		}
		ctx.FreeDesc(&d165)
		ctx.EnsureDesc(&d4)
		d166 = d4
		_ = d166
		ctx.StabilizeDescForControlFlow(&d166)
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
		ctx.EnsureDesc(&d166)
		ctx.EnsureDesc(&d166)
		var d167 scm.JITValueDesc
		if d166.Loc == scm.LocImm {
			d167 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d166.Imm.Int()))))}
		} else {
			r38 := ctx.AllocReg()
			ctx.EmitMovRegReg(r38, d166.Reg)
			ctx.EmitShlRegImm8(r38, 32)
			ctx.EmitShrRegImm8(r38, 32)
			d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
			ctx.BindReg(r38, &d167)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d168 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d168 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 48)
			r39 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r39, thisptr.Reg, off)
			d168 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r39}
			ctx.BindReg(r39, &d168)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d168)
		ctx.EnsureDesc(&d168)
		var d169 scm.JITValueDesc
		if d168.Loc == scm.LocImm {
			d169 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d168.Imm.Int()))))}
		} else {
			r40 := ctx.AllocReg()
			ctx.EmitMovRegReg(r40, d168.Reg)
			ctx.EmitShlRegImm8(r40, 56)
			ctx.EmitShrRegImm8(r40, 56)
			d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
			ctx.BindReg(r40, &d169)
		}
		ctx.FreeDesc(&d168)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d167)
		ctx.EnsureDesc(&d169)
		ctx.EnsureDescsTogether(&d167, &d169)
		var d170 scm.JITValueDesc
		if d167.Loc == scm.LocImm && d169.Loc == scm.LocImm {
			d170 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d167.Imm.Int() * d169.Imm.Int())}
		} else if d167.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d169.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d167.Imm.Int()))
			ctx.EmitImulInt64(scratch, d169.Reg)
			d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d170)
		} else if d169.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d167.Reg)
			ctx.EmitMovRegReg(scratch, d167.Reg)
			if d169.Imm.Int() >= -2147483648 && d169.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d169.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d169.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d170)
		} else {
			r41 := ctx.AllocRegExcept(d167.Reg, d169.Reg)
			ctx.EmitMovRegReg(r41, d167.Reg)
			ctx.EmitImulInt64(r41, d169.Reg)
			d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
			ctx.BindReg(r41, &d170)
		}
		if d170.Loc == scm.LocReg && d167.Loc == scm.LocReg && d170.Reg == d167.Reg {
			ctx.TransferReg(d167.Reg)
			d167.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d167)
		ctx.FreeDesc(&d169)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d170)
		var d171 scm.JITValueDesc
		if d170.Loc == scm.LocImm {
			d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d170.Imm.Int() / 64)}
		} else {
			r42 := ctx.AllocRegExcept(d170.Reg)
			ctx.EmitMovRegReg(r42, d170.Reg)
			ctx.EmitShrRegImm8(r42, 6)
			d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
			ctx.BindReg(r42, &d171)
		}
		if d171.Loc == scm.LocReg && d170.Loc == scm.LocReg && d171.Reg == d170.Reg {
			ctx.TransferReg(d170.Reg)
			d170.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d170)
		var d172 scm.JITValueDesc
		if d170.Loc == scm.LocImm {
			d172 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d170.Imm.Int() % 64)}
		} else {
			r43 := ctx.AllocRegExcept(d170.Reg)
			ctx.EmitMovRegReg(r43, d170.Reg)
			ctx.EmitAndRegImm32(r43, 63)
			d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
			ctx.BindReg(r43, &d172)
		}
		if d172.Loc == scm.LocReg && d170.Loc == scm.LocReg && d172.Reg == d170.Reg {
			ctx.TransferReg(d170.Reg)
			d170.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d170)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d173 scm.JITValueDesc
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
		d173 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r44, Reg2: r45, Reg3: r46}
		ctx.BindReg(r44, &d173)
		ctx.BindReg(r45, &d173)
		ctx.BindReg(r46, &d173)
		ctx.BindReg(r44, &d173)
		ctx.BindReg(r45, &d173)
		ctx.BindReg(r46, &d173)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d171)
		ctx.ReclaimUntrackedRegs()
		d175 = ctx.EmitSliceElementAddress(&d173, &d171, 8)
		ctx.EnsureDesc(&d175)
		ctx.EmitMovRegMem(d175.Reg, d175.Reg, 0)
		d174 = d175
		d174.Type = scm.TagInt
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d174)
		ctx.EnsureDesc(&d172)
		ctx.EnsureDescsTogether(&d174, &d172)
		var d176 scm.JITValueDesc
		if d174.Loc == scm.LocImm && d172.Loc == scm.LocImm {
			d176 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d174.Imm.Int()) << uint64(d172.Imm.Int())))}
		} else if d172.Loc == scm.LocImm {
			r47 := ctx.AllocRegExcept(d174.Reg)
			ctx.EmitMovRegReg(r47, d174.Reg)
			ctx.EmitShlRegImm8(r47, uint8(d172.Imm.Int()))
			d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
			ctx.BindReg(r47, &d176)
		} else {
			{
				shiftSrc := d174.Reg
				r48 := ctx.AllocRegExcept(d174.Reg, d172.Reg)
				ctx.EmitMovRegReg(r48, d174.Reg)
				shiftSrc = r48
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d172.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d172.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d172.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d176)
			}
		}
		if d176.Loc == scm.LocReg && d174.Loc == scm.LocReg && d176.Reg == d174.Reg {
			ctx.TransferReg(d174.Reg)
			d174.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d174)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d171)
		ctx.EnsureDesc(&d171)
		var d177 scm.JITValueDesc
		if d171.Loc == scm.LocImm {
			d177 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d171.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d171.Reg)
			ctx.EmitMovRegReg(scratch, d171.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d177)
		}
		if d177.Loc == scm.LocReg && d171.Loc == scm.LocReg && d177.Reg == d171.Reg {
			ctx.TransferReg(d171.Reg)
			d171.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d171)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d177)
		ctx.ReclaimUntrackedRegs()
		d179 = ctx.EmitSliceElementAddress(&d173, &d177, 8)
		ctx.EnsureDesc(&d179)
		ctx.EmitMovRegMem(d179.Reg, d179.Reg, 0)
		d178 = d179
		d178.Type = scm.TagInt
		ctx.FreeDesc(&d177)
		ctx.ReclaimUntrackedRegs()
		d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d172)
		ctx.EnsureDescsTogether(&d180, &d172)
		var d181 scm.JITValueDesc
		if d180.Loc == scm.LocImm && d172.Loc == scm.LocImm {
			d181 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d180.Imm.Int() - d172.Imm.Int())}
		} else if d172.Loc == scm.LocImm && d172.Imm.Int() == 0 {
			r49 := ctx.AllocRegExcept(d180.Reg)
			ctx.EmitMovRegReg(r49, d180.Reg)
			d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
			ctx.BindReg(r49, &d181)
		} else if d180.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d172.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d180.Imm.Int()))
			ctx.EmitSubInt64(scratch, d172.Reg)
			d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d181)
		} else if d172.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d180.Reg)
			ctx.EmitMovRegReg(scratch, d180.Reg)
			if d172.Imm.Int() >= -2147483648 && d172.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d172.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d172.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d181)
		} else {
			r50 := ctx.AllocRegExcept(d180.Reg, d172.Reg)
			ctx.EmitMovRegReg(r50, d180.Reg)
			ctx.EmitSubInt64(r50, d172.Reg)
			d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
			ctx.BindReg(r50, &d181)
		}
		if d181.Loc == scm.LocReg && d180.Loc == scm.LocReg && d181.Reg == d180.Reg {
			ctx.TransferReg(d180.Reg)
			d180.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d172)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d178)
		ctx.EnsureDesc(&d181)
		ctx.EnsureDescsTogether(&d178, &d181)
		var d182 scm.JITValueDesc
		if d178.Loc == scm.LocImm && d181.Loc == scm.LocImm {
			d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d178.Imm.Int()) >> uint64(d181.Imm.Int())))}
		} else if d181.Loc == scm.LocImm {
			r51 := ctx.AllocRegExcept(d178.Reg)
			ctx.EmitMovRegReg(r51, d178.Reg)
			ctx.EmitShrRegImm8(r51, uint8(d181.Imm.Int()))
			d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
			ctx.BindReg(r51, &d182)
		} else {
			{
				shiftSrc := d178.Reg
				r52 := ctx.AllocRegExcept(d178.Reg, d181.Reg)
				ctx.EmitMovRegReg(r52, d178.Reg)
				shiftSrc = r52
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d181.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d181.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d181.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d182)
			}
		}
		if d182.Loc == scm.LocReg && d178.Loc == scm.LocReg && d182.Reg == d178.Reg {
			ctx.TransferReg(d178.Reg)
			d178.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d178)
		ctx.FreeDesc(&d181)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d176)
		ctx.EnsureDesc(&d182)
		var d183 scm.JITValueDesc
		if d176.Loc == scm.LocImm && d182.Loc == scm.LocImm {
			d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d176.Imm.Int() | d182.Imm.Int())}
		} else if d176.Loc == scm.LocImm && d176.Imm.Int() == 0 {
			d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d182.Reg}
			ctx.BindReg(d182.Reg, &d183)
		} else if d182.Loc == scm.LocImm && d182.Imm.Int() == 0 {
			r53 := ctx.AllocRegExcept(d176.Reg)
			ctx.EmitMovRegReg(r53, d176.Reg)
			d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
			ctx.BindReg(r53, &d183)
		} else if d176.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d182.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d176.Imm.Int()))
			ctx.EmitOrInt64(scratch, d182.Reg)
			d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d183)
		} else if d182.Loc == scm.LocImm {
			r54 := ctx.AllocRegExcept(d176.Reg)
			ctx.EmitMovRegReg(r54, d176.Reg)
			if d182.Imm.Int() >= -2147483648 && d182.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r54, int32(d182.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d182.Imm.Int()))
				ctx.EmitOrInt64(r54, scm.RegR11)
			}
			d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
			ctx.BindReg(r54, &d183)
		} else {
			r55 := ctx.AllocRegExcept(d176.Reg, d182.Reg)
			ctx.EmitMovRegReg(r55, d176.Reg)
			ctx.EmitOrInt64(r55, d182.Reg)
			d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
			ctx.BindReg(r55, &d183)
		}
		if d183.Loc == scm.LocReg && d176.Loc == scm.LocReg && d183.Reg == d176.Reg {
			ctx.TransferReg(d176.Reg)
			d176.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d176)
		ctx.FreeDesc(&d182)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d184 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d184 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 48)
			r56 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r56, thisptr.Reg, off)
			d184 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r56}
			ctx.BindReg(r56, &d184)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d184)
		ctx.EnsureDesc(&d184)
		var d185 scm.JITValueDesc
		if d184.Loc == scm.LocImm {
			d185 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d184.Imm.Int()))))}
		} else {
			r57 := ctx.AllocReg()
			ctx.EmitMovRegReg(r57, d184.Reg)
			ctx.EmitShlRegImm8(r57, 56)
			ctx.EmitShrRegImm8(r57, 56)
			d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r57}
			ctx.BindReg(r57, &d185)
		}
		ctx.FreeDesc(&d184)
		ctx.ReclaimUntrackedRegs()
		d186 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d185)
		ctx.EnsureDescsTogether(&d186, &d185)
		var d187 scm.JITValueDesc
		if d186.Loc == scm.LocImm && d185.Loc == scm.LocImm {
			d187 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d186.Imm.Int() - d185.Imm.Int())}
		} else if d185.Loc == scm.LocImm && d185.Imm.Int() == 0 {
			r58 := ctx.AllocRegExcept(d186.Reg)
			ctx.EmitMovRegReg(r58, d186.Reg)
			d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r58}
			ctx.BindReg(r58, &d187)
		} else if d186.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d185.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d186.Imm.Int()))
			ctx.EmitSubInt64(scratch, d185.Reg)
			d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d187)
		} else if d185.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d186.Reg)
			ctx.EmitMovRegReg(scratch, d186.Reg)
			if d185.Imm.Int() >= -2147483648 && d185.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d185.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d185.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d187)
		} else {
			r59 := ctx.AllocRegExcept(d186.Reg, d185.Reg)
			ctx.EmitMovRegReg(r59, d186.Reg)
			ctx.EmitSubInt64(r59, d185.Reg)
			d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r59}
			ctx.BindReg(r59, &d187)
		}
		if d187.Loc == scm.LocReg && d186.Loc == scm.LocReg && d187.Reg == d186.Reg {
			ctx.TransferReg(d186.Reg)
			d186.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d185)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d183)
		ctx.EnsureDesc(&d187)
		ctx.EnsureDescsTogether(&d183, &d187)
		var d188 scm.JITValueDesc
		if d183.Loc == scm.LocImm && d187.Loc == scm.LocImm {
			d188 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d183.Imm.Int()) >> uint64(d187.Imm.Int())))}
		} else if d187.Loc == scm.LocImm {
			r60 := ctx.AllocRegExcept(d183.Reg)
			ctx.EmitMovRegReg(r60, d183.Reg)
			ctx.EmitShrRegImm8(r60, uint8(d187.Imm.Int()))
			d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r60}
			ctx.BindReg(r60, &d188)
		} else {
			{
				shiftSrc := d183.Reg
				r61 := ctx.AllocRegExcept(d183.Reg, d187.Reg)
				ctx.EmitMovRegReg(r61, d183.Reg)
				shiftSrc = r61
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d187.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d187.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d187.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d188)
			}
		}
		if d188.Loc == scm.LocReg && d183.Loc == scm.LocReg && d188.Reg == d183.Reg {
			ctx.TransferReg(d183.Reg)
			d183.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d183)
		ctx.FreeDesc(&d187)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d188)
		ctx.EnsureDesc(&d188)
		ctx.EnsureDesc(&d188)
		var d189 scm.JITValueDesc
		if d188.Loc == scm.LocImm {
			d189 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d188.Imm.Int()))))}
		} else {
			r62 := ctx.AllocReg()
			ctx.EmitMovRegReg(r62, d188.Reg)
			d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r62}
			ctx.BindReg(r62, &d189)
		}
		ctx.FreeDesc(&d188)
		var d190 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d190 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 56)
			r63 := ctx.AllocReg()
			ctx.EmitMovRegMem(r63, thisptr.Reg, off)
			d190 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r63}
			ctx.BindReg(r63, &d190)
		}
		ctx.EnsureDesc(&d189)
		ctx.EnsureDesc(&d190)
		ctx.EnsureDescsTogether(&d189, &d190)
		var d191 scm.JITValueDesc
		if d189.Loc == scm.LocImm && d190.Loc == scm.LocImm {
			d191 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d189.Imm.Int() + d190.Imm.Int())}
		} else if d190.Loc == scm.LocImm && d190.Imm.Int() == 0 {
			r64 := ctx.AllocRegExcept(d189.Reg)
			ctx.EmitMovRegReg(r64, d189.Reg)
			d191 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r64}
			ctx.BindReg(r64, &d191)
		} else if d189.Loc == scm.LocImm && d189.Imm.Int() == 0 {
			d191 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d190.Reg}
			ctx.BindReg(d190.Reg, &d191)
		} else if d189.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d190.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d189.Imm.Int()))
			ctx.EmitAddInt64(scratch, d190.Reg)
			d191 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d191)
		} else if d190.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d189.Reg)
			ctx.EmitMovRegReg(scratch, d189.Reg)
			if d190.Imm.Int() >= -2147483648 && d190.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d190.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d190.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d191 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d191)
		} else {
			r65 := ctx.AllocRegExcept(d189.Reg, d190.Reg)
			ctx.EmitMovRegReg(r65, d189.Reg)
			ctx.EmitAddInt64(r65, d190.Reg)
			d191 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
			ctx.BindReg(r65, &d191)
		}
		if d191.Loc == scm.LocReg && d189.Loc == scm.LocReg && d191.Reg == d189.Reg {
			ctx.TransferReg(d189.Reg)
			d189.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d191)
		ctx.FreeDesc(&d189)
		ctx.FreeDesc(&d190)
		var d192 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 80
			val := *(*bool)(unsafe.Pointer(fieldAddr))
			d192 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 80)
			r66 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r66, thisptr.Reg, off)
			d192 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r66}
			ctx.BindReg(r66, &d192)
		}
		d193 = d192
		ctx.EnsureDesc(&d193)
		if d193.Loc != scm.LocImm && d193.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d193.Loc == scm.LocImm {
			if d193.Imm.Bool() {
				if ps.General {
				}
				ps194 := scm.PhiState{General: ps.General}
				ps194.OverlayValues = make([]scm.JITValueDesc, 194)
				ps194.OverlayValues[1] = d1
				ps194.OverlayValues[2] = d2
				ps194.OverlayValues[3] = d3
				ps194.OverlayValues[4] = d4
				ps194.OverlayValues[5] = d5
				ps194.OverlayValues[6] = d6
				ps194.OverlayValues[7] = d7
				ps194.OverlayValues[8] = d8
				ps194.OverlayValues[9] = d9
				ps194.OverlayValues[10] = d10
				ps194.OverlayValues[11] = d11
				ps194.OverlayValues[12] = d12
				ps194.OverlayValues[13] = d13
				ps194.OverlayValues[14] = d14
				ps194.OverlayValues[15] = d15
				ps194.OverlayValues[17] = d17
				ps194.OverlayValues[18] = d18
				ps194.OverlayValues[19] = d19
				ps194.OverlayValues[20] = d20
				ps194.OverlayValues[21] = d21
				ps194.OverlayValues[22] = d22
				ps194.OverlayValues[23] = d23
				ps194.OverlayValues[24] = d24
				ps194.OverlayValues[25] = d25
				ps194.OverlayValues[26] = d26
				ps194.OverlayValues[27] = d27
				ps194.OverlayValues[28] = d28
				ps194.OverlayValues[29] = d29
				ps194.OverlayValues[30] = d30
				ps194.OverlayValues[31] = d31
				ps194.OverlayValues[32] = d32
				ps194.OverlayValues[33] = d33
				ps194.OverlayValues[34] = d34
				ps194.OverlayValues[35] = d35
				ps194.OverlayValues[36] = d36
				ps194.OverlayValues[37] = d37
				ps194.OverlayValues[38] = d38
				ps194.OverlayValues[39] = d39
				ps194.OverlayValues[40] = d40
				ps194.OverlayValues[41] = d41
				ps194.OverlayValues[42] = d42
				ps194.OverlayValues[43] = d43
				ps194.OverlayValues[44] = d44
				ps194.OverlayValues[45] = d45
				ps194.OverlayValues[46] = d46
				ps194.OverlayValues[47] = d47
				ps194.OverlayValues[48] = d48
				ps194.OverlayValues[49] = d49
				ps194.OverlayValues[50] = d50
				ps194.OverlayValues[53] = d53
				ps194.OverlayValues[54] = d54
				ps194.OverlayValues[55] = d55
				ps194.OverlayValues[164] = d164
				ps194.OverlayValues[165] = d165
				ps194.OverlayValues[166] = d166
				ps194.OverlayValues[167] = d167
				ps194.OverlayValues[168] = d168
				ps194.OverlayValues[169] = d169
				ps194.OverlayValues[170] = d170
				ps194.OverlayValues[171] = d171
				ps194.OverlayValues[172] = d172
				ps194.OverlayValues[173] = d173
				ps194.OverlayValues[174] = d174
				ps194.OverlayValues[175] = d175
				ps194.OverlayValues[176] = d176
				ps194.OverlayValues[177] = d177
				ps194.OverlayValues[178] = d178
				ps194.OverlayValues[179] = d179
				ps194.OverlayValues[180] = d180
				ps194.OverlayValues[181] = d181
				ps194.OverlayValues[182] = d182
				ps194.OverlayValues[183] = d183
				ps194.OverlayValues[184] = d184
				ps194.OverlayValues[185] = d185
				ps194.OverlayValues[186] = d186
				ps194.OverlayValues[187] = d187
				ps194.OverlayValues[188] = d188
				ps194.OverlayValues[189] = d189
				ps194.OverlayValues[190] = d190
				ps194.OverlayValues[191] = d191
				ps194.OverlayValues[192] = d192
				ps194.OverlayValues[193] = d193
				return bbs[13].RenderPS(ps194)
			}
			if ps.General {
			}
			ps195 := scm.PhiState{General: ps.General}
			ps195.OverlayValues = make([]scm.JITValueDesc, 194)
			ps195.OverlayValues[1] = d1
			ps195.OverlayValues[2] = d2
			ps195.OverlayValues[3] = d3
			ps195.OverlayValues[4] = d4
			ps195.OverlayValues[5] = d5
			ps195.OverlayValues[6] = d6
			ps195.OverlayValues[7] = d7
			ps195.OverlayValues[8] = d8
			ps195.OverlayValues[9] = d9
			ps195.OverlayValues[10] = d10
			ps195.OverlayValues[11] = d11
			ps195.OverlayValues[12] = d12
			ps195.OverlayValues[13] = d13
			ps195.OverlayValues[14] = d14
			ps195.OverlayValues[15] = d15
			ps195.OverlayValues[17] = d17
			ps195.OverlayValues[18] = d18
			ps195.OverlayValues[19] = d19
			ps195.OverlayValues[20] = d20
			ps195.OverlayValues[21] = d21
			ps195.OverlayValues[22] = d22
			ps195.OverlayValues[23] = d23
			ps195.OverlayValues[24] = d24
			ps195.OverlayValues[25] = d25
			ps195.OverlayValues[26] = d26
			ps195.OverlayValues[27] = d27
			ps195.OverlayValues[28] = d28
			ps195.OverlayValues[29] = d29
			ps195.OverlayValues[30] = d30
			ps195.OverlayValues[31] = d31
			ps195.OverlayValues[32] = d32
			ps195.OverlayValues[33] = d33
			ps195.OverlayValues[34] = d34
			ps195.OverlayValues[35] = d35
			ps195.OverlayValues[36] = d36
			ps195.OverlayValues[37] = d37
			ps195.OverlayValues[38] = d38
			ps195.OverlayValues[39] = d39
			ps195.OverlayValues[40] = d40
			ps195.OverlayValues[41] = d41
			ps195.OverlayValues[42] = d42
			ps195.OverlayValues[43] = d43
			ps195.OverlayValues[44] = d44
			ps195.OverlayValues[45] = d45
			ps195.OverlayValues[46] = d46
			ps195.OverlayValues[47] = d47
			ps195.OverlayValues[48] = d48
			ps195.OverlayValues[49] = d49
			ps195.OverlayValues[50] = d50
			ps195.OverlayValues[53] = d53
			ps195.OverlayValues[54] = d54
			ps195.OverlayValues[55] = d55
			ps195.OverlayValues[164] = d164
			ps195.OverlayValues[165] = d165
			ps195.OverlayValues[166] = d166
			ps195.OverlayValues[167] = d167
			ps195.OverlayValues[168] = d168
			ps195.OverlayValues[169] = d169
			ps195.OverlayValues[170] = d170
			ps195.OverlayValues[171] = d171
			ps195.OverlayValues[172] = d172
			ps195.OverlayValues[173] = d173
			ps195.OverlayValues[174] = d174
			ps195.OverlayValues[175] = d175
			ps195.OverlayValues[176] = d176
			ps195.OverlayValues[177] = d177
			ps195.OverlayValues[178] = d178
			ps195.OverlayValues[179] = d179
			ps195.OverlayValues[180] = d180
			ps195.OverlayValues[181] = d181
			ps195.OverlayValues[182] = d182
			ps195.OverlayValues[183] = d183
			ps195.OverlayValues[184] = d184
			ps195.OverlayValues[185] = d185
			ps195.OverlayValues[186] = d186
			ps195.OverlayValues[187] = d187
			ps195.OverlayValues[188] = d188
			ps195.OverlayValues[189] = d189
			ps195.OverlayValues[190] = d190
			ps195.OverlayValues[191] = d191
			ps195.OverlayValues[192] = d192
			ps195.OverlayValues[193] = d193
			return bbs[12].RenderPS(ps195)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d196 := ps.PhiValues[0]
				ctx.EnsureDesc(&d196)
				ctx.EmitStoreToStack(d196, int32(bbs[2].PhiBase)+int32(0))
			}
			ps.General = true
			return bbs[2].RenderPS(ps)
		}
		lbl19 := ctx.ReserveLabel()
		lbl20 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d193.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl19)
		ctx.EmitJmp(lbl20)
		snap197 := d1
		snap198 := d2
		snap199 := d3
		snap200 := d4
		snap201 := d5
		snap202 := d6
		snap203 := d7
		snap204 := d8
		snap205 := d9
		snap206 := d10
		snap207 := d11
		snap208 := d12
		snap209 := d13
		snap210 := d14
		snap211 := d15
		snap212 := d17
		snap213 := d18
		snap214 := d19
		snap215 := d20
		snap216 := d21
		snap217 := d22
		snap218 := d23
		snap219 := d24
		snap220 := d25
		snap221 := d26
		snap222 := d27
		snap223 := d28
		snap224 := d29
		snap225 := d30
		snap226 := d31
		snap227 := d32
		snap228 := d33
		snap229 := d34
		snap230 := d35
		snap231 := d36
		snap232 := d37
		snap233 := d38
		snap234 := d39
		snap235 := d40
		snap236 := d41
		snap237 := d42
		snap238 := d43
		snap239 := d44
		snap240 := d45
		snap241 := d46
		snap242 := d47
		snap243 := d48
		snap244 := d49
		snap245 := d50
		snap246 := d53
		snap247 := d54
		snap248 := d55
		snap249 := d164
		snap250 := d165
		snap251 := d166
		snap252 := d167
		snap253 := d168
		snap254 := d169
		snap255 := d170
		snap256 := d171
		snap257 := d172
		snap258 := d173
		snap259 := d174
		snap260 := d175
		snap261 := d176
		snap262 := d177
		snap263 := d178
		snap264 := d179
		snap265 := d180
		snap266 := d181
		snap267 := d182
		snap268 := d183
		snap269 := d184
		snap270 := d185
		snap271 := d186
		snap272 := d187
		snap273 := d188
		snap274 := d189
		snap275 := d190
		snap276 := d191
		snap277 := d192
		snap278 := d193
		snap279 := d196
		alloc280 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl19)
		ctx.EmitJmp(lbl14)
		ctx.RestoreAllocState(alloc280)
		d1 = snap197
		d2 = snap198
		d3 = snap199
		d4 = snap200
		d5 = snap201
		d6 = snap202
		d7 = snap203
		d8 = snap204
		d9 = snap205
		d10 = snap206
		d11 = snap207
		d12 = snap208
		d13 = snap209
		d14 = snap210
		d15 = snap211
		d17 = snap212
		d18 = snap213
		d19 = snap214
		d20 = snap215
		d21 = snap216
		d22 = snap217
		d23 = snap218
		d24 = snap219
		d25 = snap220
		d26 = snap221
		d27 = snap222
		d28 = snap223
		d29 = snap224
		d30 = snap225
		d31 = snap226
		d32 = snap227
		d33 = snap228
		d34 = snap229
		d35 = snap230
		d36 = snap231
		d37 = snap232
		d38 = snap233
		d39 = snap234
		d40 = snap235
		d41 = snap236
		d42 = snap237
		d43 = snap238
		d44 = snap239
		d45 = snap240
		d46 = snap241
		d47 = snap242
		d48 = snap243
		d49 = snap244
		d50 = snap245
		d53 = snap246
		d54 = snap247
		d55 = snap248
		d164 = snap249
		d165 = snap250
		d166 = snap251
		d167 = snap252
		d168 = snap253
		d169 = snap254
		d170 = snap255
		d171 = snap256
		d172 = snap257
		d173 = snap258
		d174 = snap259
		d175 = snap260
		d176 = snap261
		d177 = snap262
		d178 = snap263
		d179 = snap264
		d180 = snap265
		d181 = snap266
		d182 = snap267
		d183 = snap268
		d184 = snap269
		d185 = snap270
		d186 = snap271
		d187 = snap272
		d188 = snap273
		d189 = snap274
		d190 = snap275
		d191 = snap276
		d192 = snap277
		d193 = snap278
		d196 = snap279
		ctx.MarkLabel(lbl20)
		ctx.EmitJmp(lbl13)
		ctx.RestoreAllocState(alloc280)
		d1 = snap197
		d2 = snap198
		d3 = snap199
		d4 = snap200
		d5 = snap201
		d6 = snap202
		d7 = snap203
		d8 = snap204
		d9 = snap205
		d10 = snap206
		d11 = snap207
		d12 = snap208
		d13 = snap209
		d14 = snap210
		d15 = snap211
		d17 = snap212
		d18 = snap213
		d19 = snap214
		d20 = snap215
		d21 = snap216
		d22 = snap217
		d23 = snap218
		d24 = snap219
		d25 = snap220
		d26 = snap221
		d27 = snap222
		d28 = snap223
		d29 = snap224
		d30 = snap225
		d31 = snap226
		d32 = snap227
		d33 = snap228
		d34 = snap229
		d35 = snap230
		d36 = snap231
		d37 = snap232
		d38 = snap233
		d39 = snap234
		d40 = snap235
		d41 = snap236
		d42 = snap237
		d43 = snap238
		d44 = snap239
		d45 = snap240
		d46 = snap241
		d47 = snap242
		d48 = snap243
		d49 = snap244
		d50 = snap245
		d53 = snap246
		d54 = snap247
		d55 = snap248
		d164 = snap249
		d165 = snap250
		d166 = snap251
		d167 = snap252
		d168 = snap253
		d169 = snap254
		d170 = snap255
		d171 = snap256
		d172 = snap257
		d173 = snap258
		d174 = snap259
		d175 = snap260
		d176 = snap261
		d177 = snap262
		d178 = snap263
		d179 = snap264
		d180 = snap265
		d181 = snap266
		d182 = snap267
		d183 = snap268
		d184 = snap269
		d185 = snap270
		d186 = snap271
		d187 = snap272
		d188 = snap273
		d189 = snap274
		d190 = snap275
		d191 = snap276
		d192 = snap277
		d193 = snap278
		d196 = snap279
		ps281 := scm.PhiState{General: true}
		ps281.OverlayValues = make([]scm.JITValueDesc, 197)
		ps281.OverlayValues[1] = d1
		ps281.OverlayValues[2] = d2
		ps281.OverlayValues[3] = d3
		ps281.OverlayValues[4] = d4
		ps281.OverlayValues[5] = d5
		ps281.OverlayValues[6] = d6
		ps281.OverlayValues[7] = d7
		ps281.OverlayValues[8] = d8
		ps281.OverlayValues[9] = d9
		ps281.OverlayValues[10] = d10
		ps281.OverlayValues[11] = d11
		ps281.OverlayValues[12] = d12
		ps281.OverlayValues[13] = d13
		ps281.OverlayValues[14] = d14
		ps281.OverlayValues[15] = d15
		ps281.OverlayValues[17] = d17
		ps281.OverlayValues[18] = d18
		ps281.OverlayValues[19] = d19
		ps281.OverlayValues[20] = d20
		ps281.OverlayValues[21] = d21
		ps281.OverlayValues[22] = d22
		ps281.OverlayValues[23] = d23
		ps281.OverlayValues[24] = d24
		ps281.OverlayValues[25] = d25
		ps281.OverlayValues[26] = d26
		ps281.OverlayValues[27] = d27
		ps281.OverlayValues[28] = d28
		ps281.OverlayValues[29] = d29
		ps281.OverlayValues[30] = d30
		ps281.OverlayValues[31] = d31
		ps281.OverlayValues[32] = d32
		ps281.OverlayValues[33] = d33
		ps281.OverlayValues[34] = d34
		ps281.OverlayValues[35] = d35
		ps281.OverlayValues[36] = d36
		ps281.OverlayValues[37] = d37
		ps281.OverlayValues[38] = d38
		ps281.OverlayValues[39] = d39
		ps281.OverlayValues[40] = d40
		ps281.OverlayValues[41] = d41
		ps281.OverlayValues[42] = d42
		ps281.OverlayValues[43] = d43
		ps281.OverlayValues[44] = d44
		ps281.OverlayValues[45] = d45
		ps281.OverlayValues[46] = d46
		ps281.OverlayValues[47] = d47
		ps281.OverlayValues[48] = d48
		ps281.OverlayValues[49] = d49
		ps281.OverlayValues[50] = d50
		ps281.OverlayValues[53] = d53
		ps281.OverlayValues[54] = d54
		ps281.OverlayValues[55] = d55
		ps281.OverlayValues[164] = d164
		ps281.OverlayValues[165] = d165
		ps281.OverlayValues[166] = d166
		ps281.OverlayValues[167] = d167
		ps281.OverlayValues[168] = d168
		ps281.OverlayValues[169] = d169
		ps281.OverlayValues[170] = d170
		ps281.OverlayValues[171] = d171
		ps281.OverlayValues[172] = d172
		ps281.OverlayValues[173] = d173
		ps281.OverlayValues[174] = d174
		ps281.OverlayValues[175] = d175
		ps281.OverlayValues[176] = d176
		ps281.OverlayValues[177] = d177
		ps281.OverlayValues[178] = d178
		ps281.OverlayValues[179] = d179
		ps281.OverlayValues[180] = d180
		ps281.OverlayValues[181] = d181
		ps281.OverlayValues[182] = d182
		ps281.OverlayValues[183] = d183
		ps281.OverlayValues[184] = d184
		ps281.OverlayValues[185] = d185
		ps281.OverlayValues[186] = d186
		ps281.OverlayValues[187] = d187
		ps281.OverlayValues[188] = d188
		ps281.OverlayValues[189] = d189
		ps281.OverlayValues[190] = d190
		ps281.OverlayValues[191] = d191
		ps281.OverlayValues[192] = d192
		ps281.OverlayValues[193] = d193
		ps281.OverlayValues[196] = d196
		ps282 := scm.PhiState{General: true}
		ps282.OverlayValues = make([]scm.JITValueDesc, 197)
		ps282.OverlayValues[1] = d1
		ps282.OverlayValues[2] = d2
		ps282.OverlayValues[3] = d3
		ps282.OverlayValues[4] = d4
		ps282.OverlayValues[5] = d5
		ps282.OverlayValues[6] = d6
		ps282.OverlayValues[7] = d7
		ps282.OverlayValues[8] = d8
		ps282.OverlayValues[9] = d9
		ps282.OverlayValues[10] = d10
		ps282.OverlayValues[11] = d11
		ps282.OverlayValues[12] = d12
		ps282.OverlayValues[13] = d13
		ps282.OverlayValues[14] = d14
		ps282.OverlayValues[15] = d15
		ps282.OverlayValues[17] = d17
		ps282.OverlayValues[18] = d18
		ps282.OverlayValues[19] = d19
		ps282.OverlayValues[20] = d20
		ps282.OverlayValues[21] = d21
		ps282.OverlayValues[22] = d22
		ps282.OverlayValues[23] = d23
		ps282.OverlayValues[24] = d24
		ps282.OverlayValues[25] = d25
		ps282.OverlayValues[26] = d26
		ps282.OverlayValues[27] = d27
		ps282.OverlayValues[28] = d28
		ps282.OverlayValues[29] = d29
		ps282.OverlayValues[30] = d30
		ps282.OverlayValues[31] = d31
		ps282.OverlayValues[32] = d32
		ps282.OverlayValues[33] = d33
		ps282.OverlayValues[34] = d34
		ps282.OverlayValues[35] = d35
		ps282.OverlayValues[36] = d36
		ps282.OverlayValues[37] = d37
		ps282.OverlayValues[38] = d38
		ps282.OverlayValues[39] = d39
		ps282.OverlayValues[40] = d40
		ps282.OverlayValues[41] = d41
		ps282.OverlayValues[42] = d42
		ps282.OverlayValues[43] = d43
		ps282.OverlayValues[44] = d44
		ps282.OverlayValues[45] = d45
		ps282.OverlayValues[46] = d46
		ps282.OverlayValues[47] = d47
		ps282.OverlayValues[48] = d48
		ps282.OverlayValues[49] = d49
		ps282.OverlayValues[50] = d50
		ps282.OverlayValues[53] = d53
		ps282.OverlayValues[54] = d54
		ps282.OverlayValues[55] = d55
		ps282.OverlayValues[164] = d164
		ps282.OverlayValues[165] = d165
		ps282.OverlayValues[166] = d166
		ps282.OverlayValues[167] = d167
		ps282.OverlayValues[168] = d168
		ps282.OverlayValues[169] = d169
		ps282.OverlayValues[170] = d170
		ps282.OverlayValues[171] = d171
		ps282.OverlayValues[172] = d172
		ps282.OverlayValues[173] = d173
		ps282.OverlayValues[174] = d174
		ps282.OverlayValues[175] = d175
		ps282.OverlayValues[176] = d176
		ps282.OverlayValues[177] = d177
		ps282.OverlayValues[178] = d178
		ps282.OverlayValues[179] = d179
		ps282.OverlayValues[180] = d180
		ps282.OverlayValues[181] = d181
		ps282.OverlayValues[182] = d182
		ps282.OverlayValues[183] = d183
		ps282.OverlayValues[184] = d184
		ps282.OverlayValues[185] = d185
		ps282.OverlayValues[186] = d186
		ps282.OverlayValues[187] = d187
		ps282.OverlayValues[188] = d188
		ps282.OverlayValues[189] = d189
		ps282.OverlayValues[190] = d190
		ps282.OverlayValues[191] = d191
		ps282.OverlayValues[192] = d192
		ps282.OverlayValues[193] = d193
		ps282.OverlayValues[196] = d196
		snap283 := d1
		snap284 := d2
		snap285 := d3
		snap286 := d4
		snap287 := d5
		snap288 := d6
		snap289 := d7
		snap290 := d8
		snap291 := d9
		snap292 := d10
		snap293 := d11
		snap294 := d12
		snap295 := d13
		snap296 := d14
		snap297 := d15
		snap298 := d17
		snap299 := d18
		snap300 := d19
		snap301 := d20
		snap302 := d21
		snap303 := d22
		snap304 := d23
		snap305 := d24
		snap306 := d25
		snap307 := d26
		snap308 := d27
		snap309 := d28
		snap310 := d29
		snap311 := d30
		snap312 := d31
		snap313 := d32
		snap314 := d33
		snap315 := d34
		snap316 := d35
		snap317 := d36
		snap318 := d37
		snap319 := d38
		snap320 := d39
		snap321 := d40
		snap322 := d41
		snap323 := d42
		snap324 := d43
		snap325 := d44
		snap326 := d45
		snap327 := d46
		snap328 := d47
		snap329 := d48
		snap330 := d49
		snap331 := d50
		snap332 := d53
		snap333 := d54
		snap334 := d55
		snap335 := d164
		snap336 := d165
		snap337 := d166
		snap338 := d167
		snap339 := d168
		snap340 := d169
		snap341 := d170
		snap342 := d171
		snap343 := d172
		snap344 := d173
		snap345 := d174
		snap346 := d175
		snap347 := d176
		snap348 := d177
		snap349 := d178
		snap350 := d179
		snap351 := d180
		snap352 := d181
		snap353 := d182
		snap354 := d183
		snap355 := d184
		snap356 := d185
		snap357 := d186
		snap358 := d187
		snap359 := d188
		snap360 := d189
		snap361 := d190
		snap362 := d191
		snap363 := d192
		snap364 := d193
		snap365 := d196
		alloc366 := ctx.SnapshotAllocState()
		if !bbs[12].Rendered {
			bbs[12].RenderPS(ps282)
		}
		ctx.RestoreAllocState(alloc366)
		d1 = snap283
		d2 = snap284
		d3 = snap285
		d4 = snap286
		d5 = snap287
		d6 = snap288
		d7 = snap289
		d8 = snap290
		d9 = snap291
		d10 = snap292
		d11 = snap293
		d12 = snap294
		d13 = snap295
		d14 = snap296
		d15 = snap297
		d17 = snap298
		d18 = snap299
		d19 = snap300
		d20 = snap301
		d21 = snap302
		d22 = snap303
		d23 = snap304
		d24 = snap305
		d25 = snap306
		d26 = snap307
		d27 = snap308
		d28 = snap309
		d29 = snap310
		d30 = snap311
		d31 = snap312
		d32 = snap313
		d33 = snap314
		d34 = snap315
		d35 = snap316
		d36 = snap317
		d37 = snap318
		d38 = snap319
		d39 = snap320
		d40 = snap321
		d41 = snap322
		d42 = snap323
		d43 = snap324
		d44 = snap325
		d45 = snap326
		d46 = snap327
		d47 = snap328
		d48 = snap329
		d49 = snap330
		d50 = snap331
		d53 = snap332
		d54 = snap333
		d55 = snap334
		d164 = snap335
		d165 = snap336
		d166 = snap337
		d167 = snap338
		d168 = snap339
		d169 = snap340
		d170 = snap341
		d171 = snap342
		d172 = snap343
		d173 = snap344
		d174 = snap345
		d175 = snap346
		d176 = snap347
		d177 = snap348
		d178 = snap349
		d179 = snap350
		d180 = snap351
		d181 = snap352
		d182 = snap353
		d183 = snap354
		d184 = snap355
		d185 = snap356
		d186 = snap357
		d187 = snap358
		d188 = snap359
		d189 = snap360
		d190 = snap361
		d191 = snap362
		d192 = snap363
		d193 = snap364
		d196 = snap365
		if !bbs[13].Rendered {
			return bbs[13].RenderPS(ps281)
		}
		return result
		ctx.FreeDesc(&d192)
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
		if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != scm.LocNone {
			d176 = ps.OverlayValues[176]
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
		if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
			d196 = ps.OverlayValues[196]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		ctx.EnsureDesc(&d1)
		var d367 scm.JITValueDesc
		if d1.Loc == scm.LocImm {
			d367 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegReg(scratch, d1.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d367 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d367)
		}
		if d367.Loc == scm.LocImm {
			d367 = scm.JITValueDesc{Loc: scm.LocImm, Type: d367.Type, Imm: scm.NewInt(int64(uint64(d367.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d367.Reg, 32)
			ctx.EmitShrRegImm8(d367.Reg, 32)
		}
		if d367.Loc == scm.LocReg && d1.Loc == scm.LocReg && d367.Reg == d1.Reg {
			ctx.TransferReg(d1.Reg)
			d1.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d367)
		ctx.EmitStoreToStack(d367, int32(bbs[4].PhiBase)+int32(32))
		ctx.StabilizeDescForControlFlow(&d367)
		ctx.EnsureDesc(&d1)
		ctx.EnsureDesc(&d1)
		var d368 scm.JITValueDesc
		if d1.Loc == scm.LocImm {
			d368 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegReg(scratch, d1.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d368 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d368)
		}
		if d368.Loc == scm.LocImm {
			d368 = scm.JITValueDesc{Loc: scm.LocImm, Type: d368.Type, Imm: scm.NewInt(int64(uint64(d368.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d368.Reg, 32)
			ctx.EmitShrRegImm8(d368.Reg, 32)
		}
		if d368.Loc == scm.LocReg && d1.Loc == scm.LocReg && d368.Reg == d1.Reg {
			ctx.TransferReg(d1.Reg)
			d1.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d368)
		ctx.EmitStoreToStack(d368, int32(bbs[4].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d368)
		if ps.General {
			ctx.SyncDesc(&d2)
			if d2.Loc == scm.LocReg {
				ctx.ProtectReg(d2.Reg)
			} else if d2.Loc == scm.LocRegPair {
				ctx.ProtectReg(d2.Reg)
				ctx.ProtectReg(d2.Reg2)
			}
			d369 = d2
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
			ctx.EmitStoreToStack(d370, int32(bbs[4].PhiBase)+int32(16))
			if d2.Loc == scm.LocReg {
				ctx.UnprotectReg(d2.Reg)
			} else if d2.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d2.Reg)
				ctx.UnprotectReg(d2.Reg2)
			}
		}
		ps371 := scm.PhiState{General: ps.General}
		ps371.OverlayValues = make([]scm.JITValueDesc, 371)
		ps371.OverlayValues[1] = d1
		ps371.OverlayValues[2] = d2
		ps371.OverlayValues[3] = d3
		ps371.OverlayValues[4] = d4
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
		ps371.OverlayValues[17] = d17
		ps371.OverlayValues[18] = d18
		ps371.OverlayValues[19] = d19
		ps371.OverlayValues[20] = d20
		ps371.OverlayValues[21] = d21
		ps371.OverlayValues[22] = d22
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
		ps371.OverlayValues[53] = d53
		ps371.OverlayValues[54] = d54
		ps371.OverlayValues[55] = d55
		ps371.OverlayValues[164] = d164
		ps371.OverlayValues[165] = d165
		ps371.OverlayValues[166] = d166
		ps371.OverlayValues[167] = d167
		ps371.OverlayValues[168] = d168
		ps371.OverlayValues[169] = d169
		ps371.OverlayValues[170] = d170
		ps371.OverlayValues[171] = d171
		ps371.OverlayValues[172] = d172
		ps371.OverlayValues[173] = d173
		ps371.OverlayValues[174] = d174
		ps371.OverlayValues[175] = d175
		ps371.OverlayValues[176] = d176
		ps371.OverlayValues[177] = d177
		ps371.OverlayValues[178] = d178
		ps371.OverlayValues[179] = d179
		ps371.OverlayValues[180] = d180
		ps371.OverlayValues[181] = d181
		ps371.OverlayValues[182] = d182
		ps371.OverlayValues[183] = d183
		ps371.OverlayValues[184] = d184
		ps371.OverlayValues[185] = d185
		ps371.OverlayValues[186] = d186
		ps371.OverlayValues[187] = d187
		ps371.OverlayValues[188] = d188
		ps371.OverlayValues[189] = d189
		ps371.OverlayValues[190] = d190
		ps371.OverlayValues[191] = d191
		ps371.OverlayValues[192] = d192
		ps371.OverlayValues[193] = d193
		ps371.OverlayValues[196] = d196
		ps371.OverlayValues[367] = d367
		ps371.OverlayValues[368] = d368
		ps371.OverlayValues[369] = d369
		ps371.OverlayValues[370] = d370
		ps371.PhiValues = make([]scm.JITValueDesc, 3)
		d372 = d2
		ps371.PhiValues[1] = d372
		if ps371.General && bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
			return result
		}
		return bbs[4].RenderPS(ps371)
		return result
	}
	bbs[4].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d373 := ps.PhiValues[0]
				ctx.EnsureDesc(&d373)
				ctx.EmitStoreToStack(d373, int32(bbs[4].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d374 := ps.PhiValues[1]
				ctx.EnsureDesc(&d374)
				ctx.EmitStoreToStack(d374, int32(bbs[4].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d375 := ps.PhiValues[2]
				ctx.EnsureDesc(&d375)
				ctx.EmitStoreToStack(d375, int32(bbs[4].PhiBase)+int32(32))
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
		if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != scm.LocNone {
			d176 = ps.OverlayValues[176]
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
		if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
			d196 = ps.OverlayValues[196]
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
		var d376 scm.JITValueDesc
		if d6.Loc == scm.LocImm && d7.Loc == scm.LocImm {
			d376 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d6.Imm.Int()) == uint64(d7.Imm.Int()))}
		} else if d7.Loc == scm.LocImm {
			r67 := ctx.AllocRegExcept(d6.Reg)
			if d7.Imm.Int() >= -2147483648 && d7.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d6.Reg, int32(d7.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d7.Imm.Int()))
				ctx.EmitCmpInt64(d6.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r67, scm.CondEqual)
			d376 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r67}
			ctx.BindReg(r67, &d376)
		} else if d6.Loc == scm.LocImm {
			r68 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d6.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d7.Reg)
			ctx.EmitSetcc(r68, scm.CondEqual)
			d376 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r68}
			ctx.BindReg(r68, &d376)
		} else {
			r69 := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitCmpInt64(d6.Reg, d7.Reg)
			ctx.EmitSetcc(r69, scm.CondEqual)
			d376 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r69}
			ctx.BindReg(r69, &d376)
		}
		d377 = d376
		ctx.EnsureDesc(&d377)
		if d377.Loc != scm.LocImm && d377.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d377.Loc == scm.LocImm {
			if d377.Imm.Bool() {
				if ps.General {
					ctx.SyncDesc(&d6)
					if d6.Loc == scm.LocReg {
						ctx.ProtectReg(d6.Reg)
					} else if d6.Loc == scm.LocRegPair {
						ctx.ProtectReg(d6.Reg)
						ctx.ProtectReg(d6.Reg2)
					}
					d378 = d6
					if d378.Loc == scm.LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d378)
					d379 = d378
					if d379.Loc == scm.LocImm {
						d379 = scm.JITValueDesc{Loc: scm.LocImm, Type: d379.Type, Imm: scm.NewInt(int64(uint64(d379.Imm.Int()) & 0xffffffff))}
					} else {
						ctx.EmitShlRegImm8(d379.Reg, 32)
						ctx.EmitShrRegImm8(d379.Reg, 32)
					}
					ctx.EmitStoreToStack(d379, int32(bbs[2].PhiBase)+int32(0))
					if d6.Loc == scm.LocReg {
						ctx.UnprotectReg(d6.Reg)
					} else if d6.Loc == scm.LocRegPair {
						ctx.UnprotectReg(d6.Reg)
						ctx.UnprotectReg(d6.Reg2)
					}
				}
				ps380 := scm.PhiState{General: ps.General}
				ps380.OverlayValues = make([]scm.JITValueDesc, 380)
				ps380.OverlayValues[1] = d1
				ps380.OverlayValues[2] = d2
				ps380.OverlayValues[3] = d3
				ps380.OverlayValues[4] = d4
				ps380.OverlayValues[5] = d5
				ps380.OverlayValues[6] = d6
				ps380.OverlayValues[7] = d7
				ps380.OverlayValues[8] = d8
				ps380.OverlayValues[9] = d9
				ps380.OverlayValues[10] = d10
				ps380.OverlayValues[11] = d11
				ps380.OverlayValues[12] = d12
				ps380.OverlayValues[13] = d13
				ps380.OverlayValues[14] = d14
				ps380.OverlayValues[15] = d15
				ps380.OverlayValues[17] = d17
				ps380.OverlayValues[18] = d18
				ps380.OverlayValues[19] = d19
				ps380.OverlayValues[20] = d20
				ps380.OverlayValues[21] = d21
				ps380.OverlayValues[22] = d22
				ps380.OverlayValues[23] = d23
				ps380.OverlayValues[24] = d24
				ps380.OverlayValues[25] = d25
				ps380.OverlayValues[26] = d26
				ps380.OverlayValues[27] = d27
				ps380.OverlayValues[28] = d28
				ps380.OverlayValues[29] = d29
				ps380.OverlayValues[30] = d30
				ps380.OverlayValues[31] = d31
				ps380.OverlayValues[32] = d32
				ps380.OverlayValues[33] = d33
				ps380.OverlayValues[34] = d34
				ps380.OverlayValues[35] = d35
				ps380.OverlayValues[36] = d36
				ps380.OverlayValues[37] = d37
				ps380.OverlayValues[38] = d38
				ps380.OverlayValues[39] = d39
				ps380.OverlayValues[40] = d40
				ps380.OverlayValues[41] = d41
				ps380.OverlayValues[42] = d42
				ps380.OverlayValues[43] = d43
				ps380.OverlayValues[44] = d44
				ps380.OverlayValues[45] = d45
				ps380.OverlayValues[46] = d46
				ps380.OverlayValues[47] = d47
				ps380.OverlayValues[48] = d48
				ps380.OverlayValues[49] = d49
				ps380.OverlayValues[50] = d50
				ps380.OverlayValues[53] = d53
				ps380.OverlayValues[54] = d54
				ps380.OverlayValues[55] = d55
				ps380.OverlayValues[164] = d164
				ps380.OverlayValues[165] = d165
				ps380.OverlayValues[166] = d166
				ps380.OverlayValues[167] = d167
				ps380.OverlayValues[168] = d168
				ps380.OverlayValues[169] = d169
				ps380.OverlayValues[170] = d170
				ps380.OverlayValues[171] = d171
				ps380.OverlayValues[172] = d172
				ps380.OverlayValues[173] = d173
				ps380.OverlayValues[174] = d174
				ps380.OverlayValues[175] = d175
				ps380.OverlayValues[176] = d176
				ps380.OverlayValues[177] = d177
				ps380.OverlayValues[178] = d178
				ps380.OverlayValues[179] = d179
				ps380.OverlayValues[180] = d180
				ps380.OverlayValues[181] = d181
				ps380.OverlayValues[182] = d182
				ps380.OverlayValues[183] = d183
				ps380.OverlayValues[184] = d184
				ps380.OverlayValues[185] = d185
				ps380.OverlayValues[186] = d186
				ps380.OverlayValues[187] = d187
				ps380.OverlayValues[188] = d188
				ps380.OverlayValues[189] = d189
				ps380.OverlayValues[190] = d190
				ps380.OverlayValues[191] = d191
				ps380.OverlayValues[192] = d192
				ps380.OverlayValues[193] = d193
				ps380.OverlayValues[196] = d196
				ps380.OverlayValues[367] = d367
				ps380.OverlayValues[368] = d368
				ps380.OverlayValues[369] = d369
				ps380.OverlayValues[370] = d370
				ps380.OverlayValues[372] = d372
				ps380.OverlayValues[373] = d373
				ps380.OverlayValues[374] = d374
				ps380.OverlayValues[375] = d375
				ps380.OverlayValues[376] = d376
				ps380.OverlayValues[377] = d377
				ps380.OverlayValues[378] = d378
				ps380.OverlayValues[379] = d379
				ps380.PhiValues = make([]scm.JITValueDesc, 1)
				d381 = d6
				ps380.PhiValues[0] = d381
				return bbs[2].RenderPS(ps380)
			}
			if ps.General {
			}
			ps382 := scm.PhiState{General: ps.General}
			ps382.OverlayValues = make([]scm.JITValueDesc, 382)
			ps382.OverlayValues[1] = d1
			ps382.OverlayValues[2] = d2
			ps382.OverlayValues[3] = d3
			ps382.OverlayValues[4] = d4
			ps382.OverlayValues[5] = d5
			ps382.OverlayValues[6] = d6
			ps382.OverlayValues[7] = d7
			ps382.OverlayValues[8] = d8
			ps382.OverlayValues[9] = d9
			ps382.OverlayValues[10] = d10
			ps382.OverlayValues[11] = d11
			ps382.OverlayValues[12] = d12
			ps382.OverlayValues[13] = d13
			ps382.OverlayValues[14] = d14
			ps382.OverlayValues[15] = d15
			ps382.OverlayValues[17] = d17
			ps382.OverlayValues[18] = d18
			ps382.OverlayValues[19] = d19
			ps382.OverlayValues[20] = d20
			ps382.OverlayValues[21] = d21
			ps382.OverlayValues[22] = d22
			ps382.OverlayValues[23] = d23
			ps382.OverlayValues[24] = d24
			ps382.OverlayValues[25] = d25
			ps382.OverlayValues[26] = d26
			ps382.OverlayValues[27] = d27
			ps382.OverlayValues[28] = d28
			ps382.OverlayValues[29] = d29
			ps382.OverlayValues[30] = d30
			ps382.OverlayValues[31] = d31
			ps382.OverlayValues[32] = d32
			ps382.OverlayValues[33] = d33
			ps382.OverlayValues[34] = d34
			ps382.OverlayValues[35] = d35
			ps382.OverlayValues[36] = d36
			ps382.OverlayValues[37] = d37
			ps382.OverlayValues[38] = d38
			ps382.OverlayValues[39] = d39
			ps382.OverlayValues[40] = d40
			ps382.OverlayValues[41] = d41
			ps382.OverlayValues[42] = d42
			ps382.OverlayValues[43] = d43
			ps382.OverlayValues[44] = d44
			ps382.OverlayValues[45] = d45
			ps382.OverlayValues[46] = d46
			ps382.OverlayValues[47] = d47
			ps382.OverlayValues[48] = d48
			ps382.OverlayValues[49] = d49
			ps382.OverlayValues[50] = d50
			ps382.OverlayValues[53] = d53
			ps382.OverlayValues[54] = d54
			ps382.OverlayValues[55] = d55
			ps382.OverlayValues[164] = d164
			ps382.OverlayValues[165] = d165
			ps382.OverlayValues[166] = d166
			ps382.OverlayValues[167] = d167
			ps382.OverlayValues[168] = d168
			ps382.OverlayValues[169] = d169
			ps382.OverlayValues[170] = d170
			ps382.OverlayValues[171] = d171
			ps382.OverlayValues[172] = d172
			ps382.OverlayValues[173] = d173
			ps382.OverlayValues[174] = d174
			ps382.OverlayValues[175] = d175
			ps382.OverlayValues[176] = d176
			ps382.OverlayValues[177] = d177
			ps382.OverlayValues[178] = d178
			ps382.OverlayValues[179] = d179
			ps382.OverlayValues[180] = d180
			ps382.OverlayValues[181] = d181
			ps382.OverlayValues[182] = d182
			ps382.OverlayValues[183] = d183
			ps382.OverlayValues[184] = d184
			ps382.OverlayValues[185] = d185
			ps382.OverlayValues[186] = d186
			ps382.OverlayValues[187] = d187
			ps382.OverlayValues[188] = d188
			ps382.OverlayValues[189] = d189
			ps382.OverlayValues[190] = d190
			ps382.OverlayValues[191] = d191
			ps382.OverlayValues[192] = d192
			ps382.OverlayValues[193] = d193
			ps382.OverlayValues[196] = d196
			ps382.OverlayValues[367] = d367
			ps382.OverlayValues[368] = d368
			ps382.OverlayValues[369] = d369
			ps382.OverlayValues[370] = d370
			ps382.OverlayValues[372] = d372
			ps382.OverlayValues[373] = d373
			ps382.OverlayValues[374] = d374
			ps382.OverlayValues[375] = d375
			ps382.OverlayValues[376] = d376
			ps382.OverlayValues[377] = d377
			ps382.OverlayValues[378] = d378
			ps382.OverlayValues[379] = d379
			ps382.OverlayValues[381] = d381
			return bbs[6].RenderPS(ps382)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d383 := ps.PhiValues[0]
				ctx.EnsureDesc(&d383)
				ctx.EmitStoreToStack(d383, int32(bbs[4].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d384 := ps.PhiValues[1]
				ctx.EnsureDesc(&d384)
				ctx.EmitStoreToStack(d384, int32(bbs[4].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d385 := ps.PhiValues[2]
				ctx.EnsureDesc(&d385)
				ctx.EmitStoreToStack(d385, int32(bbs[4].PhiBase)+int32(32))
			}
			ps.General = true
			return bbs[4].RenderPS(ps)
		}
		lbl21 := ctx.ReserveLabel()
		lbl22 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d377.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl21)
		ctx.EmitJmp(lbl22)
		snap386 := d1
		snap387 := d2
		snap388 := d3
		snap389 := d4
		snap390 := d5
		snap391 := d6
		snap392 := d7
		snap393 := d8
		snap394 := d9
		snap395 := d10
		snap396 := d11
		snap397 := d12
		snap398 := d13
		snap399 := d14
		snap400 := d15
		snap401 := d17
		snap402 := d18
		snap403 := d19
		snap404 := d20
		snap405 := d21
		snap406 := d22
		snap407 := d23
		snap408 := d24
		snap409 := d25
		snap410 := d26
		snap411 := d27
		snap412 := d28
		snap413 := d29
		snap414 := d30
		snap415 := d31
		snap416 := d32
		snap417 := d33
		snap418 := d34
		snap419 := d35
		snap420 := d36
		snap421 := d37
		snap422 := d38
		snap423 := d39
		snap424 := d40
		snap425 := d41
		snap426 := d42
		snap427 := d43
		snap428 := d44
		snap429 := d45
		snap430 := d46
		snap431 := d47
		snap432 := d48
		snap433 := d49
		snap434 := d50
		snap435 := d53
		snap436 := d54
		snap437 := d55
		snap438 := d164
		snap439 := d165
		snap440 := d166
		snap441 := d167
		snap442 := d168
		snap443 := d169
		snap444 := d170
		snap445 := d171
		snap446 := d172
		snap447 := d173
		snap448 := d174
		snap449 := d175
		snap450 := d176
		snap451 := d177
		snap452 := d178
		snap453 := d179
		snap454 := d180
		snap455 := d181
		snap456 := d182
		snap457 := d183
		snap458 := d184
		snap459 := d185
		snap460 := d186
		snap461 := d187
		snap462 := d188
		snap463 := d189
		snap464 := d190
		snap465 := d191
		snap466 := d192
		snap467 := d193
		snap468 := d196
		snap469 := d367
		snap470 := d368
		snap471 := d369
		snap472 := d370
		snap473 := d372
		snap474 := d373
		snap475 := d374
		snap476 := d375
		snap477 := d376
		snap478 := d377
		snap479 := d378
		snap480 := d379
		snap481 := d381
		snap482 := d383
		snap483 := d384
		snap484 := d385
		alloc485 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl21)
		ctx.SyncDesc(&d6)
		if d6.Loc == scm.LocReg {
			ctx.ProtectReg(d6.Reg)
		} else if d6.Loc == scm.LocRegPair {
			ctx.ProtectReg(d6.Reg)
			ctx.ProtectReg(d6.Reg2)
		}
		d486 = d6
		if d486.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d486)
		d487 = d486
		if d487.Loc == scm.LocImm {
			d487 = scm.JITValueDesc{Loc: scm.LocImm, Type: d487.Type, Imm: scm.NewInt(int64(uint64(d487.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d487.Reg, 32)
			ctx.EmitShrRegImm8(d487.Reg, 32)
		}
		ctx.EmitStoreToStack(d487, int32(bbs[2].PhiBase)+int32(0))
		if d6.Loc == scm.LocReg {
			ctx.UnprotectReg(d6.Reg)
		} else if d6.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d6.Reg)
			ctx.UnprotectReg(d6.Reg2)
		}
		ctx.EmitJmp(lbl3)
		ctx.RestoreAllocState(alloc485)
		d1 = snap386
		d2 = snap387
		d3 = snap388
		d4 = snap389
		d5 = snap390
		d6 = snap391
		d7 = snap392
		d8 = snap393
		d9 = snap394
		d10 = snap395
		d11 = snap396
		d12 = snap397
		d13 = snap398
		d14 = snap399
		d15 = snap400
		d17 = snap401
		d18 = snap402
		d19 = snap403
		d20 = snap404
		d21 = snap405
		d22 = snap406
		d23 = snap407
		d24 = snap408
		d25 = snap409
		d26 = snap410
		d27 = snap411
		d28 = snap412
		d29 = snap413
		d30 = snap414
		d31 = snap415
		d32 = snap416
		d33 = snap417
		d34 = snap418
		d35 = snap419
		d36 = snap420
		d37 = snap421
		d38 = snap422
		d39 = snap423
		d40 = snap424
		d41 = snap425
		d42 = snap426
		d43 = snap427
		d44 = snap428
		d45 = snap429
		d46 = snap430
		d47 = snap431
		d48 = snap432
		d49 = snap433
		d50 = snap434
		d53 = snap435
		d54 = snap436
		d55 = snap437
		d164 = snap438
		d165 = snap439
		d166 = snap440
		d167 = snap441
		d168 = snap442
		d169 = snap443
		d170 = snap444
		d171 = snap445
		d172 = snap446
		d173 = snap447
		d174 = snap448
		d175 = snap449
		d176 = snap450
		d177 = snap451
		d178 = snap452
		d179 = snap453
		d180 = snap454
		d181 = snap455
		d182 = snap456
		d183 = snap457
		d184 = snap458
		d185 = snap459
		d186 = snap460
		d187 = snap461
		d188 = snap462
		d189 = snap463
		d190 = snap464
		d191 = snap465
		d192 = snap466
		d193 = snap467
		d196 = snap468
		d367 = snap469
		d368 = snap470
		d369 = snap471
		d370 = snap472
		d372 = snap473
		d373 = snap474
		d374 = snap475
		d375 = snap476
		d376 = snap477
		d377 = snap478
		d378 = snap479
		d379 = snap480
		d381 = snap481
		d383 = snap482
		d384 = snap483
		d385 = snap484
		ctx.MarkLabel(lbl22)
		ctx.EmitJmp(lbl7)
		ctx.RestoreAllocState(alloc485)
		d1 = snap386
		d2 = snap387
		d3 = snap388
		d4 = snap389
		d5 = snap390
		d6 = snap391
		d7 = snap392
		d8 = snap393
		d9 = snap394
		d10 = snap395
		d11 = snap396
		d12 = snap397
		d13 = snap398
		d14 = snap399
		d15 = snap400
		d17 = snap401
		d18 = snap402
		d19 = snap403
		d20 = snap404
		d21 = snap405
		d22 = snap406
		d23 = snap407
		d24 = snap408
		d25 = snap409
		d26 = snap410
		d27 = snap411
		d28 = snap412
		d29 = snap413
		d30 = snap414
		d31 = snap415
		d32 = snap416
		d33 = snap417
		d34 = snap418
		d35 = snap419
		d36 = snap420
		d37 = snap421
		d38 = snap422
		d39 = snap423
		d40 = snap424
		d41 = snap425
		d42 = snap426
		d43 = snap427
		d44 = snap428
		d45 = snap429
		d46 = snap430
		d47 = snap431
		d48 = snap432
		d49 = snap433
		d50 = snap434
		d53 = snap435
		d54 = snap436
		d55 = snap437
		d164 = snap438
		d165 = snap439
		d166 = snap440
		d167 = snap441
		d168 = snap442
		d169 = snap443
		d170 = snap444
		d171 = snap445
		d172 = snap446
		d173 = snap447
		d174 = snap448
		d175 = snap449
		d176 = snap450
		d177 = snap451
		d178 = snap452
		d179 = snap453
		d180 = snap454
		d181 = snap455
		d182 = snap456
		d183 = snap457
		d184 = snap458
		d185 = snap459
		d186 = snap460
		d187 = snap461
		d188 = snap462
		d189 = snap463
		d190 = snap464
		d191 = snap465
		d192 = snap466
		d193 = snap467
		d196 = snap468
		d367 = snap469
		d368 = snap470
		d369 = snap471
		d370 = snap472
		d372 = snap473
		d373 = snap474
		d374 = snap475
		d375 = snap476
		d376 = snap477
		d377 = snap478
		d378 = snap479
		d379 = snap480
		d381 = snap481
		d383 = snap482
		d384 = snap483
		d385 = snap484
		ps488 := scm.PhiState{General: true}
		ps488.OverlayValues = make([]scm.JITValueDesc, 488)
		ps488.OverlayValues[1] = d1
		ps488.OverlayValues[2] = d2
		ps488.OverlayValues[3] = d3
		ps488.OverlayValues[4] = d4
		ps488.OverlayValues[5] = d5
		ps488.OverlayValues[6] = d6
		ps488.OverlayValues[7] = d7
		ps488.OverlayValues[8] = d8
		ps488.OverlayValues[9] = d9
		ps488.OverlayValues[10] = d10
		ps488.OverlayValues[11] = d11
		ps488.OverlayValues[12] = d12
		ps488.OverlayValues[13] = d13
		ps488.OverlayValues[14] = d14
		ps488.OverlayValues[15] = d15
		ps488.OverlayValues[17] = d17
		ps488.OverlayValues[18] = d18
		ps488.OverlayValues[19] = d19
		ps488.OverlayValues[20] = d20
		ps488.OverlayValues[21] = d21
		ps488.OverlayValues[22] = d22
		ps488.OverlayValues[23] = d23
		ps488.OverlayValues[24] = d24
		ps488.OverlayValues[25] = d25
		ps488.OverlayValues[26] = d26
		ps488.OverlayValues[27] = d27
		ps488.OverlayValues[28] = d28
		ps488.OverlayValues[29] = d29
		ps488.OverlayValues[30] = d30
		ps488.OverlayValues[31] = d31
		ps488.OverlayValues[32] = d32
		ps488.OverlayValues[33] = d33
		ps488.OverlayValues[34] = d34
		ps488.OverlayValues[35] = d35
		ps488.OverlayValues[36] = d36
		ps488.OverlayValues[37] = d37
		ps488.OverlayValues[38] = d38
		ps488.OverlayValues[39] = d39
		ps488.OverlayValues[40] = d40
		ps488.OverlayValues[41] = d41
		ps488.OverlayValues[42] = d42
		ps488.OverlayValues[43] = d43
		ps488.OverlayValues[44] = d44
		ps488.OverlayValues[45] = d45
		ps488.OverlayValues[46] = d46
		ps488.OverlayValues[47] = d47
		ps488.OverlayValues[48] = d48
		ps488.OverlayValues[49] = d49
		ps488.OverlayValues[50] = d50
		ps488.OverlayValues[53] = d53
		ps488.OverlayValues[54] = d54
		ps488.OverlayValues[55] = d55
		ps488.OverlayValues[164] = d164
		ps488.OverlayValues[165] = d165
		ps488.OverlayValues[166] = d166
		ps488.OverlayValues[167] = d167
		ps488.OverlayValues[168] = d168
		ps488.OverlayValues[169] = d169
		ps488.OverlayValues[170] = d170
		ps488.OverlayValues[171] = d171
		ps488.OverlayValues[172] = d172
		ps488.OverlayValues[173] = d173
		ps488.OverlayValues[174] = d174
		ps488.OverlayValues[175] = d175
		ps488.OverlayValues[176] = d176
		ps488.OverlayValues[177] = d177
		ps488.OverlayValues[178] = d178
		ps488.OverlayValues[179] = d179
		ps488.OverlayValues[180] = d180
		ps488.OverlayValues[181] = d181
		ps488.OverlayValues[182] = d182
		ps488.OverlayValues[183] = d183
		ps488.OverlayValues[184] = d184
		ps488.OverlayValues[185] = d185
		ps488.OverlayValues[186] = d186
		ps488.OverlayValues[187] = d187
		ps488.OverlayValues[188] = d188
		ps488.OverlayValues[189] = d189
		ps488.OverlayValues[190] = d190
		ps488.OverlayValues[191] = d191
		ps488.OverlayValues[192] = d192
		ps488.OverlayValues[193] = d193
		ps488.OverlayValues[196] = d196
		ps488.OverlayValues[367] = d367
		ps488.OverlayValues[368] = d368
		ps488.OverlayValues[369] = d369
		ps488.OverlayValues[370] = d370
		ps488.OverlayValues[372] = d372
		ps488.OverlayValues[373] = d373
		ps488.OverlayValues[374] = d374
		ps488.OverlayValues[375] = d375
		ps488.OverlayValues[376] = d376
		ps488.OverlayValues[377] = d377
		ps488.OverlayValues[378] = d378
		ps488.OverlayValues[379] = d379
		ps488.OverlayValues[381] = d381
		ps488.OverlayValues[383] = d383
		ps488.OverlayValues[384] = d384
		ps488.OverlayValues[385] = d385
		ps488.OverlayValues[486] = d486
		ps488.OverlayValues[487] = d487
		ps488.PhiValues = make([]scm.JITValueDesc, 1)
		d490 = d6
		ps488.PhiValues[0] = d490
		ps489 := scm.PhiState{General: true}
		ps489.OverlayValues = make([]scm.JITValueDesc, 491)
		ps489.OverlayValues[1] = d1
		ps489.OverlayValues[2] = d2
		ps489.OverlayValues[3] = d3
		ps489.OverlayValues[4] = d4
		ps489.OverlayValues[5] = d5
		ps489.OverlayValues[6] = d6
		ps489.OverlayValues[7] = d7
		ps489.OverlayValues[8] = d8
		ps489.OverlayValues[9] = d9
		ps489.OverlayValues[10] = d10
		ps489.OverlayValues[11] = d11
		ps489.OverlayValues[12] = d12
		ps489.OverlayValues[13] = d13
		ps489.OverlayValues[14] = d14
		ps489.OverlayValues[15] = d15
		ps489.OverlayValues[17] = d17
		ps489.OverlayValues[18] = d18
		ps489.OverlayValues[19] = d19
		ps489.OverlayValues[20] = d20
		ps489.OverlayValues[21] = d21
		ps489.OverlayValues[22] = d22
		ps489.OverlayValues[23] = d23
		ps489.OverlayValues[24] = d24
		ps489.OverlayValues[25] = d25
		ps489.OverlayValues[26] = d26
		ps489.OverlayValues[27] = d27
		ps489.OverlayValues[28] = d28
		ps489.OverlayValues[29] = d29
		ps489.OverlayValues[30] = d30
		ps489.OverlayValues[31] = d31
		ps489.OverlayValues[32] = d32
		ps489.OverlayValues[33] = d33
		ps489.OverlayValues[34] = d34
		ps489.OverlayValues[35] = d35
		ps489.OverlayValues[36] = d36
		ps489.OverlayValues[37] = d37
		ps489.OverlayValues[38] = d38
		ps489.OverlayValues[39] = d39
		ps489.OverlayValues[40] = d40
		ps489.OverlayValues[41] = d41
		ps489.OverlayValues[42] = d42
		ps489.OverlayValues[43] = d43
		ps489.OverlayValues[44] = d44
		ps489.OverlayValues[45] = d45
		ps489.OverlayValues[46] = d46
		ps489.OverlayValues[47] = d47
		ps489.OverlayValues[48] = d48
		ps489.OverlayValues[49] = d49
		ps489.OverlayValues[50] = d50
		ps489.OverlayValues[53] = d53
		ps489.OverlayValues[54] = d54
		ps489.OverlayValues[55] = d55
		ps489.OverlayValues[164] = d164
		ps489.OverlayValues[165] = d165
		ps489.OverlayValues[166] = d166
		ps489.OverlayValues[167] = d167
		ps489.OverlayValues[168] = d168
		ps489.OverlayValues[169] = d169
		ps489.OverlayValues[170] = d170
		ps489.OverlayValues[171] = d171
		ps489.OverlayValues[172] = d172
		ps489.OverlayValues[173] = d173
		ps489.OverlayValues[174] = d174
		ps489.OverlayValues[175] = d175
		ps489.OverlayValues[176] = d176
		ps489.OverlayValues[177] = d177
		ps489.OverlayValues[178] = d178
		ps489.OverlayValues[179] = d179
		ps489.OverlayValues[180] = d180
		ps489.OverlayValues[181] = d181
		ps489.OverlayValues[182] = d182
		ps489.OverlayValues[183] = d183
		ps489.OverlayValues[184] = d184
		ps489.OverlayValues[185] = d185
		ps489.OverlayValues[186] = d186
		ps489.OverlayValues[187] = d187
		ps489.OverlayValues[188] = d188
		ps489.OverlayValues[189] = d189
		ps489.OverlayValues[190] = d190
		ps489.OverlayValues[191] = d191
		ps489.OverlayValues[192] = d192
		ps489.OverlayValues[193] = d193
		ps489.OverlayValues[196] = d196
		ps489.OverlayValues[367] = d367
		ps489.OverlayValues[368] = d368
		ps489.OverlayValues[369] = d369
		ps489.OverlayValues[370] = d370
		ps489.OverlayValues[372] = d372
		ps489.OverlayValues[373] = d373
		ps489.OverlayValues[374] = d374
		ps489.OverlayValues[375] = d375
		ps489.OverlayValues[376] = d376
		ps489.OverlayValues[377] = d377
		ps489.OverlayValues[378] = d378
		ps489.OverlayValues[379] = d379
		ps489.OverlayValues[381] = d381
		ps489.OverlayValues[383] = d383
		ps489.OverlayValues[384] = d384
		ps489.OverlayValues[385] = d385
		ps489.OverlayValues[486] = d486
		ps489.OverlayValues[487] = d487
		ps489.OverlayValues[490] = d490
		snap491 := d1
		snap492 := d2
		snap493 := d3
		snap494 := d4
		snap495 := d5
		snap496 := d6
		snap497 := d7
		snap498 := d8
		snap499 := d9
		snap500 := d10
		snap501 := d11
		snap502 := d12
		snap503 := d13
		snap504 := d14
		snap505 := d15
		snap506 := d17
		snap507 := d18
		snap508 := d19
		snap509 := d20
		snap510 := d21
		snap511 := d22
		snap512 := d23
		snap513 := d24
		snap514 := d25
		snap515 := d26
		snap516 := d27
		snap517 := d28
		snap518 := d29
		snap519 := d30
		snap520 := d31
		snap521 := d32
		snap522 := d33
		snap523 := d34
		snap524 := d35
		snap525 := d36
		snap526 := d37
		snap527 := d38
		snap528 := d39
		snap529 := d40
		snap530 := d41
		snap531 := d42
		snap532 := d43
		snap533 := d44
		snap534 := d45
		snap535 := d46
		snap536 := d47
		snap537 := d48
		snap538 := d49
		snap539 := d50
		snap540 := d53
		snap541 := d54
		snap542 := d55
		snap543 := d164
		snap544 := d165
		snap545 := d166
		snap546 := d167
		snap547 := d168
		snap548 := d169
		snap549 := d170
		snap550 := d171
		snap551 := d172
		snap552 := d173
		snap553 := d174
		snap554 := d175
		snap555 := d176
		snap556 := d177
		snap557 := d178
		snap558 := d179
		snap559 := d180
		snap560 := d181
		snap561 := d182
		snap562 := d183
		snap563 := d184
		snap564 := d185
		snap565 := d186
		snap566 := d187
		snap567 := d188
		snap568 := d189
		snap569 := d190
		snap570 := d191
		snap571 := d192
		snap572 := d193
		snap573 := d196
		snap574 := d367
		snap575 := d368
		snap576 := d369
		snap577 := d370
		snap578 := d372
		snap579 := d373
		snap580 := d374
		snap581 := d375
		snap582 := d376
		snap583 := d377
		snap584 := d378
		snap585 := d379
		snap586 := d381
		snap587 := d383
		snap588 := d384
		snap589 := d385
		snap590 := d486
		snap591 := d487
		snap592 := d490
		alloc593 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps488)
		}
		ctx.RestoreAllocState(alloc593)
		d1 = snap491
		d2 = snap492
		d3 = snap493
		d4 = snap494
		d5 = snap495
		d6 = snap496
		d7 = snap497
		d8 = snap498
		d9 = snap499
		d10 = snap500
		d11 = snap501
		d12 = snap502
		d13 = snap503
		d14 = snap504
		d15 = snap505
		d17 = snap506
		d18 = snap507
		d19 = snap508
		d20 = snap509
		d21 = snap510
		d22 = snap511
		d23 = snap512
		d24 = snap513
		d25 = snap514
		d26 = snap515
		d27 = snap516
		d28 = snap517
		d29 = snap518
		d30 = snap519
		d31 = snap520
		d32 = snap521
		d33 = snap522
		d34 = snap523
		d35 = snap524
		d36 = snap525
		d37 = snap526
		d38 = snap527
		d39 = snap528
		d40 = snap529
		d41 = snap530
		d42 = snap531
		d43 = snap532
		d44 = snap533
		d45 = snap534
		d46 = snap535
		d47 = snap536
		d48 = snap537
		d49 = snap538
		d50 = snap539
		d53 = snap540
		d54 = snap541
		d55 = snap542
		d164 = snap543
		d165 = snap544
		d166 = snap545
		d167 = snap546
		d168 = snap547
		d169 = snap548
		d170 = snap549
		d171 = snap550
		d172 = snap551
		d173 = snap552
		d174 = snap553
		d175 = snap554
		d176 = snap555
		d177 = snap556
		d178 = snap557
		d179 = snap558
		d180 = snap559
		d181 = snap560
		d182 = snap561
		d183 = snap562
		d184 = snap563
		d185 = snap564
		d186 = snap565
		d187 = snap566
		d188 = snap567
		d189 = snap568
		d190 = snap569
		d191 = snap570
		d192 = snap571
		d193 = snap572
		d196 = snap573
		d367 = snap574
		d368 = snap575
		d369 = snap576
		d370 = snap577
		d372 = snap578
		d373 = snap579
		d374 = snap580
		d375 = snap581
		d376 = snap582
		d377 = snap583
		d378 = snap584
		d379 = snap585
		d381 = snap586
		d383 = snap587
		d384 = snap588
		d385 = snap589
		d486 = snap590
		d487 = snap591
		d490 = snap592
		if !bbs[6].Rendered {
			return bbs[6].RenderPS(ps489)
		}
		return result
		ctx.FreeDesc(&d376)
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
		if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != scm.LocNone {
			d176 = ps.OverlayValues[176]
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
		if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
			d196 = ps.OverlayValues[196]
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
		if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != scm.LocNone {
			d381 = ps.OverlayValues[381]
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
		if len(ps.OverlayValues) > 486 && ps.OverlayValues[486].Loc != scm.LocNone {
			d486 = ps.OverlayValues[486]
		}
		if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != scm.LocNone {
			d487 = ps.OverlayValues[487]
		}
		if len(ps.OverlayValues) > 490 && ps.OverlayValues[490].Loc != scm.LocNone {
			d490 = ps.OverlayValues[490]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		ctx.EnsureDesc(&d1)
		var d594 scm.JITValueDesc
		if d1.Loc == scm.LocImm {
			d594 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegReg(scratch, d1.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d594 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d594)
		}
		if d594.Loc == scm.LocImm {
			d594 = scm.JITValueDesc{Loc: scm.LocImm, Type: d594.Type, Imm: scm.NewInt(int64(uint64(d594.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d594.Reg, 32)
			ctx.EmitShrRegImm8(d594.Reg, 32)
		}
		if d594.Loc == scm.LocReg && d1.Loc == scm.LocReg && d594.Reg == d1.Reg {
			ctx.TransferReg(d1.Reg)
			d1.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d594)
		ctx.EmitStoreToStack(d594, int32(bbs[4].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d594)
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
			d595 = d1
			if d595.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d595)
			d596 = d595
			if d596.Loc == scm.LocImm {
				d596 = scm.JITValueDesc{Loc: scm.LocImm, Type: d596.Type, Imm: scm.NewInt(int64(uint64(d596.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d596.Reg, 32)
				ctx.EmitShrRegImm8(d596.Reg, 32)
			}
			ctx.EmitStoreToStack(d596, int32(bbs[4].PhiBase)+int32(16))
			d597 = d3
			if d597.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d597)
			d598 = d597
			if d598.Loc == scm.LocImm {
				d598 = scm.JITValueDesc{Loc: scm.LocImm, Type: d598.Type, Imm: scm.NewInt(int64(uint64(d598.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d598.Reg, 32)
				ctx.EmitShrRegImm8(d598.Reg, 32)
			}
			ctx.EmitStoreToStack(d598, int32(bbs[4].PhiBase)+int32(32))
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
		ps599 := scm.PhiState{General: ps.General}
		ps599.OverlayValues = make([]scm.JITValueDesc, 599)
		ps599.OverlayValues[1] = d1
		ps599.OverlayValues[2] = d2
		ps599.OverlayValues[3] = d3
		ps599.OverlayValues[4] = d4
		ps599.OverlayValues[5] = d5
		ps599.OverlayValues[6] = d6
		ps599.OverlayValues[7] = d7
		ps599.OverlayValues[8] = d8
		ps599.OverlayValues[9] = d9
		ps599.OverlayValues[10] = d10
		ps599.OverlayValues[11] = d11
		ps599.OverlayValues[12] = d12
		ps599.OverlayValues[13] = d13
		ps599.OverlayValues[14] = d14
		ps599.OverlayValues[15] = d15
		ps599.OverlayValues[17] = d17
		ps599.OverlayValues[18] = d18
		ps599.OverlayValues[19] = d19
		ps599.OverlayValues[20] = d20
		ps599.OverlayValues[21] = d21
		ps599.OverlayValues[22] = d22
		ps599.OverlayValues[23] = d23
		ps599.OverlayValues[24] = d24
		ps599.OverlayValues[25] = d25
		ps599.OverlayValues[26] = d26
		ps599.OverlayValues[27] = d27
		ps599.OverlayValues[28] = d28
		ps599.OverlayValues[29] = d29
		ps599.OverlayValues[30] = d30
		ps599.OverlayValues[31] = d31
		ps599.OverlayValues[32] = d32
		ps599.OverlayValues[33] = d33
		ps599.OverlayValues[34] = d34
		ps599.OverlayValues[35] = d35
		ps599.OverlayValues[36] = d36
		ps599.OverlayValues[37] = d37
		ps599.OverlayValues[38] = d38
		ps599.OverlayValues[39] = d39
		ps599.OverlayValues[40] = d40
		ps599.OverlayValues[41] = d41
		ps599.OverlayValues[42] = d42
		ps599.OverlayValues[43] = d43
		ps599.OverlayValues[44] = d44
		ps599.OverlayValues[45] = d45
		ps599.OverlayValues[46] = d46
		ps599.OverlayValues[47] = d47
		ps599.OverlayValues[48] = d48
		ps599.OverlayValues[49] = d49
		ps599.OverlayValues[50] = d50
		ps599.OverlayValues[53] = d53
		ps599.OverlayValues[54] = d54
		ps599.OverlayValues[55] = d55
		ps599.OverlayValues[164] = d164
		ps599.OverlayValues[165] = d165
		ps599.OverlayValues[166] = d166
		ps599.OverlayValues[167] = d167
		ps599.OverlayValues[168] = d168
		ps599.OverlayValues[169] = d169
		ps599.OverlayValues[170] = d170
		ps599.OverlayValues[171] = d171
		ps599.OverlayValues[172] = d172
		ps599.OverlayValues[173] = d173
		ps599.OverlayValues[174] = d174
		ps599.OverlayValues[175] = d175
		ps599.OverlayValues[176] = d176
		ps599.OverlayValues[177] = d177
		ps599.OverlayValues[178] = d178
		ps599.OverlayValues[179] = d179
		ps599.OverlayValues[180] = d180
		ps599.OverlayValues[181] = d181
		ps599.OverlayValues[182] = d182
		ps599.OverlayValues[183] = d183
		ps599.OverlayValues[184] = d184
		ps599.OverlayValues[185] = d185
		ps599.OverlayValues[186] = d186
		ps599.OverlayValues[187] = d187
		ps599.OverlayValues[188] = d188
		ps599.OverlayValues[189] = d189
		ps599.OverlayValues[190] = d190
		ps599.OverlayValues[191] = d191
		ps599.OverlayValues[192] = d192
		ps599.OverlayValues[193] = d193
		ps599.OverlayValues[196] = d196
		ps599.OverlayValues[367] = d367
		ps599.OverlayValues[368] = d368
		ps599.OverlayValues[369] = d369
		ps599.OverlayValues[370] = d370
		ps599.OverlayValues[372] = d372
		ps599.OverlayValues[373] = d373
		ps599.OverlayValues[374] = d374
		ps599.OverlayValues[375] = d375
		ps599.OverlayValues[376] = d376
		ps599.OverlayValues[377] = d377
		ps599.OverlayValues[378] = d378
		ps599.OverlayValues[379] = d379
		ps599.OverlayValues[381] = d381
		ps599.OverlayValues[383] = d383
		ps599.OverlayValues[384] = d384
		ps599.OverlayValues[385] = d385
		ps599.OverlayValues[486] = d486
		ps599.OverlayValues[487] = d487
		ps599.OverlayValues[490] = d490
		ps599.OverlayValues[594] = d594
		ps599.OverlayValues[595] = d595
		ps599.OverlayValues[596] = d596
		ps599.OverlayValues[597] = d597
		ps599.OverlayValues[598] = d598
		ps599.PhiValues = make([]scm.JITValueDesc, 3)
		d600 = d1
		ps599.PhiValues[1] = d600
		d601 = d3
		ps599.PhiValues[2] = d601
		if ps599.General && bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
			return result
		}
		return bbs[4].RenderPS(ps599)
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
		if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != scm.LocNone {
			d176 = ps.OverlayValues[176]
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
		if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
			d196 = ps.OverlayValues[196]
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
		if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != scm.LocNone {
			d381 = ps.OverlayValues[381]
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
		if len(ps.OverlayValues) > 486 && ps.OverlayValues[486].Loc != scm.LocNone {
			d486 = ps.OverlayValues[486]
		}
		if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != scm.LocNone {
			d487 = ps.OverlayValues[487]
		}
		if len(ps.OverlayValues) > 490 && ps.OverlayValues[490].Loc != scm.LocNone {
			d490 = ps.OverlayValues[490]
		}
		if len(ps.OverlayValues) > 594 && ps.OverlayValues[594].Loc != scm.LocNone {
			d594 = ps.OverlayValues[594]
		}
		if len(ps.OverlayValues) > 595 && ps.OverlayValues[595].Loc != scm.LocNone {
			d595 = ps.OverlayValues[595]
		}
		if len(ps.OverlayValues) > 596 && ps.OverlayValues[596].Loc != scm.LocNone {
			d596 = ps.OverlayValues[596]
		}
		if len(ps.OverlayValues) > 597 && ps.OverlayValues[597].Loc != scm.LocNone {
			d597 = ps.OverlayValues[597]
		}
		if len(ps.OverlayValues) > 598 && ps.OverlayValues[598].Loc != scm.LocNone {
			d598 = ps.OverlayValues[598]
		}
		if len(ps.OverlayValues) > 600 && ps.OverlayValues[600].Loc != scm.LocNone {
			d600 = ps.OverlayValues[600]
		}
		if len(ps.OverlayValues) > 601 && ps.OverlayValues[601].Loc != scm.LocNone {
			d601 = ps.OverlayValues[601]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		d602 = d5
		_ = d602
		ctx.StabilizeDescForControlFlow(&d602)
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
		ctx.EnsureDesc(&d602)
		ctx.EnsureDesc(&d602)
		var d603 scm.JITValueDesc
		if d602.Loc == scm.LocImm {
			d603 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d602.Imm.Int()))))}
		} else {
			r70 := ctx.AllocReg()
			ctx.EmitMovRegReg(r70, d602.Reg)
			ctx.EmitShlRegImm8(r70, 32)
			ctx.EmitShrRegImm8(r70, 32)
			d603 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r70}
			ctx.BindReg(r70, &d603)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d604 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d604 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48)
			r71 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r71, thisptr.Reg, off)
			d604 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r71}
			ctx.BindReg(r71, &d604)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d604)
		ctx.EnsureDesc(&d604)
		var d605 scm.JITValueDesc
		if d604.Loc == scm.LocImm {
			d605 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d604.Imm.Int()))))}
		} else {
			r72 := ctx.AllocReg()
			ctx.EmitMovRegReg(r72, d604.Reg)
			ctx.EmitShlRegImm8(r72, 56)
			ctx.EmitShrRegImm8(r72, 56)
			d605 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r72}
			ctx.BindReg(r72, &d605)
		}
		ctx.FreeDesc(&d604)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d603)
		ctx.EnsureDesc(&d605)
		ctx.EnsureDescsTogether(&d603, &d605)
		var d606 scm.JITValueDesc
		if d603.Loc == scm.LocImm && d605.Loc == scm.LocImm {
			d606 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d603.Imm.Int() * d605.Imm.Int())}
		} else if d603.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d605.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d603.Imm.Int()))
			ctx.EmitImulInt64(scratch, d605.Reg)
			d606 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d606)
		} else if d605.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d603.Reg)
			ctx.EmitMovRegReg(scratch, d603.Reg)
			if d605.Imm.Int() >= -2147483648 && d605.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d605.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d605.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d606 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d606)
		} else {
			r73 := ctx.AllocRegExcept(d603.Reg, d605.Reg)
			ctx.EmitMovRegReg(r73, d603.Reg)
			ctx.EmitImulInt64(r73, d605.Reg)
			d606 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
			ctx.BindReg(r73, &d606)
		}
		if d606.Loc == scm.LocReg && d603.Loc == scm.LocReg && d606.Reg == d603.Reg {
			ctx.TransferReg(d603.Reg)
			d603.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d603)
		ctx.FreeDesc(&d605)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d606)
		var d607 scm.JITValueDesc
		if d606.Loc == scm.LocImm {
			d607 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d606.Imm.Int() / 64)}
		} else {
			r74 := ctx.AllocRegExcept(d606.Reg)
			ctx.EmitMovRegReg(r74, d606.Reg)
			ctx.EmitShrRegImm8(r74, 6)
			d607 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r74}
			ctx.BindReg(r74, &d607)
		}
		if d607.Loc == scm.LocReg && d606.Loc == scm.LocReg && d607.Reg == d606.Reg {
			ctx.TransferReg(d606.Reg)
			d606.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d606)
		var d608 scm.JITValueDesc
		if d606.Loc == scm.LocImm {
			d608 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d606.Imm.Int() % 64)}
		} else {
			r75 := ctx.AllocRegExcept(d606.Reg)
			ctx.EmitMovRegReg(r75, d606.Reg)
			ctx.EmitAndRegImm32(r75, 63)
			d608 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
			ctx.BindReg(r75, &d608)
		}
		if d608.Loc == scm.LocReg && d606.Loc == scm.LocReg && d608.Reg == d606.Reg {
			ctx.TransferReg(d606.Reg)
			d606.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d606)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d609 scm.JITValueDesc
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
		d609 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r76, Reg2: r77, Reg3: r78}
		ctx.BindReg(r76, &d609)
		ctx.BindReg(r77, &d609)
		ctx.BindReg(r78, &d609)
		ctx.BindReg(r76, &d609)
		ctx.BindReg(r77, &d609)
		ctx.BindReg(r78, &d609)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d607)
		ctx.ReclaimUntrackedRegs()
		d611 = ctx.EmitSliceElementAddress(&d609, &d607, 8)
		ctx.EnsureDesc(&d611)
		ctx.EmitMovRegMem(d611.Reg, d611.Reg, 0)
		d610 = d611
		d610.Type = scm.TagInt
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d610)
		ctx.EnsureDesc(&d608)
		ctx.EnsureDescsTogether(&d610, &d608)
		var d612 scm.JITValueDesc
		if d610.Loc == scm.LocImm && d608.Loc == scm.LocImm {
			d612 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d610.Imm.Int()) << uint64(d608.Imm.Int())))}
		} else if d608.Loc == scm.LocImm {
			r79 := ctx.AllocRegExcept(d610.Reg)
			ctx.EmitMovRegReg(r79, d610.Reg)
			ctx.EmitShlRegImm8(r79, uint8(d608.Imm.Int()))
			d612 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r79}
			ctx.BindReg(r79, &d612)
		} else {
			{
				shiftSrc := d610.Reg
				r80 := ctx.AllocRegExcept(d610.Reg, d608.Reg)
				ctx.EmitMovRegReg(r80, d610.Reg)
				shiftSrc = r80
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d608.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d608.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d608.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d612 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d612)
			}
		}
		if d612.Loc == scm.LocReg && d610.Loc == scm.LocReg && d612.Reg == d610.Reg {
			ctx.TransferReg(d610.Reg)
			d610.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d610)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d607)
		ctx.EnsureDesc(&d607)
		var d613 scm.JITValueDesc
		if d607.Loc == scm.LocImm {
			d613 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d607.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d607.Reg)
			ctx.EmitMovRegReg(scratch, d607.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d613 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d613)
		}
		if d613.Loc == scm.LocReg && d607.Loc == scm.LocReg && d613.Reg == d607.Reg {
			ctx.TransferReg(d607.Reg)
			d607.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d607)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d613)
		ctx.ReclaimUntrackedRegs()
		d615 = ctx.EmitSliceElementAddress(&d609, &d613, 8)
		ctx.EnsureDesc(&d615)
		ctx.EmitMovRegMem(d615.Reg, d615.Reg, 0)
		d614 = d615
		d614.Type = scm.TagInt
		ctx.FreeDesc(&d613)
		ctx.ReclaimUntrackedRegs()
		d616 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d608)
		ctx.EnsureDescsTogether(&d616, &d608)
		var d617 scm.JITValueDesc
		if d616.Loc == scm.LocImm && d608.Loc == scm.LocImm {
			d617 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d616.Imm.Int() - d608.Imm.Int())}
		} else if d608.Loc == scm.LocImm && d608.Imm.Int() == 0 {
			r81 := ctx.AllocRegExcept(d616.Reg)
			ctx.EmitMovRegReg(r81, d616.Reg)
			d617 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
			ctx.BindReg(r81, &d617)
		} else if d616.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d608.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d616.Imm.Int()))
			ctx.EmitSubInt64(scratch, d608.Reg)
			d617 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d617)
		} else if d608.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d616.Reg)
			ctx.EmitMovRegReg(scratch, d616.Reg)
			if d608.Imm.Int() >= -2147483648 && d608.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d608.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d608.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d617 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d617)
		} else {
			r82 := ctx.AllocRegExcept(d616.Reg, d608.Reg)
			ctx.EmitMovRegReg(r82, d616.Reg)
			ctx.EmitSubInt64(r82, d608.Reg)
			d617 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r82}
			ctx.BindReg(r82, &d617)
		}
		if d617.Loc == scm.LocReg && d616.Loc == scm.LocReg && d617.Reg == d616.Reg {
			ctx.TransferReg(d616.Reg)
			d616.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d608)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d614)
		ctx.EnsureDesc(&d617)
		ctx.EnsureDescsTogether(&d614, &d617)
		var d618 scm.JITValueDesc
		if d614.Loc == scm.LocImm && d617.Loc == scm.LocImm {
			d618 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d614.Imm.Int()) >> uint64(d617.Imm.Int())))}
		} else if d617.Loc == scm.LocImm {
			r83 := ctx.AllocRegExcept(d614.Reg)
			ctx.EmitMovRegReg(r83, d614.Reg)
			ctx.EmitShrRegImm8(r83, uint8(d617.Imm.Int()))
			d618 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
			ctx.BindReg(r83, &d618)
		} else {
			{
				shiftSrc := d614.Reg
				r84 := ctx.AllocRegExcept(d614.Reg, d617.Reg)
				ctx.EmitMovRegReg(r84, d614.Reg)
				shiftSrc = r84
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d617.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d617.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d617.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d618 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d618)
			}
		}
		if d618.Loc == scm.LocReg && d614.Loc == scm.LocReg && d618.Reg == d614.Reg {
			ctx.TransferReg(d614.Reg)
			d614.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d614)
		ctx.FreeDesc(&d617)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d612)
		ctx.EnsureDesc(&d618)
		var d619 scm.JITValueDesc
		if d612.Loc == scm.LocImm && d618.Loc == scm.LocImm {
			d619 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d612.Imm.Int() | d618.Imm.Int())}
		} else if d612.Loc == scm.LocImm && d612.Imm.Int() == 0 {
			d619 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d618.Reg}
			ctx.BindReg(d618.Reg, &d619)
		} else if d618.Loc == scm.LocImm && d618.Imm.Int() == 0 {
			r85 := ctx.AllocRegExcept(d612.Reg)
			ctx.EmitMovRegReg(r85, d612.Reg)
			d619 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
			ctx.BindReg(r85, &d619)
		} else if d612.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d618.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d612.Imm.Int()))
			ctx.EmitOrInt64(scratch, d618.Reg)
			d619 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d619)
		} else if d618.Loc == scm.LocImm {
			r86 := ctx.AllocRegExcept(d612.Reg)
			ctx.EmitMovRegReg(r86, d612.Reg)
			if d618.Imm.Int() >= -2147483648 && d618.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r86, int32(d618.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d618.Imm.Int()))
				ctx.EmitOrInt64(r86, scm.RegR11)
			}
			d619 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
			ctx.BindReg(r86, &d619)
		} else {
			r87 := ctx.AllocRegExcept(d612.Reg, d618.Reg)
			ctx.EmitMovRegReg(r87, d612.Reg)
			ctx.EmitOrInt64(r87, d618.Reg)
			d619 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r87}
			ctx.BindReg(r87, &d619)
		}
		if d619.Loc == scm.LocReg && d612.Loc == scm.LocReg && d619.Reg == d612.Reg {
			ctx.TransferReg(d612.Reg)
			d612.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d612)
		ctx.FreeDesc(&d618)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d620 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d620 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48)
			r88 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r88, thisptr.Reg, off)
			d620 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r88}
			ctx.BindReg(r88, &d620)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d620)
		ctx.EnsureDesc(&d620)
		var d621 scm.JITValueDesc
		if d620.Loc == scm.LocImm {
			d621 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d620.Imm.Int()))))}
		} else {
			r89 := ctx.AllocReg()
			ctx.EmitMovRegReg(r89, d620.Reg)
			ctx.EmitShlRegImm8(r89, 56)
			ctx.EmitShrRegImm8(r89, 56)
			d621 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
			ctx.BindReg(r89, &d621)
		}
		ctx.FreeDesc(&d620)
		ctx.ReclaimUntrackedRegs()
		d622 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d621)
		ctx.EnsureDescsTogether(&d622, &d621)
		var d623 scm.JITValueDesc
		if d622.Loc == scm.LocImm && d621.Loc == scm.LocImm {
			d623 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d622.Imm.Int() - d621.Imm.Int())}
		} else if d621.Loc == scm.LocImm && d621.Imm.Int() == 0 {
			r90 := ctx.AllocRegExcept(d622.Reg)
			ctx.EmitMovRegReg(r90, d622.Reg)
			d623 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
			ctx.BindReg(r90, &d623)
		} else if d622.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d621.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d622.Imm.Int()))
			ctx.EmitSubInt64(scratch, d621.Reg)
			d623 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d623)
		} else if d621.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d622.Reg)
			ctx.EmitMovRegReg(scratch, d622.Reg)
			if d621.Imm.Int() >= -2147483648 && d621.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d621.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d621.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d623 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d623)
		} else {
			r91 := ctx.AllocRegExcept(d622.Reg, d621.Reg)
			ctx.EmitMovRegReg(r91, d622.Reg)
			ctx.EmitSubInt64(r91, d621.Reg)
			d623 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r91}
			ctx.BindReg(r91, &d623)
		}
		if d623.Loc == scm.LocReg && d622.Loc == scm.LocReg && d623.Reg == d622.Reg {
			ctx.TransferReg(d622.Reg)
			d622.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d621)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d619)
		ctx.EnsureDesc(&d623)
		ctx.EnsureDescsTogether(&d619, &d623)
		var d624 scm.JITValueDesc
		if d619.Loc == scm.LocImm && d623.Loc == scm.LocImm {
			d624 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d619.Imm.Int()) >> uint64(d623.Imm.Int())))}
		} else if d623.Loc == scm.LocImm {
			r92 := ctx.AllocRegExcept(d619.Reg)
			ctx.EmitMovRegReg(r92, d619.Reg)
			ctx.EmitShrRegImm8(r92, uint8(d623.Imm.Int()))
			d624 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r92}
			ctx.BindReg(r92, &d624)
		} else {
			{
				shiftSrc := d619.Reg
				r93 := ctx.AllocRegExcept(d619.Reg, d623.Reg)
				ctx.EmitMovRegReg(r93, d619.Reg)
				shiftSrc = r93
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d623.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d623.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d623.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d624 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d624)
			}
		}
		if d624.Loc == scm.LocReg && d619.Loc == scm.LocReg && d624.Reg == d619.Reg {
			ctx.TransferReg(d619.Reg)
			d619.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d619)
		ctx.FreeDesc(&d623)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d624)
		ctx.EnsureDesc(&d624)
		ctx.EnsureDesc(&d624)
		var d625 scm.JITValueDesc
		if d624.Loc == scm.LocImm {
			d625 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d624.Imm.Int()))))}
		} else {
			r94 := ctx.AllocReg()
			ctx.EmitMovRegReg(r94, d624.Reg)
			d625 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
			ctx.BindReg(r94, &d625)
		}
		ctx.FreeDesc(&d624)
		var d626 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d626 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56)
			r95 := ctx.AllocReg()
			ctx.EmitMovRegMem(r95, thisptr.Reg, off)
			d626 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r95}
			ctx.BindReg(r95, &d626)
		}
		ctx.EnsureDesc(&d625)
		ctx.EnsureDesc(&d626)
		ctx.EnsureDescsTogether(&d625, &d626)
		var d627 scm.JITValueDesc
		if d625.Loc == scm.LocImm && d626.Loc == scm.LocImm {
			d627 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d625.Imm.Int() + d626.Imm.Int())}
		} else if d626.Loc == scm.LocImm && d626.Imm.Int() == 0 {
			r96 := ctx.AllocRegExcept(d625.Reg)
			ctx.EmitMovRegReg(r96, d625.Reg)
			d627 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
			ctx.BindReg(r96, &d627)
		} else if d625.Loc == scm.LocImm && d625.Imm.Int() == 0 {
			d627 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d626.Reg}
			ctx.BindReg(d626.Reg, &d627)
		} else if d625.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d626.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d625.Imm.Int()))
			ctx.EmitAddInt64(scratch, d626.Reg)
			d627 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d627)
		} else if d626.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d625.Reg)
			ctx.EmitMovRegReg(scratch, d625.Reg)
			if d626.Imm.Int() >= -2147483648 && d626.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d626.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d626.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d627 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d627)
		} else {
			r97 := ctx.AllocRegExcept(d625.Reg, d626.Reg)
			ctx.EmitMovRegReg(r97, d625.Reg)
			ctx.EmitAddInt64(r97, d626.Reg)
			d627 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
			ctx.BindReg(r97, &d627)
		}
		if d627.Loc == scm.LocReg && d625.Loc == scm.LocReg && d627.Reg == d625.Reg {
			ctx.TransferReg(d625.Reg)
			d625.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d625)
		ctx.FreeDesc(&d626)
		ctx.EnsureDesc(&d627)
		ctx.EnsureDesc(&d627)
		var d628 scm.JITValueDesc
		if d627.Loc == scm.LocImm {
			d628 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d627.Imm.Int()))))}
		} else {
			r98 := ctx.AllocReg()
			ctx.EmitMovRegReg(r98, d627.Reg)
			ctx.EmitShlRegImm8(r98, 32)
			ctx.EmitShrRegImm8(r98, 32)
			d628 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r98}
			ctx.BindReg(r98, &d628)
		}
		ctx.FreeDesc(&d627)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d628)
		ctx.EnsureDescsTogether(&idxInt, &d628)
		var d629 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm && d628.Loc == scm.LocImm {
			d629 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d628.Imm.Int()))}
		} else if d628.Loc == scm.LocImm {
			r99 := ctx.AllocRegExcept(idxInt.Reg)
			if d628.Imm.Int() >= -2147483648 && d628.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(idxInt.Reg, int32(d628.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d628.Imm.Int()))
				ctx.EmitCmpInt64(idxInt.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r99, scm.CondUnsignedBelow)
			d629 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r99}
			ctx.BindReg(r99, &d629)
		} else if idxInt.Loc == scm.LocImm {
			r100 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d628.Reg)
			ctx.EmitSetcc(r100, scm.CondUnsignedBelow)
			d629 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r100}
			ctx.BindReg(r100, &d629)
		} else {
			r101 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitCmpInt64(idxInt.Reg, d628.Reg)
			ctx.EmitSetcc(r101, scm.CondUnsignedBelow)
			d629 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r101}
			ctx.BindReg(r101, &d629)
		}
		ctx.FreeDesc(&d628)
		d630 = d629
		ctx.EnsureDesc(&d630)
		if d630.Loc != scm.LocImm && d630.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d630.Loc == scm.LocImm {
			if d630.Imm.Bool() {
				if ps.General {
				}
				ps631 := scm.PhiState{General: ps.General}
				ps631.OverlayValues = make([]scm.JITValueDesc, 631)
				ps631.OverlayValues[1] = d1
				ps631.OverlayValues[2] = d2
				ps631.OverlayValues[3] = d3
				ps631.OverlayValues[4] = d4
				ps631.OverlayValues[5] = d5
				ps631.OverlayValues[6] = d6
				ps631.OverlayValues[7] = d7
				ps631.OverlayValues[8] = d8
				ps631.OverlayValues[9] = d9
				ps631.OverlayValues[10] = d10
				ps631.OverlayValues[11] = d11
				ps631.OverlayValues[12] = d12
				ps631.OverlayValues[13] = d13
				ps631.OverlayValues[14] = d14
				ps631.OverlayValues[15] = d15
				ps631.OverlayValues[17] = d17
				ps631.OverlayValues[18] = d18
				ps631.OverlayValues[19] = d19
				ps631.OverlayValues[20] = d20
				ps631.OverlayValues[21] = d21
				ps631.OverlayValues[22] = d22
				ps631.OverlayValues[23] = d23
				ps631.OverlayValues[24] = d24
				ps631.OverlayValues[25] = d25
				ps631.OverlayValues[26] = d26
				ps631.OverlayValues[27] = d27
				ps631.OverlayValues[28] = d28
				ps631.OverlayValues[29] = d29
				ps631.OverlayValues[30] = d30
				ps631.OverlayValues[31] = d31
				ps631.OverlayValues[32] = d32
				ps631.OverlayValues[33] = d33
				ps631.OverlayValues[34] = d34
				ps631.OverlayValues[35] = d35
				ps631.OverlayValues[36] = d36
				ps631.OverlayValues[37] = d37
				ps631.OverlayValues[38] = d38
				ps631.OverlayValues[39] = d39
				ps631.OverlayValues[40] = d40
				ps631.OverlayValues[41] = d41
				ps631.OverlayValues[42] = d42
				ps631.OverlayValues[43] = d43
				ps631.OverlayValues[44] = d44
				ps631.OverlayValues[45] = d45
				ps631.OverlayValues[46] = d46
				ps631.OverlayValues[47] = d47
				ps631.OverlayValues[48] = d48
				ps631.OverlayValues[49] = d49
				ps631.OverlayValues[50] = d50
				ps631.OverlayValues[53] = d53
				ps631.OverlayValues[54] = d54
				ps631.OverlayValues[55] = d55
				ps631.OverlayValues[164] = d164
				ps631.OverlayValues[165] = d165
				ps631.OverlayValues[166] = d166
				ps631.OverlayValues[167] = d167
				ps631.OverlayValues[168] = d168
				ps631.OverlayValues[169] = d169
				ps631.OverlayValues[170] = d170
				ps631.OverlayValues[171] = d171
				ps631.OverlayValues[172] = d172
				ps631.OverlayValues[173] = d173
				ps631.OverlayValues[174] = d174
				ps631.OverlayValues[175] = d175
				ps631.OverlayValues[176] = d176
				ps631.OverlayValues[177] = d177
				ps631.OverlayValues[178] = d178
				ps631.OverlayValues[179] = d179
				ps631.OverlayValues[180] = d180
				ps631.OverlayValues[181] = d181
				ps631.OverlayValues[182] = d182
				ps631.OverlayValues[183] = d183
				ps631.OverlayValues[184] = d184
				ps631.OverlayValues[185] = d185
				ps631.OverlayValues[186] = d186
				ps631.OverlayValues[187] = d187
				ps631.OverlayValues[188] = d188
				ps631.OverlayValues[189] = d189
				ps631.OverlayValues[190] = d190
				ps631.OverlayValues[191] = d191
				ps631.OverlayValues[192] = d192
				ps631.OverlayValues[193] = d193
				ps631.OverlayValues[196] = d196
				ps631.OverlayValues[367] = d367
				ps631.OverlayValues[368] = d368
				ps631.OverlayValues[369] = d369
				ps631.OverlayValues[370] = d370
				ps631.OverlayValues[372] = d372
				ps631.OverlayValues[373] = d373
				ps631.OverlayValues[374] = d374
				ps631.OverlayValues[375] = d375
				ps631.OverlayValues[376] = d376
				ps631.OverlayValues[377] = d377
				ps631.OverlayValues[378] = d378
				ps631.OverlayValues[379] = d379
				ps631.OverlayValues[381] = d381
				ps631.OverlayValues[383] = d383
				ps631.OverlayValues[384] = d384
				ps631.OverlayValues[385] = d385
				ps631.OverlayValues[486] = d486
				ps631.OverlayValues[487] = d487
				ps631.OverlayValues[490] = d490
				ps631.OverlayValues[594] = d594
				ps631.OverlayValues[595] = d595
				ps631.OverlayValues[596] = d596
				ps631.OverlayValues[597] = d597
				ps631.OverlayValues[598] = d598
				ps631.OverlayValues[600] = d600
				ps631.OverlayValues[601] = d601
				ps631.OverlayValues[602] = d602
				ps631.OverlayValues[603] = d603
				ps631.OverlayValues[604] = d604
				ps631.OverlayValues[605] = d605
				ps631.OverlayValues[606] = d606
				ps631.OverlayValues[607] = d607
				ps631.OverlayValues[608] = d608
				ps631.OverlayValues[609] = d609
				ps631.OverlayValues[610] = d610
				ps631.OverlayValues[611] = d611
				ps631.OverlayValues[612] = d612
				ps631.OverlayValues[613] = d613
				ps631.OverlayValues[614] = d614
				ps631.OverlayValues[615] = d615
				ps631.OverlayValues[616] = d616
				ps631.OverlayValues[617] = d617
				ps631.OverlayValues[618] = d618
				ps631.OverlayValues[619] = d619
				ps631.OverlayValues[620] = d620
				ps631.OverlayValues[621] = d621
				ps631.OverlayValues[622] = d622
				ps631.OverlayValues[623] = d623
				ps631.OverlayValues[624] = d624
				ps631.OverlayValues[625] = d625
				ps631.OverlayValues[626] = d626
				ps631.OverlayValues[627] = d627
				ps631.OverlayValues[628] = d628
				ps631.OverlayValues[629] = d629
				ps631.OverlayValues[630] = d630
				return bbs[7].RenderPS(ps631)
			}
			if ps.General {
			}
			ps632 := scm.PhiState{General: ps.General}
			ps632.OverlayValues = make([]scm.JITValueDesc, 631)
			ps632.OverlayValues[1] = d1
			ps632.OverlayValues[2] = d2
			ps632.OverlayValues[3] = d3
			ps632.OverlayValues[4] = d4
			ps632.OverlayValues[5] = d5
			ps632.OverlayValues[6] = d6
			ps632.OverlayValues[7] = d7
			ps632.OverlayValues[8] = d8
			ps632.OverlayValues[9] = d9
			ps632.OverlayValues[10] = d10
			ps632.OverlayValues[11] = d11
			ps632.OverlayValues[12] = d12
			ps632.OverlayValues[13] = d13
			ps632.OverlayValues[14] = d14
			ps632.OverlayValues[15] = d15
			ps632.OverlayValues[17] = d17
			ps632.OverlayValues[18] = d18
			ps632.OverlayValues[19] = d19
			ps632.OverlayValues[20] = d20
			ps632.OverlayValues[21] = d21
			ps632.OverlayValues[22] = d22
			ps632.OverlayValues[23] = d23
			ps632.OverlayValues[24] = d24
			ps632.OverlayValues[25] = d25
			ps632.OverlayValues[26] = d26
			ps632.OverlayValues[27] = d27
			ps632.OverlayValues[28] = d28
			ps632.OverlayValues[29] = d29
			ps632.OverlayValues[30] = d30
			ps632.OverlayValues[31] = d31
			ps632.OverlayValues[32] = d32
			ps632.OverlayValues[33] = d33
			ps632.OverlayValues[34] = d34
			ps632.OverlayValues[35] = d35
			ps632.OverlayValues[36] = d36
			ps632.OverlayValues[37] = d37
			ps632.OverlayValues[38] = d38
			ps632.OverlayValues[39] = d39
			ps632.OverlayValues[40] = d40
			ps632.OverlayValues[41] = d41
			ps632.OverlayValues[42] = d42
			ps632.OverlayValues[43] = d43
			ps632.OverlayValues[44] = d44
			ps632.OverlayValues[45] = d45
			ps632.OverlayValues[46] = d46
			ps632.OverlayValues[47] = d47
			ps632.OverlayValues[48] = d48
			ps632.OverlayValues[49] = d49
			ps632.OverlayValues[50] = d50
			ps632.OverlayValues[53] = d53
			ps632.OverlayValues[54] = d54
			ps632.OverlayValues[55] = d55
			ps632.OverlayValues[164] = d164
			ps632.OverlayValues[165] = d165
			ps632.OverlayValues[166] = d166
			ps632.OverlayValues[167] = d167
			ps632.OverlayValues[168] = d168
			ps632.OverlayValues[169] = d169
			ps632.OverlayValues[170] = d170
			ps632.OverlayValues[171] = d171
			ps632.OverlayValues[172] = d172
			ps632.OverlayValues[173] = d173
			ps632.OverlayValues[174] = d174
			ps632.OverlayValues[175] = d175
			ps632.OverlayValues[176] = d176
			ps632.OverlayValues[177] = d177
			ps632.OverlayValues[178] = d178
			ps632.OverlayValues[179] = d179
			ps632.OverlayValues[180] = d180
			ps632.OverlayValues[181] = d181
			ps632.OverlayValues[182] = d182
			ps632.OverlayValues[183] = d183
			ps632.OverlayValues[184] = d184
			ps632.OverlayValues[185] = d185
			ps632.OverlayValues[186] = d186
			ps632.OverlayValues[187] = d187
			ps632.OverlayValues[188] = d188
			ps632.OverlayValues[189] = d189
			ps632.OverlayValues[190] = d190
			ps632.OverlayValues[191] = d191
			ps632.OverlayValues[192] = d192
			ps632.OverlayValues[193] = d193
			ps632.OverlayValues[196] = d196
			ps632.OverlayValues[367] = d367
			ps632.OverlayValues[368] = d368
			ps632.OverlayValues[369] = d369
			ps632.OverlayValues[370] = d370
			ps632.OverlayValues[372] = d372
			ps632.OverlayValues[373] = d373
			ps632.OverlayValues[374] = d374
			ps632.OverlayValues[375] = d375
			ps632.OverlayValues[376] = d376
			ps632.OverlayValues[377] = d377
			ps632.OverlayValues[378] = d378
			ps632.OverlayValues[379] = d379
			ps632.OverlayValues[381] = d381
			ps632.OverlayValues[383] = d383
			ps632.OverlayValues[384] = d384
			ps632.OverlayValues[385] = d385
			ps632.OverlayValues[486] = d486
			ps632.OverlayValues[487] = d487
			ps632.OverlayValues[490] = d490
			ps632.OverlayValues[594] = d594
			ps632.OverlayValues[595] = d595
			ps632.OverlayValues[596] = d596
			ps632.OverlayValues[597] = d597
			ps632.OverlayValues[598] = d598
			ps632.OverlayValues[600] = d600
			ps632.OverlayValues[601] = d601
			ps632.OverlayValues[602] = d602
			ps632.OverlayValues[603] = d603
			ps632.OverlayValues[604] = d604
			ps632.OverlayValues[605] = d605
			ps632.OverlayValues[606] = d606
			ps632.OverlayValues[607] = d607
			ps632.OverlayValues[608] = d608
			ps632.OverlayValues[609] = d609
			ps632.OverlayValues[610] = d610
			ps632.OverlayValues[611] = d611
			ps632.OverlayValues[612] = d612
			ps632.OverlayValues[613] = d613
			ps632.OverlayValues[614] = d614
			ps632.OverlayValues[615] = d615
			ps632.OverlayValues[616] = d616
			ps632.OverlayValues[617] = d617
			ps632.OverlayValues[618] = d618
			ps632.OverlayValues[619] = d619
			ps632.OverlayValues[620] = d620
			ps632.OverlayValues[621] = d621
			ps632.OverlayValues[622] = d622
			ps632.OverlayValues[623] = d623
			ps632.OverlayValues[624] = d624
			ps632.OverlayValues[625] = d625
			ps632.OverlayValues[626] = d626
			ps632.OverlayValues[627] = d627
			ps632.OverlayValues[628] = d628
			ps632.OverlayValues[629] = d629
			ps632.OverlayValues[630] = d630
			return bbs[9].RenderPS(ps632)
		}
		if !ps.General {
			ps.General = true
			return bbs[6].RenderPS(ps)
		}
		lbl24 := ctx.ReserveLabel()
		lbl25 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d630.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl24)
		ctx.EmitJmp(lbl25)
		snap633 := d1
		snap634 := d2
		snap635 := d3
		snap636 := d4
		snap637 := d5
		snap638 := d6
		snap639 := d7
		snap640 := d8
		snap641 := d9
		snap642 := d10
		snap643 := d11
		snap644 := d12
		snap645 := d13
		snap646 := d14
		snap647 := d15
		snap648 := d17
		snap649 := d18
		snap650 := d19
		snap651 := d20
		snap652 := d21
		snap653 := d22
		snap654 := d23
		snap655 := d24
		snap656 := d25
		snap657 := d26
		snap658 := d27
		snap659 := d28
		snap660 := d29
		snap661 := d30
		snap662 := d31
		snap663 := d32
		snap664 := d33
		snap665 := d34
		snap666 := d35
		snap667 := d36
		snap668 := d37
		snap669 := d38
		snap670 := d39
		snap671 := d40
		snap672 := d41
		snap673 := d42
		snap674 := d43
		snap675 := d44
		snap676 := d45
		snap677 := d46
		snap678 := d47
		snap679 := d48
		snap680 := d49
		snap681 := d50
		snap682 := d53
		snap683 := d54
		snap684 := d55
		snap685 := d164
		snap686 := d165
		snap687 := d166
		snap688 := d167
		snap689 := d168
		snap690 := d169
		snap691 := d170
		snap692 := d171
		snap693 := d172
		snap694 := d173
		snap695 := d174
		snap696 := d175
		snap697 := d176
		snap698 := d177
		snap699 := d178
		snap700 := d179
		snap701 := d180
		snap702 := d181
		snap703 := d182
		snap704 := d183
		snap705 := d184
		snap706 := d185
		snap707 := d186
		snap708 := d187
		snap709 := d188
		snap710 := d189
		snap711 := d190
		snap712 := d191
		snap713 := d192
		snap714 := d193
		snap715 := d196
		snap716 := d367
		snap717 := d368
		snap718 := d369
		snap719 := d370
		snap720 := d372
		snap721 := d373
		snap722 := d374
		snap723 := d375
		snap724 := d376
		snap725 := d377
		snap726 := d378
		snap727 := d379
		snap728 := d381
		snap729 := d383
		snap730 := d384
		snap731 := d385
		snap732 := d486
		snap733 := d487
		snap734 := d490
		snap735 := d594
		snap736 := d595
		snap737 := d596
		snap738 := d597
		snap739 := d598
		snap740 := d600
		snap741 := d601
		snap742 := d602
		snap743 := d603
		snap744 := d604
		snap745 := d605
		snap746 := d606
		snap747 := d607
		snap748 := d608
		snap749 := d609
		snap750 := d610
		snap751 := d611
		snap752 := d612
		snap753 := d613
		snap754 := d614
		snap755 := d615
		snap756 := d616
		snap757 := d617
		snap758 := d618
		snap759 := d619
		snap760 := d620
		snap761 := d621
		snap762 := d622
		snap763 := d623
		snap764 := d624
		snap765 := d625
		snap766 := d626
		snap767 := d627
		snap768 := d628
		snap769 := d629
		snap770 := d630
		alloc771 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl24)
		ctx.EmitJmp(lbl8)
		ctx.RestoreAllocState(alloc771)
		d1 = snap633
		d2 = snap634
		d3 = snap635
		d4 = snap636
		d5 = snap637
		d6 = snap638
		d7 = snap639
		d8 = snap640
		d9 = snap641
		d10 = snap642
		d11 = snap643
		d12 = snap644
		d13 = snap645
		d14 = snap646
		d15 = snap647
		d17 = snap648
		d18 = snap649
		d19 = snap650
		d20 = snap651
		d21 = snap652
		d22 = snap653
		d23 = snap654
		d24 = snap655
		d25 = snap656
		d26 = snap657
		d27 = snap658
		d28 = snap659
		d29 = snap660
		d30 = snap661
		d31 = snap662
		d32 = snap663
		d33 = snap664
		d34 = snap665
		d35 = snap666
		d36 = snap667
		d37 = snap668
		d38 = snap669
		d39 = snap670
		d40 = snap671
		d41 = snap672
		d42 = snap673
		d43 = snap674
		d44 = snap675
		d45 = snap676
		d46 = snap677
		d47 = snap678
		d48 = snap679
		d49 = snap680
		d50 = snap681
		d53 = snap682
		d54 = snap683
		d55 = snap684
		d164 = snap685
		d165 = snap686
		d166 = snap687
		d167 = snap688
		d168 = snap689
		d169 = snap690
		d170 = snap691
		d171 = snap692
		d172 = snap693
		d173 = snap694
		d174 = snap695
		d175 = snap696
		d176 = snap697
		d177 = snap698
		d178 = snap699
		d179 = snap700
		d180 = snap701
		d181 = snap702
		d182 = snap703
		d183 = snap704
		d184 = snap705
		d185 = snap706
		d186 = snap707
		d187 = snap708
		d188 = snap709
		d189 = snap710
		d190 = snap711
		d191 = snap712
		d192 = snap713
		d193 = snap714
		d196 = snap715
		d367 = snap716
		d368 = snap717
		d369 = snap718
		d370 = snap719
		d372 = snap720
		d373 = snap721
		d374 = snap722
		d375 = snap723
		d376 = snap724
		d377 = snap725
		d378 = snap726
		d379 = snap727
		d381 = snap728
		d383 = snap729
		d384 = snap730
		d385 = snap731
		d486 = snap732
		d487 = snap733
		d490 = snap734
		d594 = snap735
		d595 = snap736
		d596 = snap737
		d597 = snap738
		d598 = snap739
		d600 = snap740
		d601 = snap741
		d602 = snap742
		d603 = snap743
		d604 = snap744
		d605 = snap745
		d606 = snap746
		d607 = snap747
		d608 = snap748
		d609 = snap749
		d610 = snap750
		d611 = snap751
		d612 = snap752
		d613 = snap753
		d614 = snap754
		d615 = snap755
		d616 = snap756
		d617 = snap757
		d618 = snap758
		d619 = snap759
		d620 = snap760
		d621 = snap761
		d622 = snap762
		d623 = snap763
		d624 = snap764
		d625 = snap765
		d626 = snap766
		d627 = snap767
		d628 = snap768
		d629 = snap769
		d630 = snap770
		ctx.MarkLabel(lbl25)
		ctx.EmitJmp(lbl10)
		ctx.RestoreAllocState(alloc771)
		d1 = snap633
		d2 = snap634
		d3 = snap635
		d4 = snap636
		d5 = snap637
		d6 = snap638
		d7 = snap639
		d8 = snap640
		d9 = snap641
		d10 = snap642
		d11 = snap643
		d12 = snap644
		d13 = snap645
		d14 = snap646
		d15 = snap647
		d17 = snap648
		d18 = snap649
		d19 = snap650
		d20 = snap651
		d21 = snap652
		d22 = snap653
		d23 = snap654
		d24 = snap655
		d25 = snap656
		d26 = snap657
		d27 = snap658
		d28 = snap659
		d29 = snap660
		d30 = snap661
		d31 = snap662
		d32 = snap663
		d33 = snap664
		d34 = snap665
		d35 = snap666
		d36 = snap667
		d37 = snap668
		d38 = snap669
		d39 = snap670
		d40 = snap671
		d41 = snap672
		d42 = snap673
		d43 = snap674
		d44 = snap675
		d45 = snap676
		d46 = snap677
		d47 = snap678
		d48 = snap679
		d49 = snap680
		d50 = snap681
		d53 = snap682
		d54 = snap683
		d55 = snap684
		d164 = snap685
		d165 = snap686
		d166 = snap687
		d167 = snap688
		d168 = snap689
		d169 = snap690
		d170 = snap691
		d171 = snap692
		d172 = snap693
		d173 = snap694
		d174 = snap695
		d175 = snap696
		d176 = snap697
		d177 = snap698
		d178 = snap699
		d179 = snap700
		d180 = snap701
		d181 = snap702
		d182 = snap703
		d183 = snap704
		d184 = snap705
		d185 = snap706
		d186 = snap707
		d187 = snap708
		d188 = snap709
		d189 = snap710
		d190 = snap711
		d191 = snap712
		d192 = snap713
		d193 = snap714
		d196 = snap715
		d367 = snap716
		d368 = snap717
		d369 = snap718
		d370 = snap719
		d372 = snap720
		d373 = snap721
		d374 = snap722
		d375 = snap723
		d376 = snap724
		d377 = snap725
		d378 = snap726
		d379 = snap727
		d381 = snap728
		d383 = snap729
		d384 = snap730
		d385 = snap731
		d486 = snap732
		d487 = snap733
		d490 = snap734
		d594 = snap735
		d595 = snap736
		d596 = snap737
		d597 = snap738
		d598 = snap739
		d600 = snap740
		d601 = snap741
		d602 = snap742
		d603 = snap743
		d604 = snap744
		d605 = snap745
		d606 = snap746
		d607 = snap747
		d608 = snap748
		d609 = snap749
		d610 = snap750
		d611 = snap751
		d612 = snap752
		d613 = snap753
		d614 = snap754
		d615 = snap755
		d616 = snap756
		d617 = snap757
		d618 = snap758
		d619 = snap759
		d620 = snap760
		d621 = snap761
		d622 = snap762
		d623 = snap763
		d624 = snap764
		d625 = snap765
		d626 = snap766
		d627 = snap767
		d628 = snap768
		d629 = snap769
		d630 = snap770
		ps772 := scm.PhiState{General: true}
		ps772.OverlayValues = make([]scm.JITValueDesc, 631)
		ps772.OverlayValues[1] = d1
		ps772.OverlayValues[2] = d2
		ps772.OverlayValues[3] = d3
		ps772.OverlayValues[4] = d4
		ps772.OverlayValues[5] = d5
		ps772.OverlayValues[6] = d6
		ps772.OverlayValues[7] = d7
		ps772.OverlayValues[8] = d8
		ps772.OverlayValues[9] = d9
		ps772.OverlayValues[10] = d10
		ps772.OverlayValues[11] = d11
		ps772.OverlayValues[12] = d12
		ps772.OverlayValues[13] = d13
		ps772.OverlayValues[14] = d14
		ps772.OverlayValues[15] = d15
		ps772.OverlayValues[17] = d17
		ps772.OverlayValues[18] = d18
		ps772.OverlayValues[19] = d19
		ps772.OverlayValues[20] = d20
		ps772.OverlayValues[21] = d21
		ps772.OverlayValues[22] = d22
		ps772.OverlayValues[23] = d23
		ps772.OverlayValues[24] = d24
		ps772.OverlayValues[25] = d25
		ps772.OverlayValues[26] = d26
		ps772.OverlayValues[27] = d27
		ps772.OverlayValues[28] = d28
		ps772.OverlayValues[29] = d29
		ps772.OverlayValues[30] = d30
		ps772.OverlayValues[31] = d31
		ps772.OverlayValues[32] = d32
		ps772.OverlayValues[33] = d33
		ps772.OverlayValues[34] = d34
		ps772.OverlayValues[35] = d35
		ps772.OverlayValues[36] = d36
		ps772.OverlayValues[37] = d37
		ps772.OverlayValues[38] = d38
		ps772.OverlayValues[39] = d39
		ps772.OverlayValues[40] = d40
		ps772.OverlayValues[41] = d41
		ps772.OverlayValues[42] = d42
		ps772.OverlayValues[43] = d43
		ps772.OverlayValues[44] = d44
		ps772.OverlayValues[45] = d45
		ps772.OverlayValues[46] = d46
		ps772.OverlayValues[47] = d47
		ps772.OverlayValues[48] = d48
		ps772.OverlayValues[49] = d49
		ps772.OverlayValues[50] = d50
		ps772.OverlayValues[53] = d53
		ps772.OverlayValues[54] = d54
		ps772.OverlayValues[55] = d55
		ps772.OverlayValues[164] = d164
		ps772.OverlayValues[165] = d165
		ps772.OverlayValues[166] = d166
		ps772.OverlayValues[167] = d167
		ps772.OverlayValues[168] = d168
		ps772.OverlayValues[169] = d169
		ps772.OverlayValues[170] = d170
		ps772.OverlayValues[171] = d171
		ps772.OverlayValues[172] = d172
		ps772.OverlayValues[173] = d173
		ps772.OverlayValues[174] = d174
		ps772.OverlayValues[175] = d175
		ps772.OverlayValues[176] = d176
		ps772.OverlayValues[177] = d177
		ps772.OverlayValues[178] = d178
		ps772.OverlayValues[179] = d179
		ps772.OverlayValues[180] = d180
		ps772.OverlayValues[181] = d181
		ps772.OverlayValues[182] = d182
		ps772.OverlayValues[183] = d183
		ps772.OverlayValues[184] = d184
		ps772.OverlayValues[185] = d185
		ps772.OverlayValues[186] = d186
		ps772.OverlayValues[187] = d187
		ps772.OverlayValues[188] = d188
		ps772.OverlayValues[189] = d189
		ps772.OverlayValues[190] = d190
		ps772.OverlayValues[191] = d191
		ps772.OverlayValues[192] = d192
		ps772.OverlayValues[193] = d193
		ps772.OverlayValues[196] = d196
		ps772.OverlayValues[367] = d367
		ps772.OverlayValues[368] = d368
		ps772.OverlayValues[369] = d369
		ps772.OverlayValues[370] = d370
		ps772.OverlayValues[372] = d372
		ps772.OverlayValues[373] = d373
		ps772.OverlayValues[374] = d374
		ps772.OverlayValues[375] = d375
		ps772.OverlayValues[376] = d376
		ps772.OverlayValues[377] = d377
		ps772.OverlayValues[378] = d378
		ps772.OverlayValues[379] = d379
		ps772.OverlayValues[381] = d381
		ps772.OverlayValues[383] = d383
		ps772.OverlayValues[384] = d384
		ps772.OverlayValues[385] = d385
		ps772.OverlayValues[486] = d486
		ps772.OverlayValues[487] = d487
		ps772.OverlayValues[490] = d490
		ps772.OverlayValues[594] = d594
		ps772.OverlayValues[595] = d595
		ps772.OverlayValues[596] = d596
		ps772.OverlayValues[597] = d597
		ps772.OverlayValues[598] = d598
		ps772.OverlayValues[600] = d600
		ps772.OverlayValues[601] = d601
		ps772.OverlayValues[602] = d602
		ps772.OverlayValues[603] = d603
		ps772.OverlayValues[604] = d604
		ps772.OverlayValues[605] = d605
		ps772.OverlayValues[606] = d606
		ps772.OverlayValues[607] = d607
		ps772.OverlayValues[608] = d608
		ps772.OverlayValues[609] = d609
		ps772.OverlayValues[610] = d610
		ps772.OverlayValues[611] = d611
		ps772.OverlayValues[612] = d612
		ps772.OverlayValues[613] = d613
		ps772.OverlayValues[614] = d614
		ps772.OverlayValues[615] = d615
		ps772.OverlayValues[616] = d616
		ps772.OverlayValues[617] = d617
		ps772.OverlayValues[618] = d618
		ps772.OverlayValues[619] = d619
		ps772.OverlayValues[620] = d620
		ps772.OverlayValues[621] = d621
		ps772.OverlayValues[622] = d622
		ps772.OverlayValues[623] = d623
		ps772.OverlayValues[624] = d624
		ps772.OverlayValues[625] = d625
		ps772.OverlayValues[626] = d626
		ps772.OverlayValues[627] = d627
		ps772.OverlayValues[628] = d628
		ps772.OverlayValues[629] = d629
		ps772.OverlayValues[630] = d630
		ps773 := scm.PhiState{General: true}
		ps773.OverlayValues = make([]scm.JITValueDesc, 631)
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
		ps773.OverlayValues[50] = d50
		ps773.OverlayValues[53] = d53
		ps773.OverlayValues[54] = d54
		ps773.OverlayValues[55] = d55
		ps773.OverlayValues[164] = d164
		ps773.OverlayValues[165] = d165
		ps773.OverlayValues[166] = d166
		ps773.OverlayValues[167] = d167
		ps773.OverlayValues[168] = d168
		ps773.OverlayValues[169] = d169
		ps773.OverlayValues[170] = d170
		ps773.OverlayValues[171] = d171
		ps773.OverlayValues[172] = d172
		ps773.OverlayValues[173] = d173
		ps773.OverlayValues[174] = d174
		ps773.OverlayValues[175] = d175
		ps773.OverlayValues[176] = d176
		ps773.OverlayValues[177] = d177
		ps773.OverlayValues[178] = d178
		ps773.OverlayValues[179] = d179
		ps773.OverlayValues[180] = d180
		ps773.OverlayValues[181] = d181
		ps773.OverlayValues[182] = d182
		ps773.OverlayValues[183] = d183
		ps773.OverlayValues[184] = d184
		ps773.OverlayValues[185] = d185
		ps773.OverlayValues[186] = d186
		ps773.OverlayValues[187] = d187
		ps773.OverlayValues[188] = d188
		ps773.OverlayValues[189] = d189
		ps773.OverlayValues[190] = d190
		ps773.OverlayValues[191] = d191
		ps773.OverlayValues[192] = d192
		ps773.OverlayValues[193] = d193
		ps773.OverlayValues[196] = d196
		ps773.OverlayValues[367] = d367
		ps773.OverlayValues[368] = d368
		ps773.OverlayValues[369] = d369
		ps773.OverlayValues[370] = d370
		ps773.OverlayValues[372] = d372
		ps773.OverlayValues[373] = d373
		ps773.OverlayValues[374] = d374
		ps773.OverlayValues[375] = d375
		ps773.OverlayValues[376] = d376
		ps773.OverlayValues[377] = d377
		ps773.OverlayValues[378] = d378
		ps773.OverlayValues[379] = d379
		ps773.OverlayValues[381] = d381
		ps773.OverlayValues[383] = d383
		ps773.OverlayValues[384] = d384
		ps773.OverlayValues[385] = d385
		ps773.OverlayValues[486] = d486
		ps773.OverlayValues[487] = d487
		ps773.OverlayValues[490] = d490
		ps773.OverlayValues[594] = d594
		ps773.OverlayValues[595] = d595
		ps773.OverlayValues[596] = d596
		ps773.OverlayValues[597] = d597
		ps773.OverlayValues[598] = d598
		ps773.OverlayValues[600] = d600
		ps773.OverlayValues[601] = d601
		ps773.OverlayValues[602] = d602
		ps773.OverlayValues[603] = d603
		ps773.OverlayValues[604] = d604
		ps773.OverlayValues[605] = d605
		ps773.OverlayValues[606] = d606
		ps773.OverlayValues[607] = d607
		ps773.OverlayValues[608] = d608
		ps773.OverlayValues[609] = d609
		ps773.OverlayValues[610] = d610
		ps773.OverlayValues[611] = d611
		ps773.OverlayValues[612] = d612
		ps773.OverlayValues[613] = d613
		ps773.OverlayValues[614] = d614
		ps773.OverlayValues[615] = d615
		ps773.OverlayValues[616] = d616
		ps773.OverlayValues[617] = d617
		ps773.OverlayValues[618] = d618
		ps773.OverlayValues[619] = d619
		ps773.OverlayValues[620] = d620
		ps773.OverlayValues[621] = d621
		ps773.OverlayValues[622] = d622
		ps773.OverlayValues[623] = d623
		ps773.OverlayValues[624] = d624
		ps773.OverlayValues[625] = d625
		ps773.OverlayValues[626] = d626
		ps773.OverlayValues[627] = d627
		ps773.OverlayValues[628] = d628
		ps773.OverlayValues[629] = d629
		ps773.OverlayValues[630] = d630
		snap774 := d1
		snap775 := d2
		snap776 := d3
		snap777 := d4
		snap778 := d5
		snap779 := d6
		snap780 := d7
		snap781 := d8
		snap782 := d9
		snap783 := d10
		snap784 := d11
		snap785 := d12
		snap786 := d13
		snap787 := d14
		snap788 := d15
		snap789 := d17
		snap790 := d18
		snap791 := d19
		snap792 := d20
		snap793 := d21
		snap794 := d22
		snap795 := d23
		snap796 := d24
		snap797 := d25
		snap798 := d26
		snap799 := d27
		snap800 := d28
		snap801 := d29
		snap802 := d30
		snap803 := d31
		snap804 := d32
		snap805 := d33
		snap806 := d34
		snap807 := d35
		snap808 := d36
		snap809 := d37
		snap810 := d38
		snap811 := d39
		snap812 := d40
		snap813 := d41
		snap814 := d42
		snap815 := d43
		snap816 := d44
		snap817 := d45
		snap818 := d46
		snap819 := d47
		snap820 := d48
		snap821 := d49
		snap822 := d50
		snap823 := d53
		snap824 := d54
		snap825 := d55
		snap826 := d164
		snap827 := d165
		snap828 := d166
		snap829 := d167
		snap830 := d168
		snap831 := d169
		snap832 := d170
		snap833 := d171
		snap834 := d172
		snap835 := d173
		snap836 := d174
		snap837 := d175
		snap838 := d176
		snap839 := d177
		snap840 := d178
		snap841 := d179
		snap842 := d180
		snap843 := d181
		snap844 := d182
		snap845 := d183
		snap846 := d184
		snap847 := d185
		snap848 := d186
		snap849 := d187
		snap850 := d188
		snap851 := d189
		snap852 := d190
		snap853 := d191
		snap854 := d192
		snap855 := d193
		snap856 := d196
		snap857 := d367
		snap858 := d368
		snap859 := d369
		snap860 := d370
		snap861 := d372
		snap862 := d373
		snap863 := d374
		snap864 := d375
		snap865 := d376
		snap866 := d377
		snap867 := d378
		snap868 := d379
		snap869 := d381
		snap870 := d383
		snap871 := d384
		snap872 := d385
		snap873 := d486
		snap874 := d487
		snap875 := d490
		snap876 := d594
		snap877 := d595
		snap878 := d596
		snap879 := d597
		snap880 := d598
		snap881 := d600
		snap882 := d601
		snap883 := d602
		snap884 := d603
		snap885 := d604
		snap886 := d605
		snap887 := d606
		snap888 := d607
		snap889 := d608
		snap890 := d609
		snap891 := d610
		snap892 := d611
		snap893 := d612
		snap894 := d613
		snap895 := d614
		snap896 := d615
		snap897 := d616
		snap898 := d617
		snap899 := d618
		snap900 := d619
		snap901 := d620
		snap902 := d621
		snap903 := d622
		snap904 := d623
		snap905 := d624
		snap906 := d625
		snap907 := d626
		snap908 := d627
		snap909 := d628
		snap910 := d629
		snap911 := d630
		alloc912 := ctx.SnapshotAllocState()
		if !bbs[9].Rendered {
			bbs[9].RenderPS(ps773)
		}
		ctx.RestoreAllocState(alloc912)
		d1 = snap774
		d2 = snap775
		d3 = snap776
		d4 = snap777
		d5 = snap778
		d6 = snap779
		d7 = snap780
		d8 = snap781
		d9 = snap782
		d10 = snap783
		d11 = snap784
		d12 = snap785
		d13 = snap786
		d14 = snap787
		d15 = snap788
		d17 = snap789
		d18 = snap790
		d19 = snap791
		d20 = snap792
		d21 = snap793
		d22 = snap794
		d23 = snap795
		d24 = snap796
		d25 = snap797
		d26 = snap798
		d27 = snap799
		d28 = snap800
		d29 = snap801
		d30 = snap802
		d31 = snap803
		d32 = snap804
		d33 = snap805
		d34 = snap806
		d35 = snap807
		d36 = snap808
		d37 = snap809
		d38 = snap810
		d39 = snap811
		d40 = snap812
		d41 = snap813
		d42 = snap814
		d43 = snap815
		d44 = snap816
		d45 = snap817
		d46 = snap818
		d47 = snap819
		d48 = snap820
		d49 = snap821
		d50 = snap822
		d53 = snap823
		d54 = snap824
		d55 = snap825
		d164 = snap826
		d165 = snap827
		d166 = snap828
		d167 = snap829
		d168 = snap830
		d169 = snap831
		d170 = snap832
		d171 = snap833
		d172 = snap834
		d173 = snap835
		d174 = snap836
		d175 = snap837
		d176 = snap838
		d177 = snap839
		d178 = snap840
		d179 = snap841
		d180 = snap842
		d181 = snap843
		d182 = snap844
		d183 = snap845
		d184 = snap846
		d185 = snap847
		d186 = snap848
		d187 = snap849
		d188 = snap850
		d189 = snap851
		d190 = snap852
		d191 = snap853
		d192 = snap854
		d193 = snap855
		d196 = snap856
		d367 = snap857
		d368 = snap858
		d369 = snap859
		d370 = snap860
		d372 = snap861
		d373 = snap862
		d374 = snap863
		d375 = snap864
		d376 = snap865
		d377 = snap866
		d378 = snap867
		d379 = snap868
		d381 = snap869
		d383 = snap870
		d384 = snap871
		d385 = snap872
		d486 = snap873
		d487 = snap874
		d490 = snap875
		d594 = snap876
		d595 = snap877
		d596 = snap878
		d597 = snap879
		d598 = snap880
		d600 = snap881
		d601 = snap882
		d602 = snap883
		d603 = snap884
		d604 = snap885
		d605 = snap886
		d606 = snap887
		d607 = snap888
		d608 = snap889
		d609 = snap890
		d610 = snap891
		d611 = snap892
		d612 = snap893
		d613 = snap894
		d614 = snap895
		d615 = snap896
		d616 = snap897
		d617 = snap898
		d618 = snap899
		d619 = snap900
		d620 = snap901
		d621 = snap902
		d622 = snap903
		d623 = snap904
		d624 = snap905
		d625 = snap906
		d626 = snap907
		d627 = snap908
		d628 = snap909
		d629 = snap910
		d630 = snap911
		if !bbs[7].Rendered {
			return bbs[7].RenderPS(ps772)
		}
		return result
		ctx.FreeDesc(&d629)
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
		if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != scm.LocNone {
			d176 = ps.OverlayValues[176]
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
		if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
			d196 = ps.OverlayValues[196]
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
		if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != scm.LocNone {
			d381 = ps.OverlayValues[381]
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
		if len(ps.OverlayValues) > 486 && ps.OverlayValues[486].Loc != scm.LocNone {
			d486 = ps.OverlayValues[486]
		}
		if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != scm.LocNone {
			d487 = ps.OverlayValues[487]
		}
		if len(ps.OverlayValues) > 490 && ps.OverlayValues[490].Loc != scm.LocNone {
			d490 = ps.OverlayValues[490]
		}
		if len(ps.OverlayValues) > 594 && ps.OverlayValues[594].Loc != scm.LocNone {
			d594 = ps.OverlayValues[594]
		}
		if len(ps.OverlayValues) > 595 && ps.OverlayValues[595].Loc != scm.LocNone {
			d595 = ps.OverlayValues[595]
		}
		if len(ps.OverlayValues) > 596 && ps.OverlayValues[596].Loc != scm.LocNone {
			d596 = ps.OverlayValues[596]
		}
		if len(ps.OverlayValues) > 597 && ps.OverlayValues[597].Loc != scm.LocNone {
			d597 = ps.OverlayValues[597]
		}
		if len(ps.OverlayValues) > 598 && ps.OverlayValues[598].Loc != scm.LocNone {
			d598 = ps.OverlayValues[598]
		}
		if len(ps.OverlayValues) > 600 && ps.OverlayValues[600].Loc != scm.LocNone {
			d600 = ps.OverlayValues[600]
		}
		if len(ps.OverlayValues) > 601 && ps.OverlayValues[601].Loc != scm.LocNone {
			d601 = ps.OverlayValues[601]
		}
		if len(ps.OverlayValues) > 602 && ps.OverlayValues[602].Loc != scm.LocNone {
			d602 = ps.OverlayValues[602]
		}
		if len(ps.OverlayValues) > 603 && ps.OverlayValues[603].Loc != scm.LocNone {
			d603 = ps.OverlayValues[603]
		}
		if len(ps.OverlayValues) > 604 && ps.OverlayValues[604].Loc != scm.LocNone {
			d604 = ps.OverlayValues[604]
		}
		if len(ps.OverlayValues) > 605 && ps.OverlayValues[605].Loc != scm.LocNone {
			d605 = ps.OverlayValues[605]
		}
		if len(ps.OverlayValues) > 606 && ps.OverlayValues[606].Loc != scm.LocNone {
			d606 = ps.OverlayValues[606]
		}
		if len(ps.OverlayValues) > 607 && ps.OverlayValues[607].Loc != scm.LocNone {
			d607 = ps.OverlayValues[607]
		}
		if len(ps.OverlayValues) > 608 && ps.OverlayValues[608].Loc != scm.LocNone {
			d608 = ps.OverlayValues[608]
		}
		if len(ps.OverlayValues) > 609 && ps.OverlayValues[609].Loc != scm.LocNone {
			d609 = ps.OverlayValues[609]
		}
		if len(ps.OverlayValues) > 610 && ps.OverlayValues[610].Loc != scm.LocNone {
			d610 = ps.OverlayValues[610]
		}
		if len(ps.OverlayValues) > 611 && ps.OverlayValues[611].Loc != scm.LocNone {
			d611 = ps.OverlayValues[611]
		}
		if len(ps.OverlayValues) > 612 && ps.OverlayValues[612].Loc != scm.LocNone {
			d612 = ps.OverlayValues[612]
		}
		if len(ps.OverlayValues) > 613 && ps.OverlayValues[613].Loc != scm.LocNone {
			d613 = ps.OverlayValues[613]
		}
		if len(ps.OverlayValues) > 614 && ps.OverlayValues[614].Loc != scm.LocNone {
			d614 = ps.OverlayValues[614]
		}
		if len(ps.OverlayValues) > 615 && ps.OverlayValues[615].Loc != scm.LocNone {
			d615 = ps.OverlayValues[615]
		}
		if len(ps.OverlayValues) > 616 && ps.OverlayValues[616].Loc != scm.LocNone {
			d616 = ps.OverlayValues[616]
		}
		if len(ps.OverlayValues) > 617 && ps.OverlayValues[617].Loc != scm.LocNone {
			d617 = ps.OverlayValues[617]
		}
		if len(ps.OverlayValues) > 618 && ps.OverlayValues[618].Loc != scm.LocNone {
			d618 = ps.OverlayValues[618]
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
		if len(ps.OverlayValues) > 624 && ps.OverlayValues[624].Loc != scm.LocNone {
			d624 = ps.OverlayValues[624]
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d5)
		var d913 scm.JITValueDesc
		if d5.Loc == scm.LocImm {
			d913 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d913 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d913)
		}
		if d913.Loc == scm.LocImm {
			d913 = scm.JITValueDesc{Loc: scm.LocImm, Type: d913.Type, Imm: scm.NewInt(int64(uint64(d913.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d913.Reg, 32)
			ctx.EmitShrRegImm8(d913.Reg, 32)
		}
		if d913.Loc == scm.LocReg && d5.Loc == scm.LocReg && d913.Reg == d5.Reg {
			ctx.TransferReg(d5.Reg)
			d5.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d913)
		ctx.EmitStoreToStack(d913, int32(bbs[8].PhiBase)+int32(16))
		ctx.StabilizeDescForControlFlow(&d913)
		if ps.General {
			ctx.SyncDesc(&d6)
			if d6.Loc == scm.LocReg {
				ctx.ProtectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.ProtectReg(d6.Reg)
				ctx.ProtectReg(d6.Reg2)
			}
			d914 = d6
			if d914.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d914)
			d915 = d914
			if d915.Loc == scm.LocImm {
				d915 = scm.JITValueDesc{Loc: scm.LocImm, Type: d915.Type, Imm: scm.NewInt(int64(uint64(d915.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d915.Reg, 32)
				ctx.EmitShrRegImm8(d915.Reg, 32)
			}
			ctx.EmitStoreToStack(d915, int32(bbs[8].PhiBase)+int32(0))
			if d6.Loc == scm.LocReg {
				ctx.UnprotectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d6.Reg)
				ctx.UnprotectReg(d6.Reg2)
			}
		}
		ps916 := scm.PhiState{General: ps.General}
		ps916.OverlayValues = make([]scm.JITValueDesc, 916)
		ps916.OverlayValues[1] = d1
		ps916.OverlayValues[2] = d2
		ps916.OverlayValues[3] = d3
		ps916.OverlayValues[4] = d4
		ps916.OverlayValues[5] = d5
		ps916.OverlayValues[6] = d6
		ps916.OverlayValues[7] = d7
		ps916.OverlayValues[8] = d8
		ps916.OverlayValues[9] = d9
		ps916.OverlayValues[10] = d10
		ps916.OverlayValues[11] = d11
		ps916.OverlayValues[12] = d12
		ps916.OverlayValues[13] = d13
		ps916.OverlayValues[14] = d14
		ps916.OverlayValues[15] = d15
		ps916.OverlayValues[17] = d17
		ps916.OverlayValues[18] = d18
		ps916.OverlayValues[19] = d19
		ps916.OverlayValues[20] = d20
		ps916.OverlayValues[21] = d21
		ps916.OverlayValues[22] = d22
		ps916.OverlayValues[23] = d23
		ps916.OverlayValues[24] = d24
		ps916.OverlayValues[25] = d25
		ps916.OverlayValues[26] = d26
		ps916.OverlayValues[27] = d27
		ps916.OverlayValues[28] = d28
		ps916.OverlayValues[29] = d29
		ps916.OverlayValues[30] = d30
		ps916.OverlayValues[31] = d31
		ps916.OverlayValues[32] = d32
		ps916.OverlayValues[33] = d33
		ps916.OverlayValues[34] = d34
		ps916.OverlayValues[35] = d35
		ps916.OverlayValues[36] = d36
		ps916.OverlayValues[37] = d37
		ps916.OverlayValues[38] = d38
		ps916.OverlayValues[39] = d39
		ps916.OverlayValues[40] = d40
		ps916.OverlayValues[41] = d41
		ps916.OverlayValues[42] = d42
		ps916.OverlayValues[43] = d43
		ps916.OverlayValues[44] = d44
		ps916.OverlayValues[45] = d45
		ps916.OverlayValues[46] = d46
		ps916.OverlayValues[47] = d47
		ps916.OverlayValues[48] = d48
		ps916.OverlayValues[49] = d49
		ps916.OverlayValues[50] = d50
		ps916.OverlayValues[53] = d53
		ps916.OverlayValues[54] = d54
		ps916.OverlayValues[55] = d55
		ps916.OverlayValues[164] = d164
		ps916.OverlayValues[165] = d165
		ps916.OverlayValues[166] = d166
		ps916.OverlayValues[167] = d167
		ps916.OverlayValues[168] = d168
		ps916.OverlayValues[169] = d169
		ps916.OverlayValues[170] = d170
		ps916.OverlayValues[171] = d171
		ps916.OverlayValues[172] = d172
		ps916.OverlayValues[173] = d173
		ps916.OverlayValues[174] = d174
		ps916.OverlayValues[175] = d175
		ps916.OverlayValues[176] = d176
		ps916.OverlayValues[177] = d177
		ps916.OverlayValues[178] = d178
		ps916.OverlayValues[179] = d179
		ps916.OverlayValues[180] = d180
		ps916.OverlayValues[181] = d181
		ps916.OverlayValues[182] = d182
		ps916.OverlayValues[183] = d183
		ps916.OverlayValues[184] = d184
		ps916.OverlayValues[185] = d185
		ps916.OverlayValues[186] = d186
		ps916.OverlayValues[187] = d187
		ps916.OverlayValues[188] = d188
		ps916.OverlayValues[189] = d189
		ps916.OverlayValues[190] = d190
		ps916.OverlayValues[191] = d191
		ps916.OverlayValues[192] = d192
		ps916.OverlayValues[193] = d193
		ps916.OverlayValues[196] = d196
		ps916.OverlayValues[367] = d367
		ps916.OverlayValues[368] = d368
		ps916.OverlayValues[369] = d369
		ps916.OverlayValues[370] = d370
		ps916.OverlayValues[372] = d372
		ps916.OverlayValues[373] = d373
		ps916.OverlayValues[374] = d374
		ps916.OverlayValues[375] = d375
		ps916.OverlayValues[376] = d376
		ps916.OverlayValues[377] = d377
		ps916.OverlayValues[378] = d378
		ps916.OverlayValues[379] = d379
		ps916.OverlayValues[381] = d381
		ps916.OverlayValues[383] = d383
		ps916.OverlayValues[384] = d384
		ps916.OverlayValues[385] = d385
		ps916.OverlayValues[486] = d486
		ps916.OverlayValues[487] = d487
		ps916.OverlayValues[490] = d490
		ps916.OverlayValues[594] = d594
		ps916.OverlayValues[595] = d595
		ps916.OverlayValues[596] = d596
		ps916.OverlayValues[597] = d597
		ps916.OverlayValues[598] = d598
		ps916.OverlayValues[600] = d600
		ps916.OverlayValues[601] = d601
		ps916.OverlayValues[602] = d602
		ps916.OverlayValues[603] = d603
		ps916.OverlayValues[604] = d604
		ps916.OverlayValues[605] = d605
		ps916.OverlayValues[606] = d606
		ps916.OverlayValues[607] = d607
		ps916.OverlayValues[608] = d608
		ps916.OverlayValues[609] = d609
		ps916.OverlayValues[610] = d610
		ps916.OverlayValues[611] = d611
		ps916.OverlayValues[612] = d612
		ps916.OverlayValues[613] = d613
		ps916.OverlayValues[614] = d614
		ps916.OverlayValues[615] = d615
		ps916.OverlayValues[616] = d616
		ps916.OverlayValues[617] = d617
		ps916.OverlayValues[618] = d618
		ps916.OverlayValues[619] = d619
		ps916.OverlayValues[620] = d620
		ps916.OverlayValues[621] = d621
		ps916.OverlayValues[622] = d622
		ps916.OverlayValues[623] = d623
		ps916.OverlayValues[624] = d624
		ps916.OverlayValues[625] = d625
		ps916.OverlayValues[626] = d626
		ps916.OverlayValues[627] = d627
		ps916.OverlayValues[628] = d628
		ps916.OverlayValues[629] = d629
		ps916.OverlayValues[630] = d630
		ps916.OverlayValues[913] = d913
		ps916.OverlayValues[914] = d914
		ps916.OverlayValues[915] = d915
		ps916.PhiValues = make([]scm.JITValueDesc, 2)
		d917 = d6
		ps916.PhiValues[0] = d917
		if ps916.General && bbs[8].Rendered {
			ctx.EmitJmp(lbl9)
			return result
		}
		return bbs[8].RenderPS(ps916)
		return result
	}
	bbs[8].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d918 := ps.PhiValues[0]
				ctx.EnsureDesc(&d918)
				ctx.EmitStoreToStack(d918, int32(bbs[8].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d919 := ps.PhiValues[1]
				ctx.EnsureDesc(&d919)
				ctx.EmitStoreToStack(d919, int32(bbs[8].PhiBase)+int32(16))
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
		if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != scm.LocNone {
			d176 = ps.OverlayValues[176]
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
		if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
			d196 = ps.OverlayValues[196]
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
		if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != scm.LocNone {
			d381 = ps.OverlayValues[381]
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
		if len(ps.OverlayValues) > 486 && ps.OverlayValues[486].Loc != scm.LocNone {
			d486 = ps.OverlayValues[486]
		}
		if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != scm.LocNone {
			d487 = ps.OverlayValues[487]
		}
		if len(ps.OverlayValues) > 490 && ps.OverlayValues[490].Loc != scm.LocNone {
			d490 = ps.OverlayValues[490]
		}
		if len(ps.OverlayValues) > 594 && ps.OverlayValues[594].Loc != scm.LocNone {
			d594 = ps.OverlayValues[594]
		}
		if len(ps.OverlayValues) > 595 && ps.OverlayValues[595].Loc != scm.LocNone {
			d595 = ps.OverlayValues[595]
		}
		if len(ps.OverlayValues) > 596 && ps.OverlayValues[596].Loc != scm.LocNone {
			d596 = ps.OverlayValues[596]
		}
		if len(ps.OverlayValues) > 597 && ps.OverlayValues[597].Loc != scm.LocNone {
			d597 = ps.OverlayValues[597]
		}
		if len(ps.OverlayValues) > 598 && ps.OverlayValues[598].Loc != scm.LocNone {
			d598 = ps.OverlayValues[598]
		}
		if len(ps.OverlayValues) > 600 && ps.OverlayValues[600].Loc != scm.LocNone {
			d600 = ps.OverlayValues[600]
		}
		if len(ps.OverlayValues) > 601 && ps.OverlayValues[601].Loc != scm.LocNone {
			d601 = ps.OverlayValues[601]
		}
		if len(ps.OverlayValues) > 602 && ps.OverlayValues[602].Loc != scm.LocNone {
			d602 = ps.OverlayValues[602]
		}
		if len(ps.OverlayValues) > 603 && ps.OverlayValues[603].Loc != scm.LocNone {
			d603 = ps.OverlayValues[603]
		}
		if len(ps.OverlayValues) > 604 && ps.OverlayValues[604].Loc != scm.LocNone {
			d604 = ps.OverlayValues[604]
		}
		if len(ps.OverlayValues) > 605 && ps.OverlayValues[605].Loc != scm.LocNone {
			d605 = ps.OverlayValues[605]
		}
		if len(ps.OverlayValues) > 606 && ps.OverlayValues[606].Loc != scm.LocNone {
			d606 = ps.OverlayValues[606]
		}
		if len(ps.OverlayValues) > 607 && ps.OverlayValues[607].Loc != scm.LocNone {
			d607 = ps.OverlayValues[607]
		}
		if len(ps.OverlayValues) > 608 && ps.OverlayValues[608].Loc != scm.LocNone {
			d608 = ps.OverlayValues[608]
		}
		if len(ps.OverlayValues) > 609 && ps.OverlayValues[609].Loc != scm.LocNone {
			d609 = ps.OverlayValues[609]
		}
		if len(ps.OverlayValues) > 610 && ps.OverlayValues[610].Loc != scm.LocNone {
			d610 = ps.OverlayValues[610]
		}
		if len(ps.OverlayValues) > 611 && ps.OverlayValues[611].Loc != scm.LocNone {
			d611 = ps.OverlayValues[611]
		}
		if len(ps.OverlayValues) > 612 && ps.OverlayValues[612].Loc != scm.LocNone {
			d612 = ps.OverlayValues[612]
		}
		if len(ps.OverlayValues) > 613 && ps.OverlayValues[613].Loc != scm.LocNone {
			d613 = ps.OverlayValues[613]
		}
		if len(ps.OverlayValues) > 614 && ps.OverlayValues[614].Loc != scm.LocNone {
			d614 = ps.OverlayValues[614]
		}
		if len(ps.OverlayValues) > 615 && ps.OverlayValues[615].Loc != scm.LocNone {
			d615 = ps.OverlayValues[615]
		}
		if len(ps.OverlayValues) > 616 && ps.OverlayValues[616].Loc != scm.LocNone {
			d616 = ps.OverlayValues[616]
		}
		if len(ps.OverlayValues) > 617 && ps.OverlayValues[617].Loc != scm.LocNone {
			d617 = ps.OverlayValues[617]
		}
		if len(ps.OverlayValues) > 618 && ps.OverlayValues[618].Loc != scm.LocNone {
			d618 = ps.OverlayValues[618]
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
		if len(ps.OverlayValues) > 624 && ps.OverlayValues[624].Loc != scm.LocNone {
			d624 = ps.OverlayValues[624]
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
		if len(ps.OverlayValues) > 913 && ps.OverlayValues[913].Loc != scm.LocNone {
			d913 = ps.OverlayValues[913]
		}
		if len(ps.OverlayValues) > 914 && ps.OverlayValues[914].Loc != scm.LocNone {
			d914 = ps.OverlayValues[914]
		}
		if len(ps.OverlayValues) > 915 && ps.OverlayValues[915].Loc != scm.LocNone {
			d915 = ps.OverlayValues[915]
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
		var d920 scm.JITValueDesc
		if d8.Loc == scm.LocImm && d9.Loc == scm.LocImm {
			d920 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d8.Imm.Int()) == uint64(d9.Imm.Int()))}
		} else if d9.Loc == scm.LocImm {
			r102 := ctx.AllocRegExcept(d8.Reg)
			if d9.Imm.Int() >= -2147483648 && d9.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d8.Reg, int32(d9.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d9.Imm.Int()))
				ctx.EmitCmpInt64(d8.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r102, scm.CondEqual)
			d920 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r102}
			ctx.BindReg(r102, &d920)
		} else if d8.Loc == scm.LocImm {
			r103 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d8.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d9.Reg)
			ctx.EmitSetcc(r103, scm.CondEqual)
			d920 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r103}
			ctx.BindReg(r103, &d920)
		} else {
			r104 := ctx.AllocRegExcept(d8.Reg)
			ctx.EmitCmpInt64(d8.Reg, d9.Reg)
			ctx.EmitSetcc(r104, scm.CondEqual)
			d920 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r104}
			ctx.BindReg(r104, &d920)
		}
		d921 = d920
		ctx.EnsureDesc(&d921)
		if d921.Loc != scm.LocImm && d921.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d921.Loc == scm.LocImm {
			if d921.Imm.Bool() {
				if ps.General {
					ctx.SyncDesc(&d8)
					if d8.Loc == scm.LocReg {
						ctx.ProtectReg(d8.Reg)
					} else if d8.Loc == scm.LocRegPair {
						ctx.ProtectReg(d8.Reg)
						ctx.ProtectReg(d8.Reg2)
					}
					d922 = d8
					if d922.Loc == scm.LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d922)
					d923 = d922
					if d923.Loc == scm.LocImm {
						d923 = scm.JITValueDesc{Loc: scm.LocImm, Type: d923.Type, Imm: scm.NewInt(int64(uint64(d923.Imm.Int()) & 0xffffffff))}
					} else {
						ctx.EmitShlRegImm8(d923.Reg, 32)
						ctx.EmitShrRegImm8(d923.Reg, 32)
					}
					ctx.EmitStoreToStack(d923, int32(bbs[2].PhiBase)+int32(0))
					if d8.Loc == scm.LocReg {
						ctx.UnprotectReg(d8.Reg)
					} else if d8.Loc == scm.LocRegPair {
						ctx.UnprotectReg(d8.Reg)
						ctx.UnprotectReg(d8.Reg2)
					}
				}
				ps924 := scm.PhiState{General: ps.General}
				ps924.OverlayValues = make([]scm.JITValueDesc, 924)
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
				ps924.OverlayValues[23] = d23
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
				ps924.OverlayValues[53] = d53
				ps924.OverlayValues[54] = d54
				ps924.OverlayValues[55] = d55
				ps924.OverlayValues[164] = d164
				ps924.OverlayValues[165] = d165
				ps924.OverlayValues[166] = d166
				ps924.OverlayValues[167] = d167
				ps924.OverlayValues[168] = d168
				ps924.OverlayValues[169] = d169
				ps924.OverlayValues[170] = d170
				ps924.OverlayValues[171] = d171
				ps924.OverlayValues[172] = d172
				ps924.OverlayValues[173] = d173
				ps924.OverlayValues[174] = d174
				ps924.OverlayValues[175] = d175
				ps924.OverlayValues[176] = d176
				ps924.OverlayValues[177] = d177
				ps924.OverlayValues[178] = d178
				ps924.OverlayValues[179] = d179
				ps924.OverlayValues[180] = d180
				ps924.OverlayValues[181] = d181
				ps924.OverlayValues[182] = d182
				ps924.OverlayValues[183] = d183
				ps924.OverlayValues[184] = d184
				ps924.OverlayValues[185] = d185
				ps924.OverlayValues[186] = d186
				ps924.OverlayValues[187] = d187
				ps924.OverlayValues[188] = d188
				ps924.OverlayValues[189] = d189
				ps924.OverlayValues[190] = d190
				ps924.OverlayValues[191] = d191
				ps924.OverlayValues[192] = d192
				ps924.OverlayValues[193] = d193
				ps924.OverlayValues[196] = d196
				ps924.OverlayValues[367] = d367
				ps924.OverlayValues[368] = d368
				ps924.OverlayValues[369] = d369
				ps924.OverlayValues[370] = d370
				ps924.OverlayValues[372] = d372
				ps924.OverlayValues[373] = d373
				ps924.OverlayValues[374] = d374
				ps924.OverlayValues[375] = d375
				ps924.OverlayValues[376] = d376
				ps924.OverlayValues[377] = d377
				ps924.OverlayValues[378] = d378
				ps924.OverlayValues[379] = d379
				ps924.OverlayValues[381] = d381
				ps924.OverlayValues[383] = d383
				ps924.OverlayValues[384] = d384
				ps924.OverlayValues[385] = d385
				ps924.OverlayValues[486] = d486
				ps924.OverlayValues[487] = d487
				ps924.OverlayValues[490] = d490
				ps924.OverlayValues[594] = d594
				ps924.OverlayValues[595] = d595
				ps924.OverlayValues[596] = d596
				ps924.OverlayValues[597] = d597
				ps924.OverlayValues[598] = d598
				ps924.OverlayValues[600] = d600
				ps924.OverlayValues[601] = d601
				ps924.OverlayValues[602] = d602
				ps924.OverlayValues[603] = d603
				ps924.OverlayValues[604] = d604
				ps924.OverlayValues[605] = d605
				ps924.OverlayValues[606] = d606
				ps924.OverlayValues[607] = d607
				ps924.OverlayValues[608] = d608
				ps924.OverlayValues[609] = d609
				ps924.OverlayValues[610] = d610
				ps924.OverlayValues[611] = d611
				ps924.OverlayValues[612] = d612
				ps924.OverlayValues[613] = d613
				ps924.OverlayValues[614] = d614
				ps924.OverlayValues[615] = d615
				ps924.OverlayValues[616] = d616
				ps924.OverlayValues[617] = d617
				ps924.OverlayValues[618] = d618
				ps924.OverlayValues[619] = d619
				ps924.OverlayValues[620] = d620
				ps924.OverlayValues[621] = d621
				ps924.OverlayValues[622] = d622
				ps924.OverlayValues[623] = d623
				ps924.OverlayValues[624] = d624
				ps924.OverlayValues[625] = d625
				ps924.OverlayValues[626] = d626
				ps924.OverlayValues[627] = d627
				ps924.OverlayValues[628] = d628
				ps924.OverlayValues[629] = d629
				ps924.OverlayValues[630] = d630
				ps924.OverlayValues[913] = d913
				ps924.OverlayValues[914] = d914
				ps924.OverlayValues[915] = d915
				ps924.OverlayValues[917] = d917
				ps924.OverlayValues[918] = d918
				ps924.OverlayValues[919] = d919
				ps924.OverlayValues[920] = d920
				ps924.OverlayValues[921] = d921
				ps924.OverlayValues[922] = d922
				ps924.OverlayValues[923] = d923
				ps924.PhiValues = make([]scm.JITValueDesc, 1)
				d925 = d8
				ps924.PhiValues[0] = d925
				return bbs[2].RenderPS(ps924)
			}
			if ps.General {
			}
			ps926 := scm.PhiState{General: ps.General}
			ps926.OverlayValues = make([]scm.JITValueDesc, 926)
			ps926.OverlayValues[1] = d1
			ps926.OverlayValues[2] = d2
			ps926.OverlayValues[3] = d3
			ps926.OverlayValues[4] = d4
			ps926.OverlayValues[5] = d5
			ps926.OverlayValues[6] = d6
			ps926.OverlayValues[7] = d7
			ps926.OverlayValues[8] = d8
			ps926.OverlayValues[9] = d9
			ps926.OverlayValues[10] = d10
			ps926.OverlayValues[11] = d11
			ps926.OverlayValues[12] = d12
			ps926.OverlayValues[13] = d13
			ps926.OverlayValues[14] = d14
			ps926.OverlayValues[15] = d15
			ps926.OverlayValues[17] = d17
			ps926.OverlayValues[18] = d18
			ps926.OverlayValues[19] = d19
			ps926.OverlayValues[20] = d20
			ps926.OverlayValues[21] = d21
			ps926.OverlayValues[22] = d22
			ps926.OverlayValues[23] = d23
			ps926.OverlayValues[24] = d24
			ps926.OverlayValues[25] = d25
			ps926.OverlayValues[26] = d26
			ps926.OverlayValues[27] = d27
			ps926.OverlayValues[28] = d28
			ps926.OverlayValues[29] = d29
			ps926.OverlayValues[30] = d30
			ps926.OverlayValues[31] = d31
			ps926.OverlayValues[32] = d32
			ps926.OverlayValues[33] = d33
			ps926.OverlayValues[34] = d34
			ps926.OverlayValues[35] = d35
			ps926.OverlayValues[36] = d36
			ps926.OverlayValues[37] = d37
			ps926.OverlayValues[38] = d38
			ps926.OverlayValues[39] = d39
			ps926.OverlayValues[40] = d40
			ps926.OverlayValues[41] = d41
			ps926.OverlayValues[42] = d42
			ps926.OverlayValues[43] = d43
			ps926.OverlayValues[44] = d44
			ps926.OverlayValues[45] = d45
			ps926.OverlayValues[46] = d46
			ps926.OverlayValues[47] = d47
			ps926.OverlayValues[48] = d48
			ps926.OverlayValues[49] = d49
			ps926.OverlayValues[50] = d50
			ps926.OverlayValues[53] = d53
			ps926.OverlayValues[54] = d54
			ps926.OverlayValues[55] = d55
			ps926.OverlayValues[164] = d164
			ps926.OverlayValues[165] = d165
			ps926.OverlayValues[166] = d166
			ps926.OverlayValues[167] = d167
			ps926.OverlayValues[168] = d168
			ps926.OverlayValues[169] = d169
			ps926.OverlayValues[170] = d170
			ps926.OverlayValues[171] = d171
			ps926.OverlayValues[172] = d172
			ps926.OverlayValues[173] = d173
			ps926.OverlayValues[174] = d174
			ps926.OverlayValues[175] = d175
			ps926.OverlayValues[176] = d176
			ps926.OverlayValues[177] = d177
			ps926.OverlayValues[178] = d178
			ps926.OverlayValues[179] = d179
			ps926.OverlayValues[180] = d180
			ps926.OverlayValues[181] = d181
			ps926.OverlayValues[182] = d182
			ps926.OverlayValues[183] = d183
			ps926.OverlayValues[184] = d184
			ps926.OverlayValues[185] = d185
			ps926.OverlayValues[186] = d186
			ps926.OverlayValues[187] = d187
			ps926.OverlayValues[188] = d188
			ps926.OverlayValues[189] = d189
			ps926.OverlayValues[190] = d190
			ps926.OverlayValues[191] = d191
			ps926.OverlayValues[192] = d192
			ps926.OverlayValues[193] = d193
			ps926.OverlayValues[196] = d196
			ps926.OverlayValues[367] = d367
			ps926.OverlayValues[368] = d368
			ps926.OverlayValues[369] = d369
			ps926.OverlayValues[370] = d370
			ps926.OverlayValues[372] = d372
			ps926.OverlayValues[373] = d373
			ps926.OverlayValues[374] = d374
			ps926.OverlayValues[375] = d375
			ps926.OverlayValues[376] = d376
			ps926.OverlayValues[377] = d377
			ps926.OverlayValues[378] = d378
			ps926.OverlayValues[379] = d379
			ps926.OverlayValues[381] = d381
			ps926.OverlayValues[383] = d383
			ps926.OverlayValues[384] = d384
			ps926.OverlayValues[385] = d385
			ps926.OverlayValues[486] = d486
			ps926.OverlayValues[487] = d487
			ps926.OverlayValues[490] = d490
			ps926.OverlayValues[594] = d594
			ps926.OverlayValues[595] = d595
			ps926.OverlayValues[596] = d596
			ps926.OverlayValues[597] = d597
			ps926.OverlayValues[598] = d598
			ps926.OverlayValues[600] = d600
			ps926.OverlayValues[601] = d601
			ps926.OverlayValues[602] = d602
			ps926.OverlayValues[603] = d603
			ps926.OverlayValues[604] = d604
			ps926.OverlayValues[605] = d605
			ps926.OverlayValues[606] = d606
			ps926.OverlayValues[607] = d607
			ps926.OverlayValues[608] = d608
			ps926.OverlayValues[609] = d609
			ps926.OverlayValues[610] = d610
			ps926.OverlayValues[611] = d611
			ps926.OverlayValues[612] = d612
			ps926.OverlayValues[613] = d613
			ps926.OverlayValues[614] = d614
			ps926.OverlayValues[615] = d615
			ps926.OverlayValues[616] = d616
			ps926.OverlayValues[617] = d617
			ps926.OverlayValues[618] = d618
			ps926.OverlayValues[619] = d619
			ps926.OverlayValues[620] = d620
			ps926.OverlayValues[621] = d621
			ps926.OverlayValues[622] = d622
			ps926.OverlayValues[623] = d623
			ps926.OverlayValues[624] = d624
			ps926.OverlayValues[625] = d625
			ps926.OverlayValues[626] = d626
			ps926.OverlayValues[627] = d627
			ps926.OverlayValues[628] = d628
			ps926.OverlayValues[629] = d629
			ps926.OverlayValues[630] = d630
			ps926.OverlayValues[913] = d913
			ps926.OverlayValues[914] = d914
			ps926.OverlayValues[915] = d915
			ps926.OverlayValues[917] = d917
			ps926.OverlayValues[918] = d918
			ps926.OverlayValues[919] = d919
			ps926.OverlayValues[920] = d920
			ps926.OverlayValues[921] = d921
			ps926.OverlayValues[922] = d922
			ps926.OverlayValues[923] = d923
			ps926.OverlayValues[925] = d925
			return bbs[10].RenderPS(ps926)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d927 := ps.PhiValues[0]
				ctx.EnsureDesc(&d927)
				ctx.EmitStoreToStack(d927, int32(bbs[8].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d928 := ps.PhiValues[1]
				ctx.EnsureDesc(&d928)
				ctx.EmitStoreToStack(d928, int32(bbs[8].PhiBase)+int32(16))
			}
			ps.General = true
			return bbs[8].RenderPS(ps)
		}
		lbl26 := ctx.ReserveLabel()
		lbl27 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d921.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl26)
		ctx.EmitJmp(lbl27)
		snap929 := d1
		snap930 := d2
		snap931 := d3
		snap932 := d4
		snap933 := d5
		snap934 := d6
		snap935 := d7
		snap936 := d8
		snap937 := d9
		snap938 := d10
		snap939 := d11
		snap940 := d12
		snap941 := d13
		snap942 := d14
		snap943 := d15
		snap944 := d17
		snap945 := d18
		snap946 := d19
		snap947 := d20
		snap948 := d21
		snap949 := d22
		snap950 := d23
		snap951 := d24
		snap952 := d25
		snap953 := d26
		snap954 := d27
		snap955 := d28
		snap956 := d29
		snap957 := d30
		snap958 := d31
		snap959 := d32
		snap960 := d33
		snap961 := d34
		snap962 := d35
		snap963 := d36
		snap964 := d37
		snap965 := d38
		snap966 := d39
		snap967 := d40
		snap968 := d41
		snap969 := d42
		snap970 := d43
		snap971 := d44
		snap972 := d45
		snap973 := d46
		snap974 := d47
		snap975 := d48
		snap976 := d49
		snap977 := d50
		snap978 := d53
		snap979 := d54
		snap980 := d55
		snap981 := d164
		snap982 := d165
		snap983 := d166
		snap984 := d167
		snap985 := d168
		snap986 := d169
		snap987 := d170
		snap988 := d171
		snap989 := d172
		snap990 := d173
		snap991 := d174
		snap992 := d175
		snap993 := d176
		snap994 := d177
		snap995 := d178
		snap996 := d179
		snap997 := d180
		snap998 := d181
		snap999 := d182
		snap1000 := d183
		snap1001 := d184
		snap1002 := d185
		snap1003 := d186
		snap1004 := d187
		snap1005 := d188
		snap1006 := d189
		snap1007 := d190
		snap1008 := d191
		snap1009 := d192
		snap1010 := d193
		snap1011 := d196
		snap1012 := d367
		snap1013 := d368
		snap1014 := d369
		snap1015 := d370
		snap1016 := d372
		snap1017 := d373
		snap1018 := d374
		snap1019 := d375
		snap1020 := d376
		snap1021 := d377
		snap1022 := d378
		snap1023 := d379
		snap1024 := d381
		snap1025 := d383
		snap1026 := d384
		snap1027 := d385
		snap1028 := d486
		snap1029 := d487
		snap1030 := d490
		snap1031 := d594
		snap1032 := d595
		snap1033 := d596
		snap1034 := d597
		snap1035 := d598
		snap1036 := d600
		snap1037 := d601
		snap1038 := d602
		snap1039 := d603
		snap1040 := d604
		snap1041 := d605
		snap1042 := d606
		snap1043 := d607
		snap1044 := d608
		snap1045 := d609
		snap1046 := d610
		snap1047 := d611
		snap1048 := d612
		snap1049 := d613
		snap1050 := d614
		snap1051 := d615
		snap1052 := d616
		snap1053 := d617
		snap1054 := d618
		snap1055 := d619
		snap1056 := d620
		snap1057 := d621
		snap1058 := d622
		snap1059 := d623
		snap1060 := d624
		snap1061 := d625
		snap1062 := d626
		snap1063 := d627
		snap1064 := d628
		snap1065 := d629
		snap1066 := d630
		snap1067 := d913
		snap1068 := d914
		snap1069 := d915
		snap1070 := d917
		snap1071 := d918
		snap1072 := d919
		snap1073 := d920
		snap1074 := d921
		snap1075 := d922
		snap1076 := d923
		snap1077 := d925
		snap1078 := d927
		snap1079 := d928
		alloc1080 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl26)
		ctx.SyncDesc(&d8)
		if d8.Loc == scm.LocReg {
			ctx.ProtectReg(d8.Reg)
		} else if d8.Loc == scm.LocRegPair {
			ctx.ProtectReg(d8.Reg)
			ctx.ProtectReg(d8.Reg2)
		}
		d1081 = d8
		if d1081.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d1081)
		d1082 = d1081
		if d1082.Loc == scm.LocImm {
			d1082 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1082.Type, Imm: scm.NewInt(int64(uint64(d1082.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d1082.Reg, 32)
			ctx.EmitShrRegImm8(d1082.Reg, 32)
		}
		ctx.EmitStoreToStack(d1082, int32(bbs[2].PhiBase)+int32(0))
		if d8.Loc == scm.LocReg {
			ctx.UnprotectReg(d8.Reg)
		} else if d8.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d8.Reg)
			ctx.UnprotectReg(d8.Reg2)
		}
		ctx.EmitJmp(lbl3)
		ctx.RestoreAllocState(alloc1080)
		d1 = snap929
		d2 = snap930
		d3 = snap931
		d4 = snap932
		d5 = snap933
		d6 = snap934
		d7 = snap935
		d8 = snap936
		d9 = snap937
		d10 = snap938
		d11 = snap939
		d12 = snap940
		d13 = snap941
		d14 = snap942
		d15 = snap943
		d17 = snap944
		d18 = snap945
		d19 = snap946
		d20 = snap947
		d21 = snap948
		d22 = snap949
		d23 = snap950
		d24 = snap951
		d25 = snap952
		d26 = snap953
		d27 = snap954
		d28 = snap955
		d29 = snap956
		d30 = snap957
		d31 = snap958
		d32 = snap959
		d33 = snap960
		d34 = snap961
		d35 = snap962
		d36 = snap963
		d37 = snap964
		d38 = snap965
		d39 = snap966
		d40 = snap967
		d41 = snap968
		d42 = snap969
		d43 = snap970
		d44 = snap971
		d45 = snap972
		d46 = snap973
		d47 = snap974
		d48 = snap975
		d49 = snap976
		d50 = snap977
		d53 = snap978
		d54 = snap979
		d55 = snap980
		d164 = snap981
		d165 = snap982
		d166 = snap983
		d167 = snap984
		d168 = snap985
		d169 = snap986
		d170 = snap987
		d171 = snap988
		d172 = snap989
		d173 = snap990
		d174 = snap991
		d175 = snap992
		d176 = snap993
		d177 = snap994
		d178 = snap995
		d179 = snap996
		d180 = snap997
		d181 = snap998
		d182 = snap999
		d183 = snap1000
		d184 = snap1001
		d185 = snap1002
		d186 = snap1003
		d187 = snap1004
		d188 = snap1005
		d189 = snap1006
		d190 = snap1007
		d191 = snap1008
		d192 = snap1009
		d193 = snap1010
		d196 = snap1011
		d367 = snap1012
		d368 = snap1013
		d369 = snap1014
		d370 = snap1015
		d372 = snap1016
		d373 = snap1017
		d374 = snap1018
		d375 = snap1019
		d376 = snap1020
		d377 = snap1021
		d378 = snap1022
		d379 = snap1023
		d381 = snap1024
		d383 = snap1025
		d384 = snap1026
		d385 = snap1027
		d486 = snap1028
		d487 = snap1029
		d490 = snap1030
		d594 = snap1031
		d595 = snap1032
		d596 = snap1033
		d597 = snap1034
		d598 = snap1035
		d600 = snap1036
		d601 = snap1037
		d602 = snap1038
		d603 = snap1039
		d604 = snap1040
		d605 = snap1041
		d606 = snap1042
		d607 = snap1043
		d608 = snap1044
		d609 = snap1045
		d610 = snap1046
		d611 = snap1047
		d612 = snap1048
		d613 = snap1049
		d614 = snap1050
		d615 = snap1051
		d616 = snap1052
		d617 = snap1053
		d618 = snap1054
		d619 = snap1055
		d620 = snap1056
		d621 = snap1057
		d622 = snap1058
		d623 = snap1059
		d624 = snap1060
		d625 = snap1061
		d626 = snap1062
		d627 = snap1063
		d628 = snap1064
		d629 = snap1065
		d630 = snap1066
		d913 = snap1067
		d914 = snap1068
		d915 = snap1069
		d917 = snap1070
		d918 = snap1071
		d919 = snap1072
		d920 = snap1073
		d921 = snap1074
		d922 = snap1075
		d923 = snap1076
		d925 = snap1077
		d927 = snap1078
		d928 = snap1079
		ctx.MarkLabel(lbl27)
		ctx.EmitJmp(lbl11)
		ctx.RestoreAllocState(alloc1080)
		d1 = snap929
		d2 = snap930
		d3 = snap931
		d4 = snap932
		d5 = snap933
		d6 = snap934
		d7 = snap935
		d8 = snap936
		d9 = snap937
		d10 = snap938
		d11 = snap939
		d12 = snap940
		d13 = snap941
		d14 = snap942
		d15 = snap943
		d17 = snap944
		d18 = snap945
		d19 = snap946
		d20 = snap947
		d21 = snap948
		d22 = snap949
		d23 = snap950
		d24 = snap951
		d25 = snap952
		d26 = snap953
		d27 = snap954
		d28 = snap955
		d29 = snap956
		d30 = snap957
		d31 = snap958
		d32 = snap959
		d33 = snap960
		d34 = snap961
		d35 = snap962
		d36 = snap963
		d37 = snap964
		d38 = snap965
		d39 = snap966
		d40 = snap967
		d41 = snap968
		d42 = snap969
		d43 = snap970
		d44 = snap971
		d45 = snap972
		d46 = snap973
		d47 = snap974
		d48 = snap975
		d49 = snap976
		d50 = snap977
		d53 = snap978
		d54 = snap979
		d55 = snap980
		d164 = snap981
		d165 = snap982
		d166 = snap983
		d167 = snap984
		d168 = snap985
		d169 = snap986
		d170 = snap987
		d171 = snap988
		d172 = snap989
		d173 = snap990
		d174 = snap991
		d175 = snap992
		d176 = snap993
		d177 = snap994
		d178 = snap995
		d179 = snap996
		d180 = snap997
		d181 = snap998
		d182 = snap999
		d183 = snap1000
		d184 = snap1001
		d185 = snap1002
		d186 = snap1003
		d187 = snap1004
		d188 = snap1005
		d189 = snap1006
		d190 = snap1007
		d191 = snap1008
		d192 = snap1009
		d193 = snap1010
		d196 = snap1011
		d367 = snap1012
		d368 = snap1013
		d369 = snap1014
		d370 = snap1015
		d372 = snap1016
		d373 = snap1017
		d374 = snap1018
		d375 = snap1019
		d376 = snap1020
		d377 = snap1021
		d378 = snap1022
		d379 = snap1023
		d381 = snap1024
		d383 = snap1025
		d384 = snap1026
		d385 = snap1027
		d486 = snap1028
		d487 = snap1029
		d490 = snap1030
		d594 = snap1031
		d595 = snap1032
		d596 = snap1033
		d597 = snap1034
		d598 = snap1035
		d600 = snap1036
		d601 = snap1037
		d602 = snap1038
		d603 = snap1039
		d604 = snap1040
		d605 = snap1041
		d606 = snap1042
		d607 = snap1043
		d608 = snap1044
		d609 = snap1045
		d610 = snap1046
		d611 = snap1047
		d612 = snap1048
		d613 = snap1049
		d614 = snap1050
		d615 = snap1051
		d616 = snap1052
		d617 = snap1053
		d618 = snap1054
		d619 = snap1055
		d620 = snap1056
		d621 = snap1057
		d622 = snap1058
		d623 = snap1059
		d624 = snap1060
		d625 = snap1061
		d626 = snap1062
		d627 = snap1063
		d628 = snap1064
		d629 = snap1065
		d630 = snap1066
		d913 = snap1067
		d914 = snap1068
		d915 = snap1069
		d917 = snap1070
		d918 = snap1071
		d919 = snap1072
		d920 = snap1073
		d921 = snap1074
		d922 = snap1075
		d923 = snap1076
		d925 = snap1077
		d927 = snap1078
		d928 = snap1079
		ps1083 := scm.PhiState{General: true}
		ps1083.OverlayValues = make([]scm.JITValueDesc, 1083)
		ps1083.OverlayValues[1] = d1
		ps1083.OverlayValues[2] = d2
		ps1083.OverlayValues[3] = d3
		ps1083.OverlayValues[4] = d4
		ps1083.OverlayValues[5] = d5
		ps1083.OverlayValues[6] = d6
		ps1083.OverlayValues[7] = d7
		ps1083.OverlayValues[8] = d8
		ps1083.OverlayValues[9] = d9
		ps1083.OverlayValues[10] = d10
		ps1083.OverlayValues[11] = d11
		ps1083.OverlayValues[12] = d12
		ps1083.OverlayValues[13] = d13
		ps1083.OverlayValues[14] = d14
		ps1083.OverlayValues[15] = d15
		ps1083.OverlayValues[17] = d17
		ps1083.OverlayValues[18] = d18
		ps1083.OverlayValues[19] = d19
		ps1083.OverlayValues[20] = d20
		ps1083.OverlayValues[21] = d21
		ps1083.OverlayValues[22] = d22
		ps1083.OverlayValues[23] = d23
		ps1083.OverlayValues[24] = d24
		ps1083.OverlayValues[25] = d25
		ps1083.OverlayValues[26] = d26
		ps1083.OverlayValues[27] = d27
		ps1083.OverlayValues[28] = d28
		ps1083.OverlayValues[29] = d29
		ps1083.OverlayValues[30] = d30
		ps1083.OverlayValues[31] = d31
		ps1083.OverlayValues[32] = d32
		ps1083.OverlayValues[33] = d33
		ps1083.OverlayValues[34] = d34
		ps1083.OverlayValues[35] = d35
		ps1083.OverlayValues[36] = d36
		ps1083.OverlayValues[37] = d37
		ps1083.OverlayValues[38] = d38
		ps1083.OverlayValues[39] = d39
		ps1083.OverlayValues[40] = d40
		ps1083.OverlayValues[41] = d41
		ps1083.OverlayValues[42] = d42
		ps1083.OverlayValues[43] = d43
		ps1083.OverlayValues[44] = d44
		ps1083.OverlayValues[45] = d45
		ps1083.OverlayValues[46] = d46
		ps1083.OverlayValues[47] = d47
		ps1083.OverlayValues[48] = d48
		ps1083.OverlayValues[49] = d49
		ps1083.OverlayValues[50] = d50
		ps1083.OverlayValues[53] = d53
		ps1083.OverlayValues[54] = d54
		ps1083.OverlayValues[55] = d55
		ps1083.OverlayValues[164] = d164
		ps1083.OverlayValues[165] = d165
		ps1083.OverlayValues[166] = d166
		ps1083.OverlayValues[167] = d167
		ps1083.OverlayValues[168] = d168
		ps1083.OverlayValues[169] = d169
		ps1083.OverlayValues[170] = d170
		ps1083.OverlayValues[171] = d171
		ps1083.OverlayValues[172] = d172
		ps1083.OverlayValues[173] = d173
		ps1083.OverlayValues[174] = d174
		ps1083.OverlayValues[175] = d175
		ps1083.OverlayValues[176] = d176
		ps1083.OverlayValues[177] = d177
		ps1083.OverlayValues[178] = d178
		ps1083.OverlayValues[179] = d179
		ps1083.OverlayValues[180] = d180
		ps1083.OverlayValues[181] = d181
		ps1083.OverlayValues[182] = d182
		ps1083.OverlayValues[183] = d183
		ps1083.OverlayValues[184] = d184
		ps1083.OverlayValues[185] = d185
		ps1083.OverlayValues[186] = d186
		ps1083.OverlayValues[187] = d187
		ps1083.OverlayValues[188] = d188
		ps1083.OverlayValues[189] = d189
		ps1083.OverlayValues[190] = d190
		ps1083.OverlayValues[191] = d191
		ps1083.OverlayValues[192] = d192
		ps1083.OverlayValues[193] = d193
		ps1083.OverlayValues[196] = d196
		ps1083.OverlayValues[367] = d367
		ps1083.OverlayValues[368] = d368
		ps1083.OverlayValues[369] = d369
		ps1083.OverlayValues[370] = d370
		ps1083.OverlayValues[372] = d372
		ps1083.OverlayValues[373] = d373
		ps1083.OverlayValues[374] = d374
		ps1083.OverlayValues[375] = d375
		ps1083.OverlayValues[376] = d376
		ps1083.OverlayValues[377] = d377
		ps1083.OverlayValues[378] = d378
		ps1083.OverlayValues[379] = d379
		ps1083.OverlayValues[381] = d381
		ps1083.OverlayValues[383] = d383
		ps1083.OverlayValues[384] = d384
		ps1083.OverlayValues[385] = d385
		ps1083.OverlayValues[486] = d486
		ps1083.OverlayValues[487] = d487
		ps1083.OverlayValues[490] = d490
		ps1083.OverlayValues[594] = d594
		ps1083.OverlayValues[595] = d595
		ps1083.OverlayValues[596] = d596
		ps1083.OverlayValues[597] = d597
		ps1083.OverlayValues[598] = d598
		ps1083.OverlayValues[600] = d600
		ps1083.OverlayValues[601] = d601
		ps1083.OverlayValues[602] = d602
		ps1083.OverlayValues[603] = d603
		ps1083.OverlayValues[604] = d604
		ps1083.OverlayValues[605] = d605
		ps1083.OverlayValues[606] = d606
		ps1083.OverlayValues[607] = d607
		ps1083.OverlayValues[608] = d608
		ps1083.OverlayValues[609] = d609
		ps1083.OverlayValues[610] = d610
		ps1083.OverlayValues[611] = d611
		ps1083.OverlayValues[612] = d612
		ps1083.OverlayValues[613] = d613
		ps1083.OverlayValues[614] = d614
		ps1083.OverlayValues[615] = d615
		ps1083.OverlayValues[616] = d616
		ps1083.OverlayValues[617] = d617
		ps1083.OverlayValues[618] = d618
		ps1083.OverlayValues[619] = d619
		ps1083.OverlayValues[620] = d620
		ps1083.OverlayValues[621] = d621
		ps1083.OverlayValues[622] = d622
		ps1083.OverlayValues[623] = d623
		ps1083.OverlayValues[624] = d624
		ps1083.OverlayValues[625] = d625
		ps1083.OverlayValues[626] = d626
		ps1083.OverlayValues[627] = d627
		ps1083.OverlayValues[628] = d628
		ps1083.OverlayValues[629] = d629
		ps1083.OverlayValues[630] = d630
		ps1083.OverlayValues[913] = d913
		ps1083.OverlayValues[914] = d914
		ps1083.OverlayValues[915] = d915
		ps1083.OverlayValues[917] = d917
		ps1083.OverlayValues[918] = d918
		ps1083.OverlayValues[919] = d919
		ps1083.OverlayValues[920] = d920
		ps1083.OverlayValues[921] = d921
		ps1083.OverlayValues[922] = d922
		ps1083.OverlayValues[923] = d923
		ps1083.OverlayValues[925] = d925
		ps1083.OverlayValues[927] = d927
		ps1083.OverlayValues[928] = d928
		ps1083.OverlayValues[1081] = d1081
		ps1083.OverlayValues[1082] = d1082
		ps1083.PhiValues = make([]scm.JITValueDesc, 1)
		d1085 = d8
		ps1083.PhiValues[0] = d1085
		ps1084 := scm.PhiState{General: true}
		ps1084.OverlayValues = make([]scm.JITValueDesc, 1086)
		ps1084.OverlayValues[1] = d1
		ps1084.OverlayValues[2] = d2
		ps1084.OverlayValues[3] = d3
		ps1084.OverlayValues[4] = d4
		ps1084.OverlayValues[5] = d5
		ps1084.OverlayValues[6] = d6
		ps1084.OverlayValues[7] = d7
		ps1084.OverlayValues[8] = d8
		ps1084.OverlayValues[9] = d9
		ps1084.OverlayValues[10] = d10
		ps1084.OverlayValues[11] = d11
		ps1084.OverlayValues[12] = d12
		ps1084.OverlayValues[13] = d13
		ps1084.OverlayValues[14] = d14
		ps1084.OverlayValues[15] = d15
		ps1084.OverlayValues[17] = d17
		ps1084.OverlayValues[18] = d18
		ps1084.OverlayValues[19] = d19
		ps1084.OverlayValues[20] = d20
		ps1084.OverlayValues[21] = d21
		ps1084.OverlayValues[22] = d22
		ps1084.OverlayValues[23] = d23
		ps1084.OverlayValues[24] = d24
		ps1084.OverlayValues[25] = d25
		ps1084.OverlayValues[26] = d26
		ps1084.OverlayValues[27] = d27
		ps1084.OverlayValues[28] = d28
		ps1084.OverlayValues[29] = d29
		ps1084.OverlayValues[30] = d30
		ps1084.OverlayValues[31] = d31
		ps1084.OverlayValues[32] = d32
		ps1084.OverlayValues[33] = d33
		ps1084.OverlayValues[34] = d34
		ps1084.OverlayValues[35] = d35
		ps1084.OverlayValues[36] = d36
		ps1084.OverlayValues[37] = d37
		ps1084.OverlayValues[38] = d38
		ps1084.OverlayValues[39] = d39
		ps1084.OverlayValues[40] = d40
		ps1084.OverlayValues[41] = d41
		ps1084.OverlayValues[42] = d42
		ps1084.OverlayValues[43] = d43
		ps1084.OverlayValues[44] = d44
		ps1084.OverlayValues[45] = d45
		ps1084.OverlayValues[46] = d46
		ps1084.OverlayValues[47] = d47
		ps1084.OverlayValues[48] = d48
		ps1084.OverlayValues[49] = d49
		ps1084.OverlayValues[50] = d50
		ps1084.OverlayValues[53] = d53
		ps1084.OverlayValues[54] = d54
		ps1084.OverlayValues[55] = d55
		ps1084.OverlayValues[164] = d164
		ps1084.OverlayValues[165] = d165
		ps1084.OverlayValues[166] = d166
		ps1084.OverlayValues[167] = d167
		ps1084.OverlayValues[168] = d168
		ps1084.OverlayValues[169] = d169
		ps1084.OverlayValues[170] = d170
		ps1084.OverlayValues[171] = d171
		ps1084.OverlayValues[172] = d172
		ps1084.OverlayValues[173] = d173
		ps1084.OverlayValues[174] = d174
		ps1084.OverlayValues[175] = d175
		ps1084.OverlayValues[176] = d176
		ps1084.OverlayValues[177] = d177
		ps1084.OverlayValues[178] = d178
		ps1084.OverlayValues[179] = d179
		ps1084.OverlayValues[180] = d180
		ps1084.OverlayValues[181] = d181
		ps1084.OverlayValues[182] = d182
		ps1084.OverlayValues[183] = d183
		ps1084.OverlayValues[184] = d184
		ps1084.OverlayValues[185] = d185
		ps1084.OverlayValues[186] = d186
		ps1084.OverlayValues[187] = d187
		ps1084.OverlayValues[188] = d188
		ps1084.OverlayValues[189] = d189
		ps1084.OverlayValues[190] = d190
		ps1084.OverlayValues[191] = d191
		ps1084.OverlayValues[192] = d192
		ps1084.OverlayValues[193] = d193
		ps1084.OverlayValues[196] = d196
		ps1084.OverlayValues[367] = d367
		ps1084.OverlayValues[368] = d368
		ps1084.OverlayValues[369] = d369
		ps1084.OverlayValues[370] = d370
		ps1084.OverlayValues[372] = d372
		ps1084.OverlayValues[373] = d373
		ps1084.OverlayValues[374] = d374
		ps1084.OverlayValues[375] = d375
		ps1084.OverlayValues[376] = d376
		ps1084.OverlayValues[377] = d377
		ps1084.OverlayValues[378] = d378
		ps1084.OverlayValues[379] = d379
		ps1084.OverlayValues[381] = d381
		ps1084.OverlayValues[383] = d383
		ps1084.OverlayValues[384] = d384
		ps1084.OverlayValues[385] = d385
		ps1084.OverlayValues[486] = d486
		ps1084.OverlayValues[487] = d487
		ps1084.OverlayValues[490] = d490
		ps1084.OverlayValues[594] = d594
		ps1084.OverlayValues[595] = d595
		ps1084.OverlayValues[596] = d596
		ps1084.OverlayValues[597] = d597
		ps1084.OverlayValues[598] = d598
		ps1084.OverlayValues[600] = d600
		ps1084.OverlayValues[601] = d601
		ps1084.OverlayValues[602] = d602
		ps1084.OverlayValues[603] = d603
		ps1084.OverlayValues[604] = d604
		ps1084.OverlayValues[605] = d605
		ps1084.OverlayValues[606] = d606
		ps1084.OverlayValues[607] = d607
		ps1084.OverlayValues[608] = d608
		ps1084.OverlayValues[609] = d609
		ps1084.OverlayValues[610] = d610
		ps1084.OverlayValues[611] = d611
		ps1084.OverlayValues[612] = d612
		ps1084.OverlayValues[613] = d613
		ps1084.OverlayValues[614] = d614
		ps1084.OverlayValues[615] = d615
		ps1084.OverlayValues[616] = d616
		ps1084.OverlayValues[617] = d617
		ps1084.OverlayValues[618] = d618
		ps1084.OverlayValues[619] = d619
		ps1084.OverlayValues[620] = d620
		ps1084.OverlayValues[621] = d621
		ps1084.OverlayValues[622] = d622
		ps1084.OverlayValues[623] = d623
		ps1084.OverlayValues[624] = d624
		ps1084.OverlayValues[625] = d625
		ps1084.OverlayValues[626] = d626
		ps1084.OverlayValues[627] = d627
		ps1084.OverlayValues[628] = d628
		ps1084.OverlayValues[629] = d629
		ps1084.OverlayValues[630] = d630
		ps1084.OverlayValues[913] = d913
		ps1084.OverlayValues[914] = d914
		ps1084.OverlayValues[915] = d915
		ps1084.OverlayValues[917] = d917
		ps1084.OverlayValues[918] = d918
		ps1084.OverlayValues[919] = d919
		ps1084.OverlayValues[920] = d920
		ps1084.OverlayValues[921] = d921
		ps1084.OverlayValues[922] = d922
		ps1084.OverlayValues[923] = d923
		ps1084.OverlayValues[925] = d925
		ps1084.OverlayValues[927] = d927
		ps1084.OverlayValues[928] = d928
		ps1084.OverlayValues[1081] = d1081
		ps1084.OverlayValues[1082] = d1082
		ps1084.OverlayValues[1085] = d1085
		snap1086 := d1
		snap1087 := d2
		snap1088 := d3
		snap1089 := d4
		snap1090 := d5
		snap1091 := d6
		snap1092 := d7
		snap1093 := d8
		snap1094 := d9
		snap1095 := d10
		snap1096 := d11
		snap1097 := d12
		snap1098 := d13
		snap1099 := d14
		snap1100 := d15
		snap1101 := d17
		snap1102 := d18
		snap1103 := d19
		snap1104 := d20
		snap1105 := d21
		snap1106 := d22
		snap1107 := d23
		snap1108 := d24
		snap1109 := d25
		snap1110 := d26
		snap1111 := d27
		snap1112 := d28
		snap1113 := d29
		snap1114 := d30
		snap1115 := d31
		snap1116 := d32
		snap1117 := d33
		snap1118 := d34
		snap1119 := d35
		snap1120 := d36
		snap1121 := d37
		snap1122 := d38
		snap1123 := d39
		snap1124 := d40
		snap1125 := d41
		snap1126 := d42
		snap1127 := d43
		snap1128 := d44
		snap1129 := d45
		snap1130 := d46
		snap1131 := d47
		snap1132 := d48
		snap1133 := d49
		snap1134 := d50
		snap1135 := d53
		snap1136 := d54
		snap1137 := d55
		snap1138 := d164
		snap1139 := d165
		snap1140 := d166
		snap1141 := d167
		snap1142 := d168
		snap1143 := d169
		snap1144 := d170
		snap1145 := d171
		snap1146 := d172
		snap1147 := d173
		snap1148 := d174
		snap1149 := d175
		snap1150 := d176
		snap1151 := d177
		snap1152 := d178
		snap1153 := d179
		snap1154 := d180
		snap1155 := d181
		snap1156 := d182
		snap1157 := d183
		snap1158 := d184
		snap1159 := d185
		snap1160 := d186
		snap1161 := d187
		snap1162 := d188
		snap1163 := d189
		snap1164 := d190
		snap1165 := d191
		snap1166 := d192
		snap1167 := d193
		snap1168 := d196
		snap1169 := d367
		snap1170 := d368
		snap1171 := d369
		snap1172 := d370
		snap1173 := d372
		snap1174 := d373
		snap1175 := d374
		snap1176 := d375
		snap1177 := d376
		snap1178 := d377
		snap1179 := d378
		snap1180 := d379
		snap1181 := d381
		snap1182 := d383
		snap1183 := d384
		snap1184 := d385
		snap1185 := d486
		snap1186 := d487
		snap1187 := d490
		snap1188 := d594
		snap1189 := d595
		snap1190 := d596
		snap1191 := d597
		snap1192 := d598
		snap1193 := d600
		snap1194 := d601
		snap1195 := d602
		snap1196 := d603
		snap1197 := d604
		snap1198 := d605
		snap1199 := d606
		snap1200 := d607
		snap1201 := d608
		snap1202 := d609
		snap1203 := d610
		snap1204 := d611
		snap1205 := d612
		snap1206 := d613
		snap1207 := d614
		snap1208 := d615
		snap1209 := d616
		snap1210 := d617
		snap1211 := d618
		snap1212 := d619
		snap1213 := d620
		snap1214 := d621
		snap1215 := d622
		snap1216 := d623
		snap1217 := d624
		snap1218 := d625
		snap1219 := d626
		snap1220 := d627
		snap1221 := d628
		snap1222 := d629
		snap1223 := d630
		snap1224 := d913
		snap1225 := d914
		snap1226 := d915
		snap1227 := d917
		snap1228 := d918
		snap1229 := d919
		snap1230 := d920
		snap1231 := d921
		snap1232 := d922
		snap1233 := d923
		snap1234 := d925
		snap1235 := d927
		snap1236 := d928
		snap1237 := d1081
		snap1238 := d1082
		snap1239 := d1085
		alloc1240 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps1083)
		}
		ctx.RestoreAllocState(alloc1240)
		d1 = snap1086
		d2 = snap1087
		d3 = snap1088
		d4 = snap1089
		d5 = snap1090
		d6 = snap1091
		d7 = snap1092
		d8 = snap1093
		d9 = snap1094
		d10 = snap1095
		d11 = snap1096
		d12 = snap1097
		d13 = snap1098
		d14 = snap1099
		d15 = snap1100
		d17 = snap1101
		d18 = snap1102
		d19 = snap1103
		d20 = snap1104
		d21 = snap1105
		d22 = snap1106
		d23 = snap1107
		d24 = snap1108
		d25 = snap1109
		d26 = snap1110
		d27 = snap1111
		d28 = snap1112
		d29 = snap1113
		d30 = snap1114
		d31 = snap1115
		d32 = snap1116
		d33 = snap1117
		d34 = snap1118
		d35 = snap1119
		d36 = snap1120
		d37 = snap1121
		d38 = snap1122
		d39 = snap1123
		d40 = snap1124
		d41 = snap1125
		d42 = snap1126
		d43 = snap1127
		d44 = snap1128
		d45 = snap1129
		d46 = snap1130
		d47 = snap1131
		d48 = snap1132
		d49 = snap1133
		d50 = snap1134
		d53 = snap1135
		d54 = snap1136
		d55 = snap1137
		d164 = snap1138
		d165 = snap1139
		d166 = snap1140
		d167 = snap1141
		d168 = snap1142
		d169 = snap1143
		d170 = snap1144
		d171 = snap1145
		d172 = snap1146
		d173 = snap1147
		d174 = snap1148
		d175 = snap1149
		d176 = snap1150
		d177 = snap1151
		d178 = snap1152
		d179 = snap1153
		d180 = snap1154
		d181 = snap1155
		d182 = snap1156
		d183 = snap1157
		d184 = snap1158
		d185 = snap1159
		d186 = snap1160
		d187 = snap1161
		d188 = snap1162
		d189 = snap1163
		d190 = snap1164
		d191 = snap1165
		d192 = snap1166
		d193 = snap1167
		d196 = snap1168
		d367 = snap1169
		d368 = snap1170
		d369 = snap1171
		d370 = snap1172
		d372 = snap1173
		d373 = snap1174
		d374 = snap1175
		d375 = snap1176
		d376 = snap1177
		d377 = snap1178
		d378 = snap1179
		d379 = snap1180
		d381 = snap1181
		d383 = snap1182
		d384 = snap1183
		d385 = snap1184
		d486 = snap1185
		d487 = snap1186
		d490 = snap1187
		d594 = snap1188
		d595 = snap1189
		d596 = snap1190
		d597 = snap1191
		d598 = snap1192
		d600 = snap1193
		d601 = snap1194
		d602 = snap1195
		d603 = snap1196
		d604 = snap1197
		d605 = snap1198
		d606 = snap1199
		d607 = snap1200
		d608 = snap1201
		d609 = snap1202
		d610 = snap1203
		d611 = snap1204
		d612 = snap1205
		d613 = snap1206
		d614 = snap1207
		d615 = snap1208
		d616 = snap1209
		d617 = snap1210
		d618 = snap1211
		d619 = snap1212
		d620 = snap1213
		d621 = snap1214
		d622 = snap1215
		d623 = snap1216
		d624 = snap1217
		d625 = snap1218
		d626 = snap1219
		d627 = snap1220
		d628 = snap1221
		d629 = snap1222
		d630 = snap1223
		d913 = snap1224
		d914 = snap1225
		d915 = snap1226
		d917 = snap1227
		d918 = snap1228
		d919 = snap1229
		d920 = snap1230
		d921 = snap1231
		d922 = snap1232
		d923 = snap1233
		d925 = snap1234
		d927 = snap1235
		d928 = snap1236
		d1081 = snap1237
		d1082 = snap1238
		d1085 = snap1239
		if !bbs[10].Rendered {
			return bbs[10].RenderPS(ps1084)
		}
		return result
		ctx.FreeDesc(&d920)
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
		if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != scm.LocNone {
			d176 = ps.OverlayValues[176]
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
		if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
			d196 = ps.OverlayValues[196]
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
		if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != scm.LocNone {
			d381 = ps.OverlayValues[381]
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
		if len(ps.OverlayValues) > 486 && ps.OverlayValues[486].Loc != scm.LocNone {
			d486 = ps.OverlayValues[486]
		}
		if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != scm.LocNone {
			d487 = ps.OverlayValues[487]
		}
		if len(ps.OverlayValues) > 490 && ps.OverlayValues[490].Loc != scm.LocNone {
			d490 = ps.OverlayValues[490]
		}
		if len(ps.OverlayValues) > 594 && ps.OverlayValues[594].Loc != scm.LocNone {
			d594 = ps.OverlayValues[594]
		}
		if len(ps.OverlayValues) > 595 && ps.OverlayValues[595].Loc != scm.LocNone {
			d595 = ps.OverlayValues[595]
		}
		if len(ps.OverlayValues) > 596 && ps.OverlayValues[596].Loc != scm.LocNone {
			d596 = ps.OverlayValues[596]
		}
		if len(ps.OverlayValues) > 597 && ps.OverlayValues[597].Loc != scm.LocNone {
			d597 = ps.OverlayValues[597]
		}
		if len(ps.OverlayValues) > 598 && ps.OverlayValues[598].Loc != scm.LocNone {
			d598 = ps.OverlayValues[598]
		}
		if len(ps.OverlayValues) > 600 && ps.OverlayValues[600].Loc != scm.LocNone {
			d600 = ps.OverlayValues[600]
		}
		if len(ps.OverlayValues) > 601 && ps.OverlayValues[601].Loc != scm.LocNone {
			d601 = ps.OverlayValues[601]
		}
		if len(ps.OverlayValues) > 602 && ps.OverlayValues[602].Loc != scm.LocNone {
			d602 = ps.OverlayValues[602]
		}
		if len(ps.OverlayValues) > 603 && ps.OverlayValues[603].Loc != scm.LocNone {
			d603 = ps.OverlayValues[603]
		}
		if len(ps.OverlayValues) > 604 && ps.OverlayValues[604].Loc != scm.LocNone {
			d604 = ps.OverlayValues[604]
		}
		if len(ps.OverlayValues) > 605 && ps.OverlayValues[605].Loc != scm.LocNone {
			d605 = ps.OverlayValues[605]
		}
		if len(ps.OverlayValues) > 606 && ps.OverlayValues[606].Loc != scm.LocNone {
			d606 = ps.OverlayValues[606]
		}
		if len(ps.OverlayValues) > 607 && ps.OverlayValues[607].Loc != scm.LocNone {
			d607 = ps.OverlayValues[607]
		}
		if len(ps.OverlayValues) > 608 && ps.OverlayValues[608].Loc != scm.LocNone {
			d608 = ps.OverlayValues[608]
		}
		if len(ps.OverlayValues) > 609 && ps.OverlayValues[609].Loc != scm.LocNone {
			d609 = ps.OverlayValues[609]
		}
		if len(ps.OverlayValues) > 610 && ps.OverlayValues[610].Loc != scm.LocNone {
			d610 = ps.OverlayValues[610]
		}
		if len(ps.OverlayValues) > 611 && ps.OverlayValues[611].Loc != scm.LocNone {
			d611 = ps.OverlayValues[611]
		}
		if len(ps.OverlayValues) > 612 && ps.OverlayValues[612].Loc != scm.LocNone {
			d612 = ps.OverlayValues[612]
		}
		if len(ps.OverlayValues) > 613 && ps.OverlayValues[613].Loc != scm.LocNone {
			d613 = ps.OverlayValues[613]
		}
		if len(ps.OverlayValues) > 614 && ps.OverlayValues[614].Loc != scm.LocNone {
			d614 = ps.OverlayValues[614]
		}
		if len(ps.OverlayValues) > 615 && ps.OverlayValues[615].Loc != scm.LocNone {
			d615 = ps.OverlayValues[615]
		}
		if len(ps.OverlayValues) > 616 && ps.OverlayValues[616].Loc != scm.LocNone {
			d616 = ps.OverlayValues[616]
		}
		if len(ps.OverlayValues) > 617 && ps.OverlayValues[617].Loc != scm.LocNone {
			d617 = ps.OverlayValues[617]
		}
		if len(ps.OverlayValues) > 618 && ps.OverlayValues[618].Loc != scm.LocNone {
			d618 = ps.OverlayValues[618]
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
		if len(ps.OverlayValues) > 624 && ps.OverlayValues[624].Loc != scm.LocNone {
			d624 = ps.OverlayValues[624]
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
		if len(ps.OverlayValues) > 913 && ps.OverlayValues[913].Loc != scm.LocNone {
			d913 = ps.OverlayValues[913]
		}
		if len(ps.OverlayValues) > 914 && ps.OverlayValues[914].Loc != scm.LocNone {
			d914 = ps.OverlayValues[914]
		}
		if len(ps.OverlayValues) > 915 && ps.OverlayValues[915].Loc != scm.LocNone {
			d915 = ps.OverlayValues[915]
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
		if len(ps.OverlayValues) > 925 && ps.OverlayValues[925].Loc != scm.LocNone {
			d925 = ps.OverlayValues[925]
		}
		if len(ps.OverlayValues) > 927 && ps.OverlayValues[927].Loc != scm.LocNone {
			d927 = ps.OverlayValues[927]
		}
		if len(ps.OverlayValues) > 928 && ps.OverlayValues[928].Loc != scm.LocNone {
			d928 = ps.OverlayValues[928]
		}
		if len(ps.OverlayValues) > 1081 && ps.OverlayValues[1081].Loc != scm.LocNone {
			d1081 = ps.OverlayValues[1081]
		}
		if len(ps.OverlayValues) > 1082 && ps.OverlayValues[1082].Loc != scm.LocNone {
			d1082 = ps.OverlayValues[1082]
		}
		if len(ps.OverlayValues) > 1085 && ps.OverlayValues[1085].Loc != scm.LocNone {
			d1085 = ps.OverlayValues[1085]
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
			d1241 = d5
			if d1241.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d1241)
			d1242 = d1241
			if d1242.Loc == scm.LocImm {
				d1242 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1242.Type, Imm: scm.NewInt(int64(uint64(d1242.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d1242.Reg, 32)
				ctx.EmitShrRegImm8(d1242.Reg, 32)
			}
			ctx.EmitStoreToStack(d1242, int32(bbs[8].PhiBase)+int32(0))
			d1243 = d7
			if d1243.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d1243)
			d1244 = d1243
			if d1244.Loc == scm.LocImm {
				d1244 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1244.Type, Imm: scm.NewInt(int64(uint64(d1244.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d1244.Reg, 32)
				ctx.EmitShrRegImm8(d1244.Reg, 32)
			}
			ctx.EmitStoreToStack(d1244, int32(bbs[8].PhiBase)+int32(16))
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
		ps1245 := scm.PhiState{General: ps.General}
		ps1245.OverlayValues = make([]scm.JITValueDesc, 1245)
		ps1245.OverlayValues[1] = d1
		ps1245.OverlayValues[2] = d2
		ps1245.OverlayValues[3] = d3
		ps1245.OverlayValues[4] = d4
		ps1245.OverlayValues[5] = d5
		ps1245.OverlayValues[6] = d6
		ps1245.OverlayValues[7] = d7
		ps1245.OverlayValues[8] = d8
		ps1245.OverlayValues[9] = d9
		ps1245.OverlayValues[10] = d10
		ps1245.OverlayValues[11] = d11
		ps1245.OverlayValues[12] = d12
		ps1245.OverlayValues[13] = d13
		ps1245.OverlayValues[14] = d14
		ps1245.OverlayValues[15] = d15
		ps1245.OverlayValues[17] = d17
		ps1245.OverlayValues[18] = d18
		ps1245.OverlayValues[19] = d19
		ps1245.OverlayValues[20] = d20
		ps1245.OverlayValues[21] = d21
		ps1245.OverlayValues[22] = d22
		ps1245.OverlayValues[23] = d23
		ps1245.OverlayValues[24] = d24
		ps1245.OverlayValues[25] = d25
		ps1245.OverlayValues[26] = d26
		ps1245.OverlayValues[27] = d27
		ps1245.OverlayValues[28] = d28
		ps1245.OverlayValues[29] = d29
		ps1245.OverlayValues[30] = d30
		ps1245.OverlayValues[31] = d31
		ps1245.OverlayValues[32] = d32
		ps1245.OverlayValues[33] = d33
		ps1245.OverlayValues[34] = d34
		ps1245.OverlayValues[35] = d35
		ps1245.OverlayValues[36] = d36
		ps1245.OverlayValues[37] = d37
		ps1245.OverlayValues[38] = d38
		ps1245.OverlayValues[39] = d39
		ps1245.OverlayValues[40] = d40
		ps1245.OverlayValues[41] = d41
		ps1245.OverlayValues[42] = d42
		ps1245.OverlayValues[43] = d43
		ps1245.OverlayValues[44] = d44
		ps1245.OverlayValues[45] = d45
		ps1245.OverlayValues[46] = d46
		ps1245.OverlayValues[47] = d47
		ps1245.OverlayValues[48] = d48
		ps1245.OverlayValues[49] = d49
		ps1245.OverlayValues[50] = d50
		ps1245.OverlayValues[53] = d53
		ps1245.OverlayValues[54] = d54
		ps1245.OverlayValues[55] = d55
		ps1245.OverlayValues[164] = d164
		ps1245.OverlayValues[165] = d165
		ps1245.OverlayValues[166] = d166
		ps1245.OverlayValues[167] = d167
		ps1245.OverlayValues[168] = d168
		ps1245.OverlayValues[169] = d169
		ps1245.OverlayValues[170] = d170
		ps1245.OverlayValues[171] = d171
		ps1245.OverlayValues[172] = d172
		ps1245.OverlayValues[173] = d173
		ps1245.OverlayValues[174] = d174
		ps1245.OverlayValues[175] = d175
		ps1245.OverlayValues[176] = d176
		ps1245.OverlayValues[177] = d177
		ps1245.OverlayValues[178] = d178
		ps1245.OverlayValues[179] = d179
		ps1245.OverlayValues[180] = d180
		ps1245.OverlayValues[181] = d181
		ps1245.OverlayValues[182] = d182
		ps1245.OverlayValues[183] = d183
		ps1245.OverlayValues[184] = d184
		ps1245.OverlayValues[185] = d185
		ps1245.OverlayValues[186] = d186
		ps1245.OverlayValues[187] = d187
		ps1245.OverlayValues[188] = d188
		ps1245.OverlayValues[189] = d189
		ps1245.OverlayValues[190] = d190
		ps1245.OverlayValues[191] = d191
		ps1245.OverlayValues[192] = d192
		ps1245.OverlayValues[193] = d193
		ps1245.OverlayValues[196] = d196
		ps1245.OverlayValues[367] = d367
		ps1245.OverlayValues[368] = d368
		ps1245.OverlayValues[369] = d369
		ps1245.OverlayValues[370] = d370
		ps1245.OverlayValues[372] = d372
		ps1245.OverlayValues[373] = d373
		ps1245.OverlayValues[374] = d374
		ps1245.OverlayValues[375] = d375
		ps1245.OverlayValues[376] = d376
		ps1245.OverlayValues[377] = d377
		ps1245.OverlayValues[378] = d378
		ps1245.OverlayValues[379] = d379
		ps1245.OverlayValues[381] = d381
		ps1245.OverlayValues[383] = d383
		ps1245.OverlayValues[384] = d384
		ps1245.OverlayValues[385] = d385
		ps1245.OverlayValues[486] = d486
		ps1245.OverlayValues[487] = d487
		ps1245.OverlayValues[490] = d490
		ps1245.OverlayValues[594] = d594
		ps1245.OverlayValues[595] = d595
		ps1245.OverlayValues[596] = d596
		ps1245.OverlayValues[597] = d597
		ps1245.OverlayValues[598] = d598
		ps1245.OverlayValues[600] = d600
		ps1245.OverlayValues[601] = d601
		ps1245.OverlayValues[602] = d602
		ps1245.OverlayValues[603] = d603
		ps1245.OverlayValues[604] = d604
		ps1245.OverlayValues[605] = d605
		ps1245.OverlayValues[606] = d606
		ps1245.OverlayValues[607] = d607
		ps1245.OverlayValues[608] = d608
		ps1245.OverlayValues[609] = d609
		ps1245.OverlayValues[610] = d610
		ps1245.OverlayValues[611] = d611
		ps1245.OverlayValues[612] = d612
		ps1245.OverlayValues[613] = d613
		ps1245.OverlayValues[614] = d614
		ps1245.OverlayValues[615] = d615
		ps1245.OverlayValues[616] = d616
		ps1245.OverlayValues[617] = d617
		ps1245.OverlayValues[618] = d618
		ps1245.OverlayValues[619] = d619
		ps1245.OverlayValues[620] = d620
		ps1245.OverlayValues[621] = d621
		ps1245.OverlayValues[622] = d622
		ps1245.OverlayValues[623] = d623
		ps1245.OverlayValues[624] = d624
		ps1245.OverlayValues[625] = d625
		ps1245.OverlayValues[626] = d626
		ps1245.OverlayValues[627] = d627
		ps1245.OverlayValues[628] = d628
		ps1245.OverlayValues[629] = d629
		ps1245.OverlayValues[630] = d630
		ps1245.OverlayValues[913] = d913
		ps1245.OverlayValues[914] = d914
		ps1245.OverlayValues[915] = d915
		ps1245.OverlayValues[917] = d917
		ps1245.OverlayValues[918] = d918
		ps1245.OverlayValues[919] = d919
		ps1245.OverlayValues[920] = d920
		ps1245.OverlayValues[921] = d921
		ps1245.OverlayValues[922] = d922
		ps1245.OverlayValues[923] = d923
		ps1245.OverlayValues[925] = d925
		ps1245.OverlayValues[927] = d927
		ps1245.OverlayValues[928] = d928
		ps1245.OverlayValues[1081] = d1081
		ps1245.OverlayValues[1082] = d1082
		ps1245.OverlayValues[1085] = d1085
		ps1245.OverlayValues[1241] = d1241
		ps1245.OverlayValues[1242] = d1242
		ps1245.OverlayValues[1243] = d1243
		ps1245.OverlayValues[1244] = d1244
		ps1245.PhiValues = make([]scm.JITValueDesc, 2)
		d1246 = d5
		ps1245.PhiValues[0] = d1246
		d1247 = d7
		ps1245.PhiValues[1] = d1247
		if ps1245.General && bbs[8].Rendered {
			ctx.EmitJmp(lbl9)
			return result
		}
		return bbs[8].RenderPS(ps1245)
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
		if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != scm.LocNone {
			d176 = ps.OverlayValues[176]
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
		if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
			d196 = ps.OverlayValues[196]
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
		if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != scm.LocNone {
			d381 = ps.OverlayValues[381]
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
		if len(ps.OverlayValues) > 486 && ps.OverlayValues[486].Loc != scm.LocNone {
			d486 = ps.OverlayValues[486]
		}
		if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != scm.LocNone {
			d487 = ps.OverlayValues[487]
		}
		if len(ps.OverlayValues) > 490 && ps.OverlayValues[490].Loc != scm.LocNone {
			d490 = ps.OverlayValues[490]
		}
		if len(ps.OverlayValues) > 594 && ps.OverlayValues[594].Loc != scm.LocNone {
			d594 = ps.OverlayValues[594]
		}
		if len(ps.OverlayValues) > 595 && ps.OverlayValues[595].Loc != scm.LocNone {
			d595 = ps.OverlayValues[595]
		}
		if len(ps.OverlayValues) > 596 && ps.OverlayValues[596].Loc != scm.LocNone {
			d596 = ps.OverlayValues[596]
		}
		if len(ps.OverlayValues) > 597 && ps.OverlayValues[597].Loc != scm.LocNone {
			d597 = ps.OverlayValues[597]
		}
		if len(ps.OverlayValues) > 598 && ps.OverlayValues[598].Loc != scm.LocNone {
			d598 = ps.OverlayValues[598]
		}
		if len(ps.OverlayValues) > 600 && ps.OverlayValues[600].Loc != scm.LocNone {
			d600 = ps.OverlayValues[600]
		}
		if len(ps.OverlayValues) > 601 && ps.OverlayValues[601].Loc != scm.LocNone {
			d601 = ps.OverlayValues[601]
		}
		if len(ps.OverlayValues) > 602 && ps.OverlayValues[602].Loc != scm.LocNone {
			d602 = ps.OverlayValues[602]
		}
		if len(ps.OverlayValues) > 603 && ps.OverlayValues[603].Loc != scm.LocNone {
			d603 = ps.OverlayValues[603]
		}
		if len(ps.OverlayValues) > 604 && ps.OverlayValues[604].Loc != scm.LocNone {
			d604 = ps.OverlayValues[604]
		}
		if len(ps.OverlayValues) > 605 && ps.OverlayValues[605].Loc != scm.LocNone {
			d605 = ps.OverlayValues[605]
		}
		if len(ps.OverlayValues) > 606 && ps.OverlayValues[606].Loc != scm.LocNone {
			d606 = ps.OverlayValues[606]
		}
		if len(ps.OverlayValues) > 607 && ps.OverlayValues[607].Loc != scm.LocNone {
			d607 = ps.OverlayValues[607]
		}
		if len(ps.OverlayValues) > 608 && ps.OverlayValues[608].Loc != scm.LocNone {
			d608 = ps.OverlayValues[608]
		}
		if len(ps.OverlayValues) > 609 && ps.OverlayValues[609].Loc != scm.LocNone {
			d609 = ps.OverlayValues[609]
		}
		if len(ps.OverlayValues) > 610 && ps.OverlayValues[610].Loc != scm.LocNone {
			d610 = ps.OverlayValues[610]
		}
		if len(ps.OverlayValues) > 611 && ps.OverlayValues[611].Loc != scm.LocNone {
			d611 = ps.OverlayValues[611]
		}
		if len(ps.OverlayValues) > 612 && ps.OverlayValues[612].Loc != scm.LocNone {
			d612 = ps.OverlayValues[612]
		}
		if len(ps.OverlayValues) > 613 && ps.OverlayValues[613].Loc != scm.LocNone {
			d613 = ps.OverlayValues[613]
		}
		if len(ps.OverlayValues) > 614 && ps.OverlayValues[614].Loc != scm.LocNone {
			d614 = ps.OverlayValues[614]
		}
		if len(ps.OverlayValues) > 615 && ps.OverlayValues[615].Loc != scm.LocNone {
			d615 = ps.OverlayValues[615]
		}
		if len(ps.OverlayValues) > 616 && ps.OverlayValues[616].Loc != scm.LocNone {
			d616 = ps.OverlayValues[616]
		}
		if len(ps.OverlayValues) > 617 && ps.OverlayValues[617].Loc != scm.LocNone {
			d617 = ps.OverlayValues[617]
		}
		if len(ps.OverlayValues) > 618 && ps.OverlayValues[618].Loc != scm.LocNone {
			d618 = ps.OverlayValues[618]
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
		if len(ps.OverlayValues) > 624 && ps.OverlayValues[624].Loc != scm.LocNone {
			d624 = ps.OverlayValues[624]
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
		if len(ps.OverlayValues) > 913 && ps.OverlayValues[913].Loc != scm.LocNone {
			d913 = ps.OverlayValues[913]
		}
		if len(ps.OverlayValues) > 914 && ps.OverlayValues[914].Loc != scm.LocNone {
			d914 = ps.OverlayValues[914]
		}
		if len(ps.OverlayValues) > 915 && ps.OverlayValues[915].Loc != scm.LocNone {
			d915 = ps.OverlayValues[915]
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
		if len(ps.OverlayValues) > 925 && ps.OverlayValues[925].Loc != scm.LocNone {
			d925 = ps.OverlayValues[925]
		}
		if len(ps.OverlayValues) > 927 && ps.OverlayValues[927].Loc != scm.LocNone {
			d927 = ps.OverlayValues[927]
		}
		if len(ps.OverlayValues) > 928 && ps.OverlayValues[928].Loc != scm.LocNone {
			d928 = ps.OverlayValues[928]
		}
		if len(ps.OverlayValues) > 1081 && ps.OverlayValues[1081].Loc != scm.LocNone {
			d1081 = ps.OverlayValues[1081]
		}
		if len(ps.OverlayValues) > 1082 && ps.OverlayValues[1082].Loc != scm.LocNone {
			d1082 = ps.OverlayValues[1082]
		}
		if len(ps.OverlayValues) > 1085 && ps.OverlayValues[1085].Loc != scm.LocNone {
			d1085 = ps.OverlayValues[1085]
		}
		if len(ps.OverlayValues) > 1241 && ps.OverlayValues[1241].Loc != scm.LocNone {
			d1241 = ps.OverlayValues[1241]
		}
		if len(ps.OverlayValues) > 1242 && ps.OverlayValues[1242].Loc != scm.LocNone {
			d1242 = ps.OverlayValues[1242]
		}
		if len(ps.OverlayValues) > 1243 && ps.OverlayValues[1243].Loc != scm.LocNone {
			d1243 = ps.OverlayValues[1243]
		}
		if len(ps.OverlayValues) > 1244 && ps.OverlayValues[1244].Loc != scm.LocNone {
			d1244 = ps.OverlayValues[1244]
		}
		if len(ps.OverlayValues) > 1246 && ps.OverlayValues[1246].Loc != scm.LocNone {
			d1246 = ps.OverlayValues[1246]
		}
		if len(ps.OverlayValues) > 1247 && ps.OverlayValues[1247].Loc != scm.LocNone {
			d1247 = ps.OverlayValues[1247]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d8)
		ctx.EnsureDesc(&d9)
		ctx.EnsureDescsTogether(&d8, &d9)
		var d1248 scm.JITValueDesc
		if d8.Loc == scm.LocImm && d9.Loc == scm.LocImm {
			d1248 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() + d9.Imm.Int())}
		} else if d9.Loc == scm.LocImm && d9.Imm.Int() == 0 {
			r105 := ctx.AllocRegExcept(d8.Reg)
			ctx.EmitMovRegReg(r105, d8.Reg)
			d1248 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r105}
			ctx.BindReg(r105, &d1248)
		} else if d8.Loc == scm.LocImm && d8.Imm.Int() == 0 {
			d1248 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d9.Reg}
			ctx.BindReg(d9.Reg, &d1248)
		} else if d8.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d9.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d8.Imm.Int()))
			ctx.EmitAddInt64(scratch, d9.Reg)
			d1248 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1248)
		} else if d9.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d8.Reg)
			ctx.EmitMovRegReg(scratch, d8.Reg)
			if d9.Imm.Int() >= -2147483648 && d9.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d9.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d9.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d1248 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1248)
		} else {
			r106 := ctx.AllocRegExcept(d8.Reg, d9.Reg)
			ctx.EmitMovRegReg(r106, d8.Reg)
			ctx.EmitAddInt64(r106, d9.Reg)
			d1248 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r106}
			ctx.BindReg(r106, &d1248)
		}
		if d1248.Loc == scm.LocImm {
			d1248 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1248.Type, Imm: scm.NewInt(int64(uint64(d1248.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d1248.Reg, 32)
			ctx.EmitShrRegImm8(d1248.Reg, 32)
		}
		if d1248.Loc == scm.LocReg && d8.Loc == scm.LocReg && d1248.Reg == d8.Reg {
			ctx.TransferReg(d8.Reg)
			d8.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d1248)
		var d1249 scm.JITValueDesc
		if d1248.Loc == scm.LocImm {
			d1249 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1248.Imm.Int() / 2)}
		} else {
			r107 := ctx.AllocRegExcept(d1248.Reg)
			ctx.EmitMovRegReg(r107, d1248.Reg)
			ctx.EmitShrRegImm8(r107, 1)
			d1249 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r107}
			ctx.BindReg(r107, &d1249)
		}
		if d1249.Loc == scm.LocImm {
			d1249 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1249.Type, Imm: scm.NewInt(int64(uint64(d1249.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d1249.Reg, 32)
			ctx.EmitShrRegImm8(d1249.Reg, 32)
		}
		if d1249.Loc == scm.LocReg && d1248.Loc == scm.LocReg && d1249.Reg == d1248.Reg {
			ctx.TransferReg(d1248.Reg)
			d1248.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d1249)
		ctx.EmitStoreToStack(d1249, int32(bbs[1].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d1249)
		ctx.FreeDesc(&d1248)
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
			d1250 = d8
			if d1250.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d1250)
			d1251 = d1250
			if d1251.Loc == scm.LocImm {
				d1251 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1251.Type, Imm: scm.NewInt(int64(uint64(d1251.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d1251.Reg, 32)
				ctx.EmitShrRegImm8(d1251.Reg, 32)
			}
			ctx.EmitStoreToStack(d1251, int32(bbs[1].PhiBase)+int32(16))
			d1252 = d9
			if d1252.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d1252)
			d1253 = d1252
			if d1253.Loc == scm.LocImm {
				d1253 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1253.Type, Imm: scm.NewInt(int64(uint64(d1253.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d1253.Reg, 32)
				ctx.EmitShrRegImm8(d1253.Reg, 32)
			}
			ctx.EmitStoreToStack(d1253, int32(bbs[1].PhiBase)+int32(32))
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
		ps1254 := scm.PhiState{General: ps.General}
		ps1254.OverlayValues = make([]scm.JITValueDesc, 1254)
		ps1254.OverlayValues[1] = d1
		ps1254.OverlayValues[2] = d2
		ps1254.OverlayValues[3] = d3
		ps1254.OverlayValues[4] = d4
		ps1254.OverlayValues[5] = d5
		ps1254.OverlayValues[6] = d6
		ps1254.OverlayValues[7] = d7
		ps1254.OverlayValues[8] = d8
		ps1254.OverlayValues[9] = d9
		ps1254.OverlayValues[10] = d10
		ps1254.OverlayValues[11] = d11
		ps1254.OverlayValues[12] = d12
		ps1254.OverlayValues[13] = d13
		ps1254.OverlayValues[14] = d14
		ps1254.OverlayValues[15] = d15
		ps1254.OverlayValues[17] = d17
		ps1254.OverlayValues[18] = d18
		ps1254.OverlayValues[19] = d19
		ps1254.OverlayValues[20] = d20
		ps1254.OverlayValues[21] = d21
		ps1254.OverlayValues[22] = d22
		ps1254.OverlayValues[23] = d23
		ps1254.OverlayValues[24] = d24
		ps1254.OverlayValues[25] = d25
		ps1254.OverlayValues[26] = d26
		ps1254.OverlayValues[27] = d27
		ps1254.OverlayValues[28] = d28
		ps1254.OverlayValues[29] = d29
		ps1254.OverlayValues[30] = d30
		ps1254.OverlayValues[31] = d31
		ps1254.OverlayValues[32] = d32
		ps1254.OverlayValues[33] = d33
		ps1254.OverlayValues[34] = d34
		ps1254.OverlayValues[35] = d35
		ps1254.OverlayValues[36] = d36
		ps1254.OverlayValues[37] = d37
		ps1254.OverlayValues[38] = d38
		ps1254.OverlayValues[39] = d39
		ps1254.OverlayValues[40] = d40
		ps1254.OverlayValues[41] = d41
		ps1254.OverlayValues[42] = d42
		ps1254.OverlayValues[43] = d43
		ps1254.OverlayValues[44] = d44
		ps1254.OverlayValues[45] = d45
		ps1254.OverlayValues[46] = d46
		ps1254.OverlayValues[47] = d47
		ps1254.OverlayValues[48] = d48
		ps1254.OverlayValues[49] = d49
		ps1254.OverlayValues[50] = d50
		ps1254.OverlayValues[53] = d53
		ps1254.OverlayValues[54] = d54
		ps1254.OverlayValues[55] = d55
		ps1254.OverlayValues[164] = d164
		ps1254.OverlayValues[165] = d165
		ps1254.OverlayValues[166] = d166
		ps1254.OverlayValues[167] = d167
		ps1254.OverlayValues[168] = d168
		ps1254.OverlayValues[169] = d169
		ps1254.OverlayValues[170] = d170
		ps1254.OverlayValues[171] = d171
		ps1254.OverlayValues[172] = d172
		ps1254.OverlayValues[173] = d173
		ps1254.OverlayValues[174] = d174
		ps1254.OverlayValues[175] = d175
		ps1254.OverlayValues[176] = d176
		ps1254.OverlayValues[177] = d177
		ps1254.OverlayValues[178] = d178
		ps1254.OverlayValues[179] = d179
		ps1254.OverlayValues[180] = d180
		ps1254.OverlayValues[181] = d181
		ps1254.OverlayValues[182] = d182
		ps1254.OverlayValues[183] = d183
		ps1254.OverlayValues[184] = d184
		ps1254.OverlayValues[185] = d185
		ps1254.OverlayValues[186] = d186
		ps1254.OverlayValues[187] = d187
		ps1254.OverlayValues[188] = d188
		ps1254.OverlayValues[189] = d189
		ps1254.OverlayValues[190] = d190
		ps1254.OverlayValues[191] = d191
		ps1254.OverlayValues[192] = d192
		ps1254.OverlayValues[193] = d193
		ps1254.OverlayValues[196] = d196
		ps1254.OverlayValues[367] = d367
		ps1254.OverlayValues[368] = d368
		ps1254.OverlayValues[369] = d369
		ps1254.OverlayValues[370] = d370
		ps1254.OverlayValues[372] = d372
		ps1254.OverlayValues[373] = d373
		ps1254.OverlayValues[374] = d374
		ps1254.OverlayValues[375] = d375
		ps1254.OverlayValues[376] = d376
		ps1254.OverlayValues[377] = d377
		ps1254.OverlayValues[378] = d378
		ps1254.OverlayValues[379] = d379
		ps1254.OverlayValues[381] = d381
		ps1254.OverlayValues[383] = d383
		ps1254.OverlayValues[384] = d384
		ps1254.OverlayValues[385] = d385
		ps1254.OverlayValues[486] = d486
		ps1254.OverlayValues[487] = d487
		ps1254.OverlayValues[490] = d490
		ps1254.OverlayValues[594] = d594
		ps1254.OverlayValues[595] = d595
		ps1254.OverlayValues[596] = d596
		ps1254.OverlayValues[597] = d597
		ps1254.OverlayValues[598] = d598
		ps1254.OverlayValues[600] = d600
		ps1254.OverlayValues[601] = d601
		ps1254.OverlayValues[602] = d602
		ps1254.OverlayValues[603] = d603
		ps1254.OverlayValues[604] = d604
		ps1254.OverlayValues[605] = d605
		ps1254.OverlayValues[606] = d606
		ps1254.OverlayValues[607] = d607
		ps1254.OverlayValues[608] = d608
		ps1254.OverlayValues[609] = d609
		ps1254.OverlayValues[610] = d610
		ps1254.OverlayValues[611] = d611
		ps1254.OverlayValues[612] = d612
		ps1254.OverlayValues[613] = d613
		ps1254.OverlayValues[614] = d614
		ps1254.OverlayValues[615] = d615
		ps1254.OverlayValues[616] = d616
		ps1254.OverlayValues[617] = d617
		ps1254.OverlayValues[618] = d618
		ps1254.OverlayValues[619] = d619
		ps1254.OverlayValues[620] = d620
		ps1254.OverlayValues[621] = d621
		ps1254.OverlayValues[622] = d622
		ps1254.OverlayValues[623] = d623
		ps1254.OverlayValues[624] = d624
		ps1254.OverlayValues[625] = d625
		ps1254.OverlayValues[626] = d626
		ps1254.OverlayValues[627] = d627
		ps1254.OverlayValues[628] = d628
		ps1254.OverlayValues[629] = d629
		ps1254.OverlayValues[630] = d630
		ps1254.OverlayValues[913] = d913
		ps1254.OverlayValues[914] = d914
		ps1254.OverlayValues[915] = d915
		ps1254.OverlayValues[917] = d917
		ps1254.OverlayValues[918] = d918
		ps1254.OverlayValues[919] = d919
		ps1254.OverlayValues[920] = d920
		ps1254.OverlayValues[921] = d921
		ps1254.OverlayValues[922] = d922
		ps1254.OverlayValues[923] = d923
		ps1254.OverlayValues[925] = d925
		ps1254.OverlayValues[927] = d927
		ps1254.OverlayValues[928] = d928
		ps1254.OverlayValues[1081] = d1081
		ps1254.OverlayValues[1082] = d1082
		ps1254.OverlayValues[1085] = d1085
		ps1254.OverlayValues[1241] = d1241
		ps1254.OverlayValues[1242] = d1242
		ps1254.OverlayValues[1243] = d1243
		ps1254.OverlayValues[1244] = d1244
		ps1254.OverlayValues[1246] = d1246
		ps1254.OverlayValues[1247] = d1247
		ps1254.OverlayValues[1248] = d1248
		ps1254.OverlayValues[1249] = d1249
		ps1254.OverlayValues[1250] = d1250
		ps1254.OverlayValues[1251] = d1251
		ps1254.OverlayValues[1252] = d1252
		ps1254.OverlayValues[1253] = d1253
		ps1254.PhiValues = make([]scm.JITValueDesc, 3)
		d1255 = d8
		ps1254.PhiValues[1] = d1255
		d1256 = d9
		ps1254.PhiValues[2] = d1256
		if ps1254.General && bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		return bbs[1].RenderPS(ps1254)
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
		if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != scm.LocNone {
			d176 = ps.OverlayValues[176]
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
		if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
			d196 = ps.OverlayValues[196]
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
		if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != scm.LocNone {
			d381 = ps.OverlayValues[381]
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
		if len(ps.OverlayValues) > 486 && ps.OverlayValues[486].Loc != scm.LocNone {
			d486 = ps.OverlayValues[486]
		}
		if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != scm.LocNone {
			d487 = ps.OverlayValues[487]
		}
		if len(ps.OverlayValues) > 490 && ps.OverlayValues[490].Loc != scm.LocNone {
			d490 = ps.OverlayValues[490]
		}
		if len(ps.OverlayValues) > 594 && ps.OverlayValues[594].Loc != scm.LocNone {
			d594 = ps.OverlayValues[594]
		}
		if len(ps.OverlayValues) > 595 && ps.OverlayValues[595].Loc != scm.LocNone {
			d595 = ps.OverlayValues[595]
		}
		if len(ps.OverlayValues) > 596 && ps.OverlayValues[596].Loc != scm.LocNone {
			d596 = ps.OverlayValues[596]
		}
		if len(ps.OverlayValues) > 597 && ps.OverlayValues[597].Loc != scm.LocNone {
			d597 = ps.OverlayValues[597]
		}
		if len(ps.OverlayValues) > 598 && ps.OverlayValues[598].Loc != scm.LocNone {
			d598 = ps.OverlayValues[598]
		}
		if len(ps.OverlayValues) > 600 && ps.OverlayValues[600].Loc != scm.LocNone {
			d600 = ps.OverlayValues[600]
		}
		if len(ps.OverlayValues) > 601 && ps.OverlayValues[601].Loc != scm.LocNone {
			d601 = ps.OverlayValues[601]
		}
		if len(ps.OverlayValues) > 602 && ps.OverlayValues[602].Loc != scm.LocNone {
			d602 = ps.OverlayValues[602]
		}
		if len(ps.OverlayValues) > 603 && ps.OverlayValues[603].Loc != scm.LocNone {
			d603 = ps.OverlayValues[603]
		}
		if len(ps.OverlayValues) > 604 && ps.OverlayValues[604].Loc != scm.LocNone {
			d604 = ps.OverlayValues[604]
		}
		if len(ps.OverlayValues) > 605 && ps.OverlayValues[605].Loc != scm.LocNone {
			d605 = ps.OverlayValues[605]
		}
		if len(ps.OverlayValues) > 606 && ps.OverlayValues[606].Loc != scm.LocNone {
			d606 = ps.OverlayValues[606]
		}
		if len(ps.OverlayValues) > 607 && ps.OverlayValues[607].Loc != scm.LocNone {
			d607 = ps.OverlayValues[607]
		}
		if len(ps.OverlayValues) > 608 && ps.OverlayValues[608].Loc != scm.LocNone {
			d608 = ps.OverlayValues[608]
		}
		if len(ps.OverlayValues) > 609 && ps.OverlayValues[609].Loc != scm.LocNone {
			d609 = ps.OverlayValues[609]
		}
		if len(ps.OverlayValues) > 610 && ps.OverlayValues[610].Loc != scm.LocNone {
			d610 = ps.OverlayValues[610]
		}
		if len(ps.OverlayValues) > 611 && ps.OverlayValues[611].Loc != scm.LocNone {
			d611 = ps.OverlayValues[611]
		}
		if len(ps.OverlayValues) > 612 && ps.OverlayValues[612].Loc != scm.LocNone {
			d612 = ps.OverlayValues[612]
		}
		if len(ps.OverlayValues) > 613 && ps.OverlayValues[613].Loc != scm.LocNone {
			d613 = ps.OverlayValues[613]
		}
		if len(ps.OverlayValues) > 614 && ps.OverlayValues[614].Loc != scm.LocNone {
			d614 = ps.OverlayValues[614]
		}
		if len(ps.OverlayValues) > 615 && ps.OverlayValues[615].Loc != scm.LocNone {
			d615 = ps.OverlayValues[615]
		}
		if len(ps.OverlayValues) > 616 && ps.OverlayValues[616].Loc != scm.LocNone {
			d616 = ps.OverlayValues[616]
		}
		if len(ps.OverlayValues) > 617 && ps.OverlayValues[617].Loc != scm.LocNone {
			d617 = ps.OverlayValues[617]
		}
		if len(ps.OverlayValues) > 618 && ps.OverlayValues[618].Loc != scm.LocNone {
			d618 = ps.OverlayValues[618]
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
		if len(ps.OverlayValues) > 624 && ps.OverlayValues[624].Loc != scm.LocNone {
			d624 = ps.OverlayValues[624]
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
		if len(ps.OverlayValues) > 913 && ps.OverlayValues[913].Loc != scm.LocNone {
			d913 = ps.OverlayValues[913]
		}
		if len(ps.OverlayValues) > 914 && ps.OverlayValues[914].Loc != scm.LocNone {
			d914 = ps.OverlayValues[914]
		}
		if len(ps.OverlayValues) > 915 && ps.OverlayValues[915].Loc != scm.LocNone {
			d915 = ps.OverlayValues[915]
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
		if len(ps.OverlayValues) > 925 && ps.OverlayValues[925].Loc != scm.LocNone {
			d925 = ps.OverlayValues[925]
		}
		if len(ps.OverlayValues) > 927 && ps.OverlayValues[927].Loc != scm.LocNone {
			d927 = ps.OverlayValues[927]
		}
		if len(ps.OverlayValues) > 928 && ps.OverlayValues[928].Loc != scm.LocNone {
			d928 = ps.OverlayValues[928]
		}
		if len(ps.OverlayValues) > 1081 && ps.OverlayValues[1081].Loc != scm.LocNone {
			d1081 = ps.OverlayValues[1081]
		}
		if len(ps.OverlayValues) > 1082 && ps.OverlayValues[1082].Loc != scm.LocNone {
			d1082 = ps.OverlayValues[1082]
		}
		if len(ps.OverlayValues) > 1085 && ps.OverlayValues[1085].Loc != scm.LocNone {
			d1085 = ps.OverlayValues[1085]
		}
		if len(ps.OverlayValues) > 1241 && ps.OverlayValues[1241].Loc != scm.LocNone {
			d1241 = ps.OverlayValues[1241]
		}
		if len(ps.OverlayValues) > 1242 && ps.OverlayValues[1242].Loc != scm.LocNone {
			d1242 = ps.OverlayValues[1242]
		}
		if len(ps.OverlayValues) > 1243 && ps.OverlayValues[1243].Loc != scm.LocNone {
			d1243 = ps.OverlayValues[1243]
		}
		if len(ps.OverlayValues) > 1244 && ps.OverlayValues[1244].Loc != scm.LocNone {
			d1244 = ps.OverlayValues[1244]
		}
		if len(ps.OverlayValues) > 1246 && ps.OverlayValues[1246].Loc != scm.LocNone {
			d1246 = ps.OverlayValues[1246]
		}
		if len(ps.OverlayValues) > 1247 && ps.OverlayValues[1247].Loc != scm.LocNone {
			d1247 = ps.OverlayValues[1247]
		}
		if len(ps.OverlayValues) > 1248 && ps.OverlayValues[1248].Loc != scm.LocNone {
			d1248 = ps.OverlayValues[1248]
		}
		if len(ps.OverlayValues) > 1249 && ps.OverlayValues[1249].Loc != scm.LocNone {
			d1249 = ps.OverlayValues[1249]
		}
		if len(ps.OverlayValues) > 1250 && ps.OverlayValues[1250].Loc != scm.LocNone {
			d1250 = ps.OverlayValues[1250]
		}
		if len(ps.OverlayValues) > 1251 && ps.OverlayValues[1251].Loc != scm.LocNone {
			d1251 = ps.OverlayValues[1251]
		}
		if len(ps.OverlayValues) > 1252 && ps.OverlayValues[1252].Loc != scm.LocNone {
			d1252 = ps.OverlayValues[1252]
		}
		if len(ps.OverlayValues) > 1253 && ps.OverlayValues[1253].Loc != scm.LocNone {
			d1253 = ps.OverlayValues[1253]
		}
		if len(ps.OverlayValues) > 1255 && ps.OverlayValues[1255].Loc != scm.LocNone {
			d1255 = ps.OverlayValues[1255]
		}
		if len(ps.OverlayValues) > 1256 && ps.OverlayValues[1256].Loc != scm.LocNone {
			d1256 = ps.OverlayValues[1256]
		}
		ctx.ReclaimUntrackedRegs()
		d1257 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d1258 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d1258)
		ctx.BindReg(r1, &d1258)
		ctx.EnsureDesc(&d1257)
		if d1257.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d1257, &d1258)
		} else {
			switch d1257.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d1258, d1257)
			case scm.TagInt:
				ctx.EmitMakeInt(d1258, d1257)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d1258, d1257)
			case scm.TagNil:
				ctx.EmitMakeNil(d1258)
			default:
				ctx.EmitMovPairToResult(&d1257, &d1258)
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
		if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != scm.LocNone {
			d176 = ps.OverlayValues[176]
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
		if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
			d196 = ps.OverlayValues[196]
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
		if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != scm.LocNone {
			d381 = ps.OverlayValues[381]
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
		if len(ps.OverlayValues) > 486 && ps.OverlayValues[486].Loc != scm.LocNone {
			d486 = ps.OverlayValues[486]
		}
		if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != scm.LocNone {
			d487 = ps.OverlayValues[487]
		}
		if len(ps.OverlayValues) > 490 && ps.OverlayValues[490].Loc != scm.LocNone {
			d490 = ps.OverlayValues[490]
		}
		if len(ps.OverlayValues) > 594 && ps.OverlayValues[594].Loc != scm.LocNone {
			d594 = ps.OverlayValues[594]
		}
		if len(ps.OverlayValues) > 595 && ps.OverlayValues[595].Loc != scm.LocNone {
			d595 = ps.OverlayValues[595]
		}
		if len(ps.OverlayValues) > 596 && ps.OverlayValues[596].Loc != scm.LocNone {
			d596 = ps.OverlayValues[596]
		}
		if len(ps.OverlayValues) > 597 && ps.OverlayValues[597].Loc != scm.LocNone {
			d597 = ps.OverlayValues[597]
		}
		if len(ps.OverlayValues) > 598 && ps.OverlayValues[598].Loc != scm.LocNone {
			d598 = ps.OverlayValues[598]
		}
		if len(ps.OverlayValues) > 600 && ps.OverlayValues[600].Loc != scm.LocNone {
			d600 = ps.OverlayValues[600]
		}
		if len(ps.OverlayValues) > 601 && ps.OverlayValues[601].Loc != scm.LocNone {
			d601 = ps.OverlayValues[601]
		}
		if len(ps.OverlayValues) > 602 && ps.OverlayValues[602].Loc != scm.LocNone {
			d602 = ps.OverlayValues[602]
		}
		if len(ps.OverlayValues) > 603 && ps.OverlayValues[603].Loc != scm.LocNone {
			d603 = ps.OverlayValues[603]
		}
		if len(ps.OverlayValues) > 604 && ps.OverlayValues[604].Loc != scm.LocNone {
			d604 = ps.OverlayValues[604]
		}
		if len(ps.OverlayValues) > 605 && ps.OverlayValues[605].Loc != scm.LocNone {
			d605 = ps.OverlayValues[605]
		}
		if len(ps.OverlayValues) > 606 && ps.OverlayValues[606].Loc != scm.LocNone {
			d606 = ps.OverlayValues[606]
		}
		if len(ps.OverlayValues) > 607 && ps.OverlayValues[607].Loc != scm.LocNone {
			d607 = ps.OverlayValues[607]
		}
		if len(ps.OverlayValues) > 608 && ps.OverlayValues[608].Loc != scm.LocNone {
			d608 = ps.OverlayValues[608]
		}
		if len(ps.OverlayValues) > 609 && ps.OverlayValues[609].Loc != scm.LocNone {
			d609 = ps.OverlayValues[609]
		}
		if len(ps.OverlayValues) > 610 && ps.OverlayValues[610].Loc != scm.LocNone {
			d610 = ps.OverlayValues[610]
		}
		if len(ps.OverlayValues) > 611 && ps.OverlayValues[611].Loc != scm.LocNone {
			d611 = ps.OverlayValues[611]
		}
		if len(ps.OverlayValues) > 612 && ps.OverlayValues[612].Loc != scm.LocNone {
			d612 = ps.OverlayValues[612]
		}
		if len(ps.OverlayValues) > 613 && ps.OverlayValues[613].Loc != scm.LocNone {
			d613 = ps.OverlayValues[613]
		}
		if len(ps.OverlayValues) > 614 && ps.OverlayValues[614].Loc != scm.LocNone {
			d614 = ps.OverlayValues[614]
		}
		if len(ps.OverlayValues) > 615 && ps.OverlayValues[615].Loc != scm.LocNone {
			d615 = ps.OverlayValues[615]
		}
		if len(ps.OverlayValues) > 616 && ps.OverlayValues[616].Loc != scm.LocNone {
			d616 = ps.OverlayValues[616]
		}
		if len(ps.OverlayValues) > 617 && ps.OverlayValues[617].Loc != scm.LocNone {
			d617 = ps.OverlayValues[617]
		}
		if len(ps.OverlayValues) > 618 && ps.OverlayValues[618].Loc != scm.LocNone {
			d618 = ps.OverlayValues[618]
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
		if len(ps.OverlayValues) > 624 && ps.OverlayValues[624].Loc != scm.LocNone {
			d624 = ps.OverlayValues[624]
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
		if len(ps.OverlayValues) > 913 && ps.OverlayValues[913].Loc != scm.LocNone {
			d913 = ps.OverlayValues[913]
		}
		if len(ps.OverlayValues) > 914 && ps.OverlayValues[914].Loc != scm.LocNone {
			d914 = ps.OverlayValues[914]
		}
		if len(ps.OverlayValues) > 915 && ps.OverlayValues[915].Loc != scm.LocNone {
			d915 = ps.OverlayValues[915]
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
		if len(ps.OverlayValues) > 925 && ps.OverlayValues[925].Loc != scm.LocNone {
			d925 = ps.OverlayValues[925]
		}
		if len(ps.OverlayValues) > 927 && ps.OverlayValues[927].Loc != scm.LocNone {
			d927 = ps.OverlayValues[927]
		}
		if len(ps.OverlayValues) > 928 && ps.OverlayValues[928].Loc != scm.LocNone {
			d928 = ps.OverlayValues[928]
		}
		if len(ps.OverlayValues) > 1081 && ps.OverlayValues[1081].Loc != scm.LocNone {
			d1081 = ps.OverlayValues[1081]
		}
		if len(ps.OverlayValues) > 1082 && ps.OverlayValues[1082].Loc != scm.LocNone {
			d1082 = ps.OverlayValues[1082]
		}
		if len(ps.OverlayValues) > 1085 && ps.OverlayValues[1085].Loc != scm.LocNone {
			d1085 = ps.OverlayValues[1085]
		}
		if len(ps.OverlayValues) > 1241 && ps.OverlayValues[1241].Loc != scm.LocNone {
			d1241 = ps.OverlayValues[1241]
		}
		if len(ps.OverlayValues) > 1242 && ps.OverlayValues[1242].Loc != scm.LocNone {
			d1242 = ps.OverlayValues[1242]
		}
		if len(ps.OverlayValues) > 1243 && ps.OverlayValues[1243].Loc != scm.LocNone {
			d1243 = ps.OverlayValues[1243]
		}
		if len(ps.OverlayValues) > 1244 && ps.OverlayValues[1244].Loc != scm.LocNone {
			d1244 = ps.OverlayValues[1244]
		}
		if len(ps.OverlayValues) > 1246 && ps.OverlayValues[1246].Loc != scm.LocNone {
			d1246 = ps.OverlayValues[1246]
		}
		if len(ps.OverlayValues) > 1247 && ps.OverlayValues[1247].Loc != scm.LocNone {
			d1247 = ps.OverlayValues[1247]
		}
		if len(ps.OverlayValues) > 1248 && ps.OverlayValues[1248].Loc != scm.LocNone {
			d1248 = ps.OverlayValues[1248]
		}
		if len(ps.OverlayValues) > 1249 && ps.OverlayValues[1249].Loc != scm.LocNone {
			d1249 = ps.OverlayValues[1249]
		}
		if len(ps.OverlayValues) > 1250 && ps.OverlayValues[1250].Loc != scm.LocNone {
			d1250 = ps.OverlayValues[1250]
		}
		if len(ps.OverlayValues) > 1251 && ps.OverlayValues[1251].Loc != scm.LocNone {
			d1251 = ps.OverlayValues[1251]
		}
		if len(ps.OverlayValues) > 1252 && ps.OverlayValues[1252].Loc != scm.LocNone {
			d1252 = ps.OverlayValues[1252]
		}
		if len(ps.OverlayValues) > 1253 && ps.OverlayValues[1253].Loc != scm.LocNone {
			d1253 = ps.OverlayValues[1253]
		}
		if len(ps.OverlayValues) > 1255 && ps.OverlayValues[1255].Loc != scm.LocNone {
			d1255 = ps.OverlayValues[1255]
		}
		if len(ps.OverlayValues) > 1256 && ps.OverlayValues[1256].Loc != scm.LocNone {
			d1256 = ps.OverlayValues[1256]
		}
		if len(ps.OverlayValues) > 1257 && ps.OverlayValues[1257].Loc != scm.LocNone {
			d1257 = ps.OverlayValues[1257]
		}
		if len(ps.OverlayValues) > 1258 && ps.OverlayValues[1258].Loc != scm.LocNone {
			d1258 = ps.OverlayValues[1258]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d4)
		d1259 = d4
		_ = d1259
		ctx.StabilizeDescForControlFlow(&d1259)
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
		ctx.EnsureDesc(&d1259)
		ctx.EnsureDesc(&d1259)
		var d1260 scm.JITValueDesc
		if d1259.Loc == scm.LocImm {
			d1260 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d1259.Imm.Int()))))}
		} else {
			r108 := ctx.AllocReg()
			ctx.EmitMovRegReg(r108, d1259.Reg)
			ctx.EmitShlRegImm8(r108, 32)
			ctx.EmitShrRegImm8(r108, 32)
			d1260 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
			ctx.BindReg(r108, &d1260)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d1261 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d1261 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 48)
			r109 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r109, thisptr.Reg, off)
			d1261 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r109}
			ctx.BindReg(r109, &d1261)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1261)
		ctx.EnsureDesc(&d1261)
		var d1262 scm.JITValueDesc
		if d1261.Loc == scm.LocImm {
			d1262 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d1261.Imm.Int()))))}
		} else {
			r110 := ctx.AllocReg()
			ctx.EmitMovRegReg(r110, d1261.Reg)
			ctx.EmitShlRegImm8(r110, 56)
			ctx.EmitShrRegImm8(r110, 56)
			d1262 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
			ctx.BindReg(r110, &d1262)
		}
		ctx.FreeDesc(&d1261)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1260)
		ctx.EnsureDesc(&d1262)
		ctx.EnsureDescsTogether(&d1260, &d1262)
		var d1263 scm.JITValueDesc
		if d1260.Loc == scm.LocImm && d1262.Loc == scm.LocImm {
			d1263 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1260.Imm.Int() * d1262.Imm.Int())}
		} else if d1260.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1262.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1260.Imm.Int()))
			ctx.EmitImulInt64(scratch, d1262.Reg)
			d1263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1263)
		} else if d1262.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1260.Reg)
			ctx.EmitMovRegReg(scratch, d1260.Reg)
			if d1262.Imm.Int() >= -2147483648 && d1262.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d1262.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1262.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d1263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1263)
		} else {
			r111 := ctx.AllocRegExcept(d1260.Reg, d1262.Reg)
			ctx.EmitMovRegReg(r111, d1260.Reg)
			ctx.EmitImulInt64(r111, d1262.Reg)
			d1263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r111}
			ctx.BindReg(r111, &d1263)
		}
		if d1263.Loc == scm.LocReg && d1260.Loc == scm.LocReg && d1263.Reg == d1260.Reg {
			ctx.TransferReg(d1260.Reg)
			d1260.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1260)
		ctx.FreeDesc(&d1262)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1263)
		var d1264 scm.JITValueDesc
		if d1263.Loc == scm.LocImm {
			d1264 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1263.Imm.Int() / 64)}
		} else {
			r112 := ctx.AllocRegExcept(d1263.Reg)
			ctx.EmitMovRegReg(r112, d1263.Reg)
			ctx.EmitShrRegImm8(r112, 6)
			d1264 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r112}
			ctx.BindReg(r112, &d1264)
		}
		if d1264.Loc == scm.LocReg && d1263.Loc == scm.LocReg && d1264.Reg == d1263.Reg {
			ctx.TransferReg(d1263.Reg)
			d1263.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1263)
		var d1265 scm.JITValueDesc
		if d1263.Loc == scm.LocImm {
			d1265 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1263.Imm.Int() % 64)}
		} else {
			r113 := ctx.AllocRegExcept(d1263.Reg)
			ctx.EmitMovRegReg(r113, d1263.Reg)
			ctx.EmitAndRegImm32(r113, 63)
			d1265 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r113}
			ctx.BindReg(r113, &d1265)
		}
		if d1265.Loc == scm.LocReg && d1263.Loc == scm.LocReg && d1265.Reg == d1263.Reg {
			ctx.TransferReg(d1263.Reg)
			d1263.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1263)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d1266 scm.JITValueDesc
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
		d1266 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r114, Reg2: r115, Reg3: r116}
		ctx.BindReg(r114, &d1266)
		ctx.BindReg(r115, &d1266)
		ctx.BindReg(r116, &d1266)
		ctx.BindReg(r114, &d1266)
		ctx.BindReg(r115, &d1266)
		ctx.BindReg(r116, &d1266)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1264)
		ctx.ReclaimUntrackedRegs()
		d1268 = ctx.EmitSliceElementAddress(&d1266, &d1264, 8)
		ctx.EnsureDesc(&d1268)
		ctx.EmitMovRegMem(d1268.Reg, d1268.Reg, 0)
		d1267 = d1268
		d1267.Type = scm.TagInt
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1267)
		ctx.EnsureDesc(&d1265)
		ctx.EnsureDescsTogether(&d1267, &d1265)
		var d1269 scm.JITValueDesc
		if d1267.Loc == scm.LocImm && d1265.Loc == scm.LocImm {
			d1269 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1267.Imm.Int()) << uint64(d1265.Imm.Int())))}
		} else if d1265.Loc == scm.LocImm {
			r117 := ctx.AllocRegExcept(d1267.Reg)
			ctx.EmitMovRegReg(r117, d1267.Reg)
			ctx.EmitShlRegImm8(r117, uint8(d1265.Imm.Int()))
			d1269 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
			ctx.BindReg(r117, &d1269)
		} else {
			{
				shiftSrc := d1267.Reg
				r118 := ctx.AllocRegExcept(d1267.Reg, d1265.Reg)
				ctx.EmitMovRegReg(r118, d1267.Reg)
				shiftSrc = r118
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d1265.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d1265.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d1265.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d1269 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d1269)
			}
		}
		if d1269.Loc == scm.LocReg && d1267.Loc == scm.LocReg && d1269.Reg == d1267.Reg {
			ctx.TransferReg(d1267.Reg)
			d1267.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1267)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1264)
		ctx.EnsureDesc(&d1264)
		var d1270 scm.JITValueDesc
		if d1264.Loc == scm.LocImm {
			d1270 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1264.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d1264.Reg)
			ctx.EmitMovRegReg(scratch, d1264.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d1270 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1270)
		}
		if d1270.Loc == scm.LocReg && d1264.Loc == scm.LocReg && d1270.Reg == d1264.Reg {
			ctx.TransferReg(d1264.Reg)
			d1264.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1264)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1270)
		ctx.ReclaimUntrackedRegs()
		d1272 = ctx.EmitSliceElementAddress(&d1266, &d1270, 8)
		ctx.EnsureDesc(&d1272)
		ctx.EmitMovRegMem(d1272.Reg, d1272.Reg, 0)
		d1271 = d1272
		d1271.Type = scm.TagInt
		ctx.FreeDesc(&d1270)
		ctx.ReclaimUntrackedRegs()
		d1273 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d1265)
		ctx.EnsureDescsTogether(&d1273, &d1265)
		var d1274 scm.JITValueDesc
		if d1273.Loc == scm.LocImm && d1265.Loc == scm.LocImm {
			d1274 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1273.Imm.Int() - d1265.Imm.Int())}
		} else if d1265.Loc == scm.LocImm && d1265.Imm.Int() == 0 {
			r119 := ctx.AllocRegExcept(d1273.Reg)
			ctx.EmitMovRegReg(r119, d1273.Reg)
			d1274 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
			ctx.BindReg(r119, &d1274)
		} else if d1273.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1265.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1273.Imm.Int()))
			ctx.EmitSubInt64(scratch, d1265.Reg)
			d1274 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1274)
		} else if d1265.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1273.Reg)
			ctx.EmitMovRegReg(scratch, d1273.Reg)
			if d1265.Imm.Int() >= -2147483648 && d1265.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d1265.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1265.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d1274 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1274)
		} else {
			r120 := ctx.AllocRegExcept(d1273.Reg, d1265.Reg)
			ctx.EmitMovRegReg(r120, d1273.Reg)
			ctx.EmitSubInt64(r120, d1265.Reg)
			d1274 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r120}
			ctx.BindReg(r120, &d1274)
		}
		if d1274.Loc == scm.LocReg && d1273.Loc == scm.LocReg && d1274.Reg == d1273.Reg {
			ctx.TransferReg(d1273.Reg)
			d1273.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1265)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1271)
		ctx.EnsureDesc(&d1274)
		ctx.EnsureDescsTogether(&d1271, &d1274)
		var d1275 scm.JITValueDesc
		if d1271.Loc == scm.LocImm && d1274.Loc == scm.LocImm {
			d1275 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1271.Imm.Int()) >> uint64(d1274.Imm.Int())))}
		} else if d1274.Loc == scm.LocImm {
			r121 := ctx.AllocRegExcept(d1271.Reg)
			ctx.EmitMovRegReg(r121, d1271.Reg)
			ctx.EmitShrRegImm8(r121, uint8(d1274.Imm.Int()))
			d1275 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r121}
			ctx.BindReg(r121, &d1275)
		} else {
			{
				shiftSrc := d1271.Reg
				r122 := ctx.AllocRegExcept(d1271.Reg, d1274.Reg)
				ctx.EmitMovRegReg(r122, d1271.Reg)
				shiftSrc = r122
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d1274.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d1274.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d1274.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d1275 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d1275)
			}
		}
		if d1275.Loc == scm.LocReg && d1271.Loc == scm.LocReg && d1275.Reg == d1271.Reg {
			ctx.TransferReg(d1271.Reg)
			d1271.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1271)
		ctx.FreeDesc(&d1274)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1269)
		ctx.EnsureDesc(&d1275)
		var d1276 scm.JITValueDesc
		if d1269.Loc == scm.LocImm && d1275.Loc == scm.LocImm {
			d1276 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1269.Imm.Int() | d1275.Imm.Int())}
		} else if d1269.Loc == scm.LocImm && d1269.Imm.Int() == 0 {
			d1276 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1275.Reg}
			ctx.BindReg(d1275.Reg, &d1276)
		} else if d1275.Loc == scm.LocImm && d1275.Imm.Int() == 0 {
			r123 := ctx.AllocRegExcept(d1269.Reg)
			ctx.EmitMovRegReg(r123, d1269.Reg)
			d1276 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
			ctx.BindReg(r123, &d1276)
		} else if d1269.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1275.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1269.Imm.Int()))
			ctx.EmitOrInt64(scratch, d1275.Reg)
			d1276 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1276)
		} else if d1275.Loc == scm.LocImm {
			r124 := ctx.AllocRegExcept(d1269.Reg)
			ctx.EmitMovRegReg(r124, d1269.Reg)
			if d1275.Imm.Int() >= -2147483648 && d1275.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r124, int32(d1275.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1275.Imm.Int()))
				ctx.EmitOrInt64(r124, scm.RegR11)
			}
			d1276 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
			ctx.BindReg(r124, &d1276)
		} else {
			r125 := ctx.AllocRegExcept(d1269.Reg, d1275.Reg)
			ctx.EmitMovRegReg(r125, d1269.Reg)
			ctx.EmitOrInt64(r125, d1275.Reg)
			d1276 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
			ctx.BindReg(r125, &d1276)
		}
		if d1276.Loc == scm.LocReg && d1269.Loc == scm.LocReg && d1276.Reg == d1269.Reg {
			ctx.TransferReg(d1269.Reg)
			d1269.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1269)
		ctx.FreeDesc(&d1275)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d1277 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d1277 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 48)
			r126 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r126, thisptr.Reg, off)
			d1277 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r126}
			ctx.BindReg(r126, &d1277)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1277)
		ctx.EnsureDesc(&d1277)
		var d1278 scm.JITValueDesc
		if d1277.Loc == scm.LocImm {
			d1278 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d1277.Imm.Int()))))}
		} else {
			r127 := ctx.AllocReg()
			ctx.EmitMovRegReg(r127, d1277.Reg)
			ctx.EmitShlRegImm8(r127, 56)
			ctx.EmitShrRegImm8(r127, 56)
			d1278 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
			ctx.BindReg(r127, &d1278)
		}
		ctx.FreeDesc(&d1277)
		ctx.ReclaimUntrackedRegs()
		d1279 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d1278)
		ctx.EnsureDescsTogether(&d1279, &d1278)
		var d1280 scm.JITValueDesc
		if d1279.Loc == scm.LocImm && d1278.Loc == scm.LocImm {
			d1280 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1279.Imm.Int() - d1278.Imm.Int())}
		} else if d1278.Loc == scm.LocImm && d1278.Imm.Int() == 0 {
			r128 := ctx.AllocRegExcept(d1279.Reg)
			ctx.EmitMovRegReg(r128, d1279.Reg)
			d1280 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
			ctx.BindReg(r128, &d1280)
		} else if d1279.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1278.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1279.Imm.Int()))
			ctx.EmitSubInt64(scratch, d1278.Reg)
			d1280 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1280)
		} else if d1278.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1279.Reg)
			ctx.EmitMovRegReg(scratch, d1279.Reg)
			if d1278.Imm.Int() >= -2147483648 && d1278.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d1278.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1278.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d1280 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1280)
		} else {
			r129 := ctx.AllocRegExcept(d1279.Reg, d1278.Reg)
			ctx.EmitMovRegReg(r129, d1279.Reg)
			ctx.EmitSubInt64(r129, d1278.Reg)
			d1280 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
			ctx.BindReg(r129, &d1280)
		}
		if d1280.Loc == scm.LocReg && d1279.Loc == scm.LocReg && d1280.Reg == d1279.Reg {
			ctx.TransferReg(d1279.Reg)
			d1279.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1278)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1276)
		ctx.EnsureDesc(&d1280)
		ctx.EnsureDescsTogether(&d1276, &d1280)
		var d1281 scm.JITValueDesc
		if d1276.Loc == scm.LocImm && d1280.Loc == scm.LocImm {
			d1281 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1276.Imm.Int()) >> uint64(d1280.Imm.Int())))}
		} else if d1280.Loc == scm.LocImm {
			r130 := ctx.AllocRegExcept(d1276.Reg)
			ctx.EmitMovRegReg(r130, d1276.Reg)
			ctx.EmitShrRegImm8(r130, uint8(d1280.Imm.Int()))
			d1281 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
			ctx.BindReg(r130, &d1281)
		} else {
			{
				shiftSrc := d1276.Reg
				r131 := ctx.AllocRegExcept(d1276.Reg, d1280.Reg)
				ctx.EmitMovRegReg(r131, d1276.Reg)
				shiftSrc = r131
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d1280.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d1280.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d1280.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d1281 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d1281)
			}
		}
		if d1281.Loc == scm.LocReg && d1276.Loc == scm.LocReg && d1281.Reg == d1276.Reg {
			ctx.TransferReg(d1276.Reg)
			d1276.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1276)
		ctx.FreeDesc(&d1280)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1281)
		ctx.EnsureDesc(&d1281)
		ctx.EnsureDesc(&d1281)
		var d1282 scm.JITValueDesc
		if d1281.Loc == scm.LocImm {
			d1282 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d1281.Imm.Int()))))}
		} else {
			r132 := ctx.AllocReg()
			ctx.EmitMovRegReg(r132, d1281.Reg)
			d1282 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
			ctx.BindReg(r132, &d1282)
		}
		ctx.FreeDesc(&d1281)
		var d1283 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d1283 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 56)
			r133 := ctx.AllocReg()
			ctx.EmitMovRegMem(r133, thisptr.Reg, off)
			d1283 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r133}
			ctx.BindReg(r133, &d1283)
		}
		ctx.EnsureDesc(&d1282)
		ctx.EnsureDesc(&d1283)
		ctx.EnsureDescsTogether(&d1282, &d1283)
		var d1284 scm.JITValueDesc
		if d1282.Loc == scm.LocImm && d1283.Loc == scm.LocImm {
			d1284 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1282.Imm.Int() + d1283.Imm.Int())}
		} else if d1283.Loc == scm.LocImm && d1283.Imm.Int() == 0 {
			r134 := ctx.AllocRegExcept(d1282.Reg)
			ctx.EmitMovRegReg(r134, d1282.Reg)
			d1284 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r134}
			ctx.BindReg(r134, &d1284)
		} else if d1282.Loc == scm.LocImm && d1282.Imm.Int() == 0 {
			d1284 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1283.Reg}
			ctx.BindReg(d1283.Reg, &d1284)
		} else if d1282.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1283.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1282.Imm.Int()))
			ctx.EmitAddInt64(scratch, d1283.Reg)
			d1284 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1284)
		} else if d1283.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1282.Reg)
			ctx.EmitMovRegReg(scratch, d1282.Reg)
			if d1283.Imm.Int() >= -2147483648 && d1283.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d1283.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1283.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d1284 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1284)
		} else {
			r135 := ctx.AllocRegExcept(d1282.Reg, d1283.Reg)
			ctx.EmitMovRegReg(r135, d1282.Reg)
			ctx.EmitAddInt64(r135, d1283.Reg)
			d1284 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r135}
			ctx.BindReg(r135, &d1284)
		}
		if d1284.Loc == scm.LocReg && d1282.Loc == scm.LocReg && d1284.Reg == d1282.Reg {
			ctx.TransferReg(d1282.Reg)
			d1282.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1282)
		ctx.FreeDesc(&d1283)
		ctx.EnsureDesc(&d4)
		d1285 = d4
		_ = d1285
		ctx.StabilizeDescForControlFlow(&d1285)
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
		ctx.EnsureDesc(&d1285)
		ctx.EnsureDesc(&d1285)
		var d1286 scm.JITValueDesc
		if d1285.Loc == scm.LocImm {
			d1286 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d1285.Imm.Int()))))}
		} else {
			r136 := ctx.AllocReg()
			ctx.EmitMovRegReg(r136, d1285.Reg)
			ctx.EmitShlRegImm8(r136, 32)
			ctx.EmitShrRegImm8(r136, 32)
			d1286 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r136}
			ctx.BindReg(r136, &d1286)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d1287 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d1287 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48)
			r137 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r137, thisptr.Reg, off)
			d1287 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r137}
			ctx.BindReg(r137, &d1287)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1287)
		ctx.EnsureDesc(&d1287)
		var d1288 scm.JITValueDesc
		if d1287.Loc == scm.LocImm {
			d1288 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d1287.Imm.Int()))))}
		} else {
			r138 := ctx.AllocReg()
			ctx.EmitMovRegReg(r138, d1287.Reg)
			ctx.EmitShlRegImm8(r138, 56)
			ctx.EmitShrRegImm8(r138, 56)
			d1288 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
			ctx.BindReg(r138, &d1288)
		}
		ctx.FreeDesc(&d1287)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1286)
		ctx.EnsureDesc(&d1288)
		ctx.EnsureDescsTogether(&d1286, &d1288)
		var d1289 scm.JITValueDesc
		if d1286.Loc == scm.LocImm && d1288.Loc == scm.LocImm {
			d1289 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1286.Imm.Int() * d1288.Imm.Int())}
		} else if d1286.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1288.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1286.Imm.Int()))
			ctx.EmitImulInt64(scratch, d1288.Reg)
			d1289 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1289)
		} else if d1288.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1286.Reg)
			ctx.EmitMovRegReg(scratch, d1286.Reg)
			if d1288.Imm.Int() >= -2147483648 && d1288.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d1288.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1288.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d1289 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1289)
		} else {
			r139 := ctx.AllocRegExcept(d1286.Reg, d1288.Reg)
			ctx.EmitMovRegReg(r139, d1286.Reg)
			ctx.EmitImulInt64(r139, d1288.Reg)
			d1289 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r139}
			ctx.BindReg(r139, &d1289)
		}
		if d1289.Loc == scm.LocReg && d1286.Loc == scm.LocReg && d1289.Reg == d1286.Reg {
			ctx.TransferReg(d1286.Reg)
			d1286.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1286)
		ctx.FreeDesc(&d1288)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1289)
		var d1290 scm.JITValueDesc
		if d1289.Loc == scm.LocImm {
			d1290 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1289.Imm.Int() / 64)}
		} else {
			r140 := ctx.AllocRegExcept(d1289.Reg)
			ctx.EmitMovRegReg(r140, d1289.Reg)
			ctx.EmitShrRegImm8(r140, 6)
			d1290 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r140}
			ctx.BindReg(r140, &d1290)
		}
		if d1290.Loc == scm.LocReg && d1289.Loc == scm.LocReg && d1290.Reg == d1289.Reg {
			ctx.TransferReg(d1289.Reg)
			d1289.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1289)
		var d1291 scm.JITValueDesc
		if d1289.Loc == scm.LocImm {
			d1291 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1289.Imm.Int() % 64)}
		} else {
			r141 := ctx.AllocRegExcept(d1289.Reg)
			ctx.EmitMovRegReg(r141, d1289.Reg)
			ctx.EmitAndRegImm32(r141, 63)
			d1291 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
			ctx.BindReg(r141, &d1291)
		}
		if d1291.Loc == scm.LocReg && d1289.Loc == scm.LocReg && d1291.Reg == d1289.Reg {
			ctx.TransferReg(d1289.Reg)
			d1289.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1289)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d1292 scm.JITValueDesc
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
		d1292 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r142, Reg2: r143, Reg3: r144}
		ctx.BindReg(r142, &d1292)
		ctx.BindReg(r143, &d1292)
		ctx.BindReg(r144, &d1292)
		ctx.BindReg(r142, &d1292)
		ctx.BindReg(r143, &d1292)
		ctx.BindReg(r144, &d1292)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1290)
		ctx.ReclaimUntrackedRegs()
		d1294 = ctx.EmitSliceElementAddress(&d1292, &d1290, 8)
		ctx.EnsureDesc(&d1294)
		ctx.EmitMovRegMem(d1294.Reg, d1294.Reg, 0)
		d1293 = d1294
		d1293.Type = scm.TagInt
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1293)
		ctx.EnsureDesc(&d1291)
		ctx.EnsureDescsTogether(&d1293, &d1291)
		var d1295 scm.JITValueDesc
		if d1293.Loc == scm.LocImm && d1291.Loc == scm.LocImm {
			d1295 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1293.Imm.Int()) << uint64(d1291.Imm.Int())))}
		} else if d1291.Loc == scm.LocImm {
			r145 := ctx.AllocRegExcept(d1293.Reg)
			ctx.EmitMovRegReg(r145, d1293.Reg)
			ctx.EmitShlRegImm8(r145, uint8(d1291.Imm.Int()))
			d1295 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r145}
			ctx.BindReg(r145, &d1295)
		} else {
			{
				shiftSrc := d1293.Reg
				r146 := ctx.AllocRegExcept(d1293.Reg, d1291.Reg)
				ctx.EmitMovRegReg(r146, d1293.Reg)
				shiftSrc = r146
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d1291.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d1291.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d1291.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d1295 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d1295)
			}
		}
		if d1295.Loc == scm.LocReg && d1293.Loc == scm.LocReg && d1295.Reg == d1293.Reg {
			ctx.TransferReg(d1293.Reg)
			d1293.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1293)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1290)
		ctx.EnsureDesc(&d1290)
		var d1296 scm.JITValueDesc
		if d1290.Loc == scm.LocImm {
			d1296 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1290.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d1290.Reg)
			ctx.EmitMovRegReg(scratch, d1290.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d1296 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1296)
		}
		if d1296.Loc == scm.LocReg && d1290.Loc == scm.LocReg && d1296.Reg == d1290.Reg {
			ctx.TransferReg(d1290.Reg)
			d1290.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1290)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1296)
		ctx.ReclaimUntrackedRegs()
		d1298 = ctx.EmitSliceElementAddress(&d1292, &d1296, 8)
		ctx.EnsureDesc(&d1298)
		ctx.EmitMovRegMem(d1298.Reg, d1298.Reg, 0)
		d1297 = d1298
		d1297.Type = scm.TagInt
		ctx.FreeDesc(&d1296)
		ctx.ReclaimUntrackedRegs()
		d1299 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d1291)
		ctx.EnsureDescsTogether(&d1299, &d1291)
		var d1300 scm.JITValueDesc
		if d1299.Loc == scm.LocImm && d1291.Loc == scm.LocImm {
			d1300 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1299.Imm.Int() - d1291.Imm.Int())}
		} else if d1291.Loc == scm.LocImm && d1291.Imm.Int() == 0 {
			r147 := ctx.AllocRegExcept(d1299.Reg)
			ctx.EmitMovRegReg(r147, d1299.Reg)
			d1300 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r147}
			ctx.BindReg(r147, &d1300)
		} else if d1299.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1291.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1299.Imm.Int()))
			ctx.EmitSubInt64(scratch, d1291.Reg)
			d1300 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1300)
		} else if d1291.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1299.Reg)
			ctx.EmitMovRegReg(scratch, d1299.Reg)
			if d1291.Imm.Int() >= -2147483648 && d1291.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d1291.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1291.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d1300 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1300)
		} else {
			r148 := ctx.AllocRegExcept(d1299.Reg, d1291.Reg)
			ctx.EmitMovRegReg(r148, d1299.Reg)
			ctx.EmitSubInt64(r148, d1291.Reg)
			d1300 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r148}
			ctx.BindReg(r148, &d1300)
		}
		if d1300.Loc == scm.LocReg && d1299.Loc == scm.LocReg && d1300.Reg == d1299.Reg {
			ctx.TransferReg(d1299.Reg)
			d1299.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1291)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1297)
		ctx.EnsureDesc(&d1300)
		ctx.EnsureDescsTogether(&d1297, &d1300)
		var d1301 scm.JITValueDesc
		if d1297.Loc == scm.LocImm && d1300.Loc == scm.LocImm {
			d1301 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1297.Imm.Int()) >> uint64(d1300.Imm.Int())))}
		} else if d1300.Loc == scm.LocImm {
			r149 := ctx.AllocRegExcept(d1297.Reg)
			ctx.EmitMovRegReg(r149, d1297.Reg)
			ctx.EmitShrRegImm8(r149, uint8(d1300.Imm.Int()))
			d1301 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r149}
			ctx.BindReg(r149, &d1301)
		} else {
			{
				shiftSrc := d1297.Reg
				r150 := ctx.AllocRegExcept(d1297.Reg, d1300.Reg)
				ctx.EmitMovRegReg(r150, d1297.Reg)
				shiftSrc = r150
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d1300.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d1300.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d1300.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d1301 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d1301)
			}
		}
		if d1301.Loc == scm.LocReg && d1297.Loc == scm.LocReg && d1301.Reg == d1297.Reg {
			ctx.TransferReg(d1297.Reg)
			d1297.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1297)
		ctx.FreeDesc(&d1300)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1295)
		ctx.EnsureDesc(&d1301)
		var d1302 scm.JITValueDesc
		if d1295.Loc == scm.LocImm && d1301.Loc == scm.LocImm {
			d1302 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1295.Imm.Int() | d1301.Imm.Int())}
		} else if d1295.Loc == scm.LocImm && d1295.Imm.Int() == 0 {
			d1302 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1301.Reg}
			ctx.BindReg(d1301.Reg, &d1302)
		} else if d1301.Loc == scm.LocImm && d1301.Imm.Int() == 0 {
			r151 := ctx.AllocRegExcept(d1295.Reg)
			ctx.EmitMovRegReg(r151, d1295.Reg)
			d1302 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
			ctx.BindReg(r151, &d1302)
		} else if d1295.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1301.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1295.Imm.Int()))
			ctx.EmitOrInt64(scratch, d1301.Reg)
			d1302 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1302)
		} else if d1301.Loc == scm.LocImm {
			r152 := ctx.AllocRegExcept(d1295.Reg)
			ctx.EmitMovRegReg(r152, d1295.Reg)
			if d1301.Imm.Int() >= -2147483648 && d1301.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r152, int32(d1301.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1301.Imm.Int()))
				ctx.EmitOrInt64(r152, scm.RegR11)
			}
			d1302 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r152}
			ctx.BindReg(r152, &d1302)
		} else {
			r153 := ctx.AllocRegExcept(d1295.Reg, d1301.Reg)
			ctx.EmitMovRegReg(r153, d1295.Reg)
			ctx.EmitOrInt64(r153, d1301.Reg)
			d1302 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r153}
			ctx.BindReg(r153, &d1302)
		}
		if d1302.Loc == scm.LocReg && d1295.Loc == scm.LocReg && d1302.Reg == d1295.Reg {
			ctx.TransferReg(d1295.Reg)
			d1295.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1295)
		ctx.FreeDesc(&d1301)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d1303 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d1303 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48)
			r154 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r154, thisptr.Reg, off)
			d1303 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r154}
			ctx.BindReg(r154, &d1303)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1303)
		ctx.EnsureDesc(&d1303)
		var d1304 scm.JITValueDesc
		if d1303.Loc == scm.LocImm {
			d1304 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d1303.Imm.Int()))))}
		} else {
			r155 := ctx.AllocReg()
			ctx.EmitMovRegReg(r155, d1303.Reg)
			ctx.EmitShlRegImm8(r155, 56)
			ctx.EmitShrRegImm8(r155, 56)
			d1304 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r155}
			ctx.BindReg(r155, &d1304)
		}
		ctx.FreeDesc(&d1303)
		ctx.ReclaimUntrackedRegs()
		d1305 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d1304)
		ctx.EnsureDescsTogether(&d1305, &d1304)
		var d1306 scm.JITValueDesc
		if d1305.Loc == scm.LocImm && d1304.Loc == scm.LocImm {
			d1306 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1305.Imm.Int() - d1304.Imm.Int())}
		} else if d1304.Loc == scm.LocImm && d1304.Imm.Int() == 0 {
			r156 := ctx.AllocRegExcept(d1305.Reg)
			ctx.EmitMovRegReg(r156, d1305.Reg)
			d1306 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r156}
			ctx.BindReg(r156, &d1306)
		} else if d1305.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1304.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1305.Imm.Int()))
			ctx.EmitSubInt64(scratch, d1304.Reg)
			d1306 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1306)
		} else if d1304.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1305.Reg)
			ctx.EmitMovRegReg(scratch, d1305.Reg)
			if d1304.Imm.Int() >= -2147483648 && d1304.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d1304.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1304.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d1306 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1306)
		} else {
			r157 := ctx.AllocRegExcept(d1305.Reg, d1304.Reg)
			ctx.EmitMovRegReg(r157, d1305.Reg)
			ctx.EmitSubInt64(r157, d1304.Reg)
			d1306 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r157}
			ctx.BindReg(r157, &d1306)
		}
		if d1306.Loc == scm.LocReg && d1305.Loc == scm.LocReg && d1306.Reg == d1305.Reg {
			ctx.TransferReg(d1305.Reg)
			d1305.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1304)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1302)
		ctx.EnsureDesc(&d1306)
		ctx.EnsureDescsTogether(&d1302, &d1306)
		var d1307 scm.JITValueDesc
		if d1302.Loc == scm.LocImm && d1306.Loc == scm.LocImm {
			d1307 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1302.Imm.Int()) >> uint64(d1306.Imm.Int())))}
		} else if d1306.Loc == scm.LocImm {
			r158 := ctx.AllocRegExcept(d1302.Reg)
			ctx.EmitMovRegReg(r158, d1302.Reg)
			ctx.EmitShrRegImm8(r158, uint8(d1306.Imm.Int()))
			d1307 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r158}
			ctx.BindReg(r158, &d1307)
		} else {
			{
				shiftSrc := d1302.Reg
				r159 := ctx.AllocRegExcept(d1302.Reg, d1306.Reg)
				ctx.EmitMovRegReg(r159, d1302.Reg)
				shiftSrc = r159
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d1306.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d1306.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d1306.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d1307 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d1307)
			}
		}
		if d1307.Loc == scm.LocReg && d1302.Loc == scm.LocReg && d1307.Reg == d1302.Reg {
			ctx.TransferReg(d1302.Reg)
			d1302.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1302)
		ctx.FreeDesc(&d1306)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1307)
		ctx.EnsureDesc(&d1307)
		ctx.EnsureDesc(&d1307)
		var d1308 scm.JITValueDesc
		if d1307.Loc == scm.LocImm {
			d1308 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d1307.Imm.Int()))))}
		} else {
			r160 := ctx.AllocReg()
			ctx.EmitMovRegReg(r160, d1307.Reg)
			d1308 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r160}
			ctx.BindReg(r160, &d1308)
		}
		ctx.FreeDesc(&d1307)
		var d1309 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d1309 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56)
			r161 := ctx.AllocReg()
			ctx.EmitMovRegMem(r161, thisptr.Reg, off)
			d1309 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r161}
			ctx.BindReg(r161, &d1309)
		}
		ctx.EnsureDesc(&d1308)
		ctx.EnsureDesc(&d1309)
		ctx.EnsureDescsTogether(&d1308, &d1309)
		var d1310 scm.JITValueDesc
		if d1308.Loc == scm.LocImm && d1309.Loc == scm.LocImm {
			d1310 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1308.Imm.Int() + d1309.Imm.Int())}
		} else if d1309.Loc == scm.LocImm && d1309.Imm.Int() == 0 {
			r162 := ctx.AllocRegExcept(d1308.Reg)
			ctx.EmitMovRegReg(r162, d1308.Reg)
			d1310 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r162}
			ctx.BindReg(r162, &d1310)
		} else if d1308.Loc == scm.LocImm && d1308.Imm.Int() == 0 {
			d1310 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1309.Reg}
			ctx.BindReg(d1309.Reg, &d1310)
		} else if d1308.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1309.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1308.Imm.Int()))
			ctx.EmitAddInt64(scratch, d1309.Reg)
			d1310 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1310)
		} else if d1309.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1308.Reg)
			ctx.EmitMovRegReg(scratch, d1308.Reg)
			if d1309.Imm.Int() >= -2147483648 && d1309.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d1309.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1309.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d1310 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1310)
		} else {
			r163 := ctx.AllocRegExcept(d1308.Reg, d1309.Reg)
			ctx.EmitMovRegReg(r163, d1308.Reg)
			ctx.EmitAddInt64(r163, d1309.Reg)
			d1310 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r163}
			ctx.BindReg(r163, &d1310)
		}
		if d1310.Loc == scm.LocReg && d1308.Loc == scm.LocReg && d1310.Reg == d1308.Reg {
			ctx.TransferReg(d1308.Reg)
			d1308.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1308)
		ctx.FreeDesc(&d1309)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		var d1311 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm {
			d1311 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(idxInt.Imm.Int()))))}
		} else {
			r164 := ctx.AllocReg()
			ctx.EmitMovRegReg(r164, idxInt.Reg)
			ctx.EmitShlRegImm8(r164, 32)
			ctx.EmitShrRegImm8(r164, 32)
			d1311 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r164}
			ctx.BindReg(r164, &d1311)
		}
		ctx.FreeDesc(&idxInt)
		ctx.EnsureDesc(&d1311)
		ctx.EnsureDesc(&d1310)
		ctx.EnsureDescsTogether(&d1311, &d1310)
		var d1312 scm.JITValueDesc
		if d1311.Loc == scm.LocImm && d1310.Loc == scm.LocImm {
			d1312 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1311.Imm.Int() - d1310.Imm.Int())}
		} else if d1310.Loc == scm.LocImm && d1310.Imm.Int() == 0 {
			r165 := ctx.AllocRegExcept(d1311.Reg)
			ctx.EmitMovRegReg(r165, d1311.Reg)
			d1312 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r165}
			ctx.BindReg(r165, &d1312)
		} else if d1311.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1310.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1311.Imm.Int()))
			ctx.EmitSubInt64(scratch, d1310.Reg)
			d1312 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1312)
		} else if d1310.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1311.Reg)
			ctx.EmitMovRegReg(scratch, d1311.Reg)
			if d1310.Imm.Int() >= -2147483648 && d1310.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d1310.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1310.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d1312 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1312)
		} else {
			r166 := ctx.AllocRegExcept(d1311.Reg, d1310.Reg)
			ctx.EmitMovRegReg(r166, d1311.Reg)
			ctx.EmitSubInt64(r166, d1310.Reg)
			d1312 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r166}
			ctx.BindReg(r166, &d1312)
		}
		if d1312.Loc == scm.LocReg && d1311.Loc == scm.LocReg && d1312.Reg == d1311.Reg {
			ctx.TransferReg(d1311.Reg)
			d1311.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1311)
		ctx.FreeDesc(&d1310)
		ctx.EnsureDesc(&d1312)
		ctx.EnsureDesc(&d1284)
		ctx.EnsureDescsTogether(&d1312, &d1284)
		var d1313 scm.JITValueDesc
		if d1312.Loc == scm.LocImm && d1284.Loc == scm.LocImm {
			d1313 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1312.Imm.Int() * d1284.Imm.Int())}
		} else if d1312.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1284.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1312.Imm.Int()))
			ctx.EmitImulInt64(scratch, d1284.Reg)
			d1313 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1313)
		} else if d1284.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1312.Reg)
			ctx.EmitMovRegReg(scratch, d1312.Reg)
			if d1284.Imm.Int() >= -2147483648 && d1284.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d1284.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1284.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d1313 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1313)
		} else {
			r167 := ctx.AllocRegExcept(d1312.Reg, d1284.Reg)
			ctx.EmitMovRegReg(r167, d1312.Reg)
			ctx.EmitImulInt64(r167, d1284.Reg)
			d1313 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r167}
			ctx.BindReg(r167, &d1313)
		}
		if d1313.Loc == scm.LocReg && d1312.Loc == scm.LocReg && d1313.Reg == d1312.Reg {
			ctx.TransferReg(d1312.Reg)
			d1312.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1312)
		ctx.FreeDesc(&d1284)
		ctx.EnsureDesc(&d191)
		ctx.EnsureDesc(&d1313)
		ctx.EnsureDescsTogether(&d191, &d1313)
		var d1314 scm.JITValueDesc
		if d191.Loc == scm.LocImm && d1313.Loc == scm.LocImm {
			d1314 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d191.Imm.Int() + d1313.Imm.Int())}
		} else if d1313.Loc == scm.LocImm && d1313.Imm.Int() == 0 {
			r168 := ctx.AllocRegExcept(d191.Reg)
			ctx.EmitMovRegReg(r168, d191.Reg)
			d1314 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r168}
			ctx.BindReg(r168, &d1314)
		} else if d191.Loc == scm.LocImm && d191.Imm.Int() == 0 {
			d1314 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1313.Reg}
			ctx.BindReg(d1313.Reg, &d1314)
		} else if d191.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1313.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d191.Imm.Int()))
			ctx.EmitAddInt64(scratch, d1313.Reg)
			d1314 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1314)
		} else if d1313.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d191.Reg)
			ctx.EmitMovRegReg(scratch, d191.Reg)
			if d1313.Imm.Int() >= -2147483648 && d1313.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d1313.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1313.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d1314 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1314)
		} else {
			r169 := ctx.AllocRegExcept(d191.Reg, d1313.Reg)
			ctx.EmitMovRegReg(r169, d191.Reg)
			ctx.EmitAddInt64(r169, d1313.Reg)
			d1314 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r169}
			ctx.BindReg(r169, &d1314)
		}
		if d1314.Loc == scm.LocReg && d191.Loc == scm.LocReg && d1314.Reg == d191.Reg {
			ctx.TransferReg(d191.Reg)
			d191.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1313)
		ctx.EnsureDesc(&d1314)
		ctx.EnsureDesc(&d1314)
		var d1315 scm.JITValueDesc
		if d1314.Loc == scm.LocImm {
			d1315 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d1314.Imm.Int()))}
		} else {
			r170 := ctx.AllocRegExcept(d1314.Reg)
			ctx.EmitMovRegReg(r170, d1314.Reg)
			ctx.EmitCvtInt64ToFloat64(scm.RegX0, r170)
			d1315 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: r170}
			ctx.BindReg(r170, &d1315)
		}
		ctx.FreeDesc(&d1314)
		ctx.EnsureDesc(&d1315)
		d1316 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d1316)
		ctx.BindReg(r1, &d1316)
		ctx.EnsureDesc(&d1315)
		ctx.EmitMakeFloat(d1316, d1315)
		if d1315.Loc == scm.LocReg {
			ctx.FreeReg(d1315.Reg)
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
		if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != scm.LocNone {
			d176 = ps.OverlayValues[176]
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
		if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != scm.LocNone {
			d196 = ps.OverlayValues[196]
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
		if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != scm.LocNone {
			d381 = ps.OverlayValues[381]
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
		if len(ps.OverlayValues) > 486 && ps.OverlayValues[486].Loc != scm.LocNone {
			d486 = ps.OverlayValues[486]
		}
		if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != scm.LocNone {
			d487 = ps.OverlayValues[487]
		}
		if len(ps.OverlayValues) > 490 && ps.OverlayValues[490].Loc != scm.LocNone {
			d490 = ps.OverlayValues[490]
		}
		if len(ps.OverlayValues) > 594 && ps.OverlayValues[594].Loc != scm.LocNone {
			d594 = ps.OverlayValues[594]
		}
		if len(ps.OverlayValues) > 595 && ps.OverlayValues[595].Loc != scm.LocNone {
			d595 = ps.OverlayValues[595]
		}
		if len(ps.OverlayValues) > 596 && ps.OverlayValues[596].Loc != scm.LocNone {
			d596 = ps.OverlayValues[596]
		}
		if len(ps.OverlayValues) > 597 && ps.OverlayValues[597].Loc != scm.LocNone {
			d597 = ps.OverlayValues[597]
		}
		if len(ps.OverlayValues) > 598 && ps.OverlayValues[598].Loc != scm.LocNone {
			d598 = ps.OverlayValues[598]
		}
		if len(ps.OverlayValues) > 600 && ps.OverlayValues[600].Loc != scm.LocNone {
			d600 = ps.OverlayValues[600]
		}
		if len(ps.OverlayValues) > 601 && ps.OverlayValues[601].Loc != scm.LocNone {
			d601 = ps.OverlayValues[601]
		}
		if len(ps.OverlayValues) > 602 && ps.OverlayValues[602].Loc != scm.LocNone {
			d602 = ps.OverlayValues[602]
		}
		if len(ps.OverlayValues) > 603 && ps.OverlayValues[603].Loc != scm.LocNone {
			d603 = ps.OverlayValues[603]
		}
		if len(ps.OverlayValues) > 604 && ps.OverlayValues[604].Loc != scm.LocNone {
			d604 = ps.OverlayValues[604]
		}
		if len(ps.OverlayValues) > 605 && ps.OverlayValues[605].Loc != scm.LocNone {
			d605 = ps.OverlayValues[605]
		}
		if len(ps.OverlayValues) > 606 && ps.OverlayValues[606].Loc != scm.LocNone {
			d606 = ps.OverlayValues[606]
		}
		if len(ps.OverlayValues) > 607 && ps.OverlayValues[607].Loc != scm.LocNone {
			d607 = ps.OverlayValues[607]
		}
		if len(ps.OverlayValues) > 608 && ps.OverlayValues[608].Loc != scm.LocNone {
			d608 = ps.OverlayValues[608]
		}
		if len(ps.OverlayValues) > 609 && ps.OverlayValues[609].Loc != scm.LocNone {
			d609 = ps.OverlayValues[609]
		}
		if len(ps.OverlayValues) > 610 && ps.OverlayValues[610].Loc != scm.LocNone {
			d610 = ps.OverlayValues[610]
		}
		if len(ps.OverlayValues) > 611 && ps.OverlayValues[611].Loc != scm.LocNone {
			d611 = ps.OverlayValues[611]
		}
		if len(ps.OverlayValues) > 612 && ps.OverlayValues[612].Loc != scm.LocNone {
			d612 = ps.OverlayValues[612]
		}
		if len(ps.OverlayValues) > 613 && ps.OverlayValues[613].Loc != scm.LocNone {
			d613 = ps.OverlayValues[613]
		}
		if len(ps.OverlayValues) > 614 && ps.OverlayValues[614].Loc != scm.LocNone {
			d614 = ps.OverlayValues[614]
		}
		if len(ps.OverlayValues) > 615 && ps.OverlayValues[615].Loc != scm.LocNone {
			d615 = ps.OverlayValues[615]
		}
		if len(ps.OverlayValues) > 616 && ps.OverlayValues[616].Loc != scm.LocNone {
			d616 = ps.OverlayValues[616]
		}
		if len(ps.OverlayValues) > 617 && ps.OverlayValues[617].Loc != scm.LocNone {
			d617 = ps.OverlayValues[617]
		}
		if len(ps.OverlayValues) > 618 && ps.OverlayValues[618].Loc != scm.LocNone {
			d618 = ps.OverlayValues[618]
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
		if len(ps.OverlayValues) > 624 && ps.OverlayValues[624].Loc != scm.LocNone {
			d624 = ps.OverlayValues[624]
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
		if len(ps.OverlayValues) > 913 && ps.OverlayValues[913].Loc != scm.LocNone {
			d913 = ps.OverlayValues[913]
		}
		if len(ps.OverlayValues) > 914 && ps.OverlayValues[914].Loc != scm.LocNone {
			d914 = ps.OverlayValues[914]
		}
		if len(ps.OverlayValues) > 915 && ps.OverlayValues[915].Loc != scm.LocNone {
			d915 = ps.OverlayValues[915]
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
		if len(ps.OverlayValues) > 925 && ps.OverlayValues[925].Loc != scm.LocNone {
			d925 = ps.OverlayValues[925]
		}
		if len(ps.OverlayValues) > 927 && ps.OverlayValues[927].Loc != scm.LocNone {
			d927 = ps.OverlayValues[927]
		}
		if len(ps.OverlayValues) > 928 && ps.OverlayValues[928].Loc != scm.LocNone {
			d928 = ps.OverlayValues[928]
		}
		if len(ps.OverlayValues) > 1081 && ps.OverlayValues[1081].Loc != scm.LocNone {
			d1081 = ps.OverlayValues[1081]
		}
		if len(ps.OverlayValues) > 1082 && ps.OverlayValues[1082].Loc != scm.LocNone {
			d1082 = ps.OverlayValues[1082]
		}
		if len(ps.OverlayValues) > 1085 && ps.OverlayValues[1085].Loc != scm.LocNone {
			d1085 = ps.OverlayValues[1085]
		}
		if len(ps.OverlayValues) > 1241 && ps.OverlayValues[1241].Loc != scm.LocNone {
			d1241 = ps.OverlayValues[1241]
		}
		if len(ps.OverlayValues) > 1242 && ps.OverlayValues[1242].Loc != scm.LocNone {
			d1242 = ps.OverlayValues[1242]
		}
		if len(ps.OverlayValues) > 1243 && ps.OverlayValues[1243].Loc != scm.LocNone {
			d1243 = ps.OverlayValues[1243]
		}
		if len(ps.OverlayValues) > 1244 && ps.OverlayValues[1244].Loc != scm.LocNone {
			d1244 = ps.OverlayValues[1244]
		}
		if len(ps.OverlayValues) > 1246 && ps.OverlayValues[1246].Loc != scm.LocNone {
			d1246 = ps.OverlayValues[1246]
		}
		if len(ps.OverlayValues) > 1247 && ps.OverlayValues[1247].Loc != scm.LocNone {
			d1247 = ps.OverlayValues[1247]
		}
		if len(ps.OverlayValues) > 1248 && ps.OverlayValues[1248].Loc != scm.LocNone {
			d1248 = ps.OverlayValues[1248]
		}
		if len(ps.OverlayValues) > 1249 && ps.OverlayValues[1249].Loc != scm.LocNone {
			d1249 = ps.OverlayValues[1249]
		}
		if len(ps.OverlayValues) > 1250 && ps.OverlayValues[1250].Loc != scm.LocNone {
			d1250 = ps.OverlayValues[1250]
		}
		if len(ps.OverlayValues) > 1251 && ps.OverlayValues[1251].Loc != scm.LocNone {
			d1251 = ps.OverlayValues[1251]
		}
		if len(ps.OverlayValues) > 1252 && ps.OverlayValues[1252].Loc != scm.LocNone {
			d1252 = ps.OverlayValues[1252]
		}
		if len(ps.OverlayValues) > 1253 && ps.OverlayValues[1253].Loc != scm.LocNone {
			d1253 = ps.OverlayValues[1253]
		}
		if len(ps.OverlayValues) > 1255 && ps.OverlayValues[1255].Loc != scm.LocNone {
			d1255 = ps.OverlayValues[1255]
		}
		if len(ps.OverlayValues) > 1256 && ps.OverlayValues[1256].Loc != scm.LocNone {
			d1256 = ps.OverlayValues[1256]
		}
		if len(ps.OverlayValues) > 1257 && ps.OverlayValues[1257].Loc != scm.LocNone {
			d1257 = ps.OverlayValues[1257]
		}
		if len(ps.OverlayValues) > 1258 && ps.OverlayValues[1258].Loc != scm.LocNone {
			d1258 = ps.OverlayValues[1258]
		}
		if len(ps.OverlayValues) > 1259 && ps.OverlayValues[1259].Loc != scm.LocNone {
			d1259 = ps.OverlayValues[1259]
		}
		if len(ps.OverlayValues) > 1260 && ps.OverlayValues[1260].Loc != scm.LocNone {
			d1260 = ps.OverlayValues[1260]
		}
		if len(ps.OverlayValues) > 1261 && ps.OverlayValues[1261].Loc != scm.LocNone {
			d1261 = ps.OverlayValues[1261]
		}
		if len(ps.OverlayValues) > 1262 && ps.OverlayValues[1262].Loc != scm.LocNone {
			d1262 = ps.OverlayValues[1262]
		}
		if len(ps.OverlayValues) > 1263 && ps.OverlayValues[1263].Loc != scm.LocNone {
			d1263 = ps.OverlayValues[1263]
		}
		if len(ps.OverlayValues) > 1264 && ps.OverlayValues[1264].Loc != scm.LocNone {
			d1264 = ps.OverlayValues[1264]
		}
		if len(ps.OverlayValues) > 1265 && ps.OverlayValues[1265].Loc != scm.LocNone {
			d1265 = ps.OverlayValues[1265]
		}
		if len(ps.OverlayValues) > 1266 && ps.OverlayValues[1266].Loc != scm.LocNone {
			d1266 = ps.OverlayValues[1266]
		}
		if len(ps.OverlayValues) > 1267 && ps.OverlayValues[1267].Loc != scm.LocNone {
			d1267 = ps.OverlayValues[1267]
		}
		if len(ps.OverlayValues) > 1268 && ps.OverlayValues[1268].Loc != scm.LocNone {
			d1268 = ps.OverlayValues[1268]
		}
		if len(ps.OverlayValues) > 1269 && ps.OverlayValues[1269].Loc != scm.LocNone {
			d1269 = ps.OverlayValues[1269]
		}
		if len(ps.OverlayValues) > 1270 && ps.OverlayValues[1270].Loc != scm.LocNone {
			d1270 = ps.OverlayValues[1270]
		}
		if len(ps.OverlayValues) > 1271 && ps.OverlayValues[1271].Loc != scm.LocNone {
			d1271 = ps.OverlayValues[1271]
		}
		if len(ps.OverlayValues) > 1272 && ps.OverlayValues[1272].Loc != scm.LocNone {
			d1272 = ps.OverlayValues[1272]
		}
		if len(ps.OverlayValues) > 1273 && ps.OverlayValues[1273].Loc != scm.LocNone {
			d1273 = ps.OverlayValues[1273]
		}
		if len(ps.OverlayValues) > 1274 && ps.OverlayValues[1274].Loc != scm.LocNone {
			d1274 = ps.OverlayValues[1274]
		}
		if len(ps.OverlayValues) > 1275 && ps.OverlayValues[1275].Loc != scm.LocNone {
			d1275 = ps.OverlayValues[1275]
		}
		if len(ps.OverlayValues) > 1276 && ps.OverlayValues[1276].Loc != scm.LocNone {
			d1276 = ps.OverlayValues[1276]
		}
		if len(ps.OverlayValues) > 1277 && ps.OverlayValues[1277].Loc != scm.LocNone {
			d1277 = ps.OverlayValues[1277]
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
		if len(ps.OverlayValues) > 1282 && ps.OverlayValues[1282].Loc != scm.LocNone {
			d1282 = ps.OverlayValues[1282]
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
		if len(ps.OverlayValues) > 1293 && ps.OverlayValues[1293].Loc != scm.LocNone {
			d1293 = ps.OverlayValues[1293]
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
		ctx.ReclaimUntrackedRegs()
		var d1317 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 88
			val := *(*uint64)(unsafe.Pointer(fieldAddr))
			d1317 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 88)
			r171 := ctx.AllocReg()
			ctx.EmitMovRegMem(r171, thisptr.Reg, off)
			d1317 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r171}
			ctx.BindReg(r171, &d1317)
		}
		ctx.EnsureDesc(&d1317)
		ctx.EnsureDesc(&d1317)
		var d1318 scm.JITValueDesc
		if d1317.Loc == scm.LocImm {
			d1318 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d1317.Imm.Int()))))}
		} else {
			r172 := ctx.AllocReg()
			ctx.EmitMovRegReg(r172, d1317.Reg)
			d1318 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
			ctx.BindReg(r172, &d1318)
		}
		ctx.FreeDesc(&d1317)
		ctx.EnsureDesc(&d191)
		ctx.EnsureDesc(&d1318)
		ctx.EnsureDescsTogether(&d191, &d1318)
		var d1319 scm.JITValueDesc
		if d191.Loc == scm.LocImm && d1318.Loc == scm.LocImm {
			d1319 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d191.Imm.Int() == d1318.Imm.Int())}
		} else if d1318.Loc == scm.LocImm {
			r173 := ctx.AllocRegExcept(d191.Reg)
			if d1318.Imm.Int() >= -2147483648 && d1318.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d191.Reg, int32(d1318.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1318.Imm.Int()))
				ctx.EmitCmpInt64(d191.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r173, scm.CondEqual)
			d1319 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r173}
			ctx.BindReg(r173, &d1319)
		} else if d191.Loc == scm.LocImm {
			r174 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d191.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d1318.Reg)
			ctx.EmitSetcc(r174, scm.CondEqual)
			d1319 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r174}
			ctx.BindReg(r174, &d1319)
		} else {
			r175 := ctx.AllocRegExcept(d191.Reg)
			ctx.EmitCmpInt64(d191.Reg, d1318.Reg)
			ctx.EmitSetcc(r175, scm.CondEqual)
			d1319 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r175}
			ctx.BindReg(r175, &d1319)
		}
		ctx.FreeDesc(&d1318)
		d1320 = d1319
		ctx.EnsureDesc(&d1320)
		if d1320.Loc != scm.LocImm && d1320.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d1320.Loc == scm.LocImm {
			if d1320.Imm.Bool() {
				if ps.General {
				}
				ps1321 := scm.PhiState{General: ps.General}
				ps1321.OverlayValues = make([]scm.JITValueDesc, 1321)
				ps1321.OverlayValues[1] = d1
				ps1321.OverlayValues[2] = d2
				ps1321.OverlayValues[3] = d3
				ps1321.OverlayValues[4] = d4
				ps1321.OverlayValues[5] = d5
				ps1321.OverlayValues[6] = d6
				ps1321.OverlayValues[7] = d7
				ps1321.OverlayValues[8] = d8
				ps1321.OverlayValues[9] = d9
				ps1321.OverlayValues[10] = d10
				ps1321.OverlayValues[11] = d11
				ps1321.OverlayValues[12] = d12
				ps1321.OverlayValues[13] = d13
				ps1321.OverlayValues[14] = d14
				ps1321.OverlayValues[15] = d15
				ps1321.OverlayValues[17] = d17
				ps1321.OverlayValues[18] = d18
				ps1321.OverlayValues[19] = d19
				ps1321.OverlayValues[20] = d20
				ps1321.OverlayValues[21] = d21
				ps1321.OverlayValues[22] = d22
				ps1321.OverlayValues[23] = d23
				ps1321.OverlayValues[24] = d24
				ps1321.OverlayValues[25] = d25
				ps1321.OverlayValues[26] = d26
				ps1321.OverlayValues[27] = d27
				ps1321.OverlayValues[28] = d28
				ps1321.OverlayValues[29] = d29
				ps1321.OverlayValues[30] = d30
				ps1321.OverlayValues[31] = d31
				ps1321.OverlayValues[32] = d32
				ps1321.OverlayValues[33] = d33
				ps1321.OverlayValues[34] = d34
				ps1321.OverlayValues[35] = d35
				ps1321.OverlayValues[36] = d36
				ps1321.OverlayValues[37] = d37
				ps1321.OverlayValues[38] = d38
				ps1321.OverlayValues[39] = d39
				ps1321.OverlayValues[40] = d40
				ps1321.OverlayValues[41] = d41
				ps1321.OverlayValues[42] = d42
				ps1321.OverlayValues[43] = d43
				ps1321.OverlayValues[44] = d44
				ps1321.OverlayValues[45] = d45
				ps1321.OverlayValues[46] = d46
				ps1321.OverlayValues[47] = d47
				ps1321.OverlayValues[48] = d48
				ps1321.OverlayValues[49] = d49
				ps1321.OverlayValues[50] = d50
				ps1321.OverlayValues[53] = d53
				ps1321.OverlayValues[54] = d54
				ps1321.OverlayValues[55] = d55
				ps1321.OverlayValues[164] = d164
				ps1321.OverlayValues[165] = d165
				ps1321.OverlayValues[166] = d166
				ps1321.OverlayValues[167] = d167
				ps1321.OverlayValues[168] = d168
				ps1321.OverlayValues[169] = d169
				ps1321.OverlayValues[170] = d170
				ps1321.OverlayValues[171] = d171
				ps1321.OverlayValues[172] = d172
				ps1321.OverlayValues[173] = d173
				ps1321.OverlayValues[174] = d174
				ps1321.OverlayValues[175] = d175
				ps1321.OverlayValues[176] = d176
				ps1321.OverlayValues[177] = d177
				ps1321.OverlayValues[178] = d178
				ps1321.OverlayValues[179] = d179
				ps1321.OverlayValues[180] = d180
				ps1321.OverlayValues[181] = d181
				ps1321.OverlayValues[182] = d182
				ps1321.OverlayValues[183] = d183
				ps1321.OverlayValues[184] = d184
				ps1321.OverlayValues[185] = d185
				ps1321.OverlayValues[186] = d186
				ps1321.OverlayValues[187] = d187
				ps1321.OverlayValues[188] = d188
				ps1321.OverlayValues[189] = d189
				ps1321.OverlayValues[190] = d190
				ps1321.OverlayValues[191] = d191
				ps1321.OverlayValues[192] = d192
				ps1321.OverlayValues[193] = d193
				ps1321.OverlayValues[196] = d196
				ps1321.OverlayValues[367] = d367
				ps1321.OverlayValues[368] = d368
				ps1321.OverlayValues[369] = d369
				ps1321.OverlayValues[370] = d370
				ps1321.OverlayValues[372] = d372
				ps1321.OverlayValues[373] = d373
				ps1321.OverlayValues[374] = d374
				ps1321.OverlayValues[375] = d375
				ps1321.OverlayValues[376] = d376
				ps1321.OverlayValues[377] = d377
				ps1321.OverlayValues[378] = d378
				ps1321.OverlayValues[379] = d379
				ps1321.OverlayValues[381] = d381
				ps1321.OverlayValues[383] = d383
				ps1321.OverlayValues[384] = d384
				ps1321.OverlayValues[385] = d385
				ps1321.OverlayValues[486] = d486
				ps1321.OverlayValues[487] = d487
				ps1321.OverlayValues[490] = d490
				ps1321.OverlayValues[594] = d594
				ps1321.OverlayValues[595] = d595
				ps1321.OverlayValues[596] = d596
				ps1321.OverlayValues[597] = d597
				ps1321.OverlayValues[598] = d598
				ps1321.OverlayValues[600] = d600
				ps1321.OverlayValues[601] = d601
				ps1321.OverlayValues[602] = d602
				ps1321.OverlayValues[603] = d603
				ps1321.OverlayValues[604] = d604
				ps1321.OverlayValues[605] = d605
				ps1321.OverlayValues[606] = d606
				ps1321.OverlayValues[607] = d607
				ps1321.OverlayValues[608] = d608
				ps1321.OverlayValues[609] = d609
				ps1321.OverlayValues[610] = d610
				ps1321.OverlayValues[611] = d611
				ps1321.OverlayValues[612] = d612
				ps1321.OverlayValues[613] = d613
				ps1321.OverlayValues[614] = d614
				ps1321.OverlayValues[615] = d615
				ps1321.OverlayValues[616] = d616
				ps1321.OverlayValues[617] = d617
				ps1321.OverlayValues[618] = d618
				ps1321.OverlayValues[619] = d619
				ps1321.OverlayValues[620] = d620
				ps1321.OverlayValues[621] = d621
				ps1321.OverlayValues[622] = d622
				ps1321.OverlayValues[623] = d623
				ps1321.OverlayValues[624] = d624
				ps1321.OverlayValues[625] = d625
				ps1321.OverlayValues[626] = d626
				ps1321.OverlayValues[627] = d627
				ps1321.OverlayValues[628] = d628
				ps1321.OverlayValues[629] = d629
				ps1321.OverlayValues[630] = d630
				ps1321.OverlayValues[913] = d913
				ps1321.OverlayValues[914] = d914
				ps1321.OverlayValues[915] = d915
				ps1321.OverlayValues[917] = d917
				ps1321.OverlayValues[918] = d918
				ps1321.OverlayValues[919] = d919
				ps1321.OverlayValues[920] = d920
				ps1321.OverlayValues[921] = d921
				ps1321.OverlayValues[922] = d922
				ps1321.OverlayValues[923] = d923
				ps1321.OverlayValues[925] = d925
				ps1321.OverlayValues[927] = d927
				ps1321.OverlayValues[928] = d928
				ps1321.OverlayValues[1081] = d1081
				ps1321.OverlayValues[1082] = d1082
				ps1321.OverlayValues[1085] = d1085
				ps1321.OverlayValues[1241] = d1241
				ps1321.OverlayValues[1242] = d1242
				ps1321.OverlayValues[1243] = d1243
				ps1321.OverlayValues[1244] = d1244
				ps1321.OverlayValues[1246] = d1246
				ps1321.OverlayValues[1247] = d1247
				ps1321.OverlayValues[1248] = d1248
				ps1321.OverlayValues[1249] = d1249
				ps1321.OverlayValues[1250] = d1250
				ps1321.OverlayValues[1251] = d1251
				ps1321.OverlayValues[1252] = d1252
				ps1321.OverlayValues[1253] = d1253
				ps1321.OverlayValues[1255] = d1255
				ps1321.OverlayValues[1256] = d1256
				ps1321.OverlayValues[1257] = d1257
				ps1321.OverlayValues[1258] = d1258
				ps1321.OverlayValues[1259] = d1259
				ps1321.OverlayValues[1260] = d1260
				ps1321.OverlayValues[1261] = d1261
				ps1321.OverlayValues[1262] = d1262
				ps1321.OverlayValues[1263] = d1263
				ps1321.OverlayValues[1264] = d1264
				ps1321.OverlayValues[1265] = d1265
				ps1321.OverlayValues[1266] = d1266
				ps1321.OverlayValues[1267] = d1267
				ps1321.OverlayValues[1268] = d1268
				ps1321.OverlayValues[1269] = d1269
				ps1321.OverlayValues[1270] = d1270
				ps1321.OverlayValues[1271] = d1271
				ps1321.OverlayValues[1272] = d1272
				ps1321.OverlayValues[1273] = d1273
				ps1321.OverlayValues[1274] = d1274
				ps1321.OverlayValues[1275] = d1275
				ps1321.OverlayValues[1276] = d1276
				ps1321.OverlayValues[1277] = d1277
				ps1321.OverlayValues[1278] = d1278
				ps1321.OverlayValues[1279] = d1279
				ps1321.OverlayValues[1280] = d1280
				ps1321.OverlayValues[1281] = d1281
				ps1321.OverlayValues[1282] = d1282
				ps1321.OverlayValues[1283] = d1283
				ps1321.OverlayValues[1284] = d1284
				ps1321.OverlayValues[1285] = d1285
				ps1321.OverlayValues[1286] = d1286
				ps1321.OverlayValues[1287] = d1287
				ps1321.OverlayValues[1288] = d1288
				ps1321.OverlayValues[1289] = d1289
				ps1321.OverlayValues[1290] = d1290
				ps1321.OverlayValues[1291] = d1291
				ps1321.OverlayValues[1292] = d1292
				ps1321.OverlayValues[1293] = d1293
				ps1321.OverlayValues[1294] = d1294
				ps1321.OverlayValues[1295] = d1295
				ps1321.OverlayValues[1296] = d1296
				ps1321.OverlayValues[1297] = d1297
				ps1321.OverlayValues[1298] = d1298
				ps1321.OverlayValues[1299] = d1299
				ps1321.OverlayValues[1300] = d1300
				ps1321.OverlayValues[1301] = d1301
				ps1321.OverlayValues[1302] = d1302
				ps1321.OverlayValues[1303] = d1303
				ps1321.OverlayValues[1304] = d1304
				ps1321.OverlayValues[1305] = d1305
				ps1321.OverlayValues[1306] = d1306
				ps1321.OverlayValues[1307] = d1307
				ps1321.OverlayValues[1308] = d1308
				ps1321.OverlayValues[1309] = d1309
				ps1321.OverlayValues[1310] = d1310
				ps1321.OverlayValues[1311] = d1311
				ps1321.OverlayValues[1312] = d1312
				ps1321.OverlayValues[1313] = d1313
				ps1321.OverlayValues[1314] = d1314
				ps1321.OverlayValues[1315] = d1315
				ps1321.OverlayValues[1316] = d1316
				ps1321.OverlayValues[1317] = d1317
				ps1321.OverlayValues[1318] = d1318
				ps1321.OverlayValues[1319] = d1319
				ps1321.OverlayValues[1320] = d1320
				return bbs[11].RenderPS(ps1321)
			}
			if ps.General {
			}
			ps1322 := scm.PhiState{General: ps.General}
			ps1322.OverlayValues = make([]scm.JITValueDesc, 1321)
			ps1322.OverlayValues[1] = d1
			ps1322.OverlayValues[2] = d2
			ps1322.OverlayValues[3] = d3
			ps1322.OverlayValues[4] = d4
			ps1322.OverlayValues[5] = d5
			ps1322.OverlayValues[6] = d6
			ps1322.OverlayValues[7] = d7
			ps1322.OverlayValues[8] = d8
			ps1322.OverlayValues[9] = d9
			ps1322.OverlayValues[10] = d10
			ps1322.OverlayValues[11] = d11
			ps1322.OverlayValues[12] = d12
			ps1322.OverlayValues[13] = d13
			ps1322.OverlayValues[14] = d14
			ps1322.OverlayValues[15] = d15
			ps1322.OverlayValues[17] = d17
			ps1322.OverlayValues[18] = d18
			ps1322.OverlayValues[19] = d19
			ps1322.OverlayValues[20] = d20
			ps1322.OverlayValues[21] = d21
			ps1322.OverlayValues[22] = d22
			ps1322.OverlayValues[23] = d23
			ps1322.OverlayValues[24] = d24
			ps1322.OverlayValues[25] = d25
			ps1322.OverlayValues[26] = d26
			ps1322.OverlayValues[27] = d27
			ps1322.OverlayValues[28] = d28
			ps1322.OverlayValues[29] = d29
			ps1322.OverlayValues[30] = d30
			ps1322.OverlayValues[31] = d31
			ps1322.OverlayValues[32] = d32
			ps1322.OverlayValues[33] = d33
			ps1322.OverlayValues[34] = d34
			ps1322.OverlayValues[35] = d35
			ps1322.OverlayValues[36] = d36
			ps1322.OverlayValues[37] = d37
			ps1322.OverlayValues[38] = d38
			ps1322.OverlayValues[39] = d39
			ps1322.OverlayValues[40] = d40
			ps1322.OverlayValues[41] = d41
			ps1322.OverlayValues[42] = d42
			ps1322.OverlayValues[43] = d43
			ps1322.OverlayValues[44] = d44
			ps1322.OverlayValues[45] = d45
			ps1322.OverlayValues[46] = d46
			ps1322.OverlayValues[47] = d47
			ps1322.OverlayValues[48] = d48
			ps1322.OverlayValues[49] = d49
			ps1322.OverlayValues[50] = d50
			ps1322.OverlayValues[53] = d53
			ps1322.OverlayValues[54] = d54
			ps1322.OverlayValues[55] = d55
			ps1322.OverlayValues[164] = d164
			ps1322.OverlayValues[165] = d165
			ps1322.OverlayValues[166] = d166
			ps1322.OverlayValues[167] = d167
			ps1322.OverlayValues[168] = d168
			ps1322.OverlayValues[169] = d169
			ps1322.OverlayValues[170] = d170
			ps1322.OverlayValues[171] = d171
			ps1322.OverlayValues[172] = d172
			ps1322.OverlayValues[173] = d173
			ps1322.OverlayValues[174] = d174
			ps1322.OverlayValues[175] = d175
			ps1322.OverlayValues[176] = d176
			ps1322.OverlayValues[177] = d177
			ps1322.OverlayValues[178] = d178
			ps1322.OverlayValues[179] = d179
			ps1322.OverlayValues[180] = d180
			ps1322.OverlayValues[181] = d181
			ps1322.OverlayValues[182] = d182
			ps1322.OverlayValues[183] = d183
			ps1322.OverlayValues[184] = d184
			ps1322.OverlayValues[185] = d185
			ps1322.OverlayValues[186] = d186
			ps1322.OverlayValues[187] = d187
			ps1322.OverlayValues[188] = d188
			ps1322.OverlayValues[189] = d189
			ps1322.OverlayValues[190] = d190
			ps1322.OverlayValues[191] = d191
			ps1322.OverlayValues[192] = d192
			ps1322.OverlayValues[193] = d193
			ps1322.OverlayValues[196] = d196
			ps1322.OverlayValues[367] = d367
			ps1322.OverlayValues[368] = d368
			ps1322.OverlayValues[369] = d369
			ps1322.OverlayValues[370] = d370
			ps1322.OverlayValues[372] = d372
			ps1322.OverlayValues[373] = d373
			ps1322.OverlayValues[374] = d374
			ps1322.OverlayValues[375] = d375
			ps1322.OverlayValues[376] = d376
			ps1322.OverlayValues[377] = d377
			ps1322.OverlayValues[378] = d378
			ps1322.OverlayValues[379] = d379
			ps1322.OverlayValues[381] = d381
			ps1322.OverlayValues[383] = d383
			ps1322.OverlayValues[384] = d384
			ps1322.OverlayValues[385] = d385
			ps1322.OverlayValues[486] = d486
			ps1322.OverlayValues[487] = d487
			ps1322.OverlayValues[490] = d490
			ps1322.OverlayValues[594] = d594
			ps1322.OverlayValues[595] = d595
			ps1322.OverlayValues[596] = d596
			ps1322.OverlayValues[597] = d597
			ps1322.OverlayValues[598] = d598
			ps1322.OverlayValues[600] = d600
			ps1322.OverlayValues[601] = d601
			ps1322.OverlayValues[602] = d602
			ps1322.OverlayValues[603] = d603
			ps1322.OverlayValues[604] = d604
			ps1322.OverlayValues[605] = d605
			ps1322.OverlayValues[606] = d606
			ps1322.OverlayValues[607] = d607
			ps1322.OverlayValues[608] = d608
			ps1322.OverlayValues[609] = d609
			ps1322.OverlayValues[610] = d610
			ps1322.OverlayValues[611] = d611
			ps1322.OverlayValues[612] = d612
			ps1322.OverlayValues[613] = d613
			ps1322.OverlayValues[614] = d614
			ps1322.OverlayValues[615] = d615
			ps1322.OverlayValues[616] = d616
			ps1322.OverlayValues[617] = d617
			ps1322.OverlayValues[618] = d618
			ps1322.OverlayValues[619] = d619
			ps1322.OverlayValues[620] = d620
			ps1322.OverlayValues[621] = d621
			ps1322.OverlayValues[622] = d622
			ps1322.OverlayValues[623] = d623
			ps1322.OverlayValues[624] = d624
			ps1322.OverlayValues[625] = d625
			ps1322.OverlayValues[626] = d626
			ps1322.OverlayValues[627] = d627
			ps1322.OverlayValues[628] = d628
			ps1322.OverlayValues[629] = d629
			ps1322.OverlayValues[630] = d630
			ps1322.OverlayValues[913] = d913
			ps1322.OverlayValues[914] = d914
			ps1322.OverlayValues[915] = d915
			ps1322.OverlayValues[917] = d917
			ps1322.OverlayValues[918] = d918
			ps1322.OverlayValues[919] = d919
			ps1322.OverlayValues[920] = d920
			ps1322.OverlayValues[921] = d921
			ps1322.OverlayValues[922] = d922
			ps1322.OverlayValues[923] = d923
			ps1322.OverlayValues[925] = d925
			ps1322.OverlayValues[927] = d927
			ps1322.OverlayValues[928] = d928
			ps1322.OverlayValues[1081] = d1081
			ps1322.OverlayValues[1082] = d1082
			ps1322.OverlayValues[1085] = d1085
			ps1322.OverlayValues[1241] = d1241
			ps1322.OverlayValues[1242] = d1242
			ps1322.OverlayValues[1243] = d1243
			ps1322.OverlayValues[1244] = d1244
			ps1322.OverlayValues[1246] = d1246
			ps1322.OverlayValues[1247] = d1247
			ps1322.OverlayValues[1248] = d1248
			ps1322.OverlayValues[1249] = d1249
			ps1322.OverlayValues[1250] = d1250
			ps1322.OverlayValues[1251] = d1251
			ps1322.OverlayValues[1252] = d1252
			ps1322.OverlayValues[1253] = d1253
			ps1322.OverlayValues[1255] = d1255
			ps1322.OverlayValues[1256] = d1256
			ps1322.OverlayValues[1257] = d1257
			ps1322.OverlayValues[1258] = d1258
			ps1322.OverlayValues[1259] = d1259
			ps1322.OverlayValues[1260] = d1260
			ps1322.OverlayValues[1261] = d1261
			ps1322.OverlayValues[1262] = d1262
			ps1322.OverlayValues[1263] = d1263
			ps1322.OverlayValues[1264] = d1264
			ps1322.OverlayValues[1265] = d1265
			ps1322.OverlayValues[1266] = d1266
			ps1322.OverlayValues[1267] = d1267
			ps1322.OverlayValues[1268] = d1268
			ps1322.OverlayValues[1269] = d1269
			ps1322.OverlayValues[1270] = d1270
			ps1322.OverlayValues[1271] = d1271
			ps1322.OverlayValues[1272] = d1272
			ps1322.OverlayValues[1273] = d1273
			ps1322.OverlayValues[1274] = d1274
			ps1322.OverlayValues[1275] = d1275
			ps1322.OverlayValues[1276] = d1276
			ps1322.OverlayValues[1277] = d1277
			ps1322.OverlayValues[1278] = d1278
			ps1322.OverlayValues[1279] = d1279
			ps1322.OverlayValues[1280] = d1280
			ps1322.OverlayValues[1281] = d1281
			ps1322.OverlayValues[1282] = d1282
			ps1322.OverlayValues[1283] = d1283
			ps1322.OverlayValues[1284] = d1284
			ps1322.OverlayValues[1285] = d1285
			ps1322.OverlayValues[1286] = d1286
			ps1322.OverlayValues[1287] = d1287
			ps1322.OverlayValues[1288] = d1288
			ps1322.OverlayValues[1289] = d1289
			ps1322.OverlayValues[1290] = d1290
			ps1322.OverlayValues[1291] = d1291
			ps1322.OverlayValues[1292] = d1292
			ps1322.OverlayValues[1293] = d1293
			ps1322.OverlayValues[1294] = d1294
			ps1322.OverlayValues[1295] = d1295
			ps1322.OverlayValues[1296] = d1296
			ps1322.OverlayValues[1297] = d1297
			ps1322.OverlayValues[1298] = d1298
			ps1322.OverlayValues[1299] = d1299
			ps1322.OverlayValues[1300] = d1300
			ps1322.OverlayValues[1301] = d1301
			ps1322.OverlayValues[1302] = d1302
			ps1322.OverlayValues[1303] = d1303
			ps1322.OverlayValues[1304] = d1304
			ps1322.OverlayValues[1305] = d1305
			ps1322.OverlayValues[1306] = d1306
			ps1322.OverlayValues[1307] = d1307
			ps1322.OverlayValues[1308] = d1308
			ps1322.OverlayValues[1309] = d1309
			ps1322.OverlayValues[1310] = d1310
			ps1322.OverlayValues[1311] = d1311
			ps1322.OverlayValues[1312] = d1312
			ps1322.OverlayValues[1313] = d1313
			ps1322.OverlayValues[1314] = d1314
			ps1322.OverlayValues[1315] = d1315
			ps1322.OverlayValues[1316] = d1316
			ps1322.OverlayValues[1317] = d1317
			ps1322.OverlayValues[1318] = d1318
			ps1322.OverlayValues[1319] = d1319
			ps1322.OverlayValues[1320] = d1320
			return bbs[12].RenderPS(ps1322)
		}
		if !ps.General {
			ps.General = true
			return bbs[13].RenderPS(ps)
		}
		lbl30 := ctx.ReserveLabel()
		lbl31 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d1320.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl30)
		ctx.EmitJmp(lbl31)
		snap1323 := d1
		snap1324 := d2
		snap1325 := d3
		snap1326 := d4
		snap1327 := d5
		snap1328 := d6
		snap1329 := d7
		snap1330 := d8
		snap1331 := d9
		snap1332 := d10
		snap1333 := d11
		snap1334 := d12
		snap1335 := d13
		snap1336 := d14
		snap1337 := d15
		snap1338 := d17
		snap1339 := d18
		snap1340 := d19
		snap1341 := d20
		snap1342 := d21
		snap1343 := d22
		snap1344 := d23
		snap1345 := d24
		snap1346 := d25
		snap1347 := d26
		snap1348 := d27
		snap1349 := d28
		snap1350 := d29
		snap1351 := d30
		snap1352 := d31
		snap1353 := d32
		snap1354 := d33
		snap1355 := d34
		snap1356 := d35
		snap1357 := d36
		snap1358 := d37
		snap1359 := d38
		snap1360 := d39
		snap1361 := d40
		snap1362 := d41
		snap1363 := d42
		snap1364 := d43
		snap1365 := d44
		snap1366 := d45
		snap1367 := d46
		snap1368 := d47
		snap1369 := d48
		snap1370 := d49
		snap1371 := d50
		snap1372 := d53
		snap1373 := d54
		snap1374 := d55
		snap1375 := d164
		snap1376 := d165
		snap1377 := d166
		snap1378 := d167
		snap1379 := d168
		snap1380 := d169
		snap1381 := d170
		snap1382 := d171
		snap1383 := d172
		snap1384 := d173
		snap1385 := d174
		snap1386 := d175
		snap1387 := d176
		snap1388 := d177
		snap1389 := d178
		snap1390 := d179
		snap1391 := d180
		snap1392 := d181
		snap1393 := d182
		snap1394 := d183
		snap1395 := d184
		snap1396 := d185
		snap1397 := d186
		snap1398 := d187
		snap1399 := d188
		snap1400 := d189
		snap1401 := d190
		snap1402 := d191
		snap1403 := d192
		snap1404 := d193
		snap1405 := d196
		snap1406 := d367
		snap1407 := d368
		snap1408 := d369
		snap1409 := d370
		snap1410 := d372
		snap1411 := d373
		snap1412 := d374
		snap1413 := d375
		snap1414 := d376
		snap1415 := d377
		snap1416 := d378
		snap1417 := d379
		snap1418 := d381
		snap1419 := d383
		snap1420 := d384
		snap1421 := d385
		snap1422 := d486
		snap1423 := d487
		snap1424 := d490
		snap1425 := d594
		snap1426 := d595
		snap1427 := d596
		snap1428 := d597
		snap1429 := d598
		snap1430 := d600
		snap1431 := d601
		snap1432 := d602
		snap1433 := d603
		snap1434 := d604
		snap1435 := d605
		snap1436 := d606
		snap1437 := d607
		snap1438 := d608
		snap1439 := d609
		snap1440 := d610
		snap1441 := d611
		snap1442 := d612
		snap1443 := d613
		snap1444 := d614
		snap1445 := d615
		snap1446 := d616
		snap1447 := d617
		snap1448 := d618
		snap1449 := d619
		snap1450 := d620
		snap1451 := d621
		snap1452 := d622
		snap1453 := d623
		snap1454 := d624
		snap1455 := d625
		snap1456 := d626
		snap1457 := d627
		snap1458 := d628
		snap1459 := d629
		snap1460 := d630
		snap1461 := d913
		snap1462 := d914
		snap1463 := d915
		snap1464 := d917
		snap1465 := d918
		snap1466 := d919
		snap1467 := d920
		snap1468 := d921
		snap1469 := d922
		snap1470 := d923
		snap1471 := d925
		snap1472 := d927
		snap1473 := d928
		snap1474 := d1081
		snap1475 := d1082
		snap1476 := d1085
		snap1477 := d1241
		snap1478 := d1242
		snap1479 := d1243
		snap1480 := d1244
		snap1481 := d1246
		snap1482 := d1247
		snap1483 := d1248
		snap1484 := d1249
		snap1485 := d1250
		snap1486 := d1251
		snap1487 := d1252
		snap1488 := d1253
		snap1489 := d1255
		snap1490 := d1256
		snap1491 := d1257
		snap1492 := d1258
		snap1493 := d1259
		snap1494 := d1260
		snap1495 := d1261
		snap1496 := d1262
		snap1497 := d1263
		snap1498 := d1264
		snap1499 := d1265
		snap1500 := d1266
		snap1501 := d1267
		snap1502 := d1268
		snap1503 := d1269
		snap1504 := d1270
		snap1505 := d1271
		snap1506 := d1272
		snap1507 := d1273
		snap1508 := d1274
		snap1509 := d1275
		snap1510 := d1276
		snap1511 := d1277
		snap1512 := d1278
		snap1513 := d1279
		snap1514 := d1280
		snap1515 := d1281
		snap1516 := d1282
		snap1517 := d1283
		snap1518 := d1284
		snap1519 := d1285
		snap1520 := d1286
		snap1521 := d1287
		snap1522 := d1288
		snap1523 := d1289
		snap1524 := d1290
		snap1525 := d1291
		snap1526 := d1292
		snap1527 := d1293
		snap1528 := d1294
		snap1529 := d1295
		snap1530 := d1296
		snap1531 := d1297
		snap1532 := d1298
		snap1533 := d1299
		snap1534 := d1300
		snap1535 := d1301
		snap1536 := d1302
		snap1537 := d1303
		snap1538 := d1304
		snap1539 := d1305
		snap1540 := d1306
		snap1541 := d1307
		snap1542 := d1308
		snap1543 := d1309
		snap1544 := d1310
		snap1545 := d1311
		snap1546 := d1312
		snap1547 := d1313
		snap1548 := d1314
		snap1549 := d1315
		snap1550 := d1316
		snap1551 := d1317
		snap1552 := d1318
		snap1553 := d1319
		snap1554 := d1320
		alloc1555 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl30)
		ctx.EmitJmp(lbl12)
		ctx.RestoreAllocState(alloc1555)
		d1 = snap1323
		d2 = snap1324
		d3 = snap1325
		d4 = snap1326
		d5 = snap1327
		d6 = snap1328
		d7 = snap1329
		d8 = snap1330
		d9 = snap1331
		d10 = snap1332
		d11 = snap1333
		d12 = snap1334
		d13 = snap1335
		d14 = snap1336
		d15 = snap1337
		d17 = snap1338
		d18 = snap1339
		d19 = snap1340
		d20 = snap1341
		d21 = snap1342
		d22 = snap1343
		d23 = snap1344
		d24 = snap1345
		d25 = snap1346
		d26 = snap1347
		d27 = snap1348
		d28 = snap1349
		d29 = snap1350
		d30 = snap1351
		d31 = snap1352
		d32 = snap1353
		d33 = snap1354
		d34 = snap1355
		d35 = snap1356
		d36 = snap1357
		d37 = snap1358
		d38 = snap1359
		d39 = snap1360
		d40 = snap1361
		d41 = snap1362
		d42 = snap1363
		d43 = snap1364
		d44 = snap1365
		d45 = snap1366
		d46 = snap1367
		d47 = snap1368
		d48 = snap1369
		d49 = snap1370
		d50 = snap1371
		d53 = snap1372
		d54 = snap1373
		d55 = snap1374
		d164 = snap1375
		d165 = snap1376
		d166 = snap1377
		d167 = snap1378
		d168 = snap1379
		d169 = snap1380
		d170 = snap1381
		d171 = snap1382
		d172 = snap1383
		d173 = snap1384
		d174 = snap1385
		d175 = snap1386
		d176 = snap1387
		d177 = snap1388
		d178 = snap1389
		d179 = snap1390
		d180 = snap1391
		d181 = snap1392
		d182 = snap1393
		d183 = snap1394
		d184 = snap1395
		d185 = snap1396
		d186 = snap1397
		d187 = snap1398
		d188 = snap1399
		d189 = snap1400
		d190 = snap1401
		d191 = snap1402
		d192 = snap1403
		d193 = snap1404
		d196 = snap1405
		d367 = snap1406
		d368 = snap1407
		d369 = snap1408
		d370 = snap1409
		d372 = snap1410
		d373 = snap1411
		d374 = snap1412
		d375 = snap1413
		d376 = snap1414
		d377 = snap1415
		d378 = snap1416
		d379 = snap1417
		d381 = snap1418
		d383 = snap1419
		d384 = snap1420
		d385 = snap1421
		d486 = snap1422
		d487 = snap1423
		d490 = snap1424
		d594 = snap1425
		d595 = snap1426
		d596 = snap1427
		d597 = snap1428
		d598 = snap1429
		d600 = snap1430
		d601 = snap1431
		d602 = snap1432
		d603 = snap1433
		d604 = snap1434
		d605 = snap1435
		d606 = snap1436
		d607 = snap1437
		d608 = snap1438
		d609 = snap1439
		d610 = snap1440
		d611 = snap1441
		d612 = snap1442
		d613 = snap1443
		d614 = snap1444
		d615 = snap1445
		d616 = snap1446
		d617 = snap1447
		d618 = snap1448
		d619 = snap1449
		d620 = snap1450
		d621 = snap1451
		d622 = snap1452
		d623 = snap1453
		d624 = snap1454
		d625 = snap1455
		d626 = snap1456
		d627 = snap1457
		d628 = snap1458
		d629 = snap1459
		d630 = snap1460
		d913 = snap1461
		d914 = snap1462
		d915 = snap1463
		d917 = snap1464
		d918 = snap1465
		d919 = snap1466
		d920 = snap1467
		d921 = snap1468
		d922 = snap1469
		d923 = snap1470
		d925 = snap1471
		d927 = snap1472
		d928 = snap1473
		d1081 = snap1474
		d1082 = snap1475
		d1085 = snap1476
		d1241 = snap1477
		d1242 = snap1478
		d1243 = snap1479
		d1244 = snap1480
		d1246 = snap1481
		d1247 = snap1482
		d1248 = snap1483
		d1249 = snap1484
		d1250 = snap1485
		d1251 = snap1486
		d1252 = snap1487
		d1253 = snap1488
		d1255 = snap1489
		d1256 = snap1490
		d1257 = snap1491
		d1258 = snap1492
		d1259 = snap1493
		d1260 = snap1494
		d1261 = snap1495
		d1262 = snap1496
		d1263 = snap1497
		d1264 = snap1498
		d1265 = snap1499
		d1266 = snap1500
		d1267 = snap1501
		d1268 = snap1502
		d1269 = snap1503
		d1270 = snap1504
		d1271 = snap1505
		d1272 = snap1506
		d1273 = snap1507
		d1274 = snap1508
		d1275 = snap1509
		d1276 = snap1510
		d1277 = snap1511
		d1278 = snap1512
		d1279 = snap1513
		d1280 = snap1514
		d1281 = snap1515
		d1282 = snap1516
		d1283 = snap1517
		d1284 = snap1518
		d1285 = snap1519
		d1286 = snap1520
		d1287 = snap1521
		d1288 = snap1522
		d1289 = snap1523
		d1290 = snap1524
		d1291 = snap1525
		d1292 = snap1526
		d1293 = snap1527
		d1294 = snap1528
		d1295 = snap1529
		d1296 = snap1530
		d1297 = snap1531
		d1298 = snap1532
		d1299 = snap1533
		d1300 = snap1534
		d1301 = snap1535
		d1302 = snap1536
		d1303 = snap1537
		d1304 = snap1538
		d1305 = snap1539
		d1306 = snap1540
		d1307 = snap1541
		d1308 = snap1542
		d1309 = snap1543
		d1310 = snap1544
		d1311 = snap1545
		d1312 = snap1546
		d1313 = snap1547
		d1314 = snap1548
		d1315 = snap1549
		d1316 = snap1550
		d1317 = snap1551
		d1318 = snap1552
		d1319 = snap1553
		d1320 = snap1554
		ctx.MarkLabel(lbl31)
		ctx.EmitJmp(lbl13)
		ctx.RestoreAllocState(alloc1555)
		d1 = snap1323
		d2 = snap1324
		d3 = snap1325
		d4 = snap1326
		d5 = snap1327
		d6 = snap1328
		d7 = snap1329
		d8 = snap1330
		d9 = snap1331
		d10 = snap1332
		d11 = snap1333
		d12 = snap1334
		d13 = snap1335
		d14 = snap1336
		d15 = snap1337
		d17 = snap1338
		d18 = snap1339
		d19 = snap1340
		d20 = snap1341
		d21 = snap1342
		d22 = snap1343
		d23 = snap1344
		d24 = snap1345
		d25 = snap1346
		d26 = snap1347
		d27 = snap1348
		d28 = snap1349
		d29 = snap1350
		d30 = snap1351
		d31 = snap1352
		d32 = snap1353
		d33 = snap1354
		d34 = snap1355
		d35 = snap1356
		d36 = snap1357
		d37 = snap1358
		d38 = snap1359
		d39 = snap1360
		d40 = snap1361
		d41 = snap1362
		d42 = snap1363
		d43 = snap1364
		d44 = snap1365
		d45 = snap1366
		d46 = snap1367
		d47 = snap1368
		d48 = snap1369
		d49 = snap1370
		d50 = snap1371
		d53 = snap1372
		d54 = snap1373
		d55 = snap1374
		d164 = snap1375
		d165 = snap1376
		d166 = snap1377
		d167 = snap1378
		d168 = snap1379
		d169 = snap1380
		d170 = snap1381
		d171 = snap1382
		d172 = snap1383
		d173 = snap1384
		d174 = snap1385
		d175 = snap1386
		d176 = snap1387
		d177 = snap1388
		d178 = snap1389
		d179 = snap1390
		d180 = snap1391
		d181 = snap1392
		d182 = snap1393
		d183 = snap1394
		d184 = snap1395
		d185 = snap1396
		d186 = snap1397
		d187 = snap1398
		d188 = snap1399
		d189 = snap1400
		d190 = snap1401
		d191 = snap1402
		d192 = snap1403
		d193 = snap1404
		d196 = snap1405
		d367 = snap1406
		d368 = snap1407
		d369 = snap1408
		d370 = snap1409
		d372 = snap1410
		d373 = snap1411
		d374 = snap1412
		d375 = snap1413
		d376 = snap1414
		d377 = snap1415
		d378 = snap1416
		d379 = snap1417
		d381 = snap1418
		d383 = snap1419
		d384 = snap1420
		d385 = snap1421
		d486 = snap1422
		d487 = snap1423
		d490 = snap1424
		d594 = snap1425
		d595 = snap1426
		d596 = snap1427
		d597 = snap1428
		d598 = snap1429
		d600 = snap1430
		d601 = snap1431
		d602 = snap1432
		d603 = snap1433
		d604 = snap1434
		d605 = snap1435
		d606 = snap1436
		d607 = snap1437
		d608 = snap1438
		d609 = snap1439
		d610 = snap1440
		d611 = snap1441
		d612 = snap1442
		d613 = snap1443
		d614 = snap1444
		d615 = snap1445
		d616 = snap1446
		d617 = snap1447
		d618 = snap1448
		d619 = snap1449
		d620 = snap1450
		d621 = snap1451
		d622 = snap1452
		d623 = snap1453
		d624 = snap1454
		d625 = snap1455
		d626 = snap1456
		d627 = snap1457
		d628 = snap1458
		d629 = snap1459
		d630 = snap1460
		d913 = snap1461
		d914 = snap1462
		d915 = snap1463
		d917 = snap1464
		d918 = snap1465
		d919 = snap1466
		d920 = snap1467
		d921 = snap1468
		d922 = snap1469
		d923 = snap1470
		d925 = snap1471
		d927 = snap1472
		d928 = snap1473
		d1081 = snap1474
		d1082 = snap1475
		d1085 = snap1476
		d1241 = snap1477
		d1242 = snap1478
		d1243 = snap1479
		d1244 = snap1480
		d1246 = snap1481
		d1247 = snap1482
		d1248 = snap1483
		d1249 = snap1484
		d1250 = snap1485
		d1251 = snap1486
		d1252 = snap1487
		d1253 = snap1488
		d1255 = snap1489
		d1256 = snap1490
		d1257 = snap1491
		d1258 = snap1492
		d1259 = snap1493
		d1260 = snap1494
		d1261 = snap1495
		d1262 = snap1496
		d1263 = snap1497
		d1264 = snap1498
		d1265 = snap1499
		d1266 = snap1500
		d1267 = snap1501
		d1268 = snap1502
		d1269 = snap1503
		d1270 = snap1504
		d1271 = snap1505
		d1272 = snap1506
		d1273 = snap1507
		d1274 = snap1508
		d1275 = snap1509
		d1276 = snap1510
		d1277 = snap1511
		d1278 = snap1512
		d1279 = snap1513
		d1280 = snap1514
		d1281 = snap1515
		d1282 = snap1516
		d1283 = snap1517
		d1284 = snap1518
		d1285 = snap1519
		d1286 = snap1520
		d1287 = snap1521
		d1288 = snap1522
		d1289 = snap1523
		d1290 = snap1524
		d1291 = snap1525
		d1292 = snap1526
		d1293 = snap1527
		d1294 = snap1528
		d1295 = snap1529
		d1296 = snap1530
		d1297 = snap1531
		d1298 = snap1532
		d1299 = snap1533
		d1300 = snap1534
		d1301 = snap1535
		d1302 = snap1536
		d1303 = snap1537
		d1304 = snap1538
		d1305 = snap1539
		d1306 = snap1540
		d1307 = snap1541
		d1308 = snap1542
		d1309 = snap1543
		d1310 = snap1544
		d1311 = snap1545
		d1312 = snap1546
		d1313 = snap1547
		d1314 = snap1548
		d1315 = snap1549
		d1316 = snap1550
		d1317 = snap1551
		d1318 = snap1552
		d1319 = snap1553
		d1320 = snap1554
		ps1556 := scm.PhiState{General: true}
		ps1556.OverlayValues = make([]scm.JITValueDesc, 1321)
		ps1556.OverlayValues[1] = d1
		ps1556.OverlayValues[2] = d2
		ps1556.OverlayValues[3] = d3
		ps1556.OverlayValues[4] = d4
		ps1556.OverlayValues[5] = d5
		ps1556.OverlayValues[6] = d6
		ps1556.OverlayValues[7] = d7
		ps1556.OverlayValues[8] = d8
		ps1556.OverlayValues[9] = d9
		ps1556.OverlayValues[10] = d10
		ps1556.OverlayValues[11] = d11
		ps1556.OverlayValues[12] = d12
		ps1556.OverlayValues[13] = d13
		ps1556.OverlayValues[14] = d14
		ps1556.OverlayValues[15] = d15
		ps1556.OverlayValues[17] = d17
		ps1556.OverlayValues[18] = d18
		ps1556.OverlayValues[19] = d19
		ps1556.OverlayValues[20] = d20
		ps1556.OverlayValues[21] = d21
		ps1556.OverlayValues[22] = d22
		ps1556.OverlayValues[23] = d23
		ps1556.OverlayValues[24] = d24
		ps1556.OverlayValues[25] = d25
		ps1556.OverlayValues[26] = d26
		ps1556.OverlayValues[27] = d27
		ps1556.OverlayValues[28] = d28
		ps1556.OverlayValues[29] = d29
		ps1556.OverlayValues[30] = d30
		ps1556.OverlayValues[31] = d31
		ps1556.OverlayValues[32] = d32
		ps1556.OverlayValues[33] = d33
		ps1556.OverlayValues[34] = d34
		ps1556.OverlayValues[35] = d35
		ps1556.OverlayValues[36] = d36
		ps1556.OverlayValues[37] = d37
		ps1556.OverlayValues[38] = d38
		ps1556.OverlayValues[39] = d39
		ps1556.OverlayValues[40] = d40
		ps1556.OverlayValues[41] = d41
		ps1556.OverlayValues[42] = d42
		ps1556.OverlayValues[43] = d43
		ps1556.OverlayValues[44] = d44
		ps1556.OverlayValues[45] = d45
		ps1556.OverlayValues[46] = d46
		ps1556.OverlayValues[47] = d47
		ps1556.OverlayValues[48] = d48
		ps1556.OverlayValues[49] = d49
		ps1556.OverlayValues[50] = d50
		ps1556.OverlayValues[53] = d53
		ps1556.OverlayValues[54] = d54
		ps1556.OverlayValues[55] = d55
		ps1556.OverlayValues[164] = d164
		ps1556.OverlayValues[165] = d165
		ps1556.OverlayValues[166] = d166
		ps1556.OverlayValues[167] = d167
		ps1556.OverlayValues[168] = d168
		ps1556.OverlayValues[169] = d169
		ps1556.OverlayValues[170] = d170
		ps1556.OverlayValues[171] = d171
		ps1556.OverlayValues[172] = d172
		ps1556.OverlayValues[173] = d173
		ps1556.OverlayValues[174] = d174
		ps1556.OverlayValues[175] = d175
		ps1556.OverlayValues[176] = d176
		ps1556.OverlayValues[177] = d177
		ps1556.OverlayValues[178] = d178
		ps1556.OverlayValues[179] = d179
		ps1556.OverlayValues[180] = d180
		ps1556.OverlayValues[181] = d181
		ps1556.OverlayValues[182] = d182
		ps1556.OverlayValues[183] = d183
		ps1556.OverlayValues[184] = d184
		ps1556.OverlayValues[185] = d185
		ps1556.OverlayValues[186] = d186
		ps1556.OverlayValues[187] = d187
		ps1556.OverlayValues[188] = d188
		ps1556.OverlayValues[189] = d189
		ps1556.OverlayValues[190] = d190
		ps1556.OverlayValues[191] = d191
		ps1556.OverlayValues[192] = d192
		ps1556.OverlayValues[193] = d193
		ps1556.OverlayValues[196] = d196
		ps1556.OverlayValues[367] = d367
		ps1556.OverlayValues[368] = d368
		ps1556.OverlayValues[369] = d369
		ps1556.OverlayValues[370] = d370
		ps1556.OverlayValues[372] = d372
		ps1556.OverlayValues[373] = d373
		ps1556.OverlayValues[374] = d374
		ps1556.OverlayValues[375] = d375
		ps1556.OverlayValues[376] = d376
		ps1556.OverlayValues[377] = d377
		ps1556.OverlayValues[378] = d378
		ps1556.OverlayValues[379] = d379
		ps1556.OverlayValues[381] = d381
		ps1556.OverlayValues[383] = d383
		ps1556.OverlayValues[384] = d384
		ps1556.OverlayValues[385] = d385
		ps1556.OverlayValues[486] = d486
		ps1556.OverlayValues[487] = d487
		ps1556.OverlayValues[490] = d490
		ps1556.OverlayValues[594] = d594
		ps1556.OverlayValues[595] = d595
		ps1556.OverlayValues[596] = d596
		ps1556.OverlayValues[597] = d597
		ps1556.OverlayValues[598] = d598
		ps1556.OverlayValues[600] = d600
		ps1556.OverlayValues[601] = d601
		ps1556.OverlayValues[602] = d602
		ps1556.OverlayValues[603] = d603
		ps1556.OverlayValues[604] = d604
		ps1556.OverlayValues[605] = d605
		ps1556.OverlayValues[606] = d606
		ps1556.OverlayValues[607] = d607
		ps1556.OverlayValues[608] = d608
		ps1556.OverlayValues[609] = d609
		ps1556.OverlayValues[610] = d610
		ps1556.OverlayValues[611] = d611
		ps1556.OverlayValues[612] = d612
		ps1556.OverlayValues[613] = d613
		ps1556.OverlayValues[614] = d614
		ps1556.OverlayValues[615] = d615
		ps1556.OverlayValues[616] = d616
		ps1556.OverlayValues[617] = d617
		ps1556.OverlayValues[618] = d618
		ps1556.OverlayValues[619] = d619
		ps1556.OverlayValues[620] = d620
		ps1556.OverlayValues[621] = d621
		ps1556.OverlayValues[622] = d622
		ps1556.OverlayValues[623] = d623
		ps1556.OverlayValues[624] = d624
		ps1556.OverlayValues[625] = d625
		ps1556.OverlayValues[626] = d626
		ps1556.OverlayValues[627] = d627
		ps1556.OverlayValues[628] = d628
		ps1556.OverlayValues[629] = d629
		ps1556.OverlayValues[630] = d630
		ps1556.OverlayValues[913] = d913
		ps1556.OverlayValues[914] = d914
		ps1556.OverlayValues[915] = d915
		ps1556.OverlayValues[917] = d917
		ps1556.OverlayValues[918] = d918
		ps1556.OverlayValues[919] = d919
		ps1556.OverlayValues[920] = d920
		ps1556.OverlayValues[921] = d921
		ps1556.OverlayValues[922] = d922
		ps1556.OverlayValues[923] = d923
		ps1556.OverlayValues[925] = d925
		ps1556.OverlayValues[927] = d927
		ps1556.OverlayValues[928] = d928
		ps1556.OverlayValues[1081] = d1081
		ps1556.OverlayValues[1082] = d1082
		ps1556.OverlayValues[1085] = d1085
		ps1556.OverlayValues[1241] = d1241
		ps1556.OverlayValues[1242] = d1242
		ps1556.OverlayValues[1243] = d1243
		ps1556.OverlayValues[1244] = d1244
		ps1556.OverlayValues[1246] = d1246
		ps1556.OverlayValues[1247] = d1247
		ps1556.OverlayValues[1248] = d1248
		ps1556.OverlayValues[1249] = d1249
		ps1556.OverlayValues[1250] = d1250
		ps1556.OverlayValues[1251] = d1251
		ps1556.OverlayValues[1252] = d1252
		ps1556.OverlayValues[1253] = d1253
		ps1556.OverlayValues[1255] = d1255
		ps1556.OverlayValues[1256] = d1256
		ps1556.OverlayValues[1257] = d1257
		ps1556.OverlayValues[1258] = d1258
		ps1556.OverlayValues[1259] = d1259
		ps1556.OverlayValues[1260] = d1260
		ps1556.OverlayValues[1261] = d1261
		ps1556.OverlayValues[1262] = d1262
		ps1556.OverlayValues[1263] = d1263
		ps1556.OverlayValues[1264] = d1264
		ps1556.OverlayValues[1265] = d1265
		ps1556.OverlayValues[1266] = d1266
		ps1556.OverlayValues[1267] = d1267
		ps1556.OverlayValues[1268] = d1268
		ps1556.OverlayValues[1269] = d1269
		ps1556.OverlayValues[1270] = d1270
		ps1556.OverlayValues[1271] = d1271
		ps1556.OverlayValues[1272] = d1272
		ps1556.OverlayValues[1273] = d1273
		ps1556.OverlayValues[1274] = d1274
		ps1556.OverlayValues[1275] = d1275
		ps1556.OverlayValues[1276] = d1276
		ps1556.OverlayValues[1277] = d1277
		ps1556.OverlayValues[1278] = d1278
		ps1556.OverlayValues[1279] = d1279
		ps1556.OverlayValues[1280] = d1280
		ps1556.OverlayValues[1281] = d1281
		ps1556.OverlayValues[1282] = d1282
		ps1556.OverlayValues[1283] = d1283
		ps1556.OverlayValues[1284] = d1284
		ps1556.OverlayValues[1285] = d1285
		ps1556.OverlayValues[1286] = d1286
		ps1556.OverlayValues[1287] = d1287
		ps1556.OverlayValues[1288] = d1288
		ps1556.OverlayValues[1289] = d1289
		ps1556.OverlayValues[1290] = d1290
		ps1556.OverlayValues[1291] = d1291
		ps1556.OverlayValues[1292] = d1292
		ps1556.OverlayValues[1293] = d1293
		ps1556.OverlayValues[1294] = d1294
		ps1556.OverlayValues[1295] = d1295
		ps1556.OverlayValues[1296] = d1296
		ps1556.OverlayValues[1297] = d1297
		ps1556.OverlayValues[1298] = d1298
		ps1556.OverlayValues[1299] = d1299
		ps1556.OverlayValues[1300] = d1300
		ps1556.OverlayValues[1301] = d1301
		ps1556.OverlayValues[1302] = d1302
		ps1556.OverlayValues[1303] = d1303
		ps1556.OverlayValues[1304] = d1304
		ps1556.OverlayValues[1305] = d1305
		ps1556.OverlayValues[1306] = d1306
		ps1556.OverlayValues[1307] = d1307
		ps1556.OverlayValues[1308] = d1308
		ps1556.OverlayValues[1309] = d1309
		ps1556.OverlayValues[1310] = d1310
		ps1556.OverlayValues[1311] = d1311
		ps1556.OverlayValues[1312] = d1312
		ps1556.OverlayValues[1313] = d1313
		ps1556.OverlayValues[1314] = d1314
		ps1556.OverlayValues[1315] = d1315
		ps1556.OverlayValues[1316] = d1316
		ps1556.OverlayValues[1317] = d1317
		ps1556.OverlayValues[1318] = d1318
		ps1556.OverlayValues[1319] = d1319
		ps1556.OverlayValues[1320] = d1320
		ps1557 := scm.PhiState{General: true}
		ps1557.OverlayValues = make([]scm.JITValueDesc, 1321)
		ps1557.OverlayValues[1] = d1
		ps1557.OverlayValues[2] = d2
		ps1557.OverlayValues[3] = d3
		ps1557.OverlayValues[4] = d4
		ps1557.OverlayValues[5] = d5
		ps1557.OverlayValues[6] = d6
		ps1557.OverlayValues[7] = d7
		ps1557.OverlayValues[8] = d8
		ps1557.OverlayValues[9] = d9
		ps1557.OverlayValues[10] = d10
		ps1557.OverlayValues[11] = d11
		ps1557.OverlayValues[12] = d12
		ps1557.OverlayValues[13] = d13
		ps1557.OverlayValues[14] = d14
		ps1557.OverlayValues[15] = d15
		ps1557.OverlayValues[17] = d17
		ps1557.OverlayValues[18] = d18
		ps1557.OverlayValues[19] = d19
		ps1557.OverlayValues[20] = d20
		ps1557.OverlayValues[21] = d21
		ps1557.OverlayValues[22] = d22
		ps1557.OverlayValues[23] = d23
		ps1557.OverlayValues[24] = d24
		ps1557.OverlayValues[25] = d25
		ps1557.OverlayValues[26] = d26
		ps1557.OverlayValues[27] = d27
		ps1557.OverlayValues[28] = d28
		ps1557.OverlayValues[29] = d29
		ps1557.OverlayValues[30] = d30
		ps1557.OverlayValues[31] = d31
		ps1557.OverlayValues[32] = d32
		ps1557.OverlayValues[33] = d33
		ps1557.OverlayValues[34] = d34
		ps1557.OverlayValues[35] = d35
		ps1557.OverlayValues[36] = d36
		ps1557.OverlayValues[37] = d37
		ps1557.OverlayValues[38] = d38
		ps1557.OverlayValues[39] = d39
		ps1557.OverlayValues[40] = d40
		ps1557.OverlayValues[41] = d41
		ps1557.OverlayValues[42] = d42
		ps1557.OverlayValues[43] = d43
		ps1557.OverlayValues[44] = d44
		ps1557.OverlayValues[45] = d45
		ps1557.OverlayValues[46] = d46
		ps1557.OverlayValues[47] = d47
		ps1557.OverlayValues[48] = d48
		ps1557.OverlayValues[49] = d49
		ps1557.OverlayValues[50] = d50
		ps1557.OverlayValues[53] = d53
		ps1557.OverlayValues[54] = d54
		ps1557.OverlayValues[55] = d55
		ps1557.OverlayValues[164] = d164
		ps1557.OverlayValues[165] = d165
		ps1557.OverlayValues[166] = d166
		ps1557.OverlayValues[167] = d167
		ps1557.OverlayValues[168] = d168
		ps1557.OverlayValues[169] = d169
		ps1557.OverlayValues[170] = d170
		ps1557.OverlayValues[171] = d171
		ps1557.OverlayValues[172] = d172
		ps1557.OverlayValues[173] = d173
		ps1557.OverlayValues[174] = d174
		ps1557.OverlayValues[175] = d175
		ps1557.OverlayValues[176] = d176
		ps1557.OverlayValues[177] = d177
		ps1557.OverlayValues[178] = d178
		ps1557.OverlayValues[179] = d179
		ps1557.OverlayValues[180] = d180
		ps1557.OverlayValues[181] = d181
		ps1557.OverlayValues[182] = d182
		ps1557.OverlayValues[183] = d183
		ps1557.OverlayValues[184] = d184
		ps1557.OverlayValues[185] = d185
		ps1557.OverlayValues[186] = d186
		ps1557.OverlayValues[187] = d187
		ps1557.OverlayValues[188] = d188
		ps1557.OverlayValues[189] = d189
		ps1557.OverlayValues[190] = d190
		ps1557.OverlayValues[191] = d191
		ps1557.OverlayValues[192] = d192
		ps1557.OverlayValues[193] = d193
		ps1557.OverlayValues[196] = d196
		ps1557.OverlayValues[367] = d367
		ps1557.OverlayValues[368] = d368
		ps1557.OverlayValues[369] = d369
		ps1557.OverlayValues[370] = d370
		ps1557.OverlayValues[372] = d372
		ps1557.OverlayValues[373] = d373
		ps1557.OverlayValues[374] = d374
		ps1557.OverlayValues[375] = d375
		ps1557.OverlayValues[376] = d376
		ps1557.OverlayValues[377] = d377
		ps1557.OverlayValues[378] = d378
		ps1557.OverlayValues[379] = d379
		ps1557.OverlayValues[381] = d381
		ps1557.OverlayValues[383] = d383
		ps1557.OverlayValues[384] = d384
		ps1557.OverlayValues[385] = d385
		ps1557.OverlayValues[486] = d486
		ps1557.OverlayValues[487] = d487
		ps1557.OverlayValues[490] = d490
		ps1557.OverlayValues[594] = d594
		ps1557.OverlayValues[595] = d595
		ps1557.OverlayValues[596] = d596
		ps1557.OverlayValues[597] = d597
		ps1557.OverlayValues[598] = d598
		ps1557.OverlayValues[600] = d600
		ps1557.OverlayValues[601] = d601
		ps1557.OverlayValues[602] = d602
		ps1557.OverlayValues[603] = d603
		ps1557.OverlayValues[604] = d604
		ps1557.OverlayValues[605] = d605
		ps1557.OverlayValues[606] = d606
		ps1557.OverlayValues[607] = d607
		ps1557.OverlayValues[608] = d608
		ps1557.OverlayValues[609] = d609
		ps1557.OverlayValues[610] = d610
		ps1557.OverlayValues[611] = d611
		ps1557.OverlayValues[612] = d612
		ps1557.OverlayValues[613] = d613
		ps1557.OverlayValues[614] = d614
		ps1557.OverlayValues[615] = d615
		ps1557.OverlayValues[616] = d616
		ps1557.OverlayValues[617] = d617
		ps1557.OverlayValues[618] = d618
		ps1557.OverlayValues[619] = d619
		ps1557.OverlayValues[620] = d620
		ps1557.OverlayValues[621] = d621
		ps1557.OverlayValues[622] = d622
		ps1557.OverlayValues[623] = d623
		ps1557.OverlayValues[624] = d624
		ps1557.OverlayValues[625] = d625
		ps1557.OverlayValues[626] = d626
		ps1557.OverlayValues[627] = d627
		ps1557.OverlayValues[628] = d628
		ps1557.OverlayValues[629] = d629
		ps1557.OverlayValues[630] = d630
		ps1557.OverlayValues[913] = d913
		ps1557.OverlayValues[914] = d914
		ps1557.OverlayValues[915] = d915
		ps1557.OverlayValues[917] = d917
		ps1557.OverlayValues[918] = d918
		ps1557.OverlayValues[919] = d919
		ps1557.OverlayValues[920] = d920
		ps1557.OverlayValues[921] = d921
		ps1557.OverlayValues[922] = d922
		ps1557.OverlayValues[923] = d923
		ps1557.OverlayValues[925] = d925
		ps1557.OverlayValues[927] = d927
		ps1557.OverlayValues[928] = d928
		ps1557.OverlayValues[1081] = d1081
		ps1557.OverlayValues[1082] = d1082
		ps1557.OverlayValues[1085] = d1085
		ps1557.OverlayValues[1241] = d1241
		ps1557.OverlayValues[1242] = d1242
		ps1557.OverlayValues[1243] = d1243
		ps1557.OverlayValues[1244] = d1244
		ps1557.OverlayValues[1246] = d1246
		ps1557.OverlayValues[1247] = d1247
		ps1557.OverlayValues[1248] = d1248
		ps1557.OverlayValues[1249] = d1249
		ps1557.OverlayValues[1250] = d1250
		ps1557.OverlayValues[1251] = d1251
		ps1557.OverlayValues[1252] = d1252
		ps1557.OverlayValues[1253] = d1253
		ps1557.OverlayValues[1255] = d1255
		ps1557.OverlayValues[1256] = d1256
		ps1557.OverlayValues[1257] = d1257
		ps1557.OverlayValues[1258] = d1258
		ps1557.OverlayValues[1259] = d1259
		ps1557.OverlayValues[1260] = d1260
		ps1557.OverlayValues[1261] = d1261
		ps1557.OverlayValues[1262] = d1262
		ps1557.OverlayValues[1263] = d1263
		ps1557.OverlayValues[1264] = d1264
		ps1557.OverlayValues[1265] = d1265
		ps1557.OverlayValues[1266] = d1266
		ps1557.OverlayValues[1267] = d1267
		ps1557.OverlayValues[1268] = d1268
		ps1557.OverlayValues[1269] = d1269
		ps1557.OverlayValues[1270] = d1270
		ps1557.OverlayValues[1271] = d1271
		ps1557.OverlayValues[1272] = d1272
		ps1557.OverlayValues[1273] = d1273
		ps1557.OverlayValues[1274] = d1274
		ps1557.OverlayValues[1275] = d1275
		ps1557.OverlayValues[1276] = d1276
		ps1557.OverlayValues[1277] = d1277
		ps1557.OverlayValues[1278] = d1278
		ps1557.OverlayValues[1279] = d1279
		ps1557.OverlayValues[1280] = d1280
		ps1557.OverlayValues[1281] = d1281
		ps1557.OverlayValues[1282] = d1282
		ps1557.OverlayValues[1283] = d1283
		ps1557.OverlayValues[1284] = d1284
		ps1557.OverlayValues[1285] = d1285
		ps1557.OverlayValues[1286] = d1286
		ps1557.OverlayValues[1287] = d1287
		ps1557.OverlayValues[1288] = d1288
		ps1557.OverlayValues[1289] = d1289
		ps1557.OverlayValues[1290] = d1290
		ps1557.OverlayValues[1291] = d1291
		ps1557.OverlayValues[1292] = d1292
		ps1557.OverlayValues[1293] = d1293
		ps1557.OverlayValues[1294] = d1294
		ps1557.OverlayValues[1295] = d1295
		ps1557.OverlayValues[1296] = d1296
		ps1557.OverlayValues[1297] = d1297
		ps1557.OverlayValues[1298] = d1298
		ps1557.OverlayValues[1299] = d1299
		ps1557.OverlayValues[1300] = d1300
		ps1557.OverlayValues[1301] = d1301
		ps1557.OverlayValues[1302] = d1302
		ps1557.OverlayValues[1303] = d1303
		ps1557.OverlayValues[1304] = d1304
		ps1557.OverlayValues[1305] = d1305
		ps1557.OverlayValues[1306] = d1306
		ps1557.OverlayValues[1307] = d1307
		ps1557.OverlayValues[1308] = d1308
		ps1557.OverlayValues[1309] = d1309
		ps1557.OverlayValues[1310] = d1310
		ps1557.OverlayValues[1311] = d1311
		ps1557.OverlayValues[1312] = d1312
		ps1557.OverlayValues[1313] = d1313
		ps1557.OverlayValues[1314] = d1314
		ps1557.OverlayValues[1315] = d1315
		ps1557.OverlayValues[1316] = d1316
		ps1557.OverlayValues[1317] = d1317
		ps1557.OverlayValues[1318] = d1318
		ps1557.OverlayValues[1319] = d1319
		ps1557.OverlayValues[1320] = d1320
		snap1558 := d1
		snap1559 := d2
		snap1560 := d3
		snap1561 := d4
		snap1562 := d5
		snap1563 := d6
		snap1564 := d7
		snap1565 := d8
		snap1566 := d9
		snap1567 := d10
		snap1568 := d11
		snap1569 := d12
		snap1570 := d13
		snap1571 := d14
		snap1572 := d15
		snap1573 := d17
		snap1574 := d18
		snap1575 := d19
		snap1576 := d20
		snap1577 := d21
		snap1578 := d22
		snap1579 := d23
		snap1580 := d24
		snap1581 := d25
		snap1582 := d26
		snap1583 := d27
		snap1584 := d28
		snap1585 := d29
		snap1586 := d30
		snap1587 := d31
		snap1588 := d32
		snap1589 := d33
		snap1590 := d34
		snap1591 := d35
		snap1592 := d36
		snap1593 := d37
		snap1594 := d38
		snap1595 := d39
		snap1596 := d40
		snap1597 := d41
		snap1598 := d42
		snap1599 := d43
		snap1600 := d44
		snap1601 := d45
		snap1602 := d46
		snap1603 := d47
		snap1604 := d48
		snap1605 := d49
		snap1606 := d50
		snap1607 := d53
		snap1608 := d54
		snap1609 := d55
		snap1610 := d164
		snap1611 := d165
		snap1612 := d166
		snap1613 := d167
		snap1614 := d168
		snap1615 := d169
		snap1616 := d170
		snap1617 := d171
		snap1618 := d172
		snap1619 := d173
		snap1620 := d174
		snap1621 := d175
		snap1622 := d176
		snap1623 := d177
		snap1624 := d178
		snap1625 := d179
		snap1626 := d180
		snap1627 := d181
		snap1628 := d182
		snap1629 := d183
		snap1630 := d184
		snap1631 := d185
		snap1632 := d186
		snap1633 := d187
		snap1634 := d188
		snap1635 := d189
		snap1636 := d190
		snap1637 := d191
		snap1638 := d192
		snap1639 := d193
		snap1640 := d196
		snap1641 := d367
		snap1642 := d368
		snap1643 := d369
		snap1644 := d370
		snap1645 := d372
		snap1646 := d373
		snap1647 := d374
		snap1648 := d375
		snap1649 := d376
		snap1650 := d377
		snap1651 := d378
		snap1652 := d379
		snap1653 := d381
		snap1654 := d383
		snap1655 := d384
		snap1656 := d385
		snap1657 := d486
		snap1658 := d487
		snap1659 := d490
		snap1660 := d594
		snap1661 := d595
		snap1662 := d596
		snap1663 := d597
		snap1664 := d598
		snap1665 := d600
		snap1666 := d601
		snap1667 := d602
		snap1668 := d603
		snap1669 := d604
		snap1670 := d605
		snap1671 := d606
		snap1672 := d607
		snap1673 := d608
		snap1674 := d609
		snap1675 := d610
		snap1676 := d611
		snap1677 := d612
		snap1678 := d613
		snap1679 := d614
		snap1680 := d615
		snap1681 := d616
		snap1682 := d617
		snap1683 := d618
		snap1684 := d619
		snap1685 := d620
		snap1686 := d621
		snap1687 := d622
		snap1688 := d623
		snap1689 := d624
		snap1690 := d625
		snap1691 := d626
		snap1692 := d627
		snap1693 := d628
		snap1694 := d629
		snap1695 := d630
		snap1696 := d913
		snap1697 := d914
		snap1698 := d915
		snap1699 := d917
		snap1700 := d918
		snap1701 := d919
		snap1702 := d920
		snap1703 := d921
		snap1704 := d922
		snap1705 := d923
		snap1706 := d925
		snap1707 := d927
		snap1708 := d928
		snap1709 := d1081
		snap1710 := d1082
		snap1711 := d1085
		snap1712 := d1241
		snap1713 := d1242
		snap1714 := d1243
		snap1715 := d1244
		snap1716 := d1246
		snap1717 := d1247
		snap1718 := d1248
		snap1719 := d1249
		snap1720 := d1250
		snap1721 := d1251
		snap1722 := d1252
		snap1723 := d1253
		snap1724 := d1255
		snap1725 := d1256
		snap1726 := d1257
		snap1727 := d1258
		snap1728 := d1259
		snap1729 := d1260
		snap1730 := d1261
		snap1731 := d1262
		snap1732 := d1263
		snap1733 := d1264
		snap1734 := d1265
		snap1735 := d1266
		snap1736 := d1267
		snap1737 := d1268
		snap1738 := d1269
		snap1739 := d1270
		snap1740 := d1271
		snap1741 := d1272
		snap1742 := d1273
		snap1743 := d1274
		snap1744 := d1275
		snap1745 := d1276
		snap1746 := d1277
		snap1747 := d1278
		snap1748 := d1279
		snap1749 := d1280
		snap1750 := d1281
		snap1751 := d1282
		snap1752 := d1283
		snap1753 := d1284
		snap1754 := d1285
		snap1755 := d1286
		snap1756 := d1287
		snap1757 := d1288
		snap1758 := d1289
		snap1759 := d1290
		snap1760 := d1291
		snap1761 := d1292
		snap1762 := d1293
		snap1763 := d1294
		snap1764 := d1295
		snap1765 := d1296
		snap1766 := d1297
		snap1767 := d1298
		snap1768 := d1299
		snap1769 := d1300
		snap1770 := d1301
		snap1771 := d1302
		snap1772 := d1303
		snap1773 := d1304
		snap1774 := d1305
		snap1775 := d1306
		snap1776 := d1307
		snap1777 := d1308
		snap1778 := d1309
		snap1779 := d1310
		snap1780 := d1311
		snap1781 := d1312
		snap1782 := d1313
		snap1783 := d1314
		snap1784 := d1315
		snap1785 := d1316
		snap1786 := d1317
		snap1787 := d1318
		snap1788 := d1319
		snap1789 := d1320
		alloc1790 := ctx.SnapshotAllocState()
		if !bbs[12].Rendered {
			bbs[12].RenderPS(ps1557)
		}
		ctx.RestoreAllocState(alloc1790)
		d1 = snap1558
		d2 = snap1559
		d3 = snap1560
		d4 = snap1561
		d5 = snap1562
		d6 = snap1563
		d7 = snap1564
		d8 = snap1565
		d9 = snap1566
		d10 = snap1567
		d11 = snap1568
		d12 = snap1569
		d13 = snap1570
		d14 = snap1571
		d15 = snap1572
		d17 = snap1573
		d18 = snap1574
		d19 = snap1575
		d20 = snap1576
		d21 = snap1577
		d22 = snap1578
		d23 = snap1579
		d24 = snap1580
		d25 = snap1581
		d26 = snap1582
		d27 = snap1583
		d28 = snap1584
		d29 = snap1585
		d30 = snap1586
		d31 = snap1587
		d32 = snap1588
		d33 = snap1589
		d34 = snap1590
		d35 = snap1591
		d36 = snap1592
		d37 = snap1593
		d38 = snap1594
		d39 = snap1595
		d40 = snap1596
		d41 = snap1597
		d42 = snap1598
		d43 = snap1599
		d44 = snap1600
		d45 = snap1601
		d46 = snap1602
		d47 = snap1603
		d48 = snap1604
		d49 = snap1605
		d50 = snap1606
		d53 = snap1607
		d54 = snap1608
		d55 = snap1609
		d164 = snap1610
		d165 = snap1611
		d166 = snap1612
		d167 = snap1613
		d168 = snap1614
		d169 = snap1615
		d170 = snap1616
		d171 = snap1617
		d172 = snap1618
		d173 = snap1619
		d174 = snap1620
		d175 = snap1621
		d176 = snap1622
		d177 = snap1623
		d178 = snap1624
		d179 = snap1625
		d180 = snap1626
		d181 = snap1627
		d182 = snap1628
		d183 = snap1629
		d184 = snap1630
		d185 = snap1631
		d186 = snap1632
		d187 = snap1633
		d188 = snap1634
		d189 = snap1635
		d190 = snap1636
		d191 = snap1637
		d192 = snap1638
		d193 = snap1639
		d196 = snap1640
		d367 = snap1641
		d368 = snap1642
		d369 = snap1643
		d370 = snap1644
		d372 = snap1645
		d373 = snap1646
		d374 = snap1647
		d375 = snap1648
		d376 = snap1649
		d377 = snap1650
		d378 = snap1651
		d379 = snap1652
		d381 = snap1653
		d383 = snap1654
		d384 = snap1655
		d385 = snap1656
		d486 = snap1657
		d487 = snap1658
		d490 = snap1659
		d594 = snap1660
		d595 = snap1661
		d596 = snap1662
		d597 = snap1663
		d598 = snap1664
		d600 = snap1665
		d601 = snap1666
		d602 = snap1667
		d603 = snap1668
		d604 = snap1669
		d605 = snap1670
		d606 = snap1671
		d607 = snap1672
		d608 = snap1673
		d609 = snap1674
		d610 = snap1675
		d611 = snap1676
		d612 = snap1677
		d613 = snap1678
		d614 = snap1679
		d615 = snap1680
		d616 = snap1681
		d617 = snap1682
		d618 = snap1683
		d619 = snap1684
		d620 = snap1685
		d621 = snap1686
		d622 = snap1687
		d623 = snap1688
		d624 = snap1689
		d625 = snap1690
		d626 = snap1691
		d627 = snap1692
		d628 = snap1693
		d629 = snap1694
		d630 = snap1695
		d913 = snap1696
		d914 = snap1697
		d915 = snap1698
		d917 = snap1699
		d918 = snap1700
		d919 = snap1701
		d920 = snap1702
		d921 = snap1703
		d922 = snap1704
		d923 = snap1705
		d925 = snap1706
		d927 = snap1707
		d928 = snap1708
		d1081 = snap1709
		d1082 = snap1710
		d1085 = snap1711
		d1241 = snap1712
		d1242 = snap1713
		d1243 = snap1714
		d1244 = snap1715
		d1246 = snap1716
		d1247 = snap1717
		d1248 = snap1718
		d1249 = snap1719
		d1250 = snap1720
		d1251 = snap1721
		d1252 = snap1722
		d1253 = snap1723
		d1255 = snap1724
		d1256 = snap1725
		d1257 = snap1726
		d1258 = snap1727
		d1259 = snap1728
		d1260 = snap1729
		d1261 = snap1730
		d1262 = snap1731
		d1263 = snap1732
		d1264 = snap1733
		d1265 = snap1734
		d1266 = snap1735
		d1267 = snap1736
		d1268 = snap1737
		d1269 = snap1738
		d1270 = snap1739
		d1271 = snap1740
		d1272 = snap1741
		d1273 = snap1742
		d1274 = snap1743
		d1275 = snap1744
		d1276 = snap1745
		d1277 = snap1746
		d1278 = snap1747
		d1279 = snap1748
		d1280 = snap1749
		d1281 = snap1750
		d1282 = snap1751
		d1283 = snap1752
		d1284 = snap1753
		d1285 = snap1754
		d1286 = snap1755
		d1287 = snap1756
		d1288 = snap1757
		d1289 = snap1758
		d1290 = snap1759
		d1291 = snap1760
		d1292 = snap1761
		d1293 = snap1762
		d1294 = snap1763
		d1295 = snap1764
		d1296 = snap1765
		d1297 = snap1766
		d1298 = snap1767
		d1299 = snap1768
		d1300 = snap1769
		d1301 = snap1770
		d1302 = snap1771
		d1303 = snap1772
		d1304 = snap1773
		d1305 = snap1774
		d1306 = snap1775
		d1307 = snap1776
		d1308 = snap1777
		d1309 = snap1778
		d1310 = snap1779
		d1311 = snap1780
		d1312 = snap1781
		d1313 = snap1782
		d1314 = snap1783
		d1315 = snap1784
		d1316 = snap1785
		d1317 = snap1786
		d1318 = snap1787
		d1319 = snap1788
		d1320 = snap1789
		if !bbs[11].Rendered {
			return bbs[11].RenderPS(ps1556)
		}
		return result
		ctx.FreeDesc(&d1319)
		return result
	}
	ps1791 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps1791)
	ctx.MarkLabel(lbl0)
	d1792 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
	ctx.BindReg(r0, &d1792)
	ctx.BindReg(r1, &d1792)
	ctx.EmitMovPairToResult(&d1792, &result)
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
