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

func (s *StorageSeq) Serialize(f io.Writer) {
	binary.Write(f, binary.LittleEndian, uint8(11)) // 11 = StorageSeq
	io.WriteString(f, "1234567")                    // dummy
	binary.Write(f, binary.LittleEndian, uint64(s.count))
	binary.Write(f, binary.LittleEndian, uint64(s.seqCount))
	s.recordId.Serialize(f)
	s.start.Serialize(f)
	s.stride.Serialize(f)
}

func (s *StorageSeq) Deserialize(f io.Reader) uint {
	var dummy [7]byte
	f.Read(dummy[:])
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
	if s.seqCount == 0 {
		return scm.NewNil()
	}
	if pivot >= s.seqCount {
		pivot = s.seqCount - 1
	}
	min := uint32(0)
	max := s.seqCount - 1
	for {
		recid := int64(s.recordId.GetValueUInt(pivot)) + s.recordId.offset
		if i < uint32(recid) {
			if pivot == 0 {
				min, max = 0, 0
				break
			}
			max = pivot - 1
			pivot--
		} else {
			min = pivot
			pivot++
			if pivot >= s.seqCount {
				pivot = s.seqCount - 1
			}
		}
		if min == max {
			break // we found the sequence for i
		}

		// also read the next neighbour (we are in the cache line anyway and we achieve O(1) in case the same sequence is read again!)
		recid = int64(s.recordId.GetValueUInt(pivot)) + s.recordId.offset
		if i < uint32(recid) {
			if pivot == 0 {
				min, max = 0, 0
				break
			}
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
			var d20 scm.JITValueDesc
			_ = d20
			var d21 scm.JITValueDesc
			_ = d21
			var d22 scm.JITValueDesc
			_ = d22
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
			var d32 scm.JITValueDesc
			_ = d32
			var d34 scm.JITValueDesc
			_ = d34
			var d35 scm.JITValueDesc
			_ = d35
			var d36 scm.JITValueDesc
			_ = d36
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
			var d153 scm.JITValueDesc
			_ = d153
			var d154 scm.JITValueDesc
			_ = d154
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
			var d168 scm.JITValueDesc
			_ = d168
			var d170 scm.JITValueDesc
			_ = d170
			var d171 scm.JITValueDesc
			_ = d171
			var d174 scm.JITValueDesc
			_ = d174
			var d177 scm.JITValueDesc
			_ = d177
			var d178 scm.JITValueDesc
			_ = d178
			var d179 scm.JITValueDesc
			_ = d179
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
			var d199 scm.JITValueDesc
			_ = d199
			var d200 scm.JITValueDesc
			_ = d200
			var d201 scm.JITValueDesc
			_ = d201
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
			var d278 scm.JITValueDesc
			_ = d278
			var d279 scm.JITValueDesc
			_ = d279
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
			var d294 scm.JITValueDesc
			_ = d294
			var d295 scm.JITValueDesc
			_ = d295
			var d298 scm.JITValueDesc
			_ = d298
			var d302 scm.JITValueDesc
			_ = d302
			var d303 scm.JITValueDesc
			_ = d303
			var d304 scm.JITValueDesc
			_ = d304
			var d305 scm.JITValueDesc
			_ = d305
			var d307 scm.JITValueDesc
			_ = d307
			var d308 scm.JITValueDesc
			_ = d308
			var d310 scm.JITValueDesc
			_ = d310
			var d311 scm.JITValueDesc
			_ = d311
			var d312 scm.JITValueDesc
			_ = d312
			var d313 scm.JITValueDesc
			_ = d313
			var d314 scm.JITValueDesc
			_ = d314
			var d315 scm.JITValueDesc
			_ = d315
			var d317 scm.JITValueDesc
			_ = d317
			var d318 scm.JITValueDesc
			_ = d318
			var d319 scm.JITValueDesc
			_ = d319
			var d320 scm.JITValueDesc
			_ = d320
			var d321 scm.JITValueDesc
			_ = d321
			var d322 scm.JITValueDesc
			_ = d322
			var d323 scm.JITValueDesc
			_ = d323
			var d324 scm.JITValueDesc
			_ = d324
			var d325 scm.JITValueDesc
			_ = d325
			var d326 scm.JITValueDesc
			_ = d326
			var d328 scm.JITValueDesc
			_ = d328
			var d329 scm.JITValueDesc
			_ = d329
			var d330 scm.JITValueDesc
			_ = d330
			var d331 scm.JITValueDesc
			_ = d331
			var d332 scm.JITValueDesc
			_ = d332
			var d333 scm.JITValueDesc
			_ = d333
			var d334 scm.JITValueDesc
			_ = d334
			var d335 scm.JITValueDesc
			_ = d335
			var d336 scm.JITValueDesc
			_ = d336
			var d337 scm.JITValueDesc
			_ = d337
			var d338 scm.JITValueDesc
			_ = d338
			var d339 scm.JITValueDesc
			_ = d339
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
			var d348 scm.JITValueDesc
			_ = d348
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
			var d357 scm.JITValueDesc
			_ = d357
			var d358 scm.JITValueDesc
			_ = d358
			var d359 scm.JITValueDesc
			_ = d359
			var d360 scm.JITValueDesc
			_ = d360
			var d361 scm.JITValueDesc
			_ = d361
			var d362 scm.JITValueDesc
			_ = d362
			var d363 scm.JITValueDesc
			_ = d363
			var d364 scm.JITValueDesc
			_ = d364
			var d365 scm.JITValueDesc
			_ = d365
			var d366 scm.JITValueDesc
			_ = d366
			var d367 scm.JITValueDesc
			_ = d367
			var d368 scm.JITValueDesc
			_ = d368
			var d369 scm.JITValueDesc
			_ = d369
			var d370 scm.JITValueDesc
			_ = d370
			var d371 scm.JITValueDesc
			_ = d371
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
			var d380 scm.JITValueDesc
			_ = d380
			var d381 scm.JITValueDesc
			_ = d381
			var d382 scm.JITValueDesc
			_ = d382
			var d383 scm.JITValueDesc
			_ = d383
			var d384 scm.JITValueDesc
			_ = d384
			var d385 scm.JITValueDesc
			_ = d385
			var d386 scm.JITValueDesc
			_ = d386
			var d387 scm.JITValueDesc
			_ = d387
			var d388 scm.JITValueDesc
			_ = d388
			var d389 scm.JITValueDesc
			_ = d389
			var d390 scm.JITValueDesc
			_ = d390
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
			var d399 scm.JITValueDesc
			_ = d399
			var d400 scm.JITValueDesc
			_ = d400
			var d401 scm.JITValueDesc
			_ = d401
			var d402 scm.JITValueDesc
			_ = d402
			var d403 scm.JITValueDesc
			_ = d403
			var d404 scm.JITValueDesc
			_ = d404
			var d405 scm.JITValueDesc
			_ = d405
			var d406 scm.JITValueDesc
			_ = d406
			var d407 scm.JITValueDesc
			_ = d407
			var d408 scm.JITValueDesc
			_ = d408
			var d409 scm.JITValueDesc
			_ = d409
			var d410 scm.JITValueDesc
			_ = d410
			var d411 scm.JITValueDesc
			_ = d411
			var d412 scm.JITValueDesc
			_ = d412
			var d413 scm.JITValueDesc
			_ = d413
			var d414 scm.JITValueDesc
			_ = d414
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
			var d420 scm.JITValueDesc
			_ = d420
			var d421 scm.JITValueDesc
			_ = d421
			var d422 scm.JITValueDesc
			_ = d422
			var d423 scm.JITValueDesc
			_ = d423
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
			r0 := ctx.W.EmitSubRSP32Fixup()
			_ = r0
			d0 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d2 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d3 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d4 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d5 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d6 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d7 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d8 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d9 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			var bbs [23]scm.BBDescriptor
			if result.Loc == scm.LocAny {
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			r1 := ctx.AllocReg()
			r2 := ctx.AllocRegExcept(r1)
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
			bbpos_0_9 := int32(-1)
			_ = bbpos_0_9
			lbl10 := ctx.W.ReserveLabel()
			bbpos_0_10 := int32(-1)
			_ = bbpos_0_10
			lbl11 := ctx.W.ReserveLabel()
			bbpos_0_11 := int32(-1)
			_ = bbpos_0_11
			lbl12 := ctx.W.ReserveLabel()
			bbpos_0_12 := int32(-1)
			_ = bbpos_0_12
			lbl13 := ctx.W.ReserveLabel()
			bbpos_0_13 := int32(-1)
			_ = bbpos_0_13
			lbl14 := ctx.W.ReserveLabel()
			bbpos_0_14 := int32(-1)
			_ = bbpos_0_14
			lbl15 := ctx.W.ReserveLabel()
			bbpos_0_15 := int32(-1)
			_ = bbpos_0_15
			lbl16 := ctx.W.ReserveLabel()
			bbpos_0_16 := int32(-1)
			_ = bbpos_0_16
			lbl17 := ctx.W.ReserveLabel()
			bbpos_0_17 := int32(-1)
			_ = bbpos_0_17
			lbl18 := ctx.W.ReserveLabel()
			bbpos_0_18 := int32(-1)
			_ = bbpos_0_18
			lbl19 := ctx.W.ReserveLabel()
			bbpos_0_19 := int32(-1)
			_ = bbpos_0_19
			lbl20 := ctx.W.ReserveLabel()
			bbpos_0_20 := int32(-1)
			_ = bbpos_0_20
			lbl21 := ctx.W.ReserveLabel()
			bbpos_0_21 := int32(-1)
			_ = bbpos_0_21
			lbl22 := ctx.W.ReserveLabel()
			bbpos_0_22 := int32(-1)
			_ = bbpos_0_22
			lbl23 := ctx.W.ReserveLabel()
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
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
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
			if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
				d9 = ps.OverlayValues[9]
			}
			ctx.ReclaimUntrackedRegs()
			r3 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).lastValue)
				ctx.W.EmitMovRegMem64(r3, fieldAddr)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
				ctx.W.EmitMovRegMem(r3, thisptr.Reg, off)
			}
			d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r3}
			ctx.BindReg(r3, &d10)
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d10)
			var d11 scm.JITValueDesc
			if d10.Loc == scm.LocImm {
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d10.Imm.Int()))))}
			} else {
				r4 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r4, d10.Reg)
				ctx.W.EmitShlRegImm8(r4, 32)
				ctx.W.EmitShrRegImm8(r4, 32)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r4}
				ctx.BindReg(r4, &d11)
			}
			ctx.FreeDesc(&d10)
			var d12 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).seqCount)
				r5 := ctx.AllocReg()
				ctx.W.EmitMovRegMem32(r5, fieldAddr)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r5}
				ctx.BindReg(r5, &d12)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).seqCount))
				r6 := ctx.AllocReg()
				ctx.W.EmitMovRegMemL(r6, thisptr.Reg, off)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r6}
				ctx.BindReg(r6, &d12)
			}
			ctx.EnsureDesc(&d12)
			var d13 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d12.Imm.Int()) == uint64(0))}
			} else {
				r7 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitCmpRegImm32(d12.Reg, 0)
				ctx.W.EmitSetcc(r7, scm.CcE)
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r7}
				ctx.BindReg(r7, &d13)
			}
			d14 = d13
			ctx.EnsureDesc(&d14)
			if d14.Loc != scm.LocImm && d14.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d14.Loc == scm.LocImm {
				if d14.Imm.Bool() {
			ps15 := scm.PhiState{General: ps.General}
			ps15.OverlayValues = make([]scm.JITValueDesc, 15)
			ps15.OverlayValues[0] = d0
			ps15.OverlayValues[1] = d1
			ps15.OverlayValues[2] = d2
			ps15.OverlayValues[3] = d3
			ps15.OverlayValues[4] = d4
			ps15.OverlayValues[5] = d5
			ps15.OverlayValues[6] = d6
			ps15.OverlayValues[7] = d7
			ps15.OverlayValues[8] = d8
			ps15.OverlayValues[9] = d9
			ps15.OverlayValues[10] = d10
			ps15.OverlayValues[11] = d11
			ps15.OverlayValues[12] = d12
			ps15.OverlayValues[13] = d13
			ps15.OverlayValues[14] = d14
					return bbs[1].RenderPS(ps15)
				}
			ps16 := scm.PhiState{General: ps.General}
			ps16.OverlayValues = make([]scm.JITValueDesc, 15)
			ps16.OverlayValues[0] = d0
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
				return bbs[2].RenderPS(ps16)
			}
			lbl24 := ctx.W.ReserveLabel()
			lbl25 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d14.Reg, 0)
			ctx.W.EmitJcc(scm.CcNE, lbl24)
			ctx.W.EmitJmp(lbl25)
			ctx.W.MarkLabel(lbl24)
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl25)
			ctx.W.EmitJmp(lbl3)
			ps17 := scm.PhiState{General: true}
			ps17.OverlayValues = make([]scm.JITValueDesc, 15)
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
			ps18 := scm.PhiState{General: true}
			ps18.OverlayValues = make([]scm.JITValueDesc, 15)
			ps18.OverlayValues[0] = d0
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
			alloc19 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps18)
			}
			ctx.RestoreAllocState(alloc19)
			if !bbs[1].Rendered {
				return bbs[1].RenderPS(ps17)
			}
			return result
			ctx.FreeDesc(&d13)
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
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
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
			ctx.ReclaimUntrackedRegs()
			d20 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d20)
			ctx.BindReg(r2, &d20)
			ctx.W.EmitMakeNil(d20)
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
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
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
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != scm.LocNone {
				d20 = ps.OverlayValues[20]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d12)
			var d21 scm.JITValueDesc
			if d11.Loc == scm.LocImm && d12.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d11.Imm.Int()) >= uint64(d12.Imm.Int()))}
			} else if d12.Loc == scm.LocImm {
				r8 := ctx.AllocRegExcept(d11.Reg)
				if d12.Imm.Int() >= -2147483648 && d12.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d11.Reg, int32(d12.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d12.Imm.Int()))
					ctx.W.EmitCmpInt64(d11.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r8, scm.CcAE)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r8}
				ctx.BindReg(r8, &d21)
			} else if d11.Loc == scm.LocImm {
				r9 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d11.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d12.Reg)
				ctx.W.EmitSetcc(r9, scm.CcAE)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r9}
				ctx.BindReg(r9, &d21)
			} else {
				r10 := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitCmpInt64(d11.Reg, d12.Reg)
				ctx.W.EmitSetcc(r10, scm.CcAE)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r10}
				ctx.BindReg(r10, &d21)
			}
			d22 = d21
			ctx.EnsureDesc(&d22)
			if d22.Loc != scm.LocImm && d22.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d22.Loc == scm.LocImm {
				if d22.Imm.Bool() {
			ps23 := scm.PhiState{General: ps.General}
			ps23.OverlayValues = make([]scm.JITValueDesc, 23)
			ps23.OverlayValues[0] = d0
			ps23.OverlayValues[1] = d1
			ps23.OverlayValues[2] = d2
			ps23.OverlayValues[3] = d3
			ps23.OverlayValues[4] = d4
			ps23.OverlayValues[5] = d5
			ps23.OverlayValues[6] = d6
			ps23.OverlayValues[7] = d7
			ps23.OverlayValues[8] = d8
			ps23.OverlayValues[9] = d9
			ps23.OverlayValues[10] = d10
			ps23.OverlayValues[11] = d11
			ps23.OverlayValues[12] = d12
			ps23.OverlayValues[13] = d13
			ps23.OverlayValues[14] = d14
			ps23.OverlayValues[20] = d20
			ps23.OverlayValues[21] = d21
			ps23.OverlayValues[22] = d22
					return bbs[3].RenderPS(ps23)
				}
			d24 = d11
			if d24.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d24)
			d25 = d24
			if d25.Loc == scm.LocImm {
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: d25.Type, Imm: scm.NewInt(int64(uint64(d25.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d25.Reg, 32)
				ctx.W.EmitShrRegImm8(d25.Reg, 32)
			}
			ctx.EmitStoreToStack(d25, 0)
			ps26 := scm.PhiState{General: ps.General}
			ps26.OverlayValues = make([]scm.JITValueDesc, 26)
			ps26.OverlayValues[0] = d0
			ps26.OverlayValues[1] = d1
			ps26.OverlayValues[2] = d2
			ps26.OverlayValues[3] = d3
			ps26.OverlayValues[4] = d4
			ps26.OverlayValues[5] = d5
			ps26.OverlayValues[6] = d6
			ps26.OverlayValues[7] = d7
			ps26.OverlayValues[8] = d8
			ps26.OverlayValues[9] = d9
			ps26.OverlayValues[10] = d10
			ps26.OverlayValues[11] = d11
			ps26.OverlayValues[12] = d12
			ps26.OverlayValues[13] = d13
			ps26.OverlayValues[14] = d14
			ps26.OverlayValues[20] = d20
			ps26.OverlayValues[21] = d21
			ps26.OverlayValues[22] = d22
			ps26.OverlayValues[24] = d24
			ps26.OverlayValues[25] = d25
			ps26.PhiValues = make([]scm.JITValueDesc, 1)
			d27 = d11
			ps26.PhiValues[0] = d27
				return bbs[4].RenderPS(ps26)
			}
			lbl26 := ctx.W.ReserveLabel()
			lbl27 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d22.Reg, 0)
			ctx.W.EmitJcc(scm.CcNE, lbl26)
			ctx.W.EmitJmp(lbl27)
			ctx.W.MarkLabel(lbl26)
			ctx.W.EmitJmp(lbl4)
			ctx.W.MarkLabel(lbl27)
			d28 = d11
			if d28.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d28)
			d29 = d28
			if d29.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: d29.Type, Imm: scm.NewInt(int64(uint64(d29.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d29.Reg, 32)
				ctx.W.EmitShrRegImm8(d29.Reg, 32)
			}
			ctx.EmitStoreToStack(d29, 0)
			ctx.W.EmitJmp(lbl5)
			ps30 := scm.PhiState{General: true}
			ps30.OverlayValues = make([]scm.JITValueDesc, 30)
			ps30.OverlayValues[0] = d0
			ps30.OverlayValues[1] = d1
			ps30.OverlayValues[2] = d2
			ps30.OverlayValues[3] = d3
			ps30.OverlayValues[4] = d4
			ps30.OverlayValues[5] = d5
			ps30.OverlayValues[6] = d6
			ps30.OverlayValues[7] = d7
			ps30.OverlayValues[8] = d8
			ps30.OverlayValues[9] = d9
			ps30.OverlayValues[10] = d10
			ps30.OverlayValues[11] = d11
			ps30.OverlayValues[12] = d12
			ps30.OverlayValues[13] = d13
			ps30.OverlayValues[14] = d14
			ps30.OverlayValues[20] = d20
			ps30.OverlayValues[21] = d21
			ps30.OverlayValues[22] = d22
			ps30.OverlayValues[24] = d24
			ps30.OverlayValues[25] = d25
			ps30.OverlayValues[27] = d27
			ps30.OverlayValues[28] = d28
			ps30.OverlayValues[29] = d29
			ps31 := scm.PhiState{General: true}
			ps31.OverlayValues = make([]scm.JITValueDesc, 30)
			ps31.OverlayValues[0] = d0
			ps31.OverlayValues[1] = d1
			ps31.OverlayValues[2] = d2
			ps31.OverlayValues[3] = d3
			ps31.OverlayValues[4] = d4
			ps31.OverlayValues[5] = d5
			ps31.OverlayValues[6] = d6
			ps31.OverlayValues[7] = d7
			ps31.OverlayValues[8] = d8
			ps31.OverlayValues[9] = d9
			ps31.OverlayValues[10] = d10
			ps31.OverlayValues[11] = d11
			ps31.OverlayValues[12] = d12
			ps31.OverlayValues[13] = d13
			ps31.OverlayValues[14] = d14
			ps31.OverlayValues[20] = d20
			ps31.OverlayValues[21] = d21
			ps31.OverlayValues[22] = d22
			ps31.OverlayValues[24] = d24
			ps31.OverlayValues[25] = d25
			ps31.OverlayValues[27] = d27
			ps31.OverlayValues[28] = d28
			ps31.OverlayValues[29] = d29
			ps31.PhiValues = make([]scm.JITValueDesc, 1)
			d32 = d11
			ps31.PhiValues[0] = d32
			alloc33 := ctx.SnapshotAllocState()
			if !bbs[4].Rendered {
				bbs[4].RenderPS(ps31)
			}
			ctx.RestoreAllocState(alloc33)
			if !bbs[3].Rendered {
				return bbs[3].RenderPS(ps30)
			}
			return result
			ctx.FreeDesc(&d21)
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
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d12)
			var d34 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(scratch, d12.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d34)
			}
			if d34.Loc == scm.LocImm {
				d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: d34.Type, Imm: scm.NewInt(int64(uint64(d34.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d34.Reg, 32)
				ctx.W.EmitShrRegImm8(d34.Reg, 32)
			}
			if d34.Loc == scm.LocReg && d12.Loc == scm.LocReg && d34.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			d35 = d34
			if d35.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d35)
			d36 = d35
			if d36.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: d36.Type, Imm: scm.NewInt(int64(uint64(d36.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d36.Reg, 32)
				ctx.W.EmitShrRegImm8(d36.Reg, 32)
			}
			ctx.EmitStoreToStack(d36, 0)
			ps37 := scm.PhiState{General: ps.General}
			ps37.OverlayValues = make([]scm.JITValueDesc, 37)
			ps37.OverlayValues[0] = d0
			ps37.OverlayValues[1] = d1
			ps37.OverlayValues[2] = d2
			ps37.OverlayValues[3] = d3
			ps37.OverlayValues[4] = d4
			ps37.OverlayValues[5] = d5
			ps37.OverlayValues[6] = d6
			ps37.OverlayValues[7] = d7
			ps37.OverlayValues[8] = d8
			ps37.OverlayValues[9] = d9
			ps37.OverlayValues[10] = d10
			ps37.OverlayValues[11] = d11
			ps37.OverlayValues[12] = d12
			ps37.OverlayValues[13] = d13
			ps37.OverlayValues[14] = d14
			ps37.OverlayValues[20] = d20
			ps37.OverlayValues[21] = d21
			ps37.OverlayValues[22] = d22
			ps37.OverlayValues[24] = d24
			ps37.OverlayValues[25] = d25
			ps37.OverlayValues[27] = d27
			ps37.OverlayValues[28] = d28
			ps37.OverlayValues[29] = d29
			ps37.OverlayValues[32] = d32
			ps37.OverlayValues[34] = d34
			ps37.OverlayValues[35] = d35
			ps37.OverlayValues[36] = d36
			ps37.PhiValues = make([]scm.JITValueDesc, 1)
			d38 = d34
			ps37.PhiValues[0] = d38
			if ps37.General && bbs[4].Rendered {
				ctx.W.EmitJmp(lbl5)
				return result
			}
			return bbs[4].RenderPS(ps37)
			return result
			}
			bbs[4].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[4].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
						d39 := ps.PhiValues[0]
						ctx.EnsureDesc(&d39)
						ctx.EmitStoreToStack(d39, 0)
					}
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
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
				d39 = ps.OverlayValues[39]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d0 = ps.PhiValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d12)
			var d40 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(scratch, d12.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d40)
			}
			if d40.Loc == scm.LocImm {
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: d40.Type, Imm: scm.NewInt(int64(uint64(d40.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d40.Reg, 32)
				ctx.W.EmitShrRegImm8(d40.Reg, 32)
			}
			if d40.Loc == scm.LocReg && d12.Loc == scm.LocReg && d40.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			d41 = d0
			if d41.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d41)
			d42 = d41
			if d42.Loc == scm.LocImm {
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: d42.Type, Imm: scm.NewInt(int64(uint64(d42.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d42.Reg, 32)
				ctx.W.EmitShrRegImm8(d42.Reg, 32)
			}
			ctx.EmitStoreToStack(d42, 8)
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 16)
			d43 = d40
			if d43.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d43)
			d44 = d43
			if d44.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: d44.Type, Imm: scm.NewInt(int64(uint64(d44.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d44.Reg, 32)
				ctx.W.EmitShrRegImm8(d44.Reg, 32)
			}
			ctx.EmitStoreToStack(d44, 24)
			ps45 := scm.PhiState{General: ps.General}
			ps45.OverlayValues = make([]scm.JITValueDesc, 45)
			ps45.OverlayValues[0] = d0
			ps45.OverlayValues[1] = d1
			ps45.OverlayValues[2] = d2
			ps45.OverlayValues[3] = d3
			ps45.OverlayValues[4] = d4
			ps45.OverlayValues[5] = d5
			ps45.OverlayValues[6] = d6
			ps45.OverlayValues[7] = d7
			ps45.OverlayValues[8] = d8
			ps45.OverlayValues[9] = d9
			ps45.OverlayValues[10] = d10
			ps45.OverlayValues[11] = d11
			ps45.OverlayValues[12] = d12
			ps45.OverlayValues[13] = d13
			ps45.OverlayValues[14] = d14
			ps45.OverlayValues[20] = d20
			ps45.OverlayValues[21] = d21
			ps45.OverlayValues[22] = d22
			ps45.OverlayValues[24] = d24
			ps45.OverlayValues[25] = d25
			ps45.OverlayValues[27] = d27
			ps45.OverlayValues[28] = d28
			ps45.OverlayValues[29] = d29
			ps45.OverlayValues[32] = d32
			ps45.OverlayValues[34] = d34
			ps45.OverlayValues[35] = d35
			ps45.OverlayValues[36] = d36
			ps45.OverlayValues[38] = d38
			ps45.OverlayValues[39] = d39
			ps45.OverlayValues[40] = d40
			ps45.OverlayValues[41] = d41
			ps45.OverlayValues[42] = d42
			ps45.OverlayValues[43] = d43
			ps45.OverlayValues[44] = d44
			ps45.PhiValues = make([]scm.JITValueDesc, 3)
			d46 = d0
			ps45.PhiValues[0] = d46
			d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
			ps45.PhiValues[1] = d47
			d48 = d40
			ps45.PhiValues[2] = d48
			if ps45.General && bbs[5].Rendered {
				ctx.W.EmitJmp(lbl6)
				return result
			}
			return bbs[5].RenderPS(ps45)
			return result
			}
			bbs[5].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[5].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
						d49 := ps.PhiValues[0]
						ctx.EnsureDesc(&d49)
						ctx.EmitStoreToStack(d49, 8)
					}
					if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
						d50 := ps.PhiValues[1]
						ctx.EnsureDesc(&d50)
						ctx.EmitStoreToStack(d50, 16)
					}
					if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
						d51 := ps.PhiValues[2]
						ctx.EnsureDesc(&d51)
						ctx.EmitStoreToStack(d51, 24)
					}
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
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
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
			d52 = d1
			_ = d52
			r11 := d1.Loc == scm.LocReg
			r12 := d1.Reg
			if r11 { ctx.ProtectReg(r12) }
			d53 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(80)}
			lbl28 := ctx.W.ReserveLabel()
			bbpos_1_0 := int32(-1)
			_ = bbpos_1_0
			bbpos_1_1 := int32(-1)
			_ = bbpos_1_1
			bbpos_1_2 := int32(-1)
			_ = bbpos_1_2
			bbpos_1_3 := int32(-1)
			_ = bbpos_1_3
			bbpos_1_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			d53 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(80)}
			ctx.EnsureDesc(&d52)
			ctx.EnsureDesc(&d52)
			var d54 scm.JITValueDesc
			if d52.Loc == scm.LocImm {
				d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d52.Imm.Int()))))}
			} else {
				r13 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r13, d52.Reg)
				ctx.W.EmitShlRegImm8(r13, 32)
				ctx.W.EmitShrRegImm8(r13, 32)
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
				ctx.BindReg(r13, &d54)
			}
			var d55 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r14 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r14, thisptr.Reg, off)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r14}
				ctx.BindReg(r14, &d55)
			}
			ctx.EnsureDesc(&d55)
			ctx.EnsureDesc(&d55)
			var d56 scm.JITValueDesc
			if d55.Loc == scm.LocImm {
				d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d55.Imm.Int()))))}
			} else {
				r15 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r15, d55.Reg)
				ctx.W.EmitShlRegImm8(r15, 56)
				ctx.W.EmitShrRegImm8(r15, 56)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
				ctx.BindReg(r15, &d56)
			}
			ctx.FreeDesc(&d55)
			ctx.EnsureDesc(&d54)
			ctx.EnsureDesc(&d56)
			ctx.EnsureDesc(&d54)
			ctx.EnsureDesc(&d56)
			ctx.EnsureDesc(&d54)
			ctx.EnsureDesc(&d56)
			var d57 scm.JITValueDesc
			if d54.Loc == scm.LocImm && d56.Loc == scm.LocImm {
				d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d54.Imm.Int() * d56.Imm.Int())}
			} else if d54.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d56.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d54.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d56.Reg)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d57)
			} else if d56.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d54.Reg)
				ctx.W.EmitMovRegReg(scratch, d54.Reg)
				if d56.Imm.Int() >= -2147483648 && d56.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d56.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d56.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d57)
			} else {
				r16 := ctx.AllocRegExcept(d54.Reg, d56.Reg)
				ctx.W.EmitMovRegReg(r16, d54.Reg)
				ctx.W.EmitImulInt64(r16, d56.Reg)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
				ctx.BindReg(r16, &d57)
			}
			if d57.Loc == scm.LocReg && d54.Loc == scm.LocReg && d57.Reg == d54.Reg {
				ctx.TransferReg(d54.Reg)
				d54.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d54)
			ctx.FreeDesc(&d56)
			var d58 scm.JITValueDesc
			r17 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r17, uint64(dataPtr))
				d58 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r17, StackOff: int32(sliceLen)}
				ctx.BindReg(r17, &d58)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				ctx.W.EmitMovRegMem(r17, thisptr.Reg, off)
				d58 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r17}
				ctx.BindReg(r17, &d58)
			}
			ctx.BindReg(r17, &d58)
			ctx.EnsureDesc(&d57)
			var d59 scm.JITValueDesc
			if d57.Loc == scm.LocImm {
				d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d57.Imm.Int() / 64)}
			} else {
				r18 := ctx.AllocRegExcept(d57.Reg)
				ctx.W.EmitMovRegReg(r18, d57.Reg)
				ctx.W.EmitShrRegImm8(r18, 6)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
				ctx.BindReg(r18, &d59)
			}
			if d59.Loc == scm.LocReg && d57.Loc == scm.LocReg && d59.Reg == d57.Reg {
				ctx.TransferReg(d57.Reg)
				d57.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d59)
			r19 := ctx.AllocReg()
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d58)
			if d59.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r19, uint64(d59.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r19, d59.Reg)
				ctx.W.EmitShlRegImm8(r19, 3)
			}
			if d58.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d58.Imm.Int()))
				ctx.W.EmitAddInt64(r19, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r19, d58.Reg)
			}
			r20 := ctx.AllocRegExcept(r19)
			ctx.W.EmitMovRegMem(r20, r19, 0)
			ctx.FreeReg(r19)
			d60 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r20}
			ctx.BindReg(r20, &d60)
			ctx.FreeDesc(&d59)
			ctx.EnsureDesc(&d57)
			var d61 scm.JITValueDesc
			if d57.Loc == scm.LocImm {
				d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d57.Imm.Int() % 64)}
			} else {
				r21 := ctx.AllocRegExcept(d57.Reg)
				ctx.W.EmitMovRegReg(r21, d57.Reg)
				ctx.W.EmitAndRegImm32(r21, 63)
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d61)
			}
			if d61.Loc == scm.LocReg && d57.Loc == scm.LocReg && d61.Reg == d57.Reg {
				ctx.TransferReg(d57.Reg)
				d57.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d60)
			ctx.EnsureDesc(&d61)
			var d62 scm.JITValueDesc
			if d60.Loc == scm.LocImm && d61.Loc == scm.LocImm {
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d60.Imm.Int()) << uint64(d61.Imm.Int())))}
			} else if d61.Loc == scm.LocImm {
				r22 := ctx.AllocRegExcept(d60.Reg)
				ctx.W.EmitMovRegReg(r22, d60.Reg)
				ctx.W.EmitShlRegImm8(r22, uint8(d61.Imm.Int()))
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d62)
			} else {
				{
					shiftSrc := d60.Reg
					r23 := ctx.AllocRegExcept(d60.Reg)
					ctx.W.EmitMovRegReg(r23, d60.Reg)
					shiftSrc = r23
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d61.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d61.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d61.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d62)
				}
			}
			if d62.Loc == scm.LocReg && d60.Loc == scm.LocReg && d62.Reg == d60.Reg {
				ctx.TransferReg(d60.Reg)
				d60.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d60)
			ctx.FreeDesc(&d61)
			var d63 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
				r24 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r24, thisptr.Reg, off)
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r24}
				ctx.BindReg(r24, &d63)
			}
			d64 = d63
			ctx.EnsureDesc(&d64)
			if d64.Loc != scm.LocImm && d64.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl29 := ctx.W.ReserveLabel()
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			lbl32 := ctx.W.ReserveLabel()
			if d64.Loc == scm.LocImm {
				if d64.Imm.Bool() {
					ctx.W.MarkLabel(lbl31)
					ctx.W.EmitJmp(lbl29)
				} else {
					ctx.W.MarkLabel(lbl32)
			d65 = d62
			if d65.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d65)
			ctx.EmitStoreToStack(d65, 80)
					ctx.W.EmitJmp(lbl30)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d64.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl31)
				ctx.W.EmitJmp(lbl32)
				ctx.W.MarkLabel(lbl31)
				ctx.W.EmitJmp(lbl29)
				ctx.W.MarkLabel(lbl32)
			d66 = d62
			if d66.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d66)
			ctx.EmitStoreToStack(d66, 80)
				ctx.W.EmitJmp(lbl30)
			}
			ctx.FreeDesc(&d63)
			bbpos_1_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl30)
			ctx.W.ResolveFixups()
			d53 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(80)}
			var d67 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r25 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r25, thisptr.Reg, off)
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r25}
				ctx.BindReg(r25, &d67)
			}
			ctx.EnsureDesc(&d67)
			ctx.EnsureDesc(&d67)
			var d68 scm.JITValueDesc
			if d67.Loc == scm.LocImm {
				d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d67.Imm.Int()))))}
			} else {
				r26 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r26, d67.Reg)
				ctx.W.EmitShlRegImm8(r26, 56)
				ctx.W.EmitShrRegImm8(r26, 56)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
				ctx.BindReg(r26, &d68)
			}
			ctx.FreeDesc(&d67)
			d69 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d68)
			ctx.EnsureDesc(&d69)
			ctx.EnsureDesc(&d68)
			ctx.EnsureDesc(&d69)
			ctx.EnsureDesc(&d68)
			var d70 scm.JITValueDesc
			if d69.Loc == scm.LocImm && d68.Loc == scm.LocImm {
				d70 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d69.Imm.Int() - d68.Imm.Int())}
			} else if d68.Loc == scm.LocImm && d68.Imm.Int() == 0 {
				r27 := ctx.AllocRegExcept(d69.Reg)
				ctx.W.EmitMovRegReg(r27, d69.Reg)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d70)
			} else if d69.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d68.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d69.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d68.Reg)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d70)
			} else if d68.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d69.Reg)
				ctx.W.EmitMovRegReg(scratch, d69.Reg)
				if d68.Imm.Int() >= -2147483648 && d68.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d68.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d68.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d70)
			} else {
				r28 := ctx.AllocRegExcept(d69.Reg, d68.Reg)
				ctx.W.EmitMovRegReg(r28, d69.Reg)
				ctx.W.EmitSubInt64(r28, d68.Reg)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d70)
			}
			if d70.Loc == scm.LocReg && d69.Loc == scm.LocReg && d70.Reg == d69.Reg {
				ctx.TransferReg(d69.Reg)
				d69.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d68)
			ctx.EnsureDesc(&d53)
			ctx.EnsureDesc(&d70)
			var d71 scm.JITValueDesc
			if d53.Loc == scm.LocImm && d70.Loc == scm.LocImm {
				d71 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d53.Imm.Int()) >> uint64(d70.Imm.Int())))}
			} else if d70.Loc == scm.LocImm {
				r29 := ctx.AllocRegExcept(d53.Reg)
				ctx.W.EmitMovRegReg(r29, d53.Reg)
				ctx.W.EmitShrRegImm8(r29, uint8(d70.Imm.Int()))
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d71)
			} else {
				{
					shiftSrc := d53.Reg
					r30 := ctx.AllocRegExcept(d53.Reg)
					ctx.W.EmitMovRegReg(r30, d53.Reg)
					shiftSrc = r30
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d70.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d70.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d70.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d71)
				}
			}
			if d71.Loc == scm.LocReg && d53.Loc == scm.LocReg && d71.Reg == d53.Reg {
				ctx.TransferReg(d53.Reg)
				d53.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d53)
			ctx.FreeDesc(&d70)
			r31 := ctx.AllocReg()
			ctx.EnsureDesc(&d71)
			ctx.EnsureDesc(&d71)
			if d71.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r31, d71)
			}
			ctx.W.EmitJmp(lbl28)
			bbpos_1_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl29)
			ctx.W.ResolveFixups()
			d53 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(80)}
			ctx.EnsureDesc(&d57)
			var d72 scm.JITValueDesc
			if d57.Loc == scm.LocImm {
				d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d57.Imm.Int() % 64)}
			} else {
				r32 := ctx.AllocRegExcept(d57.Reg)
				ctx.W.EmitMovRegReg(r32, d57.Reg)
				ctx.W.EmitAndRegImm32(r32, 63)
				d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
				ctx.BindReg(r32, &d72)
			}
			if d72.Loc == scm.LocReg && d57.Loc == scm.LocReg && d72.Reg == d57.Reg {
				ctx.TransferReg(d57.Reg)
				d57.Loc = scm.LocNone
			}
			var d73 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d73 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r33 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r33, thisptr.Reg, off)
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r33}
				ctx.BindReg(r33, &d73)
			}
			ctx.EnsureDesc(&d73)
			ctx.EnsureDesc(&d73)
			var d74 scm.JITValueDesc
			if d73.Loc == scm.LocImm {
				d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d73.Imm.Int()))))}
			} else {
				r34 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r34, d73.Reg)
				ctx.W.EmitShlRegImm8(r34, 56)
				ctx.W.EmitShrRegImm8(r34, 56)
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
				ctx.BindReg(r34, &d74)
			}
			ctx.FreeDesc(&d73)
			ctx.EnsureDesc(&d72)
			ctx.EnsureDesc(&d74)
			ctx.EnsureDesc(&d72)
			ctx.EnsureDesc(&d74)
			ctx.EnsureDesc(&d72)
			ctx.EnsureDesc(&d74)
			var d75 scm.JITValueDesc
			if d72.Loc == scm.LocImm && d74.Loc == scm.LocImm {
				d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d72.Imm.Int() + d74.Imm.Int())}
			} else if d74.Loc == scm.LocImm && d74.Imm.Int() == 0 {
				r35 := ctx.AllocRegExcept(d72.Reg)
				ctx.W.EmitMovRegReg(r35, d72.Reg)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d75)
			} else if d72.Loc == scm.LocImm && d72.Imm.Int() == 0 {
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d74.Reg}
				ctx.BindReg(d74.Reg, &d75)
			} else if d72.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d72.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d74.Reg)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d75)
			} else if d74.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d72.Reg)
				ctx.W.EmitMovRegReg(scratch, d72.Reg)
				if d74.Imm.Int() >= -2147483648 && d74.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d74.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d74.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d75)
			} else {
				r36 := ctx.AllocRegExcept(d72.Reg, d74.Reg)
				ctx.W.EmitMovRegReg(r36, d72.Reg)
				ctx.W.EmitAddInt64(r36, d74.Reg)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d75)
			}
			if d75.Loc == scm.LocReg && d72.Loc == scm.LocReg && d75.Reg == d72.Reg {
				ctx.TransferReg(d72.Reg)
				d72.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d72)
			ctx.FreeDesc(&d74)
			ctx.EnsureDesc(&d75)
			var d76 scm.JITValueDesc
			if d75.Loc == scm.LocImm {
				d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d75.Imm.Int()) > uint64(64))}
			} else {
				r37 := ctx.AllocRegExcept(d75.Reg)
				ctx.W.EmitCmpRegImm32(d75.Reg, 64)
				ctx.W.EmitSetcc(r37, scm.CcA)
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r37}
				ctx.BindReg(r37, &d76)
			}
			ctx.FreeDesc(&d75)
			d77 = d76
			ctx.EnsureDesc(&d77)
			if d77.Loc != scm.LocImm && d77.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl33 := ctx.W.ReserveLabel()
			lbl34 := ctx.W.ReserveLabel()
			lbl35 := ctx.W.ReserveLabel()
			if d77.Loc == scm.LocImm {
				if d77.Imm.Bool() {
					ctx.W.MarkLabel(lbl34)
					ctx.W.EmitJmp(lbl33)
				} else {
					ctx.W.MarkLabel(lbl35)
			d78 = d62
			if d78.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d78)
			ctx.EmitStoreToStack(d78, 80)
					ctx.W.EmitJmp(lbl30)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d77.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl34)
				ctx.W.EmitJmp(lbl35)
				ctx.W.MarkLabel(lbl34)
				ctx.W.EmitJmp(lbl33)
				ctx.W.MarkLabel(lbl35)
			d79 = d62
			if d79.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d79)
			ctx.EmitStoreToStack(d79, 80)
				ctx.W.EmitJmp(lbl30)
			}
			ctx.FreeDesc(&d76)
			bbpos_1_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl33)
			ctx.W.ResolveFixups()
			d53 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(80)}
			ctx.EnsureDesc(&d57)
			var d80 scm.JITValueDesc
			if d57.Loc == scm.LocImm {
				d80 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d57.Imm.Int() / 64)}
			} else {
				r38 := ctx.AllocRegExcept(d57.Reg)
				ctx.W.EmitMovRegReg(r38, d57.Reg)
				ctx.W.EmitShrRegImm8(r38, 6)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
				ctx.BindReg(r38, &d80)
			}
			if d80.Loc == scm.LocReg && d57.Loc == scm.LocReg && d80.Reg == d57.Reg {
				ctx.TransferReg(d57.Reg)
				d57.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d80)
			ctx.EnsureDesc(&d80)
			var d81 scm.JITValueDesc
			if d80.Loc == scm.LocImm {
				d81 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d80.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d80.Reg)
				ctx.W.EmitMovRegReg(scratch, d80.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d81)
			}
			if d81.Loc == scm.LocReg && d80.Loc == scm.LocReg && d81.Reg == d80.Reg {
				ctx.TransferReg(d80.Reg)
				d80.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d80)
			ctx.EnsureDesc(&d81)
			r39 := ctx.AllocReg()
			ctx.EnsureDesc(&d81)
			ctx.EnsureDesc(&d58)
			if d81.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r39, uint64(d81.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r39, d81.Reg)
				ctx.W.EmitShlRegImm8(r39, 3)
			}
			if d58.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d58.Imm.Int()))
				ctx.W.EmitAddInt64(r39, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r39, d58.Reg)
			}
			r40 := ctx.AllocRegExcept(r39)
			ctx.W.EmitMovRegMem(r40, r39, 0)
			ctx.FreeReg(r39)
			d82 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r40}
			ctx.BindReg(r40, &d82)
			ctx.FreeDesc(&d81)
			ctx.EnsureDesc(&d57)
			var d83 scm.JITValueDesc
			if d57.Loc == scm.LocImm {
				d83 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d57.Imm.Int() % 64)}
			} else {
				r41 := ctx.AllocRegExcept(d57.Reg)
				ctx.W.EmitMovRegReg(r41, d57.Reg)
				ctx.W.EmitAndRegImm32(r41, 63)
				d83 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d83)
			}
			if d83.Loc == scm.LocReg && d57.Loc == scm.LocReg && d83.Reg == d57.Reg {
				ctx.TransferReg(d57.Reg)
				d57.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d57)
			d84 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d83)
			ctx.EnsureDesc(&d84)
			ctx.EnsureDesc(&d83)
			ctx.EnsureDesc(&d84)
			ctx.EnsureDesc(&d83)
			var d85 scm.JITValueDesc
			if d84.Loc == scm.LocImm && d83.Loc == scm.LocImm {
				d85 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d84.Imm.Int() - d83.Imm.Int())}
			} else if d83.Loc == scm.LocImm && d83.Imm.Int() == 0 {
				r42 := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitMovRegReg(r42, d84.Reg)
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d85)
			} else if d84.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d83.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d84.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d83.Reg)
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d85)
			} else if d83.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitMovRegReg(scratch, d84.Reg)
				if d83.Imm.Int() >= -2147483648 && d83.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d83.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d83.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d85)
			} else {
				r43 := ctx.AllocRegExcept(d84.Reg, d83.Reg)
				ctx.W.EmitMovRegReg(r43, d84.Reg)
				ctx.W.EmitSubInt64(r43, d83.Reg)
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
				ctx.BindReg(r43, &d85)
			}
			if d85.Loc == scm.LocReg && d84.Loc == scm.LocReg && d85.Reg == d84.Reg {
				ctx.TransferReg(d84.Reg)
				d84.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d83)
			ctx.EnsureDesc(&d82)
			ctx.EnsureDesc(&d85)
			var d86 scm.JITValueDesc
			if d82.Loc == scm.LocImm && d85.Loc == scm.LocImm {
				d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d82.Imm.Int()) >> uint64(d85.Imm.Int())))}
			} else if d85.Loc == scm.LocImm {
				r44 := ctx.AllocRegExcept(d82.Reg)
				ctx.W.EmitMovRegReg(r44, d82.Reg)
				ctx.W.EmitShrRegImm8(r44, uint8(d85.Imm.Int()))
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d86)
			} else {
				{
					shiftSrc := d82.Reg
					r45 := ctx.AllocRegExcept(d82.Reg)
					ctx.W.EmitMovRegReg(r45, d82.Reg)
					shiftSrc = r45
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d85.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d85.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d85.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d86)
				}
			}
			if d86.Loc == scm.LocReg && d82.Loc == scm.LocReg && d86.Reg == d82.Reg {
				ctx.TransferReg(d82.Reg)
				d82.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d82)
			ctx.FreeDesc(&d85)
			ctx.EnsureDesc(&d62)
			ctx.EnsureDesc(&d86)
			var d87 scm.JITValueDesc
			if d62.Loc == scm.LocImm && d86.Loc == scm.LocImm {
				d87 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d62.Imm.Int() | d86.Imm.Int())}
			} else if d62.Loc == scm.LocImm && d62.Imm.Int() == 0 {
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d86.Reg}
				ctx.BindReg(d86.Reg, &d87)
			} else if d86.Loc == scm.LocImm && d86.Imm.Int() == 0 {
				r46 := ctx.AllocRegExcept(d62.Reg)
				ctx.W.EmitMovRegReg(r46, d62.Reg)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d87)
			} else if d62.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d86.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d62.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d86.Reg)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d87)
			} else if d86.Loc == scm.LocImm {
				r47 := ctx.AllocRegExcept(d62.Reg)
				ctx.W.EmitMovRegReg(r47, d62.Reg)
				if d86.Imm.Int() >= -2147483648 && d86.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r47, int32(d86.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d86.Imm.Int()))
					ctx.W.EmitOrInt64(r47, scm.RegR11)
				}
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
				ctx.BindReg(r47, &d87)
			} else {
				r48 := ctx.AllocRegExcept(d62.Reg, d86.Reg)
				ctx.W.EmitMovRegReg(r48, d62.Reg)
				ctx.W.EmitOrInt64(r48, d86.Reg)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
				ctx.BindReg(r48, &d87)
			}
			if d87.Loc == scm.LocReg && d62.Loc == scm.LocReg && d87.Reg == d62.Reg {
				ctx.TransferReg(d62.Reg)
				d62.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d86)
			d88 = d87
			if d88.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d88)
			ctx.EmitStoreToStack(d88, 80)
			ctx.W.EmitJmp(lbl30)
			ctx.W.MarkLabel(lbl28)
			d89 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r31}
			ctx.BindReg(r31, &d89)
			ctx.BindReg(r31, &d89)
			if r11 { ctx.UnprotectReg(r12) }
			ctx.EnsureDesc(&d89)
			ctx.EnsureDesc(&d89)
			var d90 scm.JITValueDesc
			if d89.Loc == scm.LocImm {
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d89.Imm.Int()))))}
			} else {
				r49 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r49, d89.Reg)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
				ctx.BindReg(r49, &d90)
			}
			ctx.FreeDesc(&d89)
			var d91 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d91 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r50 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r50, thisptr.Reg, off)
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r50}
				ctx.BindReg(r50, &d91)
			}
			ctx.EnsureDesc(&d90)
			ctx.EnsureDesc(&d91)
			ctx.EnsureDesc(&d90)
			ctx.EnsureDesc(&d91)
			ctx.EnsureDesc(&d90)
			ctx.EnsureDesc(&d91)
			var d92 scm.JITValueDesc
			if d90.Loc == scm.LocImm && d91.Loc == scm.LocImm {
				d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d90.Imm.Int() + d91.Imm.Int())}
			} else if d91.Loc == scm.LocImm && d91.Imm.Int() == 0 {
				r51 := ctx.AllocRegExcept(d90.Reg)
				ctx.W.EmitMovRegReg(r51, d90.Reg)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
				ctx.BindReg(r51, &d92)
			} else if d90.Loc == scm.LocImm && d90.Imm.Int() == 0 {
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d91.Reg}
				ctx.BindReg(d91.Reg, &d92)
			} else if d90.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d91.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d90.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d91.Reg)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d92)
			} else if d91.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d90.Reg)
				ctx.W.EmitMovRegReg(scratch, d90.Reg)
				if d91.Imm.Int() >= -2147483648 && d91.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d91.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d91.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d92)
			} else {
				r52 := ctx.AllocRegExcept(d90.Reg, d91.Reg)
				ctx.W.EmitMovRegReg(r52, d90.Reg)
				ctx.W.EmitAddInt64(r52, d91.Reg)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
				ctx.BindReg(r52, &d92)
			}
			if d92.Loc == scm.LocReg && d90.Loc == scm.LocReg && d92.Reg == d90.Reg {
				ctx.TransferReg(d90.Reg)
				d90.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d90)
			ctx.FreeDesc(&d91)
			ctx.EnsureDesc(&d92)
			ctx.EnsureDesc(&d92)
			var d93 scm.JITValueDesc
			if d92.Loc == scm.LocImm {
				d93 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d92.Imm.Int()))))}
			} else {
				r53 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r53, d92.Reg)
				ctx.W.EmitShlRegImm8(r53, 32)
				ctx.W.EmitShrRegImm8(r53, 32)
				d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
				ctx.BindReg(r53, &d93)
			}
			ctx.FreeDesc(&d92)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d93)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d93)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d93)
			var d94 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d93.Loc == scm.LocImm {
				d94 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d93.Imm.Int()))}
			} else if d93.Loc == scm.LocImm {
				r54 := ctx.AllocRegExcept(idxInt.Reg)
				if d93.Imm.Int() >= -2147483648 && d93.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(idxInt.Reg, int32(d93.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d93.Imm.Int()))
					ctx.W.EmitCmpInt64(idxInt.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r54, scm.CcB)
				d94 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r54}
				ctx.BindReg(r54, &d94)
			} else if idxInt.Loc == scm.LocImm {
				r55 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d93.Reg)
				ctx.W.EmitSetcc(r55, scm.CcB)
				d94 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r55}
				ctx.BindReg(r55, &d94)
			} else {
				r56 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitCmpInt64(idxInt.Reg, d93.Reg)
				ctx.W.EmitSetcc(r56, scm.CcB)
				d94 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r56}
				ctx.BindReg(r56, &d94)
			}
			ctx.FreeDesc(&d93)
			d95 = d94
			ctx.EnsureDesc(&d95)
			if d95.Loc != scm.LocImm && d95.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d95.Loc == scm.LocImm {
				if d95.Imm.Bool() {
			ps96 := scm.PhiState{General: ps.General}
			ps96.OverlayValues = make([]scm.JITValueDesc, 96)
			ps96.OverlayValues[0] = d0
			ps96.OverlayValues[1] = d1
			ps96.OverlayValues[2] = d2
			ps96.OverlayValues[3] = d3
			ps96.OverlayValues[4] = d4
			ps96.OverlayValues[5] = d5
			ps96.OverlayValues[6] = d6
			ps96.OverlayValues[7] = d7
			ps96.OverlayValues[8] = d8
			ps96.OverlayValues[9] = d9
			ps96.OverlayValues[10] = d10
			ps96.OverlayValues[11] = d11
			ps96.OverlayValues[12] = d12
			ps96.OverlayValues[13] = d13
			ps96.OverlayValues[14] = d14
			ps96.OverlayValues[20] = d20
			ps96.OverlayValues[21] = d21
			ps96.OverlayValues[22] = d22
			ps96.OverlayValues[24] = d24
			ps96.OverlayValues[25] = d25
			ps96.OverlayValues[27] = d27
			ps96.OverlayValues[28] = d28
			ps96.OverlayValues[29] = d29
			ps96.OverlayValues[32] = d32
			ps96.OverlayValues[34] = d34
			ps96.OverlayValues[35] = d35
			ps96.OverlayValues[36] = d36
			ps96.OverlayValues[38] = d38
			ps96.OverlayValues[39] = d39
			ps96.OverlayValues[40] = d40
			ps96.OverlayValues[41] = d41
			ps96.OverlayValues[42] = d42
			ps96.OverlayValues[43] = d43
			ps96.OverlayValues[44] = d44
			ps96.OverlayValues[46] = d46
			ps96.OverlayValues[47] = d47
			ps96.OverlayValues[48] = d48
			ps96.OverlayValues[49] = d49
			ps96.OverlayValues[50] = d50
			ps96.OverlayValues[51] = d51
			ps96.OverlayValues[52] = d52
			ps96.OverlayValues[53] = d53
			ps96.OverlayValues[54] = d54
			ps96.OverlayValues[55] = d55
			ps96.OverlayValues[56] = d56
			ps96.OverlayValues[57] = d57
			ps96.OverlayValues[58] = d58
			ps96.OverlayValues[59] = d59
			ps96.OverlayValues[60] = d60
			ps96.OverlayValues[61] = d61
			ps96.OverlayValues[62] = d62
			ps96.OverlayValues[63] = d63
			ps96.OverlayValues[64] = d64
			ps96.OverlayValues[65] = d65
			ps96.OverlayValues[66] = d66
			ps96.OverlayValues[67] = d67
			ps96.OverlayValues[68] = d68
			ps96.OverlayValues[69] = d69
			ps96.OverlayValues[70] = d70
			ps96.OverlayValues[71] = d71
			ps96.OverlayValues[72] = d72
			ps96.OverlayValues[73] = d73
			ps96.OverlayValues[74] = d74
			ps96.OverlayValues[75] = d75
			ps96.OverlayValues[76] = d76
			ps96.OverlayValues[77] = d77
			ps96.OverlayValues[78] = d78
			ps96.OverlayValues[79] = d79
			ps96.OverlayValues[80] = d80
			ps96.OverlayValues[81] = d81
			ps96.OverlayValues[82] = d82
			ps96.OverlayValues[83] = d83
			ps96.OverlayValues[84] = d84
			ps96.OverlayValues[85] = d85
			ps96.OverlayValues[86] = d86
			ps96.OverlayValues[87] = d87
			ps96.OverlayValues[88] = d88
			ps96.OverlayValues[89] = d89
			ps96.OverlayValues[90] = d90
			ps96.OverlayValues[91] = d91
			ps96.OverlayValues[92] = d92
			ps96.OverlayValues[93] = d93
			ps96.OverlayValues[94] = d94
			ps96.OverlayValues[95] = d95
					return bbs[7].RenderPS(ps96)
				}
			ps97 := scm.PhiState{General: ps.General}
			ps97.OverlayValues = make([]scm.JITValueDesc, 96)
			ps97.OverlayValues[0] = d0
			ps97.OverlayValues[1] = d1
			ps97.OverlayValues[2] = d2
			ps97.OverlayValues[3] = d3
			ps97.OverlayValues[4] = d4
			ps97.OverlayValues[5] = d5
			ps97.OverlayValues[6] = d6
			ps97.OverlayValues[7] = d7
			ps97.OverlayValues[8] = d8
			ps97.OverlayValues[9] = d9
			ps97.OverlayValues[10] = d10
			ps97.OverlayValues[11] = d11
			ps97.OverlayValues[12] = d12
			ps97.OverlayValues[13] = d13
			ps97.OverlayValues[14] = d14
			ps97.OverlayValues[20] = d20
			ps97.OverlayValues[21] = d21
			ps97.OverlayValues[22] = d22
			ps97.OverlayValues[24] = d24
			ps97.OverlayValues[25] = d25
			ps97.OverlayValues[27] = d27
			ps97.OverlayValues[28] = d28
			ps97.OverlayValues[29] = d29
			ps97.OverlayValues[32] = d32
			ps97.OverlayValues[34] = d34
			ps97.OverlayValues[35] = d35
			ps97.OverlayValues[36] = d36
			ps97.OverlayValues[38] = d38
			ps97.OverlayValues[39] = d39
			ps97.OverlayValues[40] = d40
			ps97.OverlayValues[41] = d41
			ps97.OverlayValues[42] = d42
			ps97.OverlayValues[43] = d43
			ps97.OverlayValues[44] = d44
			ps97.OverlayValues[46] = d46
			ps97.OverlayValues[47] = d47
			ps97.OverlayValues[48] = d48
			ps97.OverlayValues[49] = d49
			ps97.OverlayValues[50] = d50
			ps97.OverlayValues[51] = d51
			ps97.OverlayValues[52] = d52
			ps97.OverlayValues[53] = d53
			ps97.OverlayValues[54] = d54
			ps97.OverlayValues[55] = d55
			ps97.OverlayValues[56] = d56
			ps97.OverlayValues[57] = d57
			ps97.OverlayValues[58] = d58
			ps97.OverlayValues[59] = d59
			ps97.OverlayValues[60] = d60
			ps97.OverlayValues[61] = d61
			ps97.OverlayValues[62] = d62
			ps97.OverlayValues[63] = d63
			ps97.OverlayValues[64] = d64
			ps97.OverlayValues[65] = d65
			ps97.OverlayValues[66] = d66
			ps97.OverlayValues[67] = d67
			ps97.OverlayValues[68] = d68
			ps97.OverlayValues[69] = d69
			ps97.OverlayValues[70] = d70
			ps97.OverlayValues[71] = d71
			ps97.OverlayValues[72] = d72
			ps97.OverlayValues[73] = d73
			ps97.OverlayValues[74] = d74
			ps97.OverlayValues[75] = d75
			ps97.OverlayValues[76] = d76
			ps97.OverlayValues[77] = d77
			ps97.OverlayValues[78] = d78
			ps97.OverlayValues[79] = d79
			ps97.OverlayValues[80] = d80
			ps97.OverlayValues[81] = d81
			ps97.OverlayValues[82] = d82
			ps97.OverlayValues[83] = d83
			ps97.OverlayValues[84] = d84
			ps97.OverlayValues[85] = d85
			ps97.OverlayValues[86] = d86
			ps97.OverlayValues[87] = d87
			ps97.OverlayValues[88] = d88
			ps97.OverlayValues[89] = d89
			ps97.OverlayValues[90] = d90
			ps97.OverlayValues[91] = d91
			ps97.OverlayValues[92] = d92
			ps97.OverlayValues[93] = d93
			ps97.OverlayValues[94] = d94
			ps97.OverlayValues[95] = d95
				return bbs[9].RenderPS(ps97)
			}
			lbl36 := ctx.W.ReserveLabel()
			lbl37 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d95.Reg, 0)
			ctx.W.EmitJcc(scm.CcNE, lbl36)
			ctx.W.EmitJmp(lbl37)
			ctx.W.MarkLabel(lbl36)
			ctx.W.EmitJmp(lbl8)
			ctx.W.MarkLabel(lbl37)
			ctx.W.EmitJmp(lbl10)
			ps98 := scm.PhiState{General: true}
			ps98.OverlayValues = make([]scm.JITValueDesc, 96)
			ps98.OverlayValues[0] = d0
			ps98.OverlayValues[1] = d1
			ps98.OverlayValues[2] = d2
			ps98.OverlayValues[3] = d3
			ps98.OverlayValues[4] = d4
			ps98.OverlayValues[5] = d5
			ps98.OverlayValues[6] = d6
			ps98.OverlayValues[7] = d7
			ps98.OverlayValues[8] = d8
			ps98.OverlayValues[9] = d9
			ps98.OverlayValues[10] = d10
			ps98.OverlayValues[11] = d11
			ps98.OverlayValues[12] = d12
			ps98.OverlayValues[13] = d13
			ps98.OverlayValues[14] = d14
			ps98.OverlayValues[20] = d20
			ps98.OverlayValues[21] = d21
			ps98.OverlayValues[22] = d22
			ps98.OverlayValues[24] = d24
			ps98.OverlayValues[25] = d25
			ps98.OverlayValues[27] = d27
			ps98.OverlayValues[28] = d28
			ps98.OverlayValues[29] = d29
			ps98.OverlayValues[32] = d32
			ps98.OverlayValues[34] = d34
			ps98.OverlayValues[35] = d35
			ps98.OverlayValues[36] = d36
			ps98.OverlayValues[38] = d38
			ps98.OverlayValues[39] = d39
			ps98.OverlayValues[40] = d40
			ps98.OverlayValues[41] = d41
			ps98.OverlayValues[42] = d42
			ps98.OverlayValues[43] = d43
			ps98.OverlayValues[44] = d44
			ps98.OverlayValues[46] = d46
			ps98.OverlayValues[47] = d47
			ps98.OverlayValues[48] = d48
			ps98.OverlayValues[49] = d49
			ps98.OverlayValues[50] = d50
			ps98.OverlayValues[51] = d51
			ps98.OverlayValues[52] = d52
			ps98.OverlayValues[53] = d53
			ps98.OverlayValues[54] = d54
			ps98.OverlayValues[55] = d55
			ps98.OverlayValues[56] = d56
			ps98.OverlayValues[57] = d57
			ps98.OverlayValues[58] = d58
			ps98.OverlayValues[59] = d59
			ps98.OverlayValues[60] = d60
			ps98.OverlayValues[61] = d61
			ps98.OverlayValues[62] = d62
			ps98.OverlayValues[63] = d63
			ps98.OverlayValues[64] = d64
			ps98.OverlayValues[65] = d65
			ps98.OverlayValues[66] = d66
			ps98.OverlayValues[67] = d67
			ps98.OverlayValues[68] = d68
			ps98.OverlayValues[69] = d69
			ps98.OverlayValues[70] = d70
			ps98.OverlayValues[71] = d71
			ps98.OverlayValues[72] = d72
			ps98.OverlayValues[73] = d73
			ps98.OverlayValues[74] = d74
			ps98.OverlayValues[75] = d75
			ps98.OverlayValues[76] = d76
			ps98.OverlayValues[77] = d77
			ps98.OverlayValues[78] = d78
			ps98.OverlayValues[79] = d79
			ps98.OverlayValues[80] = d80
			ps98.OverlayValues[81] = d81
			ps98.OverlayValues[82] = d82
			ps98.OverlayValues[83] = d83
			ps98.OverlayValues[84] = d84
			ps98.OverlayValues[85] = d85
			ps98.OverlayValues[86] = d86
			ps98.OverlayValues[87] = d87
			ps98.OverlayValues[88] = d88
			ps98.OverlayValues[89] = d89
			ps98.OverlayValues[90] = d90
			ps98.OverlayValues[91] = d91
			ps98.OverlayValues[92] = d92
			ps98.OverlayValues[93] = d93
			ps98.OverlayValues[94] = d94
			ps98.OverlayValues[95] = d95
			ps99 := scm.PhiState{General: true}
			ps99.OverlayValues = make([]scm.JITValueDesc, 96)
			ps99.OverlayValues[0] = d0
			ps99.OverlayValues[1] = d1
			ps99.OverlayValues[2] = d2
			ps99.OverlayValues[3] = d3
			ps99.OverlayValues[4] = d4
			ps99.OverlayValues[5] = d5
			ps99.OverlayValues[6] = d6
			ps99.OverlayValues[7] = d7
			ps99.OverlayValues[8] = d8
			ps99.OverlayValues[9] = d9
			ps99.OverlayValues[10] = d10
			ps99.OverlayValues[11] = d11
			ps99.OverlayValues[12] = d12
			ps99.OverlayValues[13] = d13
			ps99.OverlayValues[14] = d14
			ps99.OverlayValues[20] = d20
			ps99.OverlayValues[21] = d21
			ps99.OverlayValues[22] = d22
			ps99.OverlayValues[24] = d24
			ps99.OverlayValues[25] = d25
			ps99.OverlayValues[27] = d27
			ps99.OverlayValues[28] = d28
			ps99.OverlayValues[29] = d29
			ps99.OverlayValues[32] = d32
			ps99.OverlayValues[34] = d34
			ps99.OverlayValues[35] = d35
			ps99.OverlayValues[36] = d36
			ps99.OverlayValues[38] = d38
			ps99.OverlayValues[39] = d39
			ps99.OverlayValues[40] = d40
			ps99.OverlayValues[41] = d41
			ps99.OverlayValues[42] = d42
			ps99.OverlayValues[43] = d43
			ps99.OverlayValues[44] = d44
			ps99.OverlayValues[46] = d46
			ps99.OverlayValues[47] = d47
			ps99.OverlayValues[48] = d48
			ps99.OverlayValues[49] = d49
			ps99.OverlayValues[50] = d50
			ps99.OverlayValues[51] = d51
			ps99.OverlayValues[52] = d52
			ps99.OverlayValues[53] = d53
			ps99.OverlayValues[54] = d54
			ps99.OverlayValues[55] = d55
			ps99.OverlayValues[56] = d56
			ps99.OverlayValues[57] = d57
			ps99.OverlayValues[58] = d58
			ps99.OverlayValues[59] = d59
			ps99.OverlayValues[60] = d60
			ps99.OverlayValues[61] = d61
			ps99.OverlayValues[62] = d62
			ps99.OverlayValues[63] = d63
			ps99.OverlayValues[64] = d64
			ps99.OverlayValues[65] = d65
			ps99.OverlayValues[66] = d66
			ps99.OverlayValues[67] = d67
			ps99.OverlayValues[68] = d68
			ps99.OverlayValues[69] = d69
			ps99.OverlayValues[70] = d70
			ps99.OverlayValues[71] = d71
			ps99.OverlayValues[72] = d72
			ps99.OverlayValues[73] = d73
			ps99.OverlayValues[74] = d74
			ps99.OverlayValues[75] = d75
			ps99.OverlayValues[76] = d76
			ps99.OverlayValues[77] = d77
			ps99.OverlayValues[78] = d78
			ps99.OverlayValues[79] = d79
			ps99.OverlayValues[80] = d80
			ps99.OverlayValues[81] = d81
			ps99.OverlayValues[82] = d82
			ps99.OverlayValues[83] = d83
			ps99.OverlayValues[84] = d84
			ps99.OverlayValues[85] = d85
			ps99.OverlayValues[86] = d86
			ps99.OverlayValues[87] = d87
			ps99.OverlayValues[88] = d88
			ps99.OverlayValues[89] = d89
			ps99.OverlayValues[90] = d90
			ps99.OverlayValues[91] = d91
			ps99.OverlayValues[92] = d92
			ps99.OverlayValues[93] = d93
			ps99.OverlayValues[94] = d94
			ps99.OverlayValues[95] = d95
			snap100 := d1
			alloc101 := ctx.SnapshotAllocState()
			if !bbs[9].Rendered {
				bbs[9].RenderPS(ps99)
			}
			ctx.RestoreAllocState(alloc101)
			d1 = snap100
			if !bbs[7].Rendered {
				return bbs[7].RenderPS(ps98)
			}
			return result
			ctx.FreeDesc(&d94)
			return result
			}
			bbs[6].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[6].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
						d102 := ps.PhiValues[0]
						ctx.EnsureDesc(&d102)
						ctx.EmitStoreToStack(d102, 32)
					}
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
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != scm.LocNone {
				d102 = ps.OverlayValues[102]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d4 = ps.PhiValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d4)
			var d103 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d103 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(d4.Imm.Int()))))}
			} else {
				r57 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r57, d4.Reg)
				ctx.W.EmitShlRegImm8(r57, 32)
				ctx.W.EmitShrRegImm8(r57, 32)
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r57}
				ctx.BindReg(r57, &d103)
			}
			ctx.EnsureDesc(&d103)
			if thisptr.Loc == scm.LocImm {
				baseReg := ctx.AllocReg()
				if d103.Loc == scm.LocReg {
					ctx.FreeReg(baseReg)
					baseReg = ctx.AllocRegExcept(d103.Reg)
				}
				ctx.W.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).lastValue)))
				if d103.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d103.Imm.Int()))
					ctx.W.EmitStoreRegMem(scm.RegR11, baseReg, 0)
				} else {
					ctx.W.EmitStoreRegMem(d103.Reg, baseReg, 0)
				}
				ctx.FreeReg(baseReg)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
				if d103.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d103.Imm.Int()))
					ctx.W.EmitStoreRegMem(scm.RegR11, thisptr.Reg, off)
				} else {
					ctx.W.EmitStoreRegMem(d103.Reg, thisptr.Reg, off)
				}
			}
			ctx.FreeDesc(&d103)
			ctx.EnsureDesc(&d4)
			d104 = d4
			_ = d104
			r58 := d4.Loc == scm.LocReg
			r59 := d4.Reg
			if r58 { ctx.ProtectReg(r59) }
			d105 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(88)}
			lbl38 := ctx.W.ReserveLabel()
			bbpos_2_0 := int32(-1)
			_ = bbpos_2_0
			bbpos_2_1 := int32(-1)
			_ = bbpos_2_1
			bbpos_2_2 := int32(-1)
			_ = bbpos_2_2
			bbpos_2_3 := int32(-1)
			_ = bbpos_2_3
			bbpos_2_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			d105 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(88)}
			ctx.EnsureDesc(&d104)
			ctx.EnsureDesc(&d104)
			var d106 scm.JITValueDesc
			if d104.Loc == scm.LocImm {
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d104.Imm.Int()))))}
			} else {
				r60 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r60, d104.Reg)
				ctx.W.EmitShlRegImm8(r60, 32)
				ctx.W.EmitShrRegImm8(r60, 32)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r60}
				ctx.BindReg(r60, &d106)
			}
			var d107 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d107 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r61 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r61, thisptr.Reg, off)
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r61}
				ctx.BindReg(r61, &d107)
			}
			ctx.EnsureDesc(&d107)
			ctx.EnsureDesc(&d107)
			var d108 scm.JITValueDesc
			if d107.Loc == scm.LocImm {
				d108 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d107.Imm.Int()))))}
			} else {
				r62 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r62, d107.Reg)
				ctx.W.EmitShlRegImm8(r62, 56)
				ctx.W.EmitShrRegImm8(r62, 56)
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r62}
				ctx.BindReg(r62, &d108)
			}
			ctx.FreeDesc(&d107)
			ctx.EnsureDesc(&d106)
			ctx.EnsureDesc(&d108)
			ctx.EnsureDesc(&d106)
			ctx.EnsureDesc(&d108)
			ctx.EnsureDesc(&d106)
			ctx.EnsureDesc(&d108)
			var d109 scm.JITValueDesc
			if d106.Loc == scm.LocImm && d108.Loc == scm.LocImm {
				d109 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d106.Imm.Int() * d108.Imm.Int())}
			} else if d106.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d108.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d106.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d108.Reg)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d109)
			} else if d108.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d106.Reg)
				ctx.W.EmitMovRegReg(scratch, d106.Reg)
				if d108.Imm.Int() >= -2147483648 && d108.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d108.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d108.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d109)
			} else {
				r63 := ctx.AllocRegExcept(d106.Reg, d108.Reg)
				ctx.W.EmitMovRegReg(r63, d106.Reg)
				ctx.W.EmitImulInt64(r63, d108.Reg)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r63}
				ctx.BindReg(r63, &d109)
			}
			if d109.Loc == scm.LocReg && d106.Loc == scm.LocReg && d109.Reg == d106.Reg {
				ctx.TransferReg(d106.Reg)
				d106.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d106)
			ctx.FreeDesc(&d108)
			var d110 scm.JITValueDesc
			r64 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r64, uint64(dataPtr))
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r64, StackOff: int32(sliceLen)}
				ctx.BindReg(r64, &d110)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 0)
				ctx.W.EmitMovRegMem(r64, thisptr.Reg, off)
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r64}
				ctx.BindReg(r64, &d110)
			}
			ctx.BindReg(r64, &d110)
			ctx.EnsureDesc(&d109)
			var d111 scm.JITValueDesc
			if d109.Loc == scm.LocImm {
				d111 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d109.Imm.Int() / 64)}
			} else {
				r65 := ctx.AllocRegExcept(d109.Reg)
				ctx.W.EmitMovRegReg(r65, d109.Reg)
				ctx.W.EmitShrRegImm8(r65, 6)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
				ctx.BindReg(r65, &d111)
			}
			if d111.Loc == scm.LocReg && d109.Loc == scm.LocReg && d111.Reg == d109.Reg {
				ctx.TransferReg(d109.Reg)
				d109.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d111)
			r66 := ctx.AllocReg()
			ctx.EnsureDesc(&d111)
			ctx.EnsureDesc(&d110)
			if d111.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r66, uint64(d111.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r66, d111.Reg)
				ctx.W.EmitShlRegImm8(r66, 3)
			}
			if d110.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d110.Imm.Int()))
				ctx.W.EmitAddInt64(r66, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r66, d110.Reg)
			}
			r67 := ctx.AllocRegExcept(r66)
			ctx.W.EmitMovRegMem(r67, r66, 0)
			ctx.FreeReg(r66)
			d112 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r67}
			ctx.BindReg(r67, &d112)
			ctx.FreeDesc(&d111)
			ctx.EnsureDesc(&d109)
			var d113 scm.JITValueDesc
			if d109.Loc == scm.LocImm {
				d113 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d109.Imm.Int() % 64)}
			} else {
				r68 := ctx.AllocRegExcept(d109.Reg)
				ctx.W.EmitMovRegReg(r68, d109.Reg)
				ctx.W.EmitAndRegImm32(r68, 63)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r68}
				ctx.BindReg(r68, &d113)
			}
			if d113.Loc == scm.LocReg && d109.Loc == scm.LocReg && d113.Reg == d109.Reg {
				ctx.TransferReg(d109.Reg)
				d109.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d112)
			ctx.EnsureDesc(&d113)
			var d114 scm.JITValueDesc
			if d112.Loc == scm.LocImm && d113.Loc == scm.LocImm {
				d114 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d112.Imm.Int()) << uint64(d113.Imm.Int())))}
			} else if d113.Loc == scm.LocImm {
				r69 := ctx.AllocRegExcept(d112.Reg)
				ctx.W.EmitMovRegReg(r69, d112.Reg)
				ctx.W.EmitShlRegImm8(r69, uint8(d113.Imm.Int()))
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r69}
				ctx.BindReg(r69, &d114)
			} else {
				{
					shiftSrc := d112.Reg
					r70 := ctx.AllocRegExcept(d112.Reg)
					ctx.W.EmitMovRegReg(r70, d112.Reg)
					shiftSrc = r70
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d113.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d113.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d113.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d114)
				}
			}
			if d114.Loc == scm.LocReg && d112.Loc == scm.LocReg && d114.Reg == d112.Reg {
				ctx.TransferReg(d112.Reg)
				d112.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d112)
			ctx.FreeDesc(&d113)
			var d115 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d115 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 25)
				r71 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r71, thisptr.Reg, off)
				d115 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r71}
				ctx.BindReg(r71, &d115)
			}
			d116 = d115
			ctx.EnsureDesc(&d116)
			if d116.Loc != scm.LocImm && d116.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl39 := ctx.W.ReserveLabel()
			lbl40 := ctx.W.ReserveLabel()
			lbl41 := ctx.W.ReserveLabel()
			lbl42 := ctx.W.ReserveLabel()
			if d116.Loc == scm.LocImm {
				if d116.Imm.Bool() {
					ctx.W.MarkLabel(lbl41)
					ctx.W.EmitJmp(lbl39)
				} else {
					ctx.W.MarkLabel(lbl42)
			d117 = d114
			if d117.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d117)
			ctx.EmitStoreToStack(d117, 88)
					ctx.W.EmitJmp(lbl40)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d116.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl41)
				ctx.W.EmitJmp(lbl42)
				ctx.W.MarkLabel(lbl41)
				ctx.W.EmitJmp(lbl39)
				ctx.W.MarkLabel(lbl42)
			d118 = d114
			if d118.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d118)
			ctx.EmitStoreToStack(d118, 88)
				ctx.W.EmitJmp(lbl40)
			}
			ctx.FreeDesc(&d115)
			bbpos_2_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl40)
			ctx.W.ResolveFixups()
			d105 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(88)}
			var d119 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r72 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r72, thisptr.Reg, off)
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r72}
				ctx.BindReg(r72, &d119)
			}
			ctx.EnsureDesc(&d119)
			ctx.EnsureDesc(&d119)
			var d120 scm.JITValueDesc
			if d119.Loc == scm.LocImm {
				d120 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d119.Imm.Int()))))}
			} else {
				r73 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r73, d119.Reg)
				ctx.W.EmitShlRegImm8(r73, 56)
				ctx.W.EmitShrRegImm8(r73, 56)
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
				ctx.BindReg(r73, &d120)
			}
			ctx.FreeDesc(&d119)
			d121 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d120)
			ctx.EnsureDesc(&d121)
			ctx.EnsureDesc(&d120)
			ctx.EnsureDesc(&d121)
			ctx.EnsureDesc(&d120)
			var d122 scm.JITValueDesc
			if d121.Loc == scm.LocImm && d120.Loc == scm.LocImm {
				d122 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d121.Imm.Int() - d120.Imm.Int())}
			} else if d120.Loc == scm.LocImm && d120.Imm.Int() == 0 {
				r74 := ctx.AllocRegExcept(d121.Reg)
				ctx.W.EmitMovRegReg(r74, d121.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r74}
				ctx.BindReg(r74, &d122)
			} else if d121.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d120.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d121.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d120.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d122)
			} else if d120.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d121.Reg)
				ctx.W.EmitMovRegReg(scratch, d121.Reg)
				if d120.Imm.Int() >= -2147483648 && d120.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d120.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d120.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d122)
			} else {
				r75 := ctx.AllocRegExcept(d121.Reg, d120.Reg)
				ctx.W.EmitMovRegReg(r75, d121.Reg)
				ctx.W.EmitSubInt64(r75, d120.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
				ctx.BindReg(r75, &d122)
			}
			if d122.Loc == scm.LocReg && d121.Loc == scm.LocReg && d122.Reg == d121.Reg {
				ctx.TransferReg(d121.Reg)
				d121.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d120)
			ctx.EnsureDesc(&d105)
			ctx.EnsureDesc(&d122)
			var d123 scm.JITValueDesc
			if d105.Loc == scm.LocImm && d122.Loc == scm.LocImm {
				d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d105.Imm.Int()) >> uint64(d122.Imm.Int())))}
			} else if d122.Loc == scm.LocImm {
				r76 := ctx.AllocRegExcept(d105.Reg)
				ctx.W.EmitMovRegReg(r76, d105.Reg)
				ctx.W.EmitShrRegImm8(r76, uint8(d122.Imm.Int()))
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r76}
				ctx.BindReg(r76, &d123)
			} else {
				{
					shiftSrc := d105.Reg
					r77 := ctx.AllocRegExcept(d105.Reg)
					ctx.W.EmitMovRegReg(r77, d105.Reg)
					shiftSrc = r77
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d122.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d122.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d122.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d123)
				}
			}
			if d123.Loc == scm.LocReg && d105.Loc == scm.LocReg && d123.Reg == d105.Reg {
				ctx.TransferReg(d105.Reg)
				d105.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d105)
			ctx.FreeDesc(&d122)
			r78 := ctx.AllocReg()
			ctx.EnsureDesc(&d123)
			ctx.EnsureDesc(&d123)
			if d123.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r78, d123)
			}
			ctx.W.EmitJmp(lbl38)
			bbpos_2_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl39)
			ctx.W.ResolveFixups()
			d105 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(88)}
			ctx.EnsureDesc(&d109)
			var d124 scm.JITValueDesc
			if d109.Loc == scm.LocImm {
				d124 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d109.Imm.Int() % 64)}
			} else {
				r79 := ctx.AllocRegExcept(d109.Reg)
				ctx.W.EmitMovRegReg(r79, d109.Reg)
				ctx.W.EmitAndRegImm32(r79, 63)
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r79}
				ctx.BindReg(r79, &d124)
			}
			if d124.Loc == scm.LocReg && d109.Loc == scm.LocReg && d124.Reg == d109.Reg {
				ctx.TransferReg(d109.Reg)
				d109.Loc = scm.LocNone
			}
			var d125 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d125 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r80 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r80, thisptr.Reg, off)
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r80}
				ctx.BindReg(r80, &d125)
			}
			ctx.EnsureDesc(&d125)
			ctx.EnsureDesc(&d125)
			var d126 scm.JITValueDesc
			if d125.Loc == scm.LocImm {
				d126 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d125.Imm.Int()))))}
			} else {
				r81 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r81, d125.Reg)
				ctx.W.EmitShlRegImm8(r81, 56)
				ctx.W.EmitShrRegImm8(r81, 56)
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
				ctx.BindReg(r81, &d126)
			}
			ctx.FreeDesc(&d125)
			ctx.EnsureDesc(&d124)
			ctx.EnsureDesc(&d126)
			ctx.EnsureDesc(&d124)
			ctx.EnsureDesc(&d126)
			ctx.EnsureDesc(&d124)
			ctx.EnsureDesc(&d126)
			var d127 scm.JITValueDesc
			if d124.Loc == scm.LocImm && d126.Loc == scm.LocImm {
				d127 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d124.Imm.Int() + d126.Imm.Int())}
			} else if d126.Loc == scm.LocImm && d126.Imm.Int() == 0 {
				r82 := ctx.AllocRegExcept(d124.Reg)
				ctx.W.EmitMovRegReg(r82, d124.Reg)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r82}
				ctx.BindReg(r82, &d127)
			} else if d124.Loc == scm.LocImm && d124.Imm.Int() == 0 {
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d126.Reg}
				ctx.BindReg(d126.Reg, &d127)
			} else if d124.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d126.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d124.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d126.Reg)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d127)
			} else if d126.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d124.Reg)
				ctx.W.EmitMovRegReg(scratch, d124.Reg)
				if d126.Imm.Int() >= -2147483648 && d126.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d126.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d126.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d127)
			} else {
				r83 := ctx.AllocRegExcept(d124.Reg, d126.Reg)
				ctx.W.EmitMovRegReg(r83, d124.Reg)
				ctx.W.EmitAddInt64(r83, d126.Reg)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
				ctx.BindReg(r83, &d127)
			}
			if d127.Loc == scm.LocReg && d124.Loc == scm.LocReg && d127.Reg == d124.Reg {
				ctx.TransferReg(d124.Reg)
				d124.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d124)
			ctx.FreeDesc(&d126)
			ctx.EnsureDesc(&d127)
			var d128 scm.JITValueDesc
			if d127.Loc == scm.LocImm {
				d128 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d127.Imm.Int()) > uint64(64))}
			} else {
				r84 := ctx.AllocRegExcept(d127.Reg)
				ctx.W.EmitCmpRegImm32(d127.Reg, 64)
				ctx.W.EmitSetcc(r84, scm.CcA)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r84}
				ctx.BindReg(r84, &d128)
			}
			ctx.FreeDesc(&d127)
			d129 = d128
			ctx.EnsureDesc(&d129)
			if d129.Loc != scm.LocImm && d129.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl43 := ctx.W.ReserveLabel()
			lbl44 := ctx.W.ReserveLabel()
			lbl45 := ctx.W.ReserveLabel()
			if d129.Loc == scm.LocImm {
				if d129.Imm.Bool() {
					ctx.W.MarkLabel(lbl44)
					ctx.W.EmitJmp(lbl43)
				} else {
					ctx.W.MarkLabel(lbl45)
			d130 = d114
			if d130.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d130)
			ctx.EmitStoreToStack(d130, 88)
					ctx.W.EmitJmp(lbl40)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d129.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl44)
				ctx.W.EmitJmp(lbl45)
				ctx.W.MarkLabel(lbl44)
				ctx.W.EmitJmp(lbl43)
				ctx.W.MarkLabel(lbl45)
			d131 = d114
			if d131.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d131)
			ctx.EmitStoreToStack(d131, 88)
				ctx.W.EmitJmp(lbl40)
			}
			ctx.FreeDesc(&d128)
			bbpos_2_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl43)
			ctx.W.ResolveFixups()
			d105 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(88)}
			ctx.EnsureDesc(&d109)
			var d132 scm.JITValueDesc
			if d109.Loc == scm.LocImm {
				d132 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d109.Imm.Int() / 64)}
			} else {
				r85 := ctx.AllocRegExcept(d109.Reg)
				ctx.W.EmitMovRegReg(r85, d109.Reg)
				ctx.W.EmitShrRegImm8(r85, 6)
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
				ctx.BindReg(r85, &d132)
			}
			if d132.Loc == scm.LocReg && d109.Loc == scm.LocReg && d132.Reg == d109.Reg {
				ctx.TransferReg(d109.Reg)
				d109.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d132)
			ctx.EnsureDesc(&d132)
			var d133 scm.JITValueDesc
			if d132.Loc == scm.LocImm {
				d133 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d132.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d132.Reg)
				ctx.W.EmitMovRegReg(scratch, d132.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d133)
			}
			if d133.Loc == scm.LocReg && d132.Loc == scm.LocReg && d133.Reg == d132.Reg {
				ctx.TransferReg(d132.Reg)
				d132.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d132)
			ctx.EnsureDesc(&d133)
			r86 := ctx.AllocReg()
			ctx.EnsureDesc(&d133)
			ctx.EnsureDesc(&d110)
			if d133.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r86, uint64(d133.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r86, d133.Reg)
				ctx.W.EmitShlRegImm8(r86, 3)
			}
			if d110.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d110.Imm.Int()))
				ctx.W.EmitAddInt64(r86, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r86, d110.Reg)
			}
			r87 := ctx.AllocRegExcept(r86)
			ctx.W.EmitMovRegMem(r87, r86, 0)
			ctx.FreeReg(r86)
			d134 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r87}
			ctx.BindReg(r87, &d134)
			ctx.FreeDesc(&d133)
			ctx.EnsureDesc(&d109)
			var d135 scm.JITValueDesc
			if d109.Loc == scm.LocImm {
				d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d109.Imm.Int() % 64)}
			} else {
				r88 := ctx.AllocRegExcept(d109.Reg)
				ctx.W.EmitMovRegReg(r88, d109.Reg)
				ctx.W.EmitAndRegImm32(r88, 63)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
				ctx.BindReg(r88, &d135)
			}
			if d135.Loc == scm.LocReg && d109.Loc == scm.LocReg && d135.Reg == d109.Reg {
				ctx.TransferReg(d109.Reg)
				d109.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d109)
			d136 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d135)
			ctx.EnsureDesc(&d136)
			ctx.EnsureDesc(&d135)
			ctx.EnsureDesc(&d136)
			ctx.EnsureDesc(&d135)
			var d137 scm.JITValueDesc
			if d136.Loc == scm.LocImm && d135.Loc == scm.LocImm {
				d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d136.Imm.Int() - d135.Imm.Int())}
			} else if d135.Loc == scm.LocImm && d135.Imm.Int() == 0 {
				r89 := ctx.AllocRegExcept(d136.Reg)
				ctx.W.EmitMovRegReg(r89, d136.Reg)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
				ctx.BindReg(r89, &d137)
			} else if d136.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d135.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d136.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d135.Reg)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d137)
			} else if d135.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d136.Reg)
				ctx.W.EmitMovRegReg(scratch, d136.Reg)
				if d135.Imm.Int() >= -2147483648 && d135.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d135.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d135.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d137)
			} else {
				r90 := ctx.AllocRegExcept(d136.Reg, d135.Reg)
				ctx.W.EmitMovRegReg(r90, d136.Reg)
				ctx.W.EmitSubInt64(r90, d135.Reg)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
				ctx.BindReg(r90, &d137)
			}
			if d137.Loc == scm.LocReg && d136.Loc == scm.LocReg && d137.Reg == d136.Reg {
				ctx.TransferReg(d136.Reg)
				d136.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d135)
			ctx.EnsureDesc(&d134)
			ctx.EnsureDesc(&d137)
			var d138 scm.JITValueDesc
			if d134.Loc == scm.LocImm && d137.Loc == scm.LocImm {
				d138 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d134.Imm.Int()) >> uint64(d137.Imm.Int())))}
			} else if d137.Loc == scm.LocImm {
				r91 := ctx.AllocRegExcept(d134.Reg)
				ctx.W.EmitMovRegReg(r91, d134.Reg)
				ctx.W.EmitShrRegImm8(r91, uint8(d137.Imm.Int()))
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r91}
				ctx.BindReg(r91, &d138)
			} else {
				{
					shiftSrc := d134.Reg
					r92 := ctx.AllocRegExcept(d134.Reg)
					ctx.W.EmitMovRegReg(r92, d134.Reg)
					shiftSrc = r92
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d137.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d137.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d137.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d138)
				}
			}
			if d138.Loc == scm.LocReg && d134.Loc == scm.LocReg && d138.Reg == d134.Reg {
				ctx.TransferReg(d134.Reg)
				d134.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d134)
			ctx.FreeDesc(&d137)
			ctx.EnsureDesc(&d114)
			ctx.EnsureDesc(&d138)
			var d139 scm.JITValueDesc
			if d114.Loc == scm.LocImm && d138.Loc == scm.LocImm {
				d139 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d114.Imm.Int() | d138.Imm.Int())}
			} else if d114.Loc == scm.LocImm && d114.Imm.Int() == 0 {
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d138.Reg}
				ctx.BindReg(d138.Reg, &d139)
			} else if d138.Loc == scm.LocImm && d138.Imm.Int() == 0 {
				r93 := ctx.AllocRegExcept(d114.Reg)
				ctx.W.EmitMovRegReg(r93, d114.Reg)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r93}
				ctx.BindReg(r93, &d139)
			} else if d114.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d138.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d114.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d138.Reg)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d139)
			} else if d138.Loc == scm.LocImm {
				r94 := ctx.AllocRegExcept(d114.Reg)
				ctx.W.EmitMovRegReg(r94, d114.Reg)
				if d138.Imm.Int() >= -2147483648 && d138.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r94, int32(d138.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d138.Imm.Int()))
					ctx.W.EmitOrInt64(r94, scm.RegR11)
				}
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
				ctx.BindReg(r94, &d139)
			} else {
				r95 := ctx.AllocRegExcept(d114.Reg, d138.Reg)
				ctx.W.EmitMovRegReg(r95, d114.Reg)
				ctx.W.EmitOrInt64(r95, d138.Reg)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r95}
				ctx.BindReg(r95, &d139)
			}
			if d139.Loc == scm.LocReg && d114.Loc == scm.LocReg && d139.Reg == d114.Reg {
				ctx.TransferReg(d114.Reg)
				d114.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d138)
			d140 = d139
			if d140.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d140)
			ctx.EmitStoreToStack(d140, 88)
			ctx.W.EmitJmp(lbl40)
			ctx.W.MarkLabel(lbl38)
			d141 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r78}
			ctx.BindReg(r78, &d141)
			ctx.BindReg(r78, &d141)
			if r58 { ctx.UnprotectReg(r59) }
			ctx.EnsureDesc(&d141)
			ctx.EnsureDesc(&d141)
			var d142 scm.JITValueDesc
			if d141.Loc == scm.LocImm {
				d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d141.Imm.Int()))))}
			} else {
				r96 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r96, d141.Reg)
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
				ctx.BindReg(r96, &d142)
			}
			ctx.FreeDesc(&d141)
			var d143 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d143 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 32)
				r97 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r97, thisptr.Reg, off)
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r97}
				ctx.BindReg(r97, &d143)
			}
			ctx.EnsureDesc(&d142)
			ctx.EnsureDesc(&d143)
			ctx.EnsureDesc(&d142)
			ctx.EnsureDesc(&d143)
			ctx.EnsureDesc(&d142)
			ctx.EnsureDesc(&d143)
			var d144 scm.JITValueDesc
			if d142.Loc == scm.LocImm && d143.Loc == scm.LocImm {
				d144 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d142.Imm.Int() + d143.Imm.Int())}
			} else if d143.Loc == scm.LocImm && d143.Imm.Int() == 0 {
				r98 := ctx.AllocRegExcept(d142.Reg)
				ctx.W.EmitMovRegReg(r98, d142.Reg)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r98}
				ctx.BindReg(r98, &d144)
			} else if d142.Loc == scm.LocImm && d142.Imm.Int() == 0 {
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d143.Reg}
				ctx.BindReg(d143.Reg, &d144)
			} else if d142.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d143.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d142.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d143.Reg)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d144)
			} else if d143.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d142.Reg)
				ctx.W.EmitMovRegReg(scratch, d142.Reg)
				if d143.Imm.Int() >= -2147483648 && d143.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d143.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d143.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d144)
			} else {
				r99 := ctx.AllocRegExcept(d142.Reg, d143.Reg)
				ctx.W.EmitMovRegReg(r99, d142.Reg)
				ctx.W.EmitAddInt64(r99, d143.Reg)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
				ctx.BindReg(r99, &d144)
			}
			if d144.Loc == scm.LocReg && d142.Loc == scm.LocReg && d144.Reg == d142.Reg {
				ctx.TransferReg(d142.Reg)
				d142.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d142)
			ctx.FreeDesc(&d143)
			var d145 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 56)
				r100 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r100, thisptr.Reg, off)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r100}
				ctx.BindReg(r100, &d145)
			}
			d146 = d145
			ctx.EnsureDesc(&d146)
			if d146.Loc != scm.LocImm && d146.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d146.Loc == scm.LocImm {
				if d146.Imm.Bool() {
			ps147 := scm.PhiState{General: ps.General}
			ps147.OverlayValues = make([]scm.JITValueDesc, 147)
			ps147.OverlayValues[0] = d0
			ps147.OverlayValues[1] = d1
			ps147.OverlayValues[2] = d2
			ps147.OverlayValues[3] = d3
			ps147.OverlayValues[4] = d4
			ps147.OverlayValues[5] = d5
			ps147.OverlayValues[6] = d6
			ps147.OverlayValues[7] = d7
			ps147.OverlayValues[8] = d8
			ps147.OverlayValues[9] = d9
			ps147.OverlayValues[10] = d10
			ps147.OverlayValues[11] = d11
			ps147.OverlayValues[12] = d12
			ps147.OverlayValues[13] = d13
			ps147.OverlayValues[14] = d14
			ps147.OverlayValues[20] = d20
			ps147.OverlayValues[21] = d21
			ps147.OverlayValues[22] = d22
			ps147.OverlayValues[24] = d24
			ps147.OverlayValues[25] = d25
			ps147.OverlayValues[27] = d27
			ps147.OverlayValues[28] = d28
			ps147.OverlayValues[29] = d29
			ps147.OverlayValues[32] = d32
			ps147.OverlayValues[34] = d34
			ps147.OverlayValues[35] = d35
			ps147.OverlayValues[36] = d36
			ps147.OverlayValues[38] = d38
			ps147.OverlayValues[39] = d39
			ps147.OverlayValues[40] = d40
			ps147.OverlayValues[41] = d41
			ps147.OverlayValues[42] = d42
			ps147.OverlayValues[43] = d43
			ps147.OverlayValues[44] = d44
			ps147.OverlayValues[46] = d46
			ps147.OverlayValues[47] = d47
			ps147.OverlayValues[48] = d48
			ps147.OverlayValues[49] = d49
			ps147.OverlayValues[50] = d50
			ps147.OverlayValues[51] = d51
			ps147.OverlayValues[52] = d52
			ps147.OverlayValues[53] = d53
			ps147.OverlayValues[54] = d54
			ps147.OverlayValues[55] = d55
			ps147.OverlayValues[56] = d56
			ps147.OverlayValues[57] = d57
			ps147.OverlayValues[58] = d58
			ps147.OverlayValues[59] = d59
			ps147.OverlayValues[60] = d60
			ps147.OverlayValues[61] = d61
			ps147.OverlayValues[62] = d62
			ps147.OverlayValues[63] = d63
			ps147.OverlayValues[64] = d64
			ps147.OverlayValues[65] = d65
			ps147.OverlayValues[66] = d66
			ps147.OverlayValues[67] = d67
			ps147.OverlayValues[68] = d68
			ps147.OverlayValues[69] = d69
			ps147.OverlayValues[70] = d70
			ps147.OverlayValues[71] = d71
			ps147.OverlayValues[72] = d72
			ps147.OverlayValues[73] = d73
			ps147.OverlayValues[74] = d74
			ps147.OverlayValues[75] = d75
			ps147.OverlayValues[76] = d76
			ps147.OverlayValues[77] = d77
			ps147.OverlayValues[78] = d78
			ps147.OverlayValues[79] = d79
			ps147.OverlayValues[80] = d80
			ps147.OverlayValues[81] = d81
			ps147.OverlayValues[82] = d82
			ps147.OverlayValues[83] = d83
			ps147.OverlayValues[84] = d84
			ps147.OverlayValues[85] = d85
			ps147.OverlayValues[86] = d86
			ps147.OverlayValues[87] = d87
			ps147.OverlayValues[88] = d88
			ps147.OverlayValues[89] = d89
			ps147.OverlayValues[90] = d90
			ps147.OverlayValues[91] = d91
			ps147.OverlayValues[92] = d92
			ps147.OverlayValues[93] = d93
			ps147.OverlayValues[94] = d94
			ps147.OverlayValues[95] = d95
			ps147.OverlayValues[102] = d102
			ps147.OverlayValues[103] = d103
			ps147.OverlayValues[104] = d104
			ps147.OverlayValues[105] = d105
			ps147.OverlayValues[106] = d106
			ps147.OverlayValues[107] = d107
			ps147.OverlayValues[108] = d108
			ps147.OverlayValues[109] = d109
			ps147.OverlayValues[110] = d110
			ps147.OverlayValues[111] = d111
			ps147.OverlayValues[112] = d112
			ps147.OverlayValues[113] = d113
			ps147.OverlayValues[114] = d114
			ps147.OverlayValues[115] = d115
			ps147.OverlayValues[116] = d116
			ps147.OverlayValues[117] = d117
			ps147.OverlayValues[118] = d118
			ps147.OverlayValues[119] = d119
			ps147.OverlayValues[120] = d120
			ps147.OverlayValues[121] = d121
			ps147.OverlayValues[122] = d122
			ps147.OverlayValues[123] = d123
			ps147.OverlayValues[124] = d124
			ps147.OverlayValues[125] = d125
			ps147.OverlayValues[126] = d126
			ps147.OverlayValues[127] = d127
			ps147.OverlayValues[128] = d128
			ps147.OverlayValues[129] = d129
			ps147.OverlayValues[130] = d130
			ps147.OverlayValues[131] = d131
			ps147.OverlayValues[132] = d132
			ps147.OverlayValues[133] = d133
			ps147.OverlayValues[134] = d134
			ps147.OverlayValues[135] = d135
			ps147.OverlayValues[136] = d136
			ps147.OverlayValues[137] = d137
			ps147.OverlayValues[138] = d138
			ps147.OverlayValues[139] = d139
			ps147.OverlayValues[140] = d140
			ps147.OverlayValues[141] = d141
			ps147.OverlayValues[142] = d142
			ps147.OverlayValues[143] = d143
			ps147.OverlayValues[144] = d144
			ps147.OverlayValues[145] = d145
			ps147.OverlayValues[146] = d146
					return bbs[22].RenderPS(ps147)
				}
			ps148 := scm.PhiState{General: ps.General}
			ps148.OverlayValues = make([]scm.JITValueDesc, 147)
			ps148.OverlayValues[0] = d0
			ps148.OverlayValues[1] = d1
			ps148.OverlayValues[2] = d2
			ps148.OverlayValues[3] = d3
			ps148.OverlayValues[4] = d4
			ps148.OverlayValues[5] = d5
			ps148.OverlayValues[6] = d6
			ps148.OverlayValues[7] = d7
			ps148.OverlayValues[8] = d8
			ps148.OverlayValues[9] = d9
			ps148.OverlayValues[10] = d10
			ps148.OverlayValues[11] = d11
			ps148.OverlayValues[12] = d12
			ps148.OverlayValues[13] = d13
			ps148.OverlayValues[14] = d14
			ps148.OverlayValues[20] = d20
			ps148.OverlayValues[21] = d21
			ps148.OverlayValues[22] = d22
			ps148.OverlayValues[24] = d24
			ps148.OverlayValues[25] = d25
			ps148.OverlayValues[27] = d27
			ps148.OverlayValues[28] = d28
			ps148.OverlayValues[29] = d29
			ps148.OverlayValues[32] = d32
			ps148.OverlayValues[34] = d34
			ps148.OverlayValues[35] = d35
			ps148.OverlayValues[36] = d36
			ps148.OverlayValues[38] = d38
			ps148.OverlayValues[39] = d39
			ps148.OverlayValues[40] = d40
			ps148.OverlayValues[41] = d41
			ps148.OverlayValues[42] = d42
			ps148.OverlayValues[43] = d43
			ps148.OverlayValues[44] = d44
			ps148.OverlayValues[46] = d46
			ps148.OverlayValues[47] = d47
			ps148.OverlayValues[48] = d48
			ps148.OverlayValues[49] = d49
			ps148.OverlayValues[50] = d50
			ps148.OverlayValues[51] = d51
			ps148.OverlayValues[52] = d52
			ps148.OverlayValues[53] = d53
			ps148.OverlayValues[54] = d54
			ps148.OverlayValues[55] = d55
			ps148.OverlayValues[56] = d56
			ps148.OverlayValues[57] = d57
			ps148.OverlayValues[58] = d58
			ps148.OverlayValues[59] = d59
			ps148.OverlayValues[60] = d60
			ps148.OverlayValues[61] = d61
			ps148.OverlayValues[62] = d62
			ps148.OverlayValues[63] = d63
			ps148.OverlayValues[64] = d64
			ps148.OverlayValues[65] = d65
			ps148.OverlayValues[66] = d66
			ps148.OverlayValues[67] = d67
			ps148.OverlayValues[68] = d68
			ps148.OverlayValues[69] = d69
			ps148.OverlayValues[70] = d70
			ps148.OverlayValues[71] = d71
			ps148.OverlayValues[72] = d72
			ps148.OverlayValues[73] = d73
			ps148.OverlayValues[74] = d74
			ps148.OverlayValues[75] = d75
			ps148.OverlayValues[76] = d76
			ps148.OverlayValues[77] = d77
			ps148.OverlayValues[78] = d78
			ps148.OverlayValues[79] = d79
			ps148.OverlayValues[80] = d80
			ps148.OverlayValues[81] = d81
			ps148.OverlayValues[82] = d82
			ps148.OverlayValues[83] = d83
			ps148.OverlayValues[84] = d84
			ps148.OverlayValues[85] = d85
			ps148.OverlayValues[86] = d86
			ps148.OverlayValues[87] = d87
			ps148.OverlayValues[88] = d88
			ps148.OverlayValues[89] = d89
			ps148.OverlayValues[90] = d90
			ps148.OverlayValues[91] = d91
			ps148.OverlayValues[92] = d92
			ps148.OverlayValues[93] = d93
			ps148.OverlayValues[94] = d94
			ps148.OverlayValues[95] = d95
			ps148.OverlayValues[102] = d102
			ps148.OverlayValues[103] = d103
			ps148.OverlayValues[104] = d104
			ps148.OverlayValues[105] = d105
			ps148.OverlayValues[106] = d106
			ps148.OverlayValues[107] = d107
			ps148.OverlayValues[108] = d108
			ps148.OverlayValues[109] = d109
			ps148.OverlayValues[110] = d110
			ps148.OverlayValues[111] = d111
			ps148.OverlayValues[112] = d112
			ps148.OverlayValues[113] = d113
			ps148.OverlayValues[114] = d114
			ps148.OverlayValues[115] = d115
			ps148.OverlayValues[116] = d116
			ps148.OverlayValues[117] = d117
			ps148.OverlayValues[118] = d118
			ps148.OverlayValues[119] = d119
			ps148.OverlayValues[120] = d120
			ps148.OverlayValues[121] = d121
			ps148.OverlayValues[122] = d122
			ps148.OverlayValues[123] = d123
			ps148.OverlayValues[124] = d124
			ps148.OverlayValues[125] = d125
			ps148.OverlayValues[126] = d126
			ps148.OverlayValues[127] = d127
			ps148.OverlayValues[128] = d128
			ps148.OverlayValues[129] = d129
			ps148.OverlayValues[130] = d130
			ps148.OverlayValues[131] = d131
			ps148.OverlayValues[132] = d132
			ps148.OverlayValues[133] = d133
			ps148.OverlayValues[134] = d134
			ps148.OverlayValues[135] = d135
			ps148.OverlayValues[136] = d136
			ps148.OverlayValues[137] = d137
			ps148.OverlayValues[138] = d138
			ps148.OverlayValues[139] = d139
			ps148.OverlayValues[140] = d140
			ps148.OverlayValues[141] = d141
			ps148.OverlayValues[142] = d142
			ps148.OverlayValues[143] = d143
			ps148.OverlayValues[144] = d144
			ps148.OverlayValues[145] = d145
			ps148.OverlayValues[146] = d146
				return bbs[21].RenderPS(ps148)
			}
			lbl46 := ctx.W.ReserveLabel()
			lbl47 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d146.Reg, 0)
			ctx.W.EmitJcc(scm.CcNE, lbl46)
			ctx.W.EmitJmp(lbl47)
			ctx.W.MarkLabel(lbl46)
			ctx.W.EmitJmp(lbl23)
			ctx.W.MarkLabel(lbl47)
			ctx.W.EmitJmp(lbl22)
			ps149 := scm.PhiState{General: true}
			ps149.OverlayValues = make([]scm.JITValueDesc, 147)
			ps149.OverlayValues[0] = d0
			ps149.OverlayValues[1] = d1
			ps149.OverlayValues[2] = d2
			ps149.OverlayValues[3] = d3
			ps149.OverlayValues[4] = d4
			ps149.OverlayValues[5] = d5
			ps149.OverlayValues[6] = d6
			ps149.OverlayValues[7] = d7
			ps149.OverlayValues[8] = d8
			ps149.OverlayValues[9] = d9
			ps149.OverlayValues[10] = d10
			ps149.OverlayValues[11] = d11
			ps149.OverlayValues[12] = d12
			ps149.OverlayValues[13] = d13
			ps149.OverlayValues[14] = d14
			ps149.OverlayValues[20] = d20
			ps149.OverlayValues[21] = d21
			ps149.OverlayValues[22] = d22
			ps149.OverlayValues[24] = d24
			ps149.OverlayValues[25] = d25
			ps149.OverlayValues[27] = d27
			ps149.OverlayValues[28] = d28
			ps149.OverlayValues[29] = d29
			ps149.OverlayValues[32] = d32
			ps149.OverlayValues[34] = d34
			ps149.OverlayValues[35] = d35
			ps149.OverlayValues[36] = d36
			ps149.OverlayValues[38] = d38
			ps149.OverlayValues[39] = d39
			ps149.OverlayValues[40] = d40
			ps149.OverlayValues[41] = d41
			ps149.OverlayValues[42] = d42
			ps149.OverlayValues[43] = d43
			ps149.OverlayValues[44] = d44
			ps149.OverlayValues[46] = d46
			ps149.OverlayValues[47] = d47
			ps149.OverlayValues[48] = d48
			ps149.OverlayValues[49] = d49
			ps149.OverlayValues[50] = d50
			ps149.OverlayValues[51] = d51
			ps149.OverlayValues[52] = d52
			ps149.OverlayValues[53] = d53
			ps149.OverlayValues[54] = d54
			ps149.OverlayValues[55] = d55
			ps149.OverlayValues[56] = d56
			ps149.OverlayValues[57] = d57
			ps149.OverlayValues[58] = d58
			ps149.OverlayValues[59] = d59
			ps149.OverlayValues[60] = d60
			ps149.OverlayValues[61] = d61
			ps149.OverlayValues[62] = d62
			ps149.OverlayValues[63] = d63
			ps149.OverlayValues[64] = d64
			ps149.OverlayValues[65] = d65
			ps149.OverlayValues[66] = d66
			ps149.OverlayValues[67] = d67
			ps149.OverlayValues[68] = d68
			ps149.OverlayValues[69] = d69
			ps149.OverlayValues[70] = d70
			ps149.OverlayValues[71] = d71
			ps149.OverlayValues[72] = d72
			ps149.OverlayValues[73] = d73
			ps149.OverlayValues[74] = d74
			ps149.OverlayValues[75] = d75
			ps149.OverlayValues[76] = d76
			ps149.OverlayValues[77] = d77
			ps149.OverlayValues[78] = d78
			ps149.OverlayValues[79] = d79
			ps149.OverlayValues[80] = d80
			ps149.OverlayValues[81] = d81
			ps149.OverlayValues[82] = d82
			ps149.OverlayValues[83] = d83
			ps149.OverlayValues[84] = d84
			ps149.OverlayValues[85] = d85
			ps149.OverlayValues[86] = d86
			ps149.OverlayValues[87] = d87
			ps149.OverlayValues[88] = d88
			ps149.OverlayValues[89] = d89
			ps149.OverlayValues[90] = d90
			ps149.OverlayValues[91] = d91
			ps149.OverlayValues[92] = d92
			ps149.OverlayValues[93] = d93
			ps149.OverlayValues[94] = d94
			ps149.OverlayValues[95] = d95
			ps149.OverlayValues[102] = d102
			ps149.OverlayValues[103] = d103
			ps149.OverlayValues[104] = d104
			ps149.OverlayValues[105] = d105
			ps149.OverlayValues[106] = d106
			ps149.OverlayValues[107] = d107
			ps149.OverlayValues[108] = d108
			ps149.OverlayValues[109] = d109
			ps149.OverlayValues[110] = d110
			ps149.OverlayValues[111] = d111
			ps149.OverlayValues[112] = d112
			ps149.OverlayValues[113] = d113
			ps149.OverlayValues[114] = d114
			ps149.OverlayValues[115] = d115
			ps149.OverlayValues[116] = d116
			ps149.OverlayValues[117] = d117
			ps149.OverlayValues[118] = d118
			ps149.OverlayValues[119] = d119
			ps149.OverlayValues[120] = d120
			ps149.OverlayValues[121] = d121
			ps149.OverlayValues[122] = d122
			ps149.OverlayValues[123] = d123
			ps149.OverlayValues[124] = d124
			ps149.OverlayValues[125] = d125
			ps149.OverlayValues[126] = d126
			ps149.OverlayValues[127] = d127
			ps149.OverlayValues[128] = d128
			ps149.OverlayValues[129] = d129
			ps149.OverlayValues[130] = d130
			ps149.OverlayValues[131] = d131
			ps149.OverlayValues[132] = d132
			ps149.OverlayValues[133] = d133
			ps149.OverlayValues[134] = d134
			ps149.OverlayValues[135] = d135
			ps149.OverlayValues[136] = d136
			ps149.OverlayValues[137] = d137
			ps149.OverlayValues[138] = d138
			ps149.OverlayValues[139] = d139
			ps149.OverlayValues[140] = d140
			ps149.OverlayValues[141] = d141
			ps149.OverlayValues[142] = d142
			ps149.OverlayValues[143] = d143
			ps149.OverlayValues[144] = d144
			ps149.OverlayValues[145] = d145
			ps149.OverlayValues[146] = d146
			ps150 := scm.PhiState{General: true}
			ps150.OverlayValues = make([]scm.JITValueDesc, 147)
			ps150.OverlayValues[0] = d0
			ps150.OverlayValues[1] = d1
			ps150.OverlayValues[2] = d2
			ps150.OverlayValues[3] = d3
			ps150.OverlayValues[4] = d4
			ps150.OverlayValues[5] = d5
			ps150.OverlayValues[6] = d6
			ps150.OverlayValues[7] = d7
			ps150.OverlayValues[8] = d8
			ps150.OverlayValues[9] = d9
			ps150.OverlayValues[10] = d10
			ps150.OverlayValues[11] = d11
			ps150.OverlayValues[12] = d12
			ps150.OverlayValues[13] = d13
			ps150.OverlayValues[14] = d14
			ps150.OverlayValues[20] = d20
			ps150.OverlayValues[21] = d21
			ps150.OverlayValues[22] = d22
			ps150.OverlayValues[24] = d24
			ps150.OverlayValues[25] = d25
			ps150.OverlayValues[27] = d27
			ps150.OverlayValues[28] = d28
			ps150.OverlayValues[29] = d29
			ps150.OverlayValues[32] = d32
			ps150.OverlayValues[34] = d34
			ps150.OverlayValues[35] = d35
			ps150.OverlayValues[36] = d36
			ps150.OverlayValues[38] = d38
			ps150.OverlayValues[39] = d39
			ps150.OverlayValues[40] = d40
			ps150.OverlayValues[41] = d41
			ps150.OverlayValues[42] = d42
			ps150.OverlayValues[43] = d43
			ps150.OverlayValues[44] = d44
			ps150.OverlayValues[46] = d46
			ps150.OverlayValues[47] = d47
			ps150.OverlayValues[48] = d48
			ps150.OverlayValues[49] = d49
			ps150.OverlayValues[50] = d50
			ps150.OverlayValues[51] = d51
			ps150.OverlayValues[52] = d52
			ps150.OverlayValues[53] = d53
			ps150.OverlayValues[54] = d54
			ps150.OverlayValues[55] = d55
			ps150.OverlayValues[56] = d56
			ps150.OverlayValues[57] = d57
			ps150.OverlayValues[58] = d58
			ps150.OverlayValues[59] = d59
			ps150.OverlayValues[60] = d60
			ps150.OverlayValues[61] = d61
			ps150.OverlayValues[62] = d62
			ps150.OverlayValues[63] = d63
			ps150.OverlayValues[64] = d64
			ps150.OverlayValues[65] = d65
			ps150.OverlayValues[66] = d66
			ps150.OverlayValues[67] = d67
			ps150.OverlayValues[68] = d68
			ps150.OverlayValues[69] = d69
			ps150.OverlayValues[70] = d70
			ps150.OverlayValues[71] = d71
			ps150.OverlayValues[72] = d72
			ps150.OverlayValues[73] = d73
			ps150.OverlayValues[74] = d74
			ps150.OverlayValues[75] = d75
			ps150.OverlayValues[76] = d76
			ps150.OverlayValues[77] = d77
			ps150.OverlayValues[78] = d78
			ps150.OverlayValues[79] = d79
			ps150.OverlayValues[80] = d80
			ps150.OverlayValues[81] = d81
			ps150.OverlayValues[82] = d82
			ps150.OverlayValues[83] = d83
			ps150.OverlayValues[84] = d84
			ps150.OverlayValues[85] = d85
			ps150.OverlayValues[86] = d86
			ps150.OverlayValues[87] = d87
			ps150.OverlayValues[88] = d88
			ps150.OverlayValues[89] = d89
			ps150.OverlayValues[90] = d90
			ps150.OverlayValues[91] = d91
			ps150.OverlayValues[92] = d92
			ps150.OverlayValues[93] = d93
			ps150.OverlayValues[94] = d94
			ps150.OverlayValues[95] = d95
			ps150.OverlayValues[102] = d102
			ps150.OverlayValues[103] = d103
			ps150.OverlayValues[104] = d104
			ps150.OverlayValues[105] = d105
			ps150.OverlayValues[106] = d106
			ps150.OverlayValues[107] = d107
			ps150.OverlayValues[108] = d108
			ps150.OverlayValues[109] = d109
			ps150.OverlayValues[110] = d110
			ps150.OverlayValues[111] = d111
			ps150.OverlayValues[112] = d112
			ps150.OverlayValues[113] = d113
			ps150.OverlayValues[114] = d114
			ps150.OverlayValues[115] = d115
			ps150.OverlayValues[116] = d116
			ps150.OverlayValues[117] = d117
			ps150.OverlayValues[118] = d118
			ps150.OverlayValues[119] = d119
			ps150.OverlayValues[120] = d120
			ps150.OverlayValues[121] = d121
			ps150.OverlayValues[122] = d122
			ps150.OverlayValues[123] = d123
			ps150.OverlayValues[124] = d124
			ps150.OverlayValues[125] = d125
			ps150.OverlayValues[126] = d126
			ps150.OverlayValues[127] = d127
			ps150.OverlayValues[128] = d128
			ps150.OverlayValues[129] = d129
			ps150.OverlayValues[130] = d130
			ps150.OverlayValues[131] = d131
			ps150.OverlayValues[132] = d132
			ps150.OverlayValues[133] = d133
			ps150.OverlayValues[134] = d134
			ps150.OverlayValues[135] = d135
			ps150.OverlayValues[136] = d136
			ps150.OverlayValues[137] = d137
			ps150.OverlayValues[138] = d138
			ps150.OverlayValues[139] = d139
			ps150.OverlayValues[140] = d140
			ps150.OverlayValues[141] = d141
			ps150.OverlayValues[142] = d142
			ps150.OverlayValues[143] = d143
			ps150.OverlayValues[144] = d144
			ps150.OverlayValues[145] = d145
			ps150.OverlayValues[146] = d146
			snap151 := d144
			alloc152 := ctx.SnapshotAllocState()
			if !bbs[21].Rendered {
				bbs[21].RenderPS(ps150)
			}
			ctx.RestoreAllocState(alloc152)
			d144 = snap151
			if !bbs[22].Rendered {
				return bbs[22].RenderPS(ps149)
			}
			return result
			ctx.FreeDesc(&d145)
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
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d1)
			var d153 scm.JITValueDesc
			if d1.Loc == scm.LocImm {
				d153 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d1.Imm.Int()) == uint64(0))}
			} else {
				r101 := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitCmpRegImm32(d1.Reg, 0)
				ctx.W.EmitSetcc(r101, scm.CcE)
				d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r101}
				ctx.BindReg(r101, &d153)
			}
			d154 = d153
			ctx.EnsureDesc(&d154)
			if d154.Loc != scm.LocImm && d154.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d154.Loc == scm.LocImm {
				if d154.Imm.Bool() {
			ps155 := scm.PhiState{General: ps.General}
			ps155.OverlayValues = make([]scm.JITValueDesc, 155)
			ps155.OverlayValues[0] = d0
			ps155.OverlayValues[1] = d1
			ps155.OverlayValues[2] = d2
			ps155.OverlayValues[3] = d3
			ps155.OverlayValues[4] = d4
			ps155.OverlayValues[5] = d5
			ps155.OverlayValues[6] = d6
			ps155.OverlayValues[7] = d7
			ps155.OverlayValues[8] = d8
			ps155.OverlayValues[9] = d9
			ps155.OverlayValues[10] = d10
			ps155.OverlayValues[11] = d11
			ps155.OverlayValues[12] = d12
			ps155.OverlayValues[13] = d13
			ps155.OverlayValues[14] = d14
			ps155.OverlayValues[20] = d20
			ps155.OverlayValues[21] = d21
			ps155.OverlayValues[22] = d22
			ps155.OverlayValues[24] = d24
			ps155.OverlayValues[25] = d25
			ps155.OverlayValues[27] = d27
			ps155.OverlayValues[28] = d28
			ps155.OverlayValues[29] = d29
			ps155.OverlayValues[32] = d32
			ps155.OverlayValues[34] = d34
			ps155.OverlayValues[35] = d35
			ps155.OverlayValues[36] = d36
			ps155.OverlayValues[38] = d38
			ps155.OverlayValues[39] = d39
			ps155.OverlayValues[40] = d40
			ps155.OverlayValues[41] = d41
			ps155.OverlayValues[42] = d42
			ps155.OverlayValues[43] = d43
			ps155.OverlayValues[44] = d44
			ps155.OverlayValues[46] = d46
			ps155.OverlayValues[47] = d47
			ps155.OverlayValues[48] = d48
			ps155.OverlayValues[49] = d49
			ps155.OverlayValues[50] = d50
			ps155.OverlayValues[51] = d51
			ps155.OverlayValues[52] = d52
			ps155.OverlayValues[53] = d53
			ps155.OverlayValues[54] = d54
			ps155.OverlayValues[55] = d55
			ps155.OverlayValues[56] = d56
			ps155.OverlayValues[57] = d57
			ps155.OverlayValues[58] = d58
			ps155.OverlayValues[59] = d59
			ps155.OverlayValues[60] = d60
			ps155.OverlayValues[61] = d61
			ps155.OverlayValues[62] = d62
			ps155.OverlayValues[63] = d63
			ps155.OverlayValues[64] = d64
			ps155.OverlayValues[65] = d65
			ps155.OverlayValues[66] = d66
			ps155.OverlayValues[67] = d67
			ps155.OverlayValues[68] = d68
			ps155.OverlayValues[69] = d69
			ps155.OverlayValues[70] = d70
			ps155.OverlayValues[71] = d71
			ps155.OverlayValues[72] = d72
			ps155.OverlayValues[73] = d73
			ps155.OverlayValues[74] = d74
			ps155.OverlayValues[75] = d75
			ps155.OverlayValues[76] = d76
			ps155.OverlayValues[77] = d77
			ps155.OverlayValues[78] = d78
			ps155.OverlayValues[79] = d79
			ps155.OverlayValues[80] = d80
			ps155.OverlayValues[81] = d81
			ps155.OverlayValues[82] = d82
			ps155.OverlayValues[83] = d83
			ps155.OverlayValues[84] = d84
			ps155.OverlayValues[85] = d85
			ps155.OverlayValues[86] = d86
			ps155.OverlayValues[87] = d87
			ps155.OverlayValues[88] = d88
			ps155.OverlayValues[89] = d89
			ps155.OverlayValues[90] = d90
			ps155.OverlayValues[91] = d91
			ps155.OverlayValues[92] = d92
			ps155.OverlayValues[93] = d93
			ps155.OverlayValues[94] = d94
			ps155.OverlayValues[95] = d95
			ps155.OverlayValues[102] = d102
			ps155.OverlayValues[103] = d103
			ps155.OverlayValues[104] = d104
			ps155.OverlayValues[105] = d105
			ps155.OverlayValues[106] = d106
			ps155.OverlayValues[107] = d107
			ps155.OverlayValues[108] = d108
			ps155.OverlayValues[109] = d109
			ps155.OverlayValues[110] = d110
			ps155.OverlayValues[111] = d111
			ps155.OverlayValues[112] = d112
			ps155.OverlayValues[113] = d113
			ps155.OverlayValues[114] = d114
			ps155.OverlayValues[115] = d115
			ps155.OverlayValues[116] = d116
			ps155.OverlayValues[117] = d117
			ps155.OverlayValues[118] = d118
			ps155.OverlayValues[119] = d119
			ps155.OverlayValues[120] = d120
			ps155.OverlayValues[121] = d121
			ps155.OverlayValues[122] = d122
			ps155.OverlayValues[123] = d123
			ps155.OverlayValues[124] = d124
			ps155.OverlayValues[125] = d125
			ps155.OverlayValues[126] = d126
			ps155.OverlayValues[127] = d127
			ps155.OverlayValues[128] = d128
			ps155.OverlayValues[129] = d129
			ps155.OverlayValues[130] = d130
			ps155.OverlayValues[131] = d131
			ps155.OverlayValues[132] = d132
			ps155.OverlayValues[133] = d133
			ps155.OverlayValues[134] = d134
			ps155.OverlayValues[135] = d135
			ps155.OverlayValues[136] = d136
			ps155.OverlayValues[137] = d137
			ps155.OverlayValues[138] = d138
			ps155.OverlayValues[139] = d139
			ps155.OverlayValues[140] = d140
			ps155.OverlayValues[141] = d141
			ps155.OverlayValues[142] = d142
			ps155.OverlayValues[143] = d143
			ps155.OverlayValues[144] = d144
			ps155.OverlayValues[145] = d145
			ps155.OverlayValues[146] = d146
			ps155.OverlayValues[153] = d153
			ps155.OverlayValues[154] = d154
					return bbs[10].RenderPS(ps155)
				}
			ps156 := scm.PhiState{General: ps.General}
			ps156.OverlayValues = make([]scm.JITValueDesc, 155)
			ps156.OverlayValues[0] = d0
			ps156.OverlayValues[1] = d1
			ps156.OverlayValues[2] = d2
			ps156.OverlayValues[3] = d3
			ps156.OverlayValues[4] = d4
			ps156.OverlayValues[5] = d5
			ps156.OverlayValues[6] = d6
			ps156.OverlayValues[7] = d7
			ps156.OverlayValues[8] = d8
			ps156.OverlayValues[9] = d9
			ps156.OverlayValues[10] = d10
			ps156.OverlayValues[11] = d11
			ps156.OverlayValues[12] = d12
			ps156.OverlayValues[13] = d13
			ps156.OverlayValues[14] = d14
			ps156.OverlayValues[20] = d20
			ps156.OverlayValues[21] = d21
			ps156.OverlayValues[22] = d22
			ps156.OverlayValues[24] = d24
			ps156.OverlayValues[25] = d25
			ps156.OverlayValues[27] = d27
			ps156.OverlayValues[28] = d28
			ps156.OverlayValues[29] = d29
			ps156.OverlayValues[32] = d32
			ps156.OverlayValues[34] = d34
			ps156.OverlayValues[35] = d35
			ps156.OverlayValues[36] = d36
			ps156.OverlayValues[38] = d38
			ps156.OverlayValues[39] = d39
			ps156.OverlayValues[40] = d40
			ps156.OverlayValues[41] = d41
			ps156.OverlayValues[42] = d42
			ps156.OverlayValues[43] = d43
			ps156.OverlayValues[44] = d44
			ps156.OverlayValues[46] = d46
			ps156.OverlayValues[47] = d47
			ps156.OverlayValues[48] = d48
			ps156.OverlayValues[49] = d49
			ps156.OverlayValues[50] = d50
			ps156.OverlayValues[51] = d51
			ps156.OverlayValues[52] = d52
			ps156.OverlayValues[53] = d53
			ps156.OverlayValues[54] = d54
			ps156.OverlayValues[55] = d55
			ps156.OverlayValues[56] = d56
			ps156.OverlayValues[57] = d57
			ps156.OverlayValues[58] = d58
			ps156.OverlayValues[59] = d59
			ps156.OverlayValues[60] = d60
			ps156.OverlayValues[61] = d61
			ps156.OverlayValues[62] = d62
			ps156.OverlayValues[63] = d63
			ps156.OverlayValues[64] = d64
			ps156.OverlayValues[65] = d65
			ps156.OverlayValues[66] = d66
			ps156.OverlayValues[67] = d67
			ps156.OverlayValues[68] = d68
			ps156.OverlayValues[69] = d69
			ps156.OverlayValues[70] = d70
			ps156.OverlayValues[71] = d71
			ps156.OverlayValues[72] = d72
			ps156.OverlayValues[73] = d73
			ps156.OverlayValues[74] = d74
			ps156.OverlayValues[75] = d75
			ps156.OverlayValues[76] = d76
			ps156.OverlayValues[77] = d77
			ps156.OverlayValues[78] = d78
			ps156.OverlayValues[79] = d79
			ps156.OverlayValues[80] = d80
			ps156.OverlayValues[81] = d81
			ps156.OverlayValues[82] = d82
			ps156.OverlayValues[83] = d83
			ps156.OverlayValues[84] = d84
			ps156.OverlayValues[85] = d85
			ps156.OverlayValues[86] = d86
			ps156.OverlayValues[87] = d87
			ps156.OverlayValues[88] = d88
			ps156.OverlayValues[89] = d89
			ps156.OverlayValues[90] = d90
			ps156.OverlayValues[91] = d91
			ps156.OverlayValues[92] = d92
			ps156.OverlayValues[93] = d93
			ps156.OverlayValues[94] = d94
			ps156.OverlayValues[95] = d95
			ps156.OverlayValues[102] = d102
			ps156.OverlayValues[103] = d103
			ps156.OverlayValues[104] = d104
			ps156.OverlayValues[105] = d105
			ps156.OverlayValues[106] = d106
			ps156.OverlayValues[107] = d107
			ps156.OverlayValues[108] = d108
			ps156.OverlayValues[109] = d109
			ps156.OverlayValues[110] = d110
			ps156.OverlayValues[111] = d111
			ps156.OverlayValues[112] = d112
			ps156.OverlayValues[113] = d113
			ps156.OverlayValues[114] = d114
			ps156.OverlayValues[115] = d115
			ps156.OverlayValues[116] = d116
			ps156.OverlayValues[117] = d117
			ps156.OverlayValues[118] = d118
			ps156.OverlayValues[119] = d119
			ps156.OverlayValues[120] = d120
			ps156.OverlayValues[121] = d121
			ps156.OverlayValues[122] = d122
			ps156.OverlayValues[123] = d123
			ps156.OverlayValues[124] = d124
			ps156.OverlayValues[125] = d125
			ps156.OverlayValues[126] = d126
			ps156.OverlayValues[127] = d127
			ps156.OverlayValues[128] = d128
			ps156.OverlayValues[129] = d129
			ps156.OverlayValues[130] = d130
			ps156.OverlayValues[131] = d131
			ps156.OverlayValues[132] = d132
			ps156.OverlayValues[133] = d133
			ps156.OverlayValues[134] = d134
			ps156.OverlayValues[135] = d135
			ps156.OverlayValues[136] = d136
			ps156.OverlayValues[137] = d137
			ps156.OverlayValues[138] = d138
			ps156.OverlayValues[139] = d139
			ps156.OverlayValues[140] = d140
			ps156.OverlayValues[141] = d141
			ps156.OverlayValues[142] = d142
			ps156.OverlayValues[143] = d143
			ps156.OverlayValues[144] = d144
			ps156.OverlayValues[145] = d145
			ps156.OverlayValues[146] = d146
			ps156.OverlayValues[153] = d153
			ps156.OverlayValues[154] = d154
				return bbs[11].RenderPS(ps156)
			}
			lbl48 := ctx.W.ReserveLabel()
			lbl49 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d154.Reg, 0)
			ctx.W.EmitJcc(scm.CcNE, lbl48)
			ctx.W.EmitJmp(lbl49)
			ctx.W.MarkLabel(lbl48)
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl49)
			ctx.W.EmitJmp(lbl12)
			ps157 := scm.PhiState{General: true}
			ps157.OverlayValues = make([]scm.JITValueDesc, 155)
			ps157.OverlayValues[0] = d0
			ps157.OverlayValues[1] = d1
			ps157.OverlayValues[2] = d2
			ps157.OverlayValues[3] = d3
			ps157.OverlayValues[4] = d4
			ps157.OverlayValues[5] = d5
			ps157.OverlayValues[6] = d6
			ps157.OverlayValues[7] = d7
			ps157.OverlayValues[8] = d8
			ps157.OverlayValues[9] = d9
			ps157.OverlayValues[10] = d10
			ps157.OverlayValues[11] = d11
			ps157.OverlayValues[12] = d12
			ps157.OverlayValues[13] = d13
			ps157.OverlayValues[14] = d14
			ps157.OverlayValues[20] = d20
			ps157.OverlayValues[21] = d21
			ps157.OverlayValues[22] = d22
			ps157.OverlayValues[24] = d24
			ps157.OverlayValues[25] = d25
			ps157.OverlayValues[27] = d27
			ps157.OverlayValues[28] = d28
			ps157.OverlayValues[29] = d29
			ps157.OverlayValues[32] = d32
			ps157.OverlayValues[34] = d34
			ps157.OverlayValues[35] = d35
			ps157.OverlayValues[36] = d36
			ps157.OverlayValues[38] = d38
			ps157.OverlayValues[39] = d39
			ps157.OverlayValues[40] = d40
			ps157.OverlayValues[41] = d41
			ps157.OverlayValues[42] = d42
			ps157.OverlayValues[43] = d43
			ps157.OverlayValues[44] = d44
			ps157.OverlayValues[46] = d46
			ps157.OverlayValues[47] = d47
			ps157.OverlayValues[48] = d48
			ps157.OverlayValues[49] = d49
			ps157.OverlayValues[50] = d50
			ps157.OverlayValues[51] = d51
			ps157.OverlayValues[52] = d52
			ps157.OverlayValues[53] = d53
			ps157.OverlayValues[54] = d54
			ps157.OverlayValues[55] = d55
			ps157.OverlayValues[56] = d56
			ps157.OverlayValues[57] = d57
			ps157.OverlayValues[58] = d58
			ps157.OverlayValues[59] = d59
			ps157.OverlayValues[60] = d60
			ps157.OverlayValues[61] = d61
			ps157.OverlayValues[62] = d62
			ps157.OverlayValues[63] = d63
			ps157.OverlayValues[64] = d64
			ps157.OverlayValues[65] = d65
			ps157.OverlayValues[66] = d66
			ps157.OverlayValues[67] = d67
			ps157.OverlayValues[68] = d68
			ps157.OverlayValues[69] = d69
			ps157.OverlayValues[70] = d70
			ps157.OverlayValues[71] = d71
			ps157.OverlayValues[72] = d72
			ps157.OverlayValues[73] = d73
			ps157.OverlayValues[74] = d74
			ps157.OverlayValues[75] = d75
			ps157.OverlayValues[76] = d76
			ps157.OverlayValues[77] = d77
			ps157.OverlayValues[78] = d78
			ps157.OverlayValues[79] = d79
			ps157.OverlayValues[80] = d80
			ps157.OverlayValues[81] = d81
			ps157.OverlayValues[82] = d82
			ps157.OverlayValues[83] = d83
			ps157.OverlayValues[84] = d84
			ps157.OverlayValues[85] = d85
			ps157.OverlayValues[86] = d86
			ps157.OverlayValues[87] = d87
			ps157.OverlayValues[88] = d88
			ps157.OverlayValues[89] = d89
			ps157.OverlayValues[90] = d90
			ps157.OverlayValues[91] = d91
			ps157.OverlayValues[92] = d92
			ps157.OverlayValues[93] = d93
			ps157.OverlayValues[94] = d94
			ps157.OverlayValues[95] = d95
			ps157.OverlayValues[102] = d102
			ps157.OverlayValues[103] = d103
			ps157.OverlayValues[104] = d104
			ps157.OverlayValues[105] = d105
			ps157.OverlayValues[106] = d106
			ps157.OverlayValues[107] = d107
			ps157.OverlayValues[108] = d108
			ps157.OverlayValues[109] = d109
			ps157.OverlayValues[110] = d110
			ps157.OverlayValues[111] = d111
			ps157.OverlayValues[112] = d112
			ps157.OverlayValues[113] = d113
			ps157.OverlayValues[114] = d114
			ps157.OverlayValues[115] = d115
			ps157.OverlayValues[116] = d116
			ps157.OverlayValues[117] = d117
			ps157.OverlayValues[118] = d118
			ps157.OverlayValues[119] = d119
			ps157.OverlayValues[120] = d120
			ps157.OverlayValues[121] = d121
			ps157.OverlayValues[122] = d122
			ps157.OverlayValues[123] = d123
			ps157.OverlayValues[124] = d124
			ps157.OverlayValues[125] = d125
			ps157.OverlayValues[126] = d126
			ps157.OverlayValues[127] = d127
			ps157.OverlayValues[128] = d128
			ps157.OverlayValues[129] = d129
			ps157.OverlayValues[130] = d130
			ps157.OverlayValues[131] = d131
			ps157.OverlayValues[132] = d132
			ps157.OverlayValues[133] = d133
			ps157.OverlayValues[134] = d134
			ps157.OverlayValues[135] = d135
			ps157.OverlayValues[136] = d136
			ps157.OverlayValues[137] = d137
			ps157.OverlayValues[138] = d138
			ps157.OverlayValues[139] = d139
			ps157.OverlayValues[140] = d140
			ps157.OverlayValues[141] = d141
			ps157.OverlayValues[142] = d142
			ps157.OverlayValues[143] = d143
			ps157.OverlayValues[144] = d144
			ps157.OverlayValues[145] = d145
			ps157.OverlayValues[146] = d146
			ps157.OverlayValues[153] = d153
			ps157.OverlayValues[154] = d154
			ps158 := scm.PhiState{General: true}
			ps158.OverlayValues = make([]scm.JITValueDesc, 155)
			ps158.OverlayValues[0] = d0
			ps158.OverlayValues[1] = d1
			ps158.OverlayValues[2] = d2
			ps158.OverlayValues[3] = d3
			ps158.OverlayValues[4] = d4
			ps158.OverlayValues[5] = d5
			ps158.OverlayValues[6] = d6
			ps158.OverlayValues[7] = d7
			ps158.OverlayValues[8] = d8
			ps158.OverlayValues[9] = d9
			ps158.OverlayValues[10] = d10
			ps158.OverlayValues[11] = d11
			ps158.OverlayValues[12] = d12
			ps158.OverlayValues[13] = d13
			ps158.OverlayValues[14] = d14
			ps158.OverlayValues[20] = d20
			ps158.OverlayValues[21] = d21
			ps158.OverlayValues[22] = d22
			ps158.OverlayValues[24] = d24
			ps158.OverlayValues[25] = d25
			ps158.OverlayValues[27] = d27
			ps158.OverlayValues[28] = d28
			ps158.OverlayValues[29] = d29
			ps158.OverlayValues[32] = d32
			ps158.OverlayValues[34] = d34
			ps158.OverlayValues[35] = d35
			ps158.OverlayValues[36] = d36
			ps158.OverlayValues[38] = d38
			ps158.OverlayValues[39] = d39
			ps158.OverlayValues[40] = d40
			ps158.OverlayValues[41] = d41
			ps158.OverlayValues[42] = d42
			ps158.OverlayValues[43] = d43
			ps158.OverlayValues[44] = d44
			ps158.OverlayValues[46] = d46
			ps158.OverlayValues[47] = d47
			ps158.OverlayValues[48] = d48
			ps158.OverlayValues[49] = d49
			ps158.OverlayValues[50] = d50
			ps158.OverlayValues[51] = d51
			ps158.OverlayValues[52] = d52
			ps158.OverlayValues[53] = d53
			ps158.OverlayValues[54] = d54
			ps158.OverlayValues[55] = d55
			ps158.OverlayValues[56] = d56
			ps158.OverlayValues[57] = d57
			ps158.OverlayValues[58] = d58
			ps158.OverlayValues[59] = d59
			ps158.OverlayValues[60] = d60
			ps158.OverlayValues[61] = d61
			ps158.OverlayValues[62] = d62
			ps158.OverlayValues[63] = d63
			ps158.OverlayValues[64] = d64
			ps158.OverlayValues[65] = d65
			ps158.OverlayValues[66] = d66
			ps158.OverlayValues[67] = d67
			ps158.OverlayValues[68] = d68
			ps158.OverlayValues[69] = d69
			ps158.OverlayValues[70] = d70
			ps158.OverlayValues[71] = d71
			ps158.OverlayValues[72] = d72
			ps158.OverlayValues[73] = d73
			ps158.OverlayValues[74] = d74
			ps158.OverlayValues[75] = d75
			ps158.OverlayValues[76] = d76
			ps158.OverlayValues[77] = d77
			ps158.OverlayValues[78] = d78
			ps158.OverlayValues[79] = d79
			ps158.OverlayValues[80] = d80
			ps158.OverlayValues[81] = d81
			ps158.OverlayValues[82] = d82
			ps158.OverlayValues[83] = d83
			ps158.OverlayValues[84] = d84
			ps158.OverlayValues[85] = d85
			ps158.OverlayValues[86] = d86
			ps158.OverlayValues[87] = d87
			ps158.OverlayValues[88] = d88
			ps158.OverlayValues[89] = d89
			ps158.OverlayValues[90] = d90
			ps158.OverlayValues[91] = d91
			ps158.OverlayValues[92] = d92
			ps158.OverlayValues[93] = d93
			ps158.OverlayValues[94] = d94
			ps158.OverlayValues[95] = d95
			ps158.OverlayValues[102] = d102
			ps158.OverlayValues[103] = d103
			ps158.OverlayValues[104] = d104
			ps158.OverlayValues[105] = d105
			ps158.OverlayValues[106] = d106
			ps158.OverlayValues[107] = d107
			ps158.OverlayValues[108] = d108
			ps158.OverlayValues[109] = d109
			ps158.OverlayValues[110] = d110
			ps158.OverlayValues[111] = d111
			ps158.OverlayValues[112] = d112
			ps158.OverlayValues[113] = d113
			ps158.OverlayValues[114] = d114
			ps158.OverlayValues[115] = d115
			ps158.OverlayValues[116] = d116
			ps158.OverlayValues[117] = d117
			ps158.OverlayValues[118] = d118
			ps158.OverlayValues[119] = d119
			ps158.OverlayValues[120] = d120
			ps158.OverlayValues[121] = d121
			ps158.OverlayValues[122] = d122
			ps158.OverlayValues[123] = d123
			ps158.OverlayValues[124] = d124
			ps158.OverlayValues[125] = d125
			ps158.OverlayValues[126] = d126
			ps158.OverlayValues[127] = d127
			ps158.OverlayValues[128] = d128
			ps158.OverlayValues[129] = d129
			ps158.OverlayValues[130] = d130
			ps158.OverlayValues[131] = d131
			ps158.OverlayValues[132] = d132
			ps158.OverlayValues[133] = d133
			ps158.OverlayValues[134] = d134
			ps158.OverlayValues[135] = d135
			ps158.OverlayValues[136] = d136
			ps158.OverlayValues[137] = d137
			ps158.OverlayValues[138] = d138
			ps158.OverlayValues[139] = d139
			ps158.OverlayValues[140] = d140
			ps158.OverlayValues[141] = d141
			ps158.OverlayValues[142] = d142
			ps158.OverlayValues[143] = d143
			ps158.OverlayValues[144] = d144
			ps158.OverlayValues[145] = d145
			ps158.OverlayValues[146] = d146
			ps158.OverlayValues[153] = d153
			ps158.OverlayValues[154] = d154
			alloc159 := ctx.SnapshotAllocState()
			if !bbs[11].Rendered {
				bbs[11].RenderPS(ps158)
			}
			ctx.RestoreAllocState(alloc159)
			if !bbs[10].Rendered {
				return bbs[10].RenderPS(ps157)
			}
			return result
			ctx.FreeDesc(&d153)
			return result
			}
			bbs[8].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[8].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
						d160 := ps.PhiValues[0]
						ctx.EnsureDesc(&d160)
						ctx.EmitStoreToStack(d160, 40)
					}
					if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
						d161 := ps.PhiValues[1]
						ctx.EnsureDesc(&d161)
						ctx.EmitStoreToStack(d161, 48)
					}
					if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
						d162 := ps.PhiValues[2]
						ctx.EnsureDesc(&d162)
						ctx.EmitStoreToStack(d162, 56)
					}
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
			d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
				d154 = ps.OverlayValues[154]
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
			var d163 scm.JITValueDesc
			if d6.Loc == scm.LocImm && d7.Loc == scm.LocImm {
				d163 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d6.Imm.Int()) == uint64(d7.Imm.Int()))}
			} else if d7.Loc == scm.LocImm {
				r102 := ctx.AllocRegExcept(d6.Reg)
				if d7.Imm.Int() >= -2147483648 && d7.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d6.Reg, int32(d7.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d7.Imm.Int()))
					ctx.W.EmitCmpInt64(d6.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r102, scm.CcE)
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r102}
				ctx.BindReg(r102, &d163)
			} else if d6.Loc == scm.LocImm {
				r103 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d6.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d7.Reg)
				ctx.W.EmitSetcc(r103, scm.CcE)
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r103}
				ctx.BindReg(r103, &d163)
			} else {
				r104 := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitCmpInt64(d6.Reg, d7.Reg)
				ctx.W.EmitSetcc(r104, scm.CcE)
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r104}
				ctx.BindReg(r104, &d163)
			}
			d164 = d163
			ctx.EnsureDesc(&d164)
			if d164.Loc != scm.LocImm && d164.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d164.Loc == scm.LocImm {
				if d164.Imm.Bool() {
			d165 = d6
			if d165.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d165)
			d166 = d165
			if d166.Loc == scm.LocImm {
				d166 = scm.JITValueDesc{Loc: scm.LocImm, Type: d166.Type, Imm: scm.NewInt(int64(uint64(d166.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d166.Reg, 32)
				ctx.W.EmitShrRegImm8(d166.Reg, 32)
			}
			ctx.EmitStoreToStack(d166, 32)
			ps167 := scm.PhiState{General: ps.General}
			ps167.OverlayValues = make([]scm.JITValueDesc, 167)
			ps167.OverlayValues[0] = d0
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
			ps167.OverlayValues[20] = d20
			ps167.OverlayValues[21] = d21
			ps167.OverlayValues[22] = d22
			ps167.OverlayValues[24] = d24
			ps167.OverlayValues[25] = d25
			ps167.OverlayValues[27] = d27
			ps167.OverlayValues[28] = d28
			ps167.OverlayValues[29] = d29
			ps167.OverlayValues[32] = d32
			ps167.OverlayValues[34] = d34
			ps167.OverlayValues[35] = d35
			ps167.OverlayValues[36] = d36
			ps167.OverlayValues[38] = d38
			ps167.OverlayValues[39] = d39
			ps167.OverlayValues[40] = d40
			ps167.OverlayValues[41] = d41
			ps167.OverlayValues[42] = d42
			ps167.OverlayValues[43] = d43
			ps167.OverlayValues[44] = d44
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
			ps167.OverlayValues[60] = d60
			ps167.OverlayValues[61] = d61
			ps167.OverlayValues[62] = d62
			ps167.OverlayValues[63] = d63
			ps167.OverlayValues[64] = d64
			ps167.OverlayValues[65] = d65
			ps167.OverlayValues[66] = d66
			ps167.OverlayValues[67] = d67
			ps167.OverlayValues[68] = d68
			ps167.OverlayValues[69] = d69
			ps167.OverlayValues[70] = d70
			ps167.OverlayValues[71] = d71
			ps167.OverlayValues[72] = d72
			ps167.OverlayValues[73] = d73
			ps167.OverlayValues[74] = d74
			ps167.OverlayValues[75] = d75
			ps167.OverlayValues[76] = d76
			ps167.OverlayValues[77] = d77
			ps167.OverlayValues[78] = d78
			ps167.OverlayValues[79] = d79
			ps167.OverlayValues[80] = d80
			ps167.OverlayValues[81] = d81
			ps167.OverlayValues[82] = d82
			ps167.OverlayValues[83] = d83
			ps167.OverlayValues[84] = d84
			ps167.OverlayValues[85] = d85
			ps167.OverlayValues[86] = d86
			ps167.OverlayValues[87] = d87
			ps167.OverlayValues[88] = d88
			ps167.OverlayValues[89] = d89
			ps167.OverlayValues[90] = d90
			ps167.OverlayValues[91] = d91
			ps167.OverlayValues[92] = d92
			ps167.OverlayValues[93] = d93
			ps167.OverlayValues[94] = d94
			ps167.OverlayValues[95] = d95
			ps167.OverlayValues[102] = d102
			ps167.OverlayValues[103] = d103
			ps167.OverlayValues[104] = d104
			ps167.OverlayValues[105] = d105
			ps167.OverlayValues[106] = d106
			ps167.OverlayValues[107] = d107
			ps167.OverlayValues[108] = d108
			ps167.OverlayValues[109] = d109
			ps167.OverlayValues[110] = d110
			ps167.OverlayValues[111] = d111
			ps167.OverlayValues[112] = d112
			ps167.OverlayValues[113] = d113
			ps167.OverlayValues[114] = d114
			ps167.OverlayValues[115] = d115
			ps167.OverlayValues[116] = d116
			ps167.OverlayValues[117] = d117
			ps167.OverlayValues[118] = d118
			ps167.OverlayValues[119] = d119
			ps167.OverlayValues[120] = d120
			ps167.OverlayValues[121] = d121
			ps167.OverlayValues[122] = d122
			ps167.OverlayValues[123] = d123
			ps167.OverlayValues[124] = d124
			ps167.OverlayValues[125] = d125
			ps167.OverlayValues[126] = d126
			ps167.OverlayValues[127] = d127
			ps167.OverlayValues[128] = d128
			ps167.OverlayValues[129] = d129
			ps167.OverlayValues[130] = d130
			ps167.OverlayValues[131] = d131
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
			ps167.OverlayValues[153] = d153
			ps167.OverlayValues[154] = d154
			ps167.OverlayValues[160] = d160
			ps167.OverlayValues[161] = d161
			ps167.OverlayValues[162] = d162
			ps167.OverlayValues[163] = d163
			ps167.OverlayValues[164] = d164
			ps167.OverlayValues[165] = d165
			ps167.OverlayValues[166] = d166
			ps167.PhiValues = make([]scm.JITValueDesc, 1)
			d168 = d6
			ps167.PhiValues[0] = d168
					return bbs[6].RenderPS(ps167)
				}
			ps169 := scm.PhiState{General: ps.General}
			ps169.OverlayValues = make([]scm.JITValueDesc, 169)
			ps169.OverlayValues[0] = d0
			ps169.OverlayValues[1] = d1
			ps169.OverlayValues[2] = d2
			ps169.OverlayValues[3] = d3
			ps169.OverlayValues[4] = d4
			ps169.OverlayValues[5] = d5
			ps169.OverlayValues[6] = d6
			ps169.OverlayValues[7] = d7
			ps169.OverlayValues[8] = d8
			ps169.OverlayValues[9] = d9
			ps169.OverlayValues[10] = d10
			ps169.OverlayValues[11] = d11
			ps169.OverlayValues[12] = d12
			ps169.OverlayValues[13] = d13
			ps169.OverlayValues[14] = d14
			ps169.OverlayValues[20] = d20
			ps169.OverlayValues[21] = d21
			ps169.OverlayValues[22] = d22
			ps169.OverlayValues[24] = d24
			ps169.OverlayValues[25] = d25
			ps169.OverlayValues[27] = d27
			ps169.OverlayValues[28] = d28
			ps169.OverlayValues[29] = d29
			ps169.OverlayValues[32] = d32
			ps169.OverlayValues[34] = d34
			ps169.OverlayValues[35] = d35
			ps169.OverlayValues[36] = d36
			ps169.OverlayValues[38] = d38
			ps169.OverlayValues[39] = d39
			ps169.OverlayValues[40] = d40
			ps169.OverlayValues[41] = d41
			ps169.OverlayValues[42] = d42
			ps169.OverlayValues[43] = d43
			ps169.OverlayValues[44] = d44
			ps169.OverlayValues[46] = d46
			ps169.OverlayValues[47] = d47
			ps169.OverlayValues[48] = d48
			ps169.OverlayValues[49] = d49
			ps169.OverlayValues[50] = d50
			ps169.OverlayValues[51] = d51
			ps169.OverlayValues[52] = d52
			ps169.OverlayValues[53] = d53
			ps169.OverlayValues[54] = d54
			ps169.OverlayValues[55] = d55
			ps169.OverlayValues[56] = d56
			ps169.OverlayValues[57] = d57
			ps169.OverlayValues[58] = d58
			ps169.OverlayValues[59] = d59
			ps169.OverlayValues[60] = d60
			ps169.OverlayValues[61] = d61
			ps169.OverlayValues[62] = d62
			ps169.OverlayValues[63] = d63
			ps169.OverlayValues[64] = d64
			ps169.OverlayValues[65] = d65
			ps169.OverlayValues[66] = d66
			ps169.OverlayValues[67] = d67
			ps169.OverlayValues[68] = d68
			ps169.OverlayValues[69] = d69
			ps169.OverlayValues[70] = d70
			ps169.OverlayValues[71] = d71
			ps169.OverlayValues[72] = d72
			ps169.OverlayValues[73] = d73
			ps169.OverlayValues[74] = d74
			ps169.OverlayValues[75] = d75
			ps169.OverlayValues[76] = d76
			ps169.OverlayValues[77] = d77
			ps169.OverlayValues[78] = d78
			ps169.OverlayValues[79] = d79
			ps169.OverlayValues[80] = d80
			ps169.OverlayValues[81] = d81
			ps169.OverlayValues[82] = d82
			ps169.OverlayValues[83] = d83
			ps169.OverlayValues[84] = d84
			ps169.OverlayValues[85] = d85
			ps169.OverlayValues[86] = d86
			ps169.OverlayValues[87] = d87
			ps169.OverlayValues[88] = d88
			ps169.OverlayValues[89] = d89
			ps169.OverlayValues[90] = d90
			ps169.OverlayValues[91] = d91
			ps169.OverlayValues[92] = d92
			ps169.OverlayValues[93] = d93
			ps169.OverlayValues[94] = d94
			ps169.OverlayValues[95] = d95
			ps169.OverlayValues[102] = d102
			ps169.OverlayValues[103] = d103
			ps169.OverlayValues[104] = d104
			ps169.OverlayValues[105] = d105
			ps169.OverlayValues[106] = d106
			ps169.OverlayValues[107] = d107
			ps169.OverlayValues[108] = d108
			ps169.OverlayValues[109] = d109
			ps169.OverlayValues[110] = d110
			ps169.OverlayValues[111] = d111
			ps169.OverlayValues[112] = d112
			ps169.OverlayValues[113] = d113
			ps169.OverlayValues[114] = d114
			ps169.OverlayValues[115] = d115
			ps169.OverlayValues[116] = d116
			ps169.OverlayValues[117] = d117
			ps169.OverlayValues[118] = d118
			ps169.OverlayValues[119] = d119
			ps169.OverlayValues[120] = d120
			ps169.OverlayValues[121] = d121
			ps169.OverlayValues[122] = d122
			ps169.OverlayValues[123] = d123
			ps169.OverlayValues[124] = d124
			ps169.OverlayValues[125] = d125
			ps169.OverlayValues[126] = d126
			ps169.OverlayValues[127] = d127
			ps169.OverlayValues[128] = d128
			ps169.OverlayValues[129] = d129
			ps169.OverlayValues[130] = d130
			ps169.OverlayValues[131] = d131
			ps169.OverlayValues[132] = d132
			ps169.OverlayValues[133] = d133
			ps169.OverlayValues[134] = d134
			ps169.OverlayValues[135] = d135
			ps169.OverlayValues[136] = d136
			ps169.OverlayValues[137] = d137
			ps169.OverlayValues[138] = d138
			ps169.OverlayValues[139] = d139
			ps169.OverlayValues[140] = d140
			ps169.OverlayValues[141] = d141
			ps169.OverlayValues[142] = d142
			ps169.OverlayValues[143] = d143
			ps169.OverlayValues[144] = d144
			ps169.OverlayValues[145] = d145
			ps169.OverlayValues[146] = d146
			ps169.OverlayValues[153] = d153
			ps169.OverlayValues[154] = d154
			ps169.OverlayValues[160] = d160
			ps169.OverlayValues[161] = d161
			ps169.OverlayValues[162] = d162
			ps169.OverlayValues[163] = d163
			ps169.OverlayValues[164] = d164
			ps169.OverlayValues[165] = d165
			ps169.OverlayValues[166] = d166
			ps169.OverlayValues[168] = d168
				return bbs[13].RenderPS(ps169)
			}
			lbl50 := ctx.W.ReserveLabel()
			lbl51 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d164.Reg, 0)
			ctx.W.EmitJcc(scm.CcNE, lbl50)
			ctx.W.EmitJmp(lbl51)
			ctx.W.MarkLabel(lbl50)
			d170 = d6
			if d170.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d170)
			d171 = d170
			if d171.Loc == scm.LocImm {
				d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: d171.Type, Imm: scm.NewInt(int64(uint64(d171.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d171.Reg, 32)
				ctx.W.EmitShrRegImm8(d171.Reg, 32)
			}
			ctx.EmitStoreToStack(d171, 32)
			ctx.W.EmitJmp(lbl7)
			ctx.W.MarkLabel(lbl51)
			ctx.W.EmitJmp(lbl14)
			ps172 := scm.PhiState{General: true}
			ps172.OverlayValues = make([]scm.JITValueDesc, 172)
			ps172.OverlayValues[0] = d0
			ps172.OverlayValues[1] = d1
			ps172.OverlayValues[2] = d2
			ps172.OverlayValues[3] = d3
			ps172.OverlayValues[4] = d4
			ps172.OverlayValues[5] = d5
			ps172.OverlayValues[6] = d6
			ps172.OverlayValues[7] = d7
			ps172.OverlayValues[8] = d8
			ps172.OverlayValues[9] = d9
			ps172.OverlayValues[10] = d10
			ps172.OverlayValues[11] = d11
			ps172.OverlayValues[12] = d12
			ps172.OverlayValues[13] = d13
			ps172.OverlayValues[14] = d14
			ps172.OverlayValues[20] = d20
			ps172.OverlayValues[21] = d21
			ps172.OverlayValues[22] = d22
			ps172.OverlayValues[24] = d24
			ps172.OverlayValues[25] = d25
			ps172.OverlayValues[27] = d27
			ps172.OverlayValues[28] = d28
			ps172.OverlayValues[29] = d29
			ps172.OverlayValues[32] = d32
			ps172.OverlayValues[34] = d34
			ps172.OverlayValues[35] = d35
			ps172.OverlayValues[36] = d36
			ps172.OverlayValues[38] = d38
			ps172.OverlayValues[39] = d39
			ps172.OverlayValues[40] = d40
			ps172.OverlayValues[41] = d41
			ps172.OverlayValues[42] = d42
			ps172.OverlayValues[43] = d43
			ps172.OverlayValues[44] = d44
			ps172.OverlayValues[46] = d46
			ps172.OverlayValues[47] = d47
			ps172.OverlayValues[48] = d48
			ps172.OverlayValues[49] = d49
			ps172.OverlayValues[50] = d50
			ps172.OverlayValues[51] = d51
			ps172.OverlayValues[52] = d52
			ps172.OverlayValues[53] = d53
			ps172.OverlayValues[54] = d54
			ps172.OverlayValues[55] = d55
			ps172.OverlayValues[56] = d56
			ps172.OverlayValues[57] = d57
			ps172.OverlayValues[58] = d58
			ps172.OverlayValues[59] = d59
			ps172.OverlayValues[60] = d60
			ps172.OverlayValues[61] = d61
			ps172.OverlayValues[62] = d62
			ps172.OverlayValues[63] = d63
			ps172.OverlayValues[64] = d64
			ps172.OverlayValues[65] = d65
			ps172.OverlayValues[66] = d66
			ps172.OverlayValues[67] = d67
			ps172.OverlayValues[68] = d68
			ps172.OverlayValues[69] = d69
			ps172.OverlayValues[70] = d70
			ps172.OverlayValues[71] = d71
			ps172.OverlayValues[72] = d72
			ps172.OverlayValues[73] = d73
			ps172.OverlayValues[74] = d74
			ps172.OverlayValues[75] = d75
			ps172.OverlayValues[76] = d76
			ps172.OverlayValues[77] = d77
			ps172.OverlayValues[78] = d78
			ps172.OverlayValues[79] = d79
			ps172.OverlayValues[80] = d80
			ps172.OverlayValues[81] = d81
			ps172.OverlayValues[82] = d82
			ps172.OverlayValues[83] = d83
			ps172.OverlayValues[84] = d84
			ps172.OverlayValues[85] = d85
			ps172.OverlayValues[86] = d86
			ps172.OverlayValues[87] = d87
			ps172.OverlayValues[88] = d88
			ps172.OverlayValues[89] = d89
			ps172.OverlayValues[90] = d90
			ps172.OverlayValues[91] = d91
			ps172.OverlayValues[92] = d92
			ps172.OverlayValues[93] = d93
			ps172.OverlayValues[94] = d94
			ps172.OverlayValues[95] = d95
			ps172.OverlayValues[102] = d102
			ps172.OverlayValues[103] = d103
			ps172.OverlayValues[104] = d104
			ps172.OverlayValues[105] = d105
			ps172.OverlayValues[106] = d106
			ps172.OverlayValues[107] = d107
			ps172.OverlayValues[108] = d108
			ps172.OverlayValues[109] = d109
			ps172.OverlayValues[110] = d110
			ps172.OverlayValues[111] = d111
			ps172.OverlayValues[112] = d112
			ps172.OverlayValues[113] = d113
			ps172.OverlayValues[114] = d114
			ps172.OverlayValues[115] = d115
			ps172.OverlayValues[116] = d116
			ps172.OverlayValues[117] = d117
			ps172.OverlayValues[118] = d118
			ps172.OverlayValues[119] = d119
			ps172.OverlayValues[120] = d120
			ps172.OverlayValues[121] = d121
			ps172.OverlayValues[122] = d122
			ps172.OverlayValues[123] = d123
			ps172.OverlayValues[124] = d124
			ps172.OverlayValues[125] = d125
			ps172.OverlayValues[126] = d126
			ps172.OverlayValues[127] = d127
			ps172.OverlayValues[128] = d128
			ps172.OverlayValues[129] = d129
			ps172.OverlayValues[130] = d130
			ps172.OverlayValues[131] = d131
			ps172.OverlayValues[132] = d132
			ps172.OverlayValues[133] = d133
			ps172.OverlayValues[134] = d134
			ps172.OverlayValues[135] = d135
			ps172.OverlayValues[136] = d136
			ps172.OverlayValues[137] = d137
			ps172.OverlayValues[138] = d138
			ps172.OverlayValues[139] = d139
			ps172.OverlayValues[140] = d140
			ps172.OverlayValues[141] = d141
			ps172.OverlayValues[142] = d142
			ps172.OverlayValues[143] = d143
			ps172.OverlayValues[144] = d144
			ps172.OverlayValues[145] = d145
			ps172.OverlayValues[146] = d146
			ps172.OverlayValues[153] = d153
			ps172.OverlayValues[154] = d154
			ps172.OverlayValues[160] = d160
			ps172.OverlayValues[161] = d161
			ps172.OverlayValues[162] = d162
			ps172.OverlayValues[163] = d163
			ps172.OverlayValues[164] = d164
			ps172.OverlayValues[165] = d165
			ps172.OverlayValues[166] = d166
			ps172.OverlayValues[168] = d168
			ps172.OverlayValues[170] = d170
			ps172.OverlayValues[171] = d171
			ps172.PhiValues = make([]scm.JITValueDesc, 1)
			d174 = d6
			ps172.PhiValues[0] = d174
			ps173 := scm.PhiState{General: true}
			ps173.OverlayValues = make([]scm.JITValueDesc, 175)
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
			ps173.OverlayValues[20] = d20
			ps173.OverlayValues[21] = d21
			ps173.OverlayValues[22] = d22
			ps173.OverlayValues[24] = d24
			ps173.OverlayValues[25] = d25
			ps173.OverlayValues[27] = d27
			ps173.OverlayValues[28] = d28
			ps173.OverlayValues[29] = d29
			ps173.OverlayValues[32] = d32
			ps173.OverlayValues[34] = d34
			ps173.OverlayValues[35] = d35
			ps173.OverlayValues[36] = d36
			ps173.OverlayValues[38] = d38
			ps173.OverlayValues[39] = d39
			ps173.OverlayValues[40] = d40
			ps173.OverlayValues[41] = d41
			ps173.OverlayValues[42] = d42
			ps173.OverlayValues[43] = d43
			ps173.OverlayValues[44] = d44
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
			ps173.OverlayValues[62] = d62
			ps173.OverlayValues[63] = d63
			ps173.OverlayValues[64] = d64
			ps173.OverlayValues[65] = d65
			ps173.OverlayValues[66] = d66
			ps173.OverlayValues[67] = d67
			ps173.OverlayValues[68] = d68
			ps173.OverlayValues[69] = d69
			ps173.OverlayValues[70] = d70
			ps173.OverlayValues[71] = d71
			ps173.OverlayValues[72] = d72
			ps173.OverlayValues[73] = d73
			ps173.OverlayValues[74] = d74
			ps173.OverlayValues[75] = d75
			ps173.OverlayValues[76] = d76
			ps173.OverlayValues[77] = d77
			ps173.OverlayValues[78] = d78
			ps173.OverlayValues[79] = d79
			ps173.OverlayValues[80] = d80
			ps173.OverlayValues[81] = d81
			ps173.OverlayValues[82] = d82
			ps173.OverlayValues[83] = d83
			ps173.OverlayValues[84] = d84
			ps173.OverlayValues[85] = d85
			ps173.OverlayValues[86] = d86
			ps173.OverlayValues[87] = d87
			ps173.OverlayValues[88] = d88
			ps173.OverlayValues[89] = d89
			ps173.OverlayValues[90] = d90
			ps173.OverlayValues[91] = d91
			ps173.OverlayValues[92] = d92
			ps173.OverlayValues[93] = d93
			ps173.OverlayValues[94] = d94
			ps173.OverlayValues[95] = d95
			ps173.OverlayValues[102] = d102
			ps173.OverlayValues[103] = d103
			ps173.OverlayValues[104] = d104
			ps173.OverlayValues[105] = d105
			ps173.OverlayValues[106] = d106
			ps173.OverlayValues[107] = d107
			ps173.OverlayValues[108] = d108
			ps173.OverlayValues[109] = d109
			ps173.OverlayValues[110] = d110
			ps173.OverlayValues[111] = d111
			ps173.OverlayValues[112] = d112
			ps173.OverlayValues[113] = d113
			ps173.OverlayValues[114] = d114
			ps173.OverlayValues[115] = d115
			ps173.OverlayValues[116] = d116
			ps173.OverlayValues[117] = d117
			ps173.OverlayValues[118] = d118
			ps173.OverlayValues[119] = d119
			ps173.OverlayValues[120] = d120
			ps173.OverlayValues[121] = d121
			ps173.OverlayValues[122] = d122
			ps173.OverlayValues[123] = d123
			ps173.OverlayValues[124] = d124
			ps173.OverlayValues[125] = d125
			ps173.OverlayValues[126] = d126
			ps173.OverlayValues[127] = d127
			ps173.OverlayValues[128] = d128
			ps173.OverlayValues[129] = d129
			ps173.OverlayValues[130] = d130
			ps173.OverlayValues[131] = d131
			ps173.OverlayValues[132] = d132
			ps173.OverlayValues[133] = d133
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
			ps173.OverlayValues[153] = d153
			ps173.OverlayValues[154] = d154
			ps173.OverlayValues[160] = d160
			ps173.OverlayValues[161] = d161
			ps173.OverlayValues[162] = d162
			ps173.OverlayValues[163] = d163
			ps173.OverlayValues[164] = d164
			ps173.OverlayValues[165] = d165
			ps173.OverlayValues[166] = d166
			ps173.OverlayValues[168] = d168
			ps173.OverlayValues[170] = d170
			ps173.OverlayValues[171] = d171
			ps173.OverlayValues[174] = d174
			snap175 := d5
			alloc176 := ctx.SnapshotAllocState()
			if !bbs[6].Rendered {
				bbs[6].RenderPS(ps172)
			}
			ctx.RestoreAllocState(alloc176)
			d5 = snap175
			if !bbs[13].Rendered {
				return bbs[13].RenderPS(ps173)
			}
			return result
			ctx.FreeDesc(&d163)
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
					ctx.W.EmitJmp(lbl10)
					return result
				}
				bbs[9].Rendered = true
				bbs[9].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_9 = bbs[9].Address
				ctx.W.MarkLabel(lbl10)
				ctx.W.ResolveFixups()
			}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
				d154 = ps.OverlayValues[154]
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
			if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
				d168 = ps.OverlayValues[168]
			}
			if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != scm.LocNone {
				d170 = ps.OverlayValues[170]
			}
			if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != scm.LocNone {
				d171 = ps.OverlayValues[171]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			var d177 scm.JITValueDesc
			if d1.Loc == scm.LocImm {
				d177 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitMovRegReg(scratch, d1.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d177)
			}
			if d177.Loc == scm.LocImm {
				d177 = scm.JITValueDesc{Loc: scm.LocImm, Type: d177.Type, Imm: scm.NewInt(int64(uint64(d177.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d177.Reg, 32)
				ctx.W.EmitShrRegImm8(d177.Reg, 32)
			}
			if d177.Loc == scm.LocReg && d1.Loc == scm.LocReg && d177.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d12)
			var d178 scm.JITValueDesc
			if d177.Loc == scm.LocImm && d12.Loc == scm.LocImm {
				d178 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d177.Imm.Int()) >= uint64(d12.Imm.Int()))}
			} else if d12.Loc == scm.LocImm {
				r105 := ctx.AllocRegExcept(d177.Reg)
				if d12.Imm.Int() >= -2147483648 && d12.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d177.Reg, int32(d12.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d12.Imm.Int()))
					ctx.W.EmitCmpInt64(d177.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r105, scm.CcAE)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r105}
				ctx.BindReg(r105, &d178)
			} else if d177.Loc == scm.LocImm {
				r106 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d177.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d12.Reg)
				ctx.W.EmitSetcc(r106, scm.CcAE)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r106}
				ctx.BindReg(r106, &d178)
			} else {
				r107 := ctx.AllocRegExcept(d177.Reg)
				ctx.W.EmitCmpInt64(d177.Reg, d12.Reg)
				ctx.W.EmitSetcc(r107, scm.CcAE)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r107}
				ctx.BindReg(r107, &d178)
			}
			d179 = d178
			ctx.EnsureDesc(&d179)
			if d179.Loc != scm.LocImm && d179.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d179.Loc == scm.LocImm {
				if d179.Imm.Bool() {
			ps180 := scm.PhiState{General: ps.General}
			ps180.OverlayValues = make([]scm.JITValueDesc, 180)
			ps180.OverlayValues[0] = d0
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
			ps180.OverlayValues[20] = d20
			ps180.OverlayValues[21] = d21
			ps180.OverlayValues[22] = d22
			ps180.OverlayValues[24] = d24
			ps180.OverlayValues[25] = d25
			ps180.OverlayValues[27] = d27
			ps180.OverlayValues[28] = d28
			ps180.OverlayValues[29] = d29
			ps180.OverlayValues[32] = d32
			ps180.OverlayValues[34] = d34
			ps180.OverlayValues[35] = d35
			ps180.OverlayValues[36] = d36
			ps180.OverlayValues[38] = d38
			ps180.OverlayValues[39] = d39
			ps180.OverlayValues[40] = d40
			ps180.OverlayValues[41] = d41
			ps180.OverlayValues[42] = d42
			ps180.OverlayValues[43] = d43
			ps180.OverlayValues[44] = d44
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
			ps180.OverlayValues[64] = d64
			ps180.OverlayValues[65] = d65
			ps180.OverlayValues[66] = d66
			ps180.OverlayValues[67] = d67
			ps180.OverlayValues[68] = d68
			ps180.OverlayValues[69] = d69
			ps180.OverlayValues[70] = d70
			ps180.OverlayValues[71] = d71
			ps180.OverlayValues[72] = d72
			ps180.OverlayValues[73] = d73
			ps180.OverlayValues[74] = d74
			ps180.OverlayValues[75] = d75
			ps180.OverlayValues[76] = d76
			ps180.OverlayValues[77] = d77
			ps180.OverlayValues[78] = d78
			ps180.OverlayValues[79] = d79
			ps180.OverlayValues[80] = d80
			ps180.OverlayValues[81] = d81
			ps180.OverlayValues[82] = d82
			ps180.OverlayValues[83] = d83
			ps180.OverlayValues[84] = d84
			ps180.OverlayValues[85] = d85
			ps180.OverlayValues[86] = d86
			ps180.OverlayValues[87] = d87
			ps180.OverlayValues[88] = d88
			ps180.OverlayValues[89] = d89
			ps180.OverlayValues[90] = d90
			ps180.OverlayValues[91] = d91
			ps180.OverlayValues[92] = d92
			ps180.OverlayValues[93] = d93
			ps180.OverlayValues[94] = d94
			ps180.OverlayValues[95] = d95
			ps180.OverlayValues[102] = d102
			ps180.OverlayValues[103] = d103
			ps180.OverlayValues[104] = d104
			ps180.OverlayValues[105] = d105
			ps180.OverlayValues[106] = d106
			ps180.OverlayValues[107] = d107
			ps180.OverlayValues[108] = d108
			ps180.OverlayValues[109] = d109
			ps180.OverlayValues[110] = d110
			ps180.OverlayValues[111] = d111
			ps180.OverlayValues[112] = d112
			ps180.OverlayValues[113] = d113
			ps180.OverlayValues[114] = d114
			ps180.OverlayValues[115] = d115
			ps180.OverlayValues[116] = d116
			ps180.OverlayValues[117] = d117
			ps180.OverlayValues[118] = d118
			ps180.OverlayValues[119] = d119
			ps180.OverlayValues[120] = d120
			ps180.OverlayValues[121] = d121
			ps180.OverlayValues[122] = d122
			ps180.OverlayValues[123] = d123
			ps180.OverlayValues[124] = d124
			ps180.OverlayValues[125] = d125
			ps180.OverlayValues[126] = d126
			ps180.OverlayValues[127] = d127
			ps180.OverlayValues[128] = d128
			ps180.OverlayValues[129] = d129
			ps180.OverlayValues[130] = d130
			ps180.OverlayValues[131] = d131
			ps180.OverlayValues[132] = d132
			ps180.OverlayValues[133] = d133
			ps180.OverlayValues[134] = d134
			ps180.OverlayValues[135] = d135
			ps180.OverlayValues[136] = d136
			ps180.OverlayValues[137] = d137
			ps180.OverlayValues[138] = d138
			ps180.OverlayValues[139] = d139
			ps180.OverlayValues[140] = d140
			ps180.OverlayValues[141] = d141
			ps180.OverlayValues[142] = d142
			ps180.OverlayValues[143] = d143
			ps180.OverlayValues[144] = d144
			ps180.OverlayValues[145] = d145
			ps180.OverlayValues[146] = d146
			ps180.OverlayValues[153] = d153
			ps180.OverlayValues[154] = d154
			ps180.OverlayValues[160] = d160
			ps180.OverlayValues[161] = d161
			ps180.OverlayValues[162] = d162
			ps180.OverlayValues[163] = d163
			ps180.OverlayValues[164] = d164
			ps180.OverlayValues[165] = d165
			ps180.OverlayValues[166] = d166
			ps180.OverlayValues[168] = d168
			ps180.OverlayValues[170] = d170
			ps180.OverlayValues[171] = d171
			ps180.OverlayValues[174] = d174
			ps180.OverlayValues[177] = d177
			ps180.OverlayValues[178] = d178
			ps180.OverlayValues[179] = d179
					return bbs[12].RenderPS(ps180)
				}
			d181 = d177
			if d181.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d181)
			d182 = d181
			if d182.Loc == scm.LocImm {
				d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: d182.Type, Imm: scm.NewInt(int64(uint64(d182.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d182.Reg, 32)
				ctx.W.EmitShrRegImm8(d182.Reg, 32)
			}
			ctx.EmitStoreToStack(d182, 40)
			d183 = d1
			if d183.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d183)
			d184 = d183
			if d184.Loc == scm.LocImm {
				d184 = scm.JITValueDesc{Loc: scm.LocImm, Type: d184.Type, Imm: scm.NewInt(int64(uint64(d184.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d184.Reg, 32)
				ctx.W.EmitShrRegImm8(d184.Reg, 32)
			}
			ctx.EmitStoreToStack(d184, 48)
			d185 = d3
			if d185.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d185)
			d186 = d185
			if d186.Loc == scm.LocImm {
				d186 = scm.JITValueDesc{Loc: scm.LocImm, Type: d186.Type, Imm: scm.NewInt(int64(uint64(d186.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d186.Reg, 32)
				ctx.W.EmitShrRegImm8(d186.Reg, 32)
			}
			ctx.EmitStoreToStack(d186, 56)
			ps187 := scm.PhiState{General: ps.General}
			ps187.OverlayValues = make([]scm.JITValueDesc, 187)
			ps187.OverlayValues[0] = d0
			ps187.OverlayValues[1] = d1
			ps187.OverlayValues[2] = d2
			ps187.OverlayValues[3] = d3
			ps187.OverlayValues[4] = d4
			ps187.OverlayValues[5] = d5
			ps187.OverlayValues[6] = d6
			ps187.OverlayValues[7] = d7
			ps187.OverlayValues[8] = d8
			ps187.OverlayValues[9] = d9
			ps187.OverlayValues[10] = d10
			ps187.OverlayValues[11] = d11
			ps187.OverlayValues[12] = d12
			ps187.OverlayValues[13] = d13
			ps187.OverlayValues[14] = d14
			ps187.OverlayValues[20] = d20
			ps187.OverlayValues[21] = d21
			ps187.OverlayValues[22] = d22
			ps187.OverlayValues[24] = d24
			ps187.OverlayValues[25] = d25
			ps187.OverlayValues[27] = d27
			ps187.OverlayValues[28] = d28
			ps187.OverlayValues[29] = d29
			ps187.OverlayValues[32] = d32
			ps187.OverlayValues[34] = d34
			ps187.OverlayValues[35] = d35
			ps187.OverlayValues[36] = d36
			ps187.OverlayValues[38] = d38
			ps187.OverlayValues[39] = d39
			ps187.OverlayValues[40] = d40
			ps187.OverlayValues[41] = d41
			ps187.OverlayValues[42] = d42
			ps187.OverlayValues[43] = d43
			ps187.OverlayValues[44] = d44
			ps187.OverlayValues[46] = d46
			ps187.OverlayValues[47] = d47
			ps187.OverlayValues[48] = d48
			ps187.OverlayValues[49] = d49
			ps187.OverlayValues[50] = d50
			ps187.OverlayValues[51] = d51
			ps187.OverlayValues[52] = d52
			ps187.OverlayValues[53] = d53
			ps187.OverlayValues[54] = d54
			ps187.OverlayValues[55] = d55
			ps187.OverlayValues[56] = d56
			ps187.OverlayValues[57] = d57
			ps187.OverlayValues[58] = d58
			ps187.OverlayValues[59] = d59
			ps187.OverlayValues[60] = d60
			ps187.OverlayValues[61] = d61
			ps187.OverlayValues[62] = d62
			ps187.OverlayValues[63] = d63
			ps187.OverlayValues[64] = d64
			ps187.OverlayValues[65] = d65
			ps187.OverlayValues[66] = d66
			ps187.OverlayValues[67] = d67
			ps187.OverlayValues[68] = d68
			ps187.OverlayValues[69] = d69
			ps187.OverlayValues[70] = d70
			ps187.OverlayValues[71] = d71
			ps187.OverlayValues[72] = d72
			ps187.OverlayValues[73] = d73
			ps187.OverlayValues[74] = d74
			ps187.OverlayValues[75] = d75
			ps187.OverlayValues[76] = d76
			ps187.OverlayValues[77] = d77
			ps187.OverlayValues[78] = d78
			ps187.OverlayValues[79] = d79
			ps187.OverlayValues[80] = d80
			ps187.OverlayValues[81] = d81
			ps187.OverlayValues[82] = d82
			ps187.OverlayValues[83] = d83
			ps187.OverlayValues[84] = d84
			ps187.OverlayValues[85] = d85
			ps187.OverlayValues[86] = d86
			ps187.OverlayValues[87] = d87
			ps187.OverlayValues[88] = d88
			ps187.OverlayValues[89] = d89
			ps187.OverlayValues[90] = d90
			ps187.OverlayValues[91] = d91
			ps187.OverlayValues[92] = d92
			ps187.OverlayValues[93] = d93
			ps187.OverlayValues[94] = d94
			ps187.OverlayValues[95] = d95
			ps187.OverlayValues[102] = d102
			ps187.OverlayValues[103] = d103
			ps187.OverlayValues[104] = d104
			ps187.OverlayValues[105] = d105
			ps187.OverlayValues[106] = d106
			ps187.OverlayValues[107] = d107
			ps187.OverlayValues[108] = d108
			ps187.OverlayValues[109] = d109
			ps187.OverlayValues[110] = d110
			ps187.OverlayValues[111] = d111
			ps187.OverlayValues[112] = d112
			ps187.OverlayValues[113] = d113
			ps187.OverlayValues[114] = d114
			ps187.OverlayValues[115] = d115
			ps187.OverlayValues[116] = d116
			ps187.OverlayValues[117] = d117
			ps187.OverlayValues[118] = d118
			ps187.OverlayValues[119] = d119
			ps187.OverlayValues[120] = d120
			ps187.OverlayValues[121] = d121
			ps187.OverlayValues[122] = d122
			ps187.OverlayValues[123] = d123
			ps187.OverlayValues[124] = d124
			ps187.OverlayValues[125] = d125
			ps187.OverlayValues[126] = d126
			ps187.OverlayValues[127] = d127
			ps187.OverlayValues[128] = d128
			ps187.OverlayValues[129] = d129
			ps187.OverlayValues[130] = d130
			ps187.OverlayValues[131] = d131
			ps187.OverlayValues[132] = d132
			ps187.OverlayValues[133] = d133
			ps187.OverlayValues[134] = d134
			ps187.OverlayValues[135] = d135
			ps187.OverlayValues[136] = d136
			ps187.OverlayValues[137] = d137
			ps187.OverlayValues[138] = d138
			ps187.OverlayValues[139] = d139
			ps187.OverlayValues[140] = d140
			ps187.OverlayValues[141] = d141
			ps187.OverlayValues[142] = d142
			ps187.OverlayValues[143] = d143
			ps187.OverlayValues[144] = d144
			ps187.OverlayValues[145] = d145
			ps187.OverlayValues[146] = d146
			ps187.OverlayValues[153] = d153
			ps187.OverlayValues[154] = d154
			ps187.OverlayValues[160] = d160
			ps187.OverlayValues[161] = d161
			ps187.OverlayValues[162] = d162
			ps187.OverlayValues[163] = d163
			ps187.OverlayValues[164] = d164
			ps187.OverlayValues[165] = d165
			ps187.OverlayValues[166] = d166
			ps187.OverlayValues[168] = d168
			ps187.OverlayValues[170] = d170
			ps187.OverlayValues[171] = d171
			ps187.OverlayValues[174] = d174
			ps187.OverlayValues[177] = d177
			ps187.OverlayValues[178] = d178
			ps187.OverlayValues[179] = d179
			ps187.OverlayValues[181] = d181
			ps187.OverlayValues[182] = d182
			ps187.OverlayValues[183] = d183
			ps187.OverlayValues[184] = d184
			ps187.OverlayValues[185] = d185
			ps187.OverlayValues[186] = d186
			ps187.PhiValues = make([]scm.JITValueDesc, 3)
			d188 = d177
			ps187.PhiValues[0] = d188
			d189 = d1
			ps187.PhiValues[1] = d189
			d190 = d3
			ps187.PhiValues[2] = d190
				return bbs[8].RenderPS(ps187)
			}
			lbl52 := ctx.W.ReserveLabel()
			lbl53 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d179.Reg, 0)
			ctx.W.EmitJcc(scm.CcNE, lbl52)
			ctx.W.EmitJmp(lbl53)
			ctx.W.MarkLabel(lbl52)
			ctx.W.EmitJmp(lbl13)
			ctx.W.MarkLabel(lbl53)
			d191 = d177
			if d191.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d191)
			d192 = d191
			if d192.Loc == scm.LocImm {
				d192 = scm.JITValueDesc{Loc: scm.LocImm, Type: d192.Type, Imm: scm.NewInt(int64(uint64(d192.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d192.Reg, 32)
				ctx.W.EmitShrRegImm8(d192.Reg, 32)
			}
			ctx.EmitStoreToStack(d192, 40)
			d193 = d1
			if d193.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d193)
			d194 = d193
			if d194.Loc == scm.LocImm {
				d194 = scm.JITValueDesc{Loc: scm.LocImm, Type: d194.Type, Imm: scm.NewInt(int64(uint64(d194.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d194.Reg, 32)
				ctx.W.EmitShrRegImm8(d194.Reg, 32)
			}
			ctx.EmitStoreToStack(d194, 48)
			d195 = d3
			if d195.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d195)
			d196 = d195
			if d196.Loc == scm.LocImm {
				d196 = scm.JITValueDesc{Loc: scm.LocImm, Type: d196.Type, Imm: scm.NewInt(int64(uint64(d196.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d196.Reg, 32)
				ctx.W.EmitShrRegImm8(d196.Reg, 32)
			}
			ctx.EmitStoreToStack(d196, 56)
			ctx.W.EmitJmp(lbl9)
			ps197 := scm.PhiState{General: true}
			ps197.OverlayValues = make([]scm.JITValueDesc, 197)
			ps197.OverlayValues[0] = d0
			ps197.OverlayValues[1] = d1
			ps197.OverlayValues[2] = d2
			ps197.OverlayValues[3] = d3
			ps197.OverlayValues[4] = d4
			ps197.OverlayValues[5] = d5
			ps197.OverlayValues[6] = d6
			ps197.OverlayValues[7] = d7
			ps197.OverlayValues[8] = d8
			ps197.OverlayValues[9] = d9
			ps197.OverlayValues[10] = d10
			ps197.OverlayValues[11] = d11
			ps197.OverlayValues[12] = d12
			ps197.OverlayValues[13] = d13
			ps197.OverlayValues[14] = d14
			ps197.OverlayValues[20] = d20
			ps197.OverlayValues[21] = d21
			ps197.OverlayValues[22] = d22
			ps197.OverlayValues[24] = d24
			ps197.OverlayValues[25] = d25
			ps197.OverlayValues[27] = d27
			ps197.OverlayValues[28] = d28
			ps197.OverlayValues[29] = d29
			ps197.OverlayValues[32] = d32
			ps197.OverlayValues[34] = d34
			ps197.OverlayValues[35] = d35
			ps197.OverlayValues[36] = d36
			ps197.OverlayValues[38] = d38
			ps197.OverlayValues[39] = d39
			ps197.OverlayValues[40] = d40
			ps197.OverlayValues[41] = d41
			ps197.OverlayValues[42] = d42
			ps197.OverlayValues[43] = d43
			ps197.OverlayValues[44] = d44
			ps197.OverlayValues[46] = d46
			ps197.OverlayValues[47] = d47
			ps197.OverlayValues[48] = d48
			ps197.OverlayValues[49] = d49
			ps197.OverlayValues[50] = d50
			ps197.OverlayValues[51] = d51
			ps197.OverlayValues[52] = d52
			ps197.OverlayValues[53] = d53
			ps197.OverlayValues[54] = d54
			ps197.OverlayValues[55] = d55
			ps197.OverlayValues[56] = d56
			ps197.OverlayValues[57] = d57
			ps197.OverlayValues[58] = d58
			ps197.OverlayValues[59] = d59
			ps197.OverlayValues[60] = d60
			ps197.OverlayValues[61] = d61
			ps197.OverlayValues[62] = d62
			ps197.OverlayValues[63] = d63
			ps197.OverlayValues[64] = d64
			ps197.OverlayValues[65] = d65
			ps197.OverlayValues[66] = d66
			ps197.OverlayValues[67] = d67
			ps197.OverlayValues[68] = d68
			ps197.OverlayValues[69] = d69
			ps197.OverlayValues[70] = d70
			ps197.OverlayValues[71] = d71
			ps197.OverlayValues[72] = d72
			ps197.OverlayValues[73] = d73
			ps197.OverlayValues[74] = d74
			ps197.OverlayValues[75] = d75
			ps197.OverlayValues[76] = d76
			ps197.OverlayValues[77] = d77
			ps197.OverlayValues[78] = d78
			ps197.OverlayValues[79] = d79
			ps197.OverlayValues[80] = d80
			ps197.OverlayValues[81] = d81
			ps197.OverlayValues[82] = d82
			ps197.OverlayValues[83] = d83
			ps197.OverlayValues[84] = d84
			ps197.OverlayValues[85] = d85
			ps197.OverlayValues[86] = d86
			ps197.OverlayValues[87] = d87
			ps197.OverlayValues[88] = d88
			ps197.OverlayValues[89] = d89
			ps197.OverlayValues[90] = d90
			ps197.OverlayValues[91] = d91
			ps197.OverlayValues[92] = d92
			ps197.OverlayValues[93] = d93
			ps197.OverlayValues[94] = d94
			ps197.OverlayValues[95] = d95
			ps197.OverlayValues[102] = d102
			ps197.OverlayValues[103] = d103
			ps197.OverlayValues[104] = d104
			ps197.OverlayValues[105] = d105
			ps197.OverlayValues[106] = d106
			ps197.OverlayValues[107] = d107
			ps197.OverlayValues[108] = d108
			ps197.OverlayValues[109] = d109
			ps197.OverlayValues[110] = d110
			ps197.OverlayValues[111] = d111
			ps197.OverlayValues[112] = d112
			ps197.OverlayValues[113] = d113
			ps197.OverlayValues[114] = d114
			ps197.OverlayValues[115] = d115
			ps197.OverlayValues[116] = d116
			ps197.OverlayValues[117] = d117
			ps197.OverlayValues[118] = d118
			ps197.OverlayValues[119] = d119
			ps197.OverlayValues[120] = d120
			ps197.OverlayValues[121] = d121
			ps197.OverlayValues[122] = d122
			ps197.OverlayValues[123] = d123
			ps197.OverlayValues[124] = d124
			ps197.OverlayValues[125] = d125
			ps197.OverlayValues[126] = d126
			ps197.OverlayValues[127] = d127
			ps197.OverlayValues[128] = d128
			ps197.OverlayValues[129] = d129
			ps197.OverlayValues[130] = d130
			ps197.OverlayValues[131] = d131
			ps197.OverlayValues[132] = d132
			ps197.OverlayValues[133] = d133
			ps197.OverlayValues[134] = d134
			ps197.OverlayValues[135] = d135
			ps197.OverlayValues[136] = d136
			ps197.OverlayValues[137] = d137
			ps197.OverlayValues[138] = d138
			ps197.OverlayValues[139] = d139
			ps197.OverlayValues[140] = d140
			ps197.OverlayValues[141] = d141
			ps197.OverlayValues[142] = d142
			ps197.OverlayValues[143] = d143
			ps197.OverlayValues[144] = d144
			ps197.OverlayValues[145] = d145
			ps197.OverlayValues[146] = d146
			ps197.OverlayValues[153] = d153
			ps197.OverlayValues[154] = d154
			ps197.OverlayValues[160] = d160
			ps197.OverlayValues[161] = d161
			ps197.OverlayValues[162] = d162
			ps197.OverlayValues[163] = d163
			ps197.OverlayValues[164] = d164
			ps197.OverlayValues[165] = d165
			ps197.OverlayValues[166] = d166
			ps197.OverlayValues[168] = d168
			ps197.OverlayValues[170] = d170
			ps197.OverlayValues[171] = d171
			ps197.OverlayValues[174] = d174
			ps197.OverlayValues[177] = d177
			ps197.OverlayValues[178] = d178
			ps197.OverlayValues[179] = d179
			ps197.OverlayValues[181] = d181
			ps197.OverlayValues[182] = d182
			ps197.OverlayValues[183] = d183
			ps197.OverlayValues[184] = d184
			ps197.OverlayValues[185] = d185
			ps197.OverlayValues[186] = d186
			ps197.OverlayValues[188] = d188
			ps197.OverlayValues[189] = d189
			ps197.OverlayValues[190] = d190
			ps197.OverlayValues[191] = d191
			ps197.OverlayValues[192] = d192
			ps197.OverlayValues[193] = d193
			ps197.OverlayValues[194] = d194
			ps197.OverlayValues[195] = d195
			ps197.OverlayValues[196] = d196
			ps198 := scm.PhiState{General: true}
			ps198.OverlayValues = make([]scm.JITValueDesc, 197)
			ps198.OverlayValues[0] = d0
			ps198.OverlayValues[1] = d1
			ps198.OverlayValues[2] = d2
			ps198.OverlayValues[3] = d3
			ps198.OverlayValues[4] = d4
			ps198.OverlayValues[5] = d5
			ps198.OverlayValues[6] = d6
			ps198.OverlayValues[7] = d7
			ps198.OverlayValues[8] = d8
			ps198.OverlayValues[9] = d9
			ps198.OverlayValues[10] = d10
			ps198.OverlayValues[11] = d11
			ps198.OverlayValues[12] = d12
			ps198.OverlayValues[13] = d13
			ps198.OverlayValues[14] = d14
			ps198.OverlayValues[20] = d20
			ps198.OverlayValues[21] = d21
			ps198.OverlayValues[22] = d22
			ps198.OverlayValues[24] = d24
			ps198.OverlayValues[25] = d25
			ps198.OverlayValues[27] = d27
			ps198.OverlayValues[28] = d28
			ps198.OverlayValues[29] = d29
			ps198.OverlayValues[32] = d32
			ps198.OverlayValues[34] = d34
			ps198.OverlayValues[35] = d35
			ps198.OverlayValues[36] = d36
			ps198.OverlayValues[38] = d38
			ps198.OverlayValues[39] = d39
			ps198.OverlayValues[40] = d40
			ps198.OverlayValues[41] = d41
			ps198.OverlayValues[42] = d42
			ps198.OverlayValues[43] = d43
			ps198.OverlayValues[44] = d44
			ps198.OverlayValues[46] = d46
			ps198.OverlayValues[47] = d47
			ps198.OverlayValues[48] = d48
			ps198.OverlayValues[49] = d49
			ps198.OverlayValues[50] = d50
			ps198.OverlayValues[51] = d51
			ps198.OverlayValues[52] = d52
			ps198.OverlayValues[53] = d53
			ps198.OverlayValues[54] = d54
			ps198.OverlayValues[55] = d55
			ps198.OverlayValues[56] = d56
			ps198.OverlayValues[57] = d57
			ps198.OverlayValues[58] = d58
			ps198.OverlayValues[59] = d59
			ps198.OverlayValues[60] = d60
			ps198.OverlayValues[61] = d61
			ps198.OverlayValues[62] = d62
			ps198.OverlayValues[63] = d63
			ps198.OverlayValues[64] = d64
			ps198.OverlayValues[65] = d65
			ps198.OverlayValues[66] = d66
			ps198.OverlayValues[67] = d67
			ps198.OverlayValues[68] = d68
			ps198.OverlayValues[69] = d69
			ps198.OverlayValues[70] = d70
			ps198.OverlayValues[71] = d71
			ps198.OverlayValues[72] = d72
			ps198.OverlayValues[73] = d73
			ps198.OverlayValues[74] = d74
			ps198.OverlayValues[75] = d75
			ps198.OverlayValues[76] = d76
			ps198.OverlayValues[77] = d77
			ps198.OverlayValues[78] = d78
			ps198.OverlayValues[79] = d79
			ps198.OverlayValues[80] = d80
			ps198.OverlayValues[81] = d81
			ps198.OverlayValues[82] = d82
			ps198.OverlayValues[83] = d83
			ps198.OverlayValues[84] = d84
			ps198.OverlayValues[85] = d85
			ps198.OverlayValues[86] = d86
			ps198.OverlayValues[87] = d87
			ps198.OverlayValues[88] = d88
			ps198.OverlayValues[89] = d89
			ps198.OverlayValues[90] = d90
			ps198.OverlayValues[91] = d91
			ps198.OverlayValues[92] = d92
			ps198.OverlayValues[93] = d93
			ps198.OverlayValues[94] = d94
			ps198.OverlayValues[95] = d95
			ps198.OverlayValues[102] = d102
			ps198.OverlayValues[103] = d103
			ps198.OverlayValues[104] = d104
			ps198.OverlayValues[105] = d105
			ps198.OverlayValues[106] = d106
			ps198.OverlayValues[107] = d107
			ps198.OverlayValues[108] = d108
			ps198.OverlayValues[109] = d109
			ps198.OverlayValues[110] = d110
			ps198.OverlayValues[111] = d111
			ps198.OverlayValues[112] = d112
			ps198.OverlayValues[113] = d113
			ps198.OverlayValues[114] = d114
			ps198.OverlayValues[115] = d115
			ps198.OverlayValues[116] = d116
			ps198.OverlayValues[117] = d117
			ps198.OverlayValues[118] = d118
			ps198.OverlayValues[119] = d119
			ps198.OverlayValues[120] = d120
			ps198.OverlayValues[121] = d121
			ps198.OverlayValues[122] = d122
			ps198.OverlayValues[123] = d123
			ps198.OverlayValues[124] = d124
			ps198.OverlayValues[125] = d125
			ps198.OverlayValues[126] = d126
			ps198.OverlayValues[127] = d127
			ps198.OverlayValues[128] = d128
			ps198.OverlayValues[129] = d129
			ps198.OverlayValues[130] = d130
			ps198.OverlayValues[131] = d131
			ps198.OverlayValues[132] = d132
			ps198.OverlayValues[133] = d133
			ps198.OverlayValues[134] = d134
			ps198.OverlayValues[135] = d135
			ps198.OverlayValues[136] = d136
			ps198.OverlayValues[137] = d137
			ps198.OverlayValues[138] = d138
			ps198.OverlayValues[139] = d139
			ps198.OverlayValues[140] = d140
			ps198.OverlayValues[141] = d141
			ps198.OverlayValues[142] = d142
			ps198.OverlayValues[143] = d143
			ps198.OverlayValues[144] = d144
			ps198.OverlayValues[145] = d145
			ps198.OverlayValues[146] = d146
			ps198.OverlayValues[153] = d153
			ps198.OverlayValues[154] = d154
			ps198.OverlayValues[160] = d160
			ps198.OverlayValues[161] = d161
			ps198.OverlayValues[162] = d162
			ps198.OverlayValues[163] = d163
			ps198.OverlayValues[164] = d164
			ps198.OverlayValues[165] = d165
			ps198.OverlayValues[166] = d166
			ps198.OverlayValues[168] = d168
			ps198.OverlayValues[170] = d170
			ps198.OverlayValues[171] = d171
			ps198.OverlayValues[174] = d174
			ps198.OverlayValues[177] = d177
			ps198.OverlayValues[178] = d178
			ps198.OverlayValues[179] = d179
			ps198.OverlayValues[181] = d181
			ps198.OverlayValues[182] = d182
			ps198.OverlayValues[183] = d183
			ps198.OverlayValues[184] = d184
			ps198.OverlayValues[185] = d185
			ps198.OverlayValues[186] = d186
			ps198.OverlayValues[188] = d188
			ps198.OverlayValues[189] = d189
			ps198.OverlayValues[190] = d190
			ps198.OverlayValues[191] = d191
			ps198.OverlayValues[192] = d192
			ps198.OverlayValues[193] = d193
			ps198.OverlayValues[194] = d194
			ps198.OverlayValues[195] = d195
			ps198.OverlayValues[196] = d196
			ps198.PhiValues = make([]scm.JITValueDesc, 3)
			d199 = d177
			ps198.PhiValues[0] = d199
			d200 = d1
			ps198.PhiValues[1] = d200
			d201 = d3
			ps198.PhiValues[2] = d201
			alloc202 := ctx.SnapshotAllocState()
			if !bbs[8].Rendered {
				bbs[8].RenderPS(ps198)
			}
			ctx.RestoreAllocState(alloc202)
			if !bbs[12].Rendered {
				return bbs[12].RenderPS(ps197)
			}
			return result
			ctx.FreeDesc(&d178)
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
					ctx.W.EmitJmp(lbl11)
					return result
				}
				bbs[10].Rendered = true
				bbs[10].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_10 = bbs[10].Address
				ctx.W.MarkLabel(lbl11)
				ctx.W.ResolveFixups()
			}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
				d154 = ps.OverlayValues[154]
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
			if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
				d168 = ps.OverlayValues[168]
			}
			if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != scm.LocNone {
				d170 = ps.OverlayValues[170]
			}
			if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != scm.LocNone {
				d171 = ps.OverlayValues[171]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
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
			if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
				d199 = ps.OverlayValues[199]
			}
			if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
				d200 = ps.OverlayValues[200]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
				d201 = ps.OverlayValues[201]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 32)
			ps203 := scm.PhiState{General: ps.General}
			ps203.OverlayValues = make([]scm.JITValueDesc, 202)
			ps203.OverlayValues[0] = d0
			ps203.OverlayValues[1] = d1
			ps203.OverlayValues[2] = d2
			ps203.OverlayValues[3] = d3
			ps203.OverlayValues[4] = d4
			ps203.OverlayValues[5] = d5
			ps203.OverlayValues[6] = d6
			ps203.OverlayValues[7] = d7
			ps203.OverlayValues[8] = d8
			ps203.OverlayValues[9] = d9
			ps203.OverlayValues[10] = d10
			ps203.OverlayValues[11] = d11
			ps203.OverlayValues[12] = d12
			ps203.OverlayValues[13] = d13
			ps203.OverlayValues[14] = d14
			ps203.OverlayValues[20] = d20
			ps203.OverlayValues[21] = d21
			ps203.OverlayValues[22] = d22
			ps203.OverlayValues[24] = d24
			ps203.OverlayValues[25] = d25
			ps203.OverlayValues[27] = d27
			ps203.OverlayValues[28] = d28
			ps203.OverlayValues[29] = d29
			ps203.OverlayValues[32] = d32
			ps203.OverlayValues[34] = d34
			ps203.OverlayValues[35] = d35
			ps203.OverlayValues[36] = d36
			ps203.OverlayValues[38] = d38
			ps203.OverlayValues[39] = d39
			ps203.OverlayValues[40] = d40
			ps203.OverlayValues[41] = d41
			ps203.OverlayValues[42] = d42
			ps203.OverlayValues[43] = d43
			ps203.OverlayValues[44] = d44
			ps203.OverlayValues[46] = d46
			ps203.OverlayValues[47] = d47
			ps203.OverlayValues[48] = d48
			ps203.OverlayValues[49] = d49
			ps203.OverlayValues[50] = d50
			ps203.OverlayValues[51] = d51
			ps203.OverlayValues[52] = d52
			ps203.OverlayValues[53] = d53
			ps203.OverlayValues[54] = d54
			ps203.OverlayValues[55] = d55
			ps203.OverlayValues[56] = d56
			ps203.OverlayValues[57] = d57
			ps203.OverlayValues[58] = d58
			ps203.OverlayValues[59] = d59
			ps203.OverlayValues[60] = d60
			ps203.OverlayValues[61] = d61
			ps203.OverlayValues[62] = d62
			ps203.OverlayValues[63] = d63
			ps203.OverlayValues[64] = d64
			ps203.OverlayValues[65] = d65
			ps203.OverlayValues[66] = d66
			ps203.OverlayValues[67] = d67
			ps203.OverlayValues[68] = d68
			ps203.OverlayValues[69] = d69
			ps203.OverlayValues[70] = d70
			ps203.OverlayValues[71] = d71
			ps203.OverlayValues[72] = d72
			ps203.OverlayValues[73] = d73
			ps203.OverlayValues[74] = d74
			ps203.OverlayValues[75] = d75
			ps203.OverlayValues[76] = d76
			ps203.OverlayValues[77] = d77
			ps203.OverlayValues[78] = d78
			ps203.OverlayValues[79] = d79
			ps203.OverlayValues[80] = d80
			ps203.OverlayValues[81] = d81
			ps203.OverlayValues[82] = d82
			ps203.OverlayValues[83] = d83
			ps203.OverlayValues[84] = d84
			ps203.OverlayValues[85] = d85
			ps203.OverlayValues[86] = d86
			ps203.OverlayValues[87] = d87
			ps203.OverlayValues[88] = d88
			ps203.OverlayValues[89] = d89
			ps203.OverlayValues[90] = d90
			ps203.OverlayValues[91] = d91
			ps203.OverlayValues[92] = d92
			ps203.OverlayValues[93] = d93
			ps203.OverlayValues[94] = d94
			ps203.OverlayValues[95] = d95
			ps203.OverlayValues[102] = d102
			ps203.OverlayValues[103] = d103
			ps203.OverlayValues[104] = d104
			ps203.OverlayValues[105] = d105
			ps203.OverlayValues[106] = d106
			ps203.OverlayValues[107] = d107
			ps203.OverlayValues[108] = d108
			ps203.OverlayValues[109] = d109
			ps203.OverlayValues[110] = d110
			ps203.OverlayValues[111] = d111
			ps203.OverlayValues[112] = d112
			ps203.OverlayValues[113] = d113
			ps203.OverlayValues[114] = d114
			ps203.OverlayValues[115] = d115
			ps203.OverlayValues[116] = d116
			ps203.OverlayValues[117] = d117
			ps203.OverlayValues[118] = d118
			ps203.OverlayValues[119] = d119
			ps203.OverlayValues[120] = d120
			ps203.OverlayValues[121] = d121
			ps203.OverlayValues[122] = d122
			ps203.OverlayValues[123] = d123
			ps203.OverlayValues[124] = d124
			ps203.OverlayValues[125] = d125
			ps203.OverlayValues[126] = d126
			ps203.OverlayValues[127] = d127
			ps203.OverlayValues[128] = d128
			ps203.OverlayValues[129] = d129
			ps203.OverlayValues[130] = d130
			ps203.OverlayValues[131] = d131
			ps203.OverlayValues[132] = d132
			ps203.OverlayValues[133] = d133
			ps203.OverlayValues[134] = d134
			ps203.OverlayValues[135] = d135
			ps203.OverlayValues[136] = d136
			ps203.OverlayValues[137] = d137
			ps203.OverlayValues[138] = d138
			ps203.OverlayValues[139] = d139
			ps203.OverlayValues[140] = d140
			ps203.OverlayValues[141] = d141
			ps203.OverlayValues[142] = d142
			ps203.OverlayValues[143] = d143
			ps203.OverlayValues[144] = d144
			ps203.OverlayValues[145] = d145
			ps203.OverlayValues[146] = d146
			ps203.OverlayValues[153] = d153
			ps203.OverlayValues[154] = d154
			ps203.OverlayValues[160] = d160
			ps203.OverlayValues[161] = d161
			ps203.OverlayValues[162] = d162
			ps203.OverlayValues[163] = d163
			ps203.OverlayValues[164] = d164
			ps203.OverlayValues[165] = d165
			ps203.OverlayValues[166] = d166
			ps203.OverlayValues[168] = d168
			ps203.OverlayValues[170] = d170
			ps203.OverlayValues[171] = d171
			ps203.OverlayValues[174] = d174
			ps203.OverlayValues[177] = d177
			ps203.OverlayValues[178] = d178
			ps203.OverlayValues[179] = d179
			ps203.OverlayValues[181] = d181
			ps203.OverlayValues[182] = d182
			ps203.OverlayValues[183] = d183
			ps203.OverlayValues[184] = d184
			ps203.OverlayValues[185] = d185
			ps203.OverlayValues[186] = d186
			ps203.OverlayValues[188] = d188
			ps203.OverlayValues[189] = d189
			ps203.OverlayValues[190] = d190
			ps203.OverlayValues[191] = d191
			ps203.OverlayValues[192] = d192
			ps203.OverlayValues[193] = d193
			ps203.OverlayValues[194] = d194
			ps203.OverlayValues[195] = d195
			ps203.OverlayValues[196] = d196
			ps203.OverlayValues[199] = d199
			ps203.OverlayValues[200] = d200
			ps203.OverlayValues[201] = d201
			ps203.PhiValues = make([]scm.JITValueDesc, 1)
			d204 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
			ps203.PhiValues[0] = d204
			if ps203.General && bbs[6].Rendered {
				ctx.W.EmitJmp(lbl7)
				return result
			}
			return bbs[6].RenderPS(ps203)
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
					ctx.W.EmitJmp(lbl12)
					return result
				}
				bbs[11].Rendered = true
				bbs[11].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_11 = bbs[11].Address
				ctx.W.MarkLabel(lbl12)
				ctx.W.ResolveFixups()
			}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
				d154 = ps.OverlayValues[154]
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
			if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
				d168 = ps.OverlayValues[168]
			}
			if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != scm.LocNone {
				d170 = ps.OverlayValues[170]
			}
			if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != scm.LocNone {
				d171 = ps.OverlayValues[171]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
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
			if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
				d199 = ps.OverlayValues[199]
			}
			if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
				d200 = ps.OverlayValues[200]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
				d201 = ps.OverlayValues[201]
			}
			if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != scm.LocNone {
				d204 = ps.OverlayValues[204]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			var d205 scm.JITValueDesc
			if d1.Loc == scm.LocImm {
				d205 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitMovRegReg(scratch, d1.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d205 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d205)
			}
			if d205.Loc == scm.LocImm {
				d205 = scm.JITValueDesc{Loc: scm.LocImm, Type: d205.Type, Imm: scm.NewInt(int64(uint64(d205.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d205.Reg, 32)
				ctx.W.EmitShrRegImm8(d205.Reg, 32)
			}
			if d205.Loc == scm.LocReg && d1.Loc == scm.LocReg && d205.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			var d206 scm.JITValueDesc
			if d1.Loc == scm.LocImm {
				d206 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitMovRegReg(scratch, d1.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d206)
			}
			if d206.Loc == scm.LocImm {
				d206 = scm.JITValueDesc{Loc: scm.LocImm, Type: d206.Type, Imm: scm.NewInt(int64(uint64(d206.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d206.Reg, 32)
				ctx.W.EmitShrRegImm8(d206.Reg, 32)
			}
			if d206.Loc == scm.LocReg && d1.Loc == scm.LocReg && d206.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = scm.LocNone
			}
			d207 = d206
			if d207.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d207)
			d208 = d207
			if d208.Loc == scm.LocImm {
				d208 = scm.JITValueDesc{Loc: scm.LocImm, Type: d208.Type, Imm: scm.NewInt(int64(uint64(d208.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d208.Reg, 32)
				ctx.W.EmitShrRegImm8(d208.Reg, 32)
			}
			ctx.EmitStoreToStack(d208, 40)
			d209 = d2
			if d209.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d209)
			d210 = d209
			if d210.Loc == scm.LocImm {
				d210 = scm.JITValueDesc{Loc: scm.LocImm, Type: d210.Type, Imm: scm.NewInt(int64(uint64(d210.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d210.Reg, 32)
				ctx.W.EmitShrRegImm8(d210.Reg, 32)
			}
			ctx.EmitStoreToStack(d210, 48)
			d211 = d205
			if d211.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d211)
			d212 = d211
			if d212.Loc == scm.LocImm {
				d212 = scm.JITValueDesc{Loc: scm.LocImm, Type: d212.Type, Imm: scm.NewInt(int64(uint64(d212.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d212.Reg, 32)
				ctx.W.EmitShrRegImm8(d212.Reg, 32)
			}
			ctx.EmitStoreToStack(d212, 56)
			ps213 := scm.PhiState{General: ps.General}
			ps213.OverlayValues = make([]scm.JITValueDesc, 213)
			ps213.OverlayValues[0] = d0
			ps213.OverlayValues[1] = d1
			ps213.OverlayValues[2] = d2
			ps213.OverlayValues[3] = d3
			ps213.OverlayValues[4] = d4
			ps213.OverlayValues[5] = d5
			ps213.OverlayValues[6] = d6
			ps213.OverlayValues[7] = d7
			ps213.OverlayValues[8] = d8
			ps213.OverlayValues[9] = d9
			ps213.OverlayValues[10] = d10
			ps213.OverlayValues[11] = d11
			ps213.OverlayValues[12] = d12
			ps213.OverlayValues[13] = d13
			ps213.OverlayValues[14] = d14
			ps213.OverlayValues[20] = d20
			ps213.OverlayValues[21] = d21
			ps213.OverlayValues[22] = d22
			ps213.OverlayValues[24] = d24
			ps213.OverlayValues[25] = d25
			ps213.OverlayValues[27] = d27
			ps213.OverlayValues[28] = d28
			ps213.OverlayValues[29] = d29
			ps213.OverlayValues[32] = d32
			ps213.OverlayValues[34] = d34
			ps213.OverlayValues[35] = d35
			ps213.OverlayValues[36] = d36
			ps213.OverlayValues[38] = d38
			ps213.OverlayValues[39] = d39
			ps213.OverlayValues[40] = d40
			ps213.OverlayValues[41] = d41
			ps213.OverlayValues[42] = d42
			ps213.OverlayValues[43] = d43
			ps213.OverlayValues[44] = d44
			ps213.OverlayValues[46] = d46
			ps213.OverlayValues[47] = d47
			ps213.OverlayValues[48] = d48
			ps213.OverlayValues[49] = d49
			ps213.OverlayValues[50] = d50
			ps213.OverlayValues[51] = d51
			ps213.OverlayValues[52] = d52
			ps213.OverlayValues[53] = d53
			ps213.OverlayValues[54] = d54
			ps213.OverlayValues[55] = d55
			ps213.OverlayValues[56] = d56
			ps213.OverlayValues[57] = d57
			ps213.OverlayValues[58] = d58
			ps213.OverlayValues[59] = d59
			ps213.OverlayValues[60] = d60
			ps213.OverlayValues[61] = d61
			ps213.OverlayValues[62] = d62
			ps213.OverlayValues[63] = d63
			ps213.OverlayValues[64] = d64
			ps213.OverlayValues[65] = d65
			ps213.OverlayValues[66] = d66
			ps213.OverlayValues[67] = d67
			ps213.OverlayValues[68] = d68
			ps213.OverlayValues[69] = d69
			ps213.OverlayValues[70] = d70
			ps213.OverlayValues[71] = d71
			ps213.OverlayValues[72] = d72
			ps213.OverlayValues[73] = d73
			ps213.OverlayValues[74] = d74
			ps213.OverlayValues[75] = d75
			ps213.OverlayValues[76] = d76
			ps213.OverlayValues[77] = d77
			ps213.OverlayValues[78] = d78
			ps213.OverlayValues[79] = d79
			ps213.OverlayValues[80] = d80
			ps213.OverlayValues[81] = d81
			ps213.OverlayValues[82] = d82
			ps213.OverlayValues[83] = d83
			ps213.OverlayValues[84] = d84
			ps213.OverlayValues[85] = d85
			ps213.OverlayValues[86] = d86
			ps213.OverlayValues[87] = d87
			ps213.OverlayValues[88] = d88
			ps213.OverlayValues[89] = d89
			ps213.OverlayValues[90] = d90
			ps213.OverlayValues[91] = d91
			ps213.OverlayValues[92] = d92
			ps213.OverlayValues[93] = d93
			ps213.OverlayValues[94] = d94
			ps213.OverlayValues[95] = d95
			ps213.OverlayValues[102] = d102
			ps213.OverlayValues[103] = d103
			ps213.OverlayValues[104] = d104
			ps213.OverlayValues[105] = d105
			ps213.OverlayValues[106] = d106
			ps213.OverlayValues[107] = d107
			ps213.OverlayValues[108] = d108
			ps213.OverlayValues[109] = d109
			ps213.OverlayValues[110] = d110
			ps213.OverlayValues[111] = d111
			ps213.OverlayValues[112] = d112
			ps213.OverlayValues[113] = d113
			ps213.OverlayValues[114] = d114
			ps213.OverlayValues[115] = d115
			ps213.OverlayValues[116] = d116
			ps213.OverlayValues[117] = d117
			ps213.OverlayValues[118] = d118
			ps213.OverlayValues[119] = d119
			ps213.OverlayValues[120] = d120
			ps213.OverlayValues[121] = d121
			ps213.OverlayValues[122] = d122
			ps213.OverlayValues[123] = d123
			ps213.OverlayValues[124] = d124
			ps213.OverlayValues[125] = d125
			ps213.OverlayValues[126] = d126
			ps213.OverlayValues[127] = d127
			ps213.OverlayValues[128] = d128
			ps213.OverlayValues[129] = d129
			ps213.OverlayValues[130] = d130
			ps213.OverlayValues[131] = d131
			ps213.OverlayValues[132] = d132
			ps213.OverlayValues[133] = d133
			ps213.OverlayValues[134] = d134
			ps213.OverlayValues[135] = d135
			ps213.OverlayValues[136] = d136
			ps213.OverlayValues[137] = d137
			ps213.OverlayValues[138] = d138
			ps213.OverlayValues[139] = d139
			ps213.OverlayValues[140] = d140
			ps213.OverlayValues[141] = d141
			ps213.OverlayValues[142] = d142
			ps213.OverlayValues[143] = d143
			ps213.OverlayValues[144] = d144
			ps213.OverlayValues[145] = d145
			ps213.OverlayValues[146] = d146
			ps213.OverlayValues[153] = d153
			ps213.OverlayValues[154] = d154
			ps213.OverlayValues[160] = d160
			ps213.OverlayValues[161] = d161
			ps213.OverlayValues[162] = d162
			ps213.OverlayValues[163] = d163
			ps213.OverlayValues[164] = d164
			ps213.OverlayValues[165] = d165
			ps213.OverlayValues[166] = d166
			ps213.OverlayValues[168] = d168
			ps213.OverlayValues[170] = d170
			ps213.OverlayValues[171] = d171
			ps213.OverlayValues[174] = d174
			ps213.OverlayValues[177] = d177
			ps213.OverlayValues[178] = d178
			ps213.OverlayValues[179] = d179
			ps213.OverlayValues[181] = d181
			ps213.OverlayValues[182] = d182
			ps213.OverlayValues[183] = d183
			ps213.OverlayValues[184] = d184
			ps213.OverlayValues[185] = d185
			ps213.OverlayValues[186] = d186
			ps213.OverlayValues[188] = d188
			ps213.OverlayValues[189] = d189
			ps213.OverlayValues[190] = d190
			ps213.OverlayValues[191] = d191
			ps213.OverlayValues[192] = d192
			ps213.OverlayValues[193] = d193
			ps213.OverlayValues[194] = d194
			ps213.OverlayValues[195] = d195
			ps213.OverlayValues[196] = d196
			ps213.OverlayValues[199] = d199
			ps213.OverlayValues[200] = d200
			ps213.OverlayValues[201] = d201
			ps213.OverlayValues[204] = d204
			ps213.OverlayValues[205] = d205
			ps213.OverlayValues[206] = d206
			ps213.OverlayValues[207] = d207
			ps213.OverlayValues[208] = d208
			ps213.OverlayValues[209] = d209
			ps213.OverlayValues[210] = d210
			ps213.OverlayValues[211] = d211
			ps213.OverlayValues[212] = d212
			ps213.PhiValues = make([]scm.JITValueDesc, 3)
			d214 = d206
			ps213.PhiValues[0] = d214
			d215 = d2
			ps213.PhiValues[1] = d215
			d216 = d205
			ps213.PhiValues[2] = d216
			if ps213.General && bbs[8].Rendered {
				ctx.W.EmitJmp(lbl9)
				return result
			}
			return bbs[8].RenderPS(ps213)
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
					ctx.W.EmitJmp(lbl13)
					return result
				}
				bbs[12].Rendered = true
				bbs[12].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_12 = bbs[12].Address
				ctx.W.MarkLabel(lbl13)
				ctx.W.ResolveFixups()
			}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
				d154 = ps.OverlayValues[154]
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
			if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
				d168 = ps.OverlayValues[168]
			}
			if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != scm.LocNone {
				d170 = ps.OverlayValues[170]
			}
			if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != scm.LocNone {
				d171 = ps.OverlayValues[171]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
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
			if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
				d199 = ps.OverlayValues[199]
			}
			if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
				d200 = ps.OverlayValues[200]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
				d201 = ps.OverlayValues[201]
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
			if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != scm.LocNone {
				d214 = ps.OverlayValues[214]
			}
			if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != scm.LocNone {
				d215 = ps.OverlayValues[215]
			}
			if len(ps.OverlayValues) > 216 && ps.OverlayValues[216].Loc != scm.LocNone {
				d216 = ps.OverlayValues[216]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d12)
			var d217 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d217 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(scratch, d12.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d217)
			}
			if d217.Loc == scm.LocImm {
				d217 = scm.JITValueDesc{Loc: scm.LocImm, Type: d217.Type, Imm: scm.NewInt(int64(uint64(d217.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d217.Reg, 32)
				ctx.W.EmitShrRegImm8(d217.Reg, 32)
			}
			if d217.Loc == scm.LocReg && d12.Loc == scm.LocReg && d217.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			d218 = d217
			if d218.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d218)
			d219 = d218
			if d219.Loc == scm.LocImm {
				d219 = scm.JITValueDesc{Loc: scm.LocImm, Type: d219.Type, Imm: scm.NewInt(int64(uint64(d219.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d219.Reg, 32)
				ctx.W.EmitShrRegImm8(d219.Reg, 32)
			}
			ctx.EmitStoreToStack(d219, 40)
			d220 = d1
			if d220.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d220)
			d221 = d220
			if d221.Loc == scm.LocImm {
				d221 = scm.JITValueDesc{Loc: scm.LocImm, Type: d221.Type, Imm: scm.NewInt(int64(uint64(d221.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d221.Reg, 32)
				ctx.W.EmitShrRegImm8(d221.Reg, 32)
			}
			ctx.EmitStoreToStack(d221, 48)
			d222 = d3
			if d222.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d222)
			d223 = d222
			if d223.Loc == scm.LocImm {
				d223 = scm.JITValueDesc{Loc: scm.LocImm, Type: d223.Type, Imm: scm.NewInt(int64(uint64(d223.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d223.Reg, 32)
				ctx.W.EmitShrRegImm8(d223.Reg, 32)
			}
			ctx.EmitStoreToStack(d223, 56)
			ps224 := scm.PhiState{General: ps.General}
			ps224.OverlayValues = make([]scm.JITValueDesc, 224)
			ps224.OverlayValues[0] = d0
			ps224.OverlayValues[1] = d1
			ps224.OverlayValues[2] = d2
			ps224.OverlayValues[3] = d3
			ps224.OverlayValues[4] = d4
			ps224.OverlayValues[5] = d5
			ps224.OverlayValues[6] = d6
			ps224.OverlayValues[7] = d7
			ps224.OverlayValues[8] = d8
			ps224.OverlayValues[9] = d9
			ps224.OverlayValues[10] = d10
			ps224.OverlayValues[11] = d11
			ps224.OverlayValues[12] = d12
			ps224.OverlayValues[13] = d13
			ps224.OverlayValues[14] = d14
			ps224.OverlayValues[20] = d20
			ps224.OverlayValues[21] = d21
			ps224.OverlayValues[22] = d22
			ps224.OverlayValues[24] = d24
			ps224.OverlayValues[25] = d25
			ps224.OverlayValues[27] = d27
			ps224.OverlayValues[28] = d28
			ps224.OverlayValues[29] = d29
			ps224.OverlayValues[32] = d32
			ps224.OverlayValues[34] = d34
			ps224.OverlayValues[35] = d35
			ps224.OverlayValues[36] = d36
			ps224.OverlayValues[38] = d38
			ps224.OverlayValues[39] = d39
			ps224.OverlayValues[40] = d40
			ps224.OverlayValues[41] = d41
			ps224.OverlayValues[42] = d42
			ps224.OverlayValues[43] = d43
			ps224.OverlayValues[44] = d44
			ps224.OverlayValues[46] = d46
			ps224.OverlayValues[47] = d47
			ps224.OverlayValues[48] = d48
			ps224.OverlayValues[49] = d49
			ps224.OverlayValues[50] = d50
			ps224.OverlayValues[51] = d51
			ps224.OverlayValues[52] = d52
			ps224.OverlayValues[53] = d53
			ps224.OverlayValues[54] = d54
			ps224.OverlayValues[55] = d55
			ps224.OverlayValues[56] = d56
			ps224.OverlayValues[57] = d57
			ps224.OverlayValues[58] = d58
			ps224.OverlayValues[59] = d59
			ps224.OverlayValues[60] = d60
			ps224.OverlayValues[61] = d61
			ps224.OverlayValues[62] = d62
			ps224.OverlayValues[63] = d63
			ps224.OverlayValues[64] = d64
			ps224.OverlayValues[65] = d65
			ps224.OverlayValues[66] = d66
			ps224.OverlayValues[67] = d67
			ps224.OverlayValues[68] = d68
			ps224.OverlayValues[69] = d69
			ps224.OverlayValues[70] = d70
			ps224.OverlayValues[71] = d71
			ps224.OverlayValues[72] = d72
			ps224.OverlayValues[73] = d73
			ps224.OverlayValues[74] = d74
			ps224.OverlayValues[75] = d75
			ps224.OverlayValues[76] = d76
			ps224.OverlayValues[77] = d77
			ps224.OverlayValues[78] = d78
			ps224.OverlayValues[79] = d79
			ps224.OverlayValues[80] = d80
			ps224.OverlayValues[81] = d81
			ps224.OverlayValues[82] = d82
			ps224.OverlayValues[83] = d83
			ps224.OverlayValues[84] = d84
			ps224.OverlayValues[85] = d85
			ps224.OverlayValues[86] = d86
			ps224.OverlayValues[87] = d87
			ps224.OverlayValues[88] = d88
			ps224.OverlayValues[89] = d89
			ps224.OverlayValues[90] = d90
			ps224.OverlayValues[91] = d91
			ps224.OverlayValues[92] = d92
			ps224.OverlayValues[93] = d93
			ps224.OverlayValues[94] = d94
			ps224.OverlayValues[95] = d95
			ps224.OverlayValues[102] = d102
			ps224.OverlayValues[103] = d103
			ps224.OverlayValues[104] = d104
			ps224.OverlayValues[105] = d105
			ps224.OverlayValues[106] = d106
			ps224.OverlayValues[107] = d107
			ps224.OverlayValues[108] = d108
			ps224.OverlayValues[109] = d109
			ps224.OverlayValues[110] = d110
			ps224.OverlayValues[111] = d111
			ps224.OverlayValues[112] = d112
			ps224.OverlayValues[113] = d113
			ps224.OverlayValues[114] = d114
			ps224.OverlayValues[115] = d115
			ps224.OverlayValues[116] = d116
			ps224.OverlayValues[117] = d117
			ps224.OverlayValues[118] = d118
			ps224.OverlayValues[119] = d119
			ps224.OverlayValues[120] = d120
			ps224.OverlayValues[121] = d121
			ps224.OverlayValues[122] = d122
			ps224.OverlayValues[123] = d123
			ps224.OverlayValues[124] = d124
			ps224.OverlayValues[125] = d125
			ps224.OverlayValues[126] = d126
			ps224.OverlayValues[127] = d127
			ps224.OverlayValues[128] = d128
			ps224.OverlayValues[129] = d129
			ps224.OverlayValues[130] = d130
			ps224.OverlayValues[131] = d131
			ps224.OverlayValues[132] = d132
			ps224.OverlayValues[133] = d133
			ps224.OverlayValues[134] = d134
			ps224.OverlayValues[135] = d135
			ps224.OverlayValues[136] = d136
			ps224.OverlayValues[137] = d137
			ps224.OverlayValues[138] = d138
			ps224.OverlayValues[139] = d139
			ps224.OverlayValues[140] = d140
			ps224.OverlayValues[141] = d141
			ps224.OverlayValues[142] = d142
			ps224.OverlayValues[143] = d143
			ps224.OverlayValues[144] = d144
			ps224.OverlayValues[145] = d145
			ps224.OverlayValues[146] = d146
			ps224.OverlayValues[153] = d153
			ps224.OverlayValues[154] = d154
			ps224.OverlayValues[160] = d160
			ps224.OverlayValues[161] = d161
			ps224.OverlayValues[162] = d162
			ps224.OverlayValues[163] = d163
			ps224.OverlayValues[164] = d164
			ps224.OverlayValues[165] = d165
			ps224.OverlayValues[166] = d166
			ps224.OverlayValues[168] = d168
			ps224.OverlayValues[170] = d170
			ps224.OverlayValues[171] = d171
			ps224.OverlayValues[174] = d174
			ps224.OverlayValues[177] = d177
			ps224.OverlayValues[178] = d178
			ps224.OverlayValues[179] = d179
			ps224.OverlayValues[181] = d181
			ps224.OverlayValues[182] = d182
			ps224.OverlayValues[183] = d183
			ps224.OverlayValues[184] = d184
			ps224.OverlayValues[185] = d185
			ps224.OverlayValues[186] = d186
			ps224.OverlayValues[188] = d188
			ps224.OverlayValues[189] = d189
			ps224.OverlayValues[190] = d190
			ps224.OverlayValues[191] = d191
			ps224.OverlayValues[192] = d192
			ps224.OverlayValues[193] = d193
			ps224.OverlayValues[194] = d194
			ps224.OverlayValues[195] = d195
			ps224.OverlayValues[196] = d196
			ps224.OverlayValues[199] = d199
			ps224.OverlayValues[200] = d200
			ps224.OverlayValues[201] = d201
			ps224.OverlayValues[204] = d204
			ps224.OverlayValues[205] = d205
			ps224.OverlayValues[206] = d206
			ps224.OverlayValues[207] = d207
			ps224.OverlayValues[208] = d208
			ps224.OverlayValues[209] = d209
			ps224.OverlayValues[210] = d210
			ps224.OverlayValues[211] = d211
			ps224.OverlayValues[212] = d212
			ps224.OverlayValues[214] = d214
			ps224.OverlayValues[215] = d215
			ps224.OverlayValues[216] = d216
			ps224.OverlayValues[217] = d217
			ps224.OverlayValues[218] = d218
			ps224.OverlayValues[219] = d219
			ps224.OverlayValues[220] = d220
			ps224.OverlayValues[221] = d221
			ps224.OverlayValues[222] = d222
			ps224.OverlayValues[223] = d223
			ps224.PhiValues = make([]scm.JITValueDesc, 3)
			d225 = d217
			ps224.PhiValues[0] = d225
			d226 = d1
			ps224.PhiValues[1] = d226
			d227 = d3
			ps224.PhiValues[2] = d227
			if ps224.General && bbs[8].Rendered {
				ctx.W.EmitJmp(lbl9)
				return result
			}
			return bbs[8].RenderPS(ps224)
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
					ctx.W.EmitJmp(lbl14)
					return result
				}
				bbs[13].Rendered = true
				bbs[13].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_13 = bbs[13].Address
				ctx.W.MarkLabel(lbl14)
				ctx.W.ResolveFixups()
			}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
				d154 = ps.OverlayValues[154]
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
			if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
				d168 = ps.OverlayValues[168]
			}
			if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != scm.LocNone {
				d170 = ps.OverlayValues[170]
			}
			if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != scm.LocNone {
				d171 = ps.OverlayValues[171]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
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
			if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
				d199 = ps.OverlayValues[199]
			}
			if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
				d200 = ps.OverlayValues[200]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
				d201 = ps.OverlayValues[201]
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
			if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != scm.LocNone {
				d225 = ps.OverlayValues[225]
			}
			if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != scm.LocNone {
				d226 = ps.OverlayValues[226]
			}
			if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
				d227 = ps.OverlayValues[227]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d5)
			d228 = d5
			_ = d228
			r108 := d5.Loc == scm.LocReg
			r109 := d5.Reg
			if r108 { ctx.ProtectReg(r109) }
			d229 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(96)}
			lbl54 := ctx.W.ReserveLabel()
			bbpos_3_0 := int32(-1)
			_ = bbpos_3_0
			bbpos_3_1 := int32(-1)
			_ = bbpos_3_1
			bbpos_3_2 := int32(-1)
			_ = bbpos_3_2
			bbpos_3_3 := int32(-1)
			_ = bbpos_3_3
			bbpos_3_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			d229 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(96)}
			ctx.EnsureDesc(&d228)
			ctx.EnsureDesc(&d228)
			var d230 scm.JITValueDesc
			if d228.Loc == scm.LocImm {
				d230 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d228.Imm.Int()))))}
			} else {
				r110 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r110, d228.Reg)
				ctx.W.EmitShlRegImm8(r110, 32)
				ctx.W.EmitShrRegImm8(r110, 32)
				d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
				ctx.BindReg(r110, &d230)
			}
			var d231 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d231 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r111 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r111, thisptr.Reg, off)
				d231 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r111}
				ctx.BindReg(r111, &d231)
			}
			ctx.EnsureDesc(&d231)
			ctx.EnsureDesc(&d231)
			var d232 scm.JITValueDesc
			if d231.Loc == scm.LocImm {
				d232 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d231.Imm.Int()))))}
			} else {
				r112 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r112, d231.Reg)
				ctx.W.EmitShlRegImm8(r112, 56)
				ctx.W.EmitShrRegImm8(r112, 56)
				d232 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r112}
				ctx.BindReg(r112, &d232)
			}
			ctx.FreeDesc(&d231)
			ctx.EnsureDesc(&d230)
			ctx.EnsureDesc(&d232)
			ctx.EnsureDesc(&d230)
			ctx.EnsureDesc(&d232)
			ctx.EnsureDesc(&d230)
			ctx.EnsureDesc(&d232)
			var d233 scm.JITValueDesc
			if d230.Loc == scm.LocImm && d232.Loc == scm.LocImm {
				d233 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d230.Imm.Int() * d232.Imm.Int())}
			} else if d230.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d232.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d230.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d232.Reg)
				d233 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d233)
			} else if d232.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d230.Reg)
				ctx.W.EmitMovRegReg(scratch, d230.Reg)
				if d232.Imm.Int() >= -2147483648 && d232.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d232.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d232.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d233 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d233)
			} else {
				r113 := ctx.AllocRegExcept(d230.Reg, d232.Reg)
				ctx.W.EmitMovRegReg(r113, d230.Reg)
				ctx.W.EmitImulInt64(r113, d232.Reg)
				d233 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r113}
				ctx.BindReg(r113, &d233)
			}
			if d233.Loc == scm.LocReg && d230.Loc == scm.LocReg && d233.Reg == d230.Reg {
				ctx.TransferReg(d230.Reg)
				d230.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d230)
			ctx.FreeDesc(&d232)
			var d234 scm.JITValueDesc
			r114 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r114, uint64(dataPtr))
				d234 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r114, StackOff: int32(sliceLen)}
				ctx.BindReg(r114, &d234)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				ctx.W.EmitMovRegMem(r114, thisptr.Reg, off)
				d234 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r114}
				ctx.BindReg(r114, &d234)
			}
			ctx.BindReg(r114, &d234)
			ctx.EnsureDesc(&d233)
			var d235 scm.JITValueDesc
			if d233.Loc == scm.LocImm {
				d235 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d233.Imm.Int() / 64)}
			} else {
				r115 := ctx.AllocRegExcept(d233.Reg)
				ctx.W.EmitMovRegReg(r115, d233.Reg)
				ctx.W.EmitShrRegImm8(r115, 6)
				d235 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
				ctx.BindReg(r115, &d235)
			}
			if d235.Loc == scm.LocReg && d233.Loc == scm.LocReg && d235.Reg == d233.Reg {
				ctx.TransferReg(d233.Reg)
				d233.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d235)
			r116 := ctx.AllocReg()
			ctx.EnsureDesc(&d235)
			ctx.EnsureDesc(&d234)
			if d235.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r116, uint64(d235.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r116, d235.Reg)
				ctx.W.EmitShlRegImm8(r116, 3)
			}
			if d234.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d234.Imm.Int()))
				ctx.W.EmitAddInt64(r116, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r116, d234.Reg)
			}
			r117 := ctx.AllocRegExcept(r116)
			ctx.W.EmitMovRegMem(r117, r116, 0)
			ctx.FreeReg(r116)
			d236 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r117}
			ctx.BindReg(r117, &d236)
			ctx.FreeDesc(&d235)
			ctx.EnsureDesc(&d233)
			var d237 scm.JITValueDesc
			if d233.Loc == scm.LocImm {
				d237 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d233.Imm.Int() % 64)}
			} else {
				r118 := ctx.AllocRegExcept(d233.Reg)
				ctx.W.EmitMovRegReg(r118, d233.Reg)
				ctx.W.EmitAndRegImm32(r118, 63)
				d237 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r118}
				ctx.BindReg(r118, &d237)
			}
			if d237.Loc == scm.LocReg && d233.Loc == scm.LocReg && d237.Reg == d233.Reg {
				ctx.TransferReg(d233.Reg)
				d233.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d236)
			ctx.EnsureDesc(&d237)
			var d238 scm.JITValueDesc
			if d236.Loc == scm.LocImm && d237.Loc == scm.LocImm {
				d238 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d236.Imm.Int()) << uint64(d237.Imm.Int())))}
			} else if d237.Loc == scm.LocImm {
				r119 := ctx.AllocRegExcept(d236.Reg)
				ctx.W.EmitMovRegReg(r119, d236.Reg)
				ctx.W.EmitShlRegImm8(r119, uint8(d237.Imm.Int()))
				d238 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
				ctx.BindReg(r119, &d238)
			} else {
				{
					shiftSrc := d236.Reg
					r120 := ctx.AllocRegExcept(d236.Reg)
					ctx.W.EmitMovRegReg(r120, d236.Reg)
					shiftSrc = r120
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d237.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d237.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d237.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d238 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d238)
				}
			}
			if d238.Loc == scm.LocReg && d236.Loc == scm.LocReg && d238.Reg == d236.Reg {
				ctx.TransferReg(d236.Reg)
				d236.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d236)
			ctx.FreeDesc(&d237)
			var d239 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d239 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
				r121 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r121, thisptr.Reg, off)
				d239 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r121}
				ctx.BindReg(r121, &d239)
			}
			d240 = d239
			ctx.EnsureDesc(&d240)
			if d240.Loc != scm.LocImm && d240.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl55 := ctx.W.ReserveLabel()
			lbl56 := ctx.W.ReserveLabel()
			lbl57 := ctx.W.ReserveLabel()
			lbl58 := ctx.W.ReserveLabel()
			if d240.Loc == scm.LocImm {
				if d240.Imm.Bool() {
					ctx.W.MarkLabel(lbl57)
					ctx.W.EmitJmp(lbl55)
				} else {
					ctx.W.MarkLabel(lbl58)
			d241 = d238
			if d241.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d241)
			ctx.EmitStoreToStack(d241, 96)
					ctx.W.EmitJmp(lbl56)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d240.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl57)
				ctx.W.EmitJmp(lbl58)
				ctx.W.MarkLabel(lbl57)
				ctx.W.EmitJmp(lbl55)
				ctx.W.MarkLabel(lbl58)
			d242 = d238
			if d242.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d242)
			ctx.EmitStoreToStack(d242, 96)
				ctx.W.EmitJmp(lbl56)
			}
			ctx.FreeDesc(&d239)
			bbpos_3_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl56)
			ctx.W.ResolveFixups()
			d229 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(96)}
			var d243 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d243 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r122 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r122, thisptr.Reg, off)
				d243 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r122}
				ctx.BindReg(r122, &d243)
			}
			ctx.EnsureDesc(&d243)
			ctx.EnsureDesc(&d243)
			var d244 scm.JITValueDesc
			if d243.Loc == scm.LocImm {
				d244 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d243.Imm.Int()))))}
			} else {
				r123 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r123, d243.Reg)
				ctx.W.EmitShlRegImm8(r123, 56)
				ctx.W.EmitShrRegImm8(r123, 56)
				d244 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
				ctx.BindReg(r123, &d244)
			}
			ctx.FreeDesc(&d243)
			d245 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d244)
			ctx.EnsureDesc(&d245)
			ctx.EnsureDesc(&d244)
			ctx.EnsureDesc(&d245)
			ctx.EnsureDesc(&d244)
			var d246 scm.JITValueDesc
			if d245.Loc == scm.LocImm && d244.Loc == scm.LocImm {
				d246 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d245.Imm.Int() - d244.Imm.Int())}
			} else if d244.Loc == scm.LocImm && d244.Imm.Int() == 0 {
				r124 := ctx.AllocRegExcept(d245.Reg)
				ctx.W.EmitMovRegReg(r124, d245.Reg)
				d246 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
				ctx.BindReg(r124, &d246)
			} else if d245.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d244.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d245.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d244.Reg)
				d246 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d246)
			} else if d244.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d245.Reg)
				ctx.W.EmitMovRegReg(scratch, d245.Reg)
				if d244.Imm.Int() >= -2147483648 && d244.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d244.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d244.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d246 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d246)
			} else {
				r125 := ctx.AllocRegExcept(d245.Reg, d244.Reg)
				ctx.W.EmitMovRegReg(r125, d245.Reg)
				ctx.W.EmitSubInt64(r125, d244.Reg)
				d246 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
				ctx.BindReg(r125, &d246)
			}
			if d246.Loc == scm.LocReg && d245.Loc == scm.LocReg && d246.Reg == d245.Reg {
				ctx.TransferReg(d245.Reg)
				d245.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d244)
			ctx.EnsureDesc(&d229)
			ctx.EnsureDesc(&d246)
			var d247 scm.JITValueDesc
			if d229.Loc == scm.LocImm && d246.Loc == scm.LocImm {
				d247 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d229.Imm.Int()) >> uint64(d246.Imm.Int())))}
			} else if d246.Loc == scm.LocImm {
				r126 := ctx.AllocRegExcept(d229.Reg)
				ctx.W.EmitMovRegReg(r126, d229.Reg)
				ctx.W.EmitShrRegImm8(r126, uint8(d246.Imm.Int()))
				d247 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r126}
				ctx.BindReg(r126, &d247)
			} else {
				{
					shiftSrc := d229.Reg
					r127 := ctx.AllocRegExcept(d229.Reg)
					ctx.W.EmitMovRegReg(r127, d229.Reg)
					shiftSrc = r127
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d246.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d246.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d246.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d247 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d247)
				}
			}
			if d247.Loc == scm.LocReg && d229.Loc == scm.LocReg && d247.Reg == d229.Reg {
				ctx.TransferReg(d229.Reg)
				d229.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d229)
			ctx.FreeDesc(&d246)
			r128 := ctx.AllocReg()
			ctx.EnsureDesc(&d247)
			ctx.EnsureDesc(&d247)
			if d247.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r128, d247)
			}
			ctx.W.EmitJmp(lbl54)
			bbpos_3_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl55)
			ctx.W.ResolveFixups()
			d229 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(96)}
			ctx.EnsureDesc(&d233)
			var d248 scm.JITValueDesc
			if d233.Loc == scm.LocImm {
				d248 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d233.Imm.Int() % 64)}
			} else {
				r129 := ctx.AllocRegExcept(d233.Reg)
				ctx.W.EmitMovRegReg(r129, d233.Reg)
				ctx.W.EmitAndRegImm32(r129, 63)
				d248 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
				ctx.BindReg(r129, &d248)
			}
			if d248.Loc == scm.LocReg && d233.Loc == scm.LocReg && d248.Reg == d233.Reg {
				ctx.TransferReg(d233.Reg)
				d233.Loc = scm.LocNone
			}
			var d249 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d249 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r130 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r130, thisptr.Reg, off)
				d249 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r130}
				ctx.BindReg(r130, &d249)
			}
			ctx.EnsureDesc(&d249)
			ctx.EnsureDesc(&d249)
			var d250 scm.JITValueDesc
			if d249.Loc == scm.LocImm {
				d250 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d249.Imm.Int()))))}
			} else {
				r131 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r131, d249.Reg)
				ctx.W.EmitShlRegImm8(r131, 56)
				ctx.W.EmitShrRegImm8(r131, 56)
				d250 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r131}
				ctx.BindReg(r131, &d250)
			}
			ctx.FreeDesc(&d249)
			ctx.EnsureDesc(&d248)
			ctx.EnsureDesc(&d250)
			ctx.EnsureDesc(&d248)
			ctx.EnsureDesc(&d250)
			ctx.EnsureDesc(&d248)
			ctx.EnsureDesc(&d250)
			var d251 scm.JITValueDesc
			if d248.Loc == scm.LocImm && d250.Loc == scm.LocImm {
				d251 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d248.Imm.Int() + d250.Imm.Int())}
			} else if d250.Loc == scm.LocImm && d250.Imm.Int() == 0 {
				r132 := ctx.AllocRegExcept(d248.Reg)
				ctx.W.EmitMovRegReg(r132, d248.Reg)
				d251 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
				ctx.BindReg(r132, &d251)
			} else if d248.Loc == scm.LocImm && d248.Imm.Int() == 0 {
				d251 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d250.Reg}
				ctx.BindReg(d250.Reg, &d251)
			} else if d248.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d250.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d248.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d250.Reg)
				d251 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d251)
			} else if d250.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d248.Reg)
				ctx.W.EmitMovRegReg(scratch, d248.Reg)
				if d250.Imm.Int() >= -2147483648 && d250.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d250.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d250.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d251 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d251)
			} else {
				r133 := ctx.AllocRegExcept(d248.Reg, d250.Reg)
				ctx.W.EmitMovRegReg(r133, d248.Reg)
				ctx.W.EmitAddInt64(r133, d250.Reg)
				d251 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
				ctx.BindReg(r133, &d251)
			}
			if d251.Loc == scm.LocReg && d248.Loc == scm.LocReg && d251.Reg == d248.Reg {
				ctx.TransferReg(d248.Reg)
				d248.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d248)
			ctx.FreeDesc(&d250)
			ctx.EnsureDesc(&d251)
			var d252 scm.JITValueDesc
			if d251.Loc == scm.LocImm {
				d252 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d251.Imm.Int()) > uint64(64))}
			} else {
				r134 := ctx.AllocRegExcept(d251.Reg)
				ctx.W.EmitCmpRegImm32(d251.Reg, 64)
				ctx.W.EmitSetcc(r134, scm.CcA)
				d252 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r134}
				ctx.BindReg(r134, &d252)
			}
			ctx.FreeDesc(&d251)
			d253 = d252
			ctx.EnsureDesc(&d253)
			if d253.Loc != scm.LocImm && d253.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl59 := ctx.W.ReserveLabel()
			lbl60 := ctx.W.ReserveLabel()
			lbl61 := ctx.W.ReserveLabel()
			if d253.Loc == scm.LocImm {
				if d253.Imm.Bool() {
					ctx.W.MarkLabel(lbl60)
					ctx.W.EmitJmp(lbl59)
				} else {
					ctx.W.MarkLabel(lbl61)
			d254 = d238
			if d254.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d254)
			ctx.EmitStoreToStack(d254, 96)
					ctx.W.EmitJmp(lbl56)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d253.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl60)
				ctx.W.EmitJmp(lbl61)
				ctx.W.MarkLabel(lbl60)
				ctx.W.EmitJmp(lbl59)
				ctx.W.MarkLabel(lbl61)
			d255 = d238
			if d255.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d255)
			ctx.EmitStoreToStack(d255, 96)
				ctx.W.EmitJmp(lbl56)
			}
			ctx.FreeDesc(&d252)
			bbpos_3_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl59)
			ctx.W.ResolveFixups()
			d229 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(96)}
			ctx.EnsureDesc(&d233)
			var d256 scm.JITValueDesc
			if d233.Loc == scm.LocImm {
				d256 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d233.Imm.Int() / 64)}
			} else {
				r135 := ctx.AllocRegExcept(d233.Reg)
				ctx.W.EmitMovRegReg(r135, d233.Reg)
				ctx.W.EmitShrRegImm8(r135, 6)
				d256 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r135}
				ctx.BindReg(r135, &d256)
			}
			if d256.Loc == scm.LocReg && d233.Loc == scm.LocReg && d256.Reg == d233.Reg {
				ctx.TransferReg(d233.Reg)
				d233.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d256)
			ctx.EnsureDesc(&d256)
			var d257 scm.JITValueDesc
			if d256.Loc == scm.LocImm {
				d257 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d256.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d256.Reg)
				ctx.W.EmitMovRegReg(scratch, d256.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d257 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d257)
			}
			if d257.Loc == scm.LocReg && d256.Loc == scm.LocReg && d257.Reg == d256.Reg {
				ctx.TransferReg(d256.Reg)
				d256.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d256)
			ctx.EnsureDesc(&d257)
			r136 := ctx.AllocReg()
			ctx.EnsureDesc(&d257)
			ctx.EnsureDesc(&d234)
			if d257.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r136, uint64(d257.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r136, d257.Reg)
				ctx.W.EmitShlRegImm8(r136, 3)
			}
			if d234.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d234.Imm.Int()))
				ctx.W.EmitAddInt64(r136, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r136, d234.Reg)
			}
			r137 := ctx.AllocRegExcept(r136)
			ctx.W.EmitMovRegMem(r137, r136, 0)
			ctx.FreeReg(r136)
			d258 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r137}
			ctx.BindReg(r137, &d258)
			ctx.FreeDesc(&d257)
			ctx.EnsureDesc(&d233)
			var d259 scm.JITValueDesc
			if d233.Loc == scm.LocImm {
				d259 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d233.Imm.Int() % 64)}
			} else {
				r138 := ctx.AllocRegExcept(d233.Reg)
				ctx.W.EmitMovRegReg(r138, d233.Reg)
				ctx.W.EmitAndRegImm32(r138, 63)
				d259 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
				ctx.BindReg(r138, &d259)
			}
			if d259.Loc == scm.LocReg && d233.Loc == scm.LocReg && d259.Reg == d233.Reg {
				ctx.TransferReg(d233.Reg)
				d233.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d233)
			d260 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d259)
			ctx.EnsureDesc(&d260)
			ctx.EnsureDesc(&d259)
			ctx.EnsureDesc(&d260)
			ctx.EnsureDesc(&d259)
			var d261 scm.JITValueDesc
			if d260.Loc == scm.LocImm && d259.Loc == scm.LocImm {
				d261 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d260.Imm.Int() - d259.Imm.Int())}
			} else if d259.Loc == scm.LocImm && d259.Imm.Int() == 0 {
				r139 := ctx.AllocRegExcept(d260.Reg)
				ctx.W.EmitMovRegReg(r139, d260.Reg)
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r139}
				ctx.BindReg(r139, &d261)
			} else if d260.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d259.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d260.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d259.Reg)
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d261)
			} else if d259.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d260.Reg)
				ctx.W.EmitMovRegReg(scratch, d260.Reg)
				if d259.Imm.Int() >= -2147483648 && d259.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d259.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d259.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d261)
			} else {
				r140 := ctx.AllocRegExcept(d260.Reg, d259.Reg)
				ctx.W.EmitMovRegReg(r140, d260.Reg)
				ctx.W.EmitSubInt64(r140, d259.Reg)
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r140}
				ctx.BindReg(r140, &d261)
			}
			if d261.Loc == scm.LocReg && d260.Loc == scm.LocReg && d261.Reg == d260.Reg {
				ctx.TransferReg(d260.Reg)
				d260.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d259)
			ctx.EnsureDesc(&d258)
			ctx.EnsureDesc(&d261)
			var d262 scm.JITValueDesc
			if d258.Loc == scm.LocImm && d261.Loc == scm.LocImm {
				d262 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d258.Imm.Int()) >> uint64(d261.Imm.Int())))}
			} else if d261.Loc == scm.LocImm {
				r141 := ctx.AllocRegExcept(d258.Reg)
				ctx.W.EmitMovRegReg(r141, d258.Reg)
				ctx.W.EmitShrRegImm8(r141, uint8(d261.Imm.Int()))
				d262 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
				ctx.BindReg(r141, &d262)
			} else {
				{
					shiftSrc := d258.Reg
					r142 := ctx.AllocRegExcept(d258.Reg)
					ctx.W.EmitMovRegReg(r142, d258.Reg)
					shiftSrc = r142
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d261.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d261.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d261.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d262 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d262)
				}
			}
			if d262.Loc == scm.LocReg && d258.Loc == scm.LocReg && d262.Reg == d258.Reg {
				ctx.TransferReg(d258.Reg)
				d258.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d258)
			ctx.FreeDesc(&d261)
			ctx.EnsureDesc(&d238)
			ctx.EnsureDesc(&d262)
			var d263 scm.JITValueDesc
			if d238.Loc == scm.LocImm && d262.Loc == scm.LocImm {
				d263 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d238.Imm.Int() | d262.Imm.Int())}
			} else if d238.Loc == scm.LocImm && d238.Imm.Int() == 0 {
				d263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d262.Reg}
				ctx.BindReg(d262.Reg, &d263)
			} else if d262.Loc == scm.LocImm && d262.Imm.Int() == 0 {
				r143 := ctx.AllocRegExcept(d238.Reg)
				ctx.W.EmitMovRegReg(r143, d238.Reg)
				d263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r143}
				ctx.BindReg(r143, &d263)
			} else if d238.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d262.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d238.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d262.Reg)
				d263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d263)
			} else if d262.Loc == scm.LocImm {
				r144 := ctx.AllocRegExcept(d238.Reg)
				ctx.W.EmitMovRegReg(r144, d238.Reg)
				if d262.Imm.Int() >= -2147483648 && d262.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r144, int32(d262.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d262.Imm.Int()))
					ctx.W.EmitOrInt64(r144, scm.RegR11)
				}
				d263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
				ctx.BindReg(r144, &d263)
			} else {
				r145 := ctx.AllocRegExcept(d238.Reg, d262.Reg)
				ctx.W.EmitMovRegReg(r145, d238.Reg)
				ctx.W.EmitOrInt64(r145, d262.Reg)
				d263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r145}
				ctx.BindReg(r145, &d263)
			}
			if d263.Loc == scm.LocReg && d238.Loc == scm.LocReg && d263.Reg == d238.Reg {
				ctx.TransferReg(d238.Reg)
				d238.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d262)
			d264 = d263
			if d264.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d264)
			ctx.EmitStoreToStack(d264, 96)
			ctx.W.EmitJmp(lbl56)
			ctx.W.MarkLabel(lbl54)
			d265 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r128}
			ctx.BindReg(r128, &d265)
			ctx.BindReg(r128, &d265)
			if r108 { ctx.UnprotectReg(r109) }
			ctx.EnsureDesc(&d265)
			ctx.EnsureDesc(&d265)
			var d266 scm.JITValueDesc
			if d265.Loc == scm.LocImm {
				d266 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d265.Imm.Int()))))}
			} else {
				r146 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r146, d265.Reg)
				d266 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r146}
				ctx.BindReg(r146, &d266)
			}
			ctx.FreeDesc(&d265)
			var d267 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d267 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r147 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r147, thisptr.Reg, off)
				d267 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r147}
				ctx.BindReg(r147, &d267)
			}
			ctx.EnsureDesc(&d266)
			ctx.EnsureDesc(&d267)
			ctx.EnsureDesc(&d266)
			ctx.EnsureDesc(&d267)
			ctx.EnsureDesc(&d266)
			ctx.EnsureDesc(&d267)
			var d268 scm.JITValueDesc
			if d266.Loc == scm.LocImm && d267.Loc == scm.LocImm {
				d268 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d266.Imm.Int() + d267.Imm.Int())}
			} else if d267.Loc == scm.LocImm && d267.Imm.Int() == 0 {
				r148 := ctx.AllocRegExcept(d266.Reg)
				ctx.W.EmitMovRegReg(r148, d266.Reg)
				d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r148}
				ctx.BindReg(r148, &d268)
			} else if d266.Loc == scm.LocImm && d266.Imm.Int() == 0 {
				d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d267.Reg}
				ctx.BindReg(d267.Reg, &d268)
			} else if d266.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d267.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d266.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d267.Reg)
				d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d268)
			} else if d267.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d266.Reg)
				ctx.W.EmitMovRegReg(scratch, d266.Reg)
				if d267.Imm.Int() >= -2147483648 && d267.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d267.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d267.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d268)
			} else {
				r149 := ctx.AllocRegExcept(d266.Reg, d267.Reg)
				ctx.W.EmitMovRegReg(r149, d266.Reg)
				ctx.W.EmitAddInt64(r149, d267.Reg)
				d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r149}
				ctx.BindReg(r149, &d268)
			}
			if d268.Loc == scm.LocReg && d266.Loc == scm.LocReg && d268.Reg == d266.Reg {
				ctx.TransferReg(d266.Reg)
				d266.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d266)
			ctx.FreeDesc(&d267)
			ctx.EnsureDesc(&d268)
			ctx.EnsureDesc(&d268)
			var d269 scm.JITValueDesc
			if d268.Loc == scm.LocImm {
				d269 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d268.Imm.Int()))))}
			} else {
				r150 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r150, d268.Reg)
				ctx.W.EmitShlRegImm8(r150, 32)
				ctx.W.EmitShrRegImm8(r150, 32)
				d269 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r150}
				ctx.BindReg(r150, &d269)
			}
			ctx.FreeDesc(&d268)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d269)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d269)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d269)
			var d270 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d269.Loc == scm.LocImm {
				d270 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d269.Imm.Int()))}
			} else if d269.Loc == scm.LocImm {
				r151 := ctx.AllocRegExcept(idxInt.Reg)
				if d269.Imm.Int() >= -2147483648 && d269.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(idxInt.Reg, int32(d269.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d269.Imm.Int()))
					ctx.W.EmitCmpInt64(idxInt.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r151, scm.CcB)
				d270 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r151}
				ctx.BindReg(r151, &d270)
			} else if idxInt.Loc == scm.LocImm {
				r152 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d269.Reg)
				ctx.W.EmitSetcc(r152, scm.CcB)
				d270 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r152}
				ctx.BindReg(r152, &d270)
			} else {
				r153 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitCmpInt64(idxInt.Reg, d269.Reg)
				ctx.W.EmitSetcc(r153, scm.CcB)
				d270 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r153}
				ctx.BindReg(r153, &d270)
			}
			ctx.FreeDesc(&d269)
			d271 = d270
			ctx.EnsureDesc(&d271)
			if d271.Loc != scm.LocImm && d271.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d271.Loc == scm.LocImm {
				if d271.Imm.Bool() {
			ps272 := scm.PhiState{General: ps.General}
			ps272.OverlayValues = make([]scm.JITValueDesc, 272)
			ps272.OverlayValues[0] = d0
			ps272.OverlayValues[1] = d1
			ps272.OverlayValues[2] = d2
			ps272.OverlayValues[3] = d3
			ps272.OverlayValues[4] = d4
			ps272.OverlayValues[5] = d5
			ps272.OverlayValues[6] = d6
			ps272.OverlayValues[7] = d7
			ps272.OverlayValues[8] = d8
			ps272.OverlayValues[9] = d9
			ps272.OverlayValues[10] = d10
			ps272.OverlayValues[11] = d11
			ps272.OverlayValues[12] = d12
			ps272.OverlayValues[13] = d13
			ps272.OverlayValues[14] = d14
			ps272.OverlayValues[20] = d20
			ps272.OverlayValues[21] = d21
			ps272.OverlayValues[22] = d22
			ps272.OverlayValues[24] = d24
			ps272.OverlayValues[25] = d25
			ps272.OverlayValues[27] = d27
			ps272.OverlayValues[28] = d28
			ps272.OverlayValues[29] = d29
			ps272.OverlayValues[32] = d32
			ps272.OverlayValues[34] = d34
			ps272.OverlayValues[35] = d35
			ps272.OverlayValues[36] = d36
			ps272.OverlayValues[38] = d38
			ps272.OverlayValues[39] = d39
			ps272.OverlayValues[40] = d40
			ps272.OverlayValues[41] = d41
			ps272.OverlayValues[42] = d42
			ps272.OverlayValues[43] = d43
			ps272.OverlayValues[44] = d44
			ps272.OverlayValues[46] = d46
			ps272.OverlayValues[47] = d47
			ps272.OverlayValues[48] = d48
			ps272.OverlayValues[49] = d49
			ps272.OverlayValues[50] = d50
			ps272.OverlayValues[51] = d51
			ps272.OverlayValues[52] = d52
			ps272.OverlayValues[53] = d53
			ps272.OverlayValues[54] = d54
			ps272.OverlayValues[55] = d55
			ps272.OverlayValues[56] = d56
			ps272.OverlayValues[57] = d57
			ps272.OverlayValues[58] = d58
			ps272.OverlayValues[59] = d59
			ps272.OverlayValues[60] = d60
			ps272.OverlayValues[61] = d61
			ps272.OverlayValues[62] = d62
			ps272.OverlayValues[63] = d63
			ps272.OverlayValues[64] = d64
			ps272.OverlayValues[65] = d65
			ps272.OverlayValues[66] = d66
			ps272.OverlayValues[67] = d67
			ps272.OverlayValues[68] = d68
			ps272.OverlayValues[69] = d69
			ps272.OverlayValues[70] = d70
			ps272.OverlayValues[71] = d71
			ps272.OverlayValues[72] = d72
			ps272.OverlayValues[73] = d73
			ps272.OverlayValues[74] = d74
			ps272.OverlayValues[75] = d75
			ps272.OverlayValues[76] = d76
			ps272.OverlayValues[77] = d77
			ps272.OverlayValues[78] = d78
			ps272.OverlayValues[79] = d79
			ps272.OverlayValues[80] = d80
			ps272.OverlayValues[81] = d81
			ps272.OverlayValues[82] = d82
			ps272.OverlayValues[83] = d83
			ps272.OverlayValues[84] = d84
			ps272.OverlayValues[85] = d85
			ps272.OverlayValues[86] = d86
			ps272.OverlayValues[87] = d87
			ps272.OverlayValues[88] = d88
			ps272.OverlayValues[89] = d89
			ps272.OverlayValues[90] = d90
			ps272.OverlayValues[91] = d91
			ps272.OverlayValues[92] = d92
			ps272.OverlayValues[93] = d93
			ps272.OverlayValues[94] = d94
			ps272.OverlayValues[95] = d95
			ps272.OverlayValues[102] = d102
			ps272.OverlayValues[103] = d103
			ps272.OverlayValues[104] = d104
			ps272.OverlayValues[105] = d105
			ps272.OverlayValues[106] = d106
			ps272.OverlayValues[107] = d107
			ps272.OverlayValues[108] = d108
			ps272.OverlayValues[109] = d109
			ps272.OverlayValues[110] = d110
			ps272.OverlayValues[111] = d111
			ps272.OverlayValues[112] = d112
			ps272.OverlayValues[113] = d113
			ps272.OverlayValues[114] = d114
			ps272.OverlayValues[115] = d115
			ps272.OverlayValues[116] = d116
			ps272.OverlayValues[117] = d117
			ps272.OverlayValues[118] = d118
			ps272.OverlayValues[119] = d119
			ps272.OverlayValues[120] = d120
			ps272.OverlayValues[121] = d121
			ps272.OverlayValues[122] = d122
			ps272.OverlayValues[123] = d123
			ps272.OverlayValues[124] = d124
			ps272.OverlayValues[125] = d125
			ps272.OverlayValues[126] = d126
			ps272.OverlayValues[127] = d127
			ps272.OverlayValues[128] = d128
			ps272.OverlayValues[129] = d129
			ps272.OverlayValues[130] = d130
			ps272.OverlayValues[131] = d131
			ps272.OverlayValues[132] = d132
			ps272.OverlayValues[133] = d133
			ps272.OverlayValues[134] = d134
			ps272.OverlayValues[135] = d135
			ps272.OverlayValues[136] = d136
			ps272.OverlayValues[137] = d137
			ps272.OverlayValues[138] = d138
			ps272.OverlayValues[139] = d139
			ps272.OverlayValues[140] = d140
			ps272.OverlayValues[141] = d141
			ps272.OverlayValues[142] = d142
			ps272.OverlayValues[143] = d143
			ps272.OverlayValues[144] = d144
			ps272.OverlayValues[145] = d145
			ps272.OverlayValues[146] = d146
			ps272.OverlayValues[153] = d153
			ps272.OverlayValues[154] = d154
			ps272.OverlayValues[160] = d160
			ps272.OverlayValues[161] = d161
			ps272.OverlayValues[162] = d162
			ps272.OverlayValues[163] = d163
			ps272.OverlayValues[164] = d164
			ps272.OverlayValues[165] = d165
			ps272.OverlayValues[166] = d166
			ps272.OverlayValues[168] = d168
			ps272.OverlayValues[170] = d170
			ps272.OverlayValues[171] = d171
			ps272.OverlayValues[174] = d174
			ps272.OverlayValues[177] = d177
			ps272.OverlayValues[178] = d178
			ps272.OverlayValues[179] = d179
			ps272.OverlayValues[181] = d181
			ps272.OverlayValues[182] = d182
			ps272.OverlayValues[183] = d183
			ps272.OverlayValues[184] = d184
			ps272.OverlayValues[185] = d185
			ps272.OverlayValues[186] = d186
			ps272.OverlayValues[188] = d188
			ps272.OverlayValues[189] = d189
			ps272.OverlayValues[190] = d190
			ps272.OverlayValues[191] = d191
			ps272.OverlayValues[192] = d192
			ps272.OverlayValues[193] = d193
			ps272.OverlayValues[194] = d194
			ps272.OverlayValues[195] = d195
			ps272.OverlayValues[196] = d196
			ps272.OverlayValues[199] = d199
			ps272.OverlayValues[200] = d200
			ps272.OverlayValues[201] = d201
			ps272.OverlayValues[204] = d204
			ps272.OverlayValues[205] = d205
			ps272.OverlayValues[206] = d206
			ps272.OverlayValues[207] = d207
			ps272.OverlayValues[208] = d208
			ps272.OverlayValues[209] = d209
			ps272.OverlayValues[210] = d210
			ps272.OverlayValues[211] = d211
			ps272.OverlayValues[212] = d212
			ps272.OverlayValues[214] = d214
			ps272.OverlayValues[215] = d215
			ps272.OverlayValues[216] = d216
			ps272.OverlayValues[217] = d217
			ps272.OverlayValues[218] = d218
			ps272.OverlayValues[219] = d219
			ps272.OverlayValues[220] = d220
			ps272.OverlayValues[221] = d221
			ps272.OverlayValues[222] = d222
			ps272.OverlayValues[223] = d223
			ps272.OverlayValues[225] = d225
			ps272.OverlayValues[226] = d226
			ps272.OverlayValues[227] = d227
			ps272.OverlayValues[228] = d228
			ps272.OverlayValues[229] = d229
			ps272.OverlayValues[230] = d230
			ps272.OverlayValues[231] = d231
			ps272.OverlayValues[232] = d232
			ps272.OverlayValues[233] = d233
			ps272.OverlayValues[234] = d234
			ps272.OverlayValues[235] = d235
			ps272.OverlayValues[236] = d236
			ps272.OverlayValues[237] = d237
			ps272.OverlayValues[238] = d238
			ps272.OverlayValues[239] = d239
			ps272.OverlayValues[240] = d240
			ps272.OverlayValues[241] = d241
			ps272.OverlayValues[242] = d242
			ps272.OverlayValues[243] = d243
			ps272.OverlayValues[244] = d244
			ps272.OverlayValues[245] = d245
			ps272.OverlayValues[246] = d246
			ps272.OverlayValues[247] = d247
			ps272.OverlayValues[248] = d248
			ps272.OverlayValues[249] = d249
			ps272.OverlayValues[250] = d250
			ps272.OverlayValues[251] = d251
			ps272.OverlayValues[252] = d252
			ps272.OverlayValues[253] = d253
			ps272.OverlayValues[254] = d254
			ps272.OverlayValues[255] = d255
			ps272.OverlayValues[256] = d256
			ps272.OverlayValues[257] = d257
			ps272.OverlayValues[258] = d258
			ps272.OverlayValues[259] = d259
			ps272.OverlayValues[260] = d260
			ps272.OverlayValues[261] = d261
			ps272.OverlayValues[262] = d262
			ps272.OverlayValues[263] = d263
			ps272.OverlayValues[264] = d264
			ps272.OverlayValues[265] = d265
			ps272.OverlayValues[266] = d266
			ps272.OverlayValues[267] = d267
			ps272.OverlayValues[268] = d268
			ps272.OverlayValues[269] = d269
			ps272.OverlayValues[270] = d270
			ps272.OverlayValues[271] = d271
					return bbs[14].RenderPS(ps272)
				}
			ps273 := scm.PhiState{General: ps.General}
			ps273.OverlayValues = make([]scm.JITValueDesc, 272)
			ps273.OverlayValues[0] = d0
			ps273.OverlayValues[1] = d1
			ps273.OverlayValues[2] = d2
			ps273.OverlayValues[3] = d3
			ps273.OverlayValues[4] = d4
			ps273.OverlayValues[5] = d5
			ps273.OverlayValues[6] = d6
			ps273.OverlayValues[7] = d7
			ps273.OverlayValues[8] = d8
			ps273.OverlayValues[9] = d9
			ps273.OverlayValues[10] = d10
			ps273.OverlayValues[11] = d11
			ps273.OverlayValues[12] = d12
			ps273.OverlayValues[13] = d13
			ps273.OverlayValues[14] = d14
			ps273.OverlayValues[20] = d20
			ps273.OverlayValues[21] = d21
			ps273.OverlayValues[22] = d22
			ps273.OverlayValues[24] = d24
			ps273.OverlayValues[25] = d25
			ps273.OverlayValues[27] = d27
			ps273.OverlayValues[28] = d28
			ps273.OverlayValues[29] = d29
			ps273.OverlayValues[32] = d32
			ps273.OverlayValues[34] = d34
			ps273.OverlayValues[35] = d35
			ps273.OverlayValues[36] = d36
			ps273.OverlayValues[38] = d38
			ps273.OverlayValues[39] = d39
			ps273.OverlayValues[40] = d40
			ps273.OverlayValues[41] = d41
			ps273.OverlayValues[42] = d42
			ps273.OverlayValues[43] = d43
			ps273.OverlayValues[44] = d44
			ps273.OverlayValues[46] = d46
			ps273.OverlayValues[47] = d47
			ps273.OverlayValues[48] = d48
			ps273.OverlayValues[49] = d49
			ps273.OverlayValues[50] = d50
			ps273.OverlayValues[51] = d51
			ps273.OverlayValues[52] = d52
			ps273.OverlayValues[53] = d53
			ps273.OverlayValues[54] = d54
			ps273.OverlayValues[55] = d55
			ps273.OverlayValues[56] = d56
			ps273.OverlayValues[57] = d57
			ps273.OverlayValues[58] = d58
			ps273.OverlayValues[59] = d59
			ps273.OverlayValues[60] = d60
			ps273.OverlayValues[61] = d61
			ps273.OverlayValues[62] = d62
			ps273.OverlayValues[63] = d63
			ps273.OverlayValues[64] = d64
			ps273.OverlayValues[65] = d65
			ps273.OverlayValues[66] = d66
			ps273.OverlayValues[67] = d67
			ps273.OverlayValues[68] = d68
			ps273.OverlayValues[69] = d69
			ps273.OverlayValues[70] = d70
			ps273.OverlayValues[71] = d71
			ps273.OverlayValues[72] = d72
			ps273.OverlayValues[73] = d73
			ps273.OverlayValues[74] = d74
			ps273.OverlayValues[75] = d75
			ps273.OverlayValues[76] = d76
			ps273.OverlayValues[77] = d77
			ps273.OverlayValues[78] = d78
			ps273.OverlayValues[79] = d79
			ps273.OverlayValues[80] = d80
			ps273.OverlayValues[81] = d81
			ps273.OverlayValues[82] = d82
			ps273.OverlayValues[83] = d83
			ps273.OverlayValues[84] = d84
			ps273.OverlayValues[85] = d85
			ps273.OverlayValues[86] = d86
			ps273.OverlayValues[87] = d87
			ps273.OverlayValues[88] = d88
			ps273.OverlayValues[89] = d89
			ps273.OverlayValues[90] = d90
			ps273.OverlayValues[91] = d91
			ps273.OverlayValues[92] = d92
			ps273.OverlayValues[93] = d93
			ps273.OverlayValues[94] = d94
			ps273.OverlayValues[95] = d95
			ps273.OverlayValues[102] = d102
			ps273.OverlayValues[103] = d103
			ps273.OverlayValues[104] = d104
			ps273.OverlayValues[105] = d105
			ps273.OverlayValues[106] = d106
			ps273.OverlayValues[107] = d107
			ps273.OverlayValues[108] = d108
			ps273.OverlayValues[109] = d109
			ps273.OverlayValues[110] = d110
			ps273.OverlayValues[111] = d111
			ps273.OverlayValues[112] = d112
			ps273.OverlayValues[113] = d113
			ps273.OverlayValues[114] = d114
			ps273.OverlayValues[115] = d115
			ps273.OverlayValues[116] = d116
			ps273.OverlayValues[117] = d117
			ps273.OverlayValues[118] = d118
			ps273.OverlayValues[119] = d119
			ps273.OverlayValues[120] = d120
			ps273.OverlayValues[121] = d121
			ps273.OverlayValues[122] = d122
			ps273.OverlayValues[123] = d123
			ps273.OverlayValues[124] = d124
			ps273.OverlayValues[125] = d125
			ps273.OverlayValues[126] = d126
			ps273.OverlayValues[127] = d127
			ps273.OverlayValues[128] = d128
			ps273.OverlayValues[129] = d129
			ps273.OverlayValues[130] = d130
			ps273.OverlayValues[131] = d131
			ps273.OverlayValues[132] = d132
			ps273.OverlayValues[133] = d133
			ps273.OverlayValues[134] = d134
			ps273.OverlayValues[135] = d135
			ps273.OverlayValues[136] = d136
			ps273.OverlayValues[137] = d137
			ps273.OverlayValues[138] = d138
			ps273.OverlayValues[139] = d139
			ps273.OverlayValues[140] = d140
			ps273.OverlayValues[141] = d141
			ps273.OverlayValues[142] = d142
			ps273.OverlayValues[143] = d143
			ps273.OverlayValues[144] = d144
			ps273.OverlayValues[145] = d145
			ps273.OverlayValues[146] = d146
			ps273.OverlayValues[153] = d153
			ps273.OverlayValues[154] = d154
			ps273.OverlayValues[160] = d160
			ps273.OverlayValues[161] = d161
			ps273.OverlayValues[162] = d162
			ps273.OverlayValues[163] = d163
			ps273.OverlayValues[164] = d164
			ps273.OverlayValues[165] = d165
			ps273.OverlayValues[166] = d166
			ps273.OverlayValues[168] = d168
			ps273.OverlayValues[170] = d170
			ps273.OverlayValues[171] = d171
			ps273.OverlayValues[174] = d174
			ps273.OverlayValues[177] = d177
			ps273.OverlayValues[178] = d178
			ps273.OverlayValues[179] = d179
			ps273.OverlayValues[181] = d181
			ps273.OverlayValues[182] = d182
			ps273.OverlayValues[183] = d183
			ps273.OverlayValues[184] = d184
			ps273.OverlayValues[185] = d185
			ps273.OverlayValues[186] = d186
			ps273.OverlayValues[188] = d188
			ps273.OverlayValues[189] = d189
			ps273.OverlayValues[190] = d190
			ps273.OverlayValues[191] = d191
			ps273.OverlayValues[192] = d192
			ps273.OverlayValues[193] = d193
			ps273.OverlayValues[194] = d194
			ps273.OverlayValues[195] = d195
			ps273.OverlayValues[196] = d196
			ps273.OverlayValues[199] = d199
			ps273.OverlayValues[200] = d200
			ps273.OverlayValues[201] = d201
			ps273.OverlayValues[204] = d204
			ps273.OverlayValues[205] = d205
			ps273.OverlayValues[206] = d206
			ps273.OverlayValues[207] = d207
			ps273.OverlayValues[208] = d208
			ps273.OverlayValues[209] = d209
			ps273.OverlayValues[210] = d210
			ps273.OverlayValues[211] = d211
			ps273.OverlayValues[212] = d212
			ps273.OverlayValues[214] = d214
			ps273.OverlayValues[215] = d215
			ps273.OverlayValues[216] = d216
			ps273.OverlayValues[217] = d217
			ps273.OverlayValues[218] = d218
			ps273.OverlayValues[219] = d219
			ps273.OverlayValues[220] = d220
			ps273.OverlayValues[221] = d221
			ps273.OverlayValues[222] = d222
			ps273.OverlayValues[223] = d223
			ps273.OverlayValues[225] = d225
			ps273.OverlayValues[226] = d226
			ps273.OverlayValues[227] = d227
			ps273.OverlayValues[228] = d228
			ps273.OverlayValues[229] = d229
			ps273.OverlayValues[230] = d230
			ps273.OverlayValues[231] = d231
			ps273.OverlayValues[232] = d232
			ps273.OverlayValues[233] = d233
			ps273.OverlayValues[234] = d234
			ps273.OverlayValues[235] = d235
			ps273.OverlayValues[236] = d236
			ps273.OverlayValues[237] = d237
			ps273.OverlayValues[238] = d238
			ps273.OverlayValues[239] = d239
			ps273.OverlayValues[240] = d240
			ps273.OverlayValues[241] = d241
			ps273.OverlayValues[242] = d242
			ps273.OverlayValues[243] = d243
			ps273.OverlayValues[244] = d244
			ps273.OverlayValues[245] = d245
			ps273.OverlayValues[246] = d246
			ps273.OverlayValues[247] = d247
			ps273.OverlayValues[248] = d248
			ps273.OverlayValues[249] = d249
			ps273.OverlayValues[250] = d250
			ps273.OverlayValues[251] = d251
			ps273.OverlayValues[252] = d252
			ps273.OverlayValues[253] = d253
			ps273.OverlayValues[254] = d254
			ps273.OverlayValues[255] = d255
			ps273.OverlayValues[256] = d256
			ps273.OverlayValues[257] = d257
			ps273.OverlayValues[258] = d258
			ps273.OverlayValues[259] = d259
			ps273.OverlayValues[260] = d260
			ps273.OverlayValues[261] = d261
			ps273.OverlayValues[262] = d262
			ps273.OverlayValues[263] = d263
			ps273.OverlayValues[264] = d264
			ps273.OverlayValues[265] = d265
			ps273.OverlayValues[266] = d266
			ps273.OverlayValues[267] = d267
			ps273.OverlayValues[268] = d268
			ps273.OverlayValues[269] = d269
			ps273.OverlayValues[270] = d270
			ps273.OverlayValues[271] = d271
				return bbs[16].RenderPS(ps273)
			}
			lbl62 := ctx.W.ReserveLabel()
			lbl63 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d271.Reg, 0)
			ctx.W.EmitJcc(scm.CcNE, lbl62)
			ctx.W.EmitJmp(lbl63)
			ctx.W.MarkLabel(lbl62)
			ctx.W.EmitJmp(lbl15)
			ctx.W.MarkLabel(lbl63)
			ctx.W.EmitJmp(lbl17)
			ps274 := scm.PhiState{General: true}
			ps274.OverlayValues = make([]scm.JITValueDesc, 272)
			ps274.OverlayValues[0] = d0
			ps274.OverlayValues[1] = d1
			ps274.OverlayValues[2] = d2
			ps274.OverlayValues[3] = d3
			ps274.OverlayValues[4] = d4
			ps274.OverlayValues[5] = d5
			ps274.OverlayValues[6] = d6
			ps274.OverlayValues[7] = d7
			ps274.OverlayValues[8] = d8
			ps274.OverlayValues[9] = d9
			ps274.OverlayValues[10] = d10
			ps274.OverlayValues[11] = d11
			ps274.OverlayValues[12] = d12
			ps274.OverlayValues[13] = d13
			ps274.OverlayValues[14] = d14
			ps274.OverlayValues[20] = d20
			ps274.OverlayValues[21] = d21
			ps274.OverlayValues[22] = d22
			ps274.OverlayValues[24] = d24
			ps274.OverlayValues[25] = d25
			ps274.OverlayValues[27] = d27
			ps274.OverlayValues[28] = d28
			ps274.OverlayValues[29] = d29
			ps274.OverlayValues[32] = d32
			ps274.OverlayValues[34] = d34
			ps274.OverlayValues[35] = d35
			ps274.OverlayValues[36] = d36
			ps274.OverlayValues[38] = d38
			ps274.OverlayValues[39] = d39
			ps274.OverlayValues[40] = d40
			ps274.OverlayValues[41] = d41
			ps274.OverlayValues[42] = d42
			ps274.OverlayValues[43] = d43
			ps274.OverlayValues[44] = d44
			ps274.OverlayValues[46] = d46
			ps274.OverlayValues[47] = d47
			ps274.OverlayValues[48] = d48
			ps274.OverlayValues[49] = d49
			ps274.OverlayValues[50] = d50
			ps274.OverlayValues[51] = d51
			ps274.OverlayValues[52] = d52
			ps274.OverlayValues[53] = d53
			ps274.OverlayValues[54] = d54
			ps274.OverlayValues[55] = d55
			ps274.OverlayValues[56] = d56
			ps274.OverlayValues[57] = d57
			ps274.OverlayValues[58] = d58
			ps274.OverlayValues[59] = d59
			ps274.OverlayValues[60] = d60
			ps274.OverlayValues[61] = d61
			ps274.OverlayValues[62] = d62
			ps274.OverlayValues[63] = d63
			ps274.OverlayValues[64] = d64
			ps274.OverlayValues[65] = d65
			ps274.OverlayValues[66] = d66
			ps274.OverlayValues[67] = d67
			ps274.OverlayValues[68] = d68
			ps274.OverlayValues[69] = d69
			ps274.OverlayValues[70] = d70
			ps274.OverlayValues[71] = d71
			ps274.OverlayValues[72] = d72
			ps274.OverlayValues[73] = d73
			ps274.OverlayValues[74] = d74
			ps274.OverlayValues[75] = d75
			ps274.OverlayValues[76] = d76
			ps274.OverlayValues[77] = d77
			ps274.OverlayValues[78] = d78
			ps274.OverlayValues[79] = d79
			ps274.OverlayValues[80] = d80
			ps274.OverlayValues[81] = d81
			ps274.OverlayValues[82] = d82
			ps274.OverlayValues[83] = d83
			ps274.OverlayValues[84] = d84
			ps274.OverlayValues[85] = d85
			ps274.OverlayValues[86] = d86
			ps274.OverlayValues[87] = d87
			ps274.OverlayValues[88] = d88
			ps274.OverlayValues[89] = d89
			ps274.OverlayValues[90] = d90
			ps274.OverlayValues[91] = d91
			ps274.OverlayValues[92] = d92
			ps274.OverlayValues[93] = d93
			ps274.OverlayValues[94] = d94
			ps274.OverlayValues[95] = d95
			ps274.OverlayValues[102] = d102
			ps274.OverlayValues[103] = d103
			ps274.OverlayValues[104] = d104
			ps274.OverlayValues[105] = d105
			ps274.OverlayValues[106] = d106
			ps274.OverlayValues[107] = d107
			ps274.OverlayValues[108] = d108
			ps274.OverlayValues[109] = d109
			ps274.OverlayValues[110] = d110
			ps274.OverlayValues[111] = d111
			ps274.OverlayValues[112] = d112
			ps274.OverlayValues[113] = d113
			ps274.OverlayValues[114] = d114
			ps274.OverlayValues[115] = d115
			ps274.OverlayValues[116] = d116
			ps274.OverlayValues[117] = d117
			ps274.OverlayValues[118] = d118
			ps274.OverlayValues[119] = d119
			ps274.OverlayValues[120] = d120
			ps274.OverlayValues[121] = d121
			ps274.OverlayValues[122] = d122
			ps274.OverlayValues[123] = d123
			ps274.OverlayValues[124] = d124
			ps274.OverlayValues[125] = d125
			ps274.OverlayValues[126] = d126
			ps274.OverlayValues[127] = d127
			ps274.OverlayValues[128] = d128
			ps274.OverlayValues[129] = d129
			ps274.OverlayValues[130] = d130
			ps274.OverlayValues[131] = d131
			ps274.OverlayValues[132] = d132
			ps274.OverlayValues[133] = d133
			ps274.OverlayValues[134] = d134
			ps274.OverlayValues[135] = d135
			ps274.OverlayValues[136] = d136
			ps274.OverlayValues[137] = d137
			ps274.OverlayValues[138] = d138
			ps274.OverlayValues[139] = d139
			ps274.OverlayValues[140] = d140
			ps274.OverlayValues[141] = d141
			ps274.OverlayValues[142] = d142
			ps274.OverlayValues[143] = d143
			ps274.OverlayValues[144] = d144
			ps274.OverlayValues[145] = d145
			ps274.OverlayValues[146] = d146
			ps274.OverlayValues[153] = d153
			ps274.OverlayValues[154] = d154
			ps274.OverlayValues[160] = d160
			ps274.OverlayValues[161] = d161
			ps274.OverlayValues[162] = d162
			ps274.OverlayValues[163] = d163
			ps274.OverlayValues[164] = d164
			ps274.OverlayValues[165] = d165
			ps274.OverlayValues[166] = d166
			ps274.OverlayValues[168] = d168
			ps274.OverlayValues[170] = d170
			ps274.OverlayValues[171] = d171
			ps274.OverlayValues[174] = d174
			ps274.OverlayValues[177] = d177
			ps274.OverlayValues[178] = d178
			ps274.OverlayValues[179] = d179
			ps274.OverlayValues[181] = d181
			ps274.OverlayValues[182] = d182
			ps274.OverlayValues[183] = d183
			ps274.OverlayValues[184] = d184
			ps274.OverlayValues[185] = d185
			ps274.OverlayValues[186] = d186
			ps274.OverlayValues[188] = d188
			ps274.OverlayValues[189] = d189
			ps274.OverlayValues[190] = d190
			ps274.OverlayValues[191] = d191
			ps274.OverlayValues[192] = d192
			ps274.OverlayValues[193] = d193
			ps274.OverlayValues[194] = d194
			ps274.OverlayValues[195] = d195
			ps274.OverlayValues[196] = d196
			ps274.OverlayValues[199] = d199
			ps274.OverlayValues[200] = d200
			ps274.OverlayValues[201] = d201
			ps274.OverlayValues[204] = d204
			ps274.OverlayValues[205] = d205
			ps274.OverlayValues[206] = d206
			ps274.OverlayValues[207] = d207
			ps274.OverlayValues[208] = d208
			ps274.OverlayValues[209] = d209
			ps274.OverlayValues[210] = d210
			ps274.OverlayValues[211] = d211
			ps274.OverlayValues[212] = d212
			ps274.OverlayValues[214] = d214
			ps274.OverlayValues[215] = d215
			ps274.OverlayValues[216] = d216
			ps274.OverlayValues[217] = d217
			ps274.OverlayValues[218] = d218
			ps274.OverlayValues[219] = d219
			ps274.OverlayValues[220] = d220
			ps274.OverlayValues[221] = d221
			ps274.OverlayValues[222] = d222
			ps274.OverlayValues[223] = d223
			ps274.OverlayValues[225] = d225
			ps274.OverlayValues[226] = d226
			ps274.OverlayValues[227] = d227
			ps274.OverlayValues[228] = d228
			ps274.OverlayValues[229] = d229
			ps274.OverlayValues[230] = d230
			ps274.OverlayValues[231] = d231
			ps274.OverlayValues[232] = d232
			ps274.OverlayValues[233] = d233
			ps274.OverlayValues[234] = d234
			ps274.OverlayValues[235] = d235
			ps274.OverlayValues[236] = d236
			ps274.OverlayValues[237] = d237
			ps274.OverlayValues[238] = d238
			ps274.OverlayValues[239] = d239
			ps274.OverlayValues[240] = d240
			ps274.OverlayValues[241] = d241
			ps274.OverlayValues[242] = d242
			ps274.OverlayValues[243] = d243
			ps274.OverlayValues[244] = d244
			ps274.OverlayValues[245] = d245
			ps274.OverlayValues[246] = d246
			ps274.OverlayValues[247] = d247
			ps274.OverlayValues[248] = d248
			ps274.OverlayValues[249] = d249
			ps274.OverlayValues[250] = d250
			ps274.OverlayValues[251] = d251
			ps274.OverlayValues[252] = d252
			ps274.OverlayValues[253] = d253
			ps274.OverlayValues[254] = d254
			ps274.OverlayValues[255] = d255
			ps274.OverlayValues[256] = d256
			ps274.OverlayValues[257] = d257
			ps274.OverlayValues[258] = d258
			ps274.OverlayValues[259] = d259
			ps274.OverlayValues[260] = d260
			ps274.OverlayValues[261] = d261
			ps274.OverlayValues[262] = d262
			ps274.OverlayValues[263] = d263
			ps274.OverlayValues[264] = d264
			ps274.OverlayValues[265] = d265
			ps274.OverlayValues[266] = d266
			ps274.OverlayValues[267] = d267
			ps274.OverlayValues[268] = d268
			ps274.OverlayValues[269] = d269
			ps274.OverlayValues[270] = d270
			ps274.OverlayValues[271] = d271
			ps275 := scm.PhiState{General: true}
			ps275.OverlayValues = make([]scm.JITValueDesc, 272)
			ps275.OverlayValues[0] = d0
			ps275.OverlayValues[1] = d1
			ps275.OverlayValues[2] = d2
			ps275.OverlayValues[3] = d3
			ps275.OverlayValues[4] = d4
			ps275.OverlayValues[5] = d5
			ps275.OverlayValues[6] = d6
			ps275.OverlayValues[7] = d7
			ps275.OverlayValues[8] = d8
			ps275.OverlayValues[9] = d9
			ps275.OverlayValues[10] = d10
			ps275.OverlayValues[11] = d11
			ps275.OverlayValues[12] = d12
			ps275.OverlayValues[13] = d13
			ps275.OverlayValues[14] = d14
			ps275.OverlayValues[20] = d20
			ps275.OverlayValues[21] = d21
			ps275.OverlayValues[22] = d22
			ps275.OverlayValues[24] = d24
			ps275.OverlayValues[25] = d25
			ps275.OverlayValues[27] = d27
			ps275.OverlayValues[28] = d28
			ps275.OverlayValues[29] = d29
			ps275.OverlayValues[32] = d32
			ps275.OverlayValues[34] = d34
			ps275.OverlayValues[35] = d35
			ps275.OverlayValues[36] = d36
			ps275.OverlayValues[38] = d38
			ps275.OverlayValues[39] = d39
			ps275.OverlayValues[40] = d40
			ps275.OverlayValues[41] = d41
			ps275.OverlayValues[42] = d42
			ps275.OverlayValues[43] = d43
			ps275.OverlayValues[44] = d44
			ps275.OverlayValues[46] = d46
			ps275.OverlayValues[47] = d47
			ps275.OverlayValues[48] = d48
			ps275.OverlayValues[49] = d49
			ps275.OverlayValues[50] = d50
			ps275.OverlayValues[51] = d51
			ps275.OverlayValues[52] = d52
			ps275.OverlayValues[53] = d53
			ps275.OverlayValues[54] = d54
			ps275.OverlayValues[55] = d55
			ps275.OverlayValues[56] = d56
			ps275.OverlayValues[57] = d57
			ps275.OverlayValues[58] = d58
			ps275.OverlayValues[59] = d59
			ps275.OverlayValues[60] = d60
			ps275.OverlayValues[61] = d61
			ps275.OverlayValues[62] = d62
			ps275.OverlayValues[63] = d63
			ps275.OverlayValues[64] = d64
			ps275.OverlayValues[65] = d65
			ps275.OverlayValues[66] = d66
			ps275.OverlayValues[67] = d67
			ps275.OverlayValues[68] = d68
			ps275.OverlayValues[69] = d69
			ps275.OverlayValues[70] = d70
			ps275.OverlayValues[71] = d71
			ps275.OverlayValues[72] = d72
			ps275.OverlayValues[73] = d73
			ps275.OverlayValues[74] = d74
			ps275.OverlayValues[75] = d75
			ps275.OverlayValues[76] = d76
			ps275.OverlayValues[77] = d77
			ps275.OverlayValues[78] = d78
			ps275.OverlayValues[79] = d79
			ps275.OverlayValues[80] = d80
			ps275.OverlayValues[81] = d81
			ps275.OverlayValues[82] = d82
			ps275.OverlayValues[83] = d83
			ps275.OverlayValues[84] = d84
			ps275.OverlayValues[85] = d85
			ps275.OverlayValues[86] = d86
			ps275.OverlayValues[87] = d87
			ps275.OverlayValues[88] = d88
			ps275.OverlayValues[89] = d89
			ps275.OverlayValues[90] = d90
			ps275.OverlayValues[91] = d91
			ps275.OverlayValues[92] = d92
			ps275.OverlayValues[93] = d93
			ps275.OverlayValues[94] = d94
			ps275.OverlayValues[95] = d95
			ps275.OverlayValues[102] = d102
			ps275.OverlayValues[103] = d103
			ps275.OverlayValues[104] = d104
			ps275.OverlayValues[105] = d105
			ps275.OverlayValues[106] = d106
			ps275.OverlayValues[107] = d107
			ps275.OverlayValues[108] = d108
			ps275.OverlayValues[109] = d109
			ps275.OverlayValues[110] = d110
			ps275.OverlayValues[111] = d111
			ps275.OverlayValues[112] = d112
			ps275.OverlayValues[113] = d113
			ps275.OverlayValues[114] = d114
			ps275.OverlayValues[115] = d115
			ps275.OverlayValues[116] = d116
			ps275.OverlayValues[117] = d117
			ps275.OverlayValues[118] = d118
			ps275.OverlayValues[119] = d119
			ps275.OverlayValues[120] = d120
			ps275.OverlayValues[121] = d121
			ps275.OverlayValues[122] = d122
			ps275.OverlayValues[123] = d123
			ps275.OverlayValues[124] = d124
			ps275.OverlayValues[125] = d125
			ps275.OverlayValues[126] = d126
			ps275.OverlayValues[127] = d127
			ps275.OverlayValues[128] = d128
			ps275.OverlayValues[129] = d129
			ps275.OverlayValues[130] = d130
			ps275.OverlayValues[131] = d131
			ps275.OverlayValues[132] = d132
			ps275.OverlayValues[133] = d133
			ps275.OverlayValues[134] = d134
			ps275.OverlayValues[135] = d135
			ps275.OverlayValues[136] = d136
			ps275.OverlayValues[137] = d137
			ps275.OverlayValues[138] = d138
			ps275.OverlayValues[139] = d139
			ps275.OverlayValues[140] = d140
			ps275.OverlayValues[141] = d141
			ps275.OverlayValues[142] = d142
			ps275.OverlayValues[143] = d143
			ps275.OverlayValues[144] = d144
			ps275.OverlayValues[145] = d145
			ps275.OverlayValues[146] = d146
			ps275.OverlayValues[153] = d153
			ps275.OverlayValues[154] = d154
			ps275.OverlayValues[160] = d160
			ps275.OverlayValues[161] = d161
			ps275.OverlayValues[162] = d162
			ps275.OverlayValues[163] = d163
			ps275.OverlayValues[164] = d164
			ps275.OverlayValues[165] = d165
			ps275.OverlayValues[166] = d166
			ps275.OverlayValues[168] = d168
			ps275.OverlayValues[170] = d170
			ps275.OverlayValues[171] = d171
			ps275.OverlayValues[174] = d174
			ps275.OverlayValues[177] = d177
			ps275.OverlayValues[178] = d178
			ps275.OverlayValues[179] = d179
			ps275.OverlayValues[181] = d181
			ps275.OverlayValues[182] = d182
			ps275.OverlayValues[183] = d183
			ps275.OverlayValues[184] = d184
			ps275.OverlayValues[185] = d185
			ps275.OverlayValues[186] = d186
			ps275.OverlayValues[188] = d188
			ps275.OverlayValues[189] = d189
			ps275.OverlayValues[190] = d190
			ps275.OverlayValues[191] = d191
			ps275.OverlayValues[192] = d192
			ps275.OverlayValues[193] = d193
			ps275.OverlayValues[194] = d194
			ps275.OverlayValues[195] = d195
			ps275.OverlayValues[196] = d196
			ps275.OverlayValues[199] = d199
			ps275.OverlayValues[200] = d200
			ps275.OverlayValues[201] = d201
			ps275.OverlayValues[204] = d204
			ps275.OverlayValues[205] = d205
			ps275.OverlayValues[206] = d206
			ps275.OverlayValues[207] = d207
			ps275.OverlayValues[208] = d208
			ps275.OverlayValues[209] = d209
			ps275.OverlayValues[210] = d210
			ps275.OverlayValues[211] = d211
			ps275.OverlayValues[212] = d212
			ps275.OverlayValues[214] = d214
			ps275.OverlayValues[215] = d215
			ps275.OverlayValues[216] = d216
			ps275.OverlayValues[217] = d217
			ps275.OverlayValues[218] = d218
			ps275.OverlayValues[219] = d219
			ps275.OverlayValues[220] = d220
			ps275.OverlayValues[221] = d221
			ps275.OverlayValues[222] = d222
			ps275.OverlayValues[223] = d223
			ps275.OverlayValues[225] = d225
			ps275.OverlayValues[226] = d226
			ps275.OverlayValues[227] = d227
			ps275.OverlayValues[228] = d228
			ps275.OverlayValues[229] = d229
			ps275.OverlayValues[230] = d230
			ps275.OverlayValues[231] = d231
			ps275.OverlayValues[232] = d232
			ps275.OverlayValues[233] = d233
			ps275.OverlayValues[234] = d234
			ps275.OverlayValues[235] = d235
			ps275.OverlayValues[236] = d236
			ps275.OverlayValues[237] = d237
			ps275.OverlayValues[238] = d238
			ps275.OverlayValues[239] = d239
			ps275.OverlayValues[240] = d240
			ps275.OverlayValues[241] = d241
			ps275.OverlayValues[242] = d242
			ps275.OverlayValues[243] = d243
			ps275.OverlayValues[244] = d244
			ps275.OverlayValues[245] = d245
			ps275.OverlayValues[246] = d246
			ps275.OverlayValues[247] = d247
			ps275.OverlayValues[248] = d248
			ps275.OverlayValues[249] = d249
			ps275.OverlayValues[250] = d250
			ps275.OverlayValues[251] = d251
			ps275.OverlayValues[252] = d252
			ps275.OverlayValues[253] = d253
			ps275.OverlayValues[254] = d254
			ps275.OverlayValues[255] = d255
			ps275.OverlayValues[256] = d256
			ps275.OverlayValues[257] = d257
			ps275.OverlayValues[258] = d258
			ps275.OverlayValues[259] = d259
			ps275.OverlayValues[260] = d260
			ps275.OverlayValues[261] = d261
			ps275.OverlayValues[262] = d262
			ps275.OverlayValues[263] = d263
			ps275.OverlayValues[264] = d264
			ps275.OverlayValues[265] = d265
			ps275.OverlayValues[266] = d266
			ps275.OverlayValues[267] = d267
			ps275.OverlayValues[268] = d268
			ps275.OverlayValues[269] = d269
			ps275.OverlayValues[270] = d270
			ps275.OverlayValues[271] = d271
			snap276 := d5
			alloc277 := ctx.SnapshotAllocState()
			if !bbs[16].Rendered {
				bbs[16].RenderPS(ps275)
			}
			ctx.RestoreAllocState(alloc277)
			d5 = snap276
			if !bbs[14].Rendered {
				return bbs[14].RenderPS(ps274)
			}
			return result
			ctx.FreeDesc(&d270)
			return result
			}
			bbs[14].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[14].VisitCount >= 2 {
					ps.General = true
					return bbs[14].RenderPS(ps)
				}
			}
			bbs[14].VisitCount++
			if ps.General {
				if bbs[14].Rendered {
					ctx.W.EmitJmp(lbl15)
					return result
				}
				bbs[14].Rendered = true
				bbs[14].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_14 = bbs[14].Address
				ctx.W.MarkLabel(lbl15)
				ctx.W.ResolveFixups()
			}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
				d154 = ps.OverlayValues[154]
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
			if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
				d168 = ps.OverlayValues[168]
			}
			if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != scm.LocNone {
				d170 = ps.OverlayValues[170]
			}
			if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != scm.LocNone {
				d171 = ps.OverlayValues[171]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
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
			if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
				d199 = ps.OverlayValues[199]
			}
			if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
				d200 = ps.OverlayValues[200]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
				d201 = ps.OverlayValues[201]
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
			if len(ps.OverlayValues) > 254 && ps.OverlayValues[254].Loc != scm.LocNone {
				d254 = ps.OverlayValues[254]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d5)
			var d278 scm.JITValueDesc
			if d5.Loc == scm.LocImm {
				d278 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d5.Imm.Int()) == uint64(0))}
			} else {
				r154 := ctx.AllocRegExcept(d5.Reg)
				ctx.W.EmitCmpRegImm32(d5.Reg, 0)
				ctx.W.EmitSetcc(r154, scm.CcE)
				d278 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r154}
				ctx.BindReg(r154, &d278)
			}
			d279 = d278
			ctx.EnsureDesc(&d279)
			if d279.Loc != scm.LocImm && d279.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d279.Loc == scm.LocImm {
				if d279.Imm.Bool() {
			ps280 := scm.PhiState{General: ps.General}
			ps280.OverlayValues = make([]scm.JITValueDesc, 280)
			ps280.OverlayValues[0] = d0
			ps280.OverlayValues[1] = d1
			ps280.OverlayValues[2] = d2
			ps280.OverlayValues[3] = d3
			ps280.OverlayValues[4] = d4
			ps280.OverlayValues[5] = d5
			ps280.OverlayValues[6] = d6
			ps280.OverlayValues[7] = d7
			ps280.OverlayValues[8] = d8
			ps280.OverlayValues[9] = d9
			ps280.OverlayValues[10] = d10
			ps280.OverlayValues[11] = d11
			ps280.OverlayValues[12] = d12
			ps280.OverlayValues[13] = d13
			ps280.OverlayValues[14] = d14
			ps280.OverlayValues[20] = d20
			ps280.OverlayValues[21] = d21
			ps280.OverlayValues[22] = d22
			ps280.OverlayValues[24] = d24
			ps280.OverlayValues[25] = d25
			ps280.OverlayValues[27] = d27
			ps280.OverlayValues[28] = d28
			ps280.OverlayValues[29] = d29
			ps280.OverlayValues[32] = d32
			ps280.OverlayValues[34] = d34
			ps280.OverlayValues[35] = d35
			ps280.OverlayValues[36] = d36
			ps280.OverlayValues[38] = d38
			ps280.OverlayValues[39] = d39
			ps280.OverlayValues[40] = d40
			ps280.OverlayValues[41] = d41
			ps280.OverlayValues[42] = d42
			ps280.OverlayValues[43] = d43
			ps280.OverlayValues[44] = d44
			ps280.OverlayValues[46] = d46
			ps280.OverlayValues[47] = d47
			ps280.OverlayValues[48] = d48
			ps280.OverlayValues[49] = d49
			ps280.OverlayValues[50] = d50
			ps280.OverlayValues[51] = d51
			ps280.OverlayValues[52] = d52
			ps280.OverlayValues[53] = d53
			ps280.OverlayValues[54] = d54
			ps280.OverlayValues[55] = d55
			ps280.OverlayValues[56] = d56
			ps280.OverlayValues[57] = d57
			ps280.OverlayValues[58] = d58
			ps280.OverlayValues[59] = d59
			ps280.OverlayValues[60] = d60
			ps280.OverlayValues[61] = d61
			ps280.OverlayValues[62] = d62
			ps280.OverlayValues[63] = d63
			ps280.OverlayValues[64] = d64
			ps280.OverlayValues[65] = d65
			ps280.OverlayValues[66] = d66
			ps280.OverlayValues[67] = d67
			ps280.OverlayValues[68] = d68
			ps280.OverlayValues[69] = d69
			ps280.OverlayValues[70] = d70
			ps280.OverlayValues[71] = d71
			ps280.OverlayValues[72] = d72
			ps280.OverlayValues[73] = d73
			ps280.OverlayValues[74] = d74
			ps280.OverlayValues[75] = d75
			ps280.OverlayValues[76] = d76
			ps280.OverlayValues[77] = d77
			ps280.OverlayValues[78] = d78
			ps280.OverlayValues[79] = d79
			ps280.OverlayValues[80] = d80
			ps280.OverlayValues[81] = d81
			ps280.OverlayValues[82] = d82
			ps280.OverlayValues[83] = d83
			ps280.OverlayValues[84] = d84
			ps280.OverlayValues[85] = d85
			ps280.OverlayValues[86] = d86
			ps280.OverlayValues[87] = d87
			ps280.OverlayValues[88] = d88
			ps280.OverlayValues[89] = d89
			ps280.OverlayValues[90] = d90
			ps280.OverlayValues[91] = d91
			ps280.OverlayValues[92] = d92
			ps280.OverlayValues[93] = d93
			ps280.OverlayValues[94] = d94
			ps280.OverlayValues[95] = d95
			ps280.OverlayValues[102] = d102
			ps280.OverlayValues[103] = d103
			ps280.OverlayValues[104] = d104
			ps280.OverlayValues[105] = d105
			ps280.OverlayValues[106] = d106
			ps280.OverlayValues[107] = d107
			ps280.OverlayValues[108] = d108
			ps280.OverlayValues[109] = d109
			ps280.OverlayValues[110] = d110
			ps280.OverlayValues[111] = d111
			ps280.OverlayValues[112] = d112
			ps280.OverlayValues[113] = d113
			ps280.OverlayValues[114] = d114
			ps280.OverlayValues[115] = d115
			ps280.OverlayValues[116] = d116
			ps280.OverlayValues[117] = d117
			ps280.OverlayValues[118] = d118
			ps280.OverlayValues[119] = d119
			ps280.OverlayValues[120] = d120
			ps280.OverlayValues[121] = d121
			ps280.OverlayValues[122] = d122
			ps280.OverlayValues[123] = d123
			ps280.OverlayValues[124] = d124
			ps280.OverlayValues[125] = d125
			ps280.OverlayValues[126] = d126
			ps280.OverlayValues[127] = d127
			ps280.OverlayValues[128] = d128
			ps280.OverlayValues[129] = d129
			ps280.OverlayValues[130] = d130
			ps280.OverlayValues[131] = d131
			ps280.OverlayValues[132] = d132
			ps280.OverlayValues[133] = d133
			ps280.OverlayValues[134] = d134
			ps280.OverlayValues[135] = d135
			ps280.OverlayValues[136] = d136
			ps280.OverlayValues[137] = d137
			ps280.OverlayValues[138] = d138
			ps280.OverlayValues[139] = d139
			ps280.OverlayValues[140] = d140
			ps280.OverlayValues[141] = d141
			ps280.OverlayValues[142] = d142
			ps280.OverlayValues[143] = d143
			ps280.OverlayValues[144] = d144
			ps280.OverlayValues[145] = d145
			ps280.OverlayValues[146] = d146
			ps280.OverlayValues[153] = d153
			ps280.OverlayValues[154] = d154
			ps280.OverlayValues[160] = d160
			ps280.OverlayValues[161] = d161
			ps280.OverlayValues[162] = d162
			ps280.OverlayValues[163] = d163
			ps280.OverlayValues[164] = d164
			ps280.OverlayValues[165] = d165
			ps280.OverlayValues[166] = d166
			ps280.OverlayValues[168] = d168
			ps280.OverlayValues[170] = d170
			ps280.OverlayValues[171] = d171
			ps280.OverlayValues[174] = d174
			ps280.OverlayValues[177] = d177
			ps280.OverlayValues[178] = d178
			ps280.OverlayValues[179] = d179
			ps280.OverlayValues[181] = d181
			ps280.OverlayValues[182] = d182
			ps280.OverlayValues[183] = d183
			ps280.OverlayValues[184] = d184
			ps280.OverlayValues[185] = d185
			ps280.OverlayValues[186] = d186
			ps280.OverlayValues[188] = d188
			ps280.OverlayValues[189] = d189
			ps280.OverlayValues[190] = d190
			ps280.OverlayValues[191] = d191
			ps280.OverlayValues[192] = d192
			ps280.OverlayValues[193] = d193
			ps280.OverlayValues[194] = d194
			ps280.OverlayValues[195] = d195
			ps280.OverlayValues[196] = d196
			ps280.OverlayValues[199] = d199
			ps280.OverlayValues[200] = d200
			ps280.OverlayValues[201] = d201
			ps280.OverlayValues[204] = d204
			ps280.OverlayValues[205] = d205
			ps280.OverlayValues[206] = d206
			ps280.OverlayValues[207] = d207
			ps280.OverlayValues[208] = d208
			ps280.OverlayValues[209] = d209
			ps280.OverlayValues[210] = d210
			ps280.OverlayValues[211] = d211
			ps280.OverlayValues[212] = d212
			ps280.OverlayValues[214] = d214
			ps280.OverlayValues[215] = d215
			ps280.OverlayValues[216] = d216
			ps280.OverlayValues[217] = d217
			ps280.OverlayValues[218] = d218
			ps280.OverlayValues[219] = d219
			ps280.OverlayValues[220] = d220
			ps280.OverlayValues[221] = d221
			ps280.OverlayValues[222] = d222
			ps280.OverlayValues[223] = d223
			ps280.OverlayValues[225] = d225
			ps280.OverlayValues[226] = d226
			ps280.OverlayValues[227] = d227
			ps280.OverlayValues[228] = d228
			ps280.OverlayValues[229] = d229
			ps280.OverlayValues[230] = d230
			ps280.OverlayValues[231] = d231
			ps280.OverlayValues[232] = d232
			ps280.OverlayValues[233] = d233
			ps280.OverlayValues[234] = d234
			ps280.OverlayValues[235] = d235
			ps280.OverlayValues[236] = d236
			ps280.OverlayValues[237] = d237
			ps280.OverlayValues[238] = d238
			ps280.OverlayValues[239] = d239
			ps280.OverlayValues[240] = d240
			ps280.OverlayValues[241] = d241
			ps280.OverlayValues[242] = d242
			ps280.OverlayValues[243] = d243
			ps280.OverlayValues[244] = d244
			ps280.OverlayValues[245] = d245
			ps280.OverlayValues[246] = d246
			ps280.OverlayValues[247] = d247
			ps280.OverlayValues[248] = d248
			ps280.OverlayValues[249] = d249
			ps280.OverlayValues[250] = d250
			ps280.OverlayValues[251] = d251
			ps280.OverlayValues[252] = d252
			ps280.OverlayValues[253] = d253
			ps280.OverlayValues[254] = d254
			ps280.OverlayValues[255] = d255
			ps280.OverlayValues[256] = d256
			ps280.OverlayValues[257] = d257
			ps280.OverlayValues[258] = d258
			ps280.OverlayValues[259] = d259
			ps280.OverlayValues[260] = d260
			ps280.OverlayValues[261] = d261
			ps280.OverlayValues[262] = d262
			ps280.OverlayValues[263] = d263
			ps280.OverlayValues[264] = d264
			ps280.OverlayValues[265] = d265
			ps280.OverlayValues[266] = d266
			ps280.OverlayValues[267] = d267
			ps280.OverlayValues[268] = d268
			ps280.OverlayValues[269] = d269
			ps280.OverlayValues[270] = d270
			ps280.OverlayValues[271] = d271
			ps280.OverlayValues[278] = d278
			ps280.OverlayValues[279] = d279
					return bbs[17].RenderPS(ps280)
				}
			ps281 := scm.PhiState{General: ps.General}
			ps281.OverlayValues = make([]scm.JITValueDesc, 280)
			ps281.OverlayValues[0] = d0
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
			ps281.OverlayValues[20] = d20
			ps281.OverlayValues[21] = d21
			ps281.OverlayValues[22] = d22
			ps281.OverlayValues[24] = d24
			ps281.OverlayValues[25] = d25
			ps281.OverlayValues[27] = d27
			ps281.OverlayValues[28] = d28
			ps281.OverlayValues[29] = d29
			ps281.OverlayValues[32] = d32
			ps281.OverlayValues[34] = d34
			ps281.OverlayValues[35] = d35
			ps281.OverlayValues[36] = d36
			ps281.OverlayValues[38] = d38
			ps281.OverlayValues[39] = d39
			ps281.OverlayValues[40] = d40
			ps281.OverlayValues[41] = d41
			ps281.OverlayValues[42] = d42
			ps281.OverlayValues[43] = d43
			ps281.OverlayValues[44] = d44
			ps281.OverlayValues[46] = d46
			ps281.OverlayValues[47] = d47
			ps281.OverlayValues[48] = d48
			ps281.OverlayValues[49] = d49
			ps281.OverlayValues[50] = d50
			ps281.OverlayValues[51] = d51
			ps281.OverlayValues[52] = d52
			ps281.OverlayValues[53] = d53
			ps281.OverlayValues[54] = d54
			ps281.OverlayValues[55] = d55
			ps281.OverlayValues[56] = d56
			ps281.OverlayValues[57] = d57
			ps281.OverlayValues[58] = d58
			ps281.OverlayValues[59] = d59
			ps281.OverlayValues[60] = d60
			ps281.OverlayValues[61] = d61
			ps281.OverlayValues[62] = d62
			ps281.OverlayValues[63] = d63
			ps281.OverlayValues[64] = d64
			ps281.OverlayValues[65] = d65
			ps281.OverlayValues[66] = d66
			ps281.OverlayValues[67] = d67
			ps281.OverlayValues[68] = d68
			ps281.OverlayValues[69] = d69
			ps281.OverlayValues[70] = d70
			ps281.OverlayValues[71] = d71
			ps281.OverlayValues[72] = d72
			ps281.OverlayValues[73] = d73
			ps281.OverlayValues[74] = d74
			ps281.OverlayValues[75] = d75
			ps281.OverlayValues[76] = d76
			ps281.OverlayValues[77] = d77
			ps281.OverlayValues[78] = d78
			ps281.OverlayValues[79] = d79
			ps281.OverlayValues[80] = d80
			ps281.OverlayValues[81] = d81
			ps281.OverlayValues[82] = d82
			ps281.OverlayValues[83] = d83
			ps281.OverlayValues[84] = d84
			ps281.OverlayValues[85] = d85
			ps281.OverlayValues[86] = d86
			ps281.OverlayValues[87] = d87
			ps281.OverlayValues[88] = d88
			ps281.OverlayValues[89] = d89
			ps281.OverlayValues[90] = d90
			ps281.OverlayValues[91] = d91
			ps281.OverlayValues[92] = d92
			ps281.OverlayValues[93] = d93
			ps281.OverlayValues[94] = d94
			ps281.OverlayValues[95] = d95
			ps281.OverlayValues[102] = d102
			ps281.OverlayValues[103] = d103
			ps281.OverlayValues[104] = d104
			ps281.OverlayValues[105] = d105
			ps281.OverlayValues[106] = d106
			ps281.OverlayValues[107] = d107
			ps281.OverlayValues[108] = d108
			ps281.OverlayValues[109] = d109
			ps281.OverlayValues[110] = d110
			ps281.OverlayValues[111] = d111
			ps281.OverlayValues[112] = d112
			ps281.OverlayValues[113] = d113
			ps281.OverlayValues[114] = d114
			ps281.OverlayValues[115] = d115
			ps281.OverlayValues[116] = d116
			ps281.OverlayValues[117] = d117
			ps281.OverlayValues[118] = d118
			ps281.OverlayValues[119] = d119
			ps281.OverlayValues[120] = d120
			ps281.OverlayValues[121] = d121
			ps281.OverlayValues[122] = d122
			ps281.OverlayValues[123] = d123
			ps281.OverlayValues[124] = d124
			ps281.OverlayValues[125] = d125
			ps281.OverlayValues[126] = d126
			ps281.OverlayValues[127] = d127
			ps281.OverlayValues[128] = d128
			ps281.OverlayValues[129] = d129
			ps281.OverlayValues[130] = d130
			ps281.OverlayValues[131] = d131
			ps281.OverlayValues[132] = d132
			ps281.OverlayValues[133] = d133
			ps281.OverlayValues[134] = d134
			ps281.OverlayValues[135] = d135
			ps281.OverlayValues[136] = d136
			ps281.OverlayValues[137] = d137
			ps281.OverlayValues[138] = d138
			ps281.OverlayValues[139] = d139
			ps281.OverlayValues[140] = d140
			ps281.OverlayValues[141] = d141
			ps281.OverlayValues[142] = d142
			ps281.OverlayValues[143] = d143
			ps281.OverlayValues[144] = d144
			ps281.OverlayValues[145] = d145
			ps281.OverlayValues[146] = d146
			ps281.OverlayValues[153] = d153
			ps281.OverlayValues[154] = d154
			ps281.OverlayValues[160] = d160
			ps281.OverlayValues[161] = d161
			ps281.OverlayValues[162] = d162
			ps281.OverlayValues[163] = d163
			ps281.OverlayValues[164] = d164
			ps281.OverlayValues[165] = d165
			ps281.OverlayValues[166] = d166
			ps281.OverlayValues[168] = d168
			ps281.OverlayValues[170] = d170
			ps281.OverlayValues[171] = d171
			ps281.OverlayValues[174] = d174
			ps281.OverlayValues[177] = d177
			ps281.OverlayValues[178] = d178
			ps281.OverlayValues[179] = d179
			ps281.OverlayValues[181] = d181
			ps281.OverlayValues[182] = d182
			ps281.OverlayValues[183] = d183
			ps281.OverlayValues[184] = d184
			ps281.OverlayValues[185] = d185
			ps281.OverlayValues[186] = d186
			ps281.OverlayValues[188] = d188
			ps281.OverlayValues[189] = d189
			ps281.OverlayValues[190] = d190
			ps281.OverlayValues[191] = d191
			ps281.OverlayValues[192] = d192
			ps281.OverlayValues[193] = d193
			ps281.OverlayValues[194] = d194
			ps281.OverlayValues[195] = d195
			ps281.OverlayValues[196] = d196
			ps281.OverlayValues[199] = d199
			ps281.OverlayValues[200] = d200
			ps281.OverlayValues[201] = d201
			ps281.OverlayValues[204] = d204
			ps281.OverlayValues[205] = d205
			ps281.OverlayValues[206] = d206
			ps281.OverlayValues[207] = d207
			ps281.OverlayValues[208] = d208
			ps281.OverlayValues[209] = d209
			ps281.OverlayValues[210] = d210
			ps281.OverlayValues[211] = d211
			ps281.OverlayValues[212] = d212
			ps281.OverlayValues[214] = d214
			ps281.OverlayValues[215] = d215
			ps281.OverlayValues[216] = d216
			ps281.OverlayValues[217] = d217
			ps281.OverlayValues[218] = d218
			ps281.OverlayValues[219] = d219
			ps281.OverlayValues[220] = d220
			ps281.OverlayValues[221] = d221
			ps281.OverlayValues[222] = d222
			ps281.OverlayValues[223] = d223
			ps281.OverlayValues[225] = d225
			ps281.OverlayValues[226] = d226
			ps281.OverlayValues[227] = d227
			ps281.OverlayValues[228] = d228
			ps281.OverlayValues[229] = d229
			ps281.OverlayValues[230] = d230
			ps281.OverlayValues[231] = d231
			ps281.OverlayValues[232] = d232
			ps281.OverlayValues[233] = d233
			ps281.OverlayValues[234] = d234
			ps281.OverlayValues[235] = d235
			ps281.OverlayValues[236] = d236
			ps281.OverlayValues[237] = d237
			ps281.OverlayValues[238] = d238
			ps281.OverlayValues[239] = d239
			ps281.OverlayValues[240] = d240
			ps281.OverlayValues[241] = d241
			ps281.OverlayValues[242] = d242
			ps281.OverlayValues[243] = d243
			ps281.OverlayValues[244] = d244
			ps281.OverlayValues[245] = d245
			ps281.OverlayValues[246] = d246
			ps281.OverlayValues[247] = d247
			ps281.OverlayValues[248] = d248
			ps281.OverlayValues[249] = d249
			ps281.OverlayValues[250] = d250
			ps281.OverlayValues[251] = d251
			ps281.OverlayValues[252] = d252
			ps281.OverlayValues[253] = d253
			ps281.OverlayValues[254] = d254
			ps281.OverlayValues[255] = d255
			ps281.OverlayValues[256] = d256
			ps281.OverlayValues[257] = d257
			ps281.OverlayValues[258] = d258
			ps281.OverlayValues[259] = d259
			ps281.OverlayValues[260] = d260
			ps281.OverlayValues[261] = d261
			ps281.OverlayValues[262] = d262
			ps281.OverlayValues[263] = d263
			ps281.OverlayValues[264] = d264
			ps281.OverlayValues[265] = d265
			ps281.OverlayValues[266] = d266
			ps281.OverlayValues[267] = d267
			ps281.OverlayValues[268] = d268
			ps281.OverlayValues[269] = d269
			ps281.OverlayValues[270] = d270
			ps281.OverlayValues[271] = d271
			ps281.OverlayValues[278] = d278
			ps281.OverlayValues[279] = d279
				return bbs[18].RenderPS(ps281)
			}
			lbl64 := ctx.W.ReserveLabel()
			lbl65 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d279.Reg, 0)
			ctx.W.EmitJcc(scm.CcNE, lbl64)
			ctx.W.EmitJmp(lbl65)
			ctx.W.MarkLabel(lbl64)
			ctx.W.EmitJmp(lbl18)
			ctx.W.MarkLabel(lbl65)
			ctx.W.EmitJmp(lbl19)
			ps282 := scm.PhiState{General: true}
			ps282.OverlayValues = make([]scm.JITValueDesc, 280)
			ps282.OverlayValues[0] = d0
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
			ps282.OverlayValues[20] = d20
			ps282.OverlayValues[21] = d21
			ps282.OverlayValues[22] = d22
			ps282.OverlayValues[24] = d24
			ps282.OverlayValues[25] = d25
			ps282.OverlayValues[27] = d27
			ps282.OverlayValues[28] = d28
			ps282.OverlayValues[29] = d29
			ps282.OverlayValues[32] = d32
			ps282.OverlayValues[34] = d34
			ps282.OverlayValues[35] = d35
			ps282.OverlayValues[36] = d36
			ps282.OverlayValues[38] = d38
			ps282.OverlayValues[39] = d39
			ps282.OverlayValues[40] = d40
			ps282.OverlayValues[41] = d41
			ps282.OverlayValues[42] = d42
			ps282.OverlayValues[43] = d43
			ps282.OverlayValues[44] = d44
			ps282.OverlayValues[46] = d46
			ps282.OverlayValues[47] = d47
			ps282.OverlayValues[48] = d48
			ps282.OverlayValues[49] = d49
			ps282.OverlayValues[50] = d50
			ps282.OverlayValues[51] = d51
			ps282.OverlayValues[52] = d52
			ps282.OverlayValues[53] = d53
			ps282.OverlayValues[54] = d54
			ps282.OverlayValues[55] = d55
			ps282.OverlayValues[56] = d56
			ps282.OverlayValues[57] = d57
			ps282.OverlayValues[58] = d58
			ps282.OverlayValues[59] = d59
			ps282.OverlayValues[60] = d60
			ps282.OverlayValues[61] = d61
			ps282.OverlayValues[62] = d62
			ps282.OverlayValues[63] = d63
			ps282.OverlayValues[64] = d64
			ps282.OverlayValues[65] = d65
			ps282.OverlayValues[66] = d66
			ps282.OverlayValues[67] = d67
			ps282.OverlayValues[68] = d68
			ps282.OverlayValues[69] = d69
			ps282.OverlayValues[70] = d70
			ps282.OverlayValues[71] = d71
			ps282.OverlayValues[72] = d72
			ps282.OverlayValues[73] = d73
			ps282.OverlayValues[74] = d74
			ps282.OverlayValues[75] = d75
			ps282.OverlayValues[76] = d76
			ps282.OverlayValues[77] = d77
			ps282.OverlayValues[78] = d78
			ps282.OverlayValues[79] = d79
			ps282.OverlayValues[80] = d80
			ps282.OverlayValues[81] = d81
			ps282.OverlayValues[82] = d82
			ps282.OverlayValues[83] = d83
			ps282.OverlayValues[84] = d84
			ps282.OverlayValues[85] = d85
			ps282.OverlayValues[86] = d86
			ps282.OverlayValues[87] = d87
			ps282.OverlayValues[88] = d88
			ps282.OverlayValues[89] = d89
			ps282.OverlayValues[90] = d90
			ps282.OverlayValues[91] = d91
			ps282.OverlayValues[92] = d92
			ps282.OverlayValues[93] = d93
			ps282.OverlayValues[94] = d94
			ps282.OverlayValues[95] = d95
			ps282.OverlayValues[102] = d102
			ps282.OverlayValues[103] = d103
			ps282.OverlayValues[104] = d104
			ps282.OverlayValues[105] = d105
			ps282.OverlayValues[106] = d106
			ps282.OverlayValues[107] = d107
			ps282.OverlayValues[108] = d108
			ps282.OverlayValues[109] = d109
			ps282.OverlayValues[110] = d110
			ps282.OverlayValues[111] = d111
			ps282.OverlayValues[112] = d112
			ps282.OverlayValues[113] = d113
			ps282.OverlayValues[114] = d114
			ps282.OverlayValues[115] = d115
			ps282.OverlayValues[116] = d116
			ps282.OverlayValues[117] = d117
			ps282.OverlayValues[118] = d118
			ps282.OverlayValues[119] = d119
			ps282.OverlayValues[120] = d120
			ps282.OverlayValues[121] = d121
			ps282.OverlayValues[122] = d122
			ps282.OverlayValues[123] = d123
			ps282.OverlayValues[124] = d124
			ps282.OverlayValues[125] = d125
			ps282.OverlayValues[126] = d126
			ps282.OverlayValues[127] = d127
			ps282.OverlayValues[128] = d128
			ps282.OverlayValues[129] = d129
			ps282.OverlayValues[130] = d130
			ps282.OverlayValues[131] = d131
			ps282.OverlayValues[132] = d132
			ps282.OverlayValues[133] = d133
			ps282.OverlayValues[134] = d134
			ps282.OverlayValues[135] = d135
			ps282.OverlayValues[136] = d136
			ps282.OverlayValues[137] = d137
			ps282.OverlayValues[138] = d138
			ps282.OverlayValues[139] = d139
			ps282.OverlayValues[140] = d140
			ps282.OverlayValues[141] = d141
			ps282.OverlayValues[142] = d142
			ps282.OverlayValues[143] = d143
			ps282.OverlayValues[144] = d144
			ps282.OverlayValues[145] = d145
			ps282.OverlayValues[146] = d146
			ps282.OverlayValues[153] = d153
			ps282.OverlayValues[154] = d154
			ps282.OverlayValues[160] = d160
			ps282.OverlayValues[161] = d161
			ps282.OverlayValues[162] = d162
			ps282.OverlayValues[163] = d163
			ps282.OverlayValues[164] = d164
			ps282.OverlayValues[165] = d165
			ps282.OverlayValues[166] = d166
			ps282.OverlayValues[168] = d168
			ps282.OverlayValues[170] = d170
			ps282.OverlayValues[171] = d171
			ps282.OverlayValues[174] = d174
			ps282.OverlayValues[177] = d177
			ps282.OverlayValues[178] = d178
			ps282.OverlayValues[179] = d179
			ps282.OverlayValues[181] = d181
			ps282.OverlayValues[182] = d182
			ps282.OverlayValues[183] = d183
			ps282.OverlayValues[184] = d184
			ps282.OverlayValues[185] = d185
			ps282.OverlayValues[186] = d186
			ps282.OverlayValues[188] = d188
			ps282.OverlayValues[189] = d189
			ps282.OverlayValues[190] = d190
			ps282.OverlayValues[191] = d191
			ps282.OverlayValues[192] = d192
			ps282.OverlayValues[193] = d193
			ps282.OverlayValues[194] = d194
			ps282.OverlayValues[195] = d195
			ps282.OverlayValues[196] = d196
			ps282.OverlayValues[199] = d199
			ps282.OverlayValues[200] = d200
			ps282.OverlayValues[201] = d201
			ps282.OverlayValues[204] = d204
			ps282.OverlayValues[205] = d205
			ps282.OverlayValues[206] = d206
			ps282.OverlayValues[207] = d207
			ps282.OverlayValues[208] = d208
			ps282.OverlayValues[209] = d209
			ps282.OverlayValues[210] = d210
			ps282.OverlayValues[211] = d211
			ps282.OverlayValues[212] = d212
			ps282.OverlayValues[214] = d214
			ps282.OverlayValues[215] = d215
			ps282.OverlayValues[216] = d216
			ps282.OverlayValues[217] = d217
			ps282.OverlayValues[218] = d218
			ps282.OverlayValues[219] = d219
			ps282.OverlayValues[220] = d220
			ps282.OverlayValues[221] = d221
			ps282.OverlayValues[222] = d222
			ps282.OverlayValues[223] = d223
			ps282.OverlayValues[225] = d225
			ps282.OverlayValues[226] = d226
			ps282.OverlayValues[227] = d227
			ps282.OverlayValues[228] = d228
			ps282.OverlayValues[229] = d229
			ps282.OverlayValues[230] = d230
			ps282.OverlayValues[231] = d231
			ps282.OverlayValues[232] = d232
			ps282.OverlayValues[233] = d233
			ps282.OverlayValues[234] = d234
			ps282.OverlayValues[235] = d235
			ps282.OverlayValues[236] = d236
			ps282.OverlayValues[237] = d237
			ps282.OverlayValues[238] = d238
			ps282.OverlayValues[239] = d239
			ps282.OverlayValues[240] = d240
			ps282.OverlayValues[241] = d241
			ps282.OverlayValues[242] = d242
			ps282.OverlayValues[243] = d243
			ps282.OverlayValues[244] = d244
			ps282.OverlayValues[245] = d245
			ps282.OverlayValues[246] = d246
			ps282.OverlayValues[247] = d247
			ps282.OverlayValues[248] = d248
			ps282.OverlayValues[249] = d249
			ps282.OverlayValues[250] = d250
			ps282.OverlayValues[251] = d251
			ps282.OverlayValues[252] = d252
			ps282.OverlayValues[253] = d253
			ps282.OverlayValues[254] = d254
			ps282.OverlayValues[255] = d255
			ps282.OverlayValues[256] = d256
			ps282.OverlayValues[257] = d257
			ps282.OverlayValues[258] = d258
			ps282.OverlayValues[259] = d259
			ps282.OverlayValues[260] = d260
			ps282.OverlayValues[261] = d261
			ps282.OverlayValues[262] = d262
			ps282.OverlayValues[263] = d263
			ps282.OverlayValues[264] = d264
			ps282.OverlayValues[265] = d265
			ps282.OverlayValues[266] = d266
			ps282.OverlayValues[267] = d267
			ps282.OverlayValues[268] = d268
			ps282.OverlayValues[269] = d269
			ps282.OverlayValues[270] = d270
			ps282.OverlayValues[271] = d271
			ps282.OverlayValues[278] = d278
			ps282.OverlayValues[279] = d279
			ps283 := scm.PhiState{General: true}
			ps283.OverlayValues = make([]scm.JITValueDesc, 280)
			ps283.OverlayValues[0] = d0
			ps283.OverlayValues[1] = d1
			ps283.OverlayValues[2] = d2
			ps283.OverlayValues[3] = d3
			ps283.OverlayValues[4] = d4
			ps283.OverlayValues[5] = d5
			ps283.OverlayValues[6] = d6
			ps283.OverlayValues[7] = d7
			ps283.OverlayValues[8] = d8
			ps283.OverlayValues[9] = d9
			ps283.OverlayValues[10] = d10
			ps283.OverlayValues[11] = d11
			ps283.OverlayValues[12] = d12
			ps283.OverlayValues[13] = d13
			ps283.OverlayValues[14] = d14
			ps283.OverlayValues[20] = d20
			ps283.OverlayValues[21] = d21
			ps283.OverlayValues[22] = d22
			ps283.OverlayValues[24] = d24
			ps283.OverlayValues[25] = d25
			ps283.OverlayValues[27] = d27
			ps283.OverlayValues[28] = d28
			ps283.OverlayValues[29] = d29
			ps283.OverlayValues[32] = d32
			ps283.OverlayValues[34] = d34
			ps283.OverlayValues[35] = d35
			ps283.OverlayValues[36] = d36
			ps283.OverlayValues[38] = d38
			ps283.OverlayValues[39] = d39
			ps283.OverlayValues[40] = d40
			ps283.OverlayValues[41] = d41
			ps283.OverlayValues[42] = d42
			ps283.OverlayValues[43] = d43
			ps283.OverlayValues[44] = d44
			ps283.OverlayValues[46] = d46
			ps283.OverlayValues[47] = d47
			ps283.OverlayValues[48] = d48
			ps283.OverlayValues[49] = d49
			ps283.OverlayValues[50] = d50
			ps283.OverlayValues[51] = d51
			ps283.OverlayValues[52] = d52
			ps283.OverlayValues[53] = d53
			ps283.OverlayValues[54] = d54
			ps283.OverlayValues[55] = d55
			ps283.OverlayValues[56] = d56
			ps283.OverlayValues[57] = d57
			ps283.OverlayValues[58] = d58
			ps283.OverlayValues[59] = d59
			ps283.OverlayValues[60] = d60
			ps283.OverlayValues[61] = d61
			ps283.OverlayValues[62] = d62
			ps283.OverlayValues[63] = d63
			ps283.OverlayValues[64] = d64
			ps283.OverlayValues[65] = d65
			ps283.OverlayValues[66] = d66
			ps283.OverlayValues[67] = d67
			ps283.OverlayValues[68] = d68
			ps283.OverlayValues[69] = d69
			ps283.OverlayValues[70] = d70
			ps283.OverlayValues[71] = d71
			ps283.OverlayValues[72] = d72
			ps283.OverlayValues[73] = d73
			ps283.OverlayValues[74] = d74
			ps283.OverlayValues[75] = d75
			ps283.OverlayValues[76] = d76
			ps283.OverlayValues[77] = d77
			ps283.OverlayValues[78] = d78
			ps283.OverlayValues[79] = d79
			ps283.OverlayValues[80] = d80
			ps283.OverlayValues[81] = d81
			ps283.OverlayValues[82] = d82
			ps283.OverlayValues[83] = d83
			ps283.OverlayValues[84] = d84
			ps283.OverlayValues[85] = d85
			ps283.OverlayValues[86] = d86
			ps283.OverlayValues[87] = d87
			ps283.OverlayValues[88] = d88
			ps283.OverlayValues[89] = d89
			ps283.OverlayValues[90] = d90
			ps283.OverlayValues[91] = d91
			ps283.OverlayValues[92] = d92
			ps283.OverlayValues[93] = d93
			ps283.OverlayValues[94] = d94
			ps283.OverlayValues[95] = d95
			ps283.OverlayValues[102] = d102
			ps283.OverlayValues[103] = d103
			ps283.OverlayValues[104] = d104
			ps283.OverlayValues[105] = d105
			ps283.OverlayValues[106] = d106
			ps283.OverlayValues[107] = d107
			ps283.OverlayValues[108] = d108
			ps283.OverlayValues[109] = d109
			ps283.OverlayValues[110] = d110
			ps283.OverlayValues[111] = d111
			ps283.OverlayValues[112] = d112
			ps283.OverlayValues[113] = d113
			ps283.OverlayValues[114] = d114
			ps283.OverlayValues[115] = d115
			ps283.OverlayValues[116] = d116
			ps283.OverlayValues[117] = d117
			ps283.OverlayValues[118] = d118
			ps283.OverlayValues[119] = d119
			ps283.OverlayValues[120] = d120
			ps283.OverlayValues[121] = d121
			ps283.OverlayValues[122] = d122
			ps283.OverlayValues[123] = d123
			ps283.OverlayValues[124] = d124
			ps283.OverlayValues[125] = d125
			ps283.OverlayValues[126] = d126
			ps283.OverlayValues[127] = d127
			ps283.OverlayValues[128] = d128
			ps283.OverlayValues[129] = d129
			ps283.OverlayValues[130] = d130
			ps283.OverlayValues[131] = d131
			ps283.OverlayValues[132] = d132
			ps283.OverlayValues[133] = d133
			ps283.OverlayValues[134] = d134
			ps283.OverlayValues[135] = d135
			ps283.OverlayValues[136] = d136
			ps283.OverlayValues[137] = d137
			ps283.OverlayValues[138] = d138
			ps283.OverlayValues[139] = d139
			ps283.OverlayValues[140] = d140
			ps283.OverlayValues[141] = d141
			ps283.OverlayValues[142] = d142
			ps283.OverlayValues[143] = d143
			ps283.OverlayValues[144] = d144
			ps283.OverlayValues[145] = d145
			ps283.OverlayValues[146] = d146
			ps283.OverlayValues[153] = d153
			ps283.OverlayValues[154] = d154
			ps283.OverlayValues[160] = d160
			ps283.OverlayValues[161] = d161
			ps283.OverlayValues[162] = d162
			ps283.OverlayValues[163] = d163
			ps283.OverlayValues[164] = d164
			ps283.OverlayValues[165] = d165
			ps283.OverlayValues[166] = d166
			ps283.OverlayValues[168] = d168
			ps283.OverlayValues[170] = d170
			ps283.OverlayValues[171] = d171
			ps283.OverlayValues[174] = d174
			ps283.OverlayValues[177] = d177
			ps283.OverlayValues[178] = d178
			ps283.OverlayValues[179] = d179
			ps283.OverlayValues[181] = d181
			ps283.OverlayValues[182] = d182
			ps283.OverlayValues[183] = d183
			ps283.OverlayValues[184] = d184
			ps283.OverlayValues[185] = d185
			ps283.OverlayValues[186] = d186
			ps283.OverlayValues[188] = d188
			ps283.OverlayValues[189] = d189
			ps283.OverlayValues[190] = d190
			ps283.OverlayValues[191] = d191
			ps283.OverlayValues[192] = d192
			ps283.OverlayValues[193] = d193
			ps283.OverlayValues[194] = d194
			ps283.OverlayValues[195] = d195
			ps283.OverlayValues[196] = d196
			ps283.OverlayValues[199] = d199
			ps283.OverlayValues[200] = d200
			ps283.OverlayValues[201] = d201
			ps283.OverlayValues[204] = d204
			ps283.OverlayValues[205] = d205
			ps283.OverlayValues[206] = d206
			ps283.OverlayValues[207] = d207
			ps283.OverlayValues[208] = d208
			ps283.OverlayValues[209] = d209
			ps283.OverlayValues[210] = d210
			ps283.OverlayValues[211] = d211
			ps283.OverlayValues[212] = d212
			ps283.OverlayValues[214] = d214
			ps283.OverlayValues[215] = d215
			ps283.OverlayValues[216] = d216
			ps283.OverlayValues[217] = d217
			ps283.OverlayValues[218] = d218
			ps283.OverlayValues[219] = d219
			ps283.OverlayValues[220] = d220
			ps283.OverlayValues[221] = d221
			ps283.OverlayValues[222] = d222
			ps283.OverlayValues[223] = d223
			ps283.OverlayValues[225] = d225
			ps283.OverlayValues[226] = d226
			ps283.OverlayValues[227] = d227
			ps283.OverlayValues[228] = d228
			ps283.OverlayValues[229] = d229
			ps283.OverlayValues[230] = d230
			ps283.OverlayValues[231] = d231
			ps283.OverlayValues[232] = d232
			ps283.OverlayValues[233] = d233
			ps283.OverlayValues[234] = d234
			ps283.OverlayValues[235] = d235
			ps283.OverlayValues[236] = d236
			ps283.OverlayValues[237] = d237
			ps283.OverlayValues[238] = d238
			ps283.OverlayValues[239] = d239
			ps283.OverlayValues[240] = d240
			ps283.OverlayValues[241] = d241
			ps283.OverlayValues[242] = d242
			ps283.OverlayValues[243] = d243
			ps283.OverlayValues[244] = d244
			ps283.OverlayValues[245] = d245
			ps283.OverlayValues[246] = d246
			ps283.OverlayValues[247] = d247
			ps283.OverlayValues[248] = d248
			ps283.OverlayValues[249] = d249
			ps283.OverlayValues[250] = d250
			ps283.OverlayValues[251] = d251
			ps283.OverlayValues[252] = d252
			ps283.OverlayValues[253] = d253
			ps283.OverlayValues[254] = d254
			ps283.OverlayValues[255] = d255
			ps283.OverlayValues[256] = d256
			ps283.OverlayValues[257] = d257
			ps283.OverlayValues[258] = d258
			ps283.OverlayValues[259] = d259
			ps283.OverlayValues[260] = d260
			ps283.OverlayValues[261] = d261
			ps283.OverlayValues[262] = d262
			ps283.OverlayValues[263] = d263
			ps283.OverlayValues[264] = d264
			ps283.OverlayValues[265] = d265
			ps283.OverlayValues[266] = d266
			ps283.OverlayValues[267] = d267
			ps283.OverlayValues[268] = d268
			ps283.OverlayValues[269] = d269
			ps283.OverlayValues[270] = d270
			ps283.OverlayValues[271] = d271
			ps283.OverlayValues[278] = d278
			ps283.OverlayValues[279] = d279
			alloc284 := ctx.SnapshotAllocState()
			if !bbs[18].Rendered {
				bbs[18].RenderPS(ps283)
			}
			ctx.RestoreAllocState(alloc284)
			if !bbs[17].Rendered {
				return bbs[17].RenderPS(ps282)
			}
			return result
			ctx.FreeDesc(&d278)
			return result
			}
			bbs[15].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[15].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
						d285 := ps.PhiValues[0]
						ctx.EnsureDesc(&d285)
						ctx.EmitStoreToStack(d285, 64)
					}
					if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
						d286 := ps.PhiValues[1]
						ctx.EnsureDesc(&d286)
						ctx.EmitStoreToStack(d286, 72)
					}
					ps.General = true
					return bbs[15].RenderPS(ps)
				}
			}
			bbs[15].VisitCount++
			if ps.General {
				if bbs[15].Rendered {
					ctx.W.EmitJmp(lbl16)
					return result
				}
				bbs[15].Rendered = true
				bbs[15].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_15 = bbs[15].Address
				ctx.W.MarkLabel(lbl16)
				ctx.W.ResolveFixups()
			}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
				d154 = ps.OverlayValues[154]
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
			if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
				d168 = ps.OverlayValues[168]
			}
			if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != scm.LocNone {
				d170 = ps.OverlayValues[170]
			}
			if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != scm.LocNone {
				d171 = ps.OverlayValues[171]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
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
			if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
				d199 = ps.OverlayValues[199]
			}
			if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
				d200 = ps.OverlayValues[200]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
				d201 = ps.OverlayValues[201]
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
			if len(ps.OverlayValues) > 254 && ps.OverlayValues[254].Loc != scm.LocNone {
				d254 = ps.OverlayValues[254]
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
			if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != scm.LocNone {
				d278 = ps.OverlayValues[278]
			}
			if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != scm.LocNone {
				d279 = ps.OverlayValues[279]
			}
			if len(ps.OverlayValues) > 285 && ps.OverlayValues[285].Loc != scm.LocNone {
				d285 = ps.OverlayValues[285]
			}
			if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != scm.LocNone {
				d286 = ps.OverlayValues[286]
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
			var d287 scm.JITValueDesc
			if d8.Loc == scm.LocImm && d9.Loc == scm.LocImm {
				d287 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d8.Imm.Int()) == uint64(d9.Imm.Int()))}
			} else if d9.Loc == scm.LocImm {
				r155 := ctx.AllocRegExcept(d8.Reg)
				if d9.Imm.Int() >= -2147483648 && d9.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d8.Reg, int32(d9.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d9.Imm.Int()))
					ctx.W.EmitCmpInt64(d8.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r155, scm.CcE)
				d287 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r155}
				ctx.BindReg(r155, &d287)
			} else if d8.Loc == scm.LocImm {
				r156 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d8.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d9.Reg)
				ctx.W.EmitSetcc(r156, scm.CcE)
				d287 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r156}
				ctx.BindReg(r156, &d287)
			} else {
				r157 := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitCmpInt64(d8.Reg, d9.Reg)
				ctx.W.EmitSetcc(r157, scm.CcE)
				d287 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r157}
				ctx.BindReg(r157, &d287)
			}
			d288 = d287
			ctx.EnsureDesc(&d288)
			if d288.Loc != scm.LocImm && d288.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d288.Loc == scm.LocImm {
				if d288.Imm.Bool() {
			d289 = d8
			if d289.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d289)
			d290 = d289
			if d290.Loc == scm.LocImm {
				d290 = scm.JITValueDesc{Loc: scm.LocImm, Type: d290.Type, Imm: scm.NewInt(int64(uint64(d290.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d290.Reg, 32)
				ctx.W.EmitShrRegImm8(d290.Reg, 32)
			}
			ctx.EmitStoreToStack(d290, 32)
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
			ps291.OverlayValues[20] = d20
			ps291.OverlayValues[21] = d21
			ps291.OverlayValues[22] = d22
			ps291.OverlayValues[24] = d24
			ps291.OverlayValues[25] = d25
			ps291.OverlayValues[27] = d27
			ps291.OverlayValues[28] = d28
			ps291.OverlayValues[29] = d29
			ps291.OverlayValues[32] = d32
			ps291.OverlayValues[34] = d34
			ps291.OverlayValues[35] = d35
			ps291.OverlayValues[36] = d36
			ps291.OverlayValues[38] = d38
			ps291.OverlayValues[39] = d39
			ps291.OverlayValues[40] = d40
			ps291.OverlayValues[41] = d41
			ps291.OverlayValues[42] = d42
			ps291.OverlayValues[43] = d43
			ps291.OverlayValues[44] = d44
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
			ps291.OverlayValues[62] = d62
			ps291.OverlayValues[63] = d63
			ps291.OverlayValues[64] = d64
			ps291.OverlayValues[65] = d65
			ps291.OverlayValues[66] = d66
			ps291.OverlayValues[67] = d67
			ps291.OverlayValues[68] = d68
			ps291.OverlayValues[69] = d69
			ps291.OverlayValues[70] = d70
			ps291.OverlayValues[71] = d71
			ps291.OverlayValues[72] = d72
			ps291.OverlayValues[73] = d73
			ps291.OverlayValues[74] = d74
			ps291.OverlayValues[75] = d75
			ps291.OverlayValues[76] = d76
			ps291.OverlayValues[77] = d77
			ps291.OverlayValues[78] = d78
			ps291.OverlayValues[79] = d79
			ps291.OverlayValues[80] = d80
			ps291.OverlayValues[81] = d81
			ps291.OverlayValues[82] = d82
			ps291.OverlayValues[83] = d83
			ps291.OverlayValues[84] = d84
			ps291.OverlayValues[85] = d85
			ps291.OverlayValues[86] = d86
			ps291.OverlayValues[87] = d87
			ps291.OverlayValues[88] = d88
			ps291.OverlayValues[89] = d89
			ps291.OverlayValues[90] = d90
			ps291.OverlayValues[91] = d91
			ps291.OverlayValues[92] = d92
			ps291.OverlayValues[93] = d93
			ps291.OverlayValues[94] = d94
			ps291.OverlayValues[95] = d95
			ps291.OverlayValues[102] = d102
			ps291.OverlayValues[103] = d103
			ps291.OverlayValues[104] = d104
			ps291.OverlayValues[105] = d105
			ps291.OverlayValues[106] = d106
			ps291.OverlayValues[107] = d107
			ps291.OverlayValues[108] = d108
			ps291.OverlayValues[109] = d109
			ps291.OverlayValues[110] = d110
			ps291.OverlayValues[111] = d111
			ps291.OverlayValues[112] = d112
			ps291.OverlayValues[113] = d113
			ps291.OverlayValues[114] = d114
			ps291.OverlayValues[115] = d115
			ps291.OverlayValues[116] = d116
			ps291.OverlayValues[117] = d117
			ps291.OverlayValues[118] = d118
			ps291.OverlayValues[119] = d119
			ps291.OverlayValues[120] = d120
			ps291.OverlayValues[121] = d121
			ps291.OverlayValues[122] = d122
			ps291.OverlayValues[123] = d123
			ps291.OverlayValues[124] = d124
			ps291.OverlayValues[125] = d125
			ps291.OverlayValues[126] = d126
			ps291.OverlayValues[127] = d127
			ps291.OverlayValues[128] = d128
			ps291.OverlayValues[129] = d129
			ps291.OverlayValues[130] = d130
			ps291.OverlayValues[131] = d131
			ps291.OverlayValues[132] = d132
			ps291.OverlayValues[133] = d133
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
			ps291.OverlayValues[153] = d153
			ps291.OverlayValues[154] = d154
			ps291.OverlayValues[160] = d160
			ps291.OverlayValues[161] = d161
			ps291.OverlayValues[162] = d162
			ps291.OverlayValues[163] = d163
			ps291.OverlayValues[164] = d164
			ps291.OverlayValues[165] = d165
			ps291.OverlayValues[166] = d166
			ps291.OverlayValues[168] = d168
			ps291.OverlayValues[170] = d170
			ps291.OverlayValues[171] = d171
			ps291.OverlayValues[174] = d174
			ps291.OverlayValues[177] = d177
			ps291.OverlayValues[178] = d178
			ps291.OverlayValues[179] = d179
			ps291.OverlayValues[181] = d181
			ps291.OverlayValues[182] = d182
			ps291.OverlayValues[183] = d183
			ps291.OverlayValues[184] = d184
			ps291.OverlayValues[185] = d185
			ps291.OverlayValues[186] = d186
			ps291.OverlayValues[188] = d188
			ps291.OverlayValues[189] = d189
			ps291.OverlayValues[190] = d190
			ps291.OverlayValues[191] = d191
			ps291.OverlayValues[192] = d192
			ps291.OverlayValues[193] = d193
			ps291.OverlayValues[194] = d194
			ps291.OverlayValues[195] = d195
			ps291.OverlayValues[196] = d196
			ps291.OverlayValues[199] = d199
			ps291.OverlayValues[200] = d200
			ps291.OverlayValues[201] = d201
			ps291.OverlayValues[204] = d204
			ps291.OverlayValues[205] = d205
			ps291.OverlayValues[206] = d206
			ps291.OverlayValues[207] = d207
			ps291.OverlayValues[208] = d208
			ps291.OverlayValues[209] = d209
			ps291.OverlayValues[210] = d210
			ps291.OverlayValues[211] = d211
			ps291.OverlayValues[212] = d212
			ps291.OverlayValues[214] = d214
			ps291.OverlayValues[215] = d215
			ps291.OverlayValues[216] = d216
			ps291.OverlayValues[217] = d217
			ps291.OverlayValues[218] = d218
			ps291.OverlayValues[219] = d219
			ps291.OverlayValues[220] = d220
			ps291.OverlayValues[221] = d221
			ps291.OverlayValues[222] = d222
			ps291.OverlayValues[223] = d223
			ps291.OverlayValues[225] = d225
			ps291.OverlayValues[226] = d226
			ps291.OverlayValues[227] = d227
			ps291.OverlayValues[228] = d228
			ps291.OverlayValues[229] = d229
			ps291.OverlayValues[230] = d230
			ps291.OverlayValues[231] = d231
			ps291.OverlayValues[232] = d232
			ps291.OverlayValues[233] = d233
			ps291.OverlayValues[234] = d234
			ps291.OverlayValues[235] = d235
			ps291.OverlayValues[236] = d236
			ps291.OverlayValues[237] = d237
			ps291.OverlayValues[238] = d238
			ps291.OverlayValues[239] = d239
			ps291.OverlayValues[240] = d240
			ps291.OverlayValues[241] = d241
			ps291.OverlayValues[242] = d242
			ps291.OverlayValues[243] = d243
			ps291.OverlayValues[244] = d244
			ps291.OverlayValues[245] = d245
			ps291.OverlayValues[246] = d246
			ps291.OverlayValues[247] = d247
			ps291.OverlayValues[248] = d248
			ps291.OverlayValues[249] = d249
			ps291.OverlayValues[250] = d250
			ps291.OverlayValues[251] = d251
			ps291.OverlayValues[252] = d252
			ps291.OverlayValues[253] = d253
			ps291.OverlayValues[254] = d254
			ps291.OverlayValues[255] = d255
			ps291.OverlayValues[256] = d256
			ps291.OverlayValues[257] = d257
			ps291.OverlayValues[258] = d258
			ps291.OverlayValues[259] = d259
			ps291.OverlayValues[260] = d260
			ps291.OverlayValues[261] = d261
			ps291.OverlayValues[262] = d262
			ps291.OverlayValues[263] = d263
			ps291.OverlayValues[264] = d264
			ps291.OverlayValues[265] = d265
			ps291.OverlayValues[266] = d266
			ps291.OverlayValues[267] = d267
			ps291.OverlayValues[268] = d268
			ps291.OverlayValues[269] = d269
			ps291.OverlayValues[270] = d270
			ps291.OverlayValues[271] = d271
			ps291.OverlayValues[278] = d278
			ps291.OverlayValues[279] = d279
			ps291.OverlayValues[285] = d285
			ps291.OverlayValues[286] = d286
			ps291.OverlayValues[287] = d287
			ps291.OverlayValues[288] = d288
			ps291.OverlayValues[289] = d289
			ps291.OverlayValues[290] = d290
			ps291.PhiValues = make([]scm.JITValueDesc, 1)
			d292 = d8
			ps291.PhiValues[0] = d292
					return bbs[6].RenderPS(ps291)
				}
			ps293 := scm.PhiState{General: ps.General}
			ps293.OverlayValues = make([]scm.JITValueDesc, 293)
			ps293.OverlayValues[0] = d0
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
			ps293.OverlayValues[20] = d20
			ps293.OverlayValues[21] = d21
			ps293.OverlayValues[22] = d22
			ps293.OverlayValues[24] = d24
			ps293.OverlayValues[25] = d25
			ps293.OverlayValues[27] = d27
			ps293.OverlayValues[28] = d28
			ps293.OverlayValues[29] = d29
			ps293.OverlayValues[32] = d32
			ps293.OverlayValues[34] = d34
			ps293.OverlayValues[35] = d35
			ps293.OverlayValues[36] = d36
			ps293.OverlayValues[38] = d38
			ps293.OverlayValues[39] = d39
			ps293.OverlayValues[40] = d40
			ps293.OverlayValues[41] = d41
			ps293.OverlayValues[42] = d42
			ps293.OverlayValues[43] = d43
			ps293.OverlayValues[44] = d44
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
			ps293.OverlayValues[60] = d60
			ps293.OverlayValues[61] = d61
			ps293.OverlayValues[62] = d62
			ps293.OverlayValues[63] = d63
			ps293.OverlayValues[64] = d64
			ps293.OverlayValues[65] = d65
			ps293.OverlayValues[66] = d66
			ps293.OverlayValues[67] = d67
			ps293.OverlayValues[68] = d68
			ps293.OverlayValues[69] = d69
			ps293.OverlayValues[70] = d70
			ps293.OverlayValues[71] = d71
			ps293.OverlayValues[72] = d72
			ps293.OverlayValues[73] = d73
			ps293.OverlayValues[74] = d74
			ps293.OverlayValues[75] = d75
			ps293.OverlayValues[76] = d76
			ps293.OverlayValues[77] = d77
			ps293.OverlayValues[78] = d78
			ps293.OverlayValues[79] = d79
			ps293.OverlayValues[80] = d80
			ps293.OverlayValues[81] = d81
			ps293.OverlayValues[82] = d82
			ps293.OverlayValues[83] = d83
			ps293.OverlayValues[84] = d84
			ps293.OverlayValues[85] = d85
			ps293.OverlayValues[86] = d86
			ps293.OverlayValues[87] = d87
			ps293.OverlayValues[88] = d88
			ps293.OverlayValues[89] = d89
			ps293.OverlayValues[90] = d90
			ps293.OverlayValues[91] = d91
			ps293.OverlayValues[92] = d92
			ps293.OverlayValues[93] = d93
			ps293.OverlayValues[94] = d94
			ps293.OverlayValues[95] = d95
			ps293.OverlayValues[102] = d102
			ps293.OverlayValues[103] = d103
			ps293.OverlayValues[104] = d104
			ps293.OverlayValues[105] = d105
			ps293.OverlayValues[106] = d106
			ps293.OverlayValues[107] = d107
			ps293.OverlayValues[108] = d108
			ps293.OverlayValues[109] = d109
			ps293.OverlayValues[110] = d110
			ps293.OverlayValues[111] = d111
			ps293.OverlayValues[112] = d112
			ps293.OverlayValues[113] = d113
			ps293.OverlayValues[114] = d114
			ps293.OverlayValues[115] = d115
			ps293.OverlayValues[116] = d116
			ps293.OverlayValues[117] = d117
			ps293.OverlayValues[118] = d118
			ps293.OverlayValues[119] = d119
			ps293.OverlayValues[120] = d120
			ps293.OverlayValues[121] = d121
			ps293.OverlayValues[122] = d122
			ps293.OverlayValues[123] = d123
			ps293.OverlayValues[124] = d124
			ps293.OverlayValues[125] = d125
			ps293.OverlayValues[126] = d126
			ps293.OverlayValues[127] = d127
			ps293.OverlayValues[128] = d128
			ps293.OverlayValues[129] = d129
			ps293.OverlayValues[130] = d130
			ps293.OverlayValues[131] = d131
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
			ps293.OverlayValues[153] = d153
			ps293.OverlayValues[154] = d154
			ps293.OverlayValues[160] = d160
			ps293.OverlayValues[161] = d161
			ps293.OverlayValues[162] = d162
			ps293.OverlayValues[163] = d163
			ps293.OverlayValues[164] = d164
			ps293.OverlayValues[165] = d165
			ps293.OverlayValues[166] = d166
			ps293.OverlayValues[168] = d168
			ps293.OverlayValues[170] = d170
			ps293.OverlayValues[171] = d171
			ps293.OverlayValues[174] = d174
			ps293.OverlayValues[177] = d177
			ps293.OverlayValues[178] = d178
			ps293.OverlayValues[179] = d179
			ps293.OverlayValues[181] = d181
			ps293.OverlayValues[182] = d182
			ps293.OverlayValues[183] = d183
			ps293.OverlayValues[184] = d184
			ps293.OverlayValues[185] = d185
			ps293.OverlayValues[186] = d186
			ps293.OverlayValues[188] = d188
			ps293.OverlayValues[189] = d189
			ps293.OverlayValues[190] = d190
			ps293.OverlayValues[191] = d191
			ps293.OverlayValues[192] = d192
			ps293.OverlayValues[193] = d193
			ps293.OverlayValues[194] = d194
			ps293.OverlayValues[195] = d195
			ps293.OverlayValues[196] = d196
			ps293.OverlayValues[199] = d199
			ps293.OverlayValues[200] = d200
			ps293.OverlayValues[201] = d201
			ps293.OverlayValues[204] = d204
			ps293.OverlayValues[205] = d205
			ps293.OverlayValues[206] = d206
			ps293.OverlayValues[207] = d207
			ps293.OverlayValues[208] = d208
			ps293.OverlayValues[209] = d209
			ps293.OverlayValues[210] = d210
			ps293.OverlayValues[211] = d211
			ps293.OverlayValues[212] = d212
			ps293.OverlayValues[214] = d214
			ps293.OverlayValues[215] = d215
			ps293.OverlayValues[216] = d216
			ps293.OverlayValues[217] = d217
			ps293.OverlayValues[218] = d218
			ps293.OverlayValues[219] = d219
			ps293.OverlayValues[220] = d220
			ps293.OverlayValues[221] = d221
			ps293.OverlayValues[222] = d222
			ps293.OverlayValues[223] = d223
			ps293.OverlayValues[225] = d225
			ps293.OverlayValues[226] = d226
			ps293.OverlayValues[227] = d227
			ps293.OverlayValues[228] = d228
			ps293.OverlayValues[229] = d229
			ps293.OverlayValues[230] = d230
			ps293.OverlayValues[231] = d231
			ps293.OverlayValues[232] = d232
			ps293.OverlayValues[233] = d233
			ps293.OverlayValues[234] = d234
			ps293.OverlayValues[235] = d235
			ps293.OverlayValues[236] = d236
			ps293.OverlayValues[237] = d237
			ps293.OverlayValues[238] = d238
			ps293.OverlayValues[239] = d239
			ps293.OverlayValues[240] = d240
			ps293.OverlayValues[241] = d241
			ps293.OverlayValues[242] = d242
			ps293.OverlayValues[243] = d243
			ps293.OverlayValues[244] = d244
			ps293.OverlayValues[245] = d245
			ps293.OverlayValues[246] = d246
			ps293.OverlayValues[247] = d247
			ps293.OverlayValues[248] = d248
			ps293.OverlayValues[249] = d249
			ps293.OverlayValues[250] = d250
			ps293.OverlayValues[251] = d251
			ps293.OverlayValues[252] = d252
			ps293.OverlayValues[253] = d253
			ps293.OverlayValues[254] = d254
			ps293.OverlayValues[255] = d255
			ps293.OverlayValues[256] = d256
			ps293.OverlayValues[257] = d257
			ps293.OverlayValues[258] = d258
			ps293.OverlayValues[259] = d259
			ps293.OverlayValues[260] = d260
			ps293.OverlayValues[261] = d261
			ps293.OverlayValues[262] = d262
			ps293.OverlayValues[263] = d263
			ps293.OverlayValues[264] = d264
			ps293.OverlayValues[265] = d265
			ps293.OverlayValues[266] = d266
			ps293.OverlayValues[267] = d267
			ps293.OverlayValues[268] = d268
			ps293.OverlayValues[269] = d269
			ps293.OverlayValues[270] = d270
			ps293.OverlayValues[271] = d271
			ps293.OverlayValues[278] = d278
			ps293.OverlayValues[279] = d279
			ps293.OverlayValues[285] = d285
			ps293.OverlayValues[286] = d286
			ps293.OverlayValues[287] = d287
			ps293.OverlayValues[288] = d288
			ps293.OverlayValues[289] = d289
			ps293.OverlayValues[290] = d290
			ps293.OverlayValues[292] = d292
				return bbs[19].RenderPS(ps293)
			}
			lbl66 := ctx.W.ReserveLabel()
			lbl67 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d288.Reg, 0)
			ctx.W.EmitJcc(scm.CcNE, lbl66)
			ctx.W.EmitJmp(lbl67)
			ctx.W.MarkLabel(lbl66)
			d294 = d8
			if d294.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d294)
			d295 = d294
			if d295.Loc == scm.LocImm {
				d295 = scm.JITValueDesc{Loc: scm.LocImm, Type: d295.Type, Imm: scm.NewInt(int64(uint64(d295.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d295.Reg, 32)
				ctx.W.EmitShrRegImm8(d295.Reg, 32)
			}
			ctx.EmitStoreToStack(d295, 32)
			ctx.W.EmitJmp(lbl7)
			ctx.W.MarkLabel(lbl67)
			ctx.W.EmitJmp(lbl20)
			ps296 := scm.PhiState{General: true}
			ps296.OverlayValues = make([]scm.JITValueDesc, 296)
			ps296.OverlayValues[0] = d0
			ps296.OverlayValues[1] = d1
			ps296.OverlayValues[2] = d2
			ps296.OverlayValues[3] = d3
			ps296.OverlayValues[4] = d4
			ps296.OverlayValues[5] = d5
			ps296.OverlayValues[6] = d6
			ps296.OverlayValues[7] = d7
			ps296.OverlayValues[8] = d8
			ps296.OverlayValues[9] = d9
			ps296.OverlayValues[10] = d10
			ps296.OverlayValues[11] = d11
			ps296.OverlayValues[12] = d12
			ps296.OverlayValues[13] = d13
			ps296.OverlayValues[14] = d14
			ps296.OverlayValues[20] = d20
			ps296.OverlayValues[21] = d21
			ps296.OverlayValues[22] = d22
			ps296.OverlayValues[24] = d24
			ps296.OverlayValues[25] = d25
			ps296.OverlayValues[27] = d27
			ps296.OverlayValues[28] = d28
			ps296.OverlayValues[29] = d29
			ps296.OverlayValues[32] = d32
			ps296.OverlayValues[34] = d34
			ps296.OverlayValues[35] = d35
			ps296.OverlayValues[36] = d36
			ps296.OverlayValues[38] = d38
			ps296.OverlayValues[39] = d39
			ps296.OverlayValues[40] = d40
			ps296.OverlayValues[41] = d41
			ps296.OverlayValues[42] = d42
			ps296.OverlayValues[43] = d43
			ps296.OverlayValues[44] = d44
			ps296.OverlayValues[46] = d46
			ps296.OverlayValues[47] = d47
			ps296.OverlayValues[48] = d48
			ps296.OverlayValues[49] = d49
			ps296.OverlayValues[50] = d50
			ps296.OverlayValues[51] = d51
			ps296.OverlayValues[52] = d52
			ps296.OverlayValues[53] = d53
			ps296.OverlayValues[54] = d54
			ps296.OverlayValues[55] = d55
			ps296.OverlayValues[56] = d56
			ps296.OverlayValues[57] = d57
			ps296.OverlayValues[58] = d58
			ps296.OverlayValues[59] = d59
			ps296.OverlayValues[60] = d60
			ps296.OverlayValues[61] = d61
			ps296.OverlayValues[62] = d62
			ps296.OverlayValues[63] = d63
			ps296.OverlayValues[64] = d64
			ps296.OverlayValues[65] = d65
			ps296.OverlayValues[66] = d66
			ps296.OverlayValues[67] = d67
			ps296.OverlayValues[68] = d68
			ps296.OverlayValues[69] = d69
			ps296.OverlayValues[70] = d70
			ps296.OverlayValues[71] = d71
			ps296.OverlayValues[72] = d72
			ps296.OverlayValues[73] = d73
			ps296.OverlayValues[74] = d74
			ps296.OverlayValues[75] = d75
			ps296.OverlayValues[76] = d76
			ps296.OverlayValues[77] = d77
			ps296.OverlayValues[78] = d78
			ps296.OverlayValues[79] = d79
			ps296.OverlayValues[80] = d80
			ps296.OverlayValues[81] = d81
			ps296.OverlayValues[82] = d82
			ps296.OverlayValues[83] = d83
			ps296.OverlayValues[84] = d84
			ps296.OverlayValues[85] = d85
			ps296.OverlayValues[86] = d86
			ps296.OverlayValues[87] = d87
			ps296.OverlayValues[88] = d88
			ps296.OverlayValues[89] = d89
			ps296.OverlayValues[90] = d90
			ps296.OverlayValues[91] = d91
			ps296.OverlayValues[92] = d92
			ps296.OverlayValues[93] = d93
			ps296.OverlayValues[94] = d94
			ps296.OverlayValues[95] = d95
			ps296.OverlayValues[102] = d102
			ps296.OverlayValues[103] = d103
			ps296.OverlayValues[104] = d104
			ps296.OverlayValues[105] = d105
			ps296.OverlayValues[106] = d106
			ps296.OverlayValues[107] = d107
			ps296.OverlayValues[108] = d108
			ps296.OverlayValues[109] = d109
			ps296.OverlayValues[110] = d110
			ps296.OverlayValues[111] = d111
			ps296.OverlayValues[112] = d112
			ps296.OverlayValues[113] = d113
			ps296.OverlayValues[114] = d114
			ps296.OverlayValues[115] = d115
			ps296.OverlayValues[116] = d116
			ps296.OverlayValues[117] = d117
			ps296.OverlayValues[118] = d118
			ps296.OverlayValues[119] = d119
			ps296.OverlayValues[120] = d120
			ps296.OverlayValues[121] = d121
			ps296.OverlayValues[122] = d122
			ps296.OverlayValues[123] = d123
			ps296.OverlayValues[124] = d124
			ps296.OverlayValues[125] = d125
			ps296.OverlayValues[126] = d126
			ps296.OverlayValues[127] = d127
			ps296.OverlayValues[128] = d128
			ps296.OverlayValues[129] = d129
			ps296.OverlayValues[130] = d130
			ps296.OverlayValues[131] = d131
			ps296.OverlayValues[132] = d132
			ps296.OverlayValues[133] = d133
			ps296.OverlayValues[134] = d134
			ps296.OverlayValues[135] = d135
			ps296.OverlayValues[136] = d136
			ps296.OverlayValues[137] = d137
			ps296.OverlayValues[138] = d138
			ps296.OverlayValues[139] = d139
			ps296.OverlayValues[140] = d140
			ps296.OverlayValues[141] = d141
			ps296.OverlayValues[142] = d142
			ps296.OverlayValues[143] = d143
			ps296.OverlayValues[144] = d144
			ps296.OverlayValues[145] = d145
			ps296.OverlayValues[146] = d146
			ps296.OverlayValues[153] = d153
			ps296.OverlayValues[154] = d154
			ps296.OverlayValues[160] = d160
			ps296.OverlayValues[161] = d161
			ps296.OverlayValues[162] = d162
			ps296.OverlayValues[163] = d163
			ps296.OverlayValues[164] = d164
			ps296.OverlayValues[165] = d165
			ps296.OverlayValues[166] = d166
			ps296.OverlayValues[168] = d168
			ps296.OverlayValues[170] = d170
			ps296.OverlayValues[171] = d171
			ps296.OverlayValues[174] = d174
			ps296.OverlayValues[177] = d177
			ps296.OverlayValues[178] = d178
			ps296.OverlayValues[179] = d179
			ps296.OverlayValues[181] = d181
			ps296.OverlayValues[182] = d182
			ps296.OverlayValues[183] = d183
			ps296.OverlayValues[184] = d184
			ps296.OverlayValues[185] = d185
			ps296.OverlayValues[186] = d186
			ps296.OverlayValues[188] = d188
			ps296.OverlayValues[189] = d189
			ps296.OverlayValues[190] = d190
			ps296.OverlayValues[191] = d191
			ps296.OverlayValues[192] = d192
			ps296.OverlayValues[193] = d193
			ps296.OverlayValues[194] = d194
			ps296.OverlayValues[195] = d195
			ps296.OverlayValues[196] = d196
			ps296.OverlayValues[199] = d199
			ps296.OverlayValues[200] = d200
			ps296.OverlayValues[201] = d201
			ps296.OverlayValues[204] = d204
			ps296.OverlayValues[205] = d205
			ps296.OverlayValues[206] = d206
			ps296.OverlayValues[207] = d207
			ps296.OverlayValues[208] = d208
			ps296.OverlayValues[209] = d209
			ps296.OverlayValues[210] = d210
			ps296.OverlayValues[211] = d211
			ps296.OverlayValues[212] = d212
			ps296.OverlayValues[214] = d214
			ps296.OverlayValues[215] = d215
			ps296.OverlayValues[216] = d216
			ps296.OverlayValues[217] = d217
			ps296.OverlayValues[218] = d218
			ps296.OverlayValues[219] = d219
			ps296.OverlayValues[220] = d220
			ps296.OverlayValues[221] = d221
			ps296.OverlayValues[222] = d222
			ps296.OverlayValues[223] = d223
			ps296.OverlayValues[225] = d225
			ps296.OverlayValues[226] = d226
			ps296.OverlayValues[227] = d227
			ps296.OverlayValues[228] = d228
			ps296.OverlayValues[229] = d229
			ps296.OverlayValues[230] = d230
			ps296.OverlayValues[231] = d231
			ps296.OverlayValues[232] = d232
			ps296.OverlayValues[233] = d233
			ps296.OverlayValues[234] = d234
			ps296.OverlayValues[235] = d235
			ps296.OverlayValues[236] = d236
			ps296.OverlayValues[237] = d237
			ps296.OverlayValues[238] = d238
			ps296.OverlayValues[239] = d239
			ps296.OverlayValues[240] = d240
			ps296.OverlayValues[241] = d241
			ps296.OverlayValues[242] = d242
			ps296.OverlayValues[243] = d243
			ps296.OverlayValues[244] = d244
			ps296.OverlayValues[245] = d245
			ps296.OverlayValues[246] = d246
			ps296.OverlayValues[247] = d247
			ps296.OverlayValues[248] = d248
			ps296.OverlayValues[249] = d249
			ps296.OverlayValues[250] = d250
			ps296.OverlayValues[251] = d251
			ps296.OverlayValues[252] = d252
			ps296.OverlayValues[253] = d253
			ps296.OverlayValues[254] = d254
			ps296.OverlayValues[255] = d255
			ps296.OverlayValues[256] = d256
			ps296.OverlayValues[257] = d257
			ps296.OverlayValues[258] = d258
			ps296.OverlayValues[259] = d259
			ps296.OverlayValues[260] = d260
			ps296.OverlayValues[261] = d261
			ps296.OverlayValues[262] = d262
			ps296.OverlayValues[263] = d263
			ps296.OverlayValues[264] = d264
			ps296.OverlayValues[265] = d265
			ps296.OverlayValues[266] = d266
			ps296.OverlayValues[267] = d267
			ps296.OverlayValues[268] = d268
			ps296.OverlayValues[269] = d269
			ps296.OverlayValues[270] = d270
			ps296.OverlayValues[271] = d271
			ps296.OverlayValues[278] = d278
			ps296.OverlayValues[279] = d279
			ps296.OverlayValues[285] = d285
			ps296.OverlayValues[286] = d286
			ps296.OverlayValues[287] = d287
			ps296.OverlayValues[288] = d288
			ps296.OverlayValues[289] = d289
			ps296.OverlayValues[290] = d290
			ps296.OverlayValues[292] = d292
			ps296.OverlayValues[294] = d294
			ps296.OverlayValues[295] = d295
			ps296.PhiValues = make([]scm.JITValueDesc, 1)
			d298 = d8
			ps296.PhiValues[0] = d298
			ps297 := scm.PhiState{General: true}
			ps297.OverlayValues = make([]scm.JITValueDesc, 299)
			ps297.OverlayValues[0] = d0
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
			ps297.OverlayValues[20] = d20
			ps297.OverlayValues[21] = d21
			ps297.OverlayValues[22] = d22
			ps297.OverlayValues[24] = d24
			ps297.OverlayValues[25] = d25
			ps297.OverlayValues[27] = d27
			ps297.OverlayValues[28] = d28
			ps297.OverlayValues[29] = d29
			ps297.OverlayValues[32] = d32
			ps297.OverlayValues[34] = d34
			ps297.OverlayValues[35] = d35
			ps297.OverlayValues[36] = d36
			ps297.OverlayValues[38] = d38
			ps297.OverlayValues[39] = d39
			ps297.OverlayValues[40] = d40
			ps297.OverlayValues[41] = d41
			ps297.OverlayValues[42] = d42
			ps297.OverlayValues[43] = d43
			ps297.OverlayValues[44] = d44
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
			ps297.OverlayValues[62] = d62
			ps297.OverlayValues[63] = d63
			ps297.OverlayValues[64] = d64
			ps297.OverlayValues[65] = d65
			ps297.OverlayValues[66] = d66
			ps297.OverlayValues[67] = d67
			ps297.OverlayValues[68] = d68
			ps297.OverlayValues[69] = d69
			ps297.OverlayValues[70] = d70
			ps297.OverlayValues[71] = d71
			ps297.OverlayValues[72] = d72
			ps297.OverlayValues[73] = d73
			ps297.OverlayValues[74] = d74
			ps297.OverlayValues[75] = d75
			ps297.OverlayValues[76] = d76
			ps297.OverlayValues[77] = d77
			ps297.OverlayValues[78] = d78
			ps297.OverlayValues[79] = d79
			ps297.OverlayValues[80] = d80
			ps297.OverlayValues[81] = d81
			ps297.OverlayValues[82] = d82
			ps297.OverlayValues[83] = d83
			ps297.OverlayValues[84] = d84
			ps297.OverlayValues[85] = d85
			ps297.OverlayValues[86] = d86
			ps297.OverlayValues[87] = d87
			ps297.OverlayValues[88] = d88
			ps297.OverlayValues[89] = d89
			ps297.OverlayValues[90] = d90
			ps297.OverlayValues[91] = d91
			ps297.OverlayValues[92] = d92
			ps297.OverlayValues[93] = d93
			ps297.OverlayValues[94] = d94
			ps297.OverlayValues[95] = d95
			ps297.OverlayValues[102] = d102
			ps297.OverlayValues[103] = d103
			ps297.OverlayValues[104] = d104
			ps297.OverlayValues[105] = d105
			ps297.OverlayValues[106] = d106
			ps297.OverlayValues[107] = d107
			ps297.OverlayValues[108] = d108
			ps297.OverlayValues[109] = d109
			ps297.OverlayValues[110] = d110
			ps297.OverlayValues[111] = d111
			ps297.OverlayValues[112] = d112
			ps297.OverlayValues[113] = d113
			ps297.OverlayValues[114] = d114
			ps297.OverlayValues[115] = d115
			ps297.OverlayValues[116] = d116
			ps297.OverlayValues[117] = d117
			ps297.OverlayValues[118] = d118
			ps297.OverlayValues[119] = d119
			ps297.OverlayValues[120] = d120
			ps297.OverlayValues[121] = d121
			ps297.OverlayValues[122] = d122
			ps297.OverlayValues[123] = d123
			ps297.OverlayValues[124] = d124
			ps297.OverlayValues[125] = d125
			ps297.OverlayValues[126] = d126
			ps297.OverlayValues[127] = d127
			ps297.OverlayValues[128] = d128
			ps297.OverlayValues[129] = d129
			ps297.OverlayValues[130] = d130
			ps297.OverlayValues[131] = d131
			ps297.OverlayValues[132] = d132
			ps297.OverlayValues[133] = d133
			ps297.OverlayValues[134] = d134
			ps297.OverlayValues[135] = d135
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
			ps297.OverlayValues[153] = d153
			ps297.OverlayValues[154] = d154
			ps297.OverlayValues[160] = d160
			ps297.OverlayValues[161] = d161
			ps297.OverlayValues[162] = d162
			ps297.OverlayValues[163] = d163
			ps297.OverlayValues[164] = d164
			ps297.OverlayValues[165] = d165
			ps297.OverlayValues[166] = d166
			ps297.OverlayValues[168] = d168
			ps297.OverlayValues[170] = d170
			ps297.OverlayValues[171] = d171
			ps297.OverlayValues[174] = d174
			ps297.OverlayValues[177] = d177
			ps297.OverlayValues[178] = d178
			ps297.OverlayValues[179] = d179
			ps297.OverlayValues[181] = d181
			ps297.OverlayValues[182] = d182
			ps297.OverlayValues[183] = d183
			ps297.OverlayValues[184] = d184
			ps297.OverlayValues[185] = d185
			ps297.OverlayValues[186] = d186
			ps297.OverlayValues[188] = d188
			ps297.OverlayValues[189] = d189
			ps297.OverlayValues[190] = d190
			ps297.OverlayValues[191] = d191
			ps297.OverlayValues[192] = d192
			ps297.OverlayValues[193] = d193
			ps297.OverlayValues[194] = d194
			ps297.OverlayValues[195] = d195
			ps297.OverlayValues[196] = d196
			ps297.OverlayValues[199] = d199
			ps297.OverlayValues[200] = d200
			ps297.OverlayValues[201] = d201
			ps297.OverlayValues[204] = d204
			ps297.OverlayValues[205] = d205
			ps297.OverlayValues[206] = d206
			ps297.OverlayValues[207] = d207
			ps297.OverlayValues[208] = d208
			ps297.OverlayValues[209] = d209
			ps297.OverlayValues[210] = d210
			ps297.OverlayValues[211] = d211
			ps297.OverlayValues[212] = d212
			ps297.OverlayValues[214] = d214
			ps297.OverlayValues[215] = d215
			ps297.OverlayValues[216] = d216
			ps297.OverlayValues[217] = d217
			ps297.OverlayValues[218] = d218
			ps297.OverlayValues[219] = d219
			ps297.OverlayValues[220] = d220
			ps297.OverlayValues[221] = d221
			ps297.OverlayValues[222] = d222
			ps297.OverlayValues[223] = d223
			ps297.OverlayValues[225] = d225
			ps297.OverlayValues[226] = d226
			ps297.OverlayValues[227] = d227
			ps297.OverlayValues[228] = d228
			ps297.OverlayValues[229] = d229
			ps297.OverlayValues[230] = d230
			ps297.OverlayValues[231] = d231
			ps297.OverlayValues[232] = d232
			ps297.OverlayValues[233] = d233
			ps297.OverlayValues[234] = d234
			ps297.OverlayValues[235] = d235
			ps297.OverlayValues[236] = d236
			ps297.OverlayValues[237] = d237
			ps297.OverlayValues[238] = d238
			ps297.OverlayValues[239] = d239
			ps297.OverlayValues[240] = d240
			ps297.OverlayValues[241] = d241
			ps297.OverlayValues[242] = d242
			ps297.OverlayValues[243] = d243
			ps297.OverlayValues[244] = d244
			ps297.OverlayValues[245] = d245
			ps297.OverlayValues[246] = d246
			ps297.OverlayValues[247] = d247
			ps297.OverlayValues[248] = d248
			ps297.OverlayValues[249] = d249
			ps297.OverlayValues[250] = d250
			ps297.OverlayValues[251] = d251
			ps297.OverlayValues[252] = d252
			ps297.OverlayValues[253] = d253
			ps297.OverlayValues[254] = d254
			ps297.OverlayValues[255] = d255
			ps297.OverlayValues[256] = d256
			ps297.OverlayValues[257] = d257
			ps297.OverlayValues[258] = d258
			ps297.OverlayValues[259] = d259
			ps297.OverlayValues[260] = d260
			ps297.OverlayValues[261] = d261
			ps297.OverlayValues[262] = d262
			ps297.OverlayValues[263] = d263
			ps297.OverlayValues[264] = d264
			ps297.OverlayValues[265] = d265
			ps297.OverlayValues[266] = d266
			ps297.OverlayValues[267] = d267
			ps297.OverlayValues[268] = d268
			ps297.OverlayValues[269] = d269
			ps297.OverlayValues[270] = d270
			ps297.OverlayValues[271] = d271
			ps297.OverlayValues[278] = d278
			ps297.OverlayValues[279] = d279
			ps297.OverlayValues[285] = d285
			ps297.OverlayValues[286] = d286
			ps297.OverlayValues[287] = d287
			ps297.OverlayValues[288] = d288
			ps297.OverlayValues[289] = d289
			ps297.OverlayValues[290] = d290
			ps297.OverlayValues[292] = d292
			ps297.OverlayValues[294] = d294
			ps297.OverlayValues[295] = d295
			ps297.OverlayValues[298] = d298
			snap299 := d8
			snap300 := d9
			alloc301 := ctx.SnapshotAllocState()
			if !bbs[6].Rendered {
				bbs[6].RenderPS(ps296)
			}
			ctx.RestoreAllocState(alloc301)
			d8 = snap299
			d9 = snap300
			if !bbs[19].Rendered {
				return bbs[19].RenderPS(ps297)
			}
			return result
			ctx.FreeDesc(&d287)
			return result
			}
			bbs[16].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[16].VisitCount >= 2 {
					ps.General = true
					return bbs[16].RenderPS(ps)
				}
			}
			bbs[16].VisitCount++
			if ps.General {
				if bbs[16].Rendered {
					ctx.W.EmitJmp(lbl17)
					return result
				}
				bbs[16].Rendered = true
				bbs[16].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_16 = bbs[16].Address
				ctx.W.MarkLabel(lbl17)
				ctx.W.ResolveFixups()
			}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
				d154 = ps.OverlayValues[154]
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
			if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
				d168 = ps.OverlayValues[168]
			}
			if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != scm.LocNone {
				d170 = ps.OverlayValues[170]
			}
			if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != scm.LocNone {
				d171 = ps.OverlayValues[171]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
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
			if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
				d199 = ps.OverlayValues[199]
			}
			if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
				d200 = ps.OverlayValues[200]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
				d201 = ps.OverlayValues[201]
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
			if len(ps.OverlayValues) > 254 && ps.OverlayValues[254].Loc != scm.LocNone {
				d254 = ps.OverlayValues[254]
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
			if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != scm.LocNone {
				d278 = ps.OverlayValues[278]
			}
			if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != scm.LocNone {
				d279 = ps.OverlayValues[279]
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
			if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
				d294 = ps.OverlayValues[294]
			}
			if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
				d295 = ps.OverlayValues[295]
			}
			if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != scm.LocNone {
				d298 = ps.OverlayValues[298]
			}
			ctx.ReclaimUntrackedRegs()
			d302 = d5
			if d302.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d302)
			d303 = d302
			if d303.Loc == scm.LocImm {
				d303 = scm.JITValueDesc{Loc: scm.LocImm, Type: d303.Type, Imm: scm.NewInt(int64(uint64(d303.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d303.Reg, 32)
				ctx.W.EmitShrRegImm8(d303.Reg, 32)
			}
			ctx.EmitStoreToStack(d303, 64)
			d304 = d7
			if d304.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d304)
			d305 = d304
			if d305.Loc == scm.LocImm {
				d305 = scm.JITValueDesc{Loc: scm.LocImm, Type: d305.Type, Imm: scm.NewInt(int64(uint64(d305.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d305.Reg, 32)
				ctx.W.EmitShrRegImm8(d305.Reg, 32)
			}
			ctx.EmitStoreToStack(d305, 72)
			ps306 := scm.PhiState{General: ps.General}
			ps306.OverlayValues = make([]scm.JITValueDesc, 306)
			ps306.OverlayValues[0] = d0
			ps306.OverlayValues[1] = d1
			ps306.OverlayValues[2] = d2
			ps306.OverlayValues[3] = d3
			ps306.OverlayValues[4] = d4
			ps306.OverlayValues[5] = d5
			ps306.OverlayValues[6] = d6
			ps306.OverlayValues[7] = d7
			ps306.OverlayValues[8] = d8
			ps306.OverlayValues[9] = d9
			ps306.OverlayValues[10] = d10
			ps306.OverlayValues[11] = d11
			ps306.OverlayValues[12] = d12
			ps306.OverlayValues[13] = d13
			ps306.OverlayValues[14] = d14
			ps306.OverlayValues[20] = d20
			ps306.OverlayValues[21] = d21
			ps306.OverlayValues[22] = d22
			ps306.OverlayValues[24] = d24
			ps306.OverlayValues[25] = d25
			ps306.OverlayValues[27] = d27
			ps306.OverlayValues[28] = d28
			ps306.OverlayValues[29] = d29
			ps306.OverlayValues[32] = d32
			ps306.OverlayValues[34] = d34
			ps306.OverlayValues[35] = d35
			ps306.OverlayValues[36] = d36
			ps306.OverlayValues[38] = d38
			ps306.OverlayValues[39] = d39
			ps306.OverlayValues[40] = d40
			ps306.OverlayValues[41] = d41
			ps306.OverlayValues[42] = d42
			ps306.OverlayValues[43] = d43
			ps306.OverlayValues[44] = d44
			ps306.OverlayValues[46] = d46
			ps306.OverlayValues[47] = d47
			ps306.OverlayValues[48] = d48
			ps306.OverlayValues[49] = d49
			ps306.OverlayValues[50] = d50
			ps306.OverlayValues[51] = d51
			ps306.OverlayValues[52] = d52
			ps306.OverlayValues[53] = d53
			ps306.OverlayValues[54] = d54
			ps306.OverlayValues[55] = d55
			ps306.OverlayValues[56] = d56
			ps306.OverlayValues[57] = d57
			ps306.OverlayValues[58] = d58
			ps306.OverlayValues[59] = d59
			ps306.OverlayValues[60] = d60
			ps306.OverlayValues[61] = d61
			ps306.OverlayValues[62] = d62
			ps306.OverlayValues[63] = d63
			ps306.OverlayValues[64] = d64
			ps306.OverlayValues[65] = d65
			ps306.OverlayValues[66] = d66
			ps306.OverlayValues[67] = d67
			ps306.OverlayValues[68] = d68
			ps306.OverlayValues[69] = d69
			ps306.OverlayValues[70] = d70
			ps306.OverlayValues[71] = d71
			ps306.OverlayValues[72] = d72
			ps306.OverlayValues[73] = d73
			ps306.OverlayValues[74] = d74
			ps306.OverlayValues[75] = d75
			ps306.OverlayValues[76] = d76
			ps306.OverlayValues[77] = d77
			ps306.OverlayValues[78] = d78
			ps306.OverlayValues[79] = d79
			ps306.OverlayValues[80] = d80
			ps306.OverlayValues[81] = d81
			ps306.OverlayValues[82] = d82
			ps306.OverlayValues[83] = d83
			ps306.OverlayValues[84] = d84
			ps306.OverlayValues[85] = d85
			ps306.OverlayValues[86] = d86
			ps306.OverlayValues[87] = d87
			ps306.OverlayValues[88] = d88
			ps306.OverlayValues[89] = d89
			ps306.OverlayValues[90] = d90
			ps306.OverlayValues[91] = d91
			ps306.OverlayValues[92] = d92
			ps306.OverlayValues[93] = d93
			ps306.OverlayValues[94] = d94
			ps306.OverlayValues[95] = d95
			ps306.OverlayValues[102] = d102
			ps306.OverlayValues[103] = d103
			ps306.OverlayValues[104] = d104
			ps306.OverlayValues[105] = d105
			ps306.OverlayValues[106] = d106
			ps306.OverlayValues[107] = d107
			ps306.OverlayValues[108] = d108
			ps306.OverlayValues[109] = d109
			ps306.OverlayValues[110] = d110
			ps306.OverlayValues[111] = d111
			ps306.OverlayValues[112] = d112
			ps306.OverlayValues[113] = d113
			ps306.OverlayValues[114] = d114
			ps306.OverlayValues[115] = d115
			ps306.OverlayValues[116] = d116
			ps306.OverlayValues[117] = d117
			ps306.OverlayValues[118] = d118
			ps306.OverlayValues[119] = d119
			ps306.OverlayValues[120] = d120
			ps306.OverlayValues[121] = d121
			ps306.OverlayValues[122] = d122
			ps306.OverlayValues[123] = d123
			ps306.OverlayValues[124] = d124
			ps306.OverlayValues[125] = d125
			ps306.OverlayValues[126] = d126
			ps306.OverlayValues[127] = d127
			ps306.OverlayValues[128] = d128
			ps306.OverlayValues[129] = d129
			ps306.OverlayValues[130] = d130
			ps306.OverlayValues[131] = d131
			ps306.OverlayValues[132] = d132
			ps306.OverlayValues[133] = d133
			ps306.OverlayValues[134] = d134
			ps306.OverlayValues[135] = d135
			ps306.OverlayValues[136] = d136
			ps306.OverlayValues[137] = d137
			ps306.OverlayValues[138] = d138
			ps306.OverlayValues[139] = d139
			ps306.OverlayValues[140] = d140
			ps306.OverlayValues[141] = d141
			ps306.OverlayValues[142] = d142
			ps306.OverlayValues[143] = d143
			ps306.OverlayValues[144] = d144
			ps306.OverlayValues[145] = d145
			ps306.OverlayValues[146] = d146
			ps306.OverlayValues[153] = d153
			ps306.OverlayValues[154] = d154
			ps306.OverlayValues[160] = d160
			ps306.OverlayValues[161] = d161
			ps306.OverlayValues[162] = d162
			ps306.OverlayValues[163] = d163
			ps306.OverlayValues[164] = d164
			ps306.OverlayValues[165] = d165
			ps306.OverlayValues[166] = d166
			ps306.OverlayValues[168] = d168
			ps306.OverlayValues[170] = d170
			ps306.OverlayValues[171] = d171
			ps306.OverlayValues[174] = d174
			ps306.OverlayValues[177] = d177
			ps306.OverlayValues[178] = d178
			ps306.OverlayValues[179] = d179
			ps306.OverlayValues[181] = d181
			ps306.OverlayValues[182] = d182
			ps306.OverlayValues[183] = d183
			ps306.OverlayValues[184] = d184
			ps306.OverlayValues[185] = d185
			ps306.OverlayValues[186] = d186
			ps306.OverlayValues[188] = d188
			ps306.OverlayValues[189] = d189
			ps306.OverlayValues[190] = d190
			ps306.OverlayValues[191] = d191
			ps306.OverlayValues[192] = d192
			ps306.OverlayValues[193] = d193
			ps306.OverlayValues[194] = d194
			ps306.OverlayValues[195] = d195
			ps306.OverlayValues[196] = d196
			ps306.OverlayValues[199] = d199
			ps306.OverlayValues[200] = d200
			ps306.OverlayValues[201] = d201
			ps306.OverlayValues[204] = d204
			ps306.OverlayValues[205] = d205
			ps306.OverlayValues[206] = d206
			ps306.OverlayValues[207] = d207
			ps306.OverlayValues[208] = d208
			ps306.OverlayValues[209] = d209
			ps306.OverlayValues[210] = d210
			ps306.OverlayValues[211] = d211
			ps306.OverlayValues[212] = d212
			ps306.OverlayValues[214] = d214
			ps306.OverlayValues[215] = d215
			ps306.OverlayValues[216] = d216
			ps306.OverlayValues[217] = d217
			ps306.OverlayValues[218] = d218
			ps306.OverlayValues[219] = d219
			ps306.OverlayValues[220] = d220
			ps306.OverlayValues[221] = d221
			ps306.OverlayValues[222] = d222
			ps306.OverlayValues[223] = d223
			ps306.OverlayValues[225] = d225
			ps306.OverlayValues[226] = d226
			ps306.OverlayValues[227] = d227
			ps306.OverlayValues[228] = d228
			ps306.OverlayValues[229] = d229
			ps306.OverlayValues[230] = d230
			ps306.OverlayValues[231] = d231
			ps306.OverlayValues[232] = d232
			ps306.OverlayValues[233] = d233
			ps306.OverlayValues[234] = d234
			ps306.OverlayValues[235] = d235
			ps306.OverlayValues[236] = d236
			ps306.OverlayValues[237] = d237
			ps306.OverlayValues[238] = d238
			ps306.OverlayValues[239] = d239
			ps306.OverlayValues[240] = d240
			ps306.OverlayValues[241] = d241
			ps306.OverlayValues[242] = d242
			ps306.OverlayValues[243] = d243
			ps306.OverlayValues[244] = d244
			ps306.OverlayValues[245] = d245
			ps306.OverlayValues[246] = d246
			ps306.OverlayValues[247] = d247
			ps306.OverlayValues[248] = d248
			ps306.OverlayValues[249] = d249
			ps306.OverlayValues[250] = d250
			ps306.OverlayValues[251] = d251
			ps306.OverlayValues[252] = d252
			ps306.OverlayValues[253] = d253
			ps306.OverlayValues[254] = d254
			ps306.OverlayValues[255] = d255
			ps306.OverlayValues[256] = d256
			ps306.OverlayValues[257] = d257
			ps306.OverlayValues[258] = d258
			ps306.OverlayValues[259] = d259
			ps306.OverlayValues[260] = d260
			ps306.OverlayValues[261] = d261
			ps306.OverlayValues[262] = d262
			ps306.OverlayValues[263] = d263
			ps306.OverlayValues[264] = d264
			ps306.OverlayValues[265] = d265
			ps306.OverlayValues[266] = d266
			ps306.OverlayValues[267] = d267
			ps306.OverlayValues[268] = d268
			ps306.OverlayValues[269] = d269
			ps306.OverlayValues[270] = d270
			ps306.OverlayValues[271] = d271
			ps306.OverlayValues[278] = d278
			ps306.OverlayValues[279] = d279
			ps306.OverlayValues[285] = d285
			ps306.OverlayValues[286] = d286
			ps306.OverlayValues[287] = d287
			ps306.OverlayValues[288] = d288
			ps306.OverlayValues[289] = d289
			ps306.OverlayValues[290] = d290
			ps306.OverlayValues[292] = d292
			ps306.OverlayValues[294] = d294
			ps306.OverlayValues[295] = d295
			ps306.OverlayValues[298] = d298
			ps306.OverlayValues[302] = d302
			ps306.OverlayValues[303] = d303
			ps306.OverlayValues[304] = d304
			ps306.OverlayValues[305] = d305
			ps306.PhiValues = make([]scm.JITValueDesc, 2)
			d307 = d5
			ps306.PhiValues[0] = d307
			d308 = d7
			ps306.PhiValues[1] = d308
			if ps306.General && bbs[15].Rendered {
				ctx.W.EmitJmp(lbl16)
				return result
			}
			return bbs[15].RenderPS(ps306)
			return result
			}
			bbs[17].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[17].VisitCount >= 2 {
					ps.General = true
					return bbs[17].RenderPS(ps)
				}
			}
			bbs[17].VisitCount++
			if ps.General {
				if bbs[17].Rendered {
					ctx.W.EmitJmp(lbl18)
					return result
				}
				bbs[17].Rendered = true
				bbs[17].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_17 = bbs[17].Address
				ctx.W.MarkLabel(lbl18)
				ctx.W.ResolveFixups()
			}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
				d154 = ps.OverlayValues[154]
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
			if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
				d168 = ps.OverlayValues[168]
			}
			if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != scm.LocNone {
				d170 = ps.OverlayValues[170]
			}
			if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != scm.LocNone {
				d171 = ps.OverlayValues[171]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
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
			if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
				d199 = ps.OverlayValues[199]
			}
			if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
				d200 = ps.OverlayValues[200]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
				d201 = ps.OverlayValues[201]
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
			if len(ps.OverlayValues) > 254 && ps.OverlayValues[254].Loc != scm.LocNone {
				d254 = ps.OverlayValues[254]
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
			if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != scm.LocNone {
				d278 = ps.OverlayValues[278]
			}
			if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != scm.LocNone {
				d279 = ps.OverlayValues[279]
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
			if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
				d294 = ps.OverlayValues[294]
			}
			if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
				d295 = ps.OverlayValues[295]
			}
			if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != scm.LocNone {
				d298 = ps.OverlayValues[298]
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
			if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != scm.LocNone {
				d305 = ps.OverlayValues[305]
			}
			if len(ps.OverlayValues) > 307 && ps.OverlayValues[307].Loc != scm.LocNone {
				d307 = ps.OverlayValues[307]
			}
			if len(ps.OverlayValues) > 308 && ps.OverlayValues[308].Loc != scm.LocNone {
				d308 = ps.OverlayValues[308]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 32)
			ps309 := scm.PhiState{General: ps.General}
			ps309.OverlayValues = make([]scm.JITValueDesc, 309)
			ps309.OverlayValues[0] = d0
			ps309.OverlayValues[1] = d1
			ps309.OverlayValues[2] = d2
			ps309.OverlayValues[3] = d3
			ps309.OverlayValues[4] = d4
			ps309.OverlayValues[5] = d5
			ps309.OverlayValues[6] = d6
			ps309.OverlayValues[7] = d7
			ps309.OverlayValues[8] = d8
			ps309.OverlayValues[9] = d9
			ps309.OverlayValues[10] = d10
			ps309.OverlayValues[11] = d11
			ps309.OverlayValues[12] = d12
			ps309.OverlayValues[13] = d13
			ps309.OverlayValues[14] = d14
			ps309.OverlayValues[20] = d20
			ps309.OverlayValues[21] = d21
			ps309.OverlayValues[22] = d22
			ps309.OverlayValues[24] = d24
			ps309.OverlayValues[25] = d25
			ps309.OverlayValues[27] = d27
			ps309.OverlayValues[28] = d28
			ps309.OverlayValues[29] = d29
			ps309.OverlayValues[32] = d32
			ps309.OverlayValues[34] = d34
			ps309.OverlayValues[35] = d35
			ps309.OverlayValues[36] = d36
			ps309.OverlayValues[38] = d38
			ps309.OverlayValues[39] = d39
			ps309.OverlayValues[40] = d40
			ps309.OverlayValues[41] = d41
			ps309.OverlayValues[42] = d42
			ps309.OverlayValues[43] = d43
			ps309.OverlayValues[44] = d44
			ps309.OverlayValues[46] = d46
			ps309.OverlayValues[47] = d47
			ps309.OverlayValues[48] = d48
			ps309.OverlayValues[49] = d49
			ps309.OverlayValues[50] = d50
			ps309.OverlayValues[51] = d51
			ps309.OverlayValues[52] = d52
			ps309.OverlayValues[53] = d53
			ps309.OverlayValues[54] = d54
			ps309.OverlayValues[55] = d55
			ps309.OverlayValues[56] = d56
			ps309.OverlayValues[57] = d57
			ps309.OverlayValues[58] = d58
			ps309.OverlayValues[59] = d59
			ps309.OverlayValues[60] = d60
			ps309.OverlayValues[61] = d61
			ps309.OverlayValues[62] = d62
			ps309.OverlayValues[63] = d63
			ps309.OverlayValues[64] = d64
			ps309.OverlayValues[65] = d65
			ps309.OverlayValues[66] = d66
			ps309.OverlayValues[67] = d67
			ps309.OverlayValues[68] = d68
			ps309.OverlayValues[69] = d69
			ps309.OverlayValues[70] = d70
			ps309.OverlayValues[71] = d71
			ps309.OverlayValues[72] = d72
			ps309.OverlayValues[73] = d73
			ps309.OverlayValues[74] = d74
			ps309.OverlayValues[75] = d75
			ps309.OverlayValues[76] = d76
			ps309.OverlayValues[77] = d77
			ps309.OverlayValues[78] = d78
			ps309.OverlayValues[79] = d79
			ps309.OverlayValues[80] = d80
			ps309.OverlayValues[81] = d81
			ps309.OverlayValues[82] = d82
			ps309.OverlayValues[83] = d83
			ps309.OverlayValues[84] = d84
			ps309.OverlayValues[85] = d85
			ps309.OverlayValues[86] = d86
			ps309.OverlayValues[87] = d87
			ps309.OverlayValues[88] = d88
			ps309.OverlayValues[89] = d89
			ps309.OverlayValues[90] = d90
			ps309.OverlayValues[91] = d91
			ps309.OverlayValues[92] = d92
			ps309.OverlayValues[93] = d93
			ps309.OverlayValues[94] = d94
			ps309.OverlayValues[95] = d95
			ps309.OverlayValues[102] = d102
			ps309.OverlayValues[103] = d103
			ps309.OverlayValues[104] = d104
			ps309.OverlayValues[105] = d105
			ps309.OverlayValues[106] = d106
			ps309.OverlayValues[107] = d107
			ps309.OverlayValues[108] = d108
			ps309.OverlayValues[109] = d109
			ps309.OverlayValues[110] = d110
			ps309.OverlayValues[111] = d111
			ps309.OverlayValues[112] = d112
			ps309.OverlayValues[113] = d113
			ps309.OverlayValues[114] = d114
			ps309.OverlayValues[115] = d115
			ps309.OverlayValues[116] = d116
			ps309.OverlayValues[117] = d117
			ps309.OverlayValues[118] = d118
			ps309.OverlayValues[119] = d119
			ps309.OverlayValues[120] = d120
			ps309.OverlayValues[121] = d121
			ps309.OverlayValues[122] = d122
			ps309.OverlayValues[123] = d123
			ps309.OverlayValues[124] = d124
			ps309.OverlayValues[125] = d125
			ps309.OverlayValues[126] = d126
			ps309.OverlayValues[127] = d127
			ps309.OverlayValues[128] = d128
			ps309.OverlayValues[129] = d129
			ps309.OverlayValues[130] = d130
			ps309.OverlayValues[131] = d131
			ps309.OverlayValues[132] = d132
			ps309.OverlayValues[133] = d133
			ps309.OverlayValues[134] = d134
			ps309.OverlayValues[135] = d135
			ps309.OverlayValues[136] = d136
			ps309.OverlayValues[137] = d137
			ps309.OverlayValues[138] = d138
			ps309.OverlayValues[139] = d139
			ps309.OverlayValues[140] = d140
			ps309.OverlayValues[141] = d141
			ps309.OverlayValues[142] = d142
			ps309.OverlayValues[143] = d143
			ps309.OverlayValues[144] = d144
			ps309.OverlayValues[145] = d145
			ps309.OverlayValues[146] = d146
			ps309.OverlayValues[153] = d153
			ps309.OverlayValues[154] = d154
			ps309.OverlayValues[160] = d160
			ps309.OverlayValues[161] = d161
			ps309.OverlayValues[162] = d162
			ps309.OverlayValues[163] = d163
			ps309.OverlayValues[164] = d164
			ps309.OverlayValues[165] = d165
			ps309.OverlayValues[166] = d166
			ps309.OverlayValues[168] = d168
			ps309.OverlayValues[170] = d170
			ps309.OverlayValues[171] = d171
			ps309.OverlayValues[174] = d174
			ps309.OverlayValues[177] = d177
			ps309.OverlayValues[178] = d178
			ps309.OverlayValues[179] = d179
			ps309.OverlayValues[181] = d181
			ps309.OverlayValues[182] = d182
			ps309.OverlayValues[183] = d183
			ps309.OverlayValues[184] = d184
			ps309.OverlayValues[185] = d185
			ps309.OverlayValues[186] = d186
			ps309.OverlayValues[188] = d188
			ps309.OverlayValues[189] = d189
			ps309.OverlayValues[190] = d190
			ps309.OverlayValues[191] = d191
			ps309.OverlayValues[192] = d192
			ps309.OverlayValues[193] = d193
			ps309.OverlayValues[194] = d194
			ps309.OverlayValues[195] = d195
			ps309.OverlayValues[196] = d196
			ps309.OverlayValues[199] = d199
			ps309.OverlayValues[200] = d200
			ps309.OverlayValues[201] = d201
			ps309.OverlayValues[204] = d204
			ps309.OverlayValues[205] = d205
			ps309.OverlayValues[206] = d206
			ps309.OverlayValues[207] = d207
			ps309.OverlayValues[208] = d208
			ps309.OverlayValues[209] = d209
			ps309.OverlayValues[210] = d210
			ps309.OverlayValues[211] = d211
			ps309.OverlayValues[212] = d212
			ps309.OverlayValues[214] = d214
			ps309.OverlayValues[215] = d215
			ps309.OverlayValues[216] = d216
			ps309.OverlayValues[217] = d217
			ps309.OverlayValues[218] = d218
			ps309.OverlayValues[219] = d219
			ps309.OverlayValues[220] = d220
			ps309.OverlayValues[221] = d221
			ps309.OverlayValues[222] = d222
			ps309.OverlayValues[223] = d223
			ps309.OverlayValues[225] = d225
			ps309.OverlayValues[226] = d226
			ps309.OverlayValues[227] = d227
			ps309.OverlayValues[228] = d228
			ps309.OverlayValues[229] = d229
			ps309.OverlayValues[230] = d230
			ps309.OverlayValues[231] = d231
			ps309.OverlayValues[232] = d232
			ps309.OverlayValues[233] = d233
			ps309.OverlayValues[234] = d234
			ps309.OverlayValues[235] = d235
			ps309.OverlayValues[236] = d236
			ps309.OverlayValues[237] = d237
			ps309.OverlayValues[238] = d238
			ps309.OverlayValues[239] = d239
			ps309.OverlayValues[240] = d240
			ps309.OverlayValues[241] = d241
			ps309.OverlayValues[242] = d242
			ps309.OverlayValues[243] = d243
			ps309.OverlayValues[244] = d244
			ps309.OverlayValues[245] = d245
			ps309.OverlayValues[246] = d246
			ps309.OverlayValues[247] = d247
			ps309.OverlayValues[248] = d248
			ps309.OverlayValues[249] = d249
			ps309.OverlayValues[250] = d250
			ps309.OverlayValues[251] = d251
			ps309.OverlayValues[252] = d252
			ps309.OverlayValues[253] = d253
			ps309.OverlayValues[254] = d254
			ps309.OverlayValues[255] = d255
			ps309.OverlayValues[256] = d256
			ps309.OverlayValues[257] = d257
			ps309.OverlayValues[258] = d258
			ps309.OverlayValues[259] = d259
			ps309.OverlayValues[260] = d260
			ps309.OverlayValues[261] = d261
			ps309.OverlayValues[262] = d262
			ps309.OverlayValues[263] = d263
			ps309.OverlayValues[264] = d264
			ps309.OverlayValues[265] = d265
			ps309.OverlayValues[266] = d266
			ps309.OverlayValues[267] = d267
			ps309.OverlayValues[268] = d268
			ps309.OverlayValues[269] = d269
			ps309.OverlayValues[270] = d270
			ps309.OverlayValues[271] = d271
			ps309.OverlayValues[278] = d278
			ps309.OverlayValues[279] = d279
			ps309.OverlayValues[285] = d285
			ps309.OverlayValues[286] = d286
			ps309.OverlayValues[287] = d287
			ps309.OverlayValues[288] = d288
			ps309.OverlayValues[289] = d289
			ps309.OverlayValues[290] = d290
			ps309.OverlayValues[292] = d292
			ps309.OverlayValues[294] = d294
			ps309.OverlayValues[295] = d295
			ps309.OverlayValues[298] = d298
			ps309.OverlayValues[302] = d302
			ps309.OverlayValues[303] = d303
			ps309.OverlayValues[304] = d304
			ps309.OverlayValues[305] = d305
			ps309.OverlayValues[307] = d307
			ps309.OverlayValues[308] = d308
			ps309.PhiValues = make([]scm.JITValueDesc, 1)
			d310 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
			ps309.PhiValues[0] = d310
			if ps309.General && bbs[6].Rendered {
				ctx.W.EmitJmp(lbl7)
				return result
			}
			return bbs[6].RenderPS(ps309)
			return result
			}
			bbs[18].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[18].VisitCount >= 2 {
					ps.General = true
					return bbs[18].RenderPS(ps)
				}
			}
			bbs[18].VisitCount++
			if ps.General {
				if bbs[18].Rendered {
					ctx.W.EmitJmp(lbl19)
					return result
				}
				bbs[18].Rendered = true
				bbs[18].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_18 = bbs[18].Address
				ctx.W.MarkLabel(lbl19)
				ctx.W.ResolveFixups()
			}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
				d154 = ps.OverlayValues[154]
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
			if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
				d168 = ps.OverlayValues[168]
			}
			if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != scm.LocNone {
				d170 = ps.OverlayValues[170]
			}
			if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != scm.LocNone {
				d171 = ps.OverlayValues[171]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
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
			if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
				d199 = ps.OverlayValues[199]
			}
			if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
				d200 = ps.OverlayValues[200]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
				d201 = ps.OverlayValues[201]
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
			if len(ps.OverlayValues) > 254 && ps.OverlayValues[254].Loc != scm.LocNone {
				d254 = ps.OverlayValues[254]
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
			if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != scm.LocNone {
				d278 = ps.OverlayValues[278]
			}
			if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != scm.LocNone {
				d279 = ps.OverlayValues[279]
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
			if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
				d294 = ps.OverlayValues[294]
			}
			if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
				d295 = ps.OverlayValues[295]
			}
			if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != scm.LocNone {
				d298 = ps.OverlayValues[298]
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
			if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != scm.LocNone {
				d305 = ps.OverlayValues[305]
			}
			if len(ps.OverlayValues) > 307 && ps.OverlayValues[307].Loc != scm.LocNone {
				d307 = ps.OverlayValues[307]
			}
			if len(ps.OverlayValues) > 308 && ps.OverlayValues[308].Loc != scm.LocNone {
				d308 = ps.OverlayValues[308]
			}
			if len(ps.OverlayValues) > 310 && ps.OverlayValues[310].Loc != scm.LocNone {
				d310 = ps.OverlayValues[310]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d5)
			var d311 scm.JITValueDesc
			if d5.Loc == scm.LocImm {
				d311 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d5.Reg)
				ctx.W.EmitMovRegReg(scratch, d5.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d311 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d311)
			}
			if d311.Loc == scm.LocImm {
				d311 = scm.JITValueDesc{Loc: scm.LocImm, Type: d311.Type, Imm: scm.NewInt(int64(uint64(d311.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d311.Reg, 32)
				ctx.W.EmitShrRegImm8(d311.Reg, 32)
			}
			if d311.Loc == scm.LocReg && d5.Loc == scm.LocReg && d311.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = scm.LocNone
			}
			d312 = d6
			if d312.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d312)
			d313 = d312
			if d313.Loc == scm.LocImm {
				d313 = scm.JITValueDesc{Loc: scm.LocImm, Type: d313.Type, Imm: scm.NewInt(int64(uint64(d313.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d313.Reg, 32)
				ctx.W.EmitShrRegImm8(d313.Reg, 32)
			}
			ctx.EmitStoreToStack(d313, 64)
			d314 = d311
			if d314.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d314)
			d315 = d314
			if d315.Loc == scm.LocImm {
				d315 = scm.JITValueDesc{Loc: scm.LocImm, Type: d315.Type, Imm: scm.NewInt(int64(uint64(d315.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d315.Reg, 32)
				ctx.W.EmitShrRegImm8(d315.Reg, 32)
			}
			ctx.EmitStoreToStack(d315, 72)
			ps316 := scm.PhiState{General: ps.General}
			ps316.OverlayValues = make([]scm.JITValueDesc, 316)
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
			ps316.OverlayValues[20] = d20
			ps316.OverlayValues[21] = d21
			ps316.OverlayValues[22] = d22
			ps316.OverlayValues[24] = d24
			ps316.OverlayValues[25] = d25
			ps316.OverlayValues[27] = d27
			ps316.OverlayValues[28] = d28
			ps316.OverlayValues[29] = d29
			ps316.OverlayValues[32] = d32
			ps316.OverlayValues[34] = d34
			ps316.OverlayValues[35] = d35
			ps316.OverlayValues[36] = d36
			ps316.OverlayValues[38] = d38
			ps316.OverlayValues[39] = d39
			ps316.OverlayValues[40] = d40
			ps316.OverlayValues[41] = d41
			ps316.OverlayValues[42] = d42
			ps316.OverlayValues[43] = d43
			ps316.OverlayValues[44] = d44
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
			ps316.OverlayValues[153] = d153
			ps316.OverlayValues[154] = d154
			ps316.OverlayValues[160] = d160
			ps316.OverlayValues[161] = d161
			ps316.OverlayValues[162] = d162
			ps316.OverlayValues[163] = d163
			ps316.OverlayValues[164] = d164
			ps316.OverlayValues[165] = d165
			ps316.OverlayValues[166] = d166
			ps316.OverlayValues[168] = d168
			ps316.OverlayValues[170] = d170
			ps316.OverlayValues[171] = d171
			ps316.OverlayValues[174] = d174
			ps316.OverlayValues[177] = d177
			ps316.OverlayValues[178] = d178
			ps316.OverlayValues[179] = d179
			ps316.OverlayValues[181] = d181
			ps316.OverlayValues[182] = d182
			ps316.OverlayValues[183] = d183
			ps316.OverlayValues[184] = d184
			ps316.OverlayValues[185] = d185
			ps316.OverlayValues[186] = d186
			ps316.OverlayValues[188] = d188
			ps316.OverlayValues[189] = d189
			ps316.OverlayValues[190] = d190
			ps316.OverlayValues[191] = d191
			ps316.OverlayValues[192] = d192
			ps316.OverlayValues[193] = d193
			ps316.OverlayValues[194] = d194
			ps316.OverlayValues[195] = d195
			ps316.OverlayValues[196] = d196
			ps316.OverlayValues[199] = d199
			ps316.OverlayValues[200] = d200
			ps316.OverlayValues[201] = d201
			ps316.OverlayValues[204] = d204
			ps316.OverlayValues[205] = d205
			ps316.OverlayValues[206] = d206
			ps316.OverlayValues[207] = d207
			ps316.OverlayValues[208] = d208
			ps316.OverlayValues[209] = d209
			ps316.OverlayValues[210] = d210
			ps316.OverlayValues[211] = d211
			ps316.OverlayValues[212] = d212
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
			ps316.OverlayValues[241] = d241
			ps316.OverlayValues[242] = d242
			ps316.OverlayValues[243] = d243
			ps316.OverlayValues[244] = d244
			ps316.OverlayValues[245] = d245
			ps316.OverlayValues[246] = d246
			ps316.OverlayValues[247] = d247
			ps316.OverlayValues[248] = d248
			ps316.OverlayValues[249] = d249
			ps316.OverlayValues[250] = d250
			ps316.OverlayValues[251] = d251
			ps316.OverlayValues[252] = d252
			ps316.OverlayValues[253] = d253
			ps316.OverlayValues[254] = d254
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
			ps316.OverlayValues[278] = d278
			ps316.OverlayValues[279] = d279
			ps316.OverlayValues[285] = d285
			ps316.OverlayValues[286] = d286
			ps316.OverlayValues[287] = d287
			ps316.OverlayValues[288] = d288
			ps316.OverlayValues[289] = d289
			ps316.OverlayValues[290] = d290
			ps316.OverlayValues[292] = d292
			ps316.OverlayValues[294] = d294
			ps316.OverlayValues[295] = d295
			ps316.OverlayValues[298] = d298
			ps316.OverlayValues[302] = d302
			ps316.OverlayValues[303] = d303
			ps316.OverlayValues[304] = d304
			ps316.OverlayValues[305] = d305
			ps316.OverlayValues[307] = d307
			ps316.OverlayValues[308] = d308
			ps316.OverlayValues[310] = d310
			ps316.OverlayValues[311] = d311
			ps316.OverlayValues[312] = d312
			ps316.OverlayValues[313] = d313
			ps316.OverlayValues[314] = d314
			ps316.OverlayValues[315] = d315
			ps316.PhiValues = make([]scm.JITValueDesc, 2)
			d317 = d6
			ps316.PhiValues[0] = d317
			d318 = d311
			ps316.PhiValues[1] = d318
			if ps316.General && bbs[15].Rendered {
				ctx.W.EmitJmp(lbl16)
				return result
			}
			return bbs[15].RenderPS(ps316)
			return result
			}
			bbs[19].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[19].VisitCount >= 2 {
					ps.General = true
					return bbs[19].RenderPS(ps)
				}
			}
			bbs[19].VisitCount++
			if ps.General {
				if bbs[19].Rendered {
					ctx.W.EmitJmp(lbl20)
					return result
				}
				bbs[19].Rendered = true
				bbs[19].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_19 = bbs[19].Address
				ctx.W.MarkLabel(lbl20)
				ctx.W.ResolveFixups()
			}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
				d154 = ps.OverlayValues[154]
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
			if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
				d168 = ps.OverlayValues[168]
			}
			if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != scm.LocNone {
				d170 = ps.OverlayValues[170]
			}
			if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != scm.LocNone {
				d171 = ps.OverlayValues[171]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
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
			if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
				d199 = ps.OverlayValues[199]
			}
			if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
				d200 = ps.OverlayValues[200]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
				d201 = ps.OverlayValues[201]
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
			if len(ps.OverlayValues) > 254 && ps.OverlayValues[254].Loc != scm.LocNone {
				d254 = ps.OverlayValues[254]
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
			if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != scm.LocNone {
				d278 = ps.OverlayValues[278]
			}
			if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != scm.LocNone {
				d279 = ps.OverlayValues[279]
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
			if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
				d294 = ps.OverlayValues[294]
			}
			if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
				d295 = ps.OverlayValues[295]
			}
			if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != scm.LocNone {
				d298 = ps.OverlayValues[298]
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
			if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != scm.LocNone {
				d305 = ps.OverlayValues[305]
			}
			if len(ps.OverlayValues) > 307 && ps.OverlayValues[307].Loc != scm.LocNone {
				d307 = ps.OverlayValues[307]
			}
			if len(ps.OverlayValues) > 308 && ps.OverlayValues[308].Loc != scm.LocNone {
				d308 = ps.OverlayValues[308]
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
			if len(ps.OverlayValues) > 313 && ps.OverlayValues[313].Loc != scm.LocNone {
				d313 = ps.OverlayValues[313]
			}
			if len(ps.OverlayValues) > 314 && ps.OverlayValues[314].Loc != scm.LocNone {
				d314 = ps.OverlayValues[314]
			}
			if len(ps.OverlayValues) > 315 && ps.OverlayValues[315].Loc != scm.LocNone {
				d315 = ps.OverlayValues[315]
			}
			if len(ps.OverlayValues) > 317 && ps.OverlayValues[317].Loc != scm.LocNone {
				d317 = ps.OverlayValues[317]
			}
			if len(ps.OverlayValues) > 318 && ps.OverlayValues[318].Loc != scm.LocNone {
				d318 = ps.OverlayValues[318]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d9)
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d9)
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d9)
			var d319 scm.JITValueDesc
			if d8.Loc == scm.LocImm && d9.Loc == scm.LocImm {
				d319 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() + d9.Imm.Int())}
			} else if d9.Loc == scm.LocImm && d9.Imm.Int() == 0 {
				r158 := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitMovRegReg(r158, d8.Reg)
				d319 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r158}
				ctx.BindReg(r158, &d319)
			} else if d8.Loc == scm.LocImm && d8.Imm.Int() == 0 {
				d319 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d9.Reg}
				ctx.BindReg(d9.Reg, &d319)
			} else if d8.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d8.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d9.Reg)
				d319 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d319)
			} else if d9.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitMovRegReg(scratch, d8.Reg)
				if d9.Imm.Int() >= -2147483648 && d9.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d9.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d9.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d319 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d319)
			} else {
				r159 := ctx.AllocRegExcept(d8.Reg, d9.Reg)
				ctx.W.EmitMovRegReg(r159, d8.Reg)
				ctx.W.EmitAddInt64(r159, d9.Reg)
				d319 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
				ctx.BindReg(r159, &d319)
			}
			if d319.Loc == scm.LocImm {
				d319 = scm.JITValueDesc{Loc: scm.LocImm, Type: d319.Type, Imm: scm.NewInt(int64(uint64(d319.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d319.Reg, 32)
				ctx.W.EmitShrRegImm8(d319.Reg, 32)
			}
			if d319.Loc == scm.LocReg && d8.Loc == scm.LocReg && d319.Reg == d8.Reg {
				ctx.TransferReg(d8.Reg)
				d8.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d319)
			var d320 scm.JITValueDesc
			if d319.Loc == scm.LocImm {
				d320 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d319.Imm.Int() / 2)}
			} else {
				r160 := ctx.AllocRegExcept(d319.Reg)
				ctx.W.EmitMovRegReg(r160, d319.Reg)
				ctx.W.EmitShrRegImm8(r160, 1)
				d320 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r160}
				ctx.BindReg(r160, &d320)
			}
			if d320.Loc == scm.LocImm {
				d320 = scm.JITValueDesc{Loc: scm.LocImm, Type: d320.Type, Imm: scm.NewInt(int64(uint64(d320.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d320.Reg, 32)
				ctx.W.EmitShrRegImm8(d320.Reg, 32)
			}
			if d320.Loc == scm.LocReg && d319.Loc == scm.LocReg && d320.Reg == d319.Reg {
				ctx.TransferReg(d319.Reg)
				d319.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d319)
			d321 = d320
			if d321.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d321)
			d322 = d321
			if d322.Loc == scm.LocImm {
				d322 = scm.JITValueDesc{Loc: scm.LocImm, Type: d322.Type, Imm: scm.NewInt(int64(uint64(d322.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d322.Reg, 32)
				ctx.W.EmitShrRegImm8(d322.Reg, 32)
			}
			ctx.EmitStoreToStack(d322, 8)
			d323 = d8
			if d323.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d323)
			d324 = d323
			if d324.Loc == scm.LocImm {
				d324 = scm.JITValueDesc{Loc: scm.LocImm, Type: d324.Type, Imm: scm.NewInt(int64(uint64(d324.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d324.Reg, 32)
				ctx.W.EmitShrRegImm8(d324.Reg, 32)
			}
			ctx.EmitStoreToStack(d324, 16)
			d325 = d9
			if d325.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d325)
			d326 = d325
			if d326.Loc == scm.LocImm {
				d326 = scm.JITValueDesc{Loc: scm.LocImm, Type: d326.Type, Imm: scm.NewInt(int64(uint64(d326.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d326.Reg, 32)
				ctx.W.EmitShrRegImm8(d326.Reg, 32)
			}
			ctx.EmitStoreToStack(d326, 24)
			ps327 := scm.PhiState{General: ps.General}
			ps327.OverlayValues = make([]scm.JITValueDesc, 327)
			ps327.OverlayValues[0] = d0
			ps327.OverlayValues[1] = d1
			ps327.OverlayValues[2] = d2
			ps327.OverlayValues[3] = d3
			ps327.OverlayValues[4] = d4
			ps327.OverlayValues[5] = d5
			ps327.OverlayValues[6] = d6
			ps327.OverlayValues[7] = d7
			ps327.OverlayValues[8] = d8
			ps327.OverlayValues[9] = d9
			ps327.OverlayValues[10] = d10
			ps327.OverlayValues[11] = d11
			ps327.OverlayValues[12] = d12
			ps327.OverlayValues[13] = d13
			ps327.OverlayValues[14] = d14
			ps327.OverlayValues[20] = d20
			ps327.OverlayValues[21] = d21
			ps327.OverlayValues[22] = d22
			ps327.OverlayValues[24] = d24
			ps327.OverlayValues[25] = d25
			ps327.OverlayValues[27] = d27
			ps327.OverlayValues[28] = d28
			ps327.OverlayValues[29] = d29
			ps327.OverlayValues[32] = d32
			ps327.OverlayValues[34] = d34
			ps327.OverlayValues[35] = d35
			ps327.OverlayValues[36] = d36
			ps327.OverlayValues[38] = d38
			ps327.OverlayValues[39] = d39
			ps327.OverlayValues[40] = d40
			ps327.OverlayValues[41] = d41
			ps327.OverlayValues[42] = d42
			ps327.OverlayValues[43] = d43
			ps327.OverlayValues[44] = d44
			ps327.OverlayValues[46] = d46
			ps327.OverlayValues[47] = d47
			ps327.OverlayValues[48] = d48
			ps327.OverlayValues[49] = d49
			ps327.OverlayValues[50] = d50
			ps327.OverlayValues[51] = d51
			ps327.OverlayValues[52] = d52
			ps327.OverlayValues[53] = d53
			ps327.OverlayValues[54] = d54
			ps327.OverlayValues[55] = d55
			ps327.OverlayValues[56] = d56
			ps327.OverlayValues[57] = d57
			ps327.OverlayValues[58] = d58
			ps327.OverlayValues[59] = d59
			ps327.OverlayValues[60] = d60
			ps327.OverlayValues[61] = d61
			ps327.OverlayValues[62] = d62
			ps327.OverlayValues[63] = d63
			ps327.OverlayValues[64] = d64
			ps327.OverlayValues[65] = d65
			ps327.OverlayValues[66] = d66
			ps327.OverlayValues[67] = d67
			ps327.OverlayValues[68] = d68
			ps327.OverlayValues[69] = d69
			ps327.OverlayValues[70] = d70
			ps327.OverlayValues[71] = d71
			ps327.OverlayValues[72] = d72
			ps327.OverlayValues[73] = d73
			ps327.OverlayValues[74] = d74
			ps327.OverlayValues[75] = d75
			ps327.OverlayValues[76] = d76
			ps327.OverlayValues[77] = d77
			ps327.OverlayValues[78] = d78
			ps327.OverlayValues[79] = d79
			ps327.OverlayValues[80] = d80
			ps327.OverlayValues[81] = d81
			ps327.OverlayValues[82] = d82
			ps327.OverlayValues[83] = d83
			ps327.OverlayValues[84] = d84
			ps327.OverlayValues[85] = d85
			ps327.OverlayValues[86] = d86
			ps327.OverlayValues[87] = d87
			ps327.OverlayValues[88] = d88
			ps327.OverlayValues[89] = d89
			ps327.OverlayValues[90] = d90
			ps327.OverlayValues[91] = d91
			ps327.OverlayValues[92] = d92
			ps327.OverlayValues[93] = d93
			ps327.OverlayValues[94] = d94
			ps327.OverlayValues[95] = d95
			ps327.OverlayValues[102] = d102
			ps327.OverlayValues[103] = d103
			ps327.OverlayValues[104] = d104
			ps327.OverlayValues[105] = d105
			ps327.OverlayValues[106] = d106
			ps327.OverlayValues[107] = d107
			ps327.OverlayValues[108] = d108
			ps327.OverlayValues[109] = d109
			ps327.OverlayValues[110] = d110
			ps327.OverlayValues[111] = d111
			ps327.OverlayValues[112] = d112
			ps327.OverlayValues[113] = d113
			ps327.OverlayValues[114] = d114
			ps327.OverlayValues[115] = d115
			ps327.OverlayValues[116] = d116
			ps327.OverlayValues[117] = d117
			ps327.OverlayValues[118] = d118
			ps327.OverlayValues[119] = d119
			ps327.OverlayValues[120] = d120
			ps327.OverlayValues[121] = d121
			ps327.OverlayValues[122] = d122
			ps327.OverlayValues[123] = d123
			ps327.OverlayValues[124] = d124
			ps327.OverlayValues[125] = d125
			ps327.OverlayValues[126] = d126
			ps327.OverlayValues[127] = d127
			ps327.OverlayValues[128] = d128
			ps327.OverlayValues[129] = d129
			ps327.OverlayValues[130] = d130
			ps327.OverlayValues[131] = d131
			ps327.OverlayValues[132] = d132
			ps327.OverlayValues[133] = d133
			ps327.OverlayValues[134] = d134
			ps327.OverlayValues[135] = d135
			ps327.OverlayValues[136] = d136
			ps327.OverlayValues[137] = d137
			ps327.OverlayValues[138] = d138
			ps327.OverlayValues[139] = d139
			ps327.OverlayValues[140] = d140
			ps327.OverlayValues[141] = d141
			ps327.OverlayValues[142] = d142
			ps327.OverlayValues[143] = d143
			ps327.OverlayValues[144] = d144
			ps327.OverlayValues[145] = d145
			ps327.OverlayValues[146] = d146
			ps327.OverlayValues[153] = d153
			ps327.OverlayValues[154] = d154
			ps327.OverlayValues[160] = d160
			ps327.OverlayValues[161] = d161
			ps327.OverlayValues[162] = d162
			ps327.OverlayValues[163] = d163
			ps327.OverlayValues[164] = d164
			ps327.OverlayValues[165] = d165
			ps327.OverlayValues[166] = d166
			ps327.OverlayValues[168] = d168
			ps327.OverlayValues[170] = d170
			ps327.OverlayValues[171] = d171
			ps327.OverlayValues[174] = d174
			ps327.OverlayValues[177] = d177
			ps327.OverlayValues[178] = d178
			ps327.OverlayValues[179] = d179
			ps327.OverlayValues[181] = d181
			ps327.OverlayValues[182] = d182
			ps327.OverlayValues[183] = d183
			ps327.OverlayValues[184] = d184
			ps327.OverlayValues[185] = d185
			ps327.OverlayValues[186] = d186
			ps327.OverlayValues[188] = d188
			ps327.OverlayValues[189] = d189
			ps327.OverlayValues[190] = d190
			ps327.OverlayValues[191] = d191
			ps327.OverlayValues[192] = d192
			ps327.OverlayValues[193] = d193
			ps327.OverlayValues[194] = d194
			ps327.OverlayValues[195] = d195
			ps327.OverlayValues[196] = d196
			ps327.OverlayValues[199] = d199
			ps327.OverlayValues[200] = d200
			ps327.OverlayValues[201] = d201
			ps327.OverlayValues[204] = d204
			ps327.OverlayValues[205] = d205
			ps327.OverlayValues[206] = d206
			ps327.OverlayValues[207] = d207
			ps327.OverlayValues[208] = d208
			ps327.OverlayValues[209] = d209
			ps327.OverlayValues[210] = d210
			ps327.OverlayValues[211] = d211
			ps327.OverlayValues[212] = d212
			ps327.OverlayValues[214] = d214
			ps327.OverlayValues[215] = d215
			ps327.OverlayValues[216] = d216
			ps327.OverlayValues[217] = d217
			ps327.OverlayValues[218] = d218
			ps327.OverlayValues[219] = d219
			ps327.OverlayValues[220] = d220
			ps327.OverlayValues[221] = d221
			ps327.OverlayValues[222] = d222
			ps327.OverlayValues[223] = d223
			ps327.OverlayValues[225] = d225
			ps327.OverlayValues[226] = d226
			ps327.OverlayValues[227] = d227
			ps327.OverlayValues[228] = d228
			ps327.OverlayValues[229] = d229
			ps327.OverlayValues[230] = d230
			ps327.OverlayValues[231] = d231
			ps327.OverlayValues[232] = d232
			ps327.OverlayValues[233] = d233
			ps327.OverlayValues[234] = d234
			ps327.OverlayValues[235] = d235
			ps327.OverlayValues[236] = d236
			ps327.OverlayValues[237] = d237
			ps327.OverlayValues[238] = d238
			ps327.OverlayValues[239] = d239
			ps327.OverlayValues[240] = d240
			ps327.OverlayValues[241] = d241
			ps327.OverlayValues[242] = d242
			ps327.OverlayValues[243] = d243
			ps327.OverlayValues[244] = d244
			ps327.OverlayValues[245] = d245
			ps327.OverlayValues[246] = d246
			ps327.OverlayValues[247] = d247
			ps327.OverlayValues[248] = d248
			ps327.OverlayValues[249] = d249
			ps327.OverlayValues[250] = d250
			ps327.OverlayValues[251] = d251
			ps327.OverlayValues[252] = d252
			ps327.OverlayValues[253] = d253
			ps327.OverlayValues[254] = d254
			ps327.OverlayValues[255] = d255
			ps327.OverlayValues[256] = d256
			ps327.OverlayValues[257] = d257
			ps327.OverlayValues[258] = d258
			ps327.OverlayValues[259] = d259
			ps327.OverlayValues[260] = d260
			ps327.OverlayValues[261] = d261
			ps327.OverlayValues[262] = d262
			ps327.OverlayValues[263] = d263
			ps327.OverlayValues[264] = d264
			ps327.OverlayValues[265] = d265
			ps327.OverlayValues[266] = d266
			ps327.OverlayValues[267] = d267
			ps327.OverlayValues[268] = d268
			ps327.OverlayValues[269] = d269
			ps327.OverlayValues[270] = d270
			ps327.OverlayValues[271] = d271
			ps327.OverlayValues[278] = d278
			ps327.OverlayValues[279] = d279
			ps327.OverlayValues[285] = d285
			ps327.OverlayValues[286] = d286
			ps327.OverlayValues[287] = d287
			ps327.OverlayValues[288] = d288
			ps327.OverlayValues[289] = d289
			ps327.OverlayValues[290] = d290
			ps327.OverlayValues[292] = d292
			ps327.OverlayValues[294] = d294
			ps327.OverlayValues[295] = d295
			ps327.OverlayValues[298] = d298
			ps327.OverlayValues[302] = d302
			ps327.OverlayValues[303] = d303
			ps327.OverlayValues[304] = d304
			ps327.OverlayValues[305] = d305
			ps327.OverlayValues[307] = d307
			ps327.OverlayValues[308] = d308
			ps327.OverlayValues[310] = d310
			ps327.OverlayValues[311] = d311
			ps327.OverlayValues[312] = d312
			ps327.OverlayValues[313] = d313
			ps327.OverlayValues[314] = d314
			ps327.OverlayValues[315] = d315
			ps327.OverlayValues[317] = d317
			ps327.OverlayValues[318] = d318
			ps327.OverlayValues[319] = d319
			ps327.OverlayValues[320] = d320
			ps327.OverlayValues[321] = d321
			ps327.OverlayValues[322] = d322
			ps327.OverlayValues[323] = d323
			ps327.OverlayValues[324] = d324
			ps327.OverlayValues[325] = d325
			ps327.OverlayValues[326] = d326
			ps327.PhiValues = make([]scm.JITValueDesc, 3)
			d328 = d320
			ps327.PhiValues[0] = d328
			d329 = d8
			ps327.PhiValues[1] = d329
			d330 = d9
			ps327.PhiValues[2] = d330
			if ps327.General && bbs[5].Rendered {
				ctx.W.EmitJmp(lbl6)
				return result
			}
			return bbs[5].RenderPS(ps327)
			return result
			}
			bbs[20].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[20].VisitCount >= 2 {
					ps.General = true
					return bbs[20].RenderPS(ps)
				}
			}
			bbs[20].VisitCount++
			if ps.General {
				if bbs[20].Rendered {
					ctx.W.EmitJmp(lbl21)
					return result
				}
				bbs[20].Rendered = true
				bbs[20].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_20 = bbs[20].Address
				ctx.W.MarkLabel(lbl21)
				ctx.W.ResolveFixups()
			}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
				d154 = ps.OverlayValues[154]
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
			if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
				d168 = ps.OverlayValues[168]
			}
			if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != scm.LocNone {
				d170 = ps.OverlayValues[170]
			}
			if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != scm.LocNone {
				d171 = ps.OverlayValues[171]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
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
			if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
				d199 = ps.OverlayValues[199]
			}
			if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
				d200 = ps.OverlayValues[200]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
				d201 = ps.OverlayValues[201]
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
			if len(ps.OverlayValues) > 254 && ps.OverlayValues[254].Loc != scm.LocNone {
				d254 = ps.OverlayValues[254]
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
			if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != scm.LocNone {
				d278 = ps.OverlayValues[278]
			}
			if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != scm.LocNone {
				d279 = ps.OverlayValues[279]
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
			if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
				d294 = ps.OverlayValues[294]
			}
			if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
				d295 = ps.OverlayValues[295]
			}
			if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != scm.LocNone {
				d298 = ps.OverlayValues[298]
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
			if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != scm.LocNone {
				d305 = ps.OverlayValues[305]
			}
			if len(ps.OverlayValues) > 307 && ps.OverlayValues[307].Loc != scm.LocNone {
				d307 = ps.OverlayValues[307]
			}
			if len(ps.OverlayValues) > 308 && ps.OverlayValues[308].Loc != scm.LocNone {
				d308 = ps.OverlayValues[308]
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
			if len(ps.OverlayValues) > 313 && ps.OverlayValues[313].Loc != scm.LocNone {
				d313 = ps.OverlayValues[313]
			}
			if len(ps.OverlayValues) > 314 && ps.OverlayValues[314].Loc != scm.LocNone {
				d314 = ps.OverlayValues[314]
			}
			if len(ps.OverlayValues) > 315 && ps.OverlayValues[315].Loc != scm.LocNone {
				d315 = ps.OverlayValues[315]
			}
			if len(ps.OverlayValues) > 317 && ps.OverlayValues[317].Loc != scm.LocNone {
				d317 = ps.OverlayValues[317]
			}
			if len(ps.OverlayValues) > 318 && ps.OverlayValues[318].Loc != scm.LocNone {
				d318 = ps.OverlayValues[318]
			}
			if len(ps.OverlayValues) > 319 && ps.OverlayValues[319].Loc != scm.LocNone {
				d319 = ps.OverlayValues[319]
			}
			if len(ps.OverlayValues) > 320 && ps.OverlayValues[320].Loc != scm.LocNone {
				d320 = ps.OverlayValues[320]
			}
			if len(ps.OverlayValues) > 321 && ps.OverlayValues[321].Loc != scm.LocNone {
				d321 = ps.OverlayValues[321]
			}
			if len(ps.OverlayValues) > 322 && ps.OverlayValues[322].Loc != scm.LocNone {
				d322 = ps.OverlayValues[322]
			}
			if len(ps.OverlayValues) > 323 && ps.OverlayValues[323].Loc != scm.LocNone {
				d323 = ps.OverlayValues[323]
			}
			if len(ps.OverlayValues) > 324 && ps.OverlayValues[324].Loc != scm.LocNone {
				d324 = ps.OverlayValues[324]
			}
			if len(ps.OverlayValues) > 325 && ps.OverlayValues[325].Loc != scm.LocNone {
				d325 = ps.OverlayValues[325]
			}
			if len(ps.OverlayValues) > 326 && ps.OverlayValues[326].Loc != scm.LocNone {
				d326 = ps.OverlayValues[326]
			}
			if len(ps.OverlayValues) > 328 && ps.OverlayValues[328].Loc != scm.LocNone {
				d328 = ps.OverlayValues[328]
			}
			if len(ps.OverlayValues) > 329 && ps.OverlayValues[329].Loc != scm.LocNone {
				d329 = ps.OverlayValues[329]
			}
			if len(ps.OverlayValues) > 330 && ps.OverlayValues[330].Loc != scm.LocNone {
				d330 = ps.OverlayValues[330]
			}
			ctx.ReclaimUntrackedRegs()
			d331 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d331)
			ctx.BindReg(r2, &d331)
			ctx.W.EmitMakeNil(d331)
			ctx.W.EmitJmp(lbl0)
			return result
			}
			bbs[21].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[21].VisitCount >= 2 {
					ps.General = true
					return bbs[21].RenderPS(ps)
				}
			}
			bbs[21].VisitCount++
			if ps.General {
				if bbs[21].Rendered {
					ctx.W.EmitJmp(lbl22)
					return result
				}
				bbs[21].Rendered = true
				bbs[21].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_21 = bbs[21].Address
				ctx.W.MarkLabel(lbl22)
				ctx.W.ResolveFixups()
			}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
				d154 = ps.OverlayValues[154]
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
			if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
				d168 = ps.OverlayValues[168]
			}
			if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != scm.LocNone {
				d170 = ps.OverlayValues[170]
			}
			if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != scm.LocNone {
				d171 = ps.OverlayValues[171]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
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
			if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
				d199 = ps.OverlayValues[199]
			}
			if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
				d200 = ps.OverlayValues[200]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
				d201 = ps.OverlayValues[201]
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
			if len(ps.OverlayValues) > 254 && ps.OverlayValues[254].Loc != scm.LocNone {
				d254 = ps.OverlayValues[254]
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
			if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != scm.LocNone {
				d278 = ps.OverlayValues[278]
			}
			if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != scm.LocNone {
				d279 = ps.OverlayValues[279]
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
			if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
				d294 = ps.OverlayValues[294]
			}
			if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
				d295 = ps.OverlayValues[295]
			}
			if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != scm.LocNone {
				d298 = ps.OverlayValues[298]
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
			if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != scm.LocNone {
				d305 = ps.OverlayValues[305]
			}
			if len(ps.OverlayValues) > 307 && ps.OverlayValues[307].Loc != scm.LocNone {
				d307 = ps.OverlayValues[307]
			}
			if len(ps.OverlayValues) > 308 && ps.OverlayValues[308].Loc != scm.LocNone {
				d308 = ps.OverlayValues[308]
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
			if len(ps.OverlayValues) > 313 && ps.OverlayValues[313].Loc != scm.LocNone {
				d313 = ps.OverlayValues[313]
			}
			if len(ps.OverlayValues) > 314 && ps.OverlayValues[314].Loc != scm.LocNone {
				d314 = ps.OverlayValues[314]
			}
			if len(ps.OverlayValues) > 315 && ps.OverlayValues[315].Loc != scm.LocNone {
				d315 = ps.OverlayValues[315]
			}
			if len(ps.OverlayValues) > 317 && ps.OverlayValues[317].Loc != scm.LocNone {
				d317 = ps.OverlayValues[317]
			}
			if len(ps.OverlayValues) > 318 && ps.OverlayValues[318].Loc != scm.LocNone {
				d318 = ps.OverlayValues[318]
			}
			if len(ps.OverlayValues) > 319 && ps.OverlayValues[319].Loc != scm.LocNone {
				d319 = ps.OverlayValues[319]
			}
			if len(ps.OverlayValues) > 320 && ps.OverlayValues[320].Loc != scm.LocNone {
				d320 = ps.OverlayValues[320]
			}
			if len(ps.OverlayValues) > 321 && ps.OverlayValues[321].Loc != scm.LocNone {
				d321 = ps.OverlayValues[321]
			}
			if len(ps.OverlayValues) > 322 && ps.OverlayValues[322].Loc != scm.LocNone {
				d322 = ps.OverlayValues[322]
			}
			if len(ps.OverlayValues) > 323 && ps.OverlayValues[323].Loc != scm.LocNone {
				d323 = ps.OverlayValues[323]
			}
			if len(ps.OverlayValues) > 324 && ps.OverlayValues[324].Loc != scm.LocNone {
				d324 = ps.OverlayValues[324]
			}
			if len(ps.OverlayValues) > 325 && ps.OverlayValues[325].Loc != scm.LocNone {
				d325 = ps.OverlayValues[325]
			}
			if len(ps.OverlayValues) > 326 && ps.OverlayValues[326].Loc != scm.LocNone {
				d326 = ps.OverlayValues[326]
			}
			if len(ps.OverlayValues) > 328 && ps.OverlayValues[328].Loc != scm.LocNone {
				d328 = ps.OverlayValues[328]
			}
			if len(ps.OverlayValues) > 329 && ps.OverlayValues[329].Loc != scm.LocNone {
				d329 = ps.OverlayValues[329]
			}
			if len(ps.OverlayValues) > 330 && ps.OverlayValues[330].Loc != scm.LocNone {
				d330 = ps.OverlayValues[330]
			}
			if len(ps.OverlayValues) > 331 && ps.OverlayValues[331].Loc != scm.LocNone {
				d331 = ps.OverlayValues[331]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d4)
			d332 = d4
			_ = d332
			r161 := d4.Loc == scm.LocReg
			r162 := d4.Reg
			if r161 { ctx.ProtectReg(r162) }
			d333 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(104)}
			lbl68 := ctx.W.ReserveLabel()
			bbpos_4_0 := int32(-1)
			_ = bbpos_4_0
			bbpos_4_1 := int32(-1)
			_ = bbpos_4_1
			bbpos_4_2 := int32(-1)
			_ = bbpos_4_2
			bbpos_4_3 := int32(-1)
			_ = bbpos_4_3
			bbpos_4_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			d333 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(104)}
			ctx.EnsureDesc(&d332)
			ctx.EnsureDesc(&d332)
			var d334 scm.JITValueDesc
			if d332.Loc == scm.LocImm {
				d334 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d332.Imm.Int()))))}
			} else {
				r163 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r163, d332.Reg)
				ctx.W.EmitShlRegImm8(r163, 32)
				ctx.W.EmitShrRegImm8(r163, 32)
				d334 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r163}
				ctx.BindReg(r163, &d334)
			}
			var d335 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d335 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r164 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r164, thisptr.Reg, off)
				d335 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r164}
				ctx.BindReg(r164, &d335)
			}
			ctx.EnsureDesc(&d335)
			ctx.EnsureDesc(&d335)
			var d336 scm.JITValueDesc
			if d335.Loc == scm.LocImm {
				d336 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d335.Imm.Int()))))}
			} else {
				r165 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r165, d335.Reg)
				ctx.W.EmitShlRegImm8(r165, 56)
				ctx.W.EmitShrRegImm8(r165, 56)
				d336 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r165}
				ctx.BindReg(r165, &d336)
			}
			ctx.FreeDesc(&d335)
			ctx.EnsureDesc(&d334)
			ctx.EnsureDesc(&d336)
			ctx.EnsureDesc(&d334)
			ctx.EnsureDesc(&d336)
			ctx.EnsureDesc(&d334)
			ctx.EnsureDesc(&d336)
			var d337 scm.JITValueDesc
			if d334.Loc == scm.LocImm && d336.Loc == scm.LocImm {
				d337 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d334.Imm.Int() * d336.Imm.Int())}
			} else if d334.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d336.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d334.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d336.Reg)
				d337 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d337)
			} else if d336.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d334.Reg)
				ctx.W.EmitMovRegReg(scratch, d334.Reg)
				if d336.Imm.Int() >= -2147483648 && d336.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d336.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d336.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d337 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d337)
			} else {
				r166 := ctx.AllocRegExcept(d334.Reg, d336.Reg)
				ctx.W.EmitMovRegReg(r166, d334.Reg)
				ctx.W.EmitImulInt64(r166, d336.Reg)
				d337 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r166}
				ctx.BindReg(r166, &d337)
			}
			if d337.Loc == scm.LocReg && d334.Loc == scm.LocReg && d337.Reg == d334.Reg {
				ctx.TransferReg(d334.Reg)
				d334.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d334)
			ctx.FreeDesc(&d336)
			var d338 scm.JITValueDesc
			r167 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r167, uint64(dataPtr))
				d338 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r167, StackOff: int32(sliceLen)}
				ctx.BindReg(r167, &d338)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 0)
				ctx.W.EmitMovRegMem(r167, thisptr.Reg, off)
				d338 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r167}
				ctx.BindReg(r167, &d338)
			}
			ctx.BindReg(r167, &d338)
			ctx.EnsureDesc(&d337)
			var d339 scm.JITValueDesc
			if d337.Loc == scm.LocImm {
				d339 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d337.Imm.Int() / 64)}
			} else {
				r168 := ctx.AllocRegExcept(d337.Reg)
				ctx.W.EmitMovRegReg(r168, d337.Reg)
				ctx.W.EmitShrRegImm8(r168, 6)
				d339 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r168}
				ctx.BindReg(r168, &d339)
			}
			if d339.Loc == scm.LocReg && d337.Loc == scm.LocReg && d339.Reg == d337.Reg {
				ctx.TransferReg(d337.Reg)
				d337.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d339)
			r169 := ctx.AllocReg()
			ctx.EnsureDesc(&d339)
			ctx.EnsureDesc(&d338)
			if d339.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r169, uint64(d339.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r169, d339.Reg)
				ctx.W.EmitShlRegImm8(r169, 3)
			}
			if d338.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d338.Imm.Int()))
				ctx.W.EmitAddInt64(r169, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r169, d338.Reg)
			}
			r170 := ctx.AllocRegExcept(r169)
			ctx.W.EmitMovRegMem(r170, r169, 0)
			ctx.FreeReg(r169)
			d340 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r170}
			ctx.BindReg(r170, &d340)
			ctx.FreeDesc(&d339)
			ctx.EnsureDesc(&d337)
			var d341 scm.JITValueDesc
			if d337.Loc == scm.LocImm {
				d341 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d337.Imm.Int() % 64)}
			} else {
				r171 := ctx.AllocRegExcept(d337.Reg)
				ctx.W.EmitMovRegReg(r171, d337.Reg)
				ctx.W.EmitAndRegImm32(r171, 63)
				d341 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r171}
				ctx.BindReg(r171, &d341)
			}
			if d341.Loc == scm.LocReg && d337.Loc == scm.LocReg && d341.Reg == d337.Reg {
				ctx.TransferReg(d337.Reg)
				d337.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d340)
			ctx.EnsureDesc(&d341)
			var d342 scm.JITValueDesc
			if d340.Loc == scm.LocImm && d341.Loc == scm.LocImm {
				d342 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d340.Imm.Int()) << uint64(d341.Imm.Int())))}
			} else if d341.Loc == scm.LocImm {
				r172 := ctx.AllocRegExcept(d340.Reg)
				ctx.W.EmitMovRegReg(r172, d340.Reg)
				ctx.W.EmitShlRegImm8(r172, uint8(d341.Imm.Int()))
				d342 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
				ctx.BindReg(r172, &d342)
			} else {
				{
					shiftSrc := d340.Reg
					r173 := ctx.AllocRegExcept(d340.Reg)
					ctx.W.EmitMovRegReg(r173, d340.Reg)
					shiftSrc = r173
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d341.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d341.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d341.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d342 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d342)
				}
			}
			if d342.Loc == scm.LocReg && d340.Loc == scm.LocReg && d342.Reg == d340.Reg {
				ctx.TransferReg(d340.Reg)
				d340.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d340)
			ctx.FreeDesc(&d341)
			var d343 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d343 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 25)
				r174 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r174, thisptr.Reg, off)
				d343 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r174}
				ctx.BindReg(r174, &d343)
			}
			d344 = d343
			ctx.EnsureDesc(&d344)
			if d344.Loc != scm.LocImm && d344.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl69 := ctx.W.ReserveLabel()
			lbl70 := ctx.W.ReserveLabel()
			lbl71 := ctx.W.ReserveLabel()
			lbl72 := ctx.W.ReserveLabel()
			if d344.Loc == scm.LocImm {
				if d344.Imm.Bool() {
					ctx.W.MarkLabel(lbl71)
					ctx.W.EmitJmp(lbl69)
				} else {
					ctx.W.MarkLabel(lbl72)
			d345 = d342
			if d345.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d345)
			ctx.EmitStoreToStack(d345, 104)
					ctx.W.EmitJmp(lbl70)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d344.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl71)
				ctx.W.EmitJmp(lbl72)
				ctx.W.MarkLabel(lbl71)
				ctx.W.EmitJmp(lbl69)
				ctx.W.MarkLabel(lbl72)
			d346 = d342
			if d346.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d346)
			ctx.EmitStoreToStack(d346, 104)
				ctx.W.EmitJmp(lbl70)
			}
			ctx.FreeDesc(&d343)
			bbpos_4_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl70)
			ctx.W.ResolveFixups()
			d333 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(104)}
			var d347 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d347 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r175 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r175, thisptr.Reg, off)
				d347 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r175}
				ctx.BindReg(r175, &d347)
			}
			ctx.EnsureDesc(&d347)
			ctx.EnsureDesc(&d347)
			var d348 scm.JITValueDesc
			if d347.Loc == scm.LocImm {
				d348 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d347.Imm.Int()))))}
			} else {
				r176 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r176, d347.Reg)
				ctx.W.EmitShlRegImm8(r176, 56)
				ctx.W.EmitShrRegImm8(r176, 56)
				d348 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r176}
				ctx.BindReg(r176, &d348)
			}
			ctx.FreeDesc(&d347)
			d349 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d348)
			ctx.EnsureDesc(&d349)
			ctx.EnsureDesc(&d348)
			ctx.EnsureDesc(&d349)
			ctx.EnsureDesc(&d348)
			var d350 scm.JITValueDesc
			if d349.Loc == scm.LocImm && d348.Loc == scm.LocImm {
				d350 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d349.Imm.Int() - d348.Imm.Int())}
			} else if d348.Loc == scm.LocImm && d348.Imm.Int() == 0 {
				r177 := ctx.AllocRegExcept(d349.Reg)
				ctx.W.EmitMovRegReg(r177, d349.Reg)
				d350 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r177}
				ctx.BindReg(r177, &d350)
			} else if d349.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d348.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d349.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d348.Reg)
				d350 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d350)
			} else if d348.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d349.Reg)
				ctx.W.EmitMovRegReg(scratch, d349.Reg)
				if d348.Imm.Int() >= -2147483648 && d348.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d348.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d348.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d350 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d350)
			} else {
				r178 := ctx.AllocRegExcept(d349.Reg, d348.Reg)
				ctx.W.EmitMovRegReg(r178, d349.Reg)
				ctx.W.EmitSubInt64(r178, d348.Reg)
				d350 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r178}
				ctx.BindReg(r178, &d350)
			}
			if d350.Loc == scm.LocReg && d349.Loc == scm.LocReg && d350.Reg == d349.Reg {
				ctx.TransferReg(d349.Reg)
				d349.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d348)
			ctx.EnsureDesc(&d333)
			ctx.EnsureDesc(&d350)
			var d351 scm.JITValueDesc
			if d333.Loc == scm.LocImm && d350.Loc == scm.LocImm {
				d351 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d333.Imm.Int()) >> uint64(d350.Imm.Int())))}
			} else if d350.Loc == scm.LocImm {
				r179 := ctx.AllocRegExcept(d333.Reg)
				ctx.W.EmitMovRegReg(r179, d333.Reg)
				ctx.W.EmitShrRegImm8(r179, uint8(d350.Imm.Int()))
				d351 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r179}
				ctx.BindReg(r179, &d351)
			} else {
				{
					shiftSrc := d333.Reg
					r180 := ctx.AllocRegExcept(d333.Reg)
					ctx.W.EmitMovRegReg(r180, d333.Reg)
					shiftSrc = r180
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d350.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d350.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d350.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d351 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d351)
				}
			}
			if d351.Loc == scm.LocReg && d333.Loc == scm.LocReg && d351.Reg == d333.Reg {
				ctx.TransferReg(d333.Reg)
				d333.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d333)
			ctx.FreeDesc(&d350)
			r181 := ctx.AllocReg()
			ctx.EnsureDesc(&d351)
			ctx.EnsureDesc(&d351)
			if d351.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r181, d351)
			}
			ctx.W.EmitJmp(lbl68)
			bbpos_4_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl69)
			ctx.W.ResolveFixups()
			d333 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(104)}
			ctx.EnsureDesc(&d337)
			var d352 scm.JITValueDesc
			if d337.Loc == scm.LocImm {
				d352 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d337.Imm.Int() % 64)}
			} else {
				r182 := ctx.AllocRegExcept(d337.Reg)
				ctx.W.EmitMovRegReg(r182, d337.Reg)
				ctx.W.EmitAndRegImm32(r182, 63)
				d352 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r182}
				ctx.BindReg(r182, &d352)
			}
			if d352.Loc == scm.LocReg && d337.Loc == scm.LocReg && d352.Reg == d337.Reg {
				ctx.TransferReg(d337.Reg)
				d337.Loc = scm.LocNone
			}
			var d353 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d353 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r183 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r183, thisptr.Reg, off)
				d353 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r183}
				ctx.BindReg(r183, &d353)
			}
			ctx.EnsureDesc(&d353)
			ctx.EnsureDesc(&d353)
			var d354 scm.JITValueDesc
			if d353.Loc == scm.LocImm {
				d354 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d353.Imm.Int()))))}
			} else {
				r184 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r184, d353.Reg)
				ctx.W.EmitShlRegImm8(r184, 56)
				ctx.W.EmitShrRegImm8(r184, 56)
				d354 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r184}
				ctx.BindReg(r184, &d354)
			}
			ctx.FreeDesc(&d353)
			ctx.EnsureDesc(&d352)
			ctx.EnsureDesc(&d354)
			ctx.EnsureDesc(&d352)
			ctx.EnsureDesc(&d354)
			ctx.EnsureDesc(&d352)
			ctx.EnsureDesc(&d354)
			var d355 scm.JITValueDesc
			if d352.Loc == scm.LocImm && d354.Loc == scm.LocImm {
				d355 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d352.Imm.Int() + d354.Imm.Int())}
			} else if d354.Loc == scm.LocImm && d354.Imm.Int() == 0 {
				r185 := ctx.AllocRegExcept(d352.Reg)
				ctx.W.EmitMovRegReg(r185, d352.Reg)
				d355 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r185}
				ctx.BindReg(r185, &d355)
			} else if d352.Loc == scm.LocImm && d352.Imm.Int() == 0 {
				d355 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d354.Reg}
				ctx.BindReg(d354.Reg, &d355)
			} else if d352.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d354.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d352.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d354.Reg)
				d355 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d355)
			} else if d354.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d352.Reg)
				ctx.W.EmitMovRegReg(scratch, d352.Reg)
				if d354.Imm.Int() >= -2147483648 && d354.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d354.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d354.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d355 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d355)
			} else {
				r186 := ctx.AllocRegExcept(d352.Reg, d354.Reg)
				ctx.W.EmitMovRegReg(r186, d352.Reg)
				ctx.W.EmitAddInt64(r186, d354.Reg)
				d355 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r186}
				ctx.BindReg(r186, &d355)
			}
			if d355.Loc == scm.LocReg && d352.Loc == scm.LocReg && d355.Reg == d352.Reg {
				ctx.TransferReg(d352.Reg)
				d352.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d352)
			ctx.FreeDesc(&d354)
			ctx.EnsureDesc(&d355)
			var d356 scm.JITValueDesc
			if d355.Loc == scm.LocImm {
				d356 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d355.Imm.Int()) > uint64(64))}
			} else {
				r187 := ctx.AllocRegExcept(d355.Reg)
				ctx.W.EmitCmpRegImm32(d355.Reg, 64)
				ctx.W.EmitSetcc(r187, scm.CcA)
				d356 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r187}
				ctx.BindReg(r187, &d356)
			}
			ctx.FreeDesc(&d355)
			d357 = d356
			ctx.EnsureDesc(&d357)
			if d357.Loc != scm.LocImm && d357.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl73 := ctx.W.ReserveLabel()
			lbl74 := ctx.W.ReserveLabel()
			lbl75 := ctx.W.ReserveLabel()
			if d357.Loc == scm.LocImm {
				if d357.Imm.Bool() {
					ctx.W.MarkLabel(lbl74)
					ctx.W.EmitJmp(lbl73)
				} else {
					ctx.W.MarkLabel(lbl75)
			d358 = d342
			if d358.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d358)
			ctx.EmitStoreToStack(d358, 104)
					ctx.W.EmitJmp(lbl70)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d357.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl74)
				ctx.W.EmitJmp(lbl75)
				ctx.W.MarkLabel(lbl74)
				ctx.W.EmitJmp(lbl73)
				ctx.W.MarkLabel(lbl75)
			d359 = d342
			if d359.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d359)
			ctx.EmitStoreToStack(d359, 104)
				ctx.W.EmitJmp(lbl70)
			}
			ctx.FreeDesc(&d356)
			bbpos_4_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl73)
			ctx.W.ResolveFixups()
			d333 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(104)}
			ctx.EnsureDesc(&d337)
			var d360 scm.JITValueDesc
			if d337.Loc == scm.LocImm {
				d360 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d337.Imm.Int() / 64)}
			} else {
				r188 := ctx.AllocRegExcept(d337.Reg)
				ctx.W.EmitMovRegReg(r188, d337.Reg)
				ctx.W.EmitShrRegImm8(r188, 6)
				d360 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r188}
				ctx.BindReg(r188, &d360)
			}
			if d360.Loc == scm.LocReg && d337.Loc == scm.LocReg && d360.Reg == d337.Reg {
				ctx.TransferReg(d337.Reg)
				d337.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d360)
			ctx.EnsureDesc(&d360)
			var d361 scm.JITValueDesc
			if d360.Loc == scm.LocImm {
				d361 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d360.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d360.Reg)
				ctx.W.EmitMovRegReg(scratch, d360.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d361 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d361)
			}
			if d361.Loc == scm.LocReg && d360.Loc == scm.LocReg && d361.Reg == d360.Reg {
				ctx.TransferReg(d360.Reg)
				d360.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d360)
			ctx.EnsureDesc(&d361)
			r189 := ctx.AllocReg()
			ctx.EnsureDesc(&d361)
			ctx.EnsureDesc(&d338)
			if d361.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r189, uint64(d361.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r189, d361.Reg)
				ctx.W.EmitShlRegImm8(r189, 3)
			}
			if d338.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d338.Imm.Int()))
				ctx.W.EmitAddInt64(r189, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r189, d338.Reg)
			}
			r190 := ctx.AllocRegExcept(r189)
			ctx.W.EmitMovRegMem(r190, r189, 0)
			ctx.FreeReg(r189)
			d362 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r190}
			ctx.BindReg(r190, &d362)
			ctx.FreeDesc(&d361)
			ctx.EnsureDesc(&d337)
			var d363 scm.JITValueDesc
			if d337.Loc == scm.LocImm {
				d363 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d337.Imm.Int() % 64)}
			} else {
				r191 := ctx.AllocRegExcept(d337.Reg)
				ctx.W.EmitMovRegReg(r191, d337.Reg)
				ctx.W.EmitAndRegImm32(r191, 63)
				d363 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r191}
				ctx.BindReg(r191, &d363)
			}
			if d363.Loc == scm.LocReg && d337.Loc == scm.LocReg && d363.Reg == d337.Reg {
				ctx.TransferReg(d337.Reg)
				d337.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d337)
			d364 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d363)
			ctx.EnsureDesc(&d364)
			ctx.EnsureDesc(&d363)
			ctx.EnsureDesc(&d364)
			ctx.EnsureDesc(&d363)
			var d365 scm.JITValueDesc
			if d364.Loc == scm.LocImm && d363.Loc == scm.LocImm {
				d365 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d364.Imm.Int() - d363.Imm.Int())}
			} else if d363.Loc == scm.LocImm && d363.Imm.Int() == 0 {
				r192 := ctx.AllocRegExcept(d364.Reg)
				ctx.W.EmitMovRegReg(r192, d364.Reg)
				d365 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r192}
				ctx.BindReg(r192, &d365)
			} else if d364.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d363.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d364.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d363.Reg)
				d365 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d365)
			} else if d363.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d364.Reg)
				ctx.W.EmitMovRegReg(scratch, d364.Reg)
				if d363.Imm.Int() >= -2147483648 && d363.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d363.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d363.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d365 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d365)
			} else {
				r193 := ctx.AllocRegExcept(d364.Reg, d363.Reg)
				ctx.W.EmitMovRegReg(r193, d364.Reg)
				ctx.W.EmitSubInt64(r193, d363.Reg)
				d365 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r193}
				ctx.BindReg(r193, &d365)
			}
			if d365.Loc == scm.LocReg && d364.Loc == scm.LocReg && d365.Reg == d364.Reg {
				ctx.TransferReg(d364.Reg)
				d364.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d363)
			ctx.EnsureDesc(&d362)
			ctx.EnsureDesc(&d365)
			var d366 scm.JITValueDesc
			if d362.Loc == scm.LocImm && d365.Loc == scm.LocImm {
				d366 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d362.Imm.Int()) >> uint64(d365.Imm.Int())))}
			} else if d365.Loc == scm.LocImm {
				r194 := ctx.AllocRegExcept(d362.Reg)
				ctx.W.EmitMovRegReg(r194, d362.Reg)
				ctx.W.EmitShrRegImm8(r194, uint8(d365.Imm.Int()))
				d366 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r194}
				ctx.BindReg(r194, &d366)
			} else {
				{
					shiftSrc := d362.Reg
					r195 := ctx.AllocRegExcept(d362.Reg)
					ctx.W.EmitMovRegReg(r195, d362.Reg)
					shiftSrc = r195
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d365.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d365.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d365.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d366 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d366)
				}
			}
			if d366.Loc == scm.LocReg && d362.Loc == scm.LocReg && d366.Reg == d362.Reg {
				ctx.TransferReg(d362.Reg)
				d362.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d362)
			ctx.FreeDesc(&d365)
			ctx.EnsureDesc(&d342)
			ctx.EnsureDesc(&d366)
			var d367 scm.JITValueDesc
			if d342.Loc == scm.LocImm && d366.Loc == scm.LocImm {
				d367 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d342.Imm.Int() | d366.Imm.Int())}
			} else if d342.Loc == scm.LocImm && d342.Imm.Int() == 0 {
				d367 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d366.Reg}
				ctx.BindReg(d366.Reg, &d367)
			} else if d366.Loc == scm.LocImm && d366.Imm.Int() == 0 {
				r196 := ctx.AllocRegExcept(d342.Reg)
				ctx.W.EmitMovRegReg(r196, d342.Reg)
				d367 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r196}
				ctx.BindReg(r196, &d367)
			} else if d342.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d366.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d342.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d366.Reg)
				d367 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d367)
			} else if d366.Loc == scm.LocImm {
				r197 := ctx.AllocRegExcept(d342.Reg)
				ctx.W.EmitMovRegReg(r197, d342.Reg)
				if d366.Imm.Int() >= -2147483648 && d366.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r197, int32(d366.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d366.Imm.Int()))
					ctx.W.EmitOrInt64(r197, scm.RegR11)
				}
				d367 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r197}
				ctx.BindReg(r197, &d367)
			} else {
				r198 := ctx.AllocRegExcept(d342.Reg, d366.Reg)
				ctx.W.EmitMovRegReg(r198, d342.Reg)
				ctx.W.EmitOrInt64(r198, d366.Reg)
				d367 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r198}
				ctx.BindReg(r198, &d367)
			}
			if d367.Loc == scm.LocReg && d342.Loc == scm.LocReg && d367.Reg == d342.Reg {
				ctx.TransferReg(d342.Reg)
				d342.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d366)
			d368 = d367
			if d368.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d368)
			ctx.EmitStoreToStack(d368, 104)
			ctx.W.EmitJmp(lbl70)
			ctx.W.MarkLabel(lbl68)
			d369 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r181}
			ctx.BindReg(r181, &d369)
			ctx.BindReg(r181, &d369)
			if r161 { ctx.UnprotectReg(r162) }
			ctx.EnsureDesc(&d369)
			ctx.EnsureDesc(&d369)
			var d370 scm.JITValueDesc
			if d369.Loc == scm.LocImm {
				d370 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d369.Imm.Int()))))}
			} else {
				r199 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r199, d369.Reg)
				d370 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r199}
				ctx.BindReg(r199, &d370)
			}
			ctx.FreeDesc(&d369)
			var d371 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d371 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 32)
				r200 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r200, thisptr.Reg, off)
				d371 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r200}
				ctx.BindReg(r200, &d371)
			}
			ctx.EnsureDesc(&d370)
			ctx.EnsureDesc(&d371)
			ctx.EnsureDesc(&d370)
			ctx.EnsureDesc(&d371)
			ctx.EnsureDesc(&d370)
			ctx.EnsureDesc(&d371)
			var d372 scm.JITValueDesc
			if d370.Loc == scm.LocImm && d371.Loc == scm.LocImm {
				d372 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d370.Imm.Int() + d371.Imm.Int())}
			} else if d371.Loc == scm.LocImm && d371.Imm.Int() == 0 {
				r201 := ctx.AllocRegExcept(d370.Reg)
				ctx.W.EmitMovRegReg(r201, d370.Reg)
				d372 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r201}
				ctx.BindReg(r201, &d372)
			} else if d370.Loc == scm.LocImm && d370.Imm.Int() == 0 {
				d372 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d371.Reg}
				ctx.BindReg(d371.Reg, &d372)
			} else if d370.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d371.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d370.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d371.Reg)
				d372 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d372)
			} else if d371.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d370.Reg)
				ctx.W.EmitMovRegReg(scratch, d370.Reg)
				if d371.Imm.Int() >= -2147483648 && d371.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d371.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d371.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d372 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d372)
			} else {
				r202 := ctx.AllocRegExcept(d370.Reg, d371.Reg)
				ctx.W.EmitMovRegReg(r202, d370.Reg)
				ctx.W.EmitAddInt64(r202, d371.Reg)
				d372 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r202}
				ctx.BindReg(r202, &d372)
			}
			if d372.Loc == scm.LocReg && d370.Loc == scm.LocReg && d372.Reg == d370.Reg {
				ctx.TransferReg(d370.Reg)
				d370.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d370)
			ctx.FreeDesc(&d371)
			ctx.EnsureDesc(&d4)
			d373 = d4
			_ = d373
			r203 := d4.Loc == scm.LocReg
			r204 := d4.Reg
			if r203 { ctx.ProtectReg(r204) }
			d374 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(112)}
			lbl76 := ctx.W.ReserveLabel()
			bbpos_5_0 := int32(-1)
			_ = bbpos_5_0
			bbpos_5_1 := int32(-1)
			_ = bbpos_5_1
			bbpos_5_2 := int32(-1)
			_ = bbpos_5_2
			bbpos_5_3 := int32(-1)
			_ = bbpos_5_3
			bbpos_5_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			d374 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(112)}
			ctx.EnsureDesc(&d373)
			ctx.EnsureDesc(&d373)
			var d375 scm.JITValueDesc
			if d373.Loc == scm.LocImm {
				d375 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d373.Imm.Int()))))}
			} else {
				r205 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r205, d373.Reg)
				ctx.W.EmitShlRegImm8(r205, 32)
				ctx.W.EmitShrRegImm8(r205, 32)
				d375 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r205}
				ctx.BindReg(r205, &d375)
			}
			var d376 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d376 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r206 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r206, thisptr.Reg, off)
				d376 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r206}
				ctx.BindReg(r206, &d376)
			}
			ctx.EnsureDesc(&d376)
			ctx.EnsureDesc(&d376)
			var d377 scm.JITValueDesc
			if d376.Loc == scm.LocImm {
				d377 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d376.Imm.Int()))))}
			} else {
				r207 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r207, d376.Reg)
				ctx.W.EmitShlRegImm8(r207, 56)
				ctx.W.EmitShrRegImm8(r207, 56)
				d377 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r207}
				ctx.BindReg(r207, &d377)
			}
			ctx.FreeDesc(&d376)
			ctx.EnsureDesc(&d375)
			ctx.EnsureDesc(&d377)
			ctx.EnsureDesc(&d375)
			ctx.EnsureDesc(&d377)
			ctx.EnsureDesc(&d375)
			ctx.EnsureDesc(&d377)
			var d378 scm.JITValueDesc
			if d375.Loc == scm.LocImm && d377.Loc == scm.LocImm {
				d378 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d375.Imm.Int() * d377.Imm.Int())}
			} else if d375.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d377.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d375.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d377.Reg)
				d378 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d378)
			} else if d377.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d375.Reg)
				ctx.W.EmitMovRegReg(scratch, d375.Reg)
				if d377.Imm.Int() >= -2147483648 && d377.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d377.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d377.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d378 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d378)
			} else {
				r208 := ctx.AllocRegExcept(d375.Reg, d377.Reg)
				ctx.W.EmitMovRegReg(r208, d375.Reg)
				ctx.W.EmitImulInt64(r208, d377.Reg)
				d378 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r208}
				ctx.BindReg(r208, &d378)
			}
			if d378.Loc == scm.LocReg && d375.Loc == scm.LocReg && d378.Reg == d375.Reg {
				ctx.TransferReg(d375.Reg)
				d375.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d375)
			ctx.FreeDesc(&d377)
			var d379 scm.JITValueDesc
			r209 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r209, uint64(dataPtr))
				d379 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r209, StackOff: int32(sliceLen)}
				ctx.BindReg(r209, &d379)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				ctx.W.EmitMovRegMem(r209, thisptr.Reg, off)
				d379 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r209}
				ctx.BindReg(r209, &d379)
			}
			ctx.BindReg(r209, &d379)
			ctx.EnsureDesc(&d378)
			var d380 scm.JITValueDesc
			if d378.Loc == scm.LocImm {
				d380 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d378.Imm.Int() / 64)}
			} else {
				r210 := ctx.AllocRegExcept(d378.Reg)
				ctx.W.EmitMovRegReg(r210, d378.Reg)
				ctx.W.EmitShrRegImm8(r210, 6)
				d380 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r210}
				ctx.BindReg(r210, &d380)
			}
			if d380.Loc == scm.LocReg && d378.Loc == scm.LocReg && d380.Reg == d378.Reg {
				ctx.TransferReg(d378.Reg)
				d378.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d380)
			r211 := ctx.AllocReg()
			ctx.EnsureDesc(&d380)
			ctx.EnsureDesc(&d379)
			if d380.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r211, uint64(d380.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r211, d380.Reg)
				ctx.W.EmitShlRegImm8(r211, 3)
			}
			if d379.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d379.Imm.Int()))
				ctx.W.EmitAddInt64(r211, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r211, d379.Reg)
			}
			r212 := ctx.AllocRegExcept(r211)
			ctx.W.EmitMovRegMem(r212, r211, 0)
			ctx.FreeReg(r211)
			d381 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r212}
			ctx.BindReg(r212, &d381)
			ctx.FreeDesc(&d380)
			ctx.EnsureDesc(&d378)
			var d382 scm.JITValueDesc
			if d378.Loc == scm.LocImm {
				d382 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d378.Imm.Int() % 64)}
			} else {
				r213 := ctx.AllocRegExcept(d378.Reg)
				ctx.W.EmitMovRegReg(r213, d378.Reg)
				ctx.W.EmitAndRegImm32(r213, 63)
				d382 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r213}
				ctx.BindReg(r213, &d382)
			}
			if d382.Loc == scm.LocReg && d378.Loc == scm.LocReg && d382.Reg == d378.Reg {
				ctx.TransferReg(d378.Reg)
				d378.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d381)
			ctx.EnsureDesc(&d382)
			var d383 scm.JITValueDesc
			if d381.Loc == scm.LocImm && d382.Loc == scm.LocImm {
				d383 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d381.Imm.Int()) << uint64(d382.Imm.Int())))}
			} else if d382.Loc == scm.LocImm {
				r214 := ctx.AllocRegExcept(d381.Reg)
				ctx.W.EmitMovRegReg(r214, d381.Reg)
				ctx.W.EmitShlRegImm8(r214, uint8(d382.Imm.Int()))
				d383 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r214}
				ctx.BindReg(r214, &d383)
			} else {
				{
					shiftSrc := d381.Reg
					r215 := ctx.AllocRegExcept(d381.Reg)
					ctx.W.EmitMovRegReg(r215, d381.Reg)
					shiftSrc = r215
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d382.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d382.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d382.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d383 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d383)
				}
			}
			if d383.Loc == scm.LocReg && d381.Loc == scm.LocReg && d383.Reg == d381.Reg {
				ctx.TransferReg(d381.Reg)
				d381.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d381)
			ctx.FreeDesc(&d382)
			var d384 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d384 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
				r216 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r216, thisptr.Reg, off)
				d384 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r216}
				ctx.BindReg(r216, &d384)
			}
			d385 = d384
			ctx.EnsureDesc(&d385)
			if d385.Loc != scm.LocImm && d385.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl77 := ctx.W.ReserveLabel()
			lbl78 := ctx.W.ReserveLabel()
			lbl79 := ctx.W.ReserveLabel()
			lbl80 := ctx.W.ReserveLabel()
			if d385.Loc == scm.LocImm {
				if d385.Imm.Bool() {
					ctx.W.MarkLabel(lbl79)
					ctx.W.EmitJmp(lbl77)
				} else {
					ctx.W.MarkLabel(lbl80)
			d386 = d383
			if d386.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d386)
			ctx.EmitStoreToStack(d386, 112)
					ctx.W.EmitJmp(lbl78)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d385.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl79)
				ctx.W.EmitJmp(lbl80)
				ctx.W.MarkLabel(lbl79)
				ctx.W.EmitJmp(lbl77)
				ctx.W.MarkLabel(lbl80)
			d387 = d383
			if d387.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d387)
			ctx.EmitStoreToStack(d387, 112)
				ctx.W.EmitJmp(lbl78)
			}
			ctx.FreeDesc(&d384)
			bbpos_5_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl78)
			ctx.W.ResolveFixups()
			d374 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(112)}
			var d388 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d388 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r217 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r217, thisptr.Reg, off)
				d388 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r217}
				ctx.BindReg(r217, &d388)
			}
			ctx.EnsureDesc(&d388)
			ctx.EnsureDesc(&d388)
			var d389 scm.JITValueDesc
			if d388.Loc == scm.LocImm {
				d389 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d388.Imm.Int()))))}
			} else {
				r218 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r218, d388.Reg)
				ctx.W.EmitShlRegImm8(r218, 56)
				ctx.W.EmitShrRegImm8(r218, 56)
				d389 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r218}
				ctx.BindReg(r218, &d389)
			}
			ctx.FreeDesc(&d388)
			d390 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d389)
			ctx.EnsureDesc(&d390)
			ctx.EnsureDesc(&d389)
			ctx.EnsureDesc(&d390)
			ctx.EnsureDesc(&d389)
			var d391 scm.JITValueDesc
			if d390.Loc == scm.LocImm && d389.Loc == scm.LocImm {
				d391 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d390.Imm.Int() - d389.Imm.Int())}
			} else if d389.Loc == scm.LocImm && d389.Imm.Int() == 0 {
				r219 := ctx.AllocRegExcept(d390.Reg)
				ctx.W.EmitMovRegReg(r219, d390.Reg)
				d391 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r219}
				ctx.BindReg(r219, &d391)
			} else if d390.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d389.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d390.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d389.Reg)
				d391 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d391)
			} else if d389.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d390.Reg)
				ctx.W.EmitMovRegReg(scratch, d390.Reg)
				if d389.Imm.Int() >= -2147483648 && d389.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d389.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d389.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d391 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d391)
			} else {
				r220 := ctx.AllocRegExcept(d390.Reg, d389.Reg)
				ctx.W.EmitMovRegReg(r220, d390.Reg)
				ctx.W.EmitSubInt64(r220, d389.Reg)
				d391 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r220}
				ctx.BindReg(r220, &d391)
			}
			if d391.Loc == scm.LocReg && d390.Loc == scm.LocReg && d391.Reg == d390.Reg {
				ctx.TransferReg(d390.Reg)
				d390.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d389)
			ctx.EnsureDesc(&d374)
			ctx.EnsureDesc(&d391)
			var d392 scm.JITValueDesc
			if d374.Loc == scm.LocImm && d391.Loc == scm.LocImm {
				d392 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d374.Imm.Int()) >> uint64(d391.Imm.Int())))}
			} else if d391.Loc == scm.LocImm {
				r221 := ctx.AllocRegExcept(d374.Reg)
				ctx.W.EmitMovRegReg(r221, d374.Reg)
				ctx.W.EmitShrRegImm8(r221, uint8(d391.Imm.Int()))
				d392 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r221}
				ctx.BindReg(r221, &d392)
			} else {
				{
					shiftSrc := d374.Reg
					r222 := ctx.AllocRegExcept(d374.Reg)
					ctx.W.EmitMovRegReg(r222, d374.Reg)
					shiftSrc = r222
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d391.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d391.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d391.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d392 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d392)
				}
			}
			if d392.Loc == scm.LocReg && d374.Loc == scm.LocReg && d392.Reg == d374.Reg {
				ctx.TransferReg(d374.Reg)
				d374.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d374)
			ctx.FreeDesc(&d391)
			r223 := ctx.AllocReg()
			ctx.EnsureDesc(&d392)
			ctx.EnsureDesc(&d392)
			if d392.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r223, d392)
			}
			ctx.W.EmitJmp(lbl76)
			bbpos_5_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl77)
			ctx.W.ResolveFixups()
			d374 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(112)}
			ctx.EnsureDesc(&d378)
			var d393 scm.JITValueDesc
			if d378.Loc == scm.LocImm {
				d393 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d378.Imm.Int() % 64)}
			} else {
				r224 := ctx.AllocRegExcept(d378.Reg)
				ctx.W.EmitMovRegReg(r224, d378.Reg)
				ctx.W.EmitAndRegImm32(r224, 63)
				d393 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r224}
				ctx.BindReg(r224, &d393)
			}
			if d393.Loc == scm.LocReg && d378.Loc == scm.LocReg && d393.Reg == d378.Reg {
				ctx.TransferReg(d378.Reg)
				d378.Loc = scm.LocNone
			}
			var d394 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d394 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r225 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r225, thisptr.Reg, off)
				d394 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r225}
				ctx.BindReg(r225, &d394)
			}
			ctx.EnsureDesc(&d394)
			ctx.EnsureDesc(&d394)
			var d395 scm.JITValueDesc
			if d394.Loc == scm.LocImm {
				d395 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d394.Imm.Int()))))}
			} else {
				r226 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r226, d394.Reg)
				ctx.W.EmitShlRegImm8(r226, 56)
				ctx.W.EmitShrRegImm8(r226, 56)
				d395 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r226}
				ctx.BindReg(r226, &d395)
			}
			ctx.FreeDesc(&d394)
			ctx.EnsureDesc(&d393)
			ctx.EnsureDesc(&d395)
			ctx.EnsureDesc(&d393)
			ctx.EnsureDesc(&d395)
			ctx.EnsureDesc(&d393)
			ctx.EnsureDesc(&d395)
			var d396 scm.JITValueDesc
			if d393.Loc == scm.LocImm && d395.Loc == scm.LocImm {
				d396 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d393.Imm.Int() + d395.Imm.Int())}
			} else if d395.Loc == scm.LocImm && d395.Imm.Int() == 0 {
				r227 := ctx.AllocRegExcept(d393.Reg)
				ctx.W.EmitMovRegReg(r227, d393.Reg)
				d396 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r227}
				ctx.BindReg(r227, &d396)
			} else if d393.Loc == scm.LocImm && d393.Imm.Int() == 0 {
				d396 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d395.Reg}
				ctx.BindReg(d395.Reg, &d396)
			} else if d393.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d395.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d393.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d395.Reg)
				d396 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d396)
			} else if d395.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d393.Reg)
				ctx.W.EmitMovRegReg(scratch, d393.Reg)
				if d395.Imm.Int() >= -2147483648 && d395.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d395.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d395.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d396 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d396)
			} else {
				r228 := ctx.AllocRegExcept(d393.Reg, d395.Reg)
				ctx.W.EmitMovRegReg(r228, d393.Reg)
				ctx.W.EmitAddInt64(r228, d395.Reg)
				d396 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r228}
				ctx.BindReg(r228, &d396)
			}
			if d396.Loc == scm.LocReg && d393.Loc == scm.LocReg && d396.Reg == d393.Reg {
				ctx.TransferReg(d393.Reg)
				d393.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d393)
			ctx.FreeDesc(&d395)
			ctx.EnsureDesc(&d396)
			var d397 scm.JITValueDesc
			if d396.Loc == scm.LocImm {
				d397 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d396.Imm.Int()) > uint64(64))}
			} else {
				r229 := ctx.AllocRegExcept(d396.Reg)
				ctx.W.EmitCmpRegImm32(d396.Reg, 64)
				ctx.W.EmitSetcc(r229, scm.CcA)
				d397 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r229}
				ctx.BindReg(r229, &d397)
			}
			ctx.FreeDesc(&d396)
			d398 = d397
			ctx.EnsureDesc(&d398)
			if d398.Loc != scm.LocImm && d398.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl81 := ctx.W.ReserveLabel()
			lbl82 := ctx.W.ReserveLabel()
			lbl83 := ctx.W.ReserveLabel()
			if d398.Loc == scm.LocImm {
				if d398.Imm.Bool() {
					ctx.W.MarkLabel(lbl82)
					ctx.W.EmitJmp(lbl81)
				} else {
					ctx.W.MarkLabel(lbl83)
			d399 = d383
			if d399.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d399)
			ctx.EmitStoreToStack(d399, 112)
					ctx.W.EmitJmp(lbl78)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d398.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl82)
				ctx.W.EmitJmp(lbl83)
				ctx.W.MarkLabel(lbl82)
				ctx.W.EmitJmp(lbl81)
				ctx.W.MarkLabel(lbl83)
			d400 = d383
			if d400.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d400)
			ctx.EmitStoreToStack(d400, 112)
				ctx.W.EmitJmp(lbl78)
			}
			ctx.FreeDesc(&d397)
			bbpos_5_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl81)
			ctx.W.ResolveFixups()
			d374 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(112)}
			ctx.EnsureDesc(&d378)
			var d401 scm.JITValueDesc
			if d378.Loc == scm.LocImm {
				d401 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d378.Imm.Int() / 64)}
			} else {
				r230 := ctx.AllocRegExcept(d378.Reg)
				ctx.W.EmitMovRegReg(r230, d378.Reg)
				ctx.W.EmitShrRegImm8(r230, 6)
				d401 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r230}
				ctx.BindReg(r230, &d401)
			}
			if d401.Loc == scm.LocReg && d378.Loc == scm.LocReg && d401.Reg == d378.Reg {
				ctx.TransferReg(d378.Reg)
				d378.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d401)
			ctx.EnsureDesc(&d401)
			var d402 scm.JITValueDesc
			if d401.Loc == scm.LocImm {
				d402 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d401.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d401.Reg)
				ctx.W.EmitMovRegReg(scratch, d401.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d402 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d402)
			}
			if d402.Loc == scm.LocReg && d401.Loc == scm.LocReg && d402.Reg == d401.Reg {
				ctx.TransferReg(d401.Reg)
				d401.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d401)
			ctx.EnsureDesc(&d402)
			r231 := ctx.AllocReg()
			ctx.EnsureDesc(&d402)
			ctx.EnsureDesc(&d379)
			if d402.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r231, uint64(d402.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r231, d402.Reg)
				ctx.W.EmitShlRegImm8(r231, 3)
			}
			if d379.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d379.Imm.Int()))
				ctx.W.EmitAddInt64(r231, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r231, d379.Reg)
			}
			r232 := ctx.AllocRegExcept(r231)
			ctx.W.EmitMovRegMem(r232, r231, 0)
			ctx.FreeReg(r231)
			d403 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r232}
			ctx.BindReg(r232, &d403)
			ctx.FreeDesc(&d402)
			ctx.EnsureDesc(&d378)
			var d404 scm.JITValueDesc
			if d378.Loc == scm.LocImm {
				d404 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d378.Imm.Int() % 64)}
			} else {
				r233 := ctx.AllocRegExcept(d378.Reg)
				ctx.W.EmitMovRegReg(r233, d378.Reg)
				ctx.W.EmitAndRegImm32(r233, 63)
				d404 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r233}
				ctx.BindReg(r233, &d404)
			}
			if d404.Loc == scm.LocReg && d378.Loc == scm.LocReg && d404.Reg == d378.Reg {
				ctx.TransferReg(d378.Reg)
				d378.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d378)
			d405 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d404)
			ctx.EnsureDesc(&d405)
			ctx.EnsureDesc(&d404)
			ctx.EnsureDesc(&d405)
			ctx.EnsureDesc(&d404)
			var d406 scm.JITValueDesc
			if d405.Loc == scm.LocImm && d404.Loc == scm.LocImm {
				d406 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d405.Imm.Int() - d404.Imm.Int())}
			} else if d404.Loc == scm.LocImm && d404.Imm.Int() == 0 {
				r234 := ctx.AllocRegExcept(d405.Reg)
				ctx.W.EmitMovRegReg(r234, d405.Reg)
				d406 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
				ctx.BindReg(r234, &d406)
			} else if d405.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d404.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d405.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d404.Reg)
				d406 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d406)
			} else if d404.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d405.Reg)
				ctx.W.EmitMovRegReg(scratch, d405.Reg)
				if d404.Imm.Int() >= -2147483648 && d404.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d404.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d404.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d406 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d406)
			} else {
				r235 := ctx.AllocRegExcept(d405.Reg, d404.Reg)
				ctx.W.EmitMovRegReg(r235, d405.Reg)
				ctx.W.EmitSubInt64(r235, d404.Reg)
				d406 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r235}
				ctx.BindReg(r235, &d406)
			}
			if d406.Loc == scm.LocReg && d405.Loc == scm.LocReg && d406.Reg == d405.Reg {
				ctx.TransferReg(d405.Reg)
				d405.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d404)
			ctx.EnsureDesc(&d403)
			ctx.EnsureDesc(&d406)
			var d407 scm.JITValueDesc
			if d403.Loc == scm.LocImm && d406.Loc == scm.LocImm {
				d407 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d403.Imm.Int()) >> uint64(d406.Imm.Int())))}
			} else if d406.Loc == scm.LocImm {
				r236 := ctx.AllocRegExcept(d403.Reg)
				ctx.W.EmitMovRegReg(r236, d403.Reg)
				ctx.W.EmitShrRegImm8(r236, uint8(d406.Imm.Int()))
				d407 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r236}
				ctx.BindReg(r236, &d407)
			} else {
				{
					shiftSrc := d403.Reg
					r237 := ctx.AllocRegExcept(d403.Reg)
					ctx.W.EmitMovRegReg(r237, d403.Reg)
					shiftSrc = r237
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d406.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d406.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d406.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d407 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d407)
				}
			}
			if d407.Loc == scm.LocReg && d403.Loc == scm.LocReg && d407.Reg == d403.Reg {
				ctx.TransferReg(d403.Reg)
				d403.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d403)
			ctx.FreeDesc(&d406)
			ctx.EnsureDesc(&d383)
			ctx.EnsureDesc(&d407)
			var d408 scm.JITValueDesc
			if d383.Loc == scm.LocImm && d407.Loc == scm.LocImm {
				d408 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d383.Imm.Int() | d407.Imm.Int())}
			} else if d383.Loc == scm.LocImm && d383.Imm.Int() == 0 {
				d408 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d407.Reg}
				ctx.BindReg(d407.Reg, &d408)
			} else if d407.Loc == scm.LocImm && d407.Imm.Int() == 0 {
				r238 := ctx.AllocRegExcept(d383.Reg)
				ctx.W.EmitMovRegReg(r238, d383.Reg)
				d408 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r238}
				ctx.BindReg(r238, &d408)
			} else if d383.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d407.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d383.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d407.Reg)
				d408 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d408)
			} else if d407.Loc == scm.LocImm {
				r239 := ctx.AllocRegExcept(d383.Reg)
				ctx.W.EmitMovRegReg(r239, d383.Reg)
				if d407.Imm.Int() >= -2147483648 && d407.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r239, int32(d407.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d407.Imm.Int()))
					ctx.W.EmitOrInt64(r239, scm.RegR11)
				}
				d408 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r239}
				ctx.BindReg(r239, &d408)
			} else {
				r240 := ctx.AllocRegExcept(d383.Reg, d407.Reg)
				ctx.W.EmitMovRegReg(r240, d383.Reg)
				ctx.W.EmitOrInt64(r240, d407.Reg)
				d408 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r240}
				ctx.BindReg(r240, &d408)
			}
			if d408.Loc == scm.LocReg && d383.Loc == scm.LocReg && d408.Reg == d383.Reg {
				ctx.TransferReg(d383.Reg)
				d383.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d407)
			d409 = d408
			if d409.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d409)
			ctx.EmitStoreToStack(d409, 112)
			ctx.W.EmitJmp(lbl78)
			ctx.W.MarkLabel(lbl76)
			d410 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r223}
			ctx.BindReg(r223, &d410)
			ctx.BindReg(r223, &d410)
			if r203 { ctx.UnprotectReg(r204) }
			ctx.FreeDesc(&d4)
			ctx.EnsureDesc(&d410)
			ctx.EnsureDesc(&d410)
			var d411 scm.JITValueDesc
			if d410.Loc == scm.LocImm {
				d411 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d410.Imm.Int()))))}
			} else {
				r241 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r241, d410.Reg)
				d411 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r241}
				ctx.BindReg(r241, &d411)
			}
			ctx.FreeDesc(&d410)
			var d412 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d412 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r242 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r242, thisptr.Reg, off)
				d412 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r242}
				ctx.BindReg(r242, &d412)
			}
			ctx.EnsureDesc(&d411)
			ctx.EnsureDesc(&d412)
			ctx.EnsureDesc(&d411)
			ctx.EnsureDesc(&d412)
			ctx.EnsureDesc(&d411)
			ctx.EnsureDesc(&d412)
			var d413 scm.JITValueDesc
			if d411.Loc == scm.LocImm && d412.Loc == scm.LocImm {
				d413 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d411.Imm.Int() + d412.Imm.Int())}
			} else if d412.Loc == scm.LocImm && d412.Imm.Int() == 0 {
				r243 := ctx.AllocRegExcept(d411.Reg)
				ctx.W.EmitMovRegReg(r243, d411.Reg)
				d413 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r243}
				ctx.BindReg(r243, &d413)
			} else if d411.Loc == scm.LocImm && d411.Imm.Int() == 0 {
				d413 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d412.Reg}
				ctx.BindReg(d412.Reg, &d413)
			} else if d411.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d412.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d411.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d412.Reg)
				d413 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d413)
			} else if d412.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d411.Reg)
				ctx.W.EmitMovRegReg(scratch, d411.Reg)
				if d412.Imm.Int() >= -2147483648 && d412.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d412.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d412.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d413 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d413)
			} else {
				r244 := ctx.AllocRegExcept(d411.Reg, d412.Reg)
				ctx.W.EmitMovRegReg(r244, d411.Reg)
				ctx.W.EmitAddInt64(r244, d412.Reg)
				d413 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r244}
				ctx.BindReg(r244, &d413)
			}
			if d413.Loc == scm.LocReg && d411.Loc == scm.LocReg && d413.Reg == d411.Reg {
				ctx.TransferReg(d411.Reg)
				d411.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d411)
			ctx.FreeDesc(&d412)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&idxInt)
			var d414 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d414 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(idxInt.Imm.Int()))))}
			} else {
				r245 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r245, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r245, 32)
				ctx.W.EmitShrRegImm8(r245, 32)
				d414 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r245}
				ctx.BindReg(r245, &d414)
			}
			ctx.FreeDesc(&idxInt)
			ctx.EnsureDesc(&d414)
			ctx.EnsureDesc(&d413)
			ctx.EnsureDesc(&d414)
			ctx.EnsureDesc(&d413)
			ctx.EnsureDesc(&d414)
			ctx.EnsureDesc(&d413)
			var d415 scm.JITValueDesc
			if d414.Loc == scm.LocImm && d413.Loc == scm.LocImm {
				d415 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d414.Imm.Int() - d413.Imm.Int())}
			} else if d413.Loc == scm.LocImm && d413.Imm.Int() == 0 {
				r246 := ctx.AllocRegExcept(d414.Reg)
				ctx.W.EmitMovRegReg(r246, d414.Reg)
				d415 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r246}
				ctx.BindReg(r246, &d415)
			} else if d414.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d413.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d414.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d413.Reg)
				d415 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d415)
			} else if d413.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d414.Reg)
				ctx.W.EmitMovRegReg(scratch, d414.Reg)
				if d413.Imm.Int() >= -2147483648 && d413.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d413.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d413.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d415 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d415)
			} else {
				r247 := ctx.AllocRegExcept(d414.Reg, d413.Reg)
				ctx.W.EmitMovRegReg(r247, d414.Reg)
				ctx.W.EmitSubInt64(r247, d413.Reg)
				d415 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r247}
				ctx.BindReg(r247, &d415)
			}
			if d415.Loc == scm.LocReg && d414.Loc == scm.LocReg && d415.Reg == d414.Reg {
				ctx.TransferReg(d414.Reg)
				d414.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d414)
			ctx.FreeDesc(&d413)
			ctx.EnsureDesc(&d415)
			ctx.EnsureDesc(&d372)
			ctx.EnsureDesc(&d415)
			ctx.EnsureDesc(&d372)
			ctx.EnsureDesc(&d415)
			ctx.EnsureDesc(&d372)
			var d416 scm.JITValueDesc
			if d415.Loc == scm.LocImm && d372.Loc == scm.LocImm {
				d416 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d415.Imm.Int() * d372.Imm.Int())}
			} else if d415.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d372.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d415.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d372.Reg)
				d416 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d416)
			} else if d372.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d415.Reg)
				ctx.W.EmitMovRegReg(scratch, d415.Reg)
				if d372.Imm.Int() >= -2147483648 && d372.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d372.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d372.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d416 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d416)
			} else {
				r248 := ctx.AllocRegExcept(d415.Reg, d372.Reg)
				ctx.W.EmitMovRegReg(r248, d415.Reg)
				ctx.W.EmitImulInt64(r248, d372.Reg)
				d416 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r248}
				ctx.BindReg(r248, &d416)
			}
			if d416.Loc == scm.LocReg && d415.Loc == scm.LocReg && d416.Reg == d415.Reg {
				ctx.TransferReg(d415.Reg)
				d415.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d415)
			ctx.FreeDesc(&d372)
			ctx.EnsureDesc(&d144)
			ctx.EnsureDesc(&d416)
			ctx.EnsureDesc(&d144)
			ctx.EnsureDesc(&d416)
			ctx.EnsureDesc(&d144)
			ctx.EnsureDesc(&d416)
			var d417 scm.JITValueDesc
			if d144.Loc == scm.LocImm && d416.Loc == scm.LocImm {
				d417 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d144.Imm.Int() + d416.Imm.Int())}
			} else if d416.Loc == scm.LocImm && d416.Imm.Int() == 0 {
				r249 := ctx.AllocRegExcept(d144.Reg)
				ctx.W.EmitMovRegReg(r249, d144.Reg)
				d417 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r249}
				ctx.BindReg(r249, &d417)
			} else if d144.Loc == scm.LocImm && d144.Imm.Int() == 0 {
				d417 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d416.Reg}
				ctx.BindReg(d416.Reg, &d417)
			} else if d144.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d416.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d144.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d416.Reg)
				d417 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d417)
			} else if d416.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d144.Reg)
				ctx.W.EmitMovRegReg(scratch, d144.Reg)
				if d416.Imm.Int() >= -2147483648 && d416.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d416.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d416.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d417 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d417)
			} else {
				r250 := ctx.AllocRegExcept(d144.Reg, d416.Reg)
				ctx.W.EmitMovRegReg(r250, d144.Reg)
				ctx.W.EmitAddInt64(r250, d416.Reg)
				d417 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r250}
				ctx.BindReg(r250, &d417)
			}
			if d417.Loc == scm.LocReg && d144.Loc == scm.LocReg && d417.Reg == d144.Reg {
				ctx.TransferReg(d144.Reg)
				d144.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d416)
			ctx.EnsureDesc(&d417)
			ctx.EnsureDesc(&d417)
			var d418 scm.JITValueDesc
			if d417.Loc == scm.LocImm {
				d418 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d417.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(scm.RegX0, d417.Reg)
				d418 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d417.Reg}
				ctx.BindReg(d417.Reg, &d418)
			}
			ctx.FreeDesc(&d417)
			ctx.EnsureDesc(&d418)
			d419 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d419)
			ctx.BindReg(r2, &d419)
			ctx.EnsureDesc(&d418)
			ctx.W.EmitMakeFloat(d419, d418)
			if d418.Loc == scm.LocReg { ctx.FreeReg(d418.Reg) }
			ctx.W.EmitJmp(lbl0)
			return result
			}
			bbs[22].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[22].VisitCount >= 2 {
					ps.General = true
					return bbs[22].RenderPS(ps)
				}
			}
			bbs[22].VisitCount++
			if ps.General {
				if bbs[22].Rendered {
					ctx.W.EmitJmp(lbl23)
					return result
				}
				bbs[22].Rendered = true
				bbs[22].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_22 = bbs[22].Address
				ctx.W.MarkLabel(lbl23)
				ctx.W.ResolveFixups()
			}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
				d154 = ps.OverlayValues[154]
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
			if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
				d168 = ps.OverlayValues[168]
			}
			if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != scm.LocNone {
				d170 = ps.OverlayValues[170]
			}
			if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != scm.LocNone {
				d171 = ps.OverlayValues[171]
			}
			if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
				d174 = ps.OverlayValues[174]
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
			if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != scm.LocNone {
				d199 = ps.OverlayValues[199]
			}
			if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != scm.LocNone {
				d200 = ps.OverlayValues[200]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != scm.LocNone {
				d201 = ps.OverlayValues[201]
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
			if len(ps.OverlayValues) > 254 && ps.OverlayValues[254].Loc != scm.LocNone {
				d254 = ps.OverlayValues[254]
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
			if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != scm.LocNone {
				d278 = ps.OverlayValues[278]
			}
			if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != scm.LocNone {
				d279 = ps.OverlayValues[279]
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
			if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != scm.LocNone {
				d294 = ps.OverlayValues[294]
			}
			if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != scm.LocNone {
				d295 = ps.OverlayValues[295]
			}
			if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != scm.LocNone {
				d298 = ps.OverlayValues[298]
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
			if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != scm.LocNone {
				d305 = ps.OverlayValues[305]
			}
			if len(ps.OverlayValues) > 307 && ps.OverlayValues[307].Loc != scm.LocNone {
				d307 = ps.OverlayValues[307]
			}
			if len(ps.OverlayValues) > 308 && ps.OverlayValues[308].Loc != scm.LocNone {
				d308 = ps.OverlayValues[308]
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
			if len(ps.OverlayValues) > 313 && ps.OverlayValues[313].Loc != scm.LocNone {
				d313 = ps.OverlayValues[313]
			}
			if len(ps.OverlayValues) > 314 && ps.OverlayValues[314].Loc != scm.LocNone {
				d314 = ps.OverlayValues[314]
			}
			if len(ps.OverlayValues) > 315 && ps.OverlayValues[315].Loc != scm.LocNone {
				d315 = ps.OverlayValues[315]
			}
			if len(ps.OverlayValues) > 317 && ps.OverlayValues[317].Loc != scm.LocNone {
				d317 = ps.OverlayValues[317]
			}
			if len(ps.OverlayValues) > 318 && ps.OverlayValues[318].Loc != scm.LocNone {
				d318 = ps.OverlayValues[318]
			}
			if len(ps.OverlayValues) > 319 && ps.OverlayValues[319].Loc != scm.LocNone {
				d319 = ps.OverlayValues[319]
			}
			if len(ps.OverlayValues) > 320 && ps.OverlayValues[320].Loc != scm.LocNone {
				d320 = ps.OverlayValues[320]
			}
			if len(ps.OverlayValues) > 321 && ps.OverlayValues[321].Loc != scm.LocNone {
				d321 = ps.OverlayValues[321]
			}
			if len(ps.OverlayValues) > 322 && ps.OverlayValues[322].Loc != scm.LocNone {
				d322 = ps.OverlayValues[322]
			}
			if len(ps.OverlayValues) > 323 && ps.OverlayValues[323].Loc != scm.LocNone {
				d323 = ps.OverlayValues[323]
			}
			if len(ps.OverlayValues) > 324 && ps.OverlayValues[324].Loc != scm.LocNone {
				d324 = ps.OverlayValues[324]
			}
			if len(ps.OverlayValues) > 325 && ps.OverlayValues[325].Loc != scm.LocNone {
				d325 = ps.OverlayValues[325]
			}
			if len(ps.OverlayValues) > 326 && ps.OverlayValues[326].Loc != scm.LocNone {
				d326 = ps.OverlayValues[326]
			}
			if len(ps.OverlayValues) > 328 && ps.OverlayValues[328].Loc != scm.LocNone {
				d328 = ps.OverlayValues[328]
			}
			if len(ps.OverlayValues) > 329 && ps.OverlayValues[329].Loc != scm.LocNone {
				d329 = ps.OverlayValues[329]
			}
			if len(ps.OverlayValues) > 330 && ps.OverlayValues[330].Loc != scm.LocNone {
				d330 = ps.OverlayValues[330]
			}
			if len(ps.OverlayValues) > 331 && ps.OverlayValues[331].Loc != scm.LocNone {
				d331 = ps.OverlayValues[331]
			}
			if len(ps.OverlayValues) > 332 && ps.OverlayValues[332].Loc != scm.LocNone {
				d332 = ps.OverlayValues[332]
			}
			if len(ps.OverlayValues) > 333 && ps.OverlayValues[333].Loc != scm.LocNone {
				d333 = ps.OverlayValues[333]
			}
			if len(ps.OverlayValues) > 334 && ps.OverlayValues[334].Loc != scm.LocNone {
				d334 = ps.OverlayValues[334]
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
			if len(ps.OverlayValues) > 339 && ps.OverlayValues[339].Loc != scm.LocNone {
				d339 = ps.OverlayValues[339]
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
			if len(ps.OverlayValues) > 348 && ps.OverlayValues[348].Loc != scm.LocNone {
				d348 = ps.OverlayValues[348]
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
			if len(ps.OverlayValues) > 357 && ps.OverlayValues[357].Loc != scm.LocNone {
				d357 = ps.OverlayValues[357]
			}
			if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != scm.LocNone {
				d358 = ps.OverlayValues[358]
			}
			if len(ps.OverlayValues) > 359 && ps.OverlayValues[359].Loc != scm.LocNone {
				d359 = ps.OverlayValues[359]
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
			if len(ps.OverlayValues) > 363 && ps.OverlayValues[363].Loc != scm.LocNone {
				d363 = ps.OverlayValues[363]
			}
			if len(ps.OverlayValues) > 364 && ps.OverlayValues[364].Loc != scm.LocNone {
				d364 = ps.OverlayValues[364]
			}
			if len(ps.OverlayValues) > 365 && ps.OverlayValues[365].Loc != scm.LocNone {
				d365 = ps.OverlayValues[365]
			}
			if len(ps.OverlayValues) > 366 && ps.OverlayValues[366].Loc != scm.LocNone {
				d366 = ps.OverlayValues[366]
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
			if len(ps.OverlayValues) > 371 && ps.OverlayValues[371].Loc != scm.LocNone {
				d371 = ps.OverlayValues[371]
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
			if len(ps.OverlayValues) > 380 && ps.OverlayValues[380].Loc != scm.LocNone {
				d380 = ps.OverlayValues[380]
			}
			if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != scm.LocNone {
				d381 = ps.OverlayValues[381]
			}
			if len(ps.OverlayValues) > 382 && ps.OverlayValues[382].Loc != scm.LocNone {
				d382 = ps.OverlayValues[382]
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
			if len(ps.OverlayValues) > 390 && ps.OverlayValues[390].Loc != scm.LocNone {
				d390 = ps.OverlayValues[390]
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
			if len(ps.OverlayValues) > 399 && ps.OverlayValues[399].Loc != scm.LocNone {
				d399 = ps.OverlayValues[399]
			}
			if len(ps.OverlayValues) > 400 && ps.OverlayValues[400].Loc != scm.LocNone {
				d400 = ps.OverlayValues[400]
			}
			if len(ps.OverlayValues) > 401 && ps.OverlayValues[401].Loc != scm.LocNone {
				d401 = ps.OverlayValues[401]
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
			if len(ps.OverlayValues) > 405 && ps.OverlayValues[405].Loc != scm.LocNone {
				d405 = ps.OverlayValues[405]
			}
			if len(ps.OverlayValues) > 406 && ps.OverlayValues[406].Loc != scm.LocNone {
				d406 = ps.OverlayValues[406]
			}
			if len(ps.OverlayValues) > 407 && ps.OverlayValues[407].Loc != scm.LocNone {
				d407 = ps.OverlayValues[407]
			}
			if len(ps.OverlayValues) > 408 && ps.OverlayValues[408].Loc != scm.LocNone {
				d408 = ps.OverlayValues[408]
			}
			if len(ps.OverlayValues) > 409 && ps.OverlayValues[409].Loc != scm.LocNone {
				d409 = ps.OverlayValues[409]
			}
			if len(ps.OverlayValues) > 410 && ps.OverlayValues[410].Loc != scm.LocNone {
				d410 = ps.OverlayValues[410]
			}
			if len(ps.OverlayValues) > 411 && ps.OverlayValues[411].Loc != scm.LocNone {
				d411 = ps.OverlayValues[411]
			}
			if len(ps.OverlayValues) > 412 && ps.OverlayValues[412].Loc != scm.LocNone {
				d412 = ps.OverlayValues[412]
			}
			if len(ps.OverlayValues) > 413 && ps.OverlayValues[413].Loc != scm.LocNone {
				d413 = ps.OverlayValues[413]
			}
			if len(ps.OverlayValues) > 414 && ps.OverlayValues[414].Loc != scm.LocNone {
				d414 = ps.OverlayValues[414]
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
			ctx.ReclaimUntrackedRegs()
			var d420 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d420 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 64)
				r251 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r251, thisptr.Reg, off)
				d420 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r251}
				ctx.BindReg(r251, &d420)
			}
			ctx.EnsureDesc(&d420)
			ctx.EnsureDesc(&d420)
			var d421 scm.JITValueDesc
			if d420.Loc == scm.LocImm {
				d421 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d420.Imm.Int()))))}
			} else {
				r252 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r252, d420.Reg)
				d421 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r252}
				ctx.BindReg(r252, &d421)
			}
			ctx.FreeDesc(&d420)
			ctx.EnsureDesc(&d144)
			ctx.EnsureDesc(&d421)
			ctx.EnsureDesc(&d144)
			ctx.EnsureDesc(&d421)
			ctx.EnsureDesc(&d144)
			ctx.EnsureDesc(&d421)
			var d422 scm.JITValueDesc
			if d144.Loc == scm.LocImm && d421.Loc == scm.LocImm {
				d422 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d144.Imm.Int() == d421.Imm.Int())}
			} else if d421.Loc == scm.LocImm {
				r253 := ctx.AllocRegExcept(d144.Reg)
				if d421.Imm.Int() >= -2147483648 && d421.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d144.Reg, int32(d421.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d421.Imm.Int()))
					ctx.W.EmitCmpInt64(d144.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r253, scm.CcE)
				d422 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r253}
				ctx.BindReg(r253, &d422)
			} else if d144.Loc == scm.LocImm {
				r254 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d144.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d421.Reg)
				ctx.W.EmitSetcc(r254, scm.CcE)
				d422 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r254}
				ctx.BindReg(r254, &d422)
			} else {
				r255 := ctx.AllocRegExcept(d144.Reg)
				ctx.W.EmitCmpInt64(d144.Reg, d421.Reg)
				ctx.W.EmitSetcc(r255, scm.CcE)
				d422 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r255}
				ctx.BindReg(r255, &d422)
			}
			ctx.FreeDesc(&d144)
			ctx.FreeDesc(&d421)
			d423 = d422
			ctx.EnsureDesc(&d423)
			if d423.Loc != scm.LocImm && d423.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d423.Loc == scm.LocImm {
				if d423.Imm.Bool() {
			ps424 := scm.PhiState{General: ps.General}
			ps424.OverlayValues = make([]scm.JITValueDesc, 424)
			ps424.OverlayValues[0] = d0
			ps424.OverlayValues[1] = d1
			ps424.OverlayValues[2] = d2
			ps424.OverlayValues[3] = d3
			ps424.OverlayValues[4] = d4
			ps424.OverlayValues[5] = d5
			ps424.OverlayValues[6] = d6
			ps424.OverlayValues[7] = d7
			ps424.OverlayValues[8] = d8
			ps424.OverlayValues[9] = d9
			ps424.OverlayValues[10] = d10
			ps424.OverlayValues[11] = d11
			ps424.OverlayValues[12] = d12
			ps424.OverlayValues[13] = d13
			ps424.OverlayValues[14] = d14
			ps424.OverlayValues[20] = d20
			ps424.OverlayValues[21] = d21
			ps424.OverlayValues[22] = d22
			ps424.OverlayValues[24] = d24
			ps424.OverlayValues[25] = d25
			ps424.OverlayValues[27] = d27
			ps424.OverlayValues[28] = d28
			ps424.OverlayValues[29] = d29
			ps424.OverlayValues[32] = d32
			ps424.OverlayValues[34] = d34
			ps424.OverlayValues[35] = d35
			ps424.OverlayValues[36] = d36
			ps424.OverlayValues[38] = d38
			ps424.OverlayValues[39] = d39
			ps424.OverlayValues[40] = d40
			ps424.OverlayValues[41] = d41
			ps424.OverlayValues[42] = d42
			ps424.OverlayValues[43] = d43
			ps424.OverlayValues[44] = d44
			ps424.OverlayValues[46] = d46
			ps424.OverlayValues[47] = d47
			ps424.OverlayValues[48] = d48
			ps424.OverlayValues[49] = d49
			ps424.OverlayValues[50] = d50
			ps424.OverlayValues[51] = d51
			ps424.OverlayValues[52] = d52
			ps424.OverlayValues[53] = d53
			ps424.OverlayValues[54] = d54
			ps424.OverlayValues[55] = d55
			ps424.OverlayValues[56] = d56
			ps424.OverlayValues[57] = d57
			ps424.OverlayValues[58] = d58
			ps424.OverlayValues[59] = d59
			ps424.OverlayValues[60] = d60
			ps424.OverlayValues[61] = d61
			ps424.OverlayValues[62] = d62
			ps424.OverlayValues[63] = d63
			ps424.OverlayValues[64] = d64
			ps424.OverlayValues[65] = d65
			ps424.OverlayValues[66] = d66
			ps424.OverlayValues[67] = d67
			ps424.OverlayValues[68] = d68
			ps424.OverlayValues[69] = d69
			ps424.OverlayValues[70] = d70
			ps424.OverlayValues[71] = d71
			ps424.OverlayValues[72] = d72
			ps424.OverlayValues[73] = d73
			ps424.OverlayValues[74] = d74
			ps424.OverlayValues[75] = d75
			ps424.OverlayValues[76] = d76
			ps424.OverlayValues[77] = d77
			ps424.OverlayValues[78] = d78
			ps424.OverlayValues[79] = d79
			ps424.OverlayValues[80] = d80
			ps424.OverlayValues[81] = d81
			ps424.OverlayValues[82] = d82
			ps424.OverlayValues[83] = d83
			ps424.OverlayValues[84] = d84
			ps424.OverlayValues[85] = d85
			ps424.OverlayValues[86] = d86
			ps424.OverlayValues[87] = d87
			ps424.OverlayValues[88] = d88
			ps424.OverlayValues[89] = d89
			ps424.OverlayValues[90] = d90
			ps424.OverlayValues[91] = d91
			ps424.OverlayValues[92] = d92
			ps424.OverlayValues[93] = d93
			ps424.OverlayValues[94] = d94
			ps424.OverlayValues[95] = d95
			ps424.OverlayValues[102] = d102
			ps424.OverlayValues[103] = d103
			ps424.OverlayValues[104] = d104
			ps424.OverlayValues[105] = d105
			ps424.OverlayValues[106] = d106
			ps424.OverlayValues[107] = d107
			ps424.OverlayValues[108] = d108
			ps424.OverlayValues[109] = d109
			ps424.OverlayValues[110] = d110
			ps424.OverlayValues[111] = d111
			ps424.OverlayValues[112] = d112
			ps424.OverlayValues[113] = d113
			ps424.OverlayValues[114] = d114
			ps424.OverlayValues[115] = d115
			ps424.OverlayValues[116] = d116
			ps424.OverlayValues[117] = d117
			ps424.OverlayValues[118] = d118
			ps424.OverlayValues[119] = d119
			ps424.OverlayValues[120] = d120
			ps424.OverlayValues[121] = d121
			ps424.OverlayValues[122] = d122
			ps424.OverlayValues[123] = d123
			ps424.OverlayValues[124] = d124
			ps424.OverlayValues[125] = d125
			ps424.OverlayValues[126] = d126
			ps424.OverlayValues[127] = d127
			ps424.OverlayValues[128] = d128
			ps424.OverlayValues[129] = d129
			ps424.OverlayValues[130] = d130
			ps424.OverlayValues[131] = d131
			ps424.OverlayValues[132] = d132
			ps424.OverlayValues[133] = d133
			ps424.OverlayValues[134] = d134
			ps424.OverlayValues[135] = d135
			ps424.OverlayValues[136] = d136
			ps424.OverlayValues[137] = d137
			ps424.OverlayValues[138] = d138
			ps424.OverlayValues[139] = d139
			ps424.OverlayValues[140] = d140
			ps424.OverlayValues[141] = d141
			ps424.OverlayValues[142] = d142
			ps424.OverlayValues[143] = d143
			ps424.OverlayValues[144] = d144
			ps424.OverlayValues[145] = d145
			ps424.OverlayValues[146] = d146
			ps424.OverlayValues[153] = d153
			ps424.OverlayValues[154] = d154
			ps424.OverlayValues[160] = d160
			ps424.OverlayValues[161] = d161
			ps424.OverlayValues[162] = d162
			ps424.OverlayValues[163] = d163
			ps424.OverlayValues[164] = d164
			ps424.OverlayValues[165] = d165
			ps424.OverlayValues[166] = d166
			ps424.OverlayValues[168] = d168
			ps424.OverlayValues[170] = d170
			ps424.OverlayValues[171] = d171
			ps424.OverlayValues[174] = d174
			ps424.OverlayValues[177] = d177
			ps424.OverlayValues[178] = d178
			ps424.OverlayValues[179] = d179
			ps424.OverlayValues[181] = d181
			ps424.OverlayValues[182] = d182
			ps424.OverlayValues[183] = d183
			ps424.OverlayValues[184] = d184
			ps424.OverlayValues[185] = d185
			ps424.OverlayValues[186] = d186
			ps424.OverlayValues[188] = d188
			ps424.OverlayValues[189] = d189
			ps424.OverlayValues[190] = d190
			ps424.OverlayValues[191] = d191
			ps424.OverlayValues[192] = d192
			ps424.OverlayValues[193] = d193
			ps424.OverlayValues[194] = d194
			ps424.OverlayValues[195] = d195
			ps424.OverlayValues[196] = d196
			ps424.OverlayValues[199] = d199
			ps424.OverlayValues[200] = d200
			ps424.OverlayValues[201] = d201
			ps424.OverlayValues[204] = d204
			ps424.OverlayValues[205] = d205
			ps424.OverlayValues[206] = d206
			ps424.OverlayValues[207] = d207
			ps424.OverlayValues[208] = d208
			ps424.OverlayValues[209] = d209
			ps424.OverlayValues[210] = d210
			ps424.OverlayValues[211] = d211
			ps424.OverlayValues[212] = d212
			ps424.OverlayValues[214] = d214
			ps424.OverlayValues[215] = d215
			ps424.OverlayValues[216] = d216
			ps424.OverlayValues[217] = d217
			ps424.OverlayValues[218] = d218
			ps424.OverlayValues[219] = d219
			ps424.OverlayValues[220] = d220
			ps424.OverlayValues[221] = d221
			ps424.OverlayValues[222] = d222
			ps424.OverlayValues[223] = d223
			ps424.OverlayValues[225] = d225
			ps424.OverlayValues[226] = d226
			ps424.OverlayValues[227] = d227
			ps424.OverlayValues[228] = d228
			ps424.OverlayValues[229] = d229
			ps424.OverlayValues[230] = d230
			ps424.OverlayValues[231] = d231
			ps424.OverlayValues[232] = d232
			ps424.OverlayValues[233] = d233
			ps424.OverlayValues[234] = d234
			ps424.OverlayValues[235] = d235
			ps424.OverlayValues[236] = d236
			ps424.OverlayValues[237] = d237
			ps424.OverlayValues[238] = d238
			ps424.OverlayValues[239] = d239
			ps424.OverlayValues[240] = d240
			ps424.OverlayValues[241] = d241
			ps424.OverlayValues[242] = d242
			ps424.OverlayValues[243] = d243
			ps424.OverlayValues[244] = d244
			ps424.OverlayValues[245] = d245
			ps424.OverlayValues[246] = d246
			ps424.OverlayValues[247] = d247
			ps424.OverlayValues[248] = d248
			ps424.OverlayValues[249] = d249
			ps424.OverlayValues[250] = d250
			ps424.OverlayValues[251] = d251
			ps424.OverlayValues[252] = d252
			ps424.OverlayValues[253] = d253
			ps424.OverlayValues[254] = d254
			ps424.OverlayValues[255] = d255
			ps424.OverlayValues[256] = d256
			ps424.OverlayValues[257] = d257
			ps424.OverlayValues[258] = d258
			ps424.OverlayValues[259] = d259
			ps424.OverlayValues[260] = d260
			ps424.OverlayValues[261] = d261
			ps424.OverlayValues[262] = d262
			ps424.OverlayValues[263] = d263
			ps424.OverlayValues[264] = d264
			ps424.OverlayValues[265] = d265
			ps424.OverlayValues[266] = d266
			ps424.OverlayValues[267] = d267
			ps424.OverlayValues[268] = d268
			ps424.OverlayValues[269] = d269
			ps424.OverlayValues[270] = d270
			ps424.OverlayValues[271] = d271
			ps424.OverlayValues[278] = d278
			ps424.OverlayValues[279] = d279
			ps424.OverlayValues[285] = d285
			ps424.OverlayValues[286] = d286
			ps424.OverlayValues[287] = d287
			ps424.OverlayValues[288] = d288
			ps424.OverlayValues[289] = d289
			ps424.OverlayValues[290] = d290
			ps424.OverlayValues[292] = d292
			ps424.OverlayValues[294] = d294
			ps424.OverlayValues[295] = d295
			ps424.OverlayValues[298] = d298
			ps424.OverlayValues[302] = d302
			ps424.OverlayValues[303] = d303
			ps424.OverlayValues[304] = d304
			ps424.OverlayValues[305] = d305
			ps424.OverlayValues[307] = d307
			ps424.OverlayValues[308] = d308
			ps424.OverlayValues[310] = d310
			ps424.OverlayValues[311] = d311
			ps424.OverlayValues[312] = d312
			ps424.OverlayValues[313] = d313
			ps424.OverlayValues[314] = d314
			ps424.OverlayValues[315] = d315
			ps424.OverlayValues[317] = d317
			ps424.OverlayValues[318] = d318
			ps424.OverlayValues[319] = d319
			ps424.OverlayValues[320] = d320
			ps424.OverlayValues[321] = d321
			ps424.OverlayValues[322] = d322
			ps424.OverlayValues[323] = d323
			ps424.OverlayValues[324] = d324
			ps424.OverlayValues[325] = d325
			ps424.OverlayValues[326] = d326
			ps424.OverlayValues[328] = d328
			ps424.OverlayValues[329] = d329
			ps424.OverlayValues[330] = d330
			ps424.OverlayValues[331] = d331
			ps424.OverlayValues[332] = d332
			ps424.OverlayValues[333] = d333
			ps424.OverlayValues[334] = d334
			ps424.OverlayValues[335] = d335
			ps424.OverlayValues[336] = d336
			ps424.OverlayValues[337] = d337
			ps424.OverlayValues[338] = d338
			ps424.OverlayValues[339] = d339
			ps424.OverlayValues[340] = d340
			ps424.OverlayValues[341] = d341
			ps424.OverlayValues[342] = d342
			ps424.OverlayValues[343] = d343
			ps424.OverlayValues[344] = d344
			ps424.OverlayValues[345] = d345
			ps424.OverlayValues[346] = d346
			ps424.OverlayValues[347] = d347
			ps424.OverlayValues[348] = d348
			ps424.OverlayValues[349] = d349
			ps424.OverlayValues[350] = d350
			ps424.OverlayValues[351] = d351
			ps424.OverlayValues[352] = d352
			ps424.OverlayValues[353] = d353
			ps424.OverlayValues[354] = d354
			ps424.OverlayValues[355] = d355
			ps424.OverlayValues[356] = d356
			ps424.OverlayValues[357] = d357
			ps424.OverlayValues[358] = d358
			ps424.OverlayValues[359] = d359
			ps424.OverlayValues[360] = d360
			ps424.OverlayValues[361] = d361
			ps424.OverlayValues[362] = d362
			ps424.OverlayValues[363] = d363
			ps424.OverlayValues[364] = d364
			ps424.OverlayValues[365] = d365
			ps424.OverlayValues[366] = d366
			ps424.OverlayValues[367] = d367
			ps424.OverlayValues[368] = d368
			ps424.OverlayValues[369] = d369
			ps424.OverlayValues[370] = d370
			ps424.OverlayValues[371] = d371
			ps424.OverlayValues[372] = d372
			ps424.OverlayValues[373] = d373
			ps424.OverlayValues[374] = d374
			ps424.OverlayValues[375] = d375
			ps424.OverlayValues[376] = d376
			ps424.OverlayValues[377] = d377
			ps424.OverlayValues[378] = d378
			ps424.OverlayValues[379] = d379
			ps424.OverlayValues[380] = d380
			ps424.OverlayValues[381] = d381
			ps424.OverlayValues[382] = d382
			ps424.OverlayValues[383] = d383
			ps424.OverlayValues[384] = d384
			ps424.OverlayValues[385] = d385
			ps424.OverlayValues[386] = d386
			ps424.OverlayValues[387] = d387
			ps424.OverlayValues[388] = d388
			ps424.OverlayValues[389] = d389
			ps424.OverlayValues[390] = d390
			ps424.OverlayValues[391] = d391
			ps424.OverlayValues[392] = d392
			ps424.OverlayValues[393] = d393
			ps424.OverlayValues[394] = d394
			ps424.OverlayValues[395] = d395
			ps424.OverlayValues[396] = d396
			ps424.OverlayValues[397] = d397
			ps424.OverlayValues[398] = d398
			ps424.OverlayValues[399] = d399
			ps424.OverlayValues[400] = d400
			ps424.OverlayValues[401] = d401
			ps424.OverlayValues[402] = d402
			ps424.OverlayValues[403] = d403
			ps424.OverlayValues[404] = d404
			ps424.OverlayValues[405] = d405
			ps424.OverlayValues[406] = d406
			ps424.OverlayValues[407] = d407
			ps424.OverlayValues[408] = d408
			ps424.OverlayValues[409] = d409
			ps424.OverlayValues[410] = d410
			ps424.OverlayValues[411] = d411
			ps424.OverlayValues[412] = d412
			ps424.OverlayValues[413] = d413
			ps424.OverlayValues[414] = d414
			ps424.OverlayValues[415] = d415
			ps424.OverlayValues[416] = d416
			ps424.OverlayValues[417] = d417
			ps424.OverlayValues[418] = d418
			ps424.OverlayValues[419] = d419
			ps424.OverlayValues[420] = d420
			ps424.OverlayValues[421] = d421
			ps424.OverlayValues[422] = d422
			ps424.OverlayValues[423] = d423
					return bbs[20].RenderPS(ps424)
				}
			ps425 := scm.PhiState{General: ps.General}
			ps425.OverlayValues = make([]scm.JITValueDesc, 424)
			ps425.OverlayValues[0] = d0
			ps425.OverlayValues[1] = d1
			ps425.OverlayValues[2] = d2
			ps425.OverlayValues[3] = d3
			ps425.OverlayValues[4] = d4
			ps425.OverlayValues[5] = d5
			ps425.OverlayValues[6] = d6
			ps425.OverlayValues[7] = d7
			ps425.OverlayValues[8] = d8
			ps425.OverlayValues[9] = d9
			ps425.OverlayValues[10] = d10
			ps425.OverlayValues[11] = d11
			ps425.OverlayValues[12] = d12
			ps425.OverlayValues[13] = d13
			ps425.OverlayValues[14] = d14
			ps425.OverlayValues[20] = d20
			ps425.OverlayValues[21] = d21
			ps425.OverlayValues[22] = d22
			ps425.OverlayValues[24] = d24
			ps425.OverlayValues[25] = d25
			ps425.OverlayValues[27] = d27
			ps425.OverlayValues[28] = d28
			ps425.OverlayValues[29] = d29
			ps425.OverlayValues[32] = d32
			ps425.OverlayValues[34] = d34
			ps425.OverlayValues[35] = d35
			ps425.OverlayValues[36] = d36
			ps425.OverlayValues[38] = d38
			ps425.OverlayValues[39] = d39
			ps425.OverlayValues[40] = d40
			ps425.OverlayValues[41] = d41
			ps425.OverlayValues[42] = d42
			ps425.OverlayValues[43] = d43
			ps425.OverlayValues[44] = d44
			ps425.OverlayValues[46] = d46
			ps425.OverlayValues[47] = d47
			ps425.OverlayValues[48] = d48
			ps425.OverlayValues[49] = d49
			ps425.OverlayValues[50] = d50
			ps425.OverlayValues[51] = d51
			ps425.OverlayValues[52] = d52
			ps425.OverlayValues[53] = d53
			ps425.OverlayValues[54] = d54
			ps425.OverlayValues[55] = d55
			ps425.OverlayValues[56] = d56
			ps425.OverlayValues[57] = d57
			ps425.OverlayValues[58] = d58
			ps425.OverlayValues[59] = d59
			ps425.OverlayValues[60] = d60
			ps425.OverlayValues[61] = d61
			ps425.OverlayValues[62] = d62
			ps425.OverlayValues[63] = d63
			ps425.OverlayValues[64] = d64
			ps425.OverlayValues[65] = d65
			ps425.OverlayValues[66] = d66
			ps425.OverlayValues[67] = d67
			ps425.OverlayValues[68] = d68
			ps425.OverlayValues[69] = d69
			ps425.OverlayValues[70] = d70
			ps425.OverlayValues[71] = d71
			ps425.OverlayValues[72] = d72
			ps425.OverlayValues[73] = d73
			ps425.OverlayValues[74] = d74
			ps425.OverlayValues[75] = d75
			ps425.OverlayValues[76] = d76
			ps425.OverlayValues[77] = d77
			ps425.OverlayValues[78] = d78
			ps425.OverlayValues[79] = d79
			ps425.OverlayValues[80] = d80
			ps425.OverlayValues[81] = d81
			ps425.OverlayValues[82] = d82
			ps425.OverlayValues[83] = d83
			ps425.OverlayValues[84] = d84
			ps425.OverlayValues[85] = d85
			ps425.OverlayValues[86] = d86
			ps425.OverlayValues[87] = d87
			ps425.OverlayValues[88] = d88
			ps425.OverlayValues[89] = d89
			ps425.OverlayValues[90] = d90
			ps425.OverlayValues[91] = d91
			ps425.OverlayValues[92] = d92
			ps425.OverlayValues[93] = d93
			ps425.OverlayValues[94] = d94
			ps425.OverlayValues[95] = d95
			ps425.OverlayValues[102] = d102
			ps425.OverlayValues[103] = d103
			ps425.OverlayValues[104] = d104
			ps425.OverlayValues[105] = d105
			ps425.OverlayValues[106] = d106
			ps425.OverlayValues[107] = d107
			ps425.OverlayValues[108] = d108
			ps425.OverlayValues[109] = d109
			ps425.OverlayValues[110] = d110
			ps425.OverlayValues[111] = d111
			ps425.OverlayValues[112] = d112
			ps425.OverlayValues[113] = d113
			ps425.OverlayValues[114] = d114
			ps425.OverlayValues[115] = d115
			ps425.OverlayValues[116] = d116
			ps425.OverlayValues[117] = d117
			ps425.OverlayValues[118] = d118
			ps425.OverlayValues[119] = d119
			ps425.OverlayValues[120] = d120
			ps425.OverlayValues[121] = d121
			ps425.OverlayValues[122] = d122
			ps425.OverlayValues[123] = d123
			ps425.OverlayValues[124] = d124
			ps425.OverlayValues[125] = d125
			ps425.OverlayValues[126] = d126
			ps425.OverlayValues[127] = d127
			ps425.OverlayValues[128] = d128
			ps425.OverlayValues[129] = d129
			ps425.OverlayValues[130] = d130
			ps425.OverlayValues[131] = d131
			ps425.OverlayValues[132] = d132
			ps425.OverlayValues[133] = d133
			ps425.OverlayValues[134] = d134
			ps425.OverlayValues[135] = d135
			ps425.OverlayValues[136] = d136
			ps425.OverlayValues[137] = d137
			ps425.OverlayValues[138] = d138
			ps425.OverlayValues[139] = d139
			ps425.OverlayValues[140] = d140
			ps425.OverlayValues[141] = d141
			ps425.OverlayValues[142] = d142
			ps425.OverlayValues[143] = d143
			ps425.OverlayValues[144] = d144
			ps425.OverlayValues[145] = d145
			ps425.OverlayValues[146] = d146
			ps425.OverlayValues[153] = d153
			ps425.OverlayValues[154] = d154
			ps425.OverlayValues[160] = d160
			ps425.OverlayValues[161] = d161
			ps425.OverlayValues[162] = d162
			ps425.OverlayValues[163] = d163
			ps425.OverlayValues[164] = d164
			ps425.OverlayValues[165] = d165
			ps425.OverlayValues[166] = d166
			ps425.OverlayValues[168] = d168
			ps425.OverlayValues[170] = d170
			ps425.OverlayValues[171] = d171
			ps425.OverlayValues[174] = d174
			ps425.OverlayValues[177] = d177
			ps425.OverlayValues[178] = d178
			ps425.OverlayValues[179] = d179
			ps425.OverlayValues[181] = d181
			ps425.OverlayValues[182] = d182
			ps425.OverlayValues[183] = d183
			ps425.OverlayValues[184] = d184
			ps425.OverlayValues[185] = d185
			ps425.OverlayValues[186] = d186
			ps425.OverlayValues[188] = d188
			ps425.OverlayValues[189] = d189
			ps425.OverlayValues[190] = d190
			ps425.OverlayValues[191] = d191
			ps425.OverlayValues[192] = d192
			ps425.OverlayValues[193] = d193
			ps425.OverlayValues[194] = d194
			ps425.OverlayValues[195] = d195
			ps425.OverlayValues[196] = d196
			ps425.OverlayValues[199] = d199
			ps425.OverlayValues[200] = d200
			ps425.OverlayValues[201] = d201
			ps425.OverlayValues[204] = d204
			ps425.OverlayValues[205] = d205
			ps425.OverlayValues[206] = d206
			ps425.OverlayValues[207] = d207
			ps425.OverlayValues[208] = d208
			ps425.OverlayValues[209] = d209
			ps425.OverlayValues[210] = d210
			ps425.OverlayValues[211] = d211
			ps425.OverlayValues[212] = d212
			ps425.OverlayValues[214] = d214
			ps425.OverlayValues[215] = d215
			ps425.OverlayValues[216] = d216
			ps425.OverlayValues[217] = d217
			ps425.OverlayValues[218] = d218
			ps425.OverlayValues[219] = d219
			ps425.OverlayValues[220] = d220
			ps425.OverlayValues[221] = d221
			ps425.OverlayValues[222] = d222
			ps425.OverlayValues[223] = d223
			ps425.OverlayValues[225] = d225
			ps425.OverlayValues[226] = d226
			ps425.OverlayValues[227] = d227
			ps425.OverlayValues[228] = d228
			ps425.OverlayValues[229] = d229
			ps425.OverlayValues[230] = d230
			ps425.OverlayValues[231] = d231
			ps425.OverlayValues[232] = d232
			ps425.OverlayValues[233] = d233
			ps425.OverlayValues[234] = d234
			ps425.OverlayValues[235] = d235
			ps425.OverlayValues[236] = d236
			ps425.OverlayValues[237] = d237
			ps425.OverlayValues[238] = d238
			ps425.OverlayValues[239] = d239
			ps425.OverlayValues[240] = d240
			ps425.OverlayValues[241] = d241
			ps425.OverlayValues[242] = d242
			ps425.OverlayValues[243] = d243
			ps425.OverlayValues[244] = d244
			ps425.OverlayValues[245] = d245
			ps425.OverlayValues[246] = d246
			ps425.OverlayValues[247] = d247
			ps425.OverlayValues[248] = d248
			ps425.OverlayValues[249] = d249
			ps425.OverlayValues[250] = d250
			ps425.OverlayValues[251] = d251
			ps425.OverlayValues[252] = d252
			ps425.OverlayValues[253] = d253
			ps425.OverlayValues[254] = d254
			ps425.OverlayValues[255] = d255
			ps425.OverlayValues[256] = d256
			ps425.OverlayValues[257] = d257
			ps425.OverlayValues[258] = d258
			ps425.OverlayValues[259] = d259
			ps425.OverlayValues[260] = d260
			ps425.OverlayValues[261] = d261
			ps425.OverlayValues[262] = d262
			ps425.OverlayValues[263] = d263
			ps425.OverlayValues[264] = d264
			ps425.OverlayValues[265] = d265
			ps425.OverlayValues[266] = d266
			ps425.OverlayValues[267] = d267
			ps425.OverlayValues[268] = d268
			ps425.OverlayValues[269] = d269
			ps425.OverlayValues[270] = d270
			ps425.OverlayValues[271] = d271
			ps425.OverlayValues[278] = d278
			ps425.OverlayValues[279] = d279
			ps425.OverlayValues[285] = d285
			ps425.OverlayValues[286] = d286
			ps425.OverlayValues[287] = d287
			ps425.OverlayValues[288] = d288
			ps425.OverlayValues[289] = d289
			ps425.OverlayValues[290] = d290
			ps425.OverlayValues[292] = d292
			ps425.OverlayValues[294] = d294
			ps425.OverlayValues[295] = d295
			ps425.OverlayValues[298] = d298
			ps425.OverlayValues[302] = d302
			ps425.OverlayValues[303] = d303
			ps425.OverlayValues[304] = d304
			ps425.OverlayValues[305] = d305
			ps425.OverlayValues[307] = d307
			ps425.OverlayValues[308] = d308
			ps425.OverlayValues[310] = d310
			ps425.OverlayValues[311] = d311
			ps425.OverlayValues[312] = d312
			ps425.OverlayValues[313] = d313
			ps425.OverlayValues[314] = d314
			ps425.OverlayValues[315] = d315
			ps425.OverlayValues[317] = d317
			ps425.OverlayValues[318] = d318
			ps425.OverlayValues[319] = d319
			ps425.OverlayValues[320] = d320
			ps425.OverlayValues[321] = d321
			ps425.OverlayValues[322] = d322
			ps425.OverlayValues[323] = d323
			ps425.OverlayValues[324] = d324
			ps425.OverlayValues[325] = d325
			ps425.OverlayValues[326] = d326
			ps425.OverlayValues[328] = d328
			ps425.OverlayValues[329] = d329
			ps425.OverlayValues[330] = d330
			ps425.OverlayValues[331] = d331
			ps425.OverlayValues[332] = d332
			ps425.OverlayValues[333] = d333
			ps425.OverlayValues[334] = d334
			ps425.OverlayValues[335] = d335
			ps425.OverlayValues[336] = d336
			ps425.OverlayValues[337] = d337
			ps425.OverlayValues[338] = d338
			ps425.OverlayValues[339] = d339
			ps425.OverlayValues[340] = d340
			ps425.OverlayValues[341] = d341
			ps425.OverlayValues[342] = d342
			ps425.OverlayValues[343] = d343
			ps425.OverlayValues[344] = d344
			ps425.OverlayValues[345] = d345
			ps425.OverlayValues[346] = d346
			ps425.OverlayValues[347] = d347
			ps425.OverlayValues[348] = d348
			ps425.OverlayValues[349] = d349
			ps425.OverlayValues[350] = d350
			ps425.OverlayValues[351] = d351
			ps425.OverlayValues[352] = d352
			ps425.OverlayValues[353] = d353
			ps425.OverlayValues[354] = d354
			ps425.OverlayValues[355] = d355
			ps425.OverlayValues[356] = d356
			ps425.OverlayValues[357] = d357
			ps425.OverlayValues[358] = d358
			ps425.OverlayValues[359] = d359
			ps425.OverlayValues[360] = d360
			ps425.OverlayValues[361] = d361
			ps425.OverlayValues[362] = d362
			ps425.OverlayValues[363] = d363
			ps425.OverlayValues[364] = d364
			ps425.OverlayValues[365] = d365
			ps425.OverlayValues[366] = d366
			ps425.OverlayValues[367] = d367
			ps425.OverlayValues[368] = d368
			ps425.OverlayValues[369] = d369
			ps425.OverlayValues[370] = d370
			ps425.OverlayValues[371] = d371
			ps425.OverlayValues[372] = d372
			ps425.OverlayValues[373] = d373
			ps425.OverlayValues[374] = d374
			ps425.OverlayValues[375] = d375
			ps425.OverlayValues[376] = d376
			ps425.OverlayValues[377] = d377
			ps425.OverlayValues[378] = d378
			ps425.OverlayValues[379] = d379
			ps425.OverlayValues[380] = d380
			ps425.OverlayValues[381] = d381
			ps425.OverlayValues[382] = d382
			ps425.OverlayValues[383] = d383
			ps425.OverlayValues[384] = d384
			ps425.OverlayValues[385] = d385
			ps425.OverlayValues[386] = d386
			ps425.OverlayValues[387] = d387
			ps425.OverlayValues[388] = d388
			ps425.OverlayValues[389] = d389
			ps425.OverlayValues[390] = d390
			ps425.OverlayValues[391] = d391
			ps425.OverlayValues[392] = d392
			ps425.OverlayValues[393] = d393
			ps425.OverlayValues[394] = d394
			ps425.OverlayValues[395] = d395
			ps425.OverlayValues[396] = d396
			ps425.OverlayValues[397] = d397
			ps425.OverlayValues[398] = d398
			ps425.OverlayValues[399] = d399
			ps425.OverlayValues[400] = d400
			ps425.OverlayValues[401] = d401
			ps425.OverlayValues[402] = d402
			ps425.OverlayValues[403] = d403
			ps425.OverlayValues[404] = d404
			ps425.OverlayValues[405] = d405
			ps425.OverlayValues[406] = d406
			ps425.OverlayValues[407] = d407
			ps425.OverlayValues[408] = d408
			ps425.OverlayValues[409] = d409
			ps425.OverlayValues[410] = d410
			ps425.OverlayValues[411] = d411
			ps425.OverlayValues[412] = d412
			ps425.OverlayValues[413] = d413
			ps425.OverlayValues[414] = d414
			ps425.OverlayValues[415] = d415
			ps425.OverlayValues[416] = d416
			ps425.OverlayValues[417] = d417
			ps425.OverlayValues[418] = d418
			ps425.OverlayValues[419] = d419
			ps425.OverlayValues[420] = d420
			ps425.OverlayValues[421] = d421
			ps425.OverlayValues[422] = d422
			ps425.OverlayValues[423] = d423
				return bbs[21].RenderPS(ps425)
			}
			lbl84 := ctx.W.ReserveLabel()
			lbl85 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d423.Reg, 0)
			ctx.W.EmitJcc(scm.CcNE, lbl84)
			ctx.W.EmitJmp(lbl85)
			ctx.W.MarkLabel(lbl84)
			ctx.W.EmitJmp(lbl21)
			ctx.W.MarkLabel(lbl85)
			ctx.W.EmitJmp(lbl22)
			ps426 := scm.PhiState{General: true}
			ps426.OverlayValues = make([]scm.JITValueDesc, 424)
			ps426.OverlayValues[0] = d0
			ps426.OverlayValues[1] = d1
			ps426.OverlayValues[2] = d2
			ps426.OverlayValues[3] = d3
			ps426.OverlayValues[4] = d4
			ps426.OverlayValues[5] = d5
			ps426.OverlayValues[6] = d6
			ps426.OverlayValues[7] = d7
			ps426.OverlayValues[8] = d8
			ps426.OverlayValues[9] = d9
			ps426.OverlayValues[10] = d10
			ps426.OverlayValues[11] = d11
			ps426.OverlayValues[12] = d12
			ps426.OverlayValues[13] = d13
			ps426.OverlayValues[14] = d14
			ps426.OverlayValues[20] = d20
			ps426.OverlayValues[21] = d21
			ps426.OverlayValues[22] = d22
			ps426.OverlayValues[24] = d24
			ps426.OverlayValues[25] = d25
			ps426.OverlayValues[27] = d27
			ps426.OverlayValues[28] = d28
			ps426.OverlayValues[29] = d29
			ps426.OverlayValues[32] = d32
			ps426.OverlayValues[34] = d34
			ps426.OverlayValues[35] = d35
			ps426.OverlayValues[36] = d36
			ps426.OverlayValues[38] = d38
			ps426.OverlayValues[39] = d39
			ps426.OverlayValues[40] = d40
			ps426.OverlayValues[41] = d41
			ps426.OverlayValues[42] = d42
			ps426.OverlayValues[43] = d43
			ps426.OverlayValues[44] = d44
			ps426.OverlayValues[46] = d46
			ps426.OverlayValues[47] = d47
			ps426.OverlayValues[48] = d48
			ps426.OverlayValues[49] = d49
			ps426.OverlayValues[50] = d50
			ps426.OverlayValues[51] = d51
			ps426.OverlayValues[52] = d52
			ps426.OverlayValues[53] = d53
			ps426.OverlayValues[54] = d54
			ps426.OverlayValues[55] = d55
			ps426.OverlayValues[56] = d56
			ps426.OverlayValues[57] = d57
			ps426.OverlayValues[58] = d58
			ps426.OverlayValues[59] = d59
			ps426.OverlayValues[60] = d60
			ps426.OverlayValues[61] = d61
			ps426.OverlayValues[62] = d62
			ps426.OverlayValues[63] = d63
			ps426.OverlayValues[64] = d64
			ps426.OverlayValues[65] = d65
			ps426.OverlayValues[66] = d66
			ps426.OverlayValues[67] = d67
			ps426.OverlayValues[68] = d68
			ps426.OverlayValues[69] = d69
			ps426.OverlayValues[70] = d70
			ps426.OverlayValues[71] = d71
			ps426.OverlayValues[72] = d72
			ps426.OverlayValues[73] = d73
			ps426.OverlayValues[74] = d74
			ps426.OverlayValues[75] = d75
			ps426.OverlayValues[76] = d76
			ps426.OverlayValues[77] = d77
			ps426.OverlayValues[78] = d78
			ps426.OverlayValues[79] = d79
			ps426.OverlayValues[80] = d80
			ps426.OverlayValues[81] = d81
			ps426.OverlayValues[82] = d82
			ps426.OverlayValues[83] = d83
			ps426.OverlayValues[84] = d84
			ps426.OverlayValues[85] = d85
			ps426.OverlayValues[86] = d86
			ps426.OverlayValues[87] = d87
			ps426.OverlayValues[88] = d88
			ps426.OverlayValues[89] = d89
			ps426.OverlayValues[90] = d90
			ps426.OverlayValues[91] = d91
			ps426.OverlayValues[92] = d92
			ps426.OverlayValues[93] = d93
			ps426.OverlayValues[94] = d94
			ps426.OverlayValues[95] = d95
			ps426.OverlayValues[102] = d102
			ps426.OverlayValues[103] = d103
			ps426.OverlayValues[104] = d104
			ps426.OverlayValues[105] = d105
			ps426.OverlayValues[106] = d106
			ps426.OverlayValues[107] = d107
			ps426.OverlayValues[108] = d108
			ps426.OverlayValues[109] = d109
			ps426.OverlayValues[110] = d110
			ps426.OverlayValues[111] = d111
			ps426.OverlayValues[112] = d112
			ps426.OverlayValues[113] = d113
			ps426.OverlayValues[114] = d114
			ps426.OverlayValues[115] = d115
			ps426.OverlayValues[116] = d116
			ps426.OverlayValues[117] = d117
			ps426.OverlayValues[118] = d118
			ps426.OverlayValues[119] = d119
			ps426.OverlayValues[120] = d120
			ps426.OverlayValues[121] = d121
			ps426.OverlayValues[122] = d122
			ps426.OverlayValues[123] = d123
			ps426.OverlayValues[124] = d124
			ps426.OverlayValues[125] = d125
			ps426.OverlayValues[126] = d126
			ps426.OverlayValues[127] = d127
			ps426.OverlayValues[128] = d128
			ps426.OverlayValues[129] = d129
			ps426.OverlayValues[130] = d130
			ps426.OverlayValues[131] = d131
			ps426.OverlayValues[132] = d132
			ps426.OverlayValues[133] = d133
			ps426.OverlayValues[134] = d134
			ps426.OverlayValues[135] = d135
			ps426.OverlayValues[136] = d136
			ps426.OverlayValues[137] = d137
			ps426.OverlayValues[138] = d138
			ps426.OverlayValues[139] = d139
			ps426.OverlayValues[140] = d140
			ps426.OverlayValues[141] = d141
			ps426.OverlayValues[142] = d142
			ps426.OverlayValues[143] = d143
			ps426.OverlayValues[144] = d144
			ps426.OverlayValues[145] = d145
			ps426.OverlayValues[146] = d146
			ps426.OverlayValues[153] = d153
			ps426.OverlayValues[154] = d154
			ps426.OverlayValues[160] = d160
			ps426.OverlayValues[161] = d161
			ps426.OverlayValues[162] = d162
			ps426.OverlayValues[163] = d163
			ps426.OverlayValues[164] = d164
			ps426.OverlayValues[165] = d165
			ps426.OverlayValues[166] = d166
			ps426.OverlayValues[168] = d168
			ps426.OverlayValues[170] = d170
			ps426.OverlayValues[171] = d171
			ps426.OverlayValues[174] = d174
			ps426.OverlayValues[177] = d177
			ps426.OverlayValues[178] = d178
			ps426.OverlayValues[179] = d179
			ps426.OverlayValues[181] = d181
			ps426.OverlayValues[182] = d182
			ps426.OverlayValues[183] = d183
			ps426.OverlayValues[184] = d184
			ps426.OverlayValues[185] = d185
			ps426.OverlayValues[186] = d186
			ps426.OverlayValues[188] = d188
			ps426.OverlayValues[189] = d189
			ps426.OverlayValues[190] = d190
			ps426.OverlayValues[191] = d191
			ps426.OverlayValues[192] = d192
			ps426.OverlayValues[193] = d193
			ps426.OverlayValues[194] = d194
			ps426.OverlayValues[195] = d195
			ps426.OverlayValues[196] = d196
			ps426.OverlayValues[199] = d199
			ps426.OverlayValues[200] = d200
			ps426.OverlayValues[201] = d201
			ps426.OverlayValues[204] = d204
			ps426.OverlayValues[205] = d205
			ps426.OverlayValues[206] = d206
			ps426.OverlayValues[207] = d207
			ps426.OverlayValues[208] = d208
			ps426.OverlayValues[209] = d209
			ps426.OverlayValues[210] = d210
			ps426.OverlayValues[211] = d211
			ps426.OverlayValues[212] = d212
			ps426.OverlayValues[214] = d214
			ps426.OverlayValues[215] = d215
			ps426.OverlayValues[216] = d216
			ps426.OverlayValues[217] = d217
			ps426.OverlayValues[218] = d218
			ps426.OverlayValues[219] = d219
			ps426.OverlayValues[220] = d220
			ps426.OverlayValues[221] = d221
			ps426.OverlayValues[222] = d222
			ps426.OverlayValues[223] = d223
			ps426.OverlayValues[225] = d225
			ps426.OverlayValues[226] = d226
			ps426.OverlayValues[227] = d227
			ps426.OverlayValues[228] = d228
			ps426.OverlayValues[229] = d229
			ps426.OverlayValues[230] = d230
			ps426.OverlayValues[231] = d231
			ps426.OverlayValues[232] = d232
			ps426.OverlayValues[233] = d233
			ps426.OverlayValues[234] = d234
			ps426.OverlayValues[235] = d235
			ps426.OverlayValues[236] = d236
			ps426.OverlayValues[237] = d237
			ps426.OverlayValues[238] = d238
			ps426.OverlayValues[239] = d239
			ps426.OverlayValues[240] = d240
			ps426.OverlayValues[241] = d241
			ps426.OverlayValues[242] = d242
			ps426.OverlayValues[243] = d243
			ps426.OverlayValues[244] = d244
			ps426.OverlayValues[245] = d245
			ps426.OverlayValues[246] = d246
			ps426.OverlayValues[247] = d247
			ps426.OverlayValues[248] = d248
			ps426.OverlayValues[249] = d249
			ps426.OverlayValues[250] = d250
			ps426.OverlayValues[251] = d251
			ps426.OverlayValues[252] = d252
			ps426.OverlayValues[253] = d253
			ps426.OverlayValues[254] = d254
			ps426.OverlayValues[255] = d255
			ps426.OverlayValues[256] = d256
			ps426.OverlayValues[257] = d257
			ps426.OverlayValues[258] = d258
			ps426.OverlayValues[259] = d259
			ps426.OverlayValues[260] = d260
			ps426.OverlayValues[261] = d261
			ps426.OverlayValues[262] = d262
			ps426.OverlayValues[263] = d263
			ps426.OverlayValues[264] = d264
			ps426.OverlayValues[265] = d265
			ps426.OverlayValues[266] = d266
			ps426.OverlayValues[267] = d267
			ps426.OverlayValues[268] = d268
			ps426.OverlayValues[269] = d269
			ps426.OverlayValues[270] = d270
			ps426.OverlayValues[271] = d271
			ps426.OverlayValues[278] = d278
			ps426.OverlayValues[279] = d279
			ps426.OverlayValues[285] = d285
			ps426.OverlayValues[286] = d286
			ps426.OverlayValues[287] = d287
			ps426.OverlayValues[288] = d288
			ps426.OverlayValues[289] = d289
			ps426.OverlayValues[290] = d290
			ps426.OverlayValues[292] = d292
			ps426.OverlayValues[294] = d294
			ps426.OverlayValues[295] = d295
			ps426.OverlayValues[298] = d298
			ps426.OverlayValues[302] = d302
			ps426.OverlayValues[303] = d303
			ps426.OverlayValues[304] = d304
			ps426.OverlayValues[305] = d305
			ps426.OverlayValues[307] = d307
			ps426.OverlayValues[308] = d308
			ps426.OverlayValues[310] = d310
			ps426.OverlayValues[311] = d311
			ps426.OverlayValues[312] = d312
			ps426.OverlayValues[313] = d313
			ps426.OverlayValues[314] = d314
			ps426.OverlayValues[315] = d315
			ps426.OverlayValues[317] = d317
			ps426.OverlayValues[318] = d318
			ps426.OverlayValues[319] = d319
			ps426.OverlayValues[320] = d320
			ps426.OverlayValues[321] = d321
			ps426.OverlayValues[322] = d322
			ps426.OverlayValues[323] = d323
			ps426.OverlayValues[324] = d324
			ps426.OverlayValues[325] = d325
			ps426.OverlayValues[326] = d326
			ps426.OverlayValues[328] = d328
			ps426.OverlayValues[329] = d329
			ps426.OverlayValues[330] = d330
			ps426.OverlayValues[331] = d331
			ps426.OverlayValues[332] = d332
			ps426.OverlayValues[333] = d333
			ps426.OverlayValues[334] = d334
			ps426.OverlayValues[335] = d335
			ps426.OverlayValues[336] = d336
			ps426.OverlayValues[337] = d337
			ps426.OverlayValues[338] = d338
			ps426.OverlayValues[339] = d339
			ps426.OverlayValues[340] = d340
			ps426.OverlayValues[341] = d341
			ps426.OverlayValues[342] = d342
			ps426.OverlayValues[343] = d343
			ps426.OverlayValues[344] = d344
			ps426.OverlayValues[345] = d345
			ps426.OverlayValues[346] = d346
			ps426.OverlayValues[347] = d347
			ps426.OverlayValues[348] = d348
			ps426.OverlayValues[349] = d349
			ps426.OverlayValues[350] = d350
			ps426.OverlayValues[351] = d351
			ps426.OverlayValues[352] = d352
			ps426.OverlayValues[353] = d353
			ps426.OverlayValues[354] = d354
			ps426.OverlayValues[355] = d355
			ps426.OverlayValues[356] = d356
			ps426.OverlayValues[357] = d357
			ps426.OverlayValues[358] = d358
			ps426.OverlayValues[359] = d359
			ps426.OverlayValues[360] = d360
			ps426.OverlayValues[361] = d361
			ps426.OverlayValues[362] = d362
			ps426.OverlayValues[363] = d363
			ps426.OverlayValues[364] = d364
			ps426.OverlayValues[365] = d365
			ps426.OverlayValues[366] = d366
			ps426.OverlayValues[367] = d367
			ps426.OverlayValues[368] = d368
			ps426.OverlayValues[369] = d369
			ps426.OverlayValues[370] = d370
			ps426.OverlayValues[371] = d371
			ps426.OverlayValues[372] = d372
			ps426.OverlayValues[373] = d373
			ps426.OverlayValues[374] = d374
			ps426.OverlayValues[375] = d375
			ps426.OverlayValues[376] = d376
			ps426.OverlayValues[377] = d377
			ps426.OverlayValues[378] = d378
			ps426.OverlayValues[379] = d379
			ps426.OverlayValues[380] = d380
			ps426.OverlayValues[381] = d381
			ps426.OverlayValues[382] = d382
			ps426.OverlayValues[383] = d383
			ps426.OverlayValues[384] = d384
			ps426.OverlayValues[385] = d385
			ps426.OverlayValues[386] = d386
			ps426.OverlayValues[387] = d387
			ps426.OverlayValues[388] = d388
			ps426.OverlayValues[389] = d389
			ps426.OverlayValues[390] = d390
			ps426.OverlayValues[391] = d391
			ps426.OverlayValues[392] = d392
			ps426.OverlayValues[393] = d393
			ps426.OverlayValues[394] = d394
			ps426.OverlayValues[395] = d395
			ps426.OverlayValues[396] = d396
			ps426.OverlayValues[397] = d397
			ps426.OverlayValues[398] = d398
			ps426.OverlayValues[399] = d399
			ps426.OverlayValues[400] = d400
			ps426.OverlayValues[401] = d401
			ps426.OverlayValues[402] = d402
			ps426.OverlayValues[403] = d403
			ps426.OverlayValues[404] = d404
			ps426.OverlayValues[405] = d405
			ps426.OverlayValues[406] = d406
			ps426.OverlayValues[407] = d407
			ps426.OverlayValues[408] = d408
			ps426.OverlayValues[409] = d409
			ps426.OverlayValues[410] = d410
			ps426.OverlayValues[411] = d411
			ps426.OverlayValues[412] = d412
			ps426.OverlayValues[413] = d413
			ps426.OverlayValues[414] = d414
			ps426.OverlayValues[415] = d415
			ps426.OverlayValues[416] = d416
			ps426.OverlayValues[417] = d417
			ps426.OverlayValues[418] = d418
			ps426.OverlayValues[419] = d419
			ps426.OverlayValues[420] = d420
			ps426.OverlayValues[421] = d421
			ps426.OverlayValues[422] = d422
			ps426.OverlayValues[423] = d423
			ps427 := scm.PhiState{General: true}
			ps427.OverlayValues = make([]scm.JITValueDesc, 424)
			ps427.OverlayValues[0] = d0
			ps427.OverlayValues[1] = d1
			ps427.OverlayValues[2] = d2
			ps427.OverlayValues[3] = d3
			ps427.OverlayValues[4] = d4
			ps427.OverlayValues[5] = d5
			ps427.OverlayValues[6] = d6
			ps427.OverlayValues[7] = d7
			ps427.OverlayValues[8] = d8
			ps427.OverlayValues[9] = d9
			ps427.OverlayValues[10] = d10
			ps427.OverlayValues[11] = d11
			ps427.OverlayValues[12] = d12
			ps427.OverlayValues[13] = d13
			ps427.OverlayValues[14] = d14
			ps427.OverlayValues[20] = d20
			ps427.OverlayValues[21] = d21
			ps427.OverlayValues[22] = d22
			ps427.OverlayValues[24] = d24
			ps427.OverlayValues[25] = d25
			ps427.OverlayValues[27] = d27
			ps427.OverlayValues[28] = d28
			ps427.OverlayValues[29] = d29
			ps427.OverlayValues[32] = d32
			ps427.OverlayValues[34] = d34
			ps427.OverlayValues[35] = d35
			ps427.OverlayValues[36] = d36
			ps427.OverlayValues[38] = d38
			ps427.OverlayValues[39] = d39
			ps427.OverlayValues[40] = d40
			ps427.OverlayValues[41] = d41
			ps427.OverlayValues[42] = d42
			ps427.OverlayValues[43] = d43
			ps427.OverlayValues[44] = d44
			ps427.OverlayValues[46] = d46
			ps427.OverlayValues[47] = d47
			ps427.OverlayValues[48] = d48
			ps427.OverlayValues[49] = d49
			ps427.OverlayValues[50] = d50
			ps427.OverlayValues[51] = d51
			ps427.OverlayValues[52] = d52
			ps427.OverlayValues[53] = d53
			ps427.OverlayValues[54] = d54
			ps427.OverlayValues[55] = d55
			ps427.OverlayValues[56] = d56
			ps427.OverlayValues[57] = d57
			ps427.OverlayValues[58] = d58
			ps427.OverlayValues[59] = d59
			ps427.OverlayValues[60] = d60
			ps427.OverlayValues[61] = d61
			ps427.OverlayValues[62] = d62
			ps427.OverlayValues[63] = d63
			ps427.OverlayValues[64] = d64
			ps427.OverlayValues[65] = d65
			ps427.OverlayValues[66] = d66
			ps427.OverlayValues[67] = d67
			ps427.OverlayValues[68] = d68
			ps427.OverlayValues[69] = d69
			ps427.OverlayValues[70] = d70
			ps427.OverlayValues[71] = d71
			ps427.OverlayValues[72] = d72
			ps427.OverlayValues[73] = d73
			ps427.OverlayValues[74] = d74
			ps427.OverlayValues[75] = d75
			ps427.OverlayValues[76] = d76
			ps427.OverlayValues[77] = d77
			ps427.OverlayValues[78] = d78
			ps427.OverlayValues[79] = d79
			ps427.OverlayValues[80] = d80
			ps427.OverlayValues[81] = d81
			ps427.OverlayValues[82] = d82
			ps427.OverlayValues[83] = d83
			ps427.OverlayValues[84] = d84
			ps427.OverlayValues[85] = d85
			ps427.OverlayValues[86] = d86
			ps427.OverlayValues[87] = d87
			ps427.OverlayValues[88] = d88
			ps427.OverlayValues[89] = d89
			ps427.OverlayValues[90] = d90
			ps427.OverlayValues[91] = d91
			ps427.OverlayValues[92] = d92
			ps427.OverlayValues[93] = d93
			ps427.OverlayValues[94] = d94
			ps427.OverlayValues[95] = d95
			ps427.OverlayValues[102] = d102
			ps427.OverlayValues[103] = d103
			ps427.OverlayValues[104] = d104
			ps427.OverlayValues[105] = d105
			ps427.OverlayValues[106] = d106
			ps427.OverlayValues[107] = d107
			ps427.OverlayValues[108] = d108
			ps427.OverlayValues[109] = d109
			ps427.OverlayValues[110] = d110
			ps427.OverlayValues[111] = d111
			ps427.OverlayValues[112] = d112
			ps427.OverlayValues[113] = d113
			ps427.OverlayValues[114] = d114
			ps427.OverlayValues[115] = d115
			ps427.OverlayValues[116] = d116
			ps427.OverlayValues[117] = d117
			ps427.OverlayValues[118] = d118
			ps427.OverlayValues[119] = d119
			ps427.OverlayValues[120] = d120
			ps427.OverlayValues[121] = d121
			ps427.OverlayValues[122] = d122
			ps427.OverlayValues[123] = d123
			ps427.OverlayValues[124] = d124
			ps427.OverlayValues[125] = d125
			ps427.OverlayValues[126] = d126
			ps427.OverlayValues[127] = d127
			ps427.OverlayValues[128] = d128
			ps427.OverlayValues[129] = d129
			ps427.OverlayValues[130] = d130
			ps427.OverlayValues[131] = d131
			ps427.OverlayValues[132] = d132
			ps427.OverlayValues[133] = d133
			ps427.OverlayValues[134] = d134
			ps427.OverlayValues[135] = d135
			ps427.OverlayValues[136] = d136
			ps427.OverlayValues[137] = d137
			ps427.OverlayValues[138] = d138
			ps427.OverlayValues[139] = d139
			ps427.OverlayValues[140] = d140
			ps427.OverlayValues[141] = d141
			ps427.OverlayValues[142] = d142
			ps427.OverlayValues[143] = d143
			ps427.OverlayValues[144] = d144
			ps427.OverlayValues[145] = d145
			ps427.OverlayValues[146] = d146
			ps427.OverlayValues[153] = d153
			ps427.OverlayValues[154] = d154
			ps427.OverlayValues[160] = d160
			ps427.OverlayValues[161] = d161
			ps427.OverlayValues[162] = d162
			ps427.OverlayValues[163] = d163
			ps427.OverlayValues[164] = d164
			ps427.OverlayValues[165] = d165
			ps427.OverlayValues[166] = d166
			ps427.OverlayValues[168] = d168
			ps427.OverlayValues[170] = d170
			ps427.OverlayValues[171] = d171
			ps427.OverlayValues[174] = d174
			ps427.OverlayValues[177] = d177
			ps427.OverlayValues[178] = d178
			ps427.OverlayValues[179] = d179
			ps427.OverlayValues[181] = d181
			ps427.OverlayValues[182] = d182
			ps427.OverlayValues[183] = d183
			ps427.OverlayValues[184] = d184
			ps427.OverlayValues[185] = d185
			ps427.OverlayValues[186] = d186
			ps427.OverlayValues[188] = d188
			ps427.OverlayValues[189] = d189
			ps427.OverlayValues[190] = d190
			ps427.OverlayValues[191] = d191
			ps427.OverlayValues[192] = d192
			ps427.OverlayValues[193] = d193
			ps427.OverlayValues[194] = d194
			ps427.OverlayValues[195] = d195
			ps427.OverlayValues[196] = d196
			ps427.OverlayValues[199] = d199
			ps427.OverlayValues[200] = d200
			ps427.OverlayValues[201] = d201
			ps427.OverlayValues[204] = d204
			ps427.OverlayValues[205] = d205
			ps427.OverlayValues[206] = d206
			ps427.OverlayValues[207] = d207
			ps427.OverlayValues[208] = d208
			ps427.OverlayValues[209] = d209
			ps427.OverlayValues[210] = d210
			ps427.OverlayValues[211] = d211
			ps427.OverlayValues[212] = d212
			ps427.OverlayValues[214] = d214
			ps427.OverlayValues[215] = d215
			ps427.OverlayValues[216] = d216
			ps427.OverlayValues[217] = d217
			ps427.OverlayValues[218] = d218
			ps427.OverlayValues[219] = d219
			ps427.OverlayValues[220] = d220
			ps427.OverlayValues[221] = d221
			ps427.OverlayValues[222] = d222
			ps427.OverlayValues[223] = d223
			ps427.OverlayValues[225] = d225
			ps427.OverlayValues[226] = d226
			ps427.OverlayValues[227] = d227
			ps427.OverlayValues[228] = d228
			ps427.OverlayValues[229] = d229
			ps427.OverlayValues[230] = d230
			ps427.OverlayValues[231] = d231
			ps427.OverlayValues[232] = d232
			ps427.OverlayValues[233] = d233
			ps427.OverlayValues[234] = d234
			ps427.OverlayValues[235] = d235
			ps427.OverlayValues[236] = d236
			ps427.OverlayValues[237] = d237
			ps427.OverlayValues[238] = d238
			ps427.OverlayValues[239] = d239
			ps427.OverlayValues[240] = d240
			ps427.OverlayValues[241] = d241
			ps427.OverlayValues[242] = d242
			ps427.OverlayValues[243] = d243
			ps427.OverlayValues[244] = d244
			ps427.OverlayValues[245] = d245
			ps427.OverlayValues[246] = d246
			ps427.OverlayValues[247] = d247
			ps427.OverlayValues[248] = d248
			ps427.OverlayValues[249] = d249
			ps427.OverlayValues[250] = d250
			ps427.OverlayValues[251] = d251
			ps427.OverlayValues[252] = d252
			ps427.OverlayValues[253] = d253
			ps427.OverlayValues[254] = d254
			ps427.OverlayValues[255] = d255
			ps427.OverlayValues[256] = d256
			ps427.OverlayValues[257] = d257
			ps427.OverlayValues[258] = d258
			ps427.OverlayValues[259] = d259
			ps427.OverlayValues[260] = d260
			ps427.OverlayValues[261] = d261
			ps427.OverlayValues[262] = d262
			ps427.OverlayValues[263] = d263
			ps427.OverlayValues[264] = d264
			ps427.OverlayValues[265] = d265
			ps427.OverlayValues[266] = d266
			ps427.OverlayValues[267] = d267
			ps427.OverlayValues[268] = d268
			ps427.OverlayValues[269] = d269
			ps427.OverlayValues[270] = d270
			ps427.OverlayValues[271] = d271
			ps427.OverlayValues[278] = d278
			ps427.OverlayValues[279] = d279
			ps427.OverlayValues[285] = d285
			ps427.OverlayValues[286] = d286
			ps427.OverlayValues[287] = d287
			ps427.OverlayValues[288] = d288
			ps427.OverlayValues[289] = d289
			ps427.OverlayValues[290] = d290
			ps427.OverlayValues[292] = d292
			ps427.OverlayValues[294] = d294
			ps427.OverlayValues[295] = d295
			ps427.OverlayValues[298] = d298
			ps427.OverlayValues[302] = d302
			ps427.OverlayValues[303] = d303
			ps427.OverlayValues[304] = d304
			ps427.OverlayValues[305] = d305
			ps427.OverlayValues[307] = d307
			ps427.OverlayValues[308] = d308
			ps427.OverlayValues[310] = d310
			ps427.OverlayValues[311] = d311
			ps427.OverlayValues[312] = d312
			ps427.OverlayValues[313] = d313
			ps427.OverlayValues[314] = d314
			ps427.OverlayValues[315] = d315
			ps427.OverlayValues[317] = d317
			ps427.OverlayValues[318] = d318
			ps427.OverlayValues[319] = d319
			ps427.OverlayValues[320] = d320
			ps427.OverlayValues[321] = d321
			ps427.OverlayValues[322] = d322
			ps427.OverlayValues[323] = d323
			ps427.OverlayValues[324] = d324
			ps427.OverlayValues[325] = d325
			ps427.OverlayValues[326] = d326
			ps427.OverlayValues[328] = d328
			ps427.OverlayValues[329] = d329
			ps427.OverlayValues[330] = d330
			ps427.OverlayValues[331] = d331
			ps427.OverlayValues[332] = d332
			ps427.OverlayValues[333] = d333
			ps427.OverlayValues[334] = d334
			ps427.OverlayValues[335] = d335
			ps427.OverlayValues[336] = d336
			ps427.OverlayValues[337] = d337
			ps427.OverlayValues[338] = d338
			ps427.OverlayValues[339] = d339
			ps427.OverlayValues[340] = d340
			ps427.OverlayValues[341] = d341
			ps427.OverlayValues[342] = d342
			ps427.OverlayValues[343] = d343
			ps427.OverlayValues[344] = d344
			ps427.OverlayValues[345] = d345
			ps427.OverlayValues[346] = d346
			ps427.OverlayValues[347] = d347
			ps427.OverlayValues[348] = d348
			ps427.OverlayValues[349] = d349
			ps427.OverlayValues[350] = d350
			ps427.OverlayValues[351] = d351
			ps427.OverlayValues[352] = d352
			ps427.OverlayValues[353] = d353
			ps427.OverlayValues[354] = d354
			ps427.OverlayValues[355] = d355
			ps427.OverlayValues[356] = d356
			ps427.OverlayValues[357] = d357
			ps427.OverlayValues[358] = d358
			ps427.OverlayValues[359] = d359
			ps427.OverlayValues[360] = d360
			ps427.OverlayValues[361] = d361
			ps427.OverlayValues[362] = d362
			ps427.OverlayValues[363] = d363
			ps427.OverlayValues[364] = d364
			ps427.OverlayValues[365] = d365
			ps427.OverlayValues[366] = d366
			ps427.OverlayValues[367] = d367
			ps427.OverlayValues[368] = d368
			ps427.OverlayValues[369] = d369
			ps427.OverlayValues[370] = d370
			ps427.OverlayValues[371] = d371
			ps427.OverlayValues[372] = d372
			ps427.OverlayValues[373] = d373
			ps427.OverlayValues[374] = d374
			ps427.OverlayValues[375] = d375
			ps427.OverlayValues[376] = d376
			ps427.OverlayValues[377] = d377
			ps427.OverlayValues[378] = d378
			ps427.OverlayValues[379] = d379
			ps427.OverlayValues[380] = d380
			ps427.OverlayValues[381] = d381
			ps427.OverlayValues[382] = d382
			ps427.OverlayValues[383] = d383
			ps427.OverlayValues[384] = d384
			ps427.OverlayValues[385] = d385
			ps427.OverlayValues[386] = d386
			ps427.OverlayValues[387] = d387
			ps427.OverlayValues[388] = d388
			ps427.OverlayValues[389] = d389
			ps427.OverlayValues[390] = d390
			ps427.OverlayValues[391] = d391
			ps427.OverlayValues[392] = d392
			ps427.OverlayValues[393] = d393
			ps427.OverlayValues[394] = d394
			ps427.OverlayValues[395] = d395
			ps427.OverlayValues[396] = d396
			ps427.OverlayValues[397] = d397
			ps427.OverlayValues[398] = d398
			ps427.OverlayValues[399] = d399
			ps427.OverlayValues[400] = d400
			ps427.OverlayValues[401] = d401
			ps427.OverlayValues[402] = d402
			ps427.OverlayValues[403] = d403
			ps427.OverlayValues[404] = d404
			ps427.OverlayValues[405] = d405
			ps427.OverlayValues[406] = d406
			ps427.OverlayValues[407] = d407
			ps427.OverlayValues[408] = d408
			ps427.OverlayValues[409] = d409
			ps427.OverlayValues[410] = d410
			ps427.OverlayValues[411] = d411
			ps427.OverlayValues[412] = d412
			ps427.OverlayValues[413] = d413
			ps427.OverlayValues[414] = d414
			ps427.OverlayValues[415] = d415
			ps427.OverlayValues[416] = d416
			ps427.OverlayValues[417] = d417
			ps427.OverlayValues[418] = d418
			ps427.OverlayValues[419] = d419
			ps427.OverlayValues[420] = d420
			ps427.OverlayValues[421] = d421
			ps427.OverlayValues[422] = d422
			ps427.OverlayValues[423] = d423
			alloc428 := ctx.SnapshotAllocState()
			if !bbs[21].Rendered {
				bbs[21].RenderPS(ps427)
			}
			ctx.RestoreAllocState(alloc428)
			if !bbs[20].Rendered {
				return bbs[20].RenderPS(ps426)
			}
			return result
			ctx.FreeDesc(&d422)
			return result
			}
			ps429 := scm.PhiState{General: true}
			_ = bbs[0].RenderPS(ps429)
			ctx.W.MarkLabel(lbl0)
			d430 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d430)
			ctx.BindReg(r2, &d430)
			ctx.EmitMovPairToResult(&d430, &result)
			ctx.FreeReg(r1)
			ctx.FreeReg(r2)
			ctx.W.ResolveFixups()
			if idxPinned { ctx.UnprotectReg(idxPinnedReg) }
			ctx.W.PatchInt32(r0, int32(120))
			ctx.W.EmitAddRSP32(int32(120))
			return result
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
