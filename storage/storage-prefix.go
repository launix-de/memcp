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

import "fmt"
import "strings"
import "github.com/launix-de/memcp/scm"
import "unsafe"

type StoragePrefix struct {
	// prefix compression
	prefixes         StorageInt
	prefixdictionary []string      // pref
	values           StorageString // only one depth (but can be cascaded!)
}

func (s *StoragePrefix) ComputeSize() uint {
	return s.prefixes.ComputeSize() + 24 + s.values.ComputeSize()
}

func (s *StoragePrefix) String() string {
	return fmt.Sprintf("prefix[%s]-%s", s.prefixdictionary[1], s.values.String())
}

func (s *StoragePrefix) GetCachedReader() ColumnReader { return s }

func (s *StoragePrefix) GetValue(i uint32) scm.Scmer {
	inner := s.values.GetValue(i)
	if inner.IsNil() {
		return scm.NewNil()
	}
	if !inner.IsString() {
		panic("invalid value in prefix storage")
	}
	idx := int64(s.prefixes.GetValueUInt(i)) + s.prefixes.offset
	if idx >= int64(len(s.prefixdictionary)) || idx < 0 {
		panic("prefix index out of range")
	}
	prefix := s.prefixdictionary[idx]
	return scm.NewString(prefix + inner.String())
}
func (s *StoragePrefix) JITEmit(ctx *scm.JITContext, thisptr scm.JITValueDesc, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
			var d0 scm.JITValueDesc
			_ = d0
			var d1 scm.JITValueDesc
			_ = d1
			var d2 scm.JITValueDesc
			_ = d2
			var d3 scm.JITValueDesc
			_ = d3
			var r7 unsafe.Pointer
			_ = r7
			var d4 scm.JITValueDesc
			_ = d4
			var d5 scm.JITValueDesc
			_ = d5
			var d6 scm.JITValueDesc
			_ = d6
			var d7 scm.JITValueDesc
			_ = d7
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
			var d101 scm.JITValueDesc
			_ = d101
			var d102 scm.JITValueDesc
			_ = d102
			var d103 scm.JITValueDesc
			_ = d103
			var d104 scm.JITValueDesc
			_ = d104
			var d105 scm.JITValueDesc
			_ = d105
			var d106 scm.JITValueDesc
			_ = d106
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
			var d246 scm.JITValueDesc
			_ = d246
			var d247 scm.JITValueDesc
			_ = d247
			var d248 scm.JITValueDesc
			_ = d248
			var d249 scm.JITValueDesc
			_ = d249
			var d255 scm.JITValueDesc
			_ = d255
			var d256 scm.JITValueDesc
			_ = d256
			var d257 scm.JITValueDesc
			_ = d257
			var d258 scm.JITValueDesc
			_ = d258
			var d259 scm.JITValueDesc
			_ = d259
			var d260 scm.JITValueDesc
			_ = d260
			var d261 scm.JITValueDesc
			_ = d261
			var d262 scm.JITValueDesc
			_ = d262
			var d263 scm.JITValueDesc
			_ = d263
			var d264 scm.JITValueDesc
			_ = d264
			var d265 scm.JITValueDesc
			_ = d265
			var d266 scm.JITValueDesc
			_ = d266
			var d267 scm.JITValueDesc
			_ = d267
			var d268 scm.JITValueDesc
			_ = d268
			var d269 scm.JITValueDesc
			_ = d269
			var d270 scm.JITValueDesc
			_ = d270
			var d271 scm.JITValueDesc
			_ = d271
			var d272 scm.JITValueDesc
			_ = d272
			var d273 scm.JITValueDesc
			_ = d273
			var d274 scm.JITValueDesc
			_ = d274
			var d275 scm.JITValueDesc
			_ = d275
			var d276 scm.JITValueDesc
			_ = d276
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
			var d291 scm.JITValueDesc
			_ = d291
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
			var d306 scm.JITValueDesc
			_ = d306
			var d307 scm.JITValueDesc
			_ = d307
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
			var d313 scm.JITValueDesc
			_ = d313
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
			var bbs [8]scm.BBDescriptor
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
			ctx.EnsureDesc(&idxInt)
			d0 = idxInt
			_ = d0
			r2 := idxInt.Loc == scm.LocReg
			r3 := idxInt.Reg
			if r2 { ctx.ProtectReg(r3) }
			lbl9 := ctx.W.ReserveLabel()
			bbpos_1_0 := int32(-1)
			_ = bbpos_1_0
			bbpos_1_1 := int32(-1)
			_ = bbpos_1_1
			bbpos_1_2 := int32(-1)
			_ = bbpos_1_2
			bbpos_1_3 := int32(-1)
			_ = bbpos_1_3
			bbpos_1_4 := int32(-1)
			_ = bbpos_1_4
			bbpos_1_5 := int32(-1)
			_ = bbpos_1_5
			bbpos_1_6 := int32(-1)
			_ = bbpos_1_6
			bbpos_1_7 := int32(-1)
			_ = bbpos_1_7
			bbpos_1_8 := int32(-1)
			_ = bbpos_1_8
			bbpos_1_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			var d1 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 232
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d1 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 232)
				r4 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r4, thisptr.Reg, off)
				d1 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r4}
				ctx.BindReg(r4, &d1)
			}
			d2 = d1
			ctx.EnsureDesc(&d2)
			if d2.Loc != scm.LocImm && d2.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl10 := ctx.W.ReserveLabel()
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			if d2.Loc == scm.LocImm {
				if d2.Imm.Bool() {
					ctx.W.MarkLabel(lbl12)
					ctx.W.EmitJmp(lbl10)
				} else {
					ctx.W.MarkLabel(lbl13)
					ctx.W.EmitJmp(lbl11)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d2.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl12)
				ctx.W.EmitJmp(lbl13)
				ctx.W.MarkLabel(lbl12)
				ctx.W.EmitJmp(lbl10)
				ctx.W.MarkLabel(lbl13)
				ctx.W.EmitJmp(lbl11)
			}
			ctx.FreeDesc(&d1)
			bbpos_1_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl11)
			ctx.W.ResolveFixups()
			ctx.EnsureDesc(&d0)
			d3 = d0
			_ = d3
			r5 := d0.Loc == scm.LocReg
			r6 := d0.Reg
			if r5 { ctx.ProtectReg(r6) }
			r7 = ctx.W.EmitSubRSP32Fixup()
			_ = r7
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			lbl14 := ctx.W.ReserveLabel()
			bbpos_2_0 := int32(-1)
			_ = bbpos_2_0
			bbpos_2_1 := int32(-1)
			_ = bbpos_2_1
			bbpos_2_2 := int32(-1)
			_ = bbpos_2_2
			bbpos_2_3 := int32(-1)
			_ = bbpos_2_3
			bbpos_2_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d3)
			var d5 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d3.Imm.Int()))))}
			} else {
				r8 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r8, d3.Reg)
				ctx.W.EmitShlRegImm8(r8, 32)
				ctx.W.EmitShrRegImm8(r8, 32)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
				ctx.BindReg(r8, &d5)
			}
			var d6 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24)
				r9 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r9, thisptr.Reg, off)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r9}
				ctx.BindReg(r9, &d6)
			}
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d6)
			var d7 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d7 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d6.Imm.Int()))))}
			} else {
				r10 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r10, d6.Reg)
				ctx.W.EmitShlRegImm8(r10, 56)
				ctx.W.EmitShrRegImm8(r10, 56)
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
				ctx.BindReg(r10, &d7)
			}
			ctx.FreeDesc(&d6)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d7)
			var d8 scm.JITValueDesc
			if d5.Loc == scm.LocImm && d7.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() * d7.Imm.Int())}
			} else if d5.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d7.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d5.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d7.Reg)
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d8)
			} else if d7.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d5.Reg)
				ctx.W.EmitMovRegReg(scratch, d5.Reg)
				if d7.Imm.Int() >= -2147483648 && d7.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d7.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d7.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d8)
			} else {
				r11 := ctx.AllocRegExcept(d5.Reg, d7.Reg)
				ctx.W.EmitMovRegReg(r11, d5.Reg)
				ctx.W.EmitImulInt64(r11, d7.Reg)
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
				ctx.BindReg(r11, &d8)
			}
			if d8.Loc == scm.LocReg && d5.Loc == scm.LocReg && d8.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d5)
			ctx.FreeDesc(&d7)
			var d9 scm.JITValueDesc
			r12 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r12, uint64(dataPtr))
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r12, StackOff: int32(sliceLen)}
				ctx.BindReg(r12, &d9)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 0)
				ctx.W.EmitMovRegMem(r12, thisptr.Reg, off)
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r12}
				ctx.BindReg(r12, &d9)
			}
			ctx.BindReg(r12, &d9)
			ctx.EnsureDesc(&d8)
			var d10 scm.JITValueDesc
			if d8.Loc == scm.LocImm {
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() / 64)}
			} else {
				r13 := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitMovRegReg(r13, d8.Reg)
				ctx.W.EmitShrRegImm8(r13, 6)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
				ctx.BindReg(r13, &d10)
			}
			if d10.Loc == scm.LocReg && d8.Loc == scm.LocReg && d10.Reg == d8.Reg {
				ctx.TransferReg(d8.Reg)
				d8.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d10)
			r14 := ctx.AllocReg()
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d9)
			if d10.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r14, uint64(d10.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r14, d10.Reg)
				ctx.W.EmitShlRegImm8(r14, 3)
			}
			if d9.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d9.Imm.Int()))
				ctx.W.EmitAddInt64(r14, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r14, d9.Reg)
			}
			r15 := ctx.AllocRegExcept(r14)
			ctx.W.EmitMovRegMem(r15, r14, 0)
			ctx.FreeReg(r14)
			d11 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r15}
			ctx.BindReg(r15, &d11)
			ctx.FreeDesc(&d10)
			ctx.EnsureDesc(&d8)
			var d12 scm.JITValueDesc
			if d8.Loc == scm.LocImm {
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() % 64)}
			} else {
				r16 := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitMovRegReg(r16, d8.Reg)
				ctx.W.EmitAndRegImm32(r16, 63)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
				ctx.BindReg(r16, &d12)
			}
			if d12.Loc == scm.LocReg && d8.Loc == scm.LocReg && d12.Reg == d8.Reg {
				ctx.TransferReg(d8.Reg)
				d8.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d12)
			var d13 scm.JITValueDesc
			if d11.Loc == scm.LocImm && d12.Loc == scm.LocImm {
				d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d11.Imm.Int()) << uint64(d12.Imm.Int())))}
			} else if d12.Loc == scm.LocImm {
				r17 := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitMovRegReg(r17, d11.Reg)
				ctx.W.EmitShlRegImm8(r17, uint8(d12.Imm.Int()))
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
				ctx.BindReg(r17, &d13)
			} else {
				{
					shiftSrc := d11.Reg
					r18 := ctx.AllocRegExcept(d11.Reg)
					ctx.W.EmitMovRegReg(r18, d11.Reg)
					shiftSrc = r18
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d12.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d12.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d12.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d13)
				}
			}
			if d13.Loc == scm.LocReg && d11.Loc == scm.LocReg && d13.Reg == d11.Reg {
				ctx.TransferReg(d11.Reg)
				d11.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d11)
			ctx.FreeDesc(&d12)
			var d14 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 25)
				r19 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r19, thisptr.Reg, off)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r19}
				ctx.BindReg(r19, &d14)
			}
			d15 = d14
			ctx.EnsureDesc(&d15)
			if d15.Loc != scm.LocImm && d15.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			if d15.Loc == scm.LocImm {
				if d15.Imm.Bool() {
					ctx.W.MarkLabel(lbl17)
					ctx.W.EmitJmp(lbl15)
				} else {
					ctx.W.MarkLabel(lbl18)
			d16 = d13
			if d16.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d16)
			ctx.EmitStoreToStack(d16, 0)
					ctx.W.EmitJmp(lbl16)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d15.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl17)
				ctx.W.EmitJmp(lbl18)
				ctx.W.MarkLabel(lbl17)
				ctx.W.EmitJmp(lbl15)
				ctx.W.MarkLabel(lbl18)
			d17 = d13
			if d17.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d17)
			ctx.EmitStoreToStack(d17, 0)
				ctx.W.EmitJmp(lbl16)
			}
			ctx.FreeDesc(&d14)
			bbpos_2_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl16)
			ctx.W.ResolveFixups()
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			var d18 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24)
				r20 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r20, thisptr.Reg, off)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r20}
				ctx.BindReg(r20, &d18)
			}
			ctx.EnsureDesc(&d18)
			ctx.EnsureDesc(&d18)
			var d19 scm.JITValueDesc
			if d18.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d18.Imm.Int()))))}
			} else {
				r21 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r21, d18.Reg)
				ctx.W.EmitShlRegImm8(r21, 56)
				ctx.W.EmitShrRegImm8(r21, 56)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d19)
			}
			ctx.FreeDesc(&d18)
			d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d19)
			var d21 scm.JITValueDesc
			if d20.Loc == scm.LocImm && d19.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d20.Imm.Int() - d19.Imm.Int())}
			} else if d19.Loc == scm.LocImm && d19.Imm.Int() == 0 {
				r22 := ctx.AllocRegExcept(d20.Reg)
				ctx.W.EmitMovRegReg(r22, d20.Reg)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d21)
			} else if d20.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d19.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d20.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d19.Reg)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d21)
			} else if d19.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d20.Reg)
				ctx.W.EmitMovRegReg(scratch, d20.Reg)
				if d19.Imm.Int() >= -2147483648 && d19.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d19.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d19.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d21)
			} else {
				r23 := ctx.AllocRegExcept(d20.Reg, d19.Reg)
				ctx.W.EmitMovRegReg(r23, d20.Reg)
				ctx.W.EmitSubInt64(r23, d19.Reg)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
				ctx.BindReg(r23, &d21)
			}
			if d21.Loc == scm.LocReg && d20.Loc == scm.LocReg && d21.Reg == d20.Reg {
				ctx.TransferReg(d20.Reg)
				d20.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d19)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d21)
			var d22 scm.JITValueDesc
			if d4.Loc == scm.LocImm && d21.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d4.Imm.Int()) >> uint64(d21.Imm.Int())))}
			} else if d21.Loc == scm.LocImm {
				r24 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r24, d4.Reg)
				ctx.W.EmitShrRegImm8(r24, uint8(d21.Imm.Int()))
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
				ctx.BindReg(r24, &d22)
			} else {
				{
					shiftSrc := d4.Reg
					r25 := ctx.AllocRegExcept(d4.Reg)
					ctx.W.EmitMovRegReg(r25, d4.Reg)
					shiftSrc = r25
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d21.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d21.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d21.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d22)
				}
			}
			if d22.Loc == scm.LocReg && d4.Loc == scm.LocReg && d22.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d4)
			ctx.FreeDesc(&d21)
			r26 := ctx.AllocReg()
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d22)
			if d22.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r26, d22)
			}
			ctx.W.EmitJmp(lbl14)
			bbpos_2_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl15)
			ctx.W.ResolveFixups()
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d8)
			var d23 scm.JITValueDesc
			if d8.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() % 64)}
			} else {
				r27 := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitMovRegReg(r27, d8.Reg)
				ctx.W.EmitAndRegImm32(r27, 63)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d23)
			}
			if d23.Loc == scm.LocReg && d8.Loc == scm.LocReg && d23.Reg == d8.Reg {
				ctx.TransferReg(d8.Reg)
				d8.Loc = scm.LocNone
			}
			var d24 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24)
				r28 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r28, thisptr.Reg, off)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r28}
				ctx.BindReg(r28, &d24)
			}
			ctx.EnsureDesc(&d24)
			ctx.EnsureDesc(&d24)
			var d25 scm.JITValueDesc
			if d24.Loc == scm.LocImm {
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d24.Imm.Int()))))}
			} else {
				r29 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r29, d24.Reg)
				ctx.W.EmitShlRegImm8(r29, 56)
				ctx.W.EmitShrRegImm8(r29, 56)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d25)
			}
			ctx.FreeDesc(&d24)
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d25)
			var d26 scm.JITValueDesc
			if d23.Loc == scm.LocImm && d25.Loc == scm.LocImm {
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d23.Imm.Int() + d25.Imm.Int())}
			} else if d25.Loc == scm.LocImm && d25.Imm.Int() == 0 {
				r30 := ctx.AllocRegExcept(d23.Reg)
				ctx.W.EmitMovRegReg(r30, d23.Reg)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
				ctx.BindReg(r30, &d26)
			} else if d23.Loc == scm.LocImm && d23.Imm.Int() == 0 {
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d25.Reg}
				ctx.BindReg(d25.Reg, &d26)
			} else if d23.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d25.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d23.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d25.Reg)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d26)
			} else if d25.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d23.Reg)
				ctx.W.EmitMovRegReg(scratch, d23.Reg)
				if d25.Imm.Int() >= -2147483648 && d25.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d25.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d25.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d26)
			} else {
				r31 := ctx.AllocRegExcept(d23.Reg, d25.Reg)
				ctx.W.EmitMovRegReg(r31, d23.Reg)
				ctx.W.EmitAddInt64(r31, d25.Reg)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
				ctx.BindReg(r31, &d26)
			}
			if d26.Loc == scm.LocReg && d23.Loc == scm.LocReg && d26.Reg == d23.Reg {
				ctx.TransferReg(d23.Reg)
				d23.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d23)
			ctx.FreeDesc(&d25)
			ctx.EnsureDesc(&d26)
			var d27 scm.JITValueDesc
			if d26.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d26.Imm.Int()) > uint64(64))}
			} else {
				r32 := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitCmpRegImm32(d26.Reg, 64)
				ctx.W.EmitSetcc(r32, scm.CcA)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r32}
				ctx.BindReg(r32, &d27)
			}
			ctx.FreeDesc(&d26)
			d28 = d27
			ctx.EnsureDesc(&d28)
			if d28.Loc != scm.LocImm && d28.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl19 := ctx.W.ReserveLabel()
			lbl20 := ctx.W.ReserveLabel()
			lbl21 := ctx.W.ReserveLabel()
			if d28.Loc == scm.LocImm {
				if d28.Imm.Bool() {
					ctx.W.MarkLabel(lbl20)
					ctx.W.EmitJmp(lbl19)
				} else {
					ctx.W.MarkLabel(lbl21)
			d29 = d13
			if d29.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d29)
			ctx.EmitStoreToStack(d29, 0)
					ctx.W.EmitJmp(lbl16)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d28.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl20)
				ctx.W.EmitJmp(lbl21)
				ctx.W.MarkLabel(lbl20)
				ctx.W.EmitJmp(lbl19)
				ctx.W.MarkLabel(lbl21)
			d30 = d13
			if d30.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d30)
			ctx.EmitStoreToStack(d30, 0)
				ctx.W.EmitJmp(lbl16)
			}
			ctx.FreeDesc(&d27)
			bbpos_2_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl19)
			ctx.W.ResolveFixups()
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d8)
			var d31 scm.JITValueDesc
			if d8.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() / 64)}
			} else {
				r33 := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitMovRegReg(r33, d8.Reg)
				ctx.W.EmitShrRegImm8(r33, 6)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
				ctx.BindReg(r33, &d31)
			}
			if d31.Loc == scm.LocReg && d8.Loc == scm.LocReg && d31.Reg == d8.Reg {
				ctx.TransferReg(d8.Reg)
				d8.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d31)
			var d32 scm.JITValueDesc
			if d31.Loc == scm.LocImm {
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d31.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d31.Reg)
				ctx.W.EmitMovRegReg(scratch, d31.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d32)
			}
			if d32.Loc == scm.LocReg && d31.Loc == scm.LocReg && d32.Reg == d31.Reg {
				ctx.TransferReg(d31.Reg)
				d31.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d31)
			ctx.EnsureDesc(&d32)
			r34 := ctx.AllocReg()
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d9)
			if d32.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r34, uint64(d32.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r34, d32.Reg)
				ctx.W.EmitShlRegImm8(r34, 3)
			}
			if d9.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d9.Imm.Int()))
				ctx.W.EmitAddInt64(r34, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r34, d9.Reg)
			}
			r35 := ctx.AllocRegExcept(r34)
			ctx.W.EmitMovRegMem(r35, r34, 0)
			ctx.FreeReg(r34)
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r35}
			ctx.BindReg(r35, &d33)
			ctx.FreeDesc(&d32)
			ctx.EnsureDesc(&d8)
			var d34 scm.JITValueDesc
			if d8.Loc == scm.LocImm {
				d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() % 64)}
			} else {
				r36 := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitMovRegReg(r36, d8.Reg)
				ctx.W.EmitAndRegImm32(r36, 63)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d34)
			}
			if d34.Loc == scm.LocReg && d8.Loc == scm.LocReg && d34.Reg == d8.Reg {
				ctx.TransferReg(d8.Reg)
				d8.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d8)
			d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d34)
			ctx.EnsureDesc(&d35)
			ctx.EnsureDesc(&d34)
			ctx.EnsureDesc(&d35)
			ctx.EnsureDesc(&d34)
			var d36 scm.JITValueDesc
			if d35.Loc == scm.LocImm && d34.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d35.Imm.Int() - d34.Imm.Int())}
			} else if d34.Loc == scm.LocImm && d34.Imm.Int() == 0 {
				r37 := ctx.AllocRegExcept(d35.Reg)
				ctx.W.EmitMovRegReg(r37, d35.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d36)
			} else if d35.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d34.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d35.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d34.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d36)
			} else if d34.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d35.Reg)
				ctx.W.EmitMovRegReg(scratch, d35.Reg)
				if d34.Imm.Int() >= -2147483648 && d34.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d34.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d34.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d36)
			} else {
				r38 := ctx.AllocRegExcept(d35.Reg, d34.Reg)
				ctx.W.EmitMovRegReg(r38, d35.Reg)
				ctx.W.EmitSubInt64(r38, d34.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
				ctx.BindReg(r38, &d36)
			}
			if d36.Loc == scm.LocReg && d35.Loc == scm.LocReg && d36.Reg == d35.Reg {
				ctx.TransferReg(d35.Reg)
				d35.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d34)
			ctx.EnsureDesc(&d33)
			ctx.EnsureDesc(&d36)
			var d37 scm.JITValueDesc
			if d33.Loc == scm.LocImm && d36.Loc == scm.LocImm {
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d33.Imm.Int()) >> uint64(d36.Imm.Int())))}
			} else if d36.Loc == scm.LocImm {
				r39 := ctx.AllocRegExcept(d33.Reg)
				ctx.W.EmitMovRegReg(r39, d33.Reg)
				ctx.W.EmitShrRegImm8(r39, uint8(d36.Imm.Int()))
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d37)
			} else {
				{
					shiftSrc := d33.Reg
					r40 := ctx.AllocRegExcept(d33.Reg)
					ctx.W.EmitMovRegReg(r40, d33.Reg)
					shiftSrc = r40
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d36.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d36.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d36.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d37)
				}
			}
			if d37.Loc == scm.LocReg && d33.Loc == scm.LocReg && d37.Reg == d33.Reg {
				ctx.TransferReg(d33.Reg)
				d33.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d33)
			ctx.FreeDesc(&d36)
			ctx.EnsureDesc(&d13)
			ctx.EnsureDesc(&d37)
			var d38 scm.JITValueDesc
			if d13.Loc == scm.LocImm && d37.Loc == scm.LocImm {
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d13.Imm.Int() | d37.Imm.Int())}
			} else if d13.Loc == scm.LocImm && d13.Imm.Int() == 0 {
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d37.Reg}
				ctx.BindReg(d37.Reg, &d38)
			} else if d37.Loc == scm.LocImm && d37.Imm.Int() == 0 {
				r41 := ctx.AllocRegExcept(d13.Reg)
				ctx.W.EmitMovRegReg(r41, d13.Reg)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d38)
			} else if d13.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d37.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d13.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d37.Reg)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d38)
			} else if d37.Loc == scm.LocImm {
				r42 := ctx.AllocRegExcept(d13.Reg)
				ctx.W.EmitMovRegReg(r42, d13.Reg)
				if d37.Imm.Int() >= -2147483648 && d37.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r42, int32(d37.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d37.Imm.Int()))
					ctx.W.EmitOrInt64(r42, scm.RegR11)
				}
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d38)
			} else {
				r43 := ctx.AllocRegExcept(d13.Reg, d37.Reg)
				ctx.W.EmitMovRegReg(r43, d13.Reg)
				ctx.W.EmitOrInt64(r43, d37.Reg)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
				ctx.BindReg(r43, &d38)
			}
			if d38.Loc == scm.LocReg && d13.Loc == scm.LocReg && d38.Reg == d13.Reg {
				ctx.TransferReg(d13.Reg)
				d13.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d37)
			d39 = d38
			if d39.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d39)
			ctx.EmitStoreToStack(d39, 0)
			ctx.W.EmitJmp(lbl16)
			ctx.W.MarkLabel(lbl14)
			d40 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r26}
			ctx.BindReg(r26, &d40)
			ctx.BindReg(r26, &d40)
			if r5 { ctx.UnprotectReg(r6) }
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d40)
			var d41 scm.JITValueDesc
			if d40.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d40.Imm.Int()))))}
			} else {
				r44 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r44, d40.Reg)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d41)
			}
			ctx.FreeDesc(&d40)
			var d42 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 32)
				r45 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r45, thisptr.Reg, off)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r45}
				ctx.BindReg(r45, &d42)
			}
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d42)
			var d43 scm.JITValueDesc
			if d41.Loc == scm.LocImm && d42.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d41.Imm.Int() + d42.Imm.Int())}
			} else if d42.Loc == scm.LocImm && d42.Imm.Int() == 0 {
				r46 := ctx.AllocRegExcept(d41.Reg)
				ctx.W.EmitMovRegReg(r46, d41.Reg)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d43)
			} else if d41.Loc == scm.LocImm && d41.Imm.Int() == 0 {
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d42.Reg}
				ctx.BindReg(d42.Reg, &d43)
			} else if d41.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d42.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d41.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d42.Reg)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d43)
			} else if d42.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d41.Reg)
				ctx.W.EmitMovRegReg(scratch, d41.Reg)
				if d42.Imm.Int() >= -2147483648 && d42.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d42.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d42.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d43)
			} else {
				r47 := ctx.AllocRegExcept(d41.Reg, d42.Reg)
				ctx.W.EmitMovRegReg(r47, d41.Reg)
				ctx.W.EmitAddInt64(r47, d42.Reg)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
				ctx.BindReg(r47, &d43)
			}
			if d43.Loc == scm.LocReg && d41.Loc == scm.LocReg && d43.Reg == d41.Reg {
				ctx.TransferReg(d41.Reg)
				d41.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d41)
			ctx.FreeDesc(&d42)
			ctx.EnsureDesc(&d43)
			ctx.EnsureDesc(&d43)
			var d44 scm.JITValueDesc
			if d43.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d43.Imm.Int()))))}
			} else {
				r48 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r48, d43.Reg)
				ctx.W.EmitShlRegImm8(r48, 32)
				ctx.W.EmitShrRegImm8(r48, 32)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
				ctx.BindReg(r48, &d44)
			}
			ctx.FreeDesc(&d43)
			var d45 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 56)
				r49 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r49, thisptr.Reg, off)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r49}
				ctx.BindReg(r49, &d45)
			}
			d46 = d45
			ctx.EnsureDesc(&d46)
			if d46.Loc != scm.LocImm && d46.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl22 := ctx.W.ReserveLabel()
			lbl23 := ctx.W.ReserveLabel()
			lbl24 := ctx.W.ReserveLabel()
			lbl25 := ctx.W.ReserveLabel()
			if d46.Loc == scm.LocImm {
				if d46.Imm.Bool() {
					ctx.W.MarkLabel(lbl24)
					ctx.W.EmitJmp(lbl22)
				} else {
					ctx.W.MarkLabel(lbl25)
					ctx.W.EmitJmp(lbl23)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d46.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl24)
				ctx.W.EmitJmp(lbl25)
				ctx.W.MarkLabel(lbl24)
				ctx.W.EmitJmp(lbl22)
				ctx.W.MarkLabel(lbl25)
				ctx.W.EmitJmp(lbl23)
			}
			ctx.FreeDesc(&d45)
			bbpos_1_7 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl23)
			ctx.W.ResolveFixups()
			ctx.EnsureDesc(&d44)
			d47 = d44
			_ = d47
			r50 := d44.Loc == scm.LocReg
			r51 := d44.Reg
			if r50 { ctx.ProtectReg(r51) }
			d48 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			lbl26 := ctx.W.ReserveLabel()
			bbpos_3_0 := int32(-1)
			_ = bbpos_3_0
			bbpos_3_1 := int32(-1)
			_ = bbpos_3_1
			bbpos_3_2 := int32(-1)
			_ = bbpos_3_2
			bbpos_3_3 := int32(-1)
			_ = bbpos_3_3
			bbpos_3_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			d48 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d47)
			var d49 scm.JITValueDesc
			if d47.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d47.Imm.Int()))))}
			} else {
				r52 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r52, d47.Reg)
				ctx.W.EmitShlRegImm8(r52, 32)
				ctx.W.EmitShrRegImm8(r52, 32)
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
				ctx.BindReg(r52, &d49)
			}
			var d50 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r53 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r53, thisptr.Reg, off)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r53}
				ctx.BindReg(r53, &d50)
			}
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d50)
			var d51 scm.JITValueDesc
			if d50.Loc == scm.LocImm {
				d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d50.Imm.Int()))))}
			} else {
				r54 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r54, d50.Reg)
				ctx.W.EmitShlRegImm8(r54, 56)
				ctx.W.EmitShrRegImm8(r54, 56)
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
				ctx.BindReg(r54, &d51)
			}
			ctx.FreeDesc(&d50)
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d51)
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d51)
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d51)
			var d52 scm.JITValueDesc
			if d49.Loc == scm.LocImm && d51.Loc == scm.LocImm {
				d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d49.Imm.Int() * d51.Imm.Int())}
			} else if d49.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d51.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d49.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d51.Reg)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d52)
			} else if d51.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegReg(scratch, d49.Reg)
				if d51.Imm.Int() >= -2147483648 && d51.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d51.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d51.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d52)
			} else {
				r55 := ctx.AllocRegExcept(d49.Reg, d51.Reg)
				ctx.W.EmitMovRegReg(r55, d49.Reg)
				ctx.W.EmitImulInt64(r55, d51.Reg)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
				ctx.BindReg(r55, &d52)
			}
			if d52.Loc == scm.LocReg && d49.Loc == scm.LocReg && d52.Reg == d49.Reg {
				ctx.TransferReg(d49.Reg)
				d49.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d49)
			ctx.FreeDesc(&d51)
			var d53 scm.JITValueDesc
			r56 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r56, uint64(dataPtr))
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r56, StackOff: int32(sliceLen)}
				ctx.BindReg(r56, &d53)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0)
				ctx.W.EmitMovRegMem(r56, thisptr.Reg, off)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r56}
				ctx.BindReg(r56, &d53)
			}
			ctx.BindReg(r56, &d53)
			ctx.EnsureDesc(&d52)
			var d54 scm.JITValueDesc
			if d52.Loc == scm.LocImm {
				d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() / 64)}
			} else {
				r57 := ctx.AllocRegExcept(d52.Reg)
				ctx.W.EmitMovRegReg(r57, d52.Reg)
				ctx.W.EmitShrRegImm8(r57, 6)
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r57}
				ctx.BindReg(r57, &d54)
			}
			if d54.Loc == scm.LocReg && d52.Loc == scm.LocReg && d54.Reg == d52.Reg {
				ctx.TransferReg(d52.Reg)
				d52.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d54)
			r58 := ctx.AllocReg()
			ctx.EnsureDesc(&d54)
			ctx.EnsureDesc(&d53)
			if d54.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r58, uint64(d54.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r58, d54.Reg)
				ctx.W.EmitShlRegImm8(r58, 3)
			}
			if d53.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d53.Imm.Int()))
				ctx.W.EmitAddInt64(r58, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r58, d53.Reg)
			}
			r59 := ctx.AllocRegExcept(r58)
			ctx.W.EmitMovRegMem(r59, r58, 0)
			ctx.FreeReg(r58)
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r59}
			ctx.BindReg(r59, &d55)
			ctx.FreeDesc(&d54)
			ctx.EnsureDesc(&d52)
			var d56 scm.JITValueDesc
			if d52.Loc == scm.LocImm {
				d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() % 64)}
			} else {
				r60 := ctx.AllocRegExcept(d52.Reg)
				ctx.W.EmitMovRegReg(r60, d52.Reg)
				ctx.W.EmitAndRegImm32(r60, 63)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r60}
				ctx.BindReg(r60, &d56)
			}
			if d56.Loc == scm.LocReg && d52.Loc == scm.LocReg && d56.Reg == d52.Reg {
				ctx.TransferReg(d52.Reg)
				d52.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d55)
			ctx.EnsureDesc(&d56)
			var d57 scm.JITValueDesc
			if d55.Loc == scm.LocImm && d56.Loc == scm.LocImm {
				d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d55.Imm.Int()) << uint64(d56.Imm.Int())))}
			} else if d56.Loc == scm.LocImm {
				r61 := ctx.AllocRegExcept(d55.Reg)
				ctx.W.EmitMovRegReg(r61, d55.Reg)
				ctx.W.EmitShlRegImm8(r61, uint8(d56.Imm.Int()))
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r61}
				ctx.BindReg(r61, &d57)
			} else {
				{
					shiftSrc := d55.Reg
					r62 := ctx.AllocRegExcept(d55.Reg)
					ctx.W.EmitMovRegReg(r62, d55.Reg)
					shiftSrc = r62
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d56.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d56.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d56.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d57)
				}
			}
			if d57.Loc == scm.LocReg && d55.Loc == scm.LocReg && d57.Reg == d55.Reg {
				ctx.TransferReg(d55.Reg)
				d55.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d55)
			ctx.FreeDesc(&d56)
			var d58 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25)
				r63 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r63, thisptr.Reg, off)
				d58 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r63}
				ctx.BindReg(r63, &d58)
			}
			d59 = d58
			ctx.EnsureDesc(&d59)
			if d59.Loc != scm.LocImm && d59.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl27 := ctx.W.ReserveLabel()
			lbl28 := ctx.W.ReserveLabel()
			lbl29 := ctx.W.ReserveLabel()
			lbl30 := ctx.W.ReserveLabel()
			if d59.Loc == scm.LocImm {
				if d59.Imm.Bool() {
					ctx.W.MarkLabel(lbl29)
					ctx.W.EmitJmp(lbl27)
				} else {
					ctx.W.MarkLabel(lbl30)
			d60 = d57
			if d60.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d60)
			ctx.EmitStoreToStack(d60, 8)
					ctx.W.EmitJmp(lbl28)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d59.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl29)
				ctx.W.EmitJmp(lbl30)
				ctx.W.MarkLabel(lbl29)
				ctx.W.EmitJmp(lbl27)
				ctx.W.MarkLabel(lbl30)
			d61 = d57
			if d61.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d61)
			ctx.EmitStoreToStack(d61, 8)
				ctx.W.EmitJmp(lbl28)
			}
			ctx.FreeDesc(&d58)
			bbpos_3_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl28)
			ctx.W.ResolveFixups()
			d48 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			var d62 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r64 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r64, thisptr.Reg, off)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r64}
				ctx.BindReg(r64, &d62)
			}
			ctx.EnsureDesc(&d62)
			ctx.EnsureDesc(&d62)
			var d63 scm.JITValueDesc
			if d62.Loc == scm.LocImm {
				d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d62.Imm.Int()))))}
			} else {
				r65 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r65, d62.Reg)
				ctx.W.EmitShlRegImm8(r65, 56)
				ctx.W.EmitShrRegImm8(r65, 56)
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
				ctx.BindReg(r65, &d63)
			}
			ctx.FreeDesc(&d62)
			d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d63)
			ctx.EnsureDesc(&d64)
			ctx.EnsureDesc(&d63)
			ctx.EnsureDesc(&d64)
			ctx.EnsureDesc(&d63)
			var d65 scm.JITValueDesc
			if d64.Loc == scm.LocImm && d63.Loc == scm.LocImm {
				d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d64.Imm.Int() - d63.Imm.Int())}
			} else if d63.Loc == scm.LocImm && d63.Imm.Int() == 0 {
				r66 := ctx.AllocRegExcept(d64.Reg)
				ctx.W.EmitMovRegReg(r66, d64.Reg)
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
				ctx.BindReg(r66, &d65)
			} else if d64.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d63.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d64.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d63.Reg)
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d65)
			} else if d63.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d64.Reg)
				ctx.W.EmitMovRegReg(scratch, d64.Reg)
				if d63.Imm.Int() >= -2147483648 && d63.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d63.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d63.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d65)
			} else {
				r67 := ctx.AllocRegExcept(d64.Reg, d63.Reg)
				ctx.W.EmitMovRegReg(r67, d64.Reg)
				ctx.W.EmitSubInt64(r67, d63.Reg)
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r67}
				ctx.BindReg(r67, &d65)
			}
			if d65.Loc == scm.LocReg && d64.Loc == scm.LocReg && d65.Reg == d64.Reg {
				ctx.TransferReg(d64.Reg)
				d64.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d63)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d65)
			var d66 scm.JITValueDesc
			if d48.Loc == scm.LocImm && d65.Loc == scm.LocImm {
				d66 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d48.Imm.Int()) >> uint64(d65.Imm.Int())))}
			} else if d65.Loc == scm.LocImm {
				r68 := ctx.AllocRegExcept(d48.Reg)
				ctx.W.EmitMovRegReg(r68, d48.Reg)
				ctx.W.EmitShrRegImm8(r68, uint8(d65.Imm.Int()))
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r68}
				ctx.BindReg(r68, &d66)
			} else {
				{
					shiftSrc := d48.Reg
					r69 := ctx.AllocRegExcept(d48.Reg)
					ctx.W.EmitMovRegReg(r69, d48.Reg)
					shiftSrc = r69
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d65.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d65.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d65.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d66)
				}
			}
			if d66.Loc == scm.LocReg && d48.Loc == scm.LocReg && d66.Reg == d48.Reg {
				ctx.TransferReg(d48.Reg)
				d48.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d48)
			ctx.FreeDesc(&d65)
			r70 := ctx.AllocReg()
			ctx.EnsureDesc(&d66)
			ctx.EnsureDesc(&d66)
			if d66.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r70, d66)
			}
			ctx.W.EmitJmp(lbl26)
			bbpos_3_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl27)
			ctx.W.ResolveFixups()
			d48 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d52)
			var d67 scm.JITValueDesc
			if d52.Loc == scm.LocImm {
				d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() % 64)}
			} else {
				r71 := ctx.AllocRegExcept(d52.Reg)
				ctx.W.EmitMovRegReg(r71, d52.Reg)
				ctx.W.EmitAndRegImm32(r71, 63)
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r71}
				ctx.BindReg(r71, &d67)
			}
			if d67.Loc == scm.LocReg && d52.Loc == scm.LocReg && d67.Reg == d52.Reg {
				ctx.TransferReg(d52.Reg)
				d52.Loc = scm.LocNone
			}
			var d68 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r72 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r72, thisptr.Reg, off)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r72}
				ctx.BindReg(r72, &d68)
			}
			ctx.EnsureDesc(&d68)
			ctx.EnsureDesc(&d68)
			var d69 scm.JITValueDesc
			if d68.Loc == scm.LocImm {
				d69 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d68.Imm.Int()))))}
			} else {
				r73 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r73, d68.Reg)
				ctx.W.EmitShlRegImm8(r73, 56)
				ctx.W.EmitShrRegImm8(r73, 56)
				d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
				ctx.BindReg(r73, &d69)
			}
			ctx.FreeDesc(&d68)
			ctx.EnsureDesc(&d67)
			ctx.EnsureDesc(&d69)
			ctx.EnsureDesc(&d67)
			ctx.EnsureDesc(&d69)
			ctx.EnsureDesc(&d67)
			ctx.EnsureDesc(&d69)
			var d70 scm.JITValueDesc
			if d67.Loc == scm.LocImm && d69.Loc == scm.LocImm {
				d70 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d67.Imm.Int() + d69.Imm.Int())}
			} else if d69.Loc == scm.LocImm && d69.Imm.Int() == 0 {
				r74 := ctx.AllocRegExcept(d67.Reg)
				ctx.W.EmitMovRegReg(r74, d67.Reg)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r74}
				ctx.BindReg(r74, &d70)
			} else if d67.Loc == scm.LocImm && d67.Imm.Int() == 0 {
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d69.Reg}
				ctx.BindReg(d69.Reg, &d70)
			} else if d67.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d69.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d67.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d69.Reg)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d70)
			} else if d69.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d67.Reg)
				ctx.W.EmitMovRegReg(scratch, d67.Reg)
				if d69.Imm.Int() >= -2147483648 && d69.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d69.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d69.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d70)
			} else {
				r75 := ctx.AllocRegExcept(d67.Reg, d69.Reg)
				ctx.W.EmitMovRegReg(r75, d67.Reg)
				ctx.W.EmitAddInt64(r75, d69.Reg)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
				ctx.BindReg(r75, &d70)
			}
			if d70.Loc == scm.LocReg && d67.Loc == scm.LocReg && d70.Reg == d67.Reg {
				ctx.TransferReg(d67.Reg)
				d67.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d67)
			ctx.FreeDesc(&d69)
			ctx.EnsureDesc(&d70)
			var d71 scm.JITValueDesc
			if d70.Loc == scm.LocImm {
				d71 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d70.Imm.Int()) > uint64(64))}
			} else {
				r76 := ctx.AllocRegExcept(d70.Reg)
				ctx.W.EmitCmpRegImm32(d70.Reg, 64)
				ctx.W.EmitSetcc(r76, scm.CcA)
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r76}
				ctx.BindReg(r76, &d71)
			}
			ctx.FreeDesc(&d70)
			d72 = d71
			ctx.EnsureDesc(&d72)
			if d72.Loc != scm.LocImm && d72.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl31 := ctx.W.ReserveLabel()
			lbl32 := ctx.W.ReserveLabel()
			lbl33 := ctx.W.ReserveLabel()
			if d72.Loc == scm.LocImm {
				if d72.Imm.Bool() {
					ctx.W.MarkLabel(lbl32)
					ctx.W.EmitJmp(lbl31)
				} else {
					ctx.W.MarkLabel(lbl33)
			d73 = d57
			if d73.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d73)
			ctx.EmitStoreToStack(d73, 8)
					ctx.W.EmitJmp(lbl28)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d72.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl32)
				ctx.W.EmitJmp(lbl33)
				ctx.W.MarkLabel(lbl32)
				ctx.W.EmitJmp(lbl31)
				ctx.W.MarkLabel(lbl33)
			d74 = d57
			if d74.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d74)
			ctx.EmitStoreToStack(d74, 8)
				ctx.W.EmitJmp(lbl28)
			}
			ctx.FreeDesc(&d71)
			bbpos_3_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl31)
			ctx.W.ResolveFixups()
			d48 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d52)
			var d75 scm.JITValueDesc
			if d52.Loc == scm.LocImm {
				d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() / 64)}
			} else {
				r77 := ctx.AllocRegExcept(d52.Reg)
				ctx.W.EmitMovRegReg(r77, d52.Reg)
				ctx.W.EmitShrRegImm8(r77, 6)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r77}
				ctx.BindReg(r77, &d75)
			}
			if d75.Loc == scm.LocReg && d52.Loc == scm.LocReg && d75.Reg == d52.Reg {
				ctx.TransferReg(d52.Reg)
				d52.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d75)
			ctx.EnsureDesc(&d75)
			var d76 scm.JITValueDesc
			if d75.Loc == scm.LocImm {
				d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d75.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d75.Reg)
				ctx.W.EmitMovRegReg(scratch, d75.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d76)
			}
			if d76.Loc == scm.LocReg && d75.Loc == scm.LocReg && d76.Reg == d75.Reg {
				ctx.TransferReg(d75.Reg)
				d75.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d75)
			ctx.EnsureDesc(&d76)
			r78 := ctx.AllocReg()
			ctx.EnsureDesc(&d76)
			ctx.EnsureDesc(&d53)
			if d76.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r78, uint64(d76.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r78, d76.Reg)
				ctx.W.EmitShlRegImm8(r78, 3)
			}
			if d53.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d53.Imm.Int()))
				ctx.W.EmitAddInt64(r78, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r78, d53.Reg)
			}
			r79 := ctx.AllocRegExcept(r78)
			ctx.W.EmitMovRegMem(r79, r78, 0)
			ctx.FreeReg(r78)
			d77 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r79}
			ctx.BindReg(r79, &d77)
			ctx.FreeDesc(&d76)
			ctx.EnsureDesc(&d52)
			var d78 scm.JITValueDesc
			if d52.Loc == scm.LocImm {
				d78 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() % 64)}
			} else {
				r80 := ctx.AllocRegExcept(d52.Reg)
				ctx.W.EmitMovRegReg(r80, d52.Reg)
				ctx.W.EmitAndRegImm32(r80, 63)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
				ctx.BindReg(r80, &d78)
			}
			if d78.Loc == scm.LocReg && d52.Loc == scm.LocReg && d78.Reg == d52.Reg {
				ctx.TransferReg(d52.Reg)
				d52.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d52)
			d79 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d78)
			ctx.EnsureDesc(&d79)
			ctx.EnsureDesc(&d78)
			ctx.EnsureDesc(&d79)
			ctx.EnsureDesc(&d78)
			var d80 scm.JITValueDesc
			if d79.Loc == scm.LocImm && d78.Loc == scm.LocImm {
				d80 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d79.Imm.Int() - d78.Imm.Int())}
			} else if d78.Loc == scm.LocImm && d78.Imm.Int() == 0 {
				r81 := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitMovRegReg(r81, d79.Reg)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
				ctx.BindReg(r81, &d80)
			} else if d79.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d78.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d79.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d78.Reg)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d80)
			} else if d78.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitMovRegReg(scratch, d79.Reg)
				if d78.Imm.Int() >= -2147483648 && d78.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d78.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d78.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d80)
			} else {
				r82 := ctx.AllocRegExcept(d79.Reg, d78.Reg)
				ctx.W.EmitMovRegReg(r82, d79.Reg)
				ctx.W.EmitSubInt64(r82, d78.Reg)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r82}
				ctx.BindReg(r82, &d80)
			}
			if d80.Loc == scm.LocReg && d79.Loc == scm.LocReg && d80.Reg == d79.Reg {
				ctx.TransferReg(d79.Reg)
				d79.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d78)
			ctx.EnsureDesc(&d77)
			ctx.EnsureDesc(&d80)
			var d81 scm.JITValueDesc
			if d77.Loc == scm.LocImm && d80.Loc == scm.LocImm {
				d81 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d77.Imm.Int()) >> uint64(d80.Imm.Int())))}
			} else if d80.Loc == scm.LocImm {
				r83 := ctx.AllocRegExcept(d77.Reg)
				ctx.W.EmitMovRegReg(r83, d77.Reg)
				ctx.W.EmitShrRegImm8(r83, uint8(d80.Imm.Int()))
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
				ctx.BindReg(r83, &d81)
			} else {
				{
					shiftSrc := d77.Reg
					r84 := ctx.AllocRegExcept(d77.Reg)
					ctx.W.EmitMovRegReg(r84, d77.Reg)
					shiftSrc = r84
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d80.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d80.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d80.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d81)
				}
			}
			if d81.Loc == scm.LocReg && d77.Loc == scm.LocReg && d81.Reg == d77.Reg {
				ctx.TransferReg(d77.Reg)
				d77.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d77)
			ctx.FreeDesc(&d80)
			ctx.EnsureDesc(&d57)
			ctx.EnsureDesc(&d81)
			var d82 scm.JITValueDesc
			if d57.Loc == scm.LocImm && d81.Loc == scm.LocImm {
				d82 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d57.Imm.Int() | d81.Imm.Int())}
			} else if d57.Loc == scm.LocImm && d57.Imm.Int() == 0 {
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d81.Reg}
				ctx.BindReg(d81.Reg, &d82)
			} else if d81.Loc == scm.LocImm && d81.Imm.Int() == 0 {
				r85 := ctx.AllocRegExcept(d57.Reg)
				ctx.W.EmitMovRegReg(r85, d57.Reg)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
				ctx.BindReg(r85, &d82)
			} else if d57.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d81.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d57.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d81.Reg)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d82)
			} else if d81.Loc == scm.LocImm {
				r86 := ctx.AllocRegExcept(d57.Reg)
				ctx.W.EmitMovRegReg(r86, d57.Reg)
				if d81.Imm.Int() >= -2147483648 && d81.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r86, int32(d81.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d81.Imm.Int()))
					ctx.W.EmitOrInt64(r86, scm.RegR11)
				}
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
				ctx.BindReg(r86, &d82)
			} else {
				r87 := ctx.AllocRegExcept(d57.Reg, d81.Reg)
				ctx.W.EmitMovRegReg(r87, d57.Reg)
				ctx.W.EmitOrInt64(r87, d81.Reg)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r87}
				ctx.BindReg(r87, &d82)
			}
			if d82.Loc == scm.LocReg && d57.Loc == scm.LocReg && d82.Reg == d57.Reg {
				ctx.TransferReg(d57.Reg)
				d57.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d81)
			d83 = d82
			if d83.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d83)
			ctx.EmitStoreToStack(d83, 8)
			ctx.W.EmitJmp(lbl28)
			ctx.W.MarkLabel(lbl26)
			d84 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r70}
			ctx.BindReg(r70, &d84)
			ctx.BindReg(r70, &d84)
			if r50 { ctx.UnprotectReg(r51) }
			ctx.EnsureDesc(&d84)
			ctx.EnsureDesc(&d84)
			var d85 scm.JITValueDesc
			if d84.Loc == scm.LocImm {
				d85 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d84.Imm.Int()))))}
			} else {
				r88 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r88, d84.Reg)
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
				ctx.BindReg(r88, &d85)
			}
			ctx.FreeDesc(&d84)
			var d86 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32)
				r89 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r89, thisptr.Reg, off)
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r89}
				ctx.BindReg(r89, &d86)
			}
			ctx.EnsureDesc(&d85)
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d85)
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d85)
			ctx.EnsureDesc(&d86)
			var d87 scm.JITValueDesc
			if d85.Loc == scm.LocImm && d86.Loc == scm.LocImm {
				d87 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d85.Imm.Int() + d86.Imm.Int())}
			} else if d86.Loc == scm.LocImm && d86.Imm.Int() == 0 {
				r90 := ctx.AllocRegExcept(d85.Reg)
				ctx.W.EmitMovRegReg(r90, d85.Reg)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
				ctx.BindReg(r90, &d87)
			} else if d85.Loc == scm.LocImm && d85.Imm.Int() == 0 {
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d86.Reg}
				ctx.BindReg(d86.Reg, &d87)
			} else if d85.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d86.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d85.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d86.Reg)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d87)
			} else if d86.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d85.Reg)
				ctx.W.EmitMovRegReg(scratch, d85.Reg)
				if d86.Imm.Int() >= -2147483648 && d86.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d86.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d86.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d87)
			} else {
				r91 := ctx.AllocRegExcept(d85.Reg, d86.Reg)
				ctx.W.EmitMovRegReg(r91, d85.Reg)
				ctx.W.EmitAddInt64(r91, d86.Reg)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r91}
				ctx.BindReg(r91, &d87)
			}
			if d87.Loc == scm.LocReg && d85.Loc == scm.LocReg && d87.Reg == d85.Reg {
				ctx.TransferReg(d85.Reg)
				d85.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d85)
			ctx.FreeDesc(&d86)
			ctx.EnsureDesc(&d44)
			d88 = d44
			_ = d88
			r92 := d44.Loc == scm.LocReg
			r93 := d44.Reg
			if r92 { ctx.ProtectReg(r93) }
			d89 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			lbl34 := ctx.W.ReserveLabel()
			bbpos_4_0 := int32(-1)
			_ = bbpos_4_0
			bbpos_4_1 := int32(-1)
			_ = bbpos_4_1
			bbpos_4_2 := int32(-1)
			_ = bbpos_4_2
			bbpos_4_3 := int32(-1)
			_ = bbpos_4_3
			bbpos_4_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			d89 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d88)
			ctx.EnsureDesc(&d88)
			var d90 scm.JITValueDesc
			if d88.Loc == scm.LocImm {
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d88.Imm.Int()))))}
			} else {
				r94 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r94, d88.Reg)
				ctx.W.EmitShlRegImm8(r94, 32)
				ctx.W.EmitShrRegImm8(r94, 32)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
				ctx.BindReg(r94, &d90)
			}
			var d91 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d91 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r95 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r95, thisptr.Reg, off)
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r95}
				ctx.BindReg(r95, &d91)
			}
			ctx.EnsureDesc(&d91)
			ctx.EnsureDesc(&d91)
			var d92 scm.JITValueDesc
			if d91.Loc == scm.LocImm {
				d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d91.Imm.Int()))))}
			} else {
				r96 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r96, d91.Reg)
				ctx.W.EmitShlRegImm8(r96, 56)
				ctx.W.EmitShrRegImm8(r96, 56)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
				ctx.BindReg(r96, &d92)
			}
			ctx.FreeDesc(&d91)
			ctx.EnsureDesc(&d90)
			ctx.EnsureDesc(&d92)
			ctx.EnsureDesc(&d90)
			ctx.EnsureDesc(&d92)
			ctx.EnsureDesc(&d90)
			ctx.EnsureDesc(&d92)
			var d93 scm.JITValueDesc
			if d90.Loc == scm.LocImm && d92.Loc == scm.LocImm {
				d93 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d90.Imm.Int() * d92.Imm.Int())}
			} else if d90.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d92.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d90.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d92.Reg)
				d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d93)
			} else if d92.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d90.Reg)
				ctx.W.EmitMovRegReg(scratch, d90.Reg)
				if d92.Imm.Int() >= -2147483648 && d92.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d92.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d92.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d93)
			} else {
				r97 := ctx.AllocRegExcept(d90.Reg, d92.Reg)
				ctx.W.EmitMovRegReg(r97, d90.Reg)
				ctx.W.EmitImulInt64(r97, d92.Reg)
				d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
				ctx.BindReg(r97, &d93)
			}
			if d93.Loc == scm.LocReg && d90.Loc == scm.LocReg && d93.Reg == d90.Reg {
				ctx.TransferReg(d90.Reg)
				d90.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d90)
			ctx.FreeDesc(&d92)
			var d94 scm.JITValueDesc
			r98 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r98, uint64(dataPtr))
				d94 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r98, StackOff: int32(sliceLen)}
				ctx.BindReg(r98, &d94)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0)
				ctx.W.EmitMovRegMem(r98, thisptr.Reg, off)
				d94 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r98}
				ctx.BindReg(r98, &d94)
			}
			ctx.BindReg(r98, &d94)
			ctx.EnsureDesc(&d93)
			var d95 scm.JITValueDesc
			if d93.Loc == scm.LocImm {
				d95 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d93.Imm.Int() / 64)}
			} else {
				r99 := ctx.AllocRegExcept(d93.Reg)
				ctx.W.EmitMovRegReg(r99, d93.Reg)
				ctx.W.EmitShrRegImm8(r99, 6)
				d95 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
				ctx.BindReg(r99, &d95)
			}
			if d95.Loc == scm.LocReg && d93.Loc == scm.LocReg && d95.Reg == d93.Reg {
				ctx.TransferReg(d93.Reg)
				d93.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d95)
			r100 := ctx.AllocReg()
			ctx.EnsureDesc(&d95)
			ctx.EnsureDesc(&d94)
			if d95.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r100, uint64(d95.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r100, d95.Reg)
				ctx.W.EmitShlRegImm8(r100, 3)
			}
			if d94.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d94.Imm.Int()))
				ctx.W.EmitAddInt64(r100, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r100, d94.Reg)
			}
			r101 := ctx.AllocRegExcept(r100)
			ctx.W.EmitMovRegMem(r101, r100, 0)
			ctx.FreeReg(r100)
			d96 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r101}
			ctx.BindReg(r101, &d96)
			ctx.FreeDesc(&d95)
			ctx.EnsureDesc(&d93)
			var d97 scm.JITValueDesc
			if d93.Loc == scm.LocImm {
				d97 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d93.Imm.Int() % 64)}
			} else {
				r102 := ctx.AllocRegExcept(d93.Reg)
				ctx.W.EmitMovRegReg(r102, d93.Reg)
				ctx.W.EmitAndRegImm32(r102, 63)
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r102}
				ctx.BindReg(r102, &d97)
			}
			if d97.Loc == scm.LocReg && d93.Loc == scm.LocReg && d97.Reg == d93.Reg {
				ctx.TransferReg(d93.Reg)
				d93.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d96)
			ctx.EnsureDesc(&d97)
			var d98 scm.JITValueDesc
			if d96.Loc == scm.LocImm && d97.Loc == scm.LocImm {
				d98 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d96.Imm.Int()) << uint64(d97.Imm.Int())))}
			} else if d97.Loc == scm.LocImm {
				r103 := ctx.AllocRegExcept(d96.Reg)
				ctx.W.EmitMovRegReg(r103, d96.Reg)
				ctx.W.EmitShlRegImm8(r103, uint8(d97.Imm.Int()))
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r103}
				ctx.BindReg(r103, &d98)
			} else {
				{
					shiftSrc := d96.Reg
					r104 := ctx.AllocRegExcept(d96.Reg)
					ctx.W.EmitMovRegReg(r104, d96.Reg)
					shiftSrc = r104
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d97.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d97.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d97.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d98)
				}
			}
			if d98.Loc == scm.LocReg && d96.Loc == scm.LocReg && d98.Reg == d96.Reg {
				ctx.TransferReg(d96.Reg)
				d96.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d96)
			ctx.FreeDesc(&d97)
			var d99 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d99 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25)
				r105 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r105, thisptr.Reg, off)
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r105}
				ctx.BindReg(r105, &d99)
			}
			d100 = d99
			ctx.EnsureDesc(&d100)
			if d100.Loc != scm.LocImm && d100.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl35 := ctx.W.ReserveLabel()
			lbl36 := ctx.W.ReserveLabel()
			lbl37 := ctx.W.ReserveLabel()
			lbl38 := ctx.W.ReserveLabel()
			if d100.Loc == scm.LocImm {
				if d100.Imm.Bool() {
					ctx.W.MarkLabel(lbl37)
					ctx.W.EmitJmp(lbl35)
				} else {
					ctx.W.MarkLabel(lbl38)
			d101 = d98
			if d101.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d101)
			ctx.EmitStoreToStack(d101, 16)
					ctx.W.EmitJmp(lbl36)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d100.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl37)
				ctx.W.EmitJmp(lbl38)
				ctx.W.MarkLabel(lbl37)
				ctx.W.EmitJmp(lbl35)
				ctx.W.MarkLabel(lbl38)
			d102 = d98
			if d102.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d102)
			ctx.EmitStoreToStack(d102, 16)
				ctx.W.EmitJmp(lbl36)
			}
			ctx.FreeDesc(&d99)
			bbpos_4_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl36)
			ctx.W.ResolveFixups()
			d89 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			var d103 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d103 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r106 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r106, thisptr.Reg, off)
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r106}
				ctx.BindReg(r106, &d103)
			}
			ctx.EnsureDesc(&d103)
			ctx.EnsureDesc(&d103)
			var d104 scm.JITValueDesc
			if d103.Loc == scm.LocImm {
				d104 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d103.Imm.Int()))))}
			} else {
				r107 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r107, d103.Reg)
				ctx.W.EmitShlRegImm8(r107, 56)
				ctx.W.EmitShrRegImm8(r107, 56)
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r107}
				ctx.BindReg(r107, &d104)
			}
			ctx.FreeDesc(&d103)
			d105 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d104)
			ctx.EnsureDesc(&d105)
			ctx.EnsureDesc(&d104)
			ctx.EnsureDesc(&d105)
			ctx.EnsureDesc(&d104)
			var d106 scm.JITValueDesc
			if d105.Loc == scm.LocImm && d104.Loc == scm.LocImm {
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d105.Imm.Int() - d104.Imm.Int())}
			} else if d104.Loc == scm.LocImm && d104.Imm.Int() == 0 {
				r108 := ctx.AllocRegExcept(d105.Reg)
				ctx.W.EmitMovRegReg(r108, d105.Reg)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
				ctx.BindReg(r108, &d106)
			} else if d105.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d104.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d105.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d104.Reg)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d106)
			} else if d104.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d105.Reg)
				ctx.W.EmitMovRegReg(scratch, d105.Reg)
				if d104.Imm.Int() >= -2147483648 && d104.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d104.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d104.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d106)
			} else {
				r109 := ctx.AllocRegExcept(d105.Reg, d104.Reg)
				ctx.W.EmitMovRegReg(r109, d105.Reg)
				ctx.W.EmitSubInt64(r109, d104.Reg)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r109}
				ctx.BindReg(r109, &d106)
			}
			if d106.Loc == scm.LocReg && d105.Loc == scm.LocReg && d106.Reg == d105.Reg {
				ctx.TransferReg(d105.Reg)
				d105.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d104)
			ctx.EnsureDesc(&d89)
			ctx.EnsureDesc(&d106)
			var d107 scm.JITValueDesc
			if d89.Loc == scm.LocImm && d106.Loc == scm.LocImm {
				d107 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d89.Imm.Int()) >> uint64(d106.Imm.Int())))}
			} else if d106.Loc == scm.LocImm {
				r110 := ctx.AllocRegExcept(d89.Reg)
				ctx.W.EmitMovRegReg(r110, d89.Reg)
				ctx.W.EmitShrRegImm8(r110, uint8(d106.Imm.Int()))
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
				ctx.BindReg(r110, &d107)
			} else {
				{
					shiftSrc := d89.Reg
					r111 := ctx.AllocRegExcept(d89.Reg)
					ctx.W.EmitMovRegReg(r111, d89.Reg)
					shiftSrc = r111
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d106.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d106.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d106.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d107 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d107)
				}
			}
			if d107.Loc == scm.LocReg && d89.Loc == scm.LocReg && d107.Reg == d89.Reg {
				ctx.TransferReg(d89.Reg)
				d89.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d89)
			ctx.FreeDesc(&d106)
			r112 := ctx.AllocReg()
			ctx.EnsureDesc(&d107)
			ctx.EnsureDesc(&d107)
			if d107.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r112, d107)
			}
			ctx.W.EmitJmp(lbl34)
			bbpos_4_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl35)
			ctx.W.ResolveFixups()
			d89 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d93)
			var d108 scm.JITValueDesc
			if d93.Loc == scm.LocImm {
				d108 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d93.Imm.Int() % 64)}
			} else {
				r113 := ctx.AllocRegExcept(d93.Reg)
				ctx.W.EmitMovRegReg(r113, d93.Reg)
				ctx.W.EmitAndRegImm32(r113, 63)
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r113}
				ctx.BindReg(r113, &d108)
			}
			if d108.Loc == scm.LocReg && d93.Loc == scm.LocReg && d108.Reg == d93.Reg {
				ctx.TransferReg(d93.Reg)
				d93.Loc = scm.LocNone
			}
			var d109 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d109 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r114 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r114, thisptr.Reg, off)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r114}
				ctx.BindReg(r114, &d109)
			}
			ctx.EnsureDesc(&d109)
			ctx.EnsureDesc(&d109)
			var d110 scm.JITValueDesc
			if d109.Loc == scm.LocImm {
				d110 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d109.Imm.Int()))))}
			} else {
				r115 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r115, d109.Reg)
				ctx.W.EmitShlRegImm8(r115, 56)
				ctx.W.EmitShrRegImm8(r115, 56)
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
				ctx.BindReg(r115, &d110)
			}
			ctx.FreeDesc(&d109)
			ctx.EnsureDesc(&d108)
			ctx.EnsureDesc(&d110)
			ctx.EnsureDesc(&d108)
			ctx.EnsureDesc(&d110)
			ctx.EnsureDesc(&d108)
			ctx.EnsureDesc(&d110)
			var d111 scm.JITValueDesc
			if d108.Loc == scm.LocImm && d110.Loc == scm.LocImm {
				d111 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d108.Imm.Int() + d110.Imm.Int())}
			} else if d110.Loc == scm.LocImm && d110.Imm.Int() == 0 {
				r116 := ctx.AllocRegExcept(d108.Reg)
				ctx.W.EmitMovRegReg(r116, d108.Reg)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r116}
				ctx.BindReg(r116, &d111)
			} else if d108.Loc == scm.LocImm && d108.Imm.Int() == 0 {
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d110.Reg}
				ctx.BindReg(d110.Reg, &d111)
			} else if d108.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d110.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d108.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d110.Reg)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d111)
			} else if d110.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d108.Reg)
				ctx.W.EmitMovRegReg(scratch, d108.Reg)
				if d110.Imm.Int() >= -2147483648 && d110.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d110.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d110.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d111)
			} else {
				r117 := ctx.AllocRegExcept(d108.Reg, d110.Reg)
				ctx.W.EmitMovRegReg(r117, d108.Reg)
				ctx.W.EmitAddInt64(r117, d110.Reg)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
				ctx.BindReg(r117, &d111)
			}
			if d111.Loc == scm.LocReg && d108.Loc == scm.LocReg && d111.Reg == d108.Reg {
				ctx.TransferReg(d108.Reg)
				d108.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d108)
			ctx.FreeDesc(&d110)
			ctx.EnsureDesc(&d111)
			var d112 scm.JITValueDesc
			if d111.Loc == scm.LocImm {
				d112 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d111.Imm.Int()) > uint64(64))}
			} else {
				r118 := ctx.AllocRegExcept(d111.Reg)
				ctx.W.EmitCmpRegImm32(d111.Reg, 64)
				ctx.W.EmitSetcc(r118, scm.CcA)
				d112 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r118}
				ctx.BindReg(r118, &d112)
			}
			ctx.FreeDesc(&d111)
			d113 = d112
			ctx.EnsureDesc(&d113)
			if d113.Loc != scm.LocImm && d113.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl39 := ctx.W.ReserveLabel()
			lbl40 := ctx.W.ReserveLabel()
			lbl41 := ctx.W.ReserveLabel()
			if d113.Loc == scm.LocImm {
				if d113.Imm.Bool() {
					ctx.W.MarkLabel(lbl40)
					ctx.W.EmitJmp(lbl39)
				} else {
					ctx.W.MarkLabel(lbl41)
			d114 = d98
			if d114.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d114)
			ctx.EmitStoreToStack(d114, 16)
					ctx.W.EmitJmp(lbl36)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d113.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl40)
				ctx.W.EmitJmp(lbl41)
				ctx.W.MarkLabel(lbl40)
				ctx.W.EmitJmp(lbl39)
				ctx.W.MarkLabel(lbl41)
			d115 = d98
			if d115.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d115)
			ctx.EmitStoreToStack(d115, 16)
				ctx.W.EmitJmp(lbl36)
			}
			ctx.FreeDesc(&d112)
			bbpos_4_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl39)
			ctx.W.ResolveFixups()
			d89 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d93)
			var d116 scm.JITValueDesc
			if d93.Loc == scm.LocImm {
				d116 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d93.Imm.Int() / 64)}
			} else {
				r119 := ctx.AllocRegExcept(d93.Reg)
				ctx.W.EmitMovRegReg(r119, d93.Reg)
				ctx.W.EmitShrRegImm8(r119, 6)
				d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
				ctx.BindReg(r119, &d116)
			}
			if d116.Loc == scm.LocReg && d93.Loc == scm.LocReg && d116.Reg == d93.Reg {
				ctx.TransferReg(d93.Reg)
				d93.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d116)
			ctx.EnsureDesc(&d116)
			var d117 scm.JITValueDesc
			if d116.Loc == scm.LocImm {
				d117 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d116.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d116.Reg)
				ctx.W.EmitMovRegReg(scratch, d116.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d117)
			}
			if d117.Loc == scm.LocReg && d116.Loc == scm.LocReg && d117.Reg == d116.Reg {
				ctx.TransferReg(d116.Reg)
				d116.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d116)
			ctx.EnsureDesc(&d117)
			r120 := ctx.AllocReg()
			ctx.EnsureDesc(&d117)
			ctx.EnsureDesc(&d94)
			if d117.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r120, uint64(d117.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r120, d117.Reg)
				ctx.W.EmitShlRegImm8(r120, 3)
			}
			if d94.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d94.Imm.Int()))
				ctx.W.EmitAddInt64(r120, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r120, d94.Reg)
			}
			r121 := ctx.AllocRegExcept(r120)
			ctx.W.EmitMovRegMem(r121, r120, 0)
			ctx.FreeReg(r120)
			d118 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r121}
			ctx.BindReg(r121, &d118)
			ctx.FreeDesc(&d117)
			ctx.EnsureDesc(&d93)
			var d119 scm.JITValueDesc
			if d93.Loc == scm.LocImm {
				d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d93.Imm.Int() % 64)}
			} else {
				r122 := ctx.AllocRegExcept(d93.Reg)
				ctx.W.EmitMovRegReg(r122, d93.Reg)
				ctx.W.EmitAndRegImm32(r122, 63)
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
				ctx.BindReg(r122, &d119)
			}
			if d119.Loc == scm.LocReg && d93.Loc == scm.LocReg && d119.Reg == d93.Reg {
				ctx.TransferReg(d93.Reg)
				d93.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d93)
			d120 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d119)
			ctx.EnsureDesc(&d120)
			ctx.EnsureDesc(&d119)
			ctx.EnsureDesc(&d120)
			ctx.EnsureDesc(&d119)
			var d121 scm.JITValueDesc
			if d120.Loc == scm.LocImm && d119.Loc == scm.LocImm {
				d121 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d120.Imm.Int() - d119.Imm.Int())}
			} else if d119.Loc == scm.LocImm && d119.Imm.Int() == 0 {
				r123 := ctx.AllocRegExcept(d120.Reg)
				ctx.W.EmitMovRegReg(r123, d120.Reg)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
				ctx.BindReg(r123, &d121)
			} else if d120.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d119.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d120.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d119.Reg)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d121)
			} else if d119.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d120.Reg)
				ctx.W.EmitMovRegReg(scratch, d120.Reg)
				if d119.Imm.Int() >= -2147483648 && d119.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d119.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d119.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d121)
			} else {
				r124 := ctx.AllocRegExcept(d120.Reg, d119.Reg)
				ctx.W.EmitMovRegReg(r124, d120.Reg)
				ctx.W.EmitSubInt64(r124, d119.Reg)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
				ctx.BindReg(r124, &d121)
			}
			if d121.Loc == scm.LocReg && d120.Loc == scm.LocReg && d121.Reg == d120.Reg {
				ctx.TransferReg(d120.Reg)
				d120.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d119)
			ctx.EnsureDesc(&d118)
			ctx.EnsureDesc(&d121)
			var d122 scm.JITValueDesc
			if d118.Loc == scm.LocImm && d121.Loc == scm.LocImm {
				d122 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d118.Imm.Int()) >> uint64(d121.Imm.Int())))}
			} else if d121.Loc == scm.LocImm {
				r125 := ctx.AllocRegExcept(d118.Reg)
				ctx.W.EmitMovRegReg(r125, d118.Reg)
				ctx.W.EmitShrRegImm8(r125, uint8(d121.Imm.Int()))
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
				ctx.BindReg(r125, &d122)
			} else {
				{
					shiftSrc := d118.Reg
					r126 := ctx.AllocRegExcept(d118.Reg)
					ctx.W.EmitMovRegReg(r126, d118.Reg)
					shiftSrc = r126
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d121.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d121.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d121.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d122)
				}
			}
			if d122.Loc == scm.LocReg && d118.Loc == scm.LocReg && d122.Reg == d118.Reg {
				ctx.TransferReg(d118.Reg)
				d118.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d118)
			ctx.FreeDesc(&d121)
			ctx.EnsureDesc(&d98)
			ctx.EnsureDesc(&d122)
			var d123 scm.JITValueDesc
			if d98.Loc == scm.LocImm && d122.Loc == scm.LocImm {
				d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d98.Imm.Int() | d122.Imm.Int())}
			} else if d98.Loc == scm.LocImm && d98.Imm.Int() == 0 {
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d122.Reg}
				ctx.BindReg(d122.Reg, &d123)
			} else if d122.Loc == scm.LocImm && d122.Imm.Int() == 0 {
				r127 := ctx.AllocRegExcept(d98.Reg)
				ctx.W.EmitMovRegReg(r127, d98.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
				ctx.BindReg(r127, &d123)
			} else if d98.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d122.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d98.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d122.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d123)
			} else if d122.Loc == scm.LocImm {
				r128 := ctx.AllocRegExcept(d98.Reg)
				ctx.W.EmitMovRegReg(r128, d98.Reg)
				if d122.Imm.Int() >= -2147483648 && d122.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r128, int32(d122.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d122.Imm.Int()))
					ctx.W.EmitOrInt64(r128, scm.RegR11)
				}
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
				ctx.BindReg(r128, &d123)
			} else {
				r129 := ctx.AllocRegExcept(d98.Reg, d122.Reg)
				ctx.W.EmitMovRegReg(r129, d98.Reg)
				ctx.W.EmitOrInt64(r129, d122.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
				ctx.BindReg(r129, &d123)
			}
			if d123.Loc == scm.LocReg && d98.Loc == scm.LocReg && d123.Reg == d98.Reg {
				ctx.TransferReg(d98.Reg)
				d98.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d122)
			d124 = d123
			if d124.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d124)
			ctx.EmitStoreToStack(d124, 16)
			ctx.W.EmitJmp(lbl36)
			ctx.W.MarkLabel(lbl34)
			d125 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r112}
			ctx.BindReg(r112, &d125)
			ctx.BindReg(r112, &d125)
			if r92 { ctx.UnprotectReg(r93) }
			ctx.EnsureDesc(&d125)
			ctx.EnsureDesc(&d125)
			var d126 scm.JITValueDesc
			if d125.Loc == scm.LocImm {
				d126 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d125.Imm.Int()))))}
			} else {
				r130 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r130, d125.Reg)
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
				ctx.BindReg(r130, &d126)
			}
			ctx.FreeDesc(&d125)
			var d127 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d127 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32)
				r131 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r131, thisptr.Reg, off)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r131}
				ctx.BindReg(r131, &d127)
			}
			ctx.EnsureDesc(&d126)
			ctx.EnsureDesc(&d127)
			ctx.EnsureDesc(&d126)
			ctx.EnsureDesc(&d127)
			ctx.EnsureDesc(&d126)
			ctx.EnsureDesc(&d127)
			var d128 scm.JITValueDesc
			if d126.Loc == scm.LocImm && d127.Loc == scm.LocImm {
				d128 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d126.Imm.Int() + d127.Imm.Int())}
			} else if d127.Loc == scm.LocImm && d127.Imm.Int() == 0 {
				r132 := ctx.AllocRegExcept(d126.Reg)
				ctx.W.EmitMovRegReg(r132, d126.Reg)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
				ctx.BindReg(r132, &d128)
			} else if d126.Loc == scm.LocImm && d126.Imm.Int() == 0 {
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d127.Reg}
				ctx.BindReg(d127.Reg, &d128)
			} else if d126.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d127.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d126.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d127.Reg)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d128)
			} else if d127.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d126.Reg)
				ctx.W.EmitMovRegReg(scratch, d126.Reg)
				if d127.Imm.Int() >= -2147483648 && d127.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d127.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d127.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d128)
			} else {
				r133 := ctx.AllocRegExcept(d126.Reg, d127.Reg)
				ctx.W.EmitMovRegReg(r133, d126.Reg)
				ctx.W.EmitAddInt64(r133, d127.Reg)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
				ctx.BindReg(r133, &d128)
			}
			if d128.Loc == scm.LocReg && d126.Loc == scm.LocReg && d128.Reg == d126.Reg {
				ctx.TransferReg(d126.Reg)
				d126.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d126)
			ctx.FreeDesc(&d127)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d128)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d128)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d128)
			var d130 scm.JITValueDesc
			if d87.Loc == scm.LocImm && d128.Loc == scm.LocImm {
				d130 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d87.Imm.Int() + d128.Imm.Int())}
			} else if d128.Loc == scm.LocImm && d128.Imm.Int() == 0 {
				r134 := ctx.AllocRegExcept(d87.Reg)
				ctx.W.EmitMovRegReg(r134, d87.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r134}
				ctx.BindReg(r134, &d130)
			} else if d87.Loc == scm.LocImm && d87.Imm.Int() == 0 {
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d128.Reg}
				ctx.BindReg(d128.Reg, &d130)
			} else if d87.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d128.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d87.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d128.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d130)
			} else if d128.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d87.Reg)
				ctx.W.EmitMovRegReg(scratch, d87.Reg)
				if d128.Imm.Int() >= -2147483648 && d128.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d128.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d128.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d130)
			} else {
				r135 := ctx.AllocRegExcept(d87.Reg, d128.Reg)
				ctx.W.EmitMovRegReg(r135, d87.Reg)
				ctx.W.EmitAddInt64(r135, d128.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r135}
				ctx.BindReg(r135, &d130)
			}
			if d130.Loc == scm.LocReg && d87.Loc == scm.LocReg && d130.Reg == d87.Reg {
				ctx.TransferReg(d87.Reg)
				d87.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d128)
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d130)
			var d132 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 72
				r136 := ctx.AllocReg()
				r137 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r136, fieldAddr)
				ctx.W.EmitMovRegMem64(r137, fieldAddr+8)
				d132 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r136, Reg2: r137}
				ctx.BindReg(r136, &d132)
				ctx.BindReg(r137, &d132)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 72)
				r138 := ctx.AllocReg()
				r139 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r138, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r139, thisptr.Reg, off+8)
				d132 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r138, Reg2: r139}
				ctx.BindReg(r138, &d132)
				ctx.BindReg(r139, &d132)
			}
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d132)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d130)
			r140 := ctx.AllocReg()
			r141 := ctx.AllocRegExcept(r140)
			ctx.EnsureDesc(&d132)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d130)
			if d132.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r140, uint64(d132.Imm.Int()))
			} else if d132.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r140, d132.Reg)
			} else {
				ctx.W.EmitMovRegReg(r140, d132.Reg)
			}
			if d87.Loc == scm.LocImm {
				if d87.Imm.Int() != 0 {
					if d87.Imm.Int() >= -2147483648 && d87.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r140, int32(d87.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d87.Imm.Int()))
						ctx.W.EmitAddInt64(r140, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r140, d87.Reg)
			}
			if d130.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r141, uint64(d130.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r141, d130.Reg)
			}
			if d87.Loc == scm.LocImm {
				if d87.Imm.Int() >= -2147483648 && d87.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r141, int32(d87.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d87.Imm.Int()))
					ctx.W.EmitSubInt64(r141, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r141, d87.Reg)
			}
			d133 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r140, Reg2: r141}
			ctx.BindReg(r140, &d133)
			ctx.BindReg(r141, &d133)
			ctx.FreeDesc(&d87)
			ctx.FreeDesc(&d130)
			r142 := ctx.AllocReg()
			r143 := ctx.AllocRegExcept(r142)
			d134 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r142, Reg2: r143}
			ctx.BindReg(r142, &d134)
			ctx.BindReg(r143, &d134)
			d135 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d133}, 2)
			ctx.EmitMovPairToResult(&d135, &d134)
			ctx.W.EmitJmp(lbl9)
			bbpos_1_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl10)
			ctx.W.ResolveFixups()
			ctx.EnsureDesc(&d0)
			d136 = d0
			_ = d136
			r144 := d0.Loc == scm.LocReg
			r145 := d0.Reg
			if r144 { ctx.ProtectReg(r145) }
			d137 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			lbl42 := ctx.W.ReserveLabel()
			bbpos_5_0 := int32(-1)
			_ = bbpos_5_0
			bbpos_5_1 := int32(-1)
			_ = bbpos_5_1
			bbpos_5_2 := int32(-1)
			_ = bbpos_5_2
			bbpos_5_3 := int32(-1)
			_ = bbpos_5_3
			bbpos_5_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			d137 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d136)
			ctx.EnsureDesc(&d136)
			var d138 scm.JITValueDesc
			if d136.Loc == scm.LocImm {
				d138 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d136.Imm.Int()))))}
			} else {
				r146 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r146, d136.Reg)
				ctx.W.EmitShlRegImm8(r146, 32)
				ctx.W.EmitShrRegImm8(r146, 32)
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r146}
				ctx.BindReg(r146, &d138)
			}
			var d139 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d139 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r147 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r147, thisptr.Reg, off)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r147}
				ctx.BindReg(r147, &d139)
			}
			ctx.EnsureDesc(&d139)
			ctx.EnsureDesc(&d139)
			var d140 scm.JITValueDesc
			if d139.Loc == scm.LocImm {
				d140 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d139.Imm.Int()))))}
			} else {
				r148 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r148, d139.Reg)
				ctx.W.EmitShlRegImm8(r148, 56)
				ctx.W.EmitShrRegImm8(r148, 56)
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r148}
				ctx.BindReg(r148, &d140)
			}
			ctx.FreeDesc(&d139)
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
				ctx.W.EmitMovRegImm64(scratch, uint64(d138.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d140.Reg)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d141)
			} else if d140.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d138.Reg)
				ctx.W.EmitMovRegReg(scratch, d138.Reg)
				if d140.Imm.Int() >= -2147483648 && d140.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d140.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d140.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d141)
			} else {
				r149 := ctx.AllocRegExcept(d138.Reg, d140.Reg)
				ctx.W.EmitMovRegReg(r149, d138.Reg)
				ctx.W.EmitImulInt64(r149, d140.Reg)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r149}
				ctx.BindReg(r149, &d141)
			}
			if d141.Loc == scm.LocReg && d138.Loc == scm.LocReg && d141.Reg == d138.Reg {
				ctx.TransferReg(d138.Reg)
				d138.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d138)
			ctx.FreeDesc(&d140)
			var d142 scm.JITValueDesc
			r150 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r150, uint64(dataPtr))
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r150, StackOff: int32(sliceLen)}
				ctx.BindReg(r150, &d142)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0)
				ctx.W.EmitMovRegMem(r150, thisptr.Reg, off)
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r150}
				ctx.BindReg(r150, &d142)
			}
			ctx.BindReg(r150, &d142)
			ctx.EnsureDesc(&d141)
			var d143 scm.JITValueDesc
			if d141.Loc == scm.LocImm {
				d143 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d141.Imm.Int() / 64)}
			} else {
				r151 := ctx.AllocRegExcept(d141.Reg)
				ctx.W.EmitMovRegReg(r151, d141.Reg)
				ctx.W.EmitShrRegImm8(r151, 6)
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
				ctx.BindReg(r151, &d143)
			}
			if d143.Loc == scm.LocReg && d141.Loc == scm.LocReg && d143.Reg == d141.Reg {
				ctx.TransferReg(d141.Reg)
				d141.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d143)
			r152 := ctx.AllocReg()
			ctx.EnsureDesc(&d143)
			ctx.EnsureDesc(&d142)
			if d143.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r152, uint64(d143.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r152, d143.Reg)
				ctx.W.EmitShlRegImm8(r152, 3)
			}
			if d142.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d142.Imm.Int()))
				ctx.W.EmitAddInt64(r152, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r152, d142.Reg)
			}
			r153 := ctx.AllocRegExcept(r152)
			ctx.W.EmitMovRegMem(r153, r152, 0)
			ctx.FreeReg(r152)
			d144 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r153}
			ctx.BindReg(r153, &d144)
			ctx.FreeDesc(&d143)
			ctx.EnsureDesc(&d141)
			var d145 scm.JITValueDesc
			if d141.Loc == scm.LocImm {
				d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d141.Imm.Int() % 64)}
			} else {
				r154 := ctx.AllocRegExcept(d141.Reg)
				ctx.W.EmitMovRegReg(r154, d141.Reg)
				ctx.W.EmitAndRegImm32(r154, 63)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r154}
				ctx.BindReg(r154, &d145)
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
				r155 := ctx.AllocRegExcept(d144.Reg)
				ctx.W.EmitMovRegReg(r155, d144.Reg)
				ctx.W.EmitShlRegImm8(r155, uint8(d145.Imm.Int()))
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r155}
				ctx.BindReg(r155, &d146)
			} else {
				{
					shiftSrc := d144.Reg
					r156 := ctx.AllocRegExcept(d144.Reg)
					ctx.W.EmitMovRegReg(r156, d144.Reg)
					shiftSrc = r156
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d145.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d145.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d145.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
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
			var d147 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d147 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25)
				r157 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r157, thisptr.Reg, off)
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r157}
				ctx.BindReg(r157, &d147)
			}
			d148 = d147
			ctx.EnsureDesc(&d148)
			if d148.Loc != scm.LocImm && d148.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl43 := ctx.W.ReserveLabel()
			lbl44 := ctx.W.ReserveLabel()
			lbl45 := ctx.W.ReserveLabel()
			lbl46 := ctx.W.ReserveLabel()
			if d148.Loc == scm.LocImm {
				if d148.Imm.Bool() {
					ctx.W.MarkLabel(lbl45)
					ctx.W.EmitJmp(lbl43)
				} else {
					ctx.W.MarkLabel(lbl46)
			d149 = d146
			if d149.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d149)
			ctx.EmitStoreToStack(d149, 24)
					ctx.W.EmitJmp(lbl44)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d148.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl45)
				ctx.W.EmitJmp(lbl46)
				ctx.W.MarkLabel(lbl45)
				ctx.W.EmitJmp(lbl43)
				ctx.W.MarkLabel(lbl46)
			d150 = d146
			if d150.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d150)
			ctx.EmitStoreToStack(d150, 24)
				ctx.W.EmitJmp(lbl44)
			}
			ctx.FreeDesc(&d147)
			bbpos_5_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl44)
			ctx.W.ResolveFixups()
			d137 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			var d151 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r158 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r158, thisptr.Reg, off)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r158}
				ctx.BindReg(r158, &d151)
			}
			ctx.EnsureDesc(&d151)
			ctx.EnsureDesc(&d151)
			var d152 scm.JITValueDesc
			if d151.Loc == scm.LocImm {
				d152 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d151.Imm.Int()))))}
			} else {
				r159 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r159, d151.Reg)
				ctx.W.EmitShlRegImm8(r159, 56)
				ctx.W.EmitShrRegImm8(r159, 56)
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
				ctx.BindReg(r159, &d152)
			}
			ctx.FreeDesc(&d151)
			d153 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d152)
			ctx.EnsureDesc(&d153)
			ctx.EnsureDesc(&d152)
			ctx.EnsureDesc(&d153)
			ctx.EnsureDesc(&d152)
			var d154 scm.JITValueDesc
			if d153.Loc == scm.LocImm && d152.Loc == scm.LocImm {
				d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d153.Imm.Int() - d152.Imm.Int())}
			} else if d152.Loc == scm.LocImm && d152.Imm.Int() == 0 {
				r160 := ctx.AllocRegExcept(d153.Reg)
				ctx.W.EmitMovRegReg(r160, d153.Reg)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r160}
				ctx.BindReg(r160, &d154)
			} else if d153.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d152.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d153.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d152.Reg)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d154)
			} else if d152.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d153.Reg)
				ctx.W.EmitMovRegReg(scratch, d153.Reg)
				if d152.Imm.Int() >= -2147483648 && d152.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d152.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d152.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d154)
			} else {
				r161 := ctx.AllocRegExcept(d153.Reg, d152.Reg)
				ctx.W.EmitMovRegReg(r161, d153.Reg)
				ctx.W.EmitSubInt64(r161, d152.Reg)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r161}
				ctx.BindReg(r161, &d154)
			}
			if d154.Loc == scm.LocReg && d153.Loc == scm.LocReg && d154.Reg == d153.Reg {
				ctx.TransferReg(d153.Reg)
				d153.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d152)
			ctx.EnsureDesc(&d137)
			ctx.EnsureDesc(&d154)
			var d155 scm.JITValueDesc
			if d137.Loc == scm.LocImm && d154.Loc == scm.LocImm {
				d155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d137.Imm.Int()) >> uint64(d154.Imm.Int())))}
			} else if d154.Loc == scm.LocImm {
				r162 := ctx.AllocRegExcept(d137.Reg)
				ctx.W.EmitMovRegReg(r162, d137.Reg)
				ctx.W.EmitShrRegImm8(r162, uint8(d154.Imm.Int()))
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r162}
				ctx.BindReg(r162, &d155)
			} else {
				{
					shiftSrc := d137.Reg
					r163 := ctx.AllocRegExcept(d137.Reg)
					ctx.W.EmitMovRegReg(r163, d137.Reg)
					shiftSrc = r163
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d154.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d154.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d154.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d155)
				}
			}
			if d155.Loc == scm.LocReg && d137.Loc == scm.LocReg && d155.Reg == d137.Reg {
				ctx.TransferReg(d137.Reg)
				d137.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d137)
			ctx.FreeDesc(&d154)
			r164 := ctx.AllocReg()
			ctx.EnsureDesc(&d155)
			ctx.EnsureDesc(&d155)
			if d155.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r164, d155)
			}
			ctx.W.EmitJmp(lbl42)
			bbpos_5_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl43)
			ctx.W.ResolveFixups()
			d137 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d141)
			var d156 scm.JITValueDesc
			if d141.Loc == scm.LocImm {
				d156 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d141.Imm.Int() % 64)}
			} else {
				r165 := ctx.AllocRegExcept(d141.Reg)
				ctx.W.EmitMovRegReg(r165, d141.Reg)
				ctx.W.EmitAndRegImm32(r165, 63)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r165}
				ctx.BindReg(r165, &d156)
			}
			if d156.Loc == scm.LocReg && d141.Loc == scm.LocReg && d156.Reg == d141.Reg {
				ctx.TransferReg(d141.Reg)
				d141.Loc = scm.LocNone
			}
			var d157 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d157 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r166 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r166, thisptr.Reg, off)
				d157 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r166}
				ctx.BindReg(r166, &d157)
			}
			ctx.EnsureDesc(&d157)
			ctx.EnsureDesc(&d157)
			var d158 scm.JITValueDesc
			if d157.Loc == scm.LocImm {
				d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d157.Imm.Int()))))}
			} else {
				r167 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r167, d157.Reg)
				ctx.W.EmitShlRegImm8(r167, 56)
				ctx.W.EmitShrRegImm8(r167, 56)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r167}
				ctx.BindReg(r167, &d158)
			}
			ctx.FreeDesc(&d157)
			ctx.EnsureDesc(&d156)
			ctx.EnsureDesc(&d158)
			ctx.EnsureDesc(&d156)
			ctx.EnsureDesc(&d158)
			ctx.EnsureDesc(&d156)
			ctx.EnsureDesc(&d158)
			var d159 scm.JITValueDesc
			if d156.Loc == scm.LocImm && d158.Loc == scm.LocImm {
				d159 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d156.Imm.Int() + d158.Imm.Int())}
			} else if d158.Loc == scm.LocImm && d158.Imm.Int() == 0 {
				r168 := ctx.AllocRegExcept(d156.Reg)
				ctx.W.EmitMovRegReg(r168, d156.Reg)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r168}
				ctx.BindReg(r168, &d159)
			} else if d156.Loc == scm.LocImm && d156.Imm.Int() == 0 {
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d158.Reg}
				ctx.BindReg(d158.Reg, &d159)
			} else if d156.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d158.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d156.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d158.Reg)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d159)
			} else if d158.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d156.Reg)
				ctx.W.EmitMovRegReg(scratch, d156.Reg)
				if d158.Imm.Int() >= -2147483648 && d158.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d158.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d158.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d159)
			} else {
				r169 := ctx.AllocRegExcept(d156.Reg, d158.Reg)
				ctx.W.EmitMovRegReg(r169, d156.Reg)
				ctx.W.EmitAddInt64(r169, d158.Reg)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r169}
				ctx.BindReg(r169, &d159)
			}
			if d159.Loc == scm.LocReg && d156.Loc == scm.LocReg && d159.Reg == d156.Reg {
				ctx.TransferReg(d156.Reg)
				d156.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d156)
			ctx.FreeDesc(&d158)
			ctx.EnsureDesc(&d159)
			var d160 scm.JITValueDesc
			if d159.Loc == scm.LocImm {
				d160 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d159.Imm.Int()) > uint64(64))}
			} else {
				r170 := ctx.AllocRegExcept(d159.Reg)
				ctx.W.EmitCmpRegImm32(d159.Reg, 64)
				ctx.W.EmitSetcc(r170, scm.CcA)
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r170}
				ctx.BindReg(r170, &d160)
			}
			ctx.FreeDesc(&d159)
			d161 = d160
			ctx.EnsureDesc(&d161)
			if d161.Loc != scm.LocImm && d161.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl47 := ctx.W.ReserveLabel()
			lbl48 := ctx.W.ReserveLabel()
			lbl49 := ctx.W.ReserveLabel()
			if d161.Loc == scm.LocImm {
				if d161.Imm.Bool() {
					ctx.W.MarkLabel(lbl48)
					ctx.W.EmitJmp(lbl47)
				} else {
					ctx.W.MarkLabel(lbl49)
			d162 = d146
			if d162.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d162)
			ctx.EmitStoreToStack(d162, 24)
					ctx.W.EmitJmp(lbl44)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d161.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl48)
				ctx.W.EmitJmp(lbl49)
				ctx.W.MarkLabel(lbl48)
				ctx.W.EmitJmp(lbl47)
				ctx.W.MarkLabel(lbl49)
			d163 = d146
			if d163.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d163)
			ctx.EmitStoreToStack(d163, 24)
				ctx.W.EmitJmp(lbl44)
			}
			ctx.FreeDesc(&d160)
			bbpos_5_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl47)
			ctx.W.ResolveFixups()
			d137 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d141)
			var d164 scm.JITValueDesc
			if d141.Loc == scm.LocImm {
				d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d141.Imm.Int() / 64)}
			} else {
				r171 := ctx.AllocRegExcept(d141.Reg)
				ctx.W.EmitMovRegReg(r171, d141.Reg)
				ctx.W.EmitShrRegImm8(r171, 6)
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r171}
				ctx.BindReg(r171, &d164)
			}
			if d164.Loc == scm.LocReg && d141.Loc == scm.LocReg && d164.Reg == d141.Reg {
				ctx.TransferReg(d141.Reg)
				d141.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d164)
			ctx.EnsureDesc(&d164)
			var d165 scm.JITValueDesc
			if d164.Loc == scm.LocImm {
				d165 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d164.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d164.Reg)
				ctx.W.EmitMovRegReg(scratch, d164.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d165)
			}
			if d165.Loc == scm.LocReg && d164.Loc == scm.LocReg && d165.Reg == d164.Reg {
				ctx.TransferReg(d164.Reg)
				d164.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d164)
			ctx.EnsureDesc(&d165)
			r172 := ctx.AllocReg()
			ctx.EnsureDesc(&d165)
			ctx.EnsureDesc(&d142)
			if d165.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r172, uint64(d165.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r172, d165.Reg)
				ctx.W.EmitShlRegImm8(r172, 3)
			}
			if d142.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d142.Imm.Int()))
				ctx.W.EmitAddInt64(r172, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r172, d142.Reg)
			}
			r173 := ctx.AllocRegExcept(r172)
			ctx.W.EmitMovRegMem(r173, r172, 0)
			ctx.FreeReg(r172)
			d166 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r173}
			ctx.BindReg(r173, &d166)
			ctx.FreeDesc(&d165)
			ctx.EnsureDesc(&d141)
			var d167 scm.JITValueDesc
			if d141.Loc == scm.LocImm {
				d167 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d141.Imm.Int() % 64)}
			} else {
				r174 := ctx.AllocRegExcept(d141.Reg)
				ctx.W.EmitMovRegReg(r174, d141.Reg)
				ctx.W.EmitAndRegImm32(r174, 63)
				d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r174}
				ctx.BindReg(r174, &d167)
			}
			if d167.Loc == scm.LocReg && d141.Loc == scm.LocReg && d167.Reg == d141.Reg {
				ctx.TransferReg(d141.Reg)
				d141.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d141)
			d168 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d167)
			ctx.EnsureDesc(&d168)
			ctx.EnsureDesc(&d167)
			ctx.EnsureDesc(&d168)
			ctx.EnsureDesc(&d167)
			var d169 scm.JITValueDesc
			if d168.Loc == scm.LocImm && d167.Loc == scm.LocImm {
				d169 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d168.Imm.Int() - d167.Imm.Int())}
			} else if d167.Loc == scm.LocImm && d167.Imm.Int() == 0 {
				r175 := ctx.AllocRegExcept(d168.Reg)
				ctx.W.EmitMovRegReg(r175, d168.Reg)
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r175}
				ctx.BindReg(r175, &d169)
			} else if d168.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d167.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d168.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d167.Reg)
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d169)
			} else if d167.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d168.Reg)
				ctx.W.EmitMovRegReg(scratch, d168.Reg)
				if d167.Imm.Int() >= -2147483648 && d167.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d167.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d167.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d169)
			} else {
				r176 := ctx.AllocRegExcept(d168.Reg, d167.Reg)
				ctx.W.EmitMovRegReg(r176, d168.Reg)
				ctx.W.EmitSubInt64(r176, d167.Reg)
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r176}
				ctx.BindReg(r176, &d169)
			}
			if d169.Loc == scm.LocReg && d168.Loc == scm.LocReg && d169.Reg == d168.Reg {
				ctx.TransferReg(d168.Reg)
				d168.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d167)
			ctx.EnsureDesc(&d166)
			ctx.EnsureDesc(&d169)
			var d170 scm.JITValueDesc
			if d166.Loc == scm.LocImm && d169.Loc == scm.LocImm {
				d170 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d166.Imm.Int()) >> uint64(d169.Imm.Int())))}
			} else if d169.Loc == scm.LocImm {
				r177 := ctx.AllocRegExcept(d166.Reg)
				ctx.W.EmitMovRegReg(r177, d166.Reg)
				ctx.W.EmitShrRegImm8(r177, uint8(d169.Imm.Int()))
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r177}
				ctx.BindReg(r177, &d170)
			} else {
				{
					shiftSrc := d166.Reg
					r178 := ctx.AllocRegExcept(d166.Reg)
					ctx.W.EmitMovRegReg(r178, d166.Reg)
					shiftSrc = r178
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d169.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d169.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d169.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d170)
				}
			}
			if d170.Loc == scm.LocReg && d166.Loc == scm.LocReg && d170.Reg == d166.Reg {
				ctx.TransferReg(d166.Reg)
				d166.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d166)
			ctx.FreeDesc(&d169)
			ctx.EnsureDesc(&d146)
			ctx.EnsureDesc(&d170)
			var d171 scm.JITValueDesc
			if d146.Loc == scm.LocImm && d170.Loc == scm.LocImm {
				d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d146.Imm.Int() | d170.Imm.Int())}
			} else if d146.Loc == scm.LocImm && d146.Imm.Int() == 0 {
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d170.Reg}
				ctx.BindReg(d170.Reg, &d171)
			} else if d170.Loc == scm.LocImm && d170.Imm.Int() == 0 {
				r179 := ctx.AllocRegExcept(d146.Reg)
				ctx.W.EmitMovRegReg(r179, d146.Reg)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r179}
				ctx.BindReg(r179, &d171)
			} else if d146.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d170.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d146.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d170.Reg)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d171)
			} else if d170.Loc == scm.LocImm {
				r180 := ctx.AllocRegExcept(d146.Reg)
				ctx.W.EmitMovRegReg(r180, d146.Reg)
				if d170.Imm.Int() >= -2147483648 && d170.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r180, int32(d170.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d170.Imm.Int()))
					ctx.W.EmitOrInt64(r180, scm.RegR11)
				}
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r180}
				ctx.BindReg(r180, &d171)
			} else {
				r181 := ctx.AllocRegExcept(d146.Reg, d170.Reg)
				ctx.W.EmitMovRegReg(r181, d146.Reg)
				ctx.W.EmitOrInt64(r181, d170.Reg)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r181}
				ctx.BindReg(r181, &d171)
			}
			if d171.Loc == scm.LocReg && d146.Loc == scm.LocReg && d171.Reg == d146.Reg {
				ctx.TransferReg(d146.Reg)
				d146.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d170)
			d172 = d171
			if d172.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d172)
			ctx.EmitStoreToStack(d172, 24)
			ctx.W.EmitJmp(lbl44)
			ctx.W.MarkLabel(lbl42)
			d173 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r164}
			ctx.BindReg(r164, &d173)
			ctx.BindReg(r164, &d173)
			if r144 { ctx.UnprotectReg(r145) }
			ctx.EnsureDesc(&d173)
			ctx.EnsureDesc(&d173)
			var d174 scm.JITValueDesc
			if d173.Loc == scm.LocImm {
				d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d173.Imm.Int()))))}
			} else {
				r182 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r182, d173.Reg)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r182}
				ctx.BindReg(r182, &d174)
			}
			ctx.FreeDesc(&d173)
			var d175 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d175 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32)
				r183 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r183, thisptr.Reg, off)
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r183}
				ctx.BindReg(r183, &d175)
			}
			ctx.EnsureDesc(&d174)
			ctx.EnsureDesc(&d175)
			ctx.EnsureDesc(&d174)
			ctx.EnsureDesc(&d175)
			ctx.EnsureDesc(&d174)
			ctx.EnsureDesc(&d175)
			var d176 scm.JITValueDesc
			if d174.Loc == scm.LocImm && d175.Loc == scm.LocImm {
				d176 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d174.Imm.Int() + d175.Imm.Int())}
			} else if d175.Loc == scm.LocImm && d175.Imm.Int() == 0 {
				r184 := ctx.AllocRegExcept(d174.Reg)
				ctx.W.EmitMovRegReg(r184, d174.Reg)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r184}
				ctx.BindReg(r184, &d176)
			} else if d174.Loc == scm.LocImm && d174.Imm.Int() == 0 {
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d175.Reg}
				ctx.BindReg(d175.Reg, &d176)
			} else if d174.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d175.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d174.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d175.Reg)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d176)
			} else if d175.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d174.Reg)
				ctx.W.EmitMovRegReg(scratch, d174.Reg)
				if d175.Imm.Int() >= -2147483648 && d175.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d175.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d175.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d176)
			} else {
				r185 := ctx.AllocRegExcept(d174.Reg, d175.Reg)
				ctx.W.EmitMovRegReg(r185, d174.Reg)
				ctx.W.EmitAddInt64(r185, d175.Reg)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r185}
				ctx.BindReg(r185, &d176)
			}
			if d176.Loc == scm.LocReg && d174.Loc == scm.LocReg && d176.Reg == d174.Reg {
				ctx.TransferReg(d174.Reg)
				d174.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d174)
			ctx.FreeDesc(&d175)
			ctx.EnsureDesc(&d176)
			ctx.EnsureDesc(&d176)
			var d177 scm.JITValueDesc
			if d176.Loc == scm.LocImm {
				d177 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d176.Imm.Int()))))}
			} else {
				r186 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r186, d176.Reg)
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r186}
				ctx.BindReg(r186, &d177)
			}
			ctx.FreeDesc(&d176)
			var d178 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d178 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 56)
				r187 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r187, thisptr.Reg, off)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r187}
				ctx.BindReg(r187, &d178)
			}
			d179 = d178
			ctx.EnsureDesc(&d179)
			if d179.Loc != scm.LocImm && d179.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl50 := ctx.W.ReserveLabel()
			lbl51 := ctx.W.ReserveLabel()
			lbl52 := ctx.W.ReserveLabel()
			lbl53 := ctx.W.ReserveLabel()
			if d179.Loc == scm.LocImm {
				if d179.Imm.Bool() {
					ctx.W.MarkLabel(lbl52)
					ctx.W.EmitJmp(lbl50)
				} else {
					ctx.W.MarkLabel(lbl53)
					ctx.W.EmitJmp(lbl51)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d179.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl52)
				ctx.W.EmitJmp(lbl53)
				ctx.W.MarkLabel(lbl52)
				ctx.W.EmitJmp(lbl50)
				ctx.W.MarkLabel(lbl53)
				ctx.W.EmitJmp(lbl51)
			}
			ctx.FreeDesc(&d178)
			bbpos_1_4 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl51)
			ctx.W.ResolveFixups()
			ctx.EnsureDesc(&d0)
			d180 = d0
			_ = d180
			r188 := d0.Loc == scm.LocReg
			r189 := d0.Reg
			if r188 { ctx.ProtectReg(r189) }
			d181 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			lbl54 := ctx.W.ReserveLabel()
			bbpos_6_0 := int32(-1)
			_ = bbpos_6_0
			bbpos_6_1 := int32(-1)
			_ = bbpos_6_1
			bbpos_6_2 := int32(-1)
			_ = bbpos_6_2
			bbpos_6_3 := int32(-1)
			_ = bbpos_6_3
			bbpos_6_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			d181 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d180)
			ctx.EnsureDesc(&d180)
			var d182 scm.JITValueDesc
			if d180.Loc == scm.LocImm {
				d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d180.Imm.Int()))))}
			} else {
				r190 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r190, d180.Reg)
				ctx.W.EmitShlRegImm8(r190, 32)
				ctx.W.EmitShrRegImm8(r190, 32)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r190}
				ctx.BindReg(r190, &d182)
			}
			var d183 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r191 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r191, thisptr.Reg, off)
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r191}
				ctx.BindReg(r191, &d183)
			}
			ctx.EnsureDesc(&d183)
			ctx.EnsureDesc(&d183)
			var d184 scm.JITValueDesc
			if d183.Loc == scm.LocImm {
				d184 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d183.Imm.Int()))))}
			} else {
				r192 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r192, d183.Reg)
				ctx.W.EmitShlRegImm8(r192, 56)
				ctx.W.EmitShrRegImm8(r192, 56)
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r192}
				ctx.BindReg(r192, &d184)
			}
			ctx.FreeDesc(&d183)
			ctx.EnsureDesc(&d182)
			ctx.EnsureDesc(&d184)
			ctx.EnsureDesc(&d182)
			ctx.EnsureDesc(&d184)
			ctx.EnsureDesc(&d182)
			ctx.EnsureDesc(&d184)
			var d185 scm.JITValueDesc
			if d182.Loc == scm.LocImm && d184.Loc == scm.LocImm {
				d185 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d182.Imm.Int() * d184.Imm.Int())}
			} else if d182.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d184.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d182.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d184.Reg)
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d185)
			} else if d184.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d182.Reg)
				ctx.W.EmitMovRegReg(scratch, d182.Reg)
				if d184.Imm.Int() >= -2147483648 && d184.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d184.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d184.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d185)
			} else {
				r193 := ctx.AllocRegExcept(d182.Reg, d184.Reg)
				ctx.W.EmitMovRegReg(r193, d182.Reg)
				ctx.W.EmitImulInt64(r193, d184.Reg)
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r193}
				ctx.BindReg(r193, &d185)
			}
			if d185.Loc == scm.LocReg && d182.Loc == scm.LocReg && d185.Reg == d182.Reg {
				ctx.TransferReg(d182.Reg)
				d182.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d182)
			ctx.FreeDesc(&d184)
			var d186 scm.JITValueDesc
			r194 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r194, uint64(dataPtr))
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r194, StackOff: int32(sliceLen)}
				ctx.BindReg(r194, &d186)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0)
				ctx.W.EmitMovRegMem(r194, thisptr.Reg, off)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r194}
				ctx.BindReg(r194, &d186)
			}
			ctx.BindReg(r194, &d186)
			ctx.EnsureDesc(&d185)
			var d187 scm.JITValueDesc
			if d185.Loc == scm.LocImm {
				d187 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d185.Imm.Int() / 64)}
			} else {
				r195 := ctx.AllocRegExcept(d185.Reg)
				ctx.W.EmitMovRegReg(r195, d185.Reg)
				ctx.W.EmitShrRegImm8(r195, 6)
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r195}
				ctx.BindReg(r195, &d187)
			}
			if d187.Loc == scm.LocReg && d185.Loc == scm.LocReg && d187.Reg == d185.Reg {
				ctx.TransferReg(d185.Reg)
				d185.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d187)
			r196 := ctx.AllocReg()
			ctx.EnsureDesc(&d187)
			ctx.EnsureDesc(&d186)
			if d187.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r196, uint64(d187.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r196, d187.Reg)
				ctx.W.EmitShlRegImm8(r196, 3)
			}
			if d186.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d186.Imm.Int()))
				ctx.W.EmitAddInt64(r196, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r196, d186.Reg)
			}
			r197 := ctx.AllocRegExcept(r196)
			ctx.W.EmitMovRegMem(r197, r196, 0)
			ctx.FreeReg(r196)
			d188 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r197}
			ctx.BindReg(r197, &d188)
			ctx.FreeDesc(&d187)
			ctx.EnsureDesc(&d185)
			var d189 scm.JITValueDesc
			if d185.Loc == scm.LocImm {
				d189 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d185.Imm.Int() % 64)}
			} else {
				r198 := ctx.AllocRegExcept(d185.Reg)
				ctx.W.EmitMovRegReg(r198, d185.Reg)
				ctx.W.EmitAndRegImm32(r198, 63)
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r198}
				ctx.BindReg(r198, &d189)
			}
			if d189.Loc == scm.LocReg && d185.Loc == scm.LocReg && d189.Reg == d185.Reg {
				ctx.TransferReg(d185.Reg)
				d185.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d188)
			ctx.EnsureDesc(&d189)
			var d190 scm.JITValueDesc
			if d188.Loc == scm.LocImm && d189.Loc == scm.LocImm {
				d190 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d188.Imm.Int()) << uint64(d189.Imm.Int())))}
			} else if d189.Loc == scm.LocImm {
				r199 := ctx.AllocRegExcept(d188.Reg)
				ctx.W.EmitMovRegReg(r199, d188.Reg)
				ctx.W.EmitShlRegImm8(r199, uint8(d189.Imm.Int()))
				d190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r199}
				ctx.BindReg(r199, &d190)
			} else {
				{
					shiftSrc := d188.Reg
					r200 := ctx.AllocRegExcept(d188.Reg)
					ctx.W.EmitMovRegReg(r200, d188.Reg)
					shiftSrc = r200
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d189.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d189.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d189.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d190)
				}
			}
			if d190.Loc == scm.LocReg && d188.Loc == scm.LocReg && d190.Reg == d188.Reg {
				ctx.TransferReg(d188.Reg)
				d188.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d188)
			ctx.FreeDesc(&d189)
			var d191 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d191 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25)
				r201 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r201, thisptr.Reg, off)
				d191 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r201}
				ctx.BindReg(r201, &d191)
			}
			d192 = d191
			ctx.EnsureDesc(&d192)
			if d192.Loc != scm.LocImm && d192.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl55 := ctx.W.ReserveLabel()
			lbl56 := ctx.W.ReserveLabel()
			lbl57 := ctx.W.ReserveLabel()
			lbl58 := ctx.W.ReserveLabel()
			if d192.Loc == scm.LocImm {
				if d192.Imm.Bool() {
					ctx.W.MarkLabel(lbl57)
					ctx.W.EmitJmp(lbl55)
				} else {
					ctx.W.MarkLabel(lbl58)
			d193 = d190
			if d193.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d193)
			ctx.EmitStoreToStack(d193, 32)
					ctx.W.EmitJmp(lbl56)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d192.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl57)
				ctx.W.EmitJmp(lbl58)
				ctx.W.MarkLabel(lbl57)
				ctx.W.EmitJmp(lbl55)
				ctx.W.MarkLabel(lbl58)
			d194 = d190
			if d194.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d194)
			ctx.EmitStoreToStack(d194, 32)
				ctx.W.EmitJmp(lbl56)
			}
			ctx.FreeDesc(&d191)
			bbpos_6_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl56)
			ctx.W.ResolveFixups()
			d181 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			var d195 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d195 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r202 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r202, thisptr.Reg, off)
				d195 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r202}
				ctx.BindReg(r202, &d195)
			}
			ctx.EnsureDesc(&d195)
			ctx.EnsureDesc(&d195)
			var d196 scm.JITValueDesc
			if d195.Loc == scm.LocImm {
				d196 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d195.Imm.Int()))))}
			} else {
				r203 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r203, d195.Reg)
				ctx.W.EmitShlRegImm8(r203, 56)
				ctx.W.EmitShrRegImm8(r203, 56)
				d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r203}
				ctx.BindReg(r203, &d196)
			}
			ctx.FreeDesc(&d195)
			d197 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d196)
			ctx.EnsureDesc(&d197)
			ctx.EnsureDesc(&d196)
			ctx.EnsureDesc(&d197)
			ctx.EnsureDesc(&d196)
			var d198 scm.JITValueDesc
			if d197.Loc == scm.LocImm && d196.Loc == scm.LocImm {
				d198 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d197.Imm.Int() - d196.Imm.Int())}
			} else if d196.Loc == scm.LocImm && d196.Imm.Int() == 0 {
				r204 := ctx.AllocRegExcept(d197.Reg)
				ctx.W.EmitMovRegReg(r204, d197.Reg)
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r204}
				ctx.BindReg(r204, &d198)
			} else if d197.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d196.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d197.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d196.Reg)
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d198)
			} else if d196.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d197.Reg)
				ctx.W.EmitMovRegReg(scratch, d197.Reg)
				if d196.Imm.Int() >= -2147483648 && d196.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d196.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d196.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d198)
			} else {
				r205 := ctx.AllocRegExcept(d197.Reg, d196.Reg)
				ctx.W.EmitMovRegReg(r205, d197.Reg)
				ctx.W.EmitSubInt64(r205, d196.Reg)
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r205}
				ctx.BindReg(r205, &d198)
			}
			if d198.Loc == scm.LocReg && d197.Loc == scm.LocReg && d198.Reg == d197.Reg {
				ctx.TransferReg(d197.Reg)
				d197.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d196)
			ctx.EnsureDesc(&d181)
			ctx.EnsureDesc(&d198)
			var d199 scm.JITValueDesc
			if d181.Loc == scm.LocImm && d198.Loc == scm.LocImm {
				d199 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d181.Imm.Int()) >> uint64(d198.Imm.Int())))}
			} else if d198.Loc == scm.LocImm {
				r206 := ctx.AllocRegExcept(d181.Reg)
				ctx.W.EmitMovRegReg(r206, d181.Reg)
				ctx.W.EmitShrRegImm8(r206, uint8(d198.Imm.Int()))
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r206}
				ctx.BindReg(r206, &d199)
			} else {
				{
					shiftSrc := d181.Reg
					r207 := ctx.AllocRegExcept(d181.Reg)
					ctx.W.EmitMovRegReg(r207, d181.Reg)
					shiftSrc = r207
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d198.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d198.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d198.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d199)
				}
			}
			if d199.Loc == scm.LocReg && d181.Loc == scm.LocReg && d199.Reg == d181.Reg {
				ctx.TransferReg(d181.Reg)
				d181.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d181)
			ctx.FreeDesc(&d198)
			r208 := ctx.AllocReg()
			ctx.EnsureDesc(&d199)
			ctx.EnsureDesc(&d199)
			if d199.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r208, d199)
			}
			ctx.W.EmitJmp(lbl54)
			bbpos_6_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl55)
			ctx.W.ResolveFixups()
			d181 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d185)
			var d200 scm.JITValueDesc
			if d185.Loc == scm.LocImm {
				d200 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d185.Imm.Int() % 64)}
			} else {
				r209 := ctx.AllocRegExcept(d185.Reg)
				ctx.W.EmitMovRegReg(r209, d185.Reg)
				ctx.W.EmitAndRegImm32(r209, 63)
				d200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r209}
				ctx.BindReg(r209, &d200)
			}
			if d200.Loc == scm.LocReg && d185.Loc == scm.LocReg && d200.Reg == d185.Reg {
				ctx.TransferReg(d185.Reg)
				d185.Loc = scm.LocNone
			}
			var d201 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d201 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r210 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r210, thisptr.Reg, off)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r210}
				ctx.BindReg(r210, &d201)
			}
			ctx.EnsureDesc(&d201)
			ctx.EnsureDesc(&d201)
			var d202 scm.JITValueDesc
			if d201.Loc == scm.LocImm {
				d202 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d201.Imm.Int()))))}
			} else {
				r211 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r211, d201.Reg)
				ctx.W.EmitShlRegImm8(r211, 56)
				ctx.W.EmitShrRegImm8(r211, 56)
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r211}
				ctx.BindReg(r211, &d202)
			}
			ctx.FreeDesc(&d201)
			ctx.EnsureDesc(&d200)
			ctx.EnsureDesc(&d202)
			ctx.EnsureDesc(&d200)
			ctx.EnsureDesc(&d202)
			ctx.EnsureDesc(&d200)
			ctx.EnsureDesc(&d202)
			var d203 scm.JITValueDesc
			if d200.Loc == scm.LocImm && d202.Loc == scm.LocImm {
				d203 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d200.Imm.Int() + d202.Imm.Int())}
			} else if d202.Loc == scm.LocImm && d202.Imm.Int() == 0 {
				r212 := ctx.AllocRegExcept(d200.Reg)
				ctx.W.EmitMovRegReg(r212, d200.Reg)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r212}
				ctx.BindReg(r212, &d203)
			} else if d200.Loc == scm.LocImm && d200.Imm.Int() == 0 {
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d202.Reg}
				ctx.BindReg(d202.Reg, &d203)
			} else if d200.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d202.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d200.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d202.Reg)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d203)
			} else if d202.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d200.Reg)
				ctx.W.EmitMovRegReg(scratch, d200.Reg)
				if d202.Imm.Int() >= -2147483648 && d202.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d202.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d202.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d203)
			} else {
				r213 := ctx.AllocRegExcept(d200.Reg, d202.Reg)
				ctx.W.EmitMovRegReg(r213, d200.Reg)
				ctx.W.EmitAddInt64(r213, d202.Reg)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r213}
				ctx.BindReg(r213, &d203)
			}
			if d203.Loc == scm.LocReg && d200.Loc == scm.LocReg && d203.Reg == d200.Reg {
				ctx.TransferReg(d200.Reg)
				d200.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d200)
			ctx.FreeDesc(&d202)
			ctx.EnsureDesc(&d203)
			var d204 scm.JITValueDesc
			if d203.Loc == scm.LocImm {
				d204 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d203.Imm.Int()) > uint64(64))}
			} else {
				r214 := ctx.AllocRegExcept(d203.Reg)
				ctx.W.EmitCmpRegImm32(d203.Reg, 64)
				ctx.W.EmitSetcc(r214, scm.CcA)
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r214}
				ctx.BindReg(r214, &d204)
			}
			ctx.FreeDesc(&d203)
			d205 = d204
			ctx.EnsureDesc(&d205)
			if d205.Loc != scm.LocImm && d205.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl59 := ctx.W.ReserveLabel()
			lbl60 := ctx.W.ReserveLabel()
			lbl61 := ctx.W.ReserveLabel()
			if d205.Loc == scm.LocImm {
				if d205.Imm.Bool() {
					ctx.W.MarkLabel(lbl60)
					ctx.W.EmitJmp(lbl59)
				} else {
					ctx.W.MarkLabel(lbl61)
			d206 = d190
			if d206.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d206)
			ctx.EmitStoreToStack(d206, 32)
					ctx.W.EmitJmp(lbl56)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d205.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl60)
				ctx.W.EmitJmp(lbl61)
				ctx.W.MarkLabel(lbl60)
				ctx.W.EmitJmp(lbl59)
				ctx.W.MarkLabel(lbl61)
			d207 = d190
			if d207.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d207)
			ctx.EmitStoreToStack(d207, 32)
				ctx.W.EmitJmp(lbl56)
			}
			ctx.FreeDesc(&d204)
			bbpos_6_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl59)
			ctx.W.ResolveFixups()
			d181 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d185)
			var d208 scm.JITValueDesc
			if d185.Loc == scm.LocImm {
				d208 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d185.Imm.Int() / 64)}
			} else {
				r215 := ctx.AllocRegExcept(d185.Reg)
				ctx.W.EmitMovRegReg(r215, d185.Reg)
				ctx.W.EmitShrRegImm8(r215, 6)
				d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r215}
				ctx.BindReg(r215, &d208)
			}
			if d208.Loc == scm.LocReg && d185.Loc == scm.LocReg && d208.Reg == d185.Reg {
				ctx.TransferReg(d185.Reg)
				d185.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d208)
			ctx.EnsureDesc(&d208)
			var d209 scm.JITValueDesc
			if d208.Loc == scm.LocImm {
				d209 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d208.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d208.Reg)
				ctx.W.EmitMovRegReg(scratch, d208.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d209)
			}
			if d209.Loc == scm.LocReg && d208.Loc == scm.LocReg && d209.Reg == d208.Reg {
				ctx.TransferReg(d208.Reg)
				d208.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d208)
			ctx.EnsureDesc(&d209)
			r216 := ctx.AllocReg()
			ctx.EnsureDesc(&d209)
			ctx.EnsureDesc(&d186)
			if d209.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r216, uint64(d209.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r216, d209.Reg)
				ctx.W.EmitShlRegImm8(r216, 3)
			}
			if d186.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d186.Imm.Int()))
				ctx.W.EmitAddInt64(r216, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r216, d186.Reg)
			}
			r217 := ctx.AllocRegExcept(r216)
			ctx.W.EmitMovRegMem(r217, r216, 0)
			ctx.FreeReg(r216)
			d210 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r217}
			ctx.BindReg(r217, &d210)
			ctx.FreeDesc(&d209)
			ctx.EnsureDesc(&d185)
			var d211 scm.JITValueDesc
			if d185.Loc == scm.LocImm {
				d211 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d185.Imm.Int() % 64)}
			} else {
				r218 := ctx.AllocRegExcept(d185.Reg)
				ctx.W.EmitMovRegReg(r218, d185.Reg)
				ctx.W.EmitAndRegImm32(r218, 63)
				d211 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r218}
				ctx.BindReg(r218, &d211)
			}
			if d211.Loc == scm.LocReg && d185.Loc == scm.LocReg && d211.Reg == d185.Reg {
				ctx.TransferReg(d185.Reg)
				d185.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d185)
			d212 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d211)
			ctx.EnsureDesc(&d212)
			ctx.EnsureDesc(&d211)
			ctx.EnsureDesc(&d212)
			ctx.EnsureDesc(&d211)
			var d213 scm.JITValueDesc
			if d212.Loc == scm.LocImm && d211.Loc == scm.LocImm {
				d213 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d212.Imm.Int() - d211.Imm.Int())}
			} else if d211.Loc == scm.LocImm && d211.Imm.Int() == 0 {
				r219 := ctx.AllocRegExcept(d212.Reg)
				ctx.W.EmitMovRegReg(r219, d212.Reg)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r219}
				ctx.BindReg(r219, &d213)
			} else if d212.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d211.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d212.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d211.Reg)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d213)
			} else if d211.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d212.Reg)
				ctx.W.EmitMovRegReg(scratch, d212.Reg)
				if d211.Imm.Int() >= -2147483648 && d211.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d211.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d211.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d213)
			} else {
				r220 := ctx.AllocRegExcept(d212.Reg, d211.Reg)
				ctx.W.EmitMovRegReg(r220, d212.Reg)
				ctx.W.EmitSubInt64(r220, d211.Reg)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r220}
				ctx.BindReg(r220, &d213)
			}
			if d213.Loc == scm.LocReg && d212.Loc == scm.LocReg && d213.Reg == d212.Reg {
				ctx.TransferReg(d212.Reg)
				d212.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d211)
			ctx.EnsureDesc(&d210)
			ctx.EnsureDesc(&d213)
			var d214 scm.JITValueDesc
			if d210.Loc == scm.LocImm && d213.Loc == scm.LocImm {
				d214 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d210.Imm.Int()) >> uint64(d213.Imm.Int())))}
			} else if d213.Loc == scm.LocImm {
				r221 := ctx.AllocRegExcept(d210.Reg)
				ctx.W.EmitMovRegReg(r221, d210.Reg)
				ctx.W.EmitShrRegImm8(r221, uint8(d213.Imm.Int()))
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r221}
				ctx.BindReg(r221, &d214)
			} else {
				{
					shiftSrc := d210.Reg
					r222 := ctx.AllocRegExcept(d210.Reg)
					ctx.W.EmitMovRegReg(r222, d210.Reg)
					shiftSrc = r222
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d213.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d213.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d213.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d214)
				}
			}
			if d214.Loc == scm.LocReg && d210.Loc == scm.LocReg && d214.Reg == d210.Reg {
				ctx.TransferReg(d210.Reg)
				d210.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d210)
			ctx.FreeDesc(&d213)
			ctx.EnsureDesc(&d190)
			ctx.EnsureDesc(&d214)
			var d215 scm.JITValueDesc
			if d190.Loc == scm.LocImm && d214.Loc == scm.LocImm {
				d215 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d190.Imm.Int() | d214.Imm.Int())}
			} else if d190.Loc == scm.LocImm && d190.Imm.Int() == 0 {
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d214.Reg}
				ctx.BindReg(d214.Reg, &d215)
			} else if d214.Loc == scm.LocImm && d214.Imm.Int() == 0 {
				r223 := ctx.AllocRegExcept(d190.Reg)
				ctx.W.EmitMovRegReg(r223, d190.Reg)
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r223}
				ctx.BindReg(r223, &d215)
			} else if d190.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d214.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d190.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d214.Reg)
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d215)
			} else if d214.Loc == scm.LocImm {
				r224 := ctx.AllocRegExcept(d190.Reg)
				ctx.W.EmitMovRegReg(r224, d190.Reg)
				if d214.Imm.Int() >= -2147483648 && d214.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r224, int32(d214.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d214.Imm.Int()))
					ctx.W.EmitOrInt64(r224, scm.RegR11)
				}
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r224}
				ctx.BindReg(r224, &d215)
			} else {
				r225 := ctx.AllocRegExcept(d190.Reg, d214.Reg)
				ctx.W.EmitMovRegReg(r225, d190.Reg)
				ctx.W.EmitOrInt64(r225, d214.Reg)
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r225}
				ctx.BindReg(r225, &d215)
			}
			if d215.Loc == scm.LocReg && d190.Loc == scm.LocReg && d215.Reg == d190.Reg {
				ctx.TransferReg(d190.Reg)
				d190.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d214)
			d216 = d215
			if d216.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d216)
			ctx.EmitStoreToStack(d216, 32)
			ctx.W.EmitJmp(lbl56)
			ctx.W.MarkLabel(lbl54)
			d217 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r208}
			ctx.BindReg(r208, &d217)
			ctx.BindReg(r208, &d217)
			if r188 { ctx.UnprotectReg(r189) }
			ctx.EnsureDesc(&d217)
			ctx.EnsureDesc(&d217)
			var d218 scm.JITValueDesc
			if d217.Loc == scm.LocImm {
				d218 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d217.Imm.Int()))))}
			} else {
				r226 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r226, d217.Reg)
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r226}
				ctx.BindReg(r226, &d218)
			}
			ctx.FreeDesc(&d217)
			var d219 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d219 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32)
				r227 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r227, thisptr.Reg, off)
				d219 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r227}
				ctx.BindReg(r227, &d219)
			}
			ctx.EnsureDesc(&d218)
			ctx.EnsureDesc(&d219)
			ctx.EnsureDesc(&d218)
			ctx.EnsureDesc(&d219)
			ctx.EnsureDesc(&d218)
			ctx.EnsureDesc(&d219)
			var d220 scm.JITValueDesc
			if d218.Loc == scm.LocImm && d219.Loc == scm.LocImm {
				d220 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d218.Imm.Int() + d219.Imm.Int())}
			} else if d219.Loc == scm.LocImm && d219.Imm.Int() == 0 {
				r228 := ctx.AllocRegExcept(d218.Reg)
				ctx.W.EmitMovRegReg(r228, d218.Reg)
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r228}
				ctx.BindReg(r228, &d220)
			} else if d218.Loc == scm.LocImm && d218.Imm.Int() == 0 {
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d219.Reg}
				ctx.BindReg(d219.Reg, &d220)
			} else if d218.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d219.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d218.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d219.Reg)
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d220)
			} else if d219.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d218.Reg)
				ctx.W.EmitMovRegReg(scratch, d218.Reg)
				if d219.Imm.Int() >= -2147483648 && d219.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d219.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d219.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d220)
			} else {
				r229 := ctx.AllocRegExcept(d218.Reg, d219.Reg)
				ctx.W.EmitMovRegReg(r229, d218.Reg)
				ctx.W.EmitAddInt64(r229, d219.Reg)
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r229}
				ctx.BindReg(r229, &d220)
			}
			if d220.Loc == scm.LocReg && d218.Loc == scm.LocReg && d220.Reg == d218.Reg {
				ctx.TransferReg(d218.Reg)
				d218.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d218)
			ctx.FreeDesc(&d219)
			ctx.EnsureDesc(&d220)
			ctx.EnsureDesc(&d220)
			var d221 scm.JITValueDesc
			if d220.Loc == scm.LocImm {
				d221 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d220.Imm.Int()))))}
			} else {
				r230 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r230, d220.Reg)
				d221 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r230}
				ctx.BindReg(r230, &d221)
			}
			ctx.FreeDesc(&d220)
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d177)
			var d222 scm.JITValueDesc
			if d177.Loc == scm.LocImm {
				d222 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d177.Imm.Int()))))}
			} else {
				r231 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r231, d177.Reg)
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r231}
				ctx.BindReg(r231, &d222)
			}
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d221)
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d221)
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d221)
			var d223 scm.JITValueDesc
			if d177.Loc == scm.LocImm && d221.Loc == scm.LocImm {
				d223 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d177.Imm.Int() + d221.Imm.Int())}
			} else if d221.Loc == scm.LocImm && d221.Imm.Int() == 0 {
				r232 := ctx.AllocRegExcept(d177.Reg)
				ctx.W.EmitMovRegReg(r232, d177.Reg)
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r232}
				ctx.BindReg(r232, &d223)
			} else if d177.Loc == scm.LocImm && d177.Imm.Int() == 0 {
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d221.Reg}
				ctx.BindReg(d221.Reg, &d223)
			} else if d177.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d221.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d177.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d221.Reg)
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d223)
			} else if d221.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d177.Reg)
				ctx.W.EmitMovRegReg(scratch, d177.Reg)
				if d221.Imm.Int() >= -2147483648 && d221.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d221.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d221.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d223)
			} else {
				r233 := ctx.AllocRegExcept(d177.Reg, d221.Reg)
				ctx.W.EmitMovRegReg(r233, d177.Reg)
				ctx.W.EmitAddInt64(r233, d221.Reg)
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r233}
				ctx.BindReg(r233, &d223)
			}
			if d223.Loc == scm.LocReg && d177.Loc == scm.LocReg && d223.Reg == d177.Reg {
				ctx.TransferReg(d177.Reg)
				d177.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d221)
			ctx.EnsureDesc(&d223)
			ctx.EnsureDesc(&d223)
			var d224 scm.JITValueDesc
			if d223.Loc == scm.LocImm {
				d224 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d223.Imm.Int()))))}
			} else {
				r234 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r234, d223.Reg)
				d224 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
				ctx.BindReg(r234, &d224)
			}
			ctx.FreeDesc(&d223)
			ctx.EnsureDesc(&d222)
			ctx.EnsureDesc(&d224)
			ctx.EnsureDesc(&d132)
			ctx.EnsureDesc(&d222)
			ctx.EnsureDesc(&d224)
			r235 := ctx.AllocReg()
			r236 := ctx.AllocRegExcept(r235)
			ctx.EnsureDesc(&d132)
			ctx.EnsureDesc(&d222)
			ctx.EnsureDesc(&d224)
			if d132.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r235, uint64(d132.Imm.Int()))
			} else if d132.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r235, d132.Reg)
			} else {
				ctx.W.EmitMovRegReg(r235, d132.Reg)
			}
			if d222.Loc == scm.LocImm {
				if d222.Imm.Int() != 0 {
					if d222.Imm.Int() >= -2147483648 && d222.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r235, int32(d222.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d222.Imm.Int()))
						ctx.W.EmitAddInt64(r235, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r235, d222.Reg)
			}
			if d224.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r236, uint64(d224.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r236, d224.Reg)
			}
			if d222.Loc == scm.LocImm {
				if d222.Imm.Int() >= -2147483648 && d222.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r236, int32(d222.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d222.Imm.Int()))
					ctx.W.EmitSubInt64(r236, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r236, d222.Reg)
			}
			d225 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r235, Reg2: r236}
			ctx.BindReg(r235, &d225)
			ctx.BindReg(r236, &d225)
			ctx.FreeDesc(&d222)
			ctx.FreeDesc(&d224)
			d226 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r142, Reg2: r143}
			ctx.BindReg(r142, &d226)
			ctx.BindReg(r143, &d226)
			d227 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d225}, 2)
			ctx.EmitMovPairToResult(&d227, &d226)
			ctx.W.EmitJmp(lbl9)
			bbpos_1_8 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl22)
			ctx.W.ResolveFixups()
			var d228 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d228 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 64)
				r237 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r237, thisptr.Reg, off)
				d228 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r237}
				ctx.BindReg(r237, &d228)
			}
			ctx.EnsureDesc(&d228)
			ctx.EnsureDesc(&d228)
			var d229 scm.JITValueDesc
			if d228.Loc == scm.LocImm {
				d229 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d228.Imm.Int()))))}
			} else {
				r238 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r238, d228.Reg)
				ctx.W.EmitShlRegImm8(r238, 32)
				ctx.W.EmitShrRegImm8(r238, 32)
				d229 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r238}
				ctx.BindReg(r238, &d229)
			}
			ctx.FreeDesc(&d228)
			ctx.EnsureDesc(&d44)
			ctx.EnsureDesc(&d229)
			ctx.EnsureDesc(&d44)
			ctx.EnsureDesc(&d229)
			ctx.EnsureDesc(&d44)
			ctx.EnsureDesc(&d229)
			var d230 scm.JITValueDesc
			if d44.Loc == scm.LocImm && d229.Loc == scm.LocImm {
				d230 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d44.Imm.Int()) == uint64(d229.Imm.Int()))}
			} else if d229.Loc == scm.LocImm {
				r239 := ctx.AllocRegExcept(d44.Reg)
				if d229.Imm.Int() >= -2147483648 && d229.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d44.Reg, int32(d229.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d229.Imm.Int()))
					ctx.W.EmitCmpInt64(d44.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r239, scm.CcE)
				d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r239}
				ctx.BindReg(r239, &d230)
			} else if d44.Loc == scm.LocImm {
				r240 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d44.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d229.Reg)
				ctx.W.EmitSetcc(r240, scm.CcE)
				d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r240}
				ctx.BindReg(r240, &d230)
			} else {
				r241 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitCmpInt64(d44.Reg, d229.Reg)
				ctx.W.EmitSetcc(r241, scm.CcE)
				d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r241}
				ctx.BindReg(r241, &d230)
			}
			ctx.FreeDesc(&d44)
			ctx.FreeDesc(&d229)
			d231 = d230
			ctx.EnsureDesc(&d231)
			if d231.Loc != scm.LocImm && d231.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl62 := ctx.W.ReserveLabel()
			lbl63 := ctx.W.ReserveLabel()
			lbl64 := ctx.W.ReserveLabel()
			if d231.Loc == scm.LocImm {
				if d231.Imm.Bool() {
					ctx.W.MarkLabel(lbl63)
					ctx.W.EmitJmp(lbl62)
				} else {
					ctx.W.MarkLabel(lbl64)
					ctx.W.EmitJmp(lbl23)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d231.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl63)
				ctx.W.EmitJmp(lbl64)
				ctx.W.MarkLabel(lbl63)
				ctx.W.EmitJmp(lbl62)
				ctx.W.MarkLabel(lbl64)
				ctx.W.EmitJmp(lbl23)
			}
			ctx.FreeDesc(&d230)
			bbpos_1_5 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl50)
			ctx.W.ResolveFixups()
			var d232 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d232 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 64)
				r242 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r242, thisptr.Reg, off)
				d232 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r242}
				ctx.BindReg(r242, &d232)
			}
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d232)
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d232)
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d232)
			var d233 scm.JITValueDesc
			if d177.Loc == scm.LocImm && d232.Loc == scm.LocImm {
				d233 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d177.Imm.Int()) == uint64(d232.Imm.Int()))}
			} else if d232.Loc == scm.LocImm {
				r243 := ctx.AllocRegExcept(d177.Reg)
				if d232.Imm.Int() >= -2147483648 && d232.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d177.Reg, int32(d232.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d232.Imm.Int()))
					ctx.W.EmitCmpInt64(d177.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r243, scm.CcE)
				d233 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r243}
				ctx.BindReg(r243, &d233)
			} else if d177.Loc == scm.LocImm {
				r244 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d177.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d232.Reg)
				ctx.W.EmitSetcc(r244, scm.CcE)
				d233 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r244}
				ctx.BindReg(r244, &d233)
			} else {
				r245 := ctx.AllocRegExcept(d177.Reg)
				ctx.W.EmitCmpInt64(d177.Reg, d232.Reg)
				ctx.W.EmitSetcc(r245, scm.CcE)
				d233 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r245}
				ctx.BindReg(r245, &d233)
			}
			ctx.FreeDesc(&d177)
			ctx.FreeDesc(&d232)
			d234 = d233
			ctx.EnsureDesc(&d234)
			if d234.Loc != scm.LocImm && d234.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl65 := ctx.W.ReserveLabel()
			lbl66 := ctx.W.ReserveLabel()
			lbl67 := ctx.W.ReserveLabel()
			if d234.Loc == scm.LocImm {
				if d234.Imm.Bool() {
					ctx.W.MarkLabel(lbl66)
					ctx.W.EmitJmp(lbl65)
				} else {
					ctx.W.MarkLabel(lbl67)
					ctx.W.EmitJmp(lbl51)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d234.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl66)
				ctx.W.EmitJmp(lbl67)
				ctx.W.MarkLabel(lbl66)
				ctx.W.EmitJmp(lbl65)
				ctx.W.MarkLabel(lbl67)
				ctx.W.EmitJmp(lbl51)
			}
			ctx.FreeDesc(&d233)
			bbpos_1_6 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl62)
			ctx.W.ResolveFixups()
			d235 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r142, Reg2: r143}
			ctx.BindReg(r142, &d235)
			ctx.BindReg(r143, &d235)
			ctx.W.EmitMakeNil(d235)
			ctx.W.EmitJmp(lbl9)
			bbpos_1_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl65)
			ctx.W.ResolveFixups()
			d236 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r142, Reg2: r143}
			ctx.BindReg(r142, &d236)
			ctx.BindReg(r143, &d236)
			ctx.W.EmitMakeNil(d236)
			ctx.W.EmitJmp(lbl9)
			ctx.W.MarkLabel(lbl9)
			d237 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r142, Reg2: r143}
			ctx.BindReg(r142, &d237)
			ctx.BindReg(r143, &d237)
			ctx.BindReg(r142, &d237)
			ctx.BindReg(r143, &d237)
			if r2 { ctx.UnprotectReg(r3) }
			d239 = d237
			d239.ID = 0
			d238 = ctx.EmitTagEqualsBorrowed(&d239, scm.TagNil, scm.JITValueDesc{Loc: scm.LocAny})
			d240 = d238
			ctx.EnsureDesc(&d240)
			if d240.Loc != scm.LocImm && d240.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d240.Loc == scm.LocImm {
				if d240.Imm.Bool() {
			ps241 := scm.PhiState{General: ps.General}
			ps241.OverlayValues = make([]scm.JITValueDesc, 241)
			ps241.OverlayValues[0] = d0
			ps241.OverlayValues[1] = d1
			ps241.OverlayValues[2] = d2
			ps241.OverlayValues[3] = d3
			ps241.OverlayValues[4] = d4
			ps241.OverlayValues[5] = d5
			ps241.OverlayValues[6] = d6
			ps241.OverlayValues[7] = d7
			ps241.OverlayValues[8] = d8
			ps241.OverlayValues[9] = d9
			ps241.OverlayValues[10] = d10
			ps241.OverlayValues[11] = d11
			ps241.OverlayValues[12] = d12
			ps241.OverlayValues[13] = d13
			ps241.OverlayValues[14] = d14
			ps241.OverlayValues[15] = d15
			ps241.OverlayValues[16] = d16
			ps241.OverlayValues[17] = d17
			ps241.OverlayValues[18] = d18
			ps241.OverlayValues[19] = d19
			ps241.OverlayValues[20] = d20
			ps241.OverlayValues[21] = d21
			ps241.OverlayValues[22] = d22
			ps241.OverlayValues[23] = d23
			ps241.OverlayValues[24] = d24
			ps241.OverlayValues[25] = d25
			ps241.OverlayValues[26] = d26
			ps241.OverlayValues[27] = d27
			ps241.OverlayValues[28] = d28
			ps241.OverlayValues[29] = d29
			ps241.OverlayValues[30] = d30
			ps241.OverlayValues[31] = d31
			ps241.OverlayValues[32] = d32
			ps241.OverlayValues[33] = d33
			ps241.OverlayValues[34] = d34
			ps241.OverlayValues[35] = d35
			ps241.OverlayValues[36] = d36
			ps241.OverlayValues[37] = d37
			ps241.OverlayValues[38] = d38
			ps241.OverlayValues[39] = d39
			ps241.OverlayValues[40] = d40
			ps241.OverlayValues[41] = d41
			ps241.OverlayValues[42] = d42
			ps241.OverlayValues[43] = d43
			ps241.OverlayValues[44] = d44
			ps241.OverlayValues[45] = d45
			ps241.OverlayValues[46] = d46
			ps241.OverlayValues[47] = d47
			ps241.OverlayValues[48] = d48
			ps241.OverlayValues[49] = d49
			ps241.OverlayValues[50] = d50
			ps241.OverlayValues[51] = d51
			ps241.OverlayValues[52] = d52
			ps241.OverlayValues[53] = d53
			ps241.OverlayValues[54] = d54
			ps241.OverlayValues[55] = d55
			ps241.OverlayValues[56] = d56
			ps241.OverlayValues[57] = d57
			ps241.OverlayValues[58] = d58
			ps241.OverlayValues[59] = d59
			ps241.OverlayValues[60] = d60
			ps241.OverlayValues[61] = d61
			ps241.OverlayValues[62] = d62
			ps241.OverlayValues[63] = d63
			ps241.OverlayValues[64] = d64
			ps241.OverlayValues[65] = d65
			ps241.OverlayValues[66] = d66
			ps241.OverlayValues[67] = d67
			ps241.OverlayValues[68] = d68
			ps241.OverlayValues[69] = d69
			ps241.OverlayValues[70] = d70
			ps241.OverlayValues[71] = d71
			ps241.OverlayValues[72] = d72
			ps241.OverlayValues[73] = d73
			ps241.OverlayValues[74] = d74
			ps241.OverlayValues[75] = d75
			ps241.OverlayValues[76] = d76
			ps241.OverlayValues[77] = d77
			ps241.OverlayValues[78] = d78
			ps241.OverlayValues[79] = d79
			ps241.OverlayValues[80] = d80
			ps241.OverlayValues[81] = d81
			ps241.OverlayValues[82] = d82
			ps241.OverlayValues[83] = d83
			ps241.OverlayValues[84] = d84
			ps241.OverlayValues[85] = d85
			ps241.OverlayValues[86] = d86
			ps241.OverlayValues[87] = d87
			ps241.OverlayValues[88] = d88
			ps241.OverlayValues[89] = d89
			ps241.OverlayValues[90] = d90
			ps241.OverlayValues[91] = d91
			ps241.OverlayValues[92] = d92
			ps241.OverlayValues[93] = d93
			ps241.OverlayValues[94] = d94
			ps241.OverlayValues[95] = d95
			ps241.OverlayValues[96] = d96
			ps241.OverlayValues[97] = d97
			ps241.OverlayValues[98] = d98
			ps241.OverlayValues[99] = d99
			ps241.OverlayValues[100] = d100
			ps241.OverlayValues[101] = d101
			ps241.OverlayValues[102] = d102
			ps241.OverlayValues[103] = d103
			ps241.OverlayValues[104] = d104
			ps241.OverlayValues[105] = d105
			ps241.OverlayValues[106] = d106
			ps241.OverlayValues[107] = d107
			ps241.OverlayValues[108] = d108
			ps241.OverlayValues[109] = d109
			ps241.OverlayValues[110] = d110
			ps241.OverlayValues[111] = d111
			ps241.OverlayValues[112] = d112
			ps241.OverlayValues[113] = d113
			ps241.OverlayValues[114] = d114
			ps241.OverlayValues[115] = d115
			ps241.OverlayValues[116] = d116
			ps241.OverlayValues[117] = d117
			ps241.OverlayValues[118] = d118
			ps241.OverlayValues[119] = d119
			ps241.OverlayValues[120] = d120
			ps241.OverlayValues[121] = d121
			ps241.OverlayValues[122] = d122
			ps241.OverlayValues[123] = d123
			ps241.OverlayValues[124] = d124
			ps241.OverlayValues[125] = d125
			ps241.OverlayValues[126] = d126
			ps241.OverlayValues[127] = d127
			ps241.OverlayValues[128] = d128
			ps241.OverlayValues[129] = d129
			ps241.OverlayValues[130] = d130
			ps241.OverlayValues[131] = d131
			ps241.OverlayValues[132] = d132
			ps241.OverlayValues[133] = d133
			ps241.OverlayValues[134] = d134
			ps241.OverlayValues[135] = d135
			ps241.OverlayValues[136] = d136
			ps241.OverlayValues[137] = d137
			ps241.OverlayValues[138] = d138
			ps241.OverlayValues[139] = d139
			ps241.OverlayValues[140] = d140
			ps241.OverlayValues[141] = d141
			ps241.OverlayValues[142] = d142
			ps241.OverlayValues[143] = d143
			ps241.OverlayValues[144] = d144
			ps241.OverlayValues[145] = d145
			ps241.OverlayValues[146] = d146
			ps241.OverlayValues[147] = d147
			ps241.OverlayValues[148] = d148
			ps241.OverlayValues[149] = d149
			ps241.OverlayValues[150] = d150
			ps241.OverlayValues[151] = d151
			ps241.OverlayValues[152] = d152
			ps241.OverlayValues[153] = d153
			ps241.OverlayValues[154] = d154
			ps241.OverlayValues[155] = d155
			ps241.OverlayValues[156] = d156
			ps241.OverlayValues[157] = d157
			ps241.OverlayValues[158] = d158
			ps241.OverlayValues[159] = d159
			ps241.OverlayValues[160] = d160
			ps241.OverlayValues[161] = d161
			ps241.OverlayValues[162] = d162
			ps241.OverlayValues[163] = d163
			ps241.OverlayValues[164] = d164
			ps241.OverlayValues[165] = d165
			ps241.OverlayValues[166] = d166
			ps241.OverlayValues[167] = d167
			ps241.OverlayValues[168] = d168
			ps241.OverlayValues[169] = d169
			ps241.OverlayValues[170] = d170
			ps241.OverlayValues[171] = d171
			ps241.OverlayValues[172] = d172
			ps241.OverlayValues[173] = d173
			ps241.OverlayValues[174] = d174
			ps241.OverlayValues[175] = d175
			ps241.OverlayValues[176] = d176
			ps241.OverlayValues[177] = d177
			ps241.OverlayValues[178] = d178
			ps241.OverlayValues[179] = d179
			ps241.OverlayValues[180] = d180
			ps241.OverlayValues[181] = d181
			ps241.OverlayValues[182] = d182
			ps241.OverlayValues[183] = d183
			ps241.OverlayValues[184] = d184
			ps241.OverlayValues[185] = d185
			ps241.OverlayValues[186] = d186
			ps241.OverlayValues[187] = d187
			ps241.OverlayValues[188] = d188
			ps241.OverlayValues[189] = d189
			ps241.OverlayValues[190] = d190
			ps241.OverlayValues[191] = d191
			ps241.OverlayValues[192] = d192
			ps241.OverlayValues[193] = d193
			ps241.OverlayValues[194] = d194
			ps241.OverlayValues[195] = d195
			ps241.OverlayValues[196] = d196
			ps241.OverlayValues[197] = d197
			ps241.OverlayValues[198] = d198
			ps241.OverlayValues[199] = d199
			ps241.OverlayValues[200] = d200
			ps241.OverlayValues[201] = d201
			ps241.OverlayValues[202] = d202
			ps241.OverlayValues[203] = d203
			ps241.OverlayValues[204] = d204
			ps241.OverlayValues[205] = d205
			ps241.OverlayValues[206] = d206
			ps241.OverlayValues[207] = d207
			ps241.OverlayValues[208] = d208
			ps241.OverlayValues[209] = d209
			ps241.OverlayValues[210] = d210
			ps241.OverlayValues[211] = d211
			ps241.OverlayValues[212] = d212
			ps241.OverlayValues[213] = d213
			ps241.OverlayValues[214] = d214
			ps241.OverlayValues[215] = d215
			ps241.OverlayValues[216] = d216
			ps241.OverlayValues[217] = d217
			ps241.OverlayValues[218] = d218
			ps241.OverlayValues[219] = d219
			ps241.OverlayValues[220] = d220
			ps241.OverlayValues[221] = d221
			ps241.OverlayValues[222] = d222
			ps241.OverlayValues[223] = d223
			ps241.OverlayValues[224] = d224
			ps241.OverlayValues[225] = d225
			ps241.OverlayValues[226] = d226
			ps241.OverlayValues[227] = d227
			ps241.OverlayValues[228] = d228
			ps241.OverlayValues[229] = d229
			ps241.OverlayValues[230] = d230
			ps241.OverlayValues[231] = d231
			ps241.OverlayValues[232] = d232
			ps241.OverlayValues[233] = d233
			ps241.OverlayValues[234] = d234
			ps241.OverlayValues[235] = d235
			ps241.OverlayValues[236] = d236
			ps241.OverlayValues[237] = d237
			ps241.OverlayValues[238] = d238
			ps241.OverlayValues[239] = d239
			ps241.OverlayValues[240] = d240
					return bbs[1].RenderPS(ps241)
				}
			ps242 := scm.PhiState{General: ps.General}
			ps242.OverlayValues = make([]scm.JITValueDesc, 241)
			ps242.OverlayValues[0] = d0
			ps242.OverlayValues[1] = d1
			ps242.OverlayValues[2] = d2
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
			ps242.OverlayValues[18] = d18
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
			ps242.OverlayValues[52] = d52
			ps242.OverlayValues[53] = d53
			ps242.OverlayValues[54] = d54
			ps242.OverlayValues[55] = d55
			ps242.OverlayValues[56] = d56
			ps242.OverlayValues[57] = d57
			ps242.OverlayValues[58] = d58
			ps242.OverlayValues[59] = d59
			ps242.OverlayValues[60] = d60
			ps242.OverlayValues[61] = d61
			ps242.OverlayValues[62] = d62
			ps242.OverlayValues[63] = d63
			ps242.OverlayValues[64] = d64
			ps242.OverlayValues[65] = d65
			ps242.OverlayValues[66] = d66
			ps242.OverlayValues[67] = d67
			ps242.OverlayValues[68] = d68
			ps242.OverlayValues[69] = d69
			ps242.OverlayValues[70] = d70
			ps242.OverlayValues[71] = d71
			ps242.OverlayValues[72] = d72
			ps242.OverlayValues[73] = d73
			ps242.OverlayValues[74] = d74
			ps242.OverlayValues[75] = d75
			ps242.OverlayValues[76] = d76
			ps242.OverlayValues[77] = d77
			ps242.OverlayValues[78] = d78
			ps242.OverlayValues[79] = d79
			ps242.OverlayValues[80] = d80
			ps242.OverlayValues[81] = d81
			ps242.OverlayValues[82] = d82
			ps242.OverlayValues[83] = d83
			ps242.OverlayValues[84] = d84
			ps242.OverlayValues[85] = d85
			ps242.OverlayValues[86] = d86
			ps242.OverlayValues[87] = d87
			ps242.OverlayValues[88] = d88
			ps242.OverlayValues[89] = d89
			ps242.OverlayValues[90] = d90
			ps242.OverlayValues[91] = d91
			ps242.OverlayValues[92] = d92
			ps242.OverlayValues[93] = d93
			ps242.OverlayValues[94] = d94
			ps242.OverlayValues[95] = d95
			ps242.OverlayValues[96] = d96
			ps242.OverlayValues[97] = d97
			ps242.OverlayValues[98] = d98
			ps242.OverlayValues[99] = d99
			ps242.OverlayValues[100] = d100
			ps242.OverlayValues[101] = d101
			ps242.OverlayValues[102] = d102
			ps242.OverlayValues[103] = d103
			ps242.OverlayValues[104] = d104
			ps242.OverlayValues[105] = d105
			ps242.OverlayValues[106] = d106
			ps242.OverlayValues[107] = d107
			ps242.OverlayValues[108] = d108
			ps242.OverlayValues[109] = d109
			ps242.OverlayValues[110] = d110
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
			ps242.OverlayValues[140] = d140
			ps242.OverlayValues[141] = d141
			ps242.OverlayValues[142] = d142
			ps242.OverlayValues[143] = d143
			ps242.OverlayValues[144] = d144
			ps242.OverlayValues[145] = d145
			ps242.OverlayValues[146] = d146
			ps242.OverlayValues[147] = d147
			ps242.OverlayValues[148] = d148
			ps242.OverlayValues[149] = d149
			ps242.OverlayValues[150] = d150
			ps242.OverlayValues[151] = d151
			ps242.OverlayValues[152] = d152
			ps242.OverlayValues[153] = d153
			ps242.OverlayValues[154] = d154
			ps242.OverlayValues[155] = d155
			ps242.OverlayValues[156] = d156
			ps242.OverlayValues[157] = d157
			ps242.OverlayValues[158] = d158
			ps242.OverlayValues[159] = d159
			ps242.OverlayValues[160] = d160
			ps242.OverlayValues[161] = d161
			ps242.OverlayValues[162] = d162
			ps242.OverlayValues[163] = d163
			ps242.OverlayValues[164] = d164
			ps242.OverlayValues[165] = d165
			ps242.OverlayValues[166] = d166
			ps242.OverlayValues[167] = d167
			ps242.OverlayValues[168] = d168
			ps242.OverlayValues[169] = d169
			ps242.OverlayValues[170] = d170
			ps242.OverlayValues[171] = d171
			ps242.OverlayValues[172] = d172
			ps242.OverlayValues[173] = d173
			ps242.OverlayValues[174] = d174
			ps242.OverlayValues[175] = d175
			ps242.OverlayValues[176] = d176
			ps242.OverlayValues[177] = d177
			ps242.OverlayValues[178] = d178
			ps242.OverlayValues[179] = d179
			ps242.OverlayValues[180] = d180
			ps242.OverlayValues[181] = d181
			ps242.OverlayValues[182] = d182
			ps242.OverlayValues[183] = d183
			ps242.OverlayValues[184] = d184
			ps242.OverlayValues[185] = d185
			ps242.OverlayValues[186] = d186
			ps242.OverlayValues[187] = d187
			ps242.OverlayValues[188] = d188
			ps242.OverlayValues[189] = d189
			ps242.OverlayValues[190] = d190
			ps242.OverlayValues[191] = d191
			ps242.OverlayValues[192] = d192
			ps242.OverlayValues[193] = d193
			ps242.OverlayValues[194] = d194
			ps242.OverlayValues[195] = d195
			ps242.OverlayValues[196] = d196
			ps242.OverlayValues[197] = d197
			ps242.OverlayValues[198] = d198
			ps242.OverlayValues[199] = d199
			ps242.OverlayValues[200] = d200
			ps242.OverlayValues[201] = d201
			ps242.OverlayValues[202] = d202
			ps242.OverlayValues[203] = d203
			ps242.OverlayValues[204] = d204
			ps242.OverlayValues[205] = d205
			ps242.OverlayValues[206] = d206
			ps242.OverlayValues[207] = d207
			ps242.OverlayValues[208] = d208
			ps242.OverlayValues[209] = d209
			ps242.OverlayValues[210] = d210
			ps242.OverlayValues[211] = d211
			ps242.OverlayValues[212] = d212
			ps242.OverlayValues[213] = d213
			ps242.OverlayValues[214] = d214
			ps242.OverlayValues[215] = d215
			ps242.OverlayValues[216] = d216
			ps242.OverlayValues[217] = d217
			ps242.OverlayValues[218] = d218
			ps242.OverlayValues[219] = d219
			ps242.OverlayValues[220] = d220
			ps242.OverlayValues[221] = d221
			ps242.OverlayValues[222] = d222
			ps242.OverlayValues[223] = d223
			ps242.OverlayValues[224] = d224
			ps242.OverlayValues[225] = d225
			ps242.OverlayValues[226] = d226
			ps242.OverlayValues[227] = d227
			ps242.OverlayValues[228] = d228
			ps242.OverlayValues[229] = d229
			ps242.OverlayValues[230] = d230
			ps242.OverlayValues[231] = d231
			ps242.OverlayValues[232] = d232
			ps242.OverlayValues[233] = d233
			ps242.OverlayValues[234] = d234
			ps242.OverlayValues[235] = d235
			ps242.OverlayValues[236] = d236
			ps242.OverlayValues[237] = d237
			ps242.OverlayValues[238] = d238
			ps242.OverlayValues[239] = d239
			ps242.OverlayValues[240] = d240
				return bbs[2].RenderPS(ps242)
			}
			if !ps.General {
				ps.General = true
				return bbs[0].RenderPS(ps)
			}
			lbl68 := ctx.W.ReserveLabel()
			lbl69 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d240.Reg, 0)
			ctx.W.EmitJcc(scm.CcNE, lbl68)
			ctx.W.EmitJmp(lbl69)
			ctx.W.MarkLabel(lbl68)
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl69)
			ctx.W.EmitJmp(lbl3)
			ps243 := scm.PhiState{General: true}
			ps243.OverlayValues = make([]scm.JITValueDesc, 241)
			ps243.OverlayValues[0] = d0
			ps243.OverlayValues[1] = d1
			ps243.OverlayValues[2] = d2
			ps243.OverlayValues[3] = d3
			ps243.OverlayValues[4] = d4
			ps243.OverlayValues[5] = d5
			ps243.OverlayValues[6] = d6
			ps243.OverlayValues[7] = d7
			ps243.OverlayValues[8] = d8
			ps243.OverlayValues[9] = d9
			ps243.OverlayValues[10] = d10
			ps243.OverlayValues[11] = d11
			ps243.OverlayValues[12] = d12
			ps243.OverlayValues[13] = d13
			ps243.OverlayValues[14] = d14
			ps243.OverlayValues[15] = d15
			ps243.OverlayValues[16] = d16
			ps243.OverlayValues[17] = d17
			ps243.OverlayValues[18] = d18
			ps243.OverlayValues[19] = d19
			ps243.OverlayValues[20] = d20
			ps243.OverlayValues[21] = d21
			ps243.OverlayValues[22] = d22
			ps243.OverlayValues[23] = d23
			ps243.OverlayValues[24] = d24
			ps243.OverlayValues[25] = d25
			ps243.OverlayValues[26] = d26
			ps243.OverlayValues[27] = d27
			ps243.OverlayValues[28] = d28
			ps243.OverlayValues[29] = d29
			ps243.OverlayValues[30] = d30
			ps243.OverlayValues[31] = d31
			ps243.OverlayValues[32] = d32
			ps243.OverlayValues[33] = d33
			ps243.OverlayValues[34] = d34
			ps243.OverlayValues[35] = d35
			ps243.OverlayValues[36] = d36
			ps243.OverlayValues[37] = d37
			ps243.OverlayValues[38] = d38
			ps243.OverlayValues[39] = d39
			ps243.OverlayValues[40] = d40
			ps243.OverlayValues[41] = d41
			ps243.OverlayValues[42] = d42
			ps243.OverlayValues[43] = d43
			ps243.OverlayValues[44] = d44
			ps243.OverlayValues[45] = d45
			ps243.OverlayValues[46] = d46
			ps243.OverlayValues[47] = d47
			ps243.OverlayValues[48] = d48
			ps243.OverlayValues[49] = d49
			ps243.OverlayValues[50] = d50
			ps243.OverlayValues[51] = d51
			ps243.OverlayValues[52] = d52
			ps243.OverlayValues[53] = d53
			ps243.OverlayValues[54] = d54
			ps243.OverlayValues[55] = d55
			ps243.OverlayValues[56] = d56
			ps243.OverlayValues[57] = d57
			ps243.OverlayValues[58] = d58
			ps243.OverlayValues[59] = d59
			ps243.OverlayValues[60] = d60
			ps243.OverlayValues[61] = d61
			ps243.OverlayValues[62] = d62
			ps243.OverlayValues[63] = d63
			ps243.OverlayValues[64] = d64
			ps243.OverlayValues[65] = d65
			ps243.OverlayValues[66] = d66
			ps243.OverlayValues[67] = d67
			ps243.OverlayValues[68] = d68
			ps243.OverlayValues[69] = d69
			ps243.OverlayValues[70] = d70
			ps243.OverlayValues[71] = d71
			ps243.OverlayValues[72] = d72
			ps243.OverlayValues[73] = d73
			ps243.OverlayValues[74] = d74
			ps243.OverlayValues[75] = d75
			ps243.OverlayValues[76] = d76
			ps243.OverlayValues[77] = d77
			ps243.OverlayValues[78] = d78
			ps243.OverlayValues[79] = d79
			ps243.OverlayValues[80] = d80
			ps243.OverlayValues[81] = d81
			ps243.OverlayValues[82] = d82
			ps243.OverlayValues[83] = d83
			ps243.OverlayValues[84] = d84
			ps243.OverlayValues[85] = d85
			ps243.OverlayValues[86] = d86
			ps243.OverlayValues[87] = d87
			ps243.OverlayValues[88] = d88
			ps243.OverlayValues[89] = d89
			ps243.OverlayValues[90] = d90
			ps243.OverlayValues[91] = d91
			ps243.OverlayValues[92] = d92
			ps243.OverlayValues[93] = d93
			ps243.OverlayValues[94] = d94
			ps243.OverlayValues[95] = d95
			ps243.OverlayValues[96] = d96
			ps243.OverlayValues[97] = d97
			ps243.OverlayValues[98] = d98
			ps243.OverlayValues[99] = d99
			ps243.OverlayValues[100] = d100
			ps243.OverlayValues[101] = d101
			ps243.OverlayValues[102] = d102
			ps243.OverlayValues[103] = d103
			ps243.OverlayValues[104] = d104
			ps243.OverlayValues[105] = d105
			ps243.OverlayValues[106] = d106
			ps243.OverlayValues[107] = d107
			ps243.OverlayValues[108] = d108
			ps243.OverlayValues[109] = d109
			ps243.OverlayValues[110] = d110
			ps243.OverlayValues[111] = d111
			ps243.OverlayValues[112] = d112
			ps243.OverlayValues[113] = d113
			ps243.OverlayValues[114] = d114
			ps243.OverlayValues[115] = d115
			ps243.OverlayValues[116] = d116
			ps243.OverlayValues[117] = d117
			ps243.OverlayValues[118] = d118
			ps243.OverlayValues[119] = d119
			ps243.OverlayValues[120] = d120
			ps243.OverlayValues[121] = d121
			ps243.OverlayValues[122] = d122
			ps243.OverlayValues[123] = d123
			ps243.OverlayValues[124] = d124
			ps243.OverlayValues[125] = d125
			ps243.OverlayValues[126] = d126
			ps243.OverlayValues[127] = d127
			ps243.OverlayValues[128] = d128
			ps243.OverlayValues[129] = d129
			ps243.OverlayValues[130] = d130
			ps243.OverlayValues[131] = d131
			ps243.OverlayValues[132] = d132
			ps243.OverlayValues[133] = d133
			ps243.OverlayValues[134] = d134
			ps243.OverlayValues[135] = d135
			ps243.OverlayValues[136] = d136
			ps243.OverlayValues[137] = d137
			ps243.OverlayValues[138] = d138
			ps243.OverlayValues[139] = d139
			ps243.OverlayValues[140] = d140
			ps243.OverlayValues[141] = d141
			ps243.OverlayValues[142] = d142
			ps243.OverlayValues[143] = d143
			ps243.OverlayValues[144] = d144
			ps243.OverlayValues[145] = d145
			ps243.OverlayValues[146] = d146
			ps243.OverlayValues[147] = d147
			ps243.OverlayValues[148] = d148
			ps243.OverlayValues[149] = d149
			ps243.OverlayValues[150] = d150
			ps243.OverlayValues[151] = d151
			ps243.OverlayValues[152] = d152
			ps243.OverlayValues[153] = d153
			ps243.OverlayValues[154] = d154
			ps243.OverlayValues[155] = d155
			ps243.OverlayValues[156] = d156
			ps243.OverlayValues[157] = d157
			ps243.OverlayValues[158] = d158
			ps243.OverlayValues[159] = d159
			ps243.OverlayValues[160] = d160
			ps243.OverlayValues[161] = d161
			ps243.OverlayValues[162] = d162
			ps243.OverlayValues[163] = d163
			ps243.OverlayValues[164] = d164
			ps243.OverlayValues[165] = d165
			ps243.OverlayValues[166] = d166
			ps243.OverlayValues[167] = d167
			ps243.OverlayValues[168] = d168
			ps243.OverlayValues[169] = d169
			ps243.OverlayValues[170] = d170
			ps243.OverlayValues[171] = d171
			ps243.OverlayValues[172] = d172
			ps243.OverlayValues[173] = d173
			ps243.OverlayValues[174] = d174
			ps243.OverlayValues[175] = d175
			ps243.OverlayValues[176] = d176
			ps243.OverlayValues[177] = d177
			ps243.OverlayValues[178] = d178
			ps243.OverlayValues[179] = d179
			ps243.OverlayValues[180] = d180
			ps243.OverlayValues[181] = d181
			ps243.OverlayValues[182] = d182
			ps243.OverlayValues[183] = d183
			ps243.OverlayValues[184] = d184
			ps243.OverlayValues[185] = d185
			ps243.OverlayValues[186] = d186
			ps243.OverlayValues[187] = d187
			ps243.OverlayValues[188] = d188
			ps243.OverlayValues[189] = d189
			ps243.OverlayValues[190] = d190
			ps243.OverlayValues[191] = d191
			ps243.OverlayValues[192] = d192
			ps243.OverlayValues[193] = d193
			ps243.OverlayValues[194] = d194
			ps243.OverlayValues[195] = d195
			ps243.OverlayValues[196] = d196
			ps243.OverlayValues[197] = d197
			ps243.OverlayValues[198] = d198
			ps243.OverlayValues[199] = d199
			ps243.OverlayValues[200] = d200
			ps243.OverlayValues[201] = d201
			ps243.OverlayValues[202] = d202
			ps243.OverlayValues[203] = d203
			ps243.OverlayValues[204] = d204
			ps243.OverlayValues[205] = d205
			ps243.OverlayValues[206] = d206
			ps243.OverlayValues[207] = d207
			ps243.OverlayValues[208] = d208
			ps243.OverlayValues[209] = d209
			ps243.OverlayValues[210] = d210
			ps243.OverlayValues[211] = d211
			ps243.OverlayValues[212] = d212
			ps243.OverlayValues[213] = d213
			ps243.OverlayValues[214] = d214
			ps243.OverlayValues[215] = d215
			ps243.OverlayValues[216] = d216
			ps243.OverlayValues[217] = d217
			ps243.OverlayValues[218] = d218
			ps243.OverlayValues[219] = d219
			ps243.OverlayValues[220] = d220
			ps243.OverlayValues[221] = d221
			ps243.OverlayValues[222] = d222
			ps243.OverlayValues[223] = d223
			ps243.OverlayValues[224] = d224
			ps243.OverlayValues[225] = d225
			ps243.OverlayValues[226] = d226
			ps243.OverlayValues[227] = d227
			ps243.OverlayValues[228] = d228
			ps243.OverlayValues[229] = d229
			ps243.OverlayValues[230] = d230
			ps243.OverlayValues[231] = d231
			ps243.OverlayValues[232] = d232
			ps243.OverlayValues[233] = d233
			ps243.OverlayValues[234] = d234
			ps243.OverlayValues[235] = d235
			ps243.OverlayValues[236] = d236
			ps243.OverlayValues[237] = d237
			ps243.OverlayValues[238] = d238
			ps243.OverlayValues[239] = d239
			ps243.OverlayValues[240] = d240
			ps244 := scm.PhiState{General: true}
			ps244.OverlayValues = make([]scm.JITValueDesc, 241)
			ps244.OverlayValues[0] = d0
			ps244.OverlayValues[1] = d1
			ps244.OverlayValues[2] = d2
			ps244.OverlayValues[3] = d3
			ps244.OverlayValues[4] = d4
			ps244.OverlayValues[5] = d5
			ps244.OverlayValues[6] = d6
			ps244.OverlayValues[7] = d7
			ps244.OverlayValues[8] = d8
			ps244.OverlayValues[9] = d9
			ps244.OverlayValues[10] = d10
			ps244.OverlayValues[11] = d11
			ps244.OverlayValues[12] = d12
			ps244.OverlayValues[13] = d13
			ps244.OverlayValues[14] = d14
			ps244.OverlayValues[15] = d15
			ps244.OverlayValues[16] = d16
			ps244.OverlayValues[17] = d17
			ps244.OverlayValues[18] = d18
			ps244.OverlayValues[19] = d19
			ps244.OverlayValues[20] = d20
			ps244.OverlayValues[21] = d21
			ps244.OverlayValues[22] = d22
			ps244.OverlayValues[23] = d23
			ps244.OverlayValues[24] = d24
			ps244.OverlayValues[25] = d25
			ps244.OverlayValues[26] = d26
			ps244.OverlayValues[27] = d27
			ps244.OverlayValues[28] = d28
			ps244.OverlayValues[29] = d29
			ps244.OverlayValues[30] = d30
			ps244.OverlayValues[31] = d31
			ps244.OverlayValues[32] = d32
			ps244.OverlayValues[33] = d33
			ps244.OverlayValues[34] = d34
			ps244.OverlayValues[35] = d35
			ps244.OverlayValues[36] = d36
			ps244.OverlayValues[37] = d37
			ps244.OverlayValues[38] = d38
			ps244.OverlayValues[39] = d39
			ps244.OverlayValues[40] = d40
			ps244.OverlayValues[41] = d41
			ps244.OverlayValues[42] = d42
			ps244.OverlayValues[43] = d43
			ps244.OverlayValues[44] = d44
			ps244.OverlayValues[45] = d45
			ps244.OverlayValues[46] = d46
			ps244.OverlayValues[47] = d47
			ps244.OverlayValues[48] = d48
			ps244.OverlayValues[49] = d49
			ps244.OverlayValues[50] = d50
			ps244.OverlayValues[51] = d51
			ps244.OverlayValues[52] = d52
			ps244.OverlayValues[53] = d53
			ps244.OverlayValues[54] = d54
			ps244.OverlayValues[55] = d55
			ps244.OverlayValues[56] = d56
			ps244.OverlayValues[57] = d57
			ps244.OverlayValues[58] = d58
			ps244.OverlayValues[59] = d59
			ps244.OverlayValues[60] = d60
			ps244.OverlayValues[61] = d61
			ps244.OverlayValues[62] = d62
			ps244.OverlayValues[63] = d63
			ps244.OverlayValues[64] = d64
			ps244.OverlayValues[65] = d65
			ps244.OverlayValues[66] = d66
			ps244.OverlayValues[67] = d67
			ps244.OverlayValues[68] = d68
			ps244.OverlayValues[69] = d69
			ps244.OverlayValues[70] = d70
			ps244.OverlayValues[71] = d71
			ps244.OverlayValues[72] = d72
			ps244.OverlayValues[73] = d73
			ps244.OverlayValues[74] = d74
			ps244.OverlayValues[75] = d75
			ps244.OverlayValues[76] = d76
			ps244.OverlayValues[77] = d77
			ps244.OverlayValues[78] = d78
			ps244.OverlayValues[79] = d79
			ps244.OverlayValues[80] = d80
			ps244.OverlayValues[81] = d81
			ps244.OverlayValues[82] = d82
			ps244.OverlayValues[83] = d83
			ps244.OverlayValues[84] = d84
			ps244.OverlayValues[85] = d85
			ps244.OverlayValues[86] = d86
			ps244.OverlayValues[87] = d87
			ps244.OverlayValues[88] = d88
			ps244.OverlayValues[89] = d89
			ps244.OverlayValues[90] = d90
			ps244.OverlayValues[91] = d91
			ps244.OverlayValues[92] = d92
			ps244.OverlayValues[93] = d93
			ps244.OverlayValues[94] = d94
			ps244.OverlayValues[95] = d95
			ps244.OverlayValues[96] = d96
			ps244.OverlayValues[97] = d97
			ps244.OverlayValues[98] = d98
			ps244.OverlayValues[99] = d99
			ps244.OverlayValues[100] = d100
			ps244.OverlayValues[101] = d101
			ps244.OverlayValues[102] = d102
			ps244.OverlayValues[103] = d103
			ps244.OverlayValues[104] = d104
			ps244.OverlayValues[105] = d105
			ps244.OverlayValues[106] = d106
			ps244.OverlayValues[107] = d107
			ps244.OverlayValues[108] = d108
			ps244.OverlayValues[109] = d109
			ps244.OverlayValues[110] = d110
			ps244.OverlayValues[111] = d111
			ps244.OverlayValues[112] = d112
			ps244.OverlayValues[113] = d113
			ps244.OverlayValues[114] = d114
			ps244.OverlayValues[115] = d115
			ps244.OverlayValues[116] = d116
			ps244.OverlayValues[117] = d117
			ps244.OverlayValues[118] = d118
			ps244.OverlayValues[119] = d119
			ps244.OverlayValues[120] = d120
			ps244.OverlayValues[121] = d121
			ps244.OverlayValues[122] = d122
			ps244.OverlayValues[123] = d123
			ps244.OverlayValues[124] = d124
			ps244.OverlayValues[125] = d125
			ps244.OverlayValues[126] = d126
			ps244.OverlayValues[127] = d127
			ps244.OverlayValues[128] = d128
			ps244.OverlayValues[129] = d129
			ps244.OverlayValues[130] = d130
			ps244.OverlayValues[131] = d131
			ps244.OverlayValues[132] = d132
			ps244.OverlayValues[133] = d133
			ps244.OverlayValues[134] = d134
			ps244.OverlayValues[135] = d135
			ps244.OverlayValues[136] = d136
			ps244.OverlayValues[137] = d137
			ps244.OverlayValues[138] = d138
			ps244.OverlayValues[139] = d139
			ps244.OverlayValues[140] = d140
			ps244.OverlayValues[141] = d141
			ps244.OverlayValues[142] = d142
			ps244.OverlayValues[143] = d143
			ps244.OverlayValues[144] = d144
			ps244.OverlayValues[145] = d145
			ps244.OverlayValues[146] = d146
			ps244.OverlayValues[147] = d147
			ps244.OverlayValues[148] = d148
			ps244.OverlayValues[149] = d149
			ps244.OverlayValues[150] = d150
			ps244.OverlayValues[151] = d151
			ps244.OverlayValues[152] = d152
			ps244.OverlayValues[153] = d153
			ps244.OverlayValues[154] = d154
			ps244.OverlayValues[155] = d155
			ps244.OverlayValues[156] = d156
			ps244.OverlayValues[157] = d157
			ps244.OverlayValues[158] = d158
			ps244.OverlayValues[159] = d159
			ps244.OverlayValues[160] = d160
			ps244.OverlayValues[161] = d161
			ps244.OverlayValues[162] = d162
			ps244.OverlayValues[163] = d163
			ps244.OverlayValues[164] = d164
			ps244.OverlayValues[165] = d165
			ps244.OverlayValues[166] = d166
			ps244.OverlayValues[167] = d167
			ps244.OverlayValues[168] = d168
			ps244.OverlayValues[169] = d169
			ps244.OverlayValues[170] = d170
			ps244.OverlayValues[171] = d171
			ps244.OverlayValues[172] = d172
			ps244.OverlayValues[173] = d173
			ps244.OverlayValues[174] = d174
			ps244.OverlayValues[175] = d175
			ps244.OverlayValues[176] = d176
			ps244.OverlayValues[177] = d177
			ps244.OverlayValues[178] = d178
			ps244.OverlayValues[179] = d179
			ps244.OverlayValues[180] = d180
			ps244.OverlayValues[181] = d181
			ps244.OverlayValues[182] = d182
			ps244.OverlayValues[183] = d183
			ps244.OverlayValues[184] = d184
			ps244.OverlayValues[185] = d185
			ps244.OverlayValues[186] = d186
			ps244.OverlayValues[187] = d187
			ps244.OverlayValues[188] = d188
			ps244.OverlayValues[189] = d189
			ps244.OverlayValues[190] = d190
			ps244.OverlayValues[191] = d191
			ps244.OverlayValues[192] = d192
			ps244.OverlayValues[193] = d193
			ps244.OverlayValues[194] = d194
			ps244.OverlayValues[195] = d195
			ps244.OverlayValues[196] = d196
			ps244.OverlayValues[197] = d197
			ps244.OverlayValues[198] = d198
			ps244.OverlayValues[199] = d199
			ps244.OverlayValues[200] = d200
			ps244.OverlayValues[201] = d201
			ps244.OverlayValues[202] = d202
			ps244.OverlayValues[203] = d203
			ps244.OverlayValues[204] = d204
			ps244.OverlayValues[205] = d205
			ps244.OverlayValues[206] = d206
			ps244.OverlayValues[207] = d207
			ps244.OverlayValues[208] = d208
			ps244.OverlayValues[209] = d209
			ps244.OverlayValues[210] = d210
			ps244.OverlayValues[211] = d211
			ps244.OverlayValues[212] = d212
			ps244.OverlayValues[213] = d213
			ps244.OverlayValues[214] = d214
			ps244.OverlayValues[215] = d215
			ps244.OverlayValues[216] = d216
			ps244.OverlayValues[217] = d217
			ps244.OverlayValues[218] = d218
			ps244.OverlayValues[219] = d219
			ps244.OverlayValues[220] = d220
			ps244.OverlayValues[221] = d221
			ps244.OverlayValues[222] = d222
			ps244.OverlayValues[223] = d223
			ps244.OverlayValues[224] = d224
			ps244.OverlayValues[225] = d225
			ps244.OverlayValues[226] = d226
			ps244.OverlayValues[227] = d227
			ps244.OverlayValues[228] = d228
			ps244.OverlayValues[229] = d229
			ps244.OverlayValues[230] = d230
			ps244.OverlayValues[231] = d231
			ps244.OverlayValues[232] = d232
			ps244.OverlayValues[233] = d233
			ps244.OverlayValues[234] = d234
			ps244.OverlayValues[235] = d235
			ps244.OverlayValues[236] = d236
			ps244.OverlayValues[237] = d237
			ps244.OverlayValues[238] = d238
			ps244.OverlayValues[239] = d239
			ps244.OverlayValues[240] = d240
			alloc245 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps244)
			}
			ctx.RestoreAllocState(alloc245)
			if !bbs[1].Rendered {
				return bbs[1].RenderPS(ps243)
			}
			return result
			ctx.FreeDesc(&d238)
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
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
				d6 = ps.OverlayValues[6]
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
			if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != scm.LocNone {
				d101 = ps.OverlayValues[101]
			}
			if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != scm.LocNone {
				d102 = ps.OverlayValues[102]
			}
			if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != scm.LocNone {
				d103 = ps.OverlayValues[103]
			}
			if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != scm.LocNone {
				d104 = ps.OverlayValues[104]
			}
			if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != scm.LocNone {
				d105 = ps.OverlayValues[105]
			}
			if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != scm.LocNone {
				d106 = ps.OverlayValues[106]
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
			ctx.ReclaimUntrackedRegs()
			d246 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d246)
			ctx.BindReg(r1, &d246)
			ctx.W.EmitMakeNil(d246)
			ctx.W.EmitJmp(lbl0)
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
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
				d6 = ps.OverlayValues[6]
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
			if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != scm.LocNone {
				d101 = ps.OverlayValues[101]
			}
			if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != scm.LocNone {
				d102 = ps.OverlayValues[102]
			}
			if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != scm.LocNone {
				d103 = ps.OverlayValues[103]
			}
			if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != scm.LocNone {
				d104 = ps.OverlayValues[104]
			}
			if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != scm.LocNone {
				d105 = ps.OverlayValues[105]
			}
			if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != scm.LocNone {
				d106 = ps.OverlayValues[106]
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
			if len(ps.OverlayValues) > 246 && ps.OverlayValues[246].Loc != scm.LocNone {
				d246 = ps.OverlayValues[246]
			}
			ctx.ReclaimUntrackedRegs()
			d248 = d237
			d248.ID = 0
			d247 = ctx.EmitTagEqualsBorrowed(&d248, scm.TagString, scm.JITValueDesc{Loc: scm.LocAny})
			d249 = d247
			ctx.EnsureDesc(&d249)
			if d249.Loc != scm.LocImm && d249.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d249.Loc == scm.LocImm {
				if d249.Imm.Bool() {
			ps250 := scm.PhiState{General: ps.General}
			ps250.OverlayValues = make([]scm.JITValueDesc, 250)
			ps250.OverlayValues[0] = d0
			ps250.OverlayValues[1] = d1
			ps250.OverlayValues[2] = d2
			ps250.OverlayValues[3] = d3
			ps250.OverlayValues[4] = d4
			ps250.OverlayValues[5] = d5
			ps250.OverlayValues[6] = d6
			ps250.OverlayValues[7] = d7
			ps250.OverlayValues[8] = d8
			ps250.OverlayValues[9] = d9
			ps250.OverlayValues[10] = d10
			ps250.OverlayValues[11] = d11
			ps250.OverlayValues[12] = d12
			ps250.OverlayValues[13] = d13
			ps250.OverlayValues[14] = d14
			ps250.OverlayValues[15] = d15
			ps250.OverlayValues[16] = d16
			ps250.OverlayValues[17] = d17
			ps250.OverlayValues[18] = d18
			ps250.OverlayValues[19] = d19
			ps250.OverlayValues[20] = d20
			ps250.OverlayValues[21] = d21
			ps250.OverlayValues[22] = d22
			ps250.OverlayValues[23] = d23
			ps250.OverlayValues[24] = d24
			ps250.OverlayValues[25] = d25
			ps250.OverlayValues[26] = d26
			ps250.OverlayValues[27] = d27
			ps250.OverlayValues[28] = d28
			ps250.OverlayValues[29] = d29
			ps250.OverlayValues[30] = d30
			ps250.OverlayValues[31] = d31
			ps250.OverlayValues[32] = d32
			ps250.OverlayValues[33] = d33
			ps250.OverlayValues[34] = d34
			ps250.OverlayValues[35] = d35
			ps250.OverlayValues[36] = d36
			ps250.OverlayValues[37] = d37
			ps250.OverlayValues[38] = d38
			ps250.OverlayValues[39] = d39
			ps250.OverlayValues[40] = d40
			ps250.OverlayValues[41] = d41
			ps250.OverlayValues[42] = d42
			ps250.OverlayValues[43] = d43
			ps250.OverlayValues[44] = d44
			ps250.OverlayValues[45] = d45
			ps250.OverlayValues[46] = d46
			ps250.OverlayValues[47] = d47
			ps250.OverlayValues[48] = d48
			ps250.OverlayValues[49] = d49
			ps250.OverlayValues[50] = d50
			ps250.OverlayValues[51] = d51
			ps250.OverlayValues[52] = d52
			ps250.OverlayValues[53] = d53
			ps250.OverlayValues[54] = d54
			ps250.OverlayValues[55] = d55
			ps250.OverlayValues[56] = d56
			ps250.OverlayValues[57] = d57
			ps250.OverlayValues[58] = d58
			ps250.OverlayValues[59] = d59
			ps250.OverlayValues[60] = d60
			ps250.OverlayValues[61] = d61
			ps250.OverlayValues[62] = d62
			ps250.OverlayValues[63] = d63
			ps250.OverlayValues[64] = d64
			ps250.OverlayValues[65] = d65
			ps250.OverlayValues[66] = d66
			ps250.OverlayValues[67] = d67
			ps250.OverlayValues[68] = d68
			ps250.OverlayValues[69] = d69
			ps250.OverlayValues[70] = d70
			ps250.OverlayValues[71] = d71
			ps250.OverlayValues[72] = d72
			ps250.OverlayValues[73] = d73
			ps250.OverlayValues[74] = d74
			ps250.OverlayValues[75] = d75
			ps250.OverlayValues[76] = d76
			ps250.OverlayValues[77] = d77
			ps250.OverlayValues[78] = d78
			ps250.OverlayValues[79] = d79
			ps250.OverlayValues[80] = d80
			ps250.OverlayValues[81] = d81
			ps250.OverlayValues[82] = d82
			ps250.OverlayValues[83] = d83
			ps250.OverlayValues[84] = d84
			ps250.OverlayValues[85] = d85
			ps250.OverlayValues[86] = d86
			ps250.OverlayValues[87] = d87
			ps250.OverlayValues[88] = d88
			ps250.OverlayValues[89] = d89
			ps250.OverlayValues[90] = d90
			ps250.OverlayValues[91] = d91
			ps250.OverlayValues[92] = d92
			ps250.OverlayValues[93] = d93
			ps250.OverlayValues[94] = d94
			ps250.OverlayValues[95] = d95
			ps250.OverlayValues[96] = d96
			ps250.OverlayValues[97] = d97
			ps250.OverlayValues[98] = d98
			ps250.OverlayValues[99] = d99
			ps250.OverlayValues[100] = d100
			ps250.OverlayValues[101] = d101
			ps250.OverlayValues[102] = d102
			ps250.OverlayValues[103] = d103
			ps250.OverlayValues[104] = d104
			ps250.OverlayValues[105] = d105
			ps250.OverlayValues[106] = d106
			ps250.OverlayValues[107] = d107
			ps250.OverlayValues[108] = d108
			ps250.OverlayValues[109] = d109
			ps250.OverlayValues[110] = d110
			ps250.OverlayValues[111] = d111
			ps250.OverlayValues[112] = d112
			ps250.OverlayValues[113] = d113
			ps250.OverlayValues[114] = d114
			ps250.OverlayValues[115] = d115
			ps250.OverlayValues[116] = d116
			ps250.OverlayValues[117] = d117
			ps250.OverlayValues[118] = d118
			ps250.OverlayValues[119] = d119
			ps250.OverlayValues[120] = d120
			ps250.OverlayValues[121] = d121
			ps250.OverlayValues[122] = d122
			ps250.OverlayValues[123] = d123
			ps250.OverlayValues[124] = d124
			ps250.OverlayValues[125] = d125
			ps250.OverlayValues[126] = d126
			ps250.OverlayValues[127] = d127
			ps250.OverlayValues[128] = d128
			ps250.OverlayValues[129] = d129
			ps250.OverlayValues[130] = d130
			ps250.OverlayValues[131] = d131
			ps250.OverlayValues[132] = d132
			ps250.OverlayValues[133] = d133
			ps250.OverlayValues[134] = d134
			ps250.OverlayValues[135] = d135
			ps250.OverlayValues[136] = d136
			ps250.OverlayValues[137] = d137
			ps250.OverlayValues[138] = d138
			ps250.OverlayValues[139] = d139
			ps250.OverlayValues[140] = d140
			ps250.OverlayValues[141] = d141
			ps250.OverlayValues[142] = d142
			ps250.OverlayValues[143] = d143
			ps250.OverlayValues[144] = d144
			ps250.OverlayValues[145] = d145
			ps250.OverlayValues[146] = d146
			ps250.OverlayValues[147] = d147
			ps250.OverlayValues[148] = d148
			ps250.OverlayValues[149] = d149
			ps250.OverlayValues[150] = d150
			ps250.OverlayValues[151] = d151
			ps250.OverlayValues[152] = d152
			ps250.OverlayValues[153] = d153
			ps250.OverlayValues[154] = d154
			ps250.OverlayValues[155] = d155
			ps250.OverlayValues[156] = d156
			ps250.OverlayValues[157] = d157
			ps250.OverlayValues[158] = d158
			ps250.OverlayValues[159] = d159
			ps250.OverlayValues[160] = d160
			ps250.OverlayValues[161] = d161
			ps250.OverlayValues[162] = d162
			ps250.OverlayValues[163] = d163
			ps250.OverlayValues[164] = d164
			ps250.OverlayValues[165] = d165
			ps250.OverlayValues[166] = d166
			ps250.OverlayValues[167] = d167
			ps250.OverlayValues[168] = d168
			ps250.OverlayValues[169] = d169
			ps250.OverlayValues[170] = d170
			ps250.OverlayValues[171] = d171
			ps250.OverlayValues[172] = d172
			ps250.OverlayValues[173] = d173
			ps250.OverlayValues[174] = d174
			ps250.OverlayValues[175] = d175
			ps250.OverlayValues[176] = d176
			ps250.OverlayValues[177] = d177
			ps250.OverlayValues[178] = d178
			ps250.OverlayValues[179] = d179
			ps250.OverlayValues[180] = d180
			ps250.OverlayValues[181] = d181
			ps250.OverlayValues[182] = d182
			ps250.OverlayValues[183] = d183
			ps250.OverlayValues[184] = d184
			ps250.OverlayValues[185] = d185
			ps250.OverlayValues[186] = d186
			ps250.OverlayValues[187] = d187
			ps250.OverlayValues[188] = d188
			ps250.OverlayValues[189] = d189
			ps250.OverlayValues[190] = d190
			ps250.OverlayValues[191] = d191
			ps250.OverlayValues[192] = d192
			ps250.OverlayValues[193] = d193
			ps250.OverlayValues[194] = d194
			ps250.OverlayValues[195] = d195
			ps250.OverlayValues[196] = d196
			ps250.OverlayValues[197] = d197
			ps250.OverlayValues[198] = d198
			ps250.OverlayValues[199] = d199
			ps250.OverlayValues[200] = d200
			ps250.OverlayValues[201] = d201
			ps250.OverlayValues[202] = d202
			ps250.OverlayValues[203] = d203
			ps250.OverlayValues[204] = d204
			ps250.OverlayValues[205] = d205
			ps250.OverlayValues[206] = d206
			ps250.OverlayValues[207] = d207
			ps250.OverlayValues[208] = d208
			ps250.OverlayValues[209] = d209
			ps250.OverlayValues[210] = d210
			ps250.OverlayValues[211] = d211
			ps250.OverlayValues[212] = d212
			ps250.OverlayValues[213] = d213
			ps250.OverlayValues[214] = d214
			ps250.OverlayValues[215] = d215
			ps250.OverlayValues[216] = d216
			ps250.OverlayValues[217] = d217
			ps250.OverlayValues[218] = d218
			ps250.OverlayValues[219] = d219
			ps250.OverlayValues[220] = d220
			ps250.OverlayValues[221] = d221
			ps250.OverlayValues[222] = d222
			ps250.OverlayValues[223] = d223
			ps250.OverlayValues[224] = d224
			ps250.OverlayValues[225] = d225
			ps250.OverlayValues[226] = d226
			ps250.OverlayValues[227] = d227
			ps250.OverlayValues[228] = d228
			ps250.OverlayValues[229] = d229
			ps250.OverlayValues[230] = d230
			ps250.OverlayValues[231] = d231
			ps250.OverlayValues[232] = d232
			ps250.OverlayValues[233] = d233
			ps250.OverlayValues[234] = d234
			ps250.OverlayValues[235] = d235
			ps250.OverlayValues[236] = d236
			ps250.OverlayValues[237] = d237
			ps250.OverlayValues[238] = d238
			ps250.OverlayValues[239] = d239
			ps250.OverlayValues[240] = d240
			ps250.OverlayValues[246] = d246
			ps250.OverlayValues[247] = d247
			ps250.OverlayValues[248] = d248
			ps250.OverlayValues[249] = d249
					return bbs[4].RenderPS(ps250)
				}
			ps251 := scm.PhiState{General: ps.General}
			ps251.OverlayValues = make([]scm.JITValueDesc, 250)
			ps251.OverlayValues[0] = d0
			ps251.OverlayValues[1] = d1
			ps251.OverlayValues[2] = d2
			ps251.OverlayValues[3] = d3
			ps251.OverlayValues[4] = d4
			ps251.OverlayValues[5] = d5
			ps251.OverlayValues[6] = d6
			ps251.OverlayValues[7] = d7
			ps251.OverlayValues[8] = d8
			ps251.OverlayValues[9] = d9
			ps251.OverlayValues[10] = d10
			ps251.OverlayValues[11] = d11
			ps251.OverlayValues[12] = d12
			ps251.OverlayValues[13] = d13
			ps251.OverlayValues[14] = d14
			ps251.OverlayValues[15] = d15
			ps251.OverlayValues[16] = d16
			ps251.OverlayValues[17] = d17
			ps251.OverlayValues[18] = d18
			ps251.OverlayValues[19] = d19
			ps251.OverlayValues[20] = d20
			ps251.OverlayValues[21] = d21
			ps251.OverlayValues[22] = d22
			ps251.OverlayValues[23] = d23
			ps251.OverlayValues[24] = d24
			ps251.OverlayValues[25] = d25
			ps251.OverlayValues[26] = d26
			ps251.OverlayValues[27] = d27
			ps251.OverlayValues[28] = d28
			ps251.OverlayValues[29] = d29
			ps251.OverlayValues[30] = d30
			ps251.OverlayValues[31] = d31
			ps251.OverlayValues[32] = d32
			ps251.OverlayValues[33] = d33
			ps251.OverlayValues[34] = d34
			ps251.OverlayValues[35] = d35
			ps251.OverlayValues[36] = d36
			ps251.OverlayValues[37] = d37
			ps251.OverlayValues[38] = d38
			ps251.OverlayValues[39] = d39
			ps251.OverlayValues[40] = d40
			ps251.OverlayValues[41] = d41
			ps251.OverlayValues[42] = d42
			ps251.OverlayValues[43] = d43
			ps251.OverlayValues[44] = d44
			ps251.OverlayValues[45] = d45
			ps251.OverlayValues[46] = d46
			ps251.OverlayValues[47] = d47
			ps251.OverlayValues[48] = d48
			ps251.OverlayValues[49] = d49
			ps251.OverlayValues[50] = d50
			ps251.OverlayValues[51] = d51
			ps251.OverlayValues[52] = d52
			ps251.OverlayValues[53] = d53
			ps251.OverlayValues[54] = d54
			ps251.OverlayValues[55] = d55
			ps251.OverlayValues[56] = d56
			ps251.OverlayValues[57] = d57
			ps251.OverlayValues[58] = d58
			ps251.OverlayValues[59] = d59
			ps251.OverlayValues[60] = d60
			ps251.OverlayValues[61] = d61
			ps251.OverlayValues[62] = d62
			ps251.OverlayValues[63] = d63
			ps251.OverlayValues[64] = d64
			ps251.OverlayValues[65] = d65
			ps251.OverlayValues[66] = d66
			ps251.OverlayValues[67] = d67
			ps251.OverlayValues[68] = d68
			ps251.OverlayValues[69] = d69
			ps251.OverlayValues[70] = d70
			ps251.OverlayValues[71] = d71
			ps251.OverlayValues[72] = d72
			ps251.OverlayValues[73] = d73
			ps251.OverlayValues[74] = d74
			ps251.OverlayValues[75] = d75
			ps251.OverlayValues[76] = d76
			ps251.OverlayValues[77] = d77
			ps251.OverlayValues[78] = d78
			ps251.OverlayValues[79] = d79
			ps251.OverlayValues[80] = d80
			ps251.OverlayValues[81] = d81
			ps251.OverlayValues[82] = d82
			ps251.OverlayValues[83] = d83
			ps251.OverlayValues[84] = d84
			ps251.OverlayValues[85] = d85
			ps251.OverlayValues[86] = d86
			ps251.OverlayValues[87] = d87
			ps251.OverlayValues[88] = d88
			ps251.OverlayValues[89] = d89
			ps251.OverlayValues[90] = d90
			ps251.OverlayValues[91] = d91
			ps251.OverlayValues[92] = d92
			ps251.OverlayValues[93] = d93
			ps251.OverlayValues[94] = d94
			ps251.OverlayValues[95] = d95
			ps251.OverlayValues[96] = d96
			ps251.OverlayValues[97] = d97
			ps251.OverlayValues[98] = d98
			ps251.OverlayValues[99] = d99
			ps251.OverlayValues[100] = d100
			ps251.OverlayValues[101] = d101
			ps251.OverlayValues[102] = d102
			ps251.OverlayValues[103] = d103
			ps251.OverlayValues[104] = d104
			ps251.OverlayValues[105] = d105
			ps251.OverlayValues[106] = d106
			ps251.OverlayValues[107] = d107
			ps251.OverlayValues[108] = d108
			ps251.OverlayValues[109] = d109
			ps251.OverlayValues[110] = d110
			ps251.OverlayValues[111] = d111
			ps251.OverlayValues[112] = d112
			ps251.OverlayValues[113] = d113
			ps251.OverlayValues[114] = d114
			ps251.OverlayValues[115] = d115
			ps251.OverlayValues[116] = d116
			ps251.OverlayValues[117] = d117
			ps251.OverlayValues[118] = d118
			ps251.OverlayValues[119] = d119
			ps251.OverlayValues[120] = d120
			ps251.OverlayValues[121] = d121
			ps251.OverlayValues[122] = d122
			ps251.OverlayValues[123] = d123
			ps251.OverlayValues[124] = d124
			ps251.OverlayValues[125] = d125
			ps251.OverlayValues[126] = d126
			ps251.OverlayValues[127] = d127
			ps251.OverlayValues[128] = d128
			ps251.OverlayValues[129] = d129
			ps251.OverlayValues[130] = d130
			ps251.OverlayValues[131] = d131
			ps251.OverlayValues[132] = d132
			ps251.OverlayValues[133] = d133
			ps251.OverlayValues[134] = d134
			ps251.OverlayValues[135] = d135
			ps251.OverlayValues[136] = d136
			ps251.OverlayValues[137] = d137
			ps251.OverlayValues[138] = d138
			ps251.OverlayValues[139] = d139
			ps251.OverlayValues[140] = d140
			ps251.OverlayValues[141] = d141
			ps251.OverlayValues[142] = d142
			ps251.OverlayValues[143] = d143
			ps251.OverlayValues[144] = d144
			ps251.OverlayValues[145] = d145
			ps251.OverlayValues[146] = d146
			ps251.OverlayValues[147] = d147
			ps251.OverlayValues[148] = d148
			ps251.OverlayValues[149] = d149
			ps251.OverlayValues[150] = d150
			ps251.OverlayValues[151] = d151
			ps251.OverlayValues[152] = d152
			ps251.OverlayValues[153] = d153
			ps251.OverlayValues[154] = d154
			ps251.OverlayValues[155] = d155
			ps251.OverlayValues[156] = d156
			ps251.OverlayValues[157] = d157
			ps251.OverlayValues[158] = d158
			ps251.OverlayValues[159] = d159
			ps251.OverlayValues[160] = d160
			ps251.OverlayValues[161] = d161
			ps251.OverlayValues[162] = d162
			ps251.OverlayValues[163] = d163
			ps251.OverlayValues[164] = d164
			ps251.OverlayValues[165] = d165
			ps251.OverlayValues[166] = d166
			ps251.OverlayValues[167] = d167
			ps251.OverlayValues[168] = d168
			ps251.OverlayValues[169] = d169
			ps251.OverlayValues[170] = d170
			ps251.OverlayValues[171] = d171
			ps251.OverlayValues[172] = d172
			ps251.OverlayValues[173] = d173
			ps251.OverlayValues[174] = d174
			ps251.OverlayValues[175] = d175
			ps251.OverlayValues[176] = d176
			ps251.OverlayValues[177] = d177
			ps251.OverlayValues[178] = d178
			ps251.OverlayValues[179] = d179
			ps251.OverlayValues[180] = d180
			ps251.OverlayValues[181] = d181
			ps251.OverlayValues[182] = d182
			ps251.OverlayValues[183] = d183
			ps251.OverlayValues[184] = d184
			ps251.OverlayValues[185] = d185
			ps251.OverlayValues[186] = d186
			ps251.OverlayValues[187] = d187
			ps251.OverlayValues[188] = d188
			ps251.OverlayValues[189] = d189
			ps251.OverlayValues[190] = d190
			ps251.OverlayValues[191] = d191
			ps251.OverlayValues[192] = d192
			ps251.OverlayValues[193] = d193
			ps251.OverlayValues[194] = d194
			ps251.OverlayValues[195] = d195
			ps251.OverlayValues[196] = d196
			ps251.OverlayValues[197] = d197
			ps251.OverlayValues[198] = d198
			ps251.OverlayValues[199] = d199
			ps251.OverlayValues[200] = d200
			ps251.OverlayValues[201] = d201
			ps251.OverlayValues[202] = d202
			ps251.OverlayValues[203] = d203
			ps251.OverlayValues[204] = d204
			ps251.OverlayValues[205] = d205
			ps251.OverlayValues[206] = d206
			ps251.OverlayValues[207] = d207
			ps251.OverlayValues[208] = d208
			ps251.OverlayValues[209] = d209
			ps251.OverlayValues[210] = d210
			ps251.OverlayValues[211] = d211
			ps251.OverlayValues[212] = d212
			ps251.OverlayValues[213] = d213
			ps251.OverlayValues[214] = d214
			ps251.OverlayValues[215] = d215
			ps251.OverlayValues[216] = d216
			ps251.OverlayValues[217] = d217
			ps251.OverlayValues[218] = d218
			ps251.OverlayValues[219] = d219
			ps251.OverlayValues[220] = d220
			ps251.OverlayValues[221] = d221
			ps251.OverlayValues[222] = d222
			ps251.OverlayValues[223] = d223
			ps251.OverlayValues[224] = d224
			ps251.OverlayValues[225] = d225
			ps251.OverlayValues[226] = d226
			ps251.OverlayValues[227] = d227
			ps251.OverlayValues[228] = d228
			ps251.OverlayValues[229] = d229
			ps251.OverlayValues[230] = d230
			ps251.OverlayValues[231] = d231
			ps251.OverlayValues[232] = d232
			ps251.OverlayValues[233] = d233
			ps251.OverlayValues[234] = d234
			ps251.OverlayValues[235] = d235
			ps251.OverlayValues[236] = d236
			ps251.OverlayValues[237] = d237
			ps251.OverlayValues[238] = d238
			ps251.OverlayValues[239] = d239
			ps251.OverlayValues[240] = d240
			ps251.OverlayValues[246] = d246
			ps251.OverlayValues[247] = d247
			ps251.OverlayValues[248] = d248
			ps251.OverlayValues[249] = d249
				return bbs[3].RenderPS(ps251)
			}
			if !ps.General {
				ps.General = true
				return bbs[2].RenderPS(ps)
			}
			lbl70 := ctx.W.ReserveLabel()
			lbl71 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d249.Reg, 0)
			ctx.W.EmitJcc(scm.CcNE, lbl70)
			ctx.W.EmitJmp(lbl71)
			ctx.W.MarkLabel(lbl70)
			ctx.W.EmitJmp(lbl5)
			ctx.W.MarkLabel(lbl71)
			ctx.W.EmitJmp(lbl4)
			ps252 := scm.PhiState{General: true}
			ps252.OverlayValues = make([]scm.JITValueDesc, 250)
			ps252.OverlayValues[0] = d0
			ps252.OverlayValues[1] = d1
			ps252.OverlayValues[2] = d2
			ps252.OverlayValues[3] = d3
			ps252.OverlayValues[4] = d4
			ps252.OverlayValues[5] = d5
			ps252.OverlayValues[6] = d6
			ps252.OverlayValues[7] = d7
			ps252.OverlayValues[8] = d8
			ps252.OverlayValues[9] = d9
			ps252.OverlayValues[10] = d10
			ps252.OverlayValues[11] = d11
			ps252.OverlayValues[12] = d12
			ps252.OverlayValues[13] = d13
			ps252.OverlayValues[14] = d14
			ps252.OverlayValues[15] = d15
			ps252.OverlayValues[16] = d16
			ps252.OverlayValues[17] = d17
			ps252.OverlayValues[18] = d18
			ps252.OverlayValues[19] = d19
			ps252.OverlayValues[20] = d20
			ps252.OverlayValues[21] = d21
			ps252.OverlayValues[22] = d22
			ps252.OverlayValues[23] = d23
			ps252.OverlayValues[24] = d24
			ps252.OverlayValues[25] = d25
			ps252.OverlayValues[26] = d26
			ps252.OverlayValues[27] = d27
			ps252.OverlayValues[28] = d28
			ps252.OverlayValues[29] = d29
			ps252.OverlayValues[30] = d30
			ps252.OverlayValues[31] = d31
			ps252.OverlayValues[32] = d32
			ps252.OverlayValues[33] = d33
			ps252.OverlayValues[34] = d34
			ps252.OverlayValues[35] = d35
			ps252.OverlayValues[36] = d36
			ps252.OverlayValues[37] = d37
			ps252.OverlayValues[38] = d38
			ps252.OverlayValues[39] = d39
			ps252.OverlayValues[40] = d40
			ps252.OverlayValues[41] = d41
			ps252.OverlayValues[42] = d42
			ps252.OverlayValues[43] = d43
			ps252.OverlayValues[44] = d44
			ps252.OverlayValues[45] = d45
			ps252.OverlayValues[46] = d46
			ps252.OverlayValues[47] = d47
			ps252.OverlayValues[48] = d48
			ps252.OverlayValues[49] = d49
			ps252.OverlayValues[50] = d50
			ps252.OverlayValues[51] = d51
			ps252.OverlayValues[52] = d52
			ps252.OverlayValues[53] = d53
			ps252.OverlayValues[54] = d54
			ps252.OverlayValues[55] = d55
			ps252.OverlayValues[56] = d56
			ps252.OverlayValues[57] = d57
			ps252.OverlayValues[58] = d58
			ps252.OverlayValues[59] = d59
			ps252.OverlayValues[60] = d60
			ps252.OverlayValues[61] = d61
			ps252.OverlayValues[62] = d62
			ps252.OverlayValues[63] = d63
			ps252.OverlayValues[64] = d64
			ps252.OverlayValues[65] = d65
			ps252.OverlayValues[66] = d66
			ps252.OverlayValues[67] = d67
			ps252.OverlayValues[68] = d68
			ps252.OverlayValues[69] = d69
			ps252.OverlayValues[70] = d70
			ps252.OverlayValues[71] = d71
			ps252.OverlayValues[72] = d72
			ps252.OverlayValues[73] = d73
			ps252.OverlayValues[74] = d74
			ps252.OverlayValues[75] = d75
			ps252.OverlayValues[76] = d76
			ps252.OverlayValues[77] = d77
			ps252.OverlayValues[78] = d78
			ps252.OverlayValues[79] = d79
			ps252.OverlayValues[80] = d80
			ps252.OverlayValues[81] = d81
			ps252.OverlayValues[82] = d82
			ps252.OverlayValues[83] = d83
			ps252.OverlayValues[84] = d84
			ps252.OverlayValues[85] = d85
			ps252.OverlayValues[86] = d86
			ps252.OverlayValues[87] = d87
			ps252.OverlayValues[88] = d88
			ps252.OverlayValues[89] = d89
			ps252.OverlayValues[90] = d90
			ps252.OverlayValues[91] = d91
			ps252.OverlayValues[92] = d92
			ps252.OverlayValues[93] = d93
			ps252.OverlayValues[94] = d94
			ps252.OverlayValues[95] = d95
			ps252.OverlayValues[96] = d96
			ps252.OverlayValues[97] = d97
			ps252.OverlayValues[98] = d98
			ps252.OverlayValues[99] = d99
			ps252.OverlayValues[100] = d100
			ps252.OverlayValues[101] = d101
			ps252.OverlayValues[102] = d102
			ps252.OverlayValues[103] = d103
			ps252.OverlayValues[104] = d104
			ps252.OverlayValues[105] = d105
			ps252.OverlayValues[106] = d106
			ps252.OverlayValues[107] = d107
			ps252.OverlayValues[108] = d108
			ps252.OverlayValues[109] = d109
			ps252.OverlayValues[110] = d110
			ps252.OverlayValues[111] = d111
			ps252.OverlayValues[112] = d112
			ps252.OverlayValues[113] = d113
			ps252.OverlayValues[114] = d114
			ps252.OverlayValues[115] = d115
			ps252.OverlayValues[116] = d116
			ps252.OverlayValues[117] = d117
			ps252.OverlayValues[118] = d118
			ps252.OverlayValues[119] = d119
			ps252.OverlayValues[120] = d120
			ps252.OverlayValues[121] = d121
			ps252.OverlayValues[122] = d122
			ps252.OverlayValues[123] = d123
			ps252.OverlayValues[124] = d124
			ps252.OverlayValues[125] = d125
			ps252.OverlayValues[126] = d126
			ps252.OverlayValues[127] = d127
			ps252.OverlayValues[128] = d128
			ps252.OverlayValues[129] = d129
			ps252.OverlayValues[130] = d130
			ps252.OverlayValues[131] = d131
			ps252.OverlayValues[132] = d132
			ps252.OverlayValues[133] = d133
			ps252.OverlayValues[134] = d134
			ps252.OverlayValues[135] = d135
			ps252.OverlayValues[136] = d136
			ps252.OverlayValues[137] = d137
			ps252.OverlayValues[138] = d138
			ps252.OverlayValues[139] = d139
			ps252.OverlayValues[140] = d140
			ps252.OverlayValues[141] = d141
			ps252.OverlayValues[142] = d142
			ps252.OverlayValues[143] = d143
			ps252.OverlayValues[144] = d144
			ps252.OverlayValues[145] = d145
			ps252.OverlayValues[146] = d146
			ps252.OverlayValues[147] = d147
			ps252.OverlayValues[148] = d148
			ps252.OverlayValues[149] = d149
			ps252.OverlayValues[150] = d150
			ps252.OverlayValues[151] = d151
			ps252.OverlayValues[152] = d152
			ps252.OverlayValues[153] = d153
			ps252.OverlayValues[154] = d154
			ps252.OverlayValues[155] = d155
			ps252.OverlayValues[156] = d156
			ps252.OverlayValues[157] = d157
			ps252.OverlayValues[158] = d158
			ps252.OverlayValues[159] = d159
			ps252.OverlayValues[160] = d160
			ps252.OverlayValues[161] = d161
			ps252.OverlayValues[162] = d162
			ps252.OverlayValues[163] = d163
			ps252.OverlayValues[164] = d164
			ps252.OverlayValues[165] = d165
			ps252.OverlayValues[166] = d166
			ps252.OverlayValues[167] = d167
			ps252.OverlayValues[168] = d168
			ps252.OverlayValues[169] = d169
			ps252.OverlayValues[170] = d170
			ps252.OverlayValues[171] = d171
			ps252.OverlayValues[172] = d172
			ps252.OverlayValues[173] = d173
			ps252.OverlayValues[174] = d174
			ps252.OverlayValues[175] = d175
			ps252.OverlayValues[176] = d176
			ps252.OverlayValues[177] = d177
			ps252.OverlayValues[178] = d178
			ps252.OverlayValues[179] = d179
			ps252.OverlayValues[180] = d180
			ps252.OverlayValues[181] = d181
			ps252.OverlayValues[182] = d182
			ps252.OverlayValues[183] = d183
			ps252.OverlayValues[184] = d184
			ps252.OverlayValues[185] = d185
			ps252.OverlayValues[186] = d186
			ps252.OverlayValues[187] = d187
			ps252.OverlayValues[188] = d188
			ps252.OverlayValues[189] = d189
			ps252.OverlayValues[190] = d190
			ps252.OverlayValues[191] = d191
			ps252.OverlayValues[192] = d192
			ps252.OverlayValues[193] = d193
			ps252.OverlayValues[194] = d194
			ps252.OverlayValues[195] = d195
			ps252.OverlayValues[196] = d196
			ps252.OverlayValues[197] = d197
			ps252.OverlayValues[198] = d198
			ps252.OverlayValues[199] = d199
			ps252.OverlayValues[200] = d200
			ps252.OverlayValues[201] = d201
			ps252.OverlayValues[202] = d202
			ps252.OverlayValues[203] = d203
			ps252.OverlayValues[204] = d204
			ps252.OverlayValues[205] = d205
			ps252.OverlayValues[206] = d206
			ps252.OverlayValues[207] = d207
			ps252.OverlayValues[208] = d208
			ps252.OverlayValues[209] = d209
			ps252.OverlayValues[210] = d210
			ps252.OverlayValues[211] = d211
			ps252.OverlayValues[212] = d212
			ps252.OverlayValues[213] = d213
			ps252.OverlayValues[214] = d214
			ps252.OverlayValues[215] = d215
			ps252.OverlayValues[216] = d216
			ps252.OverlayValues[217] = d217
			ps252.OverlayValues[218] = d218
			ps252.OverlayValues[219] = d219
			ps252.OverlayValues[220] = d220
			ps252.OverlayValues[221] = d221
			ps252.OverlayValues[222] = d222
			ps252.OverlayValues[223] = d223
			ps252.OverlayValues[224] = d224
			ps252.OverlayValues[225] = d225
			ps252.OverlayValues[226] = d226
			ps252.OverlayValues[227] = d227
			ps252.OverlayValues[228] = d228
			ps252.OverlayValues[229] = d229
			ps252.OverlayValues[230] = d230
			ps252.OverlayValues[231] = d231
			ps252.OverlayValues[232] = d232
			ps252.OverlayValues[233] = d233
			ps252.OverlayValues[234] = d234
			ps252.OverlayValues[235] = d235
			ps252.OverlayValues[236] = d236
			ps252.OverlayValues[237] = d237
			ps252.OverlayValues[238] = d238
			ps252.OverlayValues[239] = d239
			ps252.OverlayValues[240] = d240
			ps252.OverlayValues[246] = d246
			ps252.OverlayValues[247] = d247
			ps252.OverlayValues[248] = d248
			ps252.OverlayValues[249] = d249
			ps253 := scm.PhiState{General: true}
			ps253.OverlayValues = make([]scm.JITValueDesc, 250)
			ps253.OverlayValues[0] = d0
			ps253.OverlayValues[1] = d1
			ps253.OverlayValues[2] = d2
			ps253.OverlayValues[3] = d3
			ps253.OverlayValues[4] = d4
			ps253.OverlayValues[5] = d5
			ps253.OverlayValues[6] = d6
			ps253.OverlayValues[7] = d7
			ps253.OverlayValues[8] = d8
			ps253.OverlayValues[9] = d9
			ps253.OverlayValues[10] = d10
			ps253.OverlayValues[11] = d11
			ps253.OverlayValues[12] = d12
			ps253.OverlayValues[13] = d13
			ps253.OverlayValues[14] = d14
			ps253.OverlayValues[15] = d15
			ps253.OverlayValues[16] = d16
			ps253.OverlayValues[17] = d17
			ps253.OverlayValues[18] = d18
			ps253.OverlayValues[19] = d19
			ps253.OverlayValues[20] = d20
			ps253.OverlayValues[21] = d21
			ps253.OverlayValues[22] = d22
			ps253.OverlayValues[23] = d23
			ps253.OverlayValues[24] = d24
			ps253.OverlayValues[25] = d25
			ps253.OverlayValues[26] = d26
			ps253.OverlayValues[27] = d27
			ps253.OverlayValues[28] = d28
			ps253.OverlayValues[29] = d29
			ps253.OverlayValues[30] = d30
			ps253.OverlayValues[31] = d31
			ps253.OverlayValues[32] = d32
			ps253.OverlayValues[33] = d33
			ps253.OverlayValues[34] = d34
			ps253.OverlayValues[35] = d35
			ps253.OverlayValues[36] = d36
			ps253.OverlayValues[37] = d37
			ps253.OverlayValues[38] = d38
			ps253.OverlayValues[39] = d39
			ps253.OverlayValues[40] = d40
			ps253.OverlayValues[41] = d41
			ps253.OverlayValues[42] = d42
			ps253.OverlayValues[43] = d43
			ps253.OverlayValues[44] = d44
			ps253.OverlayValues[45] = d45
			ps253.OverlayValues[46] = d46
			ps253.OverlayValues[47] = d47
			ps253.OverlayValues[48] = d48
			ps253.OverlayValues[49] = d49
			ps253.OverlayValues[50] = d50
			ps253.OverlayValues[51] = d51
			ps253.OverlayValues[52] = d52
			ps253.OverlayValues[53] = d53
			ps253.OverlayValues[54] = d54
			ps253.OverlayValues[55] = d55
			ps253.OverlayValues[56] = d56
			ps253.OverlayValues[57] = d57
			ps253.OverlayValues[58] = d58
			ps253.OverlayValues[59] = d59
			ps253.OverlayValues[60] = d60
			ps253.OverlayValues[61] = d61
			ps253.OverlayValues[62] = d62
			ps253.OverlayValues[63] = d63
			ps253.OverlayValues[64] = d64
			ps253.OverlayValues[65] = d65
			ps253.OverlayValues[66] = d66
			ps253.OverlayValues[67] = d67
			ps253.OverlayValues[68] = d68
			ps253.OverlayValues[69] = d69
			ps253.OverlayValues[70] = d70
			ps253.OverlayValues[71] = d71
			ps253.OverlayValues[72] = d72
			ps253.OverlayValues[73] = d73
			ps253.OverlayValues[74] = d74
			ps253.OverlayValues[75] = d75
			ps253.OverlayValues[76] = d76
			ps253.OverlayValues[77] = d77
			ps253.OverlayValues[78] = d78
			ps253.OverlayValues[79] = d79
			ps253.OverlayValues[80] = d80
			ps253.OverlayValues[81] = d81
			ps253.OverlayValues[82] = d82
			ps253.OverlayValues[83] = d83
			ps253.OverlayValues[84] = d84
			ps253.OverlayValues[85] = d85
			ps253.OverlayValues[86] = d86
			ps253.OverlayValues[87] = d87
			ps253.OverlayValues[88] = d88
			ps253.OverlayValues[89] = d89
			ps253.OverlayValues[90] = d90
			ps253.OverlayValues[91] = d91
			ps253.OverlayValues[92] = d92
			ps253.OverlayValues[93] = d93
			ps253.OverlayValues[94] = d94
			ps253.OverlayValues[95] = d95
			ps253.OverlayValues[96] = d96
			ps253.OverlayValues[97] = d97
			ps253.OverlayValues[98] = d98
			ps253.OverlayValues[99] = d99
			ps253.OverlayValues[100] = d100
			ps253.OverlayValues[101] = d101
			ps253.OverlayValues[102] = d102
			ps253.OverlayValues[103] = d103
			ps253.OverlayValues[104] = d104
			ps253.OverlayValues[105] = d105
			ps253.OverlayValues[106] = d106
			ps253.OverlayValues[107] = d107
			ps253.OverlayValues[108] = d108
			ps253.OverlayValues[109] = d109
			ps253.OverlayValues[110] = d110
			ps253.OverlayValues[111] = d111
			ps253.OverlayValues[112] = d112
			ps253.OverlayValues[113] = d113
			ps253.OverlayValues[114] = d114
			ps253.OverlayValues[115] = d115
			ps253.OverlayValues[116] = d116
			ps253.OverlayValues[117] = d117
			ps253.OverlayValues[118] = d118
			ps253.OverlayValues[119] = d119
			ps253.OverlayValues[120] = d120
			ps253.OverlayValues[121] = d121
			ps253.OverlayValues[122] = d122
			ps253.OverlayValues[123] = d123
			ps253.OverlayValues[124] = d124
			ps253.OverlayValues[125] = d125
			ps253.OverlayValues[126] = d126
			ps253.OverlayValues[127] = d127
			ps253.OverlayValues[128] = d128
			ps253.OverlayValues[129] = d129
			ps253.OverlayValues[130] = d130
			ps253.OverlayValues[131] = d131
			ps253.OverlayValues[132] = d132
			ps253.OverlayValues[133] = d133
			ps253.OverlayValues[134] = d134
			ps253.OverlayValues[135] = d135
			ps253.OverlayValues[136] = d136
			ps253.OverlayValues[137] = d137
			ps253.OverlayValues[138] = d138
			ps253.OverlayValues[139] = d139
			ps253.OverlayValues[140] = d140
			ps253.OverlayValues[141] = d141
			ps253.OverlayValues[142] = d142
			ps253.OverlayValues[143] = d143
			ps253.OverlayValues[144] = d144
			ps253.OverlayValues[145] = d145
			ps253.OverlayValues[146] = d146
			ps253.OverlayValues[147] = d147
			ps253.OverlayValues[148] = d148
			ps253.OverlayValues[149] = d149
			ps253.OverlayValues[150] = d150
			ps253.OverlayValues[151] = d151
			ps253.OverlayValues[152] = d152
			ps253.OverlayValues[153] = d153
			ps253.OverlayValues[154] = d154
			ps253.OverlayValues[155] = d155
			ps253.OverlayValues[156] = d156
			ps253.OverlayValues[157] = d157
			ps253.OverlayValues[158] = d158
			ps253.OverlayValues[159] = d159
			ps253.OverlayValues[160] = d160
			ps253.OverlayValues[161] = d161
			ps253.OverlayValues[162] = d162
			ps253.OverlayValues[163] = d163
			ps253.OverlayValues[164] = d164
			ps253.OverlayValues[165] = d165
			ps253.OverlayValues[166] = d166
			ps253.OverlayValues[167] = d167
			ps253.OverlayValues[168] = d168
			ps253.OverlayValues[169] = d169
			ps253.OverlayValues[170] = d170
			ps253.OverlayValues[171] = d171
			ps253.OverlayValues[172] = d172
			ps253.OverlayValues[173] = d173
			ps253.OverlayValues[174] = d174
			ps253.OverlayValues[175] = d175
			ps253.OverlayValues[176] = d176
			ps253.OverlayValues[177] = d177
			ps253.OverlayValues[178] = d178
			ps253.OverlayValues[179] = d179
			ps253.OverlayValues[180] = d180
			ps253.OverlayValues[181] = d181
			ps253.OverlayValues[182] = d182
			ps253.OverlayValues[183] = d183
			ps253.OverlayValues[184] = d184
			ps253.OverlayValues[185] = d185
			ps253.OverlayValues[186] = d186
			ps253.OverlayValues[187] = d187
			ps253.OverlayValues[188] = d188
			ps253.OverlayValues[189] = d189
			ps253.OverlayValues[190] = d190
			ps253.OverlayValues[191] = d191
			ps253.OverlayValues[192] = d192
			ps253.OverlayValues[193] = d193
			ps253.OverlayValues[194] = d194
			ps253.OverlayValues[195] = d195
			ps253.OverlayValues[196] = d196
			ps253.OverlayValues[197] = d197
			ps253.OverlayValues[198] = d198
			ps253.OverlayValues[199] = d199
			ps253.OverlayValues[200] = d200
			ps253.OverlayValues[201] = d201
			ps253.OverlayValues[202] = d202
			ps253.OverlayValues[203] = d203
			ps253.OverlayValues[204] = d204
			ps253.OverlayValues[205] = d205
			ps253.OverlayValues[206] = d206
			ps253.OverlayValues[207] = d207
			ps253.OverlayValues[208] = d208
			ps253.OverlayValues[209] = d209
			ps253.OverlayValues[210] = d210
			ps253.OverlayValues[211] = d211
			ps253.OverlayValues[212] = d212
			ps253.OverlayValues[213] = d213
			ps253.OverlayValues[214] = d214
			ps253.OverlayValues[215] = d215
			ps253.OverlayValues[216] = d216
			ps253.OverlayValues[217] = d217
			ps253.OverlayValues[218] = d218
			ps253.OverlayValues[219] = d219
			ps253.OverlayValues[220] = d220
			ps253.OverlayValues[221] = d221
			ps253.OverlayValues[222] = d222
			ps253.OverlayValues[223] = d223
			ps253.OverlayValues[224] = d224
			ps253.OverlayValues[225] = d225
			ps253.OverlayValues[226] = d226
			ps253.OverlayValues[227] = d227
			ps253.OverlayValues[228] = d228
			ps253.OverlayValues[229] = d229
			ps253.OverlayValues[230] = d230
			ps253.OverlayValues[231] = d231
			ps253.OverlayValues[232] = d232
			ps253.OverlayValues[233] = d233
			ps253.OverlayValues[234] = d234
			ps253.OverlayValues[235] = d235
			ps253.OverlayValues[236] = d236
			ps253.OverlayValues[237] = d237
			ps253.OverlayValues[238] = d238
			ps253.OverlayValues[239] = d239
			ps253.OverlayValues[240] = d240
			ps253.OverlayValues[246] = d246
			ps253.OverlayValues[247] = d247
			ps253.OverlayValues[248] = d248
			ps253.OverlayValues[249] = d249
			alloc254 := ctx.SnapshotAllocState()
			if !bbs[3].Rendered {
				bbs[3].RenderPS(ps253)
			}
			ctx.RestoreAllocState(alloc254)
			if !bbs[4].Rendered {
				return bbs[4].RenderPS(ps252)
			}
			return result
			ctx.FreeDesc(&d247)
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
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
				d6 = ps.OverlayValues[6]
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
			if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != scm.LocNone {
				d101 = ps.OverlayValues[101]
			}
			if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != scm.LocNone {
				d102 = ps.OverlayValues[102]
			}
			if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != scm.LocNone {
				d103 = ps.OverlayValues[103]
			}
			if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != scm.LocNone {
				d104 = ps.OverlayValues[104]
			}
			if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != scm.LocNone {
				d105 = ps.OverlayValues[105]
			}
			if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != scm.LocNone {
				d106 = ps.OverlayValues[106]
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
			ctx.ReclaimUntrackedRegs()
			ctx.W.EmitByte(0xCC)
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
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
				d6 = ps.OverlayValues[6]
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
			if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != scm.LocNone {
				d101 = ps.OverlayValues[101]
			}
			if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != scm.LocNone {
				d102 = ps.OverlayValues[102]
			}
			if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != scm.LocNone {
				d103 = ps.OverlayValues[103]
			}
			if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != scm.LocNone {
				d104 = ps.OverlayValues[104]
			}
			if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != scm.LocNone {
				d105 = ps.OverlayValues[105]
			}
			if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != scm.LocNone {
				d106 = ps.OverlayValues[106]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&idxInt)
			d255 = idxInt
			_ = d255
			r246 := idxInt.Loc == scm.LocReg
			r247 := idxInt.Reg
			if r246 { ctx.ProtectReg(r247) }
			d256 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			lbl72 := ctx.W.ReserveLabel()
			bbpos_7_0 := int32(-1)
			_ = bbpos_7_0
			bbpos_7_1 := int32(-1)
			_ = bbpos_7_1
			bbpos_7_2 := int32(-1)
			_ = bbpos_7_2
			bbpos_7_3 := int32(-1)
			_ = bbpos_7_3
			bbpos_7_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			d256 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			ctx.EnsureDesc(&d255)
			ctx.EnsureDesc(&d255)
			var d257 scm.JITValueDesc
			if d255.Loc == scm.LocImm {
				d257 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d255.Imm.Int()))))}
			} else {
				r248 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r248, d255.Reg)
				ctx.W.EmitShlRegImm8(r248, 32)
				ctx.W.EmitShrRegImm8(r248, 32)
				d257 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r248}
				ctx.BindReg(r248, &d257)
			}
			var d258 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d258 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
				r249 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r249, thisptr.Reg, off)
				d258 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r249}
				ctx.BindReg(r249, &d258)
			}
			ctx.EnsureDesc(&d258)
			ctx.EnsureDesc(&d258)
			var d259 scm.JITValueDesc
			if d258.Loc == scm.LocImm {
				d259 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d258.Imm.Int()))))}
			} else {
				r250 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r250, d258.Reg)
				ctx.W.EmitShlRegImm8(r250, 56)
				ctx.W.EmitShrRegImm8(r250, 56)
				d259 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r250}
				ctx.BindReg(r250, &d259)
			}
			ctx.FreeDesc(&d258)
			ctx.EnsureDesc(&d257)
			ctx.EnsureDesc(&d259)
			ctx.EnsureDesc(&d257)
			ctx.EnsureDesc(&d259)
			ctx.EnsureDesc(&d257)
			ctx.EnsureDesc(&d259)
			var d260 scm.JITValueDesc
			if d257.Loc == scm.LocImm && d259.Loc == scm.LocImm {
				d260 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d257.Imm.Int() * d259.Imm.Int())}
			} else if d257.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d259.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d257.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d259.Reg)
				d260 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d260)
			} else if d259.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d257.Reg)
				ctx.W.EmitMovRegReg(scratch, d257.Reg)
				if d259.Imm.Int() >= -2147483648 && d259.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d259.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d259.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d260 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d260)
			} else {
				r251 := ctx.AllocRegExcept(d257.Reg, d259.Reg)
				ctx.W.EmitMovRegReg(r251, d257.Reg)
				ctx.W.EmitImulInt64(r251, d259.Reg)
				d260 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r251}
				ctx.BindReg(r251, &d260)
			}
			if d260.Loc == scm.LocReg && d257.Loc == scm.LocReg && d260.Reg == d257.Reg {
				ctx.TransferReg(d257.Reg)
				d257.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d257)
			ctx.FreeDesc(&d259)
			var d261 scm.JITValueDesc
			r252 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r252, uint64(dataPtr))
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r252, StackOff: int32(sliceLen)}
				ctx.BindReg(r252, &d261)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 0)
				ctx.W.EmitMovRegMem(r252, thisptr.Reg, off)
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r252}
				ctx.BindReg(r252, &d261)
			}
			ctx.BindReg(r252, &d261)
			ctx.EnsureDesc(&d260)
			var d262 scm.JITValueDesc
			if d260.Loc == scm.LocImm {
				d262 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d260.Imm.Int() / 64)}
			} else {
				r253 := ctx.AllocRegExcept(d260.Reg)
				ctx.W.EmitMovRegReg(r253, d260.Reg)
				ctx.W.EmitShrRegImm8(r253, 6)
				d262 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r253}
				ctx.BindReg(r253, &d262)
			}
			if d262.Loc == scm.LocReg && d260.Loc == scm.LocReg && d262.Reg == d260.Reg {
				ctx.TransferReg(d260.Reg)
				d260.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d262)
			r254 := ctx.AllocReg()
			ctx.EnsureDesc(&d262)
			ctx.EnsureDesc(&d261)
			if d262.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r254, uint64(d262.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r254, d262.Reg)
				ctx.W.EmitShlRegImm8(r254, 3)
			}
			if d261.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d261.Imm.Int()))
				ctx.W.EmitAddInt64(r254, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r254, d261.Reg)
			}
			r255 := ctx.AllocRegExcept(r254)
			ctx.W.EmitMovRegMem(r255, r254, 0)
			ctx.FreeReg(r254)
			d263 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r255}
			ctx.BindReg(r255, &d263)
			ctx.FreeDesc(&d262)
			ctx.EnsureDesc(&d260)
			var d264 scm.JITValueDesc
			if d260.Loc == scm.LocImm {
				d264 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d260.Imm.Int() % 64)}
			} else {
				r256 := ctx.AllocRegExcept(d260.Reg)
				ctx.W.EmitMovRegReg(r256, d260.Reg)
				ctx.W.EmitAndRegImm32(r256, 63)
				d264 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r256}
				ctx.BindReg(r256, &d264)
			}
			if d264.Loc == scm.LocReg && d260.Loc == scm.LocReg && d264.Reg == d260.Reg {
				ctx.TransferReg(d260.Reg)
				d260.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d263)
			ctx.EnsureDesc(&d264)
			var d265 scm.JITValueDesc
			if d263.Loc == scm.LocImm && d264.Loc == scm.LocImm {
				d265 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d263.Imm.Int()) << uint64(d264.Imm.Int())))}
			} else if d264.Loc == scm.LocImm {
				r257 := ctx.AllocRegExcept(d263.Reg)
				ctx.W.EmitMovRegReg(r257, d263.Reg)
				ctx.W.EmitShlRegImm8(r257, uint8(d264.Imm.Int()))
				d265 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r257}
				ctx.BindReg(r257, &d265)
			} else {
				{
					shiftSrc := d263.Reg
					r258 := ctx.AllocRegExcept(d263.Reg)
					ctx.W.EmitMovRegReg(r258, d263.Reg)
					shiftSrc = r258
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d264.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d264.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d264.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d265 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d265)
				}
			}
			if d265.Loc == scm.LocReg && d263.Loc == scm.LocReg && d265.Reg == d263.Reg {
				ctx.TransferReg(d263.Reg)
				d263.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d263)
			ctx.FreeDesc(&d264)
			var d266 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d266 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 25)
				r259 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r259, thisptr.Reg, off)
				d266 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r259}
				ctx.BindReg(r259, &d266)
			}
			d267 = d266
			ctx.EnsureDesc(&d267)
			if d267.Loc != scm.LocImm && d267.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl73 := ctx.W.ReserveLabel()
			lbl74 := ctx.W.ReserveLabel()
			lbl75 := ctx.W.ReserveLabel()
			lbl76 := ctx.W.ReserveLabel()
			if d267.Loc == scm.LocImm {
				if d267.Imm.Bool() {
					ctx.W.MarkLabel(lbl75)
					ctx.W.EmitJmp(lbl73)
				} else {
					ctx.W.MarkLabel(lbl76)
			d268 = d265
			if d268.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d268)
			ctx.EmitStoreToStack(d268, 40)
					ctx.W.EmitJmp(lbl74)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d267.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl75)
				ctx.W.EmitJmp(lbl76)
				ctx.W.MarkLabel(lbl75)
				ctx.W.EmitJmp(lbl73)
				ctx.W.MarkLabel(lbl76)
			d269 = d265
			if d269.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d269)
			ctx.EmitStoreToStack(d269, 40)
				ctx.W.EmitJmp(lbl74)
			}
			ctx.FreeDesc(&d266)
			bbpos_7_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl74)
			ctx.W.ResolveFixups()
			d256 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			var d270 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d270 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
				r260 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r260, thisptr.Reg, off)
				d270 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r260}
				ctx.BindReg(r260, &d270)
			}
			ctx.EnsureDesc(&d270)
			ctx.EnsureDesc(&d270)
			var d271 scm.JITValueDesc
			if d270.Loc == scm.LocImm {
				d271 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d270.Imm.Int()))))}
			} else {
				r261 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r261, d270.Reg)
				ctx.W.EmitShlRegImm8(r261, 56)
				ctx.W.EmitShrRegImm8(r261, 56)
				d271 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r261}
				ctx.BindReg(r261, &d271)
			}
			ctx.FreeDesc(&d270)
			d272 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d271)
			ctx.EnsureDesc(&d272)
			ctx.EnsureDesc(&d271)
			ctx.EnsureDesc(&d272)
			ctx.EnsureDesc(&d271)
			var d273 scm.JITValueDesc
			if d272.Loc == scm.LocImm && d271.Loc == scm.LocImm {
				d273 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d272.Imm.Int() - d271.Imm.Int())}
			} else if d271.Loc == scm.LocImm && d271.Imm.Int() == 0 {
				r262 := ctx.AllocRegExcept(d272.Reg)
				ctx.W.EmitMovRegReg(r262, d272.Reg)
				d273 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r262}
				ctx.BindReg(r262, &d273)
			} else if d272.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d271.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d272.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d271.Reg)
				d273 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d273)
			} else if d271.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d272.Reg)
				ctx.W.EmitMovRegReg(scratch, d272.Reg)
				if d271.Imm.Int() >= -2147483648 && d271.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d271.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d271.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d273 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d273)
			} else {
				r263 := ctx.AllocRegExcept(d272.Reg, d271.Reg)
				ctx.W.EmitMovRegReg(r263, d272.Reg)
				ctx.W.EmitSubInt64(r263, d271.Reg)
				d273 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r263}
				ctx.BindReg(r263, &d273)
			}
			if d273.Loc == scm.LocReg && d272.Loc == scm.LocReg && d273.Reg == d272.Reg {
				ctx.TransferReg(d272.Reg)
				d272.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d271)
			ctx.EnsureDesc(&d256)
			ctx.EnsureDesc(&d273)
			var d274 scm.JITValueDesc
			if d256.Loc == scm.LocImm && d273.Loc == scm.LocImm {
				d274 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d256.Imm.Int()) >> uint64(d273.Imm.Int())))}
			} else if d273.Loc == scm.LocImm {
				r264 := ctx.AllocRegExcept(d256.Reg)
				ctx.W.EmitMovRegReg(r264, d256.Reg)
				ctx.W.EmitShrRegImm8(r264, uint8(d273.Imm.Int()))
				d274 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r264}
				ctx.BindReg(r264, &d274)
			} else {
				{
					shiftSrc := d256.Reg
					r265 := ctx.AllocRegExcept(d256.Reg)
					ctx.W.EmitMovRegReg(r265, d256.Reg)
					shiftSrc = r265
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d273.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d273.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d273.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d274 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d274)
				}
			}
			if d274.Loc == scm.LocReg && d256.Loc == scm.LocReg && d274.Reg == d256.Reg {
				ctx.TransferReg(d256.Reg)
				d256.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d256)
			ctx.FreeDesc(&d273)
			r266 := ctx.AllocReg()
			ctx.EnsureDesc(&d274)
			ctx.EnsureDesc(&d274)
			if d274.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r266, d274)
			}
			ctx.W.EmitJmp(lbl72)
			bbpos_7_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl73)
			ctx.W.ResolveFixups()
			d256 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			ctx.EnsureDesc(&d260)
			var d275 scm.JITValueDesc
			if d260.Loc == scm.LocImm {
				d275 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d260.Imm.Int() % 64)}
			} else {
				r267 := ctx.AllocRegExcept(d260.Reg)
				ctx.W.EmitMovRegReg(r267, d260.Reg)
				ctx.W.EmitAndRegImm32(r267, 63)
				d275 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r267}
				ctx.BindReg(r267, &d275)
			}
			if d275.Loc == scm.LocReg && d260.Loc == scm.LocReg && d275.Reg == d260.Reg {
				ctx.TransferReg(d260.Reg)
				d260.Loc = scm.LocNone
			}
			var d276 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d276 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
				r268 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r268, thisptr.Reg, off)
				d276 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r268}
				ctx.BindReg(r268, &d276)
			}
			ctx.EnsureDesc(&d276)
			ctx.EnsureDesc(&d276)
			var d277 scm.JITValueDesc
			if d276.Loc == scm.LocImm {
				d277 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d276.Imm.Int()))))}
			} else {
				r269 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r269, d276.Reg)
				ctx.W.EmitShlRegImm8(r269, 56)
				ctx.W.EmitShrRegImm8(r269, 56)
				d277 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r269}
				ctx.BindReg(r269, &d277)
			}
			ctx.FreeDesc(&d276)
			ctx.EnsureDesc(&d275)
			ctx.EnsureDesc(&d277)
			ctx.EnsureDesc(&d275)
			ctx.EnsureDesc(&d277)
			ctx.EnsureDesc(&d275)
			ctx.EnsureDesc(&d277)
			var d278 scm.JITValueDesc
			if d275.Loc == scm.LocImm && d277.Loc == scm.LocImm {
				d278 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d275.Imm.Int() + d277.Imm.Int())}
			} else if d277.Loc == scm.LocImm && d277.Imm.Int() == 0 {
				r270 := ctx.AllocRegExcept(d275.Reg)
				ctx.W.EmitMovRegReg(r270, d275.Reg)
				d278 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r270}
				ctx.BindReg(r270, &d278)
			} else if d275.Loc == scm.LocImm && d275.Imm.Int() == 0 {
				d278 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d277.Reg}
				ctx.BindReg(d277.Reg, &d278)
			} else if d275.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d277.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d275.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d277.Reg)
				d278 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d278)
			} else if d277.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d275.Reg)
				ctx.W.EmitMovRegReg(scratch, d275.Reg)
				if d277.Imm.Int() >= -2147483648 && d277.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d277.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d277.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d278 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d278)
			} else {
				r271 := ctx.AllocRegExcept(d275.Reg, d277.Reg)
				ctx.W.EmitMovRegReg(r271, d275.Reg)
				ctx.W.EmitAddInt64(r271, d277.Reg)
				d278 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r271}
				ctx.BindReg(r271, &d278)
			}
			if d278.Loc == scm.LocReg && d275.Loc == scm.LocReg && d278.Reg == d275.Reg {
				ctx.TransferReg(d275.Reg)
				d275.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d275)
			ctx.FreeDesc(&d277)
			ctx.EnsureDesc(&d278)
			var d279 scm.JITValueDesc
			if d278.Loc == scm.LocImm {
				d279 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d278.Imm.Int()) > uint64(64))}
			} else {
				r272 := ctx.AllocRegExcept(d278.Reg)
				ctx.W.EmitCmpRegImm32(d278.Reg, 64)
				ctx.W.EmitSetcc(r272, scm.CcA)
				d279 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r272}
				ctx.BindReg(r272, &d279)
			}
			ctx.FreeDesc(&d278)
			d280 = d279
			ctx.EnsureDesc(&d280)
			if d280.Loc != scm.LocImm && d280.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl77 := ctx.W.ReserveLabel()
			lbl78 := ctx.W.ReserveLabel()
			lbl79 := ctx.W.ReserveLabel()
			if d280.Loc == scm.LocImm {
				if d280.Imm.Bool() {
					ctx.W.MarkLabel(lbl78)
					ctx.W.EmitJmp(lbl77)
				} else {
					ctx.W.MarkLabel(lbl79)
			d281 = d265
			if d281.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d281)
			ctx.EmitStoreToStack(d281, 40)
					ctx.W.EmitJmp(lbl74)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d280.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl78)
				ctx.W.EmitJmp(lbl79)
				ctx.W.MarkLabel(lbl78)
				ctx.W.EmitJmp(lbl77)
				ctx.W.MarkLabel(lbl79)
			d282 = d265
			if d282.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d282)
			ctx.EmitStoreToStack(d282, 40)
				ctx.W.EmitJmp(lbl74)
			}
			ctx.FreeDesc(&d279)
			bbpos_7_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl77)
			ctx.W.ResolveFixups()
			d256 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			ctx.EnsureDesc(&d260)
			var d283 scm.JITValueDesc
			if d260.Loc == scm.LocImm {
				d283 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d260.Imm.Int() / 64)}
			} else {
				r273 := ctx.AllocRegExcept(d260.Reg)
				ctx.W.EmitMovRegReg(r273, d260.Reg)
				ctx.W.EmitShrRegImm8(r273, 6)
				d283 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r273}
				ctx.BindReg(r273, &d283)
			}
			if d283.Loc == scm.LocReg && d260.Loc == scm.LocReg && d283.Reg == d260.Reg {
				ctx.TransferReg(d260.Reg)
				d260.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d283)
			ctx.EnsureDesc(&d283)
			var d284 scm.JITValueDesc
			if d283.Loc == scm.LocImm {
				d284 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d283.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d283.Reg)
				ctx.W.EmitMovRegReg(scratch, d283.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d284 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d284)
			}
			if d284.Loc == scm.LocReg && d283.Loc == scm.LocReg && d284.Reg == d283.Reg {
				ctx.TransferReg(d283.Reg)
				d283.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d283)
			ctx.EnsureDesc(&d284)
			r274 := ctx.AllocReg()
			ctx.EnsureDesc(&d284)
			ctx.EnsureDesc(&d261)
			if d284.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r274, uint64(d284.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r274, d284.Reg)
				ctx.W.EmitShlRegImm8(r274, 3)
			}
			if d261.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d261.Imm.Int()))
				ctx.W.EmitAddInt64(r274, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r274, d261.Reg)
			}
			r275 := ctx.AllocRegExcept(r274)
			ctx.W.EmitMovRegMem(r275, r274, 0)
			ctx.FreeReg(r274)
			d285 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r275}
			ctx.BindReg(r275, &d285)
			ctx.FreeDesc(&d284)
			ctx.EnsureDesc(&d260)
			var d286 scm.JITValueDesc
			if d260.Loc == scm.LocImm {
				d286 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d260.Imm.Int() % 64)}
			} else {
				r276 := ctx.AllocRegExcept(d260.Reg)
				ctx.W.EmitMovRegReg(r276, d260.Reg)
				ctx.W.EmitAndRegImm32(r276, 63)
				d286 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r276}
				ctx.BindReg(r276, &d286)
			}
			if d286.Loc == scm.LocReg && d260.Loc == scm.LocReg && d286.Reg == d260.Reg {
				ctx.TransferReg(d260.Reg)
				d260.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d260)
			d287 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d286)
			ctx.EnsureDesc(&d287)
			ctx.EnsureDesc(&d286)
			ctx.EnsureDesc(&d287)
			ctx.EnsureDesc(&d286)
			var d288 scm.JITValueDesc
			if d287.Loc == scm.LocImm && d286.Loc == scm.LocImm {
				d288 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d287.Imm.Int() - d286.Imm.Int())}
			} else if d286.Loc == scm.LocImm && d286.Imm.Int() == 0 {
				r277 := ctx.AllocRegExcept(d287.Reg)
				ctx.W.EmitMovRegReg(r277, d287.Reg)
				d288 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r277}
				ctx.BindReg(r277, &d288)
			} else if d287.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d286.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d287.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d286.Reg)
				d288 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d288)
			} else if d286.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d287.Reg)
				ctx.W.EmitMovRegReg(scratch, d287.Reg)
				if d286.Imm.Int() >= -2147483648 && d286.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d286.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d286.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d288 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d288)
			} else {
				r278 := ctx.AllocRegExcept(d287.Reg, d286.Reg)
				ctx.W.EmitMovRegReg(r278, d287.Reg)
				ctx.W.EmitSubInt64(r278, d286.Reg)
				d288 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r278}
				ctx.BindReg(r278, &d288)
			}
			if d288.Loc == scm.LocReg && d287.Loc == scm.LocReg && d288.Reg == d287.Reg {
				ctx.TransferReg(d287.Reg)
				d287.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d286)
			ctx.EnsureDesc(&d285)
			ctx.EnsureDesc(&d288)
			var d289 scm.JITValueDesc
			if d285.Loc == scm.LocImm && d288.Loc == scm.LocImm {
				d289 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d285.Imm.Int()) >> uint64(d288.Imm.Int())))}
			} else if d288.Loc == scm.LocImm {
				r279 := ctx.AllocRegExcept(d285.Reg)
				ctx.W.EmitMovRegReg(r279, d285.Reg)
				ctx.W.EmitShrRegImm8(r279, uint8(d288.Imm.Int()))
				d289 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r279}
				ctx.BindReg(r279, &d289)
			} else {
				{
					shiftSrc := d285.Reg
					r280 := ctx.AllocRegExcept(d285.Reg)
					ctx.W.EmitMovRegReg(r280, d285.Reg)
					shiftSrc = r280
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d288.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d288.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d288.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d289 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d289)
				}
			}
			if d289.Loc == scm.LocReg && d285.Loc == scm.LocReg && d289.Reg == d285.Reg {
				ctx.TransferReg(d285.Reg)
				d285.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d285)
			ctx.FreeDesc(&d288)
			ctx.EnsureDesc(&d265)
			ctx.EnsureDesc(&d289)
			var d290 scm.JITValueDesc
			if d265.Loc == scm.LocImm && d289.Loc == scm.LocImm {
				d290 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d265.Imm.Int() | d289.Imm.Int())}
			} else if d265.Loc == scm.LocImm && d265.Imm.Int() == 0 {
				d290 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d289.Reg}
				ctx.BindReg(d289.Reg, &d290)
			} else if d289.Loc == scm.LocImm && d289.Imm.Int() == 0 {
				r281 := ctx.AllocRegExcept(d265.Reg)
				ctx.W.EmitMovRegReg(r281, d265.Reg)
				d290 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r281}
				ctx.BindReg(r281, &d290)
			} else if d265.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d289.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d265.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d289.Reg)
				d290 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d290)
			} else if d289.Loc == scm.LocImm {
				r282 := ctx.AllocRegExcept(d265.Reg)
				ctx.W.EmitMovRegReg(r282, d265.Reg)
				if d289.Imm.Int() >= -2147483648 && d289.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r282, int32(d289.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d289.Imm.Int()))
					ctx.W.EmitOrInt64(r282, scm.RegR11)
				}
				d290 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r282}
				ctx.BindReg(r282, &d290)
			} else {
				r283 := ctx.AllocRegExcept(d265.Reg, d289.Reg)
				ctx.W.EmitMovRegReg(r283, d265.Reg)
				ctx.W.EmitOrInt64(r283, d289.Reg)
				d290 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r283}
				ctx.BindReg(r283, &d290)
			}
			if d290.Loc == scm.LocReg && d265.Loc == scm.LocReg && d290.Reg == d265.Reg {
				ctx.TransferReg(d265.Reg)
				d265.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d289)
			d291 = d290
			if d291.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d291)
			ctx.EmitStoreToStack(d291, 40)
			ctx.W.EmitJmp(lbl74)
			ctx.W.MarkLabel(lbl72)
			d292 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r266}
			ctx.BindReg(r266, &d292)
			ctx.BindReg(r266, &d292)
			if r246 { ctx.UnprotectReg(r247) }
			ctx.FreeDesc(&idxInt)
			ctx.EnsureDesc(&d292)
			ctx.EnsureDesc(&d292)
			var d293 scm.JITValueDesc
			if d292.Loc == scm.LocImm {
				d293 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d292.Imm.Int()))))}
			} else {
				r284 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r284, d292.Reg)
				d293 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r284}
				ctx.BindReg(r284, &d293)
			}
			ctx.FreeDesc(&d292)
			var d294 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d294 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 32)
				r285 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r285, thisptr.Reg, off)
				d294 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r285}
				ctx.BindReg(r285, &d294)
			}
			ctx.EnsureDesc(&d293)
			ctx.EnsureDesc(&d294)
			ctx.EnsureDesc(&d293)
			ctx.EnsureDesc(&d294)
			ctx.EnsureDesc(&d293)
			ctx.EnsureDesc(&d294)
			var d295 scm.JITValueDesc
			if d293.Loc == scm.LocImm && d294.Loc == scm.LocImm {
				d295 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d293.Imm.Int() + d294.Imm.Int())}
			} else if d294.Loc == scm.LocImm && d294.Imm.Int() == 0 {
				r286 := ctx.AllocRegExcept(d293.Reg)
				ctx.W.EmitMovRegReg(r286, d293.Reg)
				d295 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r286}
				ctx.BindReg(r286, &d295)
			} else if d293.Loc == scm.LocImm && d293.Imm.Int() == 0 {
				d295 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d294.Reg}
				ctx.BindReg(d294.Reg, &d295)
			} else if d293.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d294.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d293.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d294.Reg)
				d295 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d295)
			} else if d294.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d293.Reg)
				ctx.W.EmitMovRegReg(scratch, d293.Reg)
				if d294.Imm.Int() >= -2147483648 && d294.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d294.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d294.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d295 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d295)
			} else {
				r287 := ctx.AllocRegExcept(d293.Reg, d294.Reg)
				ctx.W.EmitMovRegReg(r287, d293.Reg)
				ctx.W.EmitAddInt64(r287, d294.Reg)
				d295 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r287}
				ctx.BindReg(r287, &d295)
			}
			if d295.Loc == scm.LocReg && d293.Loc == scm.LocReg && d295.Reg == d293.Reg {
				ctx.TransferReg(d293.Reg)
				d293.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d293)
			ctx.FreeDesc(&d294)
			var d296 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary)
				r288 := ctx.AllocReg()
				r289 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r288, fieldAddr)
				ctx.W.EmitMovRegMem64(r289, fieldAddr+8)
				d296 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r288, Reg2: r289}
				ctx.BindReg(r288, &d296)
				ctx.BindReg(r289, &d296)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary))
				r290 := ctx.AllocReg()
				r291 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r290, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r291, thisptr.Reg, off+8)
				d296 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r290, Reg2: r291}
				ctx.BindReg(r290, &d296)
				ctx.BindReg(r291, &d296)
			}
			var d297 scm.JITValueDesc
			if d296.Loc == scm.LocImm {
				d297 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d296.StackOff))}
			} else {
				ctx.EnsureDesc(&d296)
				if d296.Loc == scm.LocRegPair {
					d297 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d296.Reg2}
					ctx.BindReg(d296.Reg2, &d297)
					ctx.BindReg(d296.Reg2, &d297)
				} else if d296.Loc == scm.LocReg {
					d297 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d296.Reg}
					ctx.BindReg(d296.Reg, &d297)
					ctx.BindReg(d296.Reg, &d297)
				} else {
					panic("len on unsupported descriptor location")
				}
			}
			ctx.EnsureDesc(&d297)
			ctx.EnsureDesc(&d297)
			ctx.EnsureDesc(&d295)
			ctx.EnsureDesc(&d297)
			ctx.EnsureDesc(&d295)
			ctx.EnsureDesc(&d297)
			ctx.EnsureDesc(&d295)
			ctx.EnsureDesc(&d297)
			var d299 scm.JITValueDesc
			if d295.Loc == scm.LocImm && d297.Loc == scm.LocImm {
				d299 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d295.Imm.Int() >= d297.Imm.Int())}
			} else if d297.Loc == scm.LocImm {
				r292 := ctx.AllocRegExcept(d295.Reg)
				if d297.Imm.Int() >= -2147483648 && d297.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d295.Reg, int32(d297.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d297.Imm.Int()))
					ctx.W.EmitCmpInt64(d295.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r292, scm.CcGE)
				d299 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r292}
				ctx.BindReg(r292, &d299)
			} else if d295.Loc == scm.LocImm {
				r293 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d295.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d297.Reg)
				ctx.W.EmitSetcc(r293, scm.CcGE)
				d299 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r293}
				ctx.BindReg(r293, &d299)
			} else {
				r294 := ctx.AllocRegExcept(d295.Reg)
				ctx.W.EmitCmpInt64(d295.Reg, d297.Reg)
				ctx.W.EmitSetcc(r294, scm.CcGE)
				d299 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r294}
				ctx.BindReg(r294, &d299)
			}
			ctx.FreeDesc(&d297)
			d300 = d299
			ctx.EnsureDesc(&d300)
			if d300.Loc != scm.LocImm && d300.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d300.Loc == scm.LocImm {
				if d300.Imm.Bool() {
			ps301 := scm.PhiState{General: ps.General}
			ps301.OverlayValues = make([]scm.JITValueDesc, 301)
			ps301.OverlayValues[0] = d0
			ps301.OverlayValues[1] = d1
			ps301.OverlayValues[2] = d2
			ps301.OverlayValues[3] = d3
			ps301.OverlayValues[4] = d4
			ps301.OverlayValues[5] = d5
			ps301.OverlayValues[6] = d6
			ps301.OverlayValues[7] = d7
			ps301.OverlayValues[8] = d8
			ps301.OverlayValues[9] = d9
			ps301.OverlayValues[10] = d10
			ps301.OverlayValues[11] = d11
			ps301.OverlayValues[12] = d12
			ps301.OverlayValues[13] = d13
			ps301.OverlayValues[14] = d14
			ps301.OverlayValues[15] = d15
			ps301.OverlayValues[16] = d16
			ps301.OverlayValues[17] = d17
			ps301.OverlayValues[18] = d18
			ps301.OverlayValues[19] = d19
			ps301.OverlayValues[20] = d20
			ps301.OverlayValues[21] = d21
			ps301.OverlayValues[22] = d22
			ps301.OverlayValues[23] = d23
			ps301.OverlayValues[24] = d24
			ps301.OverlayValues[25] = d25
			ps301.OverlayValues[26] = d26
			ps301.OverlayValues[27] = d27
			ps301.OverlayValues[28] = d28
			ps301.OverlayValues[29] = d29
			ps301.OverlayValues[30] = d30
			ps301.OverlayValues[31] = d31
			ps301.OverlayValues[32] = d32
			ps301.OverlayValues[33] = d33
			ps301.OverlayValues[34] = d34
			ps301.OverlayValues[35] = d35
			ps301.OverlayValues[36] = d36
			ps301.OverlayValues[37] = d37
			ps301.OverlayValues[38] = d38
			ps301.OverlayValues[39] = d39
			ps301.OverlayValues[40] = d40
			ps301.OverlayValues[41] = d41
			ps301.OverlayValues[42] = d42
			ps301.OverlayValues[43] = d43
			ps301.OverlayValues[44] = d44
			ps301.OverlayValues[45] = d45
			ps301.OverlayValues[46] = d46
			ps301.OverlayValues[47] = d47
			ps301.OverlayValues[48] = d48
			ps301.OverlayValues[49] = d49
			ps301.OverlayValues[50] = d50
			ps301.OverlayValues[51] = d51
			ps301.OverlayValues[52] = d52
			ps301.OverlayValues[53] = d53
			ps301.OverlayValues[54] = d54
			ps301.OverlayValues[55] = d55
			ps301.OverlayValues[56] = d56
			ps301.OverlayValues[57] = d57
			ps301.OverlayValues[58] = d58
			ps301.OverlayValues[59] = d59
			ps301.OverlayValues[60] = d60
			ps301.OverlayValues[61] = d61
			ps301.OverlayValues[62] = d62
			ps301.OverlayValues[63] = d63
			ps301.OverlayValues[64] = d64
			ps301.OverlayValues[65] = d65
			ps301.OverlayValues[66] = d66
			ps301.OverlayValues[67] = d67
			ps301.OverlayValues[68] = d68
			ps301.OverlayValues[69] = d69
			ps301.OverlayValues[70] = d70
			ps301.OverlayValues[71] = d71
			ps301.OverlayValues[72] = d72
			ps301.OverlayValues[73] = d73
			ps301.OverlayValues[74] = d74
			ps301.OverlayValues[75] = d75
			ps301.OverlayValues[76] = d76
			ps301.OverlayValues[77] = d77
			ps301.OverlayValues[78] = d78
			ps301.OverlayValues[79] = d79
			ps301.OverlayValues[80] = d80
			ps301.OverlayValues[81] = d81
			ps301.OverlayValues[82] = d82
			ps301.OverlayValues[83] = d83
			ps301.OverlayValues[84] = d84
			ps301.OverlayValues[85] = d85
			ps301.OverlayValues[86] = d86
			ps301.OverlayValues[87] = d87
			ps301.OverlayValues[88] = d88
			ps301.OverlayValues[89] = d89
			ps301.OverlayValues[90] = d90
			ps301.OverlayValues[91] = d91
			ps301.OverlayValues[92] = d92
			ps301.OverlayValues[93] = d93
			ps301.OverlayValues[94] = d94
			ps301.OverlayValues[95] = d95
			ps301.OverlayValues[96] = d96
			ps301.OverlayValues[97] = d97
			ps301.OverlayValues[98] = d98
			ps301.OverlayValues[99] = d99
			ps301.OverlayValues[100] = d100
			ps301.OverlayValues[101] = d101
			ps301.OverlayValues[102] = d102
			ps301.OverlayValues[103] = d103
			ps301.OverlayValues[104] = d104
			ps301.OverlayValues[105] = d105
			ps301.OverlayValues[106] = d106
			ps301.OverlayValues[107] = d107
			ps301.OverlayValues[108] = d108
			ps301.OverlayValues[109] = d109
			ps301.OverlayValues[110] = d110
			ps301.OverlayValues[111] = d111
			ps301.OverlayValues[112] = d112
			ps301.OverlayValues[113] = d113
			ps301.OverlayValues[114] = d114
			ps301.OverlayValues[115] = d115
			ps301.OverlayValues[116] = d116
			ps301.OverlayValues[117] = d117
			ps301.OverlayValues[118] = d118
			ps301.OverlayValues[119] = d119
			ps301.OverlayValues[120] = d120
			ps301.OverlayValues[121] = d121
			ps301.OverlayValues[122] = d122
			ps301.OverlayValues[123] = d123
			ps301.OverlayValues[124] = d124
			ps301.OverlayValues[125] = d125
			ps301.OverlayValues[126] = d126
			ps301.OverlayValues[127] = d127
			ps301.OverlayValues[128] = d128
			ps301.OverlayValues[129] = d129
			ps301.OverlayValues[130] = d130
			ps301.OverlayValues[131] = d131
			ps301.OverlayValues[132] = d132
			ps301.OverlayValues[133] = d133
			ps301.OverlayValues[134] = d134
			ps301.OverlayValues[135] = d135
			ps301.OverlayValues[136] = d136
			ps301.OverlayValues[137] = d137
			ps301.OverlayValues[138] = d138
			ps301.OverlayValues[139] = d139
			ps301.OverlayValues[140] = d140
			ps301.OverlayValues[141] = d141
			ps301.OverlayValues[142] = d142
			ps301.OverlayValues[143] = d143
			ps301.OverlayValues[144] = d144
			ps301.OverlayValues[145] = d145
			ps301.OverlayValues[146] = d146
			ps301.OverlayValues[147] = d147
			ps301.OverlayValues[148] = d148
			ps301.OverlayValues[149] = d149
			ps301.OverlayValues[150] = d150
			ps301.OverlayValues[151] = d151
			ps301.OverlayValues[152] = d152
			ps301.OverlayValues[153] = d153
			ps301.OverlayValues[154] = d154
			ps301.OverlayValues[155] = d155
			ps301.OverlayValues[156] = d156
			ps301.OverlayValues[157] = d157
			ps301.OverlayValues[158] = d158
			ps301.OverlayValues[159] = d159
			ps301.OverlayValues[160] = d160
			ps301.OverlayValues[161] = d161
			ps301.OverlayValues[162] = d162
			ps301.OverlayValues[163] = d163
			ps301.OverlayValues[164] = d164
			ps301.OverlayValues[165] = d165
			ps301.OverlayValues[166] = d166
			ps301.OverlayValues[167] = d167
			ps301.OverlayValues[168] = d168
			ps301.OverlayValues[169] = d169
			ps301.OverlayValues[170] = d170
			ps301.OverlayValues[171] = d171
			ps301.OverlayValues[172] = d172
			ps301.OverlayValues[173] = d173
			ps301.OverlayValues[174] = d174
			ps301.OverlayValues[175] = d175
			ps301.OverlayValues[176] = d176
			ps301.OverlayValues[177] = d177
			ps301.OverlayValues[178] = d178
			ps301.OverlayValues[179] = d179
			ps301.OverlayValues[180] = d180
			ps301.OverlayValues[181] = d181
			ps301.OverlayValues[182] = d182
			ps301.OverlayValues[183] = d183
			ps301.OverlayValues[184] = d184
			ps301.OverlayValues[185] = d185
			ps301.OverlayValues[186] = d186
			ps301.OverlayValues[187] = d187
			ps301.OverlayValues[188] = d188
			ps301.OverlayValues[189] = d189
			ps301.OverlayValues[190] = d190
			ps301.OverlayValues[191] = d191
			ps301.OverlayValues[192] = d192
			ps301.OverlayValues[193] = d193
			ps301.OverlayValues[194] = d194
			ps301.OverlayValues[195] = d195
			ps301.OverlayValues[196] = d196
			ps301.OverlayValues[197] = d197
			ps301.OverlayValues[198] = d198
			ps301.OverlayValues[199] = d199
			ps301.OverlayValues[200] = d200
			ps301.OverlayValues[201] = d201
			ps301.OverlayValues[202] = d202
			ps301.OverlayValues[203] = d203
			ps301.OverlayValues[204] = d204
			ps301.OverlayValues[205] = d205
			ps301.OverlayValues[206] = d206
			ps301.OverlayValues[207] = d207
			ps301.OverlayValues[208] = d208
			ps301.OverlayValues[209] = d209
			ps301.OverlayValues[210] = d210
			ps301.OverlayValues[211] = d211
			ps301.OverlayValues[212] = d212
			ps301.OverlayValues[213] = d213
			ps301.OverlayValues[214] = d214
			ps301.OverlayValues[215] = d215
			ps301.OverlayValues[216] = d216
			ps301.OverlayValues[217] = d217
			ps301.OverlayValues[218] = d218
			ps301.OverlayValues[219] = d219
			ps301.OverlayValues[220] = d220
			ps301.OverlayValues[221] = d221
			ps301.OverlayValues[222] = d222
			ps301.OverlayValues[223] = d223
			ps301.OverlayValues[224] = d224
			ps301.OverlayValues[225] = d225
			ps301.OverlayValues[226] = d226
			ps301.OverlayValues[227] = d227
			ps301.OverlayValues[228] = d228
			ps301.OverlayValues[229] = d229
			ps301.OverlayValues[230] = d230
			ps301.OverlayValues[231] = d231
			ps301.OverlayValues[232] = d232
			ps301.OverlayValues[233] = d233
			ps301.OverlayValues[234] = d234
			ps301.OverlayValues[235] = d235
			ps301.OverlayValues[236] = d236
			ps301.OverlayValues[237] = d237
			ps301.OverlayValues[238] = d238
			ps301.OverlayValues[239] = d239
			ps301.OverlayValues[240] = d240
			ps301.OverlayValues[246] = d246
			ps301.OverlayValues[247] = d247
			ps301.OverlayValues[248] = d248
			ps301.OverlayValues[249] = d249
			ps301.OverlayValues[255] = d255
			ps301.OverlayValues[256] = d256
			ps301.OverlayValues[257] = d257
			ps301.OverlayValues[258] = d258
			ps301.OverlayValues[259] = d259
			ps301.OverlayValues[260] = d260
			ps301.OverlayValues[261] = d261
			ps301.OverlayValues[262] = d262
			ps301.OverlayValues[263] = d263
			ps301.OverlayValues[264] = d264
			ps301.OverlayValues[265] = d265
			ps301.OverlayValues[266] = d266
			ps301.OverlayValues[267] = d267
			ps301.OverlayValues[268] = d268
			ps301.OverlayValues[269] = d269
			ps301.OverlayValues[270] = d270
			ps301.OverlayValues[271] = d271
			ps301.OverlayValues[272] = d272
			ps301.OverlayValues[273] = d273
			ps301.OverlayValues[274] = d274
			ps301.OverlayValues[275] = d275
			ps301.OverlayValues[276] = d276
			ps301.OverlayValues[277] = d277
			ps301.OverlayValues[278] = d278
			ps301.OverlayValues[279] = d279
			ps301.OverlayValues[280] = d280
			ps301.OverlayValues[281] = d281
			ps301.OverlayValues[282] = d282
			ps301.OverlayValues[283] = d283
			ps301.OverlayValues[284] = d284
			ps301.OverlayValues[285] = d285
			ps301.OverlayValues[286] = d286
			ps301.OverlayValues[287] = d287
			ps301.OverlayValues[288] = d288
			ps301.OverlayValues[289] = d289
			ps301.OverlayValues[290] = d290
			ps301.OverlayValues[291] = d291
			ps301.OverlayValues[292] = d292
			ps301.OverlayValues[293] = d293
			ps301.OverlayValues[294] = d294
			ps301.OverlayValues[295] = d295
			ps301.OverlayValues[296] = d296
			ps301.OverlayValues[297] = d297
			ps301.OverlayValues[298] = d298
			ps301.OverlayValues[299] = d299
			ps301.OverlayValues[300] = d300
					return bbs[5].RenderPS(ps301)
				}
			ps302 := scm.PhiState{General: ps.General}
			ps302.OverlayValues = make([]scm.JITValueDesc, 301)
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
			ps302.OverlayValues[17] = d17
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
			ps302.OverlayValues[62] = d62
			ps302.OverlayValues[63] = d63
			ps302.OverlayValues[64] = d64
			ps302.OverlayValues[65] = d65
			ps302.OverlayValues[66] = d66
			ps302.OverlayValues[67] = d67
			ps302.OverlayValues[68] = d68
			ps302.OverlayValues[69] = d69
			ps302.OverlayValues[70] = d70
			ps302.OverlayValues[71] = d71
			ps302.OverlayValues[72] = d72
			ps302.OverlayValues[73] = d73
			ps302.OverlayValues[74] = d74
			ps302.OverlayValues[75] = d75
			ps302.OverlayValues[76] = d76
			ps302.OverlayValues[77] = d77
			ps302.OverlayValues[78] = d78
			ps302.OverlayValues[79] = d79
			ps302.OverlayValues[80] = d80
			ps302.OverlayValues[81] = d81
			ps302.OverlayValues[82] = d82
			ps302.OverlayValues[83] = d83
			ps302.OverlayValues[84] = d84
			ps302.OverlayValues[85] = d85
			ps302.OverlayValues[86] = d86
			ps302.OverlayValues[87] = d87
			ps302.OverlayValues[88] = d88
			ps302.OverlayValues[89] = d89
			ps302.OverlayValues[90] = d90
			ps302.OverlayValues[91] = d91
			ps302.OverlayValues[92] = d92
			ps302.OverlayValues[93] = d93
			ps302.OverlayValues[94] = d94
			ps302.OverlayValues[95] = d95
			ps302.OverlayValues[96] = d96
			ps302.OverlayValues[97] = d97
			ps302.OverlayValues[98] = d98
			ps302.OverlayValues[99] = d99
			ps302.OverlayValues[100] = d100
			ps302.OverlayValues[101] = d101
			ps302.OverlayValues[102] = d102
			ps302.OverlayValues[103] = d103
			ps302.OverlayValues[104] = d104
			ps302.OverlayValues[105] = d105
			ps302.OverlayValues[106] = d106
			ps302.OverlayValues[107] = d107
			ps302.OverlayValues[108] = d108
			ps302.OverlayValues[109] = d109
			ps302.OverlayValues[110] = d110
			ps302.OverlayValues[111] = d111
			ps302.OverlayValues[112] = d112
			ps302.OverlayValues[113] = d113
			ps302.OverlayValues[114] = d114
			ps302.OverlayValues[115] = d115
			ps302.OverlayValues[116] = d116
			ps302.OverlayValues[117] = d117
			ps302.OverlayValues[118] = d118
			ps302.OverlayValues[119] = d119
			ps302.OverlayValues[120] = d120
			ps302.OverlayValues[121] = d121
			ps302.OverlayValues[122] = d122
			ps302.OverlayValues[123] = d123
			ps302.OverlayValues[124] = d124
			ps302.OverlayValues[125] = d125
			ps302.OverlayValues[126] = d126
			ps302.OverlayValues[127] = d127
			ps302.OverlayValues[128] = d128
			ps302.OverlayValues[129] = d129
			ps302.OverlayValues[130] = d130
			ps302.OverlayValues[131] = d131
			ps302.OverlayValues[132] = d132
			ps302.OverlayValues[133] = d133
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
			ps302.OverlayValues[173] = d173
			ps302.OverlayValues[174] = d174
			ps302.OverlayValues[175] = d175
			ps302.OverlayValues[176] = d176
			ps302.OverlayValues[177] = d177
			ps302.OverlayValues[178] = d178
			ps302.OverlayValues[179] = d179
			ps302.OverlayValues[180] = d180
			ps302.OverlayValues[181] = d181
			ps302.OverlayValues[182] = d182
			ps302.OverlayValues[183] = d183
			ps302.OverlayValues[184] = d184
			ps302.OverlayValues[185] = d185
			ps302.OverlayValues[186] = d186
			ps302.OverlayValues[187] = d187
			ps302.OverlayValues[188] = d188
			ps302.OverlayValues[189] = d189
			ps302.OverlayValues[190] = d190
			ps302.OverlayValues[191] = d191
			ps302.OverlayValues[192] = d192
			ps302.OverlayValues[193] = d193
			ps302.OverlayValues[194] = d194
			ps302.OverlayValues[195] = d195
			ps302.OverlayValues[196] = d196
			ps302.OverlayValues[197] = d197
			ps302.OverlayValues[198] = d198
			ps302.OverlayValues[199] = d199
			ps302.OverlayValues[200] = d200
			ps302.OverlayValues[201] = d201
			ps302.OverlayValues[202] = d202
			ps302.OverlayValues[203] = d203
			ps302.OverlayValues[204] = d204
			ps302.OverlayValues[205] = d205
			ps302.OverlayValues[206] = d206
			ps302.OverlayValues[207] = d207
			ps302.OverlayValues[208] = d208
			ps302.OverlayValues[209] = d209
			ps302.OverlayValues[210] = d210
			ps302.OverlayValues[211] = d211
			ps302.OverlayValues[212] = d212
			ps302.OverlayValues[213] = d213
			ps302.OverlayValues[214] = d214
			ps302.OverlayValues[215] = d215
			ps302.OverlayValues[216] = d216
			ps302.OverlayValues[217] = d217
			ps302.OverlayValues[218] = d218
			ps302.OverlayValues[219] = d219
			ps302.OverlayValues[220] = d220
			ps302.OverlayValues[221] = d221
			ps302.OverlayValues[222] = d222
			ps302.OverlayValues[223] = d223
			ps302.OverlayValues[224] = d224
			ps302.OverlayValues[225] = d225
			ps302.OverlayValues[226] = d226
			ps302.OverlayValues[227] = d227
			ps302.OverlayValues[228] = d228
			ps302.OverlayValues[229] = d229
			ps302.OverlayValues[230] = d230
			ps302.OverlayValues[231] = d231
			ps302.OverlayValues[232] = d232
			ps302.OverlayValues[233] = d233
			ps302.OverlayValues[234] = d234
			ps302.OverlayValues[235] = d235
			ps302.OverlayValues[236] = d236
			ps302.OverlayValues[237] = d237
			ps302.OverlayValues[238] = d238
			ps302.OverlayValues[239] = d239
			ps302.OverlayValues[240] = d240
			ps302.OverlayValues[246] = d246
			ps302.OverlayValues[247] = d247
			ps302.OverlayValues[248] = d248
			ps302.OverlayValues[249] = d249
			ps302.OverlayValues[255] = d255
			ps302.OverlayValues[256] = d256
			ps302.OverlayValues[257] = d257
			ps302.OverlayValues[258] = d258
			ps302.OverlayValues[259] = d259
			ps302.OverlayValues[260] = d260
			ps302.OverlayValues[261] = d261
			ps302.OverlayValues[262] = d262
			ps302.OverlayValues[263] = d263
			ps302.OverlayValues[264] = d264
			ps302.OverlayValues[265] = d265
			ps302.OverlayValues[266] = d266
			ps302.OverlayValues[267] = d267
			ps302.OverlayValues[268] = d268
			ps302.OverlayValues[269] = d269
			ps302.OverlayValues[270] = d270
			ps302.OverlayValues[271] = d271
			ps302.OverlayValues[272] = d272
			ps302.OverlayValues[273] = d273
			ps302.OverlayValues[274] = d274
			ps302.OverlayValues[275] = d275
			ps302.OverlayValues[276] = d276
			ps302.OverlayValues[277] = d277
			ps302.OverlayValues[278] = d278
			ps302.OverlayValues[279] = d279
			ps302.OverlayValues[280] = d280
			ps302.OverlayValues[281] = d281
			ps302.OverlayValues[282] = d282
			ps302.OverlayValues[283] = d283
			ps302.OverlayValues[284] = d284
			ps302.OverlayValues[285] = d285
			ps302.OverlayValues[286] = d286
			ps302.OverlayValues[287] = d287
			ps302.OverlayValues[288] = d288
			ps302.OverlayValues[289] = d289
			ps302.OverlayValues[290] = d290
			ps302.OverlayValues[291] = d291
			ps302.OverlayValues[292] = d292
			ps302.OverlayValues[293] = d293
			ps302.OverlayValues[294] = d294
			ps302.OverlayValues[295] = d295
			ps302.OverlayValues[296] = d296
			ps302.OverlayValues[297] = d297
			ps302.OverlayValues[298] = d298
			ps302.OverlayValues[299] = d299
			ps302.OverlayValues[300] = d300
				return bbs[7].RenderPS(ps302)
			}
			if !ps.General {
				ps.General = true
				return bbs[4].RenderPS(ps)
			}
			lbl80 := ctx.W.ReserveLabel()
			lbl81 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d300.Reg, 0)
			ctx.W.EmitJcc(scm.CcNE, lbl80)
			ctx.W.EmitJmp(lbl81)
			ctx.W.MarkLabel(lbl80)
			ctx.W.EmitJmp(lbl6)
			ctx.W.MarkLabel(lbl81)
			ctx.W.EmitJmp(lbl8)
			ps303 := scm.PhiState{General: true}
			ps303.OverlayValues = make([]scm.JITValueDesc, 301)
			ps303.OverlayValues[0] = d0
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
			ps303.OverlayValues[16] = d16
			ps303.OverlayValues[17] = d17
			ps303.OverlayValues[18] = d18
			ps303.OverlayValues[19] = d19
			ps303.OverlayValues[20] = d20
			ps303.OverlayValues[21] = d21
			ps303.OverlayValues[22] = d22
			ps303.OverlayValues[23] = d23
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
			ps303.OverlayValues[62] = d62
			ps303.OverlayValues[63] = d63
			ps303.OverlayValues[64] = d64
			ps303.OverlayValues[65] = d65
			ps303.OverlayValues[66] = d66
			ps303.OverlayValues[67] = d67
			ps303.OverlayValues[68] = d68
			ps303.OverlayValues[69] = d69
			ps303.OverlayValues[70] = d70
			ps303.OverlayValues[71] = d71
			ps303.OverlayValues[72] = d72
			ps303.OverlayValues[73] = d73
			ps303.OverlayValues[74] = d74
			ps303.OverlayValues[75] = d75
			ps303.OverlayValues[76] = d76
			ps303.OverlayValues[77] = d77
			ps303.OverlayValues[78] = d78
			ps303.OverlayValues[79] = d79
			ps303.OverlayValues[80] = d80
			ps303.OverlayValues[81] = d81
			ps303.OverlayValues[82] = d82
			ps303.OverlayValues[83] = d83
			ps303.OverlayValues[84] = d84
			ps303.OverlayValues[85] = d85
			ps303.OverlayValues[86] = d86
			ps303.OverlayValues[87] = d87
			ps303.OverlayValues[88] = d88
			ps303.OverlayValues[89] = d89
			ps303.OverlayValues[90] = d90
			ps303.OverlayValues[91] = d91
			ps303.OverlayValues[92] = d92
			ps303.OverlayValues[93] = d93
			ps303.OverlayValues[94] = d94
			ps303.OverlayValues[95] = d95
			ps303.OverlayValues[96] = d96
			ps303.OverlayValues[97] = d97
			ps303.OverlayValues[98] = d98
			ps303.OverlayValues[99] = d99
			ps303.OverlayValues[100] = d100
			ps303.OverlayValues[101] = d101
			ps303.OverlayValues[102] = d102
			ps303.OverlayValues[103] = d103
			ps303.OverlayValues[104] = d104
			ps303.OverlayValues[105] = d105
			ps303.OverlayValues[106] = d106
			ps303.OverlayValues[107] = d107
			ps303.OverlayValues[108] = d108
			ps303.OverlayValues[109] = d109
			ps303.OverlayValues[110] = d110
			ps303.OverlayValues[111] = d111
			ps303.OverlayValues[112] = d112
			ps303.OverlayValues[113] = d113
			ps303.OverlayValues[114] = d114
			ps303.OverlayValues[115] = d115
			ps303.OverlayValues[116] = d116
			ps303.OverlayValues[117] = d117
			ps303.OverlayValues[118] = d118
			ps303.OverlayValues[119] = d119
			ps303.OverlayValues[120] = d120
			ps303.OverlayValues[121] = d121
			ps303.OverlayValues[122] = d122
			ps303.OverlayValues[123] = d123
			ps303.OverlayValues[124] = d124
			ps303.OverlayValues[125] = d125
			ps303.OverlayValues[126] = d126
			ps303.OverlayValues[127] = d127
			ps303.OverlayValues[128] = d128
			ps303.OverlayValues[129] = d129
			ps303.OverlayValues[130] = d130
			ps303.OverlayValues[131] = d131
			ps303.OverlayValues[132] = d132
			ps303.OverlayValues[133] = d133
			ps303.OverlayValues[134] = d134
			ps303.OverlayValues[135] = d135
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
			ps303.OverlayValues[173] = d173
			ps303.OverlayValues[174] = d174
			ps303.OverlayValues[175] = d175
			ps303.OverlayValues[176] = d176
			ps303.OverlayValues[177] = d177
			ps303.OverlayValues[178] = d178
			ps303.OverlayValues[179] = d179
			ps303.OverlayValues[180] = d180
			ps303.OverlayValues[181] = d181
			ps303.OverlayValues[182] = d182
			ps303.OverlayValues[183] = d183
			ps303.OverlayValues[184] = d184
			ps303.OverlayValues[185] = d185
			ps303.OverlayValues[186] = d186
			ps303.OverlayValues[187] = d187
			ps303.OverlayValues[188] = d188
			ps303.OverlayValues[189] = d189
			ps303.OverlayValues[190] = d190
			ps303.OverlayValues[191] = d191
			ps303.OverlayValues[192] = d192
			ps303.OverlayValues[193] = d193
			ps303.OverlayValues[194] = d194
			ps303.OverlayValues[195] = d195
			ps303.OverlayValues[196] = d196
			ps303.OverlayValues[197] = d197
			ps303.OverlayValues[198] = d198
			ps303.OverlayValues[199] = d199
			ps303.OverlayValues[200] = d200
			ps303.OverlayValues[201] = d201
			ps303.OverlayValues[202] = d202
			ps303.OverlayValues[203] = d203
			ps303.OverlayValues[204] = d204
			ps303.OverlayValues[205] = d205
			ps303.OverlayValues[206] = d206
			ps303.OverlayValues[207] = d207
			ps303.OverlayValues[208] = d208
			ps303.OverlayValues[209] = d209
			ps303.OverlayValues[210] = d210
			ps303.OverlayValues[211] = d211
			ps303.OverlayValues[212] = d212
			ps303.OverlayValues[213] = d213
			ps303.OverlayValues[214] = d214
			ps303.OverlayValues[215] = d215
			ps303.OverlayValues[216] = d216
			ps303.OverlayValues[217] = d217
			ps303.OverlayValues[218] = d218
			ps303.OverlayValues[219] = d219
			ps303.OverlayValues[220] = d220
			ps303.OverlayValues[221] = d221
			ps303.OverlayValues[222] = d222
			ps303.OverlayValues[223] = d223
			ps303.OverlayValues[224] = d224
			ps303.OverlayValues[225] = d225
			ps303.OverlayValues[226] = d226
			ps303.OverlayValues[227] = d227
			ps303.OverlayValues[228] = d228
			ps303.OverlayValues[229] = d229
			ps303.OverlayValues[230] = d230
			ps303.OverlayValues[231] = d231
			ps303.OverlayValues[232] = d232
			ps303.OverlayValues[233] = d233
			ps303.OverlayValues[234] = d234
			ps303.OverlayValues[235] = d235
			ps303.OverlayValues[236] = d236
			ps303.OverlayValues[237] = d237
			ps303.OverlayValues[238] = d238
			ps303.OverlayValues[239] = d239
			ps303.OverlayValues[240] = d240
			ps303.OverlayValues[246] = d246
			ps303.OverlayValues[247] = d247
			ps303.OverlayValues[248] = d248
			ps303.OverlayValues[249] = d249
			ps303.OverlayValues[255] = d255
			ps303.OverlayValues[256] = d256
			ps303.OverlayValues[257] = d257
			ps303.OverlayValues[258] = d258
			ps303.OverlayValues[259] = d259
			ps303.OverlayValues[260] = d260
			ps303.OverlayValues[261] = d261
			ps303.OverlayValues[262] = d262
			ps303.OverlayValues[263] = d263
			ps303.OverlayValues[264] = d264
			ps303.OverlayValues[265] = d265
			ps303.OverlayValues[266] = d266
			ps303.OverlayValues[267] = d267
			ps303.OverlayValues[268] = d268
			ps303.OverlayValues[269] = d269
			ps303.OverlayValues[270] = d270
			ps303.OverlayValues[271] = d271
			ps303.OverlayValues[272] = d272
			ps303.OverlayValues[273] = d273
			ps303.OverlayValues[274] = d274
			ps303.OverlayValues[275] = d275
			ps303.OverlayValues[276] = d276
			ps303.OverlayValues[277] = d277
			ps303.OverlayValues[278] = d278
			ps303.OverlayValues[279] = d279
			ps303.OverlayValues[280] = d280
			ps303.OverlayValues[281] = d281
			ps303.OverlayValues[282] = d282
			ps303.OverlayValues[283] = d283
			ps303.OverlayValues[284] = d284
			ps303.OverlayValues[285] = d285
			ps303.OverlayValues[286] = d286
			ps303.OverlayValues[287] = d287
			ps303.OverlayValues[288] = d288
			ps303.OverlayValues[289] = d289
			ps303.OverlayValues[290] = d290
			ps303.OverlayValues[291] = d291
			ps303.OverlayValues[292] = d292
			ps303.OverlayValues[293] = d293
			ps303.OverlayValues[294] = d294
			ps303.OverlayValues[295] = d295
			ps303.OverlayValues[296] = d296
			ps303.OverlayValues[297] = d297
			ps303.OverlayValues[298] = d298
			ps303.OverlayValues[299] = d299
			ps303.OverlayValues[300] = d300
			ps304 := scm.PhiState{General: true}
			ps304.OverlayValues = make([]scm.JITValueDesc, 301)
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
			ps304.OverlayValues[17] = d17
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
			ps304.OverlayValues[62] = d62
			ps304.OverlayValues[63] = d63
			ps304.OverlayValues[64] = d64
			ps304.OverlayValues[65] = d65
			ps304.OverlayValues[66] = d66
			ps304.OverlayValues[67] = d67
			ps304.OverlayValues[68] = d68
			ps304.OverlayValues[69] = d69
			ps304.OverlayValues[70] = d70
			ps304.OverlayValues[71] = d71
			ps304.OverlayValues[72] = d72
			ps304.OverlayValues[73] = d73
			ps304.OverlayValues[74] = d74
			ps304.OverlayValues[75] = d75
			ps304.OverlayValues[76] = d76
			ps304.OverlayValues[77] = d77
			ps304.OverlayValues[78] = d78
			ps304.OverlayValues[79] = d79
			ps304.OverlayValues[80] = d80
			ps304.OverlayValues[81] = d81
			ps304.OverlayValues[82] = d82
			ps304.OverlayValues[83] = d83
			ps304.OverlayValues[84] = d84
			ps304.OverlayValues[85] = d85
			ps304.OverlayValues[86] = d86
			ps304.OverlayValues[87] = d87
			ps304.OverlayValues[88] = d88
			ps304.OverlayValues[89] = d89
			ps304.OverlayValues[90] = d90
			ps304.OverlayValues[91] = d91
			ps304.OverlayValues[92] = d92
			ps304.OverlayValues[93] = d93
			ps304.OverlayValues[94] = d94
			ps304.OverlayValues[95] = d95
			ps304.OverlayValues[96] = d96
			ps304.OverlayValues[97] = d97
			ps304.OverlayValues[98] = d98
			ps304.OverlayValues[99] = d99
			ps304.OverlayValues[100] = d100
			ps304.OverlayValues[101] = d101
			ps304.OverlayValues[102] = d102
			ps304.OverlayValues[103] = d103
			ps304.OverlayValues[104] = d104
			ps304.OverlayValues[105] = d105
			ps304.OverlayValues[106] = d106
			ps304.OverlayValues[107] = d107
			ps304.OverlayValues[108] = d108
			ps304.OverlayValues[109] = d109
			ps304.OverlayValues[110] = d110
			ps304.OverlayValues[111] = d111
			ps304.OverlayValues[112] = d112
			ps304.OverlayValues[113] = d113
			ps304.OverlayValues[114] = d114
			ps304.OverlayValues[115] = d115
			ps304.OverlayValues[116] = d116
			ps304.OverlayValues[117] = d117
			ps304.OverlayValues[118] = d118
			ps304.OverlayValues[119] = d119
			ps304.OverlayValues[120] = d120
			ps304.OverlayValues[121] = d121
			ps304.OverlayValues[122] = d122
			ps304.OverlayValues[123] = d123
			ps304.OverlayValues[124] = d124
			ps304.OverlayValues[125] = d125
			ps304.OverlayValues[126] = d126
			ps304.OverlayValues[127] = d127
			ps304.OverlayValues[128] = d128
			ps304.OverlayValues[129] = d129
			ps304.OverlayValues[130] = d130
			ps304.OverlayValues[131] = d131
			ps304.OverlayValues[132] = d132
			ps304.OverlayValues[133] = d133
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
			ps304.OverlayValues[173] = d173
			ps304.OverlayValues[174] = d174
			ps304.OverlayValues[175] = d175
			ps304.OverlayValues[176] = d176
			ps304.OverlayValues[177] = d177
			ps304.OverlayValues[178] = d178
			ps304.OverlayValues[179] = d179
			ps304.OverlayValues[180] = d180
			ps304.OverlayValues[181] = d181
			ps304.OverlayValues[182] = d182
			ps304.OverlayValues[183] = d183
			ps304.OverlayValues[184] = d184
			ps304.OverlayValues[185] = d185
			ps304.OverlayValues[186] = d186
			ps304.OverlayValues[187] = d187
			ps304.OverlayValues[188] = d188
			ps304.OverlayValues[189] = d189
			ps304.OverlayValues[190] = d190
			ps304.OverlayValues[191] = d191
			ps304.OverlayValues[192] = d192
			ps304.OverlayValues[193] = d193
			ps304.OverlayValues[194] = d194
			ps304.OverlayValues[195] = d195
			ps304.OverlayValues[196] = d196
			ps304.OverlayValues[197] = d197
			ps304.OverlayValues[198] = d198
			ps304.OverlayValues[199] = d199
			ps304.OverlayValues[200] = d200
			ps304.OverlayValues[201] = d201
			ps304.OverlayValues[202] = d202
			ps304.OverlayValues[203] = d203
			ps304.OverlayValues[204] = d204
			ps304.OverlayValues[205] = d205
			ps304.OverlayValues[206] = d206
			ps304.OverlayValues[207] = d207
			ps304.OverlayValues[208] = d208
			ps304.OverlayValues[209] = d209
			ps304.OverlayValues[210] = d210
			ps304.OverlayValues[211] = d211
			ps304.OverlayValues[212] = d212
			ps304.OverlayValues[213] = d213
			ps304.OverlayValues[214] = d214
			ps304.OverlayValues[215] = d215
			ps304.OverlayValues[216] = d216
			ps304.OverlayValues[217] = d217
			ps304.OverlayValues[218] = d218
			ps304.OverlayValues[219] = d219
			ps304.OverlayValues[220] = d220
			ps304.OverlayValues[221] = d221
			ps304.OverlayValues[222] = d222
			ps304.OverlayValues[223] = d223
			ps304.OverlayValues[224] = d224
			ps304.OverlayValues[225] = d225
			ps304.OverlayValues[226] = d226
			ps304.OverlayValues[227] = d227
			ps304.OverlayValues[228] = d228
			ps304.OverlayValues[229] = d229
			ps304.OverlayValues[230] = d230
			ps304.OverlayValues[231] = d231
			ps304.OverlayValues[232] = d232
			ps304.OverlayValues[233] = d233
			ps304.OverlayValues[234] = d234
			ps304.OverlayValues[235] = d235
			ps304.OverlayValues[236] = d236
			ps304.OverlayValues[237] = d237
			ps304.OverlayValues[238] = d238
			ps304.OverlayValues[239] = d239
			ps304.OverlayValues[240] = d240
			ps304.OverlayValues[246] = d246
			ps304.OverlayValues[247] = d247
			ps304.OverlayValues[248] = d248
			ps304.OverlayValues[249] = d249
			ps304.OverlayValues[255] = d255
			ps304.OverlayValues[256] = d256
			ps304.OverlayValues[257] = d257
			ps304.OverlayValues[258] = d258
			ps304.OverlayValues[259] = d259
			ps304.OverlayValues[260] = d260
			ps304.OverlayValues[261] = d261
			ps304.OverlayValues[262] = d262
			ps304.OverlayValues[263] = d263
			ps304.OverlayValues[264] = d264
			ps304.OverlayValues[265] = d265
			ps304.OverlayValues[266] = d266
			ps304.OverlayValues[267] = d267
			ps304.OverlayValues[268] = d268
			ps304.OverlayValues[269] = d269
			ps304.OverlayValues[270] = d270
			ps304.OverlayValues[271] = d271
			ps304.OverlayValues[272] = d272
			ps304.OverlayValues[273] = d273
			ps304.OverlayValues[274] = d274
			ps304.OverlayValues[275] = d275
			ps304.OverlayValues[276] = d276
			ps304.OverlayValues[277] = d277
			ps304.OverlayValues[278] = d278
			ps304.OverlayValues[279] = d279
			ps304.OverlayValues[280] = d280
			ps304.OverlayValues[281] = d281
			ps304.OverlayValues[282] = d282
			ps304.OverlayValues[283] = d283
			ps304.OverlayValues[284] = d284
			ps304.OverlayValues[285] = d285
			ps304.OverlayValues[286] = d286
			ps304.OverlayValues[287] = d287
			ps304.OverlayValues[288] = d288
			ps304.OverlayValues[289] = d289
			ps304.OverlayValues[290] = d290
			ps304.OverlayValues[291] = d291
			ps304.OverlayValues[292] = d292
			ps304.OverlayValues[293] = d293
			ps304.OverlayValues[294] = d294
			ps304.OverlayValues[295] = d295
			ps304.OverlayValues[296] = d296
			ps304.OverlayValues[297] = d297
			ps304.OverlayValues[298] = d298
			ps304.OverlayValues[299] = d299
			ps304.OverlayValues[300] = d300
			alloc305 := ctx.SnapshotAllocState()
			if !bbs[7].Rendered {
				bbs[7].RenderPS(ps304)
			}
			ctx.RestoreAllocState(alloc305)
			if !bbs[5].Rendered {
				return bbs[5].RenderPS(ps303)
			}
			return result
			ctx.FreeDesc(&d299)
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
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
				d6 = ps.OverlayValues[6]
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
			if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != scm.LocNone {
				d101 = ps.OverlayValues[101]
			}
			if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != scm.LocNone {
				d102 = ps.OverlayValues[102]
			}
			if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != scm.LocNone {
				d103 = ps.OverlayValues[103]
			}
			if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != scm.LocNone {
				d104 = ps.OverlayValues[104]
			}
			if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != scm.LocNone {
				d105 = ps.OverlayValues[105]
			}
			if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != scm.LocNone {
				d106 = ps.OverlayValues[106]
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
			if len(ps.OverlayValues) > 255 && ps.OverlayValues[255].Loc != scm.LocNone {
				d255 = ps.OverlayValues[255]
			}
			if len(ps.OverlayValues) > 256 && ps.OverlayValues[256].Loc != scm.LocNone {
				d256 = ps.OverlayValues[256]
			}
			if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != scm.LocNone {
				d257 = ps.OverlayValues[257]
			}
			if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != scm.LocNone {
				d258 = ps.OverlayValues[258]
			}
			if len(ps.OverlayValues) > 259 && ps.OverlayValues[259].Loc != scm.LocNone {
				d259 = ps.OverlayValues[259]
			}
			if len(ps.OverlayValues) > 260 && ps.OverlayValues[260].Loc != scm.LocNone {
				d260 = ps.OverlayValues[260]
			}
			if len(ps.OverlayValues) > 261 && ps.OverlayValues[261].Loc != scm.LocNone {
				d261 = ps.OverlayValues[261]
			}
			if len(ps.OverlayValues) > 262 && ps.OverlayValues[262].Loc != scm.LocNone {
				d262 = ps.OverlayValues[262]
			}
			if len(ps.OverlayValues) > 263 && ps.OverlayValues[263].Loc != scm.LocNone {
				d263 = ps.OverlayValues[263]
			}
			if len(ps.OverlayValues) > 264 && ps.OverlayValues[264].Loc != scm.LocNone {
				d264 = ps.OverlayValues[264]
			}
			if len(ps.OverlayValues) > 265 && ps.OverlayValues[265].Loc != scm.LocNone {
				d265 = ps.OverlayValues[265]
			}
			if len(ps.OverlayValues) > 266 && ps.OverlayValues[266].Loc != scm.LocNone {
				d266 = ps.OverlayValues[266]
			}
			if len(ps.OverlayValues) > 267 && ps.OverlayValues[267].Loc != scm.LocNone {
				d267 = ps.OverlayValues[267]
			}
			if len(ps.OverlayValues) > 268 && ps.OverlayValues[268].Loc != scm.LocNone {
				d268 = ps.OverlayValues[268]
			}
			if len(ps.OverlayValues) > 269 && ps.OverlayValues[269].Loc != scm.LocNone {
				d269 = ps.OverlayValues[269]
			}
			if len(ps.OverlayValues) > 270 && ps.OverlayValues[270].Loc != scm.LocNone {
				d270 = ps.OverlayValues[270]
			}
			if len(ps.OverlayValues) > 271 && ps.OverlayValues[271].Loc != scm.LocNone {
				d271 = ps.OverlayValues[271]
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
			if len(ps.OverlayValues) > 276 && ps.OverlayValues[276].Loc != scm.LocNone {
				d276 = ps.OverlayValues[276]
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
			ctx.ReclaimUntrackedRegs()
			ctx.W.EmitByte(0xCC)
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
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
				d6 = ps.OverlayValues[6]
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
			if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != scm.LocNone {
				d101 = ps.OverlayValues[101]
			}
			if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != scm.LocNone {
				d102 = ps.OverlayValues[102]
			}
			if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != scm.LocNone {
				d103 = ps.OverlayValues[103]
			}
			if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != scm.LocNone {
				d104 = ps.OverlayValues[104]
			}
			if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != scm.LocNone {
				d105 = ps.OverlayValues[105]
			}
			if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != scm.LocNone {
				d106 = ps.OverlayValues[106]
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
			if len(ps.OverlayValues) > 255 && ps.OverlayValues[255].Loc != scm.LocNone {
				d255 = ps.OverlayValues[255]
			}
			if len(ps.OverlayValues) > 256 && ps.OverlayValues[256].Loc != scm.LocNone {
				d256 = ps.OverlayValues[256]
			}
			if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != scm.LocNone {
				d257 = ps.OverlayValues[257]
			}
			if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != scm.LocNone {
				d258 = ps.OverlayValues[258]
			}
			if len(ps.OverlayValues) > 259 && ps.OverlayValues[259].Loc != scm.LocNone {
				d259 = ps.OverlayValues[259]
			}
			if len(ps.OverlayValues) > 260 && ps.OverlayValues[260].Loc != scm.LocNone {
				d260 = ps.OverlayValues[260]
			}
			if len(ps.OverlayValues) > 261 && ps.OverlayValues[261].Loc != scm.LocNone {
				d261 = ps.OverlayValues[261]
			}
			if len(ps.OverlayValues) > 262 && ps.OverlayValues[262].Loc != scm.LocNone {
				d262 = ps.OverlayValues[262]
			}
			if len(ps.OverlayValues) > 263 && ps.OverlayValues[263].Loc != scm.LocNone {
				d263 = ps.OverlayValues[263]
			}
			if len(ps.OverlayValues) > 264 && ps.OverlayValues[264].Loc != scm.LocNone {
				d264 = ps.OverlayValues[264]
			}
			if len(ps.OverlayValues) > 265 && ps.OverlayValues[265].Loc != scm.LocNone {
				d265 = ps.OverlayValues[265]
			}
			if len(ps.OverlayValues) > 266 && ps.OverlayValues[266].Loc != scm.LocNone {
				d266 = ps.OverlayValues[266]
			}
			if len(ps.OverlayValues) > 267 && ps.OverlayValues[267].Loc != scm.LocNone {
				d267 = ps.OverlayValues[267]
			}
			if len(ps.OverlayValues) > 268 && ps.OverlayValues[268].Loc != scm.LocNone {
				d268 = ps.OverlayValues[268]
			}
			if len(ps.OverlayValues) > 269 && ps.OverlayValues[269].Loc != scm.LocNone {
				d269 = ps.OverlayValues[269]
			}
			if len(ps.OverlayValues) > 270 && ps.OverlayValues[270].Loc != scm.LocNone {
				d270 = ps.OverlayValues[270]
			}
			if len(ps.OverlayValues) > 271 && ps.OverlayValues[271].Loc != scm.LocNone {
				d271 = ps.OverlayValues[271]
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
			if len(ps.OverlayValues) > 276 && ps.OverlayValues[276].Loc != scm.LocNone {
				d276 = ps.OverlayValues[276]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d295)
			r295 := ctx.AllocReg()
			ctx.EnsureDesc(&d295)
			ctx.EnsureDesc(&d296)
			if d295.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r295, uint64(d295.Imm.Int()) * 16)
			} else {
				ctx.W.EmitMovRegReg(r295, d295.Reg)
				ctx.W.EmitShlRegImm8(r295, 4)
			}
			if d296.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d296.Imm.Int()))
				ctx.W.EmitAddInt64(r295, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r295, d296.Reg)
			}
			r296 := ctx.AllocRegExcept(r295)
			r297 := ctx.AllocRegExcept(r295, r296)
			ctx.W.EmitMovRegMem(r296, r295, 0)
			ctx.W.EmitMovRegMem(r297, r295, 8)
			ctx.FreeReg(r295)
			d306 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r296, Reg2: r297}
			ctx.BindReg(r296, &d306)
			ctx.BindReg(r297, &d306)
			d308 = d237
			ctx.EnsureDesc(&d308)
			if d308.Loc == scm.LocImm {
				tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d308.Imm.GetTag()
				switch tag {
				case scm.TagBool:
					ctx.W.EmitMakeBool(tmpPair, d308)
				case scm.TagInt:
					ctx.W.EmitMakeInt(tmpPair, d308)
				case scm.TagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d308)
				case scm.TagNil:
					ctx.W.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d308.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d308 = tmpPair
			} else if d308.Loc == scm.LocReg {
				tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocRegExcept(d308.Reg), Reg2: ctx.AllocRegExcept(d308.Reg)}
				switch d308.Type {
				case scm.TagBool:
					ctx.W.EmitMakeBool(tmpPair, d308)
				case scm.TagInt:
					ctx.W.EmitMakeInt(tmpPair, d308)
				case scm.TagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d308)
				default:
					panic("jit: scm.Scmer.String requires scm.Scmer pair receiver")
				}
				ctx.FreeDesc(&d308)
				d308 = tmpPair
			} else if d308.Loc == scm.LocMem {
				tmpScalar := scm.JITValueDesc{Loc: scm.LocReg, Type: d308.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d308.MemPtr))
				ctx.W.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case scm.TagBool:
					ctx.W.EmitMakeBool(tmpPair, tmpScalar)
				case scm.TagInt:
					ctx.W.EmitMakeInt(tmpPair, tmpScalar)
				case scm.TagFloat:
					ctx.W.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: scm.Scmer.String requires scm.Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d308 = tmpPair
			}
			if d308.Loc != scm.LocRegPair && d308.Loc != scm.LocStackPair {
				panic("jit: scm.Scmer.String receiver not materialized as pair")
			}
			d307 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.Scmer.String), []scm.JITValueDesc{d308}, 2)
			ctx.FreeDesc(&d237)
			ctx.EnsureDesc(&d306)
			ctx.EnsureDesc(&d307)
			d309 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.ConcatStrings), []scm.JITValueDesc{d306, d307}, 2)
			ctx.FreeDesc(&d306)
			d310 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d310)
			ctx.BindReg(r1, &d310)
			d311 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d309}, 2)
			ctx.EmitMovPairToResult(&d311, &d310)
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
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
				d6 = ps.OverlayValues[6]
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
			if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != scm.LocNone {
				d101 = ps.OverlayValues[101]
			}
			if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != scm.LocNone {
				d102 = ps.OverlayValues[102]
			}
			if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != scm.LocNone {
				d103 = ps.OverlayValues[103]
			}
			if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != scm.LocNone {
				d104 = ps.OverlayValues[104]
			}
			if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != scm.LocNone {
				d105 = ps.OverlayValues[105]
			}
			if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != scm.LocNone {
				d106 = ps.OverlayValues[106]
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
			if len(ps.OverlayValues) > 255 && ps.OverlayValues[255].Loc != scm.LocNone {
				d255 = ps.OverlayValues[255]
			}
			if len(ps.OverlayValues) > 256 && ps.OverlayValues[256].Loc != scm.LocNone {
				d256 = ps.OverlayValues[256]
			}
			if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != scm.LocNone {
				d257 = ps.OverlayValues[257]
			}
			if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != scm.LocNone {
				d258 = ps.OverlayValues[258]
			}
			if len(ps.OverlayValues) > 259 && ps.OverlayValues[259].Loc != scm.LocNone {
				d259 = ps.OverlayValues[259]
			}
			if len(ps.OverlayValues) > 260 && ps.OverlayValues[260].Loc != scm.LocNone {
				d260 = ps.OverlayValues[260]
			}
			if len(ps.OverlayValues) > 261 && ps.OverlayValues[261].Loc != scm.LocNone {
				d261 = ps.OverlayValues[261]
			}
			if len(ps.OverlayValues) > 262 && ps.OverlayValues[262].Loc != scm.LocNone {
				d262 = ps.OverlayValues[262]
			}
			if len(ps.OverlayValues) > 263 && ps.OverlayValues[263].Loc != scm.LocNone {
				d263 = ps.OverlayValues[263]
			}
			if len(ps.OverlayValues) > 264 && ps.OverlayValues[264].Loc != scm.LocNone {
				d264 = ps.OverlayValues[264]
			}
			if len(ps.OverlayValues) > 265 && ps.OverlayValues[265].Loc != scm.LocNone {
				d265 = ps.OverlayValues[265]
			}
			if len(ps.OverlayValues) > 266 && ps.OverlayValues[266].Loc != scm.LocNone {
				d266 = ps.OverlayValues[266]
			}
			if len(ps.OverlayValues) > 267 && ps.OverlayValues[267].Loc != scm.LocNone {
				d267 = ps.OverlayValues[267]
			}
			if len(ps.OverlayValues) > 268 && ps.OverlayValues[268].Loc != scm.LocNone {
				d268 = ps.OverlayValues[268]
			}
			if len(ps.OverlayValues) > 269 && ps.OverlayValues[269].Loc != scm.LocNone {
				d269 = ps.OverlayValues[269]
			}
			if len(ps.OverlayValues) > 270 && ps.OverlayValues[270].Loc != scm.LocNone {
				d270 = ps.OverlayValues[270]
			}
			if len(ps.OverlayValues) > 271 && ps.OverlayValues[271].Loc != scm.LocNone {
				d271 = ps.OverlayValues[271]
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
			if len(ps.OverlayValues) > 276 && ps.OverlayValues[276].Loc != scm.LocNone {
				d276 = ps.OverlayValues[276]
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
			if len(ps.OverlayValues) > 310 && ps.OverlayValues[310].Loc != scm.LocNone {
				d310 = ps.OverlayValues[310]
			}
			if len(ps.OverlayValues) > 311 && ps.OverlayValues[311].Loc != scm.LocNone {
				d311 = ps.OverlayValues[311]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d295)
			var d312 scm.JITValueDesc
			if d295.Loc == scm.LocImm {
				d312 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d295.Imm.Int() < 0)}
			} else {
				r298 := ctx.AllocRegExcept(d295.Reg)
				ctx.W.EmitCmpRegImm32(d295.Reg, 0)
				ctx.W.EmitSetcc(r298, scm.CcL)
				d312 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r298}
				ctx.BindReg(r298, &d312)
			}
			ctx.FreeDesc(&d295)
			d313 = d312
			ctx.EnsureDesc(&d313)
			if d313.Loc != scm.LocImm && d313.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d313.Loc == scm.LocImm {
				if d313.Imm.Bool() {
			ps314 := scm.PhiState{General: ps.General}
			ps314.OverlayValues = make([]scm.JITValueDesc, 314)
			ps314.OverlayValues[0] = d0
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
			ps314.OverlayValues[18] = d18
			ps314.OverlayValues[19] = d19
			ps314.OverlayValues[20] = d20
			ps314.OverlayValues[21] = d21
			ps314.OverlayValues[22] = d22
			ps314.OverlayValues[23] = d23
			ps314.OverlayValues[24] = d24
			ps314.OverlayValues[25] = d25
			ps314.OverlayValues[26] = d26
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
			ps314.OverlayValues[64] = d64
			ps314.OverlayValues[65] = d65
			ps314.OverlayValues[66] = d66
			ps314.OverlayValues[67] = d67
			ps314.OverlayValues[68] = d68
			ps314.OverlayValues[69] = d69
			ps314.OverlayValues[70] = d70
			ps314.OverlayValues[71] = d71
			ps314.OverlayValues[72] = d72
			ps314.OverlayValues[73] = d73
			ps314.OverlayValues[74] = d74
			ps314.OverlayValues[75] = d75
			ps314.OverlayValues[76] = d76
			ps314.OverlayValues[77] = d77
			ps314.OverlayValues[78] = d78
			ps314.OverlayValues[79] = d79
			ps314.OverlayValues[80] = d80
			ps314.OverlayValues[81] = d81
			ps314.OverlayValues[82] = d82
			ps314.OverlayValues[83] = d83
			ps314.OverlayValues[84] = d84
			ps314.OverlayValues[85] = d85
			ps314.OverlayValues[86] = d86
			ps314.OverlayValues[87] = d87
			ps314.OverlayValues[88] = d88
			ps314.OverlayValues[89] = d89
			ps314.OverlayValues[90] = d90
			ps314.OverlayValues[91] = d91
			ps314.OverlayValues[92] = d92
			ps314.OverlayValues[93] = d93
			ps314.OverlayValues[94] = d94
			ps314.OverlayValues[95] = d95
			ps314.OverlayValues[96] = d96
			ps314.OverlayValues[97] = d97
			ps314.OverlayValues[98] = d98
			ps314.OverlayValues[99] = d99
			ps314.OverlayValues[100] = d100
			ps314.OverlayValues[101] = d101
			ps314.OverlayValues[102] = d102
			ps314.OverlayValues[103] = d103
			ps314.OverlayValues[104] = d104
			ps314.OverlayValues[105] = d105
			ps314.OverlayValues[106] = d106
			ps314.OverlayValues[107] = d107
			ps314.OverlayValues[108] = d108
			ps314.OverlayValues[109] = d109
			ps314.OverlayValues[110] = d110
			ps314.OverlayValues[111] = d111
			ps314.OverlayValues[112] = d112
			ps314.OverlayValues[113] = d113
			ps314.OverlayValues[114] = d114
			ps314.OverlayValues[115] = d115
			ps314.OverlayValues[116] = d116
			ps314.OverlayValues[117] = d117
			ps314.OverlayValues[118] = d118
			ps314.OverlayValues[119] = d119
			ps314.OverlayValues[120] = d120
			ps314.OverlayValues[121] = d121
			ps314.OverlayValues[122] = d122
			ps314.OverlayValues[123] = d123
			ps314.OverlayValues[124] = d124
			ps314.OverlayValues[125] = d125
			ps314.OverlayValues[126] = d126
			ps314.OverlayValues[127] = d127
			ps314.OverlayValues[128] = d128
			ps314.OverlayValues[129] = d129
			ps314.OverlayValues[130] = d130
			ps314.OverlayValues[131] = d131
			ps314.OverlayValues[132] = d132
			ps314.OverlayValues[133] = d133
			ps314.OverlayValues[134] = d134
			ps314.OverlayValues[135] = d135
			ps314.OverlayValues[136] = d136
			ps314.OverlayValues[137] = d137
			ps314.OverlayValues[138] = d138
			ps314.OverlayValues[139] = d139
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
			ps314.OverlayValues[176] = d176
			ps314.OverlayValues[177] = d177
			ps314.OverlayValues[178] = d178
			ps314.OverlayValues[179] = d179
			ps314.OverlayValues[180] = d180
			ps314.OverlayValues[181] = d181
			ps314.OverlayValues[182] = d182
			ps314.OverlayValues[183] = d183
			ps314.OverlayValues[184] = d184
			ps314.OverlayValues[185] = d185
			ps314.OverlayValues[186] = d186
			ps314.OverlayValues[187] = d187
			ps314.OverlayValues[188] = d188
			ps314.OverlayValues[189] = d189
			ps314.OverlayValues[190] = d190
			ps314.OverlayValues[191] = d191
			ps314.OverlayValues[192] = d192
			ps314.OverlayValues[193] = d193
			ps314.OverlayValues[194] = d194
			ps314.OverlayValues[195] = d195
			ps314.OverlayValues[196] = d196
			ps314.OverlayValues[197] = d197
			ps314.OverlayValues[198] = d198
			ps314.OverlayValues[199] = d199
			ps314.OverlayValues[200] = d200
			ps314.OverlayValues[201] = d201
			ps314.OverlayValues[202] = d202
			ps314.OverlayValues[203] = d203
			ps314.OverlayValues[204] = d204
			ps314.OverlayValues[205] = d205
			ps314.OverlayValues[206] = d206
			ps314.OverlayValues[207] = d207
			ps314.OverlayValues[208] = d208
			ps314.OverlayValues[209] = d209
			ps314.OverlayValues[210] = d210
			ps314.OverlayValues[211] = d211
			ps314.OverlayValues[212] = d212
			ps314.OverlayValues[213] = d213
			ps314.OverlayValues[214] = d214
			ps314.OverlayValues[215] = d215
			ps314.OverlayValues[216] = d216
			ps314.OverlayValues[217] = d217
			ps314.OverlayValues[218] = d218
			ps314.OverlayValues[219] = d219
			ps314.OverlayValues[220] = d220
			ps314.OverlayValues[221] = d221
			ps314.OverlayValues[222] = d222
			ps314.OverlayValues[223] = d223
			ps314.OverlayValues[224] = d224
			ps314.OverlayValues[225] = d225
			ps314.OverlayValues[226] = d226
			ps314.OverlayValues[227] = d227
			ps314.OverlayValues[228] = d228
			ps314.OverlayValues[229] = d229
			ps314.OverlayValues[230] = d230
			ps314.OverlayValues[231] = d231
			ps314.OverlayValues[232] = d232
			ps314.OverlayValues[233] = d233
			ps314.OverlayValues[234] = d234
			ps314.OverlayValues[235] = d235
			ps314.OverlayValues[236] = d236
			ps314.OverlayValues[237] = d237
			ps314.OverlayValues[238] = d238
			ps314.OverlayValues[239] = d239
			ps314.OverlayValues[240] = d240
			ps314.OverlayValues[246] = d246
			ps314.OverlayValues[247] = d247
			ps314.OverlayValues[248] = d248
			ps314.OverlayValues[249] = d249
			ps314.OverlayValues[255] = d255
			ps314.OverlayValues[256] = d256
			ps314.OverlayValues[257] = d257
			ps314.OverlayValues[258] = d258
			ps314.OverlayValues[259] = d259
			ps314.OverlayValues[260] = d260
			ps314.OverlayValues[261] = d261
			ps314.OverlayValues[262] = d262
			ps314.OverlayValues[263] = d263
			ps314.OverlayValues[264] = d264
			ps314.OverlayValues[265] = d265
			ps314.OverlayValues[266] = d266
			ps314.OverlayValues[267] = d267
			ps314.OverlayValues[268] = d268
			ps314.OverlayValues[269] = d269
			ps314.OverlayValues[270] = d270
			ps314.OverlayValues[271] = d271
			ps314.OverlayValues[272] = d272
			ps314.OverlayValues[273] = d273
			ps314.OverlayValues[274] = d274
			ps314.OverlayValues[275] = d275
			ps314.OverlayValues[276] = d276
			ps314.OverlayValues[277] = d277
			ps314.OverlayValues[278] = d278
			ps314.OverlayValues[279] = d279
			ps314.OverlayValues[280] = d280
			ps314.OverlayValues[281] = d281
			ps314.OverlayValues[282] = d282
			ps314.OverlayValues[283] = d283
			ps314.OverlayValues[284] = d284
			ps314.OverlayValues[285] = d285
			ps314.OverlayValues[286] = d286
			ps314.OverlayValues[287] = d287
			ps314.OverlayValues[288] = d288
			ps314.OverlayValues[289] = d289
			ps314.OverlayValues[290] = d290
			ps314.OverlayValues[291] = d291
			ps314.OverlayValues[292] = d292
			ps314.OverlayValues[293] = d293
			ps314.OverlayValues[294] = d294
			ps314.OverlayValues[295] = d295
			ps314.OverlayValues[296] = d296
			ps314.OverlayValues[297] = d297
			ps314.OverlayValues[298] = d298
			ps314.OverlayValues[299] = d299
			ps314.OverlayValues[300] = d300
			ps314.OverlayValues[306] = d306
			ps314.OverlayValues[307] = d307
			ps314.OverlayValues[308] = d308
			ps314.OverlayValues[309] = d309
			ps314.OverlayValues[310] = d310
			ps314.OverlayValues[311] = d311
			ps314.OverlayValues[312] = d312
			ps314.OverlayValues[313] = d313
					return bbs[5].RenderPS(ps314)
				}
			ps315 := scm.PhiState{General: ps.General}
			ps315.OverlayValues = make([]scm.JITValueDesc, 314)
			ps315.OverlayValues[0] = d0
			ps315.OverlayValues[1] = d1
			ps315.OverlayValues[2] = d2
			ps315.OverlayValues[3] = d3
			ps315.OverlayValues[4] = d4
			ps315.OverlayValues[5] = d5
			ps315.OverlayValues[6] = d6
			ps315.OverlayValues[7] = d7
			ps315.OverlayValues[8] = d8
			ps315.OverlayValues[9] = d9
			ps315.OverlayValues[10] = d10
			ps315.OverlayValues[11] = d11
			ps315.OverlayValues[12] = d12
			ps315.OverlayValues[13] = d13
			ps315.OverlayValues[14] = d14
			ps315.OverlayValues[15] = d15
			ps315.OverlayValues[16] = d16
			ps315.OverlayValues[17] = d17
			ps315.OverlayValues[18] = d18
			ps315.OverlayValues[19] = d19
			ps315.OverlayValues[20] = d20
			ps315.OverlayValues[21] = d21
			ps315.OverlayValues[22] = d22
			ps315.OverlayValues[23] = d23
			ps315.OverlayValues[24] = d24
			ps315.OverlayValues[25] = d25
			ps315.OverlayValues[26] = d26
			ps315.OverlayValues[27] = d27
			ps315.OverlayValues[28] = d28
			ps315.OverlayValues[29] = d29
			ps315.OverlayValues[30] = d30
			ps315.OverlayValues[31] = d31
			ps315.OverlayValues[32] = d32
			ps315.OverlayValues[33] = d33
			ps315.OverlayValues[34] = d34
			ps315.OverlayValues[35] = d35
			ps315.OverlayValues[36] = d36
			ps315.OverlayValues[37] = d37
			ps315.OverlayValues[38] = d38
			ps315.OverlayValues[39] = d39
			ps315.OverlayValues[40] = d40
			ps315.OverlayValues[41] = d41
			ps315.OverlayValues[42] = d42
			ps315.OverlayValues[43] = d43
			ps315.OverlayValues[44] = d44
			ps315.OverlayValues[45] = d45
			ps315.OverlayValues[46] = d46
			ps315.OverlayValues[47] = d47
			ps315.OverlayValues[48] = d48
			ps315.OverlayValues[49] = d49
			ps315.OverlayValues[50] = d50
			ps315.OverlayValues[51] = d51
			ps315.OverlayValues[52] = d52
			ps315.OverlayValues[53] = d53
			ps315.OverlayValues[54] = d54
			ps315.OverlayValues[55] = d55
			ps315.OverlayValues[56] = d56
			ps315.OverlayValues[57] = d57
			ps315.OverlayValues[58] = d58
			ps315.OverlayValues[59] = d59
			ps315.OverlayValues[60] = d60
			ps315.OverlayValues[61] = d61
			ps315.OverlayValues[62] = d62
			ps315.OverlayValues[63] = d63
			ps315.OverlayValues[64] = d64
			ps315.OverlayValues[65] = d65
			ps315.OverlayValues[66] = d66
			ps315.OverlayValues[67] = d67
			ps315.OverlayValues[68] = d68
			ps315.OverlayValues[69] = d69
			ps315.OverlayValues[70] = d70
			ps315.OverlayValues[71] = d71
			ps315.OverlayValues[72] = d72
			ps315.OverlayValues[73] = d73
			ps315.OverlayValues[74] = d74
			ps315.OverlayValues[75] = d75
			ps315.OverlayValues[76] = d76
			ps315.OverlayValues[77] = d77
			ps315.OverlayValues[78] = d78
			ps315.OverlayValues[79] = d79
			ps315.OverlayValues[80] = d80
			ps315.OverlayValues[81] = d81
			ps315.OverlayValues[82] = d82
			ps315.OverlayValues[83] = d83
			ps315.OverlayValues[84] = d84
			ps315.OverlayValues[85] = d85
			ps315.OverlayValues[86] = d86
			ps315.OverlayValues[87] = d87
			ps315.OverlayValues[88] = d88
			ps315.OverlayValues[89] = d89
			ps315.OverlayValues[90] = d90
			ps315.OverlayValues[91] = d91
			ps315.OverlayValues[92] = d92
			ps315.OverlayValues[93] = d93
			ps315.OverlayValues[94] = d94
			ps315.OverlayValues[95] = d95
			ps315.OverlayValues[96] = d96
			ps315.OverlayValues[97] = d97
			ps315.OverlayValues[98] = d98
			ps315.OverlayValues[99] = d99
			ps315.OverlayValues[100] = d100
			ps315.OverlayValues[101] = d101
			ps315.OverlayValues[102] = d102
			ps315.OverlayValues[103] = d103
			ps315.OverlayValues[104] = d104
			ps315.OverlayValues[105] = d105
			ps315.OverlayValues[106] = d106
			ps315.OverlayValues[107] = d107
			ps315.OverlayValues[108] = d108
			ps315.OverlayValues[109] = d109
			ps315.OverlayValues[110] = d110
			ps315.OverlayValues[111] = d111
			ps315.OverlayValues[112] = d112
			ps315.OverlayValues[113] = d113
			ps315.OverlayValues[114] = d114
			ps315.OverlayValues[115] = d115
			ps315.OverlayValues[116] = d116
			ps315.OverlayValues[117] = d117
			ps315.OverlayValues[118] = d118
			ps315.OverlayValues[119] = d119
			ps315.OverlayValues[120] = d120
			ps315.OverlayValues[121] = d121
			ps315.OverlayValues[122] = d122
			ps315.OverlayValues[123] = d123
			ps315.OverlayValues[124] = d124
			ps315.OverlayValues[125] = d125
			ps315.OverlayValues[126] = d126
			ps315.OverlayValues[127] = d127
			ps315.OverlayValues[128] = d128
			ps315.OverlayValues[129] = d129
			ps315.OverlayValues[130] = d130
			ps315.OverlayValues[131] = d131
			ps315.OverlayValues[132] = d132
			ps315.OverlayValues[133] = d133
			ps315.OverlayValues[134] = d134
			ps315.OverlayValues[135] = d135
			ps315.OverlayValues[136] = d136
			ps315.OverlayValues[137] = d137
			ps315.OverlayValues[138] = d138
			ps315.OverlayValues[139] = d139
			ps315.OverlayValues[140] = d140
			ps315.OverlayValues[141] = d141
			ps315.OverlayValues[142] = d142
			ps315.OverlayValues[143] = d143
			ps315.OverlayValues[144] = d144
			ps315.OverlayValues[145] = d145
			ps315.OverlayValues[146] = d146
			ps315.OverlayValues[147] = d147
			ps315.OverlayValues[148] = d148
			ps315.OverlayValues[149] = d149
			ps315.OverlayValues[150] = d150
			ps315.OverlayValues[151] = d151
			ps315.OverlayValues[152] = d152
			ps315.OverlayValues[153] = d153
			ps315.OverlayValues[154] = d154
			ps315.OverlayValues[155] = d155
			ps315.OverlayValues[156] = d156
			ps315.OverlayValues[157] = d157
			ps315.OverlayValues[158] = d158
			ps315.OverlayValues[159] = d159
			ps315.OverlayValues[160] = d160
			ps315.OverlayValues[161] = d161
			ps315.OverlayValues[162] = d162
			ps315.OverlayValues[163] = d163
			ps315.OverlayValues[164] = d164
			ps315.OverlayValues[165] = d165
			ps315.OverlayValues[166] = d166
			ps315.OverlayValues[167] = d167
			ps315.OverlayValues[168] = d168
			ps315.OverlayValues[169] = d169
			ps315.OverlayValues[170] = d170
			ps315.OverlayValues[171] = d171
			ps315.OverlayValues[172] = d172
			ps315.OverlayValues[173] = d173
			ps315.OverlayValues[174] = d174
			ps315.OverlayValues[175] = d175
			ps315.OverlayValues[176] = d176
			ps315.OverlayValues[177] = d177
			ps315.OverlayValues[178] = d178
			ps315.OverlayValues[179] = d179
			ps315.OverlayValues[180] = d180
			ps315.OverlayValues[181] = d181
			ps315.OverlayValues[182] = d182
			ps315.OverlayValues[183] = d183
			ps315.OverlayValues[184] = d184
			ps315.OverlayValues[185] = d185
			ps315.OverlayValues[186] = d186
			ps315.OverlayValues[187] = d187
			ps315.OverlayValues[188] = d188
			ps315.OverlayValues[189] = d189
			ps315.OverlayValues[190] = d190
			ps315.OverlayValues[191] = d191
			ps315.OverlayValues[192] = d192
			ps315.OverlayValues[193] = d193
			ps315.OverlayValues[194] = d194
			ps315.OverlayValues[195] = d195
			ps315.OverlayValues[196] = d196
			ps315.OverlayValues[197] = d197
			ps315.OverlayValues[198] = d198
			ps315.OverlayValues[199] = d199
			ps315.OverlayValues[200] = d200
			ps315.OverlayValues[201] = d201
			ps315.OverlayValues[202] = d202
			ps315.OverlayValues[203] = d203
			ps315.OverlayValues[204] = d204
			ps315.OverlayValues[205] = d205
			ps315.OverlayValues[206] = d206
			ps315.OverlayValues[207] = d207
			ps315.OverlayValues[208] = d208
			ps315.OverlayValues[209] = d209
			ps315.OverlayValues[210] = d210
			ps315.OverlayValues[211] = d211
			ps315.OverlayValues[212] = d212
			ps315.OverlayValues[213] = d213
			ps315.OverlayValues[214] = d214
			ps315.OverlayValues[215] = d215
			ps315.OverlayValues[216] = d216
			ps315.OverlayValues[217] = d217
			ps315.OverlayValues[218] = d218
			ps315.OverlayValues[219] = d219
			ps315.OverlayValues[220] = d220
			ps315.OverlayValues[221] = d221
			ps315.OverlayValues[222] = d222
			ps315.OverlayValues[223] = d223
			ps315.OverlayValues[224] = d224
			ps315.OverlayValues[225] = d225
			ps315.OverlayValues[226] = d226
			ps315.OverlayValues[227] = d227
			ps315.OverlayValues[228] = d228
			ps315.OverlayValues[229] = d229
			ps315.OverlayValues[230] = d230
			ps315.OverlayValues[231] = d231
			ps315.OverlayValues[232] = d232
			ps315.OverlayValues[233] = d233
			ps315.OverlayValues[234] = d234
			ps315.OverlayValues[235] = d235
			ps315.OverlayValues[236] = d236
			ps315.OverlayValues[237] = d237
			ps315.OverlayValues[238] = d238
			ps315.OverlayValues[239] = d239
			ps315.OverlayValues[240] = d240
			ps315.OverlayValues[246] = d246
			ps315.OverlayValues[247] = d247
			ps315.OverlayValues[248] = d248
			ps315.OverlayValues[249] = d249
			ps315.OverlayValues[255] = d255
			ps315.OverlayValues[256] = d256
			ps315.OverlayValues[257] = d257
			ps315.OverlayValues[258] = d258
			ps315.OverlayValues[259] = d259
			ps315.OverlayValues[260] = d260
			ps315.OverlayValues[261] = d261
			ps315.OverlayValues[262] = d262
			ps315.OverlayValues[263] = d263
			ps315.OverlayValues[264] = d264
			ps315.OverlayValues[265] = d265
			ps315.OverlayValues[266] = d266
			ps315.OverlayValues[267] = d267
			ps315.OverlayValues[268] = d268
			ps315.OverlayValues[269] = d269
			ps315.OverlayValues[270] = d270
			ps315.OverlayValues[271] = d271
			ps315.OverlayValues[272] = d272
			ps315.OverlayValues[273] = d273
			ps315.OverlayValues[274] = d274
			ps315.OverlayValues[275] = d275
			ps315.OverlayValues[276] = d276
			ps315.OverlayValues[277] = d277
			ps315.OverlayValues[278] = d278
			ps315.OverlayValues[279] = d279
			ps315.OverlayValues[280] = d280
			ps315.OverlayValues[281] = d281
			ps315.OverlayValues[282] = d282
			ps315.OverlayValues[283] = d283
			ps315.OverlayValues[284] = d284
			ps315.OverlayValues[285] = d285
			ps315.OverlayValues[286] = d286
			ps315.OverlayValues[287] = d287
			ps315.OverlayValues[288] = d288
			ps315.OverlayValues[289] = d289
			ps315.OverlayValues[290] = d290
			ps315.OverlayValues[291] = d291
			ps315.OverlayValues[292] = d292
			ps315.OverlayValues[293] = d293
			ps315.OverlayValues[294] = d294
			ps315.OverlayValues[295] = d295
			ps315.OverlayValues[296] = d296
			ps315.OverlayValues[297] = d297
			ps315.OverlayValues[298] = d298
			ps315.OverlayValues[299] = d299
			ps315.OverlayValues[300] = d300
			ps315.OverlayValues[306] = d306
			ps315.OverlayValues[307] = d307
			ps315.OverlayValues[308] = d308
			ps315.OverlayValues[309] = d309
			ps315.OverlayValues[310] = d310
			ps315.OverlayValues[311] = d311
			ps315.OverlayValues[312] = d312
			ps315.OverlayValues[313] = d313
				return bbs[6].RenderPS(ps315)
			}
			if !ps.General {
				ps.General = true
				return bbs[7].RenderPS(ps)
			}
			lbl82 := ctx.W.ReserveLabel()
			lbl83 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d313.Reg, 0)
			ctx.W.EmitJcc(scm.CcNE, lbl82)
			ctx.W.EmitJmp(lbl83)
			ctx.W.MarkLabel(lbl82)
			ctx.W.EmitJmp(lbl6)
			ctx.W.MarkLabel(lbl83)
			ctx.W.EmitJmp(lbl7)
			ps316 := scm.PhiState{General: true}
			ps316.OverlayValues = make([]scm.JITValueDesc, 314)
			ps316.OverlayValues[0] = d0
			ps316.OverlayValues[1] = d1
			ps316.OverlayValues[2] = d2
			ps316.OverlayValues[3] = d3
			ps316.OverlayValues[4] = d4
			ps316.OverlayValues[5] = d5
			ps316.OverlayValues[6] = d6
			ps316.OverlayValues[7] = d7
			ps316.OverlayValues[8] = d8
			ps316.OverlayValues[9] = d9
			ps316.OverlayValues[10] = d10
			ps316.OverlayValues[11] = d11
			ps316.OverlayValues[12] = d12
			ps316.OverlayValues[13] = d13
			ps316.OverlayValues[14] = d14
			ps316.OverlayValues[15] = d15
			ps316.OverlayValues[16] = d16
			ps316.OverlayValues[17] = d17
			ps316.OverlayValues[18] = d18
			ps316.OverlayValues[19] = d19
			ps316.OverlayValues[20] = d20
			ps316.OverlayValues[21] = d21
			ps316.OverlayValues[22] = d22
			ps316.OverlayValues[23] = d23
			ps316.OverlayValues[24] = d24
			ps316.OverlayValues[25] = d25
			ps316.OverlayValues[26] = d26
			ps316.OverlayValues[27] = d27
			ps316.OverlayValues[28] = d28
			ps316.OverlayValues[29] = d29
			ps316.OverlayValues[30] = d30
			ps316.OverlayValues[31] = d31
			ps316.OverlayValues[32] = d32
			ps316.OverlayValues[33] = d33
			ps316.OverlayValues[34] = d34
			ps316.OverlayValues[35] = d35
			ps316.OverlayValues[36] = d36
			ps316.OverlayValues[37] = d37
			ps316.OverlayValues[38] = d38
			ps316.OverlayValues[39] = d39
			ps316.OverlayValues[40] = d40
			ps316.OverlayValues[41] = d41
			ps316.OverlayValues[42] = d42
			ps316.OverlayValues[43] = d43
			ps316.OverlayValues[44] = d44
			ps316.OverlayValues[45] = d45
			ps316.OverlayValues[46] = d46
			ps316.OverlayValues[47] = d47
			ps316.OverlayValues[48] = d48
			ps316.OverlayValues[49] = d49
			ps316.OverlayValues[50] = d50
			ps316.OverlayValues[51] = d51
			ps316.OverlayValues[52] = d52
			ps316.OverlayValues[53] = d53
			ps316.OverlayValues[54] = d54
			ps316.OverlayValues[55] = d55
			ps316.OverlayValues[56] = d56
			ps316.OverlayValues[57] = d57
			ps316.OverlayValues[58] = d58
			ps316.OverlayValues[59] = d59
			ps316.OverlayValues[60] = d60
			ps316.OverlayValues[61] = d61
			ps316.OverlayValues[62] = d62
			ps316.OverlayValues[63] = d63
			ps316.OverlayValues[64] = d64
			ps316.OverlayValues[65] = d65
			ps316.OverlayValues[66] = d66
			ps316.OverlayValues[67] = d67
			ps316.OverlayValues[68] = d68
			ps316.OverlayValues[69] = d69
			ps316.OverlayValues[70] = d70
			ps316.OverlayValues[71] = d71
			ps316.OverlayValues[72] = d72
			ps316.OverlayValues[73] = d73
			ps316.OverlayValues[74] = d74
			ps316.OverlayValues[75] = d75
			ps316.OverlayValues[76] = d76
			ps316.OverlayValues[77] = d77
			ps316.OverlayValues[78] = d78
			ps316.OverlayValues[79] = d79
			ps316.OverlayValues[80] = d80
			ps316.OverlayValues[81] = d81
			ps316.OverlayValues[82] = d82
			ps316.OverlayValues[83] = d83
			ps316.OverlayValues[84] = d84
			ps316.OverlayValues[85] = d85
			ps316.OverlayValues[86] = d86
			ps316.OverlayValues[87] = d87
			ps316.OverlayValues[88] = d88
			ps316.OverlayValues[89] = d89
			ps316.OverlayValues[90] = d90
			ps316.OverlayValues[91] = d91
			ps316.OverlayValues[92] = d92
			ps316.OverlayValues[93] = d93
			ps316.OverlayValues[94] = d94
			ps316.OverlayValues[95] = d95
			ps316.OverlayValues[96] = d96
			ps316.OverlayValues[97] = d97
			ps316.OverlayValues[98] = d98
			ps316.OverlayValues[99] = d99
			ps316.OverlayValues[100] = d100
			ps316.OverlayValues[101] = d101
			ps316.OverlayValues[102] = d102
			ps316.OverlayValues[103] = d103
			ps316.OverlayValues[104] = d104
			ps316.OverlayValues[105] = d105
			ps316.OverlayValues[106] = d106
			ps316.OverlayValues[107] = d107
			ps316.OverlayValues[108] = d108
			ps316.OverlayValues[109] = d109
			ps316.OverlayValues[110] = d110
			ps316.OverlayValues[111] = d111
			ps316.OverlayValues[112] = d112
			ps316.OverlayValues[113] = d113
			ps316.OverlayValues[114] = d114
			ps316.OverlayValues[115] = d115
			ps316.OverlayValues[116] = d116
			ps316.OverlayValues[117] = d117
			ps316.OverlayValues[118] = d118
			ps316.OverlayValues[119] = d119
			ps316.OverlayValues[120] = d120
			ps316.OverlayValues[121] = d121
			ps316.OverlayValues[122] = d122
			ps316.OverlayValues[123] = d123
			ps316.OverlayValues[124] = d124
			ps316.OverlayValues[125] = d125
			ps316.OverlayValues[126] = d126
			ps316.OverlayValues[127] = d127
			ps316.OverlayValues[128] = d128
			ps316.OverlayValues[129] = d129
			ps316.OverlayValues[130] = d130
			ps316.OverlayValues[131] = d131
			ps316.OverlayValues[132] = d132
			ps316.OverlayValues[133] = d133
			ps316.OverlayValues[134] = d134
			ps316.OverlayValues[135] = d135
			ps316.OverlayValues[136] = d136
			ps316.OverlayValues[137] = d137
			ps316.OverlayValues[138] = d138
			ps316.OverlayValues[139] = d139
			ps316.OverlayValues[140] = d140
			ps316.OverlayValues[141] = d141
			ps316.OverlayValues[142] = d142
			ps316.OverlayValues[143] = d143
			ps316.OverlayValues[144] = d144
			ps316.OverlayValues[145] = d145
			ps316.OverlayValues[146] = d146
			ps316.OverlayValues[147] = d147
			ps316.OverlayValues[148] = d148
			ps316.OverlayValues[149] = d149
			ps316.OverlayValues[150] = d150
			ps316.OverlayValues[151] = d151
			ps316.OverlayValues[152] = d152
			ps316.OverlayValues[153] = d153
			ps316.OverlayValues[154] = d154
			ps316.OverlayValues[155] = d155
			ps316.OverlayValues[156] = d156
			ps316.OverlayValues[157] = d157
			ps316.OverlayValues[158] = d158
			ps316.OverlayValues[159] = d159
			ps316.OverlayValues[160] = d160
			ps316.OverlayValues[161] = d161
			ps316.OverlayValues[162] = d162
			ps316.OverlayValues[163] = d163
			ps316.OverlayValues[164] = d164
			ps316.OverlayValues[165] = d165
			ps316.OverlayValues[166] = d166
			ps316.OverlayValues[167] = d167
			ps316.OverlayValues[168] = d168
			ps316.OverlayValues[169] = d169
			ps316.OverlayValues[170] = d170
			ps316.OverlayValues[171] = d171
			ps316.OverlayValues[172] = d172
			ps316.OverlayValues[173] = d173
			ps316.OverlayValues[174] = d174
			ps316.OverlayValues[175] = d175
			ps316.OverlayValues[176] = d176
			ps316.OverlayValues[177] = d177
			ps316.OverlayValues[178] = d178
			ps316.OverlayValues[179] = d179
			ps316.OverlayValues[180] = d180
			ps316.OverlayValues[181] = d181
			ps316.OverlayValues[182] = d182
			ps316.OverlayValues[183] = d183
			ps316.OverlayValues[184] = d184
			ps316.OverlayValues[185] = d185
			ps316.OverlayValues[186] = d186
			ps316.OverlayValues[187] = d187
			ps316.OverlayValues[188] = d188
			ps316.OverlayValues[189] = d189
			ps316.OverlayValues[190] = d190
			ps316.OverlayValues[191] = d191
			ps316.OverlayValues[192] = d192
			ps316.OverlayValues[193] = d193
			ps316.OverlayValues[194] = d194
			ps316.OverlayValues[195] = d195
			ps316.OverlayValues[196] = d196
			ps316.OverlayValues[197] = d197
			ps316.OverlayValues[198] = d198
			ps316.OverlayValues[199] = d199
			ps316.OverlayValues[200] = d200
			ps316.OverlayValues[201] = d201
			ps316.OverlayValues[202] = d202
			ps316.OverlayValues[203] = d203
			ps316.OverlayValues[204] = d204
			ps316.OverlayValues[205] = d205
			ps316.OverlayValues[206] = d206
			ps316.OverlayValues[207] = d207
			ps316.OverlayValues[208] = d208
			ps316.OverlayValues[209] = d209
			ps316.OverlayValues[210] = d210
			ps316.OverlayValues[211] = d211
			ps316.OverlayValues[212] = d212
			ps316.OverlayValues[213] = d213
			ps316.OverlayValues[214] = d214
			ps316.OverlayValues[215] = d215
			ps316.OverlayValues[216] = d216
			ps316.OverlayValues[217] = d217
			ps316.OverlayValues[218] = d218
			ps316.OverlayValues[219] = d219
			ps316.OverlayValues[220] = d220
			ps316.OverlayValues[221] = d221
			ps316.OverlayValues[222] = d222
			ps316.OverlayValues[223] = d223
			ps316.OverlayValues[224] = d224
			ps316.OverlayValues[225] = d225
			ps316.OverlayValues[226] = d226
			ps316.OverlayValues[227] = d227
			ps316.OverlayValues[228] = d228
			ps316.OverlayValues[229] = d229
			ps316.OverlayValues[230] = d230
			ps316.OverlayValues[231] = d231
			ps316.OverlayValues[232] = d232
			ps316.OverlayValues[233] = d233
			ps316.OverlayValues[234] = d234
			ps316.OverlayValues[235] = d235
			ps316.OverlayValues[236] = d236
			ps316.OverlayValues[237] = d237
			ps316.OverlayValues[238] = d238
			ps316.OverlayValues[239] = d239
			ps316.OverlayValues[240] = d240
			ps316.OverlayValues[246] = d246
			ps316.OverlayValues[247] = d247
			ps316.OverlayValues[248] = d248
			ps316.OverlayValues[249] = d249
			ps316.OverlayValues[255] = d255
			ps316.OverlayValues[256] = d256
			ps316.OverlayValues[257] = d257
			ps316.OverlayValues[258] = d258
			ps316.OverlayValues[259] = d259
			ps316.OverlayValues[260] = d260
			ps316.OverlayValues[261] = d261
			ps316.OverlayValues[262] = d262
			ps316.OverlayValues[263] = d263
			ps316.OverlayValues[264] = d264
			ps316.OverlayValues[265] = d265
			ps316.OverlayValues[266] = d266
			ps316.OverlayValues[267] = d267
			ps316.OverlayValues[268] = d268
			ps316.OverlayValues[269] = d269
			ps316.OverlayValues[270] = d270
			ps316.OverlayValues[271] = d271
			ps316.OverlayValues[272] = d272
			ps316.OverlayValues[273] = d273
			ps316.OverlayValues[274] = d274
			ps316.OverlayValues[275] = d275
			ps316.OverlayValues[276] = d276
			ps316.OverlayValues[277] = d277
			ps316.OverlayValues[278] = d278
			ps316.OverlayValues[279] = d279
			ps316.OverlayValues[280] = d280
			ps316.OverlayValues[281] = d281
			ps316.OverlayValues[282] = d282
			ps316.OverlayValues[283] = d283
			ps316.OverlayValues[284] = d284
			ps316.OverlayValues[285] = d285
			ps316.OverlayValues[286] = d286
			ps316.OverlayValues[287] = d287
			ps316.OverlayValues[288] = d288
			ps316.OverlayValues[289] = d289
			ps316.OverlayValues[290] = d290
			ps316.OverlayValues[291] = d291
			ps316.OverlayValues[292] = d292
			ps316.OverlayValues[293] = d293
			ps316.OverlayValues[294] = d294
			ps316.OverlayValues[295] = d295
			ps316.OverlayValues[296] = d296
			ps316.OverlayValues[297] = d297
			ps316.OverlayValues[298] = d298
			ps316.OverlayValues[299] = d299
			ps316.OverlayValues[300] = d300
			ps316.OverlayValues[306] = d306
			ps316.OverlayValues[307] = d307
			ps316.OverlayValues[308] = d308
			ps316.OverlayValues[309] = d309
			ps316.OverlayValues[310] = d310
			ps316.OverlayValues[311] = d311
			ps316.OverlayValues[312] = d312
			ps316.OverlayValues[313] = d313
			ps317 := scm.PhiState{General: true}
			ps317.OverlayValues = make([]scm.JITValueDesc, 314)
			ps317.OverlayValues[0] = d0
			ps317.OverlayValues[1] = d1
			ps317.OverlayValues[2] = d2
			ps317.OverlayValues[3] = d3
			ps317.OverlayValues[4] = d4
			ps317.OverlayValues[5] = d5
			ps317.OverlayValues[6] = d6
			ps317.OverlayValues[7] = d7
			ps317.OverlayValues[8] = d8
			ps317.OverlayValues[9] = d9
			ps317.OverlayValues[10] = d10
			ps317.OverlayValues[11] = d11
			ps317.OverlayValues[12] = d12
			ps317.OverlayValues[13] = d13
			ps317.OverlayValues[14] = d14
			ps317.OverlayValues[15] = d15
			ps317.OverlayValues[16] = d16
			ps317.OverlayValues[17] = d17
			ps317.OverlayValues[18] = d18
			ps317.OverlayValues[19] = d19
			ps317.OverlayValues[20] = d20
			ps317.OverlayValues[21] = d21
			ps317.OverlayValues[22] = d22
			ps317.OverlayValues[23] = d23
			ps317.OverlayValues[24] = d24
			ps317.OverlayValues[25] = d25
			ps317.OverlayValues[26] = d26
			ps317.OverlayValues[27] = d27
			ps317.OverlayValues[28] = d28
			ps317.OverlayValues[29] = d29
			ps317.OverlayValues[30] = d30
			ps317.OverlayValues[31] = d31
			ps317.OverlayValues[32] = d32
			ps317.OverlayValues[33] = d33
			ps317.OverlayValues[34] = d34
			ps317.OverlayValues[35] = d35
			ps317.OverlayValues[36] = d36
			ps317.OverlayValues[37] = d37
			ps317.OverlayValues[38] = d38
			ps317.OverlayValues[39] = d39
			ps317.OverlayValues[40] = d40
			ps317.OverlayValues[41] = d41
			ps317.OverlayValues[42] = d42
			ps317.OverlayValues[43] = d43
			ps317.OverlayValues[44] = d44
			ps317.OverlayValues[45] = d45
			ps317.OverlayValues[46] = d46
			ps317.OverlayValues[47] = d47
			ps317.OverlayValues[48] = d48
			ps317.OverlayValues[49] = d49
			ps317.OverlayValues[50] = d50
			ps317.OverlayValues[51] = d51
			ps317.OverlayValues[52] = d52
			ps317.OverlayValues[53] = d53
			ps317.OverlayValues[54] = d54
			ps317.OverlayValues[55] = d55
			ps317.OverlayValues[56] = d56
			ps317.OverlayValues[57] = d57
			ps317.OverlayValues[58] = d58
			ps317.OverlayValues[59] = d59
			ps317.OverlayValues[60] = d60
			ps317.OverlayValues[61] = d61
			ps317.OverlayValues[62] = d62
			ps317.OverlayValues[63] = d63
			ps317.OverlayValues[64] = d64
			ps317.OverlayValues[65] = d65
			ps317.OverlayValues[66] = d66
			ps317.OverlayValues[67] = d67
			ps317.OverlayValues[68] = d68
			ps317.OverlayValues[69] = d69
			ps317.OverlayValues[70] = d70
			ps317.OverlayValues[71] = d71
			ps317.OverlayValues[72] = d72
			ps317.OverlayValues[73] = d73
			ps317.OverlayValues[74] = d74
			ps317.OverlayValues[75] = d75
			ps317.OverlayValues[76] = d76
			ps317.OverlayValues[77] = d77
			ps317.OverlayValues[78] = d78
			ps317.OverlayValues[79] = d79
			ps317.OverlayValues[80] = d80
			ps317.OverlayValues[81] = d81
			ps317.OverlayValues[82] = d82
			ps317.OverlayValues[83] = d83
			ps317.OverlayValues[84] = d84
			ps317.OverlayValues[85] = d85
			ps317.OverlayValues[86] = d86
			ps317.OverlayValues[87] = d87
			ps317.OverlayValues[88] = d88
			ps317.OverlayValues[89] = d89
			ps317.OverlayValues[90] = d90
			ps317.OverlayValues[91] = d91
			ps317.OverlayValues[92] = d92
			ps317.OverlayValues[93] = d93
			ps317.OverlayValues[94] = d94
			ps317.OverlayValues[95] = d95
			ps317.OverlayValues[96] = d96
			ps317.OverlayValues[97] = d97
			ps317.OverlayValues[98] = d98
			ps317.OverlayValues[99] = d99
			ps317.OverlayValues[100] = d100
			ps317.OverlayValues[101] = d101
			ps317.OverlayValues[102] = d102
			ps317.OverlayValues[103] = d103
			ps317.OverlayValues[104] = d104
			ps317.OverlayValues[105] = d105
			ps317.OverlayValues[106] = d106
			ps317.OverlayValues[107] = d107
			ps317.OverlayValues[108] = d108
			ps317.OverlayValues[109] = d109
			ps317.OverlayValues[110] = d110
			ps317.OverlayValues[111] = d111
			ps317.OverlayValues[112] = d112
			ps317.OverlayValues[113] = d113
			ps317.OverlayValues[114] = d114
			ps317.OverlayValues[115] = d115
			ps317.OverlayValues[116] = d116
			ps317.OverlayValues[117] = d117
			ps317.OverlayValues[118] = d118
			ps317.OverlayValues[119] = d119
			ps317.OverlayValues[120] = d120
			ps317.OverlayValues[121] = d121
			ps317.OverlayValues[122] = d122
			ps317.OverlayValues[123] = d123
			ps317.OverlayValues[124] = d124
			ps317.OverlayValues[125] = d125
			ps317.OverlayValues[126] = d126
			ps317.OverlayValues[127] = d127
			ps317.OverlayValues[128] = d128
			ps317.OverlayValues[129] = d129
			ps317.OverlayValues[130] = d130
			ps317.OverlayValues[131] = d131
			ps317.OverlayValues[132] = d132
			ps317.OverlayValues[133] = d133
			ps317.OverlayValues[134] = d134
			ps317.OverlayValues[135] = d135
			ps317.OverlayValues[136] = d136
			ps317.OverlayValues[137] = d137
			ps317.OverlayValues[138] = d138
			ps317.OverlayValues[139] = d139
			ps317.OverlayValues[140] = d140
			ps317.OverlayValues[141] = d141
			ps317.OverlayValues[142] = d142
			ps317.OverlayValues[143] = d143
			ps317.OverlayValues[144] = d144
			ps317.OverlayValues[145] = d145
			ps317.OverlayValues[146] = d146
			ps317.OverlayValues[147] = d147
			ps317.OverlayValues[148] = d148
			ps317.OverlayValues[149] = d149
			ps317.OverlayValues[150] = d150
			ps317.OverlayValues[151] = d151
			ps317.OverlayValues[152] = d152
			ps317.OverlayValues[153] = d153
			ps317.OverlayValues[154] = d154
			ps317.OverlayValues[155] = d155
			ps317.OverlayValues[156] = d156
			ps317.OverlayValues[157] = d157
			ps317.OverlayValues[158] = d158
			ps317.OverlayValues[159] = d159
			ps317.OverlayValues[160] = d160
			ps317.OverlayValues[161] = d161
			ps317.OverlayValues[162] = d162
			ps317.OverlayValues[163] = d163
			ps317.OverlayValues[164] = d164
			ps317.OverlayValues[165] = d165
			ps317.OverlayValues[166] = d166
			ps317.OverlayValues[167] = d167
			ps317.OverlayValues[168] = d168
			ps317.OverlayValues[169] = d169
			ps317.OverlayValues[170] = d170
			ps317.OverlayValues[171] = d171
			ps317.OverlayValues[172] = d172
			ps317.OverlayValues[173] = d173
			ps317.OverlayValues[174] = d174
			ps317.OverlayValues[175] = d175
			ps317.OverlayValues[176] = d176
			ps317.OverlayValues[177] = d177
			ps317.OverlayValues[178] = d178
			ps317.OverlayValues[179] = d179
			ps317.OverlayValues[180] = d180
			ps317.OverlayValues[181] = d181
			ps317.OverlayValues[182] = d182
			ps317.OverlayValues[183] = d183
			ps317.OverlayValues[184] = d184
			ps317.OverlayValues[185] = d185
			ps317.OverlayValues[186] = d186
			ps317.OverlayValues[187] = d187
			ps317.OverlayValues[188] = d188
			ps317.OverlayValues[189] = d189
			ps317.OverlayValues[190] = d190
			ps317.OverlayValues[191] = d191
			ps317.OverlayValues[192] = d192
			ps317.OverlayValues[193] = d193
			ps317.OverlayValues[194] = d194
			ps317.OverlayValues[195] = d195
			ps317.OverlayValues[196] = d196
			ps317.OverlayValues[197] = d197
			ps317.OverlayValues[198] = d198
			ps317.OverlayValues[199] = d199
			ps317.OverlayValues[200] = d200
			ps317.OverlayValues[201] = d201
			ps317.OverlayValues[202] = d202
			ps317.OverlayValues[203] = d203
			ps317.OverlayValues[204] = d204
			ps317.OverlayValues[205] = d205
			ps317.OverlayValues[206] = d206
			ps317.OverlayValues[207] = d207
			ps317.OverlayValues[208] = d208
			ps317.OverlayValues[209] = d209
			ps317.OverlayValues[210] = d210
			ps317.OverlayValues[211] = d211
			ps317.OverlayValues[212] = d212
			ps317.OverlayValues[213] = d213
			ps317.OverlayValues[214] = d214
			ps317.OverlayValues[215] = d215
			ps317.OverlayValues[216] = d216
			ps317.OverlayValues[217] = d217
			ps317.OverlayValues[218] = d218
			ps317.OverlayValues[219] = d219
			ps317.OverlayValues[220] = d220
			ps317.OverlayValues[221] = d221
			ps317.OverlayValues[222] = d222
			ps317.OverlayValues[223] = d223
			ps317.OverlayValues[224] = d224
			ps317.OverlayValues[225] = d225
			ps317.OverlayValues[226] = d226
			ps317.OverlayValues[227] = d227
			ps317.OverlayValues[228] = d228
			ps317.OverlayValues[229] = d229
			ps317.OverlayValues[230] = d230
			ps317.OverlayValues[231] = d231
			ps317.OverlayValues[232] = d232
			ps317.OverlayValues[233] = d233
			ps317.OverlayValues[234] = d234
			ps317.OverlayValues[235] = d235
			ps317.OverlayValues[236] = d236
			ps317.OverlayValues[237] = d237
			ps317.OverlayValues[238] = d238
			ps317.OverlayValues[239] = d239
			ps317.OverlayValues[240] = d240
			ps317.OverlayValues[246] = d246
			ps317.OverlayValues[247] = d247
			ps317.OverlayValues[248] = d248
			ps317.OverlayValues[249] = d249
			ps317.OverlayValues[255] = d255
			ps317.OverlayValues[256] = d256
			ps317.OverlayValues[257] = d257
			ps317.OverlayValues[258] = d258
			ps317.OverlayValues[259] = d259
			ps317.OverlayValues[260] = d260
			ps317.OverlayValues[261] = d261
			ps317.OverlayValues[262] = d262
			ps317.OverlayValues[263] = d263
			ps317.OverlayValues[264] = d264
			ps317.OverlayValues[265] = d265
			ps317.OverlayValues[266] = d266
			ps317.OverlayValues[267] = d267
			ps317.OverlayValues[268] = d268
			ps317.OverlayValues[269] = d269
			ps317.OverlayValues[270] = d270
			ps317.OverlayValues[271] = d271
			ps317.OverlayValues[272] = d272
			ps317.OverlayValues[273] = d273
			ps317.OverlayValues[274] = d274
			ps317.OverlayValues[275] = d275
			ps317.OverlayValues[276] = d276
			ps317.OverlayValues[277] = d277
			ps317.OverlayValues[278] = d278
			ps317.OverlayValues[279] = d279
			ps317.OverlayValues[280] = d280
			ps317.OverlayValues[281] = d281
			ps317.OverlayValues[282] = d282
			ps317.OverlayValues[283] = d283
			ps317.OverlayValues[284] = d284
			ps317.OverlayValues[285] = d285
			ps317.OverlayValues[286] = d286
			ps317.OverlayValues[287] = d287
			ps317.OverlayValues[288] = d288
			ps317.OverlayValues[289] = d289
			ps317.OverlayValues[290] = d290
			ps317.OverlayValues[291] = d291
			ps317.OverlayValues[292] = d292
			ps317.OverlayValues[293] = d293
			ps317.OverlayValues[294] = d294
			ps317.OverlayValues[295] = d295
			ps317.OverlayValues[296] = d296
			ps317.OverlayValues[297] = d297
			ps317.OverlayValues[298] = d298
			ps317.OverlayValues[299] = d299
			ps317.OverlayValues[300] = d300
			ps317.OverlayValues[306] = d306
			ps317.OverlayValues[307] = d307
			ps317.OverlayValues[308] = d308
			ps317.OverlayValues[309] = d309
			ps317.OverlayValues[310] = d310
			ps317.OverlayValues[311] = d311
			ps317.OverlayValues[312] = d312
			ps317.OverlayValues[313] = d313
			alloc318 := ctx.SnapshotAllocState()
			if !bbs[6].Rendered {
				bbs[6].RenderPS(ps317)
			}
			ctx.RestoreAllocState(alloc318)
			if !bbs[5].Rendered {
				return bbs[5].RenderPS(ps316)
			}
			return result
			ctx.FreeDesc(&d312)
			return result
			}
			ps319 := scm.PhiState{General: false}
			_ = bbs[0].RenderPS(ps319)
			ctx.W.MarkLabel(lbl0)
			d320 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d320)
			ctx.BindReg(r1, &d320)
			ctx.EmitMovPairToResult(&d320, &result)
			ctx.FreeReg(r0)
			ctx.FreeReg(r1)
			ctx.W.ResolveFixups()
			if idxPinned { ctx.UnprotectReg(idxPinnedReg) }
			ctx.W.PatchInt32(r7, int32(48))
			ctx.W.EmitAddRSP32(int32(48))
			return result
}

func (s *StoragePrefix) prepare() {
	// set up scan
	s.prefixes.prepare()
	s.values.prepare()
}
func (s *StoragePrefix) scan(i uint32, value scm.Scmer) {
	if value.IsNil() {
		s.values.scan(i, scm.NewNil())
		return
	}
	v := scm.String(value)

	for pfid := len(s.prefixdictionary) - 1; pfid >= 0; pfid-- {
		if strings.HasPrefix(v, s.prefixdictionary[pfid]) {
			// learn the string stripped from its prefix
			s.prefixes.scan(i, scm.NewInt(int64(pfid)))
			s.values.scan(i, scm.NewString(v[len(s.prefixdictionary[pfid]):]))
			return
		}
	}
}
func (s *StoragePrefix) init(i uint32) {
	s.prefixes.init(i)
	s.values.init(i)
}
func (s *StoragePrefix) build(i uint32, value scm.Scmer) {
	// store
	if value.IsNil() {
		s.values.build(i, scm.NewNil())
		return
	}
	v := scm.String(value)

	for pfid := len(s.prefixdictionary) - 1; pfid >= 0; pfid-- {
		if strings.HasPrefix(v, s.prefixdictionary[pfid]) {
			// learn the string stripped from its prefix
			s.prefixes.build(i, scm.NewInt(int64(pfid)))
			s.values.build(i, scm.NewString(v[len(s.prefixdictionary[pfid]):]))
			return
		}
	}
}
func (s *StoragePrefix) finish() {
	s.prefixes.finish()
	s.values.finish()
}
func (s *StoragePrefix) proposeCompression(i uint32) ColumnStorage {
	// dont't propose another pass
	// TODO: if s.values proposes a StoragePrefix, build it into our cascade??
	return nil
}
