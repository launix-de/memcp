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
	return uint(unsafe.Sizeof(*s)-unsafe.Sizeof(s.recordId)-unsafe.Sizeof(s.start)-unsafe.Sizeof(s.stride)) + s.recordId.ComputeSize() + s.start.ComputeSize() + s.stride.ComputeSize()
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
	var d184 scm.JITValueDesc
	_ = d184
	var d335 scm.JITValueDesc
	_ = d335
	var d336 scm.JITValueDesc
	_ = d336
	var d337 scm.JITValueDesc
	_ = d337
	var d338 scm.JITValueDesc
	_ = d338
	var d340 scm.JITValueDesc
	_ = d340
	var d341 scm.JITValueDesc
	_ = d341
	var d342 scm.JITValueDesc
	_ = d342
	var d343 scm.JITValueDesc
	_ = d343
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
	var d351 scm.JITValueDesc
	_ = d351
	var d352 scm.JITValueDesc
	_ = d352
	var d353 scm.JITValueDesc
	_ = d353
	var d444 scm.JITValueDesc
	_ = d444
	var d445 scm.JITValueDesc
	_ = d445
	var d448 scm.JITValueDesc
	_ = d448
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
	var d548 scm.JITValueDesc
	_ = d548
	var d549 scm.JITValueDesc
	_ = d549
	var d550 scm.JITValueDesc
	_ = d550
	var d551 scm.JITValueDesc
	_ = d551
	var d552 scm.JITValueDesc
	_ = d552
	var d553 scm.JITValueDesc
	_ = d553
	var d554 scm.JITValueDesc
	_ = d554
	var d555 scm.JITValueDesc
	_ = d555
	var d556 scm.JITValueDesc
	_ = d556
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
	var d562 scm.JITValueDesc
	_ = d562
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
	var d829 scm.JITValueDesc
	_ = d829
	var d830 scm.JITValueDesc
	_ = d830
	var d831 scm.JITValueDesc
	_ = d831
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
	var d838 scm.JITValueDesc
	_ = d838
	var d839 scm.JITValueDesc
	_ = d839
	var d841 scm.JITValueDesc
	_ = d841
	var d843 scm.JITValueDesc
	_ = d843
	var d844 scm.JITValueDesc
	_ = d844
	var d983 scm.JITValueDesc
	_ = d983
	var d984 scm.JITValueDesc
	_ = d984
	var d987 scm.JITValueDesc
	_ = d987
	var d1129 scm.JITValueDesc
	_ = d1129
	var d1130 scm.JITValueDesc
	_ = d1130
	var d1131 scm.JITValueDesc
	_ = d1131
	var d1132 scm.JITValueDesc
	_ = d1132
	var d1134 scm.JITValueDesc
	_ = d1134
	var d1135 scm.JITValueDesc
	_ = d1135
	var d1136 scm.JITValueDesc
	_ = d1136
	var d1137 scm.JITValueDesc
	_ = d1137
	var d1138 scm.JITValueDesc
	_ = d1138
	var d1139 scm.JITValueDesc
	_ = d1139
	var d1140 scm.JITValueDesc
	_ = d1140
	var d1141 scm.JITValueDesc
	_ = d1141
	var d1142 scm.JITValueDesc
	_ = d1142
	var d1143 scm.JITValueDesc
	_ = d1143
	var d1145 scm.JITValueDesc
	_ = d1145
	var d1146 scm.JITValueDesc
	_ = d1146
	var d1147 scm.JITValueDesc
	_ = d1147
	var d1148 scm.JITValueDesc
	_ = d1148
	var d1149 scm.JITValueDesc
	_ = d1149
	var d1150 scm.JITValueDesc
	_ = d1150
	var d1151 scm.JITValueDesc
	_ = d1151
	var d1152 scm.JITValueDesc
	_ = d1152
	var d1153 scm.JITValueDesc
	_ = d1153
	var d1154 scm.JITValueDesc
	_ = d1154
	var d1155 scm.JITValueDesc
	_ = d1155
	var d1156 scm.JITValueDesc
	_ = d1156
	var d1157 scm.JITValueDesc
	_ = d1157
	var d1158 scm.JITValueDesc
	_ = d1158
	var d1159 scm.JITValueDesc
	_ = d1159
	var d1160 scm.JITValueDesc
	_ = d1160
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
	var d1171 scm.JITValueDesc
	_ = d1171
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
	phiHomeOK3 := registerHomes1.Available&(uint16(1)<<1) == uint16(1)<<1
	if phiHomeOK3 {
		r1 = registerHomes1.Registers[1]
	}
	var r2 scm.Reg
	phiHomeOK4 := registerHomes1.Available&(uint16(1)<<2) == uint16(1)<<2
	if phiHomeOK4 {
		r2 = registerHomes1.Registers[2]
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
			ctx.FlushRegisterMoves()
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
		r3 := ctx.AllocReg()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).lastValue)
			ctx.EmitMovRegMem64(r3, fieldAddr)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
			ctx.EmitMovRegMem(r3, thisptr.Reg, off)
		}
		d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r3}
		ctx.BindReg(r3, &d14)
		ctx.EnsureDesc(&d14)
		ctx.EnsureDesc(&d14)
		var d15 scm.JITValueDesc
		if d14.Loc == scm.LocImm {
			d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d14.Imm.Int()))))}
		} else {
			r4 := ctx.AllocReg()
			ctx.EmitMovRegReg(r4, d14.Reg)
			ctx.EmitShlRegImm8(r4, 32)
			ctx.EmitShrRegImm8(r4, 32)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r4}
			ctx.BindReg(r4, &d15)
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
			r5 := ctx.AllocReg()
			ctx.EmitMovRegMemL(r5, thisptr.Reg, off)
			d16 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r5}
			ctx.BindReg(r5, &d16)
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
			if d15.Loc == scm.LocReg || d15.Loc == scm.LocFPReg {
				ctx.ProtectReg(d15.Reg)
			} else if d15.Loc == scm.LocRegPair {
				ctx.ProtectReg(d15.Reg)
				ctx.ProtectReg(d15.Reg2)
			}
			ctx.SyncDesc(&d17)
			if d17.Loc == scm.LocReg || d17.Loc == scm.LocFPReg {
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
			if d15.Loc == scm.LocReg || d15.Loc == scm.LocFPReg {
				ctx.UnprotectReg(d15.Reg)
			} else if d15.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d15.Reg)
				ctx.UnprotectReg(d15.Reg2)
			}
			if d17.Loc == scm.LocReg || d17.Loc == scm.LocFPReg {
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
			ctx.FlushRegisterMoves()
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
			r6 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r6, thisptr.Reg, off)
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r6}
			ctx.BindReg(r6, &d28)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d28)
		ctx.EnsureDesc(&d28)
		var d29 scm.JITValueDesc
		if d28.Loc == scm.LocImm {
			d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d28.Imm.Int()))))}
		} else {
			r7 := ctx.AllocReg()
			ctx.EmitMovRegReg(r7, d28.Reg)
			ctx.EmitShlRegImm8(r7, 56)
			ctx.EmitShrRegImm8(r7, 56)
			d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
			ctx.BindReg(r7, &d29)
		}
		ctx.FreeDesc(&d28)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d27)
		ctx.EnsureDesc(&d27)
		var d30 scm.JITValueDesc
		if d27.Loc == scm.LocImm {
			d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d27.Imm.Int()))))}
		} else {
			r8 := ctx.AllocReg()
			ctx.EmitMovRegReg(r8, d27.Reg)
			ctx.EmitShlRegImm8(r8, 32)
			ctx.EmitShrRegImm8(r8, 32)
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
			ctx.BindReg(r8, &d30)
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
			r9 := ctx.AllocRegExcept(d30.Reg, d29.Reg)
			ctx.EmitMovRegReg(r9, d30.Reg)
			ctx.EmitImulInt64(r9, d29.Reg)
			d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
			ctx.BindReg(r9, &d31)
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
			r10 := ctx.AllocRegExcept(d31.Reg)
			ctx.EmitMovRegReg(r10, d31.Reg)
			ctx.EmitShrRegImm8(r10, 6)
			d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
			ctx.BindReg(r10, &d32)
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
			r11 := ctx.AllocRegExcept(d31.Reg)
			ctx.EmitMovRegReg(r11, d31.Reg)
			ctx.EmitAndRegImm32(r11, 63)
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
			ctx.BindReg(r11, &d33)
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
			r12 := ctx.AllocReg()
			r13 := ctx.AllocRegExcept(r12)
			r14 := ctx.AllocRegExcept(r12, r13)
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			ctx.EmitMovRegMem(r12, thisptr.Reg, off)
			ctx.EmitMovRegMem(r13, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r14, thisptr.Reg, off+16)
			d34 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r12, Reg2: r13, Reg3: r14}
			ctx.BindReg(r12, &d34)
			ctx.BindReg(r13, &d34)
			ctx.BindReg(r14, &d34)
			ctx.BindReg(r12, &d34)
			ctx.BindReg(r13, &d34)
			ctx.BindReg(r14, &d34)
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
			r15 := ctx.AllocRegExcept(d35.Reg)
			ctx.EmitMovRegReg(r15, d35.Reg)
			ctx.EmitShlRegImm8(r15, uint8(d33.Imm.Int()))
			d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
			ctx.BindReg(r15, &d36)
		} else {
			{
				shiftSrc := d35.Reg
				r16 := ctx.AllocRegExcept(d35.Reg, d33.Reg)
				ctx.EmitMovRegReg(r16, d35.Reg)
				shiftSrc = r16
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
			r17 := ctx.AllocRegExcept(d39.Reg)
			ctx.EmitMovRegReg(r17, d39.Reg)
			d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			ctx.BindReg(r17, &d40)
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
			r18 := ctx.AllocRegExcept(d39.Reg, d33.Reg)
			ctx.EmitMovRegReg(r18, d39.Reg)
			ctx.EmitSubInt64(r18, d33.Reg)
			d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			ctx.BindReg(r18, &d40)
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
			r19 := ctx.AllocRegExcept(d38.Reg)
			ctx.EmitMovRegReg(r19, d38.Reg)
			ctx.EmitShrRegImm8(r19, uint8(d40.Imm.Int()))
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d41)
		} else {
			{
				shiftSrc := d38.Reg
				r20 := ctx.AllocRegExcept(d38.Reg, d40.Reg)
				ctx.EmitMovRegReg(r20, d38.Reg)
				shiftSrc = r20
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
			r21 := ctx.AllocRegExcept(d36.Reg)
			ctx.EmitMovRegReg(r21, d36.Reg)
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			ctx.BindReg(r21, &d42)
		} else if d36.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d41.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d36.Imm.Int()))
			ctx.EmitOrInt64(scratch, d41.Reg)
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d42)
		} else if d41.Loc == scm.LocImm {
			r22 := ctx.AllocRegExcept(d36.Reg)
			ctx.EmitMovRegReg(r22, d36.Reg)
			if d41.Imm.Int() >= -2147483648 && d41.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r22, int32(d41.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d41.Imm.Int()))
				ctx.EmitOrInt64(r22, scm.RegR11)
			}
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
			ctx.BindReg(r22, &d42)
		} else {
			r23 := ctx.AllocRegExcept(d36.Reg, d41.Reg)
			ctx.EmitMovRegReg(r23, d36.Reg)
			ctx.EmitOrInt64(r23, d41.Reg)
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
			ctx.BindReg(r23, &d42)
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
			r24 := ctx.AllocRegExcept(d43.Reg)
			ctx.EmitMovRegReg(r24, d43.Reg)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d44)
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
			r25 := ctx.AllocRegExcept(d43.Reg, d29.Reg)
			ctx.EmitMovRegReg(r25, d43.Reg)
			ctx.EmitSubInt64(r25, d29.Reg)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
			ctx.BindReg(r25, &d44)
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
			r26 := ctx.AllocRegExcept(d42.Reg)
			ctx.EmitMovRegReg(r26, d42.Reg)
			ctx.EmitShrRegImm8(r26, uint8(d44.Imm.Int()))
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
			ctx.BindReg(r26, &d45)
		} else {
			{
				shiftSrc := d42.Reg
				r27 := ctx.AllocRegExcept(d42.Reg, d44.Reg)
				ctx.EmitMovRegReg(r27, d42.Reg)
				shiftSrc = r27
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
			r28 := ctx.AllocReg()
			ctx.EmitMovRegReg(r28, d45.Reg)
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
			ctx.BindReg(r28, &d46)
		}
		ctx.FreeDesc(&d45)
		var d47 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56)
			r29 := ctx.AllocReg()
			ctx.EmitMovRegMem(r29, thisptr.Reg, off)
			d47 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r29}
			ctx.BindReg(r29, &d47)
		}
		ctx.EnsureDesc(&d46)
		ctx.EnsureDesc(&d47)
		ctx.EnsureDescsTogether(&d46, &d47)
		var d48 scm.JITValueDesc
		if d46.Loc == scm.LocImm && d47.Loc == scm.LocImm {
			d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d46.Imm.Int() + d47.Imm.Int())}
		} else if d47.Loc == scm.LocImm && d47.Imm.Int() == 0 {
			r30 := ctx.AllocRegExcept(d46.Reg)
			ctx.EmitMovRegReg(r30, d46.Reg)
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
			ctx.BindReg(r30, &d48)
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
			r31 := ctx.AllocRegExcept(d46.Reg, d47.Reg)
			ctx.EmitMovRegReg(r31, d46.Reg)
			ctx.EmitAddInt64(r31, d47.Reg)
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
			ctx.BindReg(r31, &d48)
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
			r32 := ctx.AllocReg()
			ctx.EmitMovRegReg(r32, d48.Reg)
			ctx.EmitShlRegImm8(r32, 32)
			ctx.EmitShrRegImm8(r32, 32)
			d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
			ctx.BindReg(r32, &d49)
		}
		ctx.FreeDesc(&d48)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d49)
		ctx.EnsureDescsTogether(&idxInt, &d49)
		var d50 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm && d49.Loc == scm.LocImm {
			d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d49.Imm.Int()))}
		} else if d49.Loc == scm.LocImm {
			r33 := ctx.AllocRegExcept(idxInt.Reg)
			if d49.Imm.Int() >= -2147483648 && d49.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(idxInt.Reg, int32(d49.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d49.Imm.Int()))
				ctx.EmitCmpInt64(idxInt.Reg, scm.RegR11)
			}
			d50 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r33, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r33, &d50)
		} else if idxInt.Loc == scm.LocImm {
			r34 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d49.Reg)
			d50 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r34, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r34, &d50)
		} else {
			r35 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitCmpInt64(idxInt.Reg, d49.Reg)
			d50 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r35, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r35, &d50)
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
		if bbs[5].Rendered {
			ctx.EmitJmp(lbl6)
		}
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
			ctx.FlushRegisterMoves()
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
			r36 := ctx.AllocReg()
			ctx.EmitMovRegReg(r36, d8.Reg)
			ctx.EmitShlRegImm8(r36, 32)
			ctx.EmitShrRegImm8(r36, 32)
			d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
			ctx.BindReg(r36, &d160)
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
			r37 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r37, thisptr.Reg, off)
			d162 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r37}
			ctx.BindReg(r37, &d162)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d162)
		ctx.EnsureDesc(&d162)
		var d163 scm.JITValueDesc
		if d162.Loc == scm.LocImm {
			d163 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d162.Imm.Int()))))}
		} else {
			r38 := ctx.AllocReg()
			ctx.EmitMovRegReg(r38, d162.Reg)
			ctx.EmitShlRegImm8(r38, 56)
			ctx.EmitShrRegImm8(r38, 56)
			d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
			ctx.BindReg(r38, &d163)
		}
		ctx.FreeDesc(&d162)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d161)
		ctx.EnsureDesc(&d161)
		var d164 scm.JITValueDesc
		if d161.Loc == scm.LocImm {
			d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d161.Imm.Int()))))}
		} else {
			r39 := ctx.AllocReg()
			ctx.EmitMovRegReg(r39, d161.Reg)
			ctx.EmitShlRegImm8(r39, 32)
			ctx.EmitShrRegImm8(r39, 32)
			d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
			ctx.BindReg(r39, &d164)
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
			r40 := ctx.AllocRegExcept(d164.Reg, d163.Reg)
			ctx.EmitMovRegReg(r40, d164.Reg)
			ctx.EmitImulInt64(r40, d163.Reg)
			d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
			ctx.BindReg(r40, &d165)
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
			r41 := ctx.AllocRegExcept(d165.Reg)
			ctx.EmitMovRegReg(r41, d165.Reg)
			ctx.EmitShrRegImm8(r41, 6)
			d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
			ctx.BindReg(r41, &d166)
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
			r42 := ctx.AllocRegExcept(d165.Reg)
			ctx.EmitMovRegReg(r42, d165.Reg)
			ctx.EmitAndRegImm32(r42, 63)
			d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
			ctx.BindReg(r42, &d167)
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
			r43 := ctx.AllocReg()
			r44 := ctx.AllocRegExcept(r43)
			r45 := ctx.AllocRegExcept(r43, r44)
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
			ctx.EmitMovRegMem(r43, thisptr.Reg, off)
			ctx.EmitMovRegMem(r44, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r45, thisptr.Reg, off+16)
			d168 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r43, Reg2: r44, Reg3: r45}
			ctx.BindReg(r43, &d168)
			ctx.BindReg(r44, &d168)
			ctx.BindReg(r45, &d168)
			ctx.BindReg(r43, &d168)
			ctx.BindReg(r44, &d168)
			ctx.BindReg(r45, &d168)
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
			r46 := ctx.AllocRegExcept(d169.Reg)
			ctx.EmitMovRegReg(r46, d169.Reg)
			ctx.EmitShlRegImm8(r46, uint8(d167.Imm.Int()))
			d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
			ctx.BindReg(r46, &d170)
		} else {
			{
				shiftSrc := d169.Reg
				r47 := ctx.AllocRegExcept(d169.Reg, d167.Reg)
				ctx.EmitMovRegReg(r47, d169.Reg)
				shiftSrc = r47
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
			r48 := ctx.AllocRegExcept(d173.Reg)
			ctx.EmitMovRegReg(r48, d173.Reg)
			d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
			ctx.BindReg(r48, &d174)
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
			r49 := ctx.AllocRegExcept(d173.Reg, d167.Reg)
			ctx.EmitMovRegReg(r49, d173.Reg)
			ctx.EmitSubInt64(r49, d167.Reg)
			d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
			ctx.BindReg(r49, &d174)
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
			r50 := ctx.AllocRegExcept(d172.Reg)
			ctx.EmitMovRegReg(r50, d172.Reg)
			ctx.EmitShrRegImm8(r50, uint8(d174.Imm.Int()))
			d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
			ctx.BindReg(r50, &d175)
		} else {
			{
				shiftSrc := d172.Reg
				r51 := ctx.AllocRegExcept(d172.Reg, d174.Reg)
				ctx.EmitMovRegReg(r51, d172.Reg)
				shiftSrc = r51
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
			r52 := ctx.AllocRegExcept(d170.Reg)
			ctx.EmitMovRegReg(r52, d170.Reg)
			d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
			ctx.BindReg(r52, &d176)
		} else if d170.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d175.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d170.Imm.Int()))
			ctx.EmitOrInt64(scratch, d175.Reg)
			d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d176)
		} else if d175.Loc == scm.LocImm {
			r53 := ctx.AllocRegExcept(d170.Reg)
			ctx.EmitMovRegReg(r53, d170.Reg)
			if d175.Imm.Int() >= -2147483648 && d175.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r53, int32(d175.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d175.Imm.Int()))
				ctx.EmitOrInt64(r53, scm.RegR11)
			}
			d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
			ctx.BindReg(r53, &d176)
		} else {
			r54 := ctx.AllocRegExcept(d170.Reg, d175.Reg)
			ctx.EmitMovRegReg(r54, d170.Reg)
			ctx.EmitOrInt64(r54, d175.Reg)
			d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
			ctx.BindReg(r54, &d176)
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
			r55 := ctx.AllocRegExcept(d177.Reg)
			ctx.EmitMovRegReg(r55, d177.Reg)
			d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
			ctx.BindReg(r55, &d178)
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
			r56 := ctx.AllocRegExcept(d177.Reg, d163.Reg)
			ctx.EmitMovRegReg(r56, d177.Reg)
			ctx.EmitSubInt64(r56, d163.Reg)
			d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r56}
			ctx.BindReg(r56, &d178)
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
			r57 := ctx.AllocRegExcept(d176.Reg)
			ctx.EmitMovRegReg(r57, d176.Reg)
			ctx.EmitShrRegImm8(r57, uint8(d178.Imm.Int()))
			d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r57}
			ctx.BindReg(r57, &d179)
		} else {
			{
				shiftSrc := d176.Reg
				r58 := ctx.AllocRegExcept(d176.Reg, d178.Reg)
				ctx.EmitMovRegReg(r58, d176.Reg)
				shiftSrc = r58
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
		ctx.StabilizeDescForControlFlow(&d179)
		var d180 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 80
			val := *(*bool)(unsafe.Pointer(fieldAddr))
			d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 80)
			r59 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r59, thisptr.Reg, off)
			d180 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r59}
			ctx.BindReg(r59, &d180)
		}
		d181 = d180
		ctx.EnsureDesc(&d181)
		if d181.Loc != scm.LocImm && d181.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d181.Loc == scm.LocImm {
			if d181.Imm.Bool() {
				if ps.General {
				}
				ps182 := scm.PhiState{General: ps.General}
				ps182.OverlayValues = make([]scm.JITValueDesc, 182)
				ps182.OverlayValues[5] = d5
				ps182.OverlayValues[6] = d6
				ps182.OverlayValues[7] = d7
				ps182.OverlayValues[8] = d8
				ps182.OverlayValues[9] = d9
				ps182.OverlayValues[10] = d10
				ps182.OverlayValues[11] = d11
				ps182.OverlayValues[12] = d12
				ps182.OverlayValues[13] = d13
				ps182.OverlayValues[14] = d14
				ps182.OverlayValues[15] = d15
				ps182.OverlayValues[16] = d16
				ps182.OverlayValues[17] = d17
				ps182.OverlayValues[18] = d18
				ps182.OverlayValues[19] = d19
				ps182.OverlayValues[21] = d21
				ps182.OverlayValues[22] = d22
				ps182.OverlayValues[23] = d23
				ps182.OverlayValues[24] = d24
				ps182.OverlayValues[25] = d25
				ps182.OverlayValues[26] = d26
				ps182.OverlayValues[27] = d27
				ps182.OverlayValues[28] = d28
				ps182.OverlayValues[29] = d29
				ps182.OverlayValues[30] = d30
				ps182.OverlayValues[31] = d31
				ps182.OverlayValues[32] = d32
				ps182.OverlayValues[33] = d33
				ps182.OverlayValues[34] = d34
				ps182.OverlayValues[35] = d35
				ps182.OverlayValues[36] = d36
				ps182.OverlayValues[37] = d37
				ps182.OverlayValues[38] = d38
				ps182.OverlayValues[39] = d39
				ps182.OverlayValues[40] = d40
				ps182.OverlayValues[41] = d41
				ps182.OverlayValues[42] = d42
				ps182.OverlayValues[43] = d43
				ps182.OverlayValues[44] = d44
				ps182.OverlayValues[45] = d45
				ps182.OverlayValues[46] = d46
				ps182.OverlayValues[47] = d47
				ps182.OverlayValues[48] = d48
				ps182.OverlayValues[49] = d49
				ps182.OverlayValues[50] = d50
				ps182.OverlayValues[51] = d51
				ps182.OverlayValues[54] = d54
				ps182.OverlayValues[55] = d55
				ps182.OverlayValues[56] = d56
				ps182.OverlayValues[159] = d159
				ps182.OverlayValues[160] = d160
				ps182.OverlayValues[161] = d161
				ps182.OverlayValues[162] = d162
				ps182.OverlayValues[163] = d163
				ps182.OverlayValues[164] = d164
				ps182.OverlayValues[165] = d165
				ps182.OverlayValues[166] = d166
				ps182.OverlayValues[167] = d167
				ps182.OverlayValues[168] = d168
				ps182.OverlayValues[169] = d169
				ps182.OverlayValues[170] = d170
				ps182.OverlayValues[171] = d171
				ps182.OverlayValues[172] = d172
				ps182.OverlayValues[173] = d173
				ps182.OverlayValues[174] = d174
				ps182.OverlayValues[175] = d175
				ps182.OverlayValues[176] = d176
				ps182.OverlayValues[177] = d177
				ps182.OverlayValues[178] = d178
				ps182.OverlayValues[179] = d179
				ps182.OverlayValues[180] = d180
				ps182.OverlayValues[181] = d181
				return bbs[13].RenderPS(ps182)
			}
			if ps.General {
			}
			ps183 := scm.PhiState{General: ps.General}
			ps183.OverlayValues = make([]scm.JITValueDesc, 182)
			ps183.OverlayValues[5] = d5
			ps183.OverlayValues[6] = d6
			ps183.OverlayValues[7] = d7
			ps183.OverlayValues[8] = d8
			ps183.OverlayValues[9] = d9
			ps183.OverlayValues[10] = d10
			ps183.OverlayValues[11] = d11
			ps183.OverlayValues[12] = d12
			ps183.OverlayValues[13] = d13
			ps183.OverlayValues[14] = d14
			ps183.OverlayValues[15] = d15
			ps183.OverlayValues[16] = d16
			ps183.OverlayValues[17] = d17
			ps183.OverlayValues[18] = d18
			ps183.OverlayValues[19] = d19
			ps183.OverlayValues[21] = d21
			ps183.OverlayValues[22] = d22
			ps183.OverlayValues[23] = d23
			ps183.OverlayValues[24] = d24
			ps183.OverlayValues[25] = d25
			ps183.OverlayValues[26] = d26
			ps183.OverlayValues[27] = d27
			ps183.OverlayValues[28] = d28
			ps183.OverlayValues[29] = d29
			ps183.OverlayValues[30] = d30
			ps183.OverlayValues[31] = d31
			ps183.OverlayValues[32] = d32
			ps183.OverlayValues[33] = d33
			ps183.OverlayValues[34] = d34
			ps183.OverlayValues[35] = d35
			ps183.OverlayValues[36] = d36
			ps183.OverlayValues[37] = d37
			ps183.OverlayValues[38] = d38
			ps183.OverlayValues[39] = d39
			ps183.OverlayValues[40] = d40
			ps183.OverlayValues[41] = d41
			ps183.OverlayValues[42] = d42
			ps183.OverlayValues[43] = d43
			ps183.OverlayValues[44] = d44
			ps183.OverlayValues[45] = d45
			ps183.OverlayValues[46] = d46
			ps183.OverlayValues[47] = d47
			ps183.OverlayValues[48] = d48
			ps183.OverlayValues[49] = d49
			ps183.OverlayValues[50] = d50
			ps183.OverlayValues[51] = d51
			ps183.OverlayValues[54] = d54
			ps183.OverlayValues[55] = d55
			ps183.OverlayValues[56] = d56
			ps183.OverlayValues[159] = d159
			ps183.OverlayValues[160] = d160
			ps183.OverlayValues[161] = d161
			ps183.OverlayValues[162] = d162
			ps183.OverlayValues[163] = d163
			ps183.OverlayValues[164] = d164
			ps183.OverlayValues[165] = d165
			ps183.OverlayValues[166] = d166
			ps183.OverlayValues[167] = d167
			ps183.OverlayValues[168] = d168
			ps183.OverlayValues[169] = d169
			ps183.OverlayValues[170] = d170
			ps183.OverlayValues[171] = d171
			ps183.OverlayValues[172] = d172
			ps183.OverlayValues[173] = d173
			ps183.OverlayValues[174] = d174
			ps183.OverlayValues[175] = d175
			ps183.OverlayValues[176] = d176
			ps183.OverlayValues[177] = d177
			ps183.OverlayValues[178] = d178
			ps183.OverlayValues[179] = d179
			ps183.OverlayValues[180] = d180
			ps183.OverlayValues[181] = d181
			return bbs[12].RenderPS(ps183)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d184 := ps.PhiValues[0]
				ctx.EnsureDesc(&d184)
				ctx.EmitStoreToStack(d184, int32(bbs[2].PhiBase)+int32(0))
			}
			ps.General = true
			return bbs[2].RenderPS(ps)
		}
		ctx.EmitCmpRegImm32(d181.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl14)
		if bbs[12].Rendered {
			ctx.EmitJmp(lbl13)
		}
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
		snap198 := d18
		snap199 := d19
		snap200 := d21
		snap201 := d22
		snap202 := d23
		snap203 := d24
		snap204 := d25
		snap205 := d26
		snap206 := d27
		snap207 := d28
		snap208 := d29
		snap209 := d30
		snap210 := d31
		snap211 := d32
		snap212 := d33
		snap213 := d34
		snap214 := d35
		snap215 := d36
		snap216 := d37
		snap217 := d38
		snap218 := d39
		snap219 := d40
		snap220 := d41
		snap221 := d42
		snap222 := d43
		snap223 := d44
		snap224 := d45
		snap225 := d46
		snap226 := d47
		snap227 := d48
		snap228 := d49
		snap229 := d50
		snap230 := d51
		snap231 := d54
		snap232 := d55
		snap233 := d56
		snap234 := d159
		snap235 := d160
		snap236 := d161
		snap237 := d162
		snap238 := d163
		snap239 := d164
		snap240 := d165
		snap241 := d166
		snap242 := d167
		snap243 := d168
		snap244 := d169
		snap245 := d170
		snap246 := d171
		snap247 := d172
		snap248 := d173
		snap249 := d174
		snap250 := d175
		snap251 := d176
		snap252 := d177
		snap253 := d178
		snap254 := d179
		snap255 := d180
		snap256 := d181
		snap257 := d184
		alloc258 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc258)
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
		d18 = snap198
		d19 = snap199
		d21 = snap200
		d22 = snap201
		d23 = snap202
		d24 = snap203
		d25 = snap204
		d26 = snap205
		d27 = snap206
		d28 = snap207
		d29 = snap208
		d30 = snap209
		d31 = snap210
		d32 = snap211
		d33 = snap212
		d34 = snap213
		d35 = snap214
		d36 = snap215
		d37 = snap216
		d38 = snap217
		d39 = snap218
		d40 = snap219
		d41 = snap220
		d42 = snap221
		d43 = snap222
		d44 = snap223
		d45 = snap224
		d46 = snap225
		d47 = snap226
		d48 = snap227
		d49 = snap228
		d50 = snap229
		d51 = snap230
		d54 = snap231
		d55 = snap232
		d56 = snap233
		d159 = snap234
		d160 = snap235
		d161 = snap236
		d162 = snap237
		d163 = snap238
		d164 = snap239
		d165 = snap240
		d166 = snap241
		d167 = snap242
		d168 = snap243
		d169 = snap244
		d170 = snap245
		d171 = snap246
		d172 = snap247
		d173 = snap248
		d174 = snap249
		d175 = snap250
		d176 = snap251
		d177 = snap252
		d178 = snap253
		d179 = snap254
		d180 = snap255
		d181 = snap256
		d184 = snap257
		ctx.RestoreAllocState(alloc258)
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
		d18 = snap198
		d19 = snap199
		d21 = snap200
		d22 = snap201
		d23 = snap202
		d24 = snap203
		d25 = snap204
		d26 = snap205
		d27 = snap206
		d28 = snap207
		d29 = snap208
		d30 = snap209
		d31 = snap210
		d32 = snap211
		d33 = snap212
		d34 = snap213
		d35 = snap214
		d36 = snap215
		d37 = snap216
		d38 = snap217
		d39 = snap218
		d40 = snap219
		d41 = snap220
		d42 = snap221
		d43 = snap222
		d44 = snap223
		d45 = snap224
		d46 = snap225
		d47 = snap226
		d48 = snap227
		d49 = snap228
		d50 = snap229
		d51 = snap230
		d54 = snap231
		d55 = snap232
		d56 = snap233
		d159 = snap234
		d160 = snap235
		d161 = snap236
		d162 = snap237
		d163 = snap238
		d164 = snap239
		d165 = snap240
		d166 = snap241
		d167 = snap242
		d168 = snap243
		d169 = snap244
		d170 = snap245
		d171 = snap246
		d172 = snap247
		d173 = snap248
		d174 = snap249
		d175 = snap250
		d176 = snap251
		d177 = snap252
		d178 = snap253
		d179 = snap254
		d180 = snap255
		d181 = snap256
		d184 = snap257
		ps259 := scm.PhiState{General: true}
		ps259.OverlayValues = make([]scm.JITValueDesc, 185)
		ps259.OverlayValues[5] = d5
		ps259.OverlayValues[6] = d6
		ps259.OverlayValues[7] = d7
		ps259.OverlayValues[8] = d8
		ps259.OverlayValues[9] = d9
		ps259.OverlayValues[10] = d10
		ps259.OverlayValues[11] = d11
		ps259.OverlayValues[12] = d12
		ps259.OverlayValues[13] = d13
		ps259.OverlayValues[14] = d14
		ps259.OverlayValues[15] = d15
		ps259.OverlayValues[16] = d16
		ps259.OverlayValues[17] = d17
		ps259.OverlayValues[18] = d18
		ps259.OverlayValues[19] = d19
		ps259.OverlayValues[21] = d21
		ps259.OverlayValues[22] = d22
		ps259.OverlayValues[23] = d23
		ps259.OverlayValues[24] = d24
		ps259.OverlayValues[25] = d25
		ps259.OverlayValues[26] = d26
		ps259.OverlayValues[27] = d27
		ps259.OverlayValues[28] = d28
		ps259.OverlayValues[29] = d29
		ps259.OverlayValues[30] = d30
		ps259.OverlayValues[31] = d31
		ps259.OverlayValues[32] = d32
		ps259.OverlayValues[33] = d33
		ps259.OverlayValues[34] = d34
		ps259.OverlayValues[35] = d35
		ps259.OverlayValues[36] = d36
		ps259.OverlayValues[37] = d37
		ps259.OverlayValues[38] = d38
		ps259.OverlayValues[39] = d39
		ps259.OverlayValues[40] = d40
		ps259.OverlayValues[41] = d41
		ps259.OverlayValues[42] = d42
		ps259.OverlayValues[43] = d43
		ps259.OverlayValues[44] = d44
		ps259.OverlayValues[45] = d45
		ps259.OverlayValues[46] = d46
		ps259.OverlayValues[47] = d47
		ps259.OverlayValues[48] = d48
		ps259.OverlayValues[49] = d49
		ps259.OverlayValues[50] = d50
		ps259.OverlayValues[51] = d51
		ps259.OverlayValues[54] = d54
		ps259.OverlayValues[55] = d55
		ps259.OverlayValues[56] = d56
		ps259.OverlayValues[159] = d159
		ps259.OverlayValues[160] = d160
		ps259.OverlayValues[161] = d161
		ps259.OverlayValues[162] = d162
		ps259.OverlayValues[163] = d163
		ps259.OverlayValues[164] = d164
		ps259.OverlayValues[165] = d165
		ps259.OverlayValues[166] = d166
		ps259.OverlayValues[167] = d167
		ps259.OverlayValues[168] = d168
		ps259.OverlayValues[169] = d169
		ps259.OverlayValues[170] = d170
		ps259.OverlayValues[171] = d171
		ps259.OverlayValues[172] = d172
		ps259.OverlayValues[173] = d173
		ps259.OverlayValues[174] = d174
		ps259.OverlayValues[175] = d175
		ps259.OverlayValues[176] = d176
		ps259.OverlayValues[177] = d177
		ps259.OverlayValues[178] = d178
		ps259.OverlayValues[179] = d179
		ps259.OverlayValues[180] = d180
		ps259.OverlayValues[181] = d181
		ps259.OverlayValues[184] = d184
		ps260 := scm.PhiState{General: true}
		ps260.OverlayValues = make([]scm.JITValueDesc, 185)
		ps260.OverlayValues[5] = d5
		ps260.OverlayValues[6] = d6
		ps260.OverlayValues[7] = d7
		ps260.OverlayValues[8] = d8
		ps260.OverlayValues[9] = d9
		ps260.OverlayValues[10] = d10
		ps260.OverlayValues[11] = d11
		ps260.OverlayValues[12] = d12
		ps260.OverlayValues[13] = d13
		ps260.OverlayValues[14] = d14
		ps260.OverlayValues[15] = d15
		ps260.OverlayValues[16] = d16
		ps260.OverlayValues[17] = d17
		ps260.OverlayValues[18] = d18
		ps260.OverlayValues[19] = d19
		ps260.OverlayValues[21] = d21
		ps260.OverlayValues[22] = d22
		ps260.OverlayValues[23] = d23
		ps260.OverlayValues[24] = d24
		ps260.OverlayValues[25] = d25
		ps260.OverlayValues[26] = d26
		ps260.OverlayValues[27] = d27
		ps260.OverlayValues[28] = d28
		ps260.OverlayValues[29] = d29
		ps260.OverlayValues[30] = d30
		ps260.OverlayValues[31] = d31
		ps260.OverlayValues[32] = d32
		ps260.OverlayValues[33] = d33
		ps260.OverlayValues[34] = d34
		ps260.OverlayValues[35] = d35
		ps260.OverlayValues[36] = d36
		ps260.OverlayValues[37] = d37
		ps260.OverlayValues[38] = d38
		ps260.OverlayValues[39] = d39
		ps260.OverlayValues[40] = d40
		ps260.OverlayValues[41] = d41
		ps260.OverlayValues[42] = d42
		ps260.OverlayValues[43] = d43
		ps260.OverlayValues[44] = d44
		ps260.OverlayValues[45] = d45
		ps260.OverlayValues[46] = d46
		ps260.OverlayValues[47] = d47
		ps260.OverlayValues[48] = d48
		ps260.OverlayValues[49] = d49
		ps260.OverlayValues[50] = d50
		ps260.OverlayValues[51] = d51
		ps260.OverlayValues[54] = d54
		ps260.OverlayValues[55] = d55
		ps260.OverlayValues[56] = d56
		ps260.OverlayValues[159] = d159
		ps260.OverlayValues[160] = d160
		ps260.OverlayValues[161] = d161
		ps260.OverlayValues[162] = d162
		ps260.OverlayValues[163] = d163
		ps260.OverlayValues[164] = d164
		ps260.OverlayValues[165] = d165
		ps260.OverlayValues[166] = d166
		ps260.OverlayValues[167] = d167
		ps260.OverlayValues[168] = d168
		ps260.OverlayValues[169] = d169
		ps260.OverlayValues[170] = d170
		ps260.OverlayValues[171] = d171
		ps260.OverlayValues[172] = d172
		ps260.OverlayValues[173] = d173
		ps260.OverlayValues[174] = d174
		ps260.OverlayValues[175] = d175
		ps260.OverlayValues[176] = d176
		ps260.OverlayValues[177] = d177
		ps260.OverlayValues[178] = d178
		ps260.OverlayValues[179] = d179
		ps260.OverlayValues[180] = d180
		ps260.OverlayValues[181] = d181
		ps260.OverlayValues[184] = d184
		snap261 := d5
		snap262 := d6
		snap263 := d7
		snap264 := d8
		snap265 := d9
		snap266 := d10
		snap267 := d11
		snap268 := d12
		snap269 := d13
		snap270 := d14
		snap271 := d15
		snap272 := d16
		snap273 := d17
		snap274 := d18
		snap275 := d19
		snap276 := d21
		snap277 := d22
		snap278 := d23
		snap279 := d24
		snap280 := d25
		snap281 := d26
		snap282 := d27
		snap283 := d28
		snap284 := d29
		snap285 := d30
		snap286 := d31
		snap287 := d32
		snap288 := d33
		snap289 := d34
		snap290 := d35
		snap291 := d36
		snap292 := d37
		snap293 := d38
		snap294 := d39
		snap295 := d40
		snap296 := d41
		snap297 := d42
		snap298 := d43
		snap299 := d44
		snap300 := d45
		snap301 := d46
		snap302 := d47
		snap303 := d48
		snap304 := d49
		snap305 := d50
		snap306 := d51
		snap307 := d54
		snap308 := d55
		snap309 := d56
		snap310 := d159
		snap311 := d160
		snap312 := d161
		snap313 := d162
		snap314 := d163
		snap315 := d164
		snap316 := d165
		snap317 := d166
		snap318 := d167
		snap319 := d168
		snap320 := d169
		snap321 := d170
		snap322 := d171
		snap323 := d172
		snap324 := d173
		snap325 := d174
		snap326 := d175
		snap327 := d176
		snap328 := d177
		snap329 := d178
		snap330 := d179
		snap331 := d180
		snap332 := d181
		snap333 := d184
		alloc334 := ctx.SnapshotAllocState()
		if !bbs[12].Rendered {
			bbs[12].RenderPS(ps260)
		}
		ctx.RestoreAllocState(alloc334)
		d5 = snap261
		d6 = snap262
		d7 = snap263
		d8 = snap264
		d9 = snap265
		d10 = snap266
		d11 = snap267
		d12 = snap268
		d13 = snap269
		d14 = snap270
		d15 = snap271
		d16 = snap272
		d17 = snap273
		d18 = snap274
		d19 = snap275
		d21 = snap276
		d22 = snap277
		d23 = snap278
		d24 = snap279
		d25 = snap280
		d26 = snap281
		d27 = snap282
		d28 = snap283
		d29 = snap284
		d30 = snap285
		d31 = snap286
		d32 = snap287
		d33 = snap288
		d34 = snap289
		d35 = snap290
		d36 = snap291
		d37 = snap292
		d38 = snap293
		d39 = snap294
		d40 = snap295
		d41 = snap296
		d42 = snap297
		d43 = snap298
		d44 = snap299
		d45 = snap300
		d46 = snap301
		d47 = snap302
		d48 = snap303
		d49 = snap304
		d50 = snap305
		d51 = snap306
		d54 = snap307
		d55 = snap308
		d56 = snap309
		d159 = snap310
		d160 = snap311
		d161 = snap312
		d162 = snap313
		d163 = snap314
		d164 = snap315
		d165 = snap316
		d166 = snap317
		d167 = snap318
		d168 = snap319
		d169 = snap320
		d170 = snap321
		d171 = snap322
		d172 = snap323
		d173 = snap324
		d174 = snap325
		d175 = snap326
		d176 = snap327
		d177 = snap328
		d178 = snap329
		d179 = snap330
		d180 = snap331
		d181 = snap332
		d184 = snap333
		if !bbs[13].Rendered {
			return bbs[13].RenderPS(ps259)
		}
		return result
		ctx.FreeDesc(&d180)
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
			ctx.FlushRegisterMoves()
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
		if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
			d184 = ps.OverlayValues[184]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d5)
		var d335 scm.JITValueDesc
		if d5.Loc == scm.LocImm {
			d335 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitSubRegImm32Low(scratch, int32(1))
			d335 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d335)
		}
		if d335.Loc == scm.LocReg && d5.Loc == scm.LocReg && d335.Reg == d5.Reg {
			ctx.TransferReg(d5.Reg)
			d5.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d335)
		ctx.EmitStoreToStack(d335, int32(bbs[4].PhiBase)+int32(32))
		ctx.StabilizeDescForControlFlow(&d335)
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d5)
		var d336 scm.JITValueDesc
		if d5.Loc == scm.LocImm {
			d336 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitSubRegImm32Low(scratch, int32(1))
			d336 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d336)
		}
		if d336.Loc == scm.LocReg && d5.Loc == scm.LocReg && d336.Reg == d5.Reg {
			ctx.TransferReg(d5.Reg)
			d5.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d336)
		ctx.EmitStoreToStack(d336, int32(bbs[4].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d336)
		if ps.General {
			ctx.SyncDesc(&d6)
			if d6.Loc == scm.LocReg || d6.Loc == scm.LocFPReg {
				ctx.ProtectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.ProtectReg(d6.Reg)
				ctx.ProtectReg(d6.Reg2)
			}
			d337 = d6
			if d337.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d337)
			d338 = d337
			if d338.Loc == scm.LocImm {
				d338 = scm.JITValueDesc{Loc: scm.LocImm, Type: d338.Type, Imm: scm.NewInt(int64(uint64(d338.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d338.Reg, 32)
				ctx.EmitShrRegImm8(d338.Reg, 32)
			}
			ctx.EmitStoreToStack(d338, int32(bbs[4].PhiBase)+int32(16))
			if d6.Loc == scm.LocReg || d6.Loc == scm.LocFPReg {
				ctx.UnprotectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d6.Reg)
				ctx.UnprotectReg(d6.Reg2)
			}
		}
		ps339 := scm.PhiState{General: ps.General}
		ps339.OverlayValues = make([]scm.JITValueDesc, 339)
		ps339.OverlayValues[5] = d5
		ps339.OverlayValues[6] = d6
		ps339.OverlayValues[7] = d7
		ps339.OverlayValues[8] = d8
		ps339.OverlayValues[9] = d9
		ps339.OverlayValues[10] = d10
		ps339.OverlayValues[11] = d11
		ps339.OverlayValues[12] = d12
		ps339.OverlayValues[13] = d13
		ps339.OverlayValues[14] = d14
		ps339.OverlayValues[15] = d15
		ps339.OverlayValues[16] = d16
		ps339.OverlayValues[17] = d17
		ps339.OverlayValues[18] = d18
		ps339.OverlayValues[19] = d19
		ps339.OverlayValues[21] = d21
		ps339.OverlayValues[22] = d22
		ps339.OverlayValues[23] = d23
		ps339.OverlayValues[24] = d24
		ps339.OverlayValues[25] = d25
		ps339.OverlayValues[26] = d26
		ps339.OverlayValues[27] = d27
		ps339.OverlayValues[28] = d28
		ps339.OverlayValues[29] = d29
		ps339.OverlayValues[30] = d30
		ps339.OverlayValues[31] = d31
		ps339.OverlayValues[32] = d32
		ps339.OverlayValues[33] = d33
		ps339.OverlayValues[34] = d34
		ps339.OverlayValues[35] = d35
		ps339.OverlayValues[36] = d36
		ps339.OverlayValues[37] = d37
		ps339.OverlayValues[38] = d38
		ps339.OverlayValues[39] = d39
		ps339.OverlayValues[40] = d40
		ps339.OverlayValues[41] = d41
		ps339.OverlayValues[42] = d42
		ps339.OverlayValues[43] = d43
		ps339.OverlayValues[44] = d44
		ps339.OverlayValues[45] = d45
		ps339.OverlayValues[46] = d46
		ps339.OverlayValues[47] = d47
		ps339.OverlayValues[48] = d48
		ps339.OverlayValues[49] = d49
		ps339.OverlayValues[50] = d50
		ps339.OverlayValues[51] = d51
		ps339.OverlayValues[54] = d54
		ps339.OverlayValues[55] = d55
		ps339.OverlayValues[56] = d56
		ps339.OverlayValues[159] = d159
		ps339.OverlayValues[160] = d160
		ps339.OverlayValues[161] = d161
		ps339.OverlayValues[162] = d162
		ps339.OverlayValues[163] = d163
		ps339.OverlayValues[164] = d164
		ps339.OverlayValues[165] = d165
		ps339.OverlayValues[166] = d166
		ps339.OverlayValues[167] = d167
		ps339.OverlayValues[168] = d168
		ps339.OverlayValues[169] = d169
		ps339.OverlayValues[170] = d170
		ps339.OverlayValues[171] = d171
		ps339.OverlayValues[172] = d172
		ps339.OverlayValues[173] = d173
		ps339.OverlayValues[174] = d174
		ps339.OverlayValues[175] = d175
		ps339.OverlayValues[176] = d176
		ps339.OverlayValues[177] = d177
		ps339.OverlayValues[178] = d178
		ps339.OverlayValues[179] = d179
		ps339.OverlayValues[180] = d180
		ps339.OverlayValues[181] = d181
		ps339.OverlayValues[184] = d184
		ps339.OverlayValues[335] = d335
		ps339.OverlayValues[336] = d336
		ps339.OverlayValues[337] = d337
		ps339.OverlayValues[338] = d338
		ps339.PhiValues = make([]scm.JITValueDesc, 3)
		d340 = d6
		ps339.PhiValues[1] = d340
		if ps339.General && bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
			return result
		}
		return bbs[4].RenderPS(ps339)
		return result
	}
	bbs[4].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d341 := ps.PhiValues[0]
				ctx.EnsureDesc(&d341)
				ctx.EmitStoreToStack(d341, int32(bbs[4].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d342 := ps.PhiValues[1]
				ctx.EnsureDesc(&d342)
				ctx.EmitStoreToStack(d342, int32(bbs[4].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d343 := ps.PhiValues[2]
				ctx.EnsureDesc(&d343)
				ctx.EmitStoreToStack(d343, int32(bbs[4].PhiBase)+int32(32))
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
			ctx.FlushRegisterMoves()
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
		if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
			d184 = ps.OverlayValues[184]
		}
		if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != scm.LocNone {
			d335 = ps.OverlayValues[335]
		}
		if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != scm.LocNone {
			d336 = ps.OverlayValues[336]
		}
		if len(ps.OverlayValues) > 337 && ps.OverlayValues[337].Loc != scm.LocNone {
			d337 = ps.OverlayValues[337]
		}
		if len(ps.OverlayValues) > 338 && ps.OverlayValues[338].Loc != scm.LocNone {
			d338 = ps.OverlayValues[338]
		}
		if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != scm.LocNone {
			d340 = ps.OverlayValues[340]
		}
		if len(ps.OverlayValues) > 341 && ps.OverlayValues[341].Loc != scm.LocNone {
			d341 = ps.OverlayValues[341]
		}
		if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != scm.LocNone {
			d342 = ps.OverlayValues[342]
		}
		if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != scm.LocNone {
			d343 = ps.OverlayValues[343]
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
		var d344 scm.JITValueDesc
		if d10.Loc == scm.LocImm && d11.Loc == scm.LocImm {
			d344 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d10.Imm.Int()) == uint64(d11.Imm.Int()))}
		} else if d11.Loc == scm.LocImm {
			r60 := ctx.AllocRegExcept(d10.Reg)
			if d11.Imm.Int() >= -2147483648 && d11.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d10.Reg, int32(d11.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d11.Imm.Int()))
				ctx.EmitCmpInt64(d10.Reg, scm.RegR11)
			}
			d344 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r60, Condition: scm.CondEqual}
			ctx.BindReg(r60, &d344)
		} else if d10.Loc == scm.LocImm {
			r61 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d10.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d11.Reg)
			d344 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r61, Condition: scm.CondEqual}
			ctx.BindReg(r61, &d344)
		} else {
			r62 := ctx.AllocRegExcept(d10.Reg)
			ctx.EmitCmpInt64(d10.Reg, d11.Reg)
			d344 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r62, Condition: scm.CondEqual}
			ctx.BindReg(r62, &d344)
		}
		d345 = d344
		ctx.EnsureDesc(&d345)
		if d345.Loc != scm.LocImm && d345.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d345.Loc == scm.LocImm {
			if d345.Imm.Bool() {
				if ps.General {
					ctx.SyncDesc(&d10)
					if d10.Loc == scm.LocReg || d10.Loc == scm.LocFPReg {
						ctx.ProtectReg(d10.Reg)
					} else if d10.Loc == scm.LocRegPair {
						ctx.ProtectReg(d10.Reg)
						ctx.ProtectReg(d10.Reg2)
					}
					d346 = d10
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
					ctx.EmitStoreToStack(d347, int32(bbs[2].PhiBase)+int32(0))
					if d10.Loc == scm.LocReg || d10.Loc == scm.LocFPReg {
						ctx.UnprotectReg(d10.Reg)
					} else if d10.Loc == scm.LocRegPair {
						ctx.UnprotectReg(d10.Reg)
						ctx.UnprotectReg(d10.Reg2)
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
				ps348.OverlayValues[184] = d184
				ps348.OverlayValues[335] = d335
				ps348.OverlayValues[336] = d336
				ps348.OverlayValues[337] = d337
				ps348.OverlayValues[338] = d338
				ps348.OverlayValues[340] = d340
				ps348.OverlayValues[341] = d341
				ps348.OverlayValues[342] = d342
				ps348.OverlayValues[343] = d343
				ps348.OverlayValues[344] = d344
				ps348.OverlayValues[345] = d345
				ps348.OverlayValues[346] = d346
				ps348.OverlayValues[347] = d347
				ps348.PhiValues = make([]scm.JITValueDesc, 1)
				d349 = d10
				ps348.PhiValues[0] = d349
				return bbs[2].RenderPS(ps348)
			}
			if ps.General {
			}
			ps350 := scm.PhiState{General: ps.General}
			ps350.OverlayValues = make([]scm.JITValueDesc, 350)
			ps350.OverlayValues[5] = d5
			ps350.OverlayValues[6] = d6
			ps350.OverlayValues[7] = d7
			ps350.OverlayValues[8] = d8
			ps350.OverlayValues[9] = d9
			ps350.OverlayValues[10] = d10
			ps350.OverlayValues[11] = d11
			ps350.OverlayValues[12] = d12
			ps350.OverlayValues[13] = d13
			ps350.OverlayValues[14] = d14
			ps350.OverlayValues[15] = d15
			ps350.OverlayValues[16] = d16
			ps350.OverlayValues[17] = d17
			ps350.OverlayValues[18] = d18
			ps350.OverlayValues[19] = d19
			ps350.OverlayValues[21] = d21
			ps350.OverlayValues[22] = d22
			ps350.OverlayValues[23] = d23
			ps350.OverlayValues[24] = d24
			ps350.OverlayValues[25] = d25
			ps350.OverlayValues[26] = d26
			ps350.OverlayValues[27] = d27
			ps350.OverlayValues[28] = d28
			ps350.OverlayValues[29] = d29
			ps350.OverlayValues[30] = d30
			ps350.OverlayValues[31] = d31
			ps350.OverlayValues[32] = d32
			ps350.OverlayValues[33] = d33
			ps350.OverlayValues[34] = d34
			ps350.OverlayValues[35] = d35
			ps350.OverlayValues[36] = d36
			ps350.OverlayValues[37] = d37
			ps350.OverlayValues[38] = d38
			ps350.OverlayValues[39] = d39
			ps350.OverlayValues[40] = d40
			ps350.OverlayValues[41] = d41
			ps350.OverlayValues[42] = d42
			ps350.OverlayValues[43] = d43
			ps350.OverlayValues[44] = d44
			ps350.OverlayValues[45] = d45
			ps350.OverlayValues[46] = d46
			ps350.OverlayValues[47] = d47
			ps350.OverlayValues[48] = d48
			ps350.OverlayValues[49] = d49
			ps350.OverlayValues[50] = d50
			ps350.OverlayValues[51] = d51
			ps350.OverlayValues[54] = d54
			ps350.OverlayValues[55] = d55
			ps350.OverlayValues[56] = d56
			ps350.OverlayValues[159] = d159
			ps350.OverlayValues[160] = d160
			ps350.OverlayValues[161] = d161
			ps350.OverlayValues[162] = d162
			ps350.OverlayValues[163] = d163
			ps350.OverlayValues[164] = d164
			ps350.OverlayValues[165] = d165
			ps350.OverlayValues[166] = d166
			ps350.OverlayValues[167] = d167
			ps350.OverlayValues[168] = d168
			ps350.OverlayValues[169] = d169
			ps350.OverlayValues[170] = d170
			ps350.OverlayValues[171] = d171
			ps350.OverlayValues[172] = d172
			ps350.OverlayValues[173] = d173
			ps350.OverlayValues[174] = d174
			ps350.OverlayValues[175] = d175
			ps350.OverlayValues[176] = d176
			ps350.OverlayValues[177] = d177
			ps350.OverlayValues[178] = d178
			ps350.OverlayValues[179] = d179
			ps350.OverlayValues[180] = d180
			ps350.OverlayValues[181] = d181
			ps350.OverlayValues[184] = d184
			ps350.OverlayValues[335] = d335
			ps350.OverlayValues[336] = d336
			ps350.OverlayValues[337] = d337
			ps350.OverlayValues[338] = d338
			ps350.OverlayValues[340] = d340
			ps350.OverlayValues[341] = d341
			ps350.OverlayValues[342] = d342
			ps350.OverlayValues[343] = d343
			ps350.OverlayValues[344] = d344
			ps350.OverlayValues[345] = d345
			ps350.OverlayValues[346] = d346
			ps350.OverlayValues[347] = d347
			ps350.OverlayValues[349] = d349
			return bbs[6].RenderPS(ps350)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d351 := ps.PhiValues[0]
				ctx.EnsureDesc(&d351)
				ctx.EmitStoreToStack(d351, int32(bbs[4].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d352 := ps.PhiValues[1]
				ctx.EnsureDesc(&d352)
				ctx.EmitStoreToStack(d352, int32(bbs[4].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d353 := ps.PhiValues[2]
				ctx.EnsureDesc(&d353)
				ctx.EmitStoreToStack(d353, int32(bbs[4].PhiBase)+int32(32))
			}
			ps.General = true
			return bbs[4].RenderPS(ps)
		}
		lbl17 := ctx.ReserveLabel()
		ctx.EmitJump(d345.Condition, lbl17)
		ctx.EmitJmp(lbl7)
		ctx.FreeDesc(&d344)
		snap354 := d5
		snap355 := d6
		snap356 := d7
		snap357 := d8
		snap358 := d9
		snap359 := d10
		snap360 := d11
		snap361 := d12
		snap362 := d13
		snap363 := d14
		snap364 := d15
		snap365 := d16
		snap366 := d17
		snap367 := d18
		snap368 := d19
		snap369 := d21
		snap370 := d22
		snap371 := d23
		snap372 := d24
		snap373 := d25
		snap374 := d26
		snap375 := d27
		snap376 := d28
		snap377 := d29
		snap378 := d30
		snap379 := d31
		snap380 := d32
		snap381 := d33
		snap382 := d34
		snap383 := d35
		snap384 := d36
		snap385 := d37
		snap386 := d38
		snap387 := d39
		snap388 := d40
		snap389 := d41
		snap390 := d42
		snap391 := d43
		snap392 := d44
		snap393 := d45
		snap394 := d46
		snap395 := d47
		snap396 := d48
		snap397 := d49
		snap398 := d50
		snap399 := d51
		snap400 := d54
		snap401 := d55
		snap402 := d56
		snap403 := d159
		snap404 := d160
		snap405 := d161
		snap406 := d162
		snap407 := d163
		snap408 := d164
		snap409 := d165
		snap410 := d166
		snap411 := d167
		snap412 := d168
		snap413 := d169
		snap414 := d170
		snap415 := d171
		snap416 := d172
		snap417 := d173
		snap418 := d174
		snap419 := d175
		snap420 := d176
		snap421 := d177
		snap422 := d178
		snap423 := d179
		snap424 := d180
		snap425 := d181
		snap426 := d184
		snap427 := d335
		snap428 := d336
		snap429 := d337
		snap430 := d338
		snap431 := d340
		snap432 := d341
		snap433 := d342
		snap434 := d343
		snap435 := d344
		snap436 := d345
		snap437 := d346
		snap438 := d347
		snap439 := d349
		snap440 := d351
		snap441 := d352
		snap442 := d353
		alloc443 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl17)
		ctx.SyncDesc(&d10)
		if d10.Loc == scm.LocReg || d10.Loc == scm.LocFPReg {
			ctx.ProtectReg(d10.Reg)
		} else if d10.Loc == scm.LocRegPair {
			ctx.ProtectReg(d10.Reg)
			ctx.ProtectReg(d10.Reg2)
		}
		d444 = d10
		if d444.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d444)
		d445 = d444
		if d445.Loc == scm.LocImm {
			d445 = scm.JITValueDesc{Loc: scm.LocImm, Type: d445.Type, Imm: scm.NewInt(int64(uint64(d445.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d445.Reg, 32)
			ctx.EmitShrRegImm8(d445.Reg, 32)
		}
		ctx.EmitStoreToStack(d445, int32(bbs[2].PhiBase)+int32(0))
		if d10.Loc == scm.LocReg || d10.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d10.Reg)
		} else if d10.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d10.Reg)
			ctx.UnprotectReg(d10.Reg2)
		}
		ctx.EmitJmp(lbl3)
		ctx.RestoreAllocState(alloc443)
		d5 = snap354
		d6 = snap355
		d7 = snap356
		d8 = snap357
		d9 = snap358
		d10 = snap359
		d11 = snap360
		d12 = snap361
		d13 = snap362
		d14 = snap363
		d15 = snap364
		d16 = snap365
		d17 = snap366
		d18 = snap367
		d19 = snap368
		d21 = snap369
		d22 = snap370
		d23 = snap371
		d24 = snap372
		d25 = snap373
		d26 = snap374
		d27 = snap375
		d28 = snap376
		d29 = snap377
		d30 = snap378
		d31 = snap379
		d32 = snap380
		d33 = snap381
		d34 = snap382
		d35 = snap383
		d36 = snap384
		d37 = snap385
		d38 = snap386
		d39 = snap387
		d40 = snap388
		d41 = snap389
		d42 = snap390
		d43 = snap391
		d44 = snap392
		d45 = snap393
		d46 = snap394
		d47 = snap395
		d48 = snap396
		d49 = snap397
		d50 = snap398
		d51 = snap399
		d54 = snap400
		d55 = snap401
		d56 = snap402
		d159 = snap403
		d160 = snap404
		d161 = snap405
		d162 = snap406
		d163 = snap407
		d164 = snap408
		d165 = snap409
		d166 = snap410
		d167 = snap411
		d168 = snap412
		d169 = snap413
		d170 = snap414
		d171 = snap415
		d172 = snap416
		d173 = snap417
		d174 = snap418
		d175 = snap419
		d176 = snap420
		d177 = snap421
		d178 = snap422
		d179 = snap423
		d180 = snap424
		d181 = snap425
		d184 = snap426
		d335 = snap427
		d336 = snap428
		d337 = snap429
		d338 = snap430
		d340 = snap431
		d341 = snap432
		d342 = snap433
		d343 = snap434
		d344 = snap435
		d345 = snap436
		d346 = snap437
		d347 = snap438
		d349 = snap439
		d351 = snap440
		d352 = snap441
		d353 = snap442
		ctx.RestoreAllocState(alloc443)
		d5 = snap354
		d6 = snap355
		d7 = snap356
		d8 = snap357
		d9 = snap358
		d10 = snap359
		d11 = snap360
		d12 = snap361
		d13 = snap362
		d14 = snap363
		d15 = snap364
		d16 = snap365
		d17 = snap366
		d18 = snap367
		d19 = snap368
		d21 = snap369
		d22 = snap370
		d23 = snap371
		d24 = snap372
		d25 = snap373
		d26 = snap374
		d27 = snap375
		d28 = snap376
		d29 = snap377
		d30 = snap378
		d31 = snap379
		d32 = snap380
		d33 = snap381
		d34 = snap382
		d35 = snap383
		d36 = snap384
		d37 = snap385
		d38 = snap386
		d39 = snap387
		d40 = snap388
		d41 = snap389
		d42 = snap390
		d43 = snap391
		d44 = snap392
		d45 = snap393
		d46 = snap394
		d47 = snap395
		d48 = snap396
		d49 = snap397
		d50 = snap398
		d51 = snap399
		d54 = snap400
		d55 = snap401
		d56 = snap402
		d159 = snap403
		d160 = snap404
		d161 = snap405
		d162 = snap406
		d163 = snap407
		d164 = snap408
		d165 = snap409
		d166 = snap410
		d167 = snap411
		d168 = snap412
		d169 = snap413
		d170 = snap414
		d171 = snap415
		d172 = snap416
		d173 = snap417
		d174 = snap418
		d175 = snap419
		d176 = snap420
		d177 = snap421
		d178 = snap422
		d179 = snap423
		d180 = snap424
		d181 = snap425
		d184 = snap426
		d335 = snap427
		d336 = snap428
		d337 = snap429
		d338 = snap430
		d340 = snap431
		d341 = snap432
		d342 = snap433
		d343 = snap434
		d344 = snap435
		d345 = snap436
		d346 = snap437
		d347 = snap438
		d349 = snap439
		d351 = snap440
		d352 = snap441
		d353 = snap442
		ps446 := scm.PhiState{General: true}
		ps446.OverlayValues = make([]scm.JITValueDesc, 446)
		ps446.OverlayValues[5] = d5
		ps446.OverlayValues[6] = d6
		ps446.OverlayValues[7] = d7
		ps446.OverlayValues[8] = d8
		ps446.OverlayValues[9] = d9
		ps446.OverlayValues[10] = d10
		ps446.OverlayValues[11] = d11
		ps446.OverlayValues[12] = d12
		ps446.OverlayValues[13] = d13
		ps446.OverlayValues[14] = d14
		ps446.OverlayValues[15] = d15
		ps446.OverlayValues[16] = d16
		ps446.OverlayValues[17] = d17
		ps446.OverlayValues[18] = d18
		ps446.OverlayValues[19] = d19
		ps446.OverlayValues[21] = d21
		ps446.OverlayValues[22] = d22
		ps446.OverlayValues[23] = d23
		ps446.OverlayValues[24] = d24
		ps446.OverlayValues[25] = d25
		ps446.OverlayValues[26] = d26
		ps446.OverlayValues[27] = d27
		ps446.OverlayValues[28] = d28
		ps446.OverlayValues[29] = d29
		ps446.OverlayValues[30] = d30
		ps446.OverlayValues[31] = d31
		ps446.OverlayValues[32] = d32
		ps446.OverlayValues[33] = d33
		ps446.OverlayValues[34] = d34
		ps446.OverlayValues[35] = d35
		ps446.OverlayValues[36] = d36
		ps446.OverlayValues[37] = d37
		ps446.OverlayValues[38] = d38
		ps446.OverlayValues[39] = d39
		ps446.OverlayValues[40] = d40
		ps446.OverlayValues[41] = d41
		ps446.OverlayValues[42] = d42
		ps446.OverlayValues[43] = d43
		ps446.OverlayValues[44] = d44
		ps446.OverlayValues[45] = d45
		ps446.OverlayValues[46] = d46
		ps446.OverlayValues[47] = d47
		ps446.OverlayValues[48] = d48
		ps446.OverlayValues[49] = d49
		ps446.OverlayValues[50] = d50
		ps446.OverlayValues[51] = d51
		ps446.OverlayValues[54] = d54
		ps446.OverlayValues[55] = d55
		ps446.OverlayValues[56] = d56
		ps446.OverlayValues[159] = d159
		ps446.OverlayValues[160] = d160
		ps446.OverlayValues[161] = d161
		ps446.OverlayValues[162] = d162
		ps446.OverlayValues[163] = d163
		ps446.OverlayValues[164] = d164
		ps446.OverlayValues[165] = d165
		ps446.OverlayValues[166] = d166
		ps446.OverlayValues[167] = d167
		ps446.OverlayValues[168] = d168
		ps446.OverlayValues[169] = d169
		ps446.OverlayValues[170] = d170
		ps446.OverlayValues[171] = d171
		ps446.OverlayValues[172] = d172
		ps446.OverlayValues[173] = d173
		ps446.OverlayValues[174] = d174
		ps446.OverlayValues[175] = d175
		ps446.OverlayValues[176] = d176
		ps446.OverlayValues[177] = d177
		ps446.OverlayValues[178] = d178
		ps446.OverlayValues[179] = d179
		ps446.OverlayValues[180] = d180
		ps446.OverlayValues[181] = d181
		ps446.OverlayValues[184] = d184
		ps446.OverlayValues[335] = d335
		ps446.OverlayValues[336] = d336
		ps446.OverlayValues[337] = d337
		ps446.OverlayValues[338] = d338
		ps446.OverlayValues[340] = d340
		ps446.OverlayValues[341] = d341
		ps446.OverlayValues[342] = d342
		ps446.OverlayValues[343] = d343
		ps446.OverlayValues[344] = d344
		ps446.OverlayValues[345] = d345
		ps446.OverlayValues[346] = d346
		ps446.OverlayValues[347] = d347
		ps446.OverlayValues[349] = d349
		ps446.OverlayValues[351] = d351
		ps446.OverlayValues[352] = d352
		ps446.OverlayValues[353] = d353
		ps446.OverlayValues[444] = d444
		ps446.OverlayValues[445] = d445
		ps446.PhiValues = make([]scm.JITValueDesc, 1)
		d448 = d10
		ps446.PhiValues[0] = d448
		ps447 := scm.PhiState{General: true}
		ps447.OverlayValues = make([]scm.JITValueDesc, 449)
		ps447.OverlayValues[5] = d5
		ps447.OverlayValues[6] = d6
		ps447.OverlayValues[7] = d7
		ps447.OverlayValues[8] = d8
		ps447.OverlayValues[9] = d9
		ps447.OverlayValues[10] = d10
		ps447.OverlayValues[11] = d11
		ps447.OverlayValues[12] = d12
		ps447.OverlayValues[13] = d13
		ps447.OverlayValues[14] = d14
		ps447.OverlayValues[15] = d15
		ps447.OverlayValues[16] = d16
		ps447.OverlayValues[17] = d17
		ps447.OverlayValues[18] = d18
		ps447.OverlayValues[19] = d19
		ps447.OverlayValues[21] = d21
		ps447.OverlayValues[22] = d22
		ps447.OverlayValues[23] = d23
		ps447.OverlayValues[24] = d24
		ps447.OverlayValues[25] = d25
		ps447.OverlayValues[26] = d26
		ps447.OverlayValues[27] = d27
		ps447.OverlayValues[28] = d28
		ps447.OverlayValues[29] = d29
		ps447.OverlayValues[30] = d30
		ps447.OverlayValues[31] = d31
		ps447.OverlayValues[32] = d32
		ps447.OverlayValues[33] = d33
		ps447.OverlayValues[34] = d34
		ps447.OverlayValues[35] = d35
		ps447.OverlayValues[36] = d36
		ps447.OverlayValues[37] = d37
		ps447.OverlayValues[38] = d38
		ps447.OverlayValues[39] = d39
		ps447.OverlayValues[40] = d40
		ps447.OverlayValues[41] = d41
		ps447.OverlayValues[42] = d42
		ps447.OverlayValues[43] = d43
		ps447.OverlayValues[44] = d44
		ps447.OverlayValues[45] = d45
		ps447.OverlayValues[46] = d46
		ps447.OverlayValues[47] = d47
		ps447.OverlayValues[48] = d48
		ps447.OverlayValues[49] = d49
		ps447.OverlayValues[50] = d50
		ps447.OverlayValues[51] = d51
		ps447.OverlayValues[54] = d54
		ps447.OverlayValues[55] = d55
		ps447.OverlayValues[56] = d56
		ps447.OverlayValues[159] = d159
		ps447.OverlayValues[160] = d160
		ps447.OverlayValues[161] = d161
		ps447.OverlayValues[162] = d162
		ps447.OverlayValues[163] = d163
		ps447.OverlayValues[164] = d164
		ps447.OverlayValues[165] = d165
		ps447.OverlayValues[166] = d166
		ps447.OverlayValues[167] = d167
		ps447.OverlayValues[168] = d168
		ps447.OverlayValues[169] = d169
		ps447.OverlayValues[170] = d170
		ps447.OverlayValues[171] = d171
		ps447.OverlayValues[172] = d172
		ps447.OverlayValues[173] = d173
		ps447.OverlayValues[174] = d174
		ps447.OverlayValues[175] = d175
		ps447.OverlayValues[176] = d176
		ps447.OverlayValues[177] = d177
		ps447.OverlayValues[178] = d178
		ps447.OverlayValues[179] = d179
		ps447.OverlayValues[180] = d180
		ps447.OverlayValues[181] = d181
		ps447.OverlayValues[184] = d184
		ps447.OverlayValues[335] = d335
		ps447.OverlayValues[336] = d336
		ps447.OverlayValues[337] = d337
		ps447.OverlayValues[338] = d338
		ps447.OverlayValues[340] = d340
		ps447.OverlayValues[341] = d341
		ps447.OverlayValues[342] = d342
		ps447.OverlayValues[343] = d343
		ps447.OverlayValues[344] = d344
		ps447.OverlayValues[345] = d345
		ps447.OverlayValues[346] = d346
		ps447.OverlayValues[347] = d347
		ps447.OverlayValues[349] = d349
		ps447.OverlayValues[351] = d351
		ps447.OverlayValues[352] = d352
		ps447.OverlayValues[353] = d353
		ps447.OverlayValues[444] = d444
		ps447.OverlayValues[445] = d445
		ps447.OverlayValues[448] = d448
		snap449 := d5
		snap450 := d6
		snap451 := d7
		snap452 := d8
		snap453 := d9
		snap454 := d10
		snap455 := d11
		snap456 := d12
		snap457 := d13
		snap458 := d14
		snap459 := d15
		snap460 := d16
		snap461 := d17
		snap462 := d18
		snap463 := d19
		snap464 := d21
		snap465 := d22
		snap466 := d23
		snap467 := d24
		snap468 := d25
		snap469 := d26
		snap470 := d27
		snap471 := d28
		snap472 := d29
		snap473 := d30
		snap474 := d31
		snap475 := d32
		snap476 := d33
		snap477 := d34
		snap478 := d35
		snap479 := d36
		snap480 := d37
		snap481 := d38
		snap482 := d39
		snap483 := d40
		snap484 := d41
		snap485 := d42
		snap486 := d43
		snap487 := d44
		snap488 := d45
		snap489 := d46
		snap490 := d47
		snap491 := d48
		snap492 := d49
		snap493 := d50
		snap494 := d51
		snap495 := d54
		snap496 := d55
		snap497 := d56
		snap498 := d159
		snap499 := d160
		snap500 := d161
		snap501 := d162
		snap502 := d163
		snap503 := d164
		snap504 := d165
		snap505 := d166
		snap506 := d167
		snap507 := d168
		snap508 := d169
		snap509 := d170
		snap510 := d171
		snap511 := d172
		snap512 := d173
		snap513 := d174
		snap514 := d175
		snap515 := d176
		snap516 := d177
		snap517 := d178
		snap518 := d179
		snap519 := d180
		snap520 := d181
		snap521 := d184
		snap522 := d335
		snap523 := d336
		snap524 := d337
		snap525 := d338
		snap526 := d340
		snap527 := d341
		snap528 := d342
		snap529 := d343
		snap530 := d344
		snap531 := d345
		snap532 := d346
		snap533 := d347
		snap534 := d349
		snap535 := d351
		snap536 := d352
		snap537 := d353
		snap538 := d444
		snap539 := d445
		snap540 := d448
		alloc541 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps446)
		}
		ctx.RestoreAllocState(alloc541)
		d5 = snap449
		d6 = snap450
		d7 = snap451
		d8 = snap452
		d9 = snap453
		d10 = snap454
		d11 = snap455
		d12 = snap456
		d13 = snap457
		d14 = snap458
		d15 = snap459
		d16 = snap460
		d17 = snap461
		d18 = snap462
		d19 = snap463
		d21 = snap464
		d22 = snap465
		d23 = snap466
		d24 = snap467
		d25 = snap468
		d26 = snap469
		d27 = snap470
		d28 = snap471
		d29 = snap472
		d30 = snap473
		d31 = snap474
		d32 = snap475
		d33 = snap476
		d34 = snap477
		d35 = snap478
		d36 = snap479
		d37 = snap480
		d38 = snap481
		d39 = snap482
		d40 = snap483
		d41 = snap484
		d42 = snap485
		d43 = snap486
		d44 = snap487
		d45 = snap488
		d46 = snap489
		d47 = snap490
		d48 = snap491
		d49 = snap492
		d50 = snap493
		d51 = snap494
		d54 = snap495
		d55 = snap496
		d56 = snap497
		d159 = snap498
		d160 = snap499
		d161 = snap500
		d162 = snap501
		d163 = snap502
		d164 = snap503
		d165 = snap504
		d166 = snap505
		d167 = snap506
		d168 = snap507
		d169 = snap508
		d170 = snap509
		d171 = snap510
		d172 = snap511
		d173 = snap512
		d174 = snap513
		d175 = snap514
		d176 = snap515
		d177 = snap516
		d178 = snap517
		d179 = snap518
		d180 = snap519
		d181 = snap520
		d184 = snap521
		d335 = snap522
		d336 = snap523
		d337 = snap524
		d338 = snap525
		d340 = snap526
		d341 = snap527
		d342 = snap528
		d343 = snap529
		d344 = snap530
		d345 = snap531
		d346 = snap532
		d347 = snap533
		d349 = snap534
		d351 = snap535
		d352 = snap536
		d353 = snap537
		d444 = snap538
		d445 = snap539
		d448 = snap540
		if !bbs[6].Rendered {
			return bbs[6].RenderPS(ps447)
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
			ctx.FlushRegisterMoves()
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
		if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
			d184 = ps.OverlayValues[184]
		}
		if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != scm.LocNone {
			d335 = ps.OverlayValues[335]
		}
		if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != scm.LocNone {
			d336 = ps.OverlayValues[336]
		}
		if len(ps.OverlayValues) > 337 && ps.OverlayValues[337].Loc != scm.LocNone {
			d337 = ps.OverlayValues[337]
		}
		if len(ps.OverlayValues) > 338 && ps.OverlayValues[338].Loc != scm.LocNone {
			d338 = ps.OverlayValues[338]
		}
		if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != scm.LocNone {
			d340 = ps.OverlayValues[340]
		}
		if len(ps.OverlayValues) > 341 && ps.OverlayValues[341].Loc != scm.LocNone {
			d341 = ps.OverlayValues[341]
		}
		if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != scm.LocNone {
			d342 = ps.OverlayValues[342]
		}
		if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != scm.LocNone {
			d343 = ps.OverlayValues[343]
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
		if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != scm.LocNone {
			d351 = ps.OverlayValues[351]
		}
		if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != scm.LocNone {
			d352 = ps.OverlayValues[352]
		}
		if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != scm.LocNone {
			d353 = ps.OverlayValues[353]
		}
		if len(ps.OverlayValues) > 444 && ps.OverlayValues[444].Loc != scm.LocNone {
			d444 = ps.OverlayValues[444]
		}
		if len(ps.OverlayValues) > 445 && ps.OverlayValues[445].Loc != scm.LocNone {
			d445 = ps.OverlayValues[445]
		}
		if len(ps.OverlayValues) > 448 && ps.OverlayValues[448].Loc != scm.LocNone {
			d448 = ps.OverlayValues[448]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d5)
		var d542 scm.JITValueDesc
		if d5.Loc == scm.LocImm {
			d542 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitAddRegImm32Low(scratch, int32(1))
			d542 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d542)
		}
		if d542.Loc == scm.LocReg && d5.Loc == scm.LocReg && d542.Reg == d5.Reg {
			ctx.TransferReg(d5.Reg)
			d5.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d542)
		ctx.EmitStoreToStack(d542, int32(bbs[4].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d542)
		if ps.General {
			ctx.SyncDesc(&d5)
			if d5.Loc == scm.LocReg || d5.Loc == scm.LocFPReg {
				ctx.ProtectReg(d5.Reg)
			} else if d5.Loc == scm.LocRegPair {
				ctx.ProtectReg(d5.Reg)
				ctx.ProtectReg(d5.Reg2)
			}
			ctx.SyncDesc(&d7)
			if d7.Loc == scm.LocReg || d7.Loc == scm.LocFPReg {
				ctx.ProtectReg(d7.Reg)
			} else if d7.Loc == scm.LocRegPair {
				ctx.ProtectReg(d7.Reg)
				ctx.ProtectReg(d7.Reg2)
			}
			d543 = d5
			if d543.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d543)
			d544 = d543
			if d544.Loc == scm.LocImm {
				d544 = scm.JITValueDesc{Loc: scm.LocImm, Type: d544.Type, Imm: scm.NewInt(int64(uint64(d544.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d544.Reg, 32)
				ctx.EmitShrRegImm8(d544.Reg, 32)
			}
			ctx.EmitStoreToStack(d544, int32(bbs[4].PhiBase)+int32(16))
			d545 = d7
			if d545.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d545)
			d546 = d545
			if d546.Loc == scm.LocImm {
				d546 = scm.JITValueDesc{Loc: scm.LocImm, Type: d546.Type, Imm: scm.NewInt(int64(uint64(d546.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d546.Reg, 32)
				ctx.EmitShrRegImm8(d546.Reg, 32)
			}
			ctx.EmitStoreToStack(d546, int32(bbs[4].PhiBase)+int32(32))
			if d5.Loc == scm.LocReg || d5.Loc == scm.LocFPReg {
				ctx.UnprotectReg(d5.Reg)
			} else if d5.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d5.Reg)
				ctx.UnprotectReg(d5.Reg2)
			}
			if d7.Loc == scm.LocReg || d7.Loc == scm.LocFPReg {
				ctx.UnprotectReg(d7.Reg)
			} else if d7.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d7.Reg)
				ctx.UnprotectReg(d7.Reg2)
			}
		}
		ps547 := scm.PhiState{General: ps.General}
		ps547.OverlayValues = make([]scm.JITValueDesc, 547)
		ps547.OverlayValues[5] = d5
		ps547.OverlayValues[6] = d6
		ps547.OverlayValues[7] = d7
		ps547.OverlayValues[8] = d8
		ps547.OverlayValues[9] = d9
		ps547.OverlayValues[10] = d10
		ps547.OverlayValues[11] = d11
		ps547.OverlayValues[12] = d12
		ps547.OverlayValues[13] = d13
		ps547.OverlayValues[14] = d14
		ps547.OverlayValues[15] = d15
		ps547.OverlayValues[16] = d16
		ps547.OverlayValues[17] = d17
		ps547.OverlayValues[18] = d18
		ps547.OverlayValues[19] = d19
		ps547.OverlayValues[21] = d21
		ps547.OverlayValues[22] = d22
		ps547.OverlayValues[23] = d23
		ps547.OverlayValues[24] = d24
		ps547.OverlayValues[25] = d25
		ps547.OverlayValues[26] = d26
		ps547.OverlayValues[27] = d27
		ps547.OverlayValues[28] = d28
		ps547.OverlayValues[29] = d29
		ps547.OverlayValues[30] = d30
		ps547.OverlayValues[31] = d31
		ps547.OverlayValues[32] = d32
		ps547.OverlayValues[33] = d33
		ps547.OverlayValues[34] = d34
		ps547.OverlayValues[35] = d35
		ps547.OverlayValues[36] = d36
		ps547.OverlayValues[37] = d37
		ps547.OverlayValues[38] = d38
		ps547.OverlayValues[39] = d39
		ps547.OverlayValues[40] = d40
		ps547.OverlayValues[41] = d41
		ps547.OverlayValues[42] = d42
		ps547.OverlayValues[43] = d43
		ps547.OverlayValues[44] = d44
		ps547.OverlayValues[45] = d45
		ps547.OverlayValues[46] = d46
		ps547.OverlayValues[47] = d47
		ps547.OverlayValues[48] = d48
		ps547.OverlayValues[49] = d49
		ps547.OverlayValues[50] = d50
		ps547.OverlayValues[51] = d51
		ps547.OverlayValues[54] = d54
		ps547.OverlayValues[55] = d55
		ps547.OverlayValues[56] = d56
		ps547.OverlayValues[159] = d159
		ps547.OverlayValues[160] = d160
		ps547.OverlayValues[161] = d161
		ps547.OverlayValues[162] = d162
		ps547.OverlayValues[163] = d163
		ps547.OverlayValues[164] = d164
		ps547.OverlayValues[165] = d165
		ps547.OverlayValues[166] = d166
		ps547.OverlayValues[167] = d167
		ps547.OverlayValues[168] = d168
		ps547.OverlayValues[169] = d169
		ps547.OverlayValues[170] = d170
		ps547.OverlayValues[171] = d171
		ps547.OverlayValues[172] = d172
		ps547.OverlayValues[173] = d173
		ps547.OverlayValues[174] = d174
		ps547.OverlayValues[175] = d175
		ps547.OverlayValues[176] = d176
		ps547.OverlayValues[177] = d177
		ps547.OverlayValues[178] = d178
		ps547.OverlayValues[179] = d179
		ps547.OverlayValues[180] = d180
		ps547.OverlayValues[181] = d181
		ps547.OverlayValues[184] = d184
		ps547.OverlayValues[335] = d335
		ps547.OverlayValues[336] = d336
		ps547.OverlayValues[337] = d337
		ps547.OverlayValues[338] = d338
		ps547.OverlayValues[340] = d340
		ps547.OverlayValues[341] = d341
		ps547.OverlayValues[342] = d342
		ps547.OverlayValues[343] = d343
		ps547.OverlayValues[344] = d344
		ps547.OverlayValues[345] = d345
		ps547.OverlayValues[346] = d346
		ps547.OverlayValues[347] = d347
		ps547.OverlayValues[349] = d349
		ps547.OverlayValues[351] = d351
		ps547.OverlayValues[352] = d352
		ps547.OverlayValues[353] = d353
		ps547.OverlayValues[444] = d444
		ps547.OverlayValues[445] = d445
		ps547.OverlayValues[448] = d448
		ps547.OverlayValues[542] = d542
		ps547.OverlayValues[543] = d543
		ps547.OverlayValues[544] = d544
		ps547.OverlayValues[545] = d545
		ps547.OverlayValues[546] = d546
		ps547.PhiValues = make([]scm.JITValueDesc, 3)
		d548 = d5
		ps547.PhiValues[1] = d548
		d549 = d7
		ps547.PhiValues[2] = d549
		if ps547.General && bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
			return result
		}
		return bbs[4].RenderPS(ps547)
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
			ctx.FlushRegisterMoves()
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
		if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
			d184 = ps.OverlayValues[184]
		}
		if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != scm.LocNone {
			d335 = ps.OverlayValues[335]
		}
		if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != scm.LocNone {
			d336 = ps.OverlayValues[336]
		}
		if len(ps.OverlayValues) > 337 && ps.OverlayValues[337].Loc != scm.LocNone {
			d337 = ps.OverlayValues[337]
		}
		if len(ps.OverlayValues) > 338 && ps.OverlayValues[338].Loc != scm.LocNone {
			d338 = ps.OverlayValues[338]
		}
		if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != scm.LocNone {
			d340 = ps.OverlayValues[340]
		}
		if len(ps.OverlayValues) > 341 && ps.OverlayValues[341].Loc != scm.LocNone {
			d341 = ps.OverlayValues[341]
		}
		if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != scm.LocNone {
			d342 = ps.OverlayValues[342]
		}
		if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != scm.LocNone {
			d343 = ps.OverlayValues[343]
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
		if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != scm.LocNone {
			d351 = ps.OverlayValues[351]
		}
		if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != scm.LocNone {
			d352 = ps.OverlayValues[352]
		}
		if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != scm.LocNone {
			d353 = ps.OverlayValues[353]
		}
		if len(ps.OverlayValues) > 444 && ps.OverlayValues[444].Loc != scm.LocNone {
			d444 = ps.OverlayValues[444]
		}
		if len(ps.OverlayValues) > 445 && ps.OverlayValues[445].Loc != scm.LocNone {
			d445 = ps.OverlayValues[445]
		}
		if len(ps.OverlayValues) > 448 && ps.OverlayValues[448].Loc != scm.LocNone {
			d448 = ps.OverlayValues[448]
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
		if len(ps.OverlayValues) > 548 && ps.OverlayValues[548].Loc != scm.LocNone {
			d548 = ps.OverlayValues[548]
		}
		if len(ps.OverlayValues) > 549 && ps.OverlayValues[549].Loc != scm.LocNone {
			d549 = ps.OverlayValues[549]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d9)
		d550 = d9
		_ = d550
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
		var d551 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d551 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48)
			r63 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r63, thisptr.Reg, off)
			d551 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r63}
			ctx.BindReg(r63, &d551)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d551)
		ctx.EnsureDesc(&d551)
		var d552 scm.JITValueDesc
		if d551.Loc == scm.LocImm {
			d552 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d551.Imm.Int()))))}
		} else {
			r64 := ctx.AllocReg()
			ctx.EmitMovRegReg(r64, d551.Reg)
			ctx.EmitShlRegImm8(r64, 56)
			ctx.EmitShrRegImm8(r64, 56)
			d552 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r64}
			ctx.BindReg(r64, &d552)
		}
		ctx.FreeDesc(&d551)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d550)
		ctx.EnsureDesc(&d550)
		var d553 scm.JITValueDesc
		if d550.Loc == scm.LocImm {
			d553 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d550.Imm.Int()))))}
		} else {
			r65 := ctx.AllocReg()
			ctx.EmitMovRegReg(r65, d550.Reg)
			ctx.EmitShlRegImm8(r65, 32)
			ctx.EmitShrRegImm8(r65, 32)
			d553 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
			ctx.BindReg(r65, &d553)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d553)
		ctx.EnsureDesc(&d552)
		ctx.EnsureDescsTogether(&d553, &d552)
		var d554 scm.JITValueDesc
		if d553.Loc == scm.LocImm && d552.Loc == scm.LocImm {
			d554 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d553.Imm.Int() * d552.Imm.Int())}
		} else if d553.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d552.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d553.Imm.Int()))
			ctx.EmitImulInt64(scratch, d552.Reg)
			d554 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d554)
		} else if d552.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d553.Reg)
			ctx.EmitMovRegReg(scratch, d553.Reg)
			if d552.Imm.Int() >= -2147483648 && d552.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d552.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d552.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d554 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d554)
		} else {
			r66 := ctx.AllocRegExcept(d553.Reg, d552.Reg)
			ctx.EmitMovRegReg(r66, d553.Reg)
			ctx.EmitImulInt64(r66, d552.Reg)
			d554 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
			ctx.BindReg(r66, &d554)
		}
		if d554.Loc == scm.LocReg && d553.Loc == scm.LocReg && d554.Reg == d553.Reg {
			ctx.TransferReg(d553.Reg)
			d553.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d553)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d554)
		var d555 scm.JITValueDesc
		if d554.Loc == scm.LocImm {
			d555 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d554.Imm.Int() / 64)}
		} else {
			r67 := ctx.AllocRegExcept(d554.Reg)
			ctx.EmitMovRegReg(r67, d554.Reg)
			ctx.EmitShrRegImm8(r67, 6)
			d555 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r67}
			ctx.BindReg(r67, &d555)
		}
		if d555.Loc == scm.LocReg && d554.Loc == scm.LocReg && d555.Reg == d554.Reg {
			ctx.TransferReg(d554.Reg)
			d554.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d554)
		var d556 scm.JITValueDesc
		if d554.Loc == scm.LocImm {
			d556 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d554.Imm.Int() % 64)}
		} else {
			r68 := ctx.AllocRegExcept(d554.Reg)
			ctx.EmitMovRegReg(r68, d554.Reg)
			ctx.EmitAndRegImm32(r68, 63)
			d556 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r68}
			ctx.BindReg(r68, &d556)
		}
		if d556.Loc == scm.LocReg && d554.Loc == scm.LocReg && d556.Reg == d554.Reg {
			ctx.TransferReg(d554.Reg)
			d554.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d554)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d557 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d557 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r69 := ctx.AllocReg()
			r70 := ctx.AllocRegExcept(r69)
			r71 := ctx.AllocRegExcept(r69, r70)
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			ctx.EmitMovRegMem(r69, thisptr.Reg, off)
			ctx.EmitMovRegMem(r70, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r71, thisptr.Reg, off+16)
			d557 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r69, Reg2: r70, Reg3: r71}
			ctx.BindReg(r69, &d557)
			ctx.BindReg(r70, &d557)
			ctx.BindReg(r71, &d557)
			ctx.BindReg(r69, &d557)
			ctx.BindReg(r70, &d557)
			ctx.BindReg(r71, &d557)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d555)
		ctx.ReclaimUntrackedRegs()
		d558 = ctx.EmitLoadScalarSliceElement(&d557, &d555, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d558)
		ctx.EnsureDesc(&d556)
		ctx.EnsureDescsTogether(&d558, &d556)
		var d559 scm.JITValueDesc
		if d558.Loc == scm.LocImm && d556.Loc == scm.LocImm {
			d559 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d558.Imm.Int()) << uint64(d556.Imm.Int())))}
		} else if d556.Loc == scm.LocImm {
			r72 := ctx.AllocRegExcept(d558.Reg)
			ctx.EmitMovRegReg(r72, d558.Reg)
			ctx.EmitShlRegImm8(r72, uint8(d556.Imm.Int()))
			d559 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r72}
			ctx.BindReg(r72, &d559)
		} else {
			{
				shiftSrc := d558.Reg
				r73 := ctx.AllocRegExcept(d558.Reg, d556.Reg)
				ctx.EmitMovRegReg(r73, d558.Reg)
				shiftSrc = r73
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d556.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d556.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d556.Reg)
				}
				ctx.EmitShlRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d559 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d559)
			}
		}
		if d559.Loc == scm.LocReg && d558.Loc == scm.LocReg && d559.Reg == d558.Reg {
			ctx.TransferReg(d558.Reg)
			d558.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d558)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d555)
		ctx.EnsureDesc(&d555)
		var d560 scm.JITValueDesc
		if d555.Loc == scm.LocImm {
			d560 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d555.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d555.Reg)
			ctx.EmitMovRegReg(scratch, d555.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d560 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d560)
		}
		if d560.Loc == scm.LocReg && d555.Loc == scm.LocReg && d560.Reg == d555.Reg {
			ctx.TransferReg(d555.Reg)
			d555.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d555)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d560)
		ctx.ReclaimUntrackedRegs()
		d561 = ctx.EmitLoadScalarSliceElement(&d557, &d560, 8, scm.TagInt)
		ctx.FreeDesc(&d560)
		ctx.ReclaimUntrackedRegs()
		d562 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d556)
		ctx.EnsureDescsTogether(&d562, &d556)
		var d563 scm.JITValueDesc
		if d562.Loc == scm.LocImm && d556.Loc == scm.LocImm {
			d563 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d562.Imm.Int() - d556.Imm.Int())}
		} else if d556.Loc == scm.LocImm && d556.Imm.Int() == 0 {
			r74 := ctx.AllocRegExcept(d562.Reg)
			ctx.EmitMovRegReg(r74, d562.Reg)
			d563 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r74}
			ctx.BindReg(r74, &d563)
		} else if d562.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d556.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d562.Imm.Int()))
			ctx.EmitSubInt64(scratch, d556.Reg)
			d563 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d563)
		} else if d556.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d562.Reg)
			ctx.EmitMovRegReg(scratch, d562.Reg)
			if d556.Imm.Int() >= -2147483648 && d556.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d556.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d556.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d563 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d563)
		} else {
			r75 := ctx.AllocRegExcept(d562.Reg, d556.Reg)
			ctx.EmitMovRegReg(r75, d562.Reg)
			ctx.EmitSubInt64(r75, d556.Reg)
			d563 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
			ctx.BindReg(r75, &d563)
		}
		if d563.Loc == scm.LocReg && d562.Loc == scm.LocReg && d563.Reg == d562.Reg {
			ctx.TransferReg(d562.Reg)
			d562.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d556)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d561)
		ctx.EnsureDesc(&d563)
		ctx.EnsureDescsTogether(&d561, &d563)
		var d564 scm.JITValueDesc
		if d561.Loc == scm.LocImm && d563.Loc == scm.LocImm {
			d564 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d561.Imm.Int()) >> uint64(d563.Imm.Int())))}
		} else if d563.Loc == scm.LocImm {
			r76 := ctx.AllocRegExcept(d561.Reg)
			ctx.EmitMovRegReg(r76, d561.Reg)
			ctx.EmitShrRegImm8(r76, uint8(d563.Imm.Int()))
			d564 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r76}
			ctx.BindReg(r76, &d564)
		} else {
			{
				shiftSrc := d561.Reg
				r77 := ctx.AllocRegExcept(d561.Reg, d563.Reg)
				ctx.EmitMovRegReg(r77, d561.Reg)
				shiftSrc = r77
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d563.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d563.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d563.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d564 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d564)
			}
		}
		if d564.Loc == scm.LocReg && d561.Loc == scm.LocReg && d564.Reg == d561.Reg {
			ctx.TransferReg(d561.Reg)
			d561.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d561)
		ctx.FreeDesc(&d563)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d559)
		ctx.EnsureDesc(&d564)
		var d565 scm.JITValueDesc
		if d559.Loc == scm.LocImm && d564.Loc == scm.LocImm {
			d565 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d559.Imm.Int() | d564.Imm.Int())}
		} else if d559.Loc == scm.LocImm && d559.Imm.Int() == 0 {
			d565 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d564.Reg}
			ctx.BindReg(d564.Reg, &d565)
		} else if d564.Loc == scm.LocImm && d564.Imm.Int() == 0 {
			r78 := ctx.AllocRegExcept(d559.Reg)
			ctx.EmitMovRegReg(r78, d559.Reg)
			d565 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r78}
			ctx.BindReg(r78, &d565)
		} else if d559.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d564.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d559.Imm.Int()))
			ctx.EmitOrInt64(scratch, d564.Reg)
			d565 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d565)
		} else if d564.Loc == scm.LocImm {
			r79 := ctx.AllocRegExcept(d559.Reg)
			ctx.EmitMovRegReg(r79, d559.Reg)
			if d564.Imm.Int() >= -2147483648 && d564.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r79, int32(d564.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d564.Imm.Int()))
				ctx.EmitOrInt64(r79, scm.RegR11)
			}
			d565 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r79}
			ctx.BindReg(r79, &d565)
		} else {
			r80 := ctx.AllocRegExcept(d559.Reg, d564.Reg)
			ctx.EmitMovRegReg(r80, d559.Reg)
			ctx.EmitOrInt64(r80, d564.Reg)
			d565 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
			ctx.BindReg(r80, &d565)
		}
		if d565.Loc == scm.LocReg && d559.Loc == scm.LocReg && d565.Reg == d559.Reg {
			ctx.TransferReg(d559.Reg)
			d559.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d559)
		ctx.FreeDesc(&d564)
		ctx.ReclaimUntrackedRegs()
		d566 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d552)
		ctx.EnsureDescsTogether(&d566, &d552)
		var d567 scm.JITValueDesc
		if d566.Loc == scm.LocImm && d552.Loc == scm.LocImm {
			d567 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d566.Imm.Int() - d552.Imm.Int())}
		} else if d552.Loc == scm.LocImm && d552.Imm.Int() == 0 {
			r81 := ctx.AllocRegExcept(d566.Reg)
			ctx.EmitMovRegReg(r81, d566.Reg)
			d567 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
			ctx.BindReg(r81, &d567)
		} else if d566.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d552.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d566.Imm.Int()))
			ctx.EmitSubInt64(scratch, d552.Reg)
			d567 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d567)
		} else if d552.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d566.Reg)
			ctx.EmitMovRegReg(scratch, d566.Reg)
			if d552.Imm.Int() >= -2147483648 && d552.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d552.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d552.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d567 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d567)
		} else {
			r82 := ctx.AllocRegExcept(d566.Reg, d552.Reg)
			ctx.EmitMovRegReg(r82, d566.Reg)
			ctx.EmitSubInt64(r82, d552.Reg)
			d567 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r82}
			ctx.BindReg(r82, &d567)
		}
		if d567.Loc == scm.LocReg && d566.Loc == scm.LocReg && d567.Reg == d566.Reg {
			ctx.TransferReg(d566.Reg)
			d566.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d552)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d565)
		ctx.EnsureDesc(&d567)
		ctx.EnsureDescsTogether(&d565, &d567)
		var d568 scm.JITValueDesc
		if d565.Loc == scm.LocImm && d567.Loc == scm.LocImm {
			d568 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d565.Imm.Int()) >> uint64(d567.Imm.Int())))}
		} else if d567.Loc == scm.LocImm {
			r83 := ctx.AllocRegExcept(d565.Reg)
			ctx.EmitMovRegReg(r83, d565.Reg)
			ctx.EmitShrRegImm8(r83, uint8(d567.Imm.Int()))
			d568 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
			ctx.BindReg(r83, &d568)
		} else {
			{
				shiftSrc := d565.Reg
				r84 := ctx.AllocRegExcept(d565.Reg, d567.Reg)
				ctx.EmitMovRegReg(r84, d565.Reg)
				shiftSrc = r84
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d567.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d567.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d567.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d568 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d568)
			}
		}
		if d568.Loc == scm.LocReg && d565.Loc == scm.LocReg && d568.Reg == d565.Reg {
			ctx.TransferReg(d565.Reg)
			d565.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d565)
		ctx.FreeDesc(&d567)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d568)
		ctx.EnsureDesc(&d568)
		ctx.EnsureDesc(&d568)
		var d569 scm.JITValueDesc
		if d568.Loc == scm.LocImm {
			d569 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d568.Imm.Int()))))}
		} else {
			r85 := ctx.AllocReg()
			ctx.EmitMovRegReg(r85, d568.Reg)
			d569 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
			ctx.BindReg(r85, &d569)
		}
		ctx.FreeDesc(&d568)
		var d570 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d570 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56)
			r86 := ctx.AllocReg()
			ctx.EmitMovRegMem(r86, thisptr.Reg, off)
			d570 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r86}
			ctx.BindReg(r86, &d570)
		}
		ctx.EnsureDesc(&d569)
		ctx.EnsureDesc(&d570)
		ctx.EnsureDescsTogether(&d569, &d570)
		var d571 scm.JITValueDesc
		if d569.Loc == scm.LocImm && d570.Loc == scm.LocImm {
			d571 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d569.Imm.Int() + d570.Imm.Int())}
		} else if d570.Loc == scm.LocImm && d570.Imm.Int() == 0 {
			r87 := ctx.AllocRegExcept(d569.Reg)
			ctx.EmitMovRegReg(r87, d569.Reg)
			d571 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r87}
			ctx.BindReg(r87, &d571)
		} else if d569.Loc == scm.LocImm && d569.Imm.Int() == 0 {
			d571 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d570.Reg}
			ctx.BindReg(d570.Reg, &d571)
		} else if d569.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d570.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d569.Imm.Int()))
			ctx.EmitAddInt64(scratch, d570.Reg)
			d571 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d571)
		} else if d570.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d569.Reg)
			ctx.EmitMovRegReg(scratch, d569.Reg)
			if d570.Imm.Int() >= -2147483648 && d570.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d570.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d570.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d571 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d571)
		} else {
			r88 := ctx.AllocRegExcept(d569.Reg, d570.Reg)
			ctx.EmitMovRegReg(r88, d569.Reg)
			ctx.EmitAddInt64(r88, d570.Reg)
			d571 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
			ctx.BindReg(r88, &d571)
		}
		if d571.Loc == scm.LocReg && d569.Loc == scm.LocReg && d571.Reg == d569.Reg {
			ctx.TransferReg(d569.Reg)
			d569.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d569)
		ctx.FreeDesc(&d570)
		ctx.EnsureDesc(&d571)
		ctx.EnsureDesc(&d571)
		var d572 scm.JITValueDesc
		if d571.Loc == scm.LocImm {
			d572 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d571.Imm.Int()))))}
		} else {
			r89 := ctx.AllocReg()
			ctx.EmitMovRegReg(r89, d571.Reg)
			ctx.EmitShlRegImm8(r89, 32)
			ctx.EmitShrRegImm8(r89, 32)
			d572 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
			ctx.BindReg(r89, &d572)
		}
		ctx.FreeDesc(&d571)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d572)
		ctx.EnsureDescsTogether(&idxInt, &d572)
		var d573 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm && d572.Loc == scm.LocImm {
			d573 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d572.Imm.Int()))}
		} else if d572.Loc == scm.LocImm {
			r90 := ctx.AllocRegExcept(idxInt.Reg)
			if d572.Imm.Int() >= -2147483648 && d572.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(idxInt.Reg, int32(d572.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d572.Imm.Int()))
				ctx.EmitCmpInt64(idxInt.Reg, scm.RegR11)
			}
			d573 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r90, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r90, &d573)
		} else if idxInt.Loc == scm.LocImm {
			r91 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d572.Reg)
			d573 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r91, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r91, &d573)
		} else {
			r92 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitCmpInt64(idxInt.Reg, d572.Reg)
			d573 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r92, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r92, &d573)
		}
		ctx.FreeDesc(&d572)
		d574 = d573
		ctx.EnsureDesc(&d574)
		if d574.Loc != scm.LocImm && d574.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d574.Loc == scm.LocImm {
			if d574.Imm.Bool() {
				if ps.General {
				}
				ps575 := scm.PhiState{General: ps.General}
				ps575.OverlayValues = make([]scm.JITValueDesc, 575)
				ps575.OverlayValues[5] = d5
				ps575.OverlayValues[6] = d6
				ps575.OverlayValues[7] = d7
				ps575.OverlayValues[8] = d8
				ps575.OverlayValues[9] = d9
				ps575.OverlayValues[10] = d10
				ps575.OverlayValues[11] = d11
				ps575.OverlayValues[12] = d12
				ps575.OverlayValues[13] = d13
				ps575.OverlayValues[14] = d14
				ps575.OverlayValues[15] = d15
				ps575.OverlayValues[16] = d16
				ps575.OverlayValues[17] = d17
				ps575.OverlayValues[18] = d18
				ps575.OverlayValues[19] = d19
				ps575.OverlayValues[21] = d21
				ps575.OverlayValues[22] = d22
				ps575.OverlayValues[23] = d23
				ps575.OverlayValues[24] = d24
				ps575.OverlayValues[25] = d25
				ps575.OverlayValues[26] = d26
				ps575.OverlayValues[27] = d27
				ps575.OverlayValues[28] = d28
				ps575.OverlayValues[29] = d29
				ps575.OverlayValues[30] = d30
				ps575.OverlayValues[31] = d31
				ps575.OverlayValues[32] = d32
				ps575.OverlayValues[33] = d33
				ps575.OverlayValues[34] = d34
				ps575.OverlayValues[35] = d35
				ps575.OverlayValues[36] = d36
				ps575.OverlayValues[37] = d37
				ps575.OverlayValues[38] = d38
				ps575.OverlayValues[39] = d39
				ps575.OverlayValues[40] = d40
				ps575.OverlayValues[41] = d41
				ps575.OverlayValues[42] = d42
				ps575.OverlayValues[43] = d43
				ps575.OverlayValues[44] = d44
				ps575.OverlayValues[45] = d45
				ps575.OverlayValues[46] = d46
				ps575.OverlayValues[47] = d47
				ps575.OverlayValues[48] = d48
				ps575.OverlayValues[49] = d49
				ps575.OverlayValues[50] = d50
				ps575.OverlayValues[51] = d51
				ps575.OverlayValues[54] = d54
				ps575.OverlayValues[55] = d55
				ps575.OverlayValues[56] = d56
				ps575.OverlayValues[159] = d159
				ps575.OverlayValues[160] = d160
				ps575.OverlayValues[161] = d161
				ps575.OverlayValues[162] = d162
				ps575.OverlayValues[163] = d163
				ps575.OverlayValues[164] = d164
				ps575.OverlayValues[165] = d165
				ps575.OverlayValues[166] = d166
				ps575.OverlayValues[167] = d167
				ps575.OverlayValues[168] = d168
				ps575.OverlayValues[169] = d169
				ps575.OverlayValues[170] = d170
				ps575.OverlayValues[171] = d171
				ps575.OverlayValues[172] = d172
				ps575.OverlayValues[173] = d173
				ps575.OverlayValues[174] = d174
				ps575.OverlayValues[175] = d175
				ps575.OverlayValues[176] = d176
				ps575.OverlayValues[177] = d177
				ps575.OverlayValues[178] = d178
				ps575.OverlayValues[179] = d179
				ps575.OverlayValues[180] = d180
				ps575.OverlayValues[181] = d181
				ps575.OverlayValues[184] = d184
				ps575.OverlayValues[335] = d335
				ps575.OverlayValues[336] = d336
				ps575.OverlayValues[337] = d337
				ps575.OverlayValues[338] = d338
				ps575.OverlayValues[340] = d340
				ps575.OverlayValues[341] = d341
				ps575.OverlayValues[342] = d342
				ps575.OverlayValues[343] = d343
				ps575.OverlayValues[344] = d344
				ps575.OverlayValues[345] = d345
				ps575.OverlayValues[346] = d346
				ps575.OverlayValues[347] = d347
				ps575.OverlayValues[349] = d349
				ps575.OverlayValues[351] = d351
				ps575.OverlayValues[352] = d352
				ps575.OverlayValues[353] = d353
				ps575.OverlayValues[444] = d444
				ps575.OverlayValues[445] = d445
				ps575.OverlayValues[448] = d448
				ps575.OverlayValues[542] = d542
				ps575.OverlayValues[543] = d543
				ps575.OverlayValues[544] = d544
				ps575.OverlayValues[545] = d545
				ps575.OverlayValues[546] = d546
				ps575.OverlayValues[548] = d548
				ps575.OverlayValues[549] = d549
				ps575.OverlayValues[550] = d550
				ps575.OverlayValues[551] = d551
				ps575.OverlayValues[552] = d552
				ps575.OverlayValues[553] = d553
				ps575.OverlayValues[554] = d554
				ps575.OverlayValues[555] = d555
				ps575.OverlayValues[556] = d556
				ps575.OverlayValues[557] = d557
				ps575.OverlayValues[558] = d558
				ps575.OverlayValues[559] = d559
				ps575.OverlayValues[560] = d560
				ps575.OverlayValues[561] = d561
				ps575.OverlayValues[562] = d562
				ps575.OverlayValues[563] = d563
				ps575.OverlayValues[564] = d564
				ps575.OverlayValues[565] = d565
				ps575.OverlayValues[566] = d566
				ps575.OverlayValues[567] = d567
				ps575.OverlayValues[568] = d568
				ps575.OverlayValues[569] = d569
				ps575.OverlayValues[570] = d570
				ps575.OverlayValues[571] = d571
				ps575.OverlayValues[572] = d572
				ps575.OverlayValues[573] = d573
				ps575.OverlayValues[574] = d574
				return bbs[7].RenderPS(ps575)
			}
			if ps.General {
			}
			ps576 := scm.PhiState{General: ps.General}
			ps576.OverlayValues = make([]scm.JITValueDesc, 575)
			ps576.OverlayValues[5] = d5
			ps576.OverlayValues[6] = d6
			ps576.OverlayValues[7] = d7
			ps576.OverlayValues[8] = d8
			ps576.OverlayValues[9] = d9
			ps576.OverlayValues[10] = d10
			ps576.OverlayValues[11] = d11
			ps576.OverlayValues[12] = d12
			ps576.OverlayValues[13] = d13
			ps576.OverlayValues[14] = d14
			ps576.OverlayValues[15] = d15
			ps576.OverlayValues[16] = d16
			ps576.OverlayValues[17] = d17
			ps576.OverlayValues[18] = d18
			ps576.OverlayValues[19] = d19
			ps576.OverlayValues[21] = d21
			ps576.OverlayValues[22] = d22
			ps576.OverlayValues[23] = d23
			ps576.OverlayValues[24] = d24
			ps576.OverlayValues[25] = d25
			ps576.OverlayValues[26] = d26
			ps576.OverlayValues[27] = d27
			ps576.OverlayValues[28] = d28
			ps576.OverlayValues[29] = d29
			ps576.OverlayValues[30] = d30
			ps576.OverlayValues[31] = d31
			ps576.OverlayValues[32] = d32
			ps576.OverlayValues[33] = d33
			ps576.OverlayValues[34] = d34
			ps576.OverlayValues[35] = d35
			ps576.OverlayValues[36] = d36
			ps576.OverlayValues[37] = d37
			ps576.OverlayValues[38] = d38
			ps576.OverlayValues[39] = d39
			ps576.OverlayValues[40] = d40
			ps576.OverlayValues[41] = d41
			ps576.OverlayValues[42] = d42
			ps576.OverlayValues[43] = d43
			ps576.OverlayValues[44] = d44
			ps576.OverlayValues[45] = d45
			ps576.OverlayValues[46] = d46
			ps576.OverlayValues[47] = d47
			ps576.OverlayValues[48] = d48
			ps576.OverlayValues[49] = d49
			ps576.OverlayValues[50] = d50
			ps576.OverlayValues[51] = d51
			ps576.OverlayValues[54] = d54
			ps576.OverlayValues[55] = d55
			ps576.OverlayValues[56] = d56
			ps576.OverlayValues[159] = d159
			ps576.OverlayValues[160] = d160
			ps576.OverlayValues[161] = d161
			ps576.OverlayValues[162] = d162
			ps576.OverlayValues[163] = d163
			ps576.OverlayValues[164] = d164
			ps576.OverlayValues[165] = d165
			ps576.OverlayValues[166] = d166
			ps576.OverlayValues[167] = d167
			ps576.OverlayValues[168] = d168
			ps576.OverlayValues[169] = d169
			ps576.OverlayValues[170] = d170
			ps576.OverlayValues[171] = d171
			ps576.OverlayValues[172] = d172
			ps576.OverlayValues[173] = d173
			ps576.OverlayValues[174] = d174
			ps576.OverlayValues[175] = d175
			ps576.OverlayValues[176] = d176
			ps576.OverlayValues[177] = d177
			ps576.OverlayValues[178] = d178
			ps576.OverlayValues[179] = d179
			ps576.OverlayValues[180] = d180
			ps576.OverlayValues[181] = d181
			ps576.OverlayValues[184] = d184
			ps576.OverlayValues[335] = d335
			ps576.OverlayValues[336] = d336
			ps576.OverlayValues[337] = d337
			ps576.OverlayValues[338] = d338
			ps576.OverlayValues[340] = d340
			ps576.OverlayValues[341] = d341
			ps576.OverlayValues[342] = d342
			ps576.OverlayValues[343] = d343
			ps576.OverlayValues[344] = d344
			ps576.OverlayValues[345] = d345
			ps576.OverlayValues[346] = d346
			ps576.OverlayValues[347] = d347
			ps576.OverlayValues[349] = d349
			ps576.OverlayValues[351] = d351
			ps576.OverlayValues[352] = d352
			ps576.OverlayValues[353] = d353
			ps576.OverlayValues[444] = d444
			ps576.OverlayValues[445] = d445
			ps576.OverlayValues[448] = d448
			ps576.OverlayValues[542] = d542
			ps576.OverlayValues[543] = d543
			ps576.OverlayValues[544] = d544
			ps576.OverlayValues[545] = d545
			ps576.OverlayValues[546] = d546
			ps576.OverlayValues[548] = d548
			ps576.OverlayValues[549] = d549
			ps576.OverlayValues[550] = d550
			ps576.OverlayValues[551] = d551
			ps576.OverlayValues[552] = d552
			ps576.OverlayValues[553] = d553
			ps576.OverlayValues[554] = d554
			ps576.OverlayValues[555] = d555
			ps576.OverlayValues[556] = d556
			ps576.OverlayValues[557] = d557
			ps576.OverlayValues[558] = d558
			ps576.OverlayValues[559] = d559
			ps576.OverlayValues[560] = d560
			ps576.OverlayValues[561] = d561
			ps576.OverlayValues[562] = d562
			ps576.OverlayValues[563] = d563
			ps576.OverlayValues[564] = d564
			ps576.OverlayValues[565] = d565
			ps576.OverlayValues[566] = d566
			ps576.OverlayValues[567] = d567
			ps576.OverlayValues[568] = d568
			ps576.OverlayValues[569] = d569
			ps576.OverlayValues[570] = d570
			ps576.OverlayValues[571] = d571
			ps576.OverlayValues[572] = d572
			ps576.OverlayValues[573] = d573
			ps576.OverlayValues[574] = d574
			return bbs[9].RenderPS(ps576)
		}
		if !ps.General {
			ps.General = true
			return bbs[6].RenderPS(ps)
		}
		ctx.EmitJump(d574.Condition, lbl8)
		if bbs[9].Rendered {
			ctx.EmitJmp(lbl10)
		}
		ctx.FreeDesc(&d573)
		snap577 := d5
		snap578 := d6
		snap579 := d7
		snap580 := d8
		snap581 := d9
		snap582 := d10
		snap583 := d11
		snap584 := d12
		snap585 := d13
		snap586 := d14
		snap587 := d15
		snap588 := d16
		snap589 := d17
		snap590 := d18
		snap591 := d19
		snap592 := d21
		snap593 := d22
		snap594 := d23
		snap595 := d24
		snap596 := d25
		snap597 := d26
		snap598 := d27
		snap599 := d28
		snap600 := d29
		snap601 := d30
		snap602 := d31
		snap603 := d32
		snap604 := d33
		snap605 := d34
		snap606 := d35
		snap607 := d36
		snap608 := d37
		snap609 := d38
		snap610 := d39
		snap611 := d40
		snap612 := d41
		snap613 := d42
		snap614 := d43
		snap615 := d44
		snap616 := d45
		snap617 := d46
		snap618 := d47
		snap619 := d48
		snap620 := d49
		snap621 := d50
		snap622 := d51
		snap623 := d54
		snap624 := d55
		snap625 := d56
		snap626 := d159
		snap627 := d160
		snap628 := d161
		snap629 := d162
		snap630 := d163
		snap631 := d164
		snap632 := d165
		snap633 := d166
		snap634 := d167
		snap635 := d168
		snap636 := d169
		snap637 := d170
		snap638 := d171
		snap639 := d172
		snap640 := d173
		snap641 := d174
		snap642 := d175
		snap643 := d176
		snap644 := d177
		snap645 := d178
		snap646 := d179
		snap647 := d180
		snap648 := d181
		snap649 := d184
		snap650 := d335
		snap651 := d336
		snap652 := d337
		snap653 := d338
		snap654 := d340
		snap655 := d341
		snap656 := d342
		snap657 := d343
		snap658 := d344
		snap659 := d345
		snap660 := d346
		snap661 := d347
		snap662 := d349
		snap663 := d351
		snap664 := d352
		snap665 := d353
		snap666 := d444
		snap667 := d445
		snap668 := d448
		snap669 := d542
		snap670 := d543
		snap671 := d544
		snap672 := d545
		snap673 := d546
		snap674 := d548
		snap675 := d549
		snap676 := d550
		snap677 := d551
		snap678 := d552
		snap679 := d553
		snap680 := d554
		snap681 := d555
		snap682 := d556
		snap683 := d557
		snap684 := d558
		snap685 := d559
		snap686 := d560
		snap687 := d561
		snap688 := d562
		snap689 := d563
		snap690 := d564
		snap691 := d565
		snap692 := d566
		snap693 := d567
		snap694 := d568
		snap695 := d569
		snap696 := d570
		snap697 := d571
		snap698 := d572
		snap699 := d573
		snap700 := d574
		alloc701 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc701)
		d5 = snap577
		d6 = snap578
		d7 = snap579
		d8 = snap580
		d9 = snap581
		d10 = snap582
		d11 = snap583
		d12 = snap584
		d13 = snap585
		d14 = snap586
		d15 = snap587
		d16 = snap588
		d17 = snap589
		d18 = snap590
		d19 = snap591
		d21 = snap592
		d22 = snap593
		d23 = snap594
		d24 = snap595
		d25 = snap596
		d26 = snap597
		d27 = snap598
		d28 = snap599
		d29 = snap600
		d30 = snap601
		d31 = snap602
		d32 = snap603
		d33 = snap604
		d34 = snap605
		d35 = snap606
		d36 = snap607
		d37 = snap608
		d38 = snap609
		d39 = snap610
		d40 = snap611
		d41 = snap612
		d42 = snap613
		d43 = snap614
		d44 = snap615
		d45 = snap616
		d46 = snap617
		d47 = snap618
		d48 = snap619
		d49 = snap620
		d50 = snap621
		d51 = snap622
		d54 = snap623
		d55 = snap624
		d56 = snap625
		d159 = snap626
		d160 = snap627
		d161 = snap628
		d162 = snap629
		d163 = snap630
		d164 = snap631
		d165 = snap632
		d166 = snap633
		d167 = snap634
		d168 = snap635
		d169 = snap636
		d170 = snap637
		d171 = snap638
		d172 = snap639
		d173 = snap640
		d174 = snap641
		d175 = snap642
		d176 = snap643
		d177 = snap644
		d178 = snap645
		d179 = snap646
		d180 = snap647
		d181 = snap648
		d184 = snap649
		d335 = snap650
		d336 = snap651
		d337 = snap652
		d338 = snap653
		d340 = snap654
		d341 = snap655
		d342 = snap656
		d343 = snap657
		d344 = snap658
		d345 = snap659
		d346 = snap660
		d347 = snap661
		d349 = snap662
		d351 = snap663
		d352 = snap664
		d353 = snap665
		d444 = snap666
		d445 = snap667
		d448 = snap668
		d542 = snap669
		d543 = snap670
		d544 = snap671
		d545 = snap672
		d546 = snap673
		d548 = snap674
		d549 = snap675
		d550 = snap676
		d551 = snap677
		d552 = snap678
		d553 = snap679
		d554 = snap680
		d555 = snap681
		d556 = snap682
		d557 = snap683
		d558 = snap684
		d559 = snap685
		d560 = snap686
		d561 = snap687
		d562 = snap688
		d563 = snap689
		d564 = snap690
		d565 = snap691
		d566 = snap692
		d567 = snap693
		d568 = snap694
		d569 = snap695
		d570 = snap696
		d571 = snap697
		d572 = snap698
		d573 = snap699
		d574 = snap700
		ctx.RestoreAllocState(alloc701)
		d5 = snap577
		d6 = snap578
		d7 = snap579
		d8 = snap580
		d9 = snap581
		d10 = snap582
		d11 = snap583
		d12 = snap584
		d13 = snap585
		d14 = snap586
		d15 = snap587
		d16 = snap588
		d17 = snap589
		d18 = snap590
		d19 = snap591
		d21 = snap592
		d22 = snap593
		d23 = snap594
		d24 = snap595
		d25 = snap596
		d26 = snap597
		d27 = snap598
		d28 = snap599
		d29 = snap600
		d30 = snap601
		d31 = snap602
		d32 = snap603
		d33 = snap604
		d34 = snap605
		d35 = snap606
		d36 = snap607
		d37 = snap608
		d38 = snap609
		d39 = snap610
		d40 = snap611
		d41 = snap612
		d42 = snap613
		d43 = snap614
		d44 = snap615
		d45 = snap616
		d46 = snap617
		d47 = snap618
		d48 = snap619
		d49 = snap620
		d50 = snap621
		d51 = snap622
		d54 = snap623
		d55 = snap624
		d56 = snap625
		d159 = snap626
		d160 = snap627
		d161 = snap628
		d162 = snap629
		d163 = snap630
		d164 = snap631
		d165 = snap632
		d166 = snap633
		d167 = snap634
		d168 = snap635
		d169 = snap636
		d170 = snap637
		d171 = snap638
		d172 = snap639
		d173 = snap640
		d174 = snap641
		d175 = snap642
		d176 = snap643
		d177 = snap644
		d178 = snap645
		d179 = snap646
		d180 = snap647
		d181 = snap648
		d184 = snap649
		d335 = snap650
		d336 = snap651
		d337 = snap652
		d338 = snap653
		d340 = snap654
		d341 = snap655
		d342 = snap656
		d343 = snap657
		d344 = snap658
		d345 = snap659
		d346 = snap660
		d347 = snap661
		d349 = snap662
		d351 = snap663
		d352 = snap664
		d353 = snap665
		d444 = snap666
		d445 = snap667
		d448 = snap668
		d542 = snap669
		d543 = snap670
		d544 = snap671
		d545 = snap672
		d546 = snap673
		d548 = snap674
		d549 = snap675
		d550 = snap676
		d551 = snap677
		d552 = snap678
		d553 = snap679
		d554 = snap680
		d555 = snap681
		d556 = snap682
		d557 = snap683
		d558 = snap684
		d559 = snap685
		d560 = snap686
		d561 = snap687
		d562 = snap688
		d563 = snap689
		d564 = snap690
		d565 = snap691
		d566 = snap692
		d567 = snap693
		d568 = snap694
		d569 = snap695
		d570 = snap696
		d571 = snap697
		d572 = snap698
		d573 = snap699
		d574 = snap700
		ps702 := scm.PhiState{General: true}
		ps702.OverlayValues = make([]scm.JITValueDesc, 575)
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
		ps702.OverlayValues[18] = d18
		ps702.OverlayValues[19] = d19
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
		ps702.OverlayValues[159] = d159
		ps702.OverlayValues[160] = d160
		ps702.OverlayValues[161] = d161
		ps702.OverlayValues[162] = d162
		ps702.OverlayValues[163] = d163
		ps702.OverlayValues[164] = d164
		ps702.OverlayValues[165] = d165
		ps702.OverlayValues[166] = d166
		ps702.OverlayValues[167] = d167
		ps702.OverlayValues[168] = d168
		ps702.OverlayValues[169] = d169
		ps702.OverlayValues[170] = d170
		ps702.OverlayValues[171] = d171
		ps702.OverlayValues[172] = d172
		ps702.OverlayValues[173] = d173
		ps702.OverlayValues[174] = d174
		ps702.OverlayValues[175] = d175
		ps702.OverlayValues[176] = d176
		ps702.OverlayValues[177] = d177
		ps702.OverlayValues[178] = d178
		ps702.OverlayValues[179] = d179
		ps702.OverlayValues[180] = d180
		ps702.OverlayValues[181] = d181
		ps702.OverlayValues[184] = d184
		ps702.OverlayValues[335] = d335
		ps702.OverlayValues[336] = d336
		ps702.OverlayValues[337] = d337
		ps702.OverlayValues[338] = d338
		ps702.OverlayValues[340] = d340
		ps702.OverlayValues[341] = d341
		ps702.OverlayValues[342] = d342
		ps702.OverlayValues[343] = d343
		ps702.OverlayValues[344] = d344
		ps702.OverlayValues[345] = d345
		ps702.OverlayValues[346] = d346
		ps702.OverlayValues[347] = d347
		ps702.OverlayValues[349] = d349
		ps702.OverlayValues[351] = d351
		ps702.OverlayValues[352] = d352
		ps702.OverlayValues[353] = d353
		ps702.OverlayValues[444] = d444
		ps702.OverlayValues[445] = d445
		ps702.OverlayValues[448] = d448
		ps702.OverlayValues[542] = d542
		ps702.OverlayValues[543] = d543
		ps702.OverlayValues[544] = d544
		ps702.OverlayValues[545] = d545
		ps702.OverlayValues[546] = d546
		ps702.OverlayValues[548] = d548
		ps702.OverlayValues[549] = d549
		ps702.OverlayValues[550] = d550
		ps702.OverlayValues[551] = d551
		ps702.OverlayValues[552] = d552
		ps702.OverlayValues[553] = d553
		ps702.OverlayValues[554] = d554
		ps702.OverlayValues[555] = d555
		ps702.OverlayValues[556] = d556
		ps702.OverlayValues[557] = d557
		ps702.OverlayValues[558] = d558
		ps702.OverlayValues[559] = d559
		ps702.OverlayValues[560] = d560
		ps702.OverlayValues[561] = d561
		ps702.OverlayValues[562] = d562
		ps702.OverlayValues[563] = d563
		ps702.OverlayValues[564] = d564
		ps702.OverlayValues[565] = d565
		ps702.OverlayValues[566] = d566
		ps702.OverlayValues[567] = d567
		ps702.OverlayValues[568] = d568
		ps702.OverlayValues[569] = d569
		ps702.OverlayValues[570] = d570
		ps702.OverlayValues[571] = d571
		ps702.OverlayValues[572] = d572
		ps702.OverlayValues[573] = d573
		ps702.OverlayValues[574] = d574
		ps703 := scm.PhiState{General: true}
		ps703.OverlayValues = make([]scm.JITValueDesc, 575)
		ps703.OverlayValues[5] = d5
		ps703.OverlayValues[6] = d6
		ps703.OverlayValues[7] = d7
		ps703.OverlayValues[8] = d8
		ps703.OverlayValues[9] = d9
		ps703.OverlayValues[10] = d10
		ps703.OverlayValues[11] = d11
		ps703.OverlayValues[12] = d12
		ps703.OverlayValues[13] = d13
		ps703.OverlayValues[14] = d14
		ps703.OverlayValues[15] = d15
		ps703.OverlayValues[16] = d16
		ps703.OverlayValues[17] = d17
		ps703.OverlayValues[18] = d18
		ps703.OverlayValues[19] = d19
		ps703.OverlayValues[21] = d21
		ps703.OverlayValues[22] = d22
		ps703.OverlayValues[23] = d23
		ps703.OverlayValues[24] = d24
		ps703.OverlayValues[25] = d25
		ps703.OverlayValues[26] = d26
		ps703.OverlayValues[27] = d27
		ps703.OverlayValues[28] = d28
		ps703.OverlayValues[29] = d29
		ps703.OverlayValues[30] = d30
		ps703.OverlayValues[31] = d31
		ps703.OverlayValues[32] = d32
		ps703.OverlayValues[33] = d33
		ps703.OverlayValues[34] = d34
		ps703.OverlayValues[35] = d35
		ps703.OverlayValues[36] = d36
		ps703.OverlayValues[37] = d37
		ps703.OverlayValues[38] = d38
		ps703.OverlayValues[39] = d39
		ps703.OverlayValues[40] = d40
		ps703.OverlayValues[41] = d41
		ps703.OverlayValues[42] = d42
		ps703.OverlayValues[43] = d43
		ps703.OverlayValues[44] = d44
		ps703.OverlayValues[45] = d45
		ps703.OverlayValues[46] = d46
		ps703.OverlayValues[47] = d47
		ps703.OverlayValues[48] = d48
		ps703.OverlayValues[49] = d49
		ps703.OverlayValues[50] = d50
		ps703.OverlayValues[51] = d51
		ps703.OverlayValues[54] = d54
		ps703.OverlayValues[55] = d55
		ps703.OverlayValues[56] = d56
		ps703.OverlayValues[159] = d159
		ps703.OverlayValues[160] = d160
		ps703.OverlayValues[161] = d161
		ps703.OverlayValues[162] = d162
		ps703.OverlayValues[163] = d163
		ps703.OverlayValues[164] = d164
		ps703.OverlayValues[165] = d165
		ps703.OverlayValues[166] = d166
		ps703.OverlayValues[167] = d167
		ps703.OverlayValues[168] = d168
		ps703.OverlayValues[169] = d169
		ps703.OverlayValues[170] = d170
		ps703.OverlayValues[171] = d171
		ps703.OverlayValues[172] = d172
		ps703.OverlayValues[173] = d173
		ps703.OverlayValues[174] = d174
		ps703.OverlayValues[175] = d175
		ps703.OverlayValues[176] = d176
		ps703.OverlayValues[177] = d177
		ps703.OverlayValues[178] = d178
		ps703.OverlayValues[179] = d179
		ps703.OverlayValues[180] = d180
		ps703.OverlayValues[181] = d181
		ps703.OverlayValues[184] = d184
		ps703.OverlayValues[335] = d335
		ps703.OverlayValues[336] = d336
		ps703.OverlayValues[337] = d337
		ps703.OverlayValues[338] = d338
		ps703.OverlayValues[340] = d340
		ps703.OverlayValues[341] = d341
		ps703.OverlayValues[342] = d342
		ps703.OverlayValues[343] = d343
		ps703.OverlayValues[344] = d344
		ps703.OverlayValues[345] = d345
		ps703.OverlayValues[346] = d346
		ps703.OverlayValues[347] = d347
		ps703.OverlayValues[349] = d349
		ps703.OverlayValues[351] = d351
		ps703.OverlayValues[352] = d352
		ps703.OverlayValues[353] = d353
		ps703.OverlayValues[444] = d444
		ps703.OverlayValues[445] = d445
		ps703.OverlayValues[448] = d448
		ps703.OverlayValues[542] = d542
		ps703.OverlayValues[543] = d543
		ps703.OverlayValues[544] = d544
		ps703.OverlayValues[545] = d545
		ps703.OverlayValues[546] = d546
		ps703.OverlayValues[548] = d548
		ps703.OverlayValues[549] = d549
		ps703.OverlayValues[550] = d550
		ps703.OverlayValues[551] = d551
		ps703.OverlayValues[552] = d552
		ps703.OverlayValues[553] = d553
		ps703.OverlayValues[554] = d554
		ps703.OverlayValues[555] = d555
		ps703.OverlayValues[556] = d556
		ps703.OverlayValues[557] = d557
		ps703.OverlayValues[558] = d558
		ps703.OverlayValues[559] = d559
		ps703.OverlayValues[560] = d560
		ps703.OverlayValues[561] = d561
		ps703.OverlayValues[562] = d562
		ps703.OverlayValues[563] = d563
		ps703.OverlayValues[564] = d564
		ps703.OverlayValues[565] = d565
		ps703.OverlayValues[566] = d566
		ps703.OverlayValues[567] = d567
		ps703.OverlayValues[568] = d568
		ps703.OverlayValues[569] = d569
		ps703.OverlayValues[570] = d570
		ps703.OverlayValues[571] = d571
		ps703.OverlayValues[572] = d572
		ps703.OverlayValues[573] = d573
		ps703.OverlayValues[574] = d574
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
		snap717 := d18
		snap718 := d19
		snap719 := d21
		snap720 := d22
		snap721 := d23
		snap722 := d24
		snap723 := d25
		snap724 := d26
		snap725 := d27
		snap726 := d28
		snap727 := d29
		snap728 := d30
		snap729 := d31
		snap730 := d32
		snap731 := d33
		snap732 := d34
		snap733 := d35
		snap734 := d36
		snap735 := d37
		snap736 := d38
		snap737 := d39
		snap738 := d40
		snap739 := d41
		snap740 := d42
		snap741 := d43
		snap742 := d44
		snap743 := d45
		snap744 := d46
		snap745 := d47
		snap746 := d48
		snap747 := d49
		snap748 := d50
		snap749 := d51
		snap750 := d54
		snap751 := d55
		snap752 := d56
		snap753 := d159
		snap754 := d160
		snap755 := d161
		snap756 := d162
		snap757 := d163
		snap758 := d164
		snap759 := d165
		snap760 := d166
		snap761 := d167
		snap762 := d168
		snap763 := d169
		snap764 := d170
		snap765 := d171
		snap766 := d172
		snap767 := d173
		snap768 := d174
		snap769 := d175
		snap770 := d176
		snap771 := d177
		snap772 := d178
		snap773 := d179
		snap774 := d180
		snap775 := d181
		snap776 := d184
		snap777 := d335
		snap778 := d336
		snap779 := d337
		snap780 := d338
		snap781 := d340
		snap782 := d341
		snap783 := d342
		snap784 := d343
		snap785 := d344
		snap786 := d345
		snap787 := d346
		snap788 := d347
		snap789 := d349
		snap790 := d351
		snap791 := d352
		snap792 := d353
		snap793 := d444
		snap794 := d445
		snap795 := d448
		snap796 := d542
		snap797 := d543
		snap798 := d544
		snap799 := d545
		snap800 := d546
		snap801 := d548
		snap802 := d549
		snap803 := d550
		snap804 := d551
		snap805 := d552
		snap806 := d553
		snap807 := d554
		snap808 := d555
		snap809 := d556
		snap810 := d557
		snap811 := d558
		snap812 := d559
		snap813 := d560
		snap814 := d561
		snap815 := d562
		snap816 := d563
		snap817 := d564
		snap818 := d565
		snap819 := d566
		snap820 := d567
		snap821 := d568
		snap822 := d569
		snap823 := d570
		snap824 := d571
		snap825 := d572
		snap826 := d573
		snap827 := d574
		alloc828 := ctx.SnapshotAllocState()
		if !bbs[9].Rendered {
			bbs[9].RenderPS(ps703)
		}
		ctx.RestoreAllocState(alloc828)
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
		d18 = snap717
		d19 = snap718
		d21 = snap719
		d22 = snap720
		d23 = snap721
		d24 = snap722
		d25 = snap723
		d26 = snap724
		d27 = snap725
		d28 = snap726
		d29 = snap727
		d30 = snap728
		d31 = snap729
		d32 = snap730
		d33 = snap731
		d34 = snap732
		d35 = snap733
		d36 = snap734
		d37 = snap735
		d38 = snap736
		d39 = snap737
		d40 = snap738
		d41 = snap739
		d42 = snap740
		d43 = snap741
		d44 = snap742
		d45 = snap743
		d46 = snap744
		d47 = snap745
		d48 = snap746
		d49 = snap747
		d50 = snap748
		d51 = snap749
		d54 = snap750
		d55 = snap751
		d56 = snap752
		d159 = snap753
		d160 = snap754
		d161 = snap755
		d162 = snap756
		d163 = snap757
		d164 = snap758
		d165 = snap759
		d166 = snap760
		d167 = snap761
		d168 = snap762
		d169 = snap763
		d170 = snap764
		d171 = snap765
		d172 = snap766
		d173 = snap767
		d174 = snap768
		d175 = snap769
		d176 = snap770
		d177 = snap771
		d178 = snap772
		d179 = snap773
		d180 = snap774
		d181 = snap775
		d184 = snap776
		d335 = snap777
		d336 = snap778
		d337 = snap779
		d338 = snap780
		d340 = snap781
		d341 = snap782
		d342 = snap783
		d343 = snap784
		d344 = snap785
		d345 = snap786
		d346 = snap787
		d347 = snap788
		d349 = snap789
		d351 = snap790
		d352 = snap791
		d353 = snap792
		d444 = snap793
		d445 = snap794
		d448 = snap795
		d542 = snap796
		d543 = snap797
		d544 = snap798
		d545 = snap799
		d546 = snap800
		d548 = snap801
		d549 = snap802
		d550 = snap803
		d551 = snap804
		d552 = snap805
		d553 = snap806
		d554 = snap807
		d555 = snap808
		d556 = snap809
		d557 = snap810
		d558 = snap811
		d559 = snap812
		d560 = snap813
		d561 = snap814
		d562 = snap815
		d563 = snap816
		d564 = snap817
		d565 = snap818
		d566 = snap819
		d567 = snap820
		d568 = snap821
		d569 = snap822
		d570 = snap823
		d571 = snap824
		d572 = snap825
		d573 = snap826
		d574 = snap827
		if !bbs[7].Rendered {
			return bbs[7].RenderPS(ps702)
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
			ctx.FlushRegisterMoves()
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
		if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
			d184 = ps.OverlayValues[184]
		}
		if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != scm.LocNone {
			d335 = ps.OverlayValues[335]
		}
		if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != scm.LocNone {
			d336 = ps.OverlayValues[336]
		}
		if len(ps.OverlayValues) > 337 && ps.OverlayValues[337].Loc != scm.LocNone {
			d337 = ps.OverlayValues[337]
		}
		if len(ps.OverlayValues) > 338 && ps.OverlayValues[338].Loc != scm.LocNone {
			d338 = ps.OverlayValues[338]
		}
		if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != scm.LocNone {
			d340 = ps.OverlayValues[340]
		}
		if len(ps.OverlayValues) > 341 && ps.OverlayValues[341].Loc != scm.LocNone {
			d341 = ps.OverlayValues[341]
		}
		if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != scm.LocNone {
			d342 = ps.OverlayValues[342]
		}
		if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != scm.LocNone {
			d343 = ps.OverlayValues[343]
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
		if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != scm.LocNone {
			d351 = ps.OverlayValues[351]
		}
		if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != scm.LocNone {
			d352 = ps.OverlayValues[352]
		}
		if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != scm.LocNone {
			d353 = ps.OverlayValues[353]
		}
		if len(ps.OverlayValues) > 444 && ps.OverlayValues[444].Loc != scm.LocNone {
			d444 = ps.OverlayValues[444]
		}
		if len(ps.OverlayValues) > 445 && ps.OverlayValues[445].Loc != scm.LocNone {
			d445 = ps.OverlayValues[445]
		}
		if len(ps.OverlayValues) > 448 && ps.OverlayValues[448].Loc != scm.LocNone {
			d448 = ps.OverlayValues[448]
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
		if len(ps.OverlayValues) > 548 && ps.OverlayValues[548].Loc != scm.LocNone {
			d548 = ps.OverlayValues[548]
		}
		if len(ps.OverlayValues) > 549 && ps.OverlayValues[549].Loc != scm.LocNone {
			d549 = ps.OverlayValues[549]
		}
		if len(ps.OverlayValues) > 550 && ps.OverlayValues[550].Loc != scm.LocNone {
			d550 = ps.OverlayValues[550]
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
		if len(ps.OverlayValues) > 555 && ps.OverlayValues[555].Loc != scm.LocNone {
			d555 = ps.OverlayValues[555]
		}
		if len(ps.OverlayValues) > 556 && ps.OverlayValues[556].Loc != scm.LocNone {
			d556 = ps.OverlayValues[556]
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
		if len(ps.OverlayValues) > 562 && ps.OverlayValues[562].Loc != scm.LocNone {
			d562 = ps.OverlayValues[562]
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d9)
		ctx.EnsureDesc(&d9)
		var d829 scm.JITValueDesc
		if d9.Loc == scm.LocImm {
			d829 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d9.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d9.Reg)
			ctx.EmitMovRegReg(scratch, d9.Reg)
			ctx.EmitSubRegImm32Low(scratch, int32(1))
			d829 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d829)
		}
		if d829.Loc == scm.LocReg && d9.Loc == scm.LocReg && d829.Reg == d9.Reg {
			ctx.TransferReg(d9.Reg)
			d9.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d829)
		ctx.EmitStoreToStack(d829, int32(bbs[8].PhiBase)+int32(16))
		ctx.StabilizeDescForControlFlow(&d829)
		if ps.General {
			ctx.SyncDesc(&d10)
			if d10.Loc == scm.LocReg || d10.Loc == scm.LocFPReg {
				ctx.ProtectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.ProtectReg(d10.Reg)
				ctx.ProtectReg(d10.Reg2)
			}
			d830 = d10
			if d830.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d830)
			d831 = d830
			if d831.Loc == scm.LocImm {
				d831 = scm.JITValueDesc{Loc: scm.LocImm, Type: d831.Type, Imm: scm.NewInt(int64(uint64(d831.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d831.Reg, 32)
				ctx.EmitShrRegImm8(d831.Reg, 32)
			}
			ctx.EmitStoreToStack(d831, int32(bbs[8].PhiBase)+int32(0))
			if d10.Loc == scm.LocReg || d10.Loc == scm.LocFPReg {
				ctx.UnprotectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d10.Reg)
				ctx.UnprotectReg(d10.Reg2)
			}
		}
		ps832 := scm.PhiState{General: ps.General}
		ps832.OverlayValues = make([]scm.JITValueDesc, 832)
		ps832.OverlayValues[5] = d5
		ps832.OverlayValues[6] = d6
		ps832.OverlayValues[7] = d7
		ps832.OverlayValues[8] = d8
		ps832.OverlayValues[9] = d9
		ps832.OverlayValues[10] = d10
		ps832.OverlayValues[11] = d11
		ps832.OverlayValues[12] = d12
		ps832.OverlayValues[13] = d13
		ps832.OverlayValues[14] = d14
		ps832.OverlayValues[15] = d15
		ps832.OverlayValues[16] = d16
		ps832.OverlayValues[17] = d17
		ps832.OverlayValues[18] = d18
		ps832.OverlayValues[19] = d19
		ps832.OverlayValues[21] = d21
		ps832.OverlayValues[22] = d22
		ps832.OverlayValues[23] = d23
		ps832.OverlayValues[24] = d24
		ps832.OverlayValues[25] = d25
		ps832.OverlayValues[26] = d26
		ps832.OverlayValues[27] = d27
		ps832.OverlayValues[28] = d28
		ps832.OverlayValues[29] = d29
		ps832.OverlayValues[30] = d30
		ps832.OverlayValues[31] = d31
		ps832.OverlayValues[32] = d32
		ps832.OverlayValues[33] = d33
		ps832.OverlayValues[34] = d34
		ps832.OverlayValues[35] = d35
		ps832.OverlayValues[36] = d36
		ps832.OverlayValues[37] = d37
		ps832.OverlayValues[38] = d38
		ps832.OverlayValues[39] = d39
		ps832.OverlayValues[40] = d40
		ps832.OverlayValues[41] = d41
		ps832.OverlayValues[42] = d42
		ps832.OverlayValues[43] = d43
		ps832.OverlayValues[44] = d44
		ps832.OverlayValues[45] = d45
		ps832.OverlayValues[46] = d46
		ps832.OverlayValues[47] = d47
		ps832.OverlayValues[48] = d48
		ps832.OverlayValues[49] = d49
		ps832.OverlayValues[50] = d50
		ps832.OverlayValues[51] = d51
		ps832.OverlayValues[54] = d54
		ps832.OverlayValues[55] = d55
		ps832.OverlayValues[56] = d56
		ps832.OverlayValues[159] = d159
		ps832.OverlayValues[160] = d160
		ps832.OverlayValues[161] = d161
		ps832.OverlayValues[162] = d162
		ps832.OverlayValues[163] = d163
		ps832.OverlayValues[164] = d164
		ps832.OverlayValues[165] = d165
		ps832.OverlayValues[166] = d166
		ps832.OverlayValues[167] = d167
		ps832.OverlayValues[168] = d168
		ps832.OverlayValues[169] = d169
		ps832.OverlayValues[170] = d170
		ps832.OverlayValues[171] = d171
		ps832.OverlayValues[172] = d172
		ps832.OverlayValues[173] = d173
		ps832.OverlayValues[174] = d174
		ps832.OverlayValues[175] = d175
		ps832.OverlayValues[176] = d176
		ps832.OverlayValues[177] = d177
		ps832.OverlayValues[178] = d178
		ps832.OverlayValues[179] = d179
		ps832.OverlayValues[180] = d180
		ps832.OverlayValues[181] = d181
		ps832.OverlayValues[184] = d184
		ps832.OverlayValues[335] = d335
		ps832.OverlayValues[336] = d336
		ps832.OverlayValues[337] = d337
		ps832.OverlayValues[338] = d338
		ps832.OverlayValues[340] = d340
		ps832.OverlayValues[341] = d341
		ps832.OverlayValues[342] = d342
		ps832.OverlayValues[343] = d343
		ps832.OverlayValues[344] = d344
		ps832.OverlayValues[345] = d345
		ps832.OverlayValues[346] = d346
		ps832.OverlayValues[347] = d347
		ps832.OverlayValues[349] = d349
		ps832.OverlayValues[351] = d351
		ps832.OverlayValues[352] = d352
		ps832.OverlayValues[353] = d353
		ps832.OverlayValues[444] = d444
		ps832.OverlayValues[445] = d445
		ps832.OverlayValues[448] = d448
		ps832.OverlayValues[542] = d542
		ps832.OverlayValues[543] = d543
		ps832.OverlayValues[544] = d544
		ps832.OverlayValues[545] = d545
		ps832.OverlayValues[546] = d546
		ps832.OverlayValues[548] = d548
		ps832.OverlayValues[549] = d549
		ps832.OverlayValues[550] = d550
		ps832.OverlayValues[551] = d551
		ps832.OverlayValues[552] = d552
		ps832.OverlayValues[553] = d553
		ps832.OverlayValues[554] = d554
		ps832.OverlayValues[555] = d555
		ps832.OverlayValues[556] = d556
		ps832.OverlayValues[557] = d557
		ps832.OverlayValues[558] = d558
		ps832.OverlayValues[559] = d559
		ps832.OverlayValues[560] = d560
		ps832.OverlayValues[561] = d561
		ps832.OverlayValues[562] = d562
		ps832.OverlayValues[563] = d563
		ps832.OverlayValues[564] = d564
		ps832.OverlayValues[565] = d565
		ps832.OverlayValues[566] = d566
		ps832.OverlayValues[567] = d567
		ps832.OverlayValues[568] = d568
		ps832.OverlayValues[569] = d569
		ps832.OverlayValues[570] = d570
		ps832.OverlayValues[571] = d571
		ps832.OverlayValues[572] = d572
		ps832.OverlayValues[573] = d573
		ps832.OverlayValues[574] = d574
		ps832.OverlayValues[829] = d829
		ps832.OverlayValues[830] = d830
		ps832.OverlayValues[831] = d831
		ps832.PhiValues = make([]scm.JITValueDesc, 2)
		d833 = d10
		ps832.PhiValues[0] = d833
		if ps832.General && bbs[8].Rendered {
			ctx.EmitJmp(lbl9)
			return result
		}
		return bbs[8].RenderPS(ps832)
		return result
	}
	bbs[8].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d834 := ps.PhiValues[0]
				ctx.EnsureDesc(&d834)
				ctx.EmitStoreToStack(d834, int32(bbs[8].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d835 := ps.PhiValues[1]
				ctx.EnsureDesc(&d835)
				ctx.EmitStoreToStack(d835, int32(bbs[8].PhiBase)+int32(16))
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
			ctx.FlushRegisterMoves()
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
		if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
			d184 = ps.OverlayValues[184]
		}
		if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != scm.LocNone {
			d335 = ps.OverlayValues[335]
		}
		if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != scm.LocNone {
			d336 = ps.OverlayValues[336]
		}
		if len(ps.OverlayValues) > 337 && ps.OverlayValues[337].Loc != scm.LocNone {
			d337 = ps.OverlayValues[337]
		}
		if len(ps.OverlayValues) > 338 && ps.OverlayValues[338].Loc != scm.LocNone {
			d338 = ps.OverlayValues[338]
		}
		if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != scm.LocNone {
			d340 = ps.OverlayValues[340]
		}
		if len(ps.OverlayValues) > 341 && ps.OverlayValues[341].Loc != scm.LocNone {
			d341 = ps.OverlayValues[341]
		}
		if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != scm.LocNone {
			d342 = ps.OverlayValues[342]
		}
		if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != scm.LocNone {
			d343 = ps.OverlayValues[343]
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
		if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != scm.LocNone {
			d351 = ps.OverlayValues[351]
		}
		if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != scm.LocNone {
			d352 = ps.OverlayValues[352]
		}
		if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != scm.LocNone {
			d353 = ps.OverlayValues[353]
		}
		if len(ps.OverlayValues) > 444 && ps.OverlayValues[444].Loc != scm.LocNone {
			d444 = ps.OverlayValues[444]
		}
		if len(ps.OverlayValues) > 445 && ps.OverlayValues[445].Loc != scm.LocNone {
			d445 = ps.OverlayValues[445]
		}
		if len(ps.OverlayValues) > 448 && ps.OverlayValues[448].Loc != scm.LocNone {
			d448 = ps.OverlayValues[448]
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
		if len(ps.OverlayValues) > 548 && ps.OverlayValues[548].Loc != scm.LocNone {
			d548 = ps.OverlayValues[548]
		}
		if len(ps.OverlayValues) > 549 && ps.OverlayValues[549].Loc != scm.LocNone {
			d549 = ps.OverlayValues[549]
		}
		if len(ps.OverlayValues) > 550 && ps.OverlayValues[550].Loc != scm.LocNone {
			d550 = ps.OverlayValues[550]
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
		if len(ps.OverlayValues) > 555 && ps.OverlayValues[555].Loc != scm.LocNone {
			d555 = ps.OverlayValues[555]
		}
		if len(ps.OverlayValues) > 556 && ps.OverlayValues[556].Loc != scm.LocNone {
			d556 = ps.OverlayValues[556]
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
		if len(ps.OverlayValues) > 562 && ps.OverlayValues[562].Loc != scm.LocNone {
			d562 = ps.OverlayValues[562]
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
		if len(ps.OverlayValues) > 829 && ps.OverlayValues[829].Loc != scm.LocNone {
			d829 = ps.OverlayValues[829]
		}
		if len(ps.OverlayValues) > 830 && ps.OverlayValues[830].Loc != scm.LocNone {
			d830 = ps.OverlayValues[830]
		}
		if len(ps.OverlayValues) > 831 && ps.OverlayValues[831].Loc != scm.LocNone {
			d831 = ps.OverlayValues[831]
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
		var d836 scm.JITValueDesc
		if d12.Loc == scm.LocImm && d13.Loc == scm.LocImm {
			d836 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d12.Imm.Int()) == uint64(d13.Imm.Int()))}
		} else if d13.Loc == scm.LocImm {
			r93 := ctx.AllocRegExcept(d12.Reg)
			if d13.Imm.Int() >= -2147483648 && d13.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d12.Reg, int32(d13.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
				ctx.EmitCmpInt64(d12.Reg, scm.RegR11)
			}
			d836 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r93, Condition: scm.CondEqual}
			ctx.BindReg(r93, &d836)
		} else if d12.Loc == scm.LocImm {
			r94 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d12.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d13.Reg)
			d836 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r94, Condition: scm.CondEqual}
			ctx.BindReg(r94, &d836)
		} else {
			r95 := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitCmpInt64(d12.Reg, d13.Reg)
			d836 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r95, Condition: scm.CondEqual}
			ctx.BindReg(r95, &d836)
		}
		d837 = d836
		ctx.EnsureDesc(&d837)
		if d837.Loc != scm.LocImm && d837.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d837.Loc == scm.LocImm {
			if d837.Imm.Bool() {
				if ps.General {
					ctx.SyncDesc(&d12)
					if d12.Loc == scm.LocReg || d12.Loc == scm.LocFPReg {
						ctx.ProtectReg(d12.Reg)
					} else if d12.Loc == scm.LocRegPair {
						ctx.ProtectReg(d12.Reg)
						ctx.ProtectReg(d12.Reg2)
					}
					d838 = d12
					if d838.Loc == scm.LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d838)
					d839 = d838
					if d839.Loc == scm.LocImm {
						d839 = scm.JITValueDesc{Loc: scm.LocImm, Type: d839.Type, Imm: scm.NewInt(int64(uint64(d839.Imm.Int()) & 0xffffffff))}
					} else {
						ctx.EmitShlRegImm8(d839.Reg, 32)
						ctx.EmitShrRegImm8(d839.Reg, 32)
					}
					ctx.EmitStoreToStack(d839, int32(bbs[2].PhiBase)+int32(0))
					if d12.Loc == scm.LocReg || d12.Loc == scm.LocFPReg {
						ctx.UnprotectReg(d12.Reg)
					} else if d12.Loc == scm.LocRegPair {
						ctx.UnprotectReg(d12.Reg)
						ctx.UnprotectReg(d12.Reg2)
					}
				}
				ps840 := scm.PhiState{General: ps.General}
				ps840.OverlayValues = make([]scm.JITValueDesc, 840)
				ps840.OverlayValues[5] = d5
				ps840.OverlayValues[6] = d6
				ps840.OverlayValues[7] = d7
				ps840.OverlayValues[8] = d8
				ps840.OverlayValues[9] = d9
				ps840.OverlayValues[10] = d10
				ps840.OverlayValues[11] = d11
				ps840.OverlayValues[12] = d12
				ps840.OverlayValues[13] = d13
				ps840.OverlayValues[14] = d14
				ps840.OverlayValues[15] = d15
				ps840.OverlayValues[16] = d16
				ps840.OverlayValues[17] = d17
				ps840.OverlayValues[18] = d18
				ps840.OverlayValues[19] = d19
				ps840.OverlayValues[21] = d21
				ps840.OverlayValues[22] = d22
				ps840.OverlayValues[23] = d23
				ps840.OverlayValues[24] = d24
				ps840.OverlayValues[25] = d25
				ps840.OverlayValues[26] = d26
				ps840.OverlayValues[27] = d27
				ps840.OverlayValues[28] = d28
				ps840.OverlayValues[29] = d29
				ps840.OverlayValues[30] = d30
				ps840.OverlayValues[31] = d31
				ps840.OverlayValues[32] = d32
				ps840.OverlayValues[33] = d33
				ps840.OverlayValues[34] = d34
				ps840.OverlayValues[35] = d35
				ps840.OverlayValues[36] = d36
				ps840.OverlayValues[37] = d37
				ps840.OverlayValues[38] = d38
				ps840.OverlayValues[39] = d39
				ps840.OverlayValues[40] = d40
				ps840.OverlayValues[41] = d41
				ps840.OverlayValues[42] = d42
				ps840.OverlayValues[43] = d43
				ps840.OverlayValues[44] = d44
				ps840.OverlayValues[45] = d45
				ps840.OverlayValues[46] = d46
				ps840.OverlayValues[47] = d47
				ps840.OverlayValues[48] = d48
				ps840.OverlayValues[49] = d49
				ps840.OverlayValues[50] = d50
				ps840.OverlayValues[51] = d51
				ps840.OverlayValues[54] = d54
				ps840.OverlayValues[55] = d55
				ps840.OverlayValues[56] = d56
				ps840.OverlayValues[159] = d159
				ps840.OverlayValues[160] = d160
				ps840.OverlayValues[161] = d161
				ps840.OverlayValues[162] = d162
				ps840.OverlayValues[163] = d163
				ps840.OverlayValues[164] = d164
				ps840.OverlayValues[165] = d165
				ps840.OverlayValues[166] = d166
				ps840.OverlayValues[167] = d167
				ps840.OverlayValues[168] = d168
				ps840.OverlayValues[169] = d169
				ps840.OverlayValues[170] = d170
				ps840.OverlayValues[171] = d171
				ps840.OverlayValues[172] = d172
				ps840.OverlayValues[173] = d173
				ps840.OverlayValues[174] = d174
				ps840.OverlayValues[175] = d175
				ps840.OverlayValues[176] = d176
				ps840.OverlayValues[177] = d177
				ps840.OverlayValues[178] = d178
				ps840.OverlayValues[179] = d179
				ps840.OverlayValues[180] = d180
				ps840.OverlayValues[181] = d181
				ps840.OverlayValues[184] = d184
				ps840.OverlayValues[335] = d335
				ps840.OverlayValues[336] = d336
				ps840.OverlayValues[337] = d337
				ps840.OverlayValues[338] = d338
				ps840.OverlayValues[340] = d340
				ps840.OverlayValues[341] = d341
				ps840.OverlayValues[342] = d342
				ps840.OverlayValues[343] = d343
				ps840.OverlayValues[344] = d344
				ps840.OverlayValues[345] = d345
				ps840.OverlayValues[346] = d346
				ps840.OverlayValues[347] = d347
				ps840.OverlayValues[349] = d349
				ps840.OverlayValues[351] = d351
				ps840.OverlayValues[352] = d352
				ps840.OverlayValues[353] = d353
				ps840.OverlayValues[444] = d444
				ps840.OverlayValues[445] = d445
				ps840.OverlayValues[448] = d448
				ps840.OverlayValues[542] = d542
				ps840.OverlayValues[543] = d543
				ps840.OverlayValues[544] = d544
				ps840.OverlayValues[545] = d545
				ps840.OverlayValues[546] = d546
				ps840.OverlayValues[548] = d548
				ps840.OverlayValues[549] = d549
				ps840.OverlayValues[550] = d550
				ps840.OverlayValues[551] = d551
				ps840.OverlayValues[552] = d552
				ps840.OverlayValues[553] = d553
				ps840.OverlayValues[554] = d554
				ps840.OverlayValues[555] = d555
				ps840.OverlayValues[556] = d556
				ps840.OverlayValues[557] = d557
				ps840.OverlayValues[558] = d558
				ps840.OverlayValues[559] = d559
				ps840.OverlayValues[560] = d560
				ps840.OverlayValues[561] = d561
				ps840.OverlayValues[562] = d562
				ps840.OverlayValues[563] = d563
				ps840.OverlayValues[564] = d564
				ps840.OverlayValues[565] = d565
				ps840.OverlayValues[566] = d566
				ps840.OverlayValues[567] = d567
				ps840.OverlayValues[568] = d568
				ps840.OverlayValues[569] = d569
				ps840.OverlayValues[570] = d570
				ps840.OverlayValues[571] = d571
				ps840.OverlayValues[572] = d572
				ps840.OverlayValues[573] = d573
				ps840.OverlayValues[574] = d574
				ps840.OverlayValues[829] = d829
				ps840.OverlayValues[830] = d830
				ps840.OverlayValues[831] = d831
				ps840.OverlayValues[833] = d833
				ps840.OverlayValues[834] = d834
				ps840.OverlayValues[835] = d835
				ps840.OverlayValues[836] = d836
				ps840.OverlayValues[837] = d837
				ps840.OverlayValues[838] = d838
				ps840.OverlayValues[839] = d839
				ps840.PhiValues = make([]scm.JITValueDesc, 1)
				d841 = d12
				ps840.PhiValues[0] = d841
				return bbs[2].RenderPS(ps840)
			}
			if ps.General {
			}
			ps842 := scm.PhiState{General: ps.General}
			ps842.OverlayValues = make([]scm.JITValueDesc, 842)
			ps842.OverlayValues[5] = d5
			ps842.OverlayValues[6] = d6
			ps842.OverlayValues[7] = d7
			ps842.OverlayValues[8] = d8
			ps842.OverlayValues[9] = d9
			ps842.OverlayValues[10] = d10
			ps842.OverlayValues[11] = d11
			ps842.OverlayValues[12] = d12
			ps842.OverlayValues[13] = d13
			ps842.OverlayValues[14] = d14
			ps842.OverlayValues[15] = d15
			ps842.OverlayValues[16] = d16
			ps842.OverlayValues[17] = d17
			ps842.OverlayValues[18] = d18
			ps842.OverlayValues[19] = d19
			ps842.OverlayValues[21] = d21
			ps842.OverlayValues[22] = d22
			ps842.OverlayValues[23] = d23
			ps842.OverlayValues[24] = d24
			ps842.OverlayValues[25] = d25
			ps842.OverlayValues[26] = d26
			ps842.OverlayValues[27] = d27
			ps842.OverlayValues[28] = d28
			ps842.OverlayValues[29] = d29
			ps842.OverlayValues[30] = d30
			ps842.OverlayValues[31] = d31
			ps842.OverlayValues[32] = d32
			ps842.OverlayValues[33] = d33
			ps842.OverlayValues[34] = d34
			ps842.OverlayValues[35] = d35
			ps842.OverlayValues[36] = d36
			ps842.OverlayValues[37] = d37
			ps842.OverlayValues[38] = d38
			ps842.OverlayValues[39] = d39
			ps842.OverlayValues[40] = d40
			ps842.OverlayValues[41] = d41
			ps842.OverlayValues[42] = d42
			ps842.OverlayValues[43] = d43
			ps842.OverlayValues[44] = d44
			ps842.OverlayValues[45] = d45
			ps842.OverlayValues[46] = d46
			ps842.OverlayValues[47] = d47
			ps842.OverlayValues[48] = d48
			ps842.OverlayValues[49] = d49
			ps842.OverlayValues[50] = d50
			ps842.OverlayValues[51] = d51
			ps842.OverlayValues[54] = d54
			ps842.OverlayValues[55] = d55
			ps842.OverlayValues[56] = d56
			ps842.OverlayValues[159] = d159
			ps842.OverlayValues[160] = d160
			ps842.OverlayValues[161] = d161
			ps842.OverlayValues[162] = d162
			ps842.OverlayValues[163] = d163
			ps842.OverlayValues[164] = d164
			ps842.OverlayValues[165] = d165
			ps842.OverlayValues[166] = d166
			ps842.OverlayValues[167] = d167
			ps842.OverlayValues[168] = d168
			ps842.OverlayValues[169] = d169
			ps842.OverlayValues[170] = d170
			ps842.OverlayValues[171] = d171
			ps842.OverlayValues[172] = d172
			ps842.OverlayValues[173] = d173
			ps842.OverlayValues[174] = d174
			ps842.OverlayValues[175] = d175
			ps842.OverlayValues[176] = d176
			ps842.OverlayValues[177] = d177
			ps842.OverlayValues[178] = d178
			ps842.OverlayValues[179] = d179
			ps842.OverlayValues[180] = d180
			ps842.OverlayValues[181] = d181
			ps842.OverlayValues[184] = d184
			ps842.OverlayValues[335] = d335
			ps842.OverlayValues[336] = d336
			ps842.OverlayValues[337] = d337
			ps842.OverlayValues[338] = d338
			ps842.OverlayValues[340] = d340
			ps842.OverlayValues[341] = d341
			ps842.OverlayValues[342] = d342
			ps842.OverlayValues[343] = d343
			ps842.OverlayValues[344] = d344
			ps842.OverlayValues[345] = d345
			ps842.OverlayValues[346] = d346
			ps842.OverlayValues[347] = d347
			ps842.OverlayValues[349] = d349
			ps842.OverlayValues[351] = d351
			ps842.OverlayValues[352] = d352
			ps842.OverlayValues[353] = d353
			ps842.OverlayValues[444] = d444
			ps842.OverlayValues[445] = d445
			ps842.OverlayValues[448] = d448
			ps842.OverlayValues[542] = d542
			ps842.OverlayValues[543] = d543
			ps842.OverlayValues[544] = d544
			ps842.OverlayValues[545] = d545
			ps842.OverlayValues[546] = d546
			ps842.OverlayValues[548] = d548
			ps842.OverlayValues[549] = d549
			ps842.OverlayValues[550] = d550
			ps842.OverlayValues[551] = d551
			ps842.OverlayValues[552] = d552
			ps842.OverlayValues[553] = d553
			ps842.OverlayValues[554] = d554
			ps842.OverlayValues[555] = d555
			ps842.OverlayValues[556] = d556
			ps842.OverlayValues[557] = d557
			ps842.OverlayValues[558] = d558
			ps842.OverlayValues[559] = d559
			ps842.OverlayValues[560] = d560
			ps842.OverlayValues[561] = d561
			ps842.OverlayValues[562] = d562
			ps842.OverlayValues[563] = d563
			ps842.OverlayValues[564] = d564
			ps842.OverlayValues[565] = d565
			ps842.OverlayValues[566] = d566
			ps842.OverlayValues[567] = d567
			ps842.OverlayValues[568] = d568
			ps842.OverlayValues[569] = d569
			ps842.OverlayValues[570] = d570
			ps842.OverlayValues[571] = d571
			ps842.OverlayValues[572] = d572
			ps842.OverlayValues[573] = d573
			ps842.OverlayValues[574] = d574
			ps842.OverlayValues[829] = d829
			ps842.OverlayValues[830] = d830
			ps842.OverlayValues[831] = d831
			ps842.OverlayValues[833] = d833
			ps842.OverlayValues[834] = d834
			ps842.OverlayValues[835] = d835
			ps842.OverlayValues[836] = d836
			ps842.OverlayValues[837] = d837
			ps842.OverlayValues[838] = d838
			ps842.OverlayValues[839] = d839
			ps842.OverlayValues[841] = d841
			return bbs[10].RenderPS(ps842)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d843 := ps.PhiValues[0]
				ctx.EnsureDesc(&d843)
				ctx.EmitStoreToStack(d843, int32(bbs[8].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d844 := ps.PhiValues[1]
				ctx.EnsureDesc(&d844)
				ctx.EmitStoreToStack(d844, int32(bbs[8].PhiBase)+int32(16))
			}
			ps.General = true
			return bbs[8].RenderPS(ps)
		}
		lbl19 := ctx.ReserveLabel()
		ctx.EmitJump(d837.Condition, lbl19)
		ctx.EmitJmp(lbl11)
		ctx.FreeDesc(&d836)
		snap845 := d5
		snap846 := d6
		snap847 := d7
		snap848 := d8
		snap849 := d9
		snap850 := d10
		snap851 := d11
		snap852 := d12
		snap853 := d13
		snap854 := d14
		snap855 := d15
		snap856 := d16
		snap857 := d17
		snap858 := d18
		snap859 := d19
		snap860 := d21
		snap861 := d22
		snap862 := d23
		snap863 := d24
		snap864 := d25
		snap865 := d26
		snap866 := d27
		snap867 := d28
		snap868 := d29
		snap869 := d30
		snap870 := d31
		snap871 := d32
		snap872 := d33
		snap873 := d34
		snap874 := d35
		snap875 := d36
		snap876 := d37
		snap877 := d38
		snap878 := d39
		snap879 := d40
		snap880 := d41
		snap881 := d42
		snap882 := d43
		snap883 := d44
		snap884 := d45
		snap885 := d46
		snap886 := d47
		snap887 := d48
		snap888 := d49
		snap889 := d50
		snap890 := d51
		snap891 := d54
		snap892 := d55
		snap893 := d56
		snap894 := d159
		snap895 := d160
		snap896 := d161
		snap897 := d162
		snap898 := d163
		snap899 := d164
		snap900 := d165
		snap901 := d166
		snap902 := d167
		snap903 := d168
		snap904 := d169
		snap905 := d170
		snap906 := d171
		snap907 := d172
		snap908 := d173
		snap909 := d174
		snap910 := d175
		snap911 := d176
		snap912 := d177
		snap913 := d178
		snap914 := d179
		snap915 := d180
		snap916 := d181
		snap917 := d184
		snap918 := d335
		snap919 := d336
		snap920 := d337
		snap921 := d338
		snap922 := d340
		snap923 := d341
		snap924 := d342
		snap925 := d343
		snap926 := d344
		snap927 := d345
		snap928 := d346
		snap929 := d347
		snap930 := d349
		snap931 := d351
		snap932 := d352
		snap933 := d353
		snap934 := d444
		snap935 := d445
		snap936 := d448
		snap937 := d542
		snap938 := d543
		snap939 := d544
		snap940 := d545
		snap941 := d546
		snap942 := d548
		snap943 := d549
		snap944 := d550
		snap945 := d551
		snap946 := d552
		snap947 := d553
		snap948 := d554
		snap949 := d555
		snap950 := d556
		snap951 := d557
		snap952 := d558
		snap953 := d559
		snap954 := d560
		snap955 := d561
		snap956 := d562
		snap957 := d563
		snap958 := d564
		snap959 := d565
		snap960 := d566
		snap961 := d567
		snap962 := d568
		snap963 := d569
		snap964 := d570
		snap965 := d571
		snap966 := d572
		snap967 := d573
		snap968 := d574
		snap969 := d829
		snap970 := d830
		snap971 := d831
		snap972 := d833
		snap973 := d834
		snap974 := d835
		snap975 := d836
		snap976 := d837
		snap977 := d838
		snap978 := d839
		snap979 := d841
		snap980 := d843
		snap981 := d844
		alloc982 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl19)
		ctx.SyncDesc(&d12)
		if d12.Loc == scm.LocReg || d12.Loc == scm.LocFPReg {
			ctx.ProtectReg(d12.Reg)
		} else if d12.Loc == scm.LocRegPair {
			ctx.ProtectReg(d12.Reg)
			ctx.ProtectReg(d12.Reg2)
		}
		d983 = d12
		if d983.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d983)
		d984 = d983
		if d984.Loc == scm.LocImm {
			d984 = scm.JITValueDesc{Loc: scm.LocImm, Type: d984.Type, Imm: scm.NewInt(int64(uint64(d984.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d984.Reg, 32)
			ctx.EmitShrRegImm8(d984.Reg, 32)
		}
		ctx.EmitStoreToStack(d984, int32(bbs[2].PhiBase)+int32(0))
		if d12.Loc == scm.LocReg || d12.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d12.Reg)
		} else if d12.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d12.Reg)
			ctx.UnprotectReg(d12.Reg2)
		}
		ctx.EmitJmp(lbl3)
		ctx.RestoreAllocState(alloc982)
		d5 = snap845
		d6 = snap846
		d7 = snap847
		d8 = snap848
		d9 = snap849
		d10 = snap850
		d11 = snap851
		d12 = snap852
		d13 = snap853
		d14 = snap854
		d15 = snap855
		d16 = snap856
		d17 = snap857
		d18 = snap858
		d19 = snap859
		d21 = snap860
		d22 = snap861
		d23 = snap862
		d24 = snap863
		d25 = snap864
		d26 = snap865
		d27 = snap866
		d28 = snap867
		d29 = snap868
		d30 = snap869
		d31 = snap870
		d32 = snap871
		d33 = snap872
		d34 = snap873
		d35 = snap874
		d36 = snap875
		d37 = snap876
		d38 = snap877
		d39 = snap878
		d40 = snap879
		d41 = snap880
		d42 = snap881
		d43 = snap882
		d44 = snap883
		d45 = snap884
		d46 = snap885
		d47 = snap886
		d48 = snap887
		d49 = snap888
		d50 = snap889
		d51 = snap890
		d54 = snap891
		d55 = snap892
		d56 = snap893
		d159 = snap894
		d160 = snap895
		d161 = snap896
		d162 = snap897
		d163 = snap898
		d164 = snap899
		d165 = snap900
		d166 = snap901
		d167 = snap902
		d168 = snap903
		d169 = snap904
		d170 = snap905
		d171 = snap906
		d172 = snap907
		d173 = snap908
		d174 = snap909
		d175 = snap910
		d176 = snap911
		d177 = snap912
		d178 = snap913
		d179 = snap914
		d180 = snap915
		d181 = snap916
		d184 = snap917
		d335 = snap918
		d336 = snap919
		d337 = snap920
		d338 = snap921
		d340 = snap922
		d341 = snap923
		d342 = snap924
		d343 = snap925
		d344 = snap926
		d345 = snap927
		d346 = snap928
		d347 = snap929
		d349 = snap930
		d351 = snap931
		d352 = snap932
		d353 = snap933
		d444 = snap934
		d445 = snap935
		d448 = snap936
		d542 = snap937
		d543 = snap938
		d544 = snap939
		d545 = snap940
		d546 = snap941
		d548 = snap942
		d549 = snap943
		d550 = snap944
		d551 = snap945
		d552 = snap946
		d553 = snap947
		d554 = snap948
		d555 = snap949
		d556 = snap950
		d557 = snap951
		d558 = snap952
		d559 = snap953
		d560 = snap954
		d561 = snap955
		d562 = snap956
		d563 = snap957
		d564 = snap958
		d565 = snap959
		d566 = snap960
		d567 = snap961
		d568 = snap962
		d569 = snap963
		d570 = snap964
		d571 = snap965
		d572 = snap966
		d573 = snap967
		d574 = snap968
		d829 = snap969
		d830 = snap970
		d831 = snap971
		d833 = snap972
		d834 = snap973
		d835 = snap974
		d836 = snap975
		d837 = snap976
		d838 = snap977
		d839 = snap978
		d841 = snap979
		d843 = snap980
		d844 = snap981
		ctx.RestoreAllocState(alloc982)
		d5 = snap845
		d6 = snap846
		d7 = snap847
		d8 = snap848
		d9 = snap849
		d10 = snap850
		d11 = snap851
		d12 = snap852
		d13 = snap853
		d14 = snap854
		d15 = snap855
		d16 = snap856
		d17 = snap857
		d18 = snap858
		d19 = snap859
		d21 = snap860
		d22 = snap861
		d23 = snap862
		d24 = snap863
		d25 = snap864
		d26 = snap865
		d27 = snap866
		d28 = snap867
		d29 = snap868
		d30 = snap869
		d31 = snap870
		d32 = snap871
		d33 = snap872
		d34 = snap873
		d35 = snap874
		d36 = snap875
		d37 = snap876
		d38 = snap877
		d39 = snap878
		d40 = snap879
		d41 = snap880
		d42 = snap881
		d43 = snap882
		d44 = snap883
		d45 = snap884
		d46 = snap885
		d47 = snap886
		d48 = snap887
		d49 = snap888
		d50 = snap889
		d51 = snap890
		d54 = snap891
		d55 = snap892
		d56 = snap893
		d159 = snap894
		d160 = snap895
		d161 = snap896
		d162 = snap897
		d163 = snap898
		d164 = snap899
		d165 = snap900
		d166 = snap901
		d167 = snap902
		d168 = snap903
		d169 = snap904
		d170 = snap905
		d171 = snap906
		d172 = snap907
		d173 = snap908
		d174 = snap909
		d175 = snap910
		d176 = snap911
		d177 = snap912
		d178 = snap913
		d179 = snap914
		d180 = snap915
		d181 = snap916
		d184 = snap917
		d335 = snap918
		d336 = snap919
		d337 = snap920
		d338 = snap921
		d340 = snap922
		d341 = snap923
		d342 = snap924
		d343 = snap925
		d344 = snap926
		d345 = snap927
		d346 = snap928
		d347 = snap929
		d349 = snap930
		d351 = snap931
		d352 = snap932
		d353 = snap933
		d444 = snap934
		d445 = snap935
		d448 = snap936
		d542 = snap937
		d543 = snap938
		d544 = snap939
		d545 = snap940
		d546 = snap941
		d548 = snap942
		d549 = snap943
		d550 = snap944
		d551 = snap945
		d552 = snap946
		d553 = snap947
		d554 = snap948
		d555 = snap949
		d556 = snap950
		d557 = snap951
		d558 = snap952
		d559 = snap953
		d560 = snap954
		d561 = snap955
		d562 = snap956
		d563 = snap957
		d564 = snap958
		d565 = snap959
		d566 = snap960
		d567 = snap961
		d568 = snap962
		d569 = snap963
		d570 = snap964
		d571 = snap965
		d572 = snap966
		d573 = snap967
		d574 = snap968
		d829 = snap969
		d830 = snap970
		d831 = snap971
		d833 = snap972
		d834 = snap973
		d835 = snap974
		d836 = snap975
		d837 = snap976
		d838 = snap977
		d839 = snap978
		d841 = snap979
		d843 = snap980
		d844 = snap981
		ps985 := scm.PhiState{General: true}
		ps985.OverlayValues = make([]scm.JITValueDesc, 985)
		ps985.OverlayValues[5] = d5
		ps985.OverlayValues[6] = d6
		ps985.OverlayValues[7] = d7
		ps985.OverlayValues[8] = d8
		ps985.OverlayValues[9] = d9
		ps985.OverlayValues[10] = d10
		ps985.OverlayValues[11] = d11
		ps985.OverlayValues[12] = d12
		ps985.OverlayValues[13] = d13
		ps985.OverlayValues[14] = d14
		ps985.OverlayValues[15] = d15
		ps985.OverlayValues[16] = d16
		ps985.OverlayValues[17] = d17
		ps985.OverlayValues[18] = d18
		ps985.OverlayValues[19] = d19
		ps985.OverlayValues[21] = d21
		ps985.OverlayValues[22] = d22
		ps985.OverlayValues[23] = d23
		ps985.OverlayValues[24] = d24
		ps985.OverlayValues[25] = d25
		ps985.OverlayValues[26] = d26
		ps985.OverlayValues[27] = d27
		ps985.OverlayValues[28] = d28
		ps985.OverlayValues[29] = d29
		ps985.OverlayValues[30] = d30
		ps985.OverlayValues[31] = d31
		ps985.OverlayValues[32] = d32
		ps985.OverlayValues[33] = d33
		ps985.OverlayValues[34] = d34
		ps985.OverlayValues[35] = d35
		ps985.OverlayValues[36] = d36
		ps985.OverlayValues[37] = d37
		ps985.OverlayValues[38] = d38
		ps985.OverlayValues[39] = d39
		ps985.OverlayValues[40] = d40
		ps985.OverlayValues[41] = d41
		ps985.OverlayValues[42] = d42
		ps985.OverlayValues[43] = d43
		ps985.OverlayValues[44] = d44
		ps985.OverlayValues[45] = d45
		ps985.OverlayValues[46] = d46
		ps985.OverlayValues[47] = d47
		ps985.OverlayValues[48] = d48
		ps985.OverlayValues[49] = d49
		ps985.OverlayValues[50] = d50
		ps985.OverlayValues[51] = d51
		ps985.OverlayValues[54] = d54
		ps985.OverlayValues[55] = d55
		ps985.OverlayValues[56] = d56
		ps985.OverlayValues[159] = d159
		ps985.OverlayValues[160] = d160
		ps985.OverlayValues[161] = d161
		ps985.OverlayValues[162] = d162
		ps985.OverlayValues[163] = d163
		ps985.OverlayValues[164] = d164
		ps985.OverlayValues[165] = d165
		ps985.OverlayValues[166] = d166
		ps985.OverlayValues[167] = d167
		ps985.OverlayValues[168] = d168
		ps985.OverlayValues[169] = d169
		ps985.OverlayValues[170] = d170
		ps985.OverlayValues[171] = d171
		ps985.OverlayValues[172] = d172
		ps985.OverlayValues[173] = d173
		ps985.OverlayValues[174] = d174
		ps985.OverlayValues[175] = d175
		ps985.OverlayValues[176] = d176
		ps985.OverlayValues[177] = d177
		ps985.OverlayValues[178] = d178
		ps985.OverlayValues[179] = d179
		ps985.OverlayValues[180] = d180
		ps985.OverlayValues[181] = d181
		ps985.OverlayValues[184] = d184
		ps985.OverlayValues[335] = d335
		ps985.OverlayValues[336] = d336
		ps985.OverlayValues[337] = d337
		ps985.OverlayValues[338] = d338
		ps985.OverlayValues[340] = d340
		ps985.OverlayValues[341] = d341
		ps985.OverlayValues[342] = d342
		ps985.OverlayValues[343] = d343
		ps985.OverlayValues[344] = d344
		ps985.OverlayValues[345] = d345
		ps985.OverlayValues[346] = d346
		ps985.OverlayValues[347] = d347
		ps985.OverlayValues[349] = d349
		ps985.OverlayValues[351] = d351
		ps985.OverlayValues[352] = d352
		ps985.OverlayValues[353] = d353
		ps985.OverlayValues[444] = d444
		ps985.OverlayValues[445] = d445
		ps985.OverlayValues[448] = d448
		ps985.OverlayValues[542] = d542
		ps985.OverlayValues[543] = d543
		ps985.OverlayValues[544] = d544
		ps985.OverlayValues[545] = d545
		ps985.OverlayValues[546] = d546
		ps985.OverlayValues[548] = d548
		ps985.OverlayValues[549] = d549
		ps985.OverlayValues[550] = d550
		ps985.OverlayValues[551] = d551
		ps985.OverlayValues[552] = d552
		ps985.OverlayValues[553] = d553
		ps985.OverlayValues[554] = d554
		ps985.OverlayValues[555] = d555
		ps985.OverlayValues[556] = d556
		ps985.OverlayValues[557] = d557
		ps985.OverlayValues[558] = d558
		ps985.OverlayValues[559] = d559
		ps985.OverlayValues[560] = d560
		ps985.OverlayValues[561] = d561
		ps985.OverlayValues[562] = d562
		ps985.OverlayValues[563] = d563
		ps985.OverlayValues[564] = d564
		ps985.OverlayValues[565] = d565
		ps985.OverlayValues[566] = d566
		ps985.OverlayValues[567] = d567
		ps985.OverlayValues[568] = d568
		ps985.OverlayValues[569] = d569
		ps985.OverlayValues[570] = d570
		ps985.OverlayValues[571] = d571
		ps985.OverlayValues[572] = d572
		ps985.OverlayValues[573] = d573
		ps985.OverlayValues[574] = d574
		ps985.OverlayValues[829] = d829
		ps985.OverlayValues[830] = d830
		ps985.OverlayValues[831] = d831
		ps985.OverlayValues[833] = d833
		ps985.OverlayValues[834] = d834
		ps985.OverlayValues[835] = d835
		ps985.OverlayValues[836] = d836
		ps985.OverlayValues[837] = d837
		ps985.OverlayValues[838] = d838
		ps985.OverlayValues[839] = d839
		ps985.OverlayValues[841] = d841
		ps985.OverlayValues[843] = d843
		ps985.OverlayValues[844] = d844
		ps985.OverlayValues[983] = d983
		ps985.OverlayValues[984] = d984
		ps985.PhiValues = make([]scm.JITValueDesc, 1)
		d987 = d12
		ps985.PhiValues[0] = d987
		ps986 := scm.PhiState{General: true}
		ps986.OverlayValues = make([]scm.JITValueDesc, 988)
		ps986.OverlayValues[5] = d5
		ps986.OverlayValues[6] = d6
		ps986.OverlayValues[7] = d7
		ps986.OverlayValues[8] = d8
		ps986.OverlayValues[9] = d9
		ps986.OverlayValues[10] = d10
		ps986.OverlayValues[11] = d11
		ps986.OverlayValues[12] = d12
		ps986.OverlayValues[13] = d13
		ps986.OverlayValues[14] = d14
		ps986.OverlayValues[15] = d15
		ps986.OverlayValues[16] = d16
		ps986.OverlayValues[17] = d17
		ps986.OverlayValues[18] = d18
		ps986.OverlayValues[19] = d19
		ps986.OverlayValues[21] = d21
		ps986.OverlayValues[22] = d22
		ps986.OverlayValues[23] = d23
		ps986.OverlayValues[24] = d24
		ps986.OverlayValues[25] = d25
		ps986.OverlayValues[26] = d26
		ps986.OverlayValues[27] = d27
		ps986.OverlayValues[28] = d28
		ps986.OverlayValues[29] = d29
		ps986.OverlayValues[30] = d30
		ps986.OverlayValues[31] = d31
		ps986.OverlayValues[32] = d32
		ps986.OverlayValues[33] = d33
		ps986.OverlayValues[34] = d34
		ps986.OverlayValues[35] = d35
		ps986.OverlayValues[36] = d36
		ps986.OverlayValues[37] = d37
		ps986.OverlayValues[38] = d38
		ps986.OverlayValues[39] = d39
		ps986.OverlayValues[40] = d40
		ps986.OverlayValues[41] = d41
		ps986.OverlayValues[42] = d42
		ps986.OverlayValues[43] = d43
		ps986.OverlayValues[44] = d44
		ps986.OverlayValues[45] = d45
		ps986.OverlayValues[46] = d46
		ps986.OverlayValues[47] = d47
		ps986.OverlayValues[48] = d48
		ps986.OverlayValues[49] = d49
		ps986.OverlayValues[50] = d50
		ps986.OverlayValues[51] = d51
		ps986.OverlayValues[54] = d54
		ps986.OverlayValues[55] = d55
		ps986.OverlayValues[56] = d56
		ps986.OverlayValues[159] = d159
		ps986.OverlayValues[160] = d160
		ps986.OverlayValues[161] = d161
		ps986.OverlayValues[162] = d162
		ps986.OverlayValues[163] = d163
		ps986.OverlayValues[164] = d164
		ps986.OverlayValues[165] = d165
		ps986.OverlayValues[166] = d166
		ps986.OverlayValues[167] = d167
		ps986.OverlayValues[168] = d168
		ps986.OverlayValues[169] = d169
		ps986.OverlayValues[170] = d170
		ps986.OverlayValues[171] = d171
		ps986.OverlayValues[172] = d172
		ps986.OverlayValues[173] = d173
		ps986.OverlayValues[174] = d174
		ps986.OverlayValues[175] = d175
		ps986.OverlayValues[176] = d176
		ps986.OverlayValues[177] = d177
		ps986.OverlayValues[178] = d178
		ps986.OverlayValues[179] = d179
		ps986.OverlayValues[180] = d180
		ps986.OverlayValues[181] = d181
		ps986.OverlayValues[184] = d184
		ps986.OverlayValues[335] = d335
		ps986.OverlayValues[336] = d336
		ps986.OverlayValues[337] = d337
		ps986.OverlayValues[338] = d338
		ps986.OverlayValues[340] = d340
		ps986.OverlayValues[341] = d341
		ps986.OverlayValues[342] = d342
		ps986.OverlayValues[343] = d343
		ps986.OverlayValues[344] = d344
		ps986.OverlayValues[345] = d345
		ps986.OverlayValues[346] = d346
		ps986.OverlayValues[347] = d347
		ps986.OverlayValues[349] = d349
		ps986.OverlayValues[351] = d351
		ps986.OverlayValues[352] = d352
		ps986.OverlayValues[353] = d353
		ps986.OverlayValues[444] = d444
		ps986.OverlayValues[445] = d445
		ps986.OverlayValues[448] = d448
		ps986.OverlayValues[542] = d542
		ps986.OverlayValues[543] = d543
		ps986.OverlayValues[544] = d544
		ps986.OverlayValues[545] = d545
		ps986.OverlayValues[546] = d546
		ps986.OverlayValues[548] = d548
		ps986.OverlayValues[549] = d549
		ps986.OverlayValues[550] = d550
		ps986.OverlayValues[551] = d551
		ps986.OverlayValues[552] = d552
		ps986.OverlayValues[553] = d553
		ps986.OverlayValues[554] = d554
		ps986.OverlayValues[555] = d555
		ps986.OverlayValues[556] = d556
		ps986.OverlayValues[557] = d557
		ps986.OverlayValues[558] = d558
		ps986.OverlayValues[559] = d559
		ps986.OverlayValues[560] = d560
		ps986.OverlayValues[561] = d561
		ps986.OverlayValues[562] = d562
		ps986.OverlayValues[563] = d563
		ps986.OverlayValues[564] = d564
		ps986.OverlayValues[565] = d565
		ps986.OverlayValues[566] = d566
		ps986.OverlayValues[567] = d567
		ps986.OverlayValues[568] = d568
		ps986.OverlayValues[569] = d569
		ps986.OverlayValues[570] = d570
		ps986.OverlayValues[571] = d571
		ps986.OverlayValues[572] = d572
		ps986.OverlayValues[573] = d573
		ps986.OverlayValues[574] = d574
		ps986.OverlayValues[829] = d829
		ps986.OverlayValues[830] = d830
		ps986.OverlayValues[831] = d831
		ps986.OverlayValues[833] = d833
		ps986.OverlayValues[834] = d834
		ps986.OverlayValues[835] = d835
		ps986.OverlayValues[836] = d836
		ps986.OverlayValues[837] = d837
		ps986.OverlayValues[838] = d838
		ps986.OverlayValues[839] = d839
		ps986.OverlayValues[841] = d841
		ps986.OverlayValues[843] = d843
		ps986.OverlayValues[844] = d844
		ps986.OverlayValues[983] = d983
		ps986.OverlayValues[984] = d984
		ps986.OverlayValues[987] = d987
		snap988 := d5
		snap989 := d6
		snap990 := d7
		snap991 := d8
		snap992 := d9
		snap993 := d10
		snap994 := d11
		snap995 := d12
		snap996 := d13
		snap997 := d14
		snap998 := d15
		snap999 := d16
		snap1000 := d17
		snap1001 := d18
		snap1002 := d19
		snap1003 := d21
		snap1004 := d22
		snap1005 := d23
		snap1006 := d24
		snap1007 := d25
		snap1008 := d26
		snap1009 := d27
		snap1010 := d28
		snap1011 := d29
		snap1012 := d30
		snap1013 := d31
		snap1014 := d32
		snap1015 := d33
		snap1016 := d34
		snap1017 := d35
		snap1018 := d36
		snap1019 := d37
		snap1020 := d38
		snap1021 := d39
		snap1022 := d40
		snap1023 := d41
		snap1024 := d42
		snap1025 := d43
		snap1026 := d44
		snap1027 := d45
		snap1028 := d46
		snap1029 := d47
		snap1030 := d48
		snap1031 := d49
		snap1032 := d50
		snap1033 := d51
		snap1034 := d54
		snap1035 := d55
		snap1036 := d56
		snap1037 := d159
		snap1038 := d160
		snap1039 := d161
		snap1040 := d162
		snap1041 := d163
		snap1042 := d164
		snap1043 := d165
		snap1044 := d166
		snap1045 := d167
		snap1046 := d168
		snap1047 := d169
		snap1048 := d170
		snap1049 := d171
		snap1050 := d172
		snap1051 := d173
		snap1052 := d174
		snap1053 := d175
		snap1054 := d176
		snap1055 := d177
		snap1056 := d178
		snap1057 := d179
		snap1058 := d180
		snap1059 := d181
		snap1060 := d184
		snap1061 := d335
		snap1062 := d336
		snap1063 := d337
		snap1064 := d338
		snap1065 := d340
		snap1066 := d341
		snap1067 := d342
		snap1068 := d343
		snap1069 := d344
		snap1070 := d345
		snap1071 := d346
		snap1072 := d347
		snap1073 := d349
		snap1074 := d351
		snap1075 := d352
		snap1076 := d353
		snap1077 := d444
		snap1078 := d445
		snap1079 := d448
		snap1080 := d542
		snap1081 := d543
		snap1082 := d544
		snap1083 := d545
		snap1084 := d546
		snap1085 := d548
		snap1086 := d549
		snap1087 := d550
		snap1088 := d551
		snap1089 := d552
		snap1090 := d553
		snap1091 := d554
		snap1092 := d555
		snap1093 := d556
		snap1094 := d557
		snap1095 := d558
		snap1096 := d559
		snap1097 := d560
		snap1098 := d561
		snap1099 := d562
		snap1100 := d563
		snap1101 := d564
		snap1102 := d565
		snap1103 := d566
		snap1104 := d567
		snap1105 := d568
		snap1106 := d569
		snap1107 := d570
		snap1108 := d571
		snap1109 := d572
		snap1110 := d573
		snap1111 := d574
		snap1112 := d829
		snap1113 := d830
		snap1114 := d831
		snap1115 := d833
		snap1116 := d834
		snap1117 := d835
		snap1118 := d836
		snap1119 := d837
		snap1120 := d838
		snap1121 := d839
		snap1122 := d841
		snap1123 := d843
		snap1124 := d844
		snap1125 := d983
		snap1126 := d984
		snap1127 := d987
		alloc1128 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps985)
		}
		ctx.RestoreAllocState(alloc1128)
		d5 = snap988
		d6 = snap989
		d7 = snap990
		d8 = snap991
		d9 = snap992
		d10 = snap993
		d11 = snap994
		d12 = snap995
		d13 = snap996
		d14 = snap997
		d15 = snap998
		d16 = snap999
		d17 = snap1000
		d18 = snap1001
		d19 = snap1002
		d21 = snap1003
		d22 = snap1004
		d23 = snap1005
		d24 = snap1006
		d25 = snap1007
		d26 = snap1008
		d27 = snap1009
		d28 = snap1010
		d29 = snap1011
		d30 = snap1012
		d31 = snap1013
		d32 = snap1014
		d33 = snap1015
		d34 = snap1016
		d35 = snap1017
		d36 = snap1018
		d37 = snap1019
		d38 = snap1020
		d39 = snap1021
		d40 = snap1022
		d41 = snap1023
		d42 = snap1024
		d43 = snap1025
		d44 = snap1026
		d45 = snap1027
		d46 = snap1028
		d47 = snap1029
		d48 = snap1030
		d49 = snap1031
		d50 = snap1032
		d51 = snap1033
		d54 = snap1034
		d55 = snap1035
		d56 = snap1036
		d159 = snap1037
		d160 = snap1038
		d161 = snap1039
		d162 = snap1040
		d163 = snap1041
		d164 = snap1042
		d165 = snap1043
		d166 = snap1044
		d167 = snap1045
		d168 = snap1046
		d169 = snap1047
		d170 = snap1048
		d171 = snap1049
		d172 = snap1050
		d173 = snap1051
		d174 = snap1052
		d175 = snap1053
		d176 = snap1054
		d177 = snap1055
		d178 = snap1056
		d179 = snap1057
		d180 = snap1058
		d181 = snap1059
		d184 = snap1060
		d335 = snap1061
		d336 = snap1062
		d337 = snap1063
		d338 = snap1064
		d340 = snap1065
		d341 = snap1066
		d342 = snap1067
		d343 = snap1068
		d344 = snap1069
		d345 = snap1070
		d346 = snap1071
		d347 = snap1072
		d349 = snap1073
		d351 = snap1074
		d352 = snap1075
		d353 = snap1076
		d444 = snap1077
		d445 = snap1078
		d448 = snap1079
		d542 = snap1080
		d543 = snap1081
		d544 = snap1082
		d545 = snap1083
		d546 = snap1084
		d548 = snap1085
		d549 = snap1086
		d550 = snap1087
		d551 = snap1088
		d552 = snap1089
		d553 = snap1090
		d554 = snap1091
		d555 = snap1092
		d556 = snap1093
		d557 = snap1094
		d558 = snap1095
		d559 = snap1096
		d560 = snap1097
		d561 = snap1098
		d562 = snap1099
		d563 = snap1100
		d564 = snap1101
		d565 = snap1102
		d566 = snap1103
		d567 = snap1104
		d568 = snap1105
		d569 = snap1106
		d570 = snap1107
		d571 = snap1108
		d572 = snap1109
		d573 = snap1110
		d574 = snap1111
		d829 = snap1112
		d830 = snap1113
		d831 = snap1114
		d833 = snap1115
		d834 = snap1116
		d835 = snap1117
		d836 = snap1118
		d837 = snap1119
		d838 = snap1120
		d839 = snap1121
		d841 = snap1122
		d843 = snap1123
		d844 = snap1124
		d983 = snap1125
		d984 = snap1126
		d987 = snap1127
		if !bbs[10].Rendered {
			return bbs[10].RenderPS(ps986)
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
			ctx.FlushRegisterMoves()
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
		if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
			d184 = ps.OverlayValues[184]
		}
		if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != scm.LocNone {
			d335 = ps.OverlayValues[335]
		}
		if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != scm.LocNone {
			d336 = ps.OverlayValues[336]
		}
		if len(ps.OverlayValues) > 337 && ps.OverlayValues[337].Loc != scm.LocNone {
			d337 = ps.OverlayValues[337]
		}
		if len(ps.OverlayValues) > 338 && ps.OverlayValues[338].Loc != scm.LocNone {
			d338 = ps.OverlayValues[338]
		}
		if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != scm.LocNone {
			d340 = ps.OverlayValues[340]
		}
		if len(ps.OverlayValues) > 341 && ps.OverlayValues[341].Loc != scm.LocNone {
			d341 = ps.OverlayValues[341]
		}
		if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != scm.LocNone {
			d342 = ps.OverlayValues[342]
		}
		if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != scm.LocNone {
			d343 = ps.OverlayValues[343]
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
		if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != scm.LocNone {
			d351 = ps.OverlayValues[351]
		}
		if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != scm.LocNone {
			d352 = ps.OverlayValues[352]
		}
		if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != scm.LocNone {
			d353 = ps.OverlayValues[353]
		}
		if len(ps.OverlayValues) > 444 && ps.OverlayValues[444].Loc != scm.LocNone {
			d444 = ps.OverlayValues[444]
		}
		if len(ps.OverlayValues) > 445 && ps.OverlayValues[445].Loc != scm.LocNone {
			d445 = ps.OverlayValues[445]
		}
		if len(ps.OverlayValues) > 448 && ps.OverlayValues[448].Loc != scm.LocNone {
			d448 = ps.OverlayValues[448]
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
		if len(ps.OverlayValues) > 548 && ps.OverlayValues[548].Loc != scm.LocNone {
			d548 = ps.OverlayValues[548]
		}
		if len(ps.OverlayValues) > 549 && ps.OverlayValues[549].Loc != scm.LocNone {
			d549 = ps.OverlayValues[549]
		}
		if len(ps.OverlayValues) > 550 && ps.OverlayValues[550].Loc != scm.LocNone {
			d550 = ps.OverlayValues[550]
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
		if len(ps.OverlayValues) > 555 && ps.OverlayValues[555].Loc != scm.LocNone {
			d555 = ps.OverlayValues[555]
		}
		if len(ps.OverlayValues) > 556 && ps.OverlayValues[556].Loc != scm.LocNone {
			d556 = ps.OverlayValues[556]
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
		if len(ps.OverlayValues) > 562 && ps.OverlayValues[562].Loc != scm.LocNone {
			d562 = ps.OverlayValues[562]
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
		if len(ps.OverlayValues) > 829 && ps.OverlayValues[829].Loc != scm.LocNone {
			d829 = ps.OverlayValues[829]
		}
		if len(ps.OverlayValues) > 830 && ps.OverlayValues[830].Loc != scm.LocNone {
			d830 = ps.OverlayValues[830]
		}
		if len(ps.OverlayValues) > 831 && ps.OverlayValues[831].Loc != scm.LocNone {
			d831 = ps.OverlayValues[831]
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
		if len(ps.OverlayValues) > 838 && ps.OverlayValues[838].Loc != scm.LocNone {
			d838 = ps.OverlayValues[838]
		}
		if len(ps.OverlayValues) > 839 && ps.OverlayValues[839].Loc != scm.LocNone {
			d839 = ps.OverlayValues[839]
		}
		if len(ps.OverlayValues) > 841 && ps.OverlayValues[841].Loc != scm.LocNone {
			d841 = ps.OverlayValues[841]
		}
		if len(ps.OverlayValues) > 843 && ps.OverlayValues[843].Loc != scm.LocNone {
			d843 = ps.OverlayValues[843]
		}
		if len(ps.OverlayValues) > 844 && ps.OverlayValues[844].Loc != scm.LocNone {
			d844 = ps.OverlayValues[844]
		}
		if len(ps.OverlayValues) > 983 && ps.OverlayValues[983].Loc != scm.LocNone {
			d983 = ps.OverlayValues[983]
		}
		if len(ps.OverlayValues) > 984 && ps.OverlayValues[984].Loc != scm.LocNone {
			d984 = ps.OverlayValues[984]
		}
		if len(ps.OverlayValues) > 987 && ps.OverlayValues[987].Loc != scm.LocNone {
			d987 = ps.OverlayValues[987]
		}
		ctx.ReclaimUntrackedRegs()
		if ps.General {
			ctx.SyncDesc(&d9)
			if d9.Loc == scm.LocReg || d9.Loc == scm.LocFPReg {
				ctx.ProtectReg(d9.Reg)
			} else if d9.Loc == scm.LocRegPair {
				ctx.ProtectReg(d9.Reg)
				ctx.ProtectReg(d9.Reg2)
			}
			ctx.SyncDesc(&d11)
			if d11.Loc == scm.LocReg || d11.Loc == scm.LocFPReg {
				ctx.ProtectReg(d11.Reg)
			} else if d11.Loc == scm.LocRegPair {
				ctx.ProtectReg(d11.Reg)
				ctx.ProtectReg(d11.Reg2)
			}
			d1129 = d9
			if d1129.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d1129)
			d1130 = d1129
			if d1130.Loc == scm.LocImm {
				d1130 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1130.Type, Imm: scm.NewInt(int64(uint64(d1130.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d1130.Reg, 32)
				ctx.EmitShrRegImm8(d1130.Reg, 32)
			}
			ctx.EmitStoreToStack(d1130, int32(bbs[8].PhiBase)+int32(0))
			d1131 = d11
			if d1131.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d1131)
			d1132 = d1131
			if d1132.Loc == scm.LocImm {
				d1132 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1132.Type, Imm: scm.NewInt(int64(uint64(d1132.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d1132.Reg, 32)
				ctx.EmitShrRegImm8(d1132.Reg, 32)
			}
			ctx.EmitStoreToStack(d1132, int32(bbs[8].PhiBase)+int32(16))
			if d9.Loc == scm.LocReg || d9.Loc == scm.LocFPReg {
				ctx.UnprotectReg(d9.Reg)
			} else if d9.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d9.Reg)
				ctx.UnprotectReg(d9.Reg2)
			}
			if d11.Loc == scm.LocReg || d11.Loc == scm.LocFPReg {
				ctx.UnprotectReg(d11.Reg)
			} else if d11.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d11.Reg)
				ctx.UnprotectReg(d11.Reg2)
			}
		}
		ps1133 := scm.PhiState{General: ps.General}
		ps1133.OverlayValues = make([]scm.JITValueDesc, 1133)
		ps1133.OverlayValues[5] = d5
		ps1133.OverlayValues[6] = d6
		ps1133.OverlayValues[7] = d7
		ps1133.OverlayValues[8] = d8
		ps1133.OverlayValues[9] = d9
		ps1133.OverlayValues[10] = d10
		ps1133.OverlayValues[11] = d11
		ps1133.OverlayValues[12] = d12
		ps1133.OverlayValues[13] = d13
		ps1133.OverlayValues[14] = d14
		ps1133.OverlayValues[15] = d15
		ps1133.OverlayValues[16] = d16
		ps1133.OverlayValues[17] = d17
		ps1133.OverlayValues[18] = d18
		ps1133.OverlayValues[19] = d19
		ps1133.OverlayValues[21] = d21
		ps1133.OverlayValues[22] = d22
		ps1133.OverlayValues[23] = d23
		ps1133.OverlayValues[24] = d24
		ps1133.OverlayValues[25] = d25
		ps1133.OverlayValues[26] = d26
		ps1133.OverlayValues[27] = d27
		ps1133.OverlayValues[28] = d28
		ps1133.OverlayValues[29] = d29
		ps1133.OverlayValues[30] = d30
		ps1133.OverlayValues[31] = d31
		ps1133.OverlayValues[32] = d32
		ps1133.OverlayValues[33] = d33
		ps1133.OverlayValues[34] = d34
		ps1133.OverlayValues[35] = d35
		ps1133.OverlayValues[36] = d36
		ps1133.OverlayValues[37] = d37
		ps1133.OverlayValues[38] = d38
		ps1133.OverlayValues[39] = d39
		ps1133.OverlayValues[40] = d40
		ps1133.OverlayValues[41] = d41
		ps1133.OverlayValues[42] = d42
		ps1133.OverlayValues[43] = d43
		ps1133.OverlayValues[44] = d44
		ps1133.OverlayValues[45] = d45
		ps1133.OverlayValues[46] = d46
		ps1133.OverlayValues[47] = d47
		ps1133.OverlayValues[48] = d48
		ps1133.OverlayValues[49] = d49
		ps1133.OverlayValues[50] = d50
		ps1133.OverlayValues[51] = d51
		ps1133.OverlayValues[54] = d54
		ps1133.OverlayValues[55] = d55
		ps1133.OverlayValues[56] = d56
		ps1133.OverlayValues[159] = d159
		ps1133.OverlayValues[160] = d160
		ps1133.OverlayValues[161] = d161
		ps1133.OverlayValues[162] = d162
		ps1133.OverlayValues[163] = d163
		ps1133.OverlayValues[164] = d164
		ps1133.OverlayValues[165] = d165
		ps1133.OverlayValues[166] = d166
		ps1133.OverlayValues[167] = d167
		ps1133.OverlayValues[168] = d168
		ps1133.OverlayValues[169] = d169
		ps1133.OverlayValues[170] = d170
		ps1133.OverlayValues[171] = d171
		ps1133.OverlayValues[172] = d172
		ps1133.OverlayValues[173] = d173
		ps1133.OverlayValues[174] = d174
		ps1133.OverlayValues[175] = d175
		ps1133.OverlayValues[176] = d176
		ps1133.OverlayValues[177] = d177
		ps1133.OverlayValues[178] = d178
		ps1133.OverlayValues[179] = d179
		ps1133.OverlayValues[180] = d180
		ps1133.OverlayValues[181] = d181
		ps1133.OverlayValues[184] = d184
		ps1133.OverlayValues[335] = d335
		ps1133.OverlayValues[336] = d336
		ps1133.OverlayValues[337] = d337
		ps1133.OverlayValues[338] = d338
		ps1133.OverlayValues[340] = d340
		ps1133.OverlayValues[341] = d341
		ps1133.OverlayValues[342] = d342
		ps1133.OverlayValues[343] = d343
		ps1133.OverlayValues[344] = d344
		ps1133.OverlayValues[345] = d345
		ps1133.OverlayValues[346] = d346
		ps1133.OverlayValues[347] = d347
		ps1133.OverlayValues[349] = d349
		ps1133.OverlayValues[351] = d351
		ps1133.OverlayValues[352] = d352
		ps1133.OverlayValues[353] = d353
		ps1133.OverlayValues[444] = d444
		ps1133.OverlayValues[445] = d445
		ps1133.OverlayValues[448] = d448
		ps1133.OverlayValues[542] = d542
		ps1133.OverlayValues[543] = d543
		ps1133.OverlayValues[544] = d544
		ps1133.OverlayValues[545] = d545
		ps1133.OverlayValues[546] = d546
		ps1133.OverlayValues[548] = d548
		ps1133.OverlayValues[549] = d549
		ps1133.OverlayValues[550] = d550
		ps1133.OverlayValues[551] = d551
		ps1133.OverlayValues[552] = d552
		ps1133.OverlayValues[553] = d553
		ps1133.OverlayValues[554] = d554
		ps1133.OverlayValues[555] = d555
		ps1133.OverlayValues[556] = d556
		ps1133.OverlayValues[557] = d557
		ps1133.OverlayValues[558] = d558
		ps1133.OverlayValues[559] = d559
		ps1133.OverlayValues[560] = d560
		ps1133.OverlayValues[561] = d561
		ps1133.OverlayValues[562] = d562
		ps1133.OverlayValues[563] = d563
		ps1133.OverlayValues[564] = d564
		ps1133.OverlayValues[565] = d565
		ps1133.OverlayValues[566] = d566
		ps1133.OverlayValues[567] = d567
		ps1133.OverlayValues[568] = d568
		ps1133.OverlayValues[569] = d569
		ps1133.OverlayValues[570] = d570
		ps1133.OverlayValues[571] = d571
		ps1133.OverlayValues[572] = d572
		ps1133.OverlayValues[573] = d573
		ps1133.OverlayValues[574] = d574
		ps1133.OverlayValues[829] = d829
		ps1133.OverlayValues[830] = d830
		ps1133.OverlayValues[831] = d831
		ps1133.OverlayValues[833] = d833
		ps1133.OverlayValues[834] = d834
		ps1133.OverlayValues[835] = d835
		ps1133.OverlayValues[836] = d836
		ps1133.OverlayValues[837] = d837
		ps1133.OverlayValues[838] = d838
		ps1133.OverlayValues[839] = d839
		ps1133.OverlayValues[841] = d841
		ps1133.OverlayValues[843] = d843
		ps1133.OverlayValues[844] = d844
		ps1133.OverlayValues[983] = d983
		ps1133.OverlayValues[984] = d984
		ps1133.OverlayValues[987] = d987
		ps1133.OverlayValues[1129] = d1129
		ps1133.OverlayValues[1130] = d1130
		ps1133.OverlayValues[1131] = d1131
		ps1133.OverlayValues[1132] = d1132
		ps1133.PhiValues = make([]scm.JITValueDesc, 2)
		d1134 = d9
		ps1133.PhiValues[0] = d1134
		d1135 = d11
		ps1133.PhiValues[1] = d1135
		if ps1133.General && bbs[8].Rendered {
			ctx.EmitJmp(lbl9)
			return result
		}
		return bbs[8].RenderPS(ps1133)
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
			ctx.FlushRegisterMoves()
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
		if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
			d184 = ps.OverlayValues[184]
		}
		if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != scm.LocNone {
			d335 = ps.OverlayValues[335]
		}
		if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != scm.LocNone {
			d336 = ps.OverlayValues[336]
		}
		if len(ps.OverlayValues) > 337 && ps.OverlayValues[337].Loc != scm.LocNone {
			d337 = ps.OverlayValues[337]
		}
		if len(ps.OverlayValues) > 338 && ps.OverlayValues[338].Loc != scm.LocNone {
			d338 = ps.OverlayValues[338]
		}
		if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != scm.LocNone {
			d340 = ps.OverlayValues[340]
		}
		if len(ps.OverlayValues) > 341 && ps.OverlayValues[341].Loc != scm.LocNone {
			d341 = ps.OverlayValues[341]
		}
		if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != scm.LocNone {
			d342 = ps.OverlayValues[342]
		}
		if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != scm.LocNone {
			d343 = ps.OverlayValues[343]
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
		if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != scm.LocNone {
			d351 = ps.OverlayValues[351]
		}
		if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != scm.LocNone {
			d352 = ps.OverlayValues[352]
		}
		if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != scm.LocNone {
			d353 = ps.OverlayValues[353]
		}
		if len(ps.OverlayValues) > 444 && ps.OverlayValues[444].Loc != scm.LocNone {
			d444 = ps.OverlayValues[444]
		}
		if len(ps.OverlayValues) > 445 && ps.OverlayValues[445].Loc != scm.LocNone {
			d445 = ps.OverlayValues[445]
		}
		if len(ps.OverlayValues) > 448 && ps.OverlayValues[448].Loc != scm.LocNone {
			d448 = ps.OverlayValues[448]
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
		if len(ps.OverlayValues) > 548 && ps.OverlayValues[548].Loc != scm.LocNone {
			d548 = ps.OverlayValues[548]
		}
		if len(ps.OverlayValues) > 549 && ps.OverlayValues[549].Loc != scm.LocNone {
			d549 = ps.OverlayValues[549]
		}
		if len(ps.OverlayValues) > 550 && ps.OverlayValues[550].Loc != scm.LocNone {
			d550 = ps.OverlayValues[550]
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
		if len(ps.OverlayValues) > 555 && ps.OverlayValues[555].Loc != scm.LocNone {
			d555 = ps.OverlayValues[555]
		}
		if len(ps.OverlayValues) > 556 && ps.OverlayValues[556].Loc != scm.LocNone {
			d556 = ps.OverlayValues[556]
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
		if len(ps.OverlayValues) > 562 && ps.OverlayValues[562].Loc != scm.LocNone {
			d562 = ps.OverlayValues[562]
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
		if len(ps.OverlayValues) > 829 && ps.OverlayValues[829].Loc != scm.LocNone {
			d829 = ps.OverlayValues[829]
		}
		if len(ps.OverlayValues) > 830 && ps.OverlayValues[830].Loc != scm.LocNone {
			d830 = ps.OverlayValues[830]
		}
		if len(ps.OverlayValues) > 831 && ps.OverlayValues[831].Loc != scm.LocNone {
			d831 = ps.OverlayValues[831]
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
		if len(ps.OverlayValues) > 838 && ps.OverlayValues[838].Loc != scm.LocNone {
			d838 = ps.OverlayValues[838]
		}
		if len(ps.OverlayValues) > 839 && ps.OverlayValues[839].Loc != scm.LocNone {
			d839 = ps.OverlayValues[839]
		}
		if len(ps.OverlayValues) > 841 && ps.OverlayValues[841].Loc != scm.LocNone {
			d841 = ps.OverlayValues[841]
		}
		if len(ps.OverlayValues) > 843 && ps.OverlayValues[843].Loc != scm.LocNone {
			d843 = ps.OverlayValues[843]
		}
		if len(ps.OverlayValues) > 844 && ps.OverlayValues[844].Loc != scm.LocNone {
			d844 = ps.OverlayValues[844]
		}
		if len(ps.OverlayValues) > 983 && ps.OverlayValues[983].Loc != scm.LocNone {
			d983 = ps.OverlayValues[983]
		}
		if len(ps.OverlayValues) > 984 && ps.OverlayValues[984].Loc != scm.LocNone {
			d984 = ps.OverlayValues[984]
		}
		if len(ps.OverlayValues) > 987 && ps.OverlayValues[987].Loc != scm.LocNone {
			d987 = ps.OverlayValues[987]
		}
		if len(ps.OverlayValues) > 1129 && ps.OverlayValues[1129].Loc != scm.LocNone {
			d1129 = ps.OverlayValues[1129]
		}
		if len(ps.OverlayValues) > 1130 && ps.OverlayValues[1130].Loc != scm.LocNone {
			d1130 = ps.OverlayValues[1130]
		}
		if len(ps.OverlayValues) > 1131 && ps.OverlayValues[1131].Loc != scm.LocNone {
			d1131 = ps.OverlayValues[1131]
		}
		if len(ps.OverlayValues) > 1132 && ps.OverlayValues[1132].Loc != scm.LocNone {
			d1132 = ps.OverlayValues[1132]
		}
		if len(ps.OverlayValues) > 1134 && ps.OverlayValues[1134].Loc != scm.LocNone {
			d1134 = ps.OverlayValues[1134]
		}
		if len(ps.OverlayValues) > 1135 && ps.OverlayValues[1135].Loc != scm.LocNone {
			d1135 = ps.OverlayValues[1135]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d12)
		ctx.EnsureDesc(&d13)
		ctx.EnsureDescsTogether(&d12, &d13)
		var d1136 scm.JITValueDesc
		if d12.Loc == scm.LocImm && d13.Loc == scm.LocImm {
			d1136 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() + d13.Imm.Int())}
		} else if d13.Loc == scm.LocImm && d13.Imm.Int() == 0 {
			r96 := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitMovRegReg(r96, d12.Reg)
			d1136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
			ctx.BindReg(r96, &d1136)
		} else if d12.Loc == scm.LocImm && d12.Imm.Int() == 0 {
			d1136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d13.Reg}
			ctx.BindReg(d13.Reg, &d1136)
		} else if d12.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d13.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d12.Imm.Int()))
			ctx.EmitAddInt32(scratch, d13.Reg)
			d1136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1136)
		} else if d13.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitMovRegReg(scratch, d12.Reg)
			if d13.Imm.Int() >= -2147483648 && d13.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32Low(scratch, int32(d13.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
				ctx.EmitAddInt32(scratch, scm.RegR11)
			}
			d1136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1136)
		} else {
			r97 := ctx.AllocRegExcept(d12.Reg, d13.Reg)
			ctx.EmitMovRegReg(r97, d12.Reg)
			ctx.EmitAddInt32(r97, d13.Reg)
			d1136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
			ctx.BindReg(r97, &d1136)
		}
		if d1136.Loc == scm.LocReg && d12.Loc == scm.LocReg && d1136.Reg == d12.Reg {
			ctx.TransferReg(d12.Reg)
			d12.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d1136)
		var d1137 scm.JITValueDesc
		if d1136.Loc == scm.LocImm {
			d1137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1136.Imm.Int() / 2)}
		} else {
			r98 := ctx.AllocRegExcept(d1136.Reg)
			ctx.EmitMovRegReg(r98, d1136.Reg)
			ctx.EmitShrRegImm8(r98, 1)
			d1137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r98}
			ctx.BindReg(r98, &d1137)
		}
		if d1137.Loc == scm.LocImm {
			d1137 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1137.Type, Imm: scm.NewInt(int64(uint64(d1137.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d1137.Reg, 32)
			ctx.EmitShrRegImm8(d1137.Reg, 32)
		}
		if d1137.Loc == scm.LocReg && d1136.Loc == scm.LocReg && d1137.Reg == d1136.Reg {
			ctx.TransferReg(d1136.Reg)
			d1136.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1136)
		if ps.General {
			ctx.SyncDesc(&d12)
			if d12.Loc == scm.LocReg || d12.Loc == scm.LocFPReg {
				ctx.ProtectReg(d12.Reg)
			} else if d12.Loc == scm.LocRegPair {
				ctx.ProtectReg(d12.Reg)
				ctx.ProtectReg(d12.Reg2)
			}
			ctx.SyncDesc(&d13)
			if d13.Loc == scm.LocReg || d13.Loc == scm.LocFPReg {
				ctx.ProtectReg(d13.Reg)
			} else if d13.Loc == scm.LocRegPair {
				ctx.ProtectReg(d13.Reg)
				ctx.ProtectReg(d13.Reg2)
			}
			ctx.SyncDesc(&d1137)
			if d1137.Loc == scm.LocReg || d1137.Loc == scm.LocFPReg {
				ctx.ProtectReg(d1137.Reg)
			} else if d1137.Loc == scm.LocRegPair {
				ctx.ProtectReg(d1137.Reg)
				ctx.ProtectReg(d1137.Reg2)
			}
			d1138 = d1137
			if d1138.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d1138)
			d1139 = d1138
			if d1139.Loc == scm.LocImm {
				d1139 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1139.Type, Imm: scm.NewInt(int64(uint64(d1139.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d1139.Reg, 32)
				ctx.EmitShrRegImm8(d1139.Reg, 32)
			}
			if phiHomeOK2 {
				ctx.EmitMovToReg(r0, d1139)
			} else {
				ctx.EmitStoreToStack(d1139, int32(bbs[1].PhiBase)+int32(0))
			}
			d1140 = d12
			if d1140.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d1140)
			d1141 = d1140
			if d1141.Loc == scm.LocImm {
				d1141 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1141.Type, Imm: scm.NewInt(int64(uint64(d1141.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d1141.Reg, 32)
				ctx.EmitShrRegImm8(d1141.Reg, 32)
			}
			if phiHomeOK3 {
				ctx.EmitMovToReg(r1, d1141)
			} else {
				ctx.EmitStoreToStack(d1141, int32(bbs[1].PhiBase)+int32(16))
			}
			d1142 = d13
			if d1142.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d1142)
			d1143 = d1142
			if d1143.Loc == scm.LocImm {
				d1143 = scm.JITValueDesc{Loc: scm.LocImm, Type: d1143.Type, Imm: scm.NewInt(int64(uint64(d1143.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d1143.Reg, 32)
				ctx.EmitShrRegImm8(d1143.Reg, 32)
			}
			if phiHomeOK4 {
				ctx.EmitMovToReg(r2, d1143)
			} else {
				ctx.EmitStoreToStack(d1143, int32(bbs[1].PhiBase)+int32(32))
			}
			if d12.Loc == scm.LocReg || d12.Loc == scm.LocFPReg {
				ctx.UnprotectReg(d12.Reg)
			} else if d12.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d12.Reg)
				ctx.UnprotectReg(d12.Reg2)
			}
			if d13.Loc == scm.LocReg || d13.Loc == scm.LocFPReg {
				ctx.UnprotectReg(d13.Reg)
			} else if d13.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d13.Reg)
				ctx.UnprotectReg(d13.Reg2)
			}
			if d1137.Loc == scm.LocReg || d1137.Loc == scm.LocFPReg {
				ctx.UnprotectReg(d1137.Reg)
			} else if d1137.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d1137.Reg)
				ctx.UnprotectReg(d1137.Reg2)
			}
		}
		ps1144 := scm.PhiState{General: ps.General}
		ps1144.OverlayValues = make([]scm.JITValueDesc, 1144)
		ps1144.OverlayValues[5] = d5
		ps1144.OverlayValues[6] = d6
		ps1144.OverlayValues[7] = d7
		ps1144.OverlayValues[8] = d8
		ps1144.OverlayValues[9] = d9
		ps1144.OverlayValues[10] = d10
		ps1144.OverlayValues[11] = d11
		ps1144.OverlayValues[12] = d12
		ps1144.OverlayValues[13] = d13
		ps1144.OverlayValues[14] = d14
		ps1144.OverlayValues[15] = d15
		ps1144.OverlayValues[16] = d16
		ps1144.OverlayValues[17] = d17
		ps1144.OverlayValues[18] = d18
		ps1144.OverlayValues[19] = d19
		ps1144.OverlayValues[21] = d21
		ps1144.OverlayValues[22] = d22
		ps1144.OverlayValues[23] = d23
		ps1144.OverlayValues[24] = d24
		ps1144.OverlayValues[25] = d25
		ps1144.OverlayValues[26] = d26
		ps1144.OverlayValues[27] = d27
		ps1144.OverlayValues[28] = d28
		ps1144.OverlayValues[29] = d29
		ps1144.OverlayValues[30] = d30
		ps1144.OverlayValues[31] = d31
		ps1144.OverlayValues[32] = d32
		ps1144.OverlayValues[33] = d33
		ps1144.OverlayValues[34] = d34
		ps1144.OverlayValues[35] = d35
		ps1144.OverlayValues[36] = d36
		ps1144.OverlayValues[37] = d37
		ps1144.OverlayValues[38] = d38
		ps1144.OverlayValues[39] = d39
		ps1144.OverlayValues[40] = d40
		ps1144.OverlayValues[41] = d41
		ps1144.OverlayValues[42] = d42
		ps1144.OverlayValues[43] = d43
		ps1144.OverlayValues[44] = d44
		ps1144.OverlayValues[45] = d45
		ps1144.OverlayValues[46] = d46
		ps1144.OverlayValues[47] = d47
		ps1144.OverlayValues[48] = d48
		ps1144.OverlayValues[49] = d49
		ps1144.OverlayValues[50] = d50
		ps1144.OverlayValues[51] = d51
		ps1144.OverlayValues[54] = d54
		ps1144.OverlayValues[55] = d55
		ps1144.OverlayValues[56] = d56
		ps1144.OverlayValues[159] = d159
		ps1144.OverlayValues[160] = d160
		ps1144.OverlayValues[161] = d161
		ps1144.OverlayValues[162] = d162
		ps1144.OverlayValues[163] = d163
		ps1144.OverlayValues[164] = d164
		ps1144.OverlayValues[165] = d165
		ps1144.OverlayValues[166] = d166
		ps1144.OverlayValues[167] = d167
		ps1144.OverlayValues[168] = d168
		ps1144.OverlayValues[169] = d169
		ps1144.OverlayValues[170] = d170
		ps1144.OverlayValues[171] = d171
		ps1144.OverlayValues[172] = d172
		ps1144.OverlayValues[173] = d173
		ps1144.OverlayValues[174] = d174
		ps1144.OverlayValues[175] = d175
		ps1144.OverlayValues[176] = d176
		ps1144.OverlayValues[177] = d177
		ps1144.OverlayValues[178] = d178
		ps1144.OverlayValues[179] = d179
		ps1144.OverlayValues[180] = d180
		ps1144.OverlayValues[181] = d181
		ps1144.OverlayValues[184] = d184
		ps1144.OverlayValues[335] = d335
		ps1144.OverlayValues[336] = d336
		ps1144.OverlayValues[337] = d337
		ps1144.OverlayValues[338] = d338
		ps1144.OverlayValues[340] = d340
		ps1144.OverlayValues[341] = d341
		ps1144.OverlayValues[342] = d342
		ps1144.OverlayValues[343] = d343
		ps1144.OverlayValues[344] = d344
		ps1144.OverlayValues[345] = d345
		ps1144.OverlayValues[346] = d346
		ps1144.OverlayValues[347] = d347
		ps1144.OverlayValues[349] = d349
		ps1144.OverlayValues[351] = d351
		ps1144.OverlayValues[352] = d352
		ps1144.OverlayValues[353] = d353
		ps1144.OverlayValues[444] = d444
		ps1144.OverlayValues[445] = d445
		ps1144.OverlayValues[448] = d448
		ps1144.OverlayValues[542] = d542
		ps1144.OverlayValues[543] = d543
		ps1144.OverlayValues[544] = d544
		ps1144.OverlayValues[545] = d545
		ps1144.OverlayValues[546] = d546
		ps1144.OverlayValues[548] = d548
		ps1144.OverlayValues[549] = d549
		ps1144.OverlayValues[550] = d550
		ps1144.OverlayValues[551] = d551
		ps1144.OverlayValues[552] = d552
		ps1144.OverlayValues[553] = d553
		ps1144.OverlayValues[554] = d554
		ps1144.OverlayValues[555] = d555
		ps1144.OverlayValues[556] = d556
		ps1144.OverlayValues[557] = d557
		ps1144.OverlayValues[558] = d558
		ps1144.OverlayValues[559] = d559
		ps1144.OverlayValues[560] = d560
		ps1144.OverlayValues[561] = d561
		ps1144.OverlayValues[562] = d562
		ps1144.OverlayValues[563] = d563
		ps1144.OverlayValues[564] = d564
		ps1144.OverlayValues[565] = d565
		ps1144.OverlayValues[566] = d566
		ps1144.OverlayValues[567] = d567
		ps1144.OverlayValues[568] = d568
		ps1144.OverlayValues[569] = d569
		ps1144.OverlayValues[570] = d570
		ps1144.OverlayValues[571] = d571
		ps1144.OverlayValues[572] = d572
		ps1144.OverlayValues[573] = d573
		ps1144.OverlayValues[574] = d574
		ps1144.OverlayValues[829] = d829
		ps1144.OverlayValues[830] = d830
		ps1144.OverlayValues[831] = d831
		ps1144.OverlayValues[833] = d833
		ps1144.OverlayValues[834] = d834
		ps1144.OverlayValues[835] = d835
		ps1144.OverlayValues[836] = d836
		ps1144.OverlayValues[837] = d837
		ps1144.OverlayValues[838] = d838
		ps1144.OverlayValues[839] = d839
		ps1144.OverlayValues[841] = d841
		ps1144.OverlayValues[843] = d843
		ps1144.OverlayValues[844] = d844
		ps1144.OverlayValues[983] = d983
		ps1144.OverlayValues[984] = d984
		ps1144.OverlayValues[987] = d987
		ps1144.OverlayValues[1129] = d1129
		ps1144.OverlayValues[1130] = d1130
		ps1144.OverlayValues[1131] = d1131
		ps1144.OverlayValues[1132] = d1132
		ps1144.OverlayValues[1134] = d1134
		ps1144.OverlayValues[1135] = d1135
		ps1144.OverlayValues[1136] = d1136
		ps1144.OverlayValues[1137] = d1137
		ps1144.OverlayValues[1138] = d1138
		ps1144.OverlayValues[1139] = d1139
		ps1144.OverlayValues[1140] = d1140
		ps1144.OverlayValues[1141] = d1141
		ps1144.OverlayValues[1142] = d1142
		ps1144.OverlayValues[1143] = d1143
		ps1144.PhiValues = make([]scm.JITValueDesc, 3)
		d1145 = d1137
		ps1144.PhiValues[0] = d1145
		d1146 = d12
		ps1144.PhiValues[1] = d1146
		d1147 = d13
		ps1144.PhiValues[2] = d1147
		if ps1144.General && bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		return bbs[1].RenderPS(ps1144)
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
			ctx.FlushRegisterMoves()
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
		if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
			d184 = ps.OverlayValues[184]
		}
		if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != scm.LocNone {
			d335 = ps.OverlayValues[335]
		}
		if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != scm.LocNone {
			d336 = ps.OverlayValues[336]
		}
		if len(ps.OverlayValues) > 337 && ps.OverlayValues[337].Loc != scm.LocNone {
			d337 = ps.OverlayValues[337]
		}
		if len(ps.OverlayValues) > 338 && ps.OverlayValues[338].Loc != scm.LocNone {
			d338 = ps.OverlayValues[338]
		}
		if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != scm.LocNone {
			d340 = ps.OverlayValues[340]
		}
		if len(ps.OverlayValues) > 341 && ps.OverlayValues[341].Loc != scm.LocNone {
			d341 = ps.OverlayValues[341]
		}
		if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != scm.LocNone {
			d342 = ps.OverlayValues[342]
		}
		if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != scm.LocNone {
			d343 = ps.OverlayValues[343]
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
		if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != scm.LocNone {
			d351 = ps.OverlayValues[351]
		}
		if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != scm.LocNone {
			d352 = ps.OverlayValues[352]
		}
		if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != scm.LocNone {
			d353 = ps.OverlayValues[353]
		}
		if len(ps.OverlayValues) > 444 && ps.OverlayValues[444].Loc != scm.LocNone {
			d444 = ps.OverlayValues[444]
		}
		if len(ps.OverlayValues) > 445 && ps.OverlayValues[445].Loc != scm.LocNone {
			d445 = ps.OverlayValues[445]
		}
		if len(ps.OverlayValues) > 448 && ps.OverlayValues[448].Loc != scm.LocNone {
			d448 = ps.OverlayValues[448]
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
		if len(ps.OverlayValues) > 548 && ps.OverlayValues[548].Loc != scm.LocNone {
			d548 = ps.OverlayValues[548]
		}
		if len(ps.OverlayValues) > 549 && ps.OverlayValues[549].Loc != scm.LocNone {
			d549 = ps.OverlayValues[549]
		}
		if len(ps.OverlayValues) > 550 && ps.OverlayValues[550].Loc != scm.LocNone {
			d550 = ps.OverlayValues[550]
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
		if len(ps.OverlayValues) > 555 && ps.OverlayValues[555].Loc != scm.LocNone {
			d555 = ps.OverlayValues[555]
		}
		if len(ps.OverlayValues) > 556 && ps.OverlayValues[556].Loc != scm.LocNone {
			d556 = ps.OverlayValues[556]
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
		if len(ps.OverlayValues) > 562 && ps.OverlayValues[562].Loc != scm.LocNone {
			d562 = ps.OverlayValues[562]
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
		if len(ps.OverlayValues) > 829 && ps.OverlayValues[829].Loc != scm.LocNone {
			d829 = ps.OverlayValues[829]
		}
		if len(ps.OverlayValues) > 830 && ps.OverlayValues[830].Loc != scm.LocNone {
			d830 = ps.OverlayValues[830]
		}
		if len(ps.OverlayValues) > 831 && ps.OverlayValues[831].Loc != scm.LocNone {
			d831 = ps.OverlayValues[831]
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
		if len(ps.OverlayValues) > 838 && ps.OverlayValues[838].Loc != scm.LocNone {
			d838 = ps.OverlayValues[838]
		}
		if len(ps.OverlayValues) > 839 && ps.OverlayValues[839].Loc != scm.LocNone {
			d839 = ps.OverlayValues[839]
		}
		if len(ps.OverlayValues) > 841 && ps.OverlayValues[841].Loc != scm.LocNone {
			d841 = ps.OverlayValues[841]
		}
		if len(ps.OverlayValues) > 843 && ps.OverlayValues[843].Loc != scm.LocNone {
			d843 = ps.OverlayValues[843]
		}
		if len(ps.OverlayValues) > 844 && ps.OverlayValues[844].Loc != scm.LocNone {
			d844 = ps.OverlayValues[844]
		}
		if len(ps.OverlayValues) > 983 && ps.OverlayValues[983].Loc != scm.LocNone {
			d983 = ps.OverlayValues[983]
		}
		if len(ps.OverlayValues) > 984 && ps.OverlayValues[984].Loc != scm.LocNone {
			d984 = ps.OverlayValues[984]
		}
		if len(ps.OverlayValues) > 987 && ps.OverlayValues[987].Loc != scm.LocNone {
			d987 = ps.OverlayValues[987]
		}
		if len(ps.OverlayValues) > 1129 && ps.OverlayValues[1129].Loc != scm.LocNone {
			d1129 = ps.OverlayValues[1129]
		}
		if len(ps.OverlayValues) > 1130 && ps.OverlayValues[1130].Loc != scm.LocNone {
			d1130 = ps.OverlayValues[1130]
		}
		if len(ps.OverlayValues) > 1131 && ps.OverlayValues[1131].Loc != scm.LocNone {
			d1131 = ps.OverlayValues[1131]
		}
		if len(ps.OverlayValues) > 1132 && ps.OverlayValues[1132].Loc != scm.LocNone {
			d1132 = ps.OverlayValues[1132]
		}
		if len(ps.OverlayValues) > 1134 && ps.OverlayValues[1134].Loc != scm.LocNone {
			d1134 = ps.OverlayValues[1134]
		}
		if len(ps.OverlayValues) > 1135 && ps.OverlayValues[1135].Loc != scm.LocNone {
			d1135 = ps.OverlayValues[1135]
		}
		if len(ps.OverlayValues) > 1136 && ps.OverlayValues[1136].Loc != scm.LocNone {
			d1136 = ps.OverlayValues[1136]
		}
		if len(ps.OverlayValues) > 1137 && ps.OverlayValues[1137].Loc != scm.LocNone {
			d1137 = ps.OverlayValues[1137]
		}
		if len(ps.OverlayValues) > 1138 && ps.OverlayValues[1138].Loc != scm.LocNone {
			d1138 = ps.OverlayValues[1138]
		}
		if len(ps.OverlayValues) > 1139 && ps.OverlayValues[1139].Loc != scm.LocNone {
			d1139 = ps.OverlayValues[1139]
		}
		if len(ps.OverlayValues) > 1140 && ps.OverlayValues[1140].Loc != scm.LocNone {
			d1140 = ps.OverlayValues[1140]
		}
		if len(ps.OverlayValues) > 1141 && ps.OverlayValues[1141].Loc != scm.LocNone {
			d1141 = ps.OverlayValues[1141]
		}
		if len(ps.OverlayValues) > 1142 && ps.OverlayValues[1142].Loc != scm.LocNone {
			d1142 = ps.OverlayValues[1142]
		}
		if len(ps.OverlayValues) > 1143 && ps.OverlayValues[1143].Loc != scm.LocNone {
			d1143 = ps.OverlayValues[1143]
		}
		if len(ps.OverlayValues) > 1145 && ps.OverlayValues[1145].Loc != scm.LocNone {
			d1145 = ps.OverlayValues[1145]
		}
		if len(ps.OverlayValues) > 1146 && ps.OverlayValues[1146].Loc != scm.LocNone {
			d1146 = ps.OverlayValues[1146]
		}
		if len(ps.OverlayValues) > 1147 && ps.OverlayValues[1147].Loc != scm.LocNone {
			d1147 = ps.OverlayValues[1147]
		}
		ctx.ReclaimUntrackedRegs()
		d1148 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d1149 = result
		ctx.EnsureDesc(&d1148)
		if d1148.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d1148, &d1149)
		} else {
			switch d1148.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d1149, d1148)
			case scm.TagInt:
				ctx.EmitMakeInt(d1149, d1148)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d1149, d1148)
			case scm.TagNil:
				ctx.EmitMakeNil(d1149)
			default:
				ctx.EmitMovPairToResult(&d1148, &d1149)
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
			ctx.FlushRegisterMoves()
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
		if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
			d184 = ps.OverlayValues[184]
		}
		if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != scm.LocNone {
			d335 = ps.OverlayValues[335]
		}
		if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != scm.LocNone {
			d336 = ps.OverlayValues[336]
		}
		if len(ps.OverlayValues) > 337 && ps.OverlayValues[337].Loc != scm.LocNone {
			d337 = ps.OverlayValues[337]
		}
		if len(ps.OverlayValues) > 338 && ps.OverlayValues[338].Loc != scm.LocNone {
			d338 = ps.OverlayValues[338]
		}
		if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != scm.LocNone {
			d340 = ps.OverlayValues[340]
		}
		if len(ps.OverlayValues) > 341 && ps.OverlayValues[341].Loc != scm.LocNone {
			d341 = ps.OverlayValues[341]
		}
		if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != scm.LocNone {
			d342 = ps.OverlayValues[342]
		}
		if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != scm.LocNone {
			d343 = ps.OverlayValues[343]
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
		if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != scm.LocNone {
			d351 = ps.OverlayValues[351]
		}
		if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != scm.LocNone {
			d352 = ps.OverlayValues[352]
		}
		if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != scm.LocNone {
			d353 = ps.OverlayValues[353]
		}
		if len(ps.OverlayValues) > 444 && ps.OverlayValues[444].Loc != scm.LocNone {
			d444 = ps.OverlayValues[444]
		}
		if len(ps.OverlayValues) > 445 && ps.OverlayValues[445].Loc != scm.LocNone {
			d445 = ps.OverlayValues[445]
		}
		if len(ps.OverlayValues) > 448 && ps.OverlayValues[448].Loc != scm.LocNone {
			d448 = ps.OverlayValues[448]
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
		if len(ps.OverlayValues) > 548 && ps.OverlayValues[548].Loc != scm.LocNone {
			d548 = ps.OverlayValues[548]
		}
		if len(ps.OverlayValues) > 549 && ps.OverlayValues[549].Loc != scm.LocNone {
			d549 = ps.OverlayValues[549]
		}
		if len(ps.OverlayValues) > 550 && ps.OverlayValues[550].Loc != scm.LocNone {
			d550 = ps.OverlayValues[550]
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
		if len(ps.OverlayValues) > 555 && ps.OverlayValues[555].Loc != scm.LocNone {
			d555 = ps.OverlayValues[555]
		}
		if len(ps.OverlayValues) > 556 && ps.OverlayValues[556].Loc != scm.LocNone {
			d556 = ps.OverlayValues[556]
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
		if len(ps.OverlayValues) > 562 && ps.OverlayValues[562].Loc != scm.LocNone {
			d562 = ps.OverlayValues[562]
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
		if len(ps.OverlayValues) > 829 && ps.OverlayValues[829].Loc != scm.LocNone {
			d829 = ps.OverlayValues[829]
		}
		if len(ps.OverlayValues) > 830 && ps.OverlayValues[830].Loc != scm.LocNone {
			d830 = ps.OverlayValues[830]
		}
		if len(ps.OverlayValues) > 831 && ps.OverlayValues[831].Loc != scm.LocNone {
			d831 = ps.OverlayValues[831]
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
		if len(ps.OverlayValues) > 838 && ps.OverlayValues[838].Loc != scm.LocNone {
			d838 = ps.OverlayValues[838]
		}
		if len(ps.OverlayValues) > 839 && ps.OverlayValues[839].Loc != scm.LocNone {
			d839 = ps.OverlayValues[839]
		}
		if len(ps.OverlayValues) > 841 && ps.OverlayValues[841].Loc != scm.LocNone {
			d841 = ps.OverlayValues[841]
		}
		if len(ps.OverlayValues) > 843 && ps.OverlayValues[843].Loc != scm.LocNone {
			d843 = ps.OverlayValues[843]
		}
		if len(ps.OverlayValues) > 844 && ps.OverlayValues[844].Loc != scm.LocNone {
			d844 = ps.OverlayValues[844]
		}
		if len(ps.OverlayValues) > 983 && ps.OverlayValues[983].Loc != scm.LocNone {
			d983 = ps.OverlayValues[983]
		}
		if len(ps.OverlayValues) > 984 && ps.OverlayValues[984].Loc != scm.LocNone {
			d984 = ps.OverlayValues[984]
		}
		if len(ps.OverlayValues) > 987 && ps.OverlayValues[987].Loc != scm.LocNone {
			d987 = ps.OverlayValues[987]
		}
		if len(ps.OverlayValues) > 1129 && ps.OverlayValues[1129].Loc != scm.LocNone {
			d1129 = ps.OverlayValues[1129]
		}
		if len(ps.OverlayValues) > 1130 && ps.OverlayValues[1130].Loc != scm.LocNone {
			d1130 = ps.OverlayValues[1130]
		}
		if len(ps.OverlayValues) > 1131 && ps.OverlayValues[1131].Loc != scm.LocNone {
			d1131 = ps.OverlayValues[1131]
		}
		if len(ps.OverlayValues) > 1132 && ps.OverlayValues[1132].Loc != scm.LocNone {
			d1132 = ps.OverlayValues[1132]
		}
		if len(ps.OverlayValues) > 1134 && ps.OverlayValues[1134].Loc != scm.LocNone {
			d1134 = ps.OverlayValues[1134]
		}
		if len(ps.OverlayValues) > 1135 && ps.OverlayValues[1135].Loc != scm.LocNone {
			d1135 = ps.OverlayValues[1135]
		}
		if len(ps.OverlayValues) > 1136 && ps.OverlayValues[1136].Loc != scm.LocNone {
			d1136 = ps.OverlayValues[1136]
		}
		if len(ps.OverlayValues) > 1137 && ps.OverlayValues[1137].Loc != scm.LocNone {
			d1137 = ps.OverlayValues[1137]
		}
		if len(ps.OverlayValues) > 1138 && ps.OverlayValues[1138].Loc != scm.LocNone {
			d1138 = ps.OverlayValues[1138]
		}
		if len(ps.OverlayValues) > 1139 && ps.OverlayValues[1139].Loc != scm.LocNone {
			d1139 = ps.OverlayValues[1139]
		}
		if len(ps.OverlayValues) > 1140 && ps.OverlayValues[1140].Loc != scm.LocNone {
			d1140 = ps.OverlayValues[1140]
		}
		if len(ps.OverlayValues) > 1141 && ps.OverlayValues[1141].Loc != scm.LocNone {
			d1141 = ps.OverlayValues[1141]
		}
		if len(ps.OverlayValues) > 1142 && ps.OverlayValues[1142].Loc != scm.LocNone {
			d1142 = ps.OverlayValues[1142]
		}
		if len(ps.OverlayValues) > 1143 && ps.OverlayValues[1143].Loc != scm.LocNone {
			d1143 = ps.OverlayValues[1143]
		}
		if len(ps.OverlayValues) > 1145 && ps.OverlayValues[1145].Loc != scm.LocNone {
			d1145 = ps.OverlayValues[1145]
		}
		if len(ps.OverlayValues) > 1146 && ps.OverlayValues[1146].Loc != scm.LocNone {
			d1146 = ps.OverlayValues[1146]
		}
		if len(ps.OverlayValues) > 1147 && ps.OverlayValues[1147].Loc != scm.LocNone {
			d1147 = ps.OverlayValues[1147]
		}
		if len(ps.OverlayValues) > 1148 && ps.OverlayValues[1148].Loc != scm.LocNone {
			d1148 = ps.OverlayValues[1148]
		}
		if len(ps.OverlayValues) > 1149 && ps.OverlayValues[1149].Loc != scm.LocNone {
			d1149 = ps.OverlayValues[1149]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d179)
		ctx.EnsureDesc(&d179)
		var d1150 scm.JITValueDesc
		if d179.Loc == scm.LocImm {
			d1150 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d179.Imm.Int()))))}
		} else {
			r99 := ctx.AllocReg()
			ctx.EmitMovRegReg(r99, d179.Reg)
			d1150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
			ctx.BindReg(r99, &d1150)
		}
		var d1151 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d1151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 56)
			r100 := ctx.AllocReg()
			ctx.EmitMovRegMem(r100, thisptr.Reg, off)
			d1151 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r100}
			ctx.BindReg(r100, &d1151)
		}
		ctx.EnsureDesc(&d1150)
		ctx.EnsureDesc(&d1151)
		ctx.EnsureDescsTogether(&d1150, &d1151)
		var d1152 scm.JITValueDesc
		if d1150.Loc == scm.LocImm && d1151.Loc == scm.LocImm {
			d1152 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1150.Imm.Int() + d1151.Imm.Int())}
		} else if d1151.Loc == scm.LocImm && d1151.Imm.Int() == 0 {
			r101 := ctx.AllocRegExcept(d1150.Reg)
			ctx.EmitMovRegReg(r101, d1150.Reg)
			d1152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r101}
			ctx.BindReg(r101, &d1152)
		} else if d1150.Loc == scm.LocImm && d1150.Imm.Int() == 0 {
			d1152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1151.Reg}
			ctx.BindReg(d1151.Reg, &d1152)
		} else if d1150.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1151.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1150.Imm.Int()))
			ctx.EmitAddInt64(scratch, d1151.Reg)
			d1152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1152)
		} else if d1151.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1150.Reg)
			ctx.EmitMovRegReg(scratch, d1150.Reg)
			if d1151.Imm.Int() >= -2147483648 && d1151.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d1151.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1151.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d1152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1152)
		} else {
			r102 := ctx.AllocRegExcept(d1150.Reg, d1151.Reg)
			ctx.EmitMovRegReg(r102, d1150.Reg)
			ctx.EmitAddInt64(r102, d1151.Reg)
			d1152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r102}
			ctx.BindReg(r102, &d1152)
		}
		if d1152.Loc == scm.LocReg && d1150.Loc == scm.LocReg && d1152.Reg == d1150.Reg {
			ctx.TransferReg(d1150.Reg)
			d1150.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1150)
		ctx.FreeDesc(&d1151)
		ctx.EnsureDesc(&d8)
		d1153 = d8
		_ = d1153
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
		var d1154 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d1154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 48)
			r103 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r103, thisptr.Reg, off)
			d1154 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r103}
			ctx.BindReg(r103, &d1154)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1154)
		ctx.EnsureDesc(&d1154)
		var d1155 scm.JITValueDesc
		if d1154.Loc == scm.LocImm {
			d1155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d1154.Imm.Int()))))}
		} else {
			r104 := ctx.AllocReg()
			ctx.EmitMovRegReg(r104, d1154.Reg)
			ctx.EmitShlRegImm8(r104, 56)
			ctx.EmitShrRegImm8(r104, 56)
			d1155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r104}
			ctx.BindReg(r104, &d1155)
		}
		ctx.FreeDesc(&d1154)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1153)
		ctx.EnsureDesc(&d1153)
		var d1156 scm.JITValueDesc
		if d1153.Loc == scm.LocImm {
			d1156 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d1153.Imm.Int()))))}
		} else {
			r105 := ctx.AllocReg()
			ctx.EmitMovRegReg(r105, d1153.Reg)
			ctx.EmitShlRegImm8(r105, 32)
			ctx.EmitShrRegImm8(r105, 32)
			d1156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r105}
			ctx.BindReg(r105, &d1156)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1156)
		ctx.EnsureDesc(&d1155)
		ctx.EnsureDescsTogether(&d1156, &d1155)
		var d1157 scm.JITValueDesc
		if d1156.Loc == scm.LocImm && d1155.Loc == scm.LocImm {
			d1157 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1156.Imm.Int() * d1155.Imm.Int())}
		} else if d1156.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1155.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1156.Imm.Int()))
			ctx.EmitImulInt64(scratch, d1155.Reg)
			d1157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1157)
		} else if d1155.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1156.Reg)
			ctx.EmitMovRegReg(scratch, d1156.Reg)
			if d1155.Imm.Int() >= -2147483648 && d1155.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d1155.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1155.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d1157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1157)
		} else {
			r106 := ctx.AllocRegExcept(d1156.Reg, d1155.Reg)
			ctx.EmitMovRegReg(r106, d1156.Reg)
			ctx.EmitImulInt64(r106, d1155.Reg)
			d1157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r106}
			ctx.BindReg(r106, &d1157)
		}
		if d1157.Loc == scm.LocReg && d1156.Loc == scm.LocReg && d1157.Reg == d1156.Reg {
			ctx.TransferReg(d1156.Reg)
			d1156.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1156)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1157)
		var d1158 scm.JITValueDesc
		if d1157.Loc == scm.LocImm {
			d1158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1157.Imm.Int() / 64)}
		} else {
			r107 := ctx.AllocRegExcept(d1157.Reg)
			ctx.EmitMovRegReg(r107, d1157.Reg)
			ctx.EmitShrRegImm8(r107, 6)
			d1158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r107}
			ctx.BindReg(r107, &d1158)
		}
		if d1158.Loc == scm.LocReg && d1157.Loc == scm.LocReg && d1158.Reg == d1157.Reg {
			ctx.TransferReg(d1157.Reg)
			d1157.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1157)
		var d1159 scm.JITValueDesc
		if d1157.Loc == scm.LocImm {
			d1159 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1157.Imm.Int() % 64)}
		} else {
			r108 := ctx.AllocRegExcept(d1157.Reg)
			ctx.EmitMovRegReg(r108, d1157.Reg)
			ctx.EmitAndRegImm32(r108, 63)
			d1159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
			ctx.BindReg(r108, &d1159)
		}
		if d1159.Loc == scm.LocReg && d1157.Loc == scm.LocReg && d1159.Reg == d1157.Reg {
			ctx.TransferReg(d1157.Reg)
			d1157.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1157)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d1160 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d1160 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r109 := ctx.AllocReg()
			r110 := ctx.AllocRegExcept(r109)
			r111 := ctx.AllocRegExcept(r109, r110)
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
			ctx.EmitMovRegMem(r109, thisptr.Reg, off)
			ctx.EmitMovRegMem(r110, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r111, thisptr.Reg, off+16)
			d1160 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r109, Reg2: r110, Reg3: r111}
			ctx.BindReg(r109, &d1160)
			ctx.BindReg(r110, &d1160)
			ctx.BindReg(r111, &d1160)
			ctx.BindReg(r109, &d1160)
			ctx.BindReg(r110, &d1160)
			ctx.BindReg(r111, &d1160)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1158)
		ctx.ReclaimUntrackedRegs()
		d1161 = ctx.EmitLoadScalarSliceElement(&d1160, &d1158, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1161)
		ctx.EnsureDesc(&d1159)
		ctx.EnsureDescsTogether(&d1161, &d1159)
		var d1162 scm.JITValueDesc
		if d1161.Loc == scm.LocImm && d1159.Loc == scm.LocImm {
			d1162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1161.Imm.Int()) << uint64(d1159.Imm.Int())))}
		} else if d1159.Loc == scm.LocImm {
			r112 := ctx.AllocRegExcept(d1161.Reg)
			ctx.EmitMovRegReg(r112, d1161.Reg)
			ctx.EmitShlRegImm8(r112, uint8(d1159.Imm.Int()))
			d1162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r112}
			ctx.BindReg(r112, &d1162)
		} else {
			{
				shiftSrc := d1161.Reg
				r113 := ctx.AllocRegExcept(d1161.Reg, d1159.Reg)
				ctx.EmitMovRegReg(r113, d1161.Reg)
				shiftSrc = r113
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d1159.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d1159.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d1159.Reg)
				}
				ctx.EmitShlRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d1162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d1162)
			}
		}
		if d1162.Loc == scm.LocReg && d1161.Loc == scm.LocReg && d1162.Reg == d1161.Reg {
			ctx.TransferReg(d1161.Reg)
			d1161.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1161)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1158)
		ctx.EnsureDesc(&d1158)
		var d1163 scm.JITValueDesc
		if d1158.Loc == scm.LocImm {
			d1163 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1158.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d1158.Reg)
			ctx.EmitMovRegReg(scratch, d1158.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d1163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1163)
		}
		if d1163.Loc == scm.LocReg && d1158.Loc == scm.LocReg && d1163.Reg == d1158.Reg {
			ctx.TransferReg(d1158.Reg)
			d1158.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1158)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1163)
		ctx.ReclaimUntrackedRegs()
		d1164 = ctx.EmitLoadScalarSliceElement(&d1160, &d1163, 8, scm.TagInt)
		ctx.FreeDesc(&d1163)
		ctx.ReclaimUntrackedRegs()
		d1165 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d1159)
		ctx.EnsureDescsTogether(&d1165, &d1159)
		var d1166 scm.JITValueDesc
		if d1165.Loc == scm.LocImm && d1159.Loc == scm.LocImm {
			d1166 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1165.Imm.Int() - d1159.Imm.Int())}
		} else if d1159.Loc == scm.LocImm && d1159.Imm.Int() == 0 {
			r114 := ctx.AllocRegExcept(d1165.Reg)
			ctx.EmitMovRegReg(r114, d1165.Reg)
			d1166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r114}
			ctx.BindReg(r114, &d1166)
		} else if d1165.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1159.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1165.Imm.Int()))
			ctx.EmitSubInt64(scratch, d1159.Reg)
			d1166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1166)
		} else if d1159.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1165.Reg)
			ctx.EmitMovRegReg(scratch, d1165.Reg)
			if d1159.Imm.Int() >= -2147483648 && d1159.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d1159.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1159.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d1166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1166)
		} else {
			r115 := ctx.AllocRegExcept(d1165.Reg, d1159.Reg)
			ctx.EmitMovRegReg(r115, d1165.Reg)
			ctx.EmitSubInt64(r115, d1159.Reg)
			d1166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
			ctx.BindReg(r115, &d1166)
		}
		if d1166.Loc == scm.LocReg && d1165.Loc == scm.LocReg && d1166.Reg == d1165.Reg {
			ctx.TransferReg(d1165.Reg)
			d1165.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1159)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1164)
		ctx.EnsureDesc(&d1166)
		ctx.EnsureDescsTogether(&d1164, &d1166)
		var d1167 scm.JITValueDesc
		if d1164.Loc == scm.LocImm && d1166.Loc == scm.LocImm {
			d1167 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1164.Imm.Int()) >> uint64(d1166.Imm.Int())))}
		} else if d1166.Loc == scm.LocImm {
			r116 := ctx.AllocRegExcept(d1164.Reg)
			ctx.EmitMovRegReg(r116, d1164.Reg)
			ctx.EmitShrRegImm8(r116, uint8(d1166.Imm.Int()))
			d1167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r116}
			ctx.BindReg(r116, &d1167)
		} else {
			{
				shiftSrc := d1164.Reg
				r117 := ctx.AllocRegExcept(d1164.Reg, d1166.Reg)
				ctx.EmitMovRegReg(r117, d1164.Reg)
				shiftSrc = r117
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d1166.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d1166.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d1166.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d1167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d1167)
			}
		}
		if d1167.Loc == scm.LocReg && d1164.Loc == scm.LocReg && d1167.Reg == d1164.Reg {
			ctx.TransferReg(d1164.Reg)
			d1164.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1164)
		ctx.FreeDesc(&d1166)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1162)
		ctx.EnsureDesc(&d1167)
		var d1168 scm.JITValueDesc
		if d1162.Loc == scm.LocImm && d1167.Loc == scm.LocImm {
			d1168 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1162.Imm.Int() | d1167.Imm.Int())}
		} else if d1162.Loc == scm.LocImm && d1162.Imm.Int() == 0 {
			d1168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1167.Reg}
			ctx.BindReg(d1167.Reg, &d1168)
		} else if d1167.Loc == scm.LocImm && d1167.Imm.Int() == 0 {
			r118 := ctx.AllocRegExcept(d1162.Reg)
			ctx.EmitMovRegReg(r118, d1162.Reg)
			d1168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r118}
			ctx.BindReg(r118, &d1168)
		} else if d1162.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1167.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1162.Imm.Int()))
			ctx.EmitOrInt64(scratch, d1167.Reg)
			d1168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1168)
		} else if d1167.Loc == scm.LocImm {
			r119 := ctx.AllocRegExcept(d1162.Reg)
			ctx.EmitMovRegReg(r119, d1162.Reg)
			if d1167.Imm.Int() >= -2147483648 && d1167.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r119, int32(d1167.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1167.Imm.Int()))
				ctx.EmitOrInt64(r119, scm.RegR11)
			}
			d1168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
			ctx.BindReg(r119, &d1168)
		} else {
			r120 := ctx.AllocRegExcept(d1162.Reg, d1167.Reg)
			ctx.EmitMovRegReg(r120, d1162.Reg)
			ctx.EmitOrInt64(r120, d1167.Reg)
			d1168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r120}
			ctx.BindReg(r120, &d1168)
		}
		if d1168.Loc == scm.LocReg && d1162.Loc == scm.LocReg && d1168.Reg == d1162.Reg {
			ctx.TransferReg(d1162.Reg)
			d1162.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1162)
		ctx.FreeDesc(&d1167)
		ctx.ReclaimUntrackedRegs()
		d1169 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d1155)
		ctx.EnsureDescsTogether(&d1169, &d1155)
		var d1170 scm.JITValueDesc
		if d1169.Loc == scm.LocImm && d1155.Loc == scm.LocImm {
			d1170 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1169.Imm.Int() - d1155.Imm.Int())}
		} else if d1155.Loc == scm.LocImm && d1155.Imm.Int() == 0 {
			r121 := ctx.AllocRegExcept(d1169.Reg)
			ctx.EmitMovRegReg(r121, d1169.Reg)
			d1170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r121}
			ctx.BindReg(r121, &d1170)
		} else if d1169.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1155.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1169.Imm.Int()))
			ctx.EmitSubInt64(scratch, d1155.Reg)
			d1170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1170)
		} else if d1155.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1169.Reg)
			ctx.EmitMovRegReg(scratch, d1169.Reg)
			if d1155.Imm.Int() >= -2147483648 && d1155.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d1155.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1155.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d1170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1170)
		} else {
			r122 := ctx.AllocRegExcept(d1169.Reg, d1155.Reg)
			ctx.EmitMovRegReg(r122, d1169.Reg)
			ctx.EmitSubInt64(r122, d1155.Reg)
			d1170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
			ctx.BindReg(r122, &d1170)
		}
		if d1170.Loc == scm.LocReg && d1169.Loc == scm.LocReg && d1170.Reg == d1169.Reg {
			ctx.TransferReg(d1169.Reg)
			d1169.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1155)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1168)
		ctx.EnsureDesc(&d1170)
		ctx.EnsureDescsTogether(&d1168, &d1170)
		var d1171 scm.JITValueDesc
		if d1168.Loc == scm.LocImm && d1170.Loc == scm.LocImm {
			d1171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1168.Imm.Int()) >> uint64(d1170.Imm.Int())))}
		} else if d1170.Loc == scm.LocImm {
			r123 := ctx.AllocRegExcept(d1168.Reg)
			ctx.EmitMovRegReg(r123, d1168.Reg)
			ctx.EmitShrRegImm8(r123, uint8(d1170.Imm.Int()))
			d1171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
			ctx.BindReg(r123, &d1171)
		} else {
			{
				shiftSrc := d1168.Reg
				r124 := ctx.AllocRegExcept(d1168.Reg, d1170.Reg)
				ctx.EmitMovRegReg(r124, d1168.Reg)
				shiftSrc = r124
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d1170.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d1170.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d1170.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d1171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d1171)
			}
		}
		if d1171.Loc == scm.LocReg && d1168.Loc == scm.LocReg && d1171.Reg == d1168.Reg {
			ctx.TransferReg(d1168.Reg)
			d1168.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1168)
		ctx.FreeDesc(&d1170)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1171)
		ctx.EnsureDesc(&d1171)
		ctx.EnsureDesc(&d1171)
		var d1172 scm.JITValueDesc
		if d1171.Loc == scm.LocImm {
			d1172 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d1171.Imm.Int()))))}
		} else {
			r125 := ctx.AllocReg()
			ctx.EmitMovRegReg(r125, d1171.Reg)
			d1172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
			ctx.BindReg(r125, &d1172)
		}
		ctx.FreeDesc(&d1171)
		var d1173 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d1173 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 56)
			r126 := ctx.AllocReg()
			ctx.EmitMovRegMem(r126, thisptr.Reg, off)
			d1173 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r126}
			ctx.BindReg(r126, &d1173)
		}
		ctx.EnsureDesc(&d1172)
		ctx.EnsureDesc(&d1173)
		ctx.EnsureDescsTogether(&d1172, &d1173)
		var d1174 scm.JITValueDesc
		if d1172.Loc == scm.LocImm && d1173.Loc == scm.LocImm {
			d1174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1172.Imm.Int() + d1173.Imm.Int())}
		} else if d1173.Loc == scm.LocImm && d1173.Imm.Int() == 0 {
			r127 := ctx.AllocRegExcept(d1172.Reg)
			ctx.EmitMovRegReg(r127, d1172.Reg)
			d1174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
			ctx.BindReg(r127, &d1174)
		} else if d1172.Loc == scm.LocImm && d1172.Imm.Int() == 0 {
			d1174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1173.Reg}
			ctx.BindReg(d1173.Reg, &d1174)
		} else if d1172.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1173.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1172.Imm.Int()))
			ctx.EmitAddInt64(scratch, d1173.Reg)
			d1174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1174)
		} else if d1173.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1172.Reg)
			ctx.EmitMovRegReg(scratch, d1172.Reg)
			if d1173.Imm.Int() >= -2147483648 && d1173.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d1173.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1173.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d1174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1174)
		} else {
			r128 := ctx.AllocRegExcept(d1172.Reg, d1173.Reg)
			ctx.EmitMovRegReg(r128, d1172.Reg)
			ctx.EmitAddInt64(r128, d1173.Reg)
			d1174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
			ctx.BindReg(r128, &d1174)
		}
		if d1174.Loc == scm.LocReg && d1172.Loc == scm.LocReg && d1174.Reg == d1172.Reg {
			ctx.TransferReg(d1172.Reg)
			d1172.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1172)
		ctx.FreeDesc(&d1173)
		ctx.EnsureDesc(&d8)
		d1175 = d8
		_ = d1175
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
		var d1176 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d1176 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48)
			r129 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r129, thisptr.Reg, off)
			d1176 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r129}
			ctx.BindReg(r129, &d1176)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1176)
		ctx.EnsureDesc(&d1176)
		var d1177 scm.JITValueDesc
		if d1176.Loc == scm.LocImm {
			d1177 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d1176.Imm.Int()))))}
		} else {
			r130 := ctx.AllocReg()
			ctx.EmitMovRegReg(r130, d1176.Reg)
			ctx.EmitShlRegImm8(r130, 56)
			ctx.EmitShrRegImm8(r130, 56)
			d1177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
			ctx.BindReg(r130, &d1177)
		}
		ctx.FreeDesc(&d1176)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1175)
		ctx.EnsureDesc(&d1175)
		var d1178 scm.JITValueDesc
		if d1175.Loc == scm.LocImm {
			d1178 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d1175.Imm.Int()))))}
		} else {
			r131 := ctx.AllocReg()
			ctx.EmitMovRegReg(r131, d1175.Reg)
			ctx.EmitShlRegImm8(r131, 32)
			ctx.EmitShrRegImm8(r131, 32)
			d1178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r131}
			ctx.BindReg(r131, &d1178)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1178)
		ctx.EnsureDesc(&d1177)
		ctx.EnsureDescsTogether(&d1178, &d1177)
		var d1179 scm.JITValueDesc
		if d1178.Loc == scm.LocImm && d1177.Loc == scm.LocImm {
			d1179 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1178.Imm.Int() * d1177.Imm.Int())}
		} else if d1178.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1177.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1178.Imm.Int()))
			ctx.EmitImulInt64(scratch, d1177.Reg)
			d1179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1179)
		} else if d1177.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1178.Reg)
			ctx.EmitMovRegReg(scratch, d1178.Reg)
			if d1177.Imm.Int() >= -2147483648 && d1177.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d1177.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1177.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d1179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1179)
		} else {
			r132 := ctx.AllocRegExcept(d1178.Reg, d1177.Reg)
			ctx.EmitMovRegReg(r132, d1178.Reg)
			ctx.EmitImulInt64(r132, d1177.Reg)
			d1179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
			ctx.BindReg(r132, &d1179)
		}
		if d1179.Loc == scm.LocReg && d1178.Loc == scm.LocReg && d1179.Reg == d1178.Reg {
			ctx.TransferReg(d1178.Reg)
			d1178.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1178)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1179)
		var d1180 scm.JITValueDesc
		if d1179.Loc == scm.LocImm {
			d1180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1179.Imm.Int() / 64)}
		} else {
			r133 := ctx.AllocRegExcept(d1179.Reg)
			ctx.EmitMovRegReg(r133, d1179.Reg)
			ctx.EmitShrRegImm8(r133, 6)
			d1180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
			ctx.BindReg(r133, &d1180)
		}
		if d1180.Loc == scm.LocReg && d1179.Loc == scm.LocReg && d1180.Reg == d1179.Reg {
			ctx.TransferReg(d1179.Reg)
			d1179.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1179)
		var d1181 scm.JITValueDesc
		if d1179.Loc == scm.LocImm {
			d1181 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1179.Imm.Int() % 64)}
		} else {
			r134 := ctx.AllocRegExcept(d1179.Reg)
			ctx.EmitMovRegReg(r134, d1179.Reg)
			ctx.EmitAndRegImm32(r134, 63)
			d1181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r134}
			ctx.BindReg(r134, &d1181)
		}
		if d1181.Loc == scm.LocReg && d1179.Loc == scm.LocReg && d1181.Reg == d1179.Reg {
			ctx.TransferReg(d1179.Reg)
			d1179.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1179)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d1182 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d1182 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r135 := ctx.AllocReg()
			r136 := ctx.AllocRegExcept(r135)
			r137 := ctx.AllocRegExcept(r135, r136)
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			ctx.EmitMovRegMem(r135, thisptr.Reg, off)
			ctx.EmitMovRegMem(r136, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r137, thisptr.Reg, off+16)
			d1182 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r135, Reg2: r136, Reg3: r137}
			ctx.BindReg(r135, &d1182)
			ctx.BindReg(r136, &d1182)
			ctx.BindReg(r137, &d1182)
			ctx.BindReg(r135, &d1182)
			ctx.BindReg(r136, &d1182)
			ctx.BindReg(r137, &d1182)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1180)
		ctx.ReclaimUntrackedRegs()
		d1183 = ctx.EmitLoadScalarSliceElement(&d1182, &d1180, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1183)
		ctx.EnsureDesc(&d1181)
		ctx.EnsureDescsTogether(&d1183, &d1181)
		var d1184 scm.JITValueDesc
		if d1183.Loc == scm.LocImm && d1181.Loc == scm.LocImm {
			d1184 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1183.Imm.Int()) << uint64(d1181.Imm.Int())))}
		} else if d1181.Loc == scm.LocImm {
			r138 := ctx.AllocRegExcept(d1183.Reg)
			ctx.EmitMovRegReg(r138, d1183.Reg)
			ctx.EmitShlRegImm8(r138, uint8(d1181.Imm.Int()))
			d1184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
			ctx.BindReg(r138, &d1184)
		} else {
			{
				shiftSrc := d1183.Reg
				r139 := ctx.AllocRegExcept(d1183.Reg, d1181.Reg)
				ctx.EmitMovRegReg(r139, d1183.Reg)
				shiftSrc = r139
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d1181.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d1181.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d1181.Reg)
				}
				ctx.EmitShlRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d1184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d1184)
			}
		}
		if d1184.Loc == scm.LocReg && d1183.Loc == scm.LocReg && d1184.Reg == d1183.Reg {
			ctx.TransferReg(d1183.Reg)
			d1183.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1183)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1180)
		ctx.EnsureDesc(&d1180)
		var d1185 scm.JITValueDesc
		if d1180.Loc == scm.LocImm {
			d1185 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1180.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d1180.Reg)
			ctx.EmitMovRegReg(scratch, d1180.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d1185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1185)
		}
		if d1185.Loc == scm.LocReg && d1180.Loc == scm.LocReg && d1185.Reg == d1180.Reg {
			ctx.TransferReg(d1180.Reg)
			d1180.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1180)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1185)
		ctx.ReclaimUntrackedRegs()
		d1186 = ctx.EmitLoadScalarSliceElement(&d1182, &d1185, 8, scm.TagInt)
		ctx.FreeDesc(&d1185)
		ctx.ReclaimUntrackedRegs()
		d1187 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d1181)
		ctx.EnsureDescsTogether(&d1187, &d1181)
		var d1188 scm.JITValueDesc
		if d1187.Loc == scm.LocImm && d1181.Loc == scm.LocImm {
			d1188 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1187.Imm.Int() - d1181.Imm.Int())}
		} else if d1181.Loc == scm.LocImm && d1181.Imm.Int() == 0 {
			r140 := ctx.AllocRegExcept(d1187.Reg)
			ctx.EmitMovRegReg(r140, d1187.Reg)
			d1188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r140}
			ctx.BindReg(r140, &d1188)
		} else if d1187.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1181.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1187.Imm.Int()))
			ctx.EmitSubInt64(scratch, d1181.Reg)
			d1188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1188)
		} else if d1181.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1187.Reg)
			ctx.EmitMovRegReg(scratch, d1187.Reg)
			if d1181.Imm.Int() >= -2147483648 && d1181.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d1181.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1181.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d1188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1188)
		} else {
			r141 := ctx.AllocRegExcept(d1187.Reg, d1181.Reg)
			ctx.EmitMovRegReg(r141, d1187.Reg)
			ctx.EmitSubInt64(r141, d1181.Reg)
			d1188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
			ctx.BindReg(r141, &d1188)
		}
		if d1188.Loc == scm.LocReg && d1187.Loc == scm.LocReg && d1188.Reg == d1187.Reg {
			ctx.TransferReg(d1187.Reg)
			d1187.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1181)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1186)
		ctx.EnsureDesc(&d1188)
		ctx.EnsureDescsTogether(&d1186, &d1188)
		var d1189 scm.JITValueDesc
		if d1186.Loc == scm.LocImm && d1188.Loc == scm.LocImm {
			d1189 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1186.Imm.Int()) >> uint64(d1188.Imm.Int())))}
		} else if d1188.Loc == scm.LocImm {
			r142 := ctx.AllocRegExcept(d1186.Reg)
			ctx.EmitMovRegReg(r142, d1186.Reg)
			ctx.EmitShrRegImm8(r142, uint8(d1188.Imm.Int()))
			d1189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r142}
			ctx.BindReg(r142, &d1189)
		} else {
			{
				shiftSrc := d1186.Reg
				r143 := ctx.AllocRegExcept(d1186.Reg, d1188.Reg)
				ctx.EmitMovRegReg(r143, d1186.Reg)
				shiftSrc = r143
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d1188.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d1188.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d1188.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d1189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d1189)
			}
		}
		if d1189.Loc == scm.LocReg && d1186.Loc == scm.LocReg && d1189.Reg == d1186.Reg {
			ctx.TransferReg(d1186.Reg)
			d1186.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1186)
		ctx.FreeDesc(&d1188)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1184)
		ctx.EnsureDesc(&d1189)
		var d1190 scm.JITValueDesc
		if d1184.Loc == scm.LocImm && d1189.Loc == scm.LocImm {
			d1190 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1184.Imm.Int() | d1189.Imm.Int())}
		} else if d1184.Loc == scm.LocImm && d1184.Imm.Int() == 0 {
			d1190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1189.Reg}
			ctx.BindReg(d1189.Reg, &d1190)
		} else if d1189.Loc == scm.LocImm && d1189.Imm.Int() == 0 {
			r144 := ctx.AllocRegExcept(d1184.Reg)
			ctx.EmitMovRegReg(r144, d1184.Reg)
			d1190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
			ctx.BindReg(r144, &d1190)
		} else if d1184.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1189.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1184.Imm.Int()))
			ctx.EmitOrInt64(scratch, d1189.Reg)
			d1190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1190)
		} else if d1189.Loc == scm.LocImm {
			r145 := ctx.AllocRegExcept(d1184.Reg)
			ctx.EmitMovRegReg(r145, d1184.Reg)
			if d1189.Imm.Int() >= -2147483648 && d1189.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r145, int32(d1189.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1189.Imm.Int()))
				ctx.EmitOrInt64(r145, scm.RegR11)
			}
			d1190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r145}
			ctx.BindReg(r145, &d1190)
		} else {
			r146 := ctx.AllocRegExcept(d1184.Reg, d1189.Reg)
			ctx.EmitMovRegReg(r146, d1184.Reg)
			ctx.EmitOrInt64(r146, d1189.Reg)
			d1190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r146}
			ctx.BindReg(r146, &d1190)
		}
		if d1190.Loc == scm.LocReg && d1184.Loc == scm.LocReg && d1190.Reg == d1184.Reg {
			ctx.TransferReg(d1184.Reg)
			d1184.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1184)
		ctx.FreeDesc(&d1189)
		ctx.ReclaimUntrackedRegs()
		d1191 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d1177)
		ctx.EnsureDescsTogether(&d1191, &d1177)
		var d1192 scm.JITValueDesc
		if d1191.Loc == scm.LocImm && d1177.Loc == scm.LocImm {
			d1192 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1191.Imm.Int() - d1177.Imm.Int())}
		} else if d1177.Loc == scm.LocImm && d1177.Imm.Int() == 0 {
			r147 := ctx.AllocRegExcept(d1191.Reg)
			ctx.EmitMovRegReg(r147, d1191.Reg)
			d1192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r147}
			ctx.BindReg(r147, &d1192)
		} else if d1191.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1177.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1191.Imm.Int()))
			ctx.EmitSubInt64(scratch, d1177.Reg)
			d1192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1192)
		} else if d1177.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1191.Reg)
			ctx.EmitMovRegReg(scratch, d1191.Reg)
			if d1177.Imm.Int() >= -2147483648 && d1177.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d1177.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1177.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d1192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1192)
		} else {
			r148 := ctx.AllocRegExcept(d1191.Reg, d1177.Reg)
			ctx.EmitMovRegReg(r148, d1191.Reg)
			ctx.EmitSubInt64(r148, d1177.Reg)
			d1192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r148}
			ctx.BindReg(r148, &d1192)
		}
		if d1192.Loc == scm.LocReg && d1191.Loc == scm.LocReg && d1192.Reg == d1191.Reg {
			ctx.TransferReg(d1191.Reg)
			d1191.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1177)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1190)
		ctx.EnsureDesc(&d1192)
		ctx.EnsureDescsTogether(&d1190, &d1192)
		var d1193 scm.JITValueDesc
		if d1190.Loc == scm.LocImm && d1192.Loc == scm.LocImm {
			d1193 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1190.Imm.Int()) >> uint64(d1192.Imm.Int())))}
		} else if d1192.Loc == scm.LocImm {
			r149 := ctx.AllocRegExcept(d1190.Reg)
			ctx.EmitMovRegReg(r149, d1190.Reg)
			ctx.EmitShrRegImm8(r149, uint8(d1192.Imm.Int()))
			d1193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r149}
			ctx.BindReg(r149, &d1193)
		} else {
			{
				shiftSrc := d1190.Reg
				r150 := ctx.AllocRegExcept(d1190.Reg, d1192.Reg)
				ctx.EmitMovRegReg(r150, d1190.Reg)
				shiftSrc = r150
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d1192.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d1192.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d1192.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d1193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d1193)
			}
		}
		if d1193.Loc == scm.LocReg && d1190.Loc == scm.LocReg && d1193.Reg == d1190.Reg {
			ctx.TransferReg(d1190.Reg)
			d1190.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1190)
		ctx.FreeDesc(&d1192)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1193)
		ctx.EnsureDesc(&d1193)
		ctx.EnsureDesc(&d1193)
		var d1194 scm.JITValueDesc
		if d1193.Loc == scm.LocImm {
			d1194 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d1193.Imm.Int()))))}
		} else {
			r151 := ctx.AllocReg()
			ctx.EmitMovRegReg(r151, d1193.Reg)
			d1194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
			ctx.BindReg(r151, &d1194)
		}
		ctx.FreeDesc(&d1193)
		var d1195 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d1195 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56)
			r152 := ctx.AllocReg()
			ctx.EmitMovRegMem(r152, thisptr.Reg, off)
			d1195 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r152}
			ctx.BindReg(r152, &d1195)
		}
		ctx.EnsureDesc(&d1194)
		ctx.EnsureDesc(&d1195)
		ctx.EnsureDescsTogether(&d1194, &d1195)
		var d1196 scm.JITValueDesc
		if d1194.Loc == scm.LocImm && d1195.Loc == scm.LocImm {
			d1196 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1194.Imm.Int() + d1195.Imm.Int())}
		} else if d1195.Loc == scm.LocImm && d1195.Imm.Int() == 0 {
			r153 := ctx.AllocRegExcept(d1194.Reg)
			ctx.EmitMovRegReg(r153, d1194.Reg)
			d1196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r153}
			ctx.BindReg(r153, &d1196)
		} else if d1194.Loc == scm.LocImm && d1194.Imm.Int() == 0 {
			d1196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1195.Reg}
			ctx.BindReg(d1195.Reg, &d1196)
		} else if d1194.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1195.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1194.Imm.Int()))
			ctx.EmitAddInt64(scratch, d1195.Reg)
			d1196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1196)
		} else if d1195.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1194.Reg)
			ctx.EmitMovRegReg(scratch, d1194.Reg)
			if d1195.Imm.Int() >= -2147483648 && d1195.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d1195.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1195.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d1196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1196)
		} else {
			r154 := ctx.AllocRegExcept(d1194.Reg, d1195.Reg)
			ctx.EmitMovRegReg(r154, d1194.Reg)
			ctx.EmitAddInt64(r154, d1195.Reg)
			d1196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r154}
			ctx.BindReg(r154, &d1196)
		}
		if d1196.Loc == scm.LocReg && d1194.Loc == scm.LocReg && d1196.Reg == d1194.Reg {
			ctx.TransferReg(d1194.Reg)
			d1194.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1194)
		ctx.FreeDesc(&d1195)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d1196)
		ctx.EnsureDescsTogether(&idxInt, &d1196)
		var d1198 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm && d1196.Loc == scm.LocImm {
			d1198 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(idxInt.Imm.Int() - d1196.Imm.Int())}
		} else if d1196.Loc == scm.LocImm && d1196.Imm.Int() == 0 {
			r155 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitMovRegReg(r155, idxInt.Reg)
			d1198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r155}
			ctx.BindReg(r155, &d1198)
		} else if idxInt.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1196.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(idxInt.Imm.Int()))
			ctx.EmitSubInt64(scratch, d1196.Reg)
			d1198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1198)
		} else if d1196.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitMovRegReg(scratch, idxInt.Reg)
			if d1196.Imm.Int() >= -2147483648 && d1196.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d1196.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1196.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d1198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1198)
		} else {
			r156 := ctx.AllocRegExcept(idxInt.Reg, d1196.Reg)
			ctx.EmitMovRegReg(r156, idxInt.Reg)
			ctx.EmitSubInt64(r156, d1196.Reg)
			d1198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r156}
			ctx.BindReg(r156, &d1198)
		}
		if d1198.Loc == scm.LocReg && idxInt.Loc == scm.LocReg && d1198.Reg == idxInt.Reg {
			ctx.TransferReg(idxInt.Reg)
			idxInt.Loc = scm.LocNone
		}
		ctx.FreeDesc(&idxInt)
		ctx.FreeDesc(&d1196)
		ctx.EnsureDesc(&d1198)
		ctx.EnsureDesc(&d1174)
		ctx.EnsureDescsTogether(&d1198, &d1174)
		var d1199 scm.JITValueDesc
		if d1198.Loc == scm.LocImm && d1174.Loc == scm.LocImm {
			d1199 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1198.Imm.Int() * d1174.Imm.Int())}
		} else if d1198.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1174.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1198.Imm.Int()))
			ctx.EmitImulInt64(scratch, d1174.Reg)
			d1199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1199)
		} else if d1174.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1198.Reg)
			ctx.EmitMovRegReg(scratch, d1198.Reg)
			if d1174.Imm.Int() >= -2147483648 && d1174.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d1174.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1174.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d1199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1199)
		} else {
			r157 := ctx.AllocRegExcept(d1198.Reg, d1174.Reg)
			ctx.EmitMovRegReg(r157, d1198.Reg)
			ctx.EmitImulInt64(r157, d1174.Reg)
			d1199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r157}
			ctx.BindReg(r157, &d1199)
		}
		if d1199.Loc == scm.LocReg && d1198.Loc == scm.LocReg && d1199.Reg == d1198.Reg {
			ctx.TransferReg(d1198.Reg)
			d1198.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1198)
		ctx.FreeDesc(&d1174)
		ctx.EnsureDesc(&d1152)
		ctx.EnsureDesc(&d1199)
		ctx.EnsureDescsTogether(&d1152, &d1199)
		var d1200 scm.JITValueDesc
		if d1152.Loc == scm.LocImm && d1199.Loc == scm.LocImm {
			d1200 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1152.Imm.Int() + d1199.Imm.Int())}
		} else if d1199.Loc == scm.LocImm && d1199.Imm.Int() == 0 {
			r158 := ctx.AllocRegExcept(d1152.Reg)
			ctx.EmitMovRegReg(r158, d1152.Reg)
			d1200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r158}
			ctx.BindReg(r158, &d1200)
		} else if d1152.Loc == scm.LocImm && d1152.Imm.Int() == 0 {
			d1200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1199.Reg}
			ctx.BindReg(d1199.Reg, &d1200)
		} else if d1152.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1199.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1152.Imm.Int()))
			ctx.EmitAddInt64(scratch, d1199.Reg)
			d1200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1200)
		} else if d1199.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1152.Reg)
			ctx.EmitMovRegReg(scratch, d1152.Reg)
			if d1199.Imm.Int() >= -2147483648 && d1199.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d1199.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1199.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d1200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d1200)
		} else {
			r159 := ctx.AllocRegExcept(d1152.Reg, d1199.Reg)
			ctx.EmitMovRegReg(r159, d1152.Reg)
			ctx.EmitAddInt64(r159, d1199.Reg)
			d1200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
			ctx.BindReg(r159, &d1200)
		}
		if d1200.Loc == scm.LocReg && d1152.Loc == scm.LocReg && d1200.Reg == d1152.Reg {
			ctx.TransferReg(d1152.Reg)
			d1152.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1152)
		ctx.FreeDesc(&d1199)
		ctx.EnsureDesc(&d1200)
		ctx.EnsureDesc(&d1200)
		var d1201 scm.JITValueDesc
		if d1200.Loc == scm.LocImm {
			d1201 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d1200.Imm.Int()))}
		} else {
			var r160 scm.Reg
			r160 = d1200.Reg
			d1200.Loc = scm.LocNone
			ctx.EmitCvtInt64ToFloat64(scm.RegX0, r160)
			d1201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: r160}
			ctx.BindReg(r160, &d1201)
		}
		ctx.FreeDesc(&d1200)
		ctx.EnsureDesc(&d1201)
		d1202 = result
		ctx.EnsureDesc(&d1201)
		ctx.EmitMakeFloat(d1202, d1201)
		if d1201.Loc == scm.LocReg {
			ctx.FreeReg(d1201.Reg)
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
			ctx.FlushRegisterMoves()
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
		if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
			d184 = ps.OverlayValues[184]
		}
		if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != scm.LocNone {
			d335 = ps.OverlayValues[335]
		}
		if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != scm.LocNone {
			d336 = ps.OverlayValues[336]
		}
		if len(ps.OverlayValues) > 337 && ps.OverlayValues[337].Loc != scm.LocNone {
			d337 = ps.OverlayValues[337]
		}
		if len(ps.OverlayValues) > 338 && ps.OverlayValues[338].Loc != scm.LocNone {
			d338 = ps.OverlayValues[338]
		}
		if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != scm.LocNone {
			d340 = ps.OverlayValues[340]
		}
		if len(ps.OverlayValues) > 341 && ps.OverlayValues[341].Loc != scm.LocNone {
			d341 = ps.OverlayValues[341]
		}
		if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != scm.LocNone {
			d342 = ps.OverlayValues[342]
		}
		if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != scm.LocNone {
			d343 = ps.OverlayValues[343]
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
		if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != scm.LocNone {
			d351 = ps.OverlayValues[351]
		}
		if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != scm.LocNone {
			d352 = ps.OverlayValues[352]
		}
		if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != scm.LocNone {
			d353 = ps.OverlayValues[353]
		}
		if len(ps.OverlayValues) > 444 && ps.OverlayValues[444].Loc != scm.LocNone {
			d444 = ps.OverlayValues[444]
		}
		if len(ps.OverlayValues) > 445 && ps.OverlayValues[445].Loc != scm.LocNone {
			d445 = ps.OverlayValues[445]
		}
		if len(ps.OverlayValues) > 448 && ps.OverlayValues[448].Loc != scm.LocNone {
			d448 = ps.OverlayValues[448]
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
		if len(ps.OverlayValues) > 548 && ps.OverlayValues[548].Loc != scm.LocNone {
			d548 = ps.OverlayValues[548]
		}
		if len(ps.OverlayValues) > 549 && ps.OverlayValues[549].Loc != scm.LocNone {
			d549 = ps.OverlayValues[549]
		}
		if len(ps.OverlayValues) > 550 && ps.OverlayValues[550].Loc != scm.LocNone {
			d550 = ps.OverlayValues[550]
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
		if len(ps.OverlayValues) > 555 && ps.OverlayValues[555].Loc != scm.LocNone {
			d555 = ps.OverlayValues[555]
		}
		if len(ps.OverlayValues) > 556 && ps.OverlayValues[556].Loc != scm.LocNone {
			d556 = ps.OverlayValues[556]
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
		if len(ps.OverlayValues) > 562 && ps.OverlayValues[562].Loc != scm.LocNone {
			d562 = ps.OverlayValues[562]
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
		if len(ps.OverlayValues) > 829 && ps.OverlayValues[829].Loc != scm.LocNone {
			d829 = ps.OverlayValues[829]
		}
		if len(ps.OverlayValues) > 830 && ps.OverlayValues[830].Loc != scm.LocNone {
			d830 = ps.OverlayValues[830]
		}
		if len(ps.OverlayValues) > 831 && ps.OverlayValues[831].Loc != scm.LocNone {
			d831 = ps.OverlayValues[831]
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
		if len(ps.OverlayValues) > 838 && ps.OverlayValues[838].Loc != scm.LocNone {
			d838 = ps.OverlayValues[838]
		}
		if len(ps.OverlayValues) > 839 && ps.OverlayValues[839].Loc != scm.LocNone {
			d839 = ps.OverlayValues[839]
		}
		if len(ps.OverlayValues) > 841 && ps.OverlayValues[841].Loc != scm.LocNone {
			d841 = ps.OverlayValues[841]
		}
		if len(ps.OverlayValues) > 843 && ps.OverlayValues[843].Loc != scm.LocNone {
			d843 = ps.OverlayValues[843]
		}
		if len(ps.OverlayValues) > 844 && ps.OverlayValues[844].Loc != scm.LocNone {
			d844 = ps.OverlayValues[844]
		}
		if len(ps.OverlayValues) > 983 && ps.OverlayValues[983].Loc != scm.LocNone {
			d983 = ps.OverlayValues[983]
		}
		if len(ps.OverlayValues) > 984 && ps.OverlayValues[984].Loc != scm.LocNone {
			d984 = ps.OverlayValues[984]
		}
		if len(ps.OverlayValues) > 987 && ps.OverlayValues[987].Loc != scm.LocNone {
			d987 = ps.OverlayValues[987]
		}
		if len(ps.OverlayValues) > 1129 && ps.OverlayValues[1129].Loc != scm.LocNone {
			d1129 = ps.OverlayValues[1129]
		}
		if len(ps.OverlayValues) > 1130 && ps.OverlayValues[1130].Loc != scm.LocNone {
			d1130 = ps.OverlayValues[1130]
		}
		if len(ps.OverlayValues) > 1131 && ps.OverlayValues[1131].Loc != scm.LocNone {
			d1131 = ps.OverlayValues[1131]
		}
		if len(ps.OverlayValues) > 1132 && ps.OverlayValues[1132].Loc != scm.LocNone {
			d1132 = ps.OverlayValues[1132]
		}
		if len(ps.OverlayValues) > 1134 && ps.OverlayValues[1134].Loc != scm.LocNone {
			d1134 = ps.OverlayValues[1134]
		}
		if len(ps.OverlayValues) > 1135 && ps.OverlayValues[1135].Loc != scm.LocNone {
			d1135 = ps.OverlayValues[1135]
		}
		if len(ps.OverlayValues) > 1136 && ps.OverlayValues[1136].Loc != scm.LocNone {
			d1136 = ps.OverlayValues[1136]
		}
		if len(ps.OverlayValues) > 1137 && ps.OverlayValues[1137].Loc != scm.LocNone {
			d1137 = ps.OverlayValues[1137]
		}
		if len(ps.OverlayValues) > 1138 && ps.OverlayValues[1138].Loc != scm.LocNone {
			d1138 = ps.OverlayValues[1138]
		}
		if len(ps.OverlayValues) > 1139 && ps.OverlayValues[1139].Loc != scm.LocNone {
			d1139 = ps.OverlayValues[1139]
		}
		if len(ps.OverlayValues) > 1140 && ps.OverlayValues[1140].Loc != scm.LocNone {
			d1140 = ps.OverlayValues[1140]
		}
		if len(ps.OverlayValues) > 1141 && ps.OverlayValues[1141].Loc != scm.LocNone {
			d1141 = ps.OverlayValues[1141]
		}
		if len(ps.OverlayValues) > 1142 && ps.OverlayValues[1142].Loc != scm.LocNone {
			d1142 = ps.OverlayValues[1142]
		}
		if len(ps.OverlayValues) > 1143 && ps.OverlayValues[1143].Loc != scm.LocNone {
			d1143 = ps.OverlayValues[1143]
		}
		if len(ps.OverlayValues) > 1145 && ps.OverlayValues[1145].Loc != scm.LocNone {
			d1145 = ps.OverlayValues[1145]
		}
		if len(ps.OverlayValues) > 1146 && ps.OverlayValues[1146].Loc != scm.LocNone {
			d1146 = ps.OverlayValues[1146]
		}
		if len(ps.OverlayValues) > 1147 && ps.OverlayValues[1147].Loc != scm.LocNone {
			d1147 = ps.OverlayValues[1147]
		}
		if len(ps.OverlayValues) > 1148 && ps.OverlayValues[1148].Loc != scm.LocNone {
			d1148 = ps.OverlayValues[1148]
		}
		if len(ps.OverlayValues) > 1149 && ps.OverlayValues[1149].Loc != scm.LocNone {
			d1149 = ps.OverlayValues[1149]
		}
		if len(ps.OverlayValues) > 1150 && ps.OverlayValues[1150].Loc != scm.LocNone {
			d1150 = ps.OverlayValues[1150]
		}
		if len(ps.OverlayValues) > 1151 && ps.OverlayValues[1151].Loc != scm.LocNone {
			d1151 = ps.OverlayValues[1151]
		}
		if len(ps.OverlayValues) > 1152 && ps.OverlayValues[1152].Loc != scm.LocNone {
			d1152 = ps.OverlayValues[1152]
		}
		if len(ps.OverlayValues) > 1153 && ps.OverlayValues[1153].Loc != scm.LocNone {
			d1153 = ps.OverlayValues[1153]
		}
		if len(ps.OverlayValues) > 1154 && ps.OverlayValues[1154].Loc != scm.LocNone {
			d1154 = ps.OverlayValues[1154]
		}
		if len(ps.OverlayValues) > 1155 && ps.OverlayValues[1155].Loc != scm.LocNone {
			d1155 = ps.OverlayValues[1155]
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
		if len(ps.OverlayValues) > 1160 && ps.OverlayValues[1160].Loc != scm.LocNone {
			d1160 = ps.OverlayValues[1160]
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
		if len(ps.OverlayValues) > 1171 && ps.OverlayValues[1171].Loc != scm.LocNone {
			d1171 = ps.OverlayValues[1171]
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
		ctx.ReclaimUntrackedRegs()
		var d1203 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 88
			val := *(*uint64)(unsafe.Pointer(fieldAddr))
			d1203 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 88)
			r161 := ctx.AllocReg()
			ctx.EmitMovRegMem(r161, thisptr.Reg, off)
			d1203 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r161}
			ctx.BindReg(r161, &d1203)
		}
		ctx.EnsureDesc(&d179)
		ctx.EnsureDesc(&d1203)
		ctx.EnsureDescsTogether(&d179, &d1203)
		var d1204 scm.JITValueDesc
		if d179.Loc == scm.LocImm && d1203.Loc == scm.LocImm {
			d1204 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d179.Imm.Int()) == uint64(d1203.Imm.Int()))}
		} else if d1203.Loc == scm.LocImm {
			r162 := ctx.AllocRegExcept(d179.Reg)
			if d1203.Imm.Int() >= -2147483648 && d1203.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d179.Reg, int32(d1203.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1203.Imm.Int()))
				ctx.EmitCmpInt64(d179.Reg, scm.RegR11)
			}
			d1204 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r162, Condition: scm.CondEqual}
			ctx.BindReg(r162, &d1204)
		} else if d179.Loc == scm.LocImm {
			r163 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d179.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d1203.Reg)
			d1204 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r163, Condition: scm.CondEqual}
			ctx.BindReg(r163, &d1204)
		} else {
			r164 := ctx.AllocRegExcept(d179.Reg)
			ctx.EmitCmpInt64(d179.Reg, d1203.Reg)
			d1204 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r164, Condition: scm.CondEqual}
			ctx.BindReg(r164, &d1204)
		}
		ctx.FreeDesc(&d1203)
		d1205 = d1204
		ctx.EnsureDesc(&d1205)
		if d1205.Loc != scm.LocImm && d1205.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d1205.Loc == scm.LocImm {
			if d1205.Imm.Bool() {
				if ps.General {
				}
				ps1206 := scm.PhiState{General: ps.General}
				ps1206.OverlayValues = make([]scm.JITValueDesc, 1206)
				ps1206.OverlayValues[5] = d5
				ps1206.OverlayValues[6] = d6
				ps1206.OverlayValues[7] = d7
				ps1206.OverlayValues[8] = d8
				ps1206.OverlayValues[9] = d9
				ps1206.OverlayValues[10] = d10
				ps1206.OverlayValues[11] = d11
				ps1206.OverlayValues[12] = d12
				ps1206.OverlayValues[13] = d13
				ps1206.OverlayValues[14] = d14
				ps1206.OverlayValues[15] = d15
				ps1206.OverlayValues[16] = d16
				ps1206.OverlayValues[17] = d17
				ps1206.OverlayValues[18] = d18
				ps1206.OverlayValues[19] = d19
				ps1206.OverlayValues[21] = d21
				ps1206.OverlayValues[22] = d22
				ps1206.OverlayValues[23] = d23
				ps1206.OverlayValues[24] = d24
				ps1206.OverlayValues[25] = d25
				ps1206.OverlayValues[26] = d26
				ps1206.OverlayValues[27] = d27
				ps1206.OverlayValues[28] = d28
				ps1206.OverlayValues[29] = d29
				ps1206.OverlayValues[30] = d30
				ps1206.OverlayValues[31] = d31
				ps1206.OverlayValues[32] = d32
				ps1206.OverlayValues[33] = d33
				ps1206.OverlayValues[34] = d34
				ps1206.OverlayValues[35] = d35
				ps1206.OverlayValues[36] = d36
				ps1206.OverlayValues[37] = d37
				ps1206.OverlayValues[38] = d38
				ps1206.OverlayValues[39] = d39
				ps1206.OverlayValues[40] = d40
				ps1206.OverlayValues[41] = d41
				ps1206.OverlayValues[42] = d42
				ps1206.OverlayValues[43] = d43
				ps1206.OverlayValues[44] = d44
				ps1206.OverlayValues[45] = d45
				ps1206.OverlayValues[46] = d46
				ps1206.OverlayValues[47] = d47
				ps1206.OverlayValues[48] = d48
				ps1206.OverlayValues[49] = d49
				ps1206.OverlayValues[50] = d50
				ps1206.OverlayValues[51] = d51
				ps1206.OverlayValues[54] = d54
				ps1206.OverlayValues[55] = d55
				ps1206.OverlayValues[56] = d56
				ps1206.OverlayValues[159] = d159
				ps1206.OverlayValues[160] = d160
				ps1206.OverlayValues[161] = d161
				ps1206.OverlayValues[162] = d162
				ps1206.OverlayValues[163] = d163
				ps1206.OverlayValues[164] = d164
				ps1206.OverlayValues[165] = d165
				ps1206.OverlayValues[166] = d166
				ps1206.OverlayValues[167] = d167
				ps1206.OverlayValues[168] = d168
				ps1206.OverlayValues[169] = d169
				ps1206.OverlayValues[170] = d170
				ps1206.OverlayValues[171] = d171
				ps1206.OverlayValues[172] = d172
				ps1206.OverlayValues[173] = d173
				ps1206.OverlayValues[174] = d174
				ps1206.OverlayValues[175] = d175
				ps1206.OverlayValues[176] = d176
				ps1206.OverlayValues[177] = d177
				ps1206.OverlayValues[178] = d178
				ps1206.OverlayValues[179] = d179
				ps1206.OverlayValues[180] = d180
				ps1206.OverlayValues[181] = d181
				ps1206.OverlayValues[184] = d184
				ps1206.OverlayValues[335] = d335
				ps1206.OverlayValues[336] = d336
				ps1206.OverlayValues[337] = d337
				ps1206.OverlayValues[338] = d338
				ps1206.OverlayValues[340] = d340
				ps1206.OverlayValues[341] = d341
				ps1206.OverlayValues[342] = d342
				ps1206.OverlayValues[343] = d343
				ps1206.OverlayValues[344] = d344
				ps1206.OverlayValues[345] = d345
				ps1206.OverlayValues[346] = d346
				ps1206.OverlayValues[347] = d347
				ps1206.OverlayValues[349] = d349
				ps1206.OverlayValues[351] = d351
				ps1206.OverlayValues[352] = d352
				ps1206.OverlayValues[353] = d353
				ps1206.OverlayValues[444] = d444
				ps1206.OverlayValues[445] = d445
				ps1206.OverlayValues[448] = d448
				ps1206.OverlayValues[542] = d542
				ps1206.OverlayValues[543] = d543
				ps1206.OverlayValues[544] = d544
				ps1206.OverlayValues[545] = d545
				ps1206.OverlayValues[546] = d546
				ps1206.OverlayValues[548] = d548
				ps1206.OverlayValues[549] = d549
				ps1206.OverlayValues[550] = d550
				ps1206.OverlayValues[551] = d551
				ps1206.OverlayValues[552] = d552
				ps1206.OverlayValues[553] = d553
				ps1206.OverlayValues[554] = d554
				ps1206.OverlayValues[555] = d555
				ps1206.OverlayValues[556] = d556
				ps1206.OverlayValues[557] = d557
				ps1206.OverlayValues[558] = d558
				ps1206.OverlayValues[559] = d559
				ps1206.OverlayValues[560] = d560
				ps1206.OverlayValues[561] = d561
				ps1206.OverlayValues[562] = d562
				ps1206.OverlayValues[563] = d563
				ps1206.OverlayValues[564] = d564
				ps1206.OverlayValues[565] = d565
				ps1206.OverlayValues[566] = d566
				ps1206.OverlayValues[567] = d567
				ps1206.OverlayValues[568] = d568
				ps1206.OverlayValues[569] = d569
				ps1206.OverlayValues[570] = d570
				ps1206.OverlayValues[571] = d571
				ps1206.OverlayValues[572] = d572
				ps1206.OverlayValues[573] = d573
				ps1206.OverlayValues[574] = d574
				ps1206.OverlayValues[829] = d829
				ps1206.OverlayValues[830] = d830
				ps1206.OverlayValues[831] = d831
				ps1206.OverlayValues[833] = d833
				ps1206.OverlayValues[834] = d834
				ps1206.OverlayValues[835] = d835
				ps1206.OverlayValues[836] = d836
				ps1206.OverlayValues[837] = d837
				ps1206.OverlayValues[838] = d838
				ps1206.OverlayValues[839] = d839
				ps1206.OverlayValues[841] = d841
				ps1206.OverlayValues[843] = d843
				ps1206.OverlayValues[844] = d844
				ps1206.OverlayValues[983] = d983
				ps1206.OverlayValues[984] = d984
				ps1206.OverlayValues[987] = d987
				ps1206.OverlayValues[1129] = d1129
				ps1206.OverlayValues[1130] = d1130
				ps1206.OverlayValues[1131] = d1131
				ps1206.OverlayValues[1132] = d1132
				ps1206.OverlayValues[1134] = d1134
				ps1206.OverlayValues[1135] = d1135
				ps1206.OverlayValues[1136] = d1136
				ps1206.OverlayValues[1137] = d1137
				ps1206.OverlayValues[1138] = d1138
				ps1206.OverlayValues[1139] = d1139
				ps1206.OverlayValues[1140] = d1140
				ps1206.OverlayValues[1141] = d1141
				ps1206.OverlayValues[1142] = d1142
				ps1206.OverlayValues[1143] = d1143
				ps1206.OverlayValues[1145] = d1145
				ps1206.OverlayValues[1146] = d1146
				ps1206.OverlayValues[1147] = d1147
				ps1206.OverlayValues[1148] = d1148
				ps1206.OverlayValues[1149] = d1149
				ps1206.OverlayValues[1150] = d1150
				ps1206.OverlayValues[1151] = d1151
				ps1206.OverlayValues[1152] = d1152
				ps1206.OverlayValues[1153] = d1153
				ps1206.OverlayValues[1154] = d1154
				ps1206.OverlayValues[1155] = d1155
				ps1206.OverlayValues[1156] = d1156
				ps1206.OverlayValues[1157] = d1157
				ps1206.OverlayValues[1158] = d1158
				ps1206.OverlayValues[1159] = d1159
				ps1206.OverlayValues[1160] = d1160
				ps1206.OverlayValues[1161] = d1161
				ps1206.OverlayValues[1162] = d1162
				ps1206.OverlayValues[1163] = d1163
				ps1206.OverlayValues[1164] = d1164
				ps1206.OverlayValues[1165] = d1165
				ps1206.OverlayValues[1166] = d1166
				ps1206.OverlayValues[1167] = d1167
				ps1206.OverlayValues[1168] = d1168
				ps1206.OverlayValues[1169] = d1169
				ps1206.OverlayValues[1170] = d1170
				ps1206.OverlayValues[1171] = d1171
				ps1206.OverlayValues[1172] = d1172
				ps1206.OverlayValues[1173] = d1173
				ps1206.OverlayValues[1174] = d1174
				ps1206.OverlayValues[1175] = d1175
				ps1206.OverlayValues[1176] = d1176
				ps1206.OverlayValues[1177] = d1177
				ps1206.OverlayValues[1178] = d1178
				ps1206.OverlayValues[1179] = d1179
				ps1206.OverlayValues[1180] = d1180
				ps1206.OverlayValues[1181] = d1181
				ps1206.OverlayValues[1182] = d1182
				ps1206.OverlayValues[1183] = d1183
				ps1206.OverlayValues[1184] = d1184
				ps1206.OverlayValues[1185] = d1185
				ps1206.OverlayValues[1186] = d1186
				ps1206.OverlayValues[1187] = d1187
				ps1206.OverlayValues[1188] = d1188
				ps1206.OverlayValues[1189] = d1189
				ps1206.OverlayValues[1190] = d1190
				ps1206.OverlayValues[1191] = d1191
				ps1206.OverlayValues[1192] = d1192
				ps1206.OverlayValues[1193] = d1193
				ps1206.OverlayValues[1194] = d1194
				ps1206.OverlayValues[1195] = d1195
				ps1206.OverlayValues[1196] = d1196
				ps1206.OverlayValues[1197] = d1197
				ps1206.OverlayValues[1198] = d1198
				ps1206.OverlayValues[1199] = d1199
				ps1206.OverlayValues[1200] = d1200
				ps1206.OverlayValues[1201] = d1201
				ps1206.OverlayValues[1202] = d1202
				ps1206.OverlayValues[1203] = d1203
				ps1206.OverlayValues[1204] = d1204
				ps1206.OverlayValues[1205] = d1205
				return bbs[11].RenderPS(ps1206)
			}
			if ps.General {
			}
			ps1207 := scm.PhiState{General: ps.General}
			ps1207.OverlayValues = make([]scm.JITValueDesc, 1206)
			ps1207.OverlayValues[5] = d5
			ps1207.OverlayValues[6] = d6
			ps1207.OverlayValues[7] = d7
			ps1207.OverlayValues[8] = d8
			ps1207.OverlayValues[9] = d9
			ps1207.OverlayValues[10] = d10
			ps1207.OverlayValues[11] = d11
			ps1207.OverlayValues[12] = d12
			ps1207.OverlayValues[13] = d13
			ps1207.OverlayValues[14] = d14
			ps1207.OverlayValues[15] = d15
			ps1207.OverlayValues[16] = d16
			ps1207.OverlayValues[17] = d17
			ps1207.OverlayValues[18] = d18
			ps1207.OverlayValues[19] = d19
			ps1207.OverlayValues[21] = d21
			ps1207.OverlayValues[22] = d22
			ps1207.OverlayValues[23] = d23
			ps1207.OverlayValues[24] = d24
			ps1207.OverlayValues[25] = d25
			ps1207.OverlayValues[26] = d26
			ps1207.OverlayValues[27] = d27
			ps1207.OverlayValues[28] = d28
			ps1207.OverlayValues[29] = d29
			ps1207.OverlayValues[30] = d30
			ps1207.OverlayValues[31] = d31
			ps1207.OverlayValues[32] = d32
			ps1207.OverlayValues[33] = d33
			ps1207.OverlayValues[34] = d34
			ps1207.OverlayValues[35] = d35
			ps1207.OverlayValues[36] = d36
			ps1207.OverlayValues[37] = d37
			ps1207.OverlayValues[38] = d38
			ps1207.OverlayValues[39] = d39
			ps1207.OverlayValues[40] = d40
			ps1207.OverlayValues[41] = d41
			ps1207.OverlayValues[42] = d42
			ps1207.OverlayValues[43] = d43
			ps1207.OverlayValues[44] = d44
			ps1207.OverlayValues[45] = d45
			ps1207.OverlayValues[46] = d46
			ps1207.OverlayValues[47] = d47
			ps1207.OverlayValues[48] = d48
			ps1207.OverlayValues[49] = d49
			ps1207.OverlayValues[50] = d50
			ps1207.OverlayValues[51] = d51
			ps1207.OverlayValues[54] = d54
			ps1207.OverlayValues[55] = d55
			ps1207.OverlayValues[56] = d56
			ps1207.OverlayValues[159] = d159
			ps1207.OverlayValues[160] = d160
			ps1207.OverlayValues[161] = d161
			ps1207.OverlayValues[162] = d162
			ps1207.OverlayValues[163] = d163
			ps1207.OverlayValues[164] = d164
			ps1207.OverlayValues[165] = d165
			ps1207.OverlayValues[166] = d166
			ps1207.OverlayValues[167] = d167
			ps1207.OverlayValues[168] = d168
			ps1207.OverlayValues[169] = d169
			ps1207.OverlayValues[170] = d170
			ps1207.OverlayValues[171] = d171
			ps1207.OverlayValues[172] = d172
			ps1207.OverlayValues[173] = d173
			ps1207.OverlayValues[174] = d174
			ps1207.OverlayValues[175] = d175
			ps1207.OverlayValues[176] = d176
			ps1207.OverlayValues[177] = d177
			ps1207.OverlayValues[178] = d178
			ps1207.OverlayValues[179] = d179
			ps1207.OverlayValues[180] = d180
			ps1207.OverlayValues[181] = d181
			ps1207.OverlayValues[184] = d184
			ps1207.OverlayValues[335] = d335
			ps1207.OverlayValues[336] = d336
			ps1207.OverlayValues[337] = d337
			ps1207.OverlayValues[338] = d338
			ps1207.OverlayValues[340] = d340
			ps1207.OverlayValues[341] = d341
			ps1207.OverlayValues[342] = d342
			ps1207.OverlayValues[343] = d343
			ps1207.OverlayValues[344] = d344
			ps1207.OverlayValues[345] = d345
			ps1207.OverlayValues[346] = d346
			ps1207.OverlayValues[347] = d347
			ps1207.OverlayValues[349] = d349
			ps1207.OverlayValues[351] = d351
			ps1207.OverlayValues[352] = d352
			ps1207.OverlayValues[353] = d353
			ps1207.OverlayValues[444] = d444
			ps1207.OverlayValues[445] = d445
			ps1207.OverlayValues[448] = d448
			ps1207.OverlayValues[542] = d542
			ps1207.OverlayValues[543] = d543
			ps1207.OverlayValues[544] = d544
			ps1207.OverlayValues[545] = d545
			ps1207.OverlayValues[546] = d546
			ps1207.OverlayValues[548] = d548
			ps1207.OverlayValues[549] = d549
			ps1207.OverlayValues[550] = d550
			ps1207.OverlayValues[551] = d551
			ps1207.OverlayValues[552] = d552
			ps1207.OverlayValues[553] = d553
			ps1207.OverlayValues[554] = d554
			ps1207.OverlayValues[555] = d555
			ps1207.OverlayValues[556] = d556
			ps1207.OverlayValues[557] = d557
			ps1207.OverlayValues[558] = d558
			ps1207.OverlayValues[559] = d559
			ps1207.OverlayValues[560] = d560
			ps1207.OverlayValues[561] = d561
			ps1207.OverlayValues[562] = d562
			ps1207.OverlayValues[563] = d563
			ps1207.OverlayValues[564] = d564
			ps1207.OverlayValues[565] = d565
			ps1207.OverlayValues[566] = d566
			ps1207.OverlayValues[567] = d567
			ps1207.OverlayValues[568] = d568
			ps1207.OverlayValues[569] = d569
			ps1207.OverlayValues[570] = d570
			ps1207.OverlayValues[571] = d571
			ps1207.OverlayValues[572] = d572
			ps1207.OverlayValues[573] = d573
			ps1207.OverlayValues[574] = d574
			ps1207.OverlayValues[829] = d829
			ps1207.OverlayValues[830] = d830
			ps1207.OverlayValues[831] = d831
			ps1207.OverlayValues[833] = d833
			ps1207.OverlayValues[834] = d834
			ps1207.OverlayValues[835] = d835
			ps1207.OverlayValues[836] = d836
			ps1207.OverlayValues[837] = d837
			ps1207.OverlayValues[838] = d838
			ps1207.OverlayValues[839] = d839
			ps1207.OverlayValues[841] = d841
			ps1207.OverlayValues[843] = d843
			ps1207.OverlayValues[844] = d844
			ps1207.OverlayValues[983] = d983
			ps1207.OverlayValues[984] = d984
			ps1207.OverlayValues[987] = d987
			ps1207.OverlayValues[1129] = d1129
			ps1207.OverlayValues[1130] = d1130
			ps1207.OverlayValues[1131] = d1131
			ps1207.OverlayValues[1132] = d1132
			ps1207.OverlayValues[1134] = d1134
			ps1207.OverlayValues[1135] = d1135
			ps1207.OverlayValues[1136] = d1136
			ps1207.OverlayValues[1137] = d1137
			ps1207.OverlayValues[1138] = d1138
			ps1207.OverlayValues[1139] = d1139
			ps1207.OverlayValues[1140] = d1140
			ps1207.OverlayValues[1141] = d1141
			ps1207.OverlayValues[1142] = d1142
			ps1207.OverlayValues[1143] = d1143
			ps1207.OverlayValues[1145] = d1145
			ps1207.OverlayValues[1146] = d1146
			ps1207.OverlayValues[1147] = d1147
			ps1207.OverlayValues[1148] = d1148
			ps1207.OverlayValues[1149] = d1149
			ps1207.OverlayValues[1150] = d1150
			ps1207.OverlayValues[1151] = d1151
			ps1207.OverlayValues[1152] = d1152
			ps1207.OverlayValues[1153] = d1153
			ps1207.OverlayValues[1154] = d1154
			ps1207.OverlayValues[1155] = d1155
			ps1207.OverlayValues[1156] = d1156
			ps1207.OverlayValues[1157] = d1157
			ps1207.OverlayValues[1158] = d1158
			ps1207.OverlayValues[1159] = d1159
			ps1207.OverlayValues[1160] = d1160
			ps1207.OverlayValues[1161] = d1161
			ps1207.OverlayValues[1162] = d1162
			ps1207.OverlayValues[1163] = d1163
			ps1207.OverlayValues[1164] = d1164
			ps1207.OverlayValues[1165] = d1165
			ps1207.OverlayValues[1166] = d1166
			ps1207.OverlayValues[1167] = d1167
			ps1207.OverlayValues[1168] = d1168
			ps1207.OverlayValues[1169] = d1169
			ps1207.OverlayValues[1170] = d1170
			ps1207.OverlayValues[1171] = d1171
			ps1207.OverlayValues[1172] = d1172
			ps1207.OverlayValues[1173] = d1173
			ps1207.OverlayValues[1174] = d1174
			ps1207.OverlayValues[1175] = d1175
			ps1207.OverlayValues[1176] = d1176
			ps1207.OverlayValues[1177] = d1177
			ps1207.OverlayValues[1178] = d1178
			ps1207.OverlayValues[1179] = d1179
			ps1207.OverlayValues[1180] = d1180
			ps1207.OverlayValues[1181] = d1181
			ps1207.OverlayValues[1182] = d1182
			ps1207.OverlayValues[1183] = d1183
			ps1207.OverlayValues[1184] = d1184
			ps1207.OverlayValues[1185] = d1185
			ps1207.OverlayValues[1186] = d1186
			ps1207.OverlayValues[1187] = d1187
			ps1207.OverlayValues[1188] = d1188
			ps1207.OverlayValues[1189] = d1189
			ps1207.OverlayValues[1190] = d1190
			ps1207.OverlayValues[1191] = d1191
			ps1207.OverlayValues[1192] = d1192
			ps1207.OverlayValues[1193] = d1193
			ps1207.OverlayValues[1194] = d1194
			ps1207.OverlayValues[1195] = d1195
			ps1207.OverlayValues[1196] = d1196
			ps1207.OverlayValues[1197] = d1197
			ps1207.OverlayValues[1198] = d1198
			ps1207.OverlayValues[1199] = d1199
			ps1207.OverlayValues[1200] = d1200
			ps1207.OverlayValues[1201] = d1201
			ps1207.OverlayValues[1202] = d1202
			ps1207.OverlayValues[1203] = d1203
			ps1207.OverlayValues[1204] = d1204
			ps1207.OverlayValues[1205] = d1205
			return bbs[12].RenderPS(ps1207)
		}
		if !ps.General {
			ps.General = true
			return bbs[13].RenderPS(ps)
		}
		ctx.EmitJump(d1205.Condition, lbl12)
		if bbs[12].Rendered {
			ctx.EmitJmp(lbl13)
		}
		ctx.FreeDesc(&d1204)
		snap1208 := d5
		snap1209 := d6
		snap1210 := d7
		snap1211 := d8
		snap1212 := d9
		snap1213 := d10
		snap1214 := d11
		snap1215 := d12
		snap1216 := d13
		snap1217 := d14
		snap1218 := d15
		snap1219 := d16
		snap1220 := d17
		snap1221 := d18
		snap1222 := d19
		snap1223 := d21
		snap1224 := d22
		snap1225 := d23
		snap1226 := d24
		snap1227 := d25
		snap1228 := d26
		snap1229 := d27
		snap1230 := d28
		snap1231 := d29
		snap1232 := d30
		snap1233 := d31
		snap1234 := d32
		snap1235 := d33
		snap1236 := d34
		snap1237 := d35
		snap1238 := d36
		snap1239 := d37
		snap1240 := d38
		snap1241 := d39
		snap1242 := d40
		snap1243 := d41
		snap1244 := d42
		snap1245 := d43
		snap1246 := d44
		snap1247 := d45
		snap1248 := d46
		snap1249 := d47
		snap1250 := d48
		snap1251 := d49
		snap1252 := d50
		snap1253 := d51
		snap1254 := d54
		snap1255 := d55
		snap1256 := d56
		snap1257 := d159
		snap1258 := d160
		snap1259 := d161
		snap1260 := d162
		snap1261 := d163
		snap1262 := d164
		snap1263 := d165
		snap1264 := d166
		snap1265 := d167
		snap1266 := d168
		snap1267 := d169
		snap1268 := d170
		snap1269 := d171
		snap1270 := d172
		snap1271 := d173
		snap1272 := d174
		snap1273 := d175
		snap1274 := d176
		snap1275 := d177
		snap1276 := d178
		snap1277 := d179
		snap1278 := d180
		snap1279 := d181
		snap1280 := d184
		snap1281 := d335
		snap1282 := d336
		snap1283 := d337
		snap1284 := d338
		snap1285 := d340
		snap1286 := d341
		snap1287 := d342
		snap1288 := d343
		snap1289 := d344
		snap1290 := d345
		snap1291 := d346
		snap1292 := d347
		snap1293 := d349
		snap1294 := d351
		snap1295 := d352
		snap1296 := d353
		snap1297 := d444
		snap1298 := d445
		snap1299 := d448
		snap1300 := d542
		snap1301 := d543
		snap1302 := d544
		snap1303 := d545
		snap1304 := d546
		snap1305 := d548
		snap1306 := d549
		snap1307 := d550
		snap1308 := d551
		snap1309 := d552
		snap1310 := d553
		snap1311 := d554
		snap1312 := d555
		snap1313 := d556
		snap1314 := d557
		snap1315 := d558
		snap1316 := d559
		snap1317 := d560
		snap1318 := d561
		snap1319 := d562
		snap1320 := d563
		snap1321 := d564
		snap1322 := d565
		snap1323 := d566
		snap1324 := d567
		snap1325 := d568
		snap1326 := d569
		snap1327 := d570
		snap1328 := d571
		snap1329 := d572
		snap1330 := d573
		snap1331 := d574
		snap1332 := d829
		snap1333 := d830
		snap1334 := d831
		snap1335 := d833
		snap1336 := d834
		snap1337 := d835
		snap1338 := d836
		snap1339 := d837
		snap1340 := d838
		snap1341 := d839
		snap1342 := d841
		snap1343 := d843
		snap1344 := d844
		snap1345 := d983
		snap1346 := d984
		snap1347 := d987
		snap1348 := d1129
		snap1349 := d1130
		snap1350 := d1131
		snap1351 := d1132
		snap1352 := d1134
		snap1353 := d1135
		snap1354 := d1136
		snap1355 := d1137
		snap1356 := d1138
		snap1357 := d1139
		snap1358 := d1140
		snap1359 := d1141
		snap1360 := d1142
		snap1361 := d1143
		snap1362 := d1145
		snap1363 := d1146
		snap1364 := d1147
		snap1365 := d1148
		snap1366 := d1149
		snap1367 := d1150
		snap1368 := d1151
		snap1369 := d1152
		snap1370 := d1153
		snap1371 := d1154
		snap1372 := d1155
		snap1373 := d1156
		snap1374 := d1157
		snap1375 := d1158
		snap1376 := d1159
		snap1377 := d1160
		snap1378 := d1161
		snap1379 := d1162
		snap1380 := d1163
		snap1381 := d1164
		snap1382 := d1165
		snap1383 := d1166
		snap1384 := d1167
		snap1385 := d1168
		snap1386 := d1169
		snap1387 := d1170
		snap1388 := d1171
		snap1389 := d1172
		snap1390 := d1173
		snap1391 := d1174
		snap1392 := d1175
		snap1393 := d1176
		snap1394 := d1177
		snap1395 := d1178
		snap1396 := d1179
		snap1397 := d1180
		snap1398 := d1181
		snap1399 := d1182
		snap1400 := d1183
		snap1401 := d1184
		snap1402 := d1185
		snap1403 := d1186
		snap1404 := d1187
		snap1405 := d1188
		snap1406 := d1189
		snap1407 := d1190
		snap1408 := d1191
		snap1409 := d1192
		snap1410 := d1193
		snap1411 := d1194
		snap1412 := d1195
		snap1413 := d1196
		snap1414 := d1197
		snap1415 := d1198
		snap1416 := d1199
		snap1417 := d1200
		snap1418 := d1201
		snap1419 := d1202
		snap1420 := d1203
		snap1421 := d1204
		snap1422 := d1205
		alloc1423 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc1423)
		d5 = snap1208
		d6 = snap1209
		d7 = snap1210
		d8 = snap1211
		d9 = snap1212
		d10 = snap1213
		d11 = snap1214
		d12 = snap1215
		d13 = snap1216
		d14 = snap1217
		d15 = snap1218
		d16 = snap1219
		d17 = snap1220
		d18 = snap1221
		d19 = snap1222
		d21 = snap1223
		d22 = snap1224
		d23 = snap1225
		d24 = snap1226
		d25 = snap1227
		d26 = snap1228
		d27 = snap1229
		d28 = snap1230
		d29 = snap1231
		d30 = snap1232
		d31 = snap1233
		d32 = snap1234
		d33 = snap1235
		d34 = snap1236
		d35 = snap1237
		d36 = snap1238
		d37 = snap1239
		d38 = snap1240
		d39 = snap1241
		d40 = snap1242
		d41 = snap1243
		d42 = snap1244
		d43 = snap1245
		d44 = snap1246
		d45 = snap1247
		d46 = snap1248
		d47 = snap1249
		d48 = snap1250
		d49 = snap1251
		d50 = snap1252
		d51 = snap1253
		d54 = snap1254
		d55 = snap1255
		d56 = snap1256
		d159 = snap1257
		d160 = snap1258
		d161 = snap1259
		d162 = snap1260
		d163 = snap1261
		d164 = snap1262
		d165 = snap1263
		d166 = snap1264
		d167 = snap1265
		d168 = snap1266
		d169 = snap1267
		d170 = snap1268
		d171 = snap1269
		d172 = snap1270
		d173 = snap1271
		d174 = snap1272
		d175 = snap1273
		d176 = snap1274
		d177 = snap1275
		d178 = snap1276
		d179 = snap1277
		d180 = snap1278
		d181 = snap1279
		d184 = snap1280
		d335 = snap1281
		d336 = snap1282
		d337 = snap1283
		d338 = snap1284
		d340 = snap1285
		d341 = snap1286
		d342 = snap1287
		d343 = snap1288
		d344 = snap1289
		d345 = snap1290
		d346 = snap1291
		d347 = snap1292
		d349 = snap1293
		d351 = snap1294
		d352 = snap1295
		d353 = snap1296
		d444 = snap1297
		d445 = snap1298
		d448 = snap1299
		d542 = snap1300
		d543 = snap1301
		d544 = snap1302
		d545 = snap1303
		d546 = snap1304
		d548 = snap1305
		d549 = snap1306
		d550 = snap1307
		d551 = snap1308
		d552 = snap1309
		d553 = snap1310
		d554 = snap1311
		d555 = snap1312
		d556 = snap1313
		d557 = snap1314
		d558 = snap1315
		d559 = snap1316
		d560 = snap1317
		d561 = snap1318
		d562 = snap1319
		d563 = snap1320
		d564 = snap1321
		d565 = snap1322
		d566 = snap1323
		d567 = snap1324
		d568 = snap1325
		d569 = snap1326
		d570 = snap1327
		d571 = snap1328
		d572 = snap1329
		d573 = snap1330
		d574 = snap1331
		d829 = snap1332
		d830 = snap1333
		d831 = snap1334
		d833 = snap1335
		d834 = snap1336
		d835 = snap1337
		d836 = snap1338
		d837 = snap1339
		d838 = snap1340
		d839 = snap1341
		d841 = snap1342
		d843 = snap1343
		d844 = snap1344
		d983 = snap1345
		d984 = snap1346
		d987 = snap1347
		d1129 = snap1348
		d1130 = snap1349
		d1131 = snap1350
		d1132 = snap1351
		d1134 = snap1352
		d1135 = snap1353
		d1136 = snap1354
		d1137 = snap1355
		d1138 = snap1356
		d1139 = snap1357
		d1140 = snap1358
		d1141 = snap1359
		d1142 = snap1360
		d1143 = snap1361
		d1145 = snap1362
		d1146 = snap1363
		d1147 = snap1364
		d1148 = snap1365
		d1149 = snap1366
		d1150 = snap1367
		d1151 = snap1368
		d1152 = snap1369
		d1153 = snap1370
		d1154 = snap1371
		d1155 = snap1372
		d1156 = snap1373
		d1157 = snap1374
		d1158 = snap1375
		d1159 = snap1376
		d1160 = snap1377
		d1161 = snap1378
		d1162 = snap1379
		d1163 = snap1380
		d1164 = snap1381
		d1165 = snap1382
		d1166 = snap1383
		d1167 = snap1384
		d1168 = snap1385
		d1169 = snap1386
		d1170 = snap1387
		d1171 = snap1388
		d1172 = snap1389
		d1173 = snap1390
		d1174 = snap1391
		d1175 = snap1392
		d1176 = snap1393
		d1177 = snap1394
		d1178 = snap1395
		d1179 = snap1396
		d1180 = snap1397
		d1181 = snap1398
		d1182 = snap1399
		d1183 = snap1400
		d1184 = snap1401
		d1185 = snap1402
		d1186 = snap1403
		d1187 = snap1404
		d1188 = snap1405
		d1189 = snap1406
		d1190 = snap1407
		d1191 = snap1408
		d1192 = snap1409
		d1193 = snap1410
		d1194 = snap1411
		d1195 = snap1412
		d1196 = snap1413
		d1197 = snap1414
		d1198 = snap1415
		d1199 = snap1416
		d1200 = snap1417
		d1201 = snap1418
		d1202 = snap1419
		d1203 = snap1420
		d1204 = snap1421
		d1205 = snap1422
		ctx.RestoreAllocState(alloc1423)
		d5 = snap1208
		d6 = snap1209
		d7 = snap1210
		d8 = snap1211
		d9 = snap1212
		d10 = snap1213
		d11 = snap1214
		d12 = snap1215
		d13 = snap1216
		d14 = snap1217
		d15 = snap1218
		d16 = snap1219
		d17 = snap1220
		d18 = snap1221
		d19 = snap1222
		d21 = snap1223
		d22 = snap1224
		d23 = snap1225
		d24 = snap1226
		d25 = snap1227
		d26 = snap1228
		d27 = snap1229
		d28 = snap1230
		d29 = snap1231
		d30 = snap1232
		d31 = snap1233
		d32 = snap1234
		d33 = snap1235
		d34 = snap1236
		d35 = snap1237
		d36 = snap1238
		d37 = snap1239
		d38 = snap1240
		d39 = snap1241
		d40 = snap1242
		d41 = snap1243
		d42 = snap1244
		d43 = snap1245
		d44 = snap1246
		d45 = snap1247
		d46 = snap1248
		d47 = snap1249
		d48 = snap1250
		d49 = snap1251
		d50 = snap1252
		d51 = snap1253
		d54 = snap1254
		d55 = snap1255
		d56 = snap1256
		d159 = snap1257
		d160 = snap1258
		d161 = snap1259
		d162 = snap1260
		d163 = snap1261
		d164 = snap1262
		d165 = snap1263
		d166 = snap1264
		d167 = snap1265
		d168 = snap1266
		d169 = snap1267
		d170 = snap1268
		d171 = snap1269
		d172 = snap1270
		d173 = snap1271
		d174 = snap1272
		d175 = snap1273
		d176 = snap1274
		d177 = snap1275
		d178 = snap1276
		d179 = snap1277
		d180 = snap1278
		d181 = snap1279
		d184 = snap1280
		d335 = snap1281
		d336 = snap1282
		d337 = snap1283
		d338 = snap1284
		d340 = snap1285
		d341 = snap1286
		d342 = snap1287
		d343 = snap1288
		d344 = snap1289
		d345 = snap1290
		d346 = snap1291
		d347 = snap1292
		d349 = snap1293
		d351 = snap1294
		d352 = snap1295
		d353 = snap1296
		d444 = snap1297
		d445 = snap1298
		d448 = snap1299
		d542 = snap1300
		d543 = snap1301
		d544 = snap1302
		d545 = snap1303
		d546 = snap1304
		d548 = snap1305
		d549 = snap1306
		d550 = snap1307
		d551 = snap1308
		d552 = snap1309
		d553 = snap1310
		d554 = snap1311
		d555 = snap1312
		d556 = snap1313
		d557 = snap1314
		d558 = snap1315
		d559 = snap1316
		d560 = snap1317
		d561 = snap1318
		d562 = snap1319
		d563 = snap1320
		d564 = snap1321
		d565 = snap1322
		d566 = snap1323
		d567 = snap1324
		d568 = snap1325
		d569 = snap1326
		d570 = snap1327
		d571 = snap1328
		d572 = snap1329
		d573 = snap1330
		d574 = snap1331
		d829 = snap1332
		d830 = snap1333
		d831 = snap1334
		d833 = snap1335
		d834 = snap1336
		d835 = snap1337
		d836 = snap1338
		d837 = snap1339
		d838 = snap1340
		d839 = snap1341
		d841 = snap1342
		d843 = snap1343
		d844 = snap1344
		d983 = snap1345
		d984 = snap1346
		d987 = snap1347
		d1129 = snap1348
		d1130 = snap1349
		d1131 = snap1350
		d1132 = snap1351
		d1134 = snap1352
		d1135 = snap1353
		d1136 = snap1354
		d1137 = snap1355
		d1138 = snap1356
		d1139 = snap1357
		d1140 = snap1358
		d1141 = snap1359
		d1142 = snap1360
		d1143 = snap1361
		d1145 = snap1362
		d1146 = snap1363
		d1147 = snap1364
		d1148 = snap1365
		d1149 = snap1366
		d1150 = snap1367
		d1151 = snap1368
		d1152 = snap1369
		d1153 = snap1370
		d1154 = snap1371
		d1155 = snap1372
		d1156 = snap1373
		d1157 = snap1374
		d1158 = snap1375
		d1159 = snap1376
		d1160 = snap1377
		d1161 = snap1378
		d1162 = snap1379
		d1163 = snap1380
		d1164 = snap1381
		d1165 = snap1382
		d1166 = snap1383
		d1167 = snap1384
		d1168 = snap1385
		d1169 = snap1386
		d1170 = snap1387
		d1171 = snap1388
		d1172 = snap1389
		d1173 = snap1390
		d1174 = snap1391
		d1175 = snap1392
		d1176 = snap1393
		d1177 = snap1394
		d1178 = snap1395
		d1179 = snap1396
		d1180 = snap1397
		d1181 = snap1398
		d1182 = snap1399
		d1183 = snap1400
		d1184 = snap1401
		d1185 = snap1402
		d1186 = snap1403
		d1187 = snap1404
		d1188 = snap1405
		d1189 = snap1406
		d1190 = snap1407
		d1191 = snap1408
		d1192 = snap1409
		d1193 = snap1410
		d1194 = snap1411
		d1195 = snap1412
		d1196 = snap1413
		d1197 = snap1414
		d1198 = snap1415
		d1199 = snap1416
		d1200 = snap1417
		d1201 = snap1418
		d1202 = snap1419
		d1203 = snap1420
		d1204 = snap1421
		d1205 = snap1422
		ps1424 := scm.PhiState{General: true}
		ps1424.OverlayValues = make([]scm.JITValueDesc, 1206)
		ps1424.OverlayValues[5] = d5
		ps1424.OverlayValues[6] = d6
		ps1424.OverlayValues[7] = d7
		ps1424.OverlayValues[8] = d8
		ps1424.OverlayValues[9] = d9
		ps1424.OverlayValues[10] = d10
		ps1424.OverlayValues[11] = d11
		ps1424.OverlayValues[12] = d12
		ps1424.OverlayValues[13] = d13
		ps1424.OverlayValues[14] = d14
		ps1424.OverlayValues[15] = d15
		ps1424.OverlayValues[16] = d16
		ps1424.OverlayValues[17] = d17
		ps1424.OverlayValues[18] = d18
		ps1424.OverlayValues[19] = d19
		ps1424.OverlayValues[21] = d21
		ps1424.OverlayValues[22] = d22
		ps1424.OverlayValues[23] = d23
		ps1424.OverlayValues[24] = d24
		ps1424.OverlayValues[25] = d25
		ps1424.OverlayValues[26] = d26
		ps1424.OverlayValues[27] = d27
		ps1424.OverlayValues[28] = d28
		ps1424.OverlayValues[29] = d29
		ps1424.OverlayValues[30] = d30
		ps1424.OverlayValues[31] = d31
		ps1424.OverlayValues[32] = d32
		ps1424.OverlayValues[33] = d33
		ps1424.OverlayValues[34] = d34
		ps1424.OverlayValues[35] = d35
		ps1424.OverlayValues[36] = d36
		ps1424.OverlayValues[37] = d37
		ps1424.OverlayValues[38] = d38
		ps1424.OverlayValues[39] = d39
		ps1424.OverlayValues[40] = d40
		ps1424.OverlayValues[41] = d41
		ps1424.OverlayValues[42] = d42
		ps1424.OverlayValues[43] = d43
		ps1424.OverlayValues[44] = d44
		ps1424.OverlayValues[45] = d45
		ps1424.OverlayValues[46] = d46
		ps1424.OverlayValues[47] = d47
		ps1424.OverlayValues[48] = d48
		ps1424.OverlayValues[49] = d49
		ps1424.OverlayValues[50] = d50
		ps1424.OverlayValues[51] = d51
		ps1424.OverlayValues[54] = d54
		ps1424.OverlayValues[55] = d55
		ps1424.OverlayValues[56] = d56
		ps1424.OverlayValues[159] = d159
		ps1424.OverlayValues[160] = d160
		ps1424.OverlayValues[161] = d161
		ps1424.OverlayValues[162] = d162
		ps1424.OverlayValues[163] = d163
		ps1424.OverlayValues[164] = d164
		ps1424.OverlayValues[165] = d165
		ps1424.OverlayValues[166] = d166
		ps1424.OverlayValues[167] = d167
		ps1424.OverlayValues[168] = d168
		ps1424.OverlayValues[169] = d169
		ps1424.OverlayValues[170] = d170
		ps1424.OverlayValues[171] = d171
		ps1424.OverlayValues[172] = d172
		ps1424.OverlayValues[173] = d173
		ps1424.OverlayValues[174] = d174
		ps1424.OverlayValues[175] = d175
		ps1424.OverlayValues[176] = d176
		ps1424.OverlayValues[177] = d177
		ps1424.OverlayValues[178] = d178
		ps1424.OverlayValues[179] = d179
		ps1424.OverlayValues[180] = d180
		ps1424.OverlayValues[181] = d181
		ps1424.OverlayValues[184] = d184
		ps1424.OverlayValues[335] = d335
		ps1424.OverlayValues[336] = d336
		ps1424.OverlayValues[337] = d337
		ps1424.OverlayValues[338] = d338
		ps1424.OverlayValues[340] = d340
		ps1424.OverlayValues[341] = d341
		ps1424.OverlayValues[342] = d342
		ps1424.OverlayValues[343] = d343
		ps1424.OverlayValues[344] = d344
		ps1424.OverlayValues[345] = d345
		ps1424.OverlayValues[346] = d346
		ps1424.OverlayValues[347] = d347
		ps1424.OverlayValues[349] = d349
		ps1424.OverlayValues[351] = d351
		ps1424.OverlayValues[352] = d352
		ps1424.OverlayValues[353] = d353
		ps1424.OverlayValues[444] = d444
		ps1424.OverlayValues[445] = d445
		ps1424.OverlayValues[448] = d448
		ps1424.OverlayValues[542] = d542
		ps1424.OverlayValues[543] = d543
		ps1424.OverlayValues[544] = d544
		ps1424.OverlayValues[545] = d545
		ps1424.OverlayValues[546] = d546
		ps1424.OverlayValues[548] = d548
		ps1424.OverlayValues[549] = d549
		ps1424.OverlayValues[550] = d550
		ps1424.OverlayValues[551] = d551
		ps1424.OverlayValues[552] = d552
		ps1424.OverlayValues[553] = d553
		ps1424.OverlayValues[554] = d554
		ps1424.OverlayValues[555] = d555
		ps1424.OverlayValues[556] = d556
		ps1424.OverlayValues[557] = d557
		ps1424.OverlayValues[558] = d558
		ps1424.OverlayValues[559] = d559
		ps1424.OverlayValues[560] = d560
		ps1424.OverlayValues[561] = d561
		ps1424.OverlayValues[562] = d562
		ps1424.OverlayValues[563] = d563
		ps1424.OverlayValues[564] = d564
		ps1424.OverlayValues[565] = d565
		ps1424.OverlayValues[566] = d566
		ps1424.OverlayValues[567] = d567
		ps1424.OverlayValues[568] = d568
		ps1424.OverlayValues[569] = d569
		ps1424.OverlayValues[570] = d570
		ps1424.OverlayValues[571] = d571
		ps1424.OverlayValues[572] = d572
		ps1424.OverlayValues[573] = d573
		ps1424.OverlayValues[574] = d574
		ps1424.OverlayValues[829] = d829
		ps1424.OverlayValues[830] = d830
		ps1424.OverlayValues[831] = d831
		ps1424.OverlayValues[833] = d833
		ps1424.OverlayValues[834] = d834
		ps1424.OverlayValues[835] = d835
		ps1424.OverlayValues[836] = d836
		ps1424.OverlayValues[837] = d837
		ps1424.OverlayValues[838] = d838
		ps1424.OverlayValues[839] = d839
		ps1424.OverlayValues[841] = d841
		ps1424.OverlayValues[843] = d843
		ps1424.OverlayValues[844] = d844
		ps1424.OverlayValues[983] = d983
		ps1424.OverlayValues[984] = d984
		ps1424.OverlayValues[987] = d987
		ps1424.OverlayValues[1129] = d1129
		ps1424.OverlayValues[1130] = d1130
		ps1424.OverlayValues[1131] = d1131
		ps1424.OverlayValues[1132] = d1132
		ps1424.OverlayValues[1134] = d1134
		ps1424.OverlayValues[1135] = d1135
		ps1424.OverlayValues[1136] = d1136
		ps1424.OverlayValues[1137] = d1137
		ps1424.OverlayValues[1138] = d1138
		ps1424.OverlayValues[1139] = d1139
		ps1424.OverlayValues[1140] = d1140
		ps1424.OverlayValues[1141] = d1141
		ps1424.OverlayValues[1142] = d1142
		ps1424.OverlayValues[1143] = d1143
		ps1424.OverlayValues[1145] = d1145
		ps1424.OverlayValues[1146] = d1146
		ps1424.OverlayValues[1147] = d1147
		ps1424.OverlayValues[1148] = d1148
		ps1424.OverlayValues[1149] = d1149
		ps1424.OverlayValues[1150] = d1150
		ps1424.OverlayValues[1151] = d1151
		ps1424.OverlayValues[1152] = d1152
		ps1424.OverlayValues[1153] = d1153
		ps1424.OverlayValues[1154] = d1154
		ps1424.OverlayValues[1155] = d1155
		ps1424.OverlayValues[1156] = d1156
		ps1424.OverlayValues[1157] = d1157
		ps1424.OverlayValues[1158] = d1158
		ps1424.OverlayValues[1159] = d1159
		ps1424.OverlayValues[1160] = d1160
		ps1424.OverlayValues[1161] = d1161
		ps1424.OverlayValues[1162] = d1162
		ps1424.OverlayValues[1163] = d1163
		ps1424.OverlayValues[1164] = d1164
		ps1424.OverlayValues[1165] = d1165
		ps1424.OverlayValues[1166] = d1166
		ps1424.OverlayValues[1167] = d1167
		ps1424.OverlayValues[1168] = d1168
		ps1424.OverlayValues[1169] = d1169
		ps1424.OverlayValues[1170] = d1170
		ps1424.OverlayValues[1171] = d1171
		ps1424.OverlayValues[1172] = d1172
		ps1424.OverlayValues[1173] = d1173
		ps1424.OverlayValues[1174] = d1174
		ps1424.OverlayValues[1175] = d1175
		ps1424.OverlayValues[1176] = d1176
		ps1424.OverlayValues[1177] = d1177
		ps1424.OverlayValues[1178] = d1178
		ps1424.OverlayValues[1179] = d1179
		ps1424.OverlayValues[1180] = d1180
		ps1424.OverlayValues[1181] = d1181
		ps1424.OverlayValues[1182] = d1182
		ps1424.OverlayValues[1183] = d1183
		ps1424.OverlayValues[1184] = d1184
		ps1424.OverlayValues[1185] = d1185
		ps1424.OverlayValues[1186] = d1186
		ps1424.OverlayValues[1187] = d1187
		ps1424.OverlayValues[1188] = d1188
		ps1424.OverlayValues[1189] = d1189
		ps1424.OverlayValues[1190] = d1190
		ps1424.OverlayValues[1191] = d1191
		ps1424.OverlayValues[1192] = d1192
		ps1424.OverlayValues[1193] = d1193
		ps1424.OverlayValues[1194] = d1194
		ps1424.OverlayValues[1195] = d1195
		ps1424.OverlayValues[1196] = d1196
		ps1424.OverlayValues[1197] = d1197
		ps1424.OverlayValues[1198] = d1198
		ps1424.OverlayValues[1199] = d1199
		ps1424.OverlayValues[1200] = d1200
		ps1424.OverlayValues[1201] = d1201
		ps1424.OverlayValues[1202] = d1202
		ps1424.OverlayValues[1203] = d1203
		ps1424.OverlayValues[1204] = d1204
		ps1424.OverlayValues[1205] = d1205
		ps1425 := scm.PhiState{General: true}
		ps1425.OverlayValues = make([]scm.JITValueDesc, 1206)
		ps1425.OverlayValues[5] = d5
		ps1425.OverlayValues[6] = d6
		ps1425.OverlayValues[7] = d7
		ps1425.OverlayValues[8] = d8
		ps1425.OverlayValues[9] = d9
		ps1425.OverlayValues[10] = d10
		ps1425.OverlayValues[11] = d11
		ps1425.OverlayValues[12] = d12
		ps1425.OverlayValues[13] = d13
		ps1425.OverlayValues[14] = d14
		ps1425.OverlayValues[15] = d15
		ps1425.OverlayValues[16] = d16
		ps1425.OverlayValues[17] = d17
		ps1425.OverlayValues[18] = d18
		ps1425.OverlayValues[19] = d19
		ps1425.OverlayValues[21] = d21
		ps1425.OverlayValues[22] = d22
		ps1425.OverlayValues[23] = d23
		ps1425.OverlayValues[24] = d24
		ps1425.OverlayValues[25] = d25
		ps1425.OverlayValues[26] = d26
		ps1425.OverlayValues[27] = d27
		ps1425.OverlayValues[28] = d28
		ps1425.OverlayValues[29] = d29
		ps1425.OverlayValues[30] = d30
		ps1425.OverlayValues[31] = d31
		ps1425.OverlayValues[32] = d32
		ps1425.OverlayValues[33] = d33
		ps1425.OverlayValues[34] = d34
		ps1425.OverlayValues[35] = d35
		ps1425.OverlayValues[36] = d36
		ps1425.OverlayValues[37] = d37
		ps1425.OverlayValues[38] = d38
		ps1425.OverlayValues[39] = d39
		ps1425.OverlayValues[40] = d40
		ps1425.OverlayValues[41] = d41
		ps1425.OverlayValues[42] = d42
		ps1425.OverlayValues[43] = d43
		ps1425.OverlayValues[44] = d44
		ps1425.OverlayValues[45] = d45
		ps1425.OverlayValues[46] = d46
		ps1425.OverlayValues[47] = d47
		ps1425.OverlayValues[48] = d48
		ps1425.OverlayValues[49] = d49
		ps1425.OverlayValues[50] = d50
		ps1425.OverlayValues[51] = d51
		ps1425.OverlayValues[54] = d54
		ps1425.OverlayValues[55] = d55
		ps1425.OverlayValues[56] = d56
		ps1425.OverlayValues[159] = d159
		ps1425.OverlayValues[160] = d160
		ps1425.OverlayValues[161] = d161
		ps1425.OverlayValues[162] = d162
		ps1425.OverlayValues[163] = d163
		ps1425.OverlayValues[164] = d164
		ps1425.OverlayValues[165] = d165
		ps1425.OverlayValues[166] = d166
		ps1425.OverlayValues[167] = d167
		ps1425.OverlayValues[168] = d168
		ps1425.OverlayValues[169] = d169
		ps1425.OverlayValues[170] = d170
		ps1425.OverlayValues[171] = d171
		ps1425.OverlayValues[172] = d172
		ps1425.OverlayValues[173] = d173
		ps1425.OverlayValues[174] = d174
		ps1425.OverlayValues[175] = d175
		ps1425.OverlayValues[176] = d176
		ps1425.OverlayValues[177] = d177
		ps1425.OverlayValues[178] = d178
		ps1425.OverlayValues[179] = d179
		ps1425.OverlayValues[180] = d180
		ps1425.OverlayValues[181] = d181
		ps1425.OverlayValues[184] = d184
		ps1425.OverlayValues[335] = d335
		ps1425.OverlayValues[336] = d336
		ps1425.OverlayValues[337] = d337
		ps1425.OverlayValues[338] = d338
		ps1425.OverlayValues[340] = d340
		ps1425.OverlayValues[341] = d341
		ps1425.OverlayValues[342] = d342
		ps1425.OverlayValues[343] = d343
		ps1425.OverlayValues[344] = d344
		ps1425.OverlayValues[345] = d345
		ps1425.OverlayValues[346] = d346
		ps1425.OverlayValues[347] = d347
		ps1425.OverlayValues[349] = d349
		ps1425.OverlayValues[351] = d351
		ps1425.OverlayValues[352] = d352
		ps1425.OverlayValues[353] = d353
		ps1425.OverlayValues[444] = d444
		ps1425.OverlayValues[445] = d445
		ps1425.OverlayValues[448] = d448
		ps1425.OverlayValues[542] = d542
		ps1425.OverlayValues[543] = d543
		ps1425.OverlayValues[544] = d544
		ps1425.OverlayValues[545] = d545
		ps1425.OverlayValues[546] = d546
		ps1425.OverlayValues[548] = d548
		ps1425.OverlayValues[549] = d549
		ps1425.OverlayValues[550] = d550
		ps1425.OverlayValues[551] = d551
		ps1425.OverlayValues[552] = d552
		ps1425.OverlayValues[553] = d553
		ps1425.OverlayValues[554] = d554
		ps1425.OverlayValues[555] = d555
		ps1425.OverlayValues[556] = d556
		ps1425.OverlayValues[557] = d557
		ps1425.OverlayValues[558] = d558
		ps1425.OverlayValues[559] = d559
		ps1425.OverlayValues[560] = d560
		ps1425.OverlayValues[561] = d561
		ps1425.OverlayValues[562] = d562
		ps1425.OverlayValues[563] = d563
		ps1425.OverlayValues[564] = d564
		ps1425.OverlayValues[565] = d565
		ps1425.OverlayValues[566] = d566
		ps1425.OverlayValues[567] = d567
		ps1425.OverlayValues[568] = d568
		ps1425.OverlayValues[569] = d569
		ps1425.OverlayValues[570] = d570
		ps1425.OverlayValues[571] = d571
		ps1425.OverlayValues[572] = d572
		ps1425.OverlayValues[573] = d573
		ps1425.OverlayValues[574] = d574
		ps1425.OverlayValues[829] = d829
		ps1425.OverlayValues[830] = d830
		ps1425.OverlayValues[831] = d831
		ps1425.OverlayValues[833] = d833
		ps1425.OverlayValues[834] = d834
		ps1425.OverlayValues[835] = d835
		ps1425.OverlayValues[836] = d836
		ps1425.OverlayValues[837] = d837
		ps1425.OverlayValues[838] = d838
		ps1425.OverlayValues[839] = d839
		ps1425.OverlayValues[841] = d841
		ps1425.OverlayValues[843] = d843
		ps1425.OverlayValues[844] = d844
		ps1425.OverlayValues[983] = d983
		ps1425.OverlayValues[984] = d984
		ps1425.OverlayValues[987] = d987
		ps1425.OverlayValues[1129] = d1129
		ps1425.OverlayValues[1130] = d1130
		ps1425.OverlayValues[1131] = d1131
		ps1425.OverlayValues[1132] = d1132
		ps1425.OverlayValues[1134] = d1134
		ps1425.OverlayValues[1135] = d1135
		ps1425.OverlayValues[1136] = d1136
		ps1425.OverlayValues[1137] = d1137
		ps1425.OverlayValues[1138] = d1138
		ps1425.OverlayValues[1139] = d1139
		ps1425.OverlayValues[1140] = d1140
		ps1425.OverlayValues[1141] = d1141
		ps1425.OverlayValues[1142] = d1142
		ps1425.OverlayValues[1143] = d1143
		ps1425.OverlayValues[1145] = d1145
		ps1425.OverlayValues[1146] = d1146
		ps1425.OverlayValues[1147] = d1147
		ps1425.OverlayValues[1148] = d1148
		ps1425.OverlayValues[1149] = d1149
		ps1425.OverlayValues[1150] = d1150
		ps1425.OverlayValues[1151] = d1151
		ps1425.OverlayValues[1152] = d1152
		ps1425.OverlayValues[1153] = d1153
		ps1425.OverlayValues[1154] = d1154
		ps1425.OverlayValues[1155] = d1155
		ps1425.OverlayValues[1156] = d1156
		ps1425.OverlayValues[1157] = d1157
		ps1425.OverlayValues[1158] = d1158
		ps1425.OverlayValues[1159] = d1159
		ps1425.OverlayValues[1160] = d1160
		ps1425.OverlayValues[1161] = d1161
		ps1425.OverlayValues[1162] = d1162
		ps1425.OverlayValues[1163] = d1163
		ps1425.OverlayValues[1164] = d1164
		ps1425.OverlayValues[1165] = d1165
		ps1425.OverlayValues[1166] = d1166
		ps1425.OverlayValues[1167] = d1167
		ps1425.OverlayValues[1168] = d1168
		ps1425.OverlayValues[1169] = d1169
		ps1425.OverlayValues[1170] = d1170
		ps1425.OverlayValues[1171] = d1171
		ps1425.OverlayValues[1172] = d1172
		ps1425.OverlayValues[1173] = d1173
		ps1425.OverlayValues[1174] = d1174
		ps1425.OverlayValues[1175] = d1175
		ps1425.OverlayValues[1176] = d1176
		ps1425.OverlayValues[1177] = d1177
		ps1425.OverlayValues[1178] = d1178
		ps1425.OverlayValues[1179] = d1179
		ps1425.OverlayValues[1180] = d1180
		ps1425.OverlayValues[1181] = d1181
		ps1425.OverlayValues[1182] = d1182
		ps1425.OverlayValues[1183] = d1183
		ps1425.OverlayValues[1184] = d1184
		ps1425.OverlayValues[1185] = d1185
		ps1425.OverlayValues[1186] = d1186
		ps1425.OverlayValues[1187] = d1187
		ps1425.OverlayValues[1188] = d1188
		ps1425.OverlayValues[1189] = d1189
		ps1425.OverlayValues[1190] = d1190
		ps1425.OverlayValues[1191] = d1191
		ps1425.OverlayValues[1192] = d1192
		ps1425.OverlayValues[1193] = d1193
		ps1425.OverlayValues[1194] = d1194
		ps1425.OverlayValues[1195] = d1195
		ps1425.OverlayValues[1196] = d1196
		ps1425.OverlayValues[1197] = d1197
		ps1425.OverlayValues[1198] = d1198
		ps1425.OverlayValues[1199] = d1199
		ps1425.OverlayValues[1200] = d1200
		ps1425.OverlayValues[1201] = d1201
		ps1425.OverlayValues[1202] = d1202
		ps1425.OverlayValues[1203] = d1203
		ps1425.OverlayValues[1204] = d1204
		ps1425.OverlayValues[1205] = d1205
		snap1426 := d5
		snap1427 := d6
		snap1428 := d7
		snap1429 := d8
		snap1430 := d9
		snap1431 := d10
		snap1432 := d11
		snap1433 := d12
		snap1434 := d13
		snap1435 := d14
		snap1436 := d15
		snap1437 := d16
		snap1438 := d17
		snap1439 := d18
		snap1440 := d19
		snap1441 := d21
		snap1442 := d22
		snap1443 := d23
		snap1444 := d24
		snap1445 := d25
		snap1446 := d26
		snap1447 := d27
		snap1448 := d28
		snap1449 := d29
		snap1450 := d30
		snap1451 := d31
		snap1452 := d32
		snap1453 := d33
		snap1454 := d34
		snap1455 := d35
		snap1456 := d36
		snap1457 := d37
		snap1458 := d38
		snap1459 := d39
		snap1460 := d40
		snap1461 := d41
		snap1462 := d42
		snap1463 := d43
		snap1464 := d44
		snap1465 := d45
		snap1466 := d46
		snap1467 := d47
		snap1468 := d48
		snap1469 := d49
		snap1470 := d50
		snap1471 := d51
		snap1472 := d54
		snap1473 := d55
		snap1474 := d56
		snap1475 := d159
		snap1476 := d160
		snap1477 := d161
		snap1478 := d162
		snap1479 := d163
		snap1480 := d164
		snap1481 := d165
		snap1482 := d166
		snap1483 := d167
		snap1484 := d168
		snap1485 := d169
		snap1486 := d170
		snap1487 := d171
		snap1488 := d172
		snap1489 := d173
		snap1490 := d174
		snap1491 := d175
		snap1492 := d176
		snap1493 := d177
		snap1494 := d178
		snap1495 := d179
		snap1496 := d180
		snap1497 := d181
		snap1498 := d184
		snap1499 := d335
		snap1500 := d336
		snap1501 := d337
		snap1502 := d338
		snap1503 := d340
		snap1504 := d341
		snap1505 := d342
		snap1506 := d343
		snap1507 := d344
		snap1508 := d345
		snap1509 := d346
		snap1510 := d347
		snap1511 := d349
		snap1512 := d351
		snap1513 := d352
		snap1514 := d353
		snap1515 := d444
		snap1516 := d445
		snap1517 := d448
		snap1518 := d542
		snap1519 := d543
		snap1520 := d544
		snap1521 := d545
		snap1522 := d546
		snap1523 := d548
		snap1524 := d549
		snap1525 := d550
		snap1526 := d551
		snap1527 := d552
		snap1528 := d553
		snap1529 := d554
		snap1530 := d555
		snap1531 := d556
		snap1532 := d557
		snap1533 := d558
		snap1534 := d559
		snap1535 := d560
		snap1536 := d561
		snap1537 := d562
		snap1538 := d563
		snap1539 := d564
		snap1540 := d565
		snap1541 := d566
		snap1542 := d567
		snap1543 := d568
		snap1544 := d569
		snap1545 := d570
		snap1546 := d571
		snap1547 := d572
		snap1548 := d573
		snap1549 := d574
		snap1550 := d829
		snap1551 := d830
		snap1552 := d831
		snap1553 := d833
		snap1554 := d834
		snap1555 := d835
		snap1556 := d836
		snap1557 := d837
		snap1558 := d838
		snap1559 := d839
		snap1560 := d841
		snap1561 := d843
		snap1562 := d844
		snap1563 := d983
		snap1564 := d984
		snap1565 := d987
		snap1566 := d1129
		snap1567 := d1130
		snap1568 := d1131
		snap1569 := d1132
		snap1570 := d1134
		snap1571 := d1135
		snap1572 := d1136
		snap1573 := d1137
		snap1574 := d1138
		snap1575 := d1139
		snap1576 := d1140
		snap1577 := d1141
		snap1578 := d1142
		snap1579 := d1143
		snap1580 := d1145
		snap1581 := d1146
		snap1582 := d1147
		snap1583 := d1148
		snap1584 := d1149
		snap1585 := d1150
		snap1586 := d1151
		snap1587 := d1152
		snap1588 := d1153
		snap1589 := d1154
		snap1590 := d1155
		snap1591 := d1156
		snap1592 := d1157
		snap1593 := d1158
		snap1594 := d1159
		snap1595 := d1160
		snap1596 := d1161
		snap1597 := d1162
		snap1598 := d1163
		snap1599 := d1164
		snap1600 := d1165
		snap1601 := d1166
		snap1602 := d1167
		snap1603 := d1168
		snap1604 := d1169
		snap1605 := d1170
		snap1606 := d1171
		snap1607 := d1172
		snap1608 := d1173
		snap1609 := d1174
		snap1610 := d1175
		snap1611 := d1176
		snap1612 := d1177
		snap1613 := d1178
		snap1614 := d1179
		snap1615 := d1180
		snap1616 := d1181
		snap1617 := d1182
		snap1618 := d1183
		snap1619 := d1184
		snap1620 := d1185
		snap1621 := d1186
		snap1622 := d1187
		snap1623 := d1188
		snap1624 := d1189
		snap1625 := d1190
		snap1626 := d1191
		snap1627 := d1192
		snap1628 := d1193
		snap1629 := d1194
		snap1630 := d1195
		snap1631 := d1196
		snap1632 := d1197
		snap1633 := d1198
		snap1634 := d1199
		snap1635 := d1200
		snap1636 := d1201
		snap1637 := d1202
		snap1638 := d1203
		snap1639 := d1204
		snap1640 := d1205
		alloc1641 := ctx.SnapshotAllocState()
		if !bbs[12].Rendered {
			bbs[12].RenderPS(ps1425)
		}
		ctx.RestoreAllocState(alloc1641)
		d5 = snap1426
		d6 = snap1427
		d7 = snap1428
		d8 = snap1429
		d9 = snap1430
		d10 = snap1431
		d11 = snap1432
		d12 = snap1433
		d13 = snap1434
		d14 = snap1435
		d15 = snap1436
		d16 = snap1437
		d17 = snap1438
		d18 = snap1439
		d19 = snap1440
		d21 = snap1441
		d22 = snap1442
		d23 = snap1443
		d24 = snap1444
		d25 = snap1445
		d26 = snap1446
		d27 = snap1447
		d28 = snap1448
		d29 = snap1449
		d30 = snap1450
		d31 = snap1451
		d32 = snap1452
		d33 = snap1453
		d34 = snap1454
		d35 = snap1455
		d36 = snap1456
		d37 = snap1457
		d38 = snap1458
		d39 = snap1459
		d40 = snap1460
		d41 = snap1461
		d42 = snap1462
		d43 = snap1463
		d44 = snap1464
		d45 = snap1465
		d46 = snap1466
		d47 = snap1467
		d48 = snap1468
		d49 = snap1469
		d50 = snap1470
		d51 = snap1471
		d54 = snap1472
		d55 = snap1473
		d56 = snap1474
		d159 = snap1475
		d160 = snap1476
		d161 = snap1477
		d162 = snap1478
		d163 = snap1479
		d164 = snap1480
		d165 = snap1481
		d166 = snap1482
		d167 = snap1483
		d168 = snap1484
		d169 = snap1485
		d170 = snap1486
		d171 = snap1487
		d172 = snap1488
		d173 = snap1489
		d174 = snap1490
		d175 = snap1491
		d176 = snap1492
		d177 = snap1493
		d178 = snap1494
		d179 = snap1495
		d180 = snap1496
		d181 = snap1497
		d184 = snap1498
		d335 = snap1499
		d336 = snap1500
		d337 = snap1501
		d338 = snap1502
		d340 = snap1503
		d341 = snap1504
		d342 = snap1505
		d343 = snap1506
		d344 = snap1507
		d345 = snap1508
		d346 = snap1509
		d347 = snap1510
		d349 = snap1511
		d351 = snap1512
		d352 = snap1513
		d353 = snap1514
		d444 = snap1515
		d445 = snap1516
		d448 = snap1517
		d542 = snap1518
		d543 = snap1519
		d544 = snap1520
		d545 = snap1521
		d546 = snap1522
		d548 = snap1523
		d549 = snap1524
		d550 = snap1525
		d551 = snap1526
		d552 = snap1527
		d553 = snap1528
		d554 = snap1529
		d555 = snap1530
		d556 = snap1531
		d557 = snap1532
		d558 = snap1533
		d559 = snap1534
		d560 = snap1535
		d561 = snap1536
		d562 = snap1537
		d563 = snap1538
		d564 = snap1539
		d565 = snap1540
		d566 = snap1541
		d567 = snap1542
		d568 = snap1543
		d569 = snap1544
		d570 = snap1545
		d571 = snap1546
		d572 = snap1547
		d573 = snap1548
		d574 = snap1549
		d829 = snap1550
		d830 = snap1551
		d831 = snap1552
		d833 = snap1553
		d834 = snap1554
		d835 = snap1555
		d836 = snap1556
		d837 = snap1557
		d838 = snap1558
		d839 = snap1559
		d841 = snap1560
		d843 = snap1561
		d844 = snap1562
		d983 = snap1563
		d984 = snap1564
		d987 = snap1565
		d1129 = snap1566
		d1130 = snap1567
		d1131 = snap1568
		d1132 = snap1569
		d1134 = snap1570
		d1135 = snap1571
		d1136 = snap1572
		d1137 = snap1573
		d1138 = snap1574
		d1139 = snap1575
		d1140 = snap1576
		d1141 = snap1577
		d1142 = snap1578
		d1143 = snap1579
		d1145 = snap1580
		d1146 = snap1581
		d1147 = snap1582
		d1148 = snap1583
		d1149 = snap1584
		d1150 = snap1585
		d1151 = snap1586
		d1152 = snap1587
		d1153 = snap1588
		d1154 = snap1589
		d1155 = snap1590
		d1156 = snap1591
		d1157 = snap1592
		d1158 = snap1593
		d1159 = snap1594
		d1160 = snap1595
		d1161 = snap1596
		d1162 = snap1597
		d1163 = snap1598
		d1164 = snap1599
		d1165 = snap1600
		d1166 = snap1601
		d1167 = snap1602
		d1168 = snap1603
		d1169 = snap1604
		d1170 = snap1605
		d1171 = snap1606
		d1172 = snap1607
		d1173 = snap1608
		d1174 = snap1609
		d1175 = snap1610
		d1176 = snap1611
		d1177 = snap1612
		d1178 = snap1613
		d1179 = snap1614
		d1180 = snap1615
		d1181 = snap1616
		d1182 = snap1617
		d1183 = snap1618
		d1184 = snap1619
		d1185 = snap1620
		d1186 = snap1621
		d1187 = snap1622
		d1188 = snap1623
		d1189 = snap1624
		d1190 = snap1625
		d1191 = snap1626
		d1192 = snap1627
		d1193 = snap1628
		d1194 = snap1629
		d1195 = snap1630
		d1196 = snap1631
		d1197 = snap1632
		d1198 = snap1633
		d1199 = snap1634
		d1200 = snap1635
		d1201 = snap1636
		d1202 = snap1637
		d1203 = snap1638
		d1204 = snap1639
		d1205 = snap1640
		if !bbs[11].Rendered {
			return bbs[11].RenderPS(ps1424)
		}
		return result
		return result
	}
	ps1642 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps1642)
	ctx.MarkLabel(lbl0)
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
	startRaw := s.start.GetValueUInt(min)
	if s.start.hasNull && startRaw == s.start.null {
		return scm.NewNil()
	}
	value = int64(startRaw) + s.start.offset
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
//
//jitgen:control-flow-stable recid count target/1 stride
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
//
//jitgen:control-flow-stable recids/2 target/1 stride
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
		// Keep one explicit induction variable across the nested segment walk;
		// the range form introduces a second copied-element phi with no benefit.
		for k := 0; k < n; k++ {
			recid := recids[k]
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

	for k := 0; k < n; k++ {
		recid := recids[k]
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
		// A value after NULL must start a fresh numeric segment.
		if !s.lastValueNil && s.lastValueFirst {
			// learn stride from second value
			s.lastValueFirst = false
			s.lastStride = v - s.lastValue.Load()
			s.lastValue.Store(v)
			s.stride.scan(s.seqCount-1, scm.NewInt(s.lastStride))
		} else if !s.lastValueNil && i != 0 && v == s.lastValue.Load()+s.lastStride {
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
		// A value after NULL must start a fresh numeric segment.
		if !s.lastValueNil && s.lastValueFirst {
			// learn stride from second value
			s.lastValueFirst = false
			s.lastStride = v - s.lastValue.Load()
			s.lastValue.Store(v)
			s.stride.build(s.seqCount-1, scm.NewInt(s.lastStride))
		} else if !s.lastValueNil && i != 0 && v == s.lastValue.Load()+s.lastStride {
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
