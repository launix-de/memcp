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
import "unsafe"
import "strings"
import "encoding/binary"
import "github.com/launix-de/memcp/scm"

type StorageString struct {
	// StorageInt for dictionary entries
	values StorageInt
	// the dictionary: bitcompress all start+end markers; use one big string for all values that is sliced of from
	dictionary string
	starts     StorageInt
	lens       StorageInt
	nodict     bool `jit:"immutable-after-finish"` // disable values array

	// helpers
	sb         strings.Builder
	reverseMap map[string][3]uint
	count      uint
	allsize    int
	// prefix statistics
	prefixstat map[string]int
	laststr    string
}

func (s *StorageString) ComputeSize() uint {
	return s.values.ComputeSize() + 8 + uint(len(s.dictionary)) + 24 + s.starts.ComputeSize() + s.lens.ComputeSize() + 8*8
}

func (s *StorageString) String() string {
	if s.nodict {
		return fmt.Sprintf("string-buffer[%d bytes]", len(s.dictionary))
	} else {
		return fmt.Sprintf("string-dict[%d entries; %d bytes]", s.count, len(s.dictionary))
	}
}

func (s *StorageString) Serialize(f io.Writer) {
	binary.Write(f, binary.LittleEndian, uint8(20)) // 20 = StorageString
	var nodict uint8 = 0
	if s.nodict {
		nodict = 1
	}
	binary.Write(f, binary.LittleEndian, uint8(nodict))
	io.WriteString(f, "123456") // dummy
	if s.nodict {
		binary.Write(f, binary.LittleEndian, uint64(s.starts.count))
	} else {
		binary.Write(f, binary.LittleEndian, uint64(s.values.count))
	}
	s.values.Serialize(f)
	s.starts.Serialize(f)
	s.lens.Serialize(f)
	binary.Write(f, binary.LittleEndian, uint64(len(s.dictionary)))
	io.WriteString(f, s.dictionary)
}

func (s *StorageString) Deserialize(f io.Reader) uint {
	var nodict uint8
	binary.Read(f, binary.LittleEndian, &nodict)
	if nodict == 1 {
		s.nodict = true
	}
	var dummy [6]byte
	f.Read(dummy[:])
	var l uint64
	binary.Read(f, binary.LittleEndian, &l)
	s.values.DeserializeEx(f, true)
	s.count = s.starts.DeserializeEx(f, true)
	s.lens.DeserializeEx(f, true)
	var dictionarylength uint64
	binary.Read(f, binary.LittleEndian, &dictionarylength)
	if dictionarylength > 0 {
		rawdata := make([]byte, dictionarylength)
		f.Read(rawdata)
		s.dictionary = unsafe.String(&rawdata[0], dictionarylength)
	}
	return uint(l)
}

func (s *StorageString) GetCachedReader() ColumnReader { return s }

