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
	var d187 scm.JITValueDesc
	_ = d187
	var d344 scm.JITValueDesc
	_ = d344
	var d345 scm.JITValueDesc
	_ = d345
	var d346 scm.JITValueDesc
	_ = d346
	var d347 scm.JITValueDesc
	_ = d347
	var d349 scm.JITValueDesc
	_ = d349
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
	var d355 scm.JITValueDesc
	_ = d355
	var d356 scm.JITValueDesc
	_ = d356
	var d358 scm.JITValueDesc
	_ = d358
	var d360 scm.JITValueDesc
	_ = d360
	var d361 scm.JITValueDesc
	_ = d361
	var d362 scm.JITValueDesc
	_ = d362
	var d456 scm.JITValueDesc
	_ = d456
	var d457 scm.JITValueDesc
	_ = d457
	var d460 scm.JITValueDesc
	_ = d460
	var d557 scm.JITValueDesc
	_ = d557
	var d558 scm.JITValueDesc
	_ = d558
	var d559 scm.JITValueDesc
	_ = d559
	var d560 scm.JITValueDesc
	_ = d560
	var d561 scm.JITValueDesc
	_ = d561
	var d563 scm.JITValueDesc
	_ = d563
	var d564 scm.JITValueDesc
	_ = d564
	var d565 scm.JITValueDesc
	_ = d565
	var d566 scm.JITValueDesc
	_ = d566
	var d567 scm.JITValueDesc
	_ = d567
	var d568 scm.JITValueDesc
	_ = d568
	var d569 scm.JITValueDesc
	_ = d569
	var d570 scm.JITValueDesc
	_ = d570
	var d571 scm.JITValueDesc
	_ = d571
	var d572 scm.JITValueDesc
	_ = d572
	var d573 scm.JITValueDesc
	_ = d573
	var d574 scm.JITValueDesc
	_ = d574
	var d575 scm.JITValueDesc
	_ = d575
	var d576 scm.JITValueDesc
	_ = d576
	var d577 scm.JITValueDesc
	_ = d577
	var d578 scm.JITValueDesc
	_ = d578
	var d579 scm.JITValueDesc
	_ = d579
	var d580 scm.JITValueDesc
	_ = d580
	var d581 scm.JITValueDesc
	_ = d581
	var d582 scm.JITValueDesc
	_ = d582
	var d583 scm.JITValueDesc
	_ = d583
	var d584 scm.JITValueDesc
	_ = d584
	var d585 scm.JITValueDesc
	_ = d585
	var d586 scm.JITValueDesc
	_ = d586
	var d587 scm.JITValueDesc
	_ = d587
	var d588 scm.JITValueDesc
	_ = d588
	var d589 scm.JITValueDesc
	_ = d589
	var d850 scm.JITValueDesc
	_ = d850
	var d851 scm.JITValueDesc
	_ = d851
	var d852 scm.JITValueDesc
	_ = d852
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
	var d862 scm.JITValueDesc
	_ = d862
	var d864 scm.JITValueDesc
	_ = d864
	var d865 scm.JITValueDesc
	_ = d865
	var d1007 scm.JITValueDesc
	_ = d1007
	var d1008 scm.JITValueDesc
	_ = d1008
	var d1011 scm.JITValueDesc
	_ = d1011
	var d1156 scm.JITValueDesc
	_ = d1156
	var d1157 scm.JITValueDesc
	_ = d1157
	var d1158 scm.JITValueDesc
	_ = d1158
	var d1159 scm.JITValueDesc
	_ = d1159
	var d1161 scm.JITValueDesc
	_ = d1161
	var d1162 scm.JITValueDesc
	_ = d1162
	var d1163 scm.JITValueDesc
	_ = d1163
	var d1164 scm.JITValueDesc
	_ = d1164
	var d1165 scm.JITValueDesc
	_ = d1165
	var d1166 scm.JITValueDesc
	_ = d1166
	var d1167 scm.JITValueDesc
	_ = d1167
	var d1168 scm.JITValueDesc
	_ = d1168
	var d1169 scm.JITValueDesc
	_ = d1169
	var d1170 scm.JITValueDesc
	_ = d1170
	var d1172 scm.JITValueDesc
	_ = d1172
	var d1173 scm.JITValueDesc
	_ = d1173
	var d1174 scm.JITValueDesc
	_ = d1174
	var d1175 scm.JITValueDesc
	_ = d1175
	var d1176 scm.JITValueDesc
	_ = d1176
	var d1177 scm.JITValueDesc
	_ = d1177
	var d1178 scm.JITValueDesc
	_ = d1178
	var d1179 scm.JITValueDesc
	_ = d1179
	var d1180 scm.JITValueDesc
	_ = d1180
	var d1181 scm.JITValueDesc
	_ = d1181
	var d1182 scm.JITValueDesc
	_ = d1182
	var d1183 scm.JITValueDesc
	_ = d1183
	var d1184 scm.JITValueDesc
	_ = d1184
	var d1185 scm.JITValueDesc
	_ = d1185
	var d1186 scm.JITValueDesc
	_ = d1186
	var d1187 scm.JITValueDesc
	_ = d1187
	var d1188 scm.JITValueDesc
	_ = d1188
	var d1189 scm.JITValueDesc
	_ = d1189
	var d1190 scm.JITValueDesc
	_ = d1190
	var d1191 scm.JITValueDesc
	_ = d1191
	var d1192 scm.JITValueDesc
	_ = d1192
	var d1193 scm.JITValueDesc
	_ = d1193
	var d1194 scm.JITValueDesc
	_ = d1194
	var d1195 scm.JITValueDesc
	_ = d1195
	var d1196 scm.JITValueDesc
	_ = d1196
	var d1197 scm.JITValueDesc
	_ = d1197
	var d1198 scm.JITValueDesc
	_ = d1198
	var d1199 scm.JITValueDesc
	_ = d1199
	var d1200 scm.JITValueDesc
	_ = d1200
	var d1201 scm.JITValueDesc
	_ = d1201
	var d1202 scm.JITValueDesc
	_ = d1202
	var d1203 scm.JITValueDesc
	_ = d1203
	var d1204 scm.JITValueDesc
	_ = d1204
	var d1205 scm.JITValueDesc
	_ = d1205
	var d1206 scm.JITValueDesc
	_ = d1206
	var d1207 scm.JITValueDesc
	_ = d1207
	var d1208 scm.JITValueDesc
	_ = d1208
	var d1209 scm.JITValueDesc
	_ = d1209
	var d1210 scm.JITValueDesc
	_ = d1210
	var d1211 scm.JITValueDesc
	_ = d1211
	var d1212 scm.JITValueDesc
	_ = d1212
	var d1213 scm.JITValueDesc
	_ = d1213
	var d1214 scm.JITValueDesc
	_ = d1214
	var d1215 scm.JITValueDesc
	_ = d1215
	var d1216 scm.JITValueDesc
	_ = d1216
	var d1217 scm.JITValueDesc
	_ = d1217
	var d1218 scm.JITValueDesc
	_ = d1218
	var d1219 scm.JITValueDesc
	_ = d1219
	var d1220 scm.JITValueDesc
	_ = d1220
	var d1221 scm.JITValueDesc
	_ = d1221
	var d1222 scm.JITValueDesc
	_ = d1222
	var d1223 scm.JITValueDesc
	_ = d1223
	var d1224 scm.JITValueDesc
	_ = d1224
	var d1225 scm.JITValueDesc
	_ = d1225
	var d1226 scm.JITValueDesc
	_ = d1226
	var d1227 scm.JITValueDesc
	_ = d1227
	var d1228 scm.JITValueDesc
	_ = d1228
	var d1229 scm.JITValueDesc
	_ = d1229
	var d1230 scm.JITValueDesc
	_ = d1230
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
			ctx.EmitSubRegImm32Low(scratch, int32(1))
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d17)
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
			if phiHomeOK2 {
				ctx.EmitMovToReg(r0, d18)
			} else {
				ctx.EmitStoreToStack(d18, int32(bbs[1].PhiBase)+int32(0))
			}
			if phiHomeOK3 {
				ctx.EmitMovToReg(r1, scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)})
			} else {
				ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(bbs[1].PhiBase)+int32(16))
			}
			d19 = d17
			if d19.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d19)
			if phiHomeOK4 {
				ctx.EmitMovToReg(r2, d19)
			} else {
				ctx.EmitStoreToStack(d19, int32(bbs[1].PhiBase)+int32(32))
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
		ps20 := scm.PhiState{General: ps.General}
		ps20.OverlayValues = make([]scm.JITValueDesc, 20)
		ps20.OverlayValues[5] = d5
		ps20.OverlayValues[6] = d6
		ps20.OverlayValues[7] = d7
		ps20.OverlayValues[8] = d8
		ps20.OverlayValues[9] = d9
		ps20.OverlayValues[10] = d10
		ps20.OverlayValues[11] = d11
		ps20.OverlayValues[12] = d12
		ps20.OverlayValues[13] = d13
		ps20.OverlayValues[14] = d14
		ps20.OverlayValues[15] = d15
		ps20.OverlayValues[16] = d16
		ps20.OverlayValues[17] = d17
		ps20.OverlayValues[18] = d18
		ps20.OverlayValues[19] = d19
		ps20.PhiValues = make([]scm.JITValueDesc, 3)
		d21 = d15
		ps20.PhiValues[0] = d21
		d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
		ps20.PhiValues[1] = d22
		d23 = d17
		ps20.PhiValues[2] = d23
		if ps20.General && bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		return bbs[1].RenderPS(ps20)
		return result
	}
	bbs[1].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d24 := ps.PhiValues[0]
				ctx.EnsureDesc(&d24)
				if phiHomeOK2 {
					ctx.EmitMovToReg(r0, d24)
				} else {
					ctx.EmitStoreToStack(d24, int32(bbs[1].PhiBase)+int32(0))
				}
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d25 := ps.PhiValues[1]
				ctx.EnsureDesc(&d25)
				if phiHomeOK3 {
					ctx.EmitMovToReg(r1, d25)
				} else {
					ctx.EmitStoreToStack(d25, int32(bbs[1].PhiBase)+int32(16))
				}
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d26 := ps.PhiValues[2]
				ctx.EnsureDesc(&d26)
				if phiHomeOK4 {
					ctx.EmitMovToReg(r2, d26)
				} else {
					ctx.EmitStoreToStack(d26, int32(bbs[1].PhiBase)+int32(32))
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
		d27 = d5
		_ = d27
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
		ctx.ReclaimUntrackedRegs()
		var d28 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48)
			r8 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r8, thisptr.Reg, off)
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r8}
			ctx.BindReg(r8, &d28)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d28)
		ctx.EnsureDesc(&d28)
		var d29 scm.JITValueDesc
		if d28.Loc == scm.LocImm {
			d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d28.Imm.Int()))))}
		} else {
			r9 := ctx.AllocReg()
			ctx.EmitMovRegReg(r9, d28.Reg)
			ctx.EmitShlRegImm8(r9, 56)
			ctx.EmitShrRegImm8(r9, 56)
			d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
			ctx.BindReg(r9, &d29)
		}
		ctx.FreeDesc(&d28)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d27)
		ctx.EnsureDesc(&d27)
		var d30 scm.JITValueDesc
		if d27.Loc == scm.LocImm {
			d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d27.Imm.Int()))))}
		} else {
			r10 := ctx.AllocReg()
			ctx.EmitMovRegReg(r10, d27.Reg)
			ctx.EmitShlRegImm8(r10, 32)
			ctx.EmitShrRegImm8(r10, 32)
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
			ctx.BindReg(r10, &d30)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d30)
		ctx.EnsureDesc(&d29)
		ctx.EnsureDescsTogether(&d30, &d29)
		var d31 scm.JITValueDesc
		if d30.Loc == scm.LocImm && d29.Loc == scm.LocImm {
			d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d30.Imm.Int() * d29.Imm.Int())}
		} else if d30.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d29.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d30.Imm.Int()))
			ctx.EmitImulInt64(scratch, d29.Reg)
			d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d31)
		} else if d29.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d30.Reg)
			ctx.EmitMovRegReg(scratch, d30.Reg)
			if d29.Imm.Int() >= -2147483648 && d29.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d29.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d29.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d31)
		} else {
			r11 := ctx.AllocRegExcept(d30.Reg, d29.Reg)
			ctx.EmitMovRegReg(r11, d30.Reg)
			ctx.EmitImulInt64(r11, d29.Reg)
			d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
			ctx.BindReg(r11, &d31)
		}
		if d31.Loc == scm.LocReg && d30.Loc == scm.LocReg && d31.Reg == d30.Reg {
			ctx.TransferReg(d30.Reg)
			d30.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d30)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d31)
		var d32 scm.JITValueDesc
		if d31.Loc == scm.LocImm {
			d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d31.Imm.Int() / 64)}
		} else {
			r12 := ctx.AllocRegExcept(d31.Reg)
			ctx.EmitMovRegReg(r12, d31.Reg)
			ctx.EmitShrRegImm8(r12, 6)
			d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
			ctx.BindReg(r12, &d32)
		}
		if d32.Loc == scm.LocReg && d31.Loc == scm.LocReg && d32.Reg == d31.Reg {
			ctx.TransferReg(d31.Reg)
			d31.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d31)
		var d33 scm.JITValueDesc
		if d31.Loc == scm.LocImm {
			d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d31.Imm.Int() % 64)}
		} else {
			r13 := ctx.AllocRegExcept(d31.Reg)
			ctx.EmitMovRegReg(r13, d31.Reg)
			ctx.EmitAndRegImm32(r13, 63)
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
			ctx.BindReg(r13, &d33)
		}
		if d33.Loc == scm.LocReg && d31.Loc == scm.LocReg && d33.Reg == d31.Reg {
			ctx.TransferReg(d31.Reg)
			d31.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d31)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d34 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d34 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r14 := ctx.AllocReg()
			r15 := ctx.AllocRegExcept(r14)
			r16 := ctx.AllocRegExcept(r14, r15)
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			ctx.EmitMovRegMem(r14, thisptr.Reg, off)
			ctx.EmitMovRegMem(r15, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r16, thisptr.Reg, off+16)
			d34 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r14, Reg2: r15, Reg3: r16}
			ctx.BindReg(r14, &d34)
			ctx.BindReg(r15, &d34)
			ctx.BindReg(r16, &d34)
			ctx.BindReg(r14, &d34)
			ctx.BindReg(r15, &d34)
			ctx.BindReg(r16, &d34)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d32)
		ctx.ReclaimUntrackedRegs()
		d35 = ctx.EmitLoadScalarSliceElement(&d34, &d32, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d35)
		ctx.EnsureDesc(&d33)
		ctx.EnsureDescsTogether(&d35, &d33)
		var d36 scm.JITValueDesc
		if d35.Loc == scm.LocImm && d33.Loc == scm.LocImm {
			d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d35.Imm.Int()) << uint64(d33.Imm.Int())))}
		} else if d33.Loc == scm.LocImm {
			r17 := ctx.AllocRegExcept(d35.Reg)
			ctx.EmitMovRegReg(r17, d35.Reg)
			ctx.EmitShlRegImm8(r17, uint8(d33.Imm.Int()))
			d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			ctx.BindReg(r17, &d36)
		} else {
			{
				shiftSrc := d35.Reg
				r18 := ctx.AllocRegExcept(d35.Reg, d33.Reg)
				ctx.EmitMovRegReg(r18, d35.Reg)
				shiftSrc = r18
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
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d36)
			}
		}
		if d36.Loc == scm.LocReg && d35.Loc == scm.LocReg && d36.Reg == d35.Reg {
			ctx.TransferReg(d35.Reg)
			d35.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d35)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d32)
		ctx.EnsureDesc(&d32)
		var d37 scm.JITValueDesc
		if d32.Loc == scm.LocImm {
			d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d32.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d32.Reg)
			ctx.EmitMovRegReg(scratch, d32.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d37)
		}
		if d37.Loc == scm.LocReg && d32.Loc == scm.LocReg && d37.Reg == d32.Reg {
			ctx.TransferReg(d32.Reg)
			d32.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d32)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d37)
		ctx.ReclaimUntrackedRegs()
		d38 = ctx.EmitLoadScalarSliceElement(&d34, &d37, 8, scm.TagInt)
		ctx.FreeDesc(&d37)
		ctx.ReclaimUntrackedRegs()
		d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d33)
		ctx.EnsureDescsTogether(&d39, &d33)
		var d40 scm.JITValueDesc
		if d39.Loc == scm.LocImm && d33.Loc == scm.LocImm {
			d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() - d33.Imm.Int())}
		} else if d33.Loc == scm.LocImm && d33.Imm.Int() == 0 {
			r19 := ctx.AllocRegExcept(d39.Reg)
			ctx.EmitMovRegReg(r19, d39.Reg)
			d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d40)
		} else if d39.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d33.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d39.Imm.Int()))
			ctx.EmitSubInt64(scratch, d33.Reg)
			d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d40)
		} else if d33.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d39.Reg)
			ctx.EmitMovRegReg(scratch, d39.Reg)
			if d33.Imm.Int() >= -2147483648 && d33.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d33.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d33.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d40)
		} else {
			r20 := ctx.AllocRegExcept(d39.Reg, d33.Reg)
			ctx.EmitMovRegReg(r20, d39.Reg)
			ctx.EmitSubInt64(r20, d33.Reg)
			d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
			ctx.BindReg(r20, &d40)
		}
		if d40.Loc == scm.LocReg && d39.Loc == scm.LocReg && d40.Reg == d39.Reg {
			ctx.TransferReg(d39.Reg)
			d39.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d33)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d38)
		ctx.EnsureDesc(&d40)
		ctx.EnsureDescsTogether(&d38, &d40)
		var d41 scm.JITValueDesc
		if d38.Loc == scm.LocImm && d40.Loc == scm.LocImm {
			d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d38.Imm.Int()) >> uint64(d40.Imm.Int())))}
		} else if d40.Loc == scm.LocImm {
			r21 := ctx.AllocRegExcept(d38.Reg)
			ctx.EmitMovRegReg(r21, d38.Reg)
			ctx.EmitShrRegImm8(r21, uint8(d40.Imm.Int()))
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			ctx.BindReg(r21, &d41)
		} else {
			{
				shiftSrc := d38.Reg
				r22 := ctx.AllocRegExcept(d38.Reg, d40.Reg)
				ctx.EmitMovRegReg(r22, d38.Reg)
				shiftSrc = r22
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d40.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d40.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d40.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d41)
			}
		}
		if d41.Loc == scm.LocReg && d38.Loc == scm.LocReg && d41.Reg == d38.Reg {
			ctx.TransferReg(d38.Reg)
			d38.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d38)
		ctx.FreeDesc(&d40)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d36)
		ctx.EnsureDesc(&d41)
		var d42 scm.JITValueDesc
		if d36.Loc == scm.LocImm && d41.Loc == scm.LocImm {
			d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d36.Imm.Int() | d41.Imm.Int())}
		} else if d36.Loc == scm.LocImm && d36.Imm.Int() == 0 {
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d41.Reg}
			ctx.BindReg(d41.Reg, &d42)
		} else if d41.Loc == scm.LocImm && d41.Imm.Int() == 0 {
			r23 := ctx.AllocRegExcept(d36.Reg)
			ctx.EmitMovRegReg(r23, d36.Reg)
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
			ctx.BindReg(r23, &d42)
		} else if d36.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d41.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d36.Imm.Int()))
			ctx.EmitOrInt64(scratch, d41.Reg)
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d42)
		} else if d41.Loc == scm.LocImm {
			r24 := ctx.AllocRegExcept(d36.Reg)
			ctx.EmitMovRegReg(r24, d36.Reg)
			if d41.Imm.Int() >= -2147483648 && d41.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r24, int32(d41.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d41.Imm.Int()))
				ctx.EmitOrInt64(r24, scm.RegR11)
			}
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d42)
		} else {
			r25 := ctx.AllocRegExcept(d36.Reg, d41.Reg)
			ctx.EmitMovRegReg(r25, d36.Reg)
			ctx.EmitOrInt64(r25, d41.Reg)
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
			ctx.BindReg(r25, &d42)
		}
		if d42.Loc == scm.LocReg && d36.Loc == scm.LocReg && d42.Reg == d36.Reg {
			ctx.TransferReg(d36.Reg)
			d36.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d36)
		ctx.FreeDesc(&d41)
		ctx.ReclaimUntrackedRegs()
		d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d29)
		ctx.EnsureDescsTogether(&d43, &d29)
		var d44 scm.JITValueDesc
		if d43.Loc == scm.LocImm && d29.Loc == scm.LocImm {
			d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d43.Imm.Int() - d29.Imm.Int())}
		} else if d29.Loc == scm.LocImm && d29.Imm.Int() == 0 {
			r26 := ctx.AllocRegExcept(d43.Reg)
			ctx.EmitMovRegReg(r26, d43.Reg)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
			ctx.BindReg(r26, &d44)
		} else if d43.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d29.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d43.Imm.Int()))
			ctx.EmitSubInt64(scratch, d29.Reg)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d44)
		} else if d29.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d43.Reg)
			ctx.EmitMovRegReg(scratch, d43.Reg)
			if d29.Imm.Int() >= -2147483648 && d29.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d29.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d29.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d44)
		} else {
			r27 := ctx.AllocRegExcept(d43.Reg, d29.Reg)
			ctx.EmitMovRegReg(r27, d43.Reg)
			ctx.EmitSubInt64(r27, d29.Reg)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
			ctx.BindReg(r27, &d44)
		}
		if d44.Loc == scm.LocReg && d43.Loc == scm.LocReg && d44.Reg == d43.Reg {
			ctx.TransferReg(d43.Reg)
			d43.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d29)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d42)
		ctx.EnsureDesc(&d44)
		ctx.EnsureDescsTogether(&d42, &d44)
		var d45 scm.JITValueDesc
		if d42.Loc == scm.LocImm && d44.Loc == scm.LocImm {
			d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d42.Imm.Int()) >> uint64(d44.Imm.Int())))}
		} else if d44.Loc == scm.LocImm {
			r28 := ctx.AllocRegExcept(d42.Reg)
			ctx.EmitMovRegReg(r28, d42.Reg)
			ctx.EmitShrRegImm8(r28, uint8(d44.Imm.Int()))
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
			ctx.BindReg(r28, &d45)
		} else {
			{
				shiftSrc := d42.Reg
				r29 := ctx.AllocRegExcept(d42.Reg, d44.Reg)
				ctx.EmitMovRegReg(r29, d42.Reg)
				shiftSrc = r29
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
		if d45.Loc == scm.LocReg && d42.Loc == scm.LocReg && d45.Reg == d42.Reg {
			ctx.TransferReg(d42.Reg)
			d42.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d42)
		ctx.FreeDesc(&d44)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d45)
		ctx.EnsureDesc(&d45)
		ctx.EnsureDesc(&d45)
		var d46 scm.JITValueDesc
		if d45.Loc == scm.LocImm {
			d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d45.Imm.Int()))))}
		} else {
			r30 := ctx.AllocReg()
			ctx.EmitMovRegReg(r30, d45.Reg)
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
			ctx.BindReg(r30, &d46)
		}
		ctx.FreeDesc(&d45)
		var d47 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56)
			r31 := ctx.AllocReg()
			ctx.EmitMovRegMem(r31, thisptr.Reg, off)
			d47 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r31}
			ctx.BindReg(r31, &d47)
		}
		ctx.EnsureDesc(&d46)
		ctx.EnsureDesc(&d47)
		ctx.EnsureDescsTogether(&d46, &d47)
		var d48 scm.JITValueDesc
		if d46.Loc == scm.LocImm && d47.Loc == scm.LocImm {
			d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d46.Imm.Int() + d47.Imm.Int())}
		} else if d47.Loc == scm.LocImm && d47.Imm.Int() == 0 {
			r32 := ctx.AllocRegExcept(d46.Reg)
			ctx.EmitMovRegReg(r32, d46.Reg)
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
			ctx.BindReg(r32, &d48)
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
			r33 := ctx.AllocRegExcept(d46.Reg, d47.Reg)
			ctx.EmitMovRegReg(r33, d46.Reg)
			ctx.EmitAddInt64(r33, d47.Reg)
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
			ctx.BindReg(r33, &d48)
		}
		if d48.Loc == scm.LocReg && d46.Loc == scm.LocReg && d48.Reg == d46.Reg {
			ctx.TransferReg(d46.Reg)
			d46.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d46)
		ctx.FreeDesc(&d47)
		ctx.EnsureDesc(&d48)
		ctx.EnsureDesc(&d48)
		var d49 scm.JITValueDesc
		if d48.Loc == scm.LocImm {
			d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d48.Imm.Int()))))}
		} else {
			r34 := ctx.AllocReg()
			ctx.EmitMovRegReg(r34, d48.Reg)
			ctx.EmitShlRegImm8(r34, 32)
			ctx.EmitShrRegImm8(r34, 32)
			d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
			ctx.BindReg(r34, &d49)
		}
		ctx.FreeDesc(&d48)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d49)
		ctx.EnsureDescsTogether(&idxInt, &d49)
		var d50 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm && d49.Loc == scm.LocImm {
			d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d49.Imm.Int()))}
		} else if d49.Loc == scm.LocImm {
			r35 := ctx.AllocRegExcept(idxInt.Reg)
			if d49.Imm.Int() >= -2147483648 && d49.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(idxInt.Reg, int32(d49.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d49.Imm.Int()))
				ctx.EmitCmpInt64(idxInt.Reg, scm.RegR11)
			}
			d50 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r35, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r35, &d50)
		} else if idxInt.Loc == scm.LocImm {
			r36 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d49.Reg)
			d50 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r36, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r36, &d50)
		} else {
			r37 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitCmpInt64(idxInt.Reg, d49.Reg)
			d50 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r37, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r37, &d50)
		}
		ctx.FreeDesc(&d49)
		d51 = d50
		ctx.EnsureDesc(&d51)
		if d51.Loc != scm.LocImm && d51.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d51.Loc == scm.LocImm {
			if d51.Imm.Bool() {
				if ps.General {
				}
				ps52 := scm.PhiState{General: ps.General}
				ps52.OverlayValues = make([]scm.JITValueDesc, 52)
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
				ps52.OverlayValues[18] = d18
				ps52.OverlayValues[19] = d19
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
			ps53.OverlayValues[18] = d18
			ps53.OverlayValues[19] = d19
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
				if phiHomeOK2 {
					ctx.EmitMovToReg(r0, d54)
				} else {
					ctx.EmitStoreToStack(d54, int32(bbs[1].PhiBase)+int32(0))
				}
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d55 := ps.PhiValues[1]
				ctx.EnsureDesc(&d55)
				if phiHomeOK3 {
					ctx.EmitMovToReg(r1, d55)
				} else {
					ctx.EmitStoreToStack(d55, int32(bbs[1].PhiBase)+int32(16))
				}
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d56 := ps.PhiValues[2]
				ctx.EnsureDesc(&d56)
				if phiHomeOK4 {
					ctx.EmitMovToReg(r2, d56)
				} else {
					ctx.EmitStoreToStack(d56, int32(bbs[1].PhiBase)+int32(32))
				}
			}
			ps.General = true
			return bbs[1].RenderPS(ps)
		}
		ctx.EmitJump(d51.Condition, lbl4)
		ctx.FreeDesc(&d50)
		snap57 := d5
		snap58 := d6
		snap59 := d7
		snap60 := d8
		snap61 := d9
		snap62 := d10
		snap63 := d11
		snap64 := d12
		snap65 := d13
		snap66 := d14
		snap67 := d15
		snap68 := d16
		snap69 := d17
		snap70 := d18
		snap71 := d19
		snap72 := d21
		snap73 := d22
		snap74 := d23
		snap75 := d24
		snap76 := d25
		snap77 := d26
		snap78 := d27
		snap79 := d28
		snap80 := d29
		snap81 := d30
		snap82 := d31
		snap83 := d32
		snap84 := d33
		snap85 := d34
		snap86 := d35
		snap87 := d36
		snap88 := d37
		snap89 := d38
		snap90 := d39
		snap91 := d40
		snap92 := d41
		snap93 := d42
		snap94 := d43
		snap95 := d44
		snap96 := d45
		snap97 := d46
		snap98 := d47
		snap99 := d48
		snap100 := d49
		snap101 := d50
		snap102 := d51
		snap103 := d54
		snap104 := d55
		snap105 := d56
		alloc106 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc106)
		d5 = snap57
		d6 = snap58
		d7 = snap59
		d8 = snap60
		d9 = snap61
		d10 = snap62
		d11 = snap63
		d12 = snap64
		d13 = snap65
		d14 = snap66
		d15 = snap67
		d16 = snap68
		d17 = snap69
		d18 = snap70
		d19 = snap71
		d21 = snap72
		d22 = snap73
		d23 = snap74
		d24 = snap75
		d25 = snap76
		d26 = snap77
		d27 = snap78
		d28 = snap79
		d29 = snap80
		d30 = snap81
		d31 = snap82
		d32 = snap83
		d33 = snap84
		d34 = snap85
		d35 = snap86
		d36 = snap87
		d37 = snap88
		d38 = snap89
		d39 = snap90
		d40 = snap91
		d41 = snap92
		d42 = snap93
		d43 = snap94
		d44 = snap95
		d45 = snap96
		d46 = snap97
		d47 = snap98
		d48 = snap99
		d49 = snap100
		d50 = snap101
		d51 = snap102
		d54 = snap103
		d55 = snap104
		d56 = snap105
		ctx.RestoreAllocState(alloc106)
		d5 = snap57
		d6 = snap58
		d7 = snap59
		d8 = snap60
		d9 = snap61
		d10 = snap62
		d11 = snap63
		d12 = snap64
		d13 = snap65
		d14 = snap66
		d15 = snap67
		d16 = snap68
		d17 = snap69
		d18 = snap70
		d19 = snap71
		d21 = snap72
		d22 = snap73
		d23 = snap74
		d24 = snap75
		d25 = snap76
		d26 = snap77
		d27 = snap78
		d28 = snap79
		d29 = snap80
		d30 = snap81
		d31 = snap82
		d32 = snap83
		d33 = snap84
		d34 = snap85
		d35 = snap86
		d36 = snap87
		d37 = snap88
		d38 = snap89
		d39 = snap90
		d40 = snap91
		d41 = snap92
		d42 = snap93
		d43 = snap94
		d44 = snap95
		d45 = snap96
		d46 = snap97
		d47 = snap98
		d48 = snap99
		d49 = snap100
		d50 = snap101
		d51 = snap102
		d54 = snap103
		d55 = snap104
		d56 = snap105
		ps107 := scm.PhiState{General: true}
		ps107.OverlayValues = make([]scm.JITValueDesc, 57)
		ps107.OverlayValues[5] = d5
		ps107.OverlayValues[6] = d6
		ps107.OverlayValues[7] = d7
		ps107.OverlayValues[8] = d8
		ps107.OverlayValues[9] = d9
		ps107.OverlayValues[10] = d10
		ps107.OverlayValues[11] = d11
		ps107.OverlayValues[12] = d12
		ps107.OverlayValues[13] = d13
		ps107.OverlayValues[14] = d14
		ps107.OverlayValues[15] = d15
		ps107.OverlayValues[16] = d16
		ps107.OverlayValues[17] = d17
		ps107.OverlayValues[18] = d18
		ps107.OverlayValues[19] = d19
		ps107.OverlayValues[21] = d21
		ps107.OverlayValues[22] = d22
		ps107.OverlayValues[23] = d23
		ps107.OverlayValues[24] = d24
		ps107.OverlayValues[25] = d25
		ps107.OverlayValues[26] = d26
		ps107.OverlayValues[27] = d27
		ps107.OverlayValues[28] = d28
		ps107.OverlayValues[29] = d29
		ps107.OverlayValues[30] = d30
		ps107.OverlayValues[31] = d31
		ps107.OverlayValues[32] = d32
		ps107.OverlayValues[33] = d33
		ps107.OverlayValues[34] = d34
		ps107.OverlayValues[35] = d35
		ps107.OverlayValues[36] = d36
		ps107.OverlayValues[37] = d37
		ps107.OverlayValues[38] = d38
		ps107.OverlayValues[39] = d39
		ps107.OverlayValues[40] = d40
		ps107.OverlayValues[41] = d41
		ps107.OverlayValues[42] = d42
		ps107.OverlayValues[43] = d43
		ps107.OverlayValues[44] = d44
		ps107.OverlayValues[45] = d45
		ps107.OverlayValues[46] = d46
		ps107.OverlayValues[47] = d47
		ps107.OverlayValues[48] = d48
		ps107.OverlayValues[49] = d49
		ps107.OverlayValues[50] = d50
		ps107.OverlayValues[51] = d51
		ps107.OverlayValues[54] = d54
		ps107.OverlayValues[55] = d55
		ps107.OverlayValues[56] = d56
		ps108 := scm.PhiState{General: true}
		ps108.OverlayValues = make([]scm.JITValueDesc, 57)
		ps108.OverlayValues[5] = d5
		ps108.OverlayValues[6] = d6
		ps108.OverlayValues[7] = d7
		ps108.OverlayValues[8] = d8
		ps108.OverlayValues[9] = d9
		ps108.OverlayValues[10] = d10
		ps108.OverlayValues[11] = d11
		ps108.OverlayValues[12] = d12
		ps108.OverlayValues[13] = d13
		ps108.OverlayValues[14] = d14
		ps108.OverlayValues[15] = d15
		ps108.OverlayValues[16] = d16
		ps108.OverlayValues[17] = d17
		ps108.OverlayValues[18] = d18
		ps108.OverlayValues[19] = d19
		ps108.OverlayValues[21] = d21
		ps108.OverlayValues[22] = d22
		ps108.OverlayValues[23] = d23
		ps108.OverlayValues[24] = d24
		ps108.OverlayValues[25] = d25
		ps108.OverlayValues[26] = d26
		ps108.OverlayValues[27] = d27
		ps108.OverlayValues[28] = d28
		ps108.OverlayValues[29] = d29
		ps108.OverlayValues[30] = d30
		ps108.OverlayValues[31] = d31
		ps108.OverlayValues[32] = d32
		ps108.OverlayValues[33] = d33
		ps108.OverlayValues[34] = d34
		ps108.OverlayValues[35] = d35
		ps108.OverlayValues[36] = d36
		ps108.OverlayValues[37] = d37
		ps108.OverlayValues[38] = d38
		ps108.OverlayValues[39] = d39
		ps108.OverlayValues[40] = d40
		ps108.OverlayValues[41] = d41
		ps108.OverlayValues[42] = d42
		ps108.OverlayValues[43] = d43
		ps108.OverlayValues[44] = d44
		ps108.OverlayValues[45] = d45
		ps108.OverlayValues[46] = d46
		ps108.OverlayValues[47] = d47
		ps108.OverlayValues[48] = d48
		ps108.OverlayValues[49] = d49
		ps108.OverlayValues[50] = d50
		ps108.OverlayValues[51] = d51
		ps108.OverlayValues[54] = d54
		ps108.OverlayValues[55] = d55
		ps108.OverlayValues[56] = d56
		snap109 := d5
		snap110 := d6
		snap111 := d7
		snap112 := d8
		snap113 := d9
		snap114 := d10
		snap115 := d11
		snap116 := d12
		snap117 := d13
		snap118 := d14
		snap119 := d15
		snap120 := d16
		snap121 := d17
		snap122 := d18
		snap123 := d19
		snap124 := d21
		snap125 := d22
		snap126 := d23
		snap127 := d24
		snap128 := d25
		snap129 := d26
		snap130 := d27
		snap131 := d28
		snap132 := d29
		snap133 := d30
		snap134 := d31
		snap135 := d32
		snap136 := d33
		snap137 := d34
		snap138 := d35
		snap139 := d36
		snap140 := d37
		snap141 := d38
		snap142 := d39
		snap143 := d40
		snap144 := d41
		snap145 := d42
		snap146 := d43
		snap147 := d44
		snap148 := d45
		snap149 := d46
		snap150 := d47
		snap151 := d48
		snap152 := d49
		snap153 := d50
		snap154 := d51
		snap155 := d54
		snap156 := d55
		snap157 := d56
		alloc158 := ctx.SnapshotAllocState()
		if !bbs[5].Rendered {
			bbs[5].RenderPS(ps108)
		}
		ctx.RestoreAllocState(alloc158)
		d5 = snap109
		d6 = snap110
		d7 = snap111
		d8 = snap112
		d9 = snap113
		d10 = snap114
		d11 = snap115
		d12 = snap116
		d13 = snap117
		d14 = snap118
		d15 = snap119
		d16 = snap120
		d17 = snap121
		d18 = snap122
		d19 = snap123
		d21 = snap124
		d22 = snap125
		d23 = snap126
		d24 = snap127
		d25 = snap128
		d26 = snap129
		d27 = snap130
		d28 = snap131
		d29 = snap132
		d30 = snap133
		d31 = snap134
		d32 = snap135
		d33 = snap136
		d34 = snap137
		d35 = snap138
		d36 = snap139
		d37 = snap140
		d38 = snap141
		d39 = snap142
		d40 = snap143
		d41 = snap144
		d42 = snap145
		d43 = snap146
		d44 = snap147
		d45 = snap148
		d46 = snap149
		d47 = snap150
		d48 = snap151
		d49 = snap152
		d50 = snap153
		d51 = snap154
		d54 = snap155
		d55 = snap156
		d56 = snap157
		if !bbs[3].Rendered {
			return bbs[3].RenderPS(ps107)
		}
		return result
		return result
	}
	bbs[2].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d159 := ps.PhiValues[0]
				ctx.EnsureDesc(&d159)
				ctx.EmitStoreToStack(d159, int32(bbs[2].PhiBase)+int32(0))
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
		if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != scm.LocNone {
			d159 = ps.OverlayValues[159]
		}
		if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
			d8 = ps.PhiValues[0]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d8)
		ctx.EnsureDesc(&d8)
		ctx.EnsureDesc(&d8)
		var d160 scm.JITValueDesc
		if d8.Loc == scm.LocImm {
			d160 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(d8.Imm.Int()))))}
		} else {
			r38 := ctx.AllocReg()
			ctx.EmitMovRegReg(r38, d8.Reg)
			ctx.EmitShlRegImm8(r38, 32)
			ctx.EmitShrRegImm8(r38, 32)
			d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
			ctx.BindReg(r38, &d160)
		}
		ctx.EnsureDesc(&d160)
		if thisptr.Loc == scm.LocImm {
			baseReg := ctx.AllocReg()
			if d160.Loc == scm.LocReg {
				ctx.FreeReg(baseReg)
				baseReg = ctx.AllocRegExcept(d160.Reg)
			}
			ctx.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int())+unsafe.Offsetof((*StorageSeq)(nil).lastValue)))
			if d160.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d160.Imm.Int()))
				ctx.EmitStoreRegMem(scm.RegR11, baseReg, 0)
			} else {
				ctx.EmitStoreRegMem(d160.Reg, baseReg, 0)
			}
			ctx.FreeReg(baseReg)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
			if d160.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d160.Imm.Int()))
				ctx.EmitStoreRegMem(scm.RegR11, thisptr.Reg, off)
			} else {
				ctx.EmitStoreRegMem(d160.Reg, thisptr.Reg, off)
			}
		}
		ctx.FreeDesc(&d160)
		ctx.EnsureDesc(&d8)
		d161 = d8
		_ = d161
		ctx.StabilizeDescForControlFlow(&d8)
		bbpos_2_0 := int32(-1)
		_ = bbpos_2_0
		lbl16 := ctx.ReserveLabel()
		_ = lbl16
		bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl16)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d162 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 48)
			r39 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r39, thisptr.Reg, off)
			d162 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r39}
			ctx.BindReg(r39, &d162)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d162)
		ctx.EnsureDesc(&d162)
		var d163 scm.JITValueDesc
		if d162.Loc == scm.LocImm {
			d163 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d162.Imm.Int()))))}
		} else {
			r40 := ctx.AllocReg()
			ctx.EmitMovRegReg(r40, d162.Reg)
			ctx.EmitShlRegImm8(r40, 56)
			ctx.EmitShrRegImm8(r40, 56)
			d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
			ctx.BindReg(r40, &d163)
		}
		ctx.FreeDesc(&d162)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d161)
		ctx.EnsureDesc(&d161)
		var d164 scm.JITValueDesc
		if d161.Loc == scm.LocImm {
			d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d161.Imm.Int()))))}
		} else {
			r41 := ctx.AllocReg()
			ctx.EmitMovRegReg(r41, d161.Reg)
			ctx.EmitShlRegImm8(r41, 32)
			ctx.EmitShrRegImm8(r41, 32)
			d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
			ctx.BindReg(r41, &d164)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d164)
		ctx.EnsureDesc(&d163)
		ctx.EnsureDescsTogether(&d164, &d163)
		var d165 scm.JITValueDesc
		if d164.Loc == scm.LocImm && d163.Loc == scm.LocImm {
			d165 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d164.Imm.Int() * d163.Imm.Int())}
		} else if d164.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d163.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d164.Imm.Int()))
			ctx.EmitImulInt64(scratch, d163.Reg)
			d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d165)
		} else if d163.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d164.Reg)
			ctx.EmitMovRegReg(scratch, d164.Reg)
			if d163.Imm.Int() >= -2147483648 && d163.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d163.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d163.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d165)
		} else {
			r42 := ctx.AllocRegExcept(d164.Reg, d163.Reg)
			ctx.EmitMovRegReg(r42, d164.Reg)
			ctx.EmitImulInt64(r42, d163.Reg)
			d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
			ctx.BindReg(r42, &d165)
		}
		if d165.Loc == scm.LocReg && d164.Loc == scm.LocReg && d165.Reg == d164.Reg {
			ctx.TransferReg(d164.Reg)
			d164.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d164)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d165)
		var d166 scm.JITValueDesc
		if d165.Loc == scm.LocImm {
			d166 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d165.Imm.Int() / 64)}
		} else {
			r43 := ctx.AllocRegExcept(d165.Reg)
			ctx.EmitMovRegReg(r43, d165.Reg)
			ctx.EmitShrRegImm8(r43, 6)
			d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
			ctx.BindReg(r43, &d166)
		}
		if d166.Loc == scm.LocReg && d165.Loc == scm.LocReg && d166.Reg == d165.Reg {
			ctx.TransferReg(d165.Reg)
			d165.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d165)
		var d167 scm.JITValueDesc
		if d165.Loc == scm.LocImm {
			d167 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d165.Imm.Int() % 64)}
		} else {
			r44 := ctx.AllocRegExcept(d165.Reg)
			ctx.EmitMovRegReg(r44, d165.Reg)
			ctx.EmitAndRegImm32(r44, 63)
			d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
			ctx.BindReg(r44, &d167)
		}
		if d167.Loc == scm.LocReg && d165.Loc == scm.LocReg && d167.Reg == d165.Reg {
			ctx.TransferReg(d165.Reg)
			d165.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d165)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d168 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d168 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r45 := ctx.AllocReg()
			r46 := ctx.AllocRegExcept(r45)
			r47 := ctx.AllocRegExcept(r45, r46)
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
			ctx.EmitMovRegMem(r45, thisptr.Reg, off)
			ctx.EmitMovRegMem(r46, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r47, thisptr.Reg, off+16)
			d168 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r45, Reg2: r46, Reg3: r47}
			ctx.BindReg(r45, &d168)
			ctx.BindReg(r46, &d168)
			ctx.BindReg(r47, &d168)
			ctx.BindReg(r45, &d168)
			ctx.BindReg(r46, &d168)
			ctx.BindReg(r47, &d168)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d166)
		ctx.ReclaimUntrackedRegs()
		d169 = ctx.EmitLoadScalarSliceElement(&d168, &d166, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d169)
		ctx.EnsureDesc(&d167)
		ctx.EnsureDescsTogether(&d169, &d167)
		var d170 scm.JITValueDesc
		if d169.Loc == scm.LocImm && d167.Loc == scm.LocImm {
			d170 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d169.Imm.Int()) << uint64(d167.Imm.Int())))}
		} else if d167.Loc == scm.LocImm {
			r48 := ctx.AllocRegExcept(d169.Reg)
			ctx.EmitMovRegReg(r48, d169.Reg)
			ctx.EmitShlRegImm8(r48, uint8(d167.Imm.Int()))
			d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
			ctx.BindReg(r48, &d170)
		} else {
			{
				shiftSrc := d169.Reg
				r49 := ctx.AllocRegExcept(d169.Reg, d167.Reg)
				ctx.EmitMovRegReg(r49, d169.Reg)
				shiftSrc = r49
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d167.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d167.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d167.Reg)
				}
				ctx.EmitShlRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d170)
			}
		}
		if d170.Loc == scm.LocReg && d169.Loc == scm.LocReg && d170.Reg == d169.Reg {
			ctx.TransferReg(d169.Reg)
			d169.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d169)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d166)
		ctx.EnsureDesc(&d166)
		var d171 scm.JITValueDesc
		if d166.Loc == scm.LocImm {
			d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d166.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d166.Reg)
			ctx.EmitMovRegReg(scratch, d166.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d171)
		}
		if d171.Loc == scm.LocReg && d166.Loc == scm.LocReg && d171.Reg == d166.Reg {
			ctx.TransferReg(d166.Reg)
			d166.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d166)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d171)
		ctx.ReclaimUntrackedRegs()
		d172 = ctx.EmitLoadScalarSliceElement(&d168, &d171, 8, scm.TagInt)
		ctx.FreeDesc(&d171)
		ctx.ReclaimUntrackedRegs()
		d173 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d167)
		ctx.EnsureDescsTogether(&d173, &d167)
		var d174 scm.JITValueDesc
		if d173.Loc == scm.LocImm && d167.Loc == scm.LocImm {
			d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d173.Imm.Int() - d167.Imm.Int())}
		} else if d167.Loc == scm.LocImm && d167.Imm.Int() == 0 {
			r50 := ctx.AllocRegExcept(d173.Reg)
			ctx.EmitMovRegReg(r50, d173.Reg)
			d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
			ctx.BindReg(r50, &d174)
		} else if d173.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d167.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d173.Imm.Int()))
			ctx.EmitSubInt64(scratch, d167.Reg)
			d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d174)
		} else if d167.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d173.Reg)
			ctx.EmitMovRegReg(scratch, d173.Reg)
			if d167.Imm.Int() >= -2147483648 && d167.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d167.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d167.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d174)
		} else {
			r51 := ctx.AllocRegExcept(d173.Reg, d167.Reg)
			ctx.EmitMovRegReg(r51, d173.Reg)
			ctx.EmitSubInt64(r51, d167.Reg)
			d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
			ctx.BindReg(r51, &d174)
		}
		if d174.Loc == scm.LocReg && d173.Loc == scm.LocReg && d174.Reg == d173.Reg {
			ctx.TransferReg(d173.Reg)
			d173.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d167)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d172)
		ctx.EnsureDesc(&d174)
		ctx.EnsureDescsTogether(&d172, &d174)
		var d175 scm.JITValueDesc
		if d172.Loc == scm.LocImm && d174.Loc == scm.LocImm {
			d175 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d172.Imm.Int()) >> uint64(d174.Imm.Int())))}
		} else if d174.Loc == scm.LocImm {
			r52 := ctx.AllocRegExcept(d172.Reg)
			ctx.EmitMovRegReg(r52, d172.Reg)
			ctx.EmitShrRegImm8(r52, uint8(d174.Imm.Int()))
			d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
			ctx.BindReg(r52, &d175)
		} else {
			{
				shiftSrc := d172.Reg
				r53 := ctx.AllocRegExcept(d172.Reg, d174.Reg)
				ctx.EmitMovRegReg(r53, d172.Reg)
				shiftSrc = r53
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d174.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d174.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d174.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d175)
			}
		}
		if d175.Loc == scm.LocReg && d172.Loc == scm.LocReg && d175.Reg == d172.Reg {
			ctx.TransferReg(d172.Reg)
			d172.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d172)
		ctx.FreeDesc(&d174)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d170)
		ctx.EnsureDesc(&d175)
		var d176 scm.JITValueDesc
		if d170.Loc == scm.LocImm && d175.Loc == scm.LocImm {
			d176 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d170.Imm.Int() | d175.Imm.Int())}
		} else if d170.Loc == scm.LocImm && d170.Imm.Int() == 0 {
			d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d175.Reg}
			ctx.BindReg(d175.Reg, &d176)
		} else if d175.Loc == scm.LocImm && d175.Imm.Int() == 0 {
			r54 := ctx.AllocRegExcept(d170.Reg)
			ctx.EmitMovRegReg(r54, d170.Reg)
			d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
			ctx.BindReg(r54, &d176)
		} else if d170.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d175.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d170.Imm.Int()))
			ctx.EmitOrInt64(scratch, d175.Reg)
			d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d176)
		} else if d175.Loc == scm.LocImm {
			r55 := ctx.AllocRegExcept(d170.Reg)
			ctx.EmitMovRegReg(r55, d170.Reg)
			if d175.Imm.Int() >= -2147483648 && d175.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r55, int32(d175.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d175.Imm.Int()))
				ctx.EmitOrInt64(r55, scm.RegR11)
			}
			d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
			ctx.BindReg(r55, &d176)
		} else {
			r56 := ctx.AllocRegExcept(d170.Reg, d175.Reg)
			ctx.EmitMovRegReg(r56, d170.Reg)
			ctx.EmitOrInt64(r56, d175.Reg)
			d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r56}
			ctx.BindReg(r56, &d176)
		}
		if d176.Loc == scm.LocReg && d170.Loc == scm.LocReg && d176.Reg == d170.Reg {
			ctx.TransferReg(d170.Reg)
			d170.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d170)
		ctx.FreeDesc(&d175)
		ctx.ReclaimUntrackedRegs()
		d177 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d163)
		ctx.EnsureDescsTogether(&d177, &d163)
		var d178 scm.JITValueDesc
		if d177.Loc == scm.LocImm && d163.Loc == scm.LocImm {
			d178 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d177.Imm.Int() - d163.Imm.Int())}
		} else if d163.Loc == scm.LocImm && d163.Imm.Int() == 0 {
			r57 := ctx.AllocRegExcept(d177.Reg)
			ctx.EmitMovRegReg(r57, d177.Reg)
			d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r57}
			ctx.BindReg(r57, &d178)
		} else if d177.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d163.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d177.Imm.Int()))
			ctx.EmitSubInt64(scratch, d163.Reg)
			d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d178)
		} else if d163.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d177.Reg)
			ctx.EmitMovRegReg(scratch, d177.Reg)
			if d163.Imm.Int() >= -2147483648 && d163.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d163.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d163.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d178)
		} else {
			r58 := ctx.AllocRegExcept(d177.Reg, d163.Reg)
			ctx.EmitMovRegReg(r58, d177.Reg)
			ctx.EmitSubInt64(r58, d163.Reg)
			d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r58}
			ctx.BindReg(r58, &d178)
		}
		if d178.Loc == scm.LocReg && d177.Loc == scm.LocReg && d178.Reg == d177.Reg {
			ctx.TransferReg(d177.Reg)
			d177.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d163)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d176)
		ctx.EnsureDesc(&d178)
		ctx.EnsureDescsTogether(&d176, &d178)
		var d179 scm.JITValueDesc
		if d176.Loc == scm.LocImm && d178.Loc == scm.LocImm {
			d179 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d176.Imm.Int()) >> uint64(d178.Imm.Int())))}
		} else if d178.Loc == scm.LocImm {
			r59 := ctx.AllocRegExcept(d176.Reg)
			ctx.EmitMovRegReg(r59, d176.Reg)
			ctx.EmitShrRegImm8(r59, uint8(d178.Imm.Int()))
			d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r59}
			ctx.BindReg(r59, &d179)
		} else {
			{
				shiftSrc := d176.Reg
				r60 := ctx.AllocRegExcept(d176.Reg, d178.Reg)
				ctx.EmitMovRegReg(r60, d176.Reg)
				shiftSrc = r60
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d178.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d178.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d178.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d179)
			}
		}
		if d179.Loc == scm.LocReg && d176.Loc == scm.LocReg && d179.Reg == d176.Reg {
			ctx.TransferReg(d176.Reg)
			d176.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d176)
		ctx.FreeDesc(&d178)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d179)
		ctx.EnsureDesc(&d179)
		ctx.EnsureDesc(&d179)
		var d180 scm.JITValueDesc
		if d179.Loc == scm.LocImm {
			d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d179.Imm.Int()))))}
		} else {
			r61 := ctx.AllocReg()
			ctx.EmitMovRegReg(r61, d179.Reg)
			d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r61}
			ctx.BindReg(r61, &d180)
		}
		ctx.FreeDesc(&d179)
		var d181 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d181 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 56)
			r62 := ctx.AllocReg()
			ctx.EmitMovRegMem(r62, thisptr.Reg, off)
			d181 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r62}
			ctx.BindReg(r62, &d181)
		}
		ctx.EnsureDesc(&d180)
		ctx.EnsureDesc(&d181)
		ctx.EnsureDescsTogether(&d180, &d181)
		var d182 scm.JITValueDesc
		if d180.Loc == scm.LocImm && d181.Loc == scm.LocImm {
			d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d180.Imm.Int() + d181.Imm.Int())}
		} else if d181.Loc == scm.LocImm && d181.Imm.Int() == 0 {
			r63 := ctx.AllocRegExcept(d180.Reg)
			ctx.EmitMovRegReg(r63, d180.Reg)
			d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r63}
			ctx.BindReg(r63, &d182)
		} else if d180.Loc == scm.LocImm && d180.Imm.Int() == 0 {
			d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d181.Reg}
			ctx.BindReg(d181.Reg, &d182)
		} else if d180.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d181.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d180.Imm.Int()))
			ctx.EmitAddInt64(scratch, d181.Reg)
			d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d182)
		} else if d181.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d180.Reg)
			ctx.EmitMovRegReg(scratch, d180.Reg)
			if d181.Imm.Int() >= -2147483648 && d181.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d181.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d181.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d182)
		} else {
			r64 := ctx.AllocRegExcept(d180.Reg, d181.Reg)
			ctx.EmitMovRegReg(r64, d180.Reg)
			ctx.EmitAddInt64(r64, d181.Reg)
			d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r64}
			ctx.BindReg(r64, &d182)
		}
		if d182.Loc == scm.LocReg && d180.Loc == scm.LocReg && d182.Reg == d180.Reg {
			ctx.TransferReg(d180.Reg)
			d180.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d182)
		ctx.FreeDesc(&d180)
		ctx.FreeDesc(&d181)
		var d183 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 80
			val := *(*bool)(unsafe.Pointer(fieldAddr))
			d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 80)
			r65 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r65, thisptr.Reg, off)
			d183 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r65}
			ctx.BindReg(r65, &d183)
		}
		d184 = d183
		ctx.EnsureDesc(&d184)
		if d184.Loc != scm.LocImm && d184.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d184.Loc == scm.LocImm {
			if d184.Imm.Bool() {
				if ps.General {
				}
				ps185 := scm.PhiState{General: ps.General}
				ps185.OverlayValues = make([]scm.JITValueDesc, 185)
				ps185.OverlayValues[5] = d5
				ps185.OverlayValues[6] = d6
				ps185.OverlayValues[7] = d7
				ps185.OverlayValues[8] = d8
				ps185.OverlayValues[9] = d9
				ps185.OverlayValues[10] = d10
				ps185.OverlayValues[11] = d11
				ps185.OverlayValues[12] = d12
				ps185.OverlayValues[13] = d13
				ps185.OverlayValues[14] = d14
				ps185.OverlayValues[15] = d15
				ps185.OverlayValues[16] = d16
				ps185.OverlayValues[17] = d17
				ps185.OverlayValues[18] = d18
				ps185.OverlayValues[19] = d19
				ps185.OverlayValues[21] = d21
				ps185.OverlayValues[22] = d22
				ps185.OverlayValues[23] = d23
				ps185.OverlayValues[24] = d24
				ps185.OverlayValues[25] = d25
				ps185.OverlayValues[26] = d26
				ps185.OverlayValues[27] = d27
				ps185.OverlayValues[28] = d28
				ps185.OverlayValues[29] = d29
				ps185.OverlayValues[30] = d30
				ps185.OverlayValues[31] = d31
				ps185.OverlayValues[32] = d32
				ps185.OverlayValues[33] = d33
				ps185.OverlayValues[34] = d34
				ps185.OverlayValues[35] = d35
				ps185.OverlayValues[36] = d36
				ps185.OverlayValues[37] = d37
				ps185.OverlayValues[38] = d38
				ps185.OverlayValues[39] = d39
				ps185.OverlayValues[40] = d40
				ps185.OverlayValues[41] = d41
				ps185.OverlayValues[42] = d42
				ps185.OverlayValues[43] = d43
				ps185.OverlayValues[44] = d44
				ps185.OverlayValues[45] = d45
				ps185.OverlayValues[46] = d46
				ps185.OverlayValues[47] = d47
				ps185.OverlayValues[48] = d48
				ps185.OverlayValues[49] = d49
				ps185.OverlayValues[50] = d50
				ps185.OverlayValues[51] = d51
				ps185.OverlayValues[54] = d54
				ps185.OverlayValues[55] = d55
				ps185.OverlayValues[56] = d56
				ps185.OverlayValues[159] = d159
				ps185.OverlayValues[160] = d160
				ps185.OverlayValues[161] = d161
				ps185.OverlayValues[162] = d162
				ps185.OverlayValues[163] = d163
				ps185.OverlayValues[164] = d164
				ps185.OverlayValues[165] = d165
				ps185.OverlayValues[166] = d166
				ps185.OverlayValues[167] = d167
				ps185.OverlayValues[168] = d168
				ps185.OverlayValues[169] = d169
				ps185.OverlayValues[170] = d170
				ps185.OverlayValues[171] = d171
				ps185.OverlayValues[172] = d172
				ps185.OverlayValues[173] = d173
				ps185.OverlayValues[174] = d174
				ps185.OverlayValues[175] = d175
				ps185.OverlayValues[176] = d176
				ps185.OverlayValues[177] = d177
				ps185.OverlayValues[178] = d178
				ps185.OverlayValues[179] = d179
				ps185.OverlayValues[180] = d180
				ps185.OverlayValues[181] = d181
				ps185.OverlayValues[182] = d182
				ps185.OverlayValues[183] = d183
				ps185.OverlayValues[184] = d184
				return bbs[13].RenderPS(ps185)
			}
			if ps.General {
			}
			ps186 := scm.PhiState{General: ps.General}
			ps186.OverlayValues = make([]scm.JITValueDesc, 185)
			ps186.OverlayValues[5] = d5
			ps186.OverlayValues[6] = d6
			ps186.OverlayValues[7] = d7
			ps186.OverlayValues[8] = d8
			ps186.OverlayValues[9] = d9
			ps186.OverlayValues[10] = d10
			ps186.OverlayValues[11] = d11
			ps186.OverlayValues[12] = d12
			ps186.OverlayValues[13] = d13
			ps186.OverlayValues[14] = d14
			ps186.OverlayValues[15] = d15
			ps186.OverlayValues[16] = d16
			ps186.OverlayValues[17] = d17
			ps186.OverlayValues[18] = d18
			ps186.OverlayValues[19] = d19
			ps186.OverlayValues[21] = d21
			ps186.OverlayValues[22] = d22
			ps186.OverlayValues[23] = d23
			ps186.OverlayValues[24] = d24
			ps186.OverlayValues[25] = d25
			ps186.OverlayValues[26] = d26
			ps186.OverlayValues[27] = d27
			ps186.OverlayValues[28] = d28
			ps186.OverlayValues[29] = d29
			ps186.OverlayValues[30] = d30
			ps186.OverlayValues[31] = d31
			ps186.OverlayValues[32] = d32
			ps186.OverlayValues[33] = d33
			ps186.OverlayValues[34] = d34
			ps186.OverlayValues[35] = d35
			ps186.OverlayValues[36] = d36
			ps186.OverlayValues[37] = d37
			ps186.OverlayValues[38] = d38
			ps186.OverlayValues[39] = d39
			ps186.OverlayValues[40] = d40
			ps186.OverlayValues[41] = d41
			ps186.OverlayValues[42] = d42
			ps186.OverlayValues[43] = d43
			ps186.OverlayValues[44] = d44
			ps186.OverlayValues[45] = d45
			ps186.OverlayValues[46] = d46
			ps186.OverlayValues[47] = d47
			ps186.OverlayValues[48] = d48
			ps186.OverlayValues[49] = d49
			ps186.OverlayValues[50] = d50
			ps186.OverlayValues[51] = d51
			ps186.OverlayValues[54] = d54
			ps186.OverlayValues[55] = d55
			ps186.OverlayValues[56] = d56
			ps186.OverlayValues[159] = d159
			ps186.OverlayValues[160] = d160
			ps186.OverlayValues[161] = d161
			ps186.OverlayValues[162] = d162
			ps186.OverlayValues[163] = d163
			ps186.OverlayValues[164] = d164
			ps186.OverlayValues[165] = d165
			ps186.OverlayValues[166] = d166
			ps186.OverlayValues[167] = d167
			ps186.OverlayValues[168] = d168
			ps186.OverlayValues[169] = d169
			ps186.OverlayValues[170] = d170
			ps186.OverlayValues[171] = d171
			ps186.OverlayValues[172] = d172
			ps186.OverlayValues[173] = d173
			ps186.OverlayValues[174] = d174
			ps186.OverlayValues[175] = d175
			ps186.OverlayValues[176] = d176
			ps186.OverlayValues[177] = d177
			ps186.OverlayValues[178] = d178
			ps186.OverlayValues[179] = d179
			ps186.OverlayValues[180] = d180
			ps186.OverlayValues[181] = d181
			ps186.OverlayValues[182] = d182
			ps186.OverlayValues[183] = d183
			ps186.OverlayValues[184] = d184
			return bbs[12].RenderPS(ps186)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d187 := ps.PhiValues[0]
				ctx.EnsureDesc(&d187)
				ctx.EmitStoreToStack(d187, int32(bbs[2].PhiBase)+int32(0))
			}
			ps.General = true
			return bbs[2].RenderPS(ps)
		}
		ctx.EmitCmpRegImm32(d184.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl14)
		snap188 := d5
		snap189 := d6
		snap190 := d7
		snap191 := d8
		snap192 := d9
		snap193 := d10
		snap194 := d11
		snap195 := d12
		snap196 := d13
		snap197 := d14
		snap198 := d15
		snap199 := d16
		snap200 := d17
		snap201 := d18
		snap202 := d19
		snap203 := d21
		snap204 := d22
		snap205 := d23
		snap206 := d24
		snap207 := d25
		snap208 := d26
		snap209 := d27
		snap210 := d28
		snap211 := d29
		snap212 := d30
		snap213 := d31
		snap214 := d32
		snap215 := d33
		snap216 := d34
		snap217 := d35
		snap218 := d36
		snap219 := d37
		snap220 := d38
		snap221 := d39
		snap222 := d40
		snap223 := d41
		snap224 := d42
		snap225 := d43
		snap226 := d44
		snap227 := d45
		snap228 := d46
		snap229 := d47
		snap230 := d48
		snap231 := d49
		snap232 := d50
		snap233 := d51
		snap234 := d54
		snap235 := d55
		snap236 := d56
		snap237 := d159
		snap238 := d160
		snap239 := d161
		snap240 := d162
		snap241 := d163
		snap242 := d164
		snap243 := d165
		snap244 := d166
		snap245 := d167
		snap246 := d168
		snap247 := d169
		snap248 := d170
		snap249 := d171
		snap250 := d172
		snap251 := d173
		snap252 := d174
		snap253 := d175
		snap254 := d176
		snap255 := d177
		snap256 := d178
		snap257 := d179
		snap258 := d180
		snap259 := d181
		snap260 := d182
		snap261 := d183
		snap262 := d184
		snap263 := d187
		alloc264 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc264)
		d5 = snap188
		d6 = snap189
		d7 = snap190
		d8 = snap191
		d9 = snap192
		d10 = snap193
		d11 = snap194
		d12 = snap195
		d13 = snap196
		d14 = snap197
		d15 = snap198
		d16 = snap199
		d17 = snap200
		d18 = snap201
		d19 = snap202
		d21 = snap203
		d22 = snap204
		d23 = snap205
		d24 = snap206
		d25 = snap207
		d26 = snap208
		d27 = snap209
		d28 = snap210
		d29 = snap211
		d30 = snap212
		d31 = snap213
		d32 = snap214
		d33 = snap215
		d34 = snap216
		d35 = snap217
		d36 = snap218
		d37 = snap219
		d38 = snap220
		d39 = snap221
		d40 = snap222
		d41 = snap223
		d42 = snap224
		d43 = snap225
		d44 = snap226
		d45 = snap227
		d46 = snap228
		d47 = snap229
		d48 = snap230
		d49 = snap231
		d50 = snap232
		d51 = snap233
		d54 = snap234
		d55 = snap235
		d56 = snap236
		d159 = snap237
		d160 = snap238
		d161 = snap239
		d162 = snap240
		d163 = snap241
		d164 = snap242
		d165 = snap243
		d166 = snap244
		d167 = snap245
		d168 = snap246
		d169 = snap247
		d170 = snap248
		d171 = snap249
		d172 = snap250
		d173 = snap251
		d174 = snap252
		d175 = snap253
		d176 = snap254
		d177 = snap255
		d178 = snap256
		d179 = snap257
		d180 = snap258
		d181 = snap259
		d182 = snap260
		d183 = snap261
		d184 = snap262
		d187 = snap263
		ctx.RestoreAllocState(alloc264)
		d5 = snap188
		d6 = snap189
		d7 = snap190
		d8 = snap191
		d9 = snap192
		d10 = snap193
		d11 = snap194
		d12 = snap195
		d13 = snap196
		d14 = snap197
		d15 = snap198
		d16 = snap199
		d17 = snap200
		d18 = snap201
		d19 = snap202
		d21 = snap203
		d22 = snap204
		d23 = snap205
		d24 = snap206
		d25 = snap207
		d26 = snap208
		d27 = snap209
		d28 = snap210
		d29 = snap211
		d30 = snap212
		d31 = snap213
		d32 = snap214
		d33 = snap215
		d34 = snap216
		d35 = snap217
		d36 = snap218
		d37 = snap219
		d38 = snap220
		d39 = snap221
		d40 = snap222
		d41 = snap223
		d42 = snap224
		d43 = snap225
		d44 = snap226
		d45 = snap227
		d46 = snap228
		d47 = snap229
		d48 = snap230
		d49 = snap231
		d50 = snap232
		d51 = snap233
		d54 = snap234
		d55 = snap235
		d56 = snap236
		d159 = snap237
		d160 = snap238
		d161 = snap239
		d162 = snap240
		d163 = snap241
		d164 = snap242
		d165 = snap243
		d166 = snap244
		d167 = snap245
		d168 = snap246
		d169 = snap247
		d170 = snap248
		d171 = snap249
		d172 = snap250
		d173 = snap251
		d174 = snap252
		d175 = snap253
		d176 = snap254
		d177 = snap255
		d178 = snap256
		d179 = snap257
		d180 = snap258
		d181 = snap259
		d182 = snap260
		d183 = snap261
		d184 = snap262
		d187 = snap263
		ps265 := scm.PhiState{General: true}
		ps265.OverlayValues = make([]scm.JITValueDesc, 188)
		ps265.OverlayValues[5] = d5
		ps265.OverlayValues[6] = d6
		ps265.OverlayValues[7] = d7
		ps265.OverlayValues[8] = d8
		ps265.OverlayValues[9] = d9
		ps265.OverlayValues[10] = d10
		ps265.OverlayValues[11] = d11
		ps265.OverlayValues[12] = d12
		ps265.OverlayValues[13] = d13
		ps265.OverlayValues[14] = d14
		ps265.OverlayValues[15] = d15
		ps265.OverlayValues[16] = d16
		ps265.OverlayValues[17] = d17
		ps265.OverlayValues[18] = d18
		ps265.OverlayValues[19] = d19
		ps265.OverlayValues[21] = d21
		ps265.OverlayValues[22] = d22
		ps265.OverlayValues[23] = d23
		ps265.OverlayValues[24] = d24
		ps265.OverlayValues[25] = d25
		ps265.OverlayValues[26] = d26
		ps265.OverlayValues[27] = d27
		ps265.OverlayValues[28] = d28
		ps265.OverlayValues[29] = d29
		ps265.OverlayValues[30] = d30
		ps265.OverlayValues[31] = d31
		ps265.OverlayValues[32] = d32
		ps265.OverlayValues[33] = d33
		ps265.OverlayValues[34] = d34
		ps265.OverlayValues[35] = d35
		ps265.OverlayValues[36] = d36
		ps265.OverlayValues[37] = d37
		ps265.OverlayValues[38] = d38
		ps265.OverlayValues[39] = d39
		ps265.OverlayValues[40] = d40
		ps265.OverlayValues[41] = d41
		ps265.OverlayValues[42] = d42
		ps265.OverlayValues[43] = d43
		ps265.OverlayValues[44] = d44
		ps265.OverlayValues[45] = d45
		ps265.OverlayValues[46] = d46
		ps265.OverlayValues[47] = d47
		ps265.OverlayValues[48] = d48
		ps265.OverlayValues[49] = d49
		ps265.OverlayValues[50] = d50
		ps265.OverlayValues[51] = d51
		ps265.OverlayValues[54] = d54
		ps265.OverlayValues[55] = d55
		ps265.OverlayValues[56] = d56
		ps265.OverlayValues[159] = d159
		ps265.OverlayValues[160] = d160
		ps265.OverlayValues[161] = d161
		ps265.OverlayValues[162] = d162
		ps265.OverlayValues[163] = d163
		ps265.OverlayValues[164] = d164
		ps265.OverlayValues[165] = d165
		ps265.OverlayValues[166] = d166
		ps265.OverlayValues[167] = d167
		ps265.OverlayValues[168] = d168
		ps265.OverlayValues[169] = d169
		ps265.OverlayValues[170] = d170
		ps265.OverlayValues[171] = d171
		ps265.OverlayValues[172] = d172
		ps265.OverlayValues[173] = d173
		ps265.OverlayValues[174] = d174
		ps265.OverlayValues[175] = d175
		ps265.OverlayValues[176] = d176
		ps265.OverlayValues[177] = d177
		ps265.OverlayValues[178] = d178
		ps265.OverlayValues[179] = d179
		ps265.OverlayValues[180] = d180
		ps265.OverlayValues[181] = d181
		ps265.OverlayValues[182] = d182
		ps265.OverlayValues[183] = d183
		ps265.OverlayValues[184] = d184
		ps265.OverlayValues[187] = d187
		ps266 := scm.PhiState{General: true}
		ps266.OverlayValues = make([]scm.JITValueDesc, 188)
		ps266.OverlayValues[5] = d5
		ps266.OverlayValues[6] = d6
		ps266.OverlayValues[7] = d7
		ps266.OverlayValues[8] = d8
		ps266.OverlayValues[9] = d9
		ps266.OverlayValues[10] = d10
		ps266.OverlayValues[11] = d11
		ps266.OverlayValues[12] = d12
		ps266.OverlayValues[13] = d13
		ps266.OverlayValues[14] = d14
		ps266.OverlayValues[15] = d15
		ps266.OverlayValues[16] = d16
		ps266.OverlayValues[17] = d17
		ps266.OverlayValues[18] = d18
		ps266.OverlayValues[19] = d19
		ps266.OverlayValues[21] = d21
		ps266.OverlayValues[22] = d22
		ps266.OverlayValues[23] = d23
		ps266.OverlayValues[24] = d24
		ps266.OverlayValues[25] = d25
		ps266.OverlayValues[26] = d26
		ps266.OverlayValues[27] = d27
		ps266.OverlayValues[28] = d28
		ps266.OverlayValues[29] = d29
		ps266.OverlayValues[30] = d30
		ps266.OverlayValues[31] = d31
		ps266.OverlayValues[32] = d32
		ps266.OverlayValues[33] = d33
		ps266.OverlayValues[34] = d34
		ps266.OverlayValues[35] = d35
		ps266.OverlayValues[36] = d36
		ps266.OverlayValues[37] = d37
		ps266.OverlayValues[38] = d38
		ps266.OverlayValues[39] = d39
		ps266.OverlayValues[40] = d40
		ps266.OverlayValues[41] = d41
		ps266.OverlayValues[42] = d42
		ps266.OverlayValues[43] = d43
		ps266.OverlayValues[44] = d44
		ps266.OverlayValues[45] = d45
		ps266.OverlayValues[46] = d46
		ps266.OverlayValues[47] = d47
		ps266.OverlayValues[48] = d48
		ps266.OverlayValues[49] = d49
		ps266.OverlayValues[50] = d50
		ps266.OverlayValues[51] = d51
		ps266.OverlayValues[54] = d54
		ps266.OverlayValues[55] = d55
		ps266.OverlayValues[56] = d56
		ps266.OverlayValues[159] = d159
		ps266.OverlayValues[160] = d160
		ps266.OverlayValues[161] = d161
		ps266.OverlayValues[162] = d162
		ps266.OverlayValues[163] = d163
		ps266.OverlayValues[164] = d164
		ps266.OverlayValues[165] = d165
		ps266.OverlayValues[166] = d166
		ps266.OverlayValues[167] = d167
		ps266.OverlayValues[168] = d168
		ps266.OverlayValues[169] = d169
		ps266.OverlayValues[170] = d170
		ps266.OverlayValues[171] = d171
		ps266.OverlayValues[172] = d172
		ps266.OverlayValues[173] = d173
		ps266.OverlayValues[174] = d174
		ps266.OverlayValues[175] = d175
		ps266.OverlayValues[176] = d176
		ps266.OverlayValues[177] = d177
		ps266.OverlayValues[178] = d178
		ps266.OverlayValues[179] = d179
		ps266.OverlayValues[180] = d180
		ps266.OverlayValues[181] = d181
		ps266.OverlayValues[182] = d182
		ps266.OverlayValues[183] = d183
		ps266.OverlayValues[184] = d184
		ps266.OverlayValues[187] = d187
		snap267 := d5
		snap268 := d6
		snap269 := d7
		snap270 := d8
		snap271 := d9
		snap272 := d10
		snap273 := d11
		snap274 := d12
		snap275 := d13
		snap276 := d14
		snap277 := d15
		snap278 := d16
		snap279 := d17
		snap280 := d18
		snap281 := d19
		snap282 := d21
		snap283 := d22
		snap284 := d23
		snap285 := d24
		snap286 := d25
		snap287 := d26
		snap288 := d27
		snap289 := d28
		snap290 := d29
		snap291 := d30
		snap292 := d31
		snap293 := d32
		snap294 := d33
		snap295 := d34
		snap296 := d35
		snap297 := d36
		snap298 := d37
		snap299 := d38
		snap300 := d39
		snap301 := d40
		snap302 := d41
		snap303 := d42
		snap304 := d43
		snap305 := d44
		snap306 := d45
		snap307 := d46
		snap308 := d47
		snap309 := d48
		snap310 := d49
		snap311 := d50
		snap312 := d51
		snap313 := d54
		snap314 := d55
		snap315 := d56
		snap316 := d159
		snap317 := d160
		snap318 := d161
		snap319 := d162
		snap320 := d163
		snap321 := d164
		snap322 := d165
		snap323 := d166
		snap324 := d167
		snap325 := d168
		snap326 := d169
		snap327 := d170
		snap328 := d171
		snap329 := d172
		snap330 := d173
		snap331 := d174
		snap332 := d175
		snap333 := d176
		snap334 := d177
		snap335 := d178
		snap336 := d179
		snap337 := d180
		snap338 := d181
		snap339 := d182
		snap340 := d183
		snap341 := d184
		snap342 := d187
		alloc343 := ctx.SnapshotAllocState()
		if !bbs[12].Rendered {
			bbs[12].RenderPS(ps266)
		}
		ctx.RestoreAllocState(alloc343)
		d5 = snap267
		d6 = snap268
		d7 = snap269
		d8 = snap270
		d9 = snap271
		d10 = snap272
		d11 = snap273
		d12 = snap274
		d13 = snap275
		d14 = snap276
		d15 = snap277
		d16 = snap278
		d17 = snap279
		d18 = snap280
		d19 = snap281
		d21 = snap282
		d22 = snap283
		d23 = snap284
		d24 = snap285
		d25 = snap286
		d26 = snap287
		d27 = snap288
		d28 = snap289
		d29 = snap290
		d30 = snap291
		d31 = snap292
		d32 = snap293
		d33 = snap294
		d34 = snap295
		d35 = snap296
		d36 = snap297
		d37 = snap298
		d38 = snap299
		d39 = snap300
		d40 = snap301
		d41 = snap302
		d42 = snap303
		d43 = snap304
		d44 = snap305
		d45 = snap306
		d46 = snap307
		d47 = snap308
		d48 = snap309
		d49 = snap310
		d50 = snap311
		d51 = snap312
		d54 = snap313
		d55 = snap314
		d56 = snap315
		d159 = snap316
		d160 = snap317
		d161 = snap318
		d162 = snap319
		d163 = snap320
		d164 = snap321
		d165 = snap322
		d166 = snap323
		d167 = snap324
		d168 = snap325
		d169 = snap326
		d170 = snap327
		d171 = snap328
		d172 = snap329
		d173 = snap330
		d174 = snap331
		d175 = snap332
		d176 = snap333
		d177 = snap334
		d178 = snap335
		d179 = snap336
		d180 = snap337
		d181 = snap338
		d182 = snap339
		d183 = snap340
		d184 = snap341
		d187 = snap342
		if !bbs[13].Rendered {
			return bbs[13].RenderPS(ps265)
		}
		return result
		ctx.FreeDesc(&d183)
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
		if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
			d187 = ps.OverlayValues[187]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d5)
		var d344 scm.JITValueDesc
		if d5.Loc == scm.LocImm {
			d344 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitSubRegImm32Low(scratch, int32(1))
			d344 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d344)
		}
		if d344.Loc == scm.LocReg && d5.Loc == scm.LocReg && d344.Reg == d5.Reg {
			ctx.TransferReg(d5.Reg)
			d5.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d344)
		ctx.EmitStoreToStack(d344, int32(bbs[4].PhiBase)+int32(32))
		ctx.StabilizeDescForControlFlow(&d344)
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d5)
		var d345 scm.JITValueDesc
		if d5.Loc == scm.LocImm {
			d345 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitSubRegImm32Low(scratch, int32(1))
			d345 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d345)
		}
		if d345.Loc == scm.LocReg && d5.Loc == scm.LocReg && d345.Reg == d5.Reg {
			ctx.TransferReg(d5.Reg)
			d5.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d345)
		ctx.EmitStoreToStack(d345, int32(bbs[4].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d345)
		if ps.General {
			ctx.SyncDesc(&d6)
			if d6.Loc == scm.LocReg {
				ctx.ProtectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.ProtectReg(d6.Reg)
				ctx.ProtectReg(d6.Reg2)
			}
			d346 = d6
			if d346.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d346)
			d347 = d346
			if d347.Loc == scm.LocImm {
				d347 = scm.JITValueDesc{Loc: scm.LocImm, Type: d347.Type, Imm: scm.NewInt(int64(uint64(d347.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d347.Reg, 32)
				ctx.EmitShrRegImm8(d347.Reg, 32)
			}
			ctx.EmitStoreToStack(d347, int32(bbs[4].PhiBase)+int32(16))
			if d6.Loc == scm.LocReg {
				ctx.UnprotectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d6.Reg)
				ctx.UnprotectReg(d6.Reg2)
			}
		}
		ps348 := scm.PhiState{General: ps.General}
		ps348.OverlayValues = make([]scm.JITValueDesc, 348)
		ps348.OverlayValues[5] = d5
		ps348.OverlayValues[6] = d6
		ps348.OverlayValues[7] = d7
		ps348.OverlayValues[8] = d8
		ps348.OverlayValues[9] = d9
		ps348.OverlayValues[10] = d10
		ps348.OverlayValues[11] = d11
		ps348.OverlayValues[12] = d12
		ps348.OverlayValues[13] = d13
		ps348.OverlayValues[14] = d14
		ps348.OverlayValues[15] = d15
		ps348.OverlayValues[16] = d16
		ps348.OverlayValues[17] = d17
		ps348.OverlayValues[18] = d18
		ps348.OverlayValues[19] = d19
		ps348.OverlayValues[21] = d21
		ps348.OverlayValues[22] = d22
		ps348.OverlayValues[23] = d23
		ps348.OverlayValues[24] = d24
		ps348.OverlayValues[25] = d25
		ps348.OverlayValues[26] = d26
		ps348.OverlayValues[27] = d27
		ps348.OverlayValues[28] = d28
		ps348.OverlayValues[29] = d29
		ps348.OverlayValues[30] = d30
		ps348.OverlayValues[31] = d31
		ps348.OverlayValues[32] = d32
		ps348.OverlayValues[33] = d33
		ps348.OverlayValues[34] = d34
		ps348.OverlayValues[35] = d35
		ps348.OverlayValues[36] = d36
		ps348.OverlayValues[37] = d37
		ps348.OverlayValues[38] = d38
		ps348.OverlayValues[39] = d39
		ps348.OverlayValues[40] = d40
		ps348.OverlayValues[41] = d41
		ps348.OverlayValues[42] = d42
		ps348.OverlayValues[43] = d43
		ps348.OverlayValues[44] = d44
		ps348.OverlayValues[45] = d45
		ps348.OverlayValues[46] = d46
		ps348.OverlayValues[47] = d47
		ps348.OverlayValues[48] = d48
		ps348.OverlayValues[49] = d49
		ps348.OverlayValues[50] = d50
		ps348.OverlayValues[51] = d51
		ps348.OverlayValues[54] = d54
		ps348.OverlayValues[55] = d55
		ps348.OverlayValues[56] = d56
		ps348.OverlayValues[159] = d159
		ps348.OverlayValues[160] = d160
		ps348.OverlayValues[161] = d161
		ps348.OverlayValues[162] = d162
		ps348.OverlayValues[163] = d163
		ps348.OverlayValues[164] = d164
		ps348.OverlayValues[165] = d165
		ps348.OverlayValues[166] = d166
		ps348.OverlayValues[167] = d167
		ps348.OverlayValues[168] = d168
		ps348.OverlayValues[169] = d169
		ps348.OverlayValues[170] = d170
		ps348.OverlayValues[171] = d171
		ps348.OverlayValues[172] = d172
		ps348.OverlayValues[173] = d173
		ps348.OverlayValues[174] = d174
		ps348.OverlayValues[175] = d175
		ps348.OverlayValues[176] = d176
		ps348.OverlayValues[177] = d177
		ps348.OverlayValues[178] = d178
		ps348.OverlayValues[179] = d179
		ps348.OverlayValues[180] = d180
		ps348.OverlayValues[181] = d181
		ps348.OverlayValues[182] = d182
		ps348.OverlayValues[183] = d183
		ps348.OverlayValues[184] = d184
		ps348.OverlayValues[187] = d187
		ps348.OverlayValues[344] = d344
		ps348.OverlayValues[345] = d345
		ps348.OverlayValues[346] = d346
		ps348.OverlayValues[347] = d347
		ps348.PhiValues = make([]scm.JITValueDesc, 3)
		d349 = d6
		ps348.PhiValues[1] = d349
		if ps348.General && bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
			return result
		}
		return bbs[4].RenderPS(ps348)
		return result
	}
	bbs[4].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d350 := ps.PhiValues[0]
				ctx.EnsureDesc(&d350)
				ctx.EmitStoreToStack(d350, int32(bbs[4].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d351 := ps.PhiValues[1]
				ctx.EnsureDesc(&d351)
				ctx.EmitStoreToStack(d351, int32(bbs[4].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d352 := ps.PhiValues[2]
				ctx.EnsureDesc(&d352)
				ctx.EmitStoreToStack(d352, int32(bbs[4].PhiBase)+int32(32))
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
		if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
			d187 = ps.OverlayValues[187]
		}
		if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != scm.LocNone {
			d344 = ps.OverlayValues[344]
		}
		if len(ps.OverlayValues) > 345 && ps.OverlayValues[345].Loc != scm.LocNone {
			d345 = ps.OverlayValues[345]
		}
		if len(ps.OverlayValues) > 346 && ps.OverlayValues[346].Loc != scm.LocNone {
			d346 = ps.OverlayValues[346]
		}
		if len(ps.OverlayValues) > 347 && ps.OverlayValues[347].Loc != scm.LocNone {
			d347 = ps.OverlayValues[347]
		}
		if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != scm.LocNone {
			d349 = ps.OverlayValues[349]
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
		var d353 scm.JITValueDesc
		if d10.Loc == scm.LocImm && d11.Loc == scm.LocImm {
			d353 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d10.Imm.Int()) == uint64(d11.Imm.Int()))}
		} else if d11.Loc == scm.LocImm {
			r66 := ctx.AllocRegExcept(d10.Reg)
			if d11.Imm.Int() >= -2147483648 && d11.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d10.Reg, int32(d11.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d11.Imm.Int()))
				ctx.EmitCmpInt64(d10.Reg, scm.RegR11)
			}
			d353 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r66, Condition: scm.CondEqual}
			ctx.BindReg(r66, &d353)
		} else if d10.Loc == scm.LocImm {
			r67 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d10.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d11.Reg)
			d353 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r67, Condition: scm.CondEqual}
			ctx.BindReg(r67, &d353)
		} else {
			r68 := ctx.AllocRegExcept(d10.Reg)
			ctx.EmitCmpInt64(d10.Reg, d11.Reg)
			d353 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r68, Condition: scm.CondEqual}
			ctx.BindReg(r68, &d353)
		}
		d354 = d353
		ctx.EnsureDesc(&d354)
		if d354.Loc != scm.LocImm && d354.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d354.Loc == scm.LocImm {
			if d354.Imm.Bool() {
				if ps.General {
					ctx.SyncDesc(&d10)
					if d10.Loc == scm.LocReg {
						ctx.ProtectReg(d10.Reg)
					} else if d10.Loc == scm.LocRegPair {
						ctx.ProtectReg(d10.Reg)
						ctx.ProtectReg(d10.Reg2)
					}
					d355 = d10
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
					ctx.EmitStoreToStack(d356, int32(bbs[2].PhiBase)+int32(0))
					if d10.Loc == scm.LocReg {
						ctx.UnprotectReg(d10.Reg)
					} else if d10.Loc == scm.LocRegPair {
						ctx.UnprotectReg(d10.Reg)
						ctx.UnprotectReg(d10.Reg2)
					}
				}
				ps357 := scm.PhiState{General: ps.General}
				ps357.OverlayValues = make([]scm.JITValueDesc, 357)
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
				ps357.OverlayValues[18] = d18
				ps357.OverlayValues[19] = d19
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
				ps357.OverlayValues[159] = d159
				ps357.OverlayValues[160] = d160
				ps357.OverlayValues[161] = d161
				ps357.OverlayValues[162] = d162
				ps357.OverlayValues[163] = d163
				ps357.OverlayValues[164] = d164
				ps357.OverlayValues[165] = d165
				ps357.OverlayValues[166] = d166
				ps357.OverlayValues[167] = d167
				ps357.OverlayValues[168] = d168
				ps357.OverlayValues[169] = d169
				ps357.OverlayValues[170] = d170
				ps357.OverlayValues[171] = d171
				ps357.OverlayValues[172] = d172
				ps357.OverlayValues[173] = d173
				ps357.OverlayValues[174] = d174
				ps357.OverlayValues[175] = d175
				ps357.OverlayValues[176] = d176
				ps357.OverlayValues[177] = d177
				ps357.OverlayValues[178] = d178
				ps357.OverlayValues[179] = d179
				ps357.OverlayValues[180] = d180
				ps357.OverlayValues[181] = d181
				ps357.OverlayValues[182] = d182
				ps357.OverlayValues[183] = d183
				ps357.OverlayValues[184] = d184
				ps357.OverlayValues[187] = d187
				ps357.OverlayValues[344] = d344
				ps357.OverlayValues[345] = d345
				ps357.OverlayValues[346] = d346
				ps357.OverlayValues[347] = d347
				ps357.OverlayValues[349] = d349
				ps357.OverlayValues[350] = d350
				ps357.OverlayValues[351] = d351
				ps357.OverlayValues[352] = d352
				ps357.OverlayValues[353] = d353
				ps357.OverlayValues[354] = d354
				ps357.OverlayValues[355] = d355
				ps357.OverlayValues[356] = d356
				ps357.PhiValues = make([]scm.JITValueDesc, 1)
				d358 = d10
				ps357.PhiValues[0] = d358
				return bbs[2].RenderPS(ps357)
			}
			if ps.General {
			}
			ps359 := scm.PhiState{General: ps.General}
			ps359.OverlayValues = make([]scm.JITValueDesc, 359)
			ps359.OverlayValues[5] = d5
			ps359.OverlayValues[6] = d6
			ps359.OverlayValues[7] = d7
			ps359.OverlayValues[8] = d8
			ps359.OverlayValues[9] = d9
			ps359.OverlayValues[10] = d10
			ps359.OverlayValues[11] = d11
			ps359.OverlayValues[12] = d12
			ps359.OverlayValues[13] = d13
			ps359.OverlayValues[14] = d14
			ps359.OverlayValues[15] = d15
			ps359.OverlayValues[16] = d16
			ps359.OverlayValues[17] = d17
			ps359.OverlayValues[18] = d18
			ps359.OverlayValues[19] = d19
			ps359.OverlayValues[21] = d21
			ps359.OverlayValues[22] = d22
			ps359.OverlayValues[23] = d23
			ps359.OverlayValues[24] = d24
			ps359.OverlayValues[25] = d25
			ps359.OverlayValues[26] = d26
			ps359.OverlayValues[27] = d27
			ps359.OverlayValues[28] = d28
			ps359.OverlayValues[29] = d29
			ps359.OverlayValues[30] = d30
			ps359.OverlayValues[31] = d31
			ps359.OverlayValues[32] = d32
			ps359.OverlayValues[33] = d33
			ps359.OverlayValues[34] = d34
			ps359.OverlayValues[35] = d35
			ps359.OverlayValues[36] = d36
			ps359.OverlayValues[37] = d37
			ps359.OverlayValues[38] = d38
			ps359.OverlayValues[39] = d39
			ps359.OverlayValues[40] = d40
			ps359.OverlayValues[41] = d41
			ps359.OverlayValues[42] = d42
			ps359.OverlayValues[43] = d43
			ps359.OverlayValues[44] = d44
			ps359.OverlayValues[45] = d45
			ps359.OverlayValues[46] = d46
			ps359.OverlayValues[47] = d47
			ps359.OverlayValues[48] = d48
			ps359.OverlayValues[49] = d49
			ps359.OverlayValues[50] = d50
			ps359.OverlayValues[51] = d51
			ps359.OverlayValues[54] = d54
			ps359.OverlayValues[55] = d55
			ps359.OverlayValues[56] = d56
			ps359.OverlayValues[159] = d159
			ps359.OverlayValues[160] = d160
			ps359.OverlayValues[161] = d161
			ps359.OverlayValues[162] = d162
			ps359.OverlayValues[163] = d163
			ps359.OverlayValues[164] = d164
			ps359.OverlayValues[165] = d165
			ps359.OverlayValues[166] = d166
			ps359.OverlayValues[167] = d167
			ps359.OverlayValues[168] = d168
			ps359.OverlayValues[169] = d169
			ps359.OverlayValues[170] = d170
			ps359.OverlayValues[171] = d171
			ps359.OverlayValues[172] = d172
			ps359.OverlayValues[173] = d173
			ps359.OverlayValues[174] = d174
			ps359.OverlayValues[175] = d175
			ps359.OverlayValues[176] = d176
			ps359.OverlayValues[177] = d177
			ps359.OverlayValues[178] = d178
			ps359.OverlayValues[179] = d179
			ps359.OverlayValues[180] = d180
			ps359.OverlayValues[181] = d181
			ps359.OverlayValues[182] = d182
			ps359.OverlayValues[183] = d183
			ps359.OverlayValues[184] = d184
			ps359.OverlayValues[187] = d187
			ps359.OverlayValues[344] = d344
			ps359.OverlayValues[345] = d345
			ps359.OverlayValues[346] = d346
			ps359.OverlayValues[347] = d347
			ps359.OverlayValues[349] = d349
			ps359.OverlayValues[350] = d350
			ps359.OverlayValues[351] = d351
			ps359.OverlayValues[352] = d352
			ps359.OverlayValues[353] = d353
			ps359.OverlayValues[354] = d354
			ps359.OverlayValues[355] = d355
			ps359.OverlayValues[356] = d356
			ps359.OverlayValues[358] = d358
			return bbs[6].RenderPS(ps359)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d360 := ps.PhiValues[0]
				ctx.EnsureDesc(&d360)
				ctx.EmitStoreToStack(d360, int32(bbs[4].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d361 := ps.PhiValues[1]
				ctx.EnsureDesc(&d361)
				ctx.EmitStoreToStack(d361, int32(bbs[4].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d362 := ps.PhiValues[2]
				ctx.EnsureDesc(&d362)
				ctx.EmitStoreToStack(d362, int32(bbs[4].PhiBase)+int32(32))
			}
			ps.General = true
			return bbs[4].RenderPS(ps)
		}
		lbl17 := ctx.ReserveLabel()
		ctx.EmitJump(d354.Condition, lbl17)
		ctx.EmitJmp(lbl7)
		ctx.FreeDesc(&d353)
		snap363 := d5
		snap364 := d6
		snap365 := d7
		snap366 := d8
		snap367 := d9
		snap368 := d10
		snap369 := d11
		snap370 := d12
		snap371 := d13
		snap372 := d14
		snap373 := d15
		snap374 := d16
		snap375 := d17
		snap376 := d18
		snap377 := d19
		snap378 := d21
		snap379 := d22
		snap380 := d23
		snap381 := d24
		snap382 := d25
		snap383 := d26
		snap384 := d27
		snap385 := d28
		snap386 := d29
		snap387 := d30
		snap388 := d31
		snap389 := d32
		snap390 := d33
		snap391 := d34
		snap392 := d35
		snap393 := d36
		snap394 := d37
		snap395 := d38
		snap396 := d39
		snap397 := d40
		snap398 := d41
		snap399 := d42
		snap400 := d43
		snap401 := d44
		snap402 := d45
		snap403 := d46
		snap404 := d47
		snap405 := d48
		snap406 := d49
		snap407 := d50
		snap408 := d51
		snap409 := d54
		snap410 := d55
		snap411 := d56
		snap412 := d159
		snap413 := d160
		snap414 := d161
		snap415 := d162
		snap416 := d163
		snap417 := d164
		snap418 := d165
		snap419 := d166
		snap420 := d167
		snap421 := d168
		snap422 := d169
		snap423 := d170
		snap424 := d171
		snap425 := d172
		snap426 := d173
		snap427 := d174
		snap428 := d175
		snap429 := d176
		snap430 := d177
		snap431 := d178
		snap432 := d179
		snap433 := d180
		snap434 := d181
		snap435 := d182
		snap436 := d183
		snap437 := d184
		snap438 := d187
		snap439 := d344
		snap440 := d345
		snap441 := d346
		snap442 := d347
		snap443 := d349
		snap444 := d350
		snap445 := d351
		snap446 := d352
		snap447 := d353
		snap448 := d354
		snap449 := d355
		snap450 := d356
		snap451 := d358
		snap452 := d360
		snap453 := d361
		snap454 := d362
		alloc455 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl17)
		ctx.SyncDesc(&d10)
		if d10.Loc == scm.LocReg {
			ctx.ProtectReg(d10.Reg)
		} else if d10.Loc == scm.LocRegPair {
			ctx.ProtectReg(d10.Reg)
			ctx.ProtectReg(d10.Reg2)
		}
		d456 = d10
		if d456.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d456)
		d457 = d456
		if d457.Loc == scm.LocImm {
			d457 = scm.JITValueDesc{Loc: scm.LocImm, Type: d457.Type, Imm: scm.NewInt(int64(uint64(d457.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d457.Reg, 32)
			ctx.EmitShrRegImm8(d457.Reg, 32)
		}
		ctx.EmitStoreToStack(d457, int32(bbs[2].PhiBase)+int32(0))
		if d10.Loc == scm.LocReg {
			ctx.UnprotectReg(d10.Reg)
		} else if d10.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d10.Reg)
			ctx.UnprotectReg(d10.Reg2)
		}
		ctx.EmitJmp(lbl3)
		ctx.RestoreAllocState(alloc455)
		d5 = snap363
		d6 = snap364
		d7 = snap365
		d8 = snap366
		d9 = snap367
		d10 = snap368
		d11 = snap369
		d12 = snap370
		d13 = snap371
		d14 = snap372
		d15 = snap373
		d16 = snap374
		d17 = snap375
		d18 = snap376
		d19 = snap377
		d21 = snap378
		d22 = snap379
		d23 = snap380
		d24 = snap381
		d25 = snap382
		d26 = snap383
		d27 = snap384
		d28 = snap385
		d29 = snap386
		d30 = snap387
		d31 = snap388
		d32 = snap389
		d33 = snap390
		d34 = snap391
		d35 = snap392
		d36 = snap393
		d37 = snap394
		d38 = snap395
		d39 = snap396
		d40 = snap397
		d41 = snap398
		d42 = snap399
		d43 = snap400
		d44 = snap401
		d45 = snap402
		d46 = snap403
		d47 = snap404
		d48 = snap405
		d49 = snap406
		d50 = snap407
		d51 = snap408
		d54 = snap409
		d55 = snap410
		d56 = snap411
		d159 = snap412
		d160 = snap413
		d161 = snap414
		d162 = snap415
		d163 = snap416
		d164 = snap417
		d165 = snap418
		d166 = snap419
		d167 = snap420
		d168 = snap421
		d169 = snap422
		d170 = snap423
		d171 = snap424
		d172 = snap425
		d173 = snap426
		d174 = snap427
		d175 = snap428
		d176 = snap429
		d177 = snap430
		d178 = snap431
		d179 = snap432
		d180 = snap433
		d181 = snap434
		d182 = snap435
		d183 = snap436
		d184 = snap437
		d187 = snap438
		d344 = snap439
		d345 = snap440
		d346 = snap441
		d347 = snap442
		d349 = snap443
		d350 = snap444
		d351 = snap445
		d352 = snap446
		d353 = snap447
		d354 = snap448
		d355 = snap449
		d356 = snap450
		d358 = snap451
		d360 = snap452
		d361 = snap453
		d362 = snap454
		ctx.RestoreAllocState(alloc455)
		d5 = snap363
		d6 = snap364
		d7 = snap365
		d8 = snap366
		d9 = snap367
		d10 = snap368
		d11 = snap369
		d12 = snap370
		d13 = snap371
		d14 = snap372
		d15 = snap373
		d16 = snap374
		d17 = snap375
		d18 = snap376
		d19 = snap377
		d21 = snap378
		d22 = snap379
		d23 = snap380
		d24 = snap381
		d25 = snap382
		d26 = snap383
		d27 = snap384
		d28 = snap385
		d29 = snap386
		d30 = snap387
		d31 = snap388
		d32 = snap389
		d33 = snap390
		d34 = snap391
		d35 = snap392
		d36 = snap393
		d37 = snap394
		d38 = snap395
		d39 = snap396
		d40 = snap397
		d41 = snap398
		d42 = snap399
		d43 = snap400
		d44 = snap401
		d45 = snap402
		d46 = snap403
		d47 = snap404
		d48 = snap405
		d49 = snap406
		d50 = snap407
		d51 = snap408
		d54 = snap409
		d55 = snap410
		d56 = snap411
		d159 = snap412
		d160 = snap413
		d161 = snap414
		d162 = snap415
		d163 = snap416
		d164 = snap417
		d165 = snap418
		d166 = snap419
		d167 = snap420
		d168 = snap421
		d169 = snap422
		d170 = snap423
		d171 = snap424
		d172 = snap425
		d173 = snap426
		d174 = snap427
		d175 = snap428
		d176 = snap429
		d177 = snap430
		d178 = snap431
		d179 = snap432
		d180 = snap433
		d181 = snap434
		d182 = snap435
		d183 = snap436
		d184 = snap437
		d187 = snap438
		d344 = snap439
		d345 = snap440
		d346 = snap441
		d347 = snap442
		d349 = snap443
		d350 = snap444
		d351 = snap445
		d352 = snap446
		d353 = snap447
		d354 = snap448
		d355 = snap449
		d356 = snap450
		d358 = snap451
		d360 = snap452
		d361 = snap453
		d362 = snap454
		ps458 := scm.PhiState{General: true}
		ps458.OverlayValues = make([]scm.JITValueDesc, 458)
		ps458.OverlayValues[5] = d5
		ps458.OverlayValues[6] = d6
		ps458.OverlayValues[7] = d7
		ps458.OverlayValues[8] = d8
		ps458.OverlayValues[9] = d9
		ps458.OverlayValues[10] = d10
		ps458.OverlayValues[11] = d11
		ps458.OverlayValues[12] = d12
		ps458.OverlayValues[13] = d13
		ps458.OverlayValues[14] = d14
		ps458.OverlayValues[15] = d15
		ps458.OverlayValues[16] = d16
		ps458.OverlayValues[17] = d17
		ps458.OverlayValues[18] = d18
		ps458.OverlayValues[19] = d19
		ps458.OverlayValues[21] = d21
		ps458.OverlayValues[22] = d22
		ps458.OverlayValues[23] = d23
		ps458.OverlayValues[24] = d24
		ps458.OverlayValues[25] = d25
		ps458.OverlayValues[26] = d26
		ps458.OverlayValues[27] = d27
		ps458.OverlayValues[28] = d28
		ps458.OverlayValues[29] = d29
		ps458.OverlayValues[30] = d30
		ps458.OverlayValues[31] = d31
		ps458.OverlayValues[32] = d32
		ps458.OverlayValues[33] = d33
		ps458.OverlayValues[34] = d34
		ps458.OverlayValues[35] = d35
		ps458.OverlayValues[36] = d36
		ps458.OverlayValues[37] = d37
		ps458.OverlayValues[38] = d38
		ps458.OverlayValues[39] = d39
		ps458.OverlayValues[40] = d40
		ps458.OverlayValues[41] = d41
		ps458.OverlayValues[42] = d42
		ps458.OverlayValues[43] = d43
		ps458.OverlayValues[44] = d44
		ps458.OverlayValues[45] = d45
		ps458.OverlayValues[46] = d46
		ps458.OverlayValues[47] = d47
		ps458.OverlayValues[48] = d48
		ps458.OverlayValues[49] = d49
		ps458.OverlayValues[50] = d50
		ps458.OverlayValues[51] = d51
		ps458.OverlayValues[54] = d54
		ps458.OverlayValues[55] = d55
		ps458.OverlayValues[56] = d56
		ps458.OverlayValues[159] = d159
		ps458.OverlayValues[160] = d160
		ps458.OverlayValues[161] = d161
		ps458.OverlayValues[162] = d162
		ps458.OverlayValues[163] = d163
		ps458.OverlayValues[164] = d164
		ps458.OverlayValues[165] = d165
		ps458.OverlayValues[166] = d166
		ps458.OverlayValues[167] = d167
		ps458.OverlayValues[168] = d168
		ps458.OverlayValues[169] = d169
		ps458.OverlayValues[170] = d170
		ps458.OverlayValues[171] = d171
		ps458.OverlayValues[172] = d172
		ps458.OverlayValues[173] = d173
		ps458.OverlayValues[174] = d174
		ps458.OverlayValues[175] = d175
		ps458.OverlayValues[176] = d176
		ps458.OverlayValues[177] = d177
		ps458.OverlayValues[178] = d178
		ps458.OverlayValues[179] = d179
		ps458.OverlayValues[180] = d180
		ps458.OverlayValues[181] = d181
		ps458.OverlayValues[182] = d182
		ps458.OverlayValues[183] = d183
		ps458.OverlayValues[184] = d184
		ps458.OverlayValues[187] = d187
		ps458.OverlayValues[344] = d344
		ps458.OverlayValues[345] = d345
		ps458.OverlayValues[346] = d346
		ps458.OverlayValues[347] = d347
		ps458.OverlayValues[349] = d349
		ps458.OverlayValues[350] = d350
		ps458.OverlayValues[351] = d351
		ps458.OverlayValues[352] = d352
		ps458.OverlayValues[353] = d353
		ps458.OverlayValues[354] = d354
		ps458.OverlayValues[355] = d355
		ps458.OverlayValues[356] = d356
		ps458.OverlayValues[358] = d358
		ps458.OverlayValues[360] = d360
		ps458.OverlayValues[361] = d361
		ps458.OverlayValues[362] = d362
		ps458.OverlayValues[456] = d456
		ps458.OverlayValues[457] = d457
		ps458.PhiValues = make([]scm.JITValueDesc, 1)
		d460 = d10
		ps458.PhiValues[0] = d460
		ps459 := scm.PhiState{General: true}
		ps459.OverlayValues = make([]scm.JITValueDesc, 461)
		ps459.OverlayValues[5] = d5
		ps459.OverlayValues[6] = d6
		ps459.OverlayValues[7] = d7
		ps459.OverlayValues[8] = d8
		ps459.OverlayValues[9] = d9
		ps459.OverlayValues[10] = d10
		ps459.OverlayValues[11] = d11
		ps459.OverlayValues[12] = d12
		ps459.OverlayValues[13] = d13
		ps459.OverlayValues[14] = d14
		ps459.OverlayValues[15] = d15
		ps459.OverlayValues[16] = d16
		ps459.OverlayValues[17] = d17
		ps459.OverlayValues[18] = d18
		ps459.OverlayValues[19] = d19
		ps459.OverlayValues[21] = d21
		ps459.OverlayValues[22] = d22
		ps459.OverlayValues[23] = d23
		ps459.OverlayValues[24] = d24
		ps459.OverlayValues[25] = d25
		ps459.OverlayValues[26] = d26
		ps459.OverlayValues[27] = d27
		ps459.OverlayValues[28] = d28
		ps459.OverlayValues[29] = d29
		ps459.OverlayValues[30] = d30
		ps459.OverlayValues[31] = d31
		ps459.OverlayValues[32] = d32
		ps459.OverlayValues[33] = d33
		ps459.OverlayValues[34] = d34
		ps459.OverlayValues[35] = d35
		ps459.OverlayValues[36] = d36
		ps459.OverlayValues[37] = d37
		ps459.OverlayValues[38] = d38
		ps459.OverlayValues[39] = d39
		ps459.OverlayValues[40] = d40
		ps459.OverlayValues[41] = d41
		ps459.OverlayValues[42] = d42
		ps459.OverlayValues[43] = d43
		ps459.OverlayValues[44] = d44
		ps459.OverlayValues[45] = d45
		ps459.OverlayValues[46] = d46
		ps459.OverlayValues[47] = d47
		ps459.OverlayValues[48] = d48
		ps459.OverlayValues[49] = d49
		ps459.OverlayValues[50] = d50
		ps459.OverlayValues[51] = d51
		ps459.OverlayValues[54] = d54
		ps459.OverlayValues[55] = d55
		ps459.OverlayValues[56] = d56
		ps459.OverlayValues[159] = d159
		ps459.OverlayValues[160] = d160
		ps459.OverlayValues[161] = d161
		ps459.OverlayValues[162] = d162
		ps459.OverlayValues[163] = d163
		ps459.OverlayValues[164] = d164
		ps459.OverlayValues[165] = d165
		ps459.OverlayValues[166] = d166
		ps459.OverlayValues[167] = d167
		ps459.OverlayValues[168] = d168
		ps459.OverlayValues[169] = d169
		ps459.OverlayValues[170] = d170
		ps459.OverlayValues[171] = d171
		ps459.OverlayValues[172] = d172
		ps459.OverlayValues[173] = d173
		ps459.OverlayValues[174] = d174
		ps459.OverlayValues[175] = d175
		ps459.OverlayValues[176] = d176
		ps459.OverlayValues[177] = d177
		ps459.OverlayValues[178] = d178
		ps459.OverlayValues[179] = d179
		ps459.OverlayValues[180] = d180
		ps459.OverlayValues[181] = d181
		ps459.OverlayValues[182] = d182
		ps459.OverlayValues[183] = d183
		ps459.OverlayValues[184] = d184
		ps459.OverlayValues[187] = d187
		ps459.OverlayValues[344] = d344
		ps459.OverlayValues[345] = d345
		ps459.OverlayValues[346] = d346
		ps459.OverlayValues[347] = d347
		ps459.OverlayValues[349] = d349
		ps459.OverlayValues[350] = d350
		ps459.OverlayValues[351] = d351
		ps459.OverlayValues[352] = d352
		ps459.OverlayValues[353] = d353
		ps459.OverlayValues[354] = d354
		ps459.OverlayValues[355] = d355
		ps459.OverlayValues[356] = d356
		ps459.OverlayValues[358] = d358
		ps459.OverlayValues[360] = d360
		ps459.OverlayValues[361] = d361
		ps459.OverlayValues[362] = d362
		ps459.OverlayValues[456] = d456
		ps459.OverlayValues[457] = d457
		ps459.OverlayValues[460] = d460
		snap461 := d5
		snap462 := d6
		snap463 := d7
		snap464 := d8
		snap465 := d9
		snap466 := d10
		snap467 := d11
		snap468 := d12
		snap469 := d13
		snap470 := d14
		snap471 := d15
		snap472 := d16
		snap473 := d17
		snap474 := d18
		snap475 := d19
		snap476 := d21
		snap477 := d22
		snap478 := d23
		snap479 := d24
		snap480 := d25
		snap481 := d26
		snap482 := d27
		snap483 := d28
		snap484 := d29
		snap485 := d30
		snap486 := d31
		snap487 := d32
		snap488 := d33
		snap489 := d34
		snap490 := d35
		snap491 := d36
		snap492 := d37
		snap493 := d38
		snap494 := d39
		snap495 := d40
		snap496 := d41
		snap497 := d42
		snap498 := d43
		snap499 := d44
		snap500 := d45
		snap501 := d46
		snap502 := d47
		snap503 := d48
		snap504 := d49
		snap505 := d50
		snap506 := d51
		snap507 := d54
		snap508 := d55
		snap509 := d56
		snap510 := d159
		snap511 := d160
		snap512 := d161
		snap513 := d162
		snap514 := d163
		snap515 := d164
		snap516 := d165
		snap517 := d166
		snap518 := d167
		snap519 := d168
		snap520 := d169
		snap521 := d170
		snap522 := d171
		snap523 := d172
		snap524 := d173
		snap525 := d174
		snap526 := d175
		snap527 := d176
		snap528 := d177
		snap529 := d178
		snap530 := d179
		snap531 := d180
		snap532 := d181
		snap533 := d182
		snap534 := d183
		snap535 := d184
		snap536 := d187
		snap537 := d344
		snap538 := d345
		snap539 := d346
		snap540 := d347
		snap541 := d349
		snap542 := d350
		snap543 := d351
		snap544 := d352
		snap545 := d353
		snap546 := d354
		snap547 := d355
		snap548 := d356
		snap549 := d358
		snap550 := d360
		snap551 := d361
		snap552 := d362
		snap553 := d456
		snap554 := d457
		snap555 := d460
		alloc556 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps458)
		}
		ctx.RestoreAllocState(alloc556)
		d5 = snap461
		d6 = snap462
		d7 = snap463
		d8 = snap464
		d9 = snap465
		d10 = snap466
		d11 = snap467
		d12 = snap468
		d13 = snap469
		d14 = snap470
		d15 = snap471
		d16 = snap472
		d17 = snap473
		d18 = snap474
		d19 = snap475
		d21 = snap476
		d22 = snap477
		d23 = snap478
		d24 = snap479
		d25 = snap480
		d26 = snap481
		d27 = snap482
		d28 = snap483
		d29 = snap484
		d30 = snap485
		d31 = snap486
		d32 = snap487
		d33 = snap488
		d34 = snap489
		d35 = snap490
		d36 = snap491
		d37 = snap492
		d38 = snap493
		d39 = snap494
		d40 = snap495
		d41 = snap496
		d42 = snap497
		d43 = snap498
		d44 = snap499
		d45 = snap500
		d46 = snap501
		d47 = snap502
		d48 = snap503
		d49 = snap504
		d50 = snap505
		d51 = snap506
		d54 = snap507
		d55 = snap508
		d56 = snap509
		d159 = snap510
		d160 = snap511
		d161 = snap512
		d162 = snap513
		d163 = snap514
		d164 = snap515
		d165 = snap516
		d166 = snap517
		d167 = snap518
		d168 = snap519
		d169 = snap520
		d170 = snap521
		d171 = snap522
		d172 = snap523
		d173 = snap524
		d174 = snap525
		d175 = snap526
		d176 = snap527
		d177 = snap528
		d178 = snap529
		d179 = snap530
		d180 = snap531
		d181 = snap532
		d182 = snap533
		d183 = snap534
		d184 = snap535
		d187 = snap536
		d344 = snap537
		d345 = snap538
		d346 = snap539
		d347 = snap540
		d349 = snap541
		d350 = snap542
		d351 = snap543
		d352 = snap544
		d353 = snap545
		d354 = snap546
		d355 = snap547
		d356 = snap548
		d358 = snap549
		d360 = snap550
		d361 = snap551
		d362 = snap552
		d456 = snap553
		d457 = snap554
		d460 = snap555
		if !bbs[6].Rendered {
			return bbs[6].RenderPS(ps459)
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
		if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
			d187 = ps.OverlayValues[187]
		}
		if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != scm.LocNone {
			d344 = ps.OverlayValues[344]
		}
		if len(ps.OverlayValues) > 345 && ps.OverlayValues[345].Loc != scm.LocNone {
			d345 = ps.OverlayValues[345]
		}
		if len(ps.OverlayValues) > 346 && ps.OverlayValues[346].Loc != scm.LocNone {
			d346 = ps.OverlayValues[346]
		}
		if len(ps.OverlayValues) > 347 && ps.OverlayValues[347].Loc != scm.LocNone {
			d347 = ps.OverlayValues[347]
		}
		if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != scm.LocNone {
			d349 = ps.OverlayValues[349]
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
		if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != scm.LocNone {
			d355 = ps.OverlayValues[355]
		}
		if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != scm.LocNone {
			d356 = ps.OverlayValues[356]
		}
		if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != scm.LocNone {
			d358 = ps.OverlayValues[358]
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
		if len(ps.OverlayValues) > 456 && ps.OverlayValues[456].Loc != scm.LocNone {
			d456 = ps.OverlayValues[456]
		}
		if len(ps.OverlayValues) > 457 && ps.OverlayValues[457].Loc != scm.LocNone {
			d457 = ps.OverlayValues[457]
		}
		if len(ps.OverlayValues) > 460 && ps.OverlayValues[460].Loc != scm.LocNone {
			d460 = ps.OverlayValues[460]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d5)
		var d557 scm.JITValueDesc
		if d5.Loc == scm.LocImm {
			d557 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitAddRegImm32Low(scratch, int32(1))
			d557 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d557)
		}
		if d557.Loc == scm.LocReg && d5.Loc == scm.LocReg && d557.Reg == d5.Reg {
			ctx.TransferReg(d5.Reg)
			d5.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d557)
		ctx.EmitStoreToStack(d557, int32(bbs[4].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d557)
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
			d558 = d5
			if d558.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d558)
			d559 = d558
			if d559.Loc == scm.LocImm {
				d559 = scm.JITValueDesc{Loc: scm.LocImm, Type: d559.Type, Imm: scm.NewInt(int64(uint64(d559.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d559.Reg, 32)
				ctx.EmitShrRegImm8(d559.Reg, 32)
			}
			ctx.EmitStoreToStack(d559, int32(bbs[4].PhiBase)+int32(16))
			d560 = d7
			if d560.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d560)
			d561 = d560
			if d561.Loc == scm.LocImm {
				d561 = scm.JITValueDesc{Loc: scm.LocImm, Type: d561.Type, Imm: scm.NewInt(int64(uint64(d561.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d561.Reg, 32)
				ctx.EmitShrRegImm8(d561.Reg, 32)
			}
			ctx.EmitStoreToStack(d561, int32(bbs[4].PhiBase)+int32(32))
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
		ps562 := scm.PhiState{General: ps.General}
		ps562.OverlayValues = make([]scm.JITValueDesc, 562)
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
		ps562.OverlayValues[21] = d21
		ps562.OverlayValues[22] = d22
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
		ps562.OverlayValues[54] = d54
		ps562.OverlayValues[55] = d55
		ps562.OverlayValues[56] = d56
		ps562.OverlayValues[159] = d159
		ps562.OverlayValues[160] = d160
		ps562.OverlayValues[161] = d161
		ps562.OverlayValues[162] = d162
		ps562.OverlayValues[163] = d163
		ps562.OverlayValues[164] = d164
		ps562.OverlayValues[165] = d165
		ps562.OverlayValues[166] = d166
		ps562.OverlayValues[167] = d167
		ps562.OverlayValues[168] = d168
		ps562.OverlayValues[169] = d169
		ps562.OverlayValues[170] = d170
		ps562.OverlayValues[171] = d171
		ps562.OverlayValues[172] = d172
		ps562.OverlayValues[173] = d173
		ps562.OverlayValues[174] = d174
		ps562.OverlayValues[175] = d175
		ps562.OverlayValues[176] = d176
		ps562.OverlayValues[177] = d177
		ps562.OverlayValues[178] = d178
		ps562.OverlayValues[179] = d179
		ps562.OverlayValues[180] = d180
		ps562.OverlayValues[181] = d181
		ps562.OverlayValues[182] = d182
		ps562.OverlayValues[183] = d183
		ps562.OverlayValues[184] = d184
		ps562.OverlayValues[187] = d187
		ps562.OverlayValues[344] = d344
		ps562.OverlayValues[345] = d345
		ps562.OverlayValues[346] = d346
		ps562.OverlayValues[347] = d347
		ps562.OverlayValues[349] = d349
		ps562.OverlayValues[350] = d350
		ps562.OverlayValues[351] = d351
		ps562.OverlayValues[352] = d352
		ps562.OverlayValues[353] = d353
		ps562.OverlayValues[354] = d354
		ps562.OverlayValues[355] = d355
		ps562.OverlayValues[356] = d356
		ps562.OverlayValues[358] = d358
		ps562.OverlayValues[360] = d360
		ps562.OverlayValues[361] = d361
		ps562.OverlayValues[362] = d362
		ps562.OverlayValues[456] = d456
		ps562.OverlayValues[457] = d457
		ps562.OverlayValues[460] = d460
		ps562.OverlayValues[557] = d557
		ps562.OverlayValues[558] = d558
		ps562.OverlayValues[559] = d559
		ps562.OverlayValues[560] = d560
		ps562.OverlayValues[561] = d561
		ps562.PhiValues = make([]scm.JITValueDesc, 3)
		d563 = d5
		ps562.PhiValues[1] = d563
		d564 = d7
		ps562.PhiValues[2] = d564
		if ps562.General && bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
			return result
		}
		return bbs[4].RenderPS(ps562)
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
		if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
			d187 = ps.OverlayValues[187]
		}
		if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != scm.LocNone {
			d344 = ps.OverlayValues[344]
		}
		if len(ps.OverlayValues) > 345 && ps.OverlayValues[345].Loc != scm.LocNone {
			d345 = ps.OverlayValues[345]
		}
		if len(ps.OverlayValues) > 346 && ps.OverlayValues[346].Loc != scm.LocNone {
			d346 = ps.OverlayValues[346]
		}
		if len(ps.OverlayValues) > 347 && ps.OverlayValues[347].Loc != scm.LocNone {
			d347 = ps.OverlayValues[347]
		}
		if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != scm.LocNone {
			d349 = ps.OverlayValues[349]
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
		if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != scm.LocNone {
			d355 = ps.OverlayValues[355]
		}
		if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != scm.LocNone {
			d356 = ps.OverlayValues[356]
		}
		if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != scm.LocNone {
			d358 = ps.OverlayValues[358]
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
		if len(ps.OverlayValues) > 456 && ps.OverlayValues[456].Loc != scm.LocNone {
			d456 = ps.OverlayValues[456]
		}
		if len(ps.OverlayValues) > 457 && ps.OverlayValues[457].Loc != scm.LocNone {
			d457 = ps.OverlayValues[457]
		}
		if len(ps.OverlayValues) > 460 && ps.OverlayValues[460].Loc != scm.LocNone {
			d460 = ps.OverlayValues[460]
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
		if len(ps.OverlayValues) > 561 && ps.OverlayValues[561].Loc != scm.LocNone {
			d561 = ps.OverlayValues[561]
		}
		if len(ps.OverlayValues) > 563 && ps.OverlayValues[563].Loc != scm.LocNone {
			d563 = ps.OverlayValues[563]
		}
		if len(ps.OverlayValues) > 564 && ps.OverlayValues[564].Loc != scm.LocNone {
			d564 = ps.OverlayValues[564]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d9)
		d565 = d9
		_ = d565
		ctx.StabilizeDescForControlFlow(&d9)
		bbpos_3_0 := int32(-1)
		_ = bbpos_3_0
		lbl18 := ctx.ReserveLabel()
		_ = lbl18
		bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl18)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d566 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d566 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48)
			r69 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r69, thisptr.Reg, off)
			d566 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r69}
			ctx.BindReg(r69, &d566)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d566)
		ctx.EnsureDesc(&d566)
		var d567 scm.JITValueDesc
		if d566.Loc == scm.LocImm {
			d567 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d566.Imm.Int()))))}
		} else {
			r70 := ctx.AllocReg()
			ctx.EmitMovRegReg(r70, d566.Reg)
			ctx.EmitShlRegImm8(r70, 56)
			ctx.EmitShrRegImm8(r70, 56)
			d567 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r70}
			ctx.BindReg(r70, &d567)
		}
		ctx.FreeDesc(&d566)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d565)
		ctx.EnsureDesc(&d565)
		var d568 scm.JITValueDesc
		if d565.Loc == scm.LocImm {
			d568 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d565.Imm.Int()))))}
		} else {
			r71 := ctx.AllocReg()
			ctx.EmitMovRegReg(r71, d565.Reg)
			ctx.EmitShlRegImm8(r71, 32)
			ctx.EmitShrRegImm8(r71, 32)
			d568 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r71}
			ctx.BindReg(r71, &d568)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d568)
		ctx.EnsureDesc(&d567)
		ctx.EnsureDescsTogether(&d568, &d567)
		var d569 scm.JITValueDesc
		if d568.Loc == scm.LocImm && d567.Loc == scm.LocImm {
			d569 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d568.Imm.Int() * d567.Imm.Int())}
		} else if d568.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d567.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d568.Imm.Int()))
			ctx.EmitImulInt64(scratch, d567.Reg)
			d569 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d569)
		} else if d567.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d568.Reg)
			ctx.EmitMovRegReg(scratch, d568.Reg)
			if d567.Imm.Int() >= -2147483648 && d567.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d567.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d567.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d569 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d569)
		} else {
			r72 := ctx.AllocRegExcept(d568.Reg, d567.Reg)
			ctx.EmitMovRegReg(r72, d568.Reg)
			ctx.EmitImulInt64(r72, d567.Reg)
			d569 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r72}
			ctx.BindReg(r72, &d569)
		}
		if d569.Loc == scm.LocReg && d568.Loc == scm.LocReg && d569.Reg == d568.Reg {
			ctx.TransferReg(d568.Reg)
			d568.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d568)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d569)
		var d570 scm.JITValueDesc
		if d569.Loc == scm.LocImm {
			d570 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d569.Imm.Int() / 64)}
		} else {
			r73 := ctx.AllocRegExcept(d569.Reg)
			ctx.EmitMovRegReg(r73, d569.Reg)
			ctx.EmitShrRegImm8(r73, 6)
			d570 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
			ctx.BindReg(r73, &d570)
		}
		if d570.Loc == scm.LocReg && d569.Loc == scm.LocReg && d570.Reg == d569.Reg {
			ctx.TransferReg(d569.Reg)
			d569.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d569)
		var d571 scm.JITValueDesc
		if d569.Loc == scm.LocImm {
			d571 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d569.Imm.Int() % 64)}
		} else {
			r74 := ctx.AllocRegExcept(d569.Reg)
			ctx.EmitMovRegReg(r74, d569.Reg)
			ctx.EmitAndRegImm32(r74, 63)
			d571 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r74}
			ctx.BindReg(r74, &d571)
		}
		if d571.Loc == scm.LocReg && d569.Loc == scm.LocReg && d571.Reg == d569.Reg {
			ctx.TransferReg(d569.Reg)
			d569.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d569)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d572 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d572 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r75 := ctx.AllocReg()
			r76 := ctx.AllocRegExcept(r75)
			r77 := ctx.AllocRegExcept(r75, r76)
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			ctx.EmitMovRegMem(r75, thisptr.Reg, off)
			ctx.EmitMovRegMem(r76, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r77, thisptr.Reg, off+16)
			d572 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r75, Reg2: r76, Reg3: r77}
			ctx.BindReg(r75, &d572)
			ctx.BindReg(r76, &d572)
			ctx.BindReg(r77, &d572)
			ctx.BindReg(r75, &d572)
			ctx.BindReg(r76, &d572)
			ctx.BindReg(r77, &d572)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d570)
		ctx.ReclaimUntrackedRegs()
		d573 = ctx.EmitLoadScalarSliceElement(&d572, &d570, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d573)
		ctx.EnsureDesc(&d571)
		ctx.EnsureDescsTogether(&d573, &d571)
		var d574 scm.JITValueDesc
		if d573.Loc == scm.LocImm && d571.Loc == scm.LocImm {
			d574 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d573.Imm.Int()) << uint64(d571.Imm.Int())))}
		} else if d571.Loc == scm.LocImm {
			r78 := ctx.AllocRegExcept(d573.Reg)
			ctx.EmitMovRegReg(r78, d573.Reg)
			ctx.EmitShlRegImm8(r78, uint8(d571.Imm.Int()))
			d574 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r78}
			ctx.BindReg(r78, &d574)
		} else {
			{
				shiftSrc := d573.Reg
				r79 := ctx.AllocRegExcept(d573.Reg, d571.Reg)
				ctx.EmitMovRegReg(r79, d573.Reg)
				shiftSrc = r79
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d571.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d571.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d571.Reg)
				}
				ctx.EmitShlRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d574 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d574)
			}
		}
		if d574.Loc == scm.LocReg && d573.Loc == scm.LocReg && d574.Reg == d573.Reg {
			ctx.TransferReg(d573.Reg)
			d573.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d573)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d570)
		ctx.EnsureDesc(&d570)
		var d575 scm.JITValueDesc
		if d570.Loc == scm.LocImm {
			d575 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d570.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d570.Reg)
			ctx.EmitMovRegReg(scratch, d570.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d575 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d575)
		}
		if d575.Loc == scm.LocReg && d570.Loc == scm.LocReg && d575.Reg == d570.Reg {
			ctx.TransferReg(d570.Reg)
			d570.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d570)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d575)
		ctx.ReclaimUntrackedRegs()
		d576 = ctx.EmitLoadScalarSliceElement(&d572, &d575, 8, scm.TagInt)
		ctx.FreeDesc(&d575)
		ctx.ReclaimUntrackedRegs()
		d577 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d571)
		ctx.EnsureDescsTogether(&d577, &d571)
		var d578 scm.JITValueDesc
		if d577.Loc == scm.LocImm && d571.Loc == scm.LocImm {
			d578 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d577.Imm.Int() - d571.Imm.Int())}
		} else if d571.Loc == scm.LocImm && d571.Imm.Int() == 0 {
			r80 := ctx.AllocRegExcept(d577.Reg)
			ctx.EmitMovRegReg(r80, d577.Reg)
			d578 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
			ctx.BindReg(r80, &d578)
		} else if d577.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d571.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d577.Imm.Int()))
			ctx.EmitSubInt64(scratch, d571.Reg)
			d578 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d578)
		} else if d571.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d577.Reg)
			ctx.EmitMovRegReg(scratch, d577.Reg)
			if d571.Imm.Int() >= -2147483648 && d571.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d571.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d571.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d578 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d578)
		} else {
			r81 := ctx.AllocRegExcept(d577.Reg, d571.Reg)
			ctx.EmitMovRegReg(r81, d577.Reg)
			ctx.EmitSubInt64(r81, d571.Reg)
			d578 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
			ctx.BindReg(r81, &d578)
		}
		if d578.Loc == scm.LocReg && d577.Loc == scm.LocReg && d578.Reg == d577.Reg {
			ctx.TransferReg(d577.Reg)
			d577.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d571)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d576)
		ctx.EnsureDesc(&d578)
		ctx.EnsureDescsTogether(&d576, &d578)
		var d579 scm.JITValueDesc
		if d576.Loc == scm.LocImm && d578.Loc == scm.LocImm {
			d579 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d576.Imm.Int()) >> uint64(d578.Imm.Int())))}
		} else if d578.Loc == scm.LocImm {
			r82 := ctx.AllocRegExcept(d576.Reg)
			ctx.EmitMovRegReg(r82, d576.Reg)
			ctx.EmitShrRegImm8(r82, uint8(d578.Imm.Int()))
			d579 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r82}
			ctx.BindReg(r82, &d579)
		} else {
			{
				shiftSrc := d576.Reg
				r83 := ctx.AllocRegExcept(d576.Reg, d578.Reg)
				ctx.EmitMovRegReg(r83, d576.Reg)
				shiftSrc = r83
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d578.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d578.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d578.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d579 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d579)
			}
		}
		if d579.Loc == scm.LocReg && d576.Loc == scm.LocReg && d579.Reg == d576.Reg {
			ctx.TransferReg(d576.Reg)
			d576.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d576)
		ctx.FreeDesc(&d578)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d574)
		ctx.EnsureDesc(&d579)
		var d580 scm.JITValueDesc
		if d574.Loc == scm.LocImm && d579.Loc == scm.LocImm {
			d580 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d574.Imm.Int() | d579.Imm.Int())}
		} else if d574.Loc == scm.LocImm && d574.Imm.Int() == 0 {
			d580 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d579.Reg}
			ctx.BindReg(d579.Reg, &d580)
		} else if d579.Loc == scm.LocImm && d579.Imm.Int() == 0 {
			r84 := ctx.AllocRegExcept(d574.Reg)
			ctx.EmitMovRegReg(r84, d574.Reg)
			d580 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r84}
			ctx.BindReg(r84, &d580)
		} else if d574.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d579.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d574.Imm.Int()))
			ctx.EmitOrInt64(scratch, d579.Reg)
			d580 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d580)
		} else if d579.Loc == scm.LocImm {
			r85 := ctx.AllocRegExcept(d574.Reg)
			ctx.EmitMovRegReg(r85, d574.Reg)
			if d579.Imm.Int() >= -2147483648 && d579.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r85, int32(d579.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d579.Imm.Int()))
				ctx.EmitOrInt64(r85, scm.RegR11)
			}
			d580 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
			ctx.BindReg(r85, &d580)
		} else {
			r86 := ctx.AllocRegExcept(d574.Reg, d579.Reg)
			ctx.EmitMovRegReg(r86, d574.Reg)
			ctx.EmitOrInt64(r86, d579.Reg)
			d580 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
			ctx.BindReg(r86, &d580)
		}
		if d580.Loc == scm.LocReg && d574.Loc == scm.LocReg && d580.Reg == d574.Reg {
			ctx.TransferReg(d574.Reg)
			d574.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d574)
		ctx.FreeDesc(&d579)
		ctx.ReclaimUntrackedRegs()
		d581 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d567)
		ctx.EnsureDescsTogether(&d581, &d567)
		var d582 scm.JITValueDesc
		if d581.Loc == scm.LocImm && d567.Loc == scm.LocImm {
			d582 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d581.Imm.Int() - d567.Imm.Int())}
		} else if d567.Loc == scm.LocImm && d567.Imm.Int() == 0 {
			r87 := ctx.AllocRegExcept(d581.Reg)
			ctx.EmitMovRegReg(r87, d581.Reg)
			d582 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r87}
			ctx.BindReg(r87, &d582)
		} else if d581.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d567.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d581.Imm.Int()))
			ctx.EmitSubInt64(scratch, d567.Reg)
			d582 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d582)
		} else if d567.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d581.Reg)
			ctx.EmitMovRegReg(scratch, d581.Reg)
			if d567.Imm.Int() >= -2147483648 && d567.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d567.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d567.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d582 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d582)
		} else {
			r88 := ctx.AllocRegExcept(d581.Reg, d567.Reg)
			ctx.EmitMovRegReg(r88, d581.Reg)
			ctx.EmitSubInt64(r88, d567.Reg)
			d582 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
			ctx.BindReg(r88, &d582)
		}
		if d582.Loc == scm.LocReg && d581.Loc == scm.LocReg && d582.Reg == d581.Reg {
			ctx.TransferReg(d581.Reg)
			d581.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d567)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d580)
		ctx.EnsureDesc(&d582)
		ctx.EnsureDescsTogether(&d580, &d582)
		var d583 scm.JITValueDesc
		if d580.Loc == scm.LocImm && d582.Loc == scm.LocImm {
			d583 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d580.Imm.Int()) >> uint64(d582.Imm.Int())))}
		} else if d582.Loc == scm.LocImm {
			r89 := ctx.AllocRegExcept(d580.Reg)
			ctx.EmitMovRegReg(r89, d580.Reg)
			ctx.EmitShrRegImm8(r89, uint8(d582.Imm.Int()))
			d583 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
			ctx.BindReg(r89, &d583)
		} else {
			{
				shiftSrc := d580.Reg
				r90 := ctx.AllocRegExcept(d580.Reg, d582.Reg)
				ctx.EmitMovRegReg(r90, d580.Reg)
				shiftSrc = r90
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d582.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d582.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d582.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d583 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d583)
			}
		}
		if d583.Loc == scm.LocReg && d580.Loc == scm.LocReg && d583.Reg == d580.Reg {
			ctx.TransferReg(d580.Reg)
			d580.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d580)
		ctx.FreeDesc(&d582)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d583)
		ctx.EnsureDesc(&d583)
		ctx.EnsureDesc(&d583)
		var d584 scm.JITValueDesc
		if d583.Loc == scm.LocImm {
			d584 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d583.Imm.Int()))))}
		} else {
			r91 := ctx.AllocReg()
			ctx.EmitMovRegReg(r91, d583.Reg)
			d584 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r91}
			ctx.BindReg(r91, &d584)
		}
		ctx.FreeDesc(&d583)
		var d585 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d585 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56)
			r92 := ctx.AllocReg()
			ctx.EmitMovRegMem(r92, thisptr.Reg, off)
			d585 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r92}
			ctx.BindReg(r92, &d585)
		}
		ctx.EnsureDesc(&d584)
		ctx.EnsureDesc(&d585)
		ctx.EnsureDescsTogether(&d584, &d585)
		var d586 scm.JITValueDesc
		if d584.Loc == scm.LocImm && d585.Loc == scm.LocImm {
			d586 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d584.Imm.Int() + d585.Imm.Int())}
		} else if d585.Loc == scm.LocImm && d585.Imm.Int() == 0 {
			r93 := ctx.AllocRegExcept(d584.Reg)
			ctx.EmitMovRegReg(r93, d584.Reg)
			d586 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r93}
			ctx.BindReg(r93, &d586)
		} else if d584.Loc == scm.LocImm && d584.Imm.Int() == 0 {
			d586 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d585.Reg}
			ctx.BindReg(d585.Reg, &d586)
		} else if d584.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d585.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d584.Imm.Int()))
			ctx.EmitAddInt64(scratch, d585.Reg)
			d586 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d586)
		} else if d585.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d584.Reg)
			ctx.EmitMovRegReg(scratch, d584.Reg)
			if d585.Imm.Int() >= -2147483648 && d585.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d585.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d585.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d586 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d586)
		} else {
			r94 := ctx.AllocRegExcept(d584.Reg, d585.Reg)
			ctx.EmitMovRegReg(r94, d584.Reg)
			ctx.EmitAddInt64(r94, d585.Reg)
			d586 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
			ctx.BindReg(r94, &d586)
		}
		if d586.Loc == scm.LocReg && d584.Loc == scm.LocReg && d586.Reg == d584.Reg {
			ctx.TransferReg(d584.Reg)
			d584.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d584)
		ctx.FreeDesc(&d585)
		ctx.EnsureDesc(&d586)
		ctx.EnsureDesc(&d586)
		var d587 scm.JITValueDesc
		if d586.Loc == scm.LocImm {
			d587 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d586.Imm.Int()))))}
		} else {
			r95 := ctx.AllocReg()
			ctx.EmitMovRegReg(r95, d586.Reg)
			ctx.EmitShlRegImm8(r95, 32)
			ctx.EmitShrRegImm8(r95, 32)
			d587 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r95}
			ctx.BindReg(r95, &d587)
		}
		ctx.FreeDesc(&d586)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d587)
		ctx.EnsureDescsTogether(&idxInt, &d587)
		var d588 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm && d587.Loc == scm.LocImm {
			d588 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d587.Imm.Int()))}
		} else if d587.Loc == scm.LocImm {
			r96 := ctx.AllocRegExcept(idxInt.Reg)
			if d587.Imm.Int() >= -2147483648 && d587.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(idxInt.Reg, int32(d587.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d587.Imm.Int()))
				ctx.EmitCmpInt64(idxInt.Reg, scm.RegR11)
			}
			d588 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r96, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r96, &d588)
		} else if idxInt.Loc == scm.LocImm {
			r97 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d587.Reg)
			d588 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r97, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r97, &d588)
		} else {
			r98 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitCmpInt64(idxInt.Reg, d587.Reg)
			d588 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r98, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r98, &d588)
		}
		ctx.FreeDesc(&d587)
		d589 = d588
		ctx.EnsureDesc(&d589)
		if d589.Loc != scm.LocImm && d589.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d589.Loc == scm.LocImm {
			if d589.Imm.Bool() {
				if ps.General {
				}
				ps590 := scm.PhiState{General: ps.General}
				ps590.OverlayValues = make([]scm.JITValueDesc, 590)
				ps590.OverlayValues[5] = d5
				ps590.OverlayValues[6] = d6
				ps590.OverlayValues[7] = d7
				ps590.OverlayValues[8] = d8
				ps590.OverlayValues[9] = d9
				ps590.OverlayValues[10] = d10
				ps590.OverlayValues[11] = d11
				ps590.OverlayValues[12] = d12
				ps590.OverlayValues[13] = d13
				ps590.OverlayValues[14] = d14
				ps590.OverlayValues[15] = d15
				ps590.OverlayValues[16] = d16
				ps590.OverlayValues[17] = d17
				ps590.OverlayValues[18] = d18
				ps590.OverlayValues[19] = d19
				ps590.OverlayValues[21] = d21
				ps590.OverlayValues[22] = d22
				ps590.OverlayValues[23] = d23
				ps590.OverlayValues[24] = d24
				ps590.OverlayValues[25] = d25
				ps590.OverlayValues[26] = d26
				ps590.OverlayValues[27] = d27
				ps590.OverlayValues[28] = d28
				ps590.OverlayValues[29] = d29
				ps590.OverlayValues[30] = d30
				ps590.OverlayValues[31] = d31
				ps590.OverlayValues[32] = d32
				ps590.OverlayValues[33] = d33
				ps590.OverlayValues[34] = d34
				ps590.OverlayValues[35] = d35
				ps590.OverlayValues[36] = d36
				ps590.OverlayValues[37] = d37
				ps590.OverlayValues[38] = d38
				ps590.OverlayValues[39] = d39
				ps590.OverlayValues[40] = d40
				ps590.OverlayValues[41] = d41
				ps590.OverlayValues[42] = d42
				ps590.OverlayValues[43] = d43
				ps590.OverlayValues[44] = d44
				ps590.OverlayValues[45] = d45
				ps590.OverlayValues[46] = d46
				ps590.OverlayValues[47] = d47
				ps590.OverlayValues[48] = d48
				ps590.OverlayValues[49] = d49
				ps590.OverlayValues[50] = d50
				ps590.OverlayValues[51] = d51
				ps590.OverlayValues[54] = d54
				ps590.OverlayValues[55] = d55
				ps590.OverlayValues[56] = d56
				ps590.OverlayValues[159] = d159
				ps590.OverlayValues[160] = d160
				ps590.OverlayValues[161] = d161
				ps590.OverlayValues[162] = d162
				ps590.OverlayValues[163] = d163
				ps590.OverlayValues[164] = d164
				ps590.OverlayValues[165] = d165
				ps590.OverlayValues[166] = d166
				ps590.OverlayValues[167] = d167
				ps590.OverlayValues[168] = d168
				ps590.OverlayValues[169] = d169
				ps590.OverlayValues[170] = d170
				ps590.OverlayValues[171] = d171
				ps590.OverlayValues[172] = d172
				ps590.OverlayValues[173] = d173
				ps590.OverlayValues[174] = d174
				ps590.OverlayValues[175] = d175
				ps590.OverlayValues[176] = d176
				ps590.OverlayValues[177] = d177
				ps590.OverlayValues[178] = d178
				ps590.OverlayValues[179] = d179
				ps590.OverlayValues[180] = d180
				ps590.OverlayValues[181] = d181
				ps590.OverlayValues[182] = d182
				ps590.OverlayValues[183] = d183
				ps590.OverlayValues[184] = d184
				ps590.OverlayValues[187] = d187
				ps590.OverlayValues[344] = d344
				ps590.OverlayValues[345] = d345
				ps590.OverlayValues[346] = d346
				ps590.OverlayValues[347] = d347
				ps590.OverlayValues[349] = d349
				ps590.OverlayValues[350] = d350
				ps590.OverlayValues[351] = d351
				ps590.OverlayValues[352] = d352
				ps590.OverlayValues[353] = d353
				ps590.OverlayValues[354] = d354
				ps590.OverlayValues[355] = d355
				ps590.OverlayValues[356] = d356
				ps590.OverlayValues[358] = d358
				ps590.OverlayValues[360] = d360
				ps590.OverlayValues[361] = d361
				ps590.OverlayValues[362] = d362
				ps590.OverlayValues[456] = d456
				ps590.OverlayValues[457] = d457
				ps590.OverlayValues[460] = d460
				ps590.OverlayValues[557] = d557
				ps590.OverlayValues[558] = d558
				ps590.OverlayValues[559] = d559
				ps590.OverlayValues[560] = d560
				ps590.OverlayValues[561] = d561
				ps590.OverlayValues[563] = d563
				ps590.OverlayValues[564] = d564
				ps590.OverlayValues[565] = d565
				ps590.OverlayValues[566] = d566
				ps590.OverlayValues[567] = d567
				ps590.OverlayValues[568] = d568
				ps590.OverlayValues[569] = d569
				ps590.OverlayValues[570] = d570
				ps590.OverlayValues[571] = d571
				ps590.OverlayValues[572] = d572
				ps590.OverlayValues[573] = d573
				ps590.OverlayValues[574] = d574
				ps590.OverlayValues[575] = d575
				ps590.OverlayValues[576] = d576
				ps590.OverlayValues[577] = d577
				ps590.OverlayValues[578] = d578
				ps590.OverlayValues[579] = d579
				ps590.OverlayValues[580] = d580
				ps590.OverlayValues[581] = d581
				ps590.OverlayValues[582] = d582
				ps590.OverlayValues[583] = d583
				ps590.OverlayValues[584] = d584
				ps590.OverlayValues[585] = d585
				ps590.OverlayValues[586] = d586
				ps590.OverlayValues[587] = d587
				ps590.OverlayValues[588] = d588
				ps590.OverlayValues[589] = d589
				return bbs[7].RenderPS(ps590)
			}
			if ps.General {
			}
			ps591 := scm.PhiState{General: ps.General}
			ps591.OverlayValues = make([]scm.JITValueDesc, 590)
			ps591.OverlayValues[5] = d5
			ps591.OverlayValues[6] = d6
			ps591.OverlayValues[7] = d7
			ps591.OverlayValues[8] = d8
			ps591.OverlayValues[9] = d9
			ps591.OverlayValues[10] = d10
			ps591.OverlayValues[11] = d11
			ps591.OverlayValues[12] = d12
			ps591.OverlayValues[13] = d13
			ps591.OverlayValues[14] = d14
			ps591.OverlayValues[15] = d15
			ps591.OverlayValues[16] = d16
			ps591.OverlayValues[17] = d17
			ps591.OverlayValues[18] = d18
			ps591.OverlayValues[19] = d19
			ps591.OverlayValues[21] = d21
			ps591.OverlayValues[22] = d22
			ps591.OverlayValues[23] = d23
			ps591.OverlayValues[24] = d24
			ps591.OverlayValues[25] = d25
			ps591.OverlayValues[26] = d26
			ps591.OverlayValues[27] = d27
			ps591.OverlayValues[28] = d28
			ps591.OverlayValues[29] = d29
			ps591.OverlayValues[30] = d30
			ps591.OverlayValues[31] = d31
			ps591.OverlayValues[32] = d32
			ps591.OverlayValues[33] = d33
			ps591.OverlayValues[34] = d34
			ps591.OverlayValues[35] = d35
			ps591.OverlayValues[36] = d36
			ps591.OverlayValues[37] = d37
			ps591.OverlayValues[38] = d38
			ps591.OverlayValues[39] = d39
			ps591.OverlayValues[40] = d40
			ps591.OverlayValues[41] = d41
			ps591.OverlayValues[42] = d42
			ps591.OverlayValues[43] = d43
			ps591.OverlayValues[44] = d44
			ps591.OverlayValues[45] = d45
			ps591.OverlayValues[46] = d46
			ps591.OverlayValues[47] = d47
			ps591.OverlayValues[48] = d48
			ps591.OverlayValues[49] = d49
			ps591.OverlayValues[50] = d50
			ps591.OverlayValues[51] = d51
			ps591.OverlayValues[54] = d54
			ps591.OverlayValues[55] = d55
			ps591.OverlayValues[56] = d56
			ps591.OverlayValues[159] = d159
			ps591.OverlayValues[160] = d160
			ps591.OverlayValues[161] = d161
			ps591.OverlayValues[162] = d162
			ps591.OverlayValues[163] = d163
			ps591.OverlayValues[164] = d164
			ps591.OverlayValues[165] = d165
			ps591.OverlayValues[166] = d166
			ps591.OverlayValues[167] = d167
			ps591.OverlayValues[168] = d168
			ps591.OverlayValues[169] = d169
			ps591.OverlayValues[170] = d170
			ps591.OverlayValues[171] = d171
			ps591.OverlayValues[172] = d172
			ps591.OverlayValues[173] = d173
			ps591.OverlayValues[174] = d174
			ps591.OverlayValues[175] = d175
			ps591.OverlayValues[176] = d176
			ps591.OverlayValues[177] = d177
			ps591.OverlayValues[178] = d178
			ps591.OverlayValues[179] = d179
			ps591.OverlayValues[180] = d180
			ps591.OverlayValues[181] = d181
			ps591.OverlayValues[182] = d182
			ps591.OverlayValues[183] = d183
			ps591.OverlayValues[184] = d184
			ps591.OverlayValues[187] = d187
			ps591.OverlayValues[344] = d344
			ps591.OverlayValues[345] = d345
			ps591.OverlayValues[346] = d346
			ps591.OverlayValues[347] = d347
			ps591.OverlayValues[349] = d349
			ps591.OverlayValues[350] = d350
			ps591.OverlayValues[351] = d351
			ps591.OverlayValues[352] = d352
			ps591.OverlayValues[353] = d353
			ps591.OverlayValues[354] = d354
			ps591.OverlayValues[355] = d355
			ps591.OverlayValues[356] = d356
			ps591.OverlayValues[358] = d358
			ps591.OverlayValues[360] = d360
			ps591.OverlayValues[361] = d361
			ps591.OverlayValues[362] = d362
			ps591.OverlayValues[456] = d456
			ps591.OverlayValues[457] = d457
			ps591.OverlayValues[460] = d460
			ps591.OverlayValues[557] = d557
			ps591.OverlayValues[558] = d558
			ps591.OverlayValues[559] = d559
			ps591.OverlayValues[560] = d560
			ps591.OverlayValues[561] = d561
			ps591.OverlayValues[563] = d563
			ps591.OverlayValues[564] = d564
			ps591.OverlayValues[565] = d565
			ps591.OverlayValues[566] = d566
			ps591.OverlayValues[567] = d567
			ps591.OverlayValues[568] = d568
			ps591.OverlayValues[569] = d569
			ps591.OverlayValues[570] = d570
			ps591.OverlayValues[571] = d571
			ps591.OverlayValues[572] = d572
			ps591.OverlayValues[573] = d573
			ps591.OverlayValues[574] = d574
			ps591.OverlayValues[575] = d575
			ps591.OverlayValues[576] = d576
			ps591.OverlayValues[577] = d577
			ps591.OverlayValues[578] = d578
			ps591.OverlayValues[579] = d579
			ps591.OverlayValues[580] = d580
			ps591.OverlayValues[581] = d581
			ps591.OverlayValues[582] = d582
			ps591.OverlayValues[583] = d583
			ps591.OverlayValues[584] = d584
			ps591.OverlayValues[585] = d585
			ps591.OverlayValues[586] = d586
			ps591.OverlayValues[587] = d587
			ps591.OverlayValues[588] = d588
			ps591.OverlayValues[589] = d589
			return bbs[9].RenderPS(ps591)
		}
		if !ps.General {
			ps.General = true
			return bbs[6].RenderPS(ps)
		}
		ctx.EmitJump(d589.Condition, lbl8)
		ctx.FreeDesc(&d588)
		snap592 := d5
		snap593 := d6
		snap594 := d7
		snap595 := d8
		snap596 := d9
		snap597 := d10
		snap598 := d11
		snap599 := d12
		snap600 := d13
		snap601 := d14
		snap602 := d15
		snap603 := d16
		snap604 := d17
		snap605 := d18
		snap606 := d19
		snap607 := d21
		snap608 := d22
		snap609 := d23
		snap610 := d24
		snap611 := d25
		snap612 := d26
		snap613 := d27
		snap614 := d28
		snap615 := d29
		snap616 := d30
		snap617 := d31
		snap618 := d32
		snap619 := d33
		snap620 := d34
		snap621 := d35
		snap622 := d36
		snap623 := d37
		snap624 := d38
		snap625 := d39
		snap626 := d40
		snap627 := d41
		snap628 := d42
		snap629 := d43
		snap630 := d44
		snap631 := d45
		snap632 := d46
		snap633 := d47
		snap634 := d48
		snap635 := d49
		snap636 := d50
		snap637 := d51
		snap638 := d54
		snap639 := d55
		snap640 := d56
		snap641 := d159
		snap642 := d160
		snap643 := d161
		snap644 := d162
		snap645 := d163
		snap646 := d164
		snap647 := d165
		snap648 := d166
		snap649 := d167
		snap650 := d168
		snap651 := d169
		snap652 := d170
		snap653 := d171
		snap654 := d172
		snap655 := d173
		snap656 := d174
		snap657 := d175
		snap658 := d176
		snap659 := d177
		snap660 := d178
		snap661 := d179
		snap662 := d180
		snap663 := d181
		snap664 := d182
		snap665 := d183
		snap666 := d184
		snap667 := d187
		snap668 := d344
		snap669 := d345
		snap670 := d346
		snap671 := d347
		snap672 := d349
		snap673 := d350
		snap674 := d351
		snap675 := d352
		snap676 := d353
		snap677 := d354
		snap678 := d355
		snap679 := d356
		snap680 := d358
		snap681 := d360
		snap682 := d361
		snap683 := d362
		snap684 := d456
		snap685 := d457
		snap686 := d460
		snap687 := d557
		snap688 := d558
		snap689 := d559
		snap690 := d560
		snap691 := d561
		snap692 := d563
		snap693 := d564
		snap694 := d565
		snap695 := d566
		snap696 := d567
		snap697 := d568
		snap698 := d569
		snap699 := d570
		snap700 := d571
		snap701 := d572
		snap702 := d573
		snap703 := d574
		snap704 := d575
		snap705 := d576
		snap706 := d577
		snap707 := d578
		snap708 := d579
		snap709 := d580
		snap710 := d581
		snap711 := d582
		snap712 := d583
		snap713 := d584
		snap714 := d585
		snap715 := d586
		snap716 := d587
		snap717 := d588
		snap718 := d589
		alloc719 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc719)
		d5 = snap592
		d6 = snap593
		d7 = snap594
		d8 = snap595
		d9 = snap596
		d10 = snap597
		d11 = snap598
		d12 = snap599
		d13 = snap600
		d14 = snap601
		d15 = snap602
		d16 = snap603
		d17 = snap604
		d18 = snap605
		d19 = snap606
		d21 = snap607
		d22 = snap608
		d23 = snap609
		d24 = snap610
		d25 = snap611
		d26 = snap612
		d27 = snap613
		d28 = snap614
		d29 = snap615
		d30 = snap616
		d31 = snap617
		d32 = snap618
		d33 = snap619
		d34 = snap620
		d35 = snap621
		d36 = snap622
		d37 = snap623
		d38 = snap624
		d39 = snap625
		d40 = snap626
		d41 = snap627
		d42 = snap628
		d43 = snap629
		d44 = snap630
		d45 = snap631
		d46 = snap632
		d47 = snap633
		d48 = snap634
		d49 = snap635
		d50 = snap636
		d51 = snap637
		d54 = snap638
		d55 = snap639
		d56 = snap640
		d159 = snap641
		d160 = snap642
		d161 = snap643
		d162 = snap644
		d163 = snap645
		d164 = snap646
		d165 = snap647
		d166 = snap648
		d167 = snap649
		d168 = snap650
		d169 = snap651
		d170 = snap652
		d171 = snap653
		d172 = snap654
		d173 = snap655
		d174 = snap656
		d175 = snap657
		d176 = snap658
		d177 = snap659
		d178 = snap660
		d179 = snap661
		d180 = snap662
		d181 = snap663
		d182 = snap664
		d183 = snap665
		d184 = snap666
		d187 = snap667
		d344 = snap668
		d345 = snap669
		d346 = snap670
		d347 = snap671
		d349 = snap672
		d350 = snap673
		d351 = snap674
		d352 = snap675
		d353 = snap676
		d354 = snap677
		d355 = snap678
		d356 = snap679
		d358 = snap680
		d360 = snap681
		d361 = snap682
		d362 = snap683
		d456 = snap684
		d457 = snap685
		d460 = snap686
		d557 = snap687
		d558 = snap688
		d559 = snap689
		d560 = snap690
		d561 = snap691
		d563 = snap692
		d564 = snap693
		d565 = snap694
		d566 = snap695
		d567 = snap696
		d568 = snap697
		d569 = snap698
		d570 = snap699
		d571 = snap700
		d572 = snap701
		d573 = snap702
		d574 = snap703
		d575 = snap704
		d576 = snap705
		d577 = snap706
		d578 = snap707
		d579 = snap708
		d580 = snap709
		d581 = snap710
		d582 = snap711
		d583 = snap712
		d584 = snap713
		d585 = snap714
		d586 = snap715
		d587 = snap716
		d588 = snap717
		d589 = snap718
		ctx.RestoreAllocState(alloc719)
		d5 = snap592
		d6 = snap593
		d7 = snap594
		d8 = snap595
		d9 = snap596
		d10 = snap597
		d11 = snap598
		d12 = snap599
		d13 = snap600
		d14 = snap601
		d15 = snap602
		d16 = snap603
		d17 = snap604
		d18 = snap605
		d19 = snap606
		d21 = snap607
		d22 = snap608
		d23 = snap609
		d24 = snap610
		d25 = snap611
		d26 = snap612
		d27 = snap613
		d28 = snap614
		d29 = snap615
		d30 = snap616
		d31 = snap617
		d32 = snap618
		d33 = snap619
		d34 = snap620
		d35 = snap621
		d36 = snap622
		d37 = snap623
		d38 = snap624
		d39 = snap625
		d40 = snap626
		d41 = snap627
		d42 = snap628
		d43 = snap629
		d44 = snap630
		d45 = snap631
		d46 = snap632
		d47 = snap633
		d48 = snap634
		d49 = snap635
		d50 = snap636
		d51 = snap637
		d54 = snap638
		d55 = snap639
		d56 = snap640
		d159 = snap641
		d160 = snap642
		d161 = snap643
		d162 = snap644
		d163 = snap645
		d164 = snap646
		d165 = snap647
		d166 = snap648
		d167 = snap649
		d168 = snap650
		d169 = snap651
		d170 = snap652
		d171 = snap653
		d172 = snap654
		d173 = snap655
		d174 = snap656
		d175 = snap657
		d176 = snap658
		d177 = snap659
		d178 = snap660
		d179 = snap661
		d180 = snap662
		d181 = snap663
		d182 = snap664
		d183 = snap665
		d184 = snap666
		d187 = snap667
		d344 = snap668
		d345 = snap669
		d346 = snap670
		d347 = snap671
		d349 = snap672
		d350 = snap673
		d351 = snap674
		d352 = snap675
		d353 = snap676
		d354 = snap677
		d355 = snap678
		d356 = snap679
		d358 = snap680
		d360 = snap681
		d361 = snap682
		d362 = snap683
		d456 = snap684
		d457 = snap685
		d460 = snap686
		d557 = snap687
		d558 = snap688
		d559 = snap689
		d560 = snap690
		d561 = snap691
		d563 = snap692
		d564 = snap693
		d565 = snap694
		d566 = snap695
		d567 = snap696
		d568 = snap697
		d569 = snap698
		d570 = snap699
		d571 = snap700
		d572 = snap701
		d573 = snap702
		d574 = snap703
		d575 = snap704
		d576 = snap705
		d577 = snap706
		d578 = snap707
		d579 = snap708
		d580 = snap709
		d581 = snap710
		d582 = snap711
		d583 = snap712
		d584 = snap713
		d585 = snap714
		d586 = snap715
		d587 = snap716
		d588 = snap717
		d589 = snap718
		ps720 := scm.PhiState{General: true}
		ps720.OverlayValues = make([]scm.JITValueDesc, 590)
		ps720.OverlayValues[5] = d5
		ps720.OverlayValues[6] = d6
		ps720.OverlayValues[7] = d7
		ps720.OverlayValues[8] = d8
		ps720.OverlayValues[9] = d9
		ps720.OverlayValues[10] = d10
		ps720.OverlayValues[11] = d11
		ps720.OverlayValues[12] = d12
		ps720.OverlayValues[13] = d13
		ps720.OverlayValues[14] = d14
		ps720.OverlayValues[15] = d15
		ps720.OverlayValues[16] = d16
		ps720.OverlayValues[17] = d17
		ps720.OverlayValues[18] = d18
		ps720.OverlayValues[19] = d19
		ps720.OverlayValues[21] = d21
		ps720.OverlayValues[22] = d22
		ps720.OverlayValues[23] = d23
		ps720.OverlayValues[24] = d24
		ps720.OverlayValues[25] = d25
		ps720.OverlayValues[26] = d26
		ps720.OverlayValues[27] = d27
		ps720.OverlayValues[28] = d28
		ps720.OverlayValues[29] = d29
		ps720.OverlayValues[30] = d30
		ps720.OverlayValues[31] = d31
		ps720.OverlayValues[32] = d32
		ps720.OverlayValues[33] = d33
		ps720.OverlayValues[34] = d34
		ps720.OverlayValues[35] = d35
		ps720.OverlayValues[36] = d36
		ps720.OverlayValues[37] = d37
		ps720.OverlayValues[38] = d38
		ps720.OverlayValues[39] = d39
		ps720.OverlayValues[40] = d40
		ps720.OverlayValues[41] = d41
		ps720.OverlayValues[42] = d42
		ps720.OverlayValues[43] = d43
		ps720.OverlayValues[44] = d44
		ps720.OverlayValues[45] = d45
		ps720.OverlayValues[46] = d46
		ps720.OverlayValues[47] = d47
		ps720.OverlayValues[48] = d48
		ps720.OverlayValues[49] = d49
		ps720.OverlayValues[50] = d50
		ps720.OverlayValues[51] = d51
		ps720.OverlayValues[54] = d54
		ps720.OverlayValues[55] = d55
		ps720.OverlayValues[56] = d56
		ps720.OverlayValues[159] = d159
		ps720.OverlayValues[160] = d160
		ps720.OverlayValues[161] = d161
		ps720.OverlayValues[162] = d162
		ps720.OverlayValues[163] = d163
		ps720.OverlayValues[164] = d164
		ps720.OverlayValues[165] = d165
		ps720.OverlayValues[166] = d166
		ps720.OverlayValues[167] = d167
		ps720.OverlayValues[168] = d168
		ps720.OverlayValues[169] = d169
		ps720.OverlayValues[170] = d170
		ps720.OverlayValues[171] = d171
		ps720.OverlayValues[172] = d172
		ps720.OverlayValues[173] = d173
		ps720.OverlayValues[174] = d174
		ps720.OverlayValues[175] = d175
		ps720.OverlayValues[176] = d176
		ps720.OverlayValues[177] = d177
		ps720.OverlayValues[178] = d178
		ps720.OverlayValues[179] = d179
		ps720.OverlayValues[180] = d180
		ps720.OverlayValues[181] = d181
		ps720.OverlayValues[182] = d182
		ps720.OverlayValues[183] = d183
		ps720.OverlayValues[184] = d184
		ps720.OverlayValues[187] = d187
		ps720.OverlayValues[344] = d344
		ps720.OverlayValues[345] = d345
		ps720.OverlayValues[346] = d346
		ps720.OverlayValues[347] = d347
		ps720.OverlayValues[349] = d349
		ps720.OverlayValues[350] = d350
		ps720.OverlayValues[351] = d351
		ps720.OverlayValues[352] = d352
		ps720.OverlayValues[353] = d353
		ps720.OverlayValues[354] = d354
		ps720.OverlayValues[355] = d355
		ps720.OverlayValues[356] = d356
		ps720.OverlayValues[358] = d358
		ps720.OverlayValues[360] = d360
		ps720.OverlayValues[361] = d361
		ps720.OverlayValues[362] = d362
		ps720.OverlayValues[456] = d456
		ps720.OverlayValues[457] = d457
		ps720.OverlayValues[460] = d460
		ps720.OverlayValues[557] = d557
		ps720.OverlayValues[558] = d558
		ps720.OverlayValues[559] = d559
		ps720.OverlayValues[560] = d560
		ps720.OverlayValues[561] = d561
		ps720.OverlayValues[563] = d563
		ps720.OverlayValues[564] = d564
		ps720.OverlayValues[565] = d565
		ps720.OverlayValues[566] = d566
		ps720.OverlayValues[567] = d567
		ps720.OverlayValues[568] = d568
		ps720.OverlayValues[569] = d569
		ps720.OverlayValues[570] = d570
		ps720.OverlayValues[571] = d571
		ps720.OverlayValues[572] = d572
		ps720.OverlayValues[573] = d573
		ps720.OverlayValues[574] = d574
		ps720.OverlayValues[575] = d575
		ps720.OverlayValues[576] = d576
		ps720.OverlayValues[577] = d577
		ps720.OverlayValues[578] = d578
		ps720.OverlayValues[579] = d579
		ps720.OverlayValues[580] = d580
		ps720.OverlayValues[581] = d581
		ps720.OverlayValues[582] = d582
		ps720.OverlayValues[583] = d583
		ps720.OverlayValues[584] = d584
		ps720.OverlayValues[585] = d585
		ps720.OverlayValues[586] = d586
		ps720.OverlayValues[587] = d587
		ps720.OverlayValues[588] = d588
		ps720.OverlayValues[589] = d589
		ps721 := scm.PhiState{General: true}
		ps721.OverlayValues = make([]scm.JITValueDesc, 590)
		ps721.OverlayValues[5] = d5
		ps721.OverlayValues[6] = d6
		ps721.OverlayValues[7] = d7
		ps721.OverlayValues[8] = d8
		ps721.OverlayValues[9] = d9
		ps721.OverlayValues[10] = d10
		ps721.OverlayValues[11] = d11
		ps721.OverlayValues[12] = d12
		ps721.OverlayValues[13] = d13
		ps721.OverlayValues[14] = d14
		ps721.OverlayValues[15] = d15
		ps721.OverlayValues[16] = d16
		ps721.OverlayValues[17] = d17
		ps721.OverlayValues[18] = d18
		ps721.OverlayValues[19] = d19
		ps721.OverlayValues[21] = d21
		ps721.OverlayValues[22] = d22
		ps721.OverlayValues[23] = d23
		ps721.OverlayValues[24] = d24
		ps721.OverlayValues[25] = d25
		ps721.OverlayValues[26] = d26
		ps721.OverlayValues[27] = d27
		ps721.OverlayValues[28] = d28
		ps721.OverlayValues[29] = d29
		ps721.OverlayValues[30] = d30
		ps721.OverlayValues[31] = d31
		ps721.OverlayValues[32] = d32
		ps721.OverlayValues[33] = d33
		ps721.OverlayValues[34] = d34
		ps721.OverlayValues[35] = d35
		ps721.OverlayValues[36] = d36
		ps721.OverlayValues[37] = d37
		ps721.OverlayValues[38] = d38
		ps721.OverlayValues[39] = d39
		ps721.OverlayValues[40] = d40
		ps721.OverlayValues[41] = d41
		ps721.OverlayValues[42] = d42
		ps721.OverlayValues[43] = d43
		ps721.OverlayValues[44] = d44
		ps721.OverlayValues[45] = d45
		ps721.OverlayValues[46] = d46
		ps721.OverlayValues[47] = d47
		ps721.OverlayValues[48] = d48
		ps721.OverlayValues[49] = d49
		ps721.OverlayValues[50] = d50
		ps721.OverlayValues[51] = d51
		ps721.OverlayValues[54] = d54
		ps721.OverlayValues[55] = d55
		ps721.OverlayValues[56] = d56
		ps721.OverlayValues[159] = d159
		ps721.OverlayValues[160] = d160
		ps721.OverlayValues[161] = d161
		ps721.OverlayValues[162] = d162
		ps721.OverlayValues[163] = d163
		ps721.OverlayValues[164] = d164
		ps721.OverlayValues[165] = d165
		ps721.OverlayValues[166] = d166
		ps721.OverlayValues[167] = d167
		ps721.OverlayValues[168] = d168
		ps721.OverlayValues[169] = d169
		ps721.OverlayValues[170] = d170
		ps721.OverlayValues[171] = d171
		ps721.OverlayValues[172] = d172
		ps721.OverlayValues[173] = d173
		ps721.OverlayValues[174] = d174
		ps721.OverlayValues[175] = d175
		ps721.OverlayValues[176] = d176
		ps721.OverlayValues[177] = d177
		ps721.OverlayValues[178] = d178
		ps721.OverlayValues[179] = d179
		ps721.OverlayValues[180] = d180
		ps721.OverlayValues[181] = d181
		ps721.OverlayValues[182] = d182
		ps721.OverlayValues[183] = d183
		ps721.OverlayValues[184] = d184
		ps721.OverlayValues[187] = d187
		ps721.OverlayValues[344] = d344
		ps721.OverlayValues[345] = d345
		ps721.OverlayValues[346] = d346
		ps721.OverlayValues[347] = d347
		ps721.OverlayValues[349] = d349
		ps721.OverlayValues[350] = d350
		ps721.OverlayValues[351] = d351
		ps721.OverlayValues[352] = d352
		ps721.OverlayValues[353] = d353
		ps721.OverlayValues[354] = d354
		ps721.OverlayValues[355] = d355
		ps721.OverlayValues[356] = d356
		ps721.OverlayValues[358] = d358
		ps721.OverlayValues[360] = d360
		ps721.OverlayValues[361] = d361
		ps721.OverlayValues[362] = d362
		ps721.OverlayValues[456] = d456
		ps721.OverlayValues[457] = d457
		ps721.OverlayValues[460] = d460
		ps721.OverlayValues[557] = d557
		ps721.OverlayValues[558] = d558
		ps721.OverlayValues[559] = d559
		ps721.OverlayValues[560] = d560
		ps721.OverlayValues[561] = d561
		ps721.OverlayValues[563] = d563
		ps721.OverlayValues[564] = d564
		ps721.OverlayValues[565] = d565
		ps721.OverlayValues[566] = d566
		ps721.OverlayValues[567] = d567
		ps721.OverlayValues[568] = d568
		ps721.OverlayValues[569] = d569
		ps721.OverlayValues[570] = d570
		ps721.OverlayValues[571] = d571
		ps721.OverlayValues[572] = d572
		ps721.OverlayValues[573] = d573
		ps721.OverlayValues[574] = d574
		ps721.OverlayValues[575] = d575
		ps721.OverlayValues[576] = d576
		ps721.OverlayValues[577] = d577
		ps721.OverlayValues[578] = d578
		ps721.OverlayValues[579] = d579
		ps721.OverlayValues[580] = d580
		ps721.OverlayValues[581] = d581
		ps721.OverlayValues[582] = d582
		ps721.OverlayValues[583] = d583
		ps721.OverlayValues[584] = d584
		ps721.OverlayValues[585] = d585
		ps721.OverlayValues[586] = d586
		ps721.OverlayValues[587] = d587
		ps721.OverlayValues[588] = d588
		ps721.OverlayValues[589] = d589
		snap722 := d5
		snap723 := d6
		snap724 := d7
		snap725 := d8
		snap726 := d9
		snap727 := d10
		snap728 := d11
		snap729 := d12
		snap730 := d13
		snap731 := d14
		snap732 := d15
		snap733 := d16
		snap734 := d17
		snap735 := d18
		snap736 := d19
		snap737 := d21
		snap738 := d22
		snap739 := d23
		snap740 := d24
		snap741 := d25
		snap742 := d26
		snap743 := d27
		snap744 := d28
		snap745 := d29
		snap746 := d30
		snap747 := d31
		snap748 := d32
		snap749 := d33
		snap750 := d34
		snap751 := d35
		snap752 := d36
		snap753 := d37
		snap754 := d38
		snap755 := d39
		snap756 := d40
		snap757 := d41
		snap758 := d42
		snap759 := d43
		snap760 := d44
		snap761 := d45
		snap762 := d46
		snap763 := d47
		snap764 := d48
		snap765 := d49
		snap766 := d50
		snap767 := d51
		snap768 := d54
		snap769 := d55
		snap770 := d56
		snap771 := d159
		snap772 := d160
		snap773 := d161
		snap774 := d162
		snap775 := d163
		snap776 := d164
		snap777 := d165
		snap778 := d166
		snap779 := d167
		snap780 := d168
		snap781 := d169
		snap782 := d170
		snap783 := d171
		snap784 := d172
		snap785 := d173
		snap786 := d174
		snap787 := d175
		snap788 := d176
		snap789 := d177
		snap790 := d178
		snap791 := d179
		snap792 := d180
		snap793 := d181
		snap794 := d182
		snap795 := d183
		snap796 := d184
		snap797 := d187
		snap798 := d344
		snap799 := d345
		snap800 := d346
		snap801 := d347
		snap802 := d349
		snap803 := d350
		snap804 := d351
		snap805 := d352
		snap806 := d353
		snap807 := d354
		snap808 := d355
		snap809 := d356
		snap810 := d358
		snap811 := d360
		snap812 := d361
		snap813 := d362
		snap814 := d456
		snap815 := d457
		snap816 := d460
		snap817 := d557
		snap818 := d558
		snap819 := d559
		snap820 := d560
		snap821 := d561
		snap822 := d563
		snap823 := d564
		snap824 := d565
		snap825 := d566
		snap826 := d567
		snap827 := d568
		snap828 := d569
		snap829 := d570
		snap830 := d571
		snap831 := d572
		snap832 := d573
		snap833 := d574
		snap834 := d575
		snap835 := d576
		snap836 := d577
		snap837 := d578
		snap838 := d579
		snap839 := d580
		snap840 := d581
		snap841 := d582
		snap842 := d583
		snap843 := d584
		snap844 := d585
		snap845 := d586
		snap846 := d587
		snap847 := d588
		snap848 := d589
		alloc849 := ctx.SnapshotAllocState()
		if !bbs[9].Rendered {
			bbs[9].RenderPS(ps721)
		}
		ctx.RestoreAllocState(alloc849)
		d5 = snap722
		d6 = snap723
		d7 = snap724
		d8 = snap725
		d9 = snap726
		d10 = snap727
		d11 = snap728
		d12 = snap729
		d13 = snap730
		d14 = snap731
		d15 = snap732
		d16 = snap733
		d17 = snap734
		d18 = snap735
		d19 = snap736
		d21 = snap737
		d22 = snap738
		d23 = snap739
		d24 = snap740
		d25 = snap741
		d26 = snap742
		d27 = snap743
		d28 = snap744
		d29 = snap745
		d30 = snap746
		d31 = snap747
		d32 = snap748
		d33 = snap749
		d34 = snap750
		d35 = snap751
		d36 = snap752
		d37 = snap753
		d38 = snap754
		d39 = snap755
		d40 = snap756
		d41 = snap757
		d42 = snap758
		d43 = snap759
		d44 = snap760
		d45 = snap761
		d46 = snap762
		d47 = snap763
		d48 = snap764
		d49 = snap765
		d50 = snap766
		d51 = snap767
		d54 = snap768
		d55 = snap769
		d56 = snap770
		d159 = snap771
		d160 = snap772
		d161 = snap773
		d162 = snap774
		d163 = snap775
		d164 = snap776
		d165 = snap777
		d166 = snap778
		d167 = snap779
		d168 = snap780
		d169 = snap781
		d170 = snap782
		d171 = snap783
		d172 = snap784
		d173 = snap785
		d174 = snap786
		d175 = snap787
		d176 = snap788
		d177 = snap789
		d178 = snap790
		d179 = snap791
		d180 = snap792
		d181 = snap793
		d182 = snap794
		d183 = snap795
		d184 = snap796
		d187 = snap797
		d344 = snap798
		d345 = snap799
		d346 = snap800
		d347 = snap801
		d349 = snap802
		d350 = snap803
		d351 = snap804
		d352 = snap805
		d353 = snap806
		d354 = snap807
		d355 = snap808
		d356 = snap809
		d358 = snap810
		d360 = snap811
		d361 = snap812
		d362 = snap813
		d456 = snap814
		d457 = snap815
		d460 = snap816
		d557 = snap817
		d558 = snap818
		d559 = snap819
		d560 = snap820
		d561 = snap821
		d563 = snap822
		d564 = snap823
		d565 = snap824
		d566 = snap825
		d567 = snap826
		d568 = snap827
		d569 = snap828
		d570 = snap829
		d571 = snap830
		d572 = snap831
		d573 = snap832
		d574 = snap833
		d575 = snap834
		d576 = snap835
		d577 = snap836
		d578 = snap837
		d579 = snap838
		d580 = snap839
		d581 = snap840
		d582 = snap841
		d583 = snap842
		d584 = snap843
		d585 = snap844
		d586 = snap845
		d587 = snap846
		d588 = snap847
		d589 = snap848
		if !bbs[7].Rendered {
			return bbs[7].RenderPS(ps720)
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
		if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
			d187 = ps.OverlayValues[187]
		}
		if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != scm.LocNone {
			d344 = ps.OverlayValues[344]
		}
		if len(ps.OverlayValues) > 345 && ps.OverlayValues[345].Loc != scm.LocNone {
			d345 = ps.OverlayValues[345]
		}
		if len(ps.OverlayValues) > 346 && ps.OverlayValues[346].Loc != scm.LocNone {
			d346 = ps.OverlayValues[346]
		}
		if len(ps.OverlayValues) > 347 && ps.OverlayValues[347].Loc != scm.LocNone {
			d347 = ps.OverlayValues[347]
		}
		if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != scm.LocNone {
			d349 = ps.OverlayValues[349]
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
		if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != scm.LocNone {
			d355 = ps.OverlayValues[355]
		}
		if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != scm.LocNone {
			d356 = ps.OverlayValues[356]
		}
		if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != scm.LocNone {
			d358 = ps.OverlayValues[358]
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
		if len(ps.OverlayValues) > 456 && ps.OverlayValues[456].Loc != scm.LocNone {
			d456 = ps.OverlayValues[456]
		}
		if len(ps.OverlayValues) > 457 && ps.OverlayValues[457].Loc != scm.LocNone {
			d457 = ps.OverlayValues[457]
		}
		if len(ps.OverlayValues) > 460 && ps.OverlayValues[460].Loc != scm.LocNone {
			d460 = ps.OverlayValues[460]
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
		if len(ps.OverlayValues) > 561 && ps.OverlayValues[561].Loc != scm.LocNone {
			d561 = ps.OverlayValues[561]
		}
		if len(ps.OverlayValues) > 563 && ps.OverlayValues[563].Loc != scm.LocNone {
			d563 = ps.OverlayValues[563]
		}
		if len(ps.OverlayValues) > 564 && ps.OverlayValues[564].Loc != scm.LocNone {
			d564 = ps.OverlayValues[564]
		}
		if len(ps.OverlayValues) > 565 && ps.OverlayValues[565].Loc != scm.LocNone {
			d565 = ps.OverlayValues[565]
		}
		if len(ps.OverlayValues) > 566 && ps.OverlayValues[566].Loc != scm.LocNone {
			d566 = ps.OverlayValues[566]
		}
		if len(ps.OverlayValues) > 567 && ps.OverlayValues[567].Loc != scm.LocNone {
			d567 = ps.OverlayValues[567]
		}
		if len(ps.OverlayValues) > 568 && ps.OverlayValues[568].Loc != scm.LocNone {
			d568 = ps.OverlayValues[568]
		}
		if len(ps.OverlayValues) > 569 && ps.OverlayValues[569].Loc != scm.LocNone {
			d569 = ps.OverlayValues[569]
		}
		if len(ps.OverlayValues) > 570 && ps.OverlayValues[570].Loc != scm.LocNone {
			d570 = ps.OverlayValues[570]
		}
		if len(ps.OverlayValues) > 571 && ps.OverlayValues[571].Loc != scm.LocNone {
			d571 = ps.OverlayValues[571]
		}
		if len(ps.OverlayValues) > 572 && ps.OverlayValues[572].Loc != scm.LocNone {
			d572 = ps.OverlayValues[572]
		}
		if len(ps.OverlayValues) > 573 && ps.OverlayValues[573].Loc != scm.LocNone {
			d573 = ps.OverlayValues[573]
		}
		if len(ps.OverlayValues) > 574 && ps.OverlayValues[574].Loc != scm.LocNone {
			d574 = ps.OverlayValues[574]
		}
		if len(ps.OverlayValues) > 575 && ps.OverlayValues[575].Loc != scm.LocNone {
			d575 = ps.OverlayValues[575]
		}
		if len(ps.OverlayValues) > 576 && ps.OverlayValues[576].Loc != scm.LocNone {
			d576 = ps.OverlayValues[576]
		}
		if len(ps.OverlayValues) > 577 && ps.OverlayValues[577].Loc != scm.LocNone {
			d577 = ps.OverlayValues[577]
		}
		if len(ps.OverlayValues) > 578 && ps.OverlayValues[578].Loc != scm.LocNone {
			d578 = ps.OverlayValues[578]
		}
		if len(ps.OverlayValues) > 579 && ps.OverlayValues[579].Loc != scm.LocNone {
			d579 = ps.OverlayValues[579]
		}
		if len(ps.OverlayValues) > 580 && ps.OverlayValues[580].Loc != scm.LocNone {
			d580 = ps.OverlayValues[580]
		}
		if len(ps.OverlayValues) > 581 && ps.OverlayValues[581].Loc != scm.LocNone {
			d581 = ps.OverlayValues[581]
		}
		if len(ps.OverlayValues) > 582 && ps.OverlayValues[582].Loc != scm.LocNone {
			d582 = ps.OverlayValues[582]
		}
		if len(ps.OverlayValues) > 583 && ps.OverlayValues[583].Loc != scm.LocNone {
			d583 = ps.OverlayValues[583]
		}
		if len(ps.OverlayValues) > 584 && ps.OverlayValues[584].Loc != scm.LocNone {
			d584 = ps.OverlayValues[584]
		}
		if len(ps.OverlayValues) > 585 && ps.OverlayValues[585].Loc != scm.LocNone {
			d585 = ps.OverlayValues[585]
		}
		if len(ps.OverlayValues) > 586 && ps.OverlayValues[586].Loc != scm.LocNone {
			d586 = ps.OverlayValues[586]
		}
		if len(ps.OverlayValues) > 587 && ps.OverlayValues[587].Loc != scm.LocNone {
			d587 = ps.OverlayValues[587]
		}
		if len(ps.OverlayValues) > 588 && ps.OverlayValues[588].Loc != scm.LocNone {
			d588 = ps.OverlayValues[588]
		}
		if len(ps.OverlayValues) > 589 && ps.OverlayValues[589].Loc != scm.LocNone {
			d589 = ps.OverlayValues[589]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d9)
		ctx.EnsureDesc(&d9)
		var d850 scm.JITValueDesc
		if d9.Loc == scm.LocImm {
			d850 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d9.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d9.Reg)
			ctx.EmitMovRegReg(scratch, d9.Reg)
			ctx.EmitSubRegImm32Low(scratch, int32(1))
			d850 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d850)
		}
		if d850.Loc == scm.LocReg && d9.Loc == scm.LocReg && d850.Reg == d9.Reg {
			ctx.TransferReg(d9.Reg)
			d9.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d850)
		ctx.EmitStoreToStack(d850, int32(bbs[8].PhiBase)+int32(16))
		ctx.StabilizeDescForControlFlow(&d850)
		if ps.General {
			ctx.SyncDesc(&d10)
			if d10.Loc == scm.LocReg {
				ctx.ProtectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.ProtectReg(d10.Reg)
				ctx.ProtectReg(d10.Reg2)
			}
			d851 = d10
			if d851.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d851)
			d852 = d851
			if d852.Loc == scm.LocImm {
				d852 = scm.JITValueDesc{Loc: scm.LocImm, Type: d852.Type, Imm: scm.NewInt(int64(uint64(d852.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d852.Reg, 32)
				ctx.EmitShrRegImm8(d852.Reg, 32)
			}
			ctx.EmitStoreToStack(d852, int32(bbs[8].PhiBase)+int32(0))
			if d10.Loc == scm.LocReg {
				ctx.UnprotectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d10.Reg)
				ctx.UnprotectReg(d10.Reg2)
			}
		}
		ps853 := scm.PhiState{General: ps.General}
		ps853.OverlayValues = make([]scm.JITValueDesc, 853)
		ps853.OverlayValues[5] = d5
		ps853.OverlayValues[6] = d6
		ps853.OverlayValues[7] = d7
		ps853.OverlayValues[8] = d8
		ps853.OverlayValues[9] = d9
		ps853.OverlayValues[10] = d10
		ps853.OverlayValues[11] = d11
		ps853.OverlayValues[12] = d12
		ps853.OverlayValues[13] = d13
		ps853.OverlayValues[14] = d14
		ps853.OverlayValues[15] = d15
		ps853.OverlayValues[16] = d16
		ps853.OverlayValues[17] = d17
		ps853.OverlayValues[18] = d18
		ps853.OverlayValues[19] = d19
		ps853.OverlayValues[21] = d21
		ps853.OverlayValues[22] = d22
		ps853.OverlayValues[23] = d23
		ps853.OverlayValues[24] = d24
		ps853.OverlayValues[25] = d25
		ps853.OverlayValues[26] = d26
		ps853.OverlayValues[27] = d27
		ps853.OverlayValues[28] = d28
		ps853.OverlayValues[29] = d29
		ps853.OverlayValues[30] = d30
		ps853.OverlayValues[31] = d31
		ps853.OverlayValues[32] = d32
		ps853.OverlayValues[33] = d33
		ps853.OverlayValues[34] = d34
		ps853.OverlayValues[35] = d35
		ps853.OverlayValues[36] = d36
		ps853.OverlayValues[37] = d37
		ps853.OverlayValues[38] = d38
		ps853.OverlayValues[39] = d39
		ps853.OverlayValues[40] = d40
		ps853.OverlayValues[41] = d41
		ps853.OverlayValues[42] = d42
		ps853.OverlayValues[43] = d43
		ps853.OverlayValues[44] = d44
		ps853.OverlayValues[45] = d45
		ps853.OverlayValues[46] = d46
		ps853.OverlayValues[47] = d47
		ps853.OverlayValues[48] = d48
		ps853.OverlayValues[49] = d49
		ps853.OverlayValues[50] = d50
		ps853.OverlayValues[51] = d51
		ps853.OverlayValues[54] = d54
		ps853.OverlayValues[55] = d55
		ps853.OverlayValues[56] = d56
		ps853.OverlayValues[159] = d159
		ps853.OverlayValues[160] = d160
		ps853.OverlayValues[161] = d161
		ps853.OverlayValues[162] = d162
		ps853.OverlayValues[163] = d163
		ps853.OverlayValues[164] = d164
		ps853.OverlayValues[165] = d165
		ps853.OverlayValues[166] = d166
		ps853.OverlayValues[167] = d167
		ps853.OverlayValues[168] = d168
		ps853.OverlayValues[169] = d169
		ps853.OverlayValues[170] = d170
		ps853.OverlayValues[171] = d171
		ps853.OverlayValues[172] = d172
		ps853.OverlayValues[173] = d173
		ps853.OverlayValues[174] = d174
		ps853.OverlayValues[175] = d175
		ps853.OverlayValues[176] = d176
		ps853.OverlayValues[177] = d177
		ps853.OverlayValues[178] = d178
		ps853.OverlayValues[179] = d179
		ps853.OverlayValues[180] = d180
		ps853.OverlayValues[181] = d181
		ps853.OverlayValues[182] = d182
		ps853.OverlayValues[183] = d183
		ps853.OverlayValues[184] = d184
		ps853.OverlayValues[187] = d187
		ps853.OverlayValues[344] = d344
		ps853.OverlayValues[345] = d345
		ps853.OverlayValues[346] = d346
		ps853.OverlayValues[347] = d347
		ps853.OverlayValues[349] = d349
		ps853.OverlayValues[350] = d350
		ps853.OverlayValues[351] = d351
		ps853.OverlayValues[352] = d352
		ps853.OverlayValues[353] = d353
		ps853.OverlayValues[354] = d354
		ps853.OverlayValues[355] = d355
		ps853.OverlayValues[356] = d356
		ps853.OverlayValues[358] = d358
		ps853.OverlayValues[360] = d360
		ps853.OverlayValues[361] = d361
		ps853.OverlayValues[362] = d362
		ps853.OverlayValues[456] = d456
		ps853.OverlayValues[457] = d457
		ps853.OverlayValues[460] = d460
		ps853.OverlayValues[557] = d557
		ps853.OverlayValues[558] = d558
		ps853.OverlayValues[559] = d559
		ps853.OverlayValues[560] = d560
		ps853.OverlayValues[561] = d561
		ps853.OverlayValues[563] = d563
		ps853.OverlayValues[564] = d564
		ps853.OverlayValues[565] = d565
		ps853.OverlayValues[566] = d566
		ps853.OverlayValues[567] = d567
		ps853.OverlayValues[568] = d568
		ps853.OverlayValues[569] = d569
		ps853.OverlayValues[570] = d570
		ps853.OverlayValues[571] = d571
		ps853.OverlayValues[572] = d572
		ps853.OverlayValues[573] = d573
		ps853.OverlayValues[574] = d574
		ps853.OverlayValues[575] = d575
		ps853.OverlayValues[576] = d576
		ps853.OverlayValues[577] = d577
		ps853.OverlayValues[578] = d578
		ps853.OverlayValues[579] = d579
		ps853.OverlayValues[580] = d580
		ps853.OverlayValues[581] = d581
		ps853.OverlayValues[582] = d582
		ps853.OverlayValues[583] = d583
		ps853.OverlayValues[584] = d584
		ps853.OverlayValues[585] = d585
		ps853.OverlayValues[586] = d586
		ps853.OverlayValues[587] = d587
		ps853.OverlayValues[588] = d588
		ps853.OverlayValues[589] = d589
		ps853.OverlayValues[850] = d850
		ps853.OverlayValues[851] = d851
		ps853.OverlayValues[852] = d852
		ps853.PhiValues = make([]scm.JITValueDesc, 2)
		d854 = d10
		ps853.PhiValues[0] = d854
		if ps853.General && bbs[8].Rendered {
			ctx.EmitJmp(lbl9)
			return result
		}
		return bbs[8].RenderPS(ps853)
		return result
	}
	bbs[8].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d855 := ps.PhiValues[0]
				ctx.EnsureDesc(&d855)
				ctx.EmitStoreToStack(d855, int32(bbs[8].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d856 := ps.PhiValues[1]
				ctx.EnsureDesc(&d856)
				ctx.EmitStoreToStack(d856, int32(bbs[8].PhiBase)+int32(16))
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
		if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
			d187 = ps.OverlayValues[187]
		}
		if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != scm.LocNone {
			d344 = ps.OverlayValues[344]
		}
		if len(ps.OverlayValues) > 345 && ps.OverlayValues[345].Loc != scm.LocNone {
			d345 = ps.OverlayValues[345]
		}
		if len(ps.OverlayValues) > 346 && ps.OverlayValues[346].Loc != scm.LocNone {
			d346 = ps.OverlayValues[346]
		}
		if len(ps.OverlayValues) > 347 && ps.OverlayValues[347].Loc != scm.LocNone {
			d347 = ps.OverlayValues[347]
		}
		if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != scm.LocNone {
			d349 = ps.OverlayValues[349]
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
		if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != scm.LocNone {
			d355 = ps.OverlayValues[355]
		}
		if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != scm.LocNone {
			d356 = ps.OverlayValues[356]
		}
		if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != scm.LocNone {
			d358 = ps.OverlayValues[358]
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
		if len(ps.OverlayValues) > 456 && ps.OverlayValues[456].Loc != scm.LocNone {
			d456 = ps.OverlayValues[456]
		}
		if len(ps.OverlayValues) > 457 && ps.OverlayValues[457].Loc != scm.LocNone {
			d457 = ps.OverlayValues[457]
		}
		if len(ps.OverlayValues) > 460 && ps.OverlayValues[460].Loc != scm.LocNone {
			d460 = ps.OverlayValues[460]
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
		if len(ps.OverlayValues) > 561 && ps.OverlayValues[561].Loc != scm.LocNone {
			d561 = ps.OverlayValues[561]
		}
		if len(ps.OverlayValues) > 563 && ps.OverlayValues[563].Loc != scm.LocNone {
			d563 = ps.OverlayValues[563]
		}
		if len(ps.OverlayValues) > 564 && ps.OverlayValues[564].Loc != scm.LocNone {
			d564 = ps.OverlayValues[564]
		}
		if len(ps.OverlayValues) > 565 && ps.OverlayValues[565].Loc != scm.LocNone {
			d565 = ps.OverlayValues[565]
		}
		if len(ps.OverlayValues) > 566 && ps.OverlayValues[566].Loc != scm.LocNone {
			d566 = ps.OverlayValues[566]
		}
		if len(ps.OverlayValues) > 567 && ps.OverlayValues[567].Loc != scm.LocNone {
			d567 = ps.OverlayValues[567]
		}
		if len(ps.OverlayValues) > 568 && ps.OverlayValues[568].Loc != scm.LocNone {
			d568 = ps.OverlayValues[568]
		}
		if len(ps.OverlayValues) > 569 && ps.OverlayValues[569].Loc != scm.LocNone {
			d569 = ps.OverlayValues[569]
		}
		if len(ps.OverlayValues) > 570 && ps.OverlayValues[570].Loc != scm.LocNone {
			d570 = ps.OverlayValues[570]
		}
		if len(ps.OverlayValues) > 571 && ps.OverlayValues[571].Loc != scm.LocNone {
			d571 = ps.OverlayValues[571]
		}
		if len(ps.OverlayValues) > 572 && ps.OverlayValues[572].Loc != scm.LocNone {
			d572 = ps.OverlayValues[572]
		}
		if len(ps.OverlayValues) > 573 && ps.OverlayValues[573].Loc != scm.LocNone {
			d573 = ps.OverlayValues[573]
		}
		if len(ps.OverlayValues) > 574 && ps.OverlayValues[574].Loc != scm.LocNone {
			d574 = ps.OverlayValues[574]
		}
		if len(ps.OverlayValues) > 575 && ps.OverlayValues[575].Loc != scm.LocNone {
			d575 = ps.OverlayValues[575]
		}
		if len(ps.OverlayValues) > 576 && ps.OverlayValues[576].Loc != scm.LocNone {
			d576 = ps.OverlayValues[576]
		}
		if len(ps.OverlayValues) > 577 && ps.OverlayValues[577].Loc != scm.LocNone {
			d577 = ps.OverlayValues[577]
		}
		if len(ps.OverlayValues) > 578 && ps.OverlayValues[578].Loc != scm.LocNone {
			d578 = ps.OverlayValues[578]
		}
		if len(ps.OverlayValues) > 579 && ps.OverlayValues[579].Loc != scm.LocNone {
			d579 = ps.OverlayValues[579]
		}
		if len(ps.OverlayValues) > 580 && ps.OverlayValues[580].Loc != scm.LocNone {
			d580 = ps.OverlayValues[580]
		}
		if len(ps.OverlayValues) > 581 && ps.OverlayValues[581].Loc != scm.LocNone {
			d581 = ps.OverlayValues[581]
		}
		if len(ps.OverlayValues) > 582 && ps.OverlayValues[582].Loc != scm.LocNone {
			d582 = ps.OverlayValues[582]
		}
		if len(ps.OverlayValues) > 583 && ps.OverlayValues[583].Loc != scm.LocNone {
			d583 = ps.OverlayValues[583]
		}
		if len(ps.OverlayValues) > 584 && ps.OverlayValues[584].Loc != scm.LocNone {
			d584 = ps.OverlayValues[584]
		}
		if len(ps.OverlayValues) > 585 && ps.OverlayValues[585].Loc != scm.LocNone {
			d585 = ps.OverlayValues[585]
		}
		if len(ps.OverlayValues) > 586 && ps.OverlayValues[586].Loc != scm.LocNone {
			d586 = ps.OverlayValues[586]
		}
		if len(ps.OverlayValues) > 587 && ps.OverlayValues[587].Loc != scm.LocNone {
			d587 = ps.OverlayValues[587]
		}
		if len(ps.OverlayValues) > 588 && ps.OverlayValues[588].Loc != scm.LocNone {
			d588 = ps.OverlayValues[588]
		}
		if len(ps.OverlayValues) > 589 && ps.OverlayValues[589].Loc != scm.LocNone {
			d589 = ps.OverlayValues[589]
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
		if len(ps.OverlayValues) > 854 && ps.OverlayValues[854].Loc != scm.LocNone {
			d854 = ps.OverlayValues[854]
		}
		if len(ps.OverlayValues) > 855 && ps.OverlayValues[855].Loc != scm.LocNone {
			d855 = ps.OverlayValues[855]
		}
		if len(ps.OverlayValues) > 856 && ps.OverlayValues[856].Loc != scm.LocNone {
			d856 = ps.OverlayValues[856]
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
		var d857 scm.JITValueDesc
		if d12.Loc == scm.LocImm && d13.Loc == scm.LocImm {
			d857 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d12.Imm.Int()) == uint64(d13.Imm.Int()))}
		} else if d13.Loc == scm.LocImm {
			r99 := ctx.AllocRegExcept(d12.Reg)
			if d13.Imm.Int() >= -2147483648 && d13.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d12.Reg, int32(d13.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
				ctx.EmitCmpInt64(d12.Reg, scm.RegR11)
			}
			d857 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r99, Condition: scm.CondEqual}
			ctx.BindReg(r99, &d857)
		} else if d12.Loc == scm.LocImm {
			r100 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d12.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d13.Reg)
			d857 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r100, Condition: scm.CondEqual}
			ctx.BindReg(r100, &d857)
		} else {
			r101 := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitCmpInt64(d12.Reg, d13.Reg)
			d857 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r101, Condition: scm.CondEqual}
			ctx.BindReg(r101, &d857)
		}
		d858 = d857
		ctx.EnsureDesc(&d858)
		if d858.Loc != scm.LocImm && d858.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d858.Loc == scm.LocImm {
			if d858.Imm.Bool() {
				if ps.General {
					ctx.SyncDesc(&d12)
					if d12.Loc == scm.LocReg {
						ctx.ProtectReg(d12.Reg)
					} else if d12.Loc == scm.LocRegPair {
						ctx.ProtectReg(d12.Reg)
						ctx.ProtectReg(d12.Reg2)
					}
					d859 = d12
					if d859.Loc == scm.LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d859)
					d860 = d859
					if d860.Loc == scm.LocImm {
						d860 = scm.JITValueDesc{Loc: scm.LocImm, Type: d860.Type, Imm: scm.NewInt(int64(uint64(d860.Imm.Int()) & 0xffffffff))}
					} else {
						ctx.EmitShlRegImm8(d860.Reg, 32)
						ctx.EmitShrRegImm8(d860.Reg, 32)
					}
					ctx.EmitStoreToStack(d860, int32(bbs[2].PhiBase)+int32(0))
					if d12.Loc == scm.LocReg {
						ctx.UnprotectReg(d12.Reg)
					} else if d12.Loc == scm.LocRegPair {
						ctx.UnprotectReg(d12.Reg)
						ctx.UnprotectReg(d12.Reg2)
					}
				}
				ps861 := scm.PhiState{General: ps.General}
				ps861.OverlayValues = make([]scm.JITValueDesc, 861)
				ps861.OverlayValues[5] = d5
				ps861.OverlayValues[6] = d6
				ps861.OverlayValues[7] = d7
				ps861.OverlayValues[8] = d8
				ps861.OverlayValues[9] = d9
				ps861.OverlayValues[10] = d10
				ps861.OverlayValues[11] = d11
				ps861.OverlayValues[12] = d12
				ps861.OverlayValues[13] = d13
				ps861.OverlayValues[14] = d14
				ps861.OverlayValues[15] = d15
				ps861.OverlayValues[16] = d16
				ps861.OverlayValues[17] = d17
				ps861.OverlayValues[18] = d18
				ps861.OverlayValues[19] = d19
				ps861.OverlayValues[21] = d21
				ps861.OverlayValues[22] = d22
				ps861.OverlayValues[23] = d23
				ps861.OverlayValues[24] = d24
				ps861.OverlayValues[25] = d25
				ps861.OverlayValues[26] = d26
				ps861.OverlayValues[27] = d27
				ps861.OverlayValues[28] = d28
				ps861.OverlayValues[29] = d29
				ps861.OverlayValues[30] = d30
				ps861.OverlayValues[31] = d31
				ps861.OverlayValues[32] = d32
				ps861.OverlayValues[33] = d33
				ps861.OverlayValues[34] = d34
				ps861.OverlayValues[35] = d35
				ps861.OverlayValues[36] = d36
				ps861.OverlayValues[37] = d37
				ps861.OverlayValues[38] = d38
				ps861.OverlayValues[39] = d39
				ps861.OverlayValues[40] = d40
				ps861.OverlayValues[41] = d41
				ps861.OverlayValues[42] = d42
				ps861.OverlayValues[43] = d43
				ps861.OverlayValues[44] = d44
				ps861.OverlayValues[45] = d45
				ps861.OverlayValues[46] = d46
				ps861.OverlayValues[47] = d47
				ps861.OverlayValues[48] = d48
				ps861.OverlayValues[49] = d49
				ps861.OverlayValues[50] = d50
				ps861.OverlayValues[51] = d51
				ps861.OverlayValues[54] = d54
				ps861.OverlayValues[55] = d55
				ps861.OverlayValues[56] = d56
				ps861.OverlayValues[159] = d159
				ps861.OverlayValues[160] = d160
				ps861.OverlayValues[161] = d161
				ps861.OverlayValues[162] = d162
				ps861.OverlayValues[163] = d163
				ps861.OverlayValues[164] = d164
				ps861.OverlayValues[165] = d165
				ps861.OverlayValues[166] = d166
				ps861.OverlayValues[167] = d167
				ps861.OverlayValues[168] = d168
				ps861.OverlayValues[169] = d169
				ps861.OverlayValues[170] = d170
				ps861.OverlayValues[171] = d171
				ps861.OverlayValues[172] = d172
				ps861.OverlayValues[173] = d173
				ps861.OverlayValues[174] = d174
				ps861.OverlayValues[175] = d175
				ps861.OverlayValues[176] = d176
				ps861.OverlayValues[177] = d177
				ps861.OverlayValues[178] = d178
				ps861.OverlayValues[179] = d179
				ps861.OverlayValues[180] = d180
				ps861.OverlayValues[181] = d181
				ps861.OverlayValues[182] = d182
				ps861.OverlayValues[183] = d183
				ps861.OverlayValues[184] = d184
				ps861.OverlayValues[187] = d187
				ps861.OverlayValues[344] = d344
				ps861.OverlayValues[345] = d345
				ps861.OverlayValues[346] = d346
				ps861.OverlayValues[347] = d347
				ps861.OverlayValues[349] = d349
				ps861.OverlayValues[350] = d350
				ps861.OverlayValues[351] = d351
				ps861.OverlayValues[352] = d352
				ps861.OverlayValues[353] = d353
				ps861.OverlayValues[354] = d354
				ps861.OverlayValues[355] = d355
				ps861.OverlayValues[356] = d356
				ps861.OverlayValues[358] = d358
				ps861.OverlayValues[360] = d360
				ps861.OverlayValues[361] = d361
				ps861.OverlayValues[362] = d362
				ps861.OverlayValues[456] = d456
				ps861.OverlayValues[457] = d457
				ps861.OverlayValues[460] = d460
				ps861.OverlayValues[557] = d557
				ps861.OverlayValues[558] = d558
				ps861.OverlayValues[559] = d559
				ps861.OverlayValues[560] = d560
				ps861.OverlayValues[561] = d561
				ps861.OverlayValues[563] = d563
				ps861.OverlayValues[564] = d564
				ps861.OverlayValues[565] = d565
				ps861.OverlayValues[566] = d566
				ps861.OverlayValues[567] = d567
				ps861.OverlayValues[568] = d568
				ps861.OverlayValues[569] = d569
				ps861.OverlayValues[570] = d570
				ps861.OverlayValues[571] = d571
				ps861.OverlayValues[572] = d572
				ps861.OverlayValues[573] = d573
				ps861.OverlayValues[574] = d574
				ps861.OverlayValues[575] = d575
				ps861.OverlayValues[576] = d576
				ps861.OverlayValues[577] = d577
				ps861.OverlayValues[578] = d578
				ps861.OverlayValues[579] = d579
				ps861.OverlayValues[580] = d580
				ps861.OverlayValues[581] = d581
				ps861.OverlayValues[582] = d582
				ps861.OverlayValues[583] = d583
				ps861.OverlayValues[584] = d584
				ps861.OverlayValues[585] = d585
				ps861.OverlayValues[586] = d586
				ps861.OverlayValues[587] = d587
				ps861.OverlayValues[588] = d588
				ps861.OverlayValues[589] = d589
				ps861.OverlayValues[850] = d850
				ps861.OverlayValues[851] = d851
				ps861.OverlayValues[852] = d852
				ps861.OverlayValues[854] = d854
				ps861.OverlayValues[855] = d855
				ps861.OverlayValues[856] = d856
				ps861.OverlayValues[857] = d857
				ps861.OverlayValues[858] = d858
				ps861.OverlayValues[859] = d859
				ps861.OverlayValues[860] = d860
				ps861.PhiValues = make([]scm.JITValueDesc, 1)
				d862 = d12
				ps861.PhiValues[0] = d862
				return bbs[2].RenderPS(ps861)
			}
			if ps.General {
			}
			ps863 := scm.PhiState{General: ps.General}
			ps863.OverlayValues = make([]scm.JITValueDesc, 863)
			ps863.OverlayValues[5] = d5
			ps863.OverlayValues[6] = d6
			ps863.OverlayValues[7] = d7
			ps863.OverlayValues[8] = d8
			ps863.OverlayValues[9] = d9
			ps863.OverlayValues[10] = d10
			ps863.OverlayValues[11] = d11
			ps863.OverlayValues[12] = d12
			ps863.OverlayValues[13] = d13
			ps863.OverlayValues[14] = d14
			ps863.OverlayValues[15] = d15
			ps863.OverlayValues[16] = d16
			ps863.OverlayValues[17] = d17
			ps863.OverlayValues[18] = d18
			ps863.OverlayValues[19] = d19
			ps863.OverlayValues[21] = d21
			ps863.OverlayValues[22] = d22
			ps863.OverlayValues[23] = d23
			ps863.OverlayValues[24] = d24
			ps863.OverlayValues[25] = d25
			ps863.OverlayValues[26] = d26
			ps863.OverlayValues[27] = d27
			ps863.OverlayValues[28] = d28
			ps863.OverlayValues[29] = d29
			ps863.OverlayValues[30] = d30
			ps863.OverlayValues[31] = d31
			ps863.OverlayValues[32] = d32
			ps863.OverlayValues[33] = d33
			ps863.OverlayValues[34] = d34
			ps863.OverlayValues[35] = d35
			ps863.OverlayValues[36] = d36
			ps863.OverlayValues[37] = d37
			ps863.OverlayValues[38] = d38
			ps863.OverlayValues[39] = d39
			ps863.OverlayValues[40] = d40
			ps863.OverlayValues[41] = d41
			ps863.OverlayValues[42] = d42
			ps863.OverlayValues[43] = d43
			ps863.OverlayValues[44] = d44
			ps863.OverlayValues[45] = d45
			ps863.OverlayValues[46] = d46
			ps863.OverlayValues[47] = d47
			ps863.OverlayValues[48] = d48
			ps863.OverlayValues[49] = d49
			ps863.OverlayValues[50] = d50
			ps863.OverlayValues[51] = d51
			ps863.OverlayValues[54] = d54
			ps863.OverlayValues[55] = d55
			ps863.OverlayValues[56] = d56
			ps863.OverlayValues[159] = d159
			ps863.OverlayValues[160] = d160
			ps863.OverlayValues[161] = d161
			ps863.OverlayValues[162] = d162
			ps863.OverlayValues[163] = d163
			ps863.OverlayValues[164] = d164
			ps863.OverlayValues[165] = d165
			ps863.OverlayValues[166] = d166
			ps863.OverlayValues[167] = d167
			ps863.OverlayValues[168] = d168
			ps863.OverlayValues[169] = d169
			ps863.OverlayValues[170] = d170
			ps863.OverlayValues[171] = d171
			ps863.OverlayValues[172] = d172
			ps863.OverlayValues[173] = d173
			ps863.OverlayValues[174] = d174
			ps863.OverlayValues[175] = d175
			ps863.OverlayValues[176] = d176
			ps863.OverlayValues[177] = d177
			ps863.OverlayValues[178] = d178
			ps863.OverlayValues[179] = d179
			ps863.OverlayValues[180] = d180
			ps863.OverlayValues[181] = d181
			ps863.OverlayValues[182] = d182
			ps863.OverlayValues[183] = d183
			ps863.OverlayValues[184] = d184
			ps863.OverlayValues[187] = d187
			ps863.OverlayValues[344] = d344
			ps863.OverlayValues[345] = d345
			ps863.OverlayValues[346] = d346
			ps863.OverlayValues[347] = d347
			ps863.OverlayValues[349] = d349
			ps863.OverlayValues[350] = d350
			ps863.OverlayValues[351] = d351
			ps863.OverlayValues[352] = d352
			ps863.OverlayValues[353] = d353
			ps863.OverlayValues[354] = d354
			ps863.OverlayValues[355] = d355
			ps863.OverlayValues[356] = d356
			ps863.OverlayValues[358] = d358
			ps863.OverlayValues[360] = d360
			ps863.OverlayValues[361] = d361
			ps863.OverlayValues[362] = d362
			ps863.OverlayValues[456] = d456
			ps863.OverlayValues[457] = d457
			ps863.OverlayValues[460] = d460
			ps863.OverlayValues[557] = d557
			ps863.OverlayValues[558] = d558
			ps863.OverlayValues[559] = d559
			ps863.OverlayValues[560] = d560
			ps863.OverlayValues[561] = d561
			ps863.OverlayValues[563] = d563
			ps863.OverlayValues[564] = d564
			ps863.OverlayValues[565] = d565
			ps863.OverlayValues[566] = d566
			ps863.OverlayValues[567] = d567
			ps863.OverlayValues[568] = d568
			ps863.OverlayValues[569] = d569
			ps863.OverlayValues[570] = d570
			ps863.OverlayValues[571] = d571
			ps863.OverlayValues[572] = d572
			ps863.OverlayValues[573] = d573
			ps863.OverlayValues[574] = d574
			ps863.OverlayValues[575] = d575
			ps863.OverlayValues[576] = d576
			ps863.OverlayValues[577] = d577
			ps863.OverlayValues[578] = d578
			ps863.OverlayValues[579] = d579
			ps863.OverlayValues[580] = d580
			ps863.OverlayValues[581] = d581
			ps863.OverlayValues[582] = d582
			ps863.OverlayValues[583] = d583
			ps863.OverlayValues[584] = d584
			ps863.OverlayValues[585] = d585
			ps863.OverlayValues[586] = d586
			ps863.OverlayValues[587] = d587
			ps863.OverlayValues[588] = d588
			ps863.OverlayValues[589] = d589
			ps863.OverlayValues[850] = d850
			ps863.OverlayValues[851] = d851
			ps863.OverlayValues[852] = d852
			ps863.OverlayValues[854] = d854
			ps863.OverlayValues[855] = d855
			ps863.OverlayValues[856] = d856
			ps863.OverlayValues[857] = d857
			ps863.OverlayValues[858] = d858
			ps863.OverlayValues[859] = d859
			ps863.OverlayValues[860] = d860
			ps863.OverlayValues[862] = d862
			return bbs[10].RenderPS(ps863)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d864 := ps.PhiValues[0]
				ctx.EnsureDesc(&d864)
				ctx.EmitStoreToStack(d864, int32(bbs[8].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d865 := ps.PhiValues[1]
				ctx.EnsureDesc(&d865)
				ctx.EmitStoreToStack(d865, int32(bbs[8].PhiBase)+int32(16))
			}
			ps.General = true
			return bbs[8].RenderPS(ps)
		}
		lbl19 := ctx.ReserveLabel()
		ctx.EmitJump(d858.Condition, lbl19)
		ctx.EmitJmp(lbl11)
		ctx.FreeDesc(&d857)
		snap866 := d5
		snap867 := d6
		snap868 := d7
		snap869 := d8
		snap870 := d9
		snap871 := d10
		snap872 := d11
		snap873 := d12
		snap874 := d13
		snap875 := d14
		snap876 := d15
		snap877 := d16
		snap878 := d17
		snap879 := d18
		snap880 := d19
		snap881 := d21
		snap882 := d22
		snap883 := d23
		snap884 := d24
		snap885 := d25
		snap886 := d26
		snap887 := d27
		snap888 := d28
		snap889 := d29
		snap890 := d30
		snap891 := d31
		snap892 := d32
		snap893 := d33
		snap894 := d34
		snap895 := d35
		snap896 := d36
		snap897 := d37
		snap898 := d38
		snap899 := d39
		snap900 := d40
		snap901 := d41
		snap902 := d42
		snap903 := d43
		snap904 := d44
		snap905 := d45
		snap906 := d46
		snap907 := d47
		snap908 := d48
		snap909 := d49
		snap910 := d50
		snap911 := d51
		snap912 := d54
		snap913 := d55
		snap914 := d56
		snap915 := d159
		snap916 := d160
		snap917 := d161
		snap918 := d162
		snap919 := d163
		snap920 := d164
		snap921 := d165
		snap922 := d166
		snap923 := d167
		snap924 := d168
		snap925 := d169
		snap926 := d170
		snap927 := d171
		snap928 := d172
		snap929 := d173
		snap930 := d174
		snap931 := d175
		snap932 := d176
		snap933 := d177
		snap934 := d178
		snap935 := d179
		snap936 := d180
		snap937 := d181
		snap938 := d182
		snap939 := d183
		snap940 := d184
		snap941 := d187
		snap942 := d344
		snap943 := d345
		snap944 := d346
		snap945 := d347
		snap946 := d349
		snap947 := d350
		snap948 := d351
		snap949 := d352
		snap950 := d353
		snap951 := d354
		snap952 := d355
		snap953 := d356
		snap954 := d358
		snap955 := d360
		snap956 := d361
		snap957 := d362
		snap958 := d456
		snap959 := d457
		snap960 := d460
		snap961 := d557
		snap962 := d558
		snap963 := d559
		snap964 := d560
		snap965 := d561
		snap966 := d563
		snap967 := d564
		snap968 := d565
		snap969 := d566
		snap970 := d567
		snap971 := d568
		snap972 := d569
		snap973 := d570
		snap974 := d571
		snap975 := d572
		snap976 := d573
		snap977 := d574
		snap978 := d575
		snap979 := d576
		snap980 := d577
		snap981 := d578
		snap982 := d579
		snap983 := d580
		snap984 := d581
		snap985 := d582
		snap986 := d583
		snap987 := d584
		snap988 := d585
		snap989 := d586
		snap990 := d587
		snap991 := d588
		snap992 := d589
		snap993 := d850
		snap994 := d851
		snap995 := d852
		snap996 := d854
		snap997 := d855
		snap998 := d856
		snap999 := d857
		snap1000 := d858
		snap1001 := d859
		snap1002 := d860
		snap1003 := d862
		snap1004 := d864
		snap1005 := d865
		alloc1006 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl19)
		ctx.SyncDesc(&d12)
		if d12.Loc == scm.LocReg {
			ctx.ProtectReg(d12.Reg)
		} else if d12.Loc == scm.LocRegPair {
			ctx.ProtectReg(d12.Reg)
			ctx.ProtectReg(d12.Reg2)
		}
		d1007 = d12
		if d1007.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d1007)
		d1008 = d1007
		if d1008.Loc == scm.LocImm {
			d1008 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1008.Type, Imm: scm.NewInt(int64(uint64(d1008.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d1008.Reg, 32)
			ctx.EmitShrRegImm8(d1008.Reg, 32)
		}
		ctx.EmitStoreToStack(d1008, int32(bbs[2].PhiBase)+int32(0))
		if d12.Loc == scm.LocReg {
			ctx.UnprotectReg(d12.Reg)
		} else if d12.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d12.Reg)
			ctx.UnprotectReg(d12.Reg2)
		}
		ctx.EmitJmp(lbl3)
		ctx.RestoreAllocState(alloc1006)
		d5 = snap866
		d6 = snap867
		d7 = snap868
		d8 = snap869
		d9 = snap870
		d10 = snap871
		d11 = snap872
		d12 = snap873
		d13 = snap874
		d14 = snap875
		d15 = snap876
		d16 = snap877
		d17 = snap878
		d18 = snap879
		d19 = snap880
		d21 = snap881
		d22 = snap882
		d23 = snap883
		d24 = snap884
		d25 = snap885
		d26 = snap886
		d27 = snap887
		d28 = snap888
		d29 = snap889
		d30 = snap890
		d31 = snap891
		d32 = snap892
		d33 = snap893
		d34 = snap894
		d35 = snap895
		d36 = snap896
		d37 = snap897
		d38 = snap898
		d39 = snap899
		d40 = snap900
		d41 = snap901
		d42 = snap902
		d43 = snap903
		d44 = snap904
		d45 = snap905
		d46 = snap906
		d47 = snap907
		d48 = snap908
		d49 = snap909
		d50 = snap910
		d51 = snap911
		d54 = snap912
		d55 = snap913
		d56 = snap914
		d159 = snap915
		d160 = snap916
		d161 = snap917
		d162 = snap918
		d163 = snap919
		d164 = snap920
		d165 = snap921
		d166 = snap922
		d167 = snap923
		d168 = snap924
		d169 = snap925
		d170 = snap926
		d171 = snap927
		d172 = snap928
		d173 = snap929
		d174 = snap930
		d175 = snap931
		d176 = snap932
		d177 = snap933
		d178 = snap934
		d179 = snap935
		d180 = snap936
		d181 = snap937
		d182 = snap938
		d183 = snap939
		d184 = snap940
		d187 = snap941
		d344 = snap942
		d345 = snap943
		d346 = snap944
		d347 = snap945
		d349 = snap946
		d350 = snap947
		d351 = snap948
		d352 = snap949
		d353 = snap950
		d354 = snap951
		d355 = snap952
		d356 = snap953
		d358 = snap954
		d360 = snap955
		d361 = snap956
		d362 = snap957
		d456 = snap958
		d457 = snap959
		d460 = snap960
		d557 = snap961
		d558 = snap962
		d559 = snap963
		d560 = snap964
		d561 = snap965
		d563 = snap966
		d564 = snap967
		d565 = snap968
		d566 = snap969
		d567 = snap970
		d568 = snap971
		d569 = snap972
		d570 = snap973
		d571 = snap974
		d572 = snap975
		d573 = snap976
		d574 = snap977
		d575 = snap978
		d576 = snap979
		d577 = snap980
		d578 = snap981
		d579 = snap982
		d580 = snap983
		d581 = snap984
		d582 = snap985
		d583 = snap986
		d584 = snap987
		d585 = snap988
		d586 = snap989
		d587 = snap990
		d588 = snap991
		d589 = snap992
		d850 = snap993
		d851 = snap994
		d852 = snap995
		d854 = snap996
		d855 = snap997
		d856 = snap998
		d857 = snap999
		d858 = snap1000
		d859 = snap1001
		d860 = snap1002
		d862 = snap1003
		d864 = snap1004
		d865 = snap1005
		ctx.RestoreAllocState(alloc1006)
		d5 = snap866
		d6 = snap867
		d7 = snap868
		d8 = snap869
		d9 = snap870
		d10 = snap871
		d11 = snap872
		d12 = snap873
		d13 = snap874
		d14 = snap875
		d15 = snap876
		d16 = snap877
		d17 = snap878
		d18 = snap879
		d19 = snap880
		d21 = snap881
		d22 = snap882
		d23 = snap883
		d24 = snap884
		d25 = snap885
		d26 = snap886
		d27 = snap887
		d28 = snap888
		d29 = snap889
		d30 = snap890
		d31 = snap891
		d32 = snap892
		d33 = snap893
		d34 = snap894
		d35 = snap895
		d36 = snap896
		d37 = snap897
		d38 = snap898
		d39 = snap899
		d40 = snap900
		d41 = snap901
		d42 = snap902
		d43 = snap903
		d44 = snap904
		d45 = snap905
		d46 = snap906
		d47 = snap907
		d48 = snap908
		d49 = snap909
		d50 = snap910
		d51 = snap911
		d54 = snap912
		d55 = snap913
		d56 = snap914
		d159 = snap915
		d160 = snap916
		d161 = snap917
		d162 = snap918
		d163 = snap919
		d164 = snap920
		d165 = snap921
		d166 = snap922
		d167 = snap923
		d168 = snap924
		d169 = snap925
		d170 = snap926
		d171 = snap927
		d172 = snap928
		d173 = snap929
		d174 = snap930
		d175 = snap931
		d176 = snap932
		d177 = snap933
		d178 = snap934
		d179 = snap935
		d180 = snap936
		d181 = snap937
		d182 = snap938
		d183 = snap939
		d184 = snap940
		d187 = snap941
		d344 = snap942
		d345 = snap943
		d346 = snap944
		d347 = snap945
		d349 = snap946
		d350 = snap947
		d351 = snap948
		d352 = snap949
		d353 = snap950
		d354 = snap951
		d355 = snap952
		d356 = snap953
		d358 = snap954
		d360 = snap955
		d361 = snap956
		d362 = snap957
		d456 = snap958
		d457 = snap959
		d460 = snap960
		d557 = snap961
		d558 = snap962
		d559 = snap963
		d560 = snap964
		d561 = snap965
		d563 = snap966
		d564 = snap967
		d565 = snap968
		d566 = snap969
		d567 = snap970
		d568 = snap971
		d569 = snap972
		d570 = snap973
		d571 = snap974
		d572 = snap975
		d573 = snap976
		d574 = snap977
		d575 = snap978
		d576 = snap979
		d577 = snap980
		d578 = snap981
		d579 = snap982
		d580 = snap983
		d581 = snap984
		d582 = snap985
		d583 = snap986
		d584 = snap987
		d585 = snap988
		d586 = snap989
		d587 = snap990
		d588 = snap991
		d589 = snap992
		d850 = snap993
		d851 = snap994
		d852 = snap995
		d854 = snap996
		d855 = snap997
		d856 = snap998
		d857 = snap999
		d858 = snap1000
		d859 = snap1001
		d860 = snap1002
		d862 = snap1003
		d864 = snap1004
		d865 = snap1005
		ps1009 := scm.PhiState{General: true}
		ps1009.OverlayValues = make([]scm.JITValueDesc, 1009)
		ps1009.OverlayValues[5] = d5
		ps1009.OverlayValues[6] = d6
		ps1009.OverlayValues[7] = d7
		ps1009.OverlayValues[8] = d8
		ps1009.OverlayValues[9] = d9
		ps1009.OverlayValues[10] = d10
		ps1009.OverlayValues[11] = d11
		ps1009.OverlayValues[12] = d12
		ps1009.OverlayValues[13] = d13
		ps1009.OverlayValues[14] = d14
		ps1009.OverlayValues[15] = d15
		ps1009.OverlayValues[16] = d16
		ps1009.OverlayValues[17] = d17
		ps1009.OverlayValues[18] = d18
		ps1009.OverlayValues[19] = d19
		ps1009.OverlayValues[21] = d21
		ps1009.OverlayValues[22] = d22
		ps1009.OverlayValues[23] = d23
		ps1009.OverlayValues[24] = d24
		ps1009.OverlayValues[25] = d25
		ps1009.OverlayValues[26] = d26
		ps1009.OverlayValues[27] = d27
		ps1009.OverlayValues[28] = d28
		ps1009.OverlayValues[29] = d29
		ps1009.OverlayValues[30] = d30
		ps1009.OverlayValues[31] = d31
		ps1009.OverlayValues[32] = d32
		ps1009.OverlayValues[33] = d33
		ps1009.OverlayValues[34] = d34
		ps1009.OverlayValues[35] = d35
		ps1009.OverlayValues[36] = d36
		ps1009.OverlayValues[37] = d37
		ps1009.OverlayValues[38] = d38
		ps1009.OverlayValues[39] = d39
		ps1009.OverlayValues[40] = d40
		ps1009.OverlayValues[41] = d41
		ps1009.OverlayValues[42] = d42
		ps1009.OverlayValues[43] = d43
		ps1009.OverlayValues[44] = d44
		ps1009.OverlayValues[45] = d45
		ps1009.OverlayValues[46] = d46
		ps1009.OverlayValues[47] = d47
		ps1009.OverlayValues[48] = d48
		ps1009.OverlayValues[49] = d49
		ps1009.OverlayValues[50] = d50
		ps1009.OverlayValues[51] = d51
		ps1009.OverlayValues[54] = d54
		ps1009.OverlayValues[55] = d55
		ps1009.OverlayValues[56] = d56
		ps1009.OverlayValues[159] = d159
		ps1009.OverlayValues[160] = d160
		ps1009.OverlayValues[161] = d161
		ps1009.OverlayValues[162] = d162
		ps1009.OverlayValues[163] = d163
		ps1009.OverlayValues[164] = d164
		ps1009.OverlayValues[165] = d165
		ps1009.OverlayValues[166] = d166
		ps1009.OverlayValues[167] = d167
		ps1009.OverlayValues[168] = d168
		ps1009.OverlayValues[169] = d169
		ps1009.OverlayValues[170] = d170
		ps1009.OverlayValues[171] = d171
		ps1009.OverlayValues[172] = d172
		ps1009.OverlayValues[173] = d173
		ps1009.OverlayValues[174] = d174
		ps1009.OverlayValues[175] = d175
		ps1009.OverlayValues[176] = d176
		ps1009.OverlayValues[177] = d177
		ps1009.OverlayValues[178] = d178
		ps1009.OverlayValues[179] = d179
		ps1009.OverlayValues[180] = d180
		ps1009.OverlayValues[181] = d181
		ps1009.OverlayValues[182] = d182
		ps1009.OverlayValues[183] = d183
		ps1009.OverlayValues[184] = d184
		ps1009.OverlayValues[187] = d187
		ps1009.OverlayValues[344] = d344
		ps1009.OverlayValues[345] = d345
		ps1009.OverlayValues[346] = d346
		ps1009.OverlayValues[347] = d347
		ps1009.OverlayValues[349] = d349
		ps1009.OverlayValues[350] = d350
		ps1009.OverlayValues[351] = d351
		ps1009.OverlayValues[352] = d352
		ps1009.OverlayValues[353] = d353
		ps1009.OverlayValues[354] = d354
		ps1009.OverlayValues[355] = d355
		ps1009.OverlayValues[356] = d356
		ps1009.OverlayValues[358] = d358
		ps1009.OverlayValues[360] = d360
		ps1009.OverlayValues[361] = d361
		ps1009.OverlayValues[362] = d362
		ps1009.OverlayValues[456] = d456
		ps1009.OverlayValues[457] = d457
		ps1009.OverlayValues[460] = d460
		ps1009.OverlayValues[557] = d557
		ps1009.OverlayValues[558] = d558
		ps1009.OverlayValues[559] = d559
		ps1009.OverlayValues[560] = d560
		ps1009.OverlayValues[561] = d561
		ps1009.OverlayValues[563] = d563
		ps1009.OverlayValues[564] = d564
		ps1009.OverlayValues[565] = d565
		ps1009.OverlayValues[566] = d566
		ps1009.OverlayValues[567] = d567
		ps1009.OverlayValues[568] = d568
		ps1009.OverlayValues[569] = d569
		ps1009.OverlayValues[570] = d570
		ps1009.OverlayValues[571] = d571
		ps1009.OverlayValues[572] = d572
		ps1009.OverlayValues[573] = d573
		ps1009.OverlayValues[574] = d574
		ps1009.OverlayValues[575] = d575
		ps1009.OverlayValues[576] = d576
		ps1009.OverlayValues[577] = d577
		ps1009.OverlayValues[578] = d578
		ps1009.OverlayValues[579] = d579
		ps1009.OverlayValues[580] = d580
		ps1009.OverlayValues[581] = d581
		ps1009.OverlayValues[582] = d582
		ps1009.OverlayValues[583] = d583
		ps1009.OverlayValues[584] = d584
		ps1009.OverlayValues[585] = d585
		ps1009.OverlayValues[586] = d586
		ps1009.OverlayValues[587] = d587
		ps1009.OverlayValues[588] = d588
		ps1009.OverlayValues[589] = d589
		ps1009.OverlayValues[850] = d850
		ps1009.OverlayValues[851] = d851
		ps1009.OverlayValues[852] = d852
		ps1009.OverlayValues[854] = d854
		ps1009.OverlayValues[855] = d855
		ps1009.OverlayValues[856] = d856
		ps1009.OverlayValues[857] = d857
		ps1009.OverlayValues[858] = d858
		ps1009.OverlayValues[859] = d859
		ps1009.OverlayValues[860] = d860
		ps1009.OverlayValues[862] = d862
		ps1009.OverlayValues[864] = d864
		ps1009.OverlayValues[865] = d865
		ps1009.OverlayValues[1007] = d1007
		ps1009.OverlayValues[1008] = d1008
		ps1009.PhiValues = make([]scm.JITValueDesc, 1)
		d1011 = d12
		ps1009.PhiValues[0] = d1011
		ps1010 := scm.PhiState{General: true}
		ps1010.OverlayValues = make([]scm.JITValueDesc, 1012)
		ps1010.OverlayValues[5] = d5
		ps1010.OverlayValues[6] = d6
		ps1010.OverlayValues[7] = d7
		ps1010.OverlayValues[8] = d8
		ps1010.OverlayValues[9] = d9
		ps1010.OverlayValues[10] = d10
		ps1010.OverlayValues[11] = d11
		ps1010.OverlayValues[12] = d12
		ps1010.OverlayValues[13] = d13
		ps1010.OverlayValues[14] = d14
		ps1010.OverlayValues[15] = d15
		ps1010.OverlayValues[16] = d16
		ps1010.OverlayValues[17] = d17
		ps1010.OverlayValues[18] = d18
		ps1010.OverlayValues[19] = d19
		ps1010.OverlayValues[21] = d21
		ps1010.OverlayValues[22] = d22
		ps1010.OverlayValues[23] = d23
		ps1010.OverlayValues[24] = d24
		ps1010.OverlayValues[25] = d25
		ps1010.OverlayValues[26] = d26
		ps1010.OverlayValues[27] = d27
		ps1010.OverlayValues[28] = d28
		ps1010.OverlayValues[29] = d29
		ps1010.OverlayValues[30] = d30
		ps1010.OverlayValues[31] = d31
		ps1010.OverlayValues[32] = d32
		ps1010.OverlayValues[33] = d33
		ps1010.OverlayValues[34] = d34
		ps1010.OverlayValues[35] = d35
		ps1010.OverlayValues[36] = d36
		ps1010.OverlayValues[37] = d37
		ps1010.OverlayValues[38] = d38
		ps1010.OverlayValues[39] = d39
		ps1010.OverlayValues[40] = d40
		ps1010.OverlayValues[41] = d41
		ps1010.OverlayValues[42] = d42
		ps1010.OverlayValues[43] = d43
		ps1010.OverlayValues[44] = d44
		ps1010.OverlayValues[45] = d45
		ps1010.OverlayValues[46] = d46
		ps1010.OverlayValues[47] = d47
		ps1010.OverlayValues[48] = d48
		ps1010.OverlayValues[49] = d49
		ps1010.OverlayValues[50] = d50
		ps1010.OverlayValues[51] = d51
		ps1010.OverlayValues[54] = d54
		ps1010.OverlayValues[55] = d55
		ps1010.OverlayValues[56] = d56
		ps1010.OverlayValues[159] = d159
		ps1010.OverlayValues[160] = d160
		ps1010.OverlayValues[161] = d161
		ps1010.OverlayValues[162] = d162
		ps1010.OverlayValues[163] = d163
		ps1010.OverlayValues[164] = d164
		ps1010.OverlayValues[165] = d165
		ps1010.OverlayValues[166] = d166
		ps1010.OverlayValues[167] = d167
		ps1010.OverlayValues[168] = d168
		ps1010.OverlayValues[169] = d169
		ps1010.OverlayValues[170] = d170
		ps1010.OverlayValues[171] = d171
		ps1010.OverlayValues[172] = d172
		ps1010.OverlayValues[173] = d173
		ps1010.OverlayValues[174] = d174
		ps1010.OverlayValues[175] = d175
		ps1010.OverlayValues[176] = d176
		ps1010.OverlayValues[177] = d177
		ps1010.OverlayValues[178] = d178
		ps1010.OverlayValues[179] = d179
		ps1010.OverlayValues[180] = d180
		ps1010.OverlayValues[181] = d181
		ps1010.OverlayValues[182] = d182
		ps1010.OverlayValues[183] = d183
		ps1010.OverlayValues[184] = d184
		ps1010.OverlayValues[187] = d187
		ps1010.OverlayValues[344] = d344
		ps1010.OverlayValues[345] = d345
		ps1010.OverlayValues[346] = d346
		ps1010.OverlayValues[347] = d347
		ps1010.OverlayValues[349] = d349
		ps1010.OverlayValues[350] = d350
		ps1010.OverlayValues[351] = d351
		ps1010.OverlayValues[352] = d352
		ps1010.OverlayValues[353] = d353
		ps1010.OverlayValues[354] = d354
		ps1010.OverlayValues[355] = d355
		ps1010.OverlayValues[356] = d356
		ps1010.OverlayValues[358] = d358
		ps1010.OverlayValues[360] = d360
		ps1010.OverlayValues[361] = d361
		ps1010.OverlayValues[362] = d362
		ps1010.OverlayValues[456] = d456
		ps1010.OverlayValues[457] = d457
		ps1010.OverlayValues[460] = d460
		ps1010.OverlayValues[557] = d557
		ps1010.OverlayValues[558] = d558
		ps1010.OverlayValues[559] = d559
		ps1010.OverlayValues[560] = d560
		ps1010.OverlayValues[561] = d561
		ps1010.OverlayValues[563] = d563
		ps1010.OverlayValues[564] = d564
		ps1010.OverlayValues[565] = d565
		ps1010.OverlayValues[566] = d566
		ps1010.OverlayValues[567] = d567
		ps1010.OverlayValues[568] = d568
		ps1010.OverlayValues[569] = d569
		ps1010.OverlayValues[570] = d570
		ps1010.OverlayValues[571] = d571
		ps1010.OverlayValues[572] = d572
		ps1010.OverlayValues[573] = d573
		ps1010.OverlayValues[574] = d574
		ps1010.OverlayValues[575] = d575
		ps1010.OverlayValues[576] = d576
		ps1010.OverlayValues[577] = d577
		ps1010.OverlayValues[578] = d578
		ps1010.OverlayValues[579] = d579
		ps1010.OverlayValues[580] = d580
		ps1010.OverlayValues[581] = d581
		ps1010.OverlayValues[582] = d582
		ps1010.OverlayValues[583] = d583
		ps1010.OverlayValues[584] = d584
		ps1010.OverlayValues[585] = d585
		ps1010.OverlayValues[586] = d586
		ps1010.OverlayValues[587] = d587
		ps1010.OverlayValues[588] = d588
		ps1010.OverlayValues[589] = d589
		ps1010.OverlayValues[850] = d850
		ps1010.OverlayValues[851] = d851
		ps1010.OverlayValues[852] = d852
		ps1010.OverlayValues[854] = d854
		ps1010.OverlayValues[855] = d855
		ps1010.OverlayValues[856] = d856
		ps1010.OverlayValues[857] = d857
		ps1010.OverlayValues[858] = d858
		ps1010.OverlayValues[859] = d859
		ps1010.OverlayValues[860] = d860
		ps1010.OverlayValues[862] = d862
		ps1010.OverlayValues[864] = d864
		ps1010.OverlayValues[865] = d865
		ps1010.OverlayValues[1007] = d1007
		ps1010.OverlayValues[1008] = d1008
		ps1010.OverlayValues[1011] = d1011
		snap1012 := d5
		snap1013 := d6
		snap1014 := d7
		snap1015 := d8
		snap1016 := d9
		snap1017 := d10
		snap1018 := d11
		snap1019 := d12
		snap1020 := d13
		snap1021 := d14
		snap1022 := d15
		snap1023 := d16
		snap1024 := d17
		snap1025 := d18
		snap1026 := d19
		snap1027 := d21
		snap1028 := d22
		snap1029 := d23
		snap1030 := d24
		snap1031 := d25
		snap1032 := d26
		snap1033 := d27
		snap1034 := d28
		snap1035 := d29
		snap1036 := d30
		snap1037 := d31
		snap1038 := d32
		snap1039 := d33
		snap1040 := d34
		snap1041 := d35
		snap1042 := d36
		snap1043 := d37
		snap1044 := d38
		snap1045 := d39
		snap1046 := d40
		snap1047 := d41
		snap1048 := d42
		snap1049 := d43
		snap1050 := d44
		snap1051 := d45
		snap1052 := d46
		snap1053 := d47
		snap1054 := d48
		snap1055 := d49
		snap1056 := d50
		snap1057 := d51
		snap1058 := d54
		snap1059 := d55
		snap1060 := d56
		snap1061 := d159
		snap1062 := d160
		snap1063 := d161
		snap1064 := d162
		snap1065 := d163
		snap1066 := d164
		snap1067 := d165
		snap1068 := d166
		snap1069 := d167
		snap1070 := d168
		snap1071 := d169
		snap1072 := d170
		snap1073 := d171
		snap1074 := d172
		snap1075 := d173
		snap1076 := d174
		snap1077 := d175
		snap1078 := d176
		snap1079 := d177
		snap1080 := d178
		snap1081 := d179
		snap1082 := d180
		snap1083 := d181
		snap1084 := d182
		snap1085 := d183
		snap1086 := d184
		snap1087 := d187
		snap1088 := d344
		snap1089 := d345
		snap1090 := d346
		snap1091 := d347
		snap1092 := d349
		snap1093 := d350
		snap1094 := d351
		snap1095 := d352
		snap1096 := d353
		snap1097 := d354
		snap1098 := d355
		snap1099 := d356
		snap1100 := d358
		snap1101 := d360
		snap1102 := d361
		snap1103 := d362
		snap1104 := d456
		snap1105 := d457
		snap1106 := d460
		snap1107 := d557
		snap1108 := d558
		snap1109 := d559
		snap1110 := d560
		snap1111 := d561
		snap1112 := d563
		snap1113 := d564
		snap1114 := d565
		snap1115 := d566
		snap1116 := d567
		snap1117 := d568
		snap1118 := d569
		snap1119 := d570
		snap1120 := d571
		snap1121 := d572
		snap1122 := d573
		snap1123 := d574
		snap1124 := d575
		snap1125 := d576
		snap1126 := d577
		snap1127 := d578
		snap1128 := d579
		snap1129 := d580
		snap1130 := d581
		snap1131 := d582
		snap1132 := d583
		snap1133 := d584
		snap1134 := d585
		snap1135 := d586
		snap1136 := d587
		snap1137 := d588
		snap1138 := d589
		snap1139 := d850
		snap1140 := d851
		snap1141 := d852
		snap1142 := d854
		snap1143 := d855
		snap1144 := d856
		snap1145 := d857
		snap1146 := d858
		snap1147 := d859
		snap1148 := d860
		snap1149 := d862
		snap1150 := d864
		snap1151 := d865
		snap1152 := d1007
		snap1153 := d1008
		snap1154 := d1011
		alloc1155 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps1009)
		}
		ctx.RestoreAllocState(alloc1155)
		d5 = snap1012
		d6 = snap1013
		d7 = snap1014
		d8 = snap1015
		d9 = snap1016
		d10 = snap1017
		d11 = snap1018
		d12 = snap1019
		d13 = snap1020
		d14 = snap1021
		d15 = snap1022
		d16 = snap1023
		d17 = snap1024
		d18 = snap1025
		d19 = snap1026
		d21 = snap1027
		d22 = snap1028
		d23 = snap1029
		d24 = snap1030
		d25 = snap1031
		d26 = snap1032
		d27 = snap1033
		d28 = snap1034
		d29 = snap1035
		d30 = snap1036
		d31 = snap1037
		d32 = snap1038
		d33 = snap1039
		d34 = snap1040
		d35 = snap1041
		d36 = snap1042
		d37 = snap1043
		d38 = snap1044
		d39 = snap1045
		d40 = snap1046
		d41 = snap1047
		d42 = snap1048
		d43 = snap1049
		d44 = snap1050
		d45 = snap1051
		d46 = snap1052
		d47 = snap1053
		d48 = snap1054
		d49 = snap1055
		d50 = snap1056
		d51 = snap1057
		d54 = snap1058
		d55 = snap1059
		d56 = snap1060
		d159 = snap1061
		d160 = snap1062
		d161 = snap1063
		d162 = snap1064
		d163 = snap1065
		d164 = snap1066
		d165 = snap1067
		d166 = snap1068
		d167 = snap1069
		d168 = snap1070
		d169 = snap1071
		d170 = snap1072
		d171 = snap1073
		d172 = snap1074
		d173 = snap1075
		d174 = snap1076
		d175 = snap1077
		d176 = snap1078
		d177 = snap1079
		d178 = snap1080
		d179 = snap1081
		d180 = snap1082
		d181 = snap1083
		d182 = snap1084
		d183 = snap1085
		d184 = snap1086
		d187 = snap1087
		d344 = snap1088
		d345 = snap1089
		d346 = snap1090
		d347 = snap1091
		d349 = snap1092
		d350 = snap1093
		d351 = snap1094
		d352 = snap1095
		d353 = snap1096
		d354 = snap1097
		d355 = snap1098
		d356 = snap1099
		d358 = snap1100
		d360 = snap1101
		d361 = snap1102
		d362 = snap1103
		d456 = snap1104
		d457 = snap1105
		d460 = snap1106
		d557 = snap1107
		d558 = snap1108
		d559 = snap1109
		d560 = snap1110
		d561 = snap1111
		d563 = snap1112
		d564 = snap1113
		d565 = snap1114
		d566 = snap1115
		d567 = snap1116
		d568 = snap1117
		d569 = snap1118
		d570 = snap1119
		d571 = snap1120
		d572 = snap1121
		d573 = snap1122
		d574 = snap1123
		d575 = snap1124
		d576 = snap1125
		d577 = snap1126
		d578 = snap1127
		d579 = snap1128
		d580 = snap1129
		d581 = snap1130
		d582 = snap1131
		d583 = snap1132
		d584 = snap1133
		d585 = snap1134
		d586 = snap1135
		d587 = snap1136
		d588 = snap1137
		d589 = snap1138
		d850 = snap1139
		d851 = snap1140
		d852 = snap1141
		d854 = snap1142
		d855 = snap1143
		d856 = snap1144
		d857 = snap1145
		d858 = snap1146
		d859 = snap1147
		d860 = snap1148
		d862 = snap1149
		d864 = snap1150
		d865 = snap1151
		d1007 = snap1152
		d1008 = snap1153
		d1011 = snap1154
		if !bbs[10].Rendered {
			return bbs[10].RenderPS(ps1010)
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
		if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
			d187 = ps.OverlayValues[187]
		}
		if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != scm.LocNone {
			d344 = ps.OverlayValues[344]
		}
		if len(ps.OverlayValues) > 345 && ps.OverlayValues[345].Loc != scm.LocNone {
			d345 = ps.OverlayValues[345]
		}
		if len(ps.OverlayValues) > 346 && ps.OverlayValues[346].Loc != scm.LocNone {
			d346 = ps.OverlayValues[346]
		}
		if len(ps.OverlayValues) > 347 && ps.OverlayValues[347].Loc != scm.LocNone {
			d347 = ps.OverlayValues[347]
		}
		if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != scm.LocNone {
			d349 = ps.OverlayValues[349]
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
		if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != scm.LocNone {
			d355 = ps.OverlayValues[355]
		}
		if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != scm.LocNone {
			d356 = ps.OverlayValues[356]
		}
		if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != scm.LocNone {
			d358 = ps.OverlayValues[358]
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
		if len(ps.OverlayValues) > 456 && ps.OverlayValues[456].Loc != scm.LocNone {
			d456 = ps.OverlayValues[456]
		}
		if len(ps.OverlayValues) > 457 && ps.OverlayValues[457].Loc != scm.LocNone {
			d457 = ps.OverlayValues[457]
		}
		if len(ps.OverlayValues) > 460 && ps.OverlayValues[460].Loc != scm.LocNone {
			d460 = ps.OverlayValues[460]
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
		if len(ps.OverlayValues) > 561 && ps.OverlayValues[561].Loc != scm.LocNone {
			d561 = ps.OverlayValues[561]
		}
		if len(ps.OverlayValues) > 563 && ps.OverlayValues[563].Loc != scm.LocNone {
			d563 = ps.OverlayValues[563]
		}
		if len(ps.OverlayValues) > 564 && ps.OverlayValues[564].Loc != scm.LocNone {
			d564 = ps.OverlayValues[564]
		}
		if len(ps.OverlayValues) > 565 && ps.OverlayValues[565].Loc != scm.LocNone {
			d565 = ps.OverlayValues[565]
		}
		if len(ps.OverlayValues) > 566 && ps.OverlayValues[566].Loc != scm.LocNone {
			d566 = ps.OverlayValues[566]
		}
		if len(ps.OverlayValues) > 567 && ps.OverlayValues[567].Loc != scm.LocNone {
			d567 = ps.OverlayValues[567]
		}
		if len(ps.OverlayValues) > 568 && ps.OverlayValues[568].Loc != scm.LocNone {
			d568 = ps.OverlayValues[568]
		}
		if len(ps.OverlayValues) > 569 && ps.OverlayValues[569].Loc != scm.LocNone {
			d569 = ps.OverlayValues[569]
		}
		if len(ps.OverlayValues) > 570 && ps.OverlayValues[570].Loc != scm.LocNone {
			d570 = ps.OverlayValues[570]
		}
		if len(ps.OverlayValues) > 571 && ps.OverlayValues[571].Loc != scm.LocNone {
			d571 = ps.OverlayValues[571]
		}
		if len(ps.OverlayValues) > 572 && ps.OverlayValues[572].Loc != scm.LocNone {
			d572 = ps.OverlayValues[572]
		}
		if len(ps.OverlayValues) > 573 && ps.OverlayValues[573].Loc != scm.LocNone {
			d573 = ps.OverlayValues[573]
		}
		if len(ps.OverlayValues) > 574 && ps.OverlayValues[574].Loc != scm.LocNone {
			d574 = ps.OverlayValues[574]
		}
		if len(ps.OverlayValues) > 575 && ps.OverlayValues[575].Loc != scm.LocNone {
			d575 = ps.OverlayValues[575]
		}
		if len(ps.OverlayValues) > 576 && ps.OverlayValues[576].Loc != scm.LocNone {
			d576 = ps.OverlayValues[576]
		}
		if len(ps.OverlayValues) > 577 && ps.OverlayValues[577].Loc != scm.LocNone {
			d577 = ps.OverlayValues[577]
		}
		if len(ps.OverlayValues) > 578 && ps.OverlayValues[578].Loc != scm.LocNone {
			d578 = ps.OverlayValues[578]
		}
		if len(ps.OverlayValues) > 579 && ps.OverlayValues[579].Loc != scm.LocNone {
			d579 = ps.OverlayValues[579]
		}
		if len(ps.OverlayValues) > 580 && ps.OverlayValues[580].Loc != scm.LocNone {
			d580 = ps.OverlayValues[580]
		}
		if len(ps.OverlayValues) > 581 && ps.OverlayValues[581].Loc != scm.LocNone {
			d581 = ps.OverlayValues[581]
		}
		if len(ps.OverlayValues) > 582 && ps.OverlayValues[582].Loc != scm.LocNone {
			d582 = ps.OverlayValues[582]
		}
		if len(ps.OverlayValues) > 583 && ps.OverlayValues[583].Loc != scm.LocNone {
			d583 = ps.OverlayValues[583]
		}
		if len(ps.OverlayValues) > 584 && ps.OverlayValues[584].Loc != scm.LocNone {
			d584 = ps.OverlayValues[584]
		}
		if len(ps.OverlayValues) > 585 && ps.OverlayValues[585].Loc != scm.LocNone {
			d585 = ps.OverlayValues[585]
		}
		if len(ps.OverlayValues) > 586 && ps.OverlayValues[586].Loc != scm.LocNone {
			d586 = ps.OverlayValues[586]
		}
		if len(ps.OverlayValues) > 587 && ps.OverlayValues[587].Loc != scm.LocNone {
			d587 = ps.OverlayValues[587]
		}
		if len(ps.OverlayValues) > 588 && ps.OverlayValues[588].Loc != scm.LocNone {
			d588 = ps.OverlayValues[588]
		}
		if len(ps.OverlayValues) > 589 && ps.OverlayValues[589].Loc != scm.LocNone {
			d589 = ps.OverlayValues[589]
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
		if len(ps.OverlayValues) > 862 && ps.OverlayValues[862].Loc != scm.LocNone {
			d862 = ps.OverlayValues[862]
		}
		if len(ps.OverlayValues) > 864 && ps.OverlayValues[864].Loc != scm.LocNone {
			d864 = ps.OverlayValues[864]
		}
		if len(ps.OverlayValues) > 865 && ps.OverlayValues[865].Loc != scm.LocNone {
			d865 = ps.OverlayValues[865]
		}
		if len(ps.OverlayValues) > 1007 && ps.OverlayValues[1007].Loc != scm.LocNone {
			d1007 = ps.OverlayValues[1007]
		}
		if len(ps.OverlayValues) > 1008 && ps.OverlayValues[1008].Loc != scm.LocNone {
			d1008 = ps.OverlayValues[1008]
		}
		if len(ps.OverlayValues) > 1011 && ps.OverlayValues[1011].Loc != scm.LocNone {
			d1011 = ps.OverlayValues[1011]
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
			d1156 = d9
			if d1156.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d1156)
			d1157 = d1156
			if d1157.Loc == scm.LocImm {
				d1157 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1157.Type, Imm: scm.NewInt(int64(uint64(d1157.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d1157.Reg, 32)
				ctx.EmitShrRegImm8(d1157.Reg, 32)
			}
			ctx.EmitStoreToStack(d1157, int32(bbs[8].PhiBase)+int32(0))
			d1158 = d11
			if d1158.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d1158)
			d1159 = d1158
			if d1159.Loc == scm.LocImm {
				d1159 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1159.Type, Imm: scm.NewInt(int64(uint64(d1159.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d1159.Reg, 32)
				ctx.EmitShrRegImm8(d1159.Reg, 32)
			}
			ctx.EmitStoreToStack(d1159, int32(bbs[8].PhiBase)+int32(16))
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
		ps1160 := scm.PhiState{General: ps.General}
		ps1160.OverlayValues = make([]scm.JITValueDesc, 1160)
		ps1160.OverlayValues[5] = d5
		ps1160.OverlayValues[6] = d6
		ps1160.OverlayValues[7] = d7
		ps1160.OverlayValues[8] = d8
		ps1160.OverlayValues[9] = d9
		ps1160.OverlayValues[10] = d10
		ps1160.OverlayValues[11] = d11
		ps1160.OverlayValues[12] = d12
		ps1160.OverlayValues[13] = d13
		ps1160.OverlayValues[14] = d14
		ps1160.OverlayValues[15] = d15
		ps1160.OverlayValues[16] = d16
		ps1160.OverlayValues[17] = d17
		ps1160.OverlayValues[18] = d18
		ps1160.OverlayValues[19] = d19
		ps1160.OverlayValues[21] = d21
		ps1160.OverlayValues[22] = d22
		ps1160.OverlayValues[23] = d23
		ps1160.OverlayValues[24] = d24
		ps1160.OverlayValues[25] = d25
		ps1160.OverlayValues[26] = d26
		ps1160.OverlayValues[27] = d27
		ps1160.OverlayValues[28] = d28
		ps1160.OverlayValues[29] = d29
		ps1160.OverlayValues[30] = d30
		ps1160.OverlayValues[31] = d31
		ps1160.OverlayValues[32] = d32
		ps1160.OverlayValues[33] = d33
		ps1160.OverlayValues[34] = d34
		ps1160.OverlayValues[35] = d35
		ps1160.OverlayValues[36] = d36
		ps1160.OverlayValues[37] = d37
		ps1160.OverlayValues[38] = d38
		ps1160.OverlayValues[39] = d39
		ps1160.OverlayValues[40] = d40
		ps1160.OverlayValues[41] = d41
		ps1160.OverlayValues[42] = d42
		ps1160.OverlayValues[43] = d43
		ps1160.OverlayValues[44] = d44
		ps1160.OverlayValues[45] = d45
		ps1160.OverlayValues[46] = d46
		ps1160.OverlayValues[47] = d47
		ps1160.OverlayValues[48] = d48
		ps1160.OverlayValues[49] = d49
		ps1160.OverlayValues[50] = d50
		ps1160.OverlayValues[51] = d51
		ps1160.OverlayValues[54] = d54
		ps1160.OverlayValues[55] = d55
		ps1160.OverlayValues[56] = d56
		ps1160.OverlayValues[159] = d159
		ps1160.OverlayValues[160] = d160
		ps1160.OverlayValues[161] = d161
		ps1160.OverlayValues[162] = d162
		ps1160.OverlayValues[163] = d163
		ps1160.OverlayValues[164] = d164
		ps1160.OverlayValues[165] = d165
		ps1160.OverlayValues[166] = d166
		ps1160.OverlayValues[167] = d167
		ps1160.OverlayValues[168] = d168
		ps1160.OverlayValues[169] = d169
		ps1160.OverlayValues[170] = d170
		ps1160.OverlayValues[171] = d171
		ps1160.OverlayValues[172] = d172
		ps1160.OverlayValues[173] = d173
		ps1160.OverlayValues[174] = d174
		ps1160.OverlayValues[175] = d175
		ps1160.OverlayValues[176] = d176
		ps1160.OverlayValues[177] = d177
		ps1160.OverlayValues[178] = d178
		ps1160.OverlayValues[179] = d179
		ps1160.OverlayValues[180] = d180
		ps1160.OverlayValues[181] = d181
		ps1160.OverlayValues[182] = d182
		ps1160.OverlayValues[183] = d183
		ps1160.OverlayValues[184] = d184
		ps1160.OverlayValues[187] = d187
		ps1160.OverlayValues[344] = d344
		ps1160.OverlayValues[345] = d345
		ps1160.OverlayValues[346] = d346
		ps1160.OverlayValues[347] = d347
		ps1160.OverlayValues[349] = d349
		ps1160.OverlayValues[350] = d350
		ps1160.OverlayValues[351] = d351
		ps1160.OverlayValues[352] = d352
		ps1160.OverlayValues[353] = d353
		ps1160.OverlayValues[354] = d354
		ps1160.OverlayValues[355] = d355
		ps1160.OverlayValues[356] = d356
		ps1160.OverlayValues[358] = d358
		ps1160.OverlayValues[360] = d360
		ps1160.OverlayValues[361] = d361
		ps1160.OverlayValues[362] = d362
		ps1160.OverlayValues[456] = d456
		ps1160.OverlayValues[457] = d457
		ps1160.OverlayValues[460] = d460
		ps1160.OverlayValues[557] = d557
		ps1160.OverlayValues[558] = d558
		ps1160.OverlayValues[559] = d559
		ps1160.OverlayValues[560] = d560
		ps1160.OverlayValues[561] = d561
		ps1160.OverlayValues[563] = d563
		ps1160.OverlayValues[564] = d564
		ps1160.OverlayValues[565] = d565
		ps1160.OverlayValues[566] = d566
		ps1160.OverlayValues[567] = d567
		ps1160.OverlayValues[568] = d568
		ps1160.OverlayValues[569] = d569
		ps1160.OverlayValues[570] = d570
		ps1160.OverlayValues[571] = d571
		ps1160.OverlayValues[572] = d572
		ps1160.OverlayValues[573] = d573
		ps1160.OverlayValues[574] = d574
		ps1160.OverlayValues[575] = d575
		ps1160.OverlayValues[576] = d576
		ps1160.OverlayValues[577] = d577
		ps1160.OverlayValues[578] = d578
		ps1160.OverlayValues[579] = d579
		ps1160.OverlayValues[580] = d580
		ps1160.OverlayValues[581] = d581
		ps1160.OverlayValues[582] = d582
		ps1160.OverlayValues[583] = d583
		ps1160.OverlayValues[584] = d584
		ps1160.OverlayValues[585] = d585
		ps1160.OverlayValues[586] = d586
		ps1160.OverlayValues[587] = d587
		ps1160.OverlayValues[588] = d588
		ps1160.OverlayValues[589] = d589
		ps1160.OverlayValues[850] = d850
		ps1160.OverlayValues[851] = d851
		ps1160.OverlayValues[852] = d852
		ps1160.OverlayValues[854] = d854
		ps1160.OverlayValues[855] = d855
		ps1160.OverlayValues[856] = d856
		ps1160.OverlayValues[857] = d857
		ps1160.OverlayValues[858] = d858
		ps1160.OverlayValues[859] = d859
		ps1160.OverlayValues[860] = d860
		ps1160.OverlayValues[862] = d862
		ps1160.OverlayValues[864] = d864
		ps1160.OverlayValues[865] = d865
		ps1160.OverlayValues[1007] = d1007
		ps1160.OverlayValues[1008] = d1008
		ps1160.OverlayValues[1011] = d1011
		ps1160.OverlayValues[1156] = d1156
		ps1160.OverlayValues[1157] = d1157
		ps1160.OverlayValues[1158] = d1158
		ps1160.OverlayValues[1159] = d1159
		ps1160.PhiValues = make([]scm.JITValueDesc, 2)
		d1161 = d9
		ps1160.PhiValues[0] = d1161
		d1162 = d11
		ps1160.PhiValues[1] = d1162
		if ps1160.General && bbs[8].Rendered {
			ctx.EmitJmp(lbl9)
			return result
		}
		return bbs[8].RenderPS(ps1160)
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
		if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
			d187 = ps.OverlayValues[187]
		}
		if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != scm.LocNone {
			d344 = ps.OverlayValues[344]
		}
		if len(ps.OverlayValues) > 345 && ps.OverlayValues[345].Loc != scm.LocNone {
			d345 = ps.OverlayValues[345]
		}
		if len(ps.OverlayValues) > 346 && ps.OverlayValues[346].Loc != scm.LocNone {
			d346 = ps.OverlayValues[346]
		}
		if len(ps.OverlayValues) > 347 && ps.OverlayValues[347].Loc != scm.LocNone {
			d347 = ps.OverlayValues[347]
		}
		if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != scm.LocNone {
			d349 = ps.OverlayValues[349]
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
		if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != scm.LocNone {
			d355 = ps.OverlayValues[355]
		}
		if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != scm.LocNone {
			d356 = ps.OverlayValues[356]
		}
		if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != scm.LocNone {
			d358 = ps.OverlayValues[358]
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
		if len(ps.OverlayValues) > 456 && ps.OverlayValues[456].Loc != scm.LocNone {
			d456 = ps.OverlayValues[456]
		}
		if len(ps.OverlayValues) > 457 && ps.OverlayValues[457].Loc != scm.LocNone {
			d457 = ps.OverlayValues[457]
		}
		if len(ps.OverlayValues) > 460 && ps.OverlayValues[460].Loc != scm.LocNone {
			d460 = ps.OverlayValues[460]
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
		if len(ps.OverlayValues) > 561 && ps.OverlayValues[561].Loc != scm.LocNone {
			d561 = ps.OverlayValues[561]
		}
		if len(ps.OverlayValues) > 563 && ps.OverlayValues[563].Loc != scm.LocNone {
			d563 = ps.OverlayValues[563]
		}
		if len(ps.OverlayValues) > 564 && ps.OverlayValues[564].Loc != scm.LocNone {
			d564 = ps.OverlayValues[564]
		}
		if len(ps.OverlayValues) > 565 && ps.OverlayValues[565].Loc != scm.LocNone {
			d565 = ps.OverlayValues[565]
		}
		if len(ps.OverlayValues) > 566 && ps.OverlayValues[566].Loc != scm.LocNone {
			d566 = ps.OverlayValues[566]
		}
		if len(ps.OverlayValues) > 567 && ps.OverlayValues[567].Loc != scm.LocNone {
			d567 = ps.OverlayValues[567]
		}
		if len(ps.OverlayValues) > 568 && ps.OverlayValues[568].Loc != scm.LocNone {
			d568 = ps.OverlayValues[568]
		}
		if len(ps.OverlayValues) > 569 && ps.OverlayValues[569].Loc != scm.LocNone {
			d569 = ps.OverlayValues[569]
		}
		if len(ps.OverlayValues) > 570 && ps.OverlayValues[570].Loc != scm.LocNone {
			d570 = ps.OverlayValues[570]
		}
		if len(ps.OverlayValues) > 571 && ps.OverlayValues[571].Loc != scm.LocNone {
			d571 = ps.OverlayValues[571]
		}
		if len(ps.OverlayValues) > 572 && ps.OverlayValues[572].Loc != scm.LocNone {
			d572 = ps.OverlayValues[572]
		}
		if len(ps.OverlayValues) > 573 && ps.OverlayValues[573].Loc != scm.LocNone {
			d573 = ps.OverlayValues[573]
		}
		if len(ps.OverlayValues) > 574 && ps.OverlayValues[574].Loc != scm.LocNone {
			d574 = ps.OverlayValues[574]
		}
		if len(ps.OverlayValues) > 575 && ps.OverlayValues[575].Loc != scm.LocNone {
			d575 = ps.OverlayValues[575]
		}
		if len(ps.OverlayValues) > 576 && ps.OverlayValues[576].Loc != scm.LocNone {
			d576 = ps.OverlayValues[576]
		}
		if len(ps.OverlayValues) > 577 && ps.OverlayValues[577].Loc != scm.LocNone {
			d577 = ps.OverlayValues[577]
		}
		if len(ps.OverlayValues) > 578 && ps.OverlayValues[578].Loc != scm.LocNone {
			d578 = ps.OverlayValues[578]
		}
		if len(ps.OverlayValues) > 579 && ps.OverlayValues[579].Loc != scm.LocNone {
			d579 = ps.OverlayValues[579]
		}
		if len(ps.OverlayValues) > 580 && ps.OverlayValues[580].Loc != scm.LocNone {
			d580 = ps.OverlayValues[580]
		}
		if len(ps.OverlayValues) > 581 && ps.OverlayValues[581].Loc != scm.LocNone {
			d581 = ps.OverlayValues[581]
		}
		if len(ps.OverlayValues) > 582 && ps.OverlayValues[582].Loc != scm.LocNone {
			d582 = ps.OverlayValues[582]
		}
		if len(ps.OverlayValues) > 583 && ps.OverlayValues[583].Loc != scm.LocNone {
			d583 = ps.OverlayValues[583]
		}
		if len(ps.OverlayValues) > 584 && ps.OverlayValues[584].Loc != scm.LocNone {
			d584 = ps.OverlayValues[584]
		}
		if len(ps.OverlayValues) > 585 && ps.OverlayValues[585].Loc != scm.LocNone {
			d585 = ps.OverlayValues[585]
		}
		if len(ps.OverlayValues) > 586 && ps.OverlayValues[586].Loc != scm.LocNone {
			d586 = ps.OverlayValues[586]
		}
		if len(ps.OverlayValues) > 587 && ps.OverlayValues[587].Loc != scm.LocNone {
			d587 = ps.OverlayValues[587]
		}
		if len(ps.OverlayValues) > 588 && ps.OverlayValues[588].Loc != scm.LocNone {
			d588 = ps.OverlayValues[588]
		}
		if len(ps.OverlayValues) > 589 && ps.OverlayValues[589].Loc != scm.LocNone {
			d589 = ps.OverlayValues[589]
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
		if len(ps.OverlayValues) > 862 && ps.OverlayValues[862].Loc != scm.LocNone {
			d862 = ps.OverlayValues[862]
		}
		if len(ps.OverlayValues) > 864 && ps.OverlayValues[864].Loc != scm.LocNone {
			d864 = ps.OverlayValues[864]
		}
		if len(ps.OverlayValues) > 865 && ps.OverlayValues[865].Loc != scm.LocNone {
			d865 = ps.OverlayValues[865]
		}
		if len(ps.OverlayValues) > 1007 && ps.OverlayValues[1007].Loc != scm.LocNone {
			d1007 = ps.OverlayValues[1007]
		}
		if len(ps.OverlayValues) > 1008 && ps.OverlayValues[1008].Loc != scm.LocNone {
			d1008 = ps.OverlayValues[1008]
		}
		if len(ps.OverlayValues) > 1011 && ps.OverlayValues[1011].Loc != scm.LocNone {
			d1011 = ps.OverlayValues[1011]
		}
		if len(ps.OverlayValues) > 1156 && ps.OverlayValues[1156].Loc != scm.LocNone {
			d1156 = ps.OverlayValues[1156]
		}
		if len(ps.OverlayValues) > 1157 && ps.OverlayValues[1157].Loc != scm.LocNone {
			d1157 = ps.OverlayValues[1157]
		}
		if len(ps.OverlayValues) > 1158 && ps.OverlayValues[1158].Loc != scm.LocNone {
			d1158 = ps.OverlayValues[1158]
		}
		if len(ps.OverlayValues) > 1159 && ps.OverlayValues[1159].Loc != scm.LocNone {
			d1159 = ps.OverlayValues[1159]
		}
		if len(ps.OverlayValues) > 1161 && ps.OverlayValues[1161].Loc != scm.LocNone {
			d1161 = ps.OverlayValues[1161]
		}
		if len(ps.OverlayValues) > 1162 && ps.OverlayValues[1162].Loc != scm.LocNone {
			d1162 = ps.OverlayValues[1162]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d12)
		ctx.EnsureDesc(&d13)
		ctx.EnsureDescsTogether(&d12, &d13)
		var d1163 scm.JITValueDesc
		if d12.Loc == scm.LocImm && d13.Loc == scm.LocImm {
			d1163 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() + d13.Imm.Int())}
		} else if d13.Loc == scm.LocImm && d13.Imm.Int() == 0 {
			r102 := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitMovRegReg(r102, d12.Reg)
			d1163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r102}
			ctx.BindReg(r102, &d1163)
		} else if d12.Loc == scm.LocImm && d12.Imm.Int() == 0 {
			d1163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d13.Reg}
			ctx.BindReg(d13.Reg, &d1163)
		} else if d12.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d13.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d12.Imm.Int()))
			ctx.EmitAddInt32(scratch, d13.Reg)
			d1163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1163)
		} else if d13.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitMovRegReg(scratch, d12.Reg)
			if d13.Imm.Int() >= -2147483648 && d13.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32Low(scratch, int32(d13.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
				ctx.EmitAddInt32(scratch, scm.RegR11)
			}
			d1163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1163)
		} else {
			r103 := ctx.AllocRegExcept(d12.Reg, d13.Reg)
			ctx.EmitMovRegReg(r103, d12.Reg)
			ctx.EmitAddInt32(r103, d13.Reg)
			d1163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r103}
			ctx.BindReg(r103, &d1163)
		}
		if d1163.Loc == scm.LocReg && d12.Loc == scm.LocReg && d1163.Reg == d12.Reg {
			ctx.TransferReg(d12.Reg)
			d12.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d1163)
		var d1164 scm.JITValueDesc
		if d1163.Loc == scm.LocImm {
			d1164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1163.Imm.Int() / 2)}
		} else {
			r104 := ctx.AllocRegExcept(d1163.Reg)
			ctx.EmitMovRegReg(r104, d1163.Reg)
			ctx.EmitShrRegImm8(r104, 1)
			d1164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r104}
			ctx.BindReg(r104, &d1164)
		}
		if d1164.Loc == scm.LocImm {
			d1164 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1164.Type, Imm: scm.NewInt(int64(uint64(d1164.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d1164.Reg, 32)
			ctx.EmitShrRegImm8(d1164.Reg, 32)
		}
		if d1164.Loc == scm.LocReg && d1163.Loc == scm.LocReg && d1164.Reg == d1163.Reg {
			ctx.TransferReg(d1163.Reg)
			d1163.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1163)
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
			ctx.SyncDesc(&d1164)
			if d1164.Loc == scm.LocReg {
				ctx.ProtectReg(d1164.Reg)
			} else if d1164.Loc == scm.LocRegPair {
				ctx.ProtectReg(d1164.Reg)
				ctx.ProtectReg(d1164.Reg2)
			}
			d1165 = d1164
			if d1165.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d1165)
			d1166 = d1165
			if d1166.Loc == scm.LocImm {
				d1166 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1166.Type, Imm: scm.NewInt(int64(uint64(d1166.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d1166.Reg, 32)
				ctx.EmitShrRegImm8(d1166.Reg, 32)
			}
			if phiHomeOK2 {
				ctx.EmitMovToReg(r0, d1166)
			} else {
				ctx.EmitStoreToStack(d1166, int32(bbs[1].PhiBase)+int32(0))
			}
			d1167 = d12
			if d1167.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d1167)
			d1168 = d1167
			if d1168.Loc == scm.LocImm {
				d1168 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1168.Type, Imm: scm.NewInt(int64(uint64(d1168.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d1168.Reg, 32)
				ctx.EmitShrRegImm8(d1168.Reg, 32)
			}
			if phiHomeOK3 {
				ctx.EmitMovToReg(r1, d1168)
			} else {
				ctx.EmitStoreToStack(d1168, int32(bbs[1].PhiBase)+int32(16))
			}
			d1169 = d13
			if d1169.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d1169)
			d1170 = d1169
			if d1170.Loc == scm.LocImm {
				d1170 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1170.Type, Imm: scm.NewInt(int64(uint64(d1170.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d1170.Reg, 32)
				ctx.EmitShrRegImm8(d1170.Reg, 32)
			}
			if phiHomeOK4 {
				ctx.EmitMovToReg(r2, d1170)
			} else {
				ctx.EmitStoreToStack(d1170, int32(bbs[1].PhiBase)+int32(32))
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
			if d1164.Loc == scm.LocReg {
				ctx.UnprotectReg(d1164.Reg)
			} else if d1164.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d1164.Reg)
				ctx.UnprotectReg(d1164.Reg2)
			}
		}
		ps1171 := scm.PhiState{General: ps.General}
		ps1171.OverlayValues = make([]scm.JITValueDesc, 1171)
		ps1171.OverlayValues[5] = d5
		ps1171.OverlayValues[6] = d6
		ps1171.OverlayValues[7] = d7
		ps1171.OverlayValues[8] = d8
		ps1171.OverlayValues[9] = d9
		ps1171.OverlayValues[10] = d10
		ps1171.OverlayValues[11] = d11
		ps1171.OverlayValues[12] = d12
		ps1171.OverlayValues[13] = d13
		ps1171.OverlayValues[14] = d14
		ps1171.OverlayValues[15] = d15
		ps1171.OverlayValues[16] = d16
		ps1171.OverlayValues[17] = d17
		ps1171.OverlayValues[18] = d18
		ps1171.OverlayValues[19] = d19
		ps1171.OverlayValues[21] = d21
		ps1171.OverlayValues[22] = d22
		ps1171.OverlayValues[23] = d23
		ps1171.OverlayValues[24] = d24
		ps1171.OverlayValues[25] = d25
		ps1171.OverlayValues[26] = d26
		ps1171.OverlayValues[27] = d27
		ps1171.OverlayValues[28] = d28
		ps1171.OverlayValues[29] = d29
		ps1171.OverlayValues[30] = d30
		ps1171.OverlayValues[31] = d31
		ps1171.OverlayValues[32] = d32
		ps1171.OverlayValues[33] = d33
		ps1171.OverlayValues[34] = d34
		ps1171.OverlayValues[35] = d35
		ps1171.OverlayValues[36] = d36
		ps1171.OverlayValues[37] = d37
		ps1171.OverlayValues[38] = d38
		ps1171.OverlayValues[39] = d39
		ps1171.OverlayValues[40] = d40
		ps1171.OverlayValues[41] = d41
		ps1171.OverlayValues[42] = d42
		ps1171.OverlayValues[43] = d43
		ps1171.OverlayValues[44] = d44
		ps1171.OverlayValues[45] = d45
		ps1171.OverlayValues[46] = d46
		ps1171.OverlayValues[47] = d47
		ps1171.OverlayValues[48] = d48
		ps1171.OverlayValues[49] = d49
		ps1171.OverlayValues[50] = d50
		ps1171.OverlayValues[51] = d51
		ps1171.OverlayValues[54] = d54
		ps1171.OverlayValues[55] = d55
		ps1171.OverlayValues[56] = d56
		ps1171.OverlayValues[159] = d159
		ps1171.OverlayValues[160] = d160
		ps1171.OverlayValues[161] = d161
		ps1171.OverlayValues[162] = d162
		ps1171.OverlayValues[163] = d163
		ps1171.OverlayValues[164] = d164
		ps1171.OverlayValues[165] = d165
		ps1171.OverlayValues[166] = d166
		ps1171.OverlayValues[167] = d167
		ps1171.OverlayValues[168] = d168
		ps1171.OverlayValues[169] = d169
		ps1171.OverlayValues[170] = d170
		ps1171.OverlayValues[171] = d171
		ps1171.OverlayValues[172] = d172
		ps1171.OverlayValues[173] = d173
		ps1171.OverlayValues[174] = d174
		ps1171.OverlayValues[175] = d175
		ps1171.OverlayValues[176] = d176
		ps1171.OverlayValues[177] = d177
		ps1171.OverlayValues[178] = d178
		ps1171.OverlayValues[179] = d179
		ps1171.OverlayValues[180] = d180
		ps1171.OverlayValues[181] = d181
		ps1171.OverlayValues[182] = d182
		ps1171.OverlayValues[183] = d183
		ps1171.OverlayValues[184] = d184
		ps1171.OverlayValues[187] = d187
		ps1171.OverlayValues[344] = d344
		ps1171.OverlayValues[345] = d345
		ps1171.OverlayValues[346] = d346
		ps1171.OverlayValues[347] = d347
		ps1171.OverlayValues[349] = d349
		ps1171.OverlayValues[350] = d350
		ps1171.OverlayValues[351] = d351
		ps1171.OverlayValues[352] = d352
		ps1171.OverlayValues[353] = d353
		ps1171.OverlayValues[354] = d354
		ps1171.OverlayValues[355] = d355
		ps1171.OverlayValues[356] = d356
		ps1171.OverlayValues[358] = d358
		ps1171.OverlayValues[360] = d360
		ps1171.OverlayValues[361] = d361
		ps1171.OverlayValues[362] = d362
		ps1171.OverlayValues[456] = d456
		ps1171.OverlayValues[457] = d457
		ps1171.OverlayValues[460] = d460
		ps1171.OverlayValues[557] = d557
		ps1171.OverlayValues[558] = d558
		ps1171.OverlayValues[559] = d559
		ps1171.OverlayValues[560] = d560
		ps1171.OverlayValues[561] = d561
		ps1171.OverlayValues[563] = d563
		ps1171.OverlayValues[564] = d564
		ps1171.OverlayValues[565] = d565
		ps1171.OverlayValues[566] = d566
		ps1171.OverlayValues[567] = d567
		ps1171.OverlayValues[568] = d568
		ps1171.OverlayValues[569] = d569
		ps1171.OverlayValues[570] = d570
		ps1171.OverlayValues[571] = d571
		ps1171.OverlayValues[572] = d572
		ps1171.OverlayValues[573] = d573
		ps1171.OverlayValues[574] = d574
		ps1171.OverlayValues[575] = d575
		ps1171.OverlayValues[576] = d576
		ps1171.OverlayValues[577] = d577
		ps1171.OverlayValues[578] = d578
		ps1171.OverlayValues[579] = d579
		ps1171.OverlayValues[580] = d580
		ps1171.OverlayValues[581] = d581
		ps1171.OverlayValues[582] = d582
		ps1171.OverlayValues[583] = d583
		ps1171.OverlayValues[584] = d584
		ps1171.OverlayValues[585] = d585
		ps1171.OverlayValues[586] = d586
		ps1171.OverlayValues[587] = d587
		ps1171.OverlayValues[588] = d588
		ps1171.OverlayValues[589] = d589
		ps1171.OverlayValues[850] = d850
		ps1171.OverlayValues[851] = d851
		ps1171.OverlayValues[852] = d852
		ps1171.OverlayValues[854] = d854
		ps1171.OverlayValues[855] = d855
		ps1171.OverlayValues[856] = d856
		ps1171.OverlayValues[857] = d857
		ps1171.OverlayValues[858] = d858
		ps1171.OverlayValues[859] = d859
		ps1171.OverlayValues[860] = d860
		ps1171.OverlayValues[862] = d862
		ps1171.OverlayValues[864] = d864
		ps1171.OverlayValues[865] = d865
		ps1171.OverlayValues[1007] = d1007
		ps1171.OverlayValues[1008] = d1008
		ps1171.OverlayValues[1011] = d1011
		ps1171.OverlayValues[1156] = d1156
		ps1171.OverlayValues[1157] = d1157
		ps1171.OverlayValues[1158] = d1158
		ps1171.OverlayValues[1159] = d1159
		ps1171.OverlayValues[1161] = d1161
		ps1171.OverlayValues[1162] = d1162
		ps1171.OverlayValues[1163] = d1163
		ps1171.OverlayValues[1164] = d1164
		ps1171.OverlayValues[1165] = d1165
		ps1171.OverlayValues[1166] = d1166
		ps1171.OverlayValues[1167] = d1167
		ps1171.OverlayValues[1168] = d1168
		ps1171.OverlayValues[1169] = d1169
		ps1171.OverlayValues[1170] = d1170
		ps1171.PhiValues = make([]scm.JITValueDesc, 3)
		d1172 = d1164
		ps1171.PhiValues[0] = d1172
		d1173 = d12
		ps1171.PhiValues[1] = d1173
		d1174 = d13
		ps1171.PhiValues[2] = d1174
		if ps1171.General && bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		return bbs[1].RenderPS(ps1171)
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
		if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
			d187 = ps.OverlayValues[187]
		}
		if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != scm.LocNone {
			d344 = ps.OverlayValues[344]
		}
		if len(ps.OverlayValues) > 345 && ps.OverlayValues[345].Loc != scm.LocNone {
			d345 = ps.OverlayValues[345]
		}
		if len(ps.OverlayValues) > 346 && ps.OverlayValues[346].Loc != scm.LocNone {
			d346 = ps.OverlayValues[346]
		}
		if len(ps.OverlayValues) > 347 && ps.OverlayValues[347].Loc != scm.LocNone {
			d347 = ps.OverlayValues[347]
		}
		if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != scm.LocNone {
			d349 = ps.OverlayValues[349]
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
		if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != scm.LocNone {
			d355 = ps.OverlayValues[355]
		}
		if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != scm.LocNone {
			d356 = ps.OverlayValues[356]
		}
		if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != scm.LocNone {
			d358 = ps.OverlayValues[358]
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
		if len(ps.OverlayValues) > 456 && ps.OverlayValues[456].Loc != scm.LocNone {
			d456 = ps.OverlayValues[456]
		}
		if len(ps.OverlayValues) > 457 && ps.OverlayValues[457].Loc != scm.LocNone {
			d457 = ps.OverlayValues[457]
		}
		if len(ps.OverlayValues) > 460 && ps.OverlayValues[460].Loc != scm.LocNone {
			d460 = ps.OverlayValues[460]
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
		if len(ps.OverlayValues) > 561 && ps.OverlayValues[561].Loc != scm.LocNone {
			d561 = ps.OverlayValues[561]
		}
		if len(ps.OverlayValues) > 563 && ps.OverlayValues[563].Loc != scm.LocNone {
			d563 = ps.OverlayValues[563]
		}
		if len(ps.OverlayValues) > 564 && ps.OverlayValues[564].Loc != scm.LocNone {
			d564 = ps.OverlayValues[564]
		}
		if len(ps.OverlayValues) > 565 && ps.OverlayValues[565].Loc != scm.LocNone {
			d565 = ps.OverlayValues[565]
		}
		if len(ps.OverlayValues) > 566 && ps.OverlayValues[566].Loc != scm.LocNone {
			d566 = ps.OverlayValues[566]
		}
		if len(ps.OverlayValues) > 567 && ps.OverlayValues[567].Loc != scm.LocNone {
			d567 = ps.OverlayValues[567]
		}
		if len(ps.OverlayValues) > 568 && ps.OverlayValues[568].Loc != scm.LocNone {
			d568 = ps.OverlayValues[568]
		}
		if len(ps.OverlayValues) > 569 && ps.OverlayValues[569].Loc != scm.LocNone {
			d569 = ps.OverlayValues[569]
		}
		if len(ps.OverlayValues) > 570 && ps.OverlayValues[570].Loc != scm.LocNone {
			d570 = ps.OverlayValues[570]
		}
		if len(ps.OverlayValues) > 571 && ps.OverlayValues[571].Loc != scm.LocNone {
			d571 = ps.OverlayValues[571]
		}
		if len(ps.OverlayValues) > 572 && ps.OverlayValues[572].Loc != scm.LocNone {
			d572 = ps.OverlayValues[572]
		}
		if len(ps.OverlayValues) > 573 && ps.OverlayValues[573].Loc != scm.LocNone {
			d573 = ps.OverlayValues[573]
		}
		if len(ps.OverlayValues) > 574 && ps.OverlayValues[574].Loc != scm.LocNone {
			d574 = ps.OverlayValues[574]
		}
		if len(ps.OverlayValues) > 575 && ps.OverlayValues[575].Loc != scm.LocNone {
			d575 = ps.OverlayValues[575]
		}
		if len(ps.OverlayValues) > 576 && ps.OverlayValues[576].Loc != scm.LocNone {
			d576 = ps.OverlayValues[576]
		}
		if len(ps.OverlayValues) > 577 && ps.OverlayValues[577].Loc != scm.LocNone {
			d577 = ps.OverlayValues[577]
		}
		if len(ps.OverlayValues) > 578 && ps.OverlayValues[578].Loc != scm.LocNone {
			d578 = ps.OverlayValues[578]
		}
		if len(ps.OverlayValues) > 579 && ps.OverlayValues[579].Loc != scm.LocNone {
			d579 = ps.OverlayValues[579]
		}
		if len(ps.OverlayValues) > 580 && ps.OverlayValues[580].Loc != scm.LocNone {
			d580 = ps.OverlayValues[580]
		}
		if len(ps.OverlayValues) > 581 && ps.OverlayValues[581].Loc != scm.LocNone {
			d581 = ps.OverlayValues[581]
		}
		if len(ps.OverlayValues) > 582 && ps.OverlayValues[582].Loc != scm.LocNone {
			d582 = ps.OverlayValues[582]
		}
		if len(ps.OverlayValues) > 583 && ps.OverlayValues[583].Loc != scm.LocNone {
			d583 = ps.OverlayValues[583]
		}
		if len(ps.OverlayValues) > 584 && ps.OverlayValues[584].Loc != scm.LocNone {
			d584 = ps.OverlayValues[584]
		}
		if len(ps.OverlayValues) > 585 && ps.OverlayValues[585].Loc != scm.LocNone {
			d585 = ps.OverlayValues[585]
		}
		if len(ps.OverlayValues) > 586 && ps.OverlayValues[586].Loc != scm.LocNone {
			d586 = ps.OverlayValues[586]
		}
		if len(ps.OverlayValues) > 587 && ps.OverlayValues[587].Loc != scm.LocNone {
			d587 = ps.OverlayValues[587]
		}
		if len(ps.OverlayValues) > 588 && ps.OverlayValues[588].Loc != scm.LocNone {
			d588 = ps.OverlayValues[588]
		}
		if len(ps.OverlayValues) > 589 && ps.OverlayValues[589].Loc != scm.LocNone {
			d589 = ps.OverlayValues[589]
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
		if len(ps.OverlayValues) > 862 && ps.OverlayValues[862].Loc != scm.LocNone {
			d862 = ps.OverlayValues[862]
		}
		if len(ps.OverlayValues) > 864 && ps.OverlayValues[864].Loc != scm.LocNone {
			d864 = ps.OverlayValues[864]
		}
		if len(ps.OverlayValues) > 865 && ps.OverlayValues[865].Loc != scm.LocNone {
			d865 = ps.OverlayValues[865]
		}
		if len(ps.OverlayValues) > 1007 && ps.OverlayValues[1007].Loc != scm.LocNone {
			d1007 = ps.OverlayValues[1007]
		}
		if len(ps.OverlayValues) > 1008 && ps.OverlayValues[1008].Loc != scm.LocNone {
			d1008 = ps.OverlayValues[1008]
		}
		if len(ps.OverlayValues) > 1011 && ps.OverlayValues[1011].Loc != scm.LocNone {
			d1011 = ps.OverlayValues[1011]
		}
		if len(ps.OverlayValues) > 1156 && ps.OverlayValues[1156].Loc != scm.LocNone {
			d1156 = ps.OverlayValues[1156]
		}
		if len(ps.OverlayValues) > 1157 && ps.OverlayValues[1157].Loc != scm.LocNone {
			d1157 = ps.OverlayValues[1157]
		}
		if len(ps.OverlayValues) > 1158 && ps.OverlayValues[1158].Loc != scm.LocNone {
			d1158 = ps.OverlayValues[1158]
		}
		if len(ps.OverlayValues) > 1159 && ps.OverlayValues[1159].Loc != scm.LocNone {
			d1159 = ps.OverlayValues[1159]
		}
		if len(ps.OverlayValues) > 1161 && ps.OverlayValues[1161].Loc != scm.LocNone {
			d1161 = ps.OverlayValues[1161]
		}
		if len(ps.OverlayValues) > 1162 && ps.OverlayValues[1162].Loc != scm.LocNone {
			d1162 = ps.OverlayValues[1162]
		}
		if len(ps.OverlayValues) > 1163 && ps.OverlayValues[1163].Loc != scm.LocNone {
			d1163 = ps.OverlayValues[1163]
		}
		if len(ps.OverlayValues) > 1164 && ps.OverlayValues[1164].Loc != scm.LocNone {
			d1164 = ps.OverlayValues[1164]
		}
		if len(ps.OverlayValues) > 1165 && ps.OverlayValues[1165].Loc != scm.LocNone {
			d1165 = ps.OverlayValues[1165]
		}
		if len(ps.OverlayValues) > 1166 && ps.OverlayValues[1166].Loc != scm.LocNone {
			d1166 = ps.OverlayValues[1166]
		}
		if len(ps.OverlayValues) > 1167 && ps.OverlayValues[1167].Loc != scm.LocNone {
			d1167 = ps.OverlayValues[1167]
		}
		if len(ps.OverlayValues) > 1168 && ps.OverlayValues[1168].Loc != scm.LocNone {
			d1168 = ps.OverlayValues[1168]
		}
		if len(ps.OverlayValues) > 1169 && ps.OverlayValues[1169].Loc != scm.LocNone {
			d1169 = ps.OverlayValues[1169]
		}
		if len(ps.OverlayValues) > 1170 && ps.OverlayValues[1170].Loc != scm.LocNone {
			d1170 = ps.OverlayValues[1170]
		}
		if len(ps.OverlayValues) > 1172 && ps.OverlayValues[1172].Loc != scm.LocNone {
			d1172 = ps.OverlayValues[1172]
		}
		if len(ps.OverlayValues) > 1173 && ps.OverlayValues[1173].Loc != scm.LocNone {
			d1173 = ps.OverlayValues[1173]
		}
		if len(ps.OverlayValues) > 1174 && ps.OverlayValues[1174].Loc != scm.LocNone {
			d1174 = ps.OverlayValues[1174]
		}
		ctx.ReclaimUntrackedRegs()
		d1175 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d1176 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r3, Reg2: r4}
		ctx.BindReg(r3, &d1176)
		ctx.BindReg(r4, &d1176)
		ctx.EnsureDesc(&d1175)
		if d1175.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d1175, &d1176)
		} else {
			switch d1175.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d1176, d1175)
			case scm.TagInt:
				ctx.EmitMakeInt(d1176, d1175)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d1176, d1175)
			case scm.TagNil:
				ctx.EmitMakeNil(d1176)
			default:
				ctx.EmitMovPairToResult(&d1175, &d1176)
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
		if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
			d187 = ps.OverlayValues[187]
		}
		if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != scm.LocNone {
			d344 = ps.OverlayValues[344]
		}
		if len(ps.OverlayValues) > 345 && ps.OverlayValues[345].Loc != scm.LocNone {
			d345 = ps.OverlayValues[345]
		}
		if len(ps.OverlayValues) > 346 && ps.OverlayValues[346].Loc != scm.LocNone {
			d346 = ps.OverlayValues[346]
		}
		if len(ps.OverlayValues) > 347 && ps.OverlayValues[347].Loc != scm.LocNone {
			d347 = ps.OverlayValues[347]
		}
		if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != scm.LocNone {
			d349 = ps.OverlayValues[349]
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
		if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != scm.LocNone {
			d355 = ps.OverlayValues[355]
		}
		if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != scm.LocNone {
			d356 = ps.OverlayValues[356]
		}
		if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != scm.LocNone {
			d358 = ps.OverlayValues[358]
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
		if len(ps.OverlayValues) > 456 && ps.OverlayValues[456].Loc != scm.LocNone {
			d456 = ps.OverlayValues[456]
		}
		if len(ps.OverlayValues) > 457 && ps.OverlayValues[457].Loc != scm.LocNone {
			d457 = ps.OverlayValues[457]
		}
		if len(ps.OverlayValues) > 460 && ps.OverlayValues[460].Loc != scm.LocNone {
			d460 = ps.OverlayValues[460]
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
		if len(ps.OverlayValues) > 561 && ps.OverlayValues[561].Loc != scm.LocNone {
			d561 = ps.OverlayValues[561]
		}
		if len(ps.OverlayValues) > 563 && ps.OverlayValues[563].Loc != scm.LocNone {
			d563 = ps.OverlayValues[563]
		}
		if len(ps.OverlayValues) > 564 && ps.OverlayValues[564].Loc != scm.LocNone {
			d564 = ps.OverlayValues[564]
		}
		if len(ps.OverlayValues) > 565 && ps.OverlayValues[565].Loc != scm.LocNone {
			d565 = ps.OverlayValues[565]
		}
		if len(ps.OverlayValues) > 566 && ps.OverlayValues[566].Loc != scm.LocNone {
			d566 = ps.OverlayValues[566]
		}
		if len(ps.OverlayValues) > 567 && ps.OverlayValues[567].Loc != scm.LocNone {
			d567 = ps.OverlayValues[567]
		}
		if len(ps.OverlayValues) > 568 && ps.OverlayValues[568].Loc != scm.LocNone {
			d568 = ps.OverlayValues[568]
		}
		if len(ps.OverlayValues) > 569 && ps.OverlayValues[569].Loc != scm.LocNone {
			d569 = ps.OverlayValues[569]
		}
		if len(ps.OverlayValues) > 570 && ps.OverlayValues[570].Loc != scm.LocNone {
			d570 = ps.OverlayValues[570]
		}
		if len(ps.OverlayValues) > 571 && ps.OverlayValues[571].Loc != scm.LocNone {
			d571 = ps.OverlayValues[571]
		}
		if len(ps.OverlayValues) > 572 && ps.OverlayValues[572].Loc != scm.LocNone {
			d572 = ps.OverlayValues[572]
		}
		if len(ps.OverlayValues) > 573 && ps.OverlayValues[573].Loc != scm.LocNone {
			d573 = ps.OverlayValues[573]
		}
		if len(ps.OverlayValues) > 574 && ps.OverlayValues[574].Loc != scm.LocNone {
			d574 = ps.OverlayValues[574]
		}
		if len(ps.OverlayValues) > 575 && ps.OverlayValues[575].Loc != scm.LocNone {
			d575 = ps.OverlayValues[575]
		}
		if len(ps.OverlayValues) > 576 && ps.OverlayValues[576].Loc != scm.LocNone {
			d576 = ps.OverlayValues[576]
		}
		if len(ps.OverlayValues) > 577 && ps.OverlayValues[577].Loc != scm.LocNone {
			d577 = ps.OverlayValues[577]
		}
		if len(ps.OverlayValues) > 578 && ps.OverlayValues[578].Loc != scm.LocNone {
			d578 = ps.OverlayValues[578]
		}
		if len(ps.OverlayValues) > 579 && ps.OverlayValues[579].Loc != scm.LocNone {
			d579 = ps.OverlayValues[579]
		}
		if len(ps.OverlayValues) > 580 && ps.OverlayValues[580].Loc != scm.LocNone {
			d580 = ps.OverlayValues[580]
		}
		if len(ps.OverlayValues) > 581 && ps.OverlayValues[581].Loc != scm.LocNone {
			d581 = ps.OverlayValues[581]
		}
		if len(ps.OverlayValues) > 582 && ps.OverlayValues[582].Loc != scm.LocNone {
			d582 = ps.OverlayValues[582]
		}
		if len(ps.OverlayValues) > 583 && ps.OverlayValues[583].Loc != scm.LocNone {
			d583 = ps.OverlayValues[583]
		}
		if len(ps.OverlayValues) > 584 && ps.OverlayValues[584].Loc != scm.LocNone {
			d584 = ps.OverlayValues[584]
		}
		if len(ps.OverlayValues) > 585 && ps.OverlayValues[585].Loc != scm.LocNone {
			d585 = ps.OverlayValues[585]
		}
		if len(ps.OverlayValues) > 586 && ps.OverlayValues[586].Loc != scm.LocNone {
			d586 = ps.OverlayValues[586]
		}
		if len(ps.OverlayValues) > 587 && ps.OverlayValues[587].Loc != scm.LocNone {
			d587 = ps.OverlayValues[587]
		}
		if len(ps.OverlayValues) > 588 && ps.OverlayValues[588].Loc != scm.LocNone {
			d588 = ps.OverlayValues[588]
		}
		if len(ps.OverlayValues) > 589 && ps.OverlayValues[589].Loc != scm.LocNone {
			d589 = ps.OverlayValues[589]
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
		if len(ps.OverlayValues) > 862 && ps.OverlayValues[862].Loc != scm.LocNone {
			d862 = ps.OverlayValues[862]
		}
		if len(ps.OverlayValues) > 864 && ps.OverlayValues[864].Loc != scm.LocNone {
			d864 = ps.OverlayValues[864]
		}
		if len(ps.OverlayValues) > 865 && ps.OverlayValues[865].Loc != scm.LocNone {
			d865 = ps.OverlayValues[865]
		}
		if len(ps.OverlayValues) > 1007 && ps.OverlayValues[1007].Loc != scm.LocNone {
			d1007 = ps.OverlayValues[1007]
		}
		if len(ps.OverlayValues) > 1008 && ps.OverlayValues[1008].Loc != scm.LocNone {
			d1008 = ps.OverlayValues[1008]
		}
		if len(ps.OverlayValues) > 1011 && ps.OverlayValues[1011].Loc != scm.LocNone {
			d1011 = ps.OverlayValues[1011]
		}
		if len(ps.OverlayValues) > 1156 && ps.OverlayValues[1156].Loc != scm.LocNone {
			d1156 = ps.OverlayValues[1156]
		}
		if len(ps.OverlayValues) > 1157 && ps.OverlayValues[1157].Loc != scm.LocNone {
			d1157 = ps.OverlayValues[1157]
		}
		if len(ps.OverlayValues) > 1158 && ps.OverlayValues[1158].Loc != scm.LocNone {
			d1158 = ps.OverlayValues[1158]
		}
		if len(ps.OverlayValues) > 1159 && ps.OverlayValues[1159].Loc != scm.LocNone {
			d1159 = ps.OverlayValues[1159]
		}
		if len(ps.OverlayValues) > 1161 && ps.OverlayValues[1161].Loc != scm.LocNone {
			d1161 = ps.OverlayValues[1161]
		}
		if len(ps.OverlayValues) > 1162 && ps.OverlayValues[1162].Loc != scm.LocNone {
			d1162 = ps.OverlayValues[1162]
		}
		if len(ps.OverlayValues) > 1163 && ps.OverlayValues[1163].Loc != scm.LocNone {
			d1163 = ps.OverlayValues[1163]
		}
		if len(ps.OverlayValues) > 1164 && ps.OverlayValues[1164].Loc != scm.LocNone {
			d1164 = ps.OverlayValues[1164]
		}
		if len(ps.OverlayValues) > 1165 && ps.OverlayValues[1165].Loc != scm.LocNone {
			d1165 = ps.OverlayValues[1165]
		}
		if len(ps.OverlayValues) > 1166 && ps.OverlayValues[1166].Loc != scm.LocNone {
			d1166 = ps.OverlayValues[1166]
		}
		if len(ps.OverlayValues) > 1167 && ps.OverlayValues[1167].Loc != scm.LocNone {
			d1167 = ps.OverlayValues[1167]
		}
		if len(ps.OverlayValues) > 1168 && ps.OverlayValues[1168].Loc != scm.LocNone {
			d1168 = ps.OverlayValues[1168]
		}
		if len(ps.OverlayValues) > 1169 && ps.OverlayValues[1169].Loc != scm.LocNone {
			d1169 = ps.OverlayValues[1169]
		}
		if len(ps.OverlayValues) > 1170 && ps.OverlayValues[1170].Loc != scm.LocNone {
			d1170 = ps.OverlayValues[1170]
		}
		if len(ps.OverlayValues) > 1172 && ps.OverlayValues[1172].Loc != scm.LocNone {
			d1172 = ps.OverlayValues[1172]
		}
		if len(ps.OverlayValues) > 1173 && ps.OverlayValues[1173].Loc != scm.LocNone {
			d1173 = ps.OverlayValues[1173]
		}
		if len(ps.OverlayValues) > 1174 && ps.OverlayValues[1174].Loc != scm.LocNone {
			d1174 = ps.OverlayValues[1174]
		}
		if len(ps.OverlayValues) > 1175 && ps.OverlayValues[1175].Loc != scm.LocNone {
			d1175 = ps.OverlayValues[1175]
		}
		if len(ps.OverlayValues) > 1176 && ps.OverlayValues[1176].Loc != scm.LocNone {
			d1176 = ps.OverlayValues[1176]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d8)
		d1177 = d8
		_ = d1177
		ctx.StabilizeDescForControlFlow(&d8)
		bbpos_4_0 := int32(-1)
		_ = bbpos_4_0
		lbl20 := ctx.ReserveLabel()
		_ = lbl20
		bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl20)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d1178 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d1178 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 48)
			r105 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r105, thisptr.Reg, off)
			d1178 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r105}
			ctx.BindReg(r105, &d1178)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1178)
		ctx.EnsureDesc(&d1178)
		var d1179 scm.JITValueDesc
		if d1178.Loc == scm.LocImm {
			d1179 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d1178.Imm.Int()))))}
		} else {
			r106 := ctx.AllocReg()
			ctx.EmitMovRegReg(r106, d1178.Reg)
			ctx.EmitShlRegImm8(r106, 56)
			ctx.EmitShrRegImm8(r106, 56)
			d1179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r106}
			ctx.BindReg(r106, &d1179)
		}
		ctx.FreeDesc(&d1178)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1177)
		ctx.EnsureDesc(&d1177)
		var d1180 scm.JITValueDesc
		if d1177.Loc == scm.LocImm {
			d1180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d1177.Imm.Int()))))}
		} else {
			r107 := ctx.AllocReg()
			ctx.EmitMovRegReg(r107, d1177.Reg)
			ctx.EmitShlRegImm8(r107, 32)
			ctx.EmitShrRegImm8(r107, 32)
			d1180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r107}
			ctx.BindReg(r107, &d1180)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1180)
		ctx.EnsureDesc(&d1179)
		ctx.EnsureDescsTogether(&d1180, &d1179)
		var d1181 scm.JITValueDesc
		if d1180.Loc == scm.LocImm && d1179.Loc == scm.LocImm {
			d1181 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1180.Imm.Int() * d1179.Imm.Int())}
		} else if d1180.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1179.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1180.Imm.Int()))
			ctx.EmitImulInt64(scratch, d1179.Reg)
			d1181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1181)
		} else if d1179.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1180.Reg)
			ctx.EmitMovRegReg(scratch, d1180.Reg)
			if d1179.Imm.Int() >= -2147483648 && d1179.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d1179.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1179.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d1181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1181)
		} else {
			r108 := ctx.AllocRegExcept(d1180.Reg, d1179.Reg)
			ctx.EmitMovRegReg(r108, d1180.Reg)
			ctx.EmitImulInt64(r108, d1179.Reg)
			d1181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
			ctx.BindReg(r108, &d1181)
		}
		if d1181.Loc == scm.LocReg && d1180.Loc == scm.LocReg && d1181.Reg == d1180.Reg {
			ctx.TransferReg(d1180.Reg)
			d1180.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1180)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1181)
		var d1182 scm.JITValueDesc
		if d1181.Loc == scm.LocImm {
			d1182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1181.Imm.Int() / 64)}
		} else {
			r109 := ctx.AllocRegExcept(d1181.Reg)
			ctx.EmitMovRegReg(r109, d1181.Reg)
			ctx.EmitShrRegImm8(r109, 6)
			d1182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r109}
			ctx.BindReg(r109, &d1182)
		}
		if d1182.Loc == scm.LocReg && d1181.Loc == scm.LocReg && d1182.Reg == d1181.Reg {
			ctx.TransferReg(d1181.Reg)
			d1181.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1181)
		var d1183 scm.JITValueDesc
		if d1181.Loc == scm.LocImm {
			d1183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1181.Imm.Int() % 64)}
		} else {
			r110 := ctx.AllocRegExcept(d1181.Reg)
			ctx.EmitMovRegReg(r110, d1181.Reg)
			ctx.EmitAndRegImm32(r110, 63)
			d1183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
			ctx.BindReg(r110, &d1183)
		}
		if d1183.Loc == scm.LocReg && d1181.Loc == scm.LocReg && d1183.Reg == d1181.Reg {
			ctx.TransferReg(d1181.Reg)
			d1181.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1181)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d1184 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d1184 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r111 := ctx.AllocReg()
			r112 := ctx.AllocRegExcept(r111)
			r113 := ctx.AllocRegExcept(r111, r112)
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
			ctx.EmitMovRegMem(r111, thisptr.Reg, off)
			ctx.EmitMovRegMem(r112, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r113, thisptr.Reg, off+16)
			d1184 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r111, Reg2: r112, Reg3: r113}
			ctx.BindReg(r111, &d1184)
			ctx.BindReg(r112, &d1184)
			ctx.BindReg(r113, &d1184)
			ctx.BindReg(r111, &d1184)
			ctx.BindReg(r112, &d1184)
			ctx.BindReg(r113, &d1184)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1182)
		ctx.ReclaimUntrackedRegs()
		d1185 = ctx.EmitLoadScalarSliceElement(&d1184, &d1182, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1185)
		ctx.EnsureDesc(&d1183)
		ctx.EnsureDescsTogether(&d1185, &d1183)
		var d1186 scm.JITValueDesc
		if d1185.Loc == scm.LocImm && d1183.Loc == scm.LocImm {
			d1186 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1185.Imm.Int()) << uint64(d1183.Imm.Int())))}
		} else if d1183.Loc == scm.LocImm {
			r114 := ctx.AllocRegExcept(d1185.Reg)
			ctx.EmitMovRegReg(r114, d1185.Reg)
			ctx.EmitShlRegImm8(r114, uint8(d1183.Imm.Int()))
			d1186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r114}
			ctx.BindReg(r114, &d1186)
		} else {
			{
				shiftSrc := d1185.Reg
				r115 := ctx.AllocRegExcept(d1185.Reg, d1183.Reg)
				ctx.EmitMovRegReg(r115, d1185.Reg)
				shiftSrc = r115
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d1183.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d1183.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d1183.Reg)
				}
				ctx.EmitShlRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d1186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d1186)
			}
		}
		if d1186.Loc == scm.LocReg && d1185.Loc == scm.LocReg && d1186.Reg == d1185.Reg {
			ctx.TransferReg(d1185.Reg)
			d1185.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1185)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1182)
		ctx.EnsureDesc(&d1182)
		var d1187 scm.JITValueDesc
		if d1182.Loc == scm.LocImm {
			d1187 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1182.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d1182.Reg)
			ctx.EmitMovRegReg(scratch, d1182.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d1187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1187)
		}
		if d1187.Loc == scm.LocReg && d1182.Loc == scm.LocReg && d1187.Reg == d1182.Reg {
			ctx.TransferReg(d1182.Reg)
			d1182.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1182)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1187)
		ctx.ReclaimUntrackedRegs()
		d1188 = ctx.EmitLoadScalarSliceElement(&d1184, &d1187, 8, scm.TagInt)
		ctx.FreeDesc(&d1187)
		ctx.ReclaimUntrackedRegs()
		d1189 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d1183)
		ctx.EnsureDescsTogether(&d1189, &d1183)
		var d1190 scm.JITValueDesc
		if d1189.Loc == scm.LocImm && d1183.Loc == scm.LocImm {
			d1190 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1189.Imm.Int() - d1183.Imm.Int())}
		} else if d1183.Loc == scm.LocImm && d1183.Imm.Int() == 0 {
			r116 := ctx.AllocRegExcept(d1189.Reg)
			ctx.EmitMovRegReg(r116, d1189.Reg)
			d1190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r116}
			ctx.BindReg(r116, &d1190)
		} else if d1189.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1183.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1189.Imm.Int()))
			ctx.EmitSubInt64(scratch, d1183.Reg)
			d1190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1190)
		} else if d1183.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1189.Reg)
			ctx.EmitMovRegReg(scratch, d1189.Reg)
			if d1183.Imm.Int() >= -2147483648 && d1183.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d1183.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1183.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d1190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1190)
		} else {
			r117 := ctx.AllocRegExcept(d1189.Reg, d1183.Reg)
			ctx.EmitMovRegReg(r117, d1189.Reg)
			ctx.EmitSubInt64(r117, d1183.Reg)
			d1190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
			ctx.BindReg(r117, &d1190)
		}
		if d1190.Loc == scm.LocReg && d1189.Loc == scm.LocReg && d1190.Reg == d1189.Reg {
			ctx.TransferReg(d1189.Reg)
			d1189.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1183)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1188)
		ctx.EnsureDesc(&d1190)
		ctx.EnsureDescsTogether(&d1188, &d1190)
		var d1191 scm.JITValueDesc
		if d1188.Loc == scm.LocImm && d1190.Loc == scm.LocImm {
			d1191 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1188.Imm.Int()) >> uint64(d1190.Imm.Int())))}
		} else if d1190.Loc == scm.LocImm {
			r118 := ctx.AllocRegExcept(d1188.Reg)
			ctx.EmitMovRegReg(r118, d1188.Reg)
			ctx.EmitShrRegImm8(r118, uint8(d1190.Imm.Int()))
			d1191 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r118}
			ctx.BindReg(r118, &d1191)
		} else {
			{
				shiftSrc := d1188.Reg
				r119 := ctx.AllocRegExcept(d1188.Reg, d1190.Reg)
				ctx.EmitMovRegReg(r119, d1188.Reg)
				shiftSrc = r119
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d1190.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d1190.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d1190.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d1191 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d1191)
			}
		}
		if d1191.Loc == scm.LocReg && d1188.Loc == scm.LocReg && d1191.Reg == d1188.Reg {
			ctx.TransferReg(d1188.Reg)
			d1188.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1188)
		ctx.FreeDesc(&d1190)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1186)
		ctx.EnsureDesc(&d1191)
		var d1192 scm.JITValueDesc
		if d1186.Loc == scm.LocImm && d1191.Loc == scm.LocImm {
			d1192 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1186.Imm.Int() | d1191.Imm.Int())}
		} else if d1186.Loc == scm.LocImm && d1186.Imm.Int() == 0 {
			d1192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1191.Reg}
			ctx.BindReg(d1191.Reg, &d1192)
		} else if d1191.Loc == scm.LocImm && d1191.Imm.Int() == 0 {
			r120 := ctx.AllocRegExcept(d1186.Reg)
			ctx.EmitMovRegReg(r120, d1186.Reg)
			d1192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r120}
			ctx.BindReg(r120, &d1192)
		} else if d1186.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1191.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1186.Imm.Int()))
			ctx.EmitOrInt64(scratch, d1191.Reg)
			d1192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1192)
		} else if d1191.Loc == scm.LocImm {
			r121 := ctx.AllocRegExcept(d1186.Reg)
			ctx.EmitMovRegReg(r121, d1186.Reg)
			if d1191.Imm.Int() >= -2147483648 && d1191.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r121, int32(d1191.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1191.Imm.Int()))
				ctx.EmitOrInt64(r121, scm.RegR11)
			}
			d1192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r121}
			ctx.BindReg(r121, &d1192)
		} else {
			r122 := ctx.AllocRegExcept(d1186.Reg, d1191.Reg)
			ctx.EmitMovRegReg(r122, d1186.Reg)
			ctx.EmitOrInt64(r122, d1191.Reg)
			d1192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
			ctx.BindReg(r122, &d1192)
		}
		if d1192.Loc == scm.LocReg && d1186.Loc == scm.LocReg && d1192.Reg == d1186.Reg {
			ctx.TransferReg(d1186.Reg)
			d1186.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1186)
		ctx.FreeDesc(&d1191)
		ctx.ReclaimUntrackedRegs()
		d1193 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d1179)
		ctx.EnsureDescsTogether(&d1193, &d1179)
		var d1194 scm.JITValueDesc
		if d1193.Loc == scm.LocImm && d1179.Loc == scm.LocImm {
			d1194 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1193.Imm.Int() - d1179.Imm.Int())}
		} else if d1179.Loc == scm.LocImm && d1179.Imm.Int() == 0 {
			r123 := ctx.AllocRegExcept(d1193.Reg)
			ctx.EmitMovRegReg(r123, d1193.Reg)
			d1194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
			ctx.BindReg(r123, &d1194)
		} else if d1193.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1179.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1193.Imm.Int()))
			ctx.EmitSubInt64(scratch, d1179.Reg)
			d1194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1194)
		} else if d1179.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1193.Reg)
			ctx.EmitMovRegReg(scratch, d1193.Reg)
			if d1179.Imm.Int() >= -2147483648 && d1179.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d1179.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1179.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d1194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1194)
		} else {
			r124 := ctx.AllocRegExcept(d1193.Reg, d1179.Reg)
			ctx.EmitMovRegReg(r124, d1193.Reg)
			ctx.EmitSubInt64(r124, d1179.Reg)
			d1194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
			ctx.BindReg(r124, &d1194)
		}
		if d1194.Loc == scm.LocReg && d1193.Loc == scm.LocReg && d1194.Reg == d1193.Reg {
			ctx.TransferReg(d1193.Reg)
			d1193.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1179)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1192)
		ctx.EnsureDesc(&d1194)
		ctx.EnsureDescsTogether(&d1192, &d1194)
		var d1195 scm.JITValueDesc
		if d1192.Loc == scm.LocImm && d1194.Loc == scm.LocImm {
			d1195 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1192.Imm.Int()) >> uint64(d1194.Imm.Int())))}
		} else if d1194.Loc == scm.LocImm {
			r125 := ctx.AllocRegExcept(d1192.Reg)
			ctx.EmitMovRegReg(r125, d1192.Reg)
			ctx.EmitShrRegImm8(r125, uint8(d1194.Imm.Int()))
			d1195 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
			ctx.BindReg(r125, &d1195)
		} else {
			{
				shiftSrc := d1192.Reg
				r126 := ctx.AllocRegExcept(d1192.Reg, d1194.Reg)
				ctx.EmitMovRegReg(r126, d1192.Reg)
				shiftSrc = r126
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d1194.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d1194.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d1194.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d1195 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d1195)
			}
		}
		if d1195.Loc == scm.LocReg && d1192.Loc == scm.LocReg && d1195.Reg == d1192.Reg {
			ctx.TransferReg(d1192.Reg)
			d1192.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1192)
		ctx.FreeDesc(&d1194)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1195)
		ctx.EnsureDesc(&d1195)
		ctx.EnsureDesc(&d1195)
		var d1196 scm.JITValueDesc
		if d1195.Loc == scm.LocImm {
			d1196 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d1195.Imm.Int()))))}
		} else {
			r127 := ctx.AllocReg()
			ctx.EmitMovRegReg(r127, d1195.Reg)
			d1196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
			ctx.BindReg(r127, &d1196)
		}
		ctx.FreeDesc(&d1195)
		var d1197 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d1197 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 56)
			r128 := ctx.AllocReg()
			ctx.EmitMovRegMem(r128, thisptr.Reg, off)
			d1197 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r128}
			ctx.BindReg(r128, &d1197)
		}
		ctx.EnsureDesc(&d1196)
		ctx.EnsureDesc(&d1197)
		ctx.EnsureDescsTogether(&d1196, &d1197)
		var d1198 scm.JITValueDesc
		if d1196.Loc == scm.LocImm && d1197.Loc == scm.LocImm {
			d1198 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1196.Imm.Int() + d1197.Imm.Int())}
		} else if d1197.Loc == scm.LocImm && d1197.Imm.Int() == 0 {
			r129 := ctx.AllocRegExcept(d1196.Reg)
			ctx.EmitMovRegReg(r129, d1196.Reg)
			d1198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
			ctx.BindReg(r129, &d1198)
		} else if d1196.Loc == scm.LocImm && d1196.Imm.Int() == 0 {
			d1198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1197.Reg}
			ctx.BindReg(d1197.Reg, &d1198)
		} else if d1196.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1197.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1196.Imm.Int()))
			ctx.EmitAddInt64(scratch, d1197.Reg)
			d1198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1198)
		} else if d1197.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1196.Reg)
			ctx.EmitMovRegReg(scratch, d1196.Reg)
			if d1197.Imm.Int() >= -2147483648 && d1197.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d1197.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1197.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d1198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1198)
		} else {
			r130 := ctx.AllocRegExcept(d1196.Reg, d1197.Reg)
			ctx.EmitMovRegReg(r130, d1196.Reg)
			ctx.EmitAddInt64(r130, d1197.Reg)
			d1198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
			ctx.BindReg(r130, &d1198)
		}
		if d1198.Loc == scm.LocReg && d1196.Loc == scm.LocReg && d1198.Reg == d1196.Reg {
			ctx.TransferReg(d1196.Reg)
			d1196.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1196)
		ctx.FreeDesc(&d1197)
		ctx.EnsureDesc(&d8)
		d1199 = d8
		_ = d1199
		bbpos_5_0 := int32(-1)
		_ = bbpos_5_0
		lbl21 := ctx.ReserveLabel()
		_ = lbl21
		bbpos_5_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl21)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d1200 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d1200 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48)
			r131 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r131, thisptr.Reg, off)
			d1200 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r131}
			ctx.BindReg(r131, &d1200)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1200)
		ctx.EnsureDesc(&d1200)
		var d1201 scm.JITValueDesc
		if d1200.Loc == scm.LocImm {
			d1201 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d1200.Imm.Int()))))}
		} else {
			r132 := ctx.AllocReg()
			ctx.EmitMovRegReg(r132, d1200.Reg)
			ctx.EmitShlRegImm8(r132, 56)
			ctx.EmitShrRegImm8(r132, 56)
			d1201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
			ctx.BindReg(r132, &d1201)
		}
		ctx.FreeDesc(&d1200)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1199)
		ctx.EnsureDesc(&d1199)
		var d1202 scm.JITValueDesc
		if d1199.Loc == scm.LocImm {
			d1202 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d1199.Imm.Int()))))}
		} else {
			r133 := ctx.AllocReg()
			ctx.EmitMovRegReg(r133, d1199.Reg)
			ctx.EmitShlRegImm8(r133, 32)
			ctx.EmitShrRegImm8(r133, 32)
			d1202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
			ctx.BindReg(r133, &d1202)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1202)
		ctx.EnsureDesc(&d1201)
		ctx.EnsureDescsTogether(&d1202, &d1201)
		var d1203 scm.JITValueDesc
		if d1202.Loc == scm.LocImm && d1201.Loc == scm.LocImm {
			d1203 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1202.Imm.Int() * d1201.Imm.Int())}
		} else if d1202.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1201.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1202.Imm.Int()))
			ctx.EmitImulInt64(scratch, d1201.Reg)
			d1203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1203)
		} else if d1201.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1202.Reg)
			ctx.EmitMovRegReg(scratch, d1202.Reg)
			if d1201.Imm.Int() >= -2147483648 && d1201.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d1201.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1201.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d1203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1203)
		} else {
			r134 := ctx.AllocRegExcept(d1202.Reg, d1201.Reg)
			ctx.EmitMovRegReg(r134, d1202.Reg)
			ctx.EmitImulInt64(r134, d1201.Reg)
			d1203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r134}
			ctx.BindReg(r134, &d1203)
		}
		if d1203.Loc == scm.LocReg && d1202.Loc == scm.LocReg && d1203.Reg == d1202.Reg {
			ctx.TransferReg(d1202.Reg)
			d1202.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1202)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1203)
		var d1204 scm.JITValueDesc
		if d1203.Loc == scm.LocImm {
			d1204 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1203.Imm.Int() / 64)}
		} else {
			r135 := ctx.AllocRegExcept(d1203.Reg)
			ctx.EmitMovRegReg(r135, d1203.Reg)
			ctx.EmitShrRegImm8(r135, 6)
			d1204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r135}
			ctx.BindReg(r135, &d1204)
		}
		if d1204.Loc == scm.LocReg && d1203.Loc == scm.LocReg && d1204.Reg == d1203.Reg {
			ctx.TransferReg(d1203.Reg)
			d1203.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1203)
		var d1205 scm.JITValueDesc
		if d1203.Loc == scm.LocImm {
			d1205 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1203.Imm.Int() % 64)}
		} else {
			r136 := ctx.AllocRegExcept(d1203.Reg)
			ctx.EmitMovRegReg(r136, d1203.Reg)
			ctx.EmitAndRegImm32(r136, 63)
			d1205 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r136}
			ctx.BindReg(r136, &d1205)
		}
		if d1205.Loc == scm.LocReg && d1203.Loc == scm.LocReg && d1205.Reg == d1203.Reg {
			ctx.TransferReg(d1203.Reg)
			d1203.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1203)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d1206 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d1206 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r137 := ctx.AllocReg()
			r138 := ctx.AllocRegExcept(r137)
			r139 := ctx.AllocRegExcept(r137, r138)
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			ctx.EmitMovRegMem(r137, thisptr.Reg, off)
			ctx.EmitMovRegMem(r138, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r139, thisptr.Reg, off+16)
			d1206 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r137, Reg2: r138, Reg3: r139}
			ctx.BindReg(r137, &d1206)
			ctx.BindReg(r138, &d1206)
			ctx.BindReg(r139, &d1206)
			ctx.BindReg(r137, &d1206)
			ctx.BindReg(r138, &d1206)
			ctx.BindReg(r139, &d1206)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1204)
		ctx.ReclaimUntrackedRegs()
		d1207 = ctx.EmitLoadScalarSliceElement(&d1206, &d1204, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1207)
		ctx.EnsureDesc(&d1205)
		ctx.EnsureDescsTogether(&d1207, &d1205)
		var d1208 scm.JITValueDesc
		if d1207.Loc == scm.LocImm && d1205.Loc == scm.LocImm {
			d1208 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1207.Imm.Int()) << uint64(d1205.Imm.Int())))}
		} else if d1205.Loc == scm.LocImm {
			r140 := ctx.AllocRegExcept(d1207.Reg)
			ctx.EmitMovRegReg(r140, d1207.Reg)
			ctx.EmitShlRegImm8(r140, uint8(d1205.Imm.Int()))
			d1208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r140}
			ctx.BindReg(r140, &d1208)
		} else {
			{
				shiftSrc := d1207.Reg
				r141 := ctx.AllocRegExcept(d1207.Reg, d1205.Reg)
				ctx.EmitMovRegReg(r141, d1207.Reg)
				shiftSrc = r141
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d1205.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d1205.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d1205.Reg)
				}
				ctx.EmitShlRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d1208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d1208)
			}
		}
		if d1208.Loc == scm.LocReg && d1207.Loc == scm.LocReg && d1208.Reg == d1207.Reg {
			ctx.TransferReg(d1207.Reg)
			d1207.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1207)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1204)
		ctx.EnsureDesc(&d1204)
		var d1209 scm.JITValueDesc
		if d1204.Loc == scm.LocImm {
			d1209 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1204.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d1204.Reg)
			ctx.EmitMovRegReg(scratch, d1204.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d1209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1209)
		}
		if d1209.Loc == scm.LocReg && d1204.Loc == scm.LocReg && d1209.Reg == d1204.Reg {
			ctx.TransferReg(d1204.Reg)
			d1204.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1204)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1209)
		ctx.ReclaimUntrackedRegs()
		d1210 = ctx.EmitLoadScalarSliceElement(&d1206, &d1209, 8, scm.TagInt)
		ctx.FreeDesc(&d1209)
		ctx.ReclaimUntrackedRegs()
		d1211 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d1205)
		ctx.EnsureDescsTogether(&d1211, &d1205)
		var d1212 scm.JITValueDesc
		if d1211.Loc == scm.LocImm && d1205.Loc == scm.LocImm {
			d1212 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1211.Imm.Int() - d1205.Imm.Int())}
		} else if d1205.Loc == scm.LocImm && d1205.Imm.Int() == 0 {
			r142 := ctx.AllocRegExcept(d1211.Reg)
			ctx.EmitMovRegReg(r142, d1211.Reg)
			d1212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r142}
			ctx.BindReg(r142, &d1212)
		} else if d1211.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1205.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1211.Imm.Int()))
			ctx.EmitSubInt64(scratch, d1205.Reg)
			d1212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1212)
		} else if d1205.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1211.Reg)
			ctx.EmitMovRegReg(scratch, d1211.Reg)
			if d1205.Imm.Int() >= -2147483648 && d1205.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d1205.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1205.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d1212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1212)
		} else {
			r143 := ctx.AllocRegExcept(d1211.Reg, d1205.Reg)
			ctx.EmitMovRegReg(r143, d1211.Reg)
			ctx.EmitSubInt64(r143, d1205.Reg)
			d1212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r143}
			ctx.BindReg(r143, &d1212)
		}
		if d1212.Loc == scm.LocReg && d1211.Loc == scm.LocReg && d1212.Reg == d1211.Reg {
			ctx.TransferReg(d1211.Reg)
			d1211.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1205)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1210)
		ctx.EnsureDesc(&d1212)
		ctx.EnsureDescsTogether(&d1210, &d1212)
		var d1213 scm.JITValueDesc
		if d1210.Loc == scm.LocImm && d1212.Loc == scm.LocImm {
			d1213 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1210.Imm.Int()) >> uint64(d1212.Imm.Int())))}
		} else if d1212.Loc == scm.LocImm {
			r144 := ctx.AllocRegExcept(d1210.Reg)
			ctx.EmitMovRegReg(r144, d1210.Reg)
			ctx.EmitShrRegImm8(r144, uint8(d1212.Imm.Int()))
			d1213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
			ctx.BindReg(r144, &d1213)
		} else {
			{
				shiftSrc := d1210.Reg
				r145 := ctx.AllocRegExcept(d1210.Reg, d1212.Reg)
				ctx.EmitMovRegReg(r145, d1210.Reg)
				shiftSrc = r145
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d1212.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d1212.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d1212.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d1213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d1213)
			}
		}
		if d1213.Loc == scm.LocReg && d1210.Loc == scm.LocReg && d1213.Reg == d1210.Reg {
			ctx.TransferReg(d1210.Reg)
			d1210.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1210)
		ctx.FreeDesc(&d1212)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1208)
		ctx.EnsureDesc(&d1213)
		var d1214 scm.JITValueDesc
		if d1208.Loc == scm.LocImm && d1213.Loc == scm.LocImm {
			d1214 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1208.Imm.Int() | d1213.Imm.Int())}
		} else if d1208.Loc == scm.LocImm && d1208.Imm.Int() == 0 {
			d1214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1213.Reg}
			ctx.BindReg(d1213.Reg, &d1214)
		} else if d1213.Loc == scm.LocImm && d1213.Imm.Int() == 0 {
			r146 := ctx.AllocRegExcept(d1208.Reg)
			ctx.EmitMovRegReg(r146, d1208.Reg)
			d1214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r146}
			ctx.BindReg(r146, &d1214)
		} else if d1208.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1213.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1208.Imm.Int()))
			ctx.EmitOrInt64(scratch, d1213.Reg)
			d1214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1214)
		} else if d1213.Loc == scm.LocImm {
			r147 := ctx.AllocRegExcept(d1208.Reg)
			ctx.EmitMovRegReg(r147, d1208.Reg)
			if d1213.Imm.Int() >= -2147483648 && d1213.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r147, int32(d1213.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1213.Imm.Int()))
				ctx.EmitOrInt64(r147, scm.RegR11)
			}
			d1214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r147}
			ctx.BindReg(r147, &d1214)
		} else {
			r148 := ctx.AllocRegExcept(d1208.Reg, d1213.Reg)
			ctx.EmitMovRegReg(r148, d1208.Reg)
			ctx.EmitOrInt64(r148, d1213.Reg)
			d1214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r148}
			ctx.BindReg(r148, &d1214)
		}
		if d1214.Loc == scm.LocReg && d1208.Loc == scm.LocReg && d1214.Reg == d1208.Reg {
			ctx.TransferReg(d1208.Reg)
			d1208.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1208)
		ctx.FreeDesc(&d1213)
		ctx.ReclaimUntrackedRegs()
		d1215 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d1201)
		ctx.EnsureDescsTogether(&d1215, &d1201)
		var d1216 scm.JITValueDesc
		if d1215.Loc == scm.LocImm && d1201.Loc == scm.LocImm {
			d1216 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1215.Imm.Int() - d1201.Imm.Int())}
		} else if d1201.Loc == scm.LocImm && d1201.Imm.Int() == 0 {
			r149 := ctx.AllocRegExcept(d1215.Reg)
			ctx.EmitMovRegReg(r149, d1215.Reg)
			d1216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r149}
			ctx.BindReg(r149, &d1216)
		} else if d1215.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1201.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1215.Imm.Int()))
			ctx.EmitSubInt64(scratch, d1201.Reg)
			d1216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1216)
		} else if d1201.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1215.Reg)
			ctx.EmitMovRegReg(scratch, d1215.Reg)
			if d1201.Imm.Int() >= -2147483648 && d1201.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d1201.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1201.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d1216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1216)
		} else {
			r150 := ctx.AllocRegExcept(d1215.Reg, d1201.Reg)
			ctx.EmitMovRegReg(r150, d1215.Reg)
			ctx.EmitSubInt64(r150, d1201.Reg)
			d1216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r150}
			ctx.BindReg(r150, &d1216)
		}
		if d1216.Loc == scm.LocReg && d1215.Loc == scm.LocReg && d1216.Reg == d1215.Reg {
			ctx.TransferReg(d1215.Reg)
			d1215.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1201)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1214)
		ctx.EnsureDesc(&d1216)
		ctx.EnsureDescsTogether(&d1214, &d1216)
		var d1217 scm.JITValueDesc
		if d1214.Loc == scm.LocImm && d1216.Loc == scm.LocImm {
			d1217 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1214.Imm.Int()) >> uint64(d1216.Imm.Int())))}
		} else if d1216.Loc == scm.LocImm {
			r151 := ctx.AllocRegExcept(d1214.Reg)
			ctx.EmitMovRegReg(r151, d1214.Reg)
			ctx.EmitShrRegImm8(r151, uint8(d1216.Imm.Int()))
			d1217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
			ctx.BindReg(r151, &d1217)
		} else {
			{
				shiftSrc := d1214.Reg
				r152 := ctx.AllocRegExcept(d1214.Reg, d1216.Reg)
				ctx.EmitMovRegReg(r152, d1214.Reg)
				shiftSrc = r152
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d1216.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d1216.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d1216.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d1217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d1217)
			}
		}
		if d1217.Loc == scm.LocReg && d1214.Loc == scm.LocReg && d1217.Reg == d1214.Reg {
			ctx.TransferReg(d1214.Reg)
			d1214.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1214)
		ctx.FreeDesc(&d1216)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1217)
		ctx.EnsureDesc(&d1217)
		ctx.EnsureDesc(&d1217)
		var d1218 scm.JITValueDesc
		if d1217.Loc == scm.LocImm {
			d1218 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d1217.Imm.Int()))))}
		} else {
			r153 := ctx.AllocReg()
			ctx.EmitMovRegReg(r153, d1217.Reg)
			d1218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r153}
			ctx.BindReg(r153, &d1218)
		}
		ctx.FreeDesc(&d1217)
		var d1219 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d1219 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56)
			r154 := ctx.AllocReg()
			ctx.EmitMovRegMem(r154, thisptr.Reg, off)
			d1219 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r154}
			ctx.BindReg(r154, &d1219)
		}
		ctx.EnsureDesc(&d1218)
		ctx.EnsureDesc(&d1219)
		ctx.EnsureDescsTogether(&d1218, &d1219)
		var d1220 scm.JITValueDesc
		if d1218.Loc == scm.LocImm && d1219.Loc == scm.LocImm {
			d1220 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1218.Imm.Int() + d1219.Imm.Int())}
		} else if d1219.Loc == scm.LocImm && d1219.Imm.Int() == 0 {
			r155 := ctx.AllocRegExcept(d1218.Reg)
			ctx.EmitMovRegReg(r155, d1218.Reg)
			d1220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r155}
			ctx.BindReg(r155, &d1220)
		} else if d1218.Loc == scm.LocImm && d1218.Imm.Int() == 0 {
			d1220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1219.Reg}
			ctx.BindReg(d1219.Reg, &d1220)
		} else if d1218.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1219.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1218.Imm.Int()))
			ctx.EmitAddInt64(scratch, d1219.Reg)
			d1220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1220)
		} else if d1219.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1218.Reg)
			ctx.EmitMovRegReg(scratch, d1218.Reg)
			if d1219.Imm.Int() >= -2147483648 && d1219.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d1219.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1219.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d1220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1220)
		} else {
			r156 := ctx.AllocRegExcept(d1218.Reg, d1219.Reg)
			ctx.EmitMovRegReg(r156, d1218.Reg)
			ctx.EmitAddInt64(r156, d1219.Reg)
			d1220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r156}
			ctx.BindReg(r156, &d1220)
		}
		if d1220.Loc == scm.LocReg && d1218.Loc == scm.LocReg && d1220.Reg == d1218.Reg {
			ctx.TransferReg(d1218.Reg)
			d1218.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1218)
		ctx.FreeDesc(&d1219)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d1220)
		ctx.EnsureDescsTogether(&idxInt, &d1220)
		var d1222 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm && d1220.Loc == scm.LocImm {
			d1222 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(idxInt.Imm.Int() - d1220.Imm.Int())}
		} else if d1220.Loc == scm.LocImm && d1220.Imm.Int() == 0 {
			r157 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitMovRegReg(r157, idxInt.Reg)
			d1222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r157}
			ctx.BindReg(r157, &d1222)
		} else if idxInt.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1220.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(idxInt.Imm.Int()))
			ctx.EmitSubInt64(scratch, d1220.Reg)
			d1222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1222)
		} else if d1220.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitMovRegReg(scratch, idxInt.Reg)
			if d1220.Imm.Int() >= -2147483648 && d1220.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d1220.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1220.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d1222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1222)
		} else {
			r158 := ctx.AllocRegExcept(idxInt.Reg, d1220.Reg)
			ctx.EmitMovRegReg(r158, idxInt.Reg)
			ctx.EmitSubInt64(r158, d1220.Reg)
			d1222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r158}
			ctx.BindReg(r158, &d1222)
		}
		if d1222.Loc == scm.LocReg && idxInt.Loc == scm.LocReg && d1222.Reg == idxInt.Reg {
			ctx.TransferReg(idxInt.Reg)
			idxInt.Loc = scm.LocNone
		}
		ctx.FreeDesc(&idxInt)
		ctx.FreeDesc(&d1220)
		ctx.EnsureDesc(&d1222)
		ctx.EnsureDesc(&d1198)
		ctx.EnsureDescsTogether(&d1222, &d1198)
		var d1223 scm.JITValueDesc
		if d1222.Loc == scm.LocImm && d1198.Loc == scm.LocImm {
			d1223 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1222.Imm.Int() * d1198.Imm.Int())}
		} else if d1222.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1198.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1222.Imm.Int()))
			ctx.EmitImulInt64(scratch, d1198.Reg)
			d1223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1223)
		} else if d1198.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1222.Reg)
			ctx.EmitMovRegReg(scratch, d1222.Reg)
			if d1198.Imm.Int() >= -2147483648 && d1198.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d1198.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1198.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d1223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1223)
		} else {
			r159 := ctx.AllocRegExcept(d1222.Reg, d1198.Reg)
			ctx.EmitMovRegReg(r159, d1222.Reg)
			ctx.EmitImulInt64(r159, d1198.Reg)
			d1223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
			ctx.BindReg(r159, &d1223)
		}
		if d1223.Loc == scm.LocReg && d1222.Loc == scm.LocReg && d1223.Reg == d1222.Reg {
			ctx.TransferReg(d1222.Reg)
			d1222.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1222)
		ctx.FreeDesc(&d1198)
		ctx.EnsureDesc(&d182)
		ctx.EnsureDesc(&d1223)
		ctx.EnsureDescsTogether(&d182, &d1223)
		var d1224 scm.JITValueDesc
		if d182.Loc == scm.LocImm && d1223.Loc == scm.LocImm {
			d1224 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d182.Imm.Int() + d1223.Imm.Int())}
		} else if d1223.Loc == scm.LocImm && d1223.Imm.Int() == 0 {
			r160 := ctx.AllocRegExcept(d182.Reg)
			ctx.EmitMovRegReg(r160, d182.Reg)
			d1224 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r160}
			ctx.BindReg(r160, &d1224)
		} else if d182.Loc == scm.LocImm && d182.Imm.Int() == 0 {
			d1224 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1223.Reg}
			ctx.BindReg(d1223.Reg, &d1224)
		} else if d182.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1223.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d182.Imm.Int()))
			ctx.EmitAddInt64(scratch, d1223.Reg)
			d1224 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1224)
		} else if d1223.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d182.Reg)
			ctx.EmitMovRegReg(scratch, d182.Reg)
			if d1223.Imm.Int() >= -2147483648 && d1223.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d1223.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1223.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d1224 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1224)
		} else {
			r161 := ctx.AllocRegExcept(d182.Reg, d1223.Reg)
			ctx.EmitMovRegReg(r161, d182.Reg)
			ctx.EmitAddInt64(r161, d1223.Reg)
			d1224 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r161}
			ctx.BindReg(r161, &d1224)
		}
		if d1224.Loc == scm.LocReg && d182.Loc == scm.LocReg && d1224.Reg == d182.Reg {
			ctx.TransferReg(d182.Reg)
			d182.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1223)
		ctx.EnsureDesc(&d1224)
		ctx.EnsureDesc(&d1224)
		var d1225 scm.JITValueDesc
		if d1224.Loc == scm.LocImm {
			d1225 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d1224.Imm.Int()))}
		} else {
			r162 := ctx.AllocRegExcept(d1224.Reg)
			ctx.EmitMovRegReg(r162, d1224.Reg)
			ctx.EmitCvtInt64ToFloat64(scm.RegX0, r162)
			d1225 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: r162}
			ctx.BindReg(r162, &d1225)
		}
		ctx.FreeDesc(&d1224)
		ctx.EnsureDesc(&d1225)
		d1226 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r3, Reg2: r4}
		ctx.BindReg(r3, &d1226)
		ctx.BindReg(r4, &d1226)
		ctx.EnsureDesc(&d1225)
		ctx.EmitMakeFloat(d1226, d1225)
		if d1225.Loc == scm.LocReg {
			ctx.FreeReg(d1225.Reg)
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
		if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
			d187 = ps.OverlayValues[187]
		}
		if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != scm.LocNone {
			d344 = ps.OverlayValues[344]
		}
		if len(ps.OverlayValues) > 345 && ps.OverlayValues[345].Loc != scm.LocNone {
			d345 = ps.OverlayValues[345]
		}
		if len(ps.OverlayValues) > 346 && ps.OverlayValues[346].Loc != scm.LocNone {
			d346 = ps.OverlayValues[346]
		}
		if len(ps.OverlayValues) > 347 && ps.OverlayValues[347].Loc != scm.LocNone {
			d347 = ps.OverlayValues[347]
		}
		if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != scm.LocNone {
			d349 = ps.OverlayValues[349]
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
		if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != scm.LocNone {
			d355 = ps.OverlayValues[355]
		}
		if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != scm.LocNone {
			d356 = ps.OverlayValues[356]
		}
		if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != scm.LocNone {
			d358 = ps.OverlayValues[358]
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
		if len(ps.OverlayValues) > 456 && ps.OverlayValues[456].Loc != scm.LocNone {
			d456 = ps.OverlayValues[456]
		}
		if len(ps.OverlayValues) > 457 && ps.OverlayValues[457].Loc != scm.LocNone {
			d457 = ps.OverlayValues[457]
		}
		if len(ps.OverlayValues) > 460 && ps.OverlayValues[460].Loc != scm.LocNone {
			d460 = ps.OverlayValues[460]
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
		if len(ps.OverlayValues) > 561 && ps.OverlayValues[561].Loc != scm.LocNone {
			d561 = ps.OverlayValues[561]
		}
		if len(ps.OverlayValues) > 563 && ps.OverlayValues[563].Loc != scm.LocNone {
			d563 = ps.OverlayValues[563]
		}
		if len(ps.OverlayValues) > 564 && ps.OverlayValues[564].Loc != scm.LocNone {
			d564 = ps.OverlayValues[564]
		}
		if len(ps.OverlayValues) > 565 && ps.OverlayValues[565].Loc != scm.LocNone {
			d565 = ps.OverlayValues[565]
		}
		if len(ps.OverlayValues) > 566 && ps.OverlayValues[566].Loc != scm.LocNone {
			d566 = ps.OverlayValues[566]
		}
		if len(ps.OverlayValues) > 567 && ps.OverlayValues[567].Loc != scm.LocNone {
			d567 = ps.OverlayValues[567]
		}
		if len(ps.OverlayValues) > 568 && ps.OverlayValues[568].Loc != scm.LocNone {
			d568 = ps.OverlayValues[568]
		}
		if len(ps.OverlayValues) > 569 && ps.OverlayValues[569].Loc != scm.LocNone {
			d569 = ps.OverlayValues[569]
		}
		if len(ps.OverlayValues) > 570 && ps.OverlayValues[570].Loc != scm.LocNone {
			d570 = ps.OverlayValues[570]
		}
		if len(ps.OverlayValues) > 571 && ps.OverlayValues[571].Loc != scm.LocNone {
			d571 = ps.OverlayValues[571]
		}
		if len(ps.OverlayValues) > 572 && ps.OverlayValues[572].Loc != scm.LocNone {
			d572 = ps.OverlayValues[572]
		}
		if len(ps.OverlayValues) > 573 && ps.OverlayValues[573].Loc != scm.LocNone {
			d573 = ps.OverlayValues[573]
		}
		if len(ps.OverlayValues) > 574 && ps.OverlayValues[574].Loc != scm.LocNone {
			d574 = ps.OverlayValues[574]
		}
		if len(ps.OverlayValues) > 575 && ps.OverlayValues[575].Loc != scm.LocNone {
			d575 = ps.OverlayValues[575]
		}
		if len(ps.OverlayValues) > 576 && ps.OverlayValues[576].Loc != scm.LocNone {
			d576 = ps.OverlayValues[576]
		}
		if len(ps.OverlayValues) > 577 && ps.OverlayValues[577].Loc != scm.LocNone {
			d577 = ps.OverlayValues[577]
		}
		if len(ps.OverlayValues) > 578 && ps.OverlayValues[578].Loc != scm.LocNone {
			d578 = ps.OverlayValues[578]
		}
		if len(ps.OverlayValues) > 579 && ps.OverlayValues[579].Loc != scm.LocNone {
			d579 = ps.OverlayValues[579]
		}
		if len(ps.OverlayValues) > 580 && ps.OverlayValues[580].Loc != scm.LocNone {
			d580 = ps.OverlayValues[580]
		}
		if len(ps.OverlayValues) > 581 && ps.OverlayValues[581].Loc != scm.LocNone {
			d581 = ps.OverlayValues[581]
		}
		if len(ps.OverlayValues) > 582 && ps.OverlayValues[582].Loc != scm.LocNone {
			d582 = ps.OverlayValues[582]
		}
		if len(ps.OverlayValues) > 583 && ps.OverlayValues[583].Loc != scm.LocNone {
			d583 = ps.OverlayValues[583]
		}
		if len(ps.OverlayValues) > 584 && ps.OverlayValues[584].Loc != scm.LocNone {
			d584 = ps.OverlayValues[584]
		}
		if len(ps.OverlayValues) > 585 && ps.OverlayValues[585].Loc != scm.LocNone {
			d585 = ps.OverlayValues[585]
		}
		if len(ps.OverlayValues) > 586 && ps.OverlayValues[586].Loc != scm.LocNone {
			d586 = ps.OverlayValues[586]
		}
		if len(ps.OverlayValues) > 587 && ps.OverlayValues[587].Loc != scm.LocNone {
			d587 = ps.OverlayValues[587]
		}
		if len(ps.OverlayValues) > 588 && ps.OverlayValues[588].Loc != scm.LocNone {
			d588 = ps.OverlayValues[588]
		}
		if len(ps.OverlayValues) > 589 && ps.OverlayValues[589].Loc != scm.LocNone {
			d589 = ps.OverlayValues[589]
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
		if len(ps.OverlayValues) > 862 && ps.OverlayValues[862].Loc != scm.LocNone {
			d862 = ps.OverlayValues[862]
		}
		if len(ps.OverlayValues) > 864 && ps.OverlayValues[864].Loc != scm.LocNone {
			d864 = ps.OverlayValues[864]
		}
		if len(ps.OverlayValues) > 865 && ps.OverlayValues[865].Loc != scm.LocNone {
			d865 = ps.OverlayValues[865]
		}
		if len(ps.OverlayValues) > 1007 && ps.OverlayValues[1007].Loc != scm.LocNone {
			d1007 = ps.OverlayValues[1007]
		}
		if len(ps.OverlayValues) > 1008 && ps.OverlayValues[1008].Loc != scm.LocNone {
			d1008 = ps.OverlayValues[1008]
		}
		if len(ps.OverlayValues) > 1011 && ps.OverlayValues[1011].Loc != scm.LocNone {
			d1011 = ps.OverlayValues[1011]
		}
		if len(ps.OverlayValues) > 1156 && ps.OverlayValues[1156].Loc != scm.LocNone {
			d1156 = ps.OverlayValues[1156]
		}
		if len(ps.OverlayValues) > 1157 && ps.OverlayValues[1157].Loc != scm.LocNone {
			d1157 = ps.OverlayValues[1157]
		}
		if len(ps.OverlayValues) > 1158 && ps.OverlayValues[1158].Loc != scm.LocNone {
			d1158 = ps.OverlayValues[1158]
		}
		if len(ps.OverlayValues) > 1159 && ps.OverlayValues[1159].Loc != scm.LocNone {
			d1159 = ps.OverlayValues[1159]
		}
		if len(ps.OverlayValues) > 1161 && ps.OverlayValues[1161].Loc != scm.LocNone {
			d1161 = ps.OverlayValues[1161]
		}
		if len(ps.OverlayValues) > 1162 && ps.OverlayValues[1162].Loc != scm.LocNone {
			d1162 = ps.OverlayValues[1162]
		}
		if len(ps.OverlayValues) > 1163 && ps.OverlayValues[1163].Loc != scm.LocNone {
			d1163 = ps.OverlayValues[1163]
		}
		if len(ps.OverlayValues) > 1164 && ps.OverlayValues[1164].Loc != scm.LocNone {
			d1164 = ps.OverlayValues[1164]
		}
		if len(ps.OverlayValues) > 1165 && ps.OverlayValues[1165].Loc != scm.LocNone {
			d1165 = ps.OverlayValues[1165]
		}
		if len(ps.OverlayValues) > 1166 && ps.OverlayValues[1166].Loc != scm.LocNone {
			d1166 = ps.OverlayValues[1166]
		}
		if len(ps.OverlayValues) > 1167 && ps.OverlayValues[1167].Loc != scm.LocNone {
			d1167 = ps.OverlayValues[1167]
		}
		if len(ps.OverlayValues) > 1168 && ps.OverlayValues[1168].Loc != scm.LocNone {
			d1168 = ps.OverlayValues[1168]
		}
		if len(ps.OverlayValues) > 1169 && ps.OverlayValues[1169].Loc != scm.LocNone {
			d1169 = ps.OverlayValues[1169]
		}
		if len(ps.OverlayValues) > 1170 && ps.OverlayValues[1170].Loc != scm.LocNone {
			d1170 = ps.OverlayValues[1170]
		}
		if len(ps.OverlayValues) > 1172 && ps.OverlayValues[1172].Loc != scm.LocNone {
			d1172 = ps.OverlayValues[1172]
		}
		if len(ps.OverlayValues) > 1173 && ps.OverlayValues[1173].Loc != scm.LocNone {
			d1173 = ps.OverlayValues[1173]
		}
		if len(ps.OverlayValues) > 1174 && ps.OverlayValues[1174].Loc != scm.LocNone {
			d1174 = ps.OverlayValues[1174]
		}
		if len(ps.OverlayValues) > 1175 && ps.OverlayValues[1175].Loc != scm.LocNone {
			d1175 = ps.OverlayValues[1175]
		}
		if len(ps.OverlayValues) > 1176 && ps.OverlayValues[1176].Loc != scm.LocNone {
			d1176 = ps.OverlayValues[1176]
		}
		if len(ps.OverlayValues) > 1177 && ps.OverlayValues[1177].Loc != scm.LocNone {
			d1177 = ps.OverlayValues[1177]
		}
		if len(ps.OverlayValues) > 1178 && ps.OverlayValues[1178].Loc != scm.LocNone {
			d1178 = ps.OverlayValues[1178]
		}
		if len(ps.OverlayValues) > 1179 && ps.OverlayValues[1179].Loc != scm.LocNone {
			d1179 = ps.OverlayValues[1179]
		}
		if len(ps.OverlayValues) > 1180 && ps.OverlayValues[1180].Loc != scm.LocNone {
			d1180 = ps.OverlayValues[1180]
		}
		if len(ps.OverlayValues) > 1181 && ps.OverlayValues[1181].Loc != scm.LocNone {
			d1181 = ps.OverlayValues[1181]
		}
		if len(ps.OverlayValues) > 1182 && ps.OverlayValues[1182].Loc != scm.LocNone {
			d1182 = ps.OverlayValues[1182]
		}
		if len(ps.OverlayValues) > 1183 && ps.OverlayValues[1183].Loc != scm.LocNone {
			d1183 = ps.OverlayValues[1183]
		}
		if len(ps.OverlayValues) > 1184 && ps.OverlayValues[1184].Loc != scm.LocNone {
			d1184 = ps.OverlayValues[1184]
		}
		if len(ps.OverlayValues) > 1185 && ps.OverlayValues[1185].Loc != scm.LocNone {
			d1185 = ps.OverlayValues[1185]
		}
		if len(ps.OverlayValues) > 1186 && ps.OverlayValues[1186].Loc != scm.LocNone {
			d1186 = ps.OverlayValues[1186]
		}
		if len(ps.OverlayValues) > 1187 && ps.OverlayValues[1187].Loc != scm.LocNone {
			d1187 = ps.OverlayValues[1187]
		}
		if len(ps.OverlayValues) > 1188 && ps.OverlayValues[1188].Loc != scm.LocNone {
			d1188 = ps.OverlayValues[1188]
		}
		if len(ps.OverlayValues) > 1189 && ps.OverlayValues[1189].Loc != scm.LocNone {
			d1189 = ps.OverlayValues[1189]
		}
		if len(ps.OverlayValues) > 1190 && ps.OverlayValues[1190].Loc != scm.LocNone {
			d1190 = ps.OverlayValues[1190]
		}
		if len(ps.OverlayValues) > 1191 && ps.OverlayValues[1191].Loc != scm.LocNone {
			d1191 = ps.OverlayValues[1191]
		}
		if len(ps.OverlayValues) > 1192 && ps.OverlayValues[1192].Loc != scm.LocNone {
			d1192 = ps.OverlayValues[1192]
		}
		if len(ps.OverlayValues) > 1193 && ps.OverlayValues[1193].Loc != scm.LocNone {
			d1193 = ps.OverlayValues[1193]
		}
		if len(ps.OverlayValues) > 1194 && ps.OverlayValues[1194].Loc != scm.LocNone {
			d1194 = ps.OverlayValues[1194]
		}
		if len(ps.OverlayValues) > 1195 && ps.OverlayValues[1195].Loc != scm.LocNone {
			d1195 = ps.OverlayValues[1195]
		}
		if len(ps.OverlayValues) > 1196 && ps.OverlayValues[1196].Loc != scm.LocNone {
			d1196 = ps.OverlayValues[1196]
		}
		if len(ps.OverlayValues) > 1197 && ps.OverlayValues[1197].Loc != scm.LocNone {
			d1197 = ps.OverlayValues[1197]
		}
		if len(ps.OverlayValues) > 1198 && ps.OverlayValues[1198].Loc != scm.LocNone {
			d1198 = ps.OverlayValues[1198]
		}
		if len(ps.OverlayValues) > 1199 && ps.OverlayValues[1199].Loc != scm.LocNone {
			d1199 = ps.OverlayValues[1199]
		}
		if len(ps.OverlayValues) > 1200 && ps.OverlayValues[1200].Loc != scm.LocNone {
			d1200 = ps.OverlayValues[1200]
		}
		if len(ps.OverlayValues) > 1201 && ps.OverlayValues[1201].Loc != scm.LocNone {
			d1201 = ps.OverlayValues[1201]
		}
		if len(ps.OverlayValues) > 1202 && ps.OverlayValues[1202].Loc != scm.LocNone {
			d1202 = ps.OverlayValues[1202]
		}
		if len(ps.OverlayValues) > 1203 && ps.OverlayValues[1203].Loc != scm.LocNone {
			d1203 = ps.OverlayValues[1203]
		}
		if len(ps.OverlayValues) > 1204 && ps.OverlayValues[1204].Loc != scm.LocNone {
			d1204 = ps.OverlayValues[1204]
		}
		if len(ps.OverlayValues) > 1205 && ps.OverlayValues[1205].Loc != scm.LocNone {
			d1205 = ps.OverlayValues[1205]
		}
		if len(ps.OverlayValues) > 1206 && ps.OverlayValues[1206].Loc != scm.LocNone {
			d1206 = ps.OverlayValues[1206]
		}
		if len(ps.OverlayValues) > 1207 && ps.OverlayValues[1207].Loc != scm.LocNone {
			d1207 = ps.OverlayValues[1207]
		}
		if len(ps.OverlayValues) > 1208 && ps.OverlayValues[1208].Loc != scm.LocNone {
			d1208 = ps.OverlayValues[1208]
		}
		if len(ps.OverlayValues) > 1209 && ps.OverlayValues[1209].Loc != scm.LocNone {
			d1209 = ps.OverlayValues[1209]
		}
		if len(ps.OverlayValues) > 1210 && ps.OverlayValues[1210].Loc != scm.LocNone {
			d1210 = ps.OverlayValues[1210]
		}
		if len(ps.OverlayValues) > 1211 && ps.OverlayValues[1211].Loc != scm.LocNone {
			d1211 = ps.OverlayValues[1211]
		}
		if len(ps.OverlayValues) > 1212 && ps.OverlayValues[1212].Loc != scm.LocNone {
			d1212 = ps.OverlayValues[1212]
		}
		if len(ps.OverlayValues) > 1213 && ps.OverlayValues[1213].Loc != scm.LocNone {
			d1213 = ps.OverlayValues[1213]
		}
		if len(ps.OverlayValues) > 1214 && ps.OverlayValues[1214].Loc != scm.LocNone {
			d1214 = ps.OverlayValues[1214]
		}
		if len(ps.OverlayValues) > 1215 && ps.OverlayValues[1215].Loc != scm.LocNone {
			d1215 = ps.OverlayValues[1215]
		}
		if len(ps.OverlayValues) > 1216 && ps.OverlayValues[1216].Loc != scm.LocNone {
			d1216 = ps.OverlayValues[1216]
		}
		if len(ps.OverlayValues) > 1217 && ps.OverlayValues[1217].Loc != scm.LocNone {
			d1217 = ps.OverlayValues[1217]
		}
		if len(ps.OverlayValues) > 1218 && ps.OverlayValues[1218].Loc != scm.LocNone {
			d1218 = ps.OverlayValues[1218]
		}
		if len(ps.OverlayValues) > 1219 && ps.OverlayValues[1219].Loc != scm.LocNone {
			d1219 = ps.OverlayValues[1219]
		}
		if len(ps.OverlayValues) > 1220 && ps.OverlayValues[1220].Loc != scm.LocNone {
			d1220 = ps.OverlayValues[1220]
		}
		if len(ps.OverlayValues) > 1221 && ps.OverlayValues[1221].Loc != scm.LocNone {
			d1221 = ps.OverlayValues[1221]
		}
		if len(ps.OverlayValues) > 1222 && ps.OverlayValues[1222].Loc != scm.LocNone {
			d1222 = ps.OverlayValues[1222]
		}
		if len(ps.OverlayValues) > 1223 && ps.OverlayValues[1223].Loc != scm.LocNone {
			d1223 = ps.OverlayValues[1223]
		}
		if len(ps.OverlayValues) > 1224 && ps.OverlayValues[1224].Loc != scm.LocNone {
			d1224 = ps.OverlayValues[1224]
		}
		if len(ps.OverlayValues) > 1225 && ps.OverlayValues[1225].Loc != scm.LocNone {
			d1225 = ps.OverlayValues[1225]
		}
		if len(ps.OverlayValues) > 1226 && ps.OverlayValues[1226].Loc != scm.LocNone {
			d1226 = ps.OverlayValues[1226]
		}
		ctx.ReclaimUntrackedRegs()
		var d1227 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 88
			val := *(*uint64)(unsafe.Pointer(fieldAddr))
			d1227 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 88)
			r163 := ctx.AllocReg()
			ctx.EmitMovRegMem(r163, thisptr.Reg, off)
			d1227 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r163}
			ctx.BindReg(r163, &d1227)
		}
		ctx.EnsureDesc(&d1227)
		ctx.EnsureDesc(&d1227)
		var d1228 scm.JITValueDesc
		if d1227.Loc == scm.LocImm {
			d1228 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d1227.Imm.Int()))))}
		} else {
			r164 := ctx.AllocReg()
			ctx.EmitMovRegReg(r164, d1227.Reg)
			d1228 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r164}
			ctx.BindReg(r164, &d1228)
		}
		ctx.FreeDesc(&d1227)
		ctx.EnsureDesc(&d182)
		ctx.EnsureDesc(&d1228)
		ctx.EnsureDescsTogether(&d182, &d1228)
		var d1229 scm.JITValueDesc
		if d182.Loc == scm.LocImm && d1228.Loc == scm.LocImm {
			d1229 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d182.Imm.Int() == d1228.Imm.Int())}
		} else if d1228.Loc == scm.LocImm {
			r165 := ctx.AllocRegExcept(d182.Reg)
			if d1228.Imm.Int() >= -2147483648 && d1228.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d182.Reg, int32(d1228.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1228.Imm.Int()))
				ctx.EmitCmpInt64(d182.Reg, scm.RegR11)
			}
			d1229 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r165, Condition: scm.CondEqual}
			ctx.BindReg(r165, &d1229)
		} else if d182.Loc == scm.LocImm {
			r166 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d182.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d1228.Reg)
			d1229 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r166, Condition: scm.CondEqual}
			ctx.BindReg(r166, &d1229)
		} else {
			r167 := ctx.AllocRegExcept(d182.Reg)
			ctx.EmitCmpInt64(d182.Reg, d1228.Reg)
			d1229 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r167, Condition: scm.CondEqual}
			ctx.BindReg(r167, &d1229)
		}
		ctx.FreeDesc(&d1228)
		d1230 = d1229
		ctx.EnsureDesc(&d1230)
		if d1230.Loc != scm.LocImm && d1230.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d1230.Loc == scm.LocImm {
			if d1230.Imm.Bool() {
				if ps.General {
				}
				ps1231 := scm.PhiState{General: ps.General}
				ps1231.OverlayValues = make([]scm.JITValueDesc, 1231)
				ps1231.OverlayValues[5] = d5
				ps1231.OverlayValues[6] = d6
				ps1231.OverlayValues[7] = d7
				ps1231.OverlayValues[8] = d8
				ps1231.OverlayValues[9] = d9
				ps1231.OverlayValues[10] = d10
				ps1231.OverlayValues[11] = d11
				ps1231.OverlayValues[12] = d12
				ps1231.OverlayValues[13] = d13
				ps1231.OverlayValues[14] = d14
				ps1231.OverlayValues[15] = d15
				ps1231.OverlayValues[16] = d16
				ps1231.OverlayValues[17] = d17
				ps1231.OverlayValues[18] = d18
				ps1231.OverlayValues[19] = d19
				ps1231.OverlayValues[21] = d21
				ps1231.OverlayValues[22] = d22
				ps1231.OverlayValues[23] = d23
				ps1231.OverlayValues[24] = d24
				ps1231.OverlayValues[25] = d25
				ps1231.OverlayValues[26] = d26
				ps1231.OverlayValues[27] = d27
				ps1231.OverlayValues[28] = d28
				ps1231.OverlayValues[29] = d29
				ps1231.OverlayValues[30] = d30
				ps1231.OverlayValues[31] = d31
				ps1231.OverlayValues[32] = d32
				ps1231.OverlayValues[33] = d33
				ps1231.OverlayValues[34] = d34
				ps1231.OverlayValues[35] = d35
				ps1231.OverlayValues[36] = d36
				ps1231.OverlayValues[37] = d37
				ps1231.OverlayValues[38] = d38
				ps1231.OverlayValues[39] = d39
				ps1231.OverlayValues[40] = d40
				ps1231.OverlayValues[41] = d41
				ps1231.OverlayValues[42] = d42
				ps1231.OverlayValues[43] = d43
				ps1231.OverlayValues[44] = d44
				ps1231.OverlayValues[45] = d45
				ps1231.OverlayValues[46] = d46
				ps1231.OverlayValues[47] = d47
				ps1231.OverlayValues[48] = d48
				ps1231.OverlayValues[49] = d49
				ps1231.OverlayValues[50] = d50
				ps1231.OverlayValues[51] = d51
				ps1231.OverlayValues[54] = d54
				ps1231.OverlayValues[55] = d55
				ps1231.OverlayValues[56] = d56
				ps1231.OverlayValues[159] = d159
				ps1231.OverlayValues[160] = d160
				ps1231.OverlayValues[161] = d161
				ps1231.OverlayValues[162] = d162
				ps1231.OverlayValues[163] = d163
				ps1231.OverlayValues[164] = d164
				ps1231.OverlayValues[165] = d165
				ps1231.OverlayValues[166] = d166
				ps1231.OverlayValues[167] = d167
				ps1231.OverlayValues[168] = d168
				ps1231.OverlayValues[169] = d169
				ps1231.OverlayValues[170] = d170
				ps1231.OverlayValues[171] = d171
				ps1231.OverlayValues[172] = d172
				ps1231.OverlayValues[173] = d173
				ps1231.OverlayValues[174] = d174
				ps1231.OverlayValues[175] = d175
				ps1231.OverlayValues[176] = d176
				ps1231.OverlayValues[177] = d177
				ps1231.OverlayValues[178] = d178
				ps1231.OverlayValues[179] = d179
				ps1231.OverlayValues[180] = d180
				ps1231.OverlayValues[181] = d181
				ps1231.OverlayValues[182] = d182
				ps1231.OverlayValues[183] = d183
				ps1231.OverlayValues[184] = d184
				ps1231.OverlayValues[187] = d187
				ps1231.OverlayValues[344] = d344
				ps1231.OverlayValues[345] = d345
				ps1231.OverlayValues[346] = d346
				ps1231.OverlayValues[347] = d347
				ps1231.OverlayValues[349] = d349
				ps1231.OverlayValues[350] = d350
				ps1231.OverlayValues[351] = d351
				ps1231.OverlayValues[352] = d352
				ps1231.OverlayValues[353] = d353
				ps1231.OverlayValues[354] = d354
				ps1231.OverlayValues[355] = d355
				ps1231.OverlayValues[356] = d356
				ps1231.OverlayValues[358] = d358
				ps1231.OverlayValues[360] = d360
				ps1231.OverlayValues[361] = d361
				ps1231.OverlayValues[362] = d362
				ps1231.OverlayValues[456] = d456
				ps1231.OverlayValues[457] = d457
				ps1231.OverlayValues[460] = d460
				ps1231.OverlayValues[557] = d557
				ps1231.OverlayValues[558] = d558
				ps1231.OverlayValues[559] = d559
				ps1231.OverlayValues[560] = d560
				ps1231.OverlayValues[561] = d561
				ps1231.OverlayValues[563] = d563
				ps1231.OverlayValues[564] = d564
				ps1231.OverlayValues[565] = d565
				ps1231.OverlayValues[566] = d566
				ps1231.OverlayValues[567] = d567
				ps1231.OverlayValues[568] = d568
				ps1231.OverlayValues[569] = d569
				ps1231.OverlayValues[570] = d570
				ps1231.OverlayValues[571] = d571
				ps1231.OverlayValues[572] = d572
				ps1231.OverlayValues[573] = d573
				ps1231.OverlayValues[574] = d574
				ps1231.OverlayValues[575] = d575
				ps1231.OverlayValues[576] = d576
				ps1231.OverlayValues[577] = d577
				ps1231.OverlayValues[578] = d578
				ps1231.OverlayValues[579] = d579
				ps1231.OverlayValues[580] = d580
				ps1231.OverlayValues[581] = d581
				ps1231.OverlayValues[582] = d582
				ps1231.OverlayValues[583] = d583
				ps1231.OverlayValues[584] = d584
				ps1231.OverlayValues[585] = d585
				ps1231.OverlayValues[586] = d586
				ps1231.OverlayValues[587] = d587
				ps1231.OverlayValues[588] = d588
				ps1231.OverlayValues[589] = d589
				ps1231.OverlayValues[850] = d850
				ps1231.OverlayValues[851] = d851
				ps1231.OverlayValues[852] = d852
				ps1231.OverlayValues[854] = d854
				ps1231.OverlayValues[855] = d855
				ps1231.OverlayValues[856] = d856
				ps1231.OverlayValues[857] = d857
				ps1231.OverlayValues[858] = d858
				ps1231.OverlayValues[859] = d859
				ps1231.OverlayValues[860] = d860
				ps1231.OverlayValues[862] = d862
				ps1231.OverlayValues[864] = d864
				ps1231.OverlayValues[865] = d865
				ps1231.OverlayValues[1007] = d1007
				ps1231.OverlayValues[1008] = d1008
				ps1231.OverlayValues[1011] = d1011
				ps1231.OverlayValues[1156] = d1156
				ps1231.OverlayValues[1157] = d1157
				ps1231.OverlayValues[1158] = d1158
				ps1231.OverlayValues[1159] = d1159
				ps1231.OverlayValues[1161] = d1161
				ps1231.OverlayValues[1162] = d1162
				ps1231.OverlayValues[1163] = d1163
				ps1231.OverlayValues[1164] = d1164
				ps1231.OverlayValues[1165] = d1165
				ps1231.OverlayValues[1166] = d1166
				ps1231.OverlayValues[1167] = d1167
				ps1231.OverlayValues[1168] = d1168
				ps1231.OverlayValues[1169] = d1169
				ps1231.OverlayValues[1170] = d1170
				ps1231.OverlayValues[1172] = d1172
				ps1231.OverlayValues[1173] = d1173
				ps1231.OverlayValues[1174] = d1174
				ps1231.OverlayValues[1175] = d1175
				ps1231.OverlayValues[1176] = d1176
				ps1231.OverlayValues[1177] = d1177
				ps1231.OverlayValues[1178] = d1178
				ps1231.OverlayValues[1179] = d1179
				ps1231.OverlayValues[1180] = d1180
				ps1231.OverlayValues[1181] = d1181
				ps1231.OverlayValues[1182] = d1182
				ps1231.OverlayValues[1183] = d1183
				ps1231.OverlayValues[1184] = d1184
				ps1231.OverlayValues[1185] = d1185
				ps1231.OverlayValues[1186] = d1186
				ps1231.OverlayValues[1187] = d1187
				ps1231.OverlayValues[1188] = d1188
				ps1231.OverlayValues[1189] = d1189
				ps1231.OverlayValues[1190] = d1190
				ps1231.OverlayValues[1191] = d1191
				ps1231.OverlayValues[1192] = d1192
				ps1231.OverlayValues[1193] = d1193
				ps1231.OverlayValues[1194] = d1194
				ps1231.OverlayValues[1195] = d1195
				ps1231.OverlayValues[1196] = d1196
				ps1231.OverlayValues[1197] = d1197
				ps1231.OverlayValues[1198] = d1198
				ps1231.OverlayValues[1199] = d1199
				ps1231.OverlayValues[1200] = d1200
				ps1231.OverlayValues[1201] = d1201
				ps1231.OverlayValues[1202] = d1202
				ps1231.OverlayValues[1203] = d1203
				ps1231.OverlayValues[1204] = d1204
				ps1231.OverlayValues[1205] = d1205
				ps1231.OverlayValues[1206] = d1206
				ps1231.OverlayValues[1207] = d1207
				ps1231.OverlayValues[1208] = d1208
				ps1231.OverlayValues[1209] = d1209
				ps1231.OverlayValues[1210] = d1210
				ps1231.OverlayValues[1211] = d1211
				ps1231.OverlayValues[1212] = d1212
				ps1231.OverlayValues[1213] = d1213
				ps1231.OverlayValues[1214] = d1214
				ps1231.OverlayValues[1215] = d1215
				ps1231.OverlayValues[1216] = d1216
				ps1231.OverlayValues[1217] = d1217
				ps1231.OverlayValues[1218] = d1218
				ps1231.OverlayValues[1219] = d1219
				ps1231.OverlayValues[1220] = d1220
				ps1231.OverlayValues[1221] = d1221
				ps1231.OverlayValues[1222] = d1222
				ps1231.OverlayValues[1223] = d1223
				ps1231.OverlayValues[1224] = d1224
				ps1231.OverlayValues[1225] = d1225
				ps1231.OverlayValues[1226] = d1226
				ps1231.OverlayValues[1227] = d1227
				ps1231.OverlayValues[1228] = d1228
				ps1231.OverlayValues[1229] = d1229
				ps1231.OverlayValues[1230] = d1230
				return bbs[11].RenderPS(ps1231)
			}
			if ps.General {
			}
			ps1232 := scm.PhiState{General: ps.General}
			ps1232.OverlayValues = make([]scm.JITValueDesc, 1231)
			ps1232.OverlayValues[5] = d5
			ps1232.OverlayValues[6] = d6
			ps1232.OverlayValues[7] = d7
			ps1232.OverlayValues[8] = d8
			ps1232.OverlayValues[9] = d9
			ps1232.OverlayValues[10] = d10
			ps1232.OverlayValues[11] = d11
			ps1232.OverlayValues[12] = d12
			ps1232.OverlayValues[13] = d13
			ps1232.OverlayValues[14] = d14
			ps1232.OverlayValues[15] = d15
			ps1232.OverlayValues[16] = d16
			ps1232.OverlayValues[17] = d17
			ps1232.OverlayValues[18] = d18
			ps1232.OverlayValues[19] = d19
			ps1232.OverlayValues[21] = d21
			ps1232.OverlayValues[22] = d22
			ps1232.OverlayValues[23] = d23
			ps1232.OverlayValues[24] = d24
			ps1232.OverlayValues[25] = d25
			ps1232.OverlayValues[26] = d26
			ps1232.OverlayValues[27] = d27
			ps1232.OverlayValues[28] = d28
			ps1232.OverlayValues[29] = d29
			ps1232.OverlayValues[30] = d30
			ps1232.OverlayValues[31] = d31
			ps1232.OverlayValues[32] = d32
			ps1232.OverlayValues[33] = d33
			ps1232.OverlayValues[34] = d34
			ps1232.OverlayValues[35] = d35
			ps1232.OverlayValues[36] = d36
			ps1232.OverlayValues[37] = d37
			ps1232.OverlayValues[38] = d38
			ps1232.OverlayValues[39] = d39
			ps1232.OverlayValues[40] = d40
			ps1232.OverlayValues[41] = d41
			ps1232.OverlayValues[42] = d42
			ps1232.OverlayValues[43] = d43
			ps1232.OverlayValues[44] = d44
			ps1232.OverlayValues[45] = d45
			ps1232.OverlayValues[46] = d46
			ps1232.OverlayValues[47] = d47
			ps1232.OverlayValues[48] = d48
			ps1232.OverlayValues[49] = d49
			ps1232.OverlayValues[50] = d50
			ps1232.OverlayValues[51] = d51
			ps1232.OverlayValues[54] = d54
			ps1232.OverlayValues[55] = d55
			ps1232.OverlayValues[56] = d56
			ps1232.OverlayValues[159] = d159
			ps1232.OverlayValues[160] = d160
			ps1232.OverlayValues[161] = d161
			ps1232.OverlayValues[162] = d162
			ps1232.OverlayValues[163] = d163
			ps1232.OverlayValues[164] = d164
			ps1232.OverlayValues[165] = d165
			ps1232.OverlayValues[166] = d166
			ps1232.OverlayValues[167] = d167
			ps1232.OverlayValues[168] = d168
			ps1232.OverlayValues[169] = d169
			ps1232.OverlayValues[170] = d170
			ps1232.OverlayValues[171] = d171
			ps1232.OverlayValues[172] = d172
			ps1232.OverlayValues[173] = d173
			ps1232.OverlayValues[174] = d174
			ps1232.OverlayValues[175] = d175
			ps1232.OverlayValues[176] = d176
			ps1232.OverlayValues[177] = d177
			ps1232.OverlayValues[178] = d178
			ps1232.OverlayValues[179] = d179
			ps1232.OverlayValues[180] = d180
			ps1232.OverlayValues[181] = d181
			ps1232.OverlayValues[182] = d182
			ps1232.OverlayValues[183] = d183
			ps1232.OverlayValues[184] = d184
			ps1232.OverlayValues[187] = d187
			ps1232.OverlayValues[344] = d344
			ps1232.OverlayValues[345] = d345
			ps1232.OverlayValues[346] = d346
			ps1232.OverlayValues[347] = d347
			ps1232.OverlayValues[349] = d349
			ps1232.OverlayValues[350] = d350
			ps1232.OverlayValues[351] = d351
			ps1232.OverlayValues[352] = d352
			ps1232.OverlayValues[353] = d353
			ps1232.OverlayValues[354] = d354
			ps1232.OverlayValues[355] = d355
			ps1232.OverlayValues[356] = d356
			ps1232.OverlayValues[358] = d358
			ps1232.OverlayValues[360] = d360
			ps1232.OverlayValues[361] = d361
			ps1232.OverlayValues[362] = d362
			ps1232.OverlayValues[456] = d456
			ps1232.OverlayValues[457] = d457
			ps1232.OverlayValues[460] = d460
			ps1232.OverlayValues[557] = d557
			ps1232.OverlayValues[558] = d558
			ps1232.OverlayValues[559] = d559
			ps1232.OverlayValues[560] = d560
			ps1232.OverlayValues[561] = d561
			ps1232.OverlayValues[563] = d563
			ps1232.OverlayValues[564] = d564
			ps1232.OverlayValues[565] = d565
			ps1232.OverlayValues[566] = d566
			ps1232.OverlayValues[567] = d567
			ps1232.OverlayValues[568] = d568
			ps1232.OverlayValues[569] = d569
			ps1232.OverlayValues[570] = d570
			ps1232.OverlayValues[571] = d571
			ps1232.OverlayValues[572] = d572
			ps1232.OverlayValues[573] = d573
			ps1232.OverlayValues[574] = d574
			ps1232.OverlayValues[575] = d575
			ps1232.OverlayValues[576] = d576
			ps1232.OverlayValues[577] = d577
			ps1232.OverlayValues[578] = d578
			ps1232.OverlayValues[579] = d579
			ps1232.OverlayValues[580] = d580
			ps1232.OverlayValues[581] = d581
			ps1232.OverlayValues[582] = d582
			ps1232.OverlayValues[583] = d583
			ps1232.OverlayValues[584] = d584
			ps1232.OverlayValues[585] = d585
			ps1232.OverlayValues[586] = d586
			ps1232.OverlayValues[587] = d587
			ps1232.OverlayValues[588] = d588
			ps1232.OverlayValues[589] = d589
			ps1232.OverlayValues[850] = d850
			ps1232.OverlayValues[851] = d851
			ps1232.OverlayValues[852] = d852
			ps1232.OverlayValues[854] = d854
			ps1232.OverlayValues[855] = d855
			ps1232.OverlayValues[856] = d856
			ps1232.OverlayValues[857] = d857
			ps1232.OverlayValues[858] = d858
			ps1232.OverlayValues[859] = d859
			ps1232.OverlayValues[860] = d860
			ps1232.OverlayValues[862] = d862
			ps1232.OverlayValues[864] = d864
			ps1232.OverlayValues[865] = d865
			ps1232.OverlayValues[1007] = d1007
			ps1232.OverlayValues[1008] = d1008
			ps1232.OverlayValues[1011] = d1011
			ps1232.OverlayValues[1156] = d1156
			ps1232.OverlayValues[1157] = d1157
			ps1232.OverlayValues[1158] = d1158
			ps1232.OverlayValues[1159] = d1159
			ps1232.OverlayValues[1161] = d1161
			ps1232.OverlayValues[1162] = d1162
			ps1232.OverlayValues[1163] = d1163
			ps1232.OverlayValues[1164] = d1164
			ps1232.OverlayValues[1165] = d1165
			ps1232.OverlayValues[1166] = d1166
			ps1232.OverlayValues[1167] = d1167
			ps1232.OverlayValues[1168] = d1168
			ps1232.OverlayValues[1169] = d1169
			ps1232.OverlayValues[1170] = d1170
			ps1232.OverlayValues[1172] = d1172
			ps1232.OverlayValues[1173] = d1173
			ps1232.OverlayValues[1174] = d1174
			ps1232.OverlayValues[1175] = d1175
			ps1232.OverlayValues[1176] = d1176
			ps1232.OverlayValues[1177] = d1177
			ps1232.OverlayValues[1178] = d1178
			ps1232.OverlayValues[1179] = d1179
			ps1232.OverlayValues[1180] = d1180
			ps1232.OverlayValues[1181] = d1181
			ps1232.OverlayValues[1182] = d1182
			ps1232.OverlayValues[1183] = d1183
			ps1232.OverlayValues[1184] = d1184
			ps1232.OverlayValues[1185] = d1185
			ps1232.OverlayValues[1186] = d1186
			ps1232.OverlayValues[1187] = d1187
			ps1232.OverlayValues[1188] = d1188
			ps1232.OverlayValues[1189] = d1189
			ps1232.OverlayValues[1190] = d1190
			ps1232.OverlayValues[1191] = d1191
			ps1232.OverlayValues[1192] = d1192
			ps1232.OverlayValues[1193] = d1193
			ps1232.OverlayValues[1194] = d1194
			ps1232.OverlayValues[1195] = d1195
			ps1232.OverlayValues[1196] = d1196
			ps1232.OverlayValues[1197] = d1197
			ps1232.OverlayValues[1198] = d1198
			ps1232.OverlayValues[1199] = d1199
			ps1232.OverlayValues[1200] = d1200
			ps1232.OverlayValues[1201] = d1201
			ps1232.OverlayValues[1202] = d1202
			ps1232.OverlayValues[1203] = d1203
			ps1232.OverlayValues[1204] = d1204
			ps1232.OverlayValues[1205] = d1205
			ps1232.OverlayValues[1206] = d1206
			ps1232.OverlayValues[1207] = d1207
			ps1232.OverlayValues[1208] = d1208
			ps1232.OverlayValues[1209] = d1209
			ps1232.OverlayValues[1210] = d1210
			ps1232.OverlayValues[1211] = d1211
			ps1232.OverlayValues[1212] = d1212
			ps1232.OverlayValues[1213] = d1213
			ps1232.OverlayValues[1214] = d1214
			ps1232.OverlayValues[1215] = d1215
			ps1232.OverlayValues[1216] = d1216
			ps1232.OverlayValues[1217] = d1217
			ps1232.OverlayValues[1218] = d1218
			ps1232.OverlayValues[1219] = d1219
			ps1232.OverlayValues[1220] = d1220
			ps1232.OverlayValues[1221] = d1221
			ps1232.OverlayValues[1222] = d1222
			ps1232.OverlayValues[1223] = d1223
			ps1232.OverlayValues[1224] = d1224
			ps1232.OverlayValues[1225] = d1225
			ps1232.OverlayValues[1226] = d1226
			ps1232.OverlayValues[1227] = d1227
			ps1232.OverlayValues[1228] = d1228
			ps1232.OverlayValues[1229] = d1229
			ps1232.OverlayValues[1230] = d1230
			return bbs[12].RenderPS(ps1232)
		}
		if !ps.General {
			ps.General = true
			return bbs[13].RenderPS(ps)
		}
		ctx.EmitJump(d1230.Condition, lbl12)
		ctx.FreeDesc(&d1229)
		snap1233 := d5
		snap1234 := d6
		snap1235 := d7
		snap1236 := d8
		snap1237 := d9
		snap1238 := d10
		snap1239 := d11
		snap1240 := d12
		snap1241 := d13
		snap1242 := d14
		snap1243 := d15
		snap1244 := d16
		snap1245 := d17
		snap1246 := d18
		snap1247 := d19
		snap1248 := d21
		snap1249 := d22
		snap1250 := d23
		snap1251 := d24
		snap1252 := d25
		snap1253 := d26
		snap1254 := d27
		snap1255 := d28
		snap1256 := d29
		snap1257 := d30
		snap1258 := d31
		snap1259 := d32
		snap1260 := d33
		snap1261 := d34
		snap1262 := d35
		snap1263 := d36
		snap1264 := d37
		snap1265 := d38
		snap1266 := d39
		snap1267 := d40
		snap1268 := d41
		snap1269 := d42
		snap1270 := d43
		snap1271 := d44
		snap1272 := d45
		snap1273 := d46
		snap1274 := d47
		snap1275 := d48
		snap1276 := d49
		snap1277 := d50
		snap1278 := d51
		snap1279 := d54
		snap1280 := d55
		snap1281 := d56
		snap1282 := d159
		snap1283 := d160
		snap1284 := d161
		snap1285 := d162
		snap1286 := d163
		snap1287 := d164
		snap1288 := d165
		snap1289 := d166
		snap1290 := d167
		snap1291 := d168
		snap1292 := d169
		snap1293 := d170
		snap1294 := d171
		snap1295 := d172
		snap1296 := d173
		snap1297 := d174
		snap1298 := d175
		snap1299 := d176
		snap1300 := d177
		snap1301 := d178
		snap1302 := d179
		snap1303 := d180
		snap1304 := d181
		snap1305 := d182
		snap1306 := d183
		snap1307 := d184
		snap1308 := d187
		snap1309 := d344
		snap1310 := d345
		snap1311 := d346
		snap1312 := d347
		snap1313 := d349
		snap1314 := d350
		snap1315 := d351
		snap1316 := d352
		snap1317 := d353
		snap1318 := d354
		snap1319 := d355
		snap1320 := d356
		snap1321 := d358
		snap1322 := d360
		snap1323 := d361
		snap1324 := d362
		snap1325 := d456
		snap1326 := d457
		snap1327 := d460
		snap1328 := d557
		snap1329 := d558
		snap1330 := d559
		snap1331 := d560
		snap1332 := d561
		snap1333 := d563
		snap1334 := d564
		snap1335 := d565
		snap1336 := d566
		snap1337 := d567
		snap1338 := d568
		snap1339 := d569
		snap1340 := d570
		snap1341 := d571
		snap1342 := d572
		snap1343 := d573
		snap1344 := d574
		snap1345 := d575
		snap1346 := d576
		snap1347 := d577
		snap1348 := d578
		snap1349 := d579
		snap1350 := d580
		snap1351 := d581
		snap1352 := d582
		snap1353 := d583
		snap1354 := d584
		snap1355 := d585
		snap1356 := d586
		snap1357 := d587
		snap1358 := d588
		snap1359 := d589
		snap1360 := d850
		snap1361 := d851
		snap1362 := d852
		snap1363 := d854
		snap1364 := d855
		snap1365 := d856
		snap1366 := d857
		snap1367 := d858
		snap1368 := d859
		snap1369 := d860
		snap1370 := d862
		snap1371 := d864
		snap1372 := d865
		snap1373 := d1007
		snap1374 := d1008
		snap1375 := d1011
		snap1376 := d1156
		snap1377 := d1157
		snap1378 := d1158
		snap1379 := d1159
		snap1380 := d1161
		snap1381 := d1162
		snap1382 := d1163
		snap1383 := d1164
		snap1384 := d1165
		snap1385 := d1166
		snap1386 := d1167
		snap1387 := d1168
		snap1388 := d1169
		snap1389 := d1170
		snap1390 := d1172
		snap1391 := d1173
		snap1392 := d1174
		snap1393 := d1175
		snap1394 := d1176
		snap1395 := d1177
		snap1396 := d1178
		snap1397 := d1179
		snap1398 := d1180
		snap1399 := d1181
		snap1400 := d1182
		snap1401 := d1183
		snap1402 := d1184
		snap1403 := d1185
		snap1404 := d1186
		snap1405 := d1187
		snap1406 := d1188
		snap1407 := d1189
		snap1408 := d1190
		snap1409 := d1191
		snap1410 := d1192
		snap1411 := d1193
		snap1412 := d1194
		snap1413 := d1195
		snap1414 := d1196
		snap1415 := d1197
		snap1416 := d1198
		snap1417 := d1199
		snap1418 := d1200
		snap1419 := d1201
		snap1420 := d1202
		snap1421 := d1203
		snap1422 := d1204
		snap1423 := d1205
		snap1424 := d1206
		snap1425 := d1207
		snap1426 := d1208
		snap1427 := d1209
		snap1428 := d1210
		snap1429 := d1211
		snap1430 := d1212
		snap1431 := d1213
		snap1432 := d1214
		snap1433 := d1215
		snap1434 := d1216
		snap1435 := d1217
		snap1436 := d1218
		snap1437 := d1219
		snap1438 := d1220
		snap1439 := d1221
		snap1440 := d1222
		snap1441 := d1223
		snap1442 := d1224
		snap1443 := d1225
		snap1444 := d1226
		snap1445 := d1227
		snap1446 := d1228
		snap1447 := d1229
		snap1448 := d1230
		alloc1449 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc1449)
		d5 = snap1233
		d6 = snap1234
		d7 = snap1235
		d8 = snap1236
		d9 = snap1237
		d10 = snap1238
		d11 = snap1239
		d12 = snap1240
		d13 = snap1241
		d14 = snap1242
		d15 = snap1243
		d16 = snap1244
		d17 = snap1245
		d18 = snap1246
		d19 = snap1247
		d21 = snap1248
		d22 = snap1249
		d23 = snap1250
		d24 = snap1251
		d25 = snap1252
		d26 = snap1253
		d27 = snap1254
		d28 = snap1255
		d29 = snap1256
		d30 = snap1257
		d31 = snap1258
		d32 = snap1259
		d33 = snap1260
		d34 = snap1261
		d35 = snap1262
		d36 = snap1263
		d37 = snap1264
		d38 = snap1265
		d39 = snap1266
		d40 = snap1267
		d41 = snap1268
		d42 = snap1269
		d43 = snap1270
		d44 = snap1271
		d45 = snap1272
		d46 = snap1273
		d47 = snap1274
		d48 = snap1275
		d49 = snap1276
		d50 = snap1277
		d51 = snap1278
		d54 = snap1279
		d55 = snap1280
		d56 = snap1281
		d159 = snap1282
		d160 = snap1283
		d161 = snap1284
		d162 = snap1285
		d163 = snap1286
		d164 = snap1287
		d165 = snap1288
		d166 = snap1289
		d167 = snap1290
		d168 = snap1291
		d169 = snap1292
		d170 = snap1293
		d171 = snap1294
		d172 = snap1295
		d173 = snap1296
		d174 = snap1297
		d175 = snap1298
		d176 = snap1299
		d177 = snap1300
		d178 = snap1301
		d179 = snap1302
		d180 = snap1303
		d181 = snap1304
		d182 = snap1305
		d183 = snap1306
		d184 = snap1307
		d187 = snap1308
		d344 = snap1309
		d345 = snap1310
		d346 = snap1311
		d347 = snap1312
		d349 = snap1313
		d350 = snap1314
		d351 = snap1315
		d352 = snap1316
		d353 = snap1317
		d354 = snap1318
		d355 = snap1319
		d356 = snap1320
		d358 = snap1321
		d360 = snap1322
		d361 = snap1323
		d362 = snap1324
		d456 = snap1325
		d457 = snap1326
		d460 = snap1327
		d557 = snap1328
		d558 = snap1329
		d559 = snap1330
		d560 = snap1331
		d561 = snap1332
		d563 = snap1333
		d564 = snap1334
		d565 = snap1335
		d566 = snap1336
		d567 = snap1337
		d568 = snap1338
		d569 = snap1339
		d570 = snap1340
		d571 = snap1341
		d572 = snap1342
		d573 = snap1343
		d574 = snap1344
		d575 = snap1345
		d576 = snap1346
		d577 = snap1347
		d578 = snap1348
		d579 = snap1349
		d580 = snap1350
		d581 = snap1351
		d582 = snap1352
		d583 = snap1353
		d584 = snap1354
		d585 = snap1355
		d586 = snap1356
		d587 = snap1357
		d588 = snap1358
		d589 = snap1359
		d850 = snap1360
		d851 = snap1361
		d852 = snap1362
		d854 = snap1363
		d855 = snap1364
		d856 = snap1365
		d857 = snap1366
		d858 = snap1367
		d859 = snap1368
		d860 = snap1369
		d862 = snap1370
		d864 = snap1371
		d865 = snap1372
		d1007 = snap1373
		d1008 = snap1374
		d1011 = snap1375
		d1156 = snap1376
		d1157 = snap1377
		d1158 = snap1378
		d1159 = snap1379
		d1161 = snap1380
		d1162 = snap1381
		d1163 = snap1382
		d1164 = snap1383
		d1165 = snap1384
		d1166 = snap1385
		d1167 = snap1386
		d1168 = snap1387
		d1169 = snap1388
		d1170 = snap1389
		d1172 = snap1390
		d1173 = snap1391
		d1174 = snap1392
		d1175 = snap1393
		d1176 = snap1394
		d1177 = snap1395
		d1178 = snap1396
		d1179 = snap1397
		d1180 = snap1398
		d1181 = snap1399
		d1182 = snap1400
		d1183 = snap1401
		d1184 = snap1402
		d1185 = snap1403
		d1186 = snap1404
		d1187 = snap1405
		d1188 = snap1406
		d1189 = snap1407
		d1190 = snap1408
		d1191 = snap1409
		d1192 = snap1410
		d1193 = snap1411
		d1194 = snap1412
		d1195 = snap1413
		d1196 = snap1414
		d1197 = snap1415
		d1198 = snap1416
		d1199 = snap1417
		d1200 = snap1418
		d1201 = snap1419
		d1202 = snap1420
		d1203 = snap1421
		d1204 = snap1422
		d1205 = snap1423
		d1206 = snap1424
		d1207 = snap1425
		d1208 = snap1426
		d1209 = snap1427
		d1210 = snap1428
		d1211 = snap1429
		d1212 = snap1430
		d1213 = snap1431
		d1214 = snap1432
		d1215 = snap1433
		d1216 = snap1434
		d1217 = snap1435
		d1218 = snap1436
		d1219 = snap1437
		d1220 = snap1438
		d1221 = snap1439
		d1222 = snap1440
		d1223 = snap1441
		d1224 = snap1442
		d1225 = snap1443
		d1226 = snap1444
		d1227 = snap1445
		d1228 = snap1446
		d1229 = snap1447
		d1230 = snap1448
		ctx.RestoreAllocState(alloc1449)
		d5 = snap1233
		d6 = snap1234
		d7 = snap1235
		d8 = snap1236
		d9 = snap1237
		d10 = snap1238
		d11 = snap1239
		d12 = snap1240
		d13 = snap1241
		d14 = snap1242
		d15 = snap1243
		d16 = snap1244
		d17 = snap1245
		d18 = snap1246
		d19 = snap1247
		d21 = snap1248
		d22 = snap1249
		d23 = snap1250
		d24 = snap1251
		d25 = snap1252
		d26 = snap1253
		d27 = snap1254
		d28 = snap1255
		d29 = snap1256
		d30 = snap1257
		d31 = snap1258
		d32 = snap1259
		d33 = snap1260
		d34 = snap1261
		d35 = snap1262
		d36 = snap1263
		d37 = snap1264
		d38 = snap1265
		d39 = snap1266
		d40 = snap1267
		d41 = snap1268
		d42 = snap1269
		d43 = snap1270
		d44 = snap1271
		d45 = snap1272
		d46 = snap1273
		d47 = snap1274
		d48 = snap1275
		d49 = snap1276
		d50 = snap1277
		d51 = snap1278
		d54 = snap1279
		d55 = snap1280
		d56 = snap1281
		d159 = snap1282
		d160 = snap1283
		d161 = snap1284
		d162 = snap1285
		d163 = snap1286
		d164 = snap1287
		d165 = snap1288
		d166 = snap1289
		d167 = snap1290
		d168 = snap1291
		d169 = snap1292
		d170 = snap1293
		d171 = snap1294
		d172 = snap1295
		d173 = snap1296
		d174 = snap1297
		d175 = snap1298
		d176 = snap1299
		d177 = snap1300
		d178 = snap1301
		d179 = snap1302
		d180 = snap1303
		d181 = snap1304
		d182 = snap1305
		d183 = snap1306
		d184 = snap1307
		d187 = snap1308
		d344 = snap1309
		d345 = snap1310
		d346 = snap1311
		d347 = snap1312
		d349 = snap1313
		d350 = snap1314
		d351 = snap1315
		d352 = snap1316
		d353 = snap1317
		d354 = snap1318
		d355 = snap1319
		d356 = snap1320
		d358 = snap1321
		d360 = snap1322
		d361 = snap1323
		d362 = snap1324
		d456 = snap1325
		d457 = snap1326
		d460 = snap1327
		d557 = snap1328
		d558 = snap1329
		d559 = snap1330
		d560 = snap1331
		d561 = snap1332
		d563 = snap1333
		d564 = snap1334
		d565 = snap1335
		d566 = snap1336
		d567 = snap1337
		d568 = snap1338
		d569 = snap1339
		d570 = snap1340
		d571 = snap1341
		d572 = snap1342
		d573 = snap1343
		d574 = snap1344
		d575 = snap1345
		d576 = snap1346
		d577 = snap1347
		d578 = snap1348
		d579 = snap1349
		d580 = snap1350
		d581 = snap1351
		d582 = snap1352
		d583 = snap1353
		d584 = snap1354
		d585 = snap1355
		d586 = snap1356
		d587 = snap1357
		d588 = snap1358
		d589 = snap1359
		d850 = snap1360
		d851 = snap1361
		d852 = snap1362
		d854 = snap1363
		d855 = snap1364
		d856 = snap1365
		d857 = snap1366
		d858 = snap1367
		d859 = snap1368
		d860 = snap1369
		d862 = snap1370
		d864 = snap1371
		d865 = snap1372
		d1007 = snap1373
		d1008 = snap1374
		d1011 = snap1375
		d1156 = snap1376
		d1157 = snap1377
		d1158 = snap1378
		d1159 = snap1379
		d1161 = snap1380
		d1162 = snap1381
		d1163 = snap1382
		d1164 = snap1383
		d1165 = snap1384
		d1166 = snap1385
		d1167 = snap1386
		d1168 = snap1387
		d1169 = snap1388
		d1170 = snap1389
		d1172 = snap1390
		d1173 = snap1391
		d1174 = snap1392
		d1175 = snap1393
		d1176 = snap1394
		d1177 = snap1395
		d1178 = snap1396
		d1179 = snap1397
		d1180 = snap1398
		d1181 = snap1399
		d1182 = snap1400
		d1183 = snap1401
		d1184 = snap1402
		d1185 = snap1403
		d1186 = snap1404
		d1187 = snap1405
		d1188 = snap1406
		d1189 = snap1407
		d1190 = snap1408
		d1191 = snap1409
		d1192 = snap1410
		d1193 = snap1411
		d1194 = snap1412
		d1195 = snap1413
		d1196 = snap1414
		d1197 = snap1415
		d1198 = snap1416
		d1199 = snap1417
		d1200 = snap1418
		d1201 = snap1419
		d1202 = snap1420
		d1203 = snap1421
		d1204 = snap1422
		d1205 = snap1423
		d1206 = snap1424
		d1207 = snap1425
		d1208 = snap1426
		d1209 = snap1427
		d1210 = snap1428
		d1211 = snap1429
		d1212 = snap1430
		d1213 = snap1431
		d1214 = snap1432
		d1215 = snap1433
		d1216 = snap1434
		d1217 = snap1435
		d1218 = snap1436
		d1219 = snap1437
		d1220 = snap1438
		d1221 = snap1439
		d1222 = snap1440
		d1223 = snap1441
		d1224 = snap1442
		d1225 = snap1443
		d1226 = snap1444
		d1227 = snap1445
		d1228 = snap1446
		d1229 = snap1447
		d1230 = snap1448
		ps1450 := scm.PhiState{General: true}
		ps1450.OverlayValues = make([]scm.JITValueDesc, 1231)
		ps1450.OverlayValues[5] = d5
		ps1450.OverlayValues[6] = d6
		ps1450.OverlayValues[7] = d7
		ps1450.OverlayValues[8] = d8
		ps1450.OverlayValues[9] = d9
		ps1450.OverlayValues[10] = d10
		ps1450.OverlayValues[11] = d11
		ps1450.OverlayValues[12] = d12
		ps1450.OverlayValues[13] = d13
		ps1450.OverlayValues[14] = d14
		ps1450.OverlayValues[15] = d15
		ps1450.OverlayValues[16] = d16
		ps1450.OverlayValues[17] = d17
		ps1450.OverlayValues[18] = d18
		ps1450.OverlayValues[19] = d19
		ps1450.OverlayValues[21] = d21
		ps1450.OverlayValues[22] = d22
		ps1450.OverlayValues[23] = d23
		ps1450.OverlayValues[24] = d24
		ps1450.OverlayValues[25] = d25
		ps1450.OverlayValues[26] = d26
		ps1450.OverlayValues[27] = d27
		ps1450.OverlayValues[28] = d28
		ps1450.OverlayValues[29] = d29
		ps1450.OverlayValues[30] = d30
		ps1450.OverlayValues[31] = d31
		ps1450.OverlayValues[32] = d32
		ps1450.OverlayValues[33] = d33
		ps1450.OverlayValues[34] = d34
		ps1450.OverlayValues[35] = d35
		ps1450.OverlayValues[36] = d36
		ps1450.OverlayValues[37] = d37
		ps1450.OverlayValues[38] = d38
		ps1450.OverlayValues[39] = d39
		ps1450.OverlayValues[40] = d40
		ps1450.OverlayValues[41] = d41
		ps1450.OverlayValues[42] = d42
		ps1450.OverlayValues[43] = d43
		ps1450.OverlayValues[44] = d44
		ps1450.OverlayValues[45] = d45
		ps1450.OverlayValues[46] = d46
		ps1450.OverlayValues[47] = d47
		ps1450.OverlayValues[48] = d48
		ps1450.OverlayValues[49] = d49
		ps1450.OverlayValues[50] = d50
		ps1450.OverlayValues[51] = d51
		ps1450.OverlayValues[54] = d54
		ps1450.OverlayValues[55] = d55
		ps1450.OverlayValues[56] = d56
		ps1450.OverlayValues[159] = d159
		ps1450.OverlayValues[160] = d160
		ps1450.OverlayValues[161] = d161
		ps1450.OverlayValues[162] = d162
		ps1450.OverlayValues[163] = d163
		ps1450.OverlayValues[164] = d164
		ps1450.OverlayValues[165] = d165
		ps1450.OverlayValues[166] = d166
		ps1450.OverlayValues[167] = d167
		ps1450.OverlayValues[168] = d168
		ps1450.OverlayValues[169] = d169
		ps1450.OverlayValues[170] = d170
		ps1450.OverlayValues[171] = d171
		ps1450.OverlayValues[172] = d172
		ps1450.OverlayValues[173] = d173
		ps1450.OverlayValues[174] = d174
		ps1450.OverlayValues[175] = d175
		ps1450.OverlayValues[176] = d176
		ps1450.OverlayValues[177] = d177
		ps1450.OverlayValues[178] = d178
		ps1450.OverlayValues[179] = d179
		ps1450.OverlayValues[180] = d180
		ps1450.OverlayValues[181] = d181
		ps1450.OverlayValues[182] = d182
		ps1450.OverlayValues[183] = d183
		ps1450.OverlayValues[184] = d184
		ps1450.OverlayValues[187] = d187
		ps1450.OverlayValues[344] = d344
		ps1450.OverlayValues[345] = d345
		ps1450.OverlayValues[346] = d346
		ps1450.OverlayValues[347] = d347
		ps1450.OverlayValues[349] = d349
		ps1450.OverlayValues[350] = d350
		ps1450.OverlayValues[351] = d351
		ps1450.OverlayValues[352] = d352
		ps1450.OverlayValues[353] = d353
		ps1450.OverlayValues[354] = d354
		ps1450.OverlayValues[355] = d355
		ps1450.OverlayValues[356] = d356
		ps1450.OverlayValues[358] = d358
		ps1450.OverlayValues[360] = d360
		ps1450.OverlayValues[361] = d361
		ps1450.OverlayValues[362] = d362
		ps1450.OverlayValues[456] = d456
		ps1450.OverlayValues[457] = d457
		ps1450.OverlayValues[460] = d460
		ps1450.OverlayValues[557] = d557
		ps1450.OverlayValues[558] = d558
		ps1450.OverlayValues[559] = d559
		ps1450.OverlayValues[560] = d560
		ps1450.OverlayValues[561] = d561
		ps1450.OverlayValues[563] = d563
		ps1450.OverlayValues[564] = d564
		ps1450.OverlayValues[565] = d565
		ps1450.OverlayValues[566] = d566
		ps1450.OverlayValues[567] = d567
		ps1450.OverlayValues[568] = d568
		ps1450.OverlayValues[569] = d569
		ps1450.OverlayValues[570] = d570
		ps1450.OverlayValues[571] = d571
		ps1450.OverlayValues[572] = d572
		ps1450.OverlayValues[573] = d573
		ps1450.OverlayValues[574] = d574
		ps1450.OverlayValues[575] = d575
		ps1450.OverlayValues[576] = d576
		ps1450.OverlayValues[577] = d577
		ps1450.OverlayValues[578] = d578
		ps1450.OverlayValues[579] = d579
		ps1450.OverlayValues[580] = d580
		ps1450.OverlayValues[581] = d581
		ps1450.OverlayValues[582] = d582
		ps1450.OverlayValues[583] = d583
		ps1450.OverlayValues[584] = d584
		ps1450.OverlayValues[585] = d585
		ps1450.OverlayValues[586] = d586
		ps1450.OverlayValues[587] = d587
		ps1450.OverlayValues[588] = d588
		ps1450.OverlayValues[589] = d589
		ps1450.OverlayValues[850] = d850
		ps1450.OverlayValues[851] = d851
		ps1450.OverlayValues[852] = d852
		ps1450.OverlayValues[854] = d854
		ps1450.OverlayValues[855] = d855
		ps1450.OverlayValues[856] = d856
		ps1450.OverlayValues[857] = d857
		ps1450.OverlayValues[858] = d858
		ps1450.OverlayValues[859] = d859
		ps1450.OverlayValues[860] = d860
		ps1450.OverlayValues[862] = d862
		ps1450.OverlayValues[864] = d864
		ps1450.OverlayValues[865] = d865
		ps1450.OverlayValues[1007] = d1007
		ps1450.OverlayValues[1008] = d1008
		ps1450.OverlayValues[1011] = d1011
		ps1450.OverlayValues[1156] = d1156
		ps1450.OverlayValues[1157] = d1157
		ps1450.OverlayValues[1158] = d1158
		ps1450.OverlayValues[1159] = d1159
		ps1450.OverlayValues[1161] = d1161
		ps1450.OverlayValues[1162] = d1162
		ps1450.OverlayValues[1163] = d1163
		ps1450.OverlayValues[1164] = d1164
		ps1450.OverlayValues[1165] = d1165
		ps1450.OverlayValues[1166] = d1166
		ps1450.OverlayValues[1167] = d1167
		ps1450.OverlayValues[1168] = d1168
		ps1450.OverlayValues[1169] = d1169
		ps1450.OverlayValues[1170] = d1170
		ps1450.OverlayValues[1172] = d1172
		ps1450.OverlayValues[1173] = d1173
		ps1450.OverlayValues[1174] = d1174
		ps1450.OverlayValues[1175] = d1175
		ps1450.OverlayValues[1176] = d1176
		ps1450.OverlayValues[1177] = d1177
		ps1450.OverlayValues[1178] = d1178
		ps1450.OverlayValues[1179] = d1179
		ps1450.OverlayValues[1180] = d1180
		ps1450.OverlayValues[1181] = d1181
		ps1450.OverlayValues[1182] = d1182
		ps1450.OverlayValues[1183] = d1183
		ps1450.OverlayValues[1184] = d1184
		ps1450.OverlayValues[1185] = d1185
		ps1450.OverlayValues[1186] = d1186
		ps1450.OverlayValues[1187] = d1187
		ps1450.OverlayValues[1188] = d1188
		ps1450.OverlayValues[1189] = d1189
		ps1450.OverlayValues[1190] = d1190
		ps1450.OverlayValues[1191] = d1191
		ps1450.OverlayValues[1192] = d1192
		ps1450.OverlayValues[1193] = d1193
		ps1450.OverlayValues[1194] = d1194
		ps1450.OverlayValues[1195] = d1195
		ps1450.OverlayValues[1196] = d1196
		ps1450.OverlayValues[1197] = d1197
		ps1450.OverlayValues[1198] = d1198
		ps1450.OverlayValues[1199] = d1199
		ps1450.OverlayValues[1200] = d1200
		ps1450.OverlayValues[1201] = d1201
		ps1450.OverlayValues[1202] = d1202
		ps1450.OverlayValues[1203] = d1203
		ps1450.OverlayValues[1204] = d1204
		ps1450.OverlayValues[1205] = d1205
		ps1450.OverlayValues[1206] = d1206
		ps1450.OverlayValues[1207] = d1207
		ps1450.OverlayValues[1208] = d1208
		ps1450.OverlayValues[1209] = d1209
		ps1450.OverlayValues[1210] = d1210
		ps1450.OverlayValues[1211] = d1211
		ps1450.OverlayValues[1212] = d1212
		ps1450.OverlayValues[1213] = d1213
		ps1450.OverlayValues[1214] = d1214
		ps1450.OverlayValues[1215] = d1215
		ps1450.OverlayValues[1216] = d1216
		ps1450.OverlayValues[1217] = d1217
		ps1450.OverlayValues[1218] = d1218
		ps1450.OverlayValues[1219] = d1219
		ps1450.OverlayValues[1220] = d1220
		ps1450.OverlayValues[1221] = d1221
		ps1450.OverlayValues[1222] = d1222
		ps1450.OverlayValues[1223] = d1223
		ps1450.OverlayValues[1224] = d1224
		ps1450.OverlayValues[1225] = d1225
		ps1450.OverlayValues[1226] = d1226
		ps1450.OverlayValues[1227] = d1227
		ps1450.OverlayValues[1228] = d1228
		ps1450.OverlayValues[1229] = d1229
		ps1450.OverlayValues[1230] = d1230
		ps1451 := scm.PhiState{General: true}
		ps1451.OverlayValues = make([]scm.JITValueDesc, 1231)
		ps1451.OverlayValues[5] = d5
		ps1451.OverlayValues[6] = d6
		ps1451.OverlayValues[7] = d7
		ps1451.OverlayValues[8] = d8
		ps1451.OverlayValues[9] = d9
		ps1451.OverlayValues[10] = d10
		ps1451.OverlayValues[11] = d11
		ps1451.OverlayValues[12] = d12
		ps1451.OverlayValues[13] = d13
		ps1451.OverlayValues[14] = d14
		ps1451.OverlayValues[15] = d15
		ps1451.OverlayValues[16] = d16
		ps1451.OverlayValues[17] = d17
		ps1451.OverlayValues[18] = d18
		ps1451.OverlayValues[19] = d19
		ps1451.OverlayValues[21] = d21
		ps1451.OverlayValues[22] = d22
		ps1451.OverlayValues[23] = d23
		ps1451.OverlayValues[24] = d24
		ps1451.OverlayValues[25] = d25
		ps1451.OverlayValues[26] = d26
		ps1451.OverlayValues[27] = d27
		ps1451.OverlayValues[28] = d28
		ps1451.OverlayValues[29] = d29
		ps1451.OverlayValues[30] = d30
		ps1451.OverlayValues[31] = d31
		ps1451.OverlayValues[32] = d32
		ps1451.OverlayValues[33] = d33
		ps1451.OverlayValues[34] = d34
		ps1451.OverlayValues[35] = d35
		ps1451.OverlayValues[36] = d36
		ps1451.OverlayValues[37] = d37
		ps1451.OverlayValues[38] = d38
		ps1451.OverlayValues[39] = d39
		ps1451.OverlayValues[40] = d40
		ps1451.OverlayValues[41] = d41
		ps1451.OverlayValues[42] = d42
		ps1451.OverlayValues[43] = d43
		ps1451.OverlayValues[44] = d44
		ps1451.OverlayValues[45] = d45
		ps1451.OverlayValues[46] = d46
		ps1451.OverlayValues[47] = d47
		ps1451.OverlayValues[48] = d48
		ps1451.OverlayValues[49] = d49
		ps1451.OverlayValues[50] = d50
		ps1451.OverlayValues[51] = d51
		ps1451.OverlayValues[54] = d54
		ps1451.OverlayValues[55] = d55
		ps1451.OverlayValues[56] = d56
		ps1451.OverlayValues[159] = d159
		ps1451.OverlayValues[160] = d160
		ps1451.OverlayValues[161] = d161
		ps1451.OverlayValues[162] = d162
		ps1451.OverlayValues[163] = d163
		ps1451.OverlayValues[164] = d164
		ps1451.OverlayValues[165] = d165
		ps1451.OverlayValues[166] = d166
		ps1451.OverlayValues[167] = d167
		ps1451.OverlayValues[168] = d168
		ps1451.OverlayValues[169] = d169
		ps1451.OverlayValues[170] = d170
		ps1451.OverlayValues[171] = d171
		ps1451.OverlayValues[172] = d172
		ps1451.OverlayValues[173] = d173
		ps1451.OverlayValues[174] = d174
		ps1451.OverlayValues[175] = d175
		ps1451.OverlayValues[176] = d176
		ps1451.OverlayValues[177] = d177
		ps1451.OverlayValues[178] = d178
		ps1451.OverlayValues[179] = d179
		ps1451.OverlayValues[180] = d180
		ps1451.OverlayValues[181] = d181
		ps1451.OverlayValues[182] = d182
		ps1451.OverlayValues[183] = d183
		ps1451.OverlayValues[184] = d184
		ps1451.OverlayValues[187] = d187
		ps1451.OverlayValues[344] = d344
		ps1451.OverlayValues[345] = d345
		ps1451.OverlayValues[346] = d346
		ps1451.OverlayValues[347] = d347
		ps1451.OverlayValues[349] = d349
		ps1451.OverlayValues[350] = d350
		ps1451.OverlayValues[351] = d351
		ps1451.OverlayValues[352] = d352
		ps1451.OverlayValues[353] = d353
		ps1451.OverlayValues[354] = d354
		ps1451.OverlayValues[355] = d355
		ps1451.OverlayValues[356] = d356
		ps1451.OverlayValues[358] = d358
		ps1451.OverlayValues[360] = d360
		ps1451.OverlayValues[361] = d361
		ps1451.OverlayValues[362] = d362
		ps1451.OverlayValues[456] = d456
		ps1451.OverlayValues[457] = d457
		ps1451.OverlayValues[460] = d460
		ps1451.OverlayValues[557] = d557
		ps1451.OverlayValues[558] = d558
		ps1451.OverlayValues[559] = d559
		ps1451.OverlayValues[560] = d560
		ps1451.OverlayValues[561] = d561
		ps1451.OverlayValues[563] = d563
		ps1451.OverlayValues[564] = d564
		ps1451.OverlayValues[565] = d565
		ps1451.OverlayValues[566] = d566
		ps1451.OverlayValues[567] = d567
		ps1451.OverlayValues[568] = d568
		ps1451.OverlayValues[569] = d569
		ps1451.OverlayValues[570] = d570
		ps1451.OverlayValues[571] = d571
		ps1451.OverlayValues[572] = d572
		ps1451.OverlayValues[573] = d573
		ps1451.OverlayValues[574] = d574
		ps1451.OverlayValues[575] = d575
		ps1451.OverlayValues[576] = d576
		ps1451.OverlayValues[577] = d577
		ps1451.OverlayValues[578] = d578
		ps1451.OverlayValues[579] = d579
		ps1451.OverlayValues[580] = d580
		ps1451.OverlayValues[581] = d581
		ps1451.OverlayValues[582] = d582
		ps1451.OverlayValues[583] = d583
		ps1451.OverlayValues[584] = d584
		ps1451.OverlayValues[585] = d585
		ps1451.OverlayValues[586] = d586
		ps1451.OverlayValues[587] = d587
		ps1451.OverlayValues[588] = d588
		ps1451.OverlayValues[589] = d589
		ps1451.OverlayValues[850] = d850
		ps1451.OverlayValues[851] = d851
		ps1451.OverlayValues[852] = d852
		ps1451.OverlayValues[854] = d854
		ps1451.OverlayValues[855] = d855
		ps1451.OverlayValues[856] = d856
		ps1451.OverlayValues[857] = d857
		ps1451.OverlayValues[858] = d858
		ps1451.OverlayValues[859] = d859
		ps1451.OverlayValues[860] = d860
		ps1451.OverlayValues[862] = d862
		ps1451.OverlayValues[864] = d864
		ps1451.OverlayValues[865] = d865
		ps1451.OverlayValues[1007] = d1007
		ps1451.OverlayValues[1008] = d1008
		ps1451.OverlayValues[1011] = d1011
		ps1451.OverlayValues[1156] = d1156
		ps1451.OverlayValues[1157] = d1157
		ps1451.OverlayValues[1158] = d1158
		ps1451.OverlayValues[1159] = d1159
		ps1451.OverlayValues[1161] = d1161
		ps1451.OverlayValues[1162] = d1162
		ps1451.OverlayValues[1163] = d1163
		ps1451.OverlayValues[1164] = d1164
		ps1451.OverlayValues[1165] = d1165
		ps1451.OverlayValues[1166] = d1166
		ps1451.OverlayValues[1167] = d1167
		ps1451.OverlayValues[1168] = d1168
		ps1451.OverlayValues[1169] = d1169
		ps1451.OverlayValues[1170] = d1170
		ps1451.OverlayValues[1172] = d1172
		ps1451.OverlayValues[1173] = d1173
		ps1451.OverlayValues[1174] = d1174
		ps1451.OverlayValues[1175] = d1175
		ps1451.OverlayValues[1176] = d1176
		ps1451.OverlayValues[1177] = d1177
		ps1451.OverlayValues[1178] = d1178
		ps1451.OverlayValues[1179] = d1179
		ps1451.OverlayValues[1180] = d1180
		ps1451.OverlayValues[1181] = d1181
		ps1451.OverlayValues[1182] = d1182
		ps1451.OverlayValues[1183] = d1183
		ps1451.OverlayValues[1184] = d1184
		ps1451.OverlayValues[1185] = d1185
		ps1451.OverlayValues[1186] = d1186
		ps1451.OverlayValues[1187] = d1187
		ps1451.OverlayValues[1188] = d1188
		ps1451.OverlayValues[1189] = d1189
		ps1451.OverlayValues[1190] = d1190
		ps1451.OverlayValues[1191] = d1191
		ps1451.OverlayValues[1192] = d1192
		ps1451.OverlayValues[1193] = d1193
		ps1451.OverlayValues[1194] = d1194
		ps1451.OverlayValues[1195] = d1195
		ps1451.OverlayValues[1196] = d1196
		ps1451.OverlayValues[1197] = d1197
		ps1451.OverlayValues[1198] = d1198
		ps1451.OverlayValues[1199] = d1199
		ps1451.OverlayValues[1200] = d1200
		ps1451.OverlayValues[1201] = d1201
		ps1451.OverlayValues[1202] = d1202
		ps1451.OverlayValues[1203] = d1203
		ps1451.OverlayValues[1204] = d1204
		ps1451.OverlayValues[1205] = d1205
		ps1451.OverlayValues[1206] = d1206
		ps1451.OverlayValues[1207] = d1207
		ps1451.OverlayValues[1208] = d1208
		ps1451.OverlayValues[1209] = d1209
		ps1451.OverlayValues[1210] = d1210
		ps1451.OverlayValues[1211] = d1211
		ps1451.OverlayValues[1212] = d1212
		ps1451.OverlayValues[1213] = d1213
		ps1451.OverlayValues[1214] = d1214
		ps1451.OverlayValues[1215] = d1215
		ps1451.OverlayValues[1216] = d1216
		ps1451.OverlayValues[1217] = d1217
		ps1451.OverlayValues[1218] = d1218
		ps1451.OverlayValues[1219] = d1219
		ps1451.OverlayValues[1220] = d1220
		ps1451.OverlayValues[1221] = d1221
		ps1451.OverlayValues[1222] = d1222
		ps1451.OverlayValues[1223] = d1223
		ps1451.OverlayValues[1224] = d1224
		ps1451.OverlayValues[1225] = d1225
		ps1451.OverlayValues[1226] = d1226
		ps1451.OverlayValues[1227] = d1227
		ps1451.OverlayValues[1228] = d1228
		ps1451.OverlayValues[1229] = d1229
		ps1451.OverlayValues[1230] = d1230
		snap1452 := d5
		snap1453 := d6
		snap1454 := d7
		snap1455 := d8
		snap1456 := d9
		snap1457 := d10
		snap1458 := d11
		snap1459 := d12
		snap1460 := d13
		snap1461 := d14
		snap1462 := d15
		snap1463 := d16
		snap1464 := d17
		snap1465 := d18
		snap1466 := d19
		snap1467 := d21
		snap1468 := d22
		snap1469 := d23
		snap1470 := d24
		snap1471 := d25
		snap1472 := d26
		snap1473 := d27
		snap1474 := d28
		snap1475 := d29
		snap1476 := d30
		snap1477 := d31
		snap1478 := d32
		snap1479 := d33
		snap1480 := d34
		snap1481 := d35
		snap1482 := d36
		snap1483 := d37
		snap1484 := d38
		snap1485 := d39
		snap1486 := d40
		snap1487 := d41
		snap1488 := d42
		snap1489 := d43
		snap1490 := d44
		snap1491 := d45
		snap1492 := d46
		snap1493 := d47
		snap1494 := d48
		snap1495 := d49
		snap1496 := d50
		snap1497 := d51
		snap1498 := d54
		snap1499 := d55
		snap1500 := d56
		snap1501 := d159
		snap1502 := d160
		snap1503 := d161
		snap1504 := d162
		snap1505 := d163
		snap1506 := d164
		snap1507 := d165
		snap1508 := d166
		snap1509 := d167
		snap1510 := d168
		snap1511 := d169
		snap1512 := d170
		snap1513 := d171
		snap1514 := d172
		snap1515 := d173
		snap1516 := d174
		snap1517 := d175
		snap1518 := d176
		snap1519 := d177
		snap1520 := d178
		snap1521 := d179
		snap1522 := d180
		snap1523 := d181
		snap1524 := d182
		snap1525 := d183
		snap1526 := d184
		snap1527 := d187
		snap1528 := d344
		snap1529 := d345
		snap1530 := d346
		snap1531 := d347
		snap1532 := d349
		snap1533 := d350
		snap1534 := d351
		snap1535 := d352
		snap1536 := d353
		snap1537 := d354
		snap1538 := d355
		snap1539 := d356
		snap1540 := d358
		snap1541 := d360
		snap1542 := d361
		snap1543 := d362
		snap1544 := d456
		snap1545 := d457
		snap1546 := d460
		snap1547 := d557
		snap1548 := d558
		snap1549 := d559
		snap1550 := d560
		snap1551 := d561
		snap1552 := d563
		snap1553 := d564
		snap1554 := d565
		snap1555 := d566
		snap1556 := d567
		snap1557 := d568
		snap1558 := d569
		snap1559 := d570
		snap1560 := d571
		snap1561 := d572
		snap1562 := d573
		snap1563 := d574
		snap1564 := d575
		snap1565 := d576
		snap1566 := d577
		snap1567 := d578
		snap1568 := d579
		snap1569 := d580
		snap1570 := d581
		snap1571 := d582
		snap1572 := d583
		snap1573 := d584
		snap1574 := d585
		snap1575 := d586
		snap1576 := d587
		snap1577 := d588
		snap1578 := d589
		snap1579 := d850
		snap1580 := d851
		snap1581 := d852
		snap1582 := d854
		snap1583 := d855
		snap1584 := d856
		snap1585 := d857
		snap1586 := d858
		snap1587 := d859
		snap1588 := d860
		snap1589 := d862
		snap1590 := d864
		snap1591 := d865
		snap1592 := d1007
		snap1593 := d1008
		snap1594 := d1011
		snap1595 := d1156
		snap1596 := d1157
		snap1597 := d1158
		snap1598 := d1159
		snap1599 := d1161
		snap1600 := d1162
		snap1601 := d1163
		snap1602 := d1164
		snap1603 := d1165
		snap1604 := d1166
		snap1605 := d1167
		snap1606 := d1168
		snap1607 := d1169
		snap1608 := d1170
		snap1609 := d1172
		snap1610 := d1173
		snap1611 := d1174
		snap1612 := d1175
		snap1613 := d1176
		snap1614 := d1177
		snap1615 := d1178
		snap1616 := d1179
		snap1617 := d1180
		snap1618 := d1181
		snap1619 := d1182
		snap1620 := d1183
		snap1621 := d1184
		snap1622 := d1185
		snap1623 := d1186
		snap1624 := d1187
		snap1625 := d1188
		snap1626 := d1189
		snap1627 := d1190
		snap1628 := d1191
		snap1629 := d1192
		snap1630 := d1193
		snap1631 := d1194
		snap1632 := d1195
		snap1633 := d1196
		snap1634 := d1197
		snap1635 := d1198
		snap1636 := d1199
		snap1637 := d1200
		snap1638 := d1201
		snap1639 := d1202
		snap1640 := d1203
		snap1641 := d1204
		snap1642 := d1205
		snap1643 := d1206
		snap1644 := d1207
		snap1645 := d1208
		snap1646 := d1209
		snap1647 := d1210
		snap1648 := d1211
		snap1649 := d1212
		snap1650 := d1213
		snap1651 := d1214
		snap1652 := d1215
		snap1653 := d1216
		snap1654 := d1217
		snap1655 := d1218
		snap1656 := d1219
		snap1657 := d1220
		snap1658 := d1221
		snap1659 := d1222
		snap1660 := d1223
		snap1661 := d1224
		snap1662 := d1225
		snap1663 := d1226
		snap1664 := d1227
		snap1665 := d1228
		snap1666 := d1229
		snap1667 := d1230
		alloc1668 := ctx.SnapshotAllocState()
		if !bbs[12].Rendered {
			bbs[12].RenderPS(ps1451)
		}
		ctx.RestoreAllocState(alloc1668)
		d5 = snap1452
		d6 = snap1453
		d7 = snap1454
		d8 = snap1455
		d9 = snap1456
		d10 = snap1457
		d11 = snap1458
		d12 = snap1459
		d13 = snap1460
		d14 = snap1461
		d15 = snap1462
		d16 = snap1463
		d17 = snap1464
		d18 = snap1465
		d19 = snap1466
		d21 = snap1467
		d22 = snap1468
		d23 = snap1469
		d24 = snap1470
		d25 = snap1471
		d26 = snap1472
		d27 = snap1473
		d28 = snap1474
		d29 = snap1475
		d30 = snap1476
		d31 = snap1477
		d32 = snap1478
		d33 = snap1479
		d34 = snap1480
		d35 = snap1481
		d36 = snap1482
		d37 = snap1483
		d38 = snap1484
		d39 = snap1485
		d40 = snap1486
		d41 = snap1487
		d42 = snap1488
		d43 = snap1489
		d44 = snap1490
		d45 = snap1491
		d46 = snap1492
		d47 = snap1493
		d48 = snap1494
		d49 = snap1495
		d50 = snap1496
		d51 = snap1497
		d54 = snap1498
		d55 = snap1499
		d56 = snap1500
		d159 = snap1501
		d160 = snap1502
		d161 = snap1503
		d162 = snap1504
		d163 = snap1505
		d164 = snap1506
		d165 = snap1507
		d166 = snap1508
		d167 = snap1509
		d168 = snap1510
		d169 = snap1511
		d170 = snap1512
		d171 = snap1513
		d172 = snap1514
		d173 = snap1515
		d174 = snap1516
		d175 = snap1517
		d176 = snap1518
		d177 = snap1519
		d178 = snap1520
		d179 = snap1521
		d180 = snap1522
		d181 = snap1523
		d182 = snap1524
		d183 = snap1525
		d184 = snap1526
		d187 = snap1527
		d344 = snap1528
		d345 = snap1529
		d346 = snap1530
		d347 = snap1531
		d349 = snap1532
		d350 = snap1533
		d351 = snap1534
		d352 = snap1535
		d353 = snap1536
		d354 = snap1537
		d355 = snap1538
		d356 = snap1539
		d358 = snap1540
		d360 = snap1541
		d361 = snap1542
		d362 = snap1543
		d456 = snap1544
		d457 = snap1545
		d460 = snap1546
		d557 = snap1547
		d558 = snap1548
		d559 = snap1549
		d560 = snap1550
		d561 = snap1551
		d563 = snap1552
		d564 = snap1553
		d565 = snap1554
		d566 = snap1555
		d567 = snap1556
		d568 = snap1557
		d569 = snap1558
		d570 = snap1559
		d571 = snap1560
		d572 = snap1561
		d573 = snap1562
		d574 = snap1563
		d575 = snap1564
		d576 = snap1565
		d577 = snap1566
		d578 = snap1567
		d579 = snap1568
		d580 = snap1569
		d581 = snap1570
		d582 = snap1571
		d583 = snap1572
		d584 = snap1573
		d585 = snap1574
		d586 = snap1575
		d587 = snap1576
		d588 = snap1577
		d589 = snap1578
		d850 = snap1579
		d851 = snap1580
		d852 = snap1581
		d854 = snap1582
		d855 = snap1583
		d856 = snap1584
		d857 = snap1585
		d858 = snap1586
		d859 = snap1587
		d860 = snap1588
		d862 = snap1589
		d864 = snap1590
		d865 = snap1591
		d1007 = snap1592
		d1008 = snap1593
		d1011 = snap1594
		d1156 = snap1595
		d1157 = snap1596
		d1158 = snap1597
		d1159 = snap1598
		d1161 = snap1599
		d1162 = snap1600
		d1163 = snap1601
		d1164 = snap1602
		d1165 = snap1603
		d1166 = snap1604
		d1167 = snap1605
		d1168 = snap1606
		d1169 = snap1607
		d1170 = snap1608
		d1172 = snap1609
		d1173 = snap1610
		d1174 = snap1611
		d1175 = snap1612
		d1176 = snap1613
		d1177 = snap1614
		d1178 = snap1615
		d1179 = snap1616
		d1180 = snap1617
		d1181 = snap1618
		d1182 = snap1619
		d1183 = snap1620
		d1184 = snap1621
		d1185 = snap1622
		d1186 = snap1623
		d1187 = snap1624
		d1188 = snap1625
		d1189 = snap1626
		d1190 = snap1627
		d1191 = snap1628
		d1192 = snap1629
		d1193 = snap1630
		d1194 = snap1631
		d1195 = snap1632
		d1196 = snap1633
		d1197 = snap1634
		d1198 = snap1635
		d1199 = snap1636
		d1200 = snap1637
		d1201 = snap1638
		d1202 = snap1639
		d1203 = snap1640
		d1204 = snap1641
		d1205 = snap1642
		d1206 = snap1643
		d1207 = snap1644
		d1208 = snap1645
		d1209 = snap1646
		d1210 = snap1647
		d1211 = snap1648
		d1212 = snap1649
		d1213 = snap1650
		d1214 = snap1651
		d1215 = snap1652
		d1216 = snap1653
		d1217 = snap1654
		d1218 = snap1655
		d1219 = snap1656
		d1220 = snap1657
		d1221 = snap1658
		d1222 = snap1659
		d1223 = snap1660
		d1224 = snap1661
		d1225 = snap1662
		d1226 = snap1663
		d1227 = snap1664
		d1228 = snap1665
		d1229 = snap1666
		d1230 = snap1667
		if !bbs[11].Rendered {
			return bbs[11].RenderPS(ps1450)
		}
		return result
		return result
	}
	ps1669 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps1669)
	ctx.MarkLabel(lbl0)
	d1670 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r3, Reg2: r4}
	ctx.BindReg(r3, &d1670)
	ctx.BindReg(r4, &d1670)
	ctx.EmitMovPairToResult(&d1670, &result)
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