func (s *StorageString) GetValue(i uint32) scm.Scmer {
	if s.nodict {
		start := uint64(int64(s.starts.GetValueUInt(i)) + s.starts.offset)
		if s.starts.hasNull && start == s.starts.null {
			return scm.NewNil()
		}
		len_ := uint64(int64(s.lens.GetValueUInt(i)) + s.lens.offset)
		startIdx := int(start)
		endIdx := int(start + len_)
		return scm.NewString(s.dictionary[startIdx:endIdx])
	} else {
		idx := uint32(int64(s.values.GetValueUInt(i)) + s.values.offset)
		if s.values.hasNull && idx == uint32(s.values.null) {
			return scm.NewNil()
		}
		start := int64(s.starts.GetValueUInt(idx)) + s.starts.offset
		len_ := int64(s.lens.GetValueUInt(idx)) + s.lens.offset
		startIdx := int(start)
		endIdx := int(start + len_)
		return scm.NewString(s.dictionary[startIdx:endIdx])
	}
}
func (s *StorageString) JITEmit(ctx *scm.JITContext, thisptr scm.JITValueDesc, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
			var d0 scm.JITValueDesc
			_ = d0
			var d1 scm.JITValueDesc
			_ = d1
			var d7 scm.JITValueDesc
			_ = d7
			var r5 unsafe.Pointer
			_ = r5
			var d8 scm.JITValueDesc
			_ = d8
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
			var d64 scm.JITValueDesc
			_ = d64
			var d65 scm.JITValueDesc
			_ = d65
			var d66 scm.JITValueDesc
			_ = d66
			var d67 scm.JITValueDesc
			_ = d67
			var d68 scm.JITValueDesc
			_ = d68
			var d69 scm.JITValueDesc
			_ = d69
			var d70 scm.JITValueDesc
			_ = d70
			var d71 scm.JITValueDesc
			_ = d71
			var d72 scm.JITValueDesc
			_ = d72
			var d73 scm.JITValueDesc
			_ = d73
			var d74 scm.JITValueDesc
			_ = d74
			var d75 scm.JITValueDesc
			_ = d75
			var d76 scm.JITValueDesc
			_ = d76
			var d77 scm.JITValueDesc
			_ = d77
			var d78 scm.JITValueDesc
			_ = d78
			var d79 scm.JITValueDesc
			_ = d79
			var d80 scm.JITValueDesc
			_ = d80
			var d81 scm.JITValueDesc
			_ = d81
			var d82 scm.JITValueDesc
			_ = d82
			var d83 scm.JITValueDesc
			_ = d83
			var d84 scm.JITValueDesc
			_ = d84
			var d85 scm.JITValueDesc
			_ = d85
			var d86 scm.JITValueDesc
			_ = d86
			var d87 scm.JITValueDesc
			_ = d87
			var d88 scm.JITValueDesc
			_ = d88
			var d89 scm.JITValueDesc
			_ = d89
			var d90 scm.JITValueDesc
			_ = d90
			var d91 scm.JITValueDesc
			_ = d91
			var d92 scm.JITValueDesc
			_ = d92
			var d93 scm.JITValueDesc
			_ = d93
			var d94 scm.JITValueDesc
			_ = d94
			var d95 scm.JITValueDesc
			_ = d95
			var d96 scm.JITValueDesc
			_ = d96
			var d97 scm.JITValueDesc
			_ = d97
			var d98 scm.JITValueDesc
			_ = d98
			var d99 scm.JITValueDesc
			_ = d99
			var d100 scm.JITValueDesc
			_ = d100
			var d107 scm.JITValueDesc
			_ = d107
			var d108 scm.JITValueDesc
			_ = d108
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
			var d207 scm.JITValueDesc
			_ = d207
			var d208 scm.JITValueDesc
			_ = d208
			var d209 scm.JITValueDesc
			_ = d209
			var d210 scm.JITValueDesc
			_ = d210
			var d211 scm.JITValueDesc
			_ = d211
			var d212 scm.JITValueDesc
			_ = d212
			var d213 scm.JITValueDesc
			_ = d213
			var d214 scm.JITValueDesc
			_ = d214
			var d215 scm.JITValueDesc
			_ = d215
			var d216 scm.JITValueDesc
			_ = d216
			var d217 scm.JITValueDesc
			_ = d217
			var d218 scm.JITValueDesc
			_ = d218
			var d219 scm.JITValueDesc
			_ = d219
			var d220 scm.JITValueDesc
			_ = d220
			var d221 scm.JITValueDesc
			_ = d221
			var d222 scm.JITValueDesc
			_ = d222
			var d223 scm.JITValueDesc
			_ = d223
			var d224 scm.JITValueDesc
			_ = d224
			var d225 scm.JITValueDesc
			_ = d225
			var d226 scm.JITValueDesc
			_ = d226
			var d227 scm.JITValueDesc
			_ = d227
			var d228 scm.JITValueDesc
			_ = d228
			var d229 scm.JITValueDesc
			_ = d229
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
			var d238 scm.JITValueDesc
			_ = d238
			var d239 scm.JITValueDesc
			_ = d239
			var d240 scm.JITValueDesc
			_ = d240
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
			var d246 scm.JITValueDesc
			_ = d246
			var d247 scm.JITValueDesc
			_ = d247
			var d248 scm.JITValueDesc
			_ = d248
			var d249 scm.JITValueDesc
			_ = d249
			var d250 scm.JITValueDesc
			_ = d250
			var d251 scm.JITValueDesc
			_ = d251
			var d252 scm.JITValueDesc
			_ = d252
			var d253 scm.JITValueDesc
			_ = d253
			var d254 scm.JITValueDesc
			_ = d254
			var d255 scm.JITValueDesc
			_ = d255
			var d256 scm.JITValueDesc
			_ = d256
			var d257 scm.JITValueDesc
			_ = d257
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
				ctx.W.EmitShlRegImm8(idxInt.Reg, 32)
				ctx.W.EmitShrRegImm8(idxInt.Reg, 32)
				ctx.BindReg(idxInt.Reg, &idxInt)
			}
			idxPinned := idxInt.Loc == scm.LocReg
			idxPinnedReg := idxInt.Reg
			if idxPinned { ctx.ProtectReg(idxPinnedReg) }
			var bbs [9]scm.BBDescriptor
			if result.Loc == scm.LocAny {
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			r0 := ctx.AllocReg()
			r1 := ctx.AllocRegExcept(r0)
			lbl0 := ctx.W.ReserveLabel()
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl1 := ctx.W.ReserveLabel()
			bbpos_0_1 := int32(-1)
			_ = bbpos_0_1
			lbl2 := ctx.W.ReserveLabel()
			bbpos_0_2 := int32(-1)
			_ = bbpos_0_2
			lbl3 := ctx.W.ReserveLabel()
			bbpos_0_3 := int32(-1)
			_ = bbpos_0_3
			lbl4 := ctx.W.ReserveLabel()
			bbpos_0_4 := int32(-1)
			_ = bbpos_0_4
			lbl5 := ctx.W.ReserveLabel()
			bbpos_0_5 := int32(-1)
			_ = bbpos_0_5
			lbl6 := ctx.W.ReserveLabel()
			bbpos_0_6 := int32(-1)
			_ = bbpos_0_6
			lbl7 := ctx.W.ReserveLabel()
			bbpos_0_7 := int32(-1)
			_ = bbpos_0_7
			lbl8 := ctx.W.ReserveLabel()
			bbpos_0_8 := int32(-1)
			_ = bbpos_0_8
			lbl9 := ctx.W.ReserveLabel()
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
					ctx.W.EmitJmp(lbl1)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.W.MarkLabel(lbl1)
				ctx.W.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			var d0 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).nodict)
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d0 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).nodict))
				r2 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r2, thisptr.Reg, off)
				d0 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r2}
				ctx.BindReg(r2, &d0)
			}
			d1 = d0
			ctx.EnsureDesc(&d1)
			if d1.Loc != scm.LocImm && d1.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d1.Loc == scm.LocImm {
				if d1.Imm.Bool() {
			ps2 := scm.PhiState{General: ps.General}
			ps2.OverlayValues = make([]scm.JITValueDesc, 2)
			ps2.OverlayValues[0] = d0
			ps2.OverlayValues[1] = d1
					return bbs[1].RenderPS(ps2)
				}
			ps3 := scm.PhiState{General: ps.General}
			ps3.OverlayValues = make([]scm.JITValueDesc, 2)
			ps3.OverlayValues[0] = d0
			ps3.OverlayValues[1] = d1
				return bbs[2].RenderPS(ps3)
			}
			lbl10 := ctx.W.ReserveLabel()
			lbl11 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d1.Reg, 0)
			ctx.W.EmitJcc(scm.CcNE, lbl10)
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl10)
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl11)
			ctx.W.EmitJmp(lbl3)
			ps4 := scm.PhiState{General: true}
			ps4.OverlayValues = make([]scm.JITValueDesc, 2)
			ps4.OverlayValues[0] = d0
			ps4.OverlayValues[1] = d1
			ps5 := scm.PhiState{General: true}
			ps5.OverlayValues = make([]scm.JITValueDesc, 2)
			ps5.OverlayValues[0] = d0
			ps5.OverlayValues[1] = d1
			alloc6 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps5)
			}
			ctx.RestoreAllocState(alloc6)
			if !bbs[1].Rendered {
				return bbs[1].RenderPS(ps4)
			}
			return result
			ctx.FreeDesc(&d0)
			return result
			}
			bbs[1].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[1].VisitCount >= 2 {
					ps.General = true
					return bbs[1].RenderPS(ps)
				}
			}
			bbs[1].VisitCount++
			if ps.General {
				if bbs[1].Rendered {
					ctx.W.EmitJmp(lbl2)
					return result
				}
				bbs[1].Rendered = true
				bbs[1].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_1 = bbs[1].Address
				ctx.W.MarkLabel(lbl2)
				ctx.W.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&idxInt)
			d7 = idxInt
			_ = d7
			r3 := idxInt.Loc == scm.LocReg
			r4 := idxInt.Reg
			if r3 { ctx.ProtectReg(r4) }
			r5 = ctx.W.EmitSubRSP32Fixup()
			_ = r5
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			lbl12 := ctx.W.ReserveLabel()
			bbpos_1_0 := int32(-1)
			_ = bbpos_1_0
			bbpos_1_1 := int32(-1)
			_ = bbpos_1_1
			bbpos_1_2 := int32(-1)
			_ = bbpos_1_2
			bbpos_1_3 := int32(-1)
			_ = bbpos_1_3
			bbpos_1_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d7)
			var d9 scm.JITValueDesc
			if d7.Loc == scm.LocImm {
				d9 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d7.Imm.Int()))))}
			} else {
				r6 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r6, d7.Reg)
				ctx.W.EmitShlRegImm8(r6, 32)
				ctx.W.EmitShrRegImm8(r6, 32)
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r6}
				ctx.BindReg(r6, &d9)
			}
			var d10 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r7 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r7, thisptr.Reg, off)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r7}
				ctx.BindReg(r7, &d10)
			}
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d10)
			var d11 scm.JITValueDesc
			if d10.Loc == scm.LocImm {
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d10.Imm.Int()))))}
			} else {
				r8 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r8, d10.Reg)
				ctx.W.EmitShlRegImm8(r8, 56)
				ctx.W.EmitShrRegImm8(r8, 56)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
				ctx.BindReg(r8, &d11)
			}
			ctx.FreeDesc(&d10)
			ctx.EnsureDesc(&d9)
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d9)
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d9)
			ctx.EnsureDesc(&d11)
			var d12 scm.JITValueDesc
			if d9.Loc == scm.LocImm && d11.Loc == scm.LocImm {
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d9.Imm.Int() * d11.Imm.Int())}
			} else if d9.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d9.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d11.Reg)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d12)
			} else if d11.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(scratch, d9.Reg)
				if d11.Imm.Int() >= -2147483648 && d11.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d11.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d11.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d12)
			} else {
				r9 := ctx.AllocRegExcept(d9.Reg, d11.Reg)
				ctx.W.EmitMovRegReg(r9, d9.Reg)
				ctx.W.EmitImulInt64(r9, d11.Reg)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
				ctx.BindReg(r9, &d12)
			}
			if d12.Loc == scm.LocReg && d9.Loc == scm.LocReg && d12.Reg == d9.Reg {
				ctx.TransferReg(d9.Reg)
				d9.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d9)
			ctx.FreeDesc(&d11)
			var d13 scm.JITValueDesc
			r10 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r10, uint64(dataPtr))
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r10, StackOff: int32(sliceLen)}
				ctx.BindReg(r10, &d13)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 0)
				ctx.W.EmitMovRegMem(r10, thisptr.Reg, off)
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r10}
				ctx.BindReg(r10, &d13)
			}
			ctx.BindReg(r10, &d13)
			ctx.EnsureDesc(&d12)
			var d14 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() / 64)}
			} else {
				r11 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r11, d12.Reg)
				ctx.W.EmitShrRegImm8(r11, 6)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
				ctx.BindReg(r11, &d14)
			}
			if d14.Loc == scm.LocReg && d12.Loc == scm.LocReg && d14.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d14)
			r12 := ctx.AllocReg()
			ctx.EnsureDesc(&d14)
			ctx.EnsureDesc(&d13)
			if d14.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r12, uint64(d14.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r12, d14.Reg)
				ctx.W.EmitShlRegImm8(r12, 3)
			}
			if d13.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
				ctx.W.EmitAddInt64(r12, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r12, d13.Reg)
			}
			r13 := ctx.AllocRegExcept(r12)
			ctx.W.EmitMovRegMem(r13, r12, 0)
			ctx.FreeReg(r12)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r13}
			ctx.BindReg(r13, &d15)
			ctx.FreeDesc(&d14)
			ctx.EnsureDesc(&d12)
			var d16 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() % 64)}
			} else {
				r14 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r14, d12.Reg)
				ctx.W.EmitAndRegImm32(r14, 63)
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
				ctx.BindReg(r14, &d16)
			}
			if d16.Loc == scm.LocReg && d12.Loc == scm.LocReg && d16.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d15)
			ctx.EnsureDesc(&d16)
			var d17 scm.JITValueDesc
			if d15.Loc == scm.LocImm && d16.Loc == scm.LocImm {
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d15.Imm.Int()) << uint64(d16.Imm.Int())))}
			} else if d16.Loc == scm.LocImm {
				r15 := ctx.AllocRegExcept(d15.Reg)
				ctx.W.EmitMovRegReg(r15, d15.Reg)
				ctx.W.EmitShlRegImm8(r15, uint8(d16.Imm.Int()))
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
				ctx.BindReg(r15, &d17)
			} else {
				{
					shiftSrc := d15.Reg
					r16 := ctx.AllocRegExcept(d15.Reg)
					ctx.W.EmitMovRegReg(r16, d15.Reg)
					shiftSrc = r16
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d16.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d16.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d16.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d17)
				}
			}
			if d17.Loc == scm.LocReg && d15.Loc == scm.LocReg && d17.Reg == d15.Reg {
				ctx.TransferReg(d15.Reg)
				d15.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d15)
			ctx.FreeDesc(&d16)
			var d18 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 25)
				r17 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r17, thisptr.Reg, off)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r17}
				ctx.BindReg(r17, &d18)
			}
			d19 = d18
			ctx.EnsureDesc(&d19)
			if d19.Loc != scm.LocImm && d19.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			if d19.Loc == scm.LocImm {
				if d19.Imm.Bool() {
					ctx.W.MarkLabel(lbl15)
					ctx.W.EmitJmp(lbl13)
				} else {
					ctx.W.MarkLabel(lbl16)
			d20 = d17
			if d20.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d20)
			ctx.EmitStoreToStack(d20, 0)
					ctx.W.EmitJmp(lbl14)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d19.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl15)
				ctx.W.EmitJmp(lbl16)
				ctx.W.MarkLabel(lbl15)
				ctx.W.EmitJmp(lbl13)
				ctx.W.MarkLabel(lbl16)
			d21 = d17
			if d21.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d21)
			ctx.EmitStoreToStack(d21, 0)
				ctx.W.EmitJmp(lbl14)
			}
			ctx.FreeDesc(&d18)
			bbpos_1_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl14)
			ctx.W.ResolveFixups()
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			var d22 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r18 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r18, thisptr.Reg, off)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r18}
				ctx.BindReg(r18, &d22)
			}
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d22)
			var d23 scm.JITValueDesc
			if d22.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d22.Imm.Int()))))}
			} else {
				r19 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r19, d22.Reg)
				ctx.W.EmitShlRegImm8(r19, 56)
				ctx.W.EmitShrRegImm8(r19, 56)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
				ctx.BindReg(r19, &d23)
			}
			ctx.FreeDesc(&d22)
			d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d24)
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d24)
			ctx.EnsureDesc(&d23)
			var d25 scm.JITValueDesc
			if d24.Loc == scm.LocImm && d23.Loc == scm.LocImm {
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d24.Imm.Int() - d23.Imm.Int())}
			} else if d23.Loc == scm.LocImm && d23.Imm.Int() == 0 {
				r20 := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitMovRegReg(r20, d24.Reg)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
				ctx.BindReg(r20, &d25)
			} else if d24.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d23.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d24.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d23.Reg)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d25)
			} else if d23.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitMovRegReg(scratch, d24.Reg)
				if d23.Imm.Int() >= -2147483648 && d23.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d23.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d23.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d25)
			} else {
				r21 := ctx.AllocRegExcept(d24.Reg, d23.Reg)
				ctx.W.EmitMovRegReg(r21, d24.Reg)
				ctx.W.EmitSubInt64(r21, d23.Reg)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d25)
			}
			if d25.Loc == scm.LocReg && d24.Loc == scm.LocReg && d25.Reg == d24.Reg {
				ctx.TransferReg(d24.Reg)
				d24.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d23)
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d25)
			var d26 scm.JITValueDesc
			if d8.Loc == scm.LocImm && d25.Loc == scm.LocImm {
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d8.Imm.Int()) >> uint64(d25.Imm.Int())))}
			} else if d25.Loc == scm.LocImm {
				r22 := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitMovRegReg(r22, d8.Reg)
				ctx.W.EmitShrRegImm8(r22, uint8(d25.Imm.Int()))
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d26)
			} else {
				{
					shiftSrc := d8.Reg
					r23 := ctx.AllocRegExcept(d8.Reg)
					ctx.W.EmitMovRegReg(r23, d8.Reg)
					shiftSrc = r23
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d25.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d25.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d25.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d26)
				}
			}
			if d26.Loc == scm.LocReg && d8.Loc == scm.LocReg && d26.Reg == d8.Reg {
				ctx.TransferReg(d8.Reg)
				d8.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d8)
			ctx.FreeDesc(&d25)
			r24 := ctx.AllocReg()
			ctx.EnsureDesc(&d26)
			ctx.EnsureDesc(&d26)
			if d26.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r24, d26)
			}
			ctx.W.EmitJmp(lbl12)
			bbpos_1_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl13)
			ctx.W.ResolveFixups()
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d12)
			var d27 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() % 64)}
			} else {
				r25 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r25, d12.Reg)
				ctx.W.EmitAndRegImm32(r25, 63)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
				ctx.BindReg(r25, &d27)
			}
			if d27.Loc == scm.LocReg && d12.Loc == scm.LocReg && d27.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			var d28 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r26 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r26, thisptr.Reg, off)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r26}
				ctx.BindReg(r26, &d28)
			}
			ctx.EnsureDesc(&d28)
			ctx.EnsureDesc(&d28)
			var d29 scm.JITValueDesc
			if d28.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d28.Imm.Int()))))}
			} else {
				r27 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r27, d28.Reg)
				ctx.W.EmitShlRegImm8(r27, 56)
				ctx.W.EmitShrRegImm8(r27, 56)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d29)
			}
			ctx.FreeDesc(&d28)
			ctx.EnsureDesc(&d27)
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d27)
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d27)
			ctx.EnsureDesc(&d29)
			var d30 scm.JITValueDesc
			if d27.Loc == scm.LocImm && d29.Loc == scm.LocImm {
				d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d27.Imm.Int() + d29.Imm.Int())}
			} else if d29.Loc == scm.LocImm && d29.Imm.Int() == 0 {
				r28 := ctx.AllocRegExcept(d27.Reg)
				ctx.W.EmitMovRegReg(r28, d27.Reg)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d30)
			} else if d27.Loc == scm.LocImm && d27.Imm.Int() == 0 {
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d29.Reg}
				ctx.BindReg(d29.Reg, &d30)
			} else if d27.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d29.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d27.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d29.Reg)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d30)
			} else if d29.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d27.Reg)
				ctx.W.EmitMovRegReg(scratch, d27.Reg)
				if d29.Imm.Int() >= -2147483648 && d29.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d29.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d29.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d30)
			} else {
				r29 := ctx.AllocRegExcept(d27.Reg, d29.Reg)
				ctx.W.EmitMovRegReg(r29, d27.Reg)
				ctx.W.EmitAddInt64(r29, d29.Reg)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d30)
			}
			if d30.Loc == scm.LocReg && d27.Loc == scm.LocReg && d30.Reg == d27.Reg {
				ctx.TransferReg(d27.Reg)
				d27.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d27)
			ctx.FreeDesc(&d29)
			ctx.EnsureDesc(&d30)
			var d31 scm.JITValueDesc
			if d30.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d30.Imm.Int()) > uint64(64))}
			} else {
				r30 := ctx.AllocRegExcept(d30.Reg)
				ctx.W.EmitCmpRegImm32(d30.Reg, 64)
				ctx.W.EmitSetcc(r30, scm.CcA)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r30}
				ctx.BindReg(r30, &d31)
			}
			ctx.FreeDesc(&d30)
			d32 = d31
			ctx.EnsureDesc(&d32)
			if d32.Loc != scm.LocImm && d32.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			lbl19 := ctx.W.ReserveLabel()
			if d32.Loc == scm.LocImm {
				if d32.Imm.Bool() {
					ctx.W.MarkLabel(lbl18)
					ctx.W.EmitJmp(lbl17)
				} else {
					ctx.W.MarkLabel(lbl19)
			d33 = d17
			if d33.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d33)
			ctx.EmitStoreToStack(d33, 0)
					ctx.W.EmitJmp(lbl14)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d32.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl18)
				ctx.W.EmitJmp(lbl19)
				ctx.W.MarkLabel(lbl18)
				ctx.W.EmitJmp(lbl17)
				ctx.W.MarkLabel(lbl19)
			d34 = d17
			if d34.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d34)
			ctx.EmitStoreToStack(d34, 0)
				ctx.W.EmitJmp(lbl14)
			}
			ctx.FreeDesc(&d31)
			bbpos_1_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl17)
			ctx.W.ResolveFixups()
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d12)
			var d35 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() / 64)}
			} else {
				r31 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r31, d12.Reg)
				ctx.W.EmitShrRegImm8(r31, 6)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
				ctx.BindReg(r31, &d35)
			}
			if d35.Loc == scm.LocReg && d12.Loc == scm.LocReg && d35.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d35)
			ctx.EnsureDesc(&d35)
			var d36 scm.JITValueDesc
			if d35.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d35.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d35.Reg)
				ctx.W.EmitMovRegReg(scratch, d35.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d36)
			}
			if d36.Loc == scm.LocReg && d35.Loc == scm.LocReg && d36.Reg == d35.Reg {
				ctx.TransferReg(d35.Reg)
				d35.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d35)
			ctx.EnsureDesc(&d36)
			r32 := ctx.AllocReg()
			ctx.EnsureDesc(&d36)
			ctx.EnsureDesc(&d13)
			if d36.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r32, uint64(d36.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r32, d36.Reg)
				ctx.W.EmitShlRegImm8(r32, 3)
			}
			if d13.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
				ctx.W.EmitAddInt64(r32, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r32, d13.Reg)
			}
			r33 := ctx.AllocRegExcept(r32)
			ctx.W.EmitMovRegMem(r33, r32, 0)
			ctx.FreeReg(r32)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r33}
			ctx.BindReg(r33, &d37)
			ctx.FreeDesc(&d36)
			ctx.EnsureDesc(&d12)
			var d38 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() % 64)}
			} else {
				r34 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r34, d12.Reg)
				ctx.W.EmitAndRegImm32(r34, 63)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
				ctx.BindReg(r34, &d38)
			}
			if d38.Loc == scm.LocReg && d12.Loc == scm.LocReg && d38.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d12)
			d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d39)
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d39)
			ctx.EnsureDesc(&d38)
			var d40 scm.JITValueDesc
			if d39.Loc == scm.LocImm && d38.Loc == scm.LocImm {
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() - d38.Imm.Int())}
			} else if d38.Loc == scm.LocImm && d38.Imm.Int() == 0 {
				r35 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r35, d39.Reg)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d40)
			} else if d39.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d38.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d39.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d38.Reg)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d40)
			} else if d38.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(scratch, d39.Reg)
				if d38.Imm.Int() >= -2147483648 && d38.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d38.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d38.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d40)
			} else {
				r36 := ctx.AllocRegExcept(d39.Reg, d38.Reg)
				ctx.W.EmitMovRegReg(r36, d39.Reg)
				ctx.W.EmitSubInt64(r36, d38.Reg)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d40)
			}
			if d40.Loc == scm.LocReg && d39.Loc == scm.LocReg && d40.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d38)
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d40)
			var d41 scm.JITValueDesc
			if d37.Loc == scm.LocImm && d40.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d37.Imm.Int()) >> uint64(d40.Imm.Int())))}
			} else if d40.Loc == scm.LocImm {
				r37 := ctx.AllocRegExcept(d37.Reg)
				ctx.W.EmitMovRegReg(r37, d37.Reg)
				ctx.W.EmitShrRegImm8(r37, uint8(d40.Imm.Int()))
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d41)
			} else {
				{
					shiftSrc := d37.Reg
					r38 := ctx.AllocRegExcept(d37.Reg)
					ctx.W.EmitMovRegReg(r38, d37.Reg)
					shiftSrc = r38
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d40.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d40.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d40.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d41)
				}
			}
			if d41.Loc == scm.LocReg && d37.Loc == scm.LocReg && d41.Reg == d37.Reg {
				ctx.TransferReg(d37.Reg)
				d37.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d37)
			ctx.FreeDesc(&d40)
			ctx.EnsureDesc(&d17)
			ctx.EnsureDesc(&d41)
			var d42 scm.JITValueDesc
			if d17.Loc == scm.LocImm && d41.Loc == scm.LocImm {
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d17.Imm.Int() | d41.Imm.Int())}
			} else if d17.Loc == scm.LocImm && d17.Imm.Int() == 0 {
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d41.Reg}
				ctx.BindReg(d41.Reg, &d42)
			} else if d41.Loc == scm.LocImm && d41.Imm.Int() == 0 {
				r39 := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(r39, d17.Reg)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d42)
			} else if d17.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d41.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d17.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d41.Reg)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d42)
			} else if d41.Loc == scm.LocImm {
				r40 := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(r40, d17.Reg)
				if d41.Imm.Int() >= -2147483648 && d41.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r40, int32(d41.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d41.Imm.Int()))
					ctx.W.EmitOrInt64(r40, scm.RegR11)
				}
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
				ctx.BindReg(r40, &d42)
			} else {
				r41 := ctx.AllocRegExcept(d17.Reg, d41.Reg)
				ctx.W.EmitMovRegReg(r41, d17.Reg)
				ctx.W.EmitOrInt64(r41, d41.Reg)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d42)
			}
			if d42.Loc == scm.LocReg && d17.Loc == scm.LocReg && d42.Reg == d17.Reg {
				ctx.TransferReg(d17.Reg)
				d17.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d41)
			d43 = d42
			if d43.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d43)
			ctx.EmitStoreToStack(d43, 0)
			ctx.W.EmitJmp(lbl14)
			ctx.W.MarkLabel(lbl12)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r24}
			ctx.BindReg(r24, &d44)
			ctx.BindReg(r24, &d44)
			if r3 { ctx.UnprotectReg(r4) }
			ctx.EnsureDesc(&d44)
			ctx.EnsureDesc(&d44)
			var d45 scm.JITValueDesc
			if d44.Loc == scm.LocImm {
				d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d44.Imm.Int()))))}
			} else {
				r42 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r42, d44.Reg)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d45)
			}
			ctx.FreeDesc(&d44)
			var d46 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 32)
				r43 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r43, thisptr.Reg, off)
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r43}
				ctx.BindReg(r43, &d46)
			}
			ctx.EnsureDesc(&d45)
			ctx.EnsureDesc(&d46)
			ctx.EnsureDesc(&d45)
			ctx.EnsureDesc(&d46)
			ctx.EnsureDesc(&d45)
			ctx.EnsureDesc(&d46)
			var d47 scm.JITValueDesc
			if d45.Loc == scm.LocImm && d46.Loc == scm.LocImm {
				d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d45.Imm.Int() + d46.Imm.Int())}
			} else if d46.Loc == scm.LocImm && d46.Imm.Int() == 0 {
				r44 := ctx.AllocRegExcept(d45.Reg)
				ctx.W.EmitMovRegReg(r44, d45.Reg)
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d47)
			} else if d45.Loc == scm.LocImm && d45.Imm.Int() == 0 {
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d46.Reg}
				ctx.BindReg(d46.Reg, &d47)
			} else if d45.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d46.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d45.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d46.Reg)
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d47)
			} else if d46.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d45.Reg)
				ctx.W.EmitMovRegReg(scratch, d45.Reg)
				if d46.Imm.Int() >= -2147483648 && d46.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d46.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d46.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d47)
			} else {
				r45 := ctx.AllocRegExcept(d45.Reg, d46.Reg)
				ctx.W.EmitMovRegReg(r45, d45.Reg)
				ctx.W.EmitAddInt64(r45, d46.Reg)
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
				ctx.BindReg(r45, &d47)
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
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d47.Imm.Int()))))}
			} else {
				r46 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r46, d47.Reg)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d48)
			}
			ctx.FreeDesc(&d47)
			var d49 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 56)
				r47 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r47, thisptr.Reg, off)
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r47}
				ctx.BindReg(r47, &d49)
			}
			d50 = d49
			ctx.EnsureDesc(&d50)
			if d50.Loc != scm.LocImm && d50.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d50.Loc == scm.LocImm {
				if d50.Imm.Bool() {
			ps51 := scm.PhiState{General: ps.General}
			ps51.OverlayValues = make([]scm.JITValueDesc, 51)
			ps51.OverlayValues[0] = d0
			ps51.OverlayValues[1] = d1
			ps51.OverlayValues[7] = d7
			ps51.OverlayValues[8] = d8
			ps51.OverlayValues[9] = d9
			ps51.OverlayValues[10] = d10
			ps51.OverlayValues[11] = d11
			ps51.OverlayValues[12] = d12
			ps51.OverlayValues[13] = d13
			ps51.OverlayValues[14] = d14
			ps51.OverlayValues[15] = d15
			ps51.OverlayValues[16] = d16
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
					return bbs[5].RenderPS(ps51)
				}
			ps52 := scm.PhiState{General: ps.General}
			ps52.OverlayValues = make([]scm.JITValueDesc, 51)
			ps52.OverlayValues[0] = d0
			ps52.OverlayValues[1] = d1
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
				return bbs[4].RenderPS(ps52)
			}
			lbl20 := ctx.W.ReserveLabel()
			lbl21 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d50.Reg, 0)
			ctx.W.EmitJcc(scm.CcNE, lbl20)
			ctx.W.EmitJmp(lbl21)
			ctx.W.MarkLabel(lbl20)
			ctx.W.EmitJmp(lbl6)
			ctx.W.MarkLabel(lbl21)
			ctx.W.EmitJmp(lbl5)
			ps53 := scm.PhiState{General: true}
			ps53.OverlayValues = make([]scm.JITValueDesc, 51)
			ps53.OverlayValues[0] = d0
			ps53.OverlayValues[1] = d1
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
			ps54 := scm.PhiState{General: true}
			ps54.OverlayValues = make([]scm.JITValueDesc, 51)
			ps54.OverlayValues[0] = d0
			ps54.OverlayValues[1] = d1
			ps54.OverlayValues[7] = d7
			ps54.OverlayValues[8] = d8
			ps54.OverlayValues[9] = d9
			ps54.OverlayValues[10] = d10
			ps54.OverlayValues[11] = d11
			ps54.OverlayValues[12] = d12
			ps54.OverlayValues[13] = d13
			ps54.OverlayValues[14] = d14
			ps54.OverlayValues[15] = d15
			ps54.OverlayValues[16] = d16
			ps54.OverlayValues[17] = d17
			ps54.OverlayValues[18] = d18
			ps54.OverlayValues[19] = d19
			ps54.OverlayValues[20] = d20
			ps54.OverlayValues[21] = d21
			ps54.OverlayValues[22] = d22
			ps54.OverlayValues[23] = d23
			ps54.OverlayValues[24] = d24
			ps54.OverlayValues[25] = d25
			ps54.OverlayValues[26] = d26
			ps54.OverlayValues[27] = d27
			ps54.OverlayValues[28] = d28
			ps54.OverlayValues[29] = d29
			ps54.OverlayValues[30] = d30
			ps54.OverlayValues[31] = d31
			ps54.OverlayValues[32] = d32
			ps54.OverlayValues[33] = d33
			ps54.OverlayValues[34] = d34
			ps54.OverlayValues[35] = d35
			ps54.OverlayValues[36] = d36
			ps54.OverlayValues[37] = d37
			ps54.OverlayValues[38] = d38
			ps54.OverlayValues[39] = d39
			ps54.OverlayValues[40] = d40
			ps54.OverlayValues[41] = d41
			ps54.OverlayValues[42] = d42
			ps54.OverlayValues[43] = d43
			ps54.OverlayValues[44] = d44
			ps54.OverlayValues[45] = d45
			ps54.OverlayValues[46] = d46
			ps54.OverlayValues[47] = d47
			ps54.OverlayValues[48] = d48
			ps54.OverlayValues[49] = d49
			ps54.OverlayValues[50] = d50
			snap55 := d48
			alloc56 := ctx.SnapshotAllocState()
			if !bbs[4].Rendered {
				bbs[4].RenderPS(ps54)
			}
			ctx.RestoreAllocState(alloc56)
			d48 = snap55
			if !bbs[5].Rendered {
				return bbs[5].RenderPS(ps53)
			}
			return result
			ctx.FreeDesc(&d49)
			return result
			}
			bbs[2].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[2].VisitCount >= 2 {
					ps.General = true
					return bbs[2].RenderPS(ps)
				}
			}
			bbs[2].VisitCount++
			if ps.General {
				if bbs[2].Rendered {
					ctx.W.EmitJmp(lbl3)
					return result
				}
				bbs[2].Rendered = true
				bbs[2].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_2 = bbs[2].Address
				ctx.W.MarkLabel(lbl3)
				ctx.W.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&idxInt)
			d57 = idxInt
			_ = d57
			r48 := idxInt.Loc == scm.LocReg
			r49 := idxInt.Reg
			if r48 { ctx.ProtectReg(r49) }
			d58 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			lbl22 := ctx.W.ReserveLabel()
			bbpos_2_0 := int32(-1)
			_ = bbpos_2_0
			bbpos_2_1 := int32(-1)
			_ = bbpos_2_1
			bbpos_2_2 := int32(-1)
			_ = bbpos_2_2
			bbpos_2_3 := int32(-1)
			_ = bbpos_2_3
			bbpos_2_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			d58 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d57)
			ctx.EnsureDesc(&d57)
			var d59 scm.JITValueDesc
			if d57.Loc == scm.LocImm {
				d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d57.Imm.Int()))))}
			} else {
				r50 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r50, d57.Reg)
				ctx.W.EmitShlRegImm8(r50, 32)
				ctx.W.EmitShrRegImm8(r50, 32)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
				ctx.BindReg(r50, &d59)
			}
			var d60 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 24)
				r51 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r51, thisptr.Reg, off)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r51}
				ctx.BindReg(r51, &d60)
			}
			ctx.EnsureDesc(&d60)
			ctx.EnsureDesc(&d60)
			var d61 scm.JITValueDesc
			if d60.Loc == scm.LocImm {
				d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d60.Imm.Int()))))}
			} else {
				r52 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r52, d60.Reg)
				ctx.W.EmitShlRegImm8(r52, 56)
				ctx.W.EmitShrRegImm8(r52, 56)
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
				ctx.BindReg(r52, &d61)
			}
			ctx.FreeDesc(&d60)
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d61)
			var d62 scm.JITValueDesc
			if d59.Loc == scm.LocImm && d61.Loc == scm.LocImm {
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d59.Imm.Int() * d61.Imm.Int())}
			} else if d59.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d61.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d59.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d61.Reg)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d62)
			} else if d61.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d59.Reg)
				ctx.W.EmitMovRegReg(scratch, d59.Reg)
				if d61.Imm.Int() >= -2147483648 && d61.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d61.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d61.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d62)
			} else {
				r53 := ctx.AllocRegExcept(d59.Reg, d61.Reg)
				ctx.W.EmitMovRegReg(r53, d59.Reg)
				ctx.W.EmitImulInt64(r53, d61.Reg)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
				ctx.BindReg(r53, &d62)
			}
			if d62.Loc == scm.LocReg && d59.Loc == scm.LocReg && d62.Reg == d59.Reg {
				ctx.TransferReg(d59.Reg)
				d59.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d59)
			ctx.FreeDesc(&d61)
			var d63 scm.JITValueDesc
			r54 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r54, uint64(dataPtr))
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r54, StackOff: int32(sliceLen)}
				ctx.BindReg(r54, &d63)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 0)
				ctx.W.EmitMovRegMem(r54, thisptr.Reg, off)
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r54}
				ctx.BindReg(r54, &d63)
			}
			ctx.BindReg(r54, &d63)
			ctx.EnsureDesc(&d62)
			var d64 scm.JITValueDesc
			if d62.Loc == scm.LocImm {
				d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d62.Imm.Int() / 64)}
			} else {
				r55 := ctx.AllocRegExcept(d62.Reg)
				ctx.W.EmitMovRegReg(r55, d62.Reg)
				ctx.W.EmitShrRegImm8(r55, 6)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
				ctx.BindReg(r55, &d64)
			}
			if d64.Loc == scm.LocReg && d62.Loc == scm.LocReg && d64.Reg == d62.Reg {
				ctx.TransferReg(d62.Reg)
				d62.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d64)
			r56 := ctx.AllocReg()
			ctx.EnsureDesc(&d64)
			ctx.EnsureDesc(&d63)
			if d64.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r56, uint64(d64.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r56, d64.Reg)
				ctx.W.EmitShlRegImm8(r56, 3)
			}
			if d63.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d63.Imm.Int()))
				ctx.W.EmitAddInt64(r56, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r56, d63.Reg)
			}
			r57 := ctx.AllocRegExcept(r56)
			ctx.W.EmitMovRegMem(r57, r56, 0)
			ctx.FreeReg(r56)
			d65 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r57}
			ctx.BindReg(r57, &d65)
			ctx.FreeDesc(&d64)
			ctx.EnsureDesc(&d62)
			var d66 scm.JITValueDesc
			if d62.Loc == scm.LocImm {
				d66 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d62.Imm.Int() % 64)}
			} else {
				r58 := ctx.AllocRegExcept(d62.Reg)
				ctx.W.EmitMovRegReg(r58, d62.Reg)
				ctx.W.EmitAndRegImm32(r58, 63)
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r58}
				ctx.BindReg(r58, &d66)
			}
			if d66.Loc == scm.LocReg && d62.Loc == scm.LocReg && d66.Reg == d62.Reg {
				ctx.TransferReg(d62.Reg)
				d62.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d65)
			ctx.EnsureDesc(&d66)
			var d67 scm.JITValueDesc
			if d65.Loc == scm.LocImm && d66.Loc == scm.LocImm {
				d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d65.Imm.Int()) << uint64(d66.Imm.Int())))}
			} else if d66.Loc == scm.LocImm {
				r59 := ctx.AllocRegExcept(d65.Reg)
				ctx.W.EmitMovRegReg(r59, d65.Reg)
				ctx.W.EmitShlRegImm8(r59, uint8(d66.Imm.Int()))
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r59}
				ctx.BindReg(r59, &d67)
			} else {
				{
					shiftSrc := d65.Reg
					r60 := ctx.AllocRegExcept(d65.Reg)
					ctx.W.EmitMovRegReg(r60, d65.Reg)
					shiftSrc = r60
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d66.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d66.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d66.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d67)
				}
			}
			if d67.Loc == scm.LocReg && d65.Loc == scm.LocReg && d67.Reg == d65.Reg {
				ctx.TransferReg(d65.Reg)
				d65.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d65)
			ctx.FreeDesc(&d66)
			var d68 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 25)
				r61 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r61, thisptr.Reg, off)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r61}
				ctx.BindReg(r61, &d68)
			}
			d69 = d68
			ctx.EnsureDesc(&d69)
			if d69.Loc != scm.LocImm && d69.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl23 := ctx.W.ReserveLabel()
			lbl24 := ctx.W.ReserveLabel()
			lbl25 := ctx.W.ReserveLabel()
			lbl26 := ctx.W.ReserveLabel()
			if d69.Loc == scm.LocImm {
				if d69.Imm.Bool() {
					ctx.W.MarkLabel(lbl25)
					ctx.W.EmitJmp(lbl23)
				} else {
					ctx.W.MarkLabel(lbl26)
			d70 = d67
			if d70.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d70)
			ctx.EmitStoreToStack(d70, 8)
					ctx.W.EmitJmp(lbl24)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d69.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl25)
				ctx.W.EmitJmp(lbl26)
				ctx.W.MarkLabel(lbl25)
				ctx.W.EmitJmp(lbl23)
				ctx.W.MarkLabel(lbl26)
			d71 = d67
			if d71.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d71)
			ctx.EmitStoreToStack(d71, 8)
				ctx.W.EmitJmp(lbl24)
			}
			ctx.FreeDesc(&d68)
			bbpos_2_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl24)
			ctx.W.ResolveFixups()
			d58 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			var d72 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 24)
				r62 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r62, thisptr.Reg, off)
				d72 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r62}
				ctx.BindReg(r62, &d72)
			}
			ctx.EnsureDesc(&d72)
			ctx.EnsureDesc(&d72)
			var d73 scm.JITValueDesc
			if d72.Loc == scm.LocImm {
				d73 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d72.Imm.Int()))))}
			} else {
				r63 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r63, d72.Reg)
				ctx.W.EmitShlRegImm8(r63, 56)
				ctx.W.EmitShrRegImm8(r63, 56)
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r63}
				ctx.BindReg(r63, &d73)
			}
			ctx.FreeDesc(&d72)
			d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d73)
			ctx.EnsureDesc(&d74)
			ctx.EnsureDesc(&d73)
			ctx.EnsureDesc(&d74)
			ctx.EnsureDesc(&d73)
			var d75 scm.JITValueDesc
			if d74.Loc == scm.LocImm && d73.Loc == scm.LocImm {
				d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() - d73.Imm.Int())}
			} else if d73.Loc == scm.LocImm && d73.Imm.Int() == 0 {
				r64 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r64, d74.Reg)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r64}
				ctx.BindReg(r64, &d75)
			} else if d74.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d73.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d74.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d73.Reg)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d75)
			} else if d73.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(scratch, d74.Reg)
				if d73.Imm.Int() >= -2147483648 && d73.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d73.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d73.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d75)
			} else {
				r65 := ctx.AllocRegExcept(d74.Reg, d73.Reg)
				ctx.W.EmitMovRegReg(r65, d74.Reg)
				ctx.W.EmitSubInt64(r65, d73.Reg)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
				ctx.BindReg(r65, &d75)
			}
			if d75.Loc == scm.LocReg && d74.Loc == scm.LocReg && d75.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d73)
			ctx.EnsureDesc(&d58)
			ctx.EnsureDesc(&d75)
			var d76 scm.JITValueDesc
			if d58.Loc == scm.LocImm && d75.Loc == scm.LocImm {
				d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d58.Imm.Int()) >> uint64(d75.Imm.Int())))}
			} else if d75.Loc == scm.LocImm {
				r66 := ctx.AllocRegExcept(d58.Reg)
				ctx.W.EmitMovRegReg(r66, d58.Reg)
				ctx.W.EmitShrRegImm8(r66, uint8(d75.Imm.Int()))
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
				ctx.BindReg(r66, &d76)
			} else {
				{
					shiftSrc := d58.Reg
					r67 := ctx.AllocRegExcept(d58.Reg)
					ctx.W.EmitMovRegReg(r67, d58.Reg)
					shiftSrc = r67
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d75.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d75.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d75.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d76)
				}
			}
			if d76.Loc == scm.LocReg && d58.Loc == scm.LocReg && d76.Reg == d58.Reg {
				ctx.TransferReg(d58.Reg)
				d58.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d58)
			ctx.FreeDesc(&d75)
			r68 := ctx.AllocReg()
			ctx.EnsureDesc(&d76)
			ctx.EnsureDesc(&d76)
			if d76.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r68, d76)
			}
			ctx.W.EmitJmp(lbl22)
			bbpos_2_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl23)
			ctx.W.ResolveFixups()
			d58 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d62)
			var d77 scm.JITValueDesc
			if d62.Loc == scm.LocImm {
				d77 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d62.Imm.Int() % 64)}
			} else {
				r69 := ctx.AllocRegExcept(d62.Reg)
				ctx.W.EmitMovRegReg(r69, d62.Reg)
				ctx.W.EmitAndRegImm32(r69, 63)
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r69}
				ctx.BindReg(r69, &d77)
			}
			if d77.Loc == scm.LocReg && d62.Loc == scm.LocReg && d77.Reg == d62.Reg {
				ctx.TransferReg(d62.Reg)
				d62.Loc = scm.LocNone
			}
			var d78 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d78 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 24)
				r70 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r70, thisptr.Reg, off)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r70}
				ctx.BindReg(r70, &d78)
			}
			ctx.EnsureDesc(&d78)
			ctx.EnsureDesc(&d78)
			var d79 scm.JITValueDesc
			if d78.Loc == scm.LocImm {
				d79 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d78.Imm.Int()))))}
			} else {
				r71 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r71, d78.Reg)
				ctx.W.EmitShlRegImm8(r71, 56)
				ctx.W.EmitShrRegImm8(r71, 56)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r71}
				ctx.BindReg(r71, &d79)
			}
			ctx.FreeDesc(&d78)
			ctx.EnsureDesc(&d77)
			ctx.EnsureDesc(&d79)
			ctx.EnsureDesc(&d77)
			ctx.EnsureDesc(&d79)
			ctx.EnsureDesc(&d77)
			ctx.EnsureDesc(&d79)
			var d80 scm.JITValueDesc
			if d77.Loc == scm.LocImm && d79.Loc == scm.LocImm {
				d80 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d77.Imm.Int() + d79.Imm.Int())}
			} else if d79.Loc == scm.LocImm && d79.Imm.Int() == 0 {
				r72 := ctx.AllocRegExcept(d77.Reg)
				ctx.W.EmitMovRegReg(r72, d77.Reg)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r72}
				ctx.BindReg(r72, &d80)
			} else if d77.Loc == scm.LocImm && d77.Imm.Int() == 0 {
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d79.Reg}
				ctx.BindReg(d79.Reg, &d80)
			} else if d77.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d77.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d79.Reg)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d80)
			} else if d79.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d77.Reg)
				ctx.W.EmitMovRegReg(scratch, d77.Reg)
				if d79.Imm.Int() >= -2147483648 && d79.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d79.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d79.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d80)
			} else {
				r73 := ctx.AllocRegExcept(d77.Reg, d79.Reg)
				ctx.W.EmitMovRegReg(r73, d77.Reg)
				ctx.W.EmitAddInt64(r73, d79.Reg)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
				ctx.BindReg(r73, &d80)
			}
			if d80.Loc == scm.LocReg && d77.Loc == scm.LocReg && d80.Reg == d77.Reg {
				ctx.TransferReg(d77.Reg)
				d77.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d77)
			ctx.FreeDesc(&d79)
			ctx.EnsureDesc(&d80)
			var d81 scm.JITValueDesc
			if d80.Loc == scm.LocImm {
				d81 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d80.Imm.Int()) > uint64(64))}
			} else {
				r74 := ctx.AllocRegExcept(d80.Reg)
				ctx.W.EmitCmpRegImm32(d80.Reg, 64)
				ctx.W.EmitSetcc(r74, scm.CcA)
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r74}
				ctx.BindReg(r74, &d81)
			}
			ctx.FreeDesc(&d80)
			d82 = d81
			ctx.EnsureDesc(&d82)
			if d82.Loc != scm.LocImm && d82.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl27 := ctx.W.ReserveLabel()
			lbl28 := ctx.W.ReserveLabel()
			lbl29 := ctx.W.ReserveLabel()
			if d82.Loc == scm.LocImm {
				if d82.Imm.Bool() {
					ctx.W.MarkLabel(lbl28)
					ctx.W.EmitJmp(lbl27)
				} else {
					ctx.W.MarkLabel(lbl29)
			d83 = d67
			if d83.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d83)
			ctx.EmitStoreToStack(d83, 8)
					ctx.W.EmitJmp(lbl24)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d82.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl28)
				ctx.W.EmitJmp(lbl29)
				ctx.W.MarkLabel(lbl28)
				ctx.W.EmitJmp(lbl27)
				ctx.W.MarkLabel(lbl29)
			d84 = d67
			if d84.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d84)
			ctx.EmitStoreToStack(d84, 8)
				ctx.W.EmitJmp(lbl24)
			}
			ctx.FreeDesc(&d81)
			bbpos_2_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl27)
			ctx.W.ResolveFixups()
			d58 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d62)
			var d85 scm.JITValueDesc
			if d62.Loc == scm.LocImm {
				d85 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d62.Imm.Int() / 64)}
			} else {
				r75 := ctx.AllocRegExcept(d62.Reg)
				ctx.W.EmitMovRegReg(r75, d62.Reg)
				ctx.W.EmitShrRegImm8(r75, 6)
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
				ctx.BindReg(r75, &d85)
			}
			if d85.Loc == scm.LocReg && d62.Loc == scm.LocReg && d85.Reg == d62.Reg {
				ctx.TransferReg(d62.Reg)
				d62.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d85)
			ctx.EnsureDesc(&d85)
			var d86 scm.JITValueDesc
			if d85.Loc == scm.LocImm {
				d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d85.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d85.Reg)
				ctx.W.EmitMovRegReg(scratch, d85.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d86)
			}
			if d86.Loc == scm.LocReg && d85.Loc == scm.LocReg && d86.Reg == d85.Reg {
				ctx.TransferReg(d85.Reg)
				d85.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d85)
			ctx.EnsureDesc(&d86)
			r76 := ctx.AllocReg()
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d63)
			if d86.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r76, uint64(d86.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r76, d86.Reg)
				ctx.W.EmitShlRegImm8(r76, 3)
			}
			if d63.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d63.Imm.Int()))
				ctx.W.EmitAddInt64(r76, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r76, d63.Reg)
			}
			r77 := ctx.AllocRegExcept(r76)
			ctx.W.EmitMovRegMem(r77, r76, 0)
			ctx.FreeReg(r76)
			d87 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r77}
			ctx.BindReg(r77, &d87)
			ctx.FreeDesc(&d86)
			ctx.EnsureDesc(&d62)
			var d88 scm.JITValueDesc
			if d62.Loc == scm.LocImm {
				d88 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d62.Imm.Int() % 64)}
			} else {
				r78 := ctx.AllocRegExcept(d62.Reg)
				ctx.W.EmitMovRegReg(r78, d62.Reg)
				ctx.W.EmitAndRegImm32(r78, 63)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r78}
				ctx.BindReg(r78, &d88)
			}
			if d88.Loc == scm.LocReg && d62.Loc == scm.LocReg && d88.Reg == d62.Reg {
				ctx.TransferReg(d62.Reg)
				d62.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d62)
			d89 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d88)
			ctx.EnsureDesc(&d89)
			ctx.EnsureDesc(&d88)
			ctx.EnsureDesc(&d89)
			ctx.EnsureDesc(&d88)
			var d90 scm.JITValueDesc
			if d89.Loc == scm.LocImm && d88.Loc == scm.LocImm {
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d89.Imm.Int() - d88.Imm.Int())}
			} else if d88.Loc == scm.LocImm && d88.Imm.Int() == 0 {
				r79 := ctx.AllocRegExcept(d89.Reg)
				ctx.W.EmitMovRegReg(r79, d89.Reg)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r79}
				ctx.BindReg(r79, &d90)
			} else if d89.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d88.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d89.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d88.Reg)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d90)
			} else if d88.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d89.Reg)
				ctx.W.EmitMovRegReg(scratch, d89.Reg)
				if d88.Imm.Int() >= -2147483648 && d88.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d88.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d88.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d90)
			} else {
				r80 := ctx.AllocRegExcept(d89.Reg, d88.Reg)
				ctx.W.EmitMovRegReg(r80, d89.Reg)
				ctx.W.EmitSubInt64(r80, d88.Reg)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
				ctx.BindReg(r80, &d90)
			}
			if d90.Loc == scm.LocReg && d89.Loc == scm.LocReg && d90.Reg == d89.Reg {
				ctx.TransferReg(d89.Reg)
				d89.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d88)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d90)
			var d91 scm.JITValueDesc
			if d87.Loc == scm.LocImm && d90.Loc == scm.LocImm {
				d91 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d87.Imm.Int()) >> uint64(d90.Imm.Int())))}
			} else if d90.Loc == scm.LocImm {
				r81 := ctx.AllocRegExcept(d87.Reg)
				ctx.W.EmitMovRegReg(r81, d87.Reg)
				ctx.W.EmitShrRegImm8(r81, uint8(d90.Imm.Int()))
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
				ctx.BindReg(r81, &d91)
			} else {
				{
					shiftSrc := d87.Reg
					r82 := ctx.AllocRegExcept(d87.Reg)
					ctx.W.EmitMovRegReg(r82, d87.Reg)
					shiftSrc = r82
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d90.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d90.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d90.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d91)
				}
			}
			if d91.Loc == scm.LocReg && d87.Loc == scm.LocReg && d91.Reg == d87.Reg {
				ctx.TransferReg(d87.Reg)
				d87.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d87)
			ctx.FreeDesc(&d90)
			ctx.EnsureDesc(&d67)
			ctx.EnsureDesc(&d91)
			var d92 scm.JITValueDesc
			if d67.Loc == scm.LocImm && d91.Loc == scm.LocImm {
				d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d67.Imm.Int() | d91.Imm.Int())}
			} else if d67.Loc == scm.LocImm && d67.Imm.Int() == 0 {
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d91.Reg}
				ctx.BindReg(d91.Reg, &d92)
			} else if d91.Loc == scm.LocImm && d91.Imm.Int() == 0 {
				r83 := ctx.AllocRegExcept(d67.Reg)
				ctx.W.EmitMovRegReg(r83, d67.Reg)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
				ctx.BindReg(r83, &d92)
			} else if d67.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d91.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d67.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d91.Reg)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d92)
			} else if d91.Loc == scm.LocImm {
				r84 := ctx.AllocRegExcept(d67.Reg)
				ctx.W.EmitMovRegReg(r84, d67.Reg)
				if d91.Imm.Int() >= -2147483648 && d91.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r84, int32(d91.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d91.Imm.Int()))
					ctx.W.EmitOrInt64(r84, scm.RegR11)
				}
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r84}
				ctx.BindReg(r84, &d92)
			} else {
				r85 := ctx.AllocRegExcept(d67.Reg, d91.Reg)
				ctx.W.EmitMovRegReg(r85, d67.Reg)
				ctx.W.EmitOrInt64(r85, d91.Reg)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
				ctx.BindReg(r85, &d92)
			}
			if d92.Loc == scm.LocReg && d67.Loc == scm.LocReg && d92.Reg == d67.Reg {
				ctx.TransferReg(d67.Reg)
				d67.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d91)
			d93 = d92
			if d93.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d93)
			ctx.EmitStoreToStack(d93, 8)
			ctx.W.EmitJmp(lbl24)
			ctx.W.MarkLabel(lbl22)
			d94 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r68}
			ctx.BindReg(r68, &d94)
			ctx.BindReg(r68, &d94)
			if r48 { ctx.UnprotectReg(r49) }
			ctx.EnsureDesc(&d94)
			ctx.EnsureDesc(&d94)
			var d95 scm.JITValueDesc
			if d94.Loc == scm.LocImm {
				d95 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d94.Imm.Int()))))}
			} else {
				r86 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r86, d94.Reg)
				d95 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
				ctx.BindReg(r86, &d95)
			}
			ctx.FreeDesc(&d94)
			var d96 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d96 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 32)
				r87 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r87, thisptr.Reg, off)
				d96 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r87}
				ctx.BindReg(r87, &d96)
			}
			ctx.EnsureDesc(&d95)
			ctx.EnsureDesc(&d96)
			ctx.EnsureDesc(&d95)
			ctx.EnsureDesc(&d96)
			ctx.EnsureDesc(&d95)
			ctx.EnsureDesc(&d96)
			var d97 scm.JITValueDesc
			if d95.Loc == scm.LocImm && d96.Loc == scm.LocImm {
				d97 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d95.Imm.Int() + d96.Imm.Int())}
			} else if d96.Loc == scm.LocImm && d96.Imm.Int() == 0 {
				r88 := ctx.AllocRegExcept(d95.Reg)
				ctx.W.EmitMovRegReg(r88, d95.Reg)
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
				ctx.BindReg(r88, &d97)
			} else if d95.Loc == scm.LocImm && d95.Imm.Int() == 0 {
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d96.Reg}
				ctx.BindReg(d96.Reg, &d97)
			} else if d95.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d96.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d95.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d96.Reg)
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d97)
			} else if d96.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d95.Reg)
				ctx.W.EmitMovRegReg(scratch, d95.Reg)
				if d96.Imm.Int() >= -2147483648 && d96.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d96.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d96.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d97)
			} else {
				r89 := ctx.AllocRegExcept(d95.Reg, d96.Reg)
				ctx.W.EmitMovRegReg(r89, d95.Reg)
				ctx.W.EmitAddInt64(r89, d96.Reg)
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
				ctx.BindReg(r89, &d97)
			}
			if d97.Loc == scm.LocReg && d95.Loc == scm.LocReg && d97.Reg == d95.Reg {
				ctx.TransferReg(d95.Reg)
				d95.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d95)
			ctx.FreeDesc(&d96)
			ctx.EnsureDesc(&d97)
			ctx.EnsureDesc(&d97)
			var d98 scm.JITValueDesc
			if d97.Loc == scm.LocImm {
				d98 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d97.Imm.Int()))))}
			} else {
				r90 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r90, d97.Reg)
				ctx.W.EmitShlRegImm8(r90, 32)
				ctx.W.EmitShrRegImm8(r90, 32)
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
				ctx.BindReg(r90, &d98)
			}
			ctx.FreeDesc(&d97)
			var d99 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d99 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 56)
				r91 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r91, thisptr.Reg, off)
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r91}
				ctx.BindReg(r91, &d99)
			}
			d100 = d99
			ctx.EnsureDesc(&d100)
			if d100.Loc != scm.LocImm && d100.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d100.Loc == scm.LocImm {
				if d100.Imm.Bool() {
			ps101 := scm.PhiState{General: ps.General}
			ps101.OverlayValues = make([]scm.JITValueDesc, 101)
			ps101.OverlayValues[0] = d0
			ps101.OverlayValues[1] = d1
			ps101.OverlayValues[7] = d7
			ps101.OverlayValues[8] = d8
			ps101.OverlayValues[9] = d9
			ps101.OverlayValues[10] = d10
			ps101.OverlayValues[11] = d11
			ps101.OverlayValues[12] = d12
			ps101.OverlayValues[13] = d13
			ps101.OverlayValues[14] = d14
			ps101.OverlayValues[15] = d15
			ps101.OverlayValues[16] = d16
			ps101.OverlayValues[17] = d17
			ps101.OverlayValues[18] = d18
			ps101.OverlayValues[19] = d19
			ps101.OverlayValues[20] = d20
			ps101.OverlayValues[21] = d21
			ps101.OverlayValues[22] = d22
			ps101.OverlayValues[23] = d23
			ps101.OverlayValues[24] = d24
			ps101.OverlayValues[25] = d25
			ps101.OverlayValues[26] = d26
			ps101.OverlayValues[27] = d27
			ps101.OverlayValues[28] = d28
			ps101.OverlayValues[29] = d29
			ps101.OverlayValues[30] = d30
			ps101.OverlayValues[31] = d31
			ps101.OverlayValues[32] = d32
			ps101.OverlayValues[33] = d33
			ps101.OverlayValues[34] = d34
			ps101.OverlayValues[35] = d35
			ps101.OverlayValues[36] = d36
			ps101.OverlayValues[37] = d37
			ps101.OverlayValues[38] = d38
			ps101.OverlayValues[39] = d39
			ps101.OverlayValues[40] = d40
			ps101.OverlayValues[41] = d41
			ps101.OverlayValues[42] = d42
			ps101.OverlayValues[43] = d43
			ps101.OverlayValues[44] = d44
			ps101.OverlayValues[45] = d45
			ps101.OverlayValues[46] = d46
			ps101.OverlayValues[47] = d47
			ps101.OverlayValues[48] = d48
			ps101.OverlayValues[49] = d49
			ps101.OverlayValues[50] = d50
			ps101.OverlayValues[57] = d57
			ps101.OverlayValues[58] = d58
			ps101.OverlayValues[59] = d59
			ps101.OverlayValues[60] = d60
			ps101.OverlayValues[61] = d61
			ps101.OverlayValues[62] = d62
			ps101.OverlayValues[63] = d63
			ps101.OverlayValues[64] = d64
			ps101.OverlayValues[65] = d65
			ps101.OverlayValues[66] = d66
			ps101.OverlayValues[67] = d67
			ps101.OverlayValues[68] = d68
			ps101.OverlayValues[69] = d69
			ps101.OverlayValues[70] = d70
			ps101.OverlayValues[71] = d71
			ps101.OverlayValues[72] = d72
			ps101.OverlayValues[73] = d73
			ps101.OverlayValues[74] = d74
			ps101.OverlayValues[75] = d75
			ps101.OverlayValues[76] = d76
			ps101.OverlayValues[77] = d77
			ps101.OverlayValues[78] = d78
			ps101.OverlayValues[79] = d79
			ps101.OverlayValues[80] = d80
			ps101.OverlayValues[81] = d81
			ps101.OverlayValues[82] = d82
			ps101.OverlayValues[83] = d83
			ps101.OverlayValues[84] = d84
			ps101.OverlayValues[85] = d85
			ps101.OverlayValues[86] = d86
			ps101.OverlayValues[87] = d87
			ps101.OverlayValues[88] = d88
			ps101.OverlayValues[89] = d89
			ps101.OverlayValues[90] = d90
			ps101.OverlayValues[91] = d91
			ps101.OverlayValues[92] = d92
			ps101.OverlayValues[93] = d93
			ps101.OverlayValues[94] = d94
			ps101.OverlayValues[95] = d95
			ps101.OverlayValues[96] = d96
			ps101.OverlayValues[97] = d97
			ps101.OverlayValues[98] = d98
			ps101.OverlayValues[99] = d99
			ps101.OverlayValues[100] = d100
					return bbs[8].RenderPS(ps101)
				}
			ps102 := scm.PhiState{General: ps.General}
			ps102.OverlayValues = make([]scm.JITValueDesc, 101)
			ps102.OverlayValues[0] = d0
			ps102.OverlayValues[1] = d1
			ps102.OverlayValues[7] = d7
			ps102.OverlayValues[8] = d8
			ps102.OverlayValues[9] = d9
			ps102.OverlayValues[10] = d10
			ps102.OverlayValues[11] = d11
			ps102.OverlayValues[12] = d12
			ps102.OverlayValues[13] = d13
			ps102.OverlayValues[14] = d14
			ps102.OverlayValues[15] = d15
			ps102.OverlayValues[16] = d16
			ps102.OverlayValues[17] = d17
			ps102.OverlayValues[18] = d18
			ps102.OverlayValues[19] = d19
			ps102.OverlayValues[20] = d20
			ps102.OverlayValues[21] = d21
			ps102.OverlayValues[22] = d22
			ps102.OverlayValues[23] = d23
			ps102.OverlayValues[24] = d24
			ps102.OverlayValues[25] = d25
			ps102.OverlayValues[26] = d26
			ps102.OverlayValues[27] = d27
			ps102.OverlayValues[28] = d28
			ps102.OverlayValues[29] = d29
			ps102.OverlayValues[30] = d30
			ps102.OverlayValues[31] = d31
			ps102.OverlayValues[32] = d32
			ps102.OverlayValues[33] = d33
			ps102.OverlayValues[34] = d34
			ps102.OverlayValues[35] = d35
			ps102.OverlayValues[36] = d36
			ps102.OverlayValues[37] = d37
			ps102.OverlayValues[38] = d38
			ps102.OverlayValues[39] = d39
			ps102.OverlayValues[40] = d40
			ps102.OverlayValues[41] = d41
			ps102.OverlayValues[42] = d42
			ps102.OverlayValues[43] = d43
			ps102.OverlayValues[44] = d44
			ps102.OverlayValues[45] = d45
			ps102.OverlayValues[46] = d46
			ps102.OverlayValues[47] = d47
			ps102.OverlayValues[48] = d48
			ps102.OverlayValues[49] = d49
			ps102.OverlayValues[50] = d50
			ps102.OverlayValues[57] = d57
			ps102.OverlayValues[58] = d58
			ps102.OverlayValues[59] = d59
			ps102.OverlayValues[60] = d60
			ps102.OverlayValues[61] = d61
			ps102.OverlayValues[62] = d62
			ps102.OverlayValues[63] = d63
			ps102.OverlayValues[64] = d64
			ps102.OverlayValues[65] = d65
			ps102.OverlayValues[66] = d66
			ps102.OverlayValues[67] = d67
			ps102.OverlayValues[68] = d68
			ps102.OverlayValues[69] = d69
			ps102.OverlayValues[70] = d70
			ps102.OverlayValues[71] = d71
			ps102.OverlayValues[72] = d72
			ps102.OverlayValues[73] = d73
			ps102.OverlayValues[74] = d74
			ps102.OverlayValues[75] = d75
			ps102.OverlayValues[76] = d76
			ps102.OverlayValues[77] = d77
			ps102.OverlayValues[78] = d78
			ps102.OverlayValues[79] = d79
			ps102.OverlayValues[80] = d80
			ps102.OverlayValues[81] = d81
			ps102.OverlayValues[82] = d82
			ps102.OverlayValues[83] = d83
			ps102.OverlayValues[84] = d84
			ps102.OverlayValues[85] = d85
			ps102.OverlayValues[86] = d86
			ps102.OverlayValues[87] = d87
			ps102.OverlayValues[88] = d88
			ps102.OverlayValues[89] = d89
			ps102.OverlayValues[90] = d90
			ps102.OverlayValues[91] = d91
			ps102.OverlayValues[92] = d92
			ps102.OverlayValues[93] = d93
			ps102.OverlayValues[94] = d94
			ps102.OverlayValues[95] = d95
			ps102.OverlayValues[96] = d96
			ps102.OverlayValues[97] = d97
			ps102.OverlayValues[98] = d98
			ps102.OverlayValues[99] = d99
			ps102.OverlayValues[100] = d100
				return bbs[7].RenderPS(ps102)
			}
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d100.Reg, 0)
			ctx.W.EmitJcc(scm.CcNE, lbl30)
			ctx.W.EmitJmp(lbl31)
			ctx.W.MarkLabel(lbl30)
			ctx.W.EmitJmp(lbl9)
			ctx.W.MarkLabel(lbl31)
			ctx.W.EmitJmp(lbl8)
			ps103 := scm.PhiState{General: true}
			ps103.OverlayValues = make([]scm.JITValueDesc, 101)
			ps103.OverlayValues[0] = d0
			ps103.OverlayValues[1] = d1
			ps103.OverlayValues[7] = d7
			ps103.OverlayValues[8] = d8
			ps103.OverlayValues[9] = d9
			ps103.OverlayValues[10] = d10
			ps103.OverlayValues[11] = d11
			ps103.OverlayValues[12] = d12
			ps103.OverlayValues[13] = d13
			ps103.OverlayValues[14] = d14
			ps103.OverlayValues[15] = d15
			ps103.OverlayValues[16] = d16
			ps103.OverlayValues[17] = d17
			ps103.OverlayValues[18] = d18
			ps103.OverlayValues[19] = d19
			ps103.OverlayValues[20] = d20
			ps103.OverlayValues[21] = d21
			ps103.OverlayValues[22] = d22
			ps103.OverlayValues[23] = d23
			ps103.OverlayValues[24] = d24
			ps103.OverlayValues[25] = d25
			ps103.OverlayValues[26] = d26
			ps103.OverlayValues[27] = d27
			ps103.OverlayValues[28] = d28
			ps103.OverlayValues[29] = d29
			ps103.OverlayValues[30] = d30
			ps103.OverlayValues[31] = d31
			ps103.OverlayValues[32] = d32
			ps103.OverlayValues[33] = d33
			ps103.OverlayValues[34] = d34
			ps103.OverlayValues[35] = d35
			ps103.OverlayValues[36] = d36
			ps103.OverlayValues[37] = d37
			ps103.OverlayValues[38] = d38
			ps103.OverlayValues[39] = d39
			ps103.OverlayValues[40] = d40
			ps103.OverlayValues[41] = d41
			ps103.OverlayValues[42] = d42
			ps103.OverlayValues[43] = d43
			ps103.OverlayValues[44] = d44
			ps103.OverlayValues[45] = d45
			ps103.OverlayValues[46] = d46
			ps103.OverlayValues[47] = d47
			ps103.OverlayValues[48] = d48
			ps103.OverlayValues[49] = d49
			ps103.OverlayValues[50] = d50
			ps103.OverlayValues[57] = d57
			ps103.OverlayValues[58] = d58
			ps103.OverlayValues[59] = d59
			ps103.OverlayValues[60] = d60
			ps103.OverlayValues[61] = d61
			ps103.OverlayValues[62] = d62
			ps103.OverlayValues[63] = d63
			ps103.OverlayValues[64] = d64
			ps103.OverlayValues[65] = d65
			ps103.OverlayValues[66] = d66
			ps103.OverlayValues[67] = d67
			ps103.OverlayValues[68] = d68
			ps103.OverlayValues[69] = d69
			ps103.OverlayValues[70] = d70
			ps103.OverlayValues[71] = d71
			ps103.OverlayValues[72] = d72
			ps103.OverlayValues[73] = d73
			ps103.OverlayValues[74] = d74
			ps103.OverlayValues[75] = d75
			ps103.OverlayValues[76] = d76
			ps103.OverlayValues[77] = d77
			ps103.OverlayValues[78] = d78
			ps103.OverlayValues[79] = d79
			ps103.OverlayValues[80] = d80
			ps103.OverlayValues[81] = d81
			ps103.OverlayValues[82] = d82
			ps103.OverlayValues[83] = d83
			ps103.OverlayValues[84] = d84
			ps103.OverlayValues[85] = d85
			ps103.OverlayValues[86] = d86
			ps103.OverlayValues[87] = d87
			ps103.OverlayValues[88] = d88
			ps103.OverlayValues[89] = d89
			ps103.OverlayValues[90] = d90
			ps103.OverlayValues[91] = d91
			ps103.OverlayValues[92] = d92
			ps103.OverlayValues[93] = d93
			ps103.OverlayValues[94] = d94
			ps103.OverlayValues[95] = d95
			ps103.OverlayValues[96] = d96
			ps103.OverlayValues[97] = d97
			ps103.OverlayValues[98] = d98
			ps103.OverlayValues[99] = d99
			ps103.OverlayValues[100] = d100
			ps104 := scm.PhiState{General: true}
			ps104.OverlayValues = make([]scm.JITValueDesc, 101)
			ps104.OverlayValues[0] = d0
			ps104.OverlayValues[1] = d1
			ps104.OverlayValues[7] = d7
			ps104.OverlayValues[8] = d8
			ps104.OverlayValues[9] = d9
			ps104.OverlayValues[10] = d10
			ps104.OverlayValues[11] = d11
			ps104.OverlayValues[12] = d12
			ps104.OverlayValues[13] = d13
			ps104.OverlayValues[14] = d14
			ps104.OverlayValues[15] = d15
			ps104.OverlayValues[16] = d16
			ps104.OverlayValues[17] = d17
			ps104.OverlayValues[18] = d18
			ps104.OverlayValues[19] = d19
			ps104.OverlayValues[20] = d20
			ps104.OverlayValues[21] = d21
			ps104.OverlayValues[22] = d22
			ps104.OverlayValues[23] = d23
			ps104.OverlayValues[24] = d24
			ps104.OverlayValues[25] = d25
			ps104.OverlayValues[26] = d26
			ps104.OverlayValues[27] = d27
			ps104.OverlayValues[28] = d28
			ps104.OverlayValues[29] = d29
			ps104.OverlayValues[30] = d30
			ps104.OverlayValues[31] = d31
			ps104.OverlayValues[32] = d32
			ps104.OverlayValues[33] = d33
			ps104.OverlayValues[34] = d34
			ps104.OverlayValues[35] = d35
			ps104.OverlayValues[36] = d36
			ps104.OverlayValues[37] = d37
			ps104.OverlayValues[38] = d38
			ps104.OverlayValues[39] = d39
			ps104.OverlayValues[40] = d40
			ps104.OverlayValues[41] = d41
			ps104.OverlayValues[42] = d42
			ps104.OverlayValues[43] = d43
			ps104.OverlayValues[44] = d44
			ps104.OverlayValues[45] = d45
			ps104.OverlayValues[46] = d46
			ps104.OverlayValues[47] = d47
			ps104.OverlayValues[48] = d48
			ps104.OverlayValues[49] = d49
			ps104.OverlayValues[50] = d50
			ps104.OverlayValues[57] = d57
			ps104.OverlayValues[58] = d58
			ps104.OverlayValues[59] = d59
			ps104.OverlayValues[60] = d60
			ps104.OverlayValues[61] = d61
			ps104.OverlayValues[62] = d62
			ps104.OverlayValues[63] = d63
			ps104.OverlayValues[64] = d64
			ps104.OverlayValues[65] = d65
			ps104.OverlayValues[66] = d66
			ps104.OverlayValues[67] = d67
			ps104.OverlayValues[68] = d68
			ps104.OverlayValues[69] = d69
			ps104.OverlayValues[70] = d70
			ps104.OverlayValues[71] = d71
			ps104.OverlayValues[72] = d72
			ps104.OverlayValues[73] = d73
			ps104.OverlayValues[74] = d74
			ps104.OverlayValues[75] = d75
			ps104.OverlayValues[76] = d76
			ps104.OverlayValues[77] = d77
			ps104.OverlayValues[78] = d78
			ps104.OverlayValues[79] = d79
			ps104.OverlayValues[80] = d80
			ps104.OverlayValues[81] = d81
			ps104.OverlayValues[82] = d82
			ps104.OverlayValues[83] = d83
			ps104.OverlayValues[84] = d84
			ps104.OverlayValues[85] = d85
			ps104.OverlayValues[86] = d86
			ps104.OverlayValues[87] = d87
			ps104.OverlayValues[88] = d88
			ps104.OverlayValues[89] = d89
			ps104.OverlayValues[90] = d90
			ps104.OverlayValues[91] = d91
			ps104.OverlayValues[92] = d92
			ps104.OverlayValues[93] = d93
			ps104.OverlayValues[94] = d94
			ps104.OverlayValues[95] = d95
			ps104.OverlayValues[96] = d96
			ps104.OverlayValues[97] = d97
			ps104.OverlayValues[98] = d98
			ps104.OverlayValues[99] = d99
			ps104.OverlayValues[100] = d100
			snap105 := d98
			alloc106 := ctx.SnapshotAllocState()
			if !bbs[7].Rendered {
				bbs[7].RenderPS(ps104)
			}
			ctx.RestoreAllocState(alloc106)
			d98 = snap105
			if !bbs[8].Rendered {
				return bbs[8].RenderPS(ps103)
			}
			return result
			ctx.FreeDesc(&d99)
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
					ctx.W.EmitJmp(lbl4)
					return result
				}
				bbs[3].Rendered = true
				bbs[3].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_3 = bbs[3].Address
				ctx.W.MarkLabel(lbl4)
				ctx.W.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
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
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
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
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != scm.LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != scm.LocNone {
				d70 = ps.OverlayValues[70]
			}
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != scm.LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != scm.LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != scm.LocNone {
				d73 = ps.OverlayValues[73]
			}
			if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != scm.LocNone {
				d74 = ps.OverlayValues[74]
			}
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != scm.LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != scm.LocNone {
				d76 = ps.OverlayValues[76]
			}
			if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
				d77 = ps.OverlayValues[77]
			}
			if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
				d78 = ps.OverlayValues[78]
			}
			if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
				d79 = ps.OverlayValues[79]
			}
			if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
				d80 = ps.OverlayValues[80]
			}
			if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
				d81 = ps.OverlayValues[81]
			}
			if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
				d82 = ps.OverlayValues[82]
			}
			if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
				d83 = ps.OverlayValues[83]
			}
			if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != scm.LocNone {
				d84 = ps.OverlayValues[84]
			}
			if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
				d85 = ps.OverlayValues[85]
			}
			if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
				d86 = ps.OverlayValues[86]
			}
			if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != scm.LocNone {
				d87 = ps.OverlayValues[87]
			}
			if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != scm.LocNone {
				d88 = ps.OverlayValues[88]
			}
			if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != scm.LocNone {
				d89 = ps.OverlayValues[89]
			}
			if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != scm.LocNone {
				d90 = ps.OverlayValues[90]
			}
			if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != scm.LocNone {
				d91 = ps.OverlayValues[91]
			}
			if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != scm.LocNone {
				d92 = ps.OverlayValues[92]
			}
			if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != scm.LocNone {
				d93 = ps.OverlayValues[93]
			}
			if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != scm.LocNone {
				d94 = ps.OverlayValues[94]
			}
			if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != scm.LocNone {
				d95 = ps.OverlayValues[95]
			}
			if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != scm.LocNone {
				d96 = ps.OverlayValues[96]
			}
			if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != scm.LocNone {
				d97 = ps.OverlayValues[97]
			}
			if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != scm.LocNone {
				d98 = ps.OverlayValues[98]
			}
			if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != scm.LocNone {
				d99 = ps.OverlayValues[99]
			}
			if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != scm.LocNone {
				d100 = ps.OverlayValues[100]
			}
			ctx.ReclaimUntrackedRegs()
			d107 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d107)
			ctx.BindReg(r1, &d107)
			ctx.W.EmitMakeNil(d107)
			ctx.W.EmitJmp(lbl0)
			return result
			}
			bbs[4].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[4].VisitCount >= 2 {
					ps.General = true
					return bbs[4].RenderPS(ps)
				}
			}
			bbs[4].VisitCount++
			if ps.General {
				if bbs[4].Rendered {
					ctx.W.EmitJmp(lbl5)
					return result
				}
				bbs[4].Rendered = true
				bbs[4].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_4 = bbs[4].Address
				ctx.W.MarkLabel(lbl5)
				ctx.W.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
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
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
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
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != scm.LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != scm.LocNone {
				d70 = ps.OverlayValues[70]
			}
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != scm.LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != scm.LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != scm.LocNone {
				d73 = ps.OverlayValues[73]
			}
			if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != scm.LocNone {
				d74 = ps.OverlayValues[74]
			}
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != scm.LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != scm.LocNone {
				d76 = ps.OverlayValues[76]
			}
			if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
				d77 = ps.OverlayValues[77]
			}
			if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
				d78 = ps.OverlayValues[78]
			}
			if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
				d79 = ps.OverlayValues[79]
			}
			if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
				d80 = ps.OverlayValues[80]
			}
			if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
				d81 = ps.OverlayValues[81]
			}
			if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
				d82 = ps.OverlayValues[82]
			}
			if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
				d83 = ps.OverlayValues[83]
			}
			if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != scm.LocNone {
				d84 = ps.OverlayValues[84]
			}
			if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
				d85 = ps.OverlayValues[85]
			}
			if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
				d86 = ps.OverlayValues[86]
			}
			if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != scm.LocNone {
				d87 = ps.OverlayValues[87]
			}
			if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != scm.LocNone {
				d88 = ps.OverlayValues[88]
			}
			if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != scm.LocNone {
				d89 = ps.OverlayValues[89]
			}
			if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != scm.LocNone {
				d90 = ps.OverlayValues[90]
			}
			if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != scm.LocNone {
				d91 = ps.OverlayValues[91]
			}
			if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != scm.LocNone {
				d92 = ps.OverlayValues[92]
			}
			if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != scm.LocNone {
				d93 = ps.OverlayValues[93]
			}
			if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != scm.LocNone {
				d94 = ps.OverlayValues[94]
			}
			if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != scm.LocNone {
				d95 = ps.OverlayValues[95]
			}
			if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != scm.LocNone {
				d96 = ps.OverlayValues[96]
			}
			if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != scm.LocNone {
				d97 = ps.OverlayValues[97]
			}
			if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != scm.LocNone {
				d98 = ps.OverlayValues[98]
			}
			if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != scm.LocNone {
				d99 = ps.OverlayValues[99]
			}
			if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != scm.LocNone {
				d100 = ps.OverlayValues[100]
			}
			if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != scm.LocNone {
				d107 = ps.OverlayValues[107]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&idxInt)
			d108 = idxInt
			_ = d108
			r92 := idxInt.Loc == scm.LocReg
			r93 := idxInt.Reg
			if r92 { ctx.ProtectReg(r93) }
			d109 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			lbl32 := ctx.W.ReserveLabel()
			bbpos_3_0 := int32(-1)
			_ = bbpos_3_0
			bbpos_3_1 := int32(-1)
			_ = bbpos_3_1
			bbpos_3_2 := int32(-1)
			_ = bbpos_3_2
			bbpos_3_3 := int32(-1)
			_ = bbpos_3_3
			bbpos_3_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			d109 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d108)
			ctx.EnsureDesc(&d108)
			var d110 scm.JITValueDesc
			if d108.Loc == scm.LocImm {
				d110 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d108.Imm.Int()))))}
			} else {
				r94 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r94, d108.Reg)
				ctx.W.EmitShlRegImm8(r94, 32)
				ctx.W.EmitShrRegImm8(r94, 32)
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
				ctx.BindReg(r94, &d110)
			}
			var d111 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d111 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r95 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r95, thisptr.Reg, off)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r95}
				ctx.BindReg(r95, &d111)
			}
			ctx.EnsureDesc(&d111)
			ctx.EnsureDesc(&d111)
			var d112 scm.JITValueDesc
			if d111.Loc == scm.LocImm {
				d112 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d111.Imm.Int()))))}
			} else {
				r96 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r96, d111.Reg)
				ctx.W.EmitShlRegImm8(r96, 56)
				ctx.W.EmitShrRegImm8(r96, 56)
				d112 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
				ctx.BindReg(r96, &d112)
			}
			ctx.FreeDesc(&d111)
			ctx.EnsureDesc(&d110)
			ctx.EnsureDesc(&d112)
			ctx.EnsureDesc(&d110)
			ctx.EnsureDesc(&d112)
			ctx.EnsureDesc(&d110)
			ctx.EnsureDesc(&d112)
			var d113 scm.JITValueDesc
			if d110.Loc == scm.LocImm && d112.Loc == scm.LocImm {
				d113 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d110.Imm.Int() * d112.Imm.Int())}
			} else if d110.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d112.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d110.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d112.Reg)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d113)
			} else if d112.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d110.Reg)
				ctx.W.EmitMovRegReg(scratch, d110.Reg)
				if d112.Imm.Int() >= -2147483648 && d112.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d112.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d112.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d113)
			} else {
				r97 := ctx.AllocRegExcept(d110.Reg, d112.Reg)
				ctx.W.EmitMovRegReg(r97, d110.Reg)
				ctx.W.EmitImulInt64(r97, d112.Reg)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
				ctx.BindReg(r97, &d113)
			}
			if d113.Loc == scm.LocReg && d110.Loc == scm.LocReg && d113.Reg == d110.Reg {
				ctx.TransferReg(d110.Reg)
				d110.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d110)
			ctx.FreeDesc(&d112)
			var d114 scm.JITValueDesc
			r98 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r98, uint64(dataPtr))
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r98, StackOff: int32(sliceLen)}
				ctx.BindReg(r98, &d114)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 0)
				ctx.W.EmitMovRegMem(r98, thisptr.Reg, off)
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r98}
				ctx.BindReg(r98, &d114)
			}
			ctx.BindReg(r98, &d114)
			ctx.EnsureDesc(&d113)
			var d115 scm.JITValueDesc
			if d113.Loc == scm.LocImm {
				d115 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d113.Imm.Int() / 64)}
			} else {
				r99 := ctx.AllocRegExcept(d113.Reg)
				ctx.W.EmitMovRegReg(r99, d113.Reg)
				ctx.W.EmitShrRegImm8(r99, 6)
				d115 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
				ctx.BindReg(r99, &d115)
			}
			if d115.Loc == scm.LocReg && d113.Loc == scm.LocReg && d115.Reg == d113.Reg {
				ctx.TransferReg(d113.Reg)
				d113.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d115)
			r100 := ctx.AllocReg()
			ctx.EnsureDesc(&d115)
			ctx.EnsureDesc(&d114)
			if d115.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r100, uint64(d115.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r100, d115.Reg)
				ctx.W.EmitShlRegImm8(r100, 3)
			}
			if d114.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d114.Imm.Int()))
				ctx.W.EmitAddInt64(r100, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r100, d114.Reg)
			}
			r101 := ctx.AllocRegExcept(r100)
			ctx.W.EmitMovRegMem(r101, r100, 0)
			ctx.FreeReg(r100)
			d116 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r101}
			ctx.BindReg(r101, &d116)
			ctx.FreeDesc(&d115)
			ctx.EnsureDesc(&d113)
			var d117 scm.JITValueDesc
			if d113.Loc == scm.LocImm {
				d117 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d113.Imm.Int() % 64)}
			} else {
				r102 := ctx.AllocRegExcept(d113.Reg)
				ctx.W.EmitMovRegReg(r102, d113.Reg)
				ctx.W.EmitAndRegImm32(r102, 63)
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r102}
				ctx.BindReg(r102, &d117)
			}
			if d117.Loc == scm.LocReg && d113.Loc == scm.LocReg && d117.Reg == d113.Reg {
				ctx.TransferReg(d113.Reg)
				d113.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d116)
			ctx.EnsureDesc(&d117)
			var d118 scm.JITValueDesc
			if d116.Loc == scm.LocImm && d117.Loc == scm.LocImm {
				d118 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d116.Imm.Int()) << uint64(d117.Imm.Int())))}
			} else if d117.Loc == scm.LocImm {
				r103 := ctx.AllocRegExcept(d116.Reg)
				ctx.W.EmitMovRegReg(r103, d116.Reg)
				ctx.W.EmitShlRegImm8(r103, uint8(d117.Imm.Int()))
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r103}
				ctx.BindReg(r103, &d118)
			} else {
				{
					shiftSrc := d116.Reg
					r104 := ctx.AllocRegExcept(d116.Reg)
					ctx.W.EmitMovRegReg(r104, d116.Reg)
					shiftSrc = r104
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d117.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d117.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d117.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d118)
				}
			}
			if d118.Loc == scm.LocReg && d116.Loc == scm.LocReg && d118.Reg == d116.Reg {
				ctx.TransferReg(d116.Reg)
				d116.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d116)
			ctx.FreeDesc(&d117)
			var d119 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 25)
				r105 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r105, thisptr.Reg, off)
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r105}
				ctx.BindReg(r105, &d119)
			}
			d120 = d119
			ctx.EnsureDesc(&d120)
			if d120.Loc != scm.LocImm && d120.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl33 := ctx.W.ReserveLabel()
			lbl34 := ctx.W.ReserveLabel()
			lbl35 := ctx.W.ReserveLabel()
			lbl36 := ctx.W.ReserveLabel()
			if d120.Loc == scm.LocImm {
				if d120.Imm.Bool() {
					ctx.W.MarkLabel(lbl35)
					ctx.W.EmitJmp(lbl33)
				} else {
					ctx.W.MarkLabel(lbl36)
			d121 = d118
			if d121.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d121)
			ctx.EmitStoreToStack(d121, 16)
					ctx.W.EmitJmp(lbl34)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d120.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl35)
				ctx.W.EmitJmp(lbl36)
				ctx.W.MarkLabel(lbl35)
				ctx.W.EmitJmp(lbl33)
				ctx.W.MarkLabel(lbl36)
			d122 = d118
			if d122.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d122)
			ctx.EmitStoreToStack(d122, 16)
				ctx.W.EmitJmp(lbl34)
			}
			ctx.FreeDesc(&d119)
			bbpos_3_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl34)
			ctx.W.ResolveFixups()
			d109 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			var d123 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r106 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r106, thisptr.Reg, off)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r106}
				ctx.BindReg(r106, &d123)
			}
			ctx.EnsureDesc(&d123)
			ctx.EnsureDesc(&d123)
			var d124 scm.JITValueDesc
			if d123.Loc == scm.LocImm {
				d124 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d123.Imm.Int()))))}
			} else {
				r107 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r107, d123.Reg)
				ctx.W.EmitShlRegImm8(r107, 56)
				ctx.W.EmitShrRegImm8(r107, 56)
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r107}
				ctx.BindReg(r107, &d124)
			}
			ctx.FreeDesc(&d123)
			d125 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d124)
			ctx.EnsureDesc(&d125)
			ctx.EnsureDesc(&d124)
			ctx.EnsureDesc(&d125)
			ctx.EnsureDesc(&d124)
			var d126 scm.JITValueDesc
			if d125.Loc == scm.LocImm && d124.Loc == scm.LocImm {
				d126 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d125.Imm.Int() - d124.Imm.Int())}
			} else if d124.Loc == scm.LocImm && d124.Imm.Int() == 0 {
				r108 := ctx.AllocRegExcept(d125.Reg)
				ctx.W.EmitMovRegReg(r108, d125.Reg)
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
				ctx.BindReg(r108, &d126)
			} else if d125.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d124.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d125.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d124.Reg)
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d126)
			} else if d124.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d125.Reg)
				ctx.W.EmitMovRegReg(scratch, d125.Reg)
				if d124.Imm.Int() >= -2147483648 && d124.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d124.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d124.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d126)
			} else {
				r109 := ctx.AllocRegExcept(d125.Reg, d124.Reg)
				ctx.W.EmitMovRegReg(r109, d125.Reg)
				ctx.W.EmitSubInt64(r109, d124.Reg)
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r109}
				ctx.BindReg(r109, &d126)
			}
			if d126.Loc == scm.LocReg && d125.Loc == scm.LocReg && d126.Reg == d125.Reg {
				ctx.TransferReg(d125.Reg)
				d125.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d124)
			ctx.EnsureDesc(&d109)
			ctx.EnsureDesc(&d126)
			var d127 scm.JITValueDesc
			if d109.Loc == scm.LocImm && d126.Loc == scm.LocImm {
				d127 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d109.Imm.Int()) >> uint64(d126.Imm.Int())))}
			} else if d126.Loc == scm.LocImm {
				r110 := ctx.AllocRegExcept(d109.Reg)
				ctx.W.EmitMovRegReg(r110, d109.Reg)
				ctx.W.EmitShrRegImm8(r110, uint8(d126.Imm.Int()))
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
				ctx.BindReg(r110, &d127)
			} else {
				{
					shiftSrc := d109.Reg
					r111 := ctx.AllocRegExcept(d109.Reg)
					ctx.W.EmitMovRegReg(r111, d109.Reg)
					shiftSrc = r111
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d126.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d126.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d126.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d127)
				}
			}
			if d127.Loc == scm.LocReg && d109.Loc == scm.LocReg && d127.Reg == d109.Reg {
				ctx.TransferReg(d109.Reg)
				d109.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d109)
			ctx.FreeDesc(&d126)
			r112 := ctx.AllocReg()
			ctx.EnsureDesc(&d127)
			ctx.EnsureDesc(&d127)
			if d127.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r112, d127)
			}
			ctx.W.EmitJmp(lbl32)
			bbpos_3_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl33)
			ctx.W.ResolveFixups()
			d109 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d113)
			var d128 scm.JITValueDesc
			if d113.Loc == scm.LocImm {
				d128 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d113.Imm.Int() % 64)}
			} else {
				r113 := ctx.AllocRegExcept(d113.Reg)
				ctx.W.EmitMovRegReg(r113, d113.Reg)
				ctx.W.EmitAndRegImm32(r113, 63)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r113}
				ctx.BindReg(r113, &d128)
			}
			if d128.Loc == scm.LocReg && d113.Loc == scm.LocReg && d128.Reg == d113.Reg {
				ctx.TransferReg(d113.Reg)
				d113.Loc = scm.LocNone
			}
			var d129 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d129 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r114 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r114, thisptr.Reg, off)
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r114}
				ctx.BindReg(r114, &d129)
			}
			ctx.EnsureDesc(&d129)
			ctx.EnsureDesc(&d129)
			var d130 scm.JITValueDesc
			if d129.Loc == scm.LocImm {
				d130 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d129.Imm.Int()))))}
			} else {
				r115 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r115, d129.Reg)
				ctx.W.EmitShlRegImm8(r115, 56)
				ctx.W.EmitShrRegImm8(r115, 56)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
				ctx.BindReg(r115, &d130)
			}
			ctx.FreeDesc(&d129)
			ctx.EnsureDesc(&d128)
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d128)
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d128)
			ctx.EnsureDesc(&d130)
			var d131 scm.JITValueDesc
			if d128.Loc == scm.LocImm && d130.Loc == scm.LocImm {
				d131 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d128.Imm.Int() + d130.Imm.Int())}
			} else if d130.Loc == scm.LocImm && d130.Imm.Int() == 0 {
				r116 := ctx.AllocRegExcept(d128.Reg)
				ctx.W.EmitMovRegReg(r116, d128.Reg)
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r116}
				ctx.BindReg(r116, &d131)
			} else if d128.Loc == scm.LocImm && d128.Imm.Int() == 0 {
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d130.Reg}
				ctx.BindReg(d130.Reg, &d131)
			} else if d128.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d130.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d128.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d130.Reg)
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d131)
			} else if d130.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d128.Reg)
				ctx.W.EmitMovRegReg(scratch, d128.Reg)
				if d130.Imm.Int() >= -2147483648 && d130.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d130.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d130.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d131)
			} else {
				r117 := ctx.AllocRegExcept(d128.Reg, d130.Reg)
				ctx.W.EmitMovRegReg(r117, d128.Reg)
				ctx.W.EmitAddInt64(r117, d130.Reg)
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
				ctx.BindReg(r117, &d131)
			}
			if d131.Loc == scm.LocReg && d128.Loc == scm.LocReg && d131.Reg == d128.Reg {
				ctx.TransferReg(d128.Reg)
				d128.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d128)
			ctx.FreeDesc(&d130)
			ctx.EnsureDesc(&d131)
			var d132 scm.JITValueDesc
			if d131.Loc == scm.LocImm {
				d132 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d131.Imm.Int()) > uint64(64))}
			} else {
				r118 := ctx.AllocRegExcept(d131.Reg)
				ctx.W.EmitCmpRegImm32(d131.Reg, 64)
				ctx.W.EmitSetcc(r118, scm.CcA)
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r118}
				ctx.BindReg(r118, &d132)
			}
			ctx.FreeDesc(&d131)
			d133 = d132
			ctx.EnsureDesc(&d133)
			if d133.Loc != scm.LocImm && d133.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl37 := ctx.W.ReserveLabel()
			lbl38 := ctx.W.ReserveLabel()
			lbl39 := ctx.W.ReserveLabel()
			if d133.Loc == scm.LocImm {
				if d133.Imm.Bool() {
					ctx.W.MarkLabel(lbl38)
					ctx.W.EmitJmp(lbl37)
				} else {
					ctx.W.MarkLabel(lbl39)
			d134 = d118
			if d134.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d134)
			ctx.EmitStoreToStack(d134, 16)
					ctx.W.EmitJmp(lbl34)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d133.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl38)
				ctx.W.EmitJmp(lbl39)
				ctx.W.MarkLabel(lbl38)
				ctx.W.EmitJmp(lbl37)
				ctx.W.MarkLabel(lbl39)
			d135 = d118
			if d135.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d135)
			ctx.EmitStoreToStack(d135, 16)
				ctx.W.EmitJmp(lbl34)
			}
			ctx.FreeDesc(&d132)
			bbpos_3_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl37)
			ctx.W.ResolveFixups()
			d109 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d113)
			var d136 scm.JITValueDesc
			if d113.Loc == scm.LocImm {
				d136 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d113.Imm.Int() / 64)}
			} else {
				r119 := ctx.AllocRegExcept(d113.Reg)
				ctx.W.EmitMovRegReg(r119, d113.Reg)
				ctx.W.EmitShrRegImm8(r119, 6)
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
				ctx.BindReg(r119, &d136)
			}
			if d136.Loc == scm.LocReg && d113.Loc == scm.LocReg && d136.Reg == d113.Reg {
				ctx.TransferReg(d113.Reg)
				d113.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d136)
			ctx.EnsureDesc(&d136)
			var d137 scm.JITValueDesc
			if d136.Loc == scm.LocImm {
				d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d136.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d136.Reg)
				ctx.W.EmitMovRegReg(scratch, d136.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d137)
			}
			if d137.Loc == scm.LocReg && d136.Loc == scm.LocReg && d137.Reg == d136.Reg {
				ctx.TransferReg(d136.Reg)
				d136.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d136)
			ctx.EnsureDesc(&d137)
			r120 := ctx.AllocReg()
			ctx.EnsureDesc(&d137)
			ctx.EnsureDesc(&d114)
			if d137.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r120, uint64(d137.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r120, d137.Reg)
				ctx.W.EmitShlRegImm8(r120, 3)
			}
			if d114.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d114.Imm.Int()))
				ctx.W.EmitAddInt64(r120, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r120, d114.Reg)
			}
			r121 := ctx.AllocRegExcept(r120)
			ctx.W.EmitMovRegMem(r121, r120, 0)
			ctx.FreeReg(r120)
			d138 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r121}
			ctx.BindReg(r121, &d138)
			ctx.FreeDesc(&d137)
			ctx.EnsureDesc(&d113)
			var d139 scm.JITValueDesc
			if d113.Loc == scm.LocImm {
				d139 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d113.Imm.Int() % 64)}
			} else {
				r122 := ctx.AllocRegExcept(d113.Reg)
				ctx.W.EmitMovRegReg(r122, d113.Reg)
				ctx.W.EmitAndRegImm32(r122, 63)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
				ctx.BindReg(r122, &d139)
			}
			if d139.Loc == scm.LocReg && d113.Loc == scm.LocReg && d139.Reg == d113.Reg {
				ctx.TransferReg(d113.Reg)
				d113.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d113)
			d140 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d139)
			ctx.EnsureDesc(&d140)
			ctx.EnsureDesc(&d139)
			ctx.EnsureDesc(&d140)
			ctx.EnsureDesc(&d139)
			var d141 scm.JITValueDesc
			if d140.Loc == scm.LocImm && d139.Loc == scm.LocImm {
				d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d140.Imm.Int() - d139.Imm.Int())}
			} else if d139.Loc == scm.LocImm && d139.Imm.Int() == 0 {
				r123 := ctx.AllocRegExcept(d140.Reg)
				ctx.W.EmitMovRegReg(r123, d140.Reg)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
				ctx.BindReg(r123, &d141)
			} else if d140.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d139.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d140.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d139.Reg)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d141)
			} else if d139.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d140.Reg)
				ctx.W.EmitMovRegReg(scratch, d140.Reg)
				if d139.Imm.Int() >= -2147483648 && d139.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d139.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d139.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d141)
			} else {
				r124 := ctx.AllocRegExcept(d140.Reg, d139.Reg)
				ctx.W.EmitMovRegReg(r124, d140.Reg)
				ctx.W.EmitSubInt64(r124, d139.Reg)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
				ctx.BindReg(r124, &d141)
			}
			if d141.Loc == scm.LocReg && d140.Loc == scm.LocReg && d141.Reg == d140.Reg {
				ctx.TransferReg(d140.Reg)
				d140.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d139)
			ctx.EnsureDesc(&d138)
			ctx.EnsureDesc(&d141)
			var d142 scm.JITValueDesc
			if d138.Loc == scm.LocImm && d141.Loc == scm.LocImm {
				d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d138.Imm.Int()) >> uint64(d141.Imm.Int())))}
			} else if d141.Loc == scm.LocImm {
				r125 := ctx.AllocRegExcept(d138.Reg)
				ctx.W.EmitMovRegReg(r125, d138.Reg)
				ctx.W.EmitShrRegImm8(r125, uint8(d141.Imm.Int()))
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
				ctx.BindReg(r125, &d142)
			} else {
				{
					shiftSrc := d138.Reg
					r126 := ctx.AllocRegExcept(d138.Reg)
					ctx.W.EmitMovRegReg(r126, d138.Reg)
					shiftSrc = r126
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d141.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d141.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d141.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d142)
				}
			}
			if d142.Loc == scm.LocReg && d138.Loc == scm.LocReg && d142.Reg == d138.Reg {
				ctx.TransferReg(d138.Reg)
				d138.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d138)
			ctx.FreeDesc(&d141)
			ctx.EnsureDesc(&d118)
			ctx.EnsureDesc(&d142)
			var d143 scm.JITValueDesc
			if d118.Loc == scm.LocImm && d142.Loc == scm.LocImm {
				d143 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d118.Imm.Int() | d142.Imm.Int())}
			} else if d118.Loc == scm.LocImm && d118.Imm.Int() == 0 {
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d142.Reg}
				ctx.BindReg(d142.Reg, &d143)
			} else if d142.Loc == scm.LocImm && d142.Imm.Int() == 0 {
				r127 := ctx.AllocRegExcept(d118.Reg)
				ctx.W.EmitMovRegReg(r127, d118.Reg)
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
				ctx.BindReg(r127, &d143)
			} else if d118.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d142.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d118.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d142.Reg)
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d143)
			} else if d142.Loc == scm.LocImm {
				r128 := ctx.AllocRegExcept(d118.Reg)
				ctx.W.EmitMovRegReg(r128, d118.Reg)
				if d142.Imm.Int() >= -2147483648 && d142.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r128, int32(d142.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d142.Imm.Int()))
					ctx.W.EmitOrInt64(r128, scm.RegR11)
				}
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
				ctx.BindReg(r128, &d143)
			} else {
				r129 := ctx.AllocRegExcept(d118.Reg, d142.Reg)
				ctx.W.EmitMovRegReg(r129, d118.Reg)
				ctx.W.EmitOrInt64(r129, d142.Reg)
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
				ctx.BindReg(r129, &d143)
			}
			if d143.Loc == scm.LocReg && d118.Loc == scm.LocReg && d143.Reg == d118.Reg {
				ctx.TransferReg(d118.Reg)
				d118.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d142)
			d144 = d143
			if d144.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d144)
			ctx.EmitStoreToStack(d144, 16)
			ctx.W.EmitJmp(lbl34)
			ctx.W.MarkLabel(lbl32)
			d145 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r112}
			ctx.BindReg(r112, &d145)
			ctx.BindReg(r112, &d145)
			if r92 { ctx.UnprotectReg(r93) }
			ctx.FreeDesc(&idxInt)
			ctx.EnsureDesc(&d145)
			ctx.EnsureDesc(&d145)
			var d146 scm.JITValueDesc
			if d145.Loc == scm.LocImm {
				d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d145.Imm.Int()))))}
			} else {
				r130 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r130, d145.Reg)
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
				ctx.BindReg(r130, &d146)
			}
			ctx.FreeDesc(&d145)
			var d147 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d147 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 32)
				r131 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r131, thisptr.Reg, off)
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r131}
				ctx.BindReg(r131, &d147)
			}
			ctx.EnsureDesc(&d146)
			ctx.EnsureDesc(&d147)
			ctx.EnsureDesc(&d146)
			ctx.EnsureDesc(&d147)
			ctx.EnsureDesc(&d146)
			ctx.EnsureDesc(&d147)
			var d148 scm.JITValueDesc
			if d146.Loc == scm.LocImm && d147.Loc == scm.LocImm {
				d148 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d146.Imm.Int() + d147.Imm.Int())}
			} else if d147.Loc == scm.LocImm && d147.Imm.Int() == 0 {
				r132 := ctx.AllocRegExcept(d146.Reg)
				ctx.W.EmitMovRegReg(r132, d146.Reg)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
				ctx.BindReg(r132, &d148)
			} else if d146.Loc == scm.LocImm && d146.Imm.Int() == 0 {
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d147.Reg}
				ctx.BindReg(d147.Reg, &d148)
			} else if d146.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d147.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d146.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d147.Reg)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d148)
			} else if d147.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d146.Reg)
				ctx.W.EmitMovRegReg(scratch, d146.Reg)
				if d147.Imm.Int() >= -2147483648 && d147.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d147.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d147.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d148)
			} else {
				r133 := ctx.AllocRegExcept(d146.Reg, d147.Reg)
				ctx.W.EmitMovRegReg(r133, d146.Reg)
				ctx.W.EmitAddInt64(r133, d147.Reg)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
				ctx.BindReg(r133, &d148)
			}
			if d148.Loc == scm.LocReg && d146.Loc == scm.LocReg && d148.Reg == d146.Reg {
				ctx.TransferReg(d146.Reg)
				d146.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d146)
			ctx.FreeDesc(&d147)
			ctx.EnsureDesc(&d148)
			ctx.EnsureDesc(&d148)
			var d149 scm.JITValueDesc
			if d148.Loc == scm.LocImm {
				d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d148.Imm.Int()))))}
			} else {
				r134 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r134, d148.Reg)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r134}
				ctx.BindReg(r134, &d149)
			}
			ctx.FreeDesc(&d148)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d48)
			var d150 scm.JITValueDesc
			if d48.Loc == scm.LocImm {
				d150 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d48.Imm.Int()))))}
			} else {
				r135 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r135, d48.Reg)
				d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r135}
				ctx.BindReg(r135, &d150)
			}
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d149)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d149)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d149)
			var d151 scm.JITValueDesc
			if d48.Loc == scm.LocImm && d149.Loc == scm.LocImm {
				d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d48.Imm.Int() + d149.Imm.Int())}
			} else if d149.Loc == scm.LocImm && d149.Imm.Int() == 0 {
				r136 := ctx.AllocRegExcept(d48.Reg)
				ctx.W.EmitMovRegReg(r136, d48.Reg)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r136}
				ctx.BindReg(r136, &d151)
			} else if d48.Loc == scm.LocImm && d48.Imm.Int() == 0 {
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d149.Reg}
				ctx.BindReg(d149.Reg, &d151)
			} else if d48.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d149.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d48.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d149.Reg)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d151)
			} else if d149.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d48.Reg)
				ctx.W.EmitMovRegReg(scratch, d48.Reg)
				if d149.Imm.Int() >= -2147483648 && d149.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d149.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d149.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d151)
			} else {
				r137 := ctx.AllocRegExcept(d48.Reg, d149.Reg)
				ctx.W.EmitMovRegReg(r137, d48.Reg)
				ctx.W.EmitAddInt64(r137, d149.Reg)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r137}
				ctx.BindReg(r137, &d151)
			}
			if d151.Loc == scm.LocReg && d48.Loc == scm.LocReg && d151.Reg == d48.Reg {
				ctx.TransferReg(d48.Reg)
				d48.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d149)
			ctx.EnsureDesc(&d151)
			ctx.EnsureDesc(&d151)
			var d152 scm.JITValueDesc
			if d151.Loc == scm.LocImm {
				d152 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d151.Imm.Int()))))}
			} else {
				r138 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r138, d151.Reg)
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
				ctx.BindReg(r138, &d152)
			}
			ctx.FreeDesc(&d151)
			var d153 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).dictionary)
				r139 := ctx.AllocReg()
				r140 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r139, fieldAddr)
				ctx.W.EmitMovRegMem64(r140, fieldAddr+8)
				d153 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r139, Reg2: r140}
				ctx.BindReg(r139, &d153)
				ctx.BindReg(r140, &d153)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).dictionary))
				r141 := ctx.AllocReg()
				r142 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r141, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r142, thisptr.Reg, off+8)
				d153 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r141, Reg2: r142}
				ctx.BindReg(r141, &d153)
				ctx.BindReg(r142, &d153)
			}
			ctx.EnsureDesc(&d150)
			ctx.EnsureDesc(&d152)
			ctx.EnsureDesc(&d153)
			ctx.EnsureDesc(&d150)
			ctx.EnsureDesc(&d152)
			r143 := ctx.AllocReg()
			r144 := ctx.AllocRegExcept(r143)
			ctx.EnsureDesc(&d153)
			ctx.EnsureDesc(&d150)
			ctx.EnsureDesc(&d152)
			if d153.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r143, uint64(d153.Imm.Int()))
			} else if d153.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r143, d153.Reg)
			} else {
				ctx.W.EmitMovRegReg(r143, d153.Reg)
			}
			if d150.Loc == scm.LocImm {
				if d150.Imm.Int() != 0 {
					if d150.Imm.Int() >= -2147483648 && d150.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r143, int32(d150.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d150.Imm.Int()))
						ctx.W.EmitAddInt64(r143, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r143, d150.Reg)
			}
			if d152.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r144, uint64(d152.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r144, d152.Reg)
			}
			if d150.Loc == scm.LocImm {
				if d150.Imm.Int() >= -2147483648 && d150.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r144, int32(d150.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d150.Imm.Int()))
					ctx.W.EmitSubInt64(r144, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r144, d150.Reg)
			}
			d154 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r143, Reg2: r144}
			ctx.BindReg(r143, &d154)
			ctx.BindReg(r144, &d154)
			ctx.FreeDesc(&d150)
			ctx.FreeDesc(&d152)
			d155 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d155)
			ctx.BindReg(r1, &d155)
			d156 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d154}, 2)
			ctx.EmitMovPairToResult(&d156, &d155)
			ctx.W.EmitJmp(lbl0)
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
					ctx.W.EmitJmp(lbl6)
					return result
				}
				bbs[5].Rendered = true
				bbs[5].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_5 = bbs[5].Address
				ctx.W.MarkLabel(lbl6)
				ctx.W.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
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
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
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
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != scm.LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != scm.LocNone {
				d70 = ps.OverlayValues[70]
			}
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != scm.LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != scm.LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != scm.LocNone {
				d73 = ps.OverlayValues[73]
			}
			if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != scm.LocNone {
				d74 = ps.OverlayValues[74]
			}
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != scm.LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != scm.LocNone {
				d76 = ps.OverlayValues[76]
			}
			if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
				d77 = ps.OverlayValues[77]
			}
			if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
				d78 = ps.OverlayValues[78]
			}
			if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
				d79 = ps.OverlayValues[79]
			}
			if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
				d80 = ps.OverlayValues[80]
			}
			if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
				d81 = ps.OverlayValues[81]
			}
			if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
				d82 = ps.OverlayValues[82]
			}
			if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
				d83 = ps.OverlayValues[83]
			}
			if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != scm.LocNone {
				d84 = ps.OverlayValues[84]
			}
			if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
				d85 = ps.OverlayValues[85]
			}
			if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
				d86 = ps.OverlayValues[86]
			}
			if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != scm.LocNone {
				d87 = ps.OverlayValues[87]
			}
			if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != scm.LocNone {
				d88 = ps.OverlayValues[88]
			}
			if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != scm.LocNone {
				d89 = ps.OverlayValues[89]
			}
			if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != scm.LocNone {
				d90 = ps.OverlayValues[90]
			}
			if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != scm.LocNone {
				d91 = ps.OverlayValues[91]
			}
			if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != scm.LocNone {
				d92 = ps.OverlayValues[92]
			}
			if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != scm.LocNone {
				d93 = ps.OverlayValues[93]
			}
			if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != scm.LocNone {
				d94 = ps.OverlayValues[94]
			}
			if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != scm.LocNone {
				d95 = ps.OverlayValues[95]
			}
			if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != scm.LocNone {
				d96 = ps.OverlayValues[96]
			}
			if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != scm.LocNone {
				d97 = ps.OverlayValues[97]
			}
			if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != scm.LocNone {
				d98 = ps.OverlayValues[98]
			}
			if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != scm.LocNone {
				d99 = ps.OverlayValues[99]
			}
			if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != scm.LocNone {
				d100 = ps.OverlayValues[100]
			}
			if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != scm.LocNone {
				d107 = ps.OverlayValues[107]
			}
			if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != scm.LocNone {
				d108 = ps.OverlayValues[108]
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
			ctx.ReclaimUntrackedRegs()
			var d157 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d157 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 64)
				r145 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r145, thisptr.Reg, off)
				d157 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r145}
				ctx.BindReg(r145, &d157)
			}
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d157)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d157)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d157)
			var d158 scm.JITValueDesc
			if d48.Loc == scm.LocImm && d157.Loc == scm.LocImm {
				d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d48.Imm.Int()) == uint64(d157.Imm.Int()))}
			} else if d157.Loc == scm.LocImm {
				r146 := ctx.AllocRegExcept(d48.Reg)
				if d157.Imm.Int() >= -2147483648 && d157.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d48.Reg, int32(d157.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d157.Imm.Int()))
					ctx.W.EmitCmpInt64(d48.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r146, scm.CcE)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r146}
				ctx.BindReg(r146, &d158)
			} else if d48.Loc == scm.LocImm {
				r147 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d48.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d157.Reg)
				ctx.W.EmitSetcc(r147, scm.CcE)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r147}
				ctx.BindReg(r147, &d158)
			} else {
				r148 := ctx.AllocRegExcept(d48.Reg)
				ctx.W.EmitCmpInt64(d48.Reg, d157.Reg)
				ctx.W.EmitSetcc(r148, scm.CcE)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r148}
				ctx.BindReg(r148, &d158)
			}
			ctx.FreeDesc(&d48)
			ctx.FreeDesc(&d157)
			d159 = d158
			ctx.EnsureDesc(&d159)
			if d159.Loc != scm.LocImm && d159.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d159.Loc == scm.LocImm {
				if d159.Imm.Bool() {
			ps160 := scm.PhiState{General: ps.General}
			ps160.OverlayValues = make([]scm.JITValueDesc, 160)
			ps160.OverlayValues[0] = d0
			ps160.OverlayValues[1] = d1
			ps160.OverlayValues[7] = d7
			ps160.OverlayValues[8] = d8
			ps160.OverlayValues[9] = d9
			ps160.OverlayValues[10] = d10
			ps160.OverlayValues[11] = d11
			ps160.OverlayValues[12] = d12
			ps160.OverlayValues[13] = d13
			ps160.OverlayValues[14] = d14
			ps160.OverlayValues[15] = d15
			ps160.OverlayValues[16] = d16
			ps160.OverlayValues[17] = d17
			ps160.OverlayValues[18] = d18
			ps160.OverlayValues[19] = d19
			ps160.OverlayValues[20] = d20
			ps160.OverlayValues[21] = d21
			ps160.OverlayValues[22] = d22
			ps160.OverlayValues[23] = d23
			ps160.OverlayValues[24] = d24
			ps160.OverlayValues[25] = d25
			ps160.OverlayValues[26] = d26
			ps160.OverlayValues[27] = d27
			ps160.OverlayValues[28] = d28
			ps160.OverlayValues[29] = d29
			ps160.OverlayValues[30] = d30
			ps160.OverlayValues[31] = d31
			ps160.OverlayValues[32] = d32
			ps160.OverlayValues[33] = d33
			ps160.OverlayValues[34] = d34
			ps160.OverlayValues[35] = d35
			ps160.OverlayValues[36] = d36
			ps160.OverlayValues[37] = d37
			ps160.OverlayValues[38] = d38
			ps160.OverlayValues[39] = d39
			ps160.OverlayValues[40] = d40
			ps160.OverlayValues[41] = d41
			ps160.OverlayValues[42] = d42
			ps160.OverlayValues[43] = d43
			ps160.OverlayValues[44] = d44
			ps160.OverlayValues[45] = d45
			ps160.OverlayValues[46] = d46
			ps160.OverlayValues[47] = d47
			ps160.OverlayValues[48] = d48
			ps160.OverlayValues[49] = d49
			ps160.OverlayValues[50] = d50
			ps160.OverlayValues[57] = d57
			ps160.OverlayValues[58] = d58
			ps160.OverlayValues[59] = d59
			ps160.OverlayValues[60] = d60
			ps160.OverlayValues[61] = d61
			ps160.OverlayValues[62] = d62
			ps160.OverlayValues[63] = d63
			ps160.OverlayValues[64] = d64
			ps160.OverlayValues[65] = d65
			ps160.OverlayValues[66] = d66
			ps160.OverlayValues[67] = d67
			ps160.OverlayValues[68] = d68
			ps160.OverlayValues[69] = d69
			ps160.OverlayValues[70] = d70
			ps160.OverlayValues[71] = d71
			ps160.OverlayValues[72] = d72
			ps160.OverlayValues[73] = d73
			ps160.OverlayValues[74] = d74
			ps160.OverlayValues[75] = d75
			ps160.OverlayValues[76] = d76
			ps160.OverlayValues[77] = d77
			ps160.OverlayValues[78] = d78
			ps160.OverlayValues[79] = d79
			ps160.OverlayValues[80] = d80
			ps160.OverlayValues[81] = d81
			ps160.OverlayValues[82] = d82
			ps160.OverlayValues[83] = d83
			ps160.OverlayValues[84] = d84
			ps160.OverlayValues[85] = d85
			ps160.OverlayValues[86] = d86
			ps160.OverlayValues[87] = d87
			ps160.OverlayValues[88] = d88
			ps160.OverlayValues[89] = d89
			ps160.OverlayValues[90] = d90
			ps160.OverlayValues[91] = d91
			ps160.OverlayValues[92] = d92
			ps160.OverlayValues[93] = d93
			ps160.OverlayValues[94] = d94
			ps160.OverlayValues[95] = d95
			ps160.OverlayValues[96] = d96
			ps160.OverlayValues[97] = d97
			ps160.OverlayValues[98] = d98
			ps160.OverlayValues[99] = d99
			ps160.OverlayValues[100] = d100
			ps160.OverlayValues[107] = d107
			ps160.OverlayValues[108] = d108
			ps160.OverlayValues[109] = d109
			ps160.OverlayValues[110] = d110
			ps160.OverlayValues[111] = d111
			ps160.OverlayValues[112] = d112
			ps160.OverlayValues[113] = d113
			ps160.OverlayValues[114] = d114
			ps160.OverlayValues[115] = d115
			ps160.OverlayValues[116] = d116
			ps160.OverlayValues[117] = d117
			ps160.OverlayValues[118] = d118
			ps160.OverlayValues[119] = d119
			ps160.OverlayValues[120] = d120
			ps160.OverlayValues[121] = d121
			ps160.OverlayValues[122] = d122
			ps160.OverlayValues[123] = d123
			ps160.OverlayValues[124] = d124
			ps160.OverlayValues[125] = d125
			ps160.OverlayValues[126] = d126
			ps160.OverlayValues[127] = d127
			ps160.OverlayValues[128] = d128
			ps160.OverlayValues[129] = d129
			ps160.OverlayValues[130] = d130
			ps160.OverlayValues[131] = d131
			ps160.OverlayValues[132] = d132
			ps160.OverlayValues[133] = d133
			ps160.OverlayValues[134] = d134
			ps160.OverlayValues[135] = d135
			ps160.OverlayValues[136] = d136
			ps160.OverlayValues[137] = d137
			ps160.OverlayValues[138] = d138
			ps160.OverlayValues[139] = d139
			ps160.OverlayValues[140] = d140
			ps160.OverlayValues[141] = d141
			ps160.OverlayValues[142] = d142
			ps160.OverlayValues[143] = d143
			ps160.OverlayValues[144] = d144
			ps160.OverlayValues[145] = d145
			ps160.OverlayValues[146] = d146
			ps160.OverlayValues[147] = d147
			ps160.OverlayValues[148] = d148
			ps160.OverlayValues[149] = d149
			ps160.OverlayValues[150] = d150
			ps160.OverlayValues[151] = d151
			ps160.OverlayValues[152] = d152
			ps160.OverlayValues[153] = d153
			ps160.OverlayValues[154] = d154
			ps160.OverlayValues[155] = d155
			ps160.OverlayValues[156] = d156
			ps160.OverlayValues[157] = d157
			ps160.OverlayValues[158] = d158
			ps160.OverlayValues[159] = d159
					return bbs[3].RenderPS(ps160)
				}
			ps161 := scm.PhiState{General: ps.General}
			ps161.OverlayValues = make([]scm.JITValueDesc, 160)
			ps161.OverlayValues[0] = d0
			ps161.OverlayValues[1] = d1
			ps161.OverlayValues[7] = d7
			ps161.OverlayValues[8] = d8
			ps161.OverlayValues[9] = d9
			ps161.OverlayValues[10] = d10
			ps161.OverlayValues[11] = d11
			ps161.OverlayValues[12] = d12
			ps161.OverlayValues[13] = d13
			ps161.OverlayValues[14] = d14
			ps161.OverlayValues[15] = d15
			ps161.OverlayValues[16] = d16
			ps161.OverlayValues[17] = d17
			ps161.OverlayValues[18] = d18
			ps161.OverlayValues[19] = d19
			ps161.OverlayValues[20] = d20
			ps161.OverlayValues[21] = d21
			ps161.OverlayValues[22] = d22
			ps161.OverlayValues[23] = d23
			ps161.OverlayValues[24] = d24
			ps161.OverlayValues[25] = d25
			ps161.OverlayValues[26] = d26
			ps161.OverlayValues[27] = d27
			ps161.OverlayValues[28] = d28
			ps161.OverlayValues[29] = d29
			ps161.OverlayValues[30] = d30
			ps161.OverlayValues[31] = d31
			ps161.OverlayValues[32] = d32
			ps161.OverlayValues[33] = d33
			ps161.OverlayValues[34] = d34
			ps161.OverlayValues[35] = d35
			ps161.OverlayValues[36] = d36
			ps161.OverlayValues[37] = d37
			ps161.OverlayValues[38] = d38
			ps161.OverlayValues[39] = d39
			ps161.OverlayValues[40] = d40
			ps161.OverlayValues[41] = d41
			ps161.OverlayValues[42] = d42
			ps161.OverlayValues[43] = d43
			ps161.OverlayValues[44] = d44
			ps161.OverlayValues[45] = d45
			ps161.OverlayValues[46] = d46
			ps161.OverlayValues[47] = d47
			ps161.OverlayValues[48] = d48
			ps161.OverlayValues[49] = d49
			ps161.OverlayValues[50] = d50
			ps161.OverlayValues[57] = d57
			ps161.OverlayValues[58] = d58
			ps161.OverlayValues[59] = d59
			ps161.OverlayValues[60] = d60
			ps161.OverlayValues[61] = d61
			ps161.OverlayValues[62] = d62
			ps161.OverlayValues[63] = d63
			ps161.OverlayValues[64] = d64
			ps161.OverlayValues[65] = d65
			ps161.OverlayValues[66] = d66
			ps161.OverlayValues[67] = d67
			ps161.OverlayValues[68] = d68
			ps161.OverlayValues[69] = d69
			ps161.OverlayValues[70] = d70
			ps161.OverlayValues[71] = d71
			ps161.OverlayValues[72] = d72
			ps161.OverlayValues[73] = d73
			ps161.OverlayValues[74] = d74
			ps161.OverlayValues[75] = d75
			ps161.OverlayValues[76] = d76
			ps161.OverlayValues[77] = d77
			ps161.OverlayValues[78] = d78
			ps161.OverlayValues[79] = d79
			ps161.OverlayValues[80] = d80
			ps161.OverlayValues[81] = d81
			ps161.OverlayValues[82] = d82
			ps161.OverlayValues[83] = d83
			ps161.OverlayValues[84] = d84
			ps161.OverlayValues[85] = d85
			ps161.OverlayValues[86] = d86
			ps161.OverlayValues[87] = d87
			ps161.OverlayValues[88] = d88
			ps161.OverlayValues[89] = d89
			ps161.OverlayValues[90] = d90
			ps161.OverlayValues[91] = d91
			ps161.OverlayValues[92] = d92
			ps161.OverlayValues[93] = d93
			ps161.OverlayValues[94] = d94
			ps161.OverlayValues[95] = d95
			ps161.OverlayValues[96] = d96
			ps161.OverlayValues[97] = d97
			ps161.OverlayValues[98] = d98
			ps161.OverlayValues[99] = d99
			ps161.OverlayValues[100] = d100
			ps161.OverlayValues[107] = d107
			ps161.OverlayValues[108] = d108
			ps161.OverlayValues[109] = d109
			ps161.OverlayValues[110] = d110
			ps161.OverlayValues[111] = d111
			ps161.OverlayValues[112] = d112
			ps161.OverlayValues[113] = d113
			ps161.OverlayValues[114] = d114
			ps161.OverlayValues[115] = d115
			ps161.OverlayValues[116] = d116
			ps161.OverlayValues[117] = d117
			ps161.OverlayValues[118] = d118
			ps161.OverlayValues[119] = d119
			ps161.OverlayValues[120] = d120
			ps161.OverlayValues[121] = d121
			ps161.OverlayValues[122] = d122
			ps161.OverlayValues[123] = d123
			ps161.OverlayValues[124] = d124
			ps161.OverlayValues[125] = d125
			ps161.OverlayValues[126] = d126
			ps161.OverlayValues[127] = d127
			ps161.OverlayValues[128] = d128
			ps161.OverlayValues[129] = d129
			ps161.OverlayValues[130] = d130
			ps161.OverlayValues[131] = d131
			ps161.OverlayValues[132] = d132
			ps161.OverlayValues[133] = d133
			ps161.OverlayValues[134] = d134
			ps161.OverlayValues[135] = d135
			ps161.OverlayValues[136] = d136
			ps161.OverlayValues[137] = d137
			ps161.OverlayValues[138] = d138
			ps161.OverlayValues[139] = d139
			ps161.OverlayValues[140] = d140
			ps161.OverlayValues[141] = d141
			ps161.OverlayValues[142] = d142
			ps161.OverlayValues[143] = d143
			ps161.OverlayValues[144] = d144
			ps161.OverlayValues[145] = d145
			ps161.OverlayValues[146] = d146
			ps161.OverlayValues[147] = d147
			ps161.OverlayValues[148] = d148
			ps161.OverlayValues[149] = d149
			ps161.OverlayValues[150] = d150
			ps161.OverlayValues[151] = d151
			ps161.OverlayValues[152] = d152
			ps161.OverlayValues[153] = d153
			ps161.OverlayValues[154] = d154
			ps161.OverlayValues[155] = d155
			ps161.OverlayValues[156] = d156
			ps161.OverlayValues[157] = d157
			ps161.OverlayValues[158] = d158
			ps161.OverlayValues[159] = d159
				return bbs[4].RenderPS(ps161)
			}
			lbl40 := ctx.W.ReserveLabel()
			lbl41 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d159.Reg, 0)
			ctx.W.EmitJcc(scm.CcNE, lbl40)
			ctx.W.EmitJmp(lbl41)
			ctx.W.MarkLabel(lbl40)
			ctx.W.EmitJmp(lbl4)
			ctx.W.MarkLabel(lbl41)
			ctx.W.EmitJmp(lbl5)
			ps162 := scm.PhiState{General: true}
			ps162.OverlayValues = make([]scm.JITValueDesc, 160)
			ps162.OverlayValues[0] = d0
			ps162.OverlayValues[1] = d1
			ps162.OverlayValues[7] = d7
			ps162.OverlayValues[8] = d8
			ps162.OverlayValues[9] = d9
			ps162.OverlayValues[10] = d10
			ps162.OverlayValues[11] = d11
			ps162.OverlayValues[12] = d12
			ps162.OverlayValues[13] = d13
			ps162.OverlayValues[14] = d14
			ps162.OverlayValues[15] = d15
			ps162.OverlayValues[16] = d16
			ps162.OverlayValues[17] = d17
			ps162.OverlayValues[18] = d18
			ps162.OverlayValues[19] = d19
			ps162.OverlayValues[20] = d20
			ps162.OverlayValues[21] = d21
			ps162.OverlayValues[22] = d22
			ps162.OverlayValues[23] = d23
			ps162.OverlayValues[24] = d24
			ps162.OverlayValues[25] = d25
			ps162.OverlayValues[26] = d26
			ps162.OverlayValues[27] = d27
			ps162.OverlayValues[28] = d28
			ps162.OverlayValues[29] = d29
			ps162.OverlayValues[30] = d30
			ps162.OverlayValues[31] = d31
			ps162.OverlayValues[32] = d32
			ps162.OverlayValues[33] = d33
			ps162.OverlayValues[34] = d34
			ps162.OverlayValues[35] = d35
			ps162.OverlayValues[36] = d36
			ps162.OverlayValues[37] = d37
			ps162.OverlayValues[38] = d38
			ps162.OverlayValues[39] = d39
			ps162.OverlayValues[40] = d40
			ps162.OverlayValues[41] = d41
			ps162.OverlayValues[42] = d42
			ps162.OverlayValues[43] = d43
			ps162.OverlayValues[44] = d44
			ps162.OverlayValues[45] = d45
			ps162.OverlayValues[46] = d46
			ps162.OverlayValues[47] = d47
			ps162.OverlayValues[48] = d48
			ps162.OverlayValues[49] = d49
			ps162.OverlayValues[50] = d50
			ps162.OverlayValues[57] = d57
			ps162.OverlayValues[58] = d58
			ps162.OverlayValues[59] = d59
			ps162.OverlayValues[60] = d60
			ps162.OverlayValues[61] = d61
			ps162.OverlayValues[62] = d62
			ps162.OverlayValues[63] = d63
			ps162.OverlayValues[64] = d64
			ps162.OverlayValues[65] = d65
			ps162.OverlayValues[66] = d66
			ps162.OverlayValues[67] = d67
			ps162.OverlayValues[68] = d68
			ps162.OverlayValues[69] = d69
			ps162.OverlayValues[70] = d70
			ps162.OverlayValues[71] = d71
			ps162.OverlayValues[72] = d72
			ps162.OverlayValues[73] = d73
			ps162.OverlayValues[74] = d74
			ps162.OverlayValues[75] = d75
			ps162.OverlayValues[76] = d76
			ps162.OverlayValues[77] = d77
			ps162.OverlayValues[78] = d78
			ps162.OverlayValues[79] = d79
			ps162.OverlayValues[80] = d80
			ps162.OverlayValues[81] = d81
			ps162.OverlayValues[82] = d82
			ps162.OverlayValues[83] = d83
			ps162.OverlayValues[84] = d84
			ps162.OverlayValues[85] = d85
			ps162.OverlayValues[86] = d86
			ps162.OverlayValues[87] = d87
			ps162.OverlayValues[88] = d88
			ps162.OverlayValues[89] = d89
			ps162.OverlayValues[90] = d90
			ps162.OverlayValues[91] = d91
			ps162.OverlayValues[92] = d92
			ps162.OverlayValues[93] = d93
			ps162.OverlayValues[94] = d94
			ps162.OverlayValues[95] = d95
			ps162.OverlayValues[96] = d96
			ps162.OverlayValues[97] = d97
			ps162.OverlayValues[98] = d98
			ps162.OverlayValues[99] = d99
			ps162.OverlayValues[100] = d100
			ps162.OverlayValues[107] = d107
			ps162.OverlayValues[108] = d108
			ps162.OverlayValues[109] = d109
			ps162.OverlayValues[110] = d110
			ps162.OverlayValues[111] = d111
			ps162.OverlayValues[112] = d112
			ps162.OverlayValues[113] = d113
			ps162.OverlayValues[114] = d114
			ps162.OverlayValues[115] = d115
			ps162.OverlayValues[116] = d116
			ps162.OverlayValues[117] = d117
			ps162.OverlayValues[118] = d118
			ps162.OverlayValues[119] = d119
			ps162.OverlayValues[120] = d120
			ps162.OverlayValues[121] = d121
			ps162.OverlayValues[122] = d122
			ps162.OverlayValues[123] = d123
			ps162.OverlayValues[124] = d124
			ps162.OverlayValues[125] = d125
			ps162.OverlayValues[126] = d126
			ps162.OverlayValues[127] = d127
			ps162.OverlayValues[128] = d128
			ps162.OverlayValues[129] = d129
			ps162.OverlayValues[130] = d130
			ps162.OverlayValues[131] = d131
			ps162.OverlayValues[132] = d132
			ps162.OverlayValues[133] = d133
			ps162.OverlayValues[134] = d134
			ps162.OverlayValues[135] = d135
			ps162.OverlayValues[136] = d136
			ps162.OverlayValues[137] = d137
			ps162.OverlayValues[138] = d138
			ps162.OverlayValues[139] = d139
			ps162.OverlayValues[140] = d140
			ps162.OverlayValues[141] = d141
			ps162.OverlayValues[142] = d142
			ps162.OverlayValues[143] = d143
			ps162.OverlayValues[144] = d144
			ps162.OverlayValues[145] = d145
			ps162.OverlayValues[146] = d146
			ps162.OverlayValues[147] = d147
			ps162.OverlayValues[148] = d148
			ps162.OverlayValues[149] = d149
			ps162.OverlayValues[150] = d150
			ps162.OverlayValues[151] = d151
			ps162.OverlayValues[152] = d152
			ps162.OverlayValues[153] = d153
			ps162.OverlayValues[154] = d154
			ps162.OverlayValues[155] = d155
			ps162.OverlayValues[156] = d156
			ps162.OverlayValues[157] = d157
			ps162.OverlayValues[158] = d158
			ps162.OverlayValues[159] = d159
			ps163 := scm.PhiState{General: true}
			ps163.OverlayValues = make([]scm.JITValueDesc, 160)
			ps163.OverlayValues[0] = d0
			ps163.OverlayValues[1] = d1
			ps163.OverlayValues[7] = d7
			ps163.OverlayValues[8] = d8
			ps163.OverlayValues[9] = d9
			ps163.OverlayValues[10] = d10
			ps163.OverlayValues[11] = d11
			ps163.OverlayValues[12] = d12
			ps163.OverlayValues[13] = d13
			ps163.OverlayValues[14] = d14
			ps163.OverlayValues[15] = d15
			ps163.OverlayValues[16] = d16
			ps163.OverlayValues[17] = d17
			ps163.OverlayValues[18] = d18
			ps163.OverlayValues[19] = d19
			ps163.OverlayValues[20] = d20
			ps163.OverlayValues[21] = d21
			ps163.OverlayValues[22] = d22
			ps163.OverlayValues[23] = d23
			ps163.OverlayValues[24] = d24
			ps163.OverlayValues[25] = d25
			ps163.OverlayValues[26] = d26
			ps163.OverlayValues[27] = d27
			ps163.OverlayValues[28] = d28
			ps163.OverlayValues[29] = d29
			ps163.OverlayValues[30] = d30
			ps163.OverlayValues[31] = d31
			ps163.OverlayValues[32] = d32
			ps163.OverlayValues[33] = d33
			ps163.OverlayValues[34] = d34
			ps163.OverlayValues[35] = d35
			ps163.OverlayValues[36] = d36
			ps163.OverlayValues[37] = d37
			ps163.OverlayValues[38] = d38
			ps163.OverlayValues[39] = d39
			ps163.OverlayValues[40] = d40
			ps163.OverlayValues[41] = d41
			ps163.OverlayValues[42] = d42
			ps163.OverlayValues[43] = d43
			ps163.OverlayValues[44] = d44
			ps163.OverlayValues[45] = d45
			ps163.OverlayValues[46] = d46
			ps163.OverlayValues[47] = d47
			ps163.OverlayValues[48] = d48
			ps163.OverlayValues[49] = d49
			ps163.OverlayValues[50] = d50
			ps163.OverlayValues[57] = d57
			ps163.OverlayValues[58] = d58
			ps163.OverlayValues[59] = d59
			ps163.OverlayValues[60] = d60
			ps163.OverlayValues[61] = d61
			ps163.OverlayValues[62] = d62
			ps163.OverlayValues[63] = d63
			ps163.OverlayValues[64] = d64
			ps163.OverlayValues[65] = d65
			ps163.OverlayValues[66] = d66
			ps163.OverlayValues[67] = d67
			ps163.OverlayValues[68] = d68
			ps163.OverlayValues[69] = d69
			ps163.OverlayValues[70] = d70
			ps163.OverlayValues[71] = d71
			ps163.OverlayValues[72] = d72
			ps163.OverlayValues[73] = d73
			ps163.OverlayValues[74] = d74
			ps163.OverlayValues[75] = d75
			ps163.OverlayValues[76] = d76
			ps163.OverlayValues[77] = d77
			ps163.OverlayValues[78] = d78
			ps163.OverlayValues[79] = d79
			ps163.OverlayValues[80] = d80
			ps163.OverlayValues[81] = d81
			ps163.OverlayValues[82] = d82
			ps163.OverlayValues[83] = d83
			ps163.OverlayValues[84] = d84
			ps163.OverlayValues[85] = d85
			ps163.OverlayValues[86] = d86
			ps163.OverlayValues[87] = d87
			ps163.OverlayValues[88] = d88
			ps163.OverlayValues[89] = d89
			ps163.OverlayValues[90] = d90
			ps163.OverlayValues[91] = d91
			ps163.OverlayValues[92] = d92
			ps163.OverlayValues[93] = d93
			ps163.OverlayValues[94] = d94
			ps163.OverlayValues[95] = d95
			ps163.OverlayValues[96] = d96
			ps163.OverlayValues[97] = d97
			ps163.OverlayValues[98] = d98
			ps163.OverlayValues[99] = d99
			ps163.OverlayValues[100] = d100
			ps163.OverlayValues[107] = d107
			ps163.OverlayValues[108] = d108
			ps163.OverlayValues[109] = d109
			ps163.OverlayValues[110] = d110
			ps163.OverlayValues[111] = d111
			ps163.OverlayValues[112] = d112
			ps163.OverlayValues[113] = d113
			ps163.OverlayValues[114] = d114
			ps163.OverlayValues[115] = d115
			ps163.OverlayValues[116] = d116
			ps163.OverlayValues[117] = d117
			ps163.OverlayValues[118] = d118
			ps163.OverlayValues[119] = d119
			ps163.OverlayValues[120] = d120
			ps163.OverlayValues[121] = d121
			ps163.OverlayValues[122] = d122
			ps163.OverlayValues[123] = d123
			ps163.OverlayValues[124] = d124
			ps163.OverlayValues[125] = d125
			ps163.OverlayValues[126] = d126
			ps163.OverlayValues[127] = d127
			ps163.OverlayValues[128] = d128
			ps163.OverlayValues[129] = d129
			ps163.OverlayValues[130] = d130
			ps163.OverlayValues[131] = d131
			ps163.OverlayValues[132] = d132
			ps163.OverlayValues[133] = d133
			ps163.OverlayValues[134] = d134
			ps163.OverlayValues[135] = d135
			ps163.OverlayValues[136] = d136
			ps163.OverlayValues[137] = d137
			ps163.OverlayValues[138] = d138
			ps163.OverlayValues[139] = d139
			ps163.OverlayValues[140] = d140
			ps163.OverlayValues[141] = d141
			ps163.OverlayValues[142] = d142
			ps163.OverlayValues[143] = d143
			ps163.OverlayValues[144] = d144
			ps163.OverlayValues[145] = d145
			ps163.OverlayValues[146] = d146
			ps163.OverlayValues[147] = d147
			ps163.OverlayValues[148] = d148
			ps163.OverlayValues[149] = d149
			ps163.OverlayValues[150] = d150
			ps163.OverlayValues[151] = d151
			ps163.OverlayValues[152] = d152
			ps163.OverlayValues[153] = d153
			ps163.OverlayValues[154] = d154
			ps163.OverlayValues[155] = d155
			ps163.OverlayValues[156] = d156
			ps163.OverlayValues[157] = d157
			ps163.OverlayValues[158] = d158
			ps163.OverlayValues[159] = d159
			alloc164 := ctx.SnapshotAllocState()
			if !bbs[4].Rendered {
				bbs[4].RenderPS(ps163)
			}
			ctx.RestoreAllocState(alloc164)
			if !bbs[3].Rendered {
				return bbs[3].RenderPS(ps162)
			}
			return result
			ctx.FreeDesc(&d158)
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
					ctx.W.EmitJmp(lbl7)
					return result
				}
				bbs[6].Rendered = true
				bbs[6].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_6 = bbs[6].Address
				ctx.W.MarkLabel(lbl7)
				ctx.W.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
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
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
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
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != scm.LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != scm.LocNone {
				d70 = ps.OverlayValues[70]
			}
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != scm.LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != scm.LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != scm.LocNone {
				d73 = ps.OverlayValues[73]
			}
			if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != scm.LocNone {
				d74 = ps.OverlayValues[74]
			}
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != scm.LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != scm.LocNone {
				d76 = ps.OverlayValues[76]
			}
			if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
				d77 = ps.OverlayValues[77]
			}
			if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
				d78 = ps.OverlayValues[78]
			}
			if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
				d79 = ps.OverlayValues[79]
			}
			if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
				d80 = ps.OverlayValues[80]
			}
			if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
				d81 = ps.OverlayValues[81]
			}
			if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
				d82 = ps.OverlayValues[82]
			}
			if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
				d83 = ps.OverlayValues[83]
			}
			if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != scm.LocNone {
				d84 = ps.OverlayValues[84]
			}
			if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
				d85 = ps.OverlayValues[85]
			}
			if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
				d86 = ps.OverlayValues[86]
			}
			if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != scm.LocNone {
				d87 = ps.OverlayValues[87]
			}
			if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != scm.LocNone {
				d88 = ps.OverlayValues[88]
			}
			if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != scm.LocNone {
				d89 = ps.OverlayValues[89]
			}
			if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != scm.LocNone {
				d90 = ps.OverlayValues[90]
			}
			if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != scm.LocNone {
				d91 = ps.OverlayValues[91]
			}
			if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != scm.LocNone {
				d92 = ps.OverlayValues[92]
			}
			if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != scm.LocNone {
				d93 = ps.OverlayValues[93]
			}
			if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != scm.LocNone {
				d94 = ps.OverlayValues[94]
			}
			if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != scm.LocNone {
				d95 = ps.OverlayValues[95]
			}
			if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != scm.LocNone {
				d96 = ps.OverlayValues[96]
			}
			if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != scm.LocNone {
				d97 = ps.OverlayValues[97]
			}
			if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != scm.LocNone {
				d98 = ps.OverlayValues[98]
			}
			if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != scm.LocNone {
				d99 = ps.OverlayValues[99]
			}
			if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != scm.LocNone {
				d100 = ps.OverlayValues[100]
			}
			if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != scm.LocNone {
				d107 = ps.OverlayValues[107]
			}
			if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != scm.LocNone {
				d108 = ps.OverlayValues[108]
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
			ctx.ReclaimUntrackedRegs()
			d165 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d165)
			ctx.BindReg(r1, &d165)
			ctx.W.EmitMakeNil(d165)
			ctx.W.EmitJmp(lbl0)
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
					ctx.W.EmitJmp(lbl8)
					return result
				}
				bbs[7].Rendered = true
				bbs[7].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_7 = bbs[7].Address
				ctx.W.MarkLabel(lbl8)
				ctx.W.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
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
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
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
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != scm.LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != scm.LocNone {
				d70 = ps.OverlayValues[70]
			}
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != scm.LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != scm.LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != scm.LocNone {
				d73 = ps.OverlayValues[73]
			}
			if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != scm.LocNone {
				d74 = ps.OverlayValues[74]
			}
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != scm.LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != scm.LocNone {
				d76 = ps.OverlayValues[76]
			}
			if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
				d77 = ps.OverlayValues[77]
			}
			if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
				d78 = ps.OverlayValues[78]
			}
			if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
				d79 = ps.OverlayValues[79]
			}
			if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
				d80 = ps.OverlayValues[80]
			}
			if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
				d81 = ps.OverlayValues[81]
			}
			if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
				d82 = ps.OverlayValues[82]
			}
			if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
				d83 = ps.OverlayValues[83]
			}
			if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != scm.LocNone {
				d84 = ps.OverlayValues[84]
			}
			if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
				d85 = ps.OverlayValues[85]
			}
			if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
				d86 = ps.OverlayValues[86]
			}
			if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != scm.LocNone {
				d87 = ps.OverlayValues[87]
			}
			if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != scm.LocNone {
				d88 = ps.OverlayValues[88]
			}
			if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != scm.LocNone {
				d89 = ps.OverlayValues[89]
			}
			if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != scm.LocNone {
				d90 = ps.OverlayValues[90]
			}
			if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != scm.LocNone {
				d91 = ps.OverlayValues[91]
			}
			if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != scm.LocNone {
				d92 = ps.OverlayValues[92]
			}
			if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != scm.LocNone {
				d93 = ps.OverlayValues[93]
			}
			if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != scm.LocNone {
				d94 = ps.OverlayValues[94]
			}
			if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != scm.LocNone {
				d95 = ps.OverlayValues[95]
			}
			if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != scm.LocNone {
				d96 = ps.OverlayValues[96]
			}
			if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != scm.LocNone {
				d97 = ps.OverlayValues[97]
			}
			if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != scm.LocNone {
				d98 = ps.OverlayValues[98]
			}
			if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != scm.LocNone {
				d99 = ps.OverlayValues[99]
			}
			if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != scm.LocNone {
				d100 = ps.OverlayValues[100]
			}
			if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != scm.LocNone {
				d107 = ps.OverlayValues[107]
			}
			if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != scm.LocNone {
				d108 = ps.OverlayValues[108]
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
			if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != scm.LocNone {
				d165 = ps.OverlayValues[165]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d98)
			d166 = d98
			_ = d166
			r149 := d98.Loc == scm.LocReg
			r150 := d98.Reg
			if r149 { ctx.ProtectReg(r150) }
			d167 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			lbl42 := ctx.W.ReserveLabel()
			bbpos_4_0 := int32(-1)
			_ = bbpos_4_0
			bbpos_4_1 := int32(-1)
			_ = bbpos_4_1
			bbpos_4_2 := int32(-1)
			_ = bbpos_4_2
			bbpos_4_3 := int32(-1)
			_ = bbpos_4_3
			bbpos_4_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			d167 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d166)
			ctx.EnsureDesc(&d166)
			var d168 scm.JITValueDesc
			if d166.Loc == scm.LocImm {
				d168 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d166.Imm.Int()))))}
			} else {
				r151 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r151, d166.Reg)
				ctx.W.EmitShlRegImm8(r151, 32)
				ctx.W.EmitShrRegImm8(r151, 32)
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
				ctx.BindReg(r151, &d168)
			}
			var d169 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d169 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r152 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r152, thisptr.Reg, off)
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r152}
				ctx.BindReg(r152, &d169)
			}
			ctx.EnsureDesc(&d169)
			ctx.EnsureDesc(&d169)
			var d170 scm.JITValueDesc
			if d169.Loc == scm.LocImm {
				d170 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d169.Imm.Int()))))}
			} else {
				r153 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r153, d169.Reg)
				ctx.W.EmitShlRegImm8(r153, 56)
				ctx.W.EmitShrRegImm8(r153, 56)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r153}
				ctx.BindReg(r153, &d170)
			}
			ctx.FreeDesc(&d169)
			ctx.EnsureDesc(&d168)
			ctx.EnsureDesc(&d170)
			ctx.EnsureDesc(&d168)
			ctx.EnsureDesc(&d170)
			ctx.EnsureDesc(&d168)
			ctx.EnsureDesc(&d170)
			var d171 scm.JITValueDesc
			if d168.Loc == scm.LocImm && d170.Loc == scm.LocImm {
				d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d168.Imm.Int() * d170.Imm.Int())}
			} else if d168.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d170.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d168.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d170.Reg)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d171)
			} else if d170.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d168.Reg)
				ctx.W.EmitMovRegReg(scratch, d168.Reg)
				if d170.Imm.Int() >= -2147483648 && d170.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d170.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d170.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d171)
			} else {
				r154 := ctx.AllocRegExcept(d168.Reg, d170.Reg)
				ctx.W.EmitMovRegReg(r154, d168.Reg)
				ctx.W.EmitImulInt64(r154, d170.Reg)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r154}
				ctx.BindReg(r154, &d171)
			}
			if d171.Loc == scm.LocReg && d168.Loc == scm.LocReg && d171.Reg == d168.Reg {
				ctx.TransferReg(d168.Reg)
				d168.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d168)
			ctx.FreeDesc(&d170)
			var d172 scm.JITValueDesc
			r155 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r155, uint64(dataPtr))
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r155, StackOff: int32(sliceLen)}
				ctx.BindReg(r155, &d172)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 0)
				ctx.W.EmitMovRegMem(r155, thisptr.Reg, off)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r155}
				ctx.BindReg(r155, &d172)
			}
			ctx.BindReg(r155, &d172)
			ctx.EnsureDesc(&d171)
			var d173 scm.JITValueDesc
			if d171.Loc == scm.LocImm {
				d173 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d171.Imm.Int() / 64)}
			} else {
				r156 := ctx.AllocRegExcept(d171.Reg)
				ctx.W.EmitMovRegReg(r156, d171.Reg)
				ctx.W.EmitShrRegImm8(r156, 6)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r156}
				ctx.BindReg(r156, &d173)
			}
			if d173.Loc == scm.LocReg && d171.Loc == scm.LocReg && d173.Reg == d171.Reg {
				ctx.TransferReg(d171.Reg)
				d171.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d173)
			r157 := ctx.AllocReg()
			ctx.EnsureDesc(&d173)
			ctx.EnsureDesc(&d172)
			if d173.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r157, uint64(d173.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r157, d173.Reg)
				ctx.W.EmitShlRegImm8(r157, 3)
			}
			if d172.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d172.Imm.Int()))
				ctx.W.EmitAddInt64(r157, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r157, d172.Reg)
			}
			r158 := ctx.AllocRegExcept(r157)
			ctx.W.EmitMovRegMem(r158, r157, 0)
			ctx.FreeReg(r157)
			d174 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r158}
			ctx.BindReg(r158, &d174)
			ctx.FreeDesc(&d173)
			ctx.EnsureDesc(&d171)
			var d175 scm.JITValueDesc
			if d171.Loc == scm.LocImm {
				d175 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d171.Imm.Int() % 64)}
			} else {
				r159 := ctx.AllocRegExcept(d171.Reg)
				ctx.W.EmitMovRegReg(r159, d171.Reg)
				ctx.W.EmitAndRegImm32(r159, 63)
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
				ctx.BindReg(r159, &d175)
			}
			if d175.Loc == scm.LocReg && d171.Loc == scm.LocReg && d175.Reg == d171.Reg {
				ctx.TransferReg(d171.Reg)
				d171.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d174)
			ctx.EnsureDesc(&d175)
			var d176 scm.JITValueDesc
			if d174.Loc == scm.LocImm && d175.Loc == scm.LocImm {
				d176 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d174.Imm.Int()) << uint64(d175.Imm.Int())))}
			} else if d175.Loc == scm.LocImm {
				r160 := ctx.AllocRegExcept(d174.Reg)
				ctx.W.EmitMovRegReg(r160, d174.Reg)
				ctx.W.EmitShlRegImm8(r160, uint8(d175.Imm.Int()))
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r160}
				ctx.BindReg(r160, &d176)
			} else {
				{
					shiftSrc := d174.Reg
					r161 := ctx.AllocRegExcept(d174.Reg)
					ctx.W.EmitMovRegReg(r161, d174.Reg)
					shiftSrc = r161
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d175.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d175.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d175.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
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
			ctx.FreeDesc(&d175)
			var d177 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d177 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 25)
				r162 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r162, thisptr.Reg, off)
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r162}
				ctx.BindReg(r162, &d177)
			}
			d178 = d177
			ctx.EnsureDesc(&d178)
			if d178.Loc != scm.LocImm && d178.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl43 := ctx.W.ReserveLabel()
			lbl44 := ctx.W.ReserveLabel()
			lbl45 := ctx.W.ReserveLabel()
			lbl46 := ctx.W.ReserveLabel()
			if d178.Loc == scm.LocImm {
				if d178.Imm.Bool() {
					ctx.W.MarkLabel(lbl45)
					ctx.W.EmitJmp(lbl43)
				} else {
					ctx.W.MarkLabel(lbl46)
			d179 = d176
			if d179.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d179)
			ctx.EmitStoreToStack(d179, 24)
					ctx.W.EmitJmp(lbl44)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d178.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl45)
				ctx.W.EmitJmp(lbl46)
				ctx.W.MarkLabel(lbl45)
				ctx.W.EmitJmp(lbl43)
				ctx.W.MarkLabel(lbl46)
			d180 = d176
			if d180.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d180)
			ctx.EmitStoreToStack(d180, 24)
				ctx.W.EmitJmp(lbl44)
			}
			ctx.FreeDesc(&d177)
			bbpos_4_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl44)
			ctx.W.ResolveFixups()
			d167 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			var d181 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d181 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r163 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r163, thisptr.Reg, off)
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r163}
				ctx.BindReg(r163, &d181)
			}
			ctx.EnsureDesc(&d181)
			ctx.EnsureDesc(&d181)
			var d182 scm.JITValueDesc
			if d181.Loc == scm.LocImm {
				d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d181.Imm.Int()))))}
			} else {
				r164 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r164, d181.Reg)
				ctx.W.EmitShlRegImm8(r164, 56)
				ctx.W.EmitShrRegImm8(r164, 56)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r164}
				ctx.BindReg(r164, &d182)
			}
			ctx.FreeDesc(&d181)
			d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d182)
			ctx.EnsureDesc(&d183)
			ctx.EnsureDesc(&d182)
			ctx.EnsureDesc(&d183)
			ctx.EnsureDesc(&d182)
			var d184 scm.JITValueDesc
			if d183.Loc == scm.LocImm && d182.Loc == scm.LocImm {
				d184 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d183.Imm.Int() - d182.Imm.Int())}
			} else if d182.Loc == scm.LocImm && d182.Imm.Int() == 0 {
				r165 := ctx.AllocRegExcept(d183.Reg)
				ctx.W.EmitMovRegReg(r165, d183.Reg)
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r165}
				ctx.BindReg(r165, &d184)
			} else if d183.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d182.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d183.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d182.Reg)
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d184)
			} else if d182.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d183.Reg)
				ctx.W.EmitMovRegReg(scratch, d183.Reg)
				if d182.Imm.Int() >= -2147483648 && d182.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d182.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d182.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d184)
			} else {
				r166 := ctx.AllocRegExcept(d183.Reg, d182.Reg)
				ctx.W.EmitMovRegReg(r166, d183.Reg)
				ctx.W.EmitSubInt64(r166, d182.Reg)
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r166}
				ctx.BindReg(r166, &d184)
			}
			if d184.Loc == scm.LocReg && d183.Loc == scm.LocReg && d184.Reg == d183.Reg {
				ctx.TransferReg(d183.Reg)
				d183.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d182)
			ctx.EnsureDesc(&d167)
			ctx.EnsureDesc(&d184)
			var d185 scm.JITValueDesc
			if d167.Loc == scm.LocImm && d184.Loc == scm.LocImm {
				d185 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d167.Imm.Int()) >> uint64(d184.Imm.Int())))}
			} else if d184.Loc == scm.LocImm {
				r167 := ctx.AllocRegExcept(d167.Reg)
				ctx.W.EmitMovRegReg(r167, d167.Reg)
				ctx.W.EmitShrRegImm8(r167, uint8(d184.Imm.Int()))
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r167}
				ctx.BindReg(r167, &d185)
			} else {
				{
					shiftSrc := d167.Reg
					r168 := ctx.AllocRegExcept(d167.Reg)
					ctx.W.EmitMovRegReg(r168, d167.Reg)
					shiftSrc = r168
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d184.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d184.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d184.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d185)
				}
			}
			if d185.Loc == scm.LocReg && d167.Loc == scm.LocReg && d185.Reg == d167.Reg {
				ctx.TransferReg(d167.Reg)
				d167.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d167)
			ctx.FreeDesc(&d184)
			r169 := ctx.AllocReg()
			ctx.EnsureDesc(&d185)
			ctx.EnsureDesc(&d185)
			if d185.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r169, d185)
			}
			ctx.W.EmitJmp(lbl42)
			bbpos_4_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl43)
			ctx.W.ResolveFixups()
			d167 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d171)
			var d186 scm.JITValueDesc
			if d171.Loc == scm.LocImm {
				d186 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d171.Imm.Int() % 64)}
			} else {
				r170 := ctx.AllocRegExcept(d171.Reg)
				ctx.W.EmitMovRegReg(r170, d171.Reg)
				ctx.W.EmitAndRegImm32(r170, 63)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r170}
				ctx.BindReg(r170, &d186)
			}
			if d186.Loc == scm.LocReg && d171.Loc == scm.LocReg && d186.Reg == d171.Reg {
				ctx.TransferReg(d171.Reg)
				d171.Loc = scm.LocNone
			}
			var d187 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d187 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r171 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r171, thisptr.Reg, off)
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r171}
				ctx.BindReg(r171, &d187)
			}
			ctx.EnsureDesc(&d187)
			ctx.EnsureDesc(&d187)
			var d188 scm.JITValueDesc
			if d187.Loc == scm.LocImm {
				d188 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d187.Imm.Int()))))}
			} else {
				r172 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r172, d187.Reg)
				ctx.W.EmitShlRegImm8(r172, 56)
				ctx.W.EmitShrRegImm8(r172, 56)
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
				ctx.BindReg(r172, &d188)
			}
			ctx.FreeDesc(&d187)
			ctx.EnsureDesc(&d186)
			ctx.EnsureDesc(&d188)
			ctx.EnsureDesc(&d186)
			ctx.EnsureDesc(&d188)
			ctx.EnsureDesc(&d186)
			ctx.EnsureDesc(&d188)
			var d189 scm.JITValueDesc
			if d186.Loc == scm.LocImm && d188.Loc == scm.LocImm {
				d189 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d186.Imm.Int() + d188.Imm.Int())}
			} else if d188.Loc == scm.LocImm && d188.Imm.Int() == 0 {
				r173 := ctx.AllocRegExcept(d186.Reg)
				ctx.W.EmitMovRegReg(r173, d186.Reg)
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r173}
				ctx.BindReg(r173, &d189)
			} else if d186.Loc == scm.LocImm && d186.Imm.Int() == 0 {
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d188.Reg}
				ctx.BindReg(d188.Reg, &d189)
			} else if d186.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d188.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d186.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d188.Reg)
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d189)
			} else if d188.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d186.Reg)
				ctx.W.EmitMovRegReg(scratch, d186.Reg)
				if d188.Imm.Int() >= -2147483648 && d188.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d188.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d188.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d189)
			} else {
				r174 := ctx.AllocRegExcept(d186.Reg, d188.Reg)
				ctx.W.EmitMovRegReg(r174, d186.Reg)
				ctx.W.EmitAddInt64(r174, d188.Reg)
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r174}
				ctx.BindReg(r174, &d189)
			}
			if d189.Loc == scm.LocReg && d186.Loc == scm.LocReg && d189.Reg == d186.Reg {
				ctx.TransferReg(d186.Reg)
				d186.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d186)
			ctx.FreeDesc(&d188)
			ctx.EnsureDesc(&d189)
			var d190 scm.JITValueDesc
			if d189.Loc == scm.LocImm {
				d190 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d189.Imm.Int()) > uint64(64))}
			} else {
				r175 := ctx.AllocRegExcept(d189.Reg)
				ctx.W.EmitCmpRegImm32(d189.Reg, 64)
				ctx.W.EmitSetcc(r175, scm.CcA)
				d190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r175}
				ctx.BindReg(r175, &d190)
			}
			ctx.FreeDesc(&d189)
			d191 = d190
			ctx.EnsureDesc(&d191)
			if d191.Loc != scm.LocImm && d191.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl47 := ctx.W.ReserveLabel()
			lbl48 := ctx.W.ReserveLabel()
			lbl49 := ctx.W.ReserveLabel()
			if d191.Loc == scm.LocImm {
				if d191.Imm.Bool() {
					ctx.W.MarkLabel(lbl48)
					ctx.W.EmitJmp(lbl47)
				} else {
					ctx.W.MarkLabel(lbl49)
			d192 = d176
			if d192.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d192)
			ctx.EmitStoreToStack(d192, 24)
					ctx.W.EmitJmp(lbl44)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d191.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl48)
				ctx.W.EmitJmp(lbl49)
				ctx.W.MarkLabel(lbl48)
				ctx.W.EmitJmp(lbl47)
				ctx.W.MarkLabel(lbl49)
			d193 = d176
			if d193.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d193)
			ctx.EmitStoreToStack(d193, 24)
				ctx.W.EmitJmp(lbl44)
			}
			ctx.FreeDesc(&d190)
			bbpos_4_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl47)
			ctx.W.ResolveFixups()
			d167 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d171)
			var d194 scm.JITValueDesc
			if d171.Loc == scm.LocImm {
				d194 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d171.Imm.Int() / 64)}
			} else {
				r176 := ctx.AllocRegExcept(d171.Reg)
				ctx.W.EmitMovRegReg(r176, d171.Reg)
				ctx.W.EmitShrRegImm8(r176, 6)
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r176}
				ctx.BindReg(r176, &d194)
			}
			if d194.Loc == scm.LocReg && d171.Loc == scm.LocReg && d194.Reg == d171.Reg {
				ctx.TransferReg(d171.Reg)
				d171.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d194)
			ctx.EnsureDesc(&d194)
			var d195 scm.JITValueDesc
			if d194.Loc == scm.LocImm {
				d195 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d194.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d194.Reg)
				ctx.W.EmitMovRegReg(scratch, d194.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d195 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d195)
			}
			if d195.Loc == scm.LocReg && d194.Loc == scm.LocReg && d195.Reg == d194.Reg {
				ctx.TransferReg(d194.Reg)
				d194.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d194)
			ctx.EnsureDesc(&d195)
			r177 := ctx.AllocReg()
			ctx.EnsureDesc(&d195)
			ctx.EnsureDesc(&d172)
			if d195.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r177, uint64(d195.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r177, d195.Reg)
				ctx.W.EmitShlRegImm8(r177, 3)
			}
			if d172.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d172.Imm.Int()))
				ctx.W.EmitAddInt64(r177, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r177, d172.Reg)
			}
			r178 := ctx.AllocRegExcept(r177)
			ctx.W.EmitMovRegMem(r178, r177, 0)
			ctx.FreeReg(r177)
			d196 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r178}
			ctx.BindReg(r178, &d196)
			ctx.FreeDesc(&d195)
			ctx.EnsureDesc(&d171)
			var d197 scm.JITValueDesc
			if d171.Loc == scm.LocImm {
				d197 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d171.Imm.Int() % 64)}
			} else {
				r179 := ctx.AllocRegExcept(d171.Reg)
				ctx.W.EmitMovRegReg(r179, d171.Reg)
				ctx.W.EmitAndRegImm32(r179, 63)
				d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r179}
				ctx.BindReg(r179, &d197)
			}
			if d197.Loc == scm.LocReg && d171.Loc == scm.LocReg && d197.Reg == d171.Reg {
				ctx.TransferReg(d171.Reg)
				d171.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d171)
			d198 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d197)
			ctx.EnsureDesc(&d198)
			ctx.EnsureDesc(&d197)
			ctx.EnsureDesc(&d198)
			ctx.EnsureDesc(&d197)
			var d199 scm.JITValueDesc
			if d198.Loc == scm.LocImm && d197.Loc == scm.LocImm {
				d199 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d198.Imm.Int() - d197.Imm.Int())}
			} else if d197.Loc == scm.LocImm && d197.Imm.Int() == 0 {
				r180 := ctx.AllocRegExcept(d198.Reg)
				ctx.W.EmitMovRegReg(r180, d198.Reg)
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r180}
				ctx.BindReg(r180, &d199)
			} else if d198.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d197.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d198.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d197.Reg)
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d199)
			} else if d197.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d198.Reg)
				ctx.W.EmitMovRegReg(scratch, d198.Reg)
				if d197.Imm.Int() >= -2147483648 && d197.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d197.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d197.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d199)
			} else {
				r181 := ctx.AllocRegExcept(d198.Reg, d197.Reg)
				ctx.W.EmitMovRegReg(r181, d198.Reg)
				ctx.W.EmitSubInt64(r181, d197.Reg)
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r181}
				ctx.BindReg(r181, &d199)
			}
			if d199.Loc == scm.LocReg && d198.Loc == scm.LocReg && d199.Reg == d198.Reg {
				ctx.TransferReg(d198.Reg)
				d198.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d197)
			ctx.EnsureDesc(&d196)
			ctx.EnsureDesc(&d199)
			var d200 scm.JITValueDesc
			if d196.Loc == scm.LocImm && d199.Loc == scm.LocImm {
				d200 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d196.Imm.Int()) >> uint64(d199.Imm.Int())))}
			} else if d199.Loc == scm.LocImm {
				r182 := ctx.AllocRegExcept(d196.Reg)
				ctx.W.EmitMovRegReg(r182, d196.Reg)
				ctx.W.EmitShrRegImm8(r182, uint8(d199.Imm.Int()))
				d200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r182}
				ctx.BindReg(r182, &d200)
			} else {
				{
					shiftSrc := d196.Reg
					r183 := ctx.AllocRegExcept(d196.Reg)
					ctx.W.EmitMovRegReg(r183, d196.Reg)
					shiftSrc = r183
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d199.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d199.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d199.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d200)
				}
			}
			if d200.Loc == scm.LocReg && d196.Loc == scm.LocReg && d200.Reg == d196.Reg {
				ctx.TransferReg(d196.Reg)
				d196.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d196)
			ctx.FreeDesc(&d199)
			ctx.EnsureDesc(&d176)
			ctx.EnsureDesc(&d200)
			var d201 scm.JITValueDesc
			if d176.Loc == scm.LocImm && d200.Loc == scm.LocImm {
				d201 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d176.Imm.Int() | d200.Imm.Int())}
			} else if d176.Loc == scm.LocImm && d176.Imm.Int() == 0 {
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d200.Reg}
				ctx.BindReg(d200.Reg, &d201)
			} else if d200.Loc == scm.LocImm && d200.Imm.Int() == 0 {
				r184 := ctx.AllocRegExcept(d176.Reg)
				ctx.W.EmitMovRegReg(r184, d176.Reg)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r184}
				ctx.BindReg(r184, &d201)
			} else if d176.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d200.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d176.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d200.Reg)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d201)
			} else if d200.Loc == scm.LocImm {
				r185 := ctx.AllocRegExcept(d176.Reg)
				ctx.W.EmitMovRegReg(r185, d176.Reg)
				if d200.Imm.Int() >= -2147483648 && d200.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r185, int32(d200.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d200.Imm.Int()))
					ctx.W.EmitOrInt64(r185, scm.RegR11)
				}
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r185}
				ctx.BindReg(r185, &d201)
			} else {
				r186 := ctx.AllocRegExcept(d176.Reg, d200.Reg)
				ctx.W.EmitMovRegReg(r186, d176.Reg)
				ctx.W.EmitOrInt64(r186, d200.Reg)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r186}
				ctx.BindReg(r186, &d201)
			}
			if d201.Loc == scm.LocReg && d176.Loc == scm.LocReg && d201.Reg == d176.Reg {
				ctx.TransferReg(d176.Reg)
				d176.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d200)
			d202 = d201
			if d202.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d202)
			ctx.EmitStoreToStack(d202, 24)
			ctx.W.EmitJmp(lbl44)
			ctx.W.MarkLabel(lbl42)
			d203 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r169}
			ctx.BindReg(r169, &d203)
			ctx.BindReg(r169, &d203)
			if r149 { ctx.UnprotectReg(r150) }
			ctx.EnsureDesc(&d203)
			ctx.EnsureDesc(&d203)
			var d204 scm.JITValueDesc
			if d203.Loc == scm.LocImm {
				d204 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d203.Imm.Int()))))}
			} else {
				r187 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r187, d203.Reg)
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r187}
				ctx.BindReg(r187, &d204)
			}
			ctx.FreeDesc(&d203)
			var d205 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d205 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 32)
				r188 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r188, thisptr.Reg, off)
				d205 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r188}
				ctx.BindReg(r188, &d205)
			}
			ctx.EnsureDesc(&d204)
			ctx.EnsureDesc(&d205)
			ctx.EnsureDesc(&d204)
			ctx.EnsureDesc(&d205)
			ctx.EnsureDesc(&d204)
			ctx.EnsureDesc(&d205)
			var d206 scm.JITValueDesc
			if d204.Loc == scm.LocImm && d205.Loc == scm.LocImm {
				d206 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d204.Imm.Int() + d205.Imm.Int())}
			} else if d205.Loc == scm.LocImm && d205.Imm.Int() == 0 {
				r189 := ctx.AllocRegExcept(d204.Reg)
				ctx.W.EmitMovRegReg(r189, d204.Reg)
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r189}
				ctx.BindReg(r189, &d206)
			} else if d204.Loc == scm.LocImm && d204.Imm.Int() == 0 {
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d205.Reg}
				ctx.BindReg(d205.Reg, &d206)
			} else if d204.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d205.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d204.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d205.Reg)
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d206)
			} else if d205.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d204.Reg)
				ctx.W.EmitMovRegReg(scratch, d204.Reg)
				if d205.Imm.Int() >= -2147483648 && d205.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d205.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d205.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d206)
			} else {
				r190 := ctx.AllocRegExcept(d204.Reg, d205.Reg)
				ctx.W.EmitMovRegReg(r190, d204.Reg)
				ctx.W.EmitAddInt64(r190, d205.Reg)
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r190}
				ctx.BindReg(r190, &d206)
			}
			if d206.Loc == scm.LocReg && d204.Loc == scm.LocReg && d206.Reg == d204.Reg {
				ctx.TransferReg(d204.Reg)
				d204.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d204)
			ctx.FreeDesc(&d205)
			ctx.EnsureDesc(&d98)
			d207 = d98
			_ = d207
			r191 := d98.Loc == scm.LocReg
			r192 := d98.Reg
			if r191 { ctx.ProtectReg(r192) }
			d208 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			lbl50 := ctx.W.ReserveLabel()
			bbpos_5_0 := int32(-1)
			_ = bbpos_5_0
			bbpos_5_1 := int32(-1)
			_ = bbpos_5_1
			bbpos_5_2 := int32(-1)
			_ = bbpos_5_2
			bbpos_5_3 := int32(-1)
			_ = bbpos_5_3
			bbpos_5_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			d208 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d207)
			ctx.EnsureDesc(&d207)
			var d209 scm.JITValueDesc
			if d207.Loc == scm.LocImm {
				d209 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d207.Imm.Int()))))}
			} else {
				r193 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r193, d207.Reg)
				ctx.W.EmitShlRegImm8(r193, 32)
				ctx.W.EmitShrRegImm8(r193, 32)
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r193}
				ctx.BindReg(r193, &d209)
			}
			var d210 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d210 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r194 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r194, thisptr.Reg, off)
				d210 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r194}
				ctx.BindReg(r194, &d210)
			}
			ctx.EnsureDesc(&d210)
			ctx.EnsureDesc(&d210)
			var d211 scm.JITValueDesc
			if d210.Loc == scm.LocImm {
				d211 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d210.Imm.Int()))))}
			} else {
				r195 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r195, d210.Reg)
				ctx.W.EmitShlRegImm8(r195, 56)
				ctx.W.EmitShrRegImm8(r195, 56)
				d211 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r195}
				ctx.BindReg(r195, &d211)
			}
			ctx.FreeDesc(&d210)
			ctx.EnsureDesc(&d209)
			ctx.EnsureDesc(&d211)
			ctx.EnsureDesc(&d209)
			ctx.EnsureDesc(&d211)
			ctx.EnsureDesc(&d209)
			ctx.EnsureDesc(&d211)
			var d212 scm.JITValueDesc
			if d209.Loc == scm.LocImm && d211.Loc == scm.LocImm {
				d212 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d209.Imm.Int() * d211.Imm.Int())}
			} else if d209.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d211.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d209.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d211.Reg)
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d212)
			} else if d211.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d209.Reg)
				ctx.W.EmitMovRegReg(scratch, d209.Reg)
				if d211.Imm.Int() >= -2147483648 && d211.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d211.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d211.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d212)
			} else {
				r196 := ctx.AllocRegExcept(d209.Reg, d211.Reg)
				ctx.W.EmitMovRegReg(r196, d209.Reg)
				ctx.W.EmitImulInt64(r196, d211.Reg)
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r196}
				ctx.BindReg(r196, &d212)
			}
			if d212.Loc == scm.LocReg && d209.Loc == scm.LocReg && d212.Reg == d209.Reg {
				ctx.TransferReg(d209.Reg)
				d209.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d209)
			ctx.FreeDesc(&d211)
			var d213 scm.JITValueDesc
			r197 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r197, uint64(dataPtr))
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r197, StackOff: int32(sliceLen)}
				ctx.BindReg(r197, &d213)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 0)
				ctx.W.EmitMovRegMem(r197, thisptr.Reg, off)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r197}
				ctx.BindReg(r197, &d213)
			}
			ctx.BindReg(r197, &d213)
			ctx.EnsureDesc(&d212)
			var d214 scm.JITValueDesc
			if d212.Loc == scm.LocImm {
				d214 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d212.Imm.Int() / 64)}
			} else {
				r198 := ctx.AllocRegExcept(d212.Reg)
				ctx.W.EmitMovRegReg(r198, d212.Reg)
				ctx.W.EmitShrRegImm8(r198, 6)
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r198}
				ctx.BindReg(r198, &d214)
			}
			if d214.Loc == scm.LocReg && d212.Loc == scm.LocReg && d214.Reg == d212.Reg {
				ctx.TransferReg(d212.Reg)
				d212.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d214)
			r199 := ctx.AllocReg()
			ctx.EnsureDesc(&d214)
			ctx.EnsureDesc(&d213)
			if d214.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r199, uint64(d214.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r199, d214.Reg)
				ctx.W.EmitShlRegImm8(r199, 3)
			}
			if d213.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d213.Imm.Int()))
				ctx.W.EmitAddInt64(r199, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r199, d213.Reg)
			}
			r200 := ctx.AllocRegExcept(r199)
			ctx.W.EmitMovRegMem(r200, r199, 0)
			ctx.FreeReg(r199)
			d215 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r200}
			ctx.BindReg(r200, &d215)
			ctx.FreeDesc(&d214)
			ctx.EnsureDesc(&d212)
			var d216 scm.JITValueDesc
			if d212.Loc == scm.LocImm {
				d216 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d212.Imm.Int() % 64)}
			} else {
				r201 := ctx.AllocRegExcept(d212.Reg)
				ctx.W.EmitMovRegReg(r201, d212.Reg)
				ctx.W.EmitAndRegImm32(r201, 63)
				d216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r201}
				ctx.BindReg(r201, &d216)
			}
			if d216.Loc == scm.LocReg && d212.Loc == scm.LocReg && d216.Reg == d212.Reg {
				ctx.TransferReg(d212.Reg)
				d212.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d215)
			ctx.EnsureDesc(&d216)
			var d217 scm.JITValueDesc
			if d215.Loc == scm.LocImm && d216.Loc == scm.LocImm {
				d217 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d215.Imm.Int()) << uint64(d216.Imm.Int())))}
			} else if d216.Loc == scm.LocImm {
				r202 := ctx.AllocRegExcept(d215.Reg)
				ctx.W.EmitMovRegReg(r202, d215.Reg)
				ctx.W.EmitShlRegImm8(r202, uint8(d216.Imm.Int()))
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r202}
				ctx.BindReg(r202, &d217)
			} else {
				{
					shiftSrc := d215.Reg
					r203 := ctx.AllocRegExcept(d215.Reg)
					ctx.W.EmitMovRegReg(r203, d215.Reg)
					shiftSrc = r203
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d216.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d216.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d216.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d217)
				}
			}
			if d217.Loc == scm.LocReg && d215.Loc == scm.LocReg && d217.Reg == d215.Reg {
				ctx.TransferReg(d215.Reg)
				d215.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d215)
			ctx.FreeDesc(&d216)
			var d218 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d218 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 25)
				r204 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r204, thisptr.Reg, off)
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r204}
				ctx.BindReg(r204, &d218)
			}
			d219 = d218
			ctx.EnsureDesc(&d219)
			if d219.Loc != scm.LocImm && d219.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl51 := ctx.W.ReserveLabel()
			lbl52 := ctx.W.ReserveLabel()
			lbl53 := ctx.W.ReserveLabel()
			lbl54 := ctx.W.ReserveLabel()
			if d219.Loc == scm.LocImm {
				if d219.Imm.Bool() {
					ctx.W.MarkLabel(lbl53)
					ctx.W.EmitJmp(lbl51)
				} else {
					ctx.W.MarkLabel(lbl54)
			d220 = d217
			if d220.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d220)
			ctx.EmitStoreToStack(d220, 32)
					ctx.W.EmitJmp(lbl52)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d219.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl53)
				ctx.W.EmitJmp(lbl54)
				ctx.W.MarkLabel(lbl53)
				ctx.W.EmitJmp(lbl51)
				ctx.W.MarkLabel(lbl54)
			d221 = d217
			if d221.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d221)
			ctx.EmitStoreToStack(d221, 32)
				ctx.W.EmitJmp(lbl52)
			}
			ctx.FreeDesc(&d218)
			bbpos_5_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl52)
			ctx.W.ResolveFixups()
			d208 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			var d222 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d222 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r205 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r205, thisptr.Reg, off)
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r205}
				ctx.BindReg(r205, &d222)
			}
			ctx.EnsureDesc(&d222)
			ctx.EnsureDesc(&d222)
			var d223 scm.JITValueDesc
			if d222.Loc == scm.LocImm {
				d223 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d222.Imm.Int()))))}
			} else {
				r206 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r206, d222.Reg)
				ctx.W.EmitShlRegImm8(r206, 56)
				ctx.W.EmitShrRegImm8(r206, 56)
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r206}
				ctx.BindReg(r206, &d223)
			}
			ctx.FreeDesc(&d222)
			d224 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d223)
			ctx.EnsureDesc(&d224)
			ctx.EnsureDesc(&d223)
			ctx.EnsureDesc(&d224)
			ctx.EnsureDesc(&d223)
			var d225 scm.JITValueDesc
			if d224.Loc == scm.LocImm && d223.Loc == scm.LocImm {
				d225 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d224.Imm.Int() - d223.Imm.Int())}
			} else if d223.Loc == scm.LocImm && d223.Imm.Int() == 0 {
				r207 := ctx.AllocRegExcept(d224.Reg)
				ctx.W.EmitMovRegReg(r207, d224.Reg)
				d225 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r207}
				ctx.BindReg(r207, &d225)
			} else if d224.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d223.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d224.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d223.Reg)
				d225 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d225)
			} else if d223.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d224.Reg)
				ctx.W.EmitMovRegReg(scratch, d224.Reg)
				if d223.Imm.Int() >= -2147483648 && d223.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d223.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d223.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d225 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d225)
			} else {
				r208 := ctx.AllocRegExcept(d224.Reg, d223.Reg)
				ctx.W.EmitMovRegReg(r208, d224.Reg)
				ctx.W.EmitSubInt64(r208, d223.Reg)
				d225 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r208}
				ctx.BindReg(r208, &d225)
			}
			if d225.Loc == scm.LocReg && d224.Loc == scm.LocReg && d225.Reg == d224.Reg {
				ctx.TransferReg(d224.Reg)
				d224.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d223)
			ctx.EnsureDesc(&d208)
			ctx.EnsureDesc(&d225)
			var d226 scm.JITValueDesc
			if d208.Loc == scm.LocImm && d225.Loc == scm.LocImm {
				d226 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d208.Imm.Int()) >> uint64(d225.Imm.Int())))}
			} else if d225.Loc == scm.LocImm {
				r209 := ctx.AllocRegExcept(d208.Reg)
				ctx.W.EmitMovRegReg(r209, d208.Reg)
				ctx.W.EmitShrRegImm8(r209, uint8(d225.Imm.Int()))
				d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r209}
				ctx.BindReg(r209, &d226)
			} else {
				{
					shiftSrc := d208.Reg
					r210 := ctx.AllocRegExcept(d208.Reg)
					ctx.W.EmitMovRegReg(r210, d208.Reg)
					shiftSrc = r210
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d225.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d225.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d225.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d226)
				}
			}
			if d226.Loc == scm.LocReg && d208.Loc == scm.LocReg && d226.Reg == d208.Reg {
				ctx.TransferReg(d208.Reg)
				d208.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d208)
			ctx.FreeDesc(&d225)
			r211 := ctx.AllocReg()
			ctx.EnsureDesc(&d226)
			ctx.EnsureDesc(&d226)
			if d226.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r211, d226)
			}
			ctx.W.EmitJmp(lbl50)
			bbpos_5_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl51)
			ctx.W.ResolveFixups()
			d208 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d212)
			var d227 scm.JITValueDesc
			if d212.Loc == scm.LocImm {
				d227 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d212.Imm.Int() % 64)}
			} else {
				r212 := ctx.AllocRegExcept(d212.Reg)
				ctx.W.EmitMovRegReg(r212, d212.Reg)
				ctx.W.EmitAndRegImm32(r212, 63)
				d227 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r212}
				ctx.BindReg(r212, &d227)
			}
			if d227.Loc == scm.LocReg && d212.Loc == scm.LocReg && d227.Reg == d212.Reg {
				ctx.TransferReg(d212.Reg)
				d212.Loc = scm.LocNone
			}
			var d228 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d228 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r213 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r213, thisptr.Reg, off)
				d228 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r213}
				ctx.BindReg(r213, &d228)
			}
			ctx.EnsureDesc(&d228)
			ctx.EnsureDesc(&d228)
			var d229 scm.JITValueDesc
			if d228.Loc == scm.LocImm {
				d229 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d228.Imm.Int()))))}
			} else {
				r214 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r214, d228.Reg)
				ctx.W.EmitShlRegImm8(r214, 56)
				ctx.W.EmitShrRegImm8(r214, 56)
				d229 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r214}
				ctx.BindReg(r214, &d229)
			}
			ctx.FreeDesc(&d228)
			ctx.EnsureDesc(&d227)
			ctx.EnsureDesc(&d229)
			ctx.EnsureDesc(&d227)
			ctx.EnsureDesc(&d229)
			ctx.EnsureDesc(&d227)
			ctx.EnsureDesc(&d229)
			var d230 scm.JITValueDesc
			if d227.Loc == scm.LocImm && d229.Loc == scm.LocImm {
				d230 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d227.Imm.Int() + d229.Imm.Int())}
			} else if d229.Loc == scm.LocImm && d229.Imm.Int() == 0 {
				r215 := ctx.AllocRegExcept(d227.Reg)
				ctx.W.EmitMovRegReg(r215, d227.Reg)
				d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r215}
				ctx.BindReg(r215, &d230)
			} else if d227.Loc == scm.LocImm && d227.Imm.Int() == 0 {
				d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d229.Reg}
				ctx.BindReg(d229.Reg, &d230)
			} else if d227.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d229.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d227.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d229.Reg)
				d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d230)
			} else if d229.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d227.Reg)
				ctx.W.EmitMovRegReg(scratch, d227.Reg)
				if d229.Imm.Int() >= -2147483648 && d229.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d229.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d229.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d230)
			} else {
				r216 := ctx.AllocRegExcept(d227.Reg, d229.Reg)
				ctx.W.EmitMovRegReg(r216, d227.Reg)
				ctx.W.EmitAddInt64(r216, d229.Reg)
				d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r216}
				ctx.BindReg(r216, &d230)
			}
			if d230.Loc == scm.LocReg && d227.Loc == scm.LocReg && d230.Reg == d227.Reg {
				ctx.TransferReg(d227.Reg)
				d227.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d227)
			ctx.FreeDesc(&d229)
			ctx.EnsureDesc(&d230)
			var d231 scm.JITValueDesc
			if d230.Loc == scm.LocImm {
				d231 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d230.Imm.Int()) > uint64(64))}
			} else {
				r217 := ctx.AllocRegExcept(d230.Reg)
				ctx.W.EmitCmpRegImm32(d230.Reg, 64)
				ctx.W.EmitSetcc(r217, scm.CcA)
				d231 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r217}
				ctx.BindReg(r217, &d231)
			}
			ctx.FreeDesc(&d230)
			d232 = d231
			ctx.EnsureDesc(&d232)
			if d232.Loc != scm.LocImm && d232.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl55 := ctx.W.ReserveLabel()
			lbl56 := ctx.W.ReserveLabel()
			lbl57 := ctx.W.ReserveLabel()
			if d232.Loc == scm.LocImm {
				if d232.Imm.Bool() {
					ctx.W.MarkLabel(lbl56)
					ctx.W.EmitJmp(lbl55)
				} else {
					ctx.W.MarkLabel(lbl57)
			d233 = d217
			if d233.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d233)
			ctx.EmitStoreToStack(d233, 32)
					ctx.W.EmitJmp(lbl52)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d232.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl56)
				ctx.W.EmitJmp(lbl57)
				ctx.W.MarkLabel(lbl56)
				ctx.W.EmitJmp(lbl55)
				ctx.W.MarkLabel(lbl57)
			d234 = d217
			if d234.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d234)
			ctx.EmitStoreToStack(d234, 32)
				ctx.W.EmitJmp(lbl52)
			}
			ctx.FreeDesc(&d231)
			bbpos_5_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl55)
			ctx.W.ResolveFixups()
			d208 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d212)
			var d235 scm.JITValueDesc
			if d212.Loc == scm.LocImm {
				d235 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d212.Imm.Int() / 64)}
			} else {
				r218 := ctx.AllocRegExcept(d212.Reg)
				ctx.W.EmitMovRegReg(r218, d212.Reg)
				ctx.W.EmitShrRegImm8(r218, 6)
				d235 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r218}
				ctx.BindReg(r218, &d235)
			}
			if d235.Loc == scm.LocReg && d212.Loc == scm.LocReg && d235.Reg == d212.Reg {
				ctx.TransferReg(d212.Reg)
				d212.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d235)
			ctx.EnsureDesc(&d235)
			var d236 scm.JITValueDesc
			if d235.Loc == scm.LocImm {
				d236 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d235.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d235.Reg)
				ctx.W.EmitMovRegReg(scratch, d235.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d236 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d236)
			}
			if d236.Loc == scm.LocReg && d235.Loc == scm.LocReg && d236.Reg == d235.Reg {
				ctx.TransferReg(d235.Reg)
				d235.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d235)
			ctx.EnsureDesc(&d236)
			r219 := ctx.AllocReg()
			ctx.EnsureDesc(&d236)
			ctx.EnsureDesc(&d213)
			if d236.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r219, uint64(d236.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r219, d236.Reg)
				ctx.W.EmitShlRegImm8(r219, 3)
			}
			if d213.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d213.Imm.Int()))
				ctx.W.EmitAddInt64(r219, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r219, d213.Reg)
			}
			r220 := ctx.AllocRegExcept(r219)
			ctx.W.EmitMovRegMem(r220, r219, 0)
			ctx.FreeReg(r219)
			d237 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r220}
			ctx.BindReg(r220, &d237)
			ctx.FreeDesc(&d236)
			ctx.EnsureDesc(&d212)
			var d238 scm.JITValueDesc
			if d212.Loc == scm.LocImm {
				d238 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d212.Imm.Int() % 64)}
			} else {
				r221 := ctx.AllocRegExcept(d212.Reg)
				ctx.W.EmitMovRegReg(r221, d212.Reg)
				ctx.W.EmitAndRegImm32(r221, 63)
				d238 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r221}
				ctx.BindReg(r221, &d238)
			}
			if d238.Loc == scm.LocReg && d212.Loc == scm.LocReg && d238.Reg == d212.Reg {
				ctx.TransferReg(d212.Reg)
				d212.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d212)
			d239 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d238)
			ctx.EnsureDesc(&d239)
			ctx.EnsureDesc(&d238)
			ctx.EnsureDesc(&d239)
			ctx.EnsureDesc(&d238)
			var d240 scm.JITValueDesc
			if d239.Loc == scm.LocImm && d238.Loc == scm.LocImm {
				d240 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d239.Imm.Int() - d238.Imm.Int())}
			} else if d238.Loc == scm.LocImm && d238.Imm.Int() == 0 {
				r222 := ctx.AllocRegExcept(d239.Reg)
				ctx.W.EmitMovRegReg(r222, d239.Reg)
				d240 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r222}
				ctx.BindReg(r222, &d240)
			} else if d239.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d238.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d239.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d238.Reg)
				d240 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d240)
			} else if d238.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d239.Reg)
				ctx.W.EmitMovRegReg(scratch, d239.Reg)
				if d238.Imm.Int() >= -2147483648 && d238.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d238.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d238.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d240 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d240)
			} else {
				r223 := ctx.AllocRegExcept(d239.Reg, d238.Reg)
				ctx.W.EmitMovRegReg(r223, d239.Reg)
				ctx.W.EmitSubInt64(r223, d238.Reg)
				d240 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r223}
				ctx.BindReg(r223, &d240)
			}
			if d240.Loc == scm.LocReg && d239.Loc == scm.LocReg && d240.Reg == d239.Reg {
				ctx.TransferReg(d239.Reg)
				d239.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d238)
			ctx.EnsureDesc(&d237)
			ctx.EnsureDesc(&d240)
			var d241 scm.JITValueDesc
			if d237.Loc == scm.LocImm && d240.Loc == scm.LocImm {
				d241 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d237.Imm.Int()) >> uint64(d240.Imm.Int())))}
			} else if d240.Loc == scm.LocImm {
				r224 := ctx.AllocRegExcept(d237.Reg)
				ctx.W.EmitMovRegReg(r224, d237.Reg)
				ctx.W.EmitShrRegImm8(r224, uint8(d240.Imm.Int()))
				d241 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r224}
				ctx.BindReg(r224, &d241)
			} else {
				{
					shiftSrc := d237.Reg
					r225 := ctx.AllocRegExcept(d237.Reg)
					ctx.W.EmitMovRegReg(r225, d237.Reg)
					shiftSrc = r225
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d240.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d240.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d240.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d241 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d241)
				}
			}
			if d241.Loc == scm.LocReg && d237.Loc == scm.LocReg && d241.Reg == d237.Reg {
				ctx.TransferReg(d237.Reg)
				d237.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d237)
			ctx.FreeDesc(&d240)
			ctx.EnsureDesc(&d217)
			ctx.EnsureDesc(&d241)
			var d242 scm.JITValueDesc
			if d217.Loc == scm.LocImm && d241.Loc == scm.LocImm {
				d242 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d217.Imm.Int() | d241.Imm.Int())}
			} else if d217.Loc == scm.LocImm && d217.Imm.Int() == 0 {
				d242 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d241.Reg}
				ctx.BindReg(d241.Reg, &d242)
			} else if d241.Loc == scm.LocImm && d241.Imm.Int() == 0 {
				r226 := ctx.AllocRegExcept(d217.Reg)
				ctx.W.EmitMovRegReg(r226, d217.Reg)
				d242 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r226}
				ctx.BindReg(r226, &d242)
			} else if d217.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d241.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d217.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d241.Reg)
				d242 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d242)
			} else if d241.Loc == scm.LocImm {
				r227 := ctx.AllocRegExcept(d217.Reg)
				ctx.W.EmitMovRegReg(r227, d217.Reg)
				if d241.Imm.Int() >= -2147483648 && d241.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r227, int32(d241.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d241.Imm.Int()))
					ctx.W.EmitOrInt64(r227, scm.RegR11)
				}
				d242 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r227}
				ctx.BindReg(r227, &d242)
			} else {
				r228 := ctx.AllocRegExcept(d217.Reg, d241.Reg)
				ctx.W.EmitMovRegReg(r228, d217.Reg)
				ctx.W.EmitOrInt64(r228, d241.Reg)
				d242 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r228}
				ctx.BindReg(r228, &d242)
			}
			if d242.Loc == scm.LocReg && d217.Loc == scm.LocReg && d242.Reg == d217.Reg {
				ctx.TransferReg(d217.Reg)
				d217.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d241)
			d243 = d242
			if d243.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d243)
			ctx.EmitStoreToStack(d243, 32)
			ctx.W.EmitJmp(lbl52)
			ctx.W.MarkLabel(lbl50)
			d244 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r211}
			ctx.BindReg(r211, &d244)
			ctx.BindReg(r211, &d244)
			if r191 { ctx.UnprotectReg(r192) }
			ctx.EnsureDesc(&d244)
			ctx.EnsureDesc(&d244)
			var d245 scm.JITValueDesc
			if d244.Loc == scm.LocImm {
				d245 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d244.Imm.Int()))))}
			} else {
				r229 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r229, d244.Reg)
				d245 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r229}
				ctx.BindReg(r229, &d245)
			}
			ctx.FreeDesc(&d244)
			var d246 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d246 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 32)
				r230 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r230, thisptr.Reg, off)
				d246 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r230}
				ctx.BindReg(r230, &d246)
			}
			ctx.EnsureDesc(&d245)
			ctx.EnsureDesc(&d246)
			ctx.EnsureDesc(&d245)
			ctx.EnsureDesc(&d246)
			ctx.EnsureDesc(&d245)
			ctx.EnsureDesc(&d246)
			var d247 scm.JITValueDesc
			if d245.Loc == scm.LocImm && d246.Loc == scm.LocImm {
				d247 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d245.Imm.Int() + d246.Imm.Int())}
			} else if d246.Loc == scm.LocImm && d246.Imm.Int() == 0 {
				r231 := ctx.AllocRegExcept(d245.Reg)
				ctx.W.EmitMovRegReg(r231, d245.Reg)
				d247 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r231}
				ctx.BindReg(r231, &d247)
			} else if d245.Loc == scm.LocImm && d245.Imm.Int() == 0 {
				d247 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d246.Reg}
				ctx.BindReg(d246.Reg, &d247)
			} else if d245.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d246.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d245.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d246.Reg)
				d247 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d247)
			} else if d246.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d245.Reg)
				ctx.W.EmitMovRegReg(scratch, d245.Reg)
				if d246.Imm.Int() >= -2147483648 && d246.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d246.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d246.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d247 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d247)
			} else {
				r232 := ctx.AllocRegExcept(d245.Reg, d246.Reg)
				ctx.W.EmitMovRegReg(r232, d245.Reg)
				ctx.W.EmitAddInt64(r232, d246.Reg)
				d247 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r232}
				ctx.BindReg(r232, &d247)
			}
			if d247.Loc == scm.LocReg && d245.Loc == scm.LocReg && d247.Reg == d245.Reg {
				ctx.TransferReg(d245.Reg)
				d245.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d245)
			ctx.FreeDesc(&d246)
			ctx.EnsureDesc(&d206)
			ctx.EnsureDesc(&d206)
			ctx.EnsureDesc(&d206)
			ctx.EnsureDesc(&d247)
			ctx.EnsureDesc(&d206)
			ctx.EnsureDesc(&d247)
			ctx.EnsureDesc(&d206)
			ctx.EnsureDesc(&d247)
			var d249 scm.JITValueDesc
			if d206.Loc == scm.LocImm && d247.Loc == scm.LocImm {
				d249 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d206.Imm.Int() + d247.Imm.Int())}
			} else if d247.Loc == scm.LocImm && d247.Imm.Int() == 0 {
				r233 := ctx.AllocRegExcept(d206.Reg)
				ctx.W.EmitMovRegReg(r233, d206.Reg)
				d249 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r233}
				ctx.BindReg(r233, &d249)
			} else if d206.Loc == scm.LocImm && d206.Imm.Int() == 0 {
				d249 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d247.Reg}
				ctx.BindReg(d247.Reg, &d249)
			} else if d206.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d247.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d206.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d247.Reg)
				d249 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d249)
			} else if d247.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d206.Reg)
				ctx.W.EmitMovRegReg(scratch, d206.Reg)
				if d247.Imm.Int() >= -2147483648 && d247.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d247.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d247.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d249 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d249)
			} else {
				r234 := ctx.AllocRegExcept(d206.Reg, d247.Reg)
				ctx.W.EmitMovRegReg(r234, d206.Reg)
				ctx.W.EmitAddInt64(r234, d247.Reg)
				d249 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
				ctx.BindReg(r234, &d249)
			}
			if d249.Loc == scm.LocReg && d206.Loc == scm.LocReg && d249.Reg == d206.Reg {
				ctx.TransferReg(d206.Reg)
				d206.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d247)
			ctx.EnsureDesc(&d249)
			ctx.EnsureDesc(&d249)
			ctx.EnsureDesc(&d206)
			ctx.EnsureDesc(&d249)
			ctx.EnsureDesc(&d153)
			ctx.EnsureDesc(&d206)
			ctx.EnsureDesc(&d249)
			r235 := ctx.AllocReg()
			r236 := ctx.AllocRegExcept(r235)
			ctx.EnsureDesc(&d153)
			ctx.EnsureDesc(&d206)
			ctx.EnsureDesc(&d249)
			if d153.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r235, uint64(d153.Imm.Int()))
			} else if d153.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r235, d153.Reg)
			} else {
				ctx.W.EmitMovRegReg(r235, d153.Reg)
			}
			if d206.Loc == scm.LocImm {
				if d206.Imm.Int() != 0 {
					if d206.Imm.Int() >= -2147483648 && d206.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r235, int32(d206.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d206.Imm.Int()))
						ctx.W.EmitAddInt64(r235, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r235, d206.Reg)
			}
			if d249.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r236, uint64(d249.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r236, d249.Reg)
			}
			if d206.Loc == scm.LocImm {
				if d206.Imm.Int() >= -2147483648 && d206.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r236, int32(d206.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d206.Imm.Int()))
					ctx.W.EmitSubInt64(r236, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r236, d206.Reg)
			}
			d251 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r235, Reg2: r236}
			ctx.BindReg(r235, &d251)
			ctx.BindReg(r236, &d251)
			ctx.FreeDesc(&d206)
			ctx.FreeDesc(&d249)
			d252 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d252)
			ctx.BindReg(r1, &d252)
			d253 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d251}, 2)
			ctx.EmitMovPairToResult(&d253, &d252)
			ctx.W.EmitJmp(lbl0)
			return result
			}
			bbs[8].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[8].VisitCount >= 2 {
					ps.General = true
					return bbs[8].RenderPS(ps)
				}
			}
			bbs[8].VisitCount++
			if ps.General {
				if bbs[8].Rendered {
					ctx.W.EmitJmp(lbl9)
					return result
				}
				bbs[8].Rendered = true
				bbs[8].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_8 = bbs[8].Address
				ctx.W.MarkLabel(lbl9)
				ctx.W.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
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
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
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
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != scm.LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != scm.LocNone {
				d70 = ps.OverlayValues[70]
			}
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != scm.LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != scm.LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != scm.LocNone {
				d73 = ps.OverlayValues[73]
			}
			if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != scm.LocNone {
				d74 = ps.OverlayValues[74]
			}
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != scm.LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != scm.LocNone {
				d76 = ps.OverlayValues[76]
			}
			if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
				d77 = ps.OverlayValues[77]
			}
			if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
				d78 = ps.OverlayValues[78]
			}
			if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
				d79 = ps.OverlayValues[79]
			}
			if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
				d80 = ps.OverlayValues[80]
			}
			if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
				d81 = ps.OverlayValues[81]
			}
			if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
				d82 = ps.OverlayValues[82]
			}
			if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
				d83 = ps.OverlayValues[83]
			}
			if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != scm.LocNone {
				d84 = ps.OverlayValues[84]
			}
			if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
				d85 = ps.OverlayValues[85]
			}
			if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
				d86 = ps.OverlayValues[86]
			}
			if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != scm.LocNone {
				d87 = ps.OverlayValues[87]
			}
			if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != scm.LocNone {
				d88 = ps.OverlayValues[88]
			}
			if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != scm.LocNone {
				d89 = ps.OverlayValues[89]
			}
			if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != scm.LocNone {
				d90 = ps.OverlayValues[90]
			}
			if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != scm.LocNone {
				d91 = ps.OverlayValues[91]
			}
			if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != scm.LocNone {
				d92 = ps.OverlayValues[92]
			}
			if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != scm.LocNone {
				d93 = ps.OverlayValues[93]
			}
			if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != scm.LocNone {
				d94 = ps.OverlayValues[94]
			}
			if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != scm.LocNone {
				d95 = ps.OverlayValues[95]
			}
			if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != scm.LocNone {
				d96 = ps.OverlayValues[96]
			}
			if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != scm.LocNone {
				d97 = ps.OverlayValues[97]
			}
			if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != scm.LocNone {
				d98 = ps.OverlayValues[98]
			}
			if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != scm.LocNone {
				d99 = ps.OverlayValues[99]
			}
			if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != scm.LocNone {
				d100 = ps.OverlayValues[100]
			}
			if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != scm.LocNone {
				d107 = ps.OverlayValues[107]
			}
			if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != scm.LocNone {
				d108 = ps.OverlayValues[108]
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
			if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != scm.LocNone {
				d207 = ps.OverlayValues[207]
			}
			if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != scm.LocNone {
				d208 = ps.OverlayValues[208]
			}
			if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != scm.LocNone {
				d209 = ps.OverlayValues[209]
			}
			if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != scm.LocNone {
				d210 = ps.OverlayValues[210]
			}
			if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != scm.LocNone {
				d211 = ps.OverlayValues[211]
			}
			if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != scm.LocNone {
				d212 = ps.OverlayValues[212]
			}
			if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != scm.LocNone {
				d213 = ps.OverlayValues[213]
			}
			if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != scm.LocNone {
				d214 = ps.OverlayValues[214]
			}
			if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != scm.LocNone {
				d215 = ps.OverlayValues[215]
			}
			if len(ps.OverlayValues) > 216 && ps.OverlayValues[216].Loc != scm.LocNone {
				d216 = ps.OverlayValues[216]
			}
			if len(ps.OverlayValues) > 217 && ps.OverlayValues[217].Loc != scm.LocNone {
				d217 = ps.OverlayValues[217]
			}
			if len(ps.OverlayValues) > 218 && ps.OverlayValues[218].Loc != scm.LocNone {
				d218 = ps.OverlayValues[218]
			}
			if len(ps.OverlayValues) > 219 && ps.OverlayValues[219].Loc != scm.LocNone {
				d219 = ps.OverlayValues[219]
			}
			if len(ps.OverlayValues) > 220 && ps.OverlayValues[220].Loc != scm.LocNone {
				d220 = ps.OverlayValues[220]
			}
			if len(ps.OverlayValues) > 221 && ps.OverlayValues[221].Loc != scm.LocNone {
				d221 = ps.OverlayValues[221]
			}
			if len(ps.OverlayValues) > 222 && ps.OverlayValues[222].Loc != scm.LocNone {
				d222 = ps.OverlayValues[222]
			}
			if len(ps.OverlayValues) > 223 && ps.OverlayValues[223].Loc != scm.LocNone {
				d223 = ps.OverlayValues[223]
			}
			if len(ps.OverlayValues) > 224 && ps.OverlayValues[224].Loc != scm.LocNone {
				d224 = ps.OverlayValues[224]
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
			if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != scm.LocNone {
				d229 = ps.OverlayValues[229]
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
			if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
				d238 = ps.OverlayValues[238]
			}
			if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
				d239 = ps.OverlayValues[239]
			}
			if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != scm.LocNone {
				d240 = ps.OverlayValues[240]
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
			if len(ps.OverlayValues) > 246 && ps.OverlayValues[246].Loc != scm.LocNone {
				d246 = ps.OverlayValues[246]
			}
			if len(ps.OverlayValues) > 247 && ps.OverlayValues[247].Loc != scm.LocNone {
				d247 = ps.OverlayValues[247]
			}
			if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != scm.LocNone {
				d248 = ps.OverlayValues[248]
			}
			if len(ps.OverlayValues) > 249 && ps.OverlayValues[249].Loc != scm.LocNone {
				d249 = ps.OverlayValues[249]
			}
			if len(ps.OverlayValues) > 250 && ps.OverlayValues[250].Loc != scm.LocNone {
				d250 = ps.OverlayValues[250]
			}
			if len(ps.OverlayValues) > 251 && ps.OverlayValues[251].Loc != scm.LocNone {
				d251 = ps.OverlayValues[251]
			}
			if len(ps.OverlayValues) > 252 && ps.OverlayValues[252].Loc != scm.LocNone {
				d252 = ps.OverlayValues[252]
			}
			if len(ps.OverlayValues) > 253 && ps.OverlayValues[253].Loc != scm.LocNone {
				d253 = ps.OverlayValues[253]
			}
			ctx.ReclaimUntrackedRegs()
			var d254 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d254 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 64)
				r237 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r237, thisptr.Reg, off)
				d254 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r237}
				ctx.BindReg(r237, &d254)
			}
			ctx.EnsureDesc(&d254)
			ctx.EnsureDesc(&d254)
			var d255 scm.JITValueDesc
			if d254.Loc == scm.LocImm {
				d255 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d254.Imm.Int()))))}
			} else {
				r238 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r238, d254.Reg)
				ctx.W.EmitShlRegImm8(r238, 32)
				ctx.W.EmitShrRegImm8(r238, 32)
				d255 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r238}
				ctx.BindReg(r238, &d255)
			}
			ctx.FreeDesc(&d254)
			ctx.EnsureDesc(&d98)
			ctx.EnsureDesc(&d255)
			ctx.EnsureDesc(&d98)
			ctx.EnsureDesc(&d255)
			ctx.EnsureDesc(&d98)
			ctx.EnsureDesc(&d255)
			var d256 scm.JITValueDesc
			if d98.Loc == scm.LocImm && d255.Loc == scm.LocImm {
				d256 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d98.Imm.Int()) == uint64(d255.Imm.Int()))}
			} else if d255.Loc == scm.LocImm {
				r239 := ctx.AllocRegExcept(d98.Reg)
				if d255.Imm.Int() >= -2147483648 && d255.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d98.Reg, int32(d255.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d255.Imm.Int()))
					ctx.W.EmitCmpInt64(d98.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r239, scm.CcE)
				d256 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r239}
				ctx.BindReg(r239, &d256)
			} else if d98.Loc == scm.LocImm {
				r240 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d98.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d255.Reg)
				ctx.W.EmitSetcc(r240, scm.CcE)
				d256 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r240}
				ctx.BindReg(r240, &d256)
			} else {
				r241 := ctx.AllocRegExcept(d98.Reg)
				ctx.W.EmitCmpInt64(d98.Reg, d255.Reg)
				ctx.W.EmitSetcc(r241, scm.CcE)
				d256 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r241}
				ctx.BindReg(r241, &d256)
			}
			ctx.FreeDesc(&d98)
			ctx.FreeDesc(&d255)
			d257 = d256
			ctx.EnsureDesc(&d257)
			if d257.Loc != scm.LocImm && d257.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d257.Loc == scm.LocImm {
				if d257.Imm.Bool() {
			ps258 := scm.PhiState{General: ps.General}
			ps258.OverlayValues = make([]scm.JITValueDesc, 258)
			ps258.OverlayValues[0] = d0
			ps258.OverlayValues[1] = d1
			ps258.OverlayValues[7] = d7
			ps258.OverlayValues[8] = d8
			ps258.OverlayValues[9] = d9
			ps258.OverlayValues[10] = d10
			ps258.OverlayValues[11] = d11
			ps258.OverlayValues[12] = d12
			ps258.OverlayValues[13] = d13
			ps258.OverlayValues[14] = d14
			ps258.OverlayValues[15] = d15
			ps258.OverlayValues[16] = d16
			ps258.OverlayValues[17] = d17
			ps258.OverlayValues[18] = d18
			ps258.OverlayValues[19] = d19
			ps258.OverlayValues[20] = d20
			ps258.OverlayValues[21] = d21
			ps258.OverlayValues[22] = d22
			ps258.OverlayValues[23] = d23
			ps258.OverlayValues[24] = d24
			ps258.OverlayValues[25] = d25
			ps258.OverlayValues[26] = d26
			ps258.OverlayValues[27] = d27
			ps258.OverlayValues[28] = d28
			ps258.OverlayValues[29] = d29
			ps258.OverlayValues[30] = d30
			ps258.OverlayValues[31] = d31
			ps258.OverlayValues[32] = d32
			ps258.OverlayValues[33] = d33
			ps258.OverlayValues[34] = d34
			ps258.OverlayValues[35] = d35
			ps258.OverlayValues[36] = d36
			ps258.OverlayValues[37] = d37
			ps258.OverlayValues[38] = d38
			ps258.OverlayValues[39] = d39
			ps258.OverlayValues[40] = d40
			ps258.OverlayValues[41] = d41
			ps258.OverlayValues[42] = d42
			ps258.OverlayValues[43] = d43
			ps258.OverlayValues[44] = d44
			ps258.OverlayValues[45] = d45
			ps258.OverlayValues[46] = d46
			ps258.OverlayValues[47] = d47
			ps258.OverlayValues[48] = d48
			ps258.OverlayValues[49] = d49
			ps258.OverlayValues[50] = d50
			ps258.OverlayValues[57] = d57
			ps258.OverlayValues[58] = d58
			ps258.OverlayValues[59] = d59
			ps258.OverlayValues[60] = d60
			ps258.OverlayValues[61] = d61
			ps258.OverlayValues[62] = d62
			ps258.OverlayValues[63] = d63
			ps258.OverlayValues[64] = d64
			ps258.OverlayValues[65] = d65
			ps258.OverlayValues[66] = d66
			ps258.OverlayValues[67] = d67
			ps258.OverlayValues[68] = d68
			ps258.OverlayValues[69] = d69
			ps258.OverlayValues[70] = d70
			ps258.OverlayValues[71] = d71
			ps258.OverlayValues[72] = d72
			ps258.OverlayValues[73] = d73
			ps258.OverlayValues[74] = d74
			ps258.OverlayValues[75] = d75
			ps258.OverlayValues[76] = d76
			ps258.OverlayValues[77] = d77
			ps258.OverlayValues[78] = d78
			ps258.OverlayValues[79] = d79
			ps258.OverlayValues[80] = d80
			ps258.OverlayValues[81] = d81
			ps258.OverlayValues[82] = d82
			ps258.OverlayValues[83] = d83
			ps258.OverlayValues[84] = d84
			ps258.OverlayValues[85] = d85
			ps258.OverlayValues[86] = d86
			ps258.OverlayValues[87] = d87
			ps258.OverlayValues[88] = d88
			ps258.OverlayValues[89] = d89
			ps258.OverlayValues[90] = d90
			ps258.OverlayValues[91] = d91
			ps258.OverlayValues[92] = d92
			ps258.OverlayValues[93] = d93
			ps258.OverlayValues[94] = d94
			ps258.OverlayValues[95] = d95
			ps258.OverlayValues[96] = d96
			ps258.OverlayValues[97] = d97
			ps258.OverlayValues[98] = d98
			ps258.OverlayValues[99] = d99
			ps258.OverlayValues[100] = d100
			ps258.OverlayValues[107] = d107
			ps258.OverlayValues[108] = d108
			ps258.OverlayValues[109] = d109
			ps258.OverlayValues[110] = d110
			ps258.OverlayValues[111] = d111
			ps258.OverlayValues[112] = d112
			ps258.OverlayValues[113] = d113
			ps258.OverlayValues[114] = d114
			ps258.OverlayValues[115] = d115
			ps258.OverlayValues[116] = d116
			ps258.OverlayValues[117] = d117
			ps258.OverlayValues[118] = d118
			ps258.OverlayValues[119] = d119
			ps258.OverlayValues[120] = d120
			ps258.OverlayValues[121] = d121
			ps258.OverlayValues[122] = d122
			ps258.OverlayValues[123] = d123
			ps258.OverlayValues[124] = d124
			ps258.OverlayValues[125] = d125
			ps258.OverlayValues[126] = d126
			ps258.OverlayValues[127] = d127
			ps258.OverlayValues[128] = d128
			ps258.OverlayValues[129] = d129
			ps258.OverlayValues[130] = d130
			ps258.OverlayValues[131] = d131
			ps258.OverlayValues[132] = d132
			ps258.OverlayValues[133] = d133
			ps258.OverlayValues[134] = d134
			ps258.OverlayValues[135] = d135
			ps258.OverlayValues[136] = d136
			ps258.OverlayValues[137] = d137
			ps258.OverlayValues[138] = d138
			ps258.OverlayValues[139] = d139
			ps258.OverlayValues[140] = d140
			ps258.OverlayValues[141] = d141
			ps258.OverlayValues[142] = d142
			ps258.OverlayValues[143] = d143
			ps258.OverlayValues[144] = d144
			ps258.OverlayValues[145] = d145
			ps258.OverlayValues[146] = d146
			ps258.OverlayValues[147] = d147
			ps258.OverlayValues[148] = d148
			ps258.OverlayValues[149] = d149
			ps258.OverlayValues[150] = d150
			ps258.OverlayValues[151] = d151
			ps258.OverlayValues[152] = d152
			ps258.OverlayValues[153] = d153
			ps258.OverlayValues[154] = d154
			ps258.OverlayValues[155] = d155
			ps258.OverlayValues[156] = d156
			ps258.OverlayValues[157] = d157
			ps258.OverlayValues[158] = d158
			ps258.OverlayValues[159] = d159
			ps258.OverlayValues[165] = d165
			ps258.OverlayValues[166] = d166
			ps258.OverlayValues[167] = d167
			ps258.OverlayValues[168] = d168
			ps258.OverlayValues[169] = d169
			ps258.OverlayValues[170] = d170
			ps258.OverlayValues[171] = d171
			ps258.OverlayValues[172] = d172
			ps258.OverlayValues[173] = d173
			ps258.OverlayValues[174] = d174
			ps258.OverlayValues[175] = d175
			ps258.OverlayValues[176] = d176
			ps258.OverlayValues[177] = d177
			ps258.OverlayValues[178] = d178
			ps258.OverlayValues[179] = d179
			ps258.OverlayValues[180] = d180
			ps258.OverlayValues[181] = d181
			ps258.OverlayValues[182] = d182
			ps258.OverlayValues[183] = d183
			ps258.OverlayValues[184] = d184
			ps258.OverlayValues[185] = d185
			ps258.OverlayValues[186] = d186
			ps258.OverlayValues[187] = d187
			ps258.OverlayValues[188] = d188
			ps258.OverlayValues[189] = d189
			ps258.OverlayValues[190] = d190
			ps258.OverlayValues[191] = d191
			ps258.OverlayValues[192] = d192
			ps258.OverlayValues[193] = d193
			ps258.OverlayValues[194] = d194
			ps258.OverlayValues[195] = d195
			ps258.OverlayValues[196] = d196
			ps258.OverlayValues[197] = d197
			ps258.OverlayValues[198] = d198
			ps258.OverlayValues[199] = d199
			ps258.OverlayValues[200] = d200
			ps258.OverlayValues[201] = d201
			ps258.OverlayValues[202] = d202
			ps258.OverlayValues[203] = d203
			ps258.OverlayValues[204] = d204
			ps258.OverlayValues[205] = d205
			ps258.OverlayValues[206] = d206
			ps258.OverlayValues[207] = d207
			ps258.OverlayValues[208] = d208
			ps258.OverlayValues[209] = d209
			ps258.OverlayValues[210] = d210
			ps258.OverlayValues[211] = d211
			ps258.OverlayValues[212] = d212
			ps258.OverlayValues[213] = d213
			ps258.OverlayValues[214] = d214
			ps258.OverlayValues[215] = d215
			ps258.OverlayValues[216] = d216
			ps258.OverlayValues[217] = d217
			ps258.OverlayValues[218] = d218
			ps258.OverlayValues[219] = d219
			ps258.OverlayValues[220] = d220
			ps258.OverlayValues[221] = d221
			ps258.OverlayValues[222] = d222
			ps258.OverlayValues[223] = d223
			ps258.OverlayValues[224] = d224
			ps258.OverlayValues[225] = d225
			ps258.OverlayValues[226] = d226
			ps258.OverlayValues[227] = d227
			ps258.OverlayValues[228] = d228
			ps258.OverlayValues[229] = d229
			ps258.OverlayValues[230] = d230
			ps258.OverlayValues[231] = d231
			ps258.OverlayValues[232] = d232
			ps258.OverlayValues[233] = d233
			ps258.OverlayValues[234] = d234
			ps258.OverlayValues[235] = d235
			ps258.OverlayValues[236] = d236
			ps258.OverlayValues[237] = d237
			ps258.OverlayValues[238] = d238
			ps258.OverlayValues[239] = d239
			ps258.OverlayValues[240] = d240
			ps258.OverlayValues[241] = d241
			ps258.OverlayValues[242] = d242
			ps258.OverlayValues[243] = d243
			ps258.OverlayValues[244] = d244
			ps258.OverlayValues[245] = d245
			ps258.OverlayValues[246] = d246
			ps258.OverlayValues[247] = d247
			ps258.OverlayValues[248] = d248
			ps258.OverlayValues[249] = d249
			ps258.OverlayValues[250] = d250
			ps258.OverlayValues[251] = d251
			ps258.OverlayValues[252] = d252
			ps258.OverlayValues[253] = d253
			ps258.OverlayValues[254] = d254
			ps258.OverlayValues[255] = d255
			ps258.OverlayValues[256] = d256
			ps258.OverlayValues[257] = d257
					return bbs[6].RenderPS(ps258)
				}
			ps259 := scm.PhiState{General: ps.General}
			ps259.OverlayValues = make([]scm.JITValueDesc, 258)
			ps259.OverlayValues[0] = d0
			ps259.OverlayValues[1] = d1
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
			ps259.OverlayValues[20] = d20
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
			ps259.OverlayValues[57] = d57
			ps259.OverlayValues[58] = d58
			ps259.OverlayValues[59] = d59
			ps259.OverlayValues[60] = d60
			ps259.OverlayValues[61] = d61
			ps259.OverlayValues[62] = d62
			ps259.OverlayValues[63] = d63
			ps259.OverlayValues[64] = d64
			ps259.OverlayValues[65] = d65
			ps259.OverlayValues[66] = d66
			ps259.OverlayValues[67] = d67
			ps259.OverlayValues[68] = d68
			ps259.OverlayValues[69] = d69
			ps259.OverlayValues[70] = d70
			ps259.OverlayValues[71] = d71
			ps259.OverlayValues[72] = d72
			ps259.OverlayValues[73] = d73
			ps259.OverlayValues[74] = d74
			ps259.OverlayValues[75] = d75
			ps259.OverlayValues[76] = d76
			ps259.OverlayValues[77] = d77
			ps259.OverlayValues[78] = d78
			ps259.OverlayValues[79] = d79
			ps259.OverlayValues[80] = d80
			ps259.OverlayValues[81] = d81
			ps259.OverlayValues[82] = d82
			ps259.OverlayValues[83] = d83
			ps259.OverlayValues[84] = d84
			ps259.OverlayValues[85] = d85
			ps259.OverlayValues[86] = d86
			ps259.OverlayValues[87] = d87
			ps259.OverlayValues[88] = d88
			ps259.OverlayValues[89] = d89
			ps259.OverlayValues[90] = d90
			ps259.OverlayValues[91] = d91
			ps259.OverlayValues[92] = d92
			ps259.OverlayValues[93] = d93
			ps259.OverlayValues[94] = d94
			ps259.OverlayValues[95] = d95
			ps259.OverlayValues[96] = d96
			ps259.OverlayValues[97] = d97
			ps259.OverlayValues[98] = d98
			ps259.OverlayValues[99] = d99
			ps259.OverlayValues[100] = d100
			ps259.OverlayValues[107] = d107
			ps259.OverlayValues[108] = d108
			ps259.OverlayValues[109] = d109
			ps259.OverlayValues[110] = d110
			ps259.OverlayValues[111] = d111
			ps259.OverlayValues[112] = d112
			ps259.OverlayValues[113] = d113
			ps259.OverlayValues[114] = d114
			ps259.OverlayValues[115] = d115
			ps259.OverlayValues[116] = d116
			ps259.OverlayValues[117] = d117
			ps259.OverlayValues[118] = d118
			ps259.OverlayValues[119] = d119
			ps259.OverlayValues[120] = d120
			ps259.OverlayValues[121] = d121
			ps259.OverlayValues[122] = d122
			ps259.OverlayValues[123] = d123
			ps259.OverlayValues[124] = d124
			ps259.OverlayValues[125] = d125
			ps259.OverlayValues[126] = d126
			ps259.OverlayValues[127] = d127
			ps259.OverlayValues[128] = d128
			ps259.OverlayValues[129] = d129
			ps259.OverlayValues[130] = d130
			ps259.OverlayValues[131] = d131
			ps259.OverlayValues[132] = d132
			ps259.OverlayValues[133] = d133
			ps259.OverlayValues[134] = d134
			ps259.OverlayValues[135] = d135
			ps259.OverlayValues[136] = d136
			ps259.OverlayValues[137] = d137
			ps259.OverlayValues[138] = d138
			ps259.OverlayValues[139] = d139
			ps259.OverlayValues[140] = d140
			ps259.OverlayValues[141] = d141
			ps259.OverlayValues[142] = d142
			ps259.OverlayValues[143] = d143
			ps259.OverlayValues[144] = d144
			ps259.OverlayValues[145] = d145
			ps259.OverlayValues[146] = d146
			ps259.OverlayValues[147] = d147
			ps259.OverlayValues[148] = d148
			ps259.OverlayValues[149] = d149
			ps259.OverlayValues[150] = d150
			ps259.OverlayValues[151] = d151
			ps259.OverlayValues[152] = d152
			ps259.OverlayValues[153] = d153
			ps259.OverlayValues[154] = d154
			ps259.OverlayValues[155] = d155
			ps259.OverlayValues[156] = d156
			ps259.OverlayValues[157] = d157
			ps259.OverlayValues[158] = d158
			ps259.OverlayValues[159] = d159
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
			ps259.OverlayValues[182] = d182
			ps259.OverlayValues[183] = d183
			ps259.OverlayValues[184] = d184
			ps259.OverlayValues[185] = d185
			ps259.OverlayValues[186] = d186
			ps259.OverlayValues[187] = d187
			ps259.OverlayValues[188] = d188
			ps259.OverlayValues[189] = d189
			ps259.OverlayValues[190] = d190
			ps259.OverlayValues[191] = d191
			ps259.OverlayValues[192] = d192
			ps259.OverlayValues[193] = d193
			ps259.OverlayValues[194] = d194
			ps259.OverlayValues[195] = d195
			ps259.OverlayValues[196] = d196
			ps259.OverlayValues[197] = d197
			ps259.OverlayValues[198] = d198
			ps259.OverlayValues[199] = d199
			ps259.OverlayValues[200] = d200
			ps259.OverlayValues[201] = d201
			ps259.OverlayValues[202] = d202
			ps259.OverlayValues[203] = d203
			ps259.OverlayValues[204] = d204
			ps259.OverlayValues[205] = d205
			ps259.OverlayValues[206] = d206
			ps259.OverlayValues[207] = d207
			ps259.OverlayValues[208] = d208
			ps259.OverlayValues[209] = d209
			ps259.OverlayValues[210] = d210
			ps259.OverlayValues[211] = d211
			ps259.OverlayValues[212] = d212
			ps259.OverlayValues[213] = d213
			ps259.OverlayValues[214] = d214
			ps259.OverlayValues[215] = d215
			ps259.OverlayValues[216] = d216
			ps259.OverlayValues[217] = d217
			ps259.OverlayValues[218] = d218
			ps259.OverlayValues[219] = d219
			ps259.OverlayValues[220] = d220
			ps259.OverlayValues[221] = d221
			ps259.OverlayValues[222] = d222
			ps259.OverlayValues[223] = d223
			ps259.OverlayValues[224] = d224
			ps259.OverlayValues[225] = d225
			ps259.OverlayValues[226] = d226
			ps259.OverlayValues[227] = d227
			ps259.OverlayValues[228] = d228
			ps259.OverlayValues[229] = d229
			ps259.OverlayValues[230] = d230
			ps259.OverlayValues[231] = d231
			ps259.OverlayValues[232] = d232
			ps259.OverlayValues[233] = d233
			ps259.OverlayValues[234] = d234
			ps259.OverlayValues[235] = d235
			ps259.OverlayValues[236] = d236
			ps259.OverlayValues[237] = d237
			ps259.OverlayValues[238] = d238
			ps259.OverlayValues[239] = d239
			ps259.OverlayValues[240] = d240
			ps259.OverlayValues[241] = d241
			ps259.OverlayValues[242] = d242
			ps259.OverlayValues[243] = d243
			ps259.OverlayValues[244] = d244
			ps259.OverlayValues[245] = d245
			ps259.OverlayValues[246] = d246
			ps259.OverlayValues[247] = d247
			ps259.OverlayValues[248] = d248
			ps259.OverlayValues[249] = d249
			ps259.OverlayValues[250] = d250
			ps259.OverlayValues[251] = d251
			ps259.OverlayValues[252] = d252
			ps259.OverlayValues[253] = d253
			ps259.OverlayValues[254] = d254
			ps259.OverlayValues[255] = d255
			ps259.OverlayValues[256] = d256
			ps259.OverlayValues[257] = d257
				return bbs[7].RenderPS(ps259)
			}
			lbl58 := ctx.W.ReserveLabel()
			lbl59 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d257.Reg, 0)
			ctx.W.EmitJcc(scm.CcNE, lbl58)
			ctx.W.EmitJmp(lbl59)
			ctx.W.MarkLabel(lbl58)
			ctx.W.EmitJmp(lbl7)
			ctx.W.MarkLabel(lbl59)
			ctx.W.EmitJmp(lbl8)
			ps260 := scm.PhiState{General: true}
			ps260.OverlayValues = make([]scm.JITValueDesc, 258)
			ps260.OverlayValues[0] = d0
			ps260.OverlayValues[1] = d1
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
			ps260.OverlayValues[20] = d20
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
			ps260.OverlayValues[57] = d57
			ps260.OverlayValues[58] = d58
			ps260.OverlayValues[59] = d59
			ps260.OverlayValues[60] = d60
			ps260.OverlayValues[61] = d61
			ps260.OverlayValues[62] = d62
			ps260.OverlayValues[63] = d63
			ps260.OverlayValues[64] = d64
			ps260.OverlayValues[65] = d65
			ps260.OverlayValues[66] = d66
			ps260.OverlayValues[67] = d67
			ps260.OverlayValues[68] = d68
			ps260.OverlayValues[69] = d69
			ps260.OverlayValues[70] = d70
			ps260.OverlayValues[71] = d71
			ps260.OverlayValues[72] = d72
			ps260.OverlayValues[73] = d73
			ps260.OverlayValues[74] = d74
			ps260.OverlayValues[75] = d75
			ps260.OverlayValues[76] = d76
			ps260.OverlayValues[77] = d77
			ps260.OverlayValues[78] = d78
			ps260.OverlayValues[79] = d79
			ps260.OverlayValues[80] = d80
			ps260.OverlayValues[81] = d81
			ps260.OverlayValues[82] = d82
			ps260.OverlayValues[83] = d83
			ps260.OverlayValues[84] = d84
			ps260.OverlayValues[85] = d85
			ps260.OverlayValues[86] = d86
			ps260.OverlayValues[87] = d87
			ps260.OverlayValues[88] = d88
			ps260.OverlayValues[89] = d89
			ps260.OverlayValues[90] = d90
			ps260.OverlayValues[91] = d91
			ps260.OverlayValues[92] = d92
			ps260.OverlayValues[93] = d93
			ps260.OverlayValues[94] = d94
			ps260.OverlayValues[95] = d95
			ps260.OverlayValues[96] = d96
			ps260.OverlayValues[97] = d97
			ps260.OverlayValues[98] = d98
			ps260.OverlayValues[99] = d99
			ps260.OverlayValues[100] = d100
			ps260.OverlayValues[107] = d107
			ps260.OverlayValues[108] = d108
			ps260.OverlayValues[109] = d109
			ps260.OverlayValues[110] = d110
			ps260.OverlayValues[111] = d111
			ps260.OverlayValues[112] = d112
			ps260.OverlayValues[113] = d113
			ps260.OverlayValues[114] = d114
			ps260.OverlayValues[115] = d115
			ps260.OverlayValues[116] = d116
			ps260.OverlayValues[117] = d117
			ps260.OverlayValues[118] = d118
			ps260.OverlayValues[119] = d119
			ps260.OverlayValues[120] = d120
			ps260.OverlayValues[121] = d121
			ps260.OverlayValues[122] = d122
			ps260.OverlayValues[123] = d123
			ps260.OverlayValues[124] = d124
			ps260.OverlayValues[125] = d125
			ps260.OverlayValues[126] = d126
			ps260.OverlayValues[127] = d127
			ps260.OverlayValues[128] = d128
			ps260.OverlayValues[129] = d129
			ps260.OverlayValues[130] = d130
			ps260.OverlayValues[131] = d131
			ps260.OverlayValues[132] = d132
			ps260.OverlayValues[133] = d133
			ps260.OverlayValues[134] = d134
			ps260.OverlayValues[135] = d135
			ps260.OverlayValues[136] = d136
			ps260.OverlayValues[137] = d137
			ps260.OverlayValues[138] = d138
			ps260.OverlayValues[139] = d139
			ps260.OverlayValues[140] = d140
			ps260.OverlayValues[141] = d141
			ps260.OverlayValues[142] = d142
			ps260.OverlayValues[143] = d143
			ps260.OverlayValues[144] = d144
			ps260.OverlayValues[145] = d145
			ps260.OverlayValues[146] = d146
			ps260.OverlayValues[147] = d147
			ps260.OverlayValues[148] = d148
			ps260.OverlayValues[149] = d149
			ps260.OverlayValues[150] = d150
			ps260.OverlayValues[151] = d151
			ps260.OverlayValues[152] = d152
			ps260.OverlayValues[153] = d153
			ps260.OverlayValues[154] = d154
			ps260.OverlayValues[155] = d155
			ps260.OverlayValues[156] = d156
			ps260.OverlayValues[157] = d157
			ps260.OverlayValues[158] = d158
			ps260.OverlayValues[159] = d159
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
			ps260.OverlayValues[182] = d182
			ps260.OverlayValues[183] = d183
			ps260.OverlayValues[184] = d184
			ps260.OverlayValues[185] = d185
			ps260.OverlayValues[186] = d186
			ps260.OverlayValues[187] = d187
			ps260.OverlayValues[188] = d188
			ps260.OverlayValues[189] = d189
			ps260.OverlayValues[190] = d190
			ps260.OverlayValues[191] = d191
			ps260.OverlayValues[192] = d192
			ps260.OverlayValues[193] = d193
			ps260.OverlayValues[194] = d194
			ps260.OverlayValues[195] = d195
			ps260.OverlayValues[196] = d196
			ps260.OverlayValues[197] = d197
			ps260.OverlayValues[198] = d198
			ps260.OverlayValues[199] = d199
			ps260.OverlayValues[200] = d200
			ps260.OverlayValues[201] = d201
			ps260.OverlayValues[202] = d202
			ps260.OverlayValues[203] = d203
			ps260.OverlayValues[204] = d204
			ps260.OverlayValues[205] = d205
			ps260.OverlayValues[206] = d206
			ps260.OverlayValues[207] = d207
			ps260.OverlayValues[208] = d208
			ps260.OverlayValues[209] = d209
			ps260.OverlayValues[210] = d210
			ps260.OverlayValues[211] = d211
			ps260.OverlayValues[212] = d212
			ps260.OverlayValues[213] = d213
			ps260.OverlayValues[214] = d214
			ps260.OverlayValues[215] = d215
			ps260.OverlayValues[216] = d216
			ps260.OverlayValues[217] = d217
			ps260.OverlayValues[218] = d218
			ps260.OverlayValues[219] = d219
			ps260.OverlayValues[220] = d220
			ps260.OverlayValues[221] = d221
			ps260.OverlayValues[222] = d222
			ps260.OverlayValues[223] = d223
			ps260.OverlayValues[224] = d224
			ps260.OverlayValues[225] = d225
			ps260.OverlayValues[226] = d226
			ps260.OverlayValues[227] = d227
			ps260.OverlayValues[228] = d228
			ps260.OverlayValues[229] = d229
			ps260.OverlayValues[230] = d230
			ps260.OverlayValues[231] = d231
			ps260.OverlayValues[232] = d232
			ps260.OverlayValues[233] = d233
			ps260.OverlayValues[234] = d234
			ps260.OverlayValues[235] = d235
			ps260.OverlayValues[236] = d236
			ps260.OverlayValues[237] = d237
			ps260.OverlayValues[238] = d238
			ps260.OverlayValues[239] = d239
			ps260.OverlayValues[240] = d240
			ps260.OverlayValues[241] = d241
			ps260.OverlayValues[242] = d242
			ps260.OverlayValues[243] = d243
			ps260.OverlayValues[244] = d244
			ps260.OverlayValues[245] = d245
			ps260.OverlayValues[246] = d246
			ps260.OverlayValues[247] = d247
			ps260.OverlayValues[248] = d248
			ps260.OverlayValues[249] = d249
			ps260.OverlayValues[250] = d250
			ps260.OverlayValues[251] = d251
			ps260.OverlayValues[252] = d252
			ps260.OverlayValues[253] = d253
			ps260.OverlayValues[254] = d254
			ps260.OverlayValues[255] = d255
			ps260.OverlayValues[256] = d256
			ps260.OverlayValues[257] = d257
			ps261 := scm.PhiState{General: true}
			ps261.OverlayValues = make([]scm.JITValueDesc, 258)
			ps261.OverlayValues[0] = d0
			ps261.OverlayValues[1] = d1
			ps261.OverlayValues[7] = d7
			ps261.OverlayValues[8] = d8
			ps261.OverlayValues[9] = d9
			ps261.OverlayValues[10] = d10
			ps261.OverlayValues[11] = d11
			ps261.OverlayValues[12] = d12
			ps261.OverlayValues[13] = d13
			ps261.OverlayValues[14] = d14
			ps261.OverlayValues[15] = d15
			ps261.OverlayValues[16] = d16
			ps261.OverlayValues[17] = d17
			ps261.OverlayValues[18] = d18
			ps261.OverlayValues[19] = d19
			ps261.OverlayValues[20] = d20
			ps261.OverlayValues[21] = d21
			ps261.OverlayValues[22] = d22
			ps261.OverlayValues[23] = d23
			ps261.OverlayValues[24] = d24
			ps261.OverlayValues[25] = d25
			ps261.OverlayValues[26] = d26
			ps261.OverlayValues[27] = d27
			ps261.OverlayValues[28] = d28
			ps261.OverlayValues[29] = d29
			ps261.OverlayValues[30] = d30
			ps261.OverlayValues[31] = d31
			ps261.OverlayValues[32] = d32
			ps261.OverlayValues[33] = d33
			ps261.OverlayValues[34] = d34
			ps261.OverlayValues[35] = d35
			ps261.OverlayValues[36] = d36
			ps261.OverlayValues[37] = d37
			ps261.OverlayValues[38] = d38
			ps261.OverlayValues[39] = d39
			ps261.OverlayValues[40] = d40
			ps261.OverlayValues[41] = d41
			ps261.OverlayValues[42] = d42
			ps261.OverlayValues[43] = d43
			ps261.OverlayValues[44] = d44
			ps261.OverlayValues[45] = d45
			ps261.OverlayValues[46] = d46
			ps261.OverlayValues[47] = d47
			ps261.OverlayValues[48] = d48
			ps261.OverlayValues[49] = d49
			ps261.OverlayValues[50] = d50
			ps261.OverlayValues[57] = d57
			ps261.OverlayValues[58] = d58
			ps261.OverlayValues[59] = d59
			ps261.OverlayValues[60] = d60
			ps261.OverlayValues[61] = d61
			ps261.OverlayValues[62] = d62
			ps261.OverlayValues[63] = d63
			ps261.OverlayValues[64] = d64
			ps261.OverlayValues[65] = d65
			ps261.OverlayValues[66] = d66
			ps261.OverlayValues[67] = d67
			ps261.OverlayValues[68] = d68
			ps261.OverlayValues[69] = d69
			ps261.OverlayValues[70] = d70
			ps261.OverlayValues[71] = d71
			ps261.OverlayValues[72] = d72
			ps261.OverlayValues[73] = d73
			ps261.OverlayValues[74] = d74
			ps261.OverlayValues[75] = d75
			ps261.OverlayValues[76] = d76
			ps261.OverlayValues[77] = d77
			ps261.OverlayValues[78] = d78
			ps261.OverlayValues[79] = d79
			ps261.OverlayValues[80] = d80
			ps261.OverlayValues[81] = d81
			ps261.OverlayValues[82] = d82
			ps261.OverlayValues[83] = d83
			ps261.OverlayValues[84] = d84
			ps261.OverlayValues[85] = d85
			ps261.OverlayValues[86] = d86
			ps261.OverlayValues[87] = d87
			ps261.OverlayValues[88] = d88
			ps261.OverlayValues[89] = d89
			ps261.OverlayValues[90] = d90
			ps261.OverlayValues[91] = d91
			ps261.OverlayValues[92] = d92
			ps261.OverlayValues[93] = d93
			ps261.OverlayValues[94] = d94
			ps261.OverlayValues[95] = d95
			ps261.OverlayValues[96] = d96
			ps261.OverlayValues[97] = d97
			ps261.OverlayValues[98] = d98
			ps261.OverlayValues[99] = d99
			ps261.OverlayValues[100] = d100
			ps261.OverlayValues[107] = d107
			ps261.OverlayValues[108] = d108
			ps261.OverlayValues[109] = d109
			ps261.OverlayValues[110] = d110
			ps261.OverlayValues[111] = d111
			ps261.OverlayValues[112] = d112
			ps261.OverlayValues[113] = d113
			ps261.OverlayValues[114] = d114
			ps261.OverlayValues[115] = d115
			ps261.OverlayValues[116] = d116
			ps261.OverlayValues[117] = d117
			ps261.OverlayValues[118] = d118
			ps261.OverlayValues[119] = d119
			ps261.OverlayValues[120] = d120
			ps261.OverlayValues[121] = d121
			ps261.OverlayValues[122] = d122
			ps261.OverlayValues[123] = d123
			ps261.OverlayValues[124] = d124
			ps261.OverlayValues[125] = d125
			ps261.OverlayValues[126] = d126
			ps261.OverlayValues[127] = d127
			ps261.OverlayValues[128] = d128
			ps261.OverlayValues[129] = d129
			ps261.OverlayValues[130] = d130
			ps261.OverlayValues[131] = d131
			ps261.OverlayValues[132] = d132
			ps261.OverlayValues[133] = d133
			ps261.OverlayValues[134] = d134
			ps261.OverlayValues[135] = d135
			ps261.OverlayValues[136] = d136
			ps261.OverlayValues[137] = d137
			ps261.OverlayValues[138] = d138
			ps261.OverlayValues[139] = d139
			ps261.OverlayValues[140] = d140
			ps261.OverlayValues[141] = d141
			ps261.OverlayValues[142] = d142
			ps261.OverlayValues[143] = d143
			ps261.OverlayValues[144] = d144
			ps261.OverlayValues[145] = d145
			ps261.OverlayValues[146] = d146
			ps261.OverlayValues[147] = d147
			ps261.OverlayValues[148] = d148
			ps261.OverlayValues[149] = d149
			ps261.OverlayValues[150] = d150
			ps261.OverlayValues[151] = d151
			ps261.OverlayValues[152] = d152
			ps261.OverlayValues[153] = d153
			ps261.OverlayValues[154] = d154
			ps261.OverlayValues[155] = d155
			ps261.OverlayValues[156] = d156
			ps261.OverlayValues[157] = d157
			ps261.OverlayValues[158] = d158
			ps261.OverlayValues[159] = d159
			ps261.OverlayValues[165] = d165
			ps261.OverlayValues[166] = d166
			ps261.OverlayValues[167] = d167
			ps261.OverlayValues[168] = d168
			ps261.OverlayValues[169] = d169
			ps261.OverlayValues[170] = d170
			ps261.OverlayValues[171] = d171
			ps261.OverlayValues[172] = d172
			ps261.OverlayValues[173] = d173
			ps261.OverlayValues[174] = d174
			ps261.OverlayValues[175] = d175
			ps261.OverlayValues[176] = d176
			ps261.OverlayValues[177] = d177
			ps261.OverlayValues[178] = d178
			ps261.OverlayValues[179] = d179
			ps261.OverlayValues[180] = d180
			ps261.OverlayValues[181] = d181
			ps261.OverlayValues[182] = d182
			ps261.OverlayValues[183] = d183
			ps261.OverlayValues[184] = d184
			ps261.OverlayValues[185] = d185
			ps261.OverlayValues[186] = d186
			ps261.OverlayValues[187] = d187
			ps261.OverlayValues[188] = d188
			ps261.OverlayValues[189] = d189
			ps261.OverlayValues[190] = d190
			ps261.OverlayValues[191] = d191
			ps261.OverlayValues[192] = d192
			ps261.OverlayValues[193] = d193
			ps261.OverlayValues[194] = d194
			ps261.OverlayValues[195] = d195
			ps261.OverlayValues[196] = d196
			ps261.OverlayValues[197] = d197
			ps261.OverlayValues[198] = d198
			ps261.OverlayValues[199] = d199
			ps261.OverlayValues[200] = d200
			ps261.OverlayValues[201] = d201
			ps261.OverlayValues[202] = d202
			ps261.OverlayValues[203] = d203
			ps261.OverlayValues[204] = d204
			ps261.OverlayValues[205] = d205
			ps261.OverlayValues[206] = d206
			ps261.OverlayValues[207] = d207
			ps261.OverlayValues[208] = d208
			ps261.OverlayValues[209] = d209
			ps261.OverlayValues[210] = d210
			ps261.OverlayValues[211] = d211
			ps261.OverlayValues[212] = d212
			ps261.OverlayValues[213] = d213
			ps261.OverlayValues[214] = d214
			ps261.OverlayValues[215] = d215
			ps261.OverlayValues[216] = d216
			ps261.OverlayValues[217] = d217
			ps261.OverlayValues[218] = d218
			ps261.OverlayValues[219] = d219
			ps261.OverlayValues[220] = d220
			ps261.OverlayValues[221] = d221
			ps261.OverlayValues[222] = d222
			ps261.OverlayValues[223] = d223
			ps261.OverlayValues[224] = d224
			ps261.OverlayValues[225] = d225
			ps261.OverlayValues[226] = d226
			ps261.OverlayValues[227] = d227
			ps261.OverlayValues[228] = d228
			ps261.OverlayValues[229] = d229
			ps261.OverlayValues[230] = d230
			ps261.OverlayValues[231] = d231
			ps261.OverlayValues[232] = d232
			ps261.OverlayValues[233] = d233
			ps261.OverlayValues[234] = d234
			ps261.OverlayValues[235] = d235
			ps261.OverlayValues[236] = d236
			ps261.OverlayValues[237] = d237
			ps261.OverlayValues[238] = d238
			ps261.OverlayValues[239] = d239
			ps261.OverlayValues[240] = d240
			ps261.OverlayValues[241] = d241
			ps261.OverlayValues[242] = d242
			ps261.OverlayValues[243] = d243
			ps261.OverlayValues[244] = d244
			ps261.OverlayValues[245] = d245
			ps261.OverlayValues[246] = d246
			ps261.OverlayValues[247] = d247
			ps261.OverlayValues[248] = d248
			ps261.OverlayValues[249] = d249
			ps261.OverlayValues[250] = d250
			ps261.OverlayValues[251] = d251
			ps261.OverlayValues[252] = d252
			ps261.OverlayValues[253] = d253
			ps261.OverlayValues[254] = d254
			ps261.OverlayValues[255] = d255
			ps261.OverlayValues[256] = d256
			ps261.OverlayValues[257] = d257
			alloc262 := ctx.SnapshotAllocState()
			if !bbs[7].Rendered {
				bbs[7].RenderPS(ps261)
			}
			ctx.RestoreAllocState(alloc262)
			if !bbs[6].Rendered {
				return bbs[6].RenderPS(ps260)
			}
			return result
			ctx.FreeDesc(&d256)
			return result
			}
			ps263 := scm.PhiState{General: false}
			_ = bbs[0].RenderPS(ps263)
			ctx.W.MarkLabel(lbl0)
			d264 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d264)
			ctx.BindReg(r1, &d264)
			ctx.EmitMovPairToResult(&d264, &result)
			ctx.FreeReg(r0)
			ctx.FreeReg(r1)
			ctx.W.ResolveFixups()
			if idxPinned { ctx.UnprotectReg(idxPinnedReg) }
			ctx.W.PatchInt32(r5, int32(40))
			ctx.W.EmitAddRSP32(int32(40))
			return result
}

func (s *StorageString) prepare() {
	// set up scan
	s.starts.prepare()
	s.lens.prepare()
	s.values.prepare()
	s.reverseMap = make(map[string][3]uint)
	s.prefixstat = make(map[string]int)
}
func (s *StorageString) scan(i uint32, value scm.Scmer) {
	// storage is so simple, dont need scan
	var v string
	if value.IsNil() {
		if s.nodict {
			s.starts.scan(i, scm.NewNil())
		} else {
			s.values.scan(i, scm.NewNil())
		}
		return
	}
	v = scm.String(value)

	// check if we have common prefix (but ignore duplicates because they are compressed by dictionary)
	if s.laststr != v {
		commonlen := 0
		for commonlen < len(s.laststr) && commonlen < len(v) && s.laststr[commonlen] == v[commonlen] {
			s.prefixstat[v[0:commonlen]] = s.prefixstat[v[0:commonlen]] + 1
			commonlen++
		}
		if v != "" {
			s.laststr = v
		}
	}

	// check for dictionary
	if i == 100 && len(s.reverseMap) > 99 {
		// nearly no repetition in the first 100 items: save the time to build reversemap
		s.nodict = true
		s.reverseMap = nil
		s.sb.Reset()
		if s.values.hasNull {
			s.starts.scan(0, scm.NewNil()) // learn NULL
		}
		// build will fill our stringbuffer
	}
	s.allsize = s.allsize + len(v)
	if s.nodict {
		s.starts.scan(i, scm.NewInt(int64(s.allsize)))
		s.lens.scan(i, scm.NewInt(int64(len(v))))
	} else {
		start, ok := s.reverseMap[v]
		if ok {
			// reuse of string
		} else {
			// learn
			start[0] = s.count
			start[1] = uint(s.sb.Len())
			start[2] = uint(len(v))
			s.sb.WriteString(v)
			s.starts.scan(uint32(start[0]), scm.NewInt(int64(start[1])))
			s.lens.scan(uint32(start[0]), scm.NewInt(int64(start[2])))
			s.reverseMap[v] = start
			s.count = s.count + 1
		}
		s.values.scan(i, scm.NewInt(int64(start[0])))
	}
}
func (s *StorageString) init(i uint32) {
	s.prefixstat = nil // free memory
	if s.nodict {
		// do not init values, sb andsoon
		s.starts.init(i)
		s.lens.init(i)
	} else {
		// allocate
		s.dictionary = s.sb.String() // extract one big slice with all strings (no extra memory structure)
		s.sb.Reset()                 // free the memory
		// prefixed strings are not accounted with that, but maybe this could be checked later??
		s.values.init(i)
		// take over dictionary
		s.starts.init(uint32(s.count))
		s.lens.init(uint32(s.count))
		for _, start := range s.reverseMap {
			// we read the value from dictionary, so we can free up all the single-strings
			s.starts.build(uint32(start[0]), scm.NewInt(int64(start[1])))
			s.lens.build(uint32(start[0]), scm.NewInt(int64(start[2])))
		}
	}
}
func (s *StorageString) build(i uint32, value scm.Scmer) {
	// store
	if value.IsNil() {
		if s.nodict {
			s.starts.build(i, scm.NewNil())
		} else {
			s.values.build(i, scm.NewNil())
		}
		return
	}
	v := scm.String(value)
	if s.nodict {
		s.starts.build(i, scm.NewInt(int64(s.sb.Len())))
		s.lens.build(i, scm.NewInt(int64(len(v))))
		s.sb.WriteString(v)
	} else {
		start := s.reverseMap[v]
		// write start+end into sub storage maps
		s.values.build(i, scm.NewInt(int64(start[0])))
	}
}
func (s *StorageString) finish() {
	if s.nodict {
		s.dictionary = s.sb.String()
		s.sb.Reset()
	} else {
		s.reverseMap = nil
		s.values.finish()
	}
	s.starts.finish()
	s.lens.finish()
}
func (s *StorageString) proposeCompression(i uint32) ColumnStorage {
	// build prefix map (maybe prefix trees later?)
	/* TODO: reactivate as soon as StoragePrefix has a proper implementation for Serialize/Deserialize
	mostprefixscore := 0
	mostprefix := ""
	for k, v := range s.prefixstat {
		if len(k) * v > mostprefixscore {
			mostprefix = k
			mostprefixscore = len(k) * v // cost saving of prefix = len(prefix) * occurance
		}
	}
	if uint(mostprefixscore) > i / 8 + 100 {
		// built a 1-bit prefix (TODO: maybe later more)
		stor := new(StoragePrefix)
		stor.prefixdictionary = []string{"", mostprefix}
		return stor
	}

	Prefix tree index:
	rootnodes = []
	for each s := range string {
		foreach k, v := rootnodes {
			pfx := commonPrefix(s, k)
			if pfx == k {
				// insert into subtree
				v.insert(s[len(pfx):], value)
			} else {
				// split the tree
				delete(rootnodes, k)
				rootnodes[pfx] = {k[len(pfx):]: v, s[len(pfx):]: value}
			}
		}
		rootnodes[s] = value
		cont:
	}
	implementation: byte stream of id, len, byte[len] + array:id->*treenode; encode bigger ids similar to utf-8: for { result = result < 7 | (byte & 127) if byte & 128 == 0 {break}}

	prefix compression: multi-stage storage
	type prefixTree struct { text string, children []prefixTree }
	type prefixTreeStorage struct { childIndexes ColumnStorage, recordIdTranslation ColumnStorage, children []prefixTreeStorage } -> Seq-compression should be very effective

	*/
	// dont't propose another pass
	return nil
}
